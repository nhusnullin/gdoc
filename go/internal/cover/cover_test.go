package cover

import (
	"errors"
	"strings"
	"testing"
)

// note builds a markdown file out of a front-matter block and a body, so a
// test states only the keys it is about.
func note(frontMatter, body string) []byte {
	return []byte("---\n" + frontMatter + "\n---\n" + body)
}

func read(t *testing.T, src []byte) (Fields, []byte) {
	t.Helper()
	f, body, err := Read(src)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return f, body
}

func TestANoteWithOnlyATitleReadsWithNumberingOnAndNoRevisions(t *testing.T) {
	f, body := read(t, note("title: Supplier Register Policy", "# Purpose\n"))

	if f.Title != "Supplier Register Policy" {
		t.Errorf("title = %q, want Supplier Register Policy", f.Title)
	}
	if !f.HeadingNumbering {
		t.Error("heading numbering is off, want on when the note says nothing")
	}
	if len(f.Revisions) != 0 {
		t.Errorf("revisions = %d, want none", len(f.Revisions))
	}
	if f.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", f.Version)
	}
	if f.Classification != "Internal (I)" {
		t.Errorf("classification = %q, want Internal (I)", f.Classification)
	}
	if f.RunningHead() != "Altery - Supplier Register Policy" {
		t.Errorf("running head = %q, want Altery - Supplier Register Policy", f.RunningHead())
	}
	if string(body) != "# Purpose\n" {
		t.Errorf("body = %q, want the markdown behind the front matter", body)
	}
}

