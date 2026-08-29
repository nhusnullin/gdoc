package auth

import (
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

// jsonRT answers every token request with one canned body.
func jsonRT(body string) rtFunc {
	return func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Request: r,
			Body: io.NopCloser(strings.NewReader(body))}, nil
	}
}

// A code exchange that comes back without a refresh token is not a login. The
// access token works for about an hour, then every refresh posts an empty
// refresh_token and fails for good, with no way back except a fresh login. The
// login asks for access_type=offline with prompt=consent, so a missing refresh
// token is an anomaly, not a normal reply.
func TestExchangeRefusesA200WithNoRefreshToken(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	c := &http.Client{Transport: jsonRT(`{"access_token":"A","expires_in":3600}`)}

	_, err := exchangeCode(c, "CODE", "V", "http://127.0.0.1:9999/callback")
	if err == nil {
		t.Fatal("a code exchange with no refresh token must not pass")
	}
	if !strings.Contains(err.Error(), "refresh token") {
		t.Fatalf("the error must name what was missing: %v", err)
	}
}

// A login that saves nothing is better than a login that overwrites a working
// token with one that expires in an hour and cannot be refreshed.
func TestARefreshlessExchangeDoesNotOverwriteAGoodToken(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	good := Token{AccessToken: "OLD", RefreshToken: "KEEPME", TokenURI: TokenURI,
		Expiry: time.Now().UTC().Add(time.Hour)}
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

// Google's granular consent screen lets a person grant a subset of what was
// asked for. The token file has to record what was granted, or auth status
// reports scopes the token does not carry and every later call 403s with
// nothing in the file to explain it.
func TestExchangeRecordsTheGrantedScopesNotTheRequestedOnes(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	c := &http.Client{Transport: jsonRT(
		`{"access_token":"A","refresh_token":"R","expires_in":3600,` +
			`"scope":"https://www.googleapis.com/auth/drive"}`)}

	tok, err := exchangeCode(c, "CODE", "V", "http://127.0.0.1:9999/callback")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://www.googleapis.com/auth/drive"}
	if !reflect.DeepEqual(tok.Scopes, want) {
		t.Fatalf("scopes: %v, want the granted set %v", tok.Scopes, want)
	}
}

// RFC 6749 section 5.1: the scope field is optional only when the grant matches
// the request. So silence means the whole requested set was granted.
func TestExchangeFallsBackToTheRequestedScopesWhenTheEndpointSaysNothing(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	c := &http.Client{Transport: jsonRT(`{"access_token":"A","refresh_token":"R","expires_in":3600}`)}

	tok, err := exchangeCode(c, "CODE", "V", "http://127.0.0.1:9999/callback")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tok.Scopes, loginScopes) {
		t.Fatalf("scopes: %v, want the requested set %v", tok.Scopes, loginScopes)
	}
}

// A partial grant is reported, not refused: the login worked, and the person
// has to be able to see why the Docs calls will fail.
func TestStatusNamesTheScopesTheTokenDoesNotCarry(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	if err := Save(Token{AccessToken: "A", RefreshToken: "R", TokenURI: TokenURI,
		Scopes: []string{"https://www.googleapis.com/auth/drive"},
		Expiry: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	out, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out.MissingScopes, []string{"https://www.googleapis.com/auth/documents"}) {
		t.Fatalf("missing_scopes: %v", out.MissingScopes)
	}
}

// A token carrying everything v2 asks for reports nothing, so the warning means
// something when it does appear.
func TestStatusIsQuietWhenEveryScopeIsThere(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	if err := Save(Token{AccessToken: "A", RefreshToken: "R", TokenURI: TokenURI,
		Scopes: loginScopes, Expiry: time.Now().UTC().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	out, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(out.MissingScopes) > 0 {
		t.Fatalf("a full grant must report nothing: %+v", out)
	}
}
