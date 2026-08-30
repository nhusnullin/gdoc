package guard

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// This file holds one family of tests: the guard must judge exactly the bytes
// that go on the wire. Four spellings send something other than what Judge
// read, a method header, a method query parameter, a Host that disagrees with
// the URL and an opaque path, and all four are refused.

// TestMethodOverrideIsRefused closes the "judged as one thing, sent as another"
// gap on the header side. Google's REST stack performs the overridden method,
// so a GET the guard allows would arrive as a DELETE it never saw.
//
// The rawMapKey half is the same trick spelled a way http.Header.Get cannot
// see. Get canonicalises the key it looks up, so it only finds an entry stored
// as "X-Http-Method-Override". A key written straight into the map is invisible
// to it, and net/http writes map keys verbatim, so the header reaches Google
// intact. That is a real bypass this test caught, which is why both spellings
// stay in the table.
func TestMethodOverrideIsRefused(t *testing.T) {
	cases := []struct {
		header, value string
		rawMapKey     bool
	}{
		{header: "X-HTTP-Method-Override", value: "DELETE"},
		{header: "X-HTTP-Method", value: "PATCH"},
		{header: "X-Method-Override", value: "DELETE"},
		{header: "X-HTTP-METHOD-OVERRIDE", value: "DELETE", rawMapKey: true},
		{header: "x-http-method", value: "DELETE", rawMapKey: true},
		{header: "X-METHOD-override", value: "DELETE", rawMapKey: true},
	}
	for _, tc := range cases {
		name := tc.header
		if tc.rawMapKey {
			name = "raw map key " + tc.header
		}
		t.Run(name, func(t *testing.T) {
			f := &fake{status: 200, body: `{}`}
			p := NewPolicy()
			p.AllowFile("DOC1", LevelSuggest)
			c := NewClient(p, f)
			req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.rawMapKey {
				req.Header[tc.header] = []string{tc.value} // not Header.Set: the spelling is the point
			} else {
				req.Header.Set(tc.header, tc.value)
			}
			if _, err := c.Do(req); err == nil || !strings.Contains(err.Error(), "guard refused") {
				t.Fatalf("want a guard refusal, got %v", err)
			}
			if len(f.seen) != 0 {
				t.Fatal("it reached the wire")
			}
		})
	}
}

// TestMethodQueryOverrideIsRefused is the query-parameter spelling of the same
// trick.
func TestMethodQueryOverrideIsRefused(t *testing.T) {
	f := &fake{status: 200, body: `{}`}
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	c := NewClient(p, f)
	_, err := c.Get("https://www.googleapis.com/drive/v3/files/DOC1?_method=DELETE")
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("want a guard refusal, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
}

// TestAHostHeaderOtherThanTheURLHostIsRefused closes the same gap on the host
// side. Go dials req.URL.Host but writes req.Host as the Host header, and as
// :authority on HTTP/2, whenever it is set. Google's frontend routes on that
// value, so a request judged against one host's grammar would be served by
// another with the credential attached.
func TestAHostHeaderOtherThanTheURLHostIsRefused(t *testing.T) {
	for _, host := range []string{"docs.googleapis.com", "gmail.googleapis.com", "evil.example"} {
		t.Run(host, func(t *testing.T) {
			f := &fake{status: 200, body: `{}`}
			p := NewPolicy()
			p.AllowFile("DOC1", LevelSuggest)
			c := NewClient(p, f)
			req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Host = host
			if _, err := c.Do(req); err == nil || !strings.Contains(err.Error(), "guard refused") {
				t.Fatalf("want a guard refusal, got %v", err)
			}
			if len(f.seen) != 0 {
				t.Fatal("it reached the wire")
			}
		})
	}
}

// A Host header that agrees with the URL is what Go itself would send, so it
// must still be carried. The check refuses a disagreement, not the field.
func TestAHostHeaderEqualToTheURLHostIsCarried(t *testing.T) {
	f := &fake{status: 200, body: `{}`}
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	c := NewClient(p, f)
	req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "www.googleapis.com"
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(f.seen) != 1 {
		t.Fatal("the request did not reach the wire")
	}
}

// TestAnOpaqueURLIsRefused is the third spelling. url.URL.RequestURI returns
// Opaque when it is set, so the transport writes Opaque and ignores Path. The
// guard reads Path, so an id in the set would carry a request to an id outside
// it, on a sub-resource the policy refuses by name.
func TestAnOpaqueURLIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	u := &url.URL{
		Scheme: "https",
		Host:   "www.googleapis.com",
		Path:   "/drive/v3/files/DOC1",
		Opaque: "/drive/v3/files/EVIL/permissions",
	}
	if u.RequestURI() != "/drive/v3/files/EVIL/permissions" {
		t.Fatalf("the premise changed: RequestURI is %q", u.RequestURI())
	}
	if err := p.Judge("GET", u, nil); err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("want a guard refusal, got %v", err)
	}
}

