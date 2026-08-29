package auth

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixture = `{"token":"OLD","refresh_token":"R1","token_uri":"https://oauth2.googleapis.com/token","client_id":"CID","client_secret":"CS","scopes":["https://www.googleapis.com/auth/drive"],"expiry":"2020-01-01T00:00:00Z"}`

func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	path := filepath.Join(dir, "oauth-token.json")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

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
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("a token that cannot be parsed must not resolve to an empty token")
	}
}

type fakeRT struct {
	body   string
	status int
	seen   string
}

func (f *fakeRT) RoundTrip(r *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(r.Body)
	f.seen = string(b)
	if !strings.Contains(f.seen, "grant_type=refresh_token") ||
		!strings.Contains(f.seen, "refresh_token=R1") {
		return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(`{"error":"bad form"}`)), Request: r}, nil
	}
	status := f.status
	if status == 0 {
		status = 200
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(f.body)), Request: r}, nil
}

func TestRefreshKeepsOldRefreshToken(t *testing.T) {
	writeFixture(t)
	tok, _ := Load()
	rt := &fakeRT{body: `{"access_token":"NEW","expires_in":3600}`}
	got, err := tok.Refresh(&http.Client{Transport: rt})
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "NEW" || got.RefreshToken != "R1" {
		t.Fatalf("refresh must keep the old refresh token: %+v", got)
	}
	if !strings.Contains(rt.seen, "client_id=CID") || !strings.Contains(rt.seen, "client_secret=CS") {
		t.Fatalf("the form must carry the client: %q", rt.seen)
	}
	if got.Expired() {
		t.Fatal("a fresh token is not expired")
	}
}

func TestRefreshTakesANewRefreshTokenWhenGiven(t *testing.T) {
	writeFixture(t)
	tok, _ := Load()
	got, err := tok.Refresh(&http.Client{Transport: &fakeRT{body: `{"access_token":"NEW","refresh_token":"R2","expires_in":3600}`}})
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
	_, err := tok.Refresh(&http.Client{Transport: &fakeRT{status: 400, body: `{"error":"invalid_grant"}`}})
	if err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("want the endpoint error named, got %v", err)
	}
}

func TestRefuse200WithNoAccessToken(t *testing.T) {
	writeFixture(t)
	tok, _ := Load()
	if _, err := tok.Refresh(&http.Client{Transport: &fakeRT{body: `{"expires_in":3600}`}}); err == nil {
		t.Fatal("a 200 with no access token must not pass")
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
	if err := Save(tok); err == nil {
		t.Skip("this filesystem let the write through; nothing to assert")
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

func TestStatusReportsThePresentToken(t *testing.T) {
	path := writeFixture(t)
	got, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	if got["token_present"] != true || got["token_path"] != path {
		t.Fatalf("status: %v", got)
	}
	if got["expired"] != true {
		t.Fatal("a 2020 expiry must report expired")
	}
	if got["client_source"] != "bundled" {
		t.Fatalf("client source: %v", got["client_source"])
	}
	scopes, ok := got["scopes"].([]string)
	if !ok || len(scopes) != 1 {
		t.Fatalf("scopes: %v", got["scopes"])
	}
}

func TestStatusWithNoToken(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	got, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	if got["token_present"] != false {
		t.Fatalf("status: %v", got)
	}
	if _, reported := got["expired"]; reported {
		t.Fatal("with no token there is nothing to call expired")
	}
}

func TestStatusSeesAClientFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "oauth-client.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	if got["client_source"] != "file" {
		t.Fatalf("a client file must win over the bundled one: %v", got["client_source"])
	}
}

func TestExpiredWithNoExpiry(t *testing.T) {
	var t0 Token
	if !t0.Expired() {
		t.Fatal("a token with no expiry is expired")
	}
}
