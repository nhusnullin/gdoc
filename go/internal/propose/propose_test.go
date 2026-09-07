package propose

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"gdoc/internal/frontmatter"
	"gdoc/internal/guard"
)

const testDocID = "1PrOpOsE000000000000000000000000000000000"

var testProposal = Proposal{
	Quoted:      "reviewed annually",
	Replacement: "reviewed every six months",
	Why:         "the policy says twice a year",
}

func decodeBatch(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("the batch body is not JSON: %v", err)
	}
	return out
}

func TestBatchSendsThreeRequestsInOrderUnderSuggestMode(t *testing.T) {
	body := decodeBatch(t, Batch(span(26, 43), testProposal))

	wc, ok := body["writeControl"].(map[string]any)
	if !ok || len(wc) != 1 || wc["writeMode"] != "SUGGEST" {
		t.Fatalf("writeControl = %v, want exactly {\"writeMode\":\"SUGGEST\"}", body["writeControl"])
	}
	reqs, ok := body["requests"].([]any)
	if !ok || len(reqs) != 3 {
		t.Fatalf("requests = %v, want three", body["requests"])
	}

	del := reqs[0].(map[string]any)["deleteContentRange"].(map[string]any)["range"].(map[string]any)
	if del["startIndex"] != float64(26) || del["endIndex"] != float64(43) {
		t.Errorf("deleteContentRange range = %v, want 26..43", del)
	}
	ins := reqs[1].(map[string]any)["insertText"].(map[string]any)
	if ins["text"] != testProposal.Replacement {
		t.Errorf("insertText text = %v", ins["text"])
	}
	if loc := ins["location"].(map[string]any); loc["index"] != float64(26) {
		t.Errorf("insertText index = %v, want the start of the span the delete covered", loc["index"])
	}
	com := reqs[2].(map[string]any)["insertComment"].(map[string]any)
	if want := "🤖 " + testProposal.Why; com["content"] != want {
		t.Errorf("insertComment content = %v, want %q", com["content"], want)
	}
	rng := com["range"].(map[string]any)
	if rng["startIndex"] != float64(26) || rng["endIndex"] != float64(26+25) {
		t.Errorf("insertComment range = %v, want the span the replacement made", rng)
	}
	if _, ok := com["assigneeEmailAddress"]; ok {
		t.Error("assigneeEmailAddress is present on a proposal that named no assignee")
	}
}

// TestBatchCountsTheReplacementInUTF16CodeUnits is the same hazard FindSpan
// has, on the other side: the comment is anchored on the span the insert made,
// and that span is as long as the replacement is in the units Docs counts.
func TestBatchCountsTheReplacementInUTF16CodeUnits(t *testing.T) {
	p := Proposal{Quoted: "annually", Replacement: "twice a year 🤖", Why: "so"}
	body := decodeBatch(t, Batch(span(26, 34), p))

	reqs := body["requests"].([]any)
	com := reqs[2].(map[string]any)["insertComment"].(map[string]any)
	rng := com["range"].(map[string]any)
	// "twice a year " is 13 units and the robot is two more.
	if rng["endIndex"] != float64(26+15) {
		t.Errorf("insertComment endIndex = %v, want %d", rng["endIndex"], 26+15)
	}
}

func TestBatchCarriesTheAssigneeWhenThereIsOne(t *testing.T) {
	p := testProposal
	p.Assignee = "nail@altery.com"
	body := decodeBatch(t, Batch(span(26, 43), p))

	com := body["requests"].([]any)[2].(map[string]any)["insertComment"].(map[string]any)
	if com["assigneeEmailAddress"] != "nail@altery.com" {
		t.Errorf("assigneeEmailAddress = %v", com["assigneeEmailAddress"])
	}
}

func TestApplyVerifiesTheHappyPathThreeWays(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)

	res, err := Apply(context.Background(), f, testDocID, testProposal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Verified {
		t.Errorf("Verified = false, checks = %+v, warnings = %v", res.Checks, res.Warnings)
	}
	if res.Checks != (Checks{SuggestionsInline: true, PreviewWithoutSuggestions: true, DocxAnchored: true}) {
		t.Errorf("checks = %+v, want all three", res.Checks)
	}
	if res.CommentID != "AAAC" || res.CommentUpdateState != "ALL_SAVED" {
		t.Errorf("comment = %q, state = %q", res.CommentID, res.CommentUpdateState)
	}
	if len(res.SuggestionIDs) != 1 || res.SuggestionIDs[0] != "suggest.abc" {
		t.Errorf("suggestion ids = %v", res.SuggestionIDs)
	}
	if res.Quoted != testProposal.Quoted || res.Replacement != testProposal.Replacement {
		t.Errorf("result = %+v, and it should carry the words it was asked to change", res)
	}
}

