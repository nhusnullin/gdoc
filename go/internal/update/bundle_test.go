package update

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The bundle the release packs, and where it sits in the zip. Both are
// literals here: a test that read them off the code would follow the code
// wherever somebody moved it.
const (
	appInZip  = "presence/GdocPresence.app"
	appTag    = "v2.1.0"
	appAsset  = "gdoc-v2.1.0-darwin-arm64.zip"
	appBinary = "the universal executable"
)

// entry is one line of a zip a test builds.
type entry struct {
	name string
	body []byte
	mode os.FileMode
}

func folder(name string) entry { return entry{name: name + "/", mode: os.ModeDir | 0o755} }

func plistXML(id, version string) []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleIdentifier</key>
	<string>` + id + `</string>
	<key>CFBundleShortVersionString</key>
	<string>` + version + `</string>
	<key>CFBundleName</key>
	<string>GdocPresence</string>
	<key>LSUIElement</key>
	<true/>
</dict>
</plist>
`)
}

// appEntries is the bundle as the release builds it: a plist, one universal
// executable, and nothing else.
func appEntries(id, version, executable string) []entry {
	return []entry{
		folder(appInZip),
		folder(appInZip + "/Contents"),
		{name: appInZip + "/Contents/Info.plist", body: plistXML(id, version), mode: 0o644},
		folder(appInZip + "/Contents/MacOS"),
		{name: appInZip + "/Contents/MacOS/Presence", body: []byte(executable), mode: 0o755},
	}
}

// releaseEntries is the whole darwin zip: the binary and the installer beside
// the bundle, as the pack step lays them out.
func releaseEntries(id, version, executable string) []entry {
	out := []entry{
		{name: "gdoc", body: []byte("the binary"), mode: 0o755},
		{name: "install.sh", body: []byte("#!/usr/bin/env bash\n"), mode: 0o755},
	}
	return append(out, appEntries(id, version, executable)...)
}

func zipOf(t *testing.T, entries []entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range entries {
		header := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		header.SetMode(e.mode)
		f, err := w.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(e.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// bundled is one release zip and the checksum file it verifies against.
type bundled struct {
	archive []byte
	sums    []byte
	asset   string
}

func zipped(t *testing.T, entries []entry) bundled {
	t.Helper()
	return zippedAs(t, appAsset, entries)
}

func zippedAs(t *testing.T, asset string, entries []entry) bundled {
	t.Helper()
	archive := zipOf(t, entries)
	return bundled{
		archive: archive,
		sums:    sumsFor([2]string{sumOf(archive), asset}),
		asset:   asset,
	}
}

func (b bundled) install(dest string) (BundleResult, error) {
	return ApplyBundle(b.archive, b.sums, b.asset, appInZip, dest)
}

// goodRelease is the zip every happy path starts from.
func goodRelease(t *testing.T) bundled {
	t.Helper()
	return zipped(t, releaseEntries(bundleIdentifier, "2.1.0", appBinary))
}

func destIn(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "GdocPresence.app")
}

// treeOf lists what is under a folder, as paths with / separators, so a test
// can state the whole of a bundle as a literal.
func treeOf(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// writeBundleAt puts a bundle on disk the way an earlier run would have left
// one, so a test can start from a machine that already has the app.
func writeBundleAt(t *testing.T, path, id, version, executable string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(path, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "Contents", "Info.plist"), plistXML(id, version), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "Contents", "MacOS", "Presence"), []byte(executable), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestABundleIsTakenOutOfTheVerifiedZipOnly(t *testing.T) {
	dest := destIn(t)
	good := goodRelease(t)
	changed := bundled{
		archive: append(append([]byte{}, good.archive...), '!'),
		sums:    good.sums,
		asset:   good.asset,
	}

	res, err := changed.install(dest)
	if err == nil {
		t.Fatal("a zip that does not match its checksum installed an app")
	}
	if !strings.Contains(err.Error(), appAsset) {
		t.Errorf("the refusal does not name the asset: %v", err)
	}
	if res.TreeSum != "" || res.Version != "" {
		t.Errorf("a refused zip carries no result: %+v", res)
	}
	mustNotExist(t, dest)
	mustNotExist(t, dest+".new")
	mustNotExist(t, BundlePreviousPath(dest))
}

func TestOnlyEntriesUnderTheBundlePathAreTaken(t *testing.T) {
	dest := destIn(t)
	entries := releaseEntries(bundleIdentifier, "2.1.0", appBinary)
	entries = append(entries,
		entry{name: "presence/README.md", body: []byte("# presence\n"), mode: 0o644},
		entry{name: "presence/GdocPresence.app.bak/Contents/Info.plist", body: plistXML(bundleIdentifier, "2.0.0"), mode: 0o644},
		entry{name: "skills/gdoc-review/SKILL.md", body: []byte("# review\n"), mode: 0o644},
	)

	if _, err := zipped(t, entries).install(dest); err != nil {
		t.Fatalf("the release did not install: %v", err)
	}

	want := []string{"Contents", "Contents/Info.plist", "Contents/MacOS", "Contents/MacOS/Presence"}
	if got := treeOf(t, dest); !equalStrings(got, want) {
		t.Errorf("the app holds %q and the bundle in the zip is %q", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestAPathOutsideTheBundleASymlinkAndAnAppleDoubleEntryAreRefusedByName(t *testing.T) {
	cases := []struct {
		name  string
		added []entry
		says  string
		// why is the words that tell this refusal from the one the next
		// check along would have given for the same entry.
		why string
	}{
		{
			"a path that climbs out of the bundle",
			[]entry{{name: appInZip + "/Contents/../../../evil", body: []byte("no"), mode: 0o644}},
			appInZip + "/Contents/../../../evil",
			"climbs",
		},
		{
			"an absolute path",
			[]entry{{name: "/etc/paths.d/gdoc", body: []byte("no"), mode: 0o644}},
			"/etc/paths.d/gdoc",
			"not a path inside the zip",
		},
		{
			"a symlink under the bundle",
			[]entry{{name: appInZip + "/Contents/MacOS/link", body: []byte("../../../../../../etc/passwd"), mode: os.ModeSymlink | 0o777}},
			appInZip + "/Contents/MacOS/link",
			"symlink",
		},
		{
			"the folder a Mac packs beside what it zipped",
			[]entry{{name: "__MACOSX/presence/GdocPresence.app/Contents/Info.plist", body: []byte("no"), mode: 0o644}},
			"__MACOSX",
			"beside a folder",
		},
		{
			"a resource fork under the bundle",
			[]entry{{name: appInZip + "/Contents/._Info.plist", body: []byte("no"), mode: 0o644}},
			appInZip + "/Contents/._Info.plist",
			"resource fork",
		},
		{
			"the same path twice",
			[]entry{{name: appInZip + "/Contents/Info.plist", body: plistXML(bundleIdentifier, "2.1.0"), mode: 0o644}},
			appInZip + "/Contents/Info.plist",
			"twice",
		},
		{
			"two paths that differ only in case",
			[]entry{{name: appInZip + "/Contents/MacOS/presence", body: []byte("no"), mode: 0o755}},
			appInZip + "/Contents/MacOS/presence",
			"one file on a Mac",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dest := destIn(t)
			entries := append(releaseEntries(bundleIdentifier, "2.1.0", appBinary), c.added...)

			_, err := zipped(t, entries).install(dest)
			if err == nil {
				t.Fatalf("%s installed", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not name %s: %v", c.says, err)
			}
			if c.why != "" && !strings.Contains(err.Error(), c.why) {
				t.Errorf("the refusal does not say %s: %v", c.why, err)
			}
			mustNotExist(t, dest)
			mustNotExist(t, dest+".new")
		})
	}
}

func TestTooManyEntriesOrTooManyBytesAreRefusedByName(t *testing.T) {
	many := releaseEntries(bundleIdentifier, "2.1.0", appBinary)
	for i := 0; i < bundleEntryCeiling; i++ {
		many = append(many, entry{
			name: fmt.Sprintf("%s/Contents/Resources/%d", appInZip, i),
			body: []byte("x"),
			mode: 0o644,
		})
	}
	// A megabyte of nothing deflates to almost nothing, so the zip stays
	// small while what it claims to unpack to does not.
	megabyte := make([]byte, 1<<20)
	heavy := releaseEntries(bundleIdentifier, "2.1.0", appBinary)
	for i := 0; i <= bundleByteCeiling>>20; i++ {
		heavy = append(heavy, entry{
			name: fmt.Sprintf("%s/Contents/Resources/block-%d", appInZip, i),
			body: megabyte,
			mode: 0o644,
		})
	}
	cases := []struct {
		name    string
		entries []entry
		says    string
	}{
		{"more entries than a bundle has", many, "2000 entries"},
		{"more bytes than a bundle holds", heavy, "67108864 bytes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dest := destIn(t)

			_, err := zipped(t, c.entries).install(dest)
			if err == nil {
				t.Fatalf("%s installed", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not name the ceiling %q: %v", c.says, err)
			}
			mustNotExist(t, dest)
			mustNotExist(t, dest+".new")
		})
	}
}

func TestAnotherIdentifierOrAnotherVersionIsRefusedByName(t *testing.T) {
	binaryPlist := append([]byte("bplist00"), 0x00, 0x01, 0x02, 0x03)
	cases := []struct {
		name    string
		entries []entry
		says    string
	}{
		{
			"another app's identifier",
			releaseEntries("com.example.other", "2.1.0", appBinary),
			"com.example.other",
		},
		{
			"a version that is not the release's",
			releaseEntries(bundleIdentifier, "2.0.9", appBinary),
			`"2.0.9"`,
		},
		{
			"a plist that is not XML",
			withPlist(releaseEntries(bundleIdentifier, "2.1.0", appBinary), binaryPlist),
			plistInBundle,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dest := destIn(t)

			_, err := zipped(t, c.entries).install(dest)
			if err == nil {
				t.Fatalf("%s installed", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not name %s: %v", c.says, err)
			}
			mustNotExist(t, dest)
			mustNotExist(t, dest+".new")
		})
	}
}

func withPlist(entries []entry, body []byte) []entry {
	out := make([]entry, len(entries))
	copy(out, entries)
	for i := range out {
		if out[i].name == appInZip+"/"+plistInBundle {
			out[i].body = body
		}
	}
	return out
}

func TestModesComeFromTheZip(t *testing.T) {
	dest := destIn(t)
	entries := releaseEntries(bundleIdentifier, "2.1.0", appBinary)
	entries = append(entries,
		folder(appInZip+"/Contents/Resources"),
		entry{name: appInZip + "/Contents/Resources/owner-only", body: []byte("x"), mode: 0o700},
		entry{name: appInZip + "/Contents/Resources/plain", body: []byte("x"), mode: 0o600},
	)

	if _, err := zipped(t, entries).install(dest); err != nil {
		t.Fatalf("the release did not install: %v", err)
	}

	cases := []struct {
		path string
		mode os.FileMode
	}{
		{"Contents", 0o755},
		{"Contents/MacOS", 0o755},
		{"Contents/MacOS/Presence", 0o755},
		{"Contents/Info.plist", 0o644},
		{"Contents/Resources/owner-only", 0o755},
		{"Contents/Resources/plain", 0o644},
	}
	for _, c := range cases {
		info, err := os.Stat(filepath.Join(dest, filepath.FromSlash(c.path)))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != c.mode {
			t.Errorf("%s is %o and the zip says %o", c.path, got, c.mode)
		}
	}
}

func TestTheStageIsReadBackAndSummed(t *testing.T) {
	dest := destIn(t)

	res, err := goodRelease(t).install(dest)
	if err != nil {
		t.Fatalf("the release did not install: %v", err)
	}

	want := sum([]byte(
		"Contents/Info.plist\x00" + sumOf(plistXML(bundleIdentifier, "2.1.0")) + "\n" +
			"Contents/MacOS/Presence\x00" + sumOf([]byte(appBinary)) + "\n",
	))
	if res.TreeSum != want {
		t.Errorf("the tree sums to %s and the two files in it sum to %s", res.TreeSum, want)
	}
	if res.Version != "2.1.0" {
		t.Errorf("the result says version %q and the plist says 2.1.0", res.Version)
	}
	if res.Previous != "" {
		t.Errorf("there was no app to keep and the result kept %s", res.Previous)
	}
}

func TestAStaleStageIsRemovedFirst(t *testing.T) {
	dest := destIn(t)
	stage := dest + ".new"
	writeBundleAt(t, stage, bundleIdentifier, "1.0.0", "half of some other release")
	if err := os.WriteFile(filepath.Join(stage, "Contents", "leftover"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := goodRelease(t).install(dest); err != nil {
		t.Fatalf("the release did not install over a stale stage: %v", err)
	}

	mustNotExist(t, stage)
	mustNotExist(t, filepath.Join(dest, "Contents", "leftover"))
	if got := readFile(t, filepath.Join(dest, "Contents", "MacOS", "Presence")); got != appBinary {
		t.Errorf("the executable is %q and the zip held %q", got, appBinary)
	}
}

func TestTheSwapKeepsThePreviousBundle(t *testing.T) {
	dest := destIn(t)
	if _, err := goodRelease(t).install(dest); err != nil {
		t.Fatalf("the first release did not install: %v", err)
	}
	next := zippedAs(t, "gdoc-v2.2.0-darwin-arm64.zip", releaseEntries(bundleIdentifier, "2.2.0", "the next executable"))

	res, err := ApplyBundle(next.archive, next.sums, next.asset, appInZip, dest)
	if err != nil {
		t.Fatalf("the second release did not install: %v", err)
	}

	previous := BundlePreviousPath(dest)
	if res.Previous != previous {
		t.Errorf("the result says the old app is at %q and it is at %s", res.Previous, previous)
	}
	if res.Version != "2.2.0" {
		t.Errorf("the result says version %q and the new plist says 2.2.0", res.Version)
	}
	if got := readFile(t, filepath.Join(dest, "Contents", "MacOS", "Presence")); got != "the next executable" {
		t.Errorf("the installed executable is %q", got)
	}
	if got := readFile(t, filepath.Join(previous, "Contents", "MacOS", "Presence")); got != appBinary {
		t.Errorf("the kept executable is %q and the first release held %q", got, appBinary)
	}
	mustNotExist(t, dest+".new")
}

// The second rename is the one that cannot be made to fail from outside
// ApplyBundle, so the swap is called on its own with a stage that is not
// there: that is exactly the state ApplyBundle would be in if the stage went
// away between the read-back and the rename.
func TestAFailedSwapPutsTheOldBundleBack(t *testing.T) {
	dest := destIn(t)
	writeBundleAt(t, dest, bundleIdentifier, "2.0.0", "the app that was there")

	previous, err := swapBundle(dest+".new", dest)
	if err == nil {
		t.Fatal("a swap with nothing staged was reported as installed")
	}
	if !strings.Contains(err.Error(), dest) {
		t.Errorf("the refusal does not name the destination: %v", err)
	}
	if previous != "" {
		t.Errorf("a failed swap kept nothing and said %q", previous)
	}
	if got := readFile(t, filepath.Join(dest, "Contents", "MacOS", "Presence")); got != "the app that was there" {
		t.Errorf("the app at the destination is %q and the old one should be back", got)
	}
	mustNotExist(t, BundlePreviousPath(dest))
}

func TestRollbackOfTheBundleRunTwiceIsWhereItStarted(t *testing.T) {
	dest := destIn(t)
	previous := BundlePreviousPath(dest)
	writeBundleAt(t, dest, bundleIdentifier, "2.1.0", "the new app")
	writeBundleAt(t, previous, bundleIdentifier, "2.0.0", "the earlier app")

	back, err := RollbackBundle(dest)
	if err != nil {
		t.Fatalf("the rollback did not run: %v", err)
	}
	if back.Version != "2.0.0" {
		t.Errorf("the rollback says version %q and the earlier app says 2.0.0", back.Version)
	}
	if back.Previous != previous {
		t.Errorf("the rollback says the app it moved aside is at %q and it is at %s", back.Previous, previous)
	}
	if got := readFile(t, filepath.Join(dest, "Contents", "MacOS", "Presence")); got != "the earlier app" {
		t.Fatalf("the app is %q and the earlier one should be back", got)
	}
	if got := readFile(t, filepath.Join(previous, "Contents", "MacOS", "Presence")); got != "the new app" {
		t.Fatalf("the kept app is %q and it should be the one rolled back from", got)
	}

	again, err := RollbackBundle(dest)
	if err != nil {
		t.Fatalf("the second rollback did not run: %v", err)
	}
	if again.Version != "2.1.0" {
		t.Errorf("the second rollback says version %q and the new app says 2.1.0", again.Version)
	}
	if got := readFile(t, filepath.Join(dest, "Contents", "MacOS", "Presence")); got != "the new app" {
		t.Errorf("the app is %q and two rollbacks are where they started", got)
	}
	if got := readFile(t, filepath.Join(previous, "Contents", "MacOS", "Presence")); got != "the earlier app" {
		t.Errorf("the kept app is %q and two rollbacks are where they started", got)
	}
	mustNotExist(t, dest+".new")
}

func TestRollbackOfTheBundleWithNoPreviousIsRefusedByName(t *testing.T) {
	dest := destIn(t)
	writeBundleAt(t, dest, bundleIdentifier, "2.1.0", "the only app")

	if _, err := RollbackBundle(dest); err == nil {
		t.Fatal("a rollback with nothing to go back to was reported as done")
	} else if !strings.Contains(err.Error(), BundlePreviousPath(dest)) {
		t.Errorf("the refusal does not name the app it looked for: %v", err)
	}
	if got := readFile(t, filepath.Join(dest, "Contents", "MacOS", "Presence")); got != "the only app" {
		t.Errorf("the app is %q and nothing should have moved", got)
	}
}

func TestADestinationThatIsASymlinkAFileOrAnotherAppMovesNothing(t *testing.T) {
	cases := []struct {
		name  string
		setUp func(t *testing.T, dest string)
		says  string
	}{
		{
			"a symlink",
			func(t *testing.T, dest string) {
				elsewhere := filepath.Join(filepath.Dir(dest), "elsewhere")
				writeBundleAt(t, elsewhere, bundleIdentifier, "2.0.0", "somewhere else")
				if err := os.Symlink(elsewhere, dest); err != nil {
					t.Fatal(err)
				}
			},
			"symlink",
		},
		{
			"a file",
			func(t *testing.T, dest string) {
				if err := os.WriteFile(dest, []byte("not a bundle"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			"is a file",
		},
		{
			"another app",
			func(t *testing.T, dest string) {
				writeBundleAt(t, dest, "com.example.other", "2.1.0", "somebody else's app")
			},
			"com.example.other",
		},
		{
			"a folder whose plist does not read",
			func(t *testing.T, dest string) {
				if err := os.MkdirAll(filepath.Join(dest, "Contents"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			plistInBundle,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dest := destIn(t)
			c.setUp(t, dest)
			before := snapshot(t, dest)

			_, err := goodRelease(t).install(dest)
			if err == nil {
				t.Fatalf("%s at the destination was replaced", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not say %s: %v", c.says, err)
			}
			if got := snapshot(t, dest); got != before {
				t.Errorf("the destination is now %q and it was %q", got, before)
			}
			mustNotExist(t, dest+".new")
			mustNotExist(t, BundlePreviousPath(dest))
		})
	}
}

// snapshot is what is at a path, in one string, so a test can say that
// nothing moved without knowing what kind of thing it is.
func snapshot(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(path)
		if err != nil {
			t.Fatal(err)
		}
		return "link to " + target
	case info.IsDir():
		return "folder " + strings.Join(treeOf(t, path), ",")
	default:
		return "file " + readFile(t, path)
	}
}

func TestAMissingApplicationsFolderIsMade(t *testing.T) {
	applications := filepath.Join(t.TempDir(), "Applications")
	dest := filepath.Join(applications, "GdocPresence.app")

	if _, err := goodRelease(t).install(dest); err != nil {
		t.Fatalf("the release did not install into a folder that was not there: %v", err)
	}

	if got := readFile(t, filepath.Join(dest, "Contents", "MacOS", "Presence")); got != appBinary {
		t.Errorf("the executable is %q and the zip held %q", got, appBinary)
	}
}

// The zips above are written by Go. This one was packed by the zip command
// itself, the way the release workflow packs the stage, and it is in the
// repository so that the folder entries, the ordering and the modes a real
// archiver writes are what the installer is held against.
func TestTheBundleComesOutOfAZipPackedByZipItself(t *testing.T) {
	archive, err := os.ReadFile(filepath.Join("testdata", "bundle.zip"))
	if err != nil {
		t.Fatal(err)
	}
	dest := destIn(t)
	packed := bundled{
		archive: archive,
		sums:    sumsFor([2]string{sumOf(archive), appAsset}),
		asset:   appAsset,
	}

	res, err := packed.install(dest)
	if err != nil {
		t.Fatalf("the packed release did not install: %v", err)
	}

	if res.Version != "2.1.0" {
		t.Errorf("the fixture installs version %q and it is stamped 2.1.0", res.Version)
	}
	want := []string{"Contents", "Contents/Info.plist", "Contents/MacOS", "Contents/MacOS/Presence"}
	if got := treeOf(t, dest); !equalStrings(got, want) {
		t.Errorf("the installed app holds %q and the fixture holds %q", got, want)
	}
	info, err := os.Stat(filepath.Join(dest, "Contents", "MacOS", "Presence"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Errorf("the executable is %o and the fixture marked it 755", got)
	}
}

// A plist is read with encoding/xml and nothing else, which is the rule that
// keeps the bundle's plist in the XML format.
func TestOnlyAnXMLPlistReads(t *testing.T) {
	values, err := plistStrings(plistXML(bundleIdentifier, "2.1.0"))
	if err != nil {
		t.Fatalf("the plist the release writes does not read: %v", err)
	}
	if values["CFBundleIdentifier"] != bundleIdentifier || values["CFBundleShortVersionString"] != "2.1.0" {
		t.Errorf("the plist reads as %v", values)
	}
	if _, ok := values["LSUIElement"]; ok {
		t.Errorf("a boolean was read as a string: %v", values)
	}
	if _, err := plistStrings([]byte("bplist00\x00\x01")); err == nil {
		t.Error("a binary plist read as XML")
	}
}

// tagIn is where the version the bundle is checked against comes from, and a
// name that is not a release's is refused rather than guessed at.
func TestTheReleasesTagIsReadOutOfTheAssetName(t *testing.T) {
	tag, err := tagIn(AssetName("v2.11.0", "darwin-arm64"))
	if err != nil {
		t.Fatalf("the asset the release names does not parse: %v", err)
	}
	if tag != "v2.11.0" {
		t.Errorf("the tag read out is %q", tag)
	}
	for _, asset := range []string{"gdoc.zip", "something-else.zip", "gdoc-2.1.0-darwin-arm64.zip"} {
		if _, err := tagIn(asset); err == nil {
			t.Errorf("%s was read as a release's zip", asset)
		}
	}
}
