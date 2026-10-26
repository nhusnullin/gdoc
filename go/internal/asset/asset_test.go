// The cache, the stamp and what each failure does with them. Every size and
// every interval here is a literal: a test that reads the constant it checks
// follows it wherever somebody moves it.

package asset

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gdoc/internal/config"
)

// theID is a file id of the shape Drive mints. The room knows nothing about
// which file it is, so one id serves every test.
const theID = "1J3uh6poVAneP_V_Pc5FuNI4-Gahlab0s"

// housePNG is what the house says of the file in these tests: a small PNG,
// stated exactly.
var housePNG = Want{Mime: "image/png"}

// fake is the reach this room is handed. It counts its calls, because one
// request a day rather than one a run is the whole reason the stamp exists.
type fake struct {
	raw    []byte
	err    error
	calls  int
	ids    []string
	limits []int64
}

func (f *fake) FileBytes(ctx context.Context, fileID string, limit int64) ([]byte, error) {
	f.calls++
	f.ids = append(f.ids, fileID)
	f.limits = append(f.limits, limit)
	if f.err != nil {
		return nil, f.err
	}
	return f.raw, nil
}

// pngOf encodes a PNG of w by h pixels, filled flat.
func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// configDir points the config package at a directory of this test's own, so
// nothing here can touch the real one.
func configDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	return dir
}

