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

// publishSteps are the four parts of a publish, in the order a publish takes
// them, and the only step names the record knows.
//
// They are here rather than in cmd/gdoc because the shape of the record is
// this package's: somebody reading a month's file reads these four words and
// no others, so a caller that drifts from the list writes no line at all
// rather than a name nobody has seen.
// TestAStepNameThatIsNotOneOfTheFourWritesNothing, with cmd/gdoc's
// TestAPublishStepsAreTheFixedNamesInOrder over a real publish.
var publishSteps = []string{"build", "upload", "read_back", "link"}

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
	Step     string `json:"step,omitempty"`
	Index    int    `json:"index,omitempty"`
	Count    int    `json:"count,omitempty"`
	Title    string `json:"title,omitempty"`
	Made     string `json:"made,omitempty"`
	OK       *bool  `json:"ok,omitempty"`
	Verified *bool  `json:"verified,omitempty"`
	Stop     string `json:"stop,omitempty"`
	Line     int    `json:"line,omitempty"`
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
	made      string
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

// Made is the document this run created, named by the caller before the end.
//
// It is one command's fact: a publish is the only run that brings a document
// into the world, and every other command was handed the document it worked
// on, which the event's doc already names. The id is the caller's because the
// id is in the answer rather than in the run, and because cmd/gdoc is the room
// that knows what a Drive id looks like: cmd/gdoc's presenceMade puts the
// value through the same resolver the line's first word goes through, so
// nothing but an id can reach a file that lasts.
// cmd/gdoc's TestAPublishEndNamesTheDocumentItCreated and
// TestOnlyPublishCarriesMade.
//
// It is safe on a nil run, as every method here is, so the hook in dispatch is
// the same lines whether the flag is on or off.
func (r *Run) Made(id string) {
	if r == nil {
		return
	}
	r.made = id
}

// Step writes the one line that says a part of this command's work is
// beginning, with the part's place in the fixed list and the length of that
// list beside its name.
//
// It is written immediately before the part it names begins, because that is
// the whole of what a step is for: a panel saying what a run is doing now. A
// line written after the work would name the part that has just finished, and
// a reader could not tell the two apart. The last step is followed by no line
// of its own: the end says the run finished.
//
// title is what the render put on the cover, which is why the first step
// carries none: at that moment the title is in a note nothing has read yet.
// cmd/gdoc's TestTheFirstStepsEndCarriesTheTitle.
//
// The run comes off the context, where dispatch put it, so the rooms between
// a command and the hook are handed no run and know nothing about the record.
// A context nobody marked and a run under a flag that is off each write
// nothing: TestAStepOutsideARunWritesNothing.
//
// There is no percentage here and nowhere to put one. The two numbers are
// places in a list of four, because what a part of a publish costs is not
// something gdoc knows before it runs it, and a number saying 60% would be a
// guess in a file that lasts: TestNoPercentageOrEstimateIsEverWritten.
func Step(ctx context.Context, name, title string) {
	runOf(ctx).step(name, title)
}

// step is the line itself, safe on a nil run as every method here is.
func (r *Run) step(name, title string) {
	if r == nil {
		return
	}
	index := stepIndex(name)
	if index == 0 {
		return
	}
	appendEvent(Event{
		Gdoc:    r.version,
		Call:    r.call,
		Event:   "step",
		Command: r.command,
		Via:     r.via,
		Doc:     r.doc,
		Step:    name,
		Index:   index,
		Count:   len(publishSteps),
		Title:   capRunes(title, titleRunes),
	}, r.errOut)
}

// stepIndex is a step's place in the fixed list, counted from one, and zero
// for a word that is not one of them.
func stepIndex(name string) int {
	for i, step := range publishSteps {
		if step == name {
			return i + 1
		}
	}
	return 0
}

// End writes the end event for a run, with the envelope's ok, the title and
// the verified lifted out of its data, the stop code of a run that failed, and
// the elapsed milliseconds.
//
// It takes the whole result and writes six facts out of it, because the
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
	stop, line := stopOf(res)
	ms := since(r.started).Milliseconds()
	appendEvent(Event{
		Gdoc:     r.version,
		Call:     r.call,
		Event:    "end",
		Command:  r.command,
		Via:      r.via,
		Doc:      r.doc,
		Title:    title,
		Made:     r.made,
		OK:       &ok,
		Verified: verified,
		Stop:     stop,
		Line:     line,
		MS:       &ms,
	}, r.errOut)
}

// stopOf is the one word a failed end says why in, and the line of the note
// the cause sits on where the cause has one.
//
// Both are the caller's: cmd/gdoc reads the typed failures the packages raise
// and names one of emit's seven codes, and this package judges nothing. What
// it does refuse is a word that is not one of the seven, a sentence included,
// because a reader of the record could not tell a word gdoc meant from a word
// that leaked out of English: TestAStopIsOneOfTheSevenCodesAndNeverASentence.
//
// A run that worked says nothing about why it stopped, whatever its caller
// set, and a line is written only beside a code, because a number alone names
// nothing: TestAStopReachesNoSuccessfulEnd.
func stopOf(res emit.Result) (string, int) {
	if res.OK || !res.Stop.Known() {
		return "", 0
	}
	if res.StopLine <= 0 {
		return string(res.Stop), 0
	}
	return string(res.Stop), res.StopLine
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

// Released, Expired, Landed and Dropped are the four ways a hold leaves, and
// at most one of them follows a held: a send consumed it, its life ran out,
// another send landed and took it away, or the sign-in it was held under
// stopped being the sign-in on the computer. Which one happened says nothing
// about whether the send succeeded; the command's own start and end say that.
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

// Dropped is the hold whose sign-in changed: cmd/gdoc found another token at
// the path, or none, and took the card away rather than letting a person
// approve a write that would go out under the next account. The line says the
// hold is gone and nothing about the account, because no account reaches this
// package: cmd/gdoc's TestTheSignInMarkReachesNoAnswerNoToolAndNoLine.
func Dropped(on bool, version, hold string, errOut io.Writer) {
	endOfHold("dropped", on, version, hold, errOut)
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

// runKey is this package's own key type for the command in flight, so no
// other package can put a run there or read one by accident.
type runKey struct{}

// WithRun puts this command's run on the context. dispatch does it once, where
// it begins the run, so a command that reports a part of its own work needs no
// new argument threaded through the rooms between it and the hook, and those
// rooms go on knowing nothing about the record.
//
// A nil run is put there too, which is the flag being off, so the lookup has
// one shape and the caller has no case to remember:
// TestAStepOutsideARunWritesNothing.
func WithRun(ctx context.Context, r *Run) context.Context {
	return context.WithValue(ctx, runKey{}, r)
}

// runOf is the run WithRun put there, and nil where nobody did.
func runOf(ctx context.Context) *Run {
	if ctx == nil {
		return nil
	}
	r, _ := ctx.Value(runKey{}).(*Run)
	return r
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
