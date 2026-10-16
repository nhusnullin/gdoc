package body

// The seven marks as the parser reads them: Obsidian's colour emoji on a
// highlight, the one Pandoc span gdoc reads, and the five shapes that refuse
// the build.
//
// Every emoji and every mark name here is written out. The parser reads the
// palette off house.yaml, so a test that read it too would follow the palette
// wherever somebody moved it and claim nothing.

import (
	"slices"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// TestAHighlightOpeningWithAColourEmojiCarriesItsMark is Obsidian 1.14's own
// syntax: one of six emoji at the start of a highlight picks its colour. The
// emoji is the instruction and not the words, so it does not reach the page.
// A plain highlight is yellow, and 🟡 says the same thing.
func TestAHighlightOpeningWithAColourEmojiCarriesItsMark(t *testing.T) {
	cases := []struct {
		highlight string
		mark      string
	}{
		{"==a gap==", "yellow"},
		{"==🟡 a gap==", "yellow"},
		{"==🟡a gap==", "yellow"},
		{"==🔴 a gap==", "red"},
		{"==🟠 a gap==", "orange"},
		{"==🟢 a gap==", "green"},
		{"==🔵 a gap==", "blue"},
		{"==🟣 a gap==", "purple"},
	}
	for _, c := range cases {
		runs := firstBlockRuns(t, "A sentence with "+c.highlight+" in it.\n")
		got := runCarrying(t, runs, "a gap")
		if got.Mark != c.mark {
			t.Errorf("%s is mark %q, want %q", c.highlight, got.Mark, c.mark)
		}
		if got.Text != "a gap" {
			t.Errorf("%s marks %q, want the emoji dropped from the words", c.highlight, got.Text)
		}
	}
}

// TestOnlyTheFirstEmojiOfAHighlightIsItsColour is the other half of the round
// trip: export writes the colour out where the marked words themselves open
// with an emoji, so "==🟡 🟢 go==" is one yellow mark over the words "🟢 go".
func TestOnlyTheFirstEmojiOfAHighlightIsItsColour(t *testing.T) {
	runs := firstBlockRuns(t, "A sentence with ==🟡 🟢 go== in it.\n")

	got := runCarrying(t, runs, "go")
	if got.Mark != "yellow" {
		t.Errorf("==🟡 🟢 go== is mark %q, want yellow", got.Mark)
	}
	if got.Text != "🟢 go" {
		t.Errorf("==🟡 🟢 go== marks %q, want the author's own emoji kept", got.Text)
	}
}

// TestAMarkInATableCellIsCarried. A cell's paragraph walks its inlines through
// the same path a paragraph's does, and the kitchen sink's own table holds a
// ==to fill in== cell.
func TestAMarkInATableCellIsCarried(t *testing.T) {
	out := walk(t, "| Field | Value |\n|---|---|\n| Owner | ==to fill in== |\n")

	if got := out.Counts.Marks["yellow"]; got != 1 {
		t.Errorf("the cell's mark was counted %d times, want 1", got)
	}
	got := serialise(t, out.Blocks)
	if !strings.Contains(got, ">to fill in<") {
		t.Errorf("the marked words are missing from the cell:\n%s", got)
	}
	if strings.Contains(got, "==") {
		t.Errorf("the delimiters reached the cell:\n%s", got)
	}
	if !strings.Contains(got, `<w:shd w:val="clear" w:color="auto" w:fill="ffd500"/>`) {
		t.Errorf("the cell's mark carries no colour:\n%s", got)
	}
}

// TestAnEmojiElsewhereInAHighlightIsText. One position carries the colour, the
// start, because a mark whose emoji could be anywhere would eat an emoji
// somebody meant to print.
func TestAnEmojiElsewhereInAHighlightIsText(t *testing.T) {
	runs := firstBlockRuns(t, "A sentence with ==a 🔴 gap== in it.\n")

	got := runCarrying(t, runs, "gap")
	if got.Mark != "yellow" {
		t.Errorf("the mark is %q, want yellow: the emoji is not at the start", got.Mark)
	}
	if !strings.Contains(got.Text, "🔴") {
		t.Errorf("the marked words are %q, want the emoji kept as words", got.Text)
	}
}

// TestAnEmojiOutsideAHighlightIsText: a coloured circle in prose is a coloured
// circle.
func TestAnEmojiOutsideAHighlightIsText(t *testing.T) {
	runs := firstBlockRuns(t, "The light turned 🔴 and we stopped.\n")

	got := runCarrying(t, runs, "🔴")
	if got.Mark != "" {
		t.Errorf("the run carries mark %q, want none", got.Mark)
	}
}

// TestAPencilSpanIsARunCarryingItsMark is the one Pandoc span this package
// reads: content in brackets, then the attribute. It is text colour rather
// than shading, which is why the syntax is not a highlight's.
func TestAPencilSpanIsARunCarryingItsMark(t *testing.T) {
	runs := firstBlockRuns(t, "A sentence with [a note to myself]{.pencil} in it.\n")

	got := runCarrying(t, runs, "a note to myself")
	if got.Mark != "pencil" {
		t.Errorf("the span is mark %q, want pencil", got.Mark)
	}
	if text := joined(runs); strings.Contains(text, "[") || strings.Contains(text, "{") {
		t.Errorf("the prose reads %q, want the syntax out of the words", text)
	}
}

// TestThePencilSyntaxIsTheHouseStyles. The spellings the parser matches are
// constants in this package, because the parsers are built before the house
// style is read; the house file states the same strings as the pencil mark's
// syntax, and the two agreeing is the whole of what makes those constants
// safe. The list is written out here rather than read off the file, so a
// spelling somebody adds is a spelling somebody states.
func TestThePencilSyntaxIsTheHouseStyles(t *testing.T) {
	mark, ok := config(t).Marks["pencil"]
	if !ok {
		t.Fatal("the house style states no pencil mark")
	}
	want := []string{".pencil", "<!--", "%%"}
	if !slices.Equal(mark.Syntax, want) {
		t.Errorf("the house style's pencil syntax is %q, want %q", mark.Syntax, want)
	}
	for _, syntax := range []string{pencilAttribute, commentOpener} {
		if !slices.Contains(mark.Syntax, syntax) {
			t.Errorf("the parser matches %q and the house style's pencil syntax %q does not state it",
				syntax, mark.Syntax)
		}
	}
}

// TestALinkIsStillALink. The span parser triggers on the same bracket a link
// opens with and runs before the link parser, so a link is where it would
// break first.
func TestALinkIsStillALink(t *testing.T) {
	runs := firstBlockRuns(t, "see [the handbook](https://example.com/h) for more\n")

	got := runCarrying(t, runs, "the handbook")
	if got.Link != "https://example.com/h" {
		t.Errorf("link = %q, want the destination to survive", got.Link)
	}
	if got.Mark != "" {
		t.Errorf("the link's words carry mark %q, want none", got.Mark)
	}
}

// TestABracketNotFollowedByAttributesIsText. Pandoc's grammar is a bracketed
// content immediately followed by an attribute list; everything else a bracket
// opens is the author's own words.
func TestABracketNotFollowedByAttributesIsText(t *testing.T) {
	runs := firstBlockRuns(t, "An [aside] in brackets, and [another] one.\n")

	text := joined(runs)
	if !strings.Contains(text, "[aside]") || !strings.Contains(text, "[another]") {
		t.Errorf("the prose reads %q, want both brackets kept as words", text)
	}
	for _, run := range runs {
		if run.Mark != "" {
			t.Errorf("run %q carries mark %q, want none", run.Text, run.Mark)
		}
	}
}

// TestAMarkMayCarryEmphasisAndCode. The mark goes down the inline tree the way
// bold does, so the words inside one keep their own formatting.
func TestAMarkMayCarryEmphasisAndCode(t *testing.T) {
	runs := firstBlockRuns(t,
		"A ==🔵 gap with **weight** and `code`== and [a **bold** note]{.pencil}.\n")

	bold := runCarrying(t, runs, "weight")
	if bold.Mark != "blue" || !bold.Bold {
		t.Errorf("the bold words inside the highlight are %#v, want a blue mark and bold", bold)
	}
	code := runCarrying(t, runs, "code")
	if code.Mark != "blue" || !code.Mono {
		t.Errorf("the code inside the highlight is %#v, want a blue mark and the mono face", code)
	}
	note := runCarrying(t, runs, "bold")
	if note.Mark != "pencil" || !note.Bold {
		t.Errorf("the bold words inside the span are %#v, want a pencil mark and bold", note)
	}
}

// TestTwoMarksSideBySideStayTwoRuns. merge joins neighbours that carry the
// same formatting, and the mark is now part of that: joined, a pencil note and
// a yellow gap would reach the document as one colour.
func TestTwoMarksSideBySideStayTwoRuns(t *testing.T) {
	runs := firstBlockRuns(t, "[pencilled]{.pencil}==marked==\n")

	if len(runs) != 2 {
		t.Fatalf("two marks side by side are %d runs: %#v", len(runs), runs)
	}
	if runs[0].Mark != "pencil" || runs[1].Mark != "yellow" {
		t.Errorf("the runs are %q and %q, want pencil then yellow", runs[0].Mark, runs[1].Mark)
	}
}

// TestTheDebtsAreCountedAndTheOthersAreNot. Three of the seven say a human
// still owes the document something, and those three reach the envelope as
// facts for the skill to judge. A zero is counted too: a note the author
// filled in is not the same fact as a note nobody looked at.
func TestTheDebtsAreCountedAndTheOthersAreNot(t *testing.T) {
	out := walk(t, "A ==gap== and ==another gap==, a ==🟣 question==, a ==🔵 note==,\n"+
		"a ==🔴 warning==, a ==🟠 aside== and [a pencilled line]{.pencil}.\n")

	want := map[string]int{"yellow": 2, "green": 0, "purple": 1}
	if len(out.Counts.Marks) != len(want) {
		t.Fatalf("the counts are %v, want the three debts and no other", out.Counts.Marks)
	}
	for name, count := range want {
		if got := out.Counts.Marks[name]; got != count {
			t.Errorf("marks[%s] = %d, want %d", name, got, count)
		}
	}
}

// TestANestedMarkIsRefusedByLine. A mark inside a mark has two colours on one
// stretch of words and Word has one, so the author is asked which they meant
// rather than being given whichever the walk wrote last.
func TestANestedMarkIsRefusedByLine(t *testing.T) {
	for _, note := range []string{
		"Words.\n\nA ==gap with [a note]{.pencil} inside it== here.\n",
		"Words.\n\nA [note with ==a gap== inside it]{.pencil} here.\n",
	} {
		got := refused(t, note)
		if !strings.Contains(got, "line 3:") {
			t.Errorf("the refusal is %q, want it to name line 3", got)
		}
		if !strings.Contains(got, "mark inside") {
			t.Errorf("the refusal is %q, and it does not say a mark is inside a mark", got)
		}
	}
}

// TestAnEmptySpanIsRefusedByLine. A span marking no words is a colour on
// nothing, which the author cannot see in the document they open.
func TestAnEmptySpanIsRefusedByLine(t *testing.T) {
	got := refused(t, "Words.\n\nAn []{.pencil} empty span.\n")

	if !strings.Contains(got, "line 3:") {
		t.Errorf("the refusal is %q, want it to name line 3", got)
	}
	if !strings.Contains(got, "no words") {
		t.Errorf("the refusal is %q, and it does not say the span marks no words", got)
	}
}

// TestASpanWithAnotherAttributeIsRefusedByLineNamingPencil. One attribute is
// read, and a span carrying another would otherwise print as the words the
// author wrote, brackets and braces included.
func TestASpanWithAnotherAttributeIsRefusedByLineNamingPencil(t *testing.T) {
	got := refused(t, "Words.\n\nA [note]{.todo} here.\n")

	if !strings.Contains(got, "line 3:") {
		t.Errorf("the refusal is %q, want it to name line 3", got)
	}
	if !strings.Contains(got, ".pencil") {
		t.Errorf("the refusal is %q, and it does not name the one attribute gdoc reads", got)
	}
}

// TestASpanCrossingAParagraphIsRefusedByLine. A span opens and closes on one
// line. One that does not is words in the document, and the words are the
// syntax.
func TestASpanCrossingAParagraphIsRefusedByLine(t *testing.T) {
	got := refused(t, "[a note that opens\n\nand closes two lines down]{.pencil}\n")

	if !strings.Contains(got, "line 3:") {
		t.Errorf("the refusal is %q, want it to name line 3", got)
	}
	if !strings.Contains(got, pencilAttribute) {
		t.Errorf("the refusal is %q, and it does not name the span it is about", got)
	}
}

// TestASpanWhoseWordsHoldACodeSpanWithABracketIsStillASpan. Nothing inside
// backticks is markdown, and both halves of the span read a line that way:
// the parser used to stop at the bracket inside the code, leave the span
// unopened and print "]{.pencil}" into the document, while checkSpans looked
// at the line with the code taken out and saw nothing wrong.
func TestASpanWhoseWordsHoldACodeSpanWithABracketIsStillASpan(t *testing.T) {
	runs := firstBlockRuns(t, "Words [see `a[0]`]{.pencil} here.\n")

	code := runCarrying(t, runs, "a[0]")
	if code.Mark != "pencil" {
		t.Errorf("the code inside the span is %#v, want a pencil mark", code)
	}
	if text := joined(runs); strings.Contains(text, "]{.pencil}") {
		t.Errorf("the prose reads %q, want the span's own syntax out of it", text)
	}
}

// TestASpanInsideACodeBlockIsNotASpan. The span scan reads the same lines the
// footnote scan reads, so an author writing the syntax out in an example is
// not refused for it. The indented block is the half that was read as prose.
func TestASpanInsideACodeBlockIsNotASpan(t *testing.T) {
	for _, markdown := range []string{
		"Words.\n\n    example [x]{.y} here\n",
		"Words.\n\n```\nexample [x]{.y} here\n```\n",
		"## Example\n    example [x]{.y} here\n",
		"Words.\n\n```\nx\n```\n    example [x]{.y} here\n",
	} {
		out, err := Render(config(t), []byte(markdown), "testdata/docs", true)
		if err != nil {
			t.Errorf("%q was refused: %v", markdown, err)
			continue
		}
		if len(out.Blocks) == 0 {
			t.Errorf("%q wrote nothing", markdown)
		}
	}
}

// TestAMarkInAHeadingIsRefusedByLine, for the reason a footnote mark in a
// heading is refused: Word copies a heading into the contents field, and the
// mark would be in the contents list as well as on the page.
func TestAMarkInAHeadingIsRefusedByLine(t *testing.T) {
	for _, note := range []string{
		"Words.\n\n## A ==marked== heading\n\nMore words.\n",
		"Words.\n\n## A [pencilled]{.pencil} heading\n\nMore words.\n",
	} {
		got := refused(t, note)
		if !strings.Contains(got, "line 3:") {
			t.Errorf("the refusal is %q, want it to name line 3", got)
		}
		if !strings.Contains(got, "heading") {
			t.Errorf("the refusal is %q, and it does not say the mark is in a heading", got)
		}
	}
}

// TestAMarkInAPicturesAltTextIsRefusedByLine. An alt text becomes the
// picture's caption, and the walk builds the caption with no palette at all,
// because a caption carries no colour. The mark was dropped there in silence:
// the colour emoji that opens a highlight reached the caption as words, and the
// envelope counted a mark the document does not carry.
func TestAMarkInAPicturesAltTextIsRefusedByLine(t *testing.T) {
	for _, note := range []string{
		"Words.\n\n![a ==\U0001F7E2 marked== alt](badge.png)\n",
		"Words.\n\n![a [pencilled]{.pencil} alt](badge.png)\n",
	} {
		got := refused(t, note)
		if !strings.Contains(got, "line 3:") {
			t.Errorf("the refusal is %q, want it to name line 3", got)
		}
		if !strings.Contains(got, "alt text") {
			t.Errorf("the refusal is %q, and it does not say the mark is in an alt text", got)
		}
	}
}

// TestAMarkedRunCarriesItsStyleAndItsShading is how a mark reaches Word: the
// run names the character style internal/render writes, and it states the
// colour itself as well, as shading for a highlight and as the run's own
// colour for the pencil. The run has to say it because Google Docs has no
// character styles: after import a mark is direct formatting, and direct
// formatting is all export can read back.
//
// Every id and every hex here is written out. The colours come from
// house.yaml, so a test that read them too would follow the palette wherever
// somebody moved it and claim nothing.
func TestAMarkedRunCarriesItsStyleAndItsShading(t *testing.T) {
	out := walk(t, "A ==a gap==, a ==🔴 warning==, a ==🟠 aside==, a ==🟢 claim==,\n"+
		"a ==🔵 change==, a ==🟣 question== and [a pencilled line]{.pencil}.\n")

	cases := []struct {
		words, style, fill, color string
	}{
		{words: "a gap", style: "Yellow", fill: "ffd500"},
		{words: "warning", style: "Red", fill: "ff6b6b"},
		{words: "aside", style: "Orange", fill: "ffb85c"},
		{words: "claim", style: "Green", fill: "3dd67a"},
		{words: "change", style: "Blue", fill: "5db3ff"},
		{words: "question", style: "Purple", fill: "ba92ff"},
		{words: "a pencilled line", style: "Pencil", color: "9a9a9a"},
	}
	for _, c := range cases {
		run := runElement(t, out.Blocks, c.words)
		rPr := run.SelectElement("w:rPr")
		if rPr == nil {
			t.Errorf("the run marking %q carries no w:rPr", c.words)
			continue
		}
		if got := value(rPr, "w:rStyle"); got != c.style {
			t.Errorf("the run marking %q names style %q, want %s", c.words, got, c.style)
		}
		shd := rPr.FindElement("w:shd")
		if c.fill == "" {
			if shd != nil {
				t.Errorf("the pencil run marking %q carries shading, want its text colour alone", c.words)
			}
			if got := value(rPr, "w:color"); got != c.color {
				t.Errorf("the run marking %q is coloured %q, want %s", c.words, got, c.color)
			}
			continue
		}
		if shd == nil {
			t.Errorf("the run marking %q carries no shading", c.words)
			continue
		}
		if got := shd.SelectAttrValue("w:val", ""); got != "clear" {
			t.Errorf("the shading on %q is pattern %q, want clear", c.words, got)
		}
		if got := shd.SelectAttrValue("w:color", ""); got != "auto" {
			t.Errorf("the shading on %q has foreground %q, want auto", c.words, got)
		}
		if got := shd.SelectAttrValue("w:fill", ""); got != c.fill {
			t.Errorf("the shading on %q is filled %q, want %s", c.words, got, c.fill)
		}
		if rPr.FindElement("w:highlight") != nil {
			t.Errorf("the run marking %q carries one of Word's sixteen named highlights, want the exact hex", c.words)
		}
	}
}

// value is one property's w:val, or the empty string when the run does not
// carry the property at all.
func value(rPr *etree.Element, tag string) string {
	e := rPr.FindElement(tag)
	if e == nil {
		return ""
	}
	return e.SelectAttrValue("w:val", "")
}

// runElement is the one run whose words are the given text, as the paragraph
// carries it. The run is where a mark's style and its colour are, which the
// Run struct cannot show.
func runElement(t *testing.T, blocks []*etree.Element, words string) *etree.Element {
	t.Helper()
	var found []*etree.Element
	for _, block := range blocks {
		for _, run := range findAll(block, "w:r") {
			text := ""
			for _, e := range run.FindElements("w:t") {
				text += e.Text()
			}
			if text == words {
				found = append(found, run)
			}
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d runs carry the words %q, want one", len(found), words)
	}
	return found[0]
}

// TestAHighlightOfOnlyItsColourEmojiIsRefusedByLine. The emoji says which
// colour and is not the words, so a highlight holding nothing else marks no
// words: withoutOpener takes the emoji off, merge drops the run that is left,
// and the author's own character is gone from the document. countMarks counts
// the node either way, so the envelope reported a colour the document does not
// carry, which is the fact the alt-text refusal exists to prevent. An empty
// span is refused for the same reason.
func TestAHighlightOfOnlyItsColourEmojiIsRefusedByLine(t *testing.T) {
	for _, note := range []string{
		"Words.\n\nA line with ==🟢== in it.\n",
		"Words.\n\nA line with ==🟡 go== and ==🟣== in it.\n",
	} {
		got := refused(t, note)
		if !strings.Contains(got, "line 3:") {
			t.Errorf("the refusal is %q, want it to name line 3", got)
		}
		if !strings.Contains(got, "no words") {
			t.Errorf("the refusal is %q, and it does not say the mark holds no words", got)
		}
	}
}

// The other side of it: a highlight whose colour emoji is followed by words is
// the mark the author meant, and one whose words hold an emoji of their own
// further in is words.
func TestAHighlightWithWordsAfterItsColourIsNotRefused(t *testing.T) {
	for _, note := range []string{
		"Words.\n\nA line with ==🟢 go== in it.\n",
		"Words.\n\nA line with ==🟢 **go**== in it.\n",
		"Words.\n\nA line with ==go 🟢== in it.\n",
	} {
		runs := firstBlockRuns(t, strings.TrimPrefix(note, "Words.\n\n"))
		if len(runs) == 0 {
			t.Errorf("%q rendered no runs", note)
		}
	}
}

// TestAPencilAttributeThatOpensNoSpanIsRefusedByLine: Pandoc's span is a
// bracket gdoc's own parser claims, and three of goldmark's parsers claim one
// first. A bracket after "!" is a picture's, a bracket holding a caret is a
// footnote mark's, and a bracket followed by a target or a reference is a
// link's. In each of those the attribute closes nothing, so "{.pencil}" is
// published as words. The line check read the shape and called it a span.
func TestAPencilAttributeThatOpensNoSpanIsRefusedByLine(t *testing.T) {
	for _, markdown := range []string{
		"Words.\n\nA picture ![a]{.pencil} here.\n",
		"Words.[^a]{.pencil}\n\n[^a]: the note.\n",
		"Words.\n\nA link [words](https://example.com){.pencil} here.\n",
		"Words.\n\nA reference [words][ref]{.pencil} here.\n\n[ref]: https://example.com\n",
		"Words.\n\nA bare {.pencil} in the prose.\n",
	} {
		got := refused(t, markdown)

		if !strings.Contains(got, pencilAttribute) {
			t.Errorf("%q: the refusal is %q, and it does not name the attribute", markdown, got)
		}
		if !strings.Contains(got, "words") {
			t.Errorf("%q: the refusal is %q, and it does not say the attribute would reach the document as words", markdown, got)
		}
	}
}

// TestAPencilAttributeInsideABlockOfHTMLRefusesNothing: a block of HTML is
// named on the envelope and left out, so the syntax written inside it reaches
// no page and refuses nothing, which is the rule a code block already has.
func TestAPencilAttributeInsideABlockOfHTMLRefusesNothing(t *testing.T) {
	out, err := Render(config(t), []byte("Words.\n\n<div>\n[a]{.pencil}\n</div>\n"), "testdata/docs", true)
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}
	if len(out.Blocks) == 0 {
		t.Error("the file wrote nothing")
	}
}

// TestASpanTheParseFormsIsNotRefused guards the check above from refusing the
// spans that do work: one of its own, two on a line, one inside emphasis and
// one inside a table cell.
func TestASpanTheParseFormsIsNotRefused(t *testing.T) {
	for _, markdown := range []string{
		"A [note]{.pencil} here.\n",
		"A [note]{.pencil} and [another]{.pencil} here.\n",
		"A *[note]{.pencil}* here.\n",
		"| a | b |\n| --- | --- |\n| [note]{.pencil} | x |\n",
	} {
		if _, err := Render(config(t), []byte(markdown), "testdata/docs", true); err != nil {
			t.Errorf("%q was refused: %v", markdown, err)
		}
	}
}

// TestASpanWhoseWordsAreOnlyACodeSpanIsStillASpan. The line scan reads the
// line with its code taken out, so a span whose whole content is a code span
// read as "[]{.pencil}" and was refused for marking no words, naming a span
// nowhere in the file. The parser forms that span: spanAhead steps over the
// code, and the words reach the document grey.
//
// A document exported from Drive is where it showed: a pencil run whose text
// is one piece of code comes home as the span, and the note would not build.
func TestASpanWhoseWordsAreOnlyACodeSpanIsStillASpan(t *testing.T) {
	source := "Words [`code`]{.pencil} here.\n"
	if _, err := Render(config(t), []byte(source), "testdata/docs", true); err != nil {
		t.Fatalf("the span was refused: %v", err)
	}

	runs := firstBlockRuns(t, source)

	code := runCarrying(t, runs, "code")
	if code.Mark != "pencil" {
		t.Errorf("the code inside the span is %#v, want a pencil mark", code)
	}
	if text := joined(runs); strings.Contains(text, "]{.pencil}") {
		t.Errorf("the prose reads %q, want the span's own syntax out of it", text)
	}
}

// The other half: an empty span on a line that also holds a code span is still
// refused. The words are empty because the author wrote no words, not because
// the scan took them out, and nothing may reach the document as a colour on
// nothing.
func TestAnEmptySpanBesideACodeSpanIsStillRefused(t *testing.T) {
	_, err := Render(config(t), []byte("Words `x` and []{.pencil} here.\n"), "testdata/docs", true)
	if err == nil {
		t.Fatal("the empty span was not refused")
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Errorf("the refusal is %q, want it to name line 1", err)
	}
}
