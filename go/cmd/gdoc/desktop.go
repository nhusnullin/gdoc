// The Claude Desktop extension, and the one program gdoc runs.
//
// Claude Desktop is chat rather than a terminal, so it does not type `gdoc`. It
// starts a command named in an extension, which is a zip holding one
// manifest.json. Every release carries the manifest as a template with two
// placeholders, `@BIN@` and `@VERSION@`; this file fills them with the path the
// binary was just installed at and the release that was installed, writes
// `gdoc.mcpb` beside the binary, and hands that file to Claude Desktop.
//
// Handing it over is one of the two places anything under go/ starts another
// program, and both of them are `/usr/bin/open` on macOS from this file. The
// other is the menu bar app: under the presence setting, a command whose run
// is recorded opens `~/Applications/GdocPresence.app` where that app is
// installed and not already running. The exception is as narrow as it can be,
// one program and two argument lists, and it is held by reading this file's
// syntax tree rather than by trusting this comment: go/boundary's
// TestOnlyDesktopRunsAProgram and
// TestTheOneProgramOpensOnlyTheExtensionAndTheApp.
//
// The extension has no settings. What differs from one machine to the next is
// the binary's path and the release, which is why the template travels in the
// release and is filled here rather than being a file a person edits.

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gdoc/internal/atomicfile"
	"gdoc/internal/config"
)

// The three names the extension is made of. templateInZip is where the release
// workflow packs the template, mcpbName is what the extension is called beside
// the binary, and manifestInMcpb is the one entry inside it. Both installers
// write the same two paths, which go/boundary's
// TestTheThreeRoutesToTheExtensionNameOnePathEach holds, so every route to the
// extension is one path.
const (
	templateInZip  = "mcpb/manifest.json"
	mcpbName       = "gdoc.mcpb"
	manifestInMcpb = "manifest.json"
)

// devVersion is what a build naming no release calls itself in a manifest. A
// manifest version field cannot be empty, so there is a number for the case
// where there is no number. The same number release/install.sh writes.
const devVersion = "0.0.0-dev"

// mcpbMode is what the extension file is left as. It is read by Claude Desktop
// and by nobody else, and it is not run, so it is not executable.
const mcpbMode = 0o644

// macOSOpen is the one program gdoc may run, by its full path. A name on PATH
// would be whatever is first on somebody's PATH today.
const macOSOpen = "/usr/bin/open"

// openInBackground is the one flag gdoc hands the opener: `open -g` starts an
// application and moves no window and no focus. The app has no window to bring
// forward and the person is in a terminal or a chat, so a run that stole the
// front would be the one thing this feature must never do.
const openInBackground = "-g"

// The two ways the one program is started, as words rather than as a bare
// true and false at a call site. andWait is the extension: a person typed
// --desktop and the sentence about what happened is what `open` says. andLeaveIt
// is the app: the exit status is never read, so a shell with no window session
// costs exactly what it cost before.
const (
	andWait    = true
	andLeaveIt = false
)

// presenceLockName is the file the app holds its lock on, inside the presence
// directory beside the month files. The app states the same literal in
// presence/Sources/PresenceApp/Places.swift, and go/boundary's
// TestTheLockNameIsOneWordInTwoLanguages holds the two together.
const presenceLockName = "app.lock"

// desktopOS is the operating system this run is on, behind a variable so a test
// can ask what a Linux run would do from a Mac.
var desktopOS = runtime.GOOS

// runProgram is the seam the one program goes through, so every test in this
// tree measures the argv gdoc would hand it and starts nothing.
var runProgram = startProgram

