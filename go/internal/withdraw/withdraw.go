// Package withdraw retracts one of gdoc's own pending suggestions, and proves
// it is gone before saying so.
//
// The permission is provenance, and nothing else. Nothing in a suggestion id
// says who wrote it, and the Docs API will happily delete anybody's, so the
// only record gdoc has of its own work is the proposals[] list propose wrote
// into the note's front matter. A suggestion missing from that list is somebody
// else's, and this package refuses it before it reads the document.
//
// The write itself is a delete over gdoc's own insertion, in SUGGEST mode like
// every other write in this milestone. Two facts have to hold before the
// retraction is Verified: the answer names the suggestion in
// deletedSuggestionIds, and a fresh read carries no run under that id. The
// first is Docs agreeing with itself, and the second is the only route that can
// say the suggestion has actually left the document.
//
// Nothing here decides whether a proposal should be withdrawn. The id arrives
// chosen, and this package checks that it is gdoc's, deletes it, and reports
// what it saw.
package withdraw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gdoc/internal/docs"
	"gdoc/internal/frontmatter"
	"gdoc/internal/propose"
)

// Result is what one withdrawal left behind.
//
// Warnings carry the reasons Verified is false, as they do in propose and
// reply, and for the same reason: everything that goes wrong after the write is
// a fact about a document that has already changed, and a caller told the run
// failed is a caller that sends the delete a second time.
type Result struct {
	SuggestionID         string   `json:"suggestion_id"`
	DeletedSuggestionIDs []string `json:"deleted_suggestion_ids,omitempty"`
	Verified             bool     `json:"verified"`
	Warnings             []string `json:"warnings,omitempty"`
}

// Session is what this package needs of a session: the read the span comes
// from, and the one write. Naming the interface here keeps net/http out of this
// room, which is what the boundary test asks of every package but the four that
// build requests.
type Session interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	PostJSON(ctx context.Context, rawURL string, body any, into any) error
}

// BatchURL is the Docs write path, spelled once. It is propose's, because the
// two packages write through the same endpoint and two copies of one URL is two
// places for it to drift.
func BatchURL(docID string) string { return propose.BatchURL(docID) }

// Mine says whether the note records this suggestion as one of gdoc's own.
//
// It is the whole permission model, in one function so that the command layer
// can ask the same question this package asks. A note that is missing, or that
// never mentioned the id, both answer no: not knowing whose suggestion it is
// must never resolve to deleting it.
func Mine(note *frontmatter.Block, suggestionID string) bool {
	if note == nil {
		return false
	}
	for _, p := range note.Proposals {
		if p.ID == suggestionID {
			return true
		}
	}
	return false
}

