package restyle

import (
	"strings"
	"testing"
)

// The landing half asks one question: the document was told these values, does
// it carry them. Every want below is written out as a literal, because a test
// that reads the request it is checking is a mirror.

// dim is one Docs Dimension, the shape both a request and a read carry.
func dim(v float64) map[string]any {
	return map[string]any{"magnitude": v, "unit": "PT"}
}

func span(start, end int) map[string]any {
	return map[string]any{"startIndex": start, "endIndex": end}
}

// sentRequests is one request of each of the four kinds, over a document with
// one paragraph and one table in it.
func sentRequests() []map[string]any {
	return []map[string]any{
		{"updateDocumentStyle": map[string]any{
			"documentStyle": map[string]any{"marginTop": dim(62.35), "marginLeft": dim(51.05)},
			"fields":        "marginTop,marginLeft",
		}},
		{"updateParagraphStyle": map[string]any{
			"range":          span(1, 20),
			"paragraphStyle": map[string]any{"alignment": "JUSTIFIED", "spaceBelow": dim(6), "lineSpacing": 115.0},
			"fields":         "alignment,spaceBelow,lineSpacing",
		}},
		{"updateTextStyle": map[string]any{
			"range":     span(1, 20),
			"textStyle": map[string]any{"weightedFontFamily": map[string]any{"fontFamily": "Aptos"}, "fontSize": dim(10.5)},
			"fields":    "weightedFontFamily,fontSize",
		}},
		{"updateTableCellStyle": map[string]any{
			"tableRange": map[string]any{
				"tableCellLocation": map[string]any{
					"tableStartLocation": map[string]any{"index": 20},
					"rowIndex":           0,
					"columnIndex":        0,
				},
				"rowSpan": 1, "columnSpan": 2,
			},
			"tableCellStyle": map[string]any{"paddingTop": dim(3)},
			"fields":         "paddingTop",
		}},
	}
}

// styledRead is a document that carries everything sentRequests asked for. The
// extra fields on it are the ones Google answers with and nobody set: a font
// weight, and a paragraph's own named style.
const styledRead = `{
  "documentId": "DOC1", "revisionId": "rev2",
  "tabs": [{"tabProperties": {"tabId": "t.0"}, "documentTab": {
    "documentStyle": {
      "marginTop": {"magnitude": 62.35, "unit": "PT"},
      "marginBottom": {"magnitude": 51, "unit": "PT"},
      "marginLeft": {"magnitude": 51.05, "unit": "PT"}},
    "body": {"content": [
      {"startIndex": 1, "endIndex": 20, "paragraph": {
        "paragraphStyle": {"namedStyleType": "NORMAL_TEXT", "alignment": "JUSTIFIED",
                           "spaceBelow": {"magnitude": 6, "unit": "PT"}, "lineSpacing": 115},
        "elements": [{"startIndex": 1, "endIndex": 20, "textRun": {"content": "The supplier\n",
          "textStyle": {"weightedFontFamily": {"fontFamily": "Aptos", "weight": 400},
                        "fontSize": {"magnitude": 10.5, "unit": "PT"}}}}]}},
      {"startIndex": 20, "endIndex": 40, "table": {"tableRows": [
        {"tableCells": [
          {"tableCellStyle": {"paddingTop": {"magnitude": 3, "unit": "PT"}}, "content": []},
          {"tableCellStyle": {"paddingTop": {"magnitude": 3, "unit": "PT"}}, "content": []}]}]}}
    ]}}}]
}`

func TestTheReadBackFindsTheStyleItSent(t *testing.T) {
	// Act
	got, warnings := Landed([]byte(styledRead), sentRequests())

	// Assert
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none on a read the styling is in", warnings)
	}
	if len(got.Checks) != 4 {
		t.Fatalf("checks = %+v, want one for each of the four kinds", got.Checks)
	}
	want := []string{"updateDocumentStyle", "updateParagraphStyle", "updateTextStyle", "updateTableCellStyle"}
	for i, c := range got.Checks {
		if c.Kind != want[i] {
			t.Errorf("check %d is %s, want %s", i, c.Kind, want[i])
		}
		if !c.Held {
			t.Errorf("%s: held = false at %s, missing %v, note %q", c.Kind, c.Where, c.Missing, c.Note)
		}
	}
	if !strings.Contains(got.Checks[1].Where, "1") {
		t.Errorf("where = %q, want the paragraph it looked at named", got.Checks[1].Where)
	}
}

