// Package reply posts one reply into one comment thread, and reads the thread
// back to see whether it is there.
//
// Nothing here decides whether a thread needs a reply, or what to say in it.
// The body arrives written, on the line or in a file the skill wrote, is
// checked for shape alone, and is sent with the mark in effect in front of it
// and nothing else added.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # gdoc adds the mark, and a mark the caller typed comes off first
//
// Check is asked of the words alone. internal/propose and internal/annotate
// hold the same rule over a reason, so all three writers add the mark and no
// caller has to know which one it is talking to. internal/plaintext holds the
// mark itself, the list of known marks and what a mark may be.
// TestReplyAcceptsABodyWithNoMark and TestCheckRefusesEachBadShapeByName are
// the pins, with TestReplyAddsTheMarkInEffect over the string that reaches
// Drive.
//
// A body that opens with a mark gdoc recognises, one of the known three or the
// mark in effect, has it taken off with plaintext.CutMark and the mark in
// effect written instead, once, and the warning names what it found. A reply
// body is prose a person may have signed by hand, and every release before
// this one asked the skill to type the mark itself, so refusing it would refuse
// the call every older skill makes. That tolerance holds for v2.11.0 only and
// the warning says so, because a release that kept it quietly would make the
// typed mark part of the call again.
// TestABodyOpeningWithAKnownMarkIsWrittenOnceWithAWarning and
// TestAnEmptyMarkStillTakesATypedMarkOff are the pins.
//
// An empty body is said to be empty rather than said to be missing the mark,
// because that is the sentence somebody can act on, and a body that is nothing
// but a mark is refused too: a bare signature is a reply that says nothing and
// every read-back would hold over it.
//
// # The markdown rule is asked behind the mark, never in front of it
//
// A Docs thread renders markdown literally, so asterisks and backticks arrive
// as typed and somebody strips them by hand. The heading arm of that rule is
// anchored to a line start, so a mark sitting in front of a "# " moves the hash
// off offset zero and the arm cannot fire: asked of the whole body, a heading
// on the first line passes while the same words on the second line are refused,
// which is one rule firing or not depending on where the author put them. So
// the rule is asked of the words, which is the string gdoc puts the mark in
// front of, and a mark the caller typed is already off by then. A heading on
// the first line is refused either way. Proposal.Check asks the same rule of
// the reason alone, for the same reason.
// TestPostRefusesMarkdownBeforeAnythingIsSent is the pin, with
// TestReplyRefusesMarkdownBeforeAnyRequest in cmd/gdoc.
//
// # The check runs before the write, and nothing after the write can fail
//
// A body Docs would mangle never reaches the document. After the write
// everything that goes wrong is a fact about a reply that already exists, so it
// is reported rather than raised: a caller told the run failed is a caller that
// posts the reply a second time. Warnings carry the reason Verified is false.
// TestPostSendsTheWordsUnderTheMarkToTheRepliesURL,
// TestPostCarriesAGuardRefusalToTheCaller,
// TestAReadBackThatFailedIsAWarningRatherThanAnError and
// TestTheGuardRefusesAReplyOnADocumentNobodyNamed are the pins.
//
// # The thread is read back on an id that survived
//
// Verified is the listing carrying the reply under the id Drive gave, with the
// words that were sent. The text is compared as well as the id, because an id
// Drive echoed back is Drive agreeing with itself and the words are what Nail
// will read. TestPostReportsTheReplyAndVerifiesItFromTheThread,
// TestPostReportsTheReplyIDWhenTheThreadDoesNotShowIt and
// TestPostDoesNotVerifyAReplyWhoseTextCameBackDifferent are the pins.
//
// A write whose answer could not be read is not a write that never happened, so
// whatever did decode is kept and the read-back is made on it. encoding/json
// saves the first type error and keeps decoding, so a 200 whose createdTime came
// back as a number still names the reply, and throwing that id away would report
// a reply Drive named in full as one that could not be looked for. With no id at
// all the run says to check the thread before posting again, which is the most
// it can honestly say. TestPostKeepsWhatAFailedAnswerStillCarried,
// TestPostSaysSoWhenTheAnswerCouldNotBeRead and
// TestPostSaysSoWhenDriveAnswersWithNoReplyID are the pins. The rule itself is
// internal/gapi's, asked here by behaviour rather than by importing that
// package: naming a Session interface is what keeps net/http out of this room,
// and an imported sentinel would bring it back through the side door.
package reply

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"gdoc/internal/comments"
	"gdoc/internal/plaintext"
)

