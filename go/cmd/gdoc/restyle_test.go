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

	"gdoc/internal/cover"
	"gdoc/internal/house"
	"gdoc/internal/prelude"
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
		{"--fields with no value", []string{"restyle", fixtureDocID, "--from", "a.json", "--fields"}, "needs a value"},
		{"--fields given an empty value", []string{"restyle", fixtureDocID, "--from", "a.json", "--fields="}, "empty value"},
		{"--fields given another flag", []string{"restyle", fixtureDocID, "--from", "a.json", "--fields", "--dry-run"}, "another flag"},
		{"--fields twice", []string{"restyle", fixtureDocID, "--from", "a.json", "--fields", "a.json", "--fields", "b.json"}, "given twice"},
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
      "pageSize": {"width": {"magnitude": 595.2755905511812, "unit": "PT"},
                   "height": {"magnitude": 841.8897637795277, "unit": "PT"}},
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

// The round trip above runs on a checkout, where the envelope carries no
// version. A release binary stamps one into every object it prints, the survey
// included, so the survey a colleague saves carries a key the checkout's never
// does. The strict read has to know that key, or every survey a release printed
// is refused by its own apply while this suite still passes: that is how v2.1.0
// shipped refusing its own surveys with `json: unknown field "version"`.
func TestASurveyAReleaseBinaryPrintedIsOneItCanApply(t *testing.T) {
	// Arrange: a release-stamped survey run, its stdout saved as the caller saves it.
	atVersion(t, "v9.9.9")
	stubSession(t, restyleSession(t))
	var printed bytes.Buffer
	if code := run(context.Background(),
		[]string{"restyle", fixtureDocID, "--dry-run"}, &printed, io.Discard); code != 0 {
		t.Fatalf("restyle --dry-run failed: exit %d, %s", code, printed.String())
	}
	if !bytes.Contains(printed.Bytes(), []byte(`"version":"v9.9.9"`)) {
		t.Fatalf("the release survey must carry the version, or this test proves nothing: %s", printed.String())
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
		t.Fatalf("the survey a release binary printed was refused by its own apply: %v (exit %d)", got, code)
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

// The prelude, and the two phases it needs. Everything below is M7c: a run
// naming --fields proposes the house template into the document as suggestions
// and then styles the body directly, and those are two permissions rather than
// one, so they are two policies and two sessions inside one command.

// fieldsFile is the cover's thirteen values as `gdoc restyle --fields` reads
// them. It is written out here rather than kept as a fixture because what a
// test asks of it is the values it names.
const fieldsFile = `{
  "title": "Supplier register policy",
  "doc_type": "Policy",
  "version": "2.1",
  "date": "2026-09-10",
  "owner": "Head of Operations",
  "classification": "Internal",
  "revisions": [{"version": "2.1", "date": "2026-09-10", "author": "Nail",
                 "change": "First issue"}]
}`

// afterThePrelude is the fixture as the fresh read between the phases sees it:
// the same document on the revision the prelude batch left it on.
func afterThePrelude(t *testing.T) string {
	t.Helper()
	return strings.Replace(readFixture(t, "single-tab.json"),
		"ALm37BXsingleTab", "ALm37BXafterThePrelude", 1)
}

// preludeWire is a whole two-phase run: the read, the prelude batch, the fresh
// read, the marker batch, the styling batch, and the three reads the read-back
// makes. Every write answer is once, because the order of the three batches is
// the thing most of these tests are about.
//
// The last Docs answer is the read-back's, and it is the fixture again: a
// document carrying no prelude and no marker at all. That is the failure path
// on purpose, and it is the one this wire can state honestly, because a fake
// replays bytes rather than applying the requests it was sent. What a prelude
// that really landed reads back as is internal/prelude's, where a document is
// written out run by run.
func preludeWire(t *testing.T) *fakeWire {
	t.Helper()
	return &fakeWire{answers: []*answer{
		{method: "GET", match: "docs.googleapis.com", json: readFixture(t, "single-tab.json"), once: true},
		{method: "POST", match: ":batchUpdate", once: true,
			json: `{"documentId":"` + fixtureDocID + `","writeControl":{"requiredRevisionId":"ALm37BXafterThePrelude"}}`},
		{method: "GET", match: "docs.googleapis.com", json: afterThePrelude(t), once: true},
		{method: "POST", match: ":batchUpdate", once: true,
			json: `{"documentId":"` + fixtureDocID + `","writeControl":{"requiredRevisionId":"ALm37BXafterTheMarker"}}`},
		{method: "POST", match: ":batchUpdate",
			json: `{"documentId":"` + fixtureDocID + `","writeControl":{"requiredRevisionId":"ALm37BXafterTheBatch"}}`},
		{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
		{method: "GET", match: "docs.googleapis.com", json: afterThePrelude(t)},
		{method: "GET", match: "/export?", bytes: witnessExport(t)},
	}}
}

// batchOf is one write the fake saw, decoded far enough to say what kind of
// requests it carried and what write mode it went out in.
type batchOf struct {
	Requests     []map[string]json.RawMessage `json:"requests"`
	WriteControl struct {
		WriteMode          string `json:"writeMode"`
		RequiredRevisionID string `json:"requiredRevisionId"`
	} `json:"writeControl"`
}

func batchesSent(t *testing.T, f *fakeWire) []batchOf {
	t.Helper()
	var out []batchOf
	for _, w := range f.writes() {
		var b batchOf
		if err := json.Unmarshal(w.Body, &b); err != nil {
			t.Fatalf("a write is not a batchUpdate body: %v", err)
		}
		out = append(out, b)
	}
	return out
}

// kinds is the request kinds one batch carried, once each and in the order they
// were first seen.
func (b batchOf) kinds() []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range b.Requests {
		for k := range r {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}

func (b batchOf) carries(kind string) bool { return contains(b.kinds(), kind) }

// The whole run: the prelude proposed, the marker written, the body styled, in
// that order and in one command.
func TestRestyleProposesThePreludeAndThenStyles(t *testing.T) {
	// Arrange
	f := stubWire(t, preludeWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	// Act
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)

	// Assert
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --fields: %v (exit %d)", got, code)
	}
	batches := batchesSent(t, f)
	if len(batches) != 3 {
		t.Fatalf("the run sent %d batches, want the prelude, the marker and the styling", len(batches))
	}
	if batches[0].WriteControl.WriteMode != "SUGGEST" {
		t.Errorf("the prelude batch went out in write mode %q, want SUGGEST: every character of the prelude is a proposal",
			batches[0].WriteControl.WriteMode)
	}
	if !batches[0].carries("insertText") {
		t.Errorf("the prelude batch carried %v, want the cover's own words", batches[0].kinds())
	}
	if batches[0].WriteControl.RequiredRevisionID != "ALm37BXsingleTab" {
		t.Errorf("the prelude batch named revision %q, want the one the fresh read carried",
			batches[0].WriteControl.RequiredRevisionID)
	}
	if !batches[1].carries("createNamedRange") || len(batches[1].Requests) != 1 {
		t.Errorf("the second batch carried %v, want the marker on its own: a styling batch that fails must still leave the prelude marked",
			batches[1].kinds())
	}
	if batches[1].WriteControl.WriteMode != "" {
		t.Errorf("the marker batch went out in write mode %q, and createNamedRange cannot be a suggestion",
			batches[1].WriteControl.WriteMode)
	}
	if batches[1].WriteControl.RequiredRevisionID != "ALm37BXafterThePrelude" {
		t.Errorf("the marker batch named revision %q, want the one the prelude batch answered with",
			batches[1].WriteControl.RequiredRevisionID)
	}
	if !batches[2].carries("updateDocumentStyle") {
		t.Errorf("the third batch carried %v, want M7b's styling", batches[2].kinds())
	}
	if batches[2].WriteControl.WriteMode != "" {
		t.Errorf("the styling batch went out in write mode %q, and M7b's styling is a direct edit",
			batches[2].WriteControl.WriteMode)
	}

	data := dataOf(t, got)
	pre, _ := data["prelude"].(map[string]any)
	if pre == nil {
		t.Fatalf("data carries no prelude object: %v", data)
	}
	if pre["paragraphs"] == float64(0) || pre["tables"] == float64(0) {
		t.Errorf("prelude = %v, want the cover's paragraphs and the front matter's tables counted", pre)
	}
	if pre["marker_created"] != true {
		t.Errorf("prelude.marker_created = %v, want the marker reported as written", pre["marker_created"])
	}
	if pre["replaced"] != nil {
		t.Errorf("prelude.replaced = %v, want nothing replaced on a document carrying no marker", pre["replaced"])
	}
}

// The one mistake that would undo this milestone's whole shape: the prelude
// sent on the policy that opened direct edit. The two phases are two policies,
// and the first of them grants nothing at all.
func TestThePreludePhaseIsSentOnAPolicyThatGrantsNothing(t *testing.T) {
	f := stubWire(t, preludeWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	if _, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields); code != 0 {
		t.Fatalf("restyle failed: exit %d", code)
	}
	if len(f.policies) != 2 {
		t.Fatalf("the run opened %d policies, want one per phase", len(f.policies))
	}
	edit := mustParse(t, "https://docs.googleapis.com/v1/documents/"+fixtureDocID+":batchUpdate")
	style := []byte(`{"requests":[{"updateParagraphStyle":{"range":{"startIndex":1,"endIndex":7},` +
		`"paragraphStyle":{"spaceAbove":{"magnitude":18,"unit":"PT"}},"fields":"spaceAbove"}}],` +
		`"writeControl":{"requiredRevisionId":"ALm37BXsingleTab"}}`)
	suggest := []byte(`{"requests":[{"insertText":{"text":"x","location":{"index":1}}}],` +
		`"writeControl":{"writeMode":"SUGGEST","requiredRevisionId":"ALm37BXsingleTab"}}`)
	marker := []byte(`{"requests":[{"createNamedRange":{"name":"gdoc:house-prelude",` +
		`"range":{"startIndex":1,"endIndex":2}}}],"writeControl":{"requiredRevisionId":"x"}}`)

	first, second := f.policies[0], f.policies[1]
	if err := first.Judge("POST", edit, style); err == nil {
		t.Error("the prelude phase's policy opened direct edit, which is the two phases collapsed into one")
	}
	if err := first.Judge("POST", edit, marker); err == nil {
		t.Error("the prelude phase's policy carried a createNamedRange, and the marker is phase 2's")
	}
	if err := first.Judge("POST", edit, suggest); err != nil {
		t.Errorf("the prelude phase must carry a SUGGEST batch with no grant at all: %v", err)
	}
	if err := second.Judge("POST", edit, style); err != nil {
		t.Errorf("the styling phase's policy must carry a styling batch: %v", err)
	}
	if err := second.Judge("POST", edit, suggest); err == nil {
		t.Error("the styling phase's policy carried an insertText, and nothing at that level may change a character")
	}

	// The bytes the run really sent, judged by the policy they really went out
	// on. The three checks above are shapes written by hand, and this is the
	// run itself: the prelude batch through a policy that granted nothing, and
	// the marker and the styling through the one that granted.
	writes := f.writes()
	if len(writes) != 3 {
		t.Fatalf("the run sent %d writes, want three", len(writes))
	}
	if err := first.Judge("POST", edit, writes[0].Body); err != nil {
		t.Errorf("the guard refused the prelude batch on a policy with no grant, which is the whole claim of this milestone: %v", err)
	}
	for i, w := range writes[1:] {
		if err := second.Judge("POST", edit, w.Body); err != nil {
			t.Errorf("the guard refused write %d of phase 2: %v", i+2, err)
		}
	}
	if err := second.Judge("POST", edit, writes[0].Body); err == nil {
		t.Error("the granted policy carried the prelude batch, so the allowlist at that level is not the bound it is documented as")
	}
	if err := first.Judge("POST", edit, writes[2].Body); err == nil {
		t.Error("the ungranted policy carried the styling batch, so a direct edit needs no grant")
	}
}

// A failed phase 1 leaves the body alone. The document then carries whatever
// the prelude batch left and nothing else, which is a document Nail can still
// read, and the report says which phase stopped.
func TestAFailedPreludePhaseDoesNotStyle(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: []*answer{
		{method: "GET", match: "docs.googleapis.com", json: readFixture(t, "single-tab.json")},
		{method: "POST", match: ":batchUpdate", err: errors.New("400: the prelude was refused")},
	}})
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a failed prelude: %v (exit %d), want a refusal", got, code)
	}
	if err, _ := got["error"].(string); !strings.Contains(err, "prelude") {
		t.Errorf("error = %q, want it to name the phase that stopped", err)
	}
	if n := len(f.writes()); n != 1 {
		t.Errorf("the run sent %d writes, want the one prelude batch and nothing after it", n)
	}
	for _, p := range f.policies {
		edit := mustParse(t, "https://docs.googleapis.com/v1/documents/"+fixtureDocID+":batchUpdate")
		style := []byte(`{"requests":[{"updateParagraphStyle":{"range":{"startIndex":1,"endIndex":7},` +
			`"paragraphStyle":{"spaceAbove":{"magnitude":18,"unit":"PT"}},"fields":"spaceAbove"}}]}`)
		if err := p.Judge("POST", edit, style); err == nil {
			t.Error("direct edit was opened on a run whose prelude phase never landed")
		}
	}
}

