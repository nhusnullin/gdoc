// The guard over what publishes a release.
//
// Nobody watches a release being built. A tag is pushed, or the clock reaches
// 02:00 UTC, and whatever comes out of those two files is what a colleague's
// machine downloads and runs. The two workflows are therefore held here
// against the rest of the tree: the asset names against internal/update, the
// platform list against release/platforms, the zip's contents against what
// release/install.sh looks for once it has unpacked it, and the checks against
// the ones the Go workflow runs on every push.
//
// The YAML is parsed rather than grepped, because a trigger is structure: a
// tags list under the wrong key is a workflow that never runs, and reading it
// as text cannot tell the difference. The shell inside a step is read as text,
// because it is text.

package boundary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"

	"gdoc/internal/update"
)

// workflowFile is the part of a GitHub Actions workflow these tests judge.
// Actions reads more keys than these; what matters here is when a workflow
// runs, what it may write, and what its steps say.
type workflowFile struct {
	Name string `yaml:"name"`
	On   struct {
		Push struct {
			Tags     []string `yaml:"tags"`
			Branches []string `yaml:"branches"`
		} `yaml:"push"`
		Schedule []struct {
			Cron string `yaml:"cron"`
		} `yaml:"schedule"`
		WorkflowCall struct {
			Inputs  map[string]workflowInput  `yaml:"inputs"`
			Secrets map[string]workflowSecret `yaml:"secrets"`
		} `yaml:"workflow_call"`
		WorkflowDispatch struct {
			Inputs map[string]workflowInput `yaml:"inputs"`
		} `yaml:"workflow_dispatch"`
	} `yaml:"on"`
	Permissions map[string]string            `yaml:"permissions"`
	Jobs        map[string]workflowJobFields `yaml:"jobs"`
}

type workflowInput struct {
	Required bool   `yaml:"required"`
	Type     string `yaml:"type"`
}

type workflowSecret struct {
	Required bool `yaml:"required"`
}

type workflowJobFields struct {
	Uses        string            `yaml:"uses"`
	With        map[string]string `yaml:"with"`
	Secrets     string            `yaml:"secrets"`
	Needs       string            `yaml:"needs"`
	RunsOn      string            `yaml:"runs-on"`
	Permissions map[string]string `yaml:"permissions"`
	Steps       []workflowStep    `yaml:"steps"`
}

type workflowStep struct {
	Name string `yaml:"name"`
	Uses string `yaml:"uses"`
	If   string `yaml:"if"`
	Run  string `yaml:"run"`
}

// TestTheReleaseRunsOnATagAndWhenTheNightlyCallsIt holds the two entrances.
// A tag Nail pushes by hand triggers the push half; a tag CI pushes triggers
// nothing at all, because a tag pushed with GITHUB_TOKEN starts no workflow,
// which is why the nightly calls this one instead of hoping.
func TestTheReleaseRunsOnATagAndWhenTheNightlyCallsIt(t *testing.T) {
	w := readWorkflow(t, "release.yml")

	if !contains(w.On.Push.Tags, "v*") {
		t.Errorf("release.yml runs on tags %v; a release is cut by pushing a v tag, so the pattern is v*", w.On.Push.Tags)
	}

	tag, ok := w.On.WorkflowCall.Inputs["tag"]
	if !ok {
		t.Fatal("release.yml takes no tag input on workflow_call; the nightly calls it with the tag it just cut")
	}
	if !tag.Required || tag.Type != "string" {
		t.Errorf("release.yml's tag input is required=%v type=%q; a caller with no tag has nothing to build", tag.Required, tag.Type)
	}

	secret, ok := w.On.WorkflowCall.Secrets[oauthSecret]
	if !ok {
		t.Fatalf("release.yml declares no %s secret on workflow_call; a called run would build a binary nobody can sign in with", oauthSecret)
	}
	if !secret.Required {
		t.Errorf("release.yml's %s secret is not required; the refusal belongs at the door, not halfway through a build", oauthSecret)
	}

	if w.Permissions["contents"] != "write" {
		t.Errorf("release.yml has contents permission %q; publishing a release writes to this repository", w.Permissions["contents"])
	}
}

