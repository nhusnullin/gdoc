# gdoc v2 Milestone 24b: what the panel needs from the binary. Plan

2026-10-17. The task list for the Go half of the specification
`docs/plans/2026-10-17-gdoc-v2-m24-the-panel-spec.md`. One commit per task.
The spec holds the rules; this plan repeats none of them. Where the code, the
spec or this plan disagree, the task stops and says so. It starts from `main`
after M23 merged and runs beside M24a in another worktree. It touches `go/`,
the two fixtures, and on the Swift side only the decoder
(`presence/Sources/PresenceCore/Event.swift`), the two switches in
`presence/Sources/PresenceCore/Fold.swift` that must learn the new kinds,
and the tests that index into the fixture. It builds no meaning for the new
fields in the app: that is M24a's last task, after this half merges.

## Principles

Serves 3 and 4, strains nothing new, as the spec says.

## Context (from discovery)

- `go/cmd/gdoc/main.go:296-323`: `dispatch` begins the run and
  `presenceDoc(c, a)` fills `doc` only when the entry's first word is
  `<url>`; publish takes `--md` and `--folder-id`, so its record has no id.
  The end lifts `title` and, for the five read-back writers, `verified`.
- `go/cmd/gdoc/publish.go:99,133,138,170`: `cmdPublish` renders, then
  `publish.Run` creates, verifies and pairs; `data.document_id` is the created
  id, cleared on rollback. `go/internal/publish/publish.go` holds `Run`.
- `go/internal/presence/event.go`: the `Event` struct and its allowlist,
  pinned by `TestTheEventCarriesOnlyItsListedFields`; `(*Run).End` takes an
  `emit.Result`, whose `Error` is a string (`go/internal/emit/emit.go:29`),
  so the error value is gone before the end is written; `doc.go` the rules;
  `testdata/events-v1.jsonl` the fixture, pinned equal to
  `presence/Tests/PresenceCoreTests/Fixtures/events-v1.jsonl` by
  `go/boundary/presence_test.go:244` `TestTheSwiftFixtureIsTheGoFixture`;
  `presenceClosedTo` at `:30` keeps `net` and `internal/gapi` out of
  `internal/presence`.
- `emit.Result.Screen` (`emit.go`, `json:"-"`) is the precedent for a field a
  caller sets and the object never carries.
- The failures a stop code names are plain `fmt.Errorf` strings today: the
  picture that cannot be embedded at `go/internal/body/image.go:292` (with
  its line), Drive's 404 and the auth refusal in `gapi.statusError`
  (`go/internal/gapi/session.go:626`), the multi-tab refusal at
  `propose.go:381` and `annotate.go:189`, the guard's refusal at
  `go/internal/guard/policy.go:464`, the handshake and dial errors from
  `net`. None is a typed error yet.
- `go/cmd/gdoc/mcphold.go`: `keep:125`, `hold:152`, `sweep:167`,
  `landed:204`; `mcprelease.go:106` `mcpRelease`; `mcpSignedIn` at
  `mcptools.go:148` checks only that a token loads; `mcpSend` opens a fresh
  session from whatever token is there; a release with no token keeps the
  hold. `go/internal/gapi/session.go:570-614`: a session's own rotation is
  not a newer sign-in (`TestARotatedRefreshTokenIsNotASecondLogin`,
  `refresh_test.go:289`), another process's write is (`newerLogin`, `adopt`).
  So a hold bound to the refresh token's hash is dropped by this process's
  own rotation too, which fails safe and is written down as a strain.
- `go/internal/auth`: the token at `config.TokenPath()`, `Load`;
  `pending.go:71` `ReadLoginLock`, `:108` `RemoveLoginLock`; M17's Keychain
  is not shipped. The login command lives in `go/cmd/gdoc/login.go` and
  `loginscreen.go`.
