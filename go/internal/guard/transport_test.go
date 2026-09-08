package guard

import (
	"bytes"
	"errors"
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

// redirectingFake answers the first request with a 302 to `to` and everything
// after it with 200. It is what lets a test follow a redirect the way the
// standard library does, headers and all, instead of calling CheckRedirect by
// hand.
type redirectingFake struct {
	to   string
	seen []*http.Request
}

func (f *redirectingFake) RoundTrip(r *http.Request) (*http.Response, error) {
	f.seen = append(f.seen, r)
	h := http.Header{"Content-Type": []string{"application/json"}}
	status := http.StatusOK
	if len(f.seen) == 1 {
		status = http.StatusFound
		h.Set("Location", f.to)
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(`{}`)),
		Header:     h,
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

// TestUploadCreateWithADisagreeingBodyIsRefused was the carry until M6. The
// request says uploadType=multipart and sends a plain JSON body, so Drive
// parses it as multipart and the guard parsed it as JSON: two readings of one
// body. It failed closed by luck rather than by design, because Drive refuses
// the request and no 2xx ever came back to learn an id from. Now that the guard
// can parse multipart, the disagreement is refused by name, before the wire.
//
// The carry lives in multipart_test.go, where all three signals agree.
func TestUploadCreateWithADisagreeingBodyIsRefused(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"UPLOADED"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	_, err := c.Post("https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart",
		"application/json", bytes.NewReader([]byte(`{"parents":["FOLDER1"]}`)))
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("want a guard refusal, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
	if _, known := p.files["UPLOADED"]; known {
		t.Fatal("an id was learned from a create the guard read the wrong way")
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

// A nil base still means real HTTPS. It is the guard's own transport rather
// than http.DefaultTransport, and TestTheGuardDoesNotDialThroughAProxy says
// why.
func TestNilBaseUsesARealTransport(t *testing.T) {
	c := NewClient(NewPolicy(), nil)
	tr, ok := c.Transport.(*transport)
	if !ok {
		t.Fatalf("client transport is not the guard: %T", c.Transport)
	}
	if _, ok := tr.base.(*http.Transport); !ok {
		t.Fatalf("a nil base must fall back to a real HTTPS transport, got %T", tr.base)
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

// A create whose metadata sits past the peek is refused rather than carried
// unread. The multipart half of the same rule is in multipart_test.go.
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

// TestAMediaUploadIsRefusedBeforeTheWire is the transport half of the upload
// rule. The refusal has to land before the request goes out, because the id
// the guard would learn afterwards comes back at full level.
func TestAMediaUploadIsRefusedBeforeTheWire(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"UPLOADED"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	_, err := c.Post("https://www.googleapis.com/upload/drive/v3/files?uploadType=media",
		"application/json", bytes.NewReader([]byte(`{"parents":["FOLDER1"]}`)))
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("want a guard refusal, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
	if _, known := p.files["UPLOADED"]; known {
		t.Fatal("an id was learned at full level from a create the guard could not check")
	}
}

// TestTheGuardDoesNotDialThroughAProxy: http.DefaultTransport reads HTTPS_PROXY,
// so a nil base would send an unjudged CONNECT to whatever host the environment
// names, before the judged request and carrying the same credential. The guard
// brings its own base for that reason.
func TestTheGuardDoesNotDialThroughAProxy(t *testing.T) {
	c := NewClient(NewPolicy(), nil)
	tr, ok := c.Transport.(*transport)
	if !ok {
		t.Fatalf("the client's transport is %T, not the guard's", c.Transport)
	}
	if tr.base == http.DefaultTransport {
		t.Fatal("the base is http.DefaultTransport, which honours HTTPS_PROXY")
	}
	base, ok := tr.base.(*http.Transport)
	if !ok {
		t.Fatalf("the guard's base is %T, so its proxy setting cannot be read", tr.base)
	}
	if base.Proxy != nil {
		t.Fatal("the base carries a proxy function, so an unjudged CONNECT can go out first")
	}
}

// GetBody is a function the caller supplied, and nothing makes what it returns
// agree with what Body carries. A guard that judged the replay and sent the
// body would have decided about bytes that never left the machine.
func TestTheGuardJudgesTheBodyItSends(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"SNEAKY"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)

	req, err := http.NewRequest("POST", "https://www.googleapis.com/drive/v3/files",
		strings.NewReader(`{"parents":["SOMEONE_ELSES_FOLDER"]}`))
	if err != nil {
		t.Fatal(err)
	}
	// The replay says the create lands in the folder the command named. The
	// body says otherwise, and the body is what goes on the wire.
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(`{"parents":["FOLDER1"]}`)), nil
	}

	if _, err := c.Do(req); err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("the guard must judge the body it sends, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
	if _, known := p.files["SNEAKY"]; known {
		t.Fatal("an id was learned from a create the guard never judged")
	}
}

// The request that goes out keeps its GetBody, which is what replays the body
// on a redirect or a retry. The guard reads Body to judge it and puts the bytes
// back in front of the rest; it does not spend the replay doing so.
func TestTheSentRequestCanStillReplayItsBody(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"NEWDOC"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)

	payload := `{"parents":["FOLDER1"],"name":"x"}`
	resp, err := c.Post("https://www.googleapis.com/drive/v3/files",
		"application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(f.seen) != 1 {
		t.Fatalf("the create must reach the wire once, got %d", len(f.seen))
	}
	sent := f.seen[0]
	if sent.GetBody == nil {
		t.Fatal("the request that goes out lost its GetBody, so a redirect cannot replay it")
	}
	rc, err := sent.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	replay, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(replay) != payload {
		t.Fatalf("the replay is not the body: %q", replay)
	}
	wire, err := io.ReadAll(sent.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(wire) != payload {
		t.Fatalf("the wire saw %q, not the body the caller wrote", wire)
	}
}

// GetBody is what replays a body BELOW the guard. http.Transport rewinds
// through it when a pooled connection breaks, inside the base transport, and
// those bytes never pass RoundTrip again. So a GetBody the guard did not write
// puts unjudged bytes on the wire on the retry. The request the guard sends
// carries a replay of the bytes it judged, or no replay at all.
func TestTheReplayIsTheBodyTheGuardJudged(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"NEWDOC"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)

	judged := `{"parents":["FOLDER1"]}`
	req, err := http.NewRequest("POST", "https://www.googleapis.com/drive/v3/files",
		strings.NewReader(judged))
	if err != nil {
		t.Fatal(err)
	}
	// The caller's replay says the create lands somewhere else. A retry inside
	// the base transport would send this, and the guard never judged it.
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(`{"parents":["SOMEONE_ELSES_FOLDER"]}`)), nil
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(f.seen) != 1 {
		t.Fatalf("the create must reach the wire once, got %d", len(f.seen))
	}
	sent := f.seen[0]
	if sent.GetBody == nil {
		t.Fatal("a body the guard holds whole must still be replayable")
	}
	rc, err := sent.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	replay, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(replay) != judged {
		t.Fatalf("a retry would send %q, which the guard never judged", replay)
	}
}

// The other half of the same rule. A body longer than the peek is streamed on
// without the guard holding it, so there is nothing to replay from, and the
// caller's own replay is not it. No GetBody means http.Transport does not
// retry, which is the refusing direction.
func TestABodyPastThePeekCarriesNoReplay(t *testing.T) {
	f := &fake{status: 200, body: `{}`}
	p := NewPolicy()
	p.Learn("DOC1") // LevelFull, so an in-place patch is carried
	c := NewClient(p, f)

	// A Drive patch, because it is the one write whose body the policy does not
	// have to read. A batchUpdate past the peek arrives at the policy truncated
	// and is refused there, which is the subject of its own test; this one is
	// about what the transport does with a body it did not hold whole.
	big := `{"trashed":true,"pad":"` + strings.Repeat("x", maxPeek) + `"}`
	req, err := http.NewRequest("PATCH", "https://www.googleapis.com/drive/v3/files/DOC1",
		strings.NewReader(big))
	if err != nil {
		t.Fatal(err)
	}
	if req.GetBody == nil {
		t.Fatal("this test is pointless if the caller had no replay to carry over")
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(f.seen) != 1 {
		t.Fatalf("the write must reach the wire once, got %d", len(f.seen))
	}
	if f.seen[0].GetBody != nil {
		t.Fatal("a body the guard did not hold whole must carry no replay")
	}
	sent, err := io.ReadAll(f.seen[0].Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(sent) != big {
		t.Fatal("the body the wire saw is not the body the caller wrote")
	}
}

// A real redirect, followed by the standard library, rather than CheckRedirect
// called by hand. Client.Do adds a Referer to the request it builds for the
// hop, and that request passes RoundTrip and the header allowlist. Calling the
// hook directly never sees that header, so it passed while a genuinely allowed
// redirect was refused on the wire.
func TestARealSameOriginRedirectIsCarried(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	f := &redirectingFake{to: "https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=text/markdown"}
	c := NewClient(p, f)
	req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer token")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("a redirect the policy allows must be carried: %v", err)
	}
	resp.Body.Close()
	if len(f.seen) != 2 {
		t.Fatalf("the redirect was not followed: %d requests reached the wire", len(f.seen))
	}
	if f.seen[1].Header.Get("Referer") == "" {
		t.Fatal("the premise changed: Client.Do no longer adds a Referer, so this test proves nothing")
	}
}

// countingBody is a request body that records how often it was closed, and can
// fail its read on demand. It exists so the RoundTripper contract RoundTrip's
// comment claims, that the body is closed on every path including a refusal, is
// held by a test rather than by review.
type countingBody struct {
	reader io.Reader
	err    error
	closes int
}

func (b *countingBody) Read(p []byte) (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	return b.reader.Read(p)
}

func (b *countingBody) Close() error {
	b.closes++
	return nil
}

// TestEveryRefusalClosesTheBody walks the four ways RoundTrip returns without
// reaching the wire. A body nobody closes leaks the connection it was read
// from, and net/http hands the transport the only reference to it.
//
// The transport is called directly rather than through the client, because
// http.Client closes the body itself on an error and would hide which of the
// two did it.
func TestEveryRefusalClosesTheBody(t *testing.T) {
	cases := []struct {
		name    string
		policy  func(*Policy)
		method  string
		url     string
		body    *countingBody
		headers map[string]string
	}{
		{
			name:   "unreadable body",
			policy: func(p *Policy) { p.AllowFile("DOC1", LevelSuggest) },
			method: "POST",
			url:    "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate",
			body:   &countingBody{err: errors.New("the body broke mid-read")},
		},
		{
			name:    "wire does not match the judgment",
			policy:  func(p *Policy) { p.AllowFile("DOC1", LevelSuggest) },
			method:  "POST",
			url:     "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate",
			body:    &countingBody{reader: strings.NewReader(`{"writeControl":{"writeMode":"SUGGEST"}}`)},
			headers: map[string]string{"X-Goog-FieldMask": "permissions"},
		},
		{
			name:   "the policy refuses",
			policy: func(p *Policy) {},
			method: "POST",
			url:    "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate",
			body:   &countingBody{reader: strings.NewReader(`{"writeControl":{"writeMode":"SUGGEST"}}`)},
		},
		{
			name:   "the parent check refuses",
			policy: func(p *Policy) { p.AllowCreateIn("FOLDER1") },
			method: "POST",
			url:    "https://www.googleapis.com/drive/v3/files",
			body:   &countingBody{reader: strings.NewReader(`{"parents":["ELSEWHERE"]}`)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := NewPolicy()
			c.policy(p)
			f := &fake{status: 200, body: `{}`}
			req, err := http.NewRequest(c.method, c.url, c.body)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer t")
			req.Header.Set("Content-Type", "application/json")
			for k, v := range c.headers {
				req.Header.Set(k, v)
			}
			resp, err := (&transport{policy: p, base: f}).RoundTrip(req)
			if err == nil {
				t.Fatal("want a refusal")
			}
			if resp != nil {
				t.Fatal("a refusal returns no response, so nothing is left to close")
			}
			if len(f.seen) != 0 {
				t.Fatal("the refused request reached the wire")
			}
			if c.body.closes != 1 {
				t.Fatalf("the body was closed %d times, want exactly once", c.body.closes)
			}
		})
	}
}
