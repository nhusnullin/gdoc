package edit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/propose"
)

const (
	testDocID = "DOC1"
	mark      = "⁘"
	yellowJS  = `{"backgroundColor":{"color":{"rgbColor":{"red":1,"green":0.9607843137254902,"blue":0.615686274509804}}}}`
	houseJS   = `{"backgroundColor":{"color":{"rgbColor":{"red":1,"green":0.8352941176470589}}}}`
)

// piece is one text run of a test document: its words and its style as the
// Docs API writes it.
type piece struct{ text, style string }

// document is a one-tab Docs answer whose paragraphs are the given runs,
// indexed from 1 the way Docs indexes a body.
func document(revision string, paragraphs ...[]piece) []byte {
	at := 1
	var paras []string
	for _, p := range paragraphs {
		var els []string
		for _, r := range p {
			style := r.style
			if style == "" {
				style = "{}"
			}
			end := at + len([]rune(r.text))
			els = append(els, fmt.Sprintf(`{"startIndex":%d,"endIndex":%d,"textRun":{"content":%q,"textStyle":%s}}`, at, end, r.text, style))
			at = end
		}
		paras = append(paras, `{"paragraph":{"elements":[`+strings.Join(els, ",")+`]}}`)
	}
	return []byte(`{"documentId":"DOC1","revisionId":"` + revision + `","tabs":[{"tabProperties":{"tabId":"t.0"},"documentTab":{"body":{"content":[` + strings.Join(paras, ",") + `]}}}]}`)
}

type call struct {
	method string
	body   json.RawMessage
}

// fakeSession answers each read with the next document in line and records
// every write.
type fakeSession struct {
	reads   [][]byte
	calls   []call
	postErr error
}

func (f *fakeSession) GetJSON(_ context.Context, _ string, into any) error {
	f.calls = append(f.calls, call{method: "GET"})
	if len(f.reads) == 0 {
		return errors.New("no read left")
	}
	next := f.reads[0]
	if len(f.reads) > 1 {
		f.reads = f.reads[1:]
	}
	return json.Unmarshal(next, into)
}

func (f *fakeSession) PostJSON(_ context.Context, _ string, body any, _ any) error {
	raw, _ := json.Marshal(body)
	f.calls = append(f.calls, call{method: "POST", body: raw})
	return f.postErr
}

func (f *fakeSession) posts() []call {
	var out []call
	for _, c := range f.calls {
		if c.method == "POST" {
			out = append(out, c)
		}
	}
	return out
}

type granted struct {
	revision string
	spans    []Span
}

func recorder(into *[]granted) Grant {
	return func(revision string, spans []Span) {
		*into = append(*into, granted{revision, append([]Span{}, spans...)})
	}
}

var (
	plainBefore = document("REV1", []piece{{"The old words stay here.\n", ""}})
	plainAfter  = document("REV2", []piece{{"The ", ""}, {"new words", yellowJS}, {" stay here.\n", ""}})
	change      = propose.Proposal{Quoted: "old words", Replacement: "new words", Why: "The register says new."}
)

func TestAnEditIsWordsForWords(t *testing.T) {
	block := propose.Proposal{Kind: propose.KindBlock, After: "The old words stay here.", Content: "A new paragraph.", Why: "Missing."}
	if err := Check(block, mark); err == nil || !strings.Contains(err.Error(), "suggestion") {
		t.Errorf("a block entry must be refused and sent to a run without --edit, got %v", err)
	}
	if err := Check(propose.Proposal{Quoted: "old words", Why: "Gone."}, mark); err == nil {
		t.Error("an edit that replaces words with nothing must be refused: there would be nothing to paint")
	}
	long := propose.Proposal{Quoted: "old words", Replacement: strings.Repeat("a", 301), Why: "Longer."}
	if err := Check(long, mark); err == nil {
		t.Error("a replacement of 301 units must be refused")
	}
	if err := Check(change, mark); err != nil {
		t.Errorf("a words entry must pass, got %v", err)
	}
}

