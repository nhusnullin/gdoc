package emit

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrintSuccessShape(t *testing.T) {
	var buf bytes.Buffer
	r := Result{OK: true, Data: map[string]any{"n": 1}, Warnings: []string{"w"}}
	if err := Print(&buf, r); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not one JSON object: %v", err)
	}
	if got["ok"] != true || got["error"] != nil {
		t.Fatalf("wrong envelope: %v", got)
	}
	if ExitCode(r) != 0 {
		t.Fatal("success must exit 0")
	}
}

func TestPrintFailureShape(t *testing.T) {
	var buf bytes.Buffer
	r := Result{OK: false, Error: "no token at /x. Run: gdoc auth login"}
	if err := Print(&buf, r); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not one JSON object: %v", err)
	}
	if got["ok"] != false {
		t.Fatalf("a failure must carry ok=false: %v", got)
	}
	msg, ok := got["error"].(string)
	if !ok || msg == "" {
		t.Fatalf("a failure must carry a message: %v", got)
	}
	if ExitCode(r) != 1 {
		t.Fatal("failure must exit non-zero")
	}
}

// One object, one newline, nothing after it. A second line would make every
// caller's parse ambiguous.
func TestPrintWritesOneLine(t *testing.T) {
	var buf bytes.Buffer
	if err := Print(&buf, Result{OK: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("the object must end in a newline: %q", out)
	}
	if strings.Count(out, "\n") != 1 {
		t.Fatalf("exactly one line, got %q", out)
	}
}

// HTML escaping is off on purpose: a document title or an error message full of
// < is unreadable to the human the skill shows it to.
func TestPrintDoesNotEscapeHTML(t *testing.T) {
	var buf bytes.Buffer
	if err := Print(&buf, Result{OK: false, Error: `a <b> & c`}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `a <b> & c`) {
		t.Fatalf("angle brackets and ampersands go out as themselves: %q", buf.String())
	}
}

// The envelope's optional keys are absent rather than null when there is
// nothing to say. A caller reading data must be able to tell "no data" from
// "data: null".
func TestEmptyFieldsAreAbsent(t *testing.T) {
	var buf bytes.Buffer
	if err := Print(&buf, Result{OK: true}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"data", "error", "warnings"} {
		if _, present := got[key]; present {
			t.Errorf("%q must be absent when there is nothing to report: %v", key, got)
		}
	}
}

// omitempty on an `any` field drops only a nil one, so an empty map and a
// false still print. That is the behaviour a command relies on when its data
// is legitimately empty, and it is worth pinning rather than assuming.
func TestAFalsyDataIsStillPrinted(t *testing.T) {
	cases := []struct {
		name string
		data any
	}{
		{"empty map", map[string]any{}},
		{"false", false},
		{"zero", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := Print(&buf, Result{OK: true, Data: tc.data}); err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if _, present := got["data"]; !present {
				t.Fatalf("data was set and must be reported: %v", got)
			}
		})
	}
}
