package annotate

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// theBody is the comment as it was sent, stated here as a literal rather than
// built from the constant: the read-backs compare against this string, and a
// test that reads the constant follows it wherever somebody moves it.
const theBody = "🤖 The 2026 register says quarterly."

// exportOf is the docx the fake hands back, two readable XML parts zipped in
// memory the way internal/docx's own tests build one.
func exportOf(t *testing.T, document, comments string) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"word/document.xml": document,
		"word/comments.xml": comments,
	} {
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

// detachedXML is the same paragraph with no comment range in it: the comment is
// in the export and it hangs off nothing. It is what a destroyed anchor looks
// like, and the one thing Drive's listing cannot tell anybody.
const detachedXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:t>The supplier register is reviewed annually by the operations team.</w:t></w:r>
      <w:r><w:commentReference w:id="0"/></w:r>
    </w:p>
  </w:body>
</w:document>`

// TestVerifiedIsBothRoutesHolding is the run where nothing went wrong. Drive's
// listing carries the id with the words that were sent and the quote as the
// anchor, and the export wraps those words around the quote, so both checks
// hold and the result claims it.
func TestVerifiedIsBothRoutesHolding(t *testing.T) {
	f := script(t, "before.json", "batch-saved.json")

	res, err := Apply(context.Background(), f, testDocID, testAnnotation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Checks != (Checks{DriveListing: true, DocxAnchored: true}) {
		t.Errorf("checks = %+v, warnings = %v", res.Checks, res.Warnings)
	}
	if !res.Verified {
		t.Errorf("verified = false where both routes held: %v", res.Warnings)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v on a run where everything held", res.Warnings)
	}
}

// TestAListingWithoutTheIdIsAWarningNotAnError is the first route failing. The
// comment is already in the document by the time this is read, so a caller told
// the run failed is a caller that writes the comment a second time.
func TestAListingWithoutTheIdIsAWarningNotAnError(t *testing.T) {
	f := script(t, "before.json", "batch-saved.json")
	f.listing = []byte(`{"comments": []}`)

	res, err := Apply(context.Background(), f, testDocID, testAnnotation)
	if err != nil {
		t.Fatalf("a read-back that did not hold was raised rather than reported: %v", err)
	}
	if res.Checks.DriveListing {
		t.Error("drive_listing = true where the listing carried no such comment")
	}
	if res.Verified {
		t.Error("verified = true on one route")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "AAACThReAd") {
		t.Errorf("warnings = %v, and one should name the id that was not there", res.Warnings)
	}
}

// TestAListingCarryingOtherWordsIsAWarning is the same route answering about
// the words rather than the id. The comment exists and reads as something else,
// which is a fact about what is in the document and not a reason to fail.
func TestAListingCarryingOtherWordsIsAWarning(t *testing.T) {
	f := script(t, "before.json", "batch-saved.json")
	f.listing = []byte(`{"comments": [{"id": "AAACThReAd", "content": "something else entirely",
		"quotedFileContent": {"value": "reviewed annually"}}]}`)

	res, err := Apply(context.Background(), f, testDocID, testAnnotation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Checks.DriveListing {
		t.Error("drive_listing = true where the listing carried other words")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "something else entirely") {
		t.Errorf("warnings = %v, and one should say what the listing carries", res.Warnings)
	}
}

// TestAListingAnchoredToOtherWordsIsAWarning is the anchor as Drive reports it.
// It is the weaker of the two anchor answers, because Drive keeps reporting the
// words a destroyed anchor used to hold, and it still catches a comment placed
// on the wrong text.
func TestAListingAnchoredToOtherWordsIsAWarning(t *testing.T) {
	f := script(t, "before.json", "batch-saved.json")
	f.listing = []byte(`{"comments": [{"id": "AAACThReAd", "content": "🤖 The 2026 register says quarterly.",
		"quotedFileContent": {"value": "the operations team"}}]}`)

	res, err := Apply(context.Background(), f, testDocID, testAnnotation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Checks.DriveListing {
		t.Error("drive_listing = true where the listing anchored the comment elsewhere")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "the operations team") {
		t.Errorf("warnings = %v, and one should say what the anchor holds", res.Warnings)
	}
}

// TestAnExportThatDoesNotAnchorTheCommentIsAWarning is the route that exists
// for this one case. The comment is in the export and it is attached to nothing,
// which Drive's listing reports as a healthy comment.
func TestAnExportThatDoesNotAnchorTheCommentIsAWarning(t *testing.T) {
	f := script(t, "before.json", "batch-saved.json")
	f.export = exportOf(t, detachedXML, string(fixture(t, "comments.xml")))

	res, err := Apply(context.Background(), f, testDocID, testAnnotation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Checks.DocxAnchored {
		t.Error("docx_anchored = true where the export attached the comment to no text")
	}
	if !res.Checks.DriveListing {
		t.Errorf("drive_listing = false, and that route held: %v", res.Warnings)
	}
	if res.Verified {
		t.Error("verified = true on a comment the export says is attached to nothing")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "not attached to any text") {
		t.Errorf("warnings = %v, and one should say it hangs off nothing", res.Warnings)
	}
}

// TestAnExportAnchoredToOtherWordsIsAWarning is the export answering the
// question the listing cannot: the comment is attached, and to the wrong words.
func TestAnExportAnchoredToOtherWordsIsAWarning(t *testing.T) {
	const elsewhereXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:t xml:space="preserve">The supplier register is reviewed annually by </w:t></w:r>
      <w:commentRangeStart w:id="0"/>
      <w:r><w:t>the operations team</w:t></w:r>
      <w:commentRangeEnd w:id="0"/>
      <w:r><w:commentReference w:id="0"/></w:r>
    </w:p>
  </w:body>
</w:document>`
	f := script(t, "before.json", "batch-saved.json")
	f.export = exportOf(t, elsewhereXML, string(fixture(t, "comments.xml")))

	res, err := Apply(context.Background(), f, testDocID, testAnnotation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Checks.DocxAnchored {
		t.Error("docx_anchored = true where the export anchored the comment to other words")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "the operations team") {
		t.Errorf("warnings = %v, and one should say what the export wrapped", res.Warnings)
	}
}

// TestTwoExportedCommentsThatDisagreeGiveNoAnswer is the ambiguity rule, the
// one internal/docx holds on the same join. The export carries no Drive comment
// id, so two comments reading the same words cannot be told apart, and taking
// the first would report this comment on the strength of another one.
func TestTwoExportedCommentsThatDisagreeGiveNoAnswer(t *testing.T) {
	const twoXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:t xml:space="preserve">The supplier register is </w:t></w:r>
      <w:commentRangeStart w:id="0"/>
      <w:r><w:t>reviewed annually</w:t></w:r>
      <w:commentRangeEnd w:id="0"/>
      <w:r><w:commentReference w:id="0"/></w:r>
      <w:r><w:t xml:space="preserve"> by the operations team.</w:t></w:r>
      <w:r><w:commentReference w:id="1"/></w:r>
    </w:p>
  </w:body>
</w:document>`
	const twoCommentsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:comment w:id="0" w:author="Nail Khusnullin" w:date="2026-09-18T09:00:00Z">
    <w:p><w:r><w:t>🤖 The 2026 register says quarterly.</w:t></w:r></w:p>
  </w:comment>
  <w:comment w:id="1" w:author="Nail Khusnullin" w:date="2026-09-18T09:01:00Z">
    <w:p><w:r><w:t>🤖 The 2026 register says quarterly.</w:t></w:r></w:p>
  </w:comment>
</w:comments>`
	f := script(t, "before.json", "batch-saved.json")
	f.export = exportOf(t, twoXML, twoCommentsXML)

	checks, warns := Verify(context.Background(), f, testDocID, "AAACThReAd", testAnnotation.Quoted, theBody)

	if checks.DocxAnchored {
		t.Error("docx_anchored = true where two comments read the same words and only one is attached")
	}
	if !strings.Contains(strings.Join(warns, " "), "do not agree") {
		t.Errorf("warnings = %v, and one should name the ambiguity", warns)
	}
}

// TestAReadBackThatFailedIsAWarningNamingTheRoute is each route unreachable.
// The other one still answers for itself: a run that lost one read-back knows
// more than a run that lost both, and the report says which.
func TestAReadBackThatFailedIsAWarningNamingTheRoute(t *testing.T) {
	for _, c := range []struct {
		name  string
		at    int
		names string
		other func(Checks) bool
	}{
		{
			name:  "Drive's listing",
			at:    2,
			names: "listing could not be read",
			other: func(c Checks) bool { return c.DocxAnchored },
		},
		{
			name:  "the export",
			at:    3,
			names: "export could not be read",
			other: func(c Checks) bool { return c.DriveListing },
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := script(t, "before.json", "batch-saved.json")
			f.failAt[c.at] = errors.New("the network went away")

			res, err := Apply(context.Background(), f, testDocID, testAnnotation)
			if err != nil {
				t.Fatalf("a read-back that could not be made was raised rather than reported: %v", err)
			}
			if res.Verified {
				t.Error("verified = true where a route could not be read")
			}
			if !strings.Contains(strings.Join(res.Warnings, " "), c.names) {
				t.Errorf("warnings = %v, and one should name %s", res.Warnings, c.name)
			}
			if !c.other(res.Checks) {
				t.Errorf("checks = %+v, and the other route still held: %v", res.Checks, res.Warnings)
			}
		})
	}
}

