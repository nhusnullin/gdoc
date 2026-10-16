# gdoc v2 Milestone 23: the presence. The channel, the flag, the delivery, the first glance. Plan

2026-10-17. The task list for the specification beside it,
`docs/plans/2026-10-17-gdoc-v2-m23-the-presence-spec.md`, whose six brief
rules are Nail's and whose design rules are listed there for him to strike.
One commit per task. The spec holds the rules. This plan repeats none of them.
Where the code, the spec or this plan disagree, the task stops and says so
rather than choosing. The design record
`docs/design/2026-10-17-presence-mechanism.md` is history; where it and the
spec differ, the spec holds. It starts from `main` after M22 and M21 merged,
and after the decision entry and the SPEC.md paragraph it names are
committed there.

## Principles

Serves 4 and 3, strains 2 and 1, as the spec says.

## Context (from discovery)

Every reference below was read on 2026-10-17 before M21 and M22 merged. Task 1
re-checked every line number in this list against `main` as it stands and
corrected the five that had moved: `present()`, both SPEC.md sentences, the
CLAUDE.md ceiling test and the README's `gdoc update` paragraph. `update.go`
and `main.go` had not moved at all.

- `go/cmd/gdoc/main.go:244` `dispatch`: matches the table entry `c`, parses
  `a`, sets `a.set`, returns `c.run(ctx, a, errOut)` at `:280`. `:179`
  `printResult` prints every terminal envelope; chat prints through
  `mcpEnvelope` in `mcptools.go`. `:220` `safeDispatch` is the chat route and
  the panic envelope, and the cache, the argument refusals, `mcpPin` and
  `mcpJudge` run before it. `:110` `route` resolves settings before dispatch,
  so a settings refusal stops every command.
- `go/cmd/gdoc/read.go:132` `documentID` is the existing id parser.
- `go/cmd/gdoc/commands.go:148` `type command struct`, the one table; `:133`
  `settingFlags()` holds `--config` and `--mark` with their sentences.
  `go/cmd/gdoc/settings.go:50` `takeSettings` takes them off the line.
  `go/cmd/gdoc/mcp.go:30` `checkMCPArgs` admits the setting flags only.
  `go/cmd/gdoc/help.go:43` `helpSettings` prints `mark` and `mark_from`.
- Tests that name "the two setting flags": `cmd/gdoc/settings_test.go:1`,
  `:361` `TestMcpTakesTheSettingFlagsAndNothingElse`, `:529`
  `TestEverySettingFlagIsInHelpAndInBothScripts`, `completion_test.go:480`,
  `help_test.go:436`. The help screens are the 15 bare goldens among the 23
  files under `cmd/gdoc/testdata/screens/`, through `helpscreen_test.go`.
  There is no `help --json` golden under `testdata/screens/`, but
  `testdata/pipe/help.stdout.golden` is the same object frozen read-only, so
  Task 1 captured `help --json` with the flag off before its change as a
  sha256 and a length, both literals, rather than as 8,921 bytes inline.
- `go/internal/settings/settings.go`: `key = "mark"`, `Given{Config, Mark}`,
  `InEffect{Mark, MarkFrom, File}`, `file{Mark *string}`, `Resolve` lays the
  argument over the file over the default; `:117` `present()` tells a key
  with no value from a key nobody wrote. `doc.go` holds the rules and
  `TestAnUnknownKeyIsRefusedByName`, `TestTheTokenIsNotASetting`. Measured
  with goccy/go-yaml v1.19.2: `yes`, `1` and `""` into a `*bool` are refused
  with a message naming the Go field, not the key, so the refusal is worded
  here; `presence:` with no value decodes to nil with no error; `True` is
  accepted from the file.
- `go/cmd/gdoc/mcphold.go:53` `mcpHoldLife`, `:113` `keep`, `:134` `hold`,
  `:149` `sweep`, `:182` `landed`; the release path is in `mcprelease.go`;
  `chat.Hold` carries `ID`, `DocID`, `Rule`, `Tool`, `Created`.
  `go/cmd/gdoc/mcplog.go` is the fact-line precedent, with five tests.
- `go/internal/config/config.go:66-115`: one function per per-user file.
  `go/internal/lastcheck/doc.go`: the stamp's strict read. `go/internal/atomicfile`.
