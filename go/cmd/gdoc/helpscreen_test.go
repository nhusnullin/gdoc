// The help screens: what a person reads on a terminal, recorded byte for byte.
//
// Every run here stubs isTerminal through terminals, names its width through
// COLUMNS and its depth through the environment, so a recorded screen is the
// same bytes on every machine and in CI. The pipe goldens next door hold the
// other half: nothing a skill reads moved.

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"gdoc/internal/lastcheck"
	"gdoc/internal/panel"
	"gdoc/internal/tty"
)

// Every command says which job it does, so none can fall off the grouped
// screen by being in no group at all.
func TestEveryCommandHasAGroup(t *testing.T) {
	for _, c := range commands() {
		if !slices.Contains(helpGroups, c.group) {
			t.Errorf("%q is in the group %q, and the four are %q", c.name, c.group, helpGroups)
		}
	}
}

// The group is for the screen and not for a skill: the object a caller parses
// carries the same fields it carried before.
func TestTheGroupIsNotInTheObject(t *testing.T) {
	freezing(t)

	stdout, _, code := pipeRun(t, "help", "--json")
	if code != 0 {
		t.Fatalf("help --json is an answer and exited %d: %s", code, stdout)
	}
	if !strings.Contains(stdout, `"name":"publish"`) {
		t.Fatalf("the object must carry the commands, and this is %q", stdout)
	}
	if strings.Contains(stdout, "group") {
		t.Errorf("the group is drawn and never printed, and the object carried it: %q", stdout)
	}
}

// The three screens, at the four widths of the width rule and once in
// truecolor. A golden that is not there is written and the test fails saying
// so, because a recorded screen is something a person reads once.
func TestTheHelpScreensAreTheirRecordedBytes(t *testing.T) {
	for _, s := range []struct {
		name string
		args []string
	}{
		{name: "help", args: []string{"help"}},
		{name: "help-publish", args: []string{"help", "publish"}},
		{name: "bare"},
	} {
		for _, width := range []int{100, 80, 60, 44} {
			t.Run(fmt.Sprintf("%s-%d", s.name, width), func(t *testing.T) {
				name := fmt.Sprintf("%s-%d.golden", s.name, width)
				recorded(t, name, screenRecord(t, width, false, s.args...))
			})
		}
		t.Run(s.name+"-80-truecolor", func(t *testing.T) {
			recorded(t, s.name+"-80-truecolor.golden", screenRecord(t, 80, true, s.args...))
		})
	}
}

