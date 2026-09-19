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
	// afterBatch is a failure raised once the answer has been decoded, which is
	// the one shape that reaches Run with fields in hand: valid JSON of the
	// wrong shape, where encoding/json saves the type error and keeps going.
	// failAt cannot model it, because it answers before the fixture is read.
	afterBatch error
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
	if err := json.Unmarshal(f.batch, into); err != nil {
		return err
	}
	return f.afterBatch
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

// note is the note's entry for this document, carrying the proposals named, so
// a test says in one line whose suggestion this is.
func note(ids ...string) *frontmatter.Entry {
	e := &frontmatter.Entry{ID: testDocID}
	for _, id := range ids {
		e.Proposals = append(e.Proposals, frontmatter.Proposal{
			ID:        id,
			CommentID: "AAAC",
			At:        time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC),
			Quoted:    "reviewed annually",
		})
	}
	return e
}

// noteBlock is the whole block around that entry, for the tests that write one.
func noteBlock(ids ...string) *frontmatter.Block {
	return &frontmatter.Block{Schema: frontmatter.Schema, Documents: []frontmatter.Entry{*note(ids...)}}
}

// TestRunRefusesASuggestionTheNoteDoesNotName is the whole permission model.
// Nothing in a suggestion id says who wrote it, so the note is the only record
// gdoc has of its own work, and a suggestion missing from it is somebody
// else's. The refusal has to land before the read, not after it.
func TestRunRefusesASuggestionTheNoteDoesNotName(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "rejected.json")

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
	f := script(t, "pending.json", "gone.json", "rejected.json")

	_, err := Run(context.Background(), f, testDocID, testSuggestion, nil)

	if err == nil {
		t.Fatal("a suggestion was withdrawn with no note to name it")
	}
	if len(f.calls) != 0 {
		t.Errorf("the fake saw %d requests without a note", len(f.calls))
	}
}

func TestRunRejectsTheSuggestionAndVerifiesItIsGone(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "rejected.json")

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
	if len(res.RejectedSuggestionIDs) != 1 || res.RejectedSuggestionIDs[0] != testSuggestion {
		t.Errorf("RejectedSuggestionIDs = %v", res.RejectedSuggestionIDs)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v on a run where everything held", res.Warnings)
	}
}

// TestRunSendsOneSuggestModeRejectNamingTheId is what actually goes on the
// wire. The request is the exact shape the guard's grant opens: rejectSuggestion
// carrying suggestionId and nothing beside it, and the write mode is what the
// guard requires of every batchUpdate on a handed-in document.
func TestRunSendsOneSuggestModeRejectNamingTheId(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "rejected.json")

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
		t.Fatalf("requests = %v, want one rejectSuggestion", body["requests"])
	}
	req := reqs[0].(map[string]any)
	rej, ok := req["rejectSuggestion"].(map[string]any)
	if !ok || len(req) != 1 {
		t.Fatalf("request = %v, want exactly one rejectSuggestion", req)
	}
	if len(rej) != 1 || rej["suggestionId"] != testSuggestion {
		t.Errorf("rejectSuggestion = %v, want exactly {\"suggestionId\":%q}", rej, testSuggestion)
	}
}

// TestRunReadsTheDocumentItselfBeforeWriting is the rule the whole design
// rests on: whether the suggestion is still pending is answered by a read made a
// moment earlier, never assumed from the note.
func TestRunReadsTheDocumentItselfBeforeWriting(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "rejected.json")

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

// TestRunTakesBackAProposalOnlyItsDeletionHalfStillCarries is the recovery
// path for what the delete-based withdraw left behind before 2026-09-07: the
// inserted words gone, the quoted words still suggested-deleted under the id.
// Either side counts as pending, so the reject goes out and takes it back.
func TestRunTakesBackAProposalOnlyItsDeletionHalfStillCarries(t *testing.T) {
	f := script(t, "half.json", "gone.json", "rejected.json")

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Verified {
		t.Errorf("Verified = false, warnings = %v", res.Warnings)
	}
	if len(f.posts()) != 1 {
		t.Errorf("writes = %d, want the one reject", len(f.posts()))
	}
}

