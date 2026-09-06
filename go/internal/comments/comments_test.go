package comments

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/guard"
)

const testDocID = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"

// judged is the strongest thing a test can say about a URL this package builds:
// not that it matches a list written twice, but that the guard carries it. A
// parameter the guard does not name on comments.list is a request that dies
// inside the process, and a test comparing two copies of the same list would
// pass while the command failed.
func judged(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("ListURL built a URL that does not parse: %v", err)
	}
	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)
	if err := p.Judge("GET", u, nil); err != nil {
		t.Fatalf("the guard refused the comments read: %v", err)
	}
	return u
}

func TestListURLIsACallTheGuardCarries(t *testing.T) {
	u := judged(t, ListURL(testDocID, "", ""))

	if u.Scheme != "https" || u.Host != "www.googleapis.com" {
		t.Errorf("host = %s://%s", u.Scheme, u.Host)
	}
	if u.Path != "/drive/v3/files/"+testDocID+"/comments" {
		t.Errorf("path = %q", u.Path)
	}
	q := u.Query()
	if q.Get("pageSize") != "100" {
		t.Errorf("pageSize = %q", q.Get("pageSize"))
	}
	if q.Get("includeDeleted") != "false" {
		t.Errorf("includeDeleted = %q; a deleted comment is not a thread to answer", q.Get("includeDeleted"))
	}
	if q.Get("fields") == "" {
		t.Fatal("the read carries no field mask, so it asks for every field")
	}
	for _, name := range []string{"replies", "quotedFileContent", "resolved", "modifiedTime"} {
		if !strings.Contains(q.Get("fields"), name) {
			t.Errorf("the field mask does not name %q, and the thread needs it", name)
		}
	}
	if strings.Contains(q.Get("fields"), "permission") {
		t.Error("the field mask names the permission surface")
	}
	if _, has := q["pageToken"]; has {
		t.Error("the first page asks for a page token")
	}
	if _, has := q["startModifiedTime"]; has {
		t.Error("a read with no cursor carries startModifiedTime")
	}
}

func TestListURLAddsThePageTokenAndTheCursorOnlyWhenGiven(t *testing.T) {
	u := judged(t, ListURL(testDocID, "PAGE2", "2026-09-06T11:00:00Z"))
	q := u.Query()
	if q.Get("pageToken") != "PAGE2" {
		t.Errorf("pageToken = %q", q.Get("pageToken"))
	}
	if q.Get("startModifiedTime") != "2026-09-06T11:00:00Z" {
		t.Errorf("startModifiedTime = %q", q.Get("startModifiedTime"))
	}
}

// fakeReader answers with one fixture per request, in order, and records the
// URLs it was asked for. Nothing here names net/http: the rooms that may are
// the ones the boundary test lists, and this is not one of them.
type fakeReader struct {
	seen  []string
	pages [][]byte
	err   error
}

