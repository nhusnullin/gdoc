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
			Tags []string `yaml:"tags"`
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
	Uses    string            `yaml:"uses"`
	With    map[string]string `yaml:"with"`
	Secrets string            `yaml:"secrets"`
	Needs   string            `yaml:"needs"`
	Steps   []workflowStep    `yaml:"steps"`
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
	if !strings.Contains(script, "gh release create") {
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

// releaseSteps is the steps of the one job in release.yml, in order, so a test
// can ask what comes before what.
func releaseSteps(t *testing.T) []workflowStep {
	t.Helper()
	w := readWorkflow(t, "release.yml")
	if len(w.Jobs) != 1 {
		t.Fatalf("release.yml holds %d jobs; it is one build that packs and publishes what it built", len(w.Jobs))
	}
	for _, job := range w.Jobs {
		if len(job.Steps) == 0 {
			t.Fatal("release.yml's job holds no step")
		}
		return job.Steps
	}
	return nil
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
