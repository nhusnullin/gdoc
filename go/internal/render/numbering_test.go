package render

import (
	"testing"

	"github.com/beevik/etree"
)

// abstractNum returns one abstract list by its id.
func abstractNum(t *testing.T, doc *etree.Document, id string) *etree.Element {
	t.Helper()
	for _, e := range doc.FindElements("//w:abstractNum") {
		if e.SelectAttrValue("w:abstractNumId", "") == id {
			return e
		}
	}
	t.Fatalf("word/numbering.xml defines no abstract list %q", id)
	return nil
}

func TestNumberingHoldsTheBulletedAndTheNumberedList(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/numbering.xml"))
	if got := len(doc.FindElements("//w:abstractNum")); got != 2 {
		t.Fatalf("word/numbering.xml defines %d abstract lists, want 2", got)
	}
	for _, c := range []struct{ numID, abstract string }{{"1", "1"}, {"2", "2"}} {
		found := false
		for _, e := range doc.FindElements("//w:num") {
			if e.SelectAttrValue("w:numId", "") == c.numID {
				found = true
				if got := e.SelectElement("w:abstractNumId").SelectAttrValue("w:val", ""); got != c.abstract {
					t.Errorf("numId %s points at abstract %q, want %s", c.numID, got, c.abstract)
				}
			}
		}
		if !found {
			t.Errorf("word/numbering.xml defines no numId %s", c.numID)
		}
	}
}

func TestTheThreeBulletGlyphsAreAtTheHouseIndents(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/numbering.xml"))
	bullets := abstractNum(t, doc, "1")
	levels := bullets.SelectElements("w:lvl")
	if len(levels) != 9 {
		t.Fatalf("the bulleted list defines %d levels, want 9", len(levels))
	}
	wantGlyph := []string{"●", "○", "■"}
	wantLeft := []string{"720", "1440", "2160"}
	for i := range wantGlyph {
		lvl := levels[i]
		if got := lvl.SelectElement("w:lvlText").SelectAttrValue("w:val", ""); got != wantGlyph[i] {
			t.Errorf("bullet level %d glyph = %q, want %q", i, got, wantGlyph[i])
		}
		if got := lvl.SelectElement("w:numFmt").SelectAttrValue("w:val", ""); got != "bullet" {
			t.Errorf("bullet level %d numFmt = %q, want bullet", i, got)
		}
		ind := lvl.FindElement("w:pPr/w:ind")
		if ind == nil {
			t.Fatalf("bullet level %d carries no indent", i)
		}
		if got := ind.SelectAttrValue("w:left", ""); got != wantLeft[i] {
			t.Errorf("bullet level %d w:left = %q, want %s twips", i, got, wantLeft[i])
		}
		if got := ind.SelectAttrValue("w:hanging", ""); got != "360" {
			t.Errorf("bullet level %d w:hanging = %q, want 360 twips", i, got)
		}
	}
}

func TestTheNumberedListStartsAtOneDecimalAtThirtySixPoints(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/numbering.xml"))
	numbered := abstractNum(t, doc, "2")
	levels := numbered.SelectElements("w:lvl")
	if len(levels) != 9 {
		t.Fatalf("the numbered list defines %d levels, want 9", len(levels))
	}
	first := levels[0]
	if got := first.SelectElement("w:numFmt").SelectAttrValue("w:val", ""); got != "decimal" {
		t.Errorf("numbered level 0 numFmt = %q, want decimal", got)
	}
	if got := first.SelectElement("w:lvlText").SelectAttrValue("w:val", ""); got != "%1." {
		t.Errorf("numbered level 0 lvlText = %q, want %%1.", got)
	}
	if got := first.SelectElement("w:start").SelectAttrValue("w:val", ""); got != "1" {
		t.Errorf("numbered level 0 start = %q, want 1", got)
	}
	ind := first.FindElement("w:pPr/w:ind")
	if got := ind.SelectAttrValue("w:left", ""); got != "720" {
		t.Errorf("numbered level 0 w:left = %q, want 720 twips", got)
	}
	if got := ind.SelectAttrValue("w:hanging", ""); got != "360" {
		t.Errorf("numbered level 0 w:hanging = %q, want 360 twips", got)
	}
	if got := levels[1].SelectElement("w:lvlText").SelectAttrValue("w:val", ""); got != "%2." {
		t.Errorf("numbered level 1 lvlText = %q, want %%2.", got)
	}
}
