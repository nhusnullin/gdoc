// Package gapi is the one room that builds the requests gdoc's reads go out
// as. It names *http.Request and *http.Client and makes neither: the client
// comes from internal/guard as a parameter, so every request here has already
// been judged before it reaches a wire. Keeping request building in one place
// is what stops the bearer, the Accept header and the refresh rule from being
// written three slightly different ways in three reader packages.
//
// A read is a GET, and a write is a POST or a PATCH carrying JSON or a POST
// carrying a multipart/related upload. All of them go through one refresh
// policy, in send below.
package gapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"

	"gdoc/internal/auth"
	"gdoc/internal/guard"
)

// sentError marks an error raised after the server had already accepted the
// request. Everything a caller does about a write depends on that difference: a
// write reported as never sent is a write somebody sends again, and the first
// one is already in the document.
//
// It is a method rather than an exported sentinel so that the writer packages
// can ask the question without importing this one. Naming *http.Client is what
// keeps them out of here, and the interface they name is the whole reason.
type sentError struct{ err error }

func (e sentError) Error() string { return e.err.Error() }
func (e sentError) Unwrap() error { return e.err }

// Sent says this failure happened after the request was accepted.
func (e sentError) Sent() bool { return true }

// sent wraps err as having happened after a 2xx answer.
func sent(err error) error { return sentError{err: err} }

// maxErrorBody caps what is read from a failed request. The body is never
// printed, only Google's error.message is, so this is a bound on the parse.
const maxErrorBody = 1 << 20

// MaxJSONBody is the ceiling on a JSON read. A Docs document with tabs and
// inline suggestions is large, and a body without a bound is a memory limit
// somebody else sets.
const MaxJSONBody = 64 << 20

// Session is one command's authenticated reach: the guard's client, the token
// that goes on every request, and the warnings the run produced.
//
// It is not safe for concurrent use. A command reads in one goroutine, and
// making it safe would mean guarding the token a refresh replaces.
type Session struct {
	policy    *guard.Policy
	client    *http.Client
	token     auth.Token
	warnings  []string
	refreshed bool // the refresh is said once, however many requests follow it
}

// Open loads the token and builds the guard's client from p. A base of nil
// means the real wire. An absent token comes back as the error auth.Load
// already returns, which names `gdoc auth login`.
func Open(p *guard.Policy, base http.RoundTripper) (*Session, error) {
	tok, err := auth.Load()
	if err != nil {
		return nil, err
	}
	return &Session{policy: p, client: guard.NewClient(p, base), token: tok}, nil
}

// Warnings is the policy's warnings followed by the session's own, in that
// order: what the guard could not do quietly first, then what this session did
// to the token.
func (s *Session) Warnings() []string {
	out := append([]string{}, s.policy.Warnings()...)
	return append(out, s.warnings...)
}

// GetJSON reads rawURL and decodes the answer into `into`.
func (s *Session) GetJSON(ctx context.Context, rawURL string, into any) error {
	body, err := s.get(ctx, rawURL, "application/json", MaxJSONBody)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("%s answered 200 with a body that is not JSON: %w", rawURL, err)
	}
	return nil
}

// GetBytes reads rawURL and returns at most limit bytes. An export is bytes
// rather than JSON, and the limit is the caller's because only the caller knows
// what it asked for.
func (s *Session) GetBytes(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	return s.get(ctx, rawURL, "*/*", limit)
}

// PostJSON sends body to rawURL as JSON and decodes the answer into `into`. An
// `into` of nil discards the answer, which is what a write whose only useful
// fact is that it succeeded wants.
//
// The body is marshalled once and each attempt is built over the same bytes, so
// the retry after a 401 sends what the first attempt sent. A body encoded again
// per attempt would be a second chance for the two to differ, and a body read
// once would make the retry an empty batchUpdate: a write that did nothing
// while the envelope said it had been sent.
func (s *Session) PostJSON(ctx context.Context, rawURL string, body any, into any) error {
	return s.writeJSON(ctx, http.MethodPost, rawURL, body, into)
}

// PatchJSON is PostJSON on the other write verb. Drive spells trashing a file
// as files.update, which is a PATCH, and the guard carries it only on a file
// gdoc itself created. The probe is what needs it: a throwaway document it
// cannot put in the trash is a document left in somebody's Drive.
func (s *Session) PatchJSON(ctx context.Context, rawURL string, body any, into any) error {
	return s.writeJSON(ctx, http.MethodPatch, rawURL, body, into)
}

