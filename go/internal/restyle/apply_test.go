package restyle

// The apply loop, read as a caller reads it: what leaves the machine, in what
// order, carrying which revision id, and what the run reports when a batch does
// not land.
//
// Every session here is scripted. The loop takes a Session interface, so no
// test in this file touches a wire, and each case is a list of answers written
// out beside the assertions about the requests that asked for them.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"gdoc/internal/guard"
)

// scripted is one session's whole behaviour: the answers it gives, in order,
// and the record of what it was asked.
type scripted struct {
	// posts are the bodies the loop sent, decoded, one per POST.
	posts []map[string]json.RawMessage
	// answers are what each POST comes back with, in order. A short list is a
	// test asking for more writes than it scripted, which fails loudly.
	answers []scriptedAnswer
	// reads is how many times the loop read the document. The loop reads
	// nothing between batches, so every test here wants 0 and GetJSON fails
	// rather than answering.
	reads int
}

// scriptedAnswer is one POST's outcome: the revision Docs reports back, or the
// error it fails with.
type scriptedAnswer struct {
	revision string
	err      error
}

func (s *scripted) PostJSON(ctx context.Context, rawURL string, body any, into any) error {
	raw, ok := body.(json.RawMessage)
	if !ok {
		return fmt.Errorf("the loop must send raw bytes it already built, got %T", body)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("the body must be JSON: %v", err)
	}
	s.posts = append(s.posts, decoded)
	if len(s.posts) > len(s.answers) {
		return fmt.Errorf("the loop sent batch %d and only %d answers were scripted", len(s.posts), len(s.answers))
	}
	a := s.answers[len(s.posts)-1]
	if a.err != nil {
		return a.err
	}
	answer := map[string]any{"writeControl": map[string]any{"requiredRevisionId": a.revision}}
	if a.revision == "" {
		answer = map[string]any{}
	}
	out, err := json.Marshal(answer)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, into)
}

func (s *scripted) GetJSON(ctx context.Context, rawURL string, into any) error {
	s.reads++
	return fmt.Errorf("the loop must not read the document: it read %s", rawURL)
}

// sent marks an error as one raised after the server had accepted the request,
// which is how internal/gapi marks a write whose answer could not be read.
type sentAnswer struct{ error }

func (sentAnswer) Sent() bool { return true }

// requestsIn is the request list of one sent batch, decoded.
func requestsIn(t *testing.T, body map[string]json.RawMessage) []map[string]json.RawMessage {
	t.Helper()
	raw, ok := body["requests"]
	if !ok {
		t.Fatalf("a batch names its requests in requests, and this one carries %v", keysOf(body))
	}
	var reqs []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &reqs); err != nil {
		t.Fatalf("the request list must decode: %v", err)
	}
	return reqs
}

// revisionIn is the requiredRevisionId of one sent batch, or "" when the batch
// named none at all.
func revisionIn(t *testing.T, body map[string]json.RawMessage) string {
	t.Helper()
	raw, ok := body["writeControl"]
	if !ok {
		return ""
	}
	var wc struct {
		RequiredRevisionID string `json:"requiredRevisionId"`
	}
	if err := json.Unmarshal(raw, &wc); err != nil {
		t.Fatalf("writeControl must decode: %v", err)
	}
	return wc.RequiredRevisionID
}

func keysOf(m map[string]json.RawMessage) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// three styling requests, each a real kind the in-place allowlist carries, so
// every body in this file is one the guard would judge.
func threeRequests() []map[string]any {
	return []map[string]any{
		{"updateParagraphStyle": map[string]any{"range": map[string]any{"startIndex": 1, "endIndex": 5}, "paragraphStyle": map[string]any{"spaceAbove": points(18)}, "fields": "spaceAbove"}},
		{"updateTextStyle": map[string]any{"range": map[string]any{"startIndex": 1, "endIndex": 5}, "textStyle": map[string]any{"fontSize": points(11)}, "fields": "fontSize"}},
		{"updateTableCellStyle": map[string]any{"tableStartLocation": map[string]any{"index": 10}, "tableCellStyle": map[string]any{"paddingTop": points(4)}, "fields": "paddingTop"}},
	}
}

