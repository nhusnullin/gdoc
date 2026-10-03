package tty

import (
	"strings"
	"testing"
)

// roleCall is one role under its own name, so a table states what every depth
// writes for it. The function is the method; nothing here reads a constant the
// package declares, because a test that reads the constant follows it wherever
// somebody moves it.
type roleCall struct {
	name string
	fn   func(Style, string) string
}

var roleCalls = []roleCall{
	{"Title", func(s Style, text string) string { return s.Title(text) }},
	{"Key", func(s Style, text string) string { return s.Key(text) }},
	{"Border", func(s Style, text string) string { return s.Border(text) }},
	{"Dim", func(s Style, text string) string { return s.Dim(text) }},
	{"OK", func(s Style, text string) string { return s.OK(text) }},
	{"Fail", func(s Style, text string) string { return s.Fail(text) }},
	{"Warn", func(s Style, text string) string { return s.Warn(text) }},
	{"Chip", func(s Style, text string) string { return s.Chip(text) }},
}

// TestNoColourWritesNoEscapeByte is the row a pipe, a file and every
// non-darwin run land on. Every role hands the text back as it came, so the
// goldens of Task 3 and of internal/panel hold plain bytes.
func TestNoColourWritesNoEscapeByte(t *testing.T) {
	s := NewStyle(NoColour)
	for _, r := range roleCalls {
		got := r.fn(s, "gdoc help")
		if got != "gdoc help" {
			t.Errorf("%s at no colour returned %q, want the text unchanged", r.name, got)
		}
		if strings.ContainsRune(got, 0x1b) {
			t.Errorf("%s at no colour wrote an escape byte: %q", r.name, got)
		}
	}
}

