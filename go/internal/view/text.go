// Package view is the document as text the AI reads, and the same document as
// a tree with character indexes on it. Both are projections of one docs.Document
// and neither is the other's summary: the text is what a reader reads, and the
// structure is what a later milestone places a proposal into.
//
// The projection is deterministic. The same document gives the same bytes, so a
// golden file is a specification rather than a snapshot. Nothing here judges
// anything: a pending suggestion is marked as pending, a comment range is
// marked as a range, and what any of it means is the skill's.
package view

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gdoc/internal/docs"
)

// The markup the projection adds. Every one of these is two characters, so a
// literal one in the document's own text is escaped with a backslash and a
// marker in the output is always gdoc's.
const (
	openInsertion = "{+"
	shutInsertion = "+}"
	openDeletion  = "{-"
	shutDeletion  = "-}"
	openComment   = "[["
	shutComment   = "]]"
)

// escapePairs is what escaping looks for. It is the markup above, and the list
// has to stay in step with it.
var escapePairs = []string{openInsertion, openDeletion, shutInsertion, shutDeletion, openComment, shutComment}

// placeholders is what a run with no text prints as. Reading the picture itself
// is docs/backlog/read-pictures-and-drawings.md; printing a placeholder and
// warning is the honest thing to do until then.
var placeholders = map[string]struct{ mark, noun string }{
	docs.KindImage:    {"[image]", "an image"},
	docs.KindDrawing:  {"[drawing]", "a drawing"},
	docs.KindEquation: {"[equation]", "an equation"},
	docs.KindObject:   {"[object]", "an embedded object"},
}

// Text projects the document into the one string the AI reads, and returns the
// warnings the projection raised: a placeholder printed in place of content,
// and a comment range it could not place in the text.
func Text(d *docs.Document) (string, []string) {
	e := &emitter{seen: map[string]bool{}}
	tabIDs := map[string]bool{}
	for _, tab := range d.Tabs {
		tabIDs[tab.ID] = true
	}
	for _, tab := range d.Tabs {
		if d.MultiTab() {
			e.chunk(fmt.Sprintf("<!-- tab %s: %s -->", tab.ID, tab.Title))
		}
		e.startTab(tab, d.CommentRanges)
		e.blocks(tab.Body)
	}
	e.unplaced(d.CommentRanges, tabIDs)
	e.appendFootnotes(d)
	return strings.Join(e.chunks, "\n\n") + "\n", e.warnings
}

// Structure is the tree as the `structure` field of the read: every paragraph
// and every run with the character indexes a write needs to address them. The
// JSON tags are docs's own, so there is one name per field in the whole binary.
func Structure(d *docs.Document) any {
	return struct {
		Tabs []docs.Tab `json:"tabs"`
	}{Tabs: d.Tabs}
}

// event is one end of a comment range, at the character index it sits at.
type event struct {
	index int
	open  bool
	id    string
}

func (ev event) marker() string {
	if ev.open {
		return openComment + "c:" + ev.id + shutComment
	}
	return openComment + "/c" + shutComment
}

// note is one footnote, in the order it was first referenced.
type note struct{ number, id string }

// emitter builds the text one chunk at a time. A chunk is a paragraph, a table
// or a tab line, and the chunks are joined with one blank line between them.
//
// out is the buffer being written to. It is swapped while a table cell is
// collected, because a cell's text is joined into a row rather than appended to
// the body, and the markers inside it have to travel with it.
type emitter struct {
	chunks   []string
	out      *strings.Builder
	events   []event
	cur      int
	seen     map[string]bool
	order    []note
	warnings []string
}

func (e *emitter) chunk(s string) {
	if s == "" {
		return
	}
	e.chunks = append(e.chunks, s)
}

// capture runs f with a fresh buffer and hands back what it wrote. The event
// cursor is not reset, so a marker inside a table cell is still emitted in
// index order across the whole tab.
func (e *emitter) capture(f func()) string {
	old := e.out
	var b strings.Builder
	e.out = &b
	f()
	e.out = old
	return b.String()
}

