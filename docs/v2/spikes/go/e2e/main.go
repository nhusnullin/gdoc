package main

// End-to-end spike: markdown -> OOXML body -> spliced into the real house
// template -> valid .docx. Uses only goldmark (parse) + etree (DOM) + stdlib
// zip. This is the Go equivalent of gdoc/render/body.py's walk().

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

const src = `# Purpose

This paragraph has **bold** and *emph* and ` + "`code`" + ` in it, justified like the house style wants.

## Policy Owner

- first bullet
- second bullet
  - a nested one

1. numbered
2. also numbered

| Risk tier | Review |
|-----------|--------|
| Tier 1    | Annual |
| Tier 2    | Biennial |
`

const wns = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

// ---- OOXML helpers, the etree equivalent of gdoc/render/ooxml.py -----------

func el(tag string, kv ...string) *etree.Element {
	e := etree.NewElement("w:" + tag)
	for i := 0; i+1 < len(kv); i += 2 {
		e.CreateAttr("w:"+kv[i], kv[i+1])
	}
	return e
}

func sub(parent *etree.Element, tag string, kv ...string) *etree.Element {
	e := el(tag, kv...)
	parent.AddChild(e)
	return e
}

type run struct {
	text   string
	bold   bool
	italic bool
	mono   bool
}

func textRun(parent *etree.Element, r run) {
	e := sub(parent, "r")
	rPr := sub(e, "rPr")
	font := "Calibri"
	if r.mono {
		font = "Consolas"
	}
	f := sub(rPr, "rFonts")
	for _, a := range []string{"ascii", "hAnsi", "cs", "eastAsia"} {
		f.CreateAttr("w:"+a, font)
	}
	if r.bold {
		sub(rPr, "b", "val", "1")
		sub(rPr, "bCs", "val", "1")
	}
	if r.italic {
		sub(rPr, "i", "val", "1")
		sub(rPr, "iCs", "val", "1")
	}
	sub(rPr, "sz", "val", "24")
	sub(rPr, "szCs", "val", "24")
	t := sub(e, "t")
	t.CreateAttr("xml:space", "preserve")
	t.SetText(r.text)
}

func paragraph(runs []run, justify bool) *etree.Element {
	p := el("p")
	pPr := sub(p, "pPr")
	sub(pPr, "spacing", "after", "240", "before", "0", "line", "276", "lineRule", "auto")
	if justify {
		sub(pPr, "jc", "val", "both")
	}
	rPr := sub(pPr, "rPr")
	sub(rPr, "sz", "val", "24")
	for _, r := range runs {
		textRun(p, r)
	}
	return p
}

func heading(level int, runs []run) *etree.Element {
	p := el("p")
	pPr := sub(p, "pPr")
	sub(pPr, "pStyle", "val", "Heading"+strconv.Itoa(level))
	sub(pPr, "ind", "left", "0", "right", "0", "hanging", "0", "firstLine", "0")
	rPr := sub(pPr, "rPr")
	sub(rPr, "color", "val", "222660")
	for _, r := range runs {
		textRun(p, r)
	}
	return p
}

func listItem(runs []run, numID, level int) *etree.Element {
	p := el("p")
	pPr := sub(p, "pPr")
	numPr := sub(pPr, "numPr")
	sub(numPr, "ilvl", "val", strconv.Itoa(level))
	sub(numPr, "numId", "val", strconv.Itoa(numID))
	sub(pPr, "spacing", "after", "60", "before", "0", "line", "276", "lineRule", "auto")
	indents := []string{"720", "1440", "2160", "2880"}
	if level >= len(indents) {
		level = len(indents) - 1
	}
	sub(pPr, "ind", "left", indents[level], "hanging", "360")
	sub(pPr, "jc", "val", "both")
	rPr := sub(pPr, "rPr")
	sub(rPr, "sz", "val", "24")
	for _, r := range runs {
		textRun(p, r)
	}
	return p
}

