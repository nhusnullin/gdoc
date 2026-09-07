// This file is the only wire in the binary. Every outbound request passes
// Policy.Judge before it reaches the underlying RoundTripper, and a refusal
// returns an error instead of falling through to the network.
package guard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxPeek caps how much of a body the guard reads, on both sides. On the
// request side it is how much metadata a create may put in front of its
// content; a create whose parents sit past the cap is refused rather than
// carried unread. On the response side it is how far the guard looks for the
// new id. Either way the rest of the body is handed on untouched.
const maxPeek = 1 << 20

// maxRedirects caps a redirect chain. A custom CheckRedirect replaces the
// standard library's own limit, so the cap has to live here.
const maxRedirects = 10

// clientTimeout bounds one request end to end. Without it a hung Google
// endpoint hangs the CLI forever, and a command that never returns prints no
// JSON object at all.
const clientTimeout = 5 * time.Minute

// methodOverrideHeaders are the headers Google's REST stack honours to perform
// a different method from the one on the wire. A request carrying one is
// judged as one thing and served as another, which is the whole gap the guard
// exists to close. The allowlist below refuses them too; they are named here so
// the refusal says which trick it stopped.
var methodOverrideHeaders = []string{
	"X-HTTP-Method-Override",
	"X-HTTP-Method",
	"X-Method-Override",
}

// allowedHeaders are the request headers the guard has decided about, keyed in
// lower case. Every other header is refused, and that is the point.
//
// A header is the second spelling of a query parameter. Google's system
// parameters (https://docs.cloud.google.com/apis/docs/system-parameters) give
// `fields` the twin X-Goog-FieldMask, `key` the twin X-Goog-Api-Key,
// `quotaUser` the twin X-Goog-Quota-User, and the method override a header of
// its own. A Drive read whose query carries no `fields` at all can still ask
// for `permissions(...)` in X-Goog-FieldMask, so a guard that judges only the
// query judges half the request.
//
// Blocking the spellings one at a time is what needed a patch each time
// somebody found another one. This is an allowlist for the same reason the
// query rule in params.go is one, and the two are the same rule over one
// request. The list below is what gdoc's own calls set plus what the standard
// library adds above the guard: the credential, the body's type and length, and
// Referer.
//
// What net/http adds, and where, decides which of them have to be here.
// http.Client.Do builds the request for a redirect hop itself: it copies the
// original headers and sets Referer, and that request then passes RoundTrip, so
// leaving Referer out refused every redirect the policy allows. Cookie is the
// other one Do can add, from a jar, and NewClient sets no jar, so a Cookie on a
// request is a second credential nobody here decided about and stays refused.
// Host, Content-Length, Connection, Accept-Encoding and the default User-Agent
// are written by http.Transport, below RoundTrip, so the guard never sees them
// and the entries below are for a caller that sets them itself.
//
// A milestone that needs another one, a resumable upload's
// X-Upload-Content-Type for instance, adds it here on purpose.
var allowedHeaders = map[string]bool{
	"authorization":   true, // the one credential gdoc sends; checkAuthorization reads its value
	"content-type":    true, // every POST and PATCH gdoc makes sets it
	"content-length":  true,
	"accept":          true,
	"accept-encoding": true,
	"user-agent":      true,
	"referer":         true, // http.Client.Do sets it on a redirect hop
}

type transport struct {
	policy *Policy
	base   http.RoundTripper
}

// The base transport's own bounds, the values http.DefaultTransport uses.
// clientTimeout bounds the whole request; these bound the parts of it that can
// hang before a response is ever begun.
const (
	dialTimeout           = 30 * time.Second
	keepAlive             = 30 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	expectContinueTimeout = 1 * time.Second
	idleConnTimeout       = 90 * time.Second
	maxIdleConns          = 100
)

