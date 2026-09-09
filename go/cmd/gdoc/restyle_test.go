package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
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

// Nothing to protect is five counts at zero, and it is a fact rather than a
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
		{"--from with no value", []string{"restyle", fixtureDocID, "--from"}, "needs a value"},
		{"--from given an empty value", []string{"restyle", fixtureDocID, "--from="}, "empty value"},
		{"--from given another flag", []string{"restyle", fixtureDocID, "--from", "--dry-run"}, "another flag"},
		{"--from twice", []string{"restyle", fixtureDocID, "--from", "a.json", "--from", "b.json"}, "given twice"},
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

// The apply half. Every test below writes the survey to a file, because that is
// how the two runs are joined: the survey run prints one JSON object, the
// caller saves it, and the apply run reads it back and rechecks the document
// against it before anything is granted.

// twoTabDocument is the one shape the apply refuses on sight. A style request
// names a range, and a range means nothing without saying which tab it is in.
const twoTabDocument = `{
  "documentId": "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
  "title": "A document in two halves",
  "revisionId": "ALm37BXsingleTab",
  "tabs": [
    {"tabProperties": {"tabId": "t.0"},
     "documentTab": {"body": {"content": [
       {"startIndex": 1, "endIndex": 7, "paragraph": {
         "paragraphStyle": {"namedStyleType": "NORMAL_TEXT"},
         "elements": [{"startIndex": 1, "endIndex": 7, "textRun": {"content": "Hello\n"}}]}}]}}},
    {"tabProperties": {"tabId": "t.1"},
     "documentTab": {"body": {"content": [
       {"startIndex": 1, "endIndex": 7, "paragraph": {
         "paragraphStyle": {"namedStyleType": "NORMAL_TEXT"},
         "elements": [{"startIndex": 1, "endIndex": 7, "textRun": {"content": "Again\n"}}]}}]}}}
  ]
}`

// surveyOf is the envelope the survey run printed, cut down to the three facts
// the apply rechecks. It decodes into the same struct the survey printed, which
// is what makes the strict read possible at all.
func surveyOf(t *testing.T, docID, revisionID string, tabs int) string {
	t.Helper()
	return fmt.Sprintf(`{"ok":true,"data":{"document_id":%q,"revision_id":%q,"tabs":%d}}`,
		docID, revisionID, tabs)
}

// applyWire is the fresh read the apply makes and the batch it sends.
func applyWire(t *testing.T) *fakeWire {
	t.Helper()
	return &fakeWire{answers: []*answer{
		{method: "GET", match: "docs.googleapis.com", json: readFixture(t, "single-tab.json")},
		{method: "POST", match: ":batchUpdate",
			json: `{"documentId":"` + fixtureDocID + `","writeControl":{"requiredRevisionId":"ALm37BXafterTheBatch"}}`},
	}}
}

func TestRestyleAppliesTheHouseStyleInPlace(t *testing.T) {
	// Arrange
	f := stubWire(t, applyWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))

	// Act
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)

	// Assert
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --from: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if data["document_id"] != fixtureDocID {
		t.Errorf("document_id = %v", data["document_id"])
	}
	if data["revision_id"] != "ALm37BXafterTheBatch" {
		t.Errorf("revision_id = %v, want the revision the batch answered with", data["revision_id"])
	}
	planned, _ := data["planned"].(map[string]any)
	if planned == nil || planned["paragraphs"] == float64(0) {
		t.Fatalf("planned = %v, want the fixture's paragraphs counted", data["planned"])
	}
	if planned["bulleted"] != float64(2) {
		t.Errorf("planned.bulleted = %v, want the two bulleted paragraphs reported and left alone", planned["bulleted"])
	}
	if planned["tables"] != float64(1) {
		t.Errorf("planned.tables = %v, want the one table reported as not laid out", planned["tables"])
	}
	applied, _ := data["applied"].(map[string]any)
	if applied == nil || applied["batches"] != float64(1) {
		t.Fatalf("applied = %v, want one batch", data["applied"])
	}
	if applied["stale"] != false {
		t.Errorf("applied.stale = %v, want false", applied["stale"])
	}

	writes := f.writes()
	if len(writes) != 1 {
		t.Fatalf("writes = %v, want one batchUpdate", writes)
	}
	var body struct {
		Requests     []map[string]json.RawMessage `json:"requests"`
		WriteControl struct {
			RequiredRevisionID string `json:"requiredRevisionId"`
		} `json:"writeControl"`
	}
	if err := json.Unmarshal(writes[0].Body, &body); err != nil {
		t.Fatalf("the batch is not JSON: %v", err)
	}
	if body.WriteControl.RequiredRevisionID != "ALm37BXsingleTab" {
		t.Errorf("requiredRevisionId = %q, want the revision the fresh read named",
			body.WriteControl.RequiredRevisionID)
	}
	if len(body.Requests) == 0 {
		t.Fatal("the batch carried no requests")
	}
	if _, page := body.Requests[0]["updateDocumentStyle"]; !page {
		t.Errorf("the first request is %v, want the page geometry", body.Requests[0])
	}
}

