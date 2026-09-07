// Package plaintext is the one rule about the words gdoc writes into a Google
// Docs thread: no markdown.
//
// A thread renders what it is given literally, so asterisks and backticks
// arrive as typed and somebody strips them by hand. v1 refused that in
// gdoc/reply.py, and v2 refuses it in the two places it writes into a thread: a
// reply, and the comment a proposal carries beside it. The rule lives here
// because two copies of one regular expression are two rules that drift.
package plaintext

import (
	"regexp"
	"strings"
)

// Prefix is what every reply and every comment gdoc writes opens with: the
// robot and one space, with nothing in front of it. It is the only record of
// authorship there is, because the Docs API cannot set an author.
const Prefix = Robot + " "

// Robot is the mark alone, without the space. Prefix is what gdoc writes; this
// is what a check asks about, because every near miss of the prefix carries the
// same mark: the robot with no space behind it, or with a space or a newline in
// front of it. A check that reads the whole prefix answers no on each of them,
// and a doubled mark is a lie about who wrote the words.
const Robot = "🤖"

// markdown is what Docs renders literally, ported from v1's assert_plain_text
// with the bracketed link added. Hyphen bullets are deliberately absent: they
// read as a list in plain text, so refusing them would cost a reply nothing was
// wrong with.
//
// The heading arm is anchored to a line start under (?m), because a `#` in the
// middle of a sentence is a number sign somebody typed.
var markdown = regexp.MustCompile("(?m)(\\*\\*|`|^[ ]{0,3}#{1,6}[ \t]|\\[[^\\]\n]*\\]\\([^)\n]*\\))")

// Markdown is the first thing in body that Docs would render literally,
// trimmed, or the empty string when there is none.
func Markdown(body string) string {
	return strings.TrimSpace(markdown.FindString(body))
}
