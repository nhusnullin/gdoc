package main

import (
	"strings"
	"testing"
)

// emptyDocument is a document with nothing in it worth protecting: no comment,
// nothing pending, no chip. It is written out here rather than kept as a
// fixture because what it is about is what it does not have.
const emptyDocument = `{
  "documentId": "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
  "title": "An empty policy",
  "revisionId": "ALm37BXempty",
  "tabs": [{"tabProperties": {"tabId": "t.0"},
            "documentTab": {"body": {"content": [
              {"startIndex": 1, "endIndex": 7,
               "paragraph": {"elements": [
                 {"startIndex": 1, "endIndex": 7, "textRun": {"content": "Hello\n"}}]}}]}}}]
}`

// restyleSession is the three answers a survey needs: the comment listing, the
// Docs read and the docx export.
func restyleSession(t *testing.T) *fakeSession {
	t.Helper()
	f := docsAndComments(t)
	f.bytes = map[string][]byte{"/export?": witnessExport(t)}
	return f
}

func TestRestyleSurveysWhatTheDocumentHolds(t *testing.T) {
	// Arrange
	stubSession(t, restyleSession(t))

	// Act
	got, code := runJSON(t, "restyle",
		"https://docs.google.com/document/d/"+fixtureDocID+"/edit", "--dry-run")

	// Assert
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --dry-run: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if data["document_id"] != fixtureDocID {
		t.Errorf("document_id = %v, want %s", data["document_id"], fixtureDocID)
	}
	if data["revision_id"] != "ALm37BXsingleTab" {
		t.Errorf("revision_id = %v, want ALm37BXsingleTab: M7b refuses a document that moved", data["revision_id"])
	}
	if data["title"] != "Supplier register policy" {
		t.Errorf("title = %v", data["title"])
	}
	if data["tabs"] != float64(1) {
		t.Errorf("tabs = %v, want 1", data["tabs"])
	}
	threads, ok := data["threads"].(map[string]any)
	if !ok {
		t.Fatalf("no threads object: %v", data)
	}
	if threads["open"] != float64(2) || threads["resolved"] != float64(0) {
		t.Errorf("threads: open = %v, resolved = %v, want 2 and 0", threads["open"], threads["resolved"])
	}
	witness, ok := threads["witness"].(map[string]any)
	if !ok {
		t.Fatalf("no witness object: %v", threads)
	}
	if witness["anchored"] != float64(1) || witness["detached"] != float64(1) {
		t.Errorf("witness = %v, want 1 anchored and 1 detached", witness)
	}
	if list, _ := threads["witnessed"].([]any); len(list) != 2 {
		t.Errorf("witnessed: got %d entries, want one per thread", len(list))
	}
	pending, _ := data["suggestions"].(map[string]any)
	if pending == nil || pending["pending"] == float64(0) {
		t.Errorf("suggestions = %v, want the fixture's pending suggestions counted", data["suggestions"])
	}
	if data["nothing_to_protect"] != false {
		t.Errorf("nothing_to_protect = %v, want false on a document with threads and suggestions in it",
			data["nothing_to_protect"])
	}
}

