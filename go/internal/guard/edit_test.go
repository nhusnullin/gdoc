package guard

// This file holds one subject: the edit level, which is the one place a
// handed-in document's words may change without a suggestion.
//
// Nail's decision of 2026-11-28, DECISIONS.md: a request that says edit mode in
// words changes the text directly and paints the new words yellow, and a second
// command takes the yellow off. The level is bounded by two exact grammars and
// by spans the command granted out of its own read, so every test here is
// written as an attack first, the way inplace_test.go is.

import (
	"fmt"
	"strings"
	"testing"
)

const (
	editURL      = "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate"
	editRevision = "REV1"
	// The yellow as the wire carries it: #FFF59D, channel over 255.
	editYellowJSON = `{"color":{"rgbColor":{"red":1,"green":0.9607843137254902,"blue":0.615686274509804}}}`
)

// editing returns the policy of one edit run: DOC1 handed in, raised to the
// edit level, and granted the span [5,14) at REV1.
func editing(t *testing.T) *Policy {
	t.Helper()
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.GrantEdit("DOC1")
	p.AllowEditSpans(EditWords, editRevision, []Span{{Start: 5, End: 14}})
	return p
}

// editBatch is the one batch an edit makes, written out because what this file
// judges is exactly those bytes.
func editBatch(revision string, start, end int, text string, paintStart, paintEnd int, colour string) []byte {
	return []byte(fmt.Sprintf(
		`{"requests":[`+
			`{"insertText":{"location":{"index":%d},"text":%q}},`+
			`{"deleteContentRange":{"range":{"startIndex":%d,"endIndex":%d}}},`+
			`{"updateTextStyle":{"range":{"startIndex":%d,"endIndex":%d},"textStyle":{"backgroundColor":%s},"fields":"backgroundColor"}}`+
			`],"writeControl":{"requiredRevisionId":%q}}`,
		end, text, start, end, paintStart, paintEnd, colour, revision))
}

func goodEdit() []byte {
	return editBatch(editRevision, 5, 14, "new words", 5, 14, editYellowJSON)
}

func clearBatch(revision string, spans ...Span) []byte {
	var reqs []string
	for _, s := range spans {
		reqs = append(reqs, fmt.Sprintf(
			`{"updateTextStyle":{"range":{"startIndex":%d,"endIndex":%d},"textStyle":{"backgroundColor":{}},"fields":"backgroundColor"}}`,
			s.Start, s.End))
	}
	return []byte(`{"requests":[` + strings.Join(reqs, ",") + `],"writeControl":{"requiredRevisionId":"` + revision + `"}}`)
}

func carried(t *testing.T, p *Policy, body []byte) {
	t.Helper()
	if err := p.Judge("POST", mustURL(t, editURL), body); err != nil {
		t.Fatalf("the batch must be carried, and it was refused: %v", err)
	}
}

func refusedEdit(t *testing.T, p *Policy, body []byte, why string) {
	t.Helper()
	if p.Judge("POST", mustURL(t, editURL), body) == nil {
		t.Fatalf("%s must be refused, and it was carried: %s", why, body)
	}
}

func TestAGrantedEditBatchIsCarried(t *testing.T) {
	carried(t, editing(t), goodEdit())
}

func TestADirectEditWithoutTheEditGrantIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.AllowEditSpans(EditWords, editRevision, []Span{{Start: 5, End: 14}})
	refusedEdit(t, p, goodEdit(), "an edit batch on a document nobody raised to the edit level")
}

func TestALevelEditBatchMeetsAGrammarWhateverWriteModeSays(t *testing.T) {
	suggest := []byte(`{"requests":[{"insertText":{"location":{"index":5},"text":"x"}}],"writeControl":{"writeMode":"SUGGEST"}}`)
	refusedEdit(t, editing(t), suggest, "a SUGGEST batch on a document at the edit level")

	both := []byte(strings.Replace(string(goodEdit()), `"writeControl":{`, `"writeControl":{"writeMode":"SUGGEST",`, 1))
	refusedEdit(t, editing(t), both, "an edit batch that also names a write mode")

	refusedEdit(t, editing(t), []byte(`{"writeControl":{"requiredRevisionId":"REV1"}}`), "a batch naming no requests")
	refusedEdit(t, editing(t), []byte(`{"requests":[],"writeControl":{"requiredRevisionId":"REV1"}}`), "a batch with an empty request list")
	extra := []byte(strings.Replace(string(goodEdit()), `"writeControl"`, `"includeTabsContent":true,"writeControl"`, 1))
	refusedEdit(t, editing(t), extra, "an edit batch with a third top-level field")
}

