// The review skill is split in two: SKILL.md holds what a session runs, and
// review.md holds how it judges what came back. A session loads SKILL.md on
// every run, so the rules that only matter once a thread is in front of it pay
// for themselves on every run they are not read. These tests hold the split:
// the rules landed in review.md exactly once, the calls stayed in SKILL.md, and
// the step numbers live.md and propose.md point at did not move.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// reviewCore is the file the rules moved into, and reviewSkill is the file that
// kept the calls.
const (
	reviewCore  = "review.md"
	reviewSkill = "SKILL.md"
)

// squashSpace is one line of text with every run of whitespace made a single
// space, so a rule that wraps in the file is still found by the sentence a test
// states.
func squashSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// readReviewFile is one file of the review skill, with its whitespace squashed,
// beside the source as written.
func readReviewFile(t *testing.T, name string) (raw, flat string) {
	t.Helper()
	path := filepath.Join(skillsDir, "gdoc-review", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s is not there: %v. It is where the review rules live", path, err)
	}
	return string(b), squashSpace(string(b))
}

// The core is read by a session that has already run the calls. A call in it is
// a second copy of a call, and two copies of a call are two calls, so the one
// that drifts sends something the skill no longer means. Plain English that
// names a command word is not a call: the rules talk about replying and reading
// all the way through.
func TestTheReviewCoreCarriesNoCall(t *testing.T) {
	// Arrange
	raw, flat := readReviewFile(t, reviewCore)

	// Act
	calls := skillCalls(filepath.Join(skillsDir, "gdoc-review", reviewCore), raw)

	// Assert
	if len(calls) != 0 {
		t.Errorf("%s holds %d calls, the first at line %d. The calls live in %s", reviewCore, len(calls), calls[0].line, reviewSkill)
	}
	if strings.Contains(raw, "```") {
		t.Errorf("%s holds a fenced block. A fence is where a session reads a command to run, and this file holds rules", reviewCore)
	}
	if strings.Contains(raw, "--") {
		t.Errorf("%s holds a flag token. A flag belongs to a call, and the calls live in %s", reviewCore, reviewSkill)
	}
	for _, word := range []string{"reply", "read"} {
		if !strings.Contains(flat, word) {
			t.Errorf("%s never says %q. The rules are plain English about replying and reading, and naming a command word is not a call", reviewCore, word)
		}
	}
}

// A file nobody is told to read is never read. The core is read before the
// first thread, because the first judgement a session makes is whether a thread
// is work at all.
func TestTheReviewSkillNamesItsCore(t *testing.T) {
	// Arrange
	_, flat := readReviewFile(t, reviewSkill)

	// Act and assert
	if !strings.Contains(flat, "`"+reviewCore+"`") {
		t.Errorf("%s never names `%s`. A session that is not told to read it never reads it", reviewSkill, reviewCore)
	}
	if !strings.Contains(flat, "Before the first thread") {
		t.Errorf("%s does not say when to read `%s`. It is read before the first thread, because deciding whether a thread is work is the first judgement of the run", reviewSkill, reviewCore)
	}
}

// reviewRules is the rule text that moved, one sentence per section, stated as
// a literal so a test does not follow a rule wherever somebody rewords it.
var reviewRules = []string{
	// Step 2: Decide which threads still need an answer
	"The binary reports facts and judges nothing. There is no `handled` field, on purpose.",
	"A resolved thread is not work. Resolving is the operator's, and they do it when they accept the answer.",
	// Before acting on an old marked comment again
	"A run can be cut off between an action and its receipt. So a marked comment with no receipt does not prove the work is undone.",
	// Two messages at most
	"A thread carries an acknowledgment and a receipt per piece of work, and nothing else.",
	"Never a progress feed. The margin is the readers' room, and the terminal is where the operator's record goes.",
	// When to stop and ask
	"Judge the draft before it is posted. Stop, show it, and wait when any of these is true:",
	"Nobody outside Altery should learn something from a gdoc reply that they could not learn from the document.",
	// If the request is for all the comments
	"By default only marked comments are work.",
	"Never batch-approve in this mode.",
	// Never
	"Never edit the document. Every change to its words is a suggestion, and the guard refuses anything else on a document that was handed in.",
	"Never resolve or reopen a thread. Resolving means the answer was accepted, and only the operator accepts.",
	"Never post a reply that is not printed in full afterwards. That printing is the record.",
}

// A rule in both files is two rules. A rule in neither is a rule the split
// dropped, and a dropped rule is the one way this refactor can post something
// the skill means to stop.
func TestEveryReviewRuleLandedOnce(t *testing.T) {
	// Arrange
	_, core := readReviewFile(t, reviewCore)
	_, skill := readReviewFile(t, reviewSkill)

	// Act and assert
	for _, rule := range reviewRules {
		want := squashSpace(rule)
		if !strings.Contains(core, want) {
			t.Errorf("%s does not hold the rule %q. The split moves rules and drops none", reviewCore, want)
		}
		if strings.Contains(skill, want) {
			t.Errorf("%s still holds the rule %q. A rule in both files is two rules, and the one that drifts is the one that is wrong", reviewSkill, want)
		}
	}
}

