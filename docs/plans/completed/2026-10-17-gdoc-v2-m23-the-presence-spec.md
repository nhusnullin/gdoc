# gdoc v2 Milestone 23: the presence. The channel, the flag, the delivery, the first glance. Specification

2026-10-17. Nail's brief, given the same night and summarised in
`docs/design/2026-10-17-presence-mechanism.md`: gdoc itself must not change
in kind; it may report what it is doing in real time, and senior engineers
decide how. The presence is a separate application whose changes never touch
gdoc. `gdoc update` updates it too. It runs only on Nail's machine for now,
behind a settings flag an argument can also set. Main gdoc stays backward
compatible. Four seats designed the mechanism in parallel and the project
manager resolved their disagreements; Nail has not reviewed the result, and
the seven decisions he is most likely to reverse are listed in that file. The
decision entry is in `docs/v2/DECISIONS.md` under 2026-10-17, and the task
list is `docs/plans/completed/2026-10-17-gdoc-v2-m23-the-presence.md`. It
starts from `main` after M22 and M21 merged.

## Principles

Serves: 4, every word costs a reader's attention. Today a live review is
hours of silence in a chat window, and alive and dead look the same. One
glance at the menu bar answers "is it alive" and "is it waiting on me" without
a word in the chat: a hold waiting, a run in flight, a watch, or a run that
stopped with its outcome unknown; the failed read-back joins the glyph with
the green check in the next milestone. Serves 3 as well: the record
the binary writes is facts only, `verified: false` stays a fact, and nothing
in the design lets the app approve a held write.

Strains: 2, nothing lives hidden beside the hub. The binary writes a history
of its own runs into the config directory, one line per command start and
end. It is in the config directory and not the hub, like `update-check.json`
and `login-pending.json`, gdoc never reads it, and it carries no document word
but a title. Nail accepted that the config directory is the place when he
chose a menu bar over a thread. And 1, if it is not inside the binary it does
not travel: the release zip carries a second program. gdoc does not depend on
it, never starts it, and is whole without it, as it is whole without the
skills and the extension template the zip already carries.

## What was found before deciding

- Every command, terminal or chat, passes `dispatch` in `go/cmd/gdoc/main.go`,
  and every envelope passes `printResult`. A chat tool call reaches `dispatch`
  through `safeDispatch`. A held chat write never reaches it; holds live in
  `mcpHolds` and die with the process after `mcpHoldLife`, 30 minutes.
- A live watch is a series of `comments --wait` processes, so a start and an
  end per call shows "watching, last seen" with no event per poll and no
  change to `internal/comments`.
- `mcplog.go` already writes one fact line per tool call with no document word
  in it, pinned by three tests. It is the precedent for lifting named keys out
  of an envelope rather than writing the envelope.
- Settings are decoded strictly and a refusal stops every command. A binary
  older than a key refuses a file that carries it.
- `~/.local/bin/gdoc` on Nail's machine is a release binary with
  `gdoc.previous` beside it, so `gdoc update` is the real route there, and
  the CLAUDE.md sentence about a link to `bin/gdoc` is stale.
- Measured: a universal Swift bundle builds with `swift build` in 13 seconds
  with no Xcode project; a bundle a Go program unzips carries no quarantine
  attribute and keeps a valid ad-hoc signature; a `DispatchSource` tail
  follows appends, partial lines and a monthly rotation; the panel renders to
  PNG from `swift test` with no Screen Recording permission. The machine holds
  no code-signing identity; Developer ID stays with M17.
- `go/boundary/workflows_test.go` fails the suite unless `release.yml` has
  exactly one job, and its `Needs` field is a string.

## Decisions taken

Nail, 2026-10-17, in chat:

| Rule | Where |
|---|---|
| gdoc does not change in kind. It may report what it is doing in real time; the engineers choose the channel | cmd, presence |
| The presence is a separate application, so a change to it never touches gdoc | presence/, release |
| `gdoc update` updates the application too | update, cmd |
| A flag in settings, also settable by an argument, makes it available only to whoever sets it | settings, cmd |
| Every existing function of gdoc keeps working exactly as today | every test that exists |
| Plan, TDD, review, run board, release, as every milestone | docs/plans |

Taken in the design pass by the team and the project manager, each one a
sentence Nail can strike:

