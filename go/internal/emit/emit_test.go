package emit

import (
	"bytes"
	"encoding/json"
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
	_ = json.Unmarshal(buf.Bytes(), &got)
	if got["ok"] != false || got["error"] == "" {
		t.Fatalf("failure must carry ok=false and a message: %v", got)
	}
	if ExitCode(r) != 1 {
		t.Fatal("failure must exit non-zero")
	}
}
