# gdoc v2 Milestone 24a: the panel, the Swift half. Plan

2026-10-17. The task list for the Swift half of the specification
`docs/plans/2026-10-17-gdoc-v2-m24-the-panel-spec.md`. One commit per task.
The spec holds the rules; this plan repeats none of them. Where the code, the
spec or this plan disagree, the task stops and says so. The visual source of
truth is `docs/design/2026-10-17-presence-screens-v3.html` (its document
titles and addresses are invented stand-ins of the same lengths): when this
plan and that file disagree on a pixel, the file wins; when the file and the
product decisions disagree on a behaviour, the spec's decision table wins.
It starts from `main` after M23 merged and runs beside M24b in another
worktree; it touches `presence/` and `docs/` only, and never
`presence/Sources/PresenceCore/Event.swift` or the two fixtures, which are
M24b's. The record's new fields reach the scenes in the last task, after
M24b merged.

## Principles

Serves 4 and 3, strains 1 and 2 as the spec says.

## Context (from discovery)

- `presence/Sources/PresenceCore/Event.swift` decodes a record line strictly
  on `v`, counts an unknown kind as skipped, ignores unknown fields; its
  `EventKind` is switched exhaustively at `Fold.swift:122,155` and
  `Event.swift:187`, so no new code in this half writes an exhaustive switch
  over it (M24b adds kinds). `Fold.swift` builds `Glance{now, today, gdoc}`
  newest first; `Liveness.swift:20` is `kill(pid, 0)` behind an injected
  check; `LogTail.swift` tails two month files with `DispatchSource`;
  `BundleVersion.swift` is the pure `same | reopenNeeded | gone`.
- `presence/Sources/PresenceUI/Glyph.swift`, `PanelView.swift` (with an
  `enum Metrics` at `:14`), `Rows.swift` (with `public struct PanelModel` at
  `:60`) are the M23 plain panel; `main.swift:57`, `StatusItem.swift:35-40`
  and `AppState.swift:90` use them, so each stays until the task that replaces
  its users. `AppState.swift:25` holds the 5 s refold and the bundle
  watcher; `StatusItem.swift:20` the item at `squareLength`; `Panel.swift`
  the borderless key-capable `KeyPanel`.
- `presence/Package.swift`: targets `PresenceCore`, `PresenceUI`, `Presence`
  (the executable), `PresenceCoreTests`, `PresenceUITests`; an executable
  target cannot be imported by a test target, so the logic the app needs
  tested lives in a library.
- `presence/Tests/PresenceCoreTests/SourceRulesTests.swift` refuses
  `URLSession`, `import Network` and `Process` in every source and test file:
  each task below that introduces an API rewrites the rule for it in the
  same commit. `Tests/PresenceUITests/RenderTests.swift` writes two PNGs
  under `PRESENCE_SHOTS`.
- The screens file: the token sheet is its `<style>` (`:root` dark values,
  `.screen.light` overrides); the scenes are the `SCENES` array, nineteen
  ids r1..r9, r3l, e1..e3, e5, e6, e8..e10 (e4 is not built); the icons are
  the `IC` object's SVG paths; the title split is `TT()`; the action set is
  `ACTS()`; the NOW order is scene e3.
- `go/cmd/gdoc/read.go:589` traps SIGINT and SIGTERM for the wait and
  answers `data.waited.interrupted` (`:415`); `skills/gdoc-review/live.md:26,43`
  reads it as "you stopped it".
- `go/cmd/gdoc/loginscreen.go:64` prints `Open this link in your browser to
  sign in:` and the URL on stderr when stderr is not a terminal; the URL is
  `https://accounts.google.com/o/oauth2/auth?...` (`go/internal/auth/login.go:25`,
  `pending.go:101-103`), not the `/v2/auth` the screens draw.
- `gdoc help --json` carries `settings.mark`, `update.installed` and
  `update.latest_stable`; `gdoc auth status` carries `token_present`,
  `expired`, `missing_scopes`, `account`, `version`. `help` runs the daily
  GitHub check once in 24 hours, so the app asks it on open only.
