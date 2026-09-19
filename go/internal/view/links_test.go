// The four facts M13 taught the projection to print: a run's link target, a
// numbered list's numbers, a floating object's placeholder, and the escaping
// that now holds across a run boundary and against gdoc's own markers. The
// goldens for the first three are in testdata beside the five older ones, and
// TestGolden reads all eight.

package view

import (
	"encoding/json"
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// project is one paragraph of runs through the whole projection, with the
// trailing newline of the text taken off. It builds the document rather than
// parsing one, because what these tests are about is the runs: a run pair that
// splits a marker is three lines here and a fixture file otherwise.
func project(runs ...docs.Run) string {
	d := &docs.Document{Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{
		{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", Runs: runs}},
	}}}}
	text, _ := Text(d)
	return strings.TrimSuffix(text, "\n")
}

// run is a text run at the indexes it occupies.
func run(s string, start int) docs.Run {
	return docs.Run{Kind: docs.KindText, Text: s, StartIndex: start, EndIndex: start + len([]rune(s))}
}

func TestALinkPrintsItsTarget(t *testing.T) {
	got, warnings := Text(fixture(t, "links.json"))
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
	for _, want := range []string{
		// A link out of the document keeps its address.
		"[guide](https://example.com/guide)",
		// A link to a heading is that heading's own words, slugged, which is
		// the form a note's own links take.
		"[the section](#the-section)",
		// A bookmark has no words, so it is named by its id.
		"[a bookmark](#id.bookmarkone)",
		"[the nested heading](#h.nested)",
		"[the bookmark](#id.nested)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Text() = %q, want it to carry %s", got, want)
		}
	}
	// A link naming only a tab has no target this projection can write, so the
	// words are printed alone rather than pointing at a guess.
	if strings.Contains(got, "[the plan]") {
		t.Errorf("a tab link was printed as a link: %q", got)
	}
	if !strings.Contains(got, "and the plan") {
		t.Errorf("Text() = %q, want the tab link's words printed alone", got)
	}
}

// The structure view carries the target too, as the link field docs decodes, so
// a caller that wants the id rather than the slug reads it there.
func TestTheStructureCarriesTheLinkField(t *testing.T) {
	raw, err := json.Marshal(Structure(fixture(t, "links.json")))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"link":{"url":"https://example.com/guide"}`, `"heading_id":"h.thesection"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("Structure() does not carry %s: %s", want, raw)
		}
	}
}

func TestANumberedListIsNumbered(t *testing.T) {
	got, _ := Text(fixture(t, "lists.json"))
	for _, want := range []string{
		"1. First step",
		// A nested numbered item is indented by three spaces per level, which
		// is the width of the number it sits under.
		"   1. A substep",
		"1. Boil the water",
		"2. Add the leaves",
		// A second list starts at one again, which is what the list id is read
		// for: two adjacent numbered lists are one long list without it.
		"1. Fill in the form",
		// A list Docs says is not numbered is a bullet, as it was before.
		"- A bullet",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Text() =\n%s\nwant it to carry %q", got, want)
		}
	}
	// The same list id in two tabs is two lists, because the lists map is the
	// tab's own: the second tab's item is a bullet and takes no number.
	if strings.Contains(got, "1. A bullet") {
		t.Errorf("the second tab's bullet took the first tab's numbering:\n%s", got)
	}
}

// A heading that is also a list item is a heading, and it takes no number: two
// markers on one paragraph would say it is two things.
func TestAHeadingInAListIsNotNumbered(t *testing.T) {
	d := &docs.Document{Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{
		{Paragraph: &docs.Paragraph{
			Style:  "HEADING_2",
			Bullet: &docs.Bullet{ListID: "kix.one", Ordered: true, Glyph: "DECIMAL"},
			Runs:   []docs.Run{run("Scope\n", 1)},
		}},
	}}}}
	got, _ := Text(d)
	if got != "## Scope\n" {
		t.Errorf("Text() = %q, want the heading alone", got)
	}
}

func TestAFloatingObjectIsAPlaceholderAndAWarning(t *testing.T) {
	got, warnings := Text(fixture(t, "positioned.json"))
	// The placeholder names the kind, so a floating drawing does not read as a
	// picture, and the object id, which is what pairs it with its bytes.
	if !strings.Contains(got, "<!-- image: floating, kix.posone -->") {
		t.Errorf("Text() =\n%s\nwant the floating picture's placeholder", got)
	}
	if !strings.Contains(got, "<!-- drawing: floating, kix.postwo -->") {
		t.Errorf("Text() =\n%s\nwant the floating drawing's placeholder", got)
	}
	// It goes after the paragraph it is anchored to, because that is the only
	// position the answer gives: a floating object sits beside the text rather
	// than in it, so it has no character index of its own.
	if at, para := strings.Index(got, "<!-- image"), strings.Index(got, "The chart sits beside this."); at < para {
		t.Errorf("the placeholder came before its paragraph:\n%s", got)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %v, want one per floating object", warnings)
	}
	if !strings.Contains(warnings[0], "kix.posone") || !strings.Contains(warnings[0], "its content is not read") {
		t.Errorf("warnings[0] = %q", warnings[0])
	}
}

