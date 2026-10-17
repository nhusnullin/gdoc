package emit

import (
	"bytes"
	"strings"
	"testing"
)

// TestStopNeverReachesTheObject is TestScreenNeverReachesTheObject for the
// other two fields a caller sets and the object never carries.
func TestStopNeverReachesTheObject(t *testing.T) {
	var stopped, plain bytes.Buffer
	failed := Result{OK: false, Error: "the picture could not be embedded",
		Stop: StopSVGPicture, StopLine: 42, Version: "v2.15.0"}
	if err := Print(&stopped, failed); err != nil {
		t.Fatal(err)
	}
	if err := Print(&plain, Result{OK: false, Error: "the picture could not be embedded", Version: "v2.15.0"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(stopped.String()); got != `{"ok":false,"error":"the picture could not be embedded","version":"v2.15.0"}` {
		t.Errorf("a stop prints the object it would have printed anyway: %s", got)
	}
	if stopped.String() != plain.String() {
		t.Errorf("Stop and StopLine change no byte of the object:\n%s\n%s", stopped.String(), plain.String())
	}
	if ExitCode(failed) != 1 || ExitCode(Result{OK: true, Stop: StopOther}) != 0 {
		t.Error("a stop changes no exit code: it stays 0 if and only if the result says ok")
	}
}

// TestTheSevenCodesAreLiterals states every code as a literal, because a test
// that reads the constant follows it wherever somebody moves it. Seven is the
// whole set: a code that is not here is not a stop code.
func TestTheSevenCodesAreLiterals(t *testing.T) {
	want := []StopCode{"svg_picture", "multi_tab", "not_found", "auth", "network", "refused", "other"}
	got := StopCodes()
	if len(got) != 7 {
		t.Fatalf("the set is seven codes, and this one is %d: %v", len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("code %d is %q, and the spec names %q", i, got[i], w)
		}
		if !w.Known() {
			t.Errorf("%q is one of the seven and is not known", w)
		}
	}
	for _, c := range []StopCode{"", "SVG_PICTURE", "svg", "the picture could not be embedded"} {
		if c.Known() {
			t.Errorf("%q is not one of the seven and says it is", c)
		}
	}
	if StopSVGPicture != "svg_picture" || StopMultiTab != "multi_tab" ||
		StopNotFound != "not_found" || StopAuth != "auth" ||
		StopNetwork != "network" || StopRefused != "refused" || StopOther != "other" {
		t.Error("a constant's value is the literal the spec names")
	}
}

// TestStopCodesIsACopy holds that a caller cannot edit the set from under the
// next caller, as Warnings does in the guard.
func TestStopCodesIsACopy(t *testing.T) {
	first := StopCodes()
	first[0] = "nonsense"
	if StopCodes()[0] != "svg_picture" {
		t.Error("StopCodes hands out a copy, so one caller's write is not the next caller's set")
	}
}