// --fields is the prelude, and the prelude is proposed into a document the
// survey described. A run naming it without --from has asked for a write with
// nothing to recheck the document against.
func TestRestyleRefusesFieldsWithoutASurvey(t *testing.T) {
	stubWire(t, applyWire(t))
	fields := tempFile(t, "fields.json", fieldsFile)

	got, code := runJSON(t, "restyle", fixtureDocID, "--fields", fields)
	if code == 0 || got["ok"] != false {
		t.Fatalf("--fields alone: %v (exit %d), want a refusal", got, code)
	}
	err, _ := got["error"].(string)
	if !strings.Contains(err, "--fields") || !strings.Contains(err, "--from") {
		t.Errorf("error = %q, want it to name both flags", err)
	}
}

// The survey writes nothing to a document, so a survey run naming the cover's
// values has asked for two different things at once.
func TestRestyleRefusesTheSurveyAndTheFieldsTogether(t *testing.T) {
	stubWire(t, applyWire(t))
	fields := tempFile(t, "fields.json", fieldsFile)

	got, code := runJSON(t, "restyle", fixtureDocID, "--dry-run", "--fields", fields)
	if code == 0 || got["ok"] != false {
		t.Fatalf("--dry-run --fields: %v (exit %d), want a refusal", got, code)
	}
	err, _ := got["error"].(string)
	if !strings.Contains(err, "--fields") || !strings.Contains(err, "--dry-run") {
		t.Errorf("error = %q, want it to name both flags", err)
	}
}

