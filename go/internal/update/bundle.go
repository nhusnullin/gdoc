package update

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// bundleIdentifier is the one bundle gdoc installs and the one bundle it
// replaces. It is a literal here rather than a value read off the zip,
// because the whole of the identity check is that the folder a release calls
// an app and the folder already in ~/Applications are both this one.
const bundleIdentifier = "com.altery.gdoc.presence"

// plistInBundle is where a macOS bundle keeps the file that says what it is.
const plistInBundle = "Contents/Info.plist"

// The ceilings. A release's bundle is a plist and one universal executable,
// around 30 entries and a few megabytes, so both of these are far above what
// a real one holds and far below what a zip bomb needs. An archive past
// either is refused by name before anything is written.
const (
	bundleEntryCeiling = 2_000
	bundleByteCeiling  = 64 << 20
)

// The modes a staged bundle is left at. The zip decides which of the two a
// file gets, and the rule is one bit: anything the archive marked executable
// is executable, everything else is not.
const (
	bundleExecMode   = 0o755
	bundleFileMode   = 0o644
	bundleFolderMode = 0o755
)

// BundleResult is what an installed, or rolled back, app bundle left on disk.
type BundleResult struct {
	// Version is CFBundleShortVersionString as the bundle's own plist spells
	// it, read off the file rather than off the release, so it is what the
	// app will say about itself.
	Version string
	// TreeSum is the SHA-256 of the whole folder: one line per file, the path
	// under the bundle, a NUL, the file's own SHA-256, sorted by path. A
	// bundle is a tree and not a file, so there is no single file to hash,
	// and the tree sum is the fact `presence_sha256` is read from.
	TreeSum string
	// Previous is where the bundle that was there went, and empty when there
	// was nothing there to keep.
	Previous string
}

// BundlePreviousPath is where ApplyBundle puts the bundle it replaced, and
// where RollbackBundle looks for it.
func BundlePreviousPath(dest string) string { return dest + ".previous" }

// BundleVersion is what the app at a path says it is, out of its own
// Info.plist, and an error where there is no app there or its plist does not
// read.
//
// It is here rather than in the command because this package is the one room
// that knows what a plist is, and the caller asks one question of it: whether
// the app on this machine is the app the release holds. A run that cannot
// tell installs, and ApplyBundle then refuses anything at the destination
// that is not this app.
//
// TestTheVersionOfAnInstalledBundleIsReadOffItsPlist.
func BundleVersion(bundle string) (string, error) { return versionAt(bundle) }

