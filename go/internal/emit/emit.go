// Package emit is the one output contract: every command prints exactly one
// JSON object to stdout and exits 0 iff ok. Spec: "Output contract".
//
// Three fields of a Result are not in that object, and all three are the
// caller's rather than this package's. A Result with Screen set carries words
// a person has already read, so a caller whose stdout is a terminal may print
// nothing there. Stop and StopLine say in one fixed word why a failed run
// failed, and on which line of a note where the cause has one, for the
// presence record to keep. Print still writes the object every time it is
// called, and none of the three is ever in it:
// TestScreenNeverReachesTheObject and TestStopNeverReachesTheObject. The
// screen rule itself, which runs the terminal check, is cmd/gdoc's run(), and
// the seven codes and what names each of them are stop.go here and cmd/gdoc's
// stop.go.
package emit

import (
	"encoding/json"
	"io"
)

// Result is the one object every command prints. Error carries what went
// wrong, Warnings what went right in a way the reader still has to know about,
// Version names the release that printed it, and the optional fields are
// absent rather than empty so a skill can tell "nothing to say" from "said
// nothing".
//
// Version is absent for a binary built from a checkout, because such a binary
// belongs to no release and naming one would be a guess. The caller sets it:
// emit knows the shape of the envelope and nothing about how gdoc was built.
type Result struct {
	OK       bool     `json:"ok"`
	Data     any      `json:"data,omitempty"`
	Error    string   `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Version  string   `json:"version,omitempty"`
	// Screen says this result's whole answer is words a person already read,
	// so a caller on a terminal may drop the object. It is the one field that
	// is not in the object: a reader of the object cannot tell a screen from
	// any other answer, and no skill has to learn it.
	// TestScreenNeverReachesTheObject.
	Screen bool `json:"-"`
	// Stop says in one fixed word why a failed run failed, and StopLine the
	// line of the note the cause sits on where the cause has one. Both follow
	// Screen: a caller sets them, the object never carries them, and no skill
	// has to learn them. The presence record is their one reader.
	// TestStopNeverReachesTheObject.
	Stop     StopCode `json:"-"`
	StopLine int      `json:"-"`
}

// Print writes the result as one JSON object and a newline. HTML escaping is
// off because the strings here are paths, URLs and Drive ids, and `<` in a
// path helps nobody read it.
func Print(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}

// ExitCode is the other half of the output contract: the process exits 0 if and
// only if the object it printed says ok.
func ExitCode(r Result) int {
	if r.OK {
		return 0
	}
	return 1
}

// Lines is a writer that prints one JSON object per line, and it is the one
// crack in "exactly one object". It exists for `comments --follow`, the poll
// held open: a window found after four minutes is news now, and a caller that
// had to wait for the envelope to read it could not act on it until the watch
// ended. So each window goes out on its own line as it is found, and the
// envelope is the last line, with the exit code that agrees with it.
//
// The shapes of those lines are the caller's, not this package's: a window is
// cmd/gdoc's follow.go and nothing here knows a cursor from a marker. What is
// here is the promise a reader needs, one object, one newline, nothing else on
// the line: TestLinesWritesOneObjectPerLine.
//
// It is handed out rather than fetched, and to one command. cmd/gdoc's run
// builds the one writer there is from stdout and passes it down the dispatch
// chain; the chat route passes none, because a session speaks JSON-RPC on that
// stream: TestOnlyFollowWritesLines and TestTheMcpRouteHandsNoLineWriter.
type Lines struct {
	enc *json.Encoder
}

// NewLines holds the writer the lines go to. The encoder is made once rather
// than per line, so the writer and its escaping are settled for the run.
func NewLines(w io.Writer) *Lines {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &Lines{enc: enc}
}

// WriteLine writes one object and a newline. The escaping is Print's, for the
// same reason: a `<` in a path or a title helps nobody read it as <.
func (l *Lines) WriteLine(v any) error {
	return l.enc.Encode(v)
}