// The fields file is read before a session is opened, so a file gdoc half
// understands never reaches a document at all.
func TestRestyleReadsTheFieldsFileStrictly(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"an unknown key", `{"title":"A policy","tile":"A policy"}`, "tile"},
		{"a second object behind the first", `{"title":"A policy"}{"title":"Another"}`, "more than one JSON object"},
		{"no title at all", `{"doc_type":"Policy"}`, "no title in the fields file"},
		{"a revision with no version", `{"title":"A policy","revisions":[{"date":"2026-09-10"}]}`, "version"},
		{"not JSON", `title: A policy`, "fields file"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := stubWire(t, applyWire(t))
			from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
			fields := tempFile(t, "fields.json", c.body)

			got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)
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

// A fields file that is not there is the same refusal, and it names the path so
// somebody can see which one gdoc looked for.
func TestRestyleRefusesAFieldsFileItCannotRead(t *testing.T) {
	f := stubWire(t, applyWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	missing := filepath.Join(t.TempDir(), "nowhere.json")

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", missing)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a missing fields file: %v (exit %d), want a refusal", got, code)
	}
	if err, _ := got["error"].(string); !strings.Contains(err, missing) {
		t.Errorf("error = %q, want it to name the file", err)
	}
	assertNothingWasGranted(t, f)
}