// baseTransport is the guard's own wire. It is not http.DefaultTransport,
// because that one reads HTTPS_PROXY: with a proxy set, a request judged as
// "GET docs.googleapis.com" leaves the machine as a CONNECT to whatever host
// the environment named, and the credential follows it there. The guard judges
// exactly the request that goes on the wire, so Proxy stays nil and the only
// host dialled is the host that was judged.
func baseTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           (&net.Dialer{Timeout: dialTimeout, KeepAlive: keepAlive}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdleConns,
		IdleConnTimeout:       idleConnTimeout,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ExpectContinueTimeout: expectContinueTimeout,
	}
}

// NewClient returns the one client the rest of gdoc may use. A nil base means
// real HTTPS, through the guard's own transport.
func NewClient(p *Policy, base http.RoundTripper) *http.Client {
	if base == nil {
		base = baseTransport()
	}
	return &http.Client{
		Transport: &transport{policy: p, base: base},
		Timeout:   clientTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("guard refused: too many redirects")
			}
			// The redirect target passes RoundTrip too, so this repeats a check
			// rather than replacing one. It is here so the refusal names the
			// redirect instead of the request that followed it.
			if err := checkWireMatchesJudgment(req); err != nil {
				return fmt.Errorf("redirect refused: %w", err)
			}
			// The body is not carried into the judgment here: a redirect the
			// guard cannot read the body of is judged as if it had none. For a
			// batchUpdate that is the refusing direction twice over, since no
			// body is no suggestion and no body is a request list the guard
			// could not read. For a comment write it is not, since no body is no
			// resolve, and the check that matters runs anyway: the redirected
			// request passes RoundTrip with its real body before it goes out.
			if err := p.Judge(req.Method, req.URL, nil); err != nil {
				return fmt.Errorf("redirect refused: %w", err)
			}
			return nil
		},
	}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// The RoundTripper contract: the request body is closed on every path,
	// errors included, and the caller's request is never modified. send is req
	// itself when there is no body, and otherwise a copy carrying the peeked
	// bytes back in front of the rest.
	body, send, err := peekBody(req)
	if err != nil {
		closeBody(req)
		return nil, fmt.Errorf("guard refused: unreadable request body: %w", err)
	}
	if err := checkWireMatchesJudgment(req); err != nil {
		closeBody(send)
		return nil, err
	}
	if err := t.policy.Judge(req.Method, req.URL, body); err != nil {
		closeBody(send)
		return nil, err
	}
	create := isCreate(req.URL, req.Method)
	if create {
		if err := t.checkParent(body); err != nil {
			closeBody(send)
			return nil, err
		}
	}
	resp, err := t.base.RoundTrip(send)
	if err != nil {
		// The contract is (nil, err) on failure. A response handed back beside
		// an error is a body nobody closes.
		return nil, err
	}
	if create && resp.StatusCode < 300 {
		t.learnFromCreate(resp)
	}
	return resp, nil
}

