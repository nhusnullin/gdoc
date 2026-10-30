// --json on help, in every spelling of the question, and --brief, the setup
// call a skill opens on.

package main

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gdoc/internal/config"
)

// helpRoute is which half of dispatch reads the flag for one spelling: the
// parser, for a line that opens with the word help, or helpWords, for --help
// and -h standing anywhere on the line. Both must come out with the flag read
// as a flag rather than carried through as a word that names no command.
type helpRoute int

const (
	routeTable helpRoute = iota
	routeAsked
)

// helpSpelling is one way of asking the question, with --json on it.
type helpSpelling struct {
	args  []string
	route helpRoute
	words []string
}

// Every spelling of the question takes --json, because a caller that wants the
// object does not know which spelling it typed. Here the flag is accepted, read
// as a flag, and the object still prints. What it changes on a terminal is the
// screen rule, and TestHelpWithJSONPrintsTheObjectOnATerminal holds that.
func TestEverySpellingOfHelpKeepsJSON(t *testing.T) {
	for _, spelling := range []helpSpelling{
		{args: []string{"help", "--json"}, route: routeTable},
		{args: []string{"help", "publish", "--json"}, route: routeTable, words: []string{"publish"}},
		{args: []string{"publish", "--help", "--json"}, route: routeAsked, words: []string{"publish"}},
		{args: []string{"--help", "--json"}, route: routeAsked},
		{args: []string{"-h", "--json"}, route: routeAsked},
	} {
		t.Run(strings.Join(spelling.args, " "), func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

			got, prose, code := runHelp(t, spelling.args...)
			if code != 0 || got["ok"] != true {
				t.Fatalf("the question must be answered: %v (exit %d)", got, code)
			}
			if len(helpCommands(t, got)) == 0 {
				t.Errorf("the object must still carry the commands: %v", got["data"])
			}
			if prose == "" {
				t.Errorf("the words a person reads must still be on stderr")
			}

			words, json := spelling.readFlag(t)
			if !json {
				t.Errorf("--json must be read as the flag, not as a word: %v", words)
			}
			if !slices.Equal(words, spelling.words) {
				t.Errorf("the words asked about are %v, and %v was read", spelling.words, words)
			}
		})
	}
}

// readFlag asks the half of dispatch this spelling goes through what it makes
// of the line: the words help answers over, and whether the object was asked
// for. The pieces are the real ones, so a spelling that stopped stripping the
// flag fails here.
func (s helpSpelling) readFlag(t *testing.T) ([]string, bool) {
	t.Helper()
	if s.route == routeAsked {
		rest, asked := helpAsked(s.args)
		if !asked {
			t.Fatalf("%v must be read as the help question", s.args)
		}
		return helpWords(rest)
	}
	c := match(s.args)
	if c == nil || c.name != "help" {
		t.Fatalf("%v must match the help entry", s.args)
	}
	a, err := parseArgsN(s.args[len(c.nameWords()):], c.flagSet(), c.wants())
	if err != nil {
		t.Fatalf("%v must parse: %v", s.args, err)
	}
	return a.positional, a.has("--json")
}

// --json is a flag of help and of nothing else. On its own it names no command,
// so it is refused by name like any other word gdoc does not answer to, and it
// does not quietly turn into the whole help.
func TestJSONAloneIsStillUnknown(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, _, code := runHelp(t, "--json")
	if code == 0 || got["ok"] != false {
		t.Fatalf("--json alone must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, `unknown command "--json"`) {
		t.Errorf("the refusal must name what was typed: %q", msg)
	}
}

// --brief is the setup call of every skill: the mark it must open a thread
// with, and the release notice, with the 7.5 KB table left out. The facts a
// skill reads are the same facts, in the same fields, so a skill that read the
// whole help reads this one without a second path.
func TestHelpBriefCarriesTheMarkAndTheNotice(t *testing.T) {
	pl := &checkPlain{listing: listing("v2.3.4", "v2.3.0", "v2.2.0")}
	checking(t, "v2.2.0", pl)

	got, _, code := runHelp(t, "help", "--brief", "--json")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the brief help is an answer: %v (exit %d)", got, code)
	}
	if got["version"] != "v2.2.0" {
		t.Errorf("the envelope carries the version: %v", got)
	}
	data := dataOf(t, got)
	if _, ok := data["commands"]; ok {
		t.Errorf("the brief help leaves the table out: %v", data)
	}
	set, ok := data["settings"].(map[string]any)
	if !ok {
		t.Fatalf("data.settings is not an object: %v", data)
	}
	if set["mark"] != "⁘" || set["mark_from"] != "default" {
		t.Errorf("the brief help says which mark is in effect and where it came from: %v", set)
	}
	facts := updateFactsOf(t, got)
	if facts["installed"] != "v2.2.0" || facts["latest_stable"] != "v2.3.0" {
		t.Errorf("the brief help carries the notice a due check wrote: %v", facts)
	}
}

