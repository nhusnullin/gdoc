package render

import (
	"testing"

	"github.com/beevik/etree"
)

// style returns one w:style by its id.
func style(t *testing.T, doc *etree.Document, id string) *etree.Element {
	t.Helper()
	for _, e := range doc.FindElements("//w:style") {
		if e.SelectAttrValue("w:styleId", "") == id {
			return e
		}
	}
	t.Fatalf("word/styles.xml defines no style %q", id)
	return nil
}

func TestStylesDefinesTheNineHouseStylesAndTheTableDefault(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/styles.xml"))
	for _, id := range []string{
		"Normal", "Heading1", "Heading2", "Heading3", "Heading4", "Heading5",
		"Heading6", "Title", "Subtitle", "TableNormal",
	} {
		style(t, doc, id)
	}
	if got := len(doc.FindElements("//w:style")); got != 10 {
		t.Errorf("word/styles.xml defines %d styles, want 10", got)
	}
	if got := style(t, doc, "Normal").SelectAttrValue("w:default", ""); got != "1" {
		t.Errorf("Normal w:default = %q, want 1", got)
	}
}

func TestHeading1IsSixteenPointBoldNavyWithTwelvePointsAbove(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/styles.xml"))
	h1 := style(t, doc, "Heading1")

	if got := h1.FindElement("w:rPr/w:sz").SelectAttrValue("w:val", ""); got != "32" {
		t.Errorf("Heading1 w:sz = %q, want 32 half-points", got)
	}
	if got := h1.FindElement("w:rPr/w:szCs").SelectAttrValue("w:val", ""); got != "32" {
		t.Errorf("Heading1 w:szCs = %q, want 32 half-points", got)
	}
	if h1.FindElement("w:rPr/w:b") == nil {
		t.Error("Heading1 is not bold")
	}
	if got := h1.FindElement("w:rPr/w:color").SelectAttrValue("w:val", ""); got != "22265f" {
		t.Errorf("Heading1 w:color = %q, want 22265f", got)
	}
	if got := h1.FindElement("w:pPr/w:spacing").SelectAttrValue("w:before", ""); got != "240" {
		t.Errorf("Heading1 w:before = %q, want 240 twips", got)
	}
	if got := h1.FindElement("w:pPr/w:spacing").SelectAttrValue("w:after", ""); got != "0" {
		t.Errorf("Heading1 w:after = %q, want 0", got)
	}
	if h1.FindElement("w:pPr/w:keepNext") == nil {
		t.Error("Heading1 does not keep with the paragraph behind it")
	}
	if got := h1.SelectElement("w:basedOn").SelectAttrValue("w:val", ""); got != "Normal" {
		t.Errorf("Heading1 is based on %q, want Normal", got)
	}
}

func TestTheTitleAndSubtitleCarryTheirOwnFaces(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/styles.xml"))

	title := style(t, doc, "Title")
	if got := title.FindElement("w:rPr/w:rFonts").SelectAttrValue("w:ascii", ""); got != "Arial" {
		t.Errorf("Title font = %q, want Arial", got)
	}
	if got := title.FindElement("w:pPr/w:jc").SelectAttrValue("w:val", ""); got != "center" {
		t.Errorf("Title alignment = %q, want center", got)
	}

	sub := style(t, doc, "Subtitle")
	if got := sub.FindElement("w:rPr/w:rFonts").SelectAttrValue("w:ascii", ""); got != "Georgia" {
		t.Errorf("Subtitle font = %q, want Georgia", got)
	}
	if got := sub.FindElement("w:rPr/w:sz").SelectAttrValue("w:val", ""); got != "48" {
		t.Errorf("Subtitle w:sz = %q, want 48 half-points", got)
	}
	if got := sub.FindElement("w:rPr/w:color").SelectAttrValue("w:val", ""); got != "666666" {
		t.Errorf("Subtitle w:color = %q, want 666666", got)
	}
	if sub.FindElement("w:rPr/w:i") == nil {
		t.Error("Subtitle is not italic")
	}
}

func TestTheDocumentDefaultsAreElevenPointCalibriAtOnePointOneFive(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/styles.xml"))
	rPr := doc.FindElement("//w:docDefaults/w:rPrDefault/w:rPr")
	if rPr == nil {
		t.Fatal("word/styles.xml carries no run defaults")
	}
	if got := rPr.SelectElement("w:rFonts").SelectAttrValue("w:ascii", ""); got != "Calibri" {
		t.Errorf("the default font is %q, want Calibri", got)
	}
	if got := rPr.SelectElement("w:sz").SelectAttrValue("w:val", ""); got != "22" {
		t.Errorf("the default size is %q, want 22 half-points", got)
	}
	spacing := doc.FindElement("//w:docDefaults/w:pPrDefault/w:pPr/w:spacing")
	if spacing == nil {
		t.Fatal("word/styles.xml carries no paragraph defaults")
	}
	if got := spacing.SelectAttrValue("w:line", ""); got != "276" {
		t.Errorf("the default line spacing is %q, want 276 (1.15)", got)
	}
	if got := spacing.SelectAttrValue("w:before", ""); got != "60" {
		t.Errorf("the default space before is %q, want 60 twips", got)
	}
	if got := spacing.SelectAttrValue("w:after", ""); got != "120" {
		t.Errorf("the default space after is %q, want 120 twips", got)
	}
}

// word/settings.xml is a sequence like every other properties element here.
// CT_Settings puts evenAndOddHeaders a long way before updateFields, and Word
// reads a part whose children are out of order as a document to repair. The
// sequence is written out as a literal list, the way the two table writers and
// w:pPr state theirs.
func TestSettingsChildrenAreInSchemaOrder(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/settings.xml"))
	var got []string
	for _, e := range doc.FindElement("//w:settings").ChildElements() {
		got = append(got, e.Tag)
	}
	want := []string{"evenAndOddHeaders", "updateFields"}
	if len(got) != len(want) {
		t.Fatalf("settings carries %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("settings child %d is %q, want %q (order: %v)", i, got[i], want[i], got)
		}
	}
}
