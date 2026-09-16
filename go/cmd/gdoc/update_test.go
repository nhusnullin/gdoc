// The update command's tests: the flags it refuses, the wire it is allowed,
// the object a skill reads, and the two files on disk afterwards.
//
// Nothing here reaches GitHub. The reach is a stub that answers a literal
// listing and literal bytes, so what is measured is what the command does with
// an answer rather than what GitHub does with a request.

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gdoc/internal/guard"
	"gdoc/internal/update"
)

// stubPlain stands in for the credential-free reach. Every URL it is handed is
// recorded, so a test that says nothing was fetched can prove it.
type stubPlain struct {
	listing string
	listErr error
	files   map[string][]byte
	got     []string
}

func (s *stubPlain) GetJSON(_ context.Context, rawURL string, into any) error {
	s.got = append(s.got, rawURL)
	if s.listErr != nil {
		return s.listErr
	}
	return json.Unmarshal([]byte(s.listing), into)
}

func (s *stubPlain) GetBytes(_ context.Context, rawURL string, _ int64) ([]byte, error) {
	s.got = append(s.got, rawURL)
	b, ok := s.files[rawURL]
	if !ok {
		return nil, fmt.Errorf("nothing is published at %s", rawURL)
	}
	return b, nil
}

func (s *stubPlain) Warnings() []string { return nil }

// installedAt writes a binary at a path no other test shares, points the
// command at it, and hands the run the stub reach. It answers the path.
func installedAt(t *testing.T, ver string, content []byte, pl plain) string {
	t.Helper()
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "gdoc")
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatal(err)
	}
	oldVersion, oldExecutable, oldOpen := version, executable, openPlain
	version = ver
	executable = func() (string, error) { return path, nil }
	openPlain = func(*guard.Policy) plain { return pl }
	t.Cleanup(func() { version, executable, openPlain = oldVersion, oldExecutable, oldOpen })
	return path
}

func assetURL(tag, name string) string {
	return "https://github.com/nhusnullin/gdoc/releases/download/" + tag + "/" + name
}

// listing is the releases answer, as api.github.com writes it, carrying the
// two assets the release workflow publishes for this machine's platform.
func listing(tags ...string) string {
	platform := update.Platform(runtime.GOOS, runtime.GOARCH)
	entries := make([]string, 0, len(tags))
	for _, tag := range tags {
		zipName := update.AssetName(tag, platform)
		sumsName := update.ChecksumsName(tag)
		entries = append(entries, fmt.Sprintf(
			`{"tag_name":%q,"draft":false,"assets":[{"name":%q,"browser_download_url":%q},{"name":%q,"browser_download_url":%q}]}`,
			tag, zipName, assetURL(tag, zipName), sumsName, assetURL(tag, sumsName)))
	}
	return "[" + strings.Join(entries, ",") + "]"
}

