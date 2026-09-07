// Package live holds the opt-in end-to-end tests: the real token, the real
// guard, and real Google Docs. Everything here is skipped unless
// GDOC_LIVE_TEST=1, so `go test ./...` on any machine runs nothing.
//
// There are three tests, and they reach Drive in different ways.
//
// The read test is pointed at a document by GDOC_LIVE_DOC_ID, with no default,
// and it writes nothing. The guard is opened with exactly the id the run was
// given, and a run that names none has nothing to read.
//
// The two write tests need GDOC_LIVE_WRITE=1 as well, and each touches only
// documents it created itself, in the test folder. M2 could not have them: a
// create and a comment are POST requests, this package would have to build
// them, and building a request means naming net/http, which the boundary test
// allows in four rooms and not in a package whose only files are tests. M3 has
// the production writers, so the first drives probe, propose, reply and
// withdraw and still builds no request of its own. M4 adds the second, which
// waits while a comment is written into a document it made and asserts the wait
// saw it.
package live

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/docx"
	"gdoc/internal/frontmatter"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/probe"
	"gdoc/internal/propose"
	"gdoc/internal/reply"
	"gdoc/internal/suggestions"
	"gdoc/internal/view"
	"gdoc/internal/withdraw"
)

const (
	liveVar   = "GDOC_LIVE_TEST"
	docVar    = "GDOC_LIVE_DOC_ID"
	recordVar = "GDOC_LIVE_RECORD"
)

// TestLiveReadOfARealDocument runs the whole read path against Google: the
// token is loaded and refreshed if it has expired, every request goes through
// the guard, and the readers run on what really came back rather than on a
// fixture somebody wrote.
func TestLiveReadOfARealDocument(t *testing.T) {
	if os.Getenv(liveVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=<document id> to read one real document with the real token", liveVar, docVar)
	}
	id := strings.TrimSpace(os.Getenv(docVar))
	if id == "" {
		t.Fatalf("%s=1 needs %s=<document id>: the guard is opened with exactly the document the run names, and there is no default", liveVar, docVar)
	}

	p := guard.NewPolicy()
	p.AllowFile(id, guard.LevelSuggest)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	ctx := context.Background()

	// The raw answer first, so the recording is what Google sent and the parse
	// runs on the same bytes.
	var raw json.RawMessage
	if err := s.GetJSON(ctx, docs.URL(id), &raw); err != nil {
		t.Fatalf("the Docs read failed: %v", err)
	}
	d, err := docs.Parse(raw)
	if err != nil {
		t.Fatalf("the Docs read did not parse: %v", err)
	}
	if d.ID != id {
		t.Errorf("the read came back for document %q, and %q was asked for", d.ID, id)
	}
	if len(d.Tabs) == 0 {
		t.Error("a real document has at least one tab; includeTabsContent may have been dropped from the URL")
	}

	text, notes := view.Text(d)
	if strings.TrimSpace(text) == "" {
		t.Error("the text projection is empty, and a real document has text in it")
	}
	if !strings.HasSuffix(text, "\n") {
		t.Error("the text must end with exactly one newline")
	}
	t.Logf("read %d tabs, %d characters, %d pending suggestions, %d projection warnings",
		len(d.Tabs), len(text), len(suggestions.List(d)), len(notes))

	raws, err := comments.Fetch(ctx, s, id, nil)
	if err != nil {
		t.Fatalf("the comment listing failed: %v", err)
	}
	threads, unplaced := comments.Threads(raws, d)
	for _, thread := range threads {
		if thread.Marker == "" {
			t.Errorf("thread %s carries no marker, and every thread carries one, %q included", thread.ID, comments.MarkerNone)
		}
	}
	t.Logf("%d threads, %d of them unplaced by the Docs read", len(threads), len(unplaced))

	export, err := docx.Export(ctx, s, id)
	if err != nil {
		t.Fatalf("the docx export failed: %v", err)
	}
	f, err := docx.Parse(export)
	if err != nil {
		t.Fatalf("the docx export did not parse: %v", err)
	}
	for _, thread := range docx.Match(threads, f) {
		if thread.Witness == "" {
			t.Errorf("thread %s came back from the witness with no answer", thread.ID)
		}
		t.Logf("thread %s: marker %s, witness %s", thread.ID, thread.Marker, thread.Witness)
	}

	if os.Getenv(recordVar) == "1" {
		record(t, raw, export)
	}
}

