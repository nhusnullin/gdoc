package annotate

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/guard"
	"gdoc/internal/propose"
)

// TestCheckRefusesEachBadShapeByName is the rule asked of every entry before
// the first one leaves the machine. Each refusal names what is wrong, because
// the caller is a skill writing a file and the file is what has to change.
func TestCheckRefusesEachBadShapeByName(t *testing.T) {
	for _, c := range []struct {
		name  string
		a     Annotation
		names string
	}{
		{
			name:  "an empty quote",
			a:     Annotation{Quoted: "", Why: "The 2026 register says quarterly."},
			names: "quotes no text",
		},
		{
			name:  "a quote carrying a line break",
			a:     Annotation{Quoted: "reviewed annually\nby the board", Why: "The 2026 register says quarterly."},
			names: "line break",
		},
		{
			name:  "an empty why",
			a:     Annotation{Quoted: "reviewed annually", Why: ""},
			names: "no reason",
		},
		{
			name:  "a why of nothing but spaces",
			a:     Annotation{Quoted: "reviewed annually", Why: "   \n  "},
			names: "no reason",
		},
		{
			name:  "a why already opening with the robot",
			a:     Annotation{Quoted: "reviewed annually", Why: "🤖 The 2026 register says quarterly."},
			names: "already opens with",
		},
		{
			name:  "a why carrying markdown",
			a:     Annotation{Quoted: "reviewed annually", Why: "The register says **quarterly**."},
			names: "**",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := c.a.Check()
			if err == nil {
				t.Fatalf("Check() of %+v gave no error, and %s is refused", c.a, c.name)
			}
			if !strings.Contains(err.Error(), c.names) {
				t.Errorf("Check() said %q, which does not name %q", err, c.names)
			}
		})
	}
}

// TestCheckAcceptsAPlainAnnotation is the other direction. An assignee is
// optional and decides nothing about the shape.
func TestCheckAcceptsAPlainAnnotation(t *testing.T) {
	for _, a := range []Annotation{
		{Quoted: "reviewed annually", Why: "The 2026 register says quarterly."},
		{Quoted: "the Cyprus entity", Why: "Named twice with two spellings.", Assignee: "x@altery.com"},
		{Quoted: "5 * 3 units", Why: "See issue #28 for the rest."},
	} {
		if err := a.Check(); err != nil {
			t.Errorf("Check() of %+v said %q, and there is nothing wrong with it", a, err)
		}
	}
}

// TestBodyIsTheRobotAndTheWhy states the prefix as a literal rather than
// reading the constant, because a test that reads the constant follows it
// wherever somebody moves it. The mark is the only record of authorship a
// comment has.
func TestBodyIsTheRobotAndTheWhy(t *testing.T) {
	if got, want := Body("The register."), "🤖 The register."; got != want {
		t.Errorf("Body(%q) = %q, want %q", "The register.", got, want)
	}
}

const testDocID = "1AnNoTaTe000000000000000000000000000000000"

var testAnnotation = Annotation{
	Quoted: "reviewed annually",
	Why:    "The 2026 register says quarterly.",
}

