// M6's two live tests, a publish and the drift gate that means something, and
// M21's heading id acceptance beside them, because it publishes the same way.
//
// All three are write tests, so each needs GDOC_LIVE_TEST=1 and
// GDOC_LIVE_WRITE=1, and each creates only documents of its own in the Drive
// test folder. The policy has one door, AllowCreateIn on the folder, so every
// document any of them touches is one the guard carried the create for. M6's
// two trash theirs on the way out; the heading id run leaves its document
// behind on purpose, because the refresh of the contents list is a click
// nothing here can make.
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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/body"
	"gdoc/internal/cover"
	"gdoc/internal/docs"
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

// headingIDNoteSource is the note M21's acceptance publishes: three headings,
// and one `#` link naming the second of them. One note, because the question
// is what Drive's import does with the hidden bookmark the docx carries on
// every heading, and three headings with one jump among them hold both halves
// of the answer.
//
// It is written into a temp directory for publishNoteSource's reason, with one
// difference: the document this one makes is left in the folder rather than
// trashed, so the note it came from would point at a document that is still
// there and still has to be read by hand. A fixture pairing with whichever run
// went last is worse than no fixture, so there is none.
const headingIDNoteSource = `---
title: Live Heading Id Test
doc_type: Note
version: 1.0
date: 2026-10-16
owner: Chief Risk Officer
classification: internal
---

# Where a heading id comes from

Drive's import mints a heading id for a heading paragraph from the hidden
bookmark the docx carries on it, and a contents entry is a link to a heading by
that id.

## Who decides

Nail decides, and this heading is the one the jump below names.

## What the reader gets

A reader who clicks an entry in the contents list lands on the section it
names. See [who decides](#who-decides) for the heading this note jumps to.
`

// The note's shape as literals: how many headings it holds, the words the
// heading the jump names shows, and the words the jump itself shows. Literals
// rather than counts read back out of the note, for the reason the page break
// measurement's two phrases are literals: what the run asserts is what a
// person really published.
const (
	headingIDNoteHeadings = 3
	headingIDTarget       = "Who decides"
	headingIDJump         = "who decides"
)

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
	// The firm's own mark, fetched the way a real publish fetches it, so the
	// document that goes into Drive is the document production puts there.
	// TestPublishCarriesTheHouseLogo is the measurement; this is the publish
	// that rests on it.
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the embedded house style did not load: %v", err)
	}
	doc := renderNoteFileWithLogo(t, note, liveHouseLogo(t, cfg))
	carriesTheHouseLogo(t, doc.docx)
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
	// Drive's own conversion is what puts the contents list where publish's
	// cover ends, so only a live run proves the marker lands on it.
	if !rep.Marked {
		t.Errorf("the cover carries no marker, so a restyle would flatten it: %v", rep.Warnings)
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
		Schema: frontmatter.Schema,
		Documents: []frontmatter.Entry{{
			ID:       rep.DocumentID,
			FolderID: folder,
			Published: &frontmatter.Published{
				At:    at,
				Title: doc.title,
				House: doc.house,
			},
		}},
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
	if len(block.Documents) != 1 {
		t.Fatalf("the note names %d documents, and this publish made one", len(block.Documents))
	}
	entry := block.Documents[0]
	if entry.ID != rep.DocumentID {
		t.Errorf("the note names document %q, and the publish made %q", entry.ID, rep.DocumentID)
	}
	if entry.FolderID != folder {
		t.Errorf("the note names folder %q, and the document was created in %q", entry.FolderID, folder)
	}
	if entry.Published == nil {
		t.Fatal("the note carries no publish record, so nothing says when the document was made")
	}
	if entry.Published.Title != doc.title {
		t.Errorf("the publish record names title %q, and %q went on the cover", entry.Published.Title, doc.title)
	}
	if entry.Published.House != doc.house {
		t.Errorf("the publish record names house %q, and the run used %q", entry.Published.House, doc.house)
	}
	if !entry.Published.At.Equal(at) {
		t.Errorf("the publish record is dated %s, and the run wrote %s", entry.Published.At, at)
	}
	// The author's own keys survive the write, which is the rule the whole
	// byte-preserving write exists for. Stated here because this is the one
	// place a real note goes through it.
	if !bytes.Contains(paired, []byte("title: Live Publish Test")) {
		t.Errorf("the author's own front matter did not survive the pairing:\n%s", paired)
	}
}

