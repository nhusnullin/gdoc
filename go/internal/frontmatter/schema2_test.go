// This file is the block as a list of documents: schema 1 still reading, the
// write that stays schema 1 until something changes, the schema 2 round trip,
// and the lookup by document id. frontmatter_test.go holds the tests that were
// there before the list, and schema_test.go holds Validate's own cases.

package frontmatter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const otherDocumentID = "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz"

// schemaOneFixtures is every note in testdata written the way publish wrote
// one until 2026-09-19. Each must still read, because these are the notes in
// somebody's vault.
var schemaOneFixtures = []string{"full.md", "minimal.md", "crlf.md", "blank-line.md", "proposal-quoted.md"}

// TestReadAcceptsASchemaOneBlockWrittenBeforeToday is the migration's whole
// promise to a note that already exists: it reads, and it reads as one entry.
func TestReadAcceptsASchemaOneBlockWrittenBeforeToday(t *testing.T) {
	for _, name := range schemaOneFixtures {
		t.Run(name, func(t *testing.T) {
			b, err := Read(fixture(t, name))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if b == nil {
				t.Fatal("no block read")
			}
			if b.Schema != 1 {
				t.Errorf("schema = %d, want 1: the file says 1", b.Schema)
			}
			if len(b.Documents) != 1 {
				t.Fatalf("documents = %d, want the one document the old block named", len(b.Documents))
			}
			if b.Documents[0].ID != testDocumentID {
				t.Errorf("documents[0].id = %q, want the old document_id %q", b.Documents[0].ID, testDocumentID)
			}
		})
	}
}