func TestEveryOptionalKeyReads(t *testing.T) {
	f, _ := read(t, note(strings.Join([]string{
		"title: Third Party Risk",
		"alt_title: Supplier Risk",
		"doc_type: Policy",
		"version: 1.0",
		"date: 2026-09-08",
		"owner: Head of Risk",
		"classification: restricted",
		"heading_numbering: none",
		"revisions:",
		"  - version: 0.9",
		"    date: 2026-08-01",
		"    author: Nail Khusnullin",
		"    approved_by: The Board",
		"    approval_date: 2026-08-15",
		"    section: All",
		"    change: First draft",
	}, "\n"), "Body\n"))

	if f.AltTitle != "Supplier Risk" {
		t.Errorf("alt_title = %q, want Supplier Risk", f.AltTitle)
	}
	if f.DocType != "Policy" {
		t.Errorf("doc_type = %q, want Policy", f.DocType)
	}
	if f.Version != "1.0" {
		t.Errorf("version = %q, want 1.0 read as written and not as a number", f.Version)
	}
	if f.Owner != "Head of Risk" {
		t.Errorf("owner = %q, want Head of Risk", f.Owner)
	}
	if f.Classification != "Restricted (R)" {
		t.Errorf("classification = %q, want Restricted (R)", f.Classification)
	}
	if f.HeadingNumbering {
		t.Error("heading numbering is on, want off when the note says none")
	}
	if f.CoverTitle() != "Third Party Risk Policy" {
		t.Errorf("cover title = %q, want Third Party Risk Policy", f.CoverTitle())
	}
	if f.CoverAltTitle() != "Supplier Risk Policy" {
		t.Errorf("cover alt title = %q, want Supplier Risk Policy", f.CoverAltTitle())
	}
	if f.RunningHead() != "Altery - Supplier Risk Policy" {
		t.Errorf("running head = %q, want the alt title's", f.RunningHead())
	}
	if len(f.Revisions) != 1 {
		t.Fatalf("revisions = %d, want 1", len(f.Revisions))
	}
	want := []string{"0.9", "1 August 2026", "Nail Khusnullin", "The Board",
		"15 August 2026", "All", "First draft"}
	got := f.Revisions[0].Values()
	if len(got) != len(want) {
		t.Fatalf("a revision row has %d cells, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("revision cell %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestATitleThatAlreadyNamesItsTypeIsNotDoubled(t *testing.T) {
	f, _ := read(t, note("title: Supplier Register Policy\ndoc_type: Policy", ""))
	if f.CoverTitle() != "Supplier Register Policy" {
		t.Errorf("cover title = %q, want Supplier Register Policy", f.CoverTitle())
	}
}

func TestAClassificationOutsideTheSetIsRefusedNamingTheValue(t *testing.T) {
	_, _, err := Read(note("title: A Policy\nclassification: Secret", ""))
	if err == nil {
		t.Fatal("a classification of Secret was accepted")
	}
	if !strings.Contains(err.Error(), "Secret") {
		t.Errorf("the refusal %q does not name the value", err)
	}
}

func TestHeadingNumberingOutsideTheSetIsRefusedNamingTheValue(t *testing.T) {
	_, _, err := Read(note("title: A Policy\nheading_numbering: sometimes", ""))
	if err == nil {
		t.Fatal("a heading_numbering of sometimes was accepted")
	}
	if !strings.Contains(err.Error(), "sometimes") {
		t.Errorf("the refusal %q does not name the value", err)
	}
}

func TestHeadingNumberingReadsATrueOrFalse(t *testing.T) {
	f, _ := read(t, note("title: A Policy\nheading_numbering: false", ""))
	if f.HeadingNumbering {
		t.Error("heading numbering is on, want off when the note says false")
	}
	f, _ = read(t, note("title: A Policy\nheading_numbering: true", ""))
	if !f.HeadingNumbering {
		t.Error("heading numbering is off, want on when the note says true")
	}
}

func TestARevisionMissingItsVersionIsRefusedNamingTheField(t *testing.T) {
	_, _, err := Read(note("title: A Policy\nrevisions:\n  - date: 2026-08-01\n    change: First draft", ""))
	if err == nil {
		t.Fatal("a revision with no version was accepted")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("the refusal %q does not name the missing field", err)
	}
}

func TestARevisionWithAnUnknownFieldIsRefusedNamingIt(t *testing.T) {
	_, _, err := Read(note("title: A Policy\nrevisions:\n  - version: 1.0\n    reviewer: Somebody", ""))
	if err == nil {
		t.Fatal("a revision with an unknown field was accepted")
	}
	if !strings.Contains(err.Error(), "reviewer") {
		t.Errorf("the refusal %q does not name the unknown field", err)
	}
}

func TestRevisionsThatAreNotAListAreRefused(t *testing.T) {
	_, _, err := Read(note("title: A Policy\nrevisions: none yet", ""))
	if err == nil {
		t.Fatal("a revisions key that is not a list was accepted")
	}
	if !strings.Contains(err.Error(), "revisions") {
		t.Errorf("the refusal %q does not name the key", err)
	}
}

func TestANoteWithNoTitleCarriesTheCandidateFromTheFirstHeading(t *testing.T) {
	_, body, err := Read(note("owner: Head of Risk", "# Supplier Register Policy\n\nText.\n"))
	var missing *MissingTitle
	if !errors.As(err, &missing) {
		t.Fatalf("Read = %v, want a MissingTitle", err)
	}
	if missing.Candidate != "Supplier Register Policy" {
		t.Errorf("candidate = %q, want Supplier Register Policy", missing.Candidate)
	}
	if missing.Source != "h1" {
		t.Errorf("source = %q, want h1", missing.Source)
	}
	if !strings.Contains(string(body), "# Supplier Register Policy") {
		t.Error("a note with no title gives back no body, so the caller cannot look for a candidate")
	}
}

func TestTheCandidateFallsBackToTheFileNameWithTheDatePrefixStripped(t *testing.T) {
	candidate, source := TitleCandidate("Text with no heading.\n", "notes/2026-09-08-supplier-register-policy.md")
	if candidate != "Supplier register policy" {
		t.Errorf("candidate = %q, want Supplier register policy", candidate)
	}
	if source != "filename" {
		t.Errorf("source = %q, want filename", source)
	}
}

func TestAHeadingInsideAFenceIsNotACandidate(t *testing.T) {
	candidate, source := TitleCandidate("```\n# Not A Title\n```\n\n# The Real One\n", "x.md")
	if candidate != "The Real One" {
		t.Errorf("candidate = %q, want The Real One", candidate)
	}
	if source != "h1" {
		t.Errorf("source = %q, want h1", source)
	}
}

func TestTheGdocKeyIsIgnoredWhateverItsShape(t *testing.T) {
	v1 := note("title: A Policy\ngdoc: 1AbCdEf_GhIjKlMnOpQrStUvWxYz0123456789", "Body\n")
	if f, _ := read(t, v1); f.Title != "A Policy" {
		t.Errorf("title = %q with v1's gdoc: string, want A Policy", f.Title)
	}
	v2 := note(strings.Join([]string{
		"title: A Policy",
		"gdoc:",
		"  schema: 1",
		"  document_id: 1AbCdEf_GhIjKlMnOpQrStUvWxYz0123456789",
		"  proposals:",
		"    - id: suggest.abc",
		"      comment_id: AAABBB",
	}, "\n"), "Body\n")
	if f, _ := read(t, v2); f.Title != "A Policy" {
		t.Errorf("title = %q with v2's gdoc: block, want A Policy", f.Title)
	}
}

func TestAnUnknownAuthorKeyIsCarriedRatherThanRefused(t *testing.T) {
	// The note is the author's file and gdoc owns one key in it. A key this
	// package does not read belongs to somebody else's tool.
	if f, _ := read(t, note("title: A Policy\ntags: [risk, policy]", "")); f.Title != "A Policy" {
		t.Errorf("title = %q, want A Policy", f.Title)
	}
}

func TestANoteWithNoFrontMatterIsNoTitleAndTheWholeFileIsBody(t *testing.T) {
	src := []byte("Just some text, no front matter at all.\n")
	_, body, err := Read(src)
	var missing *MissingTitle
	if !errors.As(err, &missing) {
		t.Fatalf("Read = %v, want a MissingTitle", err)
	}
	if string(body) != string(src) {
		t.Errorf("body = %q, want the whole file", body)
	}
}

func TestFrontMatterThatOpensAndNeverClosesIsNotFrontMatter(t *testing.T) {
	src := []byte("---\ntitle: A Policy\n\nBody with no closing delimiter.\n")
	_, body, err := Read(src)
	var missing *MissingTitle
	if !errors.As(err, &missing) {
		t.Fatalf("Read = %v, want a MissingTitle", err)
	}
	if string(body) != string(src) {
		t.Errorf("body = %q, want the whole file", body)
	}
}

func TestTheDateRendersInUKLongForm(t *testing.T) {
	f, _ := read(t, note("title: A Policy\ndate: 2026-09-08", ""))
	if f.Date != "8 September 2026" {
		t.Errorf("date = %q, want 8 September 2026", f.Date)
	}
}

func TestADateTheAuthorWroteInWordsIsLeftAlone(t *testing.T) {
	f, _ := read(t, note("title: A Policy\ndate: September 2026", ""))
	if f.Date != "September 2026" {
		t.Errorf("date = %q, want September 2026", f.Date)
	}
}

func TestFrontMatterThatIsNotAMappingIsRefused(t *testing.T) {
	_, _, err := Read([]byte("---\n- one\n- two\n---\nBody\n"))
	if err == nil {
		t.Fatal("a front-matter list was accepted")
	}
}

func TestFrontMatterThatIsNotYAMLIsRefused(t *testing.T) {
	_, _, err := Read([]byte("---\ntitle: [unclosed\n---\nBody\n"))
	if err == nil {
		t.Fatal("front matter that is not YAML was accepted")
	}
}

func TestAByteOrderMarkAndTrailingSpacesStillOpenFrontMatter(t *testing.T) {
	src := []byte("\uFEFF--- \ntitle: A Policy\n...\t\nBody\n")
	f, body := read(t, src)
	if f.Title != "A Policy" {
		t.Errorf("title = %q, want A Policy", f.Title)
	}
	if string(body) != "Body\n" {
		t.Errorf("body = %q, want Body", body)
	}
}
