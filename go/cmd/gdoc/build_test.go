package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/guard"
)

// buildNote copies the fixture note into a temp directory, because a build
// writes beside the note it was pointed at and a test must never write into
// testdata.
func buildNote(t *testing.T) (dir, md string) {
	t.Helper()
	dir = t.TempDir()
	src, err := os.ReadFile(filepath.Join("testdata", "build-note.md"))
	if err != nil {
		t.Fatal(err)
	}
	md = filepath.Join(dir, "supplier-register.md")
	if err := os.WriteFile(md, src, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, md
}

// writeNote is buildNote for a note a single test states itself.
func writeNote(t *testing.T, name, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// zipNames lists what the written file holds, in the order the archive does.
func zipNames(t *testing.T, path string) []string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("the built file is not a zip: %v", err)
	}
	defer r.Close()
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names
}

// noSession fails the test if anything opens a session. The build reaches no
// network at all, and the session factory is the only door to one.
func noSession(t *testing.T) {
	t.Helper()
	old := openSession
	openSession = func(*guard.Policy) (session, error) {
		t.Fatal("build must make no request: the session factory was called")
		return nil, nil
	}
	t.Cleanup(func() { openSession = old })
}

func TestBuildWritesADocxAndReportsWhatItWrote(t *testing.T) {
	noSession(t)
	dir, md := buildNote(t)
	out := filepath.Join(dir, "supplier-register.docx")

	got, code := runJSON(t, "build", "--md", md, "--out", out)
	if code != 0 || got["ok"] != true {
		t.Fatalf("build: %v (exit %d)", got, code)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	if data["out"] != out {
		t.Errorf("out names the file that was written: %v, want %s", data["out"], out)
	}
	if data["title"] != "Supplier Register Policy" {
		t.Errorf("title is the cover's own words: %v", data["title"])
	}
	if data["running_head"] != "Altery - Supplier Register Policy" {
		t.Errorf("running_head: %v", data["running_head"])
	}
	if data["house"] != "embedded" {
		t.Errorf("a run given no --house built from the embedded style: %v", data["house"])
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("no file was written: %v", err)
	}
	if bytes, _ := data["bytes"].(float64); int64(bytes) != info.Size() {
		t.Errorf("bytes is the size of the file: %v, want %d", data["bytes"], info.Size())
	}

	counts, ok := data["body"].(map[string]any)
	if !ok {
		t.Fatalf("no body counts: %v", data)
	}
	for field, want := range map[string]float64{
		"headings": 3, "lists": 1, "tables": 1, "images": 0, "paragraphs": 5,
	} {
		if counts[field] != want {
			t.Errorf("body.%s is %v, want %v", field, counts[field], want)
		}
	}

	// The twelve XML parts, the logo, and nothing else: this note carries no
	// picture of its own.
	want := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"word/_rels/document.xml.rels",
		"word/_rels/header2.xml.rels",
		"word/document.xml",
		"word/styles.xml",
		"word/numbering.xml",
		"word/settings.xml",
		"word/header1.xml",
		"word/header2.xml",
		"word/footer1.xml",
		"word/footer2.xml",
		"word/media/logo.png",
	}
	names := zipNames(t, out)
	if strings.Join(names, "\n") != strings.Join(want, "\n") {
		t.Errorf("the parts are:\n%s\nwant:\n%s", strings.Join(names, "\n"), strings.Join(want, "\n"))
	}
}

func TestBuildRefusesAMissingFlagByName(t *testing.T) {
	noSession(t)
	_, md := buildNote(t)
	out := filepath.Join(t.TempDir(), "out.docx")

	for _, tc := range []struct {
		name  string
		args  []string
		names string
	}{
		{"no --md", []string{"build", "--out", out}, "--md"},
		{"no --out", []string{"build", "--md", md}, "--out"},
		{"neither", []string{"build"}, "--md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, code := runJSON(t, tc.args...)
			if code == 0 || got["ok"] != false {
				t.Fatalf("a missing flag must be refused: %v (exit %d)", got, code)
			}
			if msg, _ := got["error"].(string); !strings.Contains(msg, tc.names) {
				t.Errorf("the error must name %s: %q", tc.names, msg)
			}
		})
	}
}

