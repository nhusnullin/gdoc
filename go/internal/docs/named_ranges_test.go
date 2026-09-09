package docs

import (
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gdoc/internal/guard"
)

func namedRangesFixture(t *testing.T) *Document {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "named-ranges.json"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("the named ranges fixture did not decode: %v", err)
	}
	return d
}

// Every v2 read carries includeTabsContent=true, which moves the named ranges
// into the tab and leaves the top level empty. A decoder reading the top level
// answers nothing on every document gdoc actually reads, and says so by
// reporting no named ranges rather than by failing.
func TestNamedRangesAreReadFromTheTabAndNotTheTopLevel(t *testing.T) {
	d := namedRangesFixture(t)

	var ids []string
	for _, nr := range d.NamedRanges() {
		ids = append(ids, nr.ID)
	}
	want := []string{"kix.abc123", "kix.def456", "kix.ghi789", "kix.jkl012"}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("NamedRanges() ids = %v, want %v", ids, want)
	}
	for _, id := range ids {
		if id == "kix.toplevel" {
			t.Error("the top-level namedRanges of a tabbed read were decoded; that field is empty on every read gdoc makes")
		}
	}
}

// The id is the identifier and the name is a label. Two named ranges may carry
// one name, and Docs puts both under that one key, so a decoder keyed by name
// keeps one of them and loses the other.
func TestTwoNamedRangesSharingANameKeepTheirOwnIDs(t *testing.T) {
	d := namedRangesFixture(t)

	var shared []NamedRange
	for _, nr := range d.NamedRanges() {
		if nr.Name == "gdoc-checklist" {
			shared = append(shared, nr)
		}
	}
	if len(shared) != 3 {
		t.Fatalf("%d named ranges called gdoc-checklist, want 3: two in the first tab and one in the child", len(shared))
	}
	seen := map[string]bool{}
	for _, nr := range shared {
		if seen[nr.ID] {
			t.Errorf("the id %q was decoded twice", nr.ID)
		}
		seen[nr.ID] = true
	}
	for _, id := range []string{"kix.abc123", "kix.def456", "kix.jkl012"} {
		if !seen[id] {
			t.Errorf("the id %q is missing, so one of two ranges sharing a name was dropped", id)
		}
	}
}

// A named range belongs to the tab it was decoded from, and its ranges carry
// the position. A range naming a segment is in a header, a footer or a
// footnote rather than in the body, and that is reported rather than dropped:
// a header range read as a body range names a position the body does not have.
func TestANamedRangeCarriesItsTabItsRangesAndItsSegments(t *testing.T) {
	d := namedRangesFixture(t)

	var owner NamedRange
	for _, nr := range d.NamedRanges() {
		if nr.ID == "kix.ghi789" {
			owner = nr
		}
	}
	if owner.Name != "owner-block" || owner.Tab != "t.0" {
		t.Fatalf("owner-block decoded as %+v", owner)
	}
	want := []Range{
		{Tab: "t.0", Start: 1, End: 11},
		{Tab: "t.0", Start: 4, End: 9, Segment: "h.headerone"},
	}
	if !reflect.DeepEqual(owner.Ranges, want) {
		t.Errorf("owner-block ranges = %+v, want %+v", owner.Ranges, want)
	}

	child := d.Tabs[1]
	if child.ID != "t.1" {
		t.Fatalf("the second tab is %q", child.ID)
	}
	if len(child.NamedRanges) != 1 || child.NamedRanges[0].Tab != "t.1" {
		t.Errorf("the child tab's named ranges = %+v", child.NamedRanges)
	}
	if len(d.Tabs[0].NamedRanges) != 3 {
		t.Errorf("the first tab holds %d named ranges, want 3", len(d.Tabs[0].NamedRanges))
	}
}

