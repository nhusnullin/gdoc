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
	"time"

	"gdoc/internal/auth"
	"gdoc/internal/emit"
	"gdoc/internal/gapi"
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
//
// It is the trip in its two halves, startLogin and Wait, which is what the MCP
// login tool already calls: the listener and the exchange belong to
// internal/auth, and the stream a person reads belongs here. internal/auth
// offers no call that is both halves, because the waiting line goes between
// them. loginscreen.go draws what goes on that stream.
//
// The wait has no context of its own, as it never had: nothing above a login
// can cancel a run already waiting on a browser, and internal/auth's own
// three-minute ceiling is what bounds it.
var login = func(errOut io.Writer) error {
	trip, link, err := startLogin()
	if err != nil {
		return err
	}
	defer trip.Close()
	return waitForBrowser(context.Background(), errOut, trip, link)
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
// Three rooms ask it: run(), for whether a screen replaces the object, which
// it asks of both streams; helpscreen.go, for whether a screen is drawn at all
// and whether a window with no room for one is still a person's; and the step
// list in progress.go, for whether to redraw in place.
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
//
// Both streams are asked, because the words went to errOut: a run with stdout
// on a terminal and stderr redirected wrote those words into somebody's file,
// where nobody reading the terminal can see them, so there the object is the
// only answer there is and it is printed.
// TestHelpOnATerminalWritesNothingToStdout,
// TestBareGdocOnATerminalWritesNothingToStdoutAndExitsOne,
// TestEveryOtherCommandKeepsItsObjectOnATerminal,
// TestAScreenWhoseWordsWentToAFileKeepsItsObject and
// TestNoEscapeByteReachesAPipe.
func run(ctx context.Context, args []string, out, errOut io.Writer) int {
	r := safeDispatch(ctx, args, errOut)
	// The version is set here and nowhere else, so every object carries it:
	// the answers, the refusals and the panic envelope alike. A report of
	// something odd names the build that did it without anyone being asked.
	r.Version = releaseVersion()
	if r.Screen && isTerminal(out) && isTerminal(errOut) {
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
// has to do is be readable at the width the reader has. It paints with the
// depth the environment says, and it may: run() asks whether errOut is a
// terminal before it drops an object, so this line is never written into a
// pipe or a file.
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
	// help's does, and the screen opens with the refusal's own words. The hint
	// does not follow it: see writeJSONHint.
	if len(args) == 0 {
		writeBareHelp(errOut)
		return emit.Result{OK: false, Error: needsCommand + " " + usageLine(), Screen: true}
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

// statusData is the auth report with the version beside it, and the account
// the token signs in as. `gdoc auth status` is the command a person runs to ask
// what they have, so the build is one of the facts it reports, not only a
// field of the envelope around it, and so is whose account it is.
type statusData struct {
	*auth.StatusReport
	Account     string `json:"account,omitempty"`
	AccountName string `json:"account_name,omitempty"`
	Version     string `json:"version,omitempty"`
}

// accountCeiling is how long `gdoc auth status` waits for the account read. A
// person typed the command and is waiting at a prompt, and the read is one
// small GET: long enough for a slow network, short enough that a read nobody
// answers is still an answer. It is a var so a test can hand it a hung read
// and finish: TestTheAccountCeilingIsFiveSeconds and
// TestAnAccountReadThatHangsStopsAtTheCeiling.
var accountCeiling = 5 * time.Second

// accountUnread is what the reader is told when the token is there and who it
// belongs to is not. The same sentence the MCP login tool uses, because it is
// the same fact.
const accountUnread = "the token is there and which Google account it belongs to could not be read: "

// statusReport is the report as it goes out. A nil report stays nil rather
// than becoming an object holding nothing but a version: a config dir gdoc
// cannot locate has no facts to report, and that is what the error says.
func statusReport(r *auth.StatusReport, acc gapi.Account) any {
	if r == nil {
		return r
	}
	return statusData{StatusReport: r, Account: acc.Email, AccountName: acc.Name, Version: releaseVersion()}
}

// authStatus is the report. withAccount is whether it also names the account,
// which `gdoc auth status` asks for and `gdoc auth login` does not: a login
// just made a browser trip, and the object a skill reads after one is the
// object it read before. TestAuthLoginCarriesNoAccount.
func authStatus(ctx context.Context, withAccount bool) emit.Result {
	report, err := auth.Status()
	warnings := statusWarnings(report)
	if err != nil {
		// The report goes out beside the error, warnings included: the caller
		// still learns which file was being read and what else is in the way,
		// and the error names what was wrong with it. The run that fails is the
		// one where the extra fact is worth most.
		return emit.Result{OK: false, Error: err.Error(), Data: statusReport(report, gapi.Account{}), Warnings: warnings}
	}
	acc, read := accountFor(ctx, report, withAccount)
	warnings = append(warnings, read...)
	return emit.Result{OK: true, Data: statusReport(afterRead(report, acc), acc), Warnings: warnings}
}

// afterRead is the report as the file stands once the account read is done.
// That read may have refreshed an expired access token and saved it, so the
// report read before it would say `expired: true` beside a warning that the
// token was just refreshed. A read that answered went out with a working
// token, so the file is read again; one that failed proves nothing, and the
// first reading stands. A second reading that fails leaves the first one too:
// the first was good, and the account read did not make it less so.
// TestExpiredIsWhatTheFileSaysAfterTheAccountRead and
// TestAFailedAccountReadLeavesExpiredAsRead.
func afterRead(report *auth.StatusReport, acc gapi.Account) *auth.StatusReport {
	if acc.Email == "" {
		return report
	}
	fresh, err := auth.Status()
	if err != nil || fresh == nil {
		return report
	}
	return fresh
}

// accountFor is who the token signs in as, and the warnings that read came
// with: the session's own, and the one that says it could not be read. It is
// the second caller of accountOf in this binary, and the guard moved by that
// caller rather than by a request: TestAccountOfHasTwoCallers.
//
// The session's warnings are the reason this hands back a list. The read goes
// out through an ordinary session, which refreshes an expired access token and
// saves it first. The warning is where a reader is told that it did:
// TestTheAccountReadsWarningsReachTheObject. The report is read again after a
// read that answered, so `expired` says what the file says now: afterRead.
//
// Three reports are not asked at all. A report that is not there has no token
// to read with; neither has one saying there is no token; and a token missing a
// scope gdoc asks for would have the read refused by the scope, so the warning
// that already says to sign in again is the whole answer:
// TestNoTokenMakesNoAccountRequest and TestAMissingScopeMakesNoAccountRequest.
//
// A read that fails is a warning and never a failure. The token is the fact,
// and whose it is was what could not be read:
// TestAuthStatusOfflineStillAnswersWithoutTheAccount.
func accountFor(ctx context.Context, r *auth.StatusReport, withAccount bool) (gapi.Account, []string) {
	if !withAccount || r == nil || !r.TokenPresent || len(r.MissingScopes) > 0 {
		return gapi.Account{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, accountCeiling)
	defer cancel()
	acc, read, err := accountOf(ctx)
	warnings := append([]string{}, read...)
	if err != nil {
		return gapi.Account{}, append(warnings, accountUnread+err.Error())
	}
	return acc, warnings
}

// cmdAuthStatus is the report and the panel a person reads it from. The object
// goes out as it always did, because for this command the object is the
// answer: the panel is the same facts for the reader who typed the command.
// TestTheStatusScreenSaysTheStateAndTheAccount and
// TestAuthStatusOnAPipeWritesNoStderr.
//
// A failing status draws none: the report is then a file that could not be
// read rather than a state somebody is in, and the error on stdout is what
// says so.
func cmdAuthStatus(ctx context.Context, errOut io.Writer) emit.Result {
	r := authStatus(ctx, true)
	if d, ok := r.Data.(statusData); ok && r.OK {
		writeStatusScreen(errOut, d)
	}
	return r
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
func authLogin(ctx context.Context, errOut io.Writer) emit.Result {
	if err := login(errOut); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	return authStatus(ctx, false)
}
