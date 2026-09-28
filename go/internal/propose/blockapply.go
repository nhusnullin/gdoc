// This file is the block kind end to end: its shape rule, and the one write it
// makes. block.go reads its content, place.go says where it goes, blockbatch.go
// builds its requests, blockverify.go reads it back three ways, propose.go holds
// the proposal itself and the words kind's own write, and doc.go states each
// rule with the test that pins it.

package propose

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gdoc/internal/docs"
)

// checkBlock is the block kind's shape rule, asked of the proposal alone.
//
// This is the only place the placement form is checked. PlaceAfter and
// PlaceReplace each take the quotes they need, so a block naming neither form
// would reach one of them as an empty quote and be refused in words about a
// quote that is not in the document, which names nothing the author did wrong.
//
// The content is checked for its shape here and read for its meaning in
// ApplyBlock, because ParseContent needs no document either: both run before the
// probe and before the first write.
func (p Proposal) checkBlock() error {
	if err := p.checkPlacement(); err != nil {
		return err
	}
	if p.Quoted != "" || p.Replacement != "" {
		return fmt.Errorf(
			"the block carries quoted or replacement, which belong to the words kind; a block names where it goes with after, or with replace_from and replace_to, and what it says with content")
	}
	if strings.TrimSpace(p.Content) == "" {
		return errors.New("the block carries no content, and a block proposes paragraphs")
	}
	// The ceiling is the guard's rather than Docs'. A body past the guard's peek
	// is refused there as one that could not be read, which is nothing a person
	// writing a block can act on, so it is refused here in words about what a
	// block is for. MaxContent says how the number was measured.
	if len(p.Content) > MaxContent {
		return fmt.Errorf(
			"the block's content is %d bytes, and a block proposes a section rather than a document; keep it under %d bytes, or publish the note as its own document",
			len(p.Content), MaxContent)
	}
	return p.checkWhy()
}

// checkPlacement is the one form rule: a block goes after one paragraph, or in
// place of the run from the first quote to the second, and never both.
func (p Proposal) checkPlacement() error {
	hasAfter := p.After != ""
	hasFrom := p.ReplaceFrom != ""
	hasTo := p.ReplaceTo != ""
	switch {
	case !hasAfter && !hasFrom && !hasTo:
		return errors.New(
			"the block names neither after nor replace_from and replace_to, and a block goes after a paragraph or in place of a run of them")
	case hasAfter && (hasFrom || hasTo):
		return errors.New(
			"the block names after and a replace together, and it goes either after a paragraph or in place of a run of them; name one of the two")
	case hasFrom && !hasTo:
		return errors.New(
			"the block names replace_from and no replace_to, and a replace runs from the first paragraph to the last; quote a few words of each")
	case hasTo && !hasFrom:
		return errors.New(
			"the block names replace_to and no replace_from, and a replace runs from the first paragraph to the last; quote a few words of each")
	}
	return nil
}

// ApplyBlock places one block proposal: read, place, write once, verify three
// ways. It is the same sequence Apply makes for the words kind, over the block's
// own placement and its own batch.
//
// The read comes first and the write is built from it, so no index outlives the
// answer it was computed in. Two proposals are two calls, each with its own
// read, because the first one moves the ground under the second.
//
// It fails only before the write. After the batch has gone out the block is in
// the document, and everything from there is reported rather than raised.
func ApplyBlock(ctx context.Context, s Session, docID string, p Proposal) (Result, error) {
	out := Result{After: p.After, ReplaceFrom: p.ReplaceFrom, ReplaceTo: p.ReplaceTo}
	if err := p.Check(); err != nil {
		return out, err
	}
	// The content is read before the document, because it needs no document: a
	// block gdoc cannot read is refused without a request being made at all.
	content, err := ParseContent(p.Content)
	if err != nil {
		return out, err
	}
	d, err := readOneTab(ctx, s, docID)
	if err != nil {
		return out, err
	}
	place, err := placeBlock(d, p, content)
	if err != nil {
		return out, err
	}
	if err := send(ctx, s, docID, BlockBatch(place, content, p.Why, p.Assignee), &out); err != nil {
		return out, err
	}

	checks, ids, warns := VerifyBlock(ctx, s, docID, place, content, Prefix+p.Why)
	out.Checks = checks
	out.SuggestionIDs = ids
	out.Warnings = append(out.Warnings, warns...)
	out.Verified = checks.all() && out.CommentUpdateState == stateAllSaved
	return out, nil
}

// placeBlock is where the block goes, by whichever form it names. Check has
// already required exactly one of them, so the two arms are the whole of it.
func placeBlock(d *docs.Document, p Proposal, content []Para) (Placement, error) {
	if p.After != "" {
		return PlaceAfter(d, p.After, content)
	}
	return PlaceReplace(d, p.ReplaceFrom, p.ReplaceTo)
}
