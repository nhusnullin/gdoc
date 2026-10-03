// The one palette, and every escape code gdoc writes.
//
// No other file under go/ holds an escape byte, which boundary's
// TestNoEscapeLiteralOutsideTTY reads the syntax tree to hold.

package tty

import (
	"strconv"
	"unicode/utf8"
)

// The escape codes that move the cursor or wipe part of the screen. They are
// the same at every depth, because they are not colour: a list that redraws
// itself needs them even where NO_COLOR is set.
const (
	// ClearBelow wipes from the cursor to the end of the screen, so a redraw
	// with fewer lines than the one before leaves nothing behind.
	ClearBelow = "\x1b[J"
	// ClearLine wipes from the cursor to the end of its line, which is what a
	// single waiting line redrawn with a carriage return needs.
	ClearLine = "\x1b[K"
	// WrapOff and WrapOn turn the terminal's auto-wrap off for the life of a
	// moving list and back on when it settles. A wrapped line is two rows on
	// the screen and one in the count a redraw moves up by, so in a narrow
	// terminal the list would crawl down the screen.
	WrapOff = "\x1b[?7l"
	WrapOn  = "\x1b[?7h"

	// reset closes every role. A screen is one write, so a border left
	// coloured would colour the shell prompt under it.
	reset = "\x1b[0m"
)

// Up moves the cursor up rows lines, and writes nothing at all for zero or
// fewer: a terminal reads a count of zero as one, which would eat the line
// above the first draw. TestTheLineCodesAreTheirBytes and
// TestUpIsNothingForNoRows pin both.
func Up(rows int) string {
	if rows <= 0 {
		return ""
	}
	return "\x1b[" + strconv.Itoa(rows) + "A"
}

// paint is one role's colour at the three depths that have one. Each field is
// the body of an SGR sequence, so NoColour needs no entry: it writes nothing.
type paint struct {
	trueColour  string
	twoFiftySix string
	sixteen     string
}

// Palette 4A of docs/design/panels-round-two.html. One palette for dark and
// light terminals, because gdoc cannot ask a terminal for its background:
// asking means reading the reply on stdin, and only `mcp` reads there.
//
// Two rules make one palette enough. Text has no role, so names, descriptions
// and JSON keep the foreground the person tuned. And every accent sits at
// middle brightness, at least 3.4:1 on white and 3.9:1 on Tokyo Night, which
// is enough for a title, a glyph and a short label, and that is all they
// colour.
//
// The 256 index of each is the nearest xterm colour by Lab distance, and at
// sixteen the codes name the terminal's own colours rather than ours, so the
// theme's blue is this blue. TestTrueColourIsTheHexOfPaletteFourA,
// TestTwoFiftySixIsTheNearestIndex and TestSixteenIsTheBasicAnsiCode state
// every value as a literal.
var (
	// paintTitle is #3E7BEA, the blue of a title, a placeholder and the
	// spinner.
	paintTitle = paint{"38;2;62;123;234", "38;5;33", "34"}
	// paintKey is #B87708, the yellow of a section title, a flag and the
	// release notice mark.
	paintKey = paint{"38;2;184;119;8", "38;5;130", "33"}
	// paintMuted is #7C8296, the grey of a border, a muted word and the hint
	// line. At sixteen it is faint rather than a colour, because a grey of our
	// own would vanish on one background or the other.
	paintMuted = paint{"38;2;124;130;150", "38;5;244", "2"}
	// paintOK is #2F9E57, the green of a step that finished.
	paintOK = paint{"38;2;47;158;87", "38;5;71", "32"}
	// paintFail is #D9434F, the red of a step that failed and of an error
	// title.
	paintFail = paint{"38;2;217;67;79", "38;5;203", "31"}
	// paintChip is #2F63C8 behind #FFFFFF: a label with its own background,
	// which is the version in a top border and the channel beside it. At
	// sixteen there is no pair of colours that reads on both backgrounds, so
	// the chip asks the terminal to swap its own two:
	// TestAChipIsReverseVideoOnSixteen.
	paintChip = paint{"48;2;47;99;200;38;2;255;255;255", "48;5;26;38;5;231", "7"}
)

// Style writes one role's colour around a piece of text, at one depth. Build
// it with NewStyle and hand it to internal/panel, which knows the boxes and
// not the colours.
//
// The zero Style is NoColour, which writes nothing: TestAStyleNobodySetWrites
// Nothing. That is on purpose, so a Style somebody forgot to build is the safe
// one rather than the wrong one.
type Style struct {
	depth Depth
}

// NewStyle is the Style for a depth Colour answered with.
func NewStyle(d Depth) Style { return Style{depth: d} }

// Depth is the depth this Style paints at.
func (s Style) Depth() Depth { return s.depth }

// paint wraps text in one role's sequence, or hands it straight back at
// NoColour: TestNoColourWritesNoEscapeByte. Every coloured answer ends with the
// reset, which TestEveryRoleEndsWithTheReset holds.
func (s Style) paint(p paint, text string) string {
	var body string
	switch s.depth {
	case TrueColour:
		body = p.trueColour
	case TwoFiftySix:
		body = p.twoFiftySix
	case Sixteen:
		body = p.sixteen
	default:
		return text
	}
	return "\x1b[" + body + "m" + text + reset
}

// Title is a box title, a placeholder in an example and the spinner.
func (s Style) Title(text string) string { return s.paint(paintTitle, text) }

// Key is a section title inside a box, a flag name and a notice mark.
func (s Style) Key(text string) string { return s.paint(paintKey, text) }

// Border is the box-drawing itself.
func (s Style) Border(text string) string { return s.paint(paintMuted, text) }

// Dim is a muted word: a prompt, a unit, the hint line under a help screen.
func (s Style) Dim(text string) string { return s.paint(paintMuted, text) }

// OK is a step that finished, glyph and words together.
func (s Style) OK(text string) string { return s.paint(paintOK, text) }

// Fail is a step that failed, and the title of an error box.
func (s Style) Fail(text string) string { return s.paint(paintFail, text) }

// Warn is the release notice mark, which shares the yellow of a key because it
// is the same weight of attention.
func (s Style) Warn(text string) string { return s.paint(paintKey, text) }

// Chip is a short label with its own background: the version in a top border,
// the channel beside it, an installed or failed tag.
func (s Style) Chip(text string) string { return s.paint(paintChip, text) }

// VisibleWidth is how many columns s takes on a terminal: every rune outside
// an escape sequence counts as one, which makes the box-drawing and braille a
// panel is drawn with one column each, however many bytes they take.
// TestVisibleWidthSkipsEscapes and
// TestVisibleWidthCountsBoxDrawingAndBrailleAsOne hold both halves.
//
// It is what a box measures a row with and what a golden test checks a drawn
// screen with, so a coloured line and a plain one are the same width.
func VisibleWidth(s string) int {
	cols := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			i += escapeLen(s[i:])
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
		cols++
	}
	return cols
}

// escapeLen is how many bytes the escape sequence at the start of s takes. A
// CSI sequence, which is every code this package writes, runs from ESC [ to
// the first byte in 0x40 to 0x7E. Anything else after the ESC is two bytes, and
// a lone ESC at the end is one, so the walk always moves forward.
func escapeLen(s string) int {
	if len(s) < 2 {
		return 1
	}
	if s[1] != '[' {
		return 2
	}
	for i := 2; i < len(s); i++ {
		if s[i] >= 0x40 && s[i] <= 0x7e {
			return i + 1
		}
	}
	return len(s)
}
