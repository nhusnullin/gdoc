// The follow on the command line: what it refuses before it polls, what one
// window reads as, and what the last line says.
//
// `comments --follow` is the one mode that prints more than one object, so
// every test here reads stdout through runLines, which asks of each line what
// decodeOne asks of the whole stream.

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// manyNewListing is a listing of n top-level ai? comments, all written after
// the cursor these tests hand in. It is built here rather than kept as a
// fixture because the only thing it is for is the ceiling on one line, and a
// twenty-five comment fixture file would say nothing else.
func manyNewListing(n int) string {
	var b strings.Builder
	b.WriteString(`{"comments":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"id":"C%02d","author":{"displayName":"Ada Lovelace","emailAddress":"ada@example.org"},`+
			`"createdTime":"2026-09-06T10:00:00Z","modifiedTime":"2026-09-06T10:00:00Z",`+
			`"content":"ai? question %d","resolved":false,"replies":[]}`, i, i)
	}
	b.WriteString(`]}`)
	return b.String()
}

// windowLine is the one window line the stream carried, and it fails the test
// where there is not exactly one.
func windowLine(t *testing.T, lines []map[string]any) map[string]any {
	t.Helper()
	var out []map[string]any
	for _, l := range lines {
		if l["line"] == "window" {
			out = append(out, l)
		}
	}
	if len(out) != 1 {
		t.Fatalf("want one window line, got %d: %v", len(out), lines)
	}
	return out[0]
}

// namesTheRefusal is what each of the four refusals below has to do: name every
// flag the caller got wrong, and be the command's own refusal rather than the
// parser's. The parser's unknown-flag line names --follow and the flags the
// command does take, so it would satisfy a test that only looked for names.
func namesTheRefusal(t *testing.T, got map[string]any, names ...string) {
	t.Helper()
	msg, _ := got["error"].(string)
	if strings.Contains(msg, "is not a flag this command takes") {
		t.Fatalf("the command refused this itself, and the parser answered instead: %q", msg)
	}
	for _, name := range names {
		if !strings.Contains(msg, name) {
			t.Errorf("the refusal must name %s: %q", name, msg)
		}
	}
}

// A follow with no cursor would hand over every thread in the document at once,
// which is the one-shot read under another name. So it is refused naming the
// flag that is missing, as the wait is.
func TestFollowNeedsSince(t *testing.T) {
	stubWire(t, waitWire(t))

	got, code := runJSON(t, "comments", fixtureDocID, "--follow", "29m")
	if code == 0 || got["ok"] != false {
		t.Fatalf("--follow without --since must be refused: %v (exit %d)", got, code)
	}
	namesTheRefusal(t, got, "--follow", "--since")
}

// --wait and --follow are two answers to the same question, and a line carrying
// both says neither. The refusal names the pair rather than the cursor, because
// the cursor is not what the caller got wrong.
func TestFollowRefusesWait(t *testing.T) {
	stubWire(t, waitWire(t))

	got, code := runJSON(t, "comments", fixtureDocID, "--follow", "29m", "--wait", "9m")
	if code == 0 || got["ok"] != false {
		t.Fatalf("--follow beside --wait must be refused: %v (exit %d)", got, code)
	}
	namesTheRefusal(t, got, "--follow", "--wait")
}

// A witness is an export over the one window a wait ended on. A follow ends on
// no window at all: its lines carry ids and times, and its envelope carries no
// threads, so there is nothing for an export to be matched against.
func TestFollowRefusesWitness(t *testing.T) {
	stubWire(t, waitWire(t))

	got, code := runJSON(t, "comments", fixtureDocID, "--since", waitSince, "--follow", "29m", "--witness")
	if code == 0 || got["ok"] != false {
		t.Fatalf("--follow beside --witness must be refused: %v (exit %d)", got, code)
	}
	namesTheRefusal(t, got, "--follow", "--witness")
}