// TestTrueColourIsTheHexOfPaletteFourA states palette 4A as literals: the
// hues of panels-round-two.html, at middle brightness so each reads on a dark
// and on a light terminal. Text itself has no role here, so it keeps the
// colour the person tuned.
func TestTrueColourIsTheHexOfPaletteFourA(t *testing.T) {
	s := NewStyle(TrueColour)
	for _, c := range []struct {
		name string
		got  string
		want string
	}{
		{"Title", s.Title("gdoc v2.10.0"), "\x1b[38;2;62;123;234mgdoc v2.10.0\x1b[0m"},
		{"Key", s.Key("--folder-id"), "\x1b[38;2;184;119;8m--folder-id\x1b[0m"},
		{"Warn", s.Warn("!"), "\x1b[38;2;184;119;8m!\x1b[0m"},
		{"Border", s.Border("│"), "\x1b[38;2;124;130;150m│\x1b[0m"},
		{"Dim", s.Dim("the hint"), "\x1b[38;2;124;130;150mthe hint\x1b[0m"},
		{"OK", s.OK("✓ download"), "\x1b[38;2;47;158;87m✓ download\x1b[0m"},
		{"Fail", s.Fail("✗ verify"), "\x1b[38;2;217;67;79m✗ verify\x1b[0m"},
		{"Chip", s.Chip(" stable "), "\x1b[48;2;47;99;200;38;2;255;255;255m stable \x1b[0m"},
	} {
		if c.got != c.want {
			t.Errorf("%s at truecolor wrote %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestTwoFiftySixIsTheNearestIndex is Apple Terminal, which sets
// TERM=xterm-256color and no COLORTERM. Each index is the nearest xterm colour
// to palette 4A by Lab distance, stated here as the number it is.
func TestTwoFiftySixIsTheNearestIndex(t *testing.T) {
	s := NewStyle(TwoFiftySix)
	for _, c := range []struct {
		name string
		got  string
		want string
	}{
		{"Title", s.Title("gdoc v2.10.0"), "\x1b[38;5;33mgdoc v2.10.0\x1b[0m"},
		{"Key", s.Key("--folder-id"), "\x1b[38;5;130m--folder-id\x1b[0m"},
		{"Warn", s.Warn("!"), "\x1b[38;5;130m!\x1b[0m"},
		{"Border", s.Border("│"), "\x1b[38;5;244m│\x1b[0m"},
		{"Dim", s.Dim("the hint"), "\x1b[38;5;244mthe hint\x1b[0m"},
		{"OK", s.OK("✓ download"), "\x1b[38;5;71m✓ download\x1b[0m"},
		{"Fail", s.Fail("✗ verify"), "\x1b[38;5;203m✗ verify\x1b[0m"},
		{"Chip", s.Chip(" stable "), "\x1b[48;5;26;38;5;231m stable \x1b[0m"},
	} {
		if c.got != c.want {
			t.Errorf("%s at 256 colours wrote %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestSixteenIsTheBasicAnsiCode is the floor: an old terminal, a Linux console,
// tmux without its colour setting. The codes name the terminal's own sixteen,
// so the theme's blue is this blue. Border and Dim are faint rather than a
// colour, because a grey of our own would be unreadable on one background or
// the other.
func TestSixteenIsTheBasicAnsiCode(t *testing.T) {
	s := NewStyle(Sixteen)
	for _, c := range []struct {
		name string
		got  string
		want string
	}{
		{"Title", s.Title("gdoc v2.10.0"), "\x1b[34mgdoc v2.10.0\x1b[0m"},
		{"Key", s.Key("--folder-id"), "\x1b[33m--folder-id\x1b[0m"},
		{"Warn", s.Warn("!"), "\x1b[33m!\x1b[0m"},
		{"Border", s.Border("│"), "\x1b[2m│\x1b[0m"},
		{"Dim", s.Dim("the hint"), "\x1b[2mthe hint\x1b[0m"},
		{"OK", s.OK("✓ download"), "\x1b[32m✓ download\x1b[0m"},
		{"Fail", s.Fail("✗ verify"), "\x1b[31m✗ verify\x1b[0m"},
	} {
		if c.got != c.want {
			t.Errorf("%s at 16 colours wrote %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestAChipIsReverseVideoOnSixteen is why Chip is a role and not a background
// colour. At sixteen there is no pair of colours that reads on both
// backgrounds, so the chip asks the terminal to swap its own two.
func TestAChipIsReverseVideoOnSixteen(t *testing.T) {
	got := NewStyle(Sixteen).Chip(" stable ")
	if want := "\x1b[7m stable \x1b[0m"; got != want {
		t.Errorf("a chip at 16 colours wrote %q, want reverse video %q", got, want)
	}
}

// TestEveryRoleEndsWithTheReset holds that no role leaks into the text after
// it. A screen is one write, and a border left blue would colour the prompt.
func TestEveryRoleEndsWithTheReset(t *testing.T) {
	for _, d := range []Depth{Sixteen, TwoFiftySix, TrueColour} {
		s := NewStyle(d)
		for _, r := range roleCalls {
			got := r.fn(s, "x")
			if !strings.HasSuffix(got, "\x1b[0m") {
				t.Errorf("%s at depth %d wrote %q, which does not end with the reset", r.name, d, got)
			}
			if !strings.HasPrefix(got, "\x1b[") {
				t.Errorf("%s at depth %d wrote %q, which does not open with a code", r.name, d, got)
			}
		}
	}
}

// TestAnEmptyRoleIsStillPlainAtNoColour holds the empty string, which a box
// draws when a row has no right label.
func TestAnEmptyRoleIsStillPlainAtNoColour(t *testing.T) {
	if got := NewStyle(NoColour).Title(""); got != "" {
		t.Errorf("an empty title at no colour wrote %q, want the empty string", got)
	}
}

// TestTheLineCodesAreTheirBytes pins the cursor and line codes as literals.
// They live here because internal/tty is the one package that may write an
// escape byte, which boundary's TestNoEscapeLiteralOutsideTTY holds.
func TestTheLineCodesAreTheirBytes(t *testing.T) {
	for _, c := range []struct {
		name string
		got  string
		want string
	}{
		{"Up(1)", Up(1), "\x1b[1A"},
		{"Up(12)", Up(12), "\x1b[12A"},
		{"ClearBelow", ClearBelow, "\x1b[J"},
		{"ClearLine", ClearLine, "\x1b[K"},
		{"WrapOff", WrapOff, "\x1b[?7l"},
		{"WrapOn", WrapOn, "\x1b[?7h"},
	} {
		if c.got != c.want {
			t.Errorf("%s is %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestUpIsNothingForNoRows is the first draw of a list, which has nothing
// above it to move over. Asking a terminal to go up zero rows is a code that
// means one row in some terminals, so the answer is no code at all.
func TestUpIsNothingForNoRows(t *testing.T) {
	for _, rows := range []int{0, -1} {
		if got := Up(rows); got != "" {
			t.Errorf("Up(%d) wrote %q, want nothing", rows, got)
		}
	}
}

// TestVisibleWidthSkipsEscapes is what a golden test and a box measure with.
// The colour a role wrote takes no column.
func TestVisibleWidthSkipsEscapes(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int
	}{
		{"", 0},
		{"gdoc help", 9},
		{"\x1b[38;2;62;123;234mgdoc help\x1b[0m", 9},
		{"\x1b[2m│\x1b[0m one \x1b[32m✓\x1b[0m", 7},
		{"\x1b[J", 0},
		{"\x1b[?7l" + "ab" + "\x1b[?7h", 2},
		{Up(12) + "ab", 2},
	} {
		if got := VisibleWidth(c.in); got != c.want {
			t.Errorf("VisibleWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestVisibleWidthCountsBoxDrawingAndBrailleAsOne holds the two alphabets a
// panel is drawn with. Each is one column on a terminal, however many bytes it
// takes.
func TestVisibleWidthCountsBoxDrawingAndBrailleAsOne(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int
	}{
		{"┌─┬─┐", 5},
		{"│ ┼ │", 5},
		{"└─┴─┘", 5},
		{"├─┤", 3},
		{"⠋⠙⠹⠸", 4},
		{"✓✗", 2},
	} {
		if got := VisibleWidth(c.in); got != c.want {
			t.Errorf("VisibleWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestAStyleNobodySetWritesNothing holds that the zero Style is the safe one:
// NoColour is the zero Depth, so a Style somebody forgot to build writes no
// escape byte rather than the wrong one.
func TestAStyleNobodySetWritesNothing(t *testing.T) {
	var s Style
	for _, r := range roleCalls {
		if got := r.fn(s, "gdoc help"); got != "gdoc help" {
			t.Errorf("the zero Style's %s returned %q, want the text unchanged", r.name, got)
		}
	}
}
