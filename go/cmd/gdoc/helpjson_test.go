// --json on help, in every spelling of the question.

package main

import (
	"slices"
	"strings"
	"testing"
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
