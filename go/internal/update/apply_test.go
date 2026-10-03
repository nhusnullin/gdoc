package update

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// zipHolding builds a release zip the way the release workflow packs one: the
// binary at the top, and the files beside it that the installer reads.
func zipHolding(t *testing.T, binary string, content []byte) []byte {
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
	if binary != "" {
		add(binary, content, 0o755)
	}
	add("install.sh", []byte("#!/usr/bin/env bash\n"), 0o755)
	add("skills/gdoc-review/SKILL.md", []byte("# review\n"), 0o644)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// sumsFor is SHA256SUMS-<tag> as sha256sum writes it: the hash, two spaces,
// the file name, one line per zip in the release.
func sumsFor(pairs ...[2]string) []byte {
	var b strings.Builder
	for _, p := range pairs {
		fmt.Fprintf(&b, "%s  %s\n", p[0], p[1])
	}
	return []byte(b.String())
}

func sumOf(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func fileSum(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sumOf(b)
}

// packed is one zip, its checksum file and the names both are known by.
type packed struct {
	archive []byte
	sums    []byte
	asset   string
	binary  string
}

func newRelease(t *testing.T, content string) packed {
	t.Helper()
	asset := AssetName("v2.1.0", "darwin-arm64")
	archive := zipHolding(t, "gdoc", []byte(content))
	return packed{
		archive: archive,
		sums: sumsFor(
			[2]string{sumOf([]byte("another platform")), AssetName("v2.1.0", "darwin-amd64")},
			[2]string{sumOf(archive), asset},
		),
		asset:  asset,
		binary: "gdoc",
	}
}

func (r packed) apply(path string) (Result, error) {
	return Apply(r.archive, r.sums, r.asset, r.binary, path)
}

func mustNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s is there and nothing should have written it", path)
	}
}

func TestAZipVerifiesAgainstItsOwnLineOfTheChecksumFile(t *testing.T) {
	r := newRelease(t, "the new binary")

	if err := Verify(r.archive, r.sums, r.asset); err != nil {
		t.Fatalf("the zip that was hashed into the file does not verify: %v", err)
	}
}

func TestVerifyRefusesByNameWhatDoesNotMatch(t *testing.T) {
	r := newRelease(t, "the new binary")
	cases := []struct {
		name    string
		archive []byte
		sums    []byte
		asset   string
		says    string
	}{
		{"a byte changed in the zip", append(append([]byte{}, r.archive...), '!'), r.sums, r.asset, r.asset},
		{"a download cut short", r.archive[:len(r.archive)/2], r.sums, r.asset, r.asset},
		{"no line for this asset", r.archive, r.sums, AssetName("v2.1.0", "windows-amd64"), "windows-amd64"},
		{"an empty checksum file", r.archive, nil, r.asset, r.asset},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Verify(c.archive, c.sums, c.asset)
			if err == nil {
				t.Fatal("it verified and it should not have")
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not name %q: %v", c.says, err)
			}
		})
	}
}