// The M7b run, unchanged. A restyle with --from alone styles the body and
// proposes nothing, opens one policy, and says nothing about a prelude.
func TestRestyleWithoutFieldsIsStillTheStylingRun(t *testing.T) {
	f := stubWire(t, applyWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))

	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from)
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --from: %v (exit %d)", got, code)
	}
	batches := batchesSent(t, f)
	if len(batches) != 1 {
		t.Fatalf("the run sent %d batches, want M7b's one", len(batches))
	}
	if batches[0].WriteControl.WriteMode != "" {
		t.Errorf("the styling batch went out in write mode %q, want none", batches[0].WriteControl.WriteMode)
	}
	for _, kind := range batches[0].kinds() {
		if kind == "insertText" || kind == "createNamedRange" {
			t.Errorf("the styling run sent a %s, and a run naming no --fields proposes nothing", kind)
		}
	}
	if data := dataOf(t, got); data["prelude"] != nil {
		t.Errorf("prelude = %v, want the field absent on a run that proposed none", data["prelude"])
	}
	if len(f.policies) != 1 {
		t.Errorf("the run opened %d policies, want M7b's one", len(f.policies))
	}
}

// The prelude phase is the one that goes first, and the styling is computed
// from a read taken after it: the prelude moved every index the styling
// requests name.
func TestTheStylingIsComputedFromAReadTakenAfterThePrelude(t *testing.T) {
	f := stubWire(t, preludeWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	if _, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields); code != 0 {
		t.Fatalf("restyle failed: exit %d", code)
	}
	var order []string
	for _, c := range f.calls {
		switch {
		case c.Method != "GET" && strings.Contains(c.URL, ":batchUpdate"):
			order = append(order, "write")
		case strings.Contains(c.URL, "docs.googleapis.com"):
			order = append(order, "read")
		}
	}
	if len(order) < 4 || order[0] != "read" || order[1] != "write" || order[2] != "read" {
		t.Errorf("order = %v, want a read, the prelude, then a fresh read before anything else", order)
	}
}

// One run, two phases, one report. What was proposed, what was styled, what is
// pending for Nail to accept and the manual list M7b already printed are all in
// the one object, because a caller reading two of them in two places has two
// chances to read a different answer.
func TestTheReportCarriesBothPhases(t *testing.T) {
	// Arrange
	stubWire(t, preludeWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	// Act
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)

	// Assert
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --fields: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	pre, _ := data["prelude"].(map[string]any)
	rb, _ := data["read_back"].(map[string]any)
	if pre == nil || rb == nil {
		t.Fatalf("data carries prelude %v and read_back %v, want both phases in one report", pre, rb)
	}
	check, _ := pre["read_back"].(map[string]any)
	if check == nil {
		t.Fatalf("prelude carries no read_back: %v", pre)
	}
	for _, field := range []string{"proposed", "written", "marked", "body_unchanged", "verified"} {
		if _, there := check[field]; !there {
			t.Errorf("prelude.read_back carries no %s: %v", field, check)
		}
	}
	if check["start"] != pre["start"] || check["end"] != pre["end"] {
		t.Errorf("the read-back looked in [%v,%v) and the prelude was proposed into [%v,%v)",
			check["start"], check["end"], pre["start"], pre["end"])
	}
	// M7b's list, unchanged. The prelude keeps its own beside it, because the
	// two are built from different things.
	manual, _ := rb["manual"].([]any)
	if len(manual) == 0 {
		t.Errorf("read_back.manual = %v, want M7b's list of what gdoc could not do at all", rb["manual"])
	}
	if data["applied"] == nil {
		t.Errorf("data carries no applied, so nothing says what phase 2 sent: %v", data)
	}
}

