// Package ooxml holds the low-level element helpers plus the formatting
// constants measured from the Altery Group Framework/Policy template.
//
// Every constant here was read out of the template's own word/document.xml,
// not invented. That is the whole reason the generated body matches the master:
// we reproduce the template's idioms rather than a tool's defaults.
//
// Unlike lxml, etree is not namespace-aware, so tags are written with their
// literal "w:" prefix. For this package that is a simplification rather than a
// compromise: a .docx uses one fixed prefix for the wordprocessingml namespace,
// and holding the prefix literally is what lets an untouched part round-trip
// byte for byte.
package ooxml

import (
	"strconv"

	"github.com/beevik/etree"
)

// --- measured from the template ---------------------------------------------
const (
	BodySz    = "24"       // 12pt half-points; Google Docs overrode Normal's 11pt
	HeadColor = "222660"   // run-level navy the template puts on every heading
	TblBorder = "c9c9c9"   // table-level grid
	HdrFill   = "bdcdd2"   // header-row shading
	RowFill   = "f3f8f9"   // body-row shading
	BodyFont  = "Calibri"
)

// The master draws a black border on every cell as well as the grey table grid,
// which turns a four-row table into a spreadsheet. Grey on every line lets the
// header band carry the structure instead of the gridlines.
const (
	CellBorder = TblBorder
	NoFill     = "ffffff"
)

// 3pt above and below the text in a cell. The master sets no cell margin at all,
// so descenders sit on the bottom border. Left and right stay at Word's own 108.
// The floor is 40 twips; anything below that is a defect, not a tighter style.
const (
	CellMarginV = "60"
	CellMarginH = "108"
)

// 1.15 line spacing, in 240ths of a line (240 = single). House rule for body
// prose; the master uses it on its own body paragraphs. Table cells stay single.
const LineSpacing = "276"

// Headings get the same 1.15 line so a heading that wraps to two lines is as
// readable as the prose beneath it, and so one rule covers the whole page.
const HeadingLineSpacing = LineSpacing

// 6pt beneath a heading, half the 12pt above it. The master sets zero, which
// glues the heading to the first line of its section.
const (
	HeadingSpaceAfter  = "120"
	HeadingSpaceBefore = "240"
)

// Emphasis for Heading3..Heading6, which all share the 12pt body size because
// the template's size scale runs out at level 3. The master inverts the
// hierarchy here, so emphasis is stepped down instead as depth grows.
type Emphasis struct {
	Bold   bool
	Italic bool
}

var HeadingEmphasis = map[int]Emphasis{
	3: {Bold: true, Italic: false},
	4: {Bold: false, Italic: false},
	5: {Bold: false, Italic: true},
	6: {Bold: false, Italic: true},
}

const (
	PageW    = 11906
	MarginL  = 1021
	MarginR  = 1021
)

const UsableTwips = PageW - MarginL - MarginR

// abstractNum ids already defined in the template's numbering.xml
const (
	AbstractOrdered = "3" // decimal / lowerLetter / lowerRoman
	AbstractBullet  = "2" // filled disc / hollow circle / filled square
)

var IndLeft = [4]int{720, 1440, 2160, 2880}

const (
	Hanging      = 360
	MaxListLevel = 8 // w:ilvl is 0..8
)

// --- element helpers ---------------------------------------------------------

// El makes a free-standing element. Attributes are given as name/value pairs so
// their order is fixed, which keeps output diffable between runs.
func El(tag string, attrs ...string) *etree.Element {
	e := etree.NewElement(tag)
	setAttrs(e, attrs)
	return e
}

// Sub appends a child element carrying the given attributes.
func Sub(parent *etree.Element, tag string, attrs ...string) *etree.Element {
	e := parent.CreateElement(tag)
	setAttrs(e, attrs)
	return e
}

func setAttrs(e *etree.Element, attrs []string) {
	if len(attrs)%2 != 0 {
		panic("ooxml: attributes must be name/value pairs")
	}
	for i := 0; i < len(attrs); i += 2 {
		e.CreateAttr("w:"+attrs[i], attrs[i+1])
	}
}

// SetFonts writes one font across every script slot, which is what the template
// does. Leaving eastAsia unset lets Word substitute a different face mid-word.
func SetFonts(rPr *etree.Element, font string) *etree.Element {
	f := rPr.CreateElement("w:rFonts")
	for _, slot := range []string{"ascii", "hAnsi", "cs", "eastAsia"} {
		f.CreateAttr("w:"+slot, font)
	}
	return f
}

// RunOpts is the formatting of one run. The zero value is plain body text.
type RunOpts struct {
	Bold      bool
	Italic    bool
	Mono      bool
	Color     string
	Highlight string
	Sz        string // "" means declare no size, so the style's wins
	Font      string
}

