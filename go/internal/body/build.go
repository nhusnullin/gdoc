package body

// The paragraph, heading, list item and caption builders, plus the small XML
// helpers they need. Every measurement comes from house.yaml; what is a
// constant here is a value the house file does not state, and each one carries
// the reason it is what it is.

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/beevik/etree"

	"gdoc/internal/render"
)

const (
	// A figure's caption: 10pt grey italic, under the picture.
	captionSizePt = 10.0
	captionColor  = "#595959"
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
//
// Style is the character style the run names, which is how a footnote
// reference and a mark reach Word's own style list: internal/render writes the
// style, and the run names it. Fill is the shading a highlight mark carries,
// VertAlign the superscript a footnote reference carries, and both are stated
// on the style as well, because Google Docs has no character styles and keeps
// only what the run itself says. There is no named highlight here: a mark is
// the exact hex, and the cover's own yellow placeholders are internal/render's
// and written by its own run writer.
type runOpts struct {
	SizePt    *float64
	Bold      bool
	Italic    bool
	Strike    bool
	Underline bool
	Mono      bool
	Color     string
	Font      string
	Style     string
	Fill      string
	VertAlign string
}

func (o runOpts) empty() bool {
	return o.SizePt == nil && !o.Bold && !o.Italic && !o.Strike && !o.Underline &&
		!o.Mono && o.Color == "" && o.Font == "" && o.Style == "" &&
		o.Fill == "" && o.VertAlign == ""
}

// runProps builds a w:rPr, or nil when the run states nothing. The order is
// the schema's: the character style, fonts, weight, slope, strike, colour,
// size, underline, shading, then vertical alignment.
func runProps(o runOpts) *etree.Element {
	if o.empty() {
		return nil
	}
	rPr := etree.NewElement("w:rPr")
	if o.Style != "" {
		sub(rPr, "w:rStyle", "w:val", o.Style)
	}
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
	// Underline is the last of the properties this writer had before the
	// marks: the run properties are a sequence, and Word reads one out of
	// order as a repair. The schema puts shading and vertical alignment after
	// it, in that order, which is why those two are written last.
	if o.Underline {
		sub(rPr, "w:u", "w:val", "single")
	}
	if o.Fill != "" {
		// w:val clear and w:color auto are the pattern and its foreground: a
		// solid fill of the one colour, which is what a highlight is. A
		// w:shd carrying the fill alone defaults its pattern to nil, and a
		// nil pattern draws no shading.
		sub(rPr, "w:shd", "w:val", "clear", "w:color", "auto", "w:fill", hexColor(o.Fill))
	}
	if o.VertAlign != "" {
		sub(rPr, "w:vertAlign", "w:val", o.VertAlign)
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

// marked is one run's formatting with a mark on it, or the same formatting
// back when the run carries none.
//
// A mark reaches Word twice over: the run names the character style
// internal/render writes, so Word lists the marks and a reader finds every
// yellow with one search, and the run states the colour itself, because
// Google Docs has no character styles and keeps only what the run says. A
// highlight is shading with the exact hex rather than one of Word's sixteen
// named highlights, none of which carries these hexes, and the pencil is the
// run's own colour.
//
// A mark the house style does not state is refused by name rather than
// dropped, the way an alignment it does not have is: a colour silently left
// off is a document nobody would think to check.
func (r *renderer) marked(o runOpts, name string) runOpts {
	if name == "" {
		return o
	}
	mark, ok := r.cfg.Marks[name]
	if !ok {
		r.fail("mark %q is not one the house style states", name)
		return o
	}
	o.Style = render.MarkStyleID(name)
	if mark.IsHighlight() {
		o.Fill = mark.Color
		return o
	}
	o.Color = mark.Color
	return o
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
//
// A destination opening with "#" is a jump inside this document rather than an
// address, so it is a w:anchor naming the bookmark on the heading and no
// relationship at all: written as a relationship, Word reads it as an address
// relative to the document's own location and clicking it opens nothing. A "#"
// naming no heading in the note is refused the way a code block is refused,
// named on the envelope and printed as the words the author wrote, because a
// jump that lands nowhere is worse than no jump.
func (r *renderer) addRuns(parent *etree.Element, runs []Run, base runOpts) {
	for _, run := range runs {
		// A footnote mark is a run with no text, so it is written before the
		// guard that drops one: what it says is the number, and Word prints
		// that from word/footnotes.xml.
		if run.Footnote > 0 {
			o := base
			o.Style = footnoteRefStyle
			o.VertAlign = vertAlignSuperscript
			mark := sub(parent, "w:r")
			if rPr := runProps(o); rPr != nil {
				mark.AddChild(rPr)
			}
			sub(mark, "w:footnoteReference", "w:id", strconv.Itoa(run.Footnote))
			continue
		}
		if run.Text == "" {
			continue
		}
		o := base
		o.Bold = o.Bold || run.Bold
		o.Italic = o.Italic || run.Italic
		o.Strike = run.Strike
		o.Mono = run.Mono
		o = r.marked(o, run.Mark)
		if run.Link == "" {
			textRun(parent, run.Text, o)
			continue
		}
		if anchor, ok := strings.CutPrefix(run.Link, "#"); ok {
			bookmark, found := r.anchors[anchor]
			if !found {
				r.warnDeadAnchor(anchor)
				textRun(parent, run.Text, o)
				continue
			}
			wrapper := sub(parent, "w:hyperlink", "w:anchor", bookmark)
			o.Color = linkColor
			o.Underline = true
			textRun(wrapper, run.Text, o)
			continue
		}
		wrapper := sub(parent, "w:hyperlink", "r:id", r.linkID(run.Link))
		o.Color = linkColor
		o.Underline = true
		textRun(wrapper, run.Text, o)
	}
}

// warnDeadAnchor names a jump that lands nowhere once for each line and
// destination. addRuns is reached once per run and merge splits a link's words
// at every mark boundary, so one link is several runs and warning from there
// unguarded prints the same sentence two or three times. Two links on one line
// naming the same missing heading are one sentence for the same reason: the
// author reads a line and a destination, and both copies say the same.
func (r *renderer) warnDeadAnchor(anchor string) {
	key := fmt.Sprintf("%d\x00%s", r.curLine, anchor)
	if r.deadAnchors[key] {
		return
	}
	r.deadAnchors[key] = true
	r.warn("line %d: the link to #%s names no heading in this note, so its words are printed as plain text",
		r.curLine, anchor)
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
//
// bookmark is the name Drive's import mints this heading's id from, and the
// name a jump to this heading lands on. The pair is empty and sits before the
// first run, which is the shape Google's own docx export writes and the shape
// measured to mint an id: the range carries nothing, so nothing in the
// paragraph can be moved out of it, and the pair stays inside the paragraph
// it names rather than on the paragraph mark Word drags around.
func (r *renderer) heading(level int, runs []Run, pageBreak bool, bookmark string) *etree.Element {
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
	if bookmark != "" {
		id := strconv.Itoa(r.bookmarkID)
		r.bookmarkID++
		sub(p, "w:bookmarkStart", "w:id", id, "w:name", bookmark)
		sub(p, "w:bookmarkEnd", "w:id", id)
	}
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
	// The gap between items is the house's own; the last item of a list
	// takes the body's space after instead, in endList.
	before, after := 0.0, r.cfg.Body.ListItemSpaceAfterPt
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

// footnoteParagraph is one paragraph of a note under the page: the house's
// footnote_text style, which internal/render writes, and nothing stated over
// it, because the style is where the size and the spacing of a note live.
//
// first says whether this is the note's opening paragraph, which carries
// Word's own reference mark, w:footnoteRef, in the same character style the
// mark in the prose names. Only the first: a second one in the same note
// prints the number twice.
func (r *renderer) footnoteParagraph(runs []Run, first bool) *etree.Element {
	p := r.para(paraOpts{Style: footnoteTextStyle})
	if first {
		mark := sub(p, "w:r")
		if rPr := runProps(runOpts{Style: footnoteRefStyle, VertAlign: vertAlignSuperscript}); rPr != nil {
			mark.AddChild(rPr)
		}
		sub(mark, "w:footnoteRef")
		// The space between the number and the note's first word. Word's own
		// template writes it as a run of its own, and a note whose text opens
		// hard against its number reads as one word.
		textRun(p, " ", runOpts{})
	}
	r.addRuns(p, runs, runOpts{})
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

// pageBreak is the author's own break, written as `---` on a line of its own:
// an empty paragraph carrying pageBreakBefore, which is the shape the front
// matter's three breaks already use. The spacing is stated as zero rather than
// left out, because a paragraph that holds nothing has no reason to push the
// page it opens down.
func (r *renderer) pageBreak() *etree.Element {
	before, after := 0.0, 0.0
	return r.para(paraOpts{PageBreak: true, BeforePt: &before, AfterPt: &after})
}

// isBreakParagraph says whether this block is one pageBreak wrote: a paragraph
// that breaks the page before it and holds no run at all. A heading breaks the
// page the same way and is not one, because it holds its own words.
func isBreakParagraph(block *etree.Element) bool {
	if block.Tag != "p" || len(block.SelectElements("w:r")) > 0 {
		return false
	}
	pPr := block.SelectElement("w:pPr")
	return pPr != nil && pPr.SelectElement("w:pageBreakBefore") != nil
}
