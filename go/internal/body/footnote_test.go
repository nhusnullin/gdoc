package body

// The footnotes: the reference run in the prose, the notes under the page, and
// the three shapes that refuse the build.
//
// Every number here is a literal, because the numbering is the whole claim: a
// reference takes the next number in reading order, whatever order the author
// wrote the definitions in.

import (
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// refused renders a note that must not build and gives back the refusal.
func refused(t *testing.T, markdown string) string {
	t.Helper()
	out, err := Render(config(t), []byte(markdown), "testdata/docs", true)
	if err == nil {
		t.Fatalf("the note built, with %d blocks and %d notes, and should have been refused",
			len(out.Blocks), len(out.Footnotes))
	}
	return err.Error()
}

// footnoteIDs is the number each note under the page carries, in order.
func footnoteIDs(notes []Footnote) []int {
	out := make([]int, 0, len(notes))
	for _, note := range notes {
		out = append(out, note.ID)
	}
	return out
}

// noteText is everything the note's paragraphs say.
func noteText(t *testing.T, note Footnote) string {
	t.Helper()
	var b strings.Builder
	for _, block := range note.Blocks {
		for _, text := range findAll(block.Copy(), "w:t") {
			b.WriteString(text.Text())
		}
	}
	return b.String()
}

// TestAFootnoteReferenceCarriesItsNumber: a reference in the prose is a run of
// its own carrying the number it takes, 1-based and counted in reading order.
// The run's text is empty, because the number is Word's to print.
func TestAFootnoteReferenceCarriesItsNumber(t *testing.T) {
	runs := firstBlockRuns(t, "A claim.[^a] And another.[^b]\n\n[^a]: the first.\n\n[^b]: the second.\n")

	var numbers []int
	for _, run := range runs {
		if run.Footnote == 0 {
			continue
		}
		numbers = append(numbers, run.Footnote)
		if run.Text != "" {
			t.Errorf("the reference run carries the text %q, and the number is Word's to print", run.Text)
		}
	}
	if len(numbers) != 2 || numbers[0] != 1 || numbers[1] != 2 {
		t.Errorf("the reference runs carry %v, want [1 2]", numbers)
	}
	if got := joined(runs); got != "A claim. And another." {
		t.Errorf("the prose reads %q, want the sentence with no marker in it", got)
	}
}

// TestTheBodyCarriesAFootnoteReferenceRun: the run reaches the paragraph as
// Word's own element, in the reference character style and raised, because
// Google Docs keeps no character style and only what the run itself says.
func TestTheBodyCarriesAFootnoteReferenceRun(t *testing.T) {
	out := walk(t, "A claim.[^a]\n\n[^a]: the detail.\n")
	got := serialise(t, out.Blocks)

	if !strings.Contains(got, `<w:footnoteReference w:id="1"/>`) {
		t.Errorf("the body carries no footnote reference for note 1:\n%s", got)
	}
	if !strings.Contains(got, `<w:rStyle w:val="FootnoteReference"/>`) {
		t.Errorf("the reference run names no character style:\n%s", got)
	}
	if !strings.Contains(got, `<w:vertAlign w:val="superscript"/>`) {
		t.Errorf("the reference run is not raised, so Docs would set it on the line:\n%s", got)
	}
	if out.Counts.Footnotes != 1 {
		t.Errorf("counted %d footnotes, want 1", out.Counts.Footnotes)
	}
}

// TestAFootnoteBecomesAFootnoteTextParagraph: each paragraph of a note is set
// in the house's footnote_text style, and the note opens with Word's own
// reference mark, once, however many paragraphs it holds.
func TestAFootnoteBecomesAFootnoteTextParagraph(t *testing.T) {
	out := walk(t, "A claim.[^a]\n\n[^a]: the detail.\n\n    A second paragraph.\n")

	if len(out.Footnotes) != 1 {
		t.Fatalf("the walk produced %d notes, want 1", len(out.Footnotes))
	}
	note := out.Footnotes[0]
	if note.ID != 1 {
		t.Errorf("the note is numbered %d, want 1", note.ID)
	}
	if len(note.Blocks) != 2 {
		t.Fatalf("the note holds %d paragraphs, want the two the author wrote", len(note.Blocks))
	}
	got := serialise(t, note.Blocks)
	if n := strings.Count(got, "<w:footnoteRef/>"); n != 1 {
		t.Errorf("the note carries %d reference marks, want exactly one, on its first paragraph:\n%s", n, got)
	}
	if n := strings.Count(got, `<w:pStyle w:val="FootnoteText"/>`); n != 2 {
		t.Errorf("%d of the note's paragraphs name the footnote style, want both:\n%s", n, got)
	}
	if want := "the detail."; !strings.Contains(got, want) {
		t.Errorf("the note does not carry %q:\n%s", want, got)
	}
	if strings.Contains(got, "[^a]") {
		t.Errorf("the note carries the author's own label:\n%s", got)
	}
}

// TestFootnotesAreNumberedInReferenceOrderNotDefinitionOrder: Word numbers a
// note by where its mark is, so the definition written first is note 2 when
// its mark comes second.
func TestFootnotesAreNumberedInReferenceOrderNotDefinitionOrder(t *testing.T) {
	out := walk(t, "A claim.[^second] And another.[^first]\n\n[^first]: written first.\n\n[^second]: written second.\n")

	if got := footnoteIDs(out.Footnotes); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("the notes are numbered %v, want [1 2]", got)
	}
	if got := noteText(t, out.Footnotes[0]); !strings.Contains(got, "written second") {
		t.Errorf("note 1 reads %q, want the definition the first mark names", got)
	}
	if got := noteText(t, out.Footnotes[1]); !strings.Contains(got, "written first") {
		t.Errorf("note 2 reads %q, want the definition the second mark names", got)
	}
}

