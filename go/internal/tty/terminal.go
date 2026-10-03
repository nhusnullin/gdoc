package tty

import (
	"io"
	"os"
	"strconv"
	"strings"
)

// isatty is the terminal driver's answer for one file descriptor. A platform
// file sets it: terminal_darwin.go to the TIOCGETA ioctl, terminal_other.go to
// a function that answers false. Nothing else assigns it but a test, and
// IsTerminal always goes through it.
var isatty func(fd uintptr) bool

// winsize is the window's width in columns for one file descriptor, and
// whether anybody could measure it. Set the same way, by the platform file.
var winsize func(fd uintptr) (cols int, ok bool)

// fallbackWidth is the width of a terminal nothing could measure. Eighty
// columns is what a terminal has had since the card it came from, and it is
// the width the stacked layout is drawn at.
const fallbackWidth = 80

// IsTerminal reports whether w is a stream a person is reading. It is true only
// for an *os.File the terminal driver answers for: a buffer has no descriptor,
// and a pipe, a regular file and /dev/null each have one no driver owns.
//
// This is the one definition of a terminal in the tree.
// TestTheIoctlAnswerIsTheAnswer pins that the driver's answer is the whole of
// it, and TestABufferIsNotATerminal, TestAPipeIsNotATerminal and
// TestDevNullIsNotATerminal hold the three streams that are not one.
func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty(f.Fd())
}

// Depth is how much colour a terminal takes. It is asked of the environment and
// never of the terminal, because asking the terminal means reading stdin, and
// gdoc leaves stdin alone.
type Depth int

const (
	// NoColour is the zero value on purpose: a Depth nobody set writes no
	// escape byte.
	NoColour Depth = iota
	Sixteen
	TwoFiftySix
	TrueColour
)

// Colour reads the depth out of env, which is os.Getenv in a command and a map
// in a test.
//
// Four rows, in this order. NO_COLOR set to anything but the empty string, or
// TERM=dumb, is no colour at all, and it wins over everything under it:
// TestNoColourAndADumbTerminalTurnTheColourOff, with
// TestAnEmptyNoColourIsNotSet on the empty case no-color.org calls unset. Then
// COLORTERM naming truecolor or 24bit: TestColortermNamesTrueColour. Then a
// TERM ending -256color: TestATermEndingTwoFiftySixColourIsTwoFiftySix. Then
// sixteen, the floor: TestEverythingElseIsSixteen.
func Colour(env func(string) string) Depth {
	if env("NO_COLOR") != "" || env("TERM") == "dumb" {
		return NoColour
	}
	switch env("COLORTERM") {
	case "truecolor", "24bit":
		return TrueColour
	}
	if strings.HasSuffix(env("TERM"), "-256color") {
		return TwoFiftySix
	}
	return Sixteen
}

// Width is how many columns w has. The window itself first, through the ioctl,
// which is why a person who resizes the terminal gets the screen redrawn at the
// new width on the next command: TestTheIoctlWidthWins. Then a positive
// COLUMNS, which is the only answer available where there is no ioctl:
// TestColumnsAnswersWhenTheIoctlCannot, with TestANonPositiveColumnsIsNoAnswer
// over a zero, a negative number and a word. Then eighty:
// TestEightyIsTheLastAnswer.
//
// A writer that is not a file is never measured, so a golden test sets its own
// width through COLUMNS: TestABufferHasNoDescriptorSoColumnsAnswers.
func Width(w io.Writer, env func(string) string) int {
	if f, ok := w.(*os.File); ok {
		if cols, measured := winsize(f.Fd()); measured && cols > 0 {
			return cols
		}
	}
	if cols, err := strconv.Atoi(env("COLUMNS")); err == nil && cols > 0 {
		return cols
	}
	return fallbackWidth
}
