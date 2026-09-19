// M6's two live tests: a publish, and the drift gate that means something.
//
// Both are write tests, so both need GDOC_LIVE_TEST=1 and GDOC_LIVE_WRITE=1,
// and both create only documents of their own in the Drive test folder and
// trash them on the way out. The policy has one door, AllowCreateIn on the
// folder, so every document either test touches is one the guard carried the
// create for.
//
// The drift gate is the reason this file exists at all. The offline gate in
// `make test` reads two docx files and says the generator wrote what house.yaml
// states. What a person opens is not a docx: it is what Google made of one, so
// the measurement that answers the 2026-08-29 decision is the same item list
// read out of two Docs API answers, after both files have been through Drive's
// import. It could not run before M6, because there was no upload to run it
// after.

package live

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/body"
	"gdoc/internal/cover"
	"gdoc/internal/drift"
	"gdoc/internal/drive"
	"gdoc/internal/frontmatter"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/house"
	"gdoc/internal/publish"
	"gdoc/internal/render"
)

// The note the publish test publishes. It is written into a temp directory
// rather than kept as a fixture, because the run writes the gdoc: block into it
// and a fixture that grew a pairing would be a fixture pointing at a document
// that was trashed a second later.
const publishNoteSource = `---
title: Live Publish Test
doc_type: Note
version: 1.0
date: 2026-09-08
owner: Chief Risk Officer
classification: internal
---

# Live publish test

This note exists to be published by ` + "`gdoc publish`" + ` against real Drive and
read back again. The document it makes is trashed a moment later.

## What it checks

- The upload converts, so the file comes back as a document.
- The document has one tab.
- The title Drive gave it is the title that went on the cover.
`

// policyNote is the note the drift gate builds, and it is the body package's
// own copy rather than a second one here. One note, so the goldens, the offline
// gate and this gate all measure the same words.
const policyNote = "../body/testdata/docs/03-policy.md"

// masterDocx is the Word master, read from the drift package's testdata for the
// same reason: one master, so the offline gate and this one measure against the
// same provenance rather than against two copies that can drift apart.
const masterDocx = "../drift/testdata/master.docx"

