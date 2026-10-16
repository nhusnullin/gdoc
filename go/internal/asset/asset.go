package asset

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"time"

	"gdoc/internal/config"
)

// MaxBytes is the ceiling on one asset. A body with no bound is a memory
// limit somebody else sets, and what this room fetches is a mark on a page:
// the firm's own logo is five kilobytes. The ceiling is well above anything
// real on purpose, because it is here to stop an answer nothing bounds rather
// than to trim a real one. TestTheReadIsBounded is the pin.
const MaxBytes = 1 << 20

// Interval is how long a stamp stands before this room asks Drive again. A
// day, because a logo changes less often than a release does, and because a
// build that asks once a run would make every document cost a request.
// TestTheIntervalIsADay states the literal.
const Interval = 24 * time.Hour

// FromDrive and FromCache are what Source says: where the bytes in hand came
// from. They are facts and not verdicts. A run inside the day reads the
// cache and nothing went wrong; a run that asked and was refused reads the
// cache too, and Reason is the difference.
const (
	FromDrive = "drive"
	FromCache = "cache"
)

// Drive is the whole of what this room asks of the wire: the bytes of one
// file, by id, within a ceiling. internal/gapi's authenticated session is one,
// and a test's own stand-in is another.
//
// It is an interface because the request and the bearer belong to
// internal/gapi, which is the one room that builds a request, and the client
// belongs to internal/guard, which is the one room that builds a wire. This
// room is handed its reach and builds neither.
//
// A nil Drive is a run with no sign-in: nothing to ask with. That is not an
// error here, because a cache written yesterday is still an answer.
type Drive interface {
	FileBytes(ctx context.Context, fileID string, limit int64) ([]byte, error)
}

// Want is what the caller's own file says about the asset: the mime it is and
// how many pixels across and down. The caller states it, this room holds the
// bytes to it, and neither of them knows what the picture is for.
type Want struct {
	Mime   string
	Width  int
	Height int
}

// check holds the statement itself to what this room can check. A caller
// naming a mime nothing here decodes would otherwise have its file accepted
// unread, which is the one thing a size check exists to prevent.
func (w Want) check() error {
	if w.Mime != "image/png" {
		return fmt.Errorf("the file is stated as %q, and image/png is the only mime this room checks", w.Mime)
	}
	if w.Width <= 0 || w.Height <= 0 {
		return fmt.Errorf("the file is stated as %dx%d px, and both have to be real sizes", w.Width, w.Height)
	}
	return nil
}

// Bytes is one asset in hand, and the facts about how it got there. Nothing in
// it is a verdict: whether a caller mentions the age or the reason is the
// caller's decision, and the envelope prints what is here.
type Bytes struct {
	// Raw is the file. It has been held to the Want the caller stated, so a
	// Bytes carrying Raw is a file the caller understands.
	Raw []byte
	// Source is FromDrive or FromCache.
	Source string
	// FetchedAt is when these bytes were read from Drive: now for a fetch,
	// and the stamp's own time for the cache.
	FetchedAt time.Time
	// Age is how long ago that was, and zero for a fetch.
	Age time.Duration
	// Reason is why the cache was used instead of Drive, and empty when
	// nothing went wrong: a fresh stamp, or a read that worked.
	Reason string
	// CacheError is why the cache could not be written, when it could not.
	// The bytes still came back: a cache nothing can write is a slower run,
	// not a failed one.
	CacheError string
}

// Fetch returns the bytes of one Drive file, held to what the caller says of
// it, through the cache beside the token.
//
// The order is the whole of the rule. A stamp written within Interval, with
// the bytes it was written beside still there and still the file the caller
// states, is the answer and the wire stays shut. Otherwise Drive is asked
// once. A read that worked is checked, cached and returned. A read that was
// refused, timed out or never left falls back to the cache and says why. No
// reach at all is the same case, with the sign-in as the reason. Nothing
// usable in the cache and no answer from Drive is a refusal: this room never
// returns bytes it did not check, and never returns none.
//
// A file in Drive that is not what the caller states refuses the run even when
// the cache is good, because the file is wrong and the refusal is the only
// message that reaches whoever can fix it.
func Fetch(ctx context.Context, client Drive, id string, want Want) (Bytes, error) {
	if err := want.check(); err != nil {
		return Bytes{}, err
	}
	// The id becomes a path, so an id that is not one is refused before
	// anything is read or asked.
	if _, err := config.AssetPath(id); err != nil {
		return Bytes{}, err
	}

	now := time.Now()
	c := readCache(id, want, now)
	if c.fresh {
		return c.answer(now, ""), nil
	}

	if client == nil {
		return c.orRefuse(id, now, "this machine holds no sign-in, so Drive was not asked; run: gdoc auth login")
	}
	raw, err := client.FileBytes(ctx, id, MaxBytes)
	if err != nil {
		return c.orRefuse(id, now, err.Error())
	}
	if err := hold(raw, want); err != nil {
		return Bytes{}, fmt.Errorf("file %s in Drive is not the file it is stated to be: %w", id, err)
	}

	got := Bytes{Raw: raw, Source: FromDrive, FetchedAt: now}
	if err := writeCache(id, raw, now); err != nil {
		got.CacheError = err.Error()
	}
	return got, nil
}

// hold is the whole check this room makes on bytes it did not write: a PNG
// that decodes, of the size the caller states. It runs on what Drive answered
// and on what the cache holds, because a file gdoc half understands must not
// reach a document by either route.
func hold(raw []byte, want Want) error {
	cfg, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("it is not a PNG: %w", err)
	}
	if cfg.Width != want.Width || cfg.Height != want.Height {
		return fmt.Errorf("it is stated as %dx%d px and the image is %dx%d", want.Width, want.Height, cfg.Width, cfg.Height)
	}
	return nil
}
