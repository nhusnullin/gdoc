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

// TestAPencilGoesWhereAMarkGoes. The milestone's one decision is that the grey
// mark keeps everything it does and changes its spelling, so a comment is
// refused where a highlight is refused and carried where a highlight is
// carried. Both spellings are the one node, so one case of each is the pin
// that the rule is not written twice.
func TestAPencilGoesWhereAMarkGoes(t *testing.T) {
	for _, c := range []struct{ note, names string }{
		{"Words.\n\n## A <!-- a note --> heading\n\nMore words.\n", "heading"},
		{"Words.\n\n## A %% a note %% heading\n\nMore words.\n", "heading"},
		{"Words.\n\n![a <!-- a note --> alt](badge.png)\n", "alt text"},
		{"Words.\n\nA ==gap with <!-- a note --> inside it== here.\n", "mark inside"},
		{"Words.\n\nA ==gap with %% a note %% inside it== here.\n", "mark inside"},
	} {
		got := refused(t, c.note)
		if !strings.Contains(got, "line 3:") {
			t.Errorf("%q: the refusal is %q, want it to name line 3", c.note, got)
		}
		if !strings.Contains(got, c.names) {
			t.Errorf("%q: the refusal is %q, and it does not say where the mark sits", c.note, got)
		}
	}

	for _, c := range []struct{ where, note string }{
		{"a table cell", "| a | b |\n| --- | --- |\n| x <!-- a note --> y | z |\n"},
		{"a list item", "- item <!-- a note --> here\n"},
		{"a list item in Obsidian's spelling", "- item %% a note %% here\n"},
		{"a block quote", "> quoted <!-- a note --> words\n"},
	} {
		out := walk(t, c.note)
		pencil(t, out.Blocks, "a note", c.where)
		if len(out.Warnings) != 0 {
			t.Errorf("the pencil in %s warned %q, and nothing was left out", c.where, out.Warnings)
		}
	}

	// A footnote is the one place a mark goes out and does not come home, and
	// the pencil goes there the way the others do: footnoteRuns walks a
	// definition's inlines through the same path a paragraph's take, so the
	// aside is set under the page grey. The read keeps a footnote's text
	// alone, so internal/export writes the words plain and names the loss
	// there rather than here: its own
	// TestAPencilMarkInsideAFootnoteIsNamedAsALoss is that pin, and
	// docs/backlog/marks-inside-footnotes.md is the item.
	out := walk(t, "Words.[^a]\n\n[^a]: a note with <!-- an aside --> in it.\n")
	if len(out.Footnotes) != 1 {
		t.Fatalf("the file built %d notes, want the one the mark makes", len(out.Footnotes))
	}
	pencil(t, out.Footnotes[0].Blocks, "an aside", "a footnote")
	if len(out.Warnings) != 0 {
		t.Errorf("the pencil in a footnote warned %q, and nothing was left out", out.Warnings)
	}

	// A comment on lines of its own inside a footnote is the one shape that is
	// left out there, the way every block a footnote cannot hold is left out:
	// footnoteBlocks sets paragraphs, and the house style has no grey
	// paragraph of its own under the page. It is named as a comment rather
	// than as a block of HTML, because an author told they wrote a block of
	// HTML would look for a tag that is not in the note.
	block := walk(t, "Words.[^a]\n\n[^a]: a note\n\n    <!-- an aside -->\n")
	if !warned(block.Warnings, "a comment inside a footnote") {
		t.Errorf("a comment on lines of its own inside a footnote warned %q, want it named as a comment",
			block.Warnings)
	}
	if len(block.Footnotes) != 1 {
		t.Fatalf("the file built %d notes, want the one the mark makes", len(block.Footnotes))
	}
	if got := serialise(t, block.Footnotes[0].Blocks); strings.Contains(got, "an aside") {
		t.Errorf("the aside reached the note:\n%s", got)
	}
}

// pencil is the one run carrying these words, checked for the grey. The
// character style is written out, because it is what internal/render writes
// and a test reading the constant would follow it wherever somebody moved it.
func pencil(t *testing.T, blocks []*etree.Element, words, where string) {
	t.Helper()
	rPr := runElement(t, blocks, words).SelectElement("w:rPr")
	if rPr == nil {
		t.Errorf("the pencil in %s carries no w:rPr, want the grey", where)
		return
	}
	if got := value(rPr, "w:rStyle"); got != "Pencil" {
		t.Errorf("the pencil in %s names style %q, want Pencil", where, got)
	}
}

// TestAHighlightInsideAPencilIsWords is the other side of the nesting rule.
// Nothing inside a comment is markdown, so a highlight written in one is words
// rather than a mark inside a mark: the aside is one grey run carrying the
// equals signs the author typed.
func TestAHighlightInsideAPencilIsWords(t *testing.T) {
	for _, source := range []string{
		"Words.\n\na <!-- a ==b== c --> d\n",
		"Words.\n\n<!-- a ==b== c -->\n",
	} {
		out, err := Render(config(t), []byte(source), "testdata/docs", true)
		if err != nil {
			t.Errorf("%q was refused: %v", source, err)
			continue
		}
		pencil(t, out.Blocks, "a ==b== c", "a comment holding a highlight")
	}
}
