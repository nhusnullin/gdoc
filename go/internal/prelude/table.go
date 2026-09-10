package prelude

import (
	"strings"

	"gdoc/internal/docsreq"
	"gdoc/internal/house"
)

// A table is inserted and then filled, which is more requests than the docx
// writer needs and is the shape the Docs API has. insertTable makes a grid of
// empty cells and nothing else, so every word in it is an insertText afterwards
// and every fill, padding and border is an updateTableCellStyle. The docx
// writer states a cell and its contents in the same element.
//
// The index accounting is the part to be careful with, because nothing in the
// document is read: this writer computes where each cell lands. An empty table
// is one index unit for the newline insertTable writes in front of it, one for
// the table, one for each row, and one for each cell plus one for that cell's
// own paragraph mark. A 2x2 is therefore twelve, which is what Docs recorded on
// 2026-09-10 when the probe suggested one: twelve marks for one 2x2 table.
//
// Everything is written in reading order and the cursor only moves forward, so
// an insert into one cell shifts the cells behind it and never the ranges
// already stated. A cell that already holds one paragraph mark takes its
// paragraphs joined by newlines: two paragraphs are one newline, because the
// last mark is the cell's own.

// cellPlan is one table cell as it will be written.
type cellPlan struct {
	fill   string
	valign string
	// pad is the four margins in points, top, left, bottom then right.
	pad    [4]float64
	border house.Border
	paras  []cellParaPlan
}

// cellParaPlan is one paragraph inside a cell: its look and its runs.
type cellParaPlan struct {
	spec paraSpec
	runs []run
	text string
}

// table writes one named front-matter table at the cursor.
func (b *builder) table(name string) {
	spec, ok := b.cfg.Tables[name]
	if !ok {
		b.fail("tables: no table named %q", name)
		return
	}
	cols := len(spec.ColumnsPt)
	rows := b.tableRows(name, spec, cols)
	if b.err != nil {
		return
	}
	if cols == 0 || len(rows) == 0 {
		b.fail("tables: %q has no rows or no columns, and a table needs both", name)
		return
	}

	start := b.at
	b.request("insertTable", map[string]any{
		"rows":     len(rows),
		"columns":  cols,
		"location": map[string]any{"index": start},
	})
	// insertTable writes a newline in front of the table it makes, so gdoc
	// owns one more paragraph than the table itself. It is styled rather than
	// left alone: unstated, it keeps whatever the paragraph the prelude was
	// inserted into carried.
	b.style("updateParagraphStyle", "paragraphStyle", span(start, start+1),
		b.look(paraSpec{}))
	b.style("updateTextStyle", "textStyle", span(start, start+1), b.textLook(mark{}))
	b.count++

	tableStart := start + 1
	at := tableStart + 1
	for ri, row := range rows {
		at++ // the row's own index
		for ci, cell := range row {
			at++ // the cell's own index
			at = b.cellContent(cell, at)
			b.cellStyle(tableStart, ri, ci, cell)
			b.cells++
		}
	}
	b.at = at
	b.tables++
}

// tableRows is the rows one table renders as, each padded to the table's own
// width. A row marked with a list the fields declare is one row per item, and
// the blank row a person would fill in by hand is left out once they do.
func (b *builder) tableRows(name string, spec house.Table, cols int) [][]cellPlan {
	var out [][]cellPlan
	for ri, row := range spec.Rows {
		if b.rowIsLeftOut(row) {
			continue
		}
		if len(row.Cells) > cols {
			b.fail("tables: row %d of %q has %d cells and the table has %d columns",
				ri, name, len(row.Cells), cols)
			return nil
		}
		for _, values := range b.rowValues(row) {
			cells := make([]cellPlan, cols)
			for ci := range cells {
				var cell house.Cell
				if ci < len(row.Cells) {
					cell = row.Cells[ci]
				}
				var value *string
				if values != nil && ci < len(values) {
					value = &values[ci]
				}
				cells[ci] = b.cellPlan(cell, spec.Border, value)
			}
			out = append(out, cells)
		}
	}
	return out
}

// rowIsLeftOut says whether a row the config marks with a list is dropped
// because the fields declared that list themselves. It is the blank revision
// row a person would fill in by hand, which v1 removes for the same reason.
func (b *builder) rowIsLeftOut(row house.Row) bool {
	if row.Without == "" {
		return false
	}
	return len(b.rowList(row.Without, "without")) > 0
}

