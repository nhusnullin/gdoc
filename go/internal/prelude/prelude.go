// Package prelude is the house template as Docs API requests: the cover, the
// three front-matter tables and the legend, proposed into a document that
// already exists.
//
// Two writers, one layout. internal/render writes this same template into a
// docx, and this package writes it as requests. Both read house.Config and
// neither holds a layout of its own, because a second copy of the layout is
// two documents that drift, which is the failure v1 had once its surgery file
// became the real template. A value added to house.yaml has to reach both
// writers, and the shared rules live where the value does: a placeholder's
// meaning is cover.Fields.Placeholder's, a classification cell's shading is
// house.Cell.FillFor's, and a measurement, a colour and a field mask are
// internal/docsreq's.
//
// Everything here is proposed, never written. The requests go out in
// writeMode SUGGEST on a document that was handed in and granted nothing,
// which is exactly what internal/propose sends every day, so this milestone
// needs no new guard permission for any of it. Nail accepts the prelude in the
// browser the way he accepts any suggestion, and rejecting it leaves the
// document as it was.
//
// gdoc's own words state their look in full. Text inserted into a document
// takes the look of the text it lands beside, so a cover line inserted in
// front of somebody's indented, justified, bold first paragraph would arrive
// wearing all of it. Every paragraph therefore states its named style, its
// alignment, its spacing and its indents, and every run states its face, its
// size, its weight and both of its colours. That is not the rule
// internal/restyle follows, and the difference is whose words are being
// styled: a restyle writes onto the author's own text, where a flag that
// house.yaml never stated would clear emphasis somebody meant, while these are
// gdoc's own lines and nobody else's emphasis can be in them.
//
// A table is inserted and then filled, which is more requests than the docx
// writer needs and is the shape the Docs API has: insertTable makes a grid of
// empty cells, so every word in it is an insertText afterwards and every fill,
// padding and border is an updateTableCellStyle. table.go holds the index
// accounting that goes with it, measured rather than assumed.
//
// Nothing here sends createParagraphBullets, because the house legend carries
// no bullets: its lines are a bold word, a tab and a sentence, and the master's
// own markup has no numbering on them. If a later block needs one, the reason
// it may be sent here and is refused at LevelInPlace is the same either way.
// createParagraphBullets removes the leading tabs that set a bullet's nesting
// level, so at LevelInPlace it deletes text an author typed, while here it
// would land on text gdoc itself proposed a moment earlier, where there are no
// author tabs to remove. The 2026-09-10 probe measured it accepted as a
// suggestion.
//
// A run marks what it proposed, with a named range called MarkerName, and that
// marker is gdoc's whole memory of having been here: a restyle writes no file
// beside the document and has no note to pair. marker.go holds the three shapes
// a run can meet and what it does about each, and it is the one thing in this
// package that reads the document rather than the house style.
//
// A list marker is the one look an inserted paragraph can inherit and this
// package cannot state away: removing one needs deleteParagraphBullets, which
// nothing here sends. A prelude proposed at the top of a document whose first
// paragraph is a list item arrives bulleted.
// docs/backlog/prelude-inherits-a-list-marker.md is the way out.
//
// Nothing here reaches the network, reads a file or decides anything. It is a
// pure function of the house style and the cover's values, and what the
// requests did is read back by the caller.
package prelude

import (
	"fmt"
	"strings"

	"gdoc/internal/cover"
	"gdoc/internal/docsreq"
	"gdoc/internal/house"
)

