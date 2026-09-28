// This file is where a block goes: the index its text is inserted at, the
// whole paragraphs a replace takes out, and every placement gdoc refuses by
// name. span.go finds the words in the document, block.go reads the content,
// and doc.go states the rules with the test that pins each one.

package propose

import (
	"fmt"
	"sort"
	"strings"

	"gdoc/internal/docs"
)

// Placement is where one block proposal goes, computed from the read the write
// is built from.
//
// Nothing stores one. A placement is a set of indexes in one revision of one
// document, and an index from a read a minute ago is the hazard the whole Docs
// API has, so it lives as long as the call that built it.
// TestAPlacementIsComputedFromTheReadAndNotStored is the pin.
type Placement struct {
	// Tab is the tab every index below is counted in. A document with more
	// than one tab never reaches here: FindSpan refuses it.
	Tab string
	// Insert is where the block's text goes in: the start of the paragraph
	// after the anchor, or for a replace the start of the first paragraph it
	// covers. With AtEnd it is the position of the body's final newline.
	Insert int
	// Delete is the whole paragraphs a replace takes out, from the start of
	// the first to the end of the last, marks included. It is the zero Range
	// on an after block, and Replaces says which.
	Delete docs.Range
	// AtEnd says there was no paragraph to go in front of, because the anchor
	// is the document's last one. The text then goes in before the body's
	// final newline, as a newline and the content without its trailing one, so
	// the block's last paragraph owns that final mark: MEASURED.md row 7.
	AtEnd bool
	// Paragraphs is what the document says today where the block is going: the
	// anchor paragraph for an after block, and every paragraph a replace
	// covers, each without its paragraph mark. It is what the preview read-back
	// asks about, so it holds the written text and leaves out a pending
	// insertion, which the preview does not show.
	Paragraphs []string
}

// Replaces says whether this placement takes paragraphs out as well as putting
// paragraphs in.
func (p Placement) Replaces() bool { return p.Delete.End > p.Delete.Start }

// PlaceAfter is where a block goes that follows a quoted paragraph.
//
// The text goes in at the start of the paragraph behind the anchor, never at
// the anchor's end: an end insert hands the anchor's old paragraph mark to the
// last new paragraph, and restyling that mark is a second suggestion, which is
// one withdraw could not take back whole. MEASURED.md holds both shapes.
//
// content is the block already read by ParseContent. It is needed here because
// the last position in a document is the one place the block's own shape
// decides whether it can be placed at all.
func PlaceAfter(d *docs.Document, after string, content []Para) (Placement, error) {
	span, err := FindSpan(d, after)
	if err != nil {
		return Placement{}, err
	}
	tab := d.Tabs[0]
	at, err := paragraphHolding(tab, span, after)
	if err != nil {
		return Placement{}, err
	}
	anchor := tab.Body[at].Paragraph
	place := Placement{Tab: tab.ID, Paragraphs: []string{written(anchor)}}

	if at == len(tab.Body)-1 {
		if err := endCanCarryTheBlock(anchor, content); err != nil {
			return Placement{}, err
		}
		// The body's final newline is the last unit of the last paragraph.
		place.Insert = anchor.EndIndex - 1
		place.AtEnd = true
		return place, nil
	}

	behind := tab.Body[at+1]
	if behind.Paragraph == nil {
		return Placement{}, fmt.Errorf(
			"the element behind the paragraph holding %q is %s, and a block goes in at the start of a paragraph; quote a paragraph with a paragraph behind it",
			after, whatItIs(behind))
	}
	if behind.Paragraph.StartIndex != anchor.EndIndex {
		return Placement{}, fmt.Errorf(
			"the paragraph behind the one holding %q does not begin where that paragraph ends, so something this read does not index, a section break among them, sits between them; quote a paragraph with a paragraph directly behind it",
			after)
	}
	place.Insert = behind.Paragraph.StartIndex
	return place, nil
}

