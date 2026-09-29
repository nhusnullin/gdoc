// The progress writer's tests: the plain lines a pipe gets, the redraw a
// terminal gets, and what decides between them.

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

// A buffer, a pipe and a regular file are none of them a terminal, so each
// gets the plain lines. What makes a terminal is the char device bit on the
// file's mode, and nothing else is asked.
func TestOnlyACharDeviceIsATerminal(t *testing.T) {
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

	for name, out := range map[string]any{"a buffer": &bytes.Buffer{}, "a pipe": w, "a file": f} {
		if isTerminal(out) {
			t.Errorf("%s is not a terminal", name)
		}
	}
}

// On a terminal the block is redrawn in place and coloured: a green tick for a
// done step, and the cursor moved back over what was drawn before.
func TestATerminalRunRedrawsInPlaceAndColours(t *testing.T) {
	var buf bytes.Buffer
	p := newLiveProgress(&buf, "gdoc update", true)
	p.Plan("read releases", "choose release")
	p.Start("read releases", "")
	p.Done("nhusnullin/gdoc, 3 listed")
	p.Finish("done.")

	got := buf.String()
	for _, want := range []string{colourGreen + "✓", "\x1b[2A", "read releases", "done.\n"} {
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
	if wantsColour() {
		t.Fatal("NO_COLOR is set, so there is no colour")
	}
	var buf bytes.Buffer
	p := newLiveProgress(&buf, "gdoc update", wantsColour())
	p.Plan("read releases")
	p.Start("read releases", "")
	p.Fail(errors.New("offline"))
	p.Finish("")

	got := buf.String()
	for _, colour := range []string{colourGreen, colourRed, colourDim, colourCyan} {
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
	p := newLiveProgress(&live, "gdoc update", true)
	p.Plan("read releases", "choose release")
	p.Start("read releases", "")
	p.Fail(errors.New("offline"))
	p.Finish("")

	got := live.String()
	off, on := strings.Index(got, wrapOff), strings.LastIndex(got, wrapOn)
	if off < 0 || off > strings.Index(got, clearBelow) {
		t.Errorf("auto-wrap must go off before the first frame: %q", got)
	}
	if on < 0 || on < strings.LastIndex(got, wrapOff) || on > strings.LastIndex(got, clearBelow) {
		t.Errorf("auto-wrap must come back on in the last draw, before its lines: %q", got)
	}
	if strings.Count(got, wrapOff) != 1 || strings.Count(got, wrapOn) != 1 {
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
