// The command that puts a note into Drive.
//
// `gdoc publish --md note.md --folder-id FOLDER` renders the note the way
// `build` renders it, uploads the bytes with conversion into the one folder the
// run was given, reads the new document back three ways, and records the
// pairing in the note. It is the only command that creates the gdoc: block:
// `suggestions --md`, `propose --md` and `withdraw` write into a block that is
// already there, and each refuses a note that has none.
//
// Two rules shape everything below. The run's only door is the folder: no file
// is in the reachable set when the policy is opened, and the new document's id
// is learned from the create the guard itself carried. And not knowing never
// resolves to keeping the document: a pairing that could not be recorded leaves
// a document nobody knows about, which the next publish of the same note would
// make a second of, so the document goes back and the report says so.

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"gdoc/internal/body"
	"gdoc/internal/emit"
	"gdoc/internal/frontmatter"
	"gdoc/internal/guard"
	"gdoc/internal/publish"
)

// publishData is what `gdoc publish` prints: what was made, what was checked,
// and what was written. Every field is a fact. Whether the document is right is
// answered by opening it in Drive.
//
// Title is what the read-back carried rather than what the upload asked for,
// which is publish.Report's rule: a document is found by the name Drive gave
// it. The two disagreeing is a warning naming both, and a read-back that did
// not happen leaves the field out rather than filling it in with the title that
// was asked for.
//
// RolledBack is a pointer so that a run which recorded the pairing says nothing
// about a rollback at all. false has to mean "it was tried and did not hold",
// which is the run where the live id below it matters most.
type publishData struct {
	DocumentID   string         `json:"document_id,omitempty"`
	FolderID     string         `json:"folder_id"`
	URL          string         `json:"url,omitempty"`
	Title        string         `json:"title,omitempty"`
	House        string         `json:"house"`
	Bytes        int64          `json:"bytes"`
	Tabs         int            `json:"tabs,omitempty"`
	Verified     bool           `json:"verified"`
	Checks       publish.Checks `json:"checks"`
	RolledBack   *bool          `json:"rolled_back,omitempty"`
	FilesChanged []string       `json:"files_changed,omitempty"`
	Body         body.Counts    `json:"body"`
}

func cmdPublish(a *args) emit.Result {
	md, err := required(a, "--md")
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	target, err := required(a, "--folder-id")
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	folder, err := folderID(target)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}

	// The note is read and judged before anything else happens. There is no
	// republish: a note that already names a document has one, and publishing
	// it again makes a second document with the first one's pairing left
	// pointing at neither.
	source, err := noteSource(md)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	if err := unpaired(md, source); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}

	doc, err := renderNote(source, md, a)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}

	// One door, and it is the folder. No file is named, so the document this
	// run is about is unreachable until the create the guard carried teaches
	// the policy its id.
	p := guard.NewPolicy()
	p.AllowCreateIn(folder)
	s, err := openSession(p)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	return runPublish(context.Background(), s, md, source, folder, doc)
}

// runPublish is the run itself: upload, verify, pair, and take the document
// back if the pairing could not be recorded.
func runPublish(ctx context.Context, s session, md string, source []byte, folder string, doc *noteDocx) emit.Result {
	warns := append([]string{}, doc.Warnings...)
	data := publishData{
		FolderID: folder,
		House:    doc.House,
		Bytes:    int64(len(doc.Docx)),
		Body:     doc.Counts,
	}
	rep, err := publish.Run(ctx, s, publish.Options{
		FolderID: folder,
		Title:    doc.Title,
		Docx:     doc.Docx,
	})
	data.DocumentID = rep.DocumentID
	data.URL = rep.URL
	data.Title = rep.Title
	data.Tabs = rep.Tabs
	data.Verified = rep.Verified
	data.Checks = rep.Checks
	warns = append(warns, rep.Warnings...)
	if err != nil {
		// No id, so there is nothing to verify, to record or to take back. The
		// error already names the folder, and says whether a document may be
		// sitting in it: publish.create has three failure shapes and each one
		// answers that differently.
		return emit.Result{OK: false, Error: err.Error(), Data: data, Warnings: sessionWarnings(s, warns)}
	}

	changed, pairErr := pair(md, source, folder, rep.DocumentID, doc)
	if pairErr == nil {
		data.FilesChanged = changed
		return emit.Result{OK: true, Data: data, Warnings: sessionWarnings(s, warns)}
	}

	// The document exists and nothing records it. Left there it is a document
	// nobody knows about, and the next publish of this note makes a second one,
	// so it goes back before the run answers.
	rolled, rollWarns := publish.Rollback(ctx, s, rep.DocumentID)
	warns = append(warns, rollWarns...)
	data.RolledBack = &rolled
	if rolled {
		// It is not there any more, so naming it would send somebody to look
		// for a document that has gone.
		data.DocumentID = ""
		data.URL = ""
	}
	return emit.Result{OK: false, Error: pairErr.Error(), Data: data, Warnings: sessionWarnings(s, warns)}
}

