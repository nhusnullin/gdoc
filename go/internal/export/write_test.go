package export

import (
	"crypto/sha256"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/frontmatter"
)

const (
	testDocID  = "1AbCdEfGhIjKlMnOpQrStUvWxYz012345"
	otherDocID = "1ZyXwVuTsRqPoNmLkJiHgFeDcBa543210"
)

var testAt = time.Date(2026, 9, 19, 14, 0, 0, 0, time.UTC)

// one is the request for a one-tab document, with whatever pictures the test
// hands it.
func one(dir string, pics ...Picture) Request {
	return Request{
		Out:        filepath.Join(dir, "note.md"),
		DocumentID: testDocID,
		At:         testAt,
		Tabs:       []Tab{{ID: "t.0", Title: "Scope"}},
		Pictures:   pics,
	}
}

// run plans and writes in one step, which is what every caller does: the two
// halves exist so the paths are known before the bodies name them.
func run(t *testing.T, req Request, bodies ...string) (*Written, error) {
	t.Helper()
	l, err := Plan(req)
	if err != nil {
		return nil, err
	}
	files := make([]TabFile, 0, len(req.Tabs))
	for i, tab := range req.Tabs {
		body := ""
		if i < len(bodies) {
			body = bodies[i]
		}
		files = append(files, TabFile{TabID: tab.ID, TabTitle: tab.Title, Body: body})
	}
	return Write(l, files)
}

