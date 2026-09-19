// The skills read against the table: every call a SKILL.md tells a session to
// make names a command this binary answers, with flags that command takes.
//
// A skill is instructions in a file nothing compiles, so a command renamed here
// leaves a session running a call that fails at the terminal. This test is the
// thing that notices. It reads the markdown, not the binary's output, so it
// costs no network and no token.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// skillsDir is where the skills live, from this package. Three steps up is the
// repository root, because this file sits in go/cmd/gdoc.
const skillsDir = "../../../skills"

// skillCall is one call a skill tells a session to make: the words after the
// binary and the flags on the same line, with where it was found so a failure
// names a file and a line.
type skillCall struct {
	file  string
	line  int
	words []string
	flags []string
}

// skillFiles is every SKILL.md under the skills directory, sorted, so a failure
// reads the same way twice.
func skillFiles(dir string) ([]string, error) {
	found, err := filepath.Glob(filepath.Join(dir, "*", "SKILL.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(found)
	return found, nil
}

// skillCalls finds the calls in one skill's markdown. Inside a code fence a
// call is a line opening with the binary, written either way. Outside one it is
// an inline code span opening with $GDOC, the name a skill calls the binary by,
// because prose puts a warning gdoc printed in backticks too and that is not a
// call. A flag standing on a line of its own, which is what a continued shell
// line looks like, is not a call either.
func skillCalls(file, src string) []skillCall {
	lines := strings.Split(src, "\n")
	prose := make([]string, len(lines))
	fenced := false
	var calls []skillCall
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			if call, ok := parseSkillCall(file, i+1, line, "gdoc", "$GDOC"); ok {
				calls = append(calls, call)
			}
			continue
		}
		prose[i] = line
	}
	for _, span := range inlineCodeSpans(strings.Join(prose, "\n")) {
		if call, ok := parseSkillCall(file, span.line, span.text, "$GDOC"); ok {
			calls = append(calls, call)
		}
	}
	sort.Slice(calls, func(i, j int) bool { return calls[i].line < calls[j].line })
	return calls
}

// codeSpan is what stood between two backticks, and the line the opening one
// was on.
type codeSpan struct {
	line int
	text string
}

// inlineCodeSpans is every backtick pair in the prose. A span is allowed to
// wrap, because a sentence in a skill wraps at the same width as any other, and
// a call cut in half by a line break is still a call.
func inlineCodeSpans(text string) []codeSpan {
	var spans []codeSpan
	line, open, openLine := 1, -1, 0
	for i, r := range text {
		switch r {
		case '\n':
			line++
		case '`':
			if open < 0 {
				open, openLine = i+1, line
				continue
			}
			spans = append(spans, codeSpan{line: openLine, text: text[open:i]})
			open = -1
		}
	}
	return spans
}

// parseSkillCall reads one line as a call. The first word has to be one of the
// names given, exactly, so the front matter key `gdoc:` and the sentence "gdoc
// wrote it" are both left alone. A call with no word after the binary, such as
// the bare `gdoc` that says what the binary is, asks about no command and is
// skipped.
func parseSkillCall(file string, line int, text string, binaries ...string) (skillCall, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return skillCall{}, false
	}
	named := false
	for _, b := range binaries {
		if fields[0] == b {
			named = true
		}
	}
	if !named {
		return skillCall{}, false
	}
	call := skillCall{file: file, line: line}
	for _, field := range fields[1:] {
		if strings.HasPrefix(field, "--") {
			call.flags = append(call.flags, field)
			continue
		}
		if strings.HasPrefix(field, "-") {
			continue
		}
		call.words = append(call.words, field)
	}
	if len(call.words) == 0 {
		return skillCall{}, false
	}
	return call, true
}

