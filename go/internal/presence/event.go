package presence

import (
	"context"
	"encoding/json"
	"io"
	"sync/atomic"
	"time"

	"gdoc/internal/emit"
)

// schema is the version every line carries. A reader that does not know it
// says so and stops rather than guessing at the fields it can see.
const schema = 1

// The two ways a command reaches dispatch. Nothing else is a via.
const (
	viaTerminal = "terminal"
	viaChat     = "chat"
)

// titleRunes is the cap on the one document word an event carries. A title is
// in the record so the app can fill a row without running gdoc read; the cap
// bounds the line and removes nothing a long title would have told a reader
// that the first 200 runes do not: TestTheTitleIsCappedAtTwoHundredRunes.
const titleRunes = 200

// Event is one line of the record: the fixed, allowlisted shape, never the
// envelope. Every field is in the specification's list and no field is not:
// TestTheEventCarriesOnlyItsListedFields.
//
// OK, Verified and MS are pointers because false and zero are answers here. A
// command that failed writes ok: false, a read-back that found nothing writes
// verified: false, and a command that took no measurable time writes ms: 0.
type Event struct {
	V        int    `json:"v"`
	At       string `json:"at"`
	Gdoc     string `json:"gdoc,omitempty"`
	PID      int    `json:"pid"`
	Run      string `json:"run"`
	Call     int64  `json:"call,omitempty"`
	Event    string `json:"event"`
	Command  string `json:"command,omitempty"`
	Via      string `json:"via,omitempty"`
	Doc      string `json:"doc,omitempty"`
	Wait     bool   `json:"wait,omitempty"`
	Title    string `json:"title,omitempty"`
	OK       *bool  `json:"ok,omitempty"`
	Verified *bool  `json:"verified,omitempty"`
	MS       *int64 `json:"ms,omitempty"`
	Hold     string `json:"hold,omitempty"`
	Rule     string `json:"rule,omitempty"`
	Until    string `json:"until,omitempty"`
}

// calls numbers the commands of one process, so a start and its end pair
// without the clock and without the kernel.
var calls atomic.Int64

// Run is one command in flight. It carries what its end will say that its
// start already said, and the moment it began.
//
// A Run is nil when the flag is off, and every method on it is safe there:
// the hook in dispatch is one call and one deferred call either way.
type Run struct {
	version   string
	command   string
	via       string
	doc       string
	call      int64
	readsBack bool
	started   time.Time
	errOut    io.Writer
}

// Begin writes the start event and hands back the run its end will close.
//
// readsBack is the caller's word that this command reads its own write back,
// and it is the only thing that lets a verified reach the record: update and
// restyle print verified: false to mean no write happened, and that must
// never become a failed read-back in a record meant to last.
// TestVerifiedIsLiftedOnlyWhenTheCallerSaysTheCommandReadsBack.
//
// started is the real clock and never the seam the event's at comes from,
// because the duration is monotonic elapsed time and a wall clock that walks
// backwards would otherwise write a negative one: TestTheDurationIsMonotonic.
func Begin(ctx context.Context, on bool, version, command, doc string, wait, readsBack bool, errOut io.Writer) *Run {
	if !on {
		return nil
	}
	r := &Run{
		version:   version,
		command:   command,
		via:       viaOf(ctx),
		doc:       doc,
		call:      calls.Add(1),
		readsBack: readsBack,
		started:   time.Now(),
		errOut:    errOut,
	}
	appendEvent(Event{
		Gdoc:    r.version,
		Call:    r.call,
		Event:   "start",
		Command: r.command,
		Via:     r.via,
		Doc:     r.doc,
		Wait:    wait,
	}, errOut)
	return r
}

