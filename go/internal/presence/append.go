package presence

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"gdoc/internal/config"
)

// The seams. Each is a variable rather than a call so one scripted run can
// write the fixture byte for byte: TestTheFixtureIsWhatTheWriterWrites.
//
// since is separate from now on purpose. now is the wall clock an event's at
// is written from, and since is the monotonic source a duration comes from;
// a package with one clock would write a negative duration the first time
// somebody's machine moved its own.
var (
	now     = time.Now
	since   = time.Since
	pidOf   = os.Getpid
	runID   = newRunID()
	writeTo = func(f *os.File, b []byte) (int, error) { return f.Write(b) }
)

// said holds the one line a process says on stderr when the record cannot be
// written, and swept the one directory sweep a process does. Both are
// pointers so a test can put a fresh one in.
var (
	said  = new(sync.Once)
	swept = new(sync.Once)
)

// monthFile is the exact name the record uses. The sweep removes nothing that
// does not match it, so a note somebody left in the directory stays:
// TestOnlyTheCurrentAndPreviousMonthSurviveAndOnlyExactNamesAreRemoved.
var monthFile = regexp.MustCompile(`^\d{4}-\d{2}\.jsonl$`)

// newRunID is eight random bytes as hex, taken once per process. It pairs the
// events of one run where a pid cannot, because the operating system hands
// pids out again.
func newRunID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// appendEvent stamps an event with what every event carries and appends it.
// A failure changes nothing about the command that wrote it: no byte of the
// envelope, no exit code, and one line on stderr.
func appendEvent(e Event, errOut io.Writer) {
	e.V = schema
	when := now().UTC()
	e.At = when.Format(time.RFC3339)
	e.PID = pidOf()
	e.Run = runID
	if err := appendLine(when, e); err != nil {
		say(errOut, err)
	}
}

// appendLine is the whole of the channel: one open, one write of one line
// checked for its length, one close.
//
// The path is refused before the open when something that is not a regular
// file is there, so a symlink somebody planted never carries a record
// somewhere else, and O_NOFOLLOW closes the gap between the two:
// TestASymlinkOrANonRegularFileAtThePathIsRefusedAndNothingIsWritten.
func appendLine(when time.Time, e Event) error {
	dir, err := config.PresenceDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, fileName(when))
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file, so nothing was written there", path)
	}

	var line bytes.Buffer
	enc := json.NewEncoder(&line)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(e); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE|nofollow, 0o600)
	if err != nil {
		return err
	}
	n, err := writeTo(f, line.Bytes())
	if err == nil && n != line.Len() {
		err = fmt.Errorf("%s took %d bytes of the %d one event is, so the line on disk is half of one", path, n, line.Len())
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	swept.Do(func() { sweep(dir, when) })
	return nil
}

// fileName is the month's file. The month is the event's UTC month, so a
// machine that travels renames nothing and two machines agree on which file a
// moment belongs in.
func fileName(when time.Time) string {
	return fmt.Sprintf("%04d-%02d.jsonl", when.Year(), int(when.Month()))
}

// sweep removes the months before the previous one, once per process, and
// only files whose name is exactly a month's.
//
// A removal that fails is left alone: an old file nothing can delete changes
// nothing about the record being written, and the one line on stderr belongs
// to the write that failed rather than to the housekeeping after it. A person
// deletes it, as they delete every old file when the flag is off.
func sweep(dir string, when time.Time) {
	keep := map[string]bool{
		fileName(when):                true,
		fileName(previousMonth(when)): true,
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if keep[name] || !monthFile.MatchString(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}
}

// previousMonth is the last day of the month before this one. It walks back
// by this month's own day count rather than by one month, because a month
// subtracted from the 31st is the 3rd of the wrong month.
func previousMonth(when time.Time) time.Time {
	return when.AddDate(0, 0, -when.Day())
}

// say is the one line a process says when the record could not be written.
// Once, because a chat session that cannot write would otherwise fill the log
// with the same sentence, and the first one already said everything.
func say(errOut io.Writer, err error) {
	if errOut == nil {
		return
	}
	said.Do(func() {
		fmt.Fprintf(errOut, "gdoc: the presence record was not written: %v\n", err)
	})
}
