package main

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
)

// everyCommandName is what `gdoc help` must come back with, written out word
// for word the way a reader types it. It is a literal and not a walk of the
// table, so a command added without a sentence saying what it is for fails
// here rather than arriving in the help unannounced.
var everyCommandName = []string{
	"auth status", "auth login", "read", "comments", "suggestions", "restyle",
	"probe", "reply", "propose", "withdraw", "build", "publish", "help",
}

// runHelp runs the command with both streams held apart, because the split is
// the thing under test: the object on stdout, the words a person reads on
// stderr, and nothing of either on the other.
func runHelp(t *testing.T, args ...string) (map[string]any, string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(context.Background(), args, &out, &errOut)
	return decodeOne(t, &out), errOut.String(), code
}

// helpCommands is the list out of the object, as a skill reads it.
func helpCommands(t *testing.T, got map[string]any) []any {
	t.Helper()
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	list, ok := data["commands"].([]any)
	if !ok {
		t.Fatalf("data.commands is not a list: %v", data)
	}
	return list
}

// helpNames is the name of each entry, in the order they came back.
func helpNames(t *testing.T, got map[string]any) []string {
	t.Helper()
	var names []string
	for _, entry := range helpCommands(t, got) {
		e, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("an entry is not an object: %v", entry)
		}
		name, ok := e["name"].(string)
		if !ok {
			t.Fatalf("an entry has no name: %v", e)
		}
		names = append(names, name)
	}
	return names
}

// Help is an answer, so it is one object on stdout and exit 0, and the words a
// person reads go where the login URL already goes. The output contract does
// not bend for the one command a person runs to learn the others.
func TestHelpIsOneObjectAndTheProseIsOnStderr(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, prose, code := runHelp(t, "help")
	if code != 0 || got["ok"] != true {
		t.Fatalf("help is an answer: %v (exit %d)", got, code)
	}
	if !reflect.DeepEqual(helpNames(t, got), everyCommandName) {
		t.Errorf("help must name every command, in the table's order:\n got %v\nwant %v",
			helpNames(t, got), everyCommandName)
	}
	if !strings.Contains(prose, "Usage: gdoc") {
		t.Errorf("the prose must open with the usage line: %q", prose)
	}
	for _, name := range everyCommandName {
		if !strings.Contains(prose, name) {
			t.Errorf("the prose must name %q: %q", name, prose)
		}
	}
}

// One command's entry, against literals. The test spells out what publish
// takes rather than reading the table, so a flag renamed in the table and
// nowhere else fails here, which is the whole point of a help a skill trusts.
func TestHelpForOneCommandCarriesItsWordsFlagsAndExample(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, _, code := runHelp(t, "help", "publish")
	if code != 0 || got["ok"] != true {
		t.Fatalf("help publish: %v (exit %d)", got, code)
	}
	list := helpCommands(t, got)
	if len(list) != 1 {
		t.Fatalf("publish is one command: %v", list)
	}
	entry, _ := list[0].(map[string]any)
	if entry["name"] != "publish" {
		t.Errorf("name: %v", entry["name"])
	}
	// publish names its note and its folder with flags, so it takes no words.
	// An absent list and an empty one read differently to a skill.
	words, ok := entry["words"].([]any)
	if !ok || len(words) != 0 {
		t.Errorf("publish takes no words, and the empty list must be there: %v", entry["words"])
	}
	if entry["example"] != "gdoc publish --md note.md --folder-id 1AbC..." {
		t.Errorf("example: %v", entry["example"])
	}
	if s, _ := entry["summary"].(string); s == "" {
		t.Error("a command carries one sentence saying what it is for")
	}

	wantFlags := [][2]string{
		{"--md", "<file>"},
		{"--folder-id", "<folder id>"},
		{"--house", "<file>"},
	}
	flags, ok := entry["flags"].([]any)
	if !ok || len(flags) != len(wantFlags) {
		t.Fatalf("publish takes three flags: %v", entry["flags"])
	}
	for i, want := range wantFlags {
		f, _ := flags[i].(map[string]any)
		if f["name"] != want[0] || f["value"] != want[1] {
			t.Errorf("flag %d: got %v %v, want %s %s", i, f["name"], f["value"], want[0], want[1])
		}
		if s, _ := f["summary"].(string); s == "" {
			t.Errorf("%s carries no sentence saying what it is for", want[0])
		}
	}

	// A flag that takes no value says so with an empty placeholder rather than
	// with the key missing, so a skill reads one shape for every flag.
	got, _, _ = runHelp(t, "help", "read")
	entry, _ = helpCommands(t, got)[0].(map[string]any)
	flags, _ = entry["flags"].([]any)
	if len(flags) != 1 {
		t.Fatalf("read takes one flag: %v", entry["flags"])
	}
	f, _ := flags[0].(map[string]any)
	if f["name"] != "--structure" || f["value"] != "" {
		t.Errorf("--structure takes no value: %v", f)
	}
	words, ok = entry["words"].([]any)
	if !ok || len(words) != 1 || words[0] != "<url>" {
		t.Errorf("read takes a document: %v", entry["words"])
	}
}

