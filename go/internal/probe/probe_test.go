package probe

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/drive"
	"gdoc/internal/guard"
)

const (
	testFolderID = "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"
	testDocID    = "1PrObE0000000000000000000000000000000000"
)

// call is one request the fake was asked to make. The body is what the session
// would have marshalled, so a test reads the bytes the wire would carry.
type call struct {
	method string
	url    string
	body   json.RawMessage
}

// fakeSession answers each request by what it asked for rather than by its
// place in a script, so a test that changes how far the run gets does not have
// to re-count the answers behind it. The order is asserted from calls instead.
//
// Nothing here names net/http: the rooms that may are the ones the boundary
// test lists, and this is not one of them.
type fakeSession struct {
	calls   []call
	create  []byte        // the files.create answer
	read    []byte        // the documents.get answer
	trashed []byte        // the files.get?fields=trashed answer
	failAt  map[int]error // fail the nth call, counted from zero
}

func (f *fakeSession) do(method, rawURL string, body any, into any) error {
	raw := json.RawMessage(nil)
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		raw = b
	}
	f.calls = append(f.calls, call{method: method, url: rawURL, body: raw})
	if err := f.failAt[len(f.calls)-1]; err != nil {
		return err
	}
	return decodeInto(f.answer(method, rawURL), into)
}

