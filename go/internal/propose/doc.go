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
// naming what it crossed, not as "not found". withdraw.Span refuses two spans
// with somebody else's words between them, which is this rule on the other
// side. TestFindSpanRefusesAQuoteThatRunsAcrossAFootnoteMark and
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
// naming the quoted words: the change is in the document and withdraw will
// refuse it for ever. The warning names the route the id would have come from,
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