// ApplyBundle verifies the zip, takes the app bundle out of it and puts it at
// dest, keeping whatever was there as BundlePreviousPath(dest).
//
// It is the folder case of Apply, and it is a second function rather than a
// flag because almost nothing about it is the same. A folder cannot be
// written through internal/atomicfile, which replaces one file's contents; a
// zip holding a tree can hold a path that climbs out of it, a symlink, a
// second entry for the same file and a resource fork, none of which a single
// named file can be; and rename will not replace a folder, so the bundle that
// was there has to be cleared out of the way rather than overwritten.
//
// The order is the safety, as it is in Apply. Nothing on disk moves until the
// zip has matched its checksum, every entry has been judged by name and mode,
// and the bundle's own plist has said it is this app at this release's
// version. Then the tree is written beside the destination as <dest>.new,
// read back off the disk and summed, and only then swapped in by rename.
//
// The binary and the app are installed separately and on purpose. gdoc never
// undoes a binary that landed because an app did not, so the two can sit at
// different versions, and the app is the one that says so.
//
// TestABundleIsTakenOutOfTheVerifiedZipOnly,
// TestOnlyEntriesUnderTheBundlePathAreTaken,
// TestAPathOutsideTheBundleASymlinkAndAnAppleDoubleEntryAreRefusedByName,
// TestTooManyEntriesOrTooManyBytesAreRefusedByName,
// TestAnotherIdentifierOrAnotherVersionIsRefusedByName, TestModesComeFromTheZip,
// TestTheStageIsReadBackAndSummed, TestAStaleStageIsRemovedFirst,
// TestTheSwapKeepsThePreviousBundle, TestAFailedSwapPutsTheOldBundleBack,
// TestADestinationThatIsASymlinkAFileOrAnotherAppMovesNothing and
// TestAMissingApplicationsFolderIsMade.
func ApplyBundle(archive, sums []byte, asset, inZip, dest string) (BundleResult, error) {
	if err := Verify(archive, sums, asset); err != nil {
		return BundleResult{}, err
	}
	tag, err := tagIn(asset)
	if err != nil {
		return BundleResult{}, err
	}
	entries, err := bundleEntries(archive, asset, inZip)
	if err != nil {
		return BundleResult{}, err
	}
	version, err := bundleIdentity(entries, asset, tag)
	if err != nil {
		return BundleResult{}, err
	}
	if err := destinationIsOurs(dest); err != nil {
		return BundleResult{}, err
	}
	folder := filepath.Dir(dest)
	if err := os.MkdirAll(folder, bundleFolderMode); err != nil {
		return BundleResult{}, fmt.Errorf("the folder %s could not be made: %w", folder, err)
	}
	stage := dest + ".new"
	// A stage left by a run that died is a folder rename would refuse to
	// replace, and its contents are half of some other release.
	if err := os.RemoveAll(stage); err != nil {
		return BundleResult{}, fmt.Errorf("the stage left at %s by an earlier run could not be cleared: %w", stage, err)
	}
	want, err := writeBundle(entries, stage, asset)
	if err != nil {
		os.RemoveAll(stage)
		return BundleResult{}, err
	}
	got, err := sumTree(stage)
	if err != nil {
		os.RemoveAll(stage)
		return BundleResult{}, err
	}
	if got != want {
		os.RemoveAll(stage)
		return BundleResult{}, fmt.Errorf("the bundle staged at %s sums to %s and the zip held %s, so what was written is not what was verified", stage, got, want)
	}
	previous, err := swapBundle(stage, dest)
	if err != nil {
		return BundleResult{}, err
	}
	return BundleResult{Version: version, TreeSum: got, Previous: previous}, nil
}

// RollbackBundle swaps the two bundles: what is at dest becomes the previous
// one, and the previous one becomes the app a person opens. It is a swap
// rather than a move, so a rollback taken by mistake is one more rollback
// away from where it started.
//
// Unlike the binary's Rollback, this one reports the version it put back.
// Reading a binary's version would mean running it; reading a bundle's means
// reading its plist, which is a file.
//
// TestRollbackOfTheBundleRunTwiceIsWhereItStarted.
func RollbackBundle(dest string) (BundleResult, error) {
	previous := BundlePreviousPath(dest)
	if _, err := os.Lstat(previous); err != nil {
		return BundleResult{}, fmt.Errorf("there is no %s, so there is no earlier app to go back to", previous)
	}
	held := dest + ".new"
	if err := os.RemoveAll(held); err != nil {
		return BundleResult{}, fmt.Errorf("the stage left at %s by an earlier run could not be cleared: %w", held, err)
	}
	swap := false
	if _, err := os.Lstat(dest); err == nil {
		if err := os.Rename(dest, held); err != nil {
			return BundleResult{}, fmt.Errorf("the app at %s could not be moved aside: %w", dest, err)
		}
		swap = true
	}
	if err := os.Rename(previous, dest); err != nil {
		if swap {
			os.Rename(held, dest)
		}
		return BundleResult{}, fmt.Errorf("the earlier app could not be moved to %s: %w", dest, err)
	}
	out := BundleResult{}
	if swap {
		if err := os.Rename(held, previous); err != nil {
			return BundleResult{}, fmt.Errorf("the app that was rolled back could not be kept at %s, so it is still at %s: %w", previous, held, err)
		}
		out.Previous = previous
	}
	// The swap is done by here, and undoing it would take back the rollback a
	// person asked for. Both reads below say where things ended up instead of
	// moving a third time.
	version, err := versionAt(dest)
	if err != nil {
		return BundleResult{}, fmt.Errorf("%w; the swap had already happened, so %s is the app now", err, dest)
	}
	got, err := sumTree(dest)
	if err != nil {
		return BundleResult{}, fmt.Errorf("%w; the swap had already happened, so %s is the app now", err, dest)
	}
	out.Version = version
	out.TreeSum = got
	return out, nil
}

