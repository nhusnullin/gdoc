package update

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"gdoc/internal/atomicfile"
)

// binaryMode is what an executable is left as. The zip carries a mode too, but
// a binary that arrives without the bit set is a binary nobody can run, so the
// mode here is the one that matters.
const binaryMode = 0o755

// Result is what an applied update, or a rollback, left on disk.
type Result struct {
	// Sum is the SHA-256 of the file at its final path, in hex, hashed after
	// the rename rather than before it. It is the fact `verified` is read
	// from: what is on disk now, not what was promised.
	Sum string
	// Previous is where the binary that was there went, and empty when there
	// was nothing there to keep.
	Previous string
}

// PreviousPath is where Apply puts the binary it replaced, and where Rollback
// looks for it.
func PreviousPath(path string) string { return path + ".previous" }

// Verify checks the downloaded zip against its line of the release's checksum
// file. The file is what sha256sum writes: the hash, two spaces, the name of
// one asset, one line per zip in the release.
//
// A download cut short, a byte changed on the way, and an asset the file says
// nothing about are all refused here, before anything on disk is moved.
func Verify(archive, sums []byte, asset string) error {
	want, err := sumLine(sums, asset)
	if err != nil {
		return err
	}
	got := sum(archive)
	if got != want {
		return fmt.Errorf("%s hashes to %s and the release said %s, so it is not the file the release published", asset, got, want)
	}
	return nil
}

// Apply verifies the zip, takes the binary out of it and puts it at path,
// keeping whatever was at path as PreviousPath(path).
//
// The order is the whole of the safety. Nothing on disk moves until the zip
// has matched its checksum, so a failed download leaves the old binary exactly
// where it was. Then the new binary is written beside it as <path>.new, the
// old one is renamed to <path>.previous, and the new one is renamed into
// place: a running executable cannot be written through, and rename is what
// works on both platforms. Every failure after the first rename puts the old
// binary back.
//
// Windows: the same three renames, and os.Rename replaces the file it lands
// on there as it does here. It is unmeasured until the checklist in
// docs/backlog/windows-rollout-checklist.md runs, which is the first time
// gdoc replaces itself on Windows.
//
// TestTheReplaceSequenceLeavesTheNewBinaryAndKeepsTheOld,
// TestASecondUpdateOverwritesThePrevious, TestAWrongChecksumReplacesNothing
// and TestAZipWithNoBinaryInItIsRefusedByName.
func Apply(archive, sums []byte, asset, binary, path string) (Result, error) {
	if err := Verify(archive, sums, asset); err != nil {
		return Result{}, err
	}
	content, err := fileIn(archive, asset, binary)
	if err != nil {
		return Result{}, err
	}
	staged := path + ".new"
	if err := atomicfile.Replace(staged, content, binaryMode); err != nil {
		return Result{}, fmt.Errorf("the new binary could not be written beside the old one: %w", err)
	}
	previous := PreviousPath(path)
	kept := false
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, previous); err != nil {
			os.Remove(staged)
			return Result{}, fmt.Errorf("the binary at %s could not be moved aside: %w", path, err)
		}
		kept = true
	}
	if err := os.Rename(staged, path); err != nil {
		if kept {
			os.Rename(previous, path)
		}
		os.Remove(staged)
		return Result{}, fmt.Errorf("the new binary could not be moved to %s: %w", path, err)
	}
	return landed(path, previous, kept, sum(content))
}

