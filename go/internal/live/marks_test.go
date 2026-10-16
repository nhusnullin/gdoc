// M18's acceptance for Part B: the seven marks a person wrote in a note are
// the seven colours Docs reports on the runs of the document a colleague
// opens.
//
// The probe beside this, in markprobe_test.go, asked whether Drive keeps a
// shaded run and a coloured run at all, and it uploaded a fixture a person had
// edited by hand. This one asks the question the milestone is judged on: the
// note goes through the production render, the production upload and the Docs
// read, and nothing in the path is written here.
//
// Unlike the probe it asserts, because the answers are gdoc's now, and it
// states every hex as a literal rather than reading house.yaml's palette: a
// test that read the colour it checks would follow the palette wherever
// somebody moved it and say nothing about what was published.

package live

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// publishedMark is one marked span of the kitchen sink: the house mark, the
// words the run carries, which of the two colours the mark is written as, and
// the hex it has to come back as.
//
// The words are the run's own, the marked span with its colour emoji taken off
// and its soft wrap joined, which is what internal/body writes and what
// internal/body/testdata/golden/01-kitchen-sink.xml holds.
type publishedMark struct {
	mark  string
	words string
	kind  string
	hex   string
}

// The seven, in the order the note writes them. Six are run shading, which
// Docs reports as a background, and the pencil is the run's own text colour,
// which Docs reports as a foreground.
var publishedMarks = []publishedMark{
	{"yellow", "a figure to replace", "background", "#FFD500"},
	{"green", "a claim to confirm", "background", "#3DD67A"},
	{"purple", "a question for one reviewer", "background", "#BA92FF"},
	{"blue", "a line changed since the last version", "background", "#5DB3FF"},
	{"red", "a warning", "background", "#FF6B6B"},
	{"orange", "an aside", "background", "#FFB85C"},
	{"pencil", "a line that stays in the document but is not the argument", "foreground", "#9A9A9A"},
}

// TestLivePublishCarriesTheMarksIntoDocs is the second half of acceptance 6
// of the specification: the note's seven marks, end to end.
//
// It is the same publish as the footnote acceptance beside it, over the same
// note through the same path, and it makes its own document rather than
// sharing one: a document two tests hold is a document neither owns, and the
// second one to run would be reading what the first left. One document,
// trashed on the way out.
func TestLivePublishCarriesTheMarksIntoDocs(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	path := filepath.Join(bodyNotes, kitchenSinkNote)
	built := renderNoteFile(t, path)
	title := "gdoc M18 marks acceptance " + time.Now().UTC().Format(time.RFC3339)
	id := uploadForDrift(t, ctx, s, folder, title, built.docx)

	runs := runsWithColour(t, ctx, s, id, "the published document")

	for _, m := range publishedMarks {
		run, found := runCarrying(runs, m.words)
		if !found {
			t.Errorf("the %s mark's words are in no run of the document: %q", m.mark, m.words)
			continue
		}
		got := run.background
		if m.kind == "foreground" {
			got = run.foreground
		}
		if got != m.hex {
			t.Errorf("the %s mark came back with %s %s, want %s; the run reads %q, background %s, foreground %s",
				m.mark, m.kind, got, m.hex, strings.TrimSpace(run.text), run.background, run.foreground)
			continue
		}
		t.Logf("the %s mark: %s %s on a run reading %q", m.mark, m.kind, got, strings.TrimSpace(run.text))
	}
}

// runCarrying is the run one marked span's words are in: the run that is those
// words first, and a longer run holding them second.
//
// Both answers are an answer, because a run Docs joined to its neighbour still
// carries one colour. The order is the probe's and for the probe's reason: a
// single pass on "somewhere in the run" could answer about a longer run where
// the marked one was asked for.
func runCarrying(runs []probeRun, words string) (probeRun, bool) {
	for _, r := range runs {
		if strings.TrimSpace(r.text) == words {
			return r, true
		}
	}
	for _, r := range runs {
		if strings.Contains(r.text, words) {
			return r, true
		}
	}
	return probeRun{}, false
}