// The hour is the ceiling on both flags, and each refusal quotes what it was
// given and names the flag it was written on.
func TestFollowRefusesMoreThanAnHour(t *testing.T) {
	for _, value := range []string{"2h", "0", "-1m", "soon"} {
		stubWire(t, waitWire(t))
		got, code := runJSON(t, "comments", fixtureDocID, "--since", waitSince, "--follow", value)
		if code == 0 || got["ok"] != false {
			t.Errorf("--follow %q must be refused: %v (exit %d)", value, got, code)
			continue
		}
		namesTheRefusal(t, got, "--follow", value)
	}
}

// The shape of the mode: one short line per window as it is found, the ordinary
// envelope last, and nothing on a window line that a model would have to read
// words out of. Ids, a thread, a marker and a time, and no author, no content
// and no quote.
func TestFollowWritesOneLinePerWindowAndTheEnvelopeLast(t *testing.T) {
	shortPolls(t)
	stubWire(t, waitWire(t, newsListing(t)))

	lines, code := runLines(t, context.Background(),
		"comments", fixtureDocID, "--since", waitSince, "--follow", "1ms")
	if code != 0 {
		t.Fatalf("comments --follow: exit %d, %v", code, lines)
	}
	if len(lines) != 2 {
		t.Fatalf("want one window line and the envelope, got %d: %v", len(lines), lines)
	}

	w := windowLine(t, lines)
	if w["since"] != waitSince {
		t.Errorf("since = %v, want the cursor the poll asked from", w["since"])
	}
	if w["cursor"] != waitNext {
		t.Errorf("cursor = %v, want the one the next poll asks from", w["cursor"])
	}
	noticed, _ := w["noticed"].(string)
	if _, err := time.Parse(time.RFC3339, noticed); err != nil {
		t.Errorf("noticed = %q, want an RFC3339 instant: %v", noticed, err)
	}
	if w["more"] != float64(0) {
		t.Errorf("more = %v, want nothing held back", w["more"])
	}
	items, _ := w["items"].([]any)
	if len(items) != 3 {
		t.Fatalf("items = %v, want the two comments and the one reply the fixture carries", w["items"])
	}
	for _, item := range items {
		got, _ := item.(map[string]any)
		for _, field := range []string{"id", "thread", "marker", "created"} {
			if got[field] == nil || got[field] == "" {
				t.Errorf("an item carries no %s: %v", field, got)
			}
		}
		for _, field := range []string{"author", "author_domain", "content", "quoted", "replies"} {
			if _, has := got[field]; has {
				t.Errorf("a window line must carry no %s: %v", field, got)
			}
		}
	}

	last := lines[len(lines)-1]
	if last["ok"] != true {
		t.Fatalf("the last line is the envelope, and it says %v", last)
	}
	if _, has := last["line"]; has {
		t.Errorf("the envelope is not a window line: %v", last)
	}
	data := dataOf(t, last)
	if data["cursor"] != waitNext {
		t.Errorf("the envelope's cursor = %v, want the last one handed over", data["cursor"])
	}
	if threads, _ := data["threads"].([]any); len(threads) != 0 {
		t.Errorf("threads = %v, want none: the lines were the answer", data["threads"])
	}
	if waitedOf(t, last)["interrupted"] != false {
		t.Errorf("a deadline is not an interrupt: %v", waitedOf(t, last))
	}
}

// A line is bounded. A document somebody pasted thirty comments into at once is
// a window the reader still has to be able to read, so twenty items go out and
// the rest are counted.
func TestALineListsAtMostTwentyItems(t *testing.T) {
	shortPolls(t)
	f := waitWire(t)
	f.answers = append(f.answers, &answer{method: "GET", match: "/comments?", json: manyNewListing(25), once: true})
	stubWire(t, f)

	lines, code := runLines(t, context.Background(),
		"comments", fixtureDocID, "--since", waitSince, "--follow", "1ms")
	if code != 0 {
		t.Fatalf("comments --follow: exit %d, %v", code, lines)
	}
	w := windowLine(t, lines)
	items, _ := w["items"].([]any)
	if len(items) != 20 {
		t.Errorf("items = %d, want the twenty a line carries", len(items))
	}
	if w["more"] != float64(5) {
		t.Errorf("more = %v, want the five that did not fit", w["more"])
	}
}

