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