| Rule | Where |
|---|---|
| The channel is one append-only JSONL file per month, `presence/<yyyy>-<mm>.jsonl` in the config directory, directory 0700 and file 0600. Each event is one open with `O_WRONLY\|O_APPEND\|O_CREATE\|O_NOFOLLOW`, one write of one line checked for a short write, one close. A symlink or a non-regular file at the path is refused by name and nothing is written. The month is the event's UTC month, so nothing is renamed. Local disk only: a config directory on a network or synced volume is unsupported, and the guide says so | presence, config |
| The binary never reads the file. The package exports no reader, `O_APPEND` appears in no other room, and no decision gdoc makes depends on a line in it: not a hold rule, not a receipt, not a cursor. The file is a best-effort record of observations, never authoritative state | presence, boundary |
| The event is a fixed allowlisted struct, schema `v: 1`, never the envelope. Fields: `v`, `at`, `gdoc` (the binary's version, absent for a checkout build), `pid`, `run` (a random per-process id), `call`, `event`, `command`, `via`, `doc`, `wait`, `title`, `ok`, `verified`, `ms`, `hold`, `rule`, `until`. `ok` is the envelope's `ok`. `title` is optional and lifted from the top level of `data` only when it is there as a string. `verified` is lifted only for the five commands that read a write back, reply, propose, withdraw, annotate and publish, marked in the table; `update` and `restyle` print a `verified: false` that means no write happened, and that must never become a failed read-back in a record meant to last. `ms` is monotonic elapsed time. `doc` is the id the existing id parser gave, never a raw argument, and absent for `update` and `auth login`. Never: document text, comment or reply words, quotes, proposal text, a hold's value, reason or text, error or warning sentences, any path, the account, anything about the token, argv, the cursor, the mark, or a judging word | presence |
| `title` is the one document word, capped at 200 runes, because without it the app would run `gdoc read` to fill a row. The cap bounds the line and removes no sensitivity; that is accepted with Nail's vote for titles | presence |
| Events are `start` and `end` for the commands that name a document plus `update` and `auth login`; for a chat hold, `held`, then exactly one of `released` (consumed by a send, which says nothing about whether the send succeeded), `expired`, or `landed` (removed because another send landed). `help`, `completion`, `auth status`, `build`, `probe` and the mcp session itself write none. What happens in chat before `safeDispatch` (the cache, an argument refusal, the pin, the judge) writes nothing: the presence is command execution and holds, and the guide says so | cmd |
| The end is deferred in `dispatch`, so a recovered panic still writes an `end` with `ok: false`. A start with no end whose pid is dead means the outcome is unknown, and that is the only thing the app may say about it | cmd, presence/ |
| A write that fails changes no byte of the envelope and no exit code. One line goes to stderr, once per process. The write is synchronous on local disk, and that latency is accepted | presence, cmd |
| The binary keeps the current and the previous month's file and removes older ones when it writes, under the flag: regular files of the exact `<yyyy>-<mm>.jsonl` name only. With the flag off nothing is removed, so old files then stay until a person deletes them. This reverses product decision 8 and is Nail's to reverse back | presence |
| One fixture, `go/internal/presence/testdata/events-v1.jsonl`, written by the Go writer and pinned byte for byte by a Go test, and a copy under `presence/Tests` held equal by a Go test, so the schema cannot drift between two languages | presence, presence/ |
| The flag is the setting `presence: true` and the argument `--presence true`, a third setting flag beside `--mark` and `--config`, with the same refusals. The argument overrides the file. Chat gets it from the file only. It is a setting and never an identity check | settings, cmd |
| The flag is in the command table with its sentence and marked experimental: no help screen line, no completion script line, and `help --json` carries it only when it is on. The 15 help goldens, both completion scripts and the help object stay byte for byte | cmd |
| Off means today's bytes: no `presence/` directory is created, no envelope changes, `gdoc update` and `--rollback` are unchanged, the app is never uninstalled. Proven by byte-compare tests over the stubbed session with the flag off and on | cmd |
| No skill passes the flag | boundary |
| The app travels inside both darwin zips at `presence/GdocPresence.app/`, one universal bundle, under the one `SHA256SUMS`. No other zip carries it. The bundle carries no symlink and no resource but its plist, so an unzip and a rezip keep its signature valid, which the release asserts after the repack | release |
| `release.yml` gains a second job on a pinned macOS image that tests, builds, stamps, ad-hoc signs, verifies and uploads the bundle; the ubuntu job needs it, copies it into the stage in a `darwin-*)` arm, and a step before publish lists every zip and fails unless darwin zips carry the bundle and no other does. `release.yml` runs on tags only, so `go.yml` gains the same Swift job for every push. A broken Swift build fails the release, and a failed release is re-run from the Actions page; the nightly is unchanged. The ubuntu job has no `codesign`, so the macOS job proves the zip round trip itself: it zips the bundle the way the pack step does, unzips it and verifies the signature | release, boundary |
| With the flag on, `gdoc update` installs the bundle at `~/Applications/GdocPresence.app`, keeping the previous one as `GdocPresence.app.previous`. The zip is verified first; only entries under the bundle path are taken; absolute paths, `..`, symlinks, `__MACOSX`, `._*`, a duplicate or case-colliding path, more than 2,000 entries or 64 MiB expanded, another bundle identifier and a plist version that is not the release's are refused by name; a destination that is a symlink, a file, or a bundle with another identifier is refused and nothing moves; the bundle is staged under a unique name beside the destination, read back and summed, and swapped by rename after the binary's three renames. Every app failure is a warning that names where the bundles are and never undoes the binary, so the installed app and binary can differ in version, and the app shows both. Rollback swaps the binary, then the app if a previous exists, each swap its own involution. A run with nothing newer installs the app only when the installed binary is the newest candidate and the app is missing or at another version, and never under `--check` | update, cmd |
| gdoc never launches the app. The first launch is the person, and the result says `open ~/Applications/GdocPresence.app`. After an update the app reads its on-disk `Info.plist` and shows one row, "Updated to X. Quit and reopen"; it does not relaunch itself in this milestone, because the swap while running is unmeasured. `withExtension` stays the only caller of `runProgram`, pinned | cmd, presence/, boundary |
| `release/install.sh` ignores `presence/` in this milestone, pinned by a test. Non-darwin platforms get no app, and with the flag on the updater writes one warning naming the platform | release, cmd |
| The app is native Swift: an AppKit `NSStatusItem` and a borderless key-capable `NSPanel` holding SwiftUI content, macOS 14 minimum, in its own Swift package at `presence/`, a sibling of `go/`. `PresenceCore` is Foundation only and holds the decoding, the fold, the tail, liveness and the bundle-version check; `PresenceUI` the views and the glyph; `Presence` the executable. XCTest over fixtures, render tests to PNG, no UI tests. The panel is plain native presentation in this milestone; the two palettes as drawn are M24 | presence/ |
| The app tails this and last month's file with `DispatchSource`, rescans at the month boundary on a timer, reopens when the file is replaced or removed, parses newline-terminated records only, keeps the trailing partial bytes in a bounded buffer, skips a record it cannot decode and keeps tailing, refuses an unknown `v` as "gdoc is newer than this app", ignores unknown fields, and treats every line as forgeable data: it never takes a URL, a path or a program from the log. Liveness is `kill(pid, 0)`; a reused pid within one day is an accepted, unmeasured risk, and `run` pairs events without the kernel | presence/ |
| The app launches no process in this milestone. It opens only the event files under `~/.config/gdoc-agent/presence/` and its own `Info.plist`. A source scan refuses `URLSession`, `import Network` and `Process` anywhere in the package. It never opens the token or `login-pending.json`. The account row, `auth status` and `auth login` from the panel are M24 | presence/ |
| The first glance, this milestone: the glyph is a ring while a start has no end and its pid is alive, the mark while a `comments --wait` runs, a pause glyph while a hold waits, and nothing when idle; the panel is NOW and TODAY from the fold, a row for a start with no end and a dead pid ("stopped, outcome unknown"), the app's version and the last event's `gdoc` version, the "Updated, quit and reopen" row when the plist differs, and Quit. The app's only state is its dismissals, in UserDefaults | presence/ |
| No channel into `gdoc mcp` from the app. The hold card says "Open chat" and nothing else; the panel never says "Release". Product decisions 31, 47 and 48 wait for Nail | presence/ |
| No macOS notifications. Rule 7 of the brainstorm holds for every always-visible surface: the glyph, its tooltip, the Dock. The panel opens on a click and may show titles; the signed-in email joins it with the account row in M24 | presence/ |
| CLAUDE.md gains the presence rows and the two invariants, stays under 300 lines, and its sentence about `~/.local/bin/gdoc` linking to `bin/gdoc` is corrected | docs |

## What is not built

The panel of the design files: DOCUMENTS with its 14-day horizon, Yesterday,
Show more, the action icons, Stop live, the stuck card's dismiss, the account
row, Sign in and Sign out, Launch at login, the self-relaunch after an
update, the green check, the empty and error states, the two palettes as
drawn. That is M24, planned after Nail has used M23 and the swap while
running is measured. `gdoc auth logout`. A discard of a hold from the panel.
A `gdoc status` command. A socket, a pipe, a notification. Developer ID and
notarisation. Windows. A change to `release/install.sh` or `install.sh`. A
rewrite of the help goldens. An amendment to PRINCIPLES.md: Codex read the
strain on principle 2 as needing a written exception rather than a strain,
because the file holds document ids and titles outside front matter; a
principle is argued first, so that sentence is Nail's to write or refuse, and
the gate above states the strain as it stands.

## Codex's read, 2026-10-17

Codex (gpt-6-astra, read-only) read the mechanism document against the tree
before this spec was finished. Accepted and written into the rules above:
`release.yml` runs on tags only, so the push job lives in `go.yml`; a nightly
tag whose release failed is never retried by the nightly, and the plan
review then found that the Actions page's re-run already covers it, so the
nightly stays as it is; the schema carried no gdoc version, and `ok` is on
the envelope rather than in `data`; `title` and `verified` exist on some
commands only, and the plan review narrowed `verified` to the five read-back
writers because `update` and `restyle` print a false that means no write; a recovered panic
would have left a start with no end in a live process, so the end is
deferred; `started` had no defined source, so a random `run` id pairs events
and the kernel is not asked; "the app and the binary cannot come from
different releases" contradicted "an app failure never undoes the binary",
so skew is accepted and shown; `O_WRONLY` was missing and symlinks at the
path were not refused; chat work before `safeDispatch` is not covered and is
now said so. Codex's cuts for one night, taken: no self-relaunch, no account
row and no `auth status` from the app, a plain native panel. Declined: cutting
TODAY and titles (the fold is fixture-tested and titles are Nail's vote), and
renaming "stuck", which is the product decisions' word and the app's reading
only.

## Measurement

Three, each written into `docs/v2/MEASURED.md` with how it was taken, and
Nail confirms the method before the app trusts the claim.

1. Two writers. Two built `gdoc` binaries append 10,000 lines each to one
   file in a shell loop; every line decodes strictly and all 20,000 are there.
   The `-race` test with sixteen goroutines on separate file descriptions is
   the offline half.
2. Gatekeeper, the Go route. A bundle built by the release job, unpacked by
   the updater's own code into a temp directory: `xattr -lr` shows no
   quarantine, `codesign --verify --deep --strict` says valid, `open` launches
   it with no dialog and `syspolicyd`'s log shows no block. The negative
   control: a copy with the quarantine attribute written by hand must be
   blocked. What can be measured before the first release is measured on a
   locally built bundle and said so.
3. The swap while running. A second build with another cdhash replaces the
   bundle by rename while the first runs; the running app keeps working and
   shows the "Updated, quit and reopen" row; after a quit and an `open`,
   exactly one instance runs at the new version. Whether a self-relaunch is
   safe is M24's question, asked only after this is recorded.

If measurement 2 fails on Nail's machine, the run stops and Nail decides
between the browser route with "Open Anyway" and waiting for M17's signing.

## Acceptance

Verified 2026-10-17 at the end of the run, on `d050dc4` plus this commit, with
`make test` (race), `make vet`, `make app-test` and `make app-lint` green and
`make app` building the universal bundle. One row per rule above, in the order
the two tables state them: six rules Nail gave, then the twenty-four the design
pass took. Three rows are pass with a note, and the notes are below the table.

Nail's six, from the brief:

| Rule | Held by | Result |
|---|---|---|
| gdoc does not change in kind; it reports what it is doing | `TestTheFlagChangesNoByteOfTheObject`, `TestOffTouchesNothing`, `TestWithThePresenceOffAnUpdateIsTodaysUpdate`, and the suite as it stood before the milestone | pass |
| the presence is a separate application | its own Swift package at `presence/`, built by `make app` and tested by `make app-test`; nothing under `go/` reads it, and the one word both sides share is pinned by `TestTheBundleNameIsOneWordInThreePlaces` | pass |
| `gdoc update` updates the application too | `TestAnUpToDateRunWithThePresenceOnInstallsItFromTheReleaseHere`, `TestTheSwapKeepsThePreviousBundle`, `TestRollbackWithThePresenceOnSwapsBoth`, `TestTheResultSaysToOpenTheAppOnce` | pass |
| a flag in settings, also settable by an argument | `TestPresenceIsOffByDefault`, `TestTheFileTurnsPresenceOn`, `TestThePresenceArgumentOverridesTheFile`, `TestPresenceFromSaysWhichSourceWon` | pass |
| every existing function keeps working exactly as today | `TestAnExperimentalFlagIsInTheTableAndInNoScreen` over the 15 help screens and both completion scripts, `TestTheHelpObjectCarriesThePresenceOnlyWhenItIsOn`, `TestTheFlagChangesNoByteOfTheObject`, and every golden in the tree unchanged | pass |
| plan, TDD, review, run board, release | twelve tasks, one commit each, the test first in every task that builds; the release is the nightly after the merge | pass, note 1 |

The twenty-four the design pass took:

| Rule | Held by | Result |
|---|---|---|
| one append-only JSONL file per month, 0700 and 0600, one open-write-close, a symlink refused, the UTC month, local disk only | `TestPresenceDirIsInTheConfigFolder`, `TestTheMonthNamesTheFile`, `TestTheFileIsReadableByItsOwnerAlone`, `TestASymlinkOrANonRegularFileAtThePathIsRefusedAndNothingIsWritten`, `TestAShortWriteIsAFailure`, `TestTwoWritersNeverTearALine`; the volume sentence is in `docs/guide/presence.md` and `go/internal/presence/doc.go` | pass |
| the binary never reads the file, and no decision depends on a line in it | `TestPresenceExportsNoReader`, `TestOnlyPresenceAppends`, `TestBeginHasOneCaller` | pass |
| the event is a fixed allowlisted struct, schema `v: 1`, never the envelope | `TestTheEventCarriesOnlyItsListedFields`, `TestAnEventCarriesNoWordOfTheEnvelopeButTheTitle`, `TestOkIsTheEnvelopesOk`, `TestTitleIsLiftedOnlyWhenPresentAsAString`, `TestVerifiedIsLiftedOnlyWhenTheCallerSaysTheCommandReadsBack`, `TestOnlyTheReadBackWritersCarryVerified`, `TestTheDurationIsMonotonic`, `TestAnUpdateAndALoginWriteNoDoc` | pass |
| `title` is the one document word, capped at 200 runes | `TestTheTitleIsCappedAtTwoHundredRunes`; measured at 200 bytes a line at most in MEASURED.md's 20,000-line section | pass |
| `start` and `end` for the reporting commands, `held` then exactly one ending for a hold, and nothing from `help`, `completion`, `auth status`, `build`, `probe`, the session itself or chat work before `safeDispatch` | `TestAReportingCommandWritesAStartAndAnEnd`, `TestAHelpRunWritesNoEvent`, `TestAnAuthStatusRunWritesNoEvent`, `TestAChatCallSaysViaChat`, `TestAChatRefusalBeforeDispatchWritesNothing`, `TestAHeldWriteRecordsTheRuleAndNoneOfItsWords`, `TestEveryWayAHoldLeavesIsExactlyOneLine` | pass |
| the end is deferred, so a recovered panic still writes `ok: false`, and a start with no end from a dead pid is an unknown outcome | `TestAPanicInACommandStillWritesAnEndWithOkFalse`; the app half is `testAStartWithNoEndFromALivePidIsRunning` and `testAStartWithNoEndFromADeadPidIsOutcomeUnknown` | pass |
| a failed write changes no byte of the envelope and no exit code, one line to stderr once | `TestAnUnwritableLogChangesNoByteOfTheObject`, `TestAFailedWriteSaysSoOnceOnStderrAndReturnsNothing` | pass |
| the current and the previous month kept, exact names only, and nothing removed with the flag off | `TestOnlyTheCurrentAndPreviousMonthSurviveAndOnlyExactNamesAreRemoved`, `TestOffRemovesNothing` | pass |
| one fixture, written by the Go writer and held equal in both languages | `TestTheFixtureIsWhatTheWriterWrites`, `TestTheSwiftFixtureIsTheGoFixture` | pass |
| the setting `presence: true`, the argument `--presence true`, the same refusals, the argument over the file, chat from the file | `TestABadPresenceValueIsRefusedNamingWhereItCameFrom`, `TestAPresenceKeyWithNoValueIsRefused`, `TestPresenceFromIsSetOnlyWhenTheFlagIsOn`, `TestTheMarkAndThePresenceComeOutOfOneFile`, `TestMcpTakesTheSettingFlagsAndNothingElse` | pass, note 2 |
| in the table, marked experimental: no screen line, no completion line, `help --json` only when it is on | `TestTheTableNamesTheOneExperimentalFlag`, `TestAnExperimentalFlagIsInTheTableAndInNoScreen`, `TestTheHelpObjectCarriesThePresenceOnlyWhenItIsOn` | pass |
| off means today's bytes, proven by byte compare | `TestTheFlagChangesNoByteOfTheObject`, `TestOffTouchesNothing`, `TestOffRemovesNothing`, `TestWithThePresenceOffAnUpdateIsTodaysUpdate`, `TestCheckNeverInstallsTheApp` | pass |
| no skill passes the flag | `TestNoSkillPassesThePresenceFlag` | pass |
| the app travels in both darwin zips only, one universal bundle, under the one `SHA256SUMS`, and a repack keeps the signature | `TestTheZipCarriesThePresenceForDarwinOnly`, `TestThePresenceJobProvesTheZipRoundTripKeepsTheSignature`, `TestTheBundleComesOutOfAZipPackedByZipItself`, `TestTheReleaseNamesTheFilesTheUpdaterFetches` | pass |
| `release.yml` gains the macOS job the ubuntu job needs, and `go.yml` gains the same job on every push | `TestTheReleaseWaitsForThePresenceBuild`, `TestThePresenceIsStampedWithTheTag`, `TestThePresenceJobProvesTheZipRoundTripKeepsTheSignature`, `TestTheCIBuildsThePresenceOnEveryPush`, `TestTheReleaseWalksThePlatformsFileRatherThanASecondList` | pass, note 3 |
| the install at `~/Applications`, the previous one kept, every refusal by name, the stage read back and summed, the rename after the binary's, every app failure a warning, and the rollback an involution | `TestABundleIsTakenOutOfTheVerifiedZipOnly`, `TestOnlyEntriesUnderTheBundlePathAreTaken`, `TestAPathOutsideTheBundleASymlinkAndAnAppleDoubleEntryAreRefusedByName`, `TestTooManyEntriesOrTooManyBytesAreRefusedByName`, `TestAnotherIdentifierOrAnotherVersionIsRefusedByName`, `TestModesComeFromTheZip`, `TestTheStageIsReadBackAndSummed`, `TestAStaleStageIsRemovedFirst`, `TestTheSwapKeepsThePreviousBundle`, `TestAFailedSwapPutsTheOldBundleBack`, `TestRollbackOfTheBundleRunTwiceIsWhereItStarted`, `TestRollbackOfTheBundleWithNoPreviousIsRefusedByName`, `TestADestinationThatIsASymlinkAFileOrAnotherAppMovesNothing`, `TestAMissingApplicationsFolderIsMade`, `TestAPresenceFailureIsAWarningNamingBothPathsAndTheBinaryIsUpdated`, `TestRollbackWithNoPreviousAppWarnsAndKeepsIt`, `TestCheckNeverInstallsTheApp` | pass |
| gdoc never launches the app, and the app shows a reopen row instead of relaunching | `TestTheResultSaysToOpenTheAppOnce`, `TestNothingRunsAnExternalProgram`, `TestOnlyDesktopRunsAProgram`, `TestTheOneProgramOpensOnlyTheExtension`, `testTheReopenRowAppearsWhenThePlistVersionDiffers`, `testTheBundleVersionCheckIsAPureFunction` | pass |
| `release/install.sh` ignores `presence/`, and a non-darwin platform gets one warning | `TestTheReleaseInstallerLeavesThePresenceAlone`, `TestThePresenceOffMacOSWritesNothingAndNamesThePlatform` | pass |
| native Swift, three targets, macOS 14, XCTest over fixtures, render tests to PNG, no UI tests | `presence/Package.swift` with `PresenceCore`, `PresenceUI` and `Presence`; 26 tests under `make app-test`, `make app-lint` clean, and `testThePanelRendersNowAndTodayInLightAndDark` writing the two PNGs | pass |
| the tail: two months, the month boundary, reopen on replace, newline-terminated records, a bounded partial buffer, a skipped record, an unknown `v` refused, unknown fields ignored, every line forgeable data, liveness by `kill(pid, 0)` | `testTheTailReadsAppendsWaitsForTheNewlineAndFollowsRotation`, `testTheTailReopensWhenTheFileIsReplacedOrRemoved`, `testTheTailReadsThisAndLastMonthAtLaunchAndRescansAtTheMonthBoundary`, `testOnlyNewlineTerminatedRecordsAreParsedAndTheTailIsBounded`, `testAnUndecodableRecordIsSkippedAndTailingGoesOn`, `testAnUnknownVersionIsRefusedAsNewer`, `testANewerGdocIsSaidInTheNotice` for the words a person reads, `testAVersionUnderTheKnownOneIsSkippedRatherThanCalledNewer`, `testAVersionOneLineDecodesAndAnUnknownFieldIsIgnored`, `testAMissingRequiredFieldSkipsTheRecord`, the two liveness tests | pass |
| the app launches no process and opens only the event files and its own plist | `testNoSourceFileImportsNetworkOrLaunchesAProcess` over every `.swift` file in `Sources` and `Tests`, and `TestPresenceOpensNoWireAndRunsNothing` on the Go side | pass |
| the first glance: the four glyph states, NOW and TODAY, the stopped row, the two version rows, the reopen row, Quit, and dismissals the only state | `testTheGlyphIsOneOfFourStatesByPriority`, `testNowIsHoldsThenLiveStartsThenStopped`, `testTodayIsTheLocalDayNewestFirst`, `testADuplicateRecordCountsOnce`, `testAWaitThatIsRunningIsAWatch`, `testAHeldEventIsAHoldUntilReleasedExpiredOrLanded`, `testAHoldPastItsUntilIsShownAsRanOutEvenWithNoEvent`, `testAStoppedRunWithUnknownOutcomeIsItsOwnRow`, `testTheVersionRowsNameTheAppAndTheLastGdoc`, `testThePanelRendersNowAndTodayInLightAndDark` | pass but the dismiss, which is not built: the app writes nothing in this milestone, `AppState` says so, and the row a person cannot clear is `docs/backlog/now-keeps-a-dead-run-for-two-months.md` and M24 |
| no channel into `gdoc mcp`, and the panel never says "Release" | `testNoSourceFileImportsNetworkOrLaunchesAProcess`; the panel's one control is Quit, and the hold row is a row with no action, so the "Open chat" words wait for M24's card with the rest of it | pass |
| no macOS notifications | no file in `presence/` names `UserNotifications` or any notification API; the surfaces are the glyph, its tooltip and the panel, and `presence/Info.plist` carries `LSUIElement` so there is no Dock item either | pass |
| CLAUDE.md gains the rows and the two invariants, stays under 300 lines, and the `~/.local/bin/gdoc` sentence is corrected | `TestCLAUDEmdIsUnderTheCeiling` at 297 of 300 lines, `TestTheTaskMapNamesFilesThatExist`, and the corrected paragraph at `CLAUDE.md:264` | pass |

The three notes:

1. **The release is Nail's step, as every milestone's is.** The plan, the TDD,
   the reviews and the board ran here; the nightly after the merge
   (`gh workflow run nightly.yml --ref main`) and the tag it cuts are the
   Post-completion list of the plan beside this file.
2. **`gdoc mcp` takes `--presence` on its launch line, like `--mark` and
   `--config`.** The rule's sentence, "chat gets it from the file only", holds
   where it matters: no tool call can carry the flag, and a chat tool call
   never sets it. What a session is given is resolved once, before the session
   starts, from the file or from the line that started it, which is how the two
   older setting flags already work. Reading it any other way would have made
   `mcp` the one command that refuses a setting flag.
3. **The release job is proven by reading the workflow, not by running it.**
   `TestTheReleaseWaitsForThePresenceBuild` and its four neighbours parse
   `release.yml` and `go.yml`; the first CI-built bundle, the first darwin zip
   carrying it and the first `gdoc update` that installs from a real release
   are Nail's, the morning after. The locally built bundle was measured instead,
   and MEASURED.md says which half is which.

The three measurements the Measurement section asks for are in `docs/v2/MEASURED.md`:
"The presence app built, rendered and unzipped", "What Gatekeeper says about a
bundle gdoc's own code unzipped", "Twenty thousand presence lines from two
binaries into one file" and "The bundle swapped under a process running out of
it". Each names how it was taken and which half is Nail's, because nothing here
launches a GUI app. Measurement 2 did not fail on the Go route, so the run did
not stop and M17 did not move up.
