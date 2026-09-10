package restyle

// The apply loop: the batches a restyle sends, and the revision check between
// them. page.go and style.go build the requests as pure functions of the house
// style and the document; this file is the only room in the package that sends
// anything, and it decides nothing about what to send.
//
// Three rules, and each one is the reason something below is shaped the way it
// is.
//
// Every batch carries writeControl.requiredRevisionId, and an empty one is
// refused at the call site. The survey rechecks the document before the run
// opens the grant, and a recheck followed by a write leaves a window that is
// the whole run across a dozen batches. requiredRevisionId closes it inside
// Docs: a document that moved refuses the batch itself. It is in the public
// discovery document, unlike writeMode, so it is a rule the server holds rather
// than a statement of intent. A read that carried no revision id would ship an
// empty string and Docs would take the batch, so the only protection this
// milestone has would be gone with nothing said.
//
// The loop reads between batches for the revision id, and for nothing else. An
// earlier draft of this milestone justified the loop with shifting indexes: a
// batch that inserts or deletes moves the positions later batches were computed
// from, so every batch had to be recomputed from a fresh read. That reason went
// with the bullets. None of the four request kinds the in-place level carries
// can change a character, so no index in a request built before the first batch
// can have moved by the last. What the loop still needs from Docs is the next
// revision id, which the answer usually names, and the read below is what
// stands in when it does not.
//
// A failed run has no rollback, and what it leaves behind is written down
// rather than discovered. A run that stops at batch twelve leaves a half-styled
// document, and the recovery is the document's own version history, by hand. No
// text was touched, so nothing the author wrote is lost, but their own run
// formatting inside the paragraphs that were restyled is. The warnings say that
// sentence on every path that stops early, because the caller prints them and
// the skill reads them to Nail. What the count alone cannot say is the batch
// Docs accepted whose answer could not be read: it may be in the document, so it
// is reported as MaybeApplied rather than as a batch that landed or one that
// never happened.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gdoc/internal/docs"
	"gdoc/internal/propose"
)

// Session is what the apply loop needs of a session: the write, and the read it
// falls back to when an answer carries no revision id. Naming the interface
// here rather than the struct keeps net/http out of this room, which is what
// the boundary test asks of every package but the four that build requests.
type Session interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	PostJSON(ctx context.Context, rawURL string, body any, into any) error
}

// maxBatchBytes is how large a batch body may be. The guard reads the first
// maxPeek bytes of a request, a megabyte, and judgeRequests refuses a
// batchUpdate it cannot read whole rather than carrying it unread. So a batch
// is sized well under that: half a megabyte leaves room for an envelope, for a
// revision id of any length, and for the difference between what is measured
// here and what the encoder finally writes.
//
// It is a variable rather than a constant only so a test can narrow it, the way
// cmd/gdoc's wait interval is. Nothing in production writes to it.
var maxBatchBytes = 1 << 19

// batchEnvelope is the bytes a batch spends on everything that is not a
// request: the two keys, the brackets, and the revision id inside writeControl.
// It is a fixed, generous number rather than a measurement of one body, because
// the sizing has to be an over-estimate: a batch measured exactly and then
// encoded a byte longer is a batch the guard reads half of.
const batchEnvelope = 512

// Applied is what the loop sent and what came back. Every field is a count, an
// id or a flag, and none of them says whether the document now looks right:
// that is the read-back's, and past it, Nail's.
type Applied struct {
	// Batches is how many batches Docs accepted, and Requests how many styling
	// requests were inside them. A run that stopped early reports what held.
	Batches  int `json:"batches"`
	Requests int `json:"requests"`
	// RevisionID is the revision the run last knew the document by: the one the
	// last accepted answer named, or, when no answer named one, the revision
	// the last batch was sent against.
	RevisionID string `json:"revision_id"`
	// Stale says Docs refused a batch because the document had moved under the
	// run. It is the one refusal that means somebody else was editing.
	Stale bool `json:"stale"`
	// MaybeApplied says one more batch than Batches counts may be in the
	// document: Docs accepted it and its answer could not be read, so the run
	// cannot say whether it landed. It is a separate field rather than a
	// batch counted in Batches, because Batches is what Docs confirmed and a
	// count that folded the two together could not say which it was. The
	// caller reads it beside Batches to decide whether there is anything to
	// read back, since a batch that may be in the document is a document that
	// may have been directly edited.
	// It is printed on every run, beside Stale, because a flag that vanishes
	// when it is false cannot be read the same way twice.
	MaybeApplied bool `json:"maybe_applied"`
	// MaybeRequests is how many styling requests were inside that batch. The
	// flag says a batch may be in the document and this says which requests
	// they were, because the batches are sent in order, so they are the ones
	// standing right behind Requests in the plan. The read-back needs that:
	// asked about them it says whether the styling is really there, which on
	// this one path is the only way to find out.
	MaybeRequests int `json:"maybe_requests"`
	// Warnings carry what a caller has to know about a run that stopped early,
	// and never a verdict.
	Warnings []string `json:"warnings,omitempty"`
}

