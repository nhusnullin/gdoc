// This file is the one thing a waiting sign-in writes down: the lock beside
// the token, which two gdoc processes read so that only one of them opens a
// listener. The browser trip itself is login.go's.

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"gdoc/internal/atomicfile"
	"gdoc/internal/config"
)

// LoginLock is the sign-in one process is waiting on, as the file holds it:
// which process opened the listener, the link it handed out, and when.
//
// It is three facts and no credential. The PKCE verifier never leaves the
// process that made it, so a second process cannot finish this sign-in; all it
// can do is hand the same link to the same person rather than opening a second
// listener and a second browser trip. TestASecondProcessReusesTheWaitingListener
// in cmd/gdoc is the pin on that use.
type LoginLock struct {
	PID     int       `json:"pid"`
	URL     string    `json:"url"`
	Started time.Time `json:"started"`
}

// loginLockLife is how long a written-down sign-in is believed. It is
// loginTimeout, because past it the listener that wrote the lock has stopped
// waiting and the link in it opens on nothing:
// TestALockOlderThanThreeMinutesIsNotLive.
const loginLockLife = loginTimeout

// WriteLoginLock records the sign-in this process is waiting on. Owner-only
// and through internal/atomicfile, like the token beside it: a half-written
// lock would hand somebody a truncated link.
func WriteLoginLock(l LoginLock) error {
	path, err := config.LoginPendingPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Replace(path, b, 0o600)
}

// ReadLoginLock is the sign-in another process is waiting on, or nil when
// nothing live is written down. Three things make a lock dead: no process
// behind its pid, an age over loginLockLife, and a link that is not Google's own
// sign-in page.
//
// Nothing written down is nil and no error: a machine where nobody is signing
// in is the ordinary case. A file that is there and will not parse is an error
// instead, because not knowing must not read as "nothing is waiting": the
// caller starts its own sign-in either way, and it does that knowing what it
// found. TestAnUnreadableLockIsNamed is the pin.
func ReadLoginLock(now time.Time) (*LoginLock, error) {
	path, err := config.LoginPendingPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("the waiting sign-in at %s cannot be read: %w", path, err)
	}
	var l LoginLock
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("the waiting sign-in at %s cannot be parsed: %w", path, err)
	}
	if !isLoginURL(l.URL) || !ProcessAlive(l.PID) || now.Sub(l.Started) >= loginLockLife {
		return nil, nil
	}
	return &l, nil
}

// isLoginURL answers whether a link out of the lock is the page login.go builds.
//
// The lock is written by another process and the link in it is handed to a
// person with the words open this in your browser, so it is the one value here
// that crosses a boundary into somebody's hands. Anything running as this user
// can write that file, and a link this binary would never have made reads as a
// dead lock: the caller opens its own listener and hands out its own link.
// TestALockWhoseLinkIsNotGooglesSignInIsNotLive.
func isLoginURL(raw string) bool {
	return strings.HasPrefix(raw, authEndpoint+"?")
}

// RemoveLoginLock takes the record away. A lock that is not there is not an
// error: every path that ends a sign-in removes it, and more than one of them
// can run.
func RemoveLoginLock() error {
	path, err := config.LoginPendingPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// SignedInSince reports whether the token file was written at or after t and
// can be loaded. It is how a process holding no listener sees that the sign-in
// it was told about finished somewhere else.
//
// The instant is the file's, not the token's: a token carries its expiry and
// not the moment it was saved. TestSignedInSinceSeesATokenWrittenAfterTheStart
// is the pin.
func SignedInSince(t time.Time) bool {
	path, err := config.TokenPath()
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || info.ModTime().Before(t) {
		return false
	}
	_, err = Load()
	return err == nil
}

// ProcessAlive reports whether a process with this id is still there.
//
// Signal 0 is the POSIX question "is this process reachable", and it is the
// only portable way to ask. A process owned by somebody else answers EPERM,
// which is still a process that exists. On Windows there is no such signal,
// and os.FindProcess has already asked: it opens the process and fails when
// there is none, so the answer is the one it gave.
// TestALockWhoseProcessIsGoneIsNotLive is the pin on the platforms CI runs.
//
// It is exported because two rooms ask it, this lock and cmd/gdoc's sweep of
// the call directories a chat left behind, and two readings of one question
// would disagree about EPERM and about Windows: one of them would then take a
// live process's work away. TestTheSweepAsksTheSameLivenessRuleAsTheLock in
// cmd/gdoc is the pin on that, and TestLivePIDKnowsThisProcess measures this
// rule against the one process every test has at hand.
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}
