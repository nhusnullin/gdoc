// Package view is the document as the text read prints, and the same document
// as the tree --structure prints.
//
// Both are projections of one docs.Document. Neither is the other's summary:
// the text is what the AI reads, and the structure is the same content with the
// character indexes a write would need to address it. Nothing here judges. A
// pending suggestion is marked as pending, a comment range is marked as a
// range, and what any of it means is the skill's.
//
// The projection is deterministic, which is the property every rule below
// stands on: the same document gives the same bytes, so a golden file is a
// specification rather than a snapshot. TestGolden is the pin, over the five
// fixtures in testdata, and TestTheProjectionIsDeterministic asks the same
// document twice. Two rules keep it true where the answer itself carries no
// order: ranges that open at one index are ordered by id
// (TestRangesOpeningTogetherAreOrderedByID), and a close comes before an open
// at the same index, so two ranges that touch do not read as two that nest
// (TestTwoTouchingRangesDoNotReadAsNested).
//
// This comment holds why the projection prints what it prints, with the test
// that pins each rule. What it prints is in the code beside it.
//
// # The markers, and what each one means
//
//	{+text+}[s:ID]                   a pending suggested insertion, with its id
//	{-text-}[s:ID]                   a pending suggested deletion
//	[[c:ID]]text[[/c]]               the range a comment is attached to, by
//	                                 Drive comment id
//	<!-- tab t.0: Title -->          the tab that follows, printed only when
//	                                 there is more than one
//	# to ######                      a HEADING_1 to HEADING_6 paragraph
//	[image] [drawing] [equation]     content this read does not take
//	[object]
//	[person: Name] [date: Sep 9,     a smart chip, with the label the document
//	2026] [link: Title]              shows
//	[auto text: PAGE_NUMBER]         the four remaining members that hold no
//	[page break] [column break]      text of their own
//	[rule]
//	[unknown: member]                a paragraph element gdoc has never seen,
//	                                 named by its member
//	[^1]                             a footnote reference, with its text under
//	                                 a --- rule at the end
//
// TITLE and SUBTITLE are plain paragraphs on purpose: the title is on the
// envelope already, and a document whose first line is its own title reads
// twice. A heading wins over a bullet, because a list item styled as a heading
// is a heading and printing both markers would say it is two things.
//
// Every placeholder run also appends a warning naming what the read did not
// take from it, so a policy with eight person chips and three page breaks
// answers with eleven warnings. A skill that reads a non-empty warnings list as
// a degraded read has to read the messages rather than count them.
// TestAPlaceholderIsAWarning, TestEveryDroppedElementNowPrintsAndWarns and
// TestAnUnnamedElementPrintsItsMemberAndWarns are the pins. [object] is the
// embedded object the decoder could not classify, and calling it an image would
// be a guess. Reading any of them is
// docs/backlog/read-pictures-and-drawings.md.
//
// A run carrying both an insertion id and a deletion id prints twice, as the
// deletion and then the insertion, because that is what Docs shows. The
// characters exist once, so the comment marker opens on the first copy alone
// and the second copy raises no warning of its own.
// TestTextSuggestedAndThenSuggestedAwayPrintsTwiceAndIsMarkedOnce is the pin.
//
// The index the markers are placed at counts UTF-16 code units, because that is
// what a Docs index is: a typographic quote is one unit and three bytes, so
// counting bytes would put every marker after it in the wrong place.
// TestTheIndexCountsUTF16CodeUnits is the pin.
//
// # Every marker is escaped, and so is the backslash
//
// A literal {+, {-, +}, -}, [[ or ]] in the document's own text is written with
// a backslash in front of it. That is not tidiness. Without it a document that
// quotes one of gdoc's own markers makes the AI read somebody's sentence as a
// pending suggestion, and there is no way for it to tell.
// TestEveryMarkerIsEscapedInTheDocumentsOwnText is the pin, and
// TestEscapePairsHoldsEveryMarker is the other half: escapePairs has to stay in
// step with the constants above it, so a marker added without its escape fails
// rather than reaching the text bare.
//
// The backslash is escaped too, as a pair. Without that the encoding cannot be
// read back: a backslash the author typed in front of a marker looks like the
// one gdoc writes, so a real comment anchor after a word ending in a backslash
// reads as a literal and is dropped, and a sentence the author wrote as \{+text+}
// reads as a pending suggestion gdoc never marked. Both directions hand a
// marker to the wrong side, which is the thing the escaping exists to prevent.
// The rule for a reader is a parity: an even run of backslashes is the author's
// own text and the marker behind it is gdoc's, an odd run ends in gdoc's escape
// and the marker behind it is the author's.
// TestTheDocumentsOwnBackslashIsEscaped and
// TestARealMarkerAfterTextEndingInABackslashIsNotEscaped are the two halves.
//
// The escape advances by one rune rather than two. Two literals can share a
// character: "{-}" is "{-" and then "-}", so consuming both halves of the first
// pair walks straight past the second and leaves a closing marker in the text
// the document never had. TestEscapingLeavesNoMarkerBehindWhenTwoOverlap is the
// pin.
//
// escapeAt is one function because two callers need it, the document's own text
// and a chip's label, and two copies would be two rules with one of them
// drifting.
//
// The escaping is per run. A marker whose two halves fall in two runs, and a
// document character sitting against one of gdoc's own markers, still reach the
// text unescaped. That limit is written down rather than guessed at, in
// docs/backlog/escaping-across-run-boundaries.md.
//
// # A chip's placeholder carries the label the document shows
//
// [person] alone tells a reader somebody is there and not who, which for a
// policy naming its owner is the fact that mattered. The label goes through the
// same escaping the document's own text does, for the same reason: a calendar
// entry titled [[c:X]] would otherwise read back as a comment anchor gdoc never
// wrote, and nothing downstream could tell.
// TestAChipLabelIsEscapedLikeAnyOtherText is the pin. The rest of a chip's
// detail, an email address or a link's target, reaches --structure instead.
//
// A chip inside a pending insertion prints inside that insertion's markers,
// carrying the insertion's own id, because the decoder reads the two suggestion
// id lists off every member. TestAChipInsideASuggestionCarriesItsMarker is the
// pin. It does not reach the pending listing, which walks text runs alone:
// docs/backlog/suggestions-on-elements-are-not-listed.md.
//
// Two things about the label are not escapeAt over the string as written.
//
//   - A person chip shown as an address is labelled with the address. The Docs
//     reference documents name as what is shown "instead of the person's email
//     address", so a chip displaying the address carries no name at all, and
//     reading name alone printed a bare [person] on exactly the chip whose
//     identity was on screen. The decoder falls back to email, and
//     TestTheSevenElementsDecodeIntoRuns in internal/docs is the pin: the
//     fixture every other element test reads carries a name, so the fallback
//     has a chip of its own.
//   - The label is escaped in a window that carries the placeholder's own
//     closing bracket. escapeAt looks one rune ahead, so a string's last rune is
//     always written bare, and a file somebody named "Q3 plan [draft]" printed as
//     [link: Q3 plan [draft]], with a ]] in the text gdoc never wrote.
//     TestALabelEndingInABracketDoesNotMergeWithThePlaceholders is the pin. A
//     newline inside a label becomes a space rather than nothing: the document's
//     own text is written in chunks that carry the paragraph break, and a label
//     has no chunking, so dropping it glues the words either side together.
//     TestANewlineInALabelBecomesASpace is the pin. Escaping a single [ or ]
//     inside a placeholder is the wider convention question, and it stays in
//     docs/backlog/escaping-across-run-boundaries.md.
//
// # A horizontal rule prints [rule] and not ---
//
// The footnote separator is already ---, and one line meaning two things is a
// line neither of them can be read from. A footnote referenced twice is one
// entry under that rule, numbered as the document numbers it, and a footnote's
// own paragraphs are joined with spaces rather than dropped.
// TestASecondReferenceToOneFootnotePrintsTheSameNumber and
// TestAFootnoteWithTwoParagraphsIsNotGluedTogether are the pins.
//
// # A comment range that names no position is a warning, not a marker
//
// The gating is docs.Places, the one rule the comments command reports a range
// from, so the two commands cannot drift into printing a position one of them
// would refuse. What this package keeps is the wording: two warnings naming
// which half of the rule the range failed, because a reader needs to know
// whether the range was inverted or simply outside the text.
//
// A range that does not end after it starts is the worse half. Armed, it puts
// its own close before its own open, and at the end of the last run the
// closes-only drain emits the close and leaves the open behind. Either way the
// text carries half a pair, which is exactly what the escaping exists to make
// impossible, and an inverted pair crosses every marker it passes on the way.
// The test is start >= end rather than equality: the indexes come out of the
// Docs comments key unchecked, and that decoder is loose on purpose because the
// shape is measured rather than documented, so it can hand over either.
// TestARangeEndingBeforeItStartsIsAWarningAndIsNotPrinted and
// TestARangeCoveringNoTextIsAWarningAndIsNotPrinted are the pins, with
// TestARangeOutsideTheTextIsAWarningAndIsNotPrinted and
// TestARangeNamingAnAbsentTabIsAWarning over the other two shapes.
//
// The walk arms its markers from the tab's form of the rule, docs.Tab.Places,
// because the markers go into the tab being walked. Two tabs sharing an id,
// which happens when a tab carries no tabId and takes the default t.0, would
// otherwise arm one tab on the other one's indexes and leave an opening marker
// its own text never closes. On a document whose tab ids are unique the two
// forms answer the same. TestATabSharingAnIDIsNotArmedOnTheOtherTabsText is the
// pin.
//
// # Lists and tables
//
// A list item comes back as a "- " item with two spaces of indent per nesting
// level. A numbered list reads back as a bulleted one: telling the two apart
// needs the document's lists map, which this read does not carry, so the
// numbers an author typed are not in the text at all. TestGolden is the pin on
// the shape, over a fixture holding a nested item.
//
// A table is a pipe table, the rows in reading order with a separator after the
// first, and the widths are not padded: the reader is a language model and the
// alignment would be bytes nobody reads. A cell holding two paragraphs, or a
// table of its own, is flattened with spaces, because a newline inside a cell
// would end the row.
//
// A pipe the author typed inside a cell is escaped. Unescaped it is a column
// separator, so a two-cell row holding "A | B" reads back as three columns under
// a two-column separator: an ordinary cell value would change the table's
// shape. The escaping happens on the row, after the marker escaping has already
// doubled the author's backslashes, so the parity a reader uses on a marker
// holds on a pipe too. It is done on the row rather than in the cell so a
// nested table flattened into a cell is escaped once, by the outer row.
// TestAPipeInsideACellDoesNotAddAColumn is the pin, and
// TestARangeInsideATableCellIsMarked says a comment anchored inside a cell is
// still marked.
//
// # --structure is the other view, and neither is the other's summary
//
// Structure is the same document as a tree with the character indexes on it,
// and its JSON tags are docs's own, so there is one name per field in the whole
// binary. A milestone was expected to place a proposal from it and does not: a
// proposal names text, internal/propose reads the document itself and finds the
// words, and the index never leaves the run that computed it. The view is kept
// because an index view is what a later milestone would need if one ever placed
// a change without quoting it, and because a caller reading a chip's email
// address or a link's target reads it here. TestStructureRoundTripsWithItsIndexes
// and TestStructureCarriesATablesStartIndex are the pins.
package view