// swapBundle is the two renames that put the staged bundle in place, and the
// clearing that has to happen before the first of them: rename will not
// replace a folder, so the bundle kept from the run before last goes before
// the one kept now can take its name. A failure in the second rename undoes
// the first.
//
// TestTheSwapKeepsThePreviousBundle and TestAFailedSwapPutsTheOldBundleBack.
func swapBundle(stage, dest string) (string, error) {
	previous := BundlePreviousPath(dest)
	kept := false
	if _, err := os.Lstat(dest); err == nil {
		if err := os.RemoveAll(previous); err != nil {
			os.RemoveAll(stage)
			return "", fmt.Errorf("the app kept at %s could not be cleared to make room: %w", previous, err)
		}
		if err := os.Rename(dest, previous); err != nil {
			os.RemoveAll(stage)
			return "", fmt.Errorf("the app at %s could not be moved aside: %w", dest, err)
		}
		kept = true
	}
	if err := os.Rename(stage, dest); err != nil {
		if kept {
			os.Rename(previous, dest)
		}
		os.RemoveAll(stage)
		return "", fmt.Errorf("the new app could not be moved to %s: %w", dest, err)
	}
	if kept {
		return previous, nil
	}
	return "", nil
}

// destinationIsOurs looks at what is already at dest and refuses anything
// that is not a bundle of this app. Nothing has moved when it raises.
//
// A symlink is refused because the rename would replace the link and leave
// whatever it pointed at; a file is refused because a bundle is a folder; a
// folder whose plist does not read, or names somebody else, is refused
// because not knowing never resolves to overwrite. A person who put something
// else there removes it, and the next run installs.
//
// TestADestinationThatIsASymlinkAFileOrAnotherAppMovesNothing.
func destinationIsOurs(dest string) error {
	info, err := os.Lstat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("%s could not be looked at, so nothing was moved: %w", dest, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink, and gdoc replaces an app it can see rather than one a link points at, so nothing was moved", dest)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is a file and an app is a folder, so nothing was moved", dest)
	}
	id, err := identifierAt(dest)
	if err != nil {
		return fmt.Errorf("%w, so gdoc cannot tell whose app is at %s and moved nothing", err, dest)
	}
	if id != bundleIdentifier {
		return fmt.Errorf("the app at %s says it is %s and gdoc replaces only %s, so nothing was moved", dest, id, bundleIdentifier)
	}
	return nil
}

// bundleEntry is one entry of the zip that belongs to the bundle, and where
// it goes under it.
type bundleEntry struct {
	rel  string
	file *zip.File
}