// Result is what one reply left behind. The id and the time come from Drive's
// answer to the write; Verified comes from reading the thread back, which is a
// different route to the same fact and the only one that is not Drive agreeing
// with itself.
//
// Warnings carry the reason Verified is false. They are warnings rather than an
// error on purpose: the write has already happened by then, and a caller told
// the run failed is a caller that posts the reply a second time.
type Result struct {
	ReplyID  string   `json:"reply_id,omitempty"`
	Created  string   `json:"created,omitempty"`
	Verified bool     `json:"verified"`
	Warnings []string `json:"warnings,omitempty"`
}

// Session is what this package needs of a session: the write, and the read the
// listing behind Verified is made of. Naming the interface here keeps net/http
// out of this room, which is what the boundary test asks of every package but
// the four that build requests.
type Session interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	PostJSON(ctx context.Context, rawURL string, body any, into any) error
}

// Check is the whole rule about what a reply may say, as shape rather than as
// meaning. It refuses an empty body, a body that is nothing but a mark, and a
// body carrying markdown or one of gdoc's own markers, naming what it found. It
// requires no mark: gdoc adds the mark in effect itself.
//
// Every question is asked of the words, which is the string Post puts the mark
// in front of, so a mark the caller typed changes no answer. The order matters:
// an empty body is said to be empty rather than said to carry markdown,
// because that is the sentence somebody can act on.
func Check(body, mark string) error {
	words, found := plaintext.CutMark(body, mark)
	if strings.TrimSpace(words) == "" {
		if found {
			return errors.New("the reply body is empty behind the mark")
		}
		return errors.New("the reply body is empty")
	}
	// The rule is asked of the words, never of the body with a mark still on
	// it. The heading arm is anchored to a line start, so a mark in front of it
	// moves the first character off offset zero and a reply opening with "# "
	// would pass, while the same words on the second line are refused: the rule
	// would fire or not depending on where the author put them. Proposal.Check
	// asks the same rule of the reason alone, for the same reason.
	if m := plaintext.Markdown(words); m != "" {
		return fmt.Errorf("the reply body carries markdown (%q); a Docs thread renders it literally, so it would arrive as typed", m)
	}
	// The same rule about the same thing. A review session quoting an exported
	// file back into the thread it came from would otherwise put the marker
	// into the document as words.
	if m := plaintext.Marker(words); m != "" {
		return fmt.Errorf("the reply body carries %s, one of gdoc's own markers; a marker travels out of a document and never back in, so it would arrive as typed", m)
	}
	return nil
}

// signed is what Post sends: the mark in effect, one space, then the words. A
// body that already opens with a mark gdoc recognises has it taken off first,
// and the warning that comes back names the mark the body carried rather than
// the one this run writes, because that is the character somebody has to go and
// find in their skill.
//
// One mark comes off and no more, which is plaintext.CutMark's rule: a body
// carrying two is a body gdoc signs once rather than one that quietly loses
// words nobody looked at.
func signed(body, mark string) (string, string) {
	words, found := plaintext.CutMark(body, mark)
	sent := plaintext.Prefixed(mark, words)
	if !found {
		return sent, ""
	}
	opened, _ := plaintext.MarkOpening(body, mark)
	wrote := fmt.Sprintf("it was written once, as %s", mark)
	if mark == "" {
		wrote = "the mark in effect is empty, so the words were written alone"
	}
	return sent, fmt.Sprintf(
		"the reply opened with %s, and gdoc adds the mark itself; %s. gdoc takes a mark off the front for v2.11.0 only, so stop typing one",
		opened, wrote)
}

