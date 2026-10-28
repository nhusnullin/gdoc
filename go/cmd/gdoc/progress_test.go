// The progress writer's tests: the plain lines a pipe gets, the redraw a
// terminal gets, and what decides between them.

package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/tty"
)

// sixteen is the colour a terminal test paints at: the floor every terminal
// has, where each role is one of the terminal's own eight codes. The bytes
// below are stated as literals rather than read from internal/tty, so a role
// that silently changed colour fails here.
var sixteen = tty.NewStyle(tty.Sixteen)

const (
	litGreen = "\x1b[32m"
	litRed   = "\x1b[31m"
	litDim   = "\x1b[2m"
	litBlue  = "\x1b[34m"
	litReset = "\x1b[0m"
)

// stubTerminal makes isTerminal answer the same for every writer, which is how
// a test gets the live list out of newProgress without a terminal.
func stubTerminal(t *testing.T, answer bool) {
	t.Helper()
	was := isTerminal
	isTerminal = func(io.Writer) bool { return answer }
	t.Cleanup(func() { isTerminal = was })
}

// A run whose stderr is not a terminal prints each step once, when it ends,
// and nothing else: no escape code, no pending step, no spinner frame.
func TestAPipedRunPrintsOneLinePerFinishedStep(t *testing.T) {
	var buf bytes.Buffer
	p := newProgress(&buf, "update")
	p.Plan("read releases", "choose release", "download")
	p.Start("read releases", "")
	p.Done("nhusnullin/gdoc, 3 listed")
	p.Start("choose release", "")
	p.Done("v2.1.0, stable, darwin-arm64, from v2.0.0")
	p.Start("download", "gdoc.zip, 6.1 MB")
	p.Done("")
	p.Finish("gdoc v2.1.0 installed.")

	want := "gdoc update\n" +
		"  ✓ read releases        nhusnullin/gdoc, 3 listed\n" +
		"  ✓ choose release       v2.1.0, stable, darwin-arm64, from v2.0.0\n" +
		"  ✓ download             gdoc.zip, 6.1 MB\n" +
		"gdoc v2.1.0 installed.\n"
	if got := buf.String(); got != want {
		t.Errorf("the plain lines are\n%s\nand must be\n%s", got, want)
	}
}

// A failed step is marked with its reason on the next line, and the steps
// after it never ran, so they are not drawn at all.
func TestAFailedStepIsMarkedAndTheRestAreNotDrawn(t *testing.T) {
	var buf bytes.Buffer
	p := newProgress(&buf, "update")
	p.Plan("read checksums", "download", "verify checksum", "replace binary")
	p.Start("read checksums", "SHA256SUMS-v2.1.0")
	p.Done("")
	p.Start("download", "gdoc.zip")
	p.Fail(errors.New("gdoc.zip could not be downloaded, so nothing was replaced"))
	p.Finish("")

	want := "gdoc update\n" +
		"  ✓ read checksums       SHA256SUMS-v2.1.0\n" +
		"  ✗ download             gdoc.zip\n" +
		"    gdoc.zip could not be downloaded, so nothing was replaced\n"
	if got := buf.String(); got != want {
		t.Errorf("the plain lines are\n%s\nand must be\n%s", got, want)
	}
}

// What makes a terminal is the terminal driver's answer, which internal/tty
// gives and this file asks for through one variable. A buffer has no
// descriptor; a pipe, a regular file and /dev/null each have one no driver
// owns. So each of the four gets the plain lines, and the character-device bit
// the old check read, which /dev/null has too, decides nothing any more.
//
// The variable is the whole of the decision made here, at a width that leaves
// room for a box: stub it and newProgress builds the other list, with no
// stream of any kind involved.
func TestOnlyATerminalDriverMakesATerminal(t *testing.T) {
	t.Setenv("COLUMNS", "80")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	f, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()

	for name, out := range map[string]io.Writer{
		"a buffer": &bytes.Buffer{}, "a pipe": w, "a file": f, "/dev/null": null,
	} {
		if isTerminal(out) {
			t.Errorf("%s is not a terminal", name)
		}
	}

	for _, answer := range []bool{true, false} {
		stubTerminal(t, answer)
		var buf bytes.Buffer
		p := newProgress(&buf, "update")
		p.Plan("read releases")
		p.Start("read releases", "")
		p.Done("")
		p.Finish("")
		if live := strings.ContainsRune(buf.String(), 0x1b); live != answer {
			t.Errorf("the driver answered %v and the list drew live=%v: %q", answer, live, buf.String())
		}
	}
}

// On a terminal the block is redrawn in place and coloured: a green tick for a
// done step, and the cursor moved back over what was drawn before. A box of
// two steps is four rows, the two rules included, so four is what the next
// draw moves back over.
func TestATerminalRunRedrawsInPlaceAndColours(t *testing.T) {
	var buf bytes.Buffer
	p := newLiveProgress(&buf, "update", sixteen)
	p.Plan("read releases", "choose release")
	p.Start("read releases", "")
	p.Done("nhusnullin/gdoc, 3 listed")
	p.Finish("done.")

	got := buf.String()
	for _, want := range []string{litGreen + "✓" + litReset, "\x1b[4A", "read releases", "done.\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("a terminal run must write %q: %q", want, got)
		}
	}
	if strings.Contains(got[strings.LastIndex(got, "\x1b[J"):], "choose release") {
		t.Errorf("the last draw must leave out a step that never ran: %q", got)
	}
}

