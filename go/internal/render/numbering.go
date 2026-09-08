package render

import (
	"fmt"

	"github.com/beevik/etree"
)

// A list level is 0 to 8, so a list part defines nine of them whether or not
// the house style names a glyph for each.
const listLevels = 9

// The face the bullet glyphs are drawn in. Calibri has no filled square, so a
// list three deep loses its third glyph without this.
const bulletFont = "Noto Sans Symbols"

// numberingPart writes word/numbering.xml: one abstract list for bullets and
// one for numbers, each reachable by the numId the body paragraphs name.
func (b *builder) numberingPart() []byte {
	doc, root := newPart("w:numbering")

	bullets := sub(root, "w:abstractNum", "w:abstractNumId", bulletAbstractID)
	b.bulletLevels(bullets)
	numbers := sub(root, "w:abstractNum", "w:abstractNumId", numberAbstractID)
	b.numberLevels(numbers)

	for _, pair := range [][2]string{
		{BulletNumID, bulletAbstractID},
		{NumberNumID, numberAbstractID},
	} {
		num := sub(root, "w:num", "w:numId", pair[0])
		sub(num, "w:abstractNumId", "w:val", pair[1])
	}
	return b.serialise(doc, "word/numbering.xml")
}

// The two lists, by the ids a body paragraph names. They are exported because
// the body walker writes w:numPr and has to name the same list this part
// defines.
const (
	BulletNumID = "1"
	NumberNumID = "2"
)

const (
	bulletAbstractID = "1"
	numberAbstractID = "2"
)

// bulletLevels writes one level per house glyph, each indented one step
// further, then fills the rest of the nine with the first glyph. A level the
// house does not describe still has to exist, or a list nested deeper than the
// style goes renders with no marker at all.
func (b *builder) bulletLevels(parent *etree.Element) {
	bullet := b.cfg.Body.Bullet
	for i, glyph := range bullet.Glyphs {
		lvl := sub(parent, "w:lvl", "w:ilvl", fmt.Sprint(i))
		sub(lvl, "w:start", "w:val", "1")
		sub(lvl, "w:numFmt", "w:val", "bullet")
		sub(lvl, "w:lvlText", "w:val", glyph)
		sub(lvl, "w:lvlJc", "w:val", "left")
		ind := sub(sub(lvl, "w:pPr"), "w:ind")
		ind.CreateAttr("w:left", twips(bullet.IndentStartPt*float64(i+1)))
		ind.CreateAttr("w:hanging", twips(bullet.HangingPt))
		fonts := sub(sub(lvl, "w:rPr"), "w:rFonts")
		fonts.CreateAttr("w:ascii", bulletFont)
		fonts.CreateAttr("w:hAnsi", bulletFont)
	}
	fallback := "●"
	if len(bullet.Glyphs) > 0 {
		fallback = bullet.Glyphs[0]
	}
	for i := len(bullet.Glyphs); i < listLevels; i++ {
		lvl := sub(parent, "w:lvl", "w:ilvl", fmt.Sprint(i))
		sub(lvl, "w:start", "w:val", "1")
		sub(lvl, "w:numFmt", "w:val", "bullet")
		sub(lvl, "w:lvlText", "w:val", fallback)
		sub(lvl, "w:lvlJc", "w:val", "left")
	}
}

// numberLevels writes the house's own level-1 format, then decimal levels
// beneath it. Each level prints its own number alone, so a nested list reads
// 1. then 1. rather than 1.1.
//
// Every level states w:start, the nested ones included. ECMA-376 17.9.26 reads
// an omitted start as zero, so a nested list whose level carried none opened at
// "0." and every item under it printed one lower than the author wrote. The
// master states it on all sixty-three of its own levels, which is what Word and
// Google write.
func (b *builder) numberLevels(parent *etree.Element) {
	numbered := b.cfg.Body.Numbered
	first := sub(parent, "w:lvl", "w:ilvl", "0")
	sub(first, "w:start", "w:val", "1")
	sub(first, "w:numFmt", "w:val", "decimal")
	sub(first, "w:lvlText", "w:val", numbered.Format)
	sub(first, "w:lvlJc", "w:val", "left")
	ind := sub(sub(first, "w:pPr"), "w:ind")
	ind.CreateAttr("w:left", twips(numbered.IndentStartPt))
	ind.CreateAttr("w:hanging", twips(numbered.HangingPt))

	for i := 1; i < listLevels; i++ {
		lvl := sub(parent, "w:lvl", "w:ilvl", fmt.Sprint(i))
		sub(lvl, "w:start", "w:val", "1")
		sub(lvl, "w:numFmt", "w:val", "decimal")
		sub(lvl, "w:lvlText", "w:val", fmt.Sprintf("%%%d.", i+1))
		sub(lvl, "w:lvlJc", "w:val", "left")
	}
}
