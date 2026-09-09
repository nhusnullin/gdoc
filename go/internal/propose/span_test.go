package propose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/docs"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func document(t *testing.T, name string) *docs.Document {
	t.Helper()
	d, err := docs.Parse(fixture(t, name))
	if err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	return d
}

func TestFindSpanGivesTheRangeOfTheOneMatch(t *testing.T) {
	r, err := FindSpan(document(t, "before.json"), "reviewed annually")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := docs.Range{Tab: "t.0", Start: 26, End: 43}
	if r != want {
		t.Errorf("range = %+v, want %+v", r, want)
	}
}

// TestFindSpanCountsInUTF16CodeUnits is the index hazard the whole package is
// built around. The paragraph opens with a robot, which is one rune and two
// UTF-16 code units, and the Docs API counts the units. A search that counted
// runes or bytes would put the span one or two characters out, and a
// deleteContentRange one character out cuts a word in half.
func TestFindSpanCountsInUTF16CodeUnits(t *testing.T) {
	r, err := FindSpan(document(t, "emoji.json"), "reviewed annually")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := docs.Range{Tab: "t.0", Start: 4, End: 21}
	if r != want {
		t.Errorf("range = %+v, want %+v", r, want)
	}
}

func TestFindSpanRefusesTextThatIsNotThere(t *testing.T) {
	_, err := FindSpan(document(t, "before.json"), "reviewed monthly")
	if err == nil {
		t.Fatal("text that is not in the document was found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, and it should say the quoted text was not found", err)
	}
}

func TestFindSpanRefusesTextThatOccursTwice(t *testing.T) {
	_, err := FindSpan(document(t, "twice.json"), "reviewed annually")
	if err == nil {
		t.Fatal("an ambiguous quote was accepted")
	}
	if !strings.Contains(err.Error(), "2 times") || !strings.Contains(err.Error(), "quote more") {
		t.Errorf("error = %q, and it should say how many times it occurs and what to do about it", err)
	}
}

// TestFindSpanRefusesAMatchInsideAPendingSuggestion keeps a proposal off the
// top of a proposal. Suggested text is somebody's pending intention, and a
// deleteContentRange over it makes a suggestion about a suggestion, which no
// reader can act on.
func TestFindSpanRefusesAMatchInsideAPendingSuggestion(t *testing.T) {
	_, err := FindSpan(document(t, "suggested.json"), "reviewed annually")
	if err == nil {
		t.Fatal("a match inside a pending suggestion was accepted")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q; suggested text is not text a proposal may stand on", err)
	}
}

// TestCarriesReadsThePreviewByWordsRatherThanByIndex is the helper the second
// read-back check stands on: with the suggestions hidden the quoted words are
// still somewhere in the tab, and the replacement is not.
func TestCarriesReadsThePreviewByWordsRatherThanByIndex(t *testing.T) {
	d := document(t, "preview.json")
	if !Carries(d, "t.0", "reviewed annually") {
		t.Error("the preview does not carry the quoted text")
	}
	if Carries(d, "t.0", "reviewed every six months") {
		t.Error("the preview carries the replacement, which would mean a direct edit")
	}
	if Carries(d, "t.9", "reviewed annually") {
		t.Error("a tab the document does not have answered for one it does")
	}
}

// TestFindSpanRefusesAQuoteThatRunsAcrossAFootnoteMark is the contiguity rule.
// The walk indexes text runs and skips everything else, but the document numbers
// what it skipped, so "reviewed annually" either side of a footnote mark reads
// as one string here and is two spans in the document. A range built across it
// is longer than the words in it, and the deleteContentRange marks the footnote
// mark for deletion along with them. withdraw.Span refuses two spans with
// somebody else's content between them for the same reason.
func TestFindSpanRefusesAQuoteThatRunsAcrossAFootnoteMark(t *testing.T) {
	_, err := FindSpan(document(t, "footnote.json"), "reviewed annually")
	if err == nil {
		t.Fatal("a quote spanning a footnote mark was accepted, and the delete would take the mark with it")
	}
	if !strings.Contains(err.Error(), "runs across") {
		t.Errorf("error = %q, and it should say the quote runs across content the span cannot cover", err)
	}
}

