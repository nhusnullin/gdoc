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
// tables, ==mark==, strikeout and task lists, plus the smart punctuation
// pandoc's markdown format enables by default. parse in body.go is where that
// set is stated, with the typographer writing characters rather than HTML
// entities, because a "&ldquo;" in a w:t is both wrong and a different width.
// TestSmartQuotesAreTheCharacters and TestSmartPunctuationIsCharactersNotEntities
// are the pins.
//
// Every size, colour, indent and alignment comes from house.Config. What is a
// constant in this package is a value the house file does not state, and each
// one carries the reason it is what it is. Nothing is concatenated into XML:
// every element is built with etree and serialised, so the library escapes
// every w:t. TestAnAmpersandIsEscapedOnce is the pin.
//
// A properties element is a sequence, and Word reads one out of order as a
// document to repair. This package writes two of them, a quote's w:pPr and a
// table's w:tblPr, and each writer states the sequence it writes as a literal
// list of tags: TestAQuotesPropertiesAreInSchemaOrder and
// TestATablesPropertiesAreInSchemaOrder are the pins, beside internal/render's
// own two for the parts that package writes.
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
// Fenced and indented code blocks, blocks of HTML, inline HTML, footnotes, and
// the list marker an item holding none of those can carry. Code blocks stay out
// of the house style, which is Nail's decision, and a note carrying one has to
// say so on the envelope. A picture inside a list item, a block quote or a
// table cell is the same answer: the house style puts a figure on a centred
// line of its own, which it cannot be there. All three used to collect the
// picture and throw it away, so a bullet naming one published with no picture,
// no warning and an image count of nought, and inline HTML in a cell or a quote
// was dropped the same way. TestACodeBlockWarnsAndIsLeftOut,
// TestInlineHTMLWarnsAndIsLeftOut, TestInlineHTMLInATableCellOrAQuoteWarnsToo
// and TestAPictureTheHouseStyleCannotPlaceIsNamed are the pins.
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
// # The footnote extension is on so that a footnote can be refused
//
// With it off, "[^1]" and "[^1]: the text" are ordinary markdown text, so they
// published as prose with nothing on the envelope. That is not a hypothetical
// note: internal/view writes a document's footnotes in exactly that shape, a
// marker in the prose and the definitions after a "---" line, so a note pulled
// out of a Doc and built back into one carried the markers into the published
// document. goldmark collects every definition into one list at the end of the
// file whatever order they were written in, so each footnote is named by its
// own line rather than the list's. TestAFootnoteWarnsAndIsLeftOut and
// TestEveryFootnoteIsNamedByItsOwnLine are the pins.
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
// # An anchor link is a jump to a bookmark every heading with words carries
//
// A destination opening with "#" is a place in this document, not an address.
// Written as a relationship, which is what every other link is, Word resolves
// it against the document's own location, so "[see below](#scope)" published
// as a link that opens nothing and said nothing about it. The OOXML form for
// a jump is w:hyperlink w:anchor with no relationship at all, and it lands on
// a w:bookmarkStart/w:bookmarkEnd pair, so the bookmarks come first: every
// heading that emits a paragraph carries one, whether or not this note links
// to it, because a note is edited after it is published.
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
// TestBookmarkNameIsWordSafe pins all four cases and
// TestEveryHeadingCarriesABookmark pins the pair around the runs.
//
// A "#" naming no heading in the note is warned about by line and its words
// are printed as plain text, which is how a code block is already refused: a
// jump that lands nowhere is worse than no jump, and the author is the one who
// can fix it. The heading ids are collected before the blocks are walked, so a
// link to a heading further down resolves, and the pre-walk collects only the
// headings that will carry a bookmark: a figure-only heading emits no
// paragraph, so a link naming it is as dead as a link naming nothing.
// TestAnAnchorLinkIsAJumpAndNotARelationship,
// TestAnAnchorToNoHeadingWarnsAndPrintsPlainText and
// TestAFigureOnlyHeadingCarriesNoBookmark are the pins, and
// TestALinkIsAHyperlinkWithARelationship is the https link, unchanged.
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
// abstract list states w:start 1, and honouring an author's "5." would be a
// w:startOverride on that list's own w:num, which gdoc does not write, so the
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
