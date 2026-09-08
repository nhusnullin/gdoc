package gapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gdoc/internal/auth"
	"gdoc/internal/guard"
)

const docID = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"

func docURL() string { return "https://docs.googleapis.com/v1/documents/" + docID }

// tokenFile writes a token fixture into a temp config dir and returns the path.
// expiry decides whether the session has to refresh before its first request.
func tokenFile(t *testing.T, expiry time.Time) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	tok := auth.Token{
		AccessToken:  "OLD",
		RefreshToken: "R1",
		TokenURI:     auth.TokenURI,
		ClientID:     "CID",
		ClientSecret: "CS",
		Scopes:       []string{"https://www.googleapis.com/auth/drive"},
		Expiry:       expiry,
	}
	b, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "oauth-token.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func savedToken(t *testing.T, path string) auth.Token {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var tok auth.Token
	if err := json.Unmarshal(b, &tok); err != nil {
		t.Fatal(err)
	}
	return tok
}

// seen is one request the fake wire carried, kept as facts rather than as the
// live *http.Request, whose body is closed by the time a test reads it.
type seen struct {
	Method      string
	Host        string
	Path        string
	Auth        []string
	Accept      string
	ContentType string
	Body        string
	Form        url.Values
}

// wire is the fake RoundTripper. It answers the token host with a refreshed
// access token, and hands every other request to answer, which the test
// supplies. Nothing here reaches the network.
type wire struct {
	mu      sync.Mutex
	seen    []seen
	answer  func(n int, r *http.Request) (int, string) // status, body; n counts non-token requests
	refresh string                                     // the access token a refresh hands back
}

func (w *wire) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := seen{Method: r.Method, Host: r.URL.Host, Path: r.URL.Path,
		Auth: r.Header.Values("Authorization"), Accept: r.Header.Get("Accept"),
		ContentType: r.Header.Get("Content-Type")}
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		rec.Body = string(b)
		rec.Form, _ = url.ParseQuery(rec.Body)
	}
	w.mu.Lock()
	w.seen = append(w.seen, rec)
	n := 0
	for _, s := range w.seen[:len(w.seen)-1] {
		if s.Host != "oauth2.googleapis.com" {
			n++
		}
	}
	w.mu.Unlock()

	if r.URL.Host == "oauth2.googleapis.com" {
		tok := w.refresh
		if tok == "" {
			tok = "NEW"
		}
		return reply(r, 200, `{"access_token":"`+tok+`","expires_in":3600}`), nil
	}
	status, body := 200, `{"documentId":"`+docID+`"}`
	if w.answer != nil {
		status, body = w.answer(n, r)
	}
	return reply(r, status, body), nil
}

func reply(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Request: r, Header: http.Header{},
		Body: io.NopCloser(strings.NewReader(body))}
}

func (w *wire) requests() []seen {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]seen, len(w.seen))
	copy(out, w.seen)
	return out
}

// open builds a session over the fake wire with the one document allowed.
func open(t *testing.T, w *wire) *Session {
	t.Helper()
	p := guard.NewPolicy()
	p.AllowFile(docID, guard.LevelSuggest)
	s, err := Open(p, w)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestGetJSONSendsExactlyOneBearerAndDecodes(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{}
	s := open(t, w)

	var into struct {
		DocumentID string `json:"documentId"`
	}
	if err := s.GetJSON(context.Background(), docURL(), &into); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if into.DocumentID != docID {
		t.Errorf("decoded documentId = %q, want %q", into.DocumentID, docID)
	}
	reqs := w.requests()
	if len(reqs) != 1 {
		t.Fatalf("sent %d requests, want 1: %+v", len(reqs), reqs)
	}
	if got := reqs[0].Auth; len(got) != 1 || got[0] != "Bearer OLD" {
		t.Errorf("Authorization = %v, want exactly one %q", got, "Bearer OLD")
	}
	if reqs[0].Accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", reqs[0].Accept)
	}
	if len(s.Warnings()) != 0 {
		t.Errorf("a live token warned about nothing to warn about: %v", s.Warnings())
	}
}

func TestAnExpiredTokenIsRefreshedAndSavedBeforeTheFirstRequest(t *testing.T) {
	path := tokenFile(t, time.Now().Add(-time.Hour))
	w := &wire{}
	s := open(t, w)

	if err := s.GetJSON(context.Background(), docURL(), &struct{}{}); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	reqs := w.requests()
	if len(reqs) != 2 {
		t.Fatalf("sent %d requests, want the refresh then the read: %+v", len(reqs), reqs)
	}
	if reqs[0].Host != "oauth2.googleapis.com" || reqs[0].Method != "POST" {
		t.Errorf("first request was %s %s%s, want the token POST", reqs[0].Method, reqs[0].Host, reqs[0].Path)
	}
	if got := reqs[0].Form.Get("grant_type"); got != "refresh_token" {
		t.Errorf("grant_type = %q, want refresh_token", got)
	}
	if got := reqs[1].Auth; len(got) != 1 || got[0] != "Bearer NEW" {
		t.Errorf("the read carried %v, want the refreshed token", got)
	}
	if got := savedToken(t, path).AccessToken; got != "NEW" {
		t.Errorf("saved access token = %q, want NEW", got)
	}
	if got := savedToken(t, path).RefreshToken; got != "R1" {
		t.Errorf("saved refresh token = %q, want the old one kept", got)
	}
	want := "the access token was refreshed and saved"
	if !hasWarning(s.Warnings(), want) {
		t.Errorf("Warnings() = %v, want one saying %q", s.Warnings(), want)
	}
}

