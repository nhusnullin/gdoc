package tty

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// stubIsatty replaces the ioctl answer for one test and puts the real one back
// afterwards, so a test on ubuntu stands in for a terminal on darwin.
func stubIsatty(t *testing.T, answer bool) {
	t.Helper()
	was := isatty
	isatty = func(uintptr) bool { return answer }
	t.Cleanup(func() { isatty = was })
}

// stubWinsize does the same for the width ioctl.
func stubWinsize(t *testing.T, cols int, ok bool) {
	t.Helper()
	was := winsize
	winsize = func(uintptr) (int, bool) { return cols, ok }
	t.Cleanup(func() { winsize = was })
}

// envOf turns a map into the lookup Colour and Width are handed, so no test
// touches the process environment.
func envOf(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

// tempFile is a regular file, which is the one *os.File a test can make
// without a pipe and without a terminal.
func tempFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

// TestABufferIsNotATerminal holds the first half of IsTerminal: a writer that
// is not a file has no descriptor to ask about, so the answer is no before the
// ioctl is reached. Every test in this tree writes into one of these.
func TestABufferIsNotATerminal(t *testing.T) {
	stubIsatty(t, true)
	if IsTerminal(&bytes.Buffer{}) {
		t.Error("a bytes.Buffer has no file descriptor, so it is never a terminal, whatever the ioctl would say")
	}
}

// TestAPipeIsNotATerminal is what a command run by a skill or by a shell's `|`
// gets. The real ioctl answers here on darwin and the stub answers elsewhere,
// and both answer no.
func TestAPipeIsNotATerminal(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if IsTerminal(w) {
		t.Error("a pipe is what every skill hands gdoc; it must get the plain text and the object")
	}
}

// TestDevNullIsNotATerminal is why the character-device bit progress.go read
// before this is not the question. /dev/null is a character device and no
// terminal driver answers for it.
func TestDevNullIsNotATerminal(t *testing.T) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if IsTerminal(f) {
		t.Error("/dev/null is a character device, and the terminal driver does not answer for it")
	}
}

// TestTheIoctlAnswerIsTheAnswer pins that IsTerminal adds nothing of its own to
// the driver's answer. A regular file is a terminal when the variable says so,
// and is not when it does not. That is what lets every later test stub one
// variable and get the screen or the pipe.
func TestTheIoctlAnswerIsTheAnswer(t *testing.T) {
	f := tempFile(t)

	stubIsatty(t, true)
	if !IsTerminal(f) {
		t.Error("the variable answered yes, so IsTerminal must answer yes")
	}

	stubIsatty(t, false)
	if IsTerminal(f) {
		t.Error("the variable answered no, so IsTerminal must answer no")
	}
}

// TestNoColourAndADumbTerminalTurnTheColourOff holds the first row of Colour,
// and it wins over everything under it: a person who set NO_COLOR gets no
// escape byte even on a terminal that would take twenty four bit colour.
func TestNoColourAndADumbTerminalTurnTheColourOff(t *testing.T) {
	for _, row := range []struct {
		what string
		env  map[string]string
	}{
		{"NO_COLOR set to one", map[string]string{"NO_COLOR": "1", "TERM": "xterm-256color", "COLORTERM": "truecolor"}},
		{"NO_COLOR set to any word", map[string]string{"NO_COLOR": "yes", "COLORTERM": "24bit"}},
		{"TERM is dumb", map[string]string{"TERM": "dumb", "COLORTERM": "truecolor"}},
	} {
		if got := Colour(envOf(row.env)); got != NoColour {
			t.Errorf("%s: the colour is off, and the depth must be NoColour, got %d", row.what, got)
		}
	}
}

// TestAnEmptyNoColourIsNotSet is no-color.org's own wording: the variable
// counts when it is set to anything but the empty string. A shell that exports
// an empty NO_COLOR has not asked for anything.
func TestAnEmptyNoColourIsNotSet(t *testing.T) {
	got := Colour(envOf(map[string]string{"NO_COLOR": "", "TERM": "xterm"}))
	if got != Sixteen {
		t.Errorf("an empty NO_COLOR asks for nothing, so this is an ordinary sixteen colour terminal, got %d", got)
	}
}

