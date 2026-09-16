package render

import (
	"strings"
	"testing"

	"gdoc/internal/cover"
	"github.com/beevik/etree"
)

// bodyBlocks is every block in word/document.xml, in order.
func bodyBlocks(t *testing.T, pkg *Package) []*etree.Element {
	t.Helper()
	doc := parse(t, part(t, pkg, "word/document.xml"))
	body := doc.FindElement("//w:body")
	if body == nil {
		t.Fatal("word/document.xml carries no body")
	}
	return body.ChildElements()
}

func TestTheCoverOpensWithFiveBlanksAndTheHouseName(t *testing.T) {
	pkg := build(t)
	blocks := bodyBlocks(t, pkg)
	for i := 0; i < 5; i++ {
		if blocks[i].FullTag() != "w:p" {
			t.Fatalf("block %d is %s, want an empty paragraph", i, blocks[i].FullTag())
		}
		if blocks[i].FindElement("w:r/w:t") != nil {
			t.Errorf("block %d carries text, and the cover opens with blanks", i)
		}
	}
	name := blocks[5]
	if got := name.FindElement("w:r/w:t").Text(); got != "Altery Group " {
		t.Errorf("the first cover line reads %q, want \"Altery Group \"", got)
	}
	if got := name.FindElement("w:r/w:rPr/w:sz").SelectAttrValue("w:val", ""); got != "58" {
		t.Errorf("the cover line size is %q, want 58 half-points (29pt)", got)
	}
	if name.FindElement("w:r/w:rPr/w:b") == nil {
		t.Error("the cover line is not bold")
	}
	if got := name.FindElement("w:pPr/w:jc").SelectAttrValue("w:val", ""); got != "center" {
		t.Errorf("the cover line alignment is %q, want center", got)
	}
}

func TestTheNotesTitleVersionAndDateReplaceThePlaceholders(t *testing.T) {
	pkg := build(t)
	var texts []string
	for _, e := range parse(t, part(t, pkg, "word/document.xml")).FindElements("//w:t") {
		texts = append(texts, e.Text())
	}
	for _, want := range []string{"Supplier Register Policy", "1.0", "8 September 2026"} {
		found := false
		for _, s := range texts {
			if s == want {
				found = true
			}
		}
		if !found {
			t.Errorf("the cover does not carry %q", want)
		}
	}
}

// coverLines is the text of every paragraph up to the "Version Control" label,
// which is the block the front matter opens with, so what is left is the cover
// page.
func coverLines(t *testing.T, pkg *Package) []string {
	t.Helper()
	var out []string
	for _, block := range bodyBlocks(t, pkg) {
		if block.FullTag() == "w:tbl" {
			break
		}
		if firstLabel(block) == "Version Control" {
			break
		}
		text := ""
		for _, e := range block.FindElements(".//w:t") {
			text += e.Text()
		}
		if strings.TrimSpace(text) != "" {
			out = append(out, text)
		}
	}
	return out
}

// firstLabel is one block's own text, joined.
func firstLabel(block *etree.Element) string {
	text := ""
	for _, e := range block.FindElements(".//w:t") {
		text += e.Text()
	}
	return text
}

// buildWith is Build over one note's fields.
func buildWith(t *testing.T, f cover.Fields) *Package {
	t.Helper()
	pkg, err := Build(config(t), f, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return pkg
}

// TestOneTitleLeavesNoOrAndNoSecondTitle. The master offers the title twice,
// either side of an "or", for a person filling the cover in by hand to pick
// one. A note with one title published a cover reading the title, then "or",
// then the template's own highlighted "(Name of) Framework/Policy", which is
// page one of every document. Both lines go.
func TestOneTitleLeavesNoOrAndNoSecondTitle(t *testing.T) {
	lines := coverLines(t, buildWith(t, cover.Fields{
		Title: "Supplier Register", DocType: "Policy", Version: "1.0", Date: "8 September 2026",
	}))
	want := []string{"Altery Group ", "Supplier Register Policy", "Version: 1.0", "8 September 2026"}
	if len(lines) != len(want) {
		t.Fatalf("the cover reads %q, want %q", lines, want)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("cover line %d reads %q, want %q", i, lines[i], w)
		}
	}
}