// "reviewed annually" sits at these indexes in before.json, in the UTF-16 code
// units the Docs API counts. They are literals here rather than a call to
// FindSpan, because a test that computes the span it is checking agrees with
// the walk however wrong the walk is.
const (
	quoteStart = 26
	quoteEnd   = 43
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// call is one request the fake was asked to make, so a test can assert what
// reached the wire as well as what came back.
type call struct {
	method string
	url    string
	body   json.RawMessage
}

// fakeSession is Google as far as this package is concerned: the Docs read the
// span is found in, the one batchUpdate, and the two read-backs, Drive's
// comment listing and the docx export. Nothing here names net/http: the rooms
// that may are the ones the boundary test lists, and this is not one.
type fakeSession struct {
	calls   []call
	inline  []byte
	batch   []byte
	listing []byte
	export  []byte
	failAt  map[int]error
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
	answer := []byte(`{}`)
	switch {
	case rawURL == docs.URL(testDocID):
		answer = f.inline
	case strings.Contains(rawURL, "/comments?"):
		answer = f.listing
	}
	if into == nil {
		return nil
	}
	return json.Unmarshal(answer, into)
}

func (f *fakeSession) GetBytes(_ context.Context, rawURL string, _ int64) ([]byte, error) {
	if err := f.record("GET", rawURL, nil); err != nil {
		return nil, err
	}
	return f.export, nil
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

// script is a Google that answers the read from a named fixture and the write
// from another, with both read-backs holding. A test about a read-back that
// does not hold replaces the one route it is about, so every other test says
// what it is about by saying nothing.
func script(t *testing.T, before, batch string) *fakeSession {
	t.Helper()
	return &fakeSession{
		inline:  fixture(t, before),
		batch:   fixture(t, batch),
		listing: fixture(t, "comments.json"),
		export:  exportOf(t, string(fixture(t, "document.xml")), string(fixture(t, "comments.xml"))),
		failAt:  map[int]error{},
	}
}

// span is the range the walk gives back, written out here so a test can state
// the indexes it means rather than compute them.
func span(start, end int) docs.Range {
	return docs.Range{Tab: "t.0", Start: start, End: end}
}

func decodeBatch(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("the batch body is not JSON: %v", err)
	}
	return out
}

func posts(f *fakeSession) []call {
	var out []call
	for _, c := range f.calls {
		if c.method == "POST" {
			out = append(out, c)
		}
	}
	return out
}

// TestBatchIsOneInsertCommentUnderSuggest is the whole write, and its being one
// request is the point. A batch holding an insertComment and nothing beside it
// cannot move a character whatever Google does with the write mode, which is
// what lets this writer skip the capability probe.
func TestBatchIsOneInsertCommentUnderSuggest(t *testing.T) {
	body := decodeBatch(t, Batch(span(quoteStart, quoteEnd), "🤖 The 2026 register says quarterly.", ""))

	wc, ok := body["writeControl"].(map[string]any)
	if !ok || len(wc) != 1 || wc["writeMode"] != "SUGGEST" {
		t.Fatalf("writeControl = %v, want exactly {\"writeMode\":\"SUGGEST\"}", body["writeControl"])
	}
	reqs, ok := body["requests"].([]any)
	if !ok || len(reqs) != 1 {
		t.Fatalf("requests = %v, want one", body["requests"])
	}
	only, ok := reqs[0].(map[string]any)
	if !ok || len(only) != 1 {
		t.Fatalf("the one request is %v, and it carries one kind", reqs[0])
	}
	com, ok := only["insertComment"].(map[string]any)
	if !ok {
		t.Fatalf("the one request is %v, want an insertComment", only)
	}
	if com["content"] != "🤖 The 2026 register says quarterly." {
		t.Errorf("content = %v", com["content"])
	}
	rng := com["range"].(map[string]any)
	if rng["startIndex"] != float64(quoteStart) || rng["endIndex"] != float64(quoteEnd) {
		t.Errorf("range = %v, want %d..%d, the words themselves", rng, quoteStart, quoteEnd)
	}
	if _, ok := com["assigneeEmailAddress"]; ok {
		t.Error("assigneeEmailAddress is present on an annotation that named no assignee")
	}
}

func TestBatchCarriesTheAssigneeWhenThereIsOne(t *testing.T) {
	body := decodeBatch(t, Batch(span(quoteStart, quoteEnd), "🤖 why", "nail@altery.com"))

	com := body["requests"].([]any)[0].(map[string]any)["insertComment"].(map[string]any)
	if com["assigneeEmailAddress"] != "nail@altery.com" {
		t.Errorf("assigneeEmailAddress = %v", com["assigneeEmailAddress"])
	}
}

// TestTheGuardCarriesTheAnnotateBatchOnAHandedInDocument is the other half of
// the write bar, and the reason this milestone moves no policy. The document is
// handed in, so it sits at the suggest level, and the guard carries the batch
// only because the body says SUGGEST exactly and insertComment is a request
// kind it does not refuse. Nothing is granted here: this is the plain handed-in
// case, the one every review session is.
func TestTheGuardCarriesTheAnnotateBatchOnAHandedInDocument(t *testing.T) {
	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)
	u, err := url.Parse(propose.BatchURL(testDocID))
	if err != nil {
		t.Fatal(err)
	}
	raw := Batch(span(quoteStart, quoteEnd), Body(testAnnotation.Why), "")

	if err := p.Judge("POST", u, raw); err != nil {
		t.Fatalf("the guard refused the batch annotate really builds: %v", err)
	}
}

