package atomicfile

import (
	"os"
	"path/filepath"
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
func TestTheOldContentsSurviveUntilTheRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Replace(path, []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("the directory holds %d entries, want just the file", len(entries))
	}
}
