// Package withdraw retracts one of gdoc's own pending suggestions, and proves
// it is gone before saying so.
//
// The permission is provenance, and nothing else. Nothing in a suggestion id
// says who wrote it, and the Docs API will happily reject anybody's, so the
// only record gdoc has of its own work is the proposals[] list propose wrote
// into the note's front matter. A suggestion missing from that list is somebody
// else's, and this package refuses it before it reads the document.
//
// The write is a rejectSuggestion naming the id, in SUGGEST mode like every
// other write in this milestone. Nail's decision, 2026-09-07, and it was
// measured before it was taken (DECISIONS.md, same date): a propose is a
// replace, one suggestion id over a suggested deletion and a suggested
// insertion, and a deleteContentRange over the insertion retracts only that
// half. The quoted words stay suggested-deleted under the same id, Docs answers
// updatedSummarySuggestionIds rather than deletedSuggestionIds, and nothing
// else in the ordinary request kinds takes the other half back. rejectSuggestion
// does, in one request: the document reads as it did before the proposal, the
// answer names the id in suggestionResponses[].rejectedSuggestionIds, and the 🤖
// comment survives, still anchored by id.
//
// The guard carries a rejectSuggestion only for an id the command granted with
// AllowReject, seeded from the same note Mine reads. This package never touches
// the policy: it asks the session to send, and the guard says whether that id
// is one of gdoc's own in this run. Rejecting gdoc's own unaccepted proposal is
// not resolving Nail's decision, because there was no decision yet; the rule
// that gdoc never accepts, rejects or deletes anyone else's suggestion holds,
// and the guard is what holds it.
//
// Two facts have to hold before the retraction is Verified: the answer names
// the suggestion in rejectedSuggestionIds, and a fresh read carries no run under
// that id on either side. The first is Docs agreeing with itself, and the second
// is the only route that can say the suggestion has actually left the document.
//
// Nothing here decides whether a proposal should be withdrawn. The id arrives
// chosen, and this package checks that it is gdoc's, rejects it, and reports
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
// failed is a caller that sends the reject a second time.
type Result struct {
	SuggestionID          string   `json:"suggestion_id"`
	RejectedSuggestionIDs []string `json:"rejected_suggestion_ids,omitempty"`
	Verified              bool     `json:"verified"`
	Warnings              []string `json:"warnings,omitempty"`
}

// Session is what this package needs of a session: the read the check comes
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
// It is the permission question in one function, so the command asks the same
// question this package asks rather than a paraphrase of it. A nil note names
// nothing, and so does a note whose proposals list is empty.
func Mine(note *frontmatter.Block, suggestionID string) bool {
	if note == nil || suggestionID == "" {
		return false
	}
	for _, p := range note.Proposals {
		if p.ID == suggestionID {
			return true
		}
	}
	return false
}

// Run withdraws one suggestion: refuses one the note does not name, reads the
// document to see that the suggestion is still pending, sends the reject, then
// reads again to confirm it is gone.
//
// The document is read first for the same reason propose reads it: the answer
// to "is this still pending" is only true at the moment it is read, and a
// reject of a suggestion that was already accepted or rejected would be
// reported as a withdrawal with nothing withdrawn.
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
	if !pending(d, suggestionID) {
		return out, fmt.Errorf(
			"the document carries no pending suggestion under %q; it may have been accepted, rejected or already withdrawn", suggestionID)
	}

	var answer batchAnswer
	answerRead := true
	var sentErr error
	if err := s.PostJSON(ctx, BatchURL(docID), json.RawMessage(Batch(suggestionID)), &answer); err != nil {
		if !sentAnyway(err) {
			return out, fmt.Errorf("the suggestion %q could not be withdrawn: %w", suggestionID, err)
		}
		// Docs took the reject and the answer could not be read. Raising here
		// would report a document that has already changed as one that has not,
		// and send the skill back to send the reject again. The read-back below
		// is what says which it was.
		//
		// Whatever did decode is kept, the way propose keeps a comment id it
		// was given. On this path the answer is usually empty, so
		// rejectedSuggestionIds carries nothing and Verified stays false: that
		// list is one of the two facts it rests on. The exception is valid JSON
		// of the wrong shape, where encoding/json saves the type error and
		// keeps going, and the list the server really did send is the server's
		// own word about what it rejected. Throwing that away would report a
		// withdrawal both facts confirm as unverified.
		//
		// So the warning is built after Verified is known rather than assuming
		// the worst here: telling the caller to take an entry out of the note
		// that this same run is about to remove itself is the envelope saying
		// two opposite things.
		answerRead = false
		sentErr = err
	}
	out.RejectedSuggestionIDs = answer.rejectedIDs()
	if (answerRead || len(out.RejectedSuggestionIDs) > 0) && !contains(out.RejectedSuggestionIDs, suggestionID) {
		// A 200 that names no rejected suggestion is Docs having done something
		// other than what was asked. The read-back below says what.
		//
		// The gate is the decoded list, never the path that produced it, which
		// is the rule propose reads commentUpdateState by. An answer that could
		// not be read is asked only when a list decoded anyway: an empty list
		// there is the usual case and says nothing, while a list the server
		// really did send is its own word about what it rejected, and a list
		// that names another suggestion is the alarm whichever path carried it.
		out.Warnings = append(out.Warnings, fmt.Sprintf(
			"the write did not name %q in rejectedSuggestionIds, so Docs answered without saying it rejected the suggestion",
			suggestionID))
	}

	gone, warn := goneFrom(ctx, s, docID, suggestionID)
	if warn != "" {
		out.Warnings = append(out.Warnings, warn)
	}
	out.Verified = gone && contains(out.RejectedSuggestionIDs, suggestionID)
	if sentErr != nil {
		msg := fmt.Sprintf("the reject of %q was accepted by Docs and its answer could not be read", suggestionID)
		if out.Verified {
			msg += ", though the part of it that decoded named the suggestion in rejectedSuggestionIds and the read-back found it gone"
		} else {
			msg += ", so the withdrawal cannot be confirmed and the note still records the proposal; check the document, and take the entry out by hand once the suggestion is gone"
		}
		out.Warnings = append(out.Warnings, msg+": "+sentErr.Error())
	}
	return out, nil
}

