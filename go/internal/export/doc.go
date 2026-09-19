// Package export is a Google Doc as Markdown in the hub: one file per tab, the
// house prelude taken out and listed, and the pictures named by the caller.
//
// It projects and decides nothing. What comes back is the files, the pieces it
// removed with the text each held, and the warnings: a placeholder where
// content was not read, a prelude it did not recognise, and what a file holding
// a footnote costs on its way to becoming a note. Whether a difference matters,
// which side is right, and whether a suggestion is taken are the session's, and
// no field here answers any of them.
//
// Nothing here reaches the network or the disk. Project is a function of one
// recorded read, so a golden file is a specification rather than a snapshot,
// and File puts the gdoc: block in front of a body through internal/frontmatter,
// the one writer of that block.
//
// This comment holds why the projection takes out what it takes out, with the
// test that pins each rule. What it prints is in the code beside it.
//
// # One file is one tab
//
// A document with three tabs is three files, decision 11, so the projection is
// asked one tab at a time and no file carries the "<!-- tab -->" line read
// prints. The comment ranges go with the tab they are anchored in: a range
// naming another tab would be warned about here as a range naming a tab the
// document does not have, and that tab is in the file beside this one.
// TestTwoTabsGiveTwoProjections is the pin.
//
// A link to a heading in another tab is named by its id rather than by that
// heading's words, which is what internal/view does for a heading it cannot
// see. The heading is in another file, so "#slug" into this one would point at
// nothing.
//
// # The house prelude is stripped by position, and never by its words
//
// A document restyle made carries gdoc's own named range over its prelude, and
// that range is the answer: the blocks that begin inside it go.
// TestARestyledPreludeIsStrippedOverItsNamedRange is the pin. A document
// publish made carries no marker, because it was built as a docx and uploaded,
// so the answer is the layout: from the start of the body to the end of the
// contents list, and only when the span in front of the first level-one
// heading holds the three house tables.
// TestAPublishedPreludeIsStrippedByLayout is the pin.
//
// Nothing else is stripped. An edited cover title still matches, because no
// rule here reads a word of it; a heading somebody added inside the cover does
// not, because the span in front of the first heading is then the cover alone,
// and the whole document stays in the file with a warning naming what stood
// there. TestAnEditedPreludeStaysAndWarns is the pin. Stripping on a guess
// would take somebody's own front page out of their note, and nothing puts it
// back. A document with no contents list at all is not warned about: there is
// nothing there that looks like a prelude.
//
// A piece is named by its position in the house layout. A run of paragraphs is
// one piece: the cover when nothing stripped stands in front of it, the
// contents when the contents list follows it, which is where the "Contents"
// label sits, and the legend everywhere else, which is the span between the
// tables. A run holding no text is not a piece, because the layout is half
// blank paragraphs and a list of empty strings tells a reader nothing. The text
// is the document's own words, not what this file would have printed for them:
// a piece says what stood there.
//
// # A heading gives its house number back
//
// house.yaml numbers a heading with literal text, "1-Scope", because Word's own
// list numbering would renumber the ordinary paragraphs between the headings.
// A note built from a file that kept those numbers and published again would be
// numbered twice over, "1-1-Scope", so the prefix comes off and is listed as a
// piece. The links that pointed at that heading move with it, through
// view.Slug, which is the one copy of the slug rule.
// TestAHeadingLosesItsHouseNumberAndKeepsItsLinks and
// TestTheHeadingNumberIsTheHouseSeparator are the pins, the second stating
// house.yaml's own format as a literal.
//
// It is done on the projected line rather than on the run, because a run's text
// is what the comment markers are placed in by index, and taking characters out
// of it would move every marker behind them. A number standing behind a marker
// keeps it, which is the safe direction: it stays visible, and the session
// resolving the marker takes it out with the rest.
//
// # What the file carries that the read does not
//
// A picture is written as the file the caller says it wrote, raw, where the
// picture stands. A picture in a paragraph of its own is then a line of its
// own, which is where publish puts one. A chip carries the address it points
// at behind its label, because the session merging this file into a note has no
// second read to go back to. Both are internal/view's Options, so there is one
// escaping of one document rather than two. TestAPictureIsWrittenWhereItStands
// and TestAChipExportsLabelAndTarget are the pins, with
// TestAPlainDocumentIsTheTextAndTheBlock over a document that holds no prelude
// at all: the text read prints, its markers, and a front matter holding the
// gdoc: block and nothing else. The author's own keys are the session's to add
// when it makes the file a note.
//
// A footnote is written as read writes it, "[^1]" with the notes under a rule
// at the end, and three warnings ride with it: publish refuses a note holding
// one, a suggestion inside a footnote arrives as plain text
// (docs/backlog/suggestions-inside-footnotes.md), and a comment anchored inside
// one carries no marker (docs/backlog/comment-anchors-in-headers-and-footnotes.md).
// TestAFootnoteIsWrittenAndWarned is the pin.
package export