// The prelude was proposed and the document read back holds neither it nor the
// marker over it. Every one of those is a route that did not hold, so the run
// is ok: true with verified: false and each route named. The batches Docs took
// are in the document either way, and a caller told the run failed is a caller
// that runs it again.
func TestThePreludeReadBackNamesTheRouteThatDidNotHold(t *testing.T) {
	// Arrange
	stubWire(t, preludeWire(t))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	// Act
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)

	// Assert
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --fields: %v (exit %d), want a write that happened reported as one", got, code)
	}
	data := dataOf(t, got)
	if data["verified"] != false {
		t.Errorf("verified = %v, want false: nothing in the document read back says the prelude is there",
			data["verified"])
	}
	pre, _ := data["prelude"].(map[string]any)
	check, _ := pre["read_back"].(map[string]any)
	if check["verified"] != false || check["marked"] != false {
		t.Errorf("prelude.read_back = %v, want verified and marked both false", check)
	}
	if check["body_unchanged"] != true {
		t.Errorf("body_unchanged = %v, want true: not a character of the author's own text moved",
			check["body_unchanged"])
	}
	warnings := strings.Join(warningsOf(t, got), " | ")
	if !strings.Contains(warnings, "gdoc:house-prelude") {
		t.Errorf("warnings = %s, want the marker named as the route that did not hold", warnings)
	}
}

// A styling run proposes nothing, so there is no phase 1 to read back and no
// field saying there was one.
func TestAStylingRunReadsNoPreludeBack(t *testing.T) {
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
	if data["prelude"] != nil {
		t.Errorf("prelude = %v, want the field absent on a run that proposed none", data["prelude"])
	}
	if data["verified"] != true {
		t.Errorf("verified = %v, warnings %v: M7b's run must not be made unverified by a phase it never had",
			data["verified"], warningsOf(t, got))
	}
}

// markedDocument is a document a run before this one left a prelude in, and
// settled: the marker covers [1,7), and the six characters under it carry no
// suggestion id, so this run replaces them rather than refusing.
func markedDocument(revisionID string) string {
	return `{
  "documentId": "` + fixtureDocID + `",
  "title": "Supplier register policy",
  "revisionId": "` + revisionID + `",
  "tabs": [{"tabProperties": {"tabId": "t.0"},
            "documentTab": {
              "namedRanges": {"gdoc:house-prelude": {"name": "gdoc:house-prelude", "namedRanges": [
                {"namedRangeId": "kix.marker1", "name": "gdoc:house-prelude",
                 "ranges": [{"startIndex": 1, "endIndex": 7}]}]}},
              "body": {"content": [
                {"startIndex": 0, "endIndex": 1, "sectionBreak": {"sectionStyle": {}}},
                {"startIndex": 1, "endIndex": 7,
                 "paragraph": {"paragraphStyle": {"namedStyleType": "NORMAL_TEXT"},
                   "elements": [{"startIndex": 1, "endIndex": 7,
                     "textRun": {"content": "Cover\n", "textStyle": {}}}]}},
                {"startIndex": 7, "endIndex": 25,
                 "paragraph": {"paragraphStyle": {"namedStyleType": "NORMAL_TEXT"},
                   "elements": [{"startIndex": 7, "endIndex": 25,
                     "textRun": {"content": "The author's own.\n", "textStyle": {}}}]}}
              ]}}}]
}`
}

// afterTheReplace is that document as the fresh read between the phases sees
// it: the new prelude proposed at index 1, the old one pushed along to
// [end, end+6) and marked for deletion rather than removed, and the author's
// own paragraph behind both of them.
//
// end is where the prelude this run proposes stops, which the test asks
// internal/prelude for, because that is what the command computes it from.
func afterTheReplace(end int) string {
	return fmt.Sprintf(`{
  "documentId": %q,
  "title": "Supplier register policy",
  "revisionId": "ALm37BXafterThePrelude",
  "tabs": [{"tabProperties": {"tabId": "t.0"},
            "documentTab": {
              "namedRanges": {"gdoc:house-prelude": {"name": "gdoc:house-prelude", "namedRanges": [
                {"namedRangeId": "kix.marker1", "name": "gdoc:house-prelude",
                 "ranges": [{"startIndex": %d, "endIndex": %d}]}]}},
              "body": {"content": [
                {"startIndex": 0, "endIndex": 1, "sectionBreak": {"sectionStyle": {}}},
                {"startIndex": 1, "endIndex": %d,
                 "paragraph": {"paragraphStyle": {"namedStyleType": "NORMAL_TEXT"},
                   "elements": [{"startIndex": 1, "endIndex": %d,
                     "textRun": {"content": "A Policy\n", "textStyle": {},
                                 "suggestedInsertionIds": ["suggest.new1"]}}]}},
                {"startIndex": %d, "endIndex": %d,
                 "paragraph": {"paragraphStyle": {"namedStyleType": "NORMAL_TEXT"},
                   "elements": [{"startIndex": %d, "endIndex": %d,
                     "textRun": {"content": "Cover\n", "textStyle": {},
                                 "suggestedDeletionIds": ["suggest.old1"]}}]}},
                {"startIndex": %d, "endIndex": %d,
                 "paragraph": {"paragraphStyle": {"namedStyleType": "NORMAL_TEXT"},
                   "elements": [{"startIndex": %d, "endIndex": %d,
                     "textRun": {"content": "The author's own.\n", "textStyle": {}}}]}}
              ]}}}]
}`, fixtureDocID, end, end+6, end, end, end, end+6, end, end+6, end+6, end+24, end+6, end+24)
}

