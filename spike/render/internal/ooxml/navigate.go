package ooxml

import (
	"strings"

	"github.com/beevik/etree"
)

// The navigation surface python-docx offers, reduced to what the surgery uses.
// Every one of these looks at DIRECT children only, which is what python-docx
// does too: a document's paragraphs are the ones in the body, never the ones
// inside a table cell, and a paragraph's runs are the ones it owns rather than
// the ones a hyperlink wraps.

func Children(parent *etree.Element, tag string) []*etree.Element {
	return parent.SelectElements(tag)
}

func Paragraphs(parent *etree.Element) []*etree.Element { return parent.SelectElements("w:p") }

func Tables(parent *etree.Element) []*etree.Element { return parent.SelectElements("w:tbl") }

func Rows(table *etree.Element) []*etree.Element { return table.SelectElements("w:tr") }

func Cells(row *etree.Element) []*etree.Element { return row.SelectElements("w:tc") }

func Runs(paragraph *etree.Element) []*etree.Element { return paragraph.SelectElements("w:r") }

// RunText joins what a run shows. A tab and a break are characters a reader
// sees, so they count: matching a placeholder against text that silently
// dropped them would find the wrong offset.
func RunText(run *etree.Element) string {
	var b strings.Builder
	for _, child := range run.ChildElements() {
		switch child.FullTag() {
		case "w:t":
			b.WriteString(child.Text())
		case "w:tab":
			b.WriteString("\t")
		case "w:br", "w:cr":
			b.WriteString("\n")
		}
	}
	return b.String()
}

// SetRunText replaces what a run shows, keeping its w:rPr and so its formatting.
func SetRunText(run *etree.Element, text string) {
	for _, child := range run.ChildElements() {
		if child.FullTag() != "w:rPr" {
			run.RemoveChild(child)
		}
	}
	t := run.CreateElement("w:t")
	t.CreateAttr("xml:space", "preserve")
	t.SetText(text)
}

func ParagraphText(paragraph *etree.Element) string {
	var b strings.Builder
	for _, run := range Runs(paragraph) {
		b.WriteString(RunText(run))
	}
	return b.String()
}

// CellText joins a cell's paragraphs the way python-docx does, with a newline
// between them.
func CellText(cell *etree.Element) string {
	texts := make([]string, 0, 2)
	for _, paragraph := range Paragraphs(cell) {
		texts = append(texts, ParagraphText(paragraph))
	}
	return strings.Join(texts, "\n")
}

// Remove detaches an element from its parent.
func Remove(element *etree.Element) {
	if parent := element.Parent(); parent != nil {
		parent.RemoveChild(element)
	}
}

// GetOrAddPPr returns a paragraph's w:pPr, created first in document order when
// it has none. w:pPr must be the first child or Word rejects the paragraph.
func GetOrAddPPr(paragraph *etree.Element) *etree.Element {
	if pPr := paragraph.SelectElement("w:pPr"); pPr != nil {
		return pPr
	}
	pPr := etree.NewElement("w:pPr")
	paragraph.InsertChildAt(0, pPr)
	return pPr
}
