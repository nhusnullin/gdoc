// The entry point, the dispatch table and the auth commands.
//
// The package comment is in doc.go. This file holds the one rule the three
// pieces here share: whatever happens, the caller reads one JSON object on
// stdout and an exit code that agrees with it.
//
// # auth status is a report, and being signed out is an answer
//
// Status carries auth_mode, token_path, client_source and token_present, plus
// expired, scopes and missing_scopes once a token is there. No token is
// ok: true with token_present: false, not a failure: a person asking whether
// they are signed in has been answered. TestAuthStatusWithNoTokenIsStillAReport
// and TestAuthStatusReportsAPresentToken are the pins.
//
// The shape is auth.StatusReport, a struct with json tags, never a map. The
// command reads its fields in Go, so a field renamed in auth cannot silently
// drop a warning here.
//
// Three things about those fields:
//
//   - auth_mode is the constant "oauth". gdoc has one credential and no service
//     account, so the field is a statement rather than a resolution.
//   - client_source is always "bundled". Login does not read an
//     oauth-client.json in the config dir yet, so when one is there status adds
//     client_file_ignored: true and a warning, rather than claiming an override
//     that is not wired up.
//   - missing_scopes names what gdoc asks for that the token does not carry. A
//     partial grant is reported, never refused: the login worked, and this is
//     the one place that can say why the Docs calls will 403 before they do.
//     What counts as missing is auth.MissingScopes, and the doc comment on the
//     coveredBy it rests on says why a scope that covers another is not
//     reported.
//     TestAuthStatusWarnsAboutAScopeTheTokenDoesNotCarry is the pin.
//
// A token file that exists and cannot be read is a failure, not
// token_present: false. It comes back ok: false with the path still in data and
// with the warnings it would have carried on the way out. Reported as signed
// out, somebody runs auth login, overwrites the file, and never learns what was
// wrong with it. Only an absent file means signed out.
// TestAuthStatusFailsOnAnUnreadableToken and
// TestAFailingStatusStillCarriesItsWarnings are the pins.
//
// # auth status names the account, and nothing else reads it
//
// The report also carries account and account_name, read live through
// accountOf, the one room that grants guard.AllowAccountRead. That makes
// `gdoc auth status` the grant's second caller and the binary's whole widening
// in M15: no new grant, no new host and no new request kind.
// TestAuthStatusNamesTheAccountItReadsLive is the pin, and
// TestOnlyAccountOfCallsAllowAccountRead with TestAccountOfHasTwoCallers in
// go/boundary hold that the callers are two.
//
// Three reports ask nothing. No report, no token, and a token missing a scope
// gdoc asks for: the read would be refused by the scope that is missing, and
// the warning that already says to sign in again is the answer.
// TestNoTokenMakesNoAccountRequest and TestAMissingScopeMakesNoAccountRequest.
//
// The read is bounded by accountCeiling, five seconds, because a person typed
// the command and is waiting at a prompt. A read that fails or never comes back
// is a warning and never a failure: the token is the fact, and whose it is was
// what could not be read. TestTheAccountCeilingIsFiveSeconds,
// TestAuthStatusOfflineStillAnswersWithoutTheAccount and
// TestAnAccountReadThatHangsStopsAtTheCeiling.
//
// The read's own warnings go out with the report. It goes through an ordinary
// session, which refreshes an expired access token and saves it before the
// request leaves, and the warning is what says it did. A read that failed
// carries them too, because the refresh had already happened.
// TestTheAccountReadsWarningsReachTheObject,
// TestAFailedAccountReadStillCarriesTheSessionsWarnings and
// TestTheLoginToolCarriesTheAccountReadsWarnings for the chat's own answer.
//
// After a read that answered, the token file is read again, so `expired` says
// what the file says now and never contradicts that warning; after one that
// failed, the first reading stands: TestExpiredIsWhatTheFileSaysAfterTheAccountRead
// and TestAFailedAccountReadLeavesExpiredAsRead. The panel's account row names
// the person beside the address when Google said who it is:
// TestThePanelNamesThePersonBesideTheAddress.
//
// auth login does not ask. It just made a browser trip, and the object a skill
// reads after a login is the object it read before:
// TestAuthLoginCarriesNoAccount.
//
// On a terminal the same facts are a panel on stderr: the state in words,
// signed in, signed out or scopes missing, then the account, the file, the
// scopes, what is missing, the client and the mode. No minutes, because when a
// token expires is a field for a program. The object stays on stdout, because
// for this command the object is the answer, and a pipe reads no screen at all.
// TestTheStatusScreenSaysTheStateAndTheAccount and
// TestAuthStatusOnAPipeWritesNoStderr.
//
// # auth logout removes one file, and tells Google nothing
//
// The answer is token_path and removed, and nobody signed in is
// ok: true with removed: false, for the reason auth status is an answer when
// there is no token: the question has been answered.
// TestTheLogoutCommandAnswersWhatItRemoved is the pin.
//
// cmdAuthLogout takes no context and opens no session. internal/auth's Logout
// removes the file and reads its absence back, and its doc comment holds the
// four refusals; the report goes out beside a refusal as well, so a caller
// that was refused still reads which file was left alone. What this layer
// decides is one thing: the sentence a reader meets says the grant stays at
// Google until they take it away in their account settings, because a command
// called logout that leaves the grant standing has to say so where somebody
// reads it. TestTheLogoutSummarySaysTheGrantStaysAtGoogle is the pin, and
// TestLogoutMakesNoRequest in internal/auth is the other half.
//
// It is no chat tool, and that is the one entry of the table with a reason for
// being out of mcpTools that is not about documents: a model that could sign
// the person out could stop every tool of its own session working, and nobody
// in the chat asked it to. TestLogoutIsNoChatTool and
// TestNothingInTheChatSignsAnybodyOut in go/boundary.
//
// # auth login prints the URL to stderr, and waits on one line
//
// The login is the browser trip in its two halves, auth.StartLogin and
// Pending.Wait, which is what the MCP login tool already called: the listener
// and the code exchange belong to internal/auth, and the stream a person reads
// belongs here. The link goes to stderr, the browser is waited for, the token
// is saved, and the run then reports what auth status would. The URL is a
// human word and human words have one place to go, and it is not stdout.
// TestLoginPrintsTheURLToStderrNotStdout is the pin,
// TestAFailedLoginIsOneFailingEnvelope is the other half, because a login that
// did not happen is still one object, and TestTheLoginClosesItsListenerEitherWay
// holds the port freed whichever way the wait ended. A build with no client
// secret is refused before the link line, with nothing on stderr at all:
// TestLoginRefusesABuildWithNoClientSecret runs the real login, unstubbed.
//
// On a pipe, on a file and in a window with no room for a box, the two lines
// are auth.LinkLine's own, byte for byte, because a skill and a log read them:
// TestThePipeLoginLineIsTodays and TestANarrowWindowGetsTheLoginLineAndNoBox.
//
// On a terminal the sentence sits in a box labelled with the host the link
// goes to, the link is one line of its own under the box so that a copy gets
// all of it, and the wait is one line that rewrites itself and ends as a tick
// and "signed in" or as a cross and the error the object carries:
// TestTheLoginScreensAreTheirRecordedBytes. Nothing moves the cursor and
// nothing turns auto-wrap off, because a login traps no signal: a Ctrl-C can
// land anywhere in those three minutes and has to leave the terminal as it
// found it. TestTheLoginSpinnerMovesNoCursorButCarriageReturn reads the bytes
// for both.
//
// # Five facts of the record are filled in here
//
// internal/presence is a file and a clock: it reads no environment, knows
// nothing of a Drive id and nothing of a typed error. So five of the things a
// line says are decided in this package, and the record keeps each as it was
// handed it and judges none of them.
//
// via: code is one environment variable, read once in dispatch. CLAUDECODE is
// what Claude Code's own Bash tool hands a child, measured on 2026-10-17 and
// written down in docs/v2/MEASURED.md, and any value of it counts rather than
// the literal 1, because the variable is Claude Code's and a release that
// writes true in it is still Claude Code. viaOfEnv marks the context and
// leaves a context somebody else marked alone, so a tool call of gdoc mcp
// stays chat whatever started the process: TestViaIsCodeUnderClaudeCode and
// TestViaChatWinsOverCodeInTheMcpSession.
//
// interrupted is lifted out of the waited object `gdoc comments --wait`
// prints, which is the one place in the binary that knows a wait ended early,
// and a wait nobody stopped writes nothing rather than a false:
// TestAStoppedWaitsEndSaysSo.
//
// made is the document a publish created. presenceMade names the one command
// it is true of and puts the id through documentID, the resolver the first
// word of a line goes through, so a path or a sentence under that key reaches
// no file that lasts; a publish that took its document back names nothing:
// TestAPublishEndNamesTheDocumentItCreated,
// TestARolledBackPublishNamesNoDocument and TestOnlyPublishCarriesMade.
//
// stop and line are stop.go, this package's one reader of the typed failures
// the other packages raise: a picture the house style cannot carry with the
// line of the note it sits on, Drive's status, the guard's refusal, a missing
// token, more than one tab, a wire that failed, and other for everything
// else. Nothing there reads a sentence, so a milestone that rewords a refusal
// moves no code: TestAStopIsOneOfTheSevenCodesAndNeverASentence,
// TestTheSVGRefusalCarriesItsLine, TestANotFoundIsNotFound,
// TestAnAuthRefusalIsAuth, TestAGuardRefusalIsRefused and
// TestAHandshakeTimeoutIsNetwork. The word is filled in by dispatch's own
// deferred hook where the command that failed named none, because an absent
// stop beside ok: false reads as a run that failed for no reason at all:
// TestAnUnknownFailureIsOther. Both travel on emit.Result in fields the
// printed object does not carry, the way Screen does, so none of this changes
// a byte a skill reads.
//
// step is the four parts of a publish, and the one of the five this package
// does not hand over: dispatch puts the run on the context and publish calls
// presence.Step beside the work, so no room between here and the work is
// handed a run. The names are internal/presence's own and the line is written
// immediately before the part begins, which for the two network parts means
// the line is on disk before the request leaves:
// TestAPublishStepsAreTheFixedNamesInOrder, TestTheFirstStepsEndCarriesTheTitle
// and TestAPanicAfterAStepStillWritesTheEnd, with go/boundary's
// TestStepHasOnlyPublishAsCaller over the one room that calls it.
//

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
	"gdoc/internal/presence"
	"gdoc/internal/settings"
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
//
// The three setting flags come off the line here, before the word mcp is looked
// for and before --help is, so one reading of them serves every command and
// the session alike. What is left is the line as it always was, and the
// settings go down beside it. settings.go holds the reading, and the refusal
// it gives takes the shape of whatever the line was going to be: one envelope
// for a command, one line on stderr for a session, which is what checkMCPArgs
// already does with a word it does not know. That is why takeSettings reads the
// line to the end even when it refuses it: the word mcp is still on it with
// every setting flag off, and a session told its reason in an envelope would be
// told it on the protocol stream.
// TestMcpTakesTheSettingFlagsAndNothingElse.
func route(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	rest, given, err := takeSettings(args)
	var set settings.InEffect
	if err == nil {
		set, err = resolveSettings(given)
	}
	if _, asked := helpAsked(rest); !asked && len(rest) > 0 && rest[0] == "mcp" {
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return serveMCP(ctx, in, out, errOut, rest[1:], set)
	}
	if err != nil {
		return printResult(emit.Result{OK: false, Error: err.Error()}, out, errOut)
	}
	return run(ctx, rest, out, errOut, set)
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
func run(ctx context.Context, args []string, out, errOut io.Writer, set settings.InEffect) int {
	// The run's own line and the run's own sweep, which is one of each per
	// process in production and one of each per call here, so no test inherits
	// what another said: internal/presence's Fresh.
	presence.Fresh()
	return printResult(safeDispatch(ctx, args, errOut, set), out, errOut)
}

// printResult is the result on its streams and the exit code that agrees with
// it, which is the whole of run once the work is done. It is named apart
// because route has one answer of its own: a line whose setting flags could
// not be read never reaches a command, and it is still one JSON object with
// the exit code that goes with it.
func printResult(r emit.Result, out, errOut io.Writer) int {
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
func safeDispatch(ctx context.Context, args []string, errOut io.Writer, set settings.InEffect) (r emit.Result) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Fprintf(errOut, "gdoc crashed: %v\n%s\n", p, debug.Stack())
			r = emit.Result{OK: false, Error: fmt.Sprintf("gdoc crashed: %v", p)}
		}
	}()
	return dispatching(ctx, args, errOut, set)
}

