//go:build unix

// The opener: the three questions gdoc asks before it starts the menu bar app,
// and what one command under the presence setting hands the seam.
//
// Nothing here starts a program and nothing here touches the ~/Applications of
// the machine the suite runs on: the opener goes through the recorder seam,
// the bundle is a folder this test made, and the lock is a file under a
// temporary config directory. The rows that stand for a running app hold a
// real flock on that file from the test process, because a flock belongs to
// the open file and not to the process: a second descriptor is refused the
// lock whoever holds it, which is the whole reason the app's liveness is this
// and not a pid.
//
// unix only, which is every platform the opener can answer yes on. A Windows
// build has no flock and openPresenceApp asks the platform first, so there is
// nothing there for a test to measure.

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"gdoc/internal/settings"
)

// holdTheLock takes the app's lock the way the app takes it and keeps it for
// the test, so a row that says the app is running is a kernel's answer rather
// than a flag in a fake.
func holdTheLock(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("the test could not take the lock at %s: %v", path, err)
	}
	t.Cleanup(func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	})
}

// openerSeams points the config directory, the bundle and the opener at this
// test's own, and answers the two paths the questions are about.
func openerSeams(t *testing.T, osName string, bundleThere, lockHeld bool) (*opened, string, string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	bundle := appAt(t)
	if bundleThere {
		appInstalled(t, bundle, "2.15.0")
	}
	lock := filepath.Join(dir, "presence", presenceLockName)
	if lockHeld {
		holdTheLock(t, lock)
	}
	return desktopSeams(t, osName), bundle, lock
}

// The eight combinations of the three questions. One of them starts the app,
// and it is the one where all three say yes.
func TestTheOpenerAsksThreeThingsAndRunsOpenGOnce(t *testing.T) {
	for _, row := range []struct {
		name        string
		osName      string
		bundleThere bool
		lockHeld    bool
		opens       bool
	}{
		{"macOS, installed, not running", "darwin", true, false, true},
		{"macOS, installed, running", "darwin", true, true, false},
		{"macOS, no app, no lock", "darwin", false, false, false},
		{"macOS, no app, lock held", "darwin", false, true, false},
		{"linux, installed, not running", "linux", true, false, false},
		{"linux, installed, running", "linux", true, true, false},
		{"linux, no app, no lock", "linux", false, false, false},
		{"linux, no app, lock held", "linux", false, true, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			rec, bundle, lock := openerSeams(t, row.osName, row.bundleThere, row.lockHeld)

			openPresenceApp()

			want := []started{{name: "/usr/bin/open", wait: false, args: []string{"-g", bundle}}}
			if !row.opens {
				want = nil
			}
			if !reflect.DeepEqual(rec.calls, want) {
				t.Errorf("the opener asked for %+v, and this row is %+v", rec.calls, want)
			}
			// And it creates nothing. The lock file is the app's to make: a
			// gdoc that made it would be writing into the record's directory
			// about a question it only asked.
			if _, err := os.Stat(lock); !row.lockHeld && !os.IsNotExist(err) {
				t.Errorf("%s was made by the opener: %v", lock, err)
			}
		})
	}
}

// One command, three runs. Under the flag with the app installed and not
// running it hands the seam the whole argv once; with the lock held it hands
// it nothing; with the flag off it hands it nothing either, because the flag
// off is today's bytes.
func TestACommandUnderTheFlagOpensTheAppOnceAndNothingWhenItIsRunning(t *testing.T) {
	for _, row := range []struct {
		name     string
		set      settings.InEffect
		lockHeld bool
		opens    bool
	}{
		{"the flag on and the app not running", presenceOn(), false, true},
		{"the flag on and the app running", presenceOn(), true, false},
		{"the flag off", defaultsInEffect(), false, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			rec, bundle, _ := openerSeams(t, "darwin", true, row.lockHeld)
			t.Setenv(claudeCodeEnv, "")
			stubSession(t, docsAndComments(t))

			var out, errOut bytes.Buffer
			if code := run(context.Background(), []string{"read", fixtureDocID}, &out, &errOut, row.set); code != 0 {
				t.Fatalf("the read must answer: exit %d, %s", code, errOut.String())
			}

			want := []started{{name: "/usr/bin/open", wait: false, args: []string{"-g", bundle}}}
			if !row.opens {
				want = nil
			}
			if !reflect.DeepEqual(rec.calls, want) {
				t.Errorf("the run asked for %+v, and this row is %+v", rec.calls, want)
			}
		})
	}
}