// PostMultipart uploads part to rawURL as a multipart/related create: a JSON
// metadata part first, then the bytes, which is the one body shape Drive takes
// a file and its metadata in. It is what a publish sends, and the guard reads
// the metadata part out of it to check the folder before anything goes out.
//
// The whole body and its boundary are built once, here, and not inside the
// closure send rebuilds the request through. A boundary drawn per attempt would
// put different bytes on the retry than on the attempt the guard judged, which
// is the one thing this package exists to prevent.
func (s *Session) PostMultipart(ctx context.Context, rawURL string, meta any, part []byte, partType string, into any) error {
	body, contentType, err := relatedBody(meta, part, partType)
	if err != nil {
		return fmt.Errorf("the upload for %s could not be built: %w", rawURL, err)
	}
	answer, err := s.send(ctx, rawURL, s.bodyRequest(http.MethodPost, rawURL, body, contentType), MaxJSONBody)
	if err != nil {
		return err
	}
	if into == nil {
		return nil
	}
	if err := json.Unmarshal(answer, into); err != nil {
		// Drive made the file: send only returns a body on a 2xx. So this is a
		// lost answer rather than a create that did not happen, and a caller
		// told otherwise uploads a second document.
		return sent(fmt.Errorf("%s answered 200 with a body that is not JSON: %w", rawURL, err))
	}
	return nil
}

// relatedBody assembles the two parts and returns the bytes and the media type
// that names their boundary.
//
// The metadata is encoded with encoding/json, so nothing a note holds reaches
// the wire as text somebody formatted. The parts are written with
// mime/multipart, whose boundary comes from crypto/rand: a constant would be a
// string the file's own bytes could carry, and the part that then ended early
// is the metadata part the guard judged. Each part carries its Content-Type and
// no other header, which is what the guard allows and what Drive needs.
func relatedBody(meta any, part []byte, partType string) ([]byte, string, error) {
	raw, err := json.Marshal(meta)
	if err != nil {
		return nil, "", fmt.Errorf("the metadata could not be encoded as JSON: %w", err)
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := writePart(w, metadataType, raw); err != nil {
		return nil, "", err
	}
	if err := writePart(w, partType, part); err != nil {
		return nil, "", err
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	// FormatMediaType, not a concatenation: it quotes a boundary that needs it,
	// and it is the shape mime.ParseMediaType reads back, which is what the
	// guard and Drive both parse the header with.
	contentType := mime.FormatMediaType(multipartRelated, map[string]string{"boundary": w.Boundary()})
	if contentType == "" {
		return nil, "", fmt.Errorf("the boundary %q cannot be written into a Content-Type", w.Boundary())
	}
	return buf.Bytes(), contentType, nil
}

// metadataType is the media type Google's own documentation puts on the
// metadata part of a multipart create.
const metadataType = "application/json; charset=UTF-8"

// multipartRelated is the media type of the whole body. Drive picks its parser
// from this, the /upload path and the uploadType parameter together, and the
// guard refuses the request when the three disagree.
const multipartRelated = "multipart/related"

func writePart(w *multipart.Writer, contentType string, body []byte) error {
	h := make(textproto.MIMEHeader, 1)
	h.Set("Content-Type", contentType)
	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = part.Write(body)
	return err
}

// writeJSON is both write verbs in one place, for the reason this package
// exists: a second copy of the marshal, the headers and the refresh rule is a
// second place for them to drift.
func (s *Session) writeJSON(ctx context.Context, method, rawURL string, body any, into any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("the body for %s could not be encoded as JSON: %w", rawURL, err)
	}
	answer, err := s.send(ctx, rawURL, s.bodyRequest(method, rawURL, raw, "application/json"), MaxJSONBody)
	if err != nil {
		return err
	}
	if into == nil {
		return nil
	}
	if err := json.Unmarshal(answer, into); err != nil {
		// The write was accepted: send only returns a body on a 2xx. So this is
		// a failure to read an answer, never a failure to make the change, and
		// it is marked as such.
		return sent(fmt.Errorf("%s answered 200 with a body that is not JSON: %w", rawURL, err))
	}
	return nil
}

// get reads rawURL. It is send over a GET, and the shape is kept so the two
// read helpers above stay one line each.
func (s *Session) get(ctx context.Context, rawURL, accept string, limit int64) ([]byte, error) {
	return s.send(ctx, rawURL, s.getRequest(rawURL, accept), limit)
}

// requestFor builds the request one attempt sends. It is a function rather than
// a request because a request carries its body in a reader that is read once,
// and the retry after a 401 needs a body of its own.
type requestFor func(ctx context.Context) (*http.Request, error)

// getRequest is a read: no body, and the Accept the caller asked for.
func (s *Session) getRequest(rawURL, accept string) requestFor {
	return func(ctx context.Context) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", accept)
		return req, nil
	}
}

// bodyRequest is a write. bytes.Reader is what makes http.NewRequestWithContext
// set ContentLength and GetBody itself, so the guard peeks the same bytes the
// wire carries and the transport can rewind a broken connection over them.
//
// The content type is the caller's, because a multipart body's own type names
// the boundary the parts were written with. Building that header here would be
// a second place the boundary is decided, and the guard reads the body the way
// this header says it is written.
func (s *Session) bodyRequest(method, rawURL string, raw []byte, contentType string) requestFor {
	return func(ctx context.Context) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Accept", "application/json")
		return req, nil
	}
}

