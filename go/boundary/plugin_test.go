// The guard over what a colleague installs.
//
// A colleague never clones this repository. They add it as a Claude Code
// marketplace and install one plugin from it, and what they get is whatever
// the two manifests at the root say plus whatever sits under skills/. Nobody
// reads those files on the way past, so a half made skill or a version that
// disagrees with the next tag reaches somebody else's machine unnoticed. This
// asks on every commit instead.

package boundary

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// pluginManifest is the part of .claude-plugin/plugin.json this test judges.
// Claude Code reads more; what matters here is the name a colleague types
// after the at sign and the version they report a bug against.
type pluginManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// marketplaceManifest is the part of .claude-plugin/marketplace.json this test
// judges. One plugin, sourced from the repository root, because the skills sit
// at the root beside the manifests.
type marketplaceManifest struct {
	Name    string `json:"name"`
	Plugins []struct {
		Name   string `json:"name"`
		Source string `json:"source"`
	} `json:"plugins"`
}

// tagShape is the version both the manifest and `make tag` speak: a leading v
// and three integers. The leading v is there because the version in this file
// is the tag the release carries, and reading it with a v means nobody has to
// remember which of the two spellings a given line wants.
var tagShape = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// TestThePluginNamesTheSkillsThatExist: the plugin a colleague installs is the
// three skill folders and nothing else, so the manifests and those folders
// have to agree. A directory under skills/ with no SKILL.md is a skill Claude
// Code fails to load, and it fails on the colleague's machine rather than here.
func TestThePluginNamesTheSkillsThatExist(t *testing.T) {
	var plugin pluginManifest
	readJSON(t, filepath.Join(repoRoot, ".claude-plugin", "plugin.json"), &plugin)

	if plugin.Name != "gdoc" {
		t.Errorf("plugin.json names the plugin %q; it is what a colleague types after the at sign, so it is gdoc", plugin.Name)
	}
	if plugin.Description == "" {
		t.Error("plugin.json carries no description; it is the one line /plugin shows before anyone installs")
	}
	if !tagShape.MatchString(plugin.Version) {
		t.Errorf("plugin.json carries version %q, which is not vX.Y.Z; the release workflow checks it against the tag", plugin.Version)
	}

	var market marketplaceManifest
	readJSON(t, filepath.Join(repoRoot, ".claude-plugin", "marketplace.json"), &market)

	if market.Name != "gdoc" {
		t.Errorf("marketplace.json names the marketplace %q; it is what a colleague types after the at sign, so it is gdoc", market.Name)
	}
	if len(market.Plugins) != 1 {
		t.Fatalf("marketplace.json lists %d plugins; this repository is one plugin", len(market.Plugins))
	}
	if market.Plugins[0].Name != plugin.Name {
		t.Errorf("marketplace.json lists plugin %q and plugin.json names %q; a colleague installing the first gets the second", market.Plugins[0].Name, plugin.Name)
	}
	if market.Plugins[0].Source != "./" {
		t.Errorf("marketplace.json sources the plugin from %q; the manifests and skills/ sit at the root, so it is ./", market.Plugins[0].Source)
	}

	for _, skill := range skillFolders(t) {
		path := filepath.Join(repoRoot, "skills", skill, "SKILL.md")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("skills/%s has no SKILL.md; the plugin would ship a skill Claude Code fails to load", skill)
		}
	}
}

// TestTheInstallerLinksEverySkillThePluginShips is the other direction, the
// same disappearance rule the wire and module lists hold. install.sh names the
// skills it links in one array, and that array is a second list of the same
// thing. A skill in one and not the other means a colleague on the plugin and
// a person in the checkout have different tools under the same name.
func TestTheInstallerLinksEverySkillThePluginShips(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	const opening = "SKILLS=("
	_, rest, ok := strings.Cut(string(b), opening)
	if !ok {
		t.Fatalf("install.sh holds no %s array; this test reads it to compare against skills/", opening)
	}
	list, _, ok := strings.Cut(rest, ")")
	if !ok {
		t.Fatal("install.sh's SKILLS array is not closed")
	}
	linked := strings.Fields(list)
	sort.Strings(linked)

	shipped := skillFolders(t)
	if strings.Join(linked, " ") != strings.Join(shipped, " ") {
		t.Errorf("install.sh links [%s] and the plugin ships [%s]; the two lists are the same list",
			strings.Join(linked, " "), strings.Join(shipped, " "))
	}
}

// skillFolders is every directory under skills/, sorted. It is what the plugin
// ships: Claude Code discovers a plugin's skills by walking that directory, so
// nothing lists them and nothing can be out of date with them.
func skillFolders(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repoRoot, "skills"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("skills/ holds no skill; the plugin would ship nothing")
	}
	sort.Strings(names)
	return names
}

// readJSON reads the fields this test judges and leaves the rest to Claude
// Code, which knows more keys than these structs name. Refusing an unknown key
// here would refuse every field Claude Code adds after today, and this test is
// not the authority on that file's shape: `claude plugin validate` is.
func readJSON(t *testing.T, path string, into any) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	if err := dec.Decode(into); err != nil {
		t.Fatalf("%s does not parse: %v", path, err)
	}
}

