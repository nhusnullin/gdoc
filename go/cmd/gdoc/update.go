// The command that replaces gdoc with a newer gdoc.
//
// `gdoc update` reads the releases of one GitHub repository, decides what to
// do about the difference between the newest one and the binary that is
// running, and, if there is something to take, downloads it, checks it against
// the checksum the release published, and renames it into place.
//
// Three rules shape it. It runs only when a person types it, so no other
// command reaches anything here. Nothing on disk moves before the bytes have
// matched the published checksum. And the binary it replaced is kept beside
// the new one, so `gdoc update --rollback` is one command away.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"gdoc/internal/emit"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/update"
)

// updateRepo is the repository whose releases gdoc installs from. It is the
// grant the run opens, so it is also the whole of what the run may reach.
const updateRepo = "nhusnullin/gdoc"

// releasesURL is the one call gdoc makes on api.github.com, and it asks for
// the biggest page GitHub gives.
//
// The size is the point, not a tuning. GitHub answers thirty releases when
// nobody says otherwise, and the nightly cuts one most night main moved, so a
// page of thirty stops carrying the last hand-cut stable release about a month
// after it was cut. Choose would then find no stable candidate and a bare
// `gdoc update` would report that there is no stable release at all, while
// `--nightly` still worked. A hundred is GitHub's maximum, and the guard
// admits `per_page` on this call alone and holds it to that number.
//
// A hundred moves the ceiling rather than removing it: about three months of
// nightlies. Walking pages is the thing that removes it, and it is a decision
// rather than a tidy-up, so it is in docs/backlog/ with its reason.
const releasesURL = "https://api.github.com/repos/" + updateRepo + "/releases?per_page=100"

// zipBinaryName is what the release workflow calls the binary at the top of
// the zip, and it is the one name that follows the platform. Every platform
// gets `gdoc` except Windows, where the release workflow packs `gdoc.exe`,
// because a Windows machine runs the extension and not the name.
func zipBinaryName(goos string) string {
	if goos == "windows" {
		return "gdoc.exe"
	}
	return "gdoc"
}

// The two ceilings and the download's clock. The listing has its own
// five-second bound inside gapi, because a person is waiting at a terminal for
// an answer; a zip is megabytes on whatever connection the machine has, so it
// gets minutes rather than seconds.
const (
	maxArchive      = 64 << 20
	maxChecksums    = 1 << 20
	downloadTimeout = 5 * time.Minute
)

// plain is what the update needs of a reach with no credential. *gapi.Plain
// satisfies it, and the interface is named here for the reason session is
// named in read.go: a test stands in for the wire without naming net/http.
type plain interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	GetBytes(ctx context.Context, rawURL string, limit int64) ([]byte, error)
	Warnings() []string
}

// openPlain is the reach factory behind a variable, as openSession is. The
// base transport is nil, which means the real wire, and the client is still
// the guard's.
var openPlain = func(p *guard.Policy) plain { return gapi.OpenPlain(p, nil) }

// executable is where this process's binary is, behind a variable so a test
// can point a run at a file it made itself.
var executable = os.Executable

// updateData is what `gdoc update` prints. Every field is a fact about this
// run: what was installed, what the two channels hold, what was done, and what
// the file at the path hashes to afterwards.
//
// Action is absent when the run failed part way. The words in it are the six
// update.Action carries, each of them something that finished; a download that
// did not arrive is a failure with its reason in the envelope, and writing
// "updated" beside ok: false would be the object contradicting itself.
//
// Verified and SHA256 are two different facts. SHA256 is the hash of the file
// at the path now, and it is there after an update and after a rollback alike.
// Verified says that hash came out of bytes the release published a checksum
// for, which a rollback cannot say about a binary it only put back.
type updateData struct {
	Installed     string `json:"installed,omitempty"`
	LatestStable  string `json:"latest_stable,omitempty"`
	LatestNightly string `json:"latest_nightly,omitempty"`
	Action        string `json:"action,omitempty"`
	To            string `json:"to,omitempty"`
	Run           string `json:"run,omitempty"`
	Path          string `json:"path"`
	Verified      bool   `json:"verified"`
	SHA256        string `json:"sha256,omitempty"`
	Previous      string `json:"previous,omitempty"`
	Extension     string `json:"extension,omitempty"`
	// ExtensionOpened is true where the extension was handed to Claude Desktop,
	// and absent otherwise: omitempty on a bool never prints false. It is read
	// beside Extension, which is how a skill tells the two silences apart. An
	// extension named with this absent was written and not opened, which is
	// every platform but macOS. Both absent is a run that did not finish the
	// extension step: it wrote none, or it wrote one and the open failed, and
	// the warnings say which.
	ExtensionOpened bool `json:"extension_opened,omitempty"`
}