// TestAnAlternativeTitlePrintsBothLinesAndTheOr is the other direction: the
// pair is the master's own way of showing a framework and the policy under it,
// so a note that states one keeps it.
func TestAnAlternativeTitlePrintsBothLinesAndTheOr(t *testing.T) {
	lines := coverLines(t, buildWith(t, cover.Fields{
		Title: "Supplier Register", AltTitle: "Third Party Management",
		DocType: "Policy", Version: "1.0", Date: "8 September 2026",
	}))
	want := []string{"Altery Group ", "Supplier Register Policy", "or",
		"Third Party Management Policy", "Version: 1.0", "8 September 2026"}
	if len(lines) != len(want) {
		t.Fatalf("the cover reads %q, want %q", lines, want)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("cover line %d reads %q, want %q", i, lines[i], w)
		}
	}
}

// TestTheVersionLineKeepsItsLabel. The line is two runs, "Version: " and the
// number, and only the number is the note's. Replacing the whole line with the
// value took the word "Version:" off every cover gdoc built.
func TestTheVersionLineKeepsItsLabel(t *testing.T) {
	pkg := buildWith(t, cover.Fields{Title: "Supplier Register Policy", Version: "3.1",
		Date: "8 September 2026"})
	var version *etree.Element
	for _, block := range bodyBlocks(t, pkg) {
		text := ""
		for _, e := range block.FindElements(".//w:t") {
			text += e.Text()
		}
		if strings.HasPrefix(text, "Version: ") {
			version = block
			break
		}
	}
	if version == nil {
		t.Fatal("the cover carries no version line reading \"Version: \"")
	}
	runs := version.FindElements("w:r")
	if len(runs) != 2 {
		t.Fatalf("the version line has %d runs, want the label and the number", len(runs))
	}
	if got := runs[1].FindElement("w:t").Text(); got != "3.1" {
		t.Errorf("the version reads %q, want 3.1", got)
	}
	// The yellow marks "a person fills this in". Once the note's own number is
	// there the mark is misleading, so resolving a placeholder takes it off.
	if runs[1].FindElement("w:rPr/w:highlight") != nil {
		t.Error("the note's own version is still highlighted yellow")
	}
}

