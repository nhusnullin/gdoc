package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runJSON runs the command and insists stdout is exactly one JSON object.
func runJSON(t *testing.T, args ...string) (map[string]any, int) {
	t.Helper()
	var buf bytes.Buffer
	code := run(args, &buf, io.Discard)
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

// The populated shape, through the envelope: this is what a skill reads.
func TestAuthStatusReportsAPresentToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	token := `{"token":"A","refresh_token":"R","token_uri":"https://oauth2.googleapis.com/token",` +
		`"client_id":"CID","client_secret":"CS","scopes":["https://www.googleapis.com/auth/drive"],` +
		`"expiry":"2020-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("status: %v (exit %d)", got, code)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	if data["token_present"] != true || data["expired"] != true {
		t.Fatalf("a 2020 token is present and expired: %v", data)
	}
	if data["client_source"] != "bundled" || data["auth_mode"] != "oauth" {
		t.Fatalf("v2 is OAuth only on the bundled client: %v", data)
	}
	scopes, ok := data["scopes"].([]any)
	if !ok || len(scopes) != 1 {
		t.Fatalf("scopes: %v", data["scopes"])
	}
}

// A token that exists and cannot be read must fail loudly rather than read as
// signed out, and the envelope still names the file it tried.
func TestAuthStatusFailsOnAnUnreadableToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "auth", "status")
	if code == 0 || got["ok"] != false {
		t.Fatalf("an unreadable token must fail: %v (exit %d)", got, code)
	}
	if data, ok := got["data"].(map[string]any); !ok || data["token_path"] == nil {
		t.Fatalf("the facts still come back beside the error: %v", got)
	}
}

// A command that accepts and ignores what it does not understand tells the user
// it did something it did not.
func TestTrailingArgumentsAreRefused(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	for _, args := range [][]string{
		{"auth", "status", "nonsense"},
		{"auth", "login", "--token", "/tmp/x"},
	} {
		got, code := runJSON(t, args...)
		if code == 0 || got["ok"] != false {
			t.Errorf("%v must be refused, not silently trimmed: %v", args, got)
		}
	}
}

func TestAuthStatusWithNoTokenIsStillAReport(t *testing.T) {
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
	if !strings.Contains(msg, "auth login") {
		t.Fatalf("the usage line must name login too: %q", msg)
	}
}

// A token granted less than gdoc asked for still reports ok, and says which
// scope is missing. A v1 token looks exactly like this: it carries drive plus
// documents.readonly, so the Docs writes v2 makes will be refused.
func TestAuthStatusWarnsAboutAScopeTheTokenDoesNotCarry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	token := `{"token":"A","refresh_token":"R","token_uri":"https://oauth2.googleapis.com/token",` +
		`"client_id":"CID","client_secret":"CS","scopes":["https://www.googleapis.com/auth/drive"],` +
		`"expiry":"2020-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("a partial grant is a report, not a failure: %v (exit %d)", got, code)
	}
	warnings, _ := got["warnings"].([]any)
	if len(warnings) != 1 {
		t.Fatalf("want one warning about the missing scope: %v", got["warnings"])
	}
	if w, _ := warnings[0].(string); !strings.Contains(w, "auth/documents") {
		t.Fatalf("the warning must name the scope: %q", w)
	}
}

// A failing status keeps its warnings. The run where the token cannot be read
// is the one where a person most needs to know an oauth-client.json is sitting
// there unused, and it was the run that dropped the sentence saying so.
func TestAFailingStatusStillCarriesItsWarnings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "oauth-client.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "auth", "status")
	if code == 0 || got["ok"] != false {
		t.Fatalf("an unreadable token must fail: %v (exit %d)", got, code)
	}
	warnings, _ := got["warnings"].([]any)
	if len(warnings) != 1 {
		t.Fatalf("the ignored client file must still be reported: %v", got["warnings"])
	}
	if w, _ := warnings[0].(string); !strings.Contains(w, "oauth-client.json") {
		t.Fatalf("the warning must name the file: %q", w)
	}
}

// Bare gdoc named no command, so the error must not quote one. `unknown command
// ""` names nothing and reads like a bug in the tool.
func TestNoArgumentsFails(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, code := runJSON(t)
	if code == 0 || got["ok"] != false {
		t.Fatalf("bare gdoc must fail: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if strings.Contains(msg, `""`) {
		t.Errorf("nothing was named, so nothing must be quoted back: %q", msg)
	}
	if !strings.Contains(msg, "needs a command") || !strings.Contains(msg, "auth status") {
		t.Errorf("the error must say what is missing and what exists: %q", msg)
	}
}
