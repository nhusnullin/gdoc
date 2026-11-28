package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"gdoc/internal/edit"
	"gdoc/internal/guard"
)

// editDoc is a one-paragraph Docs answer for the edit tests: "The policy is
// reviewed annually by the board.", with the quoted words at [15,32).
func editDoc(revision, middle, style string) string {
	run := func(start int, text, style string) string {
		if style == "" {
			style = "{}"
		}
		return fmt.Sprintf(`{"startIndex":%d,"endIndex":%d,"textRun":{"content":%q,"textStyle":%s}}`, start, start+len(text), text, style)
	}
	head, tail := "The policy is ", " by the board.\n"
	els := []string{run(1, head, ""), run(15, middle, style), run(15+len(middle), tail, "")}
	return `{"documentId":"` + proposeDocID + `","revisionId":"` + revision + `","tabs":[{"tabProperties":{"tabId":"t.0"},"documentTab":{"body":{"content":[{"paragraph":{"elements":[` +
		strings.Join(els, ",") + `]}}]}}}]}`
}

const editYellowStyle = `{"backgroundColor":{"color":{"rgbColor":{"red":1,"green":0.9607843137254902,"blue":0.615686274509804}}}}`

func editAnswers() []*answer {
	return []*answer{
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: editDoc("REV1", "reviewed annually", ""), once: true},
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: editDoc("REV2", "reviewed every six months", editYellowStyle)},
		{method: "POST", match: proposeDocID + ":batchUpdate", json: `{}`},
	}
}

// judged asks the run's own policy about every write the fake saw. The fake
// stands below the guard, so this is what puts the guard back in the path.
func judged(t *testing.T, f *fakeWire) {
	t.Helper()
	for _, c := range f.writes() {
		u, err := url.Parse(c.URL)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.policy.Judge(c.Method, u, c.Body); err != nil {
			t.Errorf("the run sent a write its own guard refuses: %v\n%s", err, c.Body)
		}
	}
}

func TestProposeEditWritesTheWordsDirectlyAndPaintsThem(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: editAnswers()})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--edit")
	if code != 0 || got["ok"] != true {
		t.Fatalf("propose --edit: %v (exit %d)", got, code)
	}
	writes := f.writes()
	if len(writes) != 1 {
		t.Fatalf("one edit is one write, and %d went out", len(writes))
	}
	body := string(writes[0].Body)
	if strings.Contains(body, "SUGGEST") || strings.Contains(body, "insertComment") {
		t.Errorf("an edit is not a suggestion and leaves no comment: %s", body)
	}
	if !strings.Contains(body, `"requiredRevisionId":"REV1"`) {
		t.Errorf("an edit is sent against the revision it read: %s", body)
	}
	judged(t, f)

	data := dataOf(t, got)
	if data["mode"] != "edit" || data["colour"] != "#FFF59D" {
		t.Errorf("the envelope says mode %v and colour %v, want edit and #FFF59D", data["mode"], data["colour"])
	}
	one := data["edits"].([]any)[0].(map[string]any)
	if one["sent"] != true || one["verified"] != true {
		t.Errorf("the edit reports %v", one)
	}
}

// TestEditModeIsOnlyTheFlagOnTheLine: the same file with no flag is a
// suggestion, and nothing in the settings or the environment changes that.
func TestEditModeIsOnlyTheFlagOnTheLine(t *testing.T) {
	t.Setenv("GDOC_EDIT", "1")
	f := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
	from := tempFile(t, "proposals.json", oneProposal)
	if got, code := runJSON(t, "propose", proposeDocID, "--from", from); code != 0 {
		t.Fatalf("propose: %v (exit %d)", got, code)
	}
	for _, c := range f.writes() {
		if !strings.Contains(string(c.Body), `"writeMode":"SUGGEST"`) {
			t.Errorf("without --edit every write is a suggestion: %s", c.Body)
		}
	}
	direct, _ := url.Parse("https://docs.googleapis.com/v1/documents/" + proposeDocID + ":batchUpdate")
	if f.policy.Judge("POST", direct, []byte(`{"requests":[{"insertText":{"location":{"index":1},"text":"x"}}]}`)) == nil {
		t.Error("without --edit the run's policy must refuse a direct edit")
	}
	for _, s := range settingFlags() {
		if strings.Contains(s.name, "edit") {
			t.Errorf("%s is a setting, and edit mode is never one", s.name)
		}
	}
}

func TestEditRefusesABlockBeforeAnythingIsSent(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: editAnswers()})
	from := tempFile(t, "proposals.json", `[{"quoted":"reviewed annually","replacement":"reviewed every six months","why":"twice a year"},`+
		`{"kind":"block","after":"The policy is reviewed annually by the board.","content":"A new paragraph.","why":"missing"}]`)
	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--edit")
	if code == 0 || got["ok"] != false || !strings.Contains(fmt.Sprint(got["error"]), "entry 2") {
		t.Fatalf("a block in an edit run must refuse the run and name its entry: %v (exit %d)", got, code)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may be read or sent when an entry is refused, and the wire saw %d calls", len(f.calls))
	}
}

