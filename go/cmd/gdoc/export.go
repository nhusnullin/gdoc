// The command that brings a document out of Drive and into the hub.
//
// `gdoc export` makes two reads and no writes: the Docs read `read` makes, and
// the docx export `comments --witness` makes, through the one export path in
// this binary. It opens the policy `read` opens and asks for no grant, so
// nothing it can send changes a document.
//
// It decides nothing either. The counts are counts, what the projection could
// not carry is a warning naming it, and whether the note should take any of
// what came back is the session's. internal/export holds every rule about the
// file, the pictures and the paths; this file is the arguments, the two reads
// and the envelope.
//
// # export reads twice, writes into the hub, and touches nothing in Drive
//
// `gdoc export <url> --out <file>` is the fifteenth command, decision 15, and
// not a flag on read: it writes files on this machine, which no read command
// does. It makes two requests, the Docs read read makes and the docx export
// comments --witness makes, through exportBytes, the one place this binary
// asks Drive for an export. The picture bytes are the reason for the second
// one: a picture's contentUri in the Docs answer is on a googleusercontent.com
// host the guard admits nowhere. The policy is the one read opens, one file at
// LevelSuggest with no grant of any kind, and every write verb is refused on
// it: TestExportOpensOnlyThePolicyReadOpens in internal/guard and
// TestExportSendsNothingThatWrites here.
//
// --out is the path to write, the flag build uses for the same thing, and not
// a note the run acts on. A free path takes the file. A path holding a note
// whose list names this document keeps every byte it has but the date: the
// note gains exported: {at} on that document's entry, and the copy lands
// beside it at the next free number carrying exported.note, which is what
// makes every writer refuse the copy. Any other file is a taken path and
// nothing more, and the envelope says the path was taken. Two things are
// refused at the door, before a byte is written: a note at a path this run
// would land on that names other documents, and front matter that does not
// read.
//
// Nothing an export finds in its way is ever touched, decision 2, so there is
// no --force and no flag that could grow into one. A taken name takes the next
// free number, note.2.md and then note.3.md, and a picture lands under
// assets/ beside the file as <stem>-1.png, <stem>-2.png and so on, past every
// number already taken there whatever extension wrote it. The folder is
// created when it is missing and left alone when it is not. Every new file
// goes through atomicfile.Create, which ends in a link rather than a rename,
// so a path that appeared while the run was under way is refused instead of
// replaced.
//
// The front matter of a file export creates holds the gdoc: block and nothing
// else: one entry, the document it came from, exported: {at}, the tab id when
// the document has more than one tab, and note: when it is a copy beside one.
// A title is the session's to add when it makes the file a note. A note
// written before 2026-09-19 is schema 1, and the stamp is a write that changes
// the block, so it comes back as schema 2 and the reply says so once, through
// the same rewritten as every other writer.
//
// The envelope counts distinct things. pending is the pending suggestions,
// once per id, because a replace is one change under one id; own is how many
// of those the note at --out lists under proposals, and it is absent rather
// than zero when there is no such list to read; threads is the comments the
// Docs read carried, anchored and unplaced alike. stripped is the house
// prelude pieces with the text each held, so an edit inside a cover table is
// in the envelope and the session can compare it with the note's own keys.
// What the projection could not carry is a warning naming it. Every rule about
// the file itself is internal/export's, and its package comment names the test
// for each.
//
// # The URL picks the entry, and a note gdoc copied is refused
//
// A note names every document it has been published to, under documents: in
// its gdoc: block, and the URL is what says which entry a run acts on. paired
// in paired.go is that lookup, and every command taking --md goes through it,
// so suggestions, propose and withdraw refuse the same things in the same
// words. annotate takes no --md: it leaves a comment on the words a caller
// quotes and records nothing.
//
// Three refusals. A note whose front matter does not read is frontmatter's own,
// passed through. A note that does not name this document is the wrong file,
// and the refusal names every document it does name, because the answer is
// always to open one of those. A note whose entry carries exported.note is a
// copy gdoc wrote beside somebody else's document rather than the source of
// one: there is nothing in it to record, and a snapshot written into it would
// be read next time as that document's own history.
//
// TestEveryWriterActsOnTheURLAndRefusesAnIDOutsideTheList,
// TestAProposalIsRecordedUnderItsOwnDocument,
// TestWithdrawNeverSendsAnIDFromAnotherDocument,
// TestGoneSinceReadsTheSnapshotOfTheDocumentRead and
// TestEveryWriterRefusesACopyThatNamesANote are the pins.
//
// A note published before 2026-09-19 carries schema 1, one document beside the
// schema. It still reads, and the first run that writes to it rewrites the
// block as a list. That is a change to somebody's file, so the envelope says so
// once, and the next run has nothing to say because the block is already a
// list. TestTheReplySaysOnceWhenTheBlockWasRewritten and
// TestAReadThatWritesNothingNeverSaysTheBlockWasRewritten are the pins.
//
// publish is the writer that has no URL to pick with, because it is the run
// that makes the document. It appends: a note that already names a document is
// published again, into a new one, and the entries already there are carried
// through untouched. Those are other documents' history and this run knows
// nothing about them. It refuses one note before anything leaves the machine,
// the copy carrying exported.note, because publish.publishable has to ask that
// over every entry where paired asks it over one.
// TestPublishAppendsAnEntryToAPairedNote and
// TestPublishRefusesACopyThatNamesANote are the pins.
//
// # The note is read again just before it is written
//
// The pairing is checked before the session opens, and the run then spends
// seconds to tens of seconds on the network: a read, a write and three
// read-backs per proposal for propose, and two whole-document reads plus
// a batchUpdate for withdraw. These notes live in a synced vault, so writing
// back the bytes the run started with would throw away whatever landed in that
// window.
//
// freshNote reads the file again and refuses four things rather than writing
// them: a file it cannot read again, one whose front matter no longer parses,
// one whose gdoc: block has gone, and one that no longer names this document.
// Each is a warning carrying the reason, and nothing is written into the note.
// The refusal opens by saying the note changed while the run was under way,
// because that is what tells a reader whether to run the command again or to go
// and look at the file.
//
// The block is re-parsed from the fresh bytes too, so a proposal another run
// recorded in that window survives: writing the block this run read into bytes
// it did not would keep the author's prose and still drop that entry. notePath
// therefore keeps the path, the document the run is of and the entry the
// pairing check read, and never the bytes it read them from, because a copy
// held there would only be the stale bytes somebody later wrote back.
//
// TestProposeWritesTheNoteAsItStandsWhenTheRunFinishes and
// TestWithdrawWritesTheNoteTheVaultHasNow are the pins, with
// TestProposeWarnsWhenTheNoteStopsNamingThisDocumentMidRun over the fourth
// refusal. That last one is a different rule from
// TestProposeRefusesANoteThatNamesAnotherDocument, which is the pairing check at
// the door: that one fails the run before anything is sent, and this one is
// reached with the proposals already in the document, so it warns instead.
//
// publish reads the note again for the opposite reason, and that rule is
// pair's, in publish.go beside this file: there an entry for the document this
// run just made appearing during the upload is what is refused, because a
// second one would name it twice, and the refusal is a rollback that
// internal/publish only carries out.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gdoc/internal/docs"
	"gdoc/internal/docx"
	"gdoc/internal/emit"
	"gdoc/internal/export"
	"gdoc/internal/frontmatter"
	"gdoc/internal/house"
	"gdoc/internal/suggestions"
)

