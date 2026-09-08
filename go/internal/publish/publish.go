// Package publish puts a rendered document into Drive as a Google Doc.
//
// The upload is one multipart create: a JSON metadata part naming the folder,
// the title and the Google Doc type, then the docx bytes internal/render
// wrote. Drive converts the second part into a document because the first one
// asked for it, and the new document's id comes back in the answer. That id is
// also the second of the guard's two doors, so nothing here reaches a file the
// run was not given: the policy opens with one folder and no file at all.
//
// A create that answered is not a document that is right, and this package does
// not claim it is. It reads the document back through the Docs API for its
// title and its tab count, and exports it as a docx to see whether the export
// reads as one. Three facts, and Verified is the three together. Fewer than
// three is the document reported with the route that did not hold named, never
// a failure: a document that exists is a document that exists, and a caller
// told the run failed is a caller that uploads a second one.
//
// Nothing here reads or writes a file. The note's bytes are cmd/gdoc's, which
// is what keeps this package testable on a fake session, and it is also what
// makes Rollback a separate call: whether the pairing could be recorded is a
// question about a file, and this room does not have one.
package publish

import (
	"context"
	"errors"
	"fmt"

	"gdoc/internal/docs"
	"gdoc/internal/docx"
	"gdoc/internal/drive"
)

// mimeDocument is the Drive type of a Google Doc. Naming it in the metadata is
// what turns the uploaded docx into a document rather than a docx sitting in a
// folder with a document's name.
const mimeDocument = "application/vnd.google-apps.document"

// mimeDocx is what the second part carries. It is the type Drive converts from,
// and it is the same type internal/docx asks the export back in.
const mimeDocx = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

// Options is one publish: the folder it lands in, the name that goes on it, and
// the bytes.
//
// There is no document id here, and that is the shape rather than an omission.
// A publish makes a document; a note that already names one is refused by the
// caller before this package is reached.
type Options struct {
	FolderID string
	Title    string
	Docx     []byte
}

// Check is the shape rule, asked before anything leaves the machine. Each of
// the three is refused by name: an upload missing one of them is a request
// Drive would answer to with something that names none of gdoc's own words.
func (o Options) Check() error {
	if o.FolderID == "" {
		return errors.New("the publish names no folder, and a create with no parent is a document nobody asked for in a place nobody named")
	}
	if o.Title == "" {
		return errors.New("the publish carries no title, and the title is the name the document is found by")
	}
	if len(o.Docx) == 0 {
		return errors.New("the publish carries no bytes, and there is nothing to convert into a document")
	}
	return nil
}

// Checks is which of the three read-backs held. They are three routes to one
// question, and each answers a part of it the other two cannot: the Docs read
// says the document is there and readable, the tab count says it is one
// document rather than a shape a later command would refuse, and the export
// says Drive can hand the document back as the format it came in as.
type Checks struct {
	ReadBack   bool `json:"read_back"`
	OneTab     bool `json:"one_tab"`
	DocxExport bool `json:"docx_export"`
}

// all says whether every route held.
func (c Checks) all() bool { return c.ReadBack && c.OneTab && c.DocxExport }

// Report is what the run left in Drive.
//
// Warnings carry the reasons Verified is false, the way propose.Result and
// probe.Report carry theirs, and for the same reason: everything that goes
// wrong after the create is a fact about a document that already exists.
//
// Title is what the read-back carried rather than what the upload asked for. A
// document is found by the name Drive gave it, so that is the fact worth
// reporting, and the two disagreeing is a warning naming both.
type Report struct {
	DocumentID string   `json:"document_id,omitempty"`
	FolderID   string   `json:"folder_id"`
	URL        string   `json:"url,omitempty"`
	Title      string   `json:"title,omitempty"`
	Tabs       int      `json:"tabs,omitempty"`
	Verified   bool     `json:"verified"`
	Checks     Checks   `json:"checks"`
	Warnings   []string `json:"warnings,omitempty"`
}

// Session is what this package needs of a session: the multipart write the
// upload is, the JSON read the read-back is, the byte read the export is, and
// the PATCH Drive spells trashing as. Naming the interface here keeps net/http
// out of this room, which is what the boundary test asks of every package but
// the four that build requests.
type Session interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	GetBytes(ctx context.Context, rawURL string, limit int64) ([]byte, error)
	PatchJSON(ctx context.Context, rawURL string, body any, into any) error
	PostMultipart(ctx context.Context, rawURL string, meta any, part []byte, partType string, into any) error
}

// UploadURL is files.create on the upload route, in the one body shape the
// guard carries.
//
// fields=id is not a saving: it is what the guard reads the new id out of, and
// a create whose id is not in the answer is one the policy cannot learn, so
// every request after it would be refused as a file this command was not given.
func UploadURL() string {
	return "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&fields=id&supportsAllDrives=true"
}

// DocumentURL is where a person opens the document.
func DocumentURL(id string) string {
	return "https://docs.google.com/document/d/" + id + "/edit"
}

