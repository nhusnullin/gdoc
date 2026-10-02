package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// This file is the bound on the probe leaving `propose`. Nothing asks Google up
// front whether SUGGEST is honoured today, so a day when it is not lands one
// proposal before a read-back sees it. The run stops there: the first proposal
// whose inline read-back or preview read-back cannot confirm a suggestion is the
// last thing sent. The 2026-10-02 entry in docs/v2/DECISIONS.md holds the
// decision.
//
// The three proposals below quote the same words, because the fake answers every
// read with the same fixture and the document never moves under the run. What
// these tests are about is how far the loop got, not which words it carried.
const threeProposals = `[` +
	`{"quoted":"reviewed annually","replacement":"reviewed every six months","why":"the policy says twice a year"},` +
	`{"quoted":"reviewed annually","replacement":"reviewed every six months","why":"the auditor reads this register"},` +
	`{"quoted":"reviewed annually","replacement":"reviewed every six months","why":"the contract names a half year"}]`

// runFixtures is a propose run of several proposals as the fake answers it.
//
// The preview and the export answers come first in the list and are never spent,
// because find takes the first unused answer whose match string is in the URL and
// the inline URL is a prefix of the preview's. The inline answers are the ones
// that carry the order: the command's own read, then a read and a read-back per
// proposal.
type runFixtures struct {
	proposals int
	preview   string
	export    []byte
	// firstReadBack stands in for the first proposal's inline read-back. A nil
	// one is the honest fixture, where the suggestion is there as a suggestion.
	firstReadBack *answer
}

func (rf runFixtures) answers(t *testing.T) []*answer {
	t.Helper()
	out := []*answer{
		{method: "GET", match: "PREVIEW_WITHOUT_SUGGESTIONS", json: rf.preview},
		{method: "GET", match: "/export?", bytes: rf.export},
		{method: "POST", match: proposeDocID + ":batchUpdate", json: readFixture(t, "propose-batch.json")},
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-before.json"), once: true},
	}
	for i := 0; i < rf.proposals; i++ {
		readBack := &answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-after.json"), once: true}
		if i == 0 && rf.firstReadBack != nil {
			readBack = rf.firstReadBack
		}
		out = append(out,
			&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-before.json"), once: true},
			readBack)
	}
	return out
}

// The three ways a read-back leaves a check false. Each is one route answering
// for itself: the inline view cannot see the suggestion, the preview reads as a
// document somebody edited, and a read that never arrived answers nothing.
func falseInlineCheck(t *testing.T) runFixtures {
	t.Helper()
	return runFixtures{
		proposals: 3,
		preview:   readFixture(t, "propose-preview.json"),
		export:    exportWithComment(t, true),
		// The read-back carries the words with no suggestion on them.
		firstReadBack: &answer{method: "GET", match: proposeDocID + "?includeTabsContent",
			json: readFixture(t, "propose-before.json"), once: true},
	}
}

func falsePreviewCheck(t *testing.T) runFixtures {
	t.Helper()
	return runFixtures{
		proposals: 3,
		preview:   readFixture(t, "propose-preview-edited.json"),
		export:    exportWithComment(t, true),
	}
}

func readBackThatFailed(t *testing.T) runFixtures {
	t.Helper()
	return runFixtures{
		proposals: 3,
		preview:   readFixture(t, "propose-preview.json"),
		export:    exportWithComment(t, true),
		firstReadBack: &answer{method: "GET", match: proposeDocID + "?includeTabsContent",
			err: errors.New("the connection was reset"), once: true},
	}
}

// stoppedAtTheFirst is the whole stop, asked of one run: the first proposal is
// sent and carries its checks, the two behind it never left, one batch reached
// the wire, and the envelope fails with the sentence that sends somebody to the
// browser.
func stoppedAtTheFirst(t *testing.T, f *fakeWire, got map[string]any, code int) {
	t.Helper()
	if code == 0 || got["ok"] != false {
		t.Fatalf("a proposal gdoc could not confirm must fail the run: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	list, _ := data["proposals"].([]any)
	if len(list) != 3 {
		t.Fatalf("proposals = %v, want one entry per proposal in the file", data["proposals"])
	}
	first, _ := list[0].(map[string]any)
	if first["sent"] != true {
		t.Errorf("the first proposal was written and reports sent = %v", first["sent"])
	}
	if _, has := first["checks"]; !has {
		t.Errorf("the first entry carries no checks, which is what says why the run stopped: %v", first)
	}
	for i, entry := range list[1:] {
		one, _ := entry.(map[string]any)
		if one["sent"] != false {
			t.Errorf("proposal %d was sent after the stop: %v", i+2, one)
		}
	}
	// One batch, so the stop is a stop rather than a report about a run that
	// finished anyway.
	var batches int
	for _, c := range f.writes() {
		if strings.Contains(c.URL, ":batchUpdate") {
			batches++
		}
	}
	if batches != 1 {
		t.Errorf("%d batches reached the wire, and only the first proposal may be sent: %v", batches, f.writes())
	}
	msg, _ := got["error"].(string)
	want := `gdoc could not confirm that proposal 1 ("reviewed annually") landed as a suggestion, ` +
		`so nothing after it was sent. Look at it in the browser before proposing again: ` +
		`https://docs.google.com/document/d/` + proposeDocID + `/edit`
	if msg != want {
		t.Errorf("error = %q, want %q", msg, want)
	}
}

// TestAFalseInlineCheckStopsTheRun is the SUGGEST that was not honoured, seen
// from the inline view: the words are in the document and no suggestion id is on
// them.
func TestAFalseInlineCheckStopsTheRun(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: falseInlineCheck(t).answers(t)})
	from := tempFile(t, "proposals.json", threeProposals)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from)

	stoppedAtTheFirst(t, f, got, code)
	first, _ := dataOf(t, got)["proposals"].([]any)[0].(map[string]any)
	checks, _ := first["checks"].(map[string]any)
	if checks["suggestions_inline"] != false {
		t.Errorf("suggestions_inline = %v, and the read-back carried no suggestion", checks)
	}
}

