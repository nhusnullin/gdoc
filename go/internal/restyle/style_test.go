package restyle

// This file holds the look: what a restyle writes onto a paragraph, onto the
// runs inside it and onto a table's cells. page_test.go states the rule these
// tests hold to, and it is the milestone's rule rather than this package's: a
// request names in its mask exactly what it sets, because a path in the mask
// the request leaves unset resets that property to its default.
//
// Every house value below is written out as a literal. A test reading
// cfg.Styles["heading_1"].SizePt would be a mirror, following the house style
// wherever somebody moved it, and a house style that had quietly changed would
// still pass. The house value is printed beside the want on a failure instead.

import (
	"encoding/json"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/guard"
)

// fixtureTab is the measured single-tab document internal/docs tests read: a
// HEADING_1, a NORMAL_TEXT paragraph carrying a pending suggestion, a
// HEADING_2, two bulleted paragraphs, another paragraph and a two by two table
// starting at 184.
func fixtureTab(t *testing.T) docs.Tab {
	t.Helper()
	b, err := os.ReadFile("../docs/testdata/single-tab.json")
	if err != nil {
		t.Fatalf("the fixture must be readable: %v", err)
	}
	doc, err := docs.Parse(b)
	if err != nil {
		t.Fatalf("the fixture must parse: %v", err)
	}
	if len(doc.Tabs) != 1 {
		t.Fatalf("the fixture has %d tabs, want 1", len(doc.Tabs))
	}
	return doc.Tabs[0]
}