// dispatching is dispatch behind a variable, as login, openSession and
// serveMCP are, so a test can read the line and the settings that reached it
// without a command running: TestTheSettingFlagsAreTakenAnywhereOnTheLine and
// TestTheMcpSessionHandsItsSettingsToEveryCall. Nothing in the binary assigns
// it.
var dispatching = dispatch

// dispatch takes the context so a test can hand a wait one that is already
// done, which is the answer a stopped session gets. The signal itself is
// trapped inside the wait rather than here: every other command makes its
// requests and ends, and Ctrl-C kills it the way it always did.
// set is what this run writes, resolved in route before the line was read.
// Every command is handed it on its args, so a command that writes into a
// thread asks the value rather than a constant of its own.
//
// The result is named for the presence hook alone: the end of a run is written
// in a deferred call, so a command that panicked still ends its run, with the
// zero result's ok: false, before safeDispatch above turns the panic into the
// envelope: TestAPanicInACommandStillWritesAnEndWithOkFalse.
func dispatch(ctx context.Context, args []string, errOut io.Writer, set settings.InEffect) (r emit.Result) {
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
	// --brief is not read here. It is a flag of help and of no other
	// command, so on this route it is one more word that names none, and
	// `gdoc --help --brief` is the refusal it has always been:
	// TestHelpBriefIsAFlagOfHelpAndOfNothingElse.
	if rest, asked := helpAsked(args); asked {
		words, json := helpWords(rest)
		return cmdHelp(ctx, words, json, false, errOut, set)
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
	a.set = set
	// Who is running this, where the environment says so. It is one read, here,
	// because internal/presence is a file and a clock and reads no environment
	// of its own, and a session of gdoc mcp has already marked its context,
	// which this leaves alone: TestViaIsCodeUnderClaudeCode and
	// TestViaChatWinsOverCodeInTheMcpSession.
	ctx = viaOfEnv(ctx, os.Getenv)
	// And the record, after the parse and before the work: a line that was
	// refused is no run, so nothing is written for it, and the document the
	// event names is the one the parsed line named. The two calls are the same
	// two lines whether the flag is on or off, because Begin answers nil where
	// it is off and every method on a nil run is a no-op:
	// TestTheFlagChangesNoByteOfTheObject.
	rec := presence.Begin(ctx, set.Presence && c.report, releaseVersion(),
		c.name, presenceDoc(*c, a), a.has("--wait"), c.readsBack, errOut)
	defer func() {
		// And the one word a failed run says why in, where the command that
		// failed named none: a failure gdoc did not sort is other rather than
		// nothing, because an absent stop reads as a run that failed for no
		// reason at all. stop.go holds the words and what names each of them;
		// the field never reaches the object, so this changes no byte a caller
		// reads: TestAnUnknownFailureIsOther.
		if !r.OK && !r.Stop.Known() {
			r.Stop, r.StopLine = emit.StopOther, 0
		}
		rec.Made(presenceMade(*c, r))
		rec.End(r)
	}()
	// And where this run is recorded, the app that shows it is opened if it is
	// not already open. Here, after the start line and before the work, so a
	// person reading the panel sees the run that opened it; and only where rec
	// is not nil, which is the presence setting and a command that reports, so
	// the flag off starts nothing: desktop.go's openPresenceApp says what it
	// asks before it starts anything, and
	// TestACommandUnderTheFlagOpensTheAppOnceAndNothingWhenItIsRunning.
	//
	// Every command but the one that replaces the bundle. `gdoc update` opens
	// the app once itself, after it has placed or rolled back the app, which
	// is what the 2026-10-18 decision says; an open here would start the
	// bundle this run is about to rename away, and the one it then placed
	// would find the lock held by the build it replaced:
	// TestAnUpdateOpensTheAppItPlacedAndNotTheOneItReplaces.
	if rec != nil && c.name != updateCommand {
		openPresenceApp()
	}
	// The run goes on the context, so the one command with parts can say which
	// of them it is on without a run threaded through every room between here
	// and the work: presence.WithRun and publish.go's three calls.
	return c.run(presence.WithRun(ctx, rec), a, errOut)
}

// claudeCodeEnv is the one variable that says Claude Code started this
// process. Measured 2026-10-17 from the Bash tool of a Claude Code session,
// which hands a child CLAUDECODE=1 beside its other CLAUDE_CODE_* variables;
// the measurement and its method are in docs/v2/MEASURED.md.
//
// Any value counts, and not the literal 1: the variable is Claude Code's own
// and a release that writes true or 2 in it is still Claude Code, while a
// record that said terminal in that case would be wrong about the one thing
// the field is for.
const claudeCodeEnv = "CLAUDECODE"

// claudeSessionEnv is Claude Code's own id for the session that started this
// process, which it hands a child beside CLAUDECODE. Measured 2026-10-18 from
// this session's own environment; the measurement is in docs/v2/MEASURED.md.
//
// It reaches the start line of a code run and nothing else, so the panel can
// offer to go back to the session a document was worked on from. A Claude Code
// that sets no session is still Claude Code, and the line then says nothing
// about a session: TestTheSessionOfACodeRunIsOnItsStart.
const claudeSessionEnv = "CLAUDE_CODE_SESSION_ID"

// viaOfEnv marks the run's context for the record where the environment says
// Claude Code started this process, and leaves every other run as it found it.
//
// getenv is handed in, as internal/tty takes it, so the rule is measured
// without a test setting a variable of the real process:
// TestViaIsCodeUnderClaudeCode.
func viaOfEnv(ctx context.Context, getenv func(string) string) context.Context {
	if getenv(claudeCodeEnv) == "" {
		return ctx
	}
	return presence.WithCode(ctx, getenv(claudeSessionEnv))
}

// presenceDoc is the document an event names: the id the first word of the line
// resolved to, for the commands whose first word is a document, and nothing for
// the rest.
//
// The first word is compared with the literal the table writes, as read.go
// asks for "--wait" by its own, because the word is what a reader sees in the
// usage line and not a name this file owns.
//
// A word that is no document at all answers the empty string rather than a
// refusal, because the refusal is the command's own to give: the parse has
// happened and the command has not run yet, so nothing here may decide what the
// caller is told. TestAnUpdateAndALoginWriteNoDoc.
func presenceDoc(c command, a *args) string {
	if len(c.words) == 0 || c.words[0] != "<url>" {
		return ""
	}
	id, err := documentID(a.target())
	if err != nil {
		return ""
	}
	return id
}

// presenceMade is the document a run created: the id in publish's own answer,
// for the one command that creates a document, and nothing for every other
// command, whose document was handed in and is already the event's doc.
//
// The id goes through documentID, the resolver the first word of a line goes
// through, so only something shaped like a Drive id reaches a file that lasts.
// A publish that took its document back clears the id in its data and so
// names nothing here: TestARolledBackPublishNamesNoDocument.
//
// The command is named here rather than in the table, because this is the one
// command it is true of and a column with one true row says less than a line
// that names it: TestOnlyPublishCarriesMade.
func presenceMade(c command, r emit.Result) string {
	if c.name != "publish" {
		return ""
	}
	data, ok := r.Data.(publishData)
	if !ok {
		return ""
	}
	id, err := documentID(data.DocumentID)
	if err != nil {
		return ""
	}
	return id
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
		return withStop(emit.Result{OK: false, Error: err.Error(),
			Data: statusReport(report, gapi.Account{}), Warnings: warnings}, err)
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
		return failed(err)
	}
	return authStatus(ctx, false)
}

// cmdAuthLogout is the token removed from this computer, and nothing else.
//
// It takes no context and opens no session, because there is nothing to ask
// anybody: internal/auth removes one file and reads its absence back. The
// report goes out beside a refusal as well as beside an answer, so a caller
// that was refused still reads which file was left alone.
//
// What it is not is a sign-out at Google. The grant lives in the person's
// account settings until they take it away there, which is what the command's
// own sentence says: TestTheLogoutSummarySaysTheGrantStaysAtGoogle.
func cmdAuthLogout() emit.Result {
	report, err := auth.Logout()
	if err != nil {
		return withStop(emit.Result{OK: false, Error: err.Error(), Data: report}, err)
	}
	return emit.Result{OK: true, Data: report}
}
