package main

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

const annotateDocID = "1AnNoTaTe0000000000000000000000000000000"

// The two bodies the fixtures carry, written out as literals. The prefix is the
// robot and one space, spelled here rather than read from internal/plaintext:
// a test that read the constant would follow it wherever somebody moved it, and
// what this test is about is what a reader of the document sees.
const (
	firstBody  = "🤖 The 2026 register says quarterly."
	secondBody = "🤖 Named twice with two spellings."
)

// annotateExport is the docx both read-backs are confirmed against: the two
// comments gdoc wrote, each attached to the words it was asked to comment on.
func annotateExport(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"word/document.xml": readFixture(t, "annotate-document.xml"),
		"word/comments.xml": readFixture(t, "annotate-comments.xml"),
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

// annotateAnswers is what the wire says to a run of up to two annotations: the
// read each one makes for itself, the two batches in order, and the two
// read-backs.
//
// The batch answers are spent one at a time and carry different comment ids, so
// the second annotation is not confirmed on the strength of the first one's
// listing entry. Everything else is answered as often as it is asked, because
// each annotation reads the same unchanged document: a comment moves no
// character.
func annotateAnswers(t *testing.T) []*answer {
	t.Helper()
	second := strings.Replace(readFixture(t, "annotate-batch.json"), "AAACOne", "AAACTwo", 1)
	return []*answer{
		{method: "POST", match: annotateDocID + ":batchUpdate", json: readFixture(t, "annotate-batch.json"), once: true},
		{method: "POST", match: annotateDocID + ":batchUpdate", json: second, once: true},
		{method: "GET", match: annotateDocID + "?includeTabsContent", json: readFixture(t, "annotate-before.json")},
		{method: "GET", match: "/comments?", json: readFixture(t, "annotate-comments.json")},
		{method: "GET", match: "/export?", bytes: annotateExport(t)},
	}
}

// annotationsOf is the list the envelope carries, one entry per annotation the
// run was handed.
func annotationsOf(t *testing.T, got map[string]any) []map[string]any {
	t.Helper()
	raw, _ := dataOf(t, got)["annotations"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, one := range raw {
		entry, _ := one.(map[string]any)
		out = append(out, entry)
	}
	return out
}

// The two input forms are two calls. A run naming both is a caller who wrote a
// file and then quoted something else on the command line, and running either
// half would write a comment nobody asked for.
func TestAnnotateRefusesFromBesideQuote(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	from := tempFile(t, "annotations.json", readFixture(t, "annotations.json"))
	body := tempFile(t, "why.txt", "The 2026 register says quarterly.\n")

	got, code := runJSON(t, "annotate", annotateDocID,
		"--quote", "reviewed annually", "--body-file", body, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("--from beside --quote must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "--quote") || !strings.Contains(msg, "--from") {
		t.Errorf("the refusal must name both flags: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach the wire before the call is understood: %v", f.calls)
	}
}

// --body-file beside --from is refused by the arm that names it, and not by the
// one below that arm. The one below says --quote is what says which words the
// reason goes on, and adding --quote is the one change that makes this run
// worse: it is then a run naming both forms at once, refused a second time.
func TestAnnotateRefusesFromBesideBodyFile(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	from := tempFile(t, "annotations.json", readFixture(t, "annotations.json"))
	body := tempFile(t, "why.txt", "The 2026 register says quarterly.\n")

	got, code := runJSON(t, "annotate", annotateDocID, "--from", from, "--body-file", body)
	if code == 0 || got["ok"] != false {
		t.Fatalf("--body-file beside --from must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "--from") || !strings.Contains(msg, "drop --body-file") {
		t.Errorf("the refusal must name the real mistake and the flag to drop: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach the wire before the call is understood: %v", f.calls)
	}
}

// The by-hand form is a pair. One half of it names words with no reason, and
// the other names a reason with nothing to put it on, and neither is a comment.
func TestAnnotateNeedsAQuoteWithItsBodyFile(t *testing.T) {
	body := tempFile(t, "why.txt", "The 2026 register says quarterly.\n")

	for _, one := range []struct {
		args []string
		name string
	}{
		{[]string{"--quote", "reviewed annually"}, "--body-file"},
		{[]string{"--body-file", body}, "--quote"},
		{nil, "--from"},
	} {
		f := stubWire(t, &fakeWire{})
		got, code := runJSON(t, append([]string{"annotate", annotateDocID}, one.args...)...)
		if code == 0 || got["ok"] != false {
			t.Errorf("%v must be refused: %v (exit %d)", one.args, got, code)
			continue
		}
		if msg, _ := got["error"].(string); !strings.Contains(msg, one.name) {
			t.Errorf("%v must be refused naming %s: %q", one.args, one.name, msg)
		}
		if len(f.calls) != 0 {
			t.Errorf("%v reached the wire: %v", one.args, f.calls)
		}
	}
}

// A key the file spells wrongly is a key whose value never arrives. `assignee`
// is the one that would pass unnoticed: a dropped one lands a comment assigned
// to nobody, verified and warned about by nothing.
func TestAnnotateRefusesAnUnknownKeyInTheFile(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	from := tempFile(t, "annotations.json",
		`[{"quoted":"reviewed annually","why":"The 2026 register says quarterly.","assignedTo":"x@altery.com"}]`)

	got, code := runJSON(t, "annotate", annotateDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an unknown key must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "assignedTo") {
		t.Errorf("the refusal must name the key it did not know: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach the wire over a file gdoc half understood: %v", f.calls)
	}
}

// A file with nothing in it is a run that would open a session, read somebody's
// document and write nothing. Saying so is the honest answer.
func TestAnnotateRefusesAnEmptyList(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	from := tempFile(t, "annotations.json", `[]`)

	got, code := runJSON(t, "annotate", annotateDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an empty list must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "no annotations") {
		t.Errorf("the refusal must say the file holds nothing to write: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach the wire for a run with nothing to write: %v", f.calls)
	}
}

// The binary puts the robot in front of the reason, so a reason that already
// carries one lands signed twice. reply requires the mark and annotate adds it,
// and a caller carrying the reply convention over is exactly who this catches.
func TestAnnotateRefusesARobotInTheWhyBeforeAnyRequest(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: annotateAnswers(t)})
	from := tempFile(t, "annotations.json",
		`[{"quoted":"reviewed annually","why":"🤖 The 2026 register says quarterly."}]`)

	got, code := runJSON(t, "annotate", annotateDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a reason carrying the robot must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "gdoc adds it") {
		t.Errorf("the refusal must say gdoc writes the mark itself: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("the refusal must come before the first request: %v", f.calls)
	}
}

// A Docs thread renders markdown literally, so a reason written in it arrives
// as typed and stays there. The refusal names the mark it found.
func TestAnnotateRefusesMarkdownBeforeAnyRequest(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: annotateAnswers(t)})
	from := tempFile(t, "annotations.json",
		`[{"quoted":"reviewed annually","why":"The **2026** register says quarterly."}]`)

	got, code := runJSON(t, "annotate", annotateDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("markdown in the reason must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "**") {
		t.Errorf("the refusal must name the mark: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("the refusal must come before the first request: %v", f.calls)
	}
}

// The file is checked as a batch and not entry by entry as it is sent. An entry
// the run would refuse in the middle is refused now: the first comment would
// otherwise be in somebody's document when the run stops on the second.
func TestAnnotateRefusesASecondBadEntryBeforeAnyRequest(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: annotateAnswers(t)})
	from := tempFile(t, "annotations.json",
		`[{"quoted":"reviewed annually","why":"The 2026 register says quarterly."},`+
			`{"quoted":"named twice","why":"Spelled **two** ways."}]`)

	got, code := runJSON(t, "annotate", annotateDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a file whose second entry is malformed must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "annotations[1]") {
		t.Errorf("the refusal must name the entry it refused: %q", msg)
	}
	if !strings.Contains(msg, "**") {
		t.Errorf("the refusal must name the mark it found: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("the first entry must not be written before the second is checked: %v", f.calls)
	}
}

// The whole of the file form: two entries, two batches in the file's order,
// each one read back through both routes, and one entry per annotation in the
// envelope.
func TestAnnotatePlacesEachEntryAndVerifiesIt(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: annotateAnswers(t)})
	from := tempFile(t, "annotations.json", readFixture(t, "annotations.json"))

	got, code := runJSON(t, "annotate", annotateDocID, "--from", from)
	if code != 0 || got["ok"] != true {
		t.Fatalf("two annotations that landed: %v (exit %d)", got, code)
	}
	list := annotationsOf(t, got)
	if len(list) != 2 {
		t.Fatalf("annotations = %v, want one entry per annotation in the file", list)
	}
	for i, want := range []struct {
		quoted string
		id     string
	}{
		{"reviewed annually", "AAACOne"},
		{"the Cyprus entity", "AAACTwo"},
	} {
		entry := list[i]
		if entry["quoted"] != want.quoted || entry["sent"] != true {
			t.Errorf("annotations[%d] = %v, want %q sent", i, entry, want.quoted)
		}
		if entry["comment_id"] != want.id {
			t.Errorf("annotations[%d] comment_id = %v, want %q", i, entry["comment_id"], want.id)
		}
		if entry["comment_update_state"] != "ALL_SAVED" {
			t.Errorf("annotations[%d] comment_update_state = %v", i, entry["comment_update_state"])
		}
		if entry["verified"] != true {
			t.Errorf("annotations[%d] verified = %v, and both routes hold: %v", i, entry["verified"], got["warnings"])
		}
		checks, _ := entry["checks"].(map[string]any)
		if checks["drive_listing"] != true || checks["docx_anchored"] != true {
			t.Errorf("annotations[%d] checks = %v, want both routes holding", i, checks)
		}
	}

	// The batches are what the document really got: one insertComment each,
	// under SUGGEST, carrying the body the reader sees.
	writes := f.writes()
	if len(writes) != 2 {
		t.Fatalf("%d writes, want one batch per annotation: %v", len(writes), writes)
	}
	for i, want := range []string{firstBody, secondBody} {
		body := string(writes[i].Body)
		if !strings.Contains(body, `"insertComment"`) || !strings.Contains(body, `"SUGGEST"`) {
			t.Errorf("write %d is not one suggested comment: %s", i, body)
		}
		if strings.Contains(body, "insertText") || strings.Contains(body, "deleteContentRange") {
			t.Errorf("write %d carries a request that can move a character: %s", i, body)
		}
		if !strings.Contains(body, want) {
			t.Errorf("write %d does not carry %q: %s", i, want, body)
		}
	}
}