// preludeEnd is where the prelude these fields come to stops, proposed at the
// top of a body. The test asks internal/prelude rather than writing a number
// down, because the number is the house style's and moves with it.
func preludeEnd(t *testing.T) int {
	t.Helper()
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the embedded house style did not parse: %v", err)
	}
	f, err := cover.ReadFields([]byte(fieldsFile))
	if err != nil {
		t.Fatalf("the fields file did not read: %v", err)
	}
	res, err := prelude.FrontMatter(cfg, f, 1)
	if err != nil {
		t.Fatalf("the prelude did not build: %v", err)
	}
	return res.End
}

// styledRanges is every range the styling batch named. A request that names no
// range, the page and a table cell among them, is not in the list.
func styledRanges(t *testing.T, b batchOf) [][2]int {
	t.Helper()
	var out [][2]int
	for _, r := range b.Requests {
		for _, raw := range r {
			var one struct {
				Range struct {
					StartIndex int `json:"startIndex"`
					EndIndex   int `json:"endIndex"`
				} `json:"range"`
			}
			if err := json.Unmarshal(raw, &one); err != nil || one.Range.EndIndex == 0 {
				continue
			}
			out = append(out, [2]int{one.Range.StartIndex, one.Range.EndIndex})
		}
	}
	return out
}

// replaceWire is a second run over a document gdoc has already put a prelude
// into: the marked read, the prelude batch, the fresh read that carries both
// preludes, the marker batch, the styling batch and the read-back's three
// reads.
func replaceWire(t *testing.T, end int) *fakeWire {
	t.Helper()
	return &fakeWire{answers: []*answer{
		{method: "GET", match: "docs.googleapis.com", json: markedDocument("ALm37BXmarked"), once: true},
		{method: "POST", match: ":batchUpdate", once: true,
			json: `{"documentId":"` + fixtureDocID + `","writeControl":{"requiredRevisionId":"ALm37BXafterThePrelude"}}`},
		{method: "GET", match: "docs.googleapis.com", json: afterTheReplace(end), once: true},
		{method: "POST", match: ":batchUpdate", once: true,
			json: `{"documentId":"` + fixtureDocID + `","writeControl":{"requiredRevisionId":"ALm37BXafterTheMarker"}}`},
		{method: "POST", match: ":batchUpdate",
			json: `{"documentId":"` + fixtureDocID + `","writeControl":{"requiredRevisionId":"ALm37BXafterTheBatch"}}`},
		{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
		{method: "GET", match: "docs.googleapis.com", json: afterTheReplace(end)},
		{method: "GET", match: "/export?", bytes: witnessExport(t)},
	}}
}

// A replace run hands phase 2 the span both preludes occupy, and that is the
// whole reason phaseOne carries two spans.
//
// The deletion goes out in SUGGEST mode, which marks text rather than removing
// it, so the prelude the run before this one left is still real text sitting
// behind the words that replace it. Given the marker's range alone, phase 2
// walks past the new prelude and then gives the old one the house body look by
// direct edit at LevelInPlace, which flattens a cover Nail may yet reject the
// deletion of. Nothing downstream could name it: internal/restyle reads no
// suggestion id, and the read-back compares characters.
func TestAReplaceRunStylesNeitherPrelude(t *testing.T) {
	// Arrange
	end := preludeEnd(t)
	f := stubWire(t, replaceWire(t, end))
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXmarked", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	// Act
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)

	// Assert
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --fields over a marked document: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	pre, _ := data["prelude"].(map[string]any)
	if pre == nil || pre["replaced"] == nil {
		t.Fatalf("prelude = %v, want the marker this run replaced named", pre)
	}
	batches := batchesSent(t, f)
	if len(batches) != 3 {
		t.Fatalf("the run sent %d batches, want the prelude, the marker and the styling", len(batches))
	}
	for _, r := range styledRanges(t, batches[2]) {
		if r[0] < end+6 && r[1] > end {
			t.Errorf("the styling named [%d,%d), which is inside the replaced prelude at [%d,%d): "+
				"that is a direct edit over a cover Nail may yet reject the deletion of", r[0], r[1], end, end+6)
		}
	}
	// And the author's own paragraph behind both preludes is styled, so the
	// assertion above is not holding because the run styled nothing at all.
	var reached bool
	for _, r := range styledRanges(t, batches[2]) {
		if r[0] >= end+6 {
			reached = true
		}
	}
	if !reached {
		t.Errorf("the styling named nothing behind the two preludes: ranges %v", styledRanges(t, batches[2]))
	}
	planned, _ := data["planned"].(map[string]any)
	if planned["skipped"] != float64(2) {
		t.Errorf("planned.skipped = %v, want 2: the prelude this run proposed and the one it proposed deleting",
			planned["skipped"])
	}
}

