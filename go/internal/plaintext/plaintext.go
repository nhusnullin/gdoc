// Package plaintext holds the two rules about what gdoc may write into a Google
// Docs thread: it opens with the robot, and it carries no markdown.
//
// Every writer asks this package rather than holding a copy, because two copies
// of one regular expression are two rules that drift. internal/reply asks it of
// a reply, internal/propose asks it of the reason a proposal carries as a
// comment, and internal/annotate asks it of the reason a comment carries, adding
// the prefix as propose does.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # The mark is the only record of authorship there is
//
// The Docs API cannot set an author, so everything gdoc writes is signed by
// whoever is logged in. Every reply and every comment gdoc writes opens with the
// robot and one space, with nothing in front of it, and that prefix is the whole
// of what a later run has to go on. comments.Reply.ByGdoc reads the same prefix,
// so a reply one writer sends is one a later listing recognises.
// TestByGdocIsTrueOnlyForAReplyOpeningWithTheRobot in internal/comments is the
// pin on the reading side.
//
// # Identity is never a gate, and the mark decides
//
// Drive says whether a comment's author is the account gdoc is signed in as.
// Nothing may branch on that to decide whether a comment is work, because gdoc
// is signed in as Nail: it would skip every comment he writes. The mark decides
// what gdoc wrote, and the marker in a comment's own text decides what gdoc is
// being asked to do. Neither question is ever asked of the account.
//
// # Robot and Prefix are two constants, and a check asks about Robot
//
// Prefix is what a writer writes. Robot is what a check asks about, because
// every near miss of the prefix carries the same mark: the robot with no space
// behind it, the robot behind a space, the robot behind a newline. A check
// reading the whole prefix answers no on each of them and lets a second mark
// through, and a doubled mark is a lie about who wrote the words.
//
// # The two writers own the mark differently
//
// A reply body arrives with the mark already on it, so reply.Check requires it
// and refuses a body that does not open with exactly it. A proposal's reason
// arrives without it, because propose writes the comment as Prefix plus the
// reason, so Proposal.Check refuses a reason that already carries it. An agent
// following the reply rule when it writes proposals.json has its run refused
// before anything leaves the machine, which is where that mistake belongs.
//
// # No markdown, because a thread renders it literally
//
// Asterisks and backticks arrive as typed and somebody strips them by hand.
// Markdown names the first thing in a body Docs would render that way, so the
// refusal can quote it. Hyphen bullets are deliberately absent from the pattern:
// they read as a list in plain text, so refusing them would cost a reply nothing
// was wrong with. TestMarkdownNamesWhatDocsWouldRenderLiterally and
// TestMarkdownLeavesPlainWordsAlone are the pins.
//
// The heading arm is anchored to a line start, because a hash in the middle of a
// sentence is a number sign somebody typed. That anchoring is why all three
// callers ask the rule behind the mark rather than in front of it: reply.Check's
// own comment holds the argument.
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

// markdown is what Docs renders literally: bold, code, a heading and a
// bracketed link. Hyphen bullets are deliberately absent: they read as a list in
// plain text, so refusing them would cost a reply nothing was wrong with.
//
// The heading arm is anchored to a line start under (?m), because a `#` in the
// middle of a sentence is a number sign somebody typed.
var markdown = regexp.MustCompile("(?m)(\\*\\*|`|^[ ]{0,3}#{1,6}[ \t]|\\[[^\\]\n]*\\]\\([^)\n]*\\))")

// Markdown is the first thing in body that Docs would render literally,
// trimmed, or the empty string when there is none.
func Markdown(body string) string {
	return strings.TrimSpace(markdown.FindString(body))
}
