// The skills read against the Agent Skills specification at agentskills.io:
// the front matter keys it allows, the limits it sets on each, and the budget
// it gives the SKILL.md body. The first two decide whether another agent loads
// the skill at all. The last decides what it costs every session that does.
//
// A file beside a SKILL.md is loaded only when the SKILL.md tells a session to
// read it, so every such file is named there, and every call in one is read
// against the table exactly as the SKILL.md calls are.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// specKeys are the only top-level front matter keys the specification allows.
// Anything a skill wants to add beyond them goes under `metadata`.
var specKeys = map[string]bool{
	"name":          true,
	"description":   true,
	"license":       true,
	"compatibility": true,
	"metadata":      true,
	"allowed-tools": true,
}

// The specification's limits, stated as literals for the house-style reason:
// a test that read them from somewhere would follow them wherever they moved.
const (
	specNameMax          = 64
	specDescriptionMax   = 1024
	specCompatibilityMax = 500
	// The body budget is 500 lines and 5,000 tokens. Tokens are counted here
	// as characters over four, which is the ceiling a person can check with
	// wc -c and is close to what the tokenizer gives for English prose.
	specBodyLinesMax = 500
	specBodyCharsMax = 20000
)

var specName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// skillTopLevelKeys is every key the front matter sets at its own level, in
// the order written. A line that opens with a space or a tab belongs to the key
// above it, which is how `needs` sits under `metadata`.
func skillTopLevelKeys(front []string) []string {
	var keys []string
	for _, line := range front {
		if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '#' {
			continue
		}
		key, _, found := strings.Cut(line, ":")
		if found {
			keys = append(keys, strings.TrimSpace(key))
		}
	}
	return keys
}

// skillSpecProblems is every way one SKILL.md's front matter breaks the
// specification, in words a person can act on.
func skillSpecProblems(file, folder, src string) []string {
	front, ok := skillFrontMatter(src)
	if !ok {
		return []string{fmt.Sprintf("%s opens with no front matter block", file)}
	}
	var problems []string
	for _, key := range skillTopLevelKeys(front) {
		if !specKeys[key] {
			problems = append(problems, fmt.Sprintf("%s sets %q at the top of its front matter. The Agent Skills specification allows only name, description, license, compatibility, metadata and allowed-tools, and another agent refuses the skill; put it under metadata", file, key))
		}
	}
	name := skillFrontMatterValue(src, "name")
	if len(name) > specNameMax || !specName.MatchString(name) || name != folder {
		problems = append(problems, fmt.Sprintf("%s is named %q. A name is lowercase letters, digits and single hyphens, at most %d, and the same word as its folder %q", file, name, specNameMax, folder))
	}
	if n := len(skillFrontMatterValue(src, "description")); n == 0 || n > specDescriptionMax {
		problems = append(problems, fmt.Sprintf("%s has a description of %d characters. It is between 1 and %d, and it is the whole of what decides whether the skill fires", file, n, specDescriptionMax))
	}
	if n := len(skillFrontMatterValue(src, "compatibility")); n == 0 || n > specCompatibilityMax {
		problems = append(problems, fmt.Sprintf("%s has a compatibility line of %d characters. Every skill here needs the gdoc binary and Google's network, so it says so, in at most %d", file, n, specCompatibilityMax))
	}
	for _, key := range []string{"name", "description", "compatibility"} {
		value := skillFrontMatterValue(src, key)
		if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, "'") {
			continue
		}
		if strings.Contains(value, ": ") || strings.Contains(value, " #") {
			problems = append(problems, fmt.Sprintf("%s has %q or %q inside its unquoted %s. YAML reads the first as a new key and the second as a comment, so the front matter does not parse and no agent loads the skill: reword it", file, ": ", " #", key))
		}
	}
	return problems
}

func TestEverySkillFrontMatterKeepsToTheAgentSkillsSpec(t *testing.T) {
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
		for _, problem := range skillSpecProblems(file, filepath.Base(filepath.Dir(file)), string(src)) {
			t.Error(problem)
		}
	}
}

func TestATopLevelKeyOutsideTheSpecIsCaughtAndMetadataIsNot(t *testing.T) {
	// Arrange
	outside := "---\nname: gdoc-review\ndescription: Use when.\ncompatibility: Needs gdoc.\nneeds: v2.4.0\n---\n"
	inside := "---\nname: gdoc-review\ndescription: Use when.\ncompatibility: Needs gdoc.\nmetadata:\n  needs: v2.4.0\n---\n"

	// Act
	caught := skillSpecProblems("SKILL.md", "gdoc-review", outside)
	passed := skillSpecProblems("SKILL.md", "gdoc-review", inside)

	// Assert
	if len(caught) != 1 || !strings.Contains(caught[0], `"needs"`) {
		t.Errorf("want the top-level needs named, got %v", caught)
	}
	if len(passed) != 0 {
		t.Errorf("want needs under metadata accepted, got %v", passed)
	}
	if _, ok := skillNeeds(inside); !ok {
		t.Errorf("want the needs line still read under metadata")
	}
}