// TestLivePublish is the whole publish path against Google: the multipart
// upload with conversion, the three read-backs, and the pairing written into
// the note.
//
// The note I/O is spelled here rather than imported, because cmd/gdoc is
// package main. What that costs is one copy of frontmatter.Write's arguments,
// and what it buys is the assertion that matters: the block publish writes is a
// block frontmatter.Read reads back, against a real document id.
func TestLivePublish(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	dir := t.TempDir()
	note := filepath.Join(dir, "live-publish.md")
	if err := os.WriteFile(note, []byte(publishNoteSource), 0o600); err != nil {
		t.Fatalf("the note could not be written: %v", err)
	}
	doc := renderNoteFile(t, note)
	t.Logf("rendered %d bytes titled %q from the %s house style", len(doc.docx), doc.title, doc.house)

	rep, err := publish.Run(ctx, s, publish.Options{
		FolderID: folder,
		Title:    doc.title,
		Docx:     doc.docx,
	})
	// Registered before the error is read: a create Drive accepted and this run
	// failed after is still a document in somebody's folder.
	if rep.DocumentID != "" {
		id := rep.DocumentID
		t.Cleanup(func() { trashLive(t, ctx, s, id, "the published document") })
	}
	for _, w := range rep.Warnings {
		t.Logf("publish warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the publish failed: %v", err)
	}
	if rep.DocumentID == "" {
		t.Fatal("the publish answered with no document id, so nothing below has a document to ask about")
	}
	t.Logf("published %s at %s", rep.DocumentID, rep.URL)

	if !rep.Checks.ReadBack {
		t.Error("the new document could not be read back through the Docs API")
	}
	if !rep.Checks.OneTab {
		t.Errorf("the new document came back with %d tabs, and a published note is one document with one tab", rep.Tabs)
	}
	if !rep.Checks.DocxExport {
		t.Error("the new document did not export back as a docx, so Drive does not hand back the format it went in as")
	}
	if !rep.Verified {
		t.Errorf("the publish is not verified: checks %+v", rep.Checks)
	}
	if rep.Title != doc.title {
		t.Errorf("the document came back titled %q, and the title that went on the cover is %q", rep.Title, doc.title)
	}
	if rep.Tabs != 1 {
		t.Errorf("the document came back with %d tabs, and one docx converts to one tab", rep.Tabs)
	}

	// The pairing, written the way cmd/gdoc writes it and read back the way
	// every other command reads it.
	at := time.Now().UTC().Truncate(time.Second)
	out, err := frontmatter.Write([]byte(publishNoteSource), &frontmatter.Block{
		Schema:     frontmatter.Schema,
		DocumentID: rep.DocumentID,
		FolderID:   folder,
		Published: &frontmatter.Published{
			At:    at,
			Title: doc.title,
			House: doc.house,
		},
	})
	if err != nil {
		t.Fatalf("the gdoc: block could not be written into the note: %v", err)
	}
	if err := os.WriteFile(note, out, 0o600); err != nil {
		t.Fatalf("the paired note could not be written: %v", err)
	}
	paired, err := os.ReadFile(note)
	if err != nil {
		t.Fatalf("the paired note could not be read again: %v", err)
	}
	block, err := frontmatter.Read(paired)
	if err != nil {
		t.Fatalf("the block publish wrote does not read back: %v", err)
	}
	if block == nil {
		t.Fatal("the note carries no gdoc: block after the pairing was written")
	}
	if block.DocumentID != rep.DocumentID {
		t.Errorf("the note names document %q, and the publish made %q", block.DocumentID, rep.DocumentID)
	}
	if block.FolderID != folder {
		t.Errorf("the note names folder %q, and the document was created in %q", block.FolderID, folder)
	}
	if block.Published == nil {
		t.Fatal("the note carries no publish record, so nothing says when the document was made")
	}
	if block.Published.Title != doc.title {
		t.Errorf("the publish record names title %q, and %q went on the cover", block.Published.Title, doc.title)
	}
	if block.Published.House != doc.house {
		t.Errorf("the publish record names house %q, and the run used %q", block.Published.House, doc.house)
	}
	if !block.Published.At.Equal(at) {
		t.Errorf("the publish record is dated %s, and the run wrote %s", block.Published.At, at)
	}
	// The author's own keys survive the write, which is the rule the whole
	// byte-preserving write exists for. Stated here because this is the one
	// place a real note goes through it.
	if !bytes.Contains(paired, []byte("title: Live Publish Test")) {
		t.Errorf("the author's own front matter did not survive the pairing:\n%s", paired)
	}
}

// TestLiveDrift is the measurement DECISIONS.md left open on 2026-08-29: build
// the note from the embedded house style, upload it and the Word master with
// conversion, read both back through the Docs API, and run the same item list
// the offline gate runs.
//
// This is the reading that means something, because Google's import is part of
// the result. A row that differs here and not offline is Drive's import, and a
// row that differs in both is the generator.
//
// It fails on any row that disagrees and is not already named in drift.Known,
// which is Unexplained's rule. A row that has to join Known is a decision Nail
// takes with the reason written beside it, never a test somebody loosens.
func TestLiveDrift(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	built := renderNoteFile(t, policyNote)
	master, err := os.ReadFile(masterDocx)
	if err != nil {
		t.Fatalf("the Word master could not be read: %v", err)
	}
	stamp := time.Now().UTC().Format(time.RFC3339)

	builtID := uploadForDrift(t, ctx, s, folder, "gdoc live drift built "+stamp, built.docx)
	masterID := uploadForDrift(t, ctx, s, folder, "gdoc live drift master "+stamp, master)

	builtDoc := readForDrift(t, ctx, s, builtID, "the built document")
	masterDoc := readForDrift(t, ctx, s, masterID, "the master")

	values := drift.FromDoc(builtDoc)
	rows := drift.Compare(values, drift.FromDoc(masterDoc))
	if len(rows) != len(drift.Items) {
		t.Errorf("the comparison produced %d rows and the list carries %d items", len(rows), len(drift.Items))
	}
	t.Log(drift.Summary(rows))
	t.Logf("the live table, built vs master:\n%s", drift.Table(rows))

	if unexplained := drift.Unexplained(rows); len(unexplained) > 0 {
		t.Errorf("%d rows moved after Google's import and are not in drift.Known:\n%s",
			len(unexplained), drift.Table(unexplained))
	}

	acceptance(t, values, rows)
}

