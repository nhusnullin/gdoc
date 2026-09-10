package restyle

// The look: what a restyle writes onto a paragraph, onto the runs inside it and
// onto a table's cells. page.go is the same job for the one request that names
// no range, and the rule both files hold to is the milestone's: a request names
// in its mask exactly what it sets.
//
// Three sentences decide everything below.
//
// The structure is read, never decided. A paragraph's own namedStyleType picks
// the house look and no request writes one, so a HEADING_1 stays a HEADING_1
// and body prose stays body prose. gdoc never infers structure from text, and a
// named style it has no look for is reported rather than guessed at.
//
// The house style is written where it states a value, and nowhere else. Stated
// means a value the file can tell apart from silence: an optional number that
// is there, or a non-empty string. A plain flag is never written, because
// absent and false are one word in house.yaml: bold, italic, keep_with_next and
// keep_lines_together are read as flags, so writing them would clear the
// emphasis an author put inside a paragraph on the strength of a value the file
// may never have stated. The same argument closes the border: the house style
// draws none under a paragraph, and naming borderBottom in a mask without
// setting it is exactly how a restyle would erase a rule the author drew.
//
// The text is never touched. None of the three kinds here can change a
// character, which is the property internal/guard's allowlist holds and this
// package must not test the edge of.

import (
	"sort"

	"gdoc/internal/docs"
	"gdoc/internal/docsreq"
	"gdoc/internal/house"
)

// The cell look, in points. house.yaml spells out the three front-matter tables
// cell by cell and states nothing about a table an author wrote, so these are
// the same measured values internal/body/table.go writes into a docx for a
// table the note itself holds: a restyled table then looks like a table gdoc
// builds. They are two copies of one measurement because they are written in
// two units for two APIs, and each says here what it is for.
const (
	// The grey grid, on every edge. The master draws black on every cell,
	// which turns a four-row table into a spreadsheet.
	cellBorderPt    = 0.5
	cellBorderColor = "#C9C9C9"
	// 3pt above and below the text in a cell, so descenders do not sit on the
	// bottom border, and Word's own 5.4pt either side.
	cellPadVPt = 3.0
	cellPadHPt = 5.4
)

// Plan is what a restyle will send over one tab, and what it left alone. Every
// field is a count or a list. Nothing here says whether the result is good:
// that is read in the document, and the caller reports these as facts.
type Plan struct {
	// Requests are the styling requests in document order.
	Requests []map[string]any
	// Paragraphs, Text and Cells are how many requests of each kind were
	// built. Cells is the cells covered, which is not the number of
	// updateTableCellStyle requests: one request styles a whole table.
	Paragraphs int
	Text       int
	Cells      int
	// Bulleted is the paragraphs carrying a bullet. Their list is left exactly
	// as it is: createParagraphBullets removes the leading tabs that set a
	// bullet's nesting level, so it deletes text and is not one of the four
	// kinds the in-place level carries.
	Bulleted int
	// Tables is the tables whose column widths and row heights were not
	// touched. Both need a request kind the in-place level does not carry,
	// neither was measured, and both change a table's layout rather than its
	// look.
	Tables int
	// Unstyled names the namedStyleTypes the house style has no look for, once
	// each and sorted, with "(none)" for a paragraph that carries no named
	// style at all. Those paragraphs are left untouched.
	Unstyled []string
	// Skipped is how many paragraphs and tables were left alone because they
	// are inside the span the caller named: gdoc's own house prelude. It is
	// zero on a run that named none, which is every M7b restyle.
	Skipped int
	// skip is that span, held for the walk. It is not printed: what a caller
	// reports is how much was left alone, and the range itself is the marker's,
	// which internal/prelude already names.
	skip *Span
}

// Span is a half-open range of one tab's text, [Start, End).
//
// The one caller today is the house prelude: a run that proposed a cover, three
// front-matter tables and a legend a moment earlier hands the span it wrote them
// into, so the styling phase walks past its own words.
type Span struct {
	Start int
	End   int
}

