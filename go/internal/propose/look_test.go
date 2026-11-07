package propose

import (
	"strings"
	"testing"

	"gdoc/internal/docsreq"
)

// testLook is a look handed to BlockBatch the way restyle.BlockLook hands one:
// body text justified at 12pt with the body gap, a list item with the list gap
// until the last, and a heading at a size of its own. The values are this
// test's, so a failure names the batch and never the house file.
func testLook(style string, bulleted, lastInList bool) (*docsreq.Fields, *docsreq.Fields) {
	para := docsreq.NewFields()
	para.Put("alignment", "JUSTIFIED")
	gap := 12.0
	if bulleted && !lastInList {
		gap = 3
	}
	para.Put("spaceBelow", docsreq.Points(gap))
	text := docsreq.NewFields()
	size := 12.0
	if style != "NORMAL_TEXT" {
		size = 14
	}
	text.Put("fontSize", docsreq.Points(size))
	return para, text
}

// houseSizedLook states as many fields as restyle.BlockLook does, four on the
// paragraph and three on its text, and a heading's text look is not the body's,
// as the house's is not. So a batch measured against the guard's peek is as
// large as the one a command really sends: a look the same for every style
// grouped headings with body text and hid the largest shape there is.
func houseSizedLook(style string, _, _ bool) (*docsreq.Fields, *docsreq.Fields) {
	para := docsreq.NewFields()
	para.Put("alignment", "JUSTIFIED")
	para.Put("spaceAbove", docsreq.Points(0))
	para.Put("spaceBelow", docsreq.Points(12))
	para.Put("lineSpacing", 115.0)
	text := docsreq.NewFields()
	text.Put("weightedFontFamily", map[string]any{"fontFamily": "Calibri"})
	size := 12.0
	if style != "NORMAL_TEXT" {
		size = 16
	}
	text.Put("fontSize", docsreq.Points(size))
	text.Put("foregroundColor", map[string]any{"color": map[string]any{"rgbColor": map[string]any{"red": 0.0, "green": 0.0, "blue": 0.0}}})
	return para, text
}

// magnitudeOf is the point size a style object states under one field.
func magnitudeOf(t *testing.T, set map[string]any, field string) float64 {
	t.Helper()
	v, ok := set[field].(map[string]any)
	if !ok {
		t.Fatalf("%s = %v, want a dimension", field, set[field])
	}
	return v["magnitude"].(float64)
}

