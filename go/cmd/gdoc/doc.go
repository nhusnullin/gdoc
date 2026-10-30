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
// Every command but one. mcp is a protocol session rather than an answer, and
// route in main.go sends it to serveMCP before run is reached, so no envelope
// is printed around it: TestMcpIsRoutedBeforeRun. What reaches stdout there is
// one JSON-RPC message per line and nothing else, which is the same promise in
// the shape a client can read: TestStdoutCarriesOnlyJSONRPC. The protocol's own
// rules are in internal/mcp/doc.go, and what chat adds to a command is in
// internal/chat/doc.go.
//
// And one mode that prints more than one object, held to one command by the
// shape of the code. internal/emit's Lines writes one JSON object per line,
// and `comments --follow`, the poll held open, is what it is for: a window
// found after four minutes is news now, and a caller that had to wait for the
// envelope to read it could not act on it until the watch ended. The last
// line is the envelope, and the exit code agrees with it as always:
// TestFollowWritesOneLinePerWindowAndTheEnvelopeLast and
// TestFollowsExitCodeFollowsTheLastLine.
//
// There is one such writer in a run, run builds it from stdout, and it
// travels down safeDispatch and dispatch onto the command's own args. So the
// route a call came in on decides whether the run can print lines at all: the
// chat route passes nil, because stdout there carries JSON-RPC, and a follow
// asked for with no writer is refused by name before the token is read.
// TestOnlyFollowWritesLines holds the constructor to main.go and the write
// method to the two files that hold the follow, TestTheMcpRouteHandsNoLineWriter
// holds the nil, and TestAFollowWithNoLineWriterIsRefused holds the refusal.
//
// And one narrowing. A help screen on a terminal without --json writes nothing
// to stdout, because the reader of that screen is a person and the object under
// it is 7.5 KB they did not ask for. Every other result keeps its object,
// refusals and the panic envelope included, and a pipe gets the object in every
// case. The rule and its tests are in "On a terminal a help screen replaces the
// object" below.
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
//   - auth logout: the token file taken away, internal/auth. One file on this
//     computer and no request: the grant stays at Google until the person
//     takes it away in their account settings, which is what the command's own
//     sentence says.
//   - read <url> [--structure]: the text projection, or the tree with the flag.
//     internal/docs reads, internal/view projects.
//   - comments <url> [--since] [--wait] [--follow] [--witness]: the threads
//     with their ranges, markers, replies and the next cursor, in one answer
//     or one line per window. internal/comments, with internal/docx for the
//     witness.
//   - suggestions <url> [--md]: what is pending, and with a paired note what
//     stopped being pending. internal/suggestions, internal/frontmatter.
//   - export <url> --out: the document in the hub as Markdown, one file per
//     tab, with its pictures beside it. internal/export, over the same two
//     reads read and comments --witness make.
//   - restyle <url> --dry-run | --from [--fields]: the survey, the house style
//     in place, and the proposed prelude. internal/restyle, internal/prelude.
//   - probe --folder: whether Docs honours SUGGEST today, asked by hand.
//     internal/probe.
//   - reply <url> <comment id> --body | --body-file: one reply under the mark
//     in effect, its words on the line or in a file. internal/reply.
//   - propose <url> --from [--md] [--folder]: a change as a suggestion, either
//     words inside one paragraph or a block of new paragraphs.
//     internal/propose.
//   - withdraw <url> <suggestion id> --md: gdoc taking back its own proposal.
//     internal/withdraw.
//   - annotate <url> --quote | --from [--body-file]: a comment on the words a
//     caller quotes, and nothing else. internal/annotate.
//   - build --md --out [--house] [--force]: the house-style docx, no network
//     beyond the house logo, once a day and bounded by logoCeiling, twenty
//     seconds, because a person is waiting for a file and a cached mark is
//     already in hand. internal/house, internal/cover, internal/body,
//     internal/render, and internal/asset when the style names the logo as one
//     file in Drive rather than carrying its bytes.
//     TestTheLogoCeilingIsTwentySeconds and
//     TestAHungDriveCostsTheCeilingAndThenTheCacheAnswers.
//   - publish --md --folder-id [--house]: that docx into Drive as a Google
//     Doc. internal/publish.
//   - update [--check] [--major] [--nightly] [--rollback]: this binary
//     replaced by a newer release of it. internal/update, and the one reach
//     that carries no credential, gapi.Plain. A run that installed also
//     replaces gdoc's own completion file beside the token with the loader,
//     and leaves any other file at that path alone.
//   - mcp: this binary as a stdio server for Claude
//     Desktop, one JSON-RPC message per line. internal/mcp, with mcp.go for
//     the wiring. The one command route sends past run. It offers eight tools,
//     the six of the table that chat reviews with and guide and login, and
//     every one of the six takes the code guide hands out and refuses the call
//     without it: TestToolsListListsExactlyTheEightToolsWithTheirHints and
//     TestEveryToolButGuideAndLoginRefusesAMissingOrStaleCode. A write tool
//     names its target twice over, by id and by the document's own title, and by
//     the words the thread opens with where it writes into a thread: the title
//     is read off the document on the call itself and a call naming another
//     document is refused with nothing sent, which is TestAWrongTitleIsRefused
//     and TestAThreadQuoteDifferingOnlyInQuotesOrSpacingPasses. One item per
//     write call, and no field in it the card does not draw:
//     TestASecondItemIsRefused and TestAssigneeIsRefused. Every chat write is
//     then judged by internal/chat's hold rules before anything is sent, and a
//     held write reaches no wire: the answer is ok false, sent false, and the
//     hold's id, rule, value and words, with one fixed sentence that says to
//     tell the person and stop. The hold stays in that process for 30 minutes:
//     TestAHeldWriteSendsNothing, TestTheHeldAnswerNamesTheRuleTheValueAndTheText
//     and TestAHoldLivesThirtyMinutes. What sends a held write is the one tool
//     that hold registers, confirm_ and its id, which the client draws as a card
//     and the person approves: it goes when the write goes or when the thirty
//     minutes run out, its card carries the hold, the title, the reason and the
//     text in that order, a byte of difference in any of them is refused with the
//     hold kept, a release with less than five seconds of quiet behind it is
//     refused the same way, and what goes out is the call the hold kept rather
//     than anything the release sends:
//     TestAHoldRegistersOneConfirmToolAndRemovesItOnRelease,
//     TestTheConfirmSchemaListsHoldTitleReasonText,
//     TestAByteDifferentTitleReasonOrTextIsRefused,
//     TestAReleaseInsideTheQuietGapIsRefusedAndTheHoldKept and
//     TestAReleasedHoldPostsExactlyTheHeldTextOnce. The line takes the two
//     setting flags and nothing else: they are off it before mcp is routed, the
//     session resolves them once and every tool call of it goes out under the
//     same value, the extension passes neither, and whatever is left is refused
//     on stderr with exit 1:
//     TestMcpTakesTheSettingFlagsAndNothingElse and
//     TestTheMcpSessionHandsItsSettingsToEveryCall. The
//     review core it serves is a committed copy of the skill's, held equal by
//     TestTheEmbeddedCoreIsTheSkillsCore.
//   - help [<command>|marks]: the table itself, as an object and as words.
//     help.go. marks is the one word it answers that names no command, and it
//     has its own file, helpmarks.go.
//   - completion <shell> --out [--force] [--loader]: the table as a shell
//     script, written to a file. completion.go, with the template beside it.
//     --loader writes a loader instead, a few lines of zsh that ask the gdoc on
//     PATH for its completion at every new shell, so a Tab follows the binary
//     that is installed rather than the one that wrote the file.
//
// The usage line names every command that exists, because it is joined from
// the table. Three tests hold the table and the usage line together.
// TestTheUsageLineNamesEveryCommand spells the nineteen out word for word, as a
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
// spells the fifteen lines out as a reader sees them, TestTheObjectSaysHowEachFlagStands
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
// Two columns of the table are not about the call at all. report says a run
// of this command reaches the presence record, and readsBack says the command
// reads its own write back. Both are read by dispatch and by nothing else:
// dispatch calls presence.Begin after the parse and defers its end, so a
// command that is added is recorded without its author writing a line of
// recording, and a panic safeDispatch recovers still closes the run it opened.
// report is on for the thirteen that read or write a document, replace this
// binary, make a browser trip, or take the credential away, and off for
// `auth status`, probe, build, help, completion and mcp, because a question
// about the token, a page of help and one local file are not a run anybody is
// waiting on, and mcp is the session every other event is reported under:
// TestAReportingCommandWritesAStartAndAnEnd, TestAnAuthStatusRunWritesNoEvent,
// TestAHelpRunWritesNoEvent, TestAnUpdateAndALoginWriteNoDoc and
// TestLogoutWritesAStartAndAnEndWithNoDoc. readsBack is
// on for the five writers that confirm their own write, reply, propose,
// withdraw, annotate and publish, and it is the only thing that lets a
// verified reach the record, because update and restyle print verified: false
// to mean no write happened and that must never be kept as a failed read-back:
// TestOnlyTheReadBackWritersCarryVerified.
//
// The table is a function and not a variable, and that is Go and not taste.
// help is an entry in it and reads it, so a variable would refer to itself
// through a function, which is an initialization cycle the compiler refuses.
// Each caller is handed its own slice, so nothing keeps a pointer into a table
// somebody else is reading.
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
// the pins, and TestHelpTakesWordsAndOnlyItsTwoFlags holds that help itself is
// parsed as strictly as everything else: it takes --json and refuses every
// other flag by name.
//
// The binary never prompts, and the one command that reads stdin reads a
// protocol. Every command takes its facts as arguments and answers, so nothing
// waits on a pipe nobody filled. mcp reads stdin because a JSON-RPC session is
// a stream, and it still asks no question: what it reads is messages from a
// client, never words from a person. One room names the real streams, and
// hands every other room a reader and a writer, so internal/mcp runs a whole
// session against strings in memory. TestOnlyMainNamesStdinAndStdout in
// go/boundary is the pin, and it reads the syntax tree, so this paragraph is
// prose rather than a second room.
//
// # On a terminal a help screen replaces the object
//
// A person who types gdoc help reads the help and then, today, 7.5 KB of JSON
// wrapped under it. So on a terminal without --json a help screen is the whole
// answer: the words go to stderr, stdout gets nothing, and the exit code is the
// one the object would have carried. Two results have such a screen, help when
// it answers and bare gdoc, and no third.
//
// The decision is run's, because run is the room that holds stdout. A result
// carries emit.Result.Screen, a marker with the tag json:"-" that no object can
// show: TestScreenNeverReachesTheObject in internal/emit. A successful cmdHelp
// sets it and the bare branch of dispatch sets it, and nothing else does, so a
// panic has no marker and its envelope is printed wherever stdout goes.
// TestHelpOnATerminalWritesNothingToStdout,
// TestBareGdocOnATerminalWritesNothingToStdoutAndExitsOne,
// TestARefusalKeepsItsObjectOnATerminal over gdoc help sing,
// TestAPanicKeepsItsEnvelopeOnATerminal and
// TestEveryOtherCommandKeepsItsObjectOnATerminal are the pins.
//
// --json is how a program asks for the object anyway. It prints the object
// wherever stdout goes and leaves stderr plain, so a skill driving a terminal
// reads exactly the bytes it read before. It survives every spelling, because
// helpWords strips it before the table is matched and hands cmdHelp the answer:
// TestEverySpellingOfHelpKeepsJSON and
// TestHelpWithJSONPrintsTheObjectOnATerminal. gdoc --json alone is still an
// unknown command, because a flag names no command: TestJSONAloneIsStillUnknown.
// Any other flag on help is refused by name:
// TestHelpTakesWordsAndOnlyItsTwoFlags.
//
// --brief is help's second flag, and the setup call of every skill. Under it
// the object carries settings and update and no table, which is the two facts
// a skill reads, in the same fields, for a few hundred bytes instead of 7.5 KB:
// TestHelpBriefCarriesTheMarkAndTheNotice and TestHelpBriefIsAFewHundredBytes.
// It answers about the tool, so a command word beside it is refused by name:
// TestHelpBriefTakesNoCommandWord. It is never a screen, because a caller
// asked for it, so its object prints on a terminal with the notice head alone
// on stderr: TestHelpBriefPrintsItsObjectOnATerminal. It is a flag of help and
// of nothing else, so gdoc --help --brief is still an unknown command:
// TestHelpBriefIsAFlagOfHelpAndOfNothingElse.
//
// Bare `help --json` names one more field, logo_path: where internal/asset
// keeps the house logo on this machine. It is there for the menu bar app,
// whose one help argv that is, so the app draws the firm's mark without
// holding its own copy of a Drive id. The help names the path and reads
// nothing for it, and no other form of the question carries it, so no skill's
// context and no frozen screen holds a home path:
// TestHelpNamesWhereTheLogoIsCached. The same form names logo_url, the
// picture's own page in Drive built from the house id, so the app opens it on
// a press without holding the id: TestHelpNamesTheLogosPageInDrive.
//
// The hint is the last line of a screen that dropped its object, dim, and it
// says to add --json. It is written by run, never where the object was printed,
// and never on bare gdoc, which failed: that person was reaching for a command
// and the refusal already says what to type. The ok field is what parts the
// two: TestTheHintIsTheLastLineOnlyWhenTheObjectWasDropped.
//
// What the screens hold is helpscreen.go's, and the command table is still the
// only description of a command. gdoc help draws the version box with the usage
// line, the release notice and any warning the dropped object would have
// carried, then the commands grouped by the four jobs a command does, then the
// line saying where the detail is. The group is a field on the table entry and
// reaches no object: TestEveryCommandHasAGroup, TestTheGroupIsNotInTheObject and
// TestEveryCommandIsOnTheGroupedScreen, with
// TestAWarningIsDrawnWhereTheObjectWouldHaveCarriedIt over the warning rows.
// gdoc help <command> draws that command's box, its flags as two columns, and
// its example on one line under the box:
// TestTheExampleIsOneLineUnderTheBox. Bare gdoc opens with gdoc needs a
// command., the words its object carries:
// TestBareGdocOpensWithTheWordsItRefusesWith. The three screens are recorded at
// four widths and in colour by TestTheHelpScreensAreTheirRecordedBytes, no line
// leaves its box in TestNoLineOfAScreenIsWiderThanItsBox, and NO_COLOR or
// TERM=dumb keeps the boxes and drops every escape byte:
// TestNoColourOnATerminalDrawsTheBoxesWithNoEscapeByte.
//
// # A note on an entry is what belongs to no flag
//
// A command whose note carries syntax of its own has something to say that no
// flag can hold. build has three lines, the footnote, the seven marks and the
// page break, and no other entry has any: what a footnote is, what each mark
// is written as, which refusals a note runs into, and that `---` on a line of
// its own turns the page (DECISIONS.md, 2026-10-16). They are the notes field of the table
// entry, printed under the summary in today's text and in a Notes rule on the
// screen, and they reach the object too, unlike the group: a note is a fact
// about what the command carries off a note's own lines, and a skill writing
// one is the other reader of this help. The field is absent where there is
// none, because "takes none" and "said nothing" are one answer for a line of
// prose, which they are not for the words and flags beside it.
// TestHelpBuildNamesFootnotesAndMarks is the pin, with
// TestHelpBuildSaysAThematicBreakIsAPageBreak over the third line, and the
// seven colours themselves are the topic below rather than a second copy here.
//
// # A panic is still one envelope
//
// safeDispatch recovers, prints ok: false with what happened, and puts the
// stack trace on stderr. Without it a crash prints a Go trace, nothing at all
// on stdout, and exits 2, which is the one shape no caller can read.
// TestAPanicIsStillOneEnvelope is the pin.
//
// # Three setting flags, taken off the line first, one of them experimental
//
// --config, --mark and --presence belong to no command, because they belong
// to all of them. They are lifted off the line in route, before the word mcp
// is looked for and before --help is, so one reading of them serves every
// command, the session and the help alike. What is left is the line as every
// parser already read it, and nothing below route sees any of the three:
// TestEveryCommandTakesTheSettingFlags and
// TestTheSettingFlagsAreTakenAnywhereOnTheLine.
//
// Wherever they stand, and in both spellings. `gdoc read <url> --mark ∴` and
// `gdoc --mark=∴ read <url>` are the same run, because a person does not have
// to know where in a line a flag of the whole binary goes. `--mark ""` and
// `--mark=` are the empty mark somebody asked for, which writes gdoc's words
// with nothing in front of them: TestAnEmptyMarkArgumentIsTheEmptyMark.
//
// The refusals are the parser's own, for the same reasons: a flag given twice,
// a flag with nothing after it, a flag whose value is the other setting flag,
// and a --config with an empty value each fail naming the offender.
// TestASettingFlagGivenTwiceIsRefused and
// TestASettingFlagWithNoValueIsRefused.
//
// internal/settings decides what they mean: the argument over the file over
// the built-in default, what a mark may be, and what the file may hold. What
// is decided here is the shape of the refusal, which is the shape of whatever
// the line was going to be. A command gets one envelope on stdout and exit 1,
// like every other refusal, and `gdoc mcp` gets one line on stderr and exit 1,
// because a session prints no envelope and a stray object would be a line in
// the middle of somebody's protocol:
// TestABadSettingsFileIsOneEnvelope,
// TestAConfigFileThatDoesNotExistIsRefusedOnTheLine and
// TestMcpTakesTheSettingFlagsAndNothingElse.
//
// What the run resolved to goes down on every command's args, and a session is
// handed it once at its start, so every tool call of one chat writes the same
// mark: TestTheMcpSessionHandsItsSettingsToEveryCall.
//
// The commands that write into a thread read it off those args and hand it to
// the writer: propose and annotate pass it to the shape check, which refuses a
// reason that already carries a mark, and to the write, which puts it in front
// of the reason. Nothing is remembered between runs, so the mark a comment
// carries is the mark the run that wrote it was given.
//
// They are in no usage line and in no entry of the help object, because they
// are not one command's flags. They are in the help a person reads, under
// their own heading in each command's page and once in the grouped table, and
// both completion scripts offer them after any command:
// TestEverySettingFlagIsInHelpAndInBothScripts. The --mark sentence says what
// the empty mark costs, because turning the mark off turns off the only record
// of authorship a thread has: TestTheMarkFlagSaysWhatAnEmptyMarkCosts.
//
// --presence is the third, and the one the table marks experimental. It takes
// true or false and it is off unless somebody says otherwise, and what it
// turns on is the record internal/presence appends for the menu bar app: a
// start and an end for every command whose table entry reports, and a line
// for each way a chat hold begins and leaves. It is a setting rather than a
// command of its own because a person turns it on once for their machine and
// forgets it, and an account check would be a wire call for a question about
// this computer. An experimental flag is in the table with its sentence and
// in no screen and no script, so one person trying it out changes no word of
// the help every colleague reads: the fifteen screens and both completion
// scripts are the same bytes with it on the line as without it, which
// TestAnExperimentalFlagIsInTheTableAndInNoScreen measures. It is not hidden.
// `gdoc help --json` carries presence and presence_from where it is on, which
// is the one place a caller already using it looks, and both fields are
// absent where it is off, so no skill reading the object learns a word about
// a flag that is not for it. No skill passes it either, and go/boundary's
// TestNoSkillPassesThePresenceFlag is where that is held.
//
// What they resolved to is one field of the help object, settings, which
// carries the mark in effect and which of the three sources it came from. A
// colleague who is not sure which mark this run writes asks the binary rather
// than reading a file and a line. The field is always there, and the empty
// mark is printed as the empty string, because a skill that read an absent
// field as "nothing was set" would write the default mark into a thread
// somebody asked to keep clean:
// TestHelpCarriesTheSettingsInEffectAndWhereEachCameFrom,
// TestHelpCarriesTheEmptyMarkAsAnEmptyString and
// TestHelpForOneCommandCarriesTheSettingsToo.
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
package main