// Run retracts one suggestion: check the note, read the document, delete the
// span the suggestion holds, and read it back.
//
// It fails only before the write. After the batch has gone out the document has
// changed, and everything from there is reported rather than raised, the
// failures raised after Docs answered included: see sentAnyway.
func Run(ctx context.Context, s Session, docID, suggestionID string, note *frontmatter.Block) (Result, error) {
	out := Result{SuggestionID: suggestionID}
	if suggestionID == "" {
		return out, fmt.Errorf("no suggestion was named to withdraw")
	}
	if !Mine(note, suggestionID) {
		return out, fmt.Errorf("the suggestion %q is not one of gdoc's own proposals in this note, and gdoc withdraws only what it proposed", suggestionID)
	}

	d, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		return out, fmt.Errorf("the document could not be read before withdrawing: %w", err)
	}
	if d.MultiTab() {
		// The delete names one range, and a range means nothing without saying
		// which tab it is in. Refusing is the honest answer until a milestone
		// decides how a write names a tab.
		return out, fmt.Errorf("the document has %d tabs, and a suggestion is withdrawn from a document with one", len(d.Tabs))
	}
	r, err := Span(d, suggestionID)
	if err != nil {
		return out, err
	}

	var answer batchAnswer
	answerRead := true
	var sentErr error
	if err := s.PostJSON(ctx, BatchURL(docID), json.RawMessage(Batch(r)), &answer); err != nil {
		if !sentAnyway(err) {
			return out, fmt.Errorf("the suggestion %q could not be withdrawn: %w", suggestionID, err)
		}
		// Docs took the delete and the answer could not be read. Raising here
		// would report a document that has already changed as one that has not,
		// and send the skill back to send the delete again. The read-back below
		// is what says which it was.
		//
		// Whatever did decode is kept, the way propose keeps a comment id it
		// was given. On this path the answer is usually empty, so
		// deletedSuggestionIds carries nothing and Verified stays false: that
		// list is one of the two facts it rests on. The exception is valid JSON
		// of the wrong shape, where encoding/json saves the type error and
		// keeps going, and the list the server really did send is the server's
		// own word about what it retracted. Throwing that away would report a
		// withdrawal both facts confirm as unverified.
		//
		// So the warning is built after Verified is known rather than assuming
		// the worst here: telling the caller to take an entry out of the note
		// that this same run is about to remove itself is the envelope saying
		// two opposite things.
		answerRead = false
		sentErr = err
	}
	out.DeletedSuggestionIDs = answer.deletedIDs()
	if (answerRead || len(out.DeletedSuggestionIDs) > 0) && !contains(out.DeletedSuggestionIDs, suggestionID) {
		// Text was deleted and no suggestion was retracted, which is what a
		// direct edit looks like. The read-back below says which it was.
		//
		// The gate is the decoded list, never the path that produced it, which
		// is the rule propose reads commentUpdateState by. An answer that could
		// not be read is asked only when a list decoded anyway: an empty list
		// there is the usual case and says nothing, while a list the server
		// really did send is its own word about what it retracted, and a list
		// that names another suggestion is the alarm whichever path carried it.
		out.Warnings = append(out.Warnings, fmt.Sprintf(
			"the write did not name %q in deletedSuggestionIds, so Docs deleted the words without saying it retracted the suggestion",
			suggestionID))
	}

	gone, warn := goneFrom(ctx, s, docID, suggestionID)
	if warn != "" {
		out.Warnings = append(out.Warnings, warn)
	}
	out.Verified = gone && contains(out.DeletedSuggestionIDs, suggestionID)
	if sentErr != nil {
		msg := fmt.Sprintf("the delete of %q was accepted by Docs and its answer could not be read", suggestionID)
		if out.Verified {
			msg += ", though the part of it that decoded named the suggestion in deletedSuggestionIds and the read-back found it gone"
		} else {
			msg += ", so the withdrawal cannot be confirmed and the note still records the proposal; check the document, and take the entry out by hand once the suggestion is gone"
		}
		out.Warnings = append(out.Warnings, msg+": "+sentErr.Error())
	}
	return out, nil
}

// sentAnyway says whether the delete reached Docs in spite of the error. The
// session marks the failures raised after the server accepted a request, and
// this room asks by behaviour rather than by importing that package: naming a
// Session interface here is what keeps net/http out, and an imported sentinel
// would bring it back through the side door. propose and reply each ask the
// same question the same way.
func sentAnyway(err error) bool {
	var sent interface{ Sent() bool }
	return errors.As(err, &sent) && sent.Sent()
}

// goneFrom is the read-back: a fresh document carries no run under the id.
//
// A read that could not be made says nothing about the document, so it is a
// warning and a false answer, never a true one.
func goneFrom(ctx context.Context, s Session, docID, suggestionID string) (bool, string) {
	d, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		return false, fmt.Sprintf("the delete was sent and the document could not be read back, so %q could not be confirmed as gone: %v", suggestionID, err)
	}
	for _, t := range d.Tabs {
		for _, run := range runs(t.Body) {
			if carries(run, suggestionID) {
				return false, fmt.Sprintf("the read-back still carries a run under %q, so the suggestion is still pending in the document", suggestionID)
			}
		}
	}
	return true, ""
}

// Span is the range the suggestion's inserted words hold, in the indexes of the
// one tab.
//
// Only the insertion side is deleted. The suggested deletion beside it is the
// original words, still written, and Docs retracts the whole suggestion when
// its insertion goes: taking the deletion side too would remove text that is
// really in the document.
//
// The runs have to be one span. Docs cuts one insert into as many runs as it
// likes, and those are adjacent, but two spans with somebody else's words
// between them are two spans: deleting from the first to the last would take
// those words with it, and they are not gdoc's to touch.
func Span(d *docs.Document, suggestionID string) (docs.Range, error) {
	for _, t := range d.Tabs {
		var found []docs.Run
		for _, r := range runs(t.Body) {
			if inserted(r, suggestionID) {
				found = append(found, r)
			}
		}
		if len(found) == 0 {
			continue
		}
		for i := 1; i < len(found); i++ {
			if found[i].StartIndex != found[i-1].EndIndex {
				return docs.Range{}, fmt.Errorf(
					"the suggestion %q holds %d separate spans in the document, and deleting from the first to the last would take the words between them as well",
					suggestionID, len(found))
			}
		}
		return docs.Range{Tab: t.ID, Start: found[0].StartIndex, End: found[len(found)-1].EndIndex}, nil
	}
	return docs.Range{}, fmt.Errorf(
		"the document carries no pending suggested insertion under %q; it may have been accepted, rejected or already withdrawn", suggestionID)
}