// zipHolding packs the binary the way the release workflow packs one: at the
// top of the archive, with the files the installer reads beside it.
func zipHolding(t *testing.T, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	add := func(name string, body []byte, mode os.FileMode) {
		t.Helper()
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(mode)
		f, err := w.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	add("gdoc", content, 0o755)
	add("install.sh", []byte("#!/usr/bin/env bash\n"), 0o755)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func hexSum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// published is one release's two files, keyed by the URL the listing names.
func published(t *testing.T, tag string, binary []byte) map[string][]byte {
	t.Helper()
	platform := update.Platform(runtime.GOOS, runtime.GOARCH)
	asset := update.AssetName(tag, platform)
	archive := zipHolding(t, binary)
	return map[string][]byte{
		assetURL(tag, asset):                     archive,
		assetURL(tag, update.ChecksumsName(tag)): []byte(fmt.Sprintf("%s  %s\n", hexSum(archive), asset)),
	}
}

func merged(maps ...map[string][]byte) map[string][]byte {
	out := map[string][]byte{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func updateObject(t *testing.T, got map[string]any) map[string]any {
	t.Helper()
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	return data
}

func fileText(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// nothingMoved is the whole of what a run that installs nothing leaves: the
// binary as it was, and no earlier binary beside it.
func nothingMoved(t *testing.T, path, content string) {
	t.Helper()
	if got := fileText(t, path); got != content {
		t.Errorf("the binary at %s must be untouched, and holds %q", path, got)
	}
	if _, err := os.Stat(path + ".previous"); err == nil {
		t.Errorf("nothing was installed, so there must be no %s.previous", path)
	}
}

// A flag pair that names two runs is refused by name, and so is a word and a
// flag update does not take. None of them reaches GitHub: the refusal comes
// before the wire, because a run that was not understood does not fetch.
func TestUpdateRefusesWhatMeansTwoThings(t *testing.T) {
	for _, want := range []struct {
		args  []string
		names []string
	}{
		{[]string{"update", "--rollback", "--check"}, []string{"--rollback", "--check"}},
		{[]string{"update", "--rollback", "--major"}, []string{"--rollback", "--major"}},
		{[]string{"update", "--rollback", "--nightly"}, []string{"--rollback", "--nightly"}},
		{[]string{"update", "v2.1.0"}, []string{"v2.1.0"}},
		{[]string{"update", "--force"}, []string{"--force"}},
		{[]string{"update", "--check", "--check"}, []string{"--check"}},
	} {
		pl := &stubPlain{listing: listing("v2.1.0")}
		installedAt(t, "v2.0.0", []byte("old"), pl)

		got, code := runJSON(t, want.args...)
		if code == 0 || got["ok"] != false {
			t.Errorf("%v must be refused: %v (exit %d)", want.args, got, code)
			continue
		}
		msg, _ := got["error"].(string)
		for _, name := range want.names {
			if !strings.Contains(msg, name) {
				t.Errorf("%v must be refused naming %q: %q", want.args, name, msg)
			}
		}
		if len(pl.got) != 0 {
			t.Errorf("%v was refused, so nothing may be fetched: %v", want.args, pl.got)
		}
	}
}

// --check is the rehearsal: it says what the run without it would take, and
// leaves the machine exactly as it found it.
func TestACheckWritesNothingAndNamesWhatItWouldTake(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.2", "v2.1.0", "v2.0.0"),
		files:   published(t, "v2.1.0", []byte("new")),
	}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)

	got, code := runJSON(t, "update", "--check")
	if code != 0 || got["ok"] != true {
		t.Fatalf("a check must answer: %v (exit %d)", got, code)
	}
	data := updateObject(t, got)
	for field, want := range map[string]any{
		"installed":      "v2.0.0",
		"latest_stable":  "v2.1.0",
		"latest_nightly": "v2.1.2",
		"action":         "checked",
		"to":             "v2.1.0",
		"path":           path,
	} {
		if data[field] != want {
			t.Errorf("%s must be %v, and is %v", field, want, data[field])
		}
	}
	if data["verified"] == true || data["sha256"] != nil || data["previous"] != nil {
		t.Errorf("a check installs nothing, so it verifies nothing: %v", data)
	}
	// The exact URL, query and all: the page size is what keeps the last
	// hand-cut stable release on the page a month after the nightly starts
	// cutting over it, and nothing else in this package pins it.
	if len(pl.got) != 1 || pl.got[0] != releasesURL {
		t.Errorf("a check reads %s and downloads nothing: %v", releasesURL, pl.got)
	}
	nothingMoved(t, path, "old")
}

// The object, field for field, and the two files it describes.
func TestTheUpdateObjectIsWhatTheSkillsRead(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.2", "v2.1.0", "v2.0.0"),
		files:   merged(published(t, "v2.1.0", []byte("new")), published(t, "v2.1.2", []byte("nightly"))),
	}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)

	got, code := runJSON(t, "update")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the update must answer: %v (exit %d)", got, code)
	}
	data := updateObject(t, got)
	for field, want := range map[string]any{
		"installed":      "v2.0.0",
		"latest_stable":  "v2.1.0",
		"latest_nightly": "v2.1.2",
		"action":         "updated",
		"to":             "v2.1.0",
		"path":           path,
		"verified":       true,
		"sha256":         hexSum([]byte("new")),
		"previous":       path + ".previous",
	} {
		if data[field] != want {
			t.Errorf("%s must be %v, and is %v", field, want, data[field])
		}
	}
	if fileText(t, path) != "new" {
		t.Errorf("the binary at %s must be the one that was published: %q", path, fileText(t, path))
	}
	if fileText(t, path+".previous") != "old" {
		t.Errorf("the binary that was there must be kept: %q", fileText(t, path+".previous"))
	}
}

