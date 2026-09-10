package live

// What index follows a table, and is it a place text can be inserted?
//
// Found by TestLivePreludeIsProposedNotWritten on 2026-09-10. The prelude batch
// was refused whole:
//
//	Invalid requests[106].insertText: The insertion index must be inside the
//	bounds of an existing paragraph. You can still create new paragraphs by
//	inserting newlines.
//
// The failing request is the spacer newline between two front-matter tables, at
// the index internal/prelude computes as one past the table's last cell. The
// per-cell arithmetic in that package is right and was measured: a 2x7 table
// inserted at 279 puts the first cell's content at 283, and a cell holding ten
// characters puts the next cell's content at 295. What nobody had measured is
// where a table *ends*.
//
// So this asks Google, one throwaway document per case, and logs a table. It
// asserts almost nothing, for TestLiveStyleFidelity's reason: a measurement
// that fails the build when Google answers differently has already decided the
// answer. It fails only when it cannot create or cannot trash.
//
// Everything is asked in SUGGEST mode, which is the mode the prelude sends in
// and the mode the failure happened in. A batch is one unit of index
// accounting: Docs applies its requests in order against indexes that move as
// it goes, which is exactly what the prelude relies on, so the sweep asks its
// question inside one batch rather than across two.
//
// **Accepted is not the answer, and that is the trap this probe was rewritten
// to close.** Its first run reported the first accepted index and stopped
// there. For a 2x2 inserted at 1 that is 12, which Docs takes happily and
// which is inside the last cell's own paragraph: the newline lands in the
// table rather than behind it, and a second table asked for at 13 is then
// nested in that cell. So every accepted candidate is read back, and the
// verdict is where the insert really went.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"gdoc/internal/drive"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
)

// tableAt is where the probe puts its table, which is the first index of an
// empty document's body.
const tableAt = 1

// TestLiveTableIndexProbe measures the index map of an inserted table and finds
// the index behind it that an insertText may land on.
func TestLiveTableIndexProbe(t *testing.T) {
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

	// One: the index map itself, read back off a document holding one 2x2
	// table and nothing else anybody added.
	behind := 0
	probeDoc(ctx, t, s, folder, "the index map of a 2x2", func(id string) {
		if err := suggestBatch(ctx, s, id, insertTable(tableAt, 2, 2)); err != nil {
			t.Errorf("the table could not be suggested: %v", err)
			return
		}
		after, err := readInline(ctx, s, id)
		if err != nil {
			t.Errorf("the read back failed: %v", err)
			return
		}
		logIndexMap(t, after)
		start, end, ok := tableSpan(after)
		if !ok {
			t.Error("  the read back carries no table at all")
			return
		}
		behind = end
		t.Logf("  the table spans [%d,%d), which is %d index units for a table inserted at %d",
			start, end, end-start, tableAt)
		t.Logf("  the paragraph behind it begins at %d, which is the insert index plus %d", end, end-tableAt)
	})

	// Two: which index behind the table an insertText may land on, and where
	// it really goes. One document per candidate, and the whole thing in one
	// batch, because that is how the prelude sends it.
	//
	// The candidates bracket the arithmetic internal/prelude used, which
	// computes one past the last cell's paragraph mark: 13 for a 2x2 at 1.
	t.Log("")
	t.Log("  a newline behind a 2x2 table inserted at 1, in the same batch")
	t.Log("  index  accepted  where it landed / why it was refused")
	t.Log("  -----  --------  --------------------------------------------------")
	for _, at := range []int{11, 12, 13, 14, 15, 16} {
		accepted, note := sweep(ctx, t, s, folder, at)
		mark := "NO "
		if accepted {
			mark = "yes"
		}
		t.Logf("  %5d  %-8s  %s", at, mark, note)
	}

	// Three: the case the prelude actually builds, which is a second table
	// behind the first with one paragraph between them. A newline that was
	// accepted does not by itself say a table may follow it, and the first run
	// of this probe proved that by nesting one.
	if behind > 0 {
		probeDoc(ctx, t, s, folder, fmt.Sprintf("a second table behind the first, spaced at %d", behind), func(id string) {
			err := suggestBatch(ctx, s, id,
				insertTable(tableAt, 2, 2),
				req("insertText", map[string]any{
					"location": map[string]any{"index": behind},
					"text":     "\n",
				}),
				insertTable(behind+1, 2, 2),
			)
			if err != nil {
				t.Errorf("  a second table behind the first was REFUSED: %v", err)
				return
			}
			after, err := readInline(ctx, s, id)
			if err != nil {
				t.Errorf("the read back failed: %v", err)
				return
			}
			t.Logf("  accepted, and the document holds %d top-level tables", tableCount(after))
			logIndexMap(t, after)
		})
	}
}

