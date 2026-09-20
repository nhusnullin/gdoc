package live

// M13's live export tests: the round trip over the six body notes, a proposal
// carried out into a file, the picture bytes, and a note published twice.
//
// Every test here is a write test. Each creates its own documents in the Drive
// test folder from notes on this disk, and trashes every one of them on the way
// out, so no document of Nail's is in the reachable set at all.
//
// The round trip is the one that answers a question no fixture can. A note is
// rendered into a docx, Drive imports it, the Docs API reads what Drive made,
// and the export projects that back into Markdown. Four translations, three of
// them Google's, and what the fixtures pin is the fourth. So this publishes the
// six notes internal/body renders and compares each file that comes back with
// the note it started as.
//
// The comparison cannot be equality, because a docx carries no fence, no
// emphasis character and no link to a file in the hub. It is the drift gate's
// shape instead: a named list of differences, each with the reason it is there,
// and a run fails on a difference no name covers. Every name that fired is
// logged with its count, so a difference is never quietly excused. A name that
// has to join the list is a decision Nail writes down with its reason, never a
// test somebody loosens.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/docx"
	"gdoc/internal/export"
	"gdoc/internal/frontmatter"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/propose"
	"gdoc/internal/publish"
	"gdoc/internal/withdraw"
)

// bodyNotes is the folder the six round trip notes live in. They are
// internal/body's own fixtures rather than six copies here, for the reason the
// drift gate keeps one note: one set of words, so the goldens, the offline gate
// and this run all measure the same document.
const bodyNotes = "../body/testdata/docs"

// roundTripNotes is every note internal/body renders, in the order they are
// numbered. The pictures note is named again below, because the hash test
// publishes that one alone.
var roundTripNotes = []string{
	"01-kitchen-sink.md",
	"02-pictures.md",
	"03-policy.md",
	"04-minimal.md",
	"05-edge-cases.md",
	"06-long.md",
}

// picturesNote is the note whose pictures the hash test measures, and
// policyNoteForProposal is the one the proposal test publishes.
const (
	picturesNote          = "02-pictures.md"
	policyNoteForProposal = "03-policy.md"
)

// The words the proposal test replaces, and what it replaces them with. They
// occur exactly once in the policy note, which is what FindSpan asks for.
const (
	exportQuoted = "selects, approves, monitors and exits"
	exportNew    = "selects, approves, monitors, reviews and exits"
	exportWhy    = "The sections below describe a review step, and this sentence does not list it."
)

// commandSource is cmd/gdoc's export command, read by the hash test to ask
// whether the binary has turned the hash match on yet. Reading the source is
// how internal/export's own TestNothingCanReplaceAFile asks a question about
// what a package does rather than about what it answers, and the question here
// is the same shape: the measurement is a measurement until the binary makes it
// a rule, and this is the one place that can tell.
const commandSource = "../../cmd/gdoc/export.go"

// TestLiveExportRoundTrip publishes each of the six body notes, exports the
// document Drive made of it, and compares the file with the note.
//
// It creates six documents and trashes all six. Each is a document publish
// made, so the policy has one door open, AllowCreateIn on the folder, and the
// two reads the export makes are reads of a document the guard carried the
// create for.
func TestLiveExportRoundTrip(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)
	stamp := time.Now().UTC().Format(time.RFC3339)

	for _, name := range roundTripNotes {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(bodyNotes, name)
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("the note %s could not be read: %v", path, err)
			}
			built := renderNoteFile(t, path)
			id := uploadForDrift(t, ctx, s, folder, "gdoc live export "+name+" "+stamp, built.docx)

			out := filepath.Join(t.TempDir(), name)
			run := exportDocument(t, ctx, s, id, out)
			if len(run.files) != 1 {
				t.Fatalf("the export of %s wrote %d files, and one docx converts to one tab", name, len(run.files))
			}
			if len(run.pieces) == 0 {
				t.Errorf("the export of %s stripped no prelude, and a published document carries one", name)
			}
			t.Logf("%s: %d pieces stripped, %d pictures, %d warnings",
				name, len(run.pieces), len(run.written.Pictures), len(run.warnings))

			report := roundTrip(src, run.files[0].Body)
			for _, n := range report.named {
				t.Logf("%s: %d lines differ because of %q: %s", name, n.count, n.name, n.reason)
			}
			for _, u := range report.unnamed {
				t.Errorf("%s: a difference no name covers, %s: %q", name, u.side, u.text)
			}
			if len(report.unnamed) > 0 {
				t.Errorf("%s: %d differences are unnamed; each is either a fault or a row for the list in this file, with its reason",
					name, len(report.unnamed))
			}
		})
	}
}

