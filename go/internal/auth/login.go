// This file is the browser trip. It builds the authorization URL, prints it,
// waits on the loopback listener and exchanges the code. No browser is opened
// and no external program runs: gdoc runs none at all, which is why the URL
// goes to stderr for a person to open.

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gdoc/internal/auth/loopback"
)

// authEndpoint is a page the human opens in their own browser. gdoc never
// requests it, so it is not in the guard's host allowlist.
const authEndpoint = "https://accounts.google.com/o/oauth2/auth"

// loginTimeout is how long the listener waits for the browser to come back.
const loginTimeout = 3 * time.Minute

// loginScopes is what one browser trip asks for: the full Drive scope, and the
// read/write Docs scope.
//
// The Docs scope here is documents and not documents.readonly, because gdoc
// writes suggestions through the Docs API and the read-only scope cannot carry
// a batchUpdate. The two sets are not equal, and the difference runs one way. A
// token granting Drive plus documents.readonly is missing nothing gdoc needs,
// because Drive covers the Docs calls; see coveredBy below. A token this login
// writes records documents and not documents.readonly, so a reader that asks
// for the read-only scope by name asks for its own sign-in. Do not fold the two
// sets into one by calling them equal.
var loginScopes = []string{
	"https://www.googleapis.com/auth/drive",
	"https://www.googleapis.com/auth/documents",
}

// randomToken is the source of both the PKCE verifier and the state. Both must
// be unguessable, and both are fresh per login.
//
// crypto/rand.Read is documented never to fail on any supported platform: it
// fills the slice or the program dies. That is the guarantee this line rests
// on. If the source is ever swapped for one that can fail, this has to return
// an error, because a silent failure hands out an all-zero verifier and state.
func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func buildAuthURL(redirect, state string) (verifier, authURL string) {
	verifier = randomToken()
	sum := sha256.Sum256([]byte(verifier))
	q := url.Values{
		"client_id":             {BundledClientID},
		"redirect_uri":          {redirect},
		"response_type":         {"code"},
		"scope":                 {strings.Join(loginScopes, " ")},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
		"access_type":           {"offline"},
		"prompt":                {"consent"},
	}
	return verifier, authEndpoint + "?" + q.Encode()
}

// exchangeCode turns the callback's code into a token. The client comes from
// internal/guard, so this POST is judged like every other request.
func exchangeCode(ctx context.Context, c *http.Client, code, verifier, redirect string) (Token, error) {
	r, err := postTokenForm(ctx, c, TokenURI, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {BundledClientID},
		"client_secret": {BundledClientSecret},
		"redirect_uri":  {redirect},
	}, "code exchange")
	if err != nil {
		return Token{}, err
	}
	// A code exchange with no refresh token is not a login. The access token
	// would work for about an hour, then every refresh would post an empty
	// refresh_token and fail for good, with no way back except a fresh login,
	// and Save would already have written over the token that did work. The
	// request carries access_type=offline with prompt=consent, so a missing
	// refresh token is an anomaly rather than a normal reply.
	if r.RefreshToken == "" {
		return Token{}, fmt.Errorf("code exchange returned 200 with no refresh token, " +
			"so the sign-in would stop working within the hour. Nothing was saved. Try signing in again")
	}
	return Token{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		TokenURI:     TokenURI,
		ClientID:     BundledClientID,
		ClientSecret: BundledClientSecret,
		Scopes:       grantedScopes(r.Scope),
		Expiry:       time.Now().UTC().Add(time.Duration(r.ExpiresIn) * time.Second),
	}, nil
}

// grantedScopes is what the token actually carries, which is what belongs in
// the file. Recording the requested list instead would let auth status report
// scopes the token does not have, and a later 403 would have nothing in the
// file to explain it.
//
// Silence means the grant matched the request: RFC 6749 section 5.1 makes the
// scope field optional only in that case.
func grantedScopes(scope string) []string {
	if granted := strings.Fields(scope); len(granted) > 0 {
		return granted
	}
	return loginScopes
}

// coveredBy names the scopes that stand in for another one. The Docs API
// accepts the full Drive scope on documents.get and documents.batchUpdate, so a
// token holding drive can do everything the documents scope allows. Without
// this, a token that works reports a scope missing that it does not need, which
// is a warning on the working case, and a warning on the working case is one
// people learn to ignore. drive.file is deliberately not here: it reaches only
// files the app itself created.
var coveredBy = map[string][]string{
	"https://www.googleapis.com/auth/documents": {"https://www.googleapis.com/auth/drive"},
}

// MissingScopes names the scopes v2 asks for that a token does not carry. A
// partial grant is reported, never refused: the login worked, and the person
// has to be able to see why the Docs calls will fail.
func MissingScopes(have []string) []string {
	got := make(map[string]bool, len(have))
	for _, s := range have {
		got[s] = true
	}
	var missing []string
	for _, want := range loginScopes {
		if !satisfied(got, want) {
			missing = append(missing, want)
		}
	}
	return missing
}

func satisfied(got map[string]bool, want string) bool {
	if got[want] {
		return true
	}
	for _, wider := range coveredBy[want] {
		if got[wider] {
			return true
		}
	}
	return false
}

// Login prints the authorization URL to w (stderr: stdout is reserved for the
// one JSON object) and waits. No browser is opened: v2 runs no external
// programs at all. It opens a listener on a port the kernel picks on
// 127.0.0.1, and gives up after loginTimeout.
func Login(c *http.Client, w io.Writer) error {
	state := randomToken()
	srv, err := loopback.Listen(state)
	if err != nil {
		return err
	}
	defer srv.Close()
	redirect := "http://" + srv.Addr() + "/callback"
	verifier, authURL := buildAuthURL(redirect, state)
	fmt.Fprintf(w, "Open this link in your browser to sign in:\n%s\n", authURL)
	code, err := srv.WaitCode(loginTimeout)
	if err != nil {
		return err
	}
	// The login has no context of its own: nothing above it can cancel a run
	// that is already waiting on a browser, and loginTimeout is what bounds the
	// wait. The exchange after it is bounded by the client's timeout.
	tok, err := exchangeCode(context.Background(), c, code, verifier, redirect)
	if err != nil {
		return err
	}
	return Save(tok)
}
