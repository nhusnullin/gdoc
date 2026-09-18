// This file is the annotation and the write it becomes: Annotation, the shape
// rule asked of every entry before the first one leaves the machine, Body, the
// one place the robot is put in front of the words, Batch, the single request
// that goes out, and Apply, which reads, places, writes once and reads back.
// verify.go holds the two read-backs and doc.go holds the package comment.

package annotate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gdoc/internal/docs"
	"gdoc/internal/plaintext"
	"gdoc/internal/propose"
)

// Prefix is what every comment gdoc writes opens with. It is the same mark
// reply requires on a reply and propose puts on a reason, and it is the only
// record of authorship there is: the Docs API cannot set an author, so
// everything gdoc writes is signed by whoever is logged in.
const Prefix = plaintext.Prefix

// Robot is the mark without its space, which is what Check asks about. See
// plaintext.Robot for why the check is not the whole prefix.
const Robot = plaintext.Robot

// Annotation is one comment, as the skill hands it over: the words to comment
// on, why, and optionally who should answer for it.
//
// Quoted is text, never an index. An index computed from an earlier read is the
// hazard the whole API has, so the placement is made of words and looked up in
// a document that has just come back.
type Annotation struct {
	Quoted   string `json:"quoted"`
	Why      string `json:"why"`
	Assignee string `json:"assignee,omitempty"`
}

// Check is the shape rule for one annotation, as shape rather than as meaning.
// It answers from the annotation alone, so the caller can ask it of every entry
// in the file before the first one leaves the machine: a third entry refused
// after the first two have landed is a run that half happened.
//
// Nothing here says whether the comment is worth making. The words arrive
// written and the placement arrives quoted, and this package writes what it was
// told.
func (a Annotation) Check() error {
	if a.Quoted == "" {
		return errors.New("the annotation quotes no text, so there is nothing to comment on")
	}
	// The quote is looked up by propose.FindSpan, which walks one paragraph at
	// a time, so a quote carrying a break is in no single paragraph and could
	// only ever come back as "not found". Refusing it here names the real
	// mistake, and it names it before the document is read.
	if strings.ContainsAny(a.Quoted, "\n\r") {
		return fmt.Errorf("the quoted text %q carries a line break; a comment is anchored inside one paragraph, and the words without the break are always quotable instead", a.Quoted)
	}
	// The reason is read with its surrounding space taken off, which is the
	// shape propose.Check and reply.Check both read a body in. A reason of
	// nothing but spaces is a comment that is a bare signature.
	why := strings.TrimSpace(a.Why)
	if why == "" {
		return errors.New("the annotation carries no reason, and a comment that says nothing is one Nail has to guess at")
	}
	// Body writes the comment as Prefix + Why, so a reason that already opens
	// with the robot lands as two of them. reply.Check requires the prefix and
	// this one refuses it, which is the two writers disagreeing about who owns
	// the marker: a caller carrying the reply convention over would sign the
	// comment twice, and nothing downstream would catch it. The read-backs
	// compare against the string that was sent, so both of them hold over a
	// doubled mark in the one string that records who wrote the comment.
	//
	// The test is the robot itself rather than the robot and its space, because
	// every near miss lands the same doubled mark: the robot with nothing
	// behind it, a robot behind a newline, a robot behind a leading space.
	if strings.HasPrefix(why, Robot) {
		return fmt.Errorf("the reason for the annotation already opens with %q, and gdoc adds it; write the reason without it", Prefix)
	}
	// The reason is written into a comment thread, where Docs renders markdown
	// literally. It is the same rule reply.Check and propose.Check hold, and it
	// is asked of the reason alone rather than of Prefix + Why: the heading arm
	// is anchored to a line start, so the prefix in front of it would move the
	// first character off offset zero and let a reason opening with "# "
	// through.
	if m := plaintext.Markdown(a.Why); m != "" {
		return fmt.Errorf("the reason for the annotation carries markdown (%q); a Docs thread renders it literally, so it would arrive as typed", m)
	}
	return nil
}

// Body is what the comment holds: the robot, one space, and the reason as it
// was given. It is the one place the mark is added, so a later reader asking
// what gdoc wrote is asking about one string built in one room.
func Body(why string) string {
	return Prefix + why
}

// Session is what this package needs of a session: the read the span is found
// in, the byte read the export is, and the one write. Naming the interface here
// keeps net/http out of this room, which is what the boundary test asks of
// every package but the four that build requests.
type Session interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	GetBytes(ctx context.Context, rawURL string, limit int64) ([]byte, error)
	PostJSON(ctx context.Context, rawURL string, body any, into any) error
}

// Checks is which of the two read-backs held. They are two routes to one
// question, and each answers a part of it the other cannot: Drive's listing
// says the comment exists with the words that were sent, and the export says
// those words are wrapped around the quote. The listing reports a comment whose
// anchor was destroyed as healthy, which is why it is not asked alone.
type Checks struct {
	DriveListing bool `json:"drive_listing"`
	DocxAnchored bool `json:"docx_anchored"`
}

// all says whether every route held.
//
// It is not the whole of Verified. Apply requires this and an answered
// commentUpdateState of ALL_SAVED, so a batch Docs accepted whose answer could
// not be read comes back with every route holding and Verified false.
func (c Checks) all() bool {
	return c.DriveListing && c.DocxAnchored
}

