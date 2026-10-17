package render

// The seven marks as word/styles.xml carries them, and the cover's own yellow
// beside them.
//
// Every id and every hex here is written out. The colours come from
// house.yaml, so a test that read them too would follow the palette wherever
// somebody moved it and claim nothing.

import (
	"testing"

	"gdoc/internal/cover"
)

// TestTheSevenCharacterStylesAreWritten. Each mark is a style Word lists, so a
// reader finds every yellow with one search, and each states the same colour
// the runs naming it state: shading with the exact hex for the six
// highlights, a text colour for the one grey.
func TestTheSevenCharacterStylesAreWritten(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/styles.xml"))

	cases := []struct {
		id, fill, color string
	}{
		{id: "Yellow", fill: "ffd500"},
		{id: "Green", fill: "3dd67a"},
		{id: "Purple", fill: "ba92ff"},
		{id: "Blue", fill: "5db3ff"},
		{id: "Red", fill: "ff6b6b"},
		{id: "Orange", fill: "ffb85c"},
		{id: "Pencil", color: "9a9a9a"},
	}
	for _, c := range cases {
		s := style(t, doc, c.id)
		if got := s.SelectAttrValue("w:type", ""); got != "character" {
			t.Errorf("%s is a %q style, want character", c.id, got)
		}
		if got := s.FindElement("w:name").SelectAttrValue("w:val", ""); got != c.id {
			t.Errorf("%s is named %q in Word's style list, want %s", c.id, got, c.id)
		}
		shd := s.FindElement("w:rPr/w:shd")
		if c.fill == "" {
			if shd != nil {
				t.Errorf("%s carries shading, want its text colour alone", c.id)
			}
			if got := s.FindElement("w:rPr/w:color").SelectAttrValue("w:val", ""); got != c.color {
				t.Errorf("%s is coloured %q, want %s", c.id, got, c.color)
			}
			continue
		}
		if shd == nil {
			t.Errorf("%s carries no shading", c.id)
			continue
		}
		if got := shd.SelectAttrValue("w:val", ""); got != "clear" {
			t.Errorf("%s shades with pattern %q, want clear", c.id, got)
		}
		if got := shd.SelectAttrValue("w:color", ""); got != "auto" {
			t.Errorf("%s shades with foreground %q, want auto", c.id, got)
		}
		if got := shd.SelectAttrValue("w:fill", ""); got != c.fill {
			t.Errorf("%s is filled %q, want %s", c.id, got, c.fill)
		}
		if s.FindElement("w:rPr/w:color") != nil {
			t.Errorf("%s states a text colour, want the shading alone", c.id)
		}
	}
}

// TestTheMarkStyleIDIsTheHouseKeyRaised. internal/body names the style on the
// run and this package writes the style, so the one function both call is the
// whole of what keeps the two from drifting.
func TestTheMarkStyleIDIsTheHouseKeyRaised(t *testing.T) {
	want := map[string]string{
		"yellow": "Yellow", "green": "Green", "purple": "Purple",
		"blue": "Blue", "red": "Red", "orange": "Orange", "pencil": "Pencil",
	}
	for key, id := range want {
		if got := MarkStyleID(key); got != id {
			t.Errorf("MarkStyleID(%q) = %q, want %s", key, got, id)
		}
	}
}

// TestTheCoverPlaceholdersAreNotMarks. The yellow on a cover placeholder is
// the template's, and it means "a person fills this in" rather than any of the
// seven. It keeps Word's named yellow and names no character style, so a
// restyle, an export or a reader counting marks never reads a cover as one.
func TestTheCoverPlaceholdersAreNotMarks(t *testing.T) {
	pkg, err := Build(config(t), houseLogo(t), cover.Fields{}, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	doc := parse(t, part(t, pkg, "word/document.xml"))

	found := 0
	for _, r := range doc.FindElements("//w:r") {
		if r.FindElement("w:rPr/w:highlight") == nil {
			continue
		}
		found++
		if got := r.FindElement("w:rPr/w:highlight").SelectAttrValue("w:val", ""); got != "yellow" {
			t.Errorf("a placeholder is highlighted %q, want Word's named yellow", got)
		}
		if r.FindElement("w:rPr/w:rStyle") != nil {
			t.Error("a cover placeholder names a character style, so it reads as one of the marks")
		}
		if r.FindElement("w:rPr/w:shd") != nil {
			t.Error("a cover placeholder carries run shading, so it reads as a highlight mark")
		}
	}
	if found == 0 {
		t.Error("a cover with nothing filled in carries no highlighted placeholder")
	}
}