// TestMakeTagRefusesAVersionThatIsNotAMinor pins the guard inside the tag
// target rather than trusting the person typing it. Nightly versions are CI's,
// and a patch tag cut by hand would collide with the one the nightly cuts at
// 02:00. The check is a shell condition in the Makefile, so this reads it
// there: the target has to refuse before it writes, commits, tags or pushes.
func TestMakeTagRefusesAVersionThatIsNotAMinor(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	recipe, ok := makeRecipe(string(b), "tag")
	if !ok {
		t.Fatal("the Makefile holds no tag target")
	}
	for _, want := range []string{"v[0-9]", `\.0$`, "plugin.json", "git tag", "git push"} {
		if !strings.Contains(recipe, want) {
			t.Errorf("the tag target does not mention %q; it refuses a version that is not x.y.0, writes it into plugin.json, tags and pushes", want)
		}
	}
	// The refusal comes before anything is written, or a rejected version
	// leaves plugin.json edited and the tree dirty.
	refusal := strings.Index(recipe, `\.0$`)
	write := strings.Index(recipe, "plugin.json")
	if refusal > write {
		t.Error("the tag target writes plugin.json before it checks the version; a refused version would leave the tree edited")
	}
}

// makeRecipe returns the recipe lines of one target, which are the lines
// beginning with a tab after its rule line.
func makeRecipe(makefile, target string) (string, bool) {
	var recipe []string
	found, inTarget := false, false
	for _, line := range strings.Split(makefile, "\n") {
		if strings.HasPrefix(line, "\t") {
			if inTarget {
				recipe = append(recipe, line)
			}
			continue
		}
		inTarget = strings.HasPrefix(line, target+":")
		found = found || inTarget
	}
	return strings.Join(recipe, "\n"), found
}

// TestTheVersionStampNamesOnlyATag pins what the Makefile writes into
// main.version, which is the other half of releaseVersion's rule.
//
// `gdoc` prints its version in every envelope, and the skills gate on it: an
// older binary than the skill needs is a run that stops and says `gdoc update`
// fixes it, and no version at all is a build made from source, which is not an
// error. Both branches need the stamp to be a release tag or the sentinel and
// nothing in between. `git describe --always` gives a bare commit hash when
// there is no tag, which is neither: a skill cannot compare it to a tag and a
// colleague cannot fetch it, and the source-build branch becomes unreachable.
// `--dirty` alone is the same problem in a subtler spelling, since v2.0.0-dirty
// parses as a pre-release below v2.0.0 and sorts as older than the tag it was
// built from.
//
// This reads the assignment rather than running it, for TestMakeTag's reason:
// running it would describe whatever tree the test happens to sit in. It reads
// the assignments themselves rather than any line that mentions git describe,
// because the comment above them says the same words, and prose that satisfies
// a test is a test a regressed assignment walks past.
func TestTheVersionStampNamesOnlyATag(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	assignment := func(name string) string {
		re := regexp.MustCompile(`(?m)^` + name + `\s*:?=(.*)$`)
		m := re.FindStringSubmatch(string(b))
		if m == nil {
			return ""
		}
		return strings.TrimSpace(m[1])
	}

	stamp := assignment("DESCRIBED")
	if stamp == "" {
		t.Fatal("the Makefile has no DESCRIBED assignment; the version has to come from one git describe this test can read")
	}
	if !strings.Contains(stamp, "git describe") {
		t.Fatalf("DESCRIBED is %q and it has to be a git describe; the version comes from the tag", stamp)
	}
	if !strings.Contains(stamp, "--exact-match") {
		t.Errorf("the version stamp %q does not ask for an exact match, so a checkout between tags names a release it is not", stamp)
	}
	if strings.Contains(stamp, "--always") {
		t.Errorf("the version stamp %q falls back to a commit hash, which names no release: a build with no tag has to keep the dev sentinel", stamp)
	}

	// The chain the rule is about: DESCRIBED reaches VERSION, VERSION reaches
	// main.version, and a described string that is not a clean tag becomes the
	// sentinel on the way.
	version := assignment("VERSION")
	if version == "" || !strings.Contains(version, "$(DESCRIBED)") {
		t.Errorf("VERSION is %q and it has to be built from $(DESCRIBED); a second git describe is a second rule", version)
	}
	if !strings.Contains(version, "dev") {
		t.Errorf("VERSION is %q and it never falls back to dev, so releaseVersion's empty branch is unreachable", version)
	}
	// A dirty tag is not that release either, so the described string cannot
	// reach main.version unfiltered.
	if strings.Contains(stamp, "--dirty") && !strings.Contains(version, "%-dirty") {
		t.Errorf("the version stamp %q keeps --dirty and VERSION %q does not filter it, so v2.0.0-dirty would ship as a pre-release below v2.0.0", stamp, version)
	}
	if ldflags := assignment("LDFLAGS"); !strings.Contains(ldflags, "-X main.version=$(VERSION)") {
		t.Errorf("LDFLAGS is %q and it does not stamp $(VERSION) into main.version; nothing would reach the envelope", ldflags)
	}
}
