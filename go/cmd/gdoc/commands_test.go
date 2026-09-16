package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The table is the one description of what gdoc answers, so the two halves of
// that claim are checked by running the binary rather than by reading it. Every
// entry is reachable, and a word that is in no entry is refused as unknown.
//
// The bad flag is what makes the first half readable: a command that is
// dispatched refuses the flag with its own words, and one that is not is
// refused as a command nobody knows. Neither answer touches the network.
func TestEveryCommandInTheTableIsDispatchedAndNothingElseIs(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	// A table read out of an empty slice would pass every loop below without
	// measuring anything.
	table := commands()
	if len(table) == 0 {
		t.Fatal("the command table is empty, so this test is measuring nothing")
	}

	for _, c := range table {
		args := append(strings.Fields(c.name), "--not-a-flag")
		got, code := runJSON(t, args...)
		if code == 0 || got["ok"] != false {
			t.Errorf("%s with a flag it does not take must fail: %v (exit %d)", c.name, got, code)
			continue
		}
		msg, _ := got["error"].(string)
		if strings.Contains(msg, "unknown command") {
			t.Errorf("%q is in the table, so it must be dispatched rather than refused as unknown: %q", c.name, msg)
		}
		if !strings.Contains(msg, "--not-a-flag") {
			t.Errorf("%s must name the flag it refused: %q", c.name, msg)
		}
	}

	// The other half. "auth" alone is the one worth naming: two entries start
	// with it, and neither is a command a person can run on its own.
	for _, args := range [][]string{{"sing"}, {"sing", "loudly"}, {"auth"}, {"reads"}} {
		got, code := runJSON(t, args...)
		if code == 0 || got["ok"] != false {
			t.Errorf("%v names no command and must fail: %v (exit %d)", args, got, code)
			continue
		}
		if msg, _ := got["error"].(string); !strings.Contains(msg, "unknown command") {
			t.Errorf("%v names no command, so the refusal must say so: %q", args, msg)
		}
	}
}

// Each command parses with the flag set its entry describes, and nothing else.
// A flag the entry names is known to the command, a flag it does not name is
// refused, and whether a flag carries a value follows from its kind.
//
// Both shapes below fail inside the parser, before a session is opened, so the
// test makes no request and reads no document.
func TestEachCommandParsesWithTheFlagSetItsTableEntryDescribes(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	table := commands()
	if len(table) == 0 {
		t.Fatal("the command table is empty, so this test is measuring nothing")
	}

	for _, c := range table {
		for _, f := range c.flags {
			if f.value == kindNone {
				// A value written onto a flag that takes none is refused for
				// that reason, which says the parser was fed takes-no-value.
				got, _ := runJSON(t, append(strings.Fields(c.name), f.name+"=x")...)
				msg, _ := got["error"].(string)
				if !strings.Contains(msg, "takes no value") {
					t.Errorf("the table says %s %s takes no value: %q", c.name, f.name, msg)
				}
				continue
			}
			// A value flag standing last has nothing to take, which says the
			// parser was fed takes-a-value.
			got, _ := runJSON(t, append(strings.Fields(c.name), f.name)...)
			msg, _ := got["error"].(string)
			if !strings.Contains(msg, "needs a value") {
				t.Errorf("the table says %s %s takes %s: %q", c.name, f.name, f.value.placeholder(), msg)
			}
		}

		// The refusal lists the flags the command takes, so it lists the
		// entry's flags and no others.
		got, _ := runJSON(t, append(strings.Fields(c.name), "--not-a-flag")...)
		msg, _ := got["error"].(string)
		for _, f := range c.flags {
			if !strings.Contains(msg, f.name) {
				t.Errorf("%s takes %s, and the refusal does not list it: %q", c.name, f.name, msg)
			}
		}
	}
}