- `go/cmd/gdoc/update.go:110` `updateData.Verified` has no omitempty, so a
  no-op update and a rollback print `verified: false`; `restyle.go:166` the
  same with no batches. `:207` `binaryPath`, `:276` `runUpdate`, `:372`
  `installSteps`, `:392` `releaseHere`, `:410` `refreshExtension`, `:428`
  `installDecided`, `:462` `withExtension` (the one caller of `runProgram`,
  `:476`), `:508` `install`. `go/internal/update/apply.go:42` `Verify`,
  `:73` `Apply`, `:153` `FileFrom`, `:167` `Rollback`, `:225` `fileIn`;
  single files only, and `update/doc.go:94` says so.
  `go/cmd/gdoc/desktop.go`: `runProgram`, `macOSOpen`.
  `go/boundary/boundary_test.go:573` `TestNothingRunsAnExternalProgram`,
  `:602` `TestOnlyDesktopRunsAProgram`, `:273` `TestNetHTTPStaysInItsRooms`
  (parses `.go` files only).
- `.github/workflows/release.yml`: `on: push: tags` and `workflow_call`, one
  job on `ubuntu-latest`, the pack loop over `release/platforms`, one
  `sha256sum`. `.github/workflows/go.yml`: job `test` on `ubuntu-latest`,
  on every push. `go/boundary/workflows_test.go:357` `releaseSteps` requires
  one job; `:66` `Needs string`.
  `TestTheReleaseWalksThePlatformsFileRatherThanASecondList` refuses a
  platform named in `release.yml`. A failed release is re-run from the
  Actions page; the nightly is not touched.
- `docs/v2/SPEC.md:730` "A zip carries six things"; `:754` "the one program
  gdoc runs is `/usr/bin/open`". `docs/v2/PLAN.md:33` the Done table.
- `CLAUDE.md` is 300 lines and `TestCLAUDEmdIsUnderTheCeiling`
  (`go/boundary/docs_test.go:42`) allows 300; M22 adds rows first. `:257`
  says `~/.local/bin/gdoc` links to `bin/gdoc`: true for the checkout route
  (`docs/guide/from-a-checkout.md:19`, `install.sh:20`), false on Nail's
  machine, where it is the v2.12.1 release binary with `gdoc.previous`
  beside it. `README.md` is 200 lines and `releaseREADMECeiling` is 200
  (`go/boundary/release_test.go:258`). `docs/guide/` has no page for
  `gdoc update`; the README holds it at `:171`.
  `TestAccountOfHasTwoCallers`, which Task 4 puts its own pin beside, is in
  `go/boundary/account_test.go:52` and not in `boundary_test.go`.
- Toolchain on Nail's machine, measured 2026-10-17: Xcode 26.6 selected,
  Swift 6.3.3, `swift format` 6.3.0, `xctest` present, no signing identity.
  A tools-6.0 package with `.copy("Fixtures")` and `Bundle.module` passes
  `swift test`; `swift build -c release --arch arm64 --arch x86_64` builds
  universal with no Xcode project; a target declared with no source file
  fails the build from a clean clone, and git keeps no empty directory.
- The five design files under `docs/design/` and the mechanism record.

## Development Approach

TDD, one commit per task, the test first and failing, except where a task
says a test is a pin that passes before and after. Go under `make test` with
the race flag, offline. Swift under `cd presence && swift test`, offline,
over fixtures. No task launches a GUI app or any process from Swift. Every
id, every hash, every size and every path a test states is a literal. A task
that finds the spec and the code disagree stops and says so in the task log.
The order keeps every commit green from a clean clone: the setting lands
before anything reads it, the package before the hook, the extraction before
the command, the Swift core before the executable, the executable before the
workflow that builds it.

## Validation Commands

```bash
make test
make vet
cd go && go test ./internal/settings ./internal/presence ./cmd/gdoc -run 'Presence|Setting|Flag|Event' -v
cd go && go test ./internal/update ./boundary -run 'Presence|Bundle|Program|Release' -v
# from Task 7:
cd presence && swift test
# from Task 8:
make app && ls bin/GdocPresence.app/Contents/MacOS
```

## Progress Tracking

