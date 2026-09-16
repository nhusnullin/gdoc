package render

import (
	"gdoc/internal/house"
	"github.com/beevik/etree"
)

// frontMatter builds the blocks house.yaml lists, in the order it lists them.
// Nothing here decides what belongs on the front page: the config's own
// front_matter is the order, and a block naming a label or a table that is not
// there was refused when the config was parsed.
func (b *builder) frontMatter() []*etree.Element {
	var out []*etree.Element
	for i, block := range b.cfg.FrontMatter {
		switch block.Block {
		case "cover":
			out = append(out, b.cover()...)
		case "label":
			out = append(out, b.label(block.Ref))
		case "table":
			out = append(out, b.table(b.cfg.Tables[block.Ref]))
		case "blank":
			count := block.Count
			if count == 0 {
				count = 1
			}
			for n := 0; n < count; n++ {
				out = append(out, b.blank(paraOpts{
					Align:  block.Align,
					Before: block.SpaceBeforePt,
					After:  block.SpaceAfterPt,
					Line:   block.LineSpacing,
				}, block.SizePt))
			}
		case "legend":
			out = append(out, b.legend()...)
		case "toc":
			out = append(out, b.contents())
		default:
			b.fail("front_matter[%d]: %q is not a block kind", i, block.Block)
		}
	}
	return out
}

// cover is the first page. A line naming a placeholder the note filled prints
// the note's own words as one run; without one it prints the template's, which
// are the highlighted words a person fills in by hand.
func (b *builder) cover() []*etree.Element {
	c := b.cfg.Cover
	var out []*etree.Element
	for _, blank := range c.LeadingBlanks {
		align := blank.Align
		if align == "" {
			align = c.Align
		}
		out = append(out, b.blank(paraOpts{
			Align:  align,
			Before: blank.SpaceBeforePt,
			After:  blank.SpaceAfterPt,
			Line:   blank.LineSpacing,
		}, blank.SizePt))
	}
	for _, line := range c.Lines {
		// A line that depends on a cover field the note left out is not
		// printed at all. The master offers the title twice either side of an
		// "or", for a person filling the cover in by hand to pick one, and
		// printing both published a cover reading the title, then "or", then
		// the template's own highlighted placeholder.
		if line.With != "" {
			if _, filled := b.placeholder(line.With); !filled {
				continue
			}
		}
		value, filled := b.placeholder(line.Placeholder)
		if !filled && len(line.Runs) == 0 && line.Text == "" {
			out = append(out, b.blank(paraOpts{
				Align:  c.Align,
				Before: line.SpaceBeforePt,
				After:  line.SpaceAfterPt,
				Line:   line.LineSpacing,
			}, line.SizePt))
			continue
		}
		p := b.para(paraOpts{
			Align:  c.Align,
			Before: line.SpaceBeforePt,
			After:  line.SpaceAfterPt,
		})
		switch {
		case filled:
			textRun(p, value, runOpts{SizePt: line.SizePt, Bold: line.Bold})
		case len(line.Runs) > 0:
			for _, r := range line.Runs {
				text, o := b.runText(r, runOpts{
					SizePt: line.SizePt, Bold: line.Bold,
					Color: r.Color, Highlight: r.Highlight,
				})
				textRun(p, text, o)
			}
		default:
			textRun(p, line.Text, runOpts{
				SizePt: line.SizePt, Bold: line.Bold, Highlight: line.Highlight,
			})
		}
		out = append(out, p)
	}
	for _, blank := range c.TrailingBlanks {
		out = append(out, b.blank(paraOpts{
			Align:  blank.Align,
			Before: blank.SpaceBeforePt,
			After:  blank.SpaceAfterPt,
			Line:   blank.LineSpacing,
		}, blank.SizePt))
	}
	return out
}

// label is one heading-like line in the front matter.
func (b *builder) label(ref string) *etree.Element {
	l, ok := b.cfg.Labels[ref]
	if !ok {
		b.fail("labels: no label named %q", ref)
		return el("w:p")
	}
	p := b.para(paraOpts{
		Align:     l.Align,
		Before:    l.SpaceBeforePt,
		After:     l.SpaceAfterPt,
		Line:      l.LineSpacing,
		KeepLines: l.KeepLinesTogether,
	})
	textRun(p, l.Text, runOpts{
		SizePt: l.SizePt, Bold: l.Bold, Italic: l.Italic, Color: l.Color,
	})
	return p
}

// legend is the key under the revision history: a bold word, a tab, and the
// sentence behind it.
func (b *builder) legend() []*etree.Element {
	g := b.cfg.Legend
	zero := 0.0
	var out []*etree.Element
	for i, line := range g.Lines {
		if len(line) != 2 {
			b.fail("legend.lines[%d] has %d parts, want the word and the sentence", i, len(line))
			continue
		}
		p := b.para(paraOpts{Before: &zero, After: &zero, Line: g.LineSpacing})
		textRun(p, line[0], runOpts{SizePt: g.SizePt, Bold: true, Color: g.Color})
		tabsRun(p, 1, runOpts{})
		textRun(p, line[1], runOpts{SizePt: g.SizePt, Color: g.Color})
		out = append(out, p)
	}
	return out
}