// TestApplyReadsTheCommentIDFromTheCommentThread is the shape Docs actually
// answers with, measured on 2026-09-07 on a throwaway document in the test
// folder: the id sits under insertComment.commentThread.commentId, beside an
// anchorId, the headPost and plainTextQuote. The first live write test failed
// on the flat insertComment.commentId the code had guessed, so a proposal
// landed with no comment id and withdraw would have refused it for ever. The
// fixture is that answer with the ids replaced.
func TestApplyReadsTheCommentIDFromTheCommentThread(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved-measured.json", withComment)

	res, err := Apply(context.Background(), f, testDocID, testProposal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CommentID != "AAACMeAsUrEd" {
		t.Errorf("comment id = %q, want the commentThread's AAACMeAsUrEd", res.CommentID)
	}
	if res.CommentUpdateState != "ALL_SAVED" {
		t.Errorf("state = %q", res.CommentUpdateState)
	}
	if !res.Verified {
		t.Errorf("Verified = false, checks = %+v, warnings = %v", res.Checks, res.Warnings)
	}
}

// TestApplyReadsTheDocumentItselfThenWritesOnce is the order the whole design
// rests on: an index is computed from a read made a moment earlier, never
// carried in from outside.
func TestApplyReadsTheDocumentItselfThenWritesOnce(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)

	if _, err := Apply(context.Background(), f, testDocID, testProposal); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.calls) < 2 {
		t.Fatalf("calls = %v", f.calls)
	}
	if f.calls[0].method != "GET" {
		t.Errorf("the first call is %s %s, and it should be the fresh read", f.calls[0].method, f.calls[0].url)
	}
	posts := 0
	for _, c := range f.calls {
		if c.method == "POST" {
			posts++
		}
	}
	if posts != 1 {
		t.Errorf("POSTs = %d, want exactly one batchUpdate", posts)
	}
}

func TestApplyReportsAPartialFailureFromTheCommentUpdateState(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-failed.json", withComment)

	res, err := Apply(context.Background(), f, testDocID, testProposal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true on an answer that says the comment was lost")
	}
	if res.CommentUpdateState != "ALL_FAILED_UNKNOWN_REASON" {
		t.Errorf("state = %q", res.CommentUpdateState)
	}
	if len(res.Warnings) == 0 {
		t.Error("no warning named the state")
	}
}