// sweep asks one candidate index on a document of its own, then reads back
// where the newline went. A document per candidate is the suggest probe's rule:
// a probe that measures its own leftovers answers about itself.
func sweep(ctx context.Context, t *testing.T, s *gapi.Session, folder string, at int) (bool, string) {
	accepted := false
	note := ""
	probeDocQuiet(ctx, t, s, folder, fmt.Sprintf("newline at %d", at), func(id string) {
		err := suggestBatch(ctx, s, id,
			insertTable(tableAt, 2, 2),
			req("insertText", map[string]any{
				"location": map[string]any{"index": at},
				"text":     "\n",
			}),
		)
		if err != nil {
			note = docsReason(err)
			return
		}
		accepted = true
		after, err := readInline(ctx, s, id)
		if err != nil {
			note = "accepted, and the read back failed: " + oneLine(err.Error())
			return
		}
		start, end, ok := tableSpan(after)
		switch {
		case !ok:
			note = "accepted, and the read back carries no table"
		case end-start > emptyTableUnits(2, 2):
			note = fmt.Sprintf("INSIDE the table: it spans [%d,%d), %d units rather than %d",
				start, end, end-start, emptyTableUnits(2, 2))
		default:
			note = fmt.Sprintf("behind the table, which still spans [%d,%d)", start, end)
		}
	})
	return accepted, note
}

// emptyTableUnits is how many index units an empty table of this shape takes,
// as this probe measured it: one for the table itself, one per row, one per
// cell plus one for that cell's own paragraph mark, and one more for the
// table's own end.
func emptyTableUnits(rows, cols int) int {
	return 1 + rows + rows*cols*2 + 1
}

func insertTable(at, rows, cols int) map[string]any {
	return req("insertTable", map[string]any{
		"location": map[string]any{"index": at},
		"rows":     rows,
		"columns":  cols,
	})
}

// docsReason is the sentence Docs refused with, without the URL in front of it.
func docsReason(err error) string {
	s := strings.ReplaceAll(err.Error(), "\n", " ")
	if i := strings.Index(s, "Invalid requests"); i >= 0 {
		s = s[i:]
	} else if i := strings.LastIndex(s, ": "); i >= 0 {
		s = s[i+2:]
	}
	return oneLine(s)
}

// probeDoc creates one document, hands it to the case, and trashes it whatever
// the case did.
func probeDoc(ctx context.Context, t *testing.T, s *gapi.Session, folder, name string, run func(id string)) {
	t.Log("")
	t.Logf("  %s", name)
	probeDocQuiet(ctx, t, s, folder, name, run)
}

func probeDocQuiet(ctx context.Context, t *testing.T, s *gapi.Session, folder, name string, run func(id string)) {
	id, err := createDoc(ctx, s, folder, "gdoc TABLE INDEX PROBE - "+name)
	if err != nil {
		t.Fatalf("the probe document for %s could not be created: %v", name, err)
	}
	defer func() {
		if err := drive.Trash(ctx, s, id); err != nil {
			t.Errorf("the probe document %q for %s is still in the folder: %v", id, name, err)
		}
	}()
	run(id)
}

