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
// marker in the output is always gdoc's. The backslash is escaped as well, so
// the reader can tell whose backslash it is: an even run of them is the
// author's text, an odd one ends in gdoc's escape.
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

// placeholders is what a run with no text of its own prints as. label is what
// goes inside the brackets, noun is what the warning calls it, and why is what
// the warning says gdoc did not read. Reading the picture itself is
// docs/backlog/read-pictures-and-drawings.md; printing a placeholder and
// warning is the honest thing to do until then.
//
// A chip's mark carries the chip's label as well, because the document shows
// that label on screen: "[person]" tells a reader there is somebody there and
// not who, which for a policy naming its owner is the fact that mattered. The
// label goes through escapeLabel on the way in, which is the document's own
// escaping with the two differences that function names.
//
// A horizontal rule prints as [rule] rather than as "---", which is what the
// footnote separator already is. Two meanings for one line is a line neither of
// them can be read from.
var placeholders = map[string]struct{ label, noun, why string }{
	docs.KindImage:          {"image", "an image", "its content is not read"},
	docs.KindDrawing:        {"drawing", "a drawing", "its content is not read"},
	docs.KindEquation:       {"equation", "an equation", "its content is not read"},
	docs.KindObject:         {"object", "an embedded object", "its content is not read"},
	docs.KindPerson:         {"person", "a person chip", "only the name it shows is read"},
	docs.KindDate:           {"date", "a date chip", "only the date it shows is read"},
	docs.KindRichLink:       {"link", "a link chip", "only the title it shows is read"},
	docs.KindAutoText:       {"auto text", "an automatic field", "its value is filled in when the document is laid out"},
	docs.KindPageBreak:      {"page break", "a page break", "it holds no text"},
	docs.KindColumnBreak:    {"column break", "a column break", "it holds no text"},
	docs.KindHorizontalRule: {"rule", "a horizontal rule", "it holds no text"},
	docs.KindUnknown:        {"unknown", "an element this read does not name", "gdoc has never seen it"},
}

// mark is what one placeholder run prints as, with the label the document shows
// after a colon when there is one. The label goes through escapeLabel, which is
// the document's own escaping with the two differences escapeLabel names: a
// calendar entry titled "[[c:X]]" would otherwise read back as a comment anchor
// gdoc never wrote, and nothing downstream could tell.
func mark(r docs.Run) string {
	p, ok := placeholders[r.Kind]
	if !ok {
		p = placeholders[docs.KindObject]
	}
	if label := detailLabel(r); label != "" {
		return "[" + p.label + ": " + escapeLabel(label) + string(labelShut)
	}
	return "[" + p.label + "]"
}

// detailLabel is the one word of a run's detail that belongs in the text: what
// the document shows for a chip, which auto text this is, and the member name
// of an element gdoc cannot name. The rest of the detail reaches the structure
// view, which is where a caller reads an email address or a link's target.
func detailLabel(r docs.Run) string {
	if r.Detail == nil {
		return ""
	}
	switch {
	case r.Detail.Label != "":
		return r.Detail.Label
	case r.Detail.Type != "":
		return r.Detail.Type
	}
	return r.Detail.Member
}