func TestANoteWithNoTitleKeepsTheTemplatesHighlightedPlaceholder(t *testing.T) {
	pkg, err := Build(config(t), cover.Fields{}, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	doc := parse(t, part(t, pkg, "word/document.xml"))
	found := false
	for _, r := range doc.FindElements("//w:r") {
		text := r.FindElement("w:t")
		if text == nil || text.Text() != "(Name of)" {
			continue
		}
		found = true
		if got := r.FindElement("w:rPr/w:highlight").SelectAttrValue("w:val", ""); got != "yellow" {
			t.Errorf("the placeholder highlight is %q, want yellow", got)
		}
	}
	if !found {
		t.Error("a cover with no title carries no placeholder to fill in")
	}
}

func TestTheVersionControlLabelIsNavyAndBold(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/document.xml"))
	var label *etree.Element
	for _, p := range doc.FindElements("//w:body/w:p") {
		if t2 := p.FindElement("w:r/w:t"); t2 != nil && t2.Text() == "Version Control" {
			label = p
		}
	}
	if label == nil {
		t.Fatal("word/document.xml carries no Version Control label")
	}
	rPr := label.FindElement("w:r/w:rPr")
	if rPr.SelectElement("w:b") == nil {
		t.Error("the label is not bold")
	}
	if got := rPr.SelectElement("w:color").SelectAttrValue("w:val", ""); got != "22265f" {
		t.Errorf("the label colour is %q, want 22265f", got)
	}
	if got := rPr.SelectElement("w:sz").SelectAttrValue("w:val", ""); got != "24" {
		t.Errorf("the label size is %q, want 24 half-points", got)
	}
	if got := label.FindElement("w:pPr/w:spacing").SelectAttrValue("w:before", ""); got != "360" {
		t.Errorf("the label space before is %q, want 360 twips (18pt)", got)
	}
}

func TestTheThreeFrontMatterTablesAreBuiltFromTheConfig(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/document.xml"))
	tables := doc.FindElements("//w:body/w:tbl")
	if len(tables) != 3 {
		t.Fatalf("word/document.xml carries %d tables, want 3", len(tables))
	}

	vc := tables[0]
	if got := vc.FindElement("w:tblPr/w:tblLayout").SelectAttrValue("w:type", ""); got != "fixed" {
		t.Errorf("the version control table layout is %q, want fixed", got)
	}
	if got := vc.FindElement("w:tblPr/w:tblW").SelectAttrValue("w:w", ""); got != "10185" {
		t.Errorf("the version control table width is %q, want 10185 twips", got)
	}
	if got := vc.FindElement("w:tblPr/w:tblInd").SelectAttrValue("w:w", ""); got != "100" {
		t.Errorf("the table indent is %q, want 100 twips (the first cell's left padding)", got)
	}
	cols := vc.FindElements("w:tblGrid/w:gridCol")
	if len(cols) != 2 {
		t.Fatalf("the version control table has %d columns, want 2", len(cols))
	}
	if got := cols[0].SelectAttrValue("w:w", ""); got != "3000" {
		t.Errorf("column 1 is %q twips wide, want 3000 (150pt)", got)
	}
	if got := cols[1].SelectAttrValue("w:w", ""); got != "7185" {
		t.Errorf("column 2 is %q twips wide, want 7185 (359.25pt)", got)
	}
	if got := vc.FindElement("w:tblPr/w:tblBorders/w:top").SelectAttrValue("w:sz", ""); got != "8" {
		t.Errorf("the version control border is %q eighths, want 8 (1pt)", got)
	}

	rows := vc.SelectElements("w:tr")
	if len(rows) != 5 {
		t.Fatalf("the version control table has %d rows, want 5", len(rows))
	}
	height := rows[0].FindElement("w:trPr/w:trHeight")
	if got := height.SelectAttrValue("w:val", ""); got != "570" {
		t.Errorf("row 1 is %q twips high, want 570 (28.5pt)", got)
	}
	if got := height.SelectAttrValue("w:hRule", ""); got != "atLeast" {
		t.Errorf("row 1 height rule is %q, want atLeast", got)
	}
	cell := rows[0].SelectElements("w:tc")[0]
	if got := cell.FindElement("w:tcPr/w:shd").SelectAttrValue("w:fill", ""); got != "f5d1ae" {
		t.Errorf("the first cell fill is %q, want f5d1ae", got)
	}
	if got := cell.FindElement("w:tcPr/w:tcMar/w:top").SelectAttrValue("w:w", ""); got != "100" {
		t.Errorf("the first cell top padding is %q, want 100 twips (5pt)", got)
	}
	if got := cell.FindElement("w:p/w:r/w:t").Text(); got != "Document Owner " {
		t.Errorf("the first cell reads %q, want \"Document Owner \"", got)
	}

	if got := tables[1].FindElement("w:tblPr/w:tblBorders/w:top").SelectAttrValue("w:sz", ""); got != "4" {
		t.Errorf("the revision history border is %q eighths, want 4 (0.5pt)", got)
	}
	if got := len(tables[1].FindElements("w:tblGrid/w:gridCol")); got != 7 {
		t.Errorf("the revision history table has %d columns, want 7", got)
	}
	if got := len(tables[2].SelectElements("w:tr")); got != 5 {
		t.Errorf("the document classification table has %d rows, want 5", got)
	}
}

// TestTheFrontMatterTablesPropertiesAreInSchemaOrder. CT_TblPrBase is a
// sequence, and Word reads a w:tblPr out of order the way it reads a w:pPr out
// of order: as a document to repair. tblLayout used to be written before
// tblBorders, which is positions 13 then 11.
func TestTheFrontMatterTablesPropertiesAreInSchemaOrder(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/document.xml"))
	tables := doc.FindElements("//w:body/w:tbl")
	if len(tables) != 3 {
		t.Fatalf("word/document.xml carries %d tables, want 3", len(tables))
	}
	want := []string{"w:tblStyle", "w:tblW", "w:tblInd", "w:tblBorders", "w:tblLayout"}
	for i, table := range tables {
		var got []string
		for _, e := range table.FindElement("w:tblPr").ChildElements() {
			got = append(got, e.FullTag())
		}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("table %d properties run %v, want %v", i, got, want)
		}
	}
}