// TestTheReleaseRefusesATagThePluginDoesNotName holds the version rule at the
// only place it can be held. `make tag` writes the version into plugin.json as
// it cuts an x.y.0, so a tag whose manifest says something else is a plugin
// reporting a version no colleague can place against a release. The refusal
// comes before the build, because a release that is going to be refused should
// cost nothing.
func TestTheReleaseRefusesATagThePluginDoesNotName(t *testing.T) {
	steps := releaseSteps(t)

	check := stepSaying(steps, "plugin.json")
	if check < 0 {
		t.Fatal("release.yml never reads .claude-plugin/plugin.json; nothing would hold the tag against the version the plugin reports")
	}
	if !strings.Contains(steps[check].Run, "exit 1") {
		t.Error("release.yml reads plugin.json and never exits on what it finds; a version that disagrees with the tag has to stop the run")
	}

	build := stepSaying(steps, "make dist")
	if build < 0 {
		t.Fatal("release.yml never runs make dist; there would be no binary to pack")
	}
	if check > build {
		t.Error("release.yml builds before it checks the version against the tag; the refusal is cheap and belongs first")
	}
}

// TestOnlyMakeTagMovesThePluginVersion holds the one number a colleague reads.
// The plugin is pinned by its version string: Claude Code hands somebody a new
// copy when that string changes and not otherwise, so the string is a release
// of the skills and not a release of the binary. The nightly releases the
// binary, every night main moves, and nobody wants the skills re-installed
// nightly. So the nightly stops touching the manifest and commits nothing, and
// the release workflow asks the manifest to match the tag only when the tag is
// an x.y.0 that `make tag` cut.
func TestOnlyMakeTagMovesThePluginVersion(t *testing.T) {
	night := workflowScript(t, readWorkflow(t, "nightly.yml"))
	for _, forbidden := range []string{"plugin.json", "git commit"} {
		if strings.Contains(night, forbidden) {
			t.Errorf("nightly.yml says %q; a nightly is a binary release, and moving the plugin version would re-install the skills on every colleague's machine every night", forbidden)
		}
	}

	steps := releaseSteps(t)
	check := stepSaying(steps, "plugin.json")
	if check < 0 {
		t.Fatal("release.yml never reads .claude-plugin/plugin.json; nothing would hold an x.y.0 tag against the version the plugin reports")
	}
	if !strings.Contains(steps[check].If, ".0") {
		t.Errorf("release.yml's plugin check runs on the condition %q; a nightly tag carries a patch the manifest never names, so the check is for x.y.0 tags", steps[check].If)
	}
}

// TestTheReleaseBuildsWithTheSecretAndRefusesAnEmptyOne: the client secret is
// linked in, not committed, so a build without it produces a gdoc that can
// read with a token it already holds and cannot sign anybody in. That binary
// looks fine until a new colleague runs `gdoc auth login`, which is far too
// late to find out, so an empty secret is a failed release.
func TestTheReleaseBuildsWithTheSecretAndRefusesAnEmptyOne(t *testing.T) {
	steps := releaseSteps(t)
	build := stepSaying(steps, "make dist")
	if build < 0 {
		t.Fatal("release.yml never runs make dist")
	}
	run := steps[build].Run
	if !strings.Contains(run, oauthSecret) {
		t.Errorf("the make dist step never names %s; the binary would carry no client secret", oauthSecret)
	}
	if !strings.Contains(run, `-z "$`+oauthSecret) {
		t.Errorf("the make dist step never refuses an empty %s; a release built without it cannot sign anyone in", oauthSecret)
	}
}

