package auth

// The token file itself: reading it, refreshing it, writing it back.

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/config"
)

func TestLoadReadsV1Format(t *testing.T) {
	writeFixture(t)
	tok, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if tok.RefreshToken != "R1" || tok.ClientID != "CID" {
		t.Fatalf("bad load: %+v", tok)
	}
	if tok.TokenURI != "https://oauth2.googleapis.com/token" {
		t.Fatalf("token uri: %q", tok.TokenURI)
	}
	if !tok.Expired() {
		t.Fatal("a 2020 expiry is expired")
	}
}

func TestLoadWithNoTokenNamesTheFileAndTheFix(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	_, err := Load()
	if err == nil {
		t.Fatal("a missing token must be an error")
	}
	if !strings.Contains(err.Error(), "oauth-token.json") || !strings.Contains(err.Error(), "gdoc auth login") {
		t.Fatalf("the message must name the file and the fix: %v", err)
	}
}

func TestLoadRefusesAnUnreadableToken(t *testing.T) {
	// json.Unmarshal succeeds on any JSON value, so a file holding a string or
	// a bare null gets past the parse and would leave an empty token behind.
	for _, body := range []string{"{not json", `"signed out"`, `null`, `[]`} {
		t.Run(body, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("GDOC_CONFIG_DIR", dir)
			if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(); err == nil {
				t.Fatal("a token that cannot be parsed must not resolve to an empty token")
			}
		})
	}
}

func TestRefreshKeepsOldRefreshToken(t *testing.T) {
	writeFixture(t)
	tok, _ := Load()
	rt := &formRT{body: `{"access_token":"NEW","expires_in":3600}`}
	got, err := tok.Refresh(&http.Client{Transport: rt})
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "NEW" || got.RefreshToken != "R1" {
		t.Fatalf("refresh must keep the old refresh token: %+v", got)
	}
	if rt.seen.Get("grant_type") != "refresh_token" || rt.seen.Get("refresh_token") != "R1" {
		t.Fatalf("the form must ask for a refresh with the token on disk: %v", rt.seen)
	}
	if rt.seen.Get("client_id") != "CID" || rt.seen.Get("client_secret") != "CS" {
		t.Fatalf("the form must carry the client: %v", rt.seen)
	}
	if got.Expired() {
		t.Fatal("a fresh token is not expired")
	}
}

func TestRefreshTakesANewRefreshTokenWhenGiven(t *testing.T) {
	writeFixture(t)
	tok, _ := Load()
	got, err := tok.Refresh(&http.Client{Transport: &formRT{body: `{"access_token":"NEW","refresh_token":"R2","expires_in":3600}`}})
	if err != nil {
		t.Fatal(err)
	}
	if got.RefreshToken != "R2" {
		t.Fatalf("a rotated refresh token must be kept: %+v", got)
	}
}

func TestRefreshReportsTheEndpointError(t *testing.T) {
	writeFixture(t)
	tok, _ := Load()
	_, err := tok.Refresh(&http.Client{Transport: &formRT{status: 400, body: `{"error":"invalid_grant"}`}})
	if err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("want the endpoint error named, got %v", err)
	}
}

func TestRefuse200WithNoAccessToken(t *testing.T) {
	writeFixture(t)
	tok, _ := Load()
	if _, err := tok.Refresh(&http.Client{Transport: &formRT{body: `{"expires_in":3600}`}}); err == nil {
		t.Fatal("a 200 with no access token must not pass")
	}
}

// The transport failing is what the user sees when the guard refuses the POST
// or the network is down. It must reach the caller, not be swallowed.
func TestRefreshReportsATransportFailure(t *testing.T) {
	writeFixture(t)
	tok, _ := Load()
	c := &http.Client{Transport: rtFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("guard refused: host \"evil\"")
	})}
	if _, err := tok.Refresh(c); err == nil {
		t.Fatal("a refused POST must fail the refresh")
	} else if !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("the error must carry what went wrong: %v", err)
	}
}

