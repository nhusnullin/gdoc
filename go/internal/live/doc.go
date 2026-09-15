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
//
// # GDOC_LIVE_RECORD writes somebody's document into testdata
//
// GDOC_LIVE_RECORD=1 makes the read test additionally save the Docs read and
// the docx export into testdata/. Those bytes are a real document's content,
// so a person redacts them before they are committed. The recording the
// measured anchors fixture was modelled on is gitignored for that reason.
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
//     2026-09-10, MEASURED.md "A named range over a suggested insertion".
//   - TestLiveNamedRangeOverSuggestionProbe, M7c's. One fresh document per
//     case, because a probe that measures its own leftovers answers about
//     itself.
//   - TestLiveTableIndexProbe, M7c's, and the one the prelude's arithmetic was
//     corrected from. It reads the index map of an inserted table off a real
//     document, then sweeps the indexes behind that table one document at a
//     time and reads each accepted one back to say where the insert really
//     landed. Measured 2026-09-10, MEASURED.md "A table takes one index of its
//     own at the end".
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
// # One test here asks for no network at all
//
// TestTheLiveFixturesRenderWithNoNetwork runs in `make test`, with neither
// variable set. Two of the live tests render a note before they reach Drive,
// so a note that stopped rendering or a fixture path that moved would
// otherwise be found by Nail in the middle of a live run rather than by the
// suite. It is the pin, and it is in publish_test.go beside the tests it
// covers.
package live
