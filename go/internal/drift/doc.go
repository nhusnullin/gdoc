// The Docs half of the measurement: the same values, read out of what the API
// answered.
//
// This is the reading that means something, and the reason is Google's import.
// A docx is a statement of intent; what a person opens is what Docs made of it.
// So the live gate uploads both files with conversion, reads each back through
// documents.get, and runs this side of the same item list.
//
// It is a port of compare.py, field for field, and the field names are Google's
// rather than Word's. Where the docx half has to resolve a value through
// docDefaults and a basedOn chain, this half reads it as Docs already resolved
// it, which is why the two halves of one item can report different spellings of
// the same fact and each gate stays self-consistent.
//
// No client and no session: this takes the decoded answer, so the live test
// hands it what came back and a unit test hands it a fixture.
package drift

import (
	"fmt"
	"strings"
)

// Doc is one Docs API answer, opened.
type Doc struct {
	answer map[string]any
}

// OpenDoc wraps a decoded documents.get answer.
func OpenDoc(answer map[string]any) *Doc { return &Doc{answer: answer} }

// style is the document's own style block.
func (d *Doc) style() map[string]any { return mapOf(d.answer["documentStyle"]) }

// Page is one length off documentStyle, in points. The names are documentStyle's
// own, which is why the docx half spells its section properties under them.
func (d *Doc) Page(name string) any {
	s := d.style()
	switch name {
	case "width", "height":
		return magnitude(mapOf(s["pageSize"])[name])
	}
	return magnitude(s[name])
}

// FirstPageHeaderFooter is Docs' useFirstPageHeaderFooter.
func (d *Doc) FirstPageHeaderFooter() any { return boolOf(d.style()["useFirstPageHeaderFooter"]) }

// CustomHeaderFooterMargins is Docs' useCustomHeaderFooterMargins.
func (d *Doc) CustomHeaderFooterMargins() any {
	return boolOf(d.style()["useCustomHeaderFooterMargins"])
}

// PageNumberStart is where the footer's page numbers begin.
func (d *Doc) PageNumberStart() any {
	v, ok := d.style()["pageNumberStart"].(float64)
	if !ok {
		return nil
	}
	return v
}

// segmentKey is the documentStyle field naming one header or footer. The four
// are spelled out rather than assembled: a field name is Google's word, not a
// string this package builds and hopes matches.
func segmentKey(kind, which string) string {
	switch kind + "/" + which {
	case "header/first":
		return "firstPageHeaderId"
	case "header/default":
		return "defaultHeaderId"
	case "footer/first":
		return "firstPageFooterId"
	case "footer/default":
		return "defaultFooterId"
	}
	return ""
}

// HasReference says whether the document names a header or footer of that kind.
func (d *Doc) HasReference(kind, which string) any {
	id, _ := d.style()[segmentKey(kind, which)].(string)
	return id != ""
}

// Style reads one named style as Docs resolved it.
func (d *Doc) Style(id string) Style {
	for _, raw := range listOf(mapOf(d.answer["namedStyles"])["styles"]) {
		s := mapOf(raw)
		if s["namedStyleType"] != docsStyleName(id) {
			continue
		}
		ts, ps := mapOf(s["textStyle"]), mapOf(s["paragraphStyle"])
		return Style{
			found:         true,
			FontSize:      magnitude(ts["fontSize"]),
			Font:          stringOf(mapOf(ts["weightedFontFamily"])["fontFamily"]),
			Colour:        hexOf(ts["foregroundColor"]),
			Bold:          truth(ts["bold"]),
			Italic:        truth(ts["italic"]),
			Underline:     truth(ts["underline"]),
			SmallCaps:     truth(ts["smallCaps"]),
			Strikethrough: truth(ts["strikethrough"]),
			Alignment:     stringOf(ps["alignment"]),
			LineSpacing:   plain(ps["lineSpacing"]),
			SpaceAbove:    magnitude(ps["spaceAbove"]),
			SpaceBelow:    magnitude(ps["spaceBelow"]),
			IndentStart:   magnitude(ps["indentStart"]),
			KeepWithNext:  truth(ps["keepWithNext"]),
		}
	}
	return Style{}
}