// record writes the two answers to testdata/, for the fixture the decoder is
// later tightened against.
func record(t *testing.T, read json.RawMessage, export []byte) {
	t.Helper()
	recordFile(t, "live-docs-read.json", indented(t, read))
	recordFile(t, "live-export.docx", export)
}

// indented is the JSON as a person will read it in the diff. A body that will
// not indent is not a failure of the run: the read it came from has already
// passed, and the raw bytes are still worth keeping.
func indented(t *testing.T, raw json.RawMessage) []byte {
	t.Helper()
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		t.Logf("an answer could not be indented, so it is recorded as it came: %v", err)
		return raw
	}
	return pretty.Bytes()
}

// recordFile saves one answer under testdata/. What it writes is a real
// document's real content: it is read and redacted by a person before it is
// committed, and the log line says so rather than leaving it to be discovered
// in a diff.
func recordFile(t *testing.T, name string, body []byte) {
	t.Helper()
	dir := "testdata"
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("recorded %s: this is a real document's content, so redact it before committing it", path)
}

// The write test's own variables and the folder it creates in.
const (
	writeVar  = "GDOC_LIVE_WRITE"
	folderVar = "GDOC_LIVE_FOLDER_ID"

	// testFolder is Nail's Drive test folder, and the write test creates in it
	// unless GDOC_LIVE_FOLDER_ID names another one. It is a default where the
	// read test has none, and the difference is which door the id opens: the
	// read test is handed a document to read, so a default there would read
	// somebody's real document unasked, while this is a create target and every
	// document the run touches is one it made in there.
	testFolder = "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"
)

// The sentence the subject document is written with, and the change proposed
// into it. The quoted words occur exactly once, which is what FindSpan asks
// for.
const (
	subjectSentence = "The supplier register is reviewed annually by the operations team.\n"
	subjectQuoted   = "annually"
	subjectNew      = "every six months"
	subjectWhy      = "The policy above says the register is reviewed twice a year, so this sentence disagrees with it."
	secondQuoted    = "the operations team"
	secondNew       = "the operations and risk team"
)

// TestLiveProposeReplyWithdraw is the whole write path against Google: the
// probe on a document it creates and trashes, then a second document of its
// own, a proposal into it, a reply to the comment the proposal made, the
// withdrawal of that proposal, and the trash at the end.
//
// Every document it touches is one it created. The policy has one door open,
// AllowCreateIn on the folder, so the subject document is reachable only
// because the guard carried the create that made it, and no document of Nail's
// is in the reachable set at all.
func TestLiveProposeReplyWithdraw(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to create a document in the Drive test folder and write to it; %s names another folder", liveVar, writeVar, folderVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}
	t.Logf("creating in folder %s", folder)

	p := guard.NewPolicy()
	p.AllowCreateIn(folder)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	ctx := context.Background()

	// The probe first, because it is what propose runs before it writes: an
	// unenrolled project makes every assertion below meaningless, and the
	// probe is the one that says so.
	report, err := probe.Run(ctx, s, folder)
	for _, w := range report.Warnings {
		t.Logf("probe warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the probe failed, and its document is %q: %v", report.ProbeDocumentID, err)
	}
	if !report.Enrolled {
		t.Fatalf("the probe says SUGGEST is not honoured today, so nothing below can be a suggestion; probe document %q", report.ProbeDocumentID)
	}
	if !report.Trashed {
		t.Errorf("the probe document %q was left in the folder", report.ProbeDocumentID)
	}
	t.Logf("probe: enrolled, suggestion ids %v, document %s trashed", report.SuggestionIDs, report.ProbeDocumentID)

	docID := createSubject(t, ctx, s, folder)

	result := proposeInto(t, ctx, s, docID)
	if os.Getenv(recordVar) == "1" {
		recordInline(t, ctx, s, docID)
	}
	replyTo(t, ctx, s, docID, result)
	withdrawFrom(t, ctx, p, s, docID, result)
	if os.Getenv(recordVar) == "1" {
		recordBatch(t, ctx, s, docID)
	}
}

