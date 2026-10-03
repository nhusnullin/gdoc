// Breaking a line of words to a width, and cutting the one word that cannot be
// broken.

package panel

import (
	"strings"
	"unicode/utf8"

	"gdoc/internal/tty"
)

// Wrap breaks s into lines no wider than width columns. It breaks at spaces,
// and a word longer than the whole line starts a line of its own and is cut
// where that line ends, which is what a checksum in a reason needs:
// TestAHashIsCutNotShortened. Nothing is shortened, so the pieces put back
// together are the word again and no ellipsis stands for anything a reader
// cannot see.
//
// A line that already fits comes back untouched, runs of blanks and all, which
// is what keeps a caller's own indent in a stacked screen:
// TestWrapKeepsALineThatFits. Past that width the words are rejoined with one
// space each: TestWrapBreaksAtSpaces.
//
// Wrap is for the words a command hands a box, which are plain text. A caller
// colours a whole line after wrapping it, or colours a short run it never
// wraps, and a box pads either by its visible width.
func Wrap(s string, width int) []string {
	if width <= 0 || tty.VisibleWidth(s) <= width {
		return []string{s}
	}
	var lines []string
	cur := ""
	for _, word := range strings.Fields(s) {
		switch {
		case cur == "":
			cur = word
		case tty.VisibleWidth(cur)+1+tty.VisibleWidth(word) <= width:
			cur += " " + word
		default:
			lines = append(lines, cur)
			cur = word
		}
		for tty.VisibleWidth(cur) > width {
			head, tail := cut(cur, width)
			lines = append(lines, head)
			cur = tail
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// cut splits s after width runes. It is reached only for a word wider than a
// whole line, where there is no space to break at, so the only question left is
// where the line ends.
func cut(s string, width int) (head, tail string) {
	at, cols := 0, 0
	for at < len(s) && cols < width {
		_, size := utf8.DecodeRuneInString(s[at:])
		at += size
		cols++
	}
	return s[:at], s[at:]
}

// pad is s with blanks after it until it measures width columns, and s itself
// where it is already that wide or wider. The measurement is the visible one,
// so a row the caller coloured is padded by what it takes on a screen:
// TestAStyledRowIsPaddedByItsVisibleWidth.
func pad(s string, width int) string {
	gap := width - tty.VisibleWidth(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}