// TestTheReleaseRunsTheChecksTheGoWorkflowRuns: a release is the one build
// nobody re-runs. The tag may sit on a commit whose own push went green, but
// it may also sit on a commit somebody tagged in a hurry, and the raced suite
// is where the guard's guarantees live.
func TestTheReleaseRunsTheChecksTheGoWorkflowRuns(t *testing.T) {
	script := releaseScript(t)
	for _, want := range []string{"gofmt", "go vet", "go test -race"} {
		if !strings.Contains(script, want) {
			t.Errorf("release.yml never runs %q; the Go workflow runs it on every push and a release is not the place to skip it", want)
		}
	}
}

// TestTheReleaseNamesTheFilesTheUpdaterFetches pins the workflow against the
// Go the same way the installer is pinned. `gdoc update` asks a release for
// two file names it builds itself, and this is the only place those names are
// written. A rename here with no rename there is an update that 404s on every
// machine at once.
func TestTheReleaseNamesTheFilesTheUpdaterFetches(t *testing.T) {
	script := releaseScript(t)
	// The workflow's variables are TAG and platform, so the names it builds
	// are these strings with the shell's dollar signs still in them.
	for _, want := range []string{
		update.AssetName("$TAG", "$platform"),
		update.ChecksumsName("$TAG"),
	} {
		if !strings.Contains(script, want) {
			t.Errorf("release.yml builds no %q; that is the name internal/update asks a release for", want)
		}
	}
}

// TestTheReleaseWalksThePlatformsFileRatherThanASecondList: release/platforms
// is the list, and install.sh already carries the one copy of it this tree
// tolerates, because that script leaves the repository inside the zip. The
// workflow does not leave, so it reads the file. A platform named in here as
// well would be a third copy, and the third copy is the one nobody updates.
func TestTheReleaseWalksThePlatformsFileRatherThanASecondList(t *testing.T) {
	script := releaseScript(t)
	if !strings.Contains(script, "release/platforms") {
		t.Error("release.yml never reads release/platforms; that file is the list of zips a release carries")
	}
	listed, _ := releasePlatforms(t)
	for _, p := range listed {
		if strings.Contains(script, p) {
			t.Errorf("release.yml names %s itself; release/platforms is the list, and a second copy of it here is one nobody updates", p)
		}
	}
}

// TestTheZipCarriesWhatTheInstallerLooksFor holds the packing step against the
// script that unpacks it. install.sh refuses a folder with no gdoc in it and
// refuses --skills when a skill folder is missing, and both refusals would
// land on a colleague rather than here.
func TestTheZipCarriesWhatTheInstallerLooksFor(t *testing.T) {
	script := releaseScript(t)
	for _, want := range []string{
		"release/install.sh",
		`cp README.md "$stage/README.md"`,
		"release/example",
		"skills",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("release.yml packs no %s; the zip is the binary, the installer, the README, the example and the skills", want)
		}
	}
	// The binary is renamed on the way in. install.sh looks for a file called
	// gdoc beside itself, never for gdoc-darwin-arm64.
	if !strings.Contains(script, "/gdoc") {
		t.Error("release.yml never writes the binary into the zip as gdoc; install.sh looks for that name and nothing else")
	}
	if !strings.Contains(script, releaseCreate) {
		t.Error("release.yml never creates the release; the zips would be built and left on the runner")
	}
	// Windows is the one platform whose name inside the zip is different, and
	// internal/update's zipBinaryName states the same two names as literals.
	// Drop this arm and a Windows update downloads a whole zip, verifies it,
	// and then fails saying it holds no gdoc.exe.
	if !strings.Contains(script, `name="gdoc.exe"`) {
		t.Error(`release.yml never packs the binary as gdoc.exe for windows-*; that is the name zipBinaryName tells the updater to look for`)
	}
}