func TestOneRefreshIsEnoughForTheWholeSession(t *testing.T) {
	tokenFile(t, time.Now().Add(-time.Hour))
	w := &wire{}
	s := open(t, w)

	for i := 0; i < 3; i++ {
		if err := s.GetJSON(context.Background(), docURL(), &struct{}{}); err != nil {
			t.Fatalf("GetJSON %d: %v", i, err)
		}
	}
	var refreshes int
	for _, r := range w.requests() {
		if r.Host == "oauth2.googleapis.com" {
			refreshes++
		}
	}
	if refreshes != 1 {
		t.Errorf("sent %d refreshes, want 1", refreshes)
	}
	if len(s.Warnings()) != 1 {
		t.Errorf("Warnings() = %v, want the refresh said once", s.Warnings())
	}
}

func TestA401IsRefreshedOnceAndTheRequestRetriedOnce(t *testing.T) {
	path := tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		if n == 0 {
			return 401, `{"error":{"message":"Invalid Credentials"}}`
		}
		return 200, `{"documentId":"` + docID + `"}`
	}}
	s := open(t, w)

	if err := s.GetJSON(context.Background(), docURL(), &struct{}{}); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	reqs := w.requests()
	if len(reqs) != 3 {
		t.Fatalf("sent %d requests, want the read, the refresh and the retry: %+v", len(reqs), reqs)
	}
	if reqs[1].Host != "oauth2.googleapis.com" {
		t.Errorf("second request was to %s, want the token host", reqs[1].Host)
	}
	if got := reqs[2].Auth; len(got) != 1 || got[0] != "Bearer NEW" {
		t.Errorf("the retry carried %v, want the refreshed token", got)
	}
	if got := savedToken(t, path).AccessToken; got != "NEW" {
		t.Errorf("saved access token = %q, want NEW", got)
	}
}

func TestTwoUnauthorizedAnswersStopTheSession(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		return 401, `{"error":{"message":"Invalid Credentials"}}`
	}}
	s := open(t, w)

	err := s.GetJSON(context.Background(), docURL(), &struct{}{})
	if err == nil {
		t.Fatal("two 401s came back as success")
	}
	if !strings.Contains(err.Error(), "gdoc auth login") {
		t.Errorf("error = %q, want it to name gdoc auth login", err)
	}
	if n := len(w.requests()); n != 3 {
		t.Errorf("sent %d requests, want the read, the refresh and one retry and then a stop", n)
	}
}

func TestANonSuccessCarriesTheStatusAndGooglesMessage(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		return 404, `{"error":{"code":404,"message":"Requested entity was not found.","status":"NOT_FOUND"}}`
	}}
	s := open(t, w)

	err := s.GetJSON(context.Background(), docURL(), &struct{}{})
	if err == nil {
		t.Fatal("a 404 came back as success")
	}
	for _, want := range []string{"404", "Requested entity was not found."} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to carry %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "NOT_FOUND") {
		t.Errorf("error = %q, want Google's message and not the raw body", err)
	}
}

func TestANonSuccessWithNoReadableMessageStillCarriesTheStatus(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		return 500, "<html>backend error</html>"
	}}
	s := open(t, w)

	err := s.GetJSON(context.Background(), docURL(), &struct{}{})
	if err == nil {
		t.Fatal("a 500 came back as success")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want it to carry the status", err)
	}
	if strings.Contains(err.Error(), "backend error") {
		t.Errorf("error = %q, want the status and not the raw body", err)
	}
}

func TestGetBytesReadsUpToTheLimit(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	body := strings.Repeat("x", 10)
	w := &wire{answer: func(n int, r *http.Request) (int, string) { return 200, body }}
	s := open(t, w)

	b, err := s.GetBytes(context.Background(), docURL(), 10)
	if err != nil {
		t.Fatalf("GetBytes: %v", err)
	}
	if len(b) != 10 {
		t.Errorf("read %d bytes, want all 10", len(b))
	}
	if r := w.requests()[0]; r.Accept != "*/*" {
		t.Errorf("Accept = %q, want */* on a bytes read", r.Accept)
	}
}

