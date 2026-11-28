package live

// Does an insert at the end of the words it replaces take their look?
//
// Asked on 2026-11-28 for backlog task GD-55, before the guard's edit grammar
// changed its order. Docs says inserted text takes the style of the character
// before the insertion index. The edit batch of GD-52 deleted first and
// inserted at the span's start, so the new words took the neighbour's look,
// and edit mode refused every quote at a style edge. The claim measured here:
// insert at the span's end first, then delete the span, and the new words
// take the look of the last quoted character, whatever stands beside them.
//
// Three shapes, each one batch: a bold term that opens a paragraph with plain
// text behind it, plain words directly behind a bold word, and plain words
// that open a paragraph directly in front of a bold word.
//
// Like the other probes it asserts almost nothing, and fails only when it
// cannot create, set up or trash. The document is the guard's own create.

import (
	"context"
	"os"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
)

// editLookSetup: "Crypto terms." is [1,14) and bold; "beta" is [41,45) and
// bold. "Alpha " is [35,41) and " gamma" is [45,51).
const editLookSetup = "Crypto terms. The contract stays.\nAlpha beta gamma."

func TestLiveEditLookProbe(t *testing.T) {
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
	yellow := rgb(1, 245.0/255, 157.0/255)
	bold := func(start, end int) map[string]any {
		return req("updateTextStyle", map[string]any{"range": rng(start, end),
			"textStyle": map[string]any{"bold": true}, "fields": "bold"})
	}

	probeDoc(ctx, t, s, folder, "edit mode: does an insert at the span's end keep the look", func(id string) {
		url := "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
		post := func(what string, requests ...map[string]any) {
			if err := s.PostJSON(ctx, url, map[string]any{"requests": requests}, nil); err != nil {
				t.Fatalf("%s: REFUSED: %s", what, docsReason(err))
			}
		}
		show := func(what string) {
			d, err := docs.Fetch(ctx, s, id)
			if err != nil {
				t.Fatalf("the read back after %s failed: %v", what, err)
			}
			t.Logf("  after %s:", what)
			for _, r := range blockRunsOf(d) {
				t.Logf("    [%d,%d) %q look=%q background=%q", r.StartIndex, r.EndIndex, r.Text, r.Look, r.Background)
			}
		}
		// endFirst is the proposed grammar: insert at the span's end, delete
		// the span, paint the words where the span started.
		endFirst := func(what string, start, end int, words string) {
			post(what,
				req("insertText", map[string]any{"location": map[string]any{"index": end}, "text": words}),
				req("deleteContentRange", map[string]any{"range": rng(start, end)}),
				req("updateTextStyle", map[string]any{"range": rng(start, start+len(words)),
					"textStyle": map[string]any{"backgroundColor": yellow}, "fields": "backgroundColor"}))
			show(what)
		}

		post("setup",
			req("insertText", map[string]any{"location": map[string]any{"index": 1}, "text": editLookSetup}),
			bold(1, 14), bold(41, 45))
		show("setup")

		// Last span first, so no edit moves a span still to come.
		endFirst("plain words behind a bold word", 45, 51, " delta now")
		endFirst("plain words opening a paragraph in front of a bold word", 35, 41, "Omega ")
		endFirst("a bold term opening a paragraph, plain text behind it", 1, 13, "Exchange service terms")
	})
}
