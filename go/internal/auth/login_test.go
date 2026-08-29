package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"gdoc/internal/guard"
)

// exchangeRT stands in for the token endpoint. It records the form so the test
// can prove the verifier and the code went out, and nothing here reaches the
// network.
type exchangeRT struct{ seenVerifier, seenCode, seenGrant, seenRedirect string }

func (e *exchangeRT) RoundTrip(r *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(r.Body)
	form, _ := url.ParseQuery(string(b))
	e.seenVerifier = form.Get("code_verifier")
	e.seenCode = form.Get("code")
	e.seenGrant = form.Get("grant_type")
	e.seenRedirect = form.Get("redirect_uri")
	return &http.Response{StatusCode: 200, Request: r, Body: io.NopCloser(strings.NewReader(
		`{"access_token":"A","refresh_token":"R","expires_in":3600}`))}, nil
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

func TestExchangeSendsVerifierAndSaves(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	rt := &exchangeRT{}
	c := &http.Client{Transport: rt}

	tok, err := exchangeCode(c, "CODE7", "VERIF7", "http://127.0.0.1:9999/callback")
	if err != nil {
		t.Fatal(err)
	}

	if rt.seenCode != "CODE7" || rt.seenVerifier != "VERIF7" {
		t.Fatalf("exchange form wrong: %+v", rt)
	}
	if rt.seenGrant != "authorization_code" {
		t.Fatalf("grant_type: %q", rt.seenGrant)
	}
	if rt.seenRedirect != "http://127.0.0.1:9999/callback" {
		t.Fatalf("redirect_uri: %q", rt.seenRedirect)
	}
	if tok.RefreshToken != "R" || tok.AccessToken != "A" {
		t.Fatalf("token: %+v", tok)
	}
	if tok.ClientID != BundledClientID || tok.TokenURI != TokenURI {
		t.Fatalf("token does not carry the client it was issued to: %+v", tok)
	}
	if !tok.Expiry.After(time.Now().Add(59 * time.Minute)) {
		t.Fatalf("expiry: %v", tok.Expiry)
	}
}

// A refusal from the token endpoint must not be read as a token.
func TestExchangeReportsAFailedCode(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	c := &http.Client{Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 400, Request: r, Body: io.NopCloser(strings.NewReader(
			`{"error":"invalid_grant"}`))}, nil
	})}

	if _, err := exchangeCode(c, "STALE", "V", "http://127.0.0.1:9999/callback"); err == nil {
		t.Fatal("a 400 from the token endpoint must be an error")
	} else if !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("the error must name what the endpoint said: %v", err)
	}
}

// A 200 that carries no token is not a login. Not knowing must not resolve to
// "signed in with an empty token".
func TestExchangeRefusesA200WithNoToken(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	c := &http.Client{Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Request: r, Body: io.NopCloser(strings.NewReader(
			`{"expires_in":3600}`))}, nil
	})}

	if _, err := exchangeCode(c, "CODE", "V", "http://127.0.0.1:9999/callback"); err == nil {
		t.Fatal("a 200 with no access token must not pass")
	}
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// chanWriter hands the login URL to the test the moment Login prints it, so
// the test can play the browser.
type chanWriter struct{ lines chan string }

func (c chanWriter) Write(p []byte) (int, error) {
	c.lines <- string(p)
	return len(p), nil
}

// TestLoginRoundTrip plays the whole flow with no browser and no network: the
// test reads the URL off stderr, calls the loopback address itself, and checks
// the token landed in the temp config dir.
func TestLoginRoundTrip(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	rt := &exchangeRT{}
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
	if rt.seenCode != "CODE9" {
		t.Fatalf("the code from the callback did not reach the exchange: %q", rt.seenCode)
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
	rt := &exchangeRT{}
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
	if rt.seenCode != "CODE9" {
		t.Fatalf("the stray code reached the exchange: %q", rt.seenCode)
	}
}

// The authorization server saying no is not the same as no callback arriving,
// and the message the user gets has to say which happened.
func TestLoginReportsARefusedSignIn(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	c := &http.Client{Transport: &exchangeRT{}}
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

// TestLoginThroughTheGuardsOwnClient is the composition the CLI does: the
// token exchange goes out through a guard-built client over a fake base. A
// change to TokenURI, or to the guard's token-host rule, breaks this and
// nothing else in the suite.
func TestLoginThroughTheGuardsOwnClient(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	rt := &exchangeRT{}
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
	if rt.seenCode != "CODE9" {
		t.Fatalf("the exchange did not reach the wire through the guard: %q", rt.seenCode)
	}
	if _, err := Load(); err != nil {
		t.Fatalf("the login did not save a token: %v", err)
	}
}

// urlFrom pulls the authorization URL out of the sentence Login prints.
func urlFrom(t *testing.T, printed string) *url.URL {
	t.Helper()
	i := strings.Index(printed, "https://")
	if i < 0 {
		t.Fatalf("no URL in the login message: %q", printed)
	}
	u, err := url.Parse(strings.TrimSpace(printed[i:]))
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// callbackWith builds the redirect the browser would follow.
func callbackWith(t *testing.T, redirect string, q url.Values) string {
	t.Helper()
	u, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
