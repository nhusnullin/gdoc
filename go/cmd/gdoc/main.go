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
)

// version is the release this binary was built from. The linker sets it from
// the tag, in both Make targets; a build nobody tagged keeps "dev". It is a
// var and not a const because -X can only write a var, and it is read through
// releaseVersion so one place decides what "dev" means.
var version = "dev"

// releaseVersion is the version as the envelope and the help print it: empty
// for an untagged build, so nothing a colleague sends back names a release
// that does not exist.
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
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

// run turns arguments into one JSON object on out and an exit code. Human
// words, the login URL included, go to errOut: stdout carries the object and
// nothing else.
func run(ctx context.Context, args []string, out, errOut io.Writer) int {
	r := safeDispatch(ctx, args, errOut)
	// The version is set here and nowhere else, so every object carries it:
	// the answers, the refusals and the panic envelope alike. A report of
	// something odd names the build that did it without anyone being asked.
	r.Version = releaseVersion()
	if err := emit.Print(out, r); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return emit.ExitCode(r)
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
	if len(args) == 0 {
		fmt.Fprint(errOut, helpProse(commands(), true))
		return emit.Result{OK: false, Error: "gdoc needs a command. " + usageLine()}
	}
	// --help and -h are the same question wherever they stand on the line, and
	// they are answered before the table is walked and before the parser runs.
	// So `gdoc restyle --from x --help` is an answer rather than a refusal of a
	// flag restyle does not take, and no parser refusal changes.
	if rest, asked := helpAsked(args); asked {
		return cmdHelp(helpWords(rest), errOut)
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
