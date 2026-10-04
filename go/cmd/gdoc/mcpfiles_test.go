package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"gdoc/internal/auth"
)

// The sweep at a start takes away what nobody is using and leaves what somebody
// is. Claude Desktop starts two gdoc processes at once, one for the chat and
// one for agent mode (docs/v2/MEASURED.md, measurement 3), so a sweep that took
// every directory would take the other process's call apart while it ran.
func TestOnlyDeadProcessesTempDirectoriesAreRemovedAtStart(t *testing.T) {
	root := t.TempDir()
	const alive, dead = 4242, 4343

	saved := livePID
	t.Cleanup(func() { livePID = saved })
	livePID = func(pid int) bool { return pid == alive }

	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	old := now.Add(-5 * time.Minute)

	made := func(name string, at time.Time) string {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(dir, at, at); err != nil {
			t.Fatal(err)
		}
		return dir
	}

	keep := []string{
		// A live process's fresh directory, which is a call running now.
		made(fmt.Sprintf("gdoc-mcp-%d-fresh", alive), now.Add(-10*time.Second)),
		// Somebody else's directory, which matches the glob and carries no
		// process id, so nothing here may delete it.
		made("gdoc-mcp-notapid", old),
		// And a directory of another name entirely.
		made("something-else", old),
	}
	gone := []string{
		// A process that is not running: whatever it was doing, it stopped.
		made(fmt.Sprintf("gdoc-mcp-%d-leftover", dead), now.Add(-10*time.Second)),
		// A live process's directory older than any call can be.
		made(fmt.Sprintf("gdoc-mcp-%d-stale", alive), old),
	}

	sweepCallDirs(root, now)

	for _, dir := range keep {
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("%s had to be left alone: %v", filepath.Base(dir), err)
		}
	}
	for _, dir := range gone {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("%s had to be swept: %v", filepath.Base(dir), err)
		}
	}
}

// The age the sweep measures, as a literal. 200 seconds is the longest one call
// can last, so a directory older than that belongs to no call.
func TestTheSweepTakesADirectoryOlderThanTwoHundredSeconds(t *testing.T) {
	if mcpStaleAfter != 200*time.Second {
		t.Errorf("mcpStaleAfter = %v, want 200s", mcpStaleAfter)
	}
}

// livePID answers about the one process every test has at hand, so the real
// check is measured rather than only the stub.
func TestLivePIDKnowsThisProcess(t *testing.T) {
	if !livePID(os.Getpid()) {
		t.Error("this process is running and livePID says it is not")
	}
	if livePID(0) || livePID(-1) {
		t.Error("no process has id 0 or a negative id")
	}
}

// The sweep and the login lock ask one liveness rule, not two. Two readings
// would disagree about EPERM, a process somebody else owns, and about Windows,
// where there is no signal 0 to send: either disagreement ends with the sweep
// taking a live process's call directory away while it is reading from it.
func TestTheSweepAsksTheSameLivenessRuleAsTheLock(t *testing.T) {
	saved := livePID
	t.Cleanup(func() { livePID = saved })
	if reflect.ValueOf(saved).Pointer() != reflect.ValueOf(auth.ProcessAlive).Pointer() {
		t.Error("livePID is not auth.ProcessAlive, so the sweep and the login lock judge a process by two rules")
	}
}

// The process id is read back out of the directory name, which is the only
// record of who made it.
func TestADirectoryNameCarriesItsProcessID(t *testing.T) {
	for _, one := range []struct {
		name string
		pid  int
		ok   bool
	}{
		{"gdoc-mcp-4242-917263", 4242, true},
		{"gdoc-mcp-1-x", 1, true},
		{"gdoc-mcp-notapid", 0, false},
		{"gdoc-mcp-0-x", 0, false},
		{"gdoc-mcp-", 0, false},
	} {
		pid, ok := dirPID(filepath.Join("/tmp", one.name))
		if ok != one.ok || pid != one.pid {
			t.Errorf("dirPID(%q) = %d, %v, want %d, %v", one.name, pid, ok, one.pid, one.ok)
		}
	}
}

// Two values in one call share the call's directory, and each is written under
// its own fixed name. Nothing names a file after an argument: a call that could
// choose its own file name could choose one somewhere else.
func TestOneCallMakesOneDirectoryUnderFixedNames(t *testing.T) {
	files := &callFiles{}
	defer files.remove()

	first, err := files.write("body.txt", "body", "🤖 yes")
	if err != nil {
		t.Fatal(err)
	}
	second, err := files.write("annotations.json", "annotations", "[]")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(first) != filepath.Dir(second) {
		t.Errorf("one call is one directory: %s and %s", first, second)
	}
	if filepath.Base(first) != "body.txt" || filepath.Base(second) != "annotations.json" {
		t.Errorf("the file names are fixed: %s and %s", first, second)
	}
	body, err := os.ReadFile(first)
	if err != nil || string(body) != "🤖 yes" {
		t.Errorf("the file holds the value: %q, %v", body, err)
	}
	info, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("the file is mode %o, want 600", got)
	}
}

// A call that needs no file makes no directory. Three of the six never write
// one, and a directory made for nothing is a directory a sweep has to reason
// about.
func TestACallThatNeedsNoFileMakesNoDirectory(t *testing.T) {
	files := &callFiles{}
	defer files.remove()
	if files.dir != "" {
		t.Errorf("a call starts with no directory: %q", files.dir)
	}
	if got := files.name("nothing to rewrite"); got != "nothing to rewrite" {
		t.Errorf("a call with no files rewrites nothing: %q", got)
	}
}