// TestABodyOverTheLimitIsAnErrorNamingTheLimit is about which problem the
// caller is told about. Truncating and handing the short bytes on makes the
// docx reader say "the export is not a docx" and the JSON reader say the answer
// is not JSON, and both name something the server did not do. --witness then
// marks every thread unmatched with no hint that a ceiling caused it.
func TestABodyOverTheLimitIsAnErrorNamingTheLimit(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	body := strings.Repeat("x", 100)
	w := &wire{answer: func(n int, r *http.Request) (int, string) { return 200, body }}
	s := open(t, w)

	if _, err := s.GetBytes(context.Background(), docURL(), 10); err == nil {
		t.Fatal("a body over the limit came back as a short success")
	} else if !strings.Contains(err.Error(), "10") {
		t.Errorf("error = %q, want it to name the limit it hit", err)
	}
}

func TestGetJSONRefusesABodyThatIsNotJSON(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) { return 200, "not json" }}
	s := open(t, w)

	err := s.GetJSON(context.Background(), docURL(), &struct{}{})
	if err == nil {
		t.Fatal("a 200 carrying no JSON came back as success")
	}
}

func TestADocumentTheCommandWasNotGivenIsRefusedByTheGuard(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{}
	s, err := Open(guard.NewPolicy(), w) // nothing allowed
	if err != nil {
		t.Fatal(err)
	}
	err = s.GetJSON(context.Background(), docURL(), &struct{}{})
	if err == nil {
		t.Fatal("a document nobody named was read")
	}
	if !strings.Contains(err.Error(), "was not given to this command") {
		t.Errorf("error = %q, want the guard's refusal", err)
	}
	if n := len(w.requests()); n != 0 {
		t.Errorf("the fake wire saw %d requests; a refused request must not reach it", n)
	}
}

func TestOpenWithoutATokenNamesTheLogin(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	_, err := Open(guard.NewPolicy(), &wire{})
	if err == nil {
		t.Fatal("a session opened with no token")
	}
	if !errors.Is(err, auth.ErrNoToken) {
		t.Errorf("error = %v, want auth.ErrNoToken", err)
	}
	if !strings.Contains(err.Error(), "gdoc auth login") {
		t.Errorf("error = %q, want it to name gdoc auth login", err)
	}
}

func TestARefreshThatFailsIsNamedAndNothingIsSaved(t *testing.T) {
	path := tokenFile(t, time.Now().Add(-time.Hour))
	p := guard.NewPolicy()
	p.AllowFile(docID, guard.LevelSuggest)
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "oauth2.googleapis.com" {
			return reply(r, 400, `{"error":"invalid_grant"}`), nil
		}
		t.Errorf("a read went out after the refresh failed: %s", r.URL)
		return reply(r, 200, "{}"), nil
	})
	s, err := Open(p, rt)
	if err != nil {
		t.Fatal(err)
	}
	err = s.GetJSON(context.Background(), docURL(), &struct{}{})
	if err == nil {
		t.Fatal("a failed refresh came back as success")
	}
	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("error = %q, want the refusal named", err)
	}
	if got := savedToken(t, path).AccessToken; got != "OLD" {
		t.Errorf("saved access token = %q, want the file untouched", got)
	}
}

func TestWarningsIncludeTheSessionsOwn(t *testing.T) {
	tokenFile(t, time.Now().Add(-time.Hour))
	p := guard.NewPolicy()
	p.AllowFile(docID, guard.LevelSuggest)
	w := &wire{}
	s, err := Open(p, w)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.GetJSON(context.Background(), docURL(), &struct{}{}); err != nil {
		t.Fatal(err)
	}
	got := s.Warnings()
	if len(got) != 1 || !strings.Contains(got[0], "refreshed") {
		t.Fatalf("Warnings() = %v, want the session's own", got)
	}
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func hasWarning(warnings []string, want string) bool {
	for _, w := range warnings {
		if strings.Contains(w, want) {
			return true
		}
	}
	return false
}

// batchURL is the one write path the Docs API has, and the one the guard reads
// a writeMode out of.
func batchURL() string { return docURL() + ":batchUpdate" }

// suggestBatch is the smallest body the guard carries on a handed-in document:
// one request, and writeMode spelled the way the server reads it.
func suggestBatch() map[string]any {
	return map[string]any{
		"requests": []any{
			map[string]any{"insertText": map[string]any{
				"location": map[string]any{"index": 1},
				"text":     "hello",
			}},
		},
		"writeControl": map[string]any{"writeMode": "SUGGEST"},
	}
}

func TestPostJSONSendsTheBodyItWasGivenAndDecodesTheAnswer(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		return 200, `{"documentId":"` + docID + `","writeControl":{"requiredRevisionId":"REV2"}}`
	}}
	s := open(t, w)

	var into struct {
		DocumentID   string `json:"documentId"`
		WriteControl struct {
			RequiredRevisionID string `json:"requiredRevisionId"`
		} `json:"writeControl"`
	}
	if err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), &into); err != nil {
		t.Fatalf("PostJSON: %v", err)
	}
	if into.DocumentID != docID || into.WriteControl.RequiredRevisionID != "REV2" {
		t.Errorf("decoded %+v, want the document id and the revision", into)
	}
	reqs := w.requests()
	if len(reqs) != 1 {
		t.Fatalf("sent %d requests, want 1: %+v", len(reqs), reqs)
	}
	if reqs[0].Method != http.MethodPost {
		t.Errorf("method = %q, want POST", reqs[0].Method)
	}
	if reqs[0].ContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", reqs[0].ContentType)
	}
	if reqs[0].Accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", reqs[0].Accept)
	}
	if got := reqs[0].Auth; len(got) != 1 || got[0] != "Bearer OLD" {
		t.Errorf("Authorization = %v, want exactly one bearer", got)
	}
	// The bytes the wire carried are the bytes the caller's body encodes to.
	// The guard peeks the body it judges out of the same reader, so a body that
	// arrived changed would mean the judged bytes and the sent bytes differ.
	want, err := json.Marshal(suggestBatch())
	if err != nil {
		t.Fatal(err)
	}
	if reqs[0].Body != string(want) {
		t.Errorf("body on the wire = %q, want %q", reqs[0].Body, want)
	}
}

