package render

import (
	"strings"

	"gdoc/internal/house"
	"github.com/beevik/etree"
)

// houseStyles pairs the config's ten style names with the ids Word knows them
// by. The order is the order they are written in, and a style missing from the
// config is refused in house.Validate before a build starts.
//
// FootnoteText is Word's own id and name for the notes under the page, so a
// person opening the document in Word finds the style where they expect it,
// and a note pasted out of another document lands in the house's own values
// rather than beside them.
var houseStyles = []struct {
	id     string
	name   string
	key    string
	based  string
	isDflt bool
}{
	{id: "Normal", name: "Normal", key: "normal", isDflt: true},
	{id: "Heading1", name: "heading 1", key: "heading_1", based: "Normal"},
	{id: "Heading2", name: "heading 2", key: "heading_2", based: "Normal"},
	{id: "Heading3", name: "heading 3", key: "heading_3", based: "Normal"},
	{id: "Heading4", name: "heading 4", key: "heading_4", based: "Normal"},
	{id: "Heading5", name: "heading 5", key: "heading_5", based: "Normal"},
	{id: "Heading6", name: "heading 6", key: "heading_6", based: "Normal"},
	{id: "Title", name: "Title", key: "title", based: "Normal"},
	{id: "Subtitle", name: "Subtitle", key: "subtitle", based: "Normal"},
	{id: "FootnoteText", name: "footnote text", key: "footnote_text", based: "Normal"},
}

// vertAlignSuperscript is Word's value for a raised character, which is the
// whole of what makes a footnote mark read as a mark rather than as a digit of
// the sentence. internal/body states the same value on the reference run
// itself, because Google Docs has no character styles and keeps only what the
// run says.
const vertAlignSuperscript = "superscript"

// charStyles is every w:type="character" style the part carries that no house
// value decides, in the order they are written. FootnoteReference is Word's
// own id and name. The seven marks follow it, each read off the palette.
var charStyles = []struct {
	id, name string
	props    charRunProps
}{
	{"FootnoteReference", "footnote reference", charRunProps{VertAlign: vertAlignSuperscript}},
}