func cmdUpdate(ctx context.Context, a *args, errOut io.Writer) emit.Result {
	flags := update.Flags{
		Check:    a.has("--check"),
		Major:    a.has("--major"),
		Nightly:  a.has("--nightly"),
		Rollback: a.has("--rollback"),
	}
	desktop := a.has("--desktop")
	if err := oneRun(flags); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	if err := oneDesktopRun(flags, desktop); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	path, err := binaryPath()
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	if flags.Rollback {
		return runRollback(path)
	}
	return runUpdate(ctx, path, flags, desktop, errOut)
}

// oneDesktopRun refuses the two flags --desktop cannot be typed beside.
//
// --desktop writes a file naming the version that is installed, so --check,
// which installs nothing and writes nothing, and --rollback, which puts an
// earlier binary back, each name a different run from it. Refused by name
// rather than ignored, the way oneRun refuses its own pairs.
//
// TestDesktopIsRefusedWithRollbackOrCheck.
func oneDesktopRun(f update.Flags, desktop bool) error {
	if !desktop {
		return nil
	}
	for _, other := range []struct {
		name  string
		given bool
		why   string
	}{
		{"--rollback", f.Rollback, "puts an earlier binary back, and --desktop writes the extension of the release it just installed"},
		{"--check", f.Check, "writes nothing at all, and --desktop writes the extension"},
	} {
		if other.given {
			return fmt.Errorf("%s %s: the two together name two runs, so type one of them", other.name, other.why)
		}
	}
	return nil
}

// oneRun refuses the flag pairs that name two runs. --rollback is a file move
// on this machine and the other three are about which release to fetch, so
// typing one of each says two things and gdoc does neither.
func oneRun(f update.Flags) error {
	if !f.Rollback {
		return nil
	}
	for _, other := range []struct {
		name  string
		given bool
	}{{"--check", f.Check}, {"--major", f.Major}, {"--nightly", f.Nightly}} {
		if other.given {
			return fmt.Errorf("--rollback puts the binary that was here back, and %s is about which release to fetch: the two together name two runs, so type one of them", other.name)
		}
	}
	return nil
}

// binaryPath is the file this run would replace.
//
// A symlink is refused naming both ends. `~/.local/bin/gdoc` is a link into a
// checkout on the machine gdoc is developed on, and replacing it would drop a
// release binary over the link, leaving `make build` writing to a file nobody
// runs any more. Not knowing never resolves to overwrite.
//
// The refusal holds where os.Executable returns the path gdoc was invoked
// through, which is darwin. On Linux it reads /proc/self/exe, which the kernel
// has already resolved, so an update run through such a link replaces the
// checkout's own build instead of being refused. That is the file `make build`
// writes, so the next build takes it back; it is a worse answer than the
// refusal rather than a lost binary. Windows is unmeasured until the checklist
// in docs/backlog/windows-rollout-checklist.md runs.
func binaryPath() (string, error) {
	path, err := executable()
	if err != nil {
		return "", fmt.Errorf("gdoc cannot find the binary it is running from, so there is nothing here to replace: %v", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("the binary at %s cannot be read, so there is nothing here to replace: %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return path, nil
	}
	target, err := os.Readlink(path)
	if err != nil {
		target = "somewhere this run cannot read"
	}
	return "", fmt.Errorf("%s is a symlink to %s, which is how a checkout installs gdoc: an update would put a release binary over the link and leave the checkout's own build unreachable. Run make build in that checkout instead", path, target)
}

// runRollback puts the earlier binary back. It reads no listing: what it does
// depends on one file being beside another, and GitHub has nothing to say
// about that.
//
// The object carries no `installed`. What ends up at the path is the earlier
// binary, and the only way to read its version would be to run it, which
// nothing under go/ does. The release that printed the object is already the
// envelope's `version`, and putting it under `installed` beside a `sha256` of
// the older file would be one fact contradicting another.
//
// TestRollbackPutsTheEarlierBinaryBack.
func runRollback(path string) emit.Result {
	data := updateData{Path: path}
	res, err := update.Rollback(path)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Data: data}
	}
	data.Action = string(update.RolledBack)
	data.SHA256 = res.Sum
	data.Previous = res.Previous
	return emit.Result{OK: true, Data: data}
}