func TestAnEditBatchWithoutTheGrantedRevisionIsRefused(t *testing.T) {
	refusedEdit(t, editing(t), editBatch("REV2", 5, 14, "new words", 5, 14, editYellowJSON), "an edit at another revision")
	none := []byte(strings.Replace(string(goodEdit()), `,"writeControl":{"requiredRevisionId":"REV1"}`, ``, 1))
	refusedEdit(t, editing(t), none, "an edit naming no revision")
	target := []byte(strings.Replace(string(goodEdit()), `requiredRevisionId`, `targetRevisionId`, 1))
	refusedEdit(t, editing(t), target, "an edit naming a target revision, which merges rather than refuses")
}

func TestAnEditDeletesOnlyAGrantedSpan(t *testing.T) {
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 15, "new words", 5, 14, editYellowJSON), "a delete one unit past the granted span")
	refusedEdit(t, editing(t), editBatch(editRevision, 4, 14, "new words", 4, 13, editYellowJSON), "a delete starting before the granted span")

	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.GrantEdit("DOC1")
	refusedEdit(t, p, goodEdit(), "an edit on a run that granted no span")

	withSegment := []byte(strings.Replace(string(goodEdit()), `{"startIndex":5,"endIndex":14}}}`, `{"startIndex":5,"endIndex":14,"segmentId":"kix.h1"}}}`, 1))
	refusedEdit(t, editing(t), withSegment, "a delete naming a segment")
	withTab := []byte(strings.Replace(string(goodEdit()), `{"index":14}`, `{"index":14,"tabId":"t.1"}`, 1))
	refusedEdit(t, editing(t), withTab, "an insert naming a tab")
}

func TestAnEditSpanOverTheCeilingIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.GrantEdit("DOC1")
	p.AllowEditSpans(EditWords, editRevision, []Span{{Start: 5, End: 306}})
	text := strings.Repeat("a", 10)
	refusedEdit(t, p, editBatch(editRevision, 5, 306, text, 5, 15, editYellowJSON), "a delete of 301 units")

	long := strings.Repeat("a", 301)
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, long, 5, 306, editYellowJSON), "an insert of 301 units")
}

func TestNothingAtLevelEditPaintsOutsideItsOwnInsertion(t *testing.T) {
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, "new words", 5, 15, editYellowJSON), "a paint one unit past the insertion")
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, "new words", 4, 14, editYellowJSON), "a paint starting before the insertion")
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, "new words", 5, 14, `{"color":{"rgbColor":{"red":1,"green":1,"blue":0}}}`), "a paint in another colour")

	// An emoji is two UTF-16 units, and the paint is counted in those.
	carried(t, editing(t), editBatch(editRevision, 5, 14, "ok 🙂", 5, 10, editYellowJSON))
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, "ok 🙂", 5, 9, editYellowJSON), "a paint counted in runes rather than UTF-16 units")

	star := []byte(strings.Replace(string(goodEdit()), `"fields":"backgroundColor"`, `"fields":"*"`, 1))
	refusedEdit(t, editing(t), star, "a paint whose mask is a star")
	bold := []byte(strings.Replace(string(goodEdit()), `"textStyle":{`, `"textStyle":{"bold":true,`, 1))
	refusedEdit(t, editing(t), bold, "a paint that also sets bold")
}

func TestAnEditInsertsWordsInsideOneParagraph(t *testing.T) {
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, "", 5, 5, editYellowJSON), "an edit inserting nothing")
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, "two\nlines", 5, 14, editYellowJSON), "an insert carrying a paragraph mark")
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, "a\vb", 5, 8, editYellowJSON), "an insert carrying a line break")

	// Docs strips a private-use character on insert, so two units sent are one
	// unit stored and the paint would cover a character nobody inserted.
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, "\ue000x", 5, 7, editYellowJSON), "an insert carrying a character Docs strips")
	half := []byte(strings.Replace(string(editBatch(editRevision, 5, 14, "Zx", 5, 7, editYellowJSON)), `"text":"Zx"`, `"text":"\ud83dx"`, 1))
	refusedEdit(t, editing(t), half, "an insert carrying half a surrogate pair")
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, "\ufeffx", 5, 7, editYellowJSON), "an insert carrying a byte order mark")

	elsewhere := []byte(strings.Replace(string(goodEdit()), `{"index":14}`, `{"index":13}`, 1))
	refusedEdit(t, editing(t), elsewhere, "an insert away from the end of the delete")
}

