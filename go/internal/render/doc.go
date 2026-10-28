// Package render writes the docx itself: the thirteen parts and the logo, the
// cover, the tables, the header and footer, the contents field.
//
// Build takes the house style, the bytes of the logo, the note's cover fields,
// the body elements the markdown walker produced, the pictures it read and the
// footnotes it numbered, and hands back a Package of named parts a caller
// zips. Nothing here reads the Word master, and nothing here reaches the
// network. TestBuildWritesTheThirteenPartsAndTheLogo, TestEveryXMLPartParses
// and TestTheZipListsContentTypesFirst are the pins over what a build
// produces.
//
// The logo is a parameter and not something read off the style, because since
// M22 a house file may name it as one file in Drive rather than carry its
// bytes. internal/asset is the room that turns an id into bytes, and this one
// reaches nothing: the caller fetches the mark and hands it over. Build
// refuses an empty one rather than writing a document with no mark in its
// header: TestBuildRefusesALogoWithNoBytes.
//
// The house states the mark's own width, and Build reads the PNG header so its
// height follows the picture's own ratio. Bytes that are not a PNG are refused
// by name. TestTheLogoHeightFollowsThePicturesRatio,
// TestBuildRefusesALogoThatIsNotAPNG and
// TestTheFirstPageHeaderCarriesTheLogoWhereTheHouseAnchorsIt are the pins.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # Nothing is concatenated into XML
//
// Every element is built with etree and serialised, so the library escapes
// every w:t. A string template escapes nothing: one ampersand in a note's title
// breaks the part, and a value from the note or from the style file never
// reaches the XML as text somebody formatted.
// TestATitleCarryingXMLCharactersIsEscaped is the pin.
//
// etree is not namespace aware, so a tag carries its literal prefix, which is
// what a docx holds anyway.
//
// # A properties element is a sequence, and Word reads one out of order as a
// document to repair
//
// That holds for w:pPr, where para writes the children in schema order and it
// is easy to break by appending; for w:tblPr, where the schema is tblStyle,
// tblW, jc, tblCellSpacing, tblInd, tblBorders, shd, tblLayout, tblCellMar,
// tblLook; and for word/settings.xml, where CT_Settings puts evenAndOddHeaders
// a long way in front of updateFields. The front-matter tables used to write
// tblLayout in front of tblBorders, and neither the golden parts nor the drift
// gate reads child order, so nothing else would have named it. So each writer
// states the sequence it writes as a literal list of tags:
// TestTheFrontMatterTablesPropertiesAreInSchemaOrder and
// TestSettingsChildrenAreInSchemaOrder are the pins here, beside
// internal/body's TestAQuotesPropertiesAreInSchemaOrder,
// TestATablesPropertiesAreInSchemaOrder and
// TestARunsPropertiesAreInSchemaOrder for the parts that package writes.
//
// # word/footnotes.xml is always written
//
// The thirteenth part. It carries Word's two reserved notes, the rule above
// the notes at w:id -1 and the rule above a note continued onto the next page
// at 0, then one w:footnote per entry in Build's footnotes argument, which is
// body.Result.Footnotes: one per mark in the prose, numbered from 1 in the
// order a reader meets one.
//
// The part is in the package whether or not the document has a footnote,
// because the content type and the relationship are the shell's own and a
// relationship pointing at nothing is a document Word offers to repair. The
// two separators are written for the same reason. The relationship is rId8,
// so FirstMediaRelID moved from 8 to 9 and a body picture starts there.
// TestTheFootnotesPartCarriesTheTwoSeparators,
// TestEachNoteIsAFootnoteAtItsOwnNumber,
// TestTheFootnotesPartIsDeclaredAndRelated and
// TestAnImageIdThatCollidesWithTheFootnotesIsRefused are the pins.
//
// Footnote is this package's type and internal/body aliases it, the way Media
// already works, so the numbers that package hands out are the numbers this
// part is written from and the two cannot drift.
//
// The notes are set in FootnoteText, the tenth house style, 10pt and
// single-spaced with no space above or below, and the mark in the prose names
// FootnoteReference, the one character style this part of the milestone adds.
// Both ids and both names are Word's own, so a person opening the document
// finds the styles where they expect them.
// TestTheFootnoteTextStyleIsTheHousesTenth and
// TestAFootnoteReferenceIsASuperscriptCharacterStyle are the pins here, and
// internal/house's TestTheTenStylesAreThere and
// TestFootnoteTextIsTenPointSingleSpacedAndTight over the style file.
// internal/drift reads nine named styles and not this one: Docs has no
// footnote_text to compare against, so neither gate measures it.
//
// # A character style is how a run says what it is
//
// charStyleElement writes one w:type="character" style: an id, a name and the
// run properties it carries. Both parts of the footnotes-and-marks milestone
// use it. A footnote reference is the FootnoteReference style, superscript,
// and each mark is a style of its own carrying the colour the house file
// states, so Word lists them and a reader finds every yellow with one search.
// The run names the style and states the same colour itself, because Google
// Docs has no character styles and keeps only what the run says: after import
// a mark is direct formatting, and that is what export reads back.
// TestACharacterStyleIsWrittenWithItsRunProperties is the pin over the writer,
// TestTheSevenCharacterStylesAreWritten over the seven the palette gives,
// TestTheMarkStyleIDIsTheHouseKeyRaised over the one id both packages name a
// style by, and internal/body's TestARunMayNameItsCharacterStyle and
// TestAMarkedRunCarriesItsStyleAndItsShading over the w:rStyle that names it.
//
// The cover's own yellow placeholders are not marks. They are the template's,
// they mean "a person fills this in" rather than any of the seven, and they
// keep Word's named yellow and name no character style, so nothing counting
// or exporting marks reads a cover as one:
// TestTheCoverPlaceholdersAreNotMarks is the pin.
//
// # Every list level states w:start
//
// ECMA-376 17.9.26 reads an omitted start as zero, so a nested numbered level
// that carries none opens its sub-list at "0." and prints every item under it
// one lower than the author wrote. The Word master states it on all sixty-three
// of its own levels. Nothing else would have caught it: drift.Items has no
// numbering row, so neither gate reads this part.
// TestTheNumberedListStartsAtOneDecimalAtThirtySixPoints is the pin, and it
// asks every level rather than the first.
//
// This part defines one w:num for the bullets and one per numbered list in the
// body, each numbered one stating w:startOverride 1 on all nine of its levels.
// The id alone would rest on a reader keeping its count per instance rather
// than per abstract list, and pandoc and python-docx both write the override
// instead, so the restart is stated rather than assumed. The Word master takes
// a third route, one abstract list per numbered list, which needs no override
// and costs a definition per list.
// TestOneNumberedListDefinitionPerNumberedList and
// TestEveryNumberedListOverridesItsStart are the pins. What is still not
// honoured is an author's own opening number, which is this same override
// carrying their value: internal/body warns by line.
//
// # A header or footer line carries its own size on the paragraph mark
//
// An empty line's height is its paragraph mark's size. The style file states
// 9pt on the two lines under the running head and 12pt on the footer's blank
// line, which is what the master carries there, and the size and colour were
// parsed and never written, so all three fell back to the document's 11pt
// default and the running head block came out taller than the master's.
// regionMark is where that lives.
// TestTheRunningHeadsOwnLinesCarryTheirSizeOnTheParagraphMark and
// TestTheFootersBlankLineCarriesItsSizeAndColourOnTheParagraphMark are the
// pins. No drift item reads a paragraph mark, in either gate: the Docs API has
// no paragraph mark to read, which is why the row is not there.
//
// # A block's own look reaches the page
//
// A cover line's alignment, line spacing, italic and colour override the
// cover defaults; its runs inherit that colour unless they state their own,
// and a filled placeholder keeps the line's colour while losing the
// template's highlight. A footer paragraph takes its own alignment before
// its region's and draws its own rule. A label may keep with the block behind
// it. TestACoverLineCarriesItsOwnLook,
// TestARunLineTakesTheLinesColourWhereTheRunStatesNone,
// TestAFilledLineKeepsTheLinesColour,
// TestAFooterParagraphTakesItsAlignment,
// TestAFooterParagraphDrawsItsRule and
// TestALabelKeepsWithTheBlockBehindIt are the pins.
//
// Only the first page's header may carry the logo. A logo stated in any
// footer or in the default header is refused by name:
// TestALogoInAFooterIsRefusedByName and
// TestALogoInTheDefaultHeaderIsRefusedByName.
//
// # The note's own words reach the front matter, and one of them is a mark
//
// placeholder resolves the cover fields, and a run or a cell paragraph naming
// one prints the note's value with the template's yellow and red taken off it.
// A field the note did not fill keeps the template's own highlighted
// placeholder, which is what a person filling the cover in by hand looks for.
// TestTheNotesTitleVersionAndDateReplaceThePlaceholders and
// TestANoteWithNoTitleKeepsTheTemplatesHighlightedPlaceholder are the pins.
//
// Three rules sit on top of that, and each closes a document this package used
// to write.
//
//   - A cover line naming a with: field is left out when that field is empty.
//     The master offers the title twice, either side of an "or", for a person
//     to pick one. Printing both published a page one reading the title, then
//     "or", then the template's own highlighted "(Name of) Framework/Policy".
//     TestOneTitleLeavesNoOrAndNoSecondTitle and
//     TestAnAlternativeTitlePrintsBothLinesAndTheOr are the pins.
//   - The "Version: " label is a run, and only the number beside it is the
//     note's. A line-level placeholder replaces the whole paragraph, so the
//     word went with it. TestTheVersionLineKeepsItsLabel is the pin.
//   - A cell naming a classification is shaded only when the note declares that
//     class. The master was captured with Internal marked, so writing its fills
//     verbatim marked every document Internal whatever the note said, and a
//     wrong mark is worse than a blank cell. A class the style file does not
//     describe is refused rather than shaded as nothing.
//     TestOnlyTheDeclaredClassificationIsShaded and
//     TestAClassNothingDescribesIsRefused are the pins.
//
// A row the style file marks to repeat is written once per revision the note
// declares, and a note declaring none keeps the template's own rows rather than
// printing an empty table. TestARevisionRowIsWrittenPerRevision,
// TestANoteWithNoRevisionsKeepsTheTemplatesRows and
// TestTheVersionControlTableCarriesTheNotesOwnWords are the pins.
//
// # A page turns where the style file says it turns
//
// The master pushes the version control label onto page two with eight empty
// paragraphs under the date, and the contents heading onto its own page with
// three more. Blank lines push a page only while nothing above them moves, so
// a title one line taller reflowed the whole front matter. house.yaml states
// the two turns as page_break blocks instead, and this package writes each one
// as the empty paragraph carrying w:pageBreakBefore that the first body
// heading already uses. internal/prelude reads the same two blocks and sends
// insertPageBreak, so one layout still has one statement.
// TestTheFrontMatterBreaksBeforeVersionControlAndBeforeContents is the pin.
//
// # The contents list is a Word field, not a measured list
//
// The style file states the instruction and this package emits the field, so
// Word fills it in on a refresh and Drive imports it as a live contents list.
// That is why a publish uploads once and has no measuring pass: nothing here
// lays the document out, so no page number is known until something does.
// TestTheContentsIsALiveWordField is the pin.
//
// An entry in that list opens the heading it names only because internal/body
// writes one hidden bookmark into every heading paragraph. Drive's import
// mints a heading id from such a bookmark and from nothing else, and a
// contents entry is a link to a heading by that id, so a document whose
// headings carry none gets a list where every entry lands on the start of the
// tab. The field this package emits is the same either way, which is why the
// failure was silent for a month. internal/body's
// TestEveryHeadingCarriesAHiddenBookmark is the pin on the bookmark.
//
// # Points come in and the units go out here
//
// The style file stores every measurement in points, so twips, half-points,
// eighths of a point and EMU are computed in this room and in no other. One
// value in the file has one meaning, and a unit conversion lives in one place.
// TestPointsBecomeTwipsHalfPointsEighthsAndEMU is the pin.
//
// # A value the style file states wrongly is refused, and the first failure is
// the one reported
//
// An alignment the writer has no Word value for is refused naming it, rather
// than written as a guess Word would repair the document over. The builder
// keeps the first failure and carries on, so a style file with three bad values
// names the first one every time rather than a different one per run.
// TestAnAlignmentTheHouseDoesNotHaveIsRefusedByName and
// TestTheFirstFailureIsTheOneReported are the pins.
//
// # A relationship id is a document's plumbing, so a collision is refused
//
// The shell owns rId1 to rId8: the styles, the numbering, the settings, two
// headers, two footers and the footnotes. A body picture starts at
// FirstMediaRelID, and one that takes a shell id would give a caller a
// document whose picture is the styles part. Two pictures sharing one id is the same failure one step along,
// and a relationship that is both a link and a file is a caller that built one
// of the two wrongly. Each is refused before a part is written.
// TestAnImageIdThatCollidesWithTheShellIsRefused,
// TestTwoImagesSharingOneIdAreRefused and
// TestARelationshipThatIsBothALinkAndAFileIsRefused are the pins, and
// TestAnImageIsWrittenWithItsRelationshipAndItsContentType and
// TestALinkIsAnExternalRelationshipWithNoPart are the two shapes that do carry.
//
// # No network in the render path
//
// A build is a file in and a file out. Nothing here imports net/http,
// internal/gapi, guard or auth, and cmd/gdoc opens no policy and no session for
// it, because there is no wire to judge. go/boundary's import allowlist holds
// the net/http half for every room in the tree, and internal/house's
// TestNothingHereReachesTheNetwork and internal/body's canary state the whole
// rule for those two packages, the internal/gapi import included.
// TODO(test): no canary reads this package's own imports; the two above are the
// pattern to copy if one is wanted here.
//
// # The golden parts are the specification
//
// TestTheGoldenPartsAreWhatTheHouseStyleRendersTo compares every XML part but
// the document itself against a file under testdata, so a change to the style
// file or to a writer shows up as a diff somebody reads rather than as a
// document somebody opens. A golden is regenerated with -update on purpose,
// never to make a run pass. word/document.xml is out because it carries the
// note's own words, and the tests over it ask the front matter and the section
// properties directly instead.
//
// A test in this package states its value as a literal, which is CLAUDE.md's
// invariant: if got != "480", never if got != twips(cfg.X). A test that reads
// the constant it checks follows that constant wherever somebody moves it.
package render
