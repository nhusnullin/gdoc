// This file is the projection itself: the markup it adds, the escaping that
// keeps a marker gdoc's own, the walk over a tab's blocks, and the tree
// --structure prints. doc.go holds the package comment.

package view

import (
	"fmt"
	"regexp"
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
// Marks writes a run's colour back as the syntax an author typed for it, and
// an empty list writes none.
//
// PageBreak is what a page break standing on its own is written as: the line
// before a paragraph the document breaks the page before, and a native
// pageBreak element that is a paragraph's only content. It is Picture's rule
// for a break: a non-empty answer replaces the placeholder and the warning
// goes with it, because the caller carried the fact rather than losing it, and
// an empty one leaves both where they were. A break inside a sentence is never
// written this way, because no line of markdown says a page turns
// mid-sentence.
//
// Rebuilt says this text becomes a file in the hub that gdoc builds again, so
// the projection names every stretch of the document's own words that the next
// build reads as syntax of gdoc's rather than as the words the document shows.
//
// All six are the export's, and none of them is the read's: read prints what
// the document holds and nothing that is not in it, and says nothing about a
// build that is not happening, while a file in the hub carries a picture that
// has a file beside it, a chip's target a later session has to resolve, and the
// mark the words were written with.
// TestProjectSkipsAndNamesPictures, TestAChipCarriesItsTargetWhenAskedFor,
// TestReadPrintsNoMark and TestReadNamesNothingTheNextBuildWouldRead are the
// pins.
type Options struct {
	Picture     func(objectID string) string
	Skip        func(docs.Block) bool
	ChipTargets bool
	Marks       []Mark
	PageBreak   string
	Rebuilt     bool
}

// Mark is one colour a caller can name, and what goes around the words that
// carry it. A mark states one of the two colours: a highlight is the run's
// background, and a mark written as a text colour is the run's own Colour. One
// stating neither matches no run.
//
// The syntax is the caller's, because this package knows a colour and the
// house style knows what an author types for it. internal/export builds the
// list out of house.yaml, the one place the palette is stated.
type Mark struct {
	Background string
	Colour     string
	Open, Shut string
	// Ambiguous are the openers the marked words themselves must not begin
	// with, because a reader of this file would take one of them for the
	// mark's own opener and drop it from the words. Explicit is what opens the
	// mark where they do: it says the colour outright, so whatever the words
	// begin with is the author's. Only a mark whose Open leaves the colour
	// unsaid carries either, which is the one highlight an author may write
	// bare.
	Ambiguous []string
	Explicit  string
}

// opener is what this group of marked words opens with: Open, or Explicit
// where the words would read as carrying the opener themselves.
func (m Mark) opener(rs []docs.Run) string {
	if m.Explicit == "" {
		return m.Open
	}
	for _, r := range rs {
		if r.Text == "" {
			continue
		}
		for _, a := range m.Ambiguous {
			if strings.HasPrefix(r.Text, a) {
				return m.Explicit
			}
		}
		break
	}
	return m.Open
}

// matches says whether this run carries this mark. The comparison is on the
// hex the read wrote, case folded: a mark is one exact colour, which is the
// rule the specification states, and a document styled by hand carries
// whatever Google rounded to.
func (m Mark) matches(r docs.Run) bool {
	if r.Kind != docs.KindText {
		// A chip or a picture prints a placeholder of gdoc's own, and a mark
		// around it would say gdoc's words were the author's.
		return false
	}
	if m.Background != "" {
		return strings.EqualFold(m.Background, r.Background)
	}
	return m.Colour != "" && strings.EqualFold(m.Colour, r.Colour)
}

// Text projects the document into the one string the AI reads, and returns the
// warnings the projection raised: a placeholder printed in place of content,
// and a comment range it could not place in the text.
func Text(d *docs.Document) (string, []string) {
	return Project(d, Options{})
}

// Project is Text with the options above. One emitter, so the read and the
// export cannot drift into two escapings of one document.
//
// It walks until the heading lines stop moving, and every walk but the last is
// thrown away. A link to a heading is written as "#" and goldmark's id for that
// heading, and goldmark takes the id from the whole line, so the anchor cannot
// be known until the line is written: a heading holding a hyperlink is
// "[words](url)" in the file and a heading holding a chip is "[person: Ann]",
// and goldmark reads both, markup and all. Each walk measures every heading's
// line, the next writes the document with the anchors those lines give, and a
// heading whose line the walk never reaches, one inside a table cell, keeps the
// words headingWords collected.
// TestAHeadingsAnchorIsTheIDOfTheLineItProjects is the pin.
//
// The walk repeats because a heading's own line can hold a link to another
// heading, and writing that link changes the line the first heading is named
// by. One more walk then names it correctly, and the walk after that is the
// same document again. A document whose headings link to no heading settles on
// the second walk, which is the cost this always paid.
// TestAHeadingLinkInsideAHeadingStillResolves is the pin.
//
// The ceiling is there because a cycle of headings linking to each other need
// never settle, and a document is not worth an unbounded number of walks. The
// last walk's text is the answer either way: every anchor in it is the id of a
// line some walk of this document wrote, so the file is readable, and only a
// link inside that cycle can point at the wrong heading.
func Project(d *docs.Document, o Options) (string, []string) {
	words := headingWords(d)
	_, _, lines := onePass(d, o, words)
	for i := 0; i < anchorPasses; i++ {
		text, warnings, next := onePass(d, o, anchors(words, lines))
		if sameLines(lines, next) {
			return text, warnings
		}
		lines = next
	}
	text, warnings, _ := onePass(d, o, anchors(words, lines))
	return text, warnings
}

// anchorPasses is how many times Project rewrites the document looking for the
// heading lines to settle, after the first walk that measures them.
const anchorPasses = 4

// sameLines says whether two walks wrote every heading to the same line.
func sameLines(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for id, line := range a {
		if b[id] != line {
			return false
		}
	}
	return true
}

// anchors is the words each heading is named by: the line it projected to
// where there is one, and the words it holds where there is not. An empty line
// is no line: the heading printed nothing, so it is not in the file at all and
// the words are still the better guess.
func anchors(words, lines map[string]string) map[string]string {
	out := make(map[string]string, len(words)+len(lines))
	for id, w := range words {
		out[id] = w
	}
	for id, line := range lines {
		if line != "" {
			out[id] = line
		}
	}
	return out
}

// onePass is one walk of the document, and the heading lines it wrote.
func onePass(d *docs.Document, o Options, headings map[string]string) (string, []string, map[string]string) {
	e := &emitter{seen: map[string]bool{}, headings: headings, lines: map[string]string{}, opts: o}
	for _, tab := range d.Tabs {
		if d.MultiTab() {
			e.chunk(fmt.Sprintf("<!-- tab %s: %s -->", tab.ID, tab.Title))
		}
		e.startTab(d, tab)
		e.blocks(tab.Body)
	}
	e.unplaced(d.CommentRanges)
	e.appendFootnotes(d)
	return strings.Join(e.chunks, "\n\n") + "\n", e.warnings, e.lines
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
	// nums counts a numbered list's items, by list id and nesting level, cols
	// records the column each of those levels' content starts at, and objects
	// is the tab being walked's floating objects. All three are the tab's own:
	// two tabs can name one list id and mean two different lists.
	nums     map[string]int
	cols     map[string]int
	objects  map[string]docs.Object
	headings map[string]string
	// lines is the line each heading projected to, by the heading id a link
	// names, which is what the anchor written for that link is taken from.
	lines map[string]string
	opts  Options
	// inLink says the words being written are a link's. One bracket is markup
	// there and nowhere else, because the words end at the first one a reader
	// meets.
	inLink bool
	// inHeading says the paragraph being written becomes a heading line in the
	// file, which is a line the next build carries no mark and no footnote
	// mark on. The question is the paragraph's rather than the words', so it
	// cannot be asked of the runs cannotCarry reads. A paragraph inside a
	// table cell is never one: a cell is flattened onto one line and takes no
	// "#" prefix, whatever style the paragraph carries.
	inHeading bool
	// breakAlone says the runs being written are one page break and the
	// newline its paragraph ends with, which is the only shape a caller can
	// write a line of markdown for. It is the runs' question rather than the
	// paragraph's, because a table cell hands its paragraph's runs over
	// directly, and runs sets it for either way in.
	breakAlone bool
	warnings   []string
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
	// The numbering, the columns it lines up with and the floating objects are
	// the tab's own, for the reason the arming below is: a list id names one
	// list per tab, and an object id names one object per tab. A column left
	// behind by an earlier tab would indent this tab's orphan sub-list against
	// a list that is not in it.
	e.nums, e.cols, e.objects = map[string]int{}, map[string]int{}, t.Positioned
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
	// The page turns before this paragraph, which the document says on the
	// paragraph's own style rather than as an element in it. That is the only
	// shape a break has in a document gdoc published, so the line goes in
	// front of the paragraph rather than nowhere: internal/docs's
	// PageBreakBefore holds what was measured.
	if p.PageBreakBefore {
		e.chunk(e.breakLine(p.StartIndex))
	}
	// The style is read before the runs are written, because what the next
	// build reads a mark on this line as depends on the "#" prefix below. A
	// paragraph is never written inside another, so one flag is enough.
	e.inHeading = headingLevel(p.Style) > 0
	text := e.capture(func() { e.runs(p.Runs) })
	e.inHeading = false
	// The prefix is taken even when the text is empty, so an item of a numbered
	// list that holds no text still counts: the numbers then say what the
	// document shows rather than closing the gap over an item nobody typed into.
	prefix := e.prefix(p)
	// The line, before the prefix, is what a link to this heading is named by.
	// It is recorded whatever the style, because headingWords names a paragraph
	// carrying a heading id whether or not it is one of the six levels.
	if p.HeadingID != "" {
		e.lines[p.HeadingID] = strings.TrimSpace(text)
	}
	if text != "" {
		// An empty paragraph is the document's blank line, and the chunks are
		// already separated by one.
		e.chunk(prefix + text)
	}
	e.floating(p)
}

// breakLine is what a page break standing on its own prints as: the line the
// caller named, or the same placeholder a native break gets with the same
// warning, so the two shapes read alike downstream.
//
// The warning is raised only where the placeholder is, for the reason a
// picture with a file beside the note raises none: the caller carried the fact
// into a line of its own, and nothing was lost to name.
func (e *emitter) breakLine(index int) string {
	if e.opts.PageBreak != "" {
		return e.opts.PageBreak
	}
	p := placeholders[docs.KindPageBreak]
	m := "[" + p.label + "]"
	e.warnings = append(e.warnings,
		fmt.Sprintf("%s at index %d is printed as %s: %s", p.noun, index, m, p.why))
	return m
}

// prefix is what a paragraph's style puts in front of its text. A heading wins
// over a bullet: a numbered list item styled as a heading is a heading, and
// printing both markers would say it is two things. A heading takes no number
// for the same reason, and does not advance the count.
//
// An item is indented to the column its parent's content starts at, which is
// the column CommonMark nests a sub-list from. The width of the item's own
// marker is the wrong measure and was the rule until it was measured: a bullet
// under a numbered item is two spaces against a parent content column of three,
// which goldmark reads as a second list beside the first rather than inside it,
// and a sub-list under the tenth item is three spaces against a column of four,
// which goldmark reads as the eleventh item of the outer list. Neither loses a
// character, and both change the shape of the document on the way home.
// TestASubListIndentsToItsParentsColumn and TestASubListUnderAWideMarkerClearsIt
// are the pins.
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
	marker := "- "
	if p.Bullet.Ordered {
		marker = strconv.Itoa(e.number(p.Bullet)) + ". "
	}
	indent := e.indent(p.Bullet, len(marker))
	e.setColumn(p.Bullet, indent+len(marker))
	return strings.Repeat(" ", indent) + marker
}