func TestTheLegendIsFourLinesOfABoldWordAndASentence(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/document.xml"))
	var lines []*etree.Element
	for _, p := range doc.FindElements("//w:body/w:p") {
		if first := p.FindElement("w:r/w:t"); first != nil {
			switch first.Text() {
			case "New         ", "Update     ", "Amend     ", "Remove    ":
				lines = append(lines, p)
			}
		}
	}
	if len(lines) != 4 {
		t.Fatalf("the legend has %d lines, want 4", len(lines))
	}
	runs := lines[0].SelectElements("w:r")
	if len(runs) != 3 {
		t.Fatalf("the first legend line has %d runs, want the word, the tab and the sentence", len(runs))
	}
	if runs[0].FindElement("w:rPr/w:b") == nil {
		t.Error("the legend word is not bold")
	}
	if got := runs[0].FindElement("w:rPr/w:color").SelectAttrValue("w:val", ""); got != "0f1340" {
		t.Errorf("the legend colour is %q, want 0f1340", got)
	}
	if runs[1].SelectElement("w:tab") == nil {
		t.Error("the legend word is not followed by a tab")
	}
	if got := runs[2].FindElement("w:t").Text(); got != "  New information has been added" {
		t.Errorf("the first legend sentence reads %q", got)
	}
	if got := lines[0].FindElement("w:pPr/w:spacing").SelectAttrValue("w:line", ""); got != "240" {
		t.Errorf("the legend line spacing is %q, want 240 (single)", got)
	}
}

func TestTheContentsIsALiveWordField(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/document.xml"))
	sdt := doc.FindElement("//w:body/w:sdt")
	if sdt == nil {
		t.Fatal("word/document.xml carries no contents field")
	}
	if got := sdt.FindElement("w:sdtPr/w:docPartObj/w:docPartGallery").SelectAttrValue("w:val", ""); got != "Table of Contents" {
		t.Errorf("the contents gallery is %q, want Table of Contents", got)
	}
	instr := sdt.FindElement(".//w:instrText")
	if instr == nil {
		t.Fatal("the contents field carries no instruction")
	}
	want := ` TOC \h \u \z \t "Heading 1,1,Heading 2,2,Heading 3,3," `
	if instr.Text() != want {
		t.Errorf("the contents instruction is %q, want %q", instr.Text(), want)
	}
	var kinds []string
	for _, e := range sdt.FindElements(".//w:fldChar") {
		kinds = append(kinds, e.SelectAttrValue("w:fldCharType", ""))
	}
	if len(kinds) != 3 || kinds[0] != "begin" || kinds[1] != "separate" || kinds[2] != "end" {
		t.Errorf("the contents field characters are %v, want begin, separate, end", kinds)
	}
}

func TestTheFrontMatterIsInTheOrderTheConfigLists(t *testing.T) {
	pkg := build(t)
	var kinds []string
	for _, e := range bodyBlocks(t, pkg) {
		switch e.FullTag() {
		case "w:tbl", "w:sdt", "w:sectPr":
			kinds = append(kinds, e.FullTag())
		}
	}
	want := []string{"w:tbl", "w:tbl", "w:tbl", "w:sdt", "w:sectPr"}
	if len(kinds) != len(want) {
		t.Fatalf("the document holds %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("block %d is %s, want %s", i, kinds[i], want[i])
		}
	}
}

// tableCells is one front-matter table as rows of (text, fill) pairs.
func tableCells(t *testing.T, pkg *Package, n int) [][][2]string {
	t.Helper()
	doc := parse(t, part(t, pkg, "word/document.xml"))
	tables := doc.FindElements("//w:body/w:tbl")
	if n >= len(tables) {
		t.Fatalf("the document has %d tables, and the test wants number %d", len(tables), n)
	}
	var rows [][][2]string
	for _, tr := range tables[n].FindElements("w:tr") {
		var cells [][2]string
		for _, tc := range tr.FindElements("w:tc") {
			text := ""
			for _, e := range tc.FindElements(".//w:t") {
				text += e.Text()
			}
			fill := ""
			if shd := tc.FindElement("w:tcPr/w:shd"); shd != nil {
				fill = shd.SelectAttrValue("w:fill", "")
			}
			cells = append(cells, [2]string{text, fill})
		}
		rows = append(rows, cells)
	}
	return rows
}

