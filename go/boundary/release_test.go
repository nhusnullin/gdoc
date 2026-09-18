// The guard over what a colleague unpacks.
//
// release/install.sh is the one file that runs on a machine nobody here has
// seen. It travels inside the zip, so it cannot read release/platforms or
// skills/ from a stranger's machine and carries its own copy of both lists.
// Two copies of one list is the shape that rots quietly, so the copies are
// compared here on every commit, and so are the file names the shell builds
// against the ones internal/update builds.

package boundary

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gdoc/internal/update"
)

// distPair is how `make dist` names a platform in its recipe. The platforms
// file and the installer both spell it GOOS-GOARCH, which is what
// update.Platform builds from a running binary.
var distPair = regexp.MustCompile(`GOOS=([a-z0-9]+)\s+GOARCH=([a-z0-9]+)`)

// TestTheInstallerOffersEveryPlatformTheReleaseCarries holds the two lists
// together and holds both against the Makefile. release/platforms is what the
// release workflow walks to pack zips; the PLATFORMS array inside install.sh is
// what a colleague's machine is matched against. A platform in the file and not
// the array is a zip nobody can install, and one in the array and not the file
// is a download that 404s on the machine that needs it most.
func TestTheInstallerOffersEveryPlatformTheReleaseCarries(t *testing.T) {
	listed, comments := releasePlatforms(t)
	if len(listed) == 0 {
		t.Fatal("release/platforms names no platform; a release would carry no zip at all")
	}

	offered := shellArray(t, releaseInstaller(t), "PLATFORMS=(")
	if strings.Join(offered, " ") != strings.Join(listed, " ") {
		t.Errorf("release/install.sh offers [%s] and release/platforms carries [%s]; the two are the same list",
			strings.Join(offered, " "), strings.Join(listed, " "))
	}

	built := distPlatforms(t)
	for _, p := range listed {
		if !built[p] {
			t.Errorf("release/platforms names %s and `make dist` builds no binary for it, so that zip could not be packed", p)
		}
	}

	// Windows is built and not offered, and the file has to say why, because a
	// colleague on Windows reading it otherwise finds a silence. When Windows
	// joins the list this check retires itself.
	windows := false
	for _, p := range listed {
		if strings.HasPrefix(p, "windows-") {
			windows = true
		}
	}
	if !windows && !strings.Contains(strings.ToLower(comments), "windows") {
		t.Error("release/platforms offers no windows platform and its comments do not say why; a colleague on Windows reads that file and finds nothing")
	}
}

// TestTheReleaseInstallerCopiesEverySkillThePluginShips is the same
// disappearance rule TestTheInstallerLinksEverySkillThePluginShips holds over
// the developer install. --skills copies the folders the zip carries, and the
// zip carries skills/. A skill in one list and not the other means a colleague
// on --skills and a colleague on the plugin have different tools under one name.
func TestTheReleaseInstallerCopiesEverySkillThePluginShips(t *testing.T) {
	copied := shellArray(t, releaseInstaller(t), "SKILLS=(")
	shipped := skillFolders(t)
	if strings.Join(copied, " ") != strings.Join(shipped, " ") {
		t.Errorf("release/install.sh copies [%s] and the plugin ships [%s]; the two lists are the same list",
			strings.Join(copied, " "), strings.Join(shipped, " "))
	}
}

// TestTheReleaseInstallerNamesTheFilesTheUpdaterNames pins the shell against
// the Go. `gdoc update` and this script fetch the same two files out of the
// same release, and each builds their names itself: the updater from
// update.AssetName and update.ChecksumsName, the script from its own tag and
// platform variables. If the release workflow ever renames an asset, one of the
// two is fixed and the other is forgotten, and the one that gets forgotten is
// the one no test runs.
func TestTheReleaseInstallerNamesTheFilesTheUpdaterNames(t *testing.T) {
	script := releaseInstaller(t)
	// The script's variables are named tag and platform, so the names it builds
	// are these strings with the shell's dollar signs still in them.
	for _, want := range []string{
		update.AssetName("$tag", "$platform"),
		update.ChecksumsName("$tag"),
	} {
		if !strings.Contains(script, want) {
			t.Errorf("release/install.sh builds no %q; that is the name internal/update asks the release for, so the two would fetch different files", want)
		}
	}
}

// TestTheReleaseInstallerNeverEditsTheZshrc holds the one promise the script
// makes about a file it did not write. It prints the line to add and reads the
// file to see whether the line is there already. Every mention of .zshrc is
// therefore inside a printf, a grep or a comment, and a line that is none of
// those is the script editing somebody's shell configuration.
func TestTheReleaseInstallerNeverEditsTheZshrc(t *testing.T) {
	for i, line := range strings.Split(releaseInstaller(t), "\n") {
		if !strings.Contains(line, ".zshrc") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || strings.Contains(line, "printf") || strings.Contains(line, "grep") {
			continue
		}
		t.Errorf("release/install.sh line %d names .zshrc outside a printf, a grep or a comment: %s", i+1, trimmed)
	}
}

