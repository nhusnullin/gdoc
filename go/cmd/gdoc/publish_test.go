package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/frontmatter"
)

// publishDocID is the id the fake Drive answers the create with. It is learned
// from that answer rather than handed in, which is the second of the guard's
// two doors and the reason a publish opens with no file at all.
const publishDocID = "1PuBl15h0000000000000000000000000000000"

// publishAnswers is a whole publish as the wire sees it: the multipart create,
// the Docs read-back and the docx export.
func publishAnswers(t *testing.T) []*answer {
	t.Helper()
	return []*answer{
		{method: "POST", match: "uploadType=multipart", json: `{"id":"` + publishDocID + `"}`},
		{method: "GET", match: publishDocID + "?includeTabsContent", json: readFixture(t, "publish-read-back.json")},
		{method: "GET", match: "/export?", bytes: exportWithComment(t, false)},
	}
}

// trashAnswers is the rollback: the PATCH and the read that confirms it.
func trashAnswers() []*answer {
	return []*answer{
		{method: "PATCH", match: "files/" + publishDocID, json: `{}`},
		{method: "GET", match: "fields=trashed", json: `{"trashed":true}`},
	}
}

// blockIn reads the gdoc: block the run wrote into the note.
func blockIn(t *testing.T, path string) *frontmatter.Block {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	block, err := frontmatter.Read(src)
	if err != nil {
		t.Fatalf("the note the run wrote does not parse: %v", err)
	}
	return block
}

// mustURL is the guard test's helper, here so a command test can ask what the
// policy it opened would carry.
func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// warningText is the envelope's warnings as one string to look in.
func warningText(got map[string]any) string {
	list, _ := got["warnings"].([]any)
	out := make([]string, 0, len(list))
	for _, w := range list {
		out = append(out, fmt.Sprint(w))
	}
	return strings.Join(out, " ")
}

// multipart is the create the run made, or a failed test if it made none.
func (f *fakeWire) multipart(t *testing.T) wireCall {
	t.Helper()
	for _, c := range f.calls {
		if c.PartType != "" {
			return c
		}
	}
	t.Fatalf("the run made no multipart create: %v", f.calls)
	return wireCall{}
}

func TestPublishUploadsTheNoteAndPairsIt(t *testing.T) {
	stubWire(t, &fakeWire{answers: publishAnswers(t)})
	_, md := buildNote(t)

	got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("publish: %v (exit %d)", got, code)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	if data["document_id"] != publishDocID {
		t.Errorf("document_id is the id the create answered with: %v", data["document_id"])
	}
	if data["folder_id"] != testFolderID {
		t.Errorf("folder_id = %v, want the folder the run was given", data["folder_id"])
	}
	if data["url"] != "https://docs.google.com/document/d/"+publishDocID+"/edit" {
		t.Errorf("url is where a person opens it: %v", data["url"])
	}
	if data["title"] != "Supplier Register Policy" {
		t.Errorf("title is what the read-back carried: %v", data["title"])
	}
	if data["house"] != "embedded" {
		t.Errorf("house names the style the run built from: %v", data["house"])
	}
	if size, _ := data["bytes"].(float64); size <= 0 {
		t.Errorf("bytes is the size of what was uploaded: %v", data["bytes"])
	}
	if data["verified"] != true {
		t.Errorf("verified is the three read-backs together: %v, warnings %v", data["verified"], got["warnings"])
	}
	checks, _ := data["checks"].(map[string]any)
	for _, name := range []string{"read_back", "one_tab", "docx_export"} {
		if checks[name] != true {
			t.Errorf("checks.%s = %v", name, checks[name])
		}
	}
	if _, said := data["rolled_back"]; said {
		t.Errorf("a run that recorded the pairing rolled nothing back: %v", data["rolled_back"])
	}
	counts, _ := data["body"].(map[string]any)
	if counts["headings"] != float64(3) {
		t.Errorf("body carries the walker's counts: %v", data["body"])
	}
	changed, _ := data["files_changed"].([]any)
	if len(changed) != 1 || changed[0] != md {
		t.Errorf("files_changed names the note: %v, want [%s]", data["files_changed"], md)
	}

	block := blockIn(t, md)
	if block == nil {
		t.Fatal("the note carries no gdoc: block after a publish")
	}
	if block.DocumentID != publishDocID || block.FolderID != testFolderID {
		t.Errorf("the block records the pairing: %+v", block)
	}
	if block.Published == nil {
		t.Fatal("the block carries no publish record")
	}
	if block.Published.At.IsZero() {
		t.Error("published.at is when the document was made")
	}
	if block.Published.Title != "Supplier Register Policy" {
		t.Errorf("published.title is what went on the cover: %q", block.Published.Title)
	}
	if block.Published.House != "embedded" {
		t.Errorf("published.house is where the style came from: %q", block.Published.House)
	}
}

