package body

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark/text"
)

func firstBlockRuns(t *testing.T, source string) []Run {
	t.Helper()
	bytes := []byte(source)
	root := parse().Parser().Parse(text.NewReader(bytes))
	block := root.FirstChild()
	if block == nil {
		t.Fatal("no block parsed")
	}
	return inlineRuns(block, bytes)
}

func joined(runs []Run) string {
	var b strings.Builder
	for _, run := range runs {
		b.WriteString(run.Text)
	}
	return b.String()
}

func runCarrying(t *testing.T, runs []Run, text string) Run {
	t.Helper()
	for _, run := range runs {
		if strings.Contains(run.Text, text) {
			return run
		}
	}
	t.Fatalf("no run carrying %q in %#v", text, runs)
	return Run{}
}

func TestMarksAreCarriedOntoTheRunsTheyCover(t *testing.T) {
	runs := firstBlockRuns(t, "plain **bold** *italic* `code` ==marked==\n")

	if got := runCarrying(t, runs, "bold"); !got.Bold || got.Italic {
		t.Errorf("bold run = %#v", got)
	}
	if got := runCarrying(t, runs, "italic"); !got.Italic || got.Bold {
		t.Errorf("italic run = %#v", got)
	}
	if got := runCarrying(t, runs, "code"); !got.Mono {
		t.Errorf("code run = %#v", got)
	}
	if got := runCarrying(t, runs, "marked"); got.Highlight != "yellow" {
		t.Errorf("marked run = %#v, want a yellow highlight", got)
	}
}

func TestMarksNest(t *testing.T) {
	runs := firstBlockRuns(t, "**bold with *italic* inside**\n")

	got := runCarrying(t, runs, "italic")
	if !got.Bold || !got.Italic {
		t.Errorf("nested run = %#v, want both marks", got)
	}
}

func TestALinkKeepsItsTextAndItsDestination(t *testing.T) {
	runs := firstBlockRuns(t, "see [the handbook](https://example.com/h) for more\n")

	got := runCarrying(t, runs, "the handbook")
	if got.Link != "https://example.com/h" {
		t.Errorf("link = %q, want the destination to survive", got.Link)
	}
	if !strings.Contains(joined(runs), "the handbook") {
		t.Errorf("text = %q, want the link text in the prose", joined(runs))
	}
}

func TestEntitiesAreDecodedRatherThanCarriedAsText(t *testing.T) {
	// goldmark resolves these in its HTML renderer, where they are already
	// correct. Written into a w:t they are not, and the XML escape doubles them.
	runs := firstBlockRuns(t, "AT&amp;T and &#8212; a dash\n")

	text := joined(runs)
	if !strings.Contains(text, "AT&T") {
		t.Errorf("text = %q, want a real ampersand", text)
	}
	if strings.Contains(text, "&amp;") || strings.Contains(text, "&#8212;") {
		t.Errorf("text = %q, still holds an entity reference", text)
	}
}

func TestSmartPunctuationIsCharactersNotEntities(t *testing.T) {
	runs := firstBlockRuns(t, `"quoted", an em dash --- and an ellipsis...`+"\n")

	text := joined(runs)
	for _, want := range []string{"“", "”", "—", "…"} {
		if !strings.Contains(text, want) {
			t.Errorf("text = %q, want it to carry %q", text, want)
		}
	}
	if strings.Contains(text, "&ldquo;") {
		t.Errorf("text = %q, the typographer is still emitting entities", text)
	}
}

func TestAdjacentRunsSharingTheirMarksAreJoined(t *testing.T) {
	// goldmark splits at every syntactic boundary. One w:r per word bloats the
	// document without changing a pixel.
	runs := firstBlockRuns(t, "four plain words here\n")

	if len(runs) != 1 {
		t.Errorf("runs = %d, want them merged into 1: %#v", len(runs), runs)
	}
}

func TestAPictureIsFoundHoweverDeeplyItIsWrapped(t *testing.T) {
	// Drive writes a picture on its own line as a bold heading, so the image
	// sits inside a Strong node. A top-level search finds nothing and drops it.
	source := []byte("# **![alt text](diagram.png)**\n")
	root := parse().Parser().Parse(text.NewReader(source))

	images := collectImages(root.FirstChild(), source)

	if len(images) != 1 {
		t.Fatalf("found %d images, want 1 inside the Strong node", len(images))
	}
	if images[0].Target != "diagram.png" {
		t.Errorf("target = %q", images[0].Target)
	}
	if joined(images[0].Alt) != "alt text" {
		t.Errorf("alt = %q, want it kept for the caption", joined(images[0].Alt))
	}
}

func TestAPictureIsNotRenderedAsInlineText(t *testing.T) {
	// It is lifted out and captioned instead, so its alt must not also appear
	// in the paragraph it came from.
	runs := firstBlockRuns(t, "words ![alt text](diagram.png) more words\n")

	if strings.Contains(joined(runs), "alt text") {
		t.Errorf("runs = %q, want the alt text left for the caption", joined(runs))
	}
}

func TestTaskListsCarryTheirBallotCharacter(t *testing.T) {
	source := []byte("- [ ] not done\n- [x] done\n")
	root := parse().Parser().Parse(text.NewReader(source))
	list := root.FirstChild()

	var texts []string
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		texts = append(texts, joined(inlineRuns(item, source)))
	}

	if len(texts) != 2 {
		t.Fatalf("items = %d, want 2", len(texts))
	}
	if !strings.HasPrefix(texts[0], "☐") {
		t.Errorf("unchecked item = %q, want it to open with an empty ballot box", texts[0])
	}
	if !strings.HasPrefix(texts[1], "☒") {
		t.Errorf("checked item = %q, want it to open with a crossed ballot box", texts[1])
	}
}

func TestStrikethroughKeepsItsTextAndItsMark(t *testing.T) {
	runs := firstBlockRuns(t, "text with ~~a struck-out span~~ inside\n")

	if !strings.Contains(joined(runs), "a struck-out span") {
		t.Errorf("runs = %q, want the struck text kept", joined(runs))
	}
	if got := runCarrying(t, runs, "a struck-out span"); !got.Strike {
		t.Errorf("struck run = %#v, want the strike carried onto it", got)
	}
}

// An email autolink carries the mailto: scheme, because goldmark puts it on in
// its HTML renderer and never in URL(). Written into a relationship bare, the
// address is a relative URI reference: Word resolves it against the document's
// own location and the link opens nothing. The label stays bare.
func TestAnEmailAutolinkCarriesTheMailtoScheme(t *testing.T) {
	for _, markdown := range []string{
		"write to <nail@altery.com> if unsure\n",
		"write to nail@altery.com if unsure\n",
	} {
		runs := firstBlockRuns(t, markdown)

		got := runCarrying(t, runs, "nail@altery.com")
		if got.Link != "mailto:nail@altery.com" {
			t.Errorf("%q: link = %q, want mailto:nail@altery.com", markdown, got.Link)
		}
		if got.Text != "nail@altery.com" {
			t.Errorf("%q: text = %q, want the bare address", markdown, got.Text)
		}
	}
}

// A web autolink is left exactly as it is: the scheme is already in it, and
// prefixing one would break every link in a note.
func TestAWebAutolinkKeepsItsOwnScheme(t *testing.T) {
	runs := firstBlockRuns(t, "see https://example.com/h for more\n")

	got := runCarrying(t, runs, "https://example.com/h")
	if got.Link != "https://example.com/h" {
		t.Errorf("link = %q, want the destination untouched", got.Link)
	}
}
