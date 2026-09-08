package render

import (
	"strings"
	"testing"
)

func TestTheFirstPageHeaderCarriesTheLogoWhereTheHouseAnchorsIt(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/header2.xml"))

	anchor := doc.FindElement("//wp:anchor")
	if anchor == nil {
		t.Fatal("word/header2.xml carries no positioned drawing")
	}
	if got := anchor.SelectAttrValue("behindDoc", ""); got != "0" {
		t.Errorf("behindDoc = %q, want 0", got)
	}
	for _, attr := range []string{"distT", "distB", "distL", "distR"} {
		if got := anchor.SelectAttrValue(attr, ""); got != "114300" {
			t.Errorf("%s = %q, want 114300 EMU (9pt)", attr, got)
		}
	}
	extent := anchor.SelectElement("wp:extent")
	if extent == nil {
		t.Fatal("the anchor carries no extent")
	}
	if got := extent.SelectAttrValue("cx", ""); got != "1857375" {
		t.Errorf("logo cx = %q, want 1857375 EMU (146.25pt)", got)
	}
	if got := extent.SelectAttrValue("cy", ""); got != "923925" {
		t.Errorf("logo cy = %q, want 923925 EMU (72.75pt)", got)
	}
	h := anchor.SelectElement("wp:positionH")
	if got := h.SelectAttrValue("relativeFrom", ""); got != "column" {
		t.Errorf("positionH relativeFrom = %q, want column", got)
	}
	if got := h.SelectElement("wp:posOffset").Text(); got != "5133975" {
		t.Errorf("logo left offset = %q, want 5133975 EMU (404.25pt)", got)
	}
	v := anchor.SelectElement("wp:positionV")
	if got := v.SelectAttrValue("relativeFrom", ""); got != "paragraph" {
		t.Errorf("positionV relativeFrom = %q, want paragraph", got)
	}
	if got := v.SelectElement("wp:posOffset").Text(); got != "-65405" {
		t.Errorf("logo top offset = %q, want -65405 EMU (-5.15pt)", got)
	}
	if anchor.SelectElement("wp:wrapSquare") == nil {
		t.Error("the anchor does not wrap square, and the house style says it does")
	}
	if got := doc.FindElement("//a:blip").SelectAttrValue("r:embed", ""); got != "rId1" {
		t.Errorf("the logo names relationship %q, want rId1", got)
	}
}

func TestTheFirstPageHeaderOpensWithTheInternalUseLine(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/header2.xml"))
	first := doc.FindElement("//w:hdr").ChildElements()[0]
	text := first.FindElement("w:r/w:t")
	if text == nil {
		t.Fatal("the first header paragraph carries no text")
	}
	if text.Text() != "For internal use only " {
		t.Errorf("the first line reads %q, want \"For internal use only \"", text.Text())
	}
	rPr := first.FindElement("w:r/w:rPr")
	if got := rPr.SelectElement("w:color").SelectAttrValue("w:val", ""); got != "22265f" {
		t.Errorf("the line colour is %q, want 22265f", got)
	}
	if got := rPr.SelectElement("w:sz").SelectAttrValue("w:val", ""); got != "24" {
		t.Errorf("the line size is %q, want 24 half-points", got)
	}
}

func TestTheRunningHeadIsTheNotesOwnAndTheRuleIsUnderIt(t *testing.T) {
	pkg := build(t)
	raw := part(t, pkg, "word/header1.xml")
	if strings.Contains(string(raw), "Altery - xxx Policy") {
		t.Error("word/header1.xml still carries the template's placeholder running head")
	}
	doc := parse(t, raw)
	head := doc.FindElement("//w:hdr/w:p/w:r")
	if got := head.FindElement("w:t").Text(); got != "Altery - Supplier Register Policy" {
		t.Errorf("the running head reads %q, want Altery - Supplier Register Policy", got)
	}
	if got := head.FindElement("w:rPr/w:color").SelectAttrValue("w:val", ""); got != "ff771c" {
		t.Errorf("the running head colour is %q, want ff771c", got)
	}
	if got := head.FindElement("w:rPr/w:sz").SelectAttrValue("w:val", ""); got != "24" {
		t.Errorf("the running head size is %q, want 24 half-points", got)
	}
	rect := doc.FindElement("//w:pict/v:rect")
	if rect == nil {
		t.Fatal("word/header1.xml carries no rule under the running head")
	}
	if got := rect.SelectAttrValue("style", ""); got != "width:0.0pt;height:1.5pt" {
		t.Errorf("the rule style is %q, want width:0.0pt;height:1.5pt", got)
	}
	if got := rect.SelectAttrValue("fillcolor", ""); got != "#A0A0A0" {
		t.Errorf("the rule colour is %q, want #A0A0A0", got)
	}
}

func TestAHeaderWithNoLogoCarriesNoDrawing(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/header1.xml"))
	if doc.FindElement("//w:drawing") != nil {
		t.Error("the running head's header carries a drawing, and only the first page's does")
	}
}

func TestBothFootersEndWithThePageNumberField(t *testing.T) {
	pkg := build(t)
	for _, name := range []string{"word/footer1.xml", "word/footer2.xml"} {
		doc := parse(t, part(t, pkg, name))
		instr := doc.FindElement("//w:instrText")
		if instr == nil {
			t.Errorf("%s carries no field", name)
			continue
		}
		if instr.Text() != "PAGE" {
			t.Errorf("%s field is %q, want PAGE", name, instr.Text())
		}
		var kinds []string
		for _, e := range doc.FindElements("//w:fldChar") {
			kinds = append(kinds, e.SelectAttrValue("w:fldCharType", ""))
		}
		want := []string{"begin", "separate", "end"}
		if strings.Join(kinds, ",") != strings.Join(want, ",") {
			t.Errorf("%s field characters are %v, want %v", name, kinds, want)
		}
	}
}

func TestTheFirstPageFooterIsIndentedTheWayTheHouseSaysAndCarriesItsTabs(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/footer2.xml"))
	p := doc.FindElement("//w:ftr/w:p")
	if got := p.FindElement("w:pPr/w:ind").SelectAttrValue("w:left", ""); got != "2160" {
		t.Errorf("the first page footer indent is %q, want 2160 twips (108pt)", got)
	}
	if got := len(doc.FindElements("//w:tab")); got != 8 {
		t.Errorf("the first page footer carries %d tabs, want 8", got)
	}
	if got := p.FindElement("w:pPr/w:spacing").SelectAttrValue("w:line", ""); got != "480" {
		t.Errorf("the first page footer line spacing is %q, want 480 (2.0)", got)
	}
}
