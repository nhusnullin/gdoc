package propose

import (
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// near-table.json holds "Provider" eight times: twice in the opening paragraph,
// once in the first cell of each of two table rows, and once inside the second
// row's other cell. No quote of it alone can be placed, and a cell has no more
// of the sentence to quote.

func TestNearPlacesAQuoteInTheRowTheWordsName(t *testing.T) {
	r, err := FindSpanNear(document(t, "near-table.json"), "Provider", "Holds the crypto relationship")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := docs.Range{Tab: "t.0", Start: 57, End: 65}
	if r != want {
		t.Errorf("range = %+v, want %+v", r, want)
	}
}

// The paragraph the words stand in is asked before its row, so a quote the
// paragraph holds is that one, whatever the cell beside it reads.
func TestNearAsksItsOwnParagraphBeforeItsRow(t *testing.T) {
	r, err := FindSpanNear(document(t, "near-table.json"), "Provider", "of record")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := docs.Range{Tab: "t.0", Start: 118, End: 126}
	if r != want {
		t.Errorf("range = %+v, want %+v", r, want)
	}
}

func TestNearRefusesAQuoteThatStandsTwiceInThePlace(t *testing.T) {
	_, err := FindSpanNear(document(t, "near-table.json"), "Provider", "is set out below")
	if err == nil {
		t.Fatal("a quote that stands twice beside the words was placed")
	}
	if !strings.Contains(err.Error(), "2 times") || !strings.Contains(err.Error(), "is set out below") {
		t.Errorf("error = %q, and it should count the quote and name the words", err)
	}
}

func TestNearRefusesWordsThatStandTwice(t *testing.T) {
	_, err := FindSpanNear(document(t, "near-table.json"), "role", "Provider")
	if err == nil {
		t.Fatal("words that stand eight times named a place")
	}
	if !strings.Contains(err.Error(), "near") || !strings.Contains(err.Error(), "8 times") {
		t.Errorf("error = %q, and it should say the near words stand 8 times", err)
	}
}

func TestNearRefusesWordsThatAreNotThere(t *testing.T) {
	_, err := FindSpanNear(document(t, "near-table.json"), "Provider", "Holds the fiat relationship")
	if err == nil {
		t.Fatal("words that are not in the document named a place")
	}
	if !strings.Contains(err.Error(), "near") || !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, and it should say the near words were not found", err)
	}
}

func TestNearRefusesAQuoteThatIsNotBesideTheWords(t *testing.T) {
	_, err := FindSpanNear(document(t, "near-table.json"), "Provider", "reviewed each year")
	if err == nil {
		t.Fatal("a quote was placed away from the words that name its place")
	}
	if !strings.Contains(err.Error(), "not found") || !strings.Contains(err.Error(), "reviewed each year") {
		t.Errorf("error = %q, and it should say the quote is not beside the words", err)
	}
}

// Without near nothing changed: exactly once in the document.
func TestWithoutNearAQuoteThatStandsTwiceIsStillRefused(t *testing.T) {
	_, err := FindSpanNear(document(t, "near-table.json"), "Provider", "")
	if err == nil || !strings.Contains(err.Error(), "8 times") {
		t.Fatalf("error = %v, and a quote standing 8 times is refused with its count", err)
	}
	if !strings.Contains(err.Error(), "near") {
		t.Errorf("error = %q, and it should name near as the way to say which one", err)
	}
}

// The preview route asks the place, not the tab. The other copies of the
// quote are still in the document after a direct edit took the one that was
// meant, and a question about the whole tab would pass on them.
func TestCarriesNearAsksThePlaceAndNotTheTab(t *testing.T) {
	const near = "Holds the crypto relationship"
	if held, placed := CarriesNear(document(t, "near-table.json"), "t.0", "Provider", near); !held || !placed {
		t.Errorf("held, placed = %v, %v, and the quote is beside the words", held, placed)
	}
	edited := document(t, "near-table-preview-edited.json")
	if !Carries(edited, "t.0", "Provider") {
		t.Fatal("the fixture should still hold the quote elsewhere")
	}
	if held, placed := CarriesNear(edited, "t.0", "Provider", near); held || !placed {
		t.Errorf("held, placed = %v, %v: a direct edit took the quote from the place, and the place is still there", held, placed)
	}
	if _, placed := CarriesNear(edited, "t.0", "Provider", "Provider"); placed {
		t.Error("near words that stand more than once named a place")
	}
}

// A quote that stands twice in the paragraph the near words are in is
// refused there. Widening to the row would not help, and it must not be
// tried: the row's first cell holds the quote once, and placing that one
// would pick for the caller.
func TestAQuoteTwiceInNearsParagraphIsRefusedNotWidenedToTheRow(t *testing.T) {
	_, err := FindSpanNear(document(t, "near-table.json"), "Provider", "pays the")
	if err == nil {
		t.Fatal("a quote standing twice in the near words' own paragraph was placed")
	}
	if !strings.Contains(err.Error(), "2 times") {
		t.Errorf("error = %q, and it should count the two in the paragraph", err)
	}
}

func TestCheckRefusesNearOnABlock(t *testing.T) {
	block := Proposal{Kind: KindBlock, After: "a", Content: "b", Why: "w", Near: "c"}
	if err := block.Check("⁘"); err == nil || !strings.Contains(err.Error(), "near") {
		t.Errorf("error = %v, and a block entry carrying near is refused by name", err)
	}
	blank := Proposal{Quoted: "a", Replacement: "b", Why: "w", Near: "  "}
	if err := blank.Check("⁘"); err == nil || !strings.Contains(err.Error(), "near") {
		t.Errorf("error = %v, and a blank near is refused by name", err)
	}
}