func TestBatchDeletesInsertsAndPaintsExactlyTheInsertion(t *testing.T) {
	raw := Batch(docs.Range{Tab: "t.0", Start: 5, End: 14}, "ok 🙂", "REV1")
	want := `{"requests":[` +
		`{"deleteContentRange":{"range":{"endIndex":14,"startIndex":5}}},` +
		`{"insertText":{"location":{"index":5},"text":"ok 🙂"}},` +
		`{"updateTextStyle":{"fields":"backgroundColor","range":{"endIndex":10,"startIndex":5},"textStyle":{"backgroundColor":{"color":{"rgbColor":{"blue":0.615686274509804,"green":0.9607843137254902,"red":1}}}}}}` +
		`],"writeControl":{"requiredRevisionId":"REV1"}}`
	if string(raw) != want {
		t.Errorf("the batch is\n%s\nwant\n%s", raw, want)
	}
}

func TestApplyGrantsExactlyTheSpanItRead(t *testing.T) {
	f := &fakeSession{reads: [][]byte{plainBefore, plainAfter}}
	var grants []granted
	if _, err := Apply(context.Background(), f, testDocID, change, nil, mark, recorder(&grants)); err != nil {
		t.Fatal(err)
	}
	if len(grants) != 1 || grants[0].revision != "REV1" || len(grants[0].spans) != 1 || grants[0].spans[0] != (Span{5, 14}) {
		t.Fatalf("the grant was %+v, want one span [5,14) at REV1", grants)
	}
	if len(f.posts()) != 1 {
		t.Fatalf("an edit is one write, and %d went out", len(f.posts()))
	}
}

func TestAnEditIsReadBackPaintedAndNotPending(t *testing.T) {
	apply := func(after []byte) Result {
		f := &fakeSession{reads: [][]byte{plainBefore, after}}
		res, err := Apply(context.Background(), f, testDocID, change, nil, mark, func(string, []Span) {})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	if res := apply(plainAfter); !res.Verified || len(res.Warnings) != 0 {
		t.Errorf("a painted edit must verify, got %+v", res)
	}
	unpainted := document("REV2", []piece{{"The new words stay here.\n", ""}})
	if res := apply(unpainted); res.Verified || !res.Checks.Words || res.Checks.Painted {
		t.Errorf("words that landed with no yellow are written and not verified, got %+v", res)
	}
	pending := []byte(strings.Replace(string(plainAfter), `"content":"new words",`, `"content":"new words","suggestedInsertionIds":["kix.1"],`, 1))
	if res := apply(pending); res.Verified || res.Checks.Words {
		t.Errorf("words that landed as a suggestion are not an edit, got %+v", res)
	}
	bold := document("REV2", []piece{{"The ", ""}, {"new words", `{"bold":true,` + yellowJS[1:]}, {" stay here.\n", ""}})
	if res := apply(bold); res.Verified || res.Checks.Look {
		t.Errorf("words that took another style are reported, got %+v", res)
	}
	if res := apply(plainBefore); res.Verified || res.Checks.Words {
		t.Errorf("a document that still reads the old way is not verified, got %+v", res)
	}
}

func TestAnEditSpanCarryingABackgroundOrStyleEdgeIsRefused(t *testing.T) {
	for name, before := range map[string][]byte{
		"the quote is marked":       document("REV1", []piece{{"The ", ""}, {"old words", houseJS}, {" stay here.\n", ""}}),
		"the quote is a link":       document("REV1", []piece{{"The ", ""}, {"old words", `{"link":{"url":"https://example.org"}}`}, {" stay here.\n", ""}}),
		"the quote changes style":   document("REV1", []piece{{"The ", ""}, {"old ", `{"bold":true}`}, {"words", ""}, {" stay here.\n", ""}}),
		"the text in front is bold": document("REV1", []piece{{"The ", `{"bold":true}`}, {"old words stay here.\n", ""}}),
		"the text in front is a link": document("REV1", []piece{{"The ", `{"link":{"url":"https://example.org"}}`},
			{"old words", `{"link":{"url":"https://example.org"}}`}, {" stay here.\n", ""}}),
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeSession{reads: [][]byte{before}}
			var grants []granted
			_, err := Apply(context.Background(), f, testDocID, change, nil, mark, recorder(&grants))
			if err == nil {
				t.Fatal("the edit must be refused")
			}
			if len(f.posts()) != 0 || len(grants) != 0 {
				t.Fatalf("a refused edit grants nothing and writes nothing, got %d grants and %d writes", len(grants), len(f.posts()))
			}
		})
	}
	// A whole bold phrase behind bold text is plain ground: one look throughout.
	allBold := document("REV1", []piece{{"The old words stay here.\n", `{"bold":true}`}})
	f := &fakeSession{reads: [][]byte{allBold, allBold}}
	if _, err := Apply(context.Background(), f, testDocID, change, nil, mark, func(string, []Span) {}); err != nil {
		t.Errorf("words of one style among words of that style may be edited, got %v", err)
	}
}

