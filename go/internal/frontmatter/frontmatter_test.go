package frontmatter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return src
}

func TestReadFindsNoBlockToRead(t *testing.T) {
	cases := []string{"none.md", "no-gdoc.md"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			b, err := Read(fixture(t, name))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if b != nil {
				t.Fatalf("got a block from %s: %+v", name, b)
			}
		})
	}
}

func TestReadAnEmptyFileIsNoBlock(t *testing.T) {
	b, err := Read(nil)
	if err != nil || b != nil {
		t.Fatalf("Read(nil) = %v, %v; want nil, nil", b, err)
	}
}

func TestReadDecodesTheFullBlock(t *testing.T) {
	b, err := Read(fixture(t, "full.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil {
		t.Fatal("no block read from full.md")
	}
	if b.Schema != 1 {
		t.Errorf("schema = %d, want 1", b.Schema)
	}
	if b.DocumentID != testDocumentID {
		t.Errorf("document_id = %q, want %q", b.DocumentID, testDocumentID)
	}
	if b.FolderID != "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr" {
		t.Errorf("folder_id = %q", b.FolderID)
	}
	if b.Published == nil {
		t.Fatal("published is absent")
	}
	wantPublished := time.Date(2026, 9, 6, 10, 12, 0, 0, time.UTC)
	if !b.Published.At.Equal(wantPublished) {
		t.Errorf("published.at = %v, want %v", b.Published.At, wantPublished)
	}
	if b.Published.RevisionID != "ALm37BX" {
		t.Errorf("published.revision_id = %q", b.Published.RevisionID)
	}
	if b.SuggestionsSeen == nil {
		t.Fatal("suggestions_seen is absent")
	}
	if got := len(b.SuggestionsSeen.Items); got != 2 {
		t.Fatalf("suggestions_seen.items = %d, want 2", got)
	}
	first := b.SuggestionsSeen.Items[0]
	want := SuggestionSeen{ID: "suggest.abc123", Kind: KindInsertion, Section: "Scope", Text: "critical "}
	if first != want {
		t.Errorf("items[0] = %+v, want %+v", first, want)
	}
	second := b.SuggestionsSeen.Items[1]
	if second.Kind != KindDeletion || second.Section != "" || second.Text != "annually" {
		t.Errorf("items[1] = %+v", second)
	}
	if len(b.Proposals) != 1 {
		t.Fatalf("proposals = %d, want 1", len(b.Proposals))
	}
	if b.Proposals[0].ID != "gdoc.p1" || b.Proposals[0].CommentID != "AAAABBBBCCCC" {
		t.Errorf("proposals[0] = %+v", b.Proposals[0])
	}
}

func TestReadDecodesTheMinimalBlock(t *testing.T) {
	b, err := Read(fixture(t, "minimal.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil || b.Schema != 1 || b.DocumentID != testDocumentID {
		t.Fatalf("block = %+v", b)
	}
	if b.Published != nil || b.SuggestionsSeen != nil || len(b.Proposals) != 0 {
		t.Errorf("absent keys came back set: %+v", b)
	}
}

func TestReadDecodesThroughCRLF(t *testing.T) {
	b, err := Read(fixture(t, "crlf.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil || b.DocumentID != testDocumentID {
		t.Fatalf("block = %+v", b)
	}
}

func TestReadRefusesAndNamesWhatIsWrong(t *testing.T) {
	cases := []struct {
		file  string
		names string
	}{
		{"unknown-key.md", "reviewed_by"},
		{"duplicate-key.md", "schema"},
		{"schema2.md", "schema"},
		{"no-document-id.md", "document_id"},
		{"bad-kind.md", "kind"},
		{"twice.md", "gdoc"},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			b, err := Read(fixture(t, c.file))
			if err == nil {
				t.Fatalf("%s was accepted: %+v", c.file, b)
			}
			if b != nil {
				t.Errorf("a refused read still returned a block: %+v", b)
			}
			if !strings.Contains(err.Error(), c.names) {
				t.Fatalf("error %q does not name %q", err, c.names)
			}
		})
	}
}

func TestWriteAnUnchangedBlockIsByteIdentical(t *testing.T) {
	for _, name := range []string{"full.md", "minimal.md", "crlf.md"} {
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
				t.Fatalf("round trip changed the file.\nwant:\n%q\ngot:\n%q", src, out)
			}
		})
	}
}

// spanOf returns the lines before the gdoc: key and the lines from the next
// top-level key onwards. Every byte outside the span must survive a write.
func spanOf(t *testing.T, src []byte) (before, after string) {
	t.Helper()
	lines := strings.SplitAfter(string(src), "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "gdoc:") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("no gdoc: line in %q", src)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimRight(lines[i], "\r\n")
		if trimmed == "" || strings.HasPrefix(trimmed, " ") || strings.HasPrefix(trimmed, "\t") {
			continue
		}
		end = i
		break
	}
	return strings.Join(lines[:start], ""), strings.Join(lines[end:], "")
}