// The old fields land on the entry, not beside it.
func TestASchemaOneBlockCarriesItsOldFieldsOnTheEntry(t *testing.T) {
	b, err := Read(fixture(t, "full.md"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	e := b.Documents[0]
	if e.FolderID != "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr" {
		t.Errorf("folder_id = %q", e.FolderID)
	}
	if e.Published == nil || e.Published.Title != "Supplier register policy" {
		t.Errorf("published = %+v", e.Published)
	}
	if e.SuggestionsSeen == nil || len(e.SuggestionsSeen.Items) != 2 {
		t.Errorf("suggestions_seen = %+v", e.SuggestionsSeen)
	}
	if len(e.Proposals) != 1 || e.Proposals[0].ID != "gdoc.p1" {
		t.Errorf("proposals = %+v", e.Proposals)
	}
	if e.Exported != nil || e.TabID != "" {
		t.Errorf("a schema 1 block invented fields it does not carry: %+v", e)
	}
}

// TestWriteAnUnchangedSchemaOneBlockStaysSchemaOne is the pin the migration
// must keep: a note nothing changed in is a note nothing was written to. It is
// TestWriteAnUnchangedBlockIsByteIdentical restated on the new type, and both
// stay.
func TestWriteAnUnchangedSchemaOneBlockStaysSchemaOne(t *testing.T) {
	for _, name := range schemaOneFixtures {
		t.Run(name, func(t *testing.T) {
			src := fixture(t, name)
			b, err := Read(src)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			out, err := Write(src, b)
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
			if !bytes.Equal(src, out) {
				t.Fatalf("an unchanged write moved a byte.\nwant:\n%q\ngot:\n%q", src, out)
			}
			if bytes.Contains(out, []byte("documents:")) {
				t.Error("an unchanged write rewrote the block as schema 2")
			}
		})
	}
}

// TestAChangingWriteRewritesSchemaOneAsTwo is the other half: the moment
// anything changes, the block is written in the shape this gdoc writes, and it
// reads back as what was asked for.
func TestAChangingWriteRewritesSchemaOneAsTwo(t *testing.T) {
	src := fixture(t, "minimal.md")
	b, err := Read(src)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	at := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)
	b.Documents[0].Exported = &Exported{At: at}

	out, err := Write(src, b)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Contains(out, []byte("schema: 2")) {
		t.Errorf("the block was not rewritten as schema 2:\n%s", out)
	}
	if !bytes.Contains(out, []byte("documents:")) {
		t.Errorf("the block has no documents list:\n%s", out)
	}
	if bytes.Contains(out, []byte("document_id:")) {
		t.Errorf("the old key survived the rewrite:\n%s", out)
	}
	if !bytes.Contains(out, []byte("\n# Scope\n")) {
		t.Errorf("the body did not survive:\n%s", out)
	}

	back, err := Read(out)
	if err != nil {
		t.Fatalf("Read after Write: %v", err)
	}
	if back.Schema != 2 || len(back.Documents) != 1 {
		t.Fatalf("block = %+v", back)
	}
	if back.Documents[0].ID != testDocumentID {
		t.Errorf("id = %q, want %q", back.Documents[0].ID, testDocumentID)
	}
	if back.Documents[0].Exported == nil || !back.Documents[0].Exported.At.Equal(at) {
		t.Errorf("exported = %+v, want at %v", back.Documents[0].Exported, at)
	}
}

// TestASchemaTwoBlockRoundTrips is every field of every entry going in and
// coming back out, and an unchanged write moving nothing.
func TestASchemaTwoBlockRoundTrips(t *testing.T) {
	src := fixture(t, "two-documents.md")
	b, err := Read(src)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if b.Schema != 2 {
		t.Fatalf("schema = %d, want 2", b.Schema)
	}
	if len(b.Documents) != 2 {
		t.Fatalf("documents = %d, want 2", len(b.Documents))
	}

	first, second := b.Documents[0], b.Documents[1]
	if first.ID != testDocumentID || second.ID != otherDocumentID {
		t.Fatalf("ids = %q and %q", first.ID, second.ID)
	}
	if first.Published == nil || first.Published.House != "embedded" {
		t.Errorf("documents[0].published = %+v", first.Published)
	}
	if first.Exported != nil {
		t.Errorf("documents[0].exported = %+v, want none", first.Exported)
	}
	if len(first.Proposals) != 1 || first.Proposals[0].Quoted != "reviewed annually" {
		t.Errorf("documents[0].proposals = %+v", first.Proposals)
	}
	if first.SuggestionsSeen == nil || len(first.SuggestionsSeen.Items) != 1 {
		t.Errorf("documents[0].suggestions_seen = %+v", first.SuggestionsSeen)
	}
	if second.TabID != "t.0" {
		t.Errorf("documents[1].tab_id = %q, want t.0", second.TabID)
	}
	if second.Exported == nil {
		t.Fatal("documents[1].exported is absent")
	}
	wantAt := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)
	if !second.Exported.At.Equal(wantAt) {
		t.Errorf("documents[1].exported.at = %v, want %v", second.Exported.At, wantAt)
	}
	if second.Exported.Note != "notes/supplier-register.md" {
		t.Errorf("documents[1].exported.note = %q", second.Exported.Note)
	}
	if len(second.Proposals) != 1 || second.Proposals[0].ID != "gdoc.p2" {
		t.Errorf("documents[1].proposals = %+v", second.Proposals)
	}

	out, err := Write(src, b)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Equal(src, out) {
		t.Fatalf("an unchanged write moved a byte.\nwant:\n%q\ngot:\n%q", src, out)
	}
}

// A changed schema 2 block is rendered again, and the author's lines stay where
// they were.
func TestAChangedSchemaTwoBlockIsWrittenAndReadsBack(t *testing.T) {
	src := fixture(t, "two-documents.md")
	b, err := Read(src)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	b.Documents[1].Proposals = nil

	out, err := Write(src, b)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if bytes.Contains(out, []byte("gdoc.p2")) {
		t.Errorf("the proposal that was dropped is still there:\n%s", out)
	}
	if !bytes.Contains(out, []byte("gdoc.p1")) {
		t.Errorf("the other document's proposal went with it:\n%s", out)
	}
	for _, line := range []string{"title: Supplier register policy\n", "author: Nail\n", "\n# Scope\n"} {
		if !bytes.Contains(out, []byte(line)) {
			t.Errorf("the author's %q is gone:\n%s", line, out)
		}
	}
	back, err := Read(out)
	if err != nil {
		t.Fatalf("Read after Write: %v", err)
	}
	if len(back.Documents) != 2 || len(back.Documents[1].Proposals) != 0 {
		t.Errorf("block = %+v", back)
	}
}

