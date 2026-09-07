package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/guard"
)

const (
	proposeDocID  = "1PrOpOsE000000000000000000000000000000000"
	withdrawDocID = "1WiThDrAw00000000000000000000000000000000"
	probeDocID    = "1PrObE0000000000000000000000000000000000"
	testFolderID  = "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"
)

// wireCall is one request the command made, as the fake saw it. The tests below
// read this rather than the session's internals: what a command sends is the
// half of the contract this package owns.
type wireCall struct {
	Method string
	URL    string
	Body   []byte
}

// answer is one canned reply, matched on the method and a substring of the URL.
//
// once is what makes an order testable. propose reads the document before the
// write and again after it, through the same URL, so the fixture that comes
// back first has to be spent before the second one is reached.
type answer struct {
	method string
	match  string
	json   string
	bytes  []byte
	err    error
	once   bool
	used   bool
}

// fakeWire stands in for the wire, as read_test's fakeSession does, with the
// two write verbs the M3 commands need. Naming net/http here would put cmd/gdoc
// in the boundary test's import allowlist for the sake of a stub.
type fakeWire struct {
	answers  []*answer
	calls    []wireCall
	warnings []string
	policy   *guard.Policy
}

func (f *fakeWire) find(method, rawURL string) (*answer, error) {
	for _, a := range f.answers {
		if a.used || a.method != method || !strings.Contains(rawURL, a.match) {
			continue
		}
		if a.once {
			a.used = true
		}
		return a, nil
	}
	return nil, fmt.Errorf("the fake wire has no answer for %s %s", method, rawURL)
}

func (f *fakeWire) GetJSON(_ context.Context, rawURL string, into any) error {
	f.calls = append(f.calls, wireCall{Method: "GET", URL: rawURL})
	a, err := f.find("GET", rawURL)
	if err != nil {
		return err
	}
	if a.err != nil {
		return a.err
	}
	return json.Unmarshal([]byte(a.json), into)
}

func (f *fakeWire) GetBytes(_ context.Context, rawURL string, _ int64) ([]byte, error) {
	f.calls = append(f.calls, wireCall{Method: "GET", URL: rawURL})
	a, err := f.find("GET", rawURL)
	if err != nil {
		return nil, err
	}
	if a.err != nil {
		return nil, a.err
	}
	return a.bytes, nil
}

func (f *fakeWire) PostJSON(ctx context.Context, rawURL string, body any, into any) error {
	return f.write(ctx, "POST", rawURL, body, into)
}

func (f *fakeWire) PatchJSON(ctx context.Context, rawURL string, body any, into any) error {
	return f.write(ctx, "PATCH", rawURL, body, into)
}

func (f *fakeWire) write(_ context.Context, method, rawURL string, body any, into any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	f.calls = append(f.calls, wireCall{Method: method, URL: rawURL, Body: raw})
	a, err := f.find(method, rawURL)
	if err != nil {
		return err
	}
	if a.err != nil {
		return a.err
	}
	if into == nil || a.json == "" {
		return nil
	}
	return json.Unmarshal([]byte(a.json), into)
}

func (f *fakeWire) Warnings() []string { return f.warnings }

// stubWire stands in for gapi.Open and keeps the policy the command opened, so
// a test can ask what the run was allowed to reach.
func stubWire(t *testing.T, f *fakeWire) *fakeWire {
	t.Helper()
	old := openSession
	openSession = func(p *guard.Policy) (session, error) {
		f.policy = p
		return f, nil
	}
	t.Cleanup(func() { openSession = old })
	return f
}

// writes is every request the fake saw that was not a read. A test that says
// "nothing was sent" means this list is empty.
func (f *fakeWire) writes() []wireCall {
	var out []wireCall
	for _, c := range f.calls {
		if c.Method != "GET" {
			out = append(out, c)
		}
	}
	return out
}

func tempFile(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// copyFixture puts a note in a temp directory, because the commands that take
// --md rewrite the file they are given.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	return tempFile(t, name, readFixture(t, name))
}

