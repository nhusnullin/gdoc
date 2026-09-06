// The raw side of the Docs read, and the walk that turns it into the tree.
//
// The raw structs name only the fields gdoc reads. Google adds fields to this
// response and a strict decode would fail on the next one, so the decode here
// is deliberately loose: what is named is read, what is not is ignored. The
// front matter is the opposite case and decodes strictly, because there the
// unknown key is a person's typo rather than Google's next release.
package docs

import (
	"encoding/json"
	"fmt"
	"strings"
)

type rawDocument struct {
	DocumentID    string                     `json:"documentId"`
	Title         string                     `json:"title"`
	RevisionID    string                     `json:"revisionId"`
	Body          *rawBody                   `json:"body"`
	Footnotes     map[string]rawFootnote     `json:"footnotes"`
	InlineObjects map[string]rawInlineObject `json:"inlineObjects"`
	Tabs          []rawTab                   `json:"tabs"`
	// Comments stays raw. Its shape is measured, not documented, so each entry
	// is read field by field rather than decoded into a struct that a surprise
	// would break the whole read on.
	Comments []json.RawMessage `json:"comments"`
}

type rawTab struct {
	TabProperties struct {
		TabID string `json:"tabId"`
		Title string `json:"title"`
	} `json:"tabProperties"`
	DocumentTab *rawDocumentTab `json:"documentTab"`
	ChildTabs   []rawTab        `json:"childTabs"`
}

type rawDocumentTab struct {
	Body          *rawBody                   `json:"body"`
	Footnotes     map[string]rawFootnote     `json:"footnotes"`
	InlineObjects map[string]rawInlineObject `json:"inlineObjects"`
}

type rawBody struct {
	Content []rawElement `json:"content"`
}

// content reads a body that may be absent. A tab with no body is a tab with no
// blocks, not a failed read.
func (b *rawBody) content() []rawElement {
	if b == nil {
		return nil
	}
	return b.Content
}

type rawFootnote struct {
	FootnoteID string       `json:"footnoteId"`
	Content    []rawElement `json:"content"`
}

type rawInlineObject struct {
	InlineObjectProperties struct {
		EmbeddedObject struct {
			ImageProperties           json.RawMessage `json:"imageProperties"`
			EmbeddedDrawingProperties json.RawMessage `json:"embeddedDrawingProperties"`
		} `json:"embeddedObject"`
	} `json:"inlineObjectProperties"`
}

type rawElement struct {
	StartIndex int           `json:"startIndex"`
	EndIndex   int           `json:"endIndex"`
	Paragraph  *rawParagraph `json:"paragraph"`
	Table      *rawTable     `json:"table"`
}

type rawParagraph struct {
	Elements       []rawParaElement `json:"elements"`
	ParagraphStyle struct {
		NamedStyleType string `json:"namedStyleType"`
	} `json:"paragraphStyle"`
	Bullet *struct {
		NestingLevel int `json:"nestingLevel"`
	} `json:"bullet"`
}

type rawParaElement struct {
	StartIndex          int                     `json:"startIndex"`
	EndIndex            int                     `json:"endIndex"`
	TextRun             *rawTextRun             `json:"textRun"`
	InlineObjectElement *rawInlineObjectElement `json:"inlineObjectElement"`
	FootnoteReference   *rawFootnoteReference   `json:"footnoteReference"`
	Equation            *rawSuggested           `json:"equation"`
}

// rawSuggested is the pair of id lists every element in a paragraph can carry.
// They are what SUGGESTIONS_INLINE is read for.
type rawSuggested struct {
	SuggestedInsertionIDs []string `json:"suggestedInsertionIds"`
	SuggestedDeletionIDs  []string `json:"suggestedDeletionIds"`
}

type rawTextRun struct {
	Content string `json:"content"`
	rawSuggested
}

type rawInlineObjectElement struct {
	InlineObjectID string `json:"inlineObjectId"`
	rawSuggested
}