Twelve tasks. The checkboxes live inside each task below; a task is done when
every box in it is ticked and its commit is on the branch.

## Tasks

### Task 1: the `presence` setting and the experimental `--presence` argument

**Files:**
- Modify: `go/internal/settings/settings.go`, `go/internal/settings/doc.go`, `go/internal/settings/settings_test.go`
- Modify: `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/settings.go`, `go/cmd/gdoc/help.go`
- Modify: `go/cmd/gdoc/settings_test.go`, `go/cmd/gdoc/completion_test.go`, `go/cmd/gdoc/help_test.go`

- [x] re-check the line numbers in Context against `main` as it stands and
      correct this plan where they moved
- [x] capture `gdoc help --json` with the flag off before any change, into
      the test as a literal, for the byte compare below (8,921 bytes,
      sha256 `40a28f2a…`, in `cmd/gdoc/presenceflag_test.go`)
- [x] `TestPresenceIsOffByDefault`, `TestTheFileTurnsPresenceOn`,
      `TestThePresenceArgumentOverridesTheFile`,
      `TestABadPresenceValueIsRefusedNamingWhereItCameFrom` (`yes`, `1`, an
      empty value and `True` on the argument, each naming the file and key
      `presence` or `--presence`, in gdoc's words and not go-yaml's),
      `TestAPresenceKeyWithNoValueIsRefused` (through `present()`),
      `TestPresenceFromIsSetOnlyWhenTheFlagIsOn`,
      `TestPresenceFromSaysWhichSourceWon`; `TestAnUnknownKeyIsRefusedByName`
      and `TestTheTokenIsNotASetting` unchanged
- [x] `file` gains `Presence *bool` under `presence`, `Given` gains
      `Presence *string` taking `true` or `false` only, `InEffect` gains
      `Presence bool` and `PresenceFrom`; `Resolve` lays them in the same
      order as the mark
- [x] `settingFlags()` gains the entry with its sentence and an
      `experimental` bit; `takeSettings` takes it with the same refusals as
      `--mark`; `checkMCPArgs` still admits setting flags only;
      `helpSettings` gains `presence` and `presence_from`, both `omitempty`,
      set only when on
- [x] `TestAnExperimentalFlagIsInTheTableAndInNoScreen`: the 15 help screens,
      both completion scripts and the captured `help --json` with the flag
      off are byte for byte as before; `TestEverySettingFlagIsInHelpAndInBothScripts`
      skips experimental flags, and the "two setting flags" comments say
      three, one experimental. No golden moves
- [x] `settings/doc.go` gains the rule and why the flag is a setting and not
      an account check; `make test`

### Task 2: the event, the file and the writer, `internal/presence`

**Files:**
- Create: `go/internal/presence/doc.go`, `event.go`, `append.go`, `presence_test.go`, `testdata/events-v1.jsonl`
- Modify: `go/internal/config/config.go`, `go/internal/config/config_test.go`