func TestPostJSONWithNothingToDecodeIntoDiscardsTheAnswer(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) { return 200, `{"replyId":"AAAB"}` }}
	s := open(t, w)

	if err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), nil); err != nil {
		t.Fatalf("PostJSON: %v", err)
	}
	if n := len(w.requests()); n != 1 {
		t.Errorf("sent %d requests, want 1", n)
	}
}

// TestA401OnAPostRefreshesOnceAndTheRetryCarriesTheSameBody is the reason the
// request is built fresh for each attempt. A body read once is a body the retry
// would send empty, and an empty batchUpdate is a write that quietly did
// nothing while the envelope said it had been sent.
func TestA401OnAPostRefreshesOnceAndTheRetryCarriesTheSameBody(t *testing.T) {
	path := tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		if n == 0 {
			return 401, `{"error":{"message":"Invalid Credentials"}}`
		}
		return 200, `{"documentId":"` + docID + `"}`
	}}
	s := open(t, w)

	if err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), nil); err != nil {
		t.Fatalf("PostJSON: %v", err)
	}
	reqs := w.requests()
	if len(reqs) != 3 {
		t.Fatalf("sent %d requests, want the post, the refresh and the retry: %+v", len(reqs), reqs)
	}
	if reqs[1].Host != "oauth2.googleapis.com" {
		t.Errorf("second request was to %s, want the token host", reqs[1].Host)
	}
	if got := reqs[2].Auth; len(got) != 1 || got[0] != "Bearer NEW" {
		t.Errorf("the retry carried %v, want the refreshed token", got)
	}
	if reqs[2].Body != reqs[0].Body {
		t.Errorf("the retry sent %q, want the same body as the first attempt %q", reqs[2].Body, reqs[0].Body)
	}
	if reqs[2].Body == "" {
		t.Error("the retry sent an empty body")
	}
	if got := savedToken(t, path).AccessToken; got != "NEW" {
		t.Errorf("saved access token = %q, want NEW", got)
	}
}

func TestAPostThatFailsCarriesTheStatusAndGooglesMessage(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		return 400, `{"error":{"code":400,"message":"Invalid requests[0].insertText: Index 30 must be less than the end index.","status":"INVALID_ARGUMENT"}}`
	}}
	s := open(t, w)

	err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), nil)
	if err == nil {
		t.Fatal("a 400 came back as success")
	}
	for _, want := range []string{"400", "Index 30 must be less than the end index."} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to carry %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "INVALID_ARGUMENT") {
		t.Errorf("error = %q, want Google's message and not the raw body", err)
	}
}

// TestADirectEditIsRefusedByTheGuardBeforeItReachesTheWire is the whole reason
// a write goes out through this package. The refusal is the guard's, inside the
// process, and the fake wire never sees the request at all.
func TestADirectEditIsRefusedByTheGuardBeforeItReachesTheWire(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{}
	s := open(t, w)

	body := suggestBatch()
	delete(body, "writeControl")

	err := s.PostJSON(context.Background(), batchURL(), body, nil)
	if err == nil {
		t.Fatal("a direct edit of a handed-in document was carried")
	}
	if !strings.Contains(err.Error(), "only SUGGEST is allowed") {
		t.Errorf("error = %q, want the guard's refusal", err)
	}
	if n := len(w.requests()); n != 0 {
		t.Errorf("the fake wire saw %d requests; a refused write must not reach it", n)
	}
}

func TestPostJSONRefusesAnAnswerThatIsNotJSON(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) { return 200, "not json" }}
	s := open(t, w)

	if err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), &struct{}{}); err == nil {
		t.Fatal("a 200 carrying no JSON came back as success")
	}
}

