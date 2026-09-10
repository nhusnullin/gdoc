// Package docs is the Docs read and the tree it comes back as. One request
// carries three views: the document's content with every tab in it, the pending
// suggestions with their ids, and the character ranges the comment threads are
// anchored to. read, suggestions and the range half of comments all stand on
// that one tree, so there is one read per command and one shape to test.
//
// Everything here is read-only, and Parse takes bytes, so the fixtures under
// testdata/ run the same decoder the wire feeds. Nothing in this package
// decides anything: it reports what Google answered and leaves every judgement
// to the skill reading the JSON.
package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// The run kinds. A run is a piece of a paragraph, and the kind says what the
// document holds there. Text is the only kind carrying text; the rest are
// positions something else sits at, and the reader prints a placeholder.
const (
	KindText        = "text"
	KindImage       = "image"
	KindDrawing     = "drawing"
	KindEquation    = "equation"
	KindFootnoteRef = "footnote_ref"
	// KindObject is an embedded object the read cannot classify: not an image
	// and not a drawing by the fields it carries. Calling it an image would be
	// a guess, and this package reports rather than guesses.
	KindObject = "object"
	// The three smart chips. Each is a live reference the document shows as a
	// label, and each is one character in the document's own numbering.
	KindPerson   = "person"
	KindDate     = "date"
	KindRichLink = "rich_link"
	// The four positions that carry no text: a page number or date field, and
	// the three breaks and rules.
	KindAutoText       = "auto_text"
	KindPageBreak      = "page_break"
	KindColumnBreak    = "column_break"
	KindHorizontalRule = "horizontal_rule"
	// KindUnknown is a paragraph element this read does not name. It is the
	// whole point of the walk having no silent default: seven kinds went
	// missing for a milestone because an element it could not name vanished
	// rather than reporting itself, and the next kind Google adds must not.
	KindUnknown = "unknown"
)

// defaultTabID is what the one tab of a pre-tabs document is called. Documents
// written before tabs existed answer with a top-level body and no tabs array,
// and Docs itself calls the implicit first tab t.0.
const defaultTabID = "t.0"

// Document is one Docs read. CommentRanges is keyed by the Drive comment id, so
// the comments command joins it to the thread list without a second lookup.
// Unplaced names the comments the read returned without a range this decoder
// could read; they are reported, never dropped.
type Document struct {
	ID            string            `json:"id"`
	Title         string            `json:"title"`
	RevisionID    string            `json:"revision_id"`
	Tabs          []Tab             `json:"tabs"`
	CommentRanges map[string]Range  `json:"comment_ranges,omitempty"`
	Unplaced      []string          `json:"unplaced,omitempty"`
	Footnotes     map[string]string `json:"footnotes,omitempty"`
}

// Tab is one tab's content. A document gdoc makes has exactly one.
type Tab struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Body  []Block `json:"blocks"`
	// NamedRanges are this tab's named ranges, sorted by name and then by id so
	// one document reads the same way twice. Docs keys them by name in a map,
	// and a map is walked in no order.
	NamedRanges []NamedRange `json:"named_ranges,omitempty"`
	// anchors is the tab's commentAnchors, by anchorId, each already carrying
	// this tab's id. Unexported: it feeds CommentRanges and is not part of the
	// structure a caller reads.
	anchors map[string]Range
}

// Block is one element of a body: either a paragraph or a table, never both and
// never neither. Everything else Docs puts in a body (a section break, a table
// of contents) carries no text gdoc reads, so the walk drops it.
type Block struct {
	Paragraph *Paragraph `json:"paragraph,omitempty"`
	Table     *Table     `json:"table,omitempty"`
}

// Paragraph is a run of text with one named style. Style is the document's own
// namedStyleType (HEADING_1, TITLE, NORMAL_TEXT and the rest), passed through
// rather than translated: what a style means is the reader's business.
type Paragraph struct {
	Style      string  `json:"style"`
	Bullet     *Bullet `json:"bullet,omitempty"`
	Runs       []Run   `json:"runs"`
	StartIndex int     `json:"start_index"`
	EndIndex   int     `json:"end_index"`
}

// Bullet is the list membership of a paragraph. The nesting level is the only
// fact the text projection needs; which list it belongs to changes nothing gdoc
// prints.
type Bullet struct {
	NestingLevel int `json:"nesting_level"`
}