// The metadata part is what turns a docx into a document, and it is the only
// thing that names the folder. The file part is the bytes the render wrote.
func TestPublishUploadsTheMetadataAndTheDocx(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: publishAnswers(t)})
	_, md := buildNote(t)

	if got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID); code != 0 {
		t.Fatalf("publish: %v (exit %d)", got, code)
	}
	call := f.multipart(t)
	var meta struct {
		Name     string   `json:"name"`
		MimeType string   `json:"mimeType"`
		Parents  []string `json:"parents"`
	}
	if err := json.Unmarshal(call.Body, &meta); err != nil {
		t.Fatalf("the metadata part is not JSON: %v", err)
	}
	if meta.Name != "Supplier Register Policy" {
		t.Errorf("the metadata names the document after the cover title: %q", meta.Name)
	}
	if meta.MimeType != "application/vnd.google-apps.document" {
		t.Errorf("the metadata asks for a Google Doc: %q", meta.MimeType)
	}
	if len(meta.Parents) != 1 || meta.Parents[0] != testFolderID {
		t.Errorf("the metadata names exactly the granted folder: %v", meta.Parents)
	}
	if _, err := zip.NewReader(bytes.NewReader(call.Part), int64(len(call.Part))); err != nil {
		t.Errorf("the file part is not a docx: %v", err)
	}
}

// No file is in the reachable set when the run starts: the document's own id is
// learned from the create the guard carried, and nothing else is reachable at
// all.
func TestPublishOpensWithTheFolderAndNoFile(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: publishAnswers(t)})
	_, md := buildNote(t)

	if got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID); code != 0 {
		t.Fatalf("publish: %v (exit %d)", got, code)
	}
	if f.policy == nil {
		t.Fatal("the command opened no policy")
	}
	u := mustURL(t, docs.URL(publishDocID))
	if err := f.policy.Judge("GET", u, nil); err == nil {
		t.Error("a publish opens with no file in the reachable set, and this one had one")
	}
}

// Publish makes a document. A note that already names one is refused before
// anything leaves the machine, because there is no republish.
func TestPublishRefusesANoteThatIsAlreadyPaired(t *testing.T) {
	noSession(t)
	md := copyFixture(t, "paired.md")

	got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a paired note must not be published again: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "already") {
		t.Errorf("the error must say the note is already paired: %q", msg)
	}
}

func TestPublishArgumentsAreStrict(t *testing.T) {
	_, md := buildNote(t)
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"an unknown flag", []string{"publish", "--md", md, "--folder-id", testFolderID, "--dry-run"}, "--dry-run"},
		{"a repeated flag", []string{"publish", "--md", md, "--md", md, "--folder-id", testFolderID}, "twice"},
		{"a missing value", []string{"publish", "--md"}, "needs a value"},
		{"an empty value", []string{"publish", "--md="}, "empty value"},
		{"a flag standing where a value belongs", []string{"publish", "--md", "--folder-id", testFolderID}, "another flag"},
		{"an extra positional", []string{"publish", "--md", md, "--folder-id", testFolderID, "note"}, "extra argument"},
		{"no --md", []string{"publish", "--folder-id", testFolderID}, "--md"},
		{"no --folder-id", []string{"publish", "--md", md}, "--folder-id"},
		{"a document URL as the folder", []string{"publish", "--md", md, "--folder-id",
			"https://docs.google.com/document/d/" + publishDocID + "/edit"}, "document, not a folder"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := stubWire(t, &fakeWire{})
			got, code := runJSON(t, c.args...)
			if code == 0 || got["ok"] != false {
				t.Fatalf("%s must be refused: %v (exit %d)", c.name, got, code)
			}
			if msg, _ := got["error"].(string); !strings.Contains(msg, c.want) {
				t.Errorf("the error must name the offender %q: %q", c.want, msg)
			}
			if len(f.calls) != 0 {
				t.Errorf("nothing may be sent for a run that was refused: %v", f.calls)
			}
		})
	}
}

