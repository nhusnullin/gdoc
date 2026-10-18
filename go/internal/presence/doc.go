// Package presence is the channel gdoc reports what it is doing on: one
// append-only JSONL file per month in the config directory, one line for a
// command's start and one for its end, one line for each part of the one
// command that has parts, and one line for each way a chat hold begins and
// leaves. Those are every kind there is, and the list of fields each may
// carry is one struct: TestTheEventCarriesOnlyItsListedFields.
//
// A hold begins as held and leaves as released, expired, landed or dropped,
// and a hold leaves at most once: cmd/gdoc's
// TestEveryWayAHoldLeavesIsExactlyOneLine. dropped is the sign-in the hold was
// raised under no longer being the sign-in on the computer, which cmd/gdoc
// judges by comparing a hash of the refresh token it kept beside the hold. The
// hash never reaches this package and no account ever does, so the line says
// the card is gone and nothing about whose it was:
// cmd/gdoc's TestTheSignInMarkReachesNoAnswerNoToolAndNoLine.
//
// A publish is the one command with parts, and between its start and its end
// it writes one step line for each of them: build, upload, read_back, link,
// each written immediately before that part begins, with the part's place in
// that fixed list and the length of the list beside its name. The names are
// this package's, so a caller that drifts from the four writes no line at all:
// TestAStepNameThatIsNotOneOfTheFourWritesNothing, with cmd/gdoc's
// TestAPublishStepsAreTheFixedNamesInOrder over a real run and go/boundary's
// TestStepHasOnlyPublishAsCaller over the one room that calls it.
//
// The title reaches a step as soon as the render knows it, which is why the
// build step carries none: cmd/gdoc's TestTheFirstStepsEndCarriesTheTitle.
// Nothing else about a part is written, and there is nowhere to write it: no
// percentage and no estimate, because what a part of a publish costs is not
// something gdoc knows before it runs it, and the two numbers a step carries
// are places in a list of four:
// TestNoPercentageOrEstimateIsEverWritten.
//
// It exists because of the 2026-10-17 brief: gdoc must not change in kind,
// and it may say what it is doing in real time. A live review is hours of
// silence in a chat window, and alive and dead look the same there. The
// presence app reads this file and answers "is it alive" and "is it waiting
// on me" in a menu bar glyph. docs/v2/DECISIONS.md holds the decision, and
// docs/plans/completed/2026-10-17-gdoc-v2-m23-the-presence-spec.md the rules
// this package implements.
//
// # The binary never reads it
//
// This package exports no reader. No Read or Open function, no os.Open and no
// os.ReadFile outside the tests, and no decision gdoc makes depends on a line
// in the file: not a hold rule, not a receipt, not a cursor.
// go/boundary's TestPresenceExportsNoReader holds that, and
// TestOnlyPresenceAppends holds the other half, that O_APPEND appears in no
// other room.
//
// That is the whole safety of the design. The record is a best-effort set of
// observations and never authoritative state, so a file somebody edited, a
// file half written by a disk that filled, or no file at all changes nothing
// about what the next command does. A channel the binary read back would be
// a second source of truth for holds and cursors, and the one place a hold
// lives is still the process that is holding it.
//
// # Off means today's bytes
//
// The flag is the setting presence: true and the argument --presence true.
// With it off nothing here touches the disk: no directory is made, no old
// file is removed, and nothing is written. Begin returns a nil *Run, every
// method on which is a no-op, so the hook in dispatch is the same two lines
// either way: TestOffTouchesNothing and TestOffRemovesNothing, with
// cmd/gdoc's TestTheFlagChangesNoByteOfTheObject over the envelope itself.
//
// # The event
//
// A fixed allowlisted struct at schema v: 1, never the envelope.
// TestTheEventCarriesOnlyItsListedFields compares the json tags with the
// specification's list, in its order.
//
// What is never in it: document text, comment or reply words, quotes,
// proposal text, a hold's value, reason or text, error or warning sentences,
// any path, the account, anything about the token, argv, the cursor, the mark,
// and no judging word. TestAnEventCarriesNoWordOfTheEnvelopeButTheTitle puts
// a canary in each place a result has, the text, the threads, the path, the
// account, a warning and the error, and reads the line back. A hold's value,
// its reason and its words are held by Held's signature, which takes none of
// them, and cmd/gdoc's TestAHeldWriteRecordsTheRuleAndNoneOfItsWords looks for
// them in the line a real held write wrote.
//
// via is who ran the command, in three words: terminal is a person typing,
// chat is a tool call of gdoc mcp, and code is the binary run by Claude Code's
// own tool. The first two are a mark on the context, and the third is one
// environment variable cmd/gdoc reads once, in dispatch, because this package
// is a file and a clock and reads no environment of its own. chat wins over
// code, because a tool call of a chat session is a chat call whatever started
// the process: TestChatWinsOverCode, with cmd/gdoc's
// TestViaIsCodeUnderClaudeCode and TestViaChatWinsOverCodeInTheMcpSession.
//
// wait and interrupted are the two halves of one wait: the start says the run
// is going to watch a document, and the end says a person stopped it. The
// second is lifted out of the waited object `gdoc comments --wait` prints,
// which is the only place in the binary that knows a wait ended early, and a
// wait nobody stopped writes nothing at all rather than a false:
// TestAnInterruptedWaitSaysSo, with cmd/gdoc's TestAStoppedWaitsEndSaysSo over
// a real wait somebody stopped.
//
// The one document word is the title, capped at 200 runes, because without it
// the app would run gdoc read to fill a row and pull a whole body through a
// second process. It is lifted from the top level of the envelope's data only
// where it is there as a string: TestTitleIsLiftedOnlyWhenPresentAsAString
// and TestTheTitleIsCappedAtTwoHundredRunes.
//
// made is the one other id a line may carry: the document a run created,
// which only a publish does. Every other command was handed its document and
// the doc field names it, so made is never a second name for the same thing.
// cmd/gdoc names the command and puts the id through the resolver a line's
// first word goes through, so a path or a sentence under that key reaches no
// file: cmd/gdoc's TestAPublishEndNamesTheDocumentItCreated,
// TestARolledBackPublishNamesNoDocument and TestOnlyPublishCarriesMade.
//
// stop is why a failed run failed, in one of emit's seven fixed words, with
// line beside it where the cause sits on a line of a note, which today is the
// one picture the house style cannot carry. Both are set by cmd/gdoc, which
// reads the typed failures the packages raise and never a sentence, and both
// travel on emit.Result in fields the printed object does not carry, the way
// Screen does: cmd/gdoc's stop.go holds the reading,
// TestAStopIsOneOfTheSevenCodesAndNeverASentence, TestTheSVGRefusalCarriesItsLine
// and TestAnUnknownFailureIsOther its rules.
//
// This package judges none of it. It writes a word that is one of the seven
// and nothing otherwise, because a reader of the record could not tell a word
// gdoc meant from a sentence that leaked out of English; it writes no stop
// beside a run that worked, and no line without a code:
// TestAStopIsOneOfTheSevenCodesAndNeverASentence and
// TestAStopReachesNoSuccessfulEnd. A failure gdoc did not sort is other, which
// cmd/gdoc fills in where the command named no word, so a failed end always
// says something and never a guess.
//
// ok is the envelope's ok, and a start says nothing about it:
// TestOkIsTheEnvelopesOk. verified is lifted only for the commands whose
// caller says they read their own write back, because update and restyle
// print verified: false to mean no write happened, and that must never become
// a failed read-back in a record meant to last:
// TestVerifiedIsLiftedOnlyWhenTheCallerSaysTheCommandReadsBack.
//
// ms is monotonic elapsed time, taken from a source that is not the clock the
// at is written from, so a machine whose wall clock moves writes no negative
// duration: TestTheDurationIsMonotonic.
//
// pid with run, eight random bytes taken once per process, tells one process
// from another; call pairs a start with its end inside one. The app asks the
// kernel whether a pid is alive, and a start with no end whose pid is dead is
// the one thing it may call an unknown outcome.
//
// # The file, and the one write
//
// presence/<yyyy>-<mm>.jsonl in the config directory, the directory 0700 and
// the file 0600: TestTheFileIsReadableByItsOwnerAlone. The month is the
// event's UTC month, so nothing is ever renamed and two machines agree which
// file a moment belongs in: TestTheMonthNamesTheFile.
//
// Each event is one open with O_WRONLY|O_APPEND|O_CREATE|O_NOFOLLOW, one
// write of one line checked for a short write, and one close. The append is
// what lets two processes, a terminal run and a chat session, write the same
// file with no lock and no torn line: TestTwoWritersNeverTearALine runs 16
// writers on separate opens. A short write is a failure rather than a silence,
// because the half line on disk is what a reader would have had to guess at:
// TestAShortWriteIsAFailure.
//
// A symlink or anything else that is not a regular file at the path is
// refused by name and nothing is written, so a link somebody planted carries
// no record anywhere else:
// TestASymlinkOrANonRegularFileAtThePathIsRefusedAndNothingIsWritten.
//
// Local disk only. A config directory on a network or a synced volume is
// unsupported, because the atomicity of an append under PIPE_BUF is the local
// filesystem's promise and nobody else's.
//
// One other file lives in that directory and is none of this package's
// business: presence/app.lock, which the menu bar app holds flock on for its
// whole life so that cmd/gdoc can ask whether it is running before opening
// it. Nothing here opens it, writes it or removes it: the retention sweep
// takes only names that are exactly a month's, which
// TestOnlyTheCurrentAndPreviousMonthSurviveAndOnlyExactNamesAreRemoved holds,
// and the lock is asked for by cmd/gdoc's openPresenceApp, which creates
// nothing and reads no byte:
// cmd/gdoc's TestTheOpenerAsksThreeThingsAndRunsOpenGOnce and go/boundary's
// TestTheLockNameIsOneWordInTwoLanguages over the name Go and Swift share.
// Decided 2026-10-18, DECISIONS.md, "The app opens itself".
//
// # A failure changes nothing
//
// A write that fails changes no byte of the envelope and no exit code. One
// line goes to stderr, once per run, because a chat session that cannot write
// would otherwise say the same sentence on every call:
// TestAFailedWriteSaysSoOnceOnStderrAndReturnsNothing. A run is a process
// here, so Fresh, which cmd/gdoc calls where a run begins, changes nothing a
// person sees and keeps two runs in one process from inheriting the line:
// cmd/gdoc's TestAnUnwritableLogChangesNoByteOfTheObject. None of the functions
// here returns an error, which is the same decision said in the signatures.
//
// # Retention
//
// The first write of a process removes the months before the previous one,
// and only files whose name is exactly a month's: a note or a backup somebody
// left in the directory stays where it is.
// TestOnlyTheCurrentAndPreviousMonthSurviveAndOnlyExactNamesAreRemoved. With
// the flag off nothing is removed, so what an earlier run wrote waits for a
// person: TestOffRemovesNothing.
//
// # One fixture, two languages
//
// testdata/events-v1.jsonl is what this writer writes for a scripted run that
// covers every event kind and every optional field, pinned byte for byte by
// TestTheFixtureIsWhatTheWriterWrites with the clock, the pid, the run id and
// the version behind variables. The presence app's tests decode the same
// bytes, and go/boundary holds the two copies equal, so the schema cannot
// drift between Go and Swift.
package presence
