package frontmatter

import "testing"

func parse(t *testing.T, text string) *Meta {
	t.Helper()
	meta, _, err := Parse(text, "", "note.md")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return meta
}

func TestTitleIsTheOnlyRequiredField(t *testing.T) {
	meta := parse(t, "---\ntitle: Some Title\n---\nbody\n")

	if meta.Title != "Some Title" {
		t.Errorf("title = %q, want %q", meta.Title, "Some Title")
	}
	if meta.CoverTitle != "Some Title" {
		t.Errorf("cover title = %q, want it to stand alone with no doc_type", meta.CoverTitle)
	}
	if meta.Classification != "Internal (I)" {
		t.Errorf("classification = %q, want the Internal default", meta.Classification)
	}
	if meta.ReviewFrequency != "Annually" {
		t.Errorf("review frequency = %q, want the Annually default", meta.ReviewFrequency)
	}
}

func TestVersionKeepsItsTrailingZero(t *testing.T) {
	// Written as YAML, 1.0 is a float. Decoding it as one and printing it back
	// gives "1", and the cover then disagrees with the version control table.
	meta := parse(t, "---\ntitle: T\nversion: 1.0\n---\nbody\n")

	if meta.Version != "1.0" {
		t.Errorf("version = %q, want %q", meta.Version, "1.0")
	}
}

func TestDateBecomesUKLongForm(t *testing.T) {
	meta := parse(t, "---\ntitle: T\ndate: 2026-07-15\n---\nbody\n")

	if meta.Date != "15 July 2026" {
		t.Errorf("date = %q, want %q", meta.Date, "15 July 2026")
	}
}

func TestCoverTitleAppendsDocTypeOnlyOnce(t *testing.T) {
	cases := []struct{ title, docType, want string }{
		{"Third Party Risk", "Policy", "Third Party Risk Policy"},
		{"Third Party Risk Policy", "Policy", "Third Party Risk Policy"},
		{"Third Party Risk", "", "Third Party Risk"},
	}
	for _, c := range cases {
		if got := coverTitle(c.title, c.docType); got != c.want {
			t.Errorf("coverTitle(%q, %q) = %q, want %q", c.title, c.docType, got, c.want)
		}
	}
}

func TestRunningHeadIsPrefixed(t *testing.T) {
	meta := parse(t, "---\ntitle: Access Control\ndoc_type: Policy\n---\nbody\n")

	if meta.RunningHead != "Altery - Access Control Policy" {
		t.Errorf("running head = %q", meta.RunningHead)
	}
}

func TestMissingTitleCarriesACandidateRatherThanUsingIt(t *testing.T) {
	_, _, err := Parse("---\nversion: 1.0\n---\n\n# The First Heading\n\nbody\n", "", "note.md")

	missing, ok := err.(*MissingTitle)
	if !ok {
		t.Fatalf("error = %#v, want a MissingTitle", err)
	}
	if missing.Candidate != "The First Heading" || missing.Source != "h1" {
		t.Errorf("candidate = %q from %q, want the H1", missing.Candidate, missing.Source)
	}
}

func TestTitleCandidateIgnoresAHeadingInsideAFence(t *testing.T) {
	body := "```\n# Not A Heading\n```\n\n# The Real Heading\n"

	candidate, source := TitleCandidate(body, "2026-08-29-some-note.md")

	if candidate != "The Real Heading" || source != "h1" {
		t.Errorf("candidate = %q from %q", candidate, source)
	}
}

func TestTitleCandidateFallsBackToTheFilename(t *testing.T) {
	candidate, source := TitleCandidate("no headings here\n", "2026-08-29-third-party-risk.md")

	if candidate != "Third party risk" || source != "filename" {
		t.Errorf("candidate = %q from %q", candidate, source)
	}
}

func TestUnknownClassificationIsRefused(t *testing.T) {
	_, _, err := Parse("---\ntitle: T\nclassification: secret\n---\nbody\n", "", "note.md")

	if err == nil {
		t.Fatal("an unrecognised classification must be refused, not defaulted")
	}
}

func TestClassificationAcceptsTheCoverForm(t *testing.T) {
	meta := parse(t, "---\ntitle: T\nclassification: Restricted (R)\n---\nbody\n")

	if meta.Classification != "Restricted (R)" {
		t.Errorf("classification = %q", meta.Classification)
	}
}

func TestRevisionsAreValidatedFieldByField(t *testing.T) {
	_, _, err := Parse("---\ntitle: T\nrevisions:\n  - version: 1.0\n    autor: typo\n---\nbody\n",
		"", "note.md")

	if err == nil {
		t.Fatal("a misspelled revision field must be refused, not dropped")
	}
}

func TestRevisionsReachTheTemplateInColumnOrder(t *testing.T) {
	meta := parse(t, `---
title: T
revisions:
  - version: 2.0
    date: 2026-08-01
    author: N K
    approved_by: Board
    approval_date: 2026-08-02
    section: All
    change: Reissued
---
body
`)
	if len(meta.Revisions) != 1 {
		t.Fatalf("revisions = %d, want 1", len(meta.Revisions))
	}
	want := []string{"2.0", "1 August 2026", "N K", "Board", "2 August 2026", "All", "Reissued"}
	got := meta.Revisions[0].Values()
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("column %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMissingFrontMatterNamesWhatIsNeeded(t *testing.T) {
	_, _, err := Parse("# Just A Heading\n", "", "note.md")

	if err == nil {
		t.Fatal("a file with no front matter must be refused")
	}
}