// The marker batch is Docs accepting a write whose answer could not be read.
// The run stops either way, and what it says about the marker is the one thing
// it can honestly say: the marker may be there.
func TestAMarkerBatchThatMayHaveLandedSaysSo(t *testing.T) {
	// Arrange
	w := preludeWire(t)
	w.answers[3] = &answer{method: "POST", match: ":batchUpdate", once: true,
		err: sentAnywayErr{errors.New("the answer was not JSON")}}
	stubWire(t, w)
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	// Act
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)

	// Assert
	if code == 0 || got["ok"] != false {
		t.Fatalf("a marker that could not be confirmed must stop the run: %v (exit %d)", got, code)
	}
	pre, _ := dataOf(t, got)["prelude"].(map[string]any)
	if pre == nil {
		t.Fatalf("data carries no prelude object: %v", dataOf(t, got))
	}
	if pre["marker_maybe_created"] != true {
		t.Errorf("prelude.marker_maybe_created = %v, want true: Docs took the batch and the answer was lost",
			pre["marker_maybe_created"])
	}
	if pre["marker_created"] != false {
		t.Errorf("prelude.marker_created = %v, want false: nothing confirmed it", pre["marker_created"])
	}
	err, _ := got["error"].(string)
	if !strings.Contains(err, "may or may not be there") {
		t.Errorf("error = %q, want it to say the marker may be there rather than that it did not land", err)
	}
}

// The prelude batch is the other one that can answer naming no revision, and
// the fresh read that follows it is what answers the question. So the sentence
// saying the reported revision is not the document's goes there too, and the
// styling phase is sent against what that read named rather than against the
// revision the prelude was sent on, which Docs would refuse as stale.
func TestThePreludeBatchNamingNoRevisionIsAnsweredByTheFreshRead(t *testing.T) {
	// Arrange
	w := preludeWire(t)
	w.answers[1] = &answer{method: "POST", match: ":batchUpdate", once: true,
		json: `{"documentId":"` + fixtureDocID + `"}`}
	f := stubWire(t, w)
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	// Act
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)

	// Assert
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --fields: %v (exit %d)", got, code)
	}
	batches := batchesSent(t, f)
	if len(batches) != 3 {
		t.Fatalf("the run sent %d batches, want the prelude, the marker and the styling", len(batches))
	}
	if batches[1].WriteControl.RequiredRevisionID != "ALm37BXafterThePrelude" {
		t.Errorf("the marker batch named revision %q, want the one the fresh read named",
			batches[1].WriteControl.RequiredRevisionID)
	}
	warnings := strings.Join(warningsOf(t, got), " | ")
	if strings.Contains(warnings, "not the one the document now carries") {
		t.Errorf("warnings = %s, want the sentence gone: the fresh read answered it, "+
			"and the revision the envelope reports is one Docs named", warnings)
	}
}

// The marker batch is the last batch of its own run, so an answer naming no
// revision leaves Apply reporting the revision that batch was sent against,
// which the marker has already moved the document past. The run reads for one
// rather than sending the styling phase against a revision it knows is stale.
//
// And the warning that said the reported revision is not the document's goes
// with it. It was true when Apply said it and the read is what made it false.
func TestTheStylingIsSentAgainstARevisionTheMarkerBatchDidNotName(t *testing.T) {
	// Arrange
	w := preludeWire(t)
	w.answers[3] = &answer{method: "POST", match: ":batchUpdate", once: true,
		json: `{"documentId":"` + fixtureDocID + `"}`}
	// The read that stands in for it, and it is the narrowed one.
	w.answers = append(w.answers[:4], append([]*answer{
		{method: "GET", match: "docs.googleapis.com", once: true,
			json: `{"documentId":"` + fixtureDocID + `","revisionId":"ALm37BXreadForIt","tabs":[]}`},
	}, w.answers[4:]...)...)
	f := stubWire(t, w)
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	// Act
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)

	// Assert
	if code != 0 || got["ok"] != true {
		t.Fatalf("restyle --fields: %v (exit %d)", got, code)
	}
	batches := batchesSent(t, f)
	if len(batches) != 3 {
		t.Fatalf("the run sent %d batches, want the prelude, the marker and the styling", len(batches))
	}
	if batches[2].WriteControl.RequiredRevisionID != "ALm37BXreadForIt" {
		t.Errorf("the styling batch named revision %q, want the one the run read for: "+
			"sent the marker batch's own, Docs refuses it as stale and the run blames somebody else's edit",
			batches[2].WriteControl.RequiredRevisionID)
	}
	warnings := strings.Join(warningsOf(t, got), " | ")
	if strings.Contains(warnings, "not the one the document now carries") {
		t.Errorf("warnings = %s, want the sentence gone: the run went and read, and the revision it reports is one Docs named", warnings)
	}
}

