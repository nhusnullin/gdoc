package body

import (
	"strings"
	"testing"
)

// runsOf is one cell's text as the width maths sees it.
func runsOf(values ...string) [][]Run {
	var cells [][]Run
	for _, value := range values {
		cells = append(cells, []Run{{Text: value}})
	}
	return cells
}

// A4 less the two 51.05pt margins is 493.18pt, which is 9864 twips. Every
// width test states that literal rather than reading the config.
const usablePt = 493.18

func TestTheColumnsAlwaysAddUpToTheTable(t *testing.T) {
	cases := [][][][]Run{
		{runsOf("A", "B"), runsOf("one", "two")},
		{runsOf("Field", "Value", "Note"), runsOf("Reference", "x", "y")},
		{runsOf("One"), runsOf("A sentence that is much longer than the header")},
	}
	for _, rows := range cases {
		widths := columnWidths(rows, len(rows[0]), usablePt)
		if sum(widths) != 9864 {
			t.Errorf("the columns sum to %d twips, want 9864: %v", sum(widths), widths)
		}
	}
}

// A column narrower than its longest word makes Word break the word itself, so
// a date column renders as "11/0 8/20 26".
func TestAColumnIsNeverNarrowerThanItsLongestWord(t *testing.T) {
	rows := [][][]Run{
		runsOf("Ref", "Note"),
		runsOf("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
			"A note long enough that the weighting would otherwise take the whole page"),
	}
	widths := columnWidths(rows, 2, usablePt)
	// Ten characters of body text plus the cell padding is 1440 twips.
	if widths[0] < 1440 {
		t.Errorf("the reference column is %d twips, and its floor is 1440", widths[0])
	}
	if sum(widths) != 9864 {
		t.Errorf("the columns sum to %d twips, want 9864", sum(widths))
	}
}

// The weighting is what stops one column of prose becoming a tall thin ribbon.
func TestAColumnOfProseIsWiderThanAColumnOfOneWord(t *testing.T) {
	rows := [][][]Run{
		runsOf("Word", "Sentence"),
		runsOf("x", "A sentence long enough to want most of the width for itself"),
	}
	widths := columnWidths(rows, 2, usablePt)
	if widths[1] <= widths[0] {
		t.Errorf("the prose column is %d twips and the word column %d", widths[1], widths[0])
	}
}

// A row shorter than the header still fills the table: the missing cell is
// written empty rather than left out, or the row would end early and Word
// would draw the grid crooked.
func TestAShortRowIsFilledOutWithAnEmptyCell(t *testing.T) {
	out := walk(t, "| A | B | C |\n|---|---|---|\n| one |\n")
	got := serialise(t, out.Blocks)
	rows := findAll(out.Blocks[0], "w:tr")
	if len(rows) != 2 {
		t.Fatalf("the table has %d rows, want 2", len(rows))
	}
	if cells := findAll(rows[1], "w:tc"); len(cells) != 3 {
		t.Errorf("the short row has %d cells, want the table's 3:\n%s", len(cells), got)
	}
}

// An empty cell still carries a run, because a cell with none collapses to
// nothing and the row loses its height.
func TestAnEmptyCellStillCarriesARun(t *testing.T) {
	out := walk(t, "| A | B |\n|---|---|\n| one |  |\n")
	rows := findAll(out.Blocks[0], "w:tr")
	empty := findAll(rows[1], "w:tc")[1]
	if len(findAll(empty, "w:r")) == 0 {
		t.Errorf("the empty cell carries no run: %s", empty.FullTag())
	}
}

// A cell's text is set in the house table face at the house table size.
func TestACellIsSetInTheHouseTableFace(t *testing.T) {
	out := walk(t, "| A |\n|---|\n| one |\n")
	got := serialise(t, out.Blocks)
	if !strings.Contains(got, `w:ascii="Calibri"`) {
		t.Errorf("the cell is not set in Calibri:\n%s", got)
	}
	// 12pt table text is 24 half-points.
	if !strings.Contains(got, `<w:sz w:val="24"/>`) {
		t.Errorf("the cell is not at 12pt:\n%s", got)
	}
}

// A paragraph after a table gets 12pt above it. A table carries no space
// beneath, so the next paragraph would sit hard against its bottom border.
func TestAParagraphAfterATableIsPushedOffIt(t *testing.T) {
	out := walk(t, "| A |\n|---|\n| one |\n\nWords after the table.\n")
	last := out.Blocks[len(out.Blocks)-1]
	if got := attr(t, find(last, "w:spacing"), "w:before"); got != "240" {
		t.Errorf("the paragraph after the table has %s twips above it, want 240", got)
	}
}