// Each batch after the first carries the revision the last answer named, and
// nothing carries the one the run started with twice. That is the only thing
// standing between a restyle and a document somebody edited while it ran.
func TestEachBatchCarriesTheRevisionFromTheLastAnswer(t *testing.T) {
	oneRequestPerBatch(t, threeRequests())
	s := &scripted{answers: []scriptedAnswer{{revision: "rev2"}, {revision: "rev3"}, {revision: "rev4"}}}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1")
	if err != nil {
		t.Fatalf("the run must hold: %v", err)
	}

	if len(s.posts) != 3 {
		t.Fatalf("the loop sent %d batches, want 3, one per request at a ceiling of one", len(s.posts))
	}
	for i, want := range []string{"rev1", "rev2", "rev3"} {
		if got := revisionIn(t, s.posts[i]); got != want {
			t.Errorf("batch %d carried requiredRevisionId %q, want %q, which is the revision the answer before it named", i+1, got, want)
		}
	}
	if got.Batches != 3 || got.Requests != 3 {
		t.Errorf("Applied reports %d batches and %d requests, want 3 and 3", got.Batches, got.Requests)
	}
	if got.RevisionID != "rev4" {
		t.Errorf("Applied reports revision %q, want rev4, the one the last answer named", got.RevisionID)
	}
	if got.Stale {
		t.Errorf("no batch was refused, so Stale must be false")
	}
	if s.reads != 0 {
		t.Errorf("every answer carried a revision, so the loop must read nothing back: it read %d times", s.reads)
	}
}

// A read that carried no revision id ships "" in the body, and the guard has no
// rule about it: Docs would take the batch and the only protection this
// milestone has would be gone with nothing said. So it is refused at the call
// site, before anything leaves the machine.
func TestAnEmptyRevisionIsRefusedBeforeAnythingIsSent(t *testing.T) {
	s := &scripted{}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "")

	if err == nil {
		t.Fatalf("a restyle with no revision id must be refused, got %+v", got)
	}
	if !strings.Contains(err.Error(), "revision") {
		t.Errorf("the refusal must name the revision id, got %q", err)
	}
	if len(s.posts) != 0 {
		t.Errorf("nothing may be sent, and %d batches were", len(s.posts))
	}
	if got.Batches != 0 {
		t.Errorf("Batches = %d, want 0", got.Batches)
	}
}

// Docs refusing a batch on a stale revision is the document having moved under
// the run. It is reported as that, and the run stops: a retry against a fresh
// revision would be gdoc styling a document somebody is editing, which is the
// exact case requiredRevisionId exists to refuse.
func TestAStaleRevisionIsReportedAndNeverRetried(t *testing.T) {
	oneRequestPerBatch(t, threeRequests())
	stale := fmt.Errorf("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate answered 400: Invalid requiredRevisionId. The revision ID is not the latest revision of the document.")
	s := &scripted{answers: []scriptedAnswer{{revision: "rev2"}, {err: stale}}}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1")

	if err == nil {
		t.Fatalf("a stale revision must fail the run, got %+v", got)
	}
	if !got.Stale {
		t.Errorf("Stale must say what happened, and it says %v", got.Stale)
	}
	if len(s.posts) != 2 {
		t.Errorf("the loop sent %d batches: the refused one must not be retried and the third must never be sent", len(s.posts))
	}
	if got.Batches != 1 {
		t.Errorf("one batch held, and Applied reports %d", got.Batches)
	}
	if !strings.Contains(strings.Join(got.Warnings, " "), "version history") {
		t.Errorf("a run that stopped after a batch held leaves a half-styled document, and the warnings must say so: %v", got.Warnings)
	}
}

// An ordinary failure stops the run too, and it is reported as itself rather
// than as a document that moved.
func TestABatchRefusedForAnotherReasonStopsTheRun(t *testing.T) {
	oneRequestPerBatch(t, threeRequests())
	s := &scripted{answers: []scriptedAnswer{{err: fmt.Errorf("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate answered 403: The caller does not have permission")}}}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1")

	if err == nil {
		t.Fatalf("a refused batch must fail the run, got %+v", got)
	}
	if got.Stale {
		t.Errorf("nothing said the document moved, so Stale must be false")
	}
	if len(s.posts) != 1 {
		t.Errorf("the loop sent %d batches, want 1: a refusal is never retried", len(s.posts))
	}
	if got.Batches != 0 {
		t.Errorf("no batch held, and Applied reports %d", got.Batches)
	}
	if !strings.Contains(strings.Join(got.Warnings, " "), "no batch") {
		t.Errorf("a run that stopped before anything was confirmed must say so: %v", got.Warnings)
	}
}

