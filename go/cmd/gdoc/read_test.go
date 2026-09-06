package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/guard"
)

const fixtureDocID = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"

// fakeSession stands in for the wire. The command tests hold the CLI's half of
// the contract: the arguments, the envelope and the local file write. What the
// session does with a request is internal/gapi's own test, over a fake
// RoundTripper, and naming net/http here would put cmd/gdoc in the boundary
// test's import allowlist for the sake of a stub.
//
// json and bytes answer on a substring of the URL, so a test says what Google
// returned and nothing about how the URL was built.
type fakeSession struct {
	json     map[string]string
	bytes    map[string][]byte
	err      error
	warnings []string
	urls     []string
	policy   *guard.Policy
}

func (f *fakeSession) GetJSON(_ context.Context, rawURL string, into any) error {
	f.urls = append(f.urls, rawURL)
	if f.err != nil {
		return f.err
	}
	for key, body := range f.json {
		if strings.Contains(rawURL, key) {
			return json.Unmarshal([]byte(body), into)
		}
	}
	return fmt.Errorf("the fake session has no answer for %s", rawURL)
}

func (f *fakeSession) GetBytes(_ context.Context, rawURL string, _ int64) ([]byte, error) {
	f.urls = append(f.urls, rawURL)
	if f.err != nil {
		return nil, f.err
	}
	for key, body := range f.bytes {
		if strings.Contains(rawURL, key) {
			return body, nil
		}
	}
	return nil, fmt.Errorf("the fake session has no answer for %s", rawURL)
}

func (f *fakeSession) Warnings() []string { return f.warnings }