// acceptance is SPEC item 3, asked of the document Drive made: "a publish
// produces a document with the positioned logo, a live contents list and footer
// page numbers, verified by export".
//
// Each of the three is asked twice, and the two questions are different. The
// value says the built document really carries the thing, so a generator that
// stopped emitting it fails here rather than matching a master that lost it
// too. The verdict says the master agrees, so a value that survived the import
// as something else fails as well.
func acceptance(t *testing.T, values []drift.Value, rows []drift.Row) {
	t.Helper()
	read := map[string]any{}
	for _, v := range values {
		read[v.Name] = v.V
	}
	verdicts := map[string]drift.Verdict{}
	for _, r := range rows {
		verdicts[r.Item] = r.Verdict
	}
	held := func(name string) {
		t.Helper()
		v, ok := verdicts[name]
		if !ok {
			t.Errorf("the item list carries no row called %q, so the acceptance cannot be read off it", name)
			return
		}
		if v != drift.Identical && v != drift.Close {
			t.Errorf("acceptance row %q reads %s after Google's import", name, v)
		}
	}

	// The positioned logo. The Docs API cannot create one and a docx import
	// can, which is the whole reason the document is born from a docx.
	if got := read["logo present"]; got != true {
		t.Errorf("the published document carries no logo in its first page header: logo present read %v", got)
	}
	if got := read["logo mode"]; got != "POSITIONED" {
		t.Errorf("the logo came back %v, and a positioned object is what survived the import", got)
	}
	held("logo present")
	held("logo mode")
	held("logo width (pt)")
	held("logo height (pt)")
	held("logo leftOffset (pt)")
	held("logo topOffset (pt)")

	// The live contents list. house.yaml states the field instruction and the
	// writer emits the field, so what has to survive the import is a contents
	// element Docs will refresh rather than a list of frozen words.
	if got, ok := read["live tableOfContents element"].(float64); !ok || got != 1 {
		t.Errorf("the published document carries %v contents elements, and it carries one", read["live tableOfContents element"])
	}
	held("live tableOfContents element")

	// The footer page numbers. An autoText element is the page number field;
	// the docx half and the Docs half both spell it <PAGE>, so the row reads
	// the same on both sides.
	footer, _ := read["default footer text"].(string)
	if !strings.Contains(footer, "<PAGE>") {
		t.Errorf("the published document's footer reads %q, and it carries a page number field", footer)
	}
	held("default footer text")
	held("defaultFooterId present")
}

// noteDocument is one rendered note: the bytes, the title that goes on the
// cover, and where the style came from.
type noteDocument struct {
	docx  []byte
	title string
	house string
}

// renderNoteFile runs the generator over one note. It is cmd/gdoc's render
// step, spelled here because that package is main and cannot be imported, and
// it is the same five calls internal/drift's own gate makes for the same
// reason.
func renderNoteFile(t *testing.T, path string) noteDocument {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the note %s could not be read: %v", path, err)
	}
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the embedded house style did not load: %v", err)
	}
	fields, markdown, err := cover.Read(source)
	if err != nil {
		t.Fatalf("the note's front matter did not read: %v", err)
	}
	walked, err := body.Render(cfg, markdown, filepath.Dir(path), fields.HeadingNumbering)
	if err != nil {
		t.Fatalf("the note's markdown did not render: %v", err)
	}
	for _, w := range walked.Warnings {
		t.Logf("render warning: %s", w)
	}
	pkg, err := render.Build(cfg, fields, walked.Blocks, walked.Media, walked.NumberedLists)
	if err != nil {
		t.Fatalf("the document did not build: %v", err)
	}
	var buf bytes.Buffer
	if err := pkg.Write(&buf); err != nil {
		t.Fatalf("the document did not zip: %v", err)
	}
	return noteDocument{docx: buf.Bytes(), title: fields.CoverTitle(), house: "embedded"}
}

