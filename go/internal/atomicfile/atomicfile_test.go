package atomicfile

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReplaceKeepsTheModeItWasGiven(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("before\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := Replace(path, []byte("after\n"), ModeOf(path, 0o644)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Errorf("mode = %v, want 0640: a note somebody made group readable stays that way", got)
	}
	if b, _ := os.ReadFile(path); string(b) != "after\n" {
		t.Errorf("contents = %q, want the new bytes", b)
	}
}

func TestModeOfFallsBackWhenThereIsNoFileYet(t *testing.T) {
	if got := ModeOf(filepath.Join(t.TempDir(), "absent.md"), 0o644); got != 0o644 {
		t.Errorf("ModeOf(absent) = %v, want the fallback", got)
	}
}

// A failed write leaves nothing beside the file it did not replace. Temp files
// scattered through a vault on every failed run is its own kind of damage.
func TestAFailedWriteLeavesNoTempFileBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A mode Chmod refuses is hard to arrange portably; a directory in place of
	// the target makes the rename fail instead, which is the same rollback.
	target := filepath.Join(dir, "adir")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Replace(target, []byte("x"), 0o644); err == nil {
		t.Fatal("Replace() over a directory = nil, want an error")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "note.md" && e.Name() != "adir" {
			t.Errorf("%s was left behind after a failed write", e.Name())
		}
	}
}

// The file being replaced is never truncated first. A reader that opens it
// while the write is in flight sees the old bytes or the new ones.
//
// The assertion is on the mechanism rather than on what is left behind. A
// directory holding one entry afterwards is equally true of os.WriteFile, which
// truncates the file in place, so counting entries would let a refactor that
// empties a note or an OAuth token through. A handle opened before the write
// still reads the old bytes only because a temp file was renamed over the path
// and the old inode is still there.
//
// The open-handle half of that is POSIX's, and on Windows it cannot be staged:
// os.Open asks for FILE_SHARE_READ|FILE_SHARE_WRITE and not
// FILE_SHARE_DELETE, so the MoveFileEx behind os.Rename fails with a sharing
// violation while the handle is held, and the test would die on Replace rather
// than on an assertion of its own. That is the caveat the package doc records,
// and the M9 Windows smoke test is where it is measured. The contents and the
// directory entries are checked everywhere.
func TestTheOldContentsSurviveUntilTheRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// A reader that opened the file just before the write started. Windows
	// refuses the rename underneath it, so the handle is only opened where
	// holding one across a rename is a thing that happens.
	var reader *os.File
	if runtime.GOOS != "windows" {
		reader, err = os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
	}

	if err := Replace(path, []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if reader != nil {
		held, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		if string(held) != "before\n" {
			t.Errorf("the open reader saw %q, want the old bytes: the file was written in place rather than renamed over", held)
		}
		after, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if os.SameFile(before, after) {
			t.Error("the path still names the file it named before, so nothing was renamed over it")
		}
	}
	if b, _ := os.ReadFile(path); string(b) != "after\n" {
		t.Errorf("contents = %q, want the new bytes", b)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("the directory holds %d entries, want just the file", len(entries))
	}
}
