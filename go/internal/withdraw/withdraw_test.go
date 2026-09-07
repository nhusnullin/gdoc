package withdraw

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/frontmatter"
)

const (
	testDocID       = "1WiThDrAw00000000000000000000000000000000"
	testSuggestion  = "suggest.abc"
	otherSuggestion = "suggest.somebody-else"
)

// call is one request the fake was asked to make, so a test can say what went
// out as well as what came back.
type call struct {
	method string
	url    string
	body   json.RawMessage
}

// fakeSession is Google as far as this package is concerned: the document read,
// answered from a queue because Run reads twice, and the one write.
//
// The queue is the point. A fake that answered the same bytes to both reads
// would verify the document as it was before the write, which is the failure
// this package exists to catch.
//
// Nothing here names net/http: the rooms that may are the ones the boundary
// test lists, and this is not one of them.
type fakeSession struct {
	calls  []call
	reads  [][]byte
	batch  []byte
	failAt map[int]error
}

func (f *fakeSession) record(method, rawURL string, body any) error {
	raw := json.RawMessage(nil)
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		raw = b
	}
	f.calls = append(f.calls, call{method: method, url: rawURL, body: raw})
	return f.failAt[len(f.calls)-1]
}

func (f *fakeSession) GetJSON(_ context.Context, rawURL string, into any) error {
	if err := f.record("GET", rawURL, nil); err != nil {
		return err
	}
	if len(f.reads) == 0 {
		return errors.New("the fake was asked for one read more than the test scripted")
	}
	answer := f.reads[0]
	f.reads = f.reads[1:]
	if into == nil {
		return nil
	}
	return json.Unmarshal(answer, into)
}

func (f *fakeSession) PostJSON(_ context.Context, rawURL string, body any, into any) error {
	if err := f.record("POST", rawURL, body); err != nil {
		return err
	}
	if into == nil {
		return nil
	}
	return json.Unmarshal(f.batch, into)
}

func (f *fakeSession) posts() []call {
	var out []call
	for _, c := range f.calls {
		if c.method == "POST" {
			out = append(out, c)
		}
	}
	return out
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// script is a Google that answers the two reads and the write, each from a
// named fixture.
func script(t *testing.T, before, after, batch string) *fakeSession {
	t.Helper()
	return &fakeSession{
		reads:  [][]byte{fixture(t, before), fixture(t, after)},
		batch:  fixture(t, batch),
		failAt: map[int]error{},
	}
}

// note is a front-matter block carrying the proposals named, so a test says in
// one line whose suggestion this is.
func note(ids ...string) *frontmatter.Block {
	b := &frontmatter.Block{Schema: frontmatter.Schema, DocumentID: testDocID}
	for _, id := range ids {
		b.Proposals = append(b.Proposals, frontmatter.Proposal{
			ID:        id,
			CommentID: "AAAC",
			At:        time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC),
			Quoted:    "reviewed annually",
		})
	}
	return b
}

// TestRunRefusesASuggestionTheNoteDoesNotName is the whole permission model.
// Nothing in a suggestion id says who wrote it, so the note is the only record
// gdoc has of its own work, and a suggestion missing from it is somebody
// else's. The refusal has to land before the read, not after it.
func TestRunRefusesASuggestionTheNoteDoesNotName(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "deleted.json")

	_, err := Run(context.Background(), f, testDocID, otherSuggestion, note(testSuggestion))

	if err == nil {
		t.Fatal("a suggestion the note does not name was withdrawn")
	}
	if !strings.Contains(err.Error(), otherSuggestion) {
		t.Errorf("error = %q, and it should name the suggestion it refused", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("the fake saw %d requests, and a refusal happens before any of them", len(f.calls))
	}
}

func TestRunRefusesWhenTheNoteIsMissing(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "deleted.json")

	_, err := Run(context.Background(), f, testDocID, testSuggestion, nil)

	if err == nil {
		t.Fatal("a suggestion was withdrawn with no note to name it")
	}
	if len(f.calls) != 0 {
		t.Errorf("the fake saw %d requests without a note", len(f.calls))
	}
}

func TestRunDeletesTheSuggestedSpanAndVerifiesItIsGone(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "deleted.json")

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Verified {
		t.Errorf("Verified = false, warnings = %v", res.Warnings)
	}
	if res.SuggestionID != testSuggestion {
		t.Errorf("SuggestionID = %q", res.SuggestionID)
	}
	if len(res.DeletedSuggestionIDs) != 1 || res.DeletedSuggestionIDs[0] != testSuggestion {
		t.Errorf("DeletedSuggestionIDs = %v", res.DeletedSuggestionIDs)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v on a run where everything held", res.Warnings)
	}
}