// TestAFootnoteReferencedTwiceIsTwoNotesWithOneText: decision 2.2 B. Word and
// Docs hold one note per mark, so the second mark is its own note carrying the
// same words, and the numbers after it carry on.
func TestAFootnoteReferencedTwiceIsTwoNotesWithOneText(t *testing.T) {
	out := walk(t, "One.[^a] Two.[^a] Three.[^b]\n\n[^a]: the shared detail.\n\n[^b]: the other.\n")

	if got := footnoteIDs(out.Footnotes); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("the notes are numbered %v, want [1 2 3]", got)
	}
	if out.Counts.Footnotes != 3 {
		t.Errorf("counted %d footnotes, want the three marks", out.Counts.Footnotes)
	}
	for _, at := range []int{0, 1} {
		if got := noteText(t, out.Footnotes[at]); !strings.Contains(got, "the shared detail") {
			t.Errorf("note %d reads %q, want the shared text", at+1, got)
		}
	}
	if got := noteText(t, out.Footnotes[2]); !strings.Contains(got, "the other") {
		t.Errorf("note 3 reads %q, want the second definition", got)
	}
	// Two notes carrying one text are two elements, not one held twice: the
	// part writes each note into its own w:footnote, and an element written
	// twice is re-parented out of the first.
	if out.Footnotes[0].Blocks[0] == out.Footnotes[1].Blocks[0] {
		t.Error("the two notes share one paragraph element, so writing the second moves it out of the first")
	}
	body := serialise(t, out.Blocks)
	for _, want := range []string{`w:id="1"`, `w:id="2"`, `w:id="3"`} {
		if !strings.Contains(body, want) {
			t.Errorf("the body carries no reference %s:\n%s", want, body)
		}
	}
}

