package auth

// What `gdoc auth status` answers, in every state the token file can be in.

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStatusReportsThePresentToken(t *testing.T) {
	path := writeFixture(t)
	got, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	if !got.TokenPresent || got.TokenPath != path {
		t.Fatalf("status: %+v", got)
	}
	if got.Expired == nil || !*got.Expired {
		t.Fatal("a 2020 expiry must report expired")
	}
	if got.ClientSource != "bundled" {
		t.Fatalf("client source: %v", got.ClientSource)
	}
	if len(got.Scopes) != 1 {
		t.Fatalf("scopes: %v", got.Scopes)
	}
}

func TestStatusWithNoToken(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	got, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	if got.TokenPresent {
		t.Fatalf("status: %+v", got)
	}
	if got.Expired != nil {
		t.Fatal("with no token there is nothing to call expired")
	}
}

// v1 lets oauth-client.json override the bundled client. v2's login does not
// read it, so status must not claim it is in use: every v2 token belongs to the
// bundled client, and saying otherwise sends a person looking for a quota that
// was never separated.
func TestStatusSaysAClientFileIsNotUsedYet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "oauth-client.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientSource != "bundled" {
		t.Fatalf("every v2 login uses the bundled client: %v", got.ClientSource)
	}
	if !got.ClientFileIgnored {
		t.Fatalf("the file is there and unused; status must say so: %+v", got)
	}
}

// A token that exists and cannot be read is a fault to name. Reporting it as
// "signed out" is how somebody re-runs auth login, overwrites the file, and
// never learns what was wrong with it.
func TestStatusNamesAnUnreadableToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Status()
	if err == nil {
		t.Fatal("a token that cannot be parsed must not read as signed out")
	}
	if !strings.Contains(err.Error(), "oauth-token.json") {
		t.Fatalf("the error must name the file: %v", err)
	}
	if got == nil || got.TokenPath == "" {
		t.Fatalf("the facts still come back beside the error: %+v", got)
	}
}

// A partial grant is reported, not refused: the login worked, and the person
// has to be able to see why the Docs calls will fail.
func TestStatusNamesTheScopesTheTokenDoesNotCarry(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	if err := Save(Token{AccessToken: "A", RefreshToken: "R", TokenURI: TokenURI,
		ClientID: "CID", ClientSecret: "CS",
		Scopes: []string{"https://www.googleapis.com/auth/documents.readonly"},
		Expiry: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	out, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/documents",
	}
	if !reflect.DeepEqual(out.MissingScopes, want) {
		t.Fatalf("missing_scopes: %v, want %v", out.MissingScopes, want)
	}
}

// The Docs API accepts the full Drive scope on documents.get and
// documents.batchUpdate. A v1 token carries drive plus documents.readonly, so
// it can do everything v2 asks for, and a warning on the repository owner's own
// machine would be a false one.
func TestStatusIsQuietForAV1Token(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	if err := Save(Token{AccessToken: "A", RefreshToken: "R", TokenURI: TokenURI,
		ClientID: "CID", ClientSecret: "CS",
		Scopes: []string{
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/documents.readonly",
		},
		Expiry: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	out, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(out.MissingScopes) > 0 {
		t.Fatalf("a v1 token is missing nothing v2 needs: %v", out.MissingScopes)
	}
}

// A token carrying everything v2 asks for reports nothing, so the warning means
// something when it does appear.
func TestStatusIsQuietWhenEveryScopeIsThere(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	if err := Save(Token{AccessToken: "A", RefreshToken: "R", TokenURI: TokenURI,
		ClientID: "CID", ClientSecret: "CS",
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
