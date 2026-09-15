// Package probe asks Google, on a document gdoc made for the purpose, whether
// a suggestion written today is honoured as a suggestion.
//
// The reason it exists is in docs/v2/BLOCKED-BY-API.md. writeMode is absent
// from the public Docs discovery document, and one morning a batchUpdate
// carrying SUGGEST answered 200 and made a direct edit instead. The guard
// refuses a write on a handed-in document unless the body says SUGGEST, but the
// guard reads gdoc's own words: what the server did with them is a different
// question, and only a read-back answers it.
//
// So the probe is that read-back, made where being wrong costs nothing. It
// creates a throwaway document in the folder the command was given, writes one
// sentence into it directly, suggests one word inside that sentence, reads the
// document back and looks for the word carrying a suggestion id. Then it puts
// the document in the trash and confirms it went.
// TestRunSendsTheSixRequestsInOrder is the pin over the shape of the run, and
// TestRunReportsEnrolledWhenTheWordCameBackAsASuggestion and
// TestRunReportsNotEnrolledWhenTheWordCameBackAsPlainText over the two answers.
//
// Nothing here decides anything. Enrolled is a fact about what Google answered,
// and what to do about a false one is the caller's. internal/propose is the one
// caller today, and it sends nothing at all when the answer is false:
// TestProposeSendsNothingWhenTheProbeSaysNotEnrolled in cmd/gdoc is that pin.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # It runs every time, and nothing caches the answer
//
// The binary is one-shot and holds no hidden state, so there is nowhere to put
// a cached verdict, and an asserted "already probed" handed in from outside
// reopens the exact failure the probe exists to prevent. The question is what
// Google does this morning, and yesterday's answer is not an answer to it.
// TODO(test): no test pins this rule yet. It holds because Run takes no cache
// and the process ends after one command.
//
// # It never touches the document being reviewed
//
// The policy the caller opens for a proposal has two doors: the document at
// LevelSuggest, and the probe's folder as the one place a create may land. This
// package is given only the folder, and the probe document's id is learned from
// the create the guard itself carried, which is the second door. So no
// handed-in document is reachable from here, and the probe cannot write into
// the document somebody is reviewing.
// TestTheProbeDocumentIsNeverHandedIn is the pin, with
// TestTheCreateNamesExactlyTheFolderAndAsksForADocument and
// TestTheGuardCarriesEveryOneOfTheProbesRequests beside it.
//
// The write into the probe document is direct and says so. That is not a hole
// in the SUGGEST rule: the rule is about a document gdoc was handed, and this
// one is a document gdoc made a moment earlier, at LevelFull, with nothing in
// it to protect. The suggested write that follows is the measurement itself.
// TestTheDirectInsertIsDirectAndTheSuggestInsertSaysSuggestExactly and
// TestTheSuggestInsertLandsInsideTheSentenceTheDirectInsertWrote are the pins.
//
// # Every failure path trashes, and names the document
//
// The trash runs whatever the measurement did, because a document gdoc created
// and left behind is litter in somebody's Drive and the step that failed is not
// a reason to add to it. A probe document that could not be trashed is named in
// probe_document_id with trashed false, never silent.
// TestAFailedSuggestWriteStillTrashesAndSaysSo,
// TestATrashThatDidNotHoldIsAWarningRatherThanSilence and
// TestATrashDriveDidNotConfirmIsNotReportedAsTrashed are the pins.
//
// The warning says the document may still be in the folder, and never that it
// is. It is one sentence over drive.Trash's three failures and it takes the
// weakest of them: only a read-back answering trashed false knows where the
// file is, while a failed PATCH and an unconfirmed read say nothing either way.
// Not knowing must never resolve to a fact, in either direction.
// TestATrashDriveCouldNotConfirmDoesNotClaimTheDocumentIsInTheFolder is the pin.
//
// The one case the report cannot name is the create Drive accepted whose answer
// could not be read. The id was in that answer, so there is nothing to put in
// the field and nothing to trash, and the guard learned no id either, so it
// would refuse the trash in any case. The failure says a document may be in the
// folder rather than that it was not created, because Drive made it.
// TestACreateDriveAcceptedButCouldNotBeReadSaysSo and
// TestACreateWithNoIDIsAnErrorNamingTheFolder are the pins.
//
// # The first production caller of AllowCreateIn
//
// The guard's create door was kept at M2 on the strength of publish needing it
// at M6. This package is what reached it first, at M3. The door itself, and
// what a create is judged on, are internal/guard's, in its package comment
// under "Two doors into the set, and four grants beside it".
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
// The trash is believed only on its confirming read, and that rule is
// internal/drive's. publish takes its document back the same way, and a second
// copy of those three steps is a second chance for the two to disagree about
// whether an unconfirmed trash counts as a trash.
package probe