// The steps a run narrates on stderr, in the order it takes them. The first
// two are every run's; the other five are planned only when the decision is
// to install, so a run with nothing to take never draws a download.
const (
	stepRead      = "read releases"
	stepChoose    = "choose release"
	stepChecksums = "read checksums"
	stepDownload  = "download"
	stepVerify    = "verify checksum"
	stepReplace   = "replace binary"
	stepReadBack  = "read back"
	stepTemplate  = "read extension"
	stepExtension = "write extension"
)

// runUpdate is the whole of a run that looks at GitHub, with its steps drawn
// on errOut for the person who typed it. The object is decided without
// looking at the drawing, so stdout is what it would be with nobody watching.
//
// TestAnUpdateNarratesItsStepsOnStderr,
// TestAFailedDownloadMarksItsStepAndDrawsNoLaterOne,
// TestAnUpToDateRunDrawsNoDownload and
// TestAnUnreachableGitHubIsMarkedOnTheReadStep.
//
// The list settles on every way out, a panic included, so the spinner never
// draws over the crash text safeDispatch writes: TestAPanicInAnUpdateStopsTheSpinner.
// A panic gets no result line, because nothing finished.
func runUpdate(ctx context.Context, path string, flags update.Flags, desktop bool, errOut io.Writer) emit.Result {
	pr := openProgress(errOut, "gdoc update")
	defer pr.settle()
	pr.Plan(stepRead, stepChoose)
	r := checkAndInstall(ctx, path, flags, desktop, pr)
	pr.Finish(resultLine(r))
	return r
}

// openProgress is the step list behind a variable, so a test can hand a run
// the terminal form over a buffer.
var openProgress = newProgress

// checkAndInstall reads the listing and hands it on to be decided.
func checkAndInstall(ctx context.Context, path string, flags update.Flags, desktop bool, pr *progress) emit.Result {
	installed, warns := installedVersion()
	data := updateData{Installed: installed.String(), Path: path}

	p := guard.NewPolicy()
	p.AllowUpdateFrom(updateRepo)
	reach := openPlain(p)

	pr.Start(stepRead, "")
	var entries []update.Entry
	if err := reach.GetJSON(ctx, releasesURL, &entries); err != nil {
		// Not a failure. A person on a train typed a command and is owed an
		// answer, and the answer is that today gdoc cannot say: the binary
		// they have is still the binary they have.
		pr.Fail(err)
		data.Action = string(update.Unreachable)
		warns = append(warns, fmt.Sprintf("the releases of %s could not be read, so this run cannot say whether there is a newer gdoc: %v", updateRepo, err))
		if desktop {
			warns = append(warns, "the extension is written from the release's own template, and this run read no release, so nothing was written and Claude Desktop is as it was")
		}
		return emit.Result{OK: true, Data: data, Warnings: reachWarnings(reach, warns)}
	}
	pr.Done(fmt.Sprintf("%s, %d listed", updateRepo, len(entries)))
	return decideUpdate(ctx, reach, entries, flags, desktop, installed, data, warns, pr)
}

