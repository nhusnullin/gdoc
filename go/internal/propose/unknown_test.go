package propose

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// This file is the bound on an answer that was lost. The batch was written and
// nothing came back saying what became of it, so the proposal is neither in the
// document nor out of it as far as gdoc can say. Everything here is about saying
// that rather than guessing which.

// lostAnswerError is what the session hands back for a request that was written
// and whose answer never came. The writer packages ask by behaviour, so the fake
// answers by behaviour too.
type lostAnswerError struct{ error }

func (lostAnswerError) Unknown() bool { return true }

// TestSendNamesALostAnswer is both kinds of proposal through the one write. An
// answer that was lost is not a write that never left, and it is not a write
// Docs took either: it raises, and the Result says the outcome is unknown so the
// command can say so rather than reporting a failure somebody acts on.
func TestSendNamesALostAnswer(t *testing.T) {
	t.Run("a words proposal", func(t *testing.T) {
		f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
		f.failAt[1] = lostAnswerError{errors.New("read: connection reset by peer")}

		res, err := Apply(context.Background(), f, testDocID, testProposal)

		assertLost(t, f, res, err, "reviewed annually")
	})
	t.Run("a block", func(t *testing.T) {
		f := blockApplyScript(t, "block-body.json", "block-after-inline.json", "block-body.json",
			"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))
		f.failAt[1] = lostAnswerError{errors.New("read: connection reset by peer")}

		res, err := ApplyBlock(context.Background(), f, testDocID, testBlock)

		assertLost(t, f, res, err, testBlock.After)
	})
}

// assertLost is the whole of what a lost answer leaves behind, asked of one
// proposal: it raises, the sentence says the write went out and the answer did
// not, the outcome is unknown, and no read-back ran.
func assertLost(t *testing.T, f *fakeSession, res Result, err error, quote string) {
	t.Helper()
	if err == nil {
		t.Fatal("a proposal whose answer was lost came back as success")
	}
	if !strings.Contains(err.Error(), "its answer was lost") {
		t.Errorf("error = %q, and it must say the answer was lost rather than that the write failed", err)
	}
	// The sentence may not claim the proposal is not there. That is the one
	// thing gdoc does not know on this path.
	if strings.Contains(err.Error(), "could not be written") {
		t.Errorf("error = %q, and the proposal was written", err)
	}
	if res.Outcome != OutcomeUnknown {
		t.Errorf("outcome = %q, want %q", res.Outcome, OutcomeUnknown)
	}
	if res.Quote() != quote {
		t.Errorf("quote = %q, want %q, so the command can name which proposal it was", res.Quote(), quote)
	}
	if res.Verified {
		t.Error("Verified = true on a proposal nobody could read back")
	}
	// The read and the write, and nothing after them: a read-back of a change
	// gdoc cannot say is there answers a question it was not asked.
	if len(f.calls) != 2 {
		t.Errorf("the fake saw %d calls, want the read and the write: %+v", len(f.calls), f.calls)
	}
}

// TestAnAnswerDocsTookIsStillNotUnknown keeps the outcome to the case it means.
// A failure the session marked as sent is a change that is in the document, and
// reporting it as unknown would send somebody to read a document that gdoc
// already knows the answer about.
func TestAnAnswerDocsTookIsStillNotUnknown(t *testing.T) {
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)
	f.failAt[1] = acceptedError{errors.New("the answer could not be read: unexpected EOF")}

	res, err := Apply(context.Background(), f, testDocID, testProposal)
	if err != nil {
		t.Fatalf("a batch the server accepted was reported as never sent: %v", err)
	}
	if res.Outcome != "" {
		t.Errorf("outcome = %q, and Docs took this batch", res.Outcome)
	}
}
