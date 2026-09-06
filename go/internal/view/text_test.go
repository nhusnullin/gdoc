package view

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// fixture parses one of internal/docs's testdata files. The projection is
// tested against the same fixtures the tree is, so a golden here says what the
// AI reads for a document the walk already has an assertion about.
func fixture(t *testing.T, name string) *docs.Document {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "docs", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	d, err := docs.Parse(raw)
	if err != nil {
		t.Fatalf("docs.Parse(%s): %v", name, err)
	}
	return d
}

func golden(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestGolden is the whole projection, one fixture at a time. Every rule under
// "The read text" in the plan is visible in one of these four files: headings,
// bullets and their nesting, the pipe table, the joined suggestion spans, the
// comment anchor, the escaped literal markers, the tab lines, the placeholders
// and the footnote.
func TestGolden(t *testing.T) {
	for _, name := range []string{"single-tab", "two-tabs", "pre-tabs", "objects"} {
		t.Run(name, func(t *testing.T) {
			got, _ := Text(fixture(t, name+".json"))
			if want := golden(t, name+".golden"); got != want {
				t.Errorf("Text() =\n%s\nwant\n%s", got, want)
			}
			if !strings.HasSuffix(got, "\n") || strings.HasSuffix(got, "\n\n") {
				t.Errorf("Text() does not end with exactly one newline: %q", got[max(0, len(got)-8):])
			}
		})
	}
}

func TestTheProjectionIsDeterministic(t *testing.T) {
	d := fixture(t, "single-tab.json")
	first, _ := Text(d)
	second, _ := Text(d)
	if first != second {
		t.Error("two reads of one document gave two strings")
	}
}

func TestOnlyAMultiTabDocumentCarriesTabLines(t *testing.T) {
	multi, _ := Text(fixture(t, "two-tabs.json"))
	if n := strings.Count(multi, "<!-- tab "); n != 2 {
		t.Errorf("two-tab document has %d tab lines, want 2", n)
	}
	for _, name := range []string{"single-tab.json", "pre-tabs.json"} {
		if one, _ := Text(fixture(t, name)); strings.Contains(one, "<!-- tab ") {
			t.Errorf("%s carries a tab line", name)
		}
	}
}

func TestAPlaceholderIsAWarning(t *testing.T) {
	_, warnings := Text(fixture(t, "objects.json"))
	want := []string{"[image]", "[drawing]", "[equation]"}
	for _, w := range want {
		found := false
		for _, got := range warnings {
			if strings.Contains(got, w) {
				found = true
			}
		}
		if !found {
			t.Errorf("no warning names %s: %v", w, warnings)
		}
	}
}

// A range the reader cannot place is not printed, and its id is a warning. The
// alternative is an opening marker with no close, which is worse than saying so.
func TestARangeOutsideTheTextIsAWarningAndIsNotPrinted(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":6,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":6,"textRun":{"content":"Text\n"}}]}}]},
		"comments":[{"id":"FARAWAY","range":{"startIndex":900,"endIndex":950}}]}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, warnings := Text(d)
	if strings.Contains(text, "[[c:") {
		t.Errorf("an unplaceable range was marked: %q", text)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "FARAWAY") {
		t.Errorf("warnings = %v, want one naming FARAWAY", warnings)
	}
}

func TestARangeNamingAnAbsentTabIsAWarning(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":6,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":6,"textRun":{"content":"Text\n"}}]}}]},
		"comments":[{"id":"OTHERTAB","range":{"startIndex":1,"endIndex":5,"tabId":"t.9"}}]}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, warnings := Text(d)
	if strings.Contains(text, "[[c:") {
		t.Errorf("a range naming an absent tab was marked: %q", text)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "OTHERTAB") {
		t.Errorf("warnings = %v, want one naming OTHERTAB", warnings)
	}
}

