// This file is the writer's own vocabulary: the part shell with its namespaces,
// the unit conversions, the run and paragraph builders, and the serialiser
// every part goes out through. render.go builds the package, front.go the cover
// and the front matter, headfoot.go the headers and footers, styles.go the
// named styles and the settings, numbering.go the two lists, and doc.go holds
// the package comment.

package render

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"gdoc/internal/cover"
	"gdoc/internal/house"
	"github.com/beevik/etree"
)

// The declaration every part opens with, as Word writes it.
const xmlProcInst = `version="1.0" encoding="UTF-8" standalone="yes"`

// namespaces is what a wordprocessingml root declares. The order is Word's.
var namespaces = [][2]string{
	{"xmlns:w", "http://schemas.openxmlformats.org/wordprocessingml/2006/main"},
	{"xmlns:r", "http://schemas.openxmlformats.org/officeDocument/2006/relationships"},
	{"xmlns:wp", "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"},
	{"xmlns:a", "http://schemas.openxmlformats.org/drawingml/2006/main"},
	{"xmlns:pic", "http://schemas.openxmlformats.org/drawingml/2006/picture"},
	{"xmlns:v", "urn:schemas-microsoft-com:vml"},
	{"xmlns:o", "urn:schemas-microsoft-com:office:office"},
	{"xmlns:w10", "urn:schemas-microsoft-com:office:word"},
	{"xmlns:mc", "http://schemas.openxmlformats.org/markup-compatibility/2006"},
	{"xmlns:w14", "http://schemas.microsoft.com/office/word/2010/wordml"},
}

// alignments maps a house alignment to Word's own name. Word calls a justified
// paragraph "both", and the config says what a person would say.
var alignments = map[string]string{
	"left": "left", "right": "right", "center": "center",
	"justify": "both", "start": "left", "end": "right",
}

// valigns maps a house vertical alignment to Word's own name. Word spells the
// middle "center", and the config says what a person would say.
var valigns = map[string]string{
	"top": "top", "center": "center", "middle": "center", "bottom": "bottom",
}

// twips is a point in twentieths, which is what OOXML measures a page, a
// margin, an indent and a cell in.
//
// Rounding is half away from zero, where Python's round goes to even. No house
// value lands on a half, so the two agree on every number in the file, and
// away-from-zero is the rule a person would expect if one ever did.
func twips(pt float64) string { return strconv.Itoa(int(math.Round(pt * 20))) }

// emu is a point in English Metric Units, which is what a drawing is placed in.
func emu(pt float64) string { return strconv.Itoa(int(math.Round(pt * 12700))) }

// halfPoints is a font size as OOXML states it.
func halfPoints(pt float64) string { return strconv.Itoa(int(math.Round(pt * 2))) }

// eighths is a border width as OOXML states it.
func eighths(pt float64) string { return strconv.Itoa(int(math.Round(pt * 8))) }

// lineTwentyFourths is a line spacing as OOXML states it: 240 is single.
func lineTwentyFourths(v float64) string { return strconv.Itoa(int(math.Round(v * 240))) }

// hexColor drops the hash the config writes and lower-cases the rest. An
// absent colour is black, which is what a paragraph with no colour renders as.
func hexColor(c string) string {
	if c == "" {
		return "000000"
	}
	return strings.ToLower(strings.TrimPrefix(c, "#"))
}

// builder is one build in progress. It carries the config, the note's fields
// and the first failure any helper hit, so an element helper stays a plain
// function of its input and the run still fails naming what it could not
// write. A part half built is never returned: Build checks err before it packs
// anything.
type builder struct {
	cfg    *house.Config
	fields cover.Fields
	err    error
}

// fail records the first failure and keeps the rest, because the first one is
// the one that explains the others.
func (b *builder) fail(format string, args ...any) {
	if b.err == nil {
		b.err = fmt.Errorf(format, args...)
	}
}

// align maps a house alignment onto Word's. An alignment the house does not
// have is refused by name rather than dropped: a paragraph silently left
// aligned is a document nobody would think to check.
func (b *builder) align(a string) string {
	if a == "" {
		return ""
	}
	v, ok := alignments[a]
	if !ok {
		b.fail("alignment %q is not one Word has", a)
		return ""
	}
	return v
}