// The accepted-but-not-landed row is the whole reason this half exists: a
// document carrying section breaks has its margins governed by its sectionStyle,
// and updateSectionStyle is not on the in-place allowlist, so the page request
// can be accepted and change nothing a reader sees.
func TestAStyleThatWasAcceptedAndDidNotLandIsNamed(t *testing.T) {
	read := strings.Replace(styledRead, `"marginTop": {"magnitude": 62.35, "unit": "PT"}`,
		`"marginTop": {"magnitude": 72, "unit": "PT"}`, 1)

	got, _ := Landed([]byte(read), sentRequests())

	page := got.Checks[0]
	if page.Held {
		t.Fatalf("held = true on a document whose margin is 72pt and not the 62.35 that was sent")
	}
	if len(page.Missing) != 1 || page.Missing[0] != "marginTop.magnitude" {
		t.Errorf("missing = %v, want marginTop.magnitude named", page.Missing)
	}
}

// A field the read does not carry at all is the document's own default, because
// Docs leaves a property equal to its default out of the answer. So a zero holds
// where the read is silent, and anything else does not.
func TestAZeroTheReadDoesNotCarryIsTheZeroBeingThere(t *testing.T) {
	sent := []map[string]any{
		{"updateParagraphStyle": map[string]any{
			"range":          span(1, 20),
			"paragraphStyle": map[string]any{"spaceAbove": dim(0)},
			"fields":         "spaceAbove",
		}},
	}

	got, _ := Landed([]byte(styledRead), sent)

	if !got.Checks[0].Held {
		t.Errorf("held = false, missing %v: a zero the read leaves out is the default being there",
			got.Checks[0].Missing)
	}
}

func TestANonZeroTheReadDoesNotCarryDidNotLand(t *testing.T) {
	sent := []map[string]any{
		{"updateParagraphStyle": map[string]any{
			"range":          span(1, 20),
			"paragraphStyle": map[string]any{"indentStart": dim(18)},
			"fields":         "indentStart",
		}},
	}

	got, _ := Landed([]byte(styledRead), sent)

	if got.Checks[0].Held {
		t.Error("held = true on an indent the read back does not carry at all")
	}
	if len(got.Checks[0].Missing) != 1 || got.Checks[0].Missing[0] != "indentStart" {
		t.Errorf("missing = %v, want indentStart named", got.Checks[0].Missing)
	}
}

// A lookup that found nothing is never held. It is not evidence the style
// landed, and reporting it as one would be a verified: true over a paragraph
// nobody read.
func TestAParagraphTheReadBackCannotFindIsNotHeld(t *testing.T) {
	sent := []map[string]any{
		{"updateParagraphStyle": map[string]any{
			"range":          span(999, 1010),
			"paragraphStyle": map[string]any{"spaceBelow": dim(6)},
			"fields":         "spaceBelow",
		}},
	}

	got, _ := Landed([]byte(styledRead), sent)

	if got.Checks[0].Held || got.Checks[0].Note == "" {
		t.Errorf("check = %+v, want not held with a note saying there was nowhere to look", got.Checks[0])
	}
}

// A kind the run never sent is not checked. A document with no table sends no
// updateTableCellStyle, and reporting a check that could not hold would make
// every document without a table unverifiable.
func TestAKindThatWasNeverSentIsNotChecked(t *testing.T) {
	sent := sentRequests()[:1]

	got, _ := Landed([]byte(styledRead), sent)

	if len(got.Checks) != 1 || got.Checks[0].Kind != "updateDocumentStyle" {
		t.Errorf("checks = %+v, want the one kind that was sent", got.Checks)
	}
}

// The first of each kind, and the check names which one it read. A restyle
// sends one request per paragraph, so checking every one is hundreds of lookups
// answering the same question.
func TestTheCheckReadsTheFirstRequestOfItsKind(t *testing.T) {
	sent := append(sentRequests(), map[string]any{"updateParagraphStyle": map[string]any{
		"range":          span(999, 1010),
		"paragraphStyle": map[string]any{"spaceBelow": dim(6)},
		"fields":         "spaceBelow",
	}})

	got, _ := Landed([]byte(styledRead), sent)

	if len(got.Checks) != 4 {
		t.Fatalf("checks = %d, want one per kind however many requests were sent", len(got.Checks))
	}
	if !got.Checks[1].Held {
		t.Errorf("the paragraph check read %s, want the first request of its kind", got.Checks[1].Where)
	}
}

