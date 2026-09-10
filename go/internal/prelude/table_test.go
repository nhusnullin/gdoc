package prelude

import (
	"strings"
	"testing"

	"gdoc/internal/cover"
	"gdoc/internal/house"
)

// The house values below are written out as numbers and as words on purpose.
// A test that reads cfg.Tables["version_control"].Rows[0] is a mirror: move the
// value and the assertion follows it and still passes. These say what the
// Altery front matter is.

// tableStarts is the start index of every table the requests insert, in order.
// insertTable puts a newline in front of the table it makes, so the table
// itself begins one past the index the request names.
func tableStarts(requests []map[string]any) []int {
	var out []int
	for _, r := range requests {
		body, ok := r["insertTable"].(map[string]any)
		if !ok {
			continue
		}
		out = append(out, body["location"].(map[string]any)["index"].(int)+1)
	}
	return out
}

// tableSize is the rows and columns of the nth table the requests insert.
func tableSize(requests []map[string]any, n int) (int, int) {
	seen := 0
	for _, r := range requests {
		body, ok := r["insertTable"].(map[string]any)
		if !ok {
			continue
		}
		if seen == n {
			return body["rows"].(int), body["columns"].(int)
		}
		seen++
	}
	return 0, 0
}

// cellText is the words written into every cell, keyed by the table's start
// index, the row and the column. Every cell states its own look, so the text
// that went in since the last cell was styled is that cell's.
func cellText(requests []map[string]any) map[[3]int]string {
	out := map[[3]int]string{}
	var pending []string
	for _, r := range requests {
		if body, ok := r["insertText"].(map[string]any); ok {
			pending = append(pending, body["text"].(string))
			continue
		}
		if _, ok := r["insertTable"].(map[string]any); ok {
			pending = nil // whatever came before the table is not in it
			continue
		}
		body, ok := r["updateTableCellStyle"].(map[string]any)
		if !ok {
			continue
		}
		out[cellKey(body)] = strings.Join(pending, "")
		pending = nil
	}
	return out
}

// cellStyleAt is the style one cell carries.
func cellStyleAt(requests []map[string]any, start, row, col int) map[string]any {
	for _, r := range requests {
		body, ok := r["updateTableCellStyle"].(map[string]any)
		if !ok {
			continue
		}
		if cellKey(body) == [3]int{start, row, col} {
			return body["tableCellStyle"].(map[string]any)
		}
	}
	return nil
}

func cellKey(body map[string]any) [3]int {
	at := body["tableRange"].(map[string]any)["tableCellLocation"].(map[string]any)
	return [3]int{
		at["tableStartLocation"].(map[string]any)["index"].(int),
		at["rowIndex"].(int),
		at["columnIndex"].(int),
	}
}

func TestTheVersionControlTableIsTheHouseTableCellByCell(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: five rows of a label and a value, and the value is the fields'
	// own where they filled the placeholder in.
	rows, cols := tableSize(got.Requests, 0)
	if rows != 5 || cols != 2 {
		t.Fatalf("the version control table is %dx%d, want 5x2", rows, cols)
	}
	start := tableStarts(got.Requests)[0]
	text := cellText(got.Requests)
	want := map[[3]int]string{
		{start, 0, 0}: "Document Owner ",
		{start, 0, 1}: "The Board",
		{start, 1, 0}: "Date of Last Approval",
		{start, 1, 1}: "",
		{start, 2, 0}: "Review Frequency",
		{start, 2, 1}: "Annually ",
		{start, 3, 0}: "Board Ratification Date",
		{start, 3, 1}: "",
		{start, 4, 0}: "Policy Distribution",
		{start, 4, 1}: "",
	}
	for key, w := range want {
		if got := text[key]; got != w {
			t.Errorf("cell %v reads %q, want %q", key, got, w)
		}
	}
}

