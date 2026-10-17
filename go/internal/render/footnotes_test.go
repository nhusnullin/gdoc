package render

import (
	"strconv"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// note is one footnote the walker would have handed Build: a number and one
// paragraph of words.
func note(id int, words string) Footnote {
	p := etree.NewElement("w:p")
	sub(p.CreateElement("w:pPr"), "w:pStyle", "w:val", "FootnoteText")
	p.CreateElement("w:r").CreateElement("w:t").SetText(words)
	return Footnote{ID: id, Blocks: []*etree.Element{p}}
}

// buildNotes is a package carrying the given notes.
func buildNotes(t *testing.T, notes []Footnote) *Package {
	t.Helper()
	pkg, err := Build(config(t), houseLogo(t), fields(), nil, nil, notes, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return pkg
}

// TestTheFootnotesPartCarriesTheTwoSeparators. Word writes two reserved notes
// into every footnotes part, the rule above the notes and the rule above a
// note continued onto the next page, at the ids -1 and 0. A part missing them
// is one Word repairs, so they are written whether or not the document has a
// footnote, and the numbered notes follow at 1.
func TestTheFootnotesPartCarriesTheTwoSeparators(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/footnotes.xml"))

	root := doc.Root()
	if got := root.FullTag(); got != "w:footnotes" {
		t.Fatalf("the part's root is %s, want w:footnotes", got)
	}
	notes := root.ChildElements()
	if len(notes) != 2 {
		t.Fatalf("a document with no footnotes carries %d notes, want the two separators", len(notes))
	}
	for i, c := range []struct{ kind, id, mark string }{
		{"separator", "-1", "w:separator"},
		{"continuationSeparator", "0", "w:continuationSeparator"},
	} {
		n := notes[i]
		if got := n.FullTag(); got != "w:footnote" {
			t.Errorf("note %d is a %s, want a w:footnote", i, got)
		}
		if got := n.SelectAttrValue("w:type", ""); got != c.kind {
			t.Errorf("note %d is typed %q, want %s", i, got, c.kind)
		}
		if got := n.SelectAttrValue("w:id", ""); got != c.id {
			t.Errorf("note %d has w:id %q, want %s", i, got, c.id)
		}
		if n.FindElement("w:p/w:r/"+c.mark) == nil {
			t.Errorf("note %d carries no %s", i, c.mark)
		}
	}
}

// TestEachNoteIsAFootnoteAtItsOwnNumber. The ids are internal/body's, which
// numbers a mark in reading order from 1, and the body's own
// w:footnoteReference names the same number.
func TestEachNoteIsAFootnoteAtItsOwnNumber(t *testing.T) {
	pkg := buildNotes(t, []Footnote{note(1, "the first note"), note(2, "the second")})
	doc := parse(t, part(t, pkg, "word/footnotes.xml"))

	notes := doc.Root().ChildElements()
	if len(notes) != 4 {
		t.Fatalf("the part holds %d notes, want the two separators and two notes", len(notes))
	}
	for i, words := range []string{"the first note", "the second"} {
		n := notes[2+i]
		if got := n.SelectAttrValue("w:type", ""); got != "" {
			t.Errorf("note %d is typed %q, and a numbered note is typed as nothing", i+1, got)
		}
		if got := n.SelectAttrValue("w:id", ""); got != strconv.Itoa(i+1) {
			t.Errorf("note %d has w:id %q, want %d", i+1, got, i+1)
		}
		if got := n.FindElement("w:p/w:r/w:t"); got == nil || got.Text() != words {
			t.Errorf("note %d does not carry %q", i+1, words)
		}
		if got := n.FindElement("w:p/w:pPr/w:pStyle").SelectAttrValue("w:val", ""); got != "FootnoteText" {
			t.Errorf("note %d is set in %q, want FootnoteText", i+1, got)
		}
	}
}

// TestTheFootnotesPartIsDeclaredAndRelated. A part Word cannot find is a part
// Word does not read: the content type says what it is and the relationship
// says where it is. The relationship is the shell's eighth, so a body picture
// starts one higher.
func TestTheFootnotesPartIsDeclaredAndRelated(t *testing.T) {
	pkg := build(t)

	types := parse(t, part(t, pkg, "[Content_Types].xml"))
	found := ""
	for _, e := range types.FindElements("//Override") {
		if e.SelectAttrValue("PartName", "") == "/word/footnotes.xml" {
			found = e.SelectAttrValue("ContentType", "")
		}
	}
	want := "application/vnd.openxmlformats-officedocument.wordprocessingml.footnotes+xml"
	if found != want {
		t.Errorf("/word/footnotes.xml is declared %q, want %q", found, want)
	}

	rels := parse(t, part(t, pkg, "word/_rels/document.xml.rels"))
	id, target := "", ""
	for _, e := range rels.FindElements("//Relationship") {
		if strings.HasSuffix(e.SelectAttrValue("Type", ""), "/footnotes") {
			id = e.SelectAttrValue("Id", "")
			target = e.SelectAttrValue("Target", "")
		}
	}
	if id != "rId8" {
		t.Errorf("the footnotes relationship is %q, want rId8", id)
	}
	if target != "footnotes.xml" {
		t.Errorf("the footnotes relationship targets %q, want footnotes.xml", target)
	}
	if FirstMediaRelID != 9 {
		t.Errorf("FirstMediaRelID = %d, want 9: the shell owns rId1 to rId8", FirstMediaRelID)
	}
}

// TestAnImageIdThatCollidesWithTheFootnotesIsRefused. rId8 was a picture's
// before this milestone, so the check that refuses a shell collision has to
// cover the new row rather than only the seven it was written for.
func TestAnImageIdThatCollidesWithTheFootnotesIsRefused(t *testing.T) {
	media := []Media{{RelID: "rId8", Name: "image1.png", Data: []byte("x")}}
	_, err := Build(config(t), houseLogo(t), fields(), nil, media, nil, 0)
	if err == nil {
		t.Fatal("Build accepted a picture at the footnotes relationship id")
	}
	if !strings.Contains(err.Error(), "rId8") {
		t.Errorf("the error does not name rId8: %v", err)
	}
}

// TestAFootnoteReferenceIsASuperscriptCharacterStyle. The mark in the prose is
// raised, which is the whole of what makes it read as a mark rather than as a
// digit of the sentence. It is Word's own built-in id and name, so a reader
// finds the style where they expect it.
func TestAFootnoteReferenceIsASuperscriptCharacterStyle(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/styles.xml"))

	var style *etree.Element
	for _, e := range doc.FindElements("//w:style") {
		if e.SelectAttrValue("w:styleId", "") == "FootnoteReference" {
			style = e
		}
	}
	if style == nil {
		t.Fatal("word/styles.xml carries no FootnoteReference style")
	}
	if got := style.SelectAttrValue("w:type", ""); got != "character" {
		t.Errorf("FootnoteReference is a %q style, want character", got)
	}
	if got := style.FindElement("w:name").SelectAttrValue("w:val", ""); got != "footnote reference" {
		t.Errorf("FootnoteReference is named %q, want footnote reference", got)
	}
	vert := style.FindElement("w:rPr/w:vertAlign")
	if vert == nil {
		t.Fatal("FootnoteReference states no w:vertAlign")
	}
	if got := vert.SelectAttrValue("w:val", ""); got != "superscript" {
		t.Errorf("FootnoteReference is %q, want superscript", got)
	}
}

// TestTheFootnoteTextStyleIsTheHousesTenth. The notes under the page are set
// in a paragraph style of their own, at Word's built-in id and name so the
// style list reads as a reader expects. Its values are house.yaml's and
// house_test.go states them as literals; what this asks is that they reach
// the part.
func TestTheFootnoteTextStyleIsTheHousesTenth(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/styles.xml"))

	var style *etree.Element
	for _, e := range doc.FindElements("//w:style") {
		if e.SelectAttrValue("w:styleId", "") == "FootnoteText" {
			style = e
		}
	}
	if style == nil {
		t.Fatal("word/styles.xml carries no FootnoteText style")
	}
	if got := style.SelectAttrValue("w:type", ""); got != "paragraph" {
		t.Errorf("FootnoteText is a %q style, want paragraph", got)
	}
	if got := style.FindElement("w:name").SelectAttrValue("w:val", ""); got != "footnote text" {
		t.Errorf("FootnoteText is named %q, want footnote text", got)
	}
	if got := style.FindElement("w:basedOn").SelectAttrValue("w:val", ""); got != "Normal" {
		t.Errorf("FootnoteText is based on %q, want Normal", got)
	}
	if got := style.FindElement("w:rPr/w:sz").SelectAttrValue("w:val", ""); got != "20" {
		t.Errorf("FootnoteText is %q half-points, want 20, which is 10pt", got)
	}
	spacing := style.FindElement("w:pPr/w:spacing")
	if spacing == nil {
		t.Fatal("FootnoteText states no w:spacing")
	}
	for _, c := range []struct{ attr, want string }{
		{"w:before", "0"},
		{"w:after", "0"},
		{"w:line", "240"},
	} {
		if got := spacing.SelectAttrValue(c.attr, ""); got != c.want {
			t.Errorf("FootnoteText spacing %s = %q, want %s", c.attr, got, c.want)
		}
	}
	if got := style.FindElement("w:rPr/w:rFonts").SelectAttrValue("w:ascii", ""); got != "Calibri" {
		t.Errorf("FootnoteText is set in %q, want Calibri, the body face", got)
	}
}