// Result is what one prelude will send, and where it will land.
type Result struct {
	// Requests are the requests in the order they must be applied. Each one
	// is built at the index the ones before it leave behind, so they are one
	// list rather than a set.
	Requests []map[string]any
	// Start is the index the prelude begins at, and End is one past the last
	// character it inserts. The marker a second run reads is created over that
	// range.
	Start int
	End   int
	// Paragraphs is how many paragraphs the prelude writes, Tables how many
	// tables it inserts and Cells how many of their cells it writes. They are
	// counts, not verdicts: whether the front matter is right is read in the
	// document.
	Paragraphs int
	Tables     int
	Cells      int
	// Replaces is the marker of the prelude a run before this one left, and
	// the span the requests propose deleting. It is nil on a first run, and on
	// a document whose prelude was rejected, which is the same shape: a
	// rejected insertion takes its marker with it.
	Replaces *Marker
	// Manual is what the prelude could not propose at all, each with the menu
	// path a person takes instead.
	//
	// It is the same shape internal/restyle reports its own steps in, and it
	// stays a separate list because the two are built from different things:
	// this one from the blocks house.yaml states, and restyle's from what the
	// in-place level cannot do to a document. The caller prints them in two
	// places, prelude.manual and read_back.manual, rather than merging them,
	// so a two-phase run names the contents list twice: house.yaml carries a
	// toc block and restyle's own list carries the same entry unconditionally.
	// Merging them is a decision about which object each step belongs to, not
	// a cleanup, so it is written down here rather than done quietly.
	Manual []ManualStep
}

// Occupies is the span of one tab's body gdoc's own words fill once this run's
// requests have landed, which is not always the span the marker covers.
//
// On a first run the two are the same: [Start, End) is the prelude and there is
// nothing else of gdoc's in the document. On a replace run they are not. The
// deletion goes out in SUGGEST mode, which marks text rather than removing it,
// so the prelude a run before this one left is still real text: the insert at
// Start pushes it along by exactly the length of the new prelude, and it comes
// to rest at [End, End + its own length), immediately behind the words that
// replace it.
//
// The caller that wants this is phase 2. A styling phase given [Start, End)
// alone walks past the new prelude and then gives the old one the house body
// look by direct edit, at LevelInPlace, which flattens a cover Nail may yet
// reject the deletion of and makes "rejecting it puts the document back as it
// was" false. Read is what it is styled from, and internal/restyle reads no
// suggestion id, so nothing further down could have noticed.
//
// It is not what the marker covers and not what the read-back asks about. The
// marker is this run's record of the prelude it proposed, and Verify counts
// whether every piece of that carries a suggestion id: the old prelude carries
// a deletion id rather than an insertion one, so asked about this wider span
// Verify would report gdoc's own replaced words as text somebody wrote.
func (r Result) Occupies() (start, end int) {
	if r.Replaces == nil {
		return r.Start, r.End
	}
	return r.Start, r.End + (r.Replaces.End - r.Replaces.Start)
}

// ManualStep is one thing gdoc could not do, and where a person does it.
type ManualStep struct {
	What  string `json:"what"`
	Where string `json:"where"`
}

// builder walks the house style once, inserting as it goes.
//
// The cursor only ever moves forward, and that is what makes the requests a
// list. Each insert lands at the end of what the inserts before it wrote, so
// every range this package has already stated is still the range it named:
// nothing later shifts anything earlier.
type builder struct {
	cfg    *house.Config
	fields cover.Fields
	at     int
	// requests is the list so far, and err is the first thing the house style
	// said that this package could not write. The first is kept rather than
	// the last, because the ones behind it are usually its consequences.
	requests []map[string]any
	count    int
	tables   int
	cells    int
	manual   []ManualStep
	err      error
}

// manualStep records one thing this writer could not propose. The list is a
// fact rather than a verdict: it says what is left to do, never whether the
// document is finished.
func (b *builder) manualStep(what, where string) {
	b.manual = append(b.manual, ManualStep{What: what, Where: where})
}

func (b *builder) fail(format string, args ...any) {
	if b.err == nil {
		b.err = fmt.Errorf(format, args...)
	}
}

// mark is the look of one of gdoc's own runs, as the house style states it.
type mark struct {
	sizePt    *float64
	bold      bool
	italic    bool
	underline bool
	color     string
	highlight string
	// font is the face this run is written in, for the one place the house
	// style states a face of its own: a table cell. Empty is the file's own
	// default face rather than the document's, because a run that states no
	// face takes the face of the text it was inserted beside.
	font string
}

// run is one stretch of gdoc's own words and the look it carries.
type run struct {
	text string
	look *docsreq.Fields
}