- `~/.local/bin/gdoc` is a release binary on Nail's machine and a symlink on
  the checkout route, so paths are resolved before comparing.
- `docs/v2/MEASURED.md`: the swap under a running instance (09:25) and the
  launch of the installed bundle (09:17), both 2026-10-17.
- `docs/backlog/now-keeps-a-dead-run-for-two-months.md` is closed by the
  dead-pid rule.
- `CLAUDE.md` is at 297 lines and M24b changes one word in it; this half
  owns the invariant's words in its last task, after the merge.

## Development Approach

TDD, one commit per task, the test first and failing, under
`cd presence && swift test`; `make app-lint` and `make app` green at every
commit; `make test` green with the live variables unset, because
`go/boundary` reads `presence/`. Every colour, size, weight, delay and count
a test states is a literal. No task launches a GUI app; child-process tests
run a `/bin/sh` fixture that prints an envelope, never the real gdoc.
Renders go under `PRESENCE_SHOTS`, outside the repository. Any scene data a
test needs is injected as plain values, never read from the machine. A task
that finds the screens file, the spec and the code disagree stops and says
so.

## Validation Commands

```bash
make app-lint
make app-test
make app
env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test
PRESENCE_SHOTS=/tmp/presence-shots make app-test
```

## Progress Tracking

Twenty-six tasks. The checkboxes live inside each task; a task is done when
every box is ticked and its commit is on the branch.

## Tasks

### Task 1: the token sheet

**Files:**
- Create: `presence/Sources/PresenceUI/Theme/Palette.swift`, `Typography.swift`, `Grid.swift`, `Timing.swift`
- Create: `presence/Tests/PresenceUITests/ThemeTests.swift`

- [x] `testTheDarkPaletteIsTheScreensFileLiterals` and
      `testTheLightPaletteIsTheScreensFileLiterals`: every colour of the spec's
      token sheet as a literal hex and alpha, read from the CSS by hand into
      the test; `testTheGridIsTheScreens` (320, 8, 12, 600, 36, 28, 32, 44, 28,
      108, 16, 24, 6); `testTheTimingsAreTheDecisions` (0.5, 1, 3, 5, 180, 600)
- [x] `Palette.dark` and `Palette.light` with the fields the blueprint lists,
      chosen by the SwiftUI colour scheme and never by a setting; `Typography`
      with the twelve roles; `Grid` (named so because `Metrics` already exists
      in `PanelView.swift` until Task 13); `Timing`
- [x] `make app-lint`, `make app-test`

### Task 2: the status glyph and the ring

**Files:**
- Create: `presence/Sources/PresenceUI/Components/StatusGlyph.swift`, `RingView.swift`
- Create: `presence/Tests/PresenceUITests/GlyphTests.swift`

- [x] `testEveryGlyphKindAndToneRenders` (seven kinds by six tones, 16 px,
      stroke 1.25 dark and 1.5 light, a PNG sheet under `PRESENCE_SHOTS`),
      `testTheHollowCheckIsNeutral`, `testTheRingIsTwoSizes` (16 at 1.5, 28 at
      2), `testAnIndeterminateRingIsStaticUnderReduceMotionAndInRenderMode`
- [x] the views drawn with `Path`, the geometry from the CSS; the M23
      `Glyph.swift` stays untouched until Task 14
- [x] `make app-lint`, `make app-test`

### Task 3: the action icons

**Files:**
- Create: `presence/Sources/PresenceUI/Components/IconShapes.swift`, `ActionButton.swift`
- Create: `presence/Tests/PresenceUITests/IconTests.swift`

- [ ] `testTheIconSheetRenders` (gdocs, md, branch, link, done, stop, down,
      signin, signout, x, xmark, chevron at 16 px, stroke 1.25, round caps),
      `testAnOffIconIsDimmedWithItsOwnTooltip` (0.38),
      `testADoneIconIsGreenForOneSecondThenReturns`
- [ ] `IconShape` as `Path` ports of the `IC` SVG paths; `ActionButton` 24 px,
      radius 6, states live, off(tip), done
