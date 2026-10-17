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

// TestTheOldSpanIsRefusedByLineNamingTheComment. The grey mark was a Pandoc
// span until this milestone, "[words]{.pencil}", and the spelling is gone:
// nothing parses it, so a note that still carries it would publish the
// brackets and the braces as words. It is refused by line instead, and the
// refusal names the comment to write in its place, because an author who
// typed the old spelling knows what they meant and needs the new one.
func TestTheOldSpanIsRefusedByLineNamingTheComment(t *testing.T) {
	for _, c := range []struct{ note, names string }{
		{"Words.\n\nA [a note]{.pencil} here.\n", "<!-- a note -->"},
		{"Words.\n\nA ]{.pencil} with no bracket in front of it.\n", "<!-- words -->"},
	} {
		got := refused(t, c.note)
		if !strings.Contains(got, "line 3:") {
			t.Errorf("%q: the refusal is %q, want it to name line 3", c.note, got)
		}
		if !strings.Contains(got, c.names) {
			t.Errorf("%q: the refusal is %q, and it does not name %q to write instead",
				c.note, got, c.names)
		}
		if !strings.Contains(got, "%%") {
			t.Errorf("%q: the refusal is %q, and it names one spelling of two", c.note, got)
		}
	}
}

// The other side of it: the old spelling where nothing is markdown is an
// author writing about gdoc rather than marking words, which is the rule every
// line check in this package already follows. The guide holds the span in a
// code span, and a note explaining the change would be refused for explaining
// it. A block of HTML is the fourth place, left out of the document whole.
func TestTheOldSpanWhereNothingIsMarkdownIsNotRefused(t *testing.T) {
	for _, source := range []string{
		"Words.\n\nThe old spelling was `[words]{.pencil}` once.\n",
		"Words.\n\n```\n[words]{.pencil}\n```\n",
		"Words.\n\n    [words]{.pencil}\n",
		"Words.\n\n<div>\n[a]{.pencil}\n</div>\n",
	} {
		if _, err := Render(config(t), []byte(source), "testdata/docs", true); err != nil {
			t.Errorf("%q was refused: %v", source, err)
		}
	}
}

// TestAnHTMLCommentInCodeIsCode. Nothing inside backticks is markdown, and the
// HTML spelling is no exception: a note explaining the syntax shows it in a
// code span, and the words have to come out of the document as the author
// typed them rather than grey. TestAnObsidianCommentInCodeIsCode is the same
// line for the other spelling, where this package's own parser has to be told;
// here goldmark's code span is what holds it, and the acceptance table for
// milestone 20 names both.
func TestAnHTMLCommentInCodeIsCode(t *testing.T) {
	runs := firstBlockRuns(t, "a `<!-- quiet -->` b\n")

	got := runCarrying(t, runs, "quiet")
	if !got.Mono {
		t.Errorf("the code span is %#v, want the monospace", got)
	}
	if got.Mark != "" {
		t.Errorf("the code span carries mark %q, want none", got.Mark)
	}
	if got.Text != "<!-- quiet -->" {
		t.Errorf("the code span reads %q, want the delimiters as typed", got.Text)
	}

	out := walk(t, "```\n<!-- not a comment -->\n```\n\nAfter.\n")
	if !warned(out.Warnings, "a code block") {
		t.Errorf("a code block holding a comment warned %q, want the code block named", out.Warnings)
	}
	if got := serialise(t, out.Blocks); strings.Contains(got, "not a comment") {
		t.Errorf("the fenced comment reached the document:\n%s", got)
	}
	if run := runElement(t, out.Blocks, "After."); run == nil {
		t.Error("the paragraph after the code block is gone, so the comment opened inside the code")
	}
}