// paragraph writes one of gdoc's own paragraphs: the words, then the paragraph
// style over the whole of it, then one text style per run.
//
// The paragraph style's range carries the paragraph mark and the run styles do
// not, which is the same split the docx writer makes: a mark's own size is
// what a blank paragraph's height is. A paragraph with no words is its mark
// alone, so there the mark is what carries the run's look.
func (b *builder) paragraph(runs []run, look *docsreq.Fields) {
	start := b.at
	var words strings.Builder
	for _, r := range runs {
		words.WriteString(r.text)
	}
	text := words.String()
	b.request("insertText", map[string]any{
		"text":     text + "\n",
		"location": map[string]any{"index": start},
	})
	b.at = start + docsreq.Len(text) + 1
	b.count++
	b.style("updateParagraphStyle", "paragraphStyle", span(start, b.at), look)

	if text == "" {
		if len(runs) > 0 {
			b.style("updateTextStyle", "textStyle", span(start, start+1), runs[0].look)
		}
		return
	}
	at := start
	for _, r := range runs {
		end := at + docsreq.Len(r.text)
		if end == at {
			continue // a run with no words states nothing
		}
		b.style("updateTextStyle", "textStyle", span(at, end), r.look)
		at = end
	}
}

// blank is an empty paragraph at a stated height, which is what the cover's
// vertical rhythm is made of.
func (b *builder) blank(align string, sizePt, before, after, line *float64) {
	b.paragraph(
		[]run{{look: b.textLook(mark{sizePt: sizePt})}},
		b.paragraphLook(align, before, after, line),
	)
}

// request appends one request.
func (b *builder) request(kind string, body map[string]any) {
	b.requests = append(b.requests, map[string]any{kind: body})
}

// style appends one styling request over one range, with the mask the fields
// carry. Fields that state nothing send nothing: a request setting nothing
// resets nothing.
func (b *builder) style(kind, key string, at map[string]any, f *docsreq.Fields) {
	if f == nil || f.Empty() {
		return
	}
	b.request(kind, map[string]any{"range": at, key: f.Set, "fields": f.Mask()})
}

func span(start, end int) map[string]any {
	return map[string]any{"startIndex": start, "endIndex": end}
}

// paragraphLook is what every one of gdoc's own paragraphs states.
//
// The named style is stated because an inserted paragraph takes the named
// style of the one it landed in, so a cover line in front of somebody's
// Heading 1 would be a heading, in the contents list and in the house numbering
// with it. The indents are stated for the same reason.
//
// An unstated alignment is written as START rather than left out. Left out it
// is whatever the paragraph the prelude was inserted into carried, and a cover
// justified because the author's first paragraph was is a cover nobody chose.
func (b *builder) paragraphLook(align string, before, after, line *float64) *docsreq.Fields {
	return b.look(paraSpec{align: align, before: before, after: after, line: line})
}

// paraSpec is one of gdoc's own paragraphs as the house style states it. It is
// internal/render's paraOpts in this writer's units, and the two are separate
// because one writes twips into OOXML and the other points into a request.
type paraSpec struct {
	align             string
	before            *float64
	after             *float64
	line              *float64
	indentStartPt     *float64
	indentFirstLinePt *float64
	keepWithNext      bool
	keepLinesTogether bool
}

// look is one paragraph spec as a style object and the mask that names it.
//
// The two keeps are stated on every paragraph, false where the file says
// nothing. That is the opposite of internal/restyle's rule, and the reason is
// the one this package holds everywhere: a paragraph inserted beside somebody's
// kept-with-next heading inherits the flag, and these are gdoc's own lines,
// where no author's instruction can be cleared by stating one.
//
// A first-line indent is written as the house file states it, because Docs
// measures one from the margin. internal/render subtracts the paragraph indent
// from it, because OOXML measures it from there.
func (b *builder) look(s paraSpec) *docsreq.Fields {
	f := docsreq.NewFields()
	f.Put("namedStyleType", "NORMAL_TEXT")
	f.Put("alignment", b.alignment(s.align))
	f.Put("spaceAbove", docsreq.Points(value(s.before, b.cfg.Defaults.SpaceBeforePt)))
	f.Put("spaceBelow", docsreq.Points(value(s.after, b.cfg.Defaults.SpaceAfterPt)))
	f.Put("lineSpacing", docsreq.Percent(value(s.line, b.cfg.Defaults.LineSpacing)))
	f.Put("indentStart", docsreq.Points(value(s.indentStartPt, 0)))
	f.Put("indentFirstLine", docsreq.Points(value(s.indentFirstLinePt, 0)))
	f.Put("keepWithNext", s.keepWithNext)
	f.Put("keepLinesTogether", s.keepLinesTogether)
	return f
}

