package auth

// The sign-out: one file removed, its absence read back, and four refusals
// that leave the file where it is. The rules are in doc.go.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestLogoutRemovesTheTokenAndReadsItsAbsenceBack is the ordinary run. The
// read back is the point: nothing trusts a success, so the answer says removed
// only once the file is gone.
func TestLogoutRemovesTheTokenAndReadsItsAbsenceBack(t *testing.T) {
	path := writeFixture(t)

	got, err := Logout()
	if err != nil {
		t.Fatalf("a token that is there is removed: %v", err)
	}
	if !got.Removed {
		t.Errorf("the answer says removed %t, and the file was there: %+v", got.Removed, got)
	}
	if got.TokenPath != path {
		t.Errorf("the answer names %q, want %q", got.TokenPath, path)
	}
	if _, err := os.Lstat(path); err == nil {
		t.Errorf("%s is still there", path)
	}
}

// Nothing to remove is an answer and not a failure: somebody asking to be
// signed out of a computer nobody is signed in on has been answered.
func TestLogoutWithNoTokenIsOkAndRemovesNothing(t *testing.T) {
	dir := configDir(t)

	got, err := Logout()
	if err != nil {
		t.Fatalf("no token is an answer: %v", err)
	}
	if got.Removed {
		t.Errorf("the answer says removed %t, and there was no token: %+v", got.Removed, got)
	}
	if names := dirNames(t, dir); len(names) != 0 {
		t.Errorf("the config dir holds %v, and the run had nothing to do", names)
	}
}

// The token gdoc writes is a regular file. Anything else at that path is
// something this binary did not put there, and following it would remove
// whatever is at the other end of it.
func TestLogoutRefusesASymlinkOrANonRegularFileAndRemovesNothing(t *testing.T) {
	t.Run("a symlink", func(t *testing.T) {
		dir := configDir(t)
		elsewhere := filepath.Join(t.TempDir(), "somebody-elses-token.json")
		if err := os.WriteFile(elsewhere, []byte(fixture), 0o600); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "oauth-token.json")
		if err := os.Symlink(elsewhere, path); err != nil {
			t.Fatal(err)
		}

		got, err := Logout()
		if err == nil {
			t.Fatal("a symlink at the token's path is refused")
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("the refusal must name the file: %q", err)
		}
		if got.Removed {
			t.Errorf("the answer says removed %t after a refusal", got.Removed)
		}
		if _, err := os.Lstat(path); err != nil {
			t.Errorf("the link is gone: %v", err)
		}
		if _, err := os.Stat(elsewhere); err != nil {
			t.Errorf("the file the link pointed at is gone: %v", err)
		}
	})

	t.Run("a directory", func(t *testing.T) {
		dir := configDir(t)
		path := filepath.Join(dir, "oauth-token.json")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}

		got, err := Logout()
		if err == nil {
			t.Fatal("a directory at the token's path is refused")
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("the refusal must name the file: %q", err)
		}
		if got.Removed {
			t.Errorf("the answer says removed %t after a refusal", got.Removed)
		}
		if _, err := os.Lstat(path); err != nil {
			t.Errorf("the directory is gone: %v", err)
		}
	})
}

