package live

// M11's live test: one comment on words in an ordinary paragraph, and one on
// words inside a table cell, in a document this test creates and trashes.
//
// It is here rather than in live_test.go because that file is M3's and M4's
// and is already 722 lines, and because this test asks a second question
// beside its acceptance one. The two questions are answered differently:
//
//   - The paragraph case is the acceptance bar for the milestone, so every
//     read-back holding is asserted.
//   - The table case is a measurement. docs/backlog/propose-inside-tables.md
//     records that a proposal into a table cell did not land on 2026-09-08 and
//     that nobody knows which step lied. An insertComment alone tells the
//     first suspect from the other two, because no deleteContentRange goes
//     with it. So this case logs what each route said and fails nothing: a
//     measurement that fails the build when Google answers differently has
//     already decided the answer, which is the rule TestLiveStyleFidelity and
//     the M7c probes are written by. The answer goes into the backlog item by
//     hand, after a run.
//
// Neither case runs the capability probe, because annotate does not: a batch
// holding nothing but an insertComment cannot change a character whatever
// Docs does with the write mode. That bend is DECISIONS.md, 2026-09-18.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gdoc/internal/annotate"
	"gdoc/internal/drive"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
)

// What the subject document holds, and which words each case comments on. Each
// quote occurs exactly once in the whole document, which is what FindSpan asks
// for, and no quote carries a line break.
const (
	annotateSentence = "The supplier register is reviewed annually by the operations team.\n"

	annotateParagraphQuoted = "reviewed annually"
	annotateParagraphWhy    = "The 2026 register says quarterly, so this sentence disagrees with it."

	annotateFirstCell  = "Review frequency: once a year"
	annotateSecondCell = "Owner: the Nicosia office"

	annotateCellQuoted = "once a year"
	annotateCellWhy    = "The same disagreement as above, written in the table."
)

// TestLiveAnnotateParagraphAndTable places two comments through the production
// writer and reads both of them back through both routes.
//
// Every document it touches is one it created. The policy has one door open,
// AllowCreateIn on the folder, so the subject document is reachable only
// because the guard carried the create that made it.
func TestLiveAnnotateParagraphAndTable(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to create a document in the Drive test folder and comment in it; %s names another folder", liveVar, writeVar, folderVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}
	t.Logf("creating in folder %s", folder)

	p := guard.NewPolicy()
	p.AllowCreateIn(folder)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	ctx := context.Background()

	docID := annotateSubject(t, ctx, s, folder)

	// The paragraph, and the bar. Anything short of both routes holding is a
	// failure: the unit tests already cover a route that did not hold, and
	// what this run is for is Google agreeing with both.
	paragraph, err := annotate.Apply(ctx, s, docID, annotate.Annotation{
		Quoted: annotateParagraphQuoted,
		Why:    annotateParagraphWhy,
	})
	logAnnotation(t, "paragraph", paragraph, err)
	if err != nil {
		t.Fatalf("the comment on the paragraph could not be written: %v", err)
	}
	if paragraph.CommentID == "" {
		t.Error("the comment on the paragraph landed with no comment id")
	}
	if !paragraph.Checks.DriveListing {
		t.Error("Drive's comment listing does not carry the comment on the paragraph with the words that were sent")
	}
	if !paragraph.Checks.DocxAnchored {
		t.Error("the docx export does not carry the comment on the paragraph as anchored to the quoted words")
	}
	if !paragraph.Verified {
		t.Errorf("the comment on the paragraph is not verified: state %q, checks %+v", paragraph.CommentUpdateState, paragraph.Checks)
	}

	// The table cell, and the measurement. Read the log, then write the answer
	// into docs/backlog/propose-inside-tables.md.
	cell, err := annotate.Apply(ctx, s, docID, annotate.Annotation{
		Quoted: annotateCellQuoted,
		Why:    annotateCellWhy,
	})
	logAnnotation(t, "table cell", cell, err)
	t.Logf("table cell measured: sent %t, drive_listing %t, docx_anchored %t, verified %t",
		err == nil, cell.Checks.DriveListing, cell.Checks.DocxAnchored, cell.Verified)
	t.Log("write that line into docs/backlog/propose-inside-tables.md")
}

