package propose

import (
	"context"
	"strings"
	"testing"
)

// Every mark in this file is a literal. A test reading plaintext.DefaultMark
// back would follow it wherever somebody moved it, and what these tests ask is
// which characters reach a thread.

// commentOf is the content of the insertComment request in a batch, which is
// the one string in a write that records who wrote the comment.
func commentOf(t *testing.T, raw []byte) string {
	t.Helper()
	reqs := requestsOf(t, raw)
	for _, r := range reqs {
		com, ok := r["insertComment"].(map[string]any)
		if !ok {
			continue
		}
		text, ok := com["content"].(string)
		if !ok {
			t.Fatalf("insertComment content = %v, want a string", com["content"])
		}
		return text
	}
	t.Fatal("the batch carries no insertComment")
	return ""
}

// blockPlacement is the everyday block placement the golden batches use, so a
// mark test can ask the block writer the same question as the words writer.
func blockPlacement(t *testing.T) (Placement, []Para) {
	t.Helper()
	content := blockContent(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	return place, content
}

// TestProposeWritesTheMarkInEffect is the mark as a value: both writers put the
// mark this run was given in front of the reason, and neither holds a character
// of its own.
func TestProposeWritesTheMarkInEffect(t *testing.T) {
	if got, want := commentOf(t, Batch(span(26, 43), testProposal, "∴")), "∴ the policy says twice a year"; got != want {
		t.Errorf("the words proposal wrote %q, want %q", got, want)
	}
	place, content := blockPlacement(t)
	if got, want := commentOf(t, BlockBatch(place, content, testWhy, "", "∴")), "∴ the limits section was missing"; got != want {
		t.Errorf("the block proposal wrote %q, want %q", got, want)
	}
}

// TestAnEmptyMarkWritesTheWordsAlone is decision B2's price paid in full: a
// colleague who set no mark gets the reason with no space in front of it, and
// nothing else changes.
func TestAnEmptyMarkWritesTheWordsAlone(t *testing.T) {
	t.Run("words", func(t *testing.T) {
		if got, want := commentOf(t, Batch(span(26, 43), testProposal, "")), "the policy says twice a year"; got != want {
			t.Errorf("content = %q, want %q", got, want)
		}
	})
	t.Run("block", func(t *testing.T) {
		place, content := blockPlacement(t)
		if got, want := commentOf(t, BlockBatch(place, content, testWhy, "", "")), "the limits section was missing"; got != want {
			t.Errorf("content = %q, want %q", got, want)
		}
	})
}

// TestAReasonOpeningWithAnyKnownMarkIsRefused is the writer's half of the two
// conventions: propose adds the mark, so a reason carrying one already would
// land as two. Every known mark is refused whatever this run writes, because a
// doubled mark is a lie about who wrote the words in any of them, and the mark
// in effect is refused as well.
func TestAReasonOpeningWithAnyKnownMarkIsRefused(t *testing.T) {
	const inEffect = "»"
	for _, mark := range []string{"🤖", "⁘", "∴", inEffect} {
		t.Run(mark, func(t *testing.T) {
			p := testProposal
			p.Why = mark + " the policy says twice a year"
			err := p.Check(inEffect)
			if err == nil {
				t.Fatalf("a reason opening with %q was accepted", mark)
			}
			if !strings.Contains(err.Error(), mark) {
				t.Errorf("the refusal is %q, and it should name the mark it found", err)
			}
			if !strings.Contains(err.Error(), "gdoc adds it") {
				t.Errorf("the refusal is %q, and it should say gdoc adds the mark", err)
			}

			b := testBlock
			b.Why = p.Why
			if err := b.Check(inEffect); err == nil {
				t.Errorf("the block kind accepted a reason opening with %q", mark)
			}
		})
	}
}

// TestTheReadBackComparesTheMarkThatWasSent is the third route asked about the
// string that went out rather than about a constant. The fake's export carries
// a comment under the robot, so the run that wrote the robot is confirmed and
// the run that wrote another mark is not, and the warning names what was sent.
func TestTheReadBackComparesTheMarkThatWasSent(t *testing.T) {
	t.Run("the mark that was sent is the mark the export carries", func(t *testing.T) {
		f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
		res, err := Apply(context.Background(), f, testDocID, testProposal, "🤖")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Checks.DocxAnchored {
			t.Errorf("docx_anchored = false where the export carries the comment: %v", res.Warnings)
		}
	})
	t.Run("another mark is another comment", func(t *testing.T) {
		f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
		res, err := Apply(context.Background(), f, testDocID, testProposal, "∴")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Checks.DocxAnchored {
			t.Error("docx_anchored = true where the export carries a comment under another mark")
		}
		if !strings.Contains(strings.Join(res.Warnings, " "), "∴ the policy says twice a year") {
			t.Errorf("warnings = %v, and one should name the comment that was sent", res.Warnings)
		}
	})
}
