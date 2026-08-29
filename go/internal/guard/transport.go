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
	"net/http"
	"net/url"
	"strings"
)

// maxPeek caps how much of a create response the guard reads while looking for
// the new id. The rest of the body is handed to the caller untouched.
const maxPeek = 1 << 20

// maxRedirects caps a redirect chain. A custom CheckRedirect replaces the
// standard library's own limit, so the cap has to live here.
const maxRedirects = 10

type transport struct {
	policy *Policy
	base   http.RoundTripper
}

// NewClient returns the one client the rest of gdoc may use. A nil base means
// real HTTPS.
func NewClient(p *Policy, base http.RoundTripper) *http.Client {
	if base == nil {
		base = http.DefaultTransport
	}
	return &http.Client{
		Transport: &transport{policy: p, base: base},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("guard refused: too many redirects")
			}
			// The body is not carried into the judgment here: a redirect the
			// guard cannot read the body of is judged as if it had none, which
			// is the refusing direction for anything above a read.
			if err := p.Judge(req.Method, req.URL, nil); err != nil {
				return fmt.Errorf("redirect refused: %w", err)
			}
			return nil
		},
	}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := peekBody(req)
	if err != nil {
		return nil, fmt.Errorf("guard refused: unreadable request body: %w", err)
	}
	if err := t.policy.Judge(req.Method, req.URL, body); err != nil {
		return nil, err
	}
	create := isCreate(req.URL, req.Method)
	if create {
		if err := t.checkParent(body); err != nil {
			return nil, err
		}
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if create && resp.StatusCode < 300 {
		t.learnFromCreate(resp)
	}
	return resp, nil
}

func peekBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	if req.GetBody != nil {
		rc, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}

func isCreate(u *url.URL, method string) bool {
	if method != "POST" || u.Host != "www.googleapis.com" {
		return false
	}
	p := strings.TrimPrefix(u.Path, "/upload")
	p = strings.TrimSuffix(p, "/")
	return p == "/drive/v3/files"
}

// checkParent refuses a create that does not name exactly the folder this
// command was given. For multipart uploads the metadata part is the first JSON
// object; M6 sets GetBody so the peek sees it. A create whose parents the guard
// cannot read is refused: not knowing never resolves to carrying it.
func (t *transport) checkParent(body []byte) error {
	var meta struct {
		Parents []string `json:"parents"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&meta); err != nil || len(meta.Parents) != 1 || meta.Parents[0] != t.policy.createIn {
		return fmt.Errorf("guard refused: create must name exactly the folder %q", t.policy.createIn)
	}
	return nil
}

// learnFromCreate is the second of the policy's two doors: an id that came back
// from a create the guard itself carried. The response body is restored so the
// caller reads it whole.
func (t *transport) learnFromCreate(resp *http.Response) {
	orig := resp.Body
	head, err := io.ReadAll(io.LimitReader(orig, maxPeek))
	resp.Body = readCloser{Reader: io.MultiReader(bytes.NewReader(head), orig), Closer: orig}
	if err != nil {
		return
	}
	var created struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(head, &created) == nil && created.ID != "" {
		t.policy.Learn(created.ID)
	}
}

type readCloser struct {
	io.Reader
	io.Closer
}