- `go/cmd/gdoc/commands.go`: the table; `auth login` and `auth status` are
  two entries. Pins a new command breaks: the help screens under
  `cmd/gdoc/testdata/screens/`, `cmd/gdoc/testdata/pipe/help.stdout.golden`
  (`pipe_test.go:9-12` says a changed byte is a bug, and M19, M20 and M22
  re-recorded it for a line a milestone added, which this plan writes into
  that comment as the stated exception), both completion goldens,
  `presenceflag_test.go:27-28` (the help object's byte count and sha256),
  `cmd/gdoc/presence_test.go:288-292` (the `reports` map),
  `help_test.go:15-19` (`everyCommandName`), `main_test.go:215,223`
  (`TestTheUsageLineNamesEveryCommand`), CLAUDE.md:24 ("sixteen").
- `go/cmd/gdoc/read.go:415`: `data.waited.interrupted`; `:589` traps SIGINT
  and SIGTERM for the wait.
- Swift tests that index into the fixture and shift when lines are added:
  `presence/Tests/PresenceCoreTests/EventTests.swift:15` (ten lines, `[4]`,
  `[5]`), `LogTailTests.swift:38,71,101,126`. `Fold.swift:122,155` and
  `Event.swift:187` switch exhaustively over `EventKind`. `FoldTests.swift:142,152`
  and `LivenessTests.swift:49` build an `Event` directly, so new init
  parameters need defaults.
- `CLAUDE.md` at 297 lines: this half changes "sixteen" to "seventeen" on
  line 24 and nothing else; M24a owns the invariant's words after the merge.

## Development Approach

TDD, one commit per task, the test first and failing, `make test` with the
race flag, offline, with the live variables unset in every validation run;
every Go change that touches the record regenerates the fixture and copies
it to the Swift side in the same commit, updates the Swift tests that index
into it, and teaches `Event.swift` the field or kind with a decode test, so
`TestTheSwiftFixtureIsTheGoFixture` and `make app-test` are green at every
commit. The Swift decoder gives every new init parameter a default. A task
that finds the spec and the code disagree stops and says so.

## Validation Commands

```bash
env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test
make vet
cd go && env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE go test ./cmd/gdoc ./internal/presence ./internal/auth ./internal/emit ./boundary -run 'Logout|Hold|SignIn|Presence|Event|Fixture|Step|Made|Stop' -v
make app-test
```

## Progress Tracking

Nine tasks. The checkboxes live inside each task; a task is done when every
box is ticked and its commit is on the branch.

## Tasks

### Task 1: `gdoc auth logout`

**Files:**
- Create: `go/internal/auth/logout.go`, `go/internal/auth/logout_test.go`
- Modify: `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/login.go` (the auth entries), `go/cmd/gdoc/doc.go`, `go/internal/auth/doc.go`
- Modify: `go/cmd/gdoc/testdata/screens/*.golden` (the overview and the auth screens), `go/cmd/gdoc/testdata/pipe/help.stdout.golden`, `go/cmd/gdoc/pipe_test.go` (the comment names the exception), the completion goldens, `go/cmd/gdoc/presenceflag_test.go` (the two literals re-taken), `go/cmd/gdoc/presence_test.go` (the `reports` map), `go/cmd/gdoc/help_test.go` (`everyCommandName`), `go/cmd/gdoc/main_test.go`, `go/boundary` (the command count), `CLAUDE.md:24`

- [x] `TestLogoutRemovesTheTokenAndReadsItsAbsenceBack`,
      `TestLogoutWithNoTokenIsOkAndRemovesNothing`,
      `TestLogoutRefusesASymlinkOrANonRegularFileAndRemovesNothing`,
      `TestLogoutRefusesATokenItCannotParseAndLeavesIt`,
      `TestLogoutRefusesWhileASignInIsWaiting`,
      `TestLogoutTouchesNothingElseInTheConfigDir`, `TestLogoutMakesNoRequest`,
      `TestLogoutIsNoChatTool`, `TestLogoutWritesAStartAndAnEndWithNoDoc`,
      `TestOnlyLogoutRemovesTheToken` in `go/boundary`
- [x] the command, visible, with its sentence, example and `report: true`;
      the summary says the grant stays at Google until removed in the account
      settings
- [x] every pin named above re-recorded or updated for the one new command,
      and `pipe_test.go`'s comment gains the sentence that a line a milestone
      adds for a new command is re-recorded in that milestone's commit, as
      M19, M20 and M22 did
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`, `make vet`

### Task 2: a hold remembers the sign-in it was held under

**Files:**
- Create: `go/internal/auth/mark.go` (`SignInMark`, a domain-prefixed SHA-256 of the refresh token), `go/internal/auth/mark_test.go`
- Modify: `go/cmd/gdoc/mcphold.go`, `go/cmd/gdoc/mcprelease.go`, `go/cmd/gdoc/mcphold_test.go`
- Modify: `go/internal/presence/event.go` (the `dropped` hold ending), `doc.go`, `testdata/events-v1.jsonl`, `presence/Tests/PresenceCoreTests/Fixtures/events-v1.jsonl`, `presence/Sources/PresenceCore/Event.swift`, `presence/Sources/PresenceCore/Fold.swift` (the two switches learn `dropped` as a hold ending), `presence/Tests/PresenceCoreTests/EventTests.swift`, `LogTailTests.swift`

- [x] `TestAHoldRemembersTheSignInItWasHeldUnder`,
      `TestAReleaseAfterTheSignInChangedSendsNothingAndDropsTheHold`,
      `TestAReleaseAfterASignOutSendsNothingAndDropsTheHold`,
      `TestAReleaseUnderTheSameSignInStillSends`,
      `TestAnUnreadableTokenAtReleaseKeepsTheHoldAndSendsNothing`,
      `TestARotatedRefreshTokenRefusesTheRelease` (the accepted strain),
      `TestTheSweepDropsAHoldWhoseSignInChanged`,
      `TestTheSignInMarkReachesNoAnswerNoToolAndNoLine`,
      `TestEveryWayAHoldLeavesIsExactlyOneLine` gains `dropped`;
      Swift: `testADroppedLineDecodesAsAHoldEnding`
- [x] the mark kept in `mcpHolds` beside the hold and compared at release and
      in `sweep`; the fixed sentence; `internal/chat` untouched. The tests of
      the rule are in `go/cmd/gdoc/mcpsignin_test.go`, because
      `mcphold_test.go` is already at its length; the `dropped` way out joined
      `TestEveryWayAHoldLeavesIsExactlyOneLine` in `mcphold_test.go` itself.
      The new fixture line sits after `landed`, so every index the Swift tests
      read is where it was and `LogTailTests.swift` needed no change
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`, `make app-test`

### Task 3: `made` on a publish end

**Files:**
- Modify: `go/cmd/gdoc/main.go` (the end lifts `data.document_id` through `documentID` for publish), `go/internal/presence/event.go`, `doc.go`, the two fixtures, `Event.swift`, `EventTests.swift`, `LogTailTests.swift`

- [x] `TestAPublishEndNamesTheDocumentItCreated`,
      `TestARolledBackPublishNamesNoDocument`, `TestOnlyPublishCarriesMade`,
      the field-list test updated; Swift: `testMadeDecodesAndDefaultsToNil`.
      The id reaches the record through `cmd/gdoc`'s `presenceMade`, which
      names publish and puts `data.document_id` through `documentID`, and
      `(*Run).Made` carries it to the end, nil-safe like every other method
      there. The fixture's publish pair is appended, so every index the Swift
      tests read is where it was
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`, `make app-test`

### Task 4: the typed failures a stop code can name

**Files:**
- Create: `go/internal/emit/stop.go` (`StopCode` with the seven literals and `Result.Stop StopCode` tagged `json:"-"`, `Result.StopLine int` tagged `json:"-"`), `go/internal/emit/stop_test.go`
- Modify: `go/internal/body/image.go` (a typed `PictureError{Line}`), `go/internal/gapi/session.go` (`statusError` exposes its status), `go/internal/propose/*.go` and `go/internal/annotate/*.go` (a typed `ErrMoreThanOneTab`), `go/internal/guard/policy.go` (a typed refusal), their tests

- [x] `TestStopNeverReachesTheObject` (as `TestScreenNeverReachesTheObject`),
      `TestTheSevenCodesAreLiterals`, `TestAPictureThatCannotBeEmbeddedCarriesItsLine`,
      `TestAStatusErrorExposesItsStatus`, `TestMoreThanOneTabIsATypedRefusal`,
      `TestTheGuardsRefusalIsTyped`; every existing test over those error
      strings unchanged, because the words do not change. Four more boxes the
      work asked for: `TestStopCodesIsACopy`, because `StopCodes` hands out a
      slice; `TestATokenRefusedTwiceIsAStatusError`, because the auth refusal
      gapi writes itself is not a `statusError` and Task 5 has to name it;
      `TestMoreThanOneTabIsATypedRefusal` a second time in `internal/annotate`,
      because the two packages refuse separately; and `Unwrap` on
      `*PictureError` and on the two sentence-only types, which is how every
      word a person reads stayed the word it was
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`, `make vet`

### Task 5: `stop` and `line` on a failed end

**Files:**
- Modify: `go/cmd/gdoc/main.go` and the command files that build a failed result (`publish.go`, `read.go`, `write.go` or wherever each refusal becomes an `emit.Result`): each sets `Stop` from `errors.As` over the typed failures of Task 4 and `net` errors, in `cmd/gdoc` and never inside `internal/presence`
- Modify: `go/internal/presence/event.go` (`stop`, `line` lifted from `Result.Stop` and `Result.StopLine`), `doc.go`, the two fixtures, `Event.swift`, `EventTests.swift`, `LogTailTests.swift`

- [x] `TestAStopIsOneOfTheSevenCodesAndNeverASentence` (in `cmd/gdoc` over
      the reader and in `internal/presence` over the record),
      `TestTheSVGRefusalCarriesItsLine`, `TestANotFoundIsNotFound`,
      `TestAnAuthRefusalIsAuth`, `TestAHandshakeTimeoutIsNetwork`,
      `TestAGuardRefusalIsRefused`, `TestAnUnknownFailureIsOther`,
      `TestPresenceOpensNoWireAndRunsNothing` unchanged, the canary test
      extended with an error sentence that must not reach the line; Swift:
      `testStopAndLineDecodeAndDefaultToNil`. Four things the work asked for:
      `TestAStopReachesNoSuccessfulEnd`, because a caller may set a code on a
      run that worked and the record may not keep one; the word filled in by
      `dispatch`'s own deferred hook, so a command that builds a refusal by
      hand still ends with one of the seven and no command file has to
      remember it; `cmd/gdoc`'s three status literals, because that package
      may not import `net/http` (`TestNetHTTPStaysInItsRooms`); and
      `buildFailed` now taking the error rather than its sentence, which is
      how the code reaches the envelope `build` and `publish` share
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`, `make app-test`

### Task 6: `step` events for publish

**Files:**
- Modify: `go/internal/presence/event.go`, `doc.go`, `go/cmd/gdoc/publish.go`, `go/internal/publish/publish.go` (`Options.OnStep`), `go/cmd/gdoc/main.go` (`presence.WithRun` on the context), the two fixtures, `Event.swift`, `Fold.swift` (the switches skip `step` for now), `EventTests.swift`, `LogTailTests.swift`, `go/boundary/presence_test.go`

- [x] `TestAPublishStepsAreTheFixedNamesInOrder` (build, upload, read_back,
      link, each written immediately before it begins, the two network parts
      measured from the wire itself: the step is already on disk when the
      request leaves), `TestTheFirstStepsEndCarriesTheTitle`,
      `TestStepHasOnlyPublishAsCaller` (boundary),
      `TestAPanicAfterAStepStillWritesTheEnd`,
      `TestNoPercentageOrEstimateIsEverWritten`; Swift:
      `testAStepLineDecodesWithItsNameIndexAndCount`. Three more boxes the work
      asked for: `TestAStepIsItsNameAndItsPlaceInTheFixedList`, because the
      four names and the two numbers are the record's own shape;
      `TestAStepNameThatIsNotOneOfTheFourWritesNothing`, because
      `internal/publish` names two of the four and a drift there must write no
      line rather than a word nobody has seen; and
      `TestAStepOutsideARunWritesNothing`, which is the flag being off and a
      context nobody marked. `presence.Step` reads the run off the context,
      which `dispatch` puts there, so no room between `cmd/gdoc` and the work
      is handed a run; `Fold.swift` skips `step` and keys it by the part's own
      name, so M24a's job card loses none of the four
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`, `make app-test`

### Task 7: `via: code` and `interrupted`

**Files:**
- Modify: `go/internal/presence/event.go`, `doc.go`, `go/cmd/gdoc/main.go`, `go/cmd/gdoc/mcp.go` (chat wins over code), the two fixtures, `Event.swift`, `EventTests.swift`, `LogTailTests.swift`, `docs/v2/MEASURED.md`

- [x] measured and recorded: Claude Code's Bash tool hands a child
      `CLAUDECODE=1`, beside a dozen `CLAUDE_CODE_*` variables including an
      email and two token-shaped ones, which is why `CLAUDECODE` is the one
      gdoc reads. The section and its method are in `docs/v2/MEASURED.md`,
      "What Claude Code puts in a child's environment".
      `TestViaIsCodeUnderClaudeCode`, `TestViaChatWinsOverCodeInTheMcpSession`,
      `TestAnInterruptedWaitSaysSo` (from `data.waited.interrupted`); Swift:
      `testViaCodeAndInterruptedDecode`. Five more boxes the work asked for:
      any value of the variable counts and not the literal `1`, because the
      variable is Claude Code's own; `presence.WithCode` leaves a context the
      chat session marked alone, so the mcp session needs no new line and
      `TestChatWinsOverCode` states the rule over the context itself;
      `cmd/gdoc` reads the environment and `internal/presence` still reads
      none, so the record stays a file and a clock; `recording(t)` clears the
      variable, because the suite is run from a Claude Code session and what a
      run records may not depend on who started the test; and
      `TestAStoppedWaitsEndSaysSo` over a real `comments --wait` somebody
      stopped, because the word is lifted from the command's own answer and
      nothing else in the binary knows a wait ended early
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`, `make app-test`

### Task 8: the words

**Files:**
- Modify: `go/internal/presence/doc.go`, `go/internal/emit/emit.go` (the package comment names `Stop`), `go/cmd/gdoc/doc.go`, `docs/guide/presence.md` (the logout paragraph and the hold's sign-in rule, two paragraphs appended at the end so M24a's edits merge cleanly)

- [x] the doc comments name every new field and kind with its test; the
      guide says what logout does and does not do, and that a held write dies
      with the sign-in it was held under; CLAUDE.md is not touched beyond
      Task 1's one word; `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`.
      `internal/presence`'s own doc.go named each field as its task landed, so
      this task adds the kind list to its opening paragraph, names `Stop` and
      `StopLine` in `internal/emit`'s package comment beside `Screen`, and
      gives `cmd/gdoc/doc.go` the two sections it had none of: the five facts
      of the record this layer decides, and the hold that dies with its
      sign-in, with the accepted rotation strain written down there. The
      guide's paragraphs are appended under one new heading at the end of
      `docs/guide/presence.md`

### Task 9: the acceptance rows and the close

**Files:**
- Modify: `docs/plans/2026-10-17-gdoc-v2-m24-the-panel-spec.md`, `docs/v2/PLAN.md`

- [x] the Go rows of the spec's acceptance table, under a heading "M24b",
      pass or the test that says otherwise. Every row passes: the section is
      one row per clause of the three rules this half owns, each naming its
      test, and it closes by saying which measurements are not this half's
- [x] a new M24 row in `docs/v2/PLAN.md`'s Done table with the M24b half
      written and the M24a half left as "running"
- [x] this plan moves to `docs/plans/completed/`; the spec stays until M24a
      closes, which moves it. The spec's own opening paragraph now names this
      plan at its new path, because that reference would otherwise be stale
      while the spec waits for M24a

## Post-completion

**Merging.** This half merges first; M24a then merges `main` and teaches its
scenes the new fields in its own last task.

**Release.** `make tag VERSION=v2.15.0` after both halves merged.