- [ ] `make app-lint`, `make app-test`

### Task 4: rows, sections, header, titles

**Files:**
- Create: `presence/Sources/PresenceUI/Components/RowView.swift`, `SectionView.swift`, `PanelHeader.swift`, `MiddleTitle.swift`, `QuietLine.swift`, `ChevronRow.swift`
- Create: `presence/Sources/PresenceCore/Titles.swift`
- Create: `presence/Tests/PresenceCoreTests/TitlesTests.swift`, `presence/Tests/PresenceUITests/RowTests.swift`

- [ ] `testTheSplitIsTheScreensRule` (26 whole, a tail of 14 or fewer with its
      space, `TT()` ported case by case over invented titles of 20, 40, 60 and
      80 characters), `testNoHeadUnderEightCharactersAtPanelWidth`,
      `testRowHeightsAreTheGrid` (32, 44, 28 by preference key),
      `testTheDocRowActionColumnIs108`,
      `testASectionLabelIs28AndUppercaseWithTheCount`,
      `testTheHeaderIs36WithTheMarkTheNameAndTheAccount`,
      `testAGoneTitleIsStruckThrough`
- [ ] the views in the five layouts, hover fill, busy hides icons
- [ ] `make app-lint`, `make app-test`

### Task 5: tooltip, switch, confirm, buttons

**Files:**
- Create: `presence/Sources/PresenceUI/Components/TooltipLayer.swift`, `SwitchView.swift`, `ConfirmRow.swift`, `ButtonStyles.swift`
- Create: `presence/Tests/PresenceUITests/ControlTests.swift`

- [ ] `testATooltipAppearsAfterHalfASecondAndFlipsInsideThePanel`,
      `testATooltipCanBeForcedForARender`, `testTheSwitchIsDrawnNotToggle`,
      `testAConfirmRowRevertsAfterThreeSeconds`,
      `testTheOnlyTextButtonsAreOpenChatAndTheConfirmPair` (a scan over the
      component sources for `Button(` with a text label)
- [ ] the views; the tooltip anchored by `anchorPreference`
- [ ] `make app-lint`, `make app-test`

### Task 6: the cards and the small pieces

**Files:**
- Create: `presence/Sources/PresenceUI/Components/HoldCard.swift`, `StuckCard.swift`, `LiveRow.swift`, `JobCard.swift`, `EmptyState.swift`, `SignedInBanner.swift`, `URLLine.swift`, `IconStrip.swift`, `HealthRows.swift`
- Create: `presence/Tests/PresenceUITests/CardTests.swift`

- [ ] `testTheHoldCardSaysOpenChatAndDismissAndNeverRelease`,
      `testTheStuckCardReasonFitsTwoLinesUnderEightyCharacters`,
      `testTheStuckCardDismissIsTheLastIcon`,
      `testTheLiveRowHasOneStopIconThatBecomesARingWhilePending`,
      `testTheJobCardShowsStepRowsOnlyWhenStepsAreKnown`,
      `testTheEmptyStateIsTheFirstRunText`, `testTheBannerLivesFiveSeconds`,
      `testTheStripLabelsAreDocNoteChatLink`,
      `testHealthRowsKeepTheirOrderOnEveryScreen`
- [ ] the views as the blueprint draws them, each driven by a plain model
      struct and closures, no process and no file behind any of them
- [ ] `make app-lint`, `make app-test`

### Task 7: the scroll box

**Files:**
- Create: `presence/Sources/PresenceUI/Components/ScrollBox.swift`
- Create: `presence/Tests/PresenceUITests/ScrollTests.swift`
- Modify: `docs/v2/MEASURED.md`

- [ ] measure first and write it into MEASURED.md: whether `ImageRenderer`
      draws a SwiftUI `ScrollView`; the render mode clips a plain stack if not
- [ ] `testTheBoxCapsItsHeightAndScrollsInsideWithTheThumbAndTheFade` (over
      forty plain rows, by preference key), `testShowMoreAndShowLessToggleTheList`