// TestAFailureAfterA2xxIsMarkedAsSent is the difference a writer package acts
// on. A guard refusal and an unreadable answer both come back as an error, and
// only one of them means the change never happened. A write reported as never
// sent is a write somebody sends again, on top of the first.
func TestAFailureAfterA2xxIsMarkedAsSent(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) { return 200, "not json" }}
	s := open(t, w)

	err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), &struct{}{})
	if err == nil {
		t.Fatal("a 200 carrying no JSON came back as success")
	}
	if !wasSent(err) {
		t.Errorf("error %q is not marked as sent, and the server had already taken the write", err)
	}
}

// TestAFailureBeforeTheWireIsNotMarkedAsSent is the other direction, and it is
// the half that matters most: a refusal the guard made inside the process must
// never read as a change that happened.
func TestAFailureBeforeTheWireIsNotMarkedAsSent(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{}
	s := open(t, w)

	body := suggestBatch()
	delete(body, "writeControl")

	err := s.PostJSON(context.Background(), batchURL(), body, nil)
	if err == nil {
		t.Fatal("a direct edit was carried")
	}
	if wasSent(err) {
		t.Errorf("a guard refusal is marked as sent: %q", err)
	}
}

// TestAFailedStatusIsNotMarkedAsSent keeps the mark to the case it means. A
// batchUpdate answering 4xx applied nothing, so the caller is right to treat it
// as a change that did not happen.
func TestAFailedStatusIsNotMarkedAsSent(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		return 400, `{"error":{"message":"Invalid requests[1].insertText"}}`
	}}
	s := open(t, w)

	err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), &struct{}{})
	if err == nil {
		t.Fatal("a 400 came back as success")
	}
	if wasSent(err) {
		t.Errorf("a refused write is marked as sent: %q", err)
	}
}

// wasSent asks the way the writer packages ask, by behaviour rather than by
// naming this package's own type.
func wasSent(err error) bool {
	var sent interface{ Sent() bool }
	return errors.As(err, &sent) && sent.Sent()
}

func TestPostJSONNamesABodyItCannotEncode(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{}
	s := open(t, w)

	err := s.PostJSON(context.Background(), batchURL(), make(chan int), nil)
	if err == nil {
		t.Fatal("a body that cannot be encoded came back as success")
	}
	if n := len(w.requests()); n != 0 {
		t.Errorf("the fake wire saw %d requests; a body that could not be encoded must not reach it", n)
	}
}

// TestPatchJSONSendsThePatchTheTrashNeeds is the one PATCH gdoc makes:
// files.update on a document gdoc created, to put it in the trash. A create
// gdoc cannot undo is a document left in somebody's Drive.
func TestPatchJSONSendsThePatchTheTrashNeeds(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		return 200, `{"id":"` + createdID + `","trashed":true}`
	}}
	s := openCreated(t, w)

	var into struct {
		Trashed bool `json:"trashed"`
	}
	body := map[string]any{"trashed": true}
	if err := s.PatchJSON(context.Background(), trashURL(), body, &into); err != nil {
		t.Fatalf("PatchJSON: %v", err)
	}
	if !into.Trashed {
		t.Error("the answer decoded with trashed false")
	}
	reqs := w.requests()
	if len(reqs) != 1 {
		t.Fatalf("sent %d requests, want 1", len(reqs))
	}
	if reqs[0].Method != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", reqs[0].Method)
	}
	if reqs[0].ContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", reqs[0].ContentType)
	}
	want, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if reqs[0].Body != string(want) {
		t.Errorf("body on the wire = %q, want %q", reqs[0].Body, want)
	}
}

// TestPatchJSONOnAHandedInFileIsRefusedByTheGuard is the level bar seen from
// the session: only a file gdoc created may be changed in place, and the
// refusal arrives before anything reaches the wire.
func TestPatchJSONOnAHandedInFileIsRefusedByTheGuard(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{}
	s := open(t, w)

	err := s.PatchJSON(context.Background(), "https://www.googleapis.com/drive/v3/files/"+docID+"?supportsAllDrives=true", map[string]any{"trashed": true}, nil)
	if err == nil {
		t.Fatal("a PATCH of a handed-in file was carried")
	}
	if !strings.Contains(err.Error(), "guard refused") {
		t.Errorf("error = %v, want the guard's refusal", err)
	}
	if n := len(w.requests()); n != 0 {
		t.Errorf("the wire saw %d requests, want none", n)
	}
}