func readText(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func writeText(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

// note is a paired note as publish leaves one: the block, a title the author
// owns, and a body.
func note(id string) string {
	return "---\ntitle: The scope\ngdoc:\n  schema: 2\n  documents:\n    - id: " + id +
		"\n---\n\n# Scope\n\nThe note's own words.\n"
}

// TestAFreePathIsTaken is scenario 1: a document, no note, and nothing at the
// path. The file lands where it was asked for, its picture lands in a folder
// this run made, and the block names the document and the day it was read.
func TestAFreePathIsTaken(t *testing.T) {
	dir := t.TempDir()
	w, err := run(t, one(dir, Picture{ObjectID: "kix.one", Ext: ".png", Bytes: []byte("first")}), "# Scope\n\n![](assets/note-1.png)\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(w.Files) != 1 {
		t.Fatalf("Files = %d, want one per tab", len(w.Files))
	}
	if got, want := w.Files[0].Path, filepath.Join(dir, "note.md"); got != want {
		t.Errorf("Path = %s, want %s: the path was free", got, want)
	}
	if w.Files[0].Taken || w.Files[0].Note != "" {
		t.Errorf("a free path came back as taken (%v) or beside a note (%q)", w.Files[0].Taken, w.Files[0].Note)
	}
	if len(w.Stamped) != 0 {
		t.Errorf("Stamped = %v, want nothing: there was no note to stamp", w.Stamped)
	}

	body := readText(t, filepath.Join(dir, "note.md"))
	if !strings.Contains(body, "# Scope") {
		t.Errorf("the body is not in the file:\n%s", body)
	}
	b, err := frontmatter.Read([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if b.Schema != frontmatter.Schema || len(b.Documents) != 1 {
		t.Fatalf("block = %+v, want schema %d and one entry", b, frontmatter.Schema)
	}
	e := b.Documents[0]
	if e.ID != testDocID {
		t.Errorf("id = %s, want the document this export read", e.ID)
	}
	if e.TabID != "" {
		t.Errorf("tab_id = %q, want nothing: a document with one tab is one file", e.TabID)
	}
	if e.Exported == nil || !e.Exported.At.Equal(testAt) {
		t.Errorf("exported = %+v, want the time the document was read", e.Exported)
	}
	if e.Exported != nil && e.Exported.Note != "" {
		t.Errorf("exported.note = %q, want nothing: this file is the note", e.Exported.Note)
	}

	if got := w.Pictures[0].File; got != "assets/note-1.png" {
		t.Errorf("the picture's file = %q, want assets/note-1.png", got)
	}
	if got := readText(t, filepath.Join(dir, "assets", "note-1.png")); got != "first" {
		t.Errorf("the picture holds %q, want the bytes the export carried", got)
	}
	info, err := os.Stat(filepath.Join(dir, "assets"))
	if err != nil {
		t.Fatalf("the assets folder was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("assets is not a folder")
	}
}

// TestTheNamesReachTheBody is the reason the paths are decided before
// anything is projected: the picture's line in the body names the file it
// lands in, and a picture the note already holds keeps the note's own link.
func TestTheNamesReachTheBody(t *testing.T) {
	dir := t.TempDir()
	l, err := Plan(one(dir,
		Picture{ObjectID: "kix.one", Ext: ".png", Bytes: []byte("first")},
		Picture{ObjectID: "kix.two", Matched: "assets/diagram.png"},
		Picture{ObjectID: "kix.none"},
	))
	if err != nil {
		t.Fatal(err)
	}
	names := l.PictureNames()
	if got, want := names("kix.one"), "![](assets/note-1.png)"; got != want {
		t.Errorf("the written picture = %q, want %q", got, want)
	}
	if got, want := names("kix.two"), "![](assets/diagram.png)"; got != want {
		t.Errorf("the matched picture = %q, want the note's own link %q", got, want)
	}
	if got := names("kix.none"); got != "" {
		t.Errorf("a picture with no bytes = %q, want nothing, so the placeholder stays", got)
	}
}

// TestATakenPathGetsTheNextFreeNumber is decision 2. A taken name takes the
// next free number, and nothing that was there is touched.
func TestATakenPathGetsTheNextFreeNumber(t *testing.T) {
	dir := t.TempDir()
	writeText(t, filepath.Join(dir, "note.md"), "somebody else's file\n")
	writeText(t, filepath.Join(dir, "assets", "note-1.png"), "somebody else's picture\n")
	before := hashes(t, dir)

	w, err := run(t, one(dir, Picture{ObjectID: "kix.one", Ext: ".png", Bytes: []byte("first")}), "# Scope\n")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := w.Files[0].Path, filepath.Join(dir, "note.2.md"); got != want {
		t.Errorf("Path = %s, want %s", got, want)
	}
	if !w.Files[0].Taken {
		t.Error("the path was taken and the reply does not say so")
	}
	if got := w.Pictures[0].File; got != "assets/note-2.png" {
		t.Errorf("the picture's file = %q, want assets/note-2.png: the first number was taken", got)
	}

	w, err = run(t, one(dir, Picture{ObjectID: "kix.one", Ext: ".png", Bytes: []byte("second")}), "# Scope\n")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := w.Files[0].Path, filepath.Join(dir, "note.3.md"); got != want {
		t.Errorf("Path = %s, want %s", got, want)
	}
	if got := w.Pictures[0].File; got != "assets/note-3.png" {
		t.Errorf("the picture's file = %q, want assets/note-3.png", got)
	}

	for path, sum := range before {
		if got := hashOf(t, path); got != sum {
			t.Errorf("%s changed, and nothing an export finds in its way is ever touched", path)
		}
	}
}

// TestNothingCanReplaceAFile is the invariant read off the source: there is no
// force anywhere in this package, every new file goes through
// atomicfile.Create, which refuses a path that exists, and the one call to
// Replace is the stamp on a note that was already there.
func TestNothingCanReplaceAFile(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(f os.FileInfo) bool {
		return !strings.HasSuffix(f.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, ok := pkgs["export"]
	if !ok {
		t.Fatal("the export package did not parse")
	}

	replaceIn := map[string]bool{}
	creates := 0
	for name, f := range pkg.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && strings.Contains(strings.ToLower(id.Name), "force") {
				t.Errorf("%s names %s: this package has no force, and no flag that could replace a file", name, id.Name)
			}
			return true
		})
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgName, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				switch {
				case pkgName.Name == "atomicfile" && sel.Sel.Name == "Replace":
					replaceIn[fn.Name.Name] = true
				case pkgName.Name == "atomicfile" && sel.Sel.Name == "Create":
					creates++
				case pkgName.Name == "os" && (sel.Sel.Name == "WriteFile" || sel.Sel.Name == "Create" ||
					sel.Sel.Name == "OpenFile" || sel.Sel.Name == "Rename" || sel.Sel.Name == "Remove" ||
					sel.Sel.Name == "RemoveAll" || sel.Sel.Name == "Truncate"):
					t.Errorf("%s calls os.%s: every file this package writes goes through atomicfile, which cannot replace one",
						fn.Name.Name, sel.Sel.Name)
				}
				return true
			})
		}
	}
	if creates == 0 {
		t.Error("nothing in this package calls atomicfile.Create, so something else is writing the files")
	}
	if len(replaceIn) != 1 || !replaceIn["stamp"] {
		t.Errorf("atomicfile.Replace is called in %v, want the stamp on a note and nowhere else", replaceIn)
	}
}

// TestExportStampsTheNoteAndChangesNoOtherByte is scenario 2: the note is left
// as it stands but for one dated fact, and the copy lands beside it knowing
// which note it belongs to.
func TestExportStampsTheNoteAndChangesNoOtherByte(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	writeText(t, path, note(testDocID))
	before := readText(t, path)

	w, err := run(t, one(dir), "# Scope\n\nThe document's words.\n")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := w.Files[0].Path, filepath.Join(dir, "note.2.md"); got != want {
		t.Errorf("Path = %s, want %s: the note keeps its name", got, want)
	}
	if got := w.Files[0].Note; got != "note.md" {
		t.Errorf("Note = %q, want the note this copy belongs to, relative to its own directory", got)
	}
	if len(w.Stamped) != 1 || w.Stamped[0].Path != path {
		t.Fatalf("Stamped = %+v, want the note", w.Stamped)
	}
	if w.Stamped[0].SchemaRewritten {
		t.Error("the note was already schema 2 and the reply says it was rewritten")
	}

	after := readText(t, path)
	if bodyOf(before) != bodyOf(after) {
		t.Errorf("the note's body changed:\n%s", after)
	}
	added := addedLines(t, before, after)
	for _, line := range added {
		if !strings.Contains(line, "exported") && !strings.Contains(line, "at:") {
			t.Errorf("the note gained the line %q, and the stamp is the only thing export writes into a note", line)
		}
	}
	if len(added) == 0 {
		t.Error("the note gained nothing, and it was supposed to gain the stamp")
	}
	b, err := frontmatter.Read([]byte(after))
	if err != nil {
		t.Fatal(err)
	}
	if b.Documents[0].Exported == nil || !b.Documents[0].Exported.At.Equal(testAt) {
		t.Errorf("exported = %+v, want the time the document was read", b.Documents[0].Exported)
	}

	copied, err := frontmatter.Read([]byte(readText(t, filepath.Join(dir, "note.2.md"))))
	if err != nil {
		t.Fatal(err)
	}
	if copied.Documents[0].Exported.Note != "note.md" {
		t.Errorf("the copy's exported.note = %q, want note.md, which is what makes every writer refuse it",
			copied.Documents[0].Exported.Note)
	}
}

// TestASchemaOneNoteIsRewrittenAndSaidSo is scenario 23: every note on the
// team today is schema 1, and the stamp is the write that moves it.
func TestASchemaOneNoteIsRewrittenAndSaidSo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	writeText(t, path, "---\ngdoc:\n  schema: 1\n  document_id: "+testDocID+"\n---\n\n# Scope\n")

	w, err := run(t, one(dir), "# Scope\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Stamped) != 1 || !w.Stamped[0].SchemaRewritten {
		t.Fatalf("Stamped = %+v, want the note and the line saying its block was rewritten", w.Stamped)
	}
	b, err := frontmatter.Read([]byte(readText(t, path)))
	if err != nil {
		t.Fatal(err)
	}
	if b.Schema != frontmatter.Schema {
		t.Errorf("schema = %d, want %d after a write that changed the block", b.Schema, frontmatter.Schema)
	}
	if b.Documents[0].Exported == nil {
		t.Error("the old block's one document did not get the stamp")
	}
}

