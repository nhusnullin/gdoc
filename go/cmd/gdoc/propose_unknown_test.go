package main

import (
	"errors"
	"strings"
	"testing"
)

// This file is the bound on an answer that was lost. The batch went out and
// nothing came back, so the proposal may or may not be in the document. The run
// stops there, the entry says the outcome is unknown, and the sentence sends
// somebody to read the suggestions rather than claiming either way.

// lostAnswer is what the session hands back when a request was written and its
// answer never arrived. internal/gapi marks that case, and the writer packages
// ask by behaviour, so the fake answers by behaviour too.
type lostAnswer struct{ error }

func (lostAnswer) Unknown() bool { return true }

// lostAnswerRun is a run whose first batch is written and answers nothing. The
// preview and the export are never reached, and they are in the list because the
// command's own read and the first proposal's read come off the front of it.
func lostAnswerRun(t *testing.T) []*answer {
	t.Helper()
	return []*answer{
		{method: "GET", match: "PREVIEW_WITHOUT_SUGGESTIONS", json: readFixture(t, "propose-preview.json")},
		{method: "GET", match: "/export?", bytes: exportWithComment(t, true)},
		{method: "POST", match: proposeDocID + ":batchUpdate",
			err: lostAnswer{errors.New("read: connection reset by peer")}},
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-before.json"), once: true},
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-before.json"), once: true},
	}
}

// TestALostAnswerIsOutcomeUnknownAndStops is the third case the envelope has to
// carry. `sent` is false, because gdoc got no answer saying the batch landed, and
// `outcome` is what stops that from reading as a change that never left: it says
// nobody knows. Nothing behind it is sent, and nothing is retried.
func TestALostAnswerIsOutcomeUnknownAndStops(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: lostAnswerRun(t)})
	from := tempFile(t, "proposals.json", threeProposals)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from)

	if code == 0 || got["ok"] != false {
		t.Fatalf("a proposal whose answer was lost must fail the run: %v (exit %d)", got, code)
	}
	data := dataOf(t, got)
	list, _ := data["proposals"].([]any)
	if len(list) != 3 {
		t.Fatalf("proposals = %v, want one entry per proposal in the file", data["proposals"])
	}
	first, _ := list[0].(map[string]any)
	if first["sent"] != false {
		t.Errorf("sent = %v, and gdoc got no answer saying the batch landed: %v", first["sent"], first)
	}
	if first["outcome"] != "unknown" {
		t.Errorf("outcome = %v, want \"unknown\": %v", first["outcome"], first)
	}
	// The field appears on the one entry it is true of. On every other entry it
	// would say nobody knows about a proposal that never left the machine.
	for i, entry := range list[1:] {
		one, _ := entry.(map[string]any)
		if one["sent"] != false {
			t.Errorf("proposal %d was sent after the stop: %v", i+2, one)
		}
		if _, has := one["outcome"]; has {
			t.Errorf("proposal %d carries outcome %v, and it never left the machine: %v", i+2, one["outcome"], one)
		}
	}
	// One batch. The stop is a stop, not a report about a run that finished.
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
	want := `proposal 1 ("reviewed annually") was written and its answer was lost, so it may or may not be ` +
		`in the document. Read the suggestions before proposing it again`
	if msg != want {
		t.Errorf("error = %q, want %q", msg, want)
	}
}
