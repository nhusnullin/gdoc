package render

import (
	"fmt"
	"strconv"

	"github.com/beevik/etree"
)

// A list level is 0 to 8, so a list part defines nine of them whether or not
// the house style names a glyph for each.
const listLevels = 9

// The face the bullet glyphs are drawn in. Calibri has no filled square, so a
// list three deep loses its third glyph without this.
const bulletFont = "Noto Sans Symbols"

// numberingPart writes word/numbering.xml: one abstract list for bullets and
// one for numbers, then the w:num entries a body paragraph may name. There is
// one for bullets, and one per numbered list in the body, every one of them on
// the numbered abstract list.
//
// A w:num is where Word keeps a list's running count, so two numbered lists
// sharing one carry one count: the second list printed 3. and 4. where the
// author wrote 1. and 2. One each is what makes the second list start again.
//
// Each numbered one also states the restart rather than assuming its reader
// makes it. ECMA-376 17.9.27 has w:startOverride say the number a level begins
// at when it starts in the document, and a reader that keeps the count against
// the abstract list would otherwise carry the first list's number into the
// second, which is the defect the per-list w:num exists to fix. pandoc writes
// the override on every ordered list and python-docx documents it as the way to
// restart, so the file says the restart rather than resting on one reading of
// the spec. The Word master needs none: it gives every list an abstract
// definition of its own, seven w:num entries over seven w:abstractNum, which
// costs a definition per list and is not the route this part takes. The bullet
// list takes none either: a bullet has no count to carry.
func (b *builder) numberingPart(numberedLists int) []byte {
	doc, root := newPart("w:numbering")

	bullets := sub(root, "w:abstractNum", "w:abstractNumId", bulletAbstractID)
	b.bulletLevels(bullets)
	numbers := sub(root, "w:abstractNum", "w:abstractNumId", numberAbstractID)
	b.numberLevels(numbers)

	num := sub(root, "w:num", "w:numId", BulletNumID)
	sub(num, "w:abstractNumId", "w:val", bulletAbstractID)
	for i := 1; i <= numberedLists; i++ {
		num := sub(root, "w:num", "w:numId", NumberNumID(i))
		sub(num, "w:abstractNumId", "w:val", numberAbstractID)
		for level := 0; level < listLevels; level++ {
			override := sub(num, "w:lvlOverride", "w:ilvl", fmt.Sprint(level))
			sub(override, "w:startOverride", "w:val", "1")
		}
	}
	return b.serialise(doc, "word/numbering.xml")
}

// BulletNumID is the one list every bullet names. It is exported because the
// body walker writes w:numPr and has to name the list this part defines.
const BulletNumID = "1"

// NumberNumID is the list the n-th numbered list of the body names, counting
// from 1. The ids are computed rather than named as constants because there is
// one per list and the body only knows how many once it has walked: the
// arithmetic lives here, beside the part that writes the matching w:num, so
// the two cannot drift. The first numbered list is still "2", so a body
// paragraph in a document with one numbered list names the list it always did.
func NumberNumID(n int) string {
	return strconv.Itoa(n + 1)
}

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