// A batch that failed on the request itself is three things this package cannot
// tell apart: a guard refusal, where nothing left the machine, a 4xx, where Docs
// rejected the batch whole, and a 5xx or a dropped connection, where the request
// was written and may have been applied. internal/gapi marks only the failures
// raised after a 2xx, so the definite sentence would be a claim gdoc cannot
// make, on the one grant in this binary that permits a direct edit.
func TestAFailedRequestNeverSaysTheDocumentIsAsItWas(t *testing.T) {
	oneRequestPerBatch(t, threeRequests())
	s := &scripted{answers: []scriptedAnswer{{err: fmt.Errorf("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate answered 503: backend error")}}}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1")

	if err == nil {
		t.Fatalf("a batch that did not land must fail the run, got %+v", got)
	}
	joined := strings.Join(got.Warnings, " ")
	if strings.Contains(joined, "the document is as it was") {
		t.Errorf("a 5xx may have been applied, and this says it was not: %v", got.Warnings)
	}
	if !strings.Contains(joined, "may have reached Docs") {
		t.Errorf("the warnings must say the batch may have reached Docs: %v", got.Warnings)
	}
	if !strings.Contains(joined, "version history") {
		t.Errorf("a document that may be part styled has no rollback, and the warnings must say so: %v", got.Warnings)
	}
	if got.MaybeApplied {
		t.Error("maybe_applied is what Docs accepted, and nothing here said it did")
	}
}

// A document that moved is the one refusal that keeps the definite sentence.
// Docs refuses the batch whole on requiredRevisionId, so nothing in it reached
// the document, and a run that had confirmed nothing before it leaves the
// document exactly as it was.
func TestAStaleFirstBatchStillSaysTheDocumentIsAsItWas(t *testing.T) {
	oneRequestPerBatch(t, threeRequests())
	stale := fmt.Errorf("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate answered 400: Invalid requiredRevisionId. The revision ID is not the latest revision of the document.")
	s := &scripted{answers: []scriptedAnswer{{err: stale}}}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1")

	if err == nil {
		t.Fatalf("a stale revision must fail the run, got %+v", got)
	}
	if !strings.Contains(strings.Join(got.Warnings, " "), "the document is as it was") {
		t.Errorf("a batch Docs refused whole left nothing behind, and the warnings must say so: %v", got.Warnings)
	}
}

// A write Docs accepted whose answer could not be read is not a write that
// never happened. The run stops, because the revision it would need for the
// next batch is in the answer it could not read, and the warning says the batch
// may be in the document.
func TestAnAnswerThatCouldNotBeReadStopsTheRun(t *testing.T) {
	oneRequestPerBatch(t, threeRequests())
	s := &scripted{answers: []scriptedAnswer{{revision: "rev2"}, {err: sentAnswer{errors.New("the answer was not JSON")}}}}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1")

	if err == nil {
		t.Fatalf("an unreadable answer must fail the run, got %+v", got)
	}
	if len(s.posts) != 2 {
		t.Errorf("the loop sent %d batches, want 2: a write that may have landed is never sent again", len(s.posts))
	}
	joined := strings.Join(got.Warnings, " ")
	if !strings.Contains(joined, "accepted") {
		t.Errorf("the warning must say Docs accepted the batch: %v", got.Warnings)
	}
	if !got.MaybeApplied {
		t.Error("maybe_applied = false on a batch Docs accepted: the caller reads it to decide whether there is anything to read back")
	}
	if got.Batches != 1 {
		t.Errorf("batches = %d, want 1: the confirmed count never folds in the batch that may have landed", got.Batches)
	}
	if !strings.Contains(joined, "one more may be in the document") {
		t.Errorf("the sentence about what is left behind must count the batch that may be there: %v", got.Warnings)
	}
}