// Table is one table: where it starts, and its rows of cells in reading order.
//
// StartIndex is the field no projection reads, and it is here because a write
// needs it. Docs names a table by tableStartLocation, so a caller holding the
// rows alone cannot address the table it has just walked, and M7b's restyle
// styles cells. Everything else a write would like to know about a table, its
// column widths and its row heights, needs a request kind LevelInPlace does not
// carry, so nothing decodes it: a field with no reader is a field that drifts.
type Table struct {
	StartIndex int      `json:"start_index"`
	Rows       [][]Cell `json:"rows"`
}

// Cell holds blocks, because a cell holds paragraphs and can hold a table.
type Cell struct {
	Blocks []Block `json:"blocks"`
}

// Run is one piece of a paragraph. The suggestion ids are the whole reason the
// read asks for SUGGESTIONS_INLINE: they are what says this text is pending
// rather than written. A run may carry both, and that is a replacement.
type Run struct {
	Kind         string   `json:"kind"`
	Text         string   `json:"text,omitempty"`
	StartIndex   int      `json:"start_index"`
	EndIndex     int      `json:"end_index"`
	InsertionIDs []string `json:"insertion_ids,omitempty"`
	DeletionIDs  []string `json:"deletion_ids,omitempty"`
	FootnoteID   string   `json:"footnote_id,omitempty"`
	Detail       *Detail  `json:"detail,omitempty"`
}

// Detail is what a run that is not text carries besides its position: the
// fields the reference names for a chip, the kind of an auto text, and the
// member name of an element this read does not name.
//
// One struct rather than a field per kind. Every one of these is optional, they
// are read by the same two callers, and a Run with eight more empty strings on
// it is a Run whose common case is harder to read than the rare one.
//
// Text is deliberately not one of them. Text is what the document's own text
// runs hold, and it is what a footnote's text is joined from, so a chip's label
// living there would put a person's name inside a footnote that only mentions
// them.
type Detail struct {
	// ID is the chip's own id: personId, dateId or richLinkId.
	ID string `json:"id,omitempty"`
	// Label is what the document shows on screen: the person's name, the date
	// as it is displayed, the link's title.
	Label string `json:"label,omitempty"`
	// Email is the person chip's address, and URI and MimeType are the rich
	// link's target and what kind of thing it is.
	Email    string `json:"email,omitempty"`
	URI      string `json:"uri,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	// Type is an auto text's own type: PAGE_NUMBER, PAGE_COUNT and the rest.
	Type string `json:"type,omitempty"`
	// Member is the JSON member of a paragraph element this read does not name.
	// It is the one fact a person can act on when a document holds something
	// gdoc has never seen, so it reaches the warning by name.
	Member string `json:"member,omitempty"`
}

// Range is a span of one tab, in that tab's character indexes. It is where a
// comment is anchored, and it is where a named range sits.
//
// Segment names the header, footer or footnote the span is in, and it is empty
// for the body, which is where every span gdoc computes lives. It is carried
// rather than dropped because a header span read as a body span names a
// position the body does not have, and Places would then answer about the
// wrong text. Nothing sets it on a comment anchor: rawCommentAnchor reads
// startIndex and endIndex and no segment id, and the shape was measured on
// comments anchored in the body, which would carry none whatever Docs sends for
// one anchored in a header or a footnote. So the refusal below cannot fire on
// that path today, and docs/backlog holds the unmeasured question.
type Range struct {
	Tab     string `json:"tab"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Segment string `json:"segment,omitempty"`
}

// NamedRange is one named range: a label the document keeps in step with its
// own edits, tracking the text as it moves and shrinking rather than sliding
// onto foreign text when the text around it is replaced.
//
// The id is the identifier and the name is a label. Two named ranges may carry
// one name, Docs puts both under that one key, and deleting by name deletes
// every range wearing it. So nothing here is keyed by name, and a caller that
// wants one range names its id.
type NamedRange struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Tab    string  `json:"tab"`
	Ranges []Range `json:"ranges,omitempty"`
}

// NamedRanges is every named range in the document, in tab order and then in
// each tab's own order. Tab.NamedRanges is the same list for one tab, the way
// Places is one rule with two ways in.
func (d *Document) NamedRanges() []NamedRange {
	var out []NamedRange
	for _, t := range d.Tabs {
		out = append(out, t.NamedRanges...)
	}
	return out
}

// MultiTab says whether the document has more than one tab. Every document gdoc
// makes has one; a document with more is read in full and reported, and only
// the write milestones stop on it.
func (d *Document) MultiTab() bool { return len(d.Tabs) > 1 }

