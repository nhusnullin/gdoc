// What the colleague's pages say about a footnote and about the seven marks.
//
// The palette lives in house.yaml and nowhere else, so these tests read it
// rather than restating it: a colour emoji added to the file is one the build
// page and the export page have to show, and a page that shows six of seven
// marks is a page that teaches an author to type the one gdoc refuses.

package boundary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/house"
)

// guidePage is one page under docs/guide as text. A page that is not there
// fails rather than reading as empty, because a scan of nothing passes.
func guidePage(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(repoRoot, "docs/guide", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("docs/guide/%s is named by this test and is not there: %v", name, err)
	}
	return string(b)
}

// markSyntax is every spelling an author types for a mark, out of the embedded
// house style: the two the yellow mark answers to and one each for the rest.
func markSyntax(t *testing.T) []string {
	t.Helper()
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the embedded house style must load: %v", err)
	}
	var out []string
	for _, m := range cfg.Marks {
		out = append(out, m.Syntax...)
	}
	if len(out) < len(cfg.Marks) {
		t.Fatalf("every mark states at least one syntax, and %d marks gave %d", len(cfg.Marks), len(out))
	}
	return out
}

// TestTheBuildPageShowsAFootnoteAndTheSevenMarks holds the page a colleague
// reads before writing a note. Both are new in milestone 18, and both are
// syntax an author types: a footnote that does not line up and a mark gdoc
// cannot carry are refused naming the line, so a page that shows neither sends
// somebody to a refusal with nothing to read.
func TestTheBuildPageShowsAFootnoteAndTheSevenMarks(t *testing.T) {
	text := guidePage(t, "publishing.md")
	for _, want := range append(markSyntax(t),
		// The footnote as an author writes it: the mark in the prose and
		// the definition on its own line.
		"[^1]",
		"[^1]:",
		// Whose syntax the marks are, so a colleague reads their own
		// editor's documentation for the rest.
		"Obsidian",
		// Where the binary answers the question itself, colours included.
		"gdoc help marks",
		// The refusals, each in the words the page names it by.
		"names no note",
		"nothing marks",
		"inside a heading",
		"inside a table cell",
		"a mark inside a mark",
		// The two counts the envelope carries for them.
		"body.marks",
		"body.footnotes",
	) {
		if !strings.Contains(text, want) {
			t.Errorf("docs/guide/publishing.md never says %q, and a colleague writing a note has nowhere to read it", want)
		}
	}
}

// TestTheExportPageSaysWhatComesHome holds the other direction. A mark comes
// home as the syntax it left as and a footnote as its definitions under a
// rule, so the page that says what an exported file holds has to name both,
// and the one colour it cannot name is counted on the envelope.
func TestTheExportPageSaysWhatComesHome(t *testing.T) {
	text := guidePage(t, "exporting.md")
	for _, want := range append(markSyntax(t),
		// The footnote, which the file has carried since the first export.
		"[^1]",
		// The count of what no mark claims, by hex.
		"marks.other",
		// Where the seven are listed with their colours.
		"gdoc help marks",
	) {
		if !strings.Contains(text, want) {
			t.Errorf("docs/guide/exporting.md never says %q, and somebody reading an exported file cannot tell what it holds", want)
		}
	}
}

// TestThePublishingPageSaysAThematicBreakIsAPageBreak holds the page to what
// the build does with `---` since milestone 19. The page listed "horizontal
// rules" among what a note may hold while the line drew a grey border; it
// starts a new page now (DECISIONS.md, 2026-10-16), so a page that still
// promises a rule teaches an author to expect a line nothing draws. The two
// drops are on it too, because a note with three dividers and two pages in the
// document is a difference nobody can see from the file.
func TestThePublishingPageSaysAThematicBreakIsAPageBreak(t *testing.T) {
	text := guidePage(t, "publishing.md")
	for _, want := range []string{
		// What an author types, and what the document does with it.
		"`---`",
		"page break",
		// The other two spellings goldmark reads as the same break.
		"`***`",
		"`___`",
		// The two drops, in the words the page names them by.
		"before nothing",
		"footnote definitions",
		// The count the build and the publish envelopes carry for it.
		"body.page_breaks",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("docs/guide/publishing.md never says %q, and a colleague writing a note has nowhere to read it", want)
		}
	}
	if strings.Contains(text, "horizontal rules") {
		t.Error("docs/guide/publishing.md still lists horizontal rules among what a note may hold, and `---` has not drawn one since milestone 19")
	}
}