// --nightly takes the newest release there is, whichever channel cut it.
func TestANightlyRunTakesTheNewestThereIs(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.2", "v2.1.0", "v2.0.0"),
		files:   merged(published(t, "v2.1.0", []byte("new")), published(t, "v2.1.2", []byte("nightly"))),
	}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)

	got, code := runJSON(t, "update", "--nightly")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the nightly update must answer: %v (exit %d)", got, code)
	}
	data := updateObject(t, got)
	if data["action"] != "updated" || data["to"] != "v2.1.2" {
		t.Errorf("--nightly takes v2.1.2: %v", data)
	}
	if fileText(t, path) != "nightly" {
		t.Errorf("the nightly binary must be in place: %q", fileText(t, path))
	}
}

// A stable run that has nothing to take names the nightly command rather than
// taking it, because a nightly is a decision the person makes.
func TestUpToDateNamesTheNightlyBehindIt(t *testing.T) {
	pl := &stubPlain{listing: listing("v2.1.2", "v2.1.0")}
	path := installedAt(t, "v2.1.0", []byte("old"), pl)

	got, code := runJSON(t, "update")
	if code != 0 || got["ok"] != true {
		t.Fatalf("an up-to-date binary is an answer: %v (exit %d)", got, code)
	}
	data := updateObject(t, got)
	if data["action"] != "up_to_date" || data["run"] != "gdoc update --nightly" {
		t.Errorf("up to date, with the nightly named: %v", data)
	}
	if data["to"] != nil {
		t.Errorf("there is nothing to take, so nothing is named as the target: %v", data)
	}
	nothingMoved(t, path, "old")
}

// A major is named and not taken, and the flag that names it takes it.
func TestAMajorIsNamedAndNotTakenWithoutTheFlag(t *testing.T) {
	files := published(t, "v3.0.0", []byte("three"))
	pl := &stubPlain{listing: listing("v3.0.0", "v2.0.0"), files: files}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)

	got, code := runJSON(t, "update")
	if code != 0 || got["ok"] != true {
		t.Fatalf("a major available is an answer: %v (exit %d)", got, code)
	}
	data := updateObject(t, got)
	if data["action"] != "major_available" || data["to"] != "v3.0.0" || data["run"] != "gdoc update --major" {
		t.Errorf("a major is named and not taken: %v", data)
	}
	nothingMoved(t, path, "old")
	if len(pl.got) != 1 {
		t.Errorf("nothing was taken, so nothing was downloaded: %v", pl.got)
	}

	pl2 := &stubPlain{listing: listing("v3.0.0", "v2.0.0"), files: files}
	path2 := installedAt(t, "v2.0.0", []byte("old"), pl2)
	got, code = runJSON(t, "update", "--major")
	if code != 0 || got["ok"] != true {
		t.Fatalf("--major must take it: %v (exit %d)", got, code)
	}
	if data := updateObject(t, got); data["action"] != "updated" || data["to"] != "v3.0.0" {
		t.Errorf("--major takes v3.0.0: %v", data)
	}
	if fileText(t, path2) != "three" {
		t.Errorf("the major binary must be in place: %q", fileText(t, path2))
	}
}

// GitHub not answering is a fact about today, not a broken run. A person who
// typed `gdoc update` on a train is told what happened, in an envelope that
// says ok, with the binary they have still there.
func TestAnUnreachableGitHubIsAnAnswerAndNotAFailure(t *testing.T) {
	for _, cause := range []string{
		"context deadline exceeded",
		"connect: connection refused",
		"answered 503: Service Unavailable",
		"answered 403: API rate limit exceeded for 1.2.3.4",
	} {
		pl := &stubPlain{listErr: fmt.Errorf("the request to https://api.github.com/repos/nhusnullin/gdoc/releases failed: %s", cause)}
		path := installedAt(t, "v2.0.0", []byte("old"), pl)

		got, code := runJSON(t, "update")
		if code != 0 || got["ok"] != true {
			t.Errorf("%q must be an answer: %v (exit %d)", cause, got, code)
			continue
		}
		data := updateObject(t, got)
		if data["action"] != "unreachable" || data["installed"] != "v2.0.0" {
			t.Errorf("%q must answer unreachable and name the binary there: %v", cause, data)
		}
		warnings, _ := got["warnings"].([]any)
		if len(warnings) == 0 || !strings.Contains(fmt.Sprint(warnings...), cause) {
			t.Errorf("%q must reach a warning: %v", cause, got["warnings"])
		}
		nothingMoved(t, path, "old")
	}
}

