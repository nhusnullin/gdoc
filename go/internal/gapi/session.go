// Package gapi is the one room that builds the requests gdoc's reads go out
// as. It names *http.Request and *http.Client and makes neither: the client
// comes from internal/guard as a parameter, so every request here has already
// been judged before it reaches a wire. Keeping request building in one place
// is what stops the bearer, the Accept header and the refresh rule from being
// written three slightly different ways in three reader packages.
//
// Everything here is a GET. Nothing in this package writes to Docs or Drive.
package gapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"gdoc/internal/auth"
	"gdoc/internal/guard"
)

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

// get is the whole refresh policy in one place. An expired token is refreshed
// before the first request goes out, and a 401 is refreshed once and the
// request retried once. A second 401 is not a third attempt: the token is being
// refused for a reason a refresh does not fix, and saying so is more use than
// another round trip.
func (s *Session) get(ctx context.Context, rawURL, accept string, limit int64) ([]byte, error) {
	if s.token.Expired() {
		if err := s.refresh(); err != nil {
			return nil, err
		}
	}
	body, status, err := s.attempt(ctx, rawURL, accept, limit)
	if err != nil {
		return nil, err
	}
	if status == http.StatusUnauthorized {
		if err := s.refresh(); err != nil {
			return nil, err
		}
		body, status, err = s.attempt(ctx, rawURL, accept, limit)
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
func (s *Session) attempt(ctx context.Context, rawURL, accept string, limit int64) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	// Set, not Add: two Authorization values are two credentials, and the guard
	// refuses the request rather than letting the server pick.
	req.Header.Set("Authorization", "Bearer "+s.token.AccessToken)
	req.Header.Set("Accept", accept)

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
		return nil, 0, fmt.Errorf("the answer from %s could not be read: %w", rawURL, err)
	}
	if int64(len(body)) > read {
		if !ok {
			// A failed request's body is only ever quoted from, and statusError
			// does not quote it at all. Cutting it here costs nothing.
			return body[:read], resp.StatusCode, nil
		}
		return nil, 0, fmt.Errorf("the answer from %s is larger than the %d bytes this read allows", rawURL, limit)
	}
	return body, resp.StatusCode, nil
}

// refresh exchanges the refresh token, saves the result and says so once. A
// refresh that fails names the failure and saves nothing: a half-written token
// is worse than an expired one.
func (s *Session) refresh() error {
	tok, err := s.token.Refresh(s.client)
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
