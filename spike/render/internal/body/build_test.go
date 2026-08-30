package body

import "testing"

func runsOf(texts ...string) [][]Run {
	row := make([][]Run, len(texts))
	for i, text := range texts {
		row[i] = []Run{{Text: text}}
	}
	return row
}

func TestColumnWidthsFillThePageExactly(t *testing.T) {
	rows := [][][]Run{runsOf("Version", "Date", "Owner", "Status")}

	widths := columnWidths(rows, 4)

	if got := sum(widths); got != 9864 {
		t.Errorf("total width = %d, want the text column's 9864 twips", got)
	}
}

func TestAColumnOfProseGetsMoreRoomThanAColumnOfOneWord(t *testing.T) {
	// Dividing the width equally turns the sentence column into a tall ribbon.
	rows := [][][]Run{
		runsOf("Control", "Description"),
		runsOf("AC-1", "Access to the production environment is granted only through "+
			"the identity provider, and every grant carries an expiry."),
	}

	widths := columnWidths(rows, 2)

	if widths[1] <= widths[0]*2 {
		t.Errorf("widths = %v, want the prose column much wider than the label column", widths)
	}
}

func TestAColumnIsNeverNarrowerThanItsLongestWord(t *testing.T) {
	// A column narrower than its longest unbreakable token makes Word break the
	// token itself, so a date renders as "11/0 8/20 26".
	rows := [][][]Run{
		runsOf("Date", "Notes"),
		runsOf("2026-08-29", "A description long enough to take almost the whole "+
			"page if the floor is not applied to the column beside it, which is "+
			"exactly the case that used to break the dates."),
	}

	widths := columnWidths(rows, 2)

	// 10 characters of 12pt Calibri plus the cell padding.
	const floor = 260 + 118*10
	if widths[0] < floor {
		t.Errorf("date column = %d twips, want at least %d so a date cannot break",
			widths[0], floor)
	}
}

func TestATableOfLongWordsStillFitsThePage(t *testing.T) {
	rows := [][][]Run{runsOf(
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ", "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ", "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ", "ABCDEFGHIJKLMNOPQRSTUVWXYZ")}

	widths := columnWidths(rows, 6)

	if got := sum(widths); got != 9864 {
		t.Errorf("total = %d, want the floors scaled down together rather than "+
			"one column winning", got)
	}
	for i, width := range widths {
		if width <= 0 {
			t.Errorf("column %d = %d twips, want a positive width", i, width)
		}
	}
}