// Text projects the document into the one string the AI reads, and returns the
// warnings the projection raised: a placeholder printed in place of content,
// and a comment range it could not place in the text.
func Text(d *docs.Document) (string, []string) {
	e := &emitter{seen: map[string]bool{}}
	for _, tab := range d.Tabs {
		if d.MultiTab() {
			e.chunk(fmt.Sprintf("<!-- tab %s: %s -->", tab.ID, tab.Title))
		}
		e.startTab(d, tab)
		e.blocks(tab.Body)
	}
	e.unplaced(d.CommentRanges)
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

// startTab arms the comment markers for one tab. A range Places refuses is
// dropped with a warning rather than half-printed: an opening marker with no
// close is worse than a sentence saying so.
//
// The rule is Places's, and only the wording of the warning is decided here.
// read marks a comment's range in the text and comments prints it as a
// position, and a range one of them refuses while the other prints it is a
// false fact in whichever field the skill reads. So the condition lives in one
// place, and a condition added to it takes effect in both commands at once.
//
// The two forms of Places do two different jobs, and both are needed. The
// warning is the document's, because that is the answer comments prints and
// warning here about a range the document does place would be the drift this
// exists to stop. The arming is the walked tab's, because the markers go into
// that tab's text: two tabs sharing an id, which the decoder can hand over
// because a tab with no id at all is called t.0, would otherwise arm this one
// on the other one's indexes and end the walk with an open marker and no close.
// On a document whose tab ids are unique the two answers are the same one.
func (e *emitter) startTab(d *docs.Document, t docs.Tab) {
	e.events, e.cur = nil, 0
	for _, id := range sortedIDs(d.CommentRanges) {
		r := d.CommentRanges[id]
		if r.Tab != t.ID {
			continue
		}
		if !e.seen[id] {
			e.seen[id] = true
			if !d.Places(r) {
				e.warnings = append(e.warnings, unplacedWarning(id, r))
			}
		}
		if !t.Places(r) {
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

// unplaced warns about a range naming a tab the document does not have. A range
// whose tab is here was marked or warned about while that tab was walked, which
// is what seen records, so what is left names a tab that is not.
func (e *emitter) unplaced(ranges map[string]docs.Range) {
	for _, id := range sortedIDs(ranges) {
		if e.seen[id] {
			continue
		}
		e.warnings = append(e.warnings,
			fmt.Sprintf("comment %s: its range names tab %s, which this document does not have", id, ranges[id].Tab))
	}
}

// unplacedWarning names which half of the placement rule the range failed. It
// is wording, not a second rule: Places has already refused the range, and this
// only says why in the words the reader needs.
//
// An inverted or empty range is called out on its own, because it is the worse
// half: armed, it puts its close before its open and crosses the markers it
// passes on the way, and at the end of the last run the closes-only drain emits
// the close and leaves the open behind. Either way the text carries half a
// pair, which is what the escaping exists to make impossible. The indexes come
// out of the Docs comments key unchecked, and that decoder is loose on purpose,
// so an inverted pair is a shape it can hand over.
func unplacedWarning(id string, r docs.Range) string {
	if r.Start >= r.End {
		return fmt.Sprintf("comment %s: its range %d..%d does not end after it starts, so it is not marked", id, r.Start, r.End)
	}
	return fmt.Sprintf("comment %s: its range %d..%d is not in the text of tab %s, so it is not marked", id, r.Start, r.End, r.Tab)
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
		case b.Table != nil:
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
//
// A pipe the author typed is escaped, because an unescaped one is a column
// separator: a two-cell row holding "A | B" and "C" reads back as three columns
// under a two-column separator, so an ordinary cell value changes the table's
// shape. The escaping is done here rather than in cell, so a nested table
// flattened into a cell is escaped once by the outer row.
func (e *emitter) table(t *docs.Table) {
	rows := make([]string, 0, len(t.Rows))
	width := 0
	for _, row := range t.Rows {
		cells := make([]string, 0, len(row))
		for _, c := range row {
			cells = append(cells, escapePipes(e.cell(c)))
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
		if b.Table == nil {
			continue
		}
		for _, row := range b.Table.Rows {
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
			m := mark(r)
			e.out.WriteString(m)
			e.warnings = append(e.warnings,
				fmt.Sprintf("%s at index %d is printed as %s: %s", p.noun, r.StartIndex, m, p.why))
			e.drain(r.EndIndex, true)
		}
	}
}

// plain writes runs with nothing tracked: no comment markers, no index, and no
// warning. It is the second copy of a run that was both inserted and deleted,
// which Docs shows as a deletion followed by an insertion of the same content.
//
// A run that is not text prints its placeholder here too. Skipping it wrote an
// empty "{++}" beside the deleted copy, which says an insertion of nothing is
// pending; what is pending is the chip, and the reader has to see which one.
// The warning is not repeated, because it was raised on the first copy and the
// content is one thing in the document.
func (e *emitter) plain(rs []docs.Run) {
	for _, r := range rs {
		if r.Kind == docs.KindText {
			e.writeText(r.Text, -1)
			continue
		}
		if r.Kind == docs.KindFootnoteRef {
			// footnote returns the number the first copy already recorded, so
			// the note is listed once and both copies name it.
			e.out.WriteString("[^" + e.footnote(r) + "]")
			continue
		}
		e.out.WriteString(mark(r))
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
		e.out.WriteString(escapeAt(rs, i))
		idx += utf16Len(rs[i])
	}
	if track {
		e.drain(idx, true)
	}
}

// escapeAt is what rune i of rs is written as. It is the whole escaping rule,
// and it is one function because two callers need it: the document's own text,
// which is written a rune at a time with the comment markers drained between
// them, and a chip's label, which is written as a string. Two copies of this
// would be two rules, and the one that drifted would hand a marker to the wrong
// side without anybody noticing.
func escapeAt(rs []rune, i int) string {
	if rs[i] == '\\' {
		// The escape character is escaped too, or the encoding cannot be read
		// back: the author's own backslash in front of a marker reads as
		// gdoc's, and gdoc's in front of a literal reads as the author's.
		// Either way a marker changes hands, which is what the escaping exists
		// to prevent. An even run of backslashes is the author's text, an odd
		// one ends in the escape.
		return "\\\\"
	}
	if i+1 < len(rs) && isEscapePair(rs[i], rs[i+1]) {
		// The backslash goes in and the caller moves on by one rune, not two.
		// Two literals can share a character: "{-}" is "{-" and then "-}", and
		// consuming both halves of the first pair walks straight past the
		// second, leaving "-}" in the text as a closing marker the document
		// never had.
		return "\\" + string(rs[i])
	}
	if rs[i] == '\n' {
		// The newline is where a paragraph ends, and the chunks carry that.
		return ""
	}
	return string(rs[i])
}

// escapeLabel is escapeAt over a chip's label, which carries no document index
// of its own: it sits inside a placeholder rather than in the run of text
// around it.
//
// Two things about it are not escapeAt over the string as written.
//
// The window carries the placeholder's own closing bracket. escapeAt looks one
// rune ahead, so a string's last rune is always written bare, and a title
// ending in "]" then merges with the bracket mark puts behind it: the text
// carries a "]]" that gdoc never wrote, on a document whose author only named a
// file "Q3 plan [draft]". Reading the label in a window that ends with that
// bracket is what makes the last rune half a pair like any other.
//
// A newline becomes a space rather than nothing. escapeAt drops one because the
// document's own text is written in chunks and the chunking carries the
// paragraph break; a label has no chunking, so dropping it glues the words
// either side of it together.
func escapeLabel(s string) string {
	rs := []rune(strings.ReplaceAll(s, "\n", " ") + string(labelShut))
	var b strings.Builder
	for i := 0; i < len(rs)-1; i++ {
		b.WriteString(escapeAt(rs, i))
	}
	return b.String()
}

// labelShut is the bracket mark writes behind a label, and escapeLabel reads
// the label with it on the end. One constant, because the two have to be the
// same character for the window to mean anything.
const labelShut = ']'

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

// escapePipes puts a backslash in front of every pipe in a cell. It runs after
// the marker escaping, which has already doubled the author's own backslashes,
// so the parity a reader uses on a marker holds here too: an odd run of
// backslashes before the pipe ends in gdoc's escape.
func escapePipes(cell string) string {
	return strings.ReplaceAll(cell, "|", `\|`)
}

func sortedIDs(ranges map[string]docs.Range) []string {
	ids := make([]string, 0, len(ranges))
	for id := range ranges {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
