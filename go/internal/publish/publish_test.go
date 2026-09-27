package publish

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/drive"
)

const (
	testFolderID = "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"
	testDocID    = "1PuBl15h3D0000000000000000000000000000000"
	testTitle    = "Supplier Register Policy"
)

// call is one request the fake was asked to make. meta and part are what a
// multipart upload carried, kept apart so a test reads the metadata Drive would
// parse rather than the assembled body, which this package never builds.
type call struct {
	method string
	url    string
	body   json.RawMessage
	meta   json.RawMessage
	part   []byte
	typ    string
}

// fakeSession answers each request by what it asked for rather than by its
// place in a script, the way probe's and propose's fakes do. Nothing here names
// net/http: the rooms that may are the ones the boundary test lists, and this
// is not one of them.
type fakeSession struct {
	calls   []call
	create  []byte        // the files.create answer
	read    []byte        // the documents.get answer
	marked  []byte        // the documents.get answer once a batchUpdate went out, nil for read
	export  []byte        // the docx export
	trashed []byte        // the files.get?fields=trashed answer
	failAt  map[int]error // fail the nth call, counted from zero
}

func (f *fakeSession) record(c call) error {
	f.calls = append(f.calls, c)
	return f.failAt[len(f.calls)-1]
}

func (f *fakeSession) GetJSON(_ context.Context, rawURL string, into any) error {
	if err := f.record(call{method: "GET", url: rawURL}); err != nil {
		return err
	}
	switch {
	case strings.Contains(rawURL, "fields=trashed"):
		return decodeInto(f.trashed, into)
	case f.marked != nil && f.sentBatch():
		return decodeInto(f.marked, into)
	default:
		return decodeInto(f.read, into)
	}
}

// sentBatch says whether a batchUpdate was asked for before this read.
func (f *fakeSession) sentBatch() bool {
	for _, c := range f.calls {
		if strings.HasSuffix(c.url, ":batchUpdate") {
			return true
		}
	}
	return false
}

func (f *fakeSession) PostJSON(_ context.Context, rawURL string, body any, into any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if err := f.record(call{method: "POST", url: rawURL, body: raw}); err != nil {
		return err
	}
	return decodeInto(nil, into)
}

func (f *fakeSession) GetBytes(_ context.Context, rawURL string, _ int64) ([]byte, error) {
	if err := f.record(call{method: "GET", url: rawURL}); err != nil {
		return nil, err
	}
	return f.export, nil
}

func (f *fakeSession) PatchJSON(_ context.Context, rawURL string, body any, into any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if err := f.record(call{method: "PATCH", url: rawURL, body: raw}); err != nil {
		return err
	}
	return decodeInto(nil, into)
}

func (f *fakeSession) PostMultipart(_ context.Context, rawURL string, meta any, part []byte, typ string, into any) error {
	raw, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	if err := f.record(call{method: "POST", url: rawURL, meta: raw, part: part, typ: typ}); err != nil {
		return err
	}
	return decodeInto(f.create, into)
}

