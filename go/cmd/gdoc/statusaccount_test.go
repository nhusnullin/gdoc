// `gdoc auth status` and the account it names: when the read happens, when it
// does not, what a failed read leaves behind, and the panel a person reads.
//
// Every run here stands in for the read through accountOf, which TestMain
// already replaced, so nothing reaches the network. The ceiling is a variable
// so the run that hangs is one that finishes.

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"gdoc/internal/gapi"
)

// The scopes a token carries in these runs. documents on its own is a token
// missing drive, because drive covers documents and nothing covers drive.
const (
	driveScope     = "https://www.googleapis.com/auth/drive"
	documentsScope = "https://www.googleapis.com/auth/documents"
)

// tokenWithScopes is a signed-in token file carrying exactly those scopes.
func tokenWithScopes(t *testing.T, dir string, scopes ...string) {
	t.Helper()
	quoted := make([]string, 0, len(scopes))
	for _, s := range scopes {
		quoted = append(quoted, strconv.Quote(s))
	}
	writeTokenFile(t, dir, `{"token":"ya29.not-a-real-token",`+
		`"refresh_token":"1//not-a-real-refresh",`+
		`"client_id":"not-a-real-client.apps.googleusercontent.com",`+
		`"client_secret":"not-a-real-secret",`+
		`"token_uri":"https://oauth2.googleapis.com/token",`+
		`"scopes":[`+strings.Join(quoted, ",")+`],`+
		`"expiry":"2099-01-01T00:00:00Z"}`)
}

// statusData is the data object of one `gdoc auth status` run.
func statusObject(t *testing.T, got map[string]any) map[string]any {
	t.Helper()
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	return data
}

// The ceiling is five seconds, stated here as a literal: an account read is
// one small GET, and a person who typed `gdoc auth status` is waiting at a
// prompt. Long enough for a slow network, short enough that a hung read is
// still an answer.
func TestTheAccountCeilingIsFiveSeconds(t *testing.T) {
	if accountCeiling != 5*time.Second {
		t.Errorf("the account read's ceiling is five seconds, and it is %v", accountCeiling)
	}
}

// The account is read live and goes into the data under its own two fields.
func TestAuthStatusNamesTheAccountItReadsLive(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	tokenWithScopes(t, dir, driveScope)
	reads := stubAccount(t, gapi.Account{Email: "someone@example.com", Name: "Someone"}, nil)

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("status is a report and must answer: %v (exit %d)", got, code)
	}
	if *reads != 1 {
		t.Fatalf("one status run is one account read, and this run made %d", *reads)
	}
	data := statusObject(t, got)
	if data["account"] != "someone@example.com" || data["account_name"] != "Someone" {
		t.Errorf("the account read must reach the data: %v", data)
	}
	if got["warnings"] != nil {
		t.Errorf("a read that worked warns about nothing: %v", got["warnings"])
	}
}

// No token is no account: there is nothing to read it with, and a request that
// would be refused is a request nobody should make.
func TestNoTokenMakesNoAccountRequest(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	reads := stubAccount(t, gapi.Account{Email: "someone@example.com"}, nil)

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("signed out is still an answer: %v (exit %d)", got, code)
	}
	if *reads != 0 {
		t.Errorf("a status run with no token reads no account, and this one read %d", *reads)
	}
	if data := statusObject(t, got); data["account"] != nil {
		t.Errorf("there is no account to name: %v", data)
	}
}

// A token missing a scope gdoc asks for is the same case: the warning already
// says to sign in again, and the read would be refused by the scope that is
// missing rather than answered.
func TestAMissingScopeMakesNoAccountRequest(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	tokenWithScopes(t, dir, documentsScope)
	reads := stubAccount(t, gapi.Account{Email: "someone@example.com"}, nil)

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("a partial grant is reported, not refused: %v (exit %d)", got, code)
	}
	data := statusObject(t, got)
	if data["missing_scopes"] == nil {
		t.Fatalf("this token is missing drive: %v", data)
	}
	if *reads != 0 {
		t.Errorf("a token missing a scope reads no account, and this one read %d", *reads)
	}
	if data["account"] != nil {
		t.Errorf("no account was read, so none is named: %v", data)
	}
}

// The read failing does not make a signed-in person signed out. The token is
// the fact; who it belongs to is what could not be read, and that is a warning.
func TestAuthStatusOfflineStillAnswersWithoutTheAccount(t *testing.T) {
	const reason = "the account read failed"
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	tokenWithScopes(t, dir, driveScope)
	stubAccount(t, gapi.Account{}, errors.New(reason))

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("a token that is there is still a token: %v (exit %d)", got, code)
	}
	if data := statusObject(t, got); data["account"] != nil || data["account_name"] != nil {
		t.Errorf("nothing was read, so nothing is named: %v", data)
	}
	warnings, ok := got["warnings"].([]any)
	if !ok || len(warnings) != 1 {
		t.Fatalf("one warning names the reason: %v", got["warnings"])
	}
	if w, _ := warnings[0].(string); !strings.Contains(w, reason) {
		t.Errorf("the warning must carry what went wrong, and it says %q", w)
	}
}