// startTab arms the comment markers for one tab. A range whose ends are not in
// this tab's text is dropped with a warning rather than half-printed: an
// opening marker with no close is worse than a sentence saying so.
func (e *emitter) startTab(t docs.Tab, ranges map[string]docs.Range) {
	e.events, e.cur = nil, 0
	spans := intervals(t.Body, nil)
	for _, id := range sortedIDs(ranges) {
		r := ranges[id]
		if r.Tab != t.ID {
			continue
		}
		e.seen[id] = true
		if r.Start == r.End {
			// A range covering no characters marks nothing. Arming it anyway
			// puts the close before the open at the same index, and at the end
			// of the last run the closes-only drain emits the close and leaves
			// the open behind: either way the text carries a marker that is not
			// a pair, which is what the escaping exists to make impossible.
			e.warnings = append(e.warnings,
				fmt.Sprintf("comment %s: its range %d..%d covers no text, so it is not marked", id, r.Start, r.End))
			continue
		}
		if !covers(spans, r.Start) || !covers(spans, r.End) {
			e.warnings = append(e.warnings,
				fmt.Sprintf("comment %s: its range %d..%d is not in the text of tab %s, so it is not marked", id, r.Start, r.End, r.Tab))
			continue
		}
		e.events = append(e.events, event{index: r.Start, open: true, id: id})
		e.events = append(e.events, event{index: r.End, id: id})
	}
	sort.Slice(e.events, func(i, j int) bool {
		a, b := e.events[i], e.events[j]
		if a.index != b.index {
			return a.index < b.index
		}
		if a.open != b.open {
			// A close comes before an open at the same index, so ranges that
			// touch do not read as ranges that nest.
			return b.open
		}
		return a.id < b.id
	})
}

// unplaced warns about a range naming a tab the document does not have. Every
// other range was either marked or warned about while its tab was walked.
func (e *emitter) unplaced(ranges map[string]docs.Range, tabs map[string]bool) {
	for _, id := range sortedIDs(ranges) {
		if e.seen[id] || tabs[ranges[id].Tab] {
			continue
		}
		e.warnings = append(e.warnings,
			fmt.Sprintf("comment %s: its range names tab %s, which this document does not have", id, ranges[id].Tab))
	}
}

// drain writes the markers up to and including index upTo. With closesOnly the
// walk stops at the first opening marker, so a range that opens where the
// previous one closed opens beside the text it covers.
func (e *emitter) drain(upTo int, closesOnly bool) {
	for e.cur < len(e.events) {
		ev := e.events[e.cur]
		if ev.index > upTo || (closesOnly && ev.open) {
			return
		}
		e.out.WriteString(ev.marker())
		e.cur++
	}
}

func (e *emitter) blocks(bs []docs.Block) {
	for _, b := range bs {
		switch {
		case b.Paragraph != nil:
			e.paragraph(b.Paragraph)
		case len(b.Table) > 0:
			e.table(b.Table)
		}
	}
}

func (e *emitter) paragraph(p *docs.Paragraph) {
	text := e.capture(func() { e.runs(p.Runs) })
	if text == "" {
		// An empty paragraph is the document's blank line, and the chunks are
		// already separated by one.
		return
	}
	e.chunk(prefix(p) + text)
}

// prefix is what a paragraph's style puts in front of its text. A heading wins
// over a bullet: a numbered list item styled as a heading is a heading, and
// printing both markers would say it is two things.
func prefix(p *docs.Paragraph) string {
	if n := headingLevel(p.Style); n > 0 {
		return strings.Repeat("#", n) + " "
	}
	if p.Bullet != nil {
		return strings.Repeat("  ", p.Bullet.NestingLevel) + "- "
	}
	return ""
}

// headingLevel reads HEADING_1 through HEADING_6. TITLE and SUBTITLE are plain
// paragraphs on purpose: the title is on the envelope, and a document whose
// first line is its own title reads twice.
func headingLevel(style string) int {
	rest, ok := strings.CutPrefix(style, "HEADING_")
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < 1 || n > 6 {
		return 0
	}
	return n
}

