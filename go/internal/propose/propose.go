// This file is the proposal itself: Proposal and its shape rule, the one write
// Apply makes, the batch of three requests it is made of, and Record, which
// writes what landed into the note. span.go finds the words in the document,
// verify.go reads the document back three ways, and doc.go holds the package
// comment.
package propose

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf16"

	"gdoc/internal/docs"
	"gdoc/internal/frontmatter"
	"gdoc/internal/plaintext"
)

// Prefix is what every comment gdoc writes opens with. It is the same mark
// reply puts on a reply, and it is the only record of authorship there is: the
// Docs API cannot set an author, so everything gdoc writes is signed by
// whoever is logged in, and the robot is what tells the two apart.
const Prefix = plaintext.Prefix

// Robot is the mark without its space, which is what Check asks about. See
// plaintext.Robot for why the check is not the prefix.
const Robot = plaintext.Robot

// Proposal is one change, as the skill hands it over: the words to replace, the
// words to put there, why, and optionally who should answer for it.
//
// Quoted is text, never an index. An index computed from an earlier read is the
// hazard the whole API has, so the placement is made of words and looked up in
// a document that has just come back.
type Proposal struct {
	Quoted      string `json:"quoted"`
	Replacement string `json:"replacement"`
	Why         string `json:"why"`
	Assignee    string `json:"assignee,omitempty"`
}

// Check is the shape rule for one proposal, as shape rather than as meaning. It
// answers from the proposal alone, so the caller can ask it of every proposal
// in the file before the first one leaves the machine: a third entry refused
// after the first two have landed is a run that half happened.
//
// A replacement of nothing is refused rather than sent. The batch would carry
// an insertText with no text and a comment anchored on a range of length zero,
// which is a body Docs rejects, and inlineHolds looks for an insertion that a
// plain deletion never makes, so the write could never verify either. A
// milestone that wants a deletion-only proposal gives it its own request shape.
func (p Proposal) Check() error {
	if p.Quoted == "" {
		return errors.New("the proposal quotes no text, so there is nothing to replace")
	}
	if p.Replacement == "" {
		return fmt.Errorf("the proposal replaces %q with nothing; a proposal replaces words with words, and a plain deletion is not a shape this write has", p.Quoted)
	}
	// Words with words is one paragraph's worth of words, on both sides, and
	// each side is refused here for its own reason.
	//
	// The quote is refused because a paragraph's last run carries the paragraph
	// mark itself: the Docs read hands back "...operations team.\n", so a quote
	// ending in a newline matches inside that one paragraph, and the span
	// FindSpan returns ends past the mark. The deleteContentRange built from it
	// then marks the mark for deletion, and accepting the suggestion merges the
	// paragraph with the one behind it while the insertText puts back a
	// replacement that cannot carry a break. Nothing downstream names it:
	// inlineHolds compares the deleted runs against this same Quoted, and
	// Carries finds this same string in the preview, so all three read-backs
	// hold over a proposal that removes a paragraph. A quote with a break in the
	// middle of it is already unreachable, because it spans two paragraphs and
	// the walk indexes one at a time, so this costs a caller nothing: the words
	// without the trailing mark are always writable instead.
	if strings.ContainsAny(p.Quoted, "\n\r") {
		return fmt.Errorf("the quoted text %q carries a line break; a proposal replaces words with words inside one paragraph, and a quote that takes the paragraph mark with it is a change that removes a paragraph", p.Quoted)
	}
	// The replacement is refused because of the preview check. Carries asks each
	// paragraph on its own, and a paragraph ends at its own break, so a want
	// whose newline is anywhere but the very end is in no single paragraph and
	// comes back false whatever the document holds. A trailing one is the
	// exception rather than the rule: a paragraph's last run carries the mark,
	// as the quote rule above says, so a want ending in a newline can be found.
	// Either way the answer is worthless. The rule above gives the first
	// question a Quoted that cannot carry a break; nothing gives the second
	// question that, and a replacement that contains the quote and a newline
	// together would slip past the ambiguity arm and report the preview route
	// as holding on exactly the silent direct edit it exists to name.
	if strings.ContainsAny(p.Replacement, "\n\r") {
		return fmt.Errorf("the replacement %q carries a line break; a proposal replaces words with words inside one paragraph, and a change that adds a paragraph is not a shape this write has", p.Replacement)
	}
	// Both tests read the reason with its surrounding space taken off, which is
	// the same shape reply.Check reads a body in. A reason of nothing but spaces
	// is a comment that is a bare signature, and sameWords normalises whitespace
	// on both sides, so the export would still match it and the run would report
	// verified: true over an explanation that says nothing.
	why := strings.TrimSpace(p.Why)
	if why == "" {
		return errors.New("the proposal carries no reason, and a suggestion nobody explained is one Nail has to guess at")
	}
	// Batch writes the comment as Prefix + Why, so a reason that already opens
	// with the robot lands as two of them. reply.Check requires the prefix and
	// this one refuses it, which is the two writers disagreeing about who owns
	// the marker: a caller carrying the reply convention over would sign the
	// comment twice, and nothing downstream would catch it. docxHolds compares
	// against the string that was sent, so all three read-backs pass and the
	// run reports verified: true over a doubled mark in the one string that
	// records who wrote the comment.
	//
	// The test is the robot itself rather than the robot and its space, because
	// every near miss lands the same doubled mark: "\U0001F916the policy", a robot
	// behind a newline, a robot behind a leading space. reply_test names those
	// spellings one by one, and refusing only the exact prefix here would let
	// each of them through the writer that adds the mark.
	if strings.HasPrefix(why, Robot) {
		return fmt.Errorf("the reason for the proposal already opens with %q, and gdoc adds it; write the reason without it", Prefix)
	}
	// The reason is written into a comment thread, where Docs renders markdown
	// literally. It is the same rule reply.Check holds over a reply, and it is
	// held here because this is the other place gdoc writes into a thread.
	//
	// The rule is asked of the reason alone, never of Prefix + Why. The heading
	// arm is anchored to a line start, so the prefix in front of it moves the
	// first character of the reason off offset zero and a reason opening with
	// "# " passes: the comment then lands with the hash rendered literally,
	// which is the one thing this check exists to refuse. reply.Check asks it
	// behind the prefix for the same reason: there the body arrives with the
	// mark already on it, and Check has required it by then, so trimming it is
	// exact and a heading on the reply's first line is refused too.
	if m := plaintext.Markdown(p.Why); m != "" {
		return fmt.Errorf("the reason for the proposal carries markdown (%q); a Docs thread renders it literally, so it would arrive as typed", m)
	}
	return nil
}

