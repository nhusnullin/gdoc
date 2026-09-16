package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
	return runJSONCtx(t, context.Background(), args...)
}

// runJSONCtx is runJSON with the context main builds from the signal. Only a
// wait reads it, and the tests that hand in a cancelled one are testing the
// answer a stopped session gets.
func runJSONCtx(t *testing.T, ctx context.Context, args ...string) (map[string]any, int) {
	t.Helper()
	var buf bytes.Buffer
	code := run(ctx, args, &buf, io.Discard)
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

// The usage line is joined from the command table, so it has to name every
// command that exists. The list here is written out word for word, the way a
// reader sees it, so it cannot follow a rename of anything in the code. It is
// read off the refusal an unknown command gets, which is where a person who
// typed the wrong word meets it.
func TestTheUsageLineNamesEveryCommand(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, code := runJSON(t, "sing")
	if code == 0 || got["ok"] != false {
		t.Fatalf("an unknown command must fail: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	for _, command := range []string{"auth status", "auth login", "read", "comments",
		"suggestions", "restyle", "probe", "reply", "propose", "withdraw", "build",
		"publish"} {
		if !strings.Contains(msg, command) {
			t.Errorf("the usage line must name %q: %q", command, msg)
		}
	}
}

// A token granted less than gdoc asked for still reports ok, and says which
// scope is missing. This is the granular consent screen: somebody ticked Docs
// and left Drive unticked. A token carrying the full Drive scope is NOT this
// case, because that scope covers the Docs calls too.
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

// The repository owner's own token carries drive plus documents.readonly. The
// Docs API accepts the full Drive scope, so nothing is missing and status must
// say nothing.
func TestAuthStatusIsQuietForATokenCarryingTheFullDriveScope(t *testing.T) {
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
		t.Fatalf("this is a working token: %v (exit %d)", got, code)
	}
	if w, _ := got["warnings"].([]any); len(w) != 0 {
		t.Fatalf("this token is missing nothing gdoc needs: %v", got["warnings"])
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

// The signal trap belongs to the wait, and to nothing else.
//
// signal.Notify takes the default kill away from the whole process for as long
// as it is installed, and NotifyContext never puts it back on its own. Trapped
// in main, Ctrl-C would stop being an answer for every command that does not
// read the context: `auth login` would hold the terminal for its whole login
// timeout, and a `propose` in the middle of writing into somebody's document
// could not be stopped at all. So exactly one file names os/signal, and it is
// the one holding the wait.
//
// The check fails in both directions, like the boundary tests: it fails when
// the import spreads, and it fails when read.go stops installing one.
func TestOnlyTheWaitTrapsTheSignal(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var traps []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), `"os/signal"`) {
			traps = append(traps, name)
		}
	}
	sort.Strings(traps)
	if !reflect.DeepEqual(traps, []string{"read.go"}) {
		t.Errorf("os/signal is named in %v, and only read.go's wait may trap a signal", traps)
	}
}
