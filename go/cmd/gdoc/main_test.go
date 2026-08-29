package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// decodeOne insists a stream is exactly one JSON object. It is the output
// contract itself, so it is written once: the tests that also need stderr
// decode their own stdout with it rather than copying the assertion.
//
// The end of the stream is proved by decoding a second value and requiring
// io.EOF, not by Decoder.More. More answers whether another element follows in
// the array or object being read, so it reads a closing bracket as the end and
// `{"ok":true}]` passed the helper while being output no skill can parse.
func decodeOne(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	got, err := onlyJSONObject(r)
	if err != nil {
		t.Fatalf("stdout is not exactly one JSON object: %v", err)
	}
	return got
}

// onlyJSONObject is decodeOne's judgment, split out so a test can check the
// helper itself. A helper that calls t.Fatal cannot be shown to refuse
// anything, and this one is the output contract.
func onlyJSONObject(r io.Reader) (map[string]any, error) {
	dec := json.NewDecoder(r)
	var got map[string]any
	if err := dec.Decode(&got); err != nil {
		return nil, fmt.Errorf("the first value is not a JSON object: %w", err)
	}
	var rest json.RawMessage
	if err := dec.Decode(&rest); err != io.EOF {
		return nil, fmt.Errorf("something follows the object: %q, err %v", string(rest), err)
	}
	return got, nil
}

// The helper is the contract, so it is checked in both directions. Every shape
// below is stdout no skill can parse, and Decoder.More let the trailing-bracket
// one through: More reports whether another element follows inside the array or
// object being read, and a closing bracket at the top level reads to it as the
// end of the stream.
func TestOnlyJSONObjectRefusesAnythingAfterTheObject(t *testing.T) {
	for _, out := range []string{
		`{"ok":true}]`,
		`{"ok":true}}`,
		`{"ok":true}{"ok":false}`,
		`{"ok":true} trailing`,
		`{"ok":true}[1]`,
		`[{"ok":true}]`,
		``,
	} {
		if _, err := onlyJSONObject(strings.NewReader(out)); err == nil {
			t.Errorf("%q must be refused: stdout is exactly one JSON object", out)
		}
	}
	got, err := onlyJSONObject(strings.NewReader("{\"ok\":true}\n"))
	if err != nil {
		t.Fatalf("one object, with the newline the emitter writes: %v", err)
	}
	if got["ok"] != true {
		t.Fatalf("the object came back wrong: %v", got)
	}
}

// runJSON runs the command and insists stdout is exactly one JSON object.
func runJSON(t *testing.T, args ...string) (map[string]any, int) {
	t.Helper()
	var buf bytes.Buffer
	code := run(args, &buf, io.Discard)
	return decodeOne(t, &buf), code
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
// scope is missing. This is the granular consent screen: somebody ticked Docs
// and left Drive unticked. A v1 token is NOT this case, because the full Drive
// scope it carries covers the Docs calls too.
func TestAuthStatusWarnsAboutAScopeTheTokenDoesNotCarry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	token := `{"token":"A","refresh_token":"R","token_uri":"https://oauth2.googleapis.com/token",` +
		`"client_id":"CID","client_secret":"CS","scopes":["https://www.googleapis.com/auth/documents"],` +
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
	if w, _ := warnings[0].(string); !strings.Contains(w, "auth/drive") {
		t.Fatalf("the warning must name the scope: %q", w)
	}
}

// The repository owner's own token is a v1 token: drive plus
// documents.readonly. The Docs API accepts the full Drive scope, so nothing is
// missing and status must say nothing.
func TestAuthStatusIsQuietForAV1Token(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	token := `{"token":"A","refresh_token":"R","token_uri":"https://oauth2.googleapis.com/token",` +
		`"client_id":"CID","client_secret":"CS","scopes":["https://www.googleapis.com/auth/drive",` +
		`"https://www.googleapis.com/auth/documents.readonly"],"expiry":"2020-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("a v1 token is a working token: %v (exit %d)", got, code)
	}
	if w, _ := got["warnings"].([]any); len(w) != 0 {
		t.Fatalf("a v1 token is missing nothing v2 needs: %v", got["warnings"])
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
