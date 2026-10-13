package reply

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// Every mark in this file is a literal. A test reading plaintext.DefaultMark
// back would follow it wherever somebody moved it, and what these tests ask is
// which characters reach a thread.

// sentContent is the words the one write carried, read off the fake's own
// record of the request rather than off the answer.
func sentContent(t *testing.T, f *fakeSession) string {
	t.Helper()
	if len(f.calls) == 0 {
		t.Fatal("nothing was sent")
	}
	var sent struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(f.calls[0].body, &sent); err != nil {
		t.Fatal(err)
	}
	return sent.Content
}

// TestReplyAddsTheMarkInEffect is the mark as a value: the caller hands over
// the words, and Post writes the mark this run was given in front of them, the
// way propose and annotate do.
func TestReplyAddsTheMarkInEffect(t *testing.T) {
	f := script()
	if _, err := Post(context.Background(), f, testDocID, testCommentID, "The 2026 register.", "∴"); err != nil {
		t.Fatal(err)
	}
	if got, want := sentContent(t, f), "∴ The 2026 register."; got != want {
		t.Errorf("the body sent is %q, want %q", got, want)
	}
}

// TestAnEmptyMarkWritesTheWordsAlone is decision B2's price paid in full: a
// colleague who set no mark gets the words with no space in front of them.
func TestAnEmptyMarkWritesTheWordsAlone(t *testing.T) {
	t.Run("reply", func(t *testing.T) {
		f := script()
		if _, err := Post(context.Background(), f, testDocID, testCommentID, "The 2026 register.", ""); err != nil {
			t.Fatal(err)
		}
		if got, want := sentContent(t, f), "The 2026 register."; got != want {
			t.Errorf("the body sent is %q, want %q", got, want)
		}
	})
}

// TestReplyAcceptsABodyWithNoMark is the rule reply used to hold upside down.
// The body arrives as words, because gdoc adds the mark, and a body with no
// mark on it is the normal call rather than the refused one.
func TestReplyAcceptsABodyWithNoMark(t *testing.T) {
	for _, body := range []string{
		"The 2026 register.",
		"Done. The register is now reviewed every six months.",
		"Two lines.\nThe second one says more.",
		"A hyphen bullet reads fine as plain text:\n- one\n- two",
	} {
		if err := Check(body, "⁘"); err != nil {
			t.Errorf("Check(%q) refused a plain reply: %v", body, err)
		}
	}
}

// TestABodyOpeningWithAKnownMarkIsWrittenOnceWithAWarning is the tolerance
// decision B4 grants for v2.11.0: a session that typed the mark itself, because
// that is what every release before this one asked for, gets one mark in the
// thread rather than two, and a warning saying gdoc adds it. Leading space and
// leading newline are the near misses, because a body that starts on its second
// line still opens with the mark.
func TestABodyOpeningWithAKnownMarkIsWrittenOnceWithAWarning(t *testing.T) {
	const inEffect = "⁘"
	for _, mark := range []string{"🤖", "⁘", "∴"} {
		for _, opening := range []struct {
			name string
			body string
		}{
			{"the mark and one space", mark + " The 2026 register."},
			{"a space before the mark", " " + mark + " The 2026 register."},
			{"a newline before the mark", "\n" + mark + " The 2026 register."},
			{"the mark with no space", mark + "The 2026 register."},
		} {
			t.Run(mark+", "+opening.name, func(t *testing.T) {
				f := script()
				got, err := Post(context.Background(), f, testDocID, testCommentID, opening.body, inEffect)
				if err != nil {
					t.Fatalf("a body carrying a mark was refused: %v", err)
				}
				sent := sentContent(t, f)
				if !strings.HasPrefix(sent, inEffect+" ") {
					t.Errorf("the body sent is %q, want it under %q", sent, inEffect)
				}
				if strings.Count(sent, mark) != strings.Count(inEffect, mark) {
					t.Errorf("the body sent is %q, and the mark is written twice", sent)
				}
				joined := strings.Join(got.Warnings, " ")
				if !strings.Contains(joined, mark) {
					t.Errorf("warnings = %v, and one should name the mark the body opened with", got.Warnings)
				}
				if !strings.Contains(joined, "gdoc adds the mark itself") {
					t.Errorf("warnings = %v, and one should say gdoc adds the mark", got.Warnings)
				}
				if !strings.Contains(joined, "v2.11.0") {
					t.Errorf("warnings = %v, and one should say how long this holds", got.Warnings)
				}
			})
		}
	}
}

// TestAnEmptyMarkStillTakesATypedMarkOff is the corner the two decisions make
// between them: the empty mark adds nothing, and a session that typed the robot
// anyway would otherwise have signed the reply for a colleague who chose not to.
func TestAnEmptyMarkStillTakesATypedMarkOff(t *testing.T) {
	f := script()
	got, err := Post(context.Background(), f, testDocID, testCommentID, "🤖 The 2026 register.", "")
	if err != nil {
		t.Fatal(err)
	}
	if sent, want := sentContent(t, f), "The 2026 register."; sent != want {
		t.Errorf("the body sent is %q, want %q", sent, want)
	}
	if !strings.Contains(strings.Join(got.Warnings, " "), "🤖") {
		t.Errorf("warnings = %v, and one should name the mark the body opened with", got.Warnings)
	}
}

// TestTheReadBackComparesTheMarkThatWasSent asks the read-back about the string
// that went out rather than about a constant. The fake's listing carries the
// reply under the robot, so the run writing the robot is confirmed and the run
// writing another mark is not.
func TestTheReadBackComparesTheMarkThatWasSent(t *testing.T) {
	t.Run("the mark that was sent is the mark the listing carries", func(t *testing.T) {
		f := script()
		got, err := Post(context.Background(), f, testDocID, testCommentID, goodWords, "🤖")
		if err != nil {
			t.Fatal(err)
		}
		if !got.Verified {
			t.Errorf("the reply is in the thread and Verified is false: %v", got.Warnings)
		}
	})
	t.Run("another mark is another reply", func(t *testing.T) {
		f := script()
		got, err := Post(context.Background(), f, testDocID, testCommentID, goodWords, "∴")
		if err != nil {
			t.Fatal(err)
		}
		if got.Verified {
			t.Error("the thread carries the words under another mark and Verified is true")
		}
	})
}
