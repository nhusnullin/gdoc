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
//
// # --desktop writes the Claude Desktop extension, and is the one program gdoc runs
//
// Claude Desktop is chat rather than a terminal, so it does not type `gdoc`. It
// starts a command named in an extension, which is a zip holding one
// manifest.json, and every release from v2.9.0 on carries that manifest as a
// template with two placeholders. `gdoc update --desktop` fills them with the
// path the binary sits at and the release that was installed, writes
// `gdoc.mcpb` beside the binary, and hands that file to Claude Desktop:
// TestDesktopWritesTheMcpbFromTheZipsTemplate. Decision 18 of the M14
// specification, Nail's call of 2026-10-03.
//
// The template is read between the check and the replacement, and the extension
// is written after the read-back, so the steps on stderr are in the order the
// run takes them: TestADesktopRunDrawsTheTwoExtensionStepsAndSaysQuitAndOpenAgain.
//
// Handing it over is the one place anything under go/ starts another program:
// /usr/bin/open by its full path, with the file as its only argument, on macOS
// alone. go/boundary's TestOnlyDesktopRunsAProgram reads desktop.go's syntax
// tree and holds all three of those: os/exec is imported by that one file, it
// holds one exec.Command call, and that call names the opener and one argument.
// Off macOS the file is written and nothing is started, because there is
// nothing there to hand it to: TestDesktopCallsTheRunnerWithOpenAndThePathOnly
// and TestDesktopOffMacOSWritesAndRunsNothing.
//
// The template comes out of the zip this run verified, through
// update.FileFrom, so it is always the template of the release that is
// installed and never a second download. It is read before the binary is
// replaced: a release carrying no template is refused with the gdoc on this
// machine exactly where it was, because a person who asked for Claude Desktop
// and got a new binary and a warning they scrolled past is left wondering why
// their chat has no gdoc in it:
// TestAZipWithoutTheTemplateIsRefusedBeforeTheBinaryIsReplaced.
//
// The extension follows the release rather than the binary, so a machine
// already running the newest gdoc still downloads that release's zip for the
// template and still gets the extension refreshed:
// TestDesktopWhenAlreadyNewestStillRefreshesTheExtension. A run that found a
// release it did not install, which is a major it declined, writes nothing and
// says so: the template would name a version this machine does not run, and
// not knowing never resolves to overwrite.
//
// --desktop is refused beside --rollback, which puts an earlier binary back,
// and beside --check, which writes nothing at all. Each pair names two runs:
// TestDesktopIsRefusedWithRollbackOrCheck. A path carrying a quote or a
// backslash is refused by name rather than written into a manifest Claude
// Desktop cannot read: TestAPathThatWouldBreakTheManifestIsRefusedByName.
//
// Past the point where the file is written, everything is reported rather than
// raised. The binary has already been replaced, or there was never one to
// replace, so a write that failed or an open that refused is a warning naming
// the file, which is the house rule that nothing raises over a write that
// happened.
//
// Plain `gdoc update` is the run it has always been: it writes no extension and
// starts no program, and Claude Desktop runs the new binary after a person
// quits it and opens it again: TestPlainUpdateWritesNoExtensionAndRunsNothing.
// It ends with one line when the release's template, filled for this machine,
// differs from the extension already beside the binary, and that line names
// `gdoc update --desktop`. The comparison leaves the version field out of both
// sides, so a release that changed its number and nothing else asks nobody to
// do anything, and a machine with no extension beside its binary is told
// nothing at all: TestPlainUpdateHintsWhenTheTemplateChanged and
// TestPlainUpdateIsSilentWithNoMcpbOrNoChange.
//
// The extension has no settings, which TestTheManifestDeclaresNoSettings
// holds, and it passes neither setting flag: its line is the word mcp alone,
// which TestTheManifestTemplateParses holds. What
// differs from one machine to the next is the binary's path and the release,
// which is why the template travels in the release and is filled here rather
// than being a file somebody edits. A colleague who wants another mark in a
// chat sets it in settings.yaml, which the session reads at its start like any
// other run.
//
// # The presence app is installed beside the binary, and opened
//
// With the presence on, `gdoc update` installs the menu bar app too, out of
// the same verified zip the binary came from: the darwin zips carry it at
// presence/GdocPresence.app and it lands at ~/Applications/GdocPresence.app,
// with the app that was there kept as GdocPresence.app.previous. The paths are
// behind two variables for the reason executable is, so no test touches the
// ~/Applications of the machine it runs on. The whole of the extraction is
// internal/update's ApplyBundle, and the rules it refuses by are its doc.go's.
//
// Off the flag nothing of this happens. The object a caller reads is the
// object it always was, the steps on stderr are the steps they always were,
// and no app is installed, refreshed or uninstalled:
// TestWithThePresenceOffAnUpdateIsTodaysUpdate, which also states the two
// step lists the recorded screens are drawn from as literals.
//
// Every app failure is a warning naming both paths, never a refusal. The
// binary has landed by the time the app is touched, and gdoc does not take a
// binary back because a folder beside it did not arrive, so the two may sit
// at different versions until a later run and the app is what says so:
// TestAPresenceFailureIsAWarningNamingBothPathsAndTheBinaryIsUpdated. That
// holds for a run that came only for the app too, where the release itself
// could not be read: nobody typed anything about an app on that line and the
// binary here needed nothing, so the run answers with a warning naming both
// paths: TestAnAppOnlyRefreshThatCannotReadTheReleaseIsAWarning. With
// --desktop the extension is what the person asked for, so a release that
// cannot be read is still a refusal. The four fields it reports, `presence`,
// `presence_version`, `presence_sha256` and `presence_previous`, are all
// omitempty, so a run that reached no app says nothing about one.
//
// A run with nothing newer still installs the app, out of the release the
// binary here came from, where the app is missing or at another version. It is
// how a person who has just turned the flag on gets the app without waiting
// for the next release, and it costs one download rather than a second one:
// TestAnUpToDateRunWithThePresenceOnInstallsItFromTheReleaseHere. --check
// installs nothing, the app included: TestCheckNeverInstallsTheApp. A platform
// with no app gets one warning naming itself:
// TestThePresenceOffMacOSWritesNothingAndNamesThePlatform.
//
// --rollback swaps the app after the binary, each swap its own involution, and
// a machine with no earlier app keeps the one it has and is told so:
// TestRollbackWithThePresenceOnSwapsBoth and
// TestRollbackWithNoPreviousAppWarnsAndKeepsIt.
//
// An install is a running app, and so is the next command after one. Under
// the flag, on macOS, a command whose run is recorded opens the installed app
// where that app is not already running: dispatch calls desktop.go's
// openPresenceApp right after the record's start line and only where the run
// is recorded. Every command but `update`, which is the one command that
// moves the bundle: it calls the same opener itself, once, after it placed or
// rolled the app back, because an open at the start line would have started
// the bundle the run then renamed away. So no step line asks a person to go
// and open anything.
// TestACommandUnderTheFlagOpensTheAppOnceAndNothingWhenItIsRunning,
// TestTheOpenerAsksThreeThingsAndRunsOpenGOnce,
// TestAFirstInstallOpensTheAppAndTellsNobodyToOpenIt and
// TestAnUpdateOpensTheAppItPlacedAndNotTheOneItReplaces.
//
// Whether the app is running is one lock and never a byte. The app holds
// flock on presence/app.lock in the config directory for its whole life, and
// the opener asks LOCK_EX | LOCK_NB on that file and lets go at once: it
// creates nothing, reads nothing, and a file that is not there is an app that
// never ran. Everything else is silence, so the flag on is the same object
// the flag off prints: TestTheFlagChangesNoByteOfTheObject, with go/boundary's
// TestTheLockNameIsOneWordInTwoLanguages over the one name Go and Swift share.
// The seam the one program goes through has those two callers and no third,
// which go/boundary's TestTheOneProgramOpensOnlyTheExtensionAndTheApp holds by
// reading the syntax tree. Decided 2026-10-18, DECISIONS.md, "The app opens
// itself".
//

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
// nothing to wait for.
//
// A child nobody waits for is a zombie until this process ends, and
// os.Process.Release does not change that: it drops Go's handle and never
// asks the kernel for the exit status. One run would never notice, but a
// `gdoc mcp` session is one process for an afternoon of tool calls, so the
// wait is done on a goroutine of its own: the command goes on at once and the
// child is reaped when it exits.
func startProgram(name string, wait bool, args ...string) error {
	if name != macOSOpen {
		return fmt.Errorf("gdoc runs %s and no other program, and this run asked for %s", macOSOpen, name)
	}
	cmd := exec.Command(macOSOpen, args...)
	if !wait {
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("%s %s did not start: %v", macOSOpen, strings.Join(args, " "), err)
		}
		go func() { _ = cmd.Wait() }()
		return nil
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
