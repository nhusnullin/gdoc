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

// A break before nothing is nothing. The author wrote a divider at the foot of
// the note, which is a habit rather than a page they want: the page it opens
// would hold not one word. No warning either, because what they meant is what
// the document got.
func TestAPageBreakBeforeNothingIsNothing(t *testing.T) {
	out := walk(t, "The only paragraph.\n\n---\n")

	if n := len(out.Blocks); n != 1 {
		t.Fatalf("%d blocks rendered, want the paragraph alone:\n%s", n, serialise(t, out.Blocks))
	}
	if body := serialise(t, out.Blocks); strings.Contains(body, "w:pageBreakBefore") {
		t.Error("a trailing break reached the document")
	}
	if n := len(out.Warnings); n != 0 {
		t.Errorf("the dropped break warned: %v", out.Warnings)
	}
	if out.Counts.PageBreaks != 0 {
		t.Errorf("Counts.PageBreaks is %d, want 0: it counts the paragraphs emitted", out.Counts.PageBreaks)
	}
}

// `---` before the footnote definitions is the line internal/view prints and
// internal/export writes, so a note that came home out of a document carries
// one. It is not the author's break at all, and a note exported and published
// again must not grow a blank last page. goldmark lifts the definitions into a
// FootnoteList at the end of the document, which is what the look-ahead reads.
func TestTheFootnoteSeparatorIsNotAPageBreak(t *testing.T) {
	out := walk(t, "A claim.[^1]\n\n---\n\n[^1]: The note under the page.\n")

	if body := serialise(t, out.Blocks); strings.Contains(body, "w:pageBreakBefore") {
		t.Errorf("the footnote separator reached the document as a break:\n%s", body)
	}
	if n := len(out.Footnotes); n != 1 {
		t.Fatalf("%d notes under the page, want the one the claim marks", n)
	}
	if text := strings.TrimSpace(serialiseNotes(t, out.Footnotes)); !strings.Contains(text, "The note under the page.") {
		t.Errorf("the note's words did not reach the part: %s", text)
	}
	if n := len(out.Warnings); n != 0 {
		t.Errorf("the separator warned: %v", out.Warnings)
	}
	if out.Counts.PageBreaks != 0 {
		t.Errorf("Counts.PageBreaks is %d, want 0", out.Counts.PageBreaks)
	}
}

// The house breaks the page before the first body heading, so a break the
// author wrote directly before that heading would open a page holding nothing
// and break again. The heading's own break is the one that stands.
func TestABreakBeforeTheFirstHeadingIsTheHeadingsOwn(t *testing.T) {
	out := walk(t, "An opening line.\n\n---\n\n# Purpose\n\nThe body.\n")

	if n := len(out.Blocks); n != 3 {
		t.Fatalf("%d blocks rendered, want paragraph, heading, paragraph:\n%s", n, serialise(t, out.Blocks))
	}
	breaks := 0
	for _, block := range out.Blocks {
		breaks += len(findAll(block, "w:pageBreakBefore"))
	}
	if breaks != 1 {
		t.Fatalf("%d breaks reached the document, want the heading's one:\n%s", breaks, serialise(t, out.Blocks))
	}
	if style := find(out.Blocks[1], "w:pStyle"); style == nil || find(out.Blocks[1], "w:pageBreakBefore") == nil {
		t.Errorf("the one break is not the heading's:\n%s", serialise(t, out.Blocks))
	}
	if n := len(out.Warnings); n != 0 {
		t.Errorf("the dropped break warned: %v", out.Warnings)
	}
	if out.Counts.PageBreaks != 0 {
		t.Errorf("Counts.PageBreaks is %d, want 0: the break the document got is the heading's", out.Counts.PageBreaks)
	}
}

// Two breaks in a row are one. A second empty paragraph carrying a break makes
// a page holding nothing between the two the author meant.
func TestTwoBreaksInARowAreOne(t *testing.T) {
	out := walk(t, "Before.\n\n---\n\n---\n\n---\n\nAfter.\n")

	if n := len(out.Blocks); n != 3 {
		t.Fatalf("%d blocks rendered, want paragraph, break, paragraph:\n%s", n, serialise(t, out.Blocks))
	}
	if find(out.Blocks[1], "w:pageBreakBefore") == nil {
		t.Errorf("the middle block is not the break:\n%s", serialise(t, out.Blocks))
	}
	if out.Counts.PageBreaks != 1 {
		t.Errorf("Counts.PageBreaks is %d, want 1", out.Counts.PageBreaks)
	}
	if n := len(out.Warnings); n != 0 {
		t.Errorf("the collapsed breaks warned: %v", out.Warnings)
	}
}

