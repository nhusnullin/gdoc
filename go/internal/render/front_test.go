package render

import (
	"testing"

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
	// The house file offers the title line twice, either side of an "or", so a
	// person filling the cover in by hand picks one. Nothing in the config marks
	// the second as an alternative, so the shell renders what it is given and the
	// cover milestone decides how a filled title collapses the pair.
}

func TestANoteWithNoTitleKeepsTheTemplatesHighlightedPlaceholder(t *testing.T) {
	pkg, err := Build(config(t), Fields{RunningHead: "Altery - xxx Policy"}, nil, nil)
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