// tableSpan is the first top-level table's own range, and tableCount how many
// of them there are. A second table the probe meant to put behind the first and
// really nested inside it is not a top-level table, so the count is the answer
// to "did it land beside or in".
func tableSpan(d map[string]any) (int, int, bool) {
	for _, e := range content(d) {
		m, ok := e.(map[string]any)
		if !ok || m["table"] == nil {
			continue
		}
		s, sok := m["startIndex"].(float64)
		en, eok := m["endIndex"].(float64)
		if sok && eok {
			return int(s), int(en), true
		}
	}
	return 0, 0, false
}

func tableCount(d map[string]any) int {
	n := 0
	for _, e := range content(d) {
		if m, ok := e.(map[string]any); ok && m["table"] != nil {
			n++
		}
	}
	return n
}

// logIndexMap prints every structural index the read carries, in document
// order: the paragraphs, the table, its rows, its cells and each cell's own
// content. It is the answer to "what are the real start and end indexes",
// and it is read by a person.
func logIndexMap(t *testing.T, d map[string]any) {
	t.Log("    element                       start    end")
	t.Log("    ---------------------------  ------  -----")
	for _, e := range content(d) {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		switch {
		case m["paragraph"] != nil:
			t.Logf("    %-27s  %6s  %5s", "paragraph "+quoted(paragraphText(m)), at(m, "startIndex"), at(m, "endIndex"))
		case m["table"] != nil:
			t.Logf("    %-27s  %6s  %5s", "table", at(m, "startIndex"), at(m, "endIndex"))
			logTable(t, m, "      ")
		case m["sectionBreak"] != nil:
			t.Logf("    %-27s  %6s  %5s", "sectionBreak", at(m, "startIndex"), at(m, "endIndex"))
		default:
			t.Logf("    %-27s  %6s  %5s", "("+strings.Join(keys(m), ",")+")", at(m, "startIndex"), at(m, "endIndex"))
		}
	}
}

func logTable(t *testing.T, table map[string]any, pad string) {
	rows, _ := dig(table, "table", "tableRows").([]any)
	for ri, r := range rows {
		rm, _ := r.(map[string]any)
		t.Logf("%s%-*s  %6s  %5s", pad, 29-len(pad), fmt.Sprintf("row %d", ri), at(rm, "startIndex"), at(rm, "endIndex"))
		cells, _ := dig(rm, "tableCells").([]any)
		for ci, c := range cells {
			cm, _ := c.(map[string]any)
			t.Logf("%s  %-*s  %6s  %5s", pad, 27-len(pad), fmt.Sprintf("cell %d.%d", ri, ci), at(cm, "startIndex"), at(cm, "endIndex"))
			inner, _ := dig(cm, "content").([]any)
			for _, ie := range inner {
				im, _ := ie.(map[string]any)
				if im["table"] != nil {
					t.Logf("%s    %-*s  %6s  %5s", pad, 25-len(pad), "table", at(im, "startIndex"), at(im, "endIndex"))
					continue
				}
				t.Logf("%s    %-*s  %6s  %5s", pad, 25-len(pad), "paragraph "+quoted(paragraphText(im)), at(im, "startIndex"), at(im, "endIndex"))
			}
		}
	}
}

func at(m map[string]any, key string) string {
	f, ok := m[key].(float64)
	if !ok {
		return "-"
	}
	return fmt.Sprintf("%d", int(f))
}

func paragraphText(m map[string]any) string {
	els, _ := dig(m, "paragraph", "elements").([]any)
	var b strings.Builder
	for _, e := range els {
		if s, ok := dig(e, "textRun", "content").(string); ok {
			b.WriteString(s)
		}
	}
	return b.String()
}

func quoted(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > 12 {
		s = s[:12] + "..."
	}
	return `"` + s + `"`
}

func keys(m map[string]any) []string {
	var out []string
	for k := range m {
		if k == "startIndex" || k == "endIndex" {
			continue
		}
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