func TestBuildRefusesAMissingNoteNamingThePath(t *testing.T) {
	noSession(t)
	dir := t.TempDir()
	md := filepath.Join(dir, "nowhere.md")

	got, code := runJSON(t, "build", "--md", md, "--out", filepath.Join(dir, "out.docx"))
	if code == 0 || got["ok"] != false {
		t.Fatalf("a note that is not there must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, md) {
		t.Errorf("the error must name the path: %q", msg)
	}
}

// Not knowing must never resolve to overwrite. The file gdoc did not write is
// somebody's, and --force is how they say so.
func TestBuildRefusesAnExistingOutUnlessForced(t *testing.T) {
	noSession(t)
	dir, md := buildNote(t)
	out := filepath.Join(dir, "already.docx")
	if err := os.WriteFile(out, []byte("not a docx"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "build", "--md", md, "--out", out)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an existing file must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, out) || !strings.Contains(msg, "--force") {
		t.Errorf("the error must name the file and the way to say yes: %q", msg)
	}
	if kept, _ := os.ReadFile(out); string(kept) != "not a docx" {
		t.Fatalf("the refused run must leave the file alone: %q", kept)
	}

	got, code = runJSON(t, "build", "--md", md, "--out", out, "--force")
	if code != 0 || got["ok"] != true {
		t.Fatalf("--force replaces it: %v (exit %d)", got, code)
	}
	if names := zipNames(t, out); len(names) != 13 {
		t.Errorf("the file was replaced by the built document: %v", names)
	}
}

// TestBuildRefusesAnOutThatNamesAnInput. --force is consent to replace the
// document being written, never consent to replace what the run reads. Without
// this, `--out note.md --force` read the note, rendered it and then put the
// .docx where the note had been: the source was gone the moment the document
// was built.
func TestBuildRefusesAnOutThatNamesAnInput(t *testing.T) {
	noSession(t)
	dir, md := buildNote(t)
	before, err := os.ReadFile(md)
	if err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "build", "--md", md, "--out", md, "--force")
	if code == 0 || got["ok"] != false {
		t.Fatalf("an --out that names the note must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "--md") {
		t.Errorf("the error must name the flag it collides with: %q", msg)
	}
	after, err := os.ReadFile(md)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("the note was replaced by the document built from it")
	}

	// The same rule over the style file, and over a second spelling of one
	// path: "./note.md" and "note.md" are one file.
	style := filepath.Join(dir, "draft-house.yaml")
	src, err := os.ReadFile(filepath.Join("..", "..", "internal", "house", "house.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(style, src, 0o644); err != nil {
		t.Fatal(err)
	}
	got, code = runJSON(t, "build", "--md", md, "--house", style,
		"--out", filepath.Join(dir, ".", filepath.Base(style)), "--force")
	if code == 0 || got["ok"] != false {
		t.Fatalf("an --out that names the style file must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "--house") {
		t.Errorf("the error must name the flag it collides with: %q", msg)
	}
}

// TestBuildRefusesAnOutThatNamesAPicture is the same rule over an input the
// flags do not name. The pictures a note holds are only known once the walk has
// read them, so a check made before the walk left `--out diagram.png --force`
// embedding the picture and then writing the document over it: the note's own
// reference pointed at a .docx from then on.
func TestBuildRefusesAnOutThatNamesAPicture(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "a picture the walk embedded",
			body: "![badge](badge.png)\n",
		},
		{
			// The house style puts a figure on a centred line of its own, so a
			// picture in a bullet is warned about and left out. The note still
			// names the file, and the run must still not write over it: this
			// case is the worse of the two, because the bytes are not in
			// word/media/ either and the picture is simply gone.
			name: "a picture the walk only warned about",
			body: "- see ![badge](badge.png)\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			noSession(t)
			dir := t.TempDir()
			picture := filepath.Join(dir, "badge.png")
			png, err := os.ReadFile(filepath.Join("..", "..", "internal", "body", "testdata", "docs", "badge.png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(picture, png, 0o644); err != nil {
				t.Fatal(err)
			}
			md := filepath.Join(dir, "note.md")
			note := "---\ntitle: A Policy\n---\n\n# Purpose\n\n" + c.body
			if err := os.WriteFile(md, []byte(note), 0o644); err != nil {
				t.Fatal(err)
			}

			got, code := runJSON(t, "build", "--md", md, "--out", picture, "--force")
			if code == 0 || got["ok"] != false {
				t.Fatalf("an --out that names a picture the note holds must be refused: %v (exit %d)", got, code)
			}
			if msg, _ := got["error"].(string); !strings.Contains(msg, "picture") {
				t.Errorf("the error must say what it collides with: %q", msg)
			}
			after, err := os.ReadFile(picture)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(png, after) {
				t.Fatal("the picture was replaced by the document built from the note that names it")
			}
		})
	}
}