// skillProblems is every way a call does not match the table, in words a person
// can act on. An empty list is a skill whose calls the binary all answers.
func skillProblems(calls []skillCall) []string {
	var problems []string
	for _, call := range calls {
		c := match(call.words)
		if c == nil {
			problems = append(problems, fmt.Sprintf("%s:%d: %q is no command gdoc answers. %s",
				call.file, call.line, strings.Join(call.words, " "), usageLine()))
			continue
		}
		takes := c.flagSet()
		for _, f := range call.flags {
			if _, asked := helpAsked([]string{f}); asked {
				continue
			}
			if _, ok := takes[f]; !ok {
				problems = append(problems, fmt.Sprintf("%s:%d: %s does not take %s",
					call.file, call.line, c.name, f))
			}
		}
	}
	return problems
}

func TestEverySkillNamesOnlyCommandsAndFlagsTheBinaryHas(t *testing.T) {
	files, err := skillFiles(skillsDir)
	if err != nil {
		t.Fatalf("looking for the skills: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no SKILL.md under %s. A moved skills directory is a failure, not an empty pass", skillsDir)
	}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		calls := skillCalls(file, string(src))
		if len(calls) == 0 {
			t.Errorf("%s tells a session to call gdoc nowhere this test can see", file)
		}
		for _, problem := range skillProblems(calls) {
			t.Error(problem)
		}
	}
}

func TestASkillNamingAFlagTheCommandDoesNotTakeIsCaught(t *testing.T) {
	// Arrange
	src := "```bash\n$GDOC publish --md note.md --tone friendly\n```\n"

	// Act
	problems := skillProblems(skillCalls("SKILL.md", src))

	// Assert
	if len(problems) != 1 {
		t.Fatalf("want one problem, got %v", problems)
	}
	if !strings.Contains(problems[0], "publish does not take --tone") {
		t.Errorf("the problem does not name the flag: %s", problems[0])
	}
}

func TestASkillNamingACommandTheBinaryLostIsCaught(t *testing.T) {
	// Arrange
	src := "```bash\n$GDOC sing <url>\n```\n"

	// Act
	problems := skillProblems(skillCalls("SKILL.md", src))

	// Assert
	if len(problems) != 1 {
		t.Fatalf("want one problem, got %v", problems)
	}
	if !strings.Contains(problems[0], `"sing <url>" is no command`) {
		t.Errorf("the problem does not name the words: %s", problems[0])
	}
}

func TestNoSkillFilesIsAFailureAndNotAnEmptyPass(t *testing.T) {
	// Arrange
	dir := t.TempDir()

	// Act
	files, err := skillFiles(dir)

	// Assert
	if err != nil {
		t.Fatalf("looking in an empty directory: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("want no files under an empty directory, got %v", files)
	}
}

func TestProseAboutGdocIsNotReadAsACall(t *testing.T) {
	// Arrange
	src := strings.Join([]string{
		"A note carries a `gdoc:` block, and gdoc wrote it.",
		"The run warns `gdoc cannot withdraw what it did not propose`.",
		"`gdoc` is the v2 binary, on PATH.",
		"```bash",
		"$GDOC propose <url> \\",
		"  --from /tmp/changes.json",
		"```",
	}, "\n")

	// Act
	calls := skillCalls("SKILL.md", src)

	// Assert
	if len(calls) != 1 {
		t.Fatalf("want the one real call, got %v", calls)
	}
	if calls[0].words[0] != "propose" {
		t.Errorf("want the propose call, got %v", calls[0].words)
	}
	if problems := skillProblems(calls); len(problems) != 0 {
		t.Errorf("want no problems, got %v", problems)
	}
}

// A skill ships to a colleague's machine, so it may not name the person who
// wrote it, may not point at a checkout only one machine has, and may not fetch
// a new binary on its own. It also says which gdoc it needs, because a skill and
// a binary travel apart: the plugin updates on Claude Code's toggle and the
// binary updates when somebody types `gdoc update`.

// skillFrontMatter is the lines between the opening fence and the closing one.
// A file that does not open with `---` has no front matter, and that is a
// failure rather than an empty answer: Claude Code reads that block first.
func skillFrontMatter(src string) ([]string, bool) {
	lines := strings.Split(src, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return lines[1:i], true
		}
	}
	return nil, false
}