// uploadForDrift puts one docx into the folder with conversion and hands back
// the id, through the production upload rather than a second one written here.
// The document is registered for the trash before anything is asserted about
// it.
func uploadForDrift(t *testing.T, ctx context.Context, s *gapi.Session, folder, title string, docx []byte) string {
	t.Helper()
	rep, err := publish.Run(ctx, s, publish.Options{FolderID: folder, Title: title, Docx: docx})
	if rep.DocumentID != "" {
		id := rep.DocumentID
		t.Cleanup(func() { trashLive(t, ctx, s, id, title) })
	}
	for _, w := range rep.Warnings {
		t.Logf("upload warning on %q: %s", title, w)
	}
	if err != nil {
		t.Fatalf("%q could not be uploaded into folder %q: %v", title, folder, err)
	}
	if rep.DocumentID == "" {
		t.Fatalf("the upload of %q answered with no document id", title)
	}
	t.Logf("uploaded %q as %s", title, rep.DocumentID)
	return rep.DocumentID
}

// readForDrift reads one document the way drift.Doc expects it.
//
// The URL is deliberately not docs.URL. That one asks for includeTabsContent,
// which moves the content into tabs[] and leaves the legacy body, headers and
// footers empty, and those legacy fields are exactly what this half of the item
// list reads: they carry the first tab, which is the whole of a document
// converted from one docx. A read through docs.URL would answer nil for every
// body, header and footer row and the gate would pass on a document it never
// looked inside.
func readForDrift(t *testing.T, ctx context.Context, s *gapi.Session, id, what string) *drift.Doc {
	t.Helper()
	var answer map[string]any
	if err := s.GetJSON(ctx, driftReadURL(id), &answer); err != nil {
		t.Fatalf("%s (%s) could not be read back: %v", what, id, err)
	}
	if len(answer) == 0 {
		t.Fatalf("%s (%s) came back as an empty answer", what, id)
	}
	return drift.OpenDoc(answer)
}

// driftReadURL is documents.get with no query at all: the legacy fields, which
// are the first tab. See readForDrift for why the tabs view is the wrong read
// here.
func driftReadURL(id string) string {
	return "https://docs.googleapis.com/v1/documents/" + id
}

// liveWriteSession is the opening of both tests: the two variables, the folder,
// one policy whose only door is that folder, and a session on it.
func liveWriteSession(t *testing.T) (context.Context, *gapi.Session, string) {
	t.Helper()
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to create documents in the Drive test folder; %s names another folder", liveVar, writeVar, folderVar)
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
	return context.Background(), s, folder
}

// trashLive puts one document away through the production trash, which is the
// PATCH, the read-back and believing the read. It reports rather than fails:
// a document left in the folder is worth naming loudly, and it is not the thing
// either test was asking.
func trashLive(t *testing.T, ctx context.Context, s *gapi.Session, id, what string) {
	t.Helper()
	if err := drive.Trash(ctx, s, id); err != nil {
		t.Errorf("%s (%s) is still in the folder: %v; open %s and delete it by hand",
			what, id, err, publish.DocumentURL(id))
		return
	}
	t.Logf("%s (%s) trashed", what, id)
}

// TestTheLiveFixturesRenderWithNoNetwork is the offline half of this file, and
// it runs in `make test` like everything else.
//
// Both live tests above render a note before they reach Drive, and a note that
// no longer renders, or a fixture path that moved, would otherwise be found by
// Nail in the middle of a live run rather than by the suite. So this renders
// both notes and opens the master, and touches no wire at all: it never asks
// for the two live variables, because there is nothing here to protect.
func TestTheLiveFixturesRenderWithNoNetwork(t *testing.T) {
	for _, c := range []struct{ what, source string }{
		{"the publish test's note", ""},
		{"the drift gate's note", policyNote},
	} {
		path := c.source
		if path == "" {
			path = filepath.Join(t.TempDir(), "live-publish.md")
			if err := os.WriteFile(path, []byte(publishNoteSource), 0o600); err != nil {
				t.Fatalf("%s could not be written: %v", c.what, err)
			}
		}
		doc := renderNoteFile(t, path)
		if len(doc.docx) == 0 {
			t.Errorf("%s rendered no bytes", c.what)
		}
		if doc.title == "" {
			t.Errorf("%s rendered with no title, and the title is the name the document is found by", c.what)
		}
		if err := (publish.Options{FolderID: "FOLDER", Title: doc.title, Docx: doc.docx}).Check(); err != nil {
			t.Errorf("%s does not make a publishable upload: %v", c.what, err)
		}
	}
	if _, err := os.Stat(masterDocx); err != nil {
		t.Errorf("the Word master the live gate measures against is not at %s: %v", masterDocx, err)
	}
}
