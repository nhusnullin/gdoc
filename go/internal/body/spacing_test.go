package body

import "testing"

// The numbers here are literals on purpose: a test that read house.yaml would
// follow the value wherever somebody moved it. 240 twips is 12pt, 60 is 3pt,
// 100 is 5pt. DECISIONS.md, 2026-10-14.

// A body paragraph carries 12pt beneath it and nothing above, so two
// paragraphs in a row read as two, and nobody adds a blank line by hand.
func TestABodyParagraphCarriesTwelvePointsBelow(t *testing.T) {
	out := walk(t, "One paragraph of prose.\n\nAnother paragraph of prose.\n")
	sp := find(out.Blocks[0], "w:spacing")
	if got := attr(t, sp, "w:before"); got != "0" {
		t.Errorf("a body paragraph has %s twips above it, want 0", got)
	}
	if got := attr(t, sp, "w:after"); got != "240" {
		t.Errorf("a body paragraph has %s twips below it, want 240", got)
	}
}

// 3pt between the items of a list, and 12pt under the last one, so the list
// reads as a block and the paragraph after it starts as a paragraph does.
func TestTheLastItemOfAListEndsItLikeAParagraph(t *testing.T) {
	out := walk(t, "- one\n- two\n- three\n\nAfter the list.\n")
	for i, want := range []string{"60", "60", "240"} {
		if got := attr(t, find(out.Blocks[i], "w:spacing"), "w:after"); got != want {
			t.Errorf("item %d has %s twips below it, want %s", i+1, got, want)
		}
	}
}

// The 12pt goes under the last paragraph the list emits, which in a nested
// list is the inner list's last item, not the outer item above it.
func TestANestedListEndsOnItsInnermostLastItem(t *testing.T) {
	out := walk(t, "- one\n  - inner a\n  - inner b\n- two\n  - inner c\n\nAfter.\n")
	for i, want := range []string{"60", "60", "60", "60", "240"} {
		if got := attr(t, find(out.Blocks[i], "w:spacing"), "w:after"); got != want {
			t.Errorf("list paragraph %d has %s twips below it, want %s", i+1, got, want)
		}
	}
}

// A body table's cells carry 5pt on every side, the master's own value.
func TestACellHasFivePointsOnEverySide(t *testing.T) {
	out := walk(t, "| A | B |\n|---|---|\n| one | two |\n")
	margins := find(out.Blocks[0], "w:tblCellMar")
	if margins == nil {
		t.Fatal("the table states no cell margins")
	}
	for _, side := range []string{"w:top", "w:left", "w:bottom", "w:right"} {
		if got := attr(t, find(margins, side), "w:w"); got != "100" {
			t.Errorf("%s margin is %s twips, want 100", side, got)
		}
	}
}

// Text inside a cell is single-spaced. A table is scanned, not read, and the
// cell padding supplies the gap.
func TestACellParagraphIsSingleSpaced(t *testing.T) {
	out := walk(t, "| A | B |\n|---|---|\n| one | two |\n")
	cell := find(out.Blocks[0], "w:tc")
	sp := find(cell, "w:spacing")
	if sp == nil {
		t.Fatal("the cell paragraph states no spacing")
	}
	if got := attr(t, sp, "w:line"); got != "240" {
		t.Errorf("the cell paragraph's line spacing is %s, want 240", got)
	}
}

// A list whose items rendered nothing adds no block, so the block before the
// list keeps its own spacing: a heading carries none, and must still carry
// none afterwards.
func TestAListThatRenderedNothingLeavesTheBlockBeforeItAlone(t *testing.T) {
	out := walk(t, "# Heading\n\n- ```\n  code\n  ```\n")
	if len(out.Blocks) != 1 {
		t.Fatalf("%d blocks rendered, want the heading alone", len(out.Blocks))
	}
	if sp := find(out.Blocks[0], "w:spacing"); sp != nil {
		t.Errorf("the heading grew a spacing element: %v", sp.Attr)
	}
}

// A picture with a caption carries no space beneath it, so the caption sits on
// the picture, and the caption ends the figure as a paragraph does. A picture
// with no caption ends the figure itself.
func TestAPictureSitsOnItsCaption(t *testing.T) {
	out := walk(t, "![The flow](diagram.png)\n\n![](badge.png)\n")
	if n := len(out.Blocks); n != 3 {
		t.Fatalf("%d blocks rendered, want picture, caption, picture", n)
	}
	for i, want := range []string{"0", "240", "240"} {
		if got := attr(t, find(out.Blocks[i], "w:spacing"), "w:after"); got != want {
			t.Errorf("block %d has %s twips below it, want %s", i, got, want)
		}
	}
}

// A quote ends the way a paragraph does, 12pt beneath and nothing above, so
// the values are looked at rather than inherited in silence. The page break
// that used to stand between these two blocks states its own zero spacing, and
// TestAPageBreakParagraphPushesTheNewPageNowhere reads that.
func TestAQuoteEndsLikeAParagraph(t *testing.T) {
	out := walk(t, "Prose.\n\n> Quoted words.\n")
	for i, want := range []string{"240", "240"} {
		if got := attr(t, find(out.Blocks[i], "w:spacing"), "w:after"); got != want {
			t.Errorf("block %d has %s twips below it, want %s", i, got, want)
		}
	}
}