type rawFootnoteReference struct {
	FootnoteID     string `json:"footnoteId"`
	FootnoteNumber string `json:"footnoteNumber"`
	rawSuggested
}

type rawTable struct {
	TableRows []struct {
		TableCells []struct {
			Content []rawElement `json:"content"`
		} `json:"tableCells"`
	} `json:"tableRows"`
}

// blocks walks a body into paragraphs and tables, in reading order. An element
// that is neither carries no text gdoc reads: a section break, a table of
// contents Google generated, and whatever is added next.
func blocks(content []rawElement, objs map[string]rawInlineObject) []Block {
	var out []Block
	for _, el := range content {
		switch {
		case el.Paragraph != nil:
			out = append(out, Block{Paragraph: paragraph(el, objs)})
		case el.Table != nil:
			out = append(out, Block{Table: table(el.Table, objs)})
		}
	}
	return out
}

func paragraph(el rawElement, objs map[string]rawInlineObject) *Paragraph {
	p := &Paragraph{
		Style:      el.Paragraph.ParagraphStyle.NamedStyleType,
		StartIndex: el.StartIndex,
		EndIndex:   el.EndIndex,
	}
	if el.Paragraph.Bullet != nil {
		p.Bullet = &Bullet{NestingLevel: el.Paragraph.Bullet.NestingLevel}
	}
	for _, e := range el.Paragraph.Elements {
		if r, ok := run(e, objs); ok {
			p.Runs = append(p.Runs, r)
		}
	}
	return p
}

// run reads one paragraph element. The second value is false for an element
// gdoc has no run for: a page break, a column break, a horizontal rule. They
// hold nothing to print and nothing to suggest against.
func run(e rawParaElement, objs map[string]rawInlineObject) (Run, bool) {
	r := Run{StartIndex: e.StartIndex, EndIndex: e.EndIndex}
	switch {
	case e.TextRun != nil:
		r.Kind = KindText
		r.Text = e.TextRun.Content
		r.InsertionIDs, r.DeletionIDs = e.TextRun.ids()
	case e.InlineObjectElement != nil:
		r.Kind = objectKind(objs[e.InlineObjectElement.InlineObjectID])
		r.InsertionIDs, r.DeletionIDs = e.InlineObjectElement.ids()
	case e.FootnoteReference != nil:
		r.Kind = KindFootnoteRef
		r.Text = e.FootnoteReference.FootnoteNumber
		r.FootnoteID = e.FootnoteReference.FootnoteID
		r.InsertionIDs, r.DeletionIDs = e.FootnoteReference.ids()
	case e.Equation != nil:
		r.Kind = KindEquation
		r.InsertionIDs, r.DeletionIDs = e.Equation.ids()
	default:
		return Run{}, false
	}
	return r, true
}

// ids returns the two id lists, copied. The lists reach the envelope and the
// snapshot, and a slice shared with the decoded body is a slice somebody else
// can write through.
func (s rawSuggested) ids() (insertions, deletions []string) {
	return append([]string(nil), s.SuggestedInsertionIDs...),
		append([]string(nil), s.SuggestedDeletionIDs...)
}

// objectKind says what an embedded object is, by the properties it carries. An
// object with neither is reported as an object: what it is, is a fact gdoc does
// not have.
func objectKind(o rawInlineObject) string {
	switch {
	case len(o.InlineObjectProperties.EmbeddedObject.EmbeddedDrawingProperties) > 0:
		return KindDrawing
	case len(o.InlineObjectProperties.EmbeddedObject.ImageProperties) > 0:
		return KindImage
	}
	return KindObject
}

func table(t *rawTable, objs map[string]rawInlineObject) Table {
	out := make(Table, 0, len(t.TableRows))
	for _, row := range t.TableRows {
		cells := make([]Cell, 0, len(row.TableCells))
		for _, c := range row.TableCells {
			cells = append(cells, Cell{Blocks: blocks(c.Content, objs)})
		}
		out = append(out, cells)
	}
	return out
}