// Checks is which of the three read-backs held. They are three routes to one
// question, and each answers a part of it the other two cannot: the inline view
// says the text is there as a suggestion, the preview says it is not there as
// an edit, and the export says the comment is attached to it.
type Checks struct {
	SuggestionsInline         bool `json:"suggestions_inline"`
	PreviewWithoutSuggestions bool `json:"preview_without_suggestions"`
	DocxAnchored              bool `json:"docx_anchored"`
}

// all says whether every route held.
//
// It is not the whole of Verified. Apply requires this and an answered
// commentUpdateState of ALL_SAVED, so a batch Docs accepted whose answer could
// not be read comes back with every route holding and Verified false. The
// warning there names the answer that was lost, and no route is blamed for it.
func (c Checks) all() bool {
	return c.SuggestionsInline && c.PreviewWithoutSuggestions && c.DocxAnchored
}

// Result is what one proposal left behind.
//
// Warnings carry the reasons Verified is false, the way reply.Result and
// probe.Report carry theirs, and for the same reason: everything that goes
// wrong after the batch is a fact about a change that is already in the
// document, and a caller told the run failed is a caller that writes it again.
type Result struct {
	Quoted             string   `json:"quoted"`
	Replacement        string   `json:"replacement"`
	SuggestionIDs      []string `json:"suggestion_ids,omitempty"`
	CommentID          string   `json:"comment_id,omitempty"`
	CommentUpdateState string   `json:"comment_update_state,omitempty"`
	Verified           bool     `json:"verified"`
	Checks             Checks   `json:"checks"`
	Warnings           []string `json:"warnings,omitempty"`
}

// Session is what this package needs of a session: the two reads the verify
// stands on, the byte read the export is, and the one write. Naming the
// interface here keeps net/http out of this room, which is what the boundary
// test asks of every package but the four that build requests.
type Session interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	GetBytes(ctx context.Context, rawURL string, limit int64) ([]byte, error)
	PostJSON(ctx context.Context, rawURL string, body any, into any) error
}