// TextRun appends a fully formed w:r carrying text.
//
// xml:space="preserve" is mandatory: leading and trailing spaces between runs
// are meaningful and Word silently eats them otherwise.
func TextRun(parent *etree.Element, text string, opts RunOpts) *etree.Element {
	r := parent.CreateElement("w:r")
	rPr := r.CreateElement("w:rPr")
	switch {
	case opts.Mono:
		SetFonts(rPr, "Consolas")
	case opts.Font != "":
		SetFonts(rPr, opts.Font)
	}
	if opts.Bold {
		Sub(rPr, "w:b", "val", "1")
		Sub(rPr, "w:bCs", "val", "1")
	}
	if opts.Italic {
		Sub(rPr, "w:i", "val", "1")
		Sub(rPr, "w:iCs", "val", "1")
	}
	if opts.Color != "" {
		Sub(rPr, "w:color", "val", opts.Color)
	}
	if opts.Highlight != "" {
		Sub(rPr, "w:highlight", "val", opts.Highlight)
	}
	if opts.Sz != "" {
		Sub(rPr, "w:sz", "val", opts.Sz)
		Sub(rPr, "w:szCs", "val", opts.Sz)
	}
	Sub(rPr, "w:rtl", "val", "0")
	t := r.CreateElement("w:t")
	t.CreateAttr("xml:space", "preserve")
	t.SetText(text)
	return r
}

// PPrOrder is the sequence the schema requires inside w:pPr. Word rejects
// paragraph properties whose children are out of order, and it is easy to break
// by appending: a w:spacing added after a w:ind is already invalid. Only the
// tags this code writes are listed; anything unlisted sorts to the end.
var PPrOrder = []string{
	"w:pStyle", "w:keepNext", "w:keepLines", "w:pageBreakBefore",
	"w:widowControl", "w:numPr", "w:pBdr", "w:shd", "w:tabs",
	"w:suppressAutoHyphens", "w:bidi", "w:spacing", "w:ind",
	"w:contextualSpacing", "w:jc", "w:outlineLvl", "w:rPr", "w:sectPr",
}

func pprRank(tag string) int {
	for i, name := range PPrOrder {
		if name == tag {
			return i
		}
	}
	return len(PPrOrder)
}

// SetPPrChild replaces tag inside pPr, inserted where the schema wants it.
// Existing copies are dropped first, so this is idempotent.
func SetPPrChild(pPr *etree.Element, tag string, attrs ...string) *etree.Element {
	for _, existing := range pPr.SelectElements(tag) {
		pPr.RemoveChild(existing)
	}
	element := El(tag, attrs...)
	rank := pprRank(tag)
	for _, child := range pPr.ChildElements() {
		if pprRank(child.FullTag()) > rank {
			pPr.InsertChildAt(child.Index(), element)
			return element
		}
	}
	pPr.AddChild(element)
	return element
}

// --- numbering ---------------------------------------------------------------

// Numbering hands out a fresh w:numId per list instance.
//
// Two things make this non-obvious, both learned the hard way:
//
//  1. Reuse the template's own abstractNum definitions, so bullet glyphs and the
//     decimal / lower-letter / lower-roman sequence come from the master.
//  2. A bare new w:num pointing at the same abstractNum does NOT restart
//     numbering in LibreOffice: the second ordered list carries on 4., 5., 6.
//     An explicit w:lvlOverride with w:startOverride on every one of the nine
//     levels is what forces the restart.
type Numbering struct {
	part   *etree.Element // the w:numbering root
	nextID int
}

func NewNumbering(numberingRoot *etree.Element) *Numbering {
	used := 0
	for _, num := range numberingRoot.SelectElements("w:num") {
		if id, err := strconv.Atoi(num.SelectAttrValue("w:numId", "")); err == nil && id > used {
			used = id
		}
	}
	return &Numbering{part: numberingRoot, nextID: used + 1}
}

func (n *Numbering) New(ordered bool, start int) int {
	numID := n.nextID
	n.nextID++

	abstract := AbstractBullet
	if ordered {
		abstract = AbstractOrdered
	}
	num := Sub(n.part, "w:num", "numId", strconv.Itoa(numID))
	Sub(num, "w:abstractNumId", "val", abstract)
	for level := 0; level < 9; level++ {
		override := Sub(num, "w:lvlOverride", "ilvl", strconv.Itoa(level))
		value := 1
		if level == 0 {
			value = start
		}
		Sub(override, "w:startOverride", "val", strconv.Itoa(value))
	}
	return numID
}
