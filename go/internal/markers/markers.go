// Package markers holds gdoc's own markers as literals, and the one question
// every route out of the hub asks about a line: does it carry one.
//
// internal/view writes these markers into the text `read` prints and `export`
// writes to a file. They are gdoc's, which is what lets a later session undo
// them: a person can keep an exported file with its markers for as long as they
// like, and any day after, a skill can resolve them. That is decision 3 of the
// v2 milestone 13 specification.
//
// This comment holds why a marker is refused everywhere text goes out. What is
// refused is in the code beside it.
//
// # A marker may travel out of a document and never back in
//
// A marker says a thing about a document: these words are a pending insertion,
// this range carries a comment. In the document itself it says nothing, because
// a document holds suggestions and comments as suggestions and comments. So a
// marker that reaches a document is prose: the brackets and the id arrive as
// typed, and somebody strips them by hand.
//
// Three routes take text from the hub into a document, and each asks this
// package before it sends anything:
//
//   - internal/body, so `build` and `publish` refuse a note whose body still
//     carries one, by line: TestBuildRefusesAMarkerByLine. The question is
//     asked before the note is parsed, so gdoc's own two comments are refused
//     rather than read as the grey mark they look like:
//     TestGdocsOwnCommentsAreMarkersAndNotPencil.
//   - internal/plaintext, so `reply` and `annotate` refuse one in a thread,
//     beside the markdown refusal: TestPlaintextRefusesAMarker.
//   - cmd/gdoc's read of `propose --from`, because document text never passes
//     through plaintext: TestProposeRefusesAMarkerInTheFile. A words proposal
//     has two such fields, the quote and the replacement, and a block proposal
//     has four, its two placement quotes and its content:
//     TestProposeRefusesAMarkerInABlockField.
//
// The rule is written once, here, because three copies of it are three rules
// and the one that drifts opens the door it was written to shut.
//
// # An escaped marker is the author's own text
//
// internal/view escapes a literal marker in the document's own words with a
// backslash, and escapes the backslash as well, so an even run of backslashes
// is the author's text and an odd one ends in gdoc's escape. Find reads the
// same parity, so a note written about the markers themselves is publishable
// and a forgotten marker is not: TestFindLeavesAnEscapedMarkerAlone and
// TestAnAuthorsBackslashDoesNotHideAMarker.
//
// # Three of the twelve literals are not looked for
//
// Nine of them are gdoc's markup and nothing else. The other three are text a
// note is entitled to carry.
//
// One is a single "]", which a note holds in every markdown link and every
// numbered reference. The other two are the one "-->", written twice: it shuts
// the tab comment, it shuts the floating picture's comment, and it shuts every
// grey aside an author writes, because a pencil is a comment. Looking for any
// of the three would refuse a note nothing is wrong with.
//
// Looking for none of them leaves no hole, because each marker's opener always
// comes first and every opener is looked for. The two comment openers are the
// whole of what makes gdoc's own comments markers: "<!-- tab " and
// "<!-- image: " say a thing about a document that the document itself holds
// otherwise, and "<!-- anything else -->" is the author's own aside, carried
// into the document grey. TestAPencilIsNotAMarker is that line, and
// internal/body's TestGdocsOwnCommentsAreMarkersAndNotPencil is the same line
// on the route that builds.
//
// Pairs holds all twelve because the pairs are what a person reading the
// exported file is told, and TestMarkersNamesEachOpener pins them.
package markers

import "strings"

// Pair is one marker: what opens it and what shuts it.
type Pair struct{ Open, Shut string }

// Pairs is every marker internal/view writes, in the order view's own comment
// lists them. They are literals here and literals in view; the two are kept in
// step by TestMarkersNamesEachOpener on this side and by the goldens on that
// one.
var Pairs = []Pair{
	{"{+", "+}"},
	{"{-", "-}"},
	{"[[c:", "[[/c]]"},
	{"[s:", "]"},
	{"<!-- tab ", "-->"},
	{"<!-- image: ", "-->"},
}

// looked is what Find looks for: every literal in Pairs except the bare "]" and
// the two "-->", for the reason the package comment gives.
var looked = []string{"{+", "+}", "{-", "-}", "[[c:", "[[/c]]", "[s:", "<!-- tab ", "<!-- image: "}

// Find is the first marker of gdoc's on this line, and whether there is one. An
// escaped marker is the author's own text and is not one.
//
// The first is the one that comes back because that is the one a person is sent
// to look at. A line with four markers on it is one line to fix either way.
func Find(line string) (string, bool) {
	first, at := "", -1
	for _, m := range looked {
		for i := 0; i+len(m) <= len(line); i++ {
			if line[i:i+len(m)] != m || escaped(line, i) {
				continue
			}
			if at < 0 || i < at || (i == at && len(m) > len(first)) {
				first, at = m, i
			}
			break
		}
	}
	return first, at >= 0
}

// escaped is view's parity read back: the backslashes immediately before this
// position, counted, with an odd run ending in gdoc's escape.
func escaped(line string, at int) bool {
	n := 0
	for i := at - 1; i >= 0 && line[i] == '\\'; i-- {
		n++
	}
	return n%2 == 1
}

// Lines is the first marker in these lines and the line it is on, counting from
// 1, or an empty string and 0 when there is none. It is what a caller holding a
// whole file asks, so that the line number every refusal names is counted in one
// place.
func Lines(text string) (string, int) {
	for i, line := range strings.Split(text, "\n") {
		if m, ok := Find(line); ok {
			return m, i + 1
		}
	}
	return "", 0
}