- [ ] `make app-lint`, `make app-test`

### Task 8: the process check

**Files:**
- Create: `presence/Sources/PresenceCore/ProcessCheck.swift`
- Create: `presence/Tests/PresenceCoreTests/ProcessCheckTests.swift`

- [ ] `testAWaitIsStoppableOnlyWhenEveryCheckHolds`: a table over path not
      gdoc, another uid, start time outside ten seconds before the event, a
      dead pid, an end already written, `via: chat`;
      `testPreviousCountsAsGdocAndPathsAreResolvedBeforeComparing`
- [ ] the pure compare over injected facts (resolved path, uid, start time)
- [ ] `make app-test`

### Task 9: the fold, NOW

**Files:**
- Modify: `presence/Sources/PresenceCore/Fold.swift` (new functions beside the M23 ones; the M23 `Glance` stays until Task 13)
- Create: `presence/Sources/PresenceCore/Now.swift`
- Create: `presence/Tests/PresenceCoreTests/NowTests.swift`, `Fixtures/now-order.jsonl`
- Delete: `docs/backlog/now-keeps-a-dead-run-for-two-months.md`

- [ ] `testNowOrdersHoldsStuckJobsThenWatchesOldestFirst` (scene e3),
      `testAFailedWriteIsStuckUntilItsKeyClearsOrIsDismissed` (per document
      for the commands that name one; a failed publish clears when any later
      publish starts), `testAFailedReadIsNotStuck`, `testVerifiedFalseIsNeverStuck`,
      `testADeadPidWriteWithinFourteenDaysIsStuckAndOlderIsUnknownInToday`,
      `testAHoldCardTakesTheDocumentsLatestTitle`,
      `testDismissalsAreAnInputAndTheRowStays`,
      `testTheLastVerifiedWriteIsReportedForTheCheck`
- [ ] the fold with dismissals and the clock injected; the stuck set is the
      literal list reply, propose, withdraw, annotate, publish, restyle, export
- [ ] `make app-test`

### Task 10: the fold, live sessions

**Files:**
- Create: `presence/Sources/PresenceCore/LiveSession.swift`
- Create: `presence/Tests/PresenceCoreTests/LiveSessionTests.swift`, `Fixtures/live-day.jsonl`

- [ ] `testWaitsWithinThreeMinutesAreOneSession`,
      `testRepliedAndProposedCountTheDocumentsOkEnds`,
      `testTheLastWriteTimeIsTheQuietAnchor`, `testAFailedPollIsAFact`,
      `testTheSessionKeepsPidRunAndCallForStop`,
      `testNoSessionSaysWaitingForTheAuthor` (a scan over the words)
- [ ] the session fold
- [ ] `make app-test`

### Task 11: the fold, DOCUMENTS, TODAY, Yesterday

**Files:**
- Create: `presence/Sources/PresenceCore/Documents.swift`, `Days.swift`
- Create: `presence/Tests/PresenceCoreTests/DocumentsTests.swift`, `DaysTests.swift`, `Fixtures/fourteen-days.jsonl`

- [ ] `testDocumentsGroupByIdInsideFourteenDaysNewestFirst`,
      `testAReadAloneMakesNoRow`, `testTheLabelCountsTheHorizonAndToday`,
      `testTheDefaultIsFiveWithShowMore`,
      `testTodayIsTheLocalDayNewestFirstDefaultEight`,
      `testTodayCarriesTheSyntheticRows` (hold raised, hold closed, live
      started, live stopped with its totals), `testReadsAreLeftOutOfToday`,
      `testYesterdayIsOneSummaryThatOpensInline`, `testNothingOlderAppears`,
      `testAMonthBoundaryIsNotADayBoundary`
- [ ] the two folds
- [ ] `make app-test`

### Task 12: the words

**Files:**
- Create: `presence/Sources/PresenceUI/Words.swift`
- Create: `presence/Tests/PresenceUITests/WordsTests.swift`