- [ ] `TestOffTouchesNothing` (`GDOC_CONFIG_DIR` under a 0500 directory, no
      `presence/` appears, no error), `TestTheEventCarriesOnlyItsListedFields`
      (json tags against the spec's literal list, `gdoc` and `run` among them),
      `TestAnEventCarriesNoWordOfTheEnvelopeButTheTitle` (canaries in data
      text, threads, warnings, an error quoting words, a path, an email),
      `TestOkIsTheEnvelopesOk`, `TestTitleIsLiftedOnlyWhenPresentAsAString`
      (a data with none writes none; a `title` that is a number is not
      lifted), `TestVerifiedIsLiftedOnlyWhenTheCallerSaysTheCommandReadsBack`
      (a `verified: false` from a caller without the read-back bit writes no
      `verified`), `TestTheTitleIsCappedAtTwoHundredRunes`,
      `TestTheDurationIsMonotonic`, `TestTheMonthNamesTheFile`,
      `TestTheFileIsReadableByItsOwnerAlone`,
      `TestASymlinkOrANonRegularFileAtThePathIsRefusedAndNothingIsWritten`,
      `TestAShortWriteIsAFailure`, `TestTwoWritersNeverTearALine`
      (16 goroutines on separate opens, 500 lines each, all decode),
      `TestAFailedWriteSaysSoOnceOnStderrAndReturnsNothing`,
      `TestOnlyTheCurrentAndPreviousMonthSurviveAndOnlyExactNamesAreRemoved`
      (a stray `notes.txt` and a `2026-01.jsonl.bak` in the directory stay),
      `TestOffRemovesNothing`, `TestTheFixtureIsWhatTheWriterWrites` (a
      scripted run against `events-v1.jsonl`, byte for byte, with a fixed
      clock, pid, run id and version; the script covers every event kind and
      every optional field at least once, chat and terminal both)
- [ ] `config.PresenceDir()` beside the other paths, with its test
- [ ] `Event` as the spec lists it; `Begin(ctx, on bool, version, command,
      doc string, wait, readsBack bool, errOut) *Run`, `(*Run).End(emit.Result)`,
      `Held`, `Released`, `Expired`, `Landed`, `WithChat(ctx) context.Context`
      and its reader, every one safe on nil and on off; the writer is one
      open with `O_WRONLY|O_APPEND|O_CREATE|O_NOFOLLOW` at 0600 in a 0700
      directory, one write checked for its length, one close; the run id is
      eight random bytes as hex taken once per process; the clock, the pid,
      the run id and the monotonic source behind variables for the fixture
      test
- [ ] `doc.go` holds the rules with the test for each, and why the binary
      never reads the file

### Task 3: the hook in `dispatch` and in the holds

**Files:**
- Modify: `go/cmd/gdoc/main.go`, `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/mcp.go`, `go/cmd/gdoc/mcphold.go`, `go/cmd/gdoc/mcprelease.go`
- Create: `go/cmd/gdoc/presence_test.go`
- Modify: `go/cmd/gdoc/mcphold_test.go`

- [ ] `TestTheFlagChangesNoByteOfTheObject`: read, comments --wait with a done
      context, reply and publish over the stubbed session, stdout, stderr and
      exit code equal with the flag off and on, the only difference the file;
      `TestAnUnwritableLogChangesNoByteOfTheObject` (the presence path is a
      file: same object, same exit, one stderr line);
      `TestAReportingCommandWritesAStartAndAnEnd` (doc at start, doc, title,
      ok at end, `via: terminal`);
      `TestOnlyTheReadBackWritersCarryVerified` (reply, propose, withdraw,
      annotate, publish do; read, comments, restyle, update do not);
      `TestAnAuthStatusRunWritesNoEvent`, `TestAHelpRunWritesNoEvent`;
      `TestAChatCallSaysViaChat` through `mcpSend`;
      `TestAChatRefusalBeforeDispatchWritesNothing` (a wrong title, a missing
      code); `TestAPanicInACommandStillWritesAnEndWithOkFalse` (through
      `safeDispatch`'s recovery); `TestAnUpdateAndALoginWriteNoDoc`;
      `TestAHeldWriteRecordsTheRuleAndNoneOfItsWords`,
      `TestEveryWayAHoldLeavesIsExactlyOneLine` (released, expired, landed);
      `TestStdoutCarriesOnlyJSONRPC` run again with the flag on
- [ ] the table entry gains `report bool` and `readsBack bool`: `report` true
      for read, comments, suggestions, export, restyle, reply, propose,
      withdraw, annotate, publish, update, auth login; `readsBack` true for
      reply, propose, withdraw, annotate, publish; `dispatch` calls `Begin`
      after the parse and defers `End` on a named result, so a panic that
      `safeDispatch` recovers still ends the run; `serveMCP` wraps the
      session context with `presence.WithChat`; `mcpHolds.keep`, the release
      path, the expiry branch of `sweep` and `landed` each write their event
      once
- [ ] `make test`; the canary test and the held-write test are the guards
      over the ledger, since `TestTheLedgerIsPerProcessAndWritesNothingToDisk`
      reads only `ledger.go`'s imports

### Task 4: the boundary over the file, and the words in the doc comments

**Files:**
- Create: `go/boundary/presence_test.go`
- Modify: `go/internal/comments/doc.go`, `go/internal/chat/doc.go`, `go/cmd/gdoc/doc.go`

- [ ] `TestOnlyPresenceAppends` (`O_APPEND` outside `_test.go` only in
      `internal/presence`), `TestPresenceOpensNoWireAndRunsNothing` (no `net`,
      `net/http`, `os/exec`, `internal/chat`, `internal/gapi`),
      `TestPresenceExportsNoReader` (no exported `Read*` or `Open*`, no
      `os.Open` or `os.ReadFile` outside `_test.go`), `TestBeginHasOneCaller`
      (after `TestAccountOfHasTwoCallers`), `TestNoSkillPassesThePresenceFlag`
- [ ] `comments/doc.go`'s "Nothing is kept. The wait writes no file of its
      own" gains the sentence that `dispatch` writes a start and an end for it
      under the setting, and why that is a record for another program and not
      session state; `chat/doc.go` says the hold event carries no ledger word;
      `cmd/gdoc/doc.go` gains the setting flag paragraph and the two columns
      of the table

### Task 5: a bundle out of the verified zip, `internal/update`

**Files:**
- Create: `go/internal/update/bundle.go`, `go/internal/update/bundle_test.go`, `go/internal/update/testdata/bundle.zip`
- Modify: `go/internal/update/doc.go`

- [ ] `TestABundleIsTakenOutOfTheVerifiedZipOnly` (a zip failing `Verify`
      extracts nothing), `TestOnlyEntriesUnderTheBundlePathAreTaken`,
      `TestAPathOutsideTheBundleASymlinkAndAnAppleDoubleEntryAreRefusedByName`
      (`..`, an absolute path, a symlink entry, `__MACOSX/`, `._x`, a duplicate
      path, two paths differing only in case),
      `TestTooManyEntriesOrTooManyBytesAreRefusedByName` (2,000 entries, 64
      MiB expanded), `TestAnotherIdentifierOrAnotherVersionIsRefusedByName`
      (the plist read strictly with `encoding/xml`, so the bundle's plist
      stays XML), `TestModesComeFromTheZip` (0755 where any exec bit, else
      0644, 0755 for folders), `TestTheStageIsReadBackAndSummed` (sorted path
      NUL sha256 lines), `TestAStaleStageIsRemovedFirst`,
      `TestTheSwapKeepsThePreviousBundle`, `TestAFailedSwapPutsTheOldBundleBack`,
      `TestRollbackOfTheBundleRunTwiceIsWhereItStarted`,
      `TestADestinationThatIsASymlinkAFileOrAnotherAppMovesNothing`,
      `TestAMissingApplicationsFolderIsMade`
- [ ] `ApplyBundle(archive, sums, asset, inZip, dest) (BundleResult, error)`
      and `RollbackBundle(dest)`; the stage is `<dest>.new`, removed first if
      stale, the previous `<dest>.previous`, `RemoveAll` on the stale
      previous because rename will not replace a folder, every rename after
      the first undone on failure; `BundleResult` carries `Version`,
      `TreeSum`, `Previous`
- [ ] `update/doc.go` gains the section and amends its "one named file"
      sentence, with the tests by name

### Task 6: `gdoc update` installs, refreshes and rolls back the app under the flag

**Files:**
- Modify: `go/cmd/gdoc/update.go`, `go/cmd/gdoc/update_test.go`, `go/cmd/gdoc/updatescreen_test.go`
- Modify: `go/boundary/boundary_test.go`

- [ ] pins, passing before and after: `TestWithThePresenceOffAnUpdateIsTodaysUpdate`
      (envelope, stderr, the screen goldens, and the disk, byte for byte),
      `TestTheOneProgramOpensOnlyTheExtension` in `go/boundary`
      (`withExtension` is the only caller of `runProgram`)
- [ ] failing first: `TestAPresenceFailureIsAWarningNamingBothPathsAndTheBinaryIsUpdated`,
      `TestAnUpToDateRunWithThePresenceOnInstallsItFromTheReleaseHere`,
      `TestRollbackWithThePresenceOnSwapsBoth`,
      `TestRollbackWithNoPreviousAppWarnsAndKeepsIt`,
      `TestThePresenceOffMacOSWritesNothingAndNamesThePlatform`,
      `TestCheckNeverInstallsTheApp`, `TestTheResultSaysToOpenTheAppOnce`
      (the step line names `open ~/Applications/GdocPresence.app` on a first
      install only)
- [ ] `updateData` gains `presence`, `presence_version`, `presence_sha256`,
      `presence_previous`, all omitempty; the bundle path in the zip and the
      destination behind variables like `executable`; the install runs the
      binary's `Apply`, then `ApplyBundle`, and a bundle failure is a warning
      naming both paths; the `releaseHere` route installs the app only when
      the installed binary is that release, the app is missing or at another
      version, and `--check` was not given; `runRollback` swaps the bundle
      after the binary when a previous exists and warns otherwise
- [ ] `make test`, with every update golden unchanged

### Task 7: the Swift package: the core, the fold, the tail, the shared fixture

**Files:**
- Create: `presence/Package.swift`, `presence/.swift-format`
- Create: `presence/Sources/PresenceCore/{Event,Fold,LogTail,Liveness,BundleVersion}.swift`
- Create: `presence/Tests/PresenceCoreTests/{EventTests,FoldTests,LogTailTests,LivenessTests,BundleVersionTests,SourceRulesTests}.swift`
- Create: `presence/Tests/PresenceCoreTests/Fixtures/{events-v1,day,torn,malformed,future}.jsonl`
- Modify: `Makefile`, `.gitignore`, `go/boundary/presence_test.go`

- [ ] `TestTheSwiftFixtureIsTheGoFixture` in `go/boundary`: the two
      `events-v1.jsonl` are byte for byte one file
- [ ] Swift tests, written first: `testAVersionOneLineDecodesAndAnUnknownFieldIsIgnored`,
      `testAnUnknownVersionIsRefusedAsNewer` (`future.jsonl`),
      `testAnUndecodableRecordIsSkippedAndTailingGoesOn` (`malformed.jsonl`),
      `testOnlyNewlineTerminatedRecordsAreParsedAndTheTailIsBounded`
      (`torn.jsonl`: a writer died mid-line and the next record glued on),
      `testAMissingRequiredFieldSkipsTheRecord`, `testADuplicateRecordCountsOnce`
      (`run` plus `call`), `testAStartWithNoEndFromALivePidIsRunning`,
      `testAStartWithNoEndFromADeadPidIsOutcomeUnknown`,
      `testAWaitThatIsRunningIsAWatch`,
      `testAHeldEventIsAHoldUntilReleasedExpiredOrLanded`,
      `testAHoldPastItsUntilIsShownAsRanOutEvenWithNoEvent`,
      `testTodayIsTheLocalDayNewestFirst` (clock, calendar and timezone
      injected; a UTC month boundary is not a local day boundary),
      `testTheTailReadsAppendsWaitsForTheNewlineAndFollowsRotation`,
      `testTheTailReopensWhenTheFileIsReplacedOrRemoved`,
      `testTheTailReadsThisAndLastMonthAtLaunchAndRescansAtTheMonthBoundary`,
      `testTheBundleVersionCheckIsAPureFunction` (`same`, `reopenNeeded`,
      `gone`), `testNoSourceFileImportsNetworkOrLaunchesAProcess` (the whole
      package: `URLSession`, `import Network`, and `Process` as a word, not
      `ProcessInfo`)
- [ ] `Package.swift`: tools 6.0, `.macOS(.v14)`, targets `PresenceCore` and
      `PresenceCoreTests` only (the executable and the UI targets are declared
      in Task 8, because a declared target with no source fails a clean
      clone), fixtures as `.copy("Fixtures")`; Makefile targets `app-test`
      (`cd presence && swift test`) and `app-lint`
      (`swift format lint --strict -r Sources Tests`); `.gitignore` gains
      `presence/.build/` and `presence/.swiftpm/`
- [ ] `make app-test`, `make app-lint`, `make test`

### Task 8: the app: the glyph, the panel, the reopen row

**Files:**
- Modify: `presence/Package.swift`, `Makefile`
- Create: `presence/Sources/PresenceUI/{Glyph,PanelView,Rows}.swift`, `presence/Sources/Presence/{main,StatusItem,Panel,AppState}.swift`
- Create: `presence/Tests/PresenceUITests/RenderTests.swift`, `presence/Info.plist`, `presence/bundle.sh`

- [ ] `testTheGlyphIsOneOfFourStatesByPriority` (hold over running over watch
      over idle, monochrome template images),
      `testThePanelRendersNowAndTodayInLightAndDark` (PNGs written under
      `PRESENCE_SHOTS`, a directory outside the repository, from `day.jsonl`,
      both appearances, 2x), `testAStoppedRunWithUnknownOutcomeIsItsOwnRow`,
      `testTheVersionRowsNameTheAppAndTheLastGdoc`,
      `testTheReopenRowAppearsWhenThePlistVersionDiffers`
- [ ] `Package.swift` gains `PresenceUI`, `Presence` and `PresenceUITests`;
      the shell: `NSStatusItem` with `sendAction(on: [.leftMouseUp,
      .rightMouseUp])`, a borderless key-capable `NSPanel` under the button
      with an `NSHostingView`, closed on resign key and on Esc; NOW and TODAY
      from the fold; Quit; the bundle watcher on the app's own path and its
      parent, debounced, reading `Contents/Info.plist` with
      `PropertyListSerialization` and never the cached `Bundle.main`, showing
      the "Updated to X. Quit and reopen" row on a difference and "gdoc
      presence was removed" when the bundle is gone; no process is launched;
      nothing is persisted
- [ ] `presence/Info.plist` as XML with `@VERSION@`, `LSUIElement`,
      `CFBundleIdentifier com.altery.gdoc.presence`,
      `LSMinimumSystemVersion 14.0`; `bundle.sh <version> <out>` builds
      universal with `swift build -c release --arch arm64 --arch x86_64`,
      fills the plist (`0.0.0` for `dev`), signs ad-hoc, verifies with
      `codesign --verify --deep --strict` and `lipo -archs`; Makefile `app`
      writes `bin/GdocPresence.app` from `$(VERSION)`
- [ ] `make app`, `make app-test`, `make app-lint`; the PNGs looked at

### Task 9: the release builds the app and the zips carry it

**Files:**
- Modify: `.github/workflows/release.yml`, `.github/workflows/go.yml`
- Modify: `go/boundary/workflows_test.go`, `go/boundary/release_test.go`, `go/boundary/presence_test.go`

- [ ] `releaseSteps` judges the job that runs `gh release create` instead of
      failing on two jobs; `TestTheReleaseWaitsForThePresenceBuild` (the
      publishing job needs `presence`; the presence job never runs `gh
      release` and has `contents: read`), `TestTheZipCarriesThePresenceForDarwinOnly`
      (the copy sits in a `darwin-*)` arm; a listing step names `presence/`
      and `exit 1` and comes before `gh release create`),
      `TestThePresenceIsStampedWithTheTag` (`plutil` against `$TAG`),
      `TestThePresenceJobProvesTheZipRoundTripKeepsTheSignature` (the macOS
      job zips the bundle with `zip -qr`, unzips it, and runs `codesign
      --verify --deep --strict`, because the ubuntu job has no `codesign`),
      `TestTheBundleNameIsOneWordInThreePlaces` (`GdocPresence.app` in
      `release.yml` as `presence/GdocPresence.app`, in the update constant as
      `presence/GdocPresence.app`, in the Makefile as `bin/GdocPresence.app`),
      `TestTheReleaseInstallerLeavesThePresenceAlone`,
      `TestTheCIBuildsThePresenceOnEveryPush` (the job is in `go.yml`,
      because `release.yml` runs on tags only)
- [ ] `release.yml` gains job `presence` on the macOS image whose Xcode is
      26 (the label checked against GitHub's runner list at run time and
      recorded in the task log, with `sudo xcode-select -s` to that Xcode so
      `swift format` is the version lint was written against): check out the
      tag, `swift --version`, `make app-lint`, `make app-test`,
      `presence/bundle.sh "$TAG" build/presence`, the asserts, the zip round
      trip and re-verify, `ditto -c -k --norsrc --noextattr --keepParent`,
      `actions/upload-artifact`; job `release` gains `needs: presence`, a
      download and unzip, the `darwin-*)` copy into `$stage/presence/`, and
      the listing step before publish
- [ ] `go.yml` gains the same `presence` job without the upload
- [ ] `make test`; `TestTheReleaseWalksThePlatformsFileRatherThanASecondList`
      and the nightly tests unchanged

### Task 10: the words: CLAUDE.md, the guide, SPEC.md's two sentences

**Files:**
- Modify: `CLAUDE.md`, `docs/v2/SPEC.md`
- Create: `docs/guide/presence.md`

- [ ] `CLAUDE.md`: `presence/` and `go/internal/presence/` in the table, one
      new invariant (the presence file is the one file gdoc appends to,
      written only under the setting, never read by the binary, carrying only
      the listed fields), the existing "nothing runs an external program"
      invariant gaining the clause that the app is never started by gdoc and
      `withExtension` stays the one caller, two task-map rows, and the
      `~/.local/bin/gdoc` sentence saying both routes: a link to `bin/gdoc`
      from a checkout, a release binary with `gdoc.previous` beside it where
      `gdoc update` installed. The cuts that pay for it: "Running a
      milestone" shortened to three lines pointing at
      `.ralphex/board/README.md`, and the "Skills are linked" invariant to
      five. The file stays at or under 300; `TestTheTaskMapNamesFilesThatExist`
      green
- [ ] `docs/v2/SPEC.md:719` says the zip carries seven things on darwin, the
      app the seventh; `:743` says the app is never one of them, gdoc runs
      the one program on the extension and nothing else
- [ ] `docs/guide/presence.md`: what the glyph means, the setting, the
      argument, the first `open`, the file and what is never in it, local
      disk only, the bootstrap order (update to the release that knows the
      flag, set it, run `gdoc update` again) and the rollback trap, what
      `spctl` says about an ad-hoc bundle and why that is expected, in reader
      words. No README line: the flag is experimental and the README is at
      its ceiling
- [ ] `make test`, because `go/boundary` reads these documents

### Task 11: the measurements

**Files:**
- Modify: `docs/v2/MEASURED.md`

- [ ] the prototype measurements of 2026-10-17 (the universal build time, the
      Go-unzipped bundle's attributes and signature, the headless render)
      written down with how each was taken
- [ ] two built binaries, a shell loop, 10,000 lines each into one file, a
      strict decode of all 20,000: the command, the result and the date
- [ ] the Go route on a locally built bundle, unzipped by the updater's own
      code into a temp directory: `xattr -lr`, `codesign --verify --deep
      --strict -vv`, `spctl -a -t exec -vv` recorded, with the note that
      `spctl` rejects every ad-hoc bundle and that is not the verdict; the
      launch itself is Nail's step and the section says so, with the negative
      control written out for him
- [ ] the swap while running, as far as it can be measured without launching
      a GUI app here: the rename sequence over a running `sleep` inside a
      bundle-shaped folder keeps the process alive, and the plist read after
      the swap names the new version; the live half, the reopen row appearing
      in a running app, is Nail's step and the section says so

### Task 12: the acceptance table and the close

**Files:**
- Modify: `docs/plans/2026-10-17-gdoc-v2-m23-the-presence-spec.md`, `docs/v2/PLAN.md`

- [ ] one row per rule in the spec, pass or the test that says otherwise,
      written into the spec's Acceptance section
- [ ] the M23 row in `docs/v2/PLAN.md`'s Done table, in the shape of M19's
- [ ] both plan files move to `docs/plans/completed/`

## Post-completion

**Nail's steps, the morning after.** Update to the release that carries the
flag, set `presence: true` in `~/.config/gdoc-agent/settings.yaml`, run `gdoc
update` once more, `open ~/Applications/GdocPresence.app`, then run a review
or a publish from Claude Code and watch the glyph. Then the three live
measurements as MEASURED.md writes them out.

**M24, the panel.** DOCUMENTS and the 14-day horizon, Yesterday, Show more,
the action icons, Stop live, the stuck card's dismiss, the account row and
`auth status`, Sign in and Sign out, Launch at login, the self-relaunch after
an update once the swap is measured, the green check, the empty and error
states, the two palettes as drawn. Planned after Nail has used M23.

**Nail's one principle question.** Codex read the file of document ids and
titles in the config directory as needing a written exception to principle
2 rather than a strain. A principle is argued first, so the sentence in
PRINCIPLES.md, if any, is Nail's.

**Release.** After the merge into `main`, trigger the nightly
(`gh workflow run nightly.yml --ref main`) and report the tag, as after every
merge. A release whose macOS job failed is re-run from the Actions page.