// TestTheReleaseWaitsForThePresenceBuild holds the two jobs apart and in
// order.
//
// The app is built on macOS, because codesign, lipo and plutil are there and
// the publishing job is ubuntu. So the release is two jobs, and the one that
// publishes waits: a publish that started before the bundle was built would
// upload a darwin zip with no app in it, and nothing downstream would notice,
// because `gdoc update` treats a missing bundle as a release that carries
// none. The building job writes nothing to this repository and never
// publishes: one job creates the release, and a second job that could would be
// two releases racing for one tag.
func TestTheReleaseWaitsForThePresenceBuild(t *testing.T) {
	w := readWorkflow(t, "release.yml")
	name, publisher := releasePublisher(t, w)

	if publisher.Needs != presenceJobName {
		t.Errorf("release.yml's %s job needs %q; it needs %q, or the zips are packed before the bundle they carry exists",
			name, publisher.Needs, presenceJobName)
	}
	if !jobUses(publisher, "actions/download-artifact") {
		t.Errorf("release.yml's %s job downloads no artifact; the bundle is built in another job and reaches this one no other way", name)
	}

	job := presenceJob(t, "release.yml")
	if !strings.HasPrefix(job.RunsOn, "macos-") {
		t.Errorf("release.yml's %s job runs on %q; codesign, lipo and plutil are macOS, and a Swift app bundle is built nowhere else", presenceJobName, job.RunsOn)
	}
	if !jobUses(job, "actions/upload-artifact") {
		t.Errorf("release.yml's %s job uploads no artifact; the bundle would be built and left on the runner", presenceJobName)
	}
	if script := jobScript(job); strings.Contains(script, releaseCreate) {
		t.Errorf("release.yml's %s job runs %q; one job publishes a tag, and two of them racing for one tag is a release nobody can read", presenceJobName, releaseCreate)
	}
	if got := job.Permissions["contents"]; got != "read" {
		t.Errorf("release.yml's %s job has contents permission %q; it builds and uploads and writes nothing to this repository, so it asks for read", presenceJobName, got)
	}
}

// TestTheZipCarriesThePresenceForDarwinOnly holds the app inside the two zips
// that can run it, and out of every other one.
//
// A darwin zip without the bundle is a colleague whose `gdoc update` reports a
// release that carries no app, quietly, for as long as nobody looks. A linux
// or Windows zip with it is a universal Mach-O binary on a machine that cannot
// execute a byte of it. Neither is visible on the runner, so the zips are
// listed and read back before the release is created rather than after.
func TestTheZipCarriesThePresenceForDarwinOnly(t *testing.T) {
	steps := releaseSteps(t)

	pack := stepSaying(steps, stagePresence)
	if pack < 0 {
		t.Fatalf("release.yml's pack step never copies the bundle into %s; the zips would carry no app", stagePresence)
	}
	arm, ok := caseArm(steps[pack].Run, darwinArm)
	if !ok {
		t.Fatalf("release.yml's pack step opens no %s arm; the copy would reach every platform's zip", darwinArm)
	}
	if !strings.Contains(arm, stagePresence) {
		t.Errorf("release.yml copies the bundle into %s outside the %s arm; only a Mac can run it", stagePresence, darwinArm)
	}

	list := -1
	for i, step := range steps {
		if i == pack {
			continue
		}
		if strings.Contains(step.Run, "presence/") && strings.Contains(step.Run, "exit 1") {
			list = i
			break
		}
	}
	if list < 0 {
		t.Fatal("release.yml never lists the zips and fails on what they hold; a zip packed wrong reaches a colleague before anybody looks")
	}
	publish := stepSaying(steps, releaseCreate)
	if publish < 0 {
		t.Fatalf("release.yml never runs %q", releaseCreate)
	}
	if list > publish {
		t.Error("release.yml lists the zips after it publishes them; a listing that runs after the release is a report about a release that is already out")
	}
}

