package body

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBookmarkNameIsWordSafe pins the rewrite, which is wider than anything
// goldmark's own generator emits: Word takes letters, digits and underscores
// only, opening with a letter, and at most 40 characters.
func TestBookmarkNameIsWordSafe(t *testing.T) {
	for _, c := range []struct{ id, want string }{
		{"purpose-and-scope", "h_purpose_and_scope"},
		{"1-1-purpose", "h_1_1_purpose"},
		{"a.b", "h_a_b"},
	} {
		if got := bookmarkName(c.id); got != c.want {
			t.Errorf("bookmarkName(%q) is %q, want %q", c.id, got, c.want)
		}
	}

	long := bookmarkName("what-counts-as-outsourcing-and-why-it-matters")
	if len(long) != 40 {
		t.Errorf("the long name is %d characters, want Word's ceiling of 40: %q",
			len(long), long)
	}
	if prefix := "h_what_counts_as_outsourcing_and_"; !strings.HasPrefix(long, prefix) {
		t.Errorf("the long name is %q, want it to open with %q", long, prefix)
	}
	if want := "h_what_counts_as_outsourcing_and_9cccf62"; long != want {
		t.Errorf("the long name is %q, want %q", long, want)
	}
}

// TestEveryHeadingCarriesABookmark: the bookmark is what an anchor link lands
// on, so it goes on every heading whether or not this note links to it.
func TestEveryHeadingCarriesABookmark(t *testing.T) {
	out := walk(t, "# Purpose and scope\n\ntext\n\n## Scope\n")
	got := serialise(t, out.Blocks)

	for _, want := range []string{
		`<w:bookmarkStart w:id="0" w:name="h_purpose_and_scope"/>`,
		`<w:bookmarkEnd w:id="0"/>`,
		`<w:bookmarkStart w:id="1" w:name="h_scope"/>`,
		`<w:bookmarkEnd w:id="1"/>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the body carries no %s:\n%s", want, got)
		}
	}

	for i, block := range out.Blocks {
		starts := findAll(block, "w:bookmarkStart")
		ends := findAll(block, "w:bookmarkEnd")
		if len(starts) != len(ends) {
			t.Errorf("block %d opens %d bookmarks and closes %d", i, len(starts), len(ends))
		}
	}
}

// TestAnAnchorLinkIsAJumpAndNotARelationship: an in-document jump is
// w:anchor with no relationship at all, and the link above the heading it
// names is the forward case the pre-walk exists for.
func TestAnAnchorLinkIsAJumpAndNotARelationship(t *testing.T) {
	out := walk(t, "See [below](#scope).\n\n## Scope\n")
	got := serialise(t, out.Blocks)

	if !strings.Contains(got, `<w:hyperlink w:anchor="h_scope">`) {
		t.Errorf("the anchor link is no jump:\n%s", got)
	}
	if strings.Contains(got, "r:id=") {
		t.Errorf("the jump names a relationship:\n%s", got)
	}
	if len(out.Media) != 0 {
		t.Errorf("the walk recorded %d relationships, want none", len(out.Media))
	}
	link := find(out.Blocks[0], "w:hyperlink")
	if link == nil {
		t.Fatalf("the paragraph holds no w:hyperlink:\n%s", got)
	}
	if color := find(link, "w:color"); color == nil || attr(t, color, "w:val") != "1155cc" {
		t.Errorf("the jump's words are not in the link colour:\n%s", got)
	}
	if find(link, "w:u") == nil {
		t.Errorf("the jump's words are not underlined:\n%s", got)
	}
	for _, warning := range out.Warnings {
		if strings.Contains(warning, "#scope") {
			t.Errorf("a resolved anchor warns: %q", warning)
		}
	}
}

// TestAnAnchorToNoHeadingWarnsAndPrintsPlainText: a jump to nowhere is
// refused the way a code block is refused, named on the envelope and printed
// as the words the author wrote.
func TestAnAnchorToNoHeadingWarnsAndPrintsPlainText(t *testing.T) {
	out := walk(t, "See [below](#nowhere).\n")
	got := serialise(t, out.Blocks)

	if strings.Contains(got, "w:hyperlink") {
		t.Errorf("the dead anchor is still a link:\n%s", got)
	}
	if !strings.Contains(got, ">below<") {
		t.Errorf("the link's words are missing:\n%s", got)
	}
	if len(out.Media) != 0 {
		t.Errorf("the walk recorded %d relationships, want none", len(out.Media))
	}
	want := "line 1: the link to #nowhere names no heading in this note, so its words are printed as plain text"
	if !has(out.Warnings, want) {
		t.Errorf("the warnings are %v, want one reading %q", out.Warnings, want)
	}
}

// TestAFigureOnlyHeadingCarriesNoBookmark: a heading that is nothing but a
// picture emits no heading paragraph, so there is nothing to land on and a
// link naming it is as dead as a link naming no heading at all.
func TestAFigureOnlyHeadingCarriesNoBookmark(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pic.png"), pngBytes(t, 1, 1), 0o600); err != nil {
		t.Fatalf("write pic.png: %v", err)
	}
	out, err := Render(config(t), []byte("See [it](#altpicpng).\n\n## ![alt](pic.png)\n"), dir, true)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	got := serialise(t, out.Blocks)

	if strings.Contains(got, "w:bookmarkStart") {
		t.Errorf("a figure-only heading carries a bookmark:\n%s", got)
	}
	if strings.Contains(got, "w:hyperlink") {
		t.Errorf("the link to a figure-only heading is a jump to nothing:\n%s", got)
	}
	if !strings.Contains(got, ">it<") {
		t.Errorf("the link's words are missing:\n%s", got)
	}
	want := "line 1: the link to #altpicpng names no heading in this note, so its words are printed as plain text"
	if !has(out.Warnings, want) {
		t.Errorf("the warnings are %v, want one reading %q", out.Warnings, want)
	}
}

// TestADeadAnchorInsideABlockQuoteNamesItsOwnLine: quoteBlock handles a
// top-level paragraph itself rather than through paragraphBlock, so it is the
// one walker path that has to set the current line for itself. Read off the
// block before it, the warning names the last heading and the author looks for
// the link there.
func TestADeadAnchorInsideABlockQuoteNamesItsOwnLine(t *testing.T) {
	out := walk(t, "# Purpose\n\n> See [below](#nowhere).\n")

	want := "line 3: the link to #nowhere names no heading in this note, so its words are printed as plain text"
	if !has(out.Warnings, want) {
		t.Errorf("the warnings are %v, want one reading %q", out.Warnings, want)
	}
}

// TestADeadAnchorInsideATableNamesTheTablesOwnLine: a cell is not a line the
// author would recognise, so the table branch sets the current line to the
// table's and every dead anchor in any of its cells names that. Read off the
// block before it, the warning names the last heading instead.
func TestADeadAnchorInsideATableNamesTheTablesOwnLine(t *testing.T) {
	out := walk(t, "# Purpose\n\n| a | b |\n| --- | --- |\n| See [below](#nowhere). | c |\n")

	want := "line 3: the link to #nowhere names no heading in this note, so its words are printed as plain text"
	if !has(out.Warnings, want) {
		t.Errorf("the warnings are %v, want one reading %q", out.Warnings, want)
	}
}

// has is one exact warning among the walk's, because a warning read with
// Contains passes on half a sentence.
func has(warnings []string, want string) bool {
	for _, warning := range warnings {
		if warning == want {
			return true
		}
	}
	return false
}

// TestOneDeadAnchorWarnsOnce: the check sits in addRuns, which runs once per
// run, and a link whose words carry mixed formatting is several runs. Warned
// per run, one dead link prints the identical sentence twice and the envelope
// says the note holds two of them. The sentence names a line and a
// destination and nothing else, so a second copy carries nothing.
func TestOneDeadAnchorWarnsOnce(t *testing.T) {
	out := walk(t, "# Purpose\n\nSee [**bold** and plain](#nowhere) here.\n")

	want := "line 3: the link to #nowhere names no heading in this note, so its words are printed as plain text"
	if got := count(out.Warnings, want); got != 1 {
		t.Errorf("the warnings are %v, want exactly one reading %q, got %d", out.Warnings, want, got)
	}
}

// TestTwoDeadAnchorsOnOneLineWarnOncePerDestination: two links on one line
// that jump nowhere are two separate sentences when they name different
// headings, and one when they name the same one, because the sentence the
// author reads would otherwise repeat itself word for word.
func TestTwoDeadAnchorsOnOneLineWarnOncePerDestination(t *testing.T) {
	out := walk(t, "# Purpose\n\nSee [a](#nowhere) and [b](#nowhere) and [c](#elsewhere).\n")

	same := "line 3: the link to #nowhere names no heading in this note, so its words are printed as plain text"
	other := "line 3: the link to #elsewhere names no heading in this note, so its words are printed as plain text"
	if got := count(out.Warnings, same); got != 1 {
		t.Errorf("the warnings are %v, want exactly one reading %q, got %d", out.Warnings, same, got)
	}
	if got := count(out.Warnings, other); got != 1 {
		t.Errorf("the warnings are %v, want exactly one reading %q, got %d", out.Warnings, other, got)
	}
}

// TestTheSameDeadAnchorOnTwoLinesWarnsTwice: the warning is deduplicated on
// the line as well as the destination, because two lines are two places the
// author has to go and fix.
func TestTheSameDeadAnchorOnTwoLinesWarnsTwice(t *testing.T) {
	out := walk(t, "# Purpose\n\nSee [a](#nowhere).\n\nAnd [b](#nowhere).\n")

	first := "line 3: the link to #nowhere names no heading in this note, so its words are printed as plain text"
	second := "line 5: the link to #nowhere names no heading in this note, so its words are printed as plain text"
	if !has(out.Warnings, first) || !has(out.Warnings, second) {
		t.Errorf("the warnings are %v, want one reading %q and one reading %q",
			out.Warnings, first, second)
	}
}

// count is how many of the walk's warnings read exactly want.
func count(warnings []string, want string) int {
	n := 0
	for _, warning := range warnings {
		if warning == want {
			n++
		}
	}
	return n
}