// NO_COLOR keeps the redraw and drops the colour.
func TestNoColorKeepsTheRedrawAndDropsTheColour(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	depth := tty.Colour(os.Getenv)
	if depth != tty.NoColour {
		t.Fatalf("NO_COLOR is set, so there is no colour, and the depth is %v", depth)
	}
	var buf bytes.Buffer
	p := newLiveProgress(&buf, "update", tty.NewStyle(depth))
	p.Plan("read releases")
	p.Start("read releases", "")
	p.Fail(errors.New("offline"))
	p.Finish("")

	got := buf.String()
	for _, colour := range []string{litGreen, litRed, litDim, litBlue} {
		if strings.Contains(got, colour) {
			t.Errorf("NO_COLOR is set and %q was written: %q", colour, got)
		}
	}
	if !strings.Contains(got, "✗ read releases") || !strings.Contains(got, "│ offline") {
		t.Errorf("the failure and its reason must still be drawn: %q", got)
	}
}

func TestSizesAreMegabytes(t *testing.T) {
	for n, want := range map[int64]string{0: "", 999: "1 KB", 6_100_000: "6.1 MB", 12_345_678: "12.3 MB"} {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

// A terminal narrower than a line would wrap it, and a wrapped line is two
// rows on the screen and one in the count the next redraw moves up by. So
// every draw turns auto-wrap off before it moves the cursor and on again after
// its last line, and a plain list writes neither code.
func TestEachDrawTurnsWrapOffBeforeItMovesAndOnAfterItsLines(t *testing.T) {
	var live bytes.Buffer
	p := stillProgress(t, &live, sixteen)
	p.Plan("read releases", "choose release")
	p.Start("read releases", "")
	p.Fail(errors.New("offline"))
	p.Finish("")

	got := live.String()
	// Every draw opens with the wrap-off, so splitting on it is splitting the
	// run into its draws. The four are Plan, Start, and the two Fail makes:
	// the step marked, and the list settled.
	draws := strings.Split(got, tty.WrapOff)
	if draws[0] != "" {
		t.Errorf("a draw opens with the wrap-off, and this run wrote %q before one: %q", draws[0], got)
	}
	if len(draws)-1 != 4 {
		t.Fatalf("this run draws four times, and it turned wrap off %d times: %q", len(draws)-1, got)
	}
	for i, draw := range draws[1:] {
		on := strings.Index(draw, tty.WrapOn)
		if on < 0 {
			t.Fatalf("draw %d must turn auto-wrap back on: %q", i, draw)
		}
		if strings.Count(draw, tty.WrapOn) != 1 {
			t.Errorf("draw %d turns auto-wrap on once: %q", i, draw)
		}
		clear := strings.Index(draw, tty.ClearBelow)
		if clear < 0 || clear > on {
			t.Errorf("draw %d must clear below the cursor before it writes its lines: %q", i, draw)
		}
		if up := cursorUp.FindStringIndex(draw); up != nil && up[0] > clear {
			t.Errorf("draw %d must move the cursor up before it clears: %q", i, draw)
		}
		if !strings.HasSuffix(draw[:on], "\n") {
			t.Errorf("draw %d turns auto-wrap on after its last line: %q", i, draw)
		}
	}

	var plain bytes.Buffer
	q := newProgress(&plain, "update")
	q.Plan("read releases")
	q.Start("read releases", "")
	q.Fail(errors.New("offline"))
	q.Finish("")
	if strings.Contains(plain.String(), "\x1b[") {
		t.Errorf("a plain list writes no escape code: %q", plain.String())
	}
}

// `gdoc update` traps no signal, so a Ctrl-C during the download kills the
// process where it stands. That is between two draws, because a draw is one
// write, so the terminal a person is left with must be wrapping: the last wrap
// code the run has written at any moment between two draws is the wrap-on.
func TestWrapIsOnBetweenEveryTwoDraws(t *testing.T) {
	var buf bytes.Buffer
	p := stillProgress(t, &buf, sixteen)
	for _, c := range []struct {
		what string
		draw func()
	}{
		{"the plan", func() { p.Plan("read releases", "choose release") }},
		{"a step starting", func() { p.Start("read releases", "") }},
		{"a step done", func() { p.Done("30 listed") }},
		{"the next step starting", func() { p.Start("choose release", "") }},
		{"a step failing", func() { p.Fail(errors.New("offline")) }},
	} {
		c.draw()
		got := buf.String()
		if strings.LastIndex(got, tty.WrapOn) < strings.LastIndex(got, tty.WrapOff) {
			t.Errorf("after %s the terminal must be wrapping again: %q", c.what, got)
		}
	}
}

// stillProgress is the live list with a spinner that never turns, which is how
// a test gets exactly the draws it makes: the list is handed the two channels
// Start would have made itself, with the stopped one already closed, so no
// goroutine draws under the test. It is updatescreen_test.go's liveUpdate
// without the recorded width and colour.
func stillProgress(t *testing.T, w io.Writer, style tty.Style) *progress {
	t.Helper()
	p := newLiveProgress(w, "update", style)
	p.stop, p.stopped = make(chan struct{}), make(chan struct{})
	close(p.stopped)
	return p
}

// The width rule decides too, and not only the driver: a window with no room
// for a box gets the plain lines, because the detail cell there is a sliver
// and a step's words would be drawn one character to a row.
func TestANarrowWindowGetsThePlainStepLines(t *testing.T) {
	stubTerminal(t, true)
	t.Setenv("COLUMNS", "30")

	var buf bytes.Buffer
	p := newProgress(&buf, "update")
	if p.live {
		t.Fatal("thirty columns is the plain band, and the list drew itself live")
	}
	p.Plan("read releases")
	p.Start("read releases", "")
	p.Done("nhusnullin/gdoc, 30 listed")
	p.Finish("")

	want := "gdoc update\n  \u2713 read releases        nhusnullin/gdoc, 30 listed\n"
	if got := buf.String(); got != want {
		t.Errorf("a narrow window reads\n%q\nand must read\n%q", got, want)
	}
}