// skillNeeds is the version on the front matter's `needs:` line, as three
// integers. It reads `needs: v2.0.0` and nothing looser: a version with a word
// in it, or with a piece missing, is no version and says so, because a skill
// that cannot say which binary it needs cannot check one.
func skillNeeds(src string) ([3]int, bool) {
	front, ok := skillFrontMatter(src)
	if !ok {
		return [3]int{}, false
	}
	for _, line := range front {
		rest, found := strings.CutPrefix(strings.TrimSpace(line), "needs:")
		if !found {
			continue
		}
		return parseSkillVersion(strings.TrimSpace(rest))
	}
	return [3]int{}, false
}

// parseSkillVersion reads `vX.Y.Z` into its three numbers. The binary's own
// comparison lives elsewhere; this is the test reading a line a person wrote,
// so it is strict about the shape and says nothing about what the numbers mean.
func parseSkillVersion(text string) ([3]int, bool) {
	digits, found := strings.CutPrefix(text, "v")
	if !found {
		return [3]int{}, false
	}
	parts := strings.Split(digits, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var version [3]int
	for i, part := range parts {
		if part == "" {
			return [3]int{}, false
		}
		n := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				return [3]int{}, false
			}
			n = n*10 + int(r-'0')
		}
		version[i] = n
	}
	return version, true
}

func TestNoSkillNamesAPersonOrAMachinesPath(t *testing.T) {
	files, err := skillFiles(skillsDir)
	if err != nil {
		t.Fatalf("looking for the skills: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no SKILL.md under %s. A moved skills directory is a failure, not an empty pass", skillsDir)
	}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		for _, banned := range []string{"Nail", "/Users/", "~/src/"} {
			if strings.Contains(string(src), banned) {
				t.Errorf("%s names %q. A skill runs on a colleague's machine: address whoever is at the keyboard, and carry no path into a checkout", file, banned)
			}
		}
	}
}

func TestEverySkillNamesTheGdocItNeeds(t *testing.T) {
	files, err := skillFiles(skillsDir)
	if err != nil {
		t.Fatalf("looking for the skills: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no SKILL.md under %s. A moved skills directory is a failure, not an empty pass", skillsDir)
	}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		if _, ok := skillNeeds(string(src)); !ok {
			t.Errorf("%s carries no front matter line `needs: vX.Y.Z`. A skill says which binary it needs, because the two travel apart", file)
		}
	}
}

func TestNoSkillRunsUpdateOnItsOwn(t *testing.T) {
	files, err := skillFiles(skillsDir)
	if err != nil {
		t.Fatalf("looking for the skills: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no SKILL.md under %s. A moved skills directory is a failure, not an empty pass", skillsDir)
	}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		for _, call := range skillCalls(file, string(src)) {
			if call.words[0] == "update" {
				t.Errorf("%s:%d tells a session to run update. An update happens when a person types it and never otherwise", file, call.line)
			}
		}
	}
}

func TestANeedsLineIsReadAsThreeNumbers(t *testing.T) {
	// Arrange
	src := "---\nname: gdoc-review\nneeds: v2.10.3\n---\n\n# A skill\n"

	// Act
	version, ok := skillNeeds(src)

	// Assert
	if !ok {
		t.Fatalf("want the needs line read, got nothing")
	}
	if version != [3]int{2, 10, 3} {
		t.Errorf("want 2.10.3, got %v", version)
	}
}

func TestANeedsLineThatIsNotAVersionIsCaught(t *testing.T) {
	// Arrange
	bad := []string{
		"---\nneeds: 2.0.0\n---\n",
		"---\nneeds: v2.0\n---\n",
		"---\nneeds: vlatest\n---\n",
		"---\nneeds: v2.0.x\n---\n",
		"---\nneeds: v2..0\n---\n",
		"---\nname: gdoc-review\n---\n",
		"name: gdoc-review\nneeds: v2.0.0\n",
	}

	// Act and assert
	for _, src := range bad {
		if version, ok := skillNeeds(src); ok {
			t.Errorf("want %q refused, got %v", src, version)
		}
	}
}