// annotateSubject makes the document the two cases comment in: one sentence,
// then a one-row table of two cells with a line of text in each, then the
// trash that runs whatever happens next.
//
// The writes are direct edits rather than suggestions, and that is the point of
// the level: the document was learned from a create the guard carried, so the
// policy holds it at LevelFull. A handed-in document would be refused here.
func annotateSubject(t *testing.T, ctx context.Context, s *gapi.Session, folder string) string {
	t.Helper()
	docID, err := createDoc(ctx, s, folder, "gdoc live annotate "+time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("the subject document could not be created in folder %q: %v", folder, err)
	}
	t.Logf("subject document %s", docID)
	t.Cleanup(func() {
		if err := drive.Trash(ctx, s, docID); err != nil {
			t.Errorf("the subject document %q is still in the folder: %v", docID, err)
			return
		}
		t.Logf("subject document %s trashed", docID)
	})

	if err := batch(ctx, s, docID, req("insertText", map[string]any{
		"location": map[string]any{"index": 1},
		"text":     annotateSentence,
	})); err != nil {
		t.Fatalf("the sentence could not be written into %q: %v", docID, err)
	}
	// The table goes behind the sentence, and Docs numbers a table's own
	// content, so it is written after the text that precedes it and never
	// before.
	if err := batch(ctx, s, docID, insertTable(len([]rune(annotateSentence)), 1, 2)); err != nil {
		t.Fatalf("the table could not be written into %q: %v", docID, err)
	}

	// Where the cells really are is read rather than computed: the arithmetic
	// is measured in TestLiveTableIndexProbe and it is not what this test is
	// asking about.
	d, err := readRaw(ctx, s, docID)
	if err != nil {
		t.Fatalf("the document %q could not be read back after the table: %v", docID, err)
	}
	starts := cellParagraphStarts(d)
	if len(starts) != 2 {
		t.Fatalf("the table in %q reads back with %d cell paragraphs rather than 2", docID, len(starts))
	}
	// Back to front, because an insert at the later index leaves the earlier
	// one where it was and an insert at the earlier one moves everything
	// behind it.
	if err := batch(ctx, s, docID,
		req("insertText", map[string]any{
			"location": map[string]any{"index": starts[1]},
			"text":     annotateSecondCell,
		}),
		req("insertText", map[string]any{
			"location": map[string]any{"index": starts[0]},
			"text":     annotateFirstCell,
		}),
	); err != nil {
		t.Fatalf("the cells of the table in %q could not be filled: %v", docID, err)
	}
	return docID
}

// cellParagraphStarts is the start index of the first paragraph of every cell
// of the first top-level table, in document order.
func cellParagraphStarts(d map[string]any) []int {
	var out []int
	for _, e := range content(d) {
		rows, ok := dig(e, "table", "tableRows").([]any)
		if !ok {
			continue
		}
		for _, r := range rows {
			cells, _ := dig(r, "tableCells").([]any)
			for _, c := range cells {
				inner, _ := dig(c, "content").([]any)
				for _, ie := range inner {
					if dig(ie, "paragraph") == nil {
						continue
					}
					f, ok := dig(ie, "startIndex").(float64)
					if !ok {
						continue
					}
					out = append(out, int(f))
					break
				}
			}
		}
		return out
	}
	return out
}

// logAnnotation prints one result the way a person reads it, warnings and all,
// before any assertion is made about it. A run that fails is a run whose log
// has to say why on its own.
func logAnnotation(t *testing.T, name string, r annotate.Result, err error) {
	t.Helper()
	for _, w := range r.Warnings {
		t.Logf("%s warning: %s", name, w)
	}
	if err != nil {
		t.Logf("%s: not sent: %s", name, oneLine(fmt.Sprintf("%v", err)))
		return
	}
	t.Logf("%s: comment %s, state %s, drive_listing %t, docx_anchored %t, verified %t",
		name, r.CommentID, r.CommentUpdateState, r.Checks.DriveListing, r.Checks.DocxAnchored, r.Verified)
}