// TestA401OnAPatchRefreshesOnceAndTheRetryCarriesTheSameBody is the POST rule
// on the other write verb: one refresh, one retry, and the same bytes.
func TestA401OnAPatchRefreshesOnceAndTheRetryCarriesTheSameBody(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		if n == 0 {
			return 401, `{"error":{"message":"Invalid Credentials"}}`
		}
		return 200, `{"trashed":true}`
	}}
	s := openCreated(t, w)

	body := map[string]any{"trashed": true}
	if err := s.PatchJSON(context.Background(), trashURL(), body, nil); err != nil {
		t.Fatalf("PatchJSON: %v", err)
	}
	reqs := w.requests()
	if len(reqs) != 3 {
		t.Fatalf("sent %d requests, want the patch, the refresh and the retry: %+v", len(reqs), reqs)
	}
	if reqs[1].Host != "oauth2.googleapis.com" {
		t.Errorf("second request was to %s, want the token host", reqs[1].Host)
	}
	if reqs[2].Body != reqs[0].Body || reqs[2].Body == "" {
		t.Errorf("the retry sent %q, want the same body as the first attempt %q", reqs[2].Body, reqs[0].Body)
	}
}

const createdID = "1CrEaTeD000000000000000000000000000000000"

func trashURL() string {
	return "https://www.googleapis.com/drive/v3/files/" + createdID + "?supportsAllDrives=true"
}

// openCreated builds a session whose one reachable file is one gdoc created,
// which is the only level a PATCH is carried at.
func openCreated(t *testing.T, w *wire) *Session {
	t.Helper()
	p := guard.NewPolicy()
	p.Learn(createdID)
	s, err := Open(p, w)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestASentErrorStillCarriesItsCause is the other half of wasSent. The writer
// packages ask by behaviour and never walk the chain, so nothing else here
// exercises the unwrap: a Sent() that answered true over a cause errors.Is
// could not reach would look correct in every existing test and silently break
// the first caller that asks what the failure actually was.
func TestASentErrorStillCarriesItsCause(t *testing.T) {
	cause := errors.New("the cause the caller asks for")

	err := mark(true, fmt.Errorf("the server answered 200 and then: %w", cause))

	if !wasSent(err) {
		t.Fatalf("err = %q, want it marked as sent", err)
	}
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is could not reach the cause through the sent error: %q", err)
	}
	if !errors.Is(mark(false, fmt.Errorf("nothing was sent: %w", cause)), cause) {
		t.Error("an unmarked error lost its cause too, so the wrapping itself is wrong")
	}
}

// The multipart write is M6's. A publish uploads a docx with conversion, which
// Drive takes as one multipart/related body: a JSON metadata part naming the
// folder and the title, then the file's bytes. The guard judges that body by
// reading the first part, so everything below is about the request the guard
// judged being the request that goes on the wire.

const uploadFolder = "0BxFolder00000000000000000000000000000000"

// uploadURL is the create route a multipart upload goes out on. The three
// signals the guard reads are here: the /upload path and uploadType=multipart,
// and PostMultipart sets the third, the media type.
func uploadURL() string {
	return "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&supportsAllDrives=true"
}

// openCreating builds a session whose only door is one folder to create in,
// which is the policy cmdPublish opens.
func openCreating(t *testing.T, w *wire) *Session {
	t.Helper()
	p := guard.NewPolicy()
	p.AllowCreateIn(uploadFolder)
	s, err := Open(p, w)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// uploadMeta is the metadata part a publish sends: the folder, the title and
// the conversion Drive is being asked for.
func uploadMeta() map[string]any {
	return map[string]any{
		"name":     "Supplier Register Policy",
		"parents":  []string{uploadFolder},
		"mimeType": "application/vnd.google-apps.document",
	}
}

const docxType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

// docxBytes stands in for a rendered document. It opens with the zip magic and
// carries a CR LF, so a body assembled by hand rather than by mime/multipart
// would show up as a part that ends in the wrong place.
var docxBytes = []byte("PK\x03\x04\r\n--not-a-boundary\r\nbytes")

// readParts splits a recorded request back into its parts, the way Drive's own
// parser would: the boundary comes from the Content-Type header and never from
// the body.
func readParts(t *testing.T, r seen) []struct {
	Type string
	Body []byte
} {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(r.ContentType)
	if err != nil {
		t.Fatalf("Content-Type %q cannot be read: %v", r.ContentType, err)
	}
	if mediaType != "multipart/related" {
		t.Fatalf("media type = %q, want multipart/related", mediaType)
	}
	if params["boundary"] == "" {
		t.Fatal("the Content-Type names no boundary, so nothing can tell where the metadata ends")
	}
	var out []struct {
		Type string
		Body []byte
	}
	mr := multipart.NewReader(strings.NewReader(r.Body), params["boundary"])
	for {
		part, err := mr.NextRawPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("part %d could not be read: %v", len(out)+1, err)
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("part %d could not be read: %v", len(out)+1, err)
		}
		if got := len(part.Header); got != 1 {
			t.Errorf("part %d carries %d headers %v, want only its Content-Type", len(out)+1, got, part.Header)
		}
		out = append(out, struct {
			Type string
			Body []byte
		}{part.Header.Get("Content-Type"), body})
	}
	return out
}