// Run uploads the document and reads it back three ways.
//
// It fails only before the create, and on a create with no readable id. After
// Drive has made the document everything is reported rather than raised,
// because the caller's next move depends on the document existing: a report
// with an id is a pairing to record or a document to take back, and an error is
// neither.
func Run(ctx context.Context, s Session, o Options) (Report, error) {
	rep := Report{FolderID: o.FolderID}
	if err := o.Check(); err != nil {
		return rep, err
	}
	id, err := create(ctx, s, o)
	if err != nil {
		return rep, err
	}
	rep.DocumentID = id
	rep.URL = DocumentURL(id)

	rep.Checks, rep.Title, rep.Tabs, rep.Warnings = verify(ctx, s, id, o.Title)
	rep.Verified = rep.Checks.all()
	return rep, nil
}

// create uploads the bytes and returns the id Drive answered with.
func create(ctx context.Context, s Session, o Options) (string, error) {
	meta := map[string]any{
		"name":     o.Title,
		"mimeType": mimeDocument,
		"parents":  []string{o.FolderID},
	}
	var answer struct {
		ID string `json:"id"`
	}
	if err := s.PostMultipart(ctx, UploadURL(), meta, o.Docx, mimeDocx, &answer); err != nil {
		// A failure raised after Drive accepted the create is not a create that
		// did not happen. The document is in the folder and its id was in the
		// answer nothing could read, so gdoc can neither verify it, record it
		// nor trash it. Saying it could not be created sends somebody to look
		// for a failure while the document sits in their Drive, and the next
		// publish of the same note makes a second one.
		if sentAnyway(err) {
			return "", fmt.Errorf("Drive accepted the upload and its answer could not be read, so a document may be in folder %q with no id for gdoc to name, verify or take back: %w", o.FolderID, err)
		}
		return "", fmt.Errorf("the document could not be created in folder %q: %w", o.FolderID, err)
	}
	if answer.ID == "" {
		// Without an id nothing further is reachable: the guard learns the new
		// file from this answer, so a create it could not read is a create with
		// no document behind it as far as the rest of the run is concerned. It
		// is not one as far as Drive is concerned, which is why this says the
		// document may be there rather than that it is not.
		return "", fmt.Errorf("the upload into folder %q answered with no document id, so there is nothing to verify, to record or to take back, and a document may be sitting there unnamed", o.FolderID)
	}
	return answer.ID, nil
}

// verify reads the new document back two ways and reports what each said. It
// raises nothing: every failure here is a fact about a document that exists,
// and the caller decides what to tell Nail.
func verify(ctx context.Context, s Session, id, asked string) (Checks, string, int, []string) {
	var c Checks
	var title string
	var tabs int
	var warns []string

	d, err := docs.Fetch(ctx, s, id)
	switch {
	case err != nil:
		// one_tab stays false with it: a document nothing could read is a
		// document nothing counted the tabs of, and reporting the check as
		// holding would be a fact about a read that did not happen.
		warns = append(warns, fmt.Sprintf("the document %q was created and could not be read back, so neither its title nor its tab count is known: %v", id, err))
	default:
		c.ReadBack = true
		title = d.Title
		tabs = len(d.Tabs)
		c.OneTab = tabs == 1
		if !c.OneTab {
			warns = append(warns, fmt.Sprintf("the document came back with %d tabs, and a published note is one document with one tab: a comment range and a proposal both name a position, and a position means nothing without saying which tab it is in", tabs))
		}
		if title != asked {
			// Two facts disagreeing, named rather than judged. Drive takes the
			// name from the metadata part, so the usual answer is that they
			// match and this says nothing.
			warns = append(warns, fmt.Sprintf("the upload asked for the title %q and the document came back titled %q", asked, title))
		}
	}

	b, err := docx.Export(ctx, s, id)
	switch {
	case err != nil:
		warns = append(warns, fmt.Sprintf("the document %q could not be exported as a docx, so nothing checked that Drive reads it back as the format it went in as: %v", id, err))
	default:
		if _, err := docx.Parse(b); err != nil {
			warns = append(warns, fmt.Sprintf("the docx export of %q did not read as a docx: %v", id, err))
		} else {
			c.DocxExport = true
		}
	}
	return c, title, tabs, warns
}

// Rollback takes back a document this run published, because the pairing could
// not be recorded in the note.
//
// It is a separate call rather than a step inside Run, because what it answers
// to is a file this package never touches. The three steps are internal/drive's,
// so the probe and a publish cannot disagree about whether an unconfirmed trash
// counts as a trash.
//
// A rollback that did not hold is false with a warning naming the document and
// the address to open. Not knowing must never resolve to the document being
// gone: a caller told the rollback held forgets the id, and what is left is a
// document nobody knows about that the next publish of this note makes a second
// of.
func Rollback(ctx context.Context, s Session, docID string) (bool, []string) {
	if err := drive.Trash(ctx, s, docID); err != nil {
		return false, []string{fmt.Sprintf(
			"the document %q was published and the note could not be paired to it, and it could not be taken back: %v; open %s and delete it by hand, or the next publish of this note makes a second document",
			docID, err, DocumentURL(docID))}
	}
	return true, nil
}

// sentAnyway says whether the request reached Drive in spite of the error. The
// session marks the failures raised after the server accepted a request, and
// this room asks by behaviour rather than by importing that package: naming a
// Session interface here is what keeps net/http out, and an imported sentinel
// would bring it back through the side door.
func sentAnyway(err error) bool {
	var sent interface{ Sent() bool }
	return errors.As(err, &sent) && sent.Sent()
}
