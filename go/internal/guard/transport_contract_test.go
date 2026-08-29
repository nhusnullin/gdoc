package guard

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestMethodOverrideIsRefused closes the "judged as one thing, sent as another"
// gap on the header side. Google's REST stack performs the overridden method,
// so a GET the guard allows would arrive as a DELETE it never saw.
func TestMethodOverrideIsRefused(t *testing.T) {
	cases := []struct{ header, value string }{
		{"X-HTTP-Method-Override", "DELETE"},
		{"X-HTTP-Method", "PATCH"},
		{"X-Method-Override", "DELETE"},
	}
	for _, tc := range cases {
		t.Run(tc.header, func(t *testing.T) {
			f := &fake{status: 200, body: `{}`}
			p := NewPolicy()
			p.AllowFile("DOC1", LevelSuggest)
			c := NewClient(p, f)
			req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set(tc.header, tc.value)
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

// bodyOnly is a request body with no GetBody, which is what an upload built
// from a pipe or a plain io.Reader looks like. The guard has to peek it and
// still hand the whole body on.
type bodyOnly struct{ r io.Reader }

func (b bodyOnly) Read(p []byte) (int, error) { return b.r.Read(p) }
func (b bodyOnly) Close() error               { return nil }

func TestABodyThatCannotBeReplayedIsStillSentWhole(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"NEWDOC"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)

	payload := `{"parents":["FOLDER1"],"name":"x"}`
	req, err := http.NewRequest("POST", "https://www.googleapis.com/drive/v3/files",
		bodyOnly{r: strings.NewReader(payload)})
	if err != nil {
		t.Fatal(err)
	}
	if req.GetBody != nil {
		t.Fatal("this test is pointless if the body can be replayed")
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(f.seen) != 1 {
		t.Fatalf("the create must reach the wire once, got %d", len(f.seen))
	}
	sent, err := io.ReadAll(f.seen[0].Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(sent) != payload {
		t.Fatalf("the body the wire saw is not the body the caller wrote: %q", sent)
	}
	if req.Body == nil {
		t.Fatal("RoundTrip must not take the caller's body away from it")
	}
}

// TestACreateWithNoReadableIDIsRecorded: a create the guard cannot learn an id
// from must say so. Silence turns into "file was not given to this command" on
// the next request, which names the wrong problem.
func TestACreateWithNoReadableIDIsRecorded(t *testing.T) {
	f := &fake{status: 200, body: `{"name":"x"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	resp, err := c.Post("https://www.googleapis.com/drive/v3/files",
		"application/json", bytes.NewReader([]byte(`{"parents":["FOLDER1"]}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	w := p.Warnings()
	if len(w) != 1 || !strings.Contains(w[0], "id") {
		t.Fatalf("the miss must be recorded: %v", w)
	}
}

// A response bigger than the peek must still reach the caller whole. The id is
// not learned from it, because the peeked front of the body is not valid JSON
// on its own, and that miss is recorded rather than swallowed. A real Drive
// create answers in a few hundred bytes, so this is the edge, not the path.
func TestABigCreateResponseSurvivesThePeek(t *testing.T) {
	tail := strings.Repeat("x", maxPeek)
	f := &fake{status: 200, body: `{"id":"BIGDOC","note":"` + tail + `"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	resp, err := c.Post("https://www.googleapis.com/drive/v3/files",
		"application/json", bytes.NewReader([]byte(`{"parents":["FOLDER1"]}`)))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(got) != len(f.body) {
		t.Fatalf("the caller got %d bytes of a %d byte body", len(got), len(f.body))
	}
	if _, known := p.level("BIGDOC"); known {
		t.Fatal("an id the guard did not read whole must not enter the set")
	}
	if len(p.Warnings()) != 1 {
		t.Fatalf("the unlearned create must be recorded: %v", p.Warnings())
	}
}

// A multipart create is refused today: its body opens with the MIME boundary,
// not with the metadata object. This pins the refusal so M6 notices it has to
// read the first part rather than discovering the /upload grammar is dead.
func TestAMultipartCreateIsRefusedForNow(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"X"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	body := "--BOUND\r\nContent-Type: application/json\r\n\r\n{\"parents\":[\"FOLDER1\"]}\r\n--BOUND--\r\n"
	_, err := c.Post("https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart",
		"multipart/related; boundary=BOUND", strings.NewReader(body))
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("a multipart create is refused until something reads the first part, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
}

// A create whose metadata sits past the peek is refused rather than carried
// unread.
func TestACreateBiggerThanThePeekIsRefused(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"X"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	pad := strings.Repeat("x", maxPeek)
	_, err := c.Post("https://www.googleapis.com/drive/v3/files", "application/json",
		strings.NewReader(`{"note":"`+pad+`","parents":["FOLDER1"]}`))
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("a create the guard cannot read whole must be refused, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
}

// A chain that keeps redirecting inside the allowed hosts still has to stop.
func TestTheRedirectChainIsCapped(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	c := NewClient(p, &fake{status: 200, body: `{}`})
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
	via := make([]*http.Request, maxRedirects)
	for i := range via {
		via[i] = req
	}
	next, _ := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
	if err := c.CheckRedirect(next, via); err == nil {
		t.Fatal("a redirect chain must be capped even inside the allowed hosts")
	}
}

func TestTheClientHasATimeout(t *testing.T) {
	if c := NewClient(NewPolicy(), nil); c.Timeout == 0 {
		t.Fatal("a hung endpoint must not hang the CLI forever")
	}
}
