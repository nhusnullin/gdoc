package auth

// The browser trip: the URL that is printed, the callback that comes back, and
// what the whole flow leaves on disk.

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"gdoc/internal/guard"
)

// MissingScopes has to know that one scope can stand for another. The Docs API
// accepts the full Drive scope, so a token holding drive is not missing the
// documents scope, and a warning that fires on a working token teaches people
// to ignore the warning.
func TestMissingScopesReadsDriveAsCoveringDocs(t *testing.T) {
	const (
		drive        = "https://www.googleapis.com/auth/drive"
		docs         = "https://www.googleapis.com/auth/documents"
		docsReadonly = "https://www.googleapis.com/auth/documents.readonly"
		driveFile    = "https://www.googleapis.com/auth/drive.file"
	)
	cases := []struct {
		name string
		have []string
		want []string
	}{
		{"the login this binary makes", []string{drive, docs}, nil},
		{"drive plus the read-only Docs scope", []string{drive, docsReadonly}, nil},
		{"drive on its own", []string{drive}, nil},
		{"docs on its own", []string{docs}, []string{drive}},
		{"nothing at all", nil, []string{drive, docs}},
		{"drive.file is not drive", []string{driveFile, docs}, []string{drive}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MissingScopes(c.have); !reflect.DeepEqual(got, c.want) {
				t.Errorf("MissingScopes(%v) = %v, want %v", c.have, got, c.want)
			}
		})
	}
}

func TestAuthURLCarriesPKCE(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	verifier, authURL := buildAuthURL("http://127.0.0.1:9999/callback", "STATE1")

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("PKCE method: %q", q.Get("code_challenge_method"))
	}
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if q.Get("code_challenge") != want {
		t.Fatal("challenge does not match verifier")
	}
	if !strings.Contains(q.Get("scope"), "auth/drive") || !strings.Contains(q.Get("scope"), "auth/documents") {
		t.Fatalf("scopes: %q", q.Get("scope"))
	}
	if q.Get("redirect_uri") != "http://127.0.0.1:9999/callback" {
		t.Fatalf("redirect_uri: %q", q.Get("redirect_uri"))
	}
	if q.Get("state") != "STATE1" {
		t.Fatalf("state: %q", q.Get("state"))
	}
	if q.Get("response_type") != "code" {
		t.Fatalf("response_type: %q", q.Get("response_type"))
	}
}

// Two runs must not share a verifier, or the code challenge proves nothing.
func TestAuthURLVerifierIsFreshEachTime(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	first, _ := buildAuthURL("http://127.0.0.1:9999/callback", "S")
	second, _ := buildAuthURL("http://127.0.0.1:9999/callback", "S")

	if first == second {
		t.Fatal("the verifier is reused between logins")
	}
}

// TestLoginRoundTrip plays the whole flow with no browser and no network: the
// test reads the URL off stderr, calls the loopback address itself, and checks
// the token landed in the temp config dir.
func TestLoginRoundTrip(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	rt := &formRT{}
	c := &http.Client{Transport: rt}
	w := chanWriter{lines: make(chan string, 4)}
	done := make(chan error, 1)

	go func() { done <- Login(c, w) }()

	authURL := urlFrom(t, <-w.lines)
	q := authURL.Query()
	callback := callbackWith(t, q.Get("redirect_uri"), url.Values{"code": {"CODE9"}, "state": {q.Get("state")}})
	resp, err := http.Get(callback)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if rt.seen.Get("code") != "CODE9" {
		t.Fatalf("the code from the callback did not reach the exchange: %q", rt.seen.Get("code"))
	}
	saved, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.RefreshToken != "R" {
		t.Fatalf("login did not save the token: %+v", saved)
	}
}

// A callback carrying somebody else's state is dropped, and the login goes on
// waiting for the real one. Anything on the machine can reach the loopback
// port, so a stray hit must be able neither to end the login nor to put its own
// code into it.
func TestAStrayCallbackDoesNotEndTheLogin(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	rt := &formRT{}
	c := &http.Client{Transport: rt}
	w := chanWriter{lines: make(chan string, 4)}
	done := make(chan error, 1)

	go func() { done <- Login(c, w) }()

	authURL := urlFrom(t, <-w.lines)
	q := authURL.Query()
	redirect := q.Get("redirect_uri")

	stray := callbackWith(t, redirect, url.Values{"code": {"SOMEBODY-ELSES-CODE"}, "state": {"NOT-THE-STATE"}})
	resp, err := http.Get(stray)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("a stray callback must not be answered with a success page")
	}

	real := callbackWith(t, redirect, url.Values{"code": {"CODE9"}, "state": {q.Get("state")}})
	resp, err = http.Get(real)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if err := <-done; err != nil {
		t.Fatalf("the real callback must still complete the login: %v", err)
	}
	if rt.seen.Get("code") != "CODE9" {
		t.Fatalf("the stray code reached the exchange: %q", rt.seen.Get("code"))
	}
}