func TestTheRevisionHistoryRepeatsOneRowPerRevision(t *testing.T) {
	// Arrange
	f := testFields()
	f.Revisions = []cover.Revision{
		{Version: "1.0", Date: "May 2026", Author: "N Khusnullin",
			ApprovedBy: "The Board", ApprovalDate: "May 2026",
			Section: "All", Change: "New document"},
		{Version: "1.1", Date: "June 2026", Author: "N Khusnullin",
			ApprovedBy: "The Board", ApprovalDate: "June 2026",
			Section: "Section 4", Change: "Reviewed"},
	}
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, f, 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: the header, then one row per revision, and the blank row a
	// person would fill in by hand is left out once the fields declare them.
	rows, cols := tableSize(got.Requests, 1)
	if rows != 3 || cols != 7 {
		t.Fatalf("the revision history is %dx%d, want 3x7", rows, cols)
	}
	start := tableStarts(got.Requests)[1]
	text := cellText(got.Requests)
	if got := text[[3]int{start, 0, 0}]; got != "Version No" {
		t.Errorf("the first column is headed %q, want Version No", got)
	}
	if got := text[[3]int{start, 1, 6}]; got != "New document" {
		t.Errorf("the first revision's change reads %q, want New document", got)
	}
	if got := text[[3]int{start, 2, 0}]; got != "1.1" {
		t.Errorf("the second revision's version reads %q, want 1.1", got)
	}
	for key, words := range text {
		if key[0] == start && words == "xx" {
			t.Errorf("cell %v still carries the template's xx", key)
		}
	}
}

func TestTheRevisionHistoryKeepsTheTemplatesRowsWhenTheFieldsDeclareNone(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: the header, the template's own prototype row, and the blank row
	// behind it. That is v1's early return: a note declaring no revisions
	// keeps the rows a person fills in by hand.
	rows, _ := tableSize(got.Requests, 1)
	if rows != 3 {
		t.Fatalf("the revision history has %d rows, want 3", rows)
	}
	start := tableStarts(got.Requests)[1]
	text := cellText(got.Requests)
	if got := text[[3]int{start, 1, 0}]; got != "xx" {
		t.Errorf("the prototype row reads %q, want the template's xx", got)
	}
	if got := text[[3]int{start, 1, 5}]; got != "New document " {
		t.Errorf("the prototype's section reads %q, want New document ", got)
	}
	if got := text[[3]int{start, 2, 3}]; got != "" {
		t.Errorf("the blank row is not blank: %q", got)
	}
}

func TestARevisionsCellCarriesTheHeadingsTwoParagraphs(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: the last column's heading is two paragraphs in one cell, and a
	// cell that already holds one paragraph mark takes one newline to become
	// two.
	start := tableStarts(got.Requests)[1]
	if got := cellText(got.Requests)[[3]int{start, 0, 6}]; got != "Revisions/\nChanges" {
		t.Errorf("the last heading reads %q, want two paragraphs", got)
	}
}

func TestTheClassificationTableShadesOnlyTheDeclaredClass(t *testing.T) {
	// Arrange: the fields declare Internal.
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: the Internal description is shaded #FFF2CC and the other three
	// descriptions carry no fill at all. The master was captured with Internal
	// marked, so writing its fills verbatim marks every document Internal
	// whatever the fields say.
	start := tableStarts(got.Requests)[2]
	internal := cellStyleAt(got.Requests, start, 3, 1)
	if internal == nil {
		t.Fatalf("the Internal description states no cell style")
	}
	fill := internal["backgroundColor"].(map[string]any)
	if channel(fill, "red") != 1 || channel(fill, "green") != 242.0/255 || channel(fill, "blue") != 204.0/255 {
		t.Errorf("the Internal description is shaded %v, want #FFF2CC", fill)
	}
	confidential := cellStyleAt(got.Requests, start, 1, 1)
	if confidential == nil {
		t.Fatalf("the Confidential description states no cell style")
	}
	if _, shaded := confidential["backgroundColor"].(map[string]any)["color"]; shaded {
		t.Errorf("a class the fields do not declare is shaded: %v", confidential["backgroundColor"])
	}
	// The class name beside it keeps its own fill, because it describes no
	// class: it is the row's label.
	name := cellStyleAt(got.Requests, start, 1, 0)
	if _, shaded := name["backgroundColor"].(map[string]any)["color"]; !shaded {
		t.Errorf("the Confidential label lost its fill: %v", name["backgroundColor"])
	}
}

