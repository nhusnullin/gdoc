package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	preludeDocID = "1AbCdEfGhIjKlMnOpQrStUvWxYz012345publish"
	pictureDocID = "1PiCtUrE0000000000000000000000000000pic"
)

// exportSession is the two answers an export needs, and no others: the Docs
// read and the docx export. A command that grew a third request fails here
// naming the URL it asked for.
func exportSession(t *testing.T, fixture string, export []byte) *fakeSession {
	t.Helper()
	return &fakeSession{
		json:  map[string]string{"docs.googleapis.com": readFixture(t, fixture)},
		bytes: map[string][]byte{"/export?": export},
	}
}

// emptyExport is a docx carrying no picture at all, which is what the export
// of a document with no picture in it looks like.
func emptyExport(t *testing.T) []byte {
	t.Helper()
	return zipOf(t, map[string][]byte{
		"word/document.xml": []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body/></w:document>`),
	})
}

// onePictureExport is the export of picture.json: one drawing in the body, the
// relationship it points at, and the bytes that relationship resolves to.
func onePictureExport(t *testing.T, png []byte) []byte {
	t.Helper()
	const (
		wNS      = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
		aNS      = "http://schemas.openxmlformats.org/drawingml/2006/main"
		rNS      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
		pkgRelNS = "http://schemas.openxmlformats.org/package/2006/relationships"
		imageRel = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
	)
	return zipOf(t, map[string][]byte{
		"word/document.xml": []byte(`<w:document xmlns:w="` + wNS + `" xmlns:a="` + aNS + `" xmlns:r="` + rNS +
			`"><w:body><w:p><w:r><w:drawing><a:blip r:embed="rId7"/></w:drawing></w:r></w:p></w:body></w:document>`),
		"word/_rels/document.xml.rels": []byte(`<Relationships xmlns="` + pkgRelNS +
			`"><Relationship Id="rId7" Type="` + imageRel + `" Target="media/image1.png"/></Relationships>`),
		"word/media/image1.png": png,
	})
}

