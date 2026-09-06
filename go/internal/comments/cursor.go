// The cursor is the whole of --since, and it is deliberately small.
//
// It encodes one instant: the newest activity the last run saw. The binary
// emits it, the caller hands it back on the next poll, and nothing writes it
// anywhere. A live session holds it in memory and it dies with the session;
// anything that has to survive a session lives in the front matter, and this
// does not.
//
// It is opaque on purpose. A caller that reads the instant out and does
// arithmetic on it has made the encoding a contract, and the encoding is not
// one. base64url of a small JSON object is enough to carry a version, so a
// later shape can be refused by name rather than misread.
package comments

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// cursorVersion is the shape of what String writes. ParseCursor refuses
// anything else rather than guessing: a cursor from a newer binary means the
// caller and the binary disagree about what the instant means, and reading it
// anyway is how a poll quietly reports the wrong window.
const cursorVersion = 1

// Cursor is one instant: the newest modifiedTime, across the comments and their
// replies, that the run which emitted it saw.
type Cursor struct {
	At time.Time
}

// cursorBody is what the base64 carries.
type cursorBody struct {
	V int    `json:"v"`
	T string `json:"t"`
}

// String is the value the caller hands back. UTC always, so the same instant in
// two zones is one cursor, and RawURLEncoding so it carries no character a URL
// or a shell treats as its own.
func (c *Cursor) String() string {
	if c == nil {
		return ""
	}
	b, err := json.Marshal(cursorBody{V: cursorVersion, T: c.At.UTC().Format(time.RFC3339)})
	if err != nil {
		// cursorBody is two fields of the two kinds encoding/json cannot fail
		// on. There is no error to report here and nothing that could produce
		// one.
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// since is the value ListURL puts on startModifiedTime, and "" for no cursor. A
// method on the nil pointer so the caller does not repeat the check at every
// page.
func (c *Cursor) since() string {
	if c == nil {
		return ""
	}
	return c.At.UTC().Format(time.RFC3339)
}

// ParseCursor reads what String wrote.
//
// Every failure is an error naming the problem, and never a silent fall back to
// no cursor at all. Reading from the beginning would answer a poll with every
// thread in the document, and a session reads that as news.
func ParseCursor(s string) (*Cursor, error) {
	if s == "" {
		return nil, errors.New("the cursor is empty; leave --since off to read from the beginning")
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		// Padded base64url is what an encoder that does not know this one
		// writes. Reading it costs a line and refusing it costs a support
		// question.
		raw, err = base64.URLEncoding.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("the cursor %q is not one this binary wrote: it does not decode", s)
		}
	}
	var body cursorBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("the cursor %q is not one this binary wrote: it does not hold JSON", s)
	}
	if body.V != cursorVersion {
		return nil, fmt.Errorf("the cursor %q is version %d, and this binary writes and reads version %d", s, body.V, cursorVersion)
	}
	at, err := time.Parse(time.RFC3339, body.T)
	if err != nil {
		return nil, fmt.Errorf("the cursor %q carries %q, which is not an RFC 3339 instant", s, body.T)
	}
	return &Cursor{At: at}, nil
}

// NextCursor is the newest instant this run saw: the modifiedTime of every
// thread and the createdTime of every reply, whichever is latest. A reply is in
// the set because a thread whose modifiedTime Drive did not move is still a
// thread with something new in it.
//
// prev is the floor, not just the fallback. A run that saw only older activity
// keeps the cursor it was given: a cursor that goes backwards makes the next
// poll report what this one just reported.
//
// A time that does not parse is stepped over rather than failing the run. The
// cursor is a convenience for the next poll, and losing one thread's instant
// costs one repeated thread; failing the whole listing costs the review.
func NextCursor(prev *Cursor, threads []Thread) *Cursor {
	newest := time.Time{}
	if prev != nil {
		newest = prev.At
	}
	moved := false
	consider := func(s string) {
		at, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return
		}
		if at.After(newest) {
			newest = at
			moved = true
		}
	}
	for _, t := range threads {
		consider(t.Modified)
		for _, r := range t.Replies {
			consider(r.Created)
		}
	}
	if !moved {
		// Nothing newer than what the caller already had, which includes the
		// case of having had nothing at all.
		return prev
	}
	return &Cursor{At: newest}
}
