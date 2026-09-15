// Package publish puts a rendered document into Drive as a Google Doc.
//
// The upload is one multipart create: a JSON metadata part naming the folder,
// the title and the Google Doc type, then the docx bytes internal/render
// wrote. Drive converts the second part into a document because the first one
// asked for it, and the new document's id comes back in the answer.
// TestTheUploadNamesTheFolderTheTitleAndTheConversion is the pin.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # The policy opens with a folder and no file at all
//
// Every other writer here is handed a document. This one makes it, so the run's
// only door is the folder, and the new document's id is learned from the create
// the guard itself carried. There is no fallback to a folder the note already
// names: a note whose block reads at all carries a document id, so a note that
// was published once is refused before anything leaves the machine.
// TestOptionsRefuseWhatAPublishCannotBeMadeFrom and
// TestAnOptionsFailureStopsBeforeAnythingLeavesTheMachine are the pins, and
// TestAnUploadTheGuardRefusedIsAnErrorNamingTheFolder is the door from the
// other side.
//
// # Not knowing never resolves to keeping the document
//
// A pairing that could not be recorded leaves a document nobody knows about,
// and the next publish of the same note makes a second one. So a failed
// front-matter write is rolled back: Rollback trashes the document and reads it
// back, and only a rollback that also failed puts the live id on the envelope
// with the recovery steps. Three failure shapes, and each says something
// different.
//
//   - The write failed and the rollback held: no document id, rolled_back true.
//   - The write failed and the rollback did not: the live id, rolled_back
//     false, and a warning naming the document to trash by hand.
//   - The create Drive accepted could not be read: there is no id to verify, to
//     record or to trash, so the error names the folder and says a document may
//     be in it.
//
// The third is internal/probe's shape on its own create, for the same reason:
// the guard learned no id either, so it would refuse the trash in any case.
// TestARollbackDriveConfirmedSaysSo,
// TestARollbackDriveDidNotConfirmNamesTheDocumentToDeleteByHand,
// TestARollbackDriveRefusedIsNotACleanRollback,
// TestTheReportCarriesTheIDARollbackIsMadeFrom and
// TestAnUploadWhoseAnswerWasLostNamesTheFolderAndSaysADocumentMayBeThere are
// the pins.
//
// # rolled_back is a pointer, absent on a run that recorded the pairing
//
// cmd/gdoc prints it that way on purpose. With a plain bool and omitempty the
// failed-rollback case, which is the run where the live id matters most, would
// print nothing at all; without omitempty every clean publish would say
// rolled_back false about a rollback nobody tried. A rollback that held clears
// the document id and the URL, because naming a document that has gone sends
// somebody to look for it.
//
// All three shapes are pinned in cmd/gdoc: absent in
// TestPublishUploadsTheNoteAndPairsIt, true in
// TestPublishRollsBackWhenTheNoteCannotBePaired, false in
// TestPublishReportsTheLiveIDWhenTheRollbackFailed.
//
// # verified is three read-backs, and false is not a failure
//
// The Docs read says the document is there and readable, the tab count says it
// is one document rather than a shape a later command would refuse, and the
// docx export says Drive can hand it back as the format it came in as. Fewer
// than three is the document reported with the route that did not hold named:
// a document that exists is a document that exists, and a caller told the run
// failed is a caller that uploads a second one. The export check reuses
// internal/docx's ExportURL, Export and Parse rather than a second export path.
// TestAPublishUploadsVerifiesAndReportsTheDocument,
// TestAReadBackThatFailedIsReportedRatherThanRaised,
// TestADocumentWithTwoTabsIsNotVerified, TestAnExportThatIsNotADocxIsNotVerified,
// TestAnExportThatNeverArrivedIsNotVerified and TestRunSendsTheThreeRequestsInOrder
// are the pins.
//
// # The title is two different facts
//
// The title on the envelope is the read-back's, which is what Drive named the
// file. The title in the note's publish record is the cover's, which is what
// went on the page. Run warns when they disagree, and that warning is not a
// fourth check: Drive takes the name from the metadata part and the usual
// answer is that they match. A read-back that did not happen leaves the
// envelope's field out rather than filling it in with the title that was asked
// for. TestATitleThatCameBackDifferentIsNamedOnBothSides is the pin.
//
// # The note is read again just before it is written, and this rule is the
// inverse of the other writers'
//
// The other writers refuse a note whose block has gone. Publish refuses one
// whose block has appeared. Four refusals, each of them a rollback: a note that
// could not be read again, one whose front matter no longer parses, one whose
// block appeared during the upload, and one whose bytes changed at all. The
// fourth is the whole file rather than the body alone, because the author's own
// front matter feeds the cover: a title edited during the upload is as stale a
// render as an edited paragraph, and there is no honest way to call one a
// change and the other not. The re-read itself is cmd/gdoc's, in pair, beside
// the other writers' freshNote. TestPublishRollsBackWhenTheNoteCannotBePaired,
// TestPublishReportsTheLiveIDWhenTheRollbackFailed and
// TestPublishRefusesANoteThatIsAlreadyPaired in cmd/gdoc are the pins.
//
// # Nothing here reads or writes a file
//
// The note's bytes are cmd/gdoc's, which is what keeps this package testable on
// a fake session, and it is also what makes Rollback a separate call: whether
// the pairing could be recorded is a question about a file, and this room does
// not have one. build and publish render through one function there, so the
// file build wrote and the part publish uploaded are the same bytes.
// TestBuildAndPublishRenderTheSameBytes in cmd/gdoc is the pin.
//
// # Nothing calls this from a skill
//
// A publish is invoked by hand. Wiring it into a skill is M9's, with the
// install story.
package publish
