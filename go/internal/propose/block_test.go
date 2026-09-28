package propose

import (
	"strings"
	"testing"
)

// The content of the block the guard's own TestABlockProposalNeedsNoGrant
// sends, so the two files describe one write.
const testContent = "## 3.6 Limits\n\nBody text with **bold**.\n\n- one\n- two\n"

// samePara says whether a paragraph is the one wanted, style, list kind and
// runs together. A test that compared only the text would pass over a heading
// read as body text.
func samePara(got, want Para) bool {
	if got.Style != want.Style || got.List != want.List || len(got.Runs) != len(want.Runs) {
		return false
	}
	for i := range got.Runs {
		if got.Runs[i] != want.Runs[i] {
			return false
		}
	}
	return true
}

func checkParas(t *testing.T, content string, want []Para) {
	t.Helper()
	got, err := ParseContent(content)
	if err != nil {
		t.Fatalf("ParseContent(%q) = %v, want the paragraphs", content, err)
	}
	if len(got) != len(want) {
		t.Fatalf("ParseContent(%q) gave %d paragraphs, want %d: %+v", content, len(got), len(want), got)
	}
	for i := range want {
		if !samePara(got[i], want[i]) {
			t.Errorf("paragraph %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestContentBecomesTheParagraphsTheBlockWrites is the subset in one string: a
// heading at its own level, a paragraph carrying a mark, and a bulleted list.
// Each new paragraph carries the named style it takes in the document, the list
// kind it belongs to, and its runs.
func TestContentBecomesTheParagraphsTheBlockWrites(t *testing.T) {
	checkParas(t, testContent, []Para{
		{Style: "HEADING_2", Runs: []Run{{Text: "3.6 Limits"}}},
		{Style: NormalStyle, Runs: []Run{
			{Text: "Body text with "}, {Text: "bold", Bold: true}, {Text: "."},
		}},
		{Style: NormalStyle, List: Bulleted, Runs: []Run{{Text: "one"}}},
		{Style: NormalStyle, List: Bulleted, Runs: []Run{{Text: "two"}}},
	})
}

// A heading takes the level the author wrote, one to six, and the Docs named
// style is that number. Nothing here renumbers a heading the way publish does:
// the block lands beside the document's own headings, and the level the review
// asked for is the level it gets.
func TestContentReadsAHeadingAtItsOwnLevel(t *testing.T) {
	for level, hashes := range map[int]string{1: "#", 2: "##", 3: "###", 4: "####", 5: "#####", 6: "######"} {
		want := "HEADING_" + string(rune('0'+level))
		checkParas(t, hashes+" Limits\n", []Para{
			{Style: want, Runs: []Run{{Text: "Limits"}}},
		})
	}
}

// The three marks a block carries: bold, italic and a link. Nested emphasis
// carries both marks on the one run, and a run keeps its marks across the words
// beside it.
func TestContentReadsTheMarksASentenceCarries(t *testing.T) {
	checkParas(t, "Plain **bold** and *italic* and ***both*** and [words](https://example.com/p).\n", []Para{
		{Style: NormalStyle, Runs: []Run{
			{Text: "Plain "},
			{Text: "bold", Bold: true},
			{Text: " and "},
			{Text: "italic", Italic: true},
			{Text: " and "},
			{Text: "both", Bold: true, Italic: true},
			{Text: " and "},
			{Text: "words", Link: "https://example.com/p"},
			{Text: "."},
		}},
	})
}

// A numbered list is its own list kind, because the two take different bullet
// presets in the batch Task 5 builds.
func TestContentReadsBothListKinds(t *testing.T) {
	checkParas(t, "1. first\n2. second\n", []Para{
		{Style: NormalStyle, List: Numbered, Runs: []Run{{Text: "first"}}},
		{Style: NormalStyle, List: Numbered, Runs: []Run{{Text: "second"}}},
	})
	checkParas(t, "- first\n- second\n", []Para{
		{Style: NormalStyle, List: Bulleted, Runs: []Run{{Text: "first"}}},
		{Style: NormalStyle, List: Bulleted, Runs: []Run{{Text: "second"}}},
	})
}

// A line the author wrapped is one paragraph, and the wrap is a space. A
// paragraph is a paragraph because of the empty line, never because of where
// the words ran out.
func TestContentJoinsAWrappedLineWithASpace(t *testing.T) {
	checkParas(t, "one sentence\nwrapped across lines\n", []Para{
		{Style: NormalStyle, Runs: []Run{{Text: "one sentence wrapped across lines"}}},
	})
}

// Text is what the paragraph reads as once its marks are dropped, which is what
// the insert carries.
func TestParaTextIsTheWordsWithoutTheMarks(t *testing.T) {
	paras, err := ParseContent(testContent)
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	if got := paras[1].Text(); got != "Body text with bold." {
		t.Errorf("Text() = %q, want the words with the marks dropped", got)
	}
}

// TestContentRefusesWhatTheSubsetDoesNotHold is the other half of the subset,
// and each case is refused by name rather than dropped: a file gdoc half
// understands never reaches a document, and a block silently missing its table
// is a suggestion nobody can read.
func TestContentRefusesWhatTheSubsetDoesNotHold(t *testing.T) {
	cases := []struct {
		name    string
		content string
		says    string
	}{
		{"a table", "| a | b |\n| --- | --- |\n| 1 | 2 |\n", "table"},
		{"a nested bulleted list", "- one\n    - deeper\n", "nested list"},
		{"a nested list under a number", "1. one\n    1. deeper\n", "nested list"},
		{"a picture", "Text ![alt](pic.png) more\n", "picture"},
		{"a fenced code block", "```\ncode\n```\n", "code block"},
		{"an indented code block", "one\n\n    code\n", "code block"},
		{"a block quote", "> quoted\n", "block quote"},
		{"a block of HTML", "<div>x</div>\n", "HTML"},
		{"HTML in a sentence", "Text <b>x</b> more\n", "HTML"},
		{"a horizontal rule", "one\n\n---\n\ntwo\n", "horizontal rule"},
		{"no content at all", "", "no content"},
		{"nothing but space", "   \n\n\t\n", "no content"},
		{"code in a sentence", "Use `gdoc read` first\n", "code in a sentence"},
		{"struck-out words", "~~gone~~ words\n", "struck"},
		{"a task list", "- [ ] do it\n", "task list"},
		{"a line break inside a paragraph", "one  \ntwo\n", "line break"},
		{"a heading with no words", "##\n", "no text"},
		{"a list item with no words", "- one\n-\n", "no text"},
		{"a list item with two paragraphs", "- one\n\n  second\n", "more than one paragraph"},
		{"a heading inside a list item", "- # heading\n", "heading"},
		{"a link that opens nothing", "See [the note](../notes/limits.md)\n", "address"},
		{"a link to a heading", "See [above](#limits)\n", "address"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			paras, err := ParseContent(c.content)
			if err == nil {
				t.Fatalf("ParseContent(%q) = %+v, want a refusal naming %s", c.content, paras, c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("ParseContent(%q) says %q, want it to name %q", c.content, err, c.says)
			}
		})
	}
}

// A refusal names the line the author can look at, because content is a file a
// person wrote and a block is tens of lines long.
func TestARefusedConstructNamesItsLine(t *testing.T) {
	_, err := ParseContent("one\n\ntwo\n\n> quoted\n")
	if err == nil {
		t.Fatal("a block quote is refused")
	}
	if !strings.Contains(err.Error(), "line 5") {
		t.Errorf("the refusal says %q, want it to name line 5", err)
	}
}

// A horizontal rule is the construct goldmark builds with no source position at
// all, so its line is the first line holding anything after the paragraph above
// it. Line 1 would send the author to the top of a file whose rule is further
// down, which is the wrong end of it.
func TestARuleWithNoSourcePositionStillNamesItsLine(t *testing.T) {
	_, err := ParseContent("one\n\n---\n\ntwo\n")
	if err == nil {
		t.Fatal("a horizontal rule is refused")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("the refusal says %q, want it to name line 3", err)
	}
}

// An address a document can open stays a link, whatever the scheme, and an
// email autolink gets the scheme goldmark leaves off.
func TestContentKeepsAnAddressADocumentCanOpen(t *testing.T) {
	checkParas(t, "Write to <ops@example.com> about it\n", []Para{
		{Style: NormalStyle, Runs: []Run{
			{Text: "Write to "},
			{Text: "ops@example.com", Link: "mailto:ops@example.com"},
			{Text: " about it"},
		}},
	})
	checkParas(t, "See [the doc](https://docs.google.com/document/d/ABC/edit)\n", []Para{
		{Style: NormalStyle, Runs: []Run{
			{Text: "See "},
			{Text: "the doc", Link: "https://docs.google.com/document/d/ABC/edit"},
		}},
	})
}

// TestCheckRefusesABlockFieldWithoutTheBlockKind is the mistake the decoder
// cannot catch. Both kinds are read into the one type, so a caller that wrote a
// block and forgot its kind hands over an entry whose block fields are fields
// the decoder knows.
//
// Reading it as the words kind is the wrong answer twice. An entry with no quote
// is refused for quoting no text, which names nothing the author did wrong, and
// an entry that quotes words as well would be sent as a words proposal with its
// content silently dropped. So Check names the field it saw and the kind that
// field belongs to.
func TestCheckRefusesABlockFieldWithoutTheBlockKind(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    Proposal
		says string
	}{
		{"after and no kind", Proposal{After: "reviewed annually", Content: testContent, Why: testWhy}, "after"},
		{"replace_from and no kind",
			Proposal{ReplaceFrom: "reviewed annually", ReplaceTo: "risk matrix", Content: testContent, Why: testWhy},
			"replace_from"},
		{"replace_to beside a words proposal",
			Proposal{Quoted: "reviewed annually", Replacement: "reviewed twice a year", ReplaceTo: "risk matrix", Why: testWhy},
			"replace_to"},
		{"content beside a words proposal",
			Proposal{Quoted: "reviewed annually", Replacement: "reviewed twice a year", Content: testContent, Why: testWhy},
			"content"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.p.Check()
			if err == nil {
				t.Fatal("a block field on an entry naming no kind must be refused")
			}
			if !strings.Contains(err.Error(), tc.says) {
				t.Errorf("the refusal must name the field %q: %v", tc.says, err)
			}
			if !strings.Contains(err.Error(), `"block"`) {
				t.Errorf("the refusal must name the kind the field belongs to: %v", err)
			}
		})
	}
}