// A document written before tabs existed answers with a top-level body, and its
// named ranges sit beside that body. Reading the body from one place and the
// named ranges from another would report no named ranges on exactly the
// documents whose ranges are at the top level.
func TestAPreTabsDocumentsNamedRangesAreReadFromTheTopLevel(t *testing.T) {
	raw := []byte(`{
	  "documentId": "PRETABS", "title": "Before tabs", "revisionId": "ALm37BXpre",
	  "body": {"content": [
	    {"startIndex": 1, "endIndex": 12, "paragraph": {
	      "paragraphStyle": {"namedStyleType": "NORMAL_TEXT"},
	      "elements": [{"startIndex": 1, "endIndex": 12, "textRun": {"content": "Old text.\n"}}]}}
	  ]},
	  "namedRanges": {"legacy": {"name": "legacy", "namedRanges": [
	    {"namedRangeId": "kix.old1", "name": "legacy", "ranges": [{"startIndex": 1, "endIndex": 5}]}
	  ]}}
	}`)
	d, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := d.NamedRanges()
	if len(got) != 1 || got[0].ID != "kix.old1" || got[0].Tab != defaultTabID {
		t.Fatalf("a pre-tabs document decoded %+v, want one range kix.old1 in %s", got, defaultTabID)
	}
	// The range names no tab of its own, because the document has none to name.
	if len(got[0].Ranges) != 1 || got[0].Ranges[0].Tab != defaultTabID {
		t.Errorf("ranges = %+v, want the implicit tab", got[0].Ranges)
	}
}

// A document with no named ranges reports none, and nothing here invents an
// empty list to hold.
func TestADocumentWithNoNamedRangesReportsNone(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "single-tab.json"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := d.NamedRanges(); len(got) != 0 {
		t.Errorf("NamedRanges() = %+v on a document holding none", got)
	}
}

// The narrow read is the whole point of having a second URL: it asks for the
// named ranges and not the document's content. docsReadParams already permits
// a fields mask, so this call needs no guard change, and this test is what says
// so rather than a comment claiming it.
func TestNamedRangesURLIsACallTheGuardCarries(t *testing.T) {
	u, err := url.Parse(NamedRangesURL(testDocID))
	if err != nil {
		t.Fatalf("NamedRangesURL built a URL that does not parse: %v", err)
	}
	if u.Path != "/v1/documents/"+testDocID {
		t.Errorf("path = %q", u.Path)
	}
	q := u.Query()
	if q.Get("includeTabsContent") != "true" {
		t.Errorf("includeTabsContent = %q; without it the named ranges of the other tabs are not in the answer", q.Get("includeTabsContent"))
	}
	fields := q.Get("fields")
	if !strings.Contains(fields, "namedRanges") {
		t.Errorf("fields = %q, and it does not ask for the named ranges", fields)
	}
	if strings.Contains(fields, "*") {
		t.Errorf("fields = %q asks for every field, which the guard refuses", fields)
	}
	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)
	if err := p.Judge("GET", u, nil); err != nil {
		t.Fatalf("the guard refused the named ranges read: %v", err)
	}
}

// The narrow read is parsed by the same decoder, so an answer carrying only the
// tabs and their named ranges is a Document with no blocks and the ranges in
// it. A second parser would be a second shape to keep in step.
func TestTheNarrowReadParsesThroughTheSameDecoder(t *testing.T) {
	raw := []byte(`{
	  "documentId": "1AbC",
	  "tabs": [{"tabProperties": {"tabId": "t.0"}, "documentTab": {"namedRanges": {
	    "checklist": {"name": "checklist", "namedRanges": [
	      {"namedRangeId": "kix.only", "name": "checklist", "ranges": [{"startIndex": 3, "endIndex": 9, "tabId": "t.0"}]}
	    ]}
	  }}}]
	}`)
	d, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Tabs) != 1 || len(d.Tabs[0].Body) != 0 {
		t.Fatalf("the narrow answer parsed into %d tabs with %d blocks", len(d.Tabs), len(d.Tabs[0].Body))
	}
	got := d.NamedRanges()
	if len(got) != 1 || got[0].ID != "kix.only" || got[0].Name != "checklist" {
		t.Fatalf("NamedRanges() = %+v", got)
	}
}