// The first batch taking that path is the case the count alone cannot report. A
// run with no confirmed batch and one that may be in the document must never say
// the document is as it was: it may have been directly edited, and that is the
// run the preservation facts are needed for most.
func TestAFirstBatchThatMayHaveLandedNeverSaysTheDocumentIsAsItWas(t *testing.T) {
	oneRequestPerBatch(t, threeRequests())
	s := &scripted{answers: []scriptedAnswer{{err: sentAnswer{errors.New("the answer was not JSON")}}}}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1")

	if err == nil {
		t.Fatalf("an unreadable answer must fail the run, got %+v", got)
	}
	if got.Batches != 0 || !got.MaybeApplied {
		t.Errorf("batches = %d and maybe_applied = %v, want 0 and true", got.Batches, got.MaybeApplied)
	}
	joined := strings.Join(got.Warnings, " ")
	if strings.Contains(joined, "the document is as it was") {
		t.Errorf("a batch Docs accepted may be in the document, and this says it is not: %v", got.Warnings)
	}
	if !strings.Contains(joined, "may be in the document") {
		t.Errorf("the warnings must say the batch may be there: %v", got.Warnings)
	}
}

// An answer carrying no revision id, with a batch still to send, leaves the
// loop with nothing to send the next batch under. It stops the run rather than
// reading the document for one: a revision read here is the document as it is
// now, so a foreign edit made in the gap would be adopted as this run's own and
// the batches behind it accepted against it. The batches that landed stay, and
// the warnings say the document is half styled.
func TestAnAnswerCarryingNoRevisionMidRunStopsTheRun(t *testing.T) {
	oneRequestPerBatch(t, threeRequests())
	s := &scripted{answers: []scriptedAnswer{{revision: ""}, {revision: "rev3"}, {revision: "rev4"}}}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1")

	if err == nil {
		t.Fatalf("a quiet answer mid-run must fail the run, got %+v", got)
	}
	if !strings.Contains(err.Error(), "batch 1 of 3") || !strings.Contains(err.Error(), "revision") {
		t.Errorf("the error must name the batch and the revision: %v", err)
	}
	if s.reads != 0 {
		t.Errorf("the loop read %d times, want 0: it never reads between batches", s.reads)
	}
	if len(s.posts) != 1 {
		t.Errorf("the loop sent %d batches, want 1: without a revision it sends nothing", len(s.posts))
	}
	if got.Batches != 1 {
		t.Errorf("Batches = %d, want 1: the first batch landed", got.Batches)
	}
	if got.RevisionID != "rev1" {
		t.Errorf("RevisionID = %q, want rev1, the revision that batch was sent against", got.RevisionID)
	}
	if !got.RevisionUnconfirmed {
		t.Error("RevisionUnconfirmed = false on a run whose last batch sent answered with no revision id")
	}
	// Neither flag belongs here. Docs accepted the batch and answered, so
	// nothing was refused as stale and nothing may be in the document
	// uncounted.
	if got.Stale {
		t.Error("Stale is set on a run no batch of which was refused")
	}
	if got.MaybeApplied {
		t.Error("MaybeApplied is set on a run whose every sent batch was answered")
	}
	if joined := strings.Join(got.Warnings, " "); !strings.Contains(joined, "half styled") {
		t.Errorf("the warnings must say the document is half styled: %v", got.Warnings)
	}
}

// The last batch needs no revision after it, so an answer that carries none
// costs no read. What it does cost is the run's word about where the document
// ended up, and that is said rather than guessed at.
func TestTheLastAnswerCarryingNoRevisionIsAWarningAndNotARead(t *testing.T) {
	oneRequestPerBatch(t, threeRequests())
	s := &scripted{answers: []scriptedAnswer{{revision: "rev2"}, {revision: "rev3"}, {revision: ""}}}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1")
	if err != nil {
		t.Fatalf("the run must hold: %v", err)
	}

	if s.reads != 0 {
		t.Errorf("nothing follows the last batch, so the loop must read nothing: it read %d times", s.reads)
	}
	if got.RevisionID != "rev3" {
		t.Errorf("RevisionID = %q, want rev3, the revision the last batch was sent against", got.RevisionID)
	}
	if !strings.Contains(strings.Join(got.Warnings, " "), "revision") {
		t.Errorf("the last answer named no revision, and the warnings must say so: %v", got.Warnings)
	}
	// The flag, and it is what the caller acts on rather than the sentence. A
	// marker batch is the last batch of its own run, and the styling phase that
	// follows it is sent against RevisionID: read as confirmed, that batch is
	// refused as stale and isStale names a third party for a revision gdoc
	// itself moved.
	if !got.RevisionUnconfirmed {
		t.Error("RevisionUnconfirmed = false on a run whose last answer named no revision id: a caller reading only the warning cannot go and read for one")
	}
}

