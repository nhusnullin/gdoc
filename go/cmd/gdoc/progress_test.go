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
	p := newProgress(&buf, "gdoc update")
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
	p := newProgress(&buf, "gdoc update")
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
// The variable is the whole of the decision made here: stub it and newProgress
// builds the other list, with no stream of any kind involved.
func TestOnlyATerminalDriverMakesATerminal(t *testing.T) {
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
		p := newProgress(&buf, "gdoc update")
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
// done step, and the cursor moved back over what was drawn before.
func TestATerminalRunRedrawsInPlaceAndColours(t *testing.T) {
	var buf bytes.Buffer
	p := newLiveProgress(&buf, "gdoc update", sixteen)
	p.Plan("read releases", "choose release")
	p.Start("read releases", "")
	p.Done("nhusnullin/gdoc, 3 listed")
	p.Finish("done.")

	got := buf.String()
	for _, want := range []string{litGreen + "✓" + litReset, "\x1b[2A", "read releases", "done.\n"} {
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
	p := newLiveProgress(&buf, "gdoc update", tty.NewStyle(depth))
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
	if !strings.Contains(got, "✗ read releases") || !strings.Contains(got, "    offline\n") {
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
// rows on the screen and one in the count the next redraw moves up by. So a
// live list turns auto-wrap off before its first frame and back on when it
// settles, and a plain list writes neither.
func TestALiveListTurnsWrapOffAndBackOn(t *testing.T) {
	var live bytes.Buffer
	p := newLiveProgress(&live, "gdoc update", sixteen)
	p.Plan("read releases", "choose release")
	p.Start("read releases", "")
	p.Fail(errors.New("offline"))
	p.Finish("")

	got := live.String()
	off, on := strings.Index(got, tty.WrapOff), strings.LastIndex(got, tty.WrapOn)
	if off < 0 || off > strings.Index(got, tty.ClearBelow) {
		t.Errorf("auto-wrap must go off before the first frame: %q", got)
	}
	if on < 0 || on < strings.LastIndex(got, tty.WrapOff) || on > strings.LastIndex(got, tty.ClearBelow) {
		t.Errorf("auto-wrap must come back on in the last draw, before its lines: %q", got)
	}
	if strings.Count(got, tty.WrapOff) != 1 || strings.Count(got, tty.WrapOn) != 1 {
		t.Errorf("auto-wrap goes off once and on once: %q", got)
	}

	var plain bytes.Buffer
	q := newProgress(&plain, "gdoc update")
	q.Plan("read releases")
	q.Start("read releases", "")
	q.Fail(errors.New("offline"))
	q.Finish("")
	if strings.Contains(plain.String(), "\x1b[") {
		t.Errorf("a plain list writes no escape code: %q", plain.String())
	}
}