// structure is the shape the envelope carries under `structure`. Decoding into
// it is the round trip: a field renamed in docs shows up here as a zero.
type structure struct {
	Tabs []struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Blocks []struct {
			Paragraph *struct {
				Style      string `json:"style"`
				StartIndex int    `json:"start_index"`
				EndIndex   int    `json:"end_index"`
				Runs       []struct {
					Kind       string `json:"kind"`
					Text       string `json:"text"`
					StartIndex int    `json:"start_index"`
					EndIndex   int    `json:"end_index"`
				} `json:"runs"`
			} `json:"paragraph"`
		} `json:"blocks"`
	} `json:"tabs"`
}

func TestStructureRoundTripsWithItsIndexes(t *testing.T) {
	raw, err := json.Marshal(Structure(fixture(t, "single-tab.json")))
	if err != nil {
		t.Fatal(err)
	}
	var got structure
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Tabs) != 1 || got.Tabs[0].ID != "t.0" || got.Tabs[0].Title != "Policy" {
		t.Fatalf("tabs = %+v", got.Tabs)
	}
	p := got.Tabs[0].Blocks[0].Paragraph
	if p == nil {
		t.Fatal("the first block carries no paragraph")
	}
	if p.Style != "HEADING_1" || p.StartIndex != 1 || p.EndIndex != 7 {
		t.Errorf("first paragraph = %+v", *p)
	}
	if len(p.Runs) != 1 {
		t.Fatalf("len(runs) = %d, want 1", len(p.Runs))
	}
	r := p.Runs[0]
	if r.Kind != docs.KindText || r.Text != "Scope\n" || r.StartIndex != 1 || r.EndIndex != 7 {
		t.Errorf("first run = %+v", r)
	}
}

// Text suggested and then suggested away carries both id lists on one run, and
// Docs shows it as a deletion followed by an insertion. Both copies print, and
// the second one carries no comment marker: the characters exist once in the
// document, so opening a range around them twice would describe a range the
// document does not have.
func TestTextSuggestedAndThenSuggestedAwayPrintsTwiceAndIsMarkedOnce(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":10,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":10,"textRun":{"content":"draftier\n",
		   "suggestedInsertionIds":["INS1"],"suggestedDeletionIds":["DEL1"]}}]}}]},
		"comments":[{"id":"C1","range":{"startIndex":1,"endIndex":9}}]}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, _ := Text(d)

	if n := strings.Count(text, "draftier"); n != 2 {
		t.Errorf("the text prints %d times, want 2 (a deletion then an insertion): %q", n, text)
	}
	if !strings.Contains(text, "{-") || !strings.Contains(text, "[s:DEL1]") {
		t.Errorf("no deletion span: %q", text)
	}
	if !strings.Contains(text, "{+") || !strings.Contains(text, "[s:INS1]") {
		t.Errorf("no insertion span: %q", text)
	}
	if n := strings.Count(text, "[[c:C1]]"); n != 1 {
		t.Errorf("the comment opens %d times, want exactly 1: %q", n, text)
	}
	if n := strings.Count(text, "[[/c]]"); n != 1 {
		t.Errorf("the comment closes %d times, want exactly 1: %q", n, text)
	}
	// The deletion is the first copy, so the marker belongs to it.
	del := strings.Index(text, "{-")
	ins := strings.Index(text, "{+")
	if del < 0 || ins < 0 || del > ins {
		t.Fatalf("want the deletion before the insertion: %q", text)
	}
	if at := strings.Index(text, "[[c:C1]]"); at > ins {
		t.Errorf("the comment marker is on the second copy: %q", text)
	}
}

