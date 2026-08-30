package auth

// The code exchange: what goes to the token endpoint, and what a token is made
// of when it comes back.

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestExchangeSendsVerifierAndSaves(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	rt := &formRT{}
	c := &http.Client{Transport: rt}

	tok, err := exchangeCode(c, "CODE7", "VERIF7", "http://127.0.0.1:9999/callback")
	if err != nil {
		t.Fatal(err)
	}

	if rt.seen.Get("code") != "CODE7" || rt.seen.Get("code_verifier") != "VERIF7" {
		t.Fatalf("exchange form wrong: %v", rt.seen)
	}
	if rt.seen.Get("grant_type") != "authorization_code" {
		t.Fatalf("grant_type: %q", rt.seen.Get("grant_type"))
	}
	if rt.seen.Get("redirect_uri") != "http://127.0.0.1:9999/callback" {
		t.Fatalf("redirect_uri: %q", rt.seen.Get("redirect_uri"))
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
	c := &http.Client{Transport: jsonRT(`{"expires_in":3600}`)}

	if _, err := exchangeCode(c, "CODE", "V", "http://127.0.0.1:9999/callback"); err == nil {
		t.Fatal("a 200 with no access token must not pass")
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
