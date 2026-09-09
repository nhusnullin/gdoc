package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/restyle"
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

// surveyOf is the envelope the survey run printed, cut down to the facts the
// apply rechecks. It decodes into the same struct the survey printed, which is
// what makes the strict read possible at all.
//
// The schema is one of those facts. A survey that states none was printed by a
// gdoc older than this one, and what is missing from it is the evidence the
// read-back compares against, so the apply refuses it by name rather than
// reading its absent fields as nothing lost.
func surveyOf(t *testing.T, docID, revisionID string, tabs int) string {
	t.Helper()
	return fmt.Sprintf(`{"ok":true,"data":{"schema":%d,"document_id":%q,"revision_id":%q,"tabs":%d}}`,
		restyle.Schema, docID, revisionID, tabs)
}

// applyWire is the fresh read the apply makes, the batch it sends, and the
// three reads the read-back makes once that batch has landed.
func applyWire(t *testing.T) *fakeWire {
	t.Helper()
	return &fakeWire{answers: []*answer{
		{method: "GET", match: "docs.googleapis.com", json: readFixture(t, "single-tab.json")},
		{method: "POST", match: ":batchUpdate",
			json: `{"documentId":"` + fixtureDocID + `","writeControl":{"requiredRevisionId":"ALm37BXafterTheBatch"}}`},
		{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
		{method: "GET", match: "/export?", bytes: witnessExport(t)},
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
		{"an unknown key", `{"ok":true,"data":{"schema":1,"document_id":"` + fixtureDocID +
			`","revision_id":"ALm37BXsingleTab","tabs":1,"restyled":true}}`, "restyled"},
		{"an unknown key in the envelope", `{"ok":true,"exit":0,"data":{"document_id":"` + fixtureDocID + `"}}`, "exit"},
		{"a survey that failed", `{"ok":false,"error":"the document could not be read"}`, "did not succeed"},
		{"two objects", `{"ok":true,"data":{"schema":1,"document_id":"` + fixtureDocID +
			`","revision_id":"ALm37BXsingleTab","tabs":1}}{"ok":true}`, "more than one"},
		// The two version cases. A survey with no schema is one an older gdoc
		// printed, and the fields this recheck rests on may not be in it: the
		// suggestion ids arrived at M7b, and a survey read without them reports
		// every suggestion a run destroyed as one that was never pending. A
		// survey stating another version is refused for the mirror reason.
		{"no schema", `{"ok":true,"data":{"document_id":"` + fixtureDocID +
			`","revision_id":"ALm37BXsingleTab","tabs":1}}`, "no survey schema"},
		{"another schema", `{"ok":true,"data":{"schema":2,"document_id":"` + fixtureDocID +
			`","revision_id":"ALm37BXsingleTab","tabs":1}}`, "survey schema 2"},
		{"no document named", `{"ok":true,"data":{"schema":1,"revision_id":"ALm37BXsingleTab","tabs":1}}`, "names no document"},
		{"no revision", `{"ok":true,"data":{"schema":1,"document_id":"` + fixtureDocID + `","tabs":1}}`, "no revision id"},
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

// The read-back, both halves. Under LevelSuggest a write had two bars, the
// guard's refusal and writeMode; once direct edit is open for one document that
// document has no second bar, and reading it back is what stands in its place.

// plainDocument is one NORMAL_TEXT paragraph and nothing else, so the plan over
// it is the page, one paragraph and one run: three requests, and three checks
// the read-back can make.
const plainDocument = `{
  "documentId": "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
  "title": "A plain policy",
  "revisionId": "ALm37BXplain",
  "tabs": [{"tabProperties": {"tabId": "t.0"}, "documentTab": {"body": {"content": [
    {"startIndex": 1, "endIndex": 14, "paragraph": {
      "paragraphStyle": {"namedStyleType": "NORMAL_TEXT"},
      "elements": [{"startIndex": 1, "endIndex": 14,
                    "textRun": {"content": "The supplier\n"}}]}}]}}}]
}`

// styledDocument is that document as it reads once the house style landed on
// it. Every value here is written out as a literal, because a test that reads
// house.yaml to say what the house style is would follow it anywhere.
const styledDocument = `{
  "documentId": "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
  "title": "A plain policy",
  "revisionId": "ALm37BXafterTheBatch",
  "tabs": [{"tabProperties": {"tabId": "t.0"}, "documentTab": {
    "documentStyle": {
      "pageSize": {"width": {"magnitude": 595.28, "unit": "PT"},
                   "height": {"magnitude": 841.89, "unit": "PT"}},
      "marginTop": {"magnitude": 62.35, "unit": "PT"},
      "marginBottom": {"magnitude": 51, "unit": "PT"},
      "marginLeft": {"magnitude": 51.05, "unit": "PT"},
      "marginRight": {"magnitude": 51.05, "unit": "PT"}},
    "body": {"content": [
    {"startIndex": 1, "endIndex": 14, "paragraph": {
      "paragraphStyle": {"namedStyleType": "NORMAL_TEXT", "alignment": "JUSTIFIED",
                         "lineSpacing": 115},
      "elements": [{"startIndex": 1, "endIndex": 14, "textRun": {"content": "The supplier\n",
        "textStyle": {"weightedFontFamily": {"fontFamily": "Calibri", "weight": 400},
                      "fontSize": {"magnitude": 12, "unit": "PT"},
                      "foregroundColor": {"color": {"rgbColor": {}}}}}}]}}]}}}]
}`

// landedWire is a run whose style really landed: the plain document before, the
// styled one after, and no comment to lose either way.
func landedWire(t *testing.T) *fakeWire {
	t.Helper()
	return &fakeWire{answers: []*answer{
		{method: "GET", match: "docs.googleapis.com", json: plainDocument, once: true},
		{method: "POST", match: ":batchUpdate",
			json: `{"documentId":"` + fixtureDocID + `","writeControl":{"requiredRevisionId":"ALm37BXafterTheBatch"}}`},
		{method: "GET", match: "/comments?", json: `{"comments": []}`},
		{method: "GET", match: "docs.googleapis.com", json: styledDocument},
		{method: "GET", match: "/export?", bytes: witnessExport(t)},
	}}
}

func TestRestyleReadsTheStyleBackOutOfTheDocument(t *testing.T) {
	// Arrange
	stubWire(t, landedWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXplain", 1))

	// Act
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)

	// Assert
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --from: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if data["verified"] != true {
		t.Errorf("verified = %v, warnings %v, read_back %v",
			data["verified"], warningsOf(t, got), data["read_back"])
	}
	rb, _ := data["read_back"].(map[string]any)
	if rb == nil {
		t.Fatalf("no read_back on a run that wrote a batch: %v", data)
	}
	landing, _ := rb["landing"].(map[string]any)
	checks, _ := landing["checks"].([]any)
	if len(checks) != 3 {
		t.Fatalf("checks = %v, want one for the page, one for the paragraph and one for the run", checks)
	}
	for _, c := range checks {
		one, _ := c.(map[string]any)
		if one["held"] != true {
			t.Errorf("check %v did not hold", one)
		}
	}
	preservation, _ := rb["preservation"].(map[string]any)
	if preservation == nil || preservation["intact"] != true {
		t.Errorf("preservation = %v, want intact on a document with nothing in it to lose", preservation)
	}
}

// A batch Docs accepted is not a change a reader can see. Here the document
// comes back carrying none of the styling that was sent, so every check names
// what is missing and the run reports verified: false over a write Docs took.
// The section-override shape, where the value does read back and a section's own
// governs the page, is internal/restyle's:
// TestAPageMarginASectionOverridesIsNotReportedAsLanded.
func TestRestyleSaysWhenTheStyleDidNotLand(t *testing.T) {
	w := landedWire(t)
	w.answers[3].json = plainDocument // the read back carries no styling at all
	stubWire(t, w)
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXplain", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)

	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --from: %v (exit %d), want the write reported rather than failed", got, code)
	}
	if dataOf(t, got)["verified"] != false {
		t.Error("verified = true over a document that carries none of the style that was sent")
	}
	if !hasWarning(warningsOf(t, got), "updateDocumentStyle") {
		t.Errorf("warnings = %v, want the route that did not hold named", warningsOf(t, got))
	}
}

// The measurement this milestone rests on: replacing a document's body
// destroyed every comment anchor. The before-witness comes from the survey, so
// a thread that was already detached is not read as damage this run did.
func TestRestyleReportsAThreadThatLostItsAnchor(t *testing.T) {
	w := landedWire(t)
	w.answers[2].json = `{"comments": [{"id": "c1", "content": "ai? this one",
	  "author": {"displayName": "Nail"}, "quotedFileContent": {"value": "The supplier"},
	  "modifiedTime": "2026-09-09T10:00:00.000Z"}]}`
	// The export answers for that thread and says it is no longer attached.
	w.answers[4].bytes = detachedExport(t)
	stubWire(t, w)
	// The survey says it was anchored before the run.
	from := tempFile(t, "survey.json", fmt.Sprintf(
		`{"ok":true,"data":{"schema":%d,"document_id":%q,"revision_id":"ALm37BXplain","tabs":1,
		  "threads":{"open":1,"witnessed":[{"id":"c1","witness":"anchored"}]}}}`,
		restyle.Schema, fixtureDocID))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)

	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --from: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if data["verified"] != false {
		t.Error("verified = true on a run that detached a comment")
	}
	if !hasWarning(warningsOf(t, got), "detached") {
		t.Errorf("warnings = %v, want the thread that lost its anchor named", warningsOf(t, got))
	}
	rb, _ := data["read_back"].(map[string]any)
	preservation, _ := rb["preservation"].(map[string]any)
	threads, _ := preservation["threads"].(map[string]any)
	lost, _ := threads["lost_anchor"].([]any)
	if len(lost) != 1 || lost[0] != "c1" {
		t.Errorf("lost_anchor = %v, want c1", threads["lost_anchor"])
	}
}

// detachedExport is the docx a run that broke an anchor exports as: the comment
// is still in the file and no longer attached to any text. The export is the
// only witness there is, because comments.list keeps reporting a destroyed
// anchor as healthy.
func detachedExport(t *testing.T) []byte {
	t.Helper()
	parts := map[string]string{
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body><w:p><w:r><w:t>The supplier</w:t></w:r></w:p></w:body>
</w:document>`,
		"word/comments.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:comment w:id="1" w:author="Nail" w:date="2026-09-09T10:00:00Z">
    <w:p><w:r><w:t>ai? this one</w:t></w:r></w:p>
  </w:comment>
</w:comments>`,
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

// The three things no Docs request can create, and the two this milestone chose
// not to do. SPEC has gdoc write them into the document as a finishing
// checklist; Nail's decision of 2026-09-09 is that it reports them instead.
func TestRestyleNamesWhatItCouldNotDoAtAll(t *testing.T) {
	stubWire(t, applyWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)
	if code != 0 {
		t.Fatalf("restyle failed: %v (exit %d)", got, code)
	}
	rb, _ := dataOf(t, got)["read_back"].(map[string]any)
	steps, _ := rb["manual"].([]any)
	if len(steps) < 3 {
		t.Fatalf("manual = %v, want at least the three the API cannot create", rb["manual"])
	}
	var what, where string
	for _, s := range steps {
		one, _ := s.(map[string]any)
		what += fmt.Sprint(one["what"]) + "\n"
		where += fmt.Sprint(one["where"]) + "\n"
	}
	for _, want := range []string{"header", "contents", "page numbers", "bulleted", "column widths"} {
		if !strings.Contains(what, want) {
			t.Errorf("no manual step names %q: %v", want, what)
		}
	}
	if !strings.Contains(where, "Insert > Table of contents") {
		t.Errorf("the contents list has no menu path: %v", where)
	}
}

// A run that wrote nothing has nothing to read back, and three reads there
// would be answering a question nobody asked.
func TestRestyleReadsNothingBackWhenNoBatchLanded(t *testing.T) {
	w := landedWire(t)
	w.answers[1] = &answer{method: "POST", match: ":batchUpdate",
		err: errors.New("HTTP 403: the caller does not have permission")}
	f := stubWire(t, w)
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXplain", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)

	if code == 0 || got["ok"] != false {
		t.Fatalf("restyle with a refused batch: %v (exit %d), want a failure", got, code)
	}
	if _, there := dataOf(t, got)["read_back"]; there {
		t.Error("a run that wrote nothing must read nothing back: the document is as it was")
	}
	for _, c := range f.calls {
		if strings.Contains(c.URL, "/export?") {
			t.Error("the export ran on a document nothing was written to")
		}
	}
}

// A read-back the run could not make is a warning and no read-back at all.
// Reporting a preservation half built from a listing that never arrived would
// name every thread in the survey as gone, which is the one warning that must
// never cry wolf.
func TestRestyleSaysWhenItCouldNotReadTheDocumentBack(t *testing.T) {
	w := landedWire(t)
	w.answers[2] = &answer{method: "GET", match: "/comments?",
		err: errors.New("HTTP 500: the backend is having a moment")}
	stubWire(t, w)
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXplain", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)

	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --from: %v (exit %d), want the write reported rather than failed", got, code)
	}
	data := dataOf(t, got)
	if data["verified"] != false {
		t.Error("verified = true on a run whose read-back never happened")
	}
	if _, there := data["read_back"]; there {
		t.Error("a read-back that could not be made must not be printed as one")
	}
	if !hasWarning(warningsOf(t, got), "could not be read back") {
		t.Errorf("warnings = %v, want one naming the read that failed", warningsOf(t, got))
	}
}

// The two halves are joined by a file, so that file has to round-trip. The
// survey prints its envelope through internal/emit and the apply reads it back
// through a second struct with DisallowUnknownFields set, which refuses a key it
// does not know at every level of the object.
//
// Every other test here writes its survey by hand, and a hand-written one cannot
// answer this: a field added to what the survey prints and not to what the apply
// reads would refuse every survey gdoc itself printed, the documented workflow
// would stop working, and the suite would still pass. So this one runs the
// survey, keeps the bytes it printed, and hands exactly those to the apply.
func TestASurveyThisBinaryPrintedIsOneItCanApply(t *testing.T) {
	// Arrange: the survey run, and its stdout saved the way the caller saves it.
	stubSession(t, restyleSession(t))
	var printed bytes.Buffer
	if code := run(context.Background(),
		[]string{"restyle", fixtureDocID, "--dry-run"}, &printed, io.Discard); code != 0 {
		t.Fatalf("restyle --dry-run failed: exit %d, %s", code, printed.String())
	}
	from := filepath.Join(t.TempDir(), "survey.json")
	if err := os.WriteFile(from, printed.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	// Act: the apply, reading exactly those bytes.
	stubWire(t, applyWire(t))
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)

	// Assert
	if code != 0 || got["ok"] != true {
		t.Fatalf("the survey this binary printed was refused by its own apply: %v (exit %d)\nsurvey was %s",
			got, code, printed.String())
	}
	if applied, _ := dataOf(t, got)["applied"].(map[string]any); applied == nil || applied["batches"] != float64(1) {
		t.Errorf("applied = %v, want the batch sent", dataOf(t, got)["applied"])
	}
}

// sentAnywayErr marks an error the way internal/gapi marks a write whose answer
// could not be read: the request reached Docs, and the run cannot say what it
// did. Asked by behaviour rather than by importing the sentinel, which is how
// every writer package asks it.
type sentAnywayErr struct{ error }

func (sentAnywayErr) Sent() bool { return true }

// A batch Docs accepted whose answer could not be read is not a batch that never
// happened. It may be in the document, which under this grant means the document
// may have been directly edited, so the run reads back what survived rather than
// reporting a document that is as it was.
func TestRestyleReadsBackWhenABatchMayHaveLanded(t *testing.T) {
	w := landedWire(t)
	w.answers[1] = &answer{method: "POST", match: ":batchUpdate",
		err: sentAnywayErr{errors.New("the answer was not JSON")}}
	stubWire(t, w)
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXplain", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)

	if code == 0 || got["ok"] != false {
		t.Fatalf("an unreadable answer must fail the run: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if _, there := data["read_back"]; !there {
		t.Error("a batch that may be in the document must be read back: it is the run the preservation facts are needed for most")
	}
	warns := strings.Join(warningsOf(t, got), " ")
	if strings.Contains(warns, "the document is as it was") {
		t.Errorf("the warnings say the document is unchanged over a batch Docs accepted: %v", warningsOf(t, got))
	}
	if !strings.Contains(warns, "may be in the document") {
		t.Errorf("the warnings must say the batch may be there: %v", warningsOf(t, got))
	}
	if data["verified"] != false {
		t.Error("verified = true on a run that cannot say what it wrote")
	}
	// The landing half is asked about the batch all the same. Docs accepted it,
	// so those requests left the machine, and reading them back is the only way
	// anybody finds out whether the styling is in the document. What they may
	// never do is make the run verified, which the check above states.
	rb, _ := data["read_back"].(map[string]any)
	landing, _ := rb["landing"].(map[string]any)
	checks, _ := landing["checks"].([]any)
	if len(checks) == 0 {
		t.Error("no landing check on the one run where the read-back is the only way to know what landed")
	}
	if rb["verified"] != false {
		t.Errorf("read_back.verified = %v, want false: the checks holding does not make an unreadable answer readable", rb["verified"])
	}
}