func zipOf(t *testing.T, parts map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// put writes a file under dir and answers its path.
func put(t *testing.T, dir, name, text string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// A document is what this command is pointed at, so everything that is not one
// is refused by name and before a request: a note on this machine, a Drive
// folder, and a word that is neither.
func TestExportRefusesAFileThatIsNotADocument(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "note.md")

	for _, target := range []string{
		"note.md",
		"https://drive.google.com/drive/folders/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
		"supplier register",
		"",
	} {
		got, code := runJSON(t, "export", target, "--out", out)
		if code == 0 || got["ok"] != false {
			t.Errorf("export %q must be refused: %v (exit %d)", target, got, code)
			continue
		}
		msg, _ := got["error"].(string)
		if !strings.Contains(msg, "document") {
			t.Errorf("export %q must be refused naming what it needs: %q", target, msg)
		}
		if _, err := os.Lstat(out); err == nil {
			t.Errorf("export %q wrote a file on its way to a refusal", target)
		}
	}
}

// The envelope is what the skill reads, so every count in it is checked here
// against a fixture that states it. Four documents, one fact each: the note
// that lists proposals, the note written before the list existed, the house
// prelude, and the picture.
func TestExportEnvelopeCountsPendingOwnThreadsAndStripped(t *testing.T) {
	t.Run("a note listing gdoc's own proposals", func(t *testing.T) {
		dir := t.TempDir()
		out := put(t, dir, "note.md", "---\ntitle: Supplier register policy\ngdoc:\n"+
			"  schema: 2\n  documents:\n    - id: "+fixtureDocID+"\n      proposals:\n"+
			"        - id: suggest.a1\n          comment_id: CCCC3333\n          at: 2026-09-01T09:00:00Z\n"+
			"---\n\n# Scope\n\nThe supplier register is reviewed annually.\n")
		stubSession(t, exportSession(t, "single-tab.json", emptyExport(t)))

		got, code := runJSON(t, "export", fixtureDocID, "--out", out)
		if code != 0 || got["ok"] != true {
			t.Fatalf("export: %v (exit %d)", got, code)
		}
		data := dataOf(t, got)
		if data["pending"] != float64(1) {
			t.Errorf("pending = %v, want 1: the fixture holds one replacement, under one id", data["pending"])
		}
		if data["own"] != float64(1) {
			t.Errorf("own = %v, want 1: the note lists that id under proposals", data["own"])
		}
		if data["threads"] != float64(2) {
			t.Errorf("threads = %v, want 2: one anchored comment and one the read placed nowhere", data["threads"])
		}
		if stripped, ok := data["stripped"].([]any); !ok || len(stripped) != 0 {
			t.Errorf("stripped = %v, want nothing: this document carries no house prelude", data["stripped"])
		}

		// The path held a note for this document, so the copy lands beside it
		// and the note is stamped.
		files, _ := data["files"].([]any)
		if len(files) != 1 {
			t.Fatalf("one tab is one file: %v", data["files"])
		}
		file, _ := files[0].(map[string]any)
		if file["path"] != filepath.Join(dir, "note.2.md") {
			t.Errorf("path = %v, want note.2.md beside the note", file["path"])
		}
		if file["note"] != "note.md" {
			t.Errorf("note = %v, want the note the copy belongs to", file["note"])
		}
		// Every file says which tab it came from, one-tab documents included:
		// the envelope lists the files, and a reader should not have to count
		// tabs to know which is which.
		if file["tab_id"] != "t.0" {
			t.Errorf("tab_id = %v, want the tab the file came from", file["tab_id"])
		}
		stamped, _ := data["stamped"].([]any)
		if len(stamped) != 1 {
			t.Fatalf("the note must be stamped once: %v", data["stamped"])
		}
		if hasWarning(warningsOf(t, got), "rewritten from schema") {
			t.Error("this note was already schema 2, so nothing was rewritten and nothing should say so")
		}
	})

	t.Run("a note written before the block was a list", func(t *testing.T) {
		dir := t.TempDir()
		out := put(t, dir, "note.md", readFixture(t, "paired.md"))
		stubSession(t, exportSession(t, "single-tab.json", emptyExport(t)))

		got, code := runJSON(t, "export", fixtureDocID, "--out", out)
		if code != 0 || got["ok"] != true {
			t.Fatalf("export: %v (exit %d)", got, code)
		}
		if _, has := dataOf(t, got)["own"]; has {
			t.Error("this note lists no proposals, so there is nothing to count and the field must be absent")
		}
		if !hasWarning(warningsOf(t, got), "rewritten from schema 1 to schema 2") {
			t.Errorf("the stamp moved the block to schema 2 and must say so once: %v", warningsOf(t, got))
		}
	})

	t.Run("a document publish made", func(t *testing.T) {
		dir := t.TempDir()
		out := filepath.Join(dir, "policy.md")
		stubSession(t, exportSession(t, "publish-prelude.json", emptyExport(t)))

		got, code := runJSON(t, "export", preludeDocID, "--out", out)
		if code != 0 || got["ok"] != true {
			t.Fatalf("export: %v (exit %d)", got, code)
		}
		data := dataOf(t, got)
		stripped, _ := data["stripped"].([]any)
		if len(stripped) == 0 {
			t.Fatalf("the house prelude must be stripped and listed: %v", data["stripped"])
		}
		kinds := map[string]bool{}
		for _, raw := range stripped {
			piece, _ := raw.(map[string]any)
			kind, _ := piece["kind"].(string)
			kinds[kind] = true
			if text, _ := piece["text"].(string); text == "" {
				t.Errorf("the piece %q carries no text, and the text is what the session compares", kind)
			}
		}
		for _, want := range []string{"cover", "table", "contents"} {
			if !kinds[want] {
				t.Errorf("the stripped pieces must name the %s: %v", want, kinds)
			}
		}
		body, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "\n# Scope\n") {
			t.Errorf("the file must open at the first body heading:\n%s", body)
		}
	})

	t.Run("a picture", func(t *testing.T) {
		dir := t.TempDir()
		out := filepath.Join(dir, "chart.md")
		png := []byte("\x89PNG\r\n\x1a\nthe chart bytes")
		stubSession(t, exportSession(t, "picture.json", onePictureExport(t, png)))

		got, code := runJSON(t, "export", pictureDocID, "--out", out)
		if code != 0 || got["ok"] != true {
			t.Fatalf("export: %v (exit %d)", got, code)
		}
		pictures, _ := dataOf(t, got)["pictures"].([]any)
		if len(pictures) != 1 {
			t.Fatalf("the document holds one picture: %v", dataOf(t, got)["pictures"])
		}
		picture, _ := pictures[0].(map[string]any)
		if picture["file"] != "assets/chart-1.png" {
			t.Errorf("file = %v, want assets/chart-1.png beside the note", picture["file"])
		}
		if m, has := picture["matched"]; has {
			t.Errorf("matched = %v, want nothing: there is no note here holding those bytes", m)
		}
		written, err := os.ReadFile(filepath.Join(dir, "assets", "chart-1.png"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(written, png) {
			t.Errorf("the picture on disk is not the bytes the export carried")
		}
		body, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "![](assets/chart-1.png)") {
			t.Errorf("the body must name the file the picture landed in:\n%s", body)
		}
	})
}

// export is a read. The fake session fails the run on any write verb, so the
// assertion is the run succeeding, and the policy is asked the same question
// from the other side: a batch update and a create on the one document it was
// given are both refused inside the process.
func TestExportSendsNothingThatWrites(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "note.md")
	f := stubSession(t, exportSession(t, "single-tab.json", emptyExport(t)))

	got, code := runJSON(t, "export", fixtureDocID, "--out", out)
	if code != 0 || got["ok"] != true {
		t.Fatalf("export: %v (exit %d)", got, code)
	}

	// Two requests, both GETs, both about the document it was given.
	if len(f.urls) != 2 {
		t.Errorf("export makes the Docs read and the docx export and nothing else: %v", f.urls)
	}
	for _, raw := range f.urls {
		if !strings.Contains(raw, fixtureDocID) {
			t.Errorf("a request named a document this run was not given: %s", raw)
		}
	}

	if f.policy == nil {
		t.Fatal("the command opened no policy")
	}
	for _, refused := range []struct{ method, url, body string }{
		{"POST", "https://docs.googleapis.com/v1/documents/" + fixtureDocID + ":batchUpdate", `{"requests":[]}`},
		{"PATCH", "https://www.googleapis.com/drive/v3/files/" + fixtureDocID, `{"name":"x"}`},
		{"DELETE", "https://www.googleapis.com/drive/v3/files/" + fixtureDocID, ""},
		{"POST", "https://www.googleapis.com/drive/v3/files", `{"parents":["ANY"]}`},
	} {
		if err := f.policy.Judge(refused.method, mustParse(t, refused.url), []byte(refused.body)); err == nil {
			t.Errorf("%s %s must be refused on an export's policy", refused.method, refused.url)
		}
	}
}
