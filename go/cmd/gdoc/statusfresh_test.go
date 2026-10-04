// What `gdoc auth status` says about a token the account read itself refreshed,
// and how its panel names the person.

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"gdoc/internal/gapi"
)

// tokenExpiring writes a token carrying the full drive scope that expires at
// expiry, written as RFC 3339.
func tokenExpiring(t *testing.T, dir, expiry string) {
	t.Helper()
	writeTokenFile(t, dir, `{"token":"ya29.not-a-real-token",`+
		`"refresh_token":"1//not-a-real-refresh",`+
		`"client_id":"not-a-real-client.apps.googleusercontent.com",`+
		`"client_secret":"not-a-real-secret",`+
		`"token_uri":"https://oauth2.googleapis.com/token",`+
		`"scopes":["https://www.googleapis.com/auth/drive"],`+
		`"expiry":"`+expiry+`"}`)
}

// The account read goes out through an ordinary session, which refreshes an
// expired access token and saves it before the request leaves. The report then
// says what the file says after that, so `expired` and the warning about the
// refresh never contradict each other in one object.
func TestExpiredIsWhatTheFileSaysAfterTheAccountRead(t *testing.T) {
	const refreshed = "the access token was refreshed and saved"
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	tokenExpiring(t, dir, "2000-01-01T00:00:00Z")
	old := accountOf
	accountOf = func(context.Context) (gapi.Account, []string, error) {
		tokenExpiring(t, dir, "2099-01-01T00:00:00Z")
		return gapi.Account{Email: "someone@example.com"}, []string{refreshed}, nil
	}
	t.Cleanup(func() { accountOf = old })

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("status is a report and must answer: %v (exit %d)", got, code)
	}
	if expired := statusObject(t, got)["expired"]; expired != false {
		t.Errorf("the session saved a fresh token, so expired must be false, and it is %v", expired)
	}
	if !hasWarning(warningsOf(t, got), refreshed) {
		t.Errorf("the refresh is still a warning: %v", got["warnings"])
	}
}

// A read that failed proves nothing about the file, so the report keeps what
// auth.Status read before it.
func TestAFailedAccountReadLeavesExpiredAsRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	tokenExpiring(t, dir, "2000-01-01T00:00:00Z")
	stubAccountWarnings(t, gapi.Account{}, nil, errors.New("the account read failed"))

	got, code := runJSON(t, "auth", "status")
	if code != 0 || got["ok"] != true {
		t.Fatalf("a token that is there is still a token: %v (exit %d)", got, code)
	}
	if expired := statusObject(t, got)["expired"]; expired != true {
		t.Errorf("nothing refreshed the token, so expired must stay true, and it is %v", expired)
	}
}

// The panel names the person beside the address when Google said who it is,
// and the address alone when it did not.
func TestThePanelNamesThePersonBesideTheAddress(t *testing.T) {
	t.Run("name and address", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("GDOC_CONFIG_DIR", dir)
		tokenWithScopes(t, dir, driveScope)
		stubAccount(t, gapi.Account{Email: "someone@example.com", Name: "Someone Example"}, nil)

		_, stderr, _ := statusScreenRun(t)
		if !strings.Contains(stderr, "Someone Example · someone@example.com") {
			t.Errorf("the account row must read \"Someone Example · someone@example.com\", and the panel says\n%s", stderr)
		}
	})

	t.Run("address only", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("GDOC_CONFIG_DIR", dir)
		tokenWithScopes(t, dir, driveScope)
		stubAccount(t, gapi.Account{Email: "someone@example.com"}, nil)

		_, stderr, _ := statusScreenRun(t)
		if !strings.Contains(stderr, "someone@example.com") || strings.Contains(stderr, " · someone@example.com") {
			t.Errorf("with no name the row is the address alone, and the panel says\n%s", stderr)
		}
	})
}
