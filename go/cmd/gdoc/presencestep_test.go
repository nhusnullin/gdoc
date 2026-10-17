// The four parts of a publish, as the record says them.
//
// A publish is the one command long enough that a person watching it needs to
// know which part of it is running: the render, the upload, the three
// read-backs and the note. Each is one line, written immediately before the
// part it names begins, so a panel reading the record says what is happening
// now rather than what has just finished. The tests here are over a real
// publish run, because the order is the order of the work and not a list this
// file could state.

package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"gdoc/internal/presence"
)

// stepsOf is the step lines a run wrote, in order.
func stepsOf(t *testing.T, events []presence.Event) []presence.Event {
	t.Helper()
	var out []presence.Event
	for _, e := range events {
		if e.Event == "step" {
			out = append(out, e)
		}
	}
	return out
}

// stepNames is what a set of step lines is called, in order.
func stepNames(events []presence.Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Step)
	}
	return out
}

// publishSteps is the list this package expects of a publish, as literals. A
// test that read the names out of internal/presence would follow them
// wherever somebody moved them.
var publishStepNames = []string{"build", "upload", "read_back", "link"}

// A publish writes its four steps in the order it takes them, each before the
// part it names begins. The two network parts are measured where they begin:
// the upload step is already on disk when the upload leaves, and the read_back
// step when the first read-back leaves.
func TestAPublishStepsAreTheFixedNamesInOrder(t *testing.T) {
	theHouseLogoCached(t)
	dir := os.Getenv("GDOC_CONFIG_DIR")
	_, md := buildNote(t)
	answers := publishAnswers(t)
	var atUpload, atReadBack []string
	answers[0].before = func() { atUpload = stepNames(stepsOf(t, presenceEvents(t, dir))) }
	answers[1].before = func() { atReadBack = stepNames(stepsOf(t, presenceEvents(t, dir))) }
	stubWire(t, &fakeWire{answers: answers})

	var out bytes.Buffer
	if code := run(context.Background(), []string{"publish", "--md", md, "--folder-id", testFolderID},
		&out, io.Discard, presenceOn()); code != 0 {
		t.Fatalf("publish failed: exit %d, %s", code, out.String())
	}

	events := presenceEvents(t, dir)
	steps := stepsOf(t, events)
	if got := stepNames(steps); !sameWords(got, publishStepNames) {
		t.Fatalf("the publish wrote steps %v, want %v", got, publishStepNames)
	}
	for i, e := range steps {
		if e.Index != i+1 || e.Count != len(publishStepNames) {
			t.Errorf("step %q is %d of %d, want %d of %d", e.Step, e.Index, e.Count, i+1, len(publishStepNames))
		}
		if e.Command != "publish" {
			t.Errorf("step %q names command %q", e.Step, e.Command)
		}
		if e.OK != nil || e.MS != nil || e.Made != "" {
			t.Errorf("a step says something only an end may say: %+v", e)
		}
	}

	// The whole run, in order: the start, the four steps, the end.
	if events[0].Event != "start" || events[len(events)-1].Event != "end" {
		t.Fatalf("the run is %v", kindsOf(events))
	}
	if len(events) != len(publishStepNames)+2 {
		t.Fatalf("the run wrote %d lines: %v", len(events), kindsOf(events))
	}

	// And each step was already written when its own part began.
	if !sameWords(atUpload, []string{"build", "upload"}) {
		t.Errorf("the record held %v when the upload left, want the build and the upload", atUpload)
	}
	if !sameWords(atReadBack, []string{"build", "upload", "read_back"}) {
		t.Errorf("the record held %v when the read-back left", atReadBack)
	}
}

// The render is what learns the title, so the step after it is the first line
// of the run that can carry one. The build step carries none: at that moment
// the title is in a note nothing has read yet.
func TestTheFirstStepsEndCarriesTheTitle(t *testing.T) {
	theHouseLogoCached(t)
	dir := os.Getenv("GDOC_CONFIG_DIR")
	_, md := buildNote(t)
	stubWire(t, &fakeWire{answers: publishAnswers(t)})

	var out bytes.Buffer
	if code := run(context.Background(), []string{"publish", "--md", md, "--folder-id", testFolderID},
		&out, io.Discard, presenceOn()); code != 0 {
		t.Fatalf("publish failed: exit %d, %s", code, out.String())
	}

	steps := stepsOf(t, presenceEvents(t, dir))
	if len(steps) != len(publishStepNames) {
		t.Fatalf("the publish wrote %d steps: %v", len(steps), stepNames(steps))
	}
	if steps[0].Title != "" {
		t.Errorf("the build step names title %q, and the render had not read the note yet", steps[0].Title)
	}
	for _, e := range steps[1:] {
		if e.Title != "Supplier Register Policy" {
			t.Errorf("step %q names title %q, want the one the render put on the cover", e.Step, e.Title)
		}
	}
}

// A run that crashed after a step still ends. The steps stay where they were
// and the end says ok: false, so a panel reading the record never leaves a
// publish standing on the part it died in.
func TestAPanicAfterAStepStillWritesTheEnd(t *testing.T) {
	theHouseLogoCached(t)
	dir := os.Getenv("GDOC_CONFIG_DIR")
	_, md := buildNote(t)
	answers := publishAnswers(t)
	answers[0].before = func() { panic("something impossible") }
	stubWire(t, &fakeWire{answers: answers})

	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"publish", "--md", md, "--folder-id", testFolderID},
		&out, &errOut, presenceOn()); code != 1 {
		t.Fatalf("a crash must exit 1, got %d: %s", code, out.String())
	}

	events := presenceEvents(t, dir)
	if got := stepNames(stepsOf(t, events)); !sameWords(got, []string{"build", "upload"}) {
		t.Fatalf("the crashed run wrote steps %v, want the two it had reached", got)
	}
	end := endOf(t, events)
	if end.OK == nil || *end.OK {
		t.Errorf("the end of a crashed publish says ok %v", end.OK)
	}
	if end.Command != "publish" {
		t.Errorf("the end names command %q", end.Command)
	}
}

// kindsOf is what a run wrote, as the words a failure message reads best in.
func kindsOf(events []presence.Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Event)
	}
	return out
}

// equalWords is two lists of words being the same list.
func sameWords(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