func TestAColonOrAHashInAnUnquotedValueIsCaught(t *testing.T) {
	// Arrange
	bad := []string{
		"---\nname: gdoc-review\ndescription: Use when.\ncompatibility: Needs gdoc: on PATH.\n---\n",
		"---\nname: gdoc-review\ndescription: Use when #live is said.\ncompatibility: Needs gdoc.\n---\n",
	}
	quoted := "---\nname: gdoc-review\ndescription: Use when.\ncompatibility: \"Needs gdoc: on PATH.\"\n---\n"

	// Act and assert
	for _, src := range bad {
		if problems := skillSpecProblems("SKILL.md", "gdoc-review", src); len(problems) != 1 {
			t.Errorf("want %q refused once, got %v", src, problems)
		}
	}
	if problems := skillSpecProblems("SKILL.md", "gdoc-review", quoted); len(problems) != 0 {
		t.Errorf("want a quoted value accepted, got %v", problems)
	}
}

func TestANameThatIsNotItsFolderOrNotLowercaseIsCaught(t *testing.T) {
	// Arrange
	bad := map[string]string{
		"Gdoc-Review":  "gdoc-review",
		"gdoc--review": "gdoc-review",
		"-gdoc-review": "gdoc-review",
		"gdoc-align":   "gdoc-review",
	}

	// Act and assert
	for name, folder := range bad {
		src := "---\nname: " + name + "\ndescription: Use when.\ncompatibility: Needs gdoc.\n---\n"
		if problems := skillSpecProblems("SKILL.md", folder, src); len(problems) != 1 {
			t.Errorf("want the name %q in %s refused once, got %v", name, folder, problems)
		}
	}
}

func TestEverySkillBodyFitsTheSpecBudget(t *testing.T) {
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
		lines := strings.Count(string(src), "\n")
		if lines >= specBodyLinesMax || len(src) > specBodyCharsMax {
			t.Errorf("%s is %d lines and %d characters. The whole file loads on every run, so it stays under %d lines and %d characters, and what only some runs need moves to a file beside it that the SKILL.md says when to read",
				file, lines, len(src), specBodyLinesMax, specBodyCharsMax)
		}
	}
}

// skillCompanions is every file in a skill's folder other than its SKILL.md,
// by folder, sorted.
func skillCompanions(dir string) (map[string][]string, error) {
	found, err := filepath.Glob(filepath.Join(dir, "*", "*"))
	if err != nil {
		return nil, err
	}
	sort.Strings(found)
	companions := map[string][]string{}
	for _, path := range found {
		if filepath.Base(path) == "SKILL.md" {
			continue
		}
		folder := filepath.Base(filepath.Dir(path))
		companions[folder] = append(companions[folder], path)
	}
	return companions, nil
}

// A file nobody names is never read, and a name with no file is a session
// told to read something that is not there.
func TestEveryFileBesideASkillIsNamedByIt(t *testing.T) {
	companions, err := skillCompanions(skillsDir)
	if err != nil {
		t.Fatalf("looking for the skills: %v", err)
	}
	if len(companions["gdoc-review"]) == 0 {
		t.Fatalf("no file beside skills/gdoc-review/SKILL.md. Live mode and proposing live beside it, and a moved file is a failure, not an empty pass")
	}
	for folder, paths := range companions {
		b, err := os.ReadFile(filepath.Join(skillsDir, folder, "SKILL.md"))
		if err != nil {
			t.Fatalf("reading %s's SKILL.md: %v", folder, err)
		}
		for _, path := range paths {
			if !strings.Contains(string(b), "`"+filepath.Base(path)+"`") {
				t.Errorf("%s sits beside skills/%s/SKILL.md, which never names it in backticks, so no session reads it", path, folder)
			}
		}
	}
}

// Every call in a markdown file beside a skill is read against the table the
// way a SKILL.md's are, and carries no person and no machine's path either.
// A companion file may hold no call at all, which a SKILL.md may not.
func TestEveryFileBesideASkillNamesOnlyWhatTheBinaryHas(t *testing.T) {
	companions, err := skillCompanions(skillsDir)
	if err != nil {
		t.Fatalf("looking for the skills: %v", err)
	}
	for _, paths := range companions {
		for _, path := range paths {
			if filepath.Ext(path) != ".md" {
				continue
			}
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", path, err)
			}
			for _, problem := range skillProblems(skillCalls(path, string(src))) {
				t.Error(problem)
			}
			for _, call := range skillCalls(path, string(src)) {
				if call.words[0] == "update" {
					t.Errorf("%s:%d tells a session to run update. An update happens when a person types it and never otherwise", path, call.line)
				}
			}
			for _, banned := range []string{"Nail", "/Users/", "~/src/"} {
				if strings.Contains(string(src), banned) {
					t.Errorf("%s names %q. A skill runs on a colleague's machine: address whoever is at the keyboard, and carry no path into a checkout", path, banned)
				}
			}
		}
	}
}