// Result is what one annotation left behind.
//
// Warnings carry the reasons Verified is false, the way propose.Result and
// reply.Result carry theirs, and for the same reason: everything that goes
// wrong after the batch is a fact about a comment that is already in the
// document, and a caller told the run failed is a caller that writes it again.
type Result struct {
	Quoted             string   `json:"quoted"`
	CommentID          string   `json:"comment_id,omitempty"`
	CommentUpdateState string   `json:"comment_update_state,omitempty"`
	Verified           bool     `json:"verified"`
	Checks             Checks   `json:"checks"`
	Warnings           []string `json:"warnings,omitempty"`
}

// Apply leaves one comment: read, find, write once, read back two ways.
//
// The read comes first and the write is built from it, so no index outlives the
// answer it was computed in. Two annotations are two calls, each with its own
// read. A comment moves no character, so the second read sees what the first
// one saw, but the rule is the same one propose holds and holding it in one
// shape is cheaper than holding it in two.
//
// The span walk and the write path are propose's: FindSpan and BatchURL have
// one owner each, and a second copy of either is a second place a quote can be
// placed differently.
//
// It fails only before the write. After the batch has gone out the comment
// exists, and everything from there is reported rather than raised.
func Apply(ctx context.Context, s Session, docID string, a Annotation) (Result, error) {
	out := Result{Quoted: a.Quoted}
	if err := a.Check(); err != nil {
		return out, err
	}
	d, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		return out, fmt.Errorf("the document could not be read before annotating: %w", err)
	}
	if d.MultiTab() {
		// The write names one range, and a range means nothing without saying
		// which tab it is in. Refusing is the honest answer until a milestone
		// decides how a comment names a tab. propose refuses the same document
		// with the same words.
		return out, fmt.Errorf("the document has %d tabs, and a comment is written into a document with one", len(d.Tabs))
	}
	r, err := propose.FindSpan(d, a.Quoted)
	if err != nil {
		return out, err
	}

	body := Body(a.Why)
	var answer batchAnswer
	if err := s.PostJSON(ctx, propose.BatchURL(docID), json.RawMessage(Batch(r, body, a.Assignee)), &answer); err != nil {
		return out, fmt.Errorf("the comment could not be written: %w", err)
	}
	out.CommentID = answer.commentID()
	out.CommentUpdateState = answer.state()
	if out.CommentUpdateState != stateAllSaved {
		// The status code says nothing about whether the comment was saved.
		// This is the field that does.
		out.Warnings = append(out.Warnings, fmt.Sprintf(
			"the write answered commentUpdateState %q rather than %q, so the comment may not be in the document",
			out.CommentUpdateState, stateAllSaved))
	}

	checks, warns := verify(ctx, s, docID, out.CommentID, a.Quoted, body)
	out.Checks = checks
	out.Warnings = append(out.Warnings, warns...)
	out.Verified = checks.all() && out.CommentUpdateState == stateAllSaved
	return out, nil
}

// verify is the two read-backs. verify.go replaces this body with the Drive
// listing and the export; until then nothing is read back, so nothing is
// claimed and Verified is false.
//
// TODO(verify): the two routes, each answering for itself.
func verify(context.Context, Session, string, string, string, string) (Checks, []string) {
	return Checks{}, nil
}

// stateAllSaved is the one commentUpdateState that means the comment landed.
// The other two Google documents are NO_UPDATES_REQUESTED and
// ALL_FAILED_UNKNOWN_REASON, and neither is success here: this write is nothing
// but a comment, so nothing requested would mean the comment was dropped.
const stateAllSaved = "ALL_SAVED"

// Batch is the whole write: one request, and the write mode beside it.
//
// One request is the design rather than an economy. A batch holding an
// insertComment and nothing else cannot move a character, whatever Google does
// with the write mode on the day, which is why this writer runs no capability
// probe. Adding a second request kind here would take that away.
//
// The insertComment shape is measured rather than documented: content and range
// at the top of the request, with assigneeEmailAddress beside them. Every other
// spelling answers "Cannot find field" (DECISIONS.md, 2026-08-29).
//
// The range is the quoted words themselves. propose anchors its comment on the
// span its insert made, because there the words on the page are about to be
// different ones; here nothing is inserted, so the anchor is the quote.
func Batch(r docs.Range, body, assignee string) []byte {
	comment := map[string]any{
		"range": map[string]any{
			"startIndex": r.Start,
			"endIndex":   r.End,
		},
		"content": body,
	}
	if assignee != "" {
		comment["assigneeEmailAddress"] = assignee
	}
	out := map[string]any{
		"requests":     []any{map[string]any{"insertComment": comment}},
		"writeControl": map[string]any{"writeMode": "SUGGEST"},
	}
	// The body is built from a map with no cycles and no unsupported types, so
	// the marshal cannot fail. Returning bytes rather than a map keeps the
	// caller from being able to change what the guard already judged.
	raw, _ := json.Marshal(out)
	return raw
}

// batchAnswer is what came back from the write, in the shape propose measured
// on 2026-09-07: the id sits under insertComment.commentThread.commentId,
// beside an anchorId, and commentUpdateState sits at the top of the answer.
//
// The flat insertComment.commentId was the guess made before that measurement.
// It stays as a fallback because reading one more field costs nothing, and
// reporting a comment id as missing because it arrived at a level this struct
// did not name is exactly what the first live write test did.
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
