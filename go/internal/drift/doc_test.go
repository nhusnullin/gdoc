package drift

import (
	"encoding/json"
	"testing"
)

// docsAnswer is one documents.get answer, hand written to the shape Google
// documents. It is small on purpose: what it has to prove is that this half of
// every item reads the field it names, and a whole real answer would prove the
// same thing while nobody could see which field a failing row came from.
//
// The values are the house style's, so a row read here reads the same number
// the docx half reads out of the master.
const docsAnswer = `{
 "documentStyle": {
  "pageSize": {"width": {"magnitude": 595.3, "unit": "PT"}, "height": {"magnitude": 841.9, "unit": "PT"}},
  "marginTop": {"magnitude": 62.35, "unit": "PT"},
  "marginBottom": {"magnitude": 51, "unit": "PT"},
  "marginLeft": {"magnitude": 51.05, "unit": "PT"},
  "marginRight": {"magnitude": 51.05, "unit": "PT"},
  "marginHeader": {"magnitude": 14.15, "unit": "PT"},
  "marginFooter": {"magnitude": 0, "unit": "PT"},
  "useFirstPageHeaderFooter": true,
  "useCustomHeaderFooterMargins": true,
  "pageNumberStart": 1,
  "firstPageHeaderId": "h.first",
  "defaultHeaderId": "h.default",
  "firstPageFooterId": "f.first",
  "defaultFooterId": "f.default"
 },
 "namedStyles": {"styles": [
  {"namedStyleType": "NORMAL_TEXT",
   "textStyle": {"fontSize": {"magnitude": 11, "unit": "PT"},
                 "weightedFontFamily": {"fontFamily": "Calibri"},
                 "foregroundColor": {"color": {"rgbColor": {}}}},
   "paragraphStyle": {"lineSpacing": 115,
                      "spaceAbove": {"magnitude": 3, "unit": "PT"},
                      "spaceBelow": {"magnitude": 6, "unit": "PT"}}},
  {"namedStyleType": "HEADING_1",
   "textStyle": {"fontSize": {"magnitude": 16, "unit": "PT"}, "bold": true,
                 "foregroundColor": {"color": {"rgbColor": {"red": 0.13333334, "green": 0.14901961, "blue": 0.37254903}}}},
   "paragraphStyle": {"keepWithNext": true, "spaceAbove": {"magnitude": 12, "unit": "PT"},
                      "indentStart": {"magnitude": 0, "unit": "PT"}}},
  {"namedStyleType": "TITLE",
   "textStyle": {"fontSize": {"magnitude": 16, "unit": "PT"}, "bold": true,
                 "weightedFontFamily": {"fontFamily": "Arial"}},
   "paragraphStyle": {"alignment": "CENTER"}}
 ]},
 "headers": {
  "h.default": {"content": [
   {"paragraph": {"elements": [
     {"textRun": {"content": "Altery - Supplier Management Policy",
                  "textStyle": {"fontSize": {"magnitude": 12, "unit": "PT"},
                                "foregroundColor": {"color": {"rgbColor": {"red": 1, "green": 0.4666667, "blue": 0.10980392}}}}}}]}}]},
  "h.first": {"content": [
   {"paragraph": {"positionedObjectIds": ["kix.logo"],
                  "elements": [{"textRun": {"content": "For internal use only "}}]}}]}
 },
 "footers": {
  "f.default": {"content": [
   {"paragraph": {"elements": [
     {"textRun": {"content": "For internal use only "}},
     {"autoText": {"type": "PAGE_NUMBER"}}]}}]},
  "f.first": {"content": [{"paragraph": {"elements": [{"autoText": {"type": "PAGE_NUMBER"}}]}}]}
 },
 "positionedObjects": {"kix.logo": {"positionedObjectProperties": {
   "embeddedObject": {"size": {"width": {"magnitude": 146.25, "unit": "PT"},
                               "height": {"magnitude": 72.75, "unit": "PT"}}},
   "positioning": {"layout": "WRAP_TEXT",
                   "leftOffset": {"magnitude": 404.25, "unit": "PT"},
                   "topOffset": {"magnitude": -5.15, "unit": "PT"}}}}},
 "body": {"content": [
  {"tableOfContents": {"content": []}},
  {"table": {"rows": 5, "columns": 2,
    "tableStyle": {"tableColumnProperties": [
      {"width": {"magnitude": 150, "unit": "PT"}},
      {"width": {"magnitude": 359.25, "unit": "PT"}}]},
    "tableRows": [
     {"tableRowStyle": {"minRowHeight": {"magnitude": 28.5, "unit": "PT"}},
      "tableCells": [
       {"tableCellStyle": {"backgroundColor": {"color": {"rgbColor": {"red": 0.9607843, "green": 0.81960785, "blue": 0.68235296}}},
                           "borderTop": {"width": {"magnitude": 1, "unit": "PT"},
                                         "color": {"color": {"rgbColor": {}}}}},
        "content": [{"paragraph": {"elements": [{"textRun": {"content": "Document Owner"}}]}}]},
       {"tableCellStyle": {}, "content": []}]}]}},
  {"paragraph": {"paragraphStyle": {"namedStyleType": "HEADING_1", "indentStart": {"magnitude": 0, "unit": "PT"}},
    "elements": [{"textRun": {"content": "1-Third Party and Outsourcing Policy",
                              "textStyle": {"foregroundColor": {"color": {"rgbColor": {"red": 0.13333334, "green": 0.14901961, "blue": 0.37254903}}}}}}]}},
  {"paragraph": {"paragraphStyle": {"namedStyleType": "NORMAL_TEXT", "alignment": "JUSTIFIED"},
    "elements": [{"textRun": {"content": "This policy sets out how the firm selects, approves, monitors and exits third party arrangements.",
                              "textStyle": {"fontSize": {"magnitude": 12, "unit": "PT"},
                                            "weightedFontFamily": {"fontFamily": "Calibri"}}}}]}},
  {"paragraph": {"bullet": {"listId": "kix.l0"},
    "elements": [{"textRun": {"content": "The activity is one the firm could perform itself."}}]}}
 ]}
}`

