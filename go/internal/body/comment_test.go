package body

// A pencil is a comment: what the walk makes of one inside a sentence, and
// what it still makes of raw HTML that is not one.
//
// Every spelling here is written out. The parser reads the opener off this
// package's own constant and the house style states the same string, and
// TestThePencilSyntaxIsTheHouseStyles is where the two are pinned together.

import (
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// TestAnHTMLCommentIsAPencilRun is the shape an author writes from this
// milestone on: an aside the hub's reader never pays for, grey in the
// document. The words around it are the sentence, unchanged.
func TestAnHTMLCommentIsAPencilRun(t *testing.T) {
	runs := firstBlockRuns(t, "a <!-- quiet --> b\n")

	if len(runs) != 3 {
		t.Fatalf("a sentence holding one comment is %d runs: %#v", len(runs), runs)
	}
	got := runs[1]
	if got.Mark != "pencil" {
		t.Errorf("the comment is mark %q, want pencil", got.Mark)
	}
	if got.Text != "quiet" {
		t.Errorf("the comment marks %q, want \"quiet\": one space inside each delimiter is the delimiter's", got.Text)
	}
	if got.Italic || got.Bold || got.Strike || got.Mono {
		t.Errorf("the comment's run is %#v, want the grey and nothing else", got)
	}
	if runs[0].Text != "a " || runs[2].Text != " b" {
		t.Errorf("the words around the comment read %q and %q, want them unchanged",
			runs[0].Text, runs[2].Text)
	}
	if runs[0].Mark != "" || runs[2].Mark != "" {
		t.Errorf("the words around the comment carry a mark: %#v", runs)
	}
	if text := joined(runs); strings.Contains(text, "<!--") || strings.Contains(text, "-->") {
		t.Errorf("the prose reads %q, want the delimiters out of the words", text)
	}
}

// TestTheWordsInsideAPencilAreLiteral. Nothing inside a comment is markdown:
// goldmark hands the whole comment over as one piece of raw HTML, and the
// words reach the document as the author typed them.
func TestTheWordsInsideAPencilAreLiteral(t *testing.T) {
	runs := firstBlockRuns(t, "a <!-- **x** and ==y== --> b\n")

	got := runCarrying(t, runs, "x")
	if got.Mark != "pencil" {
		t.Errorf("the comment is mark %q, want pencil", got.Mark)
	}
	if got.Text != "**x** and ==y==" {
		t.Errorf("the comment marks %q, want the asterisks and the equals signs as typed", got.Text)
	}
	if got.Bold {
		t.Error("the asterisks inside the comment made a bold run, and nothing inside a comment is markdown")
	}
	for _, run := range runs {
		if run.Mark == "yellow" {
			t.Errorf("the equals signs inside the comment made a highlight: %#v", run)
		}
	}
}

// TestAnEmptyCommentIsLeftOut. A comment holding no words marks none, so there
// is nothing to colour and nothing was lost: it goes with no warning, the way
// a blank line does.
func TestAnEmptyCommentIsLeftOut(t *testing.T) {
	for _, empty := range []string{"<!-- -->", "<!---->"} {
		source := "a " + empty + " b\n"

		runs := firstBlockRuns(t, source)
		if len(runs) != 1 || runs[0].Text != "a  b" {
			t.Errorf("%s builds %#v, want the sentence and no run of its own", empty, runs)
		}
		for _, run := range runs {
			if run.Mark != "" {
				t.Errorf("%s left a run carrying mark %q", empty, run.Mark)
			}
		}
		if out := walk(t, source); len(out.Warnings) != 0 {
			t.Errorf("%s warned %q, and a comment holding no words loses nothing", empty, out.Warnings)
		}

		// The same on a line of its own: no paragraph of its own, no warning.
		alone := walk(t, empty+"\n\nA paragraph.\n")
		if len(alone.Blocks) != 1 {
			t.Errorf("%s on a line of its own built %d blocks, want the paragraph alone:\n%s",
				empty, len(alone.Blocks), serialise(t, alone.Blocks))
		}
		if len(alone.Warnings) != 0 {
			t.Errorf("%s on a line of its own warned %q, and a comment holding no words loses nothing",
				empty, alone.Warnings)
		}
	}
}

// TestOtherRawHTMLIsStillWarnedAndLeftOut. A comment is the one shape of HTML
// this package reads. Everything else keeps the rule it has had since the
// first build: named by line, left out, never dropped in silence.
func TestOtherRawHTMLIsStillWarnedAndLeftOut(t *testing.T) {
	inline := walk(t, "a <b>x</b> b\n")
	if !warned(inline.Warnings, "inline HTML") {
		t.Errorf("a <b> in a sentence warned %q, want the inline HTML named", inline.Warnings)
	}
	if got := serialise(t, inline.Blocks); strings.Contains(got, "<b>") {
		t.Errorf("the tag reached the document:\n%s", got)
	}

	block := walk(t, "<div>\nx\n</div>\n\nA paragraph.\n")
	if !warned(block.Warnings, "a block of HTML") {
		t.Errorf("a block of HTML warned %q, want it named", block.Warnings)
	}
	if got := serialise(t, block.Blocks); strings.Contains(got, "<div>") {
		t.Errorf("the block reached the document:\n%s", got)
	}
}

// warned says whether any warning holds these words.
func warned(warnings []string, words string) bool {
	for _, w := range warnings {
		if strings.Contains(w, words) {
			return true
		}
	}
	return false
}

// TestACommentOnLinesOfItsOwnIsGreyParagraphs. An aside written on lines of
// its own stands alone in the document, grey. The lines of one stretch are
// joined by a single space, the way a paragraph's own wrapped lines are, and a
// blank line inside the comment opens a second aside: a comment long enough to
// hold one reads as two.
func TestACommentOnLinesOfItsOwnIsGreyParagraphs(t *testing.T) {
	out := walk(t, "<!-- one\ntwo\nthree\n\nfour\nfive -->\n")

	if len(out.Blocks) != 2 {
		t.Fatalf("the comment built %d blocks, want two paragraphs:\n%s",
			len(out.Blocks), serialise(t, out.Blocks))
	}
	for i, want := range []string{"one two three", "four five"} {
		runs := findAll(out.Blocks[i], "w:r")
		if len(runs) != 1 {
			t.Errorf("paragraph %d holds %d runs, want the one grey run:\n%s",
				i+1, len(runs), serialise(t, out.Blocks))
			continue
		}
		if got := runWords(runs[0]); got != want {
			t.Errorf("paragraph %d reads %q, want %q", i+1, got, want)
		}
		rPr := runs[0].SelectElement("w:rPr")
		if rPr == nil {
			t.Errorf("paragraph %d carries no w:rPr, want the pencil", i+1)
			continue
		}
		if got := value(rPr, "w:rStyle"); got != "Pencil" {
			t.Errorf("paragraph %d names style %q, want Pencil", i+1, got)
		}
	}
	if got := out.Counts.Paragraphs; got != 2 {
		t.Errorf("the comment counted %d paragraphs, want 2", got)
	}
	if len(out.Warnings) != 0 {
		t.Errorf("the comment warned %q, and nothing was left out", out.Warnings)
	}
	if got := serialise(t, out.Blocks); strings.Contains(got, "<!--") || strings.Contains(got, "-->") {
		t.Errorf("the delimiters reached the document:\n%s", got)
	}
}

// TestACommentOnOneLineOfItsOwnIsOneGreyParagraph is the shape a note is most
// likely to hold: one aside between two paragraphs of prose.
func TestACommentOnOneLineOfItsOwnIsOneGreyParagraph(t *testing.T) {
	out := walk(t, "Before.\n\n<!-- an aside -->\n\nAfter.\n")

	if len(out.Blocks) != 3 {
		t.Fatalf("the note built %d blocks, want three paragraphs:\n%s",
			len(out.Blocks), serialise(t, out.Blocks))
	}
	run := runElement(t, out.Blocks, "an aside")
	rPr := run.SelectElement("w:rPr")
	if rPr == nil {
		t.Fatal("the aside carries no w:rPr, want the pencil")
	}
	if got := value(rPr, "w:rStyle"); got != "Pencil" {
		t.Errorf("the aside names style %q, want Pencil", got)
	}
	if len(out.Warnings) != 0 {
		t.Errorf("the aside warned %q, and nothing was left out", out.Warnings)
	}
}

// runWords is the words one run carries.
func runWords(run *etree.Element) string {
	words := ""
	for _, e := range run.FindElements("w:t") {
		words += e.Text()
	}
	return words
}