func TestWriteMovesNoLineOutsideTheBlock(t *testing.T) {
	src := fixture(t, "full.md")
	b, err := Read(src)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	b.FolderID = "2ZzYyXxWwVvUuTtSsRrQqPpOoNnMmLlKk"

	out, err := Write(src, b)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if bytes.Equal(src, out) {
		t.Fatal("the changed field did not reach the file")
	}
	if !bytes.Contains(out, []byte("2ZzYyXxWwVvUuTtSsRrQqPpOoNnMmLlKk")) {
		t.Fatal("the new folder id is not in the output")
	}

	srcBefore, srcAfter := spanOf(t, src)
	outBefore, outAfter := spanOf(t, out)
	if srcBefore != outBefore {
		t.Errorf("lines before the block moved.\nwant:\n%q\ngot:\n%q", srcBefore, outBefore)
	}
	if srcAfter != outAfter {
		t.Errorf("lines after the block moved.\nwant:\n%q\ngot:\n%q", srcAfter, outAfter)
	}
}

func TestWriteKeepsCRLF(t *testing.T) {
	src := fixture(t, "crlf.md")
	b, err := Read(src)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	b.FolderID = "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"

	out, err := Write(src, b)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if bytes.Contains(bytes.ReplaceAll(out, []byte("\r\n"), nil), []byte("\n")) {
		t.Fatalf("a bare newline reached a CRLF file: %q", out)
	}
	if !bytes.Contains(out, []byte("  folder_id: 1w0SresizE9Kr810VZRJwX4JtDBF4OqNr\r\n")) {
		t.Fatalf("the new line is not CRLF terminated: %q", out)
	}
	back, err := Read(out)
	if err != nil {
		t.Fatalf("re-reading what Write produced: %v", err)
	}
	if back.FolderID != b.FolderID {
		t.Errorf("folder_id = %q, want %q", back.FolderID, b.FolderID)
	}
}

func TestWriteGivesAFileWithoutFrontMatterDelimitersAndNothingElse(t *testing.T) {
	src := fixture(t, "none.md")
	out, err := Write(src, validBlock())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.HasSuffix(out, src) {
		t.Fatalf("the body was not left alone.\nwant suffix:\n%q\ngot:\n%q", src, out)
	}
	head := string(out[:len(out)-len(src)])
	want := "---\ngdoc:\n  schema: 1\n  document_id: " + testDocumentID + "\n---\n"
	if head != want {
		t.Fatalf("front matter = %q, want %q", head, want)
	}
	back, err := Read(out)
	if err != nil || back == nil || back.DocumentID != testDocumentID {
		t.Fatalf("re-read gave %+v, %v", back, err)
	}
}

func TestWriteAppendsIntoExistingFrontMatter(t *testing.T) {
	src := fixture(t, "no-gdoc.md")
	out, err := Write(src, validBlock())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Contains(out, []byte("title: Supplier register policy\n")) {
		t.Fatalf("the author's title line did not survive: %q", out)
	}
	if !bytes.Contains(out, []byte("author: Nail\n")) {
		t.Fatalf("the author's author line did not survive: %q", out)
	}
	if !bytes.Contains(out, []byte("\n# Scope\n")) {
		t.Fatalf("the body did not survive: %q", out)
	}
	if bytes.Count(out, []byte("\n---")) != 1 || !bytes.HasPrefix(out, []byte("---\n")) {
		t.Fatalf("the delimiters were not left as they were: %q", out)
	}
	back, err := Read(out)
	if err != nil || back == nil || back.DocumentID != testDocumentID {
		t.Fatalf("re-read gave %+v, %v", back, err)
	}
}

func TestWriteRefusesAnInvalidBlockAndTouchesNothing(t *testing.T) {
	src := fixture(t, "full.md")
	out, err := Write(src, &Block{Schema: 1})
	if err == nil {
		t.Fatal("an invalid block was written")
	}
	if !strings.Contains(err.Error(), "document_id") {
		t.Errorf("error %q does not name document_id", err)
	}
	if out != nil {
		t.Errorf("a refused write returned bytes: %q", out)
	}
}

func TestWriteRefusesAFileItCouldNotRead(t *testing.T) {
	src := fixture(t, "unknown-key.md")
	out, err := Write(src, validBlock())
	if err == nil {
		t.Fatal("a file whose block could not be read was overwritten")
	}
	if out != nil {
		t.Errorf("a refused write returned bytes: %q", out)
	}
}

func TestWriteRefusesANilBlock(t *testing.T) {
	if _, err := Write(fixture(t, "full.md"), nil); err == nil {
		t.Fatal("a nil block was written")
	}
}

func TestReadRefusesAnEmptyGdocKey(t *testing.T) {
	b, err := Read(fixture(t, "empty-gdoc.md"))
	if err == nil {
		t.Fatalf("an empty gdoc: key was accepted: %+v", b)
	}
	if !strings.Contains(err.Error(), "gdoc") {
		t.Errorf("error %q does not name gdoc", err)
	}
}

func TestReadRefusesFrontMatterTheAuthorBroke(t *testing.T) {
	if _, err := Read(fixture(t, "bad-yaml.md")); err == nil {
		t.Fatal("front matter that is not YAML was accepted")
	}
}

func TestReadTreatsAnUnclosedDelimiterAsNoFrontMatter(t *testing.T) {
	b, err := Read(fixture(t, "unterminated.md"))
	if err != nil || b != nil {
		t.Fatalf("Read = %+v, %v; want nil, nil", b, err)
	}
}

func TestWriteGivesBackTheBlankLineAfterTheBlock(t *testing.T) {
	src := fixture(t, "blank-line.md")
	b, err := Read(src)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if b == nil {
		t.Fatal("no block read")
	}
	out, err := Write(src, b)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Equal(src, out) {
		t.Fatalf("the blank line before author: moved.\nwant:\n%q\ngot:\n%q", src, out)
	}
}