// TestThePresenceIsStampedWithTheTag holds the version the app shows against
// the release it came out of.
//
// The updater compares the plist inside the installed bundle with the release
// it is about to install, and installs nothing when the two agree. A bundle
// stamped 0.0.0, which is what a checkout build carries, would therefore be
// re-installed on every single `gdoc update` on every machine. The plist is
// read back with plutil rather than trusted, because bundle.sh fills it with
// sed and a sed that matched nothing exits 0.
func TestThePresenceIsStampedWithTheTag(t *testing.T) {
	job := presenceJob(t, "release.yml")
	at := -1
	for i, step := range job.Steps {
		if strings.Contains(step.Run, "plutil") {
			at = i
			break
		}
	}
	if at < 0 {
		t.Fatalf("release.yml's %s job never runs plutil; nothing would hold the bundle's version against the tag", presenceJobName)
	}
	run := job.Steps[at].Run
	for _, want := range []string{plistVersionKey, "$TAG", "exit 1"} {
		if !strings.Contains(run, want) {
			t.Errorf("release.yml's plutil step never says %q; the stamp is the tag without its v, read back and refused when it is anything else", want)
		}
	}
}

// TestThePresenceJobProvesTheZipRoundTripKeepsTheSignature holds the one claim
// the publishing job cannot check for itself.
//
// The bundle is signed ad-hoc on macOS, then unzipped, copied and zipped again
// by an ubuntu runner that has no codesign at all, and a colleague's machine
// is where an invalid signature would first be noticed, as an app that will
// not open. So the macOS job does the round trip itself, with the same zip the
// pack step uses, and verifies the signature on the other side of it.
func TestThePresenceJobProvesTheZipRoundTripKeepsTheSignature(t *testing.T) {
	for _, file := range []string{"release.yml", "go.yml"} {
		script := jobScript(presenceJob(t, file))
		zipped := strings.Index(script, packZip)
		if zipped < 0 {
			t.Errorf(".github/workflows/%s's %s job never zips the bundle with %q; the round trip the pack step does would go unmeasured", file, presenceJobName, packZip)
			continue
		}
		if !strings.Contains(script, "unzip") {
			t.Errorf(".github/workflows/%s's %s job zips the bundle and never unzips it; half a round trip proves nothing", file, presenceJobName)
		}
		verified := strings.Index(script, verifySignature)
		if verified < 0 {
			t.Errorf(".github/workflows/%s's %s job never runs %q", file, presenceJobName, verifySignature)
			continue
		}
		if verified < zipped {
			t.Errorf(".github/workflows/%s's %s job verifies the signature before the round trip; bundle.sh already did that, and what is in question is the signature afterwards", file, presenceJobName)
		}
	}
}

// TestTheCIBuildsThePresenceOnEveryPush: release.yml runs on tags, and a tag
// is pushed by a person or at 02:00 UTC by the nightly. A Swift file that does
// not compile would therefore be found by a release, which is the one build
// nobody is watching. go.yml runs on every push, so the same job lives there
// too, without the upload, because nothing downloads it.
func TestTheCIBuildsThePresenceOnEveryPush(t *testing.T) {
	w := readWorkflow(t, "go.yml")
	if len(w.On.Push.Branches) == 0 {
		t.Error("go.yml runs on no push branches; the Swift build would wait for a tag like everything in release.yml")
	}

	job := presenceJob(t, "go.yml")
	if !strings.HasPrefix(job.RunsOn, "macos-") {
		t.Errorf("go.yml's %s job runs on %q; a Swift app bundle is built on macOS", presenceJobName, job.RunsOn)
	}
	script := jobScript(job)
	for _, want := range []string{"make app-lint", "make app-test", "presence/bundle.sh"} {
		if !strings.Contains(script, want) {
			t.Errorf("go.yml's %s job never runs %q; the push build is the release's build or it guards nothing", presenceJobName, want)
		}
	}
	if jobUses(job, "actions/upload-artifact") {
		t.Errorf("go.yml's %s job uploads an artifact; nothing downloads it, and a push is not a release", presenceJobName)
	}
}

// stagePresence is the folder inside the zip the bundle is copied into, and
// darwinArm the one case arm that copy is allowed to sit in. packZip and
// verifySignature are the two lines the round trip is made of, and
// plistVersionKey the key the stamp is read back from. Every one of them is a
// literal, because a test that reads the thing it checks follows it wherever
// somebody moves it.
const (
	stagePresence   = `$stage/presence/`
	darwinArm       = "darwin-*)"
	packZip         = "zip -qr"
	verifySignature = "codesign --verify --deep --strict"
	plistVersionKey = "CFBundleShortVersionString"
)

