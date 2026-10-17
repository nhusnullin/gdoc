// M22's live measurement: the mark in a built document is the firm's own file
// in Drive, read through the asset room with the real token.
//
// Every other logo assertion in this package reads geometry, and geometry
// cannot tell the house logo from the placeholder beside it, which is the same
// 195 by 97 pixels on purpose. This is the one run that reads the bytes, so it
// is the one run that can say the tree carrying no logo did not cost the
// documents their mark.
//
// It creates nothing in Drive. What it writes is a cache and a stamp, in a
// config directory made for the run and thrown away with it.

package live

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"gdoc/internal/asset"
	"gdoc/internal/config"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/house"
)

// houseLogoSHA256 is the SHA-256 of the firm's logo, the file house.yaml names
// by id. It is the literal go/boundary's TestTheLogoIsNotInTheTree holds, and
// it is stated again here rather than imported, because the two tests ask
// opposite questions of the same bytes: that one holds that nothing in this
// tree is that file, and this one holds that what Drive answered is.
//
// A hash and not the bytes, for that test's reason: a copy of the logo here
// would be the first file to break the rule the milestone exists for.
const houseLogoSHA256 = "a0c4fc99472c7a8fb069d387b8755dd73f622c2ce261c35a34871e5f1af2b3ef"

// logoMember is where render puts the mark in every document it builds.
const logoMember = "word/media/logo.png"

// countedReach is this run's door to the firm's file, and the count of how
// often it was opened.
//
// It is cmd/gdoc's logoReach spelled here, because that package is main: one
// policy carrying the asset grant alone, and a session opened at the moment
// the room asks for the bytes rather than before. Opening late is what makes
// the second half of this test mean something: a run whose stamp is fresh
// asks nothing, loads no token and refreshes none.
//
// The grant names the id this reach was made for, and the read is of the id
// the room asks about. They are the same file, and the guard is what says so:
// a room that asked for another would be refused before anything left.
type countedReach struct {
	t     *testing.T
	id    string
	calls int
}

// FileBytes is asset.Drive: the one read this reach makes.
func (r *countedReach) FileBytes(ctx context.Context, fileID string, limit int64) ([]byte, error) {
	r.calls++
	p := guard.NewPolicy()
	p.AllowAsset(r.id)
	s, err := gapi.Open(p, nil)
	if err != nil {
		return nil, err
	}
	raw, err := s.FileBytes(ctx, fileID, limit)
	for _, w := range s.Warnings() {
		r.t.Logf("session warning: %s", w)
	}
	return raw, err
}

// TestPublishCarriesTheHouseLogo is M22's acceptance, asked of Drive.
//
// A build in a config directory with no cache reads the firm's file once, the
// bytes hash to the literal above, and the document that comes out carries
// them as word/media/logo.png. A second build in the same directory reads the
// cache and asks Drive nothing, which is the whole of what makes a house logo
// in Drive affordable: one request a day, not one a document.
func TestPublishCarriesTheHouseLogo(t *testing.T) {
	ctx := liveLogoRun(t)

	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the embedded house style did not load: %v", err)
	}
	id := cfg.LogoDrive()
	if id == "" {
		t.Fatal("the embedded house style carries its own logo bytes, so there is no file in Drive for this run to measure")
	}
	want := asset.Want{Mime: cfg.Logo.Mime, Width: cfg.Logo.Px[0], Height: cfg.Logo.Px[1]}
	t.Logf("the house names logo %s, stated as %s at %dx%d px", id, want.Mime, want.Width, want.Height)

	// The first build. Nothing is cached, so the room asks Drive once.
	first := &countedReach{t: t, id: id}
	got, err := asset.Fetch(ctx, first, id, want)
	if err != nil {
		t.Fatalf("the house logo could not be read out of Drive: %v", err)
	}
	if first.calls != 1 {
		t.Errorf("the first build asked Drive %d times, and a build with no cache asks once", first.calls)
	}
	if got.Source != asset.FromDrive {
		t.Errorf("the first build read the logo from %q, and a config directory with no cache has only Drive to read it from (reason: %q)", got.Source, got.Reason)
	}
	if got.CacheError != "" {
		t.Errorf("the fetched logo could not be cached, so the second half of this test measures nothing: %s", got.CacheError)
	}
	if sum := hex.EncodeToString(sliceSum(got.Raw)); sum != houseLogoSHA256 {
		t.Errorf("Drive answered %d bytes hashing to %s, and the firm's logo hashes to %s", len(got.Raw), sum, houseLogoSHA256)
	}
	t.Logf("read %d bytes from Drive at %s", len(got.Raw), got.FetchedAt.UTC())

	// The document those bytes go into. This is the production render, handed
	// the mark the production fetch returned.
	note := filepath.Join(t.TempDir(), "live-house-logo.md")
	if err := os.WriteFile(note, []byte(publishNoteSource), 0o600); err != nil {
		t.Fatalf("the note could not be written: %v", err)
	}
	doc := renderNoteFileWithLogo(t, note, got.Raw)
	z, err := openZip(doc.docx)
	if err != nil {
		t.Fatalf("the built document is not a zip: %v", err)
	}
	member, err := zipPart(z, logoMember)
	if err != nil {
		t.Fatalf("the built document carries no mark: %v", err)
	}
	if sum := hex.EncodeToString(sliceSum(member)); sum != houseLogoSHA256 {
		t.Errorf("%s in the built document hashes to %s, and the firm's logo hashes to %s", logoMember, sum, houseLogoSHA256)
	}
	if !bytes.Equal(member, got.Raw) {
		t.Errorf("%s is %d bytes and Drive answered %d, so the render did not carry the file through whole", logoMember, len(member), len(got.Raw))
	}

	// The cache the first build wrote, which is the file the second one reads.
	for _, p := range []struct {
		what string
		path func(string) (string, error)
	}{{"the cached bytes", config.AssetPath}, {"the stamp", config.AssetStampPath}} {
		path, err := p.path(id)
		if err != nil {
			t.Fatalf("%s has no path: %v", p.what, err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s is not at %s after a fetch: %v", p.what, path, err)
		}
	}

	// The second build, in the same config directory. The stamp is minutes
	// old, so the wire stays shut.
	second := &countedReach{t: t, id: id}
	again, err := asset.Fetch(ctx, second, id, want)
	if err != nil {
		t.Fatalf("the second build could not read the logo it had just cached: %v", err)
	}
	if second.calls != 0 {
		t.Errorf("the second build asked Drive %d times, and a stamp written a moment ago is the answer", second.calls)
	}
	if again.Source != asset.FromCache {
		t.Errorf("the second build read the logo from %q, and a fresh stamp reads the cache", again.Source)
	}
	if again.Reason != "" {
		t.Errorf("the second build names %q as the reason it used the cache, and nothing went wrong: a fresh stamp is the designed path", again.Reason)
	}
	if !bytes.Equal(again.Raw, got.Raw) {
		t.Errorf("the cache holds %d bytes and Drive answered %d", len(again.Raw), len(got.Raw))
	}
}

