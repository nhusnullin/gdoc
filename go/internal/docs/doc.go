// Package docs is the Docs read: the tabs, the document tree, the suggestion
// ids and the comment ranges.
//
// It is a pure reader. Fetch takes a Reader, which is the one method it needs
// of a session, and Parse takes bytes, so every rule here is testable on a
// fixture and the fixtures under testdata/ are the specification of what
// Google actually returns. Nothing in this package judges: it reports what the
// answer held and leaves every verdict to the skill reading the JSON.
//
// This comment holds why the decoder decodes the way it does, with the test
// that pins each rule. What the rules are is in the code beside them.
//
// # One Docs read, three views
//
// URL is documents.get with includeTabsContent=true,
// suggestionsViewMode=SUGGESTIONS_INLINE and
// commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED, and those three parameters
// carry three different facts in one call: the structure with every tab in it,
// which text is pending rather than written, and where the comment threads are
// anchored. read, suggestions and the range half of comments all stand on that
// one tree, so a command makes one Docs read and there is one shape to test.
//
// includeTabsContent is not optional. Without it the answer covers the first
// tab and says nothing about the rest, and commentsViewMode requires it.
// suggestionsViewMode is the only view that carries suggestion ids at all:
// Drive's export renders a suggested document as though nothing had been
// suggested. TestURLCarriesTheThreeParametersAndNothingElse and
// TestFetchSendsTheThreeParametersAndParsesTheAnswer are the pins, with
// TestURLIsACallTheGuardCarries asking the guard the same question.
//
// # Drive is the source of threads, Docs is the source of ranges
//
// comments.list carries the replies, the authors, resolved and modifiedTime;
// this read carries the character range, keyed by the same Drive comment id, so
// the comments command joins the two without a second lookup. A thread this
// read gave no usable range comes back in Unplaced, which the command reports
// as a warning. An unplaced comment is never a failed listing: the thread is
// still there, with its quoted text, and what is missing is only its position.
//
// The shape of the Docs comments key was measured on 2026-09-06, against a real
// document, because the reference does not describe what commentsViewMode adds.
// Each entry carries commentId and an anchorId, and the range sits in the tab
// under documentTab.commentAnchors[anchorId].ranges. anchoredRange reads that
// first, and the three shapes the decoder guessed at before the measurement
// stay behind it as fallbacks, because dropping them would be trusting one
// document to describe every document. testdata/anchors.json is the measured
// shape with placeholder text; the recording it was modelled on holds a real
// document and is gitignored. TestCommentRangesComeFromTheTabsCommentAnchors is
// the pin on the measured route, TestCommentRangeShapesTheDecoderAccepts on the
// fallbacks, and TestACommentInAMultiTabDocumentNeedsItsTab and
// TestACommentWithNoIDIsReportedByPosition on what is reported unplaced.
//
// # Places is one rule, with two ways in
//
// Document.Places is what comments reports a range from and what internal/view
// arms its markers from, so the two commands cannot drift into printing a
// position one of them would refuse. Tab.Places is the same rule asked of one
// tab, and a walker asks the tab form, because the markers go into the tab
// being walked: two tabs sharing an id, which the decoder can produce because a
// tab carrying no tabId takes the default t.0, would otherwise arm one tab on
// the other one's indexes and leave an opening marker its own text never
// closes. On a document whose tab ids are unique the two forms answer the same.
//
// Four shapes name no position, and the rule refuses all four: a range that
// does not end after it starts, one naming a tab the document does not have,
// one whose ends fall outside that tab's text, and one naming a segment, which
// is a header, a footer or a footnote rather than the body. The indexes come
// out of the comments key unchecked, and that decoder is loose on purpose
// because the shape is measured rather than documented, so it can hand over any
// of them. An inverted pair is the worst, because it crosses every marker it
// passes on the way. TestPlacesRefusesEveryRangeThatNamesNoPosition and
// TestTabPlacesAnswersForItsOwnTabWhenTwoTabsShareAnID are the pins.
//
// # A paragraph element is a run, and the default arm reports
//
// ParagraphElement is a union the reference documents as eleven members. run()
// decoded four of them, a text run, an inline object, a footnote reference and
// an equation, and everything else returned false and vanished at decode. Seven
// members went that way: the three smart chips, person, richLink and
// dateElement, plus autoText, pageBreak, columnBreak and horizontalRule.
//
// That was a live defect in read rather than a gap in a survey. An element that
// vanishes reaches neither view's placeholders nor its warnings, unlike an
// inline object gdoc cannot classify, which at least prints [object]. So every
// review session for a milestone read documents with holes in them and was told
// nothing: a policy naming its owner through a person chip read as a policy
// naming nobody.
//
// The fix is wider than the seven names. An element this decoder cannot name is
// still a run, carrying KindUnknown and the member name in Detail.Member, so it
// prints as a placeholder naming itself and warns. rawParaElement decodes twice,
// once into the named members and once into a bare map, because a second pass is
// the only way encoding/json answers "what else was in here". A decoder that
// drops what it does not recognise makes every reader downstream confidently
// wrong, and that is the class of defect, not the seven.
// TestAnElementTheWalkCannotNameIsReportedRatherThanDropped is the pin, with
// TestAnElementWithNoMemberIsStillARun beside it.
//
// Every member carries its own suggestedInsertionIds and suggestedDeletionIds,
// and the unknown arm reads them off whatever the element did name. All eleven
// documented members carry the two lists and the twelfth will. A run that kept
// the name and dropped the ids would be a pending change read prints with no
// markers and the survey counts in neither number, which is the same defect one
// field in. A member whose value is not an object says nothing about being
// suggested and does not fail the read.
// TestAnElementTheWalkCannotNameStillCarriesItsSuggestionIDs and
// TestAnUnnamedMemberThatIsNotAnObjectStillDecodes are the pins, with
// TestRunsCarryTheirSuggestionIDs over the text case.
//
// Carrying the ids is not the same as listing them. internal/suggestions walks
// text runs alone, so a suggestion carried only by a chip, a break, a rule or an
// auto text is printed by read and listed by nothing; restyle's survey counts
// those ids itself so it cannot answer "nothing to protect" over one. The
// listing half is docs/backlog/suggestions-on-elements-are-not-listed.md.
//
// # The seven elements are a fixture built from the reference
//
// The Docs API reference documents every field of all seven, so
// testdata/elements.json is written from it with placeholder text throughout. A
// fixture built from a reference is a hypothesis until a real document agrees
// with it, and that live check is outstanding: MEASURED.md's "Not measured yet"
// holds the recheck, and a disagreement is recorded there with the difference
// and the fixture corrected, never a test loosened. The decision is
// DECISIONS.md, 2026-09-09. TestTheDroppedElementsFixtureNamesAllSeven and
// TestTheSevenElementsDecodeIntoRuns are the pins.
//
// # A link is a fact about a run, and it is read from the reference
//
// A text run's textStyle.link says where that text points, and Run.Link carries
// it as one of four targets: a URL outside the document, a heading id, a
// bookmark id, or a tab. The reference documents heading and bookmark as
// objects carrying an id and the tab the target is in, with headingId and
// bookmarkId as the older flat spelling, so both forms are read into the same
// two fields and a tab id beside either of them says which tab that target is
// in. A run pointing nowhere carries no Link at all: an empty one would tell a
// reader this text points somewhere and not where.
//
// It is read on a text run and on nothing else. The three chips carry their
// own target in Detail already, and a picture that is a link is printed as a
// placeholder either way, so a link there would be a field with no reader.
// testdata/links.json is written from the reference, the way elements.json is,
// and stays a hypothesis until a live read agrees with it.
// TestARunCarriesItsLinkTarget and TestALinkInItsNestedFormIsTheSameFact are
// the pins.
//
// # A bullet names its list, and the tab's lists name the glyph
//
// Bullet carried the nesting level alone, which is all a reader printing "- "
// needs. Numbering needs two more facts: which list the item is in, so two
// adjacent lists are not counted as one, and whether that level of that list is
// numbered at all. The second sits in the tab's lists map, under the id the
// bullet names, at the item's own nesting level, as a glyphType.
//
// The map is read per tab, not per document. The id is the tab's own, two tabs
// may each name kix.list0 and mean two different lists, and a document-wide map
// would number one tab's bullets from the other tab's glyphs. Glyph is the
// answer's own glyphType, passed through rather than translated, and Ordered is
// the one question asked of it: GLYPH_TYPE_UNSPECIFIED is Docs saying this
// level is not numbered, and NONE is Docs saying this level draws an empty
// glyph, which is a list showing no marker rather than a numbered one.
// TestAGlyphOfNONEIsNotANumberedList is the pin on the second, which the reader
// would otherwise count 1., 2., 3. and export into the hub that way.
// A list the tab does not hold, or a level the list does
// not describe, leaves both empty rather than failing the read, and the reader
// prints the bullet it printed before. testdata/lists.json is written from the
// reference, and TestABulletCarriesItsListAndGlyph is the pin, with
// TestAParagraphCarriesItsHeadingID over the heading id a link points at.
//
// # The contents element is a block, because it is a boundary
//
// blocks() dropped tableOfContents with the section breaks, as an element
// carrying no text gdoc reads. It carries the text of every heading in the
// document, and more than that it is where the house prelude ends: a document
// publish made opens with the cover, three tables and this list, and export
// strips to the end of it. A dropped element is a boundary nothing downstream
// can name.
//
// So Block gains a third member, TOC, holding its entries walked as an ordinary
// body: each is a paragraph linking to the heading it names. plainText walks
// into it for the same reason. Nothing prints it yet, because internal/view
// walks Paragraph and Table and steps over anything else, which is why the five
// goldens did not move when this landed. testdata/toc.json is written from the
// reference, and TestATableOfContentsIsABlock is the pin.
//
// # A floating object is named by its paragraph and held by its tab
//
// A picture that floats is not in the text at all: the paragraph it is anchored
// to names it in positionedObjectIds, and the object sits in the tab's
// positionedObjects map. Both halves are read, Paragraph.Positioned and
// Tab.Positioned, because two paragraphs can name one object and the answer
// keys them in the tab. The kind is read the way an inline object's is, off the
// same embeddedObject, so a floating picture and an inline one answer the same
// question the same way.
//
// The positioning itself, the layout and the two offsets, is not read: gdoc
// lays nothing out, and a reader is told the picture floats rather than where
// it floats to. Before this the object was in no part of the tree and read said
// nothing at all about it. testdata/positioned.json is written from the
// reference, and TestAPositionedObjectIsCarriedOnItsParagraph is the pin.
//
// An inline object carries its own id too, in Detail.ID, which is the field a
// chip's id already uses. It is the run's one identity: a docx export of the
// same document carries the picture's bytes with nothing joining them to a
// position, so what pairs the two is the order the objects stand in, and what
// names one of them in a file list or a warning is this id. It carries no
// label, so the placeholder a reader sees is the [image] it always was.
// TestAnInlineObjectCarriesItsObjectID is the pin.
//
// # Named ranges are keyed by id, and they live in the tab
//
// NamedRange is {id, name, tab, ranges}, Document.NamedRanges is every one of
// them in tab order, and Tab.NamedRanges is the same list for one tab, the way
// Places is one rule with two ways in. The list is sorted by name and then by
// id, because the answer is a map and a map is walked in no order: a survey
// that lists a document's ranges differently on each run is a survey nobody can
// diff.
//
// The id is the identifier and the name is a label. Two ranges may wear one
// name, Docs puts both under that one key, and deleting by name deletes every
// range wearing it, so nothing here is keyed by name and a caller that wants
// one range names its id. TestTwoNamedRangesSharingANameKeepTheirOwnIDs is the
// pin.
//
// They are read from tabs[].documentTab.namedRanges, because every read gdoc
// makes carries includeTabsContent=true, which leaves the top-level field
// empty. The pre-tabs location is read too, beside the top-level body Parse
// already has a branch for: reading the body from one place and the ranges from
// another would report none on exactly the documents whose ranges are at the
// top level. TestNamedRangesAreReadFromTheTabAndNotTheTopLevel and
// TestAPreTabsDocumentsNamedRangesAreReadFromTheTopLevel are the two halves.
//
// Range carries Segment for these, and Places refuses a span that names one. A
// named range span names the header, footer or footnote it sits in, and covers
// walks the tab's body alone, so a header span answered against the body names
// a position in text it was never measured in: the pre-tabs fixture produced
// exactly that, a {4, 9, h.headerone} span falling inside tab t.0's body run
// and coming back placed. Segment is empty on every comment anchor, because
// rawCommentAnchor does not read the field, so no output shape moved when the
// refusal landed. That is the decoder and not Google: the commentAnchors shape
// was measured on comments anchored in the body, which carry no segment id
// whatever Docs sends for one anchored elsewhere, and whether Docs anchors one
// there at all is unmeasured, in
// docs/backlog/comment-anchors-in-headers-and-footnotes.md.
// TestANamedRangeCarriesItsTabItsRangesAndItsSegments is the pin.
//
// # NamedRangesURL is the narrowed read
//
// The whole document answers with the ranges already, so a survey takes them
// out of the read it has rather than making a fourth request. The narrowed read
// is for the caller that wants one fact out of that answer, and its one caller
// is restyle.RevisionOf, which cmd/gdoc's restyle makes between the marker
// batch and the styling phase, for the revision id alone: the measured saving
// was 982 bytes against 12,907 for the document itself. The apply loop reads
// nothing between its own batches.
//
// Its mask selects each tab's id and its named ranges, and childTabs whole. A
// field mask does not recurse into a nesting of unknown depth, so selecting a
// child tab's fields one level at a time would leave a deeper tab's ranges
// silently absent, which is the class of defect this package exists to refuse.
// docsReadParams already permits fields and checkFields refuses only a star, an
// empty mask and the permission surface, so the guard carries this unchanged.
// TestNamedRangesURLIsACallTheGuardCarries and
// TestTheNarrowReadParsesThroughTheSameDecoder are the pins.
//
// # Two things the tree does not carry
//
// A footnote's text is flattened into Footnotes, keyed by id, so a suggestion
// inside one is in neither read's markers nor the pending listing, and nothing
// warns. A chip inside a footnote is in no tab's blocks and is counted nowhere
// for the same reason. That hole is written down rather than guessed at, in
// docs/backlog/suggestions-inside-footnotes.md.
//
// A body over the read's ceiling is an error naming the ceiling, never a short
// read. The bound is gapi.MaxJSONBody, because a body without one is a memory
// limit somebody else sets, and truncating instead made the JSON reader say the
// answer is not JSON, which names something the server did not do. Parse
// refuses a body that is not JSON in those words, and
// TestParseRefusesABodyThatIsNotJSON is the pin.
package docs
