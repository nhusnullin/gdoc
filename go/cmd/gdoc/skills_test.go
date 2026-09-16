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