func table(rows [][][]run) *etree.Element {
	const usable = 9864
	cols := 0
	for _, r := range rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	w := usable / cols
	tbl := el("tbl")
	tblPr := sub(tbl, "tblPr")
	sub(tblPr, "tblStyle", "val", "Table4")
	sub(tblPr, "tblW", "w", strconv.Itoa(usable), "type", "dxa")
	sub(tblPr, "tblInd", "w", "0", "type", "dxa")
	borders := sub(tblPr, "tblBorders")
	for _, side := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		sub(borders, side, "color", "c9c9c9", "space", "0", "sz", "4", "val", "single")
	}
	mar := sub(tblPr, "tblCellMar")
	for _, s := range [][2]string{{"top", "60"}, {"left", "108"}, {"bottom", "60"}, {"right", "108"}} {
		sub(mar, s[0], "w", s[1], "type", "dxa")
	}
	sub(tblPr, "tblLayout", "type", "fixed")
	grid := sub(tbl, "tblGrid")
	for i := 0; i < cols; i++ {
		sub(grid, "gridCol", "w", strconv.Itoa(w))
	}
	for ri, r := range rows {
		header := ri == 0
		tr := sub(tbl, "tr")
		trPr := sub(tr, "trPr")
		sub(trPr, "cantSplit", "val", "1")
		fill := "ffffff"
		if header {
			fill = "bdcdd2"
		} else if ri%2 == 0 {
			fill = "f3f8f9"
		}
		for ci := 0; ci < cols; ci++ {
			tc := sub(tr, "tc")
			tcPr := sub(tc, "tcPr")
			sub(tcPr, "tcW", "w", strconv.Itoa(w), "type", "dxa")
			sub(tcPr, "shd", "fill", fill, "val", "clear")
			var cell []run
			if ci < len(r) {
				cell = r[ci]
			}
			if header {
				for i := range cell {
					cell[i].bold = true
				}
			}
			p := el("p")
			pPr := sub(p, "pPr")
			sub(pPr, "spacing", "after", "0", "before", "0", "lineRule", "auto")
			rPr := sub(pPr, "rPr")
			sub(rPr, "sz", "val", "24")
			if len(cell) == 0 {
				sub(p, "r")
			}
			for _, rr := range cell {
				textRun(p, rr)
			}
			tc.AddChild(p)
		}
	}
	return tbl
}

// ---- the goldmark AST walk ------------------------------------------------

func inlineRuns(n ast.Node, source []byte, bold, italic, mono bool) []run {
	var out []run
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch v := c.(type) {
		case *ast.Text:
			out = append(out, run{string(v.Segment.Value(source)), bold, italic, mono})
			if v.SoftLineBreak() {
				out = append(out, run{" ", bold, italic, mono})
			}
		case *ast.Emphasis:
			if v.Level == 2 {
				out = append(out, inlineRuns(v, source, true, italic, mono)...)
			} else {
				out = append(out, inlineRuns(v, source, bold, true, mono)...)
			}
		case *ast.CodeSpan:
			out = append(out, inlineRuns(v, source, bold, italic, true)...)
		default:
			out = append(out, inlineRuns(c, source, bold, italic, mono)...)
		}
	}
	return out
}

func cellRuns(n ast.Node, source []byte) []run { return inlineRuns(n, source, false, false, false) }