// The one line that opens direct edit, read from the other side: the policy the
// run ended on carries a styling batch on that one document, and nothing else.
func TestRestyleGrantsTheOneDocumentInPlaceAndNothingElse(t *testing.T) {
	f := stubWire(t, applyWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))

	if _, code := runJSON(t, "restyle", fixtureDocID, "--from", from); code != 0 {
		t.Fatalf("restyle failed: exit %d", code)
	}
	if f.policy == nil {
		t.Fatal("the command opened no policy")
	}
	edit := mustParse(t, "https://docs.googleapis.com/v1/documents/"+fixtureDocID+":batchUpdate")
	style := []byte(`{"requests":[{"updateParagraphStyle":{"range":{"startIndex":1,"endIndex":7},` +
		`"paragraphStyle":{"spaceAbove":{"magnitude":18,"unit":"PT"}},"fields":"spaceAbove"}}],` +
		`"writeControl":{"requiredRevisionId":"ALm37BXsingleTab"}}`)
	if err := f.policy.Judge("POST", edit, style); err != nil {
		t.Errorf("the granted document must take a styling batch: %v", err)
	}
	insert := []byte(`{"requests":[{"insertText":{"text":"x","location":{"index":1}}}]}`)
	if err := f.policy.Judge("POST", edit, insert); err == nil {
		t.Error("the grant is styling only: a kind that changes a character must be refused")
	}
	other := mustParse(t, "https://docs.googleapis.com/v1/documents/9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz:batchUpdate")
	if err := f.policy.Judge("POST", other, style); err == nil {
		t.Error("the grant is one document: another must be refused")
	}
	trash := mustParse(t, "https://www.googleapis.com/drive/v3/files/"+fixtureDocID)
	if err := f.policy.Judge("PATCH", trash, []byte(`{"trashed":true}`)); err == nil {
		t.Error("the in-place level is not LevelFull: a restyle must not be able to trash what it styles")
	}
}

// Each refusal below happens before the grant is opened, and each test asks
// that from the outside: nothing was written, and the policy the run ended on
// still refuses a direct edit.
func assertNothingWasGranted(t *testing.T, f *fakeWire) {
	t.Helper()
	if w := f.writes(); len(w) != 0 {
		t.Errorf("writes = %v, want nothing sent", w)
	}
	if f.policy == nil {
		return // refused before a session was opened, which is earlier still
	}
	edit := mustParse(t, "https://docs.googleapis.com/v1/documents/"+fixtureDocID+":batchUpdate")
	style := []byte(`{"requests":[{"updateParagraphStyle":{"range":{"startIndex":1,"endIndex":7},` +
		`"paragraphStyle":{"spaceAbove":{"magnitude":18,"unit":"PT"}},"fields":"spaceAbove"}}]}`)
	if err := f.policy.Judge("POST", edit, style); err == nil {
		t.Error("the grant was opened on a run that was refused")
	}
}

func TestRestyleRefusesASurveyOfAnotherDocument(t *testing.T) {
	f := stubWire(t, applyWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, proposeDocID, "ALm37BXsingleTab", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("restyle of another document: %v (exit %d), want a refusal", got, code)
	}
	err, _ := got["error"].(string)
	if !strings.Contains(err, proposeDocID) || !strings.Contains(err, fixtureDocID) {
		t.Errorf("error = %q, want it to name both documents", err)
	}
	assertNothingWasGranted(t, f)
	if f.policy != nil {
		t.Error("a survey of another document is refused before a session is opened")
	}
}

func TestRestyleRefusesADocumentThatMoved(t *testing.T) {
	f := stubWire(t, applyWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXtakenEarlier", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("restyle of a moved document: %v (exit %d), want a refusal", got, code)
	}
	err, _ := got["error"].(string)
	if !strings.Contains(err, "ALm37BXtakenEarlier") || !strings.Contains(err, "ALm37BXsingleTab") {
		t.Errorf("error = %q, want it to name the revision the survey was taken at and the one the document is on", err)
	}
	assertNothingWasGranted(t, f)
}