// TestPostMultipartSendsTheMetadataFirstAndTheFileSecond is the shape Drive
// reads and the shape the guard judges. The order is not a nicety: the guard
// reads the first part and stops, so a file part written first would be a
// create whose parents nothing can check.
func TestPostMultipartSendsTheMetadataFirstAndTheFileSecond(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		return 200, `{"id":"` + createdID + `","name":"Supplier Register Policy"}`
	}}
	s := openCreating(t, w)

	var into struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err := s.PostMultipart(context.Background(), uploadURL(), uploadMeta(), docxBytes, docxType, &into)
	if err != nil {
		t.Fatalf("PostMultipart: %v", err)
	}
	if into.ID != createdID || into.Name != "Supplier Register Policy" {
		t.Errorf("decoded %+v, want the new id and the title", into)
	}
	reqs := w.requests()
	if len(reqs) != 1 {
		t.Fatalf("sent %d requests, want 1: %+v", len(reqs), reqs)
	}
	if reqs[0].Method != http.MethodPost {
		t.Errorf("method = %q, want POST", reqs[0].Method)
	}
	if got := reqs[0].Auth; len(got) != 1 || got[0] != "Bearer OLD" {
		t.Errorf("Authorization = %v, want exactly one bearer", got)
	}
	if reqs[0].Accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", reqs[0].Accept)
	}
	parts := readParts(t, reqs[0])
	if len(parts) != 2 {
		t.Fatalf("the body carries %d parts, want the metadata then the file", len(parts))
	}
	if mediaType, _, err := mime.ParseMediaType(parts[0].Type); err != nil || mediaType != "application/json" {
		t.Errorf("the first part's Content-Type = %q, want application/json", parts[0].Type)
	}
	// The metadata is encoded with encoding/json, so nothing a note holds
	// reaches the wire as text somebody formatted.
	wantMeta, err := json.Marshal(uploadMeta())
	if err != nil {
		t.Fatal(err)
	}
	if string(parts[0].Body) != string(wantMeta) {
		t.Errorf("the metadata part = %q, want %q", parts[0].Body, wantMeta)
	}
	if parts[1].Type != docxType {
		t.Errorf("the file part's Content-Type = %q, want %q", parts[1].Type, docxType)
	}
	if string(parts[1].Body) != string(docxBytes) {
		t.Errorf("the file part = %q, want the bytes it was given %q", parts[1].Body, docxBytes)
	}
}

// TestPostMultipartCarriesTheCallersContext is the rule every other request
// here follows: a publish runs inside whatever bounds its caller has, and a
// request that built its own background context is the one nothing can cancel.
func TestPostMultipartCarriesTheCallersContext(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	type key struct{}
	var carried bool
	w := &wire{}
	w.answer = func(n int, r *http.Request) (int, string) {
		carried = r.Context().Value(key{}) == "the caller's"
		return 200, `{"id":"` + createdID + `"}`
	}
	s := openCreating(t, w)

	ctx := context.WithValue(context.Background(), key{}, "the caller's")
	if err := s.PostMultipart(ctx, uploadURL(), uploadMeta(), docxBytes, docxType, nil); err != nil {
		t.Fatalf("PostMultipart: %v", err)
	}
	if !carried {
		t.Error("the request did not carry the caller's context")
	}
}

// TestA401OnAMultipartRetriesWithTheSameBytesAndBoundary is why the body is
// built once, outside the closure send rebuilds the request through. A boundary
// generated per attempt would send bytes the guard never judged, which is the
// one thing this whole package exists to prevent.
func TestA401OnAMultipartRetriesWithTheSameBytesAndBoundary(t *testing.T) {
	path := tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		if n == 0 {
			return 401, `{"error":{"message":"Invalid Credentials"}}`
		}
		return 200, `{"id":"` + createdID + `"}`
	}}
	s := openCreating(t, w)

	if err := s.PostMultipart(context.Background(), uploadURL(), uploadMeta(), docxBytes, docxType, nil); err != nil {
		t.Fatalf("PostMultipart: %v", err)
	}
	reqs := w.requests()
	if len(reqs) != 3 {
		t.Fatalf("sent %d requests, want the upload, the refresh and the retry: %+v", len(reqs), reqs)
	}
	if reqs[1].Host != "oauth2.googleapis.com" {
		t.Errorf("second request was to %s, want the token host", reqs[1].Host)
	}
	if got := reqs[2].Auth; len(got) != 1 || got[0] != "Bearer NEW" {
		t.Errorf("the retry carried %v, want the refreshed token", got)
	}
	if reqs[2].Body != reqs[0].Body || reqs[2].Body == "" {
		t.Errorf("the retry sent %d bytes, want the same body as the first attempt's %d", len(reqs[2].Body), len(reqs[0].Body))
	}
	if reqs[2].ContentType != reqs[0].ContentType {
		t.Errorf("the retry declared %q, want the first attempt's boundary %q", reqs[2].ContentType, reqs[0].ContentType)
	}
	if got := savedToken(t, path).AccessToken; got != "NEW" {
		t.Errorf("saved access token = %q, want NEW", got)
	}
}