func TestEditTakesNoNote(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: editAnswers()})
	from := tempFile(t, "proposals.json", oneProposal)
	note := tempFile(t, "note.md", "# A note\n")
	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--edit", "--md", note)
	if code == 0 || !strings.Contains(fmt.Sprint(got["error"]), "--md") {
		t.Fatalf("--edit with --md must be refused by name: %v (exit %d)", got, code)
	}
	if len(f.calls) != 0 {
		t.Error("nothing may be sent")
	}
	if b, _ := os.ReadFile(note); string(b) != "# A note\n" {
		t.Error("the note must be untouched")
	}
}

func TestAnEditThatDidNotLandPaintedStopsTheRun(t *testing.T) {
	answers := []*answer{
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: editDoc("REV1", "reviewed annually", ""), once: true},
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: editDoc("REV2", "reviewed every six months", "")},
		{method: "POST", match: proposeDocID + ":batchUpdate", json: `{}`},
	}
	f := stubWire(t, &fakeWire{answers: answers})
	from := tempFile(t, "proposals.json", `[{"quoted":"reviewed annually","replacement":"reviewed every six months","why":"twice a year"},`+
		`{"quoted":"the board","replacement":"the committee","why":"renamed"}]`)
	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--edit")
	if code == 0 || got["ok"] != false {
		t.Fatalf("an edit the read-back cannot confirm stops the run: %v (exit %d)", got, code)
	}
	if len(f.writes()) != 1 {
		t.Errorf("nothing after the unconfirmed edit is sent, and %d writes went out", len(f.writes()))
	}
	edits := dataOf(t, got)["edits"].([]any)
	if edits[0].(map[string]any)["sent"] != true || edits[1].(map[string]any)["sent"] != false {
		t.Errorf("the report names what landed and what never left: %v", edits)
	}
}

func TestSeenTakesTheYellowOffAndChangesNoWord(t *testing.T) {
	answers := []*answer{
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: editDoc("REV2", "reviewed every six months", editYellowStyle), once: true},
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: editDoc("REV3", "reviewed every six months", "")},
		{method: "POST", match: proposeDocID + ":batchUpdate", json: `{}`},
	}
	f := stubWire(t, &fakeWire{answers: answers})
	got, code := runJSON(t, "seen", proposeDocID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("seen: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	if data["cleared"] != float64(1) || data["left"] != float64(0) || data["verified"] != true {
		t.Errorf("seen reports %v, want one passage cleared, none left, verified", data)
	}
	writes := f.writes()
	if len(writes) != 1 {
		t.Fatalf("a clear is one write, and %d went out", len(writes))
	}
	body := string(writes[0].Body)
	for _, kind := range []string{"insertText", "deleteContentRange", "replaceAllText"} {
		if strings.Contains(body, kind) {
			t.Errorf("a clear names no character, and this one carries %s: %s", kind, body)
		}
	}
	judged(t, f)
}

func TestSeenOnADocumentWithNoYellowWritesNothing(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: []*answer{
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: editDoc("REV1", "reviewed annually", `{"backgroundColor":{"color":{"rgbColor":{"red":1,"green":0.8352941176470589}}}}`)},
	}})
	got, code := runJSON(t, "seen", proposeDocID)
	if code != 0 || dataOf(t, got)["cleared"] != float64(0) {
		t.Fatalf("seen on a document with only the house yellow clears nothing: %v (exit %d)", got, code)
	}
	if len(f.writes()) != 0 {
		t.Error("the house yellow is the author's mark, and nothing may be written over it")
	}
}

// TestEditAndSeenAreNeverOfferedInChat: Claude Desktop's chat proposes and
// never edits, so no tool carries the flag and no tool is the clear.
func TestEditAndSeenAreNeverOfferedInChat(t *testing.T) {
	if !contains(flagsNeverOffered["propose"], editFlag) {
		t.Errorf("%s must be on the list of flags chat never fills", editFlag)
	}
	for _, c := range mcpCommands() {
		if c.tool == "seen" {
			t.Error("chat has a tool named seen, and the clear is never offered there")
		}
		for _, f := range c.flags {
			if f.flag == editFlag {
				t.Errorf("the chat tool %q fills %s", c.tool, editFlag)
			}
		}
	}
}

func TestTheEditYellowIsTheGuardsYellow(t *testing.T) {
	if edit.Yellow != guard.EditYellow {
		t.Fatalf("internal/edit paints %s and the guard carries %s", edit.Yellow, guard.EditYellow)
	}
}