// TestRunWithdrawsFromADocumentWithMoreThanOneTab: a reject names an id, not a
// range, so the tab the suggestion sits in does not matter. The delete-based
// withdraw refused these documents; this one has no reason to.
func TestRunWithdrawsFromADocumentWithMoreThanOneTab(t *testing.T) {
	f := script(t, "two-tabs.json", "gone.json", "rejected.json")

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Verified {
		t.Errorf("Verified = false, warnings = %v", res.Warnings)
	}
}

// TestRunIsUnverifiedWhenTheAnswerNamesNoRejectedSuggestion is the first of the
// two checks. A 200 that mentions no rejected id is Docs having done something
// other than what was asked; updatedSummarySuggestionIds, which the half
// retraction used to answer, is not it.
func TestRunIsUnverifiedWhenTheAnswerNamesNoRejectedSuggestion(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "silent.json")

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true on an answer that named no rejected suggestion")
	}
	if len(res.RejectedSuggestionIDs) != 0 {
		t.Errorf("RejectedSuggestionIDs = %v, and updatedSummarySuggestionIds is not a rejection", res.RejectedSuggestionIDs)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("no warning said why the withdrawal could not be confirmed")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), testSuggestion) {
		t.Errorf("warnings = %v, and they should name the suggestion", res.Warnings)
	}
}

// TestRunIsUnverifiedWhenTheReadBackStillCarriesTheId is the second check, and
// the one the answer cannot give. The write agreeing with itself is not the
// question; whether the suggestion is still in the document is.
func TestRunIsUnverifiedWhenTheReadBackStillCarriesTheId(t *testing.T) {
	f := script(t, "pending.json", "pending.json", "rejected.json")

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

// TestRunIsUnverifiedWhenTheReadBackStillCarriesTheDeletionHalf is the same
// check on the shape that made this package change: the words came out and the
// suggested deletion stayed. Half gone is not gone.
func TestRunIsUnverifiedWhenTheReadBackStillCarriesTheDeletionHalf(t *testing.T) {
	f := script(t, "pending.json", "half.json", "rejected.json")

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true while the quoted words are still suggested-deleted under the id")
	}
}

// TestRunReportsAReadBackItCouldNotMake is the same rule as everywhere else in
// this milestone: after the write nothing fails, because the change has already
// happened and a caller told the run failed is a caller that writes it again.
func TestRunReportsAReadBackItCouldNotMake(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "rejected.json")
	f.failAt[2] = errors.New("Drive said no")

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("the run failed after the write had gone out: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true on a read-back that never came back")
	}
	if len(res.RejectedSuggestionIDs) != 1 {
		t.Errorf("RejectedSuggestionIDs = %v, and the answer's ids are still a fact", res.RejectedSuggestionIDs)
	}
	if len(res.Warnings) == 0 {
		t.Error("no warning said the read-back could not be made")
	}
}

// TestRunReportsARejectWhoseAnswerCouldNotBeRead is the difference between a
// write that never left and a write Docs took. The session marks the second
// kind, and reporting it as a failure would send the skill back to reject the
// same suggestion again, over a document that has already changed.
//
// Here nothing decoded, which is the usual shape of this path, so
// rejectedSuggestionIds carries nothing, Verified stays false and the note keeps
// the entry. It is the answer for this answer rather than an invariant of the
// path: the case below is the same failure over a body that did decode.
func TestRunReportsARejectWhoseAnswerCouldNotBeRead(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "rejected.json")
	f.failAt[1] = acceptedError{errors.New("the answer could not be read: unexpected EOF")}

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("a reject the server accepted was reported as never sent: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true on a reject whose answer was never read")
	}
	if len(f.calls) != 3 {
		t.Errorf("calls = %d, and the read-back still runs on a write that landed", len(f.calls))
	}
	joined := strings.Join(res.Warnings, " ")
	if !strings.Contains(joined, "accepted by Docs") {
		t.Errorf("warnings = %v, and one should say the reject was accepted", res.Warnings)
	}
	// The rejectedSuggestionIds warning names an answer that did something
	// else, and an answer nobody could read says nothing about one. Raising it
	// here would report a second problem the server never showed.
	if strings.Contains(joined, "rejectedSuggestionIds") {
		t.Errorf("warnings = %v, and none may read a wrong answer out of an answer that was lost", res.Warnings)
	}
}

