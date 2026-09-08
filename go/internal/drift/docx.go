// The docx half of the measurement: the same values, read out of the XML.
//
// This is what the offline gate runs on, so it never touches a network and it
// never needs a token. It reads the two files as documents rather than as
// bytes: a value stated on a style and overridden on the paragraph is resolved
// the way Word resolves it, because what the comparison is about is what a
// reader sees rather than which element carries the statement.
//
// etree rather than encoding/xml, and reading rather than writing. SPEC.md's
// reason for etree is that encoding/xml corrupts OOXML on the way back out;
// nothing here writes any, and what etree gives instead is a tree that a value
// can be walked out of in one line.
package drift

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

// The two namespaces this package names, and the prefix each document is
// expected to bind them to. The prefix is read off the root rather than
// assumed: a prefix is the document's own choice, and a file that spelled it
// differently would otherwise read as an empty document. The fallback is what
// every Word and Google export writes.
const (
	wordNS  = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	drawNS  = "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
	wordPfx = "w"
	drawPfx = "wp"
)

// maxPart is the ceiling on one part read out of the archive. A zip that
// decompresses to more than this is not a document gdoc made, and the rule is
// internal/docx's: a truncated part would be blamed on the XML parser.
const maxPart = 32 << 20

// emuPerPoint is DrawingML's unit. A logo's size and offsets are stated in
// EMU and every other length in this package is a point.
const emuPerPoint = 12700

// Docx is one document, opened. Every accessor below reads from it and none of
// them changes it.
type Docx struct {
	doc    *part
	styles *part
	rels   map[string]string
	parts  map[string][]byte
}

// part is one XML part with the prefix its own root binds to WordprocessingML.
type part struct {
	root *etree.Element
	w    string
	wp   string
}

// OpenDocx reads a docx from bytes. It is bytes rather than a path so a test
// and the gate run the same reader over the same package the builder produced,
// with nothing written to disk in between.
func OpenDocx(b []byte) (*Docx, error) {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, fmt.Errorf("drift: the file is not a docx: %w", err)
	}
	d := &Docx{parts: map[string][]byte{}, rels: map[string]string{}}
	for _, f := range z.File {
		raw, err := readEntry(f)
		if err != nil {
			return nil, err
		}
		d.parts[f.Name] = raw
	}
	if d.doc, err = d.part("word/document.xml"); err != nil {
		return nil, err
	}
	if d.styles, err = d.part("word/styles.xml"); err != nil {
		return nil, err
	}
	relsPart, err := d.part("word/_rels/document.xml.rels")
	if err != nil {
		return nil, err
	}
	for _, r := range relsPart.root.SelectElements("Relationship") {
		d.rels[r.SelectAttrValue("Id", "")] = r.SelectAttrValue("Target", "")
	}
	return d, nil
}

// OpenDocxFile reads a docx off disk, which is how the gate opens the master.
func OpenDocxFile(path string) (*Docx, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("drift: %s could not be read: %w", path, err)
	}
	return OpenDocx(b)
}

// readEntry reads one zip member under the ceiling. A member over it is an
// error naming the ceiling rather than a short read, which is the rule the
// export reader already holds: a part cut mid-element is blamed on the parser.
func readEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("drift: %s could not be opened: %w", f.Name, err)
	}
	defer rc.Close()
	raw, err := io.ReadAll(io.LimitReader(rc, maxPart+1))
	if err != nil {
		return nil, fmt.Errorf("drift: %s could not be read: %w", f.Name, err)
	}
	if len(raw) > maxPart {
		return nil, fmt.Errorf("drift: %s is over the %d byte ceiling this package reads", f.Name, maxPart)
	}
	return raw, nil
}

