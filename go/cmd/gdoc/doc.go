// Command gdoc reads and writes one Google Doc. Facts in, JSON out, exit.
//
// This package is the argument layer and nothing else. It turns words into a
// document id, opens a guard.Policy naming exactly what this run may reach,
// opens a gapi.Session on the guard's client, and hands what came back to a
// pure package that decides nothing. The skill reading the JSON judges.
//
// This comment holds why the command layer refuses what it refuses, with the
// test that pins each rule. What the rules are is in the code beside them.
//
// # One JSON object, and the exit code says which
//
// Exactly one JSON object reaches stdout, always through internal/emit, and the
// exit code is 0 if and only if that object says ok. Human words go to stderr,
// the login URL among them. A caller parses stdout whole and never has to find
// the object inside something else.
//
// TestOnlyJSONObjectRefusesAnythingAfterTheObject is the pin, and it checks the
// test helper in both directions: a helper that accepts a trailing byte would
// pass every other test in this package over output no skill can read.
//
// # The twelve commands
//
// commands.go holds the table, and dispatch matches what is in it and nothing
// else. Each entry carries the words a caller types, the flags it takes with a
// sentence for each, one example, and the function that runs it. Nothing else
// in this package lists a command or a flag: the usage line is joined from the
// table, and the parser a command is handed is built from the flags its entry
// names. The list below names the package that does the work, so a reader
// looking for a rule starts there rather than here.
//
//   - auth status: the OAuth state, from internal/auth.
//   - auth login: the browser trip, internal/auth and internal/auth/loopback.
//   - read <url> [--structure]: the text projection, or the tree with the flag.
//     internal/docs reads, internal/view projects.
//   - comments <url> [--since] [--wait] [--witness]: the threads with their
//     ranges, markers, replies and the next cursor. internal/comments, with
//     internal/docx for the witness.
//   - suggestions <url> [--md]: what is pending, and with a paired note what
//     stopped being pending. internal/suggestions, internal/frontmatter.
//   - restyle <url> --dry-run | --from [--fields]: the survey, the house style
//     in place, and the proposed prelude. internal/restyle, internal/prelude.
//   - probe --folder: whether Docs honours SUGGEST today. internal/probe.
//   - reply <url> <comment id> --body-file: one robot reply. internal/reply.
//   - propose <url> --from --folder [--md]: a change as a suggestion.
//     internal/propose.
//   - withdraw <url> <suggestion id> --md: gdoc taking back its own proposal.
//     internal/withdraw.
//   - build --md --out [--house] [--force]: the house-style docx, no network
//     at all. internal/house, internal/cover, internal/body, internal/render.
//   - publish --md --folder-id [--house]: that docx into Drive as a Google
//     Doc. internal/publish.
//
// The usage line is the whole of the help today, so it has to name every
// command that exists. Three tests hold the table and the usage line together.
// TestTheUsageLineNamesEveryCommand spells the twelve out word for word, as a
// reader sees them, so it cannot follow a rename in the code.
// TestEveryCommandInTheTableIsDispatchedAndNothingElseIs runs every entry and
// asks it to refuse a flag, so a thirteenth command cannot sit in the table
// unreachable, and a word in no entry is refused as unknown.
// TestEachCommandParsesWithTheFlagSetItsTableEntryDescribes and
// TestNoCommandBuildsAFlagSetOfItsOwn hold the flags the same way: a command
// takes what its entry names, and has nowhere else to keep a flag.
// TestEveryFlagIsReadTheWayItsKindSays is the third direction, that a flag the
// table says carries a file is read for a value and one that carries none is
// read for its presence.
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
// # auth login prints the URL to stderr
//
// Login prints the authorization URL to stderr, waits for the browser to come
// back to the loopback listener, saves the token, then reports what auth status
// would. The URL is a human word and human words have one place to go, and it
// is not stdout. TestLoginPrintsTheURLToStderrNotStdout is the pin, and
// TestAFailedLoginIsOneFailingEnvelope is the other half: a login that did not
// happen is still one object.
//
// # The binary never prompts, and --help is a failure
//
// Nothing here reads stdin. A command missing something fails and says what is
// missing, and does not ask. There is no help command either, so gdoc --help is
// ok: false and exit 1, with the one-line usage string in the error. That is
// deliberate rather than an oversight: readable help would have to reach stdout
// beside the object, or exit 0 on a run that did no work, and both break the
// contract every caller has. TestUnknownCommandFailsAndNamesItself is the pin,
// and TestNoArgumentsFails covers the bare word, which quotes nothing back
// because unknown command "" names nothing and reads like a fault in the tool.
//
// TODO(test): no test pins that the binary never reads stdin. Nothing in the
// tree names os.Stdin today, so the rule holds by absence rather than by a
// check somebody would see fail.
//
// # A panic is still one envelope
//
// safeDispatch recovers, prints ok: false with what happened, and puts the
// stack trace on stderr. Without it a crash prints a Go trace, nothing at all
// on stdout, and exits 2, which is the one shape no caller can read.
// TestAPanicIsStillOneEnvelope is the pin.
//
// # GDOC_CONFIG_DIR moves the config dir
//
// It is how every Go test avoids the real config, and it is read in one place,
// internal/config. A test that wrote to the developer's own token file would
// be a test that signs them out.
//
// # Argument parsing is strict
//
// An unknown flag, a repeated flag, a missing value, an empty value written
// either way, an extra positional argument, and a flag standing where another
// flag's value belongs each fail naming the offender. A command that accepts
// and ignores what it did not understand tells the caller it did something it
// did not.
//
// The last of those is why parseArgs looks the next argument up in the
// command's own flag set rather than refusing anything that starts with a dash:
// a cursor is base64url, and "-" is in that alphabet, so a real cursor value
// would be refused as a flag.
//
// TestTheReadCommandsRefuseArgumentsTheyDoNotUnderstand,
// TestAFlagIsNotSwallowedAsAnotherFlagsValue,
// TestAnEmptyFlagValueIsRefusedBothWaysItCanBeWritten and
// TestTrailingArgumentsAreRefused are the pins, with
// TestRestyleArgumentsAreStrict, TestPublishArgumentsAreStrict and
// TestBuildRefusesAMissingFlagByName over the commands that take more.
//
// The same rule holds for the commands that take nothing. auth login takes no
// words and no flags, so gdoc auth login --token /path is refused naming the
// flag rather than read as a plain login that quietly ignored it, and bare
// gdoc auth matches no entry and is an unknown command.
//
// # Only the wait traps a signal
//
// cmdComments installs signal.NotifyContext on SIGINT and SIGTERM around the
// wait itself and takes it off on the way out. Ctrl-C during a wait prints
// ok: true with no threads, the cursor handed in and waited.interrupted: true,
// and exits 0. The output contract has to hold under the one signal a live
// session sends every time it ends.
//
// It is not in main, and that is the decision. signal.Notify takes the default
// kill away from the whole process for as long as it is installed, and
// NotifyContext never puts it back on its own. Trapped in main, Ctrl-C stops
// being an answer for every command that does not read the context: auth login
// would hold the terminal for its three minute login timeout, and a propose in
// the middle of writing into somebody's document could not be stopped at all.
//
// Two halves follow from that, and both are the same rule. The trap comes off
// the moment the wait returns, because what runs after a wait is the --witness
// export, and a Ctrl-C there has to kill the run the way it kills every other
// command. And the wait's context is derived from the caller's rather than
// shadowing it, so what runs after the wait cannot be quietly cancelled by the
// signal that ended the wait.
//
// TestOnlyTheWaitTrapsTheSignal is the pin, and it states the rule in both
// directions: os/signal is named in read.go and in no other production file
// here. TestAnInterruptedWaitIsAnAnswerAndNotAFailure covers the answer.
// dispatch still takes a context, because a test hands a wait one that is
// already done.
//
// # The note is read again just before it is written
//
// The pairing is checked before the session opens, and the run then spends
// seconds to tens of seconds on the network: a probe plus a read, a write and
// three read-backs per proposal for propose, and two whole-document reads plus
// a batchUpdate for withdraw. These notes live in a synced vault, so writing
// back the bytes the run started with would throw away whatever landed in that
// window.
//
// freshNote reads the file again and refuses four things rather than writing
// them: a file it cannot read again, one whose front matter no longer parses,
// one whose gdoc: block has gone, and one that now names another document. Each
// is a warning carrying the reason, and nothing is written into the note.
//
// The block is re-parsed from the fresh bytes too, so a proposal another run
// recorded in that window survives: writing the block this run read into bytes
// it did not would keep the author's prose and still drop that entry. notePath
// therefore keeps the path and the block the pairing check read, and never the
// bytes it read them from, because a copy held there would only be the stale
// bytes somebody later wrote back.
//
// TestProposeWritesTheNoteAsItStandsWhenTheRunFinishes and
// TestWithdrawWritesTheNoteTheVaultHasNow are the pins, with
// TestProposeWarnsWhenTheNoteStopsNamingThisDocumentMidRun over the fourth
// refusal. That last one is a different rule from
// TestProposeRefusesANoteThatNamesAnotherDocument, which is the pairing check at
// the door: that one fails the run before anything is sent, and this one is
// reached with the proposals already in the document, so it warns instead.
//
// publish reads the note again for the opposite reason, and that rule is
// pair's, in publish.go beside this file: there the block appearing during the
// upload is what is refused, and the refusal is a rollback that internal/publish
// only carries out.
package main