// TestRunKeepsTheIdsDecodedFromAnAnswerThatFailed is the other half of the
// path above: valid JSON of the wrong shape, where the list the server sent is
// in hand even though the read failed. Those ids are the server's own word
// about what it rejected, so throwing them away would report a withdrawal both
// facts confirm as unverified, and the warning would tell somebody to take an
// entry out of the note that this run removes itself.
func TestRunKeepsTheIdsDecodedFromAnAnswerThatFailed(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "rejected.json")
	f.afterBatch = acceptedError{errors.New("the answer could not be read: json: cannot unmarshal number into Go struct field")}

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("a reject the server accepted was reported as never sent: %v", err)
	}
	if !res.Verified {
		t.Errorf("Verified = false, and both facts held: %+v", res)
	}
	joined := strings.Join(res.Warnings, " ")
	if !strings.Contains(joined, "accepted by Docs") {
		t.Errorf("warnings = %v, and one should say the reject was accepted", res.Warnings)
	}
	if strings.Contains(joined, "by hand") {
		t.Errorf("warnings = %v, and none may say to take out an entry this run removes itself", res.Warnings)
	}
}

// TestRunReadsAnotherSuggestionOutOfIdsThatDecodedOnAFailedAnswer is the third
// shape of that path: the list decoded and it names somebody else's suggestion.
// The warning is gated on the decoded list rather than on the path, the way
// propose gates on a commentUpdateState it was given, so the alarm fires here.
// Asking about the path instead threw away the one fact the server did send.
func TestRunReadsAnotherSuggestionOutOfIdsThatDecodedOnAFailedAnswer(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "rejected-other.json")
	f.afterBatch = acceptedError{errors.New("the answer could not be read: json: cannot unmarshal number into Go struct field")}

	res, err := Run(context.Background(), f, testDocID, testSuggestion, note(testSuggestion))

	if err != nil {
		t.Fatalf("a reject the server accepted was reported as never sent: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true on an answer that rejected another suggestion")
	}
	joined := strings.Join(res.Warnings, " ")
	if !strings.Contains(joined, "rejectedSuggestionIds") {
		t.Errorf("warnings = %v, and one should name the list that rejected nothing of gdoc's", res.Warnings)
	}
	if !strings.Contains(joined, "accepted by Docs") {
		t.Errorf("warnings = %v, and one should say the reject was accepted", res.Warnings)
	}
}

// acceptedError is what the session hands back for a failure raised after the
// server accepted the request. The writer packages ask by behaviour, so the fake
// answers by behaviour too.
type acceptedError struct{ error }

func (acceptedError) Sent() bool { return true }