// BatchURL is the Docs write path, spelled once for the whole binary. It is
// propose's, because every write in this project goes through the same endpoint
// and two copies of one URL is two places for it to drift.
func BatchURL(docID string) string { return propose.BatchURL(docID) }

// Apply sends the styling requests, in order, in batches, each carrying the
// revision the one before it ended on.
//
// It stops at the first batch that does not land, whatever the reason, and
// never sends the same batch twice. A refusal on a stale revision is the
// document having moved under the run, and retrying it against a fresh revision
// would be gdoc styling a document somebody is editing, which is the exact case
// requiredRevisionId exists to refuse. An ordinary refusal is reported as
// itself. A write Docs accepted whose answer could not be read is neither: the
// batch may be in the document, so it is never sent again, and the run stops
// because the revision the next batch needs was in the answer it could not
// read.
func Apply(ctx context.Context, s Session, docID string, requests []map[string]any, revisionID string) (Applied, error) {
	out := Applied{RevisionID: revisionID}
	if docID == "" {
		return out, fmt.Errorf("no document was named to restyle")
	}
	if revisionID == "" {
		return out, fmt.Errorf(
			"the document was read without a revision id, and a restyle sends nothing without one: " +
				"writeControl.requiredRevisionId is what refuses a batch on a document that moved, and an empty one is a batch Docs takes anyway")
	}
	batches, err := Batches(requests)
	if err != nil {
		return out, err
	}
	if len(batches) == 0 {
		// A plan with nothing in it is a document whose paragraphs the house
		// style has no look for. A batchUpdate naming no requests would be a
		// request made for no reason.
		return out, nil
	}

	for i, b := range batches {
		body, err := batchBody(b, out.RevisionID)
		if err != nil {
			out.Warnings = append(out.Warnings, out.leftBehind(len(batches), false))
			return out, fmt.Errorf("batch %d of %d could not be built: %w", i+1, len(batches), err)
		}
		var answer batchAnswer
		if err := s.PostJSON(ctx, BatchURL(docID), json.RawMessage(body), &answer); err != nil {
			// stopped writes the flags and the warnings this report is made of,
			// so it runs before out is copied into the return value. A return
			// statement's operands and its calls are not ordered against each
			// other, so `return out, out.stopped(...)` is the compiler's choice
			// of whether MaybeApplied reaches the caller, and the read-back gate
			// is hung off exactly that flag.
			stop := out.stopped(i, len(batches), len(b), err)
			return out, stop
		}
		out.Batches++
		out.Requests += len(b)

		next := answer.WriteControl.RequiredRevisionID
		if next == "" {
			if i == len(batches)-1 {
				// Nothing follows, so there is nothing to read a revision for.
				// What the run cannot say is where the document ended up, and
				// it says that rather than reporting a revision the last batch
				// has already moved past.
				out.Warnings = append(out.Warnings,
					"the last batch was accepted and its answer named no revision id, so the revision reported here is the one that batch was sent against, not the one the document now carries")
				break
			}
			read, err := revisionOf(ctx, s, docID)
			if err != nil {
				out.Warnings = append(out.Warnings, out.leftBehind(len(batches), false))
				return out, fmt.Errorf(
					"batch %d of %d was accepted and its answer named no revision id, and the document could not be read for one, so the run stopped rather than sending the next batch without: %w",
					i+1, len(batches), err)
			}
			next = read
		}
		out.RevisionID = next
	}
	return out, nil
}

