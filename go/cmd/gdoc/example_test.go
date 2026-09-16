// The note a colleague builds first.
//
// release/example/first-note.md ships in the release zip and is the first
// thing a colleague points gdoc at. Nothing else in this repository reads it,
// so a front-matter key renamed or a block the walker stopped accepting would
// reach somebody else's machine as a refusal on their first command. This
// builds it on every commit instead.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTheReleaseExampleNoteBuilds builds the example the way a colleague does,
// from a copy, because a build writes beside the note it was pointed at and a
// test must never write into the release folder.
func TestTheReleaseExampleNoteBuilds(t *testing.T) {
	noSession(t)
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "release", "example", "first-note.md"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	md := filepath.Join(dir, "first-note.md")
	if err := os.WriteFile(md, src, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "first-note.docx")

	got, code := runJSON(t, "build", "--md", md, "--out", out)
	if code != 0 || got["ok"] != true {
		t.Fatalf("the example note in the release zip does not build: %v (exit %d)", got, code)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	if data["title"] == "" || data["title"] == nil {
		t.Errorf("the example note builds a document with no title: %v", data)
	}

	// A heading, a list and a table, because the note is what a colleague
	// reads to learn what a note may hold.
	counts, ok := data["body"].(map[string]any)
	if !ok {
		t.Fatalf("no body counts: %v", data)
	}
	for _, field := range []string{"headings", "lists", "tables"} {
		if n, _ := counts[field].(float64); n < 1 {
			t.Errorf("the example note carries no %s, and it is the one note a colleague copies", field)
		}
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("no document was written: %v", err)
	}
}