// Post writes one reply and then reads the thread it went into.
//
// The check runs before anything is sent, so a body Docs would mangle never
// reaches the document. After the write the run cannot fail: everything that
// goes wrong from there is a fact about a reply that already exists, and the
// caller needs it reported rather than raised.
func Post(ctx context.Context, s Session, docID, commentID, body, mark string) (Result, error) {
	if err := Check(body, mark); err != nil {
		return Result{}, err
	}
	// The mark goes on here and nowhere else, so the read-back below compares
	// the string that went out rather than a constant.
	body, markWarning := signed(body, mark)
	var answer struct {
		ID      string `json:"id"`
		Created string `json:"createdTime"`
		Content string `json:"content"`
	}
	// accepted is set when Drive took the write and its answer could not be read
	// whole. Raising there would send the skill back to post the reply a second
	// time, and the first one is in the thread.
	var accepted error
	if err := s.PostJSON(ctx, CreateURL(docID, commentID), map[string]any{"content": body}, &answer); err != nil {
		if !sentAnyway(err) {
			return Result{}, fmt.Errorf("the reply could not be posted to thread %q: %w", commentID, err)
		}
		accepted = err
	}
	// What did decode is kept, the way propose keeps the comment id and withdraw
	// keeps the deleted ids. encoding/json saves the first type error and keeps
	// decoding, so a 200 whose createdTime came back as a number still names the
	// reply, and throwing that id away would report a reply Drive named in full
	// as one that could not be looked for.
	out := Result{ReplyID: answer.ID, Created: answer.Created}
	if markWarning != "" {
		out.Warnings = append(out.Warnings, markWarning)
	}
	if answer.ID == "" {
		// Drive took the write and said nothing useful about it. Calling that a
		// failure would send the skill back to post it again, and the first one
		// may well be sitting in the thread.
		if accepted != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf(
				"the reply to thread %q was accepted and its answer could not be read, so it could not be looked for in the thread; check the thread before posting again: %v",
				commentID, accepted))
			return out, nil
		}
		out.Warnings = append(out.Warnings, fmt.Sprintf("Drive answered the reply to thread %q with no reply id, so it could not be looked for in the thread; check the thread before posting again", commentID))
		return out, nil
	}
	if accepted != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf(
			"the reply to thread %q was accepted and its answer could not be read whole, though the part of it that decoded carried the reply id: %v",
			commentID, accepted))
	}
	verified, warn := inThread(ctx, s, docID, commentID, answer.ID, body)
	out.Verified = verified
	if warn != "" {
		out.Warnings = append(out.Warnings, warn)
	}
	return out, nil
}

// sentAnyway says whether the write reached Drive in spite of the error. The
// session marks the failures raised after the server accepted a request, and
// this room asks by behaviour rather than by importing that package: naming a
// Session interface here is what keeps net/http out, and an imported sentinel
// would bring it back through the side door.
func sentAnyway(err error) bool {
	var sent interface{ Sent() bool }
	return errors.As(err, &sent) && sent.Sent()
}

// inThread is the read-back: the reply is verified when the listing carries it
// under the id Drive gave, with the words that were sent. The text is compared
// as well as the id, because an id Drive echoed back is Drive agreeing with
// itself and the words are what Nail will read.
func inThread(ctx context.Context, s Session, docID, commentID, replyID, body string) (bool, string) {
	raw, err := comments.Fetch(ctx, s, docID, nil)
	if err != nil {
		return false, fmt.Sprintf("the reply %q was posted and the thread could not be read back to confirm it: %v", replyID, err)
	}
	for _, c := range raw {
		if c.ID != commentID {
			continue
		}
		for _, r := range c.Replies {
			if r.ID == replyID && r.Content == body {
				return true, ""
			}
		}
		return false, fmt.Sprintf("the reply %q was posted and thread %q does not carry it with the words that were sent", replyID, commentID)
	}
	return false, fmt.Sprintf("the reply %q was posted and thread %q is not in the listing that came back", replyID, commentID)
}

// CreateURL is replies.create asking for the three fields the read-back and the
// report are made of. supportsAllDrives is deliberately absent: replies.create
// does not define it, which M2 measured against the Drive v3 discovery
// document, and a parameter a method does not define is one the server may
// reject and take the whole call with it.
func CreateURL(docID, commentID string) string {
	q := url.Values{"fields": {"id,createdTime,content"}}
	return "https://www.googleapis.com/drive/v3/files/" + docID + "/comments/" + commentID + "/replies?" + q.Encode()
}
