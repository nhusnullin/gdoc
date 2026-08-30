package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubLogin stands in for the browser trip. The flow itself is covered end to
// end in internal/auth, guard-built client included; what these tests hold is
// the CLI's half of the contract.
func stubLogin(t *testing.T, f func(io.Writer) error) {
	t.Helper()
	old := login
	login = f
	t.Cleanup(func() { login = old })
}

// The URL is a human instruction, so it goes to stderr. stdout carries the one
// JSON object and nothing else, which is what lets a skill parse it.
func TestLoginPrintsTheURLToStderrNotStdout(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"),
		[]byte(`{"token":"A","refresh_token":"R","client_id":"CID","client_secret":"CS",`+
			`"scopes":["x"],"expiry":"2020-01-01T00:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stubLogin(t, func(w io.Writer) error {
		fmt.Fprintln(w, "Open this link in your browser to sign in:")
		fmt.Fprintln(w, "https://accounts.google.com/o/oauth2/auth?x=1")
		return nil
	})

	var out, errOut bytes.Buffer
	code := run([]string{"auth", "login"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("a login that worked must exit 0: %s", out.String())
	}
	if strings.Contains(out.String(), "accounts.google.com") {
		t.Fatalf("the URL leaked into stdout: %q", out.String())
	}
	if !strings.Contains(errOut.String(), "accounts.google.com") {
		t.Fatalf("the URL must reach stderr: %q", errOut.String())
	}
	if got := decodeOne(t, &out); got["ok"] != true {
		t.Fatalf("envelope: %v", got)
	}
}

func TestAFailedLoginIsOneFailingEnvelope(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	stubLogin(t, func(io.Writer) error { return errors.New("no login callback arrived within the timeout") })

	got, code := runJSON(t, "auth", "login")
	if code == 0 || got["ok"] != false {
		t.Fatalf("a failed login must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "timeout") {
		t.Fatalf("the error must say what went wrong: %q", msg)
	}
}

// A panic is still one JSON object on stdout and a non-zero exit. A Go stack
// trace on stdout and exit 2 breaks the contract every caller relies on.
func TestAPanicIsStillOneEnvelope(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	stubLogin(t, func(io.Writer) error { panic("something impossible") })

	var out, errOut bytes.Buffer
	code := run([]string{"auth", "login"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("a crash must exit 1, got %d", code)
	}
	got := decodeOne(t, &out)
	if got["ok"] != false {
		t.Fatalf("a crash is not a success: %v", got)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "something impossible") {
		t.Fatalf("the envelope must carry what happened: %q", msg)
	}
	if !strings.Contains(errOut.String(), "goroutine") {
		t.Fatalf("the stack trace belongs on stderr: %q", errOut.String())
	}
}