// valign maps a house cell alignment onto Word's, and refuses one Word does not
// have rather than dropping it. A cell the config says is centred and the
// document leaves at the top is the config describing an output nobody wrote.
func (b *builder) valign(a string) string {
	if a == "" {
		return ""
	}
	v, ok := valigns[a]
	if !ok {
		b.fail("cell alignment %q is not one Word has", a)
		return ""
	}
	return v
}

// el makes a free-standing element. Attributes are name/value pairs, in the
// order given, and each name carries its own prefix so a wp: attribute is
// written the way a w: one is.
func el(tag string, attrs ...string) *etree.Element {
	e := etree.NewElement(tag)
	setAttrs(e, attrs)
	return e
}

// sub appends a child element carrying the given attributes.
func sub(parent *etree.Element, tag string, attrs ...string) *etree.Element {
	e := parent.CreateElement(tag)
	setAttrs(e, attrs)
	return e
}

func setAttrs(e *etree.Element, attrs []string) {
	if len(attrs)%2 != 0 {
		panic("render: attributes must be name/value pairs")
	}
	for i := 0; i < len(attrs); i += 2 {
		e.CreateAttr(attrs[i], attrs[i+1])
	}
}

// newPart starts one XML part: the declaration, then a root carrying the ten
// namespaces.
func newPart(rootTag string) (*etree.Document, *etree.Element) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", xmlProcInst)
	root := doc.CreateElement(rootTag)
	for _, ns := range namespaces {
		root.CreateAttr(ns[0], ns[1])
	}
	return doc, root
}

// serialise writes one part out. name is what a failure calls it.
func (b *builder) serialise(doc *etree.Document, name string) []byte {
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		b.fail("%s: %w", name, err)
		return nil
	}
	return buf.Bytes()
}

// runOpts is the formatting of one run. The zero value declares nothing, so
// the style's own values win.
type runOpts struct {
	SizePt    *float64
	Bold      bool
	Italic    bool
	Underline bool
	Color     string
	Font      string
	Highlight string
}

// empty says whether a run carries no formatting of its own.
func (o runOpts) empty() bool {
	return o.SizePt == nil && !o.Bold && !o.Italic && !o.Underline &&
		o.Color == "" && o.Font == "" && o.Highlight == ""
}

// runProps builds a w:rPr, or nil when the run states nothing. The order is
// the schema's: fonts, weight, slope, colour, size, highlight, then
// underline.
func runProps(o runOpts) *etree.Element {
	if o.empty() {
		return nil
	}
	rPr := el("w:rPr")
	if o.Font != "" {
		f := sub(rPr, "w:rFonts")
		for _, slot := range []string{"w:ascii", "w:hAnsi", "w:cs", "w:eastAsia"} {
			f.CreateAttr(slot, o.Font)
		}
	}
	if o.Bold {
		sub(rPr, "w:b", "w:val", "1")
		sub(rPr, "w:bCs", "w:val", "1")
	}
	if o.Italic {
		sub(rPr, "w:i", "w:val", "1")
		sub(rPr, "w:iCs", "w:val", "1")
	}
	if o.Color != "" {
		sub(rPr, "w:color", "w:val", hexColor(o.Color))
	}
	if o.SizePt != nil {
		sub(rPr, "w:sz", "w:val", halfPoints(*o.SizePt))
		sub(rPr, "w:szCs", "w:val", halfPoints(*o.SizePt))
	}
	if o.Highlight != "" {
		sub(rPr, "w:highlight", "w:val", o.Highlight)
	}
	// Underline is last because that is where the schema puts it: the run
	// properties are a sequence, and Word reads one out of order as a repair.
	if o.Underline {
		sub(rPr, "w:u", "w:val", "single")
	}
	return rPr
}

// textRun appends a run carrying text. xml:space is preserved because the
// house style's own strings end in spaces that line a header up.
func textRun(parent *etree.Element, text string, o runOpts) *etree.Element {
	r := sub(parent, "w:r")
	if rPr := runProps(o); rPr != nil {
		r.AddChild(rPr)
	}
	t := sub(r, "w:t", "xml:space", "preserve")
	t.SetText(text)
	return r
}

