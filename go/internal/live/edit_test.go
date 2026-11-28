package live

// The acceptance for edit mode, GD-52 and GD-53: a document somebody handed in
// is edited directly through the edit level, the new words are painted, and
// the clear takes the paint off and leaves the house yellow alone.
//
// Two sessions, because two levels. The first creates the document, so the
// guard holds it at the full level and the setup may write anything. The
// second is handed the id the way a command is, at the suggest level, and
// raised by GrantEdit: every write it makes goes through the edit grammars and
// the span grant, against the real Docs API.

import (
	"context"
	"os"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/edit"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/propose"
)

func TestLiveEditModeWritesPaintsAndClears(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to create a document in the Drive test folder and edit it; %s names another folder", liveVar, writeVar, folderVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}
	owner := guard.NewPolicy()
	owner.AllowCreateIn(folder)
	s, err := gapi.Open(owner, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	ctx := context.Background()

	probeDoc(ctx, t, s, folder, "edit mode acceptance", func(id string) {
		url := "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
		setup := map[string]any{"requests": []map[string]any{
			req("insertText", map[string]any{"location": map[string]any{"index": 1}, "text": "The policy is reviewed annually by the board.\nA house marked word."}),
			req("updateTextStyle", map[string]any{"range": rng(55, 61), "textStyle": map[string]any{"backgroundColor": rgb(1, 213.0/255, 0)}, "fields": "backgroundColor"}),
		}}
		if err := s.PostJSON(ctx, url, setup, nil); err != nil {
			t.Fatalf("the setup write failed: %v", err)
		}

		handed := guard.NewPolicy()
		handed.AllowFile(id, guard.LevelSuggest)
		handed.GrantEdit(id)
		e, err := gapi.Open(handed, nil)
		if err != nil {
			t.Fatalf("the second session could not be opened: %v", err)
		}
		grantFor := func(kind guard.EditKind) edit.Grant {
			return func(revision string, spans []edit.Span) {
				out := make([]guard.Span, 0, len(spans))
				for _, sp := range spans {
					out = append(out, guard.Span{Start: sp.Start, End: sp.End})
				}
				handed.AllowEditSpans(kind, revision, out)
			}
		}

		change := propose.Proposal{Quoted: "reviewed annually", Replacement: "reviewed every six months", Why: "The register says twice a year."}
		res, err := edit.Apply(ctx, e, id, change, nil, "⁘", grantFor(guard.EditWords))
		if err != nil {
			t.Fatalf("the edit was refused: %v", err)
		}
		if !res.Verified {
			t.Errorf("the edit did not verify: %+v", res)
		}

		// A suggestion on the same document must be refused in the process:
		// one run is one mode.
		suggest := map[string]any{
			"requests":     []map[string]any{req("insertText", map[string]any{"location": map[string]any{"index": 1}, "text": "x"})},
			"writeControl": map[string]any{"writeMode": "SUGGEST"},
		}
		if err := e.PostJSON(ctx, url, suggest, nil); err == nil {
			t.Error("a SUGGEST batch on a document at the edit level went out")
		}

		cleared, err := edit.Clear(ctx, e, id, grantFor(guard.EditClear))
		if err != nil {
			t.Fatalf("the clear was refused: %v", err)
		}
		if cleared.Cleared != 1 || !cleared.Verified {
			t.Errorf("the clear reports %+v, want one passage cleared and verified", cleared)
		}

		after, err := docs.Fetch(ctx, s, id)
		if err != nil {
			t.Fatalf("the last read failed: %v", err)
		}
		var text strings.Builder
		house := 0
		for _, r := range blockRunsOf(after) {
			text.WriteString(r.Text)
			if r.Background == "#FFD500" {
				house++
			}
			if strings.EqualFold(r.Background, edit.Yellow) {
				t.Errorf("the run %q still carries the edit yellow", r.Text)
			}
		}
		if want := "The policy is reviewed every six months by the board.\nA house marked word.\n"; text.String() != want {
			t.Errorf("the document reads %q, want %q", text.String(), want)
		}
		if house != 1 {
			t.Errorf("the house yellow is on %d runs after the clear, want 1: the clear must leave the author's mark alone", house)
		}
	})
}
