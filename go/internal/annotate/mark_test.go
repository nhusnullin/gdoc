package annotate

import (
	"context"
	"strings"
	"testing"
)

// Every mark in this file is a literal. A test reading plaintext.DefaultMark
// back would follow it wherever somebody moved it, and what these tests ask is
// which characters reach a thread.

// TestAnnotateWritesTheMarkInEffect is the mark as a value: Body puts the mark
// this run was given in front of the reason, and the write carries that same
// string.
func TestAnnotateWritesTheMarkInEffect(t *testing.T) {
	if got, want := Body("The register.", "∴"), "∴ The register."; got != want {
		t.Errorf("Body(%q, %q) = %q, want %q", "The register.", "∴", got, want)
	}
	body := decodeBatch(t, Batch(span(quoteStart, quoteEnd), Body(testAnnotation.Why, "∴"), ""))
	com := body["requests"].([]any)[0].(map[string]any)["insertComment"].(map[string]any)
	if got, want := com["content"], "∴ The 2026 register says quarterly."; got != want {
		t.Errorf("insertComment content = %v, want %q", got, want)
	}
}

// TestAnEmptyMarkWritesTheWordsAlone is decision B2's price paid in full: a
// colleague who set no mark gets the reason with no space in front of it.
func TestAnEmptyMarkWritesTheWordsAlone(t *testing.T) {
	t.Run("body", func(t *testing.T) {
		if got, want := Body("The register.", ""), "The register."; got != want {
			t.Errorf("Body(%q, %q) = %q, want %q", "The register.", "", got, want)
		}
	})
}

// TestAReasonOpeningWithAnyKnownMarkIsRefused is the writer's half of the two
// conventions: annotate adds the mark, so a reason carrying one already would
// land as two. Every known mark is refused whatever this run writes, and the
// mark in effect is refused as well.
func TestAReasonOpeningWithAnyKnownMarkIsRefused(t *testing.T) {
	const inEffect = "»"
	for _, mark := range []string{"🤖", "⁘", "∴", inEffect} {
		t.Run(mark, func(t *testing.T) {
			a := testAnnotation
			a.Why = mark + " The 2026 register says quarterly."
			err := a.Check(inEffect)
			if err == nil {
				t.Fatalf("a reason opening with %q was accepted", mark)
			}
			if !strings.Contains(err.Error(), mark) {
				t.Errorf("the refusal is %q, and it should name the mark it found", err)
			}
			if !strings.Contains(err.Error(), "gdoc adds it") {
				t.Errorf("the refusal is %q, and it should say gdoc adds the mark", err)
			}
		})
	}
}

// TestTheReadBackComparesTheMarkThatWasSent is both routes asked about the
// string that went out rather than about a constant. The fake's listing and
// export carry a comment under the robot, so the run that wrote the robot is
// confirmed and the run that wrote another mark is not.
func TestTheReadBackComparesTheMarkThatWasSent(t *testing.T) {
	t.Run("the mark that was sent is the mark the listing carries", func(t *testing.T) {
		f := script(t, "before.json", "batch-saved.json")
		res, err := Apply(context.Background(), f, testDocID, testAnnotation, "🤖")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Checks != (Checks{DriveListing: true, DocxAnchored: true}) {
			t.Errorf("checks = %+v, warnings = %v", res.Checks, res.Warnings)
		}
	})
	t.Run("another mark is another comment", func(t *testing.T) {
		f := script(t, "before.json", "batch-saved.json")
		res, err := Apply(context.Background(), f, testDocID, testAnnotation, "∴")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Checks.DriveListing || res.Checks.DocxAnchored {
			t.Errorf("checks = %+v, and neither route holds over a comment under another mark", res.Checks)
		}
		if !strings.Contains(strings.Join(res.Warnings, " "), "∴ The 2026 register says quarterly.") {
			t.Errorf("warnings = %v, and one should name the comment that was sent", res.Warnings)
		}
	})
}
