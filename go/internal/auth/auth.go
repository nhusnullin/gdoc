// Package auth holds the OAuth token and the bundled client. The secret is
// deliberately in version control; see CLAUDE.md and RFC 8252 section 8.5. A
// secret shipped to many users is not confidential and identifies the client,
// nothing more. What protects an account is the per-user token, which never
// leaves the machine.
//
// Refresh is one form POST, which is why x/oauth2 is not a dependency. This
// package never builds an HTTP client: it is handed one, and the only place
// that builds one is internal/guard.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gdoc/internal/config"
)

const (
	// BundledClientID and BundledClientSecret are v1's constants, copied
	// verbatim from gdoc/oauth.py. The client must stay User type Internal:
	// that exempts gdoc from OAuth verification, from the unverified-app
	// screen and from the 100-user cap, which matters because the Drive scope
	// it needs is a restricted scope.
	BundledClientID     = "4326046141-n9fho1g348nflsue7jdrj10dkst3a0a9.apps.googleusercontent.com"
	BundledClientSecret = "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"

	TokenURI = "https://oauth2.googleapis.com/token"

	// clientFileName, when it exists in the config dir, wins over the bundled
	// client. That is an override for quota, not a setup step.
	clientFileName = "oauth-client.json"

	// expirySkew treats an almost-expired token as expired, so a long upload
	// never crosses the line mid-flight.
	expirySkew = time.Minute
)

// Token is v1's oauth-token.json, the google-auth "authorized user" shape. The
// field names are the file's, so a person logged in through v1 is logged in
// here with no migration.
type Token struct {
	AccessToken  string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	TokenURI     string    `json:"token_uri"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	Scopes       []string  `json:"scopes"`
	Expiry       time.Time `json:"expiry"`
}

// Expired reports whether the access token is past use, with a minute of skew.
func (t Token) Expired() bool {
	return !t.Expiry.After(time.Now().Add(expirySkew))
}

// Load reads the token file. A file that cannot be parsed is an error, never an
// empty token: not knowing must not resolve to "signed out".
func Load() (Token, error) {
	path, err := config.TokenPath()
	if err != nil {
		return Token{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Token{}, fmt.Errorf("no OAuth token at %s. Run: gdoc auth login", path)
	}
	var t Token
	if err := json.Unmarshal(b, &t); err != nil {
		return Token{}, fmt.Errorf("the token at %s cannot be parsed: %w", path, err)
	}
	return t, nil
}

// Refresh exchanges the refresh token for a new access token. The client comes
// from internal/guard, so this POST is judged like every other request.
func (t Token) Refresh(c *http.Client) (Token, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {t.RefreshToken},
		"client_id":     {t.ClientID},
		"client_secret": {t.ClientSecret},
	}
	resp, err := c.Post(t.TokenURI, "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return Token{}, fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, summarize(body))
	}
	var r struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.AccessToken == "" {
		return Token{}, errors.New("token refresh returned 200 with no access token")
	}
	out := t
	out.AccessToken = r.AccessToken
	// Google usually omits the refresh token on a refresh. Keeping the old one
	// is what makes the next refresh work.
	if r.RefreshToken != "" {
		out.RefreshToken = r.RefreshToken
	}
	out.Expiry = time.Now().UTC().Add(time.Duration(r.ExpiresIn) * time.Second)
	return out, nil
}

func summarize(body []byte) string {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error != "" {
		return e.Error
	}
	return "unreadable error body"
}

// Save writes the token through a temp file in the same directory, then
// renames. A failed write cannot leave the old token truncated.
func Save(t Token) error {
	path, err := config.TokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".oauth-token-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	// Same-directory rename. Not guaranteed atomic on Windows; documented in
	// the spec and covered by the M9 Windows smoke test.
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// Status is what `gdoc auth status` reports. It reads and writes nothing else:
// a status run must never refresh, move or rewrite the token file.
func Status() (map[string]any, error) {
	path, err := config.TokenPath()
	if err != nil {
		return nil, err
	}
	dir, err := config.Dir()
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"auth_mode":     "oauth",
		"token_path":    path,
		"client_source": clientSource(dir),
		"token_present": false,
	}
	tok, err := Load()
	if err != nil {
		return out, nil // no token is a fact to report, not a failure
	}
	out["token_present"] = true
	out["expired"] = tok.Expired()
	out["scopes"] = tok.Scopes
	return out, nil
}

func clientSource(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, clientFileName)); err == nil {
		return "file"
	}
	return "bundled"
}
