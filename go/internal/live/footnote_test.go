// M18's acceptance for Part A: a footnote a person wrote in a note is a
// footnote in the document a colleague opens, and it is still one in the file
// the export brings home.
//
// The probe beside this, in footnoteprobe_test.go, asked whether Drive keeps a
// Word footnote at all, and it uploaded a fixture a person had edited by hand.
// This one asks the question the milestone is judged on: the note goes through
// the production render, the production upload, the Docs read and the docx
// export, and nothing in the path is written here.
//
// Unlike the probe it asserts, because the answers are gdoc's now. Two marks
// in the note are two references in the body, two notes in the footnotes map,
// and the words of both are the note's own.

package live

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/docx"
)

// kitchenSinkNote is the note that carries both a footnote and every one of
// the seven marks, so one publish answers for both halves of the milestone. It
// is the first of roundTripNotes, and
// TestTheRoundTripNoteCarriesAFootnoteAndEveryMark pins that it still holds
// them.
const kitchenSinkNote = "01-kitchen-sink.md"

// The two notes under the kitchen sink's page, in the order the document
// numbers them, which is the order the marks stand in the prose rather than
// the order the definitions are written.
//
// They are literals here and the second one carries the typographer's
// apostrophe, because that is the character the generator really writes: a
// test that read the words out of the note would pass on a document holding
// the straight quote the note holds and say nothing about what was published.
const (
	kitchenFirstNote  = "The master document the drift gate measures this build against."
	kitchenSecondNote = "Principle 4, every word costs a reader’s attention."
)

// kitchenFootnotes is how many notes the kitchen sink has: two marks, two
// definitions, two notes.
const kitchenFootnotes = 2

// TestLivePublishCarriesAFootnoteIntoDocs is the first half of acceptance 6
// of the specification: the note's footnote, end to end.
//
// It publishes the kitchen sink into the test folder, reads the document back
// bare, asserts the references and the notes Drive made of it, exports it and
// asserts the footnotes part is in the file. One document, trashed on the way
// out, so nothing of Nail's is in the reachable set.
func TestLivePublishCarriesAFootnoteIntoDocs(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	path := filepath.Join(bodyNotes, kitchenSinkNote)
	built := renderNoteFile(t, path)
	title := "gdoc M18 footnote acceptance " + time.Now().UTC().Format(time.RFC3339)
	id := uploadForDrift(t, ctx, s, folder, title, built.docx)

	var answer map[string]any
	if err := s.GetJSON(ctx, driftReadURL(id), &answer); err != nil {
		t.Fatalf("the published document (%s) could not be read back: %v", id, err)
	}
	if len(answer) == 0 {
		t.Fatalf("the published document (%s) came back as an empty answer", id)
	}

	// The body: one reference per mark the note wrote.
	if got := countKey(answer["body"], "footnoteReference"); got != kitchenFootnotes {
		t.Errorf("the body holds %d footnoteReference elements, and the note marked %d footnotes", got, kitchenFootnotes)
	}

	// The notes themselves, and the words of both.
	footnotes, ok := answer["footnotes"].(map[string]any)
	if !ok {
		t.Fatalf("the document carries no footnotes map at all, so the notes did not survive the publish")
	}
	if len(footnotes) != kitchenFootnotes {
		t.Errorf("the document carries %d footnotes, and the note wrote %d", len(footnotes), kitchenFootnotes)
	}
	words := strings.Join(stringsIn(answer["footnotes"]), " ")
	for _, want := range []string{kitchenFirstNote, kitchenSecondNote} {
		if !strings.Contains(words, want) {
			t.Errorf("no note in the document reads %q; what the map holds reads %q", want, words)
		}
	}

	// And the file a colleague gets back out of Drive.
	raw, err := docx.Export(ctx, s, id)
	if err != nil {
		t.Fatalf("the docx export of %s failed: %v", id, err)
	}
	z, err := openZip(raw)
	if err != nil {
		t.Fatalf("the exported docx of %s could not be opened: %v", id, err)
	}
	names := partNames(z)
	held := false
	for _, name := range names {
		if name == footnotesPart {
			held = true
		}
	}
	if !held {
		t.Fatalf("the export holds no %s; its %d parts are %s", footnotesPart, len(names), strings.Join(names, ", "))
	}
	part, err := zipPart(z, footnotesPart)
	if err != nil {
		t.Fatalf("the exported %s could not be read: %v", footnotesPart, err)
	}
	for _, want := range []string{kitchenFirstNote, kitchenSecondNote} {
		if !strings.Contains(string(part), want) {
			t.Errorf("the exported %s does not carry %q", footnotesPart, want)
		}
	}
	t.Logf("%d footnotes published, read back and exported, in %d parts", len(footnotes), len(names))
}