// A quote part way into a plain run is plain ground, whatever the run before
// that one looks like: the character in front of the quote is in its own run.
func TestTheNeighbourIsTheCharacterBesideTheQuote(t *testing.T) {
	before := document("REV1", []piece{{"Title: ", `{"bold":true}`}, {"The old words stay here.\n", ""}})
	after := document("REV2", []piece{{"Title: ", `{"bold":true}`}, {"The ", ""}, {"new words", yellowJS}, {" stay here.\n", ""}})
	f := &fakeSession{reads: [][]byte{before, after}}
	res, err := Apply(context.Background(), f, testDocID, change, nil, mark, func(string, []Span) {})
	if err != nil || !res.Verified {
		t.Fatalf("a quote inside a plain run behind a bold one may be edited, got %+v, %v", res, err)
	}
	// The quote at the very start of its run, behind a bold run, is at an edge.
	atEdge := document("REV1", []piece{{"Title: ", `{"bold":true}`}, {"old words stay here.\n", ""}})
	g := &fakeSession{reads: [][]byte{atEdge}}
	if _, err := Apply(context.Background(), g, testDocID, change, nil, mark, func(string, []Span) {}); err == nil {
		t.Fatal("a quote that starts where bold text ends is at a style edge and must be refused")
	}
}

// Docs joins runs of one style, so new words written beside a passage an
// earlier edit painted come back inside one yellow run that starts before them.
func TestAnEditBesideAnEarlierYellowPassageStillVerifies(t *testing.T) {
	before := document("REV1", []piece{{"The ", ""}, {"new ", yellowJS}, {"old words stay here.\n", ""}})
	after := document("REV2", []piece{{"The ", ""}, {"new new words", yellowJS}, {" stay here.\n", ""}})
	f := &fakeSession{reads: [][]byte{before, after}}
	res, err := Apply(context.Background(), f, testDocID, change, nil, mark, func(string, []Span) {})
	if err != nil || !res.Verified {
		t.Fatalf("an edit beside an earlier yellow passage must verify, got %+v, %v", res, err)
	}
}

func TestAnEditAnswerNamingNoRevisionStopsTheRun(t *testing.T) {
	f := &fakeSession{reads: [][]byte{document("", []piece{{"The old words stay here.\n", ""}})}}
	if _, err := Apply(context.Background(), f, testDocID, change, nil, mark, func(string, []Span) {}); err == nil {
		t.Fatal("a read with no revision must stop the edit")
	}
	if len(f.posts()) != 0 {
		t.Fatal("nothing may be written against a revision nobody read")
	}
}

func TestARefusedWriteIsAnErrorAndIsNotRetried(t *testing.T) {
	f := &fakeSession{reads: [][]byte{plainBefore}, postErr: errors.New("400: the required revision ID does not match")}
	_, err := Apply(context.Background(), f, testDocID, change, nil, mark, func(string, []Span) {})
	if err == nil {
		t.Fatal("a write Docs refused is an error")
	}
	if len(f.posts()) != 1 {
		t.Fatalf("a refused write is sent once, and %d went out", len(f.posts()))
	}
}

