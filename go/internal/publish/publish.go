package publish

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	// OnStep is told, immediately before each part of the run begins, what
	// that part is called. It is how a caller says what the run is doing now
	// without this package knowing what a record is: the two words below are
	// the two parts that happen here, and the caller decides what they mean
	// and whether anything is written down.
	//
	// A nil OnStep is a caller that does not want to be told, which is every
	// caller but one: cmd/gdoc's publish.
	OnStep func(name string)
}

// The two parts of a publish that happen in this room. The render before it
// and the note after it are the caller's own, and cmd/gdoc names those:
// cmd/gdoc's TestAPublishStepsAreTheFixedNamesInOrder holds all four in order.
const (
	stepUpload   = "upload"
	stepReadBack = "read_back"
)

// step tells the caller a part of the run is beginning. It is called
// immediately before the work, never after it, because a caller that reports
// what is happening now cannot tell the two apart.
func (o Options) step(name string) {
	if o.OnStep == nil {
		return
	}
	o.OnStep(name)
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
	DocumentID string `json:"document_id,omitempty"`
	FolderID   string `json:"folder_id"`
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
	Tabs       int    `json:"tabs,omitempty"`
	Verified   bool   `json:"verified"`
	Checks     Checks `json:"checks"`
	// HeadingsWithoutID is how many of the document's headings came back from
	// the read with no heading id. It is a fact beside Verified rather than a
	// check inside it: the document is there and its words are right, and what
	// is missing is only what a contents entry needs to open a heading. Zero is
	// the answer worth printing, so it has no omitempty.
	HeadingsWithoutID int `json:"headings_without_id"`
	// Marked says the marker over the cover was written and read back. It is
	// beside Verified rather than inside it, because an unmarked document is
	// still the document: mark.go says what it costs.
	Marked   bool     `json:"marked"`
	Warnings []string `json:"warnings,omitempty"`
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
	PostJSON(ctx context.Context, rawURL string, body any, into any) error
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
	o.step(stepUpload)
	id, err := create(ctx, s, o)
	if err != nil {
		return rep, err
	}
	rep.DocumentID = id
	rep.URL = DocumentURL(id)

	o.step(stepReadBack)
	var d *docs.Document
	d, rep.Checks, rep.Title, rep.Tabs, rep.HeadingsWithoutID, rep.Warnings = verify(ctx, s, id, o.Title)
	rep.Verified = rep.Checks.all()
	if d == nil || !rep.Checks.OneTab {
		rep.Warnings = append(rep.Warnings, unmarked("the document could not be read back as one tab, so there was no contents list to mark up to"))
		return rep, nil
	}
	var warns []string
	rep.Marked, warns = mark(ctx, s, id, d.Tabs[0])
	rep.Warnings = append(rep.Warnings, warns...)
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
		// Everything else says the upload failed and stops there, because this
		// arm holds three different things. A guard refusal never left the
		// machine and a 4xx is Drive rejecting the upload whole, so no document
		// was made; a 5xx or a dropped connection is the third, and gdoc cannot
		// tell it apart from either of them. "The document could not be
		// created" would claim otherwise on that third case, and sending
		// somebody to search the folder on the first two would be the warning
		// that cries wolf. The wrapped error carries the status, which is the
		// only thing here that can say which of the three this was.
		return "", fmt.Errorf("the upload into folder %q failed: %w", o.FolderID, err)
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
func verify(ctx context.Context, s Session, id, asked string) (*docs.Document, Checks, string, int, int, []string) {
	var c Checks
	var title string
	var tabs int
	var headless int
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
		var headings int
		headings, headless = countHeadings(d.Tabs)
		if headless > 0 {
			// Named rather than judged, and not a failure: the document is in
			// the folder and its words are right. What it cannot do is open a
			// heading from its contents list, and nothing in the API can give
			// the ids to a document that came back without them, so the way
			// out is a fresh publish from the note.
			warns = append(warns, fmt.Sprintf("%d of the %d headings came back with no heading id, so an entry in the contents list points at the tab rather than at its heading and a click lands on the first page: Drive's import mints the id from the hidden bookmark the docx carries on every heading, and a document already published cannot be given one, so the way to a document whose contents list opens its sections is a fresh publish from the note", headless, headings))
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
	return d, c, title, tabs, headless, warns
}

// countHeadings says how many heading paragraphs the read came back with, and
// how many of those carry no heading id.
//
// A contents entry is a link to a heading by its id, so a heading with none is
// a heading nothing in the list can open. Only HEADING_1 to HEADING_6 are
// counted: the title and the subtitle are not headings to Docs, carry no id and
// reach no contents list.
func countHeadings(tabs []docs.Tab) (headings, without int) {
	for _, t := range tabs {
		h, w := countInBlocks(t.Body)
		headings += h
		without += w
	}
	return headings, without
}

// countInBlocks walks one body, through the cells of its tables. The contents
// list is skipped: its entries are the links to the headings and never
// headings themselves.
func countInBlocks(blocks []docs.Block) (headings, without int) {
	for _, b := range blocks {
		switch {
		case b.Paragraph != nil:
			if !strings.HasPrefix(b.Paragraph.Style, "HEADING_") {
				continue
			}
			headings++
			if b.Paragraph.HeadingID == "" {
				without++
			}
		case b.Table != nil:
			for _, row := range b.Table.Rows {
				for _, cell := range row {
					h, w := countInBlocks(cell.Blocks)
					headings += h
					without += w
				}
			}
		}
	}
	return headings, without
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
