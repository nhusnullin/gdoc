// The one narrowing of the output contract: on a terminal a help screen
// replaces the object.
//
// Every test here stubs isTerminal per writer, so stdout and stderr answer
// apart and nothing depends on where the test binary's own streams go. The
// screens themselves are drawn in Task 9; what is held here is which runs
// drop the object, which keep it, and where the hint goes.

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
)

// terminals stands in for the terminal driver: the writers named are a
// terminal and every other writer is not. Comparing the interfaces is the
// whole of it, because a test hands its own buffers in.
func terminals(t *testing.T, writers ...io.Writer) {
	t.Helper()
	old := isTerminal
	isTerminal = func(w io.Writer) bool {
		return slices.ContainsFunc(writers, func(x io.Writer) bool { return x == w })
	}
	t.Cleanup(func() { isTerminal = old })
}

// screenRun is one command run with both streams a terminal, which is the
// person's case.
func screenRun(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	terminals(t, &out, &errOut)
	code = run(context.Background(), args, &out, &errOut)
	return out.String(), errOut.String(), code
}

// Help on a terminal is a screen, so the object it would have printed is not
// printed at all. The question was asked and answered, so it still exits 0.
func TestHelpOnATerminalWritesNothingToStdout(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	stdout, stderr, code := screenRun(t, "help")
	if code != 0 {
		t.Fatalf("help is an answer and must exit 0, and this run exited %d", code)
	}
	if stdout != "" {
		t.Errorf("a help screen replaces the object, and stdout carried %q", stdout)
	}
	if !strings.Contains(stderr, "gdoc <command> [words] [flags]") {
		t.Errorf("the screen a person reads must be on stderr: %q", stderr)
	}
}

// Bare gdoc is the other help screen. It did no work, so it still fails and
// still exits 1, and on a terminal it prints no object either.
func TestBareGdocOnATerminalWritesNothingToStdoutAndExitsOne(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	stdout, stderr, code := screenRun(t)
	if code != 1 {
		t.Fatalf("bare gdoc named no command, so it exits 1, and this run exited %d", code)
	}
	if stdout != "" {
		t.Errorf("bare gdoc on a terminal writes no object, and stdout carried %q", stdout)
	}
	if !strings.Contains(stderr, "gdoc <command> [words] [flags]") {
		t.Errorf("the screen a person reads must be on stderr: %q", stderr)
	}
}

// --json is the caller saying it wants the object wherever stdout goes, and it
// survives every spelling of the question: the two Task 2 reads the flag in
// are both here.
func TestHelpWithJSONPrintsTheObjectOnATerminal(t *testing.T) {
	for _, args := range [][]string{
		{"help", "--json"},
		{"help", "publish", "--json"},
		{"publish", "--help", "--json"},
		{"--help", "--json"},
		{"-h", "--json"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

			var out, errOut bytes.Buffer
			terminals(t, &out, &errOut)
			code := run(context.Background(), args, &out, &errOut)
			if code != 0 {
				t.Fatalf("%v is an answer and must exit 0, and this run exited %d", args, code)
			}
			got := decodeOne(t, &out)
			if got["ok"] != true {
				t.Fatalf("the object must say the question was answered: %v", got)
			}
			if len(helpCommands(t, got)) == 0 {
				t.Errorf("the object must carry the commands: %v", got["data"])
			}
			if strings.Contains(errOut.String(), jsonHint) {
				t.Errorf("the object was printed, so there is nothing to hint at: %q", errOut.String())
			}
		})
	}
}

// Words that name no command are refused, and a refusal is not a screen: the
// object a skill reads is the only record of what was wrong.
func TestARefusalKeepsItsObjectOnATerminal(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	stdout, _, code := screenRun(t, "help", "sing")
	if code != 1 {
		t.Fatalf("a word gdoc does not answer to is refused and exits 1, and this run exited %d", code)
	}
	got := decodeOne(t, strings.NewReader(stdout))
	if got["ok"] != false {
		t.Fatalf("a refusal is not a success: %v", got)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, `unknown command "sing"`) {
		t.Errorf("the refusal must name what was typed: %q", msg)
	}
}