// Places says whether r names a position in this document: it ends after it
// starts, it names a tab the document has, it names no segment, and both its
// endpoints fall inside that tab's text.
//
// The segment is refused rather than ignored. A header, a footer and a footnote
// each carry their own indexes, and the text those indexes are read against
// here is the tab's body, so a header span answered against the body names a
// position in text it was never measured in. Every span gdoc computes today is
// a body span, so nothing changes for the two callers; what changes is the
// answer a segmented span gets, which is now no rather than a coincidence.
//
// One rule, because two commands answer from it. read marks a comment's range
// in the text and comments prints it as a position, and a range one of them
// refuses while the other prints it is a false fact in whichever field the
// skill reads. The indexes come out of the Docs comments key unchecked, and
// that decoder is loose on purpose because the shape is measured rather than
// documented, so an empty pair, an inverted pair, an index outside the text and
// a tab the document does not have are all shapes it can hand over.
func (d *Document) Places(r Range) bool {
	for _, t := range d.Tabs {
		if t.Places(r) {
			return true
		}
	}
	return false
}

// Places says whether r names a position in this one tab. Document.Places is
// this same rule over every tab, so there is still one rule and two ways in.
//
// A walker arms from the tab it is walking, never from the document, and warns
// from the document, which is the answer comments prints. The two answers only
// differ when two tabs share an id, which the decoder can produce because a tab
// carrying no id at all is called t.0: the document form would then answer off
// the first of them, and a walker that armed a comment marker on that answer
// would open a marker its own text never closes. That is the half a pair the
// escaping exists to make impossible, so each tab answers for itself.
func (t Tab) Places(r Range) bool {
	return r.Start < r.End && r.Tab == t.ID && r.Segment == "" && t.covers(r.Start, r.End)
}

// covers says whether both indexes fall inside this tab's text runs. An index
// is in a run when it is at either end of it or between them, because a range
// closing at the last character of a run closes at that run's end index.
//
// It is unexported on purpose: Places is the whole rule, and a caller reaching
// past it for one third of the rule is how the two commands drift apart.
func (t Tab) covers(start, end int) bool {
	hasStart, hasEnd := coveringRuns(t.Body, start, end)
	return hasStart && hasEnd
}

// coveringRuns walks the blocks once, tables walked into, and says whether
// start and end each fell inside a text run. It stops as soon as both have.
//
// It allocates nothing on purpose. Places is asked once per comment, in both
// commands, so building a slice of every run's index pair per comment walks the
// whole tab and throws the walk away again for each one. A document near the
// read's ceiling is a few hundred thousand runs, and the cost is then seconds
// rather than the milliseconds an ordinary review document costs.
func coveringRuns(bs []Block, start, end int) (hasStart, hasEnd bool) {
	for _, b := range bs {
		if b.Paragraph != nil {
			for _, r := range b.Paragraph.Runs {
				hasStart = hasStart || (start >= r.StartIndex && start <= r.EndIndex)
				hasEnd = hasEnd || (end >= r.StartIndex && end <= r.EndIndex)
				if hasStart && hasEnd {
					return true, true
				}
			}
			continue
		}
		if b.Table == nil {
			continue
		}
		for _, row := range b.Table.Rows {
			for _, c := range row {
				s, e := coveringRuns(c.Blocks, start, end)
				hasStart = hasStart || s
				hasEnd = hasEnd || e
				if hasStart && hasEnd {
					return true, true
				}
			}
		}
	}
	return hasStart, hasEnd
}

// URL is the one read. includeTabsContent=true is not optional: without it the
// answer covers the first tab and says nothing about the rest, and
// commentsViewMode requires it. suggestionsViewMode is the only view carrying
// suggestion ids, and Drive's export renders a suggested document as though
// nothing had been suggested.
func URL(id string) string {
	return "https://docs.googleapis.com/v1/documents/" + id +
		"?includeTabsContent=true" +
		"&suggestionsViewMode=SUGGESTIONS_INLINE" +
		"&commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED"
}