// TestAFootnoteRunSurvivesMerge: merge drops a run with no text, which is what
// a reference run is, and joins neighbours that carry the same formatting. A
// reference joins nothing.
func TestAFootnoteRunSurvivesMerge(t *testing.T) {
	got := merge([]Run{{Text: "A claim."}, {Footnote: 1}, {Text: " More."}})

	if len(got) != 3 {
		t.Fatalf("merge left %d runs, want the three it was handed: %+v", len(got), got)
	}
	if got[1].Footnote != 1 {
		t.Errorf("the middle run is %+v, want the reference to note 1", got[1])
	}
	if got[0].Text != "A claim." || got[2].Text != " More." {
		t.Errorf("merge joined the words across the reference: %+v", got)
	}
}

// TestALinkInsideAFootnoteIsWarnedAndPrintedAsItsWords: a link is a
// relationship in word/_rels/document.xml.rels, which the footnotes part
// cannot name, so the words stay and the link goes. Said on the envelope
// rather than dropped in silence.
func TestALinkInsideAFootnoteIsWarnedAndPrintedAsItsWords(t *testing.T) {
	out := walk(t, "A claim.[^a]\n\n[^a]: see [the policy](https://example.com/p).\n")

	got := serialise(t, out.Footnotes[0].Blocks)
	if strings.Contains(got, "w:hyperlink") {
		t.Errorf("the note carries a hyperlink the footnotes part cannot relate:\n%s", got)
	}
	if !strings.Contains(got, "the policy") {
		t.Errorf("the link's words are gone from the note:\n%s", got)
	}
	if len(out.Media) != 0 {
		t.Errorf("the walk recorded %d relationships for a link inside a footnote, want none", len(out.Media))
	}
	want := "line 3: a link inside a footnote is not rendered in the house style and its words are printed as plain text"
	if !has(out.Warnings, want) {
		t.Errorf("the warnings are %v, want one reading %q", out.Warnings, want)
	}
}

// TestAFootnoteReferenceInATableCellIsRefusedByLine: the spec allows none, and
// a cell's paragraph has no footnotes part to write into.
func TestAFootnoteReferenceInATableCellIsRefusedByLine(t *testing.T) {
	got := refused(t, "| a | b |\n|---|---|\n| x[^a] | y |\n\n[^a]: the detail.\n")

	for _, want := range []string{"line 3", "footnote", "table cell"} {
		if !strings.Contains(got, want) {
			t.Errorf("the refusal does not say %q: %s", want, got)
		}
	}
}

// TestAReferenceWithNoDefinitionIsRefusedByLine. goldmark is silent here: its
// inline parser leaves "[^x]" as prose when no definition follows, so the tree
// says nothing happened and the source line is what is asked. A warning was
// argued and rejected: it lets a published document look finished with a
// citation missing from it.
func TestAReferenceWithNoDefinitionIsRefusedByLine(t *testing.T) {
	got := refused(t, "Words.\n\nA claim.[^missing]\n")

	for _, want := range []string{"line 3", "[^missing]"} {
		if !strings.Contains(got, want) {
			t.Errorf("the refusal does not say %q: %s", want, got)
		}
	}
}

// TestADefinitionNobodyReferencesIsRefusedByLine. goldmark's transformer
// removes a definition nothing references, so again the tree says nothing and
// the source line is what is asked.
func TestADefinitionNobodyReferencesIsRefusedByLine(t *testing.T) {
	got := refused(t, "Words.\n\nA claim.[^a]\n\n[^a]: the detail.\n\n[^spare]: nothing points here.\n")

	for _, want := range []string{"line 7", "[^spare]"} {
		if !strings.Contains(got, want) {
			t.Errorf("the refusal does not say %q: %s", want, got)
		}
	}
}