// stubSession stands in for gapi.Open and keeps the policy the command opened,
// so a test can ask what the run was allowed to reach.
func stubSession(t *testing.T, f *fakeSession) *fakeSession {
	t.Helper()
	old := openSession
	openSession = func(p *guard.Policy) (session, error) {
		f.policy = p
		return f, nil
	}
	t.Cleanup(func() { openSession = old })
	return f
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// docsAndComments is the pair of answers every read command needs: the Docs
// tree and the Drive comment listing.
func docsAndComments(t *testing.T) *fakeSession {
	t.Helper()
	return &fakeSession{json: map[string]string{
		"docs.googleapis.com": readFixture(t, "single-tab.json"),
		"/comments?":          readFixture(t, "comments.json"),
	}}
}

// witnessExport is the docx a real export looks like: one comment still
// anchored to text and one that lost its range.
func witnessExport(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"word/document.xml": readFixture(t, "document.xml"),
		"word/comments.xml": readFixture(t, "comments.xml"),
	} {
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

// mustParse builds the URL a policy check is made on. net/url is not the wire:
// the boundary test's rule is about net/http, and nothing here dials.
func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func dataOf(t *testing.T, got map[string]any) map[string]any {
	t.Helper()
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	return data
}

func warningsOf(t *testing.T, got map[string]any) []string {
	t.Helper()
	raw, _ := got["warnings"].([]any)
	out := make([]string, 0, len(raw))
	for _, w := range raw {
		s, _ := w.(string)
		out = append(out, s)
	}
	return out
}

func hasWarning(ws []string, substr string) bool {
	for _, w := range ws {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}

// The text is what the AI reads, so it is a golden file: the same document
// gives the same bytes, and a change to the projection has to be a change to
// the file that states it.
func TestReadPrintsTheGoldenTextAndNoStructure(t *testing.T) {
	stubSession(t, docsAndComments(t))

	got, code := runJSON(t, "read", "https://docs.google.com/document/d/"+fixtureDocID+"/edit")
	if code != 0 || got["ok"] != true {
		t.Fatalf("read: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if data["document_id"] != fixtureDocID {
		t.Errorf("document_id = %v", data["document_id"])
	}
	if data["title"] != "Supplier register policy" {
		t.Errorf("title = %v", data["title"])
	}
	if data["revision_id"] != "ALm37BXsingleTab" {
		t.Errorf("revision_id = %v", data["revision_id"])
	}
	if data["tabs"] != float64(1) || data["multi_tab"] != false {
		t.Errorf("tabs = %v, multi_tab = %v", data["tabs"], data["multi_tab"])
	}
	if want := readFixture(t, "read.golden"); data["text"] != want {
		t.Errorf("text =\n%q\nwant\n%q", data["text"], want)
	}
	if _, has := data["structure"]; has {
		t.Error("the structure is a flag, not a default: it is bytes nobody asked for")
	}
}

func TestReadWithStructureCarriesTheTree(t *testing.T) {
	stubSession(t, docsAndComments(t))

	got, code := runJSON(t, "read", fixtureDocID, "--structure")
	if code != 0 || got["ok"] != true {
		t.Fatalf("read --structure: %v (exit %d)", got, code)
	}
	structure, ok := dataOf(t, got)["structure"].(map[string]any)
	if !ok {
		t.Fatalf("no structure object: %v", dataOf(t, got))
	}
	tabs, ok := structure["tabs"].([]any)
	if !ok || len(tabs) != 1 {
		t.Fatalf("structure.tabs: %v", structure["tabs"])
	}
	tab, _ := tabs[0].(map[string]any)
	blocks, ok := tab["blocks"].([]any)
	if !ok || len(blocks) == 0 {
		t.Fatalf("the tab carries no blocks: %v", tab)
	}
	first, _ := blocks[0].(map[string]any)
	para, ok := first["paragraph"].(map[string]any)
	if !ok {
		t.Fatalf("the first block is not a paragraph: %v", first)
	}
	if para["start_index"] != float64(1) {
		t.Errorf("start_index = %v, want the fixture's 1: a write needs the real index", para["start_index"])
	}
}

// The policy is opened with exactly the one document the command was given.
// Anything else is refused inside the process, before a request is built.
func TestAReadIsBoundedToTheOneDocumentItWasGiven(t *testing.T) {
	f := stubSession(t, docsAndComments(t))

	if _, code := runJSON(t, "read", fixtureDocID); code != 0 {
		t.Fatalf("read failed: exit %d", code)
	}
	if f.policy == nil {
		t.Fatal("the command opened no policy")
	}
	allowed := mustParse(t, "https://docs.googleapis.com/v1/documents/"+fixtureDocID)
	if err := f.policy.Judge("GET", allowed, nil); err != nil {
		t.Errorf("the document it was given must be reachable: %v", err)
	}
	other := mustParse(t, "https://docs.googleapis.com/v1/documents/9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz")
	if err := f.policy.Judge("GET", other, nil); err == nil {
		t.Error("a document nobody named must be refused")
	}
	edit := mustParse(t, "https://docs.googleapis.com/v1/documents/"+fixtureDocID+":batchUpdate")
	if err := f.policy.Judge("POST", edit, []byte(`{"requests":[]}`)); err == nil {
		t.Error("a handed-in document is read and suggest only: a direct edit must be refused")
	}
}

// The unplaced thread is a fact about one thread, reported in warnings, and the
// listing still comes back. The session's warnings, which are the policy's
// followed by its own, ride on the same envelope.
func TestCommentsCarriesTheWarningsAndReportsAnUnplacedThread(t *testing.T) {
	f := docsAndComments(t)
	f.warnings = []string{"guard refused: something the policy could not do quietly"}
	stubSession(t, f)

	got, code := runJSON(t, "comments", fixtureDocID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("comments: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	threads, ok := data["threads"].([]any)
	if !ok || len(threads) != 2 {
		t.Fatalf("want the fixture's two threads: %v", data["threads"])
	}
	placed, _ := threads[0].(map[string]any)
	if placed["marker"] != "ai?" {
		t.Errorf("marker = %v, want ai?", placed["marker"])
	}
	if r, ok := placed["range"].(map[string]any); !ok || r["start"] != float64(66) {
		t.Errorf("the placed thread must carry the Docs range: %v", placed["range"])
	}
	replies, ok := placed["replies"].([]any)
	if !ok || len(replies) != 1 {
		t.Fatalf("replies: %v", placed["replies"])
	}
	if reply, _ := replies[0].(map[string]any); reply["by_gdoc"] != true {
		t.Errorf("a reply opening with the robot is gdoc's: %v", replies[0])
	}
	unplaced, _ := threads[1].(map[string]any)
	if unplaced["range"] != nil {
		t.Errorf("a thread the Docs read did not place has no range: %v", unplaced["range"])
	}
	if _, has := unplaced["witness"]; has {
		t.Error("the witness is a flag: a run that did not ask for it must not carry one")
	}

	ws := warningsOf(t, got)
	if !hasWarning(ws, "guard refused") {
		t.Errorf("the session's warnings must reach the envelope: %v", ws)
	}
	if !hasWarning(ws, "BBBB2222") {
		t.Errorf("the unplaced thread must be named in warnings: %v", ws)
	}
	// base64url of {"v":1,"t":"2026-09-06T10:45:00Z","i":["AAAA1111"]}: the
	// newest instant across the comments and their replies, which is reply R2's,
	// and the thread it belongs to.
	if data["cursor"] != "eyJ2IjoxLCJ0IjoiMjAyNi0wOS0wNlQxMDo0NTowMFoiLCJpIjpbIkFBQUExMTExIl19" {
		t.Errorf("cursor = %v, want the newest instant across comments and replies, with the thread it came from", data["cursor"])
	}
}

// A cursor the binary cannot read is refused naming it. Reading from the
// beginning instead would answer a poll with every thread in the document, and
// the session would read that as news.
func TestCommentsRefusesACursorItCannotRead(t *testing.T) {
	stubSession(t, docsAndComments(t))

	got, code := runJSON(t, "comments", fixtureDocID, "--since", "not-a-cursor")
	if code == 0 || got["ok"] != false {
		t.Fatalf("a bad cursor must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "not-a-cursor") {
		t.Errorf("the error must quote the cursor: %q", msg)
	}
}

// A cursor the binary wrote is carried onto the request as startModifiedTime.
func TestCommentsCarriesTheCursorOntoTheListing(t *testing.T) {
	f := stubSession(t, docsAndComments(t))

	got, code := runJSON(t, "comments", fixtureDocID, "--since", "eyJ2IjoxLCJ0IjoiMjAyNi0wOS0wMVQwMDowMDowMFoifQ")
	if code != 0 || got["ok"] != true {
		t.Fatalf("comments --since: %v (exit %d)", got, code)
	}
	found := false
	for _, u := range f.urls {
		if strings.Contains(u, "startModifiedTime=2026-09-01T00%3A00%3A00Z") {
			found = true
		}
	}
	if !found {
		t.Errorf("the cursor must reach the listing: %v", f.urls)
	}
}

func TestCommentsWithWitnessSetsTheWitnessPerThread(t *testing.T) {
	f := docsAndComments(t)
	f.bytes = map[string][]byte{"/export?": witnessExport(t)}
	stubSession(t, f)

	got, code := runJSON(t, "comments", fixtureDocID, "--witness")
	if code != 0 || got["ok"] != true {
		t.Fatalf("comments --witness: %v (exit %d)", got, code)
	}
	threads, _ := dataOf(t, got)["threads"].([]any)
	if len(threads) != 2 {
		t.Fatalf("threads: %v", threads)
	}
	anchored, _ := threads[0].(map[string]any)
	if anchored["witness"] != "anchored" {
		t.Errorf("witness = %v, want anchored", anchored["witness"])
	}
	detached, _ := threads[1].(map[string]any)
	if detached["witness"] != "detached" {
		t.Errorf("witness = %v, want detached: the export carries the comment and no range", detached["witness"])
	}
}

// An export the witness could not read is a warning and every thread
// unmatched, not a failed listing: the threads are the answer, and the witness
// is a second read on top of them.
func TestAWitnessThatCouldNotBeReadIsAWarning(t *testing.T) {
	f := docsAndComments(t)
	f.bytes = map[string][]byte{"/export?": []byte("<html>sign in</html>")}
	stubSession(t, f)

	got, code := runJSON(t, "comments", fixtureDocID, "--witness")
	if code != 0 || got["ok"] != true {
		t.Fatalf("a witness that failed must not fail the listing: %v (exit %d)", got, code)
	}
	if !hasWarning(warningsOf(t, got), "docx") {
		t.Errorf("the failure must be reported: %v", got["warnings"])
	}
	threads, _ := dataOf(t, got)["threads"].([]any)
	first, _ := threads[0].(map[string]any)
	if first["witness"] != "unmatched" {
		t.Errorf("witness = %v, want unmatched", first["witness"])
	}
}

func TestSuggestionsListsWhatIsPending(t *testing.T) {
	stubSession(t, docsAndComments(t))

	got, code := runJSON(t, "suggestions", fixtureDocID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("suggestions: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	pending, ok := data["pending"].([]any)
	if !ok || len(pending) != 2 {
		t.Fatalf("the fixture carries one replacement, which is two pendings: %v", data["pending"])
	}
	first, _ := pending[0].(map[string]any)
	if first["id"] != "suggest.a1" || first["kind"] != "deletion" || first["section"] != "Scope" {
		t.Errorf("the deletion comes first, in its section: %v", first)
	}
	for _, absent := range []string{"gone_since_last_look", "files_changed"} {
		if _, has := data[absent]; has {
			t.Errorf("%s belongs to --md: a run with no file has nothing to compare against", absent)
		}
	}
}

// The snapshot is written into the file's gdoc: block and nowhere near the
// author's own lines.
func TestSuggestionsWithMDWritesTheSnapshotAndLeavesEveryOtherLineAlone(t *testing.T) {
	stubSession(t, docsAndComments(t))
	stubNow(t, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	path := copyToTemp(t, "paired.md")
	before := mustRead(t, path)

	got, code := runJSON(t, "suggestions", fixtureDocID, "--md", path)
	if code != 0 || got["ok"] != true {
		t.Fatalf("suggestions --md: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	gone, ok := data["gone_since_last_look"].([]any)
	if !ok || len(gone) != 1 {
		t.Fatalf("want the one id that left: %v", data["gone_since_last_look"])
	}
	left, _ := gone[0].(map[string]any)
	if left["id"] != "suggest.old" || left["text"] != "quarterly " {
		t.Errorf("the gone item must carry what it said last time: %v", left)
	}
	if left["seen_at"] != "2026-09-01T09:00:00Z" {
		t.Errorf("seen_at = %v, want the snapshot's own time", left["seen_at"])
	}
	files, ok := data["files_changed"].([]any)
	if !ok || len(files) != 1 || files[0] != path {
		t.Fatalf("files_changed must name the file it wrote: %v", data["files_changed"])
	}

	after := mustRead(t, path)
	if !strings.Contains(after, "at: 2026-09-06T12:00:00Z") {
		t.Errorf("the new snapshot time is missing:\n%s", after)
	}
	if !strings.Contains(after, "suggest.a1") || strings.Contains(after, "suggest.old") {
		t.Errorf("the snapshot must be what is pending now:\n%s", after)
	}
	for _, line := range []string{"title: Supplier register policy", "author: Nail",
		"# Scope", "The supplier register is reviewed annually."} {
		if !strings.Contains(after, line) {
			t.Errorf("%q left the file:\n%s", line, after)
		}
	}
	if outside(before) != outside(after) {
		t.Errorf("a line outside the gdoc: block moved:\n%q\nto\n%q", outside(before), outside(after))
	}
}

// Writing this document's observation into a file paired with another one is
// the wrong file, so it is refused and nothing is written.
func TestSuggestionsRefusesAFilePairedWithAnotherDocument(t *testing.T) {
	stubSession(t, docsAndComments(t))
	path := copyToTemp(t, "other-document.md")
	before := mustRead(t, path)

	got, code := runJSON(t, "suggestions", fixtureDocID, "--md", path)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a file paired with another document must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz") || !strings.Contains(msg, fixtureDocID) {
		t.Errorf("the error must name both documents: %q", msg)
	}
	if mustRead(t, path) != before {
		t.Error("the file must be untouched")
	}
}

// A read that failed writes nothing: half the review was invisible, and a
// snapshot taken then would report the other half as gone on the next run.
func TestSuggestionsLeavesTheFileAloneWhenTheReadFailed(t *testing.T) {
	f := docsAndComments(t)
	f.err = errors.New("the token was refused twice; run: gdoc auth login")
	stubSession(t, f)
	path := copyToTemp(t, "paired.md")
	before := mustRead(t, path)

	got, code := runJSON(t, "suggestions", fixtureDocID, "--md", path)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a failed read must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "refused twice") {
		t.Errorf("the error must say what went wrong: %q", msg)
	}
	if mustRead(t, path) != before {
		t.Error("the file must be untouched")
	}
}

// A file with no gdoc: block is not paired with anything, so there is nothing
// to compare against and nowhere to record the snapshot.
func TestSuggestionsRefusesAFileWithNoBlock(t *testing.T) {
	stubSession(t, docsAndComments(t))
	path := filepath.Join(t.TempDir(), "unpaired.md")
	if err := os.WriteFile(path, []byte("# Scope\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "suggestions", fixtureDocID, "--md", path)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an unpaired file must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, path) {
		t.Errorf("the error must name the file: %q", msg)
	}
}

func TestDocumentIDAcceptsWhatNailPastes(t *testing.T) {
	for _, arg := range []string{
		"https://docs.google.com/document/d/" + fixtureDocID + "/edit?tab=t.0",
		"https://drive.google.com/open?id=" + fixtureDocID,
		fixtureDocID,
		"  " + fixtureDocID + "  ",
	} {
		got, err := documentID(arg)
		if err != nil {
			t.Errorf("documentID(%q): %v", arg, err)
			continue
		}
		if got != fixtureDocID {
			t.Errorf("documentID(%q) = %q", arg, got)
		}
	}
}

func TestDocumentIDRefusesWhatIsNotOne(t *testing.T) {
	for _, arg := range []string{
		"", "short", "https://docs.google.com/document/d/tooshort/edit",
		"/Users/nail/notes/policy.md", "https://example.com/",
		"1w0Sresiz E9Kr810VZRJwX4JtDBF4OqNr",
	} {
		got, err := documentID(arg)
		if err == nil {
			t.Errorf("documentID(%q) = %q, and it is not a document id", arg, got)
			continue
		}
		if arg != "" && !strings.Contains(err.Error(), strings.TrimSpace(arg)) {
			t.Errorf("the refusal must quote the input: %v", err)
		}
	}
}

// Nothing is ignored. A command that accepts what it does not understand tells
// the caller it did something it did not.
func TestTheReadCommandsRefuseArgumentsTheyDoNotUnderstand(t *testing.T) {
	cases := []struct {
		args  []string
		names string
	}{
		{[]string{"read", fixtureDocID, "--verbose"}, "--verbose"},
		{[]string{"read", fixtureDocID, "--structure", "--structure"}, "--structure"},
		{[]string{"read", fixtureDocID, "--structure=yes"}, "--structure"},
		{[]string{"read", fixtureDocID, "extra"}, "extra"},
		{[]string{"read"}, "document"},
		{[]string{"comments", fixtureDocID, "--since"}, "--since"},
		{[]string{"comments", fixtureDocID, "--since", "A", "--since", "B"}, "--since"},
		{[]string{"comments", fixtureDocID, "--md", "x.md"}, "--md"},
		{[]string{"suggestions", fixtureDocID, "--md"}, "--md"},
		{[]string{"suggestions", fixtureDocID, "--witness"}, "--witness"},
	}
	for _, c := range cases {
		stubSession(t, docsAndComments(t))
		got, code := runJSON(t, c.args...)
		if code == 0 || got["ok"] != false {
			t.Errorf("%v must be refused: %v", c.args, got)
			continue
		}
		if msg, _ := got["error"].(string); !strings.Contains(msg, c.names) {
			t.Errorf("%v: the error must name %q, got %q", c.args, c.names, msg)
		}
	}
}

// A command that could not open a session says so, and reaches no wire.
func TestAReadThatCannotOpenASessionFails(t *testing.T) {
	old := openSession
	openSession = func(*guard.Policy) (session, error) {
		return nil, errors.New("no token: run gdoc auth login")
	}
	t.Cleanup(func() { openSession = old })

	got, code := runJSON(t, "read", fixtureDocID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a missing token must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "auth login") {
		t.Errorf("the error must say what to run: %q", msg)
	}
}

// stubNow fixes the snapshot's clock, so the written line is a value the test
// can state rather than one it has to parse back.
func stubNow(t *testing.T, at time.Time) {
	t.Helper()
	old := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = old })
}

func copyToTemp(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(readFixture(t, name)), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// outside is every line of the file that is not inside the gdoc: block. It is
// what must come through a write byte for byte.
func outside(src string) string {
	var out []string
	inBlock := false
	for _, line := range strings.Split(src, "\n") {
		switch {
		case line == "gdoc:":
			inBlock = true
		case inBlock && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")):
			// still inside the block
		default:
			inBlock = false
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// The projection's warnings reach the envelope. `read` prints a placeholder in
// place of content it cannot read, and a caller that gets the text without the
// warning believes it read the whole document.
func TestReadCarriesTheProjectionWarnings(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "internal", "docs", "testdata", "objects.json"))
	if err != nil {
		t.Fatal(err)
	}
	stubSession(t, &fakeSession{json: map[string]string{"docs.googleapis.com": string(raw)}})

	got, code := runJSON(t, "read", fixtureDocID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("read: %v (exit %d)", got, code)
	}
	ws := warningsOf(t, got)
	for _, want := range []string{"[image]", "[drawing]", "[equation]"} {
		if !hasWarning(ws, want) {
			t.Errorf("no warning names %s: %v", want, ws)
		}
	}
}

// A comment the Docs read returned with no range this binary could read is a
// warning on `read` too. The text marks ranges, so a comment with no range is
// one the text cannot show, and silence reads as a document with no such
// comment in it.
func TestReadNamesTheCommentsItCouldNotPlace(t *testing.T) {
	doc := `{"documentId":"` + fixtureDocID + `","title":"T","body":{"content":[
		{"startIndex":1,"endIndex":6,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":6,"textRun":{"content":"Text\n"}}]}}]},
		"comments":[{"id":"NORANGE","anchor":"kix.abc"}]}`
	stubSession(t, &fakeSession{json: map[string]string{"docs.googleapis.com": doc}})

	got, code := runJSON(t, "read", fixtureDocID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("read: %v (exit %d)", got, code)
	}
	if ws := warningsOf(t, got); !hasWarning(ws, "NORANGE") {
		t.Errorf("warnings = %v, want one naming the comment with no range", ws)
	}
}

// A flag given where a value belongs is refused naming both. Swallowing it read
// `--since --witness` as a cursor and failed naming the cursor, which is not
// the problem the caller has.
func TestAFlagIsNotSwallowedAsAnotherFlagsValue(t *testing.T) {
	got, code := runJSON(t, "comments", fixtureDocID, "--since", "--witness")
	if code == 0 || got["ok"] != false {
		t.Fatalf("comments: %v (exit %d), want a refusal", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "--since") || !strings.Contains(msg, "--witness") {
		t.Errorf("error = %q, want it to name both flags", msg)
	}
}

// An empty value is refused whichever way it was written. `--md=` was already
// refused; `--md ""` reached os.ReadFile and failed naming the wrong problem.
func TestAnEmptyFlagValueIsRefusedBothWaysItCanBeWritten(t *testing.T) {
	for _, args := range [][]string{
		{"suggestions", fixtureDocID, "--md", ""},
		{"suggestions", fixtureDocID, "--md="},
	} {
		got, code := runJSON(t, args...)
		if code == 0 || got["ok"] != false {
			t.Fatalf("%v: %v (exit %d), want a refusal", args, got, code)
		}
		if msg, _ := got["error"].(string); !strings.Contains(msg, "empty value") {
			t.Errorf("%v: error = %q, want it to name the empty value", args, msg)
		}
	}
}

// The note's mode is what it was. gdoc is not the only reader of the markdown,
// and a note somebody made group readable stays that way.
func TestTheSnapshotWriteKeepsTheNotesMode(t *testing.T) {
	stubSession(t, docsAndComments(t))
	stubNow(t, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	path := copyToTemp(t, "paired.md")
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "suggestions", fixtureDocID, "--md", path)
	if code != 0 || got["ok"] != true {
		t.Fatalf("suggestions: %v (exit %d)", got, code)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o640 {
		t.Errorf("mode = %v, want 0640", mode)
	}
	// And nothing was left beside it.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("the directory holds %d entries, want just the note", len(entries))
	}
}

// A suggestion the author has edited down to whitespace is still pending, so it
// belongs in the snapshot. List drops it from what the run prints, because it
// says nothing a reader can act on, and a snapshot built from that same list
// forgets it: when it is later accepted or rejected, the run that would report
// it gone has no record that it was ever there.
func TestTheSnapshotCarriesASuggestionEditedDownToWhitespace(t *testing.T) {
	stubSession(t, &fakeSession{json: map[string]string{
		"docs.googleapis.com": `{"documentId":"` + fixtureDocID + `","body":{"content":[
			{"startIndex":1,"endIndex":3,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
			 "elements":[{"startIndex":1,"endIndex":3,"textRun":{"content":" \n","suggestedInsertionIds":["suggest.thin"]}}]}}]}}`,
	}})
	stubNow(t, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	path := copyToTemp(t, "paired.md")

	got, code := runJSON(t, "suggestions", fixtureDocID, "--md", path)
	if code != 0 || got["ok"] != true {
		t.Fatalf("suggestions --md: %v (exit %d)", got, code)
	}
	if pending, ok := dataOf(t, got)["pending"].([]any); !ok || len(pending) != 0 {
		t.Fatalf("pending = %v, want nothing printed: the text says nothing a reader can act on", pending)
	}
	after := mustRead(t, path)
	if !strings.Contains(after, "suggest.thin") {
		t.Errorf("the snapshot must carry the id that is still pending:\n%s", after)
	}
}