// The kind a flag carries decides two things a reader will trust: what help
// prints after the flag name, and whether the parser takes the next word as its
// value. Both follow from the table, so the table saying <file> where the
// command reads a yes or no is a lie nothing else would catch.
//
// The command's own reading is the witness. A flag that carries a value is
// read for the value, through a.flags or required. A flag that carries none is
// read for its presence and nothing else.
func TestEveryFlagIsReadTheWayItsKindSays(t *testing.T) {
	source := productionSource(t)

	for _, c := range commands() {
		for _, f := range c.flags {
			presence := strings.Contains(source, `a.has("`+f.name+`")`)
			value := strings.Contains(source, `a.flags["`+f.name+`"]`) ||
				strings.Contains(source, `required(a, "`+f.name+`")`)
			if f.value == kindNone {
				if value {
					t.Errorf("the table says %s %s takes no value, and the command reads one", c.name, f.name)
				}
				if !presence {
					t.Errorf("%s %s is in the table and the command reads it nowhere", c.name, f.name)
				}
				continue
			}
			if !value {
				t.Errorf("the table says %s %s takes %s, and the command never reads a value for it",
					c.name, f.name, f.value.placeholder())
			}
		}
	}
}

// productionSource is every Go file in this package that is not a test, read as
// one string. It fails when it finds none, so a moved package leaves a test
// that says so rather than one that passes over nothing.
func productionSource(t *testing.T) string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(src)
	}
	if b.Len() == 0 {
		t.Fatal("no production Go files were read, so this test is measuring nothing")
	}
	return b.String()
}

// The check above can only see the flags a command was handed. A command
// keeping a flag set of its own would be invisible to it, so this says there is
// no second place to keep one: after the table, no production file here builds
// a flagSet.
//
// It fails in both directions, like the boundary tests. It fails when a
// command writes its own set again, and it fails when it finds no files.
func TestNoCommandBuildsAFlagSetOfItsOwn(t *testing.T) {
	if strings.Contains(productionSource(t), "flagSet{") {
		t.Error("a command builds a flag set of its own: the table is the one description of what a command takes")
	}
}

// The example is the one whole call a command's help prints, and Decision 8
// tells a session to build its call from what help printed. So an example that
// the binary would refuse teaches a call that fails at the terminal.
//
// Most of that the table can check on itself: the example opens with the
// binary and the command's own words, and what follows parses with the flag set
// and the word count the entry describes. The value kinds go further. A kind
// that says how a value is written is a promise the example keeps too, so a
// duration in an example is a length of time and not a bare number.
//
// Nothing here runs a command or opens a session: the example is read, not
// executed, because its document ids and file names are placeholders.
func TestEveryExampleIsACallTheTableAccepts(t *testing.T) {
	table := commands()
	if len(table) == 0 {
		t.Fatal("the command table is empty, so this test is measuring nothing")
	}

	for _, c := range table {
		fields := strings.Fields(c.example)
		if len(fields) == 0 || fields[0] != "gdoc" {
			t.Errorf("the %s example must open with the binary: %q", c.name, c.example)
			continue
		}
		rest := fields[1:]
		words := c.nameWords()
		if len(rest) < len(words) || strings.Join(rest[:len(words)], " ") != c.name {
			t.Errorf("the %s example must name the command it is for: %q", c.name, c.example)
			continue
		}
		rest = rest[len(words):]
		if _, err := parseArgsN(rest, c.flagSet(), c.wants()); err != nil {
			t.Errorf("the %s example is a call the parser refuses: %v", c.name, err)
			continue
		}

		kinds := make(map[string]kind, len(c.flags))
		for _, f := range c.flags {
			kinds[f.name] = f.value
		}
		for i, field := range rest {
			name, value, joined := strings.Cut(field, "=")
			if !joined {
				if i+1 >= len(rest) {
					continue
				}
				value = rest[i+1]
			}
			if kinds[name] != kindDuration {
				continue
			}
			if _, err := time.ParseDuration(value); err != nil {
				t.Errorf("the %s example writes %s %q, and the command reads a length of time such as 9m",
					c.name, name, value)
			}
		}
	}
}

// --wait is refused without --since, in cmdComments and not in the table, so
// the table cannot catch this one on itself. The rule is in read.go: a wait
// with no cursor would answer with the whole document, which is the one-shot
// read under another name.
func TestTheCommentsExampleShowsTheCursorItsWaitNeeds(t *testing.T) {
	for _, c := range commands() {
		if c.name != "comments" {
			continue
		}
		if !strings.Contains(c.example, "--wait") {
			return
		}
		if !strings.Contains(c.example, "--since") {
			t.Errorf("--wait is refused without --since, and the comments example shows only the wait: %q", c.example)
		}
		return
	}
	t.Fatal("no comments command in the table, so this test is measuring nothing")
}
