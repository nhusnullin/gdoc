package main

// The credential half. It reads the token file gdoc already keeps and refreshes
// it, and it does nothing else: no login flow, no writing the token back.
//
// Reading that file from a second place is a duplication worth naming. It is
// accepted here because this is a measurement that runs once and answers a
// question, not a second tool that has to stay in step with the first. If the
// answer means gdoc grows a copy route, the route lives in gdoc's own guard and
// gapi, and this program is deleted.
//
// The shape is v1's, which is google-auth's "authorized user" file, and gdoc's
// internal/auth reads exactly the same fields.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type token struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"token"`
	Expiry       string `json:"expiry"`
}

func tokenPath() string {
	if p := os.Getenv("GDOC_TOKEN"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gdoc-agent", "oauth-token.json")
}

// accessToken loads the file and exchanges the refresh token for a fresh access
// token. It always refreshes rather than trusting the stored expiry: this runs
// for under a minute, one extra round trip costs nothing, and a stale clock is
// one more thing that could make a measurement lie.
func accessToken() (string, error) {
	path := tokenPath()
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("the gdoc token file could not be read at %s: %w (run `gdoc auth login` first)", path, err)
	}
	var t token
	if err := json.Unmarshal(b, &t); err != nil {
		return "", fmt.Errorf("the token file at %s is not the shape gdoc writes: %w", path, err)
	}
	if t.RefreshToken == "" || t.ClientID == "" || t.ClientSecret == "" {
		return "", fmt.Errorf("the token file at %s carries no refresh credential, so it cannot be refreshed", path)
	}

	form := url.Values{
		"client_id":     {t.ClientID},
		"client_secret": {t.ClientSecret},
		"refresh_token": {t.RefreshToken},
		"grant_type":    {"refresh_token"},
	}
	c := &http.Client{Timeout: 30 * time.Second}
	resp, err := c.Post("https://oauth2.googleapis.com/token",
		"application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("the token could not be refreshed: %w\n\n"+
			"If that is a TLS handshake timeout, run tools/tlsdiag: this network may be\n"+
			"interfering with Go's TLS 1.3, which looks exactly like a credential problem", err)
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("the token endpoint answered %d and its answer could not be read: %w", resp.StatusCode, err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("the token endpoint answered %d: %s %s", resp.StatusCode, out.Error, out.Description)
	}
	return out.AccessToken, nil
}