// bundleEntries judges every entry of the zip and returns the ones the bundle
// is made of, in the order the archive lists them.
//
// Two kinds of judgement happen here. A name nothing may ever carry, wherever
// it sits in the archive, is refused outright: an absolute path, a path with
// a .. or . segment, a backslash, a resource fork and the __MACOSX folder a
// macOS zip writes beside what it packed. A release gdoc installs from
// carries none of those, and an archive that does is not one to take an app
// out of at all. Everything else is judged only where it sits under the
// bundle path, and an entry elsewhere in the zip is simply not taken: the
// same archive holds the binary, the installer and the skills.
//
// TestOnlyEntriesUnderTheBundlePathAreTaken,
// TestAPathOutsideTheBundleASymlinkAndAnAppleDoubleEntryAreRefusedByName and
// TestTooManyEntriesOrTooManyBytesAreRefusedByName.
func bundleEntries(archive []byte, asset, inZip string) ([]bundleEntry, error) {
	r, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("%s is not a zip that opens: %w", asset, err)
	}
	prefix := inZip + "/"
	var out []bundleEntry
	byPath := map[string]bool{}
	byFolded := map[string]string{}
	var expanded uint64
	for _, f := range r.File {
		if err := nameIsCarriable(f.Name, asset); err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(f.Name, "/")
		if name == inZip {
			continue
		}
		rel, under := strings.CutPrefix(name, prefix)
		if !under || rel == "" {
			continue
		}
		mode := f.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			return nil, fmt.Errorf("%s holds %s as a symlink, and an app gdoc installs carries none", asset, f.Name)
		case mode.IsDir():
		case !mode.IsRegular():
			return nil, fmt.Errorf("%s holds %s, which is neither a file nor a folder, and an app gdoc installs is made of both and nothing else", asset, f.Name)
		}
		if byPath[rel] {
			return nil, fmt.Errorf("%s holds %s twice, and gdoc cannot tell which of the two the app is made of", asset, f.Name)
		}
		folded := strings.ToLower(rel)
		if first, seen := byFolded[folded]; seen {
			return nil, fmt.Errorf("%s holds both %s and %s, which are one file on a Mac's own disk", asset, prefix+first, f.Name)
		}
		byPath[rel] = true
		byFolded[folded] = rel
		if len(out)+1 > bundleEntryCeiling {
			return nil, fmt.Errorf("%s holds more than %d entries under %s, and the app is a plist and one executable", asset, bundleEntryCeiling, inZip)
		}
		expanded += f.UncompressedSize64
		if expanded > bundleByteCeiling {
			return nil, fmt.Errorf("%s unpacks to more than %d bytes under %s, and the app is a plist and one executable", asset, bundleByteCeiling, inZip)
		}
		out = append(out, bundleEntry{rel: rel, file: f})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s holds nothing under %s, so there is no app in it to take", asset, inZip)
	}
	return out, nil
}

