// This file is the cursor: what it encodes, how it is read back, how the next
// one is computed, and the narrowing that keeps a thread from being news twice.
// doc.go holds why it is shaped this way.
package comments

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// cursorVersion is the shape of what String writes. ParseCursor refuses
// anything else rather than guessing: a cursor from a newer binary means the
// caller and the binary disagree about what the instant means, and reading it
// anyway is how a poll quietly reports the wrong window.
const cursorVersion = 1

// Cursor is one instant, plus the ids reported at it: the newest modifiedTime,
// across the comments and their replies, that the run which emitted it saw, and
// every thread whose own newest instant was that one.
//
// The ids are there because the instant alone cannot answer the boundary
// question. Drive writes modifiedTime to the millisecond, so two comments can
// share an instant while only one of them has been reported, and no comparison
// on the instant tells those two comments apart. Ids do.
type Cursor struct {
	At  time.Time
	Ids []string
}

// cursorBody is what the base64 carries. The ids are omitted when there are
// none, and a cursor written before this field existed still reads: the ids
// only ever narrow further, so their absence costs one repeated thread and
// never a lost one. That is why the version did not have to move.
type cursorBody struct {
	V int      `json:"v"`
	T string   `json:"t"`
	I []string `json:"i,omitempty"`
}

// String is the value the caller hands back. UTC always, so the same instant in
// two zones is one cursor, and RawURLEncoding so it carries no character a URL
// or a shell treats as its own.
//
// RFC3339Nano, not RFC3339, and that is the difference between a poll that
// finishes and one that does not. Drive writes modifiedTime with milliseconds.
// Formatting to whole seconds hands the next poll a floor up to 999 ms below
// the activity this run just reported, Drive returns that thread again, and
// NextCursor truncates it back to the same floor: the thread is news on every
// poll for ever. ParseCursor reads both shapes, so a cursor already in a
// caller's hand still parses.
func (c *Cursor) String() string {
	if c == nil {
		return ""
	}
	b, err := json.Marshal(cursorBody{V: cursorVersion, T: c.At.UTC().Format(time.RFC3339Nano), I: c.Ids})
	if err != nil {
		// cursorBody is three fields of the kinds encoding/json cannot fail
		// on. There is no error to report here and nothing that could produce
		// one.
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// since is the value ListURL puts on startModifiedTime, and "" for no cursor. A
// method on the nil pointer so the caller does not repeat the check at every
// page.
//
// The same precision String writes, and for the same reason: a floor rounded
// down to the second re-opens a window this run already read.
//
// The instant goes out as it is, and the bound Drive applies to it is
// inclusive: startModifiedTime is documented as the minimum value of
// modifiedTime. Asking for a window starting a millisecond later would be
// asking Drive not to send a comment modified inside the cursor's own
// millisecond, and losing a comment is the wrong direction to be wrong in. So
// the request stays wide and narrow drops what came back too old or already
// reported.
func (c *Cursor) since() string {
	if c == nil {
		return ""
	}
	return c.At.UTC().Format(time.RFC3339Nano)
}

// narrow drops the comments this cursor has already reported.
//
// Without it the thread whose instant became the cursor comes back on every
// poll: the bound is inclusive, so Drive sends it again, and NextCursor cannot
// advance past an instant it already holds, so the next poll asks the same
// question. The thread is then news for ever, which is the failure String's
// millisecond precision was for. The precision fixed the rounding half of it
// and this is the boundary half.
//
// A comment is kept when anything about it is strictly newer than the cursor,
// its own modifiedTime or any reply's createdTime. The replies are in the set
// for the reason NextCursor reads them: a thread whose modifiedTime Drive did
// not move is still a thread with something new in it.
//
// A comment sitting exactly on the cursor's instant is kept too, unless the
// cursor names its id. Strictly-newer alone would drop it, and that is a lost
// comment rather than a repeated one: two comments can share a millisecond with
// only one of them reported, when the poll landed between the two writes. Then
// the second one is at the cursor's instant, has never been seen, and no
// comparison on instants can say so.
//
// A nil cursor narrows nothing, so a listing with no --since is the wire's
// words in the wire's order.
func (c *Cursor) narrow(raw []RawComment) []RawComment {
	if c == nil {
		return raw
	}
	out := make([]RawComment, 0, len(raw))
	for _, comment := range raw {
		if c.isNews(comment) {
			out = append(out, comment)
		}
	}
	return out
}

// isNews is whether one comment carries an instant this cursor has not seen.
func (c *Cursor) isNews(comment RawComment) bool {
	if c.newer(comment.ModifiedTime) {
		return true
	}
	for _, r := range comment.Replies {
		if c.newer(r.CreatedTime) {
			return true
		}
	}
	if c.reported(comment.ID) {
		// Reported at this instant already. A comment edited twice inside one
		// millisecond is the blind spot left, and it is Drive's precision
		// rather than a choice made here.
		return false
	}
	if c.sameInstant(comment.ModifiedTime) {
		return true
	}
	for _, r := range comment.Replies {
		if c.sameInstant(r.CreatedTime) {
			return true
		}
	}
	return false
}

// reported is whether this cursor names the id among the threads it carried.
func (c *Cursor) reported(id string) bool {
	for _, seen := range c.Ids {
		if seen == id {
			return true
		}
	}
	return false
}

// newer is whether s is strictly after the cursor, and true as well when s is
// not an instant at all. An instant gdoc cannot read is not evidence that
// nothing happened, so the comment carrying it is reported; NextCursor steps
// over the same case for the same reason, and the cost is one repeated thread
// rather than a lost one.
func (c *Cursor) newer(s string) bool {
	at, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return true
	}
	return at.After(c.At)
}

// sameInstant is whether s is the cursor's own instant. An unreadable instant
// is not one: newer has already reported that comment.
func (c *Cursor) sameInstant(s string) bool {
	at, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return false
	}
	return at.Equal(c.At)
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
	return &Cursor{At: at, Ids: body.I}, nil
}

