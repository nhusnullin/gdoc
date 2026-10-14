// This file holds what the colleague's pages say about the mark. The mark is a
// setting since v2 milestone 16, so a page that names one character as the mark
// is a page that goes stale the moment somebody sets another.

package boundary

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// markPages are the pages a colleague reads outside the code: the README that
// GitHub and the release zip both carry, and the guide pages it links to. The
// skills under skills/ are deliberately absent: they name the robot as one of
// the three known marks a gdoc reply may open with, which is a read rule and
// not the mark.
var markPages = []string{"README.md", "docs/guide"}

// TestNoColleaguePageNamesTheRobotAsTheMark reads those pages and fails on the
// robot character. 🤖 was the mark every release up to v2.10.0 wrote, and the
// pages told a colleague to type it into a reply and to keep it out of a reason.
// gdoc writes the mark itself now and the mark in effect is a setting, so a
// sentence naming that one character is wrong twice over: nobody types it, and
// it is not what a run writes unless somebody asked for it.
//
// It stays a known mark, which is why internal/plaintext names it and this scan
// does not reach the code.
func TestNoColleaguePageNamesTheRobotAsTheMark(t *testing.T) {
	var found []string
	for _, path := range markPageFiles(t) {
		b, err := os.ReadFile(filepath.Join(repoRoot, path))
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "🤖") {
				found = append(found, path+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
			}
		}
	}
	if len(found) > 0 {
		t.Errorf("the mark is a setting and gdoc writes it itself, and these lines still name the robot as the mark:\n%s", strings.Join(found, "\n"))
	}
}

// TestTheREADMESaysWhereTheMarkIsSetAndWhatTurningItOffCosts asks the one page
// a colleague is sure to read for the four facts about the mark: what it is,
// where it is set, the two flags that override the file, and the spelling that
// turns it off. The last one is here because the empty mark is the one setting
// whose cost is invisible until a later review, which reads gdoc's own replies
// as a person's.
func TestTheREADMESaysWhereTheMarkIsSetAndWhatTurningItOffCosts(t *testing.T) {
	text := releaseREADME(t)
	for _, want := range []string{
		// The default, stated as the character a reader will see.
		"⁘",
		// Where a colleague writes another one down.
		"settings.yaml",
		// The two flags, which win over the file.
		"--config",
		"--mark",
		// The empty mark, in the spelling that asks for it.
		`--mark ""`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("README.md never says %q, so a colleague cannot tell which mark gdoc signs with or how to change it", want)
		}
	}
}

// TestTheWritingPageNamesBothWaysToGiveAReplyBody holds the page the review
// skill's author reads. A reply body arrives on the line or in a file since
// milestone 16, and chat sends it on the line.
func TestTheWritingPageNamesBothWaysToGiveAReplyBody(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "docs/guide/writing.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{"--body ", "--body-file"} {
		if !strings.Contains(text, want) {
			t.Errorf("docs/guide/writing.md never says %q, and a skill author reading it has one of the two spellings", want)
		}
	}
}

// markPageFiles expands markPages into the paths really read: a file as itself,
// a directory as every .md file under it. A named path that is not there fails,
// because a scan of nothing passes quietly.
func markPageFiles(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, entry := range markPages {
		full := filepath.Join(repoRoot, entry)
		info, err := os.Stat(full)
		if err != nil {
			t.Fatalf("%s is named by the scan and is not there: %v", entry, err)
		}
		if !info.IsDir() {
			out = append(out, entry)
			continue
		}
		err = filepath.WalkDir(full, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || filepath.Ext(p) != ".md" {
				return nil
			}
			rel, err := filepath.Rel(repoRoot, p)
			if err != nil {
				return err
			}
			out = append(out, rel)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(out)
	return out
}