// nameIsCarriable refuses the names no entry of a release's zip ever has.
func nameIsCarriable(name, asset string) error {
	if name == "" {
		return fmt.Errorf("%s holds an entry with no name at all", asset)
	}
	if strings.HasPrefix(name, "/") || filepath.IsAbs(name) || strings.Contains(name, `\`) {
		return fmt.Errorf("%s holds %s, which is not a path inside the zip", asset, name)
	}
	for _, segment := range strings.Split(strings.TrimSuffix(name, "/"), "/") {
		switch {
		case segment == "" || segment == "." || segment == "..":
			return fmt.Errorf("%s holds %s, whose path climbs rather than names a place in the zip", asset, name)
		case segment == "__MACOSX":
			return fmt.Errorf("%s holds %s, and __MACOSX is what a Mac packs beside a folder rather than part of it", asset, name)
		case strings.HasPrefix(segment, "._"):
			return fmt.Errorf("%s holds %s, which is a resource fork and not part of the app", asset, name)
		}
	}
	return nil
}

// bundleIdentity reads the bundle's own plist out of the zip and says whether
// it is this app, at this release's version. It is the last judgement before
// anything is written.
//
// The plist is read strictly, with encoding/xml and nothing else, so a
// bundle whose plist was converted to the binary format is refused rather
// than installed unread. That keeps the format a rule the build has to hold
// rather than a detail it may change.
//
// TestAnotherIdentifierOrAnotherVersionIsRefusedByName.
func bundleIdentity(entries []bundleEntry, asset, tag string) (string, error) {
	var found *zip.File
	for _, e := range entries {
		if e.rel == plistInBundle {
			found = e.file
			break
		}
	}
	if found == nil {
		return "", fmt.Errorf("%s holds an app with no %s, so it does not say what it is", asset, plistInBundle)
	}
	body, err := readEntry(found, asset)
	if err != nil {
		return "", err
	}
	values, err := plistStrings(body)
	if err != nil {
		return "", fmt.Errorf("the %s in %s does not read as the XML a plist is: %w", plistInBundle, asset, err)
	}
	if id := values["CFBundleIdentifier"]; id != bundleIdentifier {
		return "", fmt.Errorf("the app in %s says it is %q and gdoc installs only %s", asset, id, bundleIdentifier)
	}
	version := values["CFBundleShortVersionString"]
	// The plist carries the tag without the v, which is what a version is
	// called on a Mac, and 0.0.0 for a build that came from no release.
	if want := strings.TrimPrefix(tag, "v"); version != want {
		return "", fmt.Errorf("the app in %s is stamped %q and the release is %s, so the two did not come out of the same build", asset, version, tag)
	}
	return version, nil
}

// writeBundle lays the entries out under stage and returns the tree sum of
// what it wrote. Nothing here replaces a file: the stage was just cleared, so
// every path in it is new.
func writeBundle(entries []bundleEntry, stage, asset string) (string, error) {
	var lines []string
	folders := []string{stage}
	for _, e := range entries {
		path := filepath.Join(stage, filepath.FromSlash(e.rel))
		if e.file.Mode().IsDir() {
			if err := os.MkdirAll(path, bundleFolderMode); err != nil {
				return "", fmt.Errorf("the folder %s could not be made: %w", path, err)
			}
			folders = append(folders, path)
			continue
		}
		parent := filepath.Dir(path)
		if err := os.MkdirAll(parent, bundleFolderMode); err != nil {
			return "", fmt.Errorf("the folder %s could not be made: %w", parent, err)
		}
		folders = append(folders, parent)
		body, err := readEntry(e.file, asset)
		if err != nil {
			return "", err
		}
		mode := os.FileMode(bundleFileMode)
		if e.file.Mode()&0o111 != 0 {
			mode = bundleExecMode
		}
		if err := os.WriteFile(path, body, mode); err != nil {
			return "", fmt.Errorf("%s could not be written: %w", path, err)
		}
		// A umask takes bits off what WriteFile and MkdirAll ask for, and a
		// file the zip marked executable has to stay executable whatever a
		// person's umask is.
		if err := os.Chmod(path, mode); err != nil {
			return "", fmt.Errorf("%s could not be left at %o: %w", path, mode, err)
		}
		lines = append(lines, e.rel+"\x00"+sum(body)+"\n")
	}
	for _, folder := range folders {
		if err := os.Chmod(folder, bundleFolderMode); err != nil {
			return "", fmt.Errorf("%s could not be left at %o: %w", folder, os.FileMode(bundleFolderMode), err)
		}
	}
	sort.Strings(lines)
	return sum([]byte(strings.Join(lines, ""))), nil
}

// readEntry reads one entry out of the zip, and no more bytes than its own
// header said it holds. The ceiling in bundleEntries is added up off those
// headers, so an entry that unpacks past the size it declared would walk
// straight through it.
func readEntry(f *zip.File, asset string) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("%s holds %s and it does not open: %w", asset, f.Name, err)
	}
	defer rc.Close()
	body, err := io.ReadAll(io.LimitReader(rc, int64(f.UncompressedSize64)+1))
	if err != nil {
		return nil, fmt.Errorf("%s holds %s and it does not read: %w", asset, f.Name, err)
	}
	if uint64(len(body)) != f.UncompressedSize64 {
		return nil, fmt.Errorf("%s holds %s, which unpacks to a size its own header does not say, so it was not taken", asset, f.Name)
	}
	return body, nil
}

// sumTree reads a staged or installed bundle back off the disk and sums it:
// one line per file, the path under the bundle, a NUL, the file's SHA-256,
// sorted by path, and the SHA-256 of all of that. The NUL is the separator
// because it is the one byte a path cannot hold.
//
// TestTheStageIsReadBackAndSummed.
func sumTree(root string) (string, error) {
	var lines []string
	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s is neither a file nor a folder, and the app gdoc staged is made of both and nothing else", path)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		lines = append(lines, filepath.ToSlash(rel)+"\x00"+sum(body)+"\n")
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		return "", fmt.Errorf("the app at %s could not be read back: %w", root, err)
	}
	sort.Strings(lines)
	return sum([]byte(strings.Join(lines, ""))), nil
}

// identifierAt and versionAt read one value out of the plist of a bundle on
// disk.
func identifierAt(bundle string) (string, error) { return plistValueAt(bundle, "CFBundleIdentifier") }

func versionAt(bundle string) (string, error) {
	return plistValueAt(bundle, "CFBundleShortVersionString")
}

func plistValueAt(bundle, key string) (string, error) {
	path := filepath.Join(bundle, plistInBundle)
	body, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s could not be read: %w", path, err)
	}
	values, err := plistStrings(body)
	if err != nil {
		return "", fmt.Errorf("%s does not read as the XML a plist is: %w", path, err)
	}
	value, ok := values[key]
	if !ok {
		return "", fmt.Errorf("%s names no %s", path, key)
	}
	return value, nil
}

// plistStrings reads an XML plist and returns the string values of the top
// dictionary, by key. Everything else in the file is stepped over whole: a
// plist carries booleans, numbers and arrays too, and the two values gdoc
// reads are both strings.
//
// encoding/xml is strict about what it accepts and resolves no external
// entity, so a file that is not well-formed XML raises here rather than
// decoding to an empty dictionary that would then fail the identity check
// with the wrong words.
func plistStrings(body []byte) (map[string]string, error) {
	d := xml.NewDecoder(bytes.NewReader(body))
	out := map[string]string{}
	depth := 0
	key := ""
	haveKey := false
	for {
		token, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			if end, ok := token.(xml.EndElement); ok && end.Name.Local == "dict" {
				depth--
			}
			continue
		}
		switch {
		case start.Name.Local == "plist":
			// The root, which holds the one dictionary below.
		case start.Name.Local == "dict":
			depth++
			haveKey = false
		case depth != 1:
			if err := d.Skip(); err != nil {
				return nil, err
			}
		case start.Name.Local == "key":
			if err := d.DecodeElement(&key, &start); err != nil {
				return nil, err
			}
			haveKey = true
		case haveKey && start.Name.Local == "string":
			var value string
			if err := d.DecodeElement(&value, &start); err != nil {
				return nil, err
			}
			out[key] = value
			haveKey = false
		default:
			if err := d.Skip(); err != nil {
				return nil, err
			}
			haveKey = false
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("it holds no key with a string value, so it is not a plist that says anything")
	}
	return out, nil
}

// tagIn reads the release's tag back out of the asset name. AssetName writes
// gdoc- then the tag then - then the platform, and a tag carries no dash, so
// the field between the first two dashes is the whole of it. The tag is what
// the bundle's stamp is checked against, which is why the check needs no
// argument of its own: the asset is already the release's name.
func tagIn(asset string) (string, error) {
	rest, ok := strings.CutPrefix(asset, "gdoc-")
	if !ok {
		return "", fmt.Errorf("%s is not named the way a release names its zips, so gdoc cannot tell which release it is", asset)
	}
	tag, _, ok := strings.Cut(rest, "-")
	if !ok {
		return "", fmt.Errorf("%s is not named the way a release names its zips, so gdoc cannot tell which release it is", asset)
	}
	if _, err := Parse(tag); err != nil {
		return "", fmt.Errorf("%s names %q where a release's tag goes: %w", asset, tag, err)
	}
	return tag, nil
}
