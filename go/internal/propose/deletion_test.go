package propose

import (
	"strings"
	"testing"
)

// TestADeletionRefusalSaysToQuoteANeighbour: a plain deletion is not a shape
// the write has, and the refusal names the shape that is, so the next attempt
// is the right one.
func TestADeletionRefusalSaysToQuoteANeighbour(t *testing.T) {
	err := Proposal{Quoted: "not ", Replacement: "", Why: "the limit applies"}.Check()
	if err == nil {
		t.Fatal("an empty replacement was accepted")
	}
	want := "quote a neighbouring word with it, and keep that word in the replacement"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the refusal must say %q:\n%s", want, err)
	}
}
