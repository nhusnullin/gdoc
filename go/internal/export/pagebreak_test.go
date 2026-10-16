package export

import (
	"strings"
	"testing"
)

// M19's half of the export: a page break standing on its own comes home as
// `---`, which is the line the next build reads as that same break. The
// fixture is internal/docs's page-breaks.json, in the shape the live
// measurement of 2026-10-16 found.

// exported is one fixture's single tab as the export writes it, with the lines
// split, because every rule here is about a line standing on its own.
func exported(t *testing.T, name string) (string, []string, []string) {
	t.Helper()
	files, _, _, warnings := Project(fixture(t, name), nil, nil)
	if len(files) != 1 {
		t.Fatalf("%s gave %d files, want 1", name, len(files))
	}
	return files[0].Body, strings.Split(files[0].Body, "\n"), warnings
}

// TestAPageBreakOfItsOwnExportsAsThreeHyphens is the rule. A break on a line
// of its own is a line of markdown, so it is written as the one a note holds
// for it, between blank lines, and nothing is lost to warn about.
func TestAPageBreakOfItsOwnExportsAsThreeHyphens(t *testing.T) {
	body, lines, warnings := exported(t, "page-breaks.json")
	want := "This paragraph is the last thing on its page.\n\n---\n\nA reader turns a page to reach this."
	if !strings.Contains(body, want) {
		t.Errorf("the file does not hold\n%q\nit holds:\n%s", want, body)
	}
	rules := 0
	for _, l := range lines {
		if l == "---" {
			rules++
		}
	}
	if rules != 2 {
		t.Errorf("the file holds %d `---` lines, want the author's break and the element that stands alone:\n%s", rules, body)
	}
	for _, w := range warnings {
		if strings.Contains(w, "index 68") || strings.Contains(w, "index 135") {
			t.Errorf("a break the file carried as a line was warned about anyway: %s", w)
		}
	}
}

// TestTheFirstHeadingsBreakIsLeftOut is the one drop. The house breaks the
// page before the first body heading and rides that break inside the heading
// paragraph, so the strip leaves it at the top of the file. build puts it back
// from first_heading_page_break, so a `---` written here would be a second
// break the note never asked for.
func TestTheFirstHeadingsBreakIsLeftOut(t *testing.T) {
	body, _, _ := exported(t, "page-breaks.json")
	if strings.HasPrefix(body, "---") {
		t.Errorf("the file opens with the first heading's own break:\n%s", body)
	}
	if !strings.HasPrefix(body, "# Where the page turns\n") {
		t.Errorf("the file does not open with the first heading:\n%s", body)
	}
}

// TestAPageBreakInsideASentenceKeepsItsMarker is the shape no line of markdown
// can carry. A `---` written mid-sentence would end the sentence, so the
// marker and its warning stay and a session resolves them by hand.
func TestAPageBreakInsideASentenceKeepsItsMarker(t *testing.T) {
	body, _, warnings := exported(t, "page-breaks.json")
	if want := "Before the break.[page break] After it."; !strings.Contains(body, want) {
		t.Errorf("the file does not hold %q:\n%s", want, body)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "a page break at index 123 is printed as [page break]") {
			found = true
		}
	}
	if !found {
		t.Errorf("no warning names the break inside the sentence: %v", warnings)
	}
}

// TestAHorizontalRuleStillExportsAsItsMarker holds the line that is not a
// break. A document's own rule has no spelling in a note since `---` became
// the page break, so it comes home as [rule] with its warning, for somebody to
// resolve.
func TestAHorizontalRuleStillExportsAsItsMarker(t *testing.T) {
	body, _, warnings := exported(t, "page-breaks.json")
	if !strings.Contains(body, "[rule]") {
		t.Errorf("the file holds no [rule]:\n%s", body)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "a horizontal rule at index 137 is printed as [rule]") {
			found = true
		}
	}
	if !found {
		t.Errorf("no warning names the horizontal rule: %v", warnings)
	}
}
