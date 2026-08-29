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
)

const usage = "Commands: auth status"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

// run turns arguments into one JSON object on w and an exit code.
func run(args []string, w io.Writer) int {
	r := dispatch(args)
	if err := emit.Print(w, r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return emit.ExitCode(r)
}

func dispatch(args []string) emit.Result {
	if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
		return authStatus()
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
