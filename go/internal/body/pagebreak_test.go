// This file is the author's own page break: that a thematic break renders as a
// paragraph which opens a page, that every spelling of the break is that one
// break, and that no line is drawn any more.

package body

import (
	"strings"
	"testing"
)

// A thematic break between two paragraphs is a page break: an empty paragraph
// carrying pageBreakBefore, the shape the front matter's own breaks use. The
// horizontal rule is gone, so no border reaches the body.
func TestAThematicBreakIsAPageBreakParagraph(t *testing.T) {
	out := walk(t, "Before the break.\n\n---\n\nAfter the break.\n")

	if n := len(out.Blocks); n != 3 {
		t.Fatalf("%d blocks rendered, want paragraph, break, paragraph", n)
	}
	break_ := out.Blocks[1]
	pPr := break_.SelectElement("w:pPr")
	if pPr == nil {
		t.Fatal("the break paragraph carries no w:pPr")
	}
	el := pPr.SelectElement("w:pageBreakBefore")
	if el == nil {
		t.Fatal("the break paragraph carries no w:pageBreakBefore")
	}
	if got := attr(t, el, "w:val"); got != "1" {
		t.Errorf("w:pageBreakBefore w:val = %q, want 1", got)
	}
	if runs := findAll(break_, "w:r"); len(runs) != 0 {
		t.Errorf("the break paragraph holds %d runs, want none", len(runs))
	}
	if text := strings.TrimSpace(break_.Text()); text != "" {
		t.Errorf("the break paragraph holds the text %q", text)
	}
	body := serialise(t, out.Blocks)
	if strings.Contains(body, "w:pBdr") {
		t.Error("a border reached the body: the horizontal rule is gone")
	}
	if n := len(out.Warnings); n != 0 {
		t.Errorf("the break warned: %v", out.Warnings)
	}
}

// The break paragraph holds nothing, so it states zero space above and below
// rather than inheriting a paragraph's 12pt and pushing the page it opens down.
func TestAPageBreakParagraphPushesTheNewPageNowhere(t *testing.T) {
	out := walk(t, "Before the break.\n\n---\n\nAfter the break.\n")
	if n := len(out.Blocks); n != 3 {
		t.Fatalf("%d blocks rendered, want paragraph, break, paragraph", n)
	}
	spacing := find(out.Blocks[1], "w:spacing")
	if got := attr(t, spacing, "w:before"); got != "0" {
		t.Errorf("the break has %s twips above it, want 0", got)
	}
	if got := attr(t, spacing, "w:after"); got != "0" {
		t.Errorf("the break has %s twips below it, want 0", got)
	}
}

// Every spelling goldmark reads as a thematic break is the same break. The
// node carries no spelling, and reading the source again would be a second
// parse to keep in step.
func TestEveryThematicBreakSpellingIsTheSameBreak(t *testing.T) {
	want := serialise(t, walk(t, "Before.\n\n---\n\nAfter.\n").Blocks)
	for _, spelling := range []string{"***", "___", "- - -", "-----"} {
		got := serialise(t, walk(t, "Before.\n\n"+spelling+"\n\nAfter.\n").Blocks)
		if got != want {
			t.Errorf("%q rendered differently from ---:\n%s", spelling, got)
		}
	}
}