// Nothing else is collapsed. A break between two paragraphs is the author
// saying where a page ends, however many of them the note holds, so six
// paragraphs separated by five breaks are six pages.
func TestABreakBetweenTwoParagraphsIsAlwaysHonoured(t *testing.T) {
	note := "One.\n\n---\n\nTwo.\n\n---\n\nThree.\n\n---\n\nFour.\n\n---\n\nFive.\n\n---\n\nSix.\n"
	out := walk(t, note)

	if n := len(out.Blocks); n != 11 {
		t.Fatalf("%d blocks rendered, want six paragraphs and five breaks:\n%s", n, serialise(t, out.Blocks))
	}
	for i := 1; i < len(out.Blocks); i += 2 {
		if find(out.Blocks[i], "w:pageBreakBefore") == nil {
			t.Errorf("block %d is not a break:\n%s", i, serialise(t, out.Blocks))
		}
	}
	if out.Counts.PageBreaks != 5 {
		t.Errorf("Counts.PageBreaks is %d, want 5", out.Counts.PageBreaks)
	}
	if n := len(out.Warnings); n != 0 {
		t.Errorf("an honoured break warned: %v", out.Warnings)
	}
}

// A break inside a footnote definition or a block quote is a page break in a
// place the house style has no page to break: the notes sit under the page
// Word puts them on, and a quote is a run of indented paragraphs a page
// turning through would split. Each is named by line and left out, the way a
// picture in a footnote is.
func TestABreakInsideAFootnoteOrAQuoteIsWarnedAndLeftOut(t *testing.T) {
	for _, c := range []struct {
		what string
		note string
		says string
		line string
	}{
		{
			what: "a footnote",
			note: "A claim.[^1]\n\n[^1]: The note.\n\n    ---\n\n    More of it.\n",
			says: "a page break inside a footnote",
			// The break's own line, five lines down, not the line the
			// definition opens on: a long note is where naming the wrong one
			// costs the author the most.
			line: "line 5: ",
		},
		{
			what: "a block quote",
			note: "Prose.\n\n> Quoted words.\n>\n> ---\n>\n> More quoted words.\n",
			says: "a page break inside a block quote",
			line: "line 5: ",
		},
	} {
		t.Run(c.what, func(t *testing.T) {
			out := walk(t, c.note)
			if body := serialise(t, out.Blocks); strings.Contains(body, "w:pageBreakBefore") {
				t.Errorf("a break inside %s reached the document:\n%s", c.what, body)
			}
			if out.Counts.PageBreaks != 0 {
				t.Errorf("Counts.PageBreaks is %d, want 0", out.Counts.PageBreaks)
			}
			if len(out.Warnings) != 1 {
				t.Fatalf("%d warnings, want the one naming the break: %v", len(out.Warnings), out.Warnings)
			}
			said := out.Warnings[0]
			if !strings.Contains(said, c.says) {
				t.Errorf("the warning does not say what was left out: %q, want %q in it", said, c.says)
			}
			if !strings.HasPrefix(said, c.line) {
				t.Errorf("the warning is %q, want it to open with %q", said, c.line)
			}
		})
	}
}

// Three hyphens in a table cell are three hyphens. A pipe table's cell holds
// inline text alone, so goldmark builds no thematic break there at all and
// there is nothing for the walker to leave out. The spec's own sentence said
// the walker warns about one; it does not, because nothing reaches it.
func TestThreeHyphensInACellAreNotABreak(t *testing.T) {
	out := walk(t, "| Heading | Second |\n|---|---|\n| A cell | --- |\n")

	body := serialise(t, out.Blocks)
	if strings.Contains(body, "w:pageBreakBefore") {
		t.Errorf("a cell's hyphens reached the document as a break:\n%s", body)
	}
	if out.Counts.PageBreaks != 0 {
		t.Errorf("Counts.PageBreaks is %d, want 0", out.Counts.PageBreaks)
	}
	if n := len(out.Warnings); n != 0 {
		t.Errorf("the cell warned: %v", out.Warnings)
	}
	if out.Counts.Tables != 1 {
		t.Errorf("Counts.Tables is %d, want the one table", out.Counts.Tables)
	}
}

// A break that ends a list item is a break before the next item, not a break
// before nothing. The look-ahead reads the next block the document holds
// rather than the break's own next sibling, which inside an item is nothing at
// all: the author's page turn between two items used to disappear without a
// word, and whether it survived depended on whether more of the same item
// followed it.
func TestABreakEndingAListItemIsTheBreakBeforeTheNextOne(t *testing.T) {
	out := walk(t, "1. one\n\n   ***\n\n2. two\n")

	if n := len(out.Blocks); n != 3 {
		t.Fatalf("%d blocks rendered, want item, break, item:\n%s", n, serialise(t, out.Blocks))
	}
	if find(out.Blocks[1], "w:pageBreakBefore") == nil {
		t.Errorf("the middle block is not the break:\n%s", serialise(t, out.Blocks))
	}
	if out.Counts.PageBreaks != 1 {
		t.Errorf("Counts.PageBreaks is %d, want 1", out.Counts.PageBreaks)
	}
	if n := len(out.Warnings); n != 0 {
		t.Errorf("the honoured break warned: %v", out.Warnings)
	}
}