// alignment is one house alignment as Docs spells it. A word Docs has no
// alignment for is refused by name rather than dropped, which is the docx
// writer's rule: a paragraph silently left aligned is a document nobody would
// think to check.
func (b *builder) alignment(align string) string {
	if a, ok := docsreq.Alignment(align); ok {
		return a
	}
	if strings.TrimSpace(align) != "" {
		b.fail("alignment %q is not one the Docs API has", align)
	}
	return "START"
}

// textLook is what every one of gdoc's own runs states: the face, the size,
// the weight, the slope, the underline and both colours.
//
// Both colours, and the background is the one worth saying out loud. An
// OptionalColor with no colour in it is Docs' way of saying "no fill", so a run
// the house style does not highlight states an empty one and arrives unmarked
// even when it was inserted beside somebody's highlighted sentence.
func (b *builder) textLook(m mark) *docsreq.Fields {
	f := docsreq.NewFields()
	font := m.font
	if font == "" {
		font = b.cfg.Defaults.Font
	}
	if font != "" {
		f.Put("weightedFontFamily", map[string]any{"fontFamily": font})
	}
	f.Put("fontSize", docsreq.Points(value(m.sizePt, b.cfg.Defaults.SizePt)))
	f.Put("bold", m.bold)
	f.Put("italic", m.italic)
	f.Put("underline", m.underline)
	if c, ok := docsreq.Color(b.foreground(m.color)); ok {
		f.Put("foregroundColor", c)
	}
	f.Put("backgroundColor", b.background(m.highlight))
	return f
}

// foreground is the colour a run is written in: the one the house style states
// for it, else the one it states for ordinary text. A file that states neither
// leaves the colour to the document, because a colour invented here is one
// nobody wrote down.
func (b *builder) foreground(color string) string {
	if color != "" {
		return color
	}
	return b.cfg.Styles["normal"].Color
}

// background is a house highlight as the Docs API takes it. w:highlight names
// one of OOXML's seventeen colours and Docs takes an RGB value, so the names
// are spelled out below. An unnamed highlight, and the name "none", are the
// empty OptionalColor: no fill.
func (b *builder) background(highlight string) map[string]any {
	if highlight == "" || highlight == "none" {
		return map[string]any{}
	}
	hex, ok := highlightColors[highlight]
	if !ok {
		b.fail("highlight %q is not one this writer can spell as a colour", highlight)
		return map[string]any{}
	}
	c, ok := docsreq.Color(hex)
	if !ok {
		b.fail("highlight %q is not a colour", highlight)
		return map[string]any{}
	}
	return c
}

// highlightColors is OOXML's ST_HighlightColor as RGB, which is what the Docs
// API takes. house.Validate already refuses a name outside this set, so the
// two lists are the same seventeen names and a name added to one has to reach
// the other.
var highlightColors = map[string]string{
	"black":       "#000000",
	"blue":        "#0000FF",
	"cyan":        "#00FFFF",
	"darkBlue":    "#000080",
	"darkCyan":    "#008080",
	"darkGray":    "#808080",
	"darkGreen":   "#008000",
	"darkMagenta": "#800080",
	"darkRed":     "#800000",
	"darkYellow":  "#808000",
	"green":       "#00FF00",
	"lightGray":   "#C0C0C0",
	"magenta":     "#FF00FF",
	"red":         "#FF0000",
	"white":       "#FFFFFF",
	"yellow":      "#FFFF00",
}

// cellFill is the shading one front-matter cell carries. The rule is
// house.Cell's, because internal/render shades the same cells when it writes
// the same table into a docx.
func cellFill(c house.Cell, f cover.Fields) string {
	return c.FillFor(f.Classification)
}

// value is an optional house measurement, or the default the file states for
// everything when the block says nothing.
func value(v *float64, fallback float64) float64 {
	if v == nil {
		return fallback
	}
	return *v
}
