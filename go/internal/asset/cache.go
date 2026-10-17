// The cache and what makes it usable. One file of bytes under assets/<id> and
// one stamp beside it, both this machine's own bookkeeping.

package asset

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gdoc/internal/atomicfile"
	"gdoc/internal/config"
)

// cacheMode is what the cached bytes and the stamp are left as: readable by
// their owner alone, like the token beside them. They are not a credential,
// but they are a file somebody's sign-in fetched, and the config directory is
// one person's.
const cacheMode = 0o600

// cached is what the cache holds for one id, read once at the top of a Fetch.
//
// ok and fresh are two different questions. ok is whether these bytes may
// reach a document at all, which is whether they decode as the file the caller
// states. fresh is whether they may be used without asking Drive, which also
// needs a stamp inside Interval that was written about these very bytes.
type cached struct {
	raw   []byte
	stamp Stamp
	ok    bool
	fresh bool
}

// readCache reads the bytes and the stamp and decides both questions. Every
// failure is a cache that is not fresh rather than an error: there is always
// an answer available, which is to ask Drive.
func readCache(id string, want Want, now time.Time) cached {
	path, err := config.AssetPath(id)
	if err != nil {
		return cached{}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return cached{}
	}
	if err := hold(raw, want); err != nil {
		// A cache that is not the file the caller states is no cache. Keeping
		// it would mean a document built from bytes nothing understood.
		return cached{}
	}
	c := cached{raw: raw, ok: true}

	s, stale, _ := readStamp(id, now)
	// The stamp has to be about these bytes. A stamp beside bytes it was not
	// written for says nothing about when they were fetched, so it is read as
	// no stamp: the bytes stay usable and the question is asked again.
	if s.FetchedAt.IsZero() || s.SHA256 != sum(raw) {
		return c
	}
	// A stamp that is merely old is kept, the way internal/lastcheck keeps
	// what an old check heard: a fetch that then fails reports the age of the
	// bytes it fell back on, and only a stamp about these bytes can say it.
	c.stamp, c.fresh = s, !stale
	return c
}

// answer is the cache as a Bytes. The age comes from the stamp when there is
// one about these bytes, and is zero when there is not, because this room
// states facts and does not guess at one.
func (c cached) answer(now time.Time, reason string) Bytes {
	b := Bytes{Raw: c.raw, Source: FromCache, Reason: reason}
	if !c.stamp.FetchedAt.IsZero() {
		b.FetchedAt = c.stamp.FetchedAt
		b.Age = now.Sub(c.stamp.FetchedAt)
	}
	return b
}

// orRefuse is the cache when there is one and a refusal when there is not.
// reason is what happened instead of a fetch, and it is carried either way:
// in the envelope when the run goes on, and in the error when it stops.
func (c cached) orRefuse(id string, now time.Time, reason string) (Bytes, error) {
	if c.ok {
		return c.answer(now, reason), nil
	}
	return Bytes{}, fmt.Errorf("file %s could not be read from Drive and nothing usable is cached for it: %s", id, reason)
}

// writeCache replaces the bytes when they changed and moves the stamp either
// way. The stamp records that the question was asked; the bytes are only
// written when the answer is new, so a run a day does not rewrite five
// kilobytes for nothing.
//
// The directory is made here: the first run on a machine is exactly the run
// with something to cache.
func writeCache(id string, raw []byte, now time.Time) error {
	path, err := config.AssetPath(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err != nil || !bytes.Equal(old, raw) {
		if err := atomicfile.Replace(path, raw, cacheMode); err != nil {
			return err
		}
	}
	return writeStamp(id, Stamp{FetchedAt: now, SHA256: sum(raw)})
}

// sum is the SHA-256 of b in hex, which is what the stamp holds and what ties
// a stamp to the bytes beside it.
func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