func TestAnEditBatchIsExactlyItsThreeRequests(t *testing.T) {
	noDelete := []byte(`{"requests":[` +
		`{"insertText":{"location":{"index":5},"text":"new words"}},` +
		`{"updateTextStyle":{"range":{"startIndex":5,"endIndex":14},"textStyle":{"backgroundColor":` + editYellowJSON + `},"fields":"backgroundColor"}}` +
		`],"writeControl":{"requiredRevisionId":"REV1"}}`)
	refusedEdit(t, editing(t), noDelete, "an insert with no delete in front of it")

	fourth := []byte(strings.Replace(string(goodEdit()), `],"writeControl"`, `,{"deleteContentRange":{"range":{"startIndex":5,"endIndex":14}}}],"writeControl"`, 1))
	refusedEdit(t, editing(t), fourth, "an edit batch with a fourth request")

	reordered := []byte(`{"requests":[` +
		`{"deleteContentRange":{"range":{"startIndex":5,"endIndex":14}}},` +
		`{"insertText":{"location":{"index":14},"text":"new words"}},` +
		`{"updateTextStyle":{"range":{"startIndex":5,"endIndex":14},"textStyle":{"backgroundColor":` + editYellowJSON + `},"fields":"backgroundColor"}}` +
		`],"writeControl":{"requiredRevisionId":"REV1"}}`)
	refusedEdit(t, editing(t), reordered, "an edit batch in another order")
}

func TestNoSuggestionKindOrNamedRangeCarriesAtLevelEdit(t *testing.T) {
	p := editing(t)
	p.AllowReject("kix.s1")
	p.AllowMarker(markerName, markerStart, markerEnd)
	reject := []byte(`{"requests":[{"rejectSuggestion":{"suggestionId":"kix.s1"}}],"writeControl":{"requiredRevisionId":"REV1"}}`)
	refusedEdit(t, p, reject, "a granted rejectSuggestion on a document at the edit level")
	named := []byte(`{"requests":[{"createNamedRange":` + markerRequest(markerName, markerStart, markerEnd) + `}],"writeControl":{"requiredRevisionId":"REV1"}}`)
	refusedEdit(t, p, named, "a granted createNamedRange on a document at the edit level")
	style := []byte(`{"requests":[{"updateParagraphStyle":{"range":{"startIndex":5,"endIndex":14},"paragraphStyle":{"alignment":"CENTER"},"fields":"alignment"}}],"writeControl":{"requiredRevisionId":"REV1"}}`)
	refusedEdit(t, p, style, "a paragraph style on a document at the edit level")
}

// clearing returns the policy of one `gdoc seen` run: DOC1 at the edit level,
// with two spans granted for a clear at REV1.
func clearing(t *testing.T) *Policy {
	t.Helper()
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.GrantEdit("DOC1")
	p.AllowEditSpans(EditClear, editRevision, []Span{{Start: 5, End: 14}, {Start: 40, End: 44}})
	return p
}

func TestAClearChangesNoCharacterAndTouchesOnlyGrantedSpans(t *testing.T) {
	carried(t, clearing(t), clearBatch(editRevision, Span{5, 14}, Span{40, 44}))
	carried(t, clearing(t), clearBatch(editRevision, Span{40, 44}))
	refusedEdit(t, clearing(t), clearBatch(editRevision, Span{5, 14}, Span{20, 30}), "a clear over a span nobody granted")
	refusedEdit(t, clearing(t), clearBatch("REV2", Span{5, 14}), "a clear at another revision")

	unset := []byte(strings.Replace(string(clearBatch(editRevision, Span{5, 14})), `"textStyle":{"backgroundColor":{}}`, `"textStyle":{}`, 1))
	refusedEdit(t, clearing(t), unset, "a clear that leaves the field unset rather than setting it empty")
	wide := []byte(strings.Replace(string(clearBatch(editRevision, Span{5, 14})), `"fields":"backgroundColor"`, `"fields":"backgroundColor,bold"`, 1))
	refusedEdit(t, clearing(t), wide, "a clear whose mask names a second field")

	mixed := []byte(`{"requests":[` +
		`{"updateTextStyle":{"range":{"startIndex":5,"endIndex":14},"textStyle":{"backgroundColor":{}},"fields":"backgroundColor"}},` +
		`{"deleteContentRange":{"range":{"startIndex":5,"endIndex":14}}}` +
		`],"writeControl":{"requiredRevisionId":"REV1"}}`)
	refusedEdit(t, clearing(t), mixed, "a clear with a delete behind it")
}