// answer is the fake Google: one reply per call shape, and an empty object for
// the two writes whose answer this package reads nothing out of.
func (f *fakeSession) answer(method, rawURL string) []byte {
	switch {
	case method == "POST" && strings.HasPrefix(rawURL, "https://www.googleapis.com/drive/v3/files?"):
		return f.create
	case method == "GET" && rawURL == docs.URL(testDocID):
		return f.read
	case method == "GET" && strings.Contains(rawURL, "fields=trashed"):
		return f.trashed
	}
	return []byte(`{}`)
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

func (f *fakeSession) GetJSON(_ context.Context, rawURL string, into any) error {
	return f.do("GET", rawURL, nil, into)
}

func (f *fakeSession) PostJSON(_ context.Context, rawURL string, body any, into any) error {
	return f.do("POST", rawURL, body, into)
}

func (f *fakeSession) PatchJSON(_ context.Context, rawURL string, body any, into any) error {
	return f.do("PATCH", rawURL, body, into)
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// script is a Google that answers everything, with the document read taken from
// the named fixture.
func script(t *testing.T, read string) *fakeSession {
	t.Helper()
	return &fakeSession{
		create:  []byte(`{"id":"` + testDocID + `"}`),
		read:    fixture(t, read),
		trashed: fixture(t, "trashed.json"),
		failAt:  map[int]error{},
	}
}

func TestRunReportsEnrolledWhenTheWordCameBackAsASuggestion(t *testing.T) {
	f := script(t, "enrolled.json")

	rep, err := Run(context.Background(), f, testFolderID)

	if err != nil {
		t.Fatal(err)
	}
	if !rep.Enrolled {
		t.Error("Enrolled is false, and the read-back carried a suggested insertion")
	}
	if rep.ProbeDocumentID != testDocID {
		t.Errorf("ProbeDocumentID = %q, want the id the create answered with", rep.ProbeDocumentID)
	}
	if !rep.Trashed {
		t.Error("Trashed is false, and files.get answered trashed: true")
	}
	if len(rep.SuggestionIDs) != 1 || rep.SuggestionIDs[0] != "suggest.xyh4cb4emh7y" {
		t.Errorf("SuggestionIDs = %v, want the one id the fixture carries", rep.SuggestionIDs)
	}
	if len(rep.Warnings) != 0 {
		t.Errorf("a run where every step answered carried warnings: %v", rep.Warnings)
	}
}

func TestRunReportsNotEnrolledWhenTheWordCameBackAsPlainText(t *testing.T) {
	f := script(t, "plain.json")

	rep, err := Run(context.Background(), f, testFolderID)

	if err != nil {
		t.Fatal(err)
	}
	if rep.Enrolled {
		t.Error("Enrolled is true, and the word came back as ordinary text: the write was a direct edit")
	}
	if len(rep.SuggestionIDs) != 0 {
		t.Errorf("SuggestionIDs = %v, want none", rep.SuggestionIDs)
	}
	if !rep.Trashed {
		t.Error("Trashed is false, and files.get answered trashed: true")
	}
}

func TestAFailedSuggestWriteStillTrashesAndSaysSo(t *testing.T) {
	f := script(t, "enrolled.json")
	f.failAt[2] = errors.New("documents:batchUpdate answered 403: writeMode is not available")

	rep, err := Run(context.Background(), f, testFolderID)

	if err == nil {
		t.Fatal("a failed SUGGEST write came back as success")
	}
	if !strings.Contains(err.Error(), "writeMode is not available") {
		t.Errorf("the error lost what Google said: %v", err)
	}
	if rep.ProbeDocumentID != testDocID {
		t.Errorf("ProbeDocumentID = %q, and a document that was created is never lost from the report", rep.ProbeDocumentID)
	}
	if !rep.Trashed {
		t.Error("Trashed is false: the trash runs on every path after the create succeeded")
	}
	if got := methods(f); len(got) != 5 {
		t.Errorf("the fake saw %d requests %v, want five: the read is skipped and the trash still runs", len(got), got)
	}
}

func TestACreateWithNoIDIsAnErrorNamingTheFolder(t *testing.T) {
	f := script(t, "enrolled.json")
	f.create = []byte(`{}`)

	rep, err := Run(context.Background(), f, testFolderID)

	if err == nil {
		t.Fatal("a create that answered with no id came back as success")
	}
	if !strings.Contains(err.Error(), testFolderID) {
		t.Errorf("the error does not name the folder the create targeted: %v", err)
	}
	if !strings.Contains(err.Error(), "may") {
		t.Errorf("Drive answered, so the error must not claim the document was not created: %v", err)
	}
	if rep.ProbeDocumentID != "" {
		t.Errorf("ProbeDocumentID = %q, and there is no id to name", rep.ProbeDocumentID)
	}
	if len(f.calls) != 1 {
		t.Errorf("the fake saw %d requests, want one: nothing follows a create with no id", len(f.calls))
	}
}

func TestATrashThatDidNotHoldIsAWarningRatherThanSilence(t *testing.T) {
	f := script(t, "enrolled.json")
	f.failAt[4] = errors.New("files.update answered 500")

	rep, err := Run(context.Background(), f, testFolderID)

	if err != nil {
		t.Fatalf("the probe answered its own question, so a failed trash is not its error: %v", err)
	}
	if len(f.calls) < 5 || f.calls[4].url != drive.FileURL(testDocID) {
		t.Fatalf("call 4 is %v, and this test is about the PATCH that trashes", f.calls)
	}
	if !strings.Contains(strings.Join(rep.Warnings, " "), "the trash request failed") {
		t.Errorf("warnings = %v, want the failed-PATCH sentence carried through", rep.Warnings)
	}
	if !strings.Contains(strings.Join(rep.Warnings, " "), "may still be in the folder") {
		t.Errorf("warnings = %v, and a 500 is a trash Drive may have applied, so the warning says may", rep.Warnings)
	}
	if !rep.Enrolled {
		t.Error("Enrolled is false, and the read-back carried a suggested insertion")
	}
	if rep.Trashed {
		t.Error("Trashed is true, and the trash failed")
	}
	if len(rep.Warnings) == 0 || !strings.Contains(strings.Join(rep.Warnings, " "), testDocID) {
		t.Errorf("warnings = %v, want one naming the document left behind", rep.Warnings)
	}
}

// The confirming read is the one failure that knows nothing about where the
// document is: Drive took the PATCH and then could not be asked. So the warning
// may not say the document is in the folder. Saying so contradicts the sentence
// behind it in the same line, and sends somebody to delete a document that is
// almost certainly already trashed.
func TestATrashDriveCouldNotConfirmDoesNotClaimTheDocumentIsInTheFolder(t *testing.T) {
	f := script(t, "enrolled.json")
	f.failAt[5] = errors.New("files.get answered 503")

	rep, err := Run(context.Background(), f, testFolderID)

	if err != nil {
		t.Fatalf("the probe answered its own question, so a failed confirmation is not its error: %v", err)
	}
	// The index is asserted rather than assumed. One more request anywhere
	// before the trash moves the confirming read off 5, and every assertion
	// below is satisfied by the refused-PATCH case the test above already
	// drives, so nothing else here would notice.
	if len(f.calls) < 6 || f.calls[5].url != drive.TrashedURL(testDocID) {
		t.Fatalf("call 5 is %v, and this test is about the read that confirms the trash", f.calls)
	}
	if rep.Trashed {
		t.Error("Trashed is true, and the trash was never confirmed")
	}
	warns := strings.Join(rep.Warnings, " ")
	if !strings.Contains(warns, "could not be asked to confirm it") {
		t.Errorf("warnings = %v, want the unconfirmed-read sentence rather than one of the other two trash failures", rep.Warnings)
	}
	if !strings.Contains(warns, testDocID) {
		t.Errorf("warnings = %v, want one naming the document", rep.Warnings)
	}
	if !strings.Contains(warns, "may still be in the folder") {
		t.Errorf("warnings = %v, want one saying the document may still be there", rep.Warnings)
	}
	if strings.Contains(warns, "is still in the folder") {
		t.Errorf("warnings = %v, and an unconfirmed trash knows nothing about where the document is", rep.Warnings)
	}
}

func TestATrashDriveDidNotConfirmIsNotReportedAsTrashed(t *testing.T) {
	f := script(t, "enrolled.json")
	f.trashed = []byte(`{"trashed": false}`)

	rep, err := Run(context.Background(), f, testFolderID)

	if err != nil {
		t.Fatal(err)
	}
	if rep.Trashed {
		t.Error("Trashed is true, and the read-back said the document is not in the trash")
	}
	if len(rep.Warnings) == 0 {
		t.Error("a trash Drive did not confirm passed without a warning")
	}
}

func TestRunSendsTheSixRequestsInOrder(t *testing.T) {
	f := script(t, "enrolled.json")

	if _, err := Run(context.Background(), f, testFolderID); err != nil {
		t.Fatal(err)
	}

	want := []call{
		{method: "POST", url: "https://www.googleapis.com/drive/v3/files?fields=id&supportsAllDrives=true"},
		{method: "POST", url: "https://docs.googleapis.com/v1/documents/" + testDocID + ":batchUpdate"},
		{method: "POST", url: "https://docs.googleapis.com/v1/documents/" + testDocID + ":batchUpdate"},
		{method: "GET", url: docs.URL(testDocID)},
		{method: "PATCH", url: "https://www.googleapis.com/drive/v3/files/" + testDocID + "?supportsAllDrives=true"},
		{method: "GET", url: "https://www.googleapis.com/drive/v3/files/" + testDocID + "?fields=trashed&supportsAllDrives=true"},
	}
	if len(f.calls) != len(want) {
		t.Fatalf("the fake saw %d requests %v, want %d", len(f.calls), methods(f), len(want))
	}
	for i, w := range want {
		if f.calls[i].method != w.method || f.calls[i].url != w.url {
			t.Errorf("request %d was %s %s, want %s %s", i, f.calls[i].method, f.calls[i].url, w.method, w.url)
		}
	}
}

func TestTheCreateNamesExactlyTheFolderAndAsksForADocument(t *testing.T) {
	f := script(t, "enrolled.json")

	if _, err := Run(context.Background(), f, testFolderID); err != nil {
		t.Fatal(err)
	}

	var meta struct {
		Name     string   `json:"name"`
		MimeType string   `json:"mimeType"`
		Parents  []string `json:"parents"`
	}
	if err := json.Unmarshal(f.calls[0].body, &meta); err != nil {
		t.Fatal(err)
	}
	if len(meta.Parents) != 1 || meta.Parents[0] != testFolderID {
		t.Errorf("parents = %v, want exactly the folder the command was given", meta.Parents)
	}
	if meta.MimeType != "application/vnd.google-apps.document" {
		t.Errorf("mimeType = %q, want a Docs file", meta.MimeType)
	}
	if !strings.HasPrefix(meta.Name, "gdoc probe ") {
		t.Errorf("name = %q, want a name saying what left the document behind", meta.Name)
	}
}

func TestTheDirectInsertIsDirectAndTheSuggestInsertSaysSuggestExactly(t *testing.T) {
	f := script(t, "enrolled.json")

	if _, err := Run(context.Background(), f, testFolderID); err != nil {
		t.Fatal(err)
	}

	var direct map[string]json.RawMessage
	if err := json.Unmarshal(f.calls[1].body, &direct); err != nil {
		t.Fatal(err)
	}
	for name := range direct {
		if strings.EqualFold(name, "writeControl") {
			t.Errorf("the first insert carries %q; it is a direct edit of a document gdoc created, and a suggestion there would prove nothing", name)
		}
	}

	var suggest struct {
		WriteControl map[string]string `json:"writeControl"`
	}
	if err := json.Unmarshal(f.calls[2].body, &suggest); err != nil {
		t.Fatal(err)
	}
	if len(suggest.WriteControl) != 1 || suggest.WriteControl["writeMode"] != "SUGGEST" {
		t.Errorf("writeControl = %v, want exactly {\"writeMode\":\"SUGGEST\"}", suggest.WriteControl)
	}
}

func TestTheSuggestInsertLandsInsideTheSentenceTheDirectInsertWrote(t *testing.T) {
	f := script(t, "enrolled.json")

	if _, err := Run(context.Background(), f, testFolderID); err != nil {
		t.Fatal(err)
	}

	sentence := insertedText(t, f.calls[1].body)
	if strings.Contains(sentence, probeWord) {
		t.Fatalf("the sentence already contains the probe word, so the read-back cannot tell the two apart: %q", sentence)
	}
	var body struct {
		Requests []struct {
			InsertText struct {
				Location struct {
					Index int `json:"index"`
				} `json:"location"`
				Text string `json:"text"`
			} `json:"insertText"`
		} `json:"requests"`
	}
	if err := json.Unmarshal(f.calls[2].body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Requests) != 1 {
		t.Fatalf("the SUGGEST write carried %d requests, want one", len(body.Requests))
	}
	got := body.Requests[0].InsertText
	if got.Text != probeWord {
		t.Errorf("the suggested text is %q, want %q", got.Text, probeWord)
	}
	// The sentence starts at index 1, which is where the direct insert put it,
	// so an index inside it is 1 plus an offset into the sentence.
	if got.Location.Index <= 1 || got.Location.Index >= 1+len(sentence) {
		t.Errorf("the suggested insert is at index %d, which is outside the sentence the direct insert wrote", got.Location.Index)
	}
}

func insertedText(t *testing.T, raw []byte) string {
	t.Helper()
	var body struct {
		Requests []struct {
			InsertText struct {
				Text string `json:"text"`
			} `json:"insertText"`
		} `json:"requests"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Requests) != 1 {
		t.Fatalf("the direct write carried %d requests, want one", len(body.Requests))
	}
	return body.Requests[0].InsertText.Text
}

func methods(f *fakeSession) []string {
	out := make([]string, 0, len(f.calls))
	for _, c := range f.calls {
		out = append(out, c.method)
	}
	return out
}

// TestTheGuardCarriesEverySixOfTheProbesRequests is the check that costs
// nothing to run and everything to be missing. Every URL this package builds
// has to be one the policy a probe opens will carry: AllowCreateIn on the
// folder, and the probe document learned from the create the guard itself
// carried. A URL that is right for Drive and wrong for the guard fails inside
// the process, on Nail's machine, on the first real run.
//
// Judge takes net/url rather than net/http, so this stays a room the boundary
// test does not list.
func TestTheGuardCarriesEveryOneOfTheProbesRequests(t *testing.T) {
	f := script(t, "enrolled.json")
	if _, err := Run(context.Background(), f, testFolderID); err != nil {
		t.Fatal(err)
	}

	p := guard.NewPolicy()
	p.AllowCreateIn(testFolderID)
	for i, c := range f.calls {
		if i == 1 {
			// The transport learns the new id from the create's answer, which
			// is the second of the policy's two doors. Everything after the
			// create is judged with the document in the set at full level.
			p.Learn(testDocID)
		}
		u, err := url.Parse(c.url)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		if err := p.Judge(c.method, u, c.body); err != nil {
			t.Errorf("request %d, %s %s: %v", i, c.method, c.url, err)
		}
	}
}

// TestTheProbeDocumentIsNeverHandedIn states the other half of the same rule.
// The probe writes directly, which the guard carries only at full level, so a
// probe run against a document somebody passed on the command line would be
// gdoc making a direct edit to a document it was asked to suggest on.
func TestTheProbeDocumentIsNeverHandedIn(t *testing.T) {
	p := guard.NewPolicy()
	p.AllowCreateIn(testFolderID)
	p.AllowFile(testDocID, guard.LevelSuggest)

	u, err := url.Parse(batchURL(testDocID))
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(directInsert())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Judge("POST", u, body); err == nil {
		t.Fatal("the direct insert was carried on a handed-in document")
	}
}

// acceptedError is what the session hands back for a failure raised after the
// server accepted the request. The probe asks by behaviour, the way the three
// writer packages do, so the fake answers by behaviour too.
type acceptedError struct{ error }

func (acceptedError) Sent() bool { return true }

// A create Drive accepted whose answer could not be read is not a create that
// did not happen. The document is in the folder, gdoc has no id for it, so it
// can neither name it in probe_document_id nor trash it. Saying the document
// could not be created sends somebody to look for a failure while the litter
// sits in their Drive.
func TestACreateDriveAcceptedButCouldNotBeReadSaysSo(t *testing.T) {
	f := script(t, "enrolled.json")
	f.failAt = map[int]error{0: acceptedError{errors.New("the answer is not JSON")}}

	rep, err := Run(context.Background(), f, testFolderID)
	if err == nil {
		t.Fatal("a create whose answer could not be read is still a failed probe")
	}
	if !strings.Contains(err.Error(), testFolderID) {
		t.Errorf("the failure must name the folder the document may be in: %v", err)
	}
	if !strings.Contains(err.Error(), "may") {
		t.Errorf("the failure must not claim the document was not created: %v", err)
	}
	if rep.ProbeDocumentID != "" || rep.Trashed {
		t.Errorf("there is no id to name and nothing was trashed: %+v", rep)
	}
	if len(f.calls) != 1 {
		t.Errorf("nothing more is reachable without an id: %v", f.calls)
	}
}