// TestACommentWithProseAfterItIsAPencilAndThatProse. CommonMark opens a block
// of HTML on a line beginning "<!--" and gives it the whole line, so the
// author's sentence after the closer was left out with the comment. Obsidian
// shows that line as the sentence with the aside hidden, and internal/view
// writes exactly this shape whenever a paragraph opens with a grey run, so a
// document exported and published again lost the paragraph.
//
// The markdown after the comment is markdown, which is what makes this a
// paragraph rather than a second reading of a block of HTML.
func TestACommentWithProseAfterItIsAPencilAndThatProse(t *testing.T) {
	runs := firstBlockRuns(t, "<!-- todo --> We **ship** in May.\n")

	if len(runs) != 4 {
		t.Fatalf("the line built %d runs, want the aside and the sentence: %#v", len(runs), runs)
	}
	if runs[0].Mark != "pencil" || runs[0].Text != "todo" {
		t.Errorf("the first run is %#v, want the grey aside", runs[0])
	}
	if runs[1].Mark != "" || runs[1].Text != " We " {
		t.Errorf("the words after the comment are %#v, want them black", runs[1])
	}
	if !runs[2].Bold || runs[2].Text != "ship" {
		t.Errorf("the bold word came back as %#v, want it bold: what follows a comment is markdown", runs[2])
	}

	// The whole line reaches the document, and no warning says anything was
	// left out.
	out := walk(t, "<!-- todo --> We ship in May.\n")
	if len(out.Warnings) != 0 {
		t.Errorf("the line warned %q, and nothing was left out", out.Warnings)
	}
	if got := serialise(t, out.Blocks); !strings.Contains(got, "We ship in May.") {
		t.Errorf("the author's sentence never reached the document:\n%s", got)
	}

	// A list item and a block quote are the same line with a marker in front
	// of it, and each one lost the same sentence.
	for _, source := range []string{"- <!-- aside --> item text\n", "> <!-- aside --> quoted text\n"} {
		out := walk(t, source)
		if got := serialise(t, out.Blocks); !strings.Contains(got, "text") {
			t.Errorf("%q lost its words:\n%s", source, got)
		}
		if len(out.Warnings) != 0 {
			t.Errorf("%q warned %q, and nothing was left out", source, out.Warnings)
		}
	}
}

// TestTwoCommentsOnOneLineAreTwoPencils. goldmark shuts a block of HTML on the
// first line holding the closer, so two comments on one line of their own were
// one block, and reading it as one comment put both delimiters into the
// document as grey words.
func TestTwoCommentsOnOneLineAreTwoPencils(t *testing.T) {
	runs := firstBlockRuns(t, "<!-- first --> <!-- second -->\n")

	if len(runs) != 3 {
		t.Fatalf("the line built %d runs, want two asides and the space between them: %#v", len(runs), runs)
	}
	if runs[0].Mark != "pencil" || runs[0].Text != "first" {
		t.Errorf("the first aside is %#v", runs[0])
	}
	if runs[2].Mark != "pencil" || runs[2].Text != "second" {
		t.Errorf("the second aside is %#v", runs[2])
	}
	for _, run := range runs {
		if strings.Contains(run.Text, commentOpener) || strings.Contains(run.Text, commentCloser) {
			t.Errorf("a delimiter reached the document as words: %#v", runs)
		}
	}

	// An aside on each side of the author's own words is the same block, and
	// the prose between them stays black.
	around := firstBlockRuns(t, "<!-- a --> real prose <!-- b -->\n")
	if len(around) != 3 || around[1].Mark != "" || around[1].Text != " real prose " {
		t.Fatalf("the prose between two asides came back as %#v", around)
	}

	// Wrapped over lines the pair is goldmark's block of HTML to the end, so
	// there is no pencil to read and the loss is named as the comment it is.
	out := walk(t, "<!-- a\nb --> <!-- c -->\n")
	if len(out.Blocks) != 0 {
		t.Errorf("the wrapped pair built %d blocks, want none:\n%s", len(out.Blocks), serialise(t, out.Blocks))
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "words after its closer") {
		t.Errorf("the wrapped pair warned %q, want the one sentence naming the comment", out.Warnings)
	}
}