// seedCache writes bytes into the cache and a stamp of the given age beside
// them. A zero age means no stamp at all.
func seedCache(t *testing.T, raw []byte, age time.Duration, withStamp bool) {
	t.Helper()
	p, err := config.AssetPath(theID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if raw != nil {
		if err := os.WriteFile(p, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if !withStamp {
		return
	}
	sum := sha256.Sum256(raw)
	writeStampFile(t, Stamp{FetchedAt: time.Now().Add(-age), SHA256: hex.EncodeToString(sum[:])})
}

func writeStampFile(t *testing.T, s Stamp) {
	t.Helper()
	sp, err := config.AssetStampPath(theID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(sp), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sp, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readCacheFile(t *testing.T) []byte {
	t.Helper()
	p, err := config.AssetPath(theID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	return raw
}

// The interval is a day, stated here as a literal. Releases are cut by hand
// and a logo changes less often than that, so a build pays one request a day
// at most.
func TestTheIntervalIsADay(t *testing.T) {
	if Interval != 24*time.Hour {
		t.Errorf("Interval = %v, want 24h", Interval)
	}
}

// Every way a stamp can fail to say "asked today" makes exactly one request,
// and the bytes come back as Drive's.
func TestAStaleStampFetchesOnce(t *testing.T) {
	fresh := pngOf(t, 8, 4)
	old := pngOf(t, 8, 4)
	old = append(old[:len(old):len(old)], 0) // a different file of the same shape

	cases := []struct {
		name  string
		seed  func(t *testing.T)
		cause string
	}{
		{"no cache and no stamp", func(t *testing.T) {}, "nothing cached"},
		{"a cache with no stamp", func(t *testing.T) { seedCache(t, old, 0, false) }, "no stamp"},
		{"a stamp nothing can decode", func(t *testing.T) {
			seedCache(t, old, time.Hour, true)
			sp, _ := config.AssetStampPath(theID)
			if err := os.WriteFile(sp, []byte("{not json"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "unreadable"},
		{"a stamp a newer gdoc wrote", func(t *testing.T) {
			seedCache(t, old, time.Hour, true)
			sp, _ := config.AssetStampPath(theID)
			// Written an hour ago and about these very bytes, so the
			// unknown key is the only thing left that can make it stale.
			line := fmt.Sprintf(`{"fetched_at":%q,"sha256":%q,"revision":"3"}`,
				time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), sum(old))
			if err := os.WriteFile(sp, []byte(line), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "an unknown key"},
		{"a stamp 25 hours old", func(t *testing.T) { seedCache(t, old, 25*time.Hour, true) }, "older than a day"},
		{"a stamp ahead of the clock", func(t *testing.T) { seedCache(t, old, -time.Hour, true) }, "ahead of the clock"},
		{"a fresh stamp with no bytes beside it", func(t *testing.T) { seedCache(t, nil, time.Hour, true) }, "no bytes"},
		{"a stamp whose sha256 is not the cache's", func(t *testing.T) {
			seedCache(t, old, time.Hour, true)
			writeStampFile(t, Stamp{FetchedAt: time.Now().Add(-time.Hour), SHA256: strings.Repeat("0", 64)})
		}, "another file's hash"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			configDir(t)
			c.seed(t)
			f := &fake{raw: fresh}
			got, err := Fetch(context.Background(), f, theID, housePNG)
			if err != nil {
				t.Fatalf("%s (%s): %v", c.name, c.cause, err)
			}
			if f.calls != 1 {
				t.Errorf("%d requests, want exactly one", f.calls)
			}
			if len(f.ids) != 1 || f.ids[0] != theID {
				t.Errorf("asked for %v, want the one id", f.ids)
			}
			if got.Source != FromDrive {
				t.Errorf("Source = %q, want %q", got.Source, FromDrive)
			}
			if !bytes.Equal(got.Raw, fresh) {
				t.Error("the bytes are not the ones Drive answered with")
			}
			if got.Reason != "" {
				t.Errorf("Reason = %q, want nothing: the read worked", got.Reason)
			}
			if time.Since(got.FetchedAt) > time.Minute {
				t.Errorf("FetchedAt = %v, want now", got.FetchedAt)
			}
			if !bytes.Equal(readCacheFile(t), fresh) {
				t.Error("the fetched bytes were not written to the cache")
			}
		})
	}
}

// A stamp written today means the question was already asked today, so the
// wire stays shut and the cache is the answer.
func TestAFreshStampMakesNoRequest(t *testing.T) {
	configDir(t)
	cached := pngOf(t, 8, 4)
	seedCache(t, cached, time.Hour, true)

	f := &fake{raw: pngOf(t, 8, 4)}
	got, err := Fetch(context.Background(), f, theID, housePNG)
	if err != nil {
		t.Fatal(err)
	}
	if f.calls != 0 {
		t.Errorf("%d requests, want none: the stamp is an hour old", f.calls)
	}
	if got.Source != FromCache {
		t.Errorf("Source = %q, want %q", got.Source, FromCache)
	}
	if !bytes.Equal(got.Raw, cached) {
		t.Error("the bytes are not the cached ones")
	}
	if got.Age < 59*time.Minute || got.Age > 61*time.Minute {
		t.Errorf("Age = %v, want about an hour", got.Age)
	}
	if got.Reason != "" {
		t.Errorf("Reason = %q, want nothing: nothing went wrong", got.Reason)
	}
}

// A run with no sign-in has no reach at all. With a cache it builds from it
// and says why, whatever the stamp says.
func TestNoTokenAndACacheUsesTheCache(t *testing.T) {
	configDir(t)
	cached := pngOf(t, 8, 4)
	seedCache(t, cached, 25*time.Hour, true)

	got, err := Fetch(context.Background(), nil, theID, housePNG)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != FromCache {
		t.Errorf("Source = %q, want %q", got.Source, FromCache)
	}
	if !bytes.Equal(got.Raw, cached) {
		t.Error("the bytes are not the cached ones")
	}
	if got.Age < 24*time.Hour {
		t.Errorf("Age = %v, want the stamp's own age", got.Age)
	}
	if !strings.Contains(got.Reason, "gdoc auth login") {
		t.Errorf("Reason = %q, want the sign-in named", got.Reason)
	}
}

// With no reach and nothing usable in the cache there is no answer, and the
// refusal names the one thing that fixes it. A cache that is not a PNG is no
// cache.
func TestNoTokenAndNoCacheRefusesNamingTheLogin(t *testing.T) {
	cases := []struct {
		name string
		seed func(t *testing.T)
	}{
		{"nothing cached", func(t *testing.T) {}},
		{"a cache that is not a PNG", func(t *testing.T) { seedCache(t, []byte("not a png"), time.Hour, true) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			configDir(t)
			c.seed(t)
			got, err := Fetch(context.Background(), nil, theID, housePNG)
			if err == nil {
				t.Fatalf("Fetch() = %+v, want a refusal", got)
			}
			if !strings.Contains(err.Error(), "gdoc auth login") {
				t.Errorf("the refusal is %q, want the sign-in named", err)
			}
			if got.Raw != nil {
				t.Error("a refusal carries no bytes")
			}
		})
	}
}

// Drive refusing, Drive not answering and the network not being there are the
// same case: the cache is the answer and the envelope carries what happened.
func TestARefusedReadUsesTheCacheAndNamesTheReason(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"403", fmt.Errorf("the request to %s was refused: 403 insufficient permissions", theID), "403"},
		{"404", fmt.Errorf("the request to %s was refused: 404 file not found", theID), "404"},
		{"a dial error", errors.New("dial tcp 142.250.0.1:443: connect: network is unreachable"), "network is unreachable"},
		{"a timeout", context.DeadlineExceeded, context.DeadlineExceeded.Error()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			configDir(t)
			cached := pngOf(t, 8, 4)
			seedCache(t, cached, 25*time.Hour, true)

			f := &fake{err: c.err}
			got, err := Fetch(context.Background(), f, theID, housePNG)
			if err != nil {
				t.Fatalf("want the cache rather than a refusal: %v", err)
			}
			if f.calls != 1 {
				t.Errorf("%d requests, want one: the stamp was stale", f.calls)
			}
			if got.Source != FromCache {
				t.Errorf("Source = %q, want %q", got.Source, FromCache)
			}
			if !bytes.Equal(got.Raw, cached) {
				t.Error("the bytes are not the cached ones")
			}
			if !strings.Contains(got.Reason, c.want) {
				t.Errorf("Reason = %q, want %q named in it", got.Reason, c.want)
			}
			if got.Age < 24*time.Hour {
				t.Errorf("Age = %v, want the stamp's own age", got.Age)
			}
			if !bytes.Equal(readCacheFile(t), cached) {
				t.Error("a failed read changed the cache")
			}
		})
	}
}

// The firm's file being wrong is the one failure a good cache does not cover.
// The refusal is the only message that reaches whoever can fix the file, and
// the cache is left exactly as it was.
func TestABadFileRefusesEvenWithAGoodCache(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
		want Want
		says string
	}{
		{"not a PNG", []byte("GIF89a and the rest"), housePNG, "not a PNG"},
		{"a mime this room does not carry", nil, Want{Mime: "image/jpeg"}, "image/png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			configDir(t)
			cached := pngOf(t, 8, 4)
			seedCache(t, cached, 25*time.Hour, true)

			raw := c.raw
			if raw == nil {
				raw = pngOf(t, 16, 8)
				if c.name == "a mime this room does not carry" {
					raw = pngOf(t, 8, 4)
				}
			}
			f := &fake{raw: raw}
			got, err := Fetch(context.Background(), f, theID, c.want)
			if err == nil {
				t.Fatalf("Fetch() = %+v, want a refusal", got)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal is %q, want %q named in it", err, c.says)
			}
			if got.Raw != nil {
				t.Error("a refusal carries no bytes")
			}
			if !bytes.Equal(readCacheFile(t), cached) {
				t.Error("a refused file reached the cache")
			}
		})
	}
}

func TestAPictureOfAnySizeIsCarried(t *testing.T) {
	for _, size := range []struct{ width, height int }{{16, 8}, {509, 185}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			configDir(t)
			raw := pngOf(t, size.width, size.height)
			got, err := Fetch(context.Background(), &fake{raw: raw}, theID, Want{Mime: "image/png"})
			if err != nil {
				t.Fatalf("Fetch: %v", err)
			}
			if !bytes.Equal(got.Raw, raw) {
				t.Error("the carried bytes differ from Drive's answer")
			}
			cached, err := Fetch(context.Background(), nil, theID, Want{Mime: "image/png"})
			if err != nil {
				t.Fatalf("cached Fetch: %v", err)
			}
			if cached.Source != FromCache || !bytes.Equal(cached.Raw, raw) {
				t.Errorf("cached Fetch = %+v, want the same bytes from cache", cached)
			}
		})
	}
}

// A file whose PNG header is right and whose picture is not all there is a
// file gdoc half understands, so it reaches a document by no route: not out of
// Drive, and not out of the cache. A header check alone would pass it, which
// is why hold decodes the whole image.
func TestAHalfPNGIsRefusedByBothRoutes(t *testing.T) {
	whole := pngOf(t, 8, 4)
	// The signature and the IHDR chunk, and none of the picture.
	half := whole[:40]

	t.Run("out of Drive", func(t *testing.T) {
		configDir(t)
		f := &fake{raw: half}

		got, err := Fetch(context.Background(), f, theID, housePNG)

		if err == nil {
			t.Fatalf("Fetch() = %+v, want a refusal", got)
		}
		if !strings.Contains(err.Error(), "not a PNG") {
			t.Errorf("the refusal is %q, want %q named in it", err, "not a PNG")
		}
		if got.Raw != nil {
			t.Error("a refusal carries no bytes")
		}
		if readCacheFile(t) != nil {
			t.Error("a half file reached the cache")
		}
	})

	t.Run("out of the cache", func(t *testing.T) {
		configDir(t)
		seedCache(t, half, time.Hour, true)
		f := &fake{raw: whole}

		got, err := Fetch(context.Background(), f, theID, housePNG)

		if err != nil {
			t.Fatalf("want the bytes rather than a refusal: %v", err)
		}
		if f.calls != 1 {
			t.Errorf("Drive was asked %d times, want once: a half cache is no cache", f.calls)
		}
		if got.Source != FromDrive {
			t.Errorf("Source = %q, want %q", got.Source, FromDrive)
		}
		if !bytes.Equal(got.Raw, whole) {
			t.Error("the bytes are not the ones Drive answered with")
		}
	})
}

// A PNG header states the size, and Go's decoder allocates the whole picture
// from that statement before it reads a single pixel. So the size is checked
// against the header and not against the decoded image: a file well inside
// MaxBytes that says it is thirty thousand pixels square is refused by name
// rather than allocating gigabytes to find out. The literal is the ceiling
// this test would hit without the header check.
func TestAHugeStatedSizeIsRefusedWithoutDecoding(t *testing.T) {
	configDir(t)
	// Sixty-eight bytes: the signature, an IHDR saying 30000x30000 RGBA, one
	// unusable IDAT and IEND. Decoding it whole costs about 3.4 GiB.
	raw := pngHeaderOf(t, 30000, 30000)
	if len(raw) > 1<<10 {
		t.Fatalf("the probe is %d bytes, and the point is that it is tiny", len(raw))
	}
	f := &fake{raw: raw}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	got, err := Fetch(context.Background(), f, theID, housePNG)
	runtime.ReadMemStats(&after)

	if err == nil {
		t.Fatalf("Fetch() = %+v, want a refusal", got)
	}
	if !strings.Contains(err.Error(), "30000x30000") {
		t.Errorf("the refusal is %q, want the stated size named in it", err)
	}
	if grew := after.TotalAlloc - before.TotalAlloc; grew > 64<<20 {
		t.Errorf("the refusal allocated %d MiB, want under 64 MiB: the header was not read first",
			grew/(1<<20))
	}
}

func TestASideOverTheCeilingIsRefusedBeforeDecoding(t *testing.T) {
	configDir(t)
	raw := pngHeaderOf(t, 4097, 1)
	got, err := Fetch(context.Background(), &fake{raw: raw}, theID, housePNG)
	if err == nil {
		t.Fatalf("Fetch() = %+v, want a refusal", got)
	}
	if !strings.Contains(err.Error(), "4097x1") {
		t.Errorf("the refusal is %q, want 4097x1 named in it", err)
	}
}

func TestTheSideCeilingIs4096(t *testing.T) {
	if MaxSide != 4096 {
		t.Errorf("MaxSide = %d, want 4096", MaxSide)
	}
}

// pngHeaderOf writes a PNG that states a size and carries no picture: the
// signature, an IHDR, an IDAT nothing can inflate and an IEND. It cannot go
// through png.Encode, which would write the pixels this test exists to avoid.
func pngHeaderOf(t *testing.T, w, h int) []byte {
	t.Helper()
	chunk := func(kind string, data []byte) []byte {
		var b bytes.Buffer
		_ = binary.Write(&b, binary.BigEndian, uint32(len(data)))
		b.WriteString(kind)
		b.Write(data)
		c := crc32.NewIEEE()
		c.Write([]byte(kind))
		c.Write(data)
		_ = binary.Write(&b, binary.BigEndian, c.Sum32())
		return b.Bytes()
	}
	var ihdr bytes.Buffer
	_ = binary.Write(&ihdr, binary.BigEndian, uint32(w))
	_ = binary.Write(&ihdr, binary.BigEndian, uint32(h))
	// Eight bits a channel, colour type 6 (RGBA), the three zeroes PNG fixes.
	ihdr.Write([]byte{8, 6, 0, 0, 0})

	var f bytes.Buffer
	f.WriteString("\x89PNG\r\n\x1a\n")
	f.Write(chunk("IHDR", ihdr.Bytes()))
	f.Write(chunk("IDAT", []byte{0x78, 0x9c, 0x01, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0x00, 0x01}))
	f.Write(chunk("IEND", nil))
	return f.Bytes()
}

// A cache nothing can write is a slower build, not a failed one: the bytes are
// in hand and the document is the job.
func TestACacheWriteFailureStillBuilds(t *testing.T) {
	dir := configDir(t)
	// A file where the assets directory would go, so nothing can be made there.
	if err := os.WriteFile(filepath.Join(dir, "assets"), []byte("in the way"), 0o600); err != nil {
		t.Fatal(err)
	}

	raw := pngOf(t, 8, 4)
	f := &fake{raw: raw}
	got, err := Fetch(context.Background(), f, theID, housePNG)
	if err != nil {
		t.Fatalf("want the bytes rather than a refusal: %v", err)
	}
	if !bytes.Equal(got.Raw, raw) {
		t.Error("the bytes are not the ones Drive answered with")
	}
	if got.Source != FromDrive {
		t.Errorf("Source = %q, want %q", got.Source, FromDrive)
	}
	if got.CacheError == "" {
		t.Error("CacheError is empty, and the cache was not written")
	}
}

// The cache is written the way every file gdoc replaces is written: through
// internal/atomicfile, readable by its owner alone, with no temp file left
// behind.
func TestTheCacheIsWrittenAtomically(t *testing.T) {
	dir := configDir(t)
	raw := pngOf(t, 8, 4)
	f := &fake{raw: raw}
	if _, err := Fetch(context.Background(), f, theID, housePNG); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, "assets"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != fs.FileMode(0o600) {
			t.Errorf("%s is %v, want 0600", e.Name(), got)
		}
	}
	if len(names) != 2 {
		t.Errorf("assets/ holds %v, want the bytes and the stamp and nothing else", names)
	}
}

// The stamp is one short line a person can read, in UTC to the second, with
// the hash of the bytes it was written beside.
func TestTheStampIsOneLineAPersonCanRead(t *testing.T) {
	configDir(t)
	raw := pngOf(t, 8, 4)
	f := &fake{raw: raw}
	got, err := Fetch(context.Background(), f, theID, housePNG)
	if err != nil {
		t.Fatal(err)
	}
	sp, err := config.AssetStampPath(theID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(sp)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	want := fmt.Sprintf(`{"fetched_at":%q,"sha256":%q}`,
		got.FetchedAt.UTC().Truncate(time.Second).Format(time.RFC3339), hex.EncodeToString(sum[:]))
	if string(b) != want {
		t.Errorf("the stamp reads\n%s\nwant\n%s", b, want)
	}
}

// Bytes that have not changed are not written again. The stamp moves, because
// the question was asked; the file does not, because the answer is the same.
func TestUnchangedBytesLeaveTheCacheFileAlone(t *testing.T) {
	configDir(t)
	raw := pngOf(t, 8, 4)
	seedCache(t, raw, 25*time.Hour, true)

	p, err := config.AssetPath(theID)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}

	f := &fake{raw: raw}
	if _, err := Fetch(context.Background(), f, theID, housePNG); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.ModTime().After(old.Add(time.Minute)) {
		t.Error("the cache file was written again with the bytes it already held")
	}
	_, stale, _ := readStamp(theID, time.Now())
	if stale {
		t.Error("the stamp was not moved, so the next run asks again")
	}
}

// The read is bounded. A body with no ceiling is a memory limit somebody else
// sets, and this one is a logo.
func TestTheReadIsBounded(t *testing.T) {
	configDir(t)
	f := &fake{raw: pngOf(t, 8, 4)}
	if _, err := Fetch(context.Background(), f, theID, housePNG); err != nil {
		t.Fatal(err)
	}
	if len(f.limits) != 1 || f.limits[0] != 1<<20 {
		t.Errorf("the read was bounded by %v, want one megabyte", f.limits)
	}
}

// An id that is not one never becomes a path. The id comes out of a house file
// a colleague may have written by hand.
func TestAnIdThatIsNotOneIsRefusedBeforeAnythingIsRead(t *testing.T) {
	configDir(t)
	f := &fake{raw: pngOf(t, 8, 4)}
	if got, err := Fetch(context.Background(), f, "../oauth-token.json", housePNG); err == nil {
		t.Errorf("Fetch() = %+v, want a refusal", got)
	}
	if f.calls != 0 {
		t.Errorf("%d requests, want none: the id was refused first", f.calls)
	}
}
