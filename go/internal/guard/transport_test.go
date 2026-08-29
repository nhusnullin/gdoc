package guard

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fake struct {
	status int
	body   string
	seen   []*http.Request
}

func (f *fake) RoundTrip(r *http.Request) (*http.Response, error) {
	f.seen = append(f.seen, r)
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    r,
	}, nil
}

func TestRefusedRequestNeverReachesTheWire(t *testing.T) {
	f := &fake{status: 200, body: `{}`}
	p := NewPolicy()
	c := NewClient(p, f)
	_, err := c.Get("https://docs.googleapis.com/v1/documents/EVIL")
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("want a guard refusal, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("the refused request reached the transport")
	}
}

func TestCreateTeachesThePolicy(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"NEWDOC","name":"x"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	resp, err := c.Post("https://www.googleapis.com/drive/v3/files",
		"application/json", bytes.NewReader([]byte(`{"parents":["FOLDER1"]}`)))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body) // body must survive the guard's peek
	if !strings.Contains(string(got), "NEWDOC") {
		t.Fatalf("response body was consumed: %q", got)
	}
	if p.files["NEWDOC"] != LevelFull {
		t.Fatal("the created id was not learned at LevelFull")
	}
}

func TestCreateOutsideTheNamedFolderIsRefused(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"X"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	_, err := c.Post("https://www.googleapis.com/drive/v3/files",
		"application/json", bytes.NewReader([]byte(`{"parents":["OTHER"]}`)))
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("create aimed at a folder that was never named must be refused, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
}

func TestCrossOriginRedirectRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	c := NewClient(p, &fake{status: 200, body: `{}`})
	req, _ := http.NewRequest("GET", "https://docs.googleapis.com/v1/documents/DOC1", nil)
	via := []*http.Request{req}
	next, _ := http.NewRequest("GET", "https://evil.example.com/", nil)
	if c.CheckRedirect(next, via) == nil {
		t.Fatal("a redirect off the allowed hosts must be refused")
	}
}

func TestCreateWithUnreadableParentsIsRefused(t *testing.T) {
	cases := []struct {
		name, body string
	}{
		{"no parents", `{"name":"x"}`},
		{"two parents", `{"parents":["FOLDER1","OTHER"]}`},
		{"not json", `not json at all`},
		{"empty body", ``},
	}
	for _, tc := range cases {
		f := &fake{status: 200, body: `{"id":"X"}`}
		p := NewPolicy()
		p.AllowCreateIn("FOLDER1")
		c := NewClient(p, f)
		_, err := c.Post("https://www.googleapis.com/drive/v3/files",
			"application/json", bytes.NewReader([]byte(tc.body)))
		if err == nil || !strings.Contains(err.Error(), "guard refused") {
			t.Errorf("%s: want a guard refusal, got %v", tc.name, err)
		}
		if len(f.seen) != 0 {
			t.Errorf("%s: it reached the wire", tc.name)
		}
	}
}

func TestUploadCreateAlsoTeachesThePolicy(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"UPLOADED"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	resp, err := c.Post("https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart",
		"application/json", bytes.NewReader([]byte(`{"parents":["FOLDER1"]}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if p.files["UPLOADED"] != LevelFull {
		t.Fatal("the uploaded create was not learned at LevelFull")
	}
}

func TestFailedCreateTeachesNothing(t *testing.T) {
	f := &fake{status: 403, body: `{"id":"NOTMINE"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	resp, err := c.Post("https://www.googleapis.com/drive/v3/files",
		"application/json", bytes.NewReader([]byte(`{"parents":["FOLDER1"]}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if _, known := p.files["NOTMINE"]; known {
		t.Fatal("a refused create must teach the policy nothing")
	}
}

func TestSameOriginRedirectToAKnownPathIsCarried(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	c := NewClient(p, &fake{status: 200, body: `{}`})
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
	via := []*http.Request{req}
	next, _ := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=x", nil)
	if err := c.CheckRedirect(next, via); err != nil {
		t.Fatalf("a redirect the policy allows must be carried: %v", err)
	}
}

func TestNilBaseUsesTheDefaultTransport(t *testing.T) {
	c := NewClient(NewPolicy(), nil)
	tr, ok := c.Transport.(*transport)
	if !ok {
		t.Fatalf("client transport is not the guard: %T", c.Transport)
	}
	if tr.base != http.DefaultTransport {
		t.Fatal("a nil base must fall back to the real HTTPS transport")
	}
}