// TestAWrappedCommentInsideASentenceIsOneLine. goldmark hands a comment the
// author's editor wrapped back with the newline inside it, and a newline in a
// run is a hard line break in the published paragraph. A line break inside a
// comment means no more than it does in prose, which is the rule the block
// form already follows.
func TestAWrappedCommentInsideASentenceIsOneLine(t *testing.T) {
	runs := firstBlockRuns(t, "Prose <!-- an aside that\nwraps --> more.\n")

	got := runCarrying(t, runs, "aside")
	if got.Mark != "pencil" {
		t.Errorf("the wrapped comment is mark %q, want pencil", got.Mark)
	}
	if got.Text != "an aside that wraps" {
		t.Errorf("the wrapped comment marks %q, want its lines joined by one space", got.Text)
	}
	if strings.Contains(got.Text, "\n") {
		t.Errorf("the comment carries a newline, which is a line break the author never typed: %q", got.Text)
	}
}

// TestACommentsWordsCarryNoSpaceOnEitherEdge. One space inside each delimiter
// is the delimiter's, the way the space after a highlight's colour emoji is.
// A second one is trimmed as well: both readings hand the words to
// commentParagraphs, which trims each line, so a grey run that opened or shut
// on a space the author never meant as a word is not one the walk can build.
func TestACommentsWordsCarryNoSpaceOnEitherEdge(t *testing.T) {
	runs := firstBlockRuns(t, "Prose <!--  an aside  --> more.\n")

	got := runCarrying(t, runs, "aside")
	if got.Mark != "pencil" || got.Text != "an aside" {
		t.Errorf("the comment marks %#v, want the words with no space on either edge", got)
	}

	// The block form is the same reading, and was already this one.
	out := walk(t, "<!--  an aside  -->\n")
	if got := serialise(t, out.Blocks); !strings.Contains(got, ">an aside<") {
		t.Errorf("the block form came back as\n%s", got)
	}
}

// TestAPencilsLineWrapsLikeAnyOtherParagraph. The line a pencil opens is a
// paragraph, so the line under it is that paragraph's next line, the way it
// is under any other line of prose and the way Obsidian shows it.
//
// Shutting the paragraph on every line lost that next line whole and said
// nothing: goldmark treats an open paragraph as continuable, skips Continue
// and offers the line only to the parsers that may interrupt a paragraph, and
// its own paragraph parser is not one of them. Nothing opened, so the line
// reached no node at all.
func TestAPencilsLineWrapsLikeAnyOtherParagraph(t *testing.T) {
	for _, c := range []struct{ source, lost string }{
		{"<!-- todo --> Revenue grew 10%.\nCosts fell 5%.\n", "Costs fell 5%."},
		{"- <!-- a --> item\n  more\n", "more"},
		{"> <!-- a --> b\n> more\n", "more"},
	} {
		out := walk(t, c.source)
		got := serialise(t, out.Blocks)
		if !strings.Contains(got, c.lost) {
			t.Errorf("%q lost %q and warned %q:\n%s", c.source, c.lost, out.Warnings, got)
		}
		if len(out.Warnings) != 0 {
			t.Errorf("%q warned %q, and nothing was left out", c.source, out.Warnings)
		}
	}

	// A second comment the author's editor wrapped reaches its closer on the
	// paragraph's next line, so it is the pencil it is rather than black words
	// opening with a delimiter.
	runs := firstBlockRuns(t, "<!-- a --> We ship <!-- check\nwith legal -->\n")
	got := runCarrying(t, runs, "check")
	if got.Mark != "pencil" || got.Text != "check with legal" {
		t.Errorf("the wrapped second comment came back as %#v, want the grey aside", got)
	}
	for _, run := range runs {
		if strings.Contains(run.Text, commentOpener) || strings.Contains(run.Text, commentCloser) {
			t.Errorf("a delimiter reached the document as words: %#v", runs)
		}
	}

	// A blank line still ends the paragraph, which is what makes the next
	// stretch its own.
	out := walk(t, "<!-- a --> text\n\nmore\n")
	if len(out.Blocks) != 2 {
		t.Errorf("the blank line built %d blocks, want two:\n%s", len(out.Blocks), serialise(t, out.Blocks))
	}
}

