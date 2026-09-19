// This file is what a house prelude is, by position, and what a heading number
// is, by shape. project.go holds the projection it serves.

package export

import (
	"fmt"
	"regexp"
	"strings"

	"gdoc/internal/docs"
	"gdoc/internal/prelude"
	"gdoc/internal/view"
)

// houseTables is how many tables the house front matter holds: version
// control, revision history and document classification. A span in front of
// the first body heading holding three of them is the house prelude, and a
// span holding fewer is somebody's own front page, which stays.
const houseTables = 3

// headingNumberSeparator is what the house puts between a heading's number and
// its words. house.yaml states the whole format, "{n}-{title}", and
// TestTheHeadingNumberIsTheHouseSeparator reads the file and says so, rather
// than this constant reading it: a test that reads the constant it checks
// follows it wherever somebody moves it.
const headingNumberSeparator = "-"

// numberedHeading is a heading line opening with the number publish wrote into
// it: one or more numbers with dots between them, the house separator, and the
// heading's own words behind it.
var numberedHeading = regexp.MustCompile(`^(#{1,6} )([0-9]+(?:\.[0-9]+)*)` + headingNumberSeparator + `(\S.*)$`)

// route is which shape recognised this tab, and it answers one question beside
// the strip: whether the numbers in front of the headings are gdoc's own.
//
// Only publish writes those numbers, and a document publish made comes back
// through the layout shape alone. A restyle writes none: heading numbering is
// out of that route for its own reasons, which internal/prelude's doc.go
// states. So a "1-" in a document wearing the marker is text somebody typed.
type route int

const (
	// routeNone is a tab holding nothing that looks like a house prelude.
	routeNone route = iota
	// routeMarker is the restyle shape: gdoc's own named range.
	routeMarker
	// routeLayout is the publish shape: the house layout in front of the
	// first body heading. The one route whose heading numbers are gdoc's.
	routeLayout
	// routeUnsure is a tab that looks like a house document and does not
	// match: nothing is stripped and the warning names what stayed.
	routeUnsure
)

// stripPrelude says which of this tab's blocks are the house prelude, what
// each piece of it held, what it could not recognise, and which shape
// recognised it.
//
// Two shapes, and a third that strips nothing.
//
// A refused marker is the third on its own. When gdoc's own record of the
// prelude is there and cannot be read, falling back to the layout would guess
// at exactly the span that is in doubt, so nothing is stripped and no number
// comes off. That is what keeps the restyle rule true: a document restyle made
// never loses a heading number, whether its marker reads or not.
//
// A document restyle made carries gdoc's own named range over its prelude, and
// that range is the answer: the blocks inside it go. A document publish made
// carries no marker, because it was made from a docx rather than styled in
// place, so the answer is the layout: from the start of the body to the end of
// the contents list, and only when the span in front of the first level-one
// heading holds the three house tables. Anything else stays in the file.
//
// The rule is deliberately narrow. An edited cover title still matches,
// because no rule here reads a word of it. A heading somebody added inside the
// cover does not, because the span in front of the first heading is then the
// cover alone, and what comes back is the whole document with a warning naming
// what stood there. Stripping on a guess would take somebody's own front page
// out of their note, and nothing puts it back.
func stripPrelude(t docs.Tab, m *prelude.Marker, refused bool) (func(docs.Block) bool, []Piece, []string, route) {
	if m != nil {
		skip, cut, w := byMarker(t, *m)
		return skip, cut, w, routeMarker
	}
	if refused {
		return nil, nil, nil, routeUnsure
	}
	return byLayout(t)
}

// byMarker is the restyle shape: the blocks that begin inside gdoc's own named
// range.
//
// A contents list a person inserted after that prelude stands outside the
// range and stays where it is. It prints nothing either way, because the
// projection walks paragraphs and tables and steps over a contents element,
// and a range gdoc did not put over it is not gdoc's to take.
func byMarker(t docs.Tab, m prelude.Marker) (func(docs.Block) bool, []Piece, []string) {
	var cut []docs.Block
	for _, b := range t.Body {
		start, ok := blockStart(b)
		if !ok || start < m.Start || start >= m.End {
			continue
		}
		cut = append(cut, b)
	}
	if len(cut) == 0 {
		return nil, nil, []string{fmt.Sprintf(
			"the marker %s covers [%d,%d) and no block of tab %s begins inside it, so nothing was stripped",
			m.ID, m.Start, m.End, t.ID)}
	}
	return skipper(cut), pieces(cut), nil
}

