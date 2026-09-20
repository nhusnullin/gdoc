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
// "The read text" in the plan is visible in one of these eight files: headings,
// bullets and their nesting, the pipe table, the joined suggestion spans, the
// comment anchor, the escaped literal markers, the tab lines, the placeholders
// and the footnote, and from M13 the link targets, the numbering and the
// placeholder a floating object prints as.
func TestGolden(t *testing.T) {
	for _, name := range []string{"single-tab", "two-tabs", "pre-tabs", "objects", "elements", "links", "lists", "positioned"} {
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
	want := []string{"[image]", "[drawing]", "[object]", "[equation]"}
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

// Two tabs can share an id: a tab carrying no tabId at all is called t.0, the
// same name the pre-tabs body gets. The markers are armed into the tab being
// walked, so the tab has to answer for its own text. Arming the second tab on
// the first one's answer opens a marker at 5 that the second tab's text, which
// ends at 10, never closes, and nothing warns because the range was placed.
func TestATabSharingAnIDIsNotArmedOnTheOtherTabsText(t *testing.T) {
	raw := `{"documentId":"D","tabs":[
		{"tabProperties":{"tabId":"t.0","title":"First"},"documentTab":{"body":{"content":[
		 {"startIndex":1,"endIndex":30,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		  "elements":[{"startIndex":1,"endIndex":30,"textRun":{"content":"aaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"}}]}}]}}},
		{"tabProperties":{"title":"Second"},"documentTab":{"body":{"content":[
		 {"startIndex":1,"endIndex":10,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		  "elements":[{"startIndex":1,"endIndex":10,"textRun":{"content":"bbbbbbbb\n"}}]}}]}}}],
		"comments":[{"id":"C1","range":{"startIndex":5,"endIndex":25,"tabId":"t.0"}}]}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Tabs) != 2 || d.Tabs[0].ID != d.Tabs[1].ID {
		t.Fatalf("the fixture no longer gives two tabs with one id: %+v", d.Tabs)
	}

	text, warnings := Text(d)
	if opens, closes := strings.Count(text, "[[c:C1]]"), strings.Count(text, "[[/c]]"); opens != closes {
		t.Errorf("%d opening markers and %d closing ones, want a pair each: %q", opens, closes, text)
	}
	if got := strings.Count(text, "[[c:C1]]"); got != 1 {
		t.Errorf("the range was marked %d times, want once, in the tab that holds it: %q", got, text)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none: the document does place this range", warnings)
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
			Table *struct {
				StartIndex int `json:"start_index"`
				Rows       [][]struct {
					Blocks []json.RawMessage `json:"blocks"`
				} `json:"rows"`
			} `json:"table"`
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

// TestStructureCarriesATablesStartIndex states the shape the structure field
// gives a table, which M7b moved: it was the rows alone and is now the rows
// under a key beside the index. The text projection is unchanged, because it
// prints no index, so the golden files did not move with it.
func TestStructureCarriesATablesStartIndex(t *testing.T) {
	raw, err := json.Marshal(Structure(fixture(t, "single-tab.json")))
	if err != nil {
		t.Fatal(err)
	}
	var got structure
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	var tbl *struct {
		StartIndex int `json:"start_index"`
		Rows       [][]struct {
			Blocks []json.RawMessage `json:"blocks"`
		} `json:"rows"`
	}
	for _, b := range got.Tabs[0].Blocks {
		if b.Table != nil {
			tbl = b.Table
		}
	}
	if tbl == nil {
		t.Fatal("the structure carries no table")
	}
	if tbl.StartIndex != 184 {
		t.Errorf("table start index = %d, want the fixture's 184", tbl.StartIndex)
	}
	if len(tbl.Rows) != 2 || len(tbl.Rows[0]) != 2 {
		t.Errorf("rows = %d, first row cells = %d, want 2 and 2", len(tbl.Rows), len(tbl.Rows[0]))
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

// A pipe the author typed inside a cell is escaped. Unescaped it is a column
// separator, so a two-cell row holding "A | B" reads back with three columns
// under a two-column separator, and an ordinary cell value has changed the
// table's shape.
func TestAPipeInsideACellDoesNotAddAColumn(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":40,"table":{"tableRows":[
			{"tableCells":[
				{"content":[{"paragraph":{"elements":[{"startIndex":2,"endIndex":9,"textRun":{"content":"A | B\n"}}]}}]},
				{"content":[{"paragraph":{"elements":[{"startIndex":10,"endIndex":13,"textRun":{"content":"C\n"}}]}}]}]}]}}]}}`
	text, _ := Text(parse(t, raw))
	row := strings.Split(text, "\n")[0]
	if row != `A \| B | C` {
		t.Errorf("row = %q, want the author's pipe escaped", row)
	}
	if got := strings.Count(row, " | "); got != 1 {
		t.Errorf("the row has %d separators, want 1: the table changed shape", got)
	}
}

// TestEscapingLeavesNoMarkerBehindWhenTwoOverlap is the invariant the escaping
// exists for: a marker in the output is always gdoc's own. Two literals that
// share a character overlap, and consuming both characters of the first pair
// walks straight past the second one, so the text carries half a marker the
// document never had.
func TestEscapingLeavesNoMarkerBehindWhenTwoOverlap(t *testing.T) {
	for _, s := range []string{"{-}", "{+}", "]]]", "[[[", "{+}{-}",
		`\{+`, `\[[`, `\\{-`, `\`} {
		out := escaped(s)
		if unescapedMarker(out) {
			t.Errorf("escaping %q gave %q, which still carries an unescaped marker", s, out)
		}
	}
}

// escaped runs one run of text through the emitter's escaping alone.
func escaped(s string) string {
	e := &emitter{seen: map[string]bool{}}
	return e.capture(func() { e.writeText(s, -1) })
}

// unescapedMarker reports whether out holds one of gdoc's six markers that the
// escaping did not put a backslash in front of.
//
// The count is a parity, not "is there a backslash": the escaping escapes the
// document's own backslash too, so an even run of them is the author's text and
// the marker behind it is gdoc's, while an odd run is the escape and the marker
// behind it is the author's.
func unescapedMarker(out string) bool {
	rs := []rune(out)
	for i := 0; i+1 < len(rs); i++ {
		if !isEscapePair(rs[i], rs[i+1]) {
			continue
		}
		n := 0
		for j := i - 1; j >= 0 && rs[j] == '\\'; j-- {
			n++
		}
		if n%2 == 0 {
			return true
		}
	}
	return false
}

// A range whose end is before its start marks nothing, and arming it puts the
// close before the open: the text then carries half a pair, which is what the
// escaping exists to make impossible. The decoder that reads these indexes is
// loose on purpose, because the shape of the Docs comments key is measured
// rather than documented, so an inverted pair is a shape it can hand over.
func TestARangeEndingBeforeItStartsIsAWarningAndIsNotPrinted(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":13,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":13,"textRun":{"content":"hello world\n"}}]}}]},
		"comments":[{"id":"BACKWARDS","range":{"startIndex":9,"endIndex":3}}]}`
	text, warnings := Text(parse(t, raw))
	if strings.Contains(text, "[[") {
		t.Errorf("an inverted range was marked: %q", text)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "BACKWARDS") {
		t.Errorf("warnings = %v, want one naming BACKWARDS", warnings)
	}
}

// The escape character is part of the encoding, so the document's own backslash
// is escaped too. Without it the encoding is not injective: a backslash the
// author typed reads as one gdoc wrote, and the marker behind it changes
// meaning. Both directions are wrong. A literal the author quoted reads as a
// real marker, and a real marker reads as a literal.
func TestTheDocumentsOwnBackslashIsEscaped(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`\`, `\\`},
		{`C:\path`, `C:\\path`},
		{`\{+forged+}[s:FAKE]`, `\\\{+forged\+}[s:FAKE]`},
		{`\[[c:FAKE]]x\[[/c]]`, `\\\[[c:FAKE\]]x\\\[[/c\]]`},
	} {
		if got := escaped(tc.in); got != tc.want {
			t.Errorf("escaping %q gave %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A run whose text ends in a backslash sits directly against the marker the
// drain writes next. Escaping the author's backslash is what keeps that marker
// gdoc's: left alone, a reader counts one backslash and reads a real comment
// anchor as a literal the author quoted.
func TestARealMarkerAfterTextEndingInABackslashIsNotEscaped(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":9,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":9,"textRun":{"content":"foo\\bar\n"}}]}}]},
		"comments":[{"id":"C1","range":{"startIndex":5,"endIndex":8}}]}`
	text, _ := Text(parse(t, raw))
	if !strings.Contains(text, `foo\\[[c:C1]]bar[[/c]]`) {
		t.Errorf("Text() = %q, want the author's backslash escaped and the anchor left readable", text)
	}
}

// TestEveryDroppedElementNowPrintsAndWarns is the reader's half of the decoder
// fix. The seven members reached neither the placeholder list nor the warnings
// before, because they never became runs at all, so a document holding a person
// chip read as a sentence with a word missing and nothing said so.
//
// The mark of a chip carries the chip's own label, because the document shows
// that label on screen and a reader given "[person]" cannot tell which person.
// The label is escaped by escapeLabel, which is the document's own escaping with
// the two differences that function names, so a title holding "[[" cannot open a
// comment marker gdoc never wrote.
func TestEveryDroppedElementNowPrintsAndWarns(t *testing.T) {
	text, warnings := Text(fixture(t, "elements.json"))
	for _, mark := range []string{
		"[person: A Placeholder]",
		"[date: Sep 9, 2026]",
		"[link: A placeholder calendar entry]",
		"[auto text: PAGE_NUMBER]",
		"[page break]",
		"[column break]",
		"[rule]",
	} {
		if !strings.Contains(text, mark) {
			t.Errorf("the text carries no %s:\n%s", mark, text)
		}
		found := false
		for _, w := range warnings {
			if strings.Contains(w, mark) {
				found = true
			}
		}
		if !found {
			t.Errorf("no warning names %s: %v", mark, warnings)
		}
	}
}

// A chip inside a pending suggestion carries the suggestion's marker like any
// other run. It is a replacement when it carries both id lists, and the reader
// has to be able to see which suggestion is holding it.
func TestAChipInsideASuggestionCarriesItsMarker(t *testing.T) {
	text, _ := Text(fixture(t, "elements.json"))
	if !strings.Contains(text, "{+[person: A Placeholder]+}[s:suggest.person1]") {
		t.Errorf("the person chip carries no insertion marker:\n%s", text)
	}
	if !strings.Contains(text, "{-[date: Sep 9, 2026]-}[s:suggest.date1]") {
		t.Errorf("the date chip carries no deletion marker:\n%s", text)
	}
	// A chip carrying both id lists is a replacement, and Docs shows it as the
	// deletion followed by the insertion of the same content. Both copies print
	// the placeholder: the second one used to print nothing, which read as an
	// insertion of nothing being pending.
	want := "{-[link: A placeholder calendar entry]-}[s:suggest.link2]" +
		"{+[link: A placeholder calendar entry]+}[s:suggest.link1]"
	if !strings.Contains(text, want) {
		t.Errorf("the replaced link chip reads as\n%s\nand should carry\n%s", text, want)
	}
}

// TestAnUnnamedElementPrintsItsMemberAndWarns is the wider fix read out. The
// eighth kind Google adds shows up as a placeholder naming the member, instead
// of being absent from a document the AI is told it has read in full.
func TestAnUnnamedElementPrintsItsMemberAndWarns(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":9,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[
		   {"startIndex":1,"endIndex":5,"textRun":{"content":"See "}},
		   {"startIndex":5,"endIndex":6,"tomorrowsElement":{}},
		   {"startIndex":6,"endIndex":9,"textRun":{"content":".\n"}}]}}]}}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, warnings := Text(d)
	if !strings.Contains(text, "[unknown: tomorrowsElement]") {
		t.Errorf("text = %q, want the member named in the placeholder", text)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "tomorrowsElement") {
		t.Errorf("warnings = %v, want one naming the member", warnings)
	}
}