// TestTwoMultipartUploadsUseDifferentBoundaries keeps the boundary a fresh
// random value per call rather than a constant. A constant is a string a
// caller's own bytes could carry, and the part that ends early is then the
// metadata part the guard judged.
func TestTwoMultipartUploadsUseDifferentBoundaries(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) { return 200, `{"id":"` + createdID + `"}` }}
	s := openCreating(t, w)

	for i := 0; i < 2; i++ {
		if err := s.PostMultipart(context.Background(), uploadURL(), uploadMeta(), docxBytes, docxType, nil); err != nil {
			t.Fatalf("PostMultipart %d: %v", i, err)
		}
	}
	reqs := w.requests()
	if len(reqs) != 2 {
		t.Fatalf("sent %d requests, want 2", len(reqs))
	}
	if reqs[0].ContentType == reqs[1].ContentType {
		t.Errorf("both uploads declared %q, want a boundary drawn fresh each time", reqs[0].ContentType)
	}
}

// TestAMultipartAnswerThatIsNotJSONIsMarkedAsSent is the difference publish
// acts on. Drive made the document, and the answer carrying its id is what
// could not be read: a caller told the upload failed is a caller that uploads a
// second document.
func TestAMultipartAnswerThatIsNotJSONIsMarkedAsSent(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) { return 200, "not json" }}
	s := openCreating(t, w)

	err := s.PostMultipart(context.Background(), uploadURL(), uploadMeta(), docxBytes, docxType, &struct{}{})
	if err == nil {
		t.Fatal("a 200 carrying no JSON came back as success")
	}
	if !wasSent(err) {
		t.Errorf("error %q is not marked as sent, and Drive had already made the document", err)
	}
}

// TestAMultipartCreateTheGuardRefusesIsNotMarkedAsSent is the other direction,
// and it is the half that matters most: an upload naming a folder this run was
// never given must never read as a document that exists.
func TestAMultipartCreateTheGuardRefusesIsNotMarkedAsSent(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{}
	s := openCreating(t, w)

	meta := uploadMeta()
	meta["parents"] = []string{"0BxSomeoneElsesFolder000000000000000000"}

	err := s.PostMultipart(context.Background(), uploadURL(), meta, docxBytes, docxType, nil)
	if err == nil {
		t.Fatal("an upload into a folder nobody named was carried")
	}
	if !strings.Contains(err.Error(), "guard refused") {
		t.Errorf("error = %q, want the guard's refusal", err)
	}
	if wasSent(err) {
		t.Errorf("a guard refusal is marked as sent: %q", err)
	}
	if n := len(w.requests()); n != 0 {
		t.Errorf("the fake wire saw %d requests; a refused upload must not reach it", n)
	}
}

// TestPostMultipartNamesMetadataItCannotEncode keeps a body that cannot be
// built off the wire, the way PostJSON does.
func TestPostMultipartNamesMetadataItCannotEncode(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{}
	s := openCreating(t, w)

	err := s.PostMultipart(context.Background(), uploadURL(), make(chan int), docxBytes, docxType, nil)
	if err == nil {
		t.Fatal("metadata that cannot be encoded came back as success")
	}
	if wasSent(err) {
		t.Errorf("a body that never left the machine is marked as sent: %q", err)
	}
	if n := len(w.requests()); n != 0 {
		t.Errorf("the fake wire saw %d requests; a body that could not be encoded must not reach it", n)
	}
}

// TestAMultipartUploadIsCarriedAndTeachesThePolicy is the two halves of M6
// meeting: the body gapi builds is the body the guard's own parser reads, and
// the id that comes back is learned from the create the guard carried. Written
// separately they could each be right and still disagree.
func TestAMultipartUploadIsCarriedAndTeachesThePolicy(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) { return 200, `{"id":"` + createdID + `"}` }}
	p := guard.NewPolicy()
	p.AllowCreateIn(uploadFolder)
	s, err := Open(p, w)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PostMultipart(context.Background(), uploadURL(), uploadMeta(), docxBytes, docxType, nil); err != nil {
		t.Fatalf("PostMultipart: %v", err)
	}
	if n := len(w.requests()); n != 1 {
		t.Fatalf("sent %d requests, want the upload", n)
	}
	// The new document is reachable now, and only because the guard carried the
	// create that made it. A trash on the rollback path depends on this.
	if err := s.PatchJSON(context.Background(), trashURL(), map[string]any{"trashed": true}, nil); err != nil {
		t.Errorf("the id the create returned was not learned: %v", err)
	}
	if len(p.Warnings()) != 0 {
		t.Errorf("Warnings() = %v, want none on a create whose answer read", p.Warnings())
	}
}