// one is the single request of a kind whose range starts where the caller says.
// It fails rather than returning nothing, because a missing request is the
// failure every test here is about.
func one(t *testing.T, plan Plan, kind string, start int) map[string]any {
	t.Helper()
	var found []map[string]any
	for _, req := range plan.Requests {
		body, ok := req[kind].(map[string]any)
		if !ok {
			continue
		}
		if at(body) == start {
			found = append(found, body)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d %s requests start at %d, want 1", len(found), kind, start)
	}
	return found[0]
}

// at is where a request's range starts: the range for a paragraph or a run, and
// the table's own start for a cell.
func at(body map[string]any) int {
	if r, ok := body["range"].(map[string]any); ok {
		if v, ok := r["startIndex"].(int); ok {
			return v
		}
	}
	if r, ok := body["tableRange"].(map[string]any); ok {
		loc, _ := r["tableCellLocation"].(map[string]any)
		start, _ := loc["tableStartLocation"].(map[string]any)
		if v, ok := start["index"].(int); ok {
			return v
		}
	}
	return -1
}

// magnitude reads one {magnitude, unit} out of a style object, failing when the
// unit is not PT: the house style stores points and the Docs API's only unit is
// PT, so anything else is a conversion nobody asked for.
func magnitude(t *testing.T, in map[string]any, key string) float64 {
	t.Helper()
	v, ok := in[key].(map[string]any)
	if !ok {
		t.Fatalf("the style carries no %s: %v", key, in)
	}
	if unit, _ := v["unit"].(string); unit != "PT" {
		t.Errorf("%s is in %q, want PT", key, unit)
	}
	mag, ok := v["magnitude"].(float64)
	if !ok {
		t.Fatalf("%s carries no magnitude: %v", key, v["magnitude"])
	}
	return mag
}

// colour reads an optional colour's three channels.
func colour(t *testing.T, in map[string]any, key string) (float64, float64, float64) {
	t.Helper()
	opt, ok := in[key].(map[string]any)
	if !ok {
		t.Fatalf("the style carries no %s: %v", key, in)
	}
	c, _ := opt["color"].(map[string]any)
	rgb, ok := c["rgbColor"].(map[string]any)
	if !ok {
		t.Fatalf("%s carries no rgbColor: %v", key, opt)
	}
	r, _ := rgb["red"].(float64)
	g, _ := rgb["green"].(float64)
	b, _ := rgb["blue"].(float64)
	return r, g, b
}

// styleOf is the style object of one request, by the key that request carries.
func styleOf(t *testing.T, body map[string]any, key string) map[string]any {
	t.Helper()
	s, ok := body[key].(map[string]any)
	if !ok {
		t.Fatalf("the request carries no %s: %v", key, body)
	}
	return s
}

// A HEADING_1 paragraph takes the house heading look: 12pt above, nothing
// below, the 1.15 line spacing the document defaults state, and no indent.
func TestAHeadingTakesTheHouseHeadingLook(t *testing.T) {
	cfg := embeddedHouse(t)
	plan := TabRequests(fixtureTab(t), cfg)
	body := one(t, plan, "updateParagraphStyle", 1)
	ps := styleOf(t, body, "paragraphStyle")
	h := cfg.Styles["heading_1"]
	if got := magnitude(t, ps, "spaceAbove"); got != 12 {
		t.Errorf("spaceAbove is %v, want 12 (house says %v)", got, h.SpaceBeforePt)
	}
	if got := magnitude(t, ps, "spaceBelow"); got != 0 {
		t.Errorf("spaceBelow is %v, want 0 (house says %v)", got, h.SpaceAfterPt)
	}
	if got := magnitude(t, ps, "indentStart"); got != 0 {
		t.Errorf("indentStart is %v, want 0 (house says %v)", got, h.IndentStartPt)
	}
	if got, ok := ps["lineSpacing"].(float64); !ok || got != 115 {
		t.Errorf("lineSpacing is %v, want 115 (house says %v)", ps["lineSpacing"], cfg.Defaults.LineSpacing)
	}
}

// The same heading's runs: Calibri at 16pt in the house navy. The colour is
// read as three channels because that is what the Docs API takes, and the want
// is the master's own #22265F written out.
func TestAHeadingsRunsTakeTheHouseFontSizeAndColour(t *testing.T) {
	cfg := embeddedHouse(t)
	plan := TabRequests(fixtureTab(t), cfg)
	body := one(t, plan, "updateTextStyle", 1)
	ts := styleOf(t, body, "textStyle")
	family, _ := ts["weightedFontFamily"].(map[string]any)
	if got, _ := family["fontFamily"].(string); got != "Calibri" {
		t.Errorf("the font is %q, want Calibri (house says %q)", got, cfg.Defaults.Font)
	}
	if got := magnitude(t, ts, "fontSize"); got != 16 {
		t.Errorf("fontSize is %v, want 16 (house says %v)", got, cfg.Styles["heading_1"].SizePt)
	}
	r, g, b := colour(t, ts, "foregroundColor")
	if r != 34.0/255.0 || g != 38.0/255.0 || b != 95.0/255.0 {
		t.Errorf("the colour is %v %v %v, want #22265F (house says %q)",
			r, g, b, cfg.Styles["heading_1"].Color)
	}
}

// Body prose takes the body look, which is what build writes onto an ordinary
// paragraph: 12pt, justified, no space above or below.
func TestBodyProseTakesTheHouseBodyLook(t *testing.T) {
	cfg := embeddedHouse(t)
	plan := TabRequests(fixtureTab(t), cfg)
	ps := styleOf(t, one(t, plan, "updateParagraphStyle", 7), "paragraphStyle")
	if got, _ := ps["alignment"].(string); got != "JUSTIFIED" {
		t.Errorf("the alignment is %q, want JUSTIFIED (house says %q)", got, cfg.Body.Align)
	}
	if got := magnitude(t, ps, "spaceAbove"); got != 0 {
		t.Errorf("spaceAbove is %v, want 0 (house says %v)", got, cfg.Body.SpaceBeforePt)
	}
	if got := magnitude(t, ps, "spaceBelow"); got != 0 {
		t.Errorf("spaceBelow is %v, want 0 (house says %v)", got, cfg.Body.SpaceAfterPt)
	}
	ts := styleOf(t, one(t, plan, "updateTextStyle", 7), "textStyle")
	if got := magnitude(t, ts, "fontSize"); got != 12 {
		t.Errorf("fontSize is %v, want 12 (house says %v)", got, cfg.Body.SizePt)
	}
	r, g, b := colour(t, ts, "foregroundColor")
	if r != 0 || g != 0 || b != 0 {
		t.Errorf("the colour is %v %v %v, want #000000 (house says %q)",
			r, g, b, cfg.Styles["normal"].Color)
	}
}

// The structure is read, never decided. A paragraph's namedStyleType is what
// picks the look, and no request names it: a restyle that wrote one would be
// restructuring somebody's document.
func TestNoRequestWritesANamedStyleType(t *testing.T) {
	plan := TabRequests(fixtureTab(t), embeddedHouse(t))
	raw, err := json.Marshal(plan.Requests)
	if err != nil {
		t.Fatalf("the requests must marshal: %v", err)
	}
	if strings.Contains(string(raw), "namedStyleType") {
		t.Errorf("a request names namedStyleType, which would change the structure: %s", raw)
	}
}

// The mask rule, over every request this package builds. It is the same check
// page_test.go makes on the one request that names no range, and it is written
// once over the whole plan because a builder added later must answer it too.
func TestEveryMaskNamesExactlyWhatItSets(t *testing.T) {
	plan := TabRequests(fixtureTab(t), embeddedHouse(t))
	if len(plan.Requests) == 0 {
		t.Fatal("the fixture built no requests, so this checks nothing")
	}
	for i, req := range plan.Requests {
		for kind, body := range req {
			fields, ok := body.(map[string]any)
			if !ok {
				t.Fatalf("request %d (%s) is not an object: %v", i, kind, body)
			}
			mask, ok := fields["fields"].(string)
			if !ok || mask == "" {
				t.Fatalf("request %d (%s) carries no fields mask", i, kind)
			}
			style, ok := fields[styleKeyOf(kind)].(map[string]any)
			if !ok {
				t.Fatalf("request %d (%s) carries no %s", i, kind, styleKeyOf(kind))
			}
			var set []string
			for k := range style {
				set = append(set, k)
			}
			named := strings.Split(mask, ",")
			sort.Strings(set)
			sort.Strings(named)
			if strings.Join(set, ",") != strings.Join(named, ",") {
				t.Errorf("request %d (%s) sets %v and its mask names %v, and they must be the same set",
					i, kind, set, named)
			}
		}
	}
}

// styleKeyOf names the object each request kind carries its style in.
func styleKeyOf(kind string) string {
	switch kind {
	case "updateParagraphStyle":
		return "paragraphStyle"
	case "updateTextStyle":
		return "textStyle"
	case "updateTableCellStyle":
		return "tableCellStyle"
	}
	return "documentStyle"
}

// Nothing writes a property the house style states as a plain flag, and nothing
// writes a border. A flag is a value the file cannot tell apart from silence,
// so writing it would clear an author's own emphasis on the strength of a value
// house.yaml may never have stated. A border is the other half: the house style
// draws none under a paragraph, and naming borderBottom in a mask without
// setting it is how a restyle would erase a rule the author drew.
func TestNothingWritesAFlagOrABorderTheHouseStyleDoesNotState(t *testing.T) {
	plan := TabRequests(fixtureTab(t), embeddedHouse(t))
	raw, err := json.Marshal(plan.Requests)
	if err != nil {
		t.Fatalf("the requests must marshal: %v", err)
	}
	for _, name := range []string{"bold", "italic", "underline", "keepWithNext", "keepLinesTogether"} {
		if strings.Contains(string(raw), name) {
			t.Errorf("a request names %q, which the house style states as a flag: %s", name, raw)
		}
	}
	// The borders are asked of the paragraph requests alone, because a cell's
	// four edges are borders this milestone does draw.
	for _, req := range plan.Requests {
		body, ok := req["updateParagraphStyle"]
		if !ok {
			continue
		}
		one, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("the request must marshal: %v", err)
		}
		if strings.Contains(string(one), "border") {
			t.Errorf("a paragraph request names a border, and the house style draws none: %s", one)
		}
	}
}

