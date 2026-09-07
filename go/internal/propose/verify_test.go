package propose

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// call is one request the fake was asked to make, so a test can assert the
// order of the routes as well as what came back from them.
type call struct {
	method string
	url    string
	body   json.RawMessage
}

// exportShape says whether the docx the fake hands back carries the comment the
// proposal made. It is the third read-back's only variable.
type exportShape int

const (
	withComment exportShape = iota
	withoutComment
)

// fakeSession is Google as far as this package is concerned: two Docs reads at
// the same URL, one at the preview URL, one export, and one batchUpdate.
//
// The inline read is answered from a queue rather than from a single fixture,
// because Apply reads that URL twice: once to find the span and once to see
// what the write did to it. A fake that answered the same bytes both times
// would verify the document as it was before the write.
//
// Nothing here names net/http: the rooms that may are the ones the boundary
// test lists, and this is not one of them.
type fakeSession struct {
	calls   []call
	inline  [][]byte
	preview []byte
	batch   []byte
	export  []byte
	failAt  map[int]error
	// afterBatch is a failure raised once the answer has been decoded, which is
	// the one shape that reaches Apply with fields in hand: valid JSON of the
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
	var answer []byte
	switch {
	case rawURL == docs.URL(testDocID):
		if len(f.inline) == 0 {
			return errors.New("the fake was asked for a third inline read")
		}
		answer, f.inline = f.inline[0], f.inline[1:]
	case strings.Contains(rawURL, "PREVIEW_WITHOUT_SUGGESTIONS"):
		answer = f.preview
	default:
		answer = []byte(`{}`)
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
	if err := json.Unmarshal(f.batch, into); err != nil {
		return err
	}
	return f.afterBatch
}

// script is a Google that answers every route, each from a named fixture.
func script(t *testing.T, before, after, preview, batch string, shape exportShape) *fakeSession {
	t.Helper()
	return &fakeSession{
		inline:  [][]byte{fixture(t, before), fixture(t, after)},
		preview: fixture(t, preview),
		batch:   fixture(t, batch),
		export:  exportDocx(t, shape),
		failAt:  map[int]error{},
	}
}

func span(start, end int) docs.Range {
	return docs.Range{Tab: "t.0", Start: start, End: end}
}

// exportDocx builds the export in memory, two readable XML parts rather than a
// binary blob, the way internal/docx's own tests do.
func exportDocx(t *testing.T, shape exportShape) []byte {
	t.Helper()
	const commentsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:comment w:id="0" w:author="Nail Khusnullin" w:date="2026-09-07T09:00:00Z">
    <w:p><w:r><w:t>🤖 the policy says twice a year</w:t></w:r></w:p>
  </w:comment>
</w:comments>`
	const anchoredXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:t xml:space="preserve">The supplier register is </w:t></w:r>
      <w:commentRangeStart w:id="0"/>
      <w:r><w:t>reviewed every six months</w:t></w:r>
      <w:commentRangeEnd w:id="0"/>
      <w:r><w:commentReference w:id="0"/></w:r>
    </w:p>
  </w:body>
</w:document>`
	const bareXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body><w:p><w:r><w:t>The supplier register is reviewed annually.</w:t></w:r></w:p></w:body>
</w:document>`

	parts := map[string]string{"word/document.xml": anchoredXML, "word/comments.xml": commentsXML}
	if shape == withoutComment {
		parts = map[string]string{"word/document.xml": bareXML}
	}
	return zipParts(t, parts)
}

// zipParts is the docx a fixture describes, as bytes.
func zipParts(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestVerifyHoldsOnAllThreeRoutes(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
	f.inline = f.inline[1:] // Verify makes the read-back read, not the first one.

	checks, ids, warns := Verify(context.Background(), f, testDocID, span(26, 43), testProposal, "🤖 "+testProposal.Why)

	if checks != (Checks{SuggestionsInline: true, PreviewWithoutSuggestions: true, DocxAnchored: true}) {
		t.Errorf("checks = %+v, warnings = %v", checks, warns)
	}
	if len(ids) != 1 || ids[0] != "suggest.abc" {
		t.Errorf("ids = %v", ids)
	}
	if len(warns) != 0 {
		t.Errorf("warnings = %v on a run where everything held", warns)
	}
}

// TestVerifyReadsThePreviewThroughItsOwnView is what tells a suggestion from an
// edit. The two reads are the same document through two views, and only the
// second one answers the question the guard cannot.
func TestVerifyReadsThePreviewThroughItsOwnView(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
	f.inline = f.inline[1:]

	Verify(context.Background(), f, testDocID, span(26, 43), testProposal, "🤖 why")

	var sawInline, sawPreview, sawExport bool
	for _, c := range f.calls {
		switch {
		case c.url == docs.URL(testDocID):
			sawInline = true
		case strings.Contains(c.url, "PREVIEW_WITHOUT_SUGGESTIONS"):
			sawPreview = true
		case strings.Contains(c.url, "/export?"):
			sawExport = true
		}
	}
	if !sawInline || !sawPreview || !sawExport {
		t.Errorf("routes read: inline=%v preview=%v export=%v", sawInline, sawPreview, sawExport)
	}
}

// TestVerifyReadsThePreviewByWordsRatherThanAtTheIndex is the defect that made
// the second proposal of every run report a direct edit. r.Start is counted in
// the view that shows pending suggestions; the preview hides them, so every
// position after one sits lower there. The span here begins seven units past the
// quoted words, which is where an earlier pending insertion of "really " would
// have put it, and the preview plainly carries the words. The check has to find
// them: this is the one route that names a silent direct edit, and a route that
// cries wolf is one people stop reading.
func TestVerifyReadsThePreviewByWordsRatherThanAtTheIndex(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
	f.inline = f.inline[1:]

	checks, _, warns := Verify(context.Background(), f, testDocID, span(33, 50), testProposal, "🤖 "+testProposal.Why)

	if !checks.PreviewWithoutSuggestions {
		t.Errorf("PreviewWithoutSuggestions = false where the preview carries %q: %v", testProposal.Quoted, warns)
	}
}

// TestVerifyStillCatchesADirectEditInThePreview is the half that must not be
// lost with the index. A write Google made as a plain edit leaves the
// replacement where the quoted words were, and FindSpan required those words to
// occur exactly once, so the preview no longer carries them anywhere.
func TestVerifyStillCatchesADirectEditInThePreview(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview-edited.json", "batch-saved.json", withComment)
	f.inline = f.inline[1:]

	checks, _, warns := Verify(context.Background(), f, testDocID, span(26, 43), testProposal, "🤖 "+testProposal.Why)

	if checks.PreviewWithoutSuggestions {
		t.Error("PreviewWithoutSuggestions = true on a preview carrying the replacement, which is a direct edit")
	}
	if !strings.Contains(strings.Join(warns, " "), "direct edit") {
		t.Errorf("warnings = %v, and one should name the direct edit", warns)
	}
}

// addWords is the proposal shape whose replacement carries the quoted words
// inside it, which is what a caller writes after FindSpan tells them to quote
// more of the sentence.
var addWords = Proposal{
	Quoted:      "reviewed annually",
	Replacement: "reviewed annually by the operations team",
	Why:         "say who reviews it",
}

// TestVerifyCatchesADirectEditWhoseReplacementCarriesTheQuote is the hole a
// by-words search opens on its own. The write deletes the quoted words and puts
// the replacement in their place, so a replacement containing them leaves them
// in the preview after a direct edit as well as after a suggestion. Asking only
// for the quote would report the silent direct edit as the route holding, on
// the one route that exists to catch it.
func TestVerifyCatchesADirectEditWhoseReplacementCarriesTheQuote(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview-edited-containing.json", "batch-saved.json", withComment)
	f.inline = f.inline[1:]

	checks, _, warns := Verify(context.Background(), f, testDocID, span(26, 43), addWords, "🤖 "+addWords.Why)

	if checks.PreviewWithoutSuggestions {
		t.Error("PreviewWithoutSuggestions = true on a preview carrying the replacement, which is a direct edit the quote alone cannot see")
	}
	if !strings.Contains(strings.Join(warns, " "), addWords.Replacement) {
		t.Errorf("warnings = %v, and one should name the replacement the preview carries", warns)
	}
}

// TestVerifyDoesNotCryWolfOnAnHonestAddWordsProposal is the other half. The
// preview hides the insertion, so an honest suggestion of the same shape leaves
// the quoted words and no replacement, and the route has to hold: a check that
// failed on every add-words proposal is one people stop reading.
func TestVerifyDoesNotCryWolfOnAnHonestAddWordsProposal(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview-plain.json", "batch-saved.json", withComment)
	f.inline = f.inline[1:]

	checks, _, warns := Verify(context.Background(), f, testDocID, span(26, 43), addWords, "🤖 "+addWords.Why)

	if !checks.PreviewWithoutSuggestions {
		t.Errorf("PreviewWithoutSuggestions = false where the preview carries the quoted words and not the replacement: %v", warns)
	}
}

// TestDocxHoldsRefusesTwoCommentsThatDisagree is the ambiguity rule, on the join
// internal/docx already states it for. The export carries no Drive comment id,
// so two proposals in one run carrying the same reason are two comments reading
// the same words. Taking the first would report one proposal on the strength of
// the other's comment, so neither answers.
func TestDocxHoldsRefusesTwoCommentsThatDisagree(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
	f.export = twoCommentsDocx(t)

	held, why := docxHolds(context.Background(), f, testDocID, "🤖 "+testProposal.Why)

	if held {
		t.Error("docx_anchored = true where two comments read the same words and only one is attached")
	}
	if !strings.Contains(why, "do not agree") {
		t.Errorf("warning = %q, and it should say the two comments do not agree", why)
	}
}

// twoCommentsDocx is an export carrying the same comment body twice, one
// anchored and one floating. It is the shape two proposals with one reason make.
func twoCommentsDocx(t *testing.T) []byte {
	t.Helper()
	const commentsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:comment w:id="0" w:author="Nail Khusnullin" w:date="2026-09-07T09:00:00Z">
    <w:p><w:r><w:t>🤖 the policy says twice a year</w:t></w:r></w:p>
  </w:comment>
  <w:comment w:id="1" w:author="Nail Khusnullin" w:date="2026-09-07T09:01:00Z">
    <w:p><w:r><w:t>🤖 the policy says twice a year</w:t></w:r></w:p>
  </w:comment>
</w:comments>`
	const documentXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:commentRangeStart w:id="0"/>
      <w:r><w:t>reviewed every six months</w:t></w:r>
      <w:commentRangeEnd w:id="0"/>
      <w:r><w:commentReference w:id="0"/></w:r>
      <w:r><w:commentReference w:id="1"/></w:r>
    </w:p>
  </w:body>
</w:document>`
	return zipParts(t, map[string]string{"word/document.xml": documentXML, "word/comments.xml": commentsXML})
}

// TestVerifyWarnsWhenTheExportCouldNotBeRead keeps a failed read out of the
// verdict's way. An export that did not come back says nothing about whether
// the comment is anchored, and reporting it as anchored would be a fact that
// is not true.
func TestVerifyWarnsWhenTheExportCouldNotBeRead(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
	f.inline = f.inline[1:]
	f.failAt[2] = errors.New("the export timed out")

	checks, _, warns := Verify(context.Background(), f, testDocID, span(26, 43), testProposal, "🤖 "+testProposal.Why)

	if checks.DocxAnchored {
		t.Error("DocxAnchored = true on an export that never arrived")
	}
	if !strings.Contains(strings.Join(warns, " "), "export") {
		t.Errorf("warnings = %v, and one should name the export", warns)
	}
}

// PreviewURL is the only route that can tell a suggestion from a direct edit,
// so what it asks for is stated here rather than left to the fake's substring
// match on the view mode. Both parameters carry weight: the view mode is the
// question being asked, and includeTabsContent is what makes the answer carry
// any text at all, which is also what the guard's allowlist requires of a Docs
// read. A third parameter would be refused by the guard, so the count is
// asserted too.
func TestPreviewURLAsksForThePreviewViewOfEveryTab(t *testing.T) {
	u, err := url.Parse(PreviewURL(testDocID))
	if err != nil {
		t.Fatalf("PreviewURL() is not a URL: %v", err)
	}
	if u.Host != "docs.googleapis.com" || u.Path != "/v1/documents/"+testDocID {
		t.Errorf("PreviewURL() addresses %q%q, want the Docs read of %s", u.Host, u.Path, testDocID)
	}
	q := u.Query()
	if got := q.Get("suggestionsViewMode"); got != "PREVIEW_WITHOUT_SUGGESTIONS" {
		t.Errorf("suggestionsViewMode = %q, want PREVIEW_WITHOUT_SUGGESTIONS", got)
	}
	if got := q.Get("includeTabsContent"); got != "true" {
		t.Errorf("includeTabsContent = %q, want true", got)
	}
	if len(q) != 2 {
		t.Errorf("query = %v, want those two parameters and nothing else", q)
	}
	if PreviewURL(testDocID) == docs.URL(testDocID) {
		t.Error("PreviewURL() is the inline read, so the second route reads what the first one did")
	}
}