// The link form makes one bracket markup where it was not before: a reader takes
// the words up to the first "]" it meets, so a "]" the document holds inside the
// words would cut the link short and leave the target standing as prose.
func TestABracketInsideALinksWordsIsEscaped(t *testing.T) {
	linked := docs.Run{Kind: docs.KindText, Text: "Q3 [draft] plan", StartIndex: 1, EndIndex: 16,
		Link: &docs.Link{URL: "https://example.com/q3"}}
	got := project(linked)
	if want := `[Q3 \[draft\] plan](https://example.com/q3)`; got != want {
		t.Errorf("Text() = %q, want %q", got, want)
	}
	if unescape(got) != "Q3 [draft] plan" {
		t.Errorf("unescape(%q) = %q, want the words and neither bracket", got, unescape(got))
	}
	// Outside a link the same brackets are the document's own text and stay as
	// they are: nothing reads one bracket as markup there.
	if plain := project(run("Q3 [draft] plan\n", 1)); plain != "Q3 [draft] plan" {
		t.Errorf("outside a link the text came back as %q", plain)
	}
}

// TestEscapingHoldsAcrossTwoRuns is the other half of the invariant
// TestEscapingLeavesNoMarkerBehindWhenTwoOverlap holds inside one run: a marker
// in the output is always gdoc's own. Docs splits a run at every formatting
// change, so a paragraph where the first "[" is bold and the second is not
// arrives as two runs, and the pair is only visible to a projection that holds
// the first character back until it knows what follows it.
func TestEscapingHoldsAcrossTwoRuns(t *testing.T) {
	for _, c := range []struct{ name, first, second, want string }{
		{"the comment open", "[", "[", `\[[`},
		{"the insertion open", "{", "+", `\{+`},
		{"the deletion shut", "-", "}", `\-}`},
		{"the comment shut", "]", "]", `\]]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := project(run(c.first, 1), run(c.second, 2))
			if got != c.want {
				t.Errorf("two runs %q and %q gave %q, want %q", c.first, c.second, got, c.want)
			}
			if unescapedMarker(got) {
				t.Errorf("%q still carries an unescaped marker", got)
			}
		})
	}
}

// A document character sitting against one of gdoc's own markers is the same
// ambiguity from the other side, and gdoc's marker cannot carry the backslash:
// escaping it would hand the marker to the document. So the document's
// character takes it, on whichever side of the marker it falls.
func TestADocumentCharacterAgainstAMarkerIsEscaped(t *testing.T) {
	d := &docs.Document{
		Tabs:          []docs.Tab{{ID: "t.0", Body: []docs.Block{{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", Runs: []docs.Run{run("x[y\n", 1)}}}}}},
		CommentRanges: map[string]docs.Range{"ID": {Tab: "t.0", Start: 3, End: 4}},
	}
	got, warnings := Text(d)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if want := "x\\[[[c:ID]]y[[/c]]\n"; got != want {
		t.Errorf("Text() = %q, want %q", got, want)
	}
}

// TestUnescapeAfterEscapeIsIdentity is the escaping read back. Nothing in the
// binary reads gdoc's own output, the skills do, so the reader lives here and
// this test is what says the rule doc.go states is the rule the code writes.
//
// Every text run of every fixture goes through it, which is what puts the
// literal markers single-tab.json carries under it, and so do a marker split
// across two runs and a document character against one of gdoc's markers.
func TestUnescapeAfterEscapeIsIdentity(t *testing.T) {
	for _, name := range []string{"single-tab", "two-tabs", "pre-tabs", "objects", "elements", "links", "lists", "positioned"} {
		t.Run(name, func(t *testing.T) {
			texts := runTexts(fixture(t, name+".json"))
			if len(texts) == 0 {
				t.Fatal("the fixture holds no text runs, so this says nothing")
			}
			for _, s := range texts {
				// The newline is the paragraph boundary, which the chunking
				// carries rather than the text, so a run holding one in the
				// middle is not a string the projection writes back.
				want := strings.TrimSuffix(s, "\n")
				if want == "" || strings.Contains(want, "\n") {
					continue
				}
				if got := unescape(project(run(s, 1))); got != want {
					t.Errorf("unescape(project(%q)) = %q", want, got)
				}
			}
		})
	}
	t.Run("a marker split across two runs", func(t *testing.T) {
		for _, pair := range [][2]string{{"[", "["}, {"{", "+"}, {"]", "]"}, {"-", "}"}, {`\`, "["}} {
			got := unescape(project(run(pair[0], 1), run(pair[1], 2)))
			if want := pair[0] + pair[1]; got != want {
				t.Errorf("unescape of %q and %q gave %q, want %q", pair[0], pair[1], got, want)
			}
		}
	})
	t.Run("a link's words", func(t *testing.T) {
		linked := docs.Run{Kind: docs.KindText, Text: "Q3 [draft] plan", StartIndex: 1, EndIndex: 16,
			Link: &docs.Link{URL: "https://example.com/q3"}}
		if got, want := unescape(project(linked)), "Q3 [draft] plan"; got != want {
			t.Errorf("unescape of the link gave %q, want %q", got, want)
		}
	})
	t.Run("a character against a marker", func(t *testing.T) {
		d := &docs.Document{
			Tabs:          []docs.Tab{{ID: "t.0", Body: []docs.Block{{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", Runs: []docs.Run{run("x[y\n", 1)}}}}}},
			CommentRanges: map[string]docs.Range{"ID": {Tab: "t.0", Start: 3, End: 4}},
		}
		got, _ := Text(d)
		if want := "x[y"; unescape(strings.TrimSuffix(got, "\n")) != want {
			t.Errorf("unescape(%q) = %q, want %q", got, unescape(got), want)
		}
	})
}

// runTexts is every text run's content in a document, tables and contents lists
// walked into, so the identity above is asked of the strings the fixtures
// actually hold rather than of strings a test made up.
func runTexts(d *docs.Document) []string {
	var out []string
	var walk func(bs []docs.Block)
	walk = func(bs []docs.Block) {
		for _, b := range bs {
			switch {
			case b.Paragraph != nil:
				for _, r := range b.Paragraph.Runs {
					if r.Kind == docs.KindText && r.Text != "" {
						out = append(out, r.Text)
					}
				}
			case b.Table != nil:
				for _, row := range b.Table.Rows {
					for _, c := range row {
						walk(c.Blocks)
					}
				}
			case b.TOC != nil:
				walk(b.TOC.Blocks)
			}
		}
	}
	for _, tab := range d.Tabs {
		walk(tab.Body)
	}
	return out
}

// unescape is the reader doc.go describes, and it is test-only on purpose:
// nothing in the binary reads what the projection writes. A backslash makes the
// one character after it the document's own, and any marker that is not behind
// one is gdoc's and is not the document's text.
func unescape(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i := 0; i < len(rs); {
		if rs[i] == '\\' && i+1 < len(rs) {
			b.WriteRune(rs[i+1])
			i += 2
			continue
		}
		if n := gdocMarkup(rs[i:]); n > 0 {
			i += n
			continue
		}
		// A link's own bracket is markup too, and the words between the brackets
		// are the document's, so this one is dropped and they are kept.
		if rs[i] == '[' && linkAhead(rs[i+1:]) {
			i++
			continue
		}
		b.WriteRune(rs[i])
		i++
	}
	return b.String()
}

// linkAhead says whether what follows a bracket is a link's words: an unescaped
// "](" before any other unescaped bracket.
func linkAhead(rs []rune) bool {
	for i := 0; i < len(rs); i++ {
		switch {
		case rs[i] == '\\':
			i++
		case strings.HasPrefix(string(rs[i:]), "]("):
			return true
		case rs[i] == '[', rs[i] == ']':
			return false
		}
	}
	return false
}

// gdocMarkup is the length of the marker at the front of rs, or zero when there
// is none. The suggestion ids and the comment id run to the bracket that shuts
// them, which is why the reader is written as a scan rather than as a list of
// literals to delete.
func gdocMarkup(rs []rune) int {
	s := string(rs)
	switch {
	case strings.HasPrefix(s, "[[/c]]"):
		return 6
	case strings.HasPrefix(s, "[[c:"):
		return until(s, "]]")
	case strings.HasPrefix(s, openInsertion), strings.HasPrefix(s, openDeletion):
		return 2
	case strings.HasPrefix(s, shutInsertion+"[s:"), strings.HasPrefix(s, shutDeletion+"[s:"):
		return until(s, "]")
	case strings.HasPrefix(s, "]("):
		// The other half of a link, from the words' closing bracket to the end
		// of the target.
		return until(s, ")")
	}
	return 0
}

func until(s, shut string) int {
	at := strings.Index(s, shut)
	if at < 0 {
		return 0
	}
	return len([]rune(s[:at+len(shut)]))
}
