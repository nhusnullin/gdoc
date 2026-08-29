package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// runJSON runs the command and insists stdout is exactly one JSON object.
func runJSON(t *testing.T, args ...string) (map[string]any, int) {
	t.Helper()
	var buf bytes.Buffer
	code := run(args, &buf)
	dec := json.NewDecoder(&buf)
	var got map[string]any
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("stdout is not one JSON object: %v", err)
	}
	if dec.More() {
		t.Fatal("stdout carried more than one JSON object")
	}
	return got, code
}

func TestAuthStatusReportsTheToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("status with no token is still a successful report: %v (exit %d)", got, code)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	if data["token_present"] != false {
		t.Fatalf("token_present: %v", data["token_present"])
	}
	if !strings.HasPrefix(data["token_path"].(string), dir) {
		t.Fatalf("token_path must sit in the config dir: %v", data["token_path"])
	}
}

func TestUnknownCommandFailsAndNamesItself(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, code := runJSON(t, "sing", "loudly")
	if code == 0 || got["ok"] != false {
		t.Fatalf("an unknown command must fail: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "sing loudly") || !strings.Contains(msg, "auth status") {
		t.Fatalf("the error must name the command and the ones that exist: %q", msg)
	}
}

func TestNoArgumentsFails(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, code := runJSON(t)
	if code == 0 || got["ok"] != false {
		t.Fatalf("bare gdoc must fail: %v (exit %d)", got, code)
	}
}