func TestACellStatesItsPaddingAndItsBordersFromTheHouseFile(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: the version control table pads every cell by 5pt and draws a
	// 1pt black line on every edge.
	start := tableStarts(got.Requests)[0]
	style := cellStyleAt(got.Requests, start, 0, 0)
	if style == nil {
		t.Fatalf("the first cell states no style")
	}
	for _, side := range []string{"paddingTop", "paddingLeft", "paddingBottom", "paddingRight"} {
		if magnitude(style[side]) != 5 {
			t.Errorf("%s is %v, want 5pt", side, magnitude(style[side]))
		}
	}
	border := style["borderTop"].(map[string]any)
	if magnitude(border["width"]) != 1 {
		t.Errorf("the border is %v pt, want 1", magnitude(border["width"]))
	}
	if border["dashStyle"] != "SOLID" {
		t.Errorf("the border is %v, want SOLID", border["dashStyle"])
	}
	if style["contentAlignment"] != "TOP" {
		t.Errorf("the cell aligns %v, want TOP", style["contentAlignment"])
	}
}

func TestATablesIndexesFollowTheDocsAccounting(t *testing.T) {
	// Arrange: one 2x2 table of one character per cell.
	cfg := tinyTableConfig()

	// Act
	got, err := FrontMatter(cfg, cover.Fields{Title: "A Policy"}, 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: an empty 2x2 table is twelve index units, which is the table
	// itself, one for each row, one for each cell and its own paragraph mark,
	// and one for the table's own end. A thirteenth unit goes on the newline
	// insertTable writes in front of it. Measured by TestLiveTableIndexProbe
	// on 2026-09-10, against a table asked for at index 1: the table spans
	// [2,14) and the paragraph behind it begins at 14.
	if starts := tableStarts(got.Requests); len(starts) != 1 || starts[0] != 2 {
		t.Fatalf("the table starts at %v, want 2, one past the newline in front of it", starts)
	}
	want := []int{5, 8, 12, 15}
	var at []int
	for _, r := range got.Requests {
		if body, ok := r["insertText"].(map[string]any); ok {
			at = append(at, body["location"].(map[string]any)["index"].(int))
		}
	}
	if len(at) != len(want) {
		t.Fatalf("the table writes %d cells, want %d", len(at), len(want))
	}
	for i := range want {
		if at[i] != want[i] {
			t.Errorf("cell %d is written at %d, want %d", i, at[i], want[i])
		}
	}
	if got.End != 18 {
		t.Errorf("End = %d, want 18: thirteen units for the empty table and the newline in front of it, and four characters", got.End)
	}
	if got.Tables != 1 || got.Cells != 4 {
		t.Errorf("Tables = %d and Cells = %d, want 1 and 4", got.Tables, got.Cells)
	}
}

func TestTheLegendIsABoldWordATabAndTheSentence(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: the four lines of the key under the revision history.
	lines := texts(got.Requests)
	want := []string{
		"New         \t  New information has been added",
		"Update     \t  Existing information has been updated",
		"Amend     \t  Existing information has been amended",
		"Remove    \t  Existing information has been removed",
	}
	for _, w := range want {
		if !contains(lines, w) {
			t.Errorf("the legend does not carry %q", w)
		}
	}
	// The word is bold and the sentence is not, both at 12pt in the house's
	// own dark blue.
	var at int
	for _, span := range paragraphSpans(got.Requests) {
		if at == 0 {
			at = span[0]
		}
	}
	word := firstRunOf(got.Requests, "New         \t")
	if word == nil {
		t.Fatalf("the legend's first word states no text style")
	}
	if word["bold"] != true {
		t.Errorf("the legend's word is not bold")
	}
	if magnitude(word["fontSize"]) != 12 {
		t.Errorf("the legend is %v pt, want 12", magnitude(word["fontSize"]))
	}
	fg := word["foregroundColor"].(map[string]any)
	if channel(fg, "red") != 15.0/255 || channel(fg, "green") != 19.0/255 || channel(fg, "blue") != 64.0/255 {
		t.Errorf("the legend is written in %v, want #0F1340", fg)
	}
}

// firstRunOf is the text style over the run holding exactly the given words,
// found by the span the insert that wrote them left behind.
func firstRunOf(requests []map[string]any, words string) map[string]any {
	at := -1
	for _, r := range requests {
		body, ok := r["insertText"].(map[string]any)
		if !ok {
			continue
		}
		if strings.HasPrefix(body["text"].(string), words) {
			at = body["location"].(map[string]any)["index"].(int)
			break
		}
	}
	if at < 0 {
		return nil
	}
	body := styleAt(requests, "updateTextStyle", at, at+len([]rune(words)))
	if body == nil {
		return nil
	}
	return body["textStyle"].(map[string]any)
}

func TestTheContentsListIsAManualStep(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: the Docs API has no request that makes a contents list, so the
	// prelude names it rather than leaving a heading over nothing.
	var found bool
	for _, step := range got.Manual {
		if strings.Contains(step.What, "contents list") {
			found = true
			if step.Where == "" {
				t.Errorf("the contents step names no menu path")
			}
		}
	}
	if !found {
		t.Errorf("the contents list is not in the manual steps: %+v", got.Manual)
	}
	// The heading over it is still written, because a person inserting the
	// list needs somewhere to put it.
	if !contains(texts(got.Requests), "Contents") {
		t.Errorf("the Contents heading is not in the front matter")
	}
	// And the column widths and row heights, which no request kind here sets.
	var widths bool
	for _, step := range got.Manual {
		if strings.Contains(step.What, "column widths") {
			widths = true
		}
	}
	if !widths {
		t.Errorf("the column widths are not in the manual steps: %+v", got.Manual)
	}
}

func TestEveryFrontMatterMaskNamesExactlyWhatTheRequestSets(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: a field named in a mask and left unset is a property reset to
	// its default, so a mask naming more than the request sets destroys
	// formatting and one naming less is a value the server ignores.
	kinds := map[string]string{
		"updateParagraphStyle": "paragraphStyle",
		"updateTextStyle":      "textStyle",
		"updateTableCellStyle": "tableCellStyle",
	}
	seen := map[string]int{}
	for _, r := range got.Requests {
		for kind, key := range kinds {
			body, ok := r[kind].(map[string]any)
			if !ok {
				continue
			}
			seen[kind]++
			set := body[key].(map[string]any)
			mask := strings.Split(body["fields"].(string), ",")
			if len(mask) != len(set) {
				t.Errorf("%s names %d fields and sets %d: %v against %v", kind, len(mask), len(set), mask, set)
			}
			for _, name := range mask {
				if _, ok := set[name]; !ok {
					t.Errorf("%s names %q in its mask and does not set it", kind, name)
				}
			}
			if body["fields"] == "" || body["fields"] == "*" {
				t.Errorf("%s carries the mask %q, which Docs reads as every field", kind, body["fields"])
			}
		}
	}
	for kind := range kinds {
		if seen[kind] == 0 {
			t.Errorf("no %s request was checked", kind)
		}
	}
}

func TestARowWithMoreCellsThanColumnsStopsTheFrontMatter(t *testing.T) {
	// Arrange
	cfg := tinyTableConfig()
	table := cfg.Tables["t"]
	table.Rows[0].Cells = append(table.Rows[0].Cells, cellOf("E"))
	cfg.Tables["t"] = table

	// Act
	_, err := FrontMatter(cfg, cover.Fields{Title: "A Policy"}, 1)

	// Assert
	if err == nil {
		t.Fatalf("a row wider than its table was written")
	}
	if !strings.Contains(err.Error(), "t") {
		t.Errorf("the refusal does not name the table: %v", err)
	}
}

func TestAVerticalAlignmentTheDocsAPIDoesNotHaveStopsTheFrontMatter(t *testing.T) {
	// Arrange
	cfg := tinyTableConfig()
	table := cfg.Tables["t"]
	table.Rows[0].Cells[0].Valign = "baseline"
	cfg.Tables["t"] = table

	// Act
	_, err := FrontMatter(cfg, cover.Fields{Title: "A Policy"}, 1)

	// Assert
	if err == nil {
		t.Fatalf("a cell aligned in a way Docs has no word for was written")
	}
	if !strings.Contains(err.Error(), "baseline") {
		t.Errorf("the refusal does not name the alignment: %v", err)
	}
}

func TestABlockKindTheHouseFileDoesNotHaveStopsTheFrontMatter(t *testing.T) {
	// Arrange
	cfg := tinyTableConfig()
	cfg.FrontMatter = []house.Block{{Block: "sidebar"}}

	// Act
	_, err := FrontMatter(cfg, cover.Fields{Title: "A Policy"}, 1)

	// Assert
	if err == nil {
		t.Fatalf("a front matter block gdoc cannot write was carried")
	}
	if !strings.Contains(err.Error(), "sidebar") {
		t.Errorf("the refusal does not name the block: %v", err)
	}
}

func TestTheFrontMatterStartsWhereItWasToldAndNothingReachesBehindIt(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 42)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert
	if got.Start != 42 {
		t.Errorf("Start = %d, want 42", got.Start)
	}
	for _, r := range got.Requests {
		for _, kind := range []string{"insertText", "insertTable", "insertPageBreak"} {
			body, ok := r[kind].(map[string]any)
			if !ok {
				continue
			}
			at := body["location"].(map[string]any)["index"].(int)
			if at < 42 || at >= got.End {
				t.Errorf("%s writes at %d, outside [42,%d)", kind, at, got.End)
			}
		}
	}
	if got.Paragraphs == 0 || got.Tables != 3 {
		t.Errorf("Paragraphs = %d and Tables = %d, want some paragraphs and the three house tables", got.Paragraphs, got.Tables)
	}
}