// decideUpdate chooses a release of each channel, decides, and hands a
// decision to install on to installDecided.
func decideUpdate(ctx context.Context, reach plain, entries []update.Entry, flags update.Flags, desktop bool, installed update.Version, data updateData, warns []string, pr *progress) emit.Result {
	pr.Start(stepChoose, "")
	platform := update.Platform(runtime.GOOS, runtime.GOARCH)
	stable, stableErr := update.Choose(entries, update.Stable, platform)
	nightly, nightlyErr := update.Choose(entries, update.Nightly, platform)
	data.LatestStable = stable.Version.String()
	data.LatestNightly = nightly.Version.String()

	// The channel that was asked for has to answer. The other one only fills
	// in a field, so its trouble is a warning: a nightly that carries no zip
	// for this platform is not a reason to refuse a stable update.
	asked, aside := stableErr, nightlyErr
	if flags.Channel() == update.Nightly {
		asked, aside = nightlyErr, stableErr
	}
	if asked != nil {
		pr.Fail(asked)
		return emit.Result{OK: false, Error: asked.Error(), Data: data, Warnings: reachWarnings(reach, warns)}
	}
	if aside != nil {
		warns = append(warns, fmt.Sprintf("the other channel has nothing this run could install, which changes nothing about this one: %v", aside))
	}
	latest := stable
	if flags.Channel() == update.Nightly {
		latest = nightly
	}
	pr.Done(chosenDetail(latest, flags.Channel(), platform, installed))

	d := update.Decide(update.State{Installed: installed, Stable: stable.Version, Nightly: nightly.Version}, flags)
	data.Action = string(d.Action)
	data.To = d.To.String()
	data.Run = d.Run
	if !d.Installs() {
		if desktop {
			return refreshExtension(ctx, reach, latest, installed, data, warns, pr)
		}
		return emit.Result{OK: true, Data: data, Warnings: reachWarnings(reach, warns)}
	}
	pr.Plan(installSteps(desktop)...)
	chosen := stable
	if nightly.Version == d.To {
		chosen = nightly
	}
	return installDecided(ctx, reach, chosen, desktop, data, warns, pr)
}

// installSteps is the list a run that installs draws, with the two extension
// steps in it only when --desktop was typed. The template is read before the
// binary is replaced, because a release that cannot write an extension is a
// release this run refuses while the gdoc on this machine is still untouched.
func installSteps(desktop bool) []string {
	if !desktop {
		return []string{stepChecksums, stepDownload, stepVerify, stepReplace, stepReadBack}
	}
	return []string{stepChecksums, stepDownload, stepVerify, stepTemplate, stepReplace, stepReadBack, stepExtension}
}

// refreshExtension is --desktop on a run with nothing to install. The extension
// names the binary that is here, so the template comes from the release that is
// here: the zip is downloaded for that one file and no binary moves.
//
// A run that found a release it did not install, which is a major it declined,
// writes no extension. The template would name a version this machine does not
// run, and not knowing never resolves to overwrite.
//
// TestDesktopWhenAlreadyNewestStillRefreshesTheExtension.
func refreshExtension(ctx context.Context, reach plain, rel update.Release, installed update.Version, data updateData, warns []string, pr *progress) emit.Result {
	if update.Compare(rel.Version, installed) != 0 {
		warns = append(warns, fmt.Sprintf("--desktop writes the extension of the gdoc that is installed, and this run installed nothing: %s is here and %s is what the release this run found holds, so Claude Desktop is as it was", installed, rel.Version))
		return emit.Result{OK: true, Data: data, Warnings: reachWarnings(reach, warns)}
	}
	pr.Plan(stepChecksums, stepDownload, stepVerify, stepTemplate, stepExtension)
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	template, err := readTemplate(ctx, reach, rel, pr)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Data: data, Warnings: reachWarnings(reach, warns)}
	}
	data, warns = withExtension(template, rel.Version.String(), data, warns, pr)
	return emit.Result{OK: true, Data: data, Warnings: reachWarnings(reach, warns)}
}