// TestASpanGrantIsForOneGrammar: `gdoc seen` says no word changes, and the
// guard holds it. Spans granted for a clear carry no edit, and the other way.
func TestASpanGrantIsForOneGrammar(t *testing.T) {
	refusedEdit(t, clearing(t), goodEdit(), "an edit over spans granted for a clear")
	refusedEdit(t, editing(t), clearBatch(editRevision, Span{5, 14}), "a clear over a span granted for an edit")

	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.GrantEdit("DOC1")
	p.AllowEditSpans(EditWords, editRevision, []Span{{Start: 5, End: 14}, {Start: 40, End: 44}})
	refusedEdit(t, p, goodEdit(), "an edit grant naming two spans")
	p.AllowEditSpans(EditKind(0), editRevision, []Span{{Start: 5, End: 14}})
	refusedEdit(t, p, goodEdit(), "a grant naming no grammar")
}

// TestASpanGrantIsSpentByTheBatchItCarries: one grant, one batch. The same
// body judged again is refused, so a replay is never carried.
func TestASpanGrantIsSpentByTheBatchItCarries(t *testing.T) {
	p := editing(t)
	carried(t, p, goodEdit())
	refusedEdit(t, p, goodEdit(), "the same edit a second time")

	q := clearing(t)
	carried(t, q, clearBatch(editRevision, Span{5, 14}))
	refusedEdit(t, q, clearBatch(editRevision, Span{40, 44}), "a second clear on a grant the first one spent")

	// A refused batch spends nothing: the run's own batch still goes.
	r := editing(t)
	refusedEdit(t, r, editBatch(editRevision, 5, 15, "new words", 5, 14, editYellowJSON), "a batch over another span")
	carried(t, r, goodEdit())
}

func TestAllowFileCannotOpenTheEditDoor(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelEdit)
	if _, known := p.level("DOC1"); known {
		t.Fatal("AllowFile must not hand out the edit level; GrantEdit is the one door")
	}
	if len(p.Warnings()) == 0 {
		t.Fatal("the refused hand-in must be recorded for the envelope")
	}
}

func TestGrantEditNeverAdmitsAnUnknownID(t *testing.T) {
	p := NewPolicy()
	p.GrantEdit("DOC1")
	if _, known := p.level("DOC1"); known {
		t.Fatal("GrantEdit raises an id already in the set and admits none")
	}
}

func TestGrantEditLeavesACreatedDocumentAtFull(t *testing.T) {
	p := NewPolicy()
	p.Learn("DOC1")
	p.GrantEdit("DOC1")
	if lvl, _ := p.level("DOC1"); lvl != LevelFull {
		t.Fatalf("a created document stays at the full level, and it is at %s", lvl)
	}
	if len(p.Warnings()) == 0 {
		t.Fatal("a grant that changed nothing says so")
	}
}

func TestGrantEditRefusesAnInPlaceID(t *testing.T) {
	p := granted(t)
	p.GrantEdit("DOC1")
	if lvl, _ := p.level("DOC1"); lvl != LevelInPlace {
		t.Fatalf("a document granted for a restyle stays at the in-place level, and it is at %s", lvl)
	}
	if len(p.Warnings()) == 0 {
		t.Fatal("a grant that changed nothing says so")
	}

	q := editing(t)
	q.GrantInPlace("DOC1")
	if lvl, _ := q.level("DOC1"); lvl != LevelEdit {
		t.Fatalf("a document granted for an edit stays at the edit level, and it is at %s", lvl)
	}
}

func TestTheEditGrantDiesWithThePolicy(t *testing.T) {
	carried(t, editing(t), goodEdit())
	fresh := NewPolicy()
	fresh.AllowFile("DOC1", LevelSuggest)
	refusedEdit(t, fresh, goodEdit(), "an edit under a new policy, which no earlier run can have granted")
}

func TestASecondSpanGrantReplacesTheFirst(t *testing.T) {
	p := editing(t)
	p.AllowEditSpans(EditWords, "REV2", []Span{{Start: 20, End: 24}})
	refusedEdit(t, p, goodEdit(), "an edit over the span the second grant replaced")
	carried(t, p, editBatch("REV2", 20, 24, "four", 20, 24, editYellowJSON))
}