// TestRunSendsOneSuggestModeDeleteOverTheInsertion is what actually goes on the
// wire. The span is the runs carrying the id, and the write mode is what keeps
// the retraction itself a suggestion rather than an edit.
func TestRunSendsOneSuggestModeDeleteOverTheInsertion(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "deleted.json")

	if _, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	posts := f.posts()
	if len(posts) != 1 {
		t.Fatalf("the fake saw %d writes, and a withdrawal is one batchUpdate", len(posts))
	}
	if posts[0].url != BatchURL(testDocID) {
		t.Errorf("write went to %q, want %q", posts[0].url, BatchURL(testDocID))
	}

	var body map[string]any
	if err := json.Unmarshal(posts[0].body, &body); err != nil {
		t.Fatalf("the batch body is not JSON: %v", err)
	}
	wc, ok := body["writeControl"].(map[string]any)
	if !ok || len(wc) != 1 || wc["writeMode"] != "SUGGEST" {
		t.Fatalf("writeControl = %v, want exactly {\"writeMode\":\"SUGGEST\"}", body["writeControl"])
	}
	reqs, ok := body["requests"].([]any)
	if !ok || len(reqs) != 1 {
		t.Fatalf("requests = %v, want one deleteContentRange", body["requests"])
	}
	rng := reqs[0].(map[string]any)["deleteContentRange"].(map[string]any)["range"].(map[string]any)
	if rng["startIndex"] != float64(26) || rng["endIndex"] != float64(51) {
		t.Errorf("deleteContentRange range = %v, want the inserted span 26..51", rng)
	}
}

// TestRunJoinsTheRunsDocsCutTheInsertInto is the same hazard the read-back has:
// Docs splits one insert into as many runs as it likes, and the span to delete
// is all of them, not the first.
func TestRunJoinsTheRunsDocsCutTheInsertInto(t *testing.T) {
	f := script(t, "split.json", "gone.json", "deleted.json")

	if _, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(f.posts()[0].body, &body); err != nil {
		t.Fatal(err)
	}
	rng := body["requests"].([]any)[0].(map[string]any)["deleteContentRange"].(map[string]any)["range"].(map[string]any)
	if rng["startIndex"] != float64(26) || rng["endIndex"] != float64(51) {
		t.Errorf("range = %v, want the two runs joined into 26..51", rng)
	}
}

// TestRunReadsTheDocumentItselfBeforeWriting is the rule the whole design
// rests on: the index is computed from a read made a moment earlier, never
// handed in from outside.
func TestRunReadsTheDocumentItselfBeforeWriting(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "deleted.json")

	if _, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(f.calls) != 3 {
		t.Fatalf("calls = %d, want a read, a write and a read-back", len(f.calls))
	}
	if f.calls[0].method != "GET" || f.calls[0].url != docs.URL(testDocID) {
		t.Errorf("call 1 = %s %s, want the inline read", f.calls[0].method, f.calls[0].url)
	}
	if f.calls[1].method != "POST" {
		t.Errorf("call 2 = %s, want the write", f.calls[1].method)
	}
	if f.calls[2].method != "GET" || f.calls[2].url != docs.URL(testDocID) {
		t.Errorf("call 3 = %s %s, want the read-back", f.calls[2].method, f.calls[2].url)
	}
}

// TestRunIsUnverifiedWhenTheAnswerNamesNoDeletedSuggestion is the first of the
// two checks. A 200 that mentions no suggestion id is Docs having deleted text
// without retracting anything, which is the shape of a silent direct edit.
func TestRunIsUnverifiedWhenTheAnswerNamesNoDeletedSuggestion(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "silent.json")

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true on an answer that named no deleted suggestion")
	}
	if len(res.Warnings) == 0 {
		t.Fatal("no warning said why the withdrawal could not be confirmed")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), testSuggestion) {
		t.Errorf("warnings = %v, and they should name the suggestion", res.Warnings)
	}
}

// TestRunReadsTheDeletedIdsFromAPerRequestReply is the same field one level
// down. The shape is measured rather than documented, so both places are read:
// reporting the withdrawal as unconfirmed because the id arrived nested would
// send somebody looking for a suggestion that is already gone.
func TestRunReadsTheDeletedIdsFromAPerRequestReply(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "deleted-in-reply.json")

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Verified {
		t.Errorf("Verified = false, warnings = %v", res.Warnings)
	}
}

// TestRunIsUnverifiedWhenTheReadBackStillCarriesTheId is the second check, and
// the one the answer cannot give. The write agreeing with itself is not the
// question; whether the suggestion is still in the document is.
func TestRunIsUnverifiedWhenTheReadBackStillCarriesTheId(t *testing.T) {
	f := script(t, "pending.json", "pending.json", "deleted.json")

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true while the read-back still carries the suggestion")
	}
	if len(res.Warnings) == 0 {
		t.Error("no warning said the suggestion is still in the document")
	}
}