// TestLiveExportCarriesAProposal is scenario 12 against Google: a document with
// one of gdoc's own suggestions pending in it exports with that suggestion's id
// in the file, so the session merging the file can see what is proposed and
// what is not.
//
// It publishes the policy note, proposes into the document, exports, withdraws
// the proposal and trashes the document.
func TestLiveExportCarriesAProposal(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	path := filepath.Join(bodyNotes, policyNoteForProposal)
	built := renderNoteFile(t, path)
	id := uploadForDrift(t, ctx, s, folder, "gdoc live export proposal "+time.Now().UTC().Format(time.RFC3339), built.docx)

	result, err := propose.Apply(ctx, s, id, propose.Proposal{
		Quoted:      exportQuoted,
		Replacement: exportNew,
		Why:         exportWhy,
	})
	for _, w := range result.Warnings {
		t.Logf("propose warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the proposal into %s could not be written: %v", id, err)
	}
	if len(result.SuggestionIDs) == 0 {
		t.Fatalf("the proposal landed with no suggestion id, so there is nothing for the export to carry")
	}
	if !result.Verified {
		t.Errorf("the proposal is not verified: state %q, checks %+v", result.CommentUpdateState, result.Checks)
	}
	t.Logf("proposed: suggestion ids %v, comment %s", result.SuggestionIDs, result.CommentID)

	out := filepath.Join(t.TempDir(), policyNoteForProposal)
	run := exportDocument(t, ctx, s, id, out)
	if len(run.files) != 1 {
		t.Fatalf("the export wrote %d files, and one docx converts to one tab", len(run.files))
	}
	body := run.files[0].Body
	for _, id := range result.SuggestionIDs {
		if !strings.Contains(body, "[s:"+id+"]") {
			t.Errorf("the file does not carry suggestion %q, so nothing in it says those words are proposed", id)
		}
	}
	if !strings.Contains(body, exportNew) {
		t.Errorf("the file does not carry the proposed words %q", exportNew)
	}
	if result.CommentID != "" && !strings.Contains(body, "[[c:"+result.CommentID+"]]") {
		t.Errorf("the file does not carry the comment %q the proposal made", result.CommentID)
	}

	withdrawProposal(t, ctx, s, id, result)
}

// TestLiveExportHashEquality is measurement 2 of docs/v2/MEASURED.md: does a
// PNG that went into Drive come back out of the docx export byte for byte?
//
// It publishes the pictures note, exports it, and prints the sha256 of each
// picture beside the sha256 of the file on this disk it was uploaded from. It
// asserts the equality only once cmd/gdoc turns Options.MatchByHash on, which
// is what makes this the measurement first and the pin afterwards: a test that
// asserted the answer today would be deciding what it was sent to find out.
func TestLiveExportHashEquality(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	path := filepath.Join(bodyNotes, picturesNote)
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the pictures note could not be read: %v", err)
	}
	built := renderNoteFile(t, path)
	id := uploadForDrift(t, ctx, s, folder, "gdoc live export hashes "+time.Now().UTC().Format(time.RFC3339), built.docx)

	out := filepath.Join(t.TempDir(), picturesNote)
	run := exportDocument(t, ctx, s, id, out)

	uploaded, warnings := export.NotePictures(src, bodyNotes)
	for _, w := range warnings {
		t.Logf("the note's own pictures: %s", w)
	}
	known := map[string]string{}
	for _, p := range uploaded {
		known[sum(p.Bytes)] = p.Target
	}

	pinned := hashMatchIsOn(t)
	t.Logf("the exported pictures against the %d the note was rendered from; the binary %s",
		len(uploaded), map[bool]string{true: "matches by hash, so this is asserted", false: "does not match by hash yet, so this is printed"}[pinned])
	for i, p := range run.written.Pictures {
		if len(p.Bytes) == 0 {
			t.Logf("picture %d (%s): no bytes, matched %q", i+1, p.Ext, p.Matched)
			continue
		}
		h := sum(p.Bytes)
		target, ok := known[h]
		t.Logf("picture %d (%s, %d bytes) sha256 %s: %s", i+1, p.Ext, len(p.Bytes), h,
			map[bool]string{true: "equal to " + target, false: "equal to no picture the note holds"}[ok])
		if pinned && !ok {
			t.Errorf("picture %d came back with sha256 %s, and the binary keeps the note's own picture when the bytes are the same, so a picture that matches nothing means the match is reading the wrong bytes", i+1, h)
		}
	}
}

