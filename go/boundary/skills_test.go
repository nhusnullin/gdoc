// The guard over how a skill tells a person to update.
//
// When help says a newer gdoc is published, or a skill needs a newer one than
// is installed, the session says so and names the command. A command inside a
// sentence is text a person reads and types again. A command alone in a fenced
// bash block is a row the desktop app gives a Run button. This reads the five
// skills and asks for the block, in the same words in each.

package boundary

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// updateNoticeOpeners are the first words of the three paragraphs that make up
// the update notice: the needs paragraph that stops a session on an older
// binary, the remark about a newer release, and the block both of them give.
var updateNoticeOpeners = []string{
	"`help` is the first call of every session",
	"The same object may also carry `update`",
	"The command goes in its own fenced code block tagged `bash`",
}

// updateNoticeMusts are the phrases the notice cannot lose. Each is one of the
// promises the block makes to the person clicking Run: it is a bash block, it
// holds one command with no prompt in front, and that command is the plain
// update or the major one.
var updateNoticeMusts = []string{
	"one short sentence",
	"fenced code block tagged `bash`",
	"no `$` prompt",
	"nothing else",
	"`gdoc update`",
	"`gdoc update --major`",
	"The session never runs it.",
}

// skillParagraphs is the markdown split on blank lines, each paragraph with
// its line breaks folded into single spaces, so a rewrap is not a difference.
func skillParagraphs(src string) []string {
	var paragraphs []string
	for _, block := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n\n") {
		folded := strings.Join(strings.Fields(block), " ")
		if folded != "" {
			paragraphs = append(paragraphs, folded)
		}
	}
	return paragraphs
}

// updateNotice is the three paragraphs of one skill that tell a session what to
// say about a version, joined, or what is wrong with them.
func updateNotice(src string) (string, error) {
	paragraphs := skillParagraphs(src)
	var found []string
	for _, opener := range updateNoticeOpeners {
		var hits []string
		for _, p := range paragraphs {
			if strings.HasPrefix(p, opener) {
				hits = append(hits, p)
			}
		}
		if len(hits) != 1 {
			return "", fmt.Errorf("want one paragraph opening %q, found %d", opener, len(hits))
		}
		found = append(found, hits[0])
	}
	notice := strings.Join(found, "\n\n")
	var missing []string
	for _, must := range updateNoticeMusts {
		if !strings.Contains(notice, must) {
			missing = append(missing, must)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("the update notice does not say %q", missing)
	}
	if strings.ContainsRune(notice, '\u2014') {
		return "", fmt.Errorf("the update notice carries an em dash")
	}
	return notice, nil
}

// TestEverySkillGivesTheUpdateAsARunnableBlock: each skill tells a session to
// put the update command alone in a fenced bash block, and the five say it in
// the same words, so no skill drifts back to naming the command in prose.
func TestEverySkillGivesTheUpdateAsARunnableBlock(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot, "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no SKILL.md under skills/. A moved skills directory is a failure, not an empty pass")
	}
	sort.Strings(files)

	notices := map[string][]string{}
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		notice, err := updateNotice(string(b))
		if err != nil {
			t.Errorf("%s: %v. A command inside a sentence is text to retype; alone in a bash block it is a Run button", file, err)
			continue
		}
		notices[notice] = append(notices[notice], file)
	}
	if len(notices) > 1 {
		var groups []string
		for _, group := range notices {
			groups = append(groups, "["+strings.Join(group, " ")+"]")
		}
		sort.Strings(groups)
		t.Errorf("the skills tell a session about an update in %d different ways: %s. One notice, in the same words in each", len(notices), strings.Join(groups, " "))
	}
}

// TestAnUpdateNoticeWithTheCommandInProseIsCaught is the notice as it read
// before the block: the command named inside a sentence, which the test above
// has to refuse.
func TestAnUpdateNoticeWithTheCommandInProseIsCaught(t *testing.T) {
	// Arrange
	src := "`help` is the first call of every session. An older binary is one\n" +
		"the skill is ahead of: say that `gdoc update` fixes it, and stop.\n\n" +
		"The same object may also carry `update`. Say once that a newer gdoc is\n" +
		"published and name what installs it: `gdoc update`, or\n" +
		"`gdoc update --major` for a major.\n"

	// Act
	_, err := updateNotice(src)

	// Assert
	if err == nil {
		t.Fatal("want the prose notice refused, got it accepted")
	}
}
