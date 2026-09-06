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
}

// Block is one element of a body: either a paragraph or a table, never both and
// never neither. Everything else Docs puts in a body (a section break, a table
// of contents) carries no text gdoc reads, so the walk drops it.
type Block struct {
	Paragraph *Paragraph `json:"paragraph,omitempty"`
	Table     Table      `json:"table,omitempty"`
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

// Table is rows of cells, in reading order.
type Table [][]Cell

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
}

// Range is where a comment is anchored, in the character indexes of one tab.
type Range struct {
	Tab   string `json:"tab"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// MultiTab says whether the document has more than one tab. Every document gdoc
// makes has one; a document with more is read in full and reported, and only
// the write milestones stop on it.
func (d *Document) MultiTab() bool { return len(d.Tabs) > 1 }

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
		// The pre-tabs shape: one body, and Docs calls that tab t.0.
		d.Tabs = append(d.Tabs, Tab{ID: defaultTabID, Body: blocks(r.Body.content(), r.InlineObjects)})
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
		d.addFootnotes(t.DocumentTab.Footnotes, t.DocumentTab.InlineObjects)
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
