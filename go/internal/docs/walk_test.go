package docs

import (
	"os"
	"path/filepath"
	"testing"
)

// fixture parses a testdata file the way Fetch parses a body off the wire, so
// every test below runs the code the wire feeds.
func fixture(t *testing.T, name string) *Document {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse(%s): %v", name, err)
	}
	return d
}

// paragraphs is every paragraph of a tab's body in reading order, tables aside.
func paragraphs(blocks []Block) []*Paragraph {
	var out []*Paragraph
	for _, b := range blocks {
		if b.Paragraph != nil {
			out = append(out, b.Paragraph)
		}
	}
	return out
}

// runText is a paragraph's runs joined, which is the text before any projection.
func runText(p *Paragraph) string {
	s := ""
	for _, r := range p.Runs {
		s += r.Text
	}
	return s
}

func TestDocumentHeaderFields(t *testing.T) {
	d := fixture(t, "single-tab.json")
	if d.ID != "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd" {
		t.Errorf("ID = %q", d.ID)
	}
	if d.Title != "Supplier register policy" {
		t.Errorf("Title = %q", d.Title)
	}
	if d.RevisionID != "ALm37BXsingleTab" {
		t.Errorf("RevisionID = %q", d.RevisionID)
	}
	if d.MultiTab() {
		t.Error("MultiTab() = true for a one-tab document")
	}
}

func TestTwoTabsAreTwoTabs(t *testing.T) {
	d := fixture(t, "two-tabs.json")
	if len(d.Tabs) != 2 {
		t.Fatalf("len(Tabs) = %d, want 2", len(d.Tabs))
	}
	if !d.MultiTab() {
		t.Error("MultiTab() = false with two tabs")
	}
	if d.Tabs[0].ID != "t.0" || d.Tabs[0].Title != "Overview" {
		t.Errorf("first tab = %q %q", d.Tabs[0].ID, d.Tabs[0].Title)
	}
	// A child tab is a tab. It comes after its parent, which is the order the
	// document shows it in.
	if d.Tabs[1].ID != "t.1" || d.Tabs[1].Title != "Detail" {
		t.Errorf("second tab = %q %q", d.Tabs[1].ID, d.Tabs[1].Title)
	}
	if got := runText(paragraphs(d.Tabs[1].Body)[1]); got != "The second tab's text.\n" {
		t.Errorf("second tab body = %q", got)
	}
}

func TestPreTabsShapeIsOneTab(t *testing.T) {
	d := fixture(t, "pre-tabs.json")
	if len(d.Tabs) != 1 {
		t.Fatalf("len(Tabs) = %d, want 1", len(d.Tabs))
	}
	if d.Tabs[0].ID != "t.0" || d.Tabs[0].Title != "" {
		t.Errorf("tab = %q %q, want t.0 with no title", d.Tabs[0].ID, d.Tabs[0].Title)
	}
	if d.MultiTab() {
		t.Error("MultiTab() = true for the pre-tabs shape")
	}
	ps := paragraphs(d.Tabs[0].Body)
	if len(ps) != 2 {
		t.Fatalf("len(paragraphs) = %d, want 2", len(ps))
	}
	if ps[0].Style != "TITLE" {
		t.Errorf("first paragraph style = %q", ps[0].Style)
	}
}

func TestHeadingsAndBulletsCarryTheirStyle(t *testing.T) {
	ps := paragraphs(fixture(t, "single-tab.json").Tabs[0].Body)
	if len(ps) != 6 {
		t.Fatalf("len(paragraphs) = %d, want 6", len(ps))
	}
	if ps[0].Style != "HEADING_1" || runText(ps[0]) != "Scope\n" {
		t.Errorf("first paragraph = %q %q", ps[0].Style, runText(ps[0]))
	}
	if ps[0].StartIndex != 1 || ps[0].EndIndex != 7 {
		t.Errorf("first paragraph indexes = %d..%d, want 1..7", ps[0].StartIndex, ps[0].EndIndex)
	}
	if ps[2].Style != "HEADING_2" {
		t.Errorf("third paragraph style = %q, want HEADING_2", ps[2].Style)
	}
	if ps[3].Bullet == nil || ps[3].Bullet.NestingLevel != 0 {
		t.Errorf("first bullet = %+v, want nesting level 0", ps[3].Bullet)
	}
	if ps[4].Bullet == nil || ps[4].Bullet.NestingLevel != 1 {
		t.Errorf("second bullet = %+v, want nesting level 1", ps[4].Bullet)
	}
	if ps[1].Bullet != nil {
		t.Error("a plain paragraph carries a bullet")
	}
}