// part parses one XML part and reads the prefixes its root binds.
func (d *Docx) part(name string) (*part, error) {
	raw, ok := d.parts[name]
	if !ok {
		return nil, fmt.Errorf("drift: the docx carries no %s", name)
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(raw); err != nil {
		return nil, fmt.Errorf("drift: %s is not XML: %w", name, err)
	}
	root := doc.Root()
	if root == nil {
		return nil, fmt.Errorf("drift: %s has no root element", name)
	}
	return &part{root: root, w: prefixFor(root, wordNS, wordPfx), wp: prefixFor(root, drawNS, drawPfx)}, nil
}

// prefixFor reads the prefix a root binds to one namespace. A document that
// binds it as the default namespace has no prefix to read, so the fallback
// stands and the caller reads nothing, which is the honest answer rather than a
// guess at another spelling.
func prefixFor(root *etree.Element, uri, fallback string) string {
	for _, a := range root.Attr {
		if a.Space == "xmlns" && a.Value == uri {
			return a.Key
		}
	}
	return fallback
}

// The three walkers every accessor is written in terms of. Naming an element
// through the part's own prefix is what keeps the prefix a document's choice.
func (p *part) child(el *etree.Element, tag string) *etree.Element {
	if el == nil {
		return nil
	}
	return el.SelectElement(p.w + ":" + tag)
}

func (p *part) children(el *etree.Element, tag string) []*etree.Element {
	if el == nil {
		return nil
	}
	return el.SelectElements(p.w + ":" + tag)
}

func (p *part) attr(el *etree.Element, tag, name string) string {
	c := p.child(el, tag)
	if c == nil {
		return ""
	}
	return c.SelectAttrValue(p.w+":"+name, "")
}

// path walks a chain of children in one call.
func (p *part) path(el *etree.Element, tags ...string) *etree.Element {
	for _, t := range tags {
		el = p.child(el, t)
		if el == nil {
			return nil
		}
	}
	return el
}

// body is the document body, which is where the front matter, the tables and
// the section properties all live.
func (d *Docx) body() *etree.Element { return d.doc.child(d.doc.root, "body") }

// sect is the section properties: the page, the margins, and the four header
// and footer references.
func (d *Docx) sect() *etree.Element { return d.doc.child(d.body(), "sectPr") }

// ---------------------------------------------------------------- the page

// Page is one length off the section properties, in points. The names are the
// Docs API's, because the live half of every item reads them under those names.
func (d *Docx) Page(name string) any {
	s := d.sect()
	switch name {
	case "width":
		return twipsPt(d.doc.attr(s, "pgSz", "w"))
	case "height":
		return twipsPt(d.doc.attr(s, "pgSz", "h"))
	case "marginTop":
		return twipsPt(d.doc.attr(s, "pgMar", "top"))
	case "marginBottom":
		return twipsPt(d.doc.attr(s, "pgMar", "bottom"))
	case "marginLeft":
		return twipsPt(d.doc.attr(s, "pgMar", "left"))
	case "marginRight":
		return twipsPt(d.doc.attr(s, "pgMar", "right"))
	case "marginHeader":
		return twipsPt(d.doc.attr(s, "pgMar", "header"))
	case "marginFooter":
		return twipsPt(d.doc.attr(s, "pgMar", "footer"))
	}
	return nil
}

// FirstPageHeaderFooter is Word's titlePg, which is the flag Docs reports as
// useFirstPageHeaderFooter.
func (d *Docx) FirstPageHeaderFooter() any {
	return d.doc.child(d.sect(), "titlePg") != nil
}

// CustomHeaderFooterMargins says whether the section states its own header and
// footer distances rather than taking Word's. Docs reports the same fact under
// useCustomHeaderFooterMargins.
func (d *Docx) CustomHeaderFooterMargins() any {
	s := d.sect()
	return d.doc.attr(s, "pgMar", "header") != "" && d.doc.attr(s, "pgMar", "footer") != ""
}

// PageNumberStart is where the footer's page numbers begin.
func (d *Docx) PageNumberStart() any {
	v := d.doc.attr(d.sect(), "pgNumType", "start")
	if v == "" {
		return nil
	}
	return numberOf(v)
}

// reference returns the part one header or footer reference points at.
// kind is "header" or "footer", which is "default" or "first".
func (d *Docx) reference(kind, which string) string {
	for _, ref := range d.doc.children(d.sect(), kind+"Reference") {
		if ref.SelectAttrValue(d.doc.w+":type", "") != which {
			continue
		}
		target := d.rels[ref.SelectAttrValue("r:id", "")]
		if target == "" {
			return ""
		}
		return "word/" + target
	}
	return ""
}

// HasReference says whether the section names a header or footer of that kind.
func (d *Docx) HasReference(kind, which string) any {
	return d.reference(kind, which) != ""
}

// ------------------------------------------------------------- the styles

// Style is one named style, resolved. Every field is what a paragraph in that
// style ends up with: the document defaults, then the basedOn chain, then the
// style's own statement.
type Style struct {
	FontSize      any
	Font          any
	Colour        any
	Bold          bool
	Italic        bool
	Underline     bool
	SmallCaps     bool
	Strikethrough bool
	Alignment     any
	LineSpacing   any
	SpaceAbove    any
	SpaceBelow    any
	IndentStart   any
	KeepWithNext  bool
	found         bool
}

// styleElement finds one style by its id, and it takes the last definition
// rather than the first.
//
// That is not a preference. Google's docx export writes every style block
// several times: the master declares Heading1 six times, and the first of them
// carries a different colour, no bold and no indent from the five behind it.
// Word applies the last definition of a repeated style id, so the first is what
// nobody sees. Taking it would have this package report a colour and a weight
// no reader of the master has ever seen.
func (d *Docx) styleElement(id string) *etree.Element {
	var found *etree.Element
	for _, s := range d.styles.children(d.styles.root, "style") {
		if s.SelectAttrValue(d.styles.w+":styleId", "") == id {
			found = s
		}
	}
	return found
}

// Style resolves one named style. The chain is walked base first so the
// derived statement wins, which is the order Word applies them in.
//
// A style that is not in the file at all comes back with nothing found, and
// every field nil: the honest answer to "what does this document say about
// Heading 4" when it says nothing.
//
// That early return is the whole of the promise above. The document defaults
// used to be folded in before the chain was walked, and the chain is empty when
// the style is absent, so an absent style came back stating the defaults: 11pt
// Calibri at 115%, which is what the master states in its own docDefaults too.
// Drop a named style from house.yaml and six of that style's eleven rows would
// read the defaults on both sides and compare IDENTICAL, so the gate passed on
// a style that no longer existed.
func (d *Docx) Style(id string) Style {
	chain := []*etree.Element{}
	for seen, cur := map[string]bool{}, id; cur != ""; {
		el := d.styleElement(cur)
		if el == nil || seen[cur] {
			break
		}
		seen[cur] = true
		chain = append([]*etree.Element{el}, chain...)
		cur = d.styles.attr(el, "basedOn", "val")
	}
	if len(chain) == 0 {
		return Style{}
	}

	st := Style{}
	if dd := d.styles.child(d.styles.root, "docDefaults"); dd != nil {
		applyRun(d.styles, &st, d.styles.path(dd, "rPrDefault", "rPr"))
		d.applyPara(&st, d.styles.path(dd, "pPrDefault", "pPr"))
	}
	for _, el := range chain {
		st.found = true
		applyRun(d.styles, &st, d.styles.child(el, "rPr"))
		d.applyPara(&st, d.styles.child(el, "pPr"))
	}
	if st.found && st.Colour == nil {
		// An absent w:color is OOXML's "auto", which a reader sees as black.
		// Saying nil instead would report a MISSING against a file that spells
		// the same black out, which is a difference nobody can see.
		st.Colour = "#000000"
	}
	return st
}

// applyRun folds one run-properties element into the style being resolved. The
// part it reads through is a parameter rather than the document's styles.xml,
// because a header binds the WordprocessingML prefix in its own way and the
// walkers carry their own part's. It used to be read off the receiver, with a
// caller swapping the field and swapping it back, which made a *Docx unsafe to
// read from two goroutines at once and contradicted the type's own promise
// that no accessor changes it.
func applyRun(p *part, st *Style, rPr *etree.Element) {
	if rPr == nil {
		return
	}
	if v := p.attr(rPr, "sz", "val"); v != "" {
		if n, ok := numberOf(v).(float64); ok {
			st.FontSize = n / 2
		}
	}
	if f := p.child(rPr, "rFonts"); f != nil {
		if v := f.SelectAttrValue(p.w+":ascii", ""); v != "" {
			st.Font = v
		}
	}
	if c := p.child(rPr, "color"); c != nil {
		st.Colour = hexColour(c.SelectAttrValue(p.w+":val", ""))
	}
	for _, f := range []struct {
		tag string
		to  *bool
	}{
		{"b", &st.Bold}, {"i", &st.Italic}, {"u", &st.Underline},
		{"smallCaps", &st.SmallCaps}, {"strike", &st.Strikethrough},
	} {
		if el := p.child(rPr, f.tag); el != nil {
			*f.to = onOff(el.SelectAttrValue(p.w+":val", ""))
		}
	}
}

// applyPara folds one paragraph-properties element into the style.
func (d *Docx) applyPara(st *Style, pPr *etree.Element) {
	if pPr == nil {
		return
	}
	p := d.styles
	if v := p.attr(pPr, "jc", "val"); v != "" {
		st.Alignment = alignment(v)
	}
	if sp := p.child(pPr, "spacing"); sp != nil {
		if v := sp.SelectAttrValue(p.w+":before", ""); v != "" {
			st.SpaceAbove = twipsPt(v)
		}
		if v := sp.SelectAttrValue(p.w+":after", ""); v != "" {
			st.SpaceBelow = twipsPt(v)
		}
		if v := sp.SelectAttrValue(p.w+":line", ""); v != "" {
			if n, ok := numberOf(v).(float64); ok {
				// Word states line spacing in 240ths of a line; Docs states the
				// same thing as a percentage, and this list is read under the
				// Docs names.
				st.LineSpacing = round3(n / 240 * 100)
			}
		}
	}
	if v := p.attr(pPr, "ind", "left"); v != "" {
		st.IndentStart = twipsPt(v)
	}
	if el := p.child(pPr, "keepNext"); el != nil {
		st.KeepWithNext = onOff(el.SelectAttrValue(p.w+":val", ""))
	}
}

// -------------------------------------------------- the headers and footers

// Segment is one header or footer, reduced to what the comparison asks of it.
type Segment struct {
	Exists     bool
	Paragraphs int
	Text       string
	FirstRun   Style
}

// Segment reads one header or footer through the section's own reference, so a
// document that numbered its parts differently is still read correctly.
func (d *Docx) Segment(kind, which string) Segment {
	name := d.reference(kind, which)
	if name == "" {
		return Segment{}
	}
	p, err := d.part(name)
	if err != nil {
		return Segment{}
	}
	paras := p.children(p.root, "p")
	seg := Segment{Exists: true, Paragraphs: len(paras), Text: segmentText(p, paras)}
	for _, para := range paras {
		for _, r := range p.children(para, "r") {
			if strings.TrimSpace(runText(p, r)) == "" {
				continue
			}
			st := Style{}
			applyRun(p, &st, p.child(r, "rPr"))
			seg.FirstRun = st
			return seg
		}
	}
	return seg
}

// segmentText is the words a header or footer carries.
//
// A field prints as its instruction and never as the value cached behind it: a
// footer that was saved while page one was on screen carries a "1" that nothing
// typed, and the file the same footer came out of a moment earlier does not.
// The instruction is the fact, and it is what Docs reports as an autoText.
func segmentText(p *part, paras []*etree.Element) string {
	var b strings.Builder
	for _, para := range paras {
		for _, r := range p.children(para, "r") {
			b.WriteString(runText(p, r))
		}
	}
	return b.String()
}

// runText is one run's words, with the two things that are not words spelled
// out: a field instruction, and an inline picture.
func runText(p *part, r *etree.Element) string {
	var b strings.Builder
	skip := false
	for _, el := range r.ChildElements() {
		switch el.Tag {
		case "t":
			if !skip {
				b.WriteString(el.Text())
			}
		case "tab":
			if !skip {
				b.WriteString("\t")
			}
		case "instrText":
			b.WriteString("<" + strings.TrimSpace(el.Text()) + ">")
		case "fldChar":
			switch el.SelectAttrValue(p.w+":fldCharType", "") {
			case "separate":
				skip = true
			case "end":
				skip = false
			}
		case "drawing":
			// A positioned object is not text: it is reported on its own, the
			// way the Docs answer keeps it out of the paragraph's elements.
			if p.child(el, "inline") != nil || el.SelectElement(p.wp+":inline") != nil {
				b.WriteString("<INLINE_IMAGE>")
			}
		}
	}
	return b.String()
}

// ----------------------------------------------------------------- the logo

// Logo is the positioned picture in the first page header, which is the one
// thing the Docs API cannot create and the docx can.
type Logo struct {
	Present bool
	Mode    string
	Layout  string
	Width   any
	Height  any
	Left    any
	Top     any
}

// Logo reads the anchor out of the first page header.
func (d *Docx) Logo() Logo {
	name := d.reference("header", "first")
	if name == "" {
		return Logo{}
	}
	p, err := d.part(name)
	if err != nil {
		return Logo{}
	}
	for _, anchor := range p.root.FindElements("//" + p.wp + ":anchor") {
		l := Logo{Present: true, Mode: "POSITIONED", Layout: wrapOf(p, anchor)}
		if ext := anchor.SelectElement(p.wp + ":extent"); ext != nil {
			l.Width = emuPt(ext.SelectAttrValue("cx", ""))
			l.Height = emuPt(ext.SelectAttrValue("cy", ""))
		}
		l.Left = offsetOf(p, anchor, "positionH")
		l.Top = offsetOf(p, anchor, "positionV")
		return l
	}
	for _, inline := range p.root.FindElements("//" + p.wp + ":inline") {
		l := Logo{Present: true, Mode: "INLINE"}
		if ext := inline.SelectElement(p.wp + ":extent"); ext != nil {
			l.Width = emuPt(ext.SelectAttrValue("cx", ""))
			l.Height = emuPt(ext.SelectAttrValue("cy", ""))
		}
		return l
	}
	return Logo{}
}

// wrapOf is how the picture sits against the text, which is the fact Docs
// reports as the positioning layout.
func wrapOf(p *part, anchor *etree.Element) string {
	for _, el := range anchor.ChildElements() {
		if strings.HasPrefix(el.Tag, "wrap") {
			return el.Tag
		}
	}
	return ""
}

// offsetOf reads one axis of the anchor's position.
func offsetOf(p *part, anchor *etree.Element, axis string) any {
	el := anchor.SelectElement(p.wp + ":" + axis)
	if el == nil {
		return nil
	}
	off := el.SelectElement(p.wp + ":posOffset")
	if off == nil {
		return nil
	}
	return emuPt(off.Text())
}

// ------------------------------------------------------------------ the TOC

// TOCFields is every contents field in the body, as its instruction. A live
// contents list is one field; a document that lost it has none.
func (d *Docx) TOCFields() []string {
	out := []string{}
	for _, el := range d.doc.root.FindElements("//" + d.doc.w + ":instrText") {
		instr := strings.TrimSpace(el.Text())
		if strings.HasPrefix(instr, "TOC") {
			out = append(out, instr)
		}
	}
	return out
}

// TOCInstr is the instruction of the first contents field, which is the string
// Word reads to build the list. It is trimmed: whitespace either side of a
// field instruction is not part of the instruction, and Word and Google's
// export disagree about a trailing space.
func (d *Docx) TOCInstr() any {
	fields := d.TOCFields()
	if len(fields) == 0 {
		return nil
	}
	return fields[0]
}

// --------------------------------------------------------------- the tables

// TableValues is one table, reduced to the values the comparison reads.
type TableValues struct {
	Rows       int
	Columns    int
	ColWidths  []float64
	RowHeights []float64
	Fills      []string
	Texts      []string
	Borders    []string
}

// Tables reads the body's tables in the order they appear.
func (d *Docx) Tables() []TableValues {
	p := d.doc
	out := []TableValues{}
	for _, tbl := range p.children(d.body(), "tbl") {
		t := TableValues{ColWidths: []float64{}, RowHeights: []float64{}, Fills: []string{}, Texts: []string{}, Borders: []string{}}
		for _, gc := range p.children(p.child(tbl, "tblGrid"), "gridCol") {
			if n, ok := twipsPt(gc.SelectAttrValue(p.w+":w", "")).(float64); ok {
				t.ColWidths = append(t.ColWidths, n)
			}
		}
		t.Columns = len(t.ColWidths)
		rows := p.children(tbl, "tr")
		t.Rows = len(rows)
		for _, tr := range rows {
			h := 0.0
			if n, ok := twipsPt(p.attr(p.child(tr, "trPr"), "trHeight", "val")).(float64); ok {
				h = n
			}
			t.RowHeights = append(t.RowHeights, h)
			for _, tc := range p.children(tr, "tc") {
				tcPr := p.child(tc, "tcPr")
				t.Fills = append(t.Fills, fillOf(p, tcPr))
				t.Texts = append(t.Texts, strings.TrimSpace(cellText(p, tc)))
				t.Borders = append(t.Borders, borderOf(p, tcPr))
			}
		}
		out = append(out, t)
	}
	return out
}

// fillOf is the cell's background. Word writes an absent fill two ways, no
// shading element and a shading element reading "auto", and both mean the cell
// has no fill of its own.
func fillOf(p *part, tcPr *etree.Element) string {
	v := p.attr(tcPr, "shd", "fill")
	if v == "" || strings.EqualFold(v, "auto") {
		return ""
	}
	return "#" + strings.ToUpper(v)
}

// borderOf is the border a reader sees on the cell: the first edge it declares,
// as width and colour. Google omits an interior edge when the neighbouring cell
// declares it, which is compare.py's own note, so this reads one edge rather
// than counting declarations.
func borderOf(p *part, tcPr *etree.Element) string {
	borders := p.child(tcPr, "tcBorders")
	if borders == nil {
		return ""
	}
	for _, edge := range []string{"top", "bottom", "left", "right"} {
		el := p.child(borders, edge)
		if el == nil {
			continue
		}
		width := 0.0
		if n, ok := numberOf(el.SelectAttrValue(p.w+":sz", "")).(float64); ok {
			// Word states a border in eighths of a point.
			width = n / 8
		}
		return fmt.Sprintf("%.3fpt %s", width, hexOrEmpty(el.SelectAttrValue(p.w+":color", "")))
	}
	return ""
}

// cellText is every word in one cell.
func cellText(p *part, tc *etree.Element) string {
	var b strings.Builder
	for _, para := range p.children(tc, "p") {
		for _, r := range p.children(para, "r") {
			b.WriteString(runText(p, r))
		}
	}
	return b.String()
}

// ----------------------------------------------------------------- the body

// Heading is the first heading at one level, as the reader sees it.
type Heading struct {
	Found       bool
	Text        string
	IndentStart any
	Colour      any
}

// FirstHeading finds the first paragraph in the body carrying that heading
// style, and resolves the two values the comparison reads from the paragraph
// and its first run rather than from the style alone. That is the point: the
// master states one colour on the style and another on the paragraph, and a
// reader sees the paragraph's.
func (d *Docx) FirstHeading(level int) Heading {
	p := d.doc
	id := fmt.Sprintf("Heading%d", level)
	for _, para := range p.children(d.body(), "p") {
		pPr := p.child(para, "pPr")
		if p.attr(pPr, "pStyle", "val") != id {
			continue
		}
		style := d.Style(id)
		h := Heading{Found: true, Text: strings.TrimSpace(paraText(p, para)), IndentStart: style.IndentStart, Colour: style.Colour}
		if v := p.attr(pPr, "ind", "left"); v != "" {
			h.IndentStart = twipsPt(v)
		}
		if c := d.firstRunColour(p, para); c != nil {
			h.Colour = c
		}
		return h
	}
	return Heading{}
}

// firstRunColour is the colour of the first run with words in it.
func (d *Docx) firstRunColour(p *part, para *etree.Element) any {
	for _, r := range p.children(para, "r") {
		if strings.TrimSpace(runText(p, r)) == "" {
			continue
		}
		if c := p.child(p.child(r, "rPr"), "color"); c != nil {
			return hexColour(c.SelectAttrValue(p.w+":val", ""))
		}
		return nil
	}
	return nil
}

// BodyParagraph is the first real paragraph of prose after the first heading:
// the size, the alignment and the font a reader spends most of the document
// looking at.
type BodyParagraph struct {
	Found     bool
	FontSize  any
	Alignment any
	Font      any
}

// bodyProseMin is how long a paragraph has to be before it counts as prose.
// compare.py's number, and the reason is the same: a short line after a heading
// is usually a label or a caption in one of the two documents and a sentence in
// the other, so the two would not be the same paragraph.
const bodyProseMin = 40

// BodyParagraph reads that paragraph. Every value is resolved against the
// paragraph's style, because a run that states nothing is a run in the house
// body size rather than a run with no size.
func (d *Docx) BodyParagraph() BodyParagraph {
	p := d.doc
	seen := false
	for _, para := range p.children(d.body(), "p") {
		pPr := p.child(para, "pPr")
		style := p.attr(pPr, "pStyle", "val")
		if style == "Heading1" {
			seen = true
			continue
		}
		if !seen || p.child(pPr, "numPr") != nil {
			continue
		}
		text := strings.TrimSpace(paraText(p, para))
		if len(text) <= bodyProseMin {
			continue
		}
		base := d.Style(styleOr(style, "Normal"))
		out := BodyParagraph{Found: true, FontSize: base.FontSize, Alignment: base.Alignment, Font: base.Font}
		if v := p.attr(pPr, "jc", "val"); v != "" {
			out.Alignment = alignment(v)
		}
		for _, r := range p.children(para, "r") {
			if strings.TrimSpace(runText(p, r)) == "" {
				continue
			}
			rPr := p.child(r, "rPr")
			if v := p.attr(rPr, "sz", "val"); v != "" {
				if n, ok := numberOf(v).(float64); ok {
					out.FontSize = n / 2
				}
			}
			if f := p.child(rPr, "rFonts"); f != nil {
				if v := f.SelectAttrValue(p.w+":ascii", ""); v != "" {
					out.Font = v
				}
			}
			break
		}
		return out
	}
	return BodyParagraph{}
}

// Bullets counts the body paragraphs carrying a list number, which is how many
// list items the document has.
func (d *Docx) Bullets() any {
	p := d.doc
	n := 0
	for _, para := range p.children(d.body(), "p") {
		if p.child(p.child(para, "pPr"), "numPr") != nil {
			n++
		}
	}
	return float64(n)
}

// paraText is one paragraph's words.
func paraText(p *part, para *etree.Element) string {
	var b strings.Builder
	for _, r := range p.children(para, "r") {
		b.WriteString(runText(p, r))
	}
	return b.String()
}

// ------------------------------------------------------------------ numbers

// twipsPt reads a length stated in twentieths of a point. An empty value is
// nil: the document says nothing, which is not the same as saying zero.
func twipsPt(v string) any {
	n, ok := numberOf(v).(float64)
	if !ok {
		return nil
	}
	return round3(n / 20)
}

// emuPt reads a length stated in EMU.
func emuPt(v string) any {
	n, ok := numberOf(v).(float64)
	if !ok {
		return nil
	}
	return round3(n / emuPerPoint)
}

// numberOf reads a number, whether the file wrote it whole or with a decimal
// point behind it. Google's export writes a row height as 487.96875 and Word
// writes it as 488, and a reader that took only integers would report the first
// as no height at all.
func numberOf(v string) any {
	if v == "" {
		return nil
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil
	}
	return n
}

// round3 keeps three decimals, which is what compare.py kept. Further than that
// is the difference between two ways of writing one number.
//
// It goes through math.Round rather than a cast to int64. numberOf parses with
// strconv.ParseFloat, which takes "NaN" and "Inf" without an error, and Go
// leaves a float to int64 conversion out of that range implementation
// dependent: NaN read as 0 on darwin/arm64 and as -9.2e15 on darwin/amd64, both
// of which `make dist` ships. One document would then measure two ways
// depending on which binary ran. math.Round gives NaN back, so the row goes
// DIFFERENT instead of carrying a number nobody wrote.
func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}