// parse is one inline Docs response as a document. The fixtures are the
// specification of the whole shape; these are one rule at a time.
func parse(t *testing.T, raw string) *docs.Document {
	t.Helper()
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// A range covering no characters marks nothing, so it is a warning rather than
// a marker. Armed, it printed the close before the open, and at the end of the
// last run it printed the close alone: either way the text carried half a pair,
// which is the one thing the escaping exists to make impossible.
func TestARangeCoveringNoTextIsAWarningAndIsNotPrinted(t *testing.T) {
	for _, at := range []int{3, 13} {
		raw := `{"documentId":"D","body":{"content":[
			{"startIndex":1,"endIndex":13,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
			 "elements":[{"startIndex":1,"endIndex":13,"textRun":{"content":"hello world\n"}}]}}]},
			"comments":[{"id":"EMPTY","range":{"startIndex":` + strconv.Itoa(at) + `,"endIndex":` + strconv.Itoa(at) + `}}]}`
		text, warnings := Text(parse(t, raw))
		if strings.Contains(text, "[[") {
			t.Errorf("range %d..%d was marked: %q", at, at, text)
		}
		if len(warnings) != 1 || !strings.Contains(warnings[0], "EMPTY") {
			t.Errorf("range %d..%d: warnings = %v, want one naming EMPTY", at, at, warnings)
		}
	}
}

// Two ranges that touch are two ranges, not one inside the other. The close of
// the first comes before the open of the second at the shared index.
func TestTwoTouchingRangesDoNotReadAsNested(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":13,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":13,"textRun":{"content":"hello world\n"}}]}}]},
		"comments":[{"id":"AA","range":{"startIndex":1,"endIndex":6}},
		            {"id":"BB","range":{"startIndex":6,"endIndex":12}}]}`
	text, warnings := Text(parse(t, raw))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	want := "[[c:AA]]hello[[/c]][[c:BB]] world[[/c]]\n"
	if text != want {
		t.Errorf("Text() = %q, want %q", text, want)
	}
}

// Two ranges opening at the same index are ordered by id, so the projection is
// deterministic whatever order Docs answered in.
func TestRangesOpeningTogetherAreOrderedByID(t *testing.T) {
	body := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":13,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":13,"textRun":{"content":"hello world\n"}}]}}]},`
	first := parse(t, body+`"comments":[{"id":"AA","range":{"startIndex":1,"endIndex":6}},
	                          {"id":"BB","range":{"startIndex":1,"endIndex":12}}]}`)
	second := parse(t, body+`"comments":[{"id":"BB","range":{"startIndex":1,"endIndex":12}},
	                           {"id":"AA","range":{"startIndex":1,"endIndex":6}}]}`)
	a, _ := Text(first)
	b, _ := Text(second)
	if a != b {
		t.Errorf("the answer order changed the text:\n%q\n%q", a, b)
	}
	if !strings.HasPrefix(a, "[[c:AA]][[c:BB]]") {
		t.Errorf("Text() = %q, want AA to open before BB", a)
	}
}