// contents is the Word field, which Google refreshes on import. A static list
// is refused rather than written: gdoc has no pagination pass, so the numbers
// in one would be wrong from the moment it was built.
func (b *builder) contents() *etree.Element {
	t := b.cfg.TOC
	if !t.Live {
		b.fail("toc: live is false, and gdoc writes no static contents list")
		return el("w:p")
	}
	zero, single := 0.0, 1.0
	sdt := el("w:sdt")
	sdtPr := sub(sdt, "w:sdtPr")
	sub(sdtPr, "w:id", "w:val", "707770780")
	obj := sub(sdtPr, "w:docPartObj")
	sub(obj, "w:docPartGallery", "w:val", "Table of Contents")
	sub(obj, "w:docPartUnique", "w:val", "1")

	content := sub(sdt, "w:sdtContent")
	p := b.para(paraOpts{Before: &zero, After: &zero, Line: &single})
	begin := sub(p, "w:r")
	sub(begin, "w:fldChar", "w:fldCharType", "begin")
	sub(begin, "w:instrText", "xml:space", "preserve").SetText(t.Instr)
	sub(begin, "w:fldChar", "w:fldCharType", "separate")
	size := t.SizePt
	textRun(p, "Refresh this table of contents in Google Docs.", runOpts{SizePt: &size})
	end := sub(p, "w:r")
	sub(end, "w:fldChar", "w:fldCharType", "end")
	content.AddChild(p)
	return sdt
}

// table is one front-matter table, cell by cell, at the widths and fills the
// config states.
func (b *builder) table(spec house.Table) *etree.Element {
	tt := b.cfg.TableText
	width := eighths(spec.Border.WidthPt)
	color := hexColor(spec.Border.Color)
	edges := func(parent *etree.Element, names []string) {
		for _, name := range names {
			sub(parent, "w:"+name, "w:val", "single", "w:sz", width, "w:space", "0", "w:color", color)
		}
	}

	total := 0.0
	for _, w := range spec.ColumnsPt {
		total += w
	}
	if len(spec.Rows) == 0 || len(spec.Rows[0].Cells) == 0 {
		b.fail("tables: a table with no rows cannot be written")
		return el("w:tbl")
	}

	tbl := el("w:tbl")
	tblPr := sub(tbl, "w:tblPr")
	sub(tblPr, "w:tblStyle", "w:val", "TableNormal")
	sub(tblPr, "w:tblW", "w:w", twips(total), "w:type", "dxa")
	// Word insets a table by its first cell's left margin, so the text rather
	// than the cell edge lines up with the page margin. Google copies that on
	// import, and the house tables sit on the margin, so cancel it.
	sub(tblPr, "w:tblInd", "w:w", twips(padding(spec.Rows[0].Cells[0])[1]), "w:type", "dxa")
	// CT_TblPrBase is a sequence, and Word reads a w:tblPr out of order the way
	// it reads a w:pPr out of order: as a document to repair. So tblBorders
	// comes before tblLayout, whatever order reads better here.
	edges(sub(tblPr, "w:tblBorders"),
		[]string{"top", "left", "bottom", "right", "insideH", "insideV"})
	sub(tblPr, "w:tblLayout", "w:type", "fixed")

	grid := sub(tbl, "w:tblGrid")
	for _, w := range spec.ColumnsPt {
		sub(grid, "w:gridCol", "w:w", twips(w))
	}

	classes, marked := 0, false
	for ri, row := range spec.Rows {
		if b.rowIsLeftOut(row) {
			continue
		}
		for _, values := range b.rowValues(row) {
			tr := sub(tbl, "w:tr")
			sub(sub(tr, "w:trPr"), "w:trHeight", "w:val", twips(row.MinHeightPt), "w:hRule", "atLeast")
			for ci, cell := range row.Cells {
				if ci >= len(spec.ColumnsPt) {
					b.fail("tables: row %d has more cells than the table has columns", ri)
					break
				}
				tc := sub(tr, "w:tc")
				tcPr := sub(tc, "w:tcPr")
				sub(tcPr, "w:tcW", "w:w", twips(spec.ColumnsPt[ci]), "w:type", "dxa")
				edges(sub(tcPr, "w:tcBorders"), []string{"top", "left", "bottom", "right"})
				if cell.Classification != "" {
					classes++
					if cell.Classification == b.fields.Classification {
						marked = true
					}
				}
				if fill := b.cellFill(cell); fill != "" {
					sub(tcPr, "w:shd", "w:val", "clear", "w:color", "auto", "w:fill", hexColor(fill))
				}
				pad := padding(cell)
				mar := sub(tcPr, "w:tcMar")
				for i, side := range []string{"top", "left", "bottom", "right"} {
					sub(mar, "w:"+side, "w:w", twips(pad[i]), "w:type", "dxa")
				}
				// w:vAlign comes after w:tcMar in the schema. Word's default
				// is top and every cell in the house file says top, so writing
				// it changes nothing today; not writing it meant a cell
				// changed to centre in the config produced a document that did
				// not move.
				if v := b.valign(cell.Valign); v != "" {
					sub(tcPr, "w:vAlign", "w:val", v)
				}
				for pi, cp := range cell.Paragraphs {
					// A repeated row carries one value per column, and it goes
					// in the cell's first paragraph. The rest of the cell is
					// the prototype's own, so the row keeps the template's
					// shape.
					var value *string
					if values != nil && pi == 0 && ci < len(values) {
						value = &values[ci]
					}
					tc.AddChild(b.cellParagraph(cp, tt, value))
				}
			}
		}
	}
	// A table that describes the classes and shades none of the one the note
	// asked for publishes a document with no classification at all, which is
	// the failure this whole mechanism exists to stop. It can only come from a
	// config and a note that spell the class differently, so it is refused
	// naming what was asked for. A note that states no class at all is not that
	// case: cover.Read defaults it, and a Fields built by hand with none says
	// there is nothing to mark.
	if classes > 0 && b.fields.Classification != "" && !marked {
		b.fail("tables: no cell describes the classification %q, and the document would carry no mark",
			b.fields.Classification)
	}
	return tbl
}

