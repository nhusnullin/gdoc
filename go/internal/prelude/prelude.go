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
	// Paragraphs is how many paragraphs the prelude writes. It is a count, not
	// a verdict: whether the cover is right is read in the document.
	Paragraphs int
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
	err      error
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
	color     string
	highlight string
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
	f := docsreq.NewFields()
	f.Put("namedStyleType", "NORMAL_TEXT")
	f.Put("alignment", b.alignment(align))
	f.Put("spaceAbove", docsreq.Points(value(before, b.cfg.Defaults.SpaceBeforePt)))
	f.Put("spaceBelow", docsreq.Points(value(after, b.cfg.Defaults.SpaceAfterPt)))
	f.Put("lineSpacing", docsreq.Percent(value(line, b.cfg.Defaults.LineSpacing)))
	f.Put("indentStart", docsreq.Points(0))
	f.Put("indentFirstLine", docsreq.Points(0))
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
	if font := b.cfg.Defaults.Font; font != "" {
		f.Put("weightedFontFamily", map[string]any{"fontFamily": font})
	}
	f.Put("fontSize", docsreq.Points(value(m.sizePt, b.cfg.Defaults.SizePt)))
	f.Put("bold", m.bold)
	f.Put("italic", false)
	f.Put("underline", false)
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
