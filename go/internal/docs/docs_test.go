package docs

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"
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