func TestTableTextLandsInCellsInReadingOrder(t *testing.T) {
	var tbl Table
	for _, b := range fixture(t, "single-tab.json").Tabs[0].Body {
		if b.Table != nil {
			tbl = b.Table
		}
	}
	if len(tbl) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(tbl))
	}
	want := [][]string{{"Control\n", "Owner\n"}, {"Register review\n", "Operations\n"}}
	for r, row := range tbl {
		if len(row) != 2 {
			t.Fatalf("row %d has %d cells, want 2", r, len(row))
		}
		for c, cell := range row {
			ps := paragraphs(cell.Blocks)
			if len(ps) != 1 {
				t.Fatalf("cell %d,%d has %d paragraphs, want 1", r, c, len(ps))
			}
			if got := runText(ps[0]); got != want[r][c] {
				t.Errorf("cell %d,%d = %q, want %q", r, c, got, want[r][c])
			}
		}
	}
}

func TestRunsCarryTheirSuggestionIDs(t *testing.T) {
	ps := paragraphs(fixture(t, "single-tab.json").Tabs[0].Body)
	runs := ps[1].Runs
	if len(runs) != 7 {
		t.Fatalf("len(runs) = %d, want 7", len(runs))
	}
	if got := runs[1].DeletionIDs; len(got) != 1 || got[0] != "suggest.a1" {
		t.Errorf("deletion run ids = %v", got)
	}
	if len(runs[1].InsertionIDs) != 0 {
		t.Errorf("deletion run carries insertion ids %v", runs[1].InsertionIDs)
	}
	// The insertion is two runs because the second half is bold. Both carry the
	// one id, and joining them is the reader's job, not the walk's.
	for _, i := range []int{2, 3} {
		if got := runs[i].InsertionIDs; len(got) != 1 || got[0] != "suggest.a1" {
			t.Errorf("run %d insertion ids = %v", i, got)
		}
	}
	if len(runs[0].InsertionIDs) != 0 || len(runs[0].DeletionIDs) != 0 {
		t.Error("a plain run carries suggestion ids")
	}
	if runs[0].Kind != KindText {
		t.Errorf("plain run kind = %q", runs[0].Kind)
	}
	if runs[1].StartIndex != 41 || runs[1].EndIndex != 49 {
		t.Errorf("deletion run indexes = %d..%d, want 41..49", runs[1].StartIndex, runs[1].EndIndex)
	}
}

func TestObjectRunsCarryTheirKindsAndTheFootnoteText(t *testing.T) {
	d := fixture(t, "objects.json")
	runs := paragraphs(d.Tabs[0].Body)[0].Runs
	// KindObject is in this list on purpose. An embedded object carrying
	// neither imageProperties nor embeddedDrawingProperties is a thing gdoc
	// cannot name, and calling it an image would be the guess this package
	// refuses to make. Without a case here that guess passes every test.
	want := []string{KindText, KindImage, KindText, KindDrawing, KindObject, KindEquation, KindFootnoteRef, KindText}
	if len(runs) != len(want) {
		t.Fatalf("len(runs) = %d, want %d", len(runs), len(want))
	}
	for i, k := range want {
		if runs[i].Kind != k {
			t.Errorf("run %d kind = %q, want %q", i, runs[i].Kind, k)
		}
	}
	if runs[6].FootnoteID != "kix.fn1" {
		t.Errorf("footnote reference id = %q", runs[6].FootnoteID)
	}
	if got := d.Footnotes["kix.fn1"]; got != "Measured on 2026-09-06." {
		t.Errorf("footnote text = %q", got)
	}
	if len(d.Footnotes) != 1 {
		t.Errorf("len(Footnotes) = %d, want 1", len(d.Footnotes))
	}
}

func TestCommentRangesPlaceOneAndListTheOther(t *testing.T) {
	d := fixture(t, "single-tab.json")
	got, ok := d.CommentRanges["AAAA1111"]
	if !ok {
		t.Fatalf("CommentRanges = %v, want the ranged comment", d.CommentRanges)
	}
	if got != (Range{Tab: "t.0", Start: 66, End: 85}) {
		t.Errorf("range = %+v", got)
	}
	if len(d.CommentRanges) != 1 {
		t.Errorf("len(CommentRanges) = %d, want 1", len(d.CommentRanges))
	}
	if len(d.Unplaced) != 1 || d.Unplaced[0] != "BBBB2222" {
		t.Errorf("Unplaced = %v, want [BBBB2222]", d.Unplaced)
	}
}

