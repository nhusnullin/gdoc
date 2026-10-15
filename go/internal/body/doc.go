// Package body is the note's markdown walked with goldmark into the house
// style's own paragraphs.
//
// Render takes the house style, the note's markdown, the directory a relative
// picture is resolved against and whether headings take their numbers, and
// hands back the blocks internal/render writes into word/document.xml, the
// pictures and links they name, the counts, the files the walk read, and
// everything the walk would not render. The golden documents under testdata are
// the specification of what a note becomes:
// TestTheSixDocumentsRenderToTheirGoldens is the pin over all of them.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # goldmark parses, and the OOXML is written here
//
// goldmark is used only as a markdown parser, never as a docx writer. A
// general-purpose docx writer emits its own styles, which the house style does
// not define, and a reader that meets a dangling w:pStyle discards the whole
// w:pPr around it: every bullet and every number then silently disappears. The
// extension set is the one the previous generator asked pandoc for, pipe
// tables, ==mark==, strikeout and task lists, plus footnotes and the smart
// punctuation pandoc's markdown format enables by default. parse in body.go is
// where that set is stated, with the typographer writing characters rather than
// HTML entities, because a "&ldquo;" in a w:t is both wrong and a different
// width. TestSmartQuotesAreTheCharacters and
// TestSmartPunctuationIsCharactersNotEntities are the pins.
//
// ==mark== is mark.go, written here rather than pulled in, and it carries seven
// marks rather than one highlight (DECISIONS.md, 2026-10-14). ==words== is
// yellow, a colour emoji at the start of the highlight picks one of the other
// five, and [words]{.pencil} is the one Pandoc span this package reads, text
// colour rather than shading. house.yaml states the seven with their syntax,
// kind and Doc colour, so no emoji and no hex is a constant here; the one
// string written out is the attribute ".pencil", because the parsers are built
// before any house file is read, and TestThePencilSyntaxIsTheHouseStyles pins
// it to the pencil mark's own syntax. What is carried is the mark on the run;
// what is refused by line is a nested mark, an empty span, a highlight whose
// only content is the colour emoji that opened it, which marks no words the way
// an empty span marks none and counted a colour on an envelope the document does
// not carry, a span carrying any
// other attribute, a span that does not open and close on one line, a mark
// inside a heading, for the reason a footnote mark in a heading is refused, and
// a mark inside a picture's alt text, which becomes the caption: the caption is
// built with no palette at all, so the mark was dropped there and the colour
// emoji that opened it reached the document as words. An
// emoji anywhere but the start of a highlight, and a bracket not followed by
// ]{, are the author's own words. A bracket inside a code span is neither: the
// parser steps over a code span the way the line scan does, so the two agree
// on what opens a span.
// TestAHighlightOpeningWithAColourEmojiCarriesItsMark,
// TestAnEmojiElsewhereInAHighlightIsText, TestAPencilSpanIsARunCarryingItsMark,
// TestALinkIsStillALink, TestABracketNotFollowedByAttributesIsText,
// TestASpanWithAnotherAttributeIsRefusedByLineNamingPencil,
// TestANestedMarkIsRefusedByLine, TestAnEmptySpanIsRefusedByLine,
// TestAHighlightOfOnlyItsColourEmojiIsRefusedByLine,
// TestAHighlightWithWordsAfterItsColourIsNotRefused,
// TestASpanCrossingAParagraphIsRefusedByLine,
// TestASpanWhoseWordsHoldACodeSpanWithABracketIsStillASpan,
// TestASpanInsideACodeBlockIsNotASpan and
// TestAMarkInAHeadingIsRefusedByLine and
// TestAMarkInAPicturesAltTextIsRefusedByLine are the pins, with
// TestTwoMarksSideBySideStayTwoRuns over merge and
// TestTheDebtsAreCountedAndTheOthersAreNot over the three counted on the
// envelope.
//
// A mark reaches Word twice over. The run names the character style
// internal/render writes, so Word lists the marks and a reader finds every
// yellow with one search, and the run states the colour itself, shading with
// the exact hex for a highlight and the run's own colour for the pencil.
// Twice over because Google Docs has no character styles and keeps only what
// the run says: after import a mark is direct formatting, and direct
// formatting is all export can read back. None of the six is one of Word's
// sixteen named highlights, which carry none of these hexes.
// TestAMarkedRunCarriesItsStyleAndItsShading is the pin, with
// internal/render's TestTheSevenCharacterStylesAreWritten over the styles and
// TestTheCoverPlaceholdersAreNotMarks over the one yellow that is not a mark.
//
// Every size, colour, indent and alignment comes from house.Config. What is a
// constant in this package is a value the house file does not state, and each
// one carries the reason it is what it is. Nothing is concatenated into XML:
// every element is built with etree and serialised, so the library escapes
// every w:t. TestAnAmpersandIsEscapedOnce is the pin.
//
// A properties element is a sequence, and Word reads one out of order as a
// document to repair. This package writes three of them, a quote's w:pPr, a
// table's w:tblPr and a run's w:rPr, where the character style opens the
// sequence and the shading and the vertical alignment close it. Each writer
// states the sequence it writes as a literal list of tags:
// TestAQuotesPropertiesAreInSchemaOrder,
// TestATablesPropertiesAreInSchemaOrder and
// TestARunsPropertiesAreInSchemaOrder are the pins, beside internal/render's
// own two for the parts that package writes. A run names the character style
// internal/render writes, which is how a footnote reference and a mark reach
// Word's own style list: TestARunMayNameItsCharacterStyle is the pin.
//
// # One list item takes at most one marker, whatever it holds
//
// The marker goes on the first paragraph the item actually emits, and every
// paragraph after it takes the item's indent and no marker. goldmark gives a
// loose item one paragraph per block, so numbering every one of them turned a
// two-paragraph item into two items and the author's "2." printed as "3.". A
// block quote inside an item used to number itself the same way. The hanging
// indent goes with the marker for the same reason: written on a paragraph with
// no number to fill it, it starts that paragraph's first line in the number's
// own column. TestALooseListItemIsOneItem,
// TestALooseContinuationKeepsTheIndentAndNotTheHang and
// TestAnItemTakesNoMoreThanOneMarkerWhateverItHolds are the pins.
//
// # The marker is pending on the renderer, and that is not a detail
//
// Nothing the walker can read off a child says which child emits the item's
// first paragraph. A block quote is a container whose paragraphs come back
// through block, so an item that is nothing but a quote has no *ast.Paragraph
// child at all, and marking only those left it with no number anywhere in it.
// Handing the marker to the item's first child instead breaks the mirror of
// that, an item opening with a fenced code block, which renders nothing and
// would spend the marker on a paragraph nobody sees. So itemBlocks arms
// pendingMark, paragraphBlock spends it, and itemBlocks puts the outer item's
// back on the way out so a nested list takes its own.
// TestANestedListTakesItsOwnMarkersAndLeavesTheOuterItemsAlone and
// TestAnOuterItemWhoseWordsFollowItsSubListStillTakesItsMarker are the pins.
//
// # At most, because an item can hold nothing that carries a marker
//
// An item that is only a table, only a code block, only a block of HTML or
// empty emits no list paragraph, so pendingMark is still armed when itemBlocks
// returns and the run warns. What it costs depends on the list, so the sentence
// does too. A number is a count Word carries on, so the items after it print
// one lower and the author's "3." reads as "2.", while a bullet is not a count,
// so a bulleted item loses only its own bullet and its indent and nothing after
// it moves. Telling an author to check numbering that is not wrong is the
// cry-wolf warning this tool avoids everywhere else. A table is the case that
// used to reach this in silence: it renders, at body width rather than inside
// the item, so the document looks deliberate and only the list is wrong.
// TestAnItemThatSpendsNoMarkerSaysSo is the pin, and it asks both lists.
//
// The line that warning names comes from inside the item. A goldmark ListItem
// carries no source position: its Offset is a column inside the line, and lines
// are appended to leaf blocks only. So itemLine reads the first descendant that
// has one, a fence being read one line above its own first line of code the way
// the code block's own warning reads it, and falls back to the nearest sibling
// item for an item holding nothing positioned at all. Asking the item resolved
// to line 1, which in a note is the front matter's own delimiter: the wrong end
// of the file to send somebody to. TestAnEmptyFenceNamesARealLine is the pin.
//
// # What the walker will not render is a warning naming the line, never a
// silent drop
//
// Fenced and indented code blocks, blocks of HTML, inline HTML, and the list
// marker an item holding none of those can carry. Code blocks stay out of the
// house style, which is Nail's decision, and a note carrying one has to say so
// on the envelope. A picture inside a list item, a block quote or a table cell
// is the same answer: the house style puts a figure on a centred line of its
// own, which it cannot be there. All three used to collect the picture and
// throw it away, so a bullet naming one published with no picture, no warning
// and an image count of nought, and inline HTML in a cell or a quote was
// dropped the same way. TestACodeBlockWarnsAndIsLeftOut,
// TestInlineHTMLWarnsAndIsLeftOut, TestInlineHTMLInATableCellOrAQuoteWarnsToo
// and TestAPictureTheHouseStyleCannotPlaceIsNamed are the pins.
//
// A footnote is no longer one of them: it is set in the document, and what the
// house style cannot set inside one is the warning instead. The section below
// on footnotes holds the rules and its pins.
//
// A heading that names nothing but a picture is not one of those: it emits no
// heading at all, takes no number and no contents entry, because it is a
// figure. TestAHeadingOfNothingButAPictureIsAFigure is the pin.
//
// # A picture at an http address is refused, not warned about
//
// A document built from a link is one that breaks when the link expires, and
// nothing here downloads. A relative path is resolved against the note's own
// directory and a data: URI is decoded, because those bytes arrived with the
// markdown. TestARemoteImageIsRefusedRatherThanDownloaded,
// TestARemoteImageIsRefusedNamingTheLine,
// TestARelativePathIsReadFromTheNotesDirectory and
// TestADataURIMustDeclareItselfAsABase64Image are the pins, with
// TestAPictureThatIsNotThereNamesTheLineAndTheFile and
// TestAMissingImageIsRefusedNamingTheFile over the file that is not there.
//
// # Sources names every picture the note names, placed or not
//
// A picture inside a list item, a block quote or a table cell is warned about
// and left out, and it is still a file the run must not write over. Recording
// only the embedded ones left that case worse than the one the check was
// written for: the bytes were in no word/media/ either, so the picture was
// simply gone. imagePath is the one resolver both halves use, and it is empty
// for a data: URI and for a link, because neither is a file. The caller is what
// knows where it is about to write: cmd/gdoc asks again after the walk and
// before anything is written, and TestBuildRefusesAnOutThatNamesAPicture there
// is the pin, over a picture the walk embedded and a picture it only warned
// about.
//
// # A footnote is set in the document, and one that does not line up is refused
//
// "[^1]" in the prose and "[^1]: the text" on a line of its own, which is what
// the footnote extension parses and what internal/view already prints: a marker
// in the prose and the definitions after a "---" line. A reference becomes a
// run carrying its number and internal/render writes the notes into
// word/footnotes.xml, numbered in the order a reader meets the mark, which is
// Word's own order whatever order the definitions were written in. Nothing is
// dropped and nothing is warned about for a note that lines up (DECISIONS.md,
// 2026-10-14). TestAFootnoteReferenceCarriesItsNumber,
// TestAFootnoteBecomesAFootnoteTextParagraph and
// TestFootnotesAreNumberedInReferenceOrderNotDefinitionOrder are the pins, with
// TestAFootnoteRunSurvivesMerge over the run, whose text is empty and which
// merge must keep, and TestTheBodyCarriesAFootnoteReferenceRun over the
// w:footnoteReference the paragraph gets.
//
// The number is the mark's position in that order and never
// east.FootnoteLink.Index, which is the definition's: goldmark gives both
// marks of a label written twice the same Index, and each mark is its own
// note. So the number cannot be read off a node and is counted in one pre-walk
// before the blocks, the way the heading anchors are.
//
// One label referenced twice is two notes carrying one text, because Word and
// Docs hold one note per mark and there is no second shape to write.
// TestAFootnoteReferencedTwiceIsTwoNotesWithOneText is the pin.
//
// A reference with no definition, a definition nothing references, a label
// defined twice, and a footnote mark inside a heading are each refused by line.
// The first three are read off the source lines, beside the markers check,
// because goldmark is silent in all three shapes: its parser leaves "[^x]" as
// prose when no definition follows, its transformer removes a definition nobody
// references, and it keeps the first definition of a label and removes the
// rest, so the tree says nothing happened. A warning was argued and rejected:
// it let a published document look finished with a citation missing from it.
// The heading case is Word's, which copies the mark into the contents field.
// TestAReferenceWithNoDefinitionIsRefusedByLine,
// TestADefinitionNobodyReferencesIsRefusedByLine,
// TestALabelDefinedTwiceIsRefusedNamingBothLines and
// TestAFootnoteMarkInsideAHeadingIsRefusedByLine are the pins.
//
// The heading refusal is asked of the tree as well, in headingBlock, because
// the line scan sees an ATX heading and goldmark builds a heading from a setext
// line, from inside a block quote and from inside a list item too.
// TestAFootnoteMarkInAHeadingTheLineScanCannotSeeIsRefused is the pin.
//
// The scan skips a code block and an inline code span, because nothing inside
// either is markdown: goldmark reads no mark in a span, a code block is left
// out of the document whole, and a note explaining a regular expression writes
// "[^abc]" in one. Read as a mark, that refused a note nothing was wrong with.
// A code block is a fence or four spaces, a fenced block shuts only on its own
// character in a run at least as long as its opener and carrying nothing else,
// and the whole rule lives in scan.go because the footnote scan and the span
// scan read the same lines. An indented block opens wherever a new block may:
// after a blank line, at the start of the file, and after a line that closed a
// block of its own, which is a heading, a horizontal rule, the underline of a
// setext heading or a fence that shut. The one thing it cannot do is interrupt
// a paragraph, and a scan that asked only for the blank line refused the line
// after a heading and the line after a closed fence.
//
// Both are measured from the content indent of the block they sit in and not
// from the margin: a footnote definition carries its lines at four and a list
// item at its marker's width, so a fence inside one is indented four or more
// and four spaces inside one is that block's own prose. Measured from the
// margin, a code sample in a nested list item was read as prose and the mark
// in it refused a note nothing was wrong with, and the fence of a code block
// inside a definition was never seen at all. A list item and a definition each
// run on through their blank and indented lines and end at the first line of
// prose at the margin, which is goldmark's own rule for continuing one.
// TestAFootnoteInsideACodeBlockIsNotAFootnote,
// TestAFootnoteInsideAnIndentedCodeBlockIsNotAFootnote,
// TestACodeBlockShutsOnItsOwnFenceAndNotAnother,
// TestACodeBlockInsideABlockThatIndentsItsContentIsStillCode and
// TestAMarkInAListItemsOwnLaterParagraphIsStillProse are the pins.
//
// A footnote mark inside a table cell is refused as well. The spec carries
// none, and a cell's paragraph has no footnotes part to write into, so the note
// would be numbered in the body and written nowhere.
// TestAFootnoteReferenceInATableCellIsRefusedByLine is the pin.
//
// A footnote mark inside a picture's alt text is refused for the same reason,
// and that one is asked of the tree: the alt text sits on a line of prose, so
// the line scan reads the mark as the document's own words and the definition
// it names answers for, while the walk that builds the caption drops it. The
// note was written under the page with nothing pointing at it, and the envelope
// counted a footnote the document does not have.
// TestAFootnoteMarkInsideAPicturesAltTextIsRefusedByLine is the pin.
//
// Inside a footnote, paragraphs with inline marks only. A list, a table, a
// picture, a code block or a heading there is warned about by line and left
// out, the way a picture the house style cannot place is: a note under the page
// is a sentence, and the house style has no geometry for a table in one. The
// line is the block's own, so a note written over several lines sends the
// author to the line they have to change. A picture is an inline and takes the
// line of the paragraph it sits in, and the definition's own line is the answer
// only for a block carrying no source position anywhere under it, which a
// horizontal rule is.
// TestWhatTheHouseStyleCannotSetInAFootnoteIsWarnedAndLeftOut is the pin, and
// it asks each of them, a picture in the note's second paragraph and a
// horizontal rule among them.
//
// A link inside a footnote is its words. A link is a relationship in
// word/_rels/document.xml.rels, and the footnotes part names no relationships
// at all, so the words stay and the link goes, named by line like the rest. A
// relationships part of its own is a later entry, for citations that need a
// live address. TestALinkInsideAFootnoteIsWarnedAndPrintedAsItsWords is the
// pin.
//
// A footnote mark inside a definition is the same answer: Word has one level of
// note and no second, and the numbers are the prose's reading order, so a mark
// there takes no number and is named rather than dropped in silence.
// TestAFootnoteInsideAFootnoteIsWarnedAndLeftOut is the pin. Because such a
// mark makes no note, it does not answer for a definition either: a definition
// marked only from inside another one is refused the way an unmarked one is.
// TestADefinitionMarkedOnlyFromInsideAnotherIsRefusedByLine is the pin.
//
// Because a heading inside a footnote is left out, the two walks that run
// before the blocks skip it as well: it is not a heading of this document, it
// carries no bookmark, and it does not set the depth every other heading
// numbers from. TestAHeadingInsideAFootnoteIsNoAnchor and
// TestAHeadingInsideAFootnoteSetsNoHeadingDepth are the pins.
//
// # A marker of gdoc's own is refused by line, before the note is parsed
//
// internal/view writes {+, {-, [[c: and [s: into the text `read` prints and
// `export` writes to a file, and they say a thing about a document: these words
// are a pending insertion, this range carries a comment. In a document they say
// nothing, so a note still carrying one publishes the brackets and the id as
// prose. The check is asked of the source line rather than of any block,
// because the export writes a marker into the middle of a sentence and the
// sentence is a paragraph like any other, and it is asked of internal/markers,
// which holds the literals and the escaping parity for all three routes out of
// the hub. An escaped marker is the author's own text and is left alone, so a
// note written about the markers themselves still publishes.
// TestBuildRefusesAMarkerByLine and TestBuildLeavesAnEscapedMarkerAlone are the
// pins.
//
// It is a refusal rather than the warning a code block gets. A code block
// leaves one block out of the document; a marker puts words into it that the
// author never wrote and that nothing downstream would name.
//
// # An email autolink carries the mailto: scheme, and the label does not
//
// goldmark puts the scheme on in its HTML renderer and never in AutoLink.URL,
// so the address arrives at inline.go bare. Written into a relationship as it
// arrives it is a relative URI reference, which Word resolves against the
// document's own location: the link opens nothing, and a contact address is
// ordinary in a policy. The run's text stays the bare address, which is what the
// author typed. TestAnEmailAutolinkCarriesTheMailtoScheme and
// TestAWebAutolinkKeepsItsOwnScheme are the pins.
//
// # A link to another note is its words and no link
//
// A destination with no scheme, no host and no leading "#" is a path in the
// hub: every note ends with a Related section written that way. Written as
// a relationship, Word resolves it against the document's own location, and in
// Drive that file does not exist, so the link opens nothing and nothing says
// so. Related is hub navigation, and a colleague reading in Drive needs the
// note's name, which they can search the hub for, not a jump. So the run keeps
// the words and carries no link, no relationship and no warning, because the
// note is correct and its author has nothing to fix. A folder is the same
// path with a slash. Nail's decision of 2026-09-19; rewriting the link to the
// other note's Drive document was rejected because it makes publish order
// matter. TestARelativeLinkIsItsWordsAndNoHyperlink is the pin, and
// TestALinkWithASchemeOrAnAnchorIsStillALink holds the other side of it.
//
// # An anchor link is a jump to a bookmark on the heading it names
//
// A destination opening with "#" is a place in this document, not an address.
// Written as a relationship, which is what every other link is, Word resolves
// it against the document's own location, so "[see below](#scope)" published
// as a link that opens nothing and said nothing about it. The OOXML form for
// a jump is w:hyperlink w:anchor with no relationship at all, and it lands on
// a w:bookmarkStart/w:bookmarkEnd pair, so the bookmarks come first. Only a
// heading some "#" link in the note names carries one. Google Docs keeps a
// bookmark on import and shows it as a flag on the heading, so a bookmark on
// every heading is a flag on every heading for nothing, and a later edit to
// the note goes out through a fresh publish, which writes the bookmarks from
// the note as it then stands. Nail's call of 2026-09-19. There is no way to
// keep a jump and hide the one flag it lands on.
//
// The name is goldmark's own auto heading id, which parser.WithAutoHeadingID
// already computes and keeps unique across the file, rewritten by bookmarkName
// into what Word takes: letters, digits and underscores, opening with a
// letter, at most 40 characters. The leading "h_" is what makes a heading
// called "1.1 Purpose" open with a letter; every character outside
// [A-Za-z0-9_] becomes an underscore, which is wider than the lower-case
// letters, digits and hyphens goldmark emits, on purpose, because the ids are
// the generator's and this package does not get to notice when it widens; and
// a name over the ceiling keeps its first 32 characters, so it reads in Word's
// bookmark list, and takes seven hex digits of the FNV-1a hash of the id as it
// arrived, so two long headings sharing a prefix keep two names.
// TestBookmarkNameIsWordSafe pins all four cases,
// TestOnlyALinkedHeadingCarriesABookmark pins the pair around the runs and
// its absence on a heading nothing names, and
// TestANoteWithNoAnchorLinkCarriesNoBookmark is the common case.
//
// A "#" naming no heading in the note is warned about by line and its words
// are printed as plain text, which is how a code block is already refused: a
// jump that lands nowhere is worse than no jump, and the author is the one who
// can fix it. The heading ids are collected before the blocks are walked, so a
// link to a heading further down resolves, and the pre-walk collects only the
// headings that will carry a bookmark: a figure-only heading emits no
// paragraph, so a link naming it is as dead as a link naming nothing, and so
// does a heading inside a footnote definition, which the footnote walk warns
// about and leaves out. A pre-walk that counted either would write a live
// w:anchor naming a bookmark nothing wrote, which is the one failure this is
// here to prevent, and it would do it silently.
// TestAnAnchorLinkIsAJumpAndNotARelationship,
// TestAnAnchorToNoHeadingWarnsAndPrintsPlainText,
// TestAFigureOnlyHeadingCarriesNoBookmark and
// TestAHeadingInsideAFootnoteIsNoAnchor are the pins, and
// TestALinkIsAHyperlinkWithARelationship is the https link, unchanged.
//
// The line the warning names is renderer.curLine, because addRuns is six
// callers deep from the node and has no line of its own. Every path that emits
// runs sets it where the node is in hand: headingBlock, paragraphBlock, the
// table branch of block, and quoteBlock, which handles a top-level paragraph
// itself rather than through paragraphBlock. A path that forgets names the
// block before it, and the author looks for the link there.
// TestADeadAnchorInsideABlockQuoteNamesItsOwnLine is the pin on the quote,
// which is the one that was forgotten. A table names the table's own line
// rather than the cell's, which is Nail's decision: a cell has no line of its
// own that the author would recognise.
// TestADeadAnchorInsideATableNamesTheTablesOwnLine is the pin on the table.
//
// One dead link is one sentence. addRuns runs once per run and merge splits a
// link's words at every mark boundary, so a destination inside bold text
// reached the envelope twice and the note read as holding two dead links
// where it held one. renderer.deadAnchors keeps the line and the destination
// of every sentence already raised, and two links on one line naming the same
// missing heading are one sentence for the same reason: the sentence holds a
// line and a destination and nothing else, so the second copy says nothing.
// Two lines are still two sentences, because they are two places to go and
// fix. TestOneDeadAnchorWarnsOnce,
// TestTwoDeadAnchorsOnOneLineWarnOncePerDestination and
// TestTheSameDeadAnchorOnTwoLinesWarnsTwice are the pins.
//
// # The body breathes: 12pt under a paragraph, 3pt between items, 12pt under
// the last
//
// A body paragraph carries house.yaml's body spacing, 0 above and 12 below,
// so two paragraphs in a row read as two and nobody adds a blank line by
// hand. Items of a list carry the house's list item gap between them, and the
// last paragraph an outermost list emits takes the body's space after
// instead, patched in endList once the whole list has been walked. A picture
// with a caption carries none, so it sits on its caption. A body table's
// cells carry 5pt on every side and single-spaced text. DECISIONS.md,
// 2026-10-14. TestABodyParagraphCarriesTwelvePointsBelow,
// TestTheLastItemOfAListEndsItLikeAParagraph,
// TestANestedListEndsOnItsInnermostLastItem,
// TestAListThatRenderedNothingLeavesTheBlockBeforeItAlone,
// TestAPictureSitsOnItsCaption, TestARuleAndAQuoteEndLikeAParagraph,
// TestACellHasFivePointsOnEverySide and TestACellParagraphIsSingleSpaced are
// the pins.
//
// # A numbered list starts at 1, and a list that opens elsewhere says so
//
// A w:num is where Word keeps a list's running count, so two numbered lists
// sharing one carried one count and the author's 1. and 2. printed as 3. and
// 4. The walker hands a fresh id to every numbered list that is not nested
// inside a numbered list, counts them on Result.NumberedLists, and render.Build
// writes that many w:num entries, all on the one numbered abstract list. A
// numbered list nested in a numbered item reuses its parent's id: an absent
// w:lvlRestart already restarts the inner level whenever the outer one moves,
// and a second id there would make one list two counts Word draws side by side.
// A numbered list nested in a bullet is not nested in a count at all, so it
// opens its own. TestASecondNumberedListStartsAgain,
// TestANestedNumberedListNamesItsParent,
// TestANumberedListUnderABulletOpensItsOwn and
// TestOneNumberedListDefinitionPerNumberedList are the pins.
//
// What stays wrong is the number a list opens on. Every level of the numbered
// abstract list states w:start 1, and every list's own w:num states
// w:startOverride 1 over it, which is what makes the restart the file's rather
// than one reading of the spec. Honouring an author's "5." is that same
// override carrying their number instead, which gdoc does not write, so the
// list opens at 1 and warnListNumbers names the line. The silence around a list
// that starts at 1 is deliberate, because the prose beside a numbered list
// cross-references the numbers the author wrote.
// TestANumberedListWhoseNumbersAreNotTheAuthorsSaysSo and
// TestAListThatStartsElsewhereStillSaysSo are the pins.
//
// # A heading that skips a level is numbered with a zero in it, and says so
//
// headingNumberer.prefix builds the number from every counter down to the
// heading's own level, so a "###" under a "#" reads "1.0.1-", and each heading
// under it inherits that zero. The number is the one the previous generator
// wrote and is left as it is: changing it is a decision for Nail, so prefix
// reports the zero as its second return and the walker names the line.
// TestASkippedHeadingLevelSaysSo is the pin, and TestPrefixesRunOnePerLevel is
// the numberer's own. testdata/docs/01-kitchen-sink.md carries this and both
// numbering cases above on purpose, which is how they were found.
//
// # Nothing here reaches the network and nothing here runs a program
//
// A walk is bytes in and elements out. cmd/gdoc opens no policy and no session
// for a build, because there is no wire to judge.
// TestNothingHereReachesTheNetwork is the pin: it reads every production file
// in this package and refuses net/http, os/exec and the gapi session package
// by name, beside internal/house's own copy of the rule and go/boundary's
// allowlists over the whole tree. The test reads the files as text, so naming
// one of the three in a comment here fails it, which is why this paragraph
// spells the third one out in words.
//
// A test in this package states its value as a literal, which is CLAUDE.md's
// invariant: a test that reads the constant it checks is a mirror, and it
// follows that constant wherever somebody moves it.
package body