// sliceSum is the SHA-256 of one buffer, as a slice, because an array cannot
// be handed to hex.EncodeToString.
func sliceSum(raw []byte) []byte {
	sum := sha256.Sum256(raw)
	return sum[:]
}

// liveLogoRun is this test's opening: the two variables, and a config
// directory of its own holding a copy of the real token.
//
// The directory is a temp one because the measurement is about a build that
// has nothing cached, and this machine's real config directory is where every
// other run of gdoc has been leaving its cache. Reading the question off a
// cache somebody else wrote would answer about that run instead of this one.
//
// The token is copied in because the read is of the firm's file and only the
// signed-in account may read it. A refresh during the run is written into the
// copy and the real token file is left exactly as it was, which costs the next
// real command one refresh of its own.
func liveLogoRun(t *testing.T) context.Context {
	t.Helper()
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to read the firm's logo out of Drive with the real token", liveVar, writeVar)
	}
	real, err := config.TokenPath()
	if err != nil {
		t.Fatalf("this machine's token has no path: %v", err)
	}
	raw, err := os.ReadFile(real)
	if err != nil {
		t.Fatalf("the token at %s could not be read; run: gdoc auth login: %v", real, err)
	}

	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	copied, err := config.TokenPath()
	if err != nil {
		t.Fatalf("the run's own config directory has no token path: %v", err)
	}
	if err := os.WriteFile(copied, raw, 0o600); err != nil {
		t.Fatalf("the token could not be copied into the run's own config directory: %v", err)
	}
	t.Logf("this run's config directory is %s, with no cache in it", dir)
	return context.Background()
}

// liveHouseLogo is the firm's mark, for a live test that is about something
// else and wants the document it publishes to be the document production
// publishes.
//
// It goes through the production fetch in this machine's own config
// directory, so it costs one request a day across every live run rather than
// one per test, and it holds the answer to the same hash the measurement
// above holds it to. A house style carrying its own bytes needs no reach at
// all and is handed straight back, which is what keeps this usable if the
// embedded file ever carries a logo again.
func liveHouseLogo(t *testing.T, cfg *house.Config) []byte {
	t.Helper()
	id := cfg.LogoDrive()
	if id == "" {
		return houseLogo(t, cfg)
	}
	got, err := asset.Fetch(context.Background(), &countedReach{t: t, id: id}, id, asset.Want{
		Mime:   cfg.Logo.Mime,
		Width:  cfg.Logo.Px[0],
		Height: cfg.Logo.Px[1],
	})
	if err != nil {
		t.Fatalf("the house logo could not be read: %v", err)
	}
	if got.Reason != "" {
		t.Logf("the house logo came from the cache: %s", got.Reason)
	}
	if sum := hex.EncodeToString(sliceSum(got.Raw)); sum != houseLogoSHA256 {
		t.Fatalf("the house logo hashes to %s, and the firm's logo hashes to %s", sum, houseLogoSHA256)
	}
	return got.Raw
}

// carriesTheHouseLogo holds one built document to the firm's mark. It is the
// assertion TestLivePublish makes about the bytes it is about to upload: the
// document a person opens in Drive is branded, and nothing on the way there
// swapped the picture for the placeholder the offline tests render with.
func carriesTheHouseLogo(t *testing.T, docx []byte) {
	t.Helper()
	z, err := openZip(docx)
	if err != nil {
		t.Fatalf("the built document is not a zip: %v", err)
	}
	member, err := zipPart(z, logoMember)
	if err != nil {
		t.Fatalf("the built document carries no mark: %v", err)
	}
	if sum := hex.EncodeToString(sliceSum(member)); sum != houseLogoSHA256 {
		t.Errorf("%s in the document being published hashes to %s, and the firm's logo hashes to %s", logoMember, sum, houseLogoSHA256)
	}
}
