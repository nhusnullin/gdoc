// This file is the browser trip. It builds the authorization URL, prints it,
// waits on the loopback listener and exchanges the code. No browser is opened
// and no external program runs: v2 runs none at all, which is why the URL goes
// to stderr for a person to open.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
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

// loginScopes is v1's LOGIN_SCOPES: one browser trip covers Drive and Docs.
var loginScopes = []string{
	"https://www.googleapis.com/auth/drive",
	"https://www.googleapis.com/auth/documents",
}

// randomToken is the source of both the PKCE verifier and the state. Both must
// be unguessable, and both are fresh per login.
func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
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
func exchangeCode(c *http.Client, code, verifier, redirect string) (Token, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {BundledClientID},
		"client_secret": {BundledClientSecret},
		"redirect_uri":  {redirect},
	}
	resp, err := c.Post(TokenURI, "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return Token{}, fmt.Errorf("code exchange failed (%d): %s", resp.StatusCode, summarize(body))
	}
	var r struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := unmarshalToken(body, &r); err != nil {
		return Token{}, err
	}
	if r.AccessToken == "" {
		return Token{}, errors.New("code exchange returned 200 with no access token")
	}
	return Token{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		TokenURI:     TokenURI,
		ClientID:     BundledClientID,
		ClientSecret: BundledClientSecret,
		Scopes:       loginScopes,
		Expiry:       time.Now().UTC().Add(time.Duration(r.ExpiresIn) * time.Second),
	}, nil
}

// Login prints the authorization URL to w (stderr: stdout is reserved for the
// one JSON object) and waits. No browser is opened: v2 runs no external
// programs at all.
func Login(c *http.Client, w io.Writer) error {
	srv, err := loopback.Listen()
	if err != nil {
		return err
	}
	defer srv.Close()
	state := randomToken()
	redirect := "http://" + srv.Addr() + "/callback"
	verifier, authURL := buildAuthURL(redirect, state)
	fmt.Fprintf(w, "Open this link in your browser to sign in:\n%s\n", authURL)
	code, err := srv.WaitCode(state, loginTimeout)
	if err != nil {
		return err
	}
	tok, err := exchangeCode(c, code, verifier, redirect)
	if err != nil {
		return err
	}
	return Save(tok)
}