// TestTheVersionControlTableCarriesTheNotesOwnWords. The five value cells are
// the five version-control keys a note states. They were parsed, validated and
// then dropped, so a note that named its owner published a table with the owner
// cell blank.
func TestTheVersionControlTableCarriesTheNotesOwnWords(t *testing.T) {
	rows := tableCells(t, buildWith(t, cover.Fields{
		Title: "Supplier Register Policy", Version: "1.0",
		Owner: "Chief Risk Officer", LastApproval: "14 August 2026",
		ReviewFrequency: "Twice a year", BoardRatification: "20 August 2026",
		Distribution: "All staff",
	}), 0)
	want := []string{"Chief Risk Officer", "14 August 2026", "Twice a year",
		"20 August 2026", "All staff"}
	if len(rows) != len(want) {
		t.Fatalf("the version control table has %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		if got := rows[i][1][0]; got != w {
			t.Errorf("version control row %d reads %q, want %q", i, got, w)
		}
	}
}

// TestARevisionRowIsWrittenPerRevision. The prototype row and the blank row
// behind it are the master's own, for a person filling the table in by hand.
func TestARevisionRowIsWrittenPerRevision(t *testing.T) {
	rows := tableCells(t, buildWith(t, cover.Fields{
		Title: "Supplier Register Policy", Version: "1.0",
		Revisions: []cover.Revision{
			{Version: "1.0", Date: "1 May 2026", Author: "N K", ApprovedBy: "Board",
				ApprovalDate: "2 May 2026", Section: "all", Change: "New document"},
			{Version: "1.1", Date: "1 June 2026", Author: "N K", ApprovedBy: "Board",
				ApprovalDate: "2 June 2026", Section: "4", Change: "Exit plan"},
		},
	}), 1)
	if len(rows) != 3 {
		t.Fatalf("the revision history has %d rows, want the header and two revisions", len(rows))
	}
	if got := rows[1][0][0]; got != "1.0" {
		t.Errorf("the first revision reads %q in column one, want 1.0", got)
	}
	if got := rows[2][6][0]; got != "Exit plan" {
		t.Errorf("the second revision reads %q in the last column, want \"Exit plan\"", got)
	}
	// The red "xx" marked a cell for a person to fill in, so it goes with the
	// value that replaced it.
	if run := rows[1]; run[0][0] == "xx" {
		t.Error("the prototype's own words survived a revision the note declared")
	}
}

// TestANoteWithNoRevisionsKeepsTheTemplatesRows is the other direction. A note
// declaring no revisions leaves the master's own rows, because a table with a
// header and nothing under it is worse than the prototype.
func TestANoteWithNoRevisionsKeepsTheTemplatesRows(t *testing.T) {
	rows := tableCells(t, buildWith(t, cover.Fields{
		Title: "Supplier Register Policy", Version: "1.0",
	}), 1)
	if len(rows) != 3 {
		t.Fatalf("the revision history has %d rows, want the master's three", len(rows))
	}
	if got := rows[1][0][0]; got != "xx" {
		t.Errorf("the prototype row reads %q in column one, want the template's \"xx\"", got)
	}
}

// TestOnlyTheDeclaredClassificationIsShaded. The master was captured with
// Internal marked, so its Internal description cell carries a fill and the
// other three do not. Writing the config's fills verbatim marked every
// document gdoc built as Internal whatever the note said, which is a mismarked
// document rather than a blank one.
func TestOnlyTheDeclaredClassificationIsShaded(t *testing.T) {
	rows := tableCells(t, buildWith(t, cover.Fields{
		Title: "Supplier Register Policy", Version: "1.0",
		Classification: "Restricted (R)",
	}), 2)
	if len(rows) != 5 {
		t.Fatalf("the classification table has %d rows, want the header and four classes", len(rows))
	}
	want := map[string]string{
		"Confidential (C)": "",
		"Restricted (R)":   "fce5cd",
		"Internal (I)":     "",
		"Public (P)":       "",
	}
	for _, row := range rows[1:] {
		fill, ok := want[row[0][0]]
		if !ok {
			t.Errorf("the classification table names a class %q nobody wrote down", row[0][0])
			continue
		}
		if got := row[1][1]; got != fill {
			t.Errorf("the %s description cell is filled %q, want %q", row[0][0], got, fill)
		}
	}
}

