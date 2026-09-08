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
				textRun(p, r.Text, runOpts{
					SizePt: line.SizePt, Bold: line.Bold,
					Color: r.Color, Highlight: r.Highlight,
				})
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
	sub(tblPr, "w:tblLayout", "w:type", "fixed")
	edges(sub(tblPr, "w:tblBorders"),
		[]string{"top", "left", "bottom", "right", "insideH", "insideV"})

	grid := sub(tbl, "w:tblGrid")
	for _, w := range spec.ColumnsPt {
		sub(grid, "w:gridCol", "w:w", twips(w))
	}

	for ri, row := range spec.Rows {
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
			if cell.Fill != "" {
				sub(tcPr, "w:shd", "w:val", "clear", "w:color", "auto", "w:fill", hexColor(cell.Fill))
			}
			pad := padding(cell)
			mar := sub(tcPr, "w:tcMar")
			for i, side := range []string{"top", "left", "bottom", "right"} {
				sub(mar, "w:"+side, "w:w", twips(pad[i]), "w:type", "dxa")
			}
			for _, cp := range cell.Paragraphs {
				tc.AddChild(b.cellParagraph(cp, tt))
			}
		}
	}
	return tbl
}

// cellParagraph is one paragraph inside a table cell. An empty one still
// carries a run, because a cell with no run at all collapses to nothing.
func (b *builder) cellParagraph(cp house.CellParagraph, tt house.TableText) *etree.Element {
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
		textRun(p, run.Text, runOpts{
			SizePt: &size, Font: tt.Font, Bold: run.Bold, Italic: run.Italic,
			Color: run.Color, Highlight: run.Highlight,
		})
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