// End writes the end event for a run, with the envelope's ok, the title and
// the verified lifted out of its data, and the elapsed milliseconds.
//
// It takes the whole result and writes four facts out of it, because the
// envelope itself carries document words, warnings, error sentences and
// paths, and none of those may reach a file that lasts:
// TestAnEventCarriesNoWordOfTheEnvelopeButTheTitle.
func (r *Run) End(res emit.Result) {
	if r == nil {
		return
	}
	ok := res.OK
	title, verified := lift(res.Data)
	if !r.readsBack {
		verified = nil
	}
	ms := since(r.started).Milliseconds()
	appendEvent(Event{
		Gdoc:     r.version,
		Call:     r.call,
		Event:    "end",
		Command:  r.command,
		Via:      r.via,
		Doc:      r.doc,
		Title:    title,
		OK:       &ok,
		Verified: verified,
		MS:       &ms,
	}, r.errOut)
}

// Held writes the one event that says a chat write is waiting for a person.
// The rule is the hold's name and nothing else of it travels: not its value,
// not its reason and not the text it holds.
func Held(ctx context.Context, on bool, version, command, doc, hold, rule string, until time.Time, errOut io.Writer) {
	if !on {
		return
	}
	appendEvent(Event{
		Gdoc:    version,
		Event:   "held",
		Command: command,
		Via:     viaOf(ctx),
		Doc:     doc,
		Hold:    hold,
		Rule:    rule,
		Until:   until.UTC().Format(time.RFC3339),
	}, errOut)
}

// Released, Expired and Landed are the three ways a hold leaves, and at most
// one of them follows a held: a send consumed it, its life ran out, or
// another send landed and took it away. Which one happened says nothing about
// whether the send succeeded; the command's own start and end say that.
//
// At most, and not exactly: expired is written by the session's sweep, which
// runs on the next tool call, so a chat that closed with a card open writes no
// ending at all. A hold with no ending is one a reader judges by its until,
// and that is what the app does: PresenceUI's testAHoldThatRanOutLeavesTheMenuBar.
func Released(on bool, version, hold string, errOut io.Writer) {
	endOfHold("released", on, version, hold, errOut)
}

func Expired(on bool, version, hold string, errOut io.Writer) {
	endOfHold("expired", on, version, hold, errOut)
}

func Landed(on bool, version, hold string, errOut io.Writer) {
	endOfHold("landed", on, version, hold, errOut)
}

func endOfHold(kind string, on bool, version, hold string, errOut io.Writer) {
	if !on {
		return
	}
	appendEvent(Event{Gdoc: version, Event: kind, Hold: hold}, errOut)
}

// viaKey is this package's own key type, so no other package can set or read
// the value by accident.
type viaKey struct{}

// WithChat marks a context as the chat session's, so every command under it
// says via: chat. gdoc mcp wraps its session context once; everything else is
// a terminal.
func WithChat(ctx context.Context) context.Context {
	return context.WithValue(ctx, viaKey{}, viaChat)
}

// viaOf is the reader of what WithChat set. A context nobody marked is a
// terminal, which is what a context nobody handed in is too.
func viaOf(ctx context.Context) string {
	if ctx == nil {
		return viaTerminal
	}
	if v, ok := ctx.Value(viaKey{}).(string); ok {
		return v
	}
	return viaTerminal
}

// lift reads the two fields an end event keeps out of a command's data: the
// title and the verified, each only where it is at the top level with the
// type the record expects. A title that is a number and a verified that is a
// word are not these fields, and a data that is no object has neither:
// TestTitleIsLiftedOnlyWhenPresentAsAString.
//
// It goes through JSON rather than reflection because what a skill reads is
// the printed object, so what the record keeps is the printed object's own
// top level, whatever Go shape produced it.
func lift(data any) (string, *bool) {
	if data == nil {
		return "", nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", nil
	}
	var top map[string]json.RawMessage
	if json.Unmarshal(b, &top) != nil {
		return "", nil
	}
	var title string
	if raw, ok := top["title"]; ok {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			title = capRunes(s, titleRunes)
		}
	}
	var verified *bool
	if raw, ok := top["verified"]; ok {
		var v bool
		if json.Unmarshal(raw, &v) == nil {
			verified = &v
		}
	}
	return title, verified
}

// capRunes is the first n runes of s, counted in runes and never in bytes, so
// a title in Cyrillic is cut where a title in English is.
func capRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