// docsStyleName maps the docx style id onto the name Docs gives the same style,
// so one item names one style whichever half reads it.
func docsStyleName(id string) string {
	switch id {
	case "Normal":
		return "NORMAL_TEXT"
	case "Title":
		return "TITLE"
	case "Subtitle":
		return "SUBTITLE"
	}
	if strings.HasPrefix(id, "Heading") {
		return "HEADING_" + strings.TrimPrefix(id, "Heading")
	}
	return id
}

// segment finds one header or footer by the id documentStyle names.
func (d *Doc) segment(kind, which string) map[string]any {
	id, _ := d.style()[segmentKey(kind, which)].(string)
	if id == "" {
		return nil
	}
	return mapOf(mapOf(d.answer[kind+"s"])[id])
}

// Segment reads one header or footer.
func (d *Doc) Segment(kind, which string) Segment {
	seg := d.segment(kind, which)
	if seg == nil {
		return Segment{}
	}
	out := Segment{Exists: true}
	var b strings.Builder
	for _, raw := range listOf(seg["content"]) {
		p := mapOf(mapOf(raw)["paragraph"])
		if p == nil {
			continue
		}
		out.Paragraphs++
		for _, e := range listOf(p["elements"]) {
			b.WriteString(docsElementText(mapOf(e)))
		}
	}
	out.Text = b.String()
	out.FirstRun = firstRunStyle(seg)
	return out
}

// docsElementText is one paragraph element as words. The three that are not
// words are spelled out, which is what the docx half does with the same three.
func docsElementText(e map[string]any) string {
	if tr := mapOf(e["textRun"]); tr != nil {
		s, _ := tr["content"].(string)
		return s
	}
	if at := mapOf(e["autoText"]); at != nil {
		t, _ := at["type"].(string)
		return "<" + t + ">"
	}
	if e["horizontalRule"] != nil {
		return "<HR>"
	}
	if e["inlineObjectElement"] != nil {
		return "<INLINE_IMAGE>"
	}
	return ""
}

// firstRunStyle is the style of the first run with words in it, which is what
// the running head's colour and size are read from.
func firstRunStyle(seg map[string]any) Style {
	for _, raw := range listOf(seg["content"]) {
		for _, e := range listOf(mapOf(mapOf(raw)["paragraph"])["elements"]) {
			tr := mapOf(mapOf(e)["textRun"])
			content, _ := tr["content"].(string)
			if strings.TrimSpace(content) == "" {
				continue
			}
			ts := mapOf(tr["textStyle"])
			return Style{found: true, Colour: hexOf(ts["foregroundColor"]), FontSize: magnitude(ts["fontSize"])}
		}
	}
	return Style{}
}

// Logo reads the picture in the first page header, positioned if it is one and
// inline if it is not. The positioned case is the one the whole arrangement
// stands on: the Docs API cannot create one, and a docx import can.
func (d *Doc) Logo() Logo {
	seg := d.segment("header", "first")
	if seg == nil {
		return Logo{}
	}
	for _, raw := range listOf(seg["content"]) {
		p := mapOf(mapOf(raw)["paragraph"])
		for _, id := range listOf(p["positionedObjectIds"]) {
			key, _ := id.(string)
			o := mapOf(mapOf(d.answer["positionedObjects"])[key])
			props := mapOf(o["positionedObjectProperties"])
			size := mapOf(mapOf(props["embeddedObject"])["size"])
			pos := mapOf(props["positioning"])
			return Logo{
				Present: true, Mode: "POSITIONED",
				Layout: stringOrEmpty(pos["layout"]),
				Width:  magnitude(size["width"]), Height: magnitude(size["height"]),
				Left: magnitude(pos["leftOffset"]), Top: magnitude(pos["topOffset"]),
			}
		}
		for _, e := range listOf(p["elements"]) {
			ref := mapOf(mapOf(e)["inlineObjectElement"])
			if ref == nil {
				continue
			}
			key, _ := ref["inlineObjectId"].(string)
			o := mapOf(mapOf(d.answer["inlineObjects"])[key])
			size := mapOf(mapOf(mapOf(o["inlineObjectProperties"])["embeddedObject"])["size"])
			return Logo{Present: true, Mode: "INLINE",
				Width: magnitude(size["width"]), Height: magnitude(size["height"])}
		}
	}
	return Logo{}
}