// createSubject makes the document the rest of the test writes to, puts one
// sentence in it, and registers the trash that runs whatever happens next.
//
// The insert is a direct edit rather than a suggestion, and that is the point
// of the level: the document was learned from a create the guard carried, so
// the policy holds it at LevelFull and Docs is asked plainly. A handed-in
// document would be refused here, which is the bar the whole milestone stands
// on.
func createSubject(t *testing.T, ctx context.Context, s *gapi.Session, folder string) string {
	t.Helper()
	var created struct {
		ID string `json:"id"`
	}
	body := map[string]any{
		"name":     "gdoc live write " + time.Now().UTC().Format(time.RFC3339),
		"mimeType": "application/vnd.google-apps.document",
		"parents":  []string{folder},
	}
	if err := s.PostJSON(ctx, createURL(), body, &created); err != nil {
		t.Fatalf("the subject document could not be created in folder %q: %v", folder, err)
	}
	if created.ID == "" {
		t.Fatalf("the create in folder %q answered with no document id", folder)
	}
	t.Logf("subject document %s", created.ID)
	t.Cleanup(func() { trashSubject(t, ctx, s, created.ID) })

	insert := map[string]any{"requests": []any{
		map[string]any{"insertText": map[string]any{
			"location": map[string]any{"index": 1},
			"text":     subjectSentence,
		}},
	}}
	if err := s.PostJSON(ctx, batchURL(created.ID), insert, nil); err != nil {
		t.Fatalf("the sentence could not be written into %q: %v", created.ID, err)
	}
	return created.ID
}

// proposeInto runs the production proposal and asserts all three read-backs.
// Anything short of Verified is a failure here: the unit tests already cover a
// route that did not hold, and what this run is for is Google agreeing with
// all three.
func proposeInto(t *testing.T, ctx context.Context, s *gapi.Session, docID string) propose.Result {
	t.Helper()
	result, err := propose.Apply(ctx, s, docID, propose.Proposal{
		Quoted:      subjectQuoted,
		Replacement: subjectNew,
		Why:         subjectWhy,
	})
	for _, w := range result.Warnings {
		t.Logf("propose warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the proposal could not be written: %v", err)
	}
	if len(result.SuggestionIDs) == 0 {
		t.Fatalf("the proposal landed with no suggestion id, so nothing below can withdraw it")
	}
	if result.CommentID == "" {
		t.Fatalf("the proposal landed with no comment id, so there is no thread to reply into")
	}
	if !result.Checks.SuggestionsInline {
		t.Error("the inline read did not show the replacement as a pending suggestion")
	}
	if !result.Checks.PreviewWithoutSuggestions {
		t.Error("the preview still shows the replacement, so the write was an edit rather than a suggestion")
	}
	if !result.Checks.DocxAnchored {
		t.Error("the docx export does not carry the comment as anchored to the new words")
	}
	if !result.Verified {
		t.Errorf("the proposal is not verified: state %q, checks %+v", result.CommentUpdateState, result.Checks)
	}
	t.Logf("proposed: suggestion ids %v, comment %s, state %s", result.SuggestionIDs, result.CommentID, result.CommentUpdateState)
	return result
}

// replyTo posts one reply into the thread the proposal made and asserts the
// read-back found it.
func replyTo(t *testing.T, ctx context.Context, s *gapi.Session, docID string, result propose.Result) {
	t.Helper()
	body := reply.Prefix + "This is the live write test. The proposal above is withdrawn a moment after this reply."
	posted, err := reply.Post(ctx, s, docID, result.CommentID, body)
	for _, w := range posted.Warnings {
		t.Logf("reply warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the reply to thread %q failed: %v", result.CommentID, err)
	}
	if !posted.Verified {
		t.Errorf("the reply %q was posted and the thread did not read back with it", posted.ReplyID)
	}
	t.Logf("replied: %s at %s", posted.ReplyID, posted.Created)
}

