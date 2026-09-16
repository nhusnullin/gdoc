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
	"sort"
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
	// NamedRanges is the pre-tabs location. Every read gdoc makes carries
	// includeTabsContent=true, which leaves this empty and puts the ranges in
	// the tab, so this field answers on a document written before tabs existed
	// and on nothing else.
	NamedRanges map[string]rawNamedRanges `json:"namedRanges"`
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
	// CommentAnchors is where COMMENTS_VIEW_MODE_INCLUDED puts the ranges,
	// measured 2026-09-06 on a real document: keyed by anchorId, and each
	// `comments[]` entry names its anchorId. The reference for documents.get
	// still does not describe it.
	CommentAnchors map[string]rawCommentAnchor `json:"commentAnchors"`
	// NamedRanges is where the ranges are on every read gdoc makes, keyed by
	// name. The key is not an identifier: duplicate names coexist, and each
	// entry below it holds every range wearing that name.
	NamedRanges map[string]rawNamedRanges `json:"namedRanges"`
}

// rawNamedRanges is one name's worth of ranges. Docs keys the map by name and
// then repeats the name inside, because the name identifies nothing on its own.
type rawNamedRanges struct {
	Name        string          `json:"name"`
	NamedRanges []rawNamedRange `json:"namedRanges"`
}

// rawNamedRange is one named range: an id, the name it wears, and the spans it
// covers. A span names the tab and the segment it is in, and a segment id names
// a header, a footer or a footnote rather than the body.
type rawNamedRange struct {
	NamedRangeID string `json:"namedRangeId"`
	Name         string `json:"name"`
	Ranges       []struct {
		StartIndex int    `json:"startIndex"`
		EndIndex   int    `json:"endIndex"`
		SegmentID  string `json:"segmentId"`
		TabID      string `json:"tabId"`
	} `json:"ranges"`
}

