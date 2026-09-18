package lastcheck

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"gdoc/internal/atomicfile"
)

// Interval is how long a stamp stands before gdoc asks again. A day, because
// releases are cut by hand and a colleague who hears about one the next
// morning has lost nothing, while a machine that cannot reach GitHub pays the
// ceiling once a day instead of once a run.
const Interval = 24 * time.Hour

// Stamp is what gdoc heard the last time it asked GitHub what is published,
// and when it asked. It holds facts and no judgement: whether a version is
// newer than this binary is internal/update's arithmetic, done fresh each
// time, and never written down here.
//
// A check that failed is written too, with the cause in Error and the versions
// it heard before left where they were. That is what bounds the cost of a
// network that refuses GitHub.
type Stamp struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestStable  string    `json:"latest_stable,omitempty"`
	LatestNightly string    `json:"latest_nightly,omitempty"`
	Error         string    `json:"error,omitempty"`
}

// Read returns the stamp at path, whether it is stale, and why it is stale.
// Stale is missing, unreadable, malformed, dated after the clock, or older
// than Interval against the clock the caller hands in. The reason is empty
// when the stamp is fresh.
//
// The read is strict: a key gdoc does not know and a second object behind the
// first are each refused by name, as every input gdoc reads is. The file is
// gdoc's own, so an unknown key is this binary reading a file a newer one
// wrote, and guessing at half of it is worse than asking GitHub again.
//
// A stamp that decoded and is merely old comes back with its versions, because
// a check that then fails keeps them. A stamp nothing could decode comes back
// empty: there is nothing in it to keep.
func Read(path string, now time.Time) (Stamp, bool, string) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Stamp{}, true, fmt.Sprintf("there is no record of an earlier check in %s", path)
		}
		return Stamp{}, true, fmt.Sprintf("%s could not be read: %v", path, err)
	}

	var s Stamp
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return Stamp{}, true, fmt.Sprintf("%s is not the JSON object gdoc writes there: %v", path, err)
	}
	if dec.More() {
		return Stamp{}, true, fmt.Sprintf("%s carries more than one JSON object, and which check counts is not decided here", path)
	}
	if s.CheckedAt.IsZero() {
		return Stamp{}, true, fmt.Sprintf("%s carries no checked_at, so nothing in it says when gdoc last asked", path)
	}

	age := now.Sub(s.CheckedAt)
	if age < 0 {
		return s, true, fmt.Sprintf("the last check is dated %s, ahead of the clock", s.CheckedAt.Format(time.RFC3339))
	}
	if age >= Interval {
		return s, true, fmt.Sprintf("the last check was %s, older than %s", s.CheckedAt.Format(time.RFC3339), Interval)
	}
	return s, false, ""
}

// Write replaces the stamp at path through internal/atomicfile, at 0600 like
// the token beside it. The time is written in UTC to the second, so the file
// is one short line a person can read.
//
// The directory is made when it is not there: the first run on a machine is
// exactly the run with something to record.
func Write(path string, s Stamp) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	s.CheckedAt = s.CheckedAt.UTC().Truncate(time.Second)
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return atomicfile.Replace(path, b, 0o600)
}