func TestRunFailsWhenTheWriteDoesAndSendsNothingElse(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "rejected.json")
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
// disagreeing. Rejecting nothing would be reported as a withdrawal, and the
// entry would leave the note with nothing to show for it.
func TestRunRefusesASuggestionTheDocumentDoesNotCarry(t *testing.T) {
	f := script(t, "gone.json", "gone.json", "rejected.json")

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

func TestRunFailsWhenTheDocumentCannotBeRead(t *testing.T) {
	f := script(t, "pending.json", "gone.json", "rejected.json")
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
	b := noteBlock("suggest.one", testSuggestion, "suggest.three")

	next := Forget(b, testDocID, testSuggestion)

	kept := next.Documents[0].Proposals
	if len(kept) != 2 {
		t.Fatalf("proposals = %d, want the other two", len(kept))
	}
	if kept[0].ID != "suggest.one" || kept[1].ID != "suggest.three" {
		t.Errorf("proposals = %v, and the order should be the note's own", kept)
	}
	if next.Documents[0].ID != b.Documents[0].ID || next.Schema != b.Schema {
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
	b := noteBlock("suggest.one", testSuggestion)

	next := Forget(b, testDocID, testSuggestion)

	if len(b.Documents[0].Proposals) != 2 {
		t.Fatalf("the input block now has %d proposals; Forget edited it in place", len(b.Documents[0].Proposals))
	}
	if b.Documents[0].Proposals[1].ID != testSuggestion {
		t.Errorf("the input block's entries moved: %v", b.Documents[0].Proposals)
	}
	next.Documents[0].Proposals = append(next.Documents[0].Proposals, frontmatter.Proposal{ID: "suggest.four"})
	if len(b.Documents[0].Proposals) != 2 {
		t.Error("the two blocks share the proposals slice, so writing one changes the other")
	}
	next.Documents = append(next.Documents, frontmatter.Entry{ID: "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz"})
	if len(b.Documents) != 1 {
		t.Error("the two blocks share the documents slice, so writing one changes the other")
	}
}

// A document the block does not name changes nothing. The caller checked which
// document the run was of before anything was written, and inventing an entry
// here would record a document the note was never paired with.
func TestForgetLeavesADocumentTheBlockDoesNotName(t *testing.T) {
	b := noteBlock("suggest.one", testSuggestion)

	next := Forget(b, "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz", testSuggestion)

	if len(next.Documents) != 1 || len(next.Documents[0].Proposals) != 2 {
		t.Errorf("block = %+v, want the note unchanged", next)
	}
}

// The other document's proposals stay where they are, because a suggestion id
// under one document says nothing about another.
func TestForgetTouchesOnlyTheDocumentItWasOf(t *testing.T) {
	b := noteBlock("suggest.one", testSuggestion)
	b.Documents = append(b.Documents, frontmatter.Entry{
		ID:        "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz",
		Proposals: []frontmatter.Proposal{{ID: testSuggestion, CommentID: "AAAD", At: time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC)}},
	})

	next := Forget(b, testDocID, testSuggestion)

	if len(next.Documents[0].Proposals) != 1 {
		t.Errorf("documents[0].proposals = %+v, want the one that was not withdrawn", next.Documents[0].Proposals)
	}
	if len(next.Documents[1].Proposals) != 1 {
		t.Errorf("the other document's proposal went with it: %+v", next.Documents[1].Proposals)
	}
}

func TestForgetLeavesABlockThatNeverNamedTheSuggestion(t *testing.T) {
	b := noteBlock("suggest.one")

	next := Forget(b, testDocID, testSuggestion)

	kept := next.Documents[0].Proposals
	if len(kept) != 1 || kept[0].ID != "suggest.one" {
		t.Errorf("proposals = %v, want the note unchanged", kept)
	}
}

func TestForgetOnANilBlockIsNil(t *testing.T) {
	if Forget(nil, testDocID, testSuggestion) != nil {
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

// Batch is stated on its own as well as through Run, because it is the exact
// shape the guard's grant opens, and a test that has to run the whole command
// to say so is a test that says it faintly.
func TestBatchIsOneSuggestModeRejectNamingTheId(t *testing.T) {
	var body map[string]any
	if err := json.Unmarshal(Batch(testSuggestion), &body); err != nil {
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
		t.Fatalf("requests = %v, want one rejectSuggestion", body["requests"])
	}
	req := reqs[0].(map[string]any)
	rej, ok := req["rejectSuggestion"].(map[string]any)
	if !ok || len(req) != 1 {
		t.Fatalf("request = %v, want exactly one rejectSuggestion", req)
	}
	// suggestionId and nothing else: a second field is one the guard refuses,
	// because nobody here has read what it does.
	if len(rej) != 1 || rej["suggestionId"] != testSuggestion {
		t.Errorf("rejectSuggestion = %v, want exactly {\"suggestionId\":%q}", rej, testSuggestion)
	}
}