// tabsRun appends a run that is n tab characters and nothing else.
func tabsRun(parent *etree.Element, n int, o runOpts) *etree.Element {
	r := sub(parent, "w:r")
	if rPr := runProps(o); rPr != nil {
		r.AddChild(rPr)
	}
	for i := 0; i < n; i++ {
		sub(r, "w:tab")
	}
	return r
}

// pageFieldRun appends the page number as a field, so Word and Google both
// compute it rather than gdoc guessing at a page.
func pageFieldRun(parent *etree.Element, o runOpts) *etree.Element {
	r := sub(parent, "w:r")
	if rPr := runProps(o); rPr != nil {
		r.AddChild(rPr)
	}
	sub(r, "w:fldChar", "w:fldCharType", "begin")
	sub(r, "w:instrText", "xml:space", "preserve").SetText("PAGE")
	sub(r, "w:fldChar", "w:fldCharType", "separate")
	sub(r, "w:t").SetText("1")
	sub(r, "w:fldChar", "w:fldCharType", "end")
	return r
}

// paraOpts is one paragraph's properties. Every pointer is a value the config
// may leave out, and absent means "write no attribute" rather than zero.
type paraOpts struct {
	Style       string
	Align       string
	Before      *float64
	After       *float64
	Line        *float64
	IndentStart *float64
	Hanging     *float64
	FirstLine   *float64
	KeepNext    bool
	KeepLines   bool
	PageBreak   bool
	NumID       int // 0 is no list
	ILvl        int
	Mark        *runOpts // the paragraph mark's own run properties
}

// para builds a w:p carrying its properties. The children of w:pPr are written
// in the order the schema requires: Word rejects properties that are out of
// order, and it is easy to break by appending.
func (b *builder) para(o paraOpts) *etree.Element {
	p := el("w:p")
	pPr := el("w:pPr")
	if o.Style != "" {
		sub(pPr, "w:pStyle", "w:val", o.Style)
	}
	if o.KeepNext {
		sub(pPr, "w:keepNext", "w:val", "1")
	}
	if o.KeepLines {
		sub(pPr, "w:keepLines", "w:val", "1")
	}
	if o.PageBreak {
		sub(pPr, "w:pageBreakBefore", "w:val", "1")
	}
	if o.NumID != 0 {
		numPr := sub(pPr, "w:numPr")
		sub(numPr, "w:ilvl", "w:val", strconv.Itoa(o.ILvl))
		sub(numPr, "w:numId", "w:val", strconv.Itoa(o.NumID))
	}
	if o.Before != nil || o.After != nil || o.Line != nil {
		spacing := sub(pPr, "w:spacing")
		if o.Before != nil {
			spacing.CreateAttr("w:before", twips(*o.Before))
		}
		if o.After != nil {
			spacing.CreateAttr("w:after", twips(*o.After))
		}
		if o.Line != nil {
			spacing.CreateAttr("w:line", lineTwentyFourths(*o.Line))
			spacing.CreateAttr("w:lineRule", "auto")
		}
	}
	if o.IndentStart != nil || o.Hanging != nil || o.FirstLine != nil {
		ind := sub(pPr, "w:ind")
		if o.IndentStart != nil {
			ind.CreateAttr("w:left", twips(*o.IndentStart))
		}
		if o.Hanging != nil && *o.Hanging != 0 {
			ind.CreateAttr("w:hanging", twips(*o.Hanging))
		} else {
			first := 0.0
			if o.FirstLine != nil {
				first = *o.FirstLine
			}
			ind.CreateAttr("w:firstLine", twips(first))
		}
	}
	if v := b.align(o.Align); v != "" {
		sub(pPr, "w:jc", "w:val", v)
	}
	if o.Mark != nil {
		if rPr := runProps(*o.Mark); rPr != nil {
			pPr.AddChild(rPr)
		}
	}
	if len(pPr.ChildElements()) > 0 {
		p.AddChild(pPr)
	}
	return p
}

// blank is an empty paragraph that still carries a run size, because an empty
// paragraph's height is what the cover's vertical rhythm is made of.
func (b *builder) blank(o paraOpts, sizePt *float64) *etree.Element {
	mark := runOpts{SizePt: sizePt}
	o.Mark = &mark
	p := b.para(o)
	r := sub(p, "w:r")
	if rPr := runProps(mark); rPr != nil {
		r.AddChild(rPr)
	}
	return p
}
