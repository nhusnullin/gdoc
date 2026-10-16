// Where the mark in the first-page header comes from, as the envelope reports
// it. Nothing here reaches Drive: the one door is the session factory, and
// every test in this file either stands in for it or fails the run if it is
// opened at all.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/config"
	"gdoc/internal/guard"
	"gdoc/internal/house"
)

// houseFile is the embedded style as a file, read so that a test can write a
// copy of it whose logo is in Drive. The style itself is internal/house's, and
// a test that wrote its own would be testing a file nothing ships.
const houseFile = "../../internal/house/house.yaml"

// logoFileID is the id a test house names. It is the shape Drive mints and not
// the firm's own file: nothing in this package may name a real id, because a
// test that did would be one slip away from asking Drive for it.
const logoFileID = "1TestLogoFileIdAAAAAAAAAAAAAAAAAAA"

// fileLogoHouse is a house file whose logo is bytes in the file: internal
// house's own copy of the shipped style with the placeholder's base64 where
// the firm's logo used to be. The shipped style names a file in Drive since
// M22, so a run that must fetch nothing is pointed at this one.
const fileLogoHouse = "../../internal/house/testdata/house-base64.yaml"

// placeholderLogo is the PNG every test here stands Drive up with. The firm's
// logo is one file in its Drive and the tree carries no copy of it, so the
// bytes come from internal/house's placeholder: a flat light grey of the size
// the house file states, which is what makes the room's own check pass.
const placeholderLogo = "../../internal/house/testdata/placeholder-logo.png"

// logoBytesKeys are the two keys a house file names its logo's bytes with.
// driveLogoHouse cuts at whichever of them is there, so it reads the same
// whether the shipped style carries the bytes or already names the file.
var logoBytesKeys = []string{"  base64: |-", "  drive: "}

