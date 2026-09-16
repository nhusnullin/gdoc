// Package render writes the docx itself: the twelve parts and the logo, the
// cover, the tables, the header and footer, the contents field.
//
// Build takes the house style, the note's cover fields, the body elements the
// markdown walker produced and the pictures it read, and hands back a Package
// of named parts a caller zips. Nothing here reads the Word master, and nothing
// here reaches the network. TestBuildWritesTheTwelvePartsAndTheLogo,
// TestEveryXMLPartParses and TestTheZipListsContentTypesFirst are the pins over
// what a build produces.
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
// internal/body's TestAQuotesPropertiesAreInSchemaOrder and
// TestATablesPropertiesAreInSchemaOrder for the parts that package writes.
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
// One consequence belongs to the walker rather than here, and internal/body
// says it: this part defines one w:num per list kind, and every level of the
// numbered one starts at 1, so a second top-level list carries on from the
// first and an author's own opening number is not honoured. That package warns.
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
// # The contents list is a Word field, not a measured list
//
// The style file states the instruction and this package emits the field, so
// Word fills it in on a refresh and Drive imports it as a live contents list.
// That is why a publish uploads once and has no measuring pass: nothing here
// lays the document out, so no page number is known until something does.
// TestTheContentsIsALiveWordField is the pin.
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
// The shell owns rId1 to rId7: the styles, the numbering, the settings, two
// headers and two footers. A body picture starts at FirstMediaRelID, and one
// that takes a shell id would give a caller a document whose picture is the
// styles part. Two pictures sharing one id is the same failure one step along,
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
