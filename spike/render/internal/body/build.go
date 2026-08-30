package body

// The paragraph, heading, list item and table builders. Every measurement here
// came out of the template's own document.xml; see internal/ooxml for the
// constants and why each one differs from the master's default.

import (
	"strconv"
	"strings"

	"github.com/beevik/etree"

	"spike/gdocgo/internal/ooxml"
)

const (
	captionSz    = "20" // 10pt half-points
	captionColor = "595959"
	// 3pt between list items. The master uses zero, which runs wrapped bullets
	// together into a block.
	listItemSpacing = "60"
)

type paragraphOpts struct {
	sz      string
	justify bool
	before  string
	align   string
	color   string
	italic  bool
}

func addRuns(parent *etree.Element, runs []Run, sz, color string, hyperlinks bool,
	links *linkTable) {
	for _, run := range runs {
		if run.Text == "" {
			continue
		}
		opts := ooxml.RunOpts{
			Bold: run.Bold, Italic: run.Italic, Mono: run.Mono,
			Color: color, Highlight: run.Highlight, Sz: sz,
		}
		if hyperlinks && run.Link != "" && links != nil {
			// A real hyperlink is a w:hyperlink wrapping the run, with the
			// destination held as a package relationship. The house look is
			// Word's own: blue and underlined, which is what a reader expects
			// a link to look like.
			wrapper := parent.CreateElement("w:hyperlink")
			wrapper.CreateAttr("r:id", links.idFor(run.Link))
			opts.Color = "1155cc"
			inner := ooxml.TextRun(wrapper, run.Text, opts)
			if rPr := inner.SelectElement("w:rPr"); rPr != nil {
				ooxml.Sub(rPr, "w:u", "val", "single")
			}
			continue
		}
		ooxml.TextRun(parent, run.Text, opts)
	}
}

func (r *Renderer) makeParagraph(runs []Run, opts paragraphOpts) *etree.Element {
	if opts.sz == "" {
		opts.sz = ooxml.BodySz
	}
	if opts.before == "" {
		opts.before = "0"
	}
	paragraph := etree.NewElement("w:p")
	pPr := paragraph.CreateElement("w:pPr")
	ooxml.Sub(pPr, "w:spacing", "after", "240", "before", opts.before,
		"line", ooxml.LineSpacing, "lineRule", "auto")
	switch {
	case opts.align != "":
		ooxml.Sub(pPr, "w:jc", "val", opts.align)
	case opts.justify:
		ooxml.Sub(pPr, "w:jc", "val", "both")
	}
	rPr := ooxml.Sub(pPr, "w:rPr")
	ooxml.Sub(rPr, "w:sz", "val", opts.sz)
	ooxml.Sub(rPr, "w:szCs", "val", opts.sz)

	if opts.italic {
		italicised := make([]Run, len(runs))
		copy(italicised, runs)
		for i := range italicised {
			italicised[i].Italic = true
		}
		runs = italicised
	}
	addRuns(paragraph, runs, opts.sz, opts.color, r.opts.Hyperlinks, r.links)
	return paragraph
}

// makeCaption is the figure's alt text, under the picture: centred, small, grey.
func (r *Renderer) makeCaption(runs []Run) *etree.Element {
	return r.makeParagraph(runs, paragraphOpts{sz: captionSz, align: "center",
		color: captionColor, italic: true})
}

func (r *Renderer) makeHeading(level int, runs []Run, pageBreak bool) *etree.Element {
	if level > 6 {
		level = 6
	}
	paragraph := etree.NewElement("w:p")
	pPr := paragraph.CreateElement("w:pPr")
	ooxml.Sub(pPr, "w:pStyle", "val", "Heading"+strconv.Itoa(level))
	if pageBreak {
		// The master sets pageBreakBefore on the FIRST Heading1 only, so the
		// body starts on a clean page after the contents list and every later
		// section flows normally. Setting it on all of them costs a blank page
		// per section.
		ooxml.Sub(pPr, "w:pageBreakBefore", "val", "1")
	}
	// Heading3 carries ind left=2160 hanging=360 in this template. Clearing it
	// needs both left=0 AND hanging=0; the firstLine=0 idiom the template uses
	// for H1/H2 is not enough, because those have no inherited indent to clear.
	ooxml.Sub(pPr, "w:ind", "left", "0", "right", "0", "hanging", "0", "firstLine", "0")
	rPr := ooxml.Sub(pPr, "w:rPr")
	ooxml.Sub(rPr, "w:color", "val", ooxml.HeadColor)
	addRuns(paragraph, runs, "", ooxml.HeadColor, r.opts.Hyperlinks, r.links)
	return paragraph
}