func TestBuildReportsTheHouseFileItWasGiven(t *testing.T) {
	noSession(t)
	dir, md := buildNote(t)
	// A copy of the embedded style, which is what somebody reviewing a change
	// to it hands in.
	src, err := os.ReadFile(filepath.Join("..", "..", "internal", "house", "house.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	style := filepath.Join(dir, "draft-house.yaml")
	if err := os.WriteFile(style, src, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.docx")

	got, code := runJSON(t, "build", "--md", md, "--out", out, "--house", style)
	if code != 0 || got["ok"] != true {
		t.Fatalf("build with --house: %v (exit %d)", got, code)
	}
	data, _ := got["data"].(map[string]any)
	if data["house"] != style {
		t.Errorf("a document built from a draft style says so: %v, want %s", data["house"], style)
	}
}

func TestBuildRefusesAHouseFileItCannotRead(t *testing.T) {
	noSession(t)
	dir, md := buildNote(t)
	style := filepath.Join(dir, "broken.yaml")
	if err := os.WriteFile(style, []byte("not_a_key: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "build", "--md", md, "--out", filepath.Join(dir, "out.docx"), "--house", style)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a house style that does not parse must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, style) {
		t.Errorf("the error must name the file it refused: %q", msg)
	}
}

// The binary never invents a title. It names the candidate and the skill
// proposes it, which is the same rule as everywhere else: facts here, the
// judgement in the skill.
func TestBuildNamesTheTitleCandidateWhenTheNoteHasNone(t *testing.T) {
	noSession(t)
	md := writeNote(t, "2026-09-08-supplier-register.md", "Body with no heading at all.\n")

	got, code := runJSON(t, "build", "--md", md, "--out", filepath.Join(filepath.Dir(md), "out.docx"))
	if code == 0 || got["ok"] != false {
		t.Fatalf("a note with no title must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "Supplier register") {
		t.Errorf("the error must carry the candidate from the file name: %q", msg)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(md), "out.docx")); !os.IsNotExist(err) {
		t.Errorf("a refused run writes no file: %v", err)
	}
}

func TestBuildCarriesTheBodyWarnings(t *testing.T) {
	noSession(t)
	md := writeNote(t, "note.md", "---\ntitle: A Note\n---\n\nWords.\n\n```go\nfmt.Println()\n```\n")
	out := filepath.Join(filepath.Dir(md), "out.docx")

	got, code := runJSON(t, "build", "--md", md, "--out", out)
	if code != 0 || got["ok"] != true {
		t.Fatalf("a code block is a warning, not a failure: %v (exit %d)", got, code)
	}
	warnings, _ := got["warnings"].([]any)
	if len(warnings) != 1 {
		t.Fatalf("one warning, naming the line: %v", got["warnings"])
	}
	if w, _ := warnings[0].(string); !strings.Contains(w, "line 7") || !strings.Contains(w, "code block") {
		t.Errorf("the warning must name the line and what was left out: %q", w)
	}
}
