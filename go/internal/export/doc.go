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
// Nothing here reaches the network. Project is a function of one recorded
// read, so a golden file is a specification rather than a snapshot, and File
// puts the gdoc: block in front of a body through internal/frontmatter, the
// one writer of that block. Two things touch the disk: NotePictures, which
// reads the pictures the note at --out already links to, so the export can say
// which of the document's pictures that note already holds, and Write, which
// creates the files and can replace none of them.
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
// A range naming a tab the document does not have at all belongs to no file,
// so it rides along with the first tab and is warned about in internal/view's
// own wording rather than a second one written here.
// TestACommentNamingAMissingTabIsNamed is the pin.
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
// there. TestAnEditedPreludeStaysAndWarns is the pin. A tab holding no
// level-one heading at all is not a prelude either, and is warned about the
// same way: a house document opens its body with one, and without it the span
// in front of the first heading would be the whole tab, with every table in the
// document counted as a house table.
// TestATabWithNoLevelOneHeadingIsNotAPrelude is the pin. Stripping on a guess
// would take somebody's own front page out of their note, and nothing puts it
// back. A document with no contents list at all is not warned about: there is
// nothing there that looks like a prelude.
//
// A note holding no heading at all is the one house document this gives up on.
// internal/body maps the shallowest heading in the note onto Heading 1, so a
// note whose top level is "##" is fine, but a note with none produces no
// level-one heading anywhere and its own prelude then stays in the file with
// that warning. Counting the tables over the span in front of the contents list
// instead would read a foreign document whose three tables stand there as a
// house prelude, which is the loss this guard exists to refuse.
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
// view.Anchor, which is goldmark's own heading id rule and so is the rule the
// note's links will be resolved by. One pass over the targets moves them, not
// one pass per heading: TestRepointDoesNotChainRewrites.
// TestAHeadingLosesItsHouseNumberAndKeepsItsLinks and
// TestTheHeadingNumberIsTheHouseSeparator are the pins, the second stating
// house.yaml's own format as a literal.
//
// It happens only where the layout shape recognised publish's prelude, which is
// the nearest thing to a record that publish wrote this document: publish is
// the one route that numbers a heading. "2024-2025 Budget" and "1-on-1
// meetings" are headings somebody wrote, and a document gdoc never published is
// the common case for an export: taking a prefix off those words would delete
// the author's own, and nothing puts them back.
// TestAPlainDocumentKeepsAHeadingThatOpensWithANumber is the pin.
//
// The layout shape is a position and not a proof, and this is where that
// matters most. A document gdoc never touched, laid out with a title page, the
// three tables and a contents list in front of its first Heading 1, matches it,
// and then its prelude is stripped by position and its heading prefixes come
// off with it. The project accepts that trade for the prelude, and the number
// rides on the same guess. What holds it is that nothing is thrown away
// quietly: every number taken off is a piece carrying the whole heading as it
// stood, "2024-2025 Budget", so a session reading the reply can put it back.
//
// A document restyle made is that document too, and the marker over its prelude
// does not change it. Heading numbering is out of the restyle route, which
// internal/prelude's doc.go states and its own milestone would have to open, so
// every number in front of a heading there is text somebody typed.
// TestARestyledDocumentKeepsAHeadingThatOpensWithANumber is the pin. A marker
// gdoc cannot read is the same answer rather than a fall back to the layout:
// the record of the prelude is the thing in doubt, so nothing is stripped, no
// number comes off, and the warning naming the refusal stays true.
// TestARefusedMarkerStripsNothing is the pin.
//
// Where the shape looked like a house document and did not match, the numbers
// stay and a warning names each of them, so a session taking the cover out of
// the file can take them out with it. The warning is a fact and not a verdict,
// and it is worded as one: this route is reached by any tab holding a contents
// list the house layout did not match, most of them documents gdoc never
// touched, so it says the headings open with digits and the house separator and
// names the one thing that would make them gdoc's. Whether publish wrote this
// document is the session's to know. A tab whose headings open with no digits
// raises nothing. TestAnEditedPreludeStaysAndWarns and
// TestAPlainDocumentWithAContentsListIsNotToldItsNumbersAreTheHouses are the
// pins.
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
//
// # The pictures are paired by order, and by nothing else
//
// A picture's bytes are not in the Docs answer. Its contentUri is a
// googleusercontent.com host the guard does not admit, so the bytes come from
// the docx export, which is a second read of the same document through a
// second route. Nothing crosses those routes: the read has object ids the
// export never mentions and the export has part names the read never mentions.
// So the k-th picture of Objects is the k-th of docx.Media, and order is the
// whole pairing. TestPicturesPairByOrder and
// TestObjectsAreTheTabsPicturesInBodyOrder are the pins, the second stating
// body order: an inline picture where its run stands, a floating one after the
// paragraph it is anchored to, and an equation counted as neither.
//
// When the two counts disagree, every picture is a placeholder and no file is
// written at all. A pairing one place out writes the wrong bytes under the
// right name, and nobody reading the note afterwards can see it.
// TestACountMismatchWritesNoPictureAndWarns is the pin, and
// TestTwoIdenticalPicturesKeepTheirOrder is the other direction: the same
// bytes twice are two pictures, because the position is the fact and the bytes
// are not.
//
// # The note's own picture is kept only once the bytes are measured
//
// A note that already holds the picture should keep its own file, so the SVG
// it was rendered from stays the master. That rests on a published PNG coming
// back from the export byte for byte, which is measurement 2 of
// docs/v2/MEASURED.md and is not measured yet. Until it is, Options.MatchByHash
// is off, every picture is written as a new file, and the reply says the match
// was off rather than saying nothing. TestTheHashMatchIsOffUntilMeasured is
// the pin, with TestADifferentPictureIsWrittenWhenTheMatchIsOn for the other
// answer once it is on. A matched picture carries no bytes, so no writer can
// replace a file the note already has.
//
// The note's own pictures are found through its own image links, read with
// goldmark, resolved against its directory, PNG and JPEG only. A data:
// destination came with the markdown and an http one is a download, which
// nothing here does: both are stepped over. Only the note at --out is read: a
// second note in that folder naming the same document is somebody else's
// pairing. A link that could not be read is named, because this is the one
// read that could tell the session the link is broken.
// TestTheNotesPicturesAreFoundThroughItsLinks and
// TestANotePictureThatCannotBeReadIsNamed are the pins.
//
// # The files on disk, and the one stamp
//
// Nothing an export finds in its way is ever touched, decision 2. A taken name
// takes the next free number, <stem>.2.md and then <stem>.3.md, pictures take
// the next free number under assets/, and there is no flag that could replace
// a file: TestAFreePathIsTaken, TestATakenPathGetsTheNextFreeNumber and
// TestNothingCanReplaceAFile, the last of them read off this package's own
// source, where the word force does not appear and the only call to
// atomicfile.Replace is the stamp.
//
// That rests on atomicfile.Create, which ends in a link rather than a rename,
// so a path that appeared between Plan looking and Write writing is refused
// rather than replaced. Plan and Write are two halves for one reason: a
// picture's line in the body names the file that picture lands in, so every
// path has to be decided before the text exists. TestTheNamesReachTheBody is
// that handover, with the note's own link kept for a picture that matched.
//
// The one thing export writes into a file somebody else owns is the date.
// When the path at --out holds a note whose list names this document, that
// note keeps every other byte and gains exported: {at} on that document's
// entry, and the copy lands beside it carrying exported.note, which is what
// makes every writer refuse a stray copy. A note written before 2026-09-19 is
// schema 1, and the stamp is a write that changes the block, so it comes back
// as schema 2 and the reply says so once:
// TestExportStampsTheNoteAndChangesNoOtherByte and
// TestASchemaOneNoteIsRewrittenAndSaidSo, scenarios 2 and 23. The note is read
// again a moment before it is written, because a live review session may have
// written the same block in between, decision 14.
//
// Two things are refused, at the door, before a byte is written: a note at a
// path this run would land on that names other documents, and front matter
// that does not read. Anything else at a path is a taken path and nothing
// more. Every tab's path takes the same two checks, so one bad path refuses
// the whole run rather than leaving half a document in the hub:
// TestTheDoorChecks and TestATabPathIsCheckedLikeOut.
//
// A further tab's file is <stem>-<title slugged>.md through view.Slug, which
// is the file-name rule and not the heading id rule; a tab whose title slugs to
// nothing is tab-<n> by its position,
// and two tabs with one title take the numbering rule like any other taken
// name. TestTabSlugsCollide is the pin.
package export
