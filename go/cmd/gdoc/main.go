// Command gdoc is gdoc v2. Facts in, JSON out, exit. The skill does the
// talking. Exactly one JSON object reaches stdout, and the exit code is 0 if
// and only if that object says ok.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"gdoc/internal/auth"
	"gdoc/internal/emit"
	"gdoc/internal/guard"
)

const usage = "Commands: auth status, auth login, read, comments, suggestions, probe, reply, propose, withdraw"

// login is the login flow behind a variable so a test can stand in for the
// browser trip. The client is the guard's, so even the token exchange passes a
// policy that could refuse it.
var login = func(errOut io.Writer) error {
	return auth.Login(guard.NewClient(guard.NewPolicy(), nil), errOut)
}

func main() {
	// The signal a live session ends with. Ctrl-C during a wait has to reach the
	// wait itself, because the answer it prints is the envelope it already has:
	// one JSON object, ok, no threads, and the cursor it was handed. A default
	// SIGINT kills the process mid-write, and stdout then carries half an
	// object, which is the one thing the output contract promises cannot
	// happen.
	//
	// stop before the exit rather than in a defer: os.Exit runs no deferred
	// call.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
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

// dispatch takes the context so the one command that can be stopped mid-run
// hears the signal. Every other command makes its requests and ends, so it
// carries on ignoring it, as it did before there was one.
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
		case "probe":
			return cmdProbe(args[1:])
		case "reply":
			return cmdReply(args[1:])
		case "propose":
			return cmdPropose(args[1:])
		case "withdraw":
			return cmdWithdraw(args[1:])
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