// The four re-read refusals. Each one is a document in Drive that the note
// cannot be paired to, so each one takes the document back.
func TestPublishRollsBackWhenTheNoteCannotBePaired(t *testing.T) {
	cases := []struct {
		name  string
		spoil func(t *testing.T, md string)
		want  string
	}{
		{
			name: "another run paired it while this one was uploading",
			spoil: func(t *testing.T, md string) {
				src, err := os.ReadFile(md)
				if err != nil {
					t.Fatal(err)
				}
				out, err := frontmatter.Write(src, &frontmatter.Block{
					Schema: frontmatter.Schema, DocumentID: proposeDocID})
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(md, out, 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: proposeDocID,
		},
		{
			name: "the note changed under the render",
			spoil: func(t *testing.T, md string) {
				f, err := os.OpenFile(md, os.O_APPEND|os.O_WRONLY, 0o644)
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				if _, err := f.WriteString("\nA sentence the render never saw.\n"); err != nil {
					t.Fatal(err)
				}
			},
			want: "changed",
		},
		{
			name: "the note could not be read again",
			spoil: func(t *testing.T, md string) {
				if err := os.Remove(md); err != nil {
					t.Fatal(err)
				}
			},
			want: "could not be read again",
		},
		{
			// The front matter broke while the upload was running. Its own
			// sentence says the note no longer reads, rather than the one about
			// a block that could not be written: a note gdoc cannot parse is
			// one it never got as far as writing into.
			name: "the note's front matter no longer parses",
			spoil: func(t *testing.T, md string) {
				if err := os.WriteFile(md, []byte("---\ntitle: \"unterminated\n---\n\nA body.\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "no longer reads",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, md := buildNote(t)
			answers := publishAnswers(t)
			answers[0].before = func() { c.spoil(t, md) }
			f := stubWire(t, &fakeWire{answers: append(answers, trashAnswers()...)})

			got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID)
			if code == 0 || got["ok"] != false {
				t.Fatalf("a note that cannot be paired is not a clean publish: %v (exit %d)", got, code)
			}
			if msg, _ := got["error"].(string); !strings.Contains(msg, c.want) {
				t.Errorf("the error must say what stopped the pairing (%q): %q", c.want, msg)
			}
			data, _ := got["data"].(map[string]any)
			if data["rolled_back"] != true {
				t.Errorf("rolled_back = %v, want true: the document was taken back", data["rolled_back"])
			}
			if id, said := data["document_id"]; said && id != "" {
				t.Errorf("a document that was taken back is not an id to report: %v", id)
			}
			if _, said := data["files_changed"]; said {
				t.Errorf("nothing was written into the note: %v", data["files_changed"])
			}
			var patched bool
			for _, call := range f.calls {
				if call.Method == "PATCH" && strings.Contains(call.URL, publishDocID) {
					patched = true
				}
			}
			if !patched {
				t.Errorf("the document was not trashed: %v", f.calls)
			}
		})
	}
}

// Not knowing must never resolve to the document being gone. A trash that could
// not be confirmed comes back with the live id and the steps.
func TestPublishReportsTheLiveIDWhenTheRollbackFailed(t *testing.T) {
	_, md := buildNote(t)
	answers := publishAnswers(t)
	answers[0].before = func() {
		if err := os.Remove(md); err != nil {
			t.Fatal(err)
		}
	}
	answers = append(answers, &answer{method: "PATCH", match: "files/" + publishDocID, json: `{}`},
		&answer{method: "GET", match: "fields=trashed", json: `{"trashed":false}`})
	f := stubWire(t, &fakeWire{answers: answers})

	got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a rollback that did not hold is not a clean run: %v (exit %d)", got, code)
	}
	data, _ := got["data"].(map[string]any)
	if data["rolled_back"] != false {
		t.Errorf("rolled_back = %v, want false", data["rolled_back"])
	}
	if data["document_id"] != publishDocID {
		t.Errorf("document_id = %v: a document still in Drive is named", data["document_id"])
	}
	warns := warningText(got)
	if !strings.Contains(warns, publishDocID) || !strings.Contains(warns, "by hand") {
		t.Errorf("the warnings must name the document to delete by hand: %q", warns)
	}
	if len(f.calls) == 0 {
		t.Fatal("nothing was sent")
	}
}

// A create that failed is a run with no document, so there is nothing to verify,
// to record or to take back, and the note is left alone.
func TestPublishReportsAnUploadThatFailed(t *testing.T) {
	_, md := buildNote(t)
	before, err := os.ReadFile(md)
	if err != nil {
		t.Fatal(err)
	}
	f := stubWire(t, &fakeWire{answers: []*answer{
		{method: "POST", match: "uploadType=multipart", json: `{}`},
	}})

	got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a create with no id is not a publish: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, testFolderID) {
		t.Errorf("with no id to name, the error names the folder: %q", msg)
	}
	data, _ := got["data"].(map[string]any)
	if id, said := data["document_id"]; said && id != "" {
		t.Errorf("there is no document id to report: %v", id)
	}
	after, err := os.ReadFile(md)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("a run with no document must not write into the note")
	}
	for _, call := range f.calls {
		if call.Method == "PATCH" {
			t.Errorf("there is nothing to trash: %v", call)
		}
	}
}