func decodeInto(answer []byte, into any) error {
	if into == nil {
		return nil
	}
	if len(answer) == 0 {
		answer = []byte(`{}`)
	}
	return json.Unmarshal(answer, into)
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// script is a Google that answers everything, with the read-back taken from the
// named fixture.
func script(t *testing.T, read string) *fakeSession {
	t.Helper()
	return &fakeSession{
		create:  []byte(`{"id":"` + testDocID + `"}`),
		read:    fixture(t, read),
		marked:  fixture(t, "published-marked.json"),
		export:  aDocx(t),
		trashed: fixture(t, "trashed.json"),
		failAt:  map[int]error{},
	}
}

// aDocx is an export that reads as one: a zip carrying word/document.xml, which
// is the part internal/docx refuses an export without.
func aDocx(t *testing.T) []byte {
	t.Helper()
	const documentXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body><w:p><w:r><w:t>Supplier Register Policy</w:t></w:r></w:p></w:body>
</w:document>`
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	w, err := z.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(documentXML)); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func options() Options {
	return Options{FolderID: testFolderID, Title: testTitle, Docx: []byte("PK\x03\x04 the rendered document")}
}

// acceptedError is a failure raised after the server had already accepted the
// request, which is what gapi marks a lost answer as. The packages ask by
// behaviour rather than by importing gapi, so a test fakes the behaviour.
type acceptedError struct{ error }

func (acceptedError) Sent() bool { return true }

func TestAPublishUploadsVerifiesAndReportsTheDocument(t *testing.T) {
	f := script(t, "published-contents.json")

	rep, err := Run(context.Background(), f, options())

	if err != nil {
		t.Fatal(err)
	}
	if rep.DocumentID != testDocID {
		t.Errorf("DocumentID = %q, want the id the create answered with", rep.DocumentID)
	}
	if rep.FolderID != testFolderID {
		t.Errorf("FolderID = %q, want the folder the run was given", rep.FolderID)
	}
	if rep.URL != "https://docs.google.com/document/d/"+testDocID+"/edit" {
		t.Errorf("URL = %q, want the document's own address", rep.URL)
	}
	if rep.Title != testTitle {
		t.Errorf("Title = %q, want the title the read-back carried", rep.Title)
	}
	if rep.Tabs != 1 {
		t.Errorf("Tabs = %d, want one", rep.Tabs)
	}
	if !rep.Verified {
		t.Errorf("Verified is false on a run where all three routes held: %+v %v", rep.Checks, rep.Warnings)
	}
	want := Checks{ReadBack: true, OneTab: true, DocxExport: true}
	if rep.Checks != want {
		t.Errorf("Checks = %+v, want %+v", rep.Checks, want)
	}
	if !rep.Marked {
		t.Error("Marked is false on a run whose marker read back")
	}
	if len(rep.Warnings) != 0 {
		t.Errorf("a run where every step answered carried warnings: %v", rep.Warnings)
	}
}

func TestTheUploadNamesTheFolderTheTitleAndTheConversion(t *testing.T) {
	f := script(t, "published.json")

	if _, err := Run(context.Background(), f, options()); err != nil {
		t.Fatal(err)
	}

	var meta struct {
		Name     string   `json:"name"`
		MimeType string   `json:"mimeType"`
		Parents  []string `json:"parents"`
	}
	if err := json.Unmarshal(f.calls[0].meta, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Name != testTitle {
		t.Errorf("the metadata named the file %q, want the note's title", meta.Name)
	}
	if meta.MimeType != "application/vnd.google-apps.document" {
		t.Errorf("mimeType = %q; without the Google Doc type Drive keeps the docx as a file rather than converting it", meta.MimeType)
	}
	if len(meta.Parents) != 1 || meta.Parents[0] != testFolderID {
		t.Errorf("parents = %v, want exactly the one folder the run was given", meta.Parents)
	}
	if f.calls[0].typ != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		t.Errorf("the file part is typed %q, want the docx media type", f.calls[0].typ)
	}
	if !bytes.Equal(f.calls[0].part, options().Docx) {
		t.Error("the file part is not the bytes the render produced")
	}
	if !strings.Contains(f.calls[0].url, "uploadType=multipart") {
		t.Errorf("the upload URL %q does not ask for the multipart shape the guard judges", f.calls[0].url)
	}
	if !strings.Contains(f.calls[0].url, "/upload/drive/v3/files") {
		t.Errorf("the upload URL %q is not on the upload route", f.calls[0].url)
	}
}

func TestAnUploadTheGuardRefusedIsAnErrorNamingTheFolder(t *testing.T) {
	f := script(t, "published.json")
	// A guard refusal reaches a caller as a plain error: nothing left the
	// machine, so it carries no mark saying the request was sent.
	f.failAt[0] = errors.New(`the create names folder "OTHER", which this command was not given`)

	rep, err := Run(context.Background(), f, options())

	if err == nil {
		t.Fatal("a refused upload came back as success")
	}
	if !strings.Contains(err.Error(), testFolderID) {
		t.Errorf("the error does not name the folder the upload targeted: %v", err)
	}
	if strings.Contains(err.Error(), "may be in") {
		t.Errorf("nothing left the machine, so the error must not say a document may be there: %v", err)
	}
	if rep.DocumentID != "" {
		t.Errorf("DocumentID = %q on a refusal, and no document was made", rep.DocumentID)
	}
	if len(f.calls) != 1 {
		t.Errorf("the fake saw %d requests, want one: nothing follows a refused upload", len(f.calls))
	}
}

func TestAnUploadDriveRejectedIsAnErrorNamingTheFolder(t *testing.T) {
	f := script(t, "published.json")
	f.failAt[0] = errors.New("files.create answered 403: The user does not have sufficient permissions for this folder")

	_, err := Run(context.Background(), f, options())

	if err == nil {
		t.Fatal("an upload Drive rejected came back as success")
	}
	if !strings.Contains(err.Error(), "sufficient permissions") {
		t.Errorf("the error lost what Drive said: %v", err)
	}
	if !strings.Contains(err.Error(), testFolderID) {
		t.Errorf("the error does not name the folder: %v", err)
	}
}

func TestAnUploadWhoseAnswerWasLostNamesTheFolderAndSaysADocumentMayBeThere(t *testing.T) {
	f := script(t, "published.json")
	f.failAt[0] = acceptedError{errors.New("the answer is not JSON")}

	rep, err := Run(context.Background(), f, options())

	if err == nil {
		t.Fatal("an upload Drive accepted whose answer was lost came back as success")
	}
	if !strings.Contains(err.Error(), testFolderID) {
		t.Errorf("there is no id to name, so the error must name the folder: %v", err)
	}
	if !strings.Contains(err.Error(), "may") {
		t.Errorf("Drive accepted the upload, so the error must not claim the document was not created: %v", err)
	}
	if rep.DocumentID != "" {
		t.Errorf("DocumentID = %q, and the id was in the answer nothing could read", rep.DocumentID)
	}
	if len(f.calls) != 1 {
		t.Errorf("the fake saw %d requests, want one: with no id there is nothing to verify or trash", len(f.calls))
	}
}

func TestAnUploadThatAnsweredWithNoIDIsAnErrorNamingTheFolder(t *testing.T) {
	f := script(t, "published.json")
	f.create = []byte(`{}`)

	rep, err := Run(context.Background(), f, options())

	if err == nil {
		t.Fatal("a create that answered with no id came back as success")
	}
	if !strings.Contains(err.Error(), testFolderID) {
		t.Errorf("the error does not name the folder: %v", err)
	}
	if !strings.Contains(err.Error(), "may") {
		t.Errorf("Drive answered, so the error must not claim the document was not created: %v", err)
	}
	if rep.DocumentID != "" {
		t.Errorf("DocumentID = %q, and there is no id to name", rep.DocumentID)
	}
}

func TestAReadBackThatFailedIsReportedRatherThanRaised(t *testing.T) {
	f := script(t, "published.json")
	f.failAt[1] = errors.New("documents.get answered 500")

	rep, err := Run(context.Background(), f, options())

	if err != nil {
		t.Fatalf("the document exists, so a failed read-back is a fact about it rather than the run's error: %v", err)
	}
	if rep.DocumentID != testDocID {
		t.Errorf("DocumentID = %q, and a document that was created is never lost from the report", rep.DocumentID)
	}
	if rep.Checks.ReadBack {
		t.Error("read_back is true on a read that failed")
	}
	if rep.Checks.OneTab {
		t.Error("one_tab is true on a document nothing could count the tabs of")
	}
	if rep.Verified {
		t.Error("Verified is true with a route that did not hold")
	}
	if !rep.Checks.DocxExport {
		t.Error("the export answered, and a failed Docs read is not a reason to skip it")
	}
	if !strings.Contains(strings.Join(rep.Warnings, " "), "answered 500") {
		t.Errorf("warnings = %v, want one carrying what Docs said", rep.Warnings)
	}
}

func TestADocumentWithTwoTabsIsNotVerified(t *testing.T) {
	f := script(t, "two-tabs.json")

	rep, err := Run(context.Background(), f, options())

	if err != nil {
		t.Fatal(err)
	}
	if !rep.Checks.ReadBack {
		t.Error("read_back is false, and the document was read")
	}
	if rep.Checks.OneTab {
		t.Error("one_tab is true on a document with two tabs")
	}
	if rep.Tabs != 2 {
		t.Errorf("Tabs = %d, want two", rep.Tabs)
	}
	if rep.Verified {
		t.Error("Verified is true with a route that did not hold")
	}
	if !strings.Contains(strings.Join(rep.Warnings, " "), "2 tabs") {
		t.Errorf("warnings = %v, want one naming how many tabs came back", rep.Warnings)
	}
}

func TestATitleThatCameBackDifferentIsNamedOnBothSides(t *testing.T) {
	f := script(t, "published.json")
	o := options()
	o.Title = "Supplier register policy"

	rep, err := Run(context.Background(), f, o)

	if err != nil {
		t.Fatal(err)
	}
	if rep.Title != testTitle {
		t.Errorf("Title = %q, want what the read-back carried rather than what was asked for", rep.Title)
	}
	joined := strings.Join(rep.Warnings, " ")
	if !strings.Contains(joined, testTitle) || !strings.Contains(joined, o.Title) {
		t.Errorf("warnings = %v, want one naming the title that was asked for and the one that came back", rep.Warnings)
	}
}

func TestAnExportThatIsNotADocxIsNotVerified(t *testing.T) {
	f := script(t, "published.json")
	f.export = []byte("<html><body>Sign in</body></html>")

	rep, err := Run(context.Background(), f, options())

	if err != nil {
		t.Fatalf("the document exists, so an export that did not read is a fact about it: %v", err)
	}
	if rep.Checks.DocxExport {
		t.Error("docx_export is true on bytes that are not a docx")
	}
	if !rep.Checks.ReadBack || !rep.Checks.OneTab {
		t.Errorf("the other two routes held and were dropped: %+v", rep.Checks)
	}
	if rep.Verified {
		t.Error("Verified is true with a route that did not hold")
	}
	if !strings.Contains(strings.Join(rep.Warnings, " "), "export") {
		t.Errorf("warnings = %v, want one naming the export", rep.Warnings)
	}
}

func TestAnExportThatNeverArrivedIsNotVerified(t *testing.T) {
	f := script(t, "published.json")
	f.failAt[2] = errors.New("the export timed out")

	rep, err := Run(context.Background(), f, options())

	if err != nil {
		t.Fatal(err)
	}
	if rep.Checks.DocxExport {
		t.Error("docx_export is true on an export that never arrived")
	}
	if !strings.Contains(strings.Join(rep.Warnings, " "), "timed out") {
		t.Errorf("warnings = %v, want one carrying what went wrong: %v", rep.Warnings, rep.Warnings)
	}
}

func TestRunSendsTheRequestsInOrder(t *testing.T) {
	f := script(t, "published-contents.json")

	if _, err := Run(context.Background(), f, options()); err != nil {
		t.Fatal(err)
	}

	want := []call{
		{method: "POST", url: UploadURL()},
		{method: "GET", url: docs.URL(testDocID)},
		{method: "GET", url: "https://www.googleapis.com/drive/v3/files/" + testDocID + "/export?mimeType=application%2Fvnd.openxmlformats-officedocument.wordprocessingml.document"},
		{method: "POST", url: "https://docs.googleapis.com/v1/documents/" + testDocID + ":batchUpdate"},
		{method: "GET", url: docs.URL(testDocID)},
	}
	if len(f.calls) != len(want) {
		t.Fatalf("the fake saw %d requests, want %d: the upload, the read-back, the export, the marker and its read-back", len(f.calls), len(want))
	}
	for i, w := range want {
		if f.calls[i].method != w.method || f.calls[i].url != w.url {
			t.Errorf("call %d = %s %s, want %s %s", i, f.calls[i].method, f.calls[i].url, w.method, w.url)
		}
	}
}

func TestOptionsRefuseWhatAPublishCannotBeMadeFrom(t *testing.T) {
	full := options()
	cases := []struct {
		name string
		o    Options
		want string
	}{
		{"no folder", Options{Title: full.Title, Docx: full.Docx}, "folder"},
		{"no title", Options{FolderID: full.FolderID, Docx: full.Docx}, "title"},
		{"no bytes", Options{FolderID: full.FolderID, Title: full.Title}, "bytes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.o.Check()
			if err == nil {
				t.Fatalf("%s was accepted", c.name)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal does not name what is missing: %v", err)
			}
		})
	}
}

func TestAnOptionsFailureStopsBeforeAnythingLeavesTheMachine(t *testing.T) {
	f := script(t, "published.json")

	if _, err := Run(context.Background(), f, Options{Title: testTitle, Docx: []byte("x")}); err == nil {
		t.Fatal("a publish with no folder was accepted")
	}
	if len(f.calls) != 0 {
		t.Errorf("the fake saw %d requests, want none: the shape is checked before the wire", len(f.calls))
	}
}

func TestARollbackDriveConfirmedSaysSo(t *testing.T) {
	f := script(t, "published.json")

	rolled, warns := Rollback(context.Background(), f, testDocID)

	if !rolled {
		t.Errorf("the trash was confirmed and Rollback said otherwise: %v", warns)
	}
	if len(warns) != 0 {
		t.Errorf("a confirmed rollback carried warnings: %v", warns)
	}
	if len(f.calls) != 2 {
		t.Fatalf("the fake saw %d requests, want two: the trash and the read that confirms it", len(f.calls))
	}
	if f.calls[0].url != drive.FileURL(testDocID) || f.calls[1].url != drive.TrashedURL(testDocID) {
		t.Errorf("the rollback did not go through the one trash rule: %v", f.calls)
	}
}

func TestARollbackDriveDidNotConfirmNamesTheDocumentToDeleteByHand(t *testing.T) {
	f := script(t, "published.json")
	f.trashed = []byte(`{"trashed":false}`)

	rolled, warns := Rollback(context.Background(), f, testDocID)

	if rolled {
		t.Error("Rollback reported a clean rollback on a trash Drive did not confirm; not knowing never resolves to the document being gone")
	}
	joined := strings.Join(warns, " ")
	if !strings.Contains(joined, testDocID) {
		t.Errorf("warnings = %v, want one naming the document that is still live", warns)
	}
	if !strings.Contains(joined, "https://docs.google.com/document/d/"+testDocID) {
		t.Errorf("warnings = %v, want one carrying the address to open", warns)
	}
	if !strings.Contains(joined, "by hand") {
		t.Errorf("warnings = %v, want one saying what a person has to do", warns)
	}
}

func TestARollbackDriveRefusedIsNotACleanRollback(t *testing.T) {
	f := script(t, "published.json")
	f.failAt[0] = errors.New("files.update answered 500")

	rolled, warns := Rollback(context.Background(), f, testDocID)

	if rolled {
		t.Error("Rollback reported a clean rollback on a trash Drive refused")
	}
	if !strings.Contains(strings.Join(warns, " "), "answered 500") {
		t.Errorf("warnings = %v, want one carrying what Drive said", warns)
	}
}

// Which note is worth rolling back for is the caller's decision, not this
// package's: cmd/gdoc re-reads the note after the upload and refuses a block
// that has appeared, bytes that have changed under the render, a note it could
// not read again, and one whose front matter no longer parses. Those four
// refusals are pinned by cmd/gdoc's own
// TestPublishRollsBackWhenTheNoteCannotBePaired, and no test here can reach
// them: this package never sees the note.
//
// What this package owns is the half below. A caller that has to roll back
// needs an id off the report Run handed it, and the rollback made from that id
// has to end on the read that confirms the trash rather than on the PATCH.
func TestTheReportCarriesTheIDARollbackIsMadeFrom(t *testing.T) {
	f := script(t, "published.json")

	rep, err := Run(context.Background(), f, options())
	if err != nil {
		t.Fatal(err)
	}
	if rep.DocumentID == "" {
		t.Fatal("the report carried no id, so a caller that has to roll back has nothing to trash")
	}

	rolled, warns := Rollback(context.Background(), f, rep.DocumentID)

	if !rolled {
		t.Errorf("the document this run made was not taken back: %v", warns)
	}
	if f.calls[len(f.calls)-1].url != drive.TrashedURL(testDocID) {
		t.Errorf("the last request was %q, and a rollback ends on the read that confirms it", f.calls[len(f.calls)-1].url)
	}
}

// publish marks its own cover, from index 1 to the end of the one contents
// list, so a restyle later walks past it. The request is the prelude
// package's, because the guard and the restyle read that one shape.
func TestAPublishMarksItsCoverToTheEndOfTheContentsList(t *testing.T) {
	// Arrange
	f := script(t, "published-contents.json")

	// Act
	rep, err := Run(context.Background(), f, options())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	var batch []call
	for _, c := range f.calls {
		if strings.HasSuffix(c.url, ":batchUpdate") {
			batch = append(batch, c)
		}
	}
	if len(batch) != 1 {
		t.Fatalf("the run sent %d batches, want the one marker", len(batch))
	}
	want := `{"requests":[{"createNamedRange":{"name":"gdoc:house-published","range":{"endIndex":41,"startIndex":1}}}]}`
	if string(batch[0].body) != want {
		t.Errorf("the marker batch = %s, want %s", batch[0].body, want)
	}
	if !rep.Marked {
		t.Errorf("Marked is false on a marker that read back: %v", rep.Warnings)
	}
}

// No contents list is no boundary, and a marker ending anywhere else would be
// a guess about where the note's body starts. So nothing is written, and the
// warning says what a restyle will then do to the cover.
func TestADocumentWithNoContentsListIsNotMarked(t *testing.T) {
	// Arrange
	f := script(t, "published.json")

	// Act
	rep, err := Run(context.Background(), f, options())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if f.sentBatch() {
		t.Error("the run sent a marker on a document with no contents list")
	}
	if rep.Marked {
		t.Error("Marked is true on a run that wrote no marker")
	}
	if !rep.Verified {
		t.Errorf("an unmarked document is still the document: Verified = false, %v", rep.Warnings)
	}
	if !strings.Contains(strings.Join(rep.Warnings, " "), "contents list") {
		t.Errorf("warnings = %v, want one naming the missing contents list", rep.Warnings)
	}
}

// Nothing trusts a success. A marker batch Docs answered and the read-back does
// not carry is a document with no marker, and the report says so.
func TestAMarkerTheReadBackDoesNotCarryIsNotMarked(t *testing.T) {
	// Arrange
	f := script(t, "published-contents.json")
	f.marked = fixture(t, "published-contents.json")

	// Act
	rep, err := Run(context.Background(), f, options())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if rep.Marked {
		t.Error("Marked is true on a marker the read-back does not carry")
	}
	if !strings.Contains(strings.Join(rep.Warnings, " "), "gdoc:house-published") {
		t.Errorf("warnings = %v, want one naming the marker", rep.Warnings)
	}
}

// A marker batch that failed leaves the document there and correct, only
// unmarked. It is a warning, never a failed publish: the caller's next move is
// pairing the note, and an error would lose the id.
func TestAMarkerBatchThatFailedIsAWarningNotAFailure(t *testing.T) {
	// Arrange: the fourth call is the marker batch.
	f := script(t, "published-contents.json")
	f.failAt[3] = errors.New("503 backend error")

	// Act
	rep, err := Run(context.Background(), f, options())

	// Assert
	if err != nil {
		t.Fatalf("Run() = %v, want the document reported", err)
	}
	if rep.DocumentID != testDocID {
		t.Errorf("DocumentID = %q, want the document that was made", rep.DocumentID)
	}
	if rep.Marked {
		t.Error("Marked is true on a marker batch that failed")
	}
	if !strings.Contains(strings.Join(rep.Warnings, " "), "503") {
		t.Errorf("warnings = %v, want one carrying what went wrong", rep.Warnings)
	}
}

// Two contents lists are two candidate boundaries, and picking one is a guess
// about where gdoc's own cover ends. So there is no boundary, and no marker.
func TestTwoContentsListsGiveNoCoverEnd(t *testing.T) {
	// Arrange
	one := docs.Tab{Body: []docs.Block{{TOC: &docs.TOC{StartIndex: 40, EndIndex: 90}}}}
	two := docs.Tab{Body: []docs.Block{
		{TOC: &docs.TOC{StartIndex: 40, EndIndex: 90}},
		{TOC: &docs.TOC{StartIndex: 200, EndIndex: 260}},
	}}

	// Act
	end, ok := coverEnd(one)
	_, twoOK := coverEnd(two)

	// Assert
	if !ok || end != 90 {
		t.Errorf("one contents list ends the cover at its end: got %d, %v", end, ok)
	}
	if twoOK {
		t.Error("two contents lists must give no cover end")
	}
}
