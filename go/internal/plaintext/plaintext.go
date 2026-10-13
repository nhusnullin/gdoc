// Package plaintext holds the two rules about what gdoc may write into a Google
// Docs thread: it opens with the mark in effect, and it carries no markdown.
//
// Every writer asks this package rather than holding a copy, because two copies
// of one regular expression are two rules that drift. internal/reply asks it of
// a reply, internal/propose asks it of the reason a proposal carries as a
// comment, and internal/annotate asks it of the reason a comment carries, adding
// the mark as propose does.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # The mark is the only record of authorship there is
//
// The Docs API cannot set an author, so everything gdoc writes is signed by
// whoever is logged in. Every reply and every comment gdoc writes opens with the
// mark in effect and one space, with nothing in front of it, and that opening is
// the whole of what a later run has to go on. comments.Reply.ByGdoc reads the
// known marks and the mark in effect, so a reply one writer sends is one a later
// listing recognises, whichever colleague's run wrote it.
// TestByGdocIsTrueForAKnownMarkOrTheMarkInEffect in internal/comments is the
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
// # The mark is a setting, and three marks are always known
//
// The mark a run writes is a value, not a character this package decides:
// --mark names it, settings.yaml names it, and the built-in default is the
// four-dot mark, which is decision B3 of the v2 milestone 16 specification.
// What this package owns is the list every gdoc recognises whoever wrote the
// words, 🤖, ⁘ and ∴, and the rule about what a mark may be at all.
// KnownMarks hands that list out as a new slice, so no caller can change it:
// TestTheKnownMarksAreTheRobotAndTheTwoDots and TestTheDefaultMarkIsTheFourDot
// state all four as literals.
//
// Any value may be set, which is decision B2, and the price is that a mark
// outside the known list reads as a person's own words to everybody else's
// gdoc. CheckMark refuses only what a thread would show wrong: a line break, a
// value Docs renders as markdown, one of gdoc's own markers, and more than
// eight characters counted as runes. The empty mark passes, and it writes the
// words alone: TestCheckMarkRefusesWhatAThreadWouldShowWrong,
// TestCheckMarkAcceptsEachKnownMarkTheEmptyMarkAndEightCharacters and
// TestAnEmptyMarkPrefixesNothing.
//
// Prefixed is the one writer of the mark, and the reading side is one loop
// asked three ways: MarkOpening names the mark a text opens with, OpensWithMark
// is that answer as a yes or no, and CutMark takes one mark and one space off a
// body that arrived with one already. The reader asks the known list plus the
// mark in effect, so a colleague who picked their own mark still recognises
// their own replies: TestOpensWithMarkKnowsEveryKnownMarkAndTheMarkInEffect,
// TestMarkOpeningNamesTheMarkItFound and TestCutMarkTakesOneMarkAndOneSpace.
//
// A writer that refuses a reason already carrying a mark asks MarkOpening, so
// the refusal names the character it found rather than the mark this run would
// have written.
//
// # A reader asks the mark without the space behind it
//
// A check asks about the mark alone, because every near miss of what a writer
// wrote carries the same mark: the mark with no space behind it, the mark
// behind a space, the mark behind a newline. A check reading the mark and its
// space together answers no on each of them and lets a second mark through,
// and a doubled mark is a lie about who wrote the words.
//
// The check lives here for the same reason the pattern does: every reader of
// the mark asks this package rather than writing the prefix test out again.
// internal/comments asks it of a reply and cmd/gdoc of a thread's opening
// comment, so the day a mark is added there is one line to change. Two tests hold that
// between them: TestOpensWithMarkKnowsEveryKnownMarkAndTheMarkInEffect pins
// the answers, and TestTheMarkIsReadInOnePlace in go/boundary reads the tree
// and fails on a prefix check written out again anywhere but the writers
// that refuse or take the mark off a caller's own input.
//
// # All three writers add the mark
//
// A reason and a reply body both arrive as words, because propose, annotate and
// reply each write the comment as the mark in effect plus those words. One
// convention, so an agent writing proposals.json and an agent writing a reply
// follow the same sentence.
//
// What each does with a mark the caller typed differs, and on purpose.
// Proposal.Check and Annotation.Check refuse a reason that opens with one: that
// file is written for gdoc and the refusal lands before anything leaves the
// machine. reply takes one off with CutMark and says so, because a reply body is
// prose a person may have signed by hand and every release before v2.11.0 asked
// the skill to type the mark itself. reply's own package comment holds how long
// that tolerance lasts.
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
// callers ask the rule of the words rather than of the mark and the words
// together: reply.Check's own comment holds the argument.
//
// # No marker of gdoc's own, for the same reason
//
// Marker sits beside Markdown and asks internal/markers the same question about
// the same thing. A marker says a document holds a pending suggestion or a
// comment range; written into a thread it is prose, and the brackets and the id
// arrive as typed. A marker travels out of a document and never back in, which
// is decision 3 of v2 milestone 13, and reply, annotate and propose each ask
// this one line after the markdown rule. An escaped marker is the author's own
// text, so a reply explaining the markers is still writable.
// TestPlaintextRefusesAMarker and TestMarkerLeavesPlainWordsAlone are the pins,
// with TestProposeRefusesAMarkerInTheFile in cmd/gdoc for the one route that
// does not come through here.
package plaintext

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"gdoc/internal/markers"
)

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