// A crash is one envelope wherever stdout goes. The screen rule is a decision
// cmdHelp makes about an answer, and a panic is not an answer.
func TestAPanicKeepsItsEnvelopeOnATerminal(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	stubLogin(t, func(io.Writer) error { panic("something impossible") })

	stdout, _, code := screenRun(t, "auth", "login")
	if code != 1 {
		t.Fatalf("a crash must exit 1, and this run exited %d", code)
	}
	got := decodeOne(t, strings.NewReader(stdout))
	if msg, _ := got["error"].(string); !strings.Contains(msg, "something impossible") {
		t.Fatalf("the envelope must carry what happened: %q", msg)
	}
}

// Every command that is not help keeps its object on a terminal, because the
// object is the answer rather than a copy of something a person read.
func TestEveryOtherCommandKeepsItsObjectOnATerminal(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	stdout, _, code := screenRun(t, "auth", "status")
	if code != 0 {
		t.Fatalf("auth status is a report and must exit 0, and this run exited %d", code)
	}
	got := decodeOne(t, strings.NewReader(stdout))
	if got["ok"] != true {
		t.Fatalf("auth status with no token is still an answer: %v", got)
	}
}

// The hint says where the object went, so it is written when, and only when,
// the object was dropped. Three runs: the person's, a skill's with stderr on a
// terminal, and bare gdoc, which already says what to type.
func TestTheHintIsTheLastLineOnlyWhenTheObjectWasDropped(t *testing.T) {
	t.Run("stdout a terminal, no colour", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
		t.Setenv("NO_COLOR", "1")

		_, stderr, _ := screenRun(t, "help")
		if last := lastLine(stderr); last != jsonHint {
			t.Errorf("the hint is the last line of the screen, and the last line is %q", last)
		}
	})

	t.Run("stdout a terminal, colour on", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
		t.Setenv("NO_COLOR", "")
		t.Setenv("COLORTERM", "truecolor")

		_, stderr, _ := screenRun(t, "help")
		// The hint is the muted grey of palette 4A, stated here as the bytes
		// a terminal gets rather than read off the constant that writes them.
		want := "\x1b[38;2;124;130;150m" + jsonHint + "\x1b[0m"
		if last := lastLine(stderr); last != want {
			t.Errorf("the hint is dim when colour is on:\n got %q\nwant %q", last, want)
		}
	})

	t.Run("stdout a pipe and stderr a terminal", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

		var out, errOut bytes.Buffer
		terminals(t, &errOut)
		if code := run(context.Background(), []string{"help"}, &out, &errOut); code != 0 {
			t.Fatalf("help is an answer and must exit 0, and this run exited %d", code)
		}
		if decodeOne(t, &out)["ok"] != true {
			t.Fatal("stdout is a pipe, so the object must be printed")
		}
		if strings.Contains(errOut.String(), jsonHint) {
			t.Errorf("the object was printed, so nothing is hinted at: %q", errOut.String())
		}
	})

	t.Run("bare gdoc", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

		_, stderr, _ := screenRun(t)
		if strings.Contains(stderr, jsonHint) {
			t.Errorf("bare gdoc carries no hint: %q", stderr)
		}
	})
}

// Nothing a pipe gets carries an escape byte, on either stream, for any
// command in the table. A skill reads stdout and a log keeps stderr, and
// neither is a screen.
func TestNoEscapeByteReachesAPipe(t *testing.T) {
	for _, c := range commands() {
		t.Run(c.name, func(t *testing.T) {
			installedAt(t, "v2.9.0", []byte("a binary"), &stubPlain{listErr: errors.New("this run reaches nothing")})
			stubLogin(t, func(io.Writer) error { return errors.New("this run opens no browser") })

			var out, errOut bytes.Buffer
			terminals(t)
			run(context.Background(), c.nameWords(), &out, &errOut)
			for name, got := range map[string]string{"stdout": out.String(), "stderr": errOut.String()} {
				if strings.ContainsRune(got, 0x1b) {
					t.Errorf("%s of %q carries an escape byte: %q", name, c.name, got)
				}
			}
		})
	}

	t.Run("bare gdoc", func(t *testing.T) {
		installedAt(t, "v2.9.0", []byte("a binary"), &stubPlain{listErr: errors.New("this run reaches nothing")})

		var out, errOut bytes.Buffer
		terminals(t)
		run(context.Background(), nil, &out, &errOut)
		if strings.ContainsRune(out.String()+errOut.String(), 0x1b) {
			t.Errorf("bare gdoc on a pipe carries an escape byte: %q %q", out.String(), errOut.String())
		}
	})
}

// lastLine is the last line a reader saw, which is what the hint has to be.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return lines[len(lines)-1]
}
