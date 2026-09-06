// Package live is the one opt-in end-to-end test: the real token, the real
// guard, and one real Google Doc. It is skipped unless GDOC_LIVE_TEST=1, so
// `go test ./...` on any machine runs nothing here.
//
// It reads, and it writes nothing to Drive. The plan for this milestone
// described a live test that created a document in the test folder and wrote a
// comment into it, and that cannot be built here: a create and a comment are
// POST requests, this package would have to build them, and building a request
// means naming net/http. The boundary test's import allowlist cannot admit a
// package whose only files are tests, because its disappearance half reads
// production files only and would then fail for the opposite reason. So a live
// write belongs to the milestone that has a production writer to run it
// through, which is M6, and M2's live test is a read of a document the run is
// pointed at. ⚠️ recorded in the plan.
//
// The document is named by GDOC_LIVE_DOC_ID and there is no default. The guard
// is opened with exactly the id the run was given, and a run that names none
// has nothing to read.
package live

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/docx"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/suggestions"
	"gdoc/internal/view"
)

const (
	liveVar   = "GDOC_LIVE_TEST"
	docVar    = "GDOC_LIVE_DOC_ID"
	recordVar = "GDOC_LIVE_RECORD"
)

// TestLiveReadOfARealDocument runs the whole read path against Google: the
// token is loaded and refreshed if it has expired, every request goes through
// the guard, and the readers run on what really came back rather than on a
// fixture somebody wrote.
func TestLiveReadOfARealDocument(t *testing.T) {
	if os.Getenv(liveVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=<document id> to read one real document with the real token", liveVar, docVar)
	}
	id := strings.TrimSpace(os.Getenv(docVar))
	if id == "" {
		t.Fatalf("%s=1 needs %s=<document id>: the guard is opened with exactly the document the run names, and there is no default", liveVar, docVar)
	}

	p := guard.NewPolicy()
	p.AllowFile(id, guard.LevelSuggest)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	ctx := context.Background()

	// The raw answer first, so the recording is what Google sent and the parse
	// runs on the same bytes.
	var raw json.RawMessage
	if err := s.GetJSON(ctx, docs.URL(id), &raw); err != nil {
		t.Fatalf("the Docs read failed: %v", err)
	}
	d, err := docs.Parse(raw)
	if err != nil {
		t.Fatalf("the Docs read did not parse: %v", err)
	}
	if d.ID != id {
		t.Errorf("the read came back for document %q, and %q was asked for", d.ID, id)
	}
	if len(d.Tabs) == 0 {
		t.Error("a real document has at least one tab; includeTabsContent may have been dropped from the URL")
	}

	text, notes := view.Text(d)
	if strings.TrimSpace(text) == "" {
		t.Error("the text projection is empty, and a real document has text in it")
	}
	if !strings.HasSuffix(text, "\n") {
		t.Error("the text must end with exactly one newline")
	}
	t.Logf("read %d tabs, %d characters, %d pending suggestions, %d projection warnings",
		len(d.Tabs), len(text), len(suggestions.List(d)), len(notes))

	raws, err := comments.Fetch(ctx, s, id, nil)
	if err != nil {
		t.Fatalf("the comment listing failed: %v", err)
	}
	threads, unplaced := comments.Threads(raws, d)
	for _, thread := range threads {
		if thread.Marker == "" {
			t.Errorf("thread %s carries no marker, and every thread carries one, %q included", thread.ID, comments.MarkerNone)
		}
	}
	t.Logf("%d threads, %d of them unplaced by the Docs read", len(threads), len(unplaced))

	export, err := docx.Export(ctx, s, id)
	if err != nil {
		t.Fatalf("the docx export failed: %v", err)
	}
	f, err := docx.Parse(export)
	if err != nil {
		t.Fatalf("the docx export did not parse: %v", err)
	}
	for _, thread := range docx.Match(threads, f) {
		if thread.Witness == "" {
			t.Errorf("thread %s came back from the witness with no answer", thread.ID)
		}
		t.Logf("thread %s: marker %s, witness %s", thread.ID, thread.Marker, thread.Witness)
	}

	if os.Getenv(recordVar) == "1" {
		record(t, raw, export)
	}
}

// record writes the two answers to testdata/, for the fixture the decoder is
// later tightened against. What it writes is a real document's real content: it
// is read and redacted by a person before it is committed, and the log line
// says so rather than leaving it to be discovered in a diff.
func record(t *testing.T, read json.RawMessage, export []byte) {
	t.Helper()
	dir := "testdata"
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, read, "", "  "); err != nil {
		// Not a failure of the run: the read already passed, and the raw bytes
		// are still worth keeping.
		t.Logf("the Docs read could not be indented, so it is recorded as it came: %v", err)
		pretty.Reset()
		pretty.Write(read)
	}
	for name, body := range map[string][]byte{
		"live-docs-read.json": pretty.Bytes(),
		"live-export.docx":    export,
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
		t.Logf("recorded %s: this is a real document's content, so redact it before committing it", path)
	}
}