// Marker is the first of gdoc's own markers in body, or the empty string when
// there is none. An escaped one is the author's text and is not one.
//
// It is beside Markdown because it is the same rule about the same thing: a
// thread renders what it is given, so a marker arrives with its brackets and
// its id as typed. internal/markers holds why every route out of the hub asks,
// and TestPlaintextRefusesAMarker is the pin.
func Marker(body string) string {
	m, _ := markers.Lines(body)
	return m
}

// DefaultMark is the mark gdoc writes when neither an argument nor the settings
// file names one. It is the four-dot mark, decision B3 of the v2 milestone 16
// specification: the robot stays a known mark forever, because every reply
// written before this release carries it, and what a new run writes is the
// quieter one. TestTheDefaultMarkIsTheFourDot states it as a literal.
const DefaultMark = "⁘"

// markRunes is the ceiling on a mark, counted in runes. Eight is room for a
// short word like "Bot:" and no room for a sentence, because a mark is what
// every line of gdoc's in a thread opens with and a long one is a reader's
// attention spent on the same characters over and over.
const markRunes = 8

// known is the list every gdoc recognises, whichever colleague's run wrote the
// words. It is decision B2: any value may be set, and a mark outside this list
// is read as a person's own words by everybody else's gdoc, which is the price
// of free choice. The order is the robot first, because it is the one old
// replies carry.
var known = []string{"🤖", DefaultMark, "∴"}

// KnownMarks is that list, as a new slice on every call so that no caller can
// change it. A caller handed the one slice could sort it or write over an
// entry, and every later reader would inherit that.
// TestTheKnownMarksAreTheRobotAndTheTwoDots states the three as literals and
// writes over what it is handed.
func KnownMarks() []string {
	return append([]string(nil), known...)
}