// TestADefinitionMarkedOnlyFromInsideAnotherIsRefusedByLine: the numbers are
// the prose's reading order and Word has one level of note, so a mark inside a
// definition makes no note. A definition marked only from there would reach no
// page, which is the silence the unused-definition refusal exists for.
func TestADefinitionMarkedOnlyFromInsideAnotherIsRefusedByLine(t *testing.T) {
	got := refused(t, "A claim.[^a]\n\n[^a]: see also [^b].\n\n[^b]: the other.\n")

	for _, want := range []string{"line 5", "[^b]"} {
		if !strings.Contains(got, want) {
			t.Errorf("the refusal does not say %q: %s", want, got)
		}
	}
}

// TestAFootnoteMarkInsideAHeadingIsRefusedByLine: Word copies the mark into
// the contents field, so the heading's number would print a note number in the
// contents list and the mark would point at a note from two places.
func TestAFootnoteMarkInsideAHeadingIsRefusedByLine(t *testing.T) {
	got := refused(t, "## A heading[^a]\n\nWords.\n\n[^a]: the detail.\n")

	for _, want := range []string{"line 1", "heading"} {
		if !strings.Contains(got, want) {
			t.Errorf("the refusal does not say %q: %s", want, got)
		}
	}
}

// TestAFootnoteInsideACodeBlockIsNotAFootnote: the reference and the
// definition checks read the source lines, so a fenced block writing the
// syntax out must not be read as a note. A code block is warned about and left
// out whole, so nothing in it could reach a page anyway.
func TestAFootnoteInsideACodeBlockIsNotAFootnote(t *testing.T) {
	out := walk(t, "Words.\n\n```\nA claim.[^a]\n\n[^a]: the detail.\n```\n")

	if len(out.Footnotes) != 0 {
		t.Errorf("the walk produced %d notes out of a code block, want none", len(out.Footnotes))
	}
}

// TestWhatTheHouseStyleCannotSetInAFootnoteIsWarnedAndLeftOut: a note under
// the page is a sentence, and the house style has no geometry for a table, a
// list or a figure in one. Each is named by its own line and left out, the way
// a picture the house style cannot place is.
func TestWhatTheHouseStyleCannotSetInAFootnoteIsWarnedAndLeftOut(t *testing.T) {
	for _, c := range []struct{ name, markdown, want string }{
		{"a list", "A claim.[^a]\n\n[^a]: the detail.\n\n    - one\n    - two\n",
			"line 3: a list inside a footnote is not rendered in the house style and was left out"},
		{"a table", "A claim.[^a]\n\n[^a]: the detail.\n\n    | a | b |\n    |---|---|\n    | x | y |\n",
			"line 3: a table inside a footnote is not rendered in the house style and was left out"},
		{"a code block", "A claim.[^a]\n\n[^a]: the detail.\n\n    ```\n    code\n    ```\n",
			"line 3: a code block inside a footnote is not rendered in the house style and was left out"},
		{"a heading", "A claim.[^a]\n\n[^a]: the detail.\n\n    # Hidden\n",
			"line 3: a heading inside a footnote is not rendered in the house style and was left out"},
		{"a picture", "A claim.[^a]\n\n[^a]: see ![the chart](diagram.png).\n",
			"line 3: a picture inside a footnote is not rendered in the house style and was left out"},
	} {
		out, err := Render(config(t), []byte(c.markdown), "testdata/docs", true)
		if err != nil {
			t.Fatalf("%s: Render: %v", c.name, err)
		}
		if !has(out.Warnings, c.want) {
			t.Errorf("%s: the warnings are %v, want one reading %q", c.name, out.Warnings, c.want)
		}
		if len(out.Footnotes) != 1 {
			t.Fatalf("%s: the walk produced %d notes, want the one the author wrote", c.name, len(out.Footnotes))
		}
		got := serialise(t, out.Footnotes[0].Blocks)
		if !strings.Contains(got, "the detail.") && !strings.Contains(got, "see") {
			t.Errorf("%s: the note lost its own words:\n%s", c.name, got)
		}
	}
}

