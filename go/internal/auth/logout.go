// This file is the sign-out: the token file taken away, and nothing else.
// The rules it holds are in the package comment in doc.go.

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"gdoc/internal/config"
)

// LogoutReport is what a sign-out did: the file it was about, and whether
// there was one to remove. Removed is false and the answer is still ok when
// nobody was signed in, because somebody asking to be signed out of a
// computer with no token has been answered.
type LogoutReport struct {
	TokenPath string `json:"token_path"`
	Removed   bool   `json:"removed"`
}

// Logout removes the token file and reads its absence back.
//
// It takes nothing, because there is nothing to hand it: this is one file on
// this computer, and the grant itself stays at Google until somebody takes it
// away in their account settings. TestLogoutMakesNoRequest reads the file's
// own syntax tree for that.
//
// Four things are refused, each leaving the file exactly as it was: a sign-in
// waiting on a browser, anything at that path that is not a regular file,
// a file that will not parse, and a path gdoc cannot look at. doc.go holds
// why each one is refused rather than removed.
func Logout() (LogoutReport, error) {
	path, err := config.TokenPath()
	if err != nil {
		return LogoutReport{}, err
	}
	out := LogoutReport{TokenPath: path}

	// The waiting sign-in first, before the file is even looked at. A trip
	// halfway through a browser is about to write this file, and a sign-out
	// that ran in the middle of one would take away the token the trip is
	// going to replace, leaving a person reading "Signed in" in a tab that
	// signed them in to nothing.
	if err := noSignInWaiting(path); err != nil {
		return out, err
	}

	// Lstat and not Stat: a symlink is followed by everything else in the
	// standard library, and following one here would remove whatever is at the
	// other end of it, which is a file gdoc never wrote.
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, fmt.Errorf("the token at %s cannot be read, so nothing was removed: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return out, fmt.Errorf("%s is not a regular file but %s, so nothing was removed: the token gdoc writes is a file, and this is something else",
			path, kindOf(info.Mode()))
	}
	if err := parsesAsAToken(path); err != nil {
		return out, err
	}

	if err := os.Remove(path); err != nil {
		return out, fmt.Errorf("the token at %s could not be removed: %w", path, err)
	}
	// Nothing trusts a success. The removal is read back through its own
	// question, so an answer saying removed is one somebody looked at the disk
	// for.
	if _, err := os.Lstat(path); err == nil {
		return out, fmt.Errorf("%s is still there after being removed, so gdoc is still signed in", path)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return out, fmt.Errorf("%s was removed and gdoc cannot read whether it is gone: %w", path, err)
	}
	out.Removed = true
	return out, nil
}

// noSignInWaiting answers whether a browser trip is in flight, and refuses
// while one is. A lock that is there and cannot be read is refused too: not
// knowing must not resolve to taking a credential away, and the refusal names
// the file so whoever reads it can look.
func noSignInWaiting(token string) error {
	lock, err := ReadLoginLock(time.Now())
	if err != nil {
		return fmt.Errorf("%w, so the token at %s was left alone", err, token)
	}
	if lock == nil {
		return nil
	}
	path, err := config.LoginPendingPath()
	if err != nil {
		return err
	}
	return fmt.Errorf("a sign-in started at %s is waiting on a browser, written down at %s, so the token at %s was left alone. Finish it, or wait for it to run out, and run this again",
		lock.Started.Format(time.RFC3339), path, token)
}

// parsesAsAToken is the one reading logout makes of the file's contents: that
// it is the JSON object gdoc writes there.
//
// Load's own refusals are deliberately not borrowed. A token missing the
// fields a refresh needs is exactly the file a person wants taken away, and
// one that can no longer be refreshed must not be the one file gdoc refuses to
// remove: TestLogoutRemovesATokenARefreshCouldNotUse. What is left is the
// strict reading every input gets: a file gdoc half understands reaches
// nothing, and here reaching nothing means the file stays.
func parsesAsAToken(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("the token at %s cannot be read, so nothing was removed: %w", path, err)
	}
	var t Token
	if err := json.Unmarshal(b, &t); err != nil {
		return fmt.Errorf("the token at %s cannot be parsed, so nothing was removed: %w", path, err)
	}
	return nil
}

// kindOf names what is at the path, in the words a reader needs to go and look
// at it.
func kindOf(mode fs.FileMode) string {
	switch {
	case mode&fs.ModeSymlink != 0:
		return "a symbolic link"
	case mode.IsDir():
		return "a directory"
	case mode&fs.ModeDevice != 0:
		return "a device"
	case mode&fs.ModeNamedPipe != 0:
		return "a named pipe"
	case mode&fs.ModeSocket != 0:
		return "a socket"
	}
	return "an irregular file"
}