// startProgram runs one program with the arguments it is handed.
//
// The name it is handed is checked against the one name this tree may run, so
// the narrowness does not rest on the two call sites staying the only two.
//
// wait says what the caller does afterwards. The extension waits, and the
// output is kept for the failure message: `open` says why it could not open a
// file, and that sentence is the whole of what a person needs. The app is
// started and left: a command may not grow the time an application takes to
// come up, and nothing in this process reads that exit status, so there is
// nothing to wait for. A started process is released rather than left as a
// zombie, which is what Process.Release does.
func startProgram(name string, wait bool, args ...string) error {
	if name != macOSOpen {
		return fmt.Errorf("gdoc runs %s and no other program, and this run asked for %s", macOSOpen, name)
	}
	cmd := exec.Command(macOSOpen, args...)
	if !wait {
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("%s %s did not start: %v", macOSOpen, strings.Join(args, " "), err)
		}
		return cmd.Process.Release()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s did not run: %v: %s", macOSOpen, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// openPresenceApp opens the menu bar app where it is installed and not
// running. It is the whole of M25: a person who turned the presence on sees
// what gdoc is doing without remembering to open anything.
//
// Three questions, and all three have to say yes. The platform is macOS,
// because the app is macOS only. The bundle is at the path `gdoc update`
// installs it to, because gdoc opens the app it installed and never goes
// looking for another. And the app does not hold its lock, which is the one
// thing that says whether it is already running: lockIsFree asks the kernel,
// creates nothing and reads no byte.
//
// Every other case does nothing and says nothing. There is no sentence for a
// person here and no field in the envelope: the flag off is today's bytes and
// the flag on is today's bytes too, which is what
// TestTheFlagChangesNoByteOfTheObject measures. A failure to start the opener
// is the same silence, because the command a person is waiting on has nothing
// to do with a menu bar glyph that did not appear.
//
// The caller holds the one gate that is not here: it is called where the run
// is recorded, which is the presence setting and a command that reports.
//
// TestTheOpenerAsksThreeThingsAndRunsOpenGOnce and
// TestACommandUnderTheFlagOpensTheAppOnceAndNothingWhenItIsRunning.
func openPresenceApp() {
	if desktopOS != "darwin" {
		return
	}
	bundle, err := presenceBundle()
	if err != nil {
		return
	}
	if _, err := os.Stat(bundle); err != nil {
		return
	}
	dir, err := config.PresenceDir()
	if err != nil {
		return
	}
	if !lockIsFree(filepath.Join(dir, presenceLockName)) {
		return
	}
	_ = runProgram(macOSOpen, andLeaveIt, openInBackground, bundle)
}

// mcpbPath is the extension file beside the binary at path. Beside it, because
// the manifest names that binary by its absolute path: the two belong together,
// and a person who moves one finds the other.
func mcpbPath(binary string) string {
	return filepath.Join(filepath.Dir(binary), mcpbName)
}

// fillManifest is the template with this machine's path and this release's
// version in it.
//
// Two refusals. A path carrying a quote or a backslash would end the JSON
// string it is written into, so it is refused by name rather than written into
// a manifest Claude Desktop cannot read. And what comes out is parsed before it
// is written: a template this gdoc half understands never reaches Claude
// Desktop, which would otherwise report a connector that failed for no reason
// a person can see.
//
// TestDesktopWritesTheMcpbFromTheZipsTemplate and
// TestAPathThatWouldBreakTheManifestIsRefusedByName.
func fillManifest(template []byte, binary, version string) ([]byte, error) {
	if i := strings.IndexAny(binary, `"\`); i >= 0 {
		return nil, fmt.Errorf("%s carries a %q, and a manifest naming it would not be JSON Claude Desktop can read: install gdoc at a path without one, or drop --desktop", binary, binary[i])
	}
	number := strings.TrimPrefix(version, "v")
	if number == "" {
		number = devVersion
	}
	filled := strings.NewReplacer("@BIN@", binary, "@VERSION@", number).Replace(string(template))
	if !json.Valid([]byte(filled)) {
		return nil, fmt.Errorf("%s out of the release, filled with %s, is not JSON Claude Desktop could read, so nothing was written", templateInZip, binary)
	}
	return []byte(filled), nil
}

// writeMcpb writes the extension: a zip holding the one manifest, replaced
// rather than added to, so a second run leaves one manifest in the file instead
// of a second copy beside the first.
func writeMcpb(path string, manifest []byte) error {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create(manifestInMcpb)
	if err != nil {
		return fmt.Errorf("the extension's %s could not be packed: %w", manifestInMcpb, err)
	}
	if _, err := f.Write(manifest); err != nil {
		return fmt.Errorf("the extension's %s could not be packed: %w", manifestInMcpb, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("the extension could not be packed: %w", err)
	}
	return atomicfile.Replace(path, buf.Bytes(), mcpbMode)
}

// manifestInFile reads the manifest out of the extension a person already has.
// A file that is not there, is not a zip, or holds no manifest is no manifest,
// which is what the hint below reads as nothing to say.
func manifestInFile(path string) ([]byte, bool) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, false
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != manifestInMcpb {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, false
		}
		defer rc.Close()
		b, err := io.ReadAll(rc)
		if err != nil {
			return nil, false
		}
		return b, true
	}
	return nil, false
}

// extensionChanged says whether the release's template, filled for this
// machine, is something other than the extension that is already beside the
// binary. It is what a plain update's one line about Claude Desktop rests on.
//
// The version field is left out of both sides, so a release that changed its
// number and nothing else asks nobody to do anything. Anything this cannot read
// is no change: a plain update was not asked about Claude Desktop, and guessing
// would send a person after a file that is none of this run's business.
//
// TestPlainUpdateHintsWhenTheTemplateChanged and
// TestPlainUpdateIsSilentWithNoMcpbOrNoChange.
func extensionChanged(template []byte, binary, version string) bool {
	have, ok := manifestInFile(mcpbPath(binary))
	if !ok {
		return false
	}
	want, err := fillManifest(template, binary, version)
	if err != nil {
		return false
	}
	a, aOK := withoutVersion(have)
	b, bOK := withoutVersion(want)
	return aOK && bOK && a != b
}

// withoutVersion is a manifest as one canonical string with its version field
// removed. Canonical because Go writes a map's keys in order, so two manifests
// that say the same thing in a different order are the same string here.
func withoutVersion(manifest []byte) (string, bool) {
	var into map[string]any
	if err := json.Unmarshal(manifest, &into); err != nil {
		return "", false
	}
	delete(into, "version")
	out, err := json.Marshal(into)
	if err != nil {
		return "", false
	}
	return string(out), true
}

// hintLine is the one line a plain update ends with when the release's
// extension differs from the one on this machine. It says what changed and the
// command that refreshes it, and this run writes nothing and starts nothing.
func hintLine() string {
	return "the Claude Desktop extension changed in this release, and this run left the one on this machine alone: run gdoc update --desktop to refresh it, then quit Claude Desktop and open it again"
}