// NextCursor is the newest instant this run saw: the modifiedTime of every
// thread and the createdTime of every reply, whichever is latest. A reply is in
// the set because a thread whose modifiedTime Drive did not move is still a
// thread with something new in it.
//
// It also carries the ids of the threads whose own newest instant is that one,
// which is what narrow reads at the boundary. When the instant has not moved
// those ids are added to the ones the previous cursor held rather than
// replacing them: a thread narrow dropped is a thread the caller was told about
// on an earlier poll, and forgetting its id makes the two threads sharing that
// millisecond take turns being news for ever. When the instant does move the
// older ids go, because every comment behind the new instant is strictly older
// and the strict comparison covers it.
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
	for _, t := range threads {
		at, ok := peak(t)
		if ok && at.After(newest) {
			newest = at
			moved = true
		}
	}

	fresh := idsAt(newest, threads)
	if !moved {
		// Nothing newer than what the caller already had, which includes the
		// case of having had nothing at all.
		if prev == nil {
			return nil
		}
		ids := union(prev.Ids, fresh)
		if len(ids) == len(prev.Ids) {
			return prev
		}
		return &Cursor{At: prev.At, Ids: ids}
	}
	return &Cursor{At: newest, Ids: fresh}
}

// peak is one thread's own newest instant, across its modifiedTime and its
// replies' createdTime, and false when it carries none gdoc can read.
func peak(t Thread) (time.Time, bool) {
	out := time.Time{}
	ok := false
	consider := func(s string) {
		at, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return
		}
		if !ok || at.After(out) {
			out, ok = at, true
		}
	}
	consider(t.Modified)
	for _, r := range t.Replies {
		consider(r.Created)
	}
	return out, ok
}

// idsAt is the ids of the threads whose own newest instant is this one, sorted
// so the same run always writes the same cursor.
func idsAt(instant time.Time, threads []Thread) []string {
	var out []string
	for _, t := range threads {
		if at, ok := peak(t); ok && at.Equal(instant) {
			out = append(out, t.ID)
		}
	}
	sort.Strings(out)
	return out
}

// union is the two id sets together, sorted, and never the same id twice.
func union(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	var out []string
	for _, set := range [][]string{a, b} {
		for _, id := range set {
			if seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