// TestLivePublishAgainAppendsAnEntry is scenario 1 against Google: a note that
// is already paired publishes again, into a new document, and the block comes
// back holding both.
//
// It creates two documents from one note and trashes both.
func TestLivePublishAgainAppendsAnEntry(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	dir := t.TempDir()
	note := filepath.Join(dir, "live-publish-again.md")
	if err := os.WriteFile(note, []byte(publishNoteSource), 0o600); err != nil {
		t.Fatalf("the note could not be written: %v", err)
	}
	built := renderNoteFile(t, note)

	first := publishForPairing(t, ctx, s, folder, note, built)
	second := publishForPairing(t, ctx, s, folder, note, built)
	if first == second {
		t.Fatalf("both publishes answered with document %q, and a second publish makes a second document", first)
	}

	paired, err := os.ReadFile(note)
	if err != nil {
		t.Fatalf("the paired note could not be read: %v", err)
	}
	block, err := frontmatter.Read(paired)
	if err != nil {
		t.Fatalf("the block the two publishes wrote does not read back: %v", err)
	}
	if block == nil {
		t.Fatal("the note carries no gdoc: block after two publishes")
	}
	if len(block.Documents) != 2 {
		t.Fatalf("the note names %d documents, and it was published twice:\n%s", len(block.Documents), paired)
	}
	if block.Schema != frontmatter.Schema {
		t.Errorf("the note came back at schema %d, and a block holding two documents is schema %d", block.Schema, frontmatter.Schema)
	}
	for _, id := range []string{first, second} {
		entry, err := block.Entry(id)
		if err != nil {
			t.Errorf("the note does not name document %q: %v", id, err)
			continue
		}
		if entry.Published == nil {
			t.Errorf("the entry for %q carries no publish record", id)
		}
		var d *docs.Document
		if d, err = docs.Fetch(ctx, s, id); err != nil {
			t.Errorf("document %q could not be read back, so nothing says it exists: %v", id, err)
			continue
		}
		t.Logf("document %s reads back as %q with %d tabs", id, d.Title, len(d.Tabs))
	}
}

// publishForPairing publishes the note once and writes the entry into it the
// way cmd/gdoc does: the block is read, the new document is appended, and the
// whole block is written back. It is spelled here because cmd/gdoc is package
// main, and it is the one thing this test is asking about, so it asserts the
// append rather than the render.
func publishForPairing(t *testing.T, ctx context.Context, s *gapi.Session, folder, note string, built noteDocument) string {
	t.Helper()
	rep, err := publish.Run(ctx, s, publish.Options{FolderID: folder, Title: built.title, Docx: built.docx})
	if rep.DocumentID != "" {
		id := rep.DocumentID
		t.Cleanup(func() { trashLive(t, ctx, s, id, "a document of the publish-again test") })
	}
	for _, w := range rep.Warnings {
		t.Logf("publish warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the publish failed: %v", err)
	}
	if !rep.Verified {
		t.Errorf("the publish of %s is not verified: checks %+v", rep.DocumentID, rep.Checks)
	}

	src, err := os.ReadFile(note)
	if err != nil {
		t.Fatalf("the note could not be read before pairing: %v", err)
	}
	block, err := frontmatter.Read(src)
	if err != nil {
		t.Fatalf("the note's block could not be read before pairing: %v", err)
	}
	if block == nil {
		block = &frontmatter.Block{Schema: frontmatter.Schema}
	}
	block.Documents = append(block.Documents, frontmatter.Entry{
		ID:       rep.DocumentID,
		FolderID: folder,
		Published: &frontmatter.Published{
			At:    time.Now().UTC().Truncate(time.Second),
			Title: built.title,
			House: built.house,
		},
	})
	out, err := frontmatter.Write(src, block)
	if err != nil {
		t.Fatalf("the entry for %s could not be written into the note: %v", rep.DocumentID, err)
	}
	if err := os.WriteFile(note, out, 0o600); err != nil {
		t.Fatalf("the paired note could not be written: %v", err)
	}
	t.Logf("published %s at %s", rep.DocumentID, rep.URL)
	return rep.DocumentID
}