// The by-hand form is one annotation: the words on the command line and the
// reason in a file, with the trailing newline the file ends with taken off, so
// the read-back compares the words that were sent.
func TestAnnotateByHandPlacesOne(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: annotateAnswers(t)})
	body := tempFile(t, "why.txt", "The 2026 register says quarterly.\n")

	got, code := runJSON(t, "annotate", annotateDocID, "--quote", "reviewed annually", "--body-file", body)
	if code != 0 || got["ok"] != true {
		t.Fatalf("one annotation by hand: %v (exit %d)", got, code)
	}
	list := annotationsOf(t, got)
	if len(list) != 1 {
		t.Fatalf("annotations = %v, want one", list)
	}
	if list[0]["quoted"] != "reviewed annually" || list[0]["sent"] != true || list[0]["verified"] != true {
		t.Errorf("annotations[0] = %v, want the quote sent and verified: %v", list[0], got["warnings"])
	}
	writes := f.writes()
	if len(writes) != 1 {
		t.Fatalf("%d writes, want one: %v", len(writes), writes)
	}
	if !strings.Contains(string(writes[0].Body), firstBody) {
		t.Errorf("the comment must carry the reason with the robot in front: %s", writes[0].Body)
	}
}

// A run that stops in the middle still reports every annotation the file held.
// A shortened list makes the skill match the envelope back against the file it
// wrote to work out what happened to the rest.
func TestAnnotateStopsAtTheFirstEntryThatCannotBeSent(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: annotateAnswers(t)})
	from := tempFile(t, "annotations.json",
		`[{"quoted":"reviewed annually","why":"The 2026 register says quarterly."},`+
			`{"quoted":"reviewed monthly","why":"Named twice with two spellings."}]`)

	got, code := runJSON(t, "annotate", annotateDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an annotation that could not be placed must fail the run: %v (exit %d)", got, code)
	}
	list := annotationsOf(t, got)
	if len(list) != 2 {
		t.Fatalf("annotations = %v, want one entry per annotation in the file", list)
	}
	if list[0]["sent"] != true || list[0]["comment_id"] != "AAACOne" {
		t.Errorf("the first annotation landed and reports %v", list[0])
	}
	if list[1]["sent"] != false || list[1]["quoted"] != "reviewed monthly" {
		t.Errorf("the second annotation never left and reports %v", list[1])
	}
	// The error is the placement refusal naming the words, not whatever else
	// might have stopped the run: an assertion on the report shape alone passes
	// just as happily when the second entry never got as far as the lookup.
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, `"reviewed monthly"`) || !strings.Contains(msg, "was not found in the document") {
		t.Errorf("error = %q, want the refusal naming the quote that could not be placed", msg)
	}
	if len(f.writes()) != 1 {
		t.Errorf("%d writes, and only the first annotation may have been sent: %v", len(f.writes()), f.writes())
	}
}