// TestRunReportsAReadBackItCouldNotMake is the same rule as everywhere else in
// this milestone: after the write nothing fails, because the change has already
// happened and a caller told the run failed is a caller that writes it again.
func TestRunReportsAReadBackItCouldNotMake(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "deleted.json")
	f.failAt[2] = errors.New("Drive said no")

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("the run failed after the write had gone out: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true on a read-back that never came back")
	}
	if len(res.DeletedSuggestionIDs) != 1 {
		t.Errorf("DeletedSuggestionIDs = %v, and the answer's ids are still a fact", res.DeletedSuggestionIDs)
	}
	if len(res.Warnings) == 0 {
		t.Error("no warning said the read-back could not be made")
	}
}

func TestRunFailsWhenTheWriteDoesAndSendsNothingElse(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "deleted.json")
	f.failAt[1] = errors.New("the guard refused it")

	_, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err == nil {
		t.Fatal("a write that failed was reported as a withdrawal")
	}
	if len(f.calls) != 2 {
		t.Errorf("calls = %d, and a failed write is the last thing the run does", len(f.calls))
	}
}

// TestRunRefusesASuggestionTheDocumentDoesNotCarry is the note and the document
// disagreeing. Deleting nothing would be reported as a withdrawal, and the
// entry would leave the note with the suggestion still pending.
func TestRunRefusesASuggestionTheDocumentDoesNotCarry(t *testing.T) {
	f := script(t, "gone.json", "gone.json", "deleted.json")

	_, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err == nil {
		t.Fatal("a suggestion the document does not carry was withdrawn")
	}
	if !strings.Contains(err.Error(), testSuggestion) {
		t.Errorf("error = %q, and it should name the suggestion", err)
	}
	if len(f.posts()) != 0 {
		t.Error("a write went out for a suggestion that is not in the document")
	}
}

// TestRunRefusesRunsThatAreNotOneSpan is the fail-closed direction. Deleting
// from the first run to the last would take somebody else's words in between
// with it, and those words are not gdoc's to touch.
func TestRunRefusesRunsThatAreNotOneSpan(t *testing.T) {
	f := script(t, "apart.json", "gone.json", "deleted.json")

	_, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err == nil {
		t.Fatal("two spans with other words between them were deleted as one")
	}
	if len(f.posts()) != 0 {
		t.Error("a write went out over text the suggestion does not cover")
	}
}

func TestRunRefusesADocumentWithMoreThanOneTab(t *testing.T) {
	f := script(t, "two-tabs.json", "gone.json", "deleted.json")

	_, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err == nil {
		t.Fatal("a write was placed in a document with more than one tab")
	}
	if !strings.Contains(err.Error(), "tab") {
		t.Errorf("error = %q, and it should say which shape it refused", err)
	}
	if len(f.posts()) != 0 {
		t.Error("a write went out into a document with more than one tab")
	}
}

func TestRunFailsWhenTheDocumentCannotBeRead(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "deleted.json")
	f.failAt[0] = errors.New("Docs said no")

	_, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err == nil {
		t.Fatal("a withdrawal was reported on a document that could not be read")
	}
	if len(f.posts()) != 0 {
		t.Error("a write went out after the read had failed")
	}
}

// TestForgetLeavesTheOtherProposalsInPlace is the provenance half. Forgetting
// one entry must not lose the others, because each of them is the permission to
// withdraw one more suggestion.
func TestForgetLeavesTheOtherProposalsInPlace(t *testing.T) {
	b := note("suggest.one", testSuggestion, "suggest.three")

	next := Forget(b, testSuggestion)

	if len(next.Proposals) != 2 {
		t.Fatalf("proposals = %d, want the other two", len(next.Proposals))
	}
	if next.Proposals[0].ID != "suggest.one" || next.Proposals[1].ID != "suggest.three" {
		t.Errorf("proposals = %v, and the order should be the note's own", next.Proposals)
	}
	if next.DocumentID != b.DocumentID || next.Schema != b.Schema {
		t.Errorf("the rest of the block changed: %+v", next)
	}
	if err := next.Validate(); err != nil {
		t.Errorf("the block Forget returned does not validate: %v", err)
	}
}

// TestForgetDoesNotMutateItsInput is the house rule. The caller writes the note
// only when the withdrawal held, so a Forget that edited the block in place
// would drop the provenance of a suggestion that is still in the document.
func TestForgetDoesNotMutateItsInput(t *testing.T) {
	b := note("suggest.one", testSuggestion)

	next := Forget(b, testSuggestion)

	if len(b.Proposals) != 2 {
		t.Fatalf("the input block now has %d proposals; Forget edited it in place", len(b.Proposals))
	}
	if b.Proposals[1].ID != testSuggestion {
		t.Errorf("the input block's entries moved: %v", b.Proposals)
	}
	next.Proposals = append(next.Proposals, frontmatter.Proposal{ID: "suggest.four"})
	if len(b.Proposals) != 2 {
		t.Error("the two blocks share the proposals slice, so writing one changes the other")
	}
}

