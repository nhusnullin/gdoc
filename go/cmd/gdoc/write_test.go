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
//
// before runs just as this answer is handed over. It is how a test stands where
// something else happens in the middle of a run: an edit landing in the paired
// note while the binary is out on the network, for instance.
type answer struct {
	method string
	match  string
	json   string
	bytes  []byte
	err    error
	once   bool
	used   bool
	before func()
}

// fakeWire stands in for the wire, as read_test's fakeSession does, with the
// two write verbs the M3 commands need. Naming net/http here would put cmd/gdoc
// in the boundary test's import allowlist for the sake of a stub.
type fakeWire struct {
	answers  []*answer
	calls    []wireCall
	warnings []string
	policy   *guard.Policy
	// bytesCtx is the context the last GetBytes was made on. The docx export
	// is the one read that happens after a wait, so it is the one a test asks
	// what it was bounded by.
	bytesCtx context.Context
}

func (f *fakeWire) find(method, rawURL string) (*answer, error) {
	for _, a := range f.answers {
		if a.used || a.method != method || !strings.Contains(rawURL, a.match) {
			continue
		}
		if a.once {
			a.used = true
		}
		if a.before != nil {
			a.before()
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

func (f *fakeWire) GetBytes(ctx context.Context, rawURL string, _ int64) ([]byte, error) {
	f.calls = append(f.calls, wireCall{Method: "GET", URL: rawURL})
	f.bytesCtx = ctx
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

// TestProposeRefusesABadProposalBeforeTheProbe is the shape rule at the door.
// Every proposal is in hand before anything leaves the machine, so an entry the
// run would refuse in the middle is refused now: a third proposal turned down
// after the first two have landed is a run that half happened in somebody's
// document, and it has created and trashed a probe document on the way.
func TestProposeRefusesABadProposalBeforeTheProbe(t *testing.T) {
	for _, tc := range []struct{ name, file, says string }{
		{"the second carries no reason",
			`[{"quoted":"a","replacement":"b","why":"c"},{"quoted":"d","replacement":"e"}]`,
			"proposals[1]"},
		{"the reason carries markdown",
			`[{"quoted":"a","replacement":"b","why":"use **six months**"}]`,
			"markdown"},
		{"the replacement is empty",
			`[{"quoted":"a","replacement":"","why":"c"}]`,
			"replaces words with words"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
			from := tempFile(t, "proposals.json", tc.file)

			got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
			if code == 0 || got["ok"] != false {
				t.Fatalf("a proposal that cannot be written must stop the run: %v (exit %d)", got, code)
			}
			if msg, _ := got["error"].(string); !strings.Contains(msg, tc.says) {
				t.Errorf("the error must say %q: %q", tc.says, msg)
			}
			if len(f.calls) != 0 {
				t.Errorf("nothing may reach Google, the probe document included: %v", f.calls)
			}
		})
	}
}

// TestProposeReportsEveryProposalWhenOneOfThemCannotBeSent keeps the report one
// entry per proposal in the file. The run stops at the first one that cannot be
// sent, and a shortened list makes the skill match the envelope back against the
// file it wrote to work out what happened to the rest.
func TestProposeReportsEveryProposalWhenOneOfThemCannotBeSent(t *testing.T) {
	// The fourth inline answer is the second proposal's own read, appended after
	// the preview so the preview read cannot spend it: the preview URL carries
	// the inline match string too, and find takes the first unused answer that
	// matches. With that read answered, the second proposal is refused by
	// FindSpan, which is the stop this test is about, rather than by a fake wire
	// that ran out of answers.
	answers := append(proposeAnswers(t, true),
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-after.json"), once: true})
	stubWire(t, &fakeWire{answers: answers})
	// The second quote is not in the document, so Apply refuses it after the
	// first one has landed.
	from := tempFile(t, "proposals.json",
		`[{"quoted":"reviewed annually","replacement":"reviewed every six months","why":"the policy says twice a year"},`+
			`{"quoted":"reviewed monthly","replacement":"reviewed weekly","why":"the policy says twice a year"}]`)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a proposal that could not be placed must fail the run: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	list, _ := data["proposals"].([]any)
	if len(list) != 2 {
		t.Fatalf("proposals = %v, want one entry per proposal in the file", data["proposals"])
	}
	first, _ := list[0].(map[string]any)
	if first["sent"] != true {
		t.Errorf("the first proposal landed and reports %v", first["sent"])
	}
	second, _ := list[1].(map[string]any)
	if second["sent"] != false {
		t.Errorf("the second proposal never left and reports %v", second["sent"])
	}
	if second["quoted"] != "reviewed monthly" {
		t.Errorf("the second entry = %v, want the proposal that could not be placed", second)
	}
	// The error has to be the placement refusal naming the words, not whatever
	// else stopped the run: a test that asserts only the report shape passes
	// just as happily when the second proposal never got as far as FindSpan.
	if msg, _ := got["error"].(string); !strings.Contains(msg, `"reviewed monthly"`) || !strings.Contains(msg, "was not found in the document") {
		t.Errorf("error = %q, want the refusal naming the quote that could not be placed", msg)
	}
}

// TestProposeSaysSoWhenTheNoteCannotRememberAProposal is the loss the note
// exists to prevent, said out loud. A change that landed and came back with no
// suggestion id cannot be written into the front matter, and withdraw reads
// nothing else, so gdoc will refuse to take back its own work for ever. Silence
// there reads as a verification gap rather than as a permission thrown away.
func TestProposeSaysSoWhenTheNoteCannotRememberAProposal(t *testing.T) {
	answers := proposeAnswers(t, true)
	// The batch answers with no comment id, so nothing can be recorded.
	for _, a := range answers {
		if strings.Contains(a.match, proposeDocID+":batchUpdate") {
			a.json = `{"documentId":"` + proposeDocID + `","commentUpdateState":"ALL_SAVED"}`
		}
	}
	stubWire(t, &fakeWire{answers: answers})
	from := tempFile(t, "proposals.json", oneProposal)
	note := copyFixture(t, "propose-note.md")
	before := readFixture(t, "propose-note.md")

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID, "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("a change that landed is not a failed run: %v (exit %d)", got, code)
	}
	warns := warningsOf(t, got)
	if !hasWarning(warns, "cannot withdraw it later") {
		t.Errorf("warnings = %v, and one must say the proposal cannot be withdrawn", got["warnings"])
	}
	// The comment id is the only one of the two the batch itself answers with,
	// and it is the one missing here. Naming the read-back instead would send
	// somebody to look at their network over an answer Docs gave in full.
	if !hasWarning(warns, "the write came back without a comment id") {
		t.Errorf("warnings = %v, and one must name the route the missing id would have come from", got["warnings"])
	}
	if hasWarning(warns, "read-back could not confirm") {
		t.Errorf("warnings = %v, and the read-back read its suggestion id back fine", got["warnings"])
	}
	data := dataOf(t, got)
	if _, has := data["files_changed"]; has {
		t.Errorf("files_changed = %v, and nothing was added to the note", data["files_changed"])
	}
	after, err := os.ReadFile(note)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Errorf("the note was rewritten with nothing to add:\n%s", after)
	}
}

// TestProposeWritesTheNoteAsItStandsWhenTheRunFinishes is the read-again rule.
// The note is read at the start to check the pairing, and the run then spends
// seconds to tens of seconds on the network: the probe, a read and a write per
// proposal, three read-backs each. These notes live in a synced vault, so an
// edit can land in that window, and writing the bytes the run started with would
// throw it away.
func TestProposeWritesTheNoteAsItStandsWhenTheRunFinishes(t *testing.T) {
	note := copyFixture(t, "propose-note.md")
	answers := proposeAnswers(t, true)
	// The export is the last request of the run, so this is the closest a test
	// can stand to somebody saving the file a moment before the note is written.
	for _, a := range answers {
		if strings.Contains(a.match, "/export?") {
			a.before = func() {
				src, err := os.ReadFile(note)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(note, append(src, []byte("\nA line somebody added while the run was out.\n")...), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	stubWire(t, &fakeWire{answers: answers})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID, "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("propose: %v (exit %d)", got, code)
	}
	after, err := os.ReadFile(note)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "A line somebody added while the run was out.") {
		t.Errorf("the edit made during the run was overwritten:\n%s", after)
	}
	if !strings.Contains(string(after), "suggest.abc") {
		t.Errorf("the proposal was not recorded:\n%s", after)
	}
}

// The proposals file is a hand-written input, so a key gdoc does not understand
// is refused by name rather than dropped. quoted, replacement and why are caught
// downstream by Check, because their empty values are refused; assignee is
// optional, so a misspelling of it landed a comment with nobody assigned, said
// verified: true, and warned about nothing.
func TestProposeRefusesAnUnknownKeyInTheProposalsFile(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
	from := tempFile(t, "proposals.json",
		`[{"quoted":"a","replacement":"b","why":"c","assigned_to":"nail@altery.com"}]`)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a key gdoc does not understand must stop the run: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "assigned_to") {
		t.Errorf("the error must name the key: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach Google, the probe document included: %v", f.calls)
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
		{method: "POST", match: withdrawDocID + ":batchUpdate", json: readFixture(t, "withdraw-rejected.json")},
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
	ids, _ := data["rejected_suggestion_ids"].([]any)
	if len(ids) != 1 || ids[0] != "suggest.abc" {
		t.Errorf("rejected_suggestion_ids = %v", data["rejected_suggestion_ids"])
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

// TestWithdrawWritesTheNoteTheVaultHasNow is the same rule propose's record
// holds, on the other write. Between the pairing check and the note being
// written sit two whole-document reads and a batchUpdate, which is seconds on
// the network, and these notes live in a synced vault. Writing back the bytes
// the run started with would throw away whatever landed in that window: the
// author's own prose, and a proposal another run recorded into the gdoc: block.
func TestWithdrawWritesTheNoteTheVaultHasNow(t *testing.T) {
	note := copyFixture(t, "withdraw-note.md")
	// The sync lands while the write is in flight.
	landed := func() {
		src, err := os.ReadFile(note)
		if err != nil {
			t.Fatal(err)
		}
		next := strings.Replace(string(src),
			"      quoted: the operations team\n",
			"      quoted: the operations team\n"+
				"    - id: suggest.landed\n      comment_id: AAAE\n      at: 2026-09-07T10:07:00Z\n      quoted: annually\n", 1)
		next = strings.Replace(next, "# Scope", "# Scope and owner", 1)
		if err := os.WriteFile(note, []byte(next), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	stubWire(t, &fakeWire{answers: []*answer{
		{method: "GET", match: withdrawDocID + "?includeTabsContent", json: readFixture(t, "withdraw-pending.json"), once: true},
		{method: "POST", match: withdrawDocID + ":batchUpdate", json: readFixture(t, "withdraw-rejected.json"), before: landed},
		{method: "GET", match: withdrawDocID + "?includeTabsContent", json: readFixture(t, "withdraw-gone.json")},
	}})

	got, code := runJSON(t, "withdraw", withdrawDocID, "suggest.abc", "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("withdraw: %v (exit %d)", got, code)
	}
	after, err := os.ReadFile(note)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "suggest.abc") {
		t.Errorf("the withdrawn proposal is still recorded:\n%s", after)
	}
	if !strings.Contains(string(after), "suggest.landed") {
		t.Errorf("the proposal that landed while the run was on the network was thrown away:\n%s", after)
	}
	if !strings.Contains(string(after), "# Scope and owner") {
		t.Errorf("the author's edit was thrown away:\n%s", after)
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
