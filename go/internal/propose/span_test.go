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

// TestStartsAtReadsThePreviewAtTheSamePlace is the helper the second read-back
// check stands on: at the index the span was found, the text still begins with
// the words that were quoted.
func TestStartsAtReadsThePreviewAtTheSamePlace(t *testing.T) {
	d := document(t, "preview.json")
	r := docs.Range{Tab: "t.0", Start: 26, End: 43}
	if !StartsAt(d, r, "reviewed annually") {
		t.Error("the preview does not carry the quoted text where the span said it was")
	}
	if StartsAt(d, r, "reviewed every six months") {
		t.Error("the preview carries the replacement, which would mean a direct edit")
	}
}