// openAnswer decodes the fixture the way the live gate decodes what came back.
func openAnswer(t *testing.T) *Doc {
	t.Helper()
	var answer map[string]any
	if err := json.Unmarshal([]byte(docsAnswer), &answer); err != nil {
		t.Fatalf("the fixture is not JSON: %v", err)
	}
	return OpenDoc(answer)
}

// TestFromDocReadsTheDocsAnswer states the house values as literals again, read
// this time out of what the API sends. The numbers are the same numbers the
// docx half reads out of the master, which is the point: one item, one value,
// two ways to reach it.
func TestFromDocReadsTheDocsAnswer(t *testing.T) {
	d := openAnswer(t)

	if got := d.Page("width"); got != 595.3 {
		t.Errorf("the page is %v points wide, and A4 is 595.3", got)
	}
	if got := d.Page("marginTop"); got != 62.35 {
		t.Errorf("the top margin is %v points, and the house style is 62.35", got)
	}
	if got := d.FirstPageHeaderFooter(); got != true {
		t.Errorf("useFirstPageHeaderFooter read %v", got)
	}
	if got := d.PageNumberStart(); got != 1.0 {
		t.Errorf("the page numbers start at %v", got)
	}
	for _, c := range []struct{ kind, which string }{
		{"header", "first"}, {"header", "default"}, {"footer", "first"}, {"footer", "default"},
	} {
		if got := d.HasReference(c.kind, c.which); got != true {
			t.Errorf("the answer names no %s %s", c.which, c.kind)
		}
	}

	if got := d.Style("Normal"); got.FontSize != 11.0 || got.Font != "Calibri" || got.LineSpacing != 115.0 {
		t.Errorf("body text read %v point %v at %v%%, and the house style is 11 point Calibri at 115%%",
			got.FontSize, got.Font, got.LineSpacing)
	}
	if got := d.Style("Normal").Colour; got != "#000000" {
		t.Errorf("a colour with no channels in it is %v, and Google's way of writing black is #000000", got)
	}
	h1 := d.Style("Heading1")
	if h1.FontSize != 16.0 || !h1.Bold || !h1.KeepWithNext {
		t.Errorf("Heading 1 read %v point bold %v keeping with next %v, and the house style is 16 point bold and keeps",
			h1.FontSize, h1.Bold, h1.KeepWithNext)
	}
	if h1.Colour != "#22265F" {
		t.Errorf("Heading 1 is %v, and the house navy is #22265F", h1.Colour)
	}
	if got := d.Style("Title").Alignment; got != "CENTER" {
		t.Errorf("the Title is aligned %v, and the house style centres it", got)
	}

	// A field prints as its type, which is what the docx half prints as the
	// field's instruction. Neither prints the number cached behind it.
	if got := d.Segment("footer", "default").Text; got != "For internal use only <PAGE_NUMBER>" {
		t.Errorf("the footer reads %q", got)
	}
	if got := d.Segment("header", "default").FirstRun.Colour; got != "#FF771C" {
		t.Errorf("the running head is %v, and the house orange is #FF771C", got)
	}
	if got := d.Segment("header", "default").FirstRun.FontSize; got != 12.0 {
		t.Errorf("the running head is %v points, and the house style is 12", got)
	}

	logo := d.Logo()
	if !logo.Present || logo.Mode != "POSITIONED" || logo.Layout != "WRAP_TEXT" {
		t.Fatalf("the logo read present %v mode %q layout %q", logo.Present, logo.Mode, logo.Layout)
	}
	if logo.Width != 146.25 || logo.Height != 72.75 || logo.Left != 404.25 {
		t.Errorf("the logo is %v by %v points at %v from the left, and the house style is 146.25 by 72.75 at 404.25",
			logo.Width, logo.Height, logo.Left)
	}

	if got := d.TOCFields(); len(got) != 1 {
		t.Errorf("the answer carries %d contents lists, and it carries one", len(got))
	}
	if got := d.TOCInstr(); got != nil {
		t.Errorf("the Docs answer reports no field instruction, and this side read %v", got)
	}

	tables := d.Tables()
	if len(tables) != 1 {
		t.Fatalf("the answer carries %d tables", len(tables))
	}
	if tables[0].Rows != 5 || tables[0].Columns != 2 {
		t.Errorf("the table is %dx%d, and it is 5x2", tables[0].Rows, tables[0].Columns)
	}
	if got := tables[0].ColWidths; len(got) != 2 || got[0] != 150 {
		t.Errorf("the columns are %v points", got)
	}
	if got := tables[0].Fills[0]; got != "#F5D1AE" {
		t.Errorf("the first cell is filled %v, and the house fill is #F5D1AE", got)
	}
	if got := tables[0].Fills[1]; got != "" {
		t.Errorf("a cell with no fill read %q, and it has none", got)
	}
	if got := tables[0].Texts[0]; got != "Document Owner" {
		t.Errorf("the first cell reads %q", got)
	}
	if got := tables[0].Borders[0]; got != "1.000pt #000000" {
		t.Errorf("the first cell's border reads %q, and it is one point black", got)
	}

	h := d.FirstHeading(1)
	if !h.Found || h.Text != "1-Third Party and Outsourcing Policy" || h.Colour != "#22265F" {
		t.Errorf("the first heading read %q in %v", h.Text, h.Colour)
	}
	if got := d.FirstHeading(3); got.Found {
		t.Errorf("the answer carries no level-three heading and one was found: %q", got.Text)
	}
	if got := d.BodyParagraph(); !got.Found || got.FontSize != 12.0 || got.Alignment != "JUSTIFIED" || got.Font != "Calibri" {
		t.Errorf("the body paragraph read %v point %v aligned %v", got.FontSize, got.Font, got.Alignment)
	}
	if got := d.Bullets(); got != 1.0 {
		t.Errorf("the answer carries %v list items, and it carries one", got)
	}
}