// withdrawFrom retracts the proposal, through the same provenance the command
// uses: the id is recorded into a note's front matter by propose.Record and
// read back by frontmatter.Read, so the run proves the memory as well as the
// write. A note that never named the id is a withdrawal the package refuses.
func withdrawFrom(t *testing.T, ctx context.Context, p *guard.Policy, s *gapi.Session, docID string, result propose.Result) {
	t.Helper()
	note := []byte("---\ngdoc:\n  schema: 1\n  document_id: " + docID + "\n---\n\n# Live write test\n")
	recorded, missed, err := propose.Record(note, []propose.Result{result}, time.Now().UTC())
	if err != nil {
		t.Fatalf("the proposal could not be recorded in the note: %v", err)
	}
	if len(missed) != 0 {
		t.Fatalf("the live proposal could not be remembered: %+v", missed)
	}
	block, err := frontmatter.Read(recorded)
	if err != nil {
		t.Fatalf("the note propose wrote could not be read back: %v", err)
	}
	id := result.SuggestionIDs[0]

	// Without the grant the guard refuses the reject before it leaves the
	// machine, whatever the note says: the note is the command's evidence, and
	// AllowReject is how the command hands it to the guard. Stated here against
	// the real guard, on a document gdoc itself created, because this is the
	// rule that keeps "anyone else's suggestion" true on the wire.
	if _, err := withdraw.Run(ctx, s, docID, id, block); err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("a reject with no grant must be refused by the guard, got: %v", err)
	}
	if !withdraw.Mine(block, id) {
		t.Fatalf("the note does not record %q, so nothing may be granted", id)
	}
	p.AllowReject(id)
	gone, err := withdraw.Run(ctx, s, docID, id, block)
	for _, w := range gone.Warnings {
		t.Logf("withdraw warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the suggestion %q could not be withdrawn: %v", id, err)
	}
	if !gone.Verified {
		t.Errorf("the withdrawal of %q is not verified: rejected ids %v", id, gone.RejectedSuggestionIDs)
	}
	if left := withdraw.Forget(block, id); withdraw.Mine(left, id) {
		t.Errorf("the note still records %q after the withdrawal", id)
	}
	t.Logf("withdrawn: %v", gone.RejectedSuggestionIDs)
}

// trashSubject puts the document away and confirms it went. It runs from
// t.Cleanup, so a failed assertion above still leaves nothing behind, and it
// reports rather than fails when Drive says otherwise: a document still in the
// folder is worth naming loudly and is not the thing the test was asking.
func trashSubject(t *testing.T, ctx context.Context, s *gapi.Session, docID string) {
	t.Helper()
	if err := s.PatchJSON(ctx, fileURL(docID), map[string]any{"trashed": true}, nil); err != nil {
		t.Errorf("the subject document %q could not be trashed and is still in the folder: %v", docID, err)
		return
	}
	var answer struct {
		Trashed bool `json:"trashed"`
	}
	if err := s.GetJSON(ctx, trashedURL(docID), &answer); err != nil {
		t.Errorf("the subject document %q was trashed, and Drive could not be asked to confirm it: %v", docID, err)
		return
	}
	if !answer.Trashed {
		t.Errorf("the subject document %q was trashed and Drive still reports it as not trashed", docID)
		return
	}
	t.Logf("subject document %s trashed", docID)
}

// recordInline saves the SUGGESTIONS_INLINE read while the proposal is still
// pending, which is the shape propose's unit fixtures stand on.
func recordInline(t *testing.T, ctx context.Context, s *gapi.Session, docID string) {
	t.Helper()
	var raw json.RawMessage
	if err := s.GetJSON(ctx, docs.URL(docID), &raw); err != nil {
		t.Logf("the inline read could not be recorded: %v", err)
		return
	}
	recordFile(t, "live-propose-read.json", indented(t, raw))
}

// recordBatch saves one batchUpdate answer, which no other route gives back:
// propose.Apply reads the answer and reports what it found, so a fixture of the
// raw shape has to come from a write the test makes itself.
//
// The request is built by the production builder, on a second set of words in
// the same throwaway document, so what is recorded is the answer to the call
// propose really sends. The document is trashed a moment later.
func recordBatch(t *testing.T, ctx context.Context, s *gapi.Session, docID string) {
	t.Helper()
	d, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		t.Logf("the document could not be read before the recording write: %v", err)
		return
	}
	p := propose.Proposal{Quoted: secondQuoted, Replacement: secondNew, Why: "A second proposal, made only to record the answer Docs gives."}
	r, err := propose.FindSpan(d, p.Quoted)
	if err != nil {
		t.Logf("the words to record a write on were not found: %v", err)
		return
	}
	var answer json.RawMessage
	if err := s.PostJSON(ctx, propose.BatchURL(docID), json.RawMessage(propose.Batch(r, p)), &answer); err != nil {
		t.Logf("the recording write failed: %v", err)
		return
	}
	recordFile(t, "live-propose-batch.json", indented(t, answer))
}

// The three Drive URLs the test spells for itself. They are the calls probe
// makes, and probe keeps its own copies unexported: a test is not a reason to
// widen a production package's surface.
func createURL() string {
	return "https://www.googleapis.com/drive/v3/files?fields=id&supportsAllDrives=true"
}

func batchURL(id string) string {
	return "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
}

func fileURL(id string) string {
	return "https://www.googleapis.com/drive/v3/files/" + id + "?supportsAllDrives=true"
}