// tinyTableConfig is one 2x2 table of one character per cell, which is the
// shape the index accounting was measured on.
func tinyTableConfig() *house.Config {
	return &house.Config{
		Defaults:    house.Defaults{Font: "Calibri", SizePt: 11, LineSpacing: 1.15},
		Styles:      map[string]house.Style{"normal": {Color: "#000000"}},
		TableText:   house.TableText{Font: "Calibri", DefaultSizePt: 12},
		FrontMatter: []house.Block{{Block: "table", Ref: "t"}},
		Tables: map[string]house.Table{"t": {
			ColumnsPt: []float64{100, 100},
			Border:    house.Border{WidthPt: 1, Color: "#000000"},
			Rows: []house.Row{
				{Cells: []house.Cell{cellOf("A"), cellOf("B")}},
				{Cells: []house.Cell{cellOf("C"), cellOf("D")}},
			},
		}},
	}
}

func cellOf(text string) house.Cell {
	return house.Cell{
		Valign:     "top",
		Paragraphs: []house.CellParagraph{{Runs: []house.Run{{Text: text}}}},
	}
}

// emptyTablesConfig is two 2x2 tables of empty cells with one blank paragraph
// between them, which is the shape the house front matter has and the shape the
// live probe measured.
func emptyTablesConfig() *house.Config {
	empty := house.Cell{Valign: "top"}
	table := house.Table{
		ColumnsPt: []float64{100, 100},
		Border:    house.Border{WidthPt: 1, Color: "#000000"},
		Rows: []house.Row{
			{Cells: []house.Cell{empty, empty}},
			{Cells: []house.Cell{empty, empty}},
		},
	}
	return &house.Config{
		Defaults:  house.Defaults{Font: "Calibri", SizePt: 11, LineSpacing: 1.15},
		Styles:    map[string]house.Style{"normal": {Color: "#000000"}},
		TableText: house.TableText{Font: "Calibri", DefaultSizePt: 12},
		FrontMatter: []house.Block{
			{Block: "table", Ref: "t"},
			{Block: "blank"},
			{Block: "table", Ref: "t"},
		},
		Tables: map[string]house.Table{"t": table},
	}
}

