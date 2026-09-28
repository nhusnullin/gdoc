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

// blockScript is Google after a block's write: one inline read-back, one
// preview read, and the export. VerifyBlock reads each of them once, so each is
// a single fixture rather than a queue.
func blockScript(t *testing.T, inline, preview string, export []byte) *fakeSession {
	t.Helper()
	return &fakeSession{
		inline:  [][]byte{fixture(t, inline)},
		preview: fixture(t, preview),
		export:  export,
		failAt:  map[int]error{},
	}
}

// blockExport is the export a block's write leaves behind: the robot comment
// gdoc wrote, attached to the words of the block's first new paragraph, which
// is where insertComment anchored it.
func blockExport(t *testing.T, body, on string) []byte {
	t.Helper()
	comments := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:comment w:id="0" w:author="Nail Khusnullin" w:date="2026-09-28T09:00:00Z">
    <w:p><w:r><w:t>` + body + `</w:t></w:r></w:p>
  </w:comment>
</w:comments>`
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:commentRangeStart w:id="0"/>
      <w:r><w:t>` + on + `</w:t></w:r>
      <w:commentRangeEnd w:id="0"/>
      <w:r><w:commentReference w:id="0"/></w:r>
    </w:p>
  </w:body>
</w:document>`
	return zipParts(t, map[string]string{"word/document.xml": document, "word/comments.xml": comments})
}

