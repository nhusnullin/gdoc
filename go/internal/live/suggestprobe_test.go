package live

// Can a cover page be added as a suggestion rather than as a direct edit?
//
// Nail's question, 2026-09-10, and it is the one that decides how the house
// template reaches a document somebody else owns. Adding a cover, the
// front-matter tables and the legend needs gdoc to write text, and writing text
// directly is the thing M7b's whole security argument rests on it being unable
// to do. If Docs will take the same content as a *suggestion*, that argument
// survives untouched: gdoc proposes the cover, Nail accepts it in the browser,
// and no request gdoc sends can change a character on its own authority.
//
// One thing is already known and needs no measuring: insertText in SUGGEST mode
// works, and propose does it in production every time it suggests a wording
// change. What nobody has asked Google is whether the rest of a cover can go
// the same way: a page break, a table, and the styling that makes any of it
// look like the house style.
//
// So this asks, one request kind per batch so a refusal takes only its own case
// with it, and reports what came back. It asserts almost nothing, for the
// fidelity probe's reason: a measurement that fails the build when Google
// answers differently has already decided the answer. It fails only when it
// cannot create or cannot trash.
//
// It creates its own document, so the policy has one door, AllowCreateIn, and
// the created id reaches LevelFull through the create the guard carried. No
// guard change is needed to ask the question.

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"gdoc/internal/drive"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
)

// suggestCase is one request kind asked in SUGGEST mode.
type suggestCase struct {
	name string
	// req builds the request. tableStart is the index of the table the probe
	// wrote, for the one case that addresses it.
	req func(tableStart int) map[string]any
	// what the case is really asking, printed beside the answer so the table
	// reads without the code next to it.
	asks string
}

func suggestCases() []suggestCase {
	return []suggestCase{
		{
			name: "insertText",
			asks: "the cover's words",
			req: func(int) map[string]any {
				return req("insertText", map[string]any{
					"location": map[string]any{"index": 1},
					"text":     "SUGGESTED COVER TITLE\n",
				})
			},
		},
		{
			name: "insertPageBreak",
			asks: "the page break under the cover",
			req: func(int) map[string]any {
				return req("insertPageBreak", map[string]any{
					"location": map[string]any{"index": 1},
				})
			},
		},
		{
			name: "insertTable",
			asks: "the version-control and legend tables",
			req: func(int) map[string]any {
				return req("insertTable", map[string]any{
					"location": map[string]any{"index": 1},
					"rows":     2,
					"columns":  2,
				})
			},
		},
		{
			name: "insertInlineImage",
			asks: "a logo on the cover, if one could be reached",
			req: func(int) map[string]any {
				return req("insertInlineImage", map[string]any{
					"location": map[string]any{"index": 1},
					"uri":      "https://www.google.com/images/srpr/logo3w.png",
					"objectSize": map[string]any{
						"height": map[string]any{"magnitude": 50, "unit": "PT"},
						"width":  map[string]any{"magnitude": 50, "unit": "PT"},
					},
				})
			},
		},
		{
			name: "updateParagraphStyle",
			asks: "styling the cover once it is there",
			req: func(int) map[string]any {
				return req("updateParagraphStyle", map[string]any{
					"range":          map[string]any{"startIndex": 1, "endIndex": 5},
					"paragraphStyle": map[string]any{"alignment": "CENTER"},
					"fields":         "alignment",
				})
			},
		},
		{
			name: "updateTextStyle",
			asks: "the cover's face and size",
			req: func(int) map[string]any {
				return req("updateTextStyle", map[string]any{
					"range":     map[string]any{"startIndex": 1, "endIndex": 5},
					"textStyle": map[string]any{"bold": true},
					"fields":    "bold",
				})
			},
		},
		{
			name: "createParagraphBullets",
			asks: "the legend's bullets",
			req: func(int) map[string]any {
				return req("createParagraphBullets", map[string]any{
					"range":        map[string]any{"startIndex": 1, "endIndex": 5},
					"bulletPreset": "BULLET_DISC_CIRCLE_SQUARE",
				})
			},
		},
		{
			name: "deleteContentRange",
			asks: "removing an empty page gdoc did not write",
			req: func(int) map[string]any {
				return req("deleteContentRange", map[string]any{
					"range": map[string]any{"startIndex": 1, "endIndex": 3},
				})
			},
		},
		{
			name: "createNamedRange",
			asks: "marking gdoc's own prelude so a second run can find it",
			req: func(int) map[string]any {
				return req("createNamedRange", map[string]any{
					"name":  "gdoc:house-prelude",
					"range": map[string]any{"startIndex": 1, "endIndex": 5},
				})
			},
		},
		{
			name: "updateTableCellStyle",
			asks: "shading the front-matter tables",
			req: func(start int) map[string]any {
				return req("updateTableCellStyle", map[string]any{
					"tableStartLocation": map[string]any{"index": start},
					"tableCellStyle":     map[string]any{"backgroundColor": map[string]any{"color": map[string]any{"rgbColor": map[string]any{"red": 0.9, "green": 0.9, "blue": 0.9}}}},
					"fields":             "backgroundColor",
				})
			},
		},
	}
}

// suggestBatch is batch with the SUGGEST bar on it, which is the whole point of
// this probe: the same request gdoc already sends, asked as a proposal.
func suggestBatch(ctx context.Context, s *gapi.Session, id string, requests ...map[string]any) error {
	body := map[string]any{
		"requests":     requests,
		"writeControl": map[string]any{"writeMode": "SUGGEST"},
	}
	url := "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
	return s.PostJSON(ctx, url, body, nil)
}