// PlaceReplace is where a block goes that stands in for a run of paragraphs.
//
// The run is whole paragraphs, from the start of the one holding from to the
// end of the one holding to, paragraph marks included. A partial paragraph
// would mean an accept merges what is left of it with a neighbour, which is a
// change nobody proposed.
func PlaceReplace(d *docs.Document, from, to string) (Placement, error) {
	fromSpan, err := FindSpan(d, from)
	if err != nil {
		return Placement{}, err
	}
	toSpan, err := FindSpan(d, to)
	if err != nil {
		return Placement{}, err
	}
	tab := d.Tabs[0]
	first, err := paragraphHolding(tab, fromSpan, from)
	if err != nil {
		return Placement{}, err
	}
	last, err := paragraphHolding(tab, toSpan, to)
	if err != nil {
		return Placement{}, err
	}
	if last < first || (last == first && toSpan.Start < fromSpan.Start) {
		return Placement{}, fmt.Errorf(
			"the quoted %q comes before %q in the document, and a replace runs from replace_from to replace_to; swap the two quotes", to, from)
	}
	if last == len(tab.Body)-1 {
		return Placement{}, fmt.Errorf(
			"the quoted %q is in the document's last paragraph, and Docs refuses to delete a document's final newline; leave the last paragraph out of the run, or change its words with a words proposal", to)
	}
	for at := first; at <= last; at++ {
		if tab.Body[at].Paragraph == nil {
			return Placement{}, fmt.Errorf(
				"the run from %q to %q covers %s, and a block replaces whole paragraphs and nothing else; quote a run of paragraphs",
				from, to, whatItIs(tab.Body[at]))
		}
		if at > first && tab.Body[at].Paragraph.StartIndex != tab.Body[at-1].Paragraph.EndIndex {
			return Placement{}, fmt.Errorf(
				"the run from %q to %q covers something this read does not index, a section break among them, so the deletion would take that with it; quote a run of paragraphs with nothing between them",
				from, to)
		}
		if what := whatAParagraphHolds(tab, tab.Body[at].Paragraph); what != "" {
			return Placement{}, fmt.Errorf(
				"the run from %q to %q covers a paragraph holding %s, which no read-back here can see, so the deletion would take that with the words; leave that paragraph out of the run",
				from, to, what)
		}
	}

	start := tab.Body[first].Paragraph.StartIndex
	end := tab.Body[last].Paragraph.EndIndex
	if err := nobodyElsesWork(d, tab, first, last, start, end); err != nil {
		return Placement{}, err
	}

	place := Placement{
		Tab:    tab.ID,
		Insert: start,
		Delete: docs.Range{Tab: tab.ID, Start: start, End: end},
	}
	for at := first; at <= last; at++ {
		place.Paragraphs = append(place.Paragraphs, written(tab.Body[at].Paragraph))
	}
	return place, nil
}

// paragraphHolding is the body block the span sits in.
//
// A span this cannot find is a span inside a table, because a table cell is the
// only place besides a body paragraph that the index walk reads text from. A
// block inside a table is out of scope while docs/backlog/propose-inside-tables.md
// is unsettled, so it is refused by name rather than placed against the table's
// own indexes.
func paragraphHolding(tab docs.Tab, span docs.Range, quoted string) (int, error) {
	for at, b := range tab.Body {
		p := b.Paragraph
		if p != nil && span.Start >= p.StartIndex && span.End <= p.EndIndex {
			return at, nil
		}
	}
	return 0, fmt.Errorf(
		"the quoted text %q is inside a table, and a block is not proposed inside one; propose the paragraphs around the table, and change a cell's words with a words proposal", quoted)
}

// endCanCarryTheBlock is the three refusals the last position in a document has.
//
// There is no paragraph to go in front of there, so the text goes in before the
// body's final newline and the block's last paragraph inherits that mark. All
// three refusals are about that one mark, and each names what would work
// instead. Any of them can be widened once somebody measures it, which is
// Nail's call.
//
// The mark keeps whatever the anchor states, and the batch restates nothing on
// it, so the block's own last paragraph has to want exactly what the anchor
// already has: plain body text, and no bullet. Those are the first two. The
// third is the anchor's own named style, and it is the same rule read from the
// document's side: after a heading the mark is a heading, and a block ending in
// body text would either arrive as a heading or need that mark restated, which
// is the unmeasured second-id case.
func endCanCarryTheBlock(anchor *docs.Paragraph, content []Para) error {
	if len(content) == 0 {
		return fmt.Errorf("the block carries no paragraphs, and a block proposes paragraphs")
	}
	if last := content[len(content)-1]; last.Style != NormalStyle || last.List != NotAList {
		return fmt.Errorf(
			"the quoted text is in the document's last paragraph, so the block's own last paragraph would own the document's final mark, and restyling that mark is a second suggestion this milestone did not measure; end the block with a plain paragraph, or place it after an earlier paragraph")
	}
	if anchor.Bullet != nil {
		return fmt.Errorf(
			"the document's last paragraph is a list item, so the block would take its bullet, and clearing that bullet on the paragraph that owns the final mark is the measured case that came back with two suggestion ids; place the block after an earlier paragraph, or add a plain paragraph at the end of the document first")
	}
	// An unstated style is the document saying nothing, which reads as body
	// text, so it is left alone rather than refused.
	if anchor.Style != "" && anchor.Style != NormalStyle {
		return fmt.Errorf(
			"the document's last paragraph is %s, so the block's last paragraph would own the document's final mark and arrive in that style, and restyling that mark is a second suggestion this milestone did not measure; place the block after an earlier paragraph, or add a plain paragraph at the end of the document first",
			anchor.Style)
	}
	return nil
}