func TestAnUnreadableSpanGrantTakesTheStandingOneBack(t *testing.T) {
	for name, grant := range map[string]func(p *Policy){
		"no revision":     func(p *Policy) { p.AllowEditSpans(EditWords, "", []Span{{Start: 5, End: 14}}) },
		"no span":         func(p *Policy) { p.AllowEditSpans(EditWords, editRevision, nil) },
		"an empty span":   func(p *Policy) { p.AllowEditSpans(EditWords, editRevision, []Span{{Start: 5, End: 5}}) },
		"a span before 1": func(p *Policy) { p.AllowEditSpans(EditWords, editRevision, []Span{{Start: 0, End: 4}}) },
	} {
		t.Run(name, func(t *testing.T) {
			p := editing(t)
			grant(p)
			refusedEdit(t, p, goodEdit(), "an edit after a span grant the guard could not read")
			if len(p.Warnings()) == 0 {
				t.Fatal("the unreadable grant must be recorded for the envelope")
			}
		})
	}
}

func TestTheEditLevelDoesNotReachTheDriveFile(t *testing.T) {
	p := editing(t)
	u := mustURL(t, "https://www.googleapis.com/drive/v3/files/DOC1")
	if p.Judge("PATCH", u, []byte(`{"trashed":true}`)) == nil {
		t.Fatal("an edit run must not trash or rename the document it edits")
	}
}

func TestTheEditLevelNamesItselfInWords(t *testing.T) {
	if got := LevelEdit.String(); got != "edit" {
		t.Fatalf("LevelEdit reads %q in a refusal, want %q", got, "edit")
	}
}

// TestTheEditYellowIsNoHouseColour states the colour as a literal: a test that
// read the constant would follow it wherever somebody moved it.
func TestTheEditYellowIsNoHouseColour(t *testing.T) {
	if EditYellow != "#FFF59D" {
		t.Fatalf("the edit yellow is %q, want #FFF59D", EditYellow)
	}
	for _, house := range []string{"#FFD500", "#3DD67A", "#FFFF00"} {
		if strings.EqualFold(EditYellow, house) {
			t.Fatalf("the edit yellow is the house colour %s; gdoc seen would then clear a mark the author made", house)
		}
	}
}

// The order is the look. Docs gives inserted text the style of the character
// in front of the insert, so the words go in at the end of the span they
// replace, while the old words still stand there, and take the look of the
// last of them (MEASURED.md, 2026-11-28, "An insert at the end of the words it
// replaces"). The five tests below hold the one index and the one order.
func TestAnEditInsertsAtTheGrantedSpansEndBeforeItDeletes(t *testing.T) {
	carried(t, editing(t), goodEdit())
	want := `{"requests":[{"insertText":{"location":{"index":14},"text":"new words"}},{"deleteContentRange":{"range":{"startIndex":5,"endIndex":14}}},`
	if !strings.HasPrefix(string(goodEdit()), want) {
		t.Fatalf("the carried batch is not the one this test names: %s", goodEdit())
	}
}

func oldOrder(insertAt int) []byte {
	return []byte(fmt.Sprintf(`{"requests":[`+
		`{"deleteContentRange":{"range":{"startIndex":5,"endIndex":14}}},`+
		`{"insertText":{"location":{"index":%d},"text":"new words"}},`+
		`{"updateTextStyle":{"range":{"startIndex":5,"endIndex":14},"textStyle":{"backgroundColor":`+editYellowJSON+`},"fields":"backgroundColor"}}`+
		`],"writeControl":{"requiredRevisionId":"REV1"}}`, insertAt))
}

func TestTheDeleteFirstOrderIsRefused(t *testing.T) {
	refusedEdit(t, editing(t), oldOrder(5), "the delete-first batch edit mode sent before 2026-11-28")
	refusedEdit(t, editing(t), oldOrder(14), "a delete-first batch inserting at the span's end")
}

func TestAnEditInsertingAtTheSpanStartIsRefused(t *testing.T) {
	atStart := []byte(strings.Replace(string(goodEdit()), `{"index":14}`, `{"index":5}`, 1))
	refusedEdit(t, editing(t), atStart, "an insert at the start of the span, where it takes the neighbour's look")
}

// After the delete the units [end, end+n) are the text behind the edit.
func TestAnEditPaintingAtTheInsertIndexIsRefused(t *testing.T) {
	refusedEdit(t, editing(t), editBatch(editRevision, 5, 14, "new words", 14, 23, editYellowJSON), "a paint that starts at the insert index")
}

func TestAnEditDeletingItsOwnInsertionIsRefused(t *testing.T) {
	both := []byte(strings.Replace(string(goodEdit()), `"deleteContentRange":{"range":{"startIndex":5,"endIndex":14}}`, `"deleteContentRange":{"range":{"startIndex":5,"endIndex":23}}`, 1))
	refusedEdit(t, editing(t), both, "a delete reaching over the words the batch just inserted")
}