// TOCFields is every contents element in the body.
//
// Docs reports a contents list as a structural element and never as the field
// instruction behind it, so what comes back here is one empty entry per list.
// The count is the fact both halves share; the instruction item reads nil on
// this side and says so rather than inventing a string to compare.
func (d *Doc) TOCFields() []string {
	out := []string{}
	for _, raw := range listOf(mapOf(d.answer["body"])["content"]) {
		if mapOf(raw)["tableOfContents"] != nil {
			out = append(out, "")
		}
	}
	return out
}

// TOCInstr is nil on this side: see TOCFields.
func (d *Doc) TOCInstr() any { return nil }

// Tables reads the body's tables in the order they appear.
func (d *Doc) Tables() []TableValues {
	out := []TableValues{}
	for _, raw := range listOf(mapOf(d.answer["body"])["content"]) {
		t := mapOf(mapOf(raw)["table"])
		if t == nil {
			continue
		}
		table := TableValues{ColWidths: []float64{}, RowHeights: []float64{}, Fills: []string{}, Texts: []string{}, Borders: []string{}}
		if n, ok := t["rows"].(float64); ok {
			table.Rows = int(n)
		}
		if n, ok := t["columns"].(float64); ok {
			table.Columns = int(n)
		}
		for _, col := range listOf(mapOf(t["tableStyle"])["tableColumnProperties"]) {
			if w, ok := magnitude(mapOf(col)["width"]).(float64); ok {
				table.ColWidths = append(table.ColWidths, w)
			}
		}
		for _, rowRaw := range listOf(t["tableRows"]) {
			row := mapOf(rowRaw)
			h := 0.0
			if n, ok := magnitude(mapOf(row["tableRowStyle"])["minRowHeight"]).(float64); ok {
				h = n
			}
			table.RowHeights = append(table.RowHeights, h)
			for _, cellRaw := range listOf(row["tableCells"]) {
				cell := mapOf(cellRaw)
				cs := mapOf(cell["tableCellStyle"])
				table.Fills = append(table.Fills, stringOrEmpty(hexOf(cs["backgroundColor"])))
				table.Texts = append(table.Texts, strings.TrimSpace(docsCellText(cell)))
				table.Borders = append(table.Borders, docsBorder(cs))
			}
		}
		out = append(out, table)
	}
	return out
}

// docsBorder is the border a reader sees on one cell: the first edge the cell
// declares, as width and colour. Google omits an interior edge when the
// neighbouring cell declares it, which is compare.py's own note.
func docsBorder(cs map[string]any) string {
	for _, edge := range []string{"borderTop", "borderBottom", "borderLeft", "borderRight"} {
		b := mapOf(cs[edge])
		if b == nil {
			continue
		}
		width := 0.0
		if n, ok := magnitude(b["width"]).(float64); ok {
			width = n
		}
		return fmt.Sprintf("%.3fpt %s", width, stringOrEmpty(hexOf(b["color"])))
	}
	return ""
}

// docsCellText is every word in one cell.
func docsCellText(cell map[string]any) string {
	var b strings.Builder
	for _, raw := range listOf(cell["content"]) {
		for _, e := range listOf(mapOf(mapOf(raw)["paragraph"])["elements"]) {
			b.WriteString(docsElementText(mapOf(e)))
		}
	}
	return b.String()
}