// hexColour turns Word's colour into the spelling the Docs answer is read
// under. "auto" is black, which is what a reader sees.
func hexColour(v string) any {
	if v == "" {
		return nil
	}
	if strings.EqualFold(v, "auto") {
		return "#000000"
	}
	return "#" + strings.ToUpper(v)
}

// hexOrEmpty is hexColour where an absent colour is an empty string rather than
// nil, which is what a joined list of cell borders wants.
func hexOrEmpty(v string) string {
	c, ok := hexColour(v).(string)
	if !ok {
		return ""
	}
	return c
}

// onOff reads Word's boolean attribute. An element present with no value is
// true, which is what the schema says and what a reader of `<w:b/>` expects.
func onOff(v string) bool {
	switch strings.ToLower(v) {
	case "0", "false", "off":
		return false
	}
	return true
}

// alignment maps Word's justification onto the Docs names, so the two halves of
// one item report the same word for the same paragraph.
func alignment(v string) any {
	switch strings.ToLower(v) {
	case "left", "start":
		return "START"
	case "center", "centre":
		return "CENTER"
	case "both", "distribute":
		return "JUSTIFIED"
	case "right", "end":
		return "END"
	}
	return v
}

// styleOr is the paragraph's own style, or Normal when it names none.
func styleOr(id, fallback string) string {
	if id == "" {
		return fallback
	}
	return id
}