func main() {
	md := goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough))
	source := []byte(src)
	doc := md.Parser().Parse(text.NewReader(source))

	var emitted []*etree.Element
	nextNumID := 100
	numbering := []int{} // numIds we created: (id, ordered)
	var ordered []bool

	var walk func(n ast.Node, level, numID int)
	walk = func(n ast.Node, level, numID int) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch v := c.(type) {
			case *ast.Heading:
				emitted = append(emitted, heading(v.Level, inlineRuns(v, source, false, false, false)))
			case *ast.Paragraph, *ast.TextBlock:
				runs := inlineRuns(c, source, false, false, false)
				if level == 0 {
					emitted = append(emitted, paragraph(runs, true))
				} else {
					lv := level - 1
					if lv > 3 {
						lv = 3
					}
					emitted = append(emitted, listItem(runs, numID, lv))
				}
			case *ast.List:
				id := nextNumID
				nextNumID++
				numbering = append(numbering, id)
				ordered = append(ordered, v.IsOrdered())
				for li := v.FirstChild(); li != nil; li = li.NextSibling() {
					walk(li, level+1, id)
				}
			case *east.Table:
				var rows [][][]run
				for r := v.FirstChild(); r != nil; r = r.NextSibling() {
					var cells [][]run
					for cell := r.FirstChild(); cell != nil; cell = cell.NextSibling() {
						cells = append(cells, cellRuns(cell, source))
					}
					rows = append(rows, cells)
				}
				emitted = append(emitted, table(rows))
			default:
				walk(c, level, numID)
			}
		}
	}
	walk(doc, 0, 0)
	fmt.Printf("emitted %d block elements from markdown\n", len(emitted))

	// ---- splice into the template ----------------------------------------
	zr, err := zip.OpenReader("template.docx")
	if err != nil {
		panic(err)
	}
	defer zr.Close()
	out, _ := os.Create("built.docx")
	zw := zip.NewWriter(out)

	for _, f := range zr.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()

		switch f.Name {
		case "word/document.xml":
			d := etree.NewDocument()
			if err := d.ReadFromBytes(data); err != nil {
				panic(err)
			}
			body := d.Root().SelectElement("w:body")
			sectPr := body.SelectElement("w:sectPr")
			if sectPr == nil {
				panic("no sectPr")
			}
			// strip everything from the first Heading1 to the end, keeping sectPr
			start := -1
			for i, ch := range body.ChildElements() {
				if ch.Tag != "p" {
					continue
				}
				if pPr := ch.SelectElement("w:pPr"); pPr != nil {
					if st := pPr.SelectElement("w:pStyle"); st != nil && st.SelectAttrValue("w:val", "") == "Heading1" {
						start = i
						break
					}
				}
			}
			if start < 0 {
				panic("no Heading1 in template")
			}
			removed := 0
			for _, ch := range body.ChildElements()[start:] {
				if ch == sectPr {
					continue
				}
				body.RemoveChild(ch)
				removed++
			}
			// insert our body immediately before sectPr
			idx := sectPr.Index()
			for i, e := range emitted {
				body.InsertChildAt(idx+i, e)
			}
			fmt.Printf("stripped %d sample blocks, spliced %d in before sectPr\n", removed, len(emitted))
			data, _ = d.WriteToBytes()

		case "word/numbering.xml":
			d := etree.NewDocument()
			if err := d.ReadFromBytes(data); err != nil {
				panic(err)
			}
			root := d.Root()
			for i, id := range numbering {
				num := el("num", "numId", strconv.Itoa(id))
				abs := "2"
				if ordered[i] {
					abs = "3"
				}
				sub(num, "abstractNumId", "val", abs)
				for lvl := 0; lvl < 9; lvl++ {
					ov := sub(num, "lvlOverride", "ilvl", strconv.Itoa(lvl))
					sub(ov, "startOverride", "val", "1")
				}
				root.AddChild(num)
			}
			fmt.Printf("added %d w:num definitions to numbering.xml\n", len(numbering))
			data, _ = d.WriteToBytes()
		}

		hdr := f.FileHeader
		hdr.Method = zip.Deflate
		w, _ := zw.CreateHeader(&hdr)
		w.Write(data)
	}
	zw.Close()
	out.Close()
	st, _ := os.Stat("built.docx")
	fmt.Printf("wrote built.docx (%d bytes)\n", st.Size())
	_ = strings.TrimSpace
}
