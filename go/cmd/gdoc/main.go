// The entry point, the dispatch table and the auth commands.
//
// The package comment is in doc.go. This file holds the one rule the three
// pieces here share: whatever happens, the caller reads one JSON object on
// stdout and an exit code that agrees with it.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"gdoc/internal/auth"
	"gdoc/internal/emit"
	"gdoc/internal/guard"
	"gdoc/internal/panel"
	"gdoc/internal/tty"
)

// version is the release this binary was built from. The linker sets it from
// the tag, in both Make targets; a build nobody tagged keeps "dev". It is a
// var and not a const because -X can only write a var, and it is read through
// releaseVersion so one place decides what counts as a release.
var version = "dev"

// releaseVersion is the version as the envelope and the help print it: empty
// for an untagged build, so nothing a colleague sends back names a release
// that does not exist.
//
// "dev" is the sentinel and the Makefile is what has to reach it. It stamps
// the version from `git describe --tags --exact-match --dirty`, which names a
// version only when HEAD is exactly a release with nothing uncommitted over
// it, and leaves "dev" otherwise. A stamp that resolved to a commit hash
// instead would never take this branch, and every envelope from a checkout
// would carry a number naming no release a colleague could fetch and no
// version a skill could compare. TestTheVersionStampNamesOnlyATag holds that
// half, in go/boundary.
func releaseVersion() string {
	if version == "dev" {
		return ""
	}
	return version
}

// login is the login flow behind a variable so a test can stand in for the
// browser trip. The client is the guard's, so even the token exchange passes a
// policy that could refuse it.
var login = func(errOut io.Writer) error {
	return auth.Login(guard.NewClient(guard.NewPolicy(), nil), errOut)
}

func main() {
	// No signal handling here, and that is deliberate. The one run that has to
	// hear Ctrl-C is a wait, and it installs its own handler for the length of
	// the wait and takes it off again: see cmdComments. Trapping the signal for
	// the whole process would take the default kill away from every other
	// command, and a `gdoc propose` that cannot be stopped is worse than one
	// that dies where it stands.
	os.Exit(route(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// route is what main does with the line before run sees it, and it is the one
// place the real streams are named. Everything in this file below it, and every
// package under internal/, is handed a reader and a writer:
// TestOnlyMainNamesStdinAndStdout in go/boundary.
//
// Fifteen commands print one JSON object and end. mcp does not: it is a
// JSON-RPC session over stdin and stdout, one message per line, for as long as
// Claude Desktop is open, so an envelope around it would be a stray line in
// the middle of somebody's protocol. That is the whole of what this helper
// decides.
//
// --help and -h are not routed. They are the same question about mcp that they
// are about every other command, and the table answers them, so gdoc mcp
// --help prints the usage line rather than starting a server that waits on a
// pipe nobody is going to fill. TestMcpIsRoutedBeforeRun is the pin.
func route(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	if _, asked := helpAsked(args); !asked && len(args) > 0 && args[0] == "mcp" {
		return serveMCP(ctx, in, out, errOut, args[1:])
	}
	return run(ctx, args, out, errOut)
}

// isTerminal is internal/tty's question, through one variable so a test can
// answer it per writer and stdout and stderr can be a terminal apart.
// tty.IsTerminal is the only definition of a terminal in the tree: it is the
// terminal driver's answer and not the file's mode, which is why /dev/null is
// not one. TestOnlyATerminalDriverMakesATerminal holds that this variable is
// the whole of the decision made here.
//
// Two rooms ask it: run(), for whether a screen replaces the object, and the
// step list in progress.go, for whether to redraw in place.
var isTerminal = tty.IsTerminal

// jsonHint is the last line of a help screen that dropped its object, so the
// one person who wanted that object is told, once, how to ask for it. It is
// never written where the object was printed, and never on bare gdoc, which
// already says what to type:
// TestTheHintIsTheLastLineOnlyWhenTheObjectWasDropped.
const jsonHint = "Add --json to print the JSON object a skill reads."

// run turns arguments into one JSON object on out and an exit code. Human
// words, the login URL included, go to errOut: stdout carries the object and
// nothing else.
//
// The one narrowing is the help screen. A result that says Screen is a result
// whose whole answer is words already written to errOut, and where out is a
// terminal those words are what the reader asked for: the object under them
// is 7.5 KB nobody reads. So it is not printed, and the hint takes its place
// as the last line a person sees. Everything else keeps its object wherever
// stdout goes, refusals and the panic envelope included, and a pipe gets the
// object in every case, because a pipe is a skill.
// TestHelpOnATerminalWritesNothingToStdout,
// TestBareGdocOnATerminalWritesNothingToStdoutAndExitsOne,
// TestEveryOtherCommandKeepsItsObjectOnATerminal and
// TestNoEscapeByteReachesAPipe.
func run(ctx context.Context, args []string, out, errOut io.Writer) int {
	r := safeDispatch(ctx, args, errOut)
	// The version is set here and nowhere else, so every object carries it:
	// the answers, the refusals and the panic envelope alike. A report of
	// something odd names the build that did it without anyone being asked.
	r.Version = releaseVersion()
	if r.Screen && isTerminal(out) {
		writeJSONHint(r, errOut)
		return emit.ExitCode(r)
	}
	if err := emit.Print(out, r); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return emit.ExitCode(r)
}

// writeJSONHint is the line under a screen that dropped its object, and it is
// written for help and not for bare gdoc. Bare gdoc is the one screen that
// failed: the person who typed it was reaching for a command, not for an
// object, and the refusal already tells them what to do next. So the ok field
// is what parts the two.
//
// It wraps rather than running off the window, because the one thing the line
// has to do is be readable at the width the reader has.
func writeJSONHint(r emit.Result, errOut io.Writer) {
	if !r.OK {
		return
	}
	style := tty.NewStyle(tty.Colour(os.Getenv))
	for _, line := range panel.Wrap(jsonHint, tty.Width(errOut, os.Getenv)) {
		fmt.Fprintln(errOut, style.Dim(line))
	}
}

// safeDispatch turns a panic into the envelope. Without it a crash prints a Go
// stack trace, nothing at all on stdout, and exits 2, which breaks the one
// contract every command has. The trace still goes to stderr, where it is
// readable without being mistaken for output.
func safeDispatch(ctx context.Context, args []string, errOut io.Writer) (r emit.Result) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Fprintf(errOut, "gdoc crashed: %v\n%s\n", p, debug.Stack())
			r = emit.Result{OK: false, Error: fmt.Sprintf("gdoc crashed: %v", p)}
		}
	}()
	return dispatch(ctx, args, errOut)
}

