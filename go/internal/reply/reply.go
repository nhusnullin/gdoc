// Package reply posts one reply into one comment thread, and reads the thread
// back to see whether it is there.
//
// Nothing here decides whether a thread needs a reply, or what to say in it.
// The body arrives written, in a file the skill wrote, is checked for shape
// alone, and is sent verbatim. Nothing is appended to it either: the mark is
// the prefix the skill has already written.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # The mark is required, and this writer does not add it
//
// Check refuses a body that does not open with exactly the robot and one space.
// That is the opposite of the rule Proposal.Check holds, which refuses a reason
// that already carries the mark, because propose writes the comment as the
// prefix plus the reason. A caller has to know which writer it is talking to:
// an agent following this sentence when it writes proposals.json has its run
// refused before anything leaves the machine. internal/plaintext holds the mark
// itself and what it is for. TestCheckAcceptsAPlainRobotReply and
// TestCheckRefusesEachBadShapeByName are the pins, with
// TestReplyRefusesABodyWithoutTheRobot in cmd/gdoc.
//
// An empty body is said to be empty rather than said to be missing the prefix,
// because that is the sentence somebody can act on, and a body that is empty
// behind the prefix is refused too: a bare signature is a reply that says
// nothing and every read-back would hold over it.
//
// # The markdown rule is asked behind the mark, never in front of it
//
// A Docs thread renders markdown literally, so asterisks and backticks arrive
// as typed and somebody strips them by hand. The heading arm of that rule is
// anchored to a line start, so a robot sitting in front of a "# " moves the
// hash off offset zero and the arm cannot fire: asked of the whole body, a
// heading on the first line passes while the same words on the second line are
// refused, which is one rule firing or not depending on where the author put
// them. So the prefix is trimmed first, which is exact because the line above
// has already required the body to open with exactly it, and the prefix carries
// no markdown of its own, so the trim can only bring a line start into reach.
// Nothing that was refused before starts being accepted, and a heading on the
// first line, which used to pass, is now refused. Proposal.Check asks the same
// rule of the reason alone, for the same reason.
// TestPostRefusesMarkdownBeforeAnythingIsSent is the pin, with
// TestReplyRefusesMarkdownBeforeAnyRequest in cmd/gdoc.
//
// # The check runs before the write, and nothing after the write can fail
//
// A body Docs would mangle never reaches the document. After the write
// everything that goes wrong is a fact about a reply that already exists, so it
// is reported rather than raised: a caller told the run failed is a caller that
// posts the reply a second time. Warnings carry the reason Verified is false.
// TestPostSendsTheBodyVerbatimToTheRepliesURL,
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

// Prefix is what every reply gdoc writes opens with: the robot and one space,
// with nothing in front of it. comments.Reply.ByGdoc reads the same prefix, so
// a reply this package sends is one a later listing recognises, and
// internal/propose signs its comments with the same mark.
const Prefix = plaintext.Prefix

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
// meaning. It refuses an empty body, a body that does not open with exactly the
// robot and one space, and a body carrying markdown, naming what it found.
//
// The order matters: an empty body is said to be empty rather than said to be
// missing the prefix, because that is the sentence somebody can act on.
func Check(body string) error {
	if strings.TrimSpace(body) == "" {
		return errors.New("the reply body is empty")
	}
	if !strings.HasPrefix(body, Prefix) {
		return fmt.Errorf("the reply body does not open with %q, which is how a later run tells gdoc's own replies from everybody else's", Prefix)
	}
	if strings.TrimSpace(strings.TrimPrefix(body, Prefix)) == "" {
		return errors.New("the reply body is empty behind the prefix")
	}
	// The rule is asked behind the prefix, never of the whole body. The heading
	// arm is anchored to a line start, so the robot in front of it moves the
	// first character off offset zero and a reply opening with "# " passes,
	// while the same words on the second line are refused: the rule would fire
	// or not depending on where the author put them. The prefix check above has
	// already run, so the trim is exact, and Prefix carries no markdown of its
	// own, so the trim can only bring a line start into reach: nothing that was
	// refused before starts being accepted. It goes the other way on purpose,
	// and a heading on the first line, which used to pass, is now refused.
	// Proposal.Check asks the same rule of the reason alone, for the same
	// reason.
	if m := plaintext.Markdown(strings.TrimPrefix(body, Prefix)); m != "" {
		return fmt.Errorf("the reply body carries markdown (%q); a Docs thread renders it literally, so it would arrive as typed", m)
	}
	return nil
}

// Post writes one reply and then reads the thread it went into.
//
// The check runs before anything is sent, so a body Docs would mangle never
// reaches the document. After the write the run cannot fail: everything that
// goes wrong from there is a fact about a reply that already exists, and the
// caller needs it reported rather than raised.
func Post(ctx context.Context, s Session, docID, commentID, body string) (Result, error) {
	if err := Check(body); err != nil {
		return Result{}, err
	}
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