// TestAnEmptyCommentInsideAQuoteIsLeftOut. "Empty is nothing" holds wherever a
// paragraph is emitted, and a quote is the one path that emits one without
// going through paragraphBlock: a comment holding no words left an indented
// blank line the author never typed.
func TestAnEmptyCommentInsideAQuoteIsLeftOut(t *testing.T) {
	for _, source := range []string{"> %%%%\n", "> %% %%\n", "> <!-- -->\n"} {
		out := walk(t, source)
		if len(out.Blocks) != 0 {
			t.Errorf("%q built %d blocks, want none:\n%s", source, len(out.Blocks), serialise(t, out.Blocks))
		}
		if out.Counts.Paragraphs != 0 {
			t.Errorf("%q counted %d paragraphs, want none", source, out.Counts.Paragraphs)
		}
	}
}

// TestAPencilsRefusalNamesItsOwnLine. A comment is the one inline node that
// holds its characters as segments rather than as text children, so a heading
// whose whole content is one carries no ast.Text at all: the line lookup
// climbed past the heading and named the first text anywhere in the note,
// which sent the author to line 1 of a file whose problem is on line 5. The
// span this spelling replaces had text children and never showed it.
func TestAPencilsRefusalNamesItsOwnLine(t *testing.T) {
	for _, note := range []string{
		"Words.\n\nMore words.\n\n## <!-- tbd -->\n",
		"Words.\n\nMore words.\n\n## %% tbd %%\n",
		"Words.\n\nMore words.\n\n![<!-- tbd -->](badge.png)\n",
	} {
		if got := refused(t, note); !strings.Contains(got, "line 5:") {
			t.Errorf("%q: the refusal is %q, want it to name line 5", note, got)
		}
	}
}

// TestAnUnclosedHTMLCommentIsNamedAsAComment. The milestone named the unclosed
// Obsidian fence as the comment it is and left the HTML one as "a block of
// HTML", which is the spelling the guide teaches and the one export writes. An
// author told they wrote a block of HTML goes looking for a tag that is not in
// the note, and goldmark gives an unclosed comment every line to the end of
// the note, so the sentence says that too.
func TestAnUnclosedHTMLCommentIsNamedAsAComment(t *testing.T) {
	out := walk(t, "Before.\n\n<!-- an aside with no closer\n\nAfter the aside.\n")

	if !warned(out.Warnings, "never closed") {
		t.Errorf("an unclosed comment warned %q, want it named as a comment", out.Warnings)
	}
	if warned(out.Warnings, "a block of HTML") {
		t.Errorf("an unclosed comment warned %q, want no tag named", out.Warnings)
	}
	if !warned(out.Warnings, "every line after it") {
		t.Errorf("an unclosed comment warned %q, want the lines under it named too", out.Warnings)
	}
	if got := serialise(t, out.Blocks); strings.Contains(got, "an aside") {
		t.Errorf("the unclosed comment reached the document:\n%s", got)
	}
}

// The other side of it: a block of HTML that is no comment keeps the sentence
// it has always had, because that author did write a tag.
func TestABlockOfHTMLThatIsNoCommentIsStillNamedAsOne(t *testing.T) {
	out := walk(t, "Before.\n\n<div>\nnot a comment\n</div>\n")

	if !warned(out.Warnings, "a block of HTML") {
		t.Errorf("a block of HTML warned %q, want the block named", out.Warnings)
	}
}