// TestTheNightlyCutsThePatchAndOnlyWhenMainMoved holds the one thing about
// the nightly that matters to a person who is asleep: it is quiet. A night
// where nothing landed cuts no tag, so the releases page stays a list of
// changes rather than a list of dates.
func TestTheNightlyCutsThePatchAndOnlyWhenMainMoved(t *testing.T) {
	w := readWorkflow(t, "nightly.yml")

	if len(w.On.Schedule) != 1 || w.On.Schedule[0].Cron != nightlyCron {
		t.Errorf("nightly.yml runs on schedule %v; the plan says %q, which is 02:00 UTC", w.On.Schedule, nightlyCron)
	}
	if len(w.On.WorkflowDispatch.Inputs) == 0 {
		t.Error("nightly.yml takes no workflow_dispatch input; the dry run is how somebody asks what tag it would cut")
	}

	script := workflowScript(t, w)
	for _, want := range []string{
		// The last tag is what the next one is counted from.
		"git describe --tags --abbrev=0",
		// Nothing landed since it, and the run says so and stops.
		"rev-list",
		// Only the patch moves. The minor is `make tag`'s and the nightly
		// cutting one would collide with the next release Nail cuts by hand.
		"patch + 1",
		// main as it stands, tagged. What the tag does not move is
		// TestOnlyMakeTagMovesThePluginVersion's.
		"git tag",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("nightly.yml never says %q", want)
		}
	}
	if !strings.Contains(script, "dry") {
		t.Error("nightly.yml's script never reads the dry run input; the dry run would tag for real")
	}
}

// TestTheNightlyPublishesThroughTheReleaseWorkflow: a tag pushed with
// GITHUB_TOKEN triggers no workflow, so the nightly cannot push a tag and walk
// away. It calls release.yml, which is why that file takes a tag input at all.
func TestTheNightlyPublishesThroughTheReleaseWorkflow(t *testing.T) {
	w := readWorkflow(t, "nightly.yml")

	var caller workflowJobFields
	for _, job := range w.Jobs {
		if job.Uses != "" {
			caller = job
		}
	}
	if caller.Uses != "./.github/workflows/release.yml" {
		t.Fatalf("nightly.yml calls %q; the release it cut is published by ./.github/workflows/release.yml", caller.Uses)
	}
	if caller.With["tag"] == "" {
		t.Error("nightly.yml calls the release workflow with no tag; it would build whatever main is rather than the tag it cut")
	}
	if caller.Secrets != "inherit" {
		t.Errorf("nightly.yml passes secrets %q to the release workflow; it needs inherit, or the build carries no client secret", caller.Secrets)
	}
	if w.Permissions["contents"] != "write" {
		t.Errorf("nightly.yml has contents permission %q; it pushes the tag it cut", w.Permissions["contents"])
	}
}

// oauthSecret is the repository secret the release build links into the
// binary. It is named here as a literal rather than read out of the Makefile,
// because a test that reads the name it checks follows it wherever somebody
// moves it.
const oauthSecret = "GDOC_OAUTH_CLIENT_SECRET"

// nightlyCron is 02:00 UTC, stated as the literal the workflow carries.
const nightlyCron = "0 2 * * *"

// readWorkflow parses one file under .github/workflows.
func readWorkflow(t *testing.T, name string) workflowFile {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", name))
	if err != nil {
		t.Fatal(err)
	}
	var w workflowFile
	if err := yaml.Unmarshal(b, &w); err != nil {
		t.Fatalf(".github/workflows/%s does not parse: %v", name, err)
	}
	if w.Name == "" {
		t.Errorf(".github/workflows/%s carries no name; it is what the Actions tab shows", name)
	}
	if len(w.Jobs) == 0 {
		t.Fatalf(".github/workflows/%s holds no job", name)
	}
	return w
}