func trashedURL(id string) string {
	return "https://www.googleapis.com/drive/v3/files/" + id + "?fields=trashed&supportsAllDrives=true"
}

// The wait test's own words and timings. The deadline is long enough that a
// Drive listing which takes a moment to show a new comment still lands inside
// it, and the interval is short enough that the second poll is the one that
// sees it. Both are passed to Wait rather than set on the command's constant:
// WaitOptions carries them, so this test drives its own pace without touching
// what the binary does.
const (
	waitComment  = "ai? live wait test"
	waitDeadline = 90 * time.Second
	waitInterval = 5 * time.Second

	// quietDeadline is the second wait, the one that must find nothing. It is
	// short because a document with one comment already reported has nothing to
	// wait for, and a long deadline would only make the test slow.
	quietDeadline = 15 * time.Second
)

// TestLiveWaitSeesANewComment is the live half of M4: a wait that is running
// when somebody writes a comment comes back with that comment, before its
// deadline, with a cursor the next call can use.
//
// It creates its own document in the test folder and trashes it, like the write
// test above. The comment it waits for is one it posts itself, through the same
// Drive route a person's comment arrives on, because there is nobody at a
// browser during an unattended run.
//
// The second half is the quiet case, and it is the half a wait can fail
// silently at: a cursor that did not really advance makes the same comment news
// on every poll for ever, and a wait that answers empty on the cursor it just
// handed back is what says the advance was real.
func TestLiveWaitSeesANewComment(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to create a document in the Drive test folder and wait for a comment in it; %s names another folder", liveVar, writeVar, folderVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}
	t.Logf("creating in folder %s", folder)

	p := guard.NewPolicy()
	p.AllowCreateIn(folder)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	// A second session on the same policy, for the goroutine that writes the
	// comment. The policy is mutex guarded and safe to share; a Session is not,
	// because it refreshes its own token in place, and a race there is not what
	// this test is asking about.
	writer, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the writing session could not be opened: %v", err)
	}
	ctx := context.Background()

	docID := createSubject(t, ctx, s, folder)

	// The baseline, exactly as the skill takes it: one listing, and the cursor
	// the run would have printed. A document created a moment ago has no
	// comments, so this is a nil cursor, and the wait below starts from the
	// beginning of a document with nothing in it.
	base := baselineCursor(t, ctx, s, docID)

	// The comment is written while the wait is running, one interval in, so the
	// first poll is the empty window and a later one carries the news. The
	// goroutine is the writer rather than the wait so that the wait's answer and
	// its error stay on the test's own goroutine.
	posted := make(chan string, 1)
	go func() {
		time.Sleep(waitInterval + time.Second)
		posted <- postComment(t, ctx, writer, docID, waitComment)
	}()

	started := time.Now()
	w, err := comments.Wait(ctx, base, comments.WaitOptions{
		Interval: waitInterval,
		Deadline: waitDeadline,
		Fetch:    poller(s, docID, base),
	})
	if err != nil {
		t.Fatalf("the wait failed after %d polls in %s: %v", w.Polls, time.Since(started).Round(time.Millisecond), err)
	}
	commentID := <-posted
	t.Logf("wait 1: %d polls, %s waited, %d threads, comment %s posted while it ran",
		w.Polls, w.Waited.Round(time.Millisecond), len(w.Threads), commentID)

	if w.Interrupted {
		t.Fatal("the wait reports it was interrupted, and nothing sent it a signal")
	}
	if len(w.Threads) != 1 {
		t.Fatalf("the wait came back with %d threads, and one comment was written while it ran", len(w.Threads))
	}
	got := w.Threads[0]
	if got.ID != commentID {
		t.Errorf("the wait came back with thread %q, and %q is the comment that was posted", got.ID, commentID)
	}
	if got.Content != waitComment {
		t.Errorf("the thread reads %q, and %q was written", got.Content, waitComment)
	}
	if got.Marker != "ai?" {
		t.Errorf("the thread carries marker %q, and %q opens with ai?", got.Marker, waitComment)
	}
	if w.Polls < 2 {
		t.Errorf("the wait answered on poll %d, and the comment was written after the first one", w.Polls)
	}
	if w.Waited >= waitDeadline {
		t.Errorf("the wait took %s, which is its whole deadline: it answered at the deadline rather than on the news", w.Waited)
	}
	if w.Cursor == nil {
		t.Fatal("the wait came back with news and no cursor, so the next call has nowhere to start")
	}
	if w.Cursor.String() == base.String() {
		t.Errorf("the cursor did not move: it is still %q, and a thread arrived", base.String())
	}
	// A comment created through Drive carries no anchor, so the Docs read places
	// no range on it. That is a fact about this comment and not a fault: the
	// live section of the skill says such a thread can be answered and cannot be
	// proposed into, and this is what one looks like.
	if got.Range != nil {
		t.Logf("the thread came back placed at %+v, which a Drive comment with no anchor usually is not", *got.Range)
	}
	t.Logf("thread %s: marker %s, unplaced %v, cursor %s", got.ID, got.Marker, w.Unplaced, w.Cursor.String())

	// The same document, the cursor the wait just handed back, and nothing
	// written this time. It must run out its deadline and come back empty.
	quiet := time.Now()
	q, err := comments.Wait(ctx, w.Cursor, comments.WaitOptions{
		Interval: waitInterval,
		Deadline: quietDeadline,
		Fetch:    poller(s, docID, w.Cursor),
	})
	if err != nil {
		t.Fatalf("the quiet wait failed after %d polls in %s: %v", q.Polls, time.Since(quiet).Round(time.Millisecond), err)
	}
	t.Logf("wait 2: %d polls, %s waited, %d threads", q.Polls, q.Waited.Round(time.Millisecond), len(q.Threads))
	if len(q.Threads) != 0 {
		t.Errorf("the quiet wait came back with %d threads, and the only comment in the document was already reported", len(q.Threads))
	}
	if q.Interrupted {
		t.Error("the quiet wait reports it was interrupted, and nothing sent it a signal")
	}
	if q.Cursor.String() != w.Cursor.String() {
		t.Errorf("the quiet wait moved the cursor from %q to %q, and it saw nothing", w.Cursor.String(), q.Cursor.String())
	}
}

