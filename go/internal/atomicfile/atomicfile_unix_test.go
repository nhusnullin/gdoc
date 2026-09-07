//go:build unix

package atomicfile

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// A write that fails part-way leaves the file it was replacing untouched, and
// takes its own temp file with it. That rollback is one closure shared by the
// three steps between the create and the rename, so failing the first of them
// is failing all three.
//
// The failure is staged with RLIMIT_FSIZE, which is the one portable way to
// make a write to a perfectly good file descriptor fail: the temp file is
// created, the write comes back with EFBIG, and nothing about the machine has
// to be arranged. Go leaves SIGXFSZ ignored, so the process lives to assert.
//
// The limit is process wide, so it is put back before the first assertion. A
// t.Errorf while no regular file may be written is a test that cannot say what
// went wrong. Unix only, so this file is: the Windows half is the M9 smoke
// test's, along with the rename caveat the package doc records.
func TestAWriteThatFailsPartWayKeepsTheOldFileAndLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var old syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
		t.Skipf("the file size limit could not be read, so a failing write cannot be staged here: %v", err)
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: 0, Max: old.Max}); err != nil {
		t.Skipf("the file size limit could not be set to zero, so a failing write cannot be staged here: %v", err)
	}
	err := Replace(path, []byte("after\n"), 0o644)
	if restore := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &old); restore != nil {
		t.Fatalf("the file size limit was not put back, so nothing after this test can write a file: %v", restore)
	}

	if err == nil {
		t.Fatal("Replace() under a zero file size limit = nil, want the failed write reported")
	}
	if b, _ := os.ReadFile(path); string(b) != "before\n" {
		t.Errorf("contents = %q, want the old bytes: a write that failed replaced the file anyway", b)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "note.md" {
			t.Errorf("%s was left behind after a failed write", e.Name())
		}
	}
}