// Apply places one proposal: read, find, write once, verify three ways.
//
// The read comes first and the write is built from it, so no index outlives the
// answer it was computed in. Two proposals are two calls, each with its own
// read, because the first one moves the ground under the second.
//
// It fails only before the write. After the batch has gone out the change
// exists, and everything from there is reported rather than raised.
func Apply(ctx context.Context, s Session, docID string, p Proposal) (Result, error) {
	out := Result{Quoted: p.Quoted, Replacement: p.Replacement}
	if err := p.Check(); err != nil {
		return out, err
	}
	d, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		return out, fmt.Errorf("the document could not be read before proposing: %w", err)
	}
	if d.MultiTab() {
		// The write names one range, and a range means nothing without saying
		// which tab it is in. Refusing is the honest answer until a milestone
		// decides how a proposal names a tab.
		return out, fmt.Errorf("the document has %d tabs, and a proposal is written into a document with one", len(d.Tabs))
	}
	r, err := FindSpan(d, p.Quoted)
	if err != nil {
		return out, err
	}

	var answer batchAnswer
	answerRead := true
	var sentErr error
	if err := s.PostJSON(ctx, BatchURL(docID), json.RawMessage(Batch(r, p)), &answer); err != nil {
		if !sentAnyway(err) {
			return out, fmt.Errorf("the proposal could not be written: %w", err)
		}
		// Docs took the batch and the answer could not be read. Raising here
		// would report a change that is in the document as one that never left
		// the machine, and the read-backs below are exactly what says which it
		// was. The comment id is usually lost with the answer, so Record cannot
		// remember this one, and the warning below says so when it really is
		// missing rather than whenever this path was taken.
		answerRead = false
		sentErr = err
	}
	// On the path above these are usually empty, and the warning already says
	// why: an invalid body is rejected whole, because json.Unmarshal validates
	// the input before it decodes any of it, and a body over the ceiling never
	// reaches the decoder. The one exception is valid JSON of the wrong shape,
	// where encoding/json saves the type error and keeps going, so whatever the
	// server really did send is kept. It is kept on purpose: a comment id that
	// was decoded is the provenance withdraw needs, and throwing it away is the
	// one loss this package exists to avoid.
	out.CommentID = answer.commentID()
	out.CommentUpdateState = answer.state()
	if sentErr != nil {
		msg := "the proposal was accepted by Docs and its answer could not be read"
		if out.CommentID == "" {
			msg += ", so the comment id is unknown"
		} else {
			msg += ", though the part of it that decoded carried the comment id"
		}
		out.Warnings = append(out.Warnings, msg+": "+sentErr.Error())
	}
	if (answerRead || out.CommentUpdateState != "") && out.CommentUpdateState != stateAllSaved {
		// The text can land while the comment is lost, and the status code says
		// nothing about it. This is the field that does.
		//
		// An answer that could not be read is asked only when it left a state
		// behind anyway. Saying the write "answered" the empty string names
		// something the server did not do, on top of a warning a few lines up
		// that already says the answer was lost; a state that really was
		// decoded is the server's own word and is reported.
		out.Warnings = append(out.Warnings, fmt.Sprintf(
			"the write answered commentUpdateState %q rather than %q, so the change may be in the document without the comment that explains it",
			out.CommentUpdateState, stateAllSaved))
	}

	checks, ids, warns := Verify(ctx, s, docID, r, p, Prefix+p.Why)
	out.Checks = checks
	out.SuggestionIDs = ids
	out.Warnings = append(out.Warnings, warns...)
	out.Verified = checks.all() && out.CommentUpdateState == stateAllSaved
	return out, nil
}

// sentAnyway says whether the write reached Docs in spite of the error. The
// session marks the failures raised after the server accepted a request, and
// this room asks by behaviour rather than by importing that package: naming a
// Session interface here is what keeps net/http out, and an imported sentinel
// would bring it back through the side door.
func sentAnyway(err error) bool {
	var sent interface{ Sent() bool }
	return errors.As(err, &sent) && sent.Sent()
}

// stateAllSaved is the one commentUpdateState that means the comment landed
// with the text. The other two Google documents are NO_UPDATES_REQUESTED and
// ALL_FAILED_UNKNOWN_REASON, and neither is success here: this write always
// carries a comment, so nothing requested would mean the comment was dropped.
const stateAllSaved = "ALL_SAVED"

// Batch is the whole write: one request list, applied in order inside one
// batchUpdate, so the indexes stay consistent across the three of them.
//
// The order is the reason it works. deleteContentRange marks the quoted words
// as suggested-deleted without moving them, insertText puts the replacement at
// the same start, and insertComment anchors on the span the insert just made.
// Splitting them into two batches would mean computing the second batch's
// indexes from a document the first one had already changed.
//
// The insertComment shape is measured rather than documented: content and range
// at the top of the request, with assigneeEmailAddress beside them. Every other
// spelling answers "Cannot find field" (DECISIONS.md, 2026-08-29).
func Batch(r docs.Range, p Proposal) []byte {
	comment := map[string]any{
		"range": map[string]any{
			"startIndex": r.Start,
			"endIndex":   r.Start + utf16Len(p.Replacement),
		},
		"content": Prefix + p.Why,
	}
	if p.Assignee != "" {
		comment["assigneeEmailAddress"] = p.Assignee
	}
	body := map[string]any{
		"requests": []any{
			map[string]any{"deleteContentRange": map[string]any{
				"range": map[string]any{"startIndex": r.Start, "endIndex": r.End},
			}},
			map[string]any{"insertText": map[string]any{
				"location": map[string]any{"index": r.Start},
				"text":     p.Replacement,
			}},
			map[string]any{"insertComment": comment},
		},
		"writeControl": map[string]any{"writeMode": "SUGGEST"},
	}
	// The body is built from a map with no cycles and no unsupported types, so
	// the marshal cannot fail. Returning bytes rather than a map keeps the
	// caller from being able to change what the guard already judged.
	raw, _ := json.Marshal(body)
	return raw
}