func TestASkillRunningUpdateIsSeenAsACall(t *testing.T) {
	// Arrange
	src := "```bash\n$GDOC update --nightly\n```\n"

	// Act
	calls := skillCalls("SKILL.md", src)

	// Assert
	if len(calls) != 1 {
		t.Fatalf("want the one call, got %v", calls)
	}
	if calls[0].words[0] != "update" {
		t.Errorf("want the update call seen, got %v", calls[0].words)
	}
}

// A skill's `needs` line names a stable release, `vX.Y.0`, and not a nightly.
// `gdoc update` installs a stable, and `gdoc update --nightly` is the channel a
// person opts into by typing it, so a skill that needs `v2.1.3` sends a
// colleague after a binary the plain command never brings.

// skillNeedsProblem is what is wrong with one skill's `needs` line, in words a
// person can act on, or the empty string when the line names a stable release.
func skillNeedsProblem(file, src string) string {
	version, ok := skillNeeds(src)
	if !ok {
		return fmt.Sprintf("%s carries no front matter line `needs: vX.Y.Z`. A skill says which binary it needs, because the two travel apart", file)
	}
	if version[2] != 0 {
		return fmt.Sprintf("%s needs v%d.%d.%d, which is a nightly. A skill names a stable release, vX.Y.0, because that is what `gdoc update` installs",
			file, version[0], version[1], version[2])
	}
	return ""
}

func TestEverySkillNeedsAStableRelease(t *testing.T) {
	files, err := skillFiles(skillsDir)
	if err != nil {
		t.Fatalf("looking for the skills: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no SKILL.md under %s. A moved skills directory is a failure, not an empty pass", skillsDir)
	}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		if problem := skillNeedsProblem(file, string(src)); problem != "" {
			t.Error(problem)
		}
	}
}

func TestANeedsLineNamingANightlyIsCaughtAndNamed(t *testing.T) {
	// Arrange
	src := "---\nname: gdoc-review\nneeds: v2.1.3\n---\n\n# A skill\n"

	// Act
	problem := skillNeedsProblem("skills/gdoc-review/SKILL.md", src)

	// Assert
	if problem == "" {
		t.Fatalf("want v2.1.3 refused, got no problem")
	}
	if !strings.Contains(problem, "skills/gdoc-review/SKILL.md") || !strings.Contains(problem, "v2.1.3") {
		t.Errorf("the problem names neither the file nor the version: %s", problem)
	}
	if problem := skillNeedsProblem("skills/gdoc-review/SKILL.md", "---\nneeds: v2.1.0\n---\n"); problem != "" {
		t.Errorf("want a stable release accepted, got %s", problem)
	}
}

// The repository carries five skills, and the shape of each one's front matter
// is what Claude Code reads before it reads a word of the instructions. The
// tests below hold the list and the shape.
//
// Five, because two answer "align" and "publish" between them and a sixth that
// nobody linked would be a folder in the plugin and nothing on a machine. The
// list is written here as literals: a test that reads skills/ to learn what
// skills there are cannot notice one that went missing.

// skillWants is what one skill's front matter has to say: the folder it lives
// in, and the release it needs. The name and the description are checked for
// every skill and are not in the table, because they are required of all five
// and a table column of "yes" five times says nothing.
var skillWants = []struct {
	folder string
	needs  string
}{
	{"gdoc-align", "v2.4.0"},
	{"gdoc-export", "v2.4.0"},
	{"gdoc-publish", "v2.4.0"},
	{"gdoc-restyle", "v2.0.0"},
	{"gdoc-review", "v2.4.0"},
}