// installDecided carries out a decision to install, and fills in what the
// file at the path is afterwards.
func installDecided(ctx context.Context, reach plain, chosen update.Release, desktop bool, data updateData, warns []string, pr *progress) emit.Result {
	// Past here something is replaced, so the action stops being true until it
	// is: a run that fails on the download says what it was taking and claims
	// no action at all.
	data.Action = ""
	res, template, err := install(ctx, reach, chosen, data.Path, desktop, pr)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Data: data, Warnings: reachWarnings(reach, warns)}
	}
	data.Action = string(update.Updated)
	data.Verified = true
	data.SHA256 = res.Sum
	data.Previous = res.Previous
	if desktop {
		data, warns = withExtension(template, chosen.Version.String(), data, warns, pr)
	} else if extensionChanged(template, data.Path, chosen.Version.String()) {
		warns = append(warns, hintLine())
	}
	return emit.Result{OK: true, Data: data, Warnings: reachWarnings(reach, warns)}
}

// withExtension writes the extension and hands it to Claude Desktop, as one
// step, because handing it over is what writing it was for.
//
// Everything that goes wrong here is a warning rather than a refusal. The
// binary has already been replaced by this point, or there was never one to
// replace, so the run did what it mostly came to do; a file that could not be
// written or an open that refused is reported with the file still named, which
// is the house rule that nothing trusts a success and nothing raises over a
// write that happened.
//
// TestDesktopWritesTheMcpbFromTheZipsTemplate,
// TestDesktopCallsTheRunnerWithOpenAndThePathOnly and
// TestDesktopOffMacOSWritesAndRunsNothing.
func withExtension(template []byte, version string, data updateData, warns []string, pr *progress) (updateData, []string) {
	file := mcpbPath(data.Path)
	opened := false
	err := pr.Do(stepExtension, tildePath(file), func() (string, error) {
		manifest, err := fillManifest(template, data.Path, version)
		if err != nil {
			return "", err
		}
		if err := writeMcpb(file, manifest); err != nil {
			return "", fmt.Errorf("the extension could not be written to %s: %w", file, err)
		}
		if desktopOS != "darwin" {
			return joinDetail(tildePath(file), "written, and Claude Desktop runs on macOS, so open it there"), nil
		}
		if err := runProgram(macOSOpen, file); err != nil {
			return "", err
		}
		opened = true
		return joinDetail(tildePath(file), "opened, so Claude Desktop offers to install it"), nil
	})
	if err != nil {
		return data, append(warns, err.Error())
	}
	data.Extension = file
	data.ExtensionOpened = opened
	return data, warns
}