// send is the whole refresh policy in one place. An expired token is refreshed
// before the first request goes out, and a 401 is refreshed once and the
// request retried once. A second 401 is not a third attempt: the token is being
// refused for a reason a refresh does not fix, and saying so is more use than
// another round trip.
func (s *Session) send(ctx context.Context, rawURL string, build requestFor, limit int64) ([]byte, error) {
	if s.token.Expired() {
		if err := s.refresh(ctx); err != nil {
			return nil, err
		}
	}
	body, status, err := s.attempt(ctx, build, limit, rawURL)
	if err != nil {
		return nil, err
	}
	if status == http.StatusUnauthorized {
		if err := s.refresh(ctx); err != nil {
			return nil, err
		}
		body, status, err = s.attempt(ctx, build, limit, rawURL)
		if err != nil {
			return nil, err
		}
		if status == http.StatusUnauthorized {
			return nil, errors.New("the token was refused twice; run: gdoc auth login")
		}
	}
	if status < 200 || status > 299 {
		return nil, statusError(rawURL, status, body)
	}
	return body, nil
}

// attempt sends one request and returns its body and status. A transport error,
// which is what a guard refusal is, comes back as the error; a status the
// caller has to decide about does not.
func (s *Session) attempt(ctx context.Context, build requestFor, limit int64, rawURL string) ([]byte, int, error) {
	req, err := build(ctx)
	if err != nil {
		return nil, 0, err
	}
	// Set, not Add: two Authorization values are two credentials, and the guard
	// refuses the request rather than letting the server pick.
	req.Header.Set("Authorization", "Bearer "+s.token.AccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, unwrapRequestError(err)
	}
	defer resp.Body.Close()

	ok := resp.StatusCode >= 200 && resp.StatusCode <= 299
	read := limit
	if !ok {
		read = maxErrorBody
	}
	// One byte past the ceiling, so a body that fills it can be told from a body
	// that ended. Truncating and handing the short bytes on makes the docx
	// reader say "the export is not a docx" and the JSON reader say the answer
	// is not JSON, and both name something the server did not do.
	body, err := io.ReadAll(io.LimitReader(resp.Body, read+1))
	if err != nil {
		return nil, 0, mark(ok, fmt.Errorf("the answer from %s could not be read: %w", rawURL, err))
	}
	if int64(len(body)) > read {
		if !ok {
			// A failed request's body is only ever quoted from, and statusError
			// does not quote it at all. Cutting it here costs nothing.
			return body[:read], resp.StatusCode, nil
		}
		return nil, 0, sent(fmt.Errorf("the answer from %s is larger than the %d bytes this read allows", rawURL, limit))
	}
	return body, resp.StatusCode, nil
}

// mark wraps err as sent when the server had answered 2xx before it happened.
//
// It is the most this package can say, and not the whole of what can go wrong.
// An unmarked failure is one of three: a guard refusal, where nothing left the
// machine; a 4xx, where Docs rejected the batch whole; and a 5xx or a dropped
// connection, where the request was written and gdoc cannot tell whether it was
// applied. The third is not a claim this makes either way, so a caller reading
// a transport failure or a 5xx should look at the document before sending the
// same write again.
func mark(ok bool, err error) error {
	if !ok {
		return err
	}
	return sent(err)
}

// refresh exchanges the refresh token, saves the result and says so once. A
// refresh that fails names the failure and saves nothing: a half-written token
// is worse than an expired one.
//
// It carries the caller's context, because a refresh is one more request inside
// whatever the caller is bounded by. A poll inside `comments --wait` runs on
// that call's deadline, and a refresh that ignored it would be the one request
// in a poll that neither the deadline nor a Ctrl-C could reach.
func (s *Session) refresh(ctx context.Context) error {
	tok, err := s.token.Refresh(ctx, s.client)
	if err != nil {
		return fmt.Errorf("the access token could not be refreshed: %w", err)
	}
	if err := auth.Save(tok); err != nil {
		return fmt.Errorf("the refreshed token could not be saved: %w", err)
	}
	s.token = tok
	if !s.refreshed {
		s.refreshed = true
		s.warnings = append(s.warnings, "the access token was refreshed and saved")
	}
	return nil
}

// statusError carries the status and Google's own message, and never the raw
// body. A refusal goes into the JSON a command prints, and an HTML error page
// on the envelope is noise nobody reads.
func statusError(rawURL string, status int, body []byte) error {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		return fmt.Errorf("%s answered %d: %s", rawURL, status, e.Error.Message)
	}
	return fmt.Errorf("%s answered %d with no readable message", rawURL, status)
}

// unwrapRequestError drops the *url.Error wrapper http.Client.Do puts around a
// transport failure. The wrapper repeats the method and the URL in front of a
// guard refusal that already names the document, and the refusal is the
// sentence the caller has to read.
func unwrapRequestError(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}