// A read that never comes back stops at the ceiling, and the report is the
// answer: a person asking whether they are signed in is not made to wait on
// somebody else's network.
func TestAnAccountReadThatHangsStopsAtTheCeiling(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	tokenWithScopes(t, dir, driveScope)
	old := accountCeiling
	accountCeiling = 10 * time.Millisecond
	t.Cleanup(func() { accountCeiling = old })
	oldRead := accountOf
	accountOf = func(ctx context.Context) (gapi.Account, error) {
		<-ctx.Done()
		return gapi.Account{}, ctx.Err()
	}
	t.Cleanup(func() { accountOf = oldRead })

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the report is the answer whatever the read did: %v (exit %d)", got, code)
	}
	if data := statusObject(t, got); data["account"] != nil {
		t.Errorf("a read that never answered names no account: %v", data)
	}
	if got["warnings"] == nil {
		t.Errorf("the reader is told the account could not be read: %v", got)
	}
}

// `auth login` reports what status reports, and it did the sign-in: the
// account read is a second request on top of a trip that just finished, and
// the object a skill reads after a login is the one it read before.
func TestAuthLoginCarriesNoAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	tokenWithScopes(t, dir, driveScope)
	reads := stubAccount(t, gapi.Account{Email: "someone@example.com"}, nil)
	stubLogin(t, func(io.Writer) error { return nil })

	got, code := runJSON(t, "auth", "login")
	if code != 0 || got["ok"] != true {
		t.Fatalf("a login that worked answers: %v (exit %d)", got, code)
	}
	if *reads != 0 {
		t.Errorf("auth login reads no account, and this run read %d", *reads)
	}
	if data := statusObject(t, got); data["account"] != nil {
		t.Errorf("auth login's object is unchanged: %v", data)
	}
}

// statusScreenRun is `gdoc auth status` with both streams a terminal, at
// eighty columns with the colour off.
func statusScreenRun(t *testing.T) (stdout, stderr string, code int) {
	t.Helper()
	t.Setenv("COLUMNS", "80")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "xterm")

	var out, errOut bytes.Buffer
	terminals(t, &out, &errOut)
	code = run(context.Background(), []string{"auth", "status"}, &out, &errOut)
	return out.String(), errOut.String(), code
}

// On a terminal the first row says which of the three states the report
// describes, in words, and the account sits under it. The object stays on
// stdout, because for this command the object is the answer.
func TestTheStatusScreenSaysTheStateAndTheAccount(t *testing.T) {
	t.Run("signed in", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("GDOC_CONFIG_DIR", dir)
		tokenWithScopes(t, dir, driveScope)
		stubAccount(t, gapi.Account{Email: "someone@example.com", Name: "Someone"}, nil)

		stdout, stderr, code := statusScreenRun(t)
		if code != 0 {
			t.Fatalf("status exits 0, and this run exited %d", code)
		}
		for _, want := range []string{"auth status", "✓ signed in", "someone@example.com", "drive", "bundled", "oauth"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("the panel must say %q, and it says\n%s", want, stderr)
			}
		}
		if got := decodeOne(t, strings.NewReader(stdout)); got["ok"] != true {
			t.Fatalf("the object is still the answer: %v", got)
		}
	})

	t.Run("signed out", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

		_, stderr, code := statusScreenRun(t)
		if code != 0 {
			t.Fatalf("signed out is an answer and exits 0, and this run exited %d", code)
		}
		if !strings.Contains(stderr, "✗ signed out") {
			t.Errorf("the panel must say signed out, and it says\n%s", stderr)
		}
		if strings.Contains(stderr, "@") {
			t.Errorf("there is no account to name, and the panel named one:\n%s", stderr)
		}
	})

	t.Run("scopes missing", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("GDOC_CONFIG_DIR", dir)
		tokenWithScopes(t, dir, documentsScope)

		_, stderr, code := statusScreenRun(t)
		if code != 0 {
			t.Fatalf("a partial grant is reported, not refused: exit %d", code)
		}
		if !strings.Contains(stderr, "! scopes missing") {
			t.Errorf("the panel must say scopes missing, and it says\n%s", stderr)
		}
		if !strings.Contains(stderr, "missing") || !strings.Contains(stderr, "drive") {
			t.Errorf("the panel must name what is missing, and it says\n%s", stderr)
		}
	})
}

// A pipe reads the object and nothing else: stderr stays as empty as it has
// always been, because a skill's log is not a place to draw a box.
func TestAuthStatusOnAPipeWritesNoStderr(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	tokenWithScopes(t, dir, driveScope)

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"auth", "status"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("status exits 0, and this run exited %d", code)
	}
	if errOut.String() != "" {
		t.Errorf("a pipe reads no screen, and stderr carried %q", errOut.String())
	}
	if got := decodeOne(t, &out); got["ok"] != true {
		t.Fatalf("envelope: %v", got)
	}
}
