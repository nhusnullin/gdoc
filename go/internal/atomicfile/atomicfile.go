// Package atomicfile replaces a file's contents without ever leaving it
// truncated. One room, because both files gdoc writes are files a failed write
// must not destroy: the OAuth token, which a truncated copy signs somebody out
// of, and the markdown note, which is the source the whole tool reads from and
// which gdoc is not the only reader of.
//
// Two copies of this dance existed before, in internal/auth and in cmd/gdoc,
// and the publish milestone wanted a third.
package atomicfile

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Replace writes b to path through a temp file in the same directory and a
// rename. The temp file is removed on every failure, so a failed write leaves
// nothing beside the file it did not replace.
//
// Same-directory rename. Not guaranteed atomic on Windows; documented in
// docs/v2/SPEC.md and covered by the M9 Windows smoke test.
func Replace(path string, b []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		return fail(err)
	}
	if err := tmp.Chmod(mode); err != nil {
		return fail(err)
	}
	// Sync before the rename. Without it a power cut can leave the renamed file
	// present and empty, which is the exact failure this dance exists to
	// prevent.
	if err := tmp.Sync(); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// ModeOf is the mode of the file about to be replaced, or fallback when there
// is no file there yet. A note somebody made group readable stays that way.
func ModeOf(path string, fallback fs.FileMode) fs.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm()
	}
	return fallback
}