// A plan with nothing in it sends nothing. A document whose paragraphs the
// house style has no look for is the case, and a batchUpdate naming no requests
// would be a request made for no reason.
func TestAnEmptyPlanSendsNothing(t *testing.T) {
	s := &scripted{}

	got, err := Apply(context.Background(), s, "DOC1", nil, "rev1")
	if err != nil {
		t.Fatalf("an empty plan is not a failure: %v", err)
	}
	if len(s.posts) != 0 || s.reads != 0 {
		t.Errorf("nothing may leave the machine: %d posts, %d reads", len(s.posts), s.reads)
	}
	if got.Batches != 0 || got.RevisionID != "rev1" {
		t.Errorf("Applied = %+v, want no batches and the revision it was given", got)
	}
}

// The guard refuses a batchUpdate it cannot read whole, and the transport reads
// the first megabyte of a body. So the batches are sized under that, and a
// batch is measured as the bytes that actually go out rather than as a count of
// requests.
func TestBatchesAreSizedUnderTheGuardsPeek(t *testing.T) {
	if maxBatchBytes >= 1<<20 {
		t.Fatalf("maxBatchBytes is %d, which is not under the guard's own megabyte peek", maxBatchBytes)
	}
	long := strings.Repeat("x", maxBatchBytes/3)
	var reqs []map[string]any
	for i := 0; i < 6; i++ {
		reqs = append(reqs, map[string]any{"updateParagraphStyle": map[string]any{"fields": long}})
	}

	batches, err := Batches(reqs)
	if err != nil {
		t.Fatalf("six requests of a third of the ceiling each must split, not fail: %v", err)
	}
	if len(batches) < 3 {
		t.Errorf("six requests of a third of the ceiling each went into %d batches", len(batches))
	}
	total := 0
	for i, b := range batches {
		if len(b) == 0 {
			t.Errorf("batch %d is empty, and a batch with no requests is a request made for no reason", i+1)
		}
		total += len(b)
		body, err := batchBody(b, "rev1", "")
		if err != nil {
			t.Fatalf("a batch must marshal: %v", err)
		}
		if len(body) > maxBatchBytes {
			t.Errorf("batch %d is %d bytes, over the %d ceiling", i+1, len(body), maxBatchBytes)
		}
	}
	if total != len(reqs) {
		t.Errorf("the batches hold %d requests of %d: nothing may be dropped", total, len(reqs))
	}
}

// One request too big to send at all is refused naming its kind. Splitting it
// is not something this loop can do, and sending it would be sending a body the
// guard reads half of.
func TestOneRequestOverTheCeilingIsRefusedByName(t *testing.T) {
	reqs := []map[string]any{
		{"updateTextStyle": map[string]any{"fields": strings.Repeat("x", maxBatchBytes+1)}},
	}

	if _, err := Batches(reqs); err == nil {
		t.Fatalf("a request over the ceiling must be refused")
	} else if !strings.Contains(err.Error(), "updateTextStyle") {
		t.Errorf("the refusal must name the request kind, got %q", err)
	}
}

// The body the loop builds is the body the guard judges. Building it here and
// judging it there is what keeps the two from drifting apart in silence: a
// batch carrying writeControl is a shape no test judged before this one.
func TestTheGuardCarriesTheBatchTheLoopBuilds(t *testing.T) {
	body, err := batchBody(threeRequests(), "rev1", "")
	if err != nil {
		t.Fatalf("the batch must marshal: %v", err)
	}
	p := guard.NewPolicy()
	p.AllowFile("DOC1", guard.LevelSuggest)
	p.GrantInPlace("DOC1")
	u, err := url.Parse(BatchURL("DOC1"))
	if err != nil {
		t.Fatalf("the URL must parse: %v", err)
	}
	if err := p.Judge("POST", u, body); err != nil {
		t.Fatalf("the guard refused the batch the apply loop builds: %v", err)
	}
}