// poller is the poll the command builds: the Docs read for the ranges, then the
// comment listing narrowed to the cursor. It is spelled here rather than
// imported because cmd/gdoc builds it inline, and what this test is asserting
// is Wait over the real two reads.
func poller(s *gapi.Session, docID string, since *comments.Cursor) comments.Poll {
	return func(ctx context.Context) (*docs.Document, []comments.RawComment, error) {
		d, err := docs.Fetch(ctx, s, docID)
		if err != nil {
			return nil, nil, err
		}
		raws, err := comments.Fetch(ctx, s, docID, since)
		if err != nil {
			return nil, nil, err
		}
		return d, raws, nil
	}
}

// baselineCursor is the first listing of a live session: no cursor, everything
// the document already carries, and the cursor the next call starts from.
func baselineCursor(t *testing.T, ctx context.Context, s *gapi.Session, docID string) *comments.Cursor {
	t.Helper()
	d, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		t.Fatalf("the baseline document read failed: %v", err)
	}
	raws, err := comments.Fetch(ctx, s, docID, nil)
	if err != nil {
		t.Fatalf("the baseline comment listing failed: %v", err)
	}
	threads, _ := comments.Threads(raws, d)
	if len(threads) != 0 {
		t.Fatalf("the document was created a moment ago and already carries %d threads", len(threads))
	}
	cursor := comments.NextCursor(nil, threads)
	t.Logf("baseline: %d threads, cursor %q", len(threads), cursor.String())
	return cursor
}

// postComment writes one comment into the document, the way a person's comment
// arrives: Drive's comments.create, which the guard carries as a POST on the
// comment collection. It is unanchored, because an anchor is an opaque string
// gdoc builds nowhere.
//
// It runs on its own goroutine, so it reports through t.Errorf and hands back
// an empty id rather than calling t.Fatalf, which may only be called from the
// goroutine running the test.
func postComment(t *testing.T, ctx context.Context, s *gapi.Session, docID, content string) string {
	t.Helper()
	var created struct {
		ID string `json:"id"`
	}
	if err := s.PostJSON(ctx, commentURL(docID), map[string]any{"content": content}, &created); err != nil {
		t.Errorf("the comment could not be written into %q: %v", docID, err)
		return ""
	}
	if created.ID == "" {
		t.Errorf("the comment write into %q answered with no comment id", docID)
	}
	return created.ID
}

// commentURL is Drive's comments.create. `fields` is not optional here: Drive
// refuses a comment write that does not say what it wants back.
func commentURL(id string) string {
	q := url.Values{"fields": {"id,createdTime,modifiedTime"}}
	return "https://www.googleapis.com/drive/v3/files/" + id + "/comments?" + q.Encode()
}