// checkWireMatchesJudgment is the request-level half of one invariant: the
// guard must judge exactly the bytes that go on the wire. Judge is handed a
// method and a URL, and the request can still send a different method through
// a header and a different host through req.Host. The URL-level half of the
// same invariant lives in plainPath, which closes `..`, `%2F` and an opaque
// path.
//
// Neither the method-override header nor the `_method` query parameter nor a
// Host that disagrees with the URL appears in any call gdoc makes, so refusing
// all of them costs nothing.
//
// The header allowlist is the general form of the same thing: X-Goog-FieldMask
// is `fields` by another name, and the next spelling nobody has read about yet
// is refused here without a patch.
func checkWireMatchesJudgment(req *http.Request) error {
	// Header.Get canonicalises the key it looks up, so it only finds an entry
	// stored under the canonical spelling. net/http writes map keys verbatim,
	// so a key planted as "X-HTTP-METHOD-OVERRIDE" would reach Google unseen.
	// Both loops below therefore walk the raw map and fold case themselves.
	for key, vals := range req.Header {
		if !isMethodOverride(key) {
			continue
		}
		for _, v := range vals {
			if v != "" {
				return refuse("%s: %q asks for a method other than the one judged", key, v)
			}
		}
	}
	// One name, one value, counted across every spelling of that name. Two map
	// keys that fold to the same header are two lines on the wire, and a count
	// taken one key at a time reads each of them as the only one. That is how
	// "Authorization" beside "authorization" put two credentials on a judged
	// request and left the server to pick. gdoc sends each of these once, so
	// the rule holds for all of them rather than for the credential alone.
	byName := map[string][]string{}
	for key, vals := range req.Header {
		lower := strings.ToLower(key)
		if !allowedHeaders[lower] {
			return refuse("the header %q is not one gdoc sends, and a header changes what a request returns or does as much as the query does", key)
		}
		byName[lower] = append(byName[lower], vals...)
	}
	for name, vals := range byName {
		if len(vals) != 1 {
			return refuse("the request carries %d values for the header %q, and gdoc sends one, so which one the server reads is not decided here", len(vals), name)
		}
	}
	if vals, ok := byName["authorization"]; ok {
		if err := checkAuthorization(vals); err != nil {
			return err
		}
	}
	if req.URL.Query().Has("_method") {
		return refuse("_method asks for a method other than the one judged")
	}
	// Go dials req.URL.Host but writes req.Host as the Host header, and as
	// :authority on HTTP/2, whenever it is set. Google's frontend routes on
	// that value, so a request judged against one host's grammar would be
	// served by another with the credential attached.
	if req.Host != "" && req.Host != req.URL.Host {
		return refuse("the Host header %q is not the host that was judged, %q", req.Host, req.URL.Host)
	}
	return nil
}

// checkAuthorization is the most the guard can honestly say about the
// credential on a request, and the limit is worth naming rather than papering
// over.
//
// What it checks: one value, the Bearer scheme, and a token after it. That
// refuses a second credential the server would choose between, another scheme
// carrying another principal, and an empty grant. Its caller counts the values
// across every spelling of the name first, so the count here is about the
// values this function is handed rather than the whole request.
//
// What it cannot check: whose token it is. The guard is built from a policy and
// a base transport and never sees the token; whatever attaches the credential
// does it above this point, and refreshes the value as the token expires, so
// pinning a literal value here would refuse the request after every refresh. A
// caller that swaps in another person's bearer token therefore runs the judged
// operation as that person, and the guard carries it. What the guard still
// bounds is which files are reachable and what may be done to them, which is
// principle 3's actual claim. Nothing here is a claim about identity.
//
// The refusal never prints the header value: a refusal goes in the envelope
// the caller reports, and a credential does not belong there.
func checkAuthorization(vals []string) error {
	if len(vals) != 1 {
		return refuse("a request carries one Authorization header, and this one carries %d, so which credential the server reads is not decided here", len(vals))
	}
	const scheme = "Bearer "
	rest, ok := strings.CutPrefix(vals[0], scheme)
	if !ok || strings.TrimSpace(rest) == "" {
		return refuse("the Authorization header is not a Bearer credential with a token after it, and that is the only kind gdoc sends")
	}
	return nil
}

func isMethodOverride(key string) bool {
	for _, h := range methodOverrideHeaders {
		if strings.EqualFold(key, h) {
			return true
		}
	}
	return false
}

func closeBody(req *http.Request) {
	if req != nil && req.Body != nil {
		_ = req.Body.Close()
	}
}

