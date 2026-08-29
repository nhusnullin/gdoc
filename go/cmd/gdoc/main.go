// Command gdoc is gdoc v2. Facts in, JSON out, exit. The skill does the
// talking. Exactly one JSON object reaches stdout, and the exit code is 0 if
// and only if that object says ok.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"gdoc/internal/auth"
	"gdoc/internal/emit"
	"gdoc/internal/guard"
)

const usage = "Commands: auth status, auth login"

// login is the login flow behind a variable so a test can stand in for the
// browser trip. The client is the guard's, so even the token exchange passes a
// policy that could refuse it.
var login = func(errOut io.Writer) error {
	return auth.Login(guard.NewClient(guard.NewPolicy(), nil), errOut)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run turns arguments into one JSON object on out and an exit code. Human
// words, the login URL included, go to errOut: stdout carries the object and
// nothing else.
func run(args []string, out, errOut io.Writer) int {
	r := safeDispatch(args, errOut)
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
func safeDispatch(args []string, errOut io.Writer) (r emit.Result) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Fprintf(errOut, "gdoc crashed: %v\n%s\n", p, debug.Stack())
			r = emit.Result{OK: false, Error: fmt.Sprintf("gdoc crashed: %v", p)}
		}
	}()
	return dispatch(args, errOut)
}

func dispatch(args []string, errOut io.Writer) emit.Result {
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
	return emit.Result{OK: false,
		Error: fmt.Sprintf("unknown command %q. %s", strings.Join(args, " "), usage)}
}

func authStatus() emit.Result {
	data, err := auth.Status()
	if err != nil {
		// The data goes out beside the error: the caller still learns which
		// file was being read, and the error names what was wrong with it.
		return emit.Result{OK: false, Error: err.Error(), Data: data}
	}
	r := emit.Result{OK: true, Data: data}
	if data["client_file_ignored"] == true {
		r.Warnings = append(r.Warnings,
			"an oauth-client.json sits in the config dir, but v2 does not read it yet: the bundled client is in use")
	}
	return r
}

// authLogin prints the authorization URL to errOut and waits for the browser to
// come back to the loopback listener.
func authLogin(errOut io.Writer) emit.Result {
	if err := login(errOut); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	return authStatus()
}