func TestTheReplaceSequenceLeavesTheNewBinaryAndKeepsTheOld(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gdoc")
	if err := os.WriteFile(path, []byte("the old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := newRelease(t, "the new binary")

	got, err := r.apply(path)
	if err != nil {
		t.Fatalf("the update did not apply: %v", err)
	}

	if body, err := os.ReadFile(path); err != nil || string(body) != "the new binary" {
		t.Fatalf("the file at %s is %q, %v", path, body, err)
	}
	previous := path + ".previous"
	if body, err := os.ReadFile(previous); err != nil || string(body) != "the old binary" {
		t.Fatalf("the file at %s is %q, %v", previous, body, err)
	}
	if got.Previous != previous {
		t.Errorf("the result names %q as the previous binary and it is at %q", got.Previous, previous)
	}
	if got.Sum != fileSum(t, path) {
		t.Errorf("the result says %q and the file at its final path hashes to %q", got.Sum, fileSum(t, path))
	}
	mustNotExist(t, path+".new")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o755 {
			t.Errorf("the new binary is %v and a binary is 0755", info.Mode().Perm())
		}
	}
}

func TestASecondUpdateOverwritesThePrevious(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gdoc")
	if err := os.WriteFile(path, []byte("the first binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := newRelease(t, "the second binary").apply(path); err != nil {
		t.Fatal(err)
	}

	if _, err := newRelease(t, "the third binary").apply(path); err != nil {
		t.Fatal(err)
	}

	if body, err := os.ReadFile(path); err != nil || string(body) != "the third binary" {
		t.Fatalf("the file at %s is %q, %v", path, body, err)
	}
	if body, err := os.ReadFile(path + ".previous"); err != nil || string(body) != "the second binary" {
		t.Fatalf("the previous binary is %q, %v, and one update back is the second", body, err)
	}
}

func TestApplyInstallsWhereThereWasNoBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gdoc")

	got, err := newRelease(t, "the new binary").apply(path)
	if err != nil {
		t.Fatalf("the update did not apply: %v", err)
	}

	if body, err := os.ReadFile(path); err != nil || string(body) != "the new binary" {
		t.Fatalf("the file at %s is %q, %v", path, body, err)
	}
	if got.Previous != "" {
		t.Errorf("there was nothing to keep and the result names %q", got.Previous)
	}
	mustNotExist(t, path+".previous")
}

func TestAWrongChecksumReplacesNothing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gdoc")
	if err := os.WriteFile(path, []byte("the old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := newRelease(t, "the new binary")
	r.archive = append(r.archive, '!')

	if _, err := r.apply(path); err == nil {
		t.Fatal("a zip that does not match its checksum was applied")
	}

	if body, err := os.ReadFile(path); err != nil || string(body) != "the old binary" {
		t.Fatalf("the file at %s is %q, %v, and it should not have been touched", path, body, err)
	}
	mustNotExist(t, path+".previous")
	mustNotExist(t, path+".new")
}

// A read-back that cannot be read is the other post-rename failure, and it has
// to put the old binary back the way the wrong-hash branch does. Apply cannot
// be made to hit it on purpose, so the tail is called directly with a path
// that holds nothing: that is exactly the state Apply would be in if the file
// went away between the rename and the read.
func TestAnUnreadableReadBackPutsTheOldBinaryBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gdoc")
	previous := PreviousPath(path)
	if err := os.WriteFile(previous, []byte("the old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := landed(path, previous, true, sum([]byte("the new binary")))
	if err == nil {
		t.Fatal("a binary that could not be read back was reported as installed")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the refusal does not name the file it could not read: %v", err)
	}
	if res.Sum != "" || res.Previous != "" {
		t.Errorf("a failed read-back carries no result: %+v", res)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "the old binary" {
		t.Fatalf("the file at %s is %q, %v, and the old binary should be back", path, body, err)
	}
	mustNotExist(t, previous)
}

func TestAZipWithNoBinaryInItIsRefusedByName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gdoc")
	if err := os.WriteFile(path, []byte("the old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	archive := zipHolding(t, "", nil)
	asset := AssetName("v2.1.0", "darwin-arm64")
	sums := sumsFor([2]string{sumOf(archive), asset})

	_, err := Apply(archive, sums, asset, "gdoc", path)
	if err == nil {
		t.Fatal("a zip with no binary in it was applied")
	}
	if !strings.Contains(err.Error(), "gdoc") || !strings.Contains(err.Error(), asset) {
		t.Errorf("the refusal names neither the binary nor the zip: %v", err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "the old binary" {
		t.Fatalf("the file at %s is %q, %v, and it should not have been touched", path, body, err)
	}
	mustNotExist(t, path+".new")
}

func TestRollbackSwapsTheTwoBinariesBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gdoc")
	if err := os.WriteFile(path, []byte("the old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := newRelease(t, "the new binary").apply(path); err != nil {
		t.Fatal(err)
	}

	got, err := Rollback(path)
	if err != nil {
		t.Fatalf("the rollback did not run: %v", err)
	}

	if body, err := os.ReadFile(path); err != nil || string(body) != "the old binary" {
		t.Fatalf("the file at %s is %q, %v", path, body, err)
	}
	if body, err := os.ReadFile(path + ".previous"); err != nil || string(body) != "the new binary" {
		t.Fatalf("the previous binary is %q, %v, and a rollback is a swap", body, err)
	}
	if got.Sum != fileSum(t, path) {
		t.Errorf("the result says %q and the file at its final path hashes to %q", got.Sum, fileSum(t, path))
	}
	if got.Previous != path+".previous" {
		t.Errorf("the result names %q as the previous binary", got.Previous)
	}
}

func TestARollbackRunTwiceIsWhereItStarted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gdoc")
	if err := os.WriteFile(path, []byte("the old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := newRelease(t, "the new binary").apply(path); err != nil {
		t.Fatal(err)
	}
	if _, err := Rollback(path); err != nil {
		t.Fatal(err)
	}

	if _, err := Rollback(path); err != nil {
		t.Fatalf("the second rollback did not run: %v", err)
	}

	if body, err := os.ReadFile(path); err != nil || string(body) != "the new binary" {
		t.Fatalf("the file at %s is %q, %v", path, body, err)
	}
}

func TestRollbackRefusesWhenThereIsNoPrevious(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gdoc")
	if err := os.WriteFile(path, []byte("the only binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Rollback(path)
	if err == nil {
		t.Fatal("a rollback ran with nothing to roll back to")
	}
	if !strings.Contains(err.Error(), path+".previous") {
		t.Errorf("the refusal does not name the file it looked for: %v", err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "the only binary" {
		t.Fatalf("the file at %s is %q, %v, and it should not have been touched", path, body, err)
	}
}

// A rollback that cannot keep what it rolled back from leaves that binary on
// disk and names it. The swap has already happened by then, so removing the
// file would destroy the only copy of the release a person just left.
func TestAFailedKeepLeavesTheRolledBackBinaryOnDisk(t *testing.T) {
	dir := t.TempDir()
	held := filepath.Join(dir, "gdoc.new")
	previous := filepath.Join(dir, "gdoc.previous")
	if err := os.WriteFile(held, []byte("the new binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(previous, 0o755); err != nil {
		t.Fatal(err)
	}

	err := keepRolledBack(held, previous)
	if err == nil {
		t.Fatal("a rename onto a directory was reported as a kept binary")
	}
	if !strings.Contains(err.Error(), held) {
		t.Errorf("the failure does not name where the binary is: %v", err)
	}
	if body, err := os.ReadFile(held); err != nil || string(body) != "the new binary" {
		t.Fatalf("the file at %s is %q, %v, and it is the only copy there is", held, body, err)
	}
}

// A read-back that fails with nothing to put back leaves nothing at path. The
// file there is the one that just failed verification, and leaving it is
// leaving an unverified binary for the next run to execute.
func TestAFailedReadBackWithNothingToRestoreLeavesNoBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gdoc")
	if err := os.WriteFile(path, []byte("not what the zip held"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := landed(path, PreviousPath(path), false, sum([]byte("the new binary")))
	if err == nil {
		t.Fatal("a binary that hashes wrong was reported as installed")
	}
	if res.Sum != "" || res.Previous != "" {
		t.Errorf("a failed read-back carries no result: %+v", res)
	}
	mustNotExist(t, path)
}

// FileFrom is the read beside Apply: one named file out of the same verified
// zip. `gdoc update --desktop` takes the Claude Desktop manifest template that
// way, so the template is always the one the release it installed carries, and
// it is read from bytes that have already matched the published checksum.
func TestFileFromTakesOneFileOutOfTheVerifiedZip(t *testing.T) {
	archive := zipHolding(t, "gdoc", []byte("the new binary"))
	asset := AssetName("v2.9.0", "darwin-arm64")
	sums := sumsFor([2]string{sumOf(archive), asset})

	got, err := FileFrom(archive, sums, asset, "skills/gdoc-review/SKILL.md")
	if err != nil {
		t.Fatalf("the zip holds that file: %v", err)
	}
	if string(got) != "# review\n" {
		t.Errorf("the file read out of the zip is %q", got)
	}
}

// A zip whose bytes do not match the release's checksum is refused before
// anything is read out of it, the way Apply refuses one before anything moves.
func TestFileFromRefusesAZipThatDoesNotMatchItsChecksum(t *testing.T) {
	archive := zipHolding(t, "gdoc", []byte("the new binary"))
	asset := AssetName("v2.9.0", "darwin-arm64")
	sums := sumsFor([2]string{sumOf([]byte("some other bytes")), asset})

	if _, err := FileFrom(archive, sums, asset, "install.sh"); err == nil {
		t.Fatal("a zip that does not match its checksum was read")
	}
}

// A file the zip does not hold is refused naming both the zip and the name,
// because the caller asked for one path and nothing else will do.
func TestFileFromRefusesAFileTheZipDoesNotHoldByName(t *testing.T) {
	archive := zipHolding(t, "gdoc", []byte("the new binary"))
	asset := AssetName("v2.9.0", "darwin-arm64")
	sums := sumsFor([2]string{sumOf(archive), asset})

	_, err := FileFrom(archive, sums, asset, "mcpb/manifest.json")
	if err == nil {
		t.Fatal("a file the zip does not hold was read out of it")
	}
	if !strings.Contains(err.Error(), "mcpb/manifest.json") || !strings.Contains(err.Error(), asset) {
		t.Errorf("the refusal names neither the file nor the zip: %v", err)
	}
}