// landed reads back what is now at path and says whether it is what the zip
// held. It is the whole of the tail after the last rename, in one place, so
// that "every failure after the first rename puts the old binary back" has one
// place to hold rather than two: an unreadable file and a file that hashes
// wrong both undo the swap before they raise.
//
// TestAWrongChecksumReplacesNothing, TestAnUnreadableReadBackPutsTheOldBinaryBack
// and TestAFailedReadBackWithNothingToRestoreLeavesNoBinary.
func landed(path, previous string, kept bool, want string) (Result, error) {
	// With nothing to put back, the file at path is the one that just failed
	// its read-back, and leaving it there leaves an unverified binary for the
	// next run to execute. Not knowing never resolves to overwrite, and it
	// never resolves to run something either.
	restore := func() {
		if kept {
			os.Rename(previous, path)
			return
		}
		os.Remove(path)
	}
	got, err := sumOfFile(path)
	if err != nil {
		restore()
		return Result{}, err
	}
	if got != want {
		restore()
		return Result{}, fmt.Errorf("the file at %s hashes to %s and the zip held %s, so what landed is not what was verified", path, got, want)
	}
	out := Result{Sum: got}
	if kept {
		out.Previous = previous
	}
	return out, nil
}

// Rollback swaps the two binaries: what is at path becomes the previous one,
// and the previous one becomes the binary a person runs. It is a swap rather
// than a move so that a rollback taken by mistake is one more rollback away
// from where it started.
//
// TestRollbackSwapsTheTwoBinariesBack, TestARollbackRunTwiceIsWhereItStarted
// and TestRollbackRefusesWhenThereIsNoPrevious.
func Rollback(path string) (Result, error) {
	previous := PreviousPath(path)
	if _, err := os.Stat(previous); err != nil {
		return Result{}, fmt.Errorf("there is no %s, so there is no earlier binary to go back to", previous)
	}
	held := path + ".new"
	swap := false
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, held); err != nil {
			return Result{}, fmt.Errorf("the binary at %s could not be moved aside: %w", path, err)
		}
		swap = true
	}
	if err := os.Rename(previous, path); err != nil {
		if swap {
			os.Rename(held, path)
		}
		return Result{}, fmt.Errorf("the earlier binary could not be moved to %s: %w", path, err)
	}
	out := Result{}
	if swap {
		if err := keepRolledBack(held, previous); err != nil {
			return Result{}, err
		}
		out.Previous = previous
	}
	// The swap is done by here, and undoing it would take back the rollback a
	// person asked for. So this failure raises and says where the binaries
	// ended up, rather than moving a third time.
	got, err := sumOfFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("%w; the swap had already happened, so %s is what runs now", err, path)
	}
	out.Sum = got
	return out, nil
}

// keepRolledBack puts the binary the rollback moved aside where a second
// rollback would look for it.
//
// A failure here leaves that binary at held and names it. The swap has already
// happened by this point, so the file at held is the only copy of the release
// a person just left, and removing it would take away the only way back to it
// short of another download.
//
// TestAFailedKeepLeavesTheRolledBackBinaryOnDisk.
func keepRolledBack(held, previous string) error {
	if err := os.Rename(held, previous); err != nil {
		return fmt.Errorf("the binary that was rolled back could not be kept at %s, so it is still at %s: %w", previous, held, err)
	}
	return nil
}

// fileIn reads one file out of the zip by name. The release workflow packs the
// binary at the top of the archive, so the name is exact: a zip laid out some
// other way is refused by name rather than searched, because the one thing
// this function must never do is install a file that is not the binary.
func fileIn(archive []byte, asset, name string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("%s is not a zip that opens: %w", asset, err)
	}
	for _, f := range r.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%s holds %s and it does not open: %w", asset, name, err)
		}
		defer rc.Close()
		b, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("%s holds %s and it does not read: %w", asset, name, err)
		}
		return b, nil
	}
	return nil, fmt.Errorf("%s holds no %s at its top, so there is no binary in it to install", asset, name)
}

// sumLine finds the asset's line of the checksum file.
func sumLine(sums []byte, asset string) (string, error) {
	for _, line := range strings.Split(string(sums), "\n") {
		hash, name, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		// sha256sum writes two spaces, or a space and a star for a binary
		// read. Both leave the name after the cut with one character of that
		// separator in front of it.
		if strings.TrimLeft(name, " *") == asset {
			return hash, nil
		}
	}
	return "", fmt.Errorf("the release's checksum file has no line for %s, so nothing about it could be verified", asset)
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func sumOfFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("the binary at %s could not be read back: %w", path, err)
	}
	return sum(b), nil
}