// unpaired refuses a note that already names a document.
//
// A block gdoc cannot read is refused here too, and by the same sentence it
// would be refused by anywhere else: a note whose front matter does not parse
// is one publish would put a second block into, demoting the author's keys to
// prose.
func unpaired(md string, source []byte) error {
	block, err := frontmatter.Read(source)
	if err != nil {
		return err
	}
	if block != nil {
		return fmt.Errorf("%s is already paired with document %s, and gdoc publishes a note once: open that document, or take the gdoc: block out by hand if it names a document that has gone",
			md, block.DocumentID)
	}
	return nil
}

// pair records the pairing in the note, and refuses four things rather than
// writing them: a note that could not be read again, one whose front matter no
// longer parses, one whose gdoc: block has appeared, and one whose bytes
// changed at all.
//
// The note is read again first, because seconds to tens of seconds of network
// sit between the read the render was made from and here, and these notes live
// in a synced vault. The first two refusals are the read itself: a note gdoc
// cannot read, or cannot parse, is one it cannot write into either, and writing
// over it anyway would replace whatever landed there with this run's guess.
//
// The block refusal is the inverse of freshNote's rule: that one guards a
// paired note and refuses a block that has gone, and this one starts from an
// unpaired note and refuses a block that has appeared. A block that appeared is
// another run pairing this note while this one was uploading, and overwriting
// it would leave that run's document with no record at all.
//
// The fourth refusal is the note's own bytes changing. The document in Drive is
// a render of the bytes this run read, so a note that has moved on is paired to
// a document that is no longer what it builds to. All four are a rollback
// rather than a warning: the caller's answer to each is to publish again from
// what the note says now.
func pair(md string, source []byte, folder, docID string, doc *noteDocx) ([]string, error) {
	fresh, err := os.ReadFile(md)
	if err != nil {
		return nil, fmt.Errorf("the document was published and %s could not be read again, so the pairing could not be recorded: %v", md, err)
	}
	block, err := frontmatter.Read(fresh)
	if err != nil {
		return nil, fmt.Errorf("the document was published and %s no longer reads, so the pairing could not be recorded: %v", md, err)
	}
	if block != nil {
		return nil, fmt.Errorf("the document was published and %s now names document %s, so another run paired it while this one was uploading and this run's document is not recorded anywhere",
			md, block.DocumentID)
	}
	if !bytes.Equal(fresh, source) {
		return nil, fmt.Errorf("the document was published and %s changed while it was being uploaded, so the document is a render of bytes the note no longer holds", md)
	}
	out, err := frontmatter.Write(fresh, &frontmatter.Block{
		Schema:     frontmatter.Schema,
		DocumentID: docID,
		FolderID:   folder,
		Published: &frontmatter.Published{
			At: now().UTC(),
			// What went on the cover, which is what the upload asked Drive to
			// name the file. The read-back's title is reported on the envelope
			// instead, and the two disagreeing is already a warning there.
			Title: doc.Title,
			House: doc.House,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("the document was published and the gdoc: block could not be written into %s: %v", md, err)
	}
	if err := writeFile(md, out); err != nil {
		return nil, fmt.Errorf("the document was published and %s could not be written, so the pairing could not be recorded: %v", md, err)
	}
	return []string{md}, nil
}

// sessionWarnings is the session's warnings, which are the policy's followed by
// its own, in front of the command's. It is reach.warnings for a command that
// has no reach: publish opens with a folder rather than a document.
func sessionWarnings(s session, own []string) []string {
	out := append([]string{}, s.Warnings()...)
	return append(out, own...)
}