// releaseCreate is the one line that publishes. It is what tells the jobs in
// release.yml apart: the job that says it is the release, and every other job
// is something the release waits for.
const releaseCreate = "gh release create"

// releasePublisher is the name of the job in release.yml that publishes, and
// its fields. release.yml held one job until the presence app arrived, which is
// built on macOS because codesign and lipo live there; the publishing job is
// found by what it does rather than by what it is called.
func releasePublisher(t *testing.T, w workflowFile) (string, workflowJobFields) {
	t.Helper()
	for name, job := range w.Jobs {
		for _, step := range job.Steps {
			if strings.Contains(step.Run, releaseCreate) {
				return name, job
			}
		}
	}
	t.Fatalf("no job in release.yml runs %q; the zips would be built and left on the runner", releaseCreate)
	return "", workflowJobFields{}
}

// releaseSteps is the steps of the publishing job in release.yml, in order, so
// a test can ask what comes before what.
func releaseSteps(t *testing.T) []workflowStep {
	t.Helper()
	name, job := releasePublisher(t, readWorkflow(t, "release.yml"))
	if len(job.Steps) == 0 {
		t.Fatalf("release.yml's %s job holds no step", name)
	}
	return job.Steps
}

// presenceJobName is the job that builds the app, in both workflows, written
// out because the publishing job names it in needs and a rename in one place
// is a release that waits for a job nobody runs.
const presenceJobName = "presence"

// presenceJob is that job in one workflow.
func presenceJob(t *testing.T, file string) workflowJobFields {
	t.Helper()
	w := readWorkflow(t, file)
	job, ok := w.Jobs[presenceJobName]
	if !ok {
		t.Fatalf(".github/workflows/%s holds no %s job; nothing would build the app", file, presenceJobName)
	}
	if len(job.Steps) == 0 {
		t.Fatalf(".github/workflows/%s's %s job holds no step", file, presenceJobName)
	}
	return job
}

// jobScript is every line of shell in one job, joined in step order.
func jobScript(job workflowJobFields) string {
	var runs []string
	for _, step := range job.Steps {
		runs = append(runs, step.Run)
	}
	return strings.Join(runs, "\n")
}

// jobUses answers whether a job has a step using an action, by prefix, so a
// version bump is not a failure.
func jobUses(job workflowJobFields, action string) bool {
	for _, step := range job.Steps {
		if strings.HasPrefix(step.Uses, action) {
			return true
		}
	}
	return false
}

// caseArm is the text of one arm of a shell case statement: from the pattern
// to the ;; that closes it. A test reads it to ask what a platform-specific
// step does for that platform and not for the others.
func caseArm(script, pattern string) (string, bool) {
	at := strings.Index(script, pattern)
	if at < 0 {
		return "", false
	}
	rest := script[at+len(pattern):]
	if end := strings.Index(rest, ";;"); end >= 0 {
		return rest[:end], true
	}
	return rest, true
}

// releaseScript is every line of shell in release.yml, joined.
func releaseScript(t *testing.T) string {
	t.Helper()
	return workflowScript(t, readWorkflow(t, "release.yml"))
}

// workflowScript is every run: block in a workflow, joined in no particular
// order. Tests that care about order read the steps instead.
func workflowScript(t *testing.T, w workflowFile) string {
	t.Helper()
	var runs []string
	for _, job := range w.Jobs {
		for _, step := range job.Steps {
			runs = append(runs, step.Run)
		}
	}
	return strings.Join(runs, "\n")
}

// stepSaying is the index of the first step whose shell says something, or -1.
func stepSaying(steps []workflowStep, want string) int {
	for i, step := range steps {
		if strings.Contains(step.Run, want) {
			return i
		}
	}
	return -1
}

// contains is the one-line membership test these workflow lists need.
func contains(list []string, want string) bool {
	for _, got := range list {
		if got == want {
			return true
		}
	}
	return false
}
