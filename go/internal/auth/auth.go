// This file is the token itself: the file it lives in, the two form POSTs to
// the token endpoint, the crash-safe save, and the report auth status prints.
// The rules this file holds are in the package comment in doc.go.
package auth

import (
	"context"
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

	"gdoc/internal/atomicfile"
	"gdoc/internal/config"
)

const (
	// BundledClientID and BundledClientSecret are shipped on purpose, and the
	// client must stay User type Internal. The package comment in doc.go holds
	// both rules and what would change them.
	BundledClientID     = "4326046141-n9fho1g348nflsue7jdrj10dkst3a0a9.apps.googleusercontent.com"
	BundledClientSecret = "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"

	TokenURI = "https://oauth2.googleapis.com/token"

	// clientFileName is the per-user client override, and nothing here reads
	// it yet: every login uses the bundled client, and Status says so rather
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
// here with no migration. UniverseDomain, Account and RaptToken are carried but
// never used: google-auth's Credentials.to_json writes all three when they are
// set, and a v2 save that dropped one would quietly rewrite a file both tools
// share. RaptToken is the reauth proof token, and losing it makes v1 ask for
// reauthentication again.
type Token struct {
	AccessToken    string    `json:"token"`
	RefreshToken   string    `json:"refresh_token"`
	TokenURI       string    `json:"token_uri"`
	ClientID       string    `json:"client_id"`
	ClientSecret   string    `json:"client_secret"`
	Scopes         []string  `json:"scopes"`
	RaptToken      string    `json:"rapt_token,omitempty"`
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
	if missing := t.missingFields(); len(missing) > 0 {
		return Token{}, fmt.Errorf("the token at %s is missing %s, so it cannot be refreshed. Run: gdoc auth login",
			path, strings.Join(missing, ", "))
	}
	// google-auth overrides token_uri with its own constant whichever value the
	// file carries, so a file without one is still a working login. Filling it
	// in here is what stops a refresh posting to an empty URL.
	if t.TokenURI == "" {
		t.TokenURI = TokenURI
	}
	return t, nil
}

// missingFields names the fields an authorized-user token must carry. The set
// is google-auth's: from_authorized_user_info refuses a file without
// refresh_token, client_id or client_secret, and v1 reads this same file
// through it. A file that parses but has none of them is not "signed in": it is
// a file that fails on the first refresh, after auth status has already
// reported a token present.
func (t Token) missingFields() []string {
	var missing []string
	for _, f := range []struct {
		name, value string
	}{
		{"refresh_token", t.RefreshToken},
		{"client_id", t.ClientID},
		{"client_secret", t.ClientSecret},
	} {
		if f.value == "" {
			missing = append(missing, f.name)
		}
	}
	return missing
}

// tokenResponse is what both exchanges get back from the token endpoint.
//
// Scope is what the authorization server GRANTED, which is not always what was
// asked for: Google's granular consent screen lets a person tick a subset. RFC
// 6749 section 5.1 makes the field optional only when the grant matches the
// request, so an absent scope means the whole requested set was granted.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
}

// postTokenForm is the one place that talks to the token endpoint. Both the
// refresh and the login code exchange go through it, so a 200 nobody can parse,
// a non-200, and a 200 with no token all fail the same way in both.
//
// It carries the caller's context, like every other request gdoc makes. A
// refresh happens inside a poll, and a poll inside `comments --wait` runs on
// that call's deadline: sent with http.Client.Post the POST would have carried
// no context at all, so its only bound was the guard's own client timeout,
// which is five minutes on top of the wait. A token endpoint that accepts and
// never answers would then take a nine minute wait past the ten it was chosen
// to fit inside, and a Ctrl-C could not end it either.
func postTokenForm(ctx context.Context, c *http.Client, uri string, form url.Values, what string) (tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.Do(req)
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
func (t Token) Refresh(ctx context.Context, c *http.Client) (Token, error) {
	r, err := postTokenForm(ctx, c, t.TokenURI, url.Values{
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
	// 0o600 whatever the file carried before: it is a credential, and a token
	// somebody made group readable is a token this write narrows back.
	return atomicfile.Replace(path, b, 0o600)
}

// StatusReport is the answer `gdoc auth status` prints, and the json tags are
// the field names a skill reads. It is a struct rather than a map so the
// command that renders it names the fields in Go: a key spelled in two packages
// is a warning that disappears the day one of them is renamed.
//
// The three optional fields are absent, not empty, when there is no token to
// describe. Expired is a pointer for that reason: "not expired" and "there is
// no token" are different answers.
type StatusReport struct {
	AuthMode          string   `json:"auth_mode"`
	TokenPath         string   `json:"token_path"`
	ClientSource      string   `json:"client_source"`
	TokenPresent      bool     `json:"token_present"`
	ClientFileIgnored bool     `json:"client_file_ignored,omitempty"`
	Expired           *bool    `json:"expired,omitempty"`
	Scopes            []string `json:"scopes,omitempty"`
	MissingScopes     []string `json:"missing_scopes,omitempty"`
}

// Status is what `gdoc auth status` reports. It reads and writes nothing else:
// a status run must never refresh, move or rewrite the token file.
//
// The report comes back even when the error does. A token file that exists and
// cannot be read is a fault the caller must name: reporting it as "signed out"
// is how somebody re-runs auth login, overwrites the file, and never learns
// what was wrong with it. Only a config dir gdoc cannot locate at all leaves
// nothing to report, and then the report is nil.
func Status() (*StatusReport, error) {
	path, err := config.TokenPath()
	if err != nil {
		return nil, err
	}
	out := &StatusReport{
		AuthMode:     "oauth", // v2 is OAuth only; it does not read v1's config
		TokenPath:    path,
		ClientSource: "bundled",
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), clientFileName)); err == nil {
		// v1 lets this file override the bundled client. v2's login does not
		// read it yet, so saying "file" here would name a client no token was
		// ever issued to.
		out.ClientFileIgnored = true
	}
	tok, err := Load()
	if errors.Is(err, ErrNoToken) {
		return out, nil // no token is a fact to report, not a failure
	}
	if err != nil {
		return out, err
	}
	expired := tok.Expired()
	out.TokenPresent = true
	out.Expired = &expired
	out.Scopes = tok.Scopes
	// A token can carry less than v2 asks for, from a granular consent screen
	// where somebody ticked a subset, and then the Docs calls will 403. This is
	// the one place that can say why before they do.
	//
	// A v1 login is not such a case, and MissingScopes is where that is decided:
	// v1 asks for documents.readonly rather than the read/write Docs scope, but
	// it also asks for the full Drive scope, which the Docs API accepts. So a v1
	// token reports nothing missing.
	out.MissingScopes = MissingScopes(tok.Scopes)
	return out, nil
}