// markdownExamples is every fenced ```markdown block on a page, which is what
// the page teaches a colleague to type. Prose may name a spelling gdoc refuses,
// because naming it is how somebody finds the refusal; an example may not.
func markdownExamples(text string) []string {
	const fence = "```"
	var out []string
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		if lines[i] != fence+"markdown" {
			continue
		}
		var block []string
		for i++; i < len(lines) && !strings.HasPrefix(lines[i], fence); i++ {
			block = append(block, lines[i])
		}
		out = append(out, strings.Join(block, "\n"))
	}
	return out
}

// TestThePublishingPageSaysAPencilIsAComment holds the page to what the grey
// mark is since milestone 20. A pencil is a comment, HTML's or Obsidian's, and
// the Pandoc span it was written as is refused by line now: a page that still
// shows the span in an example teaches a colleague to write a note the build
// will not take, and one that never names the span leaves somebody who wrote
// it last month with nothing to read.
func TestThePublishingPageSaysAPencilIsAComment(t *testing.T) {
	text := guidePage(t, "publishing.md")
	for _, want := range []string{
		// What the mark is, in the words the page names it by.
		"a pencil is a comment",
		// The two spellings an author types, both of them stated in
		// house.yaml and both read by the build.
		"`<!-- words -->`",
		"`%% words %%`",
		// The shape that is gone, named as the refusal it is now.
		"`[words]{.pencil}`",
		"before milestone 20",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("docs/guide/publishing.md never says %q, and a colleague writing a grey aside has nowhere to read it", want)
		}
	}
	examples := markdownExamples(text)
	if len(examples) == 0 {
		t.Fatal("docs/guide/publishing.md shows no markdown example, and this test reads what the page teaches an author to type")
	}
	for _, block := range examples {
		if strings.Contains(block, ".pencil") {
			t.Errorf("a markdown example on docs/guide/publishing.md writes .pencil, which the build refuses by line:\n%s", block)
		}
	}
}

// TestThePresencePageWalksTheSecondMorningsPanel holds the page a colleague
// reads about the menu bar app to the panel as it stands after 2026-10-18.
// Nail used the panel for a morning and the top of it moved: the account, its
// sign-out and the version are in the header, the group they left is SETTINGS
// and carries the hub, the Thread mark row is gone, the edges are pinned and
// the middle scrolls, and a document row remembers the Claude Code session its
// last run came from. A page that walks the old panel sends somebody looking
// at the foot for a row that is at the top.
func TestThePresencePageWalksTheSecondMorningsPanel(t *testing.T) {
	text := guidePage(t, "presence.md")
	for _, want := range []string{
		// The header, which is the part that is new.
		"### The header",
		// What the two things in it are, each with its one action.
		"Sign out",
		"the update icon",
		// What moved and what did not: the edges are pinned, the middle
		// scrolls, and NOW has half the body before it scrolls itself.
		"The edges are pinned and the middle scrolls",
		// DOCUMENTS: the order rule, in one place, and the two answers a
		// person gives about one document.
		"Pinned documents come first",
		"Hide from the list",
		// SETTINGS: the hub row, which is what greys the md icon until it
		// is answered.
		"Hub is the folder your notes are in",
		// The session: the field in the record, the link it builds, and
		// that nobody opens that link today.
		"the Claude Code session",
		"claude://code/continue?session=",
		// The app's own preferences, which the pins and the hides joined.
		"eight things",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("docs/guide/presence.md never says %q, and somebody reading it walks a panel that is not there", want)
		}
	}

	// The page walks the panel from the top, because a guide read in order
	// is a panel read in order.
	walk := []string{
		"### The header", "### Now", "### Documents", "### Today and Yesterday", "### Settings",
	}
	at := -1
	for _, heading := range walk {
		next := strings.Index(text, heading)
		if next < 0 {
			t.Fatalf("docs/guide/presence.md has no %q heading, and the walk is the panel's own order", heading)
		}
		if next < at {
			t.Errorf("docs/guide/presence.md puts %q above the section before it, and the panel does not", heading)
		}
		at = next
	}
}
