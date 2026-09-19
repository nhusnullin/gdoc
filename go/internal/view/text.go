// This file is the projection itself: the markup it adds, the escaping that
// keeps a marker gdoc's own, the walk over a tab's blocks, and the tree
// --structure prints. doc.go holds the package comment.

package view

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"

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

// Options is what a caller may ask this projection to do differently. The zero
// value is the read, which is why Text is Project with nothing set.
//
// Picture is what a picture object is written as, by object id. A non-empty
// answer is written into the text exactly as it comes back, so a caller that
// has the bytes on disk puts its own link there; an empty one leaves the
// placeholder and its warning where they were. Skip drops a block of a tab's
// body before it is projected, which is how a caller removes a span it
// recognised. ChipTargets adds the address a chip points at behind its label.
//
// All three are the export's, and none of them is the read's: read prints what
// the document holds and nothing that is not in it, and a file in the hub
// carries a picture that has a file beside it and a chip's target a later
// session has to resolve. TestProjectSkipsAndNamesPictures and
// TestAChipCarriesItsTargetWhenAskedFor are the pins.
type Options struct {
	Picture     func(objectID string) string
	Skip        func(docs.Block) bool
	ChipTargets bool
}

// Text projects the document into the one string the AI reads, and returns the
// warnings the projection raised: a placeholder printed in place of content,
// and a comment range it could not place in the text.
func Text(d *docs.Document) (string, []string) {
	return Project(d, Options{})
}

