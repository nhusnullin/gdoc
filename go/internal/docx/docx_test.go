package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/guard"
)

const testDocID = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"

// buildDocx zips the named parts into a docx in memory. The fixture is two
// readable XML files under testdata/ rather than a binary blob, so a test that
// fails can be read against the thing it parsed.
func buildDocx(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// wholeExport is the docx a real export looks like here: a document with one
// comment still anchored to text and one that lost its range.
func wholeExport(t *testing.T) []byte {
	t.Helper()
	return buildDocx(t, map[string]string{
		"word/document.xml": fixture(t, "document.xml"),
		"word/comments.xml": fixture(t, "comments.xml"),
	})
}

func TestParseFindsTheAnchoredCommentAndTheDetachedOne(t *testing.T) {
	f, err := Parse(wholeExport(t))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(f.Comments) != 2 {
		t.Fatalf("len(Comments) = %d, want 2: %+v", len(f.Comments), f.Comments)
	}

	anchored := f.Comments[0]
	if anchored.ID != "0" {
		t.Errorf("ID = %q, want 0", anchored.ID)
	}
	if anchored.Author != "Nail Khusnullin" {
		t.Errorf("Author = %q", anchored.Author)
	}
	if anchored.Date != "2026-09-06T10:12:00Z" {
		t.Errorf("Date = %q", anchored.Date)
	}
	if anchored.Text != "ai? which register does this refer to" {
		t.Errorf("Text = %q; the runs of one comment are one comment", anchored.Text)
	}
	if !anchored.Anchored {
		t.Error("Anchored = false, and word/document.xml carries a commentRangeStart for this id")
	}
	if anchored.Span != "the operations team" {
		t.Errorf("Span = %q, want the text between the range start and the range end", anchored.Span)
	}

	detached := f.Comments[1]
	if detached.ID != "1" {
		t.Errorf("ID = %q, want 1", detached.ID)
	}
	if detached.Anchored {
		t.Error("Anchored = true for a comment word/document.xml never opens a range for")
	}
	if detached.Span != "" {
		t.Errorf("Span = %q, and a detached comment spans nothing", detached.Span)
	}
}

func TestADocxWithoutCommentsHasNoComments(t *testing.T) {
	b := buildDocx(t, map[string]string{"word/document.xml": fixture(t, "document.xml")})

	f, err := Parse(b)
	if err != nil {
		t.Fatalf("a document nobody commented on is not an error: %v", err)
	}
	if len(f.Comments) != 0 {
		t.Errorf("Comments = %+v, want none", f.Comments)
	}
}

func TestBytesThatAreNotADocxAreRefusedByName(t *testing.T) {
	cases := []struct {
		name  string
		bytes []byte
		says  string
	}{
		{"not a zip at all", []byte("<html>Sign in to continue</html>"), "not a docx"},
		{"a zip with no document part", buildDocx(t, map[string]string{"hello.txt": "hi"}), "word/document.xml"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse(c.bytes)
			if err == nil {
				t.Fatal("Parse accepted bytes that are not a docx")
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("error = %q, and it does not name %q", err, c.says)
			}
		})
	}
}

// fakeReader answers with the bytes it was given and records what it was asked
// for. Nothing here names net/http: the rooms that may are the ones the
// boundary test lists, and this is not one of them.
type fakeReader struct {
	seen  []string
	limit int64
	body  []byte
	err   error
}

func (f *fakeReader) GetBytes(_ context.Context, rawURL string, limit int64) ([]byte, error) {
	f.seen = append(f.seen, rawURL)
	f.limit = limit
	if f.err != nil {
		return nil, f.err
	}
	return f.body, nil
}

func TestExportAsksForTheDocxMimeOnAllDrives(t *testing.T) {
	want := wholeExport(t)
	r := &fakeReader{body: want}

	got, err := Export(context.Background(), r, testDocID)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Error("Export returned bytes other than the ones the session read")
	}
	if len(r.seen) != 1 {
		t.Fatalf("the export made %d requests, want 1", len(r.seen))
	}
	if r.limit != MaxExportBytes {
		t.Errorf("limit = %d, want the export ceiling %d", r.limit, MaxExportBytes)
	}

	// The strongest thing a test can say about a URL this package builds: not
	// that it matches a list written twice, but that the guard carries it.
	u, err := url.Parse(r.seen[0])
	if err != nil {
		t.Fatalf("ExportURL built a URL that does not parse: %v", err)
	}
	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)
	if err := p.Judge("GET", u, nil); err != nil {
		t.Fatalf("the guard refused the docx export: %v", err)
	}
	if u.Path != "/drive/v3/files/"+testDocID+"/export" {
		t.Errorf("path = %q", u.Path)
	}
	q := u.Query()
	if q.Get("mimeType") != docxMime {
		t.Errorf("mimeType = %q, want the docx type", q.Get("mimeType"))
	}
	// files.export defines fileId and mimeType and nothing else, so a parameter
	// beyond them is one the server may reject and take the whole witness with.
	for _, name := range []string{"supportsAllDrives", "fields", "alt"} {
		if q.Has(name) {
			t.Errorf("the export carries %s, which files.export does not define", name)
		}
	}
}

func TestExportCarriesTheSessionsRefusal(t *testing.T) {
	r := &fakeReader{err: errRefused}

	if _, err := Export(context.Background(), r, testDocID); err == nil {
		t.Fatal("Export swallowed the refusal the session returned")
	} else if !strings.Contains(err.Error(), "refused") {
		t.Errorf("error = %q, and it does not carry what the session said", err)
	}
}

func TestATruncatedCommentPartIsRefusedRatherThanReadAsEmpty(t *testing.T) {
	b := buildDocx(t, map[string]string{
		"word/document.xml": fixture(t, "document.xml"),
		"word/comments.xml": `<w:comments xmlns:w="` + wNS + `"><w:comment w:id="0"><w:p><w:r><w:t>ai? cut off`,
	})

	if _, err := Parse(b); err == nil {
		t.Fatal("Parse read a truncated comments part as a document with no comments")
	} else if !strings.Contains(err.Error(), "word/comments.xml") {
		t.Errorf("error = %q, and it does not name the part that failed", err)
	}
}

func TestATruncatedDocumentPartIsRefused(t *testing.T) {
	b := buildDocx(t, map[string]string{
		"word/document.xml": `<w:document xmlns:w="` + wNS + `"><w:body><w:p>`,
	})

	if _, err := Parse(b); err == nil {
		t.Fatal("Parse read a truncated document part as a document with no ranges")
	} else if !strings.Contains(err.Error(), "word/document.xml") {
		t.Errorf("error = %q, and it does not name the part that failed", err)
	}
}
