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
	"os"
	"runtime"
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
}

func cmdUpdate(ctx context.Context, a *args) emit.Result {
	flags := update.Flags{
		Check:    a.has("--check"),
		Major:    a.has("--major"),
		Nightly:  a.has("--nightly"),
		Rollback: a.has("--rollback"),
	}
	if err := oneRun(flags); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	path, err := binaryPath()
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	if flags.Rollback {
		return runRollback(path)
	}
	return runUpdate(ctx, path, flags)
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
func runRollback(path string) emit.Result {
	data := updateData{Installed: releaseVersion(), Path: path}
	res, err := update.Rollback(path)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Data: data}
	}
	data.Action = string(update.RolledBack)
	data.SHA256 = res.Sum
	data.Previous = res.Previous
	return emit.Result{OK: true, Data: data}
}

// runUpdate is the whole of a run that looks at GitHub: read the listing,
// choose a release of each channel, decide, and carry the decision out.
func runUpdate(ctx context.Context, path string, flags update.Flags) emit.Result {
	installed, warns := installedVersion()
	data := updateData{Installed: installed.String(), Path: path}

	p := guard.NewPolicy()
	p.AllowUpdateFrom(updateRepo)
	reach := openPlain(p)

	var entries []update.Entry
	if err := reach.GetJSON(ctx, releasesURL, &entries); err != nil {
		// Not a failure. A person on a train typed a command and is owed an
		// answer, and the answer is that today gdoc cannot say: the binary
		// they have is still the binary they have.
		data.Action = string(update.Unreachable)
		warns = append(warns, fmt.Sprintf("the releases of %s could not be read, so this run cannot say whether there is a newer gdoc: %v", updateRepo, err))
		return emit.Result{OK: true, Data: data, Warnings: reachWarnings(reach, warns)}
	}

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
		return emit.Result{OK: false, Error: asked.Error(), Data: data, Warnings: reachWarnings(reach, warns)}
	}
	if aside != nil {
		warns = append(warns, fmt.Sprintf("the other channel has nothing this run could install, which changes nothing about this one: %v", aside))
	}

	d := update.Decide(update.State{Installed: installed, Stable: stable.Version, Nightly: nightly.Version}, flags)
	data.Action = string(d.Action)
	data.To = d.To.String()
	data.Run = d.Run
	if !d.Installs() {
		return emit.Result{OK: true, Data: data, Warnings: reachWarnings(reach, warns)}
	}

	// Past here something is replaced, so the action stops being true until it
	// is: a run that fails on the download says what it was taking and claims
	// no action at all.
	data.Action = ""
	chosen := stable
	if nightly.Version == d.To {
		chosen = nightly
	}
	res, err := install(ctx, reach, chosen, path)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Data: data, Warnings: reachWarnings(reach, warns)}
	}
	data.Action = string(update.Updated)
	data.Verified = true
	data.SHA256 = res.Sum
	data.Previous = res.Previous
	return emit.Result{OK: true, Data: data, Warnings: reachWarnings(reach, warns)}
}

// install fetches the checksum file and the zip, and hands both to the
// updater, which verifies before it moves anything. The checksum file is
// fetched first: it is the smaller read, and a release that cannot be verified
// is a release nothing may be replaced from, so there is no sense in
// downloading megabytes before finding that out.
func install(ctx context.Context, reach plain, rel update.Release, path string) (update.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	sums, err := reach.GetBytes(ctx, rel.ChecksumsURL, maxChecksums)
	if err != nil {
		return update.Result{}, fmt.Errorf("the checksums of release %s could not be read, so nothing it holds could be verified: %v", rel.Tag, err)
	}
	archive, err := reach.GetBytes(ctx, rel.AssetURL, maxArchive)
	if err != nil {
		return update.Result{}, fmt.Errorf("%s could not be downloaded, so nothing was replaced: %v", rel.AssetName, err)
	}
	return update.Apply(archive, sums, rel.AssetName, zipBinaryName(runtime.GOOS), path)
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