// skillFrontMatterValue is what the front matter says after this key, or the
// empty string when it says nothing. It is a line read, not a YAML parse: the
// three keys this file asks about are one line each, and a parser here would be
// a second decoder of a file Claude Code decodes itself.
func skillFrontMatterValue(src, key string) string {
	front, ok := skillFrontMatter(src)
	if !ok {
		return ""
	}
	for _, line := range front {
		rest, found := strings.CutPrefix(strings.TrimSpace(line), key+":")
		if !found {
			continue
		}
		return strings.TrimSpace(rest)
	}
	return ""
}

func TestTheRepositoryCarriesTheFiveSkillsAndEachSaysWhatItNeeds(t *testing.T) {
	files, err := skillFiles(skillsDir)
	if err != nil {
		t.Fatalf("looking for the skills: %v", err)
	}
	var folders []string
	for _, file := range files {
		folders = append(folders, filepath.Base(filepath.Dir(file)))
	}
	var want []string
	for _, w := range skillWants {
		want = append(want, w.folder)
	}
	if strings.Join(folders, " ") != strings.Join(want, " ") {
		t.Fatalf("skills/ holds [%s] and this milestone says [%s]. A skill nobody links is a folder in the plugin and nothing on a machine",
			strings.Join(folders, " "), strings.Join(want, " "))
	}

	for _, w := range skillWants {
		path := filepath.Join(skillsDir, w.folder, "SKILL.md")
		b, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reading %s: %v", path, err)
			continue
		}
		src := string(b)
		if got := skillFrontMatterValue(src, "name"); got != w.folder {
			t.Errorf("%s says name %q and lives in %s. Claude Code reads the name, and the two have to be one word", path, got, w.folder)
		}
		if skillFrontMatterValue(src, "description") == "" {
			t.Errorf("%s carries no description. That sentence is the whole of what decides whether the skill fires", path)
		}
		if got := skillFrontMatterValue(src, "needs"); got != w.needs {
			t.Errorf("%s needs %q and this milestone says %s. A skill and a binary travel apart, so the floor is the release whose behaviour the skill was written against", path, got, w.needs)
		}
	}
}

// A skill with disable-model-invocation on fires only when somebody types its
// name, and a session with gdoc on PATH can run any command by hand. So the
// flag does not stop the action: it skips the asking, which is the one thing
// the skills exist to do. Decision 17 of the milestone 13 specification.
func TestNoSkillRefusesToBeInvokedByTheModel(t *testing.T) {
	files, err := skillFiles(skillsDir)
	if err != nil {
		t.Fatalf("looking for the skills: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no SKILL.md under %s. A moved skills directory is a failure, not an empty pass", skillsDir)
	}
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		front, ok := skillFrontMatter(string(b))
		if !ok {
			t.Errorf("%s opens with no front matter block", file)
			continue
		}
		for _, line := range front {
			if strings.HasPrefix(strings.TrimSpace(line), "disable-model-invocation:") {
				t.Errorf("%s carries %q. Every skill starts from a colleague's sentence, and a skill that does not fire only skips the asking", file, strings.TrimSpace(line))
			}
		}
	}
}

// The marker rules are written once and read by two skills. gdoc-export owns
// the file, because it is the skill that first puts markers in front of a
// person, and gdoc-align points at it rather than holding a second copy: two
// copies of an escaping rule are two rules, and the one that drifts resolves a
// marker wrongly.
func TestTheMarkerRulesAreOneFileAndAlignNamesIt(t *testing.T) {
	const markers = "markers.md"
	path := filepath.Join(skillsDir, "gdoc-export", markers)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s is not there: %v. It is the one copy of the marker rules", path, err)
	}
	b, err := os.ReadFile(filepath.Join(skillsDir, "gdoc-align", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), markers) {
		t.Errorf("skills/gdoc-align/SKILL.md names no %s. It resolves markers, and the rules for them live beside gdoc-export's SKILL.md", markers)
	}
}