// A file that will not parse is a file gdoc half understands, and not knowing
// never resolves to removing somebody's credential. The bytes are still there
// afterwards, so whoever reads the refusal can look at them.
func TestLogoutRefusesATokenItCannotParseAndLeavesIt(t *testing.T) {
	dir := configDir(t)
	path := filepath.Join(dir, "oauth-token.json")
	const half = "{\"token\": \"A\",\n"
	if err := os.WriteFile(path, []byte(half), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Logout()
	if err == nil {
		t.Fatal("a token that cannot be parsed is refused")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the refusal must name the file: %q", err)
	}
	if got.Removed {
		t.Errorf("the answer says removed %t after a refusal", got.Removed)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != half {
		t.Errorf("the file reads %q, %v; it must be byte for byte what it was", string(b), err)
	}
}

// A token missing the fields a refresh needs is exactly the token a person
// wants taken away, so Load's refusal is not borrowed here: the only reading
// logout makes is that the file is JSON gdoc wrote.
func TestLogoutRemovesATokenARefreshCouldNotUse(t *testing.T) {
	dir := configDir(t)
	path := filepath.Join(dir, "oauth-token.json")
	if err := os.WriteFile(path, []byte(`{"token":"A"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Logout()
	if err != nil {
		t.Fatalf("a token that parses is removed: %v", err)
	}
	if !got.Removed {
		t.Errorf("the answer says removed %t: %+v", got.Removed, got)
	}
}

// A sign-in is halfway through a browser trip, and the process waiting on it
// is about to write the token. Removing it now takes away the file that trip
// is going to replace, and leaves the person with a tab that says signed in.
func TestLogoutRefusesWhileASignInIsWaiting(t *testing.T) {
	dir := configDir(t)
	path := writeToken(t, dir)
	if err := WriteLoginLock(LoginLock{
		PID: os.Getpid(), URL: authEndpoint + "?client_id=x", Started: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	got, err := Logout()
	if err == nil {
		t.Fatal("a waiting sign-in is refused")
	}
	if !strings.Contains(err.Error(), "sign-in") {
		t.Errorf("the refusal must say what is in the way: %q", err)
	}
	if got.Removed {
		t.Errorf("the answer says removed %t after a refusal", got.Removed)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Errorf("the token is gone: %v", err)
	}
}

// A path this process cannot look at is the same not-knowing as a file it
// cannot parse: nothing is removed and the refusal names the file, so whoever
// reads it can go and look. An unreadable config dir is refused at the first
// reading gdoc cannot make, and the token is still there afterwards.
func TestLogoutRefusesAPathItCannotLookAtAndRemovesNothing(t *testing.T) {
	dir := configDir(t)
	path := writeToken(t, dir)
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	// Put it back whatever happens, or the temp dir cannot be cleaned up.
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if _, err := os.Lstat(path); err == nil {
		t.Skip("this user can read inside a dir with no permissions, so there is no unreadable path to test with")
	}

	got, err := Logout()
	if err == nil {
		t.Fatal("a path gdoc cannot look at is refused")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the refusal must name the file: %q", err)
	}
	if got.Removed {
		t.Errorf("the answer says removed %t after a refusal", got.Removed)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Errorf("the token is gone: %v", err)
	}
}

// The one file, and no other. The config dir holds the settings, the release
// stamp, the record and the waiting sign-in beside the token, and a sign-out
// is about the credential alone.
func TestLogoutTouchesNothingElseInTheConfigDir(t *testing.T) {
	dir := configDir(t)
	writeToken(t, dir)
	beside := []string{"settings.yaml", "last-check.json", "oauth-client.json", "completion.zsh"}
	for _, name := range beside {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("kept"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "presence"), 0o700); err != nil {
		t.Fatal(err)
	}

	if _, err := Logout(); err != nil {
		t.Fatal(err)
	}

	want := append([]string{"presence"}, beside...)
	sort.Strings(want)
	if got := dirNames(t, dir); !equalNames(got, want) {
		t.Errorf("the config dir holds %v, want %v", got, want)
	}
}

// TestLogoutMakesNoRequest is read off the file's own syntax tree. A sign-out
// is a file being removed on this computer: it tells Google nothing, and the
// grant stays there until somebody takes it away in their account settings.
// So nothing here may open a wire, and the call takes no client to open one
// with.
func TestLogoutMakesNoRequest(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "logout.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range f.Imports {
		name := strings.Trim(spec.Path.Value, `"`)
		if name == "net" || strings.HasPrefix(name, "net/") {
			t.Errorf("logout.go imports %s, and a sign-out reaches nothing", name)
		}
	}
	var seen bool
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Logout" {
			continue
		}
		seen = true
		if n := fn.Type.Params.NumFields(); n != 0 {
			t.Errorf("Logout takes %d parameters; it is handed nothing, a client least of all", n)
		}
	}
	if !seen {
		t.Error("logout.go declares no Logout, so this test reads nothing")
	}
}

// writeToken puts the fixture token in a config dir the caller already made.
func writeToken(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "oauth-token.json")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// dirNames is what is in a directory, sorted.
func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func equalNames(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