// TestAFootnoteInsideAFootnoteIsWarnedAndLeftOut: a reference inside a
// definition takes no number, because the numbers are the prose's reading
// order and a note is not prose. Word has one level of note and no second, so
// the mark is left out and named rather than dropped in silence.
func TestAFootnoteInsideAFootnoteIsWarnedAndLeftOut(t *testing.T) {
	out := walk(t, "A claim.[^a] And.[^b]\n\n[^a]: see also [^b].\n\n[^b]: the other.\n")

	want := "line 3: a footnote inside a footnote is not rendered in the house style and was left out"
	if !has(out.Warnings, want) {
		t.Errorf("the warnings are %v, want one reading %q", out.Warnings, want)
	}
	if got := footnoteIDs(out.Footnotes); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("the notes are numbered %v, want the two marks in the prose", got)
	}
}

// TestAFootnoteCarriesTheMarksItsWordsWere: inside a note, paragraphs with
// inline marks. Bold, italic and a code span come through the one inline walk
// every other paragraph goes through.
func TestAFootnoteCarriesTheMarksItsWordsWere(t *testing.T) {
	out := walk(t, "A claim.[^a]\n\n[^a]: **bold**, *slanted* and `code`.\n")

	got := serialise(t, out.Footnotes[0].Blocks)
	for _, want := range []string{`<w:b w:val="1"/>`, `<w:i w:val="1"/>`, "Consolas"} {
		if !strings.Contains(got, want) {
			t.Errorf("the note does not carry %s:\n%s", want, got)
		}
	}
}

// TestTheNotesParagraphsAreTheWalkersOwnElements: Result.Footnotes holds
// elements nothing else owns, so the part that writes them can add them to a
// tree of its own.
func TestTheNotesParagraphsAreTheWalkersOwnElements(t *testing.T) {
	out := walk(t, "A claim.[^a]\n\n[^a]: the detail.\n")

	for _, block := range out.Footnotes[0].Blocks {
		if block.Parent() != nil {
			t.Errorf("a note's paragraph already has a parent, %s", block.Parent().FullTag())
		}
		if block.FullTag() != "w:p" {
			t.Errorf("a note holds a %s, want a w:p", block.FullTag())
		}
	}
	if out.Blocks[len(out.Blocks)-1].FindElement("//w:footnoteRef") != nil {
		t.Error("a note's reference mark reached the body")
	}
}

// TestANoteWithNoFootnoteCarriesNone: the common note is untouched.
func TestANoteWithNoFootnoteCarriesNone(t *testing.T) {
	out := walk(t, "## Heading\n\nWords, and a [link](https://example.com).\n")

	if len(out.Footnotes) != 0 {
		t.Errorf("the walk produced %d notes for a note with none", len(out.Footnotes))
	}
	if out.Counts.Footnotes != 0 {
		t.Errorf("counted %d footnotes, want none", out.Counts.Footnotes)
	}
	for _, warning := range out.Warnings {
		if strings.Contains(warning, "footnote") {
			t.Errorf("a note with no footnote warned about one: %q", warning)
		}
	}
}

// TestAFootnoteReferenceIsNotADeadAnchor guards the one place a reference run
// could be mistaken for something else: it carries no link, so the dead-anchor
// branch never sees it.
func TestAFootnoteReferenceIsNotADeadAnchor(t *testing.T) {
	r := &renderer{cfg: config(t), anchors: map[string]bool{}, deadAnchors: map[string]bool{}}
	p := etree.NewElement("w:p")
	r.addRuns(p, []Run{{Footnote: 4}}, runOpts{})

	if len(r.warnings) != 0 {
		t.Errorf("a reference run warned: %v", r.warnings)
	}
	ref := p.FindElement("//w:footnoteReference")
	if ref == nil {
		t.Fatalf("addRuns wrote no footnote reference:\n%s", p.FullTag())
	}
	if got := ref.SelectAttrValue("w:id", ""); got != "4" {
		t.Errorf("the reference names note %q, want 4", got)
	}
}
