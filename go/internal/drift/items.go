// The list. One row per value, each with the two extractors that read it, and
// the names are compare.py's so a row here is findable in the 2026-08-29
// report.
//
// Every item is built by `item`, which takes one reader and hands back both
// extractors over it. That is the whole reason this file is shaped the way it
// is: two extractors written out by hand are two chances for one row to mean
// one thing offline and another thing live, which would make the two gates
// disagree without either of them failing. The Source interface is what makes
// one reader serve both, and the compiler is what keeps the two document types
// answering the same questions.
package drift

import "fmt"

// Source is a document, whichever way it was read. Both halves answer the same
// questions and each answers them in its own vocabulary: the docx resolves a
// value through the document defaults and the basedOn chain, and the Docs
// answer carries what Google already resolved.
type Source interface {
	Page(name string) any
	FirstPageHeaderFooter() any
	CustomHeaderFooterMargins() any
	PageNumberStart() any
	HasReference(kind, which string) any
	Style(id string) Style
	Segment(kind, which string) Segment
	Logo() Logo
	TOCFields() []string
	TOCInstr() any
	Tables() []TableValues
	FirstHeading(level int) Heading
	BodyParagraph() BodyParagraph
	Bullets() any
}

// Item is one row of the comparison: a name, the tolerance it is judged under,
// and the two extractors.
type Item struct {
	Name string
	Tol  float64
	Docx func(*Docx) any
	Doc  func(*Doc) any
}

// item builds one row from one reader. See the file comment for why it is one
// reader rather than two.
func item(name string, tol float64, read func(Source) any) Item {
	return Item{
		Name: name,
		Tol:  tol,
		Docx: func(d *Docx) any { return read(d) },
		Doc:  func(d *Doc) any { return read(d) },
	}
}

// exactTol is the tolerance an item carries when nothing may move at all. Zero
// would mean "no tolerance stated", which is the default of three quarters of a
// point, so the one that means zero is spelled negative and read by tolerance().
const exactTol = -1

// widthTol is the tolerance on a table's column widths. One point, compare.py's
// number: a column a point narrower moves no text.
const widthTol = 1

// offsetTol is the tolerance on the logo's position. Two points, compare.py's
// number, and what it is really for is that the two files state the same offset
// in EMU and land a few hundredths of a point apart.
const offsetTol = 2

// The nine named styles, under the names Docs gives them, which is how the
// 2026-08-29 report lists them.
var styleIDs = []string{"Normal", "Heading1", "Heading2", "Heading3", "Heading4",
	"Heading5", "Heading6", "Title", "Subtitle"}

// The three front matter tables, in the order house.yaml lists them. The names
// are the words on the page, so a failing row says which table moved.
var tableNames = []string{"Version Control", "Revision History", "Document Classification"}

// The four headers and footers, under compare.py's labels.
var segments = []struct{ label, kind, which string }{
	{"first-page header", "header", "first"},
	{"default header", "header", "default"},
	{"first-page footer", "footer", "first"},
	{"default footer", "footer", "default"},
}

// Items is the list, built once.
var Items = buildItems()