// TestFromDocReadsEveryItem is the shape check the live gate stands on: the
// Docs half answers the whole list, in the same order and under the same names
// as the docx half, so the two gates report rows that line up.
func TestFromDocReadsEveryItem(t *testing.T) {
	values := FromDoc(openAnswer(t))
	if len(values) != len(Items) {
		t.Fatalf("%d items read as %d values", len(Items), len(values))
	}
	for i, v := range values {
		if v.Name != Items[i].Name {
			t.Fatalf("value %d is called %q and its item is %q", i, v.Name, Items[i].Name)
		}
	}
	rows := Compare(values, FromDoc(openAnswer(t)))
	for _, r := range rows {
		if r.Verdict != Identical {
			t.Errorf("one answer read against itself gave %s on %q", r.Verdict, r.Item)
		}
	}
}

// TestAnEmptyAnswerReadsAsNothingRatherThanPanicking. The live gate reads
// whatever Google sends, and a shape this package did not expect must come back
// as a document that says nothing rather than as a crash inside a test.
func TestAnEmptyAnswerReadsAsNothingRatherThanPanicking(t *testing.T) {
	d := OpenDoc(map[string]any{})
	for _, v := range FromDoc(d) {
		switch got := v.V.(type) {
		case nil, bool, string:
		case float64:
			if got != 0 {
				t.Errorf("item %q read %v out of an empty answer", v.Name, got)
			}
		case []float64:
			if len(got) != 0 {
				t.Errorf("item %q read %v out of an empty answer", v.Name, got)
			}
		default:
			t.Errorf("item %q read %T out of an empty answer", v.Name, v.V)
		}
	}
}
