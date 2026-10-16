package body

// The Obsidian spelling of a comment: %% words %% inside a sentence, and a %%
// fence on lines of its own. Both reach the node the HTML spelling reaches, so
// every rule the comment tests state holds here without being restated.
//
// Every spelling is written out. The house style states the same string as one
// of the pencil mark's syntaxes, and TestThePencilSyntaxIsTheHouseStyles is
// where the two are pinned together.

import (
	"strings"
	"testing"
)

// TestAnObsidianCommentIsAPencilRun is the second spelling of Task 1's shape:
// an aside the hub's reader never pays for, grey in the document.
func TestAnObsidianCommentIsAPencilRun(t *testing.T) {
	runs := firstBlockRuns(t, "a %% quiet %% b\n")

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
	if text := joined(runs); strings.Contains(text, "%%") {
		t.Errorf("the prose reads %q, want the delimiters out of the words", text)
	}
}

// TestTheWordsInsideAnObsidianCommentAreLiteral. Nothing inside a comment is
// markdown, whichever spelling opened it: the parser claims the whole of it in
// one piece and the words reach the document as the author typed them.
func TestTheWordsInsideAnObsidianCommentAreLiteral(t *testing.T) {
	runs := firstBlockRuns(t, "a %% **x** and ==y== %% b\n")

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

// TestAnEmptyObsidianCommentIsLeftOut. A comment holding no words marks none,
// so there is nothing to colour and nothing was lost: it goes with no warning,
// the way the HTML spelling of the same emptiness does.
func TestAnEmptyObsidianCommentIsLeftOut(t *testing.T) {
	for _, empty := range []string{"%%%%", "%% %%"} {
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

// TestAnObsidianFenceIsGreyParagraphs is Task 2's shape in the second
// spelling: %% alone on a line, the aside, %% alone on a line. The lines of
// one stretch are joined by a single space and a blank line inside the fence
// opens a second aside, exactly as the HTML spelling does.
func TestAnObsidianFenceIsGreyParagraphs(t *testing.T) {
	out := walk(t, "%%\none\ntwo\nthree\n\nfour\nfive\n%%\n")

	if len(out.Blocks) != 2 {
		t.Fatalf("the fence built %d blocks, want two paragraphs:\n%s",
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
		t.Errorf("the fence counted %d paragraphs, want 2", got)
	}
	if len(out.Warnings) != 0 {
		t.Errorf("the fence warned %q, and nothing was left out", out.Warnings)
	}
	if got := serialise(t, out.Blocks); strings.Contains(got, "%%") {
		t.Errorf("the delimiters reached the document:\n%s", got)
	}
}

// TestAnObsidianFenceInterruptsAParagraph. A comment is one of the blocks
// CommonMark lets open in the middle of a paragraph, and the HTML spelling
// does: an aside written straight under a line of prose is an aside and not
// the prose's own next line.
func TestAnObsidianFenceInterruptsAParagraph(t *testing.T) {
	out := walk(t, "Before.\n%%\nan aside\n%%\nAfter.\n")

	run := runElement(t, out.Blocks, "an aside")
	rPr := run.SelectElement("w:rPr")
	if rPr == nil {
		t.Fatal("the aside carries no w:rPr, want the pencil")
	}
	if got := value(rPr, "w:rStyle"); got != "Pencil" {
		t.Errorf("the aside names style %q, want Pencil", got)
	}
	if got := serialise(t, out.Blocks); strings.Contains(got, "%%") {
		t.Errorf("the delimiters reached the document:\n%s", got)
	}
}

// TestAnObsidianCommentOnOneLineOfItsOwnIsOneGreyParagraph is the shape a note
// is most likely to hold: one aside between two paragraphs of prose. Written
// on one line it is a paragraph holding one comment, which is the sentence
// shape, and the document reads the same either way.
func TestAnObsidianCommentOnOneLineOfItsOwnIsOneGreyParagraph(t *testing.T) {
	out := walk(t, "Before.\n\n%% an aside %%\n\nAfter.\n")

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

// TestAnObsidianCommentInCodeIsCode. Nothing inside backticks is markdown, and
// a fence this package's own parser claims must not be claimed inside a code
// block either: a note explaining the syntax would otherwise lose the rest of
// itself to a comment that was never meant to open.
func TestAnObsidianCommentInCodeIsCode(t *testing.T) {
	runs := firstBlockRuns(t, "a `%% quiet %%` b\n")

	got := runCarrying(t, runs, "quiet")
	if !got.Mono {
		t.Errorf("the code span is %#v, want the monospace", got)
	}
	if got.Mark != "" {
		t.Errorf("the code span carries mark %q, want none", got.Mark)
	}
	if got.Text != "%% quiet %%" {
		t.Errorf("the code span reads %q, want the delimiters as typed", got.Text)
	}

	out := walk(t, "```\n%%\nnot a comment\n%%\n```\n\nAfter.\n")
	if !warned(out.Warnings, "a code block") {
		t.Errorf("a code block holding a fence warned %q, want the code block named", out.Warnings)
	}
	if run := runElement(t, out.Blocks, "After."); run == nil {
		t.Error("the paragraph after the code block is gone, so the fence opened inside the code")
	}
}

// TestAnUnterminatedObsidianFenceIsNamedAndLeftOut. A fence that never closes
// runs to the end of the note, which is what goldmark does with an unclosed
// HTML comment. What is left out is named by line rather than dropped in
// silence, which is this package's rule everywhere else.
func TestAnUnterminatedObsidianFenceIsNamedAndLeftOut(t *testing.T) {
	out := walk(t, "Before.\n\n%%\nan aside\n")

	if !warned(out.Warnings, "comment") {
		t.Errorf("an unclosed fence warned %q, want the comment named", out.Warnings)
	}
	if got := serialise(t, out.Blocks); strings.Contains(got, "an aside") {
		t.Errorf("the unclosed fence reached the document:\n%s", got)
	}
}