// buildItems writes the list out in the order compare.py wrote it, so the two
// reports read side by side.
func buildItems() []Item {
	items := []Item{}
	add := func(more ...Item) { items = append(items, more...) }

	// ------------------------------------------------------ the page itself
	add(item("page width (pt)", 0, func(s Source) any { return s.Page("width") }))
	add(item("page height (pt)", 0, func(s Source) any { return s.Page("height") }))
	for _, name := range []string{"marginTop", "marginBottom", "marginLeft", "marginRight",
		"marginHeader", "marginFooter"} {
		add(item("documentStyle."+name, 0, func(s Source) any { return s.Page(name) }))
	}
	add(item("documentStyle.useFirstPageHeaderFooter", 0, func(s Source) any { return s.FirstPageHeaderFooter() }))
	add(item("documentStyle.useCustomHeaderFooterMargins", 0, func(s Source) any { return s.CustomHeaderFooterMargins() }))
	add(item("documentStyle.pageNumberStart", 0, func(s Source) any { return s.PageNumberStart() }))
	for _, seg := range segments {
		kind, which, label := seg.kind, seg.which, seg.label
		name := "defaultHeaderId present"
		switch label {
		case "first-page header":
			name = "firstPageHeaderId present"
		case "first-page footer":
			name = "firstPageFooterId present"
		case "default footer":
			name = "defaultFooterId present"
		}
		add(item(name, 0, func(s Source) any { return s.HasReference(kind, which) }))
	}

	// ----------------------------------------------------- the named styles
	for _, id := range styleIDs {
		id := id
		label := docsStyleName(id)
		add(item(label+" fontSize", 0, func(s Source) any { return s.Style(id).FontSize }))
		add(item(label+" font", 0, func(s Source) any { return s.Style(id).Font }))
		add(item(label+" colour", 0, func(s Source) any { return s.Style(id).Colour }))
		add(item(label+" bold", 0, func(s Source) any { return s.Style(id).Bold }))
		add(item(label+" italic", 0, func(s Source) any { return s.Style(id).Italic }))
		add(item(label+" alignment", 0, func(s Source) any { return s.Style(id).Alignment }))
		// Line spacing is the one measurement with no tolerance: it is a
		// multiple rather than a length, so a difference is a different look
		// rather than a rounding.
		add(item(label+" lineSpacing", exactTol, func(s Source) any { return s.Style(id).LineSpacing }))
		add(item(label+" spaceAbove", 0, func(s Source) any { return s.Style(id).SpaceAbove }))
		add(item(label+" spaceBelow", 0, func(s Source) any { return s.Style(id).SpaceBelow }))
		add(item(label+" indentStart", 0, func(s Source) any { return s.Style(id).IndentStart }))
		add(item(label+" keepWithNext", 0, func(s Source) any { return s.Style(id).KeepWithNext }))
	}

	// ------------------------------------------------ the headers and footers
	for _, seg := range segments {
		kind, which, label := seg.kind, seg.which, seg.label
		add(item(label+" exists", 0, func(s Source) any { return s.Segment(kind, which).Exists }))
		add(item(label+" paragraphs", 0, func(s Source) any { return float64(s.Segment(kind, which).Paragraphs) }))
		add(item(label+" text", 0, func(s Source) any { return s.Segment(kind, which).Text }))
	}
	add(item("running head colour", 0, func(s Source) any { return s.Segment("header", "default").FirstRun.Colour }))
	add(item("running head size", 0, func(s Source) any { return s.Segment("header", "default").FirstRun.FontSize }))

	// ----------------------------------------------------------- the logo
	add(item("logo present", 0, func(s Source) any { return s.Logo().Present }))
	add(item("logo mode", 0, func(s Source) any { return s.Logo().Mode }))
	add(item("logo layout", 0, func(s Source) any { return s.Logo().Layout }))
	add(item("logo width (pt)", 0, func(s Source) any { return s.Logo().Width }))
	add(item("logo height (pt)", 0, func(s Source) any { return s.Logo().Height }))
	add(item("logo leftOffset (pt)", offsetTol, func(s Source) any { return s.Logo().Left }))
	add(item("logo topOffset (pt)", offsetTol, func(s Source) any { return s.Logo().Top }))

	// ------------------------------------------------------------ the TOC
	add(item("live tableOfContents element", 0, func(s Source) any { return float64(len(s.TOCFields())) }))
	add(item("tableOfContents instruction", 0, func(s Source) any { return s.TOCInstr() }))

	// --------------------------------------------------------- the tables
	add(item("table count", 0, func(s Source) any { return float64(len(s.Tables())) }))
	for i, name := range tableNames {
		i, name := i, name
		add(item(fmt.Sprintf("table %s size", name), 0, func(s Source) any {
			t, ok := nthTable(s, i)
			if !ok {
				return nil
			}
			return fmt.Sprintf("%dx%d", t.Rows, t.Columns)
		}))
		add(item(fmt.Sprintf("table %s column widths", name), widthTol, func(s Source) any {
			t, ok := nthTable(s, i)
			if !ok {
				return nil
			}
			return t.ColWidths
		}))
		add(item(fmt.Sprintf("table %s cell fills", name), 0, func(s Source) any {
			t, ok := nthTable(s, i)
			if !ok {
				return nil
			}
			return joined(t.Fills)
		}))
		add(item(fmt.Sprintf("table %s cell text", name), 0, func(s Source) any {
			t, ok := nthTable(s, i)
			if !ok {
				return nil
			}
			return joined(t.Texts)
		}))
		add(item(fmt.Sprintf("table %s cell borders", name), 0, func(s Source) any {
			t, ok := nthTable(s, i)
			if !ok {
				return nil
			}
			return joined(t.Borders)
		}))
		add(item(fmt.Sprintf("table %s row heights", name), widthTol, func(s Source) any {
			t, ok := nthTable(s, i)
			if !ok {
				return nil
			}
			return t.RowHeights
		}))
	}

	// ----------------------------------------------------------- the body
	for _, level := range []int{1, 2, 3} {
		level := level
		label := fmt.Sprintf("body H%d", level)
		add(item(label+" text", 0, func(s Source) any {
			h := s.FirstHeading(level)
			if !h.Found {
				return nil
			}
			return h.Text
		}))
		add(item(label+" indentStart", 0, func(s Source) any { return s.FirstHeading(level).IndentStart }))
		add(item(label+" run colour", 0, func(s Source) any { return s.FirstHeading(level).Colour }))
	}
	add(item("body text size", 0, func(s Source) any { return s.BodyParagraph().FontSize }))
	add(item("body alignment", 0, func(s Source) any { return s.BodyParagraph().Alignment }))
	add(item("body font", 0, func(s Source) any { return s.BodyParagraph().Font }))
	add(item("bulleted paragraphs", 0, func(s Source) any { return s.Bullets() }))

	return items
}

// nthTable is the table at that position, and whether the document has one
// there. A document with fewer tables answers nil for every row of the one it
// does not have, which reads as MISSING rather than as a table of no rows.
func nthTable(s Source, i int) (TableValues, bool) {
	tables := s.Tables()
	if i >= len(tables) {
		return TableValues{}, false
	}
	return tables[i], true
}

// joined puts a list of cell values into one comparable string. The separator
// is a character no cell carries, so two cells cannot be read as one.
func joined(values []string) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += "\x1f"
		}
		out += v
	}
	return out
}

// FromDocx reads every item out of a docx.
func FromDocx(d *Docx) []Value {
	out := make([]Value, 0, len(Items))
	for _, it := range Items {
		out = append(out, Value{Name: it.Name, Tol: it.Tol, V: it.Docx(d)})
	}
	return out
}

// FromDoc reads every item out of a Docs API answer.
func FromDoc(d *Doc) []Value {
	out := make([]Value, 0, len(Items))
	for _, it := range Items {
		out = append(out, Value{Name: it.Name, Tol: it.Tol, V: it.Doc(d)})
	}
	return out
}