// byLayout is the publish shape: the span from the body's start to the end of
// the contents list, when the three house tables stand in front of the first
// level-one heading.
func byLayout(t docs.Tab) (func(docs.Block) bool, []Piece, []string, route) {
	toc := firstTOC(t.Body)
	if toc < 0 {
		// No contents list is no house prelude. A document gdoc never touched
		// is the common case, and it is not a warning: there is nothing here
		// that looks like a prelude at all.
		return nil, nil, nil, routeNone
	}
	heading, ok := firstHeading(t.Body)
	if !ok {
		// A house document always opens its body with a level-one heading, so
		// a tab with none is not one. Without that heading the span in front
		// of it is the whole tab, every table in the document counts as a
		// house table, and what came out of somebody's note would be their own
		// front page.
		//
		// A note holding no heading at all is the case this gives up on: it is
		// published with the whole house prelude and no Heading 1 behind it,
		// so its prelude stays in the file and the warning names it. Counting
		// the tables over the span in front of the contents list instead would
		// strip the front page of a document whose own three tables stand
		// there, which is the loss this guard exists to refuse.
		cut := t.Body[:toc+1]
		return nil, nil, []string{fmt.Sprintf(
			"tab %s holds a contents list and no level-one heading, so gdoc cannot tell a house prelude from the document's own front page: nothing was stripped, and what stands in front of that list is %s",
			t.ID, describe(pieces(cut)))}, routeUnsure
	}
	tables := countTables(t.Body[:heading])
	if toc > heading || tables < houseTables {
		cut := t.Body[:toc+1]
		return nil, nil, []string{fmt.Sprintf(
			"tab %s holds a contents list, and the span in front of its first level-one heading holds %d of the %d house tables, so gdoc cannot tell a house prelude from the document's own front page: nothing was stripped, and what stands in front of that list is %s",
			t.ID, tables, houseTables, describe(pieces(cut)))}, routeUnsure
	}
	cut := t.Body[:toc+1]
	return skipper(cut), pieces(cut), nil, routeLayout
}

// skipper is the set of blocks to drop, by the identity of what the block
// holds. A block is a struct of three pointers, so the pointer is what says
// this is the same block and not one that reads the same.
func skipper(cut []docs.Block) func(docs.Block) bool {
	set := make(map[any]bool, len(cut))
	for _, b := range cut {
		if k := blockKey(b); k != nil {
			set[k] = true
		}
	}
	return func(b docs.Block) bool {
		k := blockKey(b)
		return k != nil && set[k]
	}
}

func blockKey(b docs.Block) any {
	switch {
	case b.Paragraph != nil:
		return b.Paragraph
	case b.Table != nil:
		return b.Table
	case b.TOC != nil:
		return b.TOC
	}
	return nil
}

// blockStart is where a block begins in its tab's own numbering. A contents
// list carries no index of its own, so it takes the first one inside it.
func blockStart(b docs.Block) (int, bool) {
	switch {
	case b.Paragraph != nil:
		return b.Paragraph.StartIndex, true
	case b.Table != nil:
		return b.Table.StartIndex, true
	case b.TOC != nil:
		for _, inner := range b.TOC.Blocks {
			if start, ok := blockStart(inner); ok {
				return start, true
			}
		}
	}
	return 0, false
}

func firstTOC(bs []docs.Block) int {
	for i, b := range bs {
		if b.TOC != nil {
			return i
		}
	}
	return -1
}

// firstHeading is where the body starts: the first HEADING_1 paragraph, and
// whether the tab holds one at all. The second answer is the one byLayout acts
// on: a tab with no level-one heading has no span in front of it to read.
func firstHeading(bs []docs.Block) (int, bool) {
	for i, b := range bs {
		if b.Paragraph != nil && b.Paragraph.Style == "HEADING_1" {
			return i, true
		}
	}
	return len(bs), false
}

func countTables(bs []docs.Block) int {
	n := 0
	for _, b := range bs {
		if b.Table != nil {
			n++
		}
	}
	return n
}

