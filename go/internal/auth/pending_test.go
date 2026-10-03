package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gdoc/internal/config"
)

// configDir points the whole package at a temp directory, so no test here
// reads or writes the real token or the real lock.
func configDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	return dir
}

// deadPID is a process id no machine has. macOS and Linux both stay well
// under it, and a pid nothing answers for is what a lock left by a process
// that died looks like.
const deadPID = 99999998

func TestALoginLockIsWrittenAndReadBack(t *testing.T) {
	dir := configDir(t)
	started := time.Now()
	want := LoginLock{PID: os.Getpid(), URL: "https://accounts.google.com/o/oauth2/auth?x=1", Started: started}
	if err := WriteLoginLock(want); err != nil {
		t.Fatal(err)
	}

	got, err := ReadLoginLock(started.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("a lock this process holds must read as live")
	}
	if got.PID != want.PID || got.URL != want.URL || !got.Started.Equal(started) {
		t.Fatalf("read back %+v, want %+v", *got, want)
	}

	// It is a record of a sign-in in progress, so it is owner-only like the
	// token beside it.
	info, err := os.Stat(filepath.Join(dir, "login-pending.json"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("the lock file is mode %v, want 0600", info.Mode().Perm())
	}
}

func TestNoLoginLockIsNoErrorAndNoLock(t *testing.T) {
	configDir(t)
	got, err := ReadLoginLock(time.Now())
	if err != nil || got != nil {
		t.Fatalf("got %v, %v; nothing written down is nothing waiting", got, err)
	}
}

// A lock whose process is gone is a lock nothing is behind. Reading it as live
// would leave every later gdoc process answering with a link whose listener
// died with the process that opened it.
func TestALockWhoseProcessIsGoneIsNotLive(t *testing.T) {
	configDir(t)
	if err := WriteLoginLock(LoginLock{PID: deadPID, URL: "https://accounts.google.com/x", Started: time.Now()}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLoginLock(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("a lock naming pid %d read as live", deadPID)
	}
}

// Three minutes, stated as a literal: it is the same wait the listener itself
// gives up after, so a lock older than that is behind a listener that has
// stopped listening.
func TestALockOlderThanThreeMinutesIsNotLive(t *testing.T) {
	configDir(t)
	now := time.Now()
	write := func(age time.Duration) {
		t.Helper()
		if err := WriteLoginLock(LoginLock{PID: os.Getpid(), URL: "https://accounts.google.com/x", Started: now.Add(-age)}); err != nil {
			t.Fatal(err)
		}
	}

	write(2*time.Minute + 59*time.Second)
	got, err := ReadLoginLock(now)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("a sign-in started under three minutes ago is still waiting")
	}

	write(3*time.Minute + time.Second)
	got, err = ReadLoginLock(now)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("a sign-in started over three minutes ago is not waiting for anyone")
	}
}

// A lock with no link in it is nothing to answer with, whatever its pid says.
func TestALockWithNoLinkIsNotLive(t *testing.T) {
	configDir(t)
	if err := WriteLoginLock(LoginLock{PID: os.Getpid(), Started: time.Now()}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLoginLock(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("a lock carrying no link read as live")
	}
}

// A file that cannot be read is named rather than read as "nothing is
// waiting". The caller starts its own sign-in either way, and that is its
// decision to take knowing what it found.
func TestAnUnreadableLockIsNamed(t *testing.T) {
	dir := configDir(t)
	if err := os.WriteFile(filepath.Join(dir, "login-pending.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLoginLock(time.Now())
	if err == nil {
		t.Fatal("a lock that will not parse must be reported")
	}
	if got != nil {
		t.Fatalf("got %+v beside the error", *got)
	}
}

func TestRemovingALockThatIsNotThereIsNoError(t *testing.T) {
	dir := configDir(t)
	if err := RemoveLoginLock(); err != nil {
		t.Fatalf("removing nothing failed: %v", err)
	}
	if err := WriteLoginLock(LoginLock{PID: os.Getpid(), URL: "https://accounts.google.com/x", Started: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveLoginLock(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "login-pending.json")); err == nil {
		t.Fatal("the lock is still there")
	}
}

// How a process that holds no listener sees that the sign-in finished
// somewhere else: the token file is newer than the instant the waiting one
// started.
func TestSignedInSinceSeesATokenWrittenAfterTheStart(t *testing.T) {
	configDir(t)
	started := time.Now()
	if SignedInSince(started) {
		t.Fatal("no token file at all is nobody signed in")
	}

	writeTokenFixture(t)
	if !SignedInSince(started.Add(-time.Minute)) {
		t.Fatal("a token written after the sign-in started is the sign-in finishing")
	}
	if SignedInSince(started.Add(time.Hour)) {
		t.Fatal("a token older than the waiting sign-in is a different sign-in")
	}
}

// writeTokenFixture writes a token the loader accepts into the config dir the
// test set.
func writeTokenFixture(t *testing.T) {
	t.Helper()
	path, err := config.TokenPath()
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(Token{
		AccessToken:  "A",
		RefreshToken: "R",
		TokenURI:     TokenURI,
		ClientID:     "CID",
		ClientSecret: "CS",
		Scopes:       []string{"https://www.googleapis.com/auth/drive"},
		Expiry:       time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}