// Project is Text with the options above. One emitter, so the read and the
// export cannot drift into two escapings of one document.
func Project(d *docs.Document, o Options) (string, []string) {
	e := &emitter{seen: map[string]bool{}, headings: headingWords(d), opts: o}
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
//
// held is the document's own last character, written but not yet in out. It is
// kept back because whether it needs a backslash depends on what comes after it,
// and what comes after it may be the next run's first character or a marker gdoc
// is about to write. prev is the last character that did reach out, and prevGdoc
// says whether gdoc wrote it: a document character on either side of one of
// gdoc's markers is escaped, because the marker itself cannot carry the
// backslash.
type emitter struct {
	chunks   []string
	out      *strings.Builder
	held     rune
	heldEnc  string
	prev     rune
	prevGdoc bool
	events   []event
	cur      int
	seen     map[string]bool
	order    []note
	// nums counts a numbered list's items, by list id and nesting level, and
	// objects is the tab being walked's floating objects. Both are the tab's
	// own: two tabs can name one list id and mean two different lists.
	nums     map[string]int
	objects  map[string]docs.Object
	headings map[string]string
	opts     Options
	// inLink says the words being written are a link's. One bracket is markup
	// there and nowhere else, because the words end at the first one a reader
	// meets.
	inLink   bool
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
//
// The held character and what sits beside it belong to the buffer, so they are
// swapped with it and the held one is released before the buffer is handed back:
// a chunk ends where it ends, and the first character of the next one is not
// what the last character of this one is escaped against.
func (e *emitter) capture(f func()) string {
	oldOut, oldHeld, oldEnc, oldPrev, oldGdoc := e.out, e.held, e.heldEnc, e.prev, e.prevGdoc
	var b strings.Builder
	e.out, e.held, e.heldEnc, e.prev, e.prevGdoc = &b, 0, "", 0, false
	f()
	e.release(0)
	e.out, e.held, e.heldEnc, e.prev, e.prevGdoc = oldOut, oldHeld, oldEnc, oldPrev, oldGdoc
	return b.String()
}

// write puts gdoc's own markup into the text: a marker, a placeholder, a link's
// brackets. The held document character is released first, escaped when it and
// the first character of s would read as one of gdoc's pairs. The document's
// character is the one that takes the backslash, because a backslash in front of
// gdoc's own marker would hand that marker to the document.
func (e *emitter) write(s string) {
	if s == "" {
		return
	}
	rs := []rune(s)
	e.release(rs[0])
	e.out.WriteString(s)
	e.prev, e.prevGdoc = rs[len(rs)-1], true
}

// writeDoc holds one of the document's own characters back until what follows it
// is known. enc is what the character is written as when nothing after it needs
// it escaped, which is escapeAt's answer.
func (e *emitter) writeDoc(r rune, enc string) {
	e.release(r)
	if e.inLink && (r == '[' || r == ']') {
		// Inside a link's words one bracket is markup of its own: the words end
		// at the first "]" a reader meets, so a "]" the document holds would cut
		// the link short and leave its target standing in the text as prose.
		enc = escape(enc)
	}
	if e.prevGdoc && isEscapePair(e.prev, r) {
		// The character sits against the last character of a marker gdoc has
		// just written, and the two would read as one of gdoc's pairs. This is
		// the same ambiguity from the other side, and the answer is the same:
		// the document's character carries the backslash.
		enc = escape(enc)
	}
	e.held, e.heldEnc = r, enc
}

// release writes the held character, escaping it when it and next would read as
// one of gdoc's pairs. next is 0 where nothing follows, which is the end of a
// chunk.
func (e *emitter) release(next rune) {
	if e.held == 0 {
		return
	}
	enc := e.heldEnc
	if isEscapePair(e.held, next) {
		enc = escape(enc)
	}
	e.out.WriteString(enc)
	e.prev, e.prevGdoc = e.held, false
	e.held, e.heldEnc = 0, ""
}

// escape puts the backslash in front of what one character was written as, or
// leaves it alone when escapeAt has already put one there: a character escaped
// twice reads as a backslash the document held.
func escape(enc string) string {
	if strings.HasPrefix(enc, "\\") {
		return enc
	}
	return "\\" + enc
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
	// The numbering and the floating objects are the tab's own, for the reason
	// the arming below is: a list id names one list per tab, and an object id
	// names one object per tab.
	e.nums, e.objects = map[string]int{}, t.Positioned
	cut := e.cut(t.Body)
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
		if overlaps(cut, r) {
			// The words this range covers are in a block Skip takes out, so
			// there is nothing in the file for it to mark. Arming it anyway
			// would emit both markers at the first surviving text, saying a
			// comment covers words the file does not contain, and an open in
			// the cut with its close outside it would overstate the span.
			// Neither is a fact the document holds, so the range is named
			// instead.
			e.warnings = append(e.warnings, strippedWarning(id, r))
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

// cutSpan is the character range one block Skip takes out occupies. It is a
// span rather than a block because what the arming needs to know is whether a
// comment sits inside the words that are going away.
type cutSpan struct{ first, last int }

// cut is every span of this tab the caller's Skip drops, tables and contents
// lists walked into, because blocks asks Skip there too. Nothing is dropped
// when no Skip was given, which is the read.
func (e *emitter) cut(bs []docs.Block) []cutSpan {
	if e.opts.Skip == nil {
		return nil
	}
	var out []cutSpan
	for _, b := range bs {
		if e.opts.Skip(b) {
			if first, last, ok := indexes([]docs.Block{b}); ok {
				out = append(out, cutSpan{first, last})
			}
			continue
		}
		switch {
		case b.Table != nil:
			for _, row := range b.Table.Rows {
				for _, c := range row {
					out = append(out, e.cut(c.Blocks)...)
				}
			}
		case b.TOC != nil:
			out = append(out, e.cut(b.TOC.Blocks)...)
		}
	}
	return out
}

// indexes is the lowest and highest character index these blocks reach, tables
// and contents lists walked into. ok is false where nothing inside them
// carries an index, which is a block no comment can sit in.
func indexes(bs []docs.Block) (first, last int, ok bool) {
	note := func(f, l int) {
		if !ok || f < first {
			first = f
		}
		if !ok || l > last {
			last = l
		}
		ok = true
	}
	for _, b := range bs {
		switch {
		case b.Paragraph != nil:
			note(b.Paragraph.StartIndex, b.Paragraph.EndIndex)
		case b.Table != nil:
			note(b.Table.StartIndex, b.Table.StartIndex)
			for _, row := range b.Table.Rows {
				for _, c := range row {
					if f, l, got := indexes(c.Blocks); got {
						note(f, l)
					}
				}
			}
		case b.TOC != nil:
			if f, l, got := indexes(b.TOC.Blocks); got {
				note(f, l)
			}
		}
	}
	return first, last, ok
}

// overlaps says whether r shares a character with any span that is going away.
// A range that ends where a cut span begins is the first character after it and
// does not overlap, which is why the ends are exclusive on both sides.
func overlaps(cut []cutSpan, r docs.Range) bool {
	for _, c := range cut {
		if r.Start < c.last && c.first < r.End {
			return true
		}
	}
	return false
}

// strippedWarning names a range whose words this projection took out. It is the
// same shape as unplacedWarning and for the same reason: the file carries no
// marker for the comment, and a session that cannot see why would go looking
// for words that are not there.
func strippedWarning(id string, r docs.Range) string {
	return fmt.Sprintf("comment %s: its range %d..%d is in a block this projection took out, so it is not marked", id, r.Start, r.End)
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
		e.write(ev.marker())
		e.cur++
	}
}

func (e *emitter) blocks(bs []docs.Block) {
	for _, b := range bs {
		if e.opts.Skip != nil && e.opts.Skip(b) {
			continue
		}
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
	// The prefix is taken even when the text is empty, so an item of a numbered
	// list that holds no text still counts: the numbers then say what the
	// document shows rather than closing the gap over an item nobody typed into.
	prefix := e.prefix(p)
	if text != "" {
		// An empty paragraph is the document's blank line, and the chunks are
		// already separated by one.
		e.chunk(prefix + text)
	}
	e.floating(p)
}

// prefix is what a paragraph's style puts in front of its text. A heading wins
// over a bullet: a numbered list item styled as a heading is a heading, and
// printing both markers would say it is two things. A heading takes no number
// for the same reason, and does not advance the count.
//
// A numbered item is indented by three spaces per level and a bullet by two,
// which is the width of the marker each of them sits under.
func (e *emitter) prefix(p *docs.Paragraph) string {
	if p.Bullet != nil {
		e.enter(p.Bullet)
	}
	if n := headingLevel(p.Style); n > 0 {
		return strings.Repeat("#", n) + " "
	}
	if p.Bullet == nil {
		return ""
	}
	if !p.Bullet.Ordered {
		return strings.Repeat("  ", p.Bullet.NestingLevel) + "- "
	}
	return strings.Repeat("   ", p.Bullet.NestingLevel) + strconv.Itoa(e.number(p.Bullet)) + ". "
}

// number is this item's place in its own list at its own level. Counting per
// list and per level is what the list id is decoded for: two adjacent numbered
// lists read as one long list otherwise, and the second one's first item then
// says it is the fourth.
func (e *emitter) number(b *docs.Bullet) int {
	if e.nums == nil {
		e.nums = map[string]int{}
	}
	key := countKey(b)
	e.nums[key]++
	return e.nums[key]
}

// enter is the walk going down into a level of a list. Every level below the
// one it entered starts again, because Docs draws a sub-list from one each
// time the list goes into it. Carrying the count on would number the second
// sub-list 3., 4. under an item the document shows as 1., 2., which is a false
// fact about the document.
//
// It happens for every item of a list, whether or not that item takes a number
// itself. One list id can hold a bulleted level above a numbered one, which is
// what a Word multilevel list comes back as, and a heading that is also an
// item takes no number either: if only the numbered items dropped the deeper
// counts, a sub-list under either of those would carry on from the one before
// it.
func (e *emitter) enter(b *docs.Bullet) {
	for k := range e.nums {
		if id, level, ok := splitCount(k); ok && id == b.ListID && level > b.NestingLevel {
			delete(e.nums, k)
		}
	}
}

// countKey is the counter one list holds for one nesting level.
func countKey(b *docs.Bullet) string {
	return b.ListID + "\x00" + strconv.Itoa(b.NestingLevel)
}

// splitCount reads a counter's key back into the list and the level it counts.
func splitCount(key string) (string, int, bool) {
	id, rest, ok := strings.Cut(key, "\x00")
	if !ok {
		return "", 0, false
	}
	level, err := strconv.Atoi(rest)
	if err != nil {
		return "", 0, false
	}
	return id, level, true
}

// floating prints one placeholder per object anchored to this paragraph, after
// the paragraph's own text, and warns about each.
//
// The paragraph is the only position the answer gives: a floating object is laid
// out beside the text rather than in it, so it has no character index a
// placeholder could go at. The kind is named rather than the word picture,
// because a floating drawing is not one, and the object id is named because that
// is what pairs the placeholder with the bytes the docx export carries.
func (e *emitter) floating(p *docs.Paragraph) {
	for _, s := range e.floatingOf(p) {
		e.chunk(s)
	}
}

// floatingOf is what this paragraph's floating objects are written as, in
// order, with the warning for each raised as it is built.
//
// It hands the pieces back rather than writing them, because a paragraph inside
// a table cell carries them too and a chunk of its own there would land the
// placeholder under the table instead of in the cell. internal/export counts
// the objects on a cell's paragraphs, so a cell that printed nothing for one
// would name a file no line of the note points at.
func (e *emitter) floatingOf(p *docs.Paragraph) []string {
	var out []string
	for _, id := range p.Positioned {
		ph, ok := placeholders[e.objects[id].Kind]
		if !ok {
			// An id the tab does not hold, and a kind this read cannot name,
			// are the same answer: something is there and gdoc cannot say what.
			ph = placeholders[docs.KindObject]
		}
		m := "<!-- " + ph.label + ": floating, " + id + " -->"
		out = append(out, m)
		e.warnings = append(e.warnings, fmt.Sprintf(
			"%s floating beside the paragraph at index %d is printed as %s: %s", ph.noun, p.StartIndex, m, ph.why))
		// The placeholder stays even when the bytes are here, because the
		// placeholder is the one record that this picture floats beside the
		// text rather than sitting in it, and a file on its own says nothing
		// about that. So does the warning: what was lost is the position.
		if s := e.pictureOf(e.objects[id].Kind, id); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// picture is what this run is written as when it is a picture and the caller
// said what its file is called, and nothing otherwise. A drawing is a picture:
// the docx export carries its bytes like any other object's.
func (e *emitter) picture(r docs.Run) string {
	if r.Detail == nil {
		return ""
	}
	return e.pictureOf(r.Kind, r.Detail.ID)
}

// pictureOf is the same question for an object the tab holds rather than a run:
// the kind says whether it is a picture at all, and the id is what the caller
// named its file by.
func (e *emitter) pictureOf(kind, id string) string {
	if e.opts.Picture == nil || id == "" {
		return ""
	}
	if kind != docs.KindImage && kind != docs.KindDrawing {
		return ""
	}
	return e.opts.Picture(id)
}

// markOf is one placeholder with the address behind it, for a projection that
// was asked for chip targets. The read prints the label alone: the target is in
// --structure, and a reader of the text is being told somebody is there rather
// than being handed their address. A file in the hub carries it, because the
// session merging that file into a note has no second read to go back to.
func (e *emitter) markOf(r docs.Run) string {
	m := mark(r)
	if t := e.chipTarget(r); t != "" {
		return m + "(" + t + ")"
	}
	return m
}

// chipTarget is where a chip points: the rich link's address, and the person
// chip's as a mailto. A date chip points nowhere, and neither does anything
// that is not a chip.
//
// A target carrying a bracket, a parenthesis or a space is left out. The form
// is the link form, a reader takes it up to the first ")", and a target that
// cuts itself short would leave the rest of it standing in the text as prose.
func (e *emitter) chipTarget(r docs.Run) string {
	if !e.opts.ChipTargets || r.Detail == nil {
		return ""
	}
	switch r.Kind {
	case docs.KindRichLink:
		return plainTarget(r.Detail.URI)
	case docs.KindPerson:
		if r.Detail.Email == "" {
			return ""
		}
		return plainTarget("mailto:" + r.Detail.Email)
	}
	return ""
}

// Destination is a target written so that a reader takes all of it. The bare
// form ends at the first ")", so a target carrying one, a "(" or a space goes in
// the angle bracket form instead, and an angle bracket inside that form is
// escaped, because an unescaped one would end it the same way.
//
// It is exported because internal/export writes the same parentheses around a
// picture's address, and one rule in two places is two chances for one of them
// to be wrong. A chip's address does not come here: chipTarget drops a target it
// cannot write bare, because a chip prints a label rather than the words the
// address belongs to.
func Destination(target string) string {
	if !strings.ContainsAny(target, " ()<>") {
		return target
	}
	return "<" + angleBrackets.Replace(target) + ">"
}

var angleBrackets = strings.NewReplacer("<", `\<`, ">", `\>`)

func plainTarget(s string) string {
	if s == "" || strings.ContainsAny(s, "()[] \t\n") {
		return ""
	}
	return s
}

// headingWords is every heading's own words, by the id a link to it names. It is
// read before the walk starts, so a link to a heading further down the document,
// or in another tab, resolves to the words rather than to the id.
func headingWords(d *docs.Document) map[string]string {
	out := map[string]string{}
	var walk func(bs []docs.Block)
	walk = func(bs []docs.Block) {
		for _, b := range bs {
			switch {
			case b.Paragraph != nil:
				if b.Paragraph.HeadingID == "" {
					continue
				}
				var words strings.Builder
				for _, r := range b.Paragraph.Runs {
					if r.Kind == docs.KindText {
						words.WriteString(r.Text)
					}
				}
				out[b.Paragraph.HeadingID] = strings.TrimSpace(words.String())
			case b.Table != nil:
				for _, row := range b.Table.Rows {
					for _, c := range row {
						walk(c.Blocks)
					}
				}
			case b.TOC != nil:
				walk(b.TOC.Blocks)
			}
		}
	}
	for _, t := range d.Tabs {
		walk(t.Body)
	}
	return out
}

// Anchor is a heading's words as the id a link to it names. It is goldmark's
// own heading id rule, asked of goldmark rather than written out again here,
// because the ids internal/body collects are the ids goldmark made: a second
// copy of the rule is the copy that drifts, and a link whose target is one
// character off is a link `publish` prints as plain text.
//
// The rule is not Slug's. goldmark keeps ASCII letters and digits, turns each
// space, hyphen and underscore into one hyphen, and drops everything else,
// accented letters included. So "Risk & Control" is "risk--control" and
// "Résumé" is "rsum". Two rules that look alike and are not is why this one is
// asked rather than copied. TestTheAnchorViewWritesIsTheIDBodyCollects, which
// asks goldmark for the answer rather than stating one, is the pin, and
// TestAHeadingLinkRoundTripsToItsSlug is the same rule seen from the file.
//
// A fresh generator each call, so the answer is these words' own id and never
// carries the "-1" goldmark adds to a heading it has already seen. Two headings
// with the same words share an anchor and a link to the second lands on the
// first, which is what markdown itself does with them.
func Anchor(s string) string {
	return string(parser.NewContext().IDs().Generate([]byte(s), ast.KindHeading))
}

// Slug is a heading's words as a file name: lower case, every run of
// characters that are neither letters nor digits one hyphen, and nothing else.
// It keeps a letter Go calls a letter, so a tab titled "Résumé" is a file
// called "résumé" rather than a file called nothing.
//
// It is exported because internal/export names a further tab's file by its
// title and a second copy of the rule there would be the one that drifted.
// TestTabSlugsCollide is the pin.
//
// It is not the rule a heading id follows. That rule is goldmark's, and it is
// Anchor.
func Slug(s string) string {
	var b strings.Builder
	gap := false
	for _, r := range strings.ToLower(s) {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			gap = true
			continue
		}
		if gap && b.Len() > 0 {
			b.WriteByte('-')
		}
		gap = false
		b.WriteRune(r)
	}
	return b.String()
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
//
// A floating object anchored to one of the cell's paragraphs is written here
// too, after that paragraph's text, for the reason floatingOf gives.
func (e *emitter) cell(c docs.Cell) string {
	var parts []string
	for _, b := range c.Blocks {
		if b.Paragraph != nil {
			if s := e.capture(func() { e.runs(b.Paragraph.Runs) }); s != "" {
				parts = append(parts, s)
			}
			parts = append(parts, e.floatingOf(b.Paragraph)...)
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
	e.write(open)
	f()
	e.write(shut + "[s:" + strings.Join(ids, ",") + "]")
}

// body writes runs with their comment markers and their placeholders, tracking
// the character index as it goes. Runs that point at the same target are one
// link: Docs splits a run wherever the formatting changes, and two forms around
// one linked phrase would read as two links.
func (e *emitter) body(rs []docs.Run) {
	for i := 0; i < len(rs); {
		l := textLink(rs[i])
		if l == nil {
			e.one(rs[i])
			i++
			continue
		}
		j := i + 1
		for j < len(rs) && sameLink(textLink(rs[j]), l) {
			j++
		}
		e.linked(rs[i:j], l)
		i = j
	}
}

// one writes one run: its text with the comment markers that fall inside it, or
// the placeholder it prints as and the warning that goes with it.
func (e *emitter) one(r docs.Run) {
	switch r.Kind {
	case docs.KindText:
		e.writeText(r.Text, r.StartIndex)
	case docs.KindFootnoteRef:
		e.drain(r.StartIndex, false)
		e.write("[^" + e.footnote(r) + "]")
		e.drain(r.EndIndex, true)
	default:
		e.drain(r.StartIndex, false)
		if md := e.picture(r); md != "" {
			// The picture has a file beside the note, so nothing was lost and
			// there is nothing to warn about. It is written where it stands: a
			// picture in a paragraph of its own is a line of its own, which is
			// where publish puts one, and a picture in the middle of a sentence
			// stays in the middle of that sentence.
			e.write(md)
			e.drain(r.EndIndex, true)
			return
		}
		p, ok := placeholders[r.Kind]
		if !ok {
			p = placeholders[docs.KindObject]
		}
		m := e.markOf(r)
		e.write(m)
		e.warnings = append(e.warnings,
			fmt.Sprintf("%s at index %d is printed as %s: %s", p.noun, r.StartIndex, m, p.why))
		e.drain(r.EndIndex, true)
	}
}

// linked writes runs that point somewhere as [words](target).
//
// The markers that open where the link opens are drained before the bracket, so
// a comment anchored at the same character does not end up behind it: gdoc's "["
// against gdoc's "[[" is a pair neither of them can be escaped out of.
//
// A link this projection has no target for prints its words alone. A target is
// where the words point and the words are the document's own either way, so the
// text stays the document's and nothing is invented for the parentheses.
func (e *emitter) linked(rs []docs.Run, l *docs.Link) {
	target := e.target(l)
	if target == "" {
		for _, r := range rs {
			e.one(r)
		}
		return
	}
	e.drain(rs[0].StartIndex, false)
	e.write("[")
	e.inLink = true
	for _, r := range rs {
		e.one(r)
	}
	e.inLink = false
	e.write("](" + Destination(target) + ")")
}

// textLink is the target of a run of the document's own text, or nothing. Only
// text carries one: a chip has a target of its own, which reaches the structure
// view, and printing it here would put two targets on one placeholder.
func textLink(r docs.Run) *docs.Link {
	if r.Kind != docs.KindText {
		return nil
	}
	return r.Link
}

func sameLink(a, b *docs.Link) bool { return a != nil && b != nil && *a == *b }

// target is what the parentheses hold: the address of a link out of the
// document, and "#" with the heading's own words slugged for a link to a heading
// inside it, which is the form internal/body writes a note's own links in, so a
// link read out of a document and a link written into one are the same string.
//
// A heading the document does not hold, and a bookmark, are named by their id:
// there are no words to slug, and the id is the one fact there is. A link naming
// only a tab has no target at all, because a tab is a document rather than a
// place in this text.
func (e *emitter) target(l *docs.Link) string {
	switch {
	case l.URL != "":
		return l.URL
	case l.HeadingID != "":
		if words := e.headings[l.HeadingID]; words != "" {
			return "#" + Anchor(words)
		}
		return "#" + l.HeadingID
	case l.BookmarkID != "":
		return "#" + l.BookmarkID
	}
	return ""
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
			e.write("[^" + e.footnote(r) + "]")
			continue
		}
		e.write(e.markOf(r))
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
		if rs[i] != '\n' {
			// The newline is where a paragraph ends, and the chunks carry that,
			// so it is not written. It is not what anything is escaped against
			// either: a character that is not in the text cannot separate the
			// two halves of a pair from each other.
			e.writeDoc(rs[i], escapeAt(rs, i))
		}
		idx += utf16Len(rs[i])
	}
	if track {
		e.drain(idx, true)
	}
}

// escapeAt is what rune i of rs is written as, as far as the window rs shows.
// It is one function because two callers need it: the document's own text, which
// is written a rune at a time with the comment markers drained between them, and
// a chip's label, which is written as a string. Two copies of this would be two
// rules, and the one that drifted would hand a marker to the wrong side without
// anybody noticing.
//
// What it cannot see is what follows the window: the next run's first character,
// or a marker gdoc is about to write. The emitter holds the last character back
// until it knows, and release puts the backslash in then, so this answer is the
// one for a character the window has something after.
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
// A newline becomes a space rather than nothing. The projection drops one in the
// document's own text because that text is written in chunks and the chunking
// carries the paragraph break; a label has no chunking, so dropping it glues the
// words either side of it together.
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