// TestTheGuardRefusesTheSameBatchWithoutSuggestMode states what the bar is made
// of. Take writeControl out and the identical request is a direct edit, which
// the guard refuses on a document gdoc was handed.
func TestTheGuardRefusesTheSameBatchWithoutSuggestMode(t *testing.T) {
	body := decodeBatch(t, Batch(span(quoteStart, quoteEnd), Body(testAnnotation.Why), ""))
	delete(body, "writeControl")
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)
	u, err := url.Parse(propose.BatchURL(testDocID))
	if err != nil {
		t.Fatal(err)
	}
	err = p.Judge("POST", u, raw)
	if err == nil {
		t.Fatal("the batch was carried on a handed-in document without SUGGEST")
	}
	if !strings.Contains(err.Error(), "SUGGEST") {
		t.Errorf("refusal = %q, and it should name SUGGEST", err)
	}
}

// TestApplySendsTheBatchAtTheSpanItFound is the order the whole design rests
// on: the document is read first and the write is built from that read, so no
// index outlives the answer it was computed in.
func TestApplySendsTheBatchAtTheSpanItFound(t *testing.T) {
	f := script(t, "before.json", "batch-saved.json")

	if _, err := Apply(context.Background(), f, testDocID, testAnnotation); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.calls) == 0 || f.calls[0].method != "GET" {
		t.Fatalf("calls = %v, and the first is the fresh read", f.calls)
	}
	sent := posts(f)
	if len(sent) != 1 {
		t.Fatalf("POSTs = %d, want exactly one batchUpdate", len(sent))
	}
	if sent[0].url != propose.BatchURL(testDocID) {
		t.Errorf("the write went to %q", sent[0].url)
	}
	want := string(Batch(span(quoteStart, quoteEnd), Body(testAnnotation.Why), ""))
	if got := string(sent[0].body); got != want {
		t.Errorf("the body sent was\n%s\nwant\n%s", got, want)
	}
}

// TestApplyRefusesADocumentWithMoreThanOneTabBeforeAnyWrite is the limit stated
// rather than guessed at. A range means nothing without saying which tab it is
// in, and propose refuses the same document for the same reason.
func TestApplyRefusesADocumentWithMoreThanOneTabBeforeAnyWrite(t *testing.T) {
	f := script(t, "two-tabs.json", "batch-saved.json")

	_, err := Apply(context.Background(), f, testDocID, testAnnotation)
	if err == nil {
		t.Fatal("a two-tab document was written to")
	}
	if !strings.Contains(err.Error(), "2 tabs") {
		t.Errorf("error = %q, and it should name the tab count", err)
	}
	if len(posts(f)) != 0 {
		t.Fatalf("a write reached the wire: %v", f.calls)
	}
}

// TestApplyReadsTheCommentIdFromTheThreadFirst is the shape Docs really answers
// with, measured for propose on 2026-09-07: the id sits under
// insertComment.commentThread.commentId. The flat field was the guess made
// before that measurement, and it stays as a fallback because reading one more
// field costs nothing.
func TestApplyReadsTheCommentIdFromTheThreadFirst(t *testing.T) {
	for _, c := range []struct {
		fixture string
		want    string
		listing string
	}{
		{"batch-saved.json", "AAACThReAd", "comments.json"},
		{"batch-saved-flat.json", "AAACFlAt", "comments-flat.json"},
	} {
		t.Run(c.fixture, func(t *testing.T) {
			f := script(t, "before.json", c.fixture)
			// The listing is the one Drive would answer with after this write,
			// so the read-backs hold and the only thing under test is the id.
			f.listing = fixture(t, c.listing)

			res, err := Apply(context.Background(), f, testDocID, testAnnotation)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.CommentID != c.want {
				t.Errorf("comment id = %q, want %q", res.CommentID, c.want)
			}
			if res.CommentUpdateState != "ALL_SAVED" {
				t.Errorf("state = %q", res.CommentUpdateState)
			}
			if res.Quoted != testAnnotation.Quoted {
				t.Errorf("result = %+v, and it should carry the words it commented on", res)
			}
			if len(res.Warnings) != 0 {
				t.Errorf("warnings = %v, and nothing went wrong", res.Warnings)
			}
		})
	}
}