// peekBody returns the front of the request body for the guard to judge, plus
// the request to send. The peeked bytes go back in front of the rest through a
// copy of the request, because RoundTrip may not modify the one it was handed.
//
// It always reads req.Body, and never req.GetBody, and that is the whole point
// of the function. GetBody is a function the caller supplied; nothing makes
// what it returns agree with what Body carries. Judging the replay and sending
// the body would mean the guard decided about bytes that never left the
// machine, which is the one assumption this package exists to not make. Reading
// Body itself makes judged bytes and sent bytes the same bytes by construction,
// so GetBody cannot enter the judgment at all.
//
// The caller's GetBody is dropped, and this is the half a redirect argument
// does not cover. A redirect is driven by http.Client, above the guard, so the
// replayed request passes RoundTrip and is judged again. A retry is not:
// http.Transport rewinds a request through GetBody inside the base transport,
// below the guard, when a pooled connection breaks. Those bytes go on the wire
// without passing RoundTrip at all, so a GetBody returning something other than
// the peeked body is unjudged bytes on the wire.
//
// So the request the guard sends carries a replay the guard wrote itself, over
// the bytes it judged, or no replay at all. The guard writes one when it holds
// the whole body, which is a body shorter than the peek, and only when the
// caller had a replay to begin with: adding one where there was none would make
// a request retryable that its caller built not to be. A longer body is
// streamed on without being held, so there is nothing to replay from and
// GetBody stays nil. http.Transport does not retry a request it cannot rewind,
// which is the refusing direction, and the caller sees the connection error.
func peekBody(req *http.Request) ([]byte, *http.Request, error) {
	if req.Body == nil {
		return nil, req, nil
	}
	head, err := io.ReadAll(io.LimitReader(req.Body, maxPeek))
	if err != nil {
		return nil, req, err
	}
	send := req.Clone(req.Context())
	send.Body = readCloser{Reader: io.MultiReader(bytes.NewReader(head), req.Body), Closer: req.Body}
	send.GetBody = nil
	// A short read means the body ended before the cap, so head is all of it.
	// A read that fills the cap exactly may or may not have a tail, and the
	// guard does not guess: it drops the replay.
	if req.GetBody != nil && len(head) < maxPeek {
		replay := head
		send.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(replay)), nil
		}
	}
	return head, send, nil
}

// isCreate reports whether this request is the create the parent check must
// run on. The path grammar is filesCollection's, so the policy and the parent
// check cannot read the same path two ways.
func isCreate(u *url.URL, method string) bool {
	return method == "POST" && u.Host == "www.googleapis.com" && filesCollection(u.Path)
}

// checkParent refuses a create that does not name exactly the folder this
// command was given. A create whose parents the guard cannot read is refused:
// not knowing never resolves to carrying it.
//
// That includes a multipart upload today. The body of a multipart create opens
// with the MIME boundary, not with the metadata object, so this parse fails and
// the create is refused. The /upload grammar the policy allows is therefore
// unreachable until something here reads the first MIME part, which is M6's
// job: it is the milestone that publishes a docx. Failing closed is the right
// direction to be wrong in, so it stays refused rather than half-parsed.
func (t *transport) checkParent(body []byte) error {
	if err := hasDuplicateKeys(body); err != nil {
		return err
	}
	var meta struct {
		Parents []string `json:"parents"`
	}
	folder := t.policy.createFolder()
	if err := json.Unmarshal(body, &meta); err != nil || len(meta.Parents) != 1 || meta.Parents[0] != folder {
		return fmt.Errorf("guard refused: create must name exactly the folder %q", folder)
	}
	return nil
}

// learnFromCreate is the second of the policy's two doors: an id that came back
// from a create the guard itself carried. The response body is restored so the
// caller reads it whole. A create whose id the guard could not read is
// recorded, because the alternative is silence: the next request against that
// document is refused with "file was not given to this command", which names
// the wrong problem.
func (t *transport) learnFromCreate(resp *http.Response) {
	orig := resp.Body
	head, err := io.ReadAll(io.LimitReader(orig, maxPeek))
	resp.Body = readCloser{Reader: io.MultiReader(bytes.NewReader(head), orig), Closer: orig}
	if err != nil {
		t.policy.note("a create succeeded but its response could not be read, so the new id was not learned: %v", err)
		return
	}
	var created struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(head, &created) != nil || created.ID == "" {
		t.policy.note("a create succeeded but no id was found in the first %d bytes of the response, so the new file is not in the reachable set", maxPeek)
		return
	}
	t.policy.Learn(created.ID)
}

type readCloser struct {
	io.Reader
	io.Closer
}