// table is a pipe table: the rows in reading order, a separator after the
// first. The widths are not padded, because the reader is a language model and
// the alignment would be bytes nobody reads.
func (e *emitter) table(t docs.Table) {
	rows := make([]string, 0, len(t))
	width := 0
	for _, row := range t {
		cells := make([]string, 0, len(row))
		for _, c := range row {
			cells = append(cells, e.cell(c))
		}
		if len(cells) > width {
			width = len(cells)
		}
		rows = append(rows, strings.Join(cells, " | "))
	}
	if len(rows) == 0 {
		return
	}
	seps := make([]string, width)
	for i := range seps {
		seps[i] = "---"
	}
	out := append([]string{rows[0], strings.Join(seps, " | ")}, rows[1:]...)
	e.chunk(strings.Join(out, "\n"))
}

// cell is one table cell on one line. A cell holding two paragraphs, or a table
// of its own, is flattened with spaces: a newline inside a cell would end the
// row.
func (e *emitter) cell(c docs.Cell) string {
	var parts []string
	for _, b := range c.Blocks {
		if b.Paragraph != nil {
			if s := e.capture(func() { e.runs(b.Paragraph.Runs) }); s != "" {
				parts = append(parts, s)
			}
			continue
		}
		for _, row := range b.Table {
			for _, inner := range row {
				if s := e.cell(inner); s != "" {
					parts = append(parts, s)
				}
			}
		}
	}
	return strings.Join(parts, " ")
}

// runs groups the runs of a paragraph into spans. Adjacent runs carrying the
// same suggestion ids are one span, because Docs splits a suggestion wherever
// the formatting changes and two markers around one suggested phrase would read
// as two suggestions.
func (e *emitter) runs(rs []docs.Run) {
	for i := 0; i < len(rs); {
		j := i + 1
		for j < len(rs) && sameSpan(rs[i], rs[j]) {
			j++
		}
		e.span(rs[i:j])
		i = j
	}
}