// sentAnyway says whether the reject reached Docs in spite of the error. The
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
		return false, fmt.Sprintf("the reject was sent and the document could not be read back, so %q could not be confirmed as gone: %v", suggestionID, err)
	}
	if pending(d, suggestionID) {
		return false, fmt.Sprintf("the read-back still carries a run under %q, so the suggestion is still pending in the document", suggestionID)
	}
	return true, ""
}

// pending says whether any run in any tab still belongs to the suggestion, on
// either side. Either side, because a proposal is a replace: a run still marked
// for deletion under the id is half a proposal that is still pending, and it is
// the half the old delete-based withdraw used to leave behind. Asking about
// both sides is what lets this package take those back too.
//
// A reject names no range, so the tab does not matter and a document with more
// than one is read like any other.
func pending(d *docs.Document, suggestionID string) bool {
	for _, t := range d.Tabs {
		for _, r := range runs(t.Body) {
			if carries(r, suggestionID) {
				return true
			}
		}
	}
	return false
}

// Batch is the whole write: one rejectSuggestion naming the id, in SUGGEST
// mode.
//
// SUGGEST is what the guard requires of any batchUpdate on a handed-in
// document, and Docs honours a rejectSuggestion inside it, measured 2026-09-07.
// The request carries the id and nothing beside it, because that is the exact
// shape the guard's grant opens: a second field is one nobody here has read
// about, and the guard refuses it.
func Batch(suggestionID string) []byte {
	body := map[string]any{
		"requests": []any{
			map[string]any{"rejectSuggestion": map[string]any{"suggestionId": suggestionID}},
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
// The shape is measured rather than documented, on 2026-09-07 on a throwaway
// document: a rejectSuggestion answers with the id under
// suggestionResponses[].rejectedSuggestionIds, beside an empty per-request
// reply and commentUpdateState ALL_SAVED. testdata/rejected.json is that
// answer with the ids replaced. Nothing else is read for the id, because
// nothing else has been seen to carry it.
type batchAnswer struct {
	DocumentID          string `json:"documentId"`
	SuggestionResponses []struct {
		RejectedSuggestionIDs []string `json:"rejectedSuggestionIds"`
	} `json:"suggestionResponses"`
	WriteControl struct {
		RequiredRevisionID string `json:"requiredRevisionId"`
	} `json:"writeControl"`
}

// rejectedIDs is every suggestion the answer says it rejected, in the order
// they were read and without repeats.
func (a batchAnswer) rejectedIDs() []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range a.SuggestionResponses {
		for _, id := range r.RejectedSuggestionIDs {
			if id != "" && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	return out
}

// carries says whether the run belongs to the suggestion at all, on either
// side.
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
		if b.Table == nil {
			continue
		}
		for _, row := range b.Table.Rows {
			for _, c := range row {
				out = append(out, runs(c.Blocks)...)
			}
		}
	}
	return out
}