// afterPlace is the everyday block placement the read-back tests stand on: the
// four-paragraph block behind the anchor in block-body.json.
func afterPlace(t *testing.T) (Placement, []Para) {
	t.Helper()
	content := blockContent(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	return place, content
}

// replacePlace is the same block standing in for the two paragraphs from the
// anchor to the risk matrix.
func replacePlace(t *testing.T) (Placement, []Para) {
	t.Helper()
	content := blockContent(t)
	place, err := PlaceReplace(document(t, "block-body.json"), "reviewed annually", "risk matrix")
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	return place, content
}

func TestVerifyBlockHoldsOnAllThreeRoutes(t *testing.T) {
	place, content := afterPlace(t)
	f := blockScript(t, "block-after-inline.json", "block-body.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	checks, ids, warns := VerifyBlock(context.Background(), f, testDocID, place, content, Prefix+testWhy)

	if checks != (Checks{SuggestionsInline: true, PreviewWithoutSuggestions: true, DocxAnchored: true}) {
		t.Errorf("checks = %+v, warnings = %v", checks, warns)
	}
	if len(ids) != 1 || ids[0] != "suggest.block" {
		t.Errorf("ids = %v, want the one id every request of the block folded into", ids)
	}
	if len(warns) != 0 {
		t.Errorf("warnings = %v on a run where everything held", warns)
	}
}

// TestVerifyBlockHoldsForAReplace is the second half of the inline question. A
// replace has to be pending on both sides: the new paragraphs carrying an
// insertion id, and every paragraph it stands in for carrying a deletion id.
// An insertion alone would be the block added beside the words it replaces.
func TestVerifyBlockHoldsForAReplace(t *testing.T) {
	place, content := replacePlace(t)
	f := blockScript(t, "block-replace-inline.json", "block-body.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	checks, ids, warns := VerifyBlock(context.Background(), f, testDocID, place, content, Prefix+testWhy)

	if checks != (Checks{SuggestionsInline: true, PreviewWithoutSuggestions: true, DocxAnchored: true}) {
		t.Errorf("checks = %+v, warnings = %v", checks, warns)
	}
	if len(ids) != 1 || ids[0] != "suggest.block" {
		t.Errorf("ids = %v, want the one id the insert and the deletion folded into", ids)
	}
}

// TestVerifyBlockHoldsAfterTheLastParagraph reads back the one shape whose last
// new paragraph is not wholly inserted. MEASURED.md row 7: the text goes in
// before the body's final newline, so that newline stays the document's own and
// the block's last paragraph owns it. Expecting an inserted mark there would
// report the measured shape as a failure.
func TestVerifyBlockHoldsAfterTheLastParagraph(t *testing.T) {
	content := plainBlock(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "operations lead", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	f := blockScript(t, "block-at-end-inline.json", "block-body.json", blockExport(t, Prefix+testWhy, "Limits"))

	checks, ids, warns := VerifyBlock(context.Background(), f, testDocID, place, content, Prefix+testWhy)

	if checks != (Checks{SuggestionsInline: true, PreviewWithoutSuggestions: true, DocxAnchored: true}) {
		t.Errorf("checks = %+v, warnings = %v", checks, warns)
	}
	if len(ids) != 1 || ids[0] != "suggest.block" {
		t.Errorf("ids = %v", ids)
	}
}

// TestVerifyBlockCatchesANewParagraphThatIsNotSuggested is the inline check's
// whole point. A paragraph of the block that carries no insertion id is text
// somebody would find in the document with nothing to accept or reject, which
// is a direct edit of part of the block.
func TestVerifyBlockCatchesANewParagraphThatIsNotSuggested(t *testing.T) {
	place, content := afterPlace(t)
	f := blockScript(t, "block-after-inline-written.json", "block-body.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	checks, _, warns := VerifyBlock(context.Background(), f, testDocID, place, content, Prefix+testWhy)

	if checks.SuggestionsInline {
		t.Error("suggestions_inline = true where the block's last paragraph carries no insertion id")
	}
	if !strings.Contains(strings.Join(warns, " "), "index 129") {
		t.Errorf("warnings = %v, and one should name the index the paragraph was written at", warns)
	}
}

// TestVerifyBlockCatchesAReplaceThatDeletedNothing is the same question on the
// other side: the new paragraphs are pending and the old ones are untouched, so
// the document now says both.
func TestVerifyBlockCatchesAReplaceThatDeletedNothing(t *testing.T) {
	place, content := replacePlace(t)
	f := blockScript(t, "block-replace-inline-kept.json", "block-body.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	checks, _, warns := VerifyBlock(context.Background(), f, testDocID, place, content, Prefix+testWhy)

	if checks.SuggestionsInline {
		t.Error("suggestions_inline = true where the replaced paragraphs carry no deletion id")
	}
	if !strings.Contains(strings.Join(warns, " "), "suggested deletion") {
		t.Errorf("warnings = %v, and one should say the paragraph carries no suggested deletion", warns)
	}
}

// TestVerifyBlockReportsEverySuggestionIDAndWarnsAboutWithdraw is the decision
// that one id is what a verified block means. Every request of the batch folded
// into one id when it was measured, and the note records one id, so a read-back
// showing two is a proposal withdraw can only half take back. Both are reported,
// the first is the one the note will record, and the check is false.
func TestVerifyBlockReportsEverySuggestionIDAndWarnsAboutWithdraw(t *testing.T) {
	place, content := afterPlace(t)
	f := blockScript(t, "block-after-inline-two-ids.json", "block-body.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	checks, ids, warns := VerifyBlock(context.Background(), f, testDocID, place, content, Prefix+testWhy)

	if checks.SuggestionsInline {
		t.Error("suggestions_inline = true on a block that came back as two suggestions")
	}
	want := []string{"suggest.block", "suggest.blocktwo"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Errorf("ids = %v, want %v, the first one being the one the note records", ids, want)
	}
	if !strings.Contains(strings.Join(warns, " "), "withdraw") {
		t.Errorf("warnings = %v, and one should say withdraw takes back the first id only", warns)
	}
}

// TestVerifyBlockCatchesADirectEditInThePreview is the route that tells a
// suggestion from an edit. A replace Google made as a plain edit takes the old
// paragraphs out, so the preview, which shows the document with every pending
// suggestion hidden, no longer carries them.
func TestVerifyBlockCatchesADirectEditInThePreview(t *testing.T) {
	place, content := replacePlace(t)
	f := blockScript(t, "block-replace-inline.json", "block-preview-replaced.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	checks, _, warns := VerifyBlock(context.Background(), f, testDocID, place, content, Prefix+testWhy)

	if checks.PreviewWithoutSuggestions {
		t.Error("preview_without_suggestions = true on a preview that has lost the replaced paragraphs, which is a direct edit")
	}
	if !strings.Contains(strings.Join(warns, " "), "direct edit") {
		t.Errorf("warnings = %v, and one should name the direct edit", warns)
	}
}

// TestVerifyBlockHoldsForAReplaceThatKeepsItsFirstLine is the shape a rewritten
// section has: the replace covers a run of paragraphs and the content opens with
// the same line the run opened with. The question is asked about the first line
// the old run did not carry, so keeping the run's opening line does not fail the
// block, and the line behind it is the one that answers.
func TestVerifyBlockHoldsForAReplaceThatKeepsItsFirstLine(t *testing.T) {
	place, _ := replacePlace(t)
	kept, err := ParseContent("The supplier register is reviewed annually by the operations team.\n\nAnd a new sentence behind it.\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}
	// The preview is the document as it stood: both replaced paragraphs are
	// still there, because a suggested deletion leaves the words alone.
	holds, why := blockPreviewHolds(document(t, "block-body.json"), place, kept)

	if !holds {
		t.Errorf("preview_without_suggestions = false on a replace whose content keeps the run's first line: %s", why)
	}
}

// TestVerifyBlockGivesNoAnswerWhenAReplaceKeepsEveryOldParagraph is the shape
// the first question cannot answer on its own. Carries asks by substring, so a
// direct edit whose new paragraphs carry every replaced one inside them leaves
// all of them findable in the preview, and the first question holds over the
// silent direct edit this route exists to catch. The second question is what
// answers there: the line the old paragraphs did not carry is in the preview
// too, which a suggestion would have hidden.
func TestVerifyBlockGivesNoAnswerWhenAReplaceKeepsEveryOldParagraph(t *testing.T) {
	place, _ := replacePlace(t)
	kept, err := ParseContent("The supplier register is reviewed annually by the operations team.\n\n" +
		"Each supplier is scored against the risk matrix. The scores are published each quarter.\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}

	holds, why := blockPreviewHolds(document(t, "block-preview-edited-containing.json"), place, kept)

	if holds {
		t.Error("preview_without_suggestions = true on a preview carrying the replace's own new line, which is a direct edit")
	}
	if !strings.Contains(why, "no answer") || !strings.Contains(why, "published each quarter") {
		t.Errorf("the warning is %q, and it should say the route gives no answer and name the line", why)
	}
}

// TestVerifyBlockGivesNoAnswerWhenAReplaceOnlyReshapes is the replace with no
// line to ask about that the first question cannot answer for either: the
// content says every paragraph it stands on again, as a list, so a direct edit
// leaves all of them findable in the preview. Nothing distinguishes the edit
// from the suggestion here, and passing would report the one thing this route
// exists to catch as a route that held.
func TestVerifyBlockGivesNoAnswerWhenAReplaceOnlyReshapes(t *testing.T) {
	place, _ := replacePlace(t)
	listed, err := ParseContent("- The supplier register is reviewed annually by the operations team.\n" +
		"- Each supplier is scored against the risk matrix.\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}

	holds, why := blockPreviewHolds(document(t, "block-body.json"), place, listed)

	if holds {
		t.Error("preview_without_suggestions = true on a replace that says every old paragraph again, where a direct edit reads the same way")
	}
	if !strings.Contains(why, "no answer") {
		t.Errorf("the warning is %q, and it should say the route gives no answer", why)
	}
}

// TestVerifyBlockGivesNoAnswerWhenAnAfterBlockSaysNothingNew is the same gap on
// the other placement. An after block takes no paragraph out, so the first
// question holds after a direct edit too, and a block whose every line is
// inside its own anchor leaves the second question nothing to ask about.
func TestVerifyBlockGivesNoAnswerWhenAnAfterBlockSaysNothingNew(t *testing.T) {
	place, _ := afterPlace(t)
	inside, err := ParseContent("## The supplier register\n\nreviewed annually by the operations team\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}

	holds, why := blockPreviewHolds(document(t, "block-body.json"), place, inside)

	if holds {
		t.Error("preview_without_suggestions = true on an after block whose every line is inside its anchor, where a direct edit reads the same way")
	}
	if !strings.Contains(why, "no answer") {
		t.Errorf("the warning is %q, and it should say the route gives no answer", why)
	}
}

// TestVerifyBlockHoldsWhenAReplaceOnlyShortens is the replace whose content
// says nothing its own old paragraphs did not say already. There is no new
// line to ask about, and that is not a failure: a direct edit of it takes the
// old paragraphs out, which is the first question's own answer.
func TestVerifyBlockHoldsWhenAReplaceOnlyShortens(t *testing.T) {
	place, _ := replacePlace(t)
	shorter, err := ParseContent("The supplier register is reviewed annually by the operations team.\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}

	holds, why := blockPreviewHolds(document(t, "block-body.json"), place, shorter)

	if !holds {
		t.Errorf("preview_without_suggestions = false on a replace that only shortens: %s", why)
	}
}

// TestVerifyBlockGivesNoAnswerWhenThePreviewCarriesTheFirstLine is the
// ambiguity this route has to own. The block's first line being in the preview
// is what a direct edit looks like, and it is also what a document that already
// carried that line looks like. The two cannot be told apart from here, so the
// check is false with a warning naming both readings rather than a pass.
func TestVerifyBlockGivesNoAnswerWhenThePreviewCarriesTheFirstLine(t *testing.T) {
	place, content := afterPlace(t)
	f := blockScript(t, "block-after-inline.json", "block-preview-elsewhere.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	checks, _, warns := VerifyBlock(context.Background(), f, testDocID, place, content, Prefix+testWhy)

	if checks.PreviewWithoutSuggestions {
		t.Error("preview_without_suggestions = true where the preview carries the block's first line")
	}
	joined := strings.Join(warns, " ")
	if !strings.Contains(joined, "no answer") || !strings.Contains(joined, "3.6 Limits") {
		t.Errorf("warnings = %v, and one should say the route gives no answer and name the line", warns)
	}
}