// A table's cells take the padding and the borders, one request per row, and no
// shading at all: which row of somebody's table is a header is not something
// gdoc can read, and a background named in a mask replaces the fill the author
// chose.
func TestATablesCellsTakeThePaddingAndTheBorders(t *testing.T) {
	plan := TabRequests(fixtureTab(t), embeddedHouse(t))
	var rows []map[string]any
	for _, req := range plan.Requests {
		if body, ok := req["updateTableCellStyle"].(map[string]any); ok {
			rows = append(rows, body)
		}
	}
	if len(rows) != 2 {
		t.Fatalf("the fixture's table has two rows and built %d cell requests", len(rows))
	}
	for i, body := range rows {
		tr, _ := body["tableRange"].(map[string]any)
		loc, _ := tr["tableCellLocation"].(map[string]any)
		start, _ := loc["tableStartLocation"].(map[string]any)
		if got, _ := start["index"].(int); got != 184 {
			t.Errorf("row %d names table start %v, want 184", i, start["index"])
		}
		if got, _ := loc["rowIndex"].(int); got != i {
			t.Errorf("row %d names rowIndex %v", i, loc["rowIndex"])
		}
		if got, _ := loc["columnIndex"].(int); got != 0 {
			t.Errorf("row %d names columnIndex %v, want 0", i, loc["columnIndex"])
		}
		if got, _ := tr["rowSpan"].(int); got != 1 {
			t.Errorf("row %d names rowSpan %v, want 1", i, tr["rowSpan"])
		}
		if got, _ := tr["columnSpan"].(int); got != 2 {
			t.Errorf("row %d names columnSpan %v, want 2", i, tr["columnSpan"])
		}
		cs := styleOf(t, body, "tableCellStyle")
		if _, ok := cs["backgroundColor"]; ok {
			t.Errorf("row %d shades the cells, and a restyle leaves a cell's own fill alone", i)
		}
		if got := magnitude(t, cs, "paddingTop"); got != 3 {
			t.Errorf("row %d pads the top by %v, want 3", i, got)
		}
		if got := magnitude(t, cs, "paddingLeft"); got != 5.4 {
			t.Errorf("row %d pads the left by %v, want 5.4", i, got)
		}
		border, ok := cs["borderTop"].(map[string]any)
		if !ok {
			t.Fatalf("row %d draws no top border: %v", i, cs)
		}
		if got := magnitude(t, border, "width"); got != 0.5 {
			t.Errorf("row %d draws a %vpt border, want 0.5", i, got)
		}
		if got, _ := border["dashStyle"].(string); got != "SOLID" {
			t.Errorf("row %d draws a %q border, want SOLID", i, got)
		}
		r, g, b := colour(t, border, "color")
		if r != 201.0/255.0 || g != 201.0/255.0 || b != 201.0/255.0 {
			t.Errorf("row %d draws a border coloured %v %v %v, want #C9C9C9", i, r, g, b)
		}
	}
}

