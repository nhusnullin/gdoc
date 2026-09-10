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
			"tableStartLocation": map[string]any{"index": 20},
			"tableCellStyle":     map[string]any{"paddingTop": dim(3)},
			"fields":             "paddingTop",
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

// A document that came back without the value that was sent is a style that did
// not land, and the check names the field rather than the request. This is the
// plain shape of that: no section break anywhere, and a documentStyle carrying a
// margin the run never asked for. The section-override shape, where the value
// does read back and a section's own governs the page, is its own case and its
// own test: TestAPageMarginASectionOverridesIsNotReportedAsLanded.
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

// unanswered says in its own words that it is never held, and compare's own
// arithmetic can contradict it: missingFields walks the fields the request set,
// so a request that set none is missing none and Held is already true by the
// time the note is written on. A check with nothing behind it feeding verified
// a true it never measured is the one thing this half exists to refuse, so the
// invariant is written on the way out rather than left to the caller.
func TestACheckWithNothingToCompareIsNeverHeld(t *testing.T) {
	got := compare(Check{Kind: "updateDocumentStyle"}, nil, nil)

	if got.Held {
		t.Errorf("check = %+v, want not held: there was nothing to look for", got)
	}
	if got.Note == "" {
		t.Errorf("check = %+v, want a note saying the request set no style object", got)
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
			"tableStartLocation": map[string]any{"index": 25},
			"tableCellStyle":     map[string]any{"paddingTop": dim(3)},
			"fields":             "paddingTop",
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

// The accepted-but-invisible page is the case this half was written for, and
// reading documentStyle alone cannot see it: updateDocumentStyle writes that
// object, so the request reads back exactly as it was sent while the section's
// own margin is what the reader sees. The check is no answer there, never held.
func TestAPageMarginASectionOverridesIsNotReportedAsLanded(t *testing.T) {
	// Arrange: the styled read, with a section break that sets its own top
	// margin, so the value in documentStyle is the one that was sent and the
	// value on the page is the section's.
	read := strings.Replace(styledRead, `"content": [`,
		`"content": [{"startIndex": 0, "sectionBreak": {"sectionStyle": {"sectionType": "CONTINUOUS",`+
			`"marginTop": {"magnitude": 90, "unit": "PT"}}}},`, 1)

	// Act
	got, _ := Landed([]byte(read), sentRequests())

	// Assert
	page := got.Checks[0]
	if page.Held {
		t.Errorf("held = true over a margin a section break overrides: %+v", page)
	}
	if !strings.Contains(page.Note, "marginTop") || !strings.Contains(page.Note, "section") {
		t.Errorf("note = %q, want it to name the field and say a section sets it", page.Note)
	}
}

// A section that flips the page orientation shows the pageSize that was sent
// transposed, and it is the accepted-and-invisible shape one field along: the
// request reads back exactly as it was sent while a reader turns landscape
// pages. No request this level carries can set flipPageOrientation, so the page
// check would never match it by name.
func TestASectionThatFlipsTheOrientationIsNotReportedAsLanded(t *testing.T) {
	// Arrange: the document is asked for A4 portrait and answers with it, and a
	// section break flips its own pages.
	read := strings.Replace(pageSizeRead(), `"content": [`,
		`"content": [{"startIndex": 0, "sectionBreak": {"sectionStyle": {"sectionType": "NEXT_PAGE",`+
			`"flipPageOrientation": true}}},`, 1)

	// Act
	got, _ := Landed([]byte(read), pageSizeRequests())

	// Assert
	page := got.Checks[0]
	if page.Held {
		t.Errorf("held = true over a page a section turns on its side: %+v", page)
	}
	if !strings.Contains(page.Note, "flipPageOrientation") {
		t.Errorf("note = %q, want it to name the field the section states", page.Note)
	}
}

// A document already flipped throughout was flipped before this run, and a
// section agreeing with it overrides nothing anybody could see. Naming it would
// be the warning that fires on the working case.
func TestASectionAgreeingWithTheDocumentsOwnOrientationStillAnswers(t *testing.T) {
	read := strings.Replace(pageSizeRead(), `"documentStyle": {`,
		`"documentStyle": {"flipPageOrientation": true,`, 1)
	read = strings.Replace(read, `"content": [`,
		`"content": [{"startIndex": 0, "sectionBreak": {"sectionStyle": {"sectionType": "NEXT_PAGE",`+
			`"flipPageOrientation": true}}},`, 1)

	got, _ := Landed([]byte(read), pageSizeRequests())

	page := got.Checks[0]
	if !page.Held {
		t.Errorf("held = false on a document whose section flips nothing the rest does not: %+v", page)
	}
}

// pageSizeRead is styledRead answering with the page size beside its margins,
// which is what a document restyled by PageRequest comes back with.
func pageSizeRead() string {
	return strings.Replace(styledRead, `"documentStyle": {`,
		`"documentStyle": {"pageSize": {"width": {"magnitude": 595.28, "unit": "PT"},`+
			`"height": {"magnitude": 841.89, "unit": "PT"}},`, 1)
}

// pageSizeRequests is sentRequests with the page request naming a pageSize, the
// way PageRequest builds it. The orientation is only an override of a size that
// was asked for.
func pageSizeRequests() []map[string]any {
	out := sentRequests()
	out[0] = map[string]any{"updateDocumentStyle": map[string]any{
		"documentStyle": map[string]any{
			"pageSize":  map[string]any{"width": dim(595.28), "height": dim(841.89)},
			"marginTop": dim(62.35), "marginLeft": dim(51.05)},
		"fields": "pageSize,marginTop,marginLeft",
	}}
	return out
}

// A section break that overrides nothing the request set is no reason to
// withhold the answer. The default first section of every document carries a
// sectionStyle, and warning over one would be the cry-wolf warning this tool
// avoids everywhere else.
func TestASectionBreakSettingNoneOfTheFieldsSentStillAnswers(t *testing.T) {
	read := strings.Replace(styledRead, `"content": [`,
		`"content": [{"startIndex": 0, "sectionBreak": {"sectionStyle": {"sectionType": "CONTINUOUS",`+
			`"columnSeparatorStyle": "NONE"}}},`, 1)

	got, _ := Landed([]byte(read), sentRequests())

	page := got.Checks[0]
	if !page.Held {
		t.Errorf("held = false on a document whose section overrides nothing the page request set: %+v", page)
	}
}

// A section restating the margin the request sent overrides nothing a reader
// could see, so the check answers. A warning that fires on the working case is
// the defect MissingScopes was fixed for, and the case is a real one: a document
// gdoc built and published already carries the house geometry, and the ten-
// feature acceptance restyles a copy of one.
func TestASectionRestatingTheMarginThatWasSentStillAnswers(t *testing.T) {
	// Arrange: the section states 62.35pt top, which is exactly what
	// sentRequests asks for.
	read := strings.Replace(styledRead, `"content": [`,
		`"content": [{"startIndex": 0, "sectionBreak": {"sectionStyle": {"sectionType": "CONTINUOUS",`+
			`"marginTop": {"magnitude": 62.35, "unit": "PT"}}}},`, 1)

	// Act
	got, _ := Landed([]byte(read), sentRequests())

	// Assert
	page := got.Checks[0]
	if !page.Held {
		t.Errorf("held = false over a section restating the value that was sent: %+v", page)
	}
}

// And one field of two answering is not the whole answer: a section that
// restates the top margin and moves the left one still withholds it, naming the
// field it really overrode and not the one it agreed with.
func TestASectionNamesOnlyTheFieldItReallyOverrides(t *testing.T) {
	read := strings.Replace(styledRead, `"content": [`,
		`"content": [{"startIndex": 0, "sectionBreak": {"sectionStyle": {"sectionType": "CONTINUOUS",`+
			`"marginTop": {"magnitude": 62.35, "unit": "PT"},`+
			`"marginLeft": {"magnitude": 72, "unit": "PT"}}}},`, 1)

	got, _ := Landed([]byte(read), sentRequests())

	page := got.Checks[0]
	if page.Held {
		t.Errorf("held = true over a margin a section really overrides: %+v", page)
	}
	if !strings.Contains(page.Note, "marginLeft") {
		t.Errorf("note = %q, want it to name marginLeft", page.Note)
	}
	if strings.Contains(page.Note, "marginTop") {
		t.Errorf("note = %q, want it to leave out the field the section agrees with", page.Note)
	}
}

// The cell request names the table rather than a row, so the read-back reads a
// cell of that table. A row's cell count is not its width once cells are
// merged, which is why the request has no row in it to read.
func TestTheCellCheckReadsTheTableTheRequestNamed(t *testing.T) {
	got, _ := Landed([]byte(styledRead), sentRequests())

	cell := got.Checks[3]
	if !cell.Held {
		t.Errorf("held = false: %+v", cell)
	}
	if !strings.Contains(cell.Where, "the table at 20") {
		t.Errorf("where = %q, want the table the request named", cell.Where)
	}
}
