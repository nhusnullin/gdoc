// Package propose writes one change into a Google Doc as a native suggestion,
// with a comment beside it saying why, and then proves it by reading the
// document back through routes the write did not go out on.
//
// Every write here is writeControl.writeMode SUGGEST. The guard refuses a
// batchUpdate on a handed-in document without it, and that refusal is about
// gdoc's own words: what Google did with them is a different question, and one
// morning the answer was a silent direct edit. BLOCKED-BY-API.md holds both
// measurements. So the run does not stop at a 200, and internal/probe asks the
// question on a throwaway document before this package sends anything: a probe
// that comes back not enrolled means propose writes nothing at all.
// TestProposeSendsNothingWhenTheProbeSaysNotEnrolled in cmd/gdoc is the pin.
//
// Nothing here decides whether a change is worth proposing, or what to say in
// the comment. The words arrive written and the placement arrives quoted, in a
// file the skill wrote. This package finds the words, writes what it was told,
// and reports what it saw.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # There are two kinds, and the words kind names no kind at all
//
// A proposal is either the words kind, which replaces words inside one
// paragraph, or the block kind, which adds paragraphs after a quoted one or in
// place of a run of whole ones. The block names Kind "block"; the words kind
// names nothing, so every proposals file written before the block kind existed
// still reads, and a caller that never heard of a kind sends the kind it meant.
// A kind that is neither is refused by name, because reading it as the words
// kind would refuse a misspelled "blcok" for quoting no text, which names
// nothing the author did wrong. The other way round is refused by name too: an
// entry carrying after, replace_from, replace_to or content and naming no kind
// is a block somebody forgot to mark, so Check says which field it saw and which
// kind that field belongs to. A caller's decoder cannot hold this, because both
// kinds are read into the one type and those are fields it knows, and reading it
// as the words kind would either refuse it for quoting no text or, when it
// quotes text as well, send it as a words proposal with its content quietly
// dropped. TestCheckRefusesABlockFieldWithoutTheBlockKind is the pin.
//
// Apply dispatches on the field, so the command walks a file of both kinds
// without asking which an entry is. TestAnEmptyKindIsStillTheWordsKind,
// TestApplyDispatchesOnTheKind and the unknown kind case of
// TestCheckRefusesEachBadBlockByName are the pins.
//
// The two kinds share everything after the batch has gone out: one read they
// are built from, one post, and the answer read the same way, because what an
// answer means does not depend on which requests were in the body. They part
// company over what they place, what they build and what they ask the three
// read-backs. TestApplyBlockReadsPlacesWritesOnceAndVerifies and
// TestApplyBlockReportsALostAnswerRatherThanRaising are the block's half.
//
// # A block names exactly one placement form, and Check is where that is asked
//
// After a quoted paragraph, or in place of the run from replace_from to
// replace_to, and never both, never half a replace, and never neither.
// PlaceAfter and PlaceReplace each take the quotes they need, so a block naming
// neither form would reach one of them as an empty quote and be refused in
// words about a quote that is not in the document. A block carrying quoted or
// replacement is refused too: those belong to the words kind, and a block that
// carried them would have them silently ignored while the placement it really
// used came from somewhere else. TestCheckRefusesEachBadBlockByName and
// TestCheckAcceptsABlockInEitherPlacementForm are the pins.
//
// # A proposal names text, never an index
//
// The caller hands over the exact words to replace, Apply reads the document
// fresh and FindSpan looks them up in what came back, and a quote that occurs
// more than once is refused: quote more of the sentence. Text already inside a
// pending suggestion does not match either, so a proposal on top of a proposal
// is refused rather than stacked. An index computed a minute ago is the hazard
// the whole API has, and DECISIONS.md says never to act on a stored one. The
// lengths on the wire are UTF-16 code units, which is what the Docs API counts,
// so a replacement carrying a character outside the basic plane has its own
// test. TestFindSpanGivesTheRangeOfTheOneMatch,
// TestFindSpanRefusesTextThatIsNotThere, TestFindSpanRefusesTextThatOccursTwice,
// TestFindSpanRefusesAMatchInsideAPendingSuggestion,
// TestFindSpanCountsInUTF16CodeUnits and TestBatchCountsTheReplacementInUTF16CodeUnits
// are the pins.
//
// # A match has to be contiguous, and that is not a detail
//
// The index walk reads text runs and skips the rest, but the document numbers
// what it skipped, so words either side of a footnote mark, a picture, an
// equation, a page break or a smart chip read as one string in the walk and are
// two spans in the document. matches refuses a match whose span is longer than
// the words in it, because the deleteContentRange built from one would mark the
// skipped content for deletion along with them, and all three read-backs would
// still pass over it: the inline check reads text runs, so the footnote it just
// proposed deleting is invisible to it. Such a quote comes back as a refusal
// naming what it crossed, not as "not found". prelude's Decide refuses a marker
// broken into two spans with the document's own words between them, which is
// this rule on the other side:
// TestAMarkerBrokenAcrossTheAuthorsTextIsRefused. TestFindSpanRefusesAQuoteThatRunsAcrossAFootnoteMark and
// TestAQuoteCrossingAChipIsRefused are the pins, and the second says in its own
// words that the rule has to hold for the reason rather than by luck.
//
// Two things about that check are decisions rather than details.
//
// The span's end comes from the last rune of the match, never from the byte
// behind it. The position of that byte is the start of the next indexed run, so
// a quote ending exactly where a footnote mark or a picture begins would measure
// a unit too long and be refused for crossing a hole it only touches. That is
// the one refusal a reader walks straight into: read prints a footnote reference
// as [^1], so the obvious sub-quote is the words right before the mark.
// TestFindSpanPlacesAQuoteThatEndsAtAFootnoteMark,
// TestFindSpanStillPlacesAQuoteBesideAFootnoteMark and
// TestAQuoteEndingWhereAChipBeginsIsPlaced are the pins.
//
// A crossing occurrence still counts towards the exactly-once rule. FindSpan
// refuses a quote that occurs once as written text and once across a hole,
// rather than placing it on the contiguous one: picking would choose for the
// caller, and Carries rests on that same guarantee, so a dropped crossing copy
// is one the preview check would still find after a direct edit took the other.
// TestFindSpanRefusesAQuoteWhoseOtherOccurrenceCrossesAFootnoteMark is the pin.
//
// # A proposal replaces words with words
//
// An empty replacement is refused in Proposal.Check, before the probe. The batch
// would carry an insertText with no text and a comment anchored on a range of
// length zero, which Docs rejects, and inlineHolds looks for an insertion a
// plain deletion never makes, so the write could never verify either. A
// milestone that wants a deletion-only proposal gives it its own request shape.
// TestCheckRefusesEachBadProposalByName, over "no replacement", and
// TestApplyRefusesADeletionOnlyProposalBeforeAnyWrite are the pins.
//
// A replacement carrying a character Docs strips out of an inserted text is
// refused there too. Batch anchors the comment on the words it inserted,
// counted from the replacement's own length, so a unit the server drops leaves
// the anchor reaching past the replacement into words nobody proposed to
// change. It is docsreq.Strippable's set, the block kind's rule read on the
// words kind's one index. The two strippable cases of
// TestCheckRefusesEachBadProposalByName are the pin.
//
// # A block's content is a subset, and everything outside it is refused by name
//
// The second kind of proposal is a block: whole paragraphs, placed after a
// quoted paragraph or in place of a run of them. Its content arrives as
// markdown, and ParseContent reads it into the paragraphs the write is built
// from: a named style, a list membership, and runs carrying marks.
//
// The subset is paragraphs, headings one to six, bulleted and numbered lists one
// level deep, and three marks, which are bold, italic and a link. A wrapped line
// is one paragraph and the wrap is a space, because a paragraph is a paragraph
// because of the empty line between two of them. A list item is NORMAL_TEXT with
// a bullet, because in Docs a bullet is a list membership and never a style.
// TestContentBecomesTheParagraphsTheBlockWrites,
// TestContentReadsAHeadingAtItsOwnLevel, TestContentReadsTheMarksASentenceCarries,
// TestContentReadsBothListKinds, TestContentJoinsAWrappedLineWithASpace and
// TestParaTextIsTheWordsWithoutTheMarks are the pins.
//
// Everything else is a refusal naming the line, never a construct quietly
// dropped. A block that lost its table is a suggestion nobody can read and
// nobody can explain, and the words around the hole read as if they were the
// whole answer. The refusals are a table, a nested list, a picture, a code block,
// a block quote, HTML in either shape, a horizontal rule, code in a sentence,
// struck-out words, a task list, a line break inside a paragraph, a heading
// inside a list item, a list item with more than one paragraph, a paragraph or a
// heading with no words in it, and content that is empty or nothing but space.
// Two of them are decisions rather than gaps: a nested list, because nesting a
// suggested list goes in through leading tabs and that is unmeasured in SUGGEST
// mode, and a table, because docs/backlog/propose-inside-tables.md is not
// settled. TestContentRefusesWhatTheSubsetDoesNotHold carries every case. Check
// reads the content too, so every one of these refusals is made of the whole
// proposals file before the probe document exists and before the first entry
// lands, and ApplyBlock reads it again before it reads the document, so a block
// gdoc cannot read costs no request at all:
// TestApplyBlockRefusesContentItCannotReadBeforeAnyRequest and
// TestProposeRefusesBlockContentBeforeAnythingIsSent in cmd/gdoc are the pins.
//
// A refusal names the line because content is a file a person wrote and a block
// is tens of lines long. A horizontal rule is the one construct goldmark builds
// with no source position at all, so its line is the first line holding anything
// after the block above it: naming line 1 would send the author to the top of a
// file whose rule is thirty lines down, which is the wrong end of it.
// TestARefusedConstructNamesItsLine and
// TestARuleWithNoSourcePositionStillNamesItsLine are the pins.
//
// A link has to be an address a document can open, which means a scheme or a
// host. A hub path and a "#heading" jump have neither, and in a Google Doc both
// open nothing, so they are refused rather than written as dead links or
// silently unlinked. internal/body keeps the words and drops the link instead,
// because it is publishing a whole note and losing one link is cheaper than
// losing the publish. Here the content is a few lines a review wrote, so
// rewriting them costs nothing. An email autolink gets the mailto: scheme
// goldmark leaves off. TestContentKeepsAnAddressADocumentCanOpen and the two
// link cases of TestContentRefusesWhatTheSubsetDoesNotHold are the pins.
//
// A run carries the words markdown says it carries, rather than the ones the
// author had to type. goldmark leaves an entity reference and a backslash escape
// in the source segment and resolves both in its HTML renderer, so a run taken
// raw would put "R&amp;D" and "snake\_case" into somebody's document, and every
// read-back would hold, because each one compares against that same text. A
// link's destination is a source segment too, and is read the same way, so a
// query string written with &amp; opens the address the author meant.
// internal/body does the entity half of the words one file over, for the same
// reason. TestContentResolvesEntitiesAndBackslashEscapes is the pin.
//
// The decoding is one left-to-right pass, and not the order util.URLEscape
// chains goldmark's three resolvers in. Chained, the backslash comes off
// "\&amp;" and the five characters behind it are then read as an entity, so an
// author who escaped an entity on purpose gets one ampersand. CommonMark says
// an escaped character is literal and never the start of a reference, and
// goldmark's own text writer makes this same single pass. A reference is read
// by its shape where it begins, and only then handed to the resolvers, because
// both of them read every "&" in whatever they are given: handed the span up to
// the next ";", the "&" of "R&D" would resolve an entity further along the line
// and hand back the escape between them undecoded. The escaped-entity cases of
// TestContentResolvesEntitiesAndBackslashEscapes are the pin.
//
// A decoded character the document cannot take is refused, and two kinds are.
// A character the Docs API strips out of an inserted text is the first:
// docsreq.Strippable names them, and BlockBatch counts every index it sends
// from the length of this text, so a unit the server drops moves each of them
// one place and the deleteContentRange of a replace ends inside a paragraph
// nobody quoted. A line break is the second: Docs makes a paragraph of a
// newline and a soft break of a vertical tab, so one written as "&#10;" would
// arrive as a paragraph the block never proposed, styled and bulleted as part
// of the one in front of it. Both are refused where the markdown spelling of
// the same thing is, because a decoded character is as invisible in the file as
// a pasted one. The three decoded-character cases and the two raw ones of
// TestContentRefusesWhatTheSubsetDoesNotHold and
// TestStrippableNamesTheCharactersTheAPIRemoves in internal/docsreq are the
// pins. internal/cover refuses the same set in a fields file, for the same
// reason and through the same function.
//
// goldmark is configured with the GFM extensions and nothing else. GFM is on so
// that a table, a struck-out word and a task list are parsed and refused by
// name: with it off, each one is ordinary text and the table arrives in the
// document as rows of pipes. The typographer internal/body enables is off, and
// so are footnotes, because these words are a review's own and a straight quote
// turned curly is a change gdoc made that nobody asked for.
//
// gdoc's own markers are not asked about here. Every route into a document asks,
// and for the proposals file that is cmd/gdoc, which asks it of every field of
// every entry rather than of this one string.
//
// # A block goes in at the start of a paragraph, never at the end of one
//
// PlaceAfter puts the text at the start of the paragraph behind the anchor, and
// PlaceReplace at the start of the first paragraph it covers. MEASURED.md's
// block table is why: an insert at a paragraph's end hands that paragraph's old
// mark to the last new paragraph, so restyling it is a second suggestion, and a
// proposal withdraw cannot take back whole is not one this package will make. At
// a start every new paragraph owns a mark of its own, and the styles, the
// bullets and the bullet removal all fold into the one insertion id.
// TestAfterPlacesTheBlockAtTheStartOfTheNextParagraph,
// TestAfterPlacesABlockBehindAListItem and TestAReplaceCoversWholeParagraphs are
// the pins.
//
// A replace covers whole paragraphs, from the start of the one holding
// replace_from to the end of the one holding replace_to, paragraph marks
// included. A partial paragraph would mean an accept merges what is left of it
// with a neighbour, which is a change nobody proposed.
// TestAReplaceInsideOneParagraphCoversThatParagraph is the pin for the
// one-paragraph case, where both quotes land in the same paragraph.
//
// A placement is computed from the read the write is built from and is never
// stored, for the reason every index in this package is never stored.
// TestAPlacementIsComputedFromTheReadAndNotStored is the pin.
//
// # The last paragraph in a document is the one place the block's own shape matters
//
// There is no paragraph to go in front of there, so the text goes in before the
// body's final newline and the block's last paragraph inherits that mark:
// MEASURED.md row 7, which came back with one id because nothing restated that
// paragraph. BlockBatch restates nothing on it either, neither the named style
// nor the bullet removal, which is what keeps this shape the measured one:
// TestBlockBatchAfterTheLastParagraphRestatesNothingOnTheFinalMark is the pin.
//
// That leaves the block's last paragraph taking whatever the mark already
// carries, so three cases are refused there by name. A block whose last
// paragraph is not plain body text, because restyling the final mark is the
// unmeasured second-id case. A document whose last paragraph is a list item,
// because the block would take its bullet and clearing a bullet on the
// mark-owning paragraph is MEASURED.md row 4, the one measured case that gave
// two ids. And a document whose last paragraph carries any other named style,
// because the block's plain last paragraph would arrive as that style, and
// restating the mark to stop it is the unmeasured case again. Every message
// says what would work instead, and Nail can widen any of them after a
// measurement. TestAfterTheLastParagraphGoesBeforeTheFinalNewline and four
// cases of TestPlaceRefusesWhatItCannotPlaceAndNamesIt are the pins.
//
// # A replace never deletes somebody else's work, and never deletes what it cannot see
//
// When the paragraphs a replace covers hold a pending suggestion or a comment's
// anchor, the placement is refused and every suggestion id and comment id is
// named. That is a fact rather than a judgement: this package does not decide
// whether a colleague's pending edit matters, it reports that it is there and
// stops. The skills read the same two facts before they ask, and tell the person
// what is in the way.
//
// The run is refused too when it covers anything but paragraphs, or when two
// paragraphs in it do not touch, because something this read does not index, a
// section break among them, sits between them and the deleteContentRange would
// take it with them. That is the contiguity rule of the words kind, one level
// up. An after block is refused for the same reason when the element behind the
// anchor is a table or the contents list Docs generates, because there is no
// paragraph start to go in at, and when the paragraph behind it does not begin
// where the anchor ends. A quote inside a table cell is refused on both routes
// while docs/backlog/propose-inside-tables.md is unsettled.
//
// A replace reaching the body's last paragraph is refused because Docs will not
// delete a document's final newline, so the delete would half happen. Every one
// of these is a case of TestPlaceRefusesWhatItCannotPlaceAndNamesIt.
//
// # A line break is refused on both sides, in one place, for two reasons
//
// A quoted carrying one is refused because a paragraph's last text run carries
// the paragraph mark itself: the Docs read hands back "...operations team.\n",
// so a quote ending in a newline matches inside that one paragraph and the span
// FindSpan returns ends past the mark. The deleteContentRange built from it
// marks the mark for deletion, and accepting the suggestion merges the paragraph
// with the one behind it while the insertText puts back a replacement that
// cannot carry a break. Nothing downstream would name it: inlineHolds compares
// the deleted runs against that same quoted, and Carries finds that same string
// in the preview, so all three read-backs hold over a proposal that removes a
// paragraph. A quote with a break in the middle is already unreachable, because
// it spans two paragraphs and the walk indexes one at a time, so the rule costs
// a caller nothing: the words without the trailing mark are always writable
// instead.
//
// A replacement carrying one is refused because of the preview check. Carries
// asks one paragraph at a time, and a paragraph ends at its own break, so a
// string whose newline is anywhere but the very end is in no single paragraph
// and comes back false whatever the document holds. A trailing one is the
// exception, for the quote rule's own reason, so it can be found. Either way the
// answer is worthless. The quote rule gives the preview's first question a
// quoted that cannot carry a break. Nothing gives its second question that, so a
// replacement holding both the quote and a newline would fall past the ambiguity
// arm and report preview_without_suggestions as holding on exactly the silent
// direct edit that route exists to name. Both preconditions are enforced at the
// door rather than left implied. TestCheckRefusesEachBadProposalByName carries
// all four cases.
//
// # Every proposal is checked before the first one is sent
//
// Proposal.Check answers from the proposal alone, so the caller asks it of every
// entry in the file before anything leaves the machine: a third entry refused
// after the first two have landed is a run that half happened in somebody's
// document, with a probe document created and trashed on the way. Reading that
// file is cmd/gdoc's, and it reads it strictly, the way it reads every other
// input: an unknown key is refused by name, and so is a second list behind the
// first. A misspelled quoted, replacement or why is caught here because their
// empty values are refused, but assignee is optional, so a dropped one would
// land a comment with nobody assigned and warn about nothing.
// TestApplyRefusesAQuoteItCannotPlaceBeforeAnyWrite is the pin here;
// TestProposeRefusesABadProposalBeforeTheProbe,
// TestProposeRefusesAnUnknownKeyInTheProposalsFile and
// TestProposeRefusesAnEmptyProposalList in cmd/gdoc are the other half.
//
// A reason that already carries the robot is refused too, because Batch writes
// the comment as Prefix + Why and a caller following reply's convention would
// sign it twice. internal/plaintext holds that rule and says why the two writers
// own the mark differently. TestCheckRefusesEachBadProposalByName carries the
// near misses.
//
// The reason is the one rule both kinds ask, because both kinds write it into a
// thread as Prefix + Why. Everything else Check asks is the kind's own: the
// words kind's quote and replacement, and the block's placement form, its
// content read through ParseContent, and that content's ceiling.
// TestCheckRefusesEachBadBlockByName is the block's half of this rule, over the
// same reason cases.
//
// # One batch per proposal, three requests inside it
//
// deleteContentRange over the quoted span, insertText at its start, and
// insertComment over the inserted span, in that order. One batch, because Docs
// applies the requests in order with consistent indexes, so the comment lands on
// the span the insert made. Two proposals are two batches, each after its own
// fresh read, because the first moves the ground under the second. A document
// with more than one tab stops the run before anything is sent, the probe
// included: a range means nothing without saying which tab it is in.
// TestBatchSendsThreeRequestsInOrderUnderSuggestMode,
// TestApplyReadsTheDocumentItselfThenWritesOnce,
// TestBatchCarriesTheAssigneeWhenThereIsOne and
// TestApplyRefusesAMultiTabDocumentBeforeAnyWrite are the pins, with
// TestTheGuardCarriesEveryOneOfProposesRequests and
// TestTheGuardRefusesTheSameBatchWithoutSuggestMode asking the same batch of the
// guard itself.
//
// # One batch per block too, and the order of its requests is the measurement
//
// BlockBatch writes, in this order: insertText at the placement; one
// updateParagraphStyle per new paragraph naming namedStyleType; one
// updateTextStyle clearing the marks the insert inherited; one updateTextStyle
// per marked run; createParagraphBullets per stretch of list items of one kind;
// deleteParagraphBullets per stretch of new paragraphs that are not list items;
// deleteContentRange for a replace; and insertComment. Every index is counted in
// UTF-16 code units, the way FindSpan counts. TestBlockBatchForAnAfterBlock,
// TestBlockBatchForAReplace and TestBlockBatchAfterTheLastParagraph hold the
// three whole batches against golden request lists, because the order is the
// reason the batch works and a field-by-field test would say nothing about it.
// TestBlockBatchCountsInUTF16CodeUnits is the unit rule and
// TestBlockBatchNumbersANumberedList the second bullet preset.
//
// A named style is stated on every new paragraph, list items included, because
// an inserted paragraph takes the named style of the paragraph it landed in: a
// block in front of somebody's Heading 1 would arrive as headings. It is the
// rule internal/prelude holds for the same reason.
//
// Two requests undo what the insert inherited, and both are measured. The
// clearing updateTextStyle covers the whole insert and comes before the block's
// own marks, because text inserted at a paragraph's start takes that paragraph's
// first run style, so a block in front of a bold linked sentence would arrive
// bold and linked; a run that really is bold is written bold again after it. Its
// mask names every mark and its style object is empty, which is how the API is
// told to put a field back to its default: stating bold false would clear a
// boolean, but nothing states "no link" except naming the field in the mask and
// leaving it out of the object. TestBlockBatchClearsTheMarksTheInsertInherited
// is the pin, over a document whose next paragraph opens bold and linked.
//
// deleteParagraphBullets covers every new paragraph that is not a list item, and
// no paragraph that was already there, because new text takes the list
// membership of the paragraph it lands in front of and stating a named style does
// not clear it (MEASURED.md rows 5 and 6). It is one request per stretch rather
// than one per paragraph, and the stretches stop at the list between them: a
// request covering that list would take its bullets off too.
// TestBlockBatchRemovesBulletsFromItsOwnParagraphsOnly is the pin.
//
// A replace's deleteContentRange names the old paragraphs at the indexes the
// insert left them at, which is both ends moved along by the length of the
// insert, because the insert went in at the start of the first of them.
//
// The comment is anchored on the words of the first new paragraph, without its
// paragraph mark, because a range carrying a mark anchors across into the
// paragraph behind it. It carries the robot prefix and the reason, and nothing
// else: TestBlockBatchWritesTheReasonAsPlainTextUnderTheRobot.
//
// # A block's content has a ceiling, and it is the guard's rather than Docs'
//
// The guard reads a batchUpdate body to judge the requests in it, and a body
// past its peek arrives there truncated and is refused rather than carried
// unread. That refusal names the guard and a truncated body, which is nothing a
// person writing a block can act on, so MaxContent is refused first and says
// what a block is for. The number is a measurement: the batch grows fastest in
// requests per byte where the shortest paragraphs alternate with list items,
// because then each paragraph costs a named style and a bullet request of its
// own, and that shape at MaxContent builds around half the guard's ceiling.
// TestALargeBlockStaysUnderThePeek builds that shape and three more at exactly
// the limit and has the real policy judge each one, and the content over the
// ceiling case of TestCheckRefusesEachBadBlockByName is the refusal that keeps
// the guard's words off the screen.
//
// # Verified is three read-backs, and Verified false is not a failure
//
// Checks carries them as three fields, and each answers something the other two
// cannot. suggestions_inline: the replacement is in the document, carrying a
// suggestion id. preview_without_suggestions: the quoted words are still
// somewhere in the tab with suggestions hidden, so it is a suggestion and not an
// edit. docx_anchored: the docx export carries the robot comment, attached to
// text.
//
// All three, plus a write that answered commentUpdateState ALL_SAVED, is
// Verified. Anything less is the write reported with the route or the answer
// that did not hold named, because the write happened: a caller told the run
// failed is a caller that writes it again. The fourth condition is why a run
// with three true checks can still be unverified: a batch Docs accepted whose
// answer could not be read leaves no state to report, and the warning there
// names the lost answer rather than blaming a route. preview_without_suggestions
// is the route that would catch the silent direct edit, which is why it is one
// of the three rather than a nicety. TestVerifyHoldsOnAllThreeRoutes,
// TestApplyVerifiesTheHappyPathThreeWays,
// TestApplyReportsAPartialFailureFromTheCommentUpdateState,
// TestApplyReportsABatchWhoseAnswerCouldNotBeRead,
// TestApplyKeepsACommentIdDecodedFromAnAnswerThatFailed,
// TestVerifyWarnsWhenTheExportCouldNotBeRead and
// TestApplyAcceptsAnInsertDocsCutIntoTwoRuns are the pins.
//
// # The preview check asks by words, never at the index
//
// The span's start was counted in the view that shows pending suggestions, and
// the preview hides them, so every position after one sits lower there. Looking
// at that index reported the second proposal of every run, and every document
// already carrying somebody's pending insertion, as a direct edit: the one
// warning that must never cry wolf. Carries asks whether the tab still holds the
// quoted words anywhere, which holds up because FindSpan required them to occur
// exactly once, so a direct edit usually takes the only copy with it.
//
// Usually, because a replacement that contains the quote carries it through the
// edit: the write puts the replacement where the quoted words were, so "reviewed
// annually" is still in the preview inside "reviewed annually by the operations
// team", and the quote alone would report the route as holding on the exact
// failure it exists for. So Verify asks a second question in that shape only,
// and it is whether the preview carries the replacement. After an honest
// suggestion it does not, because the preview hides the insertion, unless the
// document already read that way before the write, which is a proposal that
// duplicates the words behind it. Those two cannot be told apart from here, so
// the check is false with a warning naming the ambiguity rather than one naming
// a direct edit. Asking for the replacement outside that shape would be the
// cry-wolf mistake again: a replacement that does not contain the quote can
// occur anywhere in the document.
//
// What no question catches is a second copy of the quote inside somebody's
// pending suggested deletion, which the preview still shows. That is the price
// of asking by words, and it is the cheaper of the two mistakes.
// TestVerifyReadsThePreviewByWordsRatherThanAtTheIndex,
// TestCarriesReadsThePreviewByWordsRatherThanByIndex,
// TestVerifyStillCatchesADirectEditInThePreview,
// TestVerifyCatchesADirectEditWhoseReplacementCarriesTheQuote,
// TestVerifyDoesNotCryWolfOnAnHonestAddWordsProposal,
// TestApplyFailsThePreviewCheckWhenTheWriteWasADirectEdit,
// TestVerifyReadsThePreviewThroughItsOwnView and
// TestPreviewURLAsksForThePreviewViewOfEveryTab are the pins.
//
// # A block's three read-backs are the same three routes asking its own questions
//
// VerifyBlock takes the placement and the paragraphs rather than a proposal,
// because every question is about what the write did: where the block went, what
// its paragraphs say, and which old paragraphs a replace marked.
//
// suggestions_inline asks both halves, the way the words kind does. Every new
// paragraph is at the index the write's own layout computed, carrying a suggested
// insertion that reads as the paragraph was written, and for a replace every
// paragraph it stands in for carries a suggested deletion on all of its words. An
// insertion alone would be the block added beside the paragraphs it replaces, a
// deletion alone would be those paragraphs struck out with nothing in their
// place, and a paragraph half struck out would cut somebody's sentence in two if
// it were accepted. The one paragraph that is not wholly inserted is the last one
// of a block placed after the document's last paragraph: the text went in before
// the body's final newline, so that newline stays the document's own, which is
// MEASURED.md row 7 and not a failure. TestVerifyBlockHoldsOnAllThreeRoutes,
// TestVerifyBlockHoldsForAReplace, TestVerifyBlockHoldsAfterTheLastParagraph,
// TestVerifyBlockCatchesANewParagraphThatIsNotSuggested and
// TestVerifyBlockCatchesAReplaceThatDeletedNothing are the pins.
//
// preview_without_suggestions asks two questions. The paragraphs the placement
// stood on are still in the preview, because a direct edit of a replace takes
// them out and a suggested deletion leaves them. Then the block's first new line
// is not, because the preview hides an insertion and shows a direct edit's
// written text. That second question is asked by words, like the words kind's,
// so it can only ever answer false rather than accuse: the line may be there
// because the block was written as an edit, or because the document already
// carried that line elsewhere. Those two cannot be told apart from here, so the
// check is false with a warning naming both readings. Passing instead would
// report the silent direct edit, which is the one thing this route exists to
// catch, as a route that held.
//
// The first question cannot answer alone, because Carries asks by substring: a
// replace whose new content carries the old paragraphs inside it answers it
// after a direct edit too. It is the shape the words kind asks its own second
// question for, one file over, and the reason is the same. So the line the
// second question asks about is the block's first line the placement was not
// already standing on: its own first line for an after block unless the anchor
// carries it, and for a replace the first line the old paragraphs do not
// already carry. That is what keeps the commonest block there is verifiable, a
// section rewritten under its own heading, whose opening line is one the
// replace itself covers. Skipping a line costs nothing, because a direct edit
// writes the whole block and any line of it answers for all of them.
//
// A block with no line left to ask about is where the first question has to
// hold alone, and it can do that only when the content drops a paragraph the
// placement stood on: that is the replace that only shortens, and a direct edit
// of it takes the dropped paragraph out. A replace that says every old
// paragraph again, a list conversion or a reorder, gives the first question
// nothing to catch, and an after block gives it nothing ever, because a direct
// edit there leaves the anchor where it was. Both answer no answer rather than
// passing. TestVerifyBlockCatchesADirectEditInThePreview,
// TestVerifyBlockGivesNoAnswerWhenThePreviewCarriesTheFirstLine,
// TestVerifyBlockGivesNoAnswerWhenAReplaceKeepsEveryOldParagraph,
// TestVerifyBlockGivesNoAnswerWhenAReplaceOnlyReshapes,
// TestVerifyBlockGivesNoAnswerWhenAnAfterBlockSaysNothingNew,
// TestVerifyBlockHoldsWhenAReplaceOnlyShortens and
// TestVerifyBlockHoldsForAReplaceThatKeepsItsFirstLine are the pins.
//
// docx_anchored is the words kind's own check, unchanged: the export carries the
// robot comment and it is attached to text.
//
// # One suggestion id is what a verified block means
//
// Every request of the block's batch folded into one id when it was measured, so
// frontmatter.Proposal keeps its one id and withdraw is unchanged. A read-back
// showing more than one is a block gdoc cannot take back whole, so
// suggestions_inline is false and Verified with it. Every id is reported anyway,
// in the order they were met, because the block is in the document either way and
// those ids are how somebody finishes the job by hand: the first is the one the
// note records and the one withdraw will take back, and the warning says so.
// TestVerifyBlockReportsEverySuggestionIDAndWarnsAboutWithdraw is the pin.
//
// The ids this reads are an insertion's and a deletion's, which is what the
// batch can make and what internal/docs decodes. A style change on a paragraph
// the document already had would be an id under a key neither reads, and the
// one paragraph a block ever shares is the final mark at the end of a document,
// which the batch restates nothing on. TestLiveProposeBlock asks that case
// again with the probe's wider walk, across every suggested key, because on a
// live run it is Google's answer rather than gdoc's request that decides it.
//
// # docx_anchored gives no answer when two comments disagree
//
// The export carries no Drive comment id, so two proposals in one run with the
// same reason are two comments with one body. Taking the first would report one
// proposal on the strength of the other's comment, so when the matches disagree
// about being attached the check is false with a warning naming the ambiguity.
// Duplicates that agree answer correctly for both proposals, and the check is
// whatever they agree on. It is the rule internal/docx's own witness follows, on
// the same join. TestDocxHoldsRefusesTwoCommentsThatDisagree and
// TestApplyFailsTheDocxCheckWhenTheCommentIsNotInTheExport are the pins.
//
// # Provenance is the permission to withdraw
//
// The note's proposals[] is the only place gdoc's own work is remembered, and
// withdraw refuses an id the note does not record as its own. So a propose run
// without a note still lands the suggestion and simply forgets it, and Record
// writes the entry whether or not the read-backs held: a proposal reported
// unverified is in the document either way, and a proposal gdoc has forgotten is
// one it will refuse to withdraw.
//
// What it cannot record, it names. An entry needs both ids, and the block's own
// validation refuses one missing either, so a change that landed without one of
// them in hand cannot be written down at all. Record hands those results back to
// the caller instead of dropping them, and the caller turns each into a warning
// naming the words the proposal is known by: the change is in the document and
// withdraw will refuse it for ever.
//
// Those words are Result.Quote's, because a block replaced no words. A words
// proposal is known by its quote, a block by where it went: the after quote, or
// the first of a replace's two, which is the paragraph the block went in at.
// There is one field in the note for it, so a replace keeps the first, and the
// note reads the same for both kinds. TestRecordRemembersABlockByItsPlacement
// and TestRecordRemembersAReplaceByItsFirstQuote are the pins, with
// TestProposeRecordsABlocksPlacementInTheNote in cmd/gdoc. The warning names the route the id would have come from,
// and the two routes are not the same one: the comment id is the batch's own
// answer, while the suggestion id is read out of the inline read-back
// afterwards. Blaming the write for a read-back that failed sends somebody to
// look at Docs while the envelope's other warning is already saying the re-read
// is what broke. TestRecordAppendsTheProposalToTheNote,
// TestRecordSkipsAProposalWithNoSuggestionIDAndSaysSo and
// TestRecordLeavesTheNoteAloneWhenNothingCanBeRemembered are the pins, with
// TestProposeRecordsProvenanceForAnAcceptedButUnverifiedProposal,
// TestProposeSaysSoWhenTheNoteCannotRememberAProposal and
// TestProposeReportsEveryProposalWhenOneOfThemCannotBeSent in cmd/gdoc.
//
// # Two rules this package rests on and does not hold
//
// A write whose answer could not be read is not a write that never happened.
// internal/gapi marks the failures raised after the server answered 2xx, and
// this package asks by behaviour rather than by importing that package: naming a
// Session interface here is what keeps net/http out of this room, and an
// imported sentinel would bring it back through the side door. Apply runs the
// read-backs on that path and reports the proposal with the comment id unknown,
// or with whatever the answer still carried. internal/gapi's own comment holds
// the rule and the three cases it does not cover.
//
// The note is read again just before it is written, because the run spends
// seconds to tens of seconds on the network between the pairing check and the
// write, and these notes live in a synced vault. That rule is cmd/gdoc's, in its
// package comment, under "The note is read again just before it is written".
package propose