func TestRestyleRefusesASecondTab(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: []*answer{
		{method: "GET", match: "docs.googleapis.com", json: twoTabDocument},
		{method: "POST", match: ":batchUpdate", json: `{}`},
	}})
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("restyle of a two-tab document: %v (exit %d), want a refusal", got, code)
	}
	if err, _ := got["error"].(string); !strings.Contains(err, "2 tabs") {
		t.Errorf("error = %q, want it to name the tab count", err)
	}
	assertNothingWasGranted(t, f)
}

func TestRestyleReadsTheSurveyStrictly(t *testing.T) {
	cases := []struct {
		name   string
		survey string
		want   string
	}{
		{"an unknown key", `{"ok":true,"data":{"document_id":"` + fixtureDocID +
			`","revision_id":"ALm37BXsingleTab","tabs":1,"restyled":true}}`, "restyled"},
		{"an unknown key in the envelope", `{"ok":true,"exit":0,"data":{"document_id":"` + fixtureDocID + `"}}`, "exit"},
		{"a survey that failed", `{"ok":false,"error":"the document could not be read"}`, "did not succeed"},
		{"two objects", `{"ok":true,"data":{"document_id":"` + fixtureDocID +
			`","revision_id":"ALm37BXsingleTab","tabs":1}}{"ok":true}`, "more than one"},
		{"no document named", `{"ok":true,"data":{"revision_id":"ALm37BXsingleTab","tabs":1}}`, "names no document"},
		{"no revision", `{"ok":true,"data":{"document_id":"` + fixtureDocID + `","tabs":1}}`, "no revision id"},
		{"not JSON", `this is not a survey`, "is not a survey"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := stubWire(t, applyWire(t))
			from := tempFile(t, "survey.json", c.survey)

			got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)
			if code == 0 || got["ok"] != false {
				t.Fatalf("%s: %v (exit %d), want a refusal", c.name, got, code)
			}
			if err, _ := got["error"].(string); !strings.Contains(err, c.want) {
				t.Errorf("error = %q, want it to name %q", err, c.want)
			}
			assertNothingWasGranted(t, f)
		})
	}
}

func TestRestyleRefusesASurveyItCannotRead(t *testing.T) {
	f := stubWire(t, applyWire(t))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", filepath.Join(t.TempDir(), "absent.json"))
	if code == 0 || got["ok"] != false {
		t.Fatalf("restyle with no survey file: %v (exit %d), want a refusal", got, code)
	}
	if err, _ := got["error"].(string); !strings.Contains(err, "could not be read") {
		t.Errorf("error = %q, want it to say the survey could not be read", err)
	}
	assertNothingWasGranted(t, f)
}

// A run that stops in the middle says what it left behind. There is no
// rollback, and the sentence about the author's own run formatting is the one
// the skill reads to Nail.
func TestRestyleReportsWhatAFailedRunLeftBehind(t *testing.T) {
	stubWire(t, &fakeWire{answers: []*answer{
		{method: "GET", match: "docs.googleapis.com", json: readFixture(t, "single-tab.json")},
		{method: "POST", match: ":batchUpdate",
			err: errors.New("400: Invalid requiredRevisionId: the document has been changed since that revision")},
	}})
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a refused batch: %v (exit %d), want a failure", got, code)
	}
	applied, _ := dataOf(t, got)["applied"].(map[string]any)
	if applied == nil || applied["stale"] != true {
		t.Errorf("applied = %v, want stale reported as itself", dataOf(t, got)["applied"])
	}
	if !hasWarning(warningsOf(t, got), "no batch was applied") {
		t.Errorf("warnings = %v, want one saying what the run left behind", warningsOf(t, got))
	}
}

// The two halves are two runs. A run given both flags has said two things, and
// which one it meant is not decided here.
func TestRestyleRefusesTheTwoHalvesTogether(t *testing.T) {
	stubWire(t, applyWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--dry-run", "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("both flags: %v (exit %d), want a refusal", got, code)
	}
	if err, _ := got["error"].(string); !strings.Contains(err, "--dry-run") || !strings.Contains(err, "--from") {
		t.Errorf("error = %q, want it to name both flags", err)
	}
}

// Neither flag is a run that said nothing about which half it wanted, and the
// refusal names both rather than assuming one.
func TestRestyleRefusesNeitherFlag(t *testing.T) {
	stubWire(t, applyWire(t))

	got, code := runJSON(t, "restyle", fixtureDocID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("neither flag: %v (exit %d), want a refusal", got, code)
	}
	err, _ := got["error"].(string)
	if !strings.Contains(err, "--dry-run") || !strings.Contains(err, "--from") {
		t.Errorf("error = %q, want it to name both halves", err)
	}
}