// TestApplyFailsThePreviewCheckWhenTheWriteWasADirectEdit is the whole reason
// the second read-back exists. The status code said 200 and the inline view
// carries a suggestion id, and the preview is the only place a direct edit
// shows up.
func TestApplyFailsThePreviewCheckWhenTheWriteWasADirectEdit(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview-edited.json", "batch-saved.json", withComment)

	res, err := Apply(context.Background(), f, testDocID, testProposal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Verified || res.Checks.PreviewWithoutSuggestions {
		t.Errorf("checks = %+v, and the preview shows the replacement", res.Checks)
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "preview") {
		t.Errorf("warnings = %v, and one should name the preview", res.Warnings)
	}
}

func TestApplyFailsTheDocxCheckWhenTheCommentIsNotInTheExport(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withoutComment)

	res, err := Apply(context.Background(), f, testDocID, testProposal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Verified || res.Checks.DocxAnchored {
		t.Errorf("checks = %+v, and the export carries no such comment", res.Checks)
	}
	if !res.Checks.SuggestionsInline || !res.Checks.PreviewWithoutSuggestions {
		t.Errorf("checks = %+v, and the other two routes held", res.Checks)
	}
}

// TestApplyAcceptsAnInsertDocsCutIntoTwoRuns is a fact about the answer rather
// than about the write: Docs splits one insert wherever it likes, and a check
// that read only the first run would call a landed proposal unverified.
func TestApplyAcceptsAnInsertDocsCutIntoTwoRuns(t *testing.T) {
	f := script(t, "before.json", "after-split.json", "preview.json", "batch-saved.json", withComment)

	res, err := Apply(context.Background(), f, testDocID, testProposal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Checks.SuggestionsInline {
		t.Errorf("checks = %+v, warnings = %v", res.Checks, res.Warnings)
	}
	if len(res.SuggestionIDs) != 1 || res.SuggestionIDs[0] != "suggest.abc" {
		t.Errorf("suggestion ids = %v, want the one id both runs carry", res.SuggestionIDs)
	}
}

func TestApplyRefusesAMultiTabDocumentBeforeAnyWrite(t *testing.T) {
	f := script(t, "two-tabs.json", "after.json", "preview.json", "batch-saved.json", withComment)

	_, err := Apply(context.Background(), f, testDocID, testProposal)
	if err == nil {
		t.Fatal("a two-tab document was written to")
	}
	if !strings.Contains(err.Error(), "2 tabs") {
		t.Errorf("error = %q, and it should name the tab count", err)
	}
	for _, c := range f.calls {
		if c.method != "GET" {
			t.Fatalf("a %s reached the wire: %v", c.method, f.calls)
		}
	}
}

func TestApplyRefusesAQuoteItCannotPlaceBeforeAnyWrite(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)

	_, err := Apply(context.Background(), f, testDocID, Proposal{
		Quoted: "reviewed monthly", Replacement: "reviewed weekly", Why: "so"})
	if err == nil {
		t.Fatal("a quote that is not in the document was written")
	}
	for _, c := range f.calls {
		if c.method != "GET" {
			t.Fatalf("a %s reached the wire: %v", c.method, f.calls)
		}
	}
}

// TestCheckRefusesEachBadProposalByName is the shape rule, asked of the proposal
// alone. It runs before the probe and before the first write, so a bad third
// entry stops a run that has changed nothing rather than one that has already
// landed two suggestions in somebody's document.
func TestCheckRefusesEachBadProposalByName(t *testing.T) {
	cases := []struct {
		name string
		p    Proposal
		says string
	}{
		{"no quote", Proposal{Replacement: "a", Why: "b"}, "quotes no text"},
		{"no replacement", Proposal{Quoted: "a", Why: "b"}, "replaces words with words"},
		// A paragraph's last run carries the paragraph mark, so a quote ending
		// in a newline matches inside that one paragraph and the span ends past
		// the mark. The deleteContentRange built from it merges the paragraph
		// with the one behind it, and all three read-backs hold over it: both
		// inlineHolds and Carries ask about this same string, which carries the
		// mark too.
		{"quote taking the paragraph mark", Proposal{Quoted: "a\n", Replacement: "b", Why: "d"}, "line break"},
		{"quote with a carriage return", Proposal{Quoted: "a\rb", Replacement: "c", Why: "d"}, "line break"},
		// Carries asks one paragraph at a time, so a replacement holding a line
		// break is in no single paragraph and the preview check's second
		// question can never see it. Refused at the door, the question keeps
		// the precondition it was written against.
		{"replacement with a newline", Proposal{Quoted: "a", Replacement: "b\nc", Why: "d"}, "line break"},
		{"replacement with a carriage return", Proposal{Quoted: "a", Replacement: "b\rc", Why: "d"}, "line break"},
		{"no reason", Proposal{Quoted: "a", Replacement: "b"}, "no reason"},
		{"markdown reason", Proposal{Quoted: "a", Replacement: "b", Why: "use **six months**"}, "markdown"},
		{"backtick reason", Proposal{Quoted: "a", Replacement: "b", Why: "the `annually` wording"}, "markdown"},
		// The heading arm is anchored to a line start, so asking the rule of
		// Prefix + Why moves the hash off offset zero and lets this one through
		// to the thread, where Docs renders it as typed. The same words on a
		// second line were always refused, which is the same reason written two
		// ways depending on where the author put it.
		{"heading reason", Proposal{Quoted: "a", Replacement: "b", Why: "# Six months, not twelve"}, "markdown"},
		// Batch writes the comment as Prefix + Why. A caller carrying reply's
		// convention over, where the body must open with the robot, would sign
		// the comment twice, and every read-back compares against the string
		// that was sent, so the run would report verified: true over it.
		{"reason already signed", Proposal{Quoted: "a", Replacement: "b", Why: Prefix + "the policy says six months"}, "already opens with"},
		// The near misses of the prefix. Each one lands the same doubled mark,
		// so a check that reads the whole prefix rather than the robot lets all
		// three through the writer that adds it. reply_test names the same
		// spellings, on the side that requires the mark.
		{"reason signed with no space", Proposal{Quoted: "a", Replacement: "b", Why: Robot + "the policy says six months"}, "already opens with"},
		{"reason signed behind a space", Proposal{Quoted: "a", Replacement: "b", Why: " " + Prefix + "the policy"}, "already opens with"},
		{"reason signed behind a newline", Proposal{Quoted: "a", Replacement: "b", Why: "\n" + Prefix + "the policy"}, "already opens with"},
		// A comment that is a bare signature. sameWords normalises whitespace on
		// both sides, so docx_anchored would match it and the run would report
		// verified: true over a reason that explains nothing.
		{"whitespace-only reason", Proposal{Quoted: "a", Replacement: "b", Why: "   \n\t "}, "no reason"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.p.Check()
			if err == nil {
				t.Fatalf("Check(%+v) accepted it", c.p)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("error = %q, and it should say %q", err, c.says)
			}
		})
	}
}