// A read-back that did not hold is a document that exists. It is reported with
// the route named, and the pairing is still recorded: a caller told the run
// failed is a caller that publishes a second one.
func TestPublishPairsTheNoteEvenWhenARouteDidNotHold(t *testing.T) {
	_, md := buildNote(t)
	answers := publishAnswers(t)
	answers[2] = &answer{method: "GET", match: "/export?", bytes: []byte("not a docx")}
	stubWire(t, &fakeWire{answers: answers})

	got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("a document that exists is a document that exists: %v (exit %d)", got, code)
	}
	data, _ := got["data"].(map[string]any)
	if data["verified"] != false {
		t.Errorf("verified = %v, want false: the export did not read as a docx", data["verified"])
	}
	checks, _ := data["checks"].(map[string]any)
	if checks["docx_export"] != false || checks["read_back"] != true {
		t.Errorf("the checks name the route that did not hold: %v", data["checks"])
	}
	if warns := warningText(got); !strings.Contains(warns, "docx") {
		t.Errorf("the warnings must name the route: %q", warns)
	}
	if block := blockIn(t, md); block == nil || block.DocumentID != publishDocID {
		t.Errorf("the pairing is recorded whatever the read-backs said: %+v", block)
	}
}

// One render, two commands. build writes the bytes to a file and publish
// uploads them, and if the two ever drifted a document would stop being what
// the note builds to.
func TestBuildAndPublishRenderTheSameBytes(t *testing.T) {
	dir, md := buildNote(t)
	out := filepath.Join(dir, "supplier-register.docx")
	noSession(t)
	if got, code := runJSON(t, "build", "--md", md, "--out", out); code != 0 {
		t.Fatalf("build: %v (exit %d)", got, code)
	}
	built, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	f := stubWire(t, &fakeWire{answers: publishAnswers(t)})
	if got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID); code != 0 {
		t.Fatalf("publish: %v (exit %d)", got, code)
	}
	if uploaded := f.multipart(t).Part; !bytes.Equal(built, uploaded) {
		t.Errorf("publish uploaded %d bytes and build wrote %d: the two commands share one render",
			len(uploaded), len(built))
	}
}

// A document published from a draft style says which file it came from, on the
// envelope and in the note. That is the record of what a document was built
// from, and "embedded" would be the wrong answer for a run under review.
func TestPublishReportsTheHouseFileItWasGiven(t *testing.T) {
	stubWire(t, &fakeWire{answers: publishAnswers(t)})
	dir, md := buildNote(t)
	src, err := os.ReadFile(filepath.Join("..", "..", "internal", "house", "house.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	style := filepath.Join(dir, "draft-house.yaml")
	if err := os.WriteFile(style, src, 0o644); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID, "--house", style)
	if code != 0 || got["ok"] != true {
		t.Fatalf("publish with --house: %v (exit %d)", got, code)
	}
	data, _ := got["data"].(map[string]any)
	if data["house"] != style {
		t.Errorf("house = %v, want %s", data["house"], style)
	}
	block := blockIn(t, md)
	if block == nil || block.Published == nil || block.Published.House != style {
		t.Errorf("published.house records the style the document was built from: %+v", block)
	}
}