// The whole reason for the flag is the size, so the size is pinned. A skill
// that opens on the brief help pays a few hundred bytes of its context for the
// two facts it needs, and not the thousands the table costs.
func TestHelpBriefIsAFewHundredBytes(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"help", "--brief", "--json"}, &out, &errOut, defaultsInEffect())
	if code != 0 {
		t.Fatalf("the brief help is an answer and exited %d: %s", code, out.String())
	}
	if n := out.Len(); n >= briefCeiling {
		t.Errorf("the brief help is %d bytes and must stay under %d: %s", n, briefCeiling, out.String())
	}
}

// briefCeiling is what the brief help may cost a skill's context, as a literal.
// A field added to it that pushes it past this is a decision, not a diff.
const briefCeiling = 600

// --brief is the help of the tool, not of one command, so a command word
// beside it is a call nobody can answer: a skill asking about publish wants
// the table for publish. It is refused by name rather than quietly answered
// with the tool's own facts.
func TestHelpBriefTakesNoCommandWord(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, _, code := runHelp(t, "help", "publish", "--brief")
	if code == 0 || got["ok"] != false {
		t.Fatalf("a command word beside --brief must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "--brief") || !strings.Contains(msg, "publish") {
		t.Errorf("the refusal must name the flag and the word: %q", msg)
	}
}

// A help screen replaces the object on a terminal, and the brief help is not a
// screen: it is a few hundred bytes a caller asked for by name. So its object
// prints wherever stdout goes, and stderr carries the notice head alone, with
// no table under it.
func TestHelpBriefPrintsItsObjectOnATerminal(t *testing.T) {
	pl := &checkPlain{listing: listing("v2.3.4", "v2.3.0", "v2.2.0")}
	checking(t, "v2.2.0", pl)

	var out, errOut bytes.Buffer
	terminals(t, &out, &errOut)
	code := run(context.Background(), []string{"help", "--brief"}, &out, &errOut, defaultsInEffect())
	if code != 0 {
		t.Fatalf("the brief help is an answer and exited %d: %s", code, errOut.String())
	}
	got := decodeOne(t, &out)
	if got["ok"] != true {
		t.Fatalf("the object must print on a terminal too: %v", got)
	}
	if _, ok := dataOf(t, got)["commands"]; ok {
		t.Errorf("the brief help leaves the table out: %v", got["data"])
	}
	if !strings.Contains(errOut.String(), "gdoc v2.3.0 is published") {
		t.Errorf("stderr must carry the notice head: %q", errOut.String())
	}
	if strings.Contains(errOut.String(), "gdoc <command> [words] [flags]") {
		t.Errorf("stderr must carry no help under the notice: %q", errOut.String())
	}
}

// houseLogoDrive is the Drive id of the house logo as a literal, so a run that
// names a different file fails here rather than teaching the menu bar app a
// path to a file nothing writes.
const houseLogoDrive = "1J3uh6poVAneP_V_Pc5FuNI4-Gahlab0s"

// The menu bar app draws the firm's mark in the panel, and bare `help --json`
// is the one argv it runs, so that is where the path to the cached logo is
// named. The help names it and touches nothing: no fetch, no stat, no read,
// and no assets directory where there was none. The field belongs to that one
// argv alone, so the brief help, a command's help and the flagless help on a
// pipe carry none of it: a skill reads those, and a skill draws nothing.
func TestHelpNamesWhereTheLogoIsCached(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)

	want, err := config.AssetPath(houseLogoDrive)
	if err != nil {
		t.Fatal(err)
	}

	got, _, code := runHelp(t, "help", "--json")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the full help is an answer: %v (exit %d)", got, code)
	}
	if path, _ := dataOf(t, got)["logo_path"].(string); path != want {
		t.Errorf("the full help must name %q and named %q", want, path)
	}

	if _, err := os.Lstat(want); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the help named the path and must not have written it: %v", err)
	}
	if _, err := os.Lstat(filepath.Dir(want)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the help must create no assets directory: %v", err)
	}

	for _, c := range []struct {
		name string
		args []string
	}{
		{name: "brief", args: []string{"help", "--brief", "--json"}},
		{name: "one command", args: []string{"help", "publish", "--json"}},
		{name: "no flag on a pipe", args: []string{"help"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

			got, _, code := runHelp(t, c.args...)
			if code != 0 || got["ok"] != true {
				t.Fatalf("%v is an answer: %v (exit %d)", c.args, got, code)
			}
			if _, ok := dataOf(t, got)["logo_path"]; ok {
				t.Errorf("%v must carry no logo path: %v", c.args, got["data"])
			}
		})
	}
}