func (f *fakeReader) GetJSON(_ context.Context, rawURL string, into any) error {
	f.seen = append(f.seen, rawURL)
	if f.err != nil {
		return f.err
	}
	if len(f.pages) == 0 {
		return json.Unmarshal([]byte(`{}`), into)
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return json.Unmarshal(page, into)
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func bothPages(t *testing.T) *fakeReader {
	t.Helper()
	return &fakeReader{pages: [][]byte{fixture(t, "page-one.json"), fixture(t, "page-two.json")}}
}

func TestFetchFollowsTheNextPageAndStops(t *testing.T) {
	f := bothPages(t)
	raw, err := Fetch(context.Background(), f, testDocID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seen) != 2 {
		t.Fatalf("the reader made %d requests, want 2: one page, then the token's page, then stop", len(f.seen))
	}
	first, _ := url.Parse(f.seen[0])
	if _, has := first.Query()["pageToken"]; has {
		t.Error("the first request carried a page token")
	}
	second, _ := url.Parse(f.seen[1])
	if got := second.Query().Get("pageToken"); got != "PAGE2" {
		t.Errorf("the second request carried pageToken %q, want PAGE2", got)
	}
	if len(raw) != 3 {
		t.Fatalf("Fetch returned %d comments, want 3 across the two pages", len(raw))
	}
	if raw[0].ID != "AAAA1111" || raw[2].ID != "CCCC3333" {
		t.Errorf("Fetch returned %q then %q, want the pages in order", raw[0].ID, raw[2].ID)
	}
}

func TestFetchPutsTheCursorOnEveryPage(t *testing.T) {
	f := bothPages(t)
	at := time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC)
	if _, err := Fetch(context.Background(), f, testDocID, &Cursor{At: at}); err != nil {
		t.Fatal(err)
	}
	for i, raw := range f.seen {
		u, _ := url.Parse(raw)
		if got := u.Query().Get("startModifiedTime"); got != "2026-09-06T11:00:00Z" {
			t.Errorf("request %d carried startModifiedTime %q", i, got)
		}
	}
}

func TestFetchCarriesTheSessionsRefusalOut(t *testing.T) {
	refusal := errRefused{}
	f := &fakeReader{err: refusal}
	if _, err := Fetch(context.Background(), f, testDocID, nil); err != refusal {
		t.Fatalf("Fetch returned %v, want the session's refusal unwrapped", err)
	}
}

type errRefused struct{}

func (errRefused) Error() string { return `document "OTHER" was not given to this command` }

func TestMarkerIsTheFirstTokenAndTheMatchIsExact(t *testing.T) {
	cases := map[string]string{
		"ai: answer this":          "ai:",
		"ai? which register":       "ai?",
		"ai! rewrite this":         "ai!",
		"   ai:  leading space":    "ai:",
		"ai:no space after it":     "none",
		"AI: the match is exact":   "none",
		"Ai? still not a marker":   "none",
		"please ai: not the first": "none",
		"just a comment":           "none",
		"":                         "none",
		"ai:":                      "ai:",
		"ai:\nsecond line":         "ai:",
	}
	for content, want := range cases {
		if got := markerOf(content); got != want {
			t.Errorf("markerOf(%q) = %q, want %q", content, got, want)
		}
	}
}

func TestByGdocIsTrueOnlyForAReplyOpeningWithTheRobot(t *testing.T) {
	cases := map[string]bool{
		"🤖 The 2026 register.":      true,
		"  🤖 after a space":         true,
		"thanks, that answers it":   false,
		"the 🤖 is not at the front": false,
		"":                          false,
	}
	for content, want := range cases {
		if got := byGdoc(content); got != want {
			t.Errorf("byGdoc(%q) = %v, want %v", content, got, want)
		}
	}
}

// document is the Docs half of the join: the ranges, keyed by the Drive comment
// id. Only AAAA1111 is placed, so the other two threads are the unplaced case.
func document() *docs.Document {
	return &docs.Document{
		ID: testDocID,
		CommentRanges: map[string]docs.Range{
			"AAAA1111": {Tab: "t.0", Start: 1204, End: 1223},
		},
	}
}

func threadsFromFixtures(t *testing.T) ([]Thread, []string) {
	t.Helper()
	raw, err := Fetch(context.Background(), bothPages(t), testDocID, nil)
	if err != nil {
		t.Fatal(err)
	}
	return Threads(raw, document())
}

func TestThreadsJoinsTheRangeDriveDoesNotCarry(t *testing.T) {
	got, unplaced := threadsFromFixtures(t)
	if len(got) != 3 {
		t.Fatalf("Threads returned %d threads, want 3", len(got))
	}
	first := got[0]
	if first.Range == nil {
		t.Fatal("the placed thread came back without its range")
	}
	if *first.Range != (docs.Range{Tab: "t.0", Start: 1204, End: 1223}) {
		t.Errorf("range = %+v", *first.Range)
	}
	if got[1].Range != nil || got[2].Range != nil {
		t.Error("a thread the Docs read did not place came back with a range")
	}
	want := []string{"BBBB2222", "CCCC3333"}
	if !reflect.DeepEqual(unplaced, want) {
		t.Errorf("unplaced = %v, want %v", unplaced, want)
	}
}

func TestThreadsCarriesEveryFactAndJudgesNone(t *testing.T) {
	got, _ := threadsFromFixtures(t)
	first := got[0]
	if first.ID != "AAAA1111" || first.Author != "Nail Khusnullin" {
		t.Errorf("id = %q, author = %q", first.ID, first.Author)
	}
	if first.Created != "2026-09-06T10:00:00Z" || first.Modified != "2026-09-06T10:30:00Z" {
		t.Errorf("created = %q, modified = %q", first.Created, first.Modified)
	}
	if first.Content != "ai? which register does this refer to" || first.Marker != "ai?" {
		t.Errorf("content = %q, marker = %q", first.Content, first.Marker)
	}
	if first.Quoted != "the operations team" {
		t.Errorf("quoted = %q", first.Quoted)
	}
	if first.Resolved {
		t.Error("an open thread came back resolved")
	}
	if len(first.Replies) != 2 {
		t.Fatalf("the thread carries %d replies, want 2", len(first.Replies))
	}
	if !first.Replies[0].ByGdoc || first.Replies[1].ByGdoc {
		t.Errorf("by_gdoc = %v then %v, want true then false", first.Replies[0].ByGdoc, first.Replies[1].ByGdoc)
	}
	if first.Replies[1].Author != "Ada Lovelace" || first.Replies[1].Created != "2026-09-06T10:45:00Z" {
		t.Errorf("the second reply = %+v", first.Replies[1])
	}
	if first.Witness != "" {
		t.Errorf("witness = %q, and nothing in this package sets it", first.Witness)
	}
	if got[1].Marker != "ai!" {
		t.Errorf("the leading-space marker = %q", got[1].Marker)
	}
	if !got[2].Resolved {
		t.Error("the resolved thread came back open")
	}
	if got[2].Marker != "none" {
		t.Errorf("AI: read as marker %q; the match is exact", got[2].Marker)
	}
}

func TestThreadsOnAnEmptyListIsEmpty(t *testing.T) {
	got, unplaced := Threads(nil, document())
	if len(got) != 0 || len(unplaced) != 0 {
		t.Errorf("Threads(nil) = %v, %v", got, unplaced)
	}
}

func TestThreadsWithoutADocumentPlacesNothing(t *testing.T) {
	// The witness command reads threads without a Docs read behind them. A nil
	// document is no ranges, not a crash.
	raw, err := Fetch(context.Background(), bothPages(t), testDocID, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, unplaced := Threads(raw, nil)
	if len(got) != 3 || len(unplaced) != 3 {
		t.Fatalf("Threads without a document = %d threads, %d unplaced", len(got), len(unplaced))
	}
}

func TestAThreadWithNoRepliesCarriesAnEmptyList(t *testing.T) {
	// nil replies would print as `"replies": null`, and a caller reading the
	// JSON has to special-case it. An empty list says the same thing in the
	// shape every other thread has.
	got, _ := threadsFromFixtures(t)
	if got[1].Replies == nil {
		t.Error("a thread with no replies came back with a null reply list")
	}
	b, err := json.Marshal(got[1])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"replies":[]`) {
		t.Errorf("the thread printed as %s", b)
	}
	if !strings.Contains(string(b), `"range":null`) {
		t.Errorf("an unplaced thread must print range: null, got %s", b)
	}
}