- [ ] `testTheVerbsAreTheSevenWords`, `testTimesAreClockYesterdayWeekdayThenDate`,
      `testDurationsAreMinutesAndHours`,
      `testViaIsFromTheTerminalOrFromDesktopChat`,
      `testVerifiedIsVerifiedOrNotReadBack`, `testNothingPredictsATime`
- [ ] the functions; `Rows.swift` and its `PanelModel` stay until Task 13
- [ ] `make app-lint`, `make app-test`

### Task 13: the panel model, the panel view, the scenes

**Files:**
- Create: `presence/Sources/PresenceUI/PanelModel.swift` (replaces the one in `Rows.swift`), `UIState.swift`, `Facts.swift`
- Modify: `presence/Sources/PresenceUI/PanelView.swift` (replaced), `presence/Sources/PresenceUI/Rows.swift` (removed), `presence/Sources/Presence/main.swift` and `AppState.swift` (use the new model), `presence/Package.swift` (the `Scenes` resources)
- Create: `presence/Tests/PresenceUITests/Scenes/*.jsonl`, `Scenes.swift`
- Modify: `presence/Tests/PresenceUITests/RenderTests.swift`

- [ ] `UIState` and `Facts` are plain injected data covering every state a
      later task will produce: health facts (account, auth word, versions,
      mark, gdoc path found or not, hub set or not, extension), the sign-in
      phase with its URL and countdown, the pairing map, a pending stop, the
      remembered account, expanded sections, a forced tooltip, a copied icon,
      a confirm row, the strip seen, the banner. Tasks 16 to 24 only produce
      these values and never edit `PanelView`
- [ ] one fixture per scene the v1 record can draw: r3, r3l, r4, r5, r6, r7,
      r9, e1, e2, e3, e5, e8, e10; and the stated fallbacks: r1 draws an
      indeterminate ring and "Publishing" with no title, r2 draws the reason
      "Stopped at 09:05. The terminal that ran it has the reason.", r8 draws
      the live row without the network word, e6 draws the row without the
      strike-through, e9 draws from auth facts alone; r3's "from Claude Code"
      and r9's "stopped by you" draw their fallbacks ("from the terminal",
      "stopped") until Task 26; e4 is not built and the test says so
- [ ] `testEveryScreenRendersInBothAppearances`: `<id>-dark.png`,
      `<id>-light.png` at 2x plus `index.html` with each scene's title and
      caption under `PRESENCE_SHOTS`; `testTheMetricProbesHoldOnEveryScene`;
      `testThePanelNeverPassesSixHundred` (e1, e3);
      `testEveryReasonLineIsUnderEightyCharacters`; `testNowIsHiddenWhenEmpty`;
      `testHealthIsPinnedFirstWhenSignedOut`
- [ ] `make app-lint`, `make app-test`, `make app`

### Task 14: the menu bar image

**Files:**
- Create: `presence/Sources/PresenceUI/BarImage.swift` (in the library so the tests reach it)
- Modify: `presence/Sources/Presence/StatusItem.swift`, `AppState.swift`; `presence/Sources/PresenceUI/Glyph.swift` (removed)
- Modify: `presence/Tests/PresenceUITests/RenderTests.swift` (the bar per scene)
- Create: `presence/Tests/PresenceUITests/BarTests.swift`

- [ ] `testOneTemplateImagePerState` (live, stuck, hold, job, check, idle at
      0.45), `testEveryBarPixelIsMonochrome`, `testThePriorityIsStuckHoldJobLiveIdle`,
      `testNotSignedInExpiredAndGdocNotFoundAreStuck`,
      `testNetworkTroubleDoesNotChangeTheIcon`,
      `testTheCheckLastsFiveSecondsAfterALiveVerifiedEndAndNeverForAReplay`,
      `testTheTooltipNamesTheStateAndNoDocument`, and the render test writes
      `<id>-bar-dark.png` and `<id>-bar-light.png` per scene
- [ ] `variableLength`, the composed images, the five-second timer
- [ ] `make app-lint`, `make app-test`, `make app`

### Task 15: the app library and the runner