// TestEntryFindsTheDocumentTheURLNames is the lookup every writer makes. A
// document the block does not name is a refusal that says what it does name,
// because the answer is always to open one of those instead.
func TestEntryFindsTheDocumentTheURLNames(t *testing.T) {
	b, err := Read(fixture(t, "two-documents.md"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	for _, id := range []string{testDocumentID, otherDocumentID} {
		e, err := b.Entry(id)
		if err != nil {
			t.Fatalf("Entry(%q) = %v", id, err)
		}
		if e.ID != id {
			t.Errorf("Entry(%q) gave the entry for %q", id, e.ID)
		}
	}
	e, err := b.Entry("nope")
	if err == nil {
		t.Fatalf("Entry on a document the block does not name gave %+v", e)
	}
	for _, want := range []string{testDocumentID, otherDocumentID} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

// The entry is the one in the block, so a writer that changes it and writes the
// block writes what it changed.
func TestEntryIsTheBlocksOwnEntry(t *testing.T) {
	b, err := Read(fixture(t, "two-documents.md"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	e, err := b.Entry(otherDocumentID)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	e.TabID = "t.7"
	if b.Documents[1].TabID != "t.7" {
		t.Error("the entry is a copy, so a writer's change would never reach the note")
	}
}

// TestTheShippedDecoderRefusesADocumentsKeyByName is what a colleague running
// the binary from before this milestone sees when a note has moved on. The read
// is strict before anything looks at the schema, so the sentence names the key
// it did not know rather than the version.
//
// The old decoder is the schema 1 shape, which is still in this package, so the
// sentence is pinned here rather than against a binary from git.
func TestTheShippedDecoderRefusesADocumentsKeyByName(t *testing.T) {
	span := "gdoc:\n  schema: 2\n  documents:\n    - id: " + testDocumentID + "\n"
	_, err := decodeSchemaOne(span)
	if err == nil {
		t.Fatal("the schema 1 decoder read a schema 2 block")
	}
	if !strings.Contains(err.Error(), "documents") {
		t.Errorf("the refusal %q does not name the key it did not know", err)
	}
}

// A note carrying a copy of a document names the note it was copied from, and
// nothing in this package reads that field beyond carrying it. The refusal on
// it is cmd/gdoc's, where the writers are.
func TestExportedNoteSurvivesAWrite(t *testing.T) {
	src := fixture(t, "minimal.md")
	b, err := Read(src)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	b.Documents[0].Exported = &Exported{
		At:   time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC),
		Note: "../notes/supplier-register.md",
	}
	out, err := Write(src, b)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	back, err := Read(out)
	if err != nil {
		t.Fatalf("Read after Write: %v", err)
	}
	if back.Documents[0].Exported == nil || back.Documents[0].Exported.Note != "../notes/supplier-register.md" {
		t.Errorf("exported = %+v", back.Documents[0].Exported)
	}
}

// fixturePath is testdata's own path, for the tests that write a file rather
// than read one.
func fixturePath(name string) string { return filepath.Join("testdata", name) }

// The fixtures this file adds are read by name elsewhere, so a rename that
// forgets one fails here rather than in a test that then skips silently.
func TestTheSchemaTwoFixturesAreThere(t *testing.T) {
	for _, name := range []string{"two-documents.md", "entry-unknown-key.md", "schema3.md"} {
		if _, err := os.Stat(fixturePath(name)); err != nil {
			t.Errorf("fixture %s: %v", name, err)
		}
	}
}