// CheckMark is the whole rule about what a mark may be, and it names what it
// found. The empty mark passes: writing the words alone is a choice a
// colleague may make, and the specification says what it costs.
//
// Each refusal is a thread showing the mark wrong. A line break opens the words
// on the second line and leaves the mark alone on the first. Markdown arrives
// as typed, because a thread renders it literally, and the question is asked of
// the mark with words behind it: the heading arm is anchored to a line start,
// so "#" alone is a number sign and "# the register" is a heading, which is
// what a mark in front of words makes of it. A marker of gdoc's says a thing
// about a document, and a thread turns it into prose. More than eight runes is
// the ceiling markRunes holds.
//
// TestCheckMarkRefusesWhatAThreadWouldShowWrong has one subtest per refusal,
// and TestCheckMarkAcceptsEachKnownMarkTheEmptyMarkAndEightCharacters is the
// other direction.
func CheckMark(v string) error {
	if v == "" {
		return nil
	}
	if strings.ContainsAny(v, "\r\n") {
		return errors.New("the mark holds a line break, and a mark is what one line opens with; pick a mark with no line break in it")
	}
	if m := Markdown(v + " words"); m != "" {
		return fmt.Errorf("the mark holds %q, which a Docs thread renders literally; pick a mark without markdown in it", m)
	}
	if m := Marker(v + " words"); m != "" {
		return fmt.Errorf("the mark holds %q, which is one of gdoc's own markers and reads as prose in a thread; pick another mark", m)
	}
	if n := utf8.RuneCountInString(v); n > markRunes {
		return fmt.Errorf("the mark is %d characters, and a mark may be eight at most; pick a shorter mark", n)
	}
	return nil
}

// Prefixed is what gdoc writes into a thread: the mark, one space, then the
// words. The empty mark writes the words alone, with no space in front of
// them: TestAnEmptyMarkPrefixesNothing.
func Prefixed(mark, words string) string {
	if mark == "" {
		return words
	}
	return mark + " " + words
}

// OpensWithMark answers whether a text opens with a mark gdoc recognises:
// one of the known three, or the mark in effect when a person set one of their
// own. It is not a receipt, because anybody can type the character; what says
// gdoc wrote a thread entry is the id in internal/chat's own record of what
// this process wrote.
//
// Leading whitespace is stepped over, because a reply that starts on its second
// line is still a reply that opens with the mark. An empty mark in effect adds
// nothing to the list, so that person's own replies are false here, which is
// what decision B1 says the empty mark costs.
//
// TestOpensWithMarkKnowsEveryKnownMarkAndTheMarkInEffect pins the answers.
func OpensWithMark(text, inEffect string) bool {
	_, found := MarkOpening(text, inEffect)
	return found
}

// MarkOpening is the same question answered with the mark itself: the one the
// text opens with, or the empty string and false when it opens with none. The
// longest wins where two of them match, for the reason CutMark takes the
// longest off.
//
// It is here because a writer refusing a reason that already carries a mark has
// to say which mark it found, and a refusal naming the wrong character is one a
// colleague cannot act on. OpensWithMark and CutMark are both this question
// asked a shorter way, so all three answer from one loop rather than three:
// TestMarkOpeningNamesTheMarkItFound.
func MarkOpening(text, inEffect string) (string, bool) {
	trimmed := strings.TrimLeft(text, " \t\r\n")
	longest := ""
	for _, m := range marks(inEffect) {
		if strings.HasPrefix(trimmed, m) && len(m) > len(longest) {
			longest = m
		}
	}
	return longest, longest != ""
}

// CutMark takes one mark and one following space off the front, and says
// whether it did. The mark is a known one or the mark in effect, the same list
// OpensWithMark asks, and leading whitespace is stepped over for the same
// reason: the two are asked together, and a pair that disagreed about a body
// whose first line is blank would say the mark is there and then leave it on.
//
// One mark comes off and no more. A body carrying two is a body the writer
// signs once, rather than one that quietly loses words nobody looked at:
// TestCutMarkTakesOneMarkAndOneSpace.
func CutMark(text, inEffect string) (string, bool) {
	mark, found := MarkOpening(text, inEffect)
	if !found {
		return text, false
	}
	trimmed := strings.TrimLeft(text, " \t\r\n")
	return strings.TrimPrefix(strings.TrimPrefix(trimmed, mark), " "), true
}

// marks is the list a reader asks: the known three, and the mark in effect when
// it is one of a person's own. The empty mark adds nothing, because every text
// opens with the empty string.
func marks(inEffect string) []string {
	if inEffect == "" {
		return known
	}
	return append(KnownMarks(), inEffect)
}
