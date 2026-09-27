package live

// What does Docs do with a block of new paragraphs proposed in SUGGEST mode?
//
// Asked on 2026-09-27 for docs/backlog/propose-cannot-add-paragraphs.md, whose
// design leaves three questions to measure before the plan:
//
//   - One batch that inserts several paragraphs and styles them: one
//     suggestion id, or one per paragraph, or one per request?
//   - Does deleteParagraphBullets land as a suggestion?
//   - Text inserted at the end of a list item: does it arrive bulleted?
//
// And the replace shape: a deleteContentRange over whole paragraphs, in the
// same batch as the insert that replaces them.
//
// Like the other probes it asserts almost nothing: a measurement that fails
// the build when Google answers differently has already decided the answer. It
// fails only when it cannot create, set up or trash. Each case gets a
// throwaway document of its own, set up with a direct write the guard allows
// because the guard's own create made it, and then asked in SUGGEST mode,
// which is the mode propose sends in.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"gdoc/internal/gapi"
	"gdoc/internal/guard"
)

// blockSetup is the document every case starts from. Index 1 is the start of
// "Intro". The list item is a real bullet, made by the setup batch.
const blockSetup = "Intro paragraph.\nOld one.\nOld two.\nA list item.\nClosing paragraph.\n"

func TestLiveBlockProposalProbe(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to create documents in the Drive test folder and probe them; %s names another folder", liveVar, writeVar, folderVar)
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

	// Indexes in blockSetup: each paragraph's [start, end) with its newline.
	intro := [2]int{1, 18}   // "Intro paragraph.\n"
	oldOne := [2]int{18, 27} // "Old one.\n"
	oldTwo := [2]int{27, 36} // "Old two.\n"
	item := [2]int{36, 49}   // "A list item.\n"

	block := "New heading\nNew body text.\nFirst new item\nSecond new item\n"
	// blockRequests styles a block whose first character is at at.
	blockRequests := func(at int) []map[string]any {
		h := [2]int{at, at + len("New heading\n")}
		b := [2]int{h[1], h[1] + len("New body text.\n")}
		l := [2]int{b[1], b[1] + len("First new item\nSecond new item\n")}
		return []map[string]any{
			req("updateParagraphStyle", map[string]any{"range": rng(h[0], h[1]), "paragraphStyle": map[string]any{"namedStyleType": "HEADING_2"}, "fields": "namedStyleType"}),
			req("updateParagraphStyle", map[string]any{"range": rng(b[0], l[1]), "paragraphStyle": map[string]any{"namedStyleType": "NORMAL_TEXT"}, "fields": "namedStyleType"}),
			req("updateTextStyle", map[string]any{"range": rng(b[0], b[0]+3), "textStyle": map[string]any{"bold": true}, "fields": "bold"}),
			req("createParagraphBullets", map[string]any{"range": rng(l[0], l[1]), "bulletPreset": "BULLET_DISC_CIRCLE_SQUARE"}),
		}
	}

	// One: a block after an ordinary paragraph. Inserted at the start of the
	// next paragraph, so the text arrives as whole paragraphs of its own.
	blockCase(ctx, t, s, folder, "a block after a plain paragraph", func(id string) error {
		at := intro[1]
		reqs := append([]map[string]any{req("insertText", map[string]any{"location": map[string]any{"index": at}, "text": block})}, blockRequests(at)...)
		return suggestBatch(ctx, s, id, reqs...)
	})

	// Two: the same block after the list item, inserted at the item's end,
	// before its newline, so the item's paragraph mark ends up behind the
	// block. Nothing states a bullet or a style: this is the inheritance
	// question alone.
	blockCase(ctx, t, s, folder, "plain text inserted at a list item's end, nothing stated", func(id string) error {
		at := item[1] - 1
		return suggestBatch(ctx, s, id, req("insertText", map[string]any{"location": map[string]any{"index": at}, "text": "\nInherited one.\nInherited two."}))
	})

	// Three: the same, with deleteParagraphBullets over the new paragraphs.
	blockCase(ctx, t, s, folder, "the same, with deleteParagraphBullets over the new text", func(id string) error {
		at := item[1] - 1
		text := "\nInherited one.\nInherited two."
		return suggestBatch(ctx, s, id,
			req("insertText", map[string]any{"location": map[string]any{"index": at}, "text": text}),
			req("deleteParagraphBullets", map[string]any{"range": rng(at+1, at+len(text))}),
		)
	})

	// Five and six: text inserted at the start of a list item, which is where
	// the prelude goes when a document opens with a list, and where a block
	// goes when the paragraph after its anchor is a list item.
	blockCase(ctx, t, s, folder, "whole paragraphs inserted at a list item's start, nothing stated", func(id string) error {
		return suggestBatch(ctx, s, id, req("insertText", map[string]any{"location": map[string]any{"index": item[0]}, "text": "Before one.\nBefore two.\n"}))
	})
	blockCase(ctx, t, s, folder, "the same, with deleteParagraphBullets over the new paragraphs", func(id string) error {
		text := "Before one.\nBefore two.\n"
		return suggestBatch(ctx, s, id,
			req("insertText", map[string]any{"location": map[string]any{"index": item[0]}, "text": text}),
			req("deleteParagraphBullets", map[string]any{"range": rng(item[0], item[0]+len(text))}),
		)
	})

	// Seven: a block after the last paragraph, where there is no next
	// paragraph to insert in front of. It goes in before the last paragraph's
	// newline, so the last new paragraph owns the body's final mark.
	blockCase(ctx, t, s, folder, "a block after the last paragraph, styles stated", func(id string) error {
		last := len(blockSetup) // the body's final newline, one past the text
		text := "\nTail heading\nTail body."
		return suggestBatch(ctx, s, id,
			req("insertText", map[string]any{"location": map[string]any{"index": last}, "text": text}),
			req("updateParagraphStyle", map[string]any{"range": rng(last+1, last+len("\nTail heading\n")), "paragraphStyle": map[string]any{"namedStyleType": "HEADING_2"}, "fields": "namedStyleType"}),
		)
	})

	// Four: a replace. The two old paragraphs are deleted whole, from the
	// start of the first to the end of the last, and the block goes in at the
	// start of the paragraph behind them, in one batch. The insert comes first
	// so the delete's indexes are the old ones shifted by the block's length.
	blockCase(ctx, t, s, folder, "a replace: two whole paragraphs out, the block in", func(id string) error {
		at := oldOne[0]
		reqs := []map[string]any{req("insertText", map[string]any{"location": map[string]any{"index": at}, "text": block})}
		reqs = append(reqs, blockRequests(at)...)
		n := len(block)
		reqs = append(reqs, req("deleteContentRange", map[string]any{"range": rng(oldOne[0]+n, oldTwo[1]+n)}))
		return suggestBatch(ctx, s, id, reqs...)
	})
}