// TestAWriteWithNoCommentIdIsNotSearchedFor is the answer that carried no id.
// There is nothing to look for in Drive's listing then, and saying so is more
// honest than reporting a comment that is missing.
func TestAWriteWithNoCommentIdIsNotSearchedFor(t *testing.T) {
	checks, warns := Verify(context.Background(), script(t, "before.json", "batch-saved.json"),
		testDocID, "", testAnnotation.Quoted, theBody)

	if checks.DriveListing {
		t.Error("drive_listing = true with no id to look for")
	}
	if !strings.Contains(strings.Join(warns, " "), "no comment id") {
		t.Errorf("warnings = %v, and one should say there was no id to look for", warns)
	}
}

// TestVerifyReadsTheTwoRoutesTheWriteDidNotGoOutOn is why there are two of
// them. Both are reads, neither is the batchUpdate, and one of them is a
// different product answering about the same comment.
func TestVerifyReadsTheTwoRoutesTheWriteDidNotGoOutOn(t *testing.T) {
	f := script(t, "before.json", "batch-saved.json")

	if _, err := Apply(context.Background(), f, testDocID, testAnnotation); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var read []string
	for _, c := range f.calls[2:] {
		if c.method != "GET" {
			t.Fatalf("a %s reached the wire after the write: %v", c.method, c)
		}
		read = append(read, c.url)
	}
	if len(read) != 2 {
		t.Fatalf("reads after the write = %v, want Drive's listing and the export", read)
	}
	if !strings.Contains(read[0], "/comments?") {
		t.Errorf("the first read-back was %q, want Drive's comment listing", read[0])
	}
	if !strings.Contains(read[1], "/export?") {
		t.Errorf("the second read-back was %q, want the docx export", read[1])
	}
}