func TestSeenClearsOnlyRunsOfExactlyTheEditYellow(t *testing.T) {
	before := document("REV5",
		[]piece{{"The ", ""}, {"new ", yellowJS}, {"words", yellowJS}, {" stay here.\n", ""}},
		[]piece{{"A house ", ""}, {"marked", houseJS}, {" word and ", ""}, {"another edit", yellowJS}, {".\n", ""}})
	after := document("REV6",
		[]piece{{"The new words stay here.\n", ""}},
		[]piece{{"A house ", ""}, {"marked", houseJS}, {" word and another edit.\n", ""}})
	f := &fakeSession{reads: [][]byte{before, after}}
	var grants []granted
	res, err := Clear(context.Background(), f, testDocID, recorder(&grants))
	if err != nil {
		t.Fatal(err)
	}
	if res.Cleared != 2 || res.Left != 0 || !res.Verified {
		t.Fatalf("the clear reports %+v, want two passages cleared, none left, verified", res)
	}
	want := []Span{{5, 14}, {50, 62}}
	if len(grants) != 1 || grants[0].revision != "REV5" || fmt.Sprint(grants[0].spans) != fmt.Sprint(want) {
		t.Fatalf("the grant was %+v, want %v at REV5", grants, want)
	}
	wantBody := `{"requests":[` +
		`{"updateTextStyle":{"fields":"backgroundColor","range":{"endIndex":14,"startIndex":5},"textStyle":{"backgroundColor":{}}}},` +
		`{"updateTextStyle":{"fields":"backgroundColor","range":{"endIndex":62,"startIndex":50},"textStyle":{"backgroundColor":{}}}}` +
		`],"writeControl":{"requiredRevisionId":"REV5"}}`
	if got := string(f.posts()[0].body); got != wantBody {
		t.Errorf("the clear batch is\n%s\nwant\n%s", got, wantBody)
	}
}

func TestADocumentWithNoYellowIsReadAndNotWritten(t *testing.T) {
	f := &fakeSession{reads: [][]byte{plainBefore}}
	res, err := Clear(context.Background(), f, testDocID, func(string, []Span) { t.Fatal("nothing to grant") })
	if err != nil || res.Cleared != 0 || !res.Verified || len(f.posts()) != 0 {
		t.Fatalf("got %+v, %v and %d writes; want nothing cleared, verified, no write", res, err, len(f.posts()))
	}
}

func TestAClearThatChangedAWordIsNotVerified(t *testing.T) {
	changed := document("REV2", []piece{{"The new WORDS stay here.\n", ""}})
	f := &fakeSession{reads: [][]byte{plainAfter, changed}}
	res, err := Clear(context.Background(), f, testDocID, func(string, []Span) {})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verified || len(res.Warnings) == 0 {
		t.Fatalf("a read-back with other words is reported and not verified, got %+v", res)
	}
}

func TestAYellowRunInsideASuggestionIsLeftAlone(t *testing.T) {
	pending := []byte(strings.Replace(string(plainAfter), `"content":"new words",`, `"content":"new words","suggestedInsertionIds":["kix.1"],`, 1))
	f := &fakeSession{reads: [][]byte{pending}}
	res, err := Clear(context.Background(), f, testDocID, func(string, []Span) {})
	if err != nil || res.Pending != 1 || res.Cleared != 0 || len(f.posts()) != 0 {
		t.Fatalf("got %+v, %v and %d writes; want one pending passage and no write", res, err, len(f.posts()))
	}
}

// A quote that stands twice is placed by the words near it, and the span the
// guard is granted is that one's: TestNearPlacesAQuoteInTheRowTheWordsName in
// internal/propose holds the search itself.
func TestAnEditOfARepeatedQuoteGrantsTheSpanNearNames(t *testing.T) {
	before := document("REV1",
		[]piece{{text: "Provider\n"}},
		[]piece{{text: "Provider of record\n"}})
	f := &fakeSession{reads: [][]byte{before, before}}
	var grants []granted
	p := propose.Proposal{Quoted: "Provider", Near: "of record", Replacement: "Supplier", Why: "the name changed"}
	if _, err := Apply(context.Background(), f, testDocID, p, nil, mark, recorder(&grants)); err != nil {
		t.Fatal(err)
	}
	if len(grants) != 1 || len(grants[0].spans) != 1 || grants[0].spans[0] != (Span{10, 18}) {
		t.Fatalf("the grant was %+v, want one span [10,18), the second Provider", grants)
	}
	p.Near = ""
	if _, err := Apply(context.Background(), &fakeSession{reads: [][]byte{before}}, testDocID, p, nil, mark, recorder(&grants)); err == nil {
		t.Fatal("without near a quote that stands twice was edited")
	}
}