// nobodyElsesWork refuses a replace whose paragraphs hold a pending suggestion
// or a comment's anchor, and names every id it found.
//
// The deletion would take somebody's pending edit or the words a colleague
// asked about with it, and the binary does not decide whether that matters: it
// reports what is there and stops. The skills read the same two facts before
// they ask, and tell the person what is in the way.
func nobodyElsesWork(d *docs.Document, tab docs.Tab, first, last, start, end int) error {
	seen := map[string]bool{}
	for at := first; at <= last; at++ {
		for _, r := range tab.Body[at].Paragraph.Runs {
			for _, id := range r.InsertionIDs {
				seen[id] = true
			}
			for _, id := range r.DeletionIDs {
				seen[id] = true
			}
		}
	}
	suggestions := sorted(seen)

	var comments []string
	for id, r := range d.CommentRanges {
		if r.Tab == tab.ID && r.Start < end && r.End > start {
			comments = append(comments, id)
		}
	}
	sort.Strings(comments)

	if len(suggestions) == 0 && len(comments) == 0 {
		return nil
	}
	var holds []string
	if len(suggestions) > 0 {
		holds = append(holds, plural("a pending suggestion", "pending suggestions", suggestions))
	}
	if len(comments) > 0 {
		holds = append(holds, plural("a comment", "comments", comments))
	}
	return fmt.Errorf(
		"the paragraphs this replace would delete hold work that is not gdoc's: %s; settle those first, or quote a run of paragraphs that does not hold them",
		strings.Join(holds, ", and "))
}

// plural names the ids after the word for how many of them there are.
func plural(one, many string, ids []string) string {
	word := one
	if len(ids) > 1 {
		word = many
	}
	return word + " (" + strings.Join(ids, ", ") + ")"
}

// sorted is the keys of a set, in order, so one document refuses the same way
// twice.
func sorted(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// whatAParagraphHolds names the first thing a paragraph carries that is not
// written text, or the empty string when it carries nothing but words. There
// are two halves of that: what sits in the text, and what floats beside it.
//
// TestPlaceRefusesWhatItCannotPlaceAndNamesIt holds both.
func whatAParagraphHolds(tab docs.Tab, p *docs.Paragraph) string {
	if what := whatARunIs(p); what != "" {
		return what
	}
	return whatFloatsOn(tab, p)
}

// whatFloatsOn names the first floating object anchored to a paragraph, or the
// empty string when none is.
//
// A floating object is anchored to a paragraph rather than held in its runs, so
// whatARunIs is blind to it and so is every read-back here: written reads text
// runs only, and idSet.deletionIn asks the same runs whether they carry a
// deletion id. Docs takes an anchored object with the paragraph it is anchored
// to all the same, so the run is refused for the reason an inline picture is
// refused, and in the same words.
func whatFloatsOn(tab docs.Tab, p *docs.Paragraph) string {
	for _, id := range p.Positioned {
		switch tab.Positioned[id].Kind {
		case docs.KindImage:
			return "a floating picture"
		case docs.KindDrawing:
			return "a floating drawing"
		default:
			return "a floating object"
		}
	}
	return ""
}

// whatARunIs names the first run of a paragraph that is not written text, or the
// empty string when every run is.
//
// A replace deletes whole paragraphs by their own start and end, so an inline
// picture, an equation, a footnote mark, a page break or a chip sitting inside
// one of them is inside the deleteContentRange whether the caller meant it or
// not. None of the three read-backs can see it either, by construction: written
// and idSet.deletionIn both read text runs only, so the preview check compares
// words that never mentioned it and the inline check never asks whether it
// carries a deletion id. The hazard is the words kind's crossing rule one level
// up, where FindSpan refuses a quote running across the same content for the
// same reason, so a paragraph holding any of it is refused by name rather than
// deleted unseen.
//
// TestPlaceRefusesWhatItCannotPlaceAndNamesIt holds the picture case.
func whatARunIs(p *docs.Paragraph) string {
	for _, r := range p.Runs {
		switch r.Kind {
		case docs.KindText:
			continue
		case docs.KindImage:
			return "a picture"
		case docs.KindDrawing:
			return "a drawing"
		case docs.KindEquation:
			return "an equation"
		case docs.KindFootnoteRef:
			return "a footnote mark"
		case docs.KindPageBreak:
			return "a page break"
		case docs.KindColumnBreak:
			return "a column break"
		case docs.KindHorizontalRule:
			return "a horizontal rule"
		case docs.KindAutoText:
			return "a field Docs fills in"
		case docs.KindPerson:
			return "a person chip"
		case docs.KindDate:
			return "a date chip"
		case docs.KindRichLink:
			return "a link chip"
		case docs.KindObject:
			return "an embedded object"
		default:
			return "content this read does not name"
		}
	}
	return ""
}

// whatItIs names a body element in the words a person reading the refusal would
// use. Only a table and the contents list Docs generates reach it: a paragraph
// is what the caller wanted, and everything else the read drops.
func whatItIs(b docs.Block) string {
	switch {
	case b.Table != nil:
		return "a table"
	case b.TOC != nil:
		return "the contents list Docs generates"
	default:
		return "not a paragraph"
	}
}

// written is a paragraph as the document reads it today with pending
// suggestions hidden, without its paragraph mark.
//
// A suggested insertion is left out because the preview does not show one, and
// this text is what the preview read-back asks about. A suggested deletion is
// kept, because the words are still in the document until somebody accepts it.
func written(p *docs.Paragraph) string {
	var b strings.Builder
	for _, r := range p.Runs {
		if r.Kind != docs.KindText || len(r.InsertionIDs) > 0 {
			continue
		}
		b.WriteString(r.Text)
	}
	return strings.TrimRight(b.String(), "\n")
}
