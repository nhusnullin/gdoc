// Package propose writes one change into a Google Doc as a native suggestion,
// with a comment beside it saying why, and then proves it by reading the
// document back through routes the write did not go out on.
//
// Every write here is writeControl.writeMode SUGGEST. The guard refuses a
// batchUpdate on a handed-in document without it, and that refusal is about
// gdoc's own words: what Google did with them is a different question, and one
// morning the answer was a silent direct edit. So the run does not stop at a
// 200. It reads the document with suggestions inline, reads it again with them
// hidden, and exports the docx to see whether the comment is really anchored.
// All three holding is Verified; fewer is the write reported with the route
// that did not hold named.
//
// Nothing here decides whether a change is worth proposing, or what to say in
// the comment. The words arrive written and the placement arrives quoted. This
// package finds the words, writes what it was told, and reports what it saw.
package propose

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf16"

	"gdoc/internal/docs"
	"gdoc/internal/frontmatter"
)

// Prefix is what every comment gdoc writes opens with. It is the same mark
// reply puts on a reply, and it is the only record of authorship there is: the
// Docs API cannot set an author, so everything gdoc writes is signed by
// whoever is logged in, and the robot is what tells the two apart.
const Prefix = "🤖 "

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

// Checks is which of the three read-backs held. They are three routes to one
// question, and each answers a part of it the other two cannot: the inline view
// says the text is there as a suggestion, the preview says it is not there as
// an edit, and the export says the comment is attached to it.
type Checks struct {
	SuggestionsInline         bool `json:"suggestions_inline"`
	PreviewWithoutSuggestions bool `json:"preview_without_suggestions"`
	DocxAnchored              bool `json:"docx_anchored"`
}

// all says whether every route held. Verified means this and nothing else.
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
	if p.Replacement == "" && p.Quoted == "" {
		return out, fmt.Errorf("the proposal says nothing to change")
	}
	if p.Why == "" {
		return out, fmt.Errorf("the proposal carries no reason, and a suggestion nobody explained is one Nail has to guess at")
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
	if err := s.PostJSON(ctx, BatchURL(docID), json.RawMessage(Batch(r, p)), &answer); err != nil {
		return out, fmt.Errorf("the proposal could not be written: %w", err)
	}
	out.CommentID = answer.commentID()
	out.CommentUpdateState = answer.state()
	if out.CommentUpdateState != stateAllSaved {
		// The text can land while the comment is lost, and the status code says
		// nothing about it. This is the field that does.
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
// The comment fields are read from two places because the preview's response
// shape is not in the public reference: the state has been seen at the top of
// the answer, and a per-request reply is where the rest of the API puts what a
// request produced. Reading both is a few lines; reporting a comment id as
// missing because it arrived one level down would send somebody looking for a
// comment that is sitting in the document.
type batchAnswer struct {
	DocumentID         string `json:"documentId"`
	CommentUpdateState string `json:"commentUpdateState"`
	Replies            []struct {
		CommentUpdateState string `json:"commentUpdateState"`
		InsertComment      *struct {
			CommentID          string `json:"commentId"`
			CommentUpdateState string `json:"commentUpdateState"`
		} `json:"insertComment"`
	} `json:"replies"`
	WriteControl struct {
		RequiredRevisionID string `json:"requiredRevisionId"`
	} `json:"writeControl"`
}

func (a batchAnswer) commentID() string {
	for _, r := range a.Replies {
		if r.InsertComment != nil && r.InsertComment.CommentID != "" {
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
// A result with no suggestion id is skipped rather than written with an empty
// one. An entry with no id fails the block's own validation, and that would
// lose the provenance of every other proposal in the same write, which is the
// worse of the two losses.
func Record(note []byte, results []Result, at time.Time) ([]byte, error) {
	b, err := frontmatter.Read(note)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("the note carries no gdoc: block, so there is nowhere to record the proposals")
	}
	next := *b
	next.Proposals = append(append([]frontmatter.Proposal{}, b.Proposals...), recorded(results, at)...)
	return frontmatter.Write(note, &next)
}

// recorded is the results that can be remembered, as front matter entries.
func recorded(results []Result, at time.Time) []frontmatter.Proposal {
	var out []frontmatter.Proposal
	for _, r := range results {
		if len(r.SuggestionIDs) == 0 || r.CommentID == "" {
			continue
		}
		out = append(out, frontmatter.Proposal{
			ID:        r.SuggestionIDs[0],
			CommentID: r.CommentID,
			At:        at,
			Quoted:    r.Quoted,
		})
	}
	return out
}