// covers reports whether a block lying in [start, end) is inside this span.
//
// It is an overlap rather than containment, and the direction is deliberate. A
// block that is half gdoc's own words and half the author's is one no request
// here can name without writing over one of them, and leaving it as it is costs
// a paragraph its house look, while styling it would overwrite the cover line
// Nail is being asked to accept. Nothing gdoc proposes can produce one: the
// prelude ends in a page break of its own, so its last paragraph never merges
// with the author's first.
func (s *Span) covers(start, end int) bool {
	if s == nil {
		return false
	}
	return start < s.End && end > s.Start
}

// houseStyleKey maps a Docs named style to the key house.yaml states it under.
// A style outside this map has no house look, and a restyle leaves it alone
// rather than picking the nearest one.
var houseStyleKey = map[string]string{
	"NORMAL_TEXT": "normal",
	"HEADING_1":   "heading_1",
	"HEADING_2":   "heading_2",
	"HEADING_3":   "heading_3",
	"HEADING_4":   "heading_4",
	"HEADING_5":   "heading_5",
	"HEADING_6":   "heading_6",
	"TITLE":       "title",
	"SUBTITLE":    "subtitle",
}

// TabRequests is the styling one tab takes, and what it keeps. It is a pure
// function of that tab and the house style: it reads no document of its own,
// sends nothing and decides nothing about when a caller should send what it
// returns.
//
// One tab rather than a document, because a style request names a range and a
// range means nothing without saying which tab it is in. The command refuses a
// document with more than one tab before it opens the grant.
//
// It walks the tab's body and nothing else, so a paragraph inside a footnote, a
// header or a footer keeps the look it had. internal/docs flattens a footnote
// to its text and decodes no header and no footer at all, so this walk could
// not name one of their ranges even if the level carried a request for it.
// ManualSteps does not name that gap either, so a document whose body is
// restyled and whose running head is not comes back saying nothing about it.
// docs/backlog/restyle-skips-footnotes-headers-and-footers.md is the way out.
func TabRequests(t docs.Tab, cfg *house.Config) Plan {
	return TabRequestsExcept(t, cfg, nil)
}

// TabRequestsExcept is TabRequests over everything but one span.
//
// The span is gdoc's own house prelude, proposed into the document by the phase
// that ran before this one. Every paragraph internal/prelude writes states its
// look in full, because inserted text takes the look of the text it lands
// beside, and a restyle walking over those paragraphs would give each of them
// the house body look: the cover title would be 11pt prose by the time anybody
// read it. So the styling phase walks past the words the proposing phase wrote,
// and reports how many blocks it left alone.
//
// A nil span is TabRequests, unchanged, which is every restyle that proposed no
// prelude.
func TabRequestsExcept(t docs.Tab, cfg *house.Config, skip *Span) Plan {
	p := &Plan{skip: skip}
	unknown := map[string]bool{}
	p.walk(t.Body, cfg, unknown, false)
	for name := range unknown {
		p.Unstyled = append(p.Unstyled, name)
	}
	sort.Strings(p.Unstyled)
	return *p
}

// walk is the blocks of one body, in reading order. inCell says the blocks are
// inside a table cell, where the look is the house style's table text and the
// paragraph itself is left as the author set it.
func (p *Plan) walk(blocks []docs.Block, cfg *house.Config, unknown map[string]bool, inCell bool) {
	for _, b := range blocks {
		switch {
		case b.Paragraph != nil:
			// A paragraph the prelude wrote is left exactly as the prelude
			// stated it, and the walk does not go on to its runs either.
			if p.skip.covers(b.Paragraph.StartIndex, b.Paragraph.EndIndex) {
				p.Skipped++
				continue
			}
			p.paragraph(b.Paragraph, cfg, unknown, inCell)
		case b.Table != nil:
			// A table is skipped on its start alone, because internal/docs
			// decodes no end index for one. The front matter's three tables
			// are wholly inside the prelude or wholly outside it: a table
			// starting inside the span was inserted by the same batch that
			// opened it.
			if p.skip.covers(b.Table.StartIndex, b.Table.StartIndex+1) {
				p.Skipped++
				continue
			}
			p.table(b.Table, cfg, unknown)
		}
	}
}