// TestBlockBatchWritesTheLookItIsHandedOnEveryNewParagraph is Nail's rule of
// 2026-11-07: body text is 12pt whatever gdoc is doing. A new paragraph took the
// named style and nothing else, so it showed the Normal style's 11pt and no
// justification beside body text that carries 12pt directly. Every new
// paragraph now carries the look the caller handed over, its paragraph half in
// the request that names its style and its text half over its whole span.
func TestBlockBatchWritesTheLookItIsHandedOnEveryNewParagraph(t *testing.T) {
	content := blockContent(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	l := layOutBlock(place, content)
	reqs := requestsOf(t, BlockBatch(place, content, testLook, testWhy, "", "🤖"))

	wantSize := []float64{14, 12, 12, 12}
	wantGap := []float64{12, 12, 3, 12}
	for i, p := range content {
		span := [2]int{l.Paras[i].Start, l.Paras[i].End}
		var para, text map[string]any
		for _, r := range reqs {
			if body, ok := r["updateParagraphStyle"].(map[string]any); ok && rangeIn(t, r, "updateParagraphStyle") == span {
				para = body
			}
			if body, ok := r["updateTextStyle"].(map[string]any); ok && body["fields"] == "fontSize" {
				if at := rangeIn(t, r, "updateTextStyle"); at[0] <= span[0] && span[1] <= at[1] {
					text = body
				}
			}
		}
		if para == nil {
			t.Fatalf("paragraph %d (%q) has no updateParagraphStyle over %v", i, p.Text(), span)
		}
		if para["fields"] != "namedStyleType,alignment,spaceBelow" {
			t.Errorf("paragraph %d names %v, want its style and the look in one mask", i, para["fields"])
		}
		style := para["paragraphStyle"].(map[string]any)
		if style["namedStyleType"] != p.Style || style["alignment"] != "JUSTIFIED" {
			t.Errorf("paragraph %d states %v", i, style)
		}
		if got := magnitudeOf(t, style, "spaceBelow"); got != wantGap[i] {
			t.Errorf("paragraph %d is spaced %v below, want %v", i, got, wantGap[i])
		}
		if text == nil {
			t.Fatalf("paragraph %d (%q) is not covered by a text look over %v", i, p.Text(), span)
		}
		if got := magnitudeOf(t, text["textStyle"].(map[string]any), "fontSize"); got != wantSize[i] {
			t.Errorf("paragraph %d is %vpt, want %vpt", i, got, wantSize[i])
		}
	}
}

// TestTheLookComesBetweenTheClearingAndTheMarks is the order the look needs.
// After the clearing request, because that one resets what the insert
// inherited and the look is what goes in its place; before the block's own
// marks, so a run that is bold is still bold once its paragraph has its look.
func TestTheLookComesBetweenTheClearingAndTheMarks(t *testing.T) {
	content := blockContent(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	clearing, firstLook, lastLook, firstMark := -1, -1, -1, -1
	for i, r := range requestsOf(t, BlockBatch(place, content, testLook, testWhy, "", "🤖")) {
		body, ok := r["updateTextStyle"].(map[string]any)
		if !ok {
			continue
		}
		switch fields := body["fields"].(string); {
		case fields == clearedMarks:
			clearing = i
		case fields == "fontSize":
			if firstLook < 0 {
				firstLook = i
			}
			lastLook = i
		case strings.Contains(fields, "bold") && firstMark < 0:
			firstMark = i
		}
	}
	if clearing < 0 || firstLook < 0 || firstMark < 0 {
		t.Fatalf("the batch is missing a text request: clearing %d, look %d, mark %d", clearing, firstLook, firstMark)
	}
	if !(clearing < firstLook && lastLook < firstMark) {
		t.Errorf("text requests at clearing %d, look %d to %d, first mark %d; want clearing, then the look, then the marks", clearing, firstLook, lastLook, firstMark)
	}
}

// TestNeighboursThatLookTheSameShareOneTextRequest is the size rule. The body
// paragraph and the two list items behind the heading share the body's text
// look, so one request covers all three, and the heading has its own.
func TestNeighboursThatLookTheSameShareOneTextRequest(t *testing.T) {
	content := blockContent(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	l := layOutBlock(place, content)
	var got [][2]int
	for _, r := range requestsOf(t, BlockBatch(place, content, testLook, testWhy, "", "🤖")) {
		if body, ok := r["updateTextStyle"].(map[string]any); ok && body["fields"] == "fontSize" {
			got = append(got, rangeIn(t, r, "updateTextStyle"))
		}
	}
	want := [][2]int{{l.Paras[0].Start, l.Paras[0].End}, {l.Paras[1].Start, l.Paras[3].End}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("text look requests over %v, want %v", got, want)
	}
}

// TestNoLookWritesTheBatchItAlwaysWrote is the nil case: a caller that hands no
// look gets the named style alone, as before the rule. The command always hands
// one, and TestProposeWritesTheHouseBodyLookIntoABlock in cmd/gdoc pins that.
func TestNoLookWritesTheBatchItAlwaysWrote(t *testing.T) {
	content := blockContent(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	for _, r := range requestsOf(t, BlockBatch(place, content, nil, testWhy, "", "🤖")) {
		if body, ok := r["updateParagraphStyle"].(map[string]any); ok && body["fields"] != "namedStyleType" {
			t.Errorf("with no look a paragraph names %v, want its style alone", body["fields"])
		}
		if body, ok := r["updateTextStyle"].(map[string]any); ok && body["fields"] == "fontSize" {
			t.Errorf("with no look the batch still states a size: %v", body)
		}
	}
}

// TestABlockAtTheEndTakesNoLook is Nail's choice of 2026-11-07. After the
// document's last paragraph the block's own last paragraph owns the body's old
// final mark, and measured live that day, the batch there already left two
// suggestion ids and the look made it five: every style request near that mark
// came back as an id of its own. So a block at the end is written exactly as it
// was before the look, and
// docs/backlog/a-block-at-the-end-of-a-document-is-not-one-suggestion.md holds
// the case.
func TestABlockAtTheEndTakesNoLook(t *testing.T) {
	content := plainBlock(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "operations lead", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	if !place.AtEnd {
		t.Fatal("the anchor is the document's last paragraph, and this test is about that case")
	}
	with := BlockBatch(place, content, testLook, testWhy, "", "🤖")
	without := BlockBatch(place, content, nil, testWhy, "", "🤖")
	if string(with) != string(without) {
		t.Errorf("a block at the end carries the look:\n%s\nwant the batch with none:\n%s", with, without)
	}
}

// TestAListEndsWhereTheListItemsEnd is restyle's lastInList read the same way
// here: a bulleted list straight after a numbered one is one run of list items,
// so only the item before the plain paragraph is spaced as the last one. Were
// the change of kind an end, a restyle over the accepted block would move that
// gap, and the two looks would drift apart.
func TestAListEndsWhereTheListItemsEnd(t *testing.T) {
	content, err := ParseContent("1. a\n\n- b\n\nc\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}
	var lasts []bool
	look := func(style string, bulleted, lastInList bool) (*docsreq.Fields, *docsreq.Fields) {
		lasts = append(lasts, lastInList)
		return testLook(style, bulleted, lastInList)
	}
	for i := range content {
		lookOf(look, content, i)
	}
	want := []bool{false, true, false}
	for i := range want {
		if lasts[i] != want[i] {
			t.Errorf("paragraph %d is last in its list = %v, want %v", i, lasts[i], want[i])
		}
	}
}