// MarkStyleID is the character style id a mark's runs name: the mark's own
// house key with its first letter raised, so Word's style list reads Yellow,
// Green, Purple, Blue, Red, Orange and Pencil. internal/body calls it to name
// the style on the run and this package to write the style, which is the
// whole of what keeps the two from drifting. The keys are the seven
// house.MarkNames states and nothing else, which house.Validate refuses, and
// all seven are ASCII.
func MarkStyleID(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// markRunProps is what a mark's character style states: the exact hex as
// shading for a highlight, and the run's own text colour for the pencil.
// Which way round is the house file's, read through house.Mark.
func markRunProps(m house.Mark) charRunProps {
	if m.IsHighlight() {
		return charRunProps{Fill: m.Color}
	}
	return charRunProps{Color: m.Color}
}

// stylesPart writes word/styles.xml: the document defaults, the ten house
// paragraph styles, the footnote reference and the seven marks as character
// styles, then the table default every table is based on.
func (b *builder) stylesPart() []byte {
	doc, root := newPart("w:styles")
	b.docDefaults(root)
	for _, s := range houseStyles {
		style, ok := b.cfg.Style(s.key)
		if !ok {
			b.fail("styles: %s is missing", s.key)
			continue
		}
		root.AddChild(b.styleElement(s.id, s.name, style, s.based, s.isDflt))
	}
	for _, s := range charStyles {
		root.AddChild(charStyleElement(s.id, s.name, s.props))
	}
	for _, name := range house.MarkNames() {
		mark, ok := b.cfg.Marks[name]
		if !ok {
			b.fail("styles: the mark %s is missing", name)
			continue
		}
		id := MarkStyleID(name)
		root.AddChild(charStyleElement(id, id, markRunProps(mark)))
	}
	table := sub(root, "w:style", "w:type", "table", "w:default", "1", "w:styleId", "TableNormal")
	sub(table, "w:name", "w:val", "Table Normal")
	sub(table, "w:tblPr")
	return b.serialise(doc, "word/styles.xml")
}

// docDefaults is what a run and a paragraph get before any style applies.
func (b *builder) docDefaults(root *etree.Element) {
	d := b.cfg.Defaults
	defaults := sub(root, "w:docDefaults")

	rPr := sub(sub(defaults, "w:rPrDefault"), "w:rPr")
	fonts := sub(rPr, "w:rFonts")
	for _, slot := range []string{"w:ascii", "w:hAnsi", "w:cs", "w:eastAsia"} {
		fonts.CreateAttr(slot, d.Font)
	}
	sub(rPr, "w:sz", "w:val", halfPoints(d.SizePt))
	sub(rPr, "w:szCs", "w:val", halfPoints(d.SizePt))
	sub(rPr, "w:lang", "w:val", "en-GB")

	pPr := sub(sub(defaults, "w:pPrDefault"), "w:pPr")
	sub(pPr, "w:spacing",
		"w:before", twips(d.SpaceBeforePt),
		"w:after", twips(d.SpaceAfterPt),
		"w:line", lineTwentyFourths(d.LineSpacing),
		"w:lineRule", "auto")
}

// styleElement is one named style, its paragraph properties and its run
// properties, both read out of the config.
func (b *builder) styleElement(id, name string, s house.Style, based string, isDefault bool) *etree.Element {
	style := el("w:style", "w:type", "paragraph")
	if isDefault {
		style.CreateAttr("w:default", "1")
	}
	style.CreateAttr("w:styleId", id)
	sub(style, "w:name", "w:val", name)
	if based != "" {
		sub(style, "w:basedOn", "w:val", based)
	}
	sub(style, "w:qFormat")

	p := b.para(paraOpts{
		Align:       s.Align,
		Before:      s.SpaceBeforePt,
		After:       s.SpaceAfterPt,
		Line:        s.LineSpacing,
		IndentStart: s.IndentStartPt,
		KeepNext:    s.KeepWithNext,
		KeepLines:   s.KeepLinesTogether,
	})
	if pPr := p.SelectElement("w:pPr"); pPr != nil {
		style.AddChild(pPr)
	} else {
		sub(style, "w:pPr")
	}
	if rPr := runProps(runOpts{
		SizePt: s.SizePt, Bold: s.Bold, Italic: s.Italic,
		Color: s.Color, Font: s.Font,
	}); rPr != nil {
		style.AddChild(rPr)
	}
	return style
}

// charRunProps is what a character style states on the runs that name it: a
// text colour, a shading, a vertical alignment, or nothing. Only the three
// this generator needs are here, and a style carries at most two of them: the
// footnote reference is superscript, a highlight mark is a shading, and the
// pencil is a colour.
type charRunProps struct {
	Color     string // w:color, the pencil's grey
	Fill      string // w:shd w:fill, a highlight mark's own colour
	VertAlign string // w:vertAlign, the footnote reference's superscript
}

// empty says whether the style states no run properties at all.
func (p charRunProps) empty() bool {
	return p.Color == "" && p.Fill == "" && p.VertAlign == ""
}

// charStyleElement is one w:type="character" style: the id Word knows it by,
// the name a person reads in the style list, and the run properties it
// carries. A character style is how a run says what it is rather than only
// how it looks, so Word lists the marks and a reader finds every yellow with
// one search.
//
// CT_Style is a sequence and so is the w:rPr inside it, where EG_RPrBase puts
// colour, then shading, then vertical alignment. Word reads either out of
// order as a document to repair, so both are written in the schema's order
// and TestACharacterStyleIsWrittenWithItsRunProperties states them as a
// literal list of tags.
func charStyleElement(id, name string, p charRunProps) *etree.Element {
	style := el("w:style", "w:type", "character", "w:styleId", id)
	sub(style, "w:name", "w:val", name)
	sub(style, "w:qFormat")
	if p.empty() {
		return style
	}
	rPr := sub(style, "w:rPr")
	if p.Color != "" {
		sub(rPr, "w:color", "w:val", hexColor(p.Color))
	}
	if p.Fill != "" {
		// w:val clear and w:color auto are the pattern and its foreground: a
		// solid fill of the one colour, which is what a highlight is.
		sub(rPr, "w:shd", "w:val", "clear", "w:color", "auto", "w:fill", hexColor(p.Fill))
	}
	if p.VertAlign != "" {
		sub(rPr, "w:vertAlign", "w:val", p.VertAlign)
	}
	return style
}

// settingsPart tells Word to refresh its fields on open, which is what makes
// the contents list arrive filled in rather than empty.
func (b *builder) settingsPart() []byte {
	doc, root := newPart("w:settings")
	// CT_Settings is a sequence like every other properties element here, and
	// evenAndOddHeaders comes a long way before updateFields in it. Written
	// the other way round, which is what the Python spike wrote, Word reads
	// the part as a document to repair. gen.py had the same order and the
	// master carries neither element, so nothing downstream would name it.
	sub(root, "w:evenAndOddHeaders", "w:val", "false")
	sub(root, "w:updateFields", "w:val", "true")
	return b.serialise(doc, "word/settings.xml")
}