// oneRequestPerBatch narrows the batch ceiling until each of these requests
// lands in a batch of its own, and puts the ceiling back afterwards. The
// ceiling is a package variable for this reason alone, the way cmd/gdoc's wait
// interval is: a test that had to build half a megabyte of requests to watch
// the loop send two batches would be a test nobody reads.
func oneRequestPerBatch(t *testing.T, reqs []map[string]any) {
	t.Helper()
	old := maxBatchBytes
	t.Cleanup(func() { maxBatchBytes = old })
	largest := 0
	for _, r := range reqs {
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("a request must marshal: %v", err)
		}
		if len(raw) > largest {
			largest = len(raw)
		}
	}
	// Room for the envelope and the largest request, and one byte more. A
	// second request of any size at all puts the batch over.
	maxBatchBytes = batchEnvelope + largest + 1
}

// A lost answer is a lost answer whatever words it carries, and the words it
// carries name this loop's own field. encoding/json builds a type error out of
// the struct it was decoding into, so a batchUpdate that answered 200 with a
// writeControl of the wrong shape fails with "cannot unmarshal string into Go
// struct field batchAnswer.writeControl ... requiredRevisionId", which holds the
// word isStale matches on. Read as a stale revision that batch would be reported
// as a document that moved, MaybeApplied would stay false, the run would say the
// document is as it was, and cmdRestyle would skip the read-back on the one path
// where a direct edit may be in the document. The two cases cannot collide the
// other way: Docs refuses a moved revision with a 4xx, which gapi never marks as
// sent.
func TestALostAnswerNamingTheRevisionFieldIsNotAStaleRevision(t *testing.T) {
	oneRequestPerBatch(t, threeRequests())
	var answer batchAnswer
	decode := json.Unmarshal([]byte(`{"writeControl":"not an object"}`), &answer)
	if decode == nil || !strings.Contains(strings.ToLower(decode.Error()), "revision") {
		t.Fatalf("this test is worth nothing unless the decode error names the revision field, and it says %v", decode)
	}
	lost := sentAnswer{fmt.Errorf(
		"https://docs.googleapis.com/v1/documents/DOC1:batchUpdate answered 200 with a body that is not JSON: %w", decode)}
	s := &scripted{answers: []scriptedAnswer{{err: lost}}}

	got, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1")

	if err == nil {
		t.Fatalf("a lost answer must fail the run, got %+v", got)
	}
	if got.Stale {
		t.Error("Stale = true on a batch Docs accepted: nothing said the document had moved")
	}
	if !got.MaybeApplied {
		t.Error("maybe_applied = false on a batch Docs accepted: cmdRestyle reads it to decide whether to read the document back")
	}
	if got.MaybeRequests != 1 {
		t.Errorf("maybe_requests = %d, want 1: the read-back is asked about the requests that left the machine", got.MaybeRequests)
	}
	joined := strings.Join(got.Warnings, " ")
	if !strings.Contains(joined, "accepted") {
		t.Errorf("the warning must say Docs accepted the batch: %v", got.Warnings)
	}
	if strings.Contains(joined, "the document is as it was") {
		t.Errorf("a batch that may be in the document must never be reported as a document nothing reached: %v", got.Warnings)
	}
}

// Suggest is the same loop with one field added, and these are the three things
// that field changes: what goes out, what the guard makes of it, and what a run
// that stopped early tells somebody to do about it.

// insertRequests are the prelude's shape rather than a restyle's: they change
// characters, which is exactly why they may only ever go out as suggestions.
func insertRequests() []map[string]any {
	return []map[string]any{
		{"insertText": map[string]any{"text": "A policy\n", "location": map[string]any{"index": 1}}},
		{"updateParagraphStyle": map[string]any{"range": map[string]any{"startIndex": 1, "endIndex": 10}, "paragraphStyle": map[string]any{"alignment": "CENTER"}, "fields": "alignment"}},
	}
}