// blockCase makes a document holding blockSetup, with the fourth paragraph a
// real bullet, runs one SUGGEST batch against it, and logs what came back.
func blockCase(ctx context.Context, t *testing.T, s *gapi.Session, folder, name string, run func(id string) error) {
	probeDoc(ctx, t, s, folder, name, func(id string) {
		setup := map[string]any{"requests": []map[string]any{
			req("insertText", map[string]any{"location": map[string]any{"index": 1}, "text": strings.TrimSuffix(blockSetup, "\n")}),
			req("createParagraphBullets", map[string]any{"range": rng(36, 48), "bulletPreset": "BULLET_DISC_CIRCLE_SQUARE"}),
		}}
		url := "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
		if err := s.PostJSON(ctx, url, setup, nil); err != nil {
			t.Fatalf("the setup write failed: %v", err)
		}
		if err := run(id); err != nil {
			t.Logf("  REFUSED: %s", docsReason(err))
			return
		}
		after, err := readInline(ctx, s, id)
		if err != nil {
			t.Errorf("the read back failed: %v", err)
			return
		}
		logBlock(t, after)
	})
}

// logBlock prints every paragraph with its style, its bullet and the
// suggestion ids on it, then every distinct id by the key it was found under.
func logBlock(t *testing.T, d map[string]any) {
	body, _ := d["body"].(map[string]any)
	content, _ := body["content"].([]any)
	for _, el := range content {
		e, _ := el.(map[string]any)
		par, ok := e["paragraph"].(map[string]any)
		if !ok {
			continue
		}
		style, _ := par["paragraphStyle"].(map[string]any)
		named, _ := style["namedStyleType"].(string)
		var text strings.Builder
		for _, pe := range asList(par["elements"]) {
			if tr, ok := pe.(map[string]any)["textRun"].(map[string]any); ok {
				c, _ := tr["content"].(string)
				text.WriteString(c)
			}
		}
		bullet := "-"
		if b, ok := par["bullet"].(map[string]any); ok {
			bullet = fmt.Sprintf("bullet %v", keysOf(b))
		}
		ids := map[string][]string{}
		collectIDs(par, ids)
		t.Logf("  [%v,%v) %-12s %-28s %q %s", e["startIndex"], e["endIndex"], named, bullet, text.String(), fmtIDs(ids))
	}
	all := map[string][]string{}
	collectIDs(content, all)
	t.Logf("  distinct ids in the body: %s", fmtIDs(all))
}

// collectIDs walks any JSON value and records every suggestion id by the key
// it sits under: the id lists, and the keys of the suggested-change maps.
func collectIDs(v any, into map[string][]string) {
	switch x := v.(type) {
	case map[string]any:
		for k, sub := range x {
			if strings.HasPrefix(k, "suggested") {
				switch ids := sub.(type) {
				case []any:
					for _, id := range ids {
						if s, ok := id.(string); ok {
							into[k] = appendNew(into[k], s)
						}
					}
				case map[string]any:
					for id := range ids {
						into[k] = appendNew(into[k], id)
					}
				}
			}
			collectIDs(sub, into)
		}
	case []any:
		for _, sub := range x {
			collectIDs(sub, into)
		}
	}
}

func appendNew(list []string, s string) []string {
	for _, have := range list {
		if have == s {
			return list
		}
	}
	return append(list, s)
}

func fmtIDs(ids map[string][]string) string {
	keys := keysOf(anyMap(ids))
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d%v", strings.TrimPrefix(k, "suggested"), len(ids[k]), ids[k]))
	}
	return strings.Join(parts, " ")
}

func anyMap(m map[string][]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}