**Files:**
- Modify: `presence/Package.swift` (a library target `PresenceApp` the executable and a new `PresenceAppTests` target import; `Presence` keeps `main.swift` and the AppKit shell)
- Create: `presence/Sources/PresenceApp/Runner.swift`
- Create: `presence/Tests/PresenceAppTests/RunnerTests.swift`, `Fixtures/fake-gdoc.sh` (mode 755)
- Modify: `presence/Tests/PresenceCoreTests/SourceRulesTests.swift` (`Process` only in `Runner.swift`, the argv enum closed at six)

- [ ] `testGdocIsResolvedAtLocalBinThenBesideTheBundleNeverFromPATH`,
      `testTheArgvSetIsExactlyTheSix`, `testTheChildEnvironmentIsTheFourVariables`
      (with `GDOC_CONFIG_DIR` set in the test's own environment and absent in
      the child's), `testStdinIsTheNullDevice`, `testOneChildPerKindAtATime`,
      `testTheEnvelopeIsDecodedStrictly`, `testATimeoutTerminatesTheChild`,
      over `fake-gdoc.sh`; `testProcessAppearsOnlyInTheRunner`
- [ ] the runner; the "gdoc not found" fact with the path tried, for HEALTH
- [ ] `make app-lint`, `make app-test`, `make app`

### Task 16: health facts and the polls

**Files:**
- Create: `presence/Sources/PresenceCore/HealthFacts.swift`, `presence/Sources/PresenceApp/HealthPoller.swift`
- Create: `presence/Tests/PresenceCoreTests/HealthFactsTests.swift`, `presence/Tests/PresenceAppTests/HealthPollerTests.swift`

- [ ] `testHelpJSONGivesTheMarkTheVersionAndTheLatest`,
      `testAuthStatusGivesTheThreeWordsAndTheEmail`,
      `testATokenWithNoAccountReadsSignedInAccountNotRead`,
      `testTheExpectedDomainIsAHintThatBlocksNothing`,
      `testAccountChangedIsTheAppsOwnMemory`,
      `testThePollIsOnOpenAtMostOnceAMinuteAndEveryTenMinutesWhileLive`,
      `testHelpIsAskedOnOpenAndNeverOnATimer`
- [ ] the parsers in core, the schedule in the app library, the facts fed
      into `Facts`
- [ ] `make app-test`

### Task 17: the opener, the hub index, sign in

**Files:**
- Create: `presence/Sources/PresenceApp/Opener.swift` (every `NSWorkspace` and `NSPasteboard` use), `presence/Sources/PresenceCore/HubIndex.swift`, `LinkLine.swift`, `presence/Sources/PresenceApp/SignIn.swift`
- Create: `presence/Tests/PresenceCoreTests/HubIndexTests.swift`, `LinkLineTests.swift`, `Fixtures/hub/` (three notes with `gdoc:` blocks, one without), `presence/Tests/PresenceAppTests/OpenerTests.swift`, `SignInTests.swift`
- Modify: `presence/Tests/PresenceCoreTests/SourceRulesTests.swift` (`NSWorkspace` and `NSPasteboard` only in `Opener.swift`, schemes https and obsidian, hosts docs.google.com and accounts.google.com)
- Modify: `docs/v2/MEASURED.md`

- [ ] `testTheDocsURLIsBuiltFromAValidIdOnly`, `testCopyLinkIsThePlainURL`,
      `testNoTitleEverReachesAURLOrThePasteboard`,
      `testTheHubIndexReadsOnlyTheFrontMattersGdocBlock`,
      `testTheIndexRescansOnAFileEvent`,
      `testOpenMdStaysInsideTheChosenVaultAndNeedsARegularMdFile`,
      `testAnUnpairedDocumentGreysTheIconWithItsTooltip`,
      `testAMovedNoteSaysMovedOrDeleted`,
      `testContinueWithClaudeOnlyActivatesTheApp`,
      `testTheStripShowsOnceUntilAnIconIsClicked`;
      `testTheURLIsTakenOnlyFromTheChildsStderrAfterTheSentence`,
      `testOnlyGooglesSignInURLIsAccepted` (the literal path `/o/oauth2/auth`
      from `login.go:25`; http, a lookalike host, userinfo, `javascript:` and
      the `/v2/` path each refused), `testTheCountdownIs180Seconds`,
      `testCancelTerminatesTheChild`, `testTheBannerFollowsAnOkEnvelope`
- [ ] the opener, the index with FSEvents and a debounce, the Obsidian URL,
      the sign-in flow; measured and written into MEASURED.md: whether
      LaunchServices lists a `claude://` handler on this machine (read with
      `lsregister -dump`, no launch), so Continue with Claude's prefill stays
      out until it does
- [ ] `make app-lint`, `make app-test`

### Task 18: stop live

**Files:**
- Create: `presence/Sources/PresenceApp/Stopper.swift`
- Create: `presence/Tests/PresenceAppTests/StopperTests.swift`
- Modify: `presence/Tests/PresenceCoreTests/SourceRulesTests.swift` (`kill(` with a signal only in `Stopper.swift`; `Liveness.swift`'s `kill(pid, 0)` exempt by its zero)
- Modify: `docs/v2/MEASURED.md`

- [ ] `testStopSendsOneSIGINTOnlyAfterTheProcessCheck` (a fake signaller),
      `testStopNeverSendsAnotherSignal`, `testAFailedCheckGreysStopWithItsTooltip`,
      `testTheCardShowsARingUntilTheEndArrives`
- [ ] measured and written into MEASURED.md: what `proc_pidpath` returns for
      a running `sleep` started from a file that is then renamed to
      `.previous`, as `gdoc update` renames the binary
- [ ] `make app-test`

### Task 19: the dismissals store and the update action

**Files:**
- Create: `presence/Sources/PresenceApp/Preferences.swift`, `UpdateAction.swift`
- Create: `presence/Tests/PresenceAppTests/PreferencesTests.swift`, `UpdateActionTests.swift`

- [ ] `testDismissalsAreKeyedByRunAndCallOrHoldId`,
      `testDismissalsArePrunedPastSixtyTwoDays`,
      `testOnlyTheSixKeysPersist` (dismissed, hubPath, stripSeen,
      loginItem.choice, relaunch, lastAccount); `testUpdateNeverPassesAFlag`,
      `testTheGdocRowShowsTheRingUntilTheEndThenTheNewVersion`,
      `testInstallExtensionIsUpdateDesktop`
- [ ] `make app-test`

### Task 20: the relaunch

**Files:**
- Create: `presence/Sources/PresenceCore/RelaunchPolicy.swift`, `presence/Sources/PresenceApp/Relauncher.swift`
- Create: `presence/Tests/PresenceCoreTests/RelaunchPolicyTests.swift`
- Modify: `presence/Tests/PresenceCoreTests/SourceRulesTests.swift` (`openApplication` only in `Relauncher.swift`)

- [ ] `testRelaunchHappensOncePerOnDiskVersionInTenMinutes`,
      `testNoRelaunchWhileAnUpdateRunsOrAChildLives`,
      `testNoRelaunchIntoAnotherIdentifierOrAnInvalidSignature`,
      `testGoneNeverRelaunches`, `testAFailedOpenLeavesTheReopenRow`,
      `testANewInstanceWaitsForItsLauncherThenASecondInstanceQuits`
- [ ] the policy as a pure function, the executor with
      `createsNewApplicationInstance`, `activates = false`, the
      `--relaunched-by` argument, the three guards
- [ ] `make app-lint`, `make app-test`, `make app`

### Task 21: launch at login

**Files:**
- Create: `presence/Sources/PresenceApp/LoginItem.swift`
- Create: `presence/Tests/PresenceAppTests/LoginItemTests.swift`
- Modify: `presence/Tests/PresenceCoreTests/SourceRulesTests.swift` (`SMAppService` only in `LoginItem.swift`)

- [ ] `testTheFirstEverRunRegistersOnceAndRecordsOn`,
      `testAPersonsOffIsNeverOverridden`, `testARelaunchNeverRegisters`,
      `testRequiresApprovalShowsTheRowAndIsNotRetried`, over an injected
      service
- [ ] `make app-lint`, `make app-test`, `make app`

### Task 22: wiring the shell

**Files:**
- Modify: `presence/Sources/Presence/main.swift`, `AppState.swift`, `StatusItem.swift`, `Panel.swift`

- [ ] `testTheShellOnlyWiresAndHoldsNoLogic` (a scan: the executable target's
      files import `PresenceApp` and `PresenceUI` and define no function over
      events, processes or URLs)
- [ ] the shell feeds `Facts` and `UIState` from the pollers, the tail, the
      stores and the actions, and nothing else
- [ ] `make app-lint`, `make app-test`, `make app`

### Task 23: the reviewer's pass, first round

**Files:**
- Modify: whatever the findings name under `presence/`

- [ ] render every scene (`PRESENCE_SHOTS=... make app-test`), then run the
      design principles' checklist over `index.html` as a reviewer who drew
      nothing (a subagent), and fix every finding in this task before anything
      is shown; the findings and fixes are listed in the commit message
- [ ] `make app-lint`, `make app-test`

### Task 24: the words

**Files:**
- Modify: `docs/guide/presence.md`, `CLAUDE.md`

- [ ] `docs/guide/presence.md` gains the panel: each section, each action,
      Sign in and Sign out, Launch at login, the six argv lists the app runs
      and what it never does, that `help --json` on open is the daily check
      gdoc already makes, the hub folder, the dismissal, what Stop live can
      and cannot stop; CLAUDE.md's presence invariant names the six argv lists
      and the hold's sign-in rule in place of words cut elsewhere, the file at
      or under 300; `TestTheTaskMapNamesFilesThatExist` green
- [ ] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`

### Task 25: after M24b merged, the fields reach the scenes

**Files:**
- Modify: `presence/Sources/PresenceCore/Fold.swift`, `Now.swift`, `LiveSession.swift`, `presence/Sources/PresenceUI/Words.swift`, `JobCard.swift`, `StuckCard.swift`, `PanelModel.swift`, the scene fixtures, `RenderTests.swift`

- [ ] this task starts only after `main` with M24b is merged into this branch
      and `make app-test` is green on the merge; it says so in its commit
- [ ] `testADroppedHoldEndsTheCard`, `testAPublishedRowTakesItsIdFromMade`,
      `testTheStuckReasonIsTheCodesWordsAndTheLine`,
      `testStepEventsDriveTheJobCardAndTheBarRing`,
      `testViaCodeReadsFromClaudeCode`, `testAnInterruptedWaitIsLiveStoppedByYou`;
      scenes r1, r2, r3, r8, r9, e6 and e9 re-fixtured on the new fields and
      re-rendered; the reviewer's pass run again over the changed scenes
- [ ] `make app-lint`, `make app-test`, `make app`

### Task 26: the acceptance rows and the close

**Files:**
- Modify: `docs/plans/2026-10-17-gdoc-v2-m24-the-panel-spec.md`, `docs/v2/PLAN.md`

- [ ] the Swift rows of the spec's acceptance table under a heading "M24a",
      pass or the test that says otherwise
- [ ] the M24a half of the M24 row in `docs/v2/PLAN.md`'s Done table
- [ ] this plan and the spec move to `docs/plans/completed/`

## Post-completion

**Nail's measurements, confirmed by him, into MEASURED.md.** Self-relaunch
twenty times under a swapped bundle, one instance each time at the new
version. The login item under an ad-hoc update through a log out and log in,
with the count of macOS notices. A SIGINT from the panel to a wait under
Claude Code's Bash tool: the end carries `interrupted: true` and the skill
prints its totals.

**Nail's morning after.** `gdoc update` to the release, then nothing: the app
is already registered to start at login and relaunches itself after the
update. The first open after a fresh install stays his.

**Release.** `make tag VERSION=v2.15.0` after both halves merged.