// indent is the column this item's marker starts at: the column its parent's
// content starts at, which setColumn recorded when that parent was written.
//
// A list whose first item is already nested has no parent to measure, and the
// item's own marker width per level is the only guess there is. It is what the
// projection did for every item before the columns were tracked, so the shape
// of an orphan sub-list is unchanged. TestASubListWithNoParentFallsBackToItsLevel
// is the pin.
func (e *emitter) indent(b *docs.Bullet, width int) int {
	if b.NestingLevel <= 0 {
		return 0
	}
	if col, ok := e.cols[levelKey(b.ListID, b.NestingLevel-1)]; ok {
		return col
	}
	return b.NestingLevel * width
}

// setColumn records where this item's content starts, for the items nested
// under it to line up with.
func (e *emitter) setColumn(b *docs.Bullet, col int) {
	if e.cols == nil {
		e.cols = map[string]int{}
	}
	e.cols[countKey(b)] = col
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
// The columns go with the counts. A sub-list drawn again under a later item
// lines up with that item, so a column measured under the one before it is a
// fact about a list level that has closed.
func (e *emitter) enter(b *docs.Bullet) {
	for k := range e.nums {
		if id, level, ok := splitCount(k); ok && id == b.ListID && level > b.NestingLevel {
			delete(e.nums, k)
		}
	}
	for k := range e.cols {
		if id, level, ok := splitCount(k); ok && id == b.ListID && level > b.NestingLevel {
			delete(e.cols, k)
		}
	}
}

// countKey is the counter one list holds for one nesting level.
func countKey(b *docs.Bullet) string {
	return levelKey(b.ListID, b.NestingLevel)
}

// levelKey is countKey's key for a level named on its own, so an item can ask
// for the level above its own.
func levelKey(listID string, level int) string {
	return listID + "\x00" + strconv.Itoa(level)
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

// standaloneBreak says whether these runs are one native page break and
// nothing else: the break, and the newline the paragraph ends with. That is a
// break on a line of its own rather than inside a sentence, which is the only
// shape PageBreak is written for.
//
// A break carrying a suggestion id is not one. The markers that say a
// suggestion is pending are written around the span, and the caller's line
// would go inside them: "{----}" is neither the suggestion nor the break.
func standaloneBreak(rs []docs.Run) bool {
	found := false
	for _, r := range rs {
		switch {
		case r.Kind == docs.KindPageBreak:
			if found || len(r.InsertionIDs) > 0 || len(r.DeletionIDs) > 0 {
				return false
			}
			found = true
		case r.Kind == docs.KindText && strings.TrimSpace(r.Text) == "":
		default:
			return false
		}
	}
	return found
}

// runs groups the runs of a paragraph into spans. Adjacent runs carrying the
// same suggestion ids are one span, because Docs splits a suggestion wherever
// the formatting changes and two markers around one suggested phrase would read
// as two suggestions.
func (e *emitter) runs(rs []docs.Run) {
	e.breakAlone = standaloneBreak(rs)
	e.ownSyntax(rs)
	for i := 0; i < len(rs); {
		j := i + 1
		for j < len(rs) && sameSpan(rs[i], rs[j]) {
			j++
		}
		e.span(rs[i:j])
		i = j
	}
}

// footnoteMark is the shape internal/body reads as a footnote mark.
//
// The projection writes a real note's reference as "[^n]", so the document's
// own words holding that shape are read by the next build as a mark of gdoc's.
// A label the file does not define refuses the file by line, blaming the file
// rather than the export; a label it does define grows a second reference to a
// note the document never had, and the words the author typed go with it.
//
// A backslash is not the answer, for the reason cannotCarry gives: internal/body
// hands goldmark's text through as it stands and never unescapes a punctuation
// mark, and body's own line scan reads the mark through the backslash anyway.
// docs/backlog/build-never-unescapes-a-backslash.md is where that stands. Until
// then the words go home as the document holds them and the loss is named,
// which is what a mark that cannot carry its syntax already does.
var footnoteMark = regexp.MustCompile(`\[\^[^\]\n]+\]`)

// ownSyntax names every stretch of the document's own words that the next
// build would read as syntax of gdoc's.
//
// Consecutive text runs are joined before the search, because Docs splits a run
// wherever the formatting changes, so "[^" and "1]" can be two runs of one
// sentence. A run that is not text stops the join: gdoc writes its own markup
// there, and a bracket of gdoc's is not the author's.
//
// Nothing is said where the text is not going to be built. Options.Rebuilt
// holds why.
func (e *emitter) ownSyntax(rs []docs.Run) {
	if !e.opts.Rebuilt {
		return
	}
	for i := 0; i < len(rs); {
		if rs[i].Kind != docs.KindText {
			i++
			continue
		}
		var b strings.Builder
		at, j := rs[i].StartIndex, i
		for ; j < len(rs) && rs[j].Kind == docs.KindText; j++ {
			b.WriteString(rs[j].Text)
		}
		text := b.String()
		for _, f := range e.gdocShapes(text) {
			e.warnings = append(e.warnings, fmt.Sprintf(
				"the words at index %d hold %s, which the next build reads as %s rather than as the words this document shows",
				at+units(text[:f.at]), f.what, f.noun))
		}
		i = j
	}
}

// shape is one stretch of the document's own words the next build reads as
// syntax of gdoc's: where it starts in the text, what it says, and what a
// warning calls it.
type shape struct {
	at         int
	what, noun string
}

// gdocShapes is every such stretch of this text, in the order it is read.
//
// Two shapes, both of them markup gdoc writes into the file itself. The first
// is a real note's reference, "[^n]", and footnoteMark holds what the next
// build makes of one. The second is the attribute a mark's span closes with,
// "{.pencil}", which comes off the palette rather than being a constant here,
// because the house style is the one place the syntax is stated. The attribute
// is searched for on its own, without the bracket in front of it, because all
// three shapes it appears in are losses: internal/body refuses an attribute
// that closes no span it reads and refuses a closer no line opened, and a
// whole span written in the document's own words publishes a grey phrase the
// document never carried with the brackets dropped out of the text.
//
// Neither shape can be escaped out of the way: internal/body hands goldmark's
// text through as it stands and never unescapes a punctuation mark, so a
// backslash written here reaches the document as a backslash. The words go
// home as the document holds them and the loss is named, which is what a mark
// that cannot carry its syntax already does.
func (e *emitter) gdocShapes(text string) []shape {
	var out []shape
	for _, m := range footnoteMark.FindAllStringIndex(text, -1) {
		out = append(out, shape{at: m[0], what: text[m[0]:m[1]], noun: "a footnote mark of gdoc's"})
	}
	for _, a := range e.spanAttributes() {
		for from := 0; ; {
			i := strings.Index(text[from:], a)
			if i < 0 {
				break
			}
			out = append(out, shape{at: from + i, what: a, noun: "the attribute gdoc closes a mark's span with"})
			from += i + len(a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].at < out[j].at })
	return out
}

// spanAttributes is the attribute each span in the palette closes with, once
// each. A mark written as a delimiter pair has none, and two marks sharing one
// attribute would otherwise name the same words twice.
func (e *emitter) spanAttributes() []string {
	var out []string
	for i := range e.opts.Marks {
		shut := e.opts.Marks[i].Shut
		brace := strings.Index(shut, "{")
		if brace < 0 || !strings.Contains(shut, "]") {
			continue
		}
		a := shut[brace:]
		seen := false
		for _, had := range out {
			if had == a {
				seen = true
			}
		}
		if !seen {
			out = append(out, a)
		}
	}
	return out
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

// body writes runs with the marks the caller named around them, and the runs
// that carry no mark as they stand.
//
// Adjacent runs carrying one mark are one mark: Docs splits a run wherever the
// formatting changes, so a highlight over a phrase holding bold arrives as
// several runs sharing one background, and a pair of delimiters around each of
// them would say the author marked the words three times.
//
// A group with no words of its own is written bare: the paragraph's own
// newline, which the chunks carry, and the whitespace an edit can leave marked
// on its own. An empty "====" would be a mark over nothing, and "== ==" does
// not read as a mark at all.
func (e *emitter) body(rs []docs.Run) {
	if len(e.opts.Marks) == 0 {
		e.links(rs)
		return
	}
	for i := 0; i < len(rs); {
		m := e.markFor(rs[i])
		j := i + 1
		for j < len(rs) && e.markFor(rs[j]) == m {
			j++
		}
		if m == nil || !hasWords(rs[i:j]) {
			e.links(rs[i:j])
			i = j
			continue
		}
		head, words, tail := edges(rs[i:j])
		e.links(head)
		// The markers that open where the mark opens are drained before the
		// delimiter, for the reason linked drains before its bracket: a
		// comment anchored at the same character would otherwise end up
		// behind markup gdoc wrote.
		e.drain(words[0].StartIndex, false)
		why := e.cannotMarkHere()
		if why == "" {
			why = e.cannotCarry(m, words)
		}
		if why == "" {
			why = e.cannotOpenHere(m, words)
		}
		if why != "" {
			e.warnings = append(e.warnings, fmt.Sprintf(
				"the mark on the words at index %d is printed as the words alone: %s", words[0].StartIndex, why))
			e.links(words)
			e.links(tail)
			i = j
			continue
		}
		e.write(m.opener(words))
		e.links(words)
		e.write(m.Shut)
		e.links(tail)
		i = j
	}
}

// cannotMarkHere is why no mark can be written on this paragraph at all, or
// nothing where one can.
//
// A heading is the one paragraph that carries none. internal/body refuses a
// mark inside a heading by line, because Word copies a heading into the
// contents field and the mark would be in the contents list as well as on the
// page, so a highlighted heading written home as "## Scope ==to fill in==" is
// a file the next build refuses, blaming the file rather than the export. The
// words go home alone and the loss is named, which is what cannotCarry does
// for words that cannot carry the syntax.
//
// It is asked before cannotCarry, because this is the reason a reader needs:
// the words are fine and the line they are on is not.
// internal/export's TestAMarkInsideAHeadingComesHomeAsItsWords and
// TestAMarkOutsideAHeadingIsNotNamedAsAHeadingLoss are the pins.
func (e *emitter) cannotMarkHere() string {
	if !e.inHeading {
		return ""
	}
	return "these words are a heading's, and the next build refuses a mark inside a heading, because Word copies a heading into the contents field and the mark would be in the contents list as well as on the page"
}

// cannotOpenHere is why this mark's opener cannot be written where the text
// stands, or nothing where it can be.
//
// A delimiter pair is read by its run of characters, so an opener written right
// against the closer of the mark before it is one run of four: the next build
// reads one pair out of it and leaves the other two characters in the words. A
// highlight on "ab" and another on "cd" with nothing between them came home as
// "==ab====cd==", which publishes as one colour over "ab====cd" or as one
// colour and the other's syntax printed as words, and nothing said so.
//
// cannotCarry reads the group's own words, and this is the one thing outside
// them that changes what the delimiter reads as. The character before the
// opener has to be one gdoc wrote: a character of the document's own separates
// the two runs, whatever it is, and a held character is the document's. The
// second mark is the one that loses its syntax, because the first is already
// written, and the loss is named the way every other one is.
// TestAMarkRightAgainstAnotherMarkIsPrintedAsItsWords is the pin.
func (e *emitter) cannotOpenHere(m *Mark, words []docs.Run) string {
	first := edgeRune(m.opener(words), true)
	if first == 0 || e.held != 0 || !e.prevGdoc || e.prev != first {
		return ""
	}
	return "the mark before these words closes right against this one's delimiter, and the two would read as one run of characters"
}

// cannotCarry is why this mark cannot be written around these words, or nothing
// where it can be. The words are then written as they stand and the loss is
// named, which is what a placeholder does for a picture.
//
// A mark comes home as the syntax an author types, and the next build reads
// that syntax back. Two shapes of words it cannot be read around:
//
//   - A span's words end at the first bracket a reader meets, so a bracket the
//     document holds closes the span early, and so do the brackets gdoc writes
//     for a run that is not text: a footnote mark is "[^3]". internal/body
//     refuses the line rather than guessing, so the file would not build at
//     all. A link is the same bracket from the other side: "[[words](url)]"
//     opens nothing either side can read. Words opening with a caret are the
//     same bracket once more: the span's own "[" in front of them is how a
//     footnote mark is written, and goldmark's footnote parser runs ahead of
//     the span parser, so "[^1]{.pencil}" is a mark of gdoc's to the next
//     build. A label the file does not define refuses it by line; a label it
//     does define grows a second reference to a note the document never had,
//     and the words and the colour go with it.
//   - A delimiter pair is read by its run of characters, so the pair itself
//     inside the words closes the mark early, and one of its characters against
//     either delimiter lengthens that run and comes home as a stray character
//     in the document.
//
// Every mark in the palette is asked, not this one alone. A mark the words hold
// is another mark's syntax inside this one, which the next build refuses as a
// mark inside a mark, and a span's closer inside a highlight's words is one it
// refuses as a span the line does not open. internal/export's
// TestAMarkWhoseWordsHoldAnotherMarksSyntaxComesHomeAsWords is the pin.
//
// A backslash is not the answer. internal/body hands goldmark's text through as
// it stands and never unescapes a punctuation mark, so a backslash written into
// the file reaches the document as a backslash. A colour nobody can see in the
// file is better than a file that will not build, and better than a backslash
// nobody typed in the document. internal/export's
// TestAMarkWhoseWordsCannotCarryItsSyntaxComesHomeAsWordsAndIsNamed is the pin.
func (e *emitter) cannotCarry(m *Mark, rs []docs.Run) string {
	words, allText := writtenWords(rs)
	if strings.Contains(m.Open, "[") || strings.Contains(m.Shut, "]") {
		if !allText {
			return "a span marks words, and gdoc writes this content as markup of its own, which carries the bracket that would close the span"
		}
		if strings.ContainsAny(words, "[]") {
			return "the words hold a bracket, and a span's words end at the first bracket a reader meets"
		}
		if strings.HasPrefix(words, "^") {
			return "the words open with a caret, and a bracket in front of one is how a footnote mark is written"
		}
		for _, r := range rs {
			if l := textLink(r); l != nil && e.target(l) != "" {
				return "the words point somewhere, and a link is written with the bracket that would close the span"
			}
		}
		return e.holdsAnotherMark(m, words)
	}
	if strings.Contains(words, m.Shut) || strings.Contains(words, m.Open) {
		return "the words hold the delimiter the mark is written with, which would close it early"
	}
	if edgeRune(m.Open, false) != 0 && strings.HasPrefix(words, string(edgeRune(m.Open, false))) {
		return "the words open with the character the mark's own delimiter is made of, which would lengthen it"
	}
	if edgeRune(m.Shut, true) != 0 && strings.HasSuffix(words, string(edgeRune(m.Shut, true))) {
		return "the words end with the character the mark's own delimiter is made of, which would lengthen it"
	}
	return e.holdsAnotherMark(m, words)
}

// holdsAnotherMark is why these words hold the syntax of a mark that is not
// this one, or nothing where they hold none.
//
// A span's closer is enough on its own: the next build refuses a closer the
// line does not open, whatever sits in front of it. A delimiter pair takes
// both halves, because one "==" in a sentence is a sentence and the build
// reads a mark only where a pair closes. Either way the mark around these
// words would nest inside this one, and two colours on one stretch of words is
// what the house style has no answer for.
func (e *emitter) holdsAnotherMark(m *Mark, words string) string {
	for i := range e.opts.Marks {
		other := &e.opts.Marks[i]
		// A mark that shuts with nothing is written around nothing, and
		// counting the empty string in the words would refuse every one of
		// them.
		if other.Shut == "" || (other.Open == m.Open && other.Shut == m.Shut) {
			continue
		}
		if strings.Contains(other.Shut, "]") {
			if strings.Contains(words, other.Shut) {
				return "the words hold the closer of another mark, which the next build reads as a span this line does not open"
			}
			continue
		}
		if strings.Count(words, other.Shut) > 1 {
			return "the words hold the delimiter another mark is written with, twice, which the next build reads as a mark inside this one"
		}
	}
	return ""
}

// writtenWords is the document's own text in these runs, and whether every one
// of them is text: a run that is not writes gdoc's markup instead, which is
// what the caller is asking about.
func writtenWords(rs []docs.Run) (words string, allText bool) {
	var b strings.Builder
	allText = true
	for _, r := range rs {
		if r.Kind != docs.KindText {
			allText = false
			continue
		}
		b.WriteString(r.Text)
	}
	return b.String(), allText
}

// edgeRune is the first or the last rune of a delimiter, or 0 where there is
// none.
func edgeRune(s string, first bool) rune {
	rs := []rune(s)
	if len(rs) == 0 {
		return 0
	}
	if first {
		return rs[0]
	}
	return rs[len(rs)-1]
}

// markFor is the mark this run carries, as the one element of the caller's
// list it is, so two runs carrying one mark compare equal by pointer.
func (e *emitter) markFor(r docs.Run) *Mark {
	for i := range e.opts.Marks {
		if e.opts.Marks[i].matches(r) {
			return &e.opts.Marks[i]
		}
	}
	return nil
}

// markEdge says this rune is whitespace that cannot sit inside a mark's
// delimiters. It is the set goldmark's own flanking test reads, which is
// unicode.IsSpace: a space and a tab, the newline and the carriage return,
// Docs' soft line break at U+000B, the form feed, and the non-breaking space.
// A narrower set left a mark that ended in a soft break written as
// "==words\v==", whose closing delimiter is not right-flanking, so the
// highlight came home as two pairs of equals signs in the words.
// TestAMarksOwnSoftBreakAndNonBreakingSpaceAreWrittenOutsideTheDelimiters is
// the pin.
func markEdge(r rune) bool { return unicode.IsSpace(r) }

// edges is the group's whitespace at either end, split off the words between
// them, so a mark is written around the words alone.
//
// A highlight's delimiter is read by flanking: "==" with a space after it does
// not open a highlight and "==" with a space before it does not close one. A
// run in Docs can begin or end with a space, which is what an edit leaves
// behind, and written as "==words ==" the delimiters would come home as words
// and the mark would be gone. The space is the document's either way, so it is
// written outside the delimiters rather than dropped.
//
// A run that points somewhere is left whole: splitting it would write its
// target twice, once around the space and once around the words.
func edges(rs []docs.Run) (head, words, tail []docs.Run) {
	words = rs
	if first, rest, ok := cutEdge(words, true); ok {
		head, words = first, rest
	}
	if last, rest, ok := cutEdge(words, false); ok {
		tail, words = last, rest
	}
	return head, words, tail
}

// cutEdge takes the whitespace off one end of the group: the runs that hold
// nothing else, and then the whitespace of the run that holds words too.
//
// The group is copied, because the runs it is a window on belong to the
// paragraph. The walk stops at the first run that holds a word, which the
// group has by hasWords over this same whitespace set, so it never empties.
func cutEdge(rs []docs.Run, front bool) (cut, rest []docs.Run, ok bool) {
	rest = append([]docs.Run{}, rs...)
	for {
		at := 0
		if !front {
			at = len(rest) - 1
		}
		r := rest[at]
		if r.Link != nil {
			// Splitting it would write its target twice, once around the
			// space and once around the words.
			break
		}
		if strings.TrimFunc(r.Text, markEdge) == "" {
			cut, rest, ok = append(cut, r), append(rest[:at], rest[at+1:]...), true
			continue
		}
		n := edgeWidth(r.Text, front)
		if n == 0 {
			break
		}
		space, words := splitRun(r, n, front)
		rest[at] = words
		cut, ok = append(cut, space), true
		break
	}
	if !front {
		// The run nearest the words was taken first, so the tail reads
		// backwards.
		for i, j := 0, len(cut)-1; i < j; i, j = i+1, j-1 {
			cut[i], cut[j] = cut[j], cut[i]
		}
	}
	return cut, rest, ok
}

// edgeWidth is how many bytes of whitespace the text carries at the end
// cutEdge is working on. Bytes rather than characters, because it is where
// splitRun cuts the string; a Docs index is counted there, off the text the
// cut gave.
func edgeWidth(text string, front bool) int {
	if front {
		return len(text) - len(strings.TrimLeftFunc(text, markEdge))
	}
	return len(text) - len(strings.TrimRightFunc(text, markEdge))
}

// splitRun is the run's whitespace at one end and the rest of it, each
// carrying the index its own text begins at. n is bytes, which is where the
// string is cut, and the boundary is the UTF-16 units of the piece cut off,
// counted from the end it was taken from: a Docs index counts units, and a
// non-breaking space is one unit in two bytes.
//
// It is counted from that end rather than from the other one, because the
// words between the two ends are not ASCII. The tail's boundary was once
// StartIndex plus the byte length of the words, which is a byte too far for
// every multi-byte rune in them: a comment opening on the word after the space
// was then drained before the space and came home behind the delimiter.
// TestATailCutCountsTheIndexInUTF16Units is the pin.
func splitRun(r docs.Run, n int, front bool) (space, words docs.Run) {
	space, words = r, r
	if front {
		space.Text, words.Text = r.Text[:n], r.Text[n:]
		space.EndIndex = r.StartIndex + units(space.Text)
		words.StartIndex = space.EndIndex
		return space, words
	}
	at := len(r.Text) - n
	words.Text, space.Text = r.Text[:at], r.Text[at:]
	space.StartIndex = r.EndIndex - units(space.Text)
	words.EndIndex = space.StartIndex
	return space, words
}

// hasWords says whether these runs hold a character a mark can sit around.
//
// The set trimmed here is markEdge, the same set cutEdge takes off either end,
// and the two cannot differ. A group counted as holding words and then emptied
// by the cut left the cut nothing to index: a space an author highlighted on
// its own answered yes to "\n" alone and no to " \t\r\n", and the export of a
// document anybody could write panicked.
// TestAMarkedRunOfOnlyWhitespaceComesHomeAsItStands is the pin.
func hasWords(rs []docs.Run) bool {
	for _, r := range rs {
		if strings.TrimFunc(r.Text, markEdge) != "" {
			return true
		}
	}
	return false
}

// units is the text's length in UTF-16 units, which is what a Docs index
// counts.
func units(s string) int {
	n := 0
	for _, r := range s {
		n += utf16Len(r)
	}
	return n
}

// links writes runs with their comment markers and their placeholders, tracking
// the character index as it goes. Runs that point at the same target are one
// link: Docs splits a run wherever the formatting changes, and two forms around
// one linked phrase would read as two links.
func (e *emitter) links(rs []docs.Run) {
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
		number := e.footnote(r)
		// The reference is written where the document holds it, heading or
		// not: the note's own words go under the body, and the next build
		// refuses a definition nothing marks as well. So the file is one that
		// build refuses either way, and the one thing this projection can do
		// is say which words have to move. internal/export's
		// TestAFootnoteMarkInsideAHeadingIsNamedAsALoss is the pin.
		if e.inHeading && e.opts.Rebuilt {
			e.warnings = append(e.warnings, fmt.Sprintf(
				"the footnote mark [^%s] at index %d is a heading's, and the next build refuses a footnote mark inside a heading, because Word copies a heading into the contents field and the note would be marked from the contents list as well as from the page",
				number, r.StartIndex))
		}
		e.write("[^" + number + "]")
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
		if r.Kind == docs.KindPageBreak && e.breakAlone && e.opts.PageBreak != "" {
			// A break of its own, and the caller said what line it is. Like a
			// picture with a file beside the note, the fact is carried rather
			// than lost, so there is nothing to warn about.
			e.write(e.opts.PageBreak)
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
//
// A note's own words are the document's own words, and they are written into
// the file the same way the prose is, so they are asked the same question:
// ownSyntax walks the prose runs and never reaches here, and a note holding
// "[^9]" went home in a file the next build refuses by line, blaming the file
// rather than the export. The warning names the note rather than an index,
// because a second copy of the document's text occupies no index of its own,
// which is what writeText's -1 says.
// internal/export's
// TestAFootnoteWhoseOwnWordsWouldReadAsAFootnoteMarkIsNamedAsALoss is the pin.
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
		if e.opts.Rebuilt {
			for _, f := range e.gdocShapes(text) {
				e.warnings = append(e.warnings, fmt.Sprintf(
					"the footnote [^%s] holds %s in its own words, which the next build reads as %s rather than as the words this document shows",
					n.number, f.what, f.noun))
			}
		}
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
