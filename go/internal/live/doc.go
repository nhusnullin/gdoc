// Package live holds the opt-in end-to-end tests, the reads and the writes,
// plus the one render check that asks for no network.
//
// Everything else in this tree runs against a fixture. A fixture is what
// Google sent on the day somebody saved it, so a suite of them says the
// readers still read that day's answer and says nothing at all about today's.
// These tests use the real token, the real guard and real Google Docs, and
// they are the only place that difference can be measured.
//
// Tests only. No production code lives here, and no request is built here
// either: every write goes out through the writer package that owns it, so
// this package names no *http.Request and is on neither of the boundary
// test's two allowlists. That is why M2 could have no write test at all,
// and why M3 could: a create and a comment are POST requests, and building
// one means naming net/http, which the wire rule allows in four rooms and
// not in a package whose only files are tests.
//
// This comment holds what each test does and what it costs. What the tests
// assert is in the code beside them.
//
// # Two variables, because a read and a write are two decisions
//
// GDOC_LIVE_TEST=1 runs the read test. GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1
// adds the write tests beside it. Everything is skipped without them, so
// `go test ./...` on any machine runs nothing here.
//
// Two rather than one, because a live read is somebody's document and a live
// write is a document that did not exist a second ago. The second is a
// different decision and it is made on purpose each time. The unattended run
// sets neither.
//
// A document id has no default anywhere in this package, and that is the
// guard's rule rather than a convenience: a policy is opened with exactly the
// id the run named, so a run that names none has nothing to reach.
// GDOC_LIVE_FOLDER_ID is the exception and does have one, the Drive test
// folder, because a folder here is a create target and not a document.
//
//   - GDOC_LIVE_DOC_ID: the document the read test reads.
//   - GDOC_LIVE_FOLDER_ID: where the write tests create, defaulting to the
//     Drive test folder.
//   - GDOC_LIVE_IDEAL_DOC_ID: the document the ten-feature acceptance copies.
//   - GDOC_LIVE_ANCHOR_DOC_ID: the document the anchors test copies.
//   - GDOC_LIVE_PRELUDE_DOC_ID: the document the prelude acceptance copies.
//   - GDOC_LIVE_ACCEPTED_DOC_ID: the probe document Nail has accepted by hand.
//   - GDOC_LIVE_EXPORT_DOC_ID: the document M13's measurement reads both ways.
//     Nail makes it by hand in the test folder, and it holds, in this order,
//     one inline PNG picture uploaded from the pictures note under
//     internal/body/testdata/docs, one Google Drawing, one floating picture,
//     and a second tab titled Appendix with one picture in it. The
//     measurement creates nothing and writes nothing to Drive.
//   - GDOC_LIVE_PUBLISHED_DOC_ID: a document publish made, in the test folder.
//   - GDOC_LIVE_RESTYLED_DOC_ID: a document restyle styled with --fields, in
//     the test folder. Those two are read only under GDOC_LIVE_RECORD, for the
//     two prelude fixtures the export's strip rests on.
//
// # GDOC_LIVE_RECORD writes somebody's document into testdata
//
// GDOC_LIVE_RECORD=1 makes the read test additionally save the Docs read and
// the docx export into testdata/. Those bytes are a real document's content,
// so a person redacts them before they are committed. The recording the
// measured anchors fixture was modelled on is gitignored for that reason.
//
// The measurement below records elsewhere, because what it saves is another
// package's fixture: the read and the export of GDOC_LIVE_EXPORT_DOC_ID go
// under internal/export/testdata/fixture-measured/, and the reads of
// GDOC_LIVE_PUBLISHED_DOC_ID and GDOC_LIVE_RESTYLED_DOC_ID go under
// publish-prelude/ and restyle-prelude/ beside it. The same warning holds:
// they are somebody's document until a person has read them.
//
// # The read test creates nothing
//
// TestLiveReadOfARealDocument runs the whole read path against Google: the
// token is loaded and refreshed if it has expired, every request goes through
// the guard, and the readers run on what really came back. It reads and it
// writes nothing, to Drive or to disk, unless GDOC_LIVE_RECORD says otherwise.
//
// # Which test creates, and which copies
//
// The difference decides what a run can damage, so it is stated per test
// rather than left to be read out of the code.
//
// Created from nothing, in the test folder, and trashed on the way out:
//
//   - TestLiveProposeReplyWithdraw, M3's. It creates a document, proposes into
//     it, replies, withdraws and trashes it, asserting every read-back on the
//     way.
//   - TestLiveWaitSeesANewComment, M4's. It starts a wait, posts a comment
//     into its own document while that wait is running, checks the comment
//     came back before the deadline, then waits again on the cursor it was
//     handed and checks that window is empty, up to 90 seconds and then 15.
//   - TestLivePublish, M6's. It publishes a temp note into the folder and
//     asserts the read-back, the title, the one tab and the block written into
//     the note.
//   - TestLiveDrift, M6's. It uploads the built document and the master with
//     conversion and runs drift.Items over both. That is the measurement that
//     means something, because Google's import is part of the result.
//   - TestLiveStyleFidelity, the measurement M7b was scoped from. It sends
//     each candidate styling request kind in a batch of its own and logs what
//     landed, asserting almost nothing: a measurement that fails the build
//     when Google answers differently has already decided the answer.
//   - TestLiveSuggestedInsertProbe, M7c's. One candidate request kind per
//     case, recording which of them Docs accepts as a suggestion. Nine of ten,
//     and the tenth is createNamedRange, refused in Docs' own words. Measured
//     2026-09-10, and the ten-row table is in DECISIONS.md, "The house template
//     reaches a document as a suggestion, not as a direct edit".
//   - TestLiveNamedRangeOverSuggestionProbe, M7c's. One fresh document per
//     case, because a probe that measures its own leftovers answers about
//     itself. Measured 2026-09-10, MEASURED.md "A named range over a suggested
//     insertion".
//   - TestLiveTableIndexProbe, M7c's, and the one the prelude's arithmetic was
//     corrected from. It reads the index map of an inserted table off a real
//     document, then sweeps the indexes behind that table one document at a
//     time and reads each accepted one back to say where the insert really
//     landed. Measured 2026-09-10, MEASURED.md "A table takes one index of its
//     own at the end".
//   - TestLiveExportRoundTrip, M13's, and the one that answers a question no
//     fixture can. It publishes each of the six notes under
//     internal/body/testdata/docs, exports the document Drive made of it, and
//     compares the file that came back with the note it started as. Six
//     documents, all six trashed. The comparison cannot be equality, because a
//     docx carries no fence, no emphasis character and no link to a file in the
//     hub, so it is the drift gate's shape: a named list of differences, each
//     with the reason it is there, in roundtrip_test.go. Every name that fired
//     is logged with its count, and a line no name covers fails the run. A name
//     that has to join the list is a decision Nail writes down with its reason,
//     never a test somebody loosens.
//   - TestLiveExportCarriesAProposal, M13's. It publishes the policy note,
//     proposes one change into the document, exports, and asserts the
//     suggestion id and the comment id are in the file, so a session merging
//     that file can see what is proposed and what is not. It withdraws the
//     proposal through a second policy granted that one id, and trashes the
//     document.
//   - TestLiveExportHashEquality, M13's, and a measurement until the binary
//     makes it a rule. It publishes the pictures note, exports it, and prints
//     the sha256 of each picture beside the sha256 of the file on this disk it
//     was uploaded from. It asserts the equality only once cmd/gdoc turns
//     export.Options.MatchByHash on, which it asks by reading that file: a
//     constant here would be a second answer that could disagree with the
//     binary. One document, trashed. This is measurement 2 of
//     docs/v2/MEASURED.md.
//   - TestLivePublishAgainAppendsAnEntry, M13's. It publishes one note twice
//     and reads the block back: two entries, two different documents, both
//     readable through the Docs API, both trashed. That is scenario 1, the
//     refusal M13 took out.
//   - TestLiveAnnotateParagraphAndTable, M11's. It creates a document holding
//     one sentence and a one-row table of two cells, fills the cells, leaves
//     one comment on words in the sentence and one on words inside a cell, and
//     reads both back through Drive's listing and the docx export. The
//     paragraph case is the milestone's acceptance bar and asserts both routes.
//     The table case is a measurement and asserts nothing: it is the question
//     docs/backlog/propose-inside-tables.md was left open on, and an
//     insertComment with no deleteContentRange beside it is what tells the
//     suspects apart. Measured 2026-09-18: both comments verified, in the cell
//     as in the paragraph.
//
// Copied, and the original never written to:
//
//   - TestLiveRestylePreservesTenFeatures, M7b's acceptance. It copies
//     GDOC_LIVE_IDEAL_DOC_ID with copyComments=true under AllowCopy, checks
//     the copy holds all ten features before a single request is built,
//     restyles the copy at LevelInPlace, asserts all ten again, and re-reads
//     the original on every path to prove its revisionId never moved.
//   - TestLiveRestyleKeepsAnchorsAndSuggestions, the cheap half of that
//     acceptance. It asks whether an in-place restyle keeps one comment
//     attached to its words and one suggestion pending, which is a question
//     any reviewed document can answer, so it runs on any day against
//     GDOC_LIVE_ANCHOR_DOC_ID.
//   - TestLivePreludeIsProposedNotWritten, M7c's acceptance. It copies
//     GDOC_LIVE_PRELUDE_DOC_ID, proposes the house prelude into the copy on a
//     policy that granted nothing, sends the marker once before the grant to
//     assert the guard refuses it, then grants and sends it again, and reads
//     the whole thing back through prelude.Verify: every piece carrying a
//     suggestion id, the marker over the span that was proposed, and the
//     author's own text character for character what it was.
//
// A copy is the shape that keeps a run harmless, and AllowCopy exists for it:
// the source is handed in at LevelSuggest with no in-place grant, so a restyle
// of the source itself is refused before it leaves the machine.
//
// Reads only, and it needs a document somebody prepared:
// TestLiveNamedRangeAfterAcceptedByHand reads GDOC_LIVE_ACCEPTED_DOC_ID back
// once Nail has accepted the probe's suggestion in the browser. The accept row
// cannot be measured from here at all, and that is the guard working: gdoc
// cannot accept its own suggestion, so the probe leaves that document in the
// folder and prints its URL.
//
// # Three tests leave their document behind, on purpose
//
// The anchors test, the prelude acceptance and the named range probe's accept
// row each leave a document in the folder with its URL in the log. The numbers
// say the anchors survived and the suggestion ids are there; whether the cover
// reads right is Nail's, in the document. Trash them once you have looked.
//
// # The export measurement asserts nothing
//
// TestLiveExportMeasurements is M13's, and it is the one test here that is a
// measurement and nothing else. It reads GDOC_LIVE_EXPORT_DOC_ID through the
// Docs API and through the docx export and prints the two lists of pictures
// side by side: the object ids in body order per tab against the media parts
// in word/document.xml order, the sha256 of each media part against the
// pictures the fixture was uploaded from, and the count of w:drawing against
// each tab's own count. It fails when a read or an export fails and on
// nothing else, because the three answers are Google's and a test that
// asserted them would be deciding what it was sent to find out. Task 13 of
// M13 writes them into MEASURED.md.
//
// # Two tests here ask for no network at all
//
// TestTheLiveFixturesRenderWithNoNetwork runs in `make test`, with neither
// variable set. Two of the live tests render a note before they reach Drive,
// so a note that stopped rendering or a fixture path that moved would
// otherwise be found by Nail in the middle of a live run rather than by the
// suite. It is the pin, and it is in publish_test.go beside the tests it
// covers.
//
// TestTheMeasurementReadsTheFixtureItPairsAgainst is the same pin for the
// measurement, in export_measure_test.go: it compares what Drive sends back
// against the pictures on this disk, so a picture that moved out of
// internal/body/testdata/docs is found by the suite instead.
package live