// The shape Google actually returns, measured on 2026-09-06 against a real
// document with six comments: each `comments[]` entry carries `commentId` and
// an `anchorId`, and the range lives in the tab, under
// `documentTab.commentAnchors[anchorId].ranges`. The fixture is that shape with
// placeholder text; the recording it was modelled on holds a real document and
// stays out of the tree.
func TestCommentRangesComeFromTheTabsCommentAnchors(t *testing.T) {
	d := fixture(t, "anchors.json")
	want := map[string]Range{
		"C-FIRST":  {Tab: "t.0", Start: 7, End: 15},
		"C-TWO":    {Tab: "t.0", Start: 22, End: 31}, // an anchor with several ranges places on its first
		"C-SECOND": {Tab: "t.1", Start: 13, End: 27}, // the tab is the one whose anchors named it
	}
	for id, w := range want {
		got, ok := d.CommentRanges[id]
		if !ok || got != w {
			t.Errorf("%s: range = %+v (placed %v), want %+v", id, got, ok, w)
		}
	}
	if len(d.CommentRanges) != len(want) {
		t.Errorf("len(CommentRanges) = %d, want %d: %v", len(d.CommentRanges), len(want), d.CommentRanges)
	}
	// An anchorId no tab knows is a comment the read did not place: reported,
	// never guessed.
	if len(d.Unplaced) != 1 || d.Unplaced[0] != "C-LOST" {
		t.Errorf("Unplaced = %v, want [C-LOST]", d.Unplaced)
	}
}

func TestCommentRangeShapesTheDecoderAccepts(t *testing.T) {
	// The shape of the Docs read's `comments` key is measured, not documented,
	// so the decoder tries the three places a range has been seen. Each case is
	// one of them, and the last is an entry with no range at all.
	cases := []struct {
		name  string
		entry string
		want  *Range
	}{
		{"range", `{"id":"C1","range":{"startIndex":4,"endIndex":9}}`, &Range{Tab: "t.0", Start: 4, End: 9}},
		{"anchor.range", `{"id":"C1","anchor":{"range":{"startIndex":4,"endIndex":9}}}`, &Range{Tab: "t.0", Start: 4, End: 9}},
		{"top level", `{"id":"C1","startIndex":4,"endIndex":9}`, &Range{Tab: "t.0", Start: 4, End: 9}},
		{"anchor is a string", `{"id":"C1","anchor":"kix.abc","startIndex":4,"endIndex":9}`, &Range{Tab: "t.0", Start: 4, End: 9}},
		{"no range", `{"id":"C1"}`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw := `{"documentId":"D","body":{"content":[]},"comments":[` + c.entry + `]}`
			d, err := Parse([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			got, ok := d.CommentRanges["C1"]
			if c.want == nil {
				if ok {
					t.Fatalf("placed %+v, want unplaced", got)
				}
				if len(d.Unplaced) != 1 || d.Unplaced[0] != "C1" {
					t.Fatalf("Unplaced = %v", d.Unplaced)
				}
				return
			}
			if !ok || got != *c.want {
				t.Fatalf("range = %+v (placed %v), want %+v", got, ok, *c.want)
			}
		})
	}
}

func TestACommentInAMultiTabDocumentNeedsItsTab(t *testing.T) {
	// One tab is the tab. With more than one, a range without a tabId names no
	// position gdoc can print, so it is reported unplaced rather than guessed.
	raw := `{"documentId":"D","tabs":[
		{"tabProperties":{"tabId":"t.0"},"documentTab":{"body":{"content":[]}}},
		{"tabProperties":{"tabId":"t.1"},"documentTab":{"body":{"content":[]}}}],
		"comments":[{"id":"C1","range":{"startIndex":4,"endIndex":9}},
		            {"id":"C2","range":{"startIndex":4,"endIndex":9,"tabId":"t.1"}}]}`
	d, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Unplaced) != 1 || d.Unplaced[0] != "C1" {
		t.Errorf("Unplaced = %v, want [C1]", d.Unplaced)
	}
	if got := d.CommentRanges["C2"]; got != (Range{Tab: "t.1", Start: 4, End: 9}) {
		t.Errorf("C2 range = %+v", got)
	}
}

func TestACommentWithNoIDIsReportedByPosition(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[]},"comments":[{"range":{"startIndex":1,"endIndex":2}}]}`
	d, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Unplaced) != 1 || d.Unplaced[0] != "comments[0] (no id)" {
		t.Errorf("Unplaced = %v", d.Unplaced)
	}
}