func sameSpan(a, b docs.Run) bool {
	return equal(a.InsertionIDs, b.InsertionIDs) && equal(a.DeletionIDs, b.DeletionIDs)
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (e *emitter) span(rs []docs.Run) {
	ins, del := rs[0].InsertionIDs, rs[0].DeletionIDs
	switch {
	case len(del) > 0 && len(ins) > 0:
		// Both lists on one run is text that was suggested and then suggested
		// away. Docs shows it as a deletion followed by an insertion, so that
		// is what prints. The second copy carries no comment markers: the
		// characters exist once, and marking them twice would open a range the
		// document does not have.
		e.wrap(openDeletion, shutDeletion, del, func() { e.body(rs) })
		e.wrap(openInsertion, shutInsertion, ins, func() { e.plain(rs) })
	case len(ins) > 0:
		e.wrap(openInsertion, shutInsertion, ins, func() { e.body(rs) })
	case len(del) > 0:
		e.wrap(openDeletion, shutDeletion, del, func() { e.body(rs) })
	default:
		e.body(rs)
	}
}

func (e *emitter) wrap(open, shut string, ids []string, f func()) {
	e.out.WriteString(open)
	f()
	e.out.WriteString(shut + "[s:" + strings.Join(ids, ",") + "]")
}

// body writes runs with their comment markers and their placeholders, tracking
// the character index as it goes.
func (e *emitter) body(rs []docs.Run) {
	for _, r := range rs {
		switch r.Kind {
		case docs.KindText:
			e.writeText(r.Text, r.StartIndex)
		case docs.KindFootnoteRef:
			e.drain(r.StartIndex, false)
			e.out.WriteString("[^" + e.footnote(r) + "]")
			e.drain(r.EndIndex, true)
		default:
			e.drain(r.StartIndex, false)
			p, ok := placeholders[r.Kind]
			if !ok {
				p = placeholders[docs.KindObject]
			}
			e.out.WriteString(p.mark)
			e.warnings = append(e.warnings,
				fmt.Sprintf("%s at index %d is printed as %s: its content is not read", p.noun, r.StartIndex, p.mark))
			e.drain(r.EndIndex, true)
		}
	}
}

// plain writes the text of runs with nothing tracked: no comment markers, no
// index. It is the second copy of a run that was both inserted and deleted.
func (e *emitter) plain(rs []docs.Run) {
	for _, r := range rs {
		if r.Kind == docs.KindText {
			e.writeText(r.Text, -1)
		}
	}
}

// footnote records a reference and returns the number to print. A footnote
// referenced twice is one entry under the body, which is what the document
// shows.
func (e *emitter) footnote(r docs.Run) string {
	// The number already recorded for this footnote, so a second reference
	// prints what the first one did. Counting again would number the same note
	// twice over when Docs left the number out of the run.
	for _, n := range e.order {
		if n.id == r.FootnoteID {
			return n.number
		}
	}
	n := r.Text
	if n == "" {
		n = strconv.Itoa(len(e.order) + 1)
	}
	e.order = append(e.order, note{number: n, id: r.FootnoteID})
	return n
}

// appendFootnotes puts the footnote texts under the body, after a rule, in the
// order they were referenced.
func (e *emitter) appendFootnotes(d *docs.Document) {
	if len(e.order) == 0 {
		return
	}
	e.chunk("---")
	for _, n := range e.order {
		// A footnote's paragraphs are joined with newlines, and a chunk is one
		// line, so they become spaces. Dropping them instead glued the last word
		// of one paragraph to the first word of the next.
		text := strings.ReplaceAll(d.Footnotes[n.id], "\n", " ")
		e.chunk("[^" + n.number + "]: " + e.capture(func() { e.writeText(text, -1) }))
	}
}

// writeText writes one run's text, escaping the markup the document happens to
// contain and emitting the comment markers that fall inside it. start is the
// character index the text begins at, or -1 for text that is a second copy and
// occupies no index of its own.
//
// The index counts UTF-16 code units, because that is what a Docs index is. A
// typographic quote is one unit and three bytes, so counting bytes would put
// every marker after it in the wrong place.
func (e *emitter) writeText(s string, start int) {
	rs := []rune(s)
	idx := start
	track := start >= 0
	for i := 0; i < len(rs); i++ {
		if track {
			e.drain(idx, false)
		}
		if i+1 < len(rs) && isEscapePair(rs[i], rs[i+1]) {
			// The backslash goes in and the loop moves on by one rune, not two.
			// Two literals can share a character: "{-}" is "{-" and then "-}",
			// and consuming both halves of the first pair walks straight past
			// the second, leaving "-}" in the text as a closing marker the
			// document never had.
			e.out.WriteString("\\")
			e.out.WriteRune(rs[i])
			idx += utf16Len(rs[i])
			continue
		}
		if rs[i] != '\n' {
			// The newline is where a paragraph ends, and the chunks carry that.
			e.out.WriteRune(rs[i])
		}
		idx += utf16Len(rs[i])
	}
	if track {
		e.drain(idx, true)
	}
}

func isEscapePair(a, b rune) bool {
	pair := string([]rune{a, b})
	for _, p := range escapePairs {
		if pair == p {
			return true
		}
	}
	return false
}

func utf16Len(r rune) int {
	if r > 0xFFFF {
		return 2
	}
	return 1
}

// intervals is every text position a comment range can be placed at: the closed
// index range of every run in the body, tables included.
func intervals(bs []docs.Block, out [][2]int) [][2]int {
	for _, b := range bs {
		if b.Paragraph != nil {
			for _, r := range b.Paragraph.Runs {
				out = append(out, [2]int{r.StartIndex, r.EndIndex})
			}
			continue
		}
		for _, row := range b.Table {
			for _, c := range row {
				out = intervals(c.Blocks, out)
			}
		}
	}
	return out
}

func covers(spans [][2]int, i int) bool {
	for _, s := range spans {
		if i >= s[0] && i <= s[1] {
			return true
		}
	}
	return false
}

func sortedIDs(ranges map[string]docs.Range) []string {
	ids := make([]string, 0, len(ranges))
	for id := range ranges {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