// reviewSteps are the step headings in the order a run walks them. live.md
// points at Step 3 and Step 8, and propose.md at Step 7, so a step that loses
// its number leaves those files pointing at nothing.
var reviewSteps = []string{
	"## Step 1: Read the document",
	"## Step 2: Decide which threads still need an answer",
	"## Step 3: Say what the run found",
	"## Step 4: Ground the answer",
	"## Step 5: Answer an `ai?`",
	"## Step 6: Carry out an `ai!`",
	"## Step 7: Propose a change to the document",
	"## Step 8: Report",
}

func TestTheReviewStepsKeepTheirNumbers(t *testing.T) {
	// Arrange
	raw, _ := readReviewFile(t, reviewSkill)

	// Act and assert
	at := 0
	for _, step := range reviewSteps {
		found := strings.Index(raw[at:], step)
		if found < 0 {
			t.Fatalf("%s has no heading %q after the step above it. live.md and propose.md point at these numbers", reviewSkill, step)
		}
		at += found + len(step)
	}
	// A step whose rules moved keeps its heading and one line pointing at its
	// section of the core.
	step2 := raw[strings.Index(raw, reviewSteps[1]):strings.Index(raw, reviewSteps[2])]
	if !strings.Contains(step2, reviewCore) {
		t.Errorf("Step 2 in %s does not point at %s. Its rules moved, so the heading has to say where they went", reviewSkill, reviewCore)
	}
}

// reviewReadAgainRules is the side-by-side guard that runs last: the thread as
// it is now, read with the draft already written. Stated as literals, because
// the rule is the wording and not the idea.
var reviewReadAgainRules = []string{
	"read the thread again just before the reply goes out",
	"A 🤖 reply appeared since that read, and this session did not write it: do not post.",
	"A 🤖 reply this session wrote itself never stops it.",
}

// Two answers to one thread is the failure both front doors can cause. A chat
// and a live Claude Code session read the same thread, think, and post, and the
// thinking is the gap. The only thing that closes it is reading the thread once
// more before the reply goes out. A session's own acknowledgment is not another
// answer, so it never stops the receipt that is its other half.
func TestTheCoreReadsTheThreadAgainBeforePosting(t *testing.T) {
	// Arrange
	_, core := readReviewFile(t, reviewCore)
	_, skill := readReviewFile(t, reviewSkill)

	// Act and assert
	for _, rule := range reviewReadAgainRules {
		want := squashSpace(rule)
		if !strings.Contains(core, want) {
			t.Errorf("%s does not hold %q. Without it two front doors answer one thread twice", reviewCore, want)
		}
		if strings.Contains(skill, want) {
			t.Errorf("%s holds %q too. A rule in both files is two rules, and this one reads the same through either front door", reviewSkill, want)
		}
	}
}

// reviewLiveRules divide the work while a live session runs: the marker says
// whose it is, and not knowing is a question for the person.
var reviewLiveRules = []string{
	"While a live session runs, marked comments are its work.",
	"A chat leaves them alone, and says so in one line.",
	"Ask when you do not know whether one runs.",
}

// One document, one token, and two sessions that can both see the marker. The
// live session is the one with a hub, so the marked comments are its work. A
// chat that guesses answers a comment the live session is already carrying out.
func TestTheCoreLeavesMarkedCommentsToALiveSession(t *testing.T) {
	// Arrange
	_, core := readReviewFile(t, reviewCore)
	_, skill := readReviewFile(t, reviewSkill)

	// Act and assert
	for _, rule := range reviewLiveRules {
		want := squashSpace(rule)
		if !strings.Contains(core, want) {
			t.Errorf("%s does not hold %q. A chat that does not leave marked comments alone answers them beside the live session", reviewCore, want)
		}
		if strings.Contains(skill, want) {
			t.Errorf("%s holds %q too, and the guard belongs to both front doors at once", reviewSkill, want)
		}
	}
}

// reviewAnnotateFlow is the flow for a comment on words the person names: find
// them, read both halves back, post after a yes, and ask rather than guess when
// the quote is not one place in the document.
var reviewAnnotateFlow = []string{
	"Find the exact words in the document's text.",
	"Read the quote and the comment back to the person",
	"Post after a yes",
	"A quote that occurs twice is refused",
	"Ask the person for more words",
}

// A comment on words the person names is the one piece of work that starts from
// them rather than from a thread somebody else opened. It can land on the wrong
// words, which nothing later undoes, so the quote is read back before it is
// written and an ambiguous one is a question and never a guess.
func TestTheCoreHasTheAnnotateFlow(t *testing.T) {
	// Arrange
	_, core := readReviewFile(t, reviewCore)

	// Act and assert
	for _, step := range reviewAnnotateFlow {
		want := squashSpace(step)
		if !strings.Contains(core, want) {
			t.Errorf("%s does not hold %q. The flow is read the same way through either front door, so it lives here", reviewCore, want)
		}
	}
}