// A cell's paragraphs take the table text the house style states and nothing
// else. The body's justified alignment is not the look inside a narrow cell,
// and house.yaml states no spacing for a cell of a table the author wrote, so
// the paragraph itself is left as it is.
func TestACellsTextTakesTheHouseTableText(t *testing.T) {
	cfg := embeddedHouse(t)
	plan := TabRequests(fixtureTab(t), cfg)
	ts := styleOf(t, one(t, plan, "updateTextStyle", 186), "textStyle")
	family, _ := ts["weightedFontFamily"].(map[string]any)
	if got, _ := family["fontFamily"].(string); got != "Calibri" {
		t.Errorf("the cell font is %q, want Calibri (house says %q)", got, cfg.TableText.Font)
	}
	if got := magnitude(t, ts, "fontSize"); got != 12 {
		t.Errorf("the cell size is %v, want 12 (house says %v)", got, cfg.TableText.DefaultSizePt)
	}
	for _, req := range plan.Requests {
		body, ok := req["updateParagraphStyle"].(map[string]any)
		if ok && at(body) == 186 {
			t.Errorf("a cell paragraph was restyled, and the house style states nothing to write there: %v", body)
		}
	}
}

// The list is left as it is, and it is counted rather than explained.
// createParagraphBullets removes the leading tabs that set a bullet's nesting
// level, so it is not one of the four kinds this level carries.
func TestABulletedParagraphKeepsItsBulletAndIsCounted(t *testing.T) {
	plan := TabRequests(fixtureTab(t), embeddedHouse(t))
	if plan.Bulleted != 2 {
		t.Errorf("the fixture carries two bulleted paragraphs and %d were counted", plan.Bulleted)
	}
	raw, _ := json.Marshal(plan.Requests)
	if strings.Contains(string(raw), "createParagraphBullets") {
		t.Errorf("a request creates bullets, which removes the author's leading tabs: %s", raw)
	}
}

