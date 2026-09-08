package body

// A pipe table, built with v1's recipe.
//
// The house table styles are functionally empty: they carry no borders and no
// fill, and every visible line and shade in the master is direct formatting in
// document.xml. Relying on the style would produce an invisible table, so the
// recipe is written per cell.
//
// The four colours and the two row heights are measured values that house.yaml
// does not state: the file spells out the three front-matter tables cell by
// cell, and a table the note itself writes is not one of those. Each constant
// carries what it is for.

import (
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

const (
	// The grey grid, on every edge. The master draws black on every cell as
	// well, which turns a four-row table into a spreadsheet.
	tableBorderColor = "#c9c9c9"
	// The header band, and the alternate body band. Shading every body row the
	// same makes a wide table hard to track across; banding does the tracking
	// for the reader and leaves the header as the only strong horizontal.
	tableHeaderFill = "#bdcdd2"
	tableBandFill   = "#f3f8f9"
	tableNoFill     = "#ffffff"
	// Half a point on every line, which is the master's own weight.
	tableBorderPt = 0.5
	// 3pt above and below the text in a cell. The master sets no cell margin
	// at all, so descenders sit on the bottom border and a table reads as a
	// wall. Left and right stay at Word's own 5.4pt.
	cellMarginVPt = 3.0
	cellMarginHPt = 5.4
	// The row heights, at least. A header row is taller because it carries the
	// band.
	headerRowHeightPt = 19.5
	bodyRowHeightPt   = 15.0
)

// makeTable builds one w:tbl from the rows the walker flattened. Row zero is
// the header.
func (r *renderer) makeTable(rows [][][]Run) *etree.Element {
	columns := 0
	for _, row := range rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	usable := r.cfg.Page.UsableWidthPt()
	widths := columnWidths(rows, columns, usable)

	tbl := etree.NewElement("w:tbl")
	tblPr := sub(tbl, "w:tblPr")
	sub(tblPr, "w:tblStyle", "w:val", "TableNormal")
	sub(tblPr, "w:tblW", "w:w", twips(usable), "w:type", "dxa")
	sub(tblPr, "w:jc", "w:val", "left")
	sub(tblPr, "w:tblInd", "w:w", "0", "w:type", "dxa")
	edges(sub(tblPr, "w:tblBorders"),
		[]string{"top", "left", "bottom", "right", "insideH", "insideV"})
	// CT_TblPrBase is a sequence: tblLayout comes before tblCellMar, which
	// comes before tblLook. Word reads a w:tblPr out of order the way it reads
	// a w:pPr out of order, as a document to repair.
	sub(tblPr, "w:tblLayout", "w:type", "fixed")
	margins := sub(tblPr, "w:tblCellMar")
	for _, pair := range [][2]any{
		{"top", cellMarginVPt}, {"left", cellMarginHPt},
		{"bottom", cellMarginVPt}, {"right", cellMarginHPt},
	} {
		sub(margins, "w:"+pair[0].(string), "w:w", twips(pair[1].(float64)), "w:type", "dxa")
	}
	sub(tblPr, "w:tblLook", "w:val", "0400")

	grid := sub(tbl, "w:tblGrid")
	for _, width := range widths {
		sub(grid, "w:gridCol", "w:w", strconv.Itoa(width))
	}

	for index, row := range rows {
		header := index == 0
		tr := sub(tbl, "w:tr")
		trPr := sub(tr, "w:trPr")
		// The master lets a row break across a page, which strands half a cell
		// of prose at the top of the next page under no header.
		sub(trPr, "w:cantSplit", "w:val", "1")
		height, repeat := bodyRowHeightPt, "0"
		fill := tableNoFill
		switch {
		case header:
			height, repeat, fill = headerRowHeightPt, "1", tableHeaderFill
		case index%2 == 0:
			fill = tableBandFill
		}
		sub(trPr, "w:trHeight", "w:val", twips(height), "w:hRule", "atLeast")
		sub(trPr, "w:tblHeader", "w:val", repeat)

		for column := 0; column < columns; column++ {
			tc := sub(tr, "w:tc")
			tcPr := sub(tc, "w:tcPr")
			sub(tcPr, "w:tcW", "w:w", strconv.Itoa(widths[column]), "w:type", "dxa")
			edges(sub(tcPr, "w:tcBorders"), []string{"top", "left", "bottom", "right"})
			sub(tcPr, "w:shd", "w:val", "clear", "w:color", "auto", "w:fill", hexColor(fill))
			var cell []Run
			if column < len(row) {
				cell = row[column]
			}
			tc.AddChild(r.cellParagraph(cell, header))
		}
	}
	return tbl
}

// edges draws the same single line on each named side.
func edges(parent *etree.Element, sides []string) {
	for _, side := range sides {
		sub(parent, "w:"+side, "w:val", "single", "w:sz", eighths(tableBorderPt),
			"w:space", "0", "w:color", hexColor(tableBorderColor))
	}
}

// cellParagraph is one paragraph inside a body table's cell. An empty cell
// still carries a run, because a cell with none collapses to nothing.
func (r *renderer) cellParagraph(runs []Run, header bool) *etree.Element {
	size := r.cfg.TableText.DefaultSizePt
	zero := 0.0
	align := ""
	if header {
		align = "center"
	}
	mark := runOpts{SizePt: &size, Font: r.cfg.TableText.Font, Bold: header}
	p := r.para(paraOpts{
		Align: align, KeepNext: true, KeepLines: true,
		BeforePt: &zero, AfterPt: &zero, Mark: &mark,
	})
	r.addRuns(p, runs, runOpts{SizePt: &size, Font: r.cfg.TableText.Font, Bold: header})
	if len(p.SelectElements("w:r")) == 0 && len(p.SelectElements("w:hyperlink")) == 0 {
		sub(p, "w:r")
	}
	return p
}

// --- column widths -----------------------------------------------------------

// Rough advance width of one character of body text, in twips, plus the cell
// padding. Only used to stop a column being narrower than its longest
// unbreakable word, so an approximation is enough.
const (
	twipsPerChar = 118
	cellPadding  = 260
	maxFloor     = 2400
	// A very long word is allowed to break rather than reserve the whole page.
	maxFloorChars = 10
	// The weight one column may carry, in characters of its longest cell.
	minWeight = 8
	maxWeight = 60
)

// columnWidths shares the usable width out in proportion to how much text each
// column holds.
//
// Dividing the width equally looks fine until one column carries a sentence
// and the others carry a single word, at which point the sentence column
// becomes a tall thin ribbon. Weighting by the longest cell fixes that.
//
// The floor matters as much as the weight: a column narrower than its longest
// word makes Word break the word itself, so a date column renders as
// "11/0 8/20 26".
func columnWidths(rows [][][]Run, columns int, usablePt float64) []int {
	total := int(usablePt*20 + 0.5)
	if columns == 0 {
		return nil
	}
	weights := make([]int, columns)
	floors := make([]int, columns)
	for index := 0; index < columns; index++ {
		longestCell, longestWord := 0, 0
		for _, row := range rows {
			if index >= len(row) {
				continue
			}
			text := plainText(row[index])
			if length := len([]rune(text)); length > longestCell {
				longestCell = length
			}
			for _, word := range strings.Fields(text) {
				if length := len([]rune(word)); length > longestWord {
					longestWord = length
				}
			}
		}
		weights[index] = min(max(longestCell, minWeight), maxWeight)
		floors[index] = min(cellPadding+twipsPerChar*min(longestWord, maxFloorChars), maxFloor)
	}

	// A table of long words can want more than the page; scale the floors down
	// together rather than letting one column win.
	if sum(floors) > total {
		scale := float64(total) / float64(sum(floors))
		for i := range floors {
			floors[i] = int(float64(floors[i]) * scale)
		}
	}

	totalWeight := sum(weights)
	widths := make([]int, columns)
	for i, weight := range weights {
		widths[i] = total * weight / totalWeight
	}

	// Raise anything below its floor, and pay for it from the columns that
	// have room to spare, in proportion to how much spare they have.
	deficit := 0
	surplus := make([]int, columns)
	for i := range widths {
		if floors[i] > widths[i] {
			deficit += floors[i] - widths[i]
		}
		surplus[i] = max(0, widths[i]-floors[i])
	}
	if deficit > 0 {
		totalSurplus := sum(surplus)
		for i := range widths {
			widths[i] = max(widths[i], floors[i])
			if totalSurplus > 0 {
				widths[i] -= deficit * surplus[i] / totalSurplus
			}
		}
	}

	// The columns have to add up to the table: rounding leaves a twip or two
	// over, and a fixed-layout table whose grid disagrees with its width is one
	// Word lays out its own way.
	if over := sum(widths) - total; over != 0 {
		widest := 0
		for i, width := range widths {
			if width > widths[widest] {
				widest = i
			}
		}
		widths[widest] -= over
	}
	return widths
}

func sum(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}