// TestApplyRefusesADeletionOnlyProposalBeforeAnyWrite is the empty replacement,
// end to end. The batch it would build carries an insertText with no text and a
// comment anchored on a range of length zero, which Docs rejects, and it would
// reach Google after the probe had created and trashed a document.
func TestApplyRefusesADeletionOnlyProposalBeforeAnyWrite(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)

	_, err := Apply(context.Background(), f, testDocID, Proposal{
		Quoted: "reviewed annually", Why: "the sentence says nothing"})
	if err == nil {
		t.Fatal("a proposal replacing words with nothing was written")
	}
	if len(f.calls) != 0 {
		t.Errorf("the shape was checked after %d requests, and it needs no request at all", len(f.calls))
	}
}

// TestApplyReportsABatchWhoseAnswerCouldNotBeRead is the difference between a
// write that never left and a write the server took. The session marks the
// second kind, and reporting it as a failure would send the skill back to
// propose the same change again, on top of the one already in the document.
func TestApplyReportsABatchWhoseAnswerCouldNotBeRead(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
	f.failAt[1] = acceptedError{errors.New("the answer could not be read: unexpected EOF")}

	res, err := Apply(context.Background(), f, testDocID, testProposal)
	if err != nil {
		t.Fatalf("a batch the server accepted was reported as never sent: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true on a write whose answer was never read")
	}
	if res.CommentID != "" {
		t.Errorf("comment id = %q, and the answer carrying it was never read", res.CommentID)
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "accepted by Docs") {
		t.Errorf("warnings = %v, and one should say the write was accepted", res.Warnings)
	}
	// The commentUpdateState warning is asked only of an answer that was read.
	// Here nothing was read, so reporting that the write "answered" the empty
	// string names something the server did not do, beside a warning that
	// already says the answer was lost.
	if strings.Contains(strings.Join(res.Warnings, " "), "commentUpdateState") {
		t.Errorf("warnings = %v, and none may quote a state the server never sent", res.Warnings)
	}
	if !res.Checks.SuggestionsInline {
		t.Errorf("checks = %+v, and the read-backs still ran", res.Checks)
	}
}

// TestApplyKeepsACommentIdDecodedFromAnAnswerThatFailed is the other half of
// the path above: valid JSON of the wrong shape, where the comment id is in
// hand even though the read failed. That id is the provenance withdraw needs,
// so it is kept, and the warning says what was lost rather than claiming the id
// is unknown beside a Result that carries it.
func TestApplyKeepsACommentIdDecodedFromAnAnswerThatFailed(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
	f.afterBatch = acceptedError{errors.New("the answer could not be read: json: cannot unmarshal number into Go struct field")}

	res, err := Apply(context.Background(), f, testDocID, testProposal)
	if err != nil {
		t.Fatalf("a batch the server accepted was reported as never sent: %v", err)
	}
	if res.CommentID == "" {
		t.Error("the comment id the answer carried was thrown away with the failure")
	}
	joined := strings.Join(res.Warnings, " ")
	if !strings.Contains(joined, "accepted by Docs") {
		t.Errorf("warnings = %v, and one should say the write was accepted", res.Warnings)
	}
	if strings.Contains(joined, "comment id is unknown") {
		t.Errorf("warnings = %v, and the comment id is in the Result", res.Warnings)
	}
}

// acceptedError is what the session hands back for a failure raised after the
// server accepted the request. The writer packages ask by behaviour, so the fake
// answers by behaviour too.
type acceptedError struct{ error }

func (acceptedError) Sent() bool { return true }