// paragraph is one paragraph: its own look, then its runs.
//
// A paragraph inside a table cell takes the table text and nothing else. The
// body's justified alignment is not the look inside a narrow cell, and
// house.yaml states no spacing for a cell of a table the author wrote, so
// writing one would be gdoc inventing a value rather than applying the style.
func (p *Plan) paragraph(par *docs.Paragraph, cfg *house.Config, unknown map[string]bool, inCell bool) {
	if par.EndIndex <= par.StartIndex {
		return // a paragraph with no span is one no range can name
	}
	if par.Bullet != nil {
		p.Bulleted++
	}
	if inCell {
		p.add("updateTextStyle", "textStyle", rangeOf(par), cellText(cfg))
		p.Text++
		return
	}
	key, ok := houseStyleKey[par.Style]
	if !ok {
		name := par.Style
		if name == "" {
			name = "(none)"
		}
		unknown[name] = true
		return
	}
	style := cfg.Styles[key]
	p.add("updateParagraphStyle", "paragraphStyle", rangeOf(par), paragraphLook(style, key, cfg))
	p.Paragraphs++
	p.add("updateTextStyle", "textStyle", rangeOf(par), textLook(style, key, cfg))
	p.Text++
}

// table is one table: its cells' appearance in one request, and then the
// paragraphs inside those cells. A table inside a cell carries its own start
// index, so the recursion addresses it rather than the outer table.
//
// The request names tableStartLocation rather than a tableRange, which the Docs
// reference documents as applying the update "to all the cells in the table".
// One request for the table rather than one per row, and the reason is merges
// rather than economy. TableCellLocation.columnIndex is a zero-based grid
// column, and Table.columns says "It is possible for a table to be
// non-rectangular, so some rows may have a different number of cells": a row
// whose first two columns are merged carries one cell object for the pair, so a
// span of len(cells) covered the merged cell and left the row's last column
// with the look it had. internal/docs decodes no columnSpan and no column
// count, so the row's real width is not something this builder could compute,
// and a range that covers every cell needs neither.
func (p *Plan) table(t *docs.Table, cfg *house.Config, unknown map[string]bool) {
	p.Tables++
	look := cellLook()
	cells := 0
	for _, row := range t.Rows {
		cells += len(row)
	}
	if cells > 0 {
		p.Requests = append(p.Requests, map[string]any{
			"updateTableCellStyle": map[string]any{
				"tableStartLocation": map[string]any{"index": t.StartIndex},
				"tableCellStyle":     look.Set,
				"fields":             look.Mask(),
			},
		})
		p.Cells += cells
	}
	for _, cells := range t.Rows {
		for _, c := range cells {
			p.walk(c.Blocks, cfg, unknown, true)
		}
	}
}

// add appends one request over one range, with the mask the fields carry.
func (p *Plan) add(kind, key string, span map[string]any, f *fields) {
	if f.Empty() {
		return // a request setting nothing is a request that resets nothing
	}
	p.Requests = append(p.Requests, map[string]any{
		kind: map[string]any{
			"range":  span,
			key:      f.Set,
			"fields": f.Mask(),
		},
	})
}

// rangeOf is one paragraph's span. The end includes the paragraph mark, which
// is what a Docs range over a paragraph carries.
func rangeOf(par *docs.Paragraph) map[string]any {
	return map[string]any{"startIndex": par.StartIndex, "endIndex": par.EndIndex}
}