// exportData is what `gdoc export` prints, and every field is a fact.
//
// The counts are counts of distinct things: a replace proposal is one pending
// suggestion here and two entries in `gdoc suggestions`, because own is
// compared against the note's proposals, which hold one id per proposal.
// Threads is the comments the Docs read carried, anchored and unplaced alike.
//
// Own is a pointer so a run whose --out holds no note for this document omits
// it. A zero there would say "none of them is gdoc's own", which is a claim
// this run had nothing to check.
type exportData struct {
	DocumentID string           `json:"document_id"`
	Title      string           `json:"title"`
	RevisionID string           `json:"revision_id"`
	Tabs       int              `json:"tabs"`
	MultiTab   bool             `json:"multi_tab"`
	Files      []export.Placed  `json:"files"`
	Pictures   []export.Picture `json:"pictures"`
	Pending    int              `json:"pending"`
	Own        *int             `json:"own,omitempty"`
	Threads    int              `json:"threads"`
	Stripped   []export.Piece   `json:"stripped"`
	Stamped    []export.Stamp   `json:"stamped,omitempty"`
	Marks      exportMarks      `json:"marks"`
}

// exportMarks is what the document's colours cost this file. The six house
// highlights and the pencil come home as the syntax an author typed, so they
// are not counted here; other is every background no mark claims, by hex, and
// those words came home as prose. The count is a fact and names nothing good
// or bad about it: a document somebody highlighted by hand in Docs is a
// document with marks gdoc has no syntax for, and what to do about that is
// the session's question.
type exportMarks struct {
	Other map[string]int `json:"other"`
}