func TestRecordAppendsTheProposalToTheNote(t *testing.T) {
	note := []byte("---\ngdoc:\n  schema: 1\n  document_id: " + testDocID + "\n---\n\n# Scope\n")
	at := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	res := Result{
		Quoted:        "reviewed annually",
		SuggestionIDs: []string{"suggest.abc"},
		CommentID:     "AAAC",
	}

	out, missed, err := Record(note, []Result{res}, at)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(missed) != 0 {
		t.Fatalf("missed = %+v, and this result carries both ids", missed)
	}
	b, err := frontmatter.Read(out)
	if err != nil {
		t.Fatalf("the note it wrote does not read back: %v", err)
	}
	if len(b.Proposals) != 1 {
		t.Fatalf("proposals = %+v", b.Proposals)
	}
	p := b.Proposals[0]
	if p.ID != "suggest.abc" || p.CommentID != "AAAC" || p.Quoted != "reviewed annually" || !p.At.Equal(at) {
		t.Errorf("proposals[0] = %+v", p)
	}
	if !strings.Contains(string(out), "# Scope") {
		t.Error("the note's own text did not come through")
	}
}

// TestRecordSkipsAProposalWithNoSuggestionIDAndSaysSo keeps the note writable
// and keeps the loss visible. A result with no id is one nothing can withdraw
// later, and an entry with an empty id is a block that fails its own
// validation, which would lose the provenance of every other proposal in the
// same write. So it is skipped, and it comes back named: a proposal that is in
// the document and not in the note is the one thing this file exists to stop,
// and skipping it quietly is that loss with nobody told.
func TestRecordSkipsAProposalWithNoSuggestionIDAndSaysSo(t *testing.T) {
	note := []byte("---\ngdoc:\n  schema: 1\n  document_id: " + testDocID + "\n---\n")
	at := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	results := []Result{
		{Quoted: "a", CommentID: "AAAC"},
		{Quoted: "b", SuggestionIDs: []string{"suggest.def"}, CommentID: "AAAD"},
	}

	out, missed, err := Record(note, results, at)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := frontmatter.Read(out)
	if err != nil {
		t.Fatalf("the note it wrote does not read back: %v", err)
	}
	if len(b.Proposals) != 1 || b.Proposals[0].ID != "suggest.def" {
		t.Errorf("proposals = %+v", b.Proposals)
	}
	if len(missed) != 1 || missed[0].Quoted != "a" {
		t.Fatalf("missed = %+v, want the one result that could not be remembered", missed)
	}
}

// TestRecordLeavesTheNoteAloneWhenNothingCanBeRemembered is the same rule when
// every result is missing an id. There is nothing to add, so the bytes come back
// as they went in and the caller has nothing to write: a note reported as
// changed that gdoc did not change is a false fact in files_changed.
func TestRecordLeavesTheNoteAloneWhenNothingCanBeRemembered(t *testing.T) {
	note := []byte("---\ngdoc:\n  schema: 1\n  document_id: " + testDocID + "\n---\n")
	at := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)

	out, missed, err := Record(note, []Result{{Quoted: "a", CommentID: "AAAC"}}, at)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != string(note) {
		t.Errorf("the note was rewritten:\n%s", out)
	}
	if len(missed) != 1 {
		t.Fatalf("missed = %+v", missed)
	}
}

// TestTheGuardCarriesEveryOneOfProposesRequests is the other half of the write
// bar. The document is handed in, so it sits at suggest level, and the guard
// carries the batch only because the body says SUGGEST exactly. A test here
// catches a body this package rephrases in a way the guard reads as an edit,
// which is the one refusal that would only show up against live Drive.
//
// Judge takes net/url rather than net/http, so this stays a room the boundary
// test does not list.
func TestTheGuardCarriesEveryOneOfProposesRequests(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
	if _, err := Apply(context.Background(), f, testDocID, testProposal); err != nil {
		t.Fatal(err)
	}

	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)
	for i, c := range f.calls {
		u, err := url.Parse(c.url)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		if err := p.Judge(c.method, u, c.body); err != nil {
			t.Errorf("request %d, %s %s: %v", i, c.method, c.url, err)
		}
	}
}

// TestTheGuardRefusesTheSameBatchWithoutSuggestMode states what the bar is
// made of. Take writeControl out and the identical three requests are a direct
// edit, which the guard refuses on a document gdoc was handed.
func TestTheGuardRefusesTheSameBatchWithoutSuggestMode(t *testing.T) {
	body := decodeBatch(t, Batch(span(26, 43), testProposal))
	delete(body, "writeControl")
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)
	u, err := url.Parse(BatchURL(testDocID))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Judge("POST", u, raw); err == nil {
		t.Fatal("the batch was carried on a handed-in document without SUGGEST")
	}
}
