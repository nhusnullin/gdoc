// Package withdraw retracts one of gdoc's own pending suggestions, and proves
// it is gone before saying so.
//
// Nothing here decides whether a proposal should be withdrawn. The id arrives
// chosen, in a note the skill and Nail have already read. This package checks
// that the id is gdoc's own, rejects it, and reports what it saw.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # A withdrawal is a rejectSuggestion, not a delete
//
// The write is one rejectSuggestion naming the id, in SUGGEST mode like every
// other write on a handed-in document. Nail's decision of 2026-09-07, in
// DECISIONS.md, and it was measured before it was taken.
//
// It was a deleteContentRange over the insertion until the first live write
// test ran. A proposal is a replace, one suggestion id over a suggested
// deletion and a suggested insertion, and that delete retracts only the
// insertion half: the new words go, Docs answers updatedSummarySuggestionIds
// rather than deletedSuggestionIds, and the quoted words stay suggested-deleted
// under the same id. A second delete over them is a no-op, and one delete over
// both halves marks the new words inserted and deleted at once. rejectSuggestion
// takes the whole thing back in one request: the document reads as it did
// before the proposal, the answer names the id in
// suggestionResponses[].rejectedSuggestionIds, and the robot comment survives,
// still anchored by id, so the skill can reply into it.
// TestBatchIsOneSuggestModeRejectNamingTheId and
// TestRunSendsOneSuggestModeRejectNamingTheId are the pins, with
// TestRunTakesBackAProposalOnlyItsDeletionHalfStillCarries over the half the
// old delete used to leave behind.
//
// A reject names no range, so a document with more than one tab is withdrawn
// from like any other. propose refuses such a document, because a range means
// nothing without saying which tab it is in, and this package has no range.
// TestRunWithdrawsFromADocumentWithMoreThanOneTab is the pin.
//
// # Provenance is the permission, and the guard is what holds it
//
// Nothing in a suggestion id says who wrote it, and the Docs API will happily
// reject anybody's. The only record gdoc has of its own work is the proposals[]
// list propose wrote into the note's front matter, which is why withdraw
// requires --md. Mine is that question in one function, so the command asks the
// same question this package asks rather than a paraphrase of it, and a
// suggestion missing from the list is refused before the document is read.
// TestMineReadsTheNoteAndNothingElse,
// TestRunRefusesASuggestionTheNoteDoesNotName and
// TestRunRefusesWhenTheNoteIsMissing are the pins, with
// TestWithdrawRefusesASuggestionTheNoteDoesNotRecord in cmd/gdoc.
//
// The refusal in the guard is the one that counts. The guard refuses every
// batchUpdate request kind whose name carries "suggestion", and the one door
// through that wall is Policy.AllowReject(id), which cmd/gdoc seeds from the
// same note Mine reads, after Mine has said so and before the session is built.
// So this package never touches the policy: it asks the session to send, and
// the guard says whether that id is one of gdoc's own in this run.
// acceptSuggestion and deleteSuggestion stay refused whatever id they name, a
// rejectSuggestion naming another id is refused, and a second field beside the
// id is refused because nobody here has read what it does. That is
// internal/guard's, in its package comment under "Two doors into the set, and
// five grants beside it", and
// TestAGrantedRejectSuggestionCarriesAndNothingElseInTheFamilyDoes is the pin.
//
// Rejecting gdoc's own unaccepted proposal is not resolving Nail's decision,
// because there was no decision yet. The rule that gdoc never accepts, rejects
// or deletes anyone else's suggestion holds, and the grant is one id wide and
// dies with the process.
//
// # The document is read before the write, and again after it
//
// The answer to "is this still pending" is only true at the moment it is read,
// so a reject of a suggestion that was already accepted or rejected would be
// reported as a withdrawal with nothing withdrawn. The read before the write is
// what refuses that, and pending asks about both sides of the id, because a run
// still marked for deletion is half a proposal that is still pending.
// TestRunReadsTheDocumentItselfBeforeWriting,
// TestRunRefusesASuggestionTheDocumentDoesNotCarry and
// TestRunFailsWhenTheDocumentCannotBeRead are the pins.
//
// # Gone only when both facts hold
//
// Verified is the answer naming the suggestion in rejectedSuggestionIds and a
// fresh read carrying no run under that id on either side. The first is Docs
// agreeing with itself; the second is the only route that can say the
// suggestion has actually left the document. Either side, because taking back
// only the insertion half is the failure this package was rewritten to close.
// TestRunRejectsTheSuggestionAndVerifiesItIsGone,
// TestRunIsUnverifiedWhenTheAnswerNamesNoRejectedSuggestion,
// TestRunIsUnverifiedWhenTheReadBackStillCarriesTheId,
// TestRunIsUnverifiedWhenTheReadBackStillCarriesTheDeletionHalf and
// TestRunReportsAReadBackItCouldNotMake are the pins.
//
// The entry leaves the note only then. Forgetting it while the suggestion is
// still pending would leave gdoc refusing to withdraw its own work, so Forget
// copies rather than edits and the caller writes the note only when the
// withdrawal held. TestForgetLeavesTheOtherProposalsInPlace,
// TestForgetDoesNotMutateItsInput, TestForgetLeavesABlockThatNeverNamedTheSuggestion
// and TestForgetOnANilBlockIsNil are the pins, with
// TestWithdrawRetractsAndForgetsTheProposal in cmd/gdoc.
//
// The price is a reject Docs accepted whose answer could not be read. Usually
// no ids came back, so Verified stays false, the entry stays in the note, and a
// retry is refused by the pending check rather than by the note. The run says
// so, and says to take the entry out by hand once the suggestion is gone.
// Relaxing the two facts to one on that path is a decision for Nail, not a
// refactor. TestRunReportsARejectWhoseAnswerCouldNotBeRead is the pin.
//
// # Usually is the whole word, and the gate is the decoded field
//
// Valid JSON of the wrong shape is the one failure that reaches the caller with
// fields in hand, because encoding/json saves the first type error and keeps
// decoding. So the rejectedSuggestionIds the server really did send are kept
// whatever path the answer arrived on, and both the warning and Verified are
// built from what decoded rather than from the path being taken. An empty list
// on that path is the usual case and says nothing; a list the server really
// sent is its own word about what it rejected; a list naming another suggestion
// is the alarm whichever path carried it. A withdrawal whose two facts both
// held is reported verified, and the warning then says the answer was lost
// without telling anybody to take out an entry the same run removed.
// TestRunKeepsTheIdsDecodedFromAnAnswerThatFailed and
// TestRunReadsAnotherSuggestionOutOfIdsThatDecodedOnAFailedAnswer are the pins.
//
// A write that never left the machine is different again, and nothing else is
// sent after it: a guard refusal and a 4xx are the write failing whole.
// TestRunFailsWhenTheWriteDoesAndSendsNothingElse is the pin.
//
// # The robot comment stays where it is
//
// A withdrawn proposal leaves its comment in the thread. The guard's
// commentWrites carries POST and nothing else, so editing or deleting a comment
// is a write it does not carry, and no command here needs one: the skill
// replies into the thread saying the proposal was withdrawn. A milestone that
// needs PATCH or DELETE adds it beside its caller. The rule is held where the
// refusal is: guard's TestNoCommentPatchOrDelete is the pin.
//
// # Two rules this package rests on and does not hold
//
// A write whose answer could not be read is not a write that never happened.
// internal/gapi marks the failures raised after the server answered 2xx, and
// this package asks by behaviour rather than by importing that package: naming
// a Session interface here is what keeps net/http out of this room, and an
// imported sentinel would bring it back through the side door. internal/gapi's
// own comment holds the rule and the three cases it does not cover.
//
// The note is read again just before it is written, because the run spends
// two whole-document reads and a batchUpdate on the network between the pairing
// check and the write, and these notes live in a synced vault. That rule is
// cmd/gdoc's, in its package comment, under "The note is read again just before
// it is written". TestWithdrawWritesTheNoteTheVaultHasNow is the pin.
package withdraw