// namedRanges flattens one tab's named ranges, keyed by name in the answer and
// by nothing here: the list is the shape, and the id is the identifier.
//
// tabID is the tab the map was read from, and it is what a span with no tabId
// of its own takes. A pre-tabs document names no tab anywhere, and the ranges
// there belong to the implicit first tab, which is the same tab its body went
// into.
//
// The result is sorted by name and then by id. The answer is a map, a map is
// walked in no order, and a survey that lists a document's named ranges in a
// different order on each run is a survey nobody can diff.
func namedRanges(raw map[string]rawNamedRanges, tabID string) []NamedRange {
	var out []NamedRange
	for key, group := range raw {
		for _, nr := range group.NamedRanges {
			name := nr.Name
			if name == "" {
				// The map key is the name, and the entries under it repeat it.
				// A range that did not repeat it still wears it.
				name = group.Name
			}
			if name == "" {
				name = key
			}
			one := NamedRange{ID: nr.NamedRangeID, Name: name, Tab: tabID}
			for _, r := range nr.Ranges {
				tab := r.TabID
				if tab == "" {
					tab = tabID
				}
				one.Ranges = append(one.Ranges, Range{
					Tab: tab, Start: r.StartIndex, End: r.EndIndex, Segment: r.SegmentID,
				})
			}
			out = append(out, one)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// rawCommentAnchor is one anchor's ranges. A comment on one span has one; the
// shape allows several, and the first is where the comment sits.
type rawCommentAnchor struct {
	AnchorID string `json:"anchorId"`
	Ranges   []struct {
		StartIndex int `json:"startIndex"`
		EndIndex   int `json:"endIndex"`
	} `json:"ranges"`
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

// rawParaElement is one member of the ParagraphElement union, which the
// reference documents as eleven members. All eleven are named here, and an
// element naming none of them is still decoded: unnamed carries whatever it did
// name, so the walk can report it instead of dropping it.
type rawParaElement struct {
	StartIndex          int                     `json:"startIndex"`
	EndIndex            int                     `json:"endIndex"`
	TextRun             *rawTextRun             `json:"textRun"`
	InlineObjectElement *rawInlineObjectElement `json:"inlineObjectElement"`
	FootnoteReference   *rawFootnoteReference   `json:"footnoteReference"`
	Equation            *rawSuggested           `json:"equation"`
	Person              *rawPerson              `json:"person"`
	RichLink            *rawRichLink            `json:"richLink"`
	DateElement         *rawDateElement         `json:"dateElement"`
	AutoText            *rawAutoText            `json:"autoText"`
	PageBreak           *rawSuggested           `json:"pageBreak"`
	ColumnBreak         *rawSuggested           `json:"columnBreak"`
	HorizontalRule      *rawSuggested           `json:"horizontalRule"`
	// unnamed is every member of this element that is not in the list above,
	// sorted. It is what the warning names when Google adds a twelfth.
	unnamed []string
	// unnamedSuggested is what those members said about being suggested. Every
	// one of the eleven carries the two id lists, so the twelfth will too, and
	// a run that kept the member name and dropped the ids is a pending change
	// read prints with no markers and the survey counts in neither number.
	unnamedSuggested rawSuggested
}

// namedMembers is the union as this decoder knows it, plus the two indexes
// every element carries. A key outside this set is a member gdoc has never
// seen, and it is reported rather than ignored.
var namedMembers = map[string]bool{
	"startIndex": true, "endIndex": true,
	"textRun": true, "inlineObjectElement": true, "footnoteReference": true,
	"equation": true, "person": true, "richLink": true, "dateElement": true,
	"autoText": true, "pageBreak": true, "columnBreak": true, "horizontalRule": true,
}

// UnmarshalJSON decodes the element twice: once into the fields above, and once
// into a bare map, to learn which members it carried. The second pass is the
// only way encoding/json can answer "what else was in here", and the answer is
// what stops the next element kind from vanishing the way seven of them did.
func (e *rawParaElement) UnmarshalJSON(b []byte) error {
	// The alias drops the method, so this is the ordinary decode rather than a
	// recursive call into itself.
	type alias rawParaElement
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*e = rawParaElement(a)
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(b, &keys); err != nil {
		return err
	}
	for k := range keys {
		if !namedMembers[k] {
			e.unnamed = append(e.unnamed, k)
		}
	}
	// Sorted, because a map is walked in no order and a warning that reads
	// differently on two runs of one document is a warning nobody trusts. The
	// ids are read in that same order for the same reason.
	sort.Strings(e.unnamed)
	for _, name := range e.unnamed {
		var s rawSuggested
		// A member that is not an object, or one whose id lists are shaped
		// some other way, says nothing about being suggested. It is still an
		// element at a position with a name, so the decode carries on rather
		// than failing the whole document over a member gdoc has never seen.
		if err := json.Unmarshal(keys[name], &s); err != nil {
			continue
		}
		e.unnamedSuggested.SuggestedInsertionIDs = append(
			e.unnamedSuggested.SuggestedInsertionIDs, s.SuggestedInsertionIDs...)
		e.unnamedSuggested.SuggestedDeletionIDs = append(
			e.unnamedSuggested.SuggestedDeletionIDs, s.SuggestedDeletionIDs...)
	}
	return nil
}

// rawPerson is a person chip: a live reference to somebody, shown as their name
// or, when there is no name to show, as their address.
type rawPerson struct {
	PersonID         string `json:"personId"`
	PersonProperties struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"personProperties"`
	rawSuggested
}

// label is what the chip shows. The reference documents name as the name shown
// "instead of the person's email address", and email as always present, so a
// chip displaying the address carries no name at all. Falling back to the
// address is what keeps read carrying who the chip names: a bare [person] tells
// a reader somebody is there and not who, which for a policy naming its owner
// is the fact that mattered.
func (p rawPerson) label() string {
	if p.PersonProperties.Name != "" {
		return p.PersonProperties.Name
	}
	return p.PersonProperties.Email
}

// rawRichLink is a smart chip pointing at a Drive file, a calendar entry or a
// web page. mimeType is what kind of thing it points at.
type rawRichLink struct {
	RichLinkID         string `json:"richLinkId"`
	RichLinkProperties struct {
		Title    string `json:"title"`
		URI      string `json:"uri"`
		MimeType string `json:"mimeType"`
	} `json:"richLinkProperties"`
	rawSuggested
}

// rawDateElement is a date chip. displayText is the date as the document shows
// it, which is what a reader sees and the only part of it gdoc prints.
type rawDateElement struct {
	DateID                string `json:"dateId"`
	DateElementProperties struct {
		DisplayText string `json:"displayText"`
	} `json:"dateElementProperties"`
	rawSuggested
}

// rawAutoText is a field Docs fills in, a page number or a date. Its type is
// the only fact about it: the value is computed at layout, and gdoc lays
// nothing out.
type rawAutoText struct {
	Type string `json:"type"`
	rawSuggested
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
			out = append(out, Block{Table: table(el, objs)})
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
		p.Runs = append(p.Runs, run(e, objs))
	}
	return p
}

// run reads one paragraph element. Every element becomes a run, including one
// this decoder cannot name.
//
// It used to answer false for anything outside four cases, and seven of the
// union's eleven members fell through it: the three chips, an auto text, a page
// break, a column break and a horizontal rule. They reached neither the text
// nor the warnings, so every review since M2 read documents with holes in them
// and was told nothing. The default arm reports now, which is the wider half of
// that fix: a decoder that drops what it does not recognise makes every reader
// downstream confidently wrong, and the twelfth member arrives as a placeholder
// naming itself instead of as an absence.
func run(e rawParaElement, objs map[string]rawInlineObject) Run {
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
	case e.Person != nil:
		r.Kind = KindPerson
		r.Detail = &Detail{
			ID:    e.Person.PersonID,
			Label: e.Person.label(),
			Email: e.Person.PersonProperties.Email,
		}
		r.InsertionIDs, r.DeletionIDs = e.Person.ids()
	case e.RichLink != nil:
		r.Kind = KindRichLink
		r.Detail = &Detail{
			ID:       e.RichLink.RichLinkID,
			Label:    e.RichLink.RichLinkProperties.Title,
			URI:      e.RichLink.RichLinkProperties.URI,
			MimeType: e.RichLink.RichLinkProperties.MimeType,
		}
		r.InsertionIDs, r.DeletionIDs = e.RichLink.ids()
	case e.DateElement != nil:
		r.Kind = KindDate
		r.Detail = &Detail{
			ID:    e.DateElement.DateID,
			Label: e.DateElement.DateElementProperties.DisplayText,
		}
		r.InsertionIDs, r.DeletionIDs = e.DateElement.ids()
	case e.AutoText != nil:
		r.Kind = KindAutoText
		r.Detail = &Detail{Type: e.AutoText.Type}
		r.InsertionIDs, r.DeletionIDs = e.AutoText.ids()
	case e.PageBreak != nil:
		r.Kind = KindPageBreak
		r.InsertionIDs, r.DeletionIDs = e.PageBreak.ids()
	case e.ColumnBreak != nil:
		r.Kind = KindColumnBreak
		r.InsertionIDs, r.DeletionIDs = e.ColumnBreak.ids()
	case e.HorizontalRule != nil:
		r.Kind = KindHorizontalRule
		r.InsertionIDs, r.DeletionIDs = e.HorizontalRule.ids()
	default:
		// The element still holds a position, so it is still a run. What it is
		// called is the one fact a person can act on, and an element naming
		// nothing at all says so by naming nothing.
		r.Kind = KindUnknown
		if len(e.unnamed) > 0 {
			r.Detail = &Detail{Member: strings.Join(e.unnamed, ", ")}
		}
		// The ids come off the member itself, like every arm above. What the
		// element is called is one fact; whether somebody is waiting on it is
		// the other, and dropping the second makes every reader downstream
		// confidently wrong about a document with a pending change in it.
		r.InsertionIDs, r.DeletionIDs = e.unnamedSuggested.ids()
	}
	return r
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

// table walks one table element. The index comes from the element itself rather
// than from anything inside it, which is what makes a table inside a cell carry
// its own: a nested table is an element of that cell's content, walked by the
// same two functions.
func table(el rawElement, objs map[string]rawInlineObject) *Table {
	out := &Table{StartIndex: el.StartIndex, Rows: make([][]Cell, 0, len(el.Table.TableRows))}
	for _, row := range el.Table.TableRows {
		cells := make([]Cell, 0, len(row.TableCells))
		for _, c := range row.TableCells {
			cells = append(cells, Cell{Blocks: blocks(c.Content, objs)})
		}
		out.Rows = append(out.Rows, cells)
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
// The shape was measured on 2026-09-06, against a real document: each entry
// carries `commentId` and an `anchorId`, and the range sits in the tab, under
// `documentTab.commentAnchors[anchorId].ranges`. That lookup runs first. The
// reference for documents.get still does not describe what commentsViewMode
// adds, so the three shapes the decoder guessed at before the measurement stay
// as fallbacks, and an entry none of them reads is reported as unplaced. An
// unplaced comment is a fact the command puts in warnings; the thread itself
// still comes back, from Drive, with its quoted text.
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
		if r, ok := anchoredRange(firstString(fields, "anchorId"), tabs); ok {
			placed[id] = r
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

// anchoredRange is the measured shape: the comment names an anchorId, and the
// tab that holds that anchor knows its range. The tab is the one whose anchors
// named it, so a multi-tab document needs no tabId on the comment.
func anchoredRange(anchorID string, tabs []Tab) (Range, bool) {
	if anchorID == "" {
		return Range{}, false
	}
	for _, t := range tabs {
		if r, ok := t.anchors[anchorID]; ok {
			return r, true
		}
	}
	return Range{}, false
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