// The two things a table keeps: its column widths and its row heights. Both
// need a request kind the in-place level does not carry, so the table is
// counted and the caller reports it.
func TestATablesWidthsAndHeightsAreCountedNotWritten(t *testing.T) {
	plan := TabRequests(fixtureTab(t), embeddedHouse(t))
	if plan.Tables != 1 {
		t.Errorf("the fixture holds one table and %d were counted", plan.Tables)
	}
	raw, _ := json.Marshal(plan.Requests)
	for _, kind := range []string{"updateTableColumnProperties", "updateTableRowStyle"} {
		if strings.Contains(string(raw), kind) {
			t.Errorf("a request names %s, which is not on the in-place allowlist: %s", kind, raw)
		}
	}
}

// A named style the house style has no look for is reported and never guessed
// at. Nothing infers a look from the words in the paragraph.
func TestAStyleTheHouseDoesNotKnowIsReportedAndNotGuessed(t *testing.T) {
	tab := docs.Tab{ID: "t.0", Body: []docs.Block{
		{Paragraph: &docs.Paragraph{Style: "HEADING_9", StartIndex: 1, EndIndex: 8}},
		{Paragraph: &docs.Paragraph{Style: "", StartIndex: 8, EndIndex: 20}},
	}}
	plan := TabRequests(tab, embeddedHouse(t))
	if len(plan.Requests) != 0 {
		t.Errorf("a paragraph gdoc has no look for was styled anyway: %v", plan.Requests)
	}
	if strings.Join(plan.Unstyled, ",") != "(none),HEADING_9" {
		t.Errorf("the styles reported as unknown are %v, want [(none) HEADING_9]", plan.Unstyled)
	}
	if plan.Paragraphs != 0 {
		t.Errorf("%d paragraphs were counted as restyled, want 0", plan.Paragraphs)
	}
}

// A tab with nothing in it builds nothing. A batch with no requests is a call
// that spends a revision id for no reason.
func TestAnEmptyTabBuildsNothing(t *testing.T) {
	plan := TabRequests(docs.Tab{ID: "t.0"}, embeddedHouse(t))
	if len(plan.Requests) != 0 {
		t.Errorf("an empty tab built %d requests", len(plan.Requests))
	}
}

// The counts are what was built, not what the document holds.
func TestThePlanCountsWhatItBuilt(t *testing.T) {
	plan := TabRequests(fixtureTab(t), embeddedHouse(t))
	// Six body paragraphs carry a named style the house knows, and the four
	// cell paragraphs are text only.
	if plan.Paragraphs != 6 {
		t.Errorf("%d paragraph requests were counted, want 6", plan.Paragraphs)
	}
	if plan.Text != 10 {
		t.Errorf("%d text requests were counted, want 10", plan.Text)
	}
	if plan.Cells != 4 {
		t.Errorf("%d cells were counted, want 4", plan.Cells)
	}
	kinds := map[string]int{}
	for _, req := range plan.Requests {
		for kind := range req {
			kinds[kind]++
		}
	}
	if kinds["updateParagraphStyle"] != plan.Paragraphs || kinds["updateTextStyle"] != plan.Text {
		t.Errorf("the counts say %d paragraphs and %d text, and the requests hold %v",
			plan.Paragraphs, plan.Text, kinds)
	}
}

// The builder is judged by the guard it writes for, whole. A refusal here means
// the two drifted, and the guard is the one thing standing between this
// milestone and a document it may not change.
func TestTheGuardCarriesEveryRestyleRequest(t *testing.T) {
	cfg := embeddedHouse(t)
	plan := TabRequests(fixtureTab(t), cfg)
	requests := append([]map[string]any{PageRequest(cfg)}, plan.Requests...)
	body, err := json.Marshal(map[string]any{"requests": requests})
	if err != nil {
		t.Fatalf("the batch must marshal: %v", err)
	}
	p := guard.NewPolicy()
	p.AllowFile("DOC1", guard.LevelSuggest)
	p.GrantInPlace("DOC1")
	u, err := url.Parse("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if err != nil {
		t.Fatalf("the URL must parse: %v", err)
	}
	if err := p.Judge("POST", u, body); err != nil {
		t.Fatalf("the guard refused the batch this milestone builds: %v", err)
	}
}
