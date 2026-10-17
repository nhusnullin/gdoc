// The stamp: when this machine last asked Drive for one file, and what it got.
// internal/lastcheck's shape with its own two keys, and its rules.

package asset

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
	"gdoc/internal/config"
)

// Stamp is when the bytes beside it were fetched, and their hash. It holds
// facts and no judgement: whether it is still worth standing on is the
// arithmetic readStamp does over the clock it is handed, never written here.
//
// The hash is what ties the stamp to the file beside it. Without it a stamp
// left over from one asset would speak for whatever bytes landed at that path
// afterwards.
type Stamp struct {
	FetchedAt time.Time `json:"fetched_at"`
	SHA256    string    `json:"sha256"`
}

// readStamp returns the stamp for id, whether it is stale, and why. Stale is
// missing, unreadable, malformed, carrying no fetched_at, dated after the
// clock, or older than Interval.
//
// The read is strict, as every input gdoc reads is: a key gdoc does not know
// and a second object behind the first are each refused by name. The file is
// gdoc's own, so an unknown key is this binary reading what a newer one wrote,
// and standing on half of it is worse than asking Drive again. Every refusal
// is stale rather than fatal for the same reason: there is always an answer
// available.
func readStamp(id string, now time.Time) (Stamp, bool, string) {
	path, err := config.AssetStampPath(id)
	if err != nil {
		return Stamp{}, true, err.Error()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Stamp{}, true, fmt.Sprintf("there is no record of an earlier fetch in %s", path)
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
		return Stamp{}, true, fmt.Sprintf("%s carries more than one JSON object, and which fetch counts is not decided here", path)
	}
	if s.FetchedAt.IsZero() {
		return Stamp{}, true, fmt.Sprintf("%s carries no fetched_at, so nothing in it says when gdoc last asked", path)
	}

	age := now.Sub(s.FetchedAt)
	if age < 0 {
		return s, true, fmt.Sprintf("the last fetch is dated %s, ahead of the clock", s.FetchedAt.Format(time.RFC3339))
	}
	if age >= Interval {
		return s, true, fmt.Sprintf("the last fetch was %s, older than %s", s.FetchedAt.Format(time.RFC3339), Interval)
	}
	return s, false, ""
}

// writeStamp replaces the stamp for id through internal/atomicfile, at the
// mode the bytes beside it get. The time is written in UTC to the second, so
// the file is one short line a person looking in the config directory can
// read. TestTheStampIsOneLineAPersonCanRead holds those bytes.
func writeStamp(id string, s Stamp) error {
	path, err := config.AssetStampPath(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	s.FetchedAt = s.FetchedAt.UTC().Truncate(time.Second)
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return atomicfile.Replace(path, b, cacheMode)
}