// A cell inside a nested table carries its own start index, so the walk goes
// into cells rather than answering with the outer table's.
func TestACellOfANestedTableIsFound(t *testing.T) {
	read := `{"tabs": [{"documentTab": {"body": {"content": [
	  {"startIndex": 20, "table": {"tableRows": [{"tableCells": [{"tableCellStyle": {},
	    "content": [{"startIndex": 25, "table": {"tableRows": [{"tableCells": [
	      {"tableCellStyle": {"paddingTop": {"magnitude": 3, "unit": "PT"}}, "content": []}]}]}}]}]}]}}]}}}]}`
	sent := []map[string]any{
		{"updateTableCellStyle": map[string]any{
			"tableRange": map[string]any{"tableCellLocation": map[string]any{
				"tableStartLocation": map[string]any{"index": 25}, "rowIndex": 0, "columnIndex": 0}},
			"tableCellStyle": map[string]any{"paddingTop": dim(3)},
			"fields":         "paddingTop",
		}},
	}

	got, _ := Landed([]byte(read), sent)

	if !got.Checks[0].Held {
		t.Errorf("check = %+v, want the nested table's own cell read", got.Checks[0])
	}
}

// A document written before tabs existed answers with a top-level body, and the
// read-back reads it the way internal/docs does.
func TestAPreTabsDocumentIsReadBackToo(t *testing.T) {
	read := `{"documentStyle": {"marginTop": {"magnitude": 62.35, "unit": "PT"}},
	         "body": {"content": [{"startIndex": 1, "paragraph": {
	           "paragraphStyle": {"spaceBelow": {"magnitude": 6, "unit": "PT"}}, "elements": []}}]}}`
	sent := []map[string]any{
		{"updateDocumentStyle": map[string]any{
			"documentStyle": map[string]any{"marginTop": dim(62.35)}, "fields": "marginTop"}},
		{"updateParagraphStyle": map[string]any{
			"range":          span(1, 20),
			"paragraphStyle": map[string]any{"spaceBelow": dim(6)},
			"fields":         "spaceBelow"}},
	}

	got, _ := Landed([]byte(read), sent)

	for _, c := range got.Checks {
		if !c.Held {
			t.Errorf("%s: held = false, missing %v, note %q", c.Kind, c.Missing, c.Note)
		}
	}
}

// A read the decoder could not read is a warning and no checks, never a held
// check: nothing there says the style landed.
func TestAReadBackThatDidNotDecodeSaysSo(t *testing.T) {
	got, warnings := Landed([]byte("not json"), sentRequests())

	if len(got.Checks) != 0 {
		t.Errorf("checks = %+v, want none on a read nobody could decode", got.Checks)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "styling") {
		t.Errorf("warnings = %v, want one naming the read that could not be decoded", warnings)
	}
}

// A colour is written as a channel out of 255 and comes back through the API's
// own precision, so the comparison is a tolerance and not an equality.
func TestAColourReadBackWithinToleranceHolds(t *testing.T) {
	sent := []map[string]any{
		{"updateTextStyle": map[string]any{
			"range": span(1, 20),
			"textStyle": map[string]any{"foregroundColor": map[string]any{
				"color": map[string]any{"rgbColor": map[string]any{
					"red": 0.29411764705882354, "green": 0.29411764705882354, "blue": 0.29411764705882354}}}},
			"fields": "foregroundColor",
		}},
	}
	read := strings.Replace(styledRead, `"fontSize": {"magnitude": 10.5, "unit": "PT"}`,
		`"fontSize": {"magnitude": 10.5, "unit": "PT"},
		 "foregroundColor": {"color": {"rgbColor": {"red": 0.29411766, "green": 0.29411766, "blue": 0.29411766}}}`, 1)

	got, _ := Landed([]byte(read), sent)

	if !got.Checks[0].Held {
		t.Errorf("held = false, missing %v: a colour is compared within a tolerance", got.Checks[0].Missing)
	}
}