// rowValues is the value sets one configured row renders as: one nil for an
// ordinary row, which writes the config's own words, and one list of cell
// values per item for a row the fields fill in. A row marked for a list the
// fields left empty is written once, as the template's own row.
func (b *builder) rowValues(row house.Row) [][]string {
	if row.Repeat == "" {
		return [][]string{nil}
	}
	values := b.rowList(row.Repeat, "repeat")
	if len(values) == 0 {
		return [][]string{nil}
	}
	return values
}

// rowList is the fields' own rows for a named list. A name this package does
// not know is refused rather than read as an empty list: a table silently left
// as the template's is a document nobody would think to check.
func (b *builder) rowList(name, key string) [][]string {
	if name != "revisions" {
		b.fail("tables: %s %q is not a list the cover fields declare", key, name)
		return nil
	}
	out := make([][]string, 0, len(b.fields.Revisions))
	for _, rev := range b.fields.Revisions {
		out = append(out, rev.Values())
	}
	return out
}

// cellPlan is one configured cell as it will be written, with the fields' own
// value in its first paragraph when the row is one they filled in.
func (b *builder) cellPlan(cell house.Cell, border house.Border, value *string) cellPlan {
	out := cellPlan{
		fill:   cellFill(cell, b.fields),
		valign: cell.Valign,
		pad:    padding(cell),
		border: border,
	}
	tt := b.cfg.TableText
	for pi, cp := range cell.Paragraphs {
		plan := cellParaPlan{spec: paraSpec{
			align:             cp.Align,
			before:            cp.SpaceBeforePt,
			after:             cp.SpaceAfterPt,
			line:              cp.LineSpacing,
			indentStartPt:     cp.IndentStartPt,
			indentFirstLinePt: cp.IndentFirstLinePt,
			keepWithNext:      cp.KeepWithNext,
			keepLinesTogether: cp.KeepLinesTogether,
		}}
		switch {
		case value != nil && pi == 0:
			// The prototype's first run is the formatting and the fields' own
			// words are the content. The yellow and the red go with the "xx"
			// they marked, because they mean "not filled in yet".
			m := mark{font: tt.Font}
			if len(cp.Runs) > 0 {
				proto := cp.Runs[0]
				m.sizePt, m.bold, m.italic = proto.SizePt, proto.Bold, proto.Italic
			}
			plan.runs = []run{{text: *value, look: b.textLook(b.cellSize(m))}}
		default:
			for _, r := range cp.Runs {
				text, m := b.runText(r, mark{
					sizePt: r.SizePt, font: tt.Font, bold: r.Bold, italic: r.Italic,
					underline: r.Underline, color: r.Color, highlight: r.Highlight,
				})
				plan.runs = append(plan.runs, run{text: text, look: b.textLook(b.cellSize(m))})
			}
		}
		var words strings.Builder
		for _, r := range plan.runs {
			words.WriteString(r.text)
		}
		plan.text = words.String()
		out.paras = append(out.paras, plan)
	}
	return out
}

// cellSize is one cell run's mark at the table's own default size when the
// file states none for it. A cell run that fell back to the document default
// would be 11pt in a table the house sets at 12.
func (b *builder) cellSize(m mark) mark {
	if m.sizePt == nil {
		size := b.cfg.TableText.DefaultSizePt
		m.sizePt = &size
	}
	return m
}

// cellContent writes one cell's paragraphs into a cell that already holds one
// empty paragraph, and answers the index one past the cell's last mark.
func (b *builder) cellContent(c cellPlan, at int) int {
	if len(c.paras) == 0 {
		// The cell Docs made still has its own paragraph mark.
		b.count++
		return at + 1
	}
	texts := make([]string, len(c.paras))
	for i, p := range c.paras {
		texts[i] = p.text
	}
	// The last paragraph's mark is the one the cell already holds, so the
	// paragraphs are joined by newlines rather than each ending in one.
	if joined := strings.Join(texts, "\n"); joined != "" {
		b.request("insertText", map[string]any{
			"text":     joined,
			"location": map[string]any{"index": at},
		})
	}
	pos := at
	for _, p := range c.paras {
		end := pos + docsreq.Len(p.text) + 1
		b.style("updateParagraphStyle", "paragraphStyle", span(pos, end), b.look(p.spec))
		b.count++
		if p.text == "" {
			// An empty paragraph is its mark alone, and the mark is what
			// carries a blank line's height.
			if len(p.runs) > 0 {
				b.style("updateTextStyle", "textStyle", span(pos, pos+1), p.runs[0].look)
			}
			pos = end
			continue
		}
		rat := pos
		for _, r := range p.runs {
			rend := rat + docsreq.Len(r.text)
			if rend == rat {
				continue // a run with no words states nothing
			}
			b.style("updateTextStyle", "textStyle", span(rat, rend), r.look)
			rat = rend
		}
		pos = end
	}
	return pos
}