// stopped is the one place a failed batch becomes a report: the flag, the
// warnings and the error a caller prints. n is how many requests were in the
// batch that did not land, which only the accepted-and-unreadable case records.
func (out *Applied) stopped(i, total, n int, err error) error {
	switch {
	// sentAnyway is asked first, and the order is the rule rather than a
	// preference. isStale reads the message for a word, and the message of a
	// lost answer is encoding/json's, built out of the struct it was decoding
	// into: a writeControl of the wrong shape fails naming
	// batchAnswer.writeControl and requiredRevisionId, which carries that word.
	// Read as a moved document, a batch Docs accepted would leave MaybeApplied
	// false, say the document is as it was, and take the read-back with it, on
	// the one path where a direct edit may be in the document. The two cannot
	// collide the other way: Docs refuses a moved revision with a 4xx, and gapi
	// marks only the failures raised after a 2xx.
	case sentAnyway(err):
		// The flag goes on before the sentence below is built, because that
		// sentence says what is in the document and this is the one path where
		// a batch nothing counted may be in it.
		out.MaybeApplied = true
		out.MaybeRequests = n
		out.Warnings = append(out.Warnings,
			fmt.Sprintf("batch %d of %d was accepted by Docs and its answer could not be read, so that batch may be in the document and it is never sent again", i+1, total),
			out.leftBehind(total, false))
		return fmt.Errorf("batch %d of %d was accepted by Docs and its answer could not be read: %w", i+1, total, err)
	case isStale(err):
		out.Stale = true
		out.Warnings = append(out.Warnings, out.leftBehind(total, false))
		return fmt.Errorf(
			"batch %d of %d was refused because the document moved under this run, so somebody edited it after the survey was taken, and the run stopped rather than styling a document being edited: %w",
			i+1, total, err)
	default:
		out.Warnings = append(out.Warnings, out.leftBehind(total, true))
		return fmt.Errorf("batch %d of %d did not land: %w", i+1, total, err)
	}
}

// leftBehind is what a run that stopped early leaves in the document, in one
// sentence, whichever way it stopped. There is no rollback: a restyle is many
// batches and undoing the ones that landed would be a second run of writes on a
// document already in a state nobody chose.
//
// It reads MaybeApplied rather than the count alone, because "the document is
// as it was" is false on the one path where Docs accepted a batch and the run
// could not read the answer. Two sentences contradicting each other in one
// warnings list is worse than either of them, and the skill reads that list to
// Nail.
//
// maybeReached is the other half of that, and it is the wider one. A batch that
// failed on a request the transport made is one of three things gdoc cannot
// tell apart: a guard refusal, where nothing left the machine; a 4xx, where Docs
// rejected the batch whole; and a 5xx or a dropped connection, where the request
// was written and may have been applied. internal/gapi marks only the failures
// raised after a 2xx, and its own doc comment names the third case, so on that
// path the definite sentence is a claim this package cannot make. The definite
// one is kept for the paths where nothing was sent: a batch that could not be
// built, and one Docs refused on a revision that had moved, which it refuses
// whole.
//
// The read-back is not opened up to match. That gate is Batches or MaybeApplied,
// so a run that stopped this way still makes no reads, and the reason is the
// same uncertainty read the other way: three reads after every guard refusal
// would be paid on the common case to answer the rare one. The sentence says to
// look at the document instead.
func (out *Applied) leftBehind(total int, maybeReached bool) string {
	switch {
	case out.Batches == 0 && !out.MaybeApplied && maybeReached:
		return fmt.Sprintf(
			"no batch of the %d was confirmed applied and the batch that failed may have reached Docs, so the document is either as it was or part styled. %s",
			total, noRollback)
	case out.Batches == 0 && !out.MaybeApplied:
		return "no batch was applied, so the document is as it was"
	case out.Batches == 0:
		return fmt.Sprintf(
			"no batch of the %d was confirmed applied and one may be in the document, so the document is either as it was or part styled. %s",
			total, noRollback)
	case out.MaybeApplied:
		return fmt.Sprintf(
			"%d of %d batches were applied, one more may be in the document, and the run stopped, so the document is part styled. %s",
			out.Batches, total, noRollback)
	default:
		return fmt.Sprintf(
			"%d of %d batches were applied and the run stopped, so the document is half styled. %s",
			out.Batches, total, noRollback)
	}
}

// noRollback is the recovery, and it is one sentence in one place because every
// path that stops early says it and two copies would be two sentences that
// drift.
const noRollback = "There is no rollback, and the recovery is the document's own version history, by hand. " +
	"No text was touched, so nothing the author wrote is lost, but their own run formatting inside the paragraphs that were restyled is"

