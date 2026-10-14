package render

// word/footnotes.xml: Word's two reserved separators, then one note per mark
// the markdown walker numbered.

import (
	"strconv"

	"github.com/beevik/etree"
)

// Footnote is one note under the page: the number the body's own reference run
// names, and the paragraphs the note holds, each set in the FootnoteText
// style. internal/body builds these and aliases this type, so the numbers it
// hands out are the numbers this part is written from and the two cannot
// drift, the way Media already works.
type Footnote struct {
	ID     int
	Blocks []*etree.Element
}

// The two notes Word reserves in every footnotes part, at the two ids no
// numbered note can take: the rule drawn above the notes, and the rule above
// a note continued onto the next page. A part without them is one Word offers
// to repair, so both are written whether or not the document has a footnote.
// A numbered note starts at 1, which is where internal/body starts counting.
const (
	footnoteSeparatorID    = -1
	footnoteContinuationID = 0
)

// footnotesPart writes word/footnotes.xml. The part is always in the package,
// because the relationship and the content type are the shell's and a
// relationship pointing at nothing is a document Word repairs.
func (b *builder) footnotesPart(notes []Footnote) []byte {
	doc, root := newPart("w:footnotes")
	b.separatorNote(root, "separator", footnoteSeparatorID, "w:separator")
	b.separatorNote(root, "continuationSeparator", footnoteContinuationID, "w:continuationSeparator")
	for _, n := range notes {
		note := sub(root, "w:footnote", "w:id", strconv.Itoa(n.ID))
		for _, block := range n.Blocks {
			note.AddChild(block)
		}
	}
	return b.serialise(doc, "word/footnotes.xml")
}

// separatorNote is one of the two reserved notes: a paragraph whose only run
// is the rule Word draws.
//
// The spacing is stated rather than inherited. This document's defaults put
// 3pt above and 6pt below every paragraph at 1.15 line spacing, and a rule is
// not a paragraph of prose: inherited, the separator pushes the first note
// down by nine points nobody asked for. The three numbers are Word's own for
// these two notes.
func (b *builder) separatorNote(root *etree.Element, kind string, id int, mark string) {
	note := sub(root, "w:footnote", "w:type", kind, "w:id", strconv.Itoa(id))
	p := sub(note, "w:p")
	sub(sub(p, "w:pPr"), "w:spacing",
		"w:before", "0", "w:after", "0", "w:line", "240", "w:lineRule", "auto")
	sub(sub(p, "w:r"), mark)
}