// NamedRangesURL is the same read narrowed to the named ranges. The whole
// document answers with them too, so this exists for the caller that wants one
// fact out of that answer rather than the prose: the measured saving was 982
// bytes against 12,907 for the document itself. Its caller is restyle's
// revisionOf, which reads it between batches for the revision id alone.
//
// includeTabsContent stays true, because that is where the ranges are. The
// mask selects each tab's id and its named ranges, and childTabs whole: a mask
// does not recurse into a nesting of unknown depth, so selecting the child
// tabs' fields one level at a time would leave a deeper tab's ranges silently
// absent, which is the defect this milestone exists to stop. Asking for a
// child tab in full costs bytes on the documents that have one and loses
// nothing on any of them.
//
// docsReadParams already permits fields, so the guard carries this unchanged,
// and checkFields refuses only an empty mask, a star and the permission
// surface, none of which is here.
func NamedRangesURL(id string) string {
	return "https://docs.googleapis.com/v1/documents/" + id +
		"?includeTabsContent=true" +
		"&fields=" + url.QueryEscape("documentId,revisionId,tabs(tabProperties/tabId,documentTab/namedRanges,childTabs)")
}

// Reader is the one thing this package needs of a session: a JSON GET. A
// *gapi.Session satisfies it, and so does a fake in a test. Naming the
// interface here rather than the struct keeps net/http out of this room and out
// of its tests, which is what the boundary test asks of every package but the
// four that build requests.
type Reader interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
}

// Fetch reads the document through the session's guarded client. A document the
// command was not given is refused inside the process, before anything reaches
// a wire: the refusal comes back from the guard, through the session, as this
// function's error.
func Fetch(ctx context.Context, s Reader, id string) (*Document, error) {
	var raw json.RawMessage
	if err := s.GetJSON(ctx, URL(id), &raw); err != nil {
		return nil, err
	}
	return Parse(raw)
}

// Parse decodes one Docs read. It is the same code Fetch runs, so a fixture is
// a test of the wire path.
func Parse(raw []byte) (*Document, error) {
	var r rawDocument
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("the Docs read did not decode: %w", err)
	}
	d := &Document{
		ID:         r.DocumentID,
		Title:      r.Title,
		RevisionID: r.RevisionID,
		Footnotes:  map[string]string{},
	}
	if len(r.Tabs) > 0 {
		for _, t := range r.Tabs {
			d.appendTab(t)
		}
	} else {
		// The pre-tabs shape: one body, and Docs calls that tab t.0. The named
		// ranges sit beside that body, at the top level, and are read from
		// there for the same reason the body is: on this shape there is no tab
		// to read either of them from.
		d.Tabs = append(d.Tabs, Tab{
			ID:          defaultTabID,
			Body:        blocks(r.Body.content(), r.InlineObjects),
			NamedRanges: namedRanges(r.NamedRanges, defaultTabID),
		})
		d.addFootnotes(r.Footnotes, r.InlineObjects)
	}
	d.CommentRanges, d.Unplaced = commentRanges(r.Comments, d.Tabs)
	if len(d.Footnotes) == 0 {
		d.Footnotes = nil
	}
	return d, nil
}

// appendTab flattens a tab and the tabs under it, parent first. A child tab is
// a tab: it holds its own body, and reading order is the order the document
// shows them in.
func (d *Document) appendTab(t rawTab) {
	tab := Tab{ID: t.TabProperties.TabID, Title: t.TabProperties.Title}
	if tab.ID == "" {
		tab.ID = defaultTabID
	}
	if t.DocumentTab != nil {
		tab.Body = blocks(t.DocumentTab.Body.content(), t.DocumentTab.InlineObjects)
		tab.NamedRanges = namedRanges(t.DocumentTab.NamedRanges, tab.ID)
		d.addFootnotes(t.DocumentTab.Footnotes, t.DocumentTab.InlineObjects)
		for anchorID, a := range t.DocumentTab.CommentAnchors {
			if len(a.Ranges) == 0 {
				continue // an anchor with no range places nothing; the comment stays unplaced
			}
			if tab.anchors == nil {
				tab.anchors = map[string]Range{}
			}
			tab.anchors[anchorID] = Range{Tab: tab.ID, Start: a.Ranges[0].StartIndex, End: a.Ranges[0].EndIndex}
		}
	}
	d.Tabs = append(d.Tabs, tab)
	for _, child := range t.ChildTabs {
		d.appendTab(child)
	}
}

// addFootnotes records each footnote's text under its id. The text is what the
// footnote says, with the trailing newline of its last paragraph dropped,
// because the reader prints it as a line of its own.
func (d *Document) addFootnotes(fns map[string]rawFootnote, objs map[string]rawInlineObject) {
	for id, fn := range fns {
		d.Footnotes[id] = plainText(blocks(fn.Content, objs))
	}
}