// install fetches the checksum file and the zip, and hands both to the
// updater, which verifies before it moves anything. The checksum file is
// fetched first: it is the smaller read, and a release that cannot be verified
// is a release nothing may be replaced from, so there is no sense in
// downloading megabytes before finding that out.
//
// Each read, the check and the replacement is a step on pr. The check runs as
// its own step before Apply, which checks again: the second hash is the
// safety, and the first is there so a person sees which of the two failed.
// The read back is inside Apply, so a read back that fails is drawn on the
// replace step, and the read back step is drawn only once there is a hash.
//
// The second return is the Claude Desktop manifest template out of the same
// verified zip, and it is read before the binary is replaced. With --desktop a
// release that carries none is refused right there, while the gdoc on this
// machine is still the gdoc that was here. Without it the template is read for
// the comparison the hint rests on, and a release that carries none is no
// hint and no trouble at all.
func install(ctx context.Context, reach plain, rel update.Release, path string, desktop bool, pr *progress) (update.Result, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	sumsName := update.ChecksumsName(rel.Tag)
	var sums, archive, template []byte
	var res update.Result
	steps := []struct {
		name, detail string
		run          func() (string, error)
	}{
		{stepChecksums, sumsName, func() (string, error) {
			var err error
			if sums, err = reach.GetBytes(ctx, rel.ChecksumsURL, maxChecksums); err != nil {
				return "", fmt.Errorf("the checksums of release %s could not be read, so nothing it holds could be verified: %v", rel.Tag, err)
			}
			return "", nil
		}},
		{stepDownload, joinDetail(rel.AssetName, humanSize(rel.AssetSize)), func() (string, error) {
			var err error
			if archive, err = reach.GetBytes(ctx, rel.AssetURL, maxArchive); err != nil {
				return "", fmt.Errorf("%s could not be downloaded, so nothing was replaced: %v", rel.AssetName, err)
			}
			return "", nil
		}},
		{stepVerify, "", func() (string, error) {
			return "matches " + sumsName, update.Verify(archive, sums, rel.AssetName)
		}},
		{stepTemplate, templateInZip, func() (string, error) {
			var err error
			if template, err = update.FileFrom(archive, sums, rel.AssetName, templateInZip); err != nil {
				if desktop {
					return "", fmt.Errorf("%v, so nothing was replaced and no extension was written. %s is packed into every release from v2.9.0 on: install a newer release, or drop --desktop", err, templateInZip)
				}
				// A plain run was not asked about Claude Desktop, so a release
				// that carries no template is one this run has nothing to say
				// about.
				template = nil
			}
			return "", nil
		}},
		{stepReplace, tildePath(path), func() (string, error) {
			var err error
			res, err = update.Apply(archive, sums, rel.AssetName, zipBinaryName(runtime.GOOS), path)
			if err != nil || res.Previous == "" {
				return "", err
			}
			return tildePath(path) + ", the old one kept beside it", nil
		}},
	}
	for _, s := range steps {
		// The template read is a drawn step only when the extension is what
		// the run came for. A plain run reads it between the same two steps
		// and says nothing about it, because nothing about Claude Desktop is
		// what a plain update is.
		if s.name == stepTemplate && !desktop {
			if _, err := s.run(); err != nil {
				return update.Result{}, nil, err
			}
			continue
		}
		if err := pr.Do(s.name, s.detail, s.run); err != nil {
			return update.Result{}, nil, err
		}
	}
	pr.Start(stepReadBack, "")
	pr.Done("sha256 " + shortSum(res.Sum))
	return res, template, nil
}

// readTemplate is the Claude Desktop manifest template out of a release this
// run is not installing anything from. It is the two reads and the check with
// nothing after them: a refresh writes the extension and never a binary.
//
// TestDesktopWhenAlreadyNewestStillRefreshesTheExtension.
func readTemplate(ctx context.Context, reach plain, rel update.Release, pr *progress) ([]byte, error) {
	sumsName := update.ChecksumsName(rel.Tag)
	var sums, archive, template []byte
	for _, s := range []struct {
		name, detail string
		run          func() (string, error)
	}{
		{stepChecksums, sumsName, func() (string, error) {
			var err error
			if sums, err = reach.GetBytes(ctx, rel.ChecksumsURL, maxChecksums); err != nil {
				return "", fmt.Errorf("the checksums of release %s could not be read, so nothing it holds could be verified: %v", rel.Tag, err)
			}
			return "", nil
		}},
		{stepDownload, joinDetail(rel.AssetName, humanSize(rel.AssetSize)), func() (string, error) {
			var err error
			if archive, err = reach.GetBytes(ctx, rel.AssetURL, maxArchive); err != nil {
				return "", fmt.Errorf("%s could not be downloaded, so no extension was written: %v", rel.AssetName, err)
			}
			return "", nil
		}},
		{stepVerify, "", func() (string, error) {
			return "matches " + sumsName, update.Verify(archive, sums, rel.AssetName)
		}},
		{stepTemplate, templateInZip, func() (string, error) {
			var err error
			if template, err = update.FileFrom(archive, sums, rel.AssetName, templateInZip); err != nil {
				return "", fmt.Errorf("%v, so no extension was written. %s is packed into every release from v2.9.0 on: install a newer release, or drop --desktop", err, templateInZip)
			}
			return "", nil
		}},
	} {
		if err := pr.Do(s.name, s.detail, s.run); err != nil {
			return nil, err
		}
	}
	return template, nil
}