func TestLiveSuggestedInsertProbe(t *testing.T) {
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

	type result struct {
		name, asks string
		accepted   bool
		reason     string
		marks      map[string]int
	}
	var results []result

	// One fresh document per case, and that is the whole method.
	//
	// The first two runs of this probe used one document for every case, and
	// both answers were wrong. Seven suggested inserts pile up, so by the fifth
	// case the indexes every later request named were inside content the earlier
	// ones had already suggested. Two cases were then refused for a stale index
	// and read as Docs limits, and two more were accepted with no new suggestion
	// mark and read as silent direct edits, when what had really happened is
	// that a style applied to already-suggested text folds into the suggestion
	// that is already there. A probe that measures its own leftovers answers
	// about itself. So each case gets a document nobody has suggested anything
	// in, holding the author's own text, which is the shape a real cover insert
	// meets.
	for _, c := range suggestCases() {
		r := result{name: c.name, asks: c.asks}
		id, err := createDoc(ctx, s, folder, "gdoc SUGGEST PROBE - "+c.name)
		if err != nil {
			t.Fatalf("the probe document for %s could not be created: %v", c.name, err)
		}
		func() {
			defer func() {
				if err := drive.Trash(ctx, s, id); err != nil {
					t.Errorf("the probe document %q for %s is still in the folder: %v", id, c.name, err)
				}
			}()
			if err := writeBody(ctx, s, id); err != nil {
				t.Errorf("the content for %s could not be written: %v", c.name, err)
				return
			}
			before, err := readInline(ctx, s, id)
			if err != nil {
				t.Errorf("the read before %s failed: %v", c.name, err)
				return
			}
			start := 0
			if ts, ok := tableStart(before); ok {
				start = ts
			}
			if err := suggestBatch(ctx, s, id, c.req(start)); err != nil {
				r.reason = strings.ReplaceAll(err.Error(), "\n", " ")
				return
			}
			r.accepted = true
			after, err := readInline(ctx, s, id)
			if err != nil {
				t.Errorf("the read after %s failed: %v", c.name, err)
				return
			}
			r.marks = map[string]int{}
			for k, n := range suggestionIDsIn(after) {
				if d := n - suggestionIDsIn(before)[k]; d > 0 {
					r.marks[k] = d
				}
			}
		}()
		results = append(results, r)
	}

	t.Log("")
	t.Log("  request kind             accepted  asks")
	t.Log("  -----------------------  --------  ----------------------------------------")
	for _, r := range results {
		mark := "NO "
		if r.accepted {
			mark = "yes"
		}
		t.Logf("  %-23s  %-8s  %s", r.name, mark, r.asks)
		if r.reason != "" {
			t.Logf("      refused: %s", r.reason)
		}
		if r.accepted {
			if len(r.marks) == 0 {
				t.Log("      *** accepted and NO suggestion recorded: Docs took it as a DIRECT EDIT ***")
			} else {
				var ks []string
				for k := range r.marks {
					ks = append(ks, k)
				}
				sort.Strings(ks)
				for _, k := range ks {
					t.Logf("      suggested: %s +%d", k, r.marks[k])
				}
			}
		}
	}
	t.Log("  A kind that was accepted and left no suggestion id is the dangerous answer:")
	t.Log("  Docs took it as a direct edit while gdoc asked for a suggestion.")
}

// readInline is the read that carries suggestion ids, which readRaw does not
// ask for.
func readInline(ctx context.Context, s *gapi.Session, id string) (map[string]any, error) {
	var raw json.RawMessage
	url := "https://docs.googleapis.com/v1/documents/" + id +
		"?includeTabsContent=true&suggestionsViewMode=SUGGESTIONS_INLINE"
	if err := s.GetJSON(ctx, url, &raw); err != nil {
		return nil, err
	}
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, err
	}
	// With includeTabsContent the body lives under the first tab, so the tab is
	// flattened in the way readRaw flattens it. Without this, tableStart looks
	// for a table at the top level, finds none, and every caller sends index 0:
	// which is what "The provided table start location is invalid" was, twice.
	if tabs, ok := d["tabs"].([]any); ok && len(tabs) > 0 {
		if dt, ok := dig(tabs[0], "documentTab").(map[string]any); ok {
			for k, v := range dt {
				d[k] = v
			}
		}
	}
	return d, nil
}

// suggestionIDsIn walks a read-back document for every field Docs uses to
// record a suggestion, not only the two that mark inserted and deleted text.
//
// The narrow version of this walk was the probe's own bug and it nearly
// reported a false alarm. A suggested style change is not an insertion: Docs
// records it under suggestedParagraphStyleChanges, suggestedTextStyleChanges,
// suggestedBulletChanges and their siblings, so a walk looking only for
// suggestedInsertionIds saw one id where there were several kinds of proposal,
// and the probe's own rule would have read six accepted requests as silent
// direct edits.
func suggestionIDsIn(doc map[string]any) map[string]int {
	out := map[string]int{}
	// The tab subtree alone when there is one. readInline flattens
	// tabs[0].documentTab to the top level so that tableStart can find the
	// table, and a walk over the whole answer then counts every mark twice,
	// once in the tab and once in the copy.
	if tabs, ok := doc["tabs"]; ok {
		doc = map[string]any{"tabs": tabs}
	}
	var walk func(v any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			for k, sub := range t {
				if strings.HasPrefix(k, "suggested") {
					switch list := sub.(type) {
					case []any:
						out[k] += len(list)
					case map[string]any:
						out[k] += len(list)
					}
				}
				walk(sub)
			}
		case []any:
			for _, sub := range t {
				walk(sub)
			}
		}
	}
	walk(doc)
	return out
}