// plainText is every run's text in reading order, with the trailing newline
// dropped. It is what a footnote says, and nothing else uses it: the document
// body has its own projection, in internal/view.
func plainText(bs []Block) string {
	var b strings.Builder
	for _, blk := range bs {
		if blk.Paragraph == nil {
			continue
		}
		for _, r := range blk.Paragraph.Runs {
			b.WriteString(r.Text)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// commentRanges reads the top-level comments array of the Docs response.
//
// The shape is measured, not documented: the reference for documents.get does
// not describe what commentsViewMode adds, so the decoder tries the three
// places a range has been seen and reports the entry as unplaced when none of
// them reads. An unplaced comment is a fact the command puts in warnings; the
// thread itself still comes back, from Drive, with its quoted text.
func commentRanges(raws []json.RawMessage, tabs []Tab) (map[string]Range, []string) {
	placed := map[string]Range{}
	var unplaced []string
	for i, raw := range raws {
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil {
			unplaced = append(unplaced, fmt.Sprintf("comments[%d] (unreadable)", i))
			continue
		}
		id := firstString(fields, "id", "commentId")
		if id == "" {
			// Nothing to key it by, so it is named by where it was.
			unplaced = append(unplaced, fmt.Sprintf("comments[%d] (no id)", i))
			continue
		}
		if r, ok := rangeOf(fields, tabs); ok {
			placed[id] = r
			continue
		}
		unplaced = append(unplaced, id)
	}
	if len(placed) == 0 {
		placed = nil
	}
	return placed, unplaced
}

// rangeOf looks for a range in the three shapes it has been seen in: a `range`
// object, a `range` under `anchor`, and start and end at the top level. An
// `anchor` that is a string, which is what Drive's own comments carry, is
// stepped over rather than failing the entry.
func rangeOf(fields map[string]json.RawMessage, tabs []Tab) (Range, bool) {
	if r, ok := decodeRange(fields["range"], fields, tabs); ok {
		return r, true
	}
	var anchor map[string]json.RawMessage
	if raw, has := fields["anchor"]; has && json.Unmarshal(raw, &anchor) == nil {
		if r, ok := decodeRange(anchor["range"], anchor, tabs); ok {
			return r, true
		}
		if r, ok := indexRange(anchor, fields, tabs); ok {
			return r, true
		}
	}
	return indexRange(fields, fields, tabs)
}

// decodeRange reads a `range` object. outer is where the tab id is looked for
// when the range itself does not name one.
func decodeRange(raw json.RawMessage, outer map[string]json.RawMessage, tabs []Tab) (Range, bool) {
	if len(raw) == 0 {
		return Range{}, false
	}
	var inner map[string]json.RawMessage
	if json.Unmarshal(raw, &inner) != nil {
		return Range{}, false
	}
	return indexRange(inner, outer, tabs)
}

// indexRange reads startIndex and endIndex out of one object and pairs them
// with a tab. A comment with no readable tab in a document with more than one
// names no position gdoc can print, so it comes back unplaced rather than
// guessed at.
func indexRange(obj, outer map[string]json.RawMessage, tabs []Tab) (Range, bool) {
	start, okStart := intField(obj, "startIndex")
	end, okEnd := intField(obj, "endIndex")
	if !okStart || !okEnd {
		return Range{}, false
	}
	tab := firstString(obj, "tabId")
	if tab == "" {
		tab = firstString(outer, "tabId")
	}
	if tab == "" {
		if len(tabs) != 1 {
			return Range{}, false
		}
		tab = tabs[0].ID
	}
	return Range{Tab: tab, Start: start, End: end}, true
}

func firstString(fields map[string]json.RawMessage, names ...string) string {
	for _, name := range names {
		var s string
		if raw, has := fields[name]; has && json.Unmarshal(raw, &s) == nil && s != "" {
			return s
		}
	}
	return ""
}

func intField(fields map[string]json.RawMessage, name string) (int, bool) {
	raw, has := fields[name]
	if !has {
		return 0, false
	}
	var n int
	if json.Unmarshal(raw, &n) != nil {
		return 0, false
	}
	return n, true
}
