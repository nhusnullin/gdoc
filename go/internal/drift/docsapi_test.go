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
//
// The two body paragraphs state nothing on their runs, which is what a
// converted document looks like: Google leaves an inherited property out of the
// message, so the heading's colour and the prose's size and font are only
// readable through the chain. HEADING_1 states no font and no line spacing for
// the same reason, one level up.
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
    "elements": [{"textRun": {"content": "1-Third Party and Outsourcing Policy", "textStyle": {}}}]}},
  {"paragraph": {"paragraphStyle": {"namedStyleType": "NORMAL_TEXT", "alignment": "JUSTIFIED"},
    "elements": [{"textRun": {"content": "This policy sets out how the firm selects, approves, monitors and exits third party arrangements.",
                              "textStyle": {}}}]}},
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
	if got := d.CustomHeaderFooterMargins(); got != true {
		t.Errorf("useCustomHeaderFooterMargins read %v, and the house style sets both distances", got)
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
	if got := d.BodyParagraph(); !got.Found || got.FontSize != 11.0 || got.Alignment != "JUSTIFIED" || got.Font != "Calibri" {
		t.Errorf("the body paragraph read %v point %v aligned %v, and the house body is 11 point Calibri",
			got.FontSize, got.Font, got.Alignment)
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

// inheritedAnswer is the case the main fixture cannot show through a public
// reader: what a run keeps when it states almost nothing.
//
// The header's paragraph names HEADING_1 and its run states one size. A reader
// sees 20 point bold navy Calibri: the size is the run's, the weight and the
// colour are the heading's, and the font falls all the way to NORMAL_TEXT. The
// footer is the other end, a paragraph naming no style at all, which is
// NORMAL_TEXT by the API's own rule.
const inheritedAnswer = `{
 "documentStyle": {"defaultHeaderId": "h.default", "defaultFooterId": "f.default"},
 "namedStyles": {"styles": [
  {"namedStyleType": "NORMAL_TEXT",
   "textStyle": {"fontSize": {"magnitude": 11, "unit": "PT"},
                 "weightedFontFamily": {"fontFamily": "Calibri"},
                 "foregroundColor": {"color": {"rgbColor": {}}}},
   "paragraphStyle": {"lineSpacing": 115}},
  {"namedStyleType": "HEADING_1",
   "textStyle": {"fontSize": {"magnitude": 16, "unit": "PT"}, "bold": true,
                 "foregroundColor": {"color": {"rgbColor": {"red": 0.13333334, "green": 0.14901961, "blue": 0.37254903}}}},
   "paragraphStyle": {"keepWithNext": true}}
 ]},
 "headers": {"h.default": {"content": [
  {"paragraph": {"paragraphStyle": {"namedStyleType": "HEADING_1"},
    "elements": [{"textRun": {"content": "Third Party Policy",
                              "textStyle": {"fontSize": {"magnitude": 20, "unit": "PT"}}}}]}}]}},
 "footers": {"f.default": {"content": [
  {"paragraph": {"elements": [{"textRun": {"content": "For internal use only"}}]}}]}}
}`

// TestARunThatStatesNothingReadsTheStyleBehindIt is the resolution chain, in
// both directions: what the run states wins, and what it leaves out is
// inherited rather than read as nil.
//
// The Docs API leaves an inherited property out of the message altogether, so a
// reader that took textRun.textStyle alone reported nil for every run of a
// converted document, and against a docx half that resolves that is a DIFFERENT
// row nobody wants, while against another Docs answer it is an IDENTICAL row
// measuring nothing.
func TestARunThatStatesNothingReadsTheStyleBehindIt(t *testing.T) {
	var answer map[string]any
	if err := json.Unmarshal([]byte(inheritedAnswer), &answer); err != nil {
		t.Fatalf("the fixture is not JSON: %v", err)
	}
	d := OpenDoc(answer)

	run := d.Segment("header", "default").FirstRun
	if run.FontSize != 20.0 {
		t.Errorf("the run reads %v point, and the run itself states 20", run.FontSize)
	}
	if !run.Bold {
		t.Error("the run reads not bold, and HEADING_1 is bold: an absent bold on a run is inheritance, not a false")
	}
	if run.Colour != "#22265F" {
		t.Errorf("the run reads %v, and HEADING_1 is the house navy #22265F", run.Colour)
	}
	if run.Font != "Calibri" {
		t.Errorf("the run reads %v, and neither it nor HEADING_1 states a font, so it is NORMAL_TEXT's Calibri", run.Font)
	}
	if run.LineSpacing != 115.0 {
		t.Errorf("the run's paragraph reads %v%%, and NORMAL_TEXT is 115%%", run.LineSpacing)
	}
	if !run.KeepWithNext {
		t.Error("the run's paragraph does not keep with the next, and HEADING_1 does")
	}

	plain := d.Segment("footer", "default").FirstRun
	if plain.FontSize != 11.0 || plain.Font != "Calibri" || plain.Colour != "#000000" {
		t.Errorf("a paragraph naming no style read %v point %v in %v, and it is NORMAL_TEXT's 11 point Calibri in black",
			plain.FontSize, plain.Font, plain.Colour)
	}
	if plain.Bold {
		t.Error("a paragraph naming no style read bold, and NORMAL_TEXT is not")
	}
}

// TestANamedStyleInheritsFromNormalTextAndAnAbsentOneIsNotFound. A named style
// inherits from NORMAL_TEXT, which is the last link of the same chain. A style
// the answer does not carry at all is a different thing: it is not found, the
// way the docx half answers for a style that is in no file, because reporting
// the inherited values under its name would invent a style Docs never sent.
func TestANamedStyleInheritsFromNormalTextAndAnAbsentOneIsNotFound(t *testing.T) {
	d := openAnswer(t)

	h1 := d.Style("Heading1")
	if h1.Font != "Calibri" {
		t.Errorf("Heading 1 reads %v, and it states no font, so it is NORMAL_TEXT's Calibri", h1.Font)
	}
	if h1.LineSpacing != 115.0 {
		t.Errorf("Heading 1 reads %v%%, and it states no line spacing, so it is NORMAL_TEXT's 115%%", h1.LineSpacing)
	}
	if h1.SpaceAbove != 12.0 || h1.SpaceBelow != 6.0 {
		t.Errorf("Heading 1 reads %v above and %v below, and it states 12 above while inheriting 6 below",
			h1.SpaceAbove, h1.SpaceBelow)
	}

	absent := d.Style("Heading4")
	if absent.FontSize != nil || absent.Font != nil || absent.Colour != nil {
		t.Errorf("the answer carries no HEADING_4 and it read %v point %v in %v",
			absent.FontSize, absent.Font, absent.Colour)
	}
}

// TestTheHeadingsColourComesFromTheStyleWhenTheRunIsSilent. The heading rows
// are the ones the M5 review found: a converted document's heading run states
// no colour, so this row read nil on the Docs side while the docx half read the
// navy off the style.
func TestTheHeadingsColourComesFromTheStyleWhenTheRunIsSilent(t *testing.T) {
	h := openAnswer(t).FirstHeading(1)
	if !h.Found || h.Colour != "#22265F" {
		t.Errorf("the first heading read %v, and its run states no colour, so it is HEADING_1's navy #22265F", h.Colour)
	}
}

// docsSilent is every row this fixture is not expected to answer, written out.
//
// It is the pin `unstated` is for the docx half, and it exists for the same
// reason: a row that is nil on both sides compares as IDENTICAL and measures
// nothing, so a reader that quietly stopped resolving a value would pass this
// package's own tests. TestFromDocReadsEveryItem cannot see it, because it
// compares one answer against itself.
//
// This is the Docs half alone. The offline gate reads two docx files, so its
// 169 rows do not move and nothing here is re-baselined.
//
// Four reasons, and each is a fact about the fixture rather than about the
// reader:
//
//   - a value the style leaves to the reader's default, which is the same
//     reason `unstated` gives: a style stating no justification is left
//     aligned, and one stating no indent starts at the margin;
//   - the six named styles the fixture does not carry, which are not found, so
//     every value on them is nil rather than NORMAL_TEXT's inherited one;
//   - the two front matter tables and the two heading levels the fixture does
//     not carry;
//   - the contents field instruction, which Docs never sends: see TOCInstr.
var docsSilent = []string{
	"NORMAL_TEXT alignment", "NORMAL_TEXT indentStart",
	"HEADING_1 alignment", "TITLE indentStart",

	"HEADING_2 fontSize", "HEADING_2 font", "HEADING_2 colour", "HEADING_2 alignment",
	"HEADING_2 lineSpacing", "HEADING_2 spaceAbove", "HEADING_2 spaceBelow", "HEADING_2 indentStart",
	"HEADING_3 fontSize", "HEADING_3 font", "HEADING_3 colour", "HEADING_3 alignment",
	"HEADING_3 lineSpacing", "HEADING_3 spaceAbove", "HEADING_3 spaceBelow", "HEADING_3 indentStart",
	"HEADING_4 fontSize", "HEADING_4 font", "HEADING_4 colour", "HEADING_4 alignment",
	"HEADING_4 lineSpacing", "HEADING_4 spaceAbove", "HEADING_4 spaceBelow", "HEADING_4 indentStart",
	"HEADING_5 fontSize", "HEADING_5 font", "HEADING_5 colour", "HEADING_5 alignment",
	"HEADING_5 lineSpacing", "HEADING_5 spaceAbove", "HEADING_5 spaceBelow", "HEADING_5 indentStart",
	"HEADING_6 fontSize", "HEADING_6 font", "HEADING_6 colour", "HEADING_6 alignment",
	"HEADING_6 lineSpacing", "HEADING_6 spaceAbove", "HEADING_6 spaceBelow", "HEADING_6 indentStart",
	"SUBTITLE fontSize", "SUBTITLE font", "SUBTITLE colour", "SUBTITLE alignment",
	"SUBTITLE lineSpacing", "SUBTITLE spaceAbove", "SUBTITLE spaceBelow", "SUBTITLE indentStart",

	"table Revision History size", "table Revision History column widths",
	"table Revision History cell fills", "table Revision History cell text",
	"table Revision History cell borders", "table Revision History row heights",
	"table Document Classification size", "table Document Classification column widths",
	"table Document Classification cell fills", "table Document Classification cell text",
	"table Document Classification cell borders", "table Document Classification row heights",
	"body H2 text", "body H2 indentStart", "body H2 run colour",
	"body H3 text", "body H3 indentStart", "body H3 run colour",

	"tableOfContents instruction",
}

// TestTheDocsFixtureAnswersTheRowsItIsPinnedTo fails in both directions, the
// way TestTheGateReadsTheWholeList does on the docx half: a row that starts
// reading nil has to be explained, and a row that starts answering has to come
// out of the list.
func TestTheDocsFixtureAnswersTheRowsItIsPinnedTo(t *testing.T) {
	want := map[string]bool{}
	for _, name := range docsSilent {
		want[name] = true
	}
	for _, v := range FromDoc(openAnswer(t)) {
		if v.V == nil && !want[v.Name] {
			t.Errorf("item %q reads nothing out of the fixture, and it is not one of the rows the fixture is known to leave silent", v.Name)
		}
		if v.V != nil && want[v.Name] {
			t.Errorf("item %q now reads a value, so it is no longer one of the silent ones", v.Name)
		}
		delete(want, v.Name)
	}
	for name := range want {
		t.Errorf("%q is pinned as silent and is not an item at all", name)
	}
}