// The same URL without the opaque form is a plain read the guard carries. The
// refusal above is about Opaque, not about the path.
func TestTheSamePathWithoutOpaqueIsCarried(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	u := &url.URL{Scheme: "https", Host: "www.googleapis.com", Path: "/drive/v3/files/DOC1"}
	if err := p.Judge("GET", u, nil); err != nil {
		t.Fatalf("a plain read must still pass: %v", err)
	}
}

// TestAnUnknownHeaderIsRefused is the header half of the allowlist, and
// X-Goog-FieldMask is why it exists. Google's system parameters give `fields` a
// header twin, so a Drive read whose query names no fields at all can still ask
// for `permissions(...)` in a header, and a guard that reads only the query
// carries it.
//
// The lower-case map keys are the same trick spelled the way Header.Get cannot
// see. net/http writes map keys verbatim, so a key planted under a
// non-canonical spelling reaches Google intact, and the check must fold case
// over the raw map rather than look up one canonical name.
func TestAnUnknownHeaderIsRefused(t *testing.T) {
	cases := []struct {
		header, value string
		rawMapKey     bool
	}{
		{header: "X-Goog-FieldMask", value: "permissions(emailAddress)"},
		{header: "x-goog-fieldmask", value: "permissions(emailAddress)", rawMapKey: true},
		{header: "X-GOOG-FIELDMASK", value: "*", rawMapKey: true},
		{header: "X-Goog-Api-Key", value: "somekey"},
		{header: "X-Goog-User-Project", value: "someproject"},
		{header: "X-Goog-Quota-User", value: "someone"},
		{header: "X-Server-Timeout", value: "1"},
		{header: "X-Goog-Request-Params", value: "x"},
		{header: "X-Invented-Tomorrow", value: "x"},
	}
	for _, tc := range cases {
		name := tc.header
		if tc.rawMapKey {
			name = "raw map key " + tc.header
		}
		t.Run(name, func(t *testing.T) {
			f := &fake{status: 200, body: `{}`}
			p := NewPolicy()
			p.AllowFile("DOC1", LevelSuggest)
			c := NewClient(p, f)
			req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.rawMapKey {
				req.Header[tc.header] = []string{tc.value} // not Header.Set: the spelling is the point
			} else {
				req.Header.Set(tc.header, tc.value)
			}
			if _, err := c.Do(req); err == nil || !strings.Contains(err.Error(), "guard refused") {
				t.Fatalf("want a guard refusal, got %v", err)
			}
			if len(f.seen) != 0 {
				t.Fatal("it reached the wire")
			}
		})
	}
}

// The headers gdoc's own calls set must still be carried. The allowlist refuses
// what nobody decided about, not everything.
func TestTheHeadersGdocSendsAreCarried(t *testing.T) {
	f := &fake{status: 200, body: `{}`}
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	c := NewClient(p, f)
	req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("a request carrying only gdoc's own headers must go out: %v", err)
	}
	resp.Body.Close()
	if len(f.seen) != 1 {
		t.Fatal("the request did not reach the wire")
	}
}