// The release published a checksum and the bytes do not match it. Nothing on
// disk moves, and the refusal names the asset.
func TestAZipThatDoesNotMatchTheChecksumReplacesNothing(t *testing.T) {
	platform := update.Platform(runtime.GOOS, runtime.GOARCH)
	asset := update.AssetName("v2.1.0", platform)
	files := published(t, "v2.1.0", []byte("new"))
	files[assetURL("v2.1.0", update.ChecksumsName("v2.1.0"))] = []byte(fmt.Sprintf("%s  %s\n", hexSum([]byte("something else")), asset))
	pl := &stubPlain{listing: listing("v2.1.0", "v2.0.0"), files: files}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)

	got, code := runJSON(t, "update")
	if code == 0 || got["ok"] != false {
		t.Fatalf("a zip that does not verify must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, asset) {
		t.Errorf("the refusal must name the asset: %q", msg)
	}
	if data := updateObject(t, got); data["action"] != nil {
		t.Errorf("nothing was done, so the object claims no action: %v", data)
	}
	nothingMoved(t, path, "old")
}

// --rollback puts the earlier binary back, reads no listing at all, and is a
// swap: what was running is kept as the previous one.
func TestRollbackPutsTheEarlierBinaryBack(t *testing.T) {
	pl := &stubPlain{listing: listing("v2.1.0")}
	path := installedAt(t, "v2.1.0", []byte("new"), pl)
	if err := os.WriteFile(path+".previous", []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "update", "--rollback")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the rollback must answer: %v (exit %d)", got, code)
	}
	data := updateObject(t, got)
	if data["action"] != "rolled_back" || data["previous"] != path+".previous" {
		t.Errorf("the rollback says what it did: %v", data)
	}
	if data["sha256"] != hexSum([]byte("old")) {
		t.Errorf("the hash is of the binary now at the path: %v", data)
	}
	if data["verified"] == true {
		t.Errorf("a rollback is checked against no published checksum, so it claims none: %v", data)
	}
	if _, ok := data["installed"]; ok {
		t.Errorf("the binary at the path is the earlier one and gdoc cannot read its version, so the object claims none: %v", data)
	}
	if fileText(t, path) != "old" || fileText(t, path+".previous") != "new" {
		t.Errorf("the two binaries must have swapped: %q and %q", fileText(t, path), fileText(t, path+".previous"))
	}
	if len(pl.got) != 0 {
		t.Errorf("a rollback is a local file move and reads nothing: %v", pl.got)
	}
}