// Batch is the whole write: one deleteContentRange, in SUGGEST mode.
//
// SUGGEST is what the guard requires of any batchUpdate on a handed-in
// document, and it is right here for the same reason it is right in propose:
// gdoc says what it intends, and the read-back says what Google did with it.
func Batch(r docs.Range) []byte {
	body := map[string]any{
		"requests": []any{
			map[string]any{"deleteContentRange": map[string]any{
				"range": map[string]any{"startIndex": r.Start, "endIndex": r.End},
			}},
		},
		"writeControl": map[string]any{"writeMode": "SUGGEST"},
	}
	// The body is built from a map with no cycles and no unsupported types, so
	// the marshal cannot fail. Returning bytes rather than a map keeps the
	// caller from being able to change what the guard already judged.
	raw, _ := json.Marshal(body)
	return raw
}

// Forget returns a new block without the suggestion's entry, leaving the rest
// of the note as it was.
//
// It copies rather than edits. The caller writes the note only when the
// withdrawal held, so a Forget that changed the block in place would drop the
// provenance of a suggestion that is still sitting in the document, and gdoc
// would then refuse to withdraw its own work.
func Forget(note *frontmatter.Block, suggestionID string) *frontmatter.Block {
	if note == nil {
		return nil
	}
	next := *note
	next.Proposals = nil
	for _, p := range note.Proposals {
		if p.ID != suggestionID {
			next.Proposals = append(next.Proposals, p)
		}
	}
	return &next
}

// batchAnswer is what came back from the write.
//
// deletedSuggestionIds is measured rather than documented (DECISIONS.md,
// 2026-08-29): a deleteContentRange in SUGGEST mode over gdoc's own insertion
// answers with it. It has been seen at the top of the answer, and a per-request
// reply is where the rest of the API puts what a request produced, so both are
// read. Reporting a withdrawal as unconfirmed because the id arrived one level
// down would send somebody looking for a suggestion that is already gone.
type batchAnswer struct {
	DocumentID           string   `json:"documentId"`
	DeletedSuggestionIDs []string `json:"deletedSuggestionIds"`
	Replies              []struct {
		DeletedSuggestionIDs []string `json:"deletedSuggestionIds"`
		DeleteContentRange   *struct {
			DeletedSuggestionIDs []string `json:"deletedSuggestionIds"`
		} `json:"deleteContentRange"`
	} `json:"replies"`
	WriteControl struct {
		RequiredRevisionID string `json:"requiredRevisionId"`
	} `json:"writeControl"`
}

// deletedIDs is every suggestion the answer says it retracted, from wherever it
// put them, in the order they were read and without repeats.
func (a batchAnswer) deletedIDs() []string {
	var out []string
	seen := map[string]bool{}
	add := func(ids []string) {
		for _, id := range ids {
			if id != "" && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	add(a.DeletedSuggestionIDs)
	for _, r := range a.Replies {
		add(r.DeletedSuggestionIDs)
		if r.DeleteContentRange != nil {
			add(r.DeleteContentRange.DeletedSuggestionIDs)
		}
	}
	return out
}

// inserted says whether the run is text this suggestion put there.
func inserted(r docs.Run, suggestionID string) bool {
	return r.Kind == docs.KindText && contains(r.InsertionIDs, suggestionID)
}

// carries says whether the run belongs to the suggestion at all, on either
// side. The read-back asks this rather than asking about the insertion alone:
// a run still marked for deletion under the id is the other half of a proposal
// that is still pending.
func carries(r docs.Run, suggestionID string) bool {
	return contains(r.InsertionIDs, suggestionID) || contains(r.DeletionIDs, suggestionID)
}

func contains(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

// runs is every text run of a body in reading order, tables walked into. A
// cell's text carries its own indexes in the same tab, so a suggestion inside
// one is a suggestion gdoc can withdraw.
func runs(bs []docs.Block) []docs.Run {
	var out []docs.Run
	for _, b := range bs {
		if b.Paragraph != nil {
			for _, r := range b.Paragraph.Runs {
				if r.Kind == docs.KindText {
					out = append(out, r)
				}
			}
			continue
		}
		for _, row := range b.Table {
			for _, c := range row {
				out = append(out, runs(c.Blocks)...)
			}
		}
	}
	return out
}
