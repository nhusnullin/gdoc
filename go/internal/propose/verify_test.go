package propose

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	return json.Unmarshal(f.batch, into)
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
