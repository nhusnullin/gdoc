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
	"io/fs"
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

	// clientFileName is v1's per-user client override. v2 does not read it
	// yet: every v2 login uses the bundled client, and Status says so rather
	// than claiming an override that is not wired up.
	clientFileName = "oauth-client.json"

	// expirySkew treats an almost-expired token as expired, so a long upload
	// never crosses the line mid-flight.
	expirySkew = time.Minute

	// maxTokenBody caps what is read from the token endpoint.
	maxTokenBody = 1 << 20
)

// ErrNoToken is "there is no token file". It is separate from every other read
// failure on purpose: a file that exists and cannot be read is a fault to
// report, and only an absent one means signed out.
var ErrNoToken = errors.New("no OAuth token")

// Token is v1's oauth-token.json, the google-auth "authorized user" shape. The
// field names are the file's, so a person logged in through v1 is logged in
// here with no migration. UniverseDomain and Account are carried but never
// used: google-auth writes them, and a v2 save that dropped them would quietly
// rewrite a file both tools share.
type Token struct {
	AccessToken    string    `json:"token"`
	RefreshToken   string    `json:"refresh_token"`
	TokenURI       string    `json:"token_uri"`
	ClientID       string    `json:"client_id"`
	ClientSecret   string    `json:"client_secret"`
	Scopes         []string  `json:"scopes"`
	UniverseDomain string    `json:"universe_domain,omitempty"`
	Account        string    `json:"account,omitempty"`
	Expiry         time.Time `json:"expiry"`
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
	if errors.Is(err, fs.ErrNotExist) {
		return Token{}, fmt.Errorf("%w at %s. Run: gdoc auth login", ErrNoToken, path)
	}
	if err != nil {
		return Token{}, fmt.Errorf("the token at %s cannot be read: %w", path, err)
	}
	var t Token
	if err := json.Unmarshal(b, &t); err != nil {
		return Token{}, fmt.Errorf("the token at %s cannot be parsed: %w", path, err)
	}
	return t, nil
}

// tokenResponse is what both exchanges get back from the token endpoint.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// postTokenForm is the one place that talks to the token endpoint. Both the
// refresh and the login code exchange go through it, so a 200 nobody can parse,
// a non-200, and a 200 with no token all fail the same way in both.
func postTokenForm(c *http.Client, uri string, form url.Values, what string) (tokenResponse, error) {
	resp, err := c.Post(uri, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTokenBody))
	if err != nil {
		return tokenResponse{}, fmt.Errorf("%s: the token endpoint's answer could not be read: %w", what, err)
	}
	if resp.StatusCode != http.StatusOK {
		return tokenResponse{}, fmt.Errorf("%s failed (%d): %s", what, resp.StatusCode, summarize(body))
	}
	var r tokenResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return tokenResponse{}, fmt.Errorf("%s: the token endpoint returned 200 with an unreadable body", what)
	}
	if r.AccessToken == "" {
		return tokenResponse{}, fmt.Errorf("%s returned 200 with no access token", what)
	}
	return r, nil
}

// Refresh exchanges the refresh token for a new access token. The client comes
// from internal/guard, so this POST is judged like every other request.
func (t Token) Refresh(c *http.Client) (Token, error) {
	r, err := postTokenForm(c, t.TokenURI, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {t.RefreshToken},
		"client_id":     {t.ClientID},
		"client_secret": {t.ClientSecret},
	}, "token refresh")
	if err != nil {
		return Token{}, err
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
	// Sync before the rename. Without it a power cut can leave the renamed file
	// present and empty, which is the exact failure the temp-and-rename dance
	// exists to prevent.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	// Same-directory rename. Not guaranteed atomic on Windows; documented in
	// docs/v2/SPEC.md and covered by the M9 Windows smoke test.
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// Status is what `gdoc auth status` reports. It reads and writes nothing else:
// a status run must never refresh, move or rewrite the token file.
//
// The map comes back even when the error does. A token file that exists and
// cannot be read is a fault the caller must name: reporting it as "signed out"
// is how somebody re-runs auth login, overwrites the file, and never learns
// what was wrong with it.
func Status() (map[string]any, error) {
	path, err := config.TokenPath()
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"auth_mode":     "oauth", // v2 is OAuth only; it does not read v1's config
		"token_path":    path,
		"client_source": "bundled",
		"token_present": false,
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), clientFileName)); err == nil {
		// v1 lets this file override the bundled client. v2's login does not
		// read it yet, so saying "file" here would name a client no token was
		// ever issued to.
		out["client_file_ignored"] = true
	}
	tok, err := Load()
	if errors.Is(err, ErrNoToken) {
		return out, nil // no token is a fact to report, not a failure
	}
	if err != nil {
		return out, err
	}
	out["token_present"] = true
	out["expired"] = tok.Expired()
	out["scopes"] = tok.Scopes
	return out, nil
}
