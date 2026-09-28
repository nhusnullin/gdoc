// This file is the three read-backs one block proposal gets: the block pending
// inline as one suggestion, the preview view still reading as the document did,
// and the robot comment anchored in the export. verify.go holds the words kind's
// own three, blockbatch.go holds the write these questions are asked about, and
// doc.go states each rule with the test that pins it.

package propose

import (
	"context"
	"fmt"
	"strings"

	"gdoc/internal/docs"
)

// VerifyBlock reads one block proposal back through the same three routes as
// Verify, and asks each of them the block's own question.
//
// It takes the placement and the paragraphs rather than a proposal, because the
// question each route asks is about what the write did: where the block went,
// what its paragraphs say, and which old paragraphs a replace marked. The
// proposal that asked for it adds nothing here.
//
// Like Verify it never fails. Each route answers for itself, a route that could
// not be read is a warning naming it, and the caller reports the checks as they
// came: a verification that raised would hide a block that is already in the
// document.
//
// The caller has content that ParseContent read, so there is at least one
// paragraph in it, which is what layOutBlock and the first line below rest on.
func VerifyBlock(ctx context.Context, s Session, docID string, place Placement, content []Para, commentBody string) (Checks, []string, []string) {
	var checks Checks
	var warns []string

	inline, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		warns = append(warns, fmt.Sprintf("the document could not be read back with suggestions inline, so the block could not be confirmed as pending: %v", err))
	}
	var ids []string
	if inline != nil {
		var why string
		checks.SuggestionsInline, ids, why = blockInlineHolds(inline, place, content)
		if why != "" {
			warns = append(warns, why)
		}
		if len(ids) > 1 {
			// One id is what a verified block means. Every request of the batch
			// folded into one when it was measured, the note records one, and
			// withdraw takes back the one it recorded, so a block that came
			// back as two suggestions is one gdoc cannot take back whole. The
			// route is false and both ids are reported: the change is in the
			// document either way, and the ids are how somebody finishes the
			// job by hand.
			checks.SuggestionsInline = false
			warns = append(warns, fmt.Sprintf(
				"the block came back as %d suggestions (%s) rather than one, and withdraw takes back the first of them, which is the one the note records; the rest stay in the document until somebody rejects them there",
				len(ids), strings.Join(ids, ", ")))
		}
	}

	preview, err := fetchPreview(ctx, s, docID)
	if err != nil {
		warns = append(warns, fmt.Sprintf("the document could not be read back in the preview view, so the block could not be told from a direct edit: %v", err))
	} else {
		var why string
		checks.PreviewWithoutSuggestions, why = blockPreviewHolds(preview, place, content)
		if why != "" {
			warns = append(warns, why)
		}
	}

	anchored, why := docxHolds(ctx, s, docID, commentBody)
	checks.DocxAnchored = anchored
	if why != "" {
		warns = append(warns, why)
	}
	return checks, ids, warns
}

// blockInlineHolds is the first check: every new paragraph is there carrying a
// suggested insertion, and for a replace every paragraph it stands in for is
// there carrying a suggested deletion.
//
// Both halves matter, and for the same reason the words kind asks both: an
// insertion alone would be the block added beside the paragraphs it replaces,
// and a deletion alone would be those paragraphs struck out with nothing put in
// their place.
//
// The paragraphs are found at the indexes the write's own layout computed, which
// is the one thing the read-back and the batch must agree about. The ids are
// every suggestion id met on the way, in that order, so the caller can say how
// many the block became.
func blockInlineHolds(d *docs.Document, place Placement, content []Para) (bool, []string, string) {
	paras := bodyParagraphs(d, place.Tab)
	l := layOutBlock(place, content)
	met := &idSet{}

	for i, c := range content {
		at := l.Paras[i].Start
		p := paraAt(paras, at)
		if p == nil {
			return false, met.ids, fmt.Sprintf(
				"the read-back carries no paragraph beginning at index %d, where the block's paragraph %d of %d was written", at, i+1, len(content))
		}
		// Every new paragraph carries its own mark, except the last one of a
		// block placed after the document's last paragraph: there the text went
		// in before the body's final newline, so that newline is the document's
		// own and is not part of the insertion. MEASURED.md row 7.
		want := c.Text() + "\n"
		if place.AtEnd && i == len(content)-1 {
			want = c.Text()
		}
		if got := met.insertionIn(p); got != want {
			return false, met.ids, fmt.Sprintf(
				"the read-back carries %q as the suggested insertion at index %d, where %q was written", got, at, want)
		}
	}
	if !place.Replaces() {
		return true, met.ids, ""
	}

	// The old paragraphs are where the insert left them: it went in at the start
	// of the first of them, so both ends moved along by its length. It is the
	// same shift blockbatch.go computed the deleteContentRange at.
	shift := utf16Len(l.Text)
	start, end := place.Delete.Start+shift, place.Delete.End+shift
	var found int
	for _, p := range paras {
		if p.StartIndex < start || p.EndIndex > end {
			continue
		}
		found++
		if !met.deletionIn(p) {
			return false, met.ids, fmt.Sprintf(
				"the read-back carries the paragraph %q at index %d with no suggested deletion on it, where the replace marked it for deletion", written(p), p.StartIndex)
		}
	}
	if found != len(place.Paragraphs) {
		return false, met.ids, fmt.Sprintf(
			"the read-back carries %d paragraphs between the indexes %d and %d, where the replace covered %d of them", found, start, end, len(place.Paragraphs))
	}
	return true, met.ids, ""
}