// paragraphLook is the paragraph properties the house style states for one
// named style.
//
// Body prose is the one style that does not read from styles alone. build
// writes an ordinary paragraph at the body size, the body alignment and the
// body spacing, over a Normal style that states 11pt: a restyle reading the
// style alone would make a document that does not match one gdoc built from a
// note. Everything else reads its own style, and the document defaults fill in
// what a style leaves unsaid, which is what word/styles.xml does in the docx.
func paragraphLook(s house.Style, key string, cfg *house.Config) *fields {
	f := newFields()
	align := s.Align
	before, after := s.SpaceBeforePt, s.SpaceAfterPt
	if key == "normal" {
		align = cfg.Body.Align
		body := cfg.Body
		before, after = &body.SpaceBeforePt, &body.SpaceAfterPt
	}
	if a, ok := docsreq.Alignment(align); ok {
		f.Put("alignment", a)
	}
	f.Put("spaceAbove", points(value(before, cfg.Defaults.SpaceBeforePt)))
	f.Put("spaceBelow", points(value(after, cfg.Defaults.SpaceAfterPt)))
	f.Put("lineSpacing", percent(value(s.LineSpacing, cfg.Defaults.LineSpacing)))
	if s.IndentStartPt != nil {
		f.Put("indentStart", points(*s.IndentStartPt))
	}
	return f
}

// textLook is the run properties: the face, the size and the colour. Bold and
// italic are deliberately absent, and the file comment above says why.
func textLook(s house.Style, key string, cfg *house.Config) *fields {
	f := newFields()
	font := s.Font
	if font == "" {
		font = cfg.Defaults.Font
	}
	size := value(s.SizePt, cfg.Defaults.SizePt)
	if key == "normal" {
		size = cfg.Body.SizePt
	}
	if font != "" {
		f.Put("weightedFontFamily", map[string]any{"fontFamily": font})
	}
	f.Put("fontSize", points(size))
	if c, ok := optionalColor(s.Color); ok {
		f.Put("foregroundColor", c)
	}
	return f
}

// cellText is what a paragraph inside a table cell takes: the face and the size
// house.yaml states under table_text.
func cellText(cfg *house.Config) *fields {
	f := newFields()
	font := cfg.TableText.Font
	if font == "" {
		font = cfg.Defaults.Font
	}
	if font != "" {
		f.Put("weightedFontFamily", map[string]any{"fontFamily": font})
	}
	f.Put("fontSize", points(cfg.TableText.DefaultSizePt))
	return f
}

// cellLook is the appearance every cell takes: the padding on four sides and
// the grey grid on four edges.
//
// No shading. Which row of somebody's table is a header is not something gdoc
// can read, and a background named in a mask replaces the fill the author
// chose, which on a table that carries meaning in its colours is the loss this
// milestone's mask rule exists to prevent.
func cellLook() *fields {
	f := newFields()
	for _, side := range []string{"Top", "Bottom", "Left", "Right"} {
		pad := cellPadVPt
		if side == "Left" || side == "Right" {
			pad = cellPadHPt
		}
		f.Put("padding"+side, points(pad))
	}
	border := map[string]any{
		"width":     points(cellBorderPt),
		"dashStyle": "SOLID",
	}
	if c, ok := optionalColor(cellBorderColor); ok {
		border["color"] = c
	}
	for _, side := range []string{"Top", "Bottom", "Left", "Right"} {
		f.Put("border"+side, border)
	}
	return f
}

// percent is a house line spacing as the Docs API takes it. The rule is
// docsreq's, because internal/prelude states the same spacings on the
// paragraphs it proposes.
func percent(multiplier float64) float64 { return docsreq.Percent(multiplier) }

// value is an optional house number, or the default the file states for
// everything when the style says nothing.
func value(v *float64, fallback float64) float64 {
	if v == nil {
		return fallback
	}
	return *v
}

// optionalColor is one house colour as the Docs API takes it, which is
// docsreq's rule: a value that is not a six-digit hex colour is not written,
// because writing a field the guard would carry but Docs would reject costs
// the whole batch.
func optionalColor(hex string) (map[string]any, bool) { return docsreq.Color(hex) }

// fields is a style object and the mask that names it, built together. The
// type is docsreq's, so this package and internal/prelude cannot grow two
// rules about how a mask is written.
type fields = docsreq.Fields

func newFields() *fields { return docsreq.NewFields() }
