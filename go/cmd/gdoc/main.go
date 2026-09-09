// Command gdoc is gdoc v2. Facts in, JSON out, exit. The skill does the
// talking. Exactly one JSON object reaches stdout, and the exit code is 0 if
// and only if that object says ok.
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

const usage = "Commands: auth status, auth login, read, comments, suggestions, restyle, probe, reply, propose, withdraw, build, publish"

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
	// Exactly two words, and no more. A command that accepts and ignores what
	// it does not understand tells the user it did something it did not:
	// `gdoc auth login --token /path` must not read as a plain login.
	if len(args) == 2 && args[0] == "auth" {
		switch args[1] {
		case "status":
			return authStatus()
		case "login":
			return authLogin(errOut)
		}
	}
	// The read and write commands take their own arguments, so they are matched
	// on the first word and parse the rest themselves. Strictly: parseArgs
	// refuses an unknown flag, a repeated one and an extra positional argument.
	if len(args) > 0 {
		switch args[0] {
		case "read":
			return cmdRead(args[1:])
		case "comments":
			return cmdComments(ctx, args[1:])
		case "suggestions":
			return cmdSuggestions(args[1:])
		case "restyle":
			return cmdRestyle(args[1:])
		case "probe":
			return cmdProbe(args[1:])
		case "reply":
			return cmdReply(args[1:])
		case "propose":
			return cmdPropose(args[1:])
		case "withdraw":
			return cmdWithdraw(args[1:])
		case "build":
			return cmdBuild(args[1:])
		case "publish":
			return cmdPublish(args[1:])
		}
	}
	// Bare gdoc named no command, so there is nothing to quote back: `unknown
	// command ""` names nothing and reads like a fault in the tool.
	if len(args) == 0 {
		return emit.Result{OK: false, Error: "gdoc needs a command. " + usage}
	}
	return emit.Result{OK: false,
		Error: fmt.Sprintf("unknown command %q. %s", strings.Join(args, " "), usage)}
}

func authStatus() emit.Result {
	report, err := auth.Status()
	if err != nil {
		// The report goes out beside the error, warnings included: the caller
		// still learns which file was being read and what else is in the way,
		// and the error names what was wrong with it. The run that fails is the
		// one where the extra fact is worth most.
		return emit.Result{OK: false, Error: err.Error(), Data: report, Warnings: statusWarnings(report)}
	}
	return emit.Result{OK: true, Data: report, Warnings: statusWarnings(report)}
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