// help matches by prefix, so the words a person half remembers are enough.
// What matches nothing is refused the way an unknown command is, because that
// is what it is.
func TestHelpMatchesByPrefixAndRefusesWhatItDoesNotKnow(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, prose, code := runHelp(t, "help", "auth")
	if code != 0 || got["ok"] != true {
		t.Fatalf("help auth: %v (exit %d)", got, code)
	}
	if want := []string{"auth status", "auth login"}; !reflect.DeepEqual(helpNames(t, got), want) {
		t.Errorf("help auth is both auth commands: %v", helpNames(t, got))
	}
	if !strings.Contains(prose, "auth status") || !strings.Contains(prose, "auth login") {
		t.Errorf("the prose must carry both: %q", prose)
	}

	got, _, code = runHelp(t, "help", "auth", "status")
	if code != 0 || !reflect.DeepEqual(helpNames(t, got), []string{"auth status"}) {
		t.Errorf("help auth status is one command: %v (exit %d)", got, code)
	}

	got, _, code = runHelp(t, "help", "sing")
	if code == 0 || got["ok"] != false {
		t.Fatalf("help sing names no command and must fail: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "unknown command") || !strings.Contains(msg, "sing") {
		t.Errorf("the refusal must name what it does not know: %q", msg)
	}
	if !strings.Contains(msg, "auth status") || !strings.Contains(msg, "publish") {
		t.Errorf("the refusal must carry the usage line: %q", msg)
	}
}

// --help and -h are the same question wherever they stand. They are read
// before the table is walked and before the parser runs, so `restyle --from x
// --help` is an answer rather than a refusal of a flag restyle does not take.
func TestDashDashHelpIsAnAliasAnywhereOnTheLine(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	same := func(alias []string, plain []string) {
		t.Helper()
		got, _, code := runHelp(t, alias...)
		want, _, wantCode := runHelp(t, plain...)
		if code != wantCode {
			t.Errorf("%v exits %d and %v exits %d", alias, code, plain, wantCode)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%v must answer as %v does:\n got %v\nwant %v", alias, plain, got, want)
		}
	}

	same([]string{"--help"}, []string{"help"})
	same([]string{"-h"}, []string{"help"})
	same([]string{"read", "--help"}, []string{"help", "read"})
	same([]string{"--help", "read"}, []string{"help", "read"})
	same([]string{"-h", "read"}, []string{"help", "read"})
	same([]string{"restyle", "--from", "x", "--help"}, []string{"help", "restyle"})
	same([]string{"auth", "status", "--help"}, []string{"help", "auth", "status"})

	// A word in no entry is still refused, with the help asked for or without.
	same([]string{"sing", "--help"}, []string{"help", "sing"})
}

// Bare gdoc did no work, so it keeps today's failure and today's object. What
// changes is that the person who typed it now reads what they could have
// typed, on the stream the words belong on.
func TestBareGdocStillFailsAndPrintsTheHelpToStderr(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, prose, code := runHelp(t)
	if code != 1 || got["ok"] != false {
		t.Fatalf("bare gdoc must still fail: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "needs a command") {
		t.Errorf("the object is unchanged: %q", msg)
	}
	if got["data"] != nil {
		t.Errorf("a run that did no work carries no data: %v", got["data"])
	}
	for _, name := range everyCommandName {
		if !strings.Contains(prose, name) {
			t.Errorf("the whole help must be on stderr, and %q is not: %q", name, prose)
		}
	}
}

// help takes words and no flags, and says so by name like every other command
// here. Nothing is accepted and ignored.
func TestHelpTakesWordsAndNoFlags(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	for _, args := range [][]string{
		{"help", "--md", "x"},
		{"help", "--structure"},
		{"help", "publish", "--folder-id", "1AbC"},
	} {
		got, _, code := runHelp(t, args...)
		if code == 0 || got["ok"] != false {
			t.Errorf("%v must be refused: %v (exit %d)", args, got, code)
			continue
		}
		msg, _ := got["error"].(string)
		if !strings.Contains(msg, "not a flag this command takes") {
			t.Errorf("%v must be refused by name: %q", args, msg)
		}
	}
}