// driveLogoHouse writes a copy of the embedded style whose logo is one file in
// Drive: every placement value kept, and the key that names the bytes replaced
// by logo.drive, which is this package's own test id and never the firm's.
func driveLogoHouse(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(houseFile)
	if err != nil {
		t.Fatal(err)
	}
	cut := -1
	for _, key := range logoBytesKeys {
		if at := strings.Index(string(src), key); at >= 0 {
			cut = at
			break
		}
	}
	if cut < 0 {
		t.Fatalf("%s names its logo's bytes with none of %v, so this test cannot build its Drive twin", houseFile, logoBytesKeys)
	}
	path := filepath.Join(t.TempDir(), "drive-logo.yaml")
	text := string(src[:cut]) + "  drive: " + logoFileID + "\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// housePNG is what a stubbed Drive answers with: a PNG of the size the house
// file states, so the room's own check passes and what is left under test is
// where the bytes came from.
func housePNG(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(placeholderLogo)
	if err != nil {
		t.Fatalf("read the placeholder logo: %v", err)
	}
	return raw
}

// logoWire is a session that answers one file's bytes, or refuses, and records
// what it was asked for. Every other method fails the test: the logo fetch is
// one GET and nothing else.
type logoWire struct {
	t   *testing.T
	png []byte
	err error
	// hang is Drive answering nothing at all: the read waits for the
	// context and ends with whatever ended it. It is the office network
	// rather than a refusal, and it is the one failure a status code cannot
	// stand in for, because its cost is time.
	hang  bool
	asked []string
}

func (w *logoWire) FileBytes(ctx context.Context, fileID string, _ int64) ([]byte, error) {
	w.asked = append(w.asked, fileID)
	if w.hang {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if w.err != nil {
		return nil, w.err
	}
	return w.png, nil
}

func (w *logoWire) GetJSON(_ context.Context, rawURL string, _ any) error {
	w.t.Errorf("the logo fetch reads bytes and no JSON: %s", rawURL)
	return fmt.Errorf("no JSON here")
}

func (w *logoWire) GetBytes(_ context.Context, rawURL string, _ int64) ([]byte, error) {
	w.t.Errorf("the logo fetch goes through FileBytes: %s", rawURL)
	return nil, fmt.Errorf("no export here")
}

// The write verbs refuse rather than failing the test, because a publish
// whose logo is stubbed reaches this same wire for its upload. What holds the
// fetch to one read is the policy it opened, which
// TestTheLogoFetchOpensTheAssetGrantAndNothingElse judges.
func (w *logoWire) PostJSON(_ context.Context, rawURL string, _ any, _ any) error {
	return fmt.Errorf("this wire answers one file's bytes and nothing else: %s", rawURL)
}

func (w *logoWire) PatchJSON(_ context.Context, rawURL string, _ any, _ any) error {
	return fmt.Errorf("this wire answers one file's bytes and nothing else: %s", rawURL)
}

func (w *logoWire) PostMultipart(_ context.Context, rawURL string, _ any, _ []byte, _ string, _ any) error {
	return fmt.Errorf("this wire answers one file's bytes and nothing else: %s", rawURL)
}

func (w *logoWire) Warnings() []string { return nil }

// logoFetching points the run at a config dir of its own and hands it the
// wire. It answers the policy the fetch opened, so a test can judge it.
func logoFetching(t *testing.T, w *logoWire) func() *guard.Policy {
	t.Helper()
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	var opened *guard.Policy
	old := openSession
	openSession = func(p *guard.Policy) (session, error) {
		opened = p
		return w, nil
	}
	t.Cleanup(func() { openSession = old })
	return func() *guard.Policy { return opened }
}

// cachedLogo writes bytes and a stamp of the given age into the run's config
// dir, which is how a test says the last fetch was yesterday.
func cachedLogo(t *testing.T, id string, raw []byte, age time.Duration) {
	t.Helper()
	path, err := config.AssetPath(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	stamp, err := config.AssetStampPath(id)
	if err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age).UTC().Truncate(time.Second).Format(time.RFC3339)
	text := fmt.Sprintf(`{"fetched_at":%q,"sha256":%q}`, when, assetSum(raw))
	if err := os.WriteFile(stamp, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// theHouseLogoCached hands the run a config dir of its own with the shipped
// house's logo already in it and its stamp fresh. That is the state every
// second run on a machine is in: the asset room answers out of the cache and
// asks Drive nothing. It is what lets a test about anything but the logo stand
// beside the real thing, the shipped style with no --house, and still open no
// wire at all. A cold cache is this file's own business.
func theHouseLogoCached(t *testing.T) {
	t.Helper()
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the embedded house style did not load: %v", err)
	}
	if cfg.LogoDrive() == "" {
		return
	}
	cachedLogo(t, cfg.LogoDrive(), housePNG(t), 0)
}

// logoBlockOf is the logo block out of the object, as a skill reads it.
func logoBlockOf(t *testing.T, got map[string]any) map[string]any {
	t.Helper()
	block, ok := dataOf(t, got)["logo"].(map[string]any)
	if !ok {
		t.Fatalf("data.logo is not an object: %v", got["data"])
	}
	return block
}

// A house whose logo is bytes in the file is built with no request at all, and
// the envelope says so in one word. The shipped style names a file in Drive, so
// the house this one is given is the copy that still carries its bytes.
func TestBuildReportsTheLogoSourceFile(t *testing.T) {
	noSession(t)
	dir, md := buildNote(t)
	out := filepath.Join(dir, "out.docx")

	got, code := runJSON(t, "build", "--md", md, "--out", out, "--house", fileLogoHouse)
	if code != 0 || got["ok"] != true {
		t.Fatalf("build: %v (exit %d)", got, code)
	}
	block := logoBlockOf(t, got)
	if block["source"] != "file" {
		t.Errorf("a logo in the house file came from the file: %v", block)
	}
	for _, key := range []string{"id", "fetched_at", "age_seconds", "reason", "cache_error"} {
		if _, ok := block[key]; ok {
			t.Errorf("nothing was fetched, so %s is left out: %v", key, block)
		}
	}
}

// A house whose logo is one file in Drive is fetched once, and the envelope
// names the file and when it was read.
func TestBuildReportsTheLogoSourceDrive(t *testing.T) {
	wire := &logoWire{t: t, png: housePNG(t)}
	logoFetching(t, wire)
	dir, md := buildNote(t)
	out := filepath.Join(dir, "out.docx")

	got, code := runJSON(t, "build", "--md", md, "--out", out, "--house", driveLogoHouse(t))
	if code != 0 || got["ok"] != true {
		t.Fatalf("build: %v (exit %d)", got, code)
	}
	if len(wire.asked) != 1 || wire.asked[0] != logoFileID {
		t.Errorf("one read of the one file the house names: %v", wire.asked)
	}
	block := logoBlockOf(t, got)
	if block["source"] != "drive" {
		t.Errorf("the bytes came from Drive: %v", block)
	}
	if block["id"] != logoFileID {
		t.Errorf("id names the file the house names: %v", block)
	}
	when, _ := block["fetched_at"].(string)
	if _, err := time.Parse(time.RFC3339, when); err != nil {
		t.Errorf("fetched_at is one instant in RFC 3339: %v", block)
	}
	if _, ok := block["reason"]; ok {
		t.Errorf("nothing went wrong, so there is no reason: %v", block)
	}
	// The document carries the mark it fetched.
	if !containsName(zipNames(t, out), "word/media/logo.png") {
		t.Errorf("the built document carries no logo: %v", zipNames(t, out))
	}
}

// Drive refusing with a cache beside the token builds from the cache, and the
// envelope carries its age and why Drive was not the answer.
func TestBuildReportsTheLogoSourceCache(t *testing.T) {
	wire := &logoWire{t: t, err: fmt.Errorf("403 the caller does not have permission")}
	logoFetching(t, wire)
	cachedLogo(t, logoFileID, housePNG(t), 30*time.Hour)
	dir, md := buildNote(t)
	out := filepath.Join(dir, "out.docx")

	got, code := runJSON(t, "build", "--md", md, "--out", out, "--house", driveLogoHouse(t))
	if code != 0 || got["ok"] != true {
		t.Fatalf("build: %v (exit %d)", got, code)
	}
	block := logoBlockOf(t, got)
	if block["source"] != "cache" {
		t.Errorf("Drive refused, so the cache is the answer: %v", block)
	}
	age, _ := block["age_seconds"].(float64)
	if age < float64(29*time.Hour/time.Second) {
		t.Errorf("age_seconds is how old the cached bytes are: %v", block)
	}
	reason, _ := block["reason"].(string)
	if !strings.Contains(reason, "403") {
		t.Errorf("reason is what Drive answered: %v", block)
	}
	// A person hears about it, because a document built from yesterday's mark
	// is a fact worth one line.
	if !anyWarning(got, "cache") {
		t.Errorf("a cached logo earns one warning: %v", got["warnings"])
	}
}

// A fresh stamp beside usable bytes makes no request at all: once a day, not
// once a run.
func TestBuildWithAFreshCacheAsksDriveNothing(t *testing.T) {
	wire := &logoWire{t: t, png: housePNG(t)}
	logoFetching(t, wire)
	cachedLogo(t, logoFileID, housePNG(t), time.Hour)
	dir, md := buildNote(t)

	got, code := runJSON(t, "build", "--md", md, "--out", filepath.Join(dir, "out.docx"), "--house", driveLogoHouse(t))
	if code != 0 || got["ok"] != true {
		t.Fatalf("build: %v (exit %d)", got, code)
	}
	if len(wire.asked) != 0 {
		t.Errorf("a fresh stamp asks Drive nothing: %v", wire.asked)
	}
	if logoBlockOf(t, got)["source"] != "cache" {
		t.Errorf("the bytes came from the cache: %v", logoBlockOf(t, got))
	}
}

// The ceiling is twenty seconds, stated here as a literal so that moving it
// is a decision somebody makes rather than a constant a test follows.
//
// Twenty rather than the two the update check gets, because that one is a
// fetch nobody asked for and this one is a document that cannot be built
// without it: the first build on a machine has no cache to fall back on, and
// the deadline has to cover a token refresh and the read behind it. Twenty
// rather than the guard's five minutes, because a build is a local thing and
// a hung network must not turn it into a wait.
func TestTheLogoCeilingIsTwentySeconds(t *testing.T) {
	if logoCeiling != 20*time.Second {
		t.Errorf("the logo fetch's ceiling is twenty seconds, and it is %v", logoCeiling)
	}
}

// Drive answering nothing at all costs the ceiling and no more. The cache is
// then the answer, exactly as a refusal makes it, and the envelope says the
// deadline was what stopped the read.
//
// Without a deadline of its own the fetch inherits the guard's five minutes,
// doubled by the retry after a 401, so a build that had yesterday's mark in
// hand would sit on a hung network for ten minutes before using it. That is
// the whole reason the ceiling exists.
func TestAHungDriveCostsTheCeilingAndThenTheCacheAnswers(t *testing.T) {
	wire := &logoWire{t: t, hang: true}
	logoFetching(t, wire)
	cachedLogo(t, logoFileID, housePNG(t), 30*time.Hour)
	old := logoCeiling
	logoCeiling = 20 * time.Millisecond
	t.Cleanup(func() { logoCeiling = old })
	dir, md := buildNote(t)

	start := time.Now()
	got, code := runJSON(t, "build", "--md", md, "--out", filepath.Join(dir, "out.docx"), "--house", driveLogoHouse(t))
	took := time.Since(start)

	if code != 0 || got["ok"] != true {
		t.Fatalf("a hung Drive with a cache still builds: %v (exit %d)", got, code)
	}
	// Generous against a loaded machine, and still three orders of magnitude
	// under the five minutes the guard alone would have allowed.
	if took > 5*time.Second {
		t.Errorf("the fetch is bounded by the ceiling, and the build took %v", took)
	}
	block := logoBlockOf(t, got)
	if block["source"] != "cache" {
		t.Errorf("Drive never answered, so the cache is the answer: %v", block)
	}
	if reason, _ := block["reason"].(string); !strings.Contains(reason, "deadline") {
		t.Errorf("reason says the read ran out of time: %v", block)
	}
}

// No sign-in and no cache refuses the build and names the login. It never
// falls back to an unbranded document.
func TestBuildWithNoTokenAndNoCacheRefusesNamingTheLogin(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	dir, md := buildNote(t)
	out := filepath.Join(dir, "out.docx")

	got, code := runJSON(t, "build", "--md", md, "--out", out, "--house", driveLogoHouse(t))
	if code == 0 || got["ok"] == true {
		t.Fatalf("a build with no logo is refused: %v (exit %d)", got, code)
	}
	why, _ := got["error"].(string)
	if !strings.Contains(why, "gdoc auth login") {
		t.Errorf("the refusal names the login: %q", why)
	}
	if _, err := os.Stat(out); err == nil {
		t.Errorf("nothing was written, because nothing was built")
	}
}

// The file in Drive being the wrong file refuses the run even with a good
// cache: the firm's file is wrong and the refusal is the only message that
// reaches whoever can fix it.
func TestBuildRefusesAWrongFileInDriveWithAGoodCache(t *testing.T) {
	wire := &logoWire{t: t, png: wrongSizePNG(t)}
	logoFetching(t, wire)
	cachedLogo(t, logoFileID, housePNG(t), 30*time.Hour)
	dir, md := buildNote(t)

	got, code := runJSON(t, "build", "--md", md, "--out", filepath.Join(dir, "out.docx"), "--house", driveLogoHouse(t))
	if code == 0 || got["ok"] == true {
		t.Fatalf("a wrong file in Drive stops the run: %v (exit %d)", got, code)
	}
	if why, _ := got["error"].(string); !strings.Contains(why, logoFileID) {
		t.Errorf("the refusal names the file: %q", why)
	}
}

// The policy the fetch opens is the seventh grant and nothing else, judged the
// way TestTheHelpCheckOpensThePolicyTheUpdateOpens judges the check's.
func TestTheLogoFetchOpensTheAssetGrantAndNothingElse(t *testing.T) {
	wire := &logoWire{t: t, png: housePNG(t)}
	opened := logoFetching(t, wire)
	dir, md := buildNote(t)

	got, code := runJSON(t, "build", "--md", md, "--out", filepath.Join(dir, "out.docx"), "--house", driveLogoHouse(t))
	if code != 0 || got["ok"] != true {
		t.Fatalf("build: %v (exit %d)", got, code)
	}
	p := opened()
	if p == nil {
		t.Fatal("the fetch opened no policy")
	}
	judge := func(method, rawURL string) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			t.Fatal(err)
		}
		return p.Judge(method, u, nil)
	}
	admitted := "https://www.googleapis.com/drive/v3/files/" + logoFileID + "?alt=media&supportsAllDrives=true"
	if err := judge("GET", admitted); err != nil {
		t.Errorf("the one read the fetch makes: %v", err)
	}
	for _, refused := range [][2]string{
		{"GET", "https://www.googleapis.com/drive/v3/files/" + logoFileID},
		{"GET", "https://www.googleapis.com/drive/v3/files/" + logoFileID + "?alt=media&supportsAllDrives=true&fields=id"},
		{"POST", "https://www.googleapis.com/drive/v3/files/" + logoFileID + "/copy?alt=media&supportsAllDrives=true"},
		{"GET", "https://docs.googleapis.com/v1/documents/" + logoFileID},
		{"GET", "https://www.googleapis.com/drive/v3/files/1AbCdEfGhIjKlMnOpQrStUvWxYz012345?alt=media&supportsAllDrives=true"},
		{"GET", "https://api.github.com/repos/nhusnullin/gdoc/releases"},
	} {
		if err := judge(refused[0], refused[1]); err == nil {
			t.Errorf("%s %s must be refused by the fetch's policy", refused[0], refused[1])
		}
	}
}

// publish reports the same block over its own envelope, because both commands
// render through one renderNote and a second reading would slowly stop being
// the same one.
func TestPublishReportsTheLogoSource(t *testing.T) {
	wire := &logoWire{t: t, png: housePNG(t)}
	logoFetching(t, wire)
	cachedLogo(t, logoFileID, housePNG(t), time.Hour)
	_, md := buildNote(t)

	// The upload itself is not what this test is about, so it is left to fail
	// on the wire the logo stub is: what matters is the block in the envelope.
	got, _ := runJSON(t, "publish", "--md", md, "--folder-id", "1AbCdEfGhIjKlMnOpQrStUvWxYz012345", "--house", driveLogoHouse(t))
	block := logoBlockOf(t, got)
	if block["source"] != "cache" {
		t.Errorf("publish prints where the logo came from: %v", block)
	}
	if block["id"] != logoFileID {
		t.Errorf("id names the file the house names: %v", block)
	}
}

// A house whose logo is bytes is reported as such by publish too.
func TestPublishReportsALogoInTheFile(t *testing.T) {
	_, md := buildNote(t)
	wire := &logoWire{t: t}
	logoFetching(t, wire)

	got, _ := runJSON(t, "publish", "--md", md, "--folder-id", "1AbCdEfGhIjKlMnOpQrStUvWxYz012345", "--house", fileLogoHouse)
	if block := logoBlockOf(t, got); block["source"] != "file" {
		t.Errorf("publish prints where the logo came from: %v", block)
	}
	if len(wire.asked) != 0 {
		t.Errorf("a logo in the house file is fetched from nowhere: %v", wire.asked)
	}
}

// wrongSizePNG is a PNG of the wrong size: the one file that must refuse the
// run however good the cache is.
func wrongSizePNG(t *testing.T) []byte {
	t.Helper()
	// A 1x1 greyscale PNG, written out as base64 so this test states its own
	// bytes rather than building an image.
	raw, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAAAAAA6fptVAAAACklEQVR4nGP4DwABAQEAWk1v8QAAAABJRU5ErkJggg==")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// assetSum is the hash the stamp beside the cached bytes carries. It is
// spelled out here rather than read from internal/asset, because this test
// writes the two files an earlier run would have left behind.
func assetSum(raw []byte) string {
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

// containsName says whether the archive holds that part.
func containsName(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

// anyWarning says whether one of the envelope's warnings mentions that word.
func anyWarning(got map[string]any, word string) bool {
	list, _ := got["warnings"].([]any)
	for _, w := range list {
		if s, ok := w.(string); ok && strings.Contains(s, word) {
			return true
		}
	}
	return false
}

// Every command but build and publish is exactly as offline from Drive's file
// bytes as it was. A read with a config dir holding nothing fetches no logo,
// prints no logo block, and still answers: the fake session refuses FileBytes
// by name, so a read that had grown one would fail here rather than pass
// quietly. TestReadExportRestyleAndHelpNeverReachTheAssetRoom in go/boundary
// holds the same rule over the call graph.
func TestReadNeverFetchesTheLogo(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	stubSession(t, docsAndComments(t))

	got, code := runJSON(t, "read", fixtureDocID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("read: %v (exit %d)", got, code)
	}
	if _, ok := dataOf(t, got)["logo"]; ok {
		t.Errorf("read renders nothing, so it says nothing about a logo: %v", got["data"])
	}
}