// dispatch takes the context so a test can hand a wait one that is already
// done, which is the answer a stopped session gets. The signal itself is
// trapped inside the wait rather than here: every other command makes its
// requests and ends, and Ctrl-C kills it the way it always did.
func dispatch(ctx context.Context, args []string, errOut io.Writer) emit.Result {
	// Bare gdoc named no command, so there is nothing to quote back: `unknown
	// command ""` names nothing and reads like a fault in the tool. The run did
	// no work, so it still fails and still exits 1. What is new is that the
	// person who typed it reads the whole help, on the stream words go to.
	//
	// It is the second help screen, so on a terminal the object goes the way
	// help's does. The hint does not follow it: see writeJSONHint.
	if len(args) == 0 {
		fmt.Fprint(errOut, helpProse(commands(), true))
		return emit.Result{OK: false, Error: "gdoc needs a command. " + usageLine(), Screen: true}
	}
	// --help and -h are the same question wherever they stand on the line, and
	// they are answered before the table is walked and before the parser runs.
	// So `gdoc restyle --from x --help` is an answer rather than a refusal of a
	// flag restyle does not take, and no parser refusal changes.
	if rest, asked := helpAsked(args); asked {
		words, json := helpWords(rest)
		return cmdHelp(ctx, words, json, errOut)
	}
	c := match(args)
	if c == nil {
		return unknownCommand(args)
	}
	// The rest of the line is parsed here, with the flag set and the word count
	// the command's own table entry describes. Strictly, as everywhere: an
	// unknown flag, a repeated one, a missing value and an extra word each fail
	// naming the offender. So `gdoc auth login --token /path` is refused for
	// the flag rather than read as a plain login that quietly dropped it, and
	// `gdoc auth` on its own matches no entry and is an unknown command.
	a, err := parseArgsN(args[len(c.nameWords()):], c.flagSet(), c.wants())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	return c.run(ctx, a, errOut)
}

// statusData is the auth report with the version beside it. `gdoc auth status`
// is the command a person runs to ask what they have, so the build is one of
// the facts it reports, not only a field of the envelope around it.
type statusData struct {
	*auth.StatusReport
	Version string `json:"version,omitempty"`
}

// statusReport is the report as it goes out. A nil report stays nil rather
// than becoming an object holding nothing but a version: a config dir gdoc
// cannot locate has no facts to report, and that is what the error says.
func statusReport(r *auth.StatusReport) any {
	if r == nil {
		return r
	}
	return statusData{StatusReport: r, Version: releaseVersion()}
}

func authStatus() emit.Result {
	report, err := auth.Status()
	if err != nil {
		// The report goes out beside the error, warnings included: the caller
		// still learns which file was being read and what else is in the way,
		// and the error names what was wrong with it. The run that fails is the
		// one where the extra fact is worth most.
		return emit.Result{OK: false, Error: err.Error(), Data: statusReport(report), Warnings: statusWarnings(report)}
	}
	return emit.Result{OK: true, Data: statusReport(report), Warnings: statusWarnings(report)}
}

// statusWarnings says what the report cannot say in a field: a fact that is
// true, not a failure, and still changes what the reader should do next.
func statusWarnings(r *auth.StatusReport) []string {
	if r == nil {
		return nil
	}
	var w []string
	if r.ClientFileIgnored {
		w = append(w, "an oauth-client.json sits in the config dir, but v2 does not read it yet: the bundled client is in use")
	}
	if len(r.MissingScopes) > 0 {
		w = append(w, fmt.Sprintf(
			"the token does not carry every scope gdoc asks for, so those calls will be refused: %s. Run: gdoc auth login",
			strings.Join(r.MissingScopes, ", ")))
	}
	return w
}

// authLogin prints the authorization URL to errOut and waits for the browser to
// come back to the loopback listener.
func authLogin(errOut io.Writer) emit.Result {
	if err := login(errOut); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	return authStatus()
}