// cellStyle is one cell's own look: its padding, its fill, its borders and
// where its text sits in it.
//
// One request per cell rather than one per table, which is the opposite of
// internal/restyle's rule and for the opposite reason. A restyle gives every
// cell of somebody's table one look and cannot read the table's real width, so
// it names the table. Here the fills differ cell by cell, because a
// classification row is shaded only when the fields declare that class, and
// this writer built the grid itself so it knows exactly how wide it is.
func (b *builder) cellStyle(tableStart, row, col int, c cellPlan) {
	f := docsreq.NewFields()
	for i, side := range []string{"paddingTop", "paddingLeft", "paddingBottom", "paddingRight"} {
		f.Put(side, docsreq.Points(c.pad[i]))
	}
	f.Put("contentAlignment", b.contentAlignment(c.valign))
	// An OptionalColor with no colour in it is Docs' way of saying "no fill",
	// so a cell the house style does not shade arrives unshaded rather than
	// keeping whatever a cell in the document beside it carried.
	fill := map[string]any{}
	if c.fill != "" {
		color, ok := docsreq.Color(c.fill)
		if !ok {
			b.fail("tables: %q is not a colour", c.fill)
			return
		}
		fill = color
	}
	f.Put("backgroundColor", fill)
	border := map[string]any{
		"width":     docsreq.Points(c.border.WidthPt),
		"dashStyle": "SOLID",
	}
	if c.border.Color != "" {
		color, ok := docsreq.Color(c.border.Color)
		if !ok {
			b.fail("tables: %q is not a border colour", c.border.Color)
			return
		}
		border["color"] = color
	}
	for _, side := range []string{"borderTop", "borderBottom", "borderLeft", "borderRight"} {
		f.Put(side, border)
	}
	b.request("updateTableCellStyle", map[string]any{
		"tableRange": map[string]any{
			"tableCellLocation": map[string]any{
				"tableStartLocation": map[string]any{"index": tableStart},
				"rowIndex":           row,
				"columnIndex":        col,
			},
			"rowSpan":    1,
			"columnSpan": 1,
		},
		"tableCellStyle": f.Set,
		"fields":         f.Mask(),
	})
}

// contentAlignments maps the house style's own word onto the Docs alignment of
// a cell's contents.
var contentAlignments = map[string]string{
	"top": "TOP", "center": "MIDDLE", "centre": "MIDDLE", "middle": "MIDDLE",
	"bottom": "BOTTOM",
}

// contentAlignment is where a cell's text sits in it. A word Docs has no
// alignment for is refused by name rather than dropped, which is the rule the
// paragraph alignment follows: a cell silently top aligned is a table nobody
// would think to check. A file that states nothing is TOP, which is what every
// cell in the house file says and what Word does.
func (b *builder) contentAlignment(word string) string {
	if word == "" {
		return "TOP"
	}
	if v, ok := contentAlignments[strings.ToLower(strings.TrimSpace(word))]; ok {
		return v
	}
	b.fail("valign %q is not one the Docs API has", word)
	return "TOP"
}

// padding is one cell's four margins in points, top, left, bottom then right.
// pad_tlbr_pt wins over pad_pt, which is the same number on all four sides. It
// is internal/render's rule in this writer's units.
func padding(c house.Cell) [4]float64 {
	if len(c.PadTLBRPt) == 4 {
		return [4]float64{c.PadTLBRPt[0], c.PadTLBRPt[1], c.PadTLBRPt[2], c.PadTLBRPt[3]}
	}
	pad := 0.0
	if c.PadPt != nil {
		pad = *c.PadPt
	}
	return [4]float64{pad, pad, pad, pad}
}