// The window between phase 1's last answer and the fresh read is the one place
// the run stops chaining a revision through, and it must not be a gap.
//
// Somebody editing there is carried by every batch that follows if the marker
// requires the revision the fresh read named: the marker lands, the styling
// chains off it, and phase 2 rewrites their paragraph by direct edit at the
// in-place level. So the marker is sent against phase 1's own answer, and Docs
// is what refuses it. The test is the revisions on the wire rather than the
// refusal, because the refusal is Docs' and a fake cannot make it honestly.
func TestTheMarkerIsSentAgainstTheRevisionThePreludePhaseEndedOn(t *testing.T) {
	// Arrange: the fresh read names a revision phase 1 never wrote, which is
	// somebody else's edit inside that window.
	w := preludeWire(t)
	w.answers[2] = &answer{method: "GET", match: "docs.googleapis.com", once: true,
		json: strings.Replace(readFixture(t, "single-tab.json"),
			"ALm37BXsingleTab", "ALm37BXsomebodyElseEdited", 1)}
	f := stubWire(t, w)
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	// Act
	got, _ := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)

	// Assert
	batches := batchesSent(t, f)
	if len(batches) < 2 {
		t.Fatalf("the run sent %d batches, want the prelude and the marker: %v", len(batches), got)
	}
	if batches[1].WriteControl.RequiredRevisionID != "ALm37BXafterThePrelude" {
		t.Errorf("the marker batch named revision %q, want the one phase 1's own answer named: "+
			"sent the fresh read's, an edit made in that window is carried rather than refused, "+
			"and phase 2 styles it by direct edit",
			batches[1].WriteControl.RequiredRevisionID)
	}
}

// A prelude phase that stopped left words in the document with nothing marking
// them, and the run has to say so.
//
// It is the hazard the marker failure names one block further down, arriving by
// a different route: the next run finds no marker, proposes at index 1, and
// puts a second cover in front of the first. Rejecting what landed is the
// recovery, and somebody who is not told to do it before running again gets two
// stacked covers of which only the first can be rejected.
//
// The batch Docs accepted whose answer could not be read is the shape this is
// worst in. The live prelude goes out in one batch, so on that path the whole
// unmarked prelude may be in the document while the run counts no batch at all.
func TestAPreludePhaseThatStoppedSaysToRejectBeforeRunningAgain(t *testing.T) {
	// Arrange
	w := preludeWire(t)
	w.answers[1] = &answer{method: "POST", match: ":batchUpdate",
		err: sentAnywayErr{errors.New("the answer was not JSON")}}
	stubWire(t, w)
	from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
	fields := tempFile(t, "fields.json", fieldsFile)

	// Act
	got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)

	// Assert
	if code == 0 || got["ok"] != false {
		t.Fatalf("a prelude phase that stopped: %v (exit %d), want a refusal", got, code)
	}
	warnings := strings.Join(warningsOf(t, got), " | ")
	if !strings.Contains(warnings, "before running this again") {
		t.Errorf("warnings = %s, want the sentence that says to accept or reject what landed "+
			"before running again: unsaid, the next run proposes a second prelude in front of it", warnings)
	}
}

// Every return between the prelude landing and the marker landing leaves the
// whole prelude in the document with nothing marking it, which is the state the
// next run puts a second cover in front of. Each of those returns names its own
// failure, and none of those sentences says a word about the prelude.
func TestAnUnmarkedPreludeSaysSoOnEveryPathThatLeavesOne(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(w *fakeWire)
	}{
		{"the fresh read failed", func(w *fakeWire) {
			w.answers[2] = &answer{method: "GET", match: "docs.googleapis.com",
				err: errors.New("the document could not be read"), once: true}
		}},
		{"the document grew a second tab", func(w *fakeWire) {
			w.answers[2] = &answer{method: "GET", match: "docs.googleapis.com",
				json: twoTabDocument, once: true}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			w := preludeWire(t)
			c.break_(w)
			stubWire(t, w)
			from := tempFile(t, "survey.json", surveyOf(t, fixtureDocID, "ALm37BXsingleTab", 1))
			fields := tempFile(t, "fields.json", fieldsFile)

			// Act
			got, code := runJSON(t, "restyle", fixtureDocID, "--from", from, "--fields", fields)

			// Assert
			if code == 0 || got["ok"] != false {
				t.Fatalf("a run that stopped after the prelude: %v (exit %d), want a refusal", got, code)
			}
			warnings := strings.Join(warningsOf(t, got), " | ")
			if !strings.Contains(warnings, "nothing marks it") {
				t.Errorf("warnings = %s, want the sentence saying the prelude landed unmarked: "+
					"unsaid, the next run proposes a second prelude in front of it", warnings)
			}
			if !strings.Contains(warnings, "before running this again") {
				t.Errorf("warnings = %s, want the recovery beside it", warnings)
			}
		})
	}
}