// Every line a box is made of measures the width it was drawn at, with the
// escapes not counted, so a coloured screen is measured the way a terminal
// measures it. The lines outside the box are the caller's own: an example is
// one line a person copies, whatever the window is.
func TestNoLineOfAScreenIsWiderThanItsBox(t *testing.T) {
	names, err := filepath.Glob(filepath.Join("testdata", "screens", "*.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < 15 {
		t.Fatalf("three screens at four widths and one depth each is fifteen goldens, and testdata holds %d", len(names))
	}
	for _, name := range names {
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		width := panel.DrawWidth(screenWidthOf(t, filepath.Base(name)))
		for i, line := range strings.Split(strings.TrimRight(string(body), "\n"), "\n") {
			if !strings.ContainsAny(line[:min(3, len(line))], "┌├└│") {
				continue
			}
			if got := tty.VisibleWidth(line); got != width {
				t.Errorf("%s line %d is a box line and measures %d of %d: %q", name, i, got, width, line)
			}
		}
	}
}

// The grouped screen names every command gdoc answers to, under the job it
// does, and says how many there are.
func TestEveryCommandIsOnTheGroupedScreen(t *testing.T) {
	screen := screenRecord(t, 100, false, "help")

	for _, c := range commands() {
		if !strings.Contains(screen, c.name) {
			t.Errorf("the screen must name %q: %q", c.name, screen)
		}
	}
	for _, group := range helpGroups {
		if !strings.Contains(screen, group) {
			t.Errorf("the screen must name the group %q: %q", group, screen)
		}
	}
	if want := fmt.Sprintf("%d commands", len(commands())); !strings.Contains(screen, want) {
		t.Errorf("the top border says how many commands there are, %q: %q", want, screen)
	}
}

// NO_COLOR and TERM=dumb turn the colour off and leave the boxes, because the
// object is gone on a terminal either way and the screen is the whole answer.
func TestNoColourOnATerminalDrawsTheBoxesWithNoEscapeByte(t *testing.T) {
	for _, c := range []struct{ name, key, value string }{
		{name: "NO_COLOR", key: "NO_COLOR", value: "1"},
		{name: "TERM=dumb", key: "TERM", value: "dumb"},
	} {
		t.Run(c.name, func(t *testing.T) {
			freezing(t)
			t.Setenv("COLUMNS", "80")
			t.Setenv("NO_COLOR", "")
			t.Setenv("TERM", "xterm")
			t.Setenv("COLORTERM", "truecolor")
			t.Setenv(c.key, c.value)

			var out, errOut bytes.Buffer
			terminals(t, &out, &errOut)
			run(context.Background(), []string{"help"}, &out, &errOut)

			screen := errOut.String()
			if strings.ContainsRune(screen, 0x1b) {
				t.Errorf("%s writes no escape byte, and the screen carried one: %q", c.name, screen)
			}
			if !strings.Contains(screen, "┌") {
				t.Errorf("%s turns the colour off and keeps the boxes: %q", c.name, screen)
			}
		})
	}
}

// A warning the envelope would have carried is drawn in the box, because on a
// terminal the object that carried it is gone and a warning nobody reads is a
// warning nobody was given.
func TestAWarningIsDrawnWhereTheObjectWouldHaveCarriedIt(t *testing.T) {
	path := checking(t, "v2.9.0", &stubPlain{listErr: errors.New("no such host")})
	stampedAt(t, path, 48*time.Hour, lastcheck.Stamp{})
	t.Setenv("COLUMNS", "100")
	t.Setenv("NO_COLOR", "1")

	var out, errOut bytes.Buffer
	terminals(t, &out, &errOut)
	if code := run(context.Background(), []string{"help"}, &out, &errOut); code != 0 {
		t.Fatalf("a check that failed is still an answer, and this run exited %d", code)
	}

	screen := errOut.String()
	if !strings.Contains(screen, "no such host") {
		t.Errorf("the screen must say what stopped the check: %q", screen)
	}
	if !strings.Contains(screen, warnMark+" the releases of") {
		t.Errorf("a warning opens with its mark: %q", screen)
	}
}

// The example is printed under the box, flush left, as one line, so a copy
// runs in any shell with no line-break logic to undo.
func TestTheExampleIsOneLineUnderTheBox(t *testing.T) {
	screen := screenRecord(t, 80, false, "help", "comments")

	var c command
	for _, entry := range commands() {
		if entry.name == "comments" {
			c = entry
		}
	}
	if len(c.example) < 80 {
		t.Fatalf("this test is about an example wider than the window, and comments' is %d columns", len(c.example))
	}
	lines := strings.Split(screen, "\n")
	at := slices.Index(lines, c.example)
	if at < 0 {
		t.Fatalf("the example is one line of its own, and the screen is %q", screen)
	}
	if !strings.HasPrefix(lines[at-1], "└") {
		t.Errorf("the example sits under the box, and the line above it is %q", lines[at-1])
	}
	if !strings.Contains(lines[at-1], "Example") {
		t.Errorf("the bottom border names what follows it, and it is %q", lines[at-1])
	}
}

// Bare gdoc opens with the words of the refusal, so a person reads why they
// got the help they did not ask for.
func TestBareGdocOpensWithTheWordsItRefusesWith(t *testing.T) {
	screen := screenRecord(t, 80, false)

	if first := strings.SplitN(screen, "\n", 2)[0]; first != needsCommand {
		t.Errorf("the first line is %q and the screen opens with %q", first, needsCommand)
	}
	if !strings.Contains(screen, "┌") {
		t.Errorf("the help follows it as the same screen: %q", screen)
	}
}

// help reads the table and nothing else. No status bar, no "signed in" in a
// border, and so no way for help to fail on a token file: the three files that
// draw it name the auth package nowhere.
func TestHelpNeverReadsTheToken(t *testing.T) {
	for _, name := range []string{"help.go", "helpscreen.go", "notice.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			if strings.Contains(imported.Path.Value, "internal/auth") {
				t.Errorf("%s imports %s, and help reads no token", name, imported.Path.Value)
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "auth" {
				t.Errorf("%s names auth.%s, and help reads no token", name, sel.Sel.Name)
			}
			return true
		})
	}
}

// screenRecord is one run with both streams a terminal, at the width and depth
// the test names, and the screen it drew on stderr. Everything a run would
// otherwise read off the machine is fixed: the version, the release stamp and
// the reach, through freezing, and the window through COLUMNS, which is the
// only answer there is for a buffer.
func screenRecord(t *testing.T, width int, truecolor bool, args ...string) string {
	t.Helper()
	freezing(t)
	t.Setenv("COLUMNS", strconv.Itoa(width))
	t.Setenv("TERM", "xterm")
	if truecolor {
		t.Setenv("NO_COLOR", "")
		t.Setenv("COLORTERM", "truecolor")
	} else {
		t.Setenv("NO_COLOR", "1")
		t.Setenv("COLORTERM", "")
	}

	var out, errOut bytes.Buffer
	terminals(t, &out, &errOut)
	run(context.Background(), args, &out, &errOut)
	if out.Len() != 0 {
		t.Errorf("a help screen replaces the object, and stdout carried %q", out.String())
	}
	return errOut.String()
}

// recorded compares a drawn screen with the file that holds it. A golden that
// is not there yet is written and the test fails saying so; one that is there
// is never replaced from a run.
func recorded(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "screens", name)
	want, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Fatalf("%s held nothing, so this run wrote it. Read it, then run the test again.", path)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s is\n%s\nand the recorded screen is\n%s", name, got, string(want))
	}
}

// screenWidthOf reads the width back out of a golden's name, which is what the
// width assertion measures against.
func screenWidthOf(t *testing.T, name string) int {
	t.Helper()
	for _, part := range strings.Split(strings.TrimSuffix(name, ".golden"), "-") {
		if n, err := strconv.Atoi(part); err == nil {
			return n
		}
	}
	t.Fatalf("%s does not name its width", name)
	return 0
}
