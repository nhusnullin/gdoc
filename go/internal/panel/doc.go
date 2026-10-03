// Package panel draws the boxes a person reads in a terminal: a titled box, a
// name beside its words or stacked above them, and the width rule that chooses
// between the two.
//
// It exists because of the 2026-10-03 decision that on a terminal a help
// screen is for the person. internal/tty answers what one stream is; this
// package turns strings into lines; cmd/gdoc decides which screen to draw.
// docs/v2/DECISIONS.md holds the decision.
//
// # It knows no command
//
// Nothing here knows what gdoc's commands are, which of them read and which
// write, or what a flag is. A caller hands Box rows of text and gets lines
// back. That is what lets the goldens in testdata stand for every screen in
// the tree: they are drawn from a sample this package's own test writes, not
// from the command table.
//
// # The width rule
//
// Layout answers which of three layouts a width gets: TwoColumns at eighty
// columns and over, Stacked from fifty to seventy-nine, and Plain under fifty,
// where a caller draws no box at all and prints the plain lines a pipe gets.
// Every edge is a literal in TestTheLayoutBandsAreTheirEdges.
//
// DrawWidth is how wide the box itself is: the window up to a hundred columns,
// and a hundred past that, because a line wider than that is more than an eye
// reads in one go. New puts every width through it, so no Panel can be wider:
// TestABoxIsNeverWiderThanAHundred.
//
// Plain is a band, not a refusal. A box asked for at forty-four columns is
// drawn, which is what the narrow golden holds; the band is what a caller
// consults before it asks.
//
// # A box is three kinds of rule and two kinds of row
//
// The top rule carries the box's title and, at its right end, an optional
// label: TestTheTitleSitsInTheTopBorder, with TestABoxWithNoTitleIsAPlainRule
// on a border nobody titled. A Section is the same rule across the middle with
// its name in it: TestASectionIsARuleWithItsName.
//
// Between the rules a Line is one line across the box and a Pair is a name in
// the left cell and words in the right one. Where a box holds pairs, the rules
// around them carry the joint the column crosses them at: a down joint where
// the column opens, a cross where a section carries it through, an up joint
// where it closes, and nothing at all in a box with no pair in it:
// TestTheColumnJointsAreDownCrossAndUp and TestABoxWithNoPairHasNoJoint.
//
// The column sits where WithColumn put it, or three past the widest left cell
// where nobody named one: TestTheColumnFitsTheWidestLeft. A section name long
// enough to reach the column takes the joint's own cell, because the name is
// what a person reads: TestATitleThatReachesTheColumnTakesTheJoint. A caller
// that wants its joints names a column past its longest section name, which is
// what the eighty-column golden does.
//
// Nothing is ever drawn over a corner. A title too wide for its border is
// dropped, and so is a label the title already reaches:
// TestATitleTooWideForTheBorderIsDropped and
// TestALabelThatWouldRunIntoTheTitleIsDropped.
//
// # The text is the caller's and the drawing is ours
//
// A box paints its own drawing, its title and its label, through the roles in
// internal/tty, and hands a caller's text through untouched. So cmd/gdoc
// decides that a flag is a key and a placeholder is a title, and this package
// never names a colour.
//
// A row the caller coloured is padded by what it takes on a screen rather than
// by how many bytes it holds, so a coloured line and a plain one are one
// width: TestAStyledRowIsPaddedByItsVisibleWidth.
//
// # Wrapping, and the one word that cannot be wrapped
//
// Wrap breaks at spaces: TestWrapBreaksAtSpaces. A line that already fits
// comes back untouched, runs of blanks and all, which is what keeps a caller's
// own indent in a stacked screen: TestWrapKeepsALineThatFits.
//
// A word longer than the whole line, which is what a checksum in a reason is,
// starts a line of its own and is cut where that line ends. Nothing is
// shortened: the pieces put back together are the word again, and no ellipsis
// stands for anything a reader cannot see: TestAHashIsCutNotShortened. Wrap is
// for plain text; a caller colours a whole line after wrapping it, or colours a
// short run it never wraps.
//
// Box wraps a row that runs past its cell too, so a box stays square when a
// caller forgets. In the ordinary path the caller has already wrapped to
// TextWidth or ColumnWidth, and that wrap hands the line straight back.
//
// # The goldens
//
// testdata holds the same sample screen at a hundred, eighty, sixty and
// forty-four columns with no colour, which is the width rule end to end, and
// at eighty columns at each of the three depths, which is the bytes a terminal
// gets: TestTheScreenAtEveryWidth and TestTheScreenAtEveryDepth. A golden that
// is not there is written and the test fails saying so, because a recorded
// screen is something a person reads once.
//
// TestNoLineIsWiderThanItsBox walks every golden with tty.VisibleWidth, so a
// coloured screen is measured the way a terminal measures it and no line can
// creep past the box it was drawn in.
//
// # What is not here
//
// No command, no flag and no colour of its own. No read of stdin and no
// network. And no escape byte: every one of them comes from internal/tty,
// which boundary's TestNoEscapeLiteralOutsideTTY holds.
package panel