// The authorization server saying no is not the same as no callback arriving,
// and the message the user gets has to say which happened.
func TestLoginReportsARefusedSignIn(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	c := &http.Client{Transport: &formRT{}}
	w := chanWriter{lines: make(chan string, 4)}
	done := make(chan error, 1)

	go func() { done <- Login(c, w) }()

	authURL := urlFrom(t, <-w.lines)
	q := authURL.Query()
	denied := callbackWith(t, q.Get("redirect_uri"), url.Values{
		"error": {"access_denied"},
		"state": {q.Get("state")},
	})
	resp, err := http.Get(denied)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	err = <-done
	if err == nil {
		t.Fatal("a refused sign-in must fail the login")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Fatalf("the error must say the user was refused, not that a code was missing: %v", err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("a refused login must not save a token")
	}
}

// A login that saves nothing is better than a login that overwrites a working
// token with one that expires in an hour and cannot be refreshed.
func TestARefreshlessExchangeDoesNotOverwriteAGoodToken(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	good := Token{AccessToken: "OLD", RefreshToken: "KEEPME", TokenURI: TokenURI,
		ClientID: "CID", ClientSecret: "CS", Expiry: time.Now().UTC().Add(time.Hour)}
	if err := Save(good); err != nil {
		t.Fatal(err)
	}
	c := &http.Client{Transport: jsonRT(`{"access_token":"A","expires_in":3600}`)}
	w := chanWriter{lines: make(chan string, 4)}
	done := make(chan error, 1)

	go func() { done <- Login(c, w) }()
	authURL := urlFrom(t, <-w.lines)
	q := authURL.Query()
	resp, err := http.Get(callbackWith(t, q.Get("redirect_uri"),
		url.Values{"code": {"CODE9"}, "state": {q.Get("state")}}))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if err := <-done; err == nil {
		t.Fatal("the login must fail")
	}
	saved, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.RefreshToken != "KEEPME" {
		t.Fatalf("the working token was overwritten: %+v", saved)
	}
}

// TestLoginThroughTheGuardsOwnClient is the composition the CLI does: the
// token exchange goes out through a guard-built client over a fake base. A
// change to TokenURI, or to the guard's token-host rule, breaks this and
// nothing else in the suite.
func TestLoginThroughTheGuardsOwnClient(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	rt := &formRT{}
	c := guard.NewClient(guard.NewPolicy(), rt)
	w := chanWriter{lines: make(chan string, 4)}
	done := make(chan error, 1)

	go func() { done <- Login(c, w) }()

	authURL := urlFrom(t, <-w.lines)
	q := authURL.Query()
	cb := callbackWith(t, q.Get("redirect_uri"), url.Values{"code": {"CODE9"}, "state": {q.Get("state")}})
	resp, err := http.Get(cb)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if err := <-done; err != nil {
		t.Fatalf("an empty policy still carries the token endpoint: %v", err)
	}
	if rt.seen.Get("code") != "CODE9" {
		t.Fatalf("the exchange did not reach the wire through the guard: %q", rt.seen.Get("code"))
	}
	if _, err := Load(); err != nil {
		t.Fatalf("the login did not save a token: %v", err)
	}
}

// The tests below drive a fake Google, which validates nothing, so any
// non-empty secret serves. The value is set here rather than in source
// because the real one is injected at build time and is never in the tree.
func init() {
	if BundledClientSecret == "" {
		BundledClientSecret = "test-client-secret"
	}
}

// A build without the secret cannot sign anyone in, and it says so before
// opening a listener or printing a URL, because a login that fails at the
// code exchange minutes later would blame Google for a build problem.
func TestLoginRefusesABuildWithNoClientSecret(t *testing.T) {
	saved := BundledClientSecret
	BundledClientSecret = ""
	t.Cleanup(func() { BundledClientSecret = saved })

	var w bytes.Buffer
	err := Login(&http.Client{Transport: refuseAll{}}, &w)
	if err == nil {
		t.Fatal("a build with no client secret must refuse to log in")
	}
	for _, want := range []string{"client secret", "GDOC_OAUTH_CLIENT_SECRET"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must say %q: %q", want, err)
		}
	}
	if w.Len() != 0 {
		t.Errorf("nothing must be printed before the refusal: %q", w.String())
	}
}

// refuseAll fails any request, so a test can prove nothing left the machine.
type refuseAll struct{}

func (refuseAll) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected request to %s", r.URL)
}