// utf16Len is how long a string is in the units the Docs API counts. A rune
// outside the basic plane is two of them, and an emoji is usually one of those.
func utf16Len(s string) int { return len(utf16.Encode([]rune(s))) }

// BatchURL is the one write path the Docs API has.
func BatchURL(docID string) string {
	return "https://docs.googleapis.com/v1/documents/" + docID + ":batchUpdate"
}

// batchAnswer is what came back from the write.
//
// The preview's response shape is not in the public reference, so it was
// measured, on 2026-09-07, on a throwaway document in the test folder. The
// insertComment reply carries a commentThread, and the id is
// replies[i].insertComment.commentThread.commentId, beside an anchorId, the
// headPost that is the comment's first post, a status and plainTextQuote. The
// state sits at the top of the answer as commentUpdateState, and a
// suggestionResponses list beside it names the suggestion ids each request
// created or touched.
//
// The flat insertComment.commentId was the guess made before the measurement.
// It stays as a fallback because reading one more field costs nothing, and
// reporting a comment id as missing because it arrived at a level this struct
// did not name is exactly what the first live write test did: the proposal
// landed, the id was in the answer, and withdraw would have refused it for
// ever. testdata/batch-saved-measured.json is the measured answer.
type batchAnswer struct {
	DocumentID         string `json:"documentId"`
	CommentUpdateState string `json:"commentUpdateState"`
	Replies            []struct {
		CommentUpdateState string `json:"commentUpdateState"`
		InsertComment      *struct {
			CommentID          string `json:"commentId"`
			CommentUpdateState string `json:"commentUpdateState"`
			CommentThread      *struct {
				CommentID string `json:"commentId"`
				AnchorID  string `json:"anchorId"`
			} `json:"commentThread"`
		} `json:"insertComment"`
	} `json:"replies"`
	WriteControl struct {
		RequiredRevisionID string `json:"requiredRevisionId"`
	} `json:"writeControl"`
}

func (a batchAnswer) commentID() string {
	for _, r := range a.Replies {
		if r.InsertComment == nil {
			continue
		}
		if t := r.InsertComment.CommentThread; t != nil && t.CommentID != "" {
			return t.CommentID
		}
		if r.InsertComment.CommentID != "" {
			return r.InsertComment.CommentID
		}
	}
	return ""
}

func (a batchAnswer) state() string {
	if a.CommentUpdateState != "" {
		return a.CommentUpdateState
	}
	for _, r := range a.Replies {
		if r.CommentUpdateState != "" {
			return r.CommentUpdateState
		}
		if r.InsertComment != nil && r.InsertComment.CommentUpdateState != "" {
			return r.InsertComment.CommentUpdateState
		}
	}
	return ""
}

// Record writes what landed into the note's front matter, which is the only
// place gdoc's own proposals are remembered. withdraw reads it back: a
// suggestion gdoc cannot find here is somebody else's, and gdoc does not
// retract those.
//
// A result with no suggestion id or no comment id cannot be written: an entry
// missing either fails the block's own validation, and that would lose the
// provenance of every other proposal in the same write, which is the worse of
// the two losses. Those results come back in the second return value instead,
// because losing them quietly is what leaves gdoc refusing to withdraw a
// suggestion it wrote. The caller says so.
func Record(note []byte, results []Result, at time.Time) ([]byte, []Result, error) {
	b, err := frontmatter.Read(note)
	if err != nil {
		return nil, nil, err
	}
	if b == nil {
		return nil, nil, fmt.Errorf("the note carries no gdoc: block, so there is nowhere to record the proposals")
	}
	entries, missed := recorded(results, at)
	if len(entries) == 0 {
		return note, missed, nil
	}
	next := *b
	next.Proposals = append(append([]frontmatter.Proposal{}, b.Proposals...), entries...)
	out, err := frontmatter.Write(note, &next)
	if err != nil {
		return nil, nil, err
	}
	return out, missed, nil
}

// recorded is the results that can be remembered, as front matter entries, and
// the results that cannot.
func recorded(results []Result, at time.Time) ([]frontmatter.Proposal, []Result) {
	var out []frontmatter.Proposal
	var missed []Result
	for _, r := range results {
		if len(r.SuggestionIDs) == 0 || r.CommentID == "" {
			missed = append(missed, r)
			continue
		}
		out = append(out, frontmatter.Proposal{
			ID:        r.SuggestionIDs[0],
			CommentID: r.CommentID,
			At:        at,
			Quoted:    r.Quoted,
		})
	}
	return out, missed
}