// The index Docs gives counts UTF-16 code units. A typographic quote is one
// unit and three bytes and an emoji is two units and four, so counting runes or
// bytes puts every marker after one of them in the wrong place.
func TestTheIndexCountsUTF16CodeUnits(t *testing.T) {
	// “ is 1 unit, 🤖 is 2. Content is `“a🤖b cd\n`: indexes 1..9, so `cd`
	// starts at 7 and the paragraph ends at 10.
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":10,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":10,"textRun":{"content":"“a🤖b cd\n"}}]}}]},
		"comments":[{"id":"C1","range":{"startIndex":7,"endIndex":9}}]}`
	text, warnings := Text(parse(t, raw))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	want := "“a\U0001f916b [[c:C1]]cd[[/c]]\n"
	if text != want {
		t.Errorf("Text() = %q, want %q", text, want)
	}
}

// Every marker the projection adds is escaped when the document's own text
// carries it. The two deletion markers had no case anywhere, and a document
// quoting `{-annually-}` would have read as a pending suggested deletion.
func TestEveryMarkerIsEscapedInTheDocumentsOwnText(t *testing.T) {
	for _, marker := range escapePairs {
		raw := `{"documentId":"D","body":{"content":[
			{"startIndex":1,"endIndex":9,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
			 "elements":[{"startIndex":1,"endIndex":9,"textRun":{"content":` +
			strconv.Quote("a "+marker+" b\n") + `}}]}}]}}`
		text, _ := Text(parse(t, raw))
		if want := "a \\" + marker + " b\n"; text != want {
			t.Errorf("Text() = %q, want %q", text, want)
		}
	}
}

// escapePairs has to hold every marker the projection writes. The list is the
// only thing standing between somebody's sentence and the AI reading it as
// gdoc's own markup, so the rule is a test rather than a review note.
func TestEscapePairsHoldsEveryMarker(t *testing.T) {
	for _, marker := range []string{openInsertion, shutInsertion, openDeletion, shutDeletion, openComment, shutComment} {
		found := false
		for _, p := range escapePairs {
			if p == marker {
				found = true
			}
		}
		if !found {
			t.Errorf("escapePairs does not hold %q", marker)
		}
	}
}

// A footnote's paragraphs are joined with newlines, and the footnote prints on
// one line. They become spaces: dropping them glued the last word of one
// paragraph to the first word of the next.
func TestAFootnoteWithTwoParagraphsIsNotGluedTogether(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":8,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":6,"textRun":{"content":"Text "}},
		             {"startIndex":6,"endIndex":7,"footnoteReference":{"footnoteId":"kix.f1","footnoteNumber":"1"}},
		             {"startIndex":7,"endIndex":8,"textRun":{"content":"\n"}}]}}]},
		"footnotes":{"kix.f1":{"footnoteId":"kix.f1","content":[
			{"paragraph":{"elements":[{"textRun":{"content":"first\n"}}]}},
			{"paragraph":{"elements":[{"textRun":{"content":"second\n"}}]}}]}}}`
	text, _ := Text(parse(t, raw))
	if !strings.Contains(text, "[^1]: first second") {
		t.Errorf("Text() =\n%s\nwant the two paragraphs separated", text)
	}
}

// A footnote referenced twice is one entry under the body, and both references
// print the same number. Counting again numbered the second reference as a new
// note while recording nothing, so the body and the reference disagreed.
func TestASecondReferenceToOneFootnotePrintsTheSameNumber(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":9,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":5,"textRun":{"content":"one "}},
		             {"startIndex":5,"endIndex":6,"footnoteReference":{"footnoteId":"kix.f1"}},
		             {"startIndex":6,"endIndex":8,"textRun":{"content":" x"}},
		             {"startIndex":8,"endIndex":9,"footnoteReference":{"footnoteId":"kix.f1"}}]}}]},
		"footnotes":{"kix.f1":{"footnoteId":"kix.f1","content":[
			{"paragraph":{"elements":[{"textRun":{"content":"the note\n"}}]}}]}}}`
	text, _ := Text(parse(t, raw))
	if n := strings.Count(text, "[^1]"); n != 3 {
		t.Errorf("[^1] appears %d times, want 3 (two references and the body): %q", n, text)
	}
	if strings.Contains(text, "[^2]") {
		t.Errorf("the second reference to one footnote is numbered 2: %q", text)
	}
}

// A comment anchored to text inside a table cell is marked there. The reachable
// positions are collected by walking into the tables, and a walk that stopped
// at the table would drop the range with a warning instead.
func TestARangeInsideATableCellIsMarked(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":20,"table":{"tableRows":[{"tableCells":[
			{"content":[{"paragraph":{"elements":[{"startIndex":4,"endIndex":10,"textRun":{"content":"cell\n"}}]}}]}]}]}}]},
		"comments":[{"id":"INCELL","range":{"startIndex":4,"endIndex":8}}]}`
	text, warnings := Text(parse(t, raw))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if !strings.Contains(text, "[[c:INCELL]]cell[[/c]]") {
		t.Errorf("Text() = %q, want the cell text marked", text)
	}
}