func TestASuggestedBatchCarriesTheModeAndTheRevision(t *testing.T) {
	// Arrange
	s := &scripted{answers: []scriptedAnswer{{revision: "rev2"}}}

	// Act
	out, err := Suggest(context.Background(), s, "DOC1", insertRequests(), "rev1")

	// Assert
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if out.Batches != 1 || out.Requests != 2 {
		t.Errorf("applied = %+v, want one batch of two requests", out)
	}
	if len(s.posts) != 1 {
		t.Fatalf("the loop sent %d batches, want one", len(s.posts))
	}
	var control struct {
		WriteMode          string `json:"writeMode"`
		RequiredRevisionID string `json:"requiredRevisionId"`
	}
	if err := json.Unmarshal(s.posts[0]["writeControl"], &control); err != nil {
		t.Fatalf("the batch carried no readable writeControl: %v", err)
	}
	if control.WriteMode != "SUGGEST" {
		t.Errorf("writeMode = %q, want SUGGEST", control.WriteMode)
	}
	if control.RequiredRevisionID != "rev1" {
		t.Errorf("requiredRevisionId = %q, want rev1: a suggested batch is still refused on a document that moved",
			control.RequiredRevisionID)
	}
}

// A direct batch says nothing about a write mode, and that is what makes the
// two functions two: a caller cannot get the prelude out through Apply.
func TestADirectBatchNamesNoWriteMode(t *testing.T) {
	s := &scripted{answers: []scriptedAnswer{{revision: "rev2"}}}
	if _, err := Apply(context.Background(), s, "DOC1", threeRequests(), "rev1"); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, named := s.posts[0]["writeControl"]; !named {
		t.Fatal("the batch carried no writeControl at all")
	}
	if strings.Contains(string(s.posts[0]["writeControl"]), "writeMode") {
		t.Errorf("writeControl = %s, want no write mode on a direct edit", s.posts[0]["writeControl"])
	}
}

// The guard is what holds the mode, not this package. A document handed in with
// no grant of any kind carries the prelude's own batch, and refuses the same
// requests sent directly.
func TestTheGuardCarriesASuggestedBatchWithNoGrantAndRefusesADirectOne(t *testing.T) {
	suggested, err := batchBody(insertRequests(), "rev1", "SUGGEST")
	if err != nil {
		t.Fatalf("the batch must marshal: %v", err)
	}
	direct, err := batchBody(insertRequests(), "rev1", "")
	if err != nil {
		t.Fatalf("the batch must marshal: %v", err)
	}
	p := guard.NewPolicy()
	p.AllowFile("DOC1", guard.LevelSuggest)
	u, err := url.Parse(BatchURL("DOC1"))
	if err != nil {
		t.Fatalf("the URL must parse: %v", err)
	}
	if err := p.Judge("POST", u, suggested); err != nil {
		t.Errorf("the prelude needs no permission this milestone added: %v", err)
	}
	if err := p.Judge("POST", u, direct); err == nil {
		t.Error("the same requests sent directly must be refused: nothing may write a character on gdoc's own authority")
	}
}

// What a run that stopped early leaves behind is a proposal, so the sentence it
// prints is a rejection rather than the version history. Sending somebody to
// undo a suggestion by hand is the wrong recovery for the right problem.
func TestASuggestedRunThatStoppedSaysToRejectRatherThanToUndo(t *testing.T) {
	oneRequestPerBatch(t, insertRequests())
	s := &scripted{answers: []scriptedAnswer{{revision: "rev2"}, {err: errors.New("500: nothing came back")}}}

	out, err := Suggest(context.Background(), s, "DOC1", insertRequests(), "rev1")
	if err == nil {
		t.Fatal("a batch that did not land must be an error")
	}
	said := strings.Join(out.Warnings, " | ")
	if !strings.Contains(said, "rejecting it in the browser") {
		t.Errorf("warnings = %q, want the recovery a proposal has", said)
	}
	if strings.Contains(said, "version history") {
		t.Errorf("warnings = %q, want no version history: nothing here was written directly", said)
	}
}