func (r *Renderer) makeListItem(runs []Run, numID, level int) *etree.Element {
	paragraph := etree.NewElement("w:p")
	pPr := paragraph.CreateElement("w:pPr")
	numPr := ooxml.Sub(pPr, "w:numPr")
	ooxml.Sub(numPr, "w:ilvl", "val", strconv.Itoa(level))
	ooxml.Sub(numPr, "w:numId", "val", strconv.Itoa(numID))
	ooxml.Sub(pPr, "w:spacing", "after", listItemSpacing, "before", "0",
		"line", ooxml.LineSpacing, "lineRule", "auto")
	indent := ooxml.IndLeft[min(level, len(ooxml.IndLeft)-1)]
	ooxml.Sub(pPr, "w:ind", "left", strconv.Itoa(indent), "hanging",
		strconv.Itoa(ooxml.Hanging))
	ooxml.Sub(pPr, "w:jc", "val", "both")
	rPr := ooxml.Sub(pPr, "w:rPr")
	ooxml.Sub(rPr, "w:sz", "val", ooxml.BodySz)
	ooxml.Sub(rPr, "w:szCs", "val", ooxml.BodySz)
	addRuns(paragraph, runs, ooxml.BodySz, "", r.opts.Hyperlinks, r.links)
	return paragraph
}

func (r *Renderer) makeCellParagraph(runs []Run, header bool) *etree.Element {
	paragraph := etree.NewElement("w:p")
	pPr := paragraph.CreateElement("w:pPr")
	ooxml.Sub(pPr, "w:keepNext", "val", "1")
	ooxml.Sub(pPr, "w:keepLines", "val", "1")
	ooxml.Sub(pPr, "w:spacing", "after", "0", "before", "0", "lineRule", "auto")
	if header {
		ooxml.Sub(pPr, "w:jc", "val", "center")
	}
	rPr := ooxml.Sub(pPr, "w:rPr")
	ooxml.SetFonts(rPr, ooxml.BodyFont)
	if header {
		ooxml.Sub(rPr, "w:b", "val", "1")
		ooxml.Sub(rPr, "w:bCs", "val", "1")
	}
	ooxml.Sub(rPr, "w:sz", "val", ooxml.BodySz)
	ooxml.Sub(rPr, "w:szCs", "val", ooxml.BodySz)

	for _, run := range runs {
		if run.Text == "" {
			continue
		}
		opts := ooxml.RunOpts{
			Bold: run.Bold || header, Italic: run.Italic, Mono: run.Mono,
			Highlight: run.Highlight, Sz: ooxml.BodySz, Font: ooxml.BodyFont,
		}
		if r.opts.Hyperlinks && run.Link != "" && r.links != nil {
			wrapper := paragraph.CreateElement("w:hyperlink")
			wrapper.CreateAttr("r:id", r.links.idFor(run.Link))
			opts.Color = "1155cc"
			inner := ooxml.TextRun(wrapper, run.Text, opts)
			if innerRPr := inner.SelectElement("w:rPr"); innerRPr != nil {
				ooxml.Sub(innerRPr, "w:u", "val", "single")
			}
			continue
		}
		ooxml.TextRun(paragraph, run.Text, opts)
	}
	if len(paragraph.SelectElements("w:r")) == 0 &&
		len(paragraph.SelectElements("w:hyperlink")) == 0 {
		paragraph.CreateElement("w:r")
	}
	return paragraph
}

// --- column widths -----------------------------------------------------------

// Rough advance width of one character of 12pt Calibri, in twips, plus the cell
// padding. Only used to stop a column being narrower than its longest
// unbreakable word, so an approximation is enough.
const (
	twipsPerChar = 118
	cellPadding  = 260
	maxFloor     = 2400
	// A very long word is allowed to break rather than reserve the whole page.
	maxFloorChars = 10
)