// TestEveryTableCellSaysWhereItsTextSits. house.yaml states valign on 41
// cells, and a config value nothing writes is the config describing an output
// nobody produced.
func TestEveryTableCellSaysWhereItsTextSits(t *testing.T) {
	doc := parse(t, part(t, build(t), "word/document.xml"))
	cells := doc.FindElements("//w:body/w:tbl/w:tr/w:tc")
	if len(cells) != 41 {
		t.Fatalf("the front matter has %d cells, want 41", len(cells))
	}
	for i, tc := range cells {
		v := tc.FindElement("w:tcPr/w:vAlign")
		if v == nil {
			t.Fatalf("cell %d states no vertical alignment", i)
		}
		if got := v.SelectAttrValue("w:val", ""); got != "top" {
			t.Errorf("cell %d is aligned %q, want top", i, got)
		}
	}
}

// TestAClassNothingDescribesIsRefused. A config and a note that spell the class
// differently would publish a document describing four classes and marking
// none, which is the failure the shading exists to stop.
func TestAClassNothingDescribesIsRefused(t *testing.T) {
	_, err := Build(config(t), cover.Fields{
		Title: "Supplier Register Policy", Version: "1.0",
		Classification: "Secret (S)",
	}, nil, nil)
	if err == nil {
		t.Fatal("a class no cell describes was accepted")
	}
	if !strings.Contains(err.Error(), "Secret (S)") {
		t.Errorf("the refusal does not name the class: %v", err)
	}
}

// TestTheFrontMatterBreaksBeforeVersionControlAndBeforeContents. The master
// pushes the version control label onto page two with eight empty paragraphs
// under the date, and the contents heading onto its own page with three more.
// Blank lines push a page only as long as nothing above them moves, so a
// slightly taller title reflowed the whole front matter. The two breaks are the
// same layout stated rather than measured.
func TestTheFrontMatterBreaksBeforeVersionControlAndBeforeContents(t *testing.T) {
	// Arrange
	pkg := build(t)

	// Act
	blocks := bodyBlocks(t, pkg)

	// Assert: two breaks, and nothing else in the document carries one.
	var breaks []int
	for i, block := range blocks {
		if block.FindElement("w:pPr/w:pageBreakBefore") != nil {
			breaks = append(breaks, i)
		}
	}
	if len(breaks) != 2 {
		t.Fatalf("the front matter carries %d page breaks, want 2", len(breaks))
	}

	// Assert: each break is an empty paragraph, and the block under it is the
	// label the new page opens with.
	for i, at := range breaks {
		if got := blocks[at].FullTag(); got != "w:p" {
			t.Errorf("break %d is a %s, want an empty paragraph", i, got)
		}
		if blocks[at].FindElement(".//w:t") != nil {
			t.Errorf("break %d carries text, and a page break writes none", i)
		}
	}
	if got := firstLabel(blocks[breaks[0]+1]); got != "Version Control" {
		t.Errorf("the first break is followed by %q, want the Version Control label", got)
	}
	if got := firstLabel(blocks[breaks[1]+1]); got != "Contents" {
		t.Errorf("the second break is followed by %q, want the Contents label", got)
	}

	// Assert: the cover's trailing blanks are gone, so the date line is the
	// last thing on page one and the break comes straight after it.
	if got := firstLabel(blocks[breaks[0]-1]); got != "8 September 2026" {
		t.Errorf("the block above the first break reads %q, and the cover now ends at the date line", got)
	}
	// The three blanks that pushed the contents heading are gone too: the
	// classification table is the block above the second break.
	if got := blocks[breaks[1]-1].FullTag(); got != "w:tbl" {
		t.Errorf("the block above the second break is a %s, want the classification table", got)
	}
}