// pieces is what a stripped span was made of, named by position in the house
// layout and never by its words.
//
// A run of paragraphs is one piece. It is the cover when nothing stripped
// stands in front of it, the contents when the contents list follows it, which
// is where the "Contents" heading sits, and the legend everywhere else, which
// is the span between the tables. A run holding no text at all is not a piece:
// the house layout is half blank paragraphs, and a list of empty strings tells
// a reader nothing.
func pieces(bs []docs.Block) []Piece {
	var out []Piece
	var group []string
	seen := false

	// take empties the run of paragraphs being collected and returns its text.
	take := func() string {
		text := collapse(strings.Join(group, " "))
		group = nil
		return text
	}
	flush := func() {
		text := take()
		if text == "" {
			return
		}
		if seen {
			out = append(out, Piece{Kind: PieceLegend, Text: text})
			return
		}
		out = append(out, Piece{Kind: PieceCover, Text: text})
	}

	for i, b := range bs {
		switch {
		case b.Paragraph != nil:
			group = append(group, plainText(b))
		case b.Table != nil:
			flush()
			seen = true
			out = append(out, Piece{Kind: PieceTable, Text: plainText(b)})
		case b.TOC != nil:
			// The label above the contents list goes into the contents piece,
			// so the list and the word above it are the one thing the page
			// showed rather than two pieces a reader has to put together.
			seen = true
			out = append(out, Piece{Kind: PieceContents, Text: join(take(), plainText(b))})
		}
		if i == len(bs)-1 {
			flush()
		}
	}
	return out
}

// join is a contents label and the list under it as one line, or whichever of
// them there is.
func join(label, list string) string {
	switch {
	case label == "":
		return list
	case list == "":
		return label
	}
	return label + ": " + list
}

// describe is a piece list in one sentence, for the warning about a prelude
// this projection did not recognise.
func describe(ps []Piece) string {
	if len(ps) == 0 {
		return "nothing this projection can name"
	}
	parts := make([]string, 0, len(ps))
	for _, p := range ps {
		parts = append(parts, fmt.Sprintf("%s (%s)", p.Kind, p.Text))
	}
	return strings.Join(parts, ", ")
}

// stripHeadingNumbers takes the "1-" and "1.1-" prefixes publish wrote into
// the headings themselves back out, and records each as a piece.
//
// The number is text in the document, because house.yaml says the numbering is
// literal: Word's own list numbering would renumber the ordinary paragraphs
// between the headings. So a note built from this file and published again
// would be numbered twice over, "1-1-Introduction", unless it comes out here.
//
// It works on the projected line rather than on the run, because a run's text
// is what the comment markers are placed in by index: taking characters out of
// it would move every marker behind them. A line whose number is behind a
// marker keeps it, which is the safe direction: the number stays visible and a
// session takes it out with the marker.
func stripHeadingNumbers(md string) (string, []Piece) {
	lines := strings.Split(md, "\n")
	var out []Piece
	moved := map[string]string{}
	for i, line := range lines {
		m := numberedHeading.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		numbered, words := m[2]+headingNumberSeparator+m[3], m[3]
		lines[i] = m[1] + words
		out = append(out, Piece{Kind: PieceHeadingNumber, Text: numbered})
		if from, to := view.Slug(numbered), view.Slug(words); from != to {
			moved[from] = to
		}
	}
	return repoint(strings.Join(lines, "\n"), moved), out
}

// keptNumbers names the headings that still open the way a house number does,
// for a tab whose prelude gdoc could not recognise. It changes nothing: it says
// what is in the file, because the warning about the prelude that stayed is
// about the front page alone and the session reading it decides about these
// too.
//
// It is a fact and not a verdict, and the wording is the whole of the care
// here. The shape it matches is digits and the house separator, which is what
// "2024-2025 Budget" and "1-on-1 meetings" open with as well, and this route is
// reached by any tab holding a contents list that did not match the house
// layout, most of them documents gdoc never touched. So it says what the
// headings look like and names the one thing that would make them gdoc's, and
// leaves that to the session, which knows where the document came from. A tab
// whose headings open with no digits raises nothing.
func keptNumbers(tab, md string) []string {
	_, numbers := stripHeadingNumbers(md)
	if len(numbers) == 0 {
		return nil
	}
	texts := make([]string, 0, len(numbers))
	for _, p := range numbers {
		texts = append(texts, p.Text)
	}
	return []string{fmt.Sprintf(
		"tab %s keeps %d headings opening with digits and %q (%s): gdoc takes a number off only where the layout shape recognised publish's prelude, so these are house numbers if gdoc publish wrote this document, and the author's own words if it did not",
		tab, len(numbers), headingNumberSeparator, strings.Join(texts, ", "))}
}

// repoint moves the links that pointed at a heading the number came off. The
// target is the heading's own words slugged, so a heading that loses "1-" loses
// it from every link into it as well, and a file whose links land nowhere is
// not a file a session can merge.
func repoint(md string, moved map[string]string) string {
	for from, to := range moved {
		md = strings.ReplaceAll(md, "](#"+from+")", "](#"+to+")")
	}
	return md
}
