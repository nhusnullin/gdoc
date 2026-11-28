package live

// What does Docs do with a direct edit that paints its own insertion, and what
// takes the paint off again?
//
// Asked on 2026-11-28 for backlog tasks GD-52 and GD-53, edit mode, before the
// guard's grammar for it was written:
//
//   - One batch, not SUGGEST: delete the quoted words, insert the new ones at
//     the same index, paint exactly the insertion. Does the colour cover the
//     insertion and nothing beside it, and is anything left pending?
//   - writeControl.requiredRevisionId with a revision the document has moved
//     past: refused, and with what?
//   - Which spelling clears a background: the field named in the mask and left
//     unset, or the field set to an empty colour?
//   - Does clearing one colour's runs leave a run of another colour alone?
//
// Like the other probes it asserts almost nothing: a measurement that fails
// the build when Google answers differently has already decided the answer. It
// fails only when it cannot create, set up or trash. The document is the
// guard's own create, so every write here is allowed at the full level.

import (
	"context"
	"os"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
)

// editSetup is the document the probe starts from. Index 1 is the start of
// "The". "old words" sits at [5, 14), and "marked" at [33, 39).
const editSetup = "The old words stay here.\nA house marked word.\nLast paragraph.\n"

func TestLiveEditModeProbe(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to create a document in the Drive test folder and probe it; %s names another folder", liveVar, writeVar, folderVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}
	p := guard.NewPolicy()
	p.AllowCreateIn(folder)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	ctx := context.Background()
	edit := rgb(1, 245.0/255, 157.0/255) // #FFF59D
	house := rgb(1, 213.0/255, 0)        // #FFD500
	paint := func(start, end int, colour map[string]any) map[string]any {
		return req("updateTextStyle", map[string]any{"range": rng(start, end),
			"textStyle": map[string]any{"backgroundColor": colour}, "fields": "backgroundColor"})
	}

	probeDoc(ctx, t, s, folder, "edit mode: a painted direct edit, a stale revision, two ways to clear", func(id string) {
		url := "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
		post := func(what string, revision string, requests ...map[string]any) bool {
			body := map[string]any{"requests": requests}
			if revision != "" {
				body["writeControl"] = map[string]any{"requiredRevisionId": revision}
			}
			if err := s.PostJSON(ctx, url, body, nil); err != nil {
				t.Logf("  %s: REFUSED: %s", what, docsReason(err))
				return false
			}
			t.Logf("  %s: carried", what)
			return true
		}
		show := func(what string) *docs.Document {
			d, err := docs.Fetch(ctx, s, id)
			if err != nil {
				t.Fatalf("the read back after %s failed: %v", what, err)
			}
			t.Logf("  after %s, revision %q:", what, d.RevisionID)
			for _, r := range blockRunsOf(d) {
				t.Logf("    [%d,%d) %q background=%q ins=%v del=%v", r.StartIndex, r.EndIndex, r.Text, r.Background, r.InsertionIDs, r.DeletionIDs)
			}
			return d
		}

		if !post("setup", "",
			req("insertText", map[string]any{"location": map[string]any{"index": 1}, "text": strings.TrimSuffix(editSetup, "\n")}),
			paint(33, 39, house)) {
			t.Fatal("the setup write failed")
		}
		before := show("setup")

		// One: the edit batch at the revision just read.
		newWords := "new words, longer"
		post("the edit batch at the fresh revision", before.RevisionID,
			req("deleteContentRange", map[string]any{"range": rng(5, 14)}),
			req("insertText", map[string]any{"location": map[string]any{"index": 5}, "text": newWords}),
			paint(5, 5+len(newWords), edit))
		after := show("the edit")

		// Two: the same shape at the revision the edit has just moved past.
		post("an edit batch at the stale revision", before.RevisionID,
			req("insertText", map[string]any{"location": map[string]any{"index": 1}, "text": "STALE "}),
			paint(1, 7, edit))
		show("the stale attempt")

		// Three: clear by naming the field and leaving it unset.
		post("clear, field named and left unset", after.RevisionID,
			req("updateTextStyle", map[string]any{"range": rng(5, 5+len(newWords)),
				"textStyle": map[string]any{}, "fields": "backgroundColor"}))
		cleared := show("the unset clear")

		// Four: paint again, then clear by setting an empty colour.
		post("paint again", cleared.RevisionID, paint(5, 5+len(newWords), edit))
		again := show("painting again")
		post("clear, an empty colour set", again.RevisionID,
			req("updateTextStyle", map[string]any{"range": rng(5, 5+len(newWords)),
				"textStyle": map[string]any{"backgroundColor": map[string]any{}}, "fields": "backgroundColor"}))
		show("the empty-colour clear")
	})
}

// blockRunsOf is every text run of the first tab's body, paragraphs only,
// which is all the probe document holds.
func blockRunsOf(d *docs.Document) []docs.Run {
	var out []docs.Run
	if len(d.Tabs) == 0 {
		return out
	}
	for _, b := range d.Tabs[0].Body {
		if b.Paragraph != nil {
			out = append(out, b.Paragraph.Runs...)
		}
	}
	return out
}