func TestSaveIsCrashSafe(t *testing.T) {
	path := writeFixture(t)
	tok, _ := Load()
	tok.AccessToken = "SAVED"
	if err := Save(tok); err != nil {
		t.Fatal(err)
	}
	again, _ := Load()
	if again.AccessToken != "SAVED" {
		t.Fatal("save did not persist")
	}
	if again.RefreshToken != "R1" {
		t.Fatalf("save lost the rest of the token: %+v", again)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

func TestFailedSaveLeavesTheOriginalIntact(t *testing.T) {
	path := writeFixture(t)
	dir := filepath.Dir(path)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	tok, _ := Load()
	tok.AccessToken = "NEVER"
	err := Save(tok)
	if err == nil && os.Geteuid() == 0 {
		// Root writes through a read-only directory, so there is no failed
		// save to observe. Say so out loud rather than passing quietly.
		t.Skip("running as root: a read-only directory does not block the write, so this case cannot be staged here")
	}
	if err == nil {
		t.Fatal("the directory was read-only and the save still went through")
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	again, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if again.AccessToken != "OLD" {
		t.Fatalf("a failed save must not touch the original: %+v", again)
	}
}

// The token file holds a refresh token. Its permissions are part of the
// contract, not an accident of how Save happens to be written today.
func TestSaveWritesAPrivateFile(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", filepath.Join(t.TempDir(), "made-by-save"))
	if err := Save(Token{AccessToken: "A", RefreshToken: "R"}); err != nil {
		t.Fatal(err)
	}
	path, err := config.TokenPath()
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("the token file is %o, want 600", got)
	}
	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if got := di.Mode().Perm(); got != 0o700 {
		t.Errorf("the config dir is %o, want 700", got)
	}
}

// A token file written by google-auth carries fields v2 does not use. A v2 save
// must not quietly drop them from a file both tools share.
func TestSaveKeepsFieldsV2DoesNotUse(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	written := `{"token":"A","refresh_token":"R","client_id":"CID","client_secret":"CS",` +
		`"universe_domain":"googleapis.com","account":"nail@example.com","rapt_token":"RAPT"}`
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte(written), 0o600); err != nil {
		t.Fatal(err)
	}
	tok, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := Save(tok); err != nil {
		t.Fatal(err)
	}
	again, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if again.UniverseDomain != "googleapis.com" || again.Account != "nail@example.com" {
		t.Fatalf("a save dropped what google-auth wrote: %+v", again)
	}
	// rapt_token is the reauth proof token. google-auth writes it whenever the
	// account has one, and a v1 refresh needs it back.
	if again.RaptToken != "RAPT" {
		t.Fatalf("a save dropped rapt_token: %+v", again)
	}
}

// Load refuses a token file that parses but cannot be used. google-auth's
// from_authorized_user_info requires refresh_token, client_id and
// client_secret, so a file missing one of them is not a login: it is a file
// that will fail on the first refresh, after auth status has already said a
// token is present.
func TestLoadRefusesATokenMissingTheFieldsARefreshNeeds(t *testing.T) {
	cases := map[string]string{
		"empty object":     `{}`,
		"no refresh token": `{"client_id":"CID","client_secret":"CS"}`,
		"no client id":     `{"refresh_token":"R","client_secret":"CS"}`,
		"no client secret": `{"refresh_token":"R","client_id":"CID"}`,
		"blank client id":  `{"refresh_token":"R","client_id":"","client_secret":"CS"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("GDOC_CONFIG_DIR", dir)
			if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load()
			if err == nil {
				t.Fatal("a token that cannot be refreshed must not read as a token")
			}
			if errors.Is(err, ErrNoToken) {
				t.Fatalf("a file that exists is not a missing token: %v", err)
			}
			if !strings.Contains(err.Error(), "gdoc auth login") {
				t.Fatalf("the message must name the fix: %v", err)
			}
		})
	}
}

// The token endpoint is google-auth's constant and it overrides whatever the
// file says, so a file with no token_uri is usable. v2 fills the same value in
// rather than posting a refresh to an empty URL.
func TestLoadFillsInTheTokenEndpointWhenTheFileOmitsIt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	body := `{"token":"A","refresh_token":"R","client_id":"CID","client_secret":"CS"}`
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	tok, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if tok.TokenURI != TokenURI {
		t.Fatalf("token uri is %q, want %q", tok.TokenURI, TokenURI)
	}
}

func TestExpiredWithNoExpiry(t *testing.T) {
	var t0 Token
	if !t0.Expired() {
		t.Fatal("a token with no expiry is expired")
	}
}