// TestAnUpdateStateOtherThanSavedIsAWarning is the field that says the comment
// was lost. The status code says nothing about it, and nothing raises after the
// write: the batch went out, and a caller told the run failed is a caller that
// sends it again.
func TestAnUpdateStateOtherThanSavedIsAWarning(t *testing.T) {
	f := script(t, "before.json", "batch-failed.json")

	res, err := Apply(context.Background(), f, testDocID, testAnnotation)
	if err != nil {
		t.Fatalf("an answer saying the comment was lost was raised rather than reported: %v", err)
	}
	if res.CommentUpdateState != "ALL_FAILED_UNKNOWN_REASON" {
		t.Errorf("state = %q", res.CommentUpdateState)
	}
	if res.Verified {
		t.Error("Verified = true on an answer that says the comment was lost")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "ALL_FAILED_UNKNOWN_REASON") {
		t.Errorf("warnings = %v, and one should name the state", res.Warnings)
	}
}

// TestApplyRefusesBeforeTheWriteAndSendsNothing is the shape rule and the span
// walk, each stopping a run that has changed nothing. A refusal after the batch
// has gone out is a comment nobody asked for sitting in somebody's document.
func TestApplyRefusesBeforeTheWriteAndSendsNothing(t *testing.T) {
	for _, c := range []struct {
		name  string
		a     Annotation
		names string
	}{
		{
			name:  "a shape the rule refuses",
			a:     Annotation{Quoted: "reviewed annually", Why: ""},
			names: "no reason",
		},
		{
			name:  "a quote that is not in the document",
			a:     Annotation{Quoted: "reviewed monthly", Why: "The 2026 register says quarterly."},
			names: "was not found",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := script(t, "before.json", "batch-saved.json")

			_, err := Apply(context.Background(), f, testDocID, c.a)
			if err == nil {
				t.Fatalf("%s was written", c.name)
			}
			if !strings.Contains(err.Error(), c.names) {
				t.Errorf("error = %q, and it should say %q", err, c.names)
			}
			if len(posts(f)) != 0 {
				t.Fatalf("a write reached the wire: %v", f.calls)
			}
		})
	}
}

// TestApplyRefusesABadShapeBeforeReadingAnything is the other half of the rule
// above: the shape is answered from the annotation alone, so a bad entry costs
// no request at all. It is what lets a caller check a whole file before the
// first entry leaves the machine.
func TestApplyRefusesABadShapeBeforeReadingAnything(t *testing.T) {
	f := script(t, "before.json", "batch-saved.json")

	_, err := Apply(context.Background(), f, testDocID, Annotation{Quoted: "", Why: "so"})
	if err == nil {
		t.Fatal("an annotation quoting nothing was written")
	}
	if len(f.calls) != 0 {
		t.Errorf("the shape was checked after %d requests, and it needs no request at all", len(f.calls))
	}
}

// TestAGuardRefusalIsAnErrorAndTheWireSawNoBatch is the failure that never
// reached Google. The guard answers before the request goes out, so nothing is
// in the document and the caller is told plainly.
func TestAGuardRefusalIsAnErrorAndTheWireSawNoBatch(t *testing.T) {
	f := script(t, "before.json", "batch-saved.json")
	f.failAt[1] = errors.New("refused by policy: document was not given to this command")

	res, err := Apply(context.Background(), f, testDocID, testAnnotation)
	if err == nil {
		t.Fatal("a refused write was reported as a comment that landed")
	}
	if !strings.Contains(err.Error(), "refused by policy") {
		t.Errorf("error = %q, and it should carry the refusal", err)
	}
	if res.CommentID != "" {
		t.Errorf("comment id = %q on a write that never left", res.CommentID)
	}
}
