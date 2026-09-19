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
// # The commands, and the one table that describes them
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
//   - annotate <url> --quote | --from [--body-file]: a comment on the words a
//     caller quotes, and nothing else. internal/annotate.
//   - build --md --out [--house] [--force]: the house-style docx, no network
//     at all. internal/house, internal/cover, internal/body, internal/render.
//   - publish --md --folder-id [--house]: that docx into Drive as a Google
//     Doc. internal/publish.
//   - update [--check] [--major] [--nightly] [--rollback]: this binary
//     replaced by a newer release of it. internal/update, and the one reach
//     that carries no credential, gapi.Plain.
//   - help [<command>]: the table itself, as an object and as words. help.go.
//   - completion <shell> --out [--force]: the table as a shell script, written
//     to a file. completion.go, with the template beside it.
//
// The usage line names every command that exists, because it is joined from
// the table. Three tests hold the table and the usage line together.
// TestTheUsageLineNamesEveryCommand spells the sixteen out word for word, as a
// reader sees them, so it cannot follow a rename in the code.
// TestEveryCommandInTheTableIsDispatchedAndNothingElseIs runs every entry and
// asks it to refuse a flag, so a new command cannot sit in the table
// unreachable, and a word in no entry is refused as unknown.
// TestEachCommandParsesWithTheFlagSetItsTableEntryDescribes and
// TestNoCommandBuildsAFlagSetOfItsOwn hold the flags the same way: a command
// takes what its entry names, and has nowhere else to keep a flag.
// TestEveryFlagIsReadTheWayItsKindSays is the third direction, that a flag the
// table says carries a file is read for a value and one that carries none is
// read for its presence. It also holds the one kind that is words rather than
// a name for something somewhere: a text flag is read as it was typed, spaces
// and quotes included, and an empty value is refused naming the flag.
//
// The usage line is held the same way, and for the same reason. Each flag
// carries how it stands in the call beside its kind, so the line brackets what
// may be left out and puts a bar between alternatives: publish reads --md
// --folder-id [--house <file>] and restyle reads --dry-run | --from <file>,
// which is the list above rather than every flag run together. The object
// carries the same thing as a word, because a skill reads the object where a
// person reads the line. TestTheUsageLineMarksWhatIsOptionalAndWhatIsAnAlternative
// spells the fourteen lines out as a reader sees them, TestTheObjectSaysHowEachFlagStands
// holds the word beside them, and TestEveryRequiredFlagIsOneTheCommandRefusesToRunWithout
// is the binary's own witness: a flag the table calls required is refused by
// name when it is missing, and a flag marked wrong in either direction fails
// there before it can reach a reader.
//
// The example is held the same way, because a skill builds its call from what
// help printed and an example the binary would refuse teaches a call that
// fails. TestEveryExampleIsACallTheTableAccepts reads every example without
// running it: it opens with the binary and the command's own words, the rest
// parses with that entry's flag set and word count, and a value whose kind
// says how it is written is written that way, so a duration is 9m and not 60.
// TestTheCommentsExampleShowsTheCursorItsWaitNeeds holds the one rule the
// table cannot see, that cmdComments refuses --wait without --since.
//
// The table is a function and not a variable, and that is Go and not taste.
// help is an entry in it and reads it, so a variable would refer to itself
// through a function, which is an initialization cycle the compiler refuses.
// Each caller is handed its own slice, so nothing keeps a pointer into a table
// somebody else is reading.
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
// # The binary never prompts, and help is an answer
//
// Nothing here reads stdin. A command missing something fails and says what is
// missing, and does not ask.
//
// help is the one command whose whole output is words, and it still keeps the
// contract. The object goes to stdout, the words a person reads go to stderr
// where the login URL already goes, and it exits 0, because a question was
// asked and answered. That is the same shape auth status has with no token.
// TestHelpIsOneObjectAndTheProseIsOnStderr is the pin, with
// TestHelpForOneCommandCarriesItsWordsFlagsAndExample over the shape a skill
// reads and TestHelpMatchesByPrefixAndRefusesWhatItDoesNotKnow over the prefix
// match and the refusal.
//
// Until M7d gdoc --help was ok: false, and the reason written here was that
// readable help would have to reach stdout beside the object or exit 0 on a run
// that did no work. stderr answers the first and a question answered is not a
// run that did no work, so the rule retired. The decision is in
// docs/v2/DECISIONS.md, dated 2026-09-16.
//
// --help and -h are aliases for help, wherever they stand on the line, and they
// are read before the table is walked and before the parser runs. So gdoc
// restyle --from x --help is an answer rather than a refusal of a flag restyle
// does not take, and every parser refusal below is untouched.
// TestDashDashHelpIsAnAliasAnywhereOnTheLine is the pin.
//
// Bare gdoc is the one place the two halves part. It did no work, so it is
// still ok: false and exit 1, with the object unchanged and nothing quoted back
// because unknown command "" names nothing and reads like a fault in the tool.
// The whole help goes to stderr beside it, so the person who typed it reads
// what they could have typed. TestUnknownCommandFailsAndNamesItself,
// TestNoArgumentsFails and TestBareGdocStillFailsAndPrintsTheHelpToStderr are
// the pins, and TestHelpTakesWordsAndNoFlags holds that help itself is parsed
// as strictly as everything else.
//
// TODO(test): no test pins that the binary never reads stdin. Nothing in the
// tree names os.Stdin today, so the rule holds by absence rather than by a
// check somebody would see fail.
//
// # Completion is a file, and the reason is the output contract
//
// gdoc completion <shell> --out <path> renders the command table as a script
// for that shell, writes it through internal/atomicfile, and prints one object
// saying the shell, the absolute path it wrote and the line to add. That last
// key names the file the line goes in, add_to_zshrc or add_to_bashrc, so a
// bash user is never handed a line about .zshrc. The script itself never
// reaches stdout. A script there would make this the one command
// whose stdout is not an object, and the contract is worth more than a file
// that has to be written again after an upgrade. install.sh writes it again on
// every run, or warns saying why it could not, and it never edits .zshrc: it
// prints the line and a person adds it.
//
// The script is a rendering of the table help prints, so a Tab offers a word
// or a flag the table holds and nothing else. A flag whose kind is a file
// offers file names, and every other kind offers nothing, because nothing on
// this machine knows a Drive folder id, a cursor, a wait length or a quotation
// out of somebody's document, and neither does anything know a document URL.
// TestTheZshScriptNamesEveryCommandAndEveryFlag and
// TestTheBashScriptNamesEveryCommandAndEveryFlag are the pins, and each asks
// the table rather than a list of its own, so a command added without a line
// in the script fails there. TestATextFlagIsOpaqueToBothScripts is the pin for
// the text kind, which annotate carries as --quote.
//
// The bash script sets complete -o filenames for the whole command, so a
// directory offered for a file flag gets a trailing slash and no trailing
// space, and a space in a path is escaped as it is inserted. Scoping the option
// to the one arm that offers file names needs compopt, and macOS ships bash
// 3.2, which has none, so it is set for all of them. The cost is the one place
// a Tab offers what the parser refuses: a command word matching a directory in
// the caller's current folder gets a slash too, so in a folder holding build/,
// gdoc bu<Tab> completes to gdoc build/ and the binary refuses it by name. One
// visible character on a rare word against every directory a file flag ever
// descends into. zsh has neither problem, because _files does the work there.
// TestTheBashScriptNamesEveryCommandAndEveryFlag pins the registration line
// the option lives on.
//
// Both shells group the table the same way, through byFirstWord, because both
// complete the way a person types: one word, then a second word or a flag.
// They differ in the language each says it in. zsh reads a description beside
// every word and a spec per flag; bash has neither, so a flag is a word in a
// compgen -W list and what follows it is a case over the word before the
// cursor.
//
// --out is build's rule and build's own check: a file already there is refused
// without --force, a directory is refused whatever the flag says, and a
// refused run writes nothing.
// TestCompletionRefusesAnExistingOutUnlessForced is the pin, with
// TestCompletionWritesTheFileAndReportsTheLineToAdd over what the object says
// and TestCompletionArgumentsAreStrict over the four refusals.
//
// completion counts its own word rather than leaving it to the parser, which
// is why its table entry says anyWords. The parser's refusal for one missing
// word names a document, and what is missing here is a shell.
//
// zsh and bash today, PowerShell at M9 with the Windows smoke test. Each is a
// row in shells(), which is what a refusal reads the known shells out of:
// TestCompletionArgumentsAreStrict, with
// TestCompletionBashWritesTheFileAndNamesBashrc over the second row's report.
//
// No test sources the script in a real shell, because nothing under go/ runs an
// external program and TestNothingRunsAnExternalProgram holds that over the
// test files too. The script is checked structurally here, and a person types
// Tab at it once per milestone.
//
// # The update runs when it is typed; help asks once a day
//
// `gdoc update` is the one command that writes over the binary a person is
// running, and that is why nothing starts it but a person typing it. Nothing
// installs unasked, ever.
//
// Asking what is published is a smaller thing than installing it, and `gdoc
// help` does it by itself, at most once in 24 hours. Help is the one place: it
// is the first call of every skill session, it already carries the version,
// and no document is open in front of it. A colleague who never reads the
// releases page hears about a release from the tool itself, in the session
// they already opened.
//
// What keeps that bounded is that nothing moves. The check reads one listing,
// writes one file of gdoc's own, replaces no binary and touches no document.
// It runs under a two-second ceiling instead of the listing's five, and a
// failed check is written down too, so a network that refuses GitHub costs two
// seconds a day and not two seconds a run: TestTheCheckIsBoundedByTwoSeconds
// and TestAnUnreachableGitHubIsStampedAndHelpStillAnswers. A stamp younger
// than a day is the answer on its own, with no request at all:
// TestAFreshStampMakesNoRequest. A build from a checkout names no release, so
// it has nothing to compare and never asks: TestACheckoutBuildNeverChecks. A
// word that names no command is refused before any of it happens, so a
// mistyped help costs nothing at all:
// TestAnUnknownHelpWordIsRefusedBeforeTheCheck.
//
// No other command checks. There is no check before a build and none on the
// way to reading somebody's document. A build is a person waiting for a docx
// with no network at all, and a review session is somebody's document open in
// front of them; a background fetch in either is a second thing happening that
// nobody asked for, and on a slow connection it is the command taking longer
// for a reason the person cannot see. The rule is held by reading this package
// rather than by trusting it: TestNothingChecksForUpdatesUnasked says one
// entry in the table names cmdUpdate, and that update.go and notice.go are the
// only two files here that reach internal/update at all.
// TestReadNeverReachesTheCheck runs a read with a month-old stamp and a reach
// that fails the test if it is called, and TestNoSkillRunsUpdateOnItsOwn says
// no skill runs the updater either. A skill may say there is a newer gdoc,
// because saying is not running.
//
// The check judges nothing. The object under `update` carries `installed`,
// `latest_stable`, `latest_nightly`, `checked_at` and `error`, and the one
// line for a person goes to stderr where the help prose already goes, so
// stdout is still exactly one JSON object.
//
// What either of them may reach is the fifth grant, AllowUpdateFrom, and
// nothing else: the releases listing of one repository, the download under it,
// and the asset host the download redirects to, all GET, all without a
// credential. TestTheUpdateRunReachesTheReleasesAndNothingElse judges the
// policy the command opens and TestTheHelpCheckOpensThePolicyTheUpdateOpens
// judges the policy the check opens, so a document id cannot come along for
// the ride on either.
//
// GitHub not answering is an answer. A listing that times out, refuses the
// connection, answers 5xx or answers the rate limit is ok: true with
// action: unreachable and a warning naming the cause, because the person asked
// a question and the honest answer is that today gdoc cannot say. Nothing on
// disk is touched: TestAnUnreachableGitHubIsAnAnswerAndNotAFailure.
//
// The flags name one run each. --rollback is a file move on this machine and
// --check, --major and --nightly are about which release to fetch, so typing
// one of each is refused naming both: TestUpdateRefusesWhatMeansTwoThings. The
// rest of the policy is internal/update's, and its table is a test over
// literals there rather than a test over a wire here.
//
// Two refusals belong to this layer. A symlinked binary is a checkout install,
// and replacing it would drop a release over the link and leave `make build`
// writing to a file nobody runs: TestASymlinkedBinaryIsRefusedByName. And a
// run that fails part way carries no action at all, because every word in that
// field is something that finished, and "updated" beside ok: false would be
// the object contradicting itself:
// TestAZipThatDoesNotMatchTheChecksumReplacesNothing.
//
// The object says two things about the file it left behind. sha256 is the hash
// of what is at the path now, after an update and after a rollback alike.
// verified says that hash came out of bytes the release published a checksum
// for, which a rollback cannot say about a binary it only put back:
// TestTheUpdateObjectIsWhatTheSkillsRead and
// TestRollbackPutsTheEarlierBinaryBack.
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
// The last of those is why parseArgsN looks the next argument up in the
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
// # annotate takes no folder and no note
//
// The writers before it each carry something annotate does not, and in both
// cases what is missing is a question this command cannot ask wrongly.
//
// No folder, so no probe. propose creates a throwaway document every run to
// find out whether Docs honours SUGGEST today, because a SUGGEST that is
// quietly ignored turns a proposal into a direct edit of somebody's prose. The
// batch annotate sends holds one insertComment and nothing else, and no
// insertComment can move a character whatever the write mode does. So the probe
// has no question to answer here, and running it would litter a folder asking
// it. PRINCIPLES.md says the probe runs every time, and this is the second
// writer it does not run for, restyle's phase 1 being the first: that file's
// 2026-09-18 amendment note names both, docs/v2/DECISIONS.md holds the bend
// under the same date, and internal/annotate's Batch holds the shape the
// reasoning rests on.
//
// No note, so no provenance. The note exists so withdraw can recognise gdoc's
// own pending suggestions later, and a comment is not a suggestion: it is in
// the thread, signed with the robot, and a person deletes it in the browser in
// one gesture. Recording it would be provenance for a permission nothing uses.
//
// The two input forms are two calls, and a run naming both is refused before a
// session opens, by the flag it was given rather than by the flag it was
// missing: TestAnnotateRefusesFromBesideQuote,
// TestAnnotateRefusesFromBesideBodyFile and
// TestAnnotateNeedsAQuoteWithItsBodyFile. Every entry is checked before the
// first one is sent, so a file whose second comment is malformed writes neither:
// TestAnnotateRefusesASecondBadEntryBeforeAnyRequest hands in that file and says
// the wire saw nothing, and
// TestAnnotateRefusesARobotInTheWhyBeforeAnyRequest and
// TestAnnotateRefusesMarkdownBeforeAnyRequest are the two shapes it refuses.
// TestAnnotatePlacesEachEntryAndVerifiesIt is the whole run, and
// TestAnnotateStopsAtTheFirstEntryThatCannotBeSent is the report keeping one
// entry per annotation when it stops in the middle.
//
// # The URL picks the entry, and a note gdoc copied is refused
//
// A note names every document it has been published to, under documents: in
// its gdoc: block, and the URL is what says which entry a run acts on. paired
// in paired.go is that lookup, and every command taking --md goes through it,
// so suggestions, propose and withdraw refuse the same things in the same
// words. annotate takes no --md: it leaves a comment on the words a caller
// quotes and records nothing.
//
// Three refusals. A note whose front matter does not read is frontmatter's own,
// passed through. A note that does not name this document is the wrong file,
// and the refusal names every document it does name, because the answer is
// always to open one of those. A note whose entry carries exported.note is a
// copy gdoc wrote beside somebody else's document rather than the source of
// one: there is nothing in it to record, and a snapshot written into it would
// be read next time as that document's own history.
//
// TestEveryWriterActsOnTheURLAndRefusesAnIDOutsideTheList,
// TestAProposalIsRecordedUnderItsOwnDocument,
// TestWithdrawNeverSendsAnIDFromAnotherDocument,
// TestGoneSinceReadsTheSnapshotOfTheDocumentRead and
// TestEveryWriterRefusesACopyThatNamesANote are the pins.
//
// A note published before 2026-09-19 carries schema 1, one document beside the
// schema. It still reads, and the first run that writes to it rewrites the
// block as a list. That is a change to somebody's file, so the envelope says so
// once, and the next run has nothing to say because the block is already a
// list. TestTheReplySaysOnceWhenTheBlockWasRewritten and
// TestAReadThatWritesNothingNeverSaysTheBlockWasRewritten are the pins.
//
// publish is the writer that has no URL to pick with, because it is the run
// that makes the document. It appends: a note that already names a document is
// published again, into a new one, and the entries already there are carried
// through untouched. Those are other documents' history and this run knows
// nothing about them. It refuses one note before anything leaves the machine,
// the copy carrying exported.note, because publish.publishable has to ask that
// over every entry where paired asks it over one.
// TestPublishAppendsAnEntryToAPairedNote and
// TestPublishRefusesACopyThatNamesANote are the pins.
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
// one whose gdoc: block has gone, and one that no longer names this document.
// Each is a warning carrying the reason, and nothing is written into the note.
// The refusal opens by saying the note changed while the run was under way,
// because that is what tells a reader whether to run the command again or to go
// and look at the file.
//
// The block is re-parsed from the fresh bytes too, so a proposal another run
// recorded in that window survives: writing the block this run read into bytes
// it did not would keep the author's prose and still drop that entry. notePath
// therefore keeps the path, the document the run is of and the entry the
// pairing check read, and never the bytes it read them from, because a copy
// held there would only be the stale bytes somebody later wrote back.
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
// pair's, in publish.go beside this file: there an entry for the document this
// run just made appearing during the upload is what is refused, because a
// second one would name it twice, and the refusal is a rollback that
// internal/publish only carries out.
package main