// A break that ends the last item of a list, with prose after the list, is
// honoured for the same reason, and the list still ends on the item above it:
// the break paragraph holds nothing and states zero space on both sides, so
// the space the body puts under a list belongs to the item.
func TestAListEndingOnABreakEndsOnTheItemAboveIt(t *testing.T) {
	out := walk(t, "1. one\n\n   ***\n\nProse.\n")

	if n := len(out.Blocks); n != 3 {
		t.Fatalf("%d blocks rendered, want item, break, paragraph:\n%s", n, serialise(t, out.Blocks))
	}
	if find(out.Blocks[1], "w:pageBreakBefore") == nil {
		t.Fatalf("the middle block is not the break:\n%s", serialise(t, out.Blocks))
	}
	if got := attr(t, find(out.Blocks[0], "w:spacing"), "w:after"); got != "240" {
		t.Errorf("the item has %s twips below it, want the body's 240", got)
	}
	if got := attr(t, find(out.Blocks[1], "w:spacing"), "w:after"); got != "0" {
		t.Errorf("the break has %s twips below it, want 0", got)
	}
	if out.Counts.PageBreaks != 1 {
		t.Errorf("Counts.PageBreaks is %d, want 1", out.Counts.PageBreaks)
	}
}

// Two items each holding one break are one break, the way two breaks in a row
// in the body are: the look-ahead descends into the item after the break, so
// the break that ends the first item sees the break that opens the second and
// stands down. Reading the item itself as the next block emitted both and the
// document opened a page holding nothing.
func TestTwoItemsEachHoldingABreakAreOneBreak(t *testing.T) {
	for _, note := range []string{
		"- ***\n- ***\n\nProse.\n",
		"1. one\n\n   ***\n\n2. ***\n\n   two\n",
	} {
		t.Run(note, func(t *testing.T) {
			out := walk(t, note)

			if body := serialise(t, out.Blocks); strings.Count(body, "w:pageBreakBefore") != 1 {
				t.Errorf("%d break paragraphs, want the one:\n%s", strings.Count(body, "w:pageBreakBefore"), body)
			}
			if out.Counts.PageBreaks != 1 {
				t.Errorf("Counts.PageBreaks is %d, want 1", out.Counts.PageBreaks)
			}
		})
	}
}

// A break that ends a list ending the note is still a break before nothing,
// because nothing in the document follows the list either.
func TestABreakEndingTheLastListItemOfTheNoteIsNothing(t *testing.T) {
	out := walk(t, "1. one\n\n   ***\n")

	if body := serialise(t, out.Blocks); strings.Contains(body, "w:pageBreakBefore") {
		t.Errorf("a trailing break inside a list reached the document:\n%s", body)
	}
	if out.Counts.PageBreaks != 0 {
		t.Errorf("Counts.PageBreaks is %d, want 0", out.Counts.PageBreaks)
	}
	if n := len(out.Warnings); n != 0 {
		t.Errorf("the dropped break warned: %v", out.Warnings)
	}
}

// Every break inside a quote is warned about, wherever it stands in the quote:
// the quote is asked before the look-ahead, which reads a break that ends the
// quote as a break before nothing and would drop it in silence.
func TestEveryBreakInsideAQuoteIsWarnedAbout(t *testing.T) {
	for _, c := range []struct {
		what  string
		note  string
		warns int
	}{
		{what: "ending the quote", note: "> Quoted.\n>\n> ---\n\nProse.\n", warns: 1},
		{what: "two in a row", note: "> Quoted.\n>\n> ---\n>\n> ---\n>\n> More.\n", warns: 2},
		{what: "ending the note", note: "Prose.\n\n> Quoted.\n>\n> ---\n", warns: 1},
	} {
		t.Run(c.what, func(t *testing.T) {
			out := walk(t, c.note)
			if body := serialise(t, out.Blocks); strings.Contains(body, "w:pageBreakBefore") {
				t.Errorf("a break inside a quote reached the document:\n%s", body)
			}
			if out.Counts.PageBreaks != 0 {
				t.Errorf("Counts.PageBreaks is %d, want 0", out.Counts.PageBreaks)
			}
			if n := len(out.Warnings); n != c.warns {
				t.Fatalf("%d warnings, want %d naming the break: %v", n, c.warns, out.Warnings)
			}
			for _, said := range out.Warnings {
				if !strings.Contains(said, "a page break inside a block quote") {
					t.Errorf("the warning does not say what was left out: %q", said)
				}
			}
		})
	}
}

// The warning names the line the break is on. goldmark appends no lines to a
// thematic break and builds no text inside one, so the line comes from the
// node's own position: reading its lines named line 1 for every break in every
// note, which in a note is the front matter's own delimiter.
func TestABreakInsideAQuoteNamesItsLine(t *testing.T) {
	out := walk(t, "Prose.\n\n> Quoted words.\n>\n> ---\n>\n> More quoted words.\n")

	if len(out.Warnings) != 1 {
		t.Fatalf("%d warnings, want the one naming the break: %v", len(out.Warnings), out.Warnings)
	}
	if said := out.Warnings[0]; !strings.HasPrefix(said, "line 5: ") {
		t.Errorf("the warning is %q, want it to name line 5, where the break stands", said)
	}
}
