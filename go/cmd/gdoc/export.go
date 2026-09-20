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
		return emit.Result{OK: false, Error: err.Error()}
	}
	r, err := open(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	d, err := docs.Fetch(ctx, r.session, r.id)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
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
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings(notes...)}
	}

	files, pieces, w := export.Project(d, layout.PictureNames())
	notes = append(notes, w...)

	written, err := export.Write(layout, files)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings(notes...)}
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