// cmdExport is the whole of the command, in the order the work has to happen.
//
// --out is read before the document is opened, so a run that named no file is
// refused without a request. The two reads come next, then the pictures are
// paired, then the paths are planned, and only then is anything projected: a
// picture's line in the body names the file that picture lands in, so every
// path has to exist before the text does. Plan is also the door, so a note at
// --out naming other documents refuses the run before a byte is written.
func cmdExport(ctx context.Context, a *args) emit.Result {
	out, err := required(a, "--out")
	if err != nil {
		return failed(err)
	}
	r, err := open(a.target())
	if err != nil {
		return failed(err)
	}
	d, err := docs.Fetch(ctx, r.session, r.id)
	if err != nil {
		return withStop(emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}, err)
	}

	notes := unplacedWarnings(d)

	// The export is a second read of the same document through a second
	// route, and it carries the picture bytes the Docs answer does not. One
	// that could not be read is a warning rather than a failed run: what it
	// costs is the pictures, which the count mismatch below then reports as
	// placeholders, and the text is the answer.
	media, err := exportMedia(ctx, r)
	if err != nil {
		notes = append(notes, fmt.Sprintf(
			"the docx export could not be read, so no picture carries bytes: %v", err))
	}

	src := outSource(out)
	notePics, w := export.NotePictures(src, filepath.Dir(out))
	notes = append(notes, w...)

	pics, w := export.Pictures(documentObjects(d), media, notePics, export.Options{})
	notes = append(notes, w...)

	layout, err := export.Plan(export.Request{
		Out:        out,
		DocumentID: r.id,
		At:         now().UTC(),
		Tabs:       exportTabs(d),
		Pictures:   pics,
	})
	if err != nil {
		return withStop(emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings(notes...)}, err)
	}

	cfg, err := house.Load()
	if err != nil {
		return withStop(emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings(notes...)}, err)
	}
	files, pieces, other, w := export.Project(d, layout.PictureNames(), export.Palette(cfg))
	notes = append(notes, w...)

	written, err := export.Write(layout, files)
	if err != nil {
		return withStop(emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings(notes...)}, err)
	}
	notes = append(notes, written.Warnings...)
	for _, s := range written.Stamped {
		notes = append(notes, rewritten(s.Path, s.SchemaRewritten)...)
	}

	pending := pendingIDs(d)
	data := exportData{
		DocumentID: d.ID,
		Title:      d.Title,
		RevisionID: d.RevisionID,
		Tabs:       len(d.Tabs),
		MultiTab:   d.MultiTab(),
		Files:      written.Files,
		Pictures:   written.Pictures,
		Pending:    len(pending),
		Own:        ownProposals(src, r.id, pending),
		Threads:    len(d.CommentRanges) + len(d.Unplaced),
		Stripped:   pieces,
		Stamped:    written.Stamped,
		Marks:      exportMarks{Other: other},
	}
	if data.Pictures == nil {
		data.Pictures = []export.Picture{}
	}
	if data.Stripped == nil {
		data.Stripped = []export.Piece{}
	}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings(notes...)}
}

// exportMedia is the picture parts of the docx export, in body order. It goes
// through exportBytes, which is the one place this binary asks Drive for an
// export, so `comments --witness`, the survey and this command cannot drift
// into asking for different bytes.
func exportMedia(ctx context.Context, r *reach) ([]docx.Medium, error) {
	b, err := exportBytes(ctx, r)
	if err != nil {
		return nil, err
	}
	return docx.Media(b)
}

// exportTabs is the document's tabs as the writer needs them, in document
// order: which tab it is, and the title a further tab's file name comes from.
func exportTabs(d *docs.Document) []export.Tab {
	tabs := make([]export.Tab, 0, len(d.Tabs))
	for _, t := range d.Tabs {
		tabs = append(tabs, export.Tab{ID: t.ID, Title: t.Title})
	}
	return tabs
}

// documentObjects is every picture of the document in body order, tab by tab.
// The export is of the whole document, so the pairing is over the whole of it
// too: counting one tab against every tab's media would put it out by however
// many pictures stand in front.
func documentObjects(d *docs.Document) []docs.Object {
	var out []docs.Object
	for _, t := range d.Tabs {
		out = append(out, export.Objects(t)...)
	}
	return out
}

// outSource is the bytes at --out, or nothing. A path that holds nothing, a
// directory, or a file this process may not read is nothing here: what is at
// --out is export.Plan's door to judge, and reading it twice in two places
// would be two answers to one question.
func outSource(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

// pendingIDs is every suggestion still pending in the document, once each. A
// replace is an insertion and a deletion under one id, and counting it twice
// would say two where the person who proposed it made one change.
func pendingIDs(d *docs.Document) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range suggestions.IDs(d) {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// ownProposals is how many of the pending suggestions the note at --out
// records as gdoc's own, or nothing at all when there is no such record.
//
// Nothing here is a judgement: the note's block lists the ids gdoc proposed
// from it, and this counts how many of them the document still holds. A file
// at --out that is not a note for this document has nothing to say about it,
// and neither has a note that lists no proposals, so both come back absent
// rather than zero.
func ownProposals(src []byte, documentID string, pending []string) *int {
	block, err := frontmatter.Read(src)
	if err != nil || block == nil {
		return nil
	}
	entry, err := block.Entry(documentID)
	if err != nil || len(entry.Proposals) == 0 {
		return nil
	}
	proposed := map[string]bool{}
	for _, p := range entry.Proposals {
		proposed[p.ID] = true
	}
	count := 0
	for _, id := range pending {
		if proposed[id] {
			count++
		}
	}
	return &count
}