// insertsAt is the index every insertText and insertTable names, in the order
// the requests are sent, with the kind beside it.
func insertsAt(requests []map[string]any) [][2]any {
	var out [][2]any
	for _, r := range requests {
		for _, kind := range []string{"insertText", "insertTable"} {
			body, ok := r[kind].(map[string]any)
			if !ok {
				continue
			}
			out = append(out, [2]any{kind, body["location"].(map[string]any)["index"].(int)})
		}
	}
	return out
}

// TestTheIndexBehindATableIsTheTablesOwnEnd states what Docs really does with a
// table's indexes, measured by TestLiveTableIndexProbe on 2026-09-10 and
// written out here as numbers.
//
// A 2x2 table of empty cells, asked for at index 1:
//
//	paragraph   [1,2)    the newline insertTable writes in front of the table
//	table       [2,14)
//	  row 0     [3,8)      cell 0.0 [4,6), its paragraph [5,6)
//	                       cell 0.1 [6,8), its paragraph [7,8)
//	  row 1     [8,13)     cell 1.0 [9,11), its paragraph [10,11)
//	                       cell 1.1 [11,13), its paragraph [12,13)
//	paragraph   [14,15)  what follows the table
//
// The last cell ends at 13 and the table ends at 14, so the table takes one
// index of its own at the end that no row, no cell and no paragraph mark
// accounts for. 13 is that index, and Docs refuses an insertText there: it is
// inside no paragraph. 12 is accepted and lands inside the last cell, which is
// the answer that reads like success and is not one.
//
// This is what the live prelude was refused on, on 2026-09-10:
// requests[106].insertText, the spacer newline between two front-matter tables,
// at the index this builder computed as one past the last cell.
func TestTheIndexBehindATableIsTheTablesOwnEnd(t *testing.T) {
	// Arrange: two empty 2x2 tables with a blank paragraph between them.
	cfg := emptyTablesConfig()

	// Act
	got, err := FrontMatter(cfg, cover.Fields{Title: "A Policy"}, 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: the first table at 1, the spacer newline at 14, which is the
	// first table's own end and the start of the paragraph behind it, and the
	// second table at 15, one past the newline the spacer wrote.
	want := [][2]any{
		{"insertTable", 1},
		{"insertText", 14},
		{"insertTable", 15},
	}
	sent := insertsAt(got.Requests)
	if len(sent) != len(want) {
		t.Fatalf("the front matter sends %d inserts, want %d: %v", len(sent), len(want), sent)
	}
	for i := range want {
		if sent[i] != want[i] {
			t.Errorf("insert %d is %v at %v, want %v at %v", i, sent[i][0], sent[i][1], want[i][0], want[i][1])
		}
	}
	// And the whole thing ends at 28: the second table's own end, which is the
	// start of the paragraph behind it.
	if got.End != 28 {
		t.Errorf("End = %d, want 28: twelve units per empty table, one for the newline in front of each, and one for the spacer paragraph", got.End)
	}
}