// TestColortermNamesTrueColour is the second row. Both spellings are in the
// wild and both mean the same thing.
func TestColortermNamesTrueColour(t *testing.T) {
	for _, value := range []string{"truecolor", "24bit"} {
		got := Colour(envOf(map[string]string{"COLORTERM": value, "TERM": "xterm"}))
		if got != TrueColour {
			t.Errorf("COLORTERM=%s is twenty four bit colour, got %d", value, got)
		}
	}
}

// TestATermEndingTwoFiftySixColourIsTwoFiftySix is the third row, and it is
// asked only after COLORTERM, because a true colour terminal usually sets
// TERM=xterm-256color too.
func TestATermEndingTwoFiftySixColourIsTwoFiftySix(t *testing.T) {
	for _, value := range []string{"xterm-256color", "screen-256color"} {
		got := Colour(envOf(map[string]string{"TERM": value}))
		if got != TwoFiftySix {
			t.Errorf("TERM=%s names a two hundred and fifty six colour terminal, got %d", value, got)
		}
	}
}

// TestEverythingElseIsSixteen is the last row, and it is the one a plain
// terminal and an empty environment both land on. Sixteen colours is where the
// chip becomes reverse video, which is Task 5's rule.
func TestEverythingElseIsSixteen(t *testing.T) {
	for _, row := range []struct {
		what string
		env  map[string]string
	}{
		{"a plain xterm", map[string]string{"TERM": "xterm"}},
		{"linux console", map[string]string{"TERM": "linux"}},
		{"nothing set at all", map[string]string{}},
	} {
		if got := Colour(envOf(row.env)); got != Sixteen {
			t.Errorf("%s: sixteen colours is the floor, got %d", row.what, got)
		}
	}
}

// TestTheIoctlWidthWins is the first row of Width. The window the person
// resized is the truth, and nothing in the environment overrules it.
func TestTheIoctlWidthWins(t *testing.T) {
	stubWinsize(t, 132, true)
	got := Width(tempFile(t), envOf(map[string]string{"COLUMNS": "40"}))
	if got != 132 {
		t.Errorf("the window is 132 columns wide, so that is the width, got %d", got)
	}
}

// TestColumnsAnswersWhenTheIoctlCannot is the second row. Every platform but
// darwin has no ioctl here at all, so COLUMNS is the only answer a person can
// give there.
func TestColumnsAnswersWhenTheIoctlCannot(t *testing.T) {
	stubWinsize(t, 0, false)
	got := Width(tempFile(t), envOf(map[string]string{"COLUMNS": "72"}))
	if got != 72 {
		t.Errorf("COLUMNS=72 is the only answer available, got %d", got)
	}
}

// TestANonPositiveColumnsIsNoAnswer keeps a zero, a negative number and a word
// out of the layout, because a width of zero would divide the two column table
// by nothing.
func TestANonPositiveColumnsIsNoAnswer(t *testing.T) {
	stubWinsize(t, 0, false)
	for _, value := range []string{"0", "-5", "wide", "72.5", ""} {
		got := Width(tempFile(t), envOf(map[string]string{"COLUMNS": value}))
		if got != 80 {
			t.Errorf("COLUMNS=%q is not a width, so eighty is the answer, got %d", value, got)
		}
	}
}

// TestEightyIsTheLastAnswer is the third row, and it is what a pipe and a
// cron job get. Eighty columns is the width every terminal has had since the
// card it came from.
func TestEightyIsTheLastAnswer(t *testing.T) {
	stubWinsize(t, 0, false)
	if got := Width(tempFile(t), envOf(map[string]string{})); got != 80 {
		t.Errorf("with no ioctl and no COLUMNS the width is eighty, got %d", got)
	}
}

// TestABufferHasNoDescriptorSoColumnsAnswers is the other half of the first
// row. A writer that is not a file is never handed to the ioctl, so the
// environment answers for it, which is what lets a golden test set its own
// width.
func TestABufferHasNoDescriptorSoColumnsAnswers(t *testing.T) {
	stubWinsize(t, 132, true)
	got := Width(&bytes.Buffer{}, envOf(map[string]string{"COLUMNS": "44"}))
	if got != 44 {
		t.Errorf("a buffer has no descriptor to measure, so COLUMNS=44 answers, got %d", got)
	}
}