// withdrawProposal takes the proposal back, through the provenance the command
// uses: the id is recorded into a note and read back, and the guard is granted
// that one id for that one run.
func withdrawProposal(t *testing.T, ctx context.Context, s *gapi.Session, docID string, result propose.Result) {
	t.Helper()
	note := []byte("---\ngdoc:\n  schema: 2\n  documents:\n    - id: " + docID + "\n---\n\n# Live export test\n")
	recorded, missed, err := propose.Record(note, docID, []propose.Result{result}, time.Now().UTC())
	if err != nil {
		t.Fatalf("the proposal could not be recorded in a note: %v", err)
	}
	if len(missed) != 0 {
		t.Fatalf("the live proposal could not be remembered: %+v", missed)
	}
	block, err := frontmatter.Read(recorded)
	if err != nil {
		t.Fatalf("the note propose wrote could not be read back: %v", err)
	}
	entry, err := block.Entry(docID)
	if err != nil {
		t.Fatalf("the note propose wrote does not name the document: %v", err)
	}

	// A second policy, granted the one id, because the session above is the one
	// the export read through and a grant is for one run and one object.
	p := guard.NewPolicy()
	p.AllowFile(docID, guard.LevelSuggest)
	id := result.SuggestionIDs[0]
	p.AllowReject(id)
	s2, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the withdrawing session could not be opened: %v", err)
	}
	gone, err := withdraw.Run(ctx, s2, docID, id, entry)
	for _, w := range gone.Warnings {
		t.Logf("withdraw warning: %s", w)
	}
	if err != nil {
		t.Errorf("the suggestion %q could not be withdrawn: %v", id, err)
		return
	}
	if !gone.Verified {
		t.Errorf("the withdrawal of %q is not verified: rejected ids %v", id, gone.RejectedSuggestionIDs)
	}
	t.Logf("withdrawn: %v", gone.RejectedSuggestionIDs)
}

// exported is what one run of the export path produced.
type exported struct {
	doc      *docs.Document
	files    []export.TabFile
	pieces   []export.Piece
	written  *export.Written
	warnings []string
}

// exportDocument is cmd/gdoc's export command without the envelope, in the
// order the work has to happen: the two reads, the pairing, the paths, and only
// then the projection, because a picture's line in the body names the file that
// picture lands in. It is spelled here because cmd/gdoc is package main, and
// every step is the production call.
func exportDocument(t *testing.T, ctx context.Context, s *gapi.Session, id, out string) exported {
	t.Helper()
	d, err := docs.Fetch(ctx, s, id)
	if err != nil {
		t.Fatalf("the Docs read of %s failed: %v", id, err)
	}
	var notes []string

	raw, err := docx.Export(ctx, s, id)
	if err != nil {
		t.Fatalf("the docx export of %s failed: %v", id, err)
	}
	media, err := docx.Media(raw)
	if err != nil {
		t.Fatalf("the docx export of %s could not be walked for its pictures: %v", id, err)
	}

	src, _ := os.ReadFile(out)
	notePics, w := export.NotePictures(src, filepath.Dir(out))
	notes = append(notes, w...)

	var objects []docs.Object
	for _, tab := range d.Tabs {
		objects = append(objects, export.Objects(tab)...)
	}
	pics, w := export.Pictures(objects, media, notePics, export.Options{})
	notes = append(notes, w...)

	layout, err := export.Plan(export.Request{
		Out:        out,
		DocumentID: id,
		At:         time.Now().UTC(),
		Tabs:       exportTabsOf(d),
		Pictures:   pics,
	})
	if err != nil {
		t.Fatalf("the export of %s could not be planned: %v", id, err)
	}
	files, pieces, w := export.Project(d, layout.PictureNames())
	notes = append(notes, w...)

	written, err := export.Write(layout, files)
	if err != nil {
		t.Fatalf("the export of %s could not be written: %v", id, err)
	}
	for _, note := range notes {
		t.Logf("export warning: %s", note)
	}
	return exported{doc: d, files: files, pieces: pieces, written: written, warnings: notes}
}

// exportTabsOf is the document's tabs as the writer needs them, in document
// order.
func exportTabsOf(d *docs.Document) []export.Tab {
	tabs := make([]export.Tab, 0, len(d.Tabs))
	for _, t := range d.Tabs {
		tabs = append(tabs, export.Tab{ID: t.ID, Title: t.Title})
	}
	return tabs
}

// hashMatchIsOn says whether cmd/gdoc has turned Options.MatchByHash on. The
// question is asked of the source rather than of a constant here, because a
// constant here would be a second answer that could disagree with the binary.
func hashMatchIsOn(t *testing.T) bool {
	t.Helper()
	b, err := os.ReadFile(commandSource)
	if err != nil {
		t.Fatalf("the export command's source could not be read at %s, so the test cannot say whether the hash match is on: %v", commandSource, err)
	}
	return strings.Contains(string(b), "MatchByHash")
}

// sum is the sha256 of some bytes as hex, which is what a picture is named by
// in this file's log.
func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
