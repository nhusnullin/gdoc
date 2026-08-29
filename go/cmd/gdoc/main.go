// Command gdoc is gdoc v2. Facts in, JSON out, exit. The skill does the
// talking. Exactly one JSON object reaches stdout, and the exit code is 0 if
// and only if that object says ok.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gdoc/internal/auth"
	"gdoc/internal/emit"
	"gdoc/internal/guard"
)

const usage = "Commands: auth status, auth login"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run turns arguments into one JSON object on out and an exit code. Human
// words, the login URL included, go to errOut: stdout carries the object and
// nothing else.
func run(args []string, out, errOut io.Writer) int {
	r := dispatch(args, errOut)
	if err := emit.Print(out, r); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return emit.ExitCode(r)
}

func dispatch(args []string, errOut io.Writer) emit.Result {
	if len(args) >= 2 && args[0] == "auth" {
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
		return emit.Result{OK: false, Error: err.Error()}
	}
	return emit.Result{OK: true, Data: data}
}

// authLogin prints the authorization URL to errOut and waits for the browser to
// come back to the loopback listener. The client is the guard's, so even the
// token exchange passes a policy that could refuse it.
func authLogin(errOut io.Writer) emit.Result {
	c := guard.NewClient(guard.NewPolicy(), nil)
	if err := auth.Login(c, errOut); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	return authStatus()
}