// releaseInstaller is the script this file judges.
func releaseInstaller(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "release", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// releasePlatforms reads release/platforms: the platform lines sorted, so they
// compare as a set against the installer's array, and the comment lines joined,
// which the Windows check reads.
func releasePlatforms(t *testing.T) ([]string, string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "release", "platforms"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	var comments []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
		case strings.HasPrefix(line, "#"):
			comments = append(comments, line)
		default:
			names = append(names, line)
		}
	}
	sort.Strings(names)
	return names, strings.Join(comments, "\n")
}

// distPlatforms is every GOOS-GOARCH pair the dist target builds a binary for.
func distPlatforms(t *testing.T) map[string]bool {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	recipe, ok := makeRecipe(string(b), "dist")
	if !ok {
		t.Fatal("the Makefile holds no dist target")
	}
	built := make(map[string]bool)
	for _, m := range distPair.FindAllStringSubmatch(recipe, -1) {
		built[update.Platform(m[1], m[2])] = true
	}
	if len(built) == 0 {
		t.Fatal("the dist target builds nothing this test could read")
	}
	return built
}

// shellArray reads one bash array literal out of a script, sorted, so the two
// lists compare as sets rather than as whatever order somebody typed.
func shellArray(t *testing.T, script, opening string) []string {
	t.Helper()
	_, rest, ok := strings.Cut(script, opening)
	if !ok {
		t.Fatalf("release/install.sh holds no %s array; this test reads it to compare against the repository", opening)
	}
	list, _, ok := strings.Cut(rest, ")")
	if !ok {
		t.Fatalf("release/install.sh's %s array is not closed", opening)
	}
	names := strings.Fields(list)
	sort.Strings(names)
	return names
}

// releaseREADMECeiling is the line count the colleague's README stays under.
// It is the first and often the only gdoc document a colleague reads, on
// GitHub and inside the zip, which carry the same file. The page installs,
// shows the three things to try with the words to type, explains the preview
// API in a table, and lists every command so a colleague can build a skill of
// their own. Longer than that belongs in `gdoc help`, which cannot go stale,
// or in `docs/guide/`.
const releaseREADMECeiling = 200

// TestTheReleaseREADMEIsUnderTheCeiling holds the length and the house writing
// rule. The em dash is checked here rather than trusted, because this file is
// read by people outside this repository and the rule is easy to lose in an
// edit somebody made in a hurry.
func TestTheReleaseREADMEIsUnderTheCeiling(t *testing.T) {
	text := releaseREADME(t)
	if lines := len(strings.Split(strings.TrimRight(text, "\n"), "\n")); lines > releaseREADMECeiling {
		t.Errorf("README.md is %d lines and the ceiling is %d; what does not fit goes into `gdoc help`", lines, releaseREADMECeiling)
	}
	if i := strings.Index(text, "—"); i >= 0 {
		t.Errorf("README.md carries an em dash at byte %d; the house rule is a comma, a colon or a full stop", i)
	}
}

// TestTheReleaseREADMENamesEveryRouteToTheSkills asks that the three policy
// routes are all there. A colleague whose machine refuses a marketplace and
// finds only the `/plugin` commands has no way on, and the route they need is
// the one nobody tests by hand because this machine is not restricted.
func TestTheReleaseREADMENamesEveryRouteToTheSkills(t *testing.T) {
	text := releaseREADME(t)
	for _, want := range []string{
		// The one line that installs the binary, and the update that is
		// never taken for anyone.
		"release/install.sh",
		"gdoc update",
		"gdoc auth login",
		// Route one, plugins open.
		"/plugin marketplace add nhusnullin/gdoc",
		"/plugin install altery@gdoc",
		// Route two, marketplaces refused and local skills still loading.
		"--skills global",
		// Route three, nothing but plugins and managed settings.
		"extraKnownMarketplaces",
		"enabledPlugins",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("README.md never says %q, and a colleague on that route has nowhere to go", want)
		}
	}
	for _, skill := range skillFolders(t) {
		if !strings.Contains(text, skill) {
			t.Errorf("README.md never names the %s skill, which the plugin installs", skill)
		}
	}
}

// TestTheIssueTemplateAsksForTheFourFacts holds the report form. A report
// missing the version is a report nobody can place against a release, and a
// report missing the object gdoc printed is a guess about what happened.
func TestTheIssueTemplateAsksForTheFourFacts(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, ".github", "ISSUE_TEMPLATE", "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{"gdoc help", "version", "command", "object", "expected"} {
		if !strings.Contains(strings.ToLower(text), want) {
			t.Errorf(".github/ISSUE_TEMPLATE/report.md never asks for %q", want)
		}
	}
	if i := strings.Index(text, "—"); i >= 0 {
		t.Errorf(".github/ISSUE_TEMPLATE/report.md carries an em dash at byte %d", i)
	}
}

// releaseREADME is the README the zip carries, which is the repository's own
// README.md. One file, so GitHub and the zip never drift apart.
func releaseREADME(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