// A poll that failed is not retried in silence, here either. The exit code
// follows the last line, and that line carries the last cursor handed over: the
// windows already printed are not asked for again.
func TestFollowsExitCodeFollowsTheLastLine(t *testing.T) {
	shortPolls(t)
	stubWire(t, waitWire(t,
		newsListing(t),
		&answer{method: "GET", match: "/comments?", err: errors.New("drive said 503"), once: true},
	))

	lines, code := runLines(t, context.Background(),
		"comments", fixtureDocID, "--since", waitSince, "--follow", "29m")
	if code == 0 {
		t.Fatalf("a failed poll must fail the run: exit %d, %v", code, lines)
	}
	windowLine(t, lines)
	last := lines[len(lines)-1]
	if last["ok"] != false {
		t.Fatalf("the last line must say ok: false: %v", last)
	}
	if msg, _ := last["error"].(string); !strings.Contains(msg, "503") {
		t.Errorf("the error must carry what failed: %q", msg)
	}
	data := dataOf(t, last)
	if data["cursor"] != waitNext {
		t.Errorf("cursor = %v, want the last one handed over", data["cursor"])
	}
	if waitedOf(t, last)["polls"] != float64(2) {
		t.Errorf("polls = %v, want the two it made", waitedOf(t, last))
	}
}

// Ctrl-C is how a watch ends, so it is an answer and not a crash: the windows
// already printed stand, the envelope is ok, and the flag says which of the two
// quiet endings this was.
func TestAnInterruptedFollowIsAnAnswer(t *testing.T) {
	shortPolls(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	// Cancelled as the second listing is handed over, so the follow has a
	// window behind it when the signal arrives.
	quiet := quietListing()
	quiet.before = cancel
	stubWire(t, waitWire(t, newsListing(t), quiet))

	lines, code := runLines(t, ctx, "comments", fixtureDocID, "--since", waitSince, "--follow", "29m")
	if code != 0 {
		t.Fatalf("an interrupt must still be a successful envelope: exit %d, %v", code, lines)
	}
	w := windowLine(t, lines)
	if w["cursor"] != waitNext {
		t.Errorf("the window line = %v, want the one the fixture advances to", w["cursor"])
	}
	last := lines[len(lines)-1]
	if last["ok"] != true {
		t.Fatalf("the envelope after an interrupt is ok: %v", last)
	}
	if waitedOf(t, last)["interrupted"] != true {
		t.Errorf("interrupted = %v, want true", waitedOf(t, last))
	}
	if dataOf(t, last)["cursor"] != waitNext {
		t.Errorf("cursor = %v, want the last one handed over", dataOf(t, last)["cursor"])
	}
}

// The wait's own words are not the follow's. parseWait reads both flags now,
// and every refusal --wait printed before the follow existed is the same
// sentence, byte for byte: a skill matching on them is matching on these.
func TestTheWaitKeepsItsOwnWords(t *testing.T) {
	for _, c := range []struct {
		value string
		want  string
	}{
		{"soon", `--wait "soon" is not a length of time. Write it as 9m or 90s`},
		{"0", `--wait "0" is not a length of time to look for`},
		{"2h", `--wait "2h" is longer than the 1h0m0s this command will look for`},
	} {
		stubWire(t, waitWire(t))
		got, code := runJSON(t, "comments", fixtureDocID, "--since", waitSince, "--wait", c.value)
		if code == 0 || got["ok"] != false {
			t.Errorf("--wait %q must be refused: %v (exit %d)", c.value, got, code)
			continue
		}
		if msg, _ := got["error"].(string); msg != c.want {
			t.Errorf("--wait %q reads\n%q\nand has always read\n%q", c.value, msg, c.want)
		}
	}
}
