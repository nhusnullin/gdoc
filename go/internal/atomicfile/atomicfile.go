// Package atomicfile replaces a file's contents without ever leaving it
// truncated. One room, because both files gdoc writes are files a failed write
// must not destroy: the OAuth token, which a truncated copy signs somebody out
// of, and the markdown note, which is the source the whole tool reads from and
// which gdoc is not the only reader of.
//
// Two copies of this dance existed before, in internal/auth and in cmd/gdoc,
// and the publish milestone wanted a third.
//
// # Two verbs, because replacing and creating are different promises
//
// Replace ends in os.Rename, which replaces whatever is at the path. That is
// the promise the token and the note want: the file is already there and its
// new contents go in without a moment where it is empty.
//
// Create ends in os.Link, which fails when the path is already there. That is
// the promise export wants: it takes the next free name, so a path that
// appeared between the moment it looked and the moment it writes must be
// refused rather than replaced. A rename cannot say no, so a caller checking
// first and renaming second has a gap nothing closes. TestCreateWritesAFileThatWasNotThere
// and TestCreateRefusesAPathThatExists are the pins, and internal/export's
// TestNothingCanReplaceAFile holds the other half: nothing that writes a new
// file there reaches for Replace.
package atomicfile

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// NewMode is the mode a file Create makes gets. There is no file there to
// take a mode from, so this package names one rather than leaving it to
// whatever umask the temp file was born under.
const NewMode fs.FileMode = 0o644

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

// Create writes b to a path that must not exist. The temp file goes in the
// same directory and is linked to path, so the check and the write are one
// step: a path that appeared since the caller looked is refused, and the error
// reads as fs.ErrExist.
//
// The temp file is removed whether the link worked or not, so a run that
// refuses leaves nothing beside the file it did not touch.
func Create(path string, b []byte) error {
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
	if err := tmp.Chmod(NewMode); err != nil {
		return fail(err)
	}
	// Sync before the link, for the reason Replace syncs before the rename: a
	// power cut must not leave the new name pointing at an empty file.
	if err := tmp.Sync(); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	err = os.Link(name, path)
	os.Remove(name)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s is already there, and gdoc never replaces a file: %w", path, fs.ErrExist)
		}
		return err
	}
	return nil
}