// probeAnswers is the six-step probe, answered from the enrolled fixture.
func probeAnswers(t *testing.T, inline string) []*answer {
	t.Helper()
	return []*answer{
		{method: "POST", match: "files?fields=id", json: `{"id":"` + probeDocID + `"}`},
		{method: "POST", match: probeDocID + ":batchUpdate", json: `{}`},
		{method: "GET", match: probeDocID + "?includeTabsContent", json: readFixture(t, inline)},
		{method: "PATCH", match: "files/" + probeDocID, json: `{}`},
		{method: "GET", match: "fields=trashed", json: `{"trashed":true}`},
	}
}

// exportWithComment is the docx a verified proposal is confirmed against: the
// comment gdoc wrote, attached to the words it wrote.
func exportWithComment(t *testing.T, anchored bool) []byte {
	t.Helper()
	const commentsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:comment w:id="0" w:author="Nail Khusnullin" w:date="2026-09-07T09:00:00Z">
    <w:p><w:r><w:t>🤖 the policy says twice a year</w:t></w:r></w:p>
  </w:comment>
</w:comments>`
	const anchoredXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:t xml:space="preserve">The supplier register is </w:t></w:r>
      <w:commentRangeStart w:id="0"/>
      <w:r><w:t>reviewed every six months</w:t></w:r>
      <w:commentRangeEnd w:id="0"/>
      <w:r><w:commentReference w:id="0"/></w:r>
    </w:p>
  </w:body>
</w:document>`
	const bareXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body><w:p><w:r><w:t>The supplier register is reviewed annually.</w:t></w:r></w:p></w:body>
</w:document>`

	parts := map[string]string{"word/document.xml": anchoredXML, "word/comments.xml": commentsXML}
	if !anchored {
		parts = map[string]string{"word/document.xml": bareXML}
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// proposeAnswers is the probe followed by one proposal: the read before, the
// batch, and the three read-backs the verification is made of.
func proposeAnswers(t *testing.T, anchored bool) []*answer {
	t.Helper()
	out := probeAnswers(t, "probe-enrolled.json")
	return append(out,
		// Three reads of the inline view, in order: the command's own, which is
		// where the tab count comes from, the one Apply makes before it writes,
		// and the read-back. Same URL, so each is spent before the next is
		// reached, and the preview below is a different URL again.
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-before.json"), once: true},
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-before.json"), once: true},
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-after.json"), once: true},
		&answer{method: "POST", match: proposeDocID + ":batchUpdate", json: readFixture(t, "propose-batch.json")},
		&answer{method: "GET", match: "PREVIEW_WITHOUT_SUGGESTIONS", json: readFixture(t, "propose-preview.json")},
		&answer{method: "GET", match: "/export?", bytes: exportWithComment(t, anchored)},
	)
}

const oneProposal = `[{"quoted":"reviewed annually","replacement":"reviewed every six months",` +
	`"why":"the policy says twice a year"}]`

// Every command prints exactly one JSON object, and runJSON is that assertion.
// The probe's verdict is the fixture's, not a guess: enrolled means a run in
// the read-back carried a suggestion id.
func TestProbeReportsTheVerdictAndTrashesItsDocument(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: probeAnswers(t, "probe-enrolled.json")})

	got, code := runJSON(t, "probe", "--folder", "https://drive.google.com/drive/folders/"+testFolderID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("probe: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if data["enrolled"] != true {
		t.Errorf("enrolled = %v, and the fixture carries a suggested insertion", data["enrolled"])
	}
	if data["probe_document_id"] != probeDocID {
		t.Errorf("probe_document_id = %v", data["probe_document_id"])
	}
	if data["trashed"] != true {
		t.Errorf("trashed = %v: a document gdoc created is litter until it says otherwise", data["trashed"])
	}
	ids, _ := data["suggestion_ids"].([]any)
	if len(ids) != 1 || ids[0] != "suggest.xyh4cb4emh7y" {
		t.Errorf("suggestion_ids = %v", data["suggestion_ids"])
	}
	if f.policy == nil {
		t.Fatal("the command opened no policy")
	}
}

// The probe folder is the one door a create has. A document URL carries a
// valid-looking id, so passing it through would create nothing anywhere and
// come back as a 404 naming neither mistake.
func TestProbeRefusesADocumentURLAsItsFolder(t *testing.T) {
	f := stubWire(t, &fakeWire{})

	got, code := runJSON(t, "probe", "--folder", "https://docs.google.com/document/d/"+proposeDocID+"/edit")
	if code == 0 || got["ok"] != false {
		t.Fatalf("a document URL is not a folder: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "document, not a folder") {
		t.Errorf("the error must say which mistake was made: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may be sent for an argument that was refused: %v", f.calls)
	}
}

func TestProbeNeedsAFolder(t *testing.T) {
	stubWire(t, &fakeWire{})

	got, code := runJSON(t, "probe")
	if code == 0 || got["ok"] != false {
		t.Fatalf("probe without a folder must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "--folder") {
		t.Errorf("the error must name the flag that is missing: %q", msg)
	}
}

// A Docs thread renders what it is given literally, so markdown is refused
// before anything reaches Drive, and the refusal quotes what it found.
func TestReplyRefusesMarkdownBeforeAnyRequest(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	path := tempFile(t, "reply.txt", "🤖 See **the** register.")

	got, code := runJSON(t, "reply", proposeDocID, "AAAA1111", "--body-file", path)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a markdown body must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "markdown") || !strings.Contains(msg, "**") {
		t.Errorf("the error must name what it found: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach Drive for a body that was refused: %v", f.calls)
	}
}

// A reply that does not open with the robot is refused for the same reason: the
// prefix is the only record of who wrote it.
func TestReplyRefusesABodyWithoutTheRobot(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	path := tempFile(t, "reply.txt", "The 2026 register.")

	got, code := runJSON(t, "reply", proposeDocID, "AAAA1111", "--body-file", path)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a body without the prefix must be refused: %v (exit %d)", got, code)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach Drive: %v", f.calls)
	}
}

func TestReplyPostsTheBodyAndReadsTheThreadBack(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: []*answer{
		{method: "POST", match: "/comments/AAAA1111/replies", json: `{"id":"R1","createdTime":"2026-09-06T10:45:00Z","content":"🤖 The 2026 register."}`},
		{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
	}})
	path := tempFile(t, "reply.txt", "🤖 The 2026 register.\n")

	got, code := runJSON(t, "reply", fixtureDocID, "AAAA1111", "--body-file", path)
	if code != 0 || got["ok"] != true {
		t.Fatalf("reply: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if data["reply_id"] != "R1" || data["verified"] != true {
		t.Errorf("data = %v, want the reply id and a read-back that found it", data)
	}
	if data["comment_id"] != "AAAA1111" || data["document_id"] != fixtureDocID {
		t.Errorf("data = %v, want the thread and the document named back", data)
	}
	posts := f.writes()
	if len(posts) != 1 {
		t.Fatalf("one reply is one write: %v", posts)
	}
	// The file ends with a newline, as text files do, and Drive is sent the
	// words rather than the newline: the read-back compares what was sent.
	var body map[string]any
	if err := json.Unmarshal(posts[0].Body, &body); err != nil {
		t.Fatal(err)
	}
	if body["content"] != "🤖 The 2026 register." {
		t.Errorf("content = %q", body["content"])
	}
}

func TestReplyNeedsABodyFile(t *testing.T) {
	stubWire(t, &fakeWire{})

	got, code := runJSON(t, "reply", proposeDocID, "AAAA1111")
	if code == 0 || got["ok"] != false {
		t.Fatalf("reply without a body file must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "--body-file") {
		t.Errorf("the error must name the flag: %q", msg)
	}
}

func TestProposeWritesTheSuggestionAndVerifiesIt(t *testing.T) {
	stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("propose: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if probe, _ := data["probe"].(map[string]any); probe == nil || probe["enrolled"] != true {
		t.Fatalf("the probe's verdict rides with the run: %v", data["probe"])
	}
	list, _ := data["proposals"].([]any)
	if len(list) != 1 {
		t.Fatalf("proposals = %v", data["proposals"])
	}
	one, _ := list[0].(map[string]any)
	if one["verified"] != true || one["sent"] != true {
		t.Errorf("proposal = %v, want a sent and verified change", one)
	}
	if one["comment_id"] != "AAAC" || one["comment_update_state"] != "ALL_SAVED" {
		t.Errorf("proposal = %v, want the comment the batch made", one)
	}
	checks, _ := one["checks"].(map[string]any)
	if checks["suggestions_inline"] != true || checks["preview_without_suggestions"] != true || checks["docx_anchored"] != true {
		t.Errorf("checks = %v, want all three routes", checks)
	}
	if _, has := data["files_changed"]; has {
		t.Error("no --md was given, so no file may be named as changed")
	}
}

// The probe is the loudness the whole milestone rests on. A project Google no
// longer honours SUGGEST for must stop the run before anything is written into
// the document being reviewed.
func TestProposeSendsNothingWhenTheProbeSaysNotEnrolled(t *testing.T) {
	// The probe reads back plain text where the suggested word should be, which
	// is Google answering that SUGGEST was not honoured.
	answers := append(probeAnswers(t, "propose-before.json"),
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-before.json")})
	f := stubWire(t, &fakeWire{answers: answers})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a project that is not enrolled must stop the run: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "SUGGEST") {
		t.Errorf("the error must name what Google did not honour: %q", msg)
	}
	data := dataOf(t, got)
	list, _ := data["proposals"].([]any)
	if len(list) != 1 {
		t.Fatalf("every proposal is still reported: %v", data["proposals"])
	}
	if one, _ := list[0].(map[string]any); one["sent"] != false {
		t.Errorf("proposal = %v, want sent false", one)
	}
	for _, c := range f.writes() {
		if strings.Contains(c.URL, proposeDocID) {
			t.Errorf("nothing may be written into the document being reviewed: %v", c)
		}
	}
}

// A proposal that landed and could not be confirmed is still gdoc's, so its
// provenance is written down: without it withdraw would refuse to take back
// gdoc's own work.
func TestProposeRecordsProvenanceForAnAcceptedButUnverifiedProposal(t *testing.T) {
	stubWire(t, &fakeWire{answers: proposeAnswers(t, false)})
	from := tempFile(t, "proposals.json", oneProposal)
	note := copyFixture(t, "propose-note.md")
	before := readFixture(t, "propose-note.md")

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID, "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("a change that landed is not a failed run: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	list, _ := data["proposals"].([]any)
	one, _ := list[0].(map[string]any)
	if one["verified"] != false {
		t.Fatalf("the export carries no comment, so the third route did not hold: %v", one)
	}
	if checks, _ := one["checks"].(map[string]any); checks["docx_anchored"] != false {
		t.Errorf("checks = %v, want the docx route false", one["checks"])
	}
	if !hasWarning(warningsOf(t, got), "docx") && !hasWarning(warningsOf(t, got), "export") {
		t.Errorf("the route that did not hold must be named: %v", got["warnings"])
	}
	files, _ := data["files_changed"].([]any)
	if len(files) != 1 || files[0] != note {
		t.Fatalf("files_changed = %v, want the note", data["files_changed"])
	}

	after, err := os.ReadFile(note)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "suggest.abc") {
		t.Errorf("the note does not record the proposal:\n%s", after)
	}
	// Everything outside the gdoc: block is the author's, and it comes through
	// byte for byte.
	for _, line := range []string{"title: Supplier register policy", "author: Nail", "# Scope"} {
		if !strings.Contains(string(after), line) {
			t.Errorf("the author's %q is gone:\n%s", line, after)
		}
	}
	if string(after) == before {
		t.Error("the note was not written at all")
	}
}

// The note is the pairing, so a note paired with another document is refused
// before anything is written: recording this document's proposal in it would
// hand withdraw the wrong permission.
func TestProposeRefusesANoteThatNamesAnotherDocument(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
	from := tempFile(t, "proposals.json", oneProposal)
	note := copyFixture(t, "other-document.md")

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID, "--md", note)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a note paired elsewhere must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "paired") {
		t.Errorf("the error must say what is wrong with the note: %q", msg)
	}
	if len(f.writes()) != 0 {
		t.Errorf("nothing may be written before the note is checked: %v", f.writes())
	}
}

func TestProposeNeedsProposalsAndAFolder(t *testing.T) {
	stubWire(t, &fakeWire{})
	from := tempFile(t, "proposals.json", oneProposal)

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"propose", proposeDocID, "--folder", testFolderID}, "--from"},
		{[]string{"propose", proposeDocID, "--from", from}, "--folder"},
	} {
		got, code := runJSON(t, tc.args...)
		if code == 0 || got["ok"] != false {
			t.Fatalf("%v must fail: %v", tc.args, got)
		}
		if msg, _ := got["error"].(string); !strings.Contains(msg, tc.want) {
			t.Errorf("the error must name %s: %q", tc.want, msg)
		}
	}
}

func TestProposeRefusesAnEmptyProposalList(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	from := tempFile(t, "proposals.json", "[]")

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an empty list is nothing to propose: %v (exit %d)", got, code)
	}
	if len(f.calls) != 0 {
		t.Errorf("no probe document may be created for a run with nothing to do: %v", f.calls)
	}
}

// Provenance is the permission, so the note is not optional.
func TestWithdrawNeedsTheNote(t *testing.T) {
	f := stubWire(t, &fakeWire{})

	got, code := runJSON(t, "withdraw", withdrawDocID, "suggest.abc")
	if code == 0 || got["ok"] != false {
		t.Fatalf("withdraw without --md must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "--md") {
		t.Errorf("the error must name the flag: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may be sent: %v", f.calls)
	}
}

func TestWithdrawRefusesASuggestionTheNoteDoesNotRecord(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: []*answer{
		{method: "GET", match: withdrawDocID + "?includeTabsContent", json: readFixture(t, "withdraw-pending.json")},
	}})
	note := copyFixture(t, "withdraw-note.md")

	got, code := runJSON(t, "withdraw", withdrawDocID, "suggest.somebody-else", "--md", note)
	if code == 0 || got["ok"] != false {
		t.Fatalf("gdoc withdraws only what it proposed: %v (exit %d)", got, code)
	}
	if len(f.writes()) != 0 {
		t.Errorf("nothing may be written: %v", f.writes())
	}
}

func TestWithdrawRetractsAndForgetsTheProposal(t *testing.T) {
	stubWire(t, &fakeWire{answers: []*answer{
		{method: "GET", match: withdrawDocID + "?includeTabsContent", json: readFixture(t, "withdraw-pending.json"), once: true},
		{method: "POST", match: withdrawDocID + ":batchUpdate", json: readFixture(t, "withdraw-deleted.json")},
		{method: "GET", match: withdrawDocID + "?includeTabsContent", json: readFixture(t, "withdraw-gone.json")},
	}})
	note := copyFixture(t, "withdraw-note.md")

	got, code := runJSON(t, "withdraw", withdrawDocID, "suggest.abc", "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("withdraw: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if data["verified"] != true {
		t.Fatalf("the answer named the id and the read-back shows it gone: %v", data)
	}
	ids, _ := data["deleted_suggestion_ids"].([]any)
	if len(ids) != 1 || ids[0] != "suggest.abc" {
		t.Errorf("deleted_suggestion_ids = %v", data["deleted_suggestion_ids"])
	}
	files, _ := data["files_changed"].([]any)
	if len(files) != 1 || files[0] != note {
		t.Fatalf("files_changed = %v", data["files_changed"])
	}

	after, err := os.ReadFile(note)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "suggest.abc") {
		t.Errorf("the withdrawn proposal is still recorded:\n%s", after)
	}
	if !strings.Contains(string(after), "suggest.other") {
		t.Errorf("the other proposal was forgotten with it:\n%s", after)
	}
}

// The usage line is the whole of the help, so it has to name every command that
// exists: there is no help command to read instead.
func TestTheUsageLineNamesTheNineCommands(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, _ := runJSON(t, "--help")
	msg, _ := got["error"].(string)
	for _, command := range []string{
		"auth status", "auth login", "read", "comments", "suggestions",
		"probe", "reply", "propose", "withdraw",
	} {
		if !strings.Contains(msg, command) {
			t.Errorf("the usage line must name %q: %q", command, msg)
		}
	}
}