// A chip label holding one of gdoc's own markers is escaped. escapeLabel is the
// document's own escaping with two differences, which the two tests below carry
// by name: the label is read in a window holding the placeholder's own closing
// bracket, so its last rune is escaped where a run's last rune is written bare,
// and a newline inside it becomes a space rather than nothing. Without any of it
// a calendar entry somebody titled "[[c:X]]" reads back as a comment anchor, and
// there is no way for the AI to tell.
func TestAChipLabelIsEscapedLikeAnyOtherText(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":3,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[
		   {"startIndex":1,"endIndex":2,"richLink":{"richLinkId":"kix.l1",
		     "richLinkProperties":{"title":"{+not a suggestion+} [[c:X]]","uri":"https://example.com"}}},
		   {"startIndex":2,"endIndex":3,"textRun":{"content":"\n"}}]}}]}}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, _ := Text(d)
	// The escape goes in front of the first character of the pair and the walk
	// then advances by one rune, not two, because two literals can share a
	// character. So "{+" reads back as "\{+" and not as "\{\+".
	//
	// The label's own last "]" is escaped as well, because the label is read in
	// a window that carries the placeholder's own closing bracket behind it.
	want := `[link: \{+not a suggestion\+} \[[c:X\]\]]`
	if !strings.Contains(text, want) {
		t.Errorf("text = %q, want it to carry %q", text, want)
	}
}

