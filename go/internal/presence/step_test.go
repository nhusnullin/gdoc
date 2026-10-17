// What a step line is, and the two things it can never become.
//
// A step says a part of one command's work is beginning. The name is one of a
// fixed list and the two numbers are places in that list, so the record says
// where a run is and never how far along it is: a percentage would be a guess
// gdoc has nothing to make out of, and an estimate would be a promise the
// network does not keep.

package presence

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestAStepIsItsNameAndItsPlaceInTheFixedList holds the line's shape: the four
// names in the order a publish takes them, each with its place and the length
// of the list, and the command and the call of the run it belongs to.
func TestAStepIsItsNameAndItsPlaceInTheFixedList(t *testing.T) {
	dir, errOut := scene(t)
	ctx := WithRun(context.Background(), Begin(context.Background(), true, version, "publish", "", false, true, errOut))

	for _, name := range publishSteps {
		Step(ctx, name, "")
	}

	events := decoded(t, dir, "2026-10.jsonl")
	if len(events) != 1+len(publishSteps) {
		t.Fatalf("the run wrote %d lines, want the start and one per step: %+v", len(events), events)
	}
	for i, e := range events[1:] {
		if e.Event != "step" {
			t.Fatalf("line %d is a %q", i+1, e.Event)
		}
		if e.Step != publishSteps[i] {
			t.Errorf("step %d is named %q, want %q", i+1, e.Step, publishSteps[i])
		}
		if e.Index != i+1 {
			t.Errorf("step %q is index %d, want %d", e.Step, e.Index, i+1)
		}
		if e.Count != len(publishSteps) {
			t.Errorf("step %q counts %d, want %d", e.Step, e.Count, len(publishSteps))
		}
		if e.Command != "publish" || e.Via != "terminal" {
			t.Errorf("step %q names command %q via %q", e.Step, e.Command, e.Via)
		}
		if e.Call != events[0].Call {
			t.Errorf("step %q is call %d and its run began as call %d", e.Step, e.Call, events[0].Call)
		}
		if e.OK != nil || e.MS != nil {
			t.Errorf("a step says something about the outcome: %+v", e)
		}
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr: %q", errOut.String())
	}
}

// A word that is not one of the four writes no line. The four are the record's
// own list, so a caller that drifts from it writes nothing rather than a name
// a reader of a month's file has never seen.
func TestAStepNameThatIsNotOneOfTheFourWritesNothing(t *testing.T) {
	dir, errOut := scene(t)
	ctx := WithRun(context.Background(), Begin(context.Background(), true, version, "publish", "", false, true, errOut))

	for _, name := range []string{"", "Build", "uploading", "read-back", "pair", "build "} {
		Step(ctx, name, "Supplier register")
	}

	events := decoded(t, dir, "2026-10.jsonl")
	if len(events) != 1 {
		t.Fatalf("a name that is not one of the four wrote a line: %+v", events[1:])
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr: %q", errOut.String())
	}
}

// A step outside a run writes nothing, and that is two cases: a context nobody
// marked, and the flag being off, which is the nil run WithRun carries.
func TestAStepOutsideARunWritesNothing(t *testing.T) {
	dir, errOut := scene(t)

	Step(context.Background(), "build", "")
	Step(WithRun(context.Background(), Begin(context.Background(), false, version, "publish", "", false, true, errOut)), "build", "")
	var noContext context.Context
	Step(noContext, "build", "")

	if _, err := os.Stat(filepath.Join(dir, "presence")); !os.IsNotExist(err) {
		t.Fatalf("a step with no run wrote a record: %v", err)
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr: %q", errOut.String())
	}
}

// progressWords are what a progress bar is made of. None of them is a field of
// the event and none is a key in the fixture, because none of them is a fact:
// gdoc does not know what a part of a publish costs before it runs it, and a
// number that says 60% would be the tool guessing in a file that lasts.
//
// The two numbers a step does carry are places in a fixed list of four, which
// is why index and count are not on this list.
var progressWords = []string{
	"percent", "percentage", "pct", "eta", "estimate", "estimated",
	"remaining", "progress", "ratio", "fraction", "elapsed", "left",
}

// TestNoPercentageOrEstimateIsEverWritten holds the record to places in a list.
// It reads the event's own fields and every key of the fixture, so a field
// somebody adds and a line the writer writes are both measured.
func TestNoPercentageOrEstimateIsEverWritten(t *testing.T) {
	ty := reflect.TypeOf(Event{})
	for i := 0; i < ty.NumField(); i++ {
		tag := strings.Split(ty.Field(i).Tag.Get("json"), ",")[0]
		for _, word := range progressWords {
			if strings.Contains(strings.ToLower(tag), word) {
				t.Errorf("the event carries %q, and the record says where a run is rather than how far along it is", tag)
			}
		}
	}

	b, err := os.ReadFile(filepath.Join("testdata", "events-v1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for n, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		var top map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &top); err != nil {
			t.Fatalf("line %d: %v", n+1, err)
		}
		for key := range top {
			for _, word := range progressWords {
				if strings.Contains(strings.ToLower(key), word) {
					t.Errorf("line %d carries key %q", n+1, key)
				}
			}
		}
	}
}