func TestARollbackWithNoEarlierBinaryIsRefusedByName(t *testing.T) {
	pl := &stubPlain{}
	path := installedAt(t, "v2.1.0", []byte("new"), pl)

	got, code := runJSON(t, "update", "--rollback")
	if code == 0 || got["ok"] != false {
		t.Fatalf("there is nothing to go back to, so the run must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, path+".previous") {
		t.Errorf("the refusal must name the file it looked for: %q", msg)
	}
	if fileText(t, path) != "new" {
		t.Errorf("the binary must be untouched: %q", fileText(t, path))
	}
}

// A checkout install is a symlink into somebody's repository. Replacing it
// would put a release binary where the link is and leave the checkout's own
// build unreachable, so it is refused naming both paths.
func TestASymlinkedBinaryIsRefusedByName(t *testing.T) {
	pl := &stubPlain{listing: listing("v2.1.0", "v2.0.0")}
	binary := installedAt(t, "v2.0.0", []byte("old"), pl)
	link := filepath.Join(t.TempDir(), "gdoc")
	if err := os.Symlink(binary, link); err != nil {
		t.Fatal(err)
	}
	executable = func() (string, error) { return link, nil }

	got, code := runJSON(t, "update")
	if code == 0 || got["ok"] != false {
		t.Fatalf("a symlinked binary must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, link) || !strings.Contains(msg, binary) {
		t.Errorf("the refusal must name the link and what it points at: %q", msg)
	}
	if len(pl.got) != 0 {
		t.Errorf("there is nothing to replace, so nothing is fetched: %v", pl.got)
	}
}

// The policy the run opens is the fifth grant and nothing else: the releases
// listing of this one repository, and no Google file at all.
func TestTheUpdateRunReachesTheReleasesAndNothingElse(t *testing.T) {
	pl := &stubPlain{listing: listing("v2.0.0")}
	installedAt(t, "v2.0.0", []byte("old"), pl)
	var opened *guard.Policy
	openPlain = func(p *guard.Policy) plain { opened = p; return pl }

	if _, code := runJSON(t, "update"); code != 0 {
		t.Fatalf("the run must answer: exit %d", code)
	}
	if opened == nil {
		t.Fatal("the run opened no policy")
	}
	judge := func(method, rawURL string) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			t.Fatal(err)
		}
		return opened.Judge(method, u, nil)
	}
	if err := judge("GET", "https://api.github.com/repos/nhusnullin/gdoc/releases"); err != nil {
		t.Errorf("the listing is the one call the run makes: %v", err)
	}
	for _, refused := range [][2]string{
		{"GET", "https://api.github.com/repos/someone/else/releases"},
		{"POST", "https://api.github.com/repos/nhusnullin/gdoc/releases"},
		{"GET", "https://docs.googleapis.com/v1/documents/1AbCdEfGhIjKlMnOpQrStUvWxYz012345"},
		{"GET", "https://www.googleapis.com/drive/v3/files/1AbCdEfGhIjKlMnOpQrStUvWxYz012345"},
	} {
		if err := judge(refused[0], refused[1]); err == nil {
			t.Errorf("%s %s must be refused by the update's policy", refused[0], refused[1])
		}
	}
}

// Nothing checks for an update unasked. The table is the whole of it: one
// entry names cmdUpdate, and no other file in this package reaches the
// updater, so no run of read, publish or restyle can end up at GitHub.
//
// It reads the syntax rather than the text, because doc.go says all this in
// prose and a prose sentence naming the updater is not a call to it.
func TestNothingChecksForUpdatesUnasked(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		seen++
		imported := false
		for _, imp := range file.Imports {
			if imp.Path.Value == `"gdoc/internal/update"` {
				imported = true
			}
		}
		named := 0
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == "cmdUpdate" {
				named++
			}
			return true
		})
		switch name {
		case "commands.go":
			if named != 1 {
				t.Errorf("the table names cmdUpdate once, and names it %d times", named)
			}
			if imported {
				t.Error("the table describes the command and never runs the updater")
			}
		case "update.go":
			if !imported {
				t.Error("update.go is where the updater is reached, and it does not import it")
			}
		default:
			if imported || named > 0 {
				t.Errorf("%s reaches the updater, and only the update command may", name)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no production Go files were read, so this test is measuring nothing")
	}
}

// TestTheZipBinaryNameFollowsThePlatform states the name the updater looks for
// at the top of a zip, as a literal per platform.
//
// release.yml packs the binary as `gdoc` on every platform except Windows,
// where it packs `gdoc.exe`. The updater matches that name exactly and never
// searches, so a single name would download and verify a whole Windows zip and
// then fail saying it holds no gdoc. Windows is not in release/platforms yet,
// so nothing reaches this today; the line that adds it is what makes it live,
// and that line should not also have to find this.
//
// The names are written out here rather than read out of release.yml, so this
// test does not follow the workflow wherever somebody moves it. The other side
// of the pair, that release.yml really packs those names, is
// TestTheZipCarriesWhatTheInstallerLooksFor in go/boundary.
func TestTheZipBinaryNameFollowsThePlatform(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin":  "gdoc",
		"linux":   "gdoc",
		"windows": "gdoc.exe",
	} {
		if got := zipBinaryName(goos); got != want {
			t.Errorf("on %s the zip holds %q, and the updater looks for %q", goos, want, got)
		}
	}
}