func TestForgetLeavesABlockThatNeverNamedTheSuggestion(t *testing.T) {
	b := note("suggest.one")

	next := Forget(b, testSuggestion)

	if len(next.Proposals) != 1 || next.Proposals[0].ID != "suggest.one" {
		t.Errorf("proposals = %v, want the note unchanged", next.Proposals)
	}
}

func TestForgetOnANilBlockIsNil(t *testing.T) {
	if Forget(nil, testSuggestion) != nil {
		t.Error("Forget invented a block out of nothing")
	}
}

func TestMineReadsTheNoteAndNothingElse(t *testing.T) {
	if !Mine(note("suggest.one", testSuggestion), testSuggestion) {
		t.Error("a suggestion the note names was called somebody else's")
	}
	if Mine(note("suggest.one"), testSuggestion) {
		t.Error("a suggestion the note does not name was called gdoc's own")
	}
	if Mine(nil, testSuggestion) {
		t.Error("a suggestion was called gdoc's own with no note to say so")
	}
}

// Span and Batch are the two halves of the write, and Run is tested over both
// together. They are stated here on their own as well, because between them
// they decide which characters leave a document Nail is reading, and a test
// that has to run the whole command to say so is a test that says it faintly.

// The runs Docs cut one insert into are adjacent, so the span is the first
// run's start to the last run's end. The deletion side beside them is the
// author's own words, still written, and it is not in the span.
func TestSpanIsTheWholeInsertionAndNothingBesideIt(t *testing.T) {
	d, err := docs.Parse(fixture(t, "split.json"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := Span(d, testSuggestion)
	if err != nil {
		t.Fatalf("Span(): %v", err)
	}
	if r.Start != 26 || r.End != 51 {
		t.Errorf("Span() = %d..%d, want the two runs joined into 26..51", r.Start, r.End)
	}
	if r.Tab != d.Tabs[0].ID {
		t.Errorf("Span() names tab %q, want %q", r.Tab, d.Tabs[0].ID)
	}
}

// A suggestion the document does not carry is a refusal naming the id, not an
// empty range. A zero range would delete from the top of the document.
func TestSpanRefusesASuggestionThatIsNotPending(t *testing.T) {
	d, err := docs.Parse(fixture(t, "gone.json"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := Span(d, testSuggestion)
	if err == nil {
		t.Fatalf("Span() = %+v, want a refusal naming the suggestion", r)
	}
	if !strings.Contains(err.Error(), testSuggestion) {
		t.Errorf("Span() error = %v, want the suggestion id named in it", err)
	}
}

// Batch is one deleteContentRange over exactly that span, in SUGGEST mode. The
// mode is what the guard reads before it carries the request, and the range is
// what Docs acts on, so both are stated as literals.
func TestBatchIsOneSuggestModeDeleteOverThatSpan(t *testing.T) {
	var body map[string]any
	if err := json.Unmarshal(Batch(docs.Range{Tab: "t.0", Start: 26, End: 51}), &body); err != nil {
		t.Fatalf("Batch() is not JSON: %v", err)
	}
	if len(body) != 2 {
		t.Errorf("the batch carries %v, want requests and writeControl and nothing else", body)
	}
	wc, ok := body["writeControl"].(map[string]any)
	if !ok || len(wc) != 1 || wc["writeMode"] != "SUGGEST" {
		t.Fatalf("writeControl = %v, want exactly {\"writeMode\":\"SUGGEST\"}", body["writeControl"])
	}
	reqs, ok := body["requests"].([]any)
	if !ok || len(reqs) != 1 {
		t.Fatalf("requests = %v, want one deleteContentRange", body["requests"])
	}
	del, ok := reqs[0].(map[string]any)["deleteContentRange"].(map[string]any)
	if !ok {
		t.Fatalf("request = %v, want a deleteContentRange", reqs[0])
	}
	rng, ok := del["range"].(map[string]any)
	if !ok {
		t.Fatalf("deleteContentRange = %v, want a range", del)
	}
	if rng["startIndex"] != float64(26) || rng["endIndex"] != float64(51) {
		t.Errorf("range = %v, want 26..51", rng)
	}
	// No tabId, on purpose. Run refuses a document with more than one tab, so
	// the range is unambiguous without one, and a milestone that teaches a
	// write to name a tab is the one that adds it here.
	if len(rng) != 2 {
		t.Errorf("range = %v, want startIndex and endIndex and nothing else", rng)
	}
}