// columnWidths shares the page width out in proportion to how much text each
// column holds.
//
// Dividing the width equally looks fine until one column carries a sentence and
// the others carry a single word, at which point the sentence column becomes a
// tall thin ribbon. Weighting by the longest cell fixes that.
//
// The floor matters as much as the weight: a column narrower than its longest
// word makes Word break the word itself, so a date column renders as
// "11/0 8/20 26".
func columnWidths(rows [][][]Run, columnCount int) []int {
	const minWeight, maxWeight = 8, 60

	weights := make([]int, columnCount)
	floors := make([]int, columnCount)
	for index := 0; index < columnCount; index++ {
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
	if total := sum(floors); total > ooxml.UsableTwips {
		scale := float64(ooxml.UsableTwips) / float64(total)
		for i := range floors {
			floors[i] = int(float64(floors[i]) * scale)
		}
	}

	totalWeight := sum(weights)
	widths := make([]int, columnCount)
	for i, weight := range weights {
		widths[i] = ooxml.UsableTwips * weight / totalWeight
	}

	// Raise anything below its floor, and pay for it from the columns that have
	// room to spare, in proportion to how much spare they have.
	deficit := 0
	surplus := make([]int, columnCount)
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

	if drift := sum(widths) - ooxml.UsableTwips; drift != 0 {
		widest := 0
		for i, width := range widths {
			if width > widths[widest] {
				widest = i
			}
		}
		widths[widest] -= drift
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

// makeTable builds a table using the template's direct-formatting recipe.
//
// The template's Table1..Table5 styles are functionally empty: they carry no
// borders and no fill, and every visible line and shade in the master is direct
// formatting in document.xml. Relying on the style would produce an invisible
// table, so the recipe is reproduced per cell.
func (r *Renderer) makeTable(rows [][][]Run) (*etree.Element, error) {
	if len(rows) == 0 {
		return nil, errorf("cannot build a table with no rows")
	}
	columnCount := 0
	for _, row := range rows {
		if len(row) > columnCount {
			columnCount = len(row)
		}
	}
	widths := columnWidths(rows, columnCount)

	table := etree.NewElement("w:tbl")
	tblPr := table.CreateElement("w:tblPr")
	ooxml.Sub(tblPr, "w:tblStyle", "val", "Table4")
	ooxml.Sub(tblPr, "w:tblW", "w", strconv.Itoa(ooxml.UsableTwips), "type", "dxa")
	ooxml.Sub(tblPr, "w:jc", "val", "left")
	ooxml.Sub(tblPr, "w:tblInd", "w", "0", "type", "dxa")
	borders := ooxml.Sub(tblPr, "w:tblBorders")
	for _, side := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		ooxml.Sub(borders, "w:"+side, "color", ooxml.TblBorder, "space", "0",
			"sz", "4", "val", "single")
	}
	// The master sets no cell margin at all, so descenders sit on the bottom
	// border and a table reads as a wall. 3pt top and bottom is the single
	// cheapest change to how a table feels.
	margins := ooxml.Sub(tblPr, "w:tblCellMar")
	for _, pair := range [][2]string{{"top", ooxml.CellMarginV}, {"left", ooxml.CellMarginH},
		{"bottom", ooxml.CellMarginV}, {"right", ooxml.CellMarginH}} {
		ooxml.Sub(margins, "w:"+pair[0], "w", pair[1], "type", "dxa")
	}
	ooxml.Sub(tblPr, "w:tblLayout", "type", "fixed")
	ooxml.Sub(tblPr, "w:tblLook", "val", "0400")

	grid := ooxml.Sub(table, "w:tblGrid")
	for _, width := range widths {
		ooxml.Sub(grid, "w:gridCol", "w", strconv.Itoa(width))
	}

	for rowIndex, row := range rows {
		header := rowIndex == 0
		tr := ooxml.Sub(table, "w:tr")
		trPr := ooxml.Sub(tr, "w:trPr")
		// The master allows a row to break across a page, which strands half a
		// cell of prose at the top of the next page under no header.
		ooxml.Sub(trPr, "w:cantSplit", "val", "1")
		height, tableHeader := "300", "0"
		if header {
			height, tableHeader = "390", "1"
		}
		ooxml.Sub(trPr, "w:trHeight", "val", height, "hRule", "atLeast")
		ooxml.Sub(trPr, "w:tblHeader", "val", tableHeader)
		// Shading every body row the same makes a wide table hard to track
		// across. Banding alternate rows does the tracking for the reader, and
		// leaves the header band as the only strong horizontal.
		fill := ooxml.NoFill
		switch {
		case header:
			fill = ooxml.HdrFill
		case rowIndex%2 == 0:
			fill = ooxml.RowFill
		}
		for columnIndex := 0; columnIndex < columnCount; columnIndex++ {
			tc := ooxml.Sub(tr, "w:tc")
			tcPr := ooxml.Sub(tc, "w:tcPr")
			ooxml.Sub(tcPr, "w:tcW", "w", strconv.Itoa(widths[columnIndex]), "type", "dxa")
			cellBorders := ooxml.Sub(tcPr, "w:tcBorders")
			for _, side := range []string{"top", "left", "bottom", "right"} {
				ooxml.Sub(cellBorders, "w:"+side, "color", ooxml.CellBorder,
					"space", "0", "sz", "4", "val", "single")
			}
			ooxml.Sub(tcPr, "w:shd", "fill", fill, "val", "clear")
			var cellRuns []Run
			if columnIndex < len(row) {
				cellRuns = row[columnIndex]
			}
			tc.AddChild(r.makeCellParagraph(cellRuns, header))
		}
	}
	return table, nil
}