// TestAPublishedHeadingCarriesAnIdAndAJumpIsAHeadingLink is M21's acceptance,
// and it is the question no fixture can answer: what Drive's import does with
// the hidden bookmark internal/body writes on every heading.
//
// The offline tests of internal/body say the docx carries one `_h_<slug>` pair
// before each heading's runs and that a `#` link anchors at that name. What no
// test on this machine can say is whether Google still mints a heading id from
// it, and a heading with no id is a contents entry that opens the first page
// instead of the section. That failure is silent in every offline reading, and
// it was silent in every document gdoc published before 2026-10-16.
//
// So this publishes the note above through the production render and the
// production upload, reads the document back through the read every command
// makes, and asks three things: every heading came back with an id, the jump
// came back as a heading link naming the right heading's id, and the publish
// report counted no heading without one.
//
// The document is left in the folder rather than trashed, because the one step
// no API can take is the refresh of the contents list. The Docs API has no
// request that builds or refreshes one, so an entry that opens its heading is
// a click in the UI and Nail's eyes on the hover card. The URL is in the log.
func TestAPublishedHeadingCarriesAnIdAndAJumpIsAHeadingLink(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	dir := t.TempDir()
	note := filepath.Join(dir, "live-heading-id.md")
	if err := os.WriteFile(note, []byte(headingIDNoteSource), 0o600); err != nil {
		t.Fatalf("the note could not be written: %v", err)
	}
	built := renderNoteFile(t, note)
	title := "gdoc M21 heading id " + time.Now().UTC().Format(time.RFC3339)

	rep, err := publish.Run(ctx, s, publish.Options{FolderID: folder, Title: title, Docx: built.docx})
	// Registered before the error is read, for the reason TestLivePublish
	// registers its trash there: a create Drive accepted and this run failed
	// after is still a document somebody has to find.
	if rep.DocumentID != "" {
		id := rep.DocumentID
		t.Cleanup(func() {
			t.Logf("the document is left for the refresh a person has to click: %s", publish.DocumentURL(id))
			t.Logf("refresh the contents list, check an entry names its heading, then trash it in folder %s", folder)
		})
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

	// The read-back's own count first, because that is the fact the skill acts
	// on and the one that would have caught this a month ago.
	if rep.HeadingsWithoutID != 0 {
		t.Errorf("the publish reports headings_without_id: %d, and Drive's import mints an id for every heading that carries the hidden bookmark", rep.HeadingsWithoutID)
	}

	// Then the document itself, through the read every command makes, so what
	// is asserted below is what `read` would print.
	var raw json.RawMessage
	if err := s.GetJSON(ctx, docs.URL(rep.DocumentID), &raw); err != nil {
		t.Fatalf("the published document (%s) could not be read back: %v", rep.DocumentID, err)
	}
	d, err := docs.Parse(raw)
	if err != nil {
		t.Fatalf("the read of %s did not parse: %v", rep.DocumentID, err)
	}
	t.Logf("read %s back as %d tabs titled %q", rep.DocumentID, len(d.Tabs), d.Title)

	headings, jumps := headingsAndJumps(d)
	if len(headings) != headingIDNoteHeadings {
		t.Fatalf("the document came back with %d headings and the note holds %d, so neither list below is about the note", len(headings), headingIDNoteHeadings)
	}
	var target string
	for _, h := range headings {
		t.Logf("heading %q carries heading id %q", h.text, h.id)
		if h.id == "" {
			t.Errorf("the heading %q came back with no heading id, so a contents entry naming it opens the first page instead", h.text)
		}
		if strings.Contains(h.text, headingIDTarget) {
			target = h.id
		}
	}
	if target == "" {
		t.Fatalf("no heading of the document shows %q with an id, so there is nothing the jump could name", headingIDTarget)
	}

	// The jump. One run, because the note holds one `#` link, and a document
	// that came back with more than one would be reporting the contents list
	// rather than the note.
	if len(jumps) != 1 {
		t.Fatalf("the document came back with %d links to a heading, and the note holds one: %+v", len(jumps), jumps)
	}
	jump := jumps[0]
	t.Logf("the jump %q points at heading id %q", jump.text, jump.headingID)
	if !strings.Contains(jump.text, headingIDJump) {
		t.Errorf("the one heading link shows %q, and the note's jump shows %q", jump.text, headingIDJump)
	}
	if jump.headingID != target {
		t.Errorf("the jump points at heading id %q and %q is the heading it names: a jump anchored at the hidden bookmark comes back as a link to that heading", jump.headingID, target)
	}
}

// liveHeading is one heading paragraph of a published document: the words it
// shows, and the id a link to it names.
type liveHeading struct {
	text string
	id   string
}

// liveJump is one run pointing at a heading of the same document: the words it
// shows, and the heading id it names.
type liveJump struct {
	text      string
	headingID string
}

// headingsAndJumps reads both lists off one document, through the tables of
// every tab and never into the contents list.
//
// The contents list is skipped for countHeadings' reason: its entries are the
// links to the headings and never headings themselves, so a walk that went in
// would count Docs' own list twice over and find a heading link per entry.
func headingsAndJumps(d *docs.Document) ([]liveHeading, []liveJump) {
	var headings []liveHeading
	var jumps []liveJump
	var blocks func([]docs.Block)
	blocks = func(bs []docs.Block) {
		for _, b := range bs {
			switch {
			case b.Paragraph != nil:
				var text string
				for _, r := range b.Paragraph.Runs {
					text += r.Text
					if r.Link != nil && r.Link.HeadingID != "" {
						jumps = append(jumps, liveJump{text: strings.TrimSpace(r.Text), headingID: r.Link.HeadingID})
					}
				}
				if strings.HasPrefix(b.Paragraph.Style, "HEADING_") {
					headings = append(headings, liveHeading{text: strings.TrimSpace(text), id: b.Paragraph.HeadingID})
				}
			case b.Table != nil:
				for _, row := range b.Table.Rows {
					for _, cell := range row {
						blocks(cell.Blocks)
					}
				}
			}
		}
	}
	for _, tab := range d.Tabs {
		blocks(tab.Body)
	}
	return headings, jumps
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

// renderNoteFile runs the generator over one note with the placeholder mark,
// which is what every test here but one wants: a document whose logo is the
// right shape, since what they measure about it is geometry.
//
// TestPublishCarriesTheHouseLogo is the one, and it hands over the bytes it
// read out of the firm's Drive.
func renderNoteFile(t *testing.T, path string) noteDocument {
	t.Helper()
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the embedded house style did not load: %v", err)
	}
	return renderNoteFileWithLogo(t, path, houseLogo(t, cfg))
}

// renderNoteFileWithLogo is cmd/gdoc's render step, spelled here because that
// package is main and cannot be imported, and it is the same five calls
// internal/drift's own gate makes for the same reason.
//
// The mark is a parameter for render.Build's own reason: a house style may
// name its logo as one file in Drive, and only internal/asset can turn that
// into bytes.
func renderNoteFileWithLogo(t *testing.T, path string, logo []byte) noteDocument {
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
	pkg, err := render.Build(cfg, logo, fields, walked.Blocks, walked.Media, walked.Footnotes, walked.NumberedLists)
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
// Three live tests above render a note before they reach Drive, and a note that
// no longer renders, or a fixture path that moved, would otherwise be found by
// Nail in the middle of a live run rather than by the suite. So this renders
// all three notes and opens the master, and touches no wire at all: it never
// asks for the two live variables, because there is nothing here to protect.
//
// The heading id note is held to its shape as well, because that run asserts
// three headings and one jump: a note somebody edited down to two headings, or
// one whose `#` link lost its target, would leave the live run failing on the
// note rather than on Drive.
func TestTheLiveFixturesRenderWithNoNetwork(t *testing.T) {
	for _, c := range []struct{ what, source, inline string }{
		{what: "the publish test's note", inline: publishNoteSource},
		{what: "the heading id test's note", inline: headingIDNoteSource},
		{what: "the drift gate's note", source: policyNote},
	} {
		path := c.source
		if path == "" {
			path = filepath.Join(t.TempDir(), "live-note.md")
			if err := os.WriteFile(path, []byte(c.inline), 0o600); err != nil {
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
	if n := strings.Count(headingIDNoteSource, "\n# ") + strings.Count(headingIDNoteSource, "\n## "); n != headingIDNoteHeadings {
		t.Errorf("the heading id note holds %d headings and its run asserts %d", n, headingIDNoteHeadings)
	}
	if !strings.Contains(headingIDNoteSource, "["+headingIDJump+"](#who-decides)") {
		t.Errorf("the heading id note holds no `#` link showing %q, so its run has no jump to measure", headingIDJump)
	}
	if !strings.Contains(headingIDNoteSource, "## "+headingIDTarget) {
		t.Errorf("the heading id note holds no heading reading %q, so its jump names a heading that is not there", headingIDTarget)
	}
}

// placeholderLogo is the PNG these renders are handed when the house names its
// logo in Drive rather than carrying it. The firm's logo is one file in its
// Drive and the tree holds no copy, and nothing in this file measures the
// picture: every logo assertion above reads geometry. The run that reads the
// real bytes out of Drive is TestPublishCarriesTheHouseLogo.
const placeholderLogo = "../house/testdata/placeholder-logo.png"

// houseLogo is the mark render is handed. Build takes the logo from its caller
// now rather than reading it off the config, because a house style may name
// one file in Drive instead of carrying the bytes.
func houseLogo(t *testing.T, cfg *house.Config) []byte {
	t.Helper()
	if cfg.LogoDrive() != "" {
		raw, err := os.ReadFile(placeholderLogo)
		if err != nil {
			t.Fatalf("read the placeholder logo: %v", err)
		}
		return raw
	}
	raw, err := cfg.LogoBytes()
	if err != nil {
		t.Fatalf("LogoBytes: %v", err)
	}
	return raw
}