// chosenDetail is the choose step's line: the release, the channel and the
// platform, and what it would replace.
func chosenDetail(rel update.Release, ch update.Channel, platform string, installed update.Version) string {
	from := "from a checkout build"
	if !installed.IsZero() {
		from = "from " + installed.String()
	}
	return joinDetail(rel.Version.String(), ch.String(), platform, from)
}

// resultLine is the one line a person reads under the steps, and none for a
// run that failed: the red cross and its reason are already the last thing on
// the screen. It restates the object's own fields, so it cannot say more than
// the object does.
func resultLine(r emit.Result) string {
	d, ok := r.Data.(updateData)
	if !r.OK || !ok {
		return ""
	}
	return strings.TrimSpace(actionLine(d) + " " + extensionLine(d))
}

// extensionLine is what a --desktop run adds to that line, and nothing at all
// for a run that wrote no extension. Installing one restarts the chat process
// and leaves agent mode on the binary it already started, so the words are quit
// and open again, which is what the chat notice says too
// (docs/v2/MEASURED.md, measurement 2).
//
// TestDesktopWritesTheMcpbFromTheZipsTemplate and
// TestDesktopOffMacOSWritesAndRunsNothing.
func extensionLine(d updateData) string {
	switch {
	case d.Extension == "":
		return ""
	case d.ExtensionOpened:
		return "Claude Desktop offers to install the extension: install it, then quit Claude Desktop and open it again."
	default:
		return fmt.Sprintf("The extension is at %s. Claude Desktop runs on macOS, so open that file there.", tildePath(d.Extension))
	}
}

// actionLine is the run itself in one sentence, read out of the object's own
// fields.
func actionLine(d updateData) string {
	switch update.Action(d.Action) {
	case update.Updated:
		return fmt.Sprintf("gdoc %s installed. gdoc update --rollback goes back.", d.To)
	case update.MajorAvailable:
		return fmt.Sprintf("gdoc %s is a major release, so it was not installed. %s installs it.", d.To, d.Run)
	case update.Unreachable:
		return "GitHub did not answer, so the gdoc here is unchanged."
	case update.Checked:
		if d.To != "" {
			return fmt.Sprintf("gdoc %s is published. --check installs nothing.", d.To)
		}
	}
	if d.Run != "" {
		return fmt.Sprintf("Nothing newer to install. A newer nightly is there: %s.", d.Run)
	}
	return "Nothing newer to install."
}

// joinDetail joins the parts of a step's detail that are there.
func joinDetail(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, s := range parts {
		if s != "" {
			kept = append(kept, s)
		}
	}
	return strings.Join(kept, ", ")
}

// tildePath is a path under the home directory as a person writes it.
func tildePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || !strings.HasPrefix(path, home+string(os.PathSeparator)) {
		return path
	}
	return "~" + path[len(home):]
}

// shortSum is enough of a hash for a person to compare by eye. The whole of it
// is in the object.
func shortSum(sum string) string {
	const shown = 12
	if len(sum) <= shown {
		return sum
	}
	return sum[:shown]
}

// installedVersion is the running binary's own tag, and the warning for a
// binary that does not carry one. A checkout build names no release, and a
// tag `git describe` decorated with a commit is not a release either; both
// leave the zero version, which is below everything, so the run reports what
// is published rather than guessing what is here.
func installedVersion() (update.Version, []string) {
	tag := releaseVersion()
	if tag == "" {
		return update.Version{}, []string{"this gdoc was built from a checkout and names no release, so every release looks newer than it"}
	}
	v, err := update.Parse(tag)
	if err != nil {
		return update.Version{}, []string{fmt.Sprintf("this gdoc names %q, which is not a release tag, so there is nothing here to compare a release against: %v", tag, err)}
	}
	return v, nil
}

// reachWarnings is the policy's warnings in front of the command's, which is
// sessionWarnings for the one command that has no session.
func reachWarnings(reach plain, own []string) []string {
	out := append([]string{}, reach.Warnings()...)
	return append(out, own...)
}
