package render

import (
	"gdoc/internal/house"
	"github.com/beevik/etree"
)

// houseStyles pairs the config's nine style names with the ids Word knows them
// by. The order is the order they are written in, and a style missing from the
// config is refused in house.Validate before a build starts.
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
}

// stylesPart writes word/styles.xml: the document defaults, then the nine
// house styles, then the table default every table is based on.
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

// settingsPart tells Word to refresh its fields on open, which is what makes
// the contents list arrive filled in rather than empty.
func (b *builder) settingsPart() []byte {
	doc, root := newPart("w:settings")
	sub(root, "w:updateFields", "w:val", "true")
	sub(root, "w:evenAndOddHeaders", "w:val", "false")
	return b.serialise(doc, "word/settings.xml")
}