// TestFindSpanPlacesAQuoteThatEndsAtAFootnoteMark is the boundary the
// contiguity rule is easiest to get wrong on. The quote ends exactly where the
// footnote mark begins, so it crosses nothing, and its span is the words alone.
// Reading the position of the byte behind the quote answers with the start of
// the run after the mark, which is a unit too far, and the quote is then refused
// for crossing a hole it only touches.
func TestFindSpanPlacesAQuoteThatEndsAtAFootnoteMark(t *testing.T) {
	r, err := FindSpan(document(t, "footnote.json"), "The supplier register is reviewed")
	if err != nil {
		t.Fatalf("a quote ending where the footnote mark begins was refused: %v", err)
	}
	want := docs.Range{Tab: "t.0", Start: 1, End: 34}
	if r != want {
		t.Errorf("range = %+v, want %+v", r, want)
	}
}

// TestFindSpanRefusesAQuoteWhoseOtherOccurrenceCrossesAFootnoteMark keeps the
// exactly-once rule whole. A crossing occurrence is still an occurrence: placing
// the contiguous one would pick for the caller on a document where the quoted
// words appear twice, and Carries would still find the crossing copy after a
// direct edit took the other, reporting a suggestion that is not there.
func TestFindSpanRefusesAQuoteWhoseOtherOccurrenceCrossesAFootnoteMark(t *testing.T) {
	_, err := FindSpan(document(t, "footnote-twice.json"), "reviewed annually")
	if err == nil {
		t.Fatal("a quote with a second, crossing occurrence was placed on the first")
	}
	if !strings.Contains(err.Error(), "quote more of the sentence") {
		t.Errorf("error = %q, and it should ask for more of the sentence", err)
	}
	if !strings.Contains(err.Error(), "does not index") {
		t.Errorf("error = %q, and it should say the other occurrence crosses content the read does not index", err)
	}
}

// TestFindSpanStillPlacesAQuoteBesideAFootnoteMark is the other direction. The
// contiguity rule refuses a quote that crosses the mark, and it must not refuse
// one that merely sits next to it: the paragraph is ordinary text either side.
func TestFindSpanStillPlacesAQuoteBesideAFootnoteMark(t *testing.T) {
	r, err := FindSpan(document(t, "footnote.json"), "operations team")
	if err != nil {
		t.Fatalf("a quote that crosses nothing was refused: %v", err)
	}
	want := docs.Range{Tab: "t.0", Start: 52, End: 67}
	if r != want {
		t.Errorf("range = %+v, want %+v", r, want)
	}
}

// TestAQuoteCrossingAChipIsRefused is the contiguity rule across a change that
// moved the ground under it. A person chip, a date chip, a calendar link, an
// auto text, a page break, a column break and a horizontal rule used to vanish
// at decode, so they were holes this walk skipped without ever seeing them.
// They are runs now, and index still skips them because they carry no text, so
// the rule has to hold for the reason rather than by luck: the document numbers
// the chip, the text either side of it is contiguous in the walk and is two
// spans in the document, and a deleteContentRange over one span would take the
// chip with it.
func TestAQuoteCrossingAChipIsRefused(t *testing.T) {
	// "Owner: A reviews it" reads as one string in the walk. In the document
	// the chip sits at 8..9 between the two halves.
	d := chipDocument(t)
	_, err := FindSpan(d, "Owner:  reviews")
	if err == nil {
		t.Fatal("a quote spanning a person chip was accepted, and the delete would take the chip with it")
	}
	if !strings.Contains(err.Error(), "does not index") {
		t.Errorf("error = %q, and it should say the quote crosses content the read does not index", err)
	}
}

// The other side of the same rule: a quote that stops where the chip begins
// crosses nothing, so it is placed, and its span is the words alone.
func TestAQuoteEndingWhereAChipBeginsIsPlaced(t *testing.T) {
	r, err := FindSpan(chipDocument(t), "Owner: ")
	if err != nil {
		t.Fatalf("a quote ending where the chip begins was refused: %v", err)
	}
	if want := (docs.Range{Tab: "t.0", Start: 1, End: 8}); r != want {
		t.Errorf("range = %+v, want %+v", r, want)
	}
}

// chipDocument is one paragraph with a person chip in the middle of it, at the
// indexes the document numbers it with.
func chipDocument(t *testing.T) *docs.Document {
	t.Helper()
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":18,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[
		   {"startIndex":1,"endIndex":8,"textRun":{"content":"Owner: "}},
		   {"startIndex":8,"endIndex":9,"person":{"personId":"kix.p1",
		     "personProperties":{"name":"A Placeholder","email":"placeholder@example.com"}}},
		   {"startIndex":9,"endIndex":18,"textRun":{"content":" reviews\n"}}]}}]}}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return d
}
