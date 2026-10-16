package render

import (
	"strings"
	"testing"
)

// TestACharacterStyleIsWrittenWithItsRunProperties. Both parts of this
// milestone name a character style on a run: the footnote reference, which is
// superscript, and a mark, which is a shading or a text colour. CT_Style is a
// sequence and so is the w:rPr inside it, and Word reads either out of order
// as a document to repair, so the writer states both orders as a literal list
// of tags. No one style carries all three properties; this one does so that
// the order of the three is pinned once.
func TestACharacterStyleIsWrittenWithItsRunProperties(t *testing.T) {
	style := charStyleElement("Yellow", "Yellow", charRunProps{
		Color: "#9A9A9A", Fill: "#FFD500", VertAlign: "superscript",
	})

	if got := style.FullTag(); got != "w:style" {
		t.Errorf("the writer made a %s, want a w:style", got)
	}
	if got := style.SelectAttrValue("w:type", ""); got != "character" {
		t.Errorf("w:type = %q, want character", got)
	}
	if got := style.SelectAttrValue("w:styleId", ""); got != "Yellow" {
		t.Errorf("w:styleId = %q, want Yellow", got)
	}
	if got := style.FindElement("w:name").SelectAttrValue("w:val", ""); got != "Yellow" {
		t.Errorf("w:name = %q, want Yellow", got)
	}

	var tags []string
	for _, e := range style.ChildElements() {
		tags = append(tags, e.FullTag())
	}
	want := []string{"w:name", "w:qFormat", "w:rPr"}
	if strings.Join(tags, " ") != strings.Join(want, " ") {
		t.Fatalf("the style's children run %v, want %v", tags, want)
	}

	rPr := style.SelectElement("w:rPr")
	var props []string
	for _, e := range rPr.ChildElements() {
		props = append(props, e.FullTag())
	}
	wantProps := []string{"w:color", "w:shd", "w:vertAlign"}
	if strings.Join(props, " ") != strings.Join(wantProps, " ") {
		t.Fatalf("the style's run properties run %v, want %v", props, wantProps)
	}

	if got := rPr.SelectElement("w:color").SelectAttrValue("w:val", ""); got != "9a9a9a" {
		t.Errorf("w:color = %q, want 9a9a9a", got)
	}
	shd := rPr.SelectElement("w:shd")
	if got := shd.SelectAttrValue("w:val", ""); got != "clear" {
		t.Errorf("w:shd w:val = %q, want clear", got)
	}
	if got := shd.SelectAttrValue("w:color", ""); got != "auto" {
		t.Errorf("w:shd w:color = %q, want auto", got)
	}
	if got := shd.SelectAttrValue("w:fill", ""); got != "ffd500" {
		t.Errorf("w:shd w:fill = %q, want ffd500", got)
	}
	if got := rPr.SelectElement("w:vertAlign").SelectAttrValue("w:val", ""); got != "superscript" {
		t.Errorf("w:vertAlign = %q, want superscript", got)
	}
}

// TestACharacterStyleThatStatesNothingCarriesNoRunProperties. An empty w:rPr
// is not wrong, but a style that states nothing is a style nobody asked for,
// and the empty element would hide the mistake.
func TestACharacterStyleThatStatesNothingCarriesNoRunProperties(t *testing.T) {
	style := charStyleElement("Plain", "Plain", charRunProps{})
	if style.SelectElement("w:rPr") != nil {
		t.Error("a character style stating no properties still carries a w:rPr")
	}
}
