package docs

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"gdoc/internal/guard"
)

const testDocID = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"

func TestURLCarriesTheThreeParametersAndNothingElse(t *testing.T) {
	u, err := url.Parse(URL(testDocID))
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "https" || u.Host != "docs.googleapis.com" {
		t.Errorf("host = %s://%s", u.Scheme, u.Host)
	}
	if u.Path != "/v1/documents/"+testDocID {
		t.Errorf("path = %q", u.Path)
	}
	want := map[string]string{
		"includeTabsContent":  "true",
		"suggestionsViewMode": "SUGGESTIONS_INLINE",
		"commentsViewMode":    "COMMENTS_VIEW_MODE_INCLUDED",
	}
	got := u.Query()
	if len(got) != len(want) {
		t.Fatalf("query = %v, want exactly %v", got, want)
	}
	for k, v := range want {
		if got.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, got.Get(k), v)
		}
	}
}

func TestParseRefusesABodyThatIsNotJSON(t *testing.T) {
	if _, err := Parse([]byte("<html>no</html>")); err == nil {
		t.Fatal("Parse accepted a body that is not JSON")
	}
}

// fakeReader is the seam docs reads through: one JSON GET. It records the URL
// it was asked for, so a test can assert on the request the reader built, and
// it answers with a fixture. Nothing here names net/http, which is the point:
// the four rooms that may name it are the four the boundary test lists.
type fakeReader struct {
	seen []string
	body []byte
	err  error
}

func (f *fakeReader) GetJSON(_ context.Context, rawURL string, into any) error {
	f.seen = append(f.seen, rawURL)
	if f.err != nil {
		return f.err
	}
	return json.Unmarshal(f.body, into)
}

func TestFetchSendsTheThreeParametersAndParsesTheAnswer(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "single-tab.json"))
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeReader{body: raw}
	d, err := Fetch(context.Background(), f, testDocID)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seen) != 1 {
		t.Fatalf("the reader made %d requests, want 1", len(f.seen))
	}
	u, err := url.Parse(f.seen[0])
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("includeTabsContent") != "true" {
		t.Errorf("includeTabsContent = %q; a read without it sees the first tab only", q.Get("includeTabsContent"))
	}
	if q.Get("suggestionsViewMode") != "SUGGESTIONS_INLINE" {
		t.Errorf("suggestionsViewMode = %q", q.Get("suggestionsViewMode"))
	}
	if q.Get("commentsViewMode") != "COMMENTS_VIEW_MODE_INCLUDED" {
		t.Errorf("commentsViewMode = %q", q.Get("commentsViewMode"))
	}
	if u.Path != "/v1/documents/"+testDocID {
		t.Errorf("path = %q", u.Path)
	}
	if d.Title != "Supplier register policy" || len(d.Tabs) != 1 {
		t.Errorf("Fetch parsed %q with %d tabs", d.Title, len(d.Tabs))
	}
}

func TestFetchCarriesTheSessionsRefusalOut(t *testing.T) {
	// A document the command was not given is refused by the guard, inside the
	// process. The reader has nothing to add to that sentence, so it hands it
	// back unwrapped rather than reporting a parse failure over the top of it.
	refusal := errors.New("document \"OTHER\" was not given to this command")
	f := &fakeReader{err: refusal}
	if _, err := Fetch(context.Background(), f, testDocID); !errors.Is(err, refusal) {
		t.Fatalf("Fetch returned %v, want the session's refusal", err)
	}
}

// TestURLIsACallTheGuardCarries is the strongest thing this file can say about
// the read URL: not that it matches a list written twice, but that the guard
// carries it. comments and docx each prove their own URL this way; without it,
// a parameter added here that docsReadParams does not name passes the whole
// offline suite and fails only against Google.
func TestURLIsACallTheGuardCarries(t *testing.T) {
	u, err := url.Parse(URL(testDocID))
	if err != nil {
		t.Fatalf("URL built a URL that does not parse: %v", err)
	}
	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)
	if err := p.Judge("GET", u, nil); err != nil {
		t.Fatalf("the guard refused the Docs read: %v", err)
	}
}

// Places is the one rule read marks from and comments reports from, so the four
// shapes it refuses are named here rather than left to the two callers.
func TestPlacesRefusesEveryRangeThatNamesNoPosition(t *testing.T) {
	d := &Document{Tabs: []Tab{
		{ID: "t.0", Body: []Block{{Paragraph: &Paragraph{
			StartIndex: 1, EndIndex: 20,
			Runs: []Run{{Kind: KindText, StartIndex: 1, EndIndex: 20}},
		}}}},
		{ID: "t.1", Body: []Block{{Table: Table{{{Blocks: []Block{{Paragraph: &Paragraph{
			StartIndex: 4, EndIndex: 10,
			Runs: []Run{{Kind: KindText, StartIndex: 4, EndIndex: 10}},
		}}}}}}}}},
	}}

	for _, c := range []struct {
		why  string
		r    Range
		want bool
	}{
		{"a range inside the text", Range{Tab: "t.0", Start: 3, End: 9}, true},
		{"a range closing at the run's end index", Range{Tab: "t.0", Start: 1, End: 20}, true},
		{"a range inside a table cell in another tab", Range{Tab: "t.1", Start: 4, End: 10}, true},
		{"an inverted pair", Range{Tab: "t.0", Start: 9, End: 3}, false},
		{"an empty pair", Range{Tab: "t.0", Start: 4, End: 4}, false},
		{"an end past the text", Range{Tab: "t.0", Start: 3, End: 900}, false},
		{"a start before the text", Range{Tab: "t.0", Start: 0, End: 9}, false},
		{"a tab the document does not have", Range{Tab: "t.9", Start: 3, End: 9}, false},
		{"no tab named at all", Range{Start: 3, End: 9}, false},
		{"a range in one tab named against another", Range{Tab: "t.1", Start: 3, End: 9}, false},
	} {
		if got := d.Places(c.r); got != c.want {
			t.Errorf("Places(%+v) = %v, want %v: %s", c.r, got, c.want, c.why)
		}
	}
}

// Tab.Places answers for one tab, and Document.Places is that over every tab.
// The two forms only part company when two tabs share an id, which the decoder
// can hand over because a tab with no id at all is called t.0. A walker asks
// the tab it is walking: answering off the document would arm a comment marker
// on indexes the walked tab's own text does not hold, and the walk would end
// with an opening marker and no close.
func TestTabPlacesAnswersForItsOwnTabWhenTwoTabsShareAnID(t *testing.T) {
	first := Tab{ID: "t.0", Body: []Block{{Paragraph: &Paragraph{
		StartIndex: 1, EndIndex: 2000,
		Runs: []Run{{Kind: KindText, StartIndex: 1, EndIndex: 2000}},
	}}}}
	second := Tab{ID: "t.0", Body: []Block{{Paragraph: &Paragraph{
		StartIndex: 1, EndIndex: 1210,
		Runs: []Run{{Kind: KindText, StartIndex: 1, EndIndex: 1210}},
	}}}}
	d := &Document{Tabs: []Tab{first, second}}
	r := Range{Tab: "t.0", Start: 1204, End: 1223}

	if !first.Places(r) {
		t.Error("the first tab refused a range inside its own text")
	}
	if second.Places(r) {
		t.Error("the second tab placed a range that ends past its own text")
	}
	if !d.Places(r) {
		t.Error("the document refused a range one of its tabs holds")
	}
}