// TestAnUnclosedCommentInsideASentenceIsRefusedByLine. An opener on a line of
// its own opens a block, which the warning above names and leaves out. An
// opener in the middle of a sentence opens nothing: goldmark reads raw HTML
// only once it has the closer, so the aside stayed prose, black, with the
// typographer rewriting the two dashes into an en dash on the way. The author
// wrote words nobody was meant to read and the build said ok, which is the
// one failure every line check in this package exists to prevent, so the note
// is refused by line the way the old span is.
func TestAnUnclosedCommentInsideASentenceIsRefusedByLine(t *testing.T) {
	for _, note := range []string{
		"Words.\n\nSome prose <!-- an aside with no closer\n",
		"Words.\n\nSome prose <!--an aside with no closer\n",
		"Words.\n\nSome prose <!-- an aside\nthat wraps with no closer\n",
		"Words.\n\nA <!-- shut one --> and <!-- one left open\n",
	} {
		got := refused(t, note)
		if !strings.Contains(got, "line 3:") {
			t.Errorf("%q: the refusal is %q, want it to name line 3", note, got)
		}
		if !strings.Contains(got, commentOpener) {
			t.Errorf("%q: the refusal is %q, and it does not name the opener", note, got)
		}
		if !strings.Contains(got, commentCloser) {
			t.Errorf("%q: the refusal is %q, and it does not name the closer to write", note, got)
		}
	}
}

// The other side of it: every shape where goldmark did read the opener, and
// every shape where nothing on the line is markdown. A comment that wraps over
// lines is one comment, an opener inside a comment's own words is that
// comment's words, and an opener inside backticks or inside a code block is
// the author writing about the syntax rather than using it.
func TestAnOpenerGoldmarkReadIsNotRefused(t *testing.T) {
	for _, note := range []string{
		"Words.\n\nA <!-- quiet --> aside.\n",
		"Words.\n\nA <!-- an aside\nthat wraps --> over lines.\n",
		"Words.\n\nA <!-- aside about <!-- itself --> here.\n",
		"Words.\n\nA <!-- one --> and <!-- two --> on a line.\n",
		"Words.\n\nThe opener is `<!--` and the closer is `-->`.\n",
		"Words.\n\nA <!-- an aside holding `<!--` in code --> here.\n",
		"Words.\n\n```\n<!-- an aside with no closer\n```\n",
		"Words.\n\n    <!-- an aside with no closer\n",
		"Words.\n\n<!-- an aside with no closer\n",
		"Words.\n\n<div>\n<!-- an aside with no closer\n</div>\n",
	} {
		if _, err := Render(config(t), []byte(note), "testdata/docs", true); err != nil {
			t.Errorf("%q was refused: %v", note, err)
		}
	}
}

// TestAnEscapedOpenerIsNotRefused. A backslash makes the one character after
// it the author's own, which is the rule `read` writes home with and the one
// way a note about gdoc's own comment markers publishes at all: the marker
// refusal leaves an escaped marker alone, and TestBuildLeavesAnEscapedMarkerAlone
// is its pin. An escaped opener opens nothing and is meant to open nothing,
// so the characters the author typed are exactly what reaches the page and
// there is no closer for the line to be missing.
func TestAnEscapedOpenerIsNotRefused(t *testing.T) {
	for _, note := range []string{
		"Words.\n\nAn escaped opener \\<!-- aside --> here.\n",
		"Words.\n\nAn escaped opener \\<!-- with no closer.\n",
		"Words.\n\nExport writes \\<!-- image: picture.png --> into the note.\n",
		"Words.\n\nA \\<!-- escaped one and a <!-- read one --> here.\n",
		"Words.\n\nA <!-- aside about \\<!-- itself --> here.\n",
	} {
		if _, err := Render(config(t), []byte(note), "testdata/docs", true); err != nil {
			t.Errorf("%q was refused: %v", note, err)
		}
	}
}

// The other side of the parity: two backslashes are one of the author's own,
// so the opener after them is gdoc's reading again and an opener nothing
// closes is still refused.
func TestAnOpenerAfterAnEscapedBackslashIsStillRefused(t *testing.T) {
	note := "Words.\n\nAn escaped backslash \\\\<!-- and no closer.\n"
	if got := refused(t, note); !strings.Contains(got, "line 3:") {
		t.Errorf("the refusal is %q, want it to name line 3", got)
	}
}
