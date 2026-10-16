package view

import (
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// M19's half of the read: a page break prints where the page turns, in both of
// the shapes Docs states one in. The fixture is internal/docs's
// page-breaks.json, written in the shape the live measurement of 2026-10-16
// found.

// TestABreakBeforeAParagraphPrintsAsAPageBreakLine is the flag's line. Drive's
// docx import states a body break as pageBreakBefore on an empty paragraph, so
// the paragraph itself prints nothing and the break is the only thing to print:
// a line of its own between the two paragraphs the author wrote.
func TestABreakBeforeAParagraphPrintsAsAPageBreakLine(t *testing.T) {
	text, warnings := Text(fixture(t, "page-breaks.json"))
	want := "This paragraph is the last thing on its page.\n\n[page break]\n\nA reader turns a page to reach this."
	if !strings.Contains(text, want) {
		t.Errorf("the projection does not hold\n%q\nit holds:\n%s", want, text)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "a page break at index 68 is printed as [page break]") {
			found = true
		}
	}
	if !found {
		t.Errorf("no warning names the break before the paragraph at index 68: %v", warnings)
	}
}

// TestTheHousesFirstHeadingBreakPrintsBeforeTheHeading is the second shape the
// measurement found: the house's own break on the first body heading rides
// inside the heading paragraph rather than standing before it. The line is
// still printed before the heading, because that is where the page turns.
func TestTheHousesFirstHeadingBreakPrintsBeforeTheHeading(t *testing.T) {
	text, _ := Text(fixture(t, "page-breaks.json"))
	want := "[page break]\n\n# Where the page turns"
	if !strings.Contains(text, want) {
		t.Errorf("the projection does not hold\n%q\nit holds:\n%s", want, text)
	}
}

// TestANativePageBreakElementStillPrintsWhereItStands holds the shape that was
// already read: a pageBreak element is a run, and a run prints where it stands.
// One of the fixture's two is in the middle of a sentence and one is a
// paragraph's only content, and the read prints both the same way, because
// `read` prints what the document holds and decides nothing about it.
func TestANativePageBreakElementStillPrintsWhereItStands(t *testing.T) {
	text, warnings := Text(fixture(t, "page-breaks.json"))
	if want := "Before the break.[page break] After it."; !strings.Contains(text, want) {
		t.Errorf("the projection does not hold %q:\n%s", want, text)
	}
	if n := strings.Count(text, "[page break]"); n != 4 {
		t.Errorf("the projection holds %d page break marks, want the 2 flags and the 2 elements:\n%s", n, text)
	}
	if want := "[rule]"; !strings.Contains(text, want) {
		t.Errorf("the projection does not hold %q, and a horizontal rule is still a rule:\n%s", want, text)
	}
	if len(warnings) < 5 {
		t.Errorf("the projection raised %d warnings, want one per break and one for the rule: %v", len(warnings), warnings)
	}
}

// TestACallerThatNamesALineForABreakGetsItAndNoWarning is the door export
// comes through. A page break standing on its own is a line of markdown a
// caller can write, so the caller says what that line is and nothing is lost
// to warn about. A break inside a sentence is not that shape and keeps its
// placeholder and its warning, because no line of markdown says a page turns
// mid-sentence.
func TestACallerThatNamesALineForABreakGetsItAndNoWarning(t *testing.T) {
	text, warnings := Project(fixture(t, "page-breaks.json"), Options{PageBreak: "---"})
	if n := strings.Count(text, "[page break]"); n != 1 {
		t.Errorf("the projection holds %d page break marks, want only the one inside a sentence:\n%s", n, text)
	}
	lines := 0
	for _, l := range strings.Split(text, "\n") {
		if l == "---" {
			lines++
		}
	}
	if lines != 3 {
		t.Errorf("the projection holds %d `---` lines, want the 2 flags and the standalone element:\n%s", lines, text)
	}
	breaks := 0
	for _, w := range warnings {
		if strings.Contains(w, "a page break") {
			breaks++
		}
	}
	if breaks != 1 {
		t.Errorf("the projection warned about %d page breaks, want only the one it could not write a line for: %v", breaks, warnings)
	}
}

// breakRun is a native page break element, the shape a docx import leaves in a
// paragraph. The newline a paragraph ends with is the text run beside it.
func breakRun(at int) []docs.Run {
	return []docs.Run{
		{Kind: docs.KindPageBreak, StartIndex: at, EndIndex: at + 1},
		{Kind: docs.KindText, Text: "\n", StartIndex: at + 1, EndIndex: at + 2},
	}
}

// TestOnlyABodyParagraphsBreakIsWrittenAsTheCallersLine: the caller's line is a
// line of markdown, so it is written only where the paragraph is a line of the
// file on its own. A heading takes a "#" prefix and an item of a numbered list
// takes its number, and "# ---" is a heading reading "---" to the next build
// while "1. ---" is an item holding a break. Each keeps the placeholder and the
// warning that says the break did not survive, which is what the read printed
// for every break before the line existed.
func TestOnlyABodyParagraphsBreakIsWrittenAsTheCallersLine(t *testing.T) {
	for _, c := range []struct {
		what  string
		block docs.Block
		want  string
	}{
		{
			what:  "a heading",
			block: docs.Block{Paragraph: &docs.Paragraph{Style: "HEADING_1", Runs: breakRun(1)}},
			want:  "# [page break]",
		},
		{
			what: "a numbered item",
			block: docs.Block{Paragraph: &docs.Paragraph{
				Style:  "NORMAL_TEXT",
				Bullet: &docs.Bullet{Ordered: true, ListID: "l1"},
				Runs:   breakRun(1),
			}},
			want: "1. [page break]",
		},
		{
			// The cell of a two-column table, because a one-column table's
			// own separator row is the three hyphens this test looks for.
			what: "a table cell",
			block: docs.Block{Table: &docs.Table{StartIndex: 1, Rows: [][]docs.Cell{{
				{Blocks: []docs.Block{{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", Runs: breakRun(2)}}}},
				{Blocks: []docs.Block{{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", Runs: []docs.Run{run("a cell", 5)}}}}},
			}}}},
			want: "[page break] | a cell",
		},
	} {
		t.Run(c.what, func(t *testing.T) {
			d := &docs.Document{Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{c.block}}}}
			text, warnings := Project(d, Options{PageBreak: "---"})
			for _, line := range strings.Split(text, "\n") {
				if line == "---" {
					t.Errorf("the caller's line was written where it does not stand alone:\n%s", text)
				}
			}
			if !strings.Contains(text, c.want) {
				t.Errorf("the projection does not hold %q:\n%s", c.want, text)
			}
			found := false
			for _, w := range warnings {
				if strings.Contains(w, "a page break at index") {
					found = true
				}
			}
			if !found {
				t.Errorf("nothing warned that the break was printed as a placeholder: %v", warnings)
			}
		})
	}
}
