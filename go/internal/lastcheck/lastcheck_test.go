package lastcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The file is gdoc's own, and something else reads it: a person looking at
// ~/.config/gdoc-agent to see what the tool has been doing. So the bytes are
// pinned here as a literal, one line, in the shape docs/plans names.
func TestAWrittenStampReadsBackByteForByte(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "update-check.json")
	stamp := Stamp{
		CheckedAt:     time.Date(2026, 9, 18, 7, 12, 3, 0, time.UTC),
		LatestStable:  "v2.3.0",
		LatestNightly: "v2.3.4",
	}

	// Act
	if err := Write(path, stamp); err != nil {
		t.Fatalf("Write: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading it back: %v", err)
	}

	// Assert
	const want = `{"checked_at":"2026-09-18T07:12:03Z","latest_stable":"v2.3.0","latest_nightly":"v2.3.4"}`
	if string(b) != want {
		t.Errorf("the file holds\n%s\nwant\n%s", b, want)
	}

	got, stale, why := Read(path, stamp.CheckedAt.Add(time.Hour))
	if stale {
		t.Errorf("the stamp it just wrote is stale: %s", why)
	}
	if got != stamp {
		t.Errorf("Read() = %+v, want %+v", got, stamp)
	}
}

// A failed check keeps the versions it heard last and adds the cause, so the
// error field is part of the file's shape and not a second file.
func TestAFailedCheckKeepsWhatItHeardAndNamesTheCause(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update-check.json")
	stamp := Stamp{
		CheckedAt:     time.Date(2026, 9, 19, 7, 0, 11, 0, time.UTC),
		LatestStable:  "v2.3.0",
		LatestNightly: "v2.3.4",
		Error:         "the request failed: context deadline exceeded",
	}
	if err := Write(path, stamp); err != nil {
		t.Fatalf("Write: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading it back: %v", err)
	}
	const want = `{"checked_at":"2026-09-19T07:00:11Z","latest_stable":"v2.3.0","latest_nightly":"v2.3.4","error":"the request failed: context deadline exceeded"}`
	if string(b) != want {
		t.Errorf("the file holds\n%s\nwant\n%s", b, want)
	}
	got, _, _ := Read(path, stamp.CheckedAt)
	if got != stamp {
		t.Errorf("Read() = %+v, want %+v", got, stamp)
	}
}

// Stale has four causes and each one says what was wrong, because the answer
// reaches a person through a warning and "stale" alone names nothing.
func TestMissingCorruptAndOldAreEachStaleAndNamed(t *testing.T) {
	now := time.Date(2026, 9, 18, 7, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name  string
		file  string // the file's contents, or absent for no file at all
		write bool
		stale bool
		says  string
		keeps Stamp
	}{
		{
			name:  "no file at all",
			stale: true,
			says:  "no record",
		},
		{
			name:  "a file nothing can decode",
			file:  `{`,
			write: true,
			stale: true,
			says:  "is not the JSON object",
		},
		{
			name:  "a key gdoc does not know",
			file:  `{"checked_at":"2026-09-18T06:00:00Z","latest_beta":"v2.3.0"}`,
			write: true,
			stale: true,
			says:  "latest_beta",
		},
		{
			name:  "a second object behind the first",
			file:  `{"checked_at":"2026-09-18T06:00:00Z"}{"checked_at":"2026-09-18T06:30:00Z"}`,
			write: true,
			stale: true,
			says:  "more than one JSON object",
		},
		{
			name:  "no checked_at, so nothing says when gdoc asked",
			file:  `{"latest_stable":"v2.3.0"}`,
			write: true,
			stale: true,
			says:  "carries no checked_at",
		},
		{
			name:  "twenty five hours old",
			file:  `{"checked_at":"2026-09-17T06:00:00Z","latest_stable":"v2.3.0"}`,
			write: true,
			stale: true,
			says:  "older than",
			keeps: Stamp{
				CheckedAt:    time.Date(2026, 9, 17, 6, 0, 0, 0, time.UTC),
				LatestStable: "v2.3.0",
			},
		},
		{
			name:  "dated after the clock, so the clock or the file moved",
			file:  `{"checked_at":"2026-09-19T07:00:00Z","latest_stable":"v2.3.0"}`,
			write: true,
			stale: true,
			says:  "ahead of the clock",
			keeps: Stamp{
				CheckedAt:    time.Date(2026, 9, 19, 7, 0, 0, 0, time.UTC),
				LatestStable: "v2.3.0",
			},
		},
		{
			name:  "twenty three hours old",
			file:  `{"checked_at":"2026-09-17T08:00:00Z","latest_stable":"v2.3.0"}`,
			write: true,
			stale: false,
			keeps: Stamp{
				CheckedAt:    time.Date(2026, 9, 17, 8, 0, 0, 0, time.UTC),
				LatestStable: "v2.3.0",
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "update-check.json")
			if c.write {
				if err := os.WriteFile(path, []byte(c.file), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			got, stale, why := Read(path, now)

			if stale != c.stale {
				t.Fatalf("Read() stale = %v, want %v (why: %q)", stale, c.stale, why)
			}
			if !c.stale {
				if why != "" {
					t.Errorf("a fresh stamp says %q, want nothing", why)
				}
			} else if !strings.Contains(why, c.says) {
				t.Errorf("Read() why = %q, want it to name %q", why, c.says)
			}
			if got != c.keeps {
				t.Errorf("Read() = %+v, want %+v", got, c.keeps)
			}
		})
	}
}

// The literal, never the constant. A test that reads the constant follows it
// wherever somebody moves it.
func TestTheIntervalIsADay(t *testing.T) {
	if Interval != 24*time.Hour {
		t.Errorf("Interval = %v, want 24h", Interval)
	}
}

// The write goes through internal/atomicfile, and a temp file left in the
// config directory is litter beside somebody's token.
func TestAWriteLeavesNoTempFileBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update-check.json")
	if err := Write(path, Stamp{CheckedAt: time.Now(), LatestStable: "v2.3.0"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := Write(path, Stamp{CheckedAt: time.Now(), LatestStable: "v2.4.0"}); err != nil {
		t.Fatalf("the second Write: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "update-check.json" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("the directory holds %v, want update-check.json alone", names)
	}
}

// The stamp is not a credential, but it sits in the same directory as one and
// it is nobody else's business how often this machine asks GitHub.
func TestTheStampIsWrittenReadableByItsOwnerAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update-check.json")
	if err := Write(path, Stamp{CheckedAt: time.Now()}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode = %v, want 0600", got)
	}
}

// A directory that is not there yet is the first run on a machine, and the
// first run is exactly the run that has something to say.
func TestWriteMakesTheDirectoryItWritesInto(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gdoc-agent", "update-check.json")
	if err := Write(path, Stamp{CheckedAt: time.Now()}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("after Write: %v", err)
	}
}

// A write that cannot happen is an error the caller reports, never a silence.
func TestAWriteThatCannotHappenSaysSo(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Skipf("this filesystem does not do read-only directories: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	if os.Geteuid() == 0 {
		t.Skip("root writes into a read-only directory anyway")
	}

	err := Write(filepath.Join(dir, "update-check.json"), Stamp{CheckedAt: time.Now()})
	if err == nil {
		t.Fatal("Write into a read-only directory = nil, want the failure")
	}
}