// cellFill is the shading one cell carries in the document being built. The
// rule is house.Cell's, because internal/prelude shades the same cells when it
// proposes the same table into a Google Doc.
func (b *builder) cellFill(cell house.Cell) string {
	return cell.FillFor(b.fields.Classification)
}

// rowIsLeftOut says whether a row the config marks with a list is dropped
// because the note declared that list itself. It is the blank revision row a
// person would fill in by hand, and a note that states its own revisions has
// no use for it.
func (b *builder) rowIsLeftOut(row house.Row) bool {
	if row.Without == "" {
		return false
	}
	return len(b.rowList(row.Without, "without")) > 0
}

// rowValues is the value sets one configured row renders as: one nil for an
// ordinary row, which prints the config's own words, and one list of cell
// values per item for a row the note fills in. A row marked for a list the
// note left empty prints once, as the template's own row.
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

// rowList is the note's own rows for a named list. A name this package does
// not know is refused rather than read as an empty list: a table silently left
// as the template's is a document nobody would think to check.
func (b *builder) rowList(name, key string) [][]string {
	if name != "revisions" {
		b.fail("tables: %s %q is not a list a note declares", key, name)
		return nil
	}
	out := make([][]string, 0, len(b.fields.Revisions))
	for _, rev := range b.fields.Revisions {
		out = append(out, rev.Values())
	}
	return out
}

// cellParagraph is one paragraph inside a table cell. An empty one still
// carries a run, because a cell with no run at all collapses to nothing.
func (b *builder) cellParagraph(cp house.CellParagraph, tt house.TableText, value *string) *etree.Element {
	opts := paraOpts{
		Align:       cp.Align,
		Before:      cp.SpaceBeforePt,
		After:       cp.SpaceAfterPt,
		Line:        cp.LineSpacing,
		IndentStart: cp.IndentStartPt,
		KeepNext:    cp.KeepWithNext,
		KeepLines:   cp.KeepLinesTogether,
	}
	// Docs measures a first-line indent from the margin and OOXML measures it
	// from the paragraph indent, so the config's value is the difference here.
	if cp.IndentFirstLinePt != nil {
		start := 0.0
		if cp.IndentStartPt != nil {
			start = *cp.IndentStartPt
		}
		first := *cp.IndentFirstLinePt - start
		opts.FirstLine = &first
	}
	p := b.para(opts)
	if value != nil {
		// The prototype's first run is the formatting, and the note's words
		// are the content. The yellow and the red go with the "xx" they marked.
		size := tt.DefaultSizePt
		o := runOpts{SizePt: &size, Font: tt.Font}
		if len(cp.Runs) > 0 {
			proto := cp.Runs[0]
			if proto.SizePt != nil {
				size = *proto.SizePt
			}
			o.Bold, o.Italic = proto.Bold, proto.Italic
		}
		textRun(p, *value, o)
		return p
	}
	if len(cp.Runs) == 0 {
		size := tt.DefaultSizePt
		r := sub(p, "w:r")
		if rPr := runProps(runOpts{SizePt: &size, Font: tt.Font}); rPr != nil {
			r.AddChild(rPr)
		}
		return p
	}
	for _, run := range cp.Runs {
		size := tt.DefaultSizePt
		if run.SizePt != nil {
			size = *run.SizePt
		}
		text, o := b.runText(run, runOpts{
			SizePt: &size, Font: tt.Font, Bold: run.Bold, Italic: run.Italic,
			Color: run.Color, Highlight: run.Highlight,
		})
		textRun(p, text, o)
	}
	return p
}

// padding is one cell's four margins in points, top, left, bottom then right.
// pad_tlbr_pt wins over pad_pt, which is the same number on all four sides.
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