// TestTheDoorChecks is the two refusals the spec names, each answered before a
// byte reaches the disk.
func TestTheDoorChecks(t *testing.T) {
	t.Run("a note naming other documents", func(t *testing.T) {
		dir := t.TempDir()
		writeText(t, filepath.Join(dir, "note.md"), note(otherDocID))
		_, err := run(t, one(dir), "# Scope\n")
		if err == nil {
			t.Fatal("a note paired with another document was written into, want a refusal")
		}
		if !strings.Contains(err.Error(), otherDocID) {
			t.Errorf("err = %v, want the documents the note does name", err)
		}
		nothingNew(t, dir, "note.md")
	})

	t.Run("broken front matter", func(t *testing.T) {
		dir := t.TempDir()
		writeText(t, filepath.Join(dir, "note.md"), "---\ngdoc:\n  schema: 2\n  documents: [{id: nope}]\n---\n\nwords\n")
		_, err := run(t, one(dir), "# Scope\n")
		if err == nil {
			t.Fatal("a note whose block does not read was written beside, want a refusal")
		}
		if !strings.Contains(err.Error(), "note.md") {
			t.Errorf("err = %v, want the file it refused", err)
		}
		nothingNew(t, dir, "note.md")
	})

	t.Run("any other file is taken, not refused", func(t *testing.T) {
		dir := t.TempDir()
		writeText(t, filepath.Join(dir, "note.md"), "# Somebody's own file\n")
		w, err := run(t, one(dir), "# Scope\n")
		if err != nil {
			t.Fatalf("a file that is not a note is not a refusal: %v", err)
		}
		if got, want := w.Files[0].Path, filepath.Join(dir, "note.2.md"); got != want {
			t.Errorf("Path = %s, want %s", got, want)
		}
	})
}

