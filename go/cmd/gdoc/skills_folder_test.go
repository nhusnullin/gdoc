// propose stopped creating a working copy in v2.8.0, and its --folder flag is
// accepted and ignored for one release so an old skill keeps working. These
// tests hold the two skills that propose to the new shape: neither passes the
// flag, and both tell a session what to do with the two answers that replaced
// the probe's, a stopped run and a lost answer.

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// proposingSkills are the files a session reads before it runs propose.
var proposingSkills = []string{
	"gdoc-review/propose.md",
	"gdoc-align/SKILL.md",
}

// folderFlag is the flag as a call passes it, spaced or joined. A longer flag
// that only starts with the same letters, --folder-id say, is not it.
var folderFlag = regexp.MustCompile(`--folder([ =]|$)`)

func readSkill(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(skillsDir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestNoSkillPassesAFolderToPropose(t *testing.T) {
	for _, rel := range proposingSkills {
		src := readSkill(t, rel)
		if loc := folderFlag.FindStringIndex(src); loc != nil {
			t.Errorf("%s still passes --folder: %q. propose ignores it since v2.8.0", rel, src[loc[0]:loc[1]])
		}
		if regexp.MustCompile(`\benrolled\b`).MatchString(src) {
			t.Errorf("%s still reads enrolled, which propose no longer reports", rel)
		}
	}
}

func TestTheFolderPatternIgnoresALongerFlag(t *testing.T) {
	for in, want := range map[string]bool{
		"--folder 1AbC":    true,
		"--folder=1AbC":    true,
		"--folder":         true,
		"--folder-id 1AbC": false,
		"--folders":        false,
	} {
		if got := folderFlag.MatchString(in); got != want {
			t.Errorf("folderFlag.MatchString(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestTheProposingSkillsNameTheStopAndALostAnswer(t *testing.T) {
	for _, rel := range proposingSkills {
		src := squashSpace(readSkill(t, rel))
		for _, want := range []string{
			`outcome: "unknown"`,
			"could not confirm",
			"`suggestions` before proposing anything again",
		} {
			if !regexp.MustCompile(regexp.QuoteMeta(want)).MatchString(src) {
				t.Errorf("%s does not say %q", rel, want)
			}
		}
	}
}