// FirstHeading finds the first paragraph in the body at that heading level.
func (d *Doc) FirstHeading(level int) Heading {
	want := fmt.Sprintf("HEADING_%d", level)
	for _, raw := range listOf(mapOf(d.answer["body"])["content"]) {
		p := mapOf(mapOf(raw)["paragraph"])
		ps := mapOf(p["paragraphStyle"])
		if ps["namedStyleType"] != want {
			continue
		}
		var b strings.Builder
		var colour any
		for _, e := range listOf(p["elements"]) {
			tr := mapOf(mapOf(e)["textRun"])
			content, _ := tr["content"].(string)
			b.WriteString(content)
			if colour == nil && strings.TrimSpace(content) != "" {
				colour = hexOf(mapOf(tr["textStyle"])["foregroundColor"])
			}
		}
		return Heading{Found: true, Text: strings.TrimSpace(b.String()),
			IndentStart: magnitude(ps["indentStart"]), Colour: colour}
	}
	return Heading{}
}

// BodyParagraph is the first real paragraph of prose after the first heading.
func (d *Doc) BodyParagraph() BodyParagraph {
	seen := false
	for _, raw := range listOf(mapOf(d.answer["body"])["content"]) {
		p := mapOf(mapOf(raw)["paragraph"])
		if p == nil {
			continue
		}
		ps := mapOf(p["paragraphStyle"])
		if ps["namedStyleType"] == "HEADING_1" {
			seen = true
			continue
		}
		if !seen || p["bullet"] != nil {
			continue
		}
		var b strings.Builder
		var ts map[string]any
		for _, e := range listOf(p["elements"]) {
			tr := mapOf(mapOf(e)["textRun"])
			content, _ := tr["content"].(string)
			b.WriteString(content)
			if ts == nil && strings.TrimSpace(content) != "" {
				ts = mapOf(tr["textStyle"])
			}
		}
		if len(strings.TrimSpace(b.String())) <= bodyProseMin {
			continue
		}
		return BodyParagraph{Found: true,
			FontSize:  magnitude(ts["fontSize"]),
			Alignment: stringOf(ps["alignment"]),
			Font:      stringOf(mapOf(ts["weightedFontFamily"])["fontFamily"]),
		}
	}
	return BodyParagraph{}
}

// Bullets counts the body paragraphs carrying a list bullet.
func (d *Doc) Bullets() any {
	n := 0
	for _, raw := range listOf(mapOf(d.answer["body"])["content"]) {
		if mapOf(mapOf(raw)["paragraph"])["bullet"] != nil {
			n++
		}
	}
	return float64(n)
}

// --------------------------------------------------------- reading the JSON

// mapOf reads one object out of a decoded answer. Anything else is an empty
// object rather than a panic: this walks a shape Google documents and does not
// promise, so a field that is not there is a field this document does not have.
func mapOf(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// listOf reads one array.
func listOf(v any) []any {
	l, _ := v.([]any)
	return l
}

// magnitude reads a Docs dimension, which is an object carrying a magnitude and
// a unit. Points are the only unit the API uses for these.
func magnitude(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	n, ok := m["magnitude"].(float64)
	if !ok {
		return nil
	}
	return round3(n)
}

// plain reads a bare number.
func plain(v any) any {
	n, ok := v.(float64)
	if !ok {
		return nil
	}
	return n
}

// stringOf reads a string, and nil when there is none.
func stringOf(v any) any {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	return s
}

// stringOrEmpty is stringOf where an absent value is an empty string, which is
// what a joined list of cell fills wants.
func stringOrEmpty(v any) string {
	s, _ := v.(string)
	return s
}

// truth reads a Docs boolean, where absent means false.
func truth(v any) bool {
	b, _ := v.(bool)
	return b
}

// boolOf reads a Docs boolean as a value, where absent is still false: the API
// leaves a false flag out, and reporting nil would read as a document that does
// not have the flag at all.
func boolOf(v any) any {
	b, _ := v.(bool)
	return b
}

// hexOf turns a Docs colour into the spelling the docx half writes. Each
// channel is a fraction of one, and a channel that is not there is zero, which
// is the API's own way of writing black.
func hexOf(v any) any {
	rgb := mapOf(mapOf(mapOf(v)["color"])["rgbColor"])
	if rgb == nil {
		return nil
	}
	channel := func(name string) int {
		f, _ := rgb[name].(float64)
		return int(f*255 + 0.5)
	}
	return fmt.Sprintf("#%02X%02X%02X", channel("red"), channel("green"), channel("blue"))
}
