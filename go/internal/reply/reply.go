// Package reply posts one reply into one comment thread, and reads the thread
// back to see whether it is there.
//
// Two rules live here, and both are about what Docs does with text rather than
// about what gdoc means by it. A thread renders what it is given literally, so
// a reply carrying markdown arrives as asterisks and backticks somebody has to
// strip by hand: v1 refused that in gdoc/reply.py and v2 refuses it here. And
// every reply gdoc writes opens with the robot, which is how a later run tells
// gdoc's own words from Nail's without asking who the credential belongs to.
//
// Nothing here decides whether a thread needs a reply, or what to say in it.
// The body arrives written, is checked for shape alone, and is sent verbatim:
// v1 appended its own marker, and v2 does not, because the mark is the prefix
// the skill already wrote.
package reply

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"gdoc/internal/comments"
)

// Prefix is what every reply gdoc writes opens with: the robot and one space,
// with nothing in front of it. comments.Reply.ByGdoc reads the same prefix, so
// a reply this package sends is one a later listing recognises.
const Prefix = "🤖 "

// markdown is what Docs renders literally, ported from v1's assert_plain_text
// with the bracketed link added. Hyphen bullets are deliberately absent: they
// read as a list in plain text, so refusing them would cost a reply nothing was
// wrong with.
//
// The heading arm is anchored to a line start under (?m), because a `#` in the
// middle of a sentence is a number sign somebody typed.
var markdown = regexp.MustCompile("(?m)(\\*\\*|`|^[ ]{0,3}#{1,6}[ \t]|\\[[^\\]\n]*\\]\\([^)\n]*\\))")

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
	if m := markdown.FindString(body); m != "" {
		return fmt.Errorf("the reply body carries markdown (%q); a Docs thread renders it literally, so it would arrive as typed", strings.TrimSpace(m))
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
	if err := s.PostJSON(ctx, CreateURL(docID, commentID), map[string]any{"content": body}, &answer); err != nil {
		return Result{}, fmt.Errorf("the reply could not be posted to thread %q: %w", commentID, err)
	}
	out := Result{ReplyID: answer.ID, Created: answer.Created}
	if answer.ID == "" {
		// Drive took the write and said nothing useful about it. Calling that a
		// failure would send the skill back to post it again, and the first one
		// may well be sitting in the thread.
		out.Warnings = append(out.Warnings, fmt.Sprintf("Drive answered the reply to thread %q with no reply id, so it could not be looked for in the thread; check the thread before posting again", commentID))
		return out, nil
	}
	verified, warn := inThread(ctx, s, docID, commentID, answer.ID, body)
	out.Verified = verified
	if warn != "" {
		out.Warnings = append(out.Warnings, warn)
	}
	return out, nil
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