// The Authorization header was on the allowlist with no opinion about its
// value, so any credential in any scheme rode along on a judged request. The
// guard does not hold the token and cannot say whose it is, so what it checks
// is the shape: one value, the Bearer scheme, something after it.
// checkAuthorization names the part it cannot check.
func TestTheAuthorizationHeaderMustBeABearerCredential(t *testing.T) {
	cases := []struct {
		name   string
		values []string
	}{
		{"another scheme entirely", []string{"Basic dXNlcjpwYXNz"}},
		{"a scheme the guard has never decided about", []string{"GoogleLogin auth=x"}},
		{"the scheme with nothing after it", []string{"Bearer "}},
		{"no scheme at all", []string{"token"}},
		{"two credentials, and the server picks", []string{"Bearer one", "Bearer two"}},
		{"an empty header", []string{""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fake{status: 200, body: `{}`}
			p := NewPolicy()
			p.AllowFile("DOC1", LevelSuggest)
			c := NewClient(p, f)
			req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header["Authorization"] = tc.values
			_, err = c.Do(req)
			if err == nil || !strings.Contains(err.Error(), "guard refused") {
				t.Fatalf("want a guard refusal, got %v", err)
			}
			if strings.Contains(err.Error(), "dXNlcjpwYXNz") {
				t.Fatal("the refusal printed the credential it refused")
			}
			if len(f.seen) != 0 {
				t.Fatal("it reached the wire")
			}
		})
	}
}

// One header spelled two ways is two headers on the wire. net/http writes map
// keys verbatim, so "Authorization" and "authorization" are two entries here
// and two Authorization lines to Google, each of them passing a count that was
// taken one key at a time. The credential case is the one that matters, and
// the rule is the same for every allowlisted header: gdoc sends each once, and
// which copy the server reads is not decided here.
func TestTwoSpellingsOfOneHeaderAreCountedTogether(t *testing.T) {
	cases := []struct {
		name   string
		header map[string][]string
	}{
		{"two Authorization headers in two casings", map[string][]string{
			"Authorization": {"Bearer one"},
			"authorization": {"Bearer two"},
		}},
		{"three casings of one credential", map[string][]string{
			"Authorization": {"Bearer one"},
			"authorization": {"Bearer two"},
			"AUTHORIZATION": {"Bearer three"},
		}},
		{"two Content-Type headers in two casings", map[string][]string{
			"Authorization": {"Bearer one"},
			"Content-Type":  {"application/json"},
			"content-type":  {"multipart/related; boundary=x"},
		}},
		{"one key carrying two values", map[string][]string{
			"Accept": {"application/json", "text/plain"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fake{status: 200, body: `{}`}
			p := NewPolicy()
			p.AllowFile("DOC1", LevelSuggest)
			c := NewClient(p, f)
			req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range tc.header {
				req.Header[k] = v // not Header.Set: the spelling is the point
			}
			_, err = c.Do(req)
			if err == nil || !strings.Contains(err.Error(), "guard refused") {
				t.Fatalf("want a guard refusal, got %v", err)
			}
			if strings.Contains(err.Error(), "Bearer") {
				t.Fatal("the refusal printed the credential it refused")
			}
			if len(f.seen) != 0 {
				t.Fatal("it reached the wire")
			}
		})
	}
}

// A method override spelled twice in two casings is still refused. The
// override loop already walks the raw map, so this pins the behaviour rather
// than reporting a gap.
func TestAMethodOverrideInTwoCasingsIsRefused(t *testing.T) {
	f := &fake{status: 200, body: `{}`}
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	c := NewClient(p, f)
	req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header["X-HTTP-Method-Override"] = []string{""}
	req.Header["x-http-method-override"] = []string{"DELETE"}
	if _, err := c.Do(req); err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("want a guard refusal, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
}