// The listing goes out before the Docs read, and the order is the same rule
// `comments` holds: a comment written between the two reads is in whichever ran
// second, and a comment the Docs read has not got to yet is one the next read
// places, while a comment listed after the document was read is one with no
// anchor in a document read before it existed.
func TestRestyleListsTheCommentsBeforeItReadsTheDocument(t *testing.T) {
	f := stubSession(t, restyleSession(t))

	if _, code := runJSON(t, "restyle", fixtureDocID, "--dry-run"); code != 0 {
		t.Fatalf("restyle failed: exit %d", code)
	}
	var order []string
	for _, u := range f.urls {
		switch {
		case strings.Contains(u, "/comments?"):
			order = append(order, "comments")
		case strings.Contains(u, "/export?"):
			order = append(order, "export")
		case strings.Contains(u, "docs.googleapis.com"):
			order = append(order, "document")
		}
	}
	if len(order) < 2 || order[0] != "comments" || order[1] != "document" {
		t.Errorf("read order = %v, want the comment listing before the Docs read", order)
	}
	if !contains(order, "export") {
		t.Error("the survey made no export, so no thread has a witness")
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// Nothing to protect is three counts at zero, and it is a fact rather than a
// recommendation. Whether the document is worth restyling is Nail's.
func TestRestyleSaysNothingToProtectOnAnEmptyDocument(t *testing.T) {
	stubSession(t, &fakeSession{
		json: map[string]string{
			"docs.googleapis.com": emptyDocument,
			"/comments?":          `{"comments": []}`,
		},
		bytes: map[string][]byte{"/export?": witnessExport(t)},
	})

	got, code := runJSON(t, "restyle", fixtureDocID, "--dry-run")
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --dry-run: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if data["nothing_to_protect"] != true {
		t.Errorf("nothing_to_protect = %v, want true: no comment, nothing pending, no chip", data["nothing_to_protect"])
	}
	threads, _ := data["threads"].(map[string]any)
	if threads == nil || threads["open"] != float64(0) {
		t.Errorf("threads = %v, want none", data["threads"])
	}
}

// The apply is M7b's, and the refusal has to say so. A caller that reads "not a
// flag this command takes" learns the command is broken rather than that the
// half it wants has not been written yet.
func TestRestyleRefusesTheApplyAndNamesTheMilestone(t *testing.T) {
	stubSession(t, restyleSession(t))

	got, code := runJSON(t, "restyle", fixtureDocID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("restyle with no --dry-run: %v (exit %d), want a refusal", got, code)
	}
	err, _ := got["error"].(string)
	if !strings.Contains(err, "--dry-run") || !strings.Contains(err, "M7b") {
		t.Errorf("error = %q, want it to name --dry-run and M7b", err)
	}
}

// The survey writes nothing to a document, so the policy it opens is the read
// commands': the one document it was given, at LevelSuggest, and no grant.
func TestRestyleOpensTheOneDocumentWithNoGrant(t *testing.T) {
	f := stubSession(t, restyleSession(t))

	if _, code := runJSON(t, "restyle", fixtureDocID, "--dry-run"); code != 0 {
		t.Fatalf("restyle failed: exit %d", code)
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
	direct := []byte(`{"requests":[{"insertText":{"text":"x","location":{"index":1}}}]}`)
	if err := f.policy.Judge("POST", edit, direct); err == nil {
		t.Error("a direct edit must be refused: a handed-in document is read and suggest only")
	}
	reject := []byte(`{"requests":[{"rejectSuggestion":{"suggestionId":"suggest.abc"}}],` +
		`"writeControl":{"writeMode":"SUGGEST"}}`)
	if err := f.policy.Judge("POST", edit, reject); err == nil {
		t.Error("a survey grants nothing, so a rejectSuggestion must be refused")
	}
	create := mustParse(t, "https://www.googleapis.com/drive/v3/files?fields=id")
	if err := f.policy.Judge("POST", create, []byte(`{"parents":["FOLDER1"]}`)); err == nil {
		t.Error("a survey creates nothing, so a create must be refused")
	}
}

// An export that could not be read is a warning on a survey that still carries
// its threads, exactly as `comments --witness` behaves. The witness is a second
// read on top of the threads, and the threads are the answer.
func TestRestyleReportsAnExportThatCouldNotBeRead(t *testing.T) {
	f := restyleSession(t)
	f.bytes = map[string][]byte{"/export?": []byte("this is not a docx")}
	stubSession(t, f)

	got, code := runJSON(t, "restyle", fixtureDocID, "--dry-run")
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --dry-run: %v (exit %d), want the survey to survive a failed export", got, code)
	}
	if !hasWarning(warningsOf(t, got), "the docx export could not be read") {
		t.Errorf("warnings = %v, want one naming the export", warningsOf(t, got))
	}
	threads, _ := dataOf(t, got)["threads"].(map[string]any)
	witness, _ := threads["witness"].(map[string]any)
	if witness["unmatched"] != float64(2) {
		t.Errorf("witness = %v, want every thread unmatched", witness)
	}
}

// A Docs read that failed is a failed survey. There is no honest report to
// print: the chips, the suggestions and the named ranges are all in that read.
func TestRestyleFailsWhenADocumentReadFailed(t *testing.T) {
	f := restyleSession(t)
	delete(f.json, "docs.googleapis.com")
	stubSession(t, f)

	got, code := runJSON(t, "restyle", fixtureDocID, "--dry-run")
	if code == 0 || got["ok"] != false {
		t.Fatalf("restyle with no document read: %v (exit %d), want a failure", got, code)
	}
	if err, _ := got["error"].(string); err == "" {
		t.Error("a failed read must say what failed")
	}
}

func TestRestyleArgumentsAreStrict(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"no document", []string{"restyle", "--dry-run"}, "needs a document"},
		{"not a document", []string{"restyle", "policy.md", "--dry-run"}, "not a Google Docs URL"},
		{"unknown flag", []string{"restyle", fixtureDocID, "--dry-run", "--force"}, "not a flag this command takes"},
		{"repeated flag", []string{"restyle", fixtureDocID, "--dry-run", "--dry-run"}, "given twice"},
		{"a value it takes none of", []string{"restyle", fixtureDocID, "--dry-run=yes"}, "takes no value"},
		{"an extra word", []string{"restyle", fixtureDocID, fixtureDocID, "--dry-run"}, "extra argument"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stubSession(t, restyleSession(t))

			got, code := runJSON(t, c.args...)
			if code == 0 || got["ok"] != false {
				t.Fatalf("%v: %v (exit %d), want a refusal", c.args, got, code)
			}
			if err, _ := got["error"].(string); !strings.Contains(err, c.want) {
				t.Errorf("error = %q, want it to name %q", err, c.want)
			}
		})
	}
}