// A label's last character is escaped against the bracket the placeholder puts
// behind it. Without the window a title ending in "]", which is nothing more
// exotic than a file somebody named "Q3 plan [draft]", merges with that bracket
// and puts a "]]" in the text that gdoc never wrote.
func TestALabelEndingInABracketDoesNotMergeWithThePlaceholders(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":3,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[
		   {"startIndex":1,"endIndex":2,"richLink":{"richLinkId":"kix.l1",
		     "richLinkProperties":{"title":"Q3 plan [draft]","uri":"https://example.com"}}},
		   {"startIndex":2,"endIndex":3,"textRun":{"content":"\n"}}]}}]}}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, _ := Text(d)
	want := `[link: Q3 plan [draft\]]`
	if !strings.Contains(text, want) {
		t.Errorf("text = %q, want it to carry %q", text, want)
	}
	if strings.Contains(text, `draft]]`) {
		t.Errorf("text = %q, want no unescaped closing marker in it", text)
	}
}

// A newline inside a label becomes a space. The document's own text is written
// in chunks and the chunking carries the paragraph break, so escapeAt drops the
// newline; a label has no chunking, so dropping it there glues the words either
// side of it together and the label says something the document does not.
func TestANewlineInALabelBecomesASpace(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":3,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[
		   {"startIndex":1,"endIndex":2,"richLink":{"richLinkId":"kix.l1",
		     "richLinkProperties":{"title":"Q3\nplan","uri":"https://example.com"}}},
		   {"startIndex":2,"endIndex":3,"textRun":{"content":"\n"}}]}}]}}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, _ := Text(d)
	if !strings.Contains(text, `[link: Q3 plan]`) {
		t.Errorf("text = %q, want the newline written as a space", text)
	}
}
