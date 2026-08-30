package ooxml

import (
	"strings"
	"testing"

	"github.com/beevik/etree"
)

func serialise(t *testing.T, element *etree.Element) string {
	t.Helper()
	document := etree.NewDocument()
	document.SetRoot(element.Copy())
	out, err := document.WriteToString()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPPrChildrenAreWrittenInSchemaOrder(t *testing.T) {
	// Word rejects paragraph properties whose children are out of sequence, and
	// it is easy to break by appending: a w:spacing added after a w:ind is
	// already invalid.
	pPr := etree.NewElement("w:pPr")
	SetPPrChild(pPr, "w:ind", "left", "0")
	SetPPrChild(pPr, "w:spacing", "after", "240")
	SetPPrChild(pPr, "w:pStyle", "val", "Heading1")

	xml := serialise(t, pPr)
	style := strings.Index(xml, "w:pStyle")
	spacing := strings.Index(xml, "w:spacing")
	indent := strings.Index(xml, "w:ind ")
	if !(style < spacing && spacing < indent) {
		t.Errorf("order is pStyle=%d spacing=%d ind=%d, want that sequence:\n%s",
			style, spacing, indent, xml)
	}
}

func TestSettingAChildTwiceReplacesItRatherThanDuplicating(t *testing.T) {
	pPr := etree.NewElement("w:pPr")
	SetPPrChild(pPr, "w:spacing", "after", "240")
	SetPPrChild(pPr, "w:spacing", "after", "120")

	if got := len(pPr.SelectElements("w:spacing")); got != 1 {
		t.Errorf("w:spacing count = %d, want 1", got)
	}
	if got := pPr.SelectElement("w:spacing").SelectAttrValue("w:after", ""); got != "120" {
		t.Errorf("after = %q, want the second value", got)
	}
}

func TestARunPreservesTheSpacesAroundIt(t *testing.T) {
	// Leading and trailing spaces between runs are meaningful, and Word
	// silently eats them without xml:space="preserve".
	paragraph := etree.NewElement("w:p")
	TextRun(paragraph, " leading and trailing ", RunOpts{Sz: BodySz})

	xml := serialise(t, paragraph)
	if !strings.Contains(xml, `xml:space="preserve"`) {
		t.Errorf("no xml:space on the text:\n%s", xml)
	}
}

func TestMonoRunsAskForConsolas(t *testing.T) {
	paragraph := etree.NewElement("w:p")
	TextRun(paragraph, "code", RunOpts{Mono: true})

	if !strings.Contains(serialise(t, paragraph), `w:ascii="Consolas"`) {
		t.Error("a mono run must name a monospaced face")
	}
}

func TestAFontIsSetAcrossEveryScriptSlot(t *testing.T) {
	// Leaving eastAsia unset lets Word substitute a different face mid-word.
	rPr := etree.NewElement("w:rPr")
	SetFonts(rPr, "Calibri")

	xml := serialise(t, rPr)
	for _, slot := range []string{"ascii", "hAnsi", "cs", "eastAsia"} {
		if !strings.Contains(xml, `w:`+slot+`="Calibri"`) {
			t.Errorf("slot %q not set:\n%s", slot, xml)
		}
	}
}

func TestANewListRestartsItsNumbering(t *testing.T) {
	// A bare new w:num pointing at the same abstractNum does not restart
	// numbering in LibreOffice: the second ordered list carries on 4., 5., 6.
	root := etree.NewElement("w:numbering")
	Sub(root, "w:num", "numId", "3")
	numbering := NewNumbering(root)

	id := numbering.New(true, 1)

	if id != 4 {
		t.Errorf("numId = %d, want one past the template's highest", id)
	}
	created := root.SelectElements("w:num")[1]
	if got := len(created.SelectElements("w:lvlOverride")); got != 9 {
		t.Errorf("lvlOverride count = %d, want one per level", got)
	}
}

func TestAnOrderedListCanStartAtAnyNumber(t *testing.T) {
	numbering := NewNumbering(etree.NewElement("w:numbering"))

	numbering.New(true, 7)

	xml := serialise(t, numbering.part)
	if !strings.Contains(xml, `w:startOverride w:val="7"`) {
		t.Errorf("the start was not carried into the override:\n%s", xml)
	}
}

func TestTheTextColumnIsThePageLessBothMargins(t *testing.T) {
	// Written as a literal on purpose. Computing it from the constants would
	// make the assertion follow the constants rather than check them.
	if UsableTwips != 9864 {
		t.Errorf("usable width = %d twips, want 9864 for an A4 page with 1021 margins",
			UsableTwips)
	}
}

func TestTheHouseBodyIs12Point(t *testing.T) {
	// Half-points, so 24. Google Docs overrode Normal's 11pt in the master.
	if BodySz != "24" {
		t.Errorf("body size = %q half-points, want 12pt", BodySz)
	}
}

func TestLineSpacingIsOnePointOneFive(t *testing.T) {
	// In 240ths of a line, so 276.
	if LineSpacing != "276" {
		t.Errorf("line spacing = %q, want 1.15 lines", LineSpacing)
	}
}

func TestACellHasEnoughRoomForItsDescenders(t *testing.T) {
	// The master sets no cell margin at all, so descenders sit on the bottom
	// border. 40 twips is the floor: below that is a defect, not a tighter
	// style choice.
	if CellMarginV < "40" {
		t.Errorf("cell margin = %q twips, below the 40 twip floor", CellMarginV)
	}
}

func TestWeightFallsAsHeadingDepthGrows(t *testing.T) {
	// The master inverts this: Heading3 is regular while Heading4 and Heading6
	// are bold, so a child heading looks stronger than its parent.
	if !HeadingEmphasis[3].Bold {
		t.Error("Heading3 must be the strongest of the deep levels")
	}
	if HeadingEmphasis[4].Bold || HeadingEmphasis[4].Italic {
		t.Error("Heading4 must be plainer than Heading3")
	}
	if !HeadingEmphasis[5].Italic || HeadingEmphasis[5].Bold {
		t.Error("Heading5 must be plainer than Heading4")
	}
}