// TestAFalsePreviewCheckStopsTheRun is the same day seen from the other route:
// the preview no longer carries the quoted words, which is the shape of a
// document that was edited.
func TestAFalsePreviewCheckStopsTheRun(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: falsePreviewCheck(t).answers(t)})
	from := tempFile(t, "proposals.json", threeProposals)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from)

	stoppedAtTheFirst(t, f, got, code)
	first, _ := dataOf(t, got)["proposals"].([]any)[0].(map[string]any)
	checks, _ := first["checks"].(map[string]any)
	if checks["preview_without_suggestions"] != false {
		t.Errorf("preview_without_suggestions = %v, and the preview carried the replacement: %v", checks["preview_without_suggestions"], checks)
	}
	if checks["suggestions_inline"] != true {
		t.Errorf("suggestions_inline = %v, and the inline read-back held: %v", checks["suggestions_inline"], checks)
	}
}

// TestAReadBackThatFailedStopsTheRun is a read-back that could not be made,
// counted as false. gdoc does not know what is in the document, and not knowing
// never resolves to sending the rest.
func TestAReadBackThatFailedStopsTheRun(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: readBackThatFailed(t).answers(t)})
	from := tempFile(t, "proposals.json", threeProposals)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from)

	stoppedAtTheFirst(t, f, got, code)
	if !hasWarning(warningsOf(t, got), "could not be read back with suggestions inline") {
		t.Errorf("warnings = %v, and one must name the read that failed", warningsOf(t, got))
	}
}

// TestTheStopNeverClaimsADirectEdit is what the sentence may not say. gdoc knows
// a read-back did not confirm a suggestion; it does not know that somebody's
// document was edited, and a run that says so sends a colleague looking for
// damage that may not be there. The preview's own warning may name the shape it
// saw. The envelope's error may not.
func TestTheStopNeverClaimsADirectEdit(t *testing.T) {
	for name, fixtures := range map[string]runFixtures{
		"the inline check is false":  falseInlineCheck(t),
		"the preview check is false": falsePreviewCheck(t),
		"the read-back failed":       readBackThatFailed(t),
	} {
		t.Run(name, func(t *testing.T) {
			stubWire(t, &fakeWire{answers: fixtures.answers(t)})
			from := tempFile(t, "proposals.json", threeProposals)

			got, _ := runJSON(t, "propose", proposeDocID, "--from", from)

			msg, _ := got["error"].(string)
			if strings.Contains(strings.ToLower(msg), "direct edit") {
				t.Errorf("error = %q, and gdoc cannot know the document was edited", msg)
			}
		})
	}
}

// TestAFalseDocxCheckAloneDoesNotStop is the route that answers a different
// question. The export says whether the comment is attached to the words, and a
// comment that is not there is an explanation lost, not a change that went in as
// an edit. The suggestion is in the document as a suggestion, so the rest of the
// file is sent and the losses are warnings.
func TestAFalseDocxCheckAloneDoesNotStop(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: runFixtures{
		proposals: 3,
		preview:   readFixture(t, "propose-preview.json"),
		export:    exportWithComment(t, false),
	}.answers(t)})
	from := tempFile(t, "proposals.json", threeProposals)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from)
	if code != 0 || got["ok"] != true {
		t.Fatalf("a comment the export could not confirm is not a run that stops: %v (exit %d)", got, code)
	}
	list, _ := dataOf(t, got)["proposals"].([]any)
	if len(list) != 3 {
		t.Fatalf("proposals = %v, want three", list)
	}
	for i, entry := range list {
		one, _ := entry.(map[string]any)
		if one["sent"] != true {
			t.Errorf("proposal %d was not sent: %v", i+1, one)
		}
		checks, _ := one["checks"].(map[string]any)
		if checks["docx_anchored"] != false || checks["suggestions_inline"] != true || checks["preview_without_suggestions"] != true {
			t.Errorf("proposal %d checks = %v, want the export false and the two document routes true", i+1, checks)
		}
	}
	var batches int
	for _, c := range f.writes() {
		if strings.Contains(c.URL, ":batchUpdate") {
			batches++
		}
	}
	if batches != 3 {
		t.Errorf("%d batches reached the wire, and all three proposals were to be sent", batches)
	}
}

// TestAStoppedRunRecordsWhatWasSent is the note's half of the stop. The first
// proposal is in somebody's document, so its provenance is written down: a
// proposal gdoc has forgotten is one it will refuse to withdraw. The two that
// never left are not in the note, because there is nothing there to take back.
func TestAStoppedRunRecordsWhatWasSent(t *testing.T) {
	stubWire(t, &fakeWire{answers: falsePreviewCheck(t).answers(t)})
	from := tempFile(t, "proposals.json", threeProposals)
	note := copyFixture(t, "propose-note.md")

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--md", note)
	if code == 0 || got["ok"] != false {
		t.Fatalf("the run must stop: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	changed, _ := data["files_changed"].([]any)
	if len(changed) != 1 || changed[0] != note {
		t.Errorf("files_changed = %v, want the note the run was given", data["files_changed"])
	}
	after, err := os.ReadFile(note)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(after), "suggest.abc"); n != 1 {
		t.Errorf("the note records %d proposals and one was sent:\n%s", n, after)
	}
}