// Batches splits the requests into bodies the guard can read whole, keeping
// them in document order.
//
// The measure is bytes rather than a count of requests, because that is what
// the guard's peek is measured in: an updateParagraphStyle over a short range
// and one carrying a long field mask are one request each and nothing like the
// same size. A single request too large to send at all is refused naming its
// kind, because splitting one is not something this loop can do and sending it
// would be sending a body the guard reads half of.
func Batches(requests []map[string]any) ([][]map[string]any, error) {
	var out [][]map[string]any
	var cur []map[string]any
	used := 0
	for i, r := range requests {
		raw, err := json.Marshal(r)
		if err != nil {
			return nil, fmt.Errorf("request %d could not be built: %w", i+1, err)
		}
		n := len(raw) + 1 // the comma that separates it from the one before
		if n+batchEnvelope > maxBatchBytes {
			return nil, fmt.Errorf(
				"one %s request is %d bytes, over the %d a batch may carry, and a request is never split: the guard refuses a batchUpdate it cannot read whole",
				kindOf(r), len(raw), maxBatchBytes-batchEnvelope)
		}
		if len(cur) > 0 && used+n+batchEnvelope > maxBatchBytes {
			out = append(out, cur)
			cur, used = nil, 0
		}
		cur = append(cur, r)
		used += n
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out, nil
}

// batchBody is one batch as it goes out: the requests, and the revision the
// document must still be on for Docs to take them.
//
// It returns bytes rather than a map, so what the guard judged is what the
// session sends and no caller in between can change it. That is withdraw.Batch's
// rule, and it is the same rule.
func batchBody(requests []map[string]any, revisionID string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"requests":     requests,
		"writeControl": map[string]any{"requiredRevisionId": revisionID},
	})
}

// kindOf names the one request kind a request carries, for a refusal. A request
// naming none or naming two is the guard's to refuse, and this only has to
// print something a person can search for.
func kindOf(r map[string]any) string {
	var names []string
	for k := range r {
		names = append(names, k)
	}
	if len(names) == 1 {
		return names[0]
	}
	return fmt.Sprintf("%v", names)
}

// batchAnswer is the part of BatchUpdateDocumentResponse this loop reads: the
// revision the next batch has to name. Everything else in the answer is about
// requests this milestone does not send.
type batchAnswer struct {
	WriteControl struct {
		RequiredRevisionID string `json:"requiredRevisionId"`
	} `json:"writeControl"`
}

// revisionOf reads the document for its revision id alone.
//
// It is the narrowed read rather than the whole document, because the revision
// is the only thing the loop wants: the requests were built before the first
// batch and none of the four kinds a restyle sends can move an index, so
// re-reading the prose would be paying for an answer nothing reads. That read's
// mask names documentId, revisionId and the named ranges, and it was measured
// at 982 bytes against 12,907 for the document itself.
//
// It breaks the chain, and that is the cost of this fallback rather than a
// property of it. Every other batch is sent against the revision the batch
// before it produced, so an edit somebody else made in between refuses the next
// batch at Docs. A revision read here is the document as it is now, foreign edit
// included, so that edit is adopted as this run's own and the batches behind it
// are accepted against it. Two things bound the cost. The answer was measured
// carrying the field (docs/v2/DECISIONS.md, 2026-09-09), so this is the rare
// path, and none of the four kinds the level carries can change a character, so
// the loss is a paragraph's own run formatting rather than a word of anybody's
// text. Stopping the run instead, which is what a refused batch gets, trades a
// rare wrong landing for a half-styled document on every run whose answer went
// quiet, and choosing between the two is Nail's.
// docs/backlog/restyle-revision-fallback-breaks-the-chain.md holds it.
func revisionOf(ctx context.Context, s Session, docID string) (string, error) {
	var raw json.RawMessage
	if err := s.GetJSON(ctx, docs.NamedRangesURL(docID), &raw); err != nil {
		return "", err
	}
	var r struct {
		RevisionID string `json:"revisionId"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("the read for the document's revision id did not decode: %w", err)
	}
	if r.RevisionID == "" {
		return "", fmt.Errorf("the read carried no revision id, and a batch is never sent without one")
	}
	return r.RevisionID, nil
}

// isStale says whether Docs refused the batch because the document had moved.
//
// The session reports a failed request as its status and Google's own message,
// so the only thing here to read is that message, and the match is the word
// Google uses for the field it refused. A refusal that does not carry the word
// is reported as an ordinary refusal rather than guessed at: calling a
// permission error a document that moved would send somebody to look at the
// wrong thing, and the run stops either way.
//
// A word is all there is to read, so this is asked after sentAnyway rather than
// before it: a lost answer carries encoding/json's own message, which names the
// field this loop decodes into and so carries the word. stopped holds that
// order, and swapping the two arms back is the defect rather than a tidy-up.
func isStale(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "revision")
}

// sentAnyway says whether the batch reached Docs in spite of the error. The
// session marks the failures raised after the server accepted a request, and
// this room asks by behaviour rather than by importing that package, exactly as
// propose, reply and withdraw ask it: naming a Session interface here is what
// keeps net/http out, and an imported sentinel would bring it back through the
// side door.
func sentAnyway(err error) bool {
	var sent interface{ Sent() bool }
	return errors.As(err, &sent) && sent.Sent()
}