// blockPreviewHolds is the second check, and the only route that can tell a
// suggestion from a direct edit: the document with every pending suggestion
// hidden still reads the way it read before the write.
//
// Two questions, and the second is the one an after block rests on. The
// paragraphs the placement stood on have to still be there, because a direct
// edit of a replace takes them out, and a suggested deletion leaves them.
// Then the block's first line must not be there, because a direct edit of an
// after block leaves the anchor alone and puts the new paragraphs in as written
// text.
//
// The second question is asked by words, like the words kind's, and that is why
// it can only ever be a false answer rather than an accusation: the line may be
// in the preview because the block was written as an edit, or because the
// document already carried that line somewhere else. Those two cannot be told
// apart from here, so the check is false with a warning naming both readings.
// Passing instead would be the wrong mistake: it would report the silent direct
// edit, which is the one thing this route exists to catch, as a route that held.
func blockPreviewHolds(d *docs.Document, place Placement, content []Para) (bool, string) {
	for _, p := range place.Paragraphs {
		if !Carries(d, place.Tab, p) {
			return false, fmt.Sprintf(
				"the preview view no longer carries the paragraph %q, which is what a direct edit looks like rather than a suggestion", p)
		}
	}
	if first := content[0].Text(); Carries(d, place.Tab, first) {
		return false, fmt.Sprintf(
			"the preview view carries the block's first line %q, so this route gives no answer: the block may have been written as a direct edit rather than a suggestion, or the document may have carried that line before the write", first)
	}
	return true, ""
}

// idSet is the suggestion ids a read-back carried, in the order they were met
// and without repeats.
//
// The order is what makes the first one the first one: the note records one id,
// and it is the id of the block's own first paragraph, so a block that came back
// as two suggestions reports the recorded one first and the stragglers after it.
type idSet struct {
	seen map[string]bool
	ids  []string
}

func (s *idSet) add(ids []string) {
	for _, id := range ids {
		if s.seen == nil {
			s.seen = map[string]bool{}
		}
		if !s.seen[id] {
			s.seen[id] = true
			s.ids = append(s.ids, id)
		}
	}
}

// insertionIn is the paragraph's suggested insertion as text, and it remembers
// every id that text carried. A paragraph Docs cut into several runs is one
// insertion, so the runs are joined rather than counted.
func (s *idSet) insertionIn(p *docs.Paragraph) string {
	var b strings.Builder
	for _, r := range p.Runs {
		if r.Kind != docs.KindText || len(r.InsertionIDs) == 0 {
			continue
		}
		b.WriteString(r.Text)
		s.add(r.InsertionIDs)
	}
	return b.String()
}

// deletionIn says whether every one of the paragraph's words is marked for
// deletion, and remembers the ids that marked them. Every word, because a
// paragraph half struck out is a deletion that would leave somebody's sentence
// cut in two if it were accepted.
func (s *idSet) deletionIn(p *docs.Paragraph) bool {
	words := false
	for _, r := range p.Runs {
		if r.Kind != docs.KindText {
			continue
		}
		words = true
		if len(r.DeletionIDs) == 0 {
			return false
		}
		s.add(r.DeletionIDs)
	}
	return words
}

// bodyParagraphs is one tab's body paragraphs, in reading order.
//
// It does not walk into a table, which is where it parts company with textRuns.
// A block is never placed inside one, because place.go refuses a quote in a
// table cell by name, so a cell's paragraph can only be one this read-back was
// never asking about.
func bodyParagraphs(d *docs.Document, tabID string) []*docs.Paragraph {
	for _, t := range d.Tabs {
		if t.ID != tabID {
			continue
		}
		var out []*docs.Paragraph
		for _, b := range t.Body {
			if b.Paragraph != nil {
				out = append(out, b.Paragraph)
			}
		}
		return out
	}
	return nil
}

// paraAt is the paragraph beginning at one index, or nothing.
func paraAt(paras []*docs.Paragraph, start int) *docs.Paragraph {
	for _, p := range paras {
		if p.StartIndex == start {
			return p
		}
	}
	return nil
}