// TestATabPathIsCheckedLikeOut is decision 11: every tab's path takes the
// checks --out takes, and one bad path refuses the whole run before any file
// is written.
func TestATabPathIsCheckedLikeOut(t *testing.T) {
	dir := t.TempDir()
	writeText(t, filepath.Join(dir, "note-appendix.md"), note(otherDocID))

	req := one(dir)
	req.Tabs = []Tab{{ID: "t.0", Title: "Scope"}, {ID: "t.1", Title: "Appendix"}}
	_, err := run(t, req, "# Scope\n", "# Appendix\n")
	if err == nil {
		t.Fatal("a second tab's path holding another document's note was written to, want a refusal")
	}
	if !strings.Contains(err.Error(), "note-appendix.md") {
		t.Errorf("err = %v, want the tab's own path", err)
	}
	nothingNew(t, dir, "note-appendix.md")
}

// TestTabSlugsCollide is the naming rule for a document with several tabs: the
// slug of the title, tab-<n> when there is no title, and the numbering rule of
// decision 2 when two tabs are called the same thing.
func TestTabSlugsCollide(t *testing.T) {
	dir := t.TempDir()
	req := one(dir)
	req.Tabs = []Tab{
		{ID: "t.0", Title: "Scope"},
		{ID: "t.1", Title: "Appendix A & B"},
		{ID: "t.2", Title: ""},
		{ID: "t.3", Title: "Appendix A & B"},
	}
	w, err := run(t, req, "one\n", "two\n", "three\n", "four\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"note.md", "note-appendix-a-b.md", "note-tab-3.md", "note-appendix-a-b.2.md"}
	for i, name := range want {
		if got := filepath.Base(w.Files[i].Path); got != name {
			t.Errorf("tab %d landed at %s, want %s", i, got, name)
		}
		if _, err := os.Stat(w.Files[i].Path); err != nil {
			t.Errorf("%s was not written: %v", name, err)
		}
	}

	b, err := frontmatter.Read([]byte(readText(t, w.Files[1].Path)))
	if err != nil {
		t.Fatal(err)
	}
	if got := b.Documents[0].TabID; got != "t.1" {
		t.Errorf("tab_id = %q, want t.1: a file from one tab of several says which", got)
	}
}

// nothingNew is the directory holding what it held before the run and nothing
// else. A refusal at the door writes no file at all.
func nothingNew(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	held := map[string]bool{}
	for _, name := range want {
		held[name] = true
	}
	for _, e := range entries {
		if !held[e.Name()] {
			t.Errorf("%s was written, and a refusal at the door writes nothing", e.Name())
		}
	}
}

// hashes is every file under dir by path, so a test can say nothing that was
// there changed.
func hashes(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		out[path] = hashOf(t, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func hashOf(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return string(sum[:])
}

// bodyOf is everything after the front matter, which is the part of a note
// export never touches.
func bodyOf(src string) string {
	_, body, ok := strings.Cut(strings.TrimPrefix(src, "---\n"), "\n---\n")
	if !ok {
		return src
	}
	return body
}

// addedLines is what the second file has that the first did not, in order. It
// fails the test when a line went missing, because the stamp only ever adds.
func addedLines(t *testing.T, before, after string) []string {
	t.Helper()
	old := strings.Split(before, "\n")
	var added []string
	i := 0
	for _, line := range strings.Split(after, "\n") {
		if i < len(old) && old[i] == line {
			i++
			continue
		}
		added = append(added, line)
	}
	if i != len(old) {
		t.Errorf("the note lost the line %q, and the stamp only adds", old[i])
	}
	return added
}
