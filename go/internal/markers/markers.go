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
//     carries one, by line: TestBuildRefusesAMarkerByLine.
//   - internal/plaintext, so `reply` and `annotate` refuse one in a thread,
//     beside the markdown refusal: TestPlaintextRefusesAMarker.
//   - cmd/gdoc's read of `propose --from`, because document text never passes
//     through plaintext: TestProposeRefusesAMarkerInTheFile.
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
// # The closing bracket of [s:ID] is not looked for
//
// Seven of the eight literals are gdoc's markup and nothing else. The eighth is
// a single "]", which a note carries in every markdown link and every numbered
// reference. Looking for it would refuse a note nothing is wrong with, and
// looking for it is not needed: "[s:" always comes first, and it is looked for.
// Pairs holds all eight because the pairs are what a person reading the
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
}

// looked is what Find looks for: every literal in Pairs except the bare "]",
// for the reason the package comment gives.
var looked = []string{"{+", "+}", "{-", "-}", "[[c:", "[[/c]]", "[s:"}

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
