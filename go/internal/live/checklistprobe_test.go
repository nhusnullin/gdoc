package live

// What does Docs do with a checklist, a person chip and a date chip?
//
// Asked on 2026-11-05 for the tasks research: a note line `- [ ] task` should
// arrive in the document as a checklist item, with whom and by when on it.
// The schema (revision 20261028) has BULLET_CHECKBOX, insertPerson and
// insertDate, and no field for a checked state or for a task assignment. This
// probe asks what the schema cannot say: does each land, what does the read
// show for it, and does each land as a suggestion in SUGGEST mode.
//
// Like the other probes it asserts almost nothing. It fails only when it
// cannot create, set up or trash.

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"gdoc/internal/gapi"
	"gdoc/internal/guard"
)

const checklistSetup = "Questions\nFirst task\nSecond task\nThird task\n"

func TestLiveChecklistProbe(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to probe checklists in the Drive test folder", liveVar, writeVar)
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
	who := strings.TrimSpace(os.Getenv("GDOC_PROBE_PERSON"))
	if who == "" {
		t.Fatal("GDOC_PROBE_PERSON names the email the person chip carries")
	}
	due := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)

	// Indexes: "Questions\n" is [1,11), the three items [11,22) [22,34) [34,45).
	items := rng(11, 44)
	firstEnd := 21  // before "First task"'s newline
	secondEnd := 33 // before "Second task"'s newline

	checklist := func() map[string]any {
		return req("createParagraphBullets", map[string]any{"range": items, "bulletPreset": "BULLET_CHECKBOX"})
	}
	// Chips go in from the back so the earlier indexes stay put.
	chips := func() []map[string]any {
		return []map[string]any{
			req("insertDate", map[string]any{"location": map[string]any{"index": secondEnd},
				"dateElementProperties": map[string]any{"timestamp": due, "dateFormat": "DATE_FORMAT_MONTH_DAY_YEAR_ABBREVIATED"}}),
			req("insertText", map[string]any{"location": map[string]any{"index": secondEnd}, "text": " due "}),
			req("insertPerson", map[string]any{"location": map[string]any{"index": firstEnd},
				"personProperties": map[string]any{"email": who}}),
			req("insertText", map[string]any{"location": map[string]any{"index": firstEnd}, "text": " "}),
		}
	}

	cases := []struct {
		name    string
		suggest bool
		reqs    []map[string]any
	}{
		{"direct: checklist only", false, []map[string]any{checklist()}},
		{"direct: checklist, person chip, date chip", false, append(chips(), checklist())},
		{"suggest: checklist only", true, []map[string]any{checklist()}},
		{"suggest: person chip and date chip", true, chips()},
	}
	for _, c := range cases {
		probeDoc(ctx, t, s, folder, c.name, func(id string) {
			setup := map[string]any{"requests": []map[string]any{
				req("insertText", map[string]any{"location": map[string]any{"index": 1}, "text": strings.TrimSuffix(checklistSetup, "\n")}),
			}}
			url := "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
			if err := s.PostJSON(ctx, url, setup, nil); err != nil {
				t.Fatalf("the setup write failed: %v", err)
			}
			var err error
			if c.suggest {
				err = suggestBatch(ctx, s, id, c.reqs...)
			} else {
				err = s.PostJSON(ctx, url, map[string]any{"requests": c.reqs}, nil)
			}
			if err != nil {
				t.Logf("  REFUSED: %s", docsReason(err))
				return
			}
			after, err := readInline(ctx, s, id)
			if err != nil {
				t.Errorf("the read back failed: %v", err)
				return
			}
			logChecklist(t, after)
		})
	}
}

// TestLiveChecklistForHands makes one checklist document and keeps it, for the
// two things no request can do: a person ticks the first box and presses
// "Assign as a task" on the second. GDOC_PROBE_READ_ID then reads it back, so
// the read shows what a tick and a Docs task look like, if anything.
func TestLiveChecklistForHands(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1", liveVar, writeVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}
	ctx := context.Background()
	p := guard.NewPolicy()
	if id := strings.TrimSpace(os.Getenv("GDOC_PROBE_READ_ID")); id != "" {
		p.AllowFile(id, guard.LevelSuggest)
		s, err := gapi.Open(p, nil)
		if err != nil {
			t.Fatalf("the session could not be opened: %v", err)
		}
		after, err := readInline(ctx, s, id)
		if err != nil {
			t.Fatalf("the read failed: %v", err)
		}
		logChecklist(t, after)
		return
	}
	p.AllowCreateIn(folder)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	id, err := createDoc(ctx, s, folder, "gdoc CHECKLIST - tick the first, assign the second")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	url := "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
	setup := map[string]any{"requests": []map[string]any{
		req("insertText", map[string]any{"location": map[string]any{"index": 1}, "text": "Questions\nTick this box\nAssign this one as a task\nLeave this one\n"}),
		req("createParagraphBullets", map[string]any{"range": rng(11, 64), "bulletPreset": "BULLET_CHECKBOX"}),
	}}
	if err := s.PostJSON(ctx, url, setup, nil); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Logf("kept: https://docs.google.com/document/d/%s/edit", id)
}

// logChecklist prints each paragraph's elements and bullet, and the list
// definitions the bullets point at, as raw JSON trimmed to a line.
func logChecklist(t *testing.T, d map[string]any) {
	for _, e := range content(d) {
		m, _ := e.(map[string]any)
		para, ok := m["paragraph"].(map[string]any)
		if !ok {
			continue
		}
		b, _ := json.Marshal(map[string]any{"bullet": para["bullet"], "elements": para["elements"],
			"suggestedBulletChanges": para["suggestedBulletChanges"]})
		t.Logf("    para %s", trimTo(string(b), 900))
	}
	if lists, ok := d["lists"].(map[string]any); ok {
		for id, l := range lists {
			lvl := dig(l, "listProperties", "nestingLevels")
			first := any(nil)
			if arr, ok := lvl.([]any); ok && len(arr) > 0 {
				first = arr[0]
			}
			b, _ := json.Marshal(map[string]any{"level0": first, "suggestedInsertionId": dig(l, "suggestedInsertionId")})
			t.Logf("    list %s %s", id, trimTo(string(b), 600))
		}
	}
}

func trimTo(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
