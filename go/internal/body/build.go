package body

// The paragraph, heading, list item and caption builders, plus the small XML
// helpers they need. Every measurement comes from house.yaml; what is a
// constant here is a value the house file does not state, and each one carries
// the reason it is what it is.

import (
	"math"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

const (
	// A figure's caption: 10pt grey italic, under the picture.
	captionSizePt = 10.0
	captionColor  = "#595959"
	// 3pt between list items. The house style sets none, which runs wrapped
	// bullets together into a block.
	listItemSpacePt = 3.0
	// 12pt above a paragraph that follows a table. A table carries no space
	// beneath it, so the next paragraph would otherwise sit hard against its
	// bottom border.
	afterTableSpacePt = 12.0
	// The face a code span is set in. Calibri has no fixed pitch, so an
	// identifier in prose reads as prose without this.
	monoFont = "Consolas"
	// Word's own link colour, blue and underlined, which is what a reader
	// expects a link to look like.
	linkColor = "#1155cc"
	// w:ilvl runs 0 to 8, so a list nested deeper than nine is drawn at the
	// deepest level the part defines rather than at one it does not.
	maxListLevel = 8
)

// alignments maps a house alignment to Word's own name. Word calls a justified
// paragraph "both", and the config says what a person would say.
var alignments = map[string]string{
	"left": "left", "right": "right", "center": "center",
	"justify": "both", "start": "left", "end": "right",
}

// twips is a point in twentieths, which is what OOXML measures an indent, a
// row height and a cell in.
func twips(pt float64) string { return strconv.Itoa(int(math.Round(pt * 20))) }

// halfPoints is a font size as OOXML states it.
func halfPoints(pt float64) string { return strconv.Itoa(int(math.Round(pt * 2))) }

// eighths is a border width as OOXML states it.
func eighths(pt float64) string { return strconv.Itoa(int(math.Round(pt * 8))) }

// hexColor drops the hash the config writes and lower-cases the rest. An
// absent colour is left to the style, so the caller checks for the empty
// string before it asks.
func hexColor(c string) string {
	return strings.ToLower(strings.TrimPrefix(c, "#"))
}

// sub appends a child element carrying the given attributes, as name/value
// pairs in the order given.
func sub(parent *etree.Element, tag string, attrs ...string) *etree.Element {
	e := parent.CreateElement(tag)
	if len(attrs)%2 != 0 {
		panic("body: attributes must be name/value pairs")
	}
	for i := 0; i < len(attrs); i += 2 {
		e.CreateAttr(attrs[i], attrs[i+1])
	}
	return e
}

// runOpts is the formatting of one run. The zero value declares nothing, so
// the style's own values win.
type runOpts struct {
	SizePt    *float64
	Bold      bool
	Italic    bool
	Strike    bool
	Underline bool
	Mono      bool
	Color     string
	Font      string
	Highlight string
}

func (o runOpts) empty() bool {
	return o.SizePt == nil && !o.Bold && !o.Italic && !o.Strike && !o.Underline &&
		!o.Mono && o.Color == "" && o.Font == "" && o.Highlight == ""
}

// runProps builds a w:rPr, or nil when the run states nothing. The order is
// the schema's: fonts, weight, slope, strike, colour, size, highlight, then
// underline.
func runProps(o runOpts) *etree.Element {
	if o.empty() {
		return nil
	}
	rPr := etree.NewElement("w:rPr")
	font := o.Font
	if o.Mono {
		font = monoFont
	}
	if font != "" {
		f := sub(rPr, "w:rFonts")
		for _, slot := range []string{"w:ascii", "w:hAnsi", "w:cs", "w:eastAsia"} {
			f.CreateAttr(slot, font)
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
	if o.Strike {
		sub(rPr, "w:strike", "w:val", "1")
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
// space between two runs is the space between two words.
func textRun(parent *etree.Element, text string, o runOpts) *etree.Element {
	r := sub(parent, "w:r")
	if rPr := runProps(o); rPr != nil {
		r.AddChild(rPr)
	}
	// A hard line break inside one run is a break element, not a newline: a
	// newline in a w:t is whitespace Word collapses.
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			sub(r, "w:br")
		}
		if line == "" {
			continue
		}
		sub(r, "w:t", "xml:space", "preserve").SetText(line)
	}
	return r
}

// paraOpts is one paragraph's properties. Every pointer is a value a caller
// may leave out, and absent means "write no attribute" rather than zero.
type paraOpts struct {
	Style       string
	Align       string
	BeforePt    *float64
	AfterPt     *float64
	LineSpacing *float64
	IndentPt    *float64
	HangingPt   *float64
	KeepNext    bool
	KeepLines   bool
	PageBreak   bool
	NumID       string // "" is not a list
	ILvl        int
	Mark        *runOpts // the paragraph mark's own run properties
}

// para builds a w:p carrying its properties. The children of w:pPr are written
// in the order the schema requires: Word rejects properties that are out of
// order, and it is easy to break by appending.
func (r *renderer) para(o paraOpts) *etree.Element {
	p := etree.NewElement("w:p")
	pPr := etree.NewElement("w:pPr")
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
	if o.NumID != "" {
		numPr := sub(pPr, "w:numPr")
		sub(numPr, "w:ilvl", "w:val", strconv.Itoa(o.ILvl))
		sub(numPr, "w:numId", "w:val", o.NumID)
	}
	if o.BeforePt != nil || o.AfterPt != nil || o.LineSpacing != nil {
		spacing := sub(pPr, "w:spacing")
		if o.BeforePt != nil {
			spacing.CreateAttr("w:before", twips(*o.BeforePt))
		}
		if o.AfterPt != nil {
			spacing.CreateAttr("w:after", twips(*o.AfterPt))
		}
		if o.LineSpacing != nil {
			spacing.CreateAttr("w:line", strconv.Itoa(int(math.Round(*o.LineSpacing*240))))
			spacing.CreateAttr("w:lineRule", "auto")
		}
	}
	if o.IndentPt != nil || o.HangingPt != nil {
		ind := sub(pPr, "w:ind")
		if o.IndentPt != nil {
			ind.CreateAttr("w:left", twips(*o.IndentPt))
		}
		if o.HangingPt != nil {
			ind.CreateAttr("w:hanging", twips(*o.HangingPt))
		}
	}
	if v := r.align(o.Align); v != "" {
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

// align maps a house alignment onto Word's. One the house does not have is
// refused by name rather than dropped: a paragraph silently left aligned is a
// document nobody would think to check.
func (r *renderer) align(a string) string {
	if a == "" {
		return ""
	}
	v, ok := alignments[a]
	if !ok {
		r.fail("alignment %q is not one Word has", a)
		return ""
	}
	return v
}

// addRuns writes a stretch of marked text into a paragraph or a cell. A run
// carrying a link is wrapped in a w:hyperlink naming the relationship the
// destination took.
func (r *renderer) addRuns(parent *etree.Element, runs []Run, base runOpts) {
	for _, run := range runs {
		if run.Text == "" {
			continue
		}
		o := base
		o.Bold = o.Bold || run.Bold
		o.Italic = o.Italic || run.Italic
		o.Strike = run.Strike
		o.Mono = run.Mono
		if run.Highlight != "" {
			o.Highlight = run.Highlight
		}
		if run.Link == "" {
			textRun(parent, run.Text, o)
			continue
		}
		wrapper := sub(parent, "w:hyperlink", "r:id", r.linkID(run.Link))
		o.Color = linkColor
		o.Underline = true
		textRun(wrapper, run.Text, o)
	}
}

// bodyRun is the run formatting an ordinary body paragraph starts from.
func (r *renderer) bodyRun() runOpts {
	size := r.cfg.Body.SizePt
	return runOpts{SizePt: &size}
}

// paragraph is one body paragraph: the house size, the house alignment and the
// house spacing.
func (r *renderer) paragraph(runs []Run) *etree.Element {
	before := r.cfg.Body.SpaceBeforePt
	if r.afterTable {
		before = afterTableSpacePt
	}
	after := r.cfg.Body.SpaceAfterPt
	size := r.cfg.Body.SizePt
	mark := runOpts{SizePt: &size}
	p := r.para(paraOpts{
		Align:    r.cfg.Body.Align,
		BeforePt: &before,
		AfterPt:  &after,
		Mark:     &mark,
	})
	r.addRuns(p, runs, r.bodyRun())
	return p
}

// heading is one heading paragraph, at the house style for its level and in
// the colour that style states.
func (r *renderer) heading(level int, runs []Run, pageBreak bool) *etree.Element {
	if level > 6 {
		level = 6
	}
	style, _ := r.cfg.Style("heading_" + strconv.Itoa(level))
	zero := 0.0
	mark := runOpts{Color: style.Color}
	p := r.para(paraOpts{
		Style:     "Heading" + strconv.Itoa(level),
		PageBreak: pageBreak,
		IndentPt:  &zero,
		HangingPt: &zero,
		Mark:      &mark,
	})
	r.addRuns(p, runs, runOpts{Color: style.Color})
	return p
}

// listItem is one bullet or one numbered item, at the level it sits at.
//
// The hanging indent is the marker's own column, so a paragraph with no marker
// does not get one: written on a continuation paragraph, w:hanging starts its
// first line in the column the number would have sat in and leaves the rest of
// the paragraph a step to the right of it.
func (r *renderer) listItem(runs []Run, numID string, level int, ordered bool) *etree.Element {
	indent, hanging := r.cfg.Body.Bullet.IndentStartPt, r.cfg.Body.Bullet.HangingPt
	if ordered {
		indent, hanging = r.cfg.Body.Numbered.IndentStartPt, r.cfg.Body.Numbered.HangingPt
	}
	indent = indent * float64(level+1)
	before, after := 0.0, listItemSpacePt
	size := r.cfg.Body.SizePt
	mark := runOpts{SizePt: &size}
	hangingPt := &hanging
	if numID == "" {
		hangingPt = nil
	}
	p := r.para(paraOpts{
		Align:     r.cfg.Body.Align,
		BeforePt:  &before,
		AfterPt:   &after,
		IndentPt:  &indent,
		HangingPt: hangingPt,
		NumID:     numID,
		ILvl:      level,
		Mark:      &mark,
	})
	r.addRuns(p, runs, r.bodyRun())
	return p
}

// caption is a figure's alt text, under the picture: centred, small, grey and
// italic.
func (r *renderer) caption(runs []Run) *etree.Element {
	size := captionSizePt
	before, after := 0.0, r.cfg.Body.SpaceAfterPt
	mark := runOpts{SizePt: &size, Color: captionColor, Italic: true}
	p := r.para(paraOpts{
		Align:    "center",
		BeforePt: &before,
		AfterPt:  &after,
		Mark:     &mark,
	})
	r.addRuns(p, runs, runOpts{SizePt: &size, Color: captionColor, Italic: true})
	return p
}

// quote is a block quote's paragraph: the body paragraph, indented one step
// per level of quoting.
//
// The indent is built with the paragraph rather than patched into a finished
// w:pPr. Patched in, its position was found by looking for w:jc, and a house
// file that states no body alignment has none: the fallback appended the w:ind
// after the paragraph mark's w:rPr, which is the last element the schema
// allows, and Word reads a w:pPr out of order as a repair.
func (r *renderer) quote(runs []Run, depth int) *etree.Element {
	before := r.cfg.Body.SpaceBeforePt
	if r.afterTable {
		before = afterTableSpacePt
	}
	after := r.cfg.Body.SpaceAfterPt
	size := r.cfg.Body.SizePt
	mark := runOpts{SizePt: &size}
	indent := r.cfg.Body.Bullet.IndentStartPt * float64(depth)
	p := r.para(paraOpts{
		Align:    r.cfg.Body.Align,
		BeforePt: &before,
		AfterPt:  &after,
		IndentPt: &indent,
		Mark:     &mark,
	})
	r.addRuns(p, runs, r.bodyRun())
	return p
}

// rule is a horizontal rule: an empty paragraph carrying a bottom border.
func (r *renderer) rule() *etree.Element {
	before, after := 0.0, r.cfg.Body.SpaceAfterPt
	p := r.para(paraOpts{BeforePt: &before, AfterPt: &after})
	pPr := p.SelectElement("w:pPr")
	bdr := etree.NewElement("w:pBdr")
	sub(bdr, "w:bottom", "w:val", "single", "w:sz", eighths(ruleWidthPt),
		"w:space", "1", "w:color", hexColor(r.ruleColor()))
	// w:pBdr comes before w:spacing in the schema.
	pPr.InsertChildAt(0, bdr)
	return p
}

// The horizontal rule's own line. The house file states no rule for the body,
// so this is the running head's weight at the palette's grey.
const ruleWidthPt = 0.5

// ruleColor is the palette's grey, or black when the file names none.
func (r *renderer) ruleColor() string {
	if c, ok := r.cfg.Palette["rule_grey"]; ok {
		return c
	}
	return "#000000"
}
