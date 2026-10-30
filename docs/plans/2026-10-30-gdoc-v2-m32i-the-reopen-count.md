# gdoc v2 Milestone 32i: the reopen count. Plan

Tasks: 5
Review: ui

2026-10-30. Nail pressed Update to v2.29.0 in the panel and wrote: "when i
click update you show then text, quit and reopen, i expect timer where,
which indicate that to be reopenned". The record and the app's own stamp
say what happened on his machine: `gdoc update` ended at 08:30:05Z, the
relaunch stamp was written at 08:30:10Z, and the process running since
carries `--relaunched-by <pid>`. The app did reopen itself, about seven
seconds after the update ended, and for those seconds the panel said
"Updated to v2.29.0. Quit and reopen", which promises a manual act the app
was about to do itself. The decision is `docs/v2/DECISIONS.md`, 2026-10-30,
"The reopen count", with its register row, and the spec sentence in the
panel paragraph of `docs/v2/SPEC.md`, both on `main` before this run. The
change is words, a count the app already waits out, and the order of two
steps that exist: a Swift change and no Go change. One commit per task.
Where the code, the decision and this plan disagree, the task stops and
says so.

## Principles

Serves 4 of `PRINCIPLES.md` (every word costs a reader's attention: the
row says what the app is doing, in the words a person reads at a glance,
and never asks for a hand the app does not need) and 3 (uncertainty never
resolves toward the destructive answer: every guard of the relaunch
policy stays as it is, the policy is asked before the count begins and
again before the open, and a count is a wait and not a new door). Serves
principle 5 of `docs/design/2026-10-17-presence-design-principles.md`,
facts and not promises: "Quit and reopen" was a promise the app broke
seven seconds later, and the count is a clock the app itself runs and acts
on at zero, which is the one form the sign-in's countdown was allowed in
(`Facts.swift:39` to 41: "the child's three minutes and never a guess").

Strains the design checklist's line "nothing predicts a time" (the
principles file, lines 83 to 84) in the one way the sign-in countdown
already strains it, and Nail asked for it in those words. No invariant in
`CLAUDE.md` moves: the argv set stays six, `Process` stays in the Runner,
`openApplication` stays in the Relauncher, and no read, grant, URL or
presence field changes. The decision entry says the trust guardian is not
called, and this plan calls nobody.

## Context (from discovery)

Checked against `main` at 57db8f6, with the decision entry and the spec
sentence on the lead's working tree (`git status` lists `DECISIONS.md` and
`SPEC.md` modified). M32h, "the mark, sharp and a press"
(`docs/plans/2026-10-30-gdoc-v2-m32h-the-mark-sharp-and-a-press.md`,
`DECISIONS.md:1815`), is drafted beside this plan and runs first. It
touches `HealthPoller.swift`, `UpdateAction.swift`, `Facts.swift`,
`PanelModel.swift`, `PanelWords.swift`, `StateTests.swift`,
`SessionTests.swift`, `Scenes.swift`, `Shots.swift`, `RenderTests.swift`,
`docs/guide/presence.md`, `go/boundary/guide_test.go` and the principles
file, which are files Tasks 2 to 5 touch. So this run is cut from the
`main` M32h merged to, and Task 5's amendment goes after M32h's. The line
numbers below were read today, before that merge, and may move by a few
lines.

- **The press path.** `SessionActions.swift:59` to 60, `.update:
  install()`; `install()` at lines 303 to 321 sets `installing = true`,
  calls `retick()`, awaits `updates.update()`, then sets `installing =
  false`, calls `retick()`, `recompose()` and `ask(.changed)`, which runs
  the two polls, `help --json` and `auth status`, as children.
  `UpdateAction.update()` is `UpdateAction.swift:123` to 127 and `ran()`
  at 217 to 233 answers `.installed`, `.unchanged` or `.failed`. The argv
  lists are `Runner.swift:33` to 34 and 42 to 43, `["update"]` and
  `["update", "--desktop"]`, and nothing here adds one.
- **The two intervals.** `Session.swift:33`, `refoldEvery = 5`, and 38,
  `whileAChildIsUp = 1`, whose comment names "the ring and the sign-in
  countdown" as the two things on the screen that move. `retick()` at
  511, `signingIn || installing ? whileAChildIsUp : refoldEvery`, and the
  timer is recreated only when the interval changes. `tick()` at 520 to
  525 refolds, reads the sign-in phase and calls `reaskTheReopen()` at
  532 to 537, which asks again only when `reopenHeld` is up and neither
  guard is.
- **The row today.** `PanelModel.swift:713` to 722, `notice(_:)`: line
  716 is `case .reopenNeeded(let version): return "Updated to \(version).
  Quit and reopen"` and 717 the removed bundle. `PanelView.swift:60` to
  63 draws it as the exclamation row in the `.hold` tone above every
  section, probed as `"notice"`, whenever `model.notice` is set. The
  version in it is the plist's own string, `2.29.0`, where the header
  writes `v2.14.0` (`BundleVersion.check`, `BundleVersion.swift:33` to
  36; `RenderTests.swift:398`); the lead decided the row writes it as the
  header does. `Facts.bundle` is `Facts.swift:266` to 268, default
  `.same` in the init at 313. The sentences that belong to no row live in
  `Sentences`, `PanelWords.swift:84`. Two rooms rebuild `Facts` field by
  field and would drop a new field to its default:
  `HealthPoller.applied(to:)` (`HealthPoller.swift:178` to 207, `bundle:
  facts.bundle` at 201) and `UpdateAction.applied(to:)`
  (`UpdateAction.swift:162` to 192, `bundle: facts.bundle` at 184), which
  `Session.recompose()` runs one after the other (445 to 446).
  `BundleVersion.swift:10` to 11 still says "This milestone never
  relaunches itself" and 19 to 20 still describes the row as "Quit and
  reopen"; `BundleVersionTests.swift:19` carries the same stale message.
- **The watch and the ask.** `BundleWatch.swift:21`, `bundleDebounce =
  0.75`; `start` at 36 to 40 watches the bundle and its parent; `settle()`
  at 76 to 86 reads the plist once per burst and hands
  `BundleVersion.check` to the session. `Session.swift:242` builds the
  watcher and 272 starts it into `bundleMoved(to:)`, lines 860 to 868,
  which stores the state, recomposes and calls
  `reopenIfTheBundleMoved()`, lines 872 to 885: it asks the relauncher
  with `updateIsRunning()` (896 to 899, the fold's jobs contain `update`,
  which a terminal's `gdoc update` writes as much as the app's own) and
  `aChildIsUp()` (903 to 905, any of the six calls in flight, so a poll
  counts), and sets `reopenHeld` when the answer was a hold on one of
  those two. `reopenHeld` and its comment are lines 87 to 97. Both
  `bundleMoved` and `reopenIfTheBundleMoved` are private; `tickingEvery`
  at 181 and `tookThePhase` at 562 are internal, which is the shape a
  test reaches in. A session built with no bundle (`swift run`, and every
  `SessionTests.made`) has no relauncher (229 to 233) and no watcher.
- **The relauncher.** `Relauncher.swift:114` to 137, `reopen()`: builds
  the `RelaunchAsk`, calls `RelaunchPolicy.decide`, writes the stamp with
  `preferences.noteRelaunch` (132), `await sleep(after)` at 133, opens
  the bundle at 134 to 135 with `[RelaunchPolicy.argument, String(pid)]`,
  and answers `RelaunchPolicy.after(opened:)`. Its file comment at 6 to
  7 says "waits the two seconds". `SystemLaunchpad` at 43 to 69:
  `createsNewApplicationInstance = true`, `activates = false`. The new
  instance's `settle()` at 145 to 157 waits up to the patience for the
  launching pid, asking every `asksEvery` (164, 0.25 s), through the same
  `sleep` closure, so the closure stays. `Lock.swift:133` to 135 gives a
  relaunched instance that patience for the lock. The session builds it
  with the default readers, `identifierRead` (168) and
  `signatureChecked` (180), and the `pad` the init takes at 197.
- **The policy.** `RelaunchPolicy.swift:133`, `settle = 2`, with the
  comment "so the rename that changed the bundle has finished arriving";
  136 `once = 600`; 140 `patience = 5`; 144 the argument.
  `RelaunchDecision` at 101 to 107, whose `.relaunch` comment at 103 to
  104 says "after this many seconds". `decide` at 153 to 171 in the order
  same, gone, update running, child alive, another identifier, invalid
  signature, already tried, then `.relaunch(version:, after: settle)`;
  it writes nothing, so it can be asked as often as the session likes.
  `after(opened:)` at 174 to 176. The `keepTheReopenRow` comment at 113
  to 115 still says SETTINGS says quit and reopen; the row has been the
  notice above every section since the second morning. The stamp is
  `Preferences.swift:220` and 236.
- **The tests that hold the shape.** `SourceRulesTests.swift:104`
  (`Process` only in the Runner), 127 (the argv enum closed at six), 245
  (the system asked only through the opener and the relauncher), 283
  (only the relauncher opens an application); `RunnerTests.swift:62`.
  `RelaunchPolicyTests.swift:19` asserts `.relaunch(version: onDisk,
  after: 2)` at 24, 36 and 42 and `RelaunchPolicy.settle == 2` at 46; 49,
  69, 88, 108 and 130 are the other rows of the table.
  `RelauncherTests.swift:23` and 64 hold the held-then-opened route with
  a `FakePad` (98 to 117, which always answers true) and `sleep: { _ in
  }` (94); its file comment at 5 to 6 says "waits two / seconds" across
  a line break. `SessionTests.made` at 1335 to 1363 builds every session
  with `bundle: nil` and `identifier: ""`; `Session.init` (192 to 207)
  already takes `pad:`, `clock:` and `quit:`. The facts land through two
  actors, so a session test waits on `refreshed(_:)` (1367 to 1374,
  which awaits `recompose()`) or an expectation, and `answered(_:)`
  (1378 to 1387) is what starts the session and its watcher.
  `testTheCheckIsOnTheScreenBeforeTheTwoPollsComeBack` at 287 is the
  shape of a session test that presses Update.
  `HealthPollerTests.testThePolledFactsReachTheFacts` (117) and
  `UpdateActionTests` (23 onward) are where a rebuilt field is pinned.
- **The model's tests and the words.** `StateTests.drawn(...)` at 1028
  to 1037 is `PanelModel.build(events:facts:ui:now:calendar:alive:)`;
  `testTheModelCarriesTheLogoAndOnlyFromTheFacts` at 1015 is the shape of
  a facts-to-model test. `WordsTests.testNothingPredictsATime` at 189 to
  230 holds its needles spelled in pieces and scans `Words.swift` alone,
  not `PanelWords.swift`. The sign-in countdown is
  `PanelHeaderModel.swift:184` to 192, "2:43 left of the three minutes
  gdoc waits", and `Scenes.stillYourCall` (`Scenes.swift:154` to 156)
  calls it "the one number in the panel about something that has not
  happened", which stops being true today; `RenderTests.swift:606` pins
  that list at seven and 619 to 623 its subjects, "decision 39" among
  them. The index page's lead is `RenderTests.swift:755` to 779, with
  `markRule` printed at 762. Running update on the header:
  `versionWaiting` true and `update` nil (`StateTests.swift:199` to 202).
- **Scenes and renders.** `Scene` is `Scenes.swift:32` to 51;
  `machine(...)` at 259 to 299 has no `bundle` parameter, so no scene has
  ever drawn the reopen row, and it hard-codes the app at `2.14.0` and
  gdoc at `v2.14.0`; the ids at 302 to 526 skip e4 and e7, which were
  retired and are not reused, and run in the order of their day. e5 (469
  to 485) is a scene with no fixture, e8 (503 to 514) is the countdown
  scene. `RenderTests.swift:54` pins `Scenes.all.count` at 18.
  `panelVocabulary` (`Shots.swift:487` to 492) takes `model.notice` into
  the vocabulary, so a sentence a scene draws is a sentence a sheet may
  use; no sheet draws the notice today (`Scenes.sheets`, 203 to 237).
  Renders land under `$PRESENCE_SHOTS`. `docs/team/README.md:59` and 99
  pin the design reviewer's reading at 18 dark whole-panel images.
- **The documents.** `DECISIONS.md:1866` to 1911 is the entry, 121 its
  register row, 1009 to 1013 the panel entry it builds on.
  `SPEC.md:146` to 151 the sentence. `MEASURED.md:284` to 345: the swap
  keeps the process alive and it must reopen its own path, not re-exec.
  The M24 plan, `docs/plans/completed/2026-10-17-gdoc-v2-m24-the-panel-spec.md:91`,
  "waits two seconds ... A failed open leaves the reopen row", stays as
  the record of its day. `docs/guide/presence.md:406` to 418 ("A run that
  installed something relaunches the app") and 602 to 612 ("it waits two
  seconds", "`Updated to <version>. Quit and reopen`"); the strings
  `TestThePresencePageWalksTheThirdLooksPanel` requires are
  `go/boundary/guide_test.go:200` to 240 and none of them is in those two
  passages. The principles file's last heading is "## Amendments,
  2026-10-29, the firm's mark beside DOCUMENTS" at 771, and M32h's goes
  after it. `CLAUDE.md` names no sentence of the row.

### The design

- **The count is pure and lives in PresenceCore.** `ReopenCount` is an
  enum of four states: `.held`, `.counting(began: Date)`, `.opening` and
  `.refused`. One function, `asked(_ gate: ReopenGate, now: Date)`, hands
  back the moved value, what the row says (`ReopenStanding`: `.held`,
  `.counting(Int)` from 5 to 1, `.refused`) and whether this is the one
  moment to open. The gate is what the session learned on this ask, and
  it comes from the policy: `.clear` when `RelaunchPolicy.decide` says
  `.relaunch`, `.held` when it holds on `.updateRunning` or
  `.childAlive`, the two guards that clear by themselves, and `.refused`
  on any other hold. Asking the policy writes nothing, so it is asked on
  every tick while a count is pending and once more before the open.
- **The transitions.** `.held` stays held on a held gate, begins
  counting on a clear one, and goes refused on a refused one. A count
  under way keeps counting through a held gate before five seconds, so
  a poll child of a few hundred milliseconds does not move the row; at
  five seconds a clear gate opens, once, and moves the count to
  `.opening`, while a held gate at five seconds sends it back to `.held`
  and the count restarts from 5 when the gate clears. `.opening` answers
  `opensNow == false` to every later ask and keeps "Reopens in 1" on the
  row until the old instance quits; the session turns it into `.refused`
  only when the relauncher answered that the open did not happen.
  `.refused` is for good. The seconds on the row are clamped to 1 to 5,
  so a clock that goes back reads 5 and never 6. A session with no
  relauncher refuses at once, as a `swift run` build does today.
- **The count is the settle.** `RelaunchDecision.relaunch` loses its
  `after`, `RelaunchPolicy.settle` goes, and `reopen()` no longer sleeps:
  the open is asked for at zero. Five seconds after the child ended is
  longer than the two the settle gave, on both routes: the app's own
  press, where the child's end is after the swap, and a terminal's `gdoc
  update`, whose record holds the gate until its end line. The stamp,
  the ten minutes per on-disk version, the identifier, the signature and
  the new instance's own wait do not move.
- **The facts carry the standing beside the bundle.** `Facts.reopen:
  ReopenStanding`, default `.held`, read only under `.reopenNeeded`;
  `BundleState` stays the disk's fact and `BundleVersion.check`,
  `RelaunchAsk` and every test over them keep their shape. The two
  field-by-field rebuilds carry it through. The model's `notice`
  switches on both and writes the version as the header does: "Updated
  to v2.29.0. Reopens in 5".
- **One ask at a time in the session.** The session keeps one
  `ReopenCount` and a generation number in the place of `reopenHeld`;
  `bundleMoved` resets both, and an answer that comes back from a Task
  started under an older generation is dropped, so a relaunch asked for
  before a rollback cannot mark the new count refused. The `.opening`
  state is set synchronously before the Task starts, so the next tick,
  which may land before the open answers, opens nothing. The tick is one
  second while a count is pending, which includes a hold whose reason is
  a terminal's `gdoc update` live in the record.

### What the lead changes on `main` before the run

Done before the launch: the decision entry with its register row and the
spec sentence, committed, and M32h merged. Nothing else.

### Out of scope

- A count on Install extension; a ring or bar for the count; a count
  under five.
- Any guard of `RelaunchPolicy.decide`, the stamp, the ten minutes, the
  new instance's patience, `Lock`, or `BundleWatch`'s debounce.
- A new argv list, a `Process`, a second caller of `openApplication`, a
  new read or file.
- The Go binary, `gdoc update` itself, and the record.
- The sheets: the two new scenes carry the two sentences and the sheets
  draw no notice.
- The M24 plan's text, which stays as the record of its day.

## Development Approach

- TDD: the failing test first, in the file the Context names, then the
  smallest change that passes it.
- One commit per task, message in the house form, no attribution lines.
- Nothing is loosened: the source rules, the argv count, the policy's
  table and the scene pin each move only with its reason, and every new
  rule gets its own test.
- No Go source changes. The one Go test run is the boundary test over the
  guide, after Task 5's edit.
- Plain English in every comment, caption and sentence, no em dashes, and
  none of the forecast words `WordsTests.testNothingPredictsATime` bans.
- A boolean is `is`, `has`, `should` or `can`: `isRefused`, never
  `refused` beside a `refused()`.

## Validation Commands

```
make app-lint
make app-test
make app
```

Task 4 only:

```
PRESENCE_SHOTS="$HOME/gdoc-m32i-board/shots" make app-test
```

Task 5 only:

```
cd go && go test ./boundary/ -run 'TestThePresencePageWalksTheThirdLooksPanel|TestCLAUDEmd|TestTheTaskMap'
```

## Progress Tracking

Tick each box as it is done. A task ends with its commit.

## Tasks

### Task 1: the count, pure, in PresenceCore

**Files**

- `presence/Sources/PresenceCore/ReopenCount.swift` (new)
- `presence/Sources/PresenceCore/RelaunchPolicy.swift`
- `presence/Sources/PresenceCore/BundleVersion.swift`
- `presence/Sources/PresenceApp/Relauncher.swift`
- `presence/Tests/PresenceCoreTests/ReopenCountTests.swift` (new)
- `presence/Tests/PresenceCoreTests/RelaunchPolicyTests.swift`
- `presence/Tests/PresenceCoreTests/BundleVersionTests.swift`
- `presence/Tests/PresenceAppTests/RelauncherTests.swift`

**Steps**

- [ ] `ReopenCountTests.swift`, new, six tests over a fixed `Date`, `t0`,
      and offsets from it. Red, because the type does not exist.
      `testTheCountBeginsOnTheFirstClearGateAndNotBefore`:
      `ReopenCount.held.asked(.held, now: t0)` answers standing `.held`,
      `opensNow` false and `.held`; the same at `t0 + 30` is still
      `.held`; the first `asked(.clear, now: t1)` answers `.counting(5)`,
      `opensNow` false, and `.counting(began: t1)`.
      `testTheCountReadsFiveDownToOneAndOpensOnceAtFive`: from
      `.counting(began: t1)` with a clear gate, the standing at `+0` and
      `+0.99` is `.counting(5)`; `+1` `.counting(4)`; `+2` `.counting(3)`;
      `+3` `.counting(2)`; `+4` and `+4.99` `.counting(1)` with
      `opensNow` false; `+5` is `.counting(1)` with `opensNow` true and
      the value `.opening`; the next ask from `.opening` at `+5.5`, `+6`
      and `+60` is `.counting(1)` with `opensNow` false and still
      `.opening`, whatever the gate. One open per count is this row.
      `testAHeldGateBeforeFiveSecondsKeepsCountingAndAtFiveGoesBackToHeld`:
      from `.counting(began: t1)`, `asked(.held, now: t1 + 2)` is
      `.counting(3)`, value unchanged; `asked(.held, now: t1 + 5)` is
      standing `.held`, `opensNow` false, value `.held`; then
      `asked(.clear, now: t1 + 9)` is `.counting(5)` with `began` at
      `t1 + 9`: the count restarts and never freezes at 1.
      `testARefusedGateRefusesForGoodFromAnyState`: `.held`,
      `.counting(began: t1)` and `.opening` each answer `.refused` to a
      refused gate, and `.refused` answers `.refused` with `opensNow`
      false to `.clear`, `.held` and `.refused` at any moment;
      `ReopenCount.refused.isRefused` is true and the other three are
      false.
      `testTheRowIsClampedToOneThroughFive`: from `.counting(began:
      t1)`, an ask at `t1 - 3` (a clock that went back) reads
      `.counting(5)`, and at `t1 + 40` from a count that was never
      opened (a held gate at 40) the standing is `.held` and not a
      negative number.
      `testTheCountIsFiveSeconds`: `ReopenCount.seconds` is the literal
      5.
- [ ] `ReopenCount.swift`, new, with a file comment in the house voice:
      what the count is (the five seconds between an update that
      replaced the bundle and the open, which the person reads down on
      the notice row), why it is pure (the session asks the policy and
      the clock, this decides, and every answer is reachable from a
      test), why a held gate during the count does not restart it (a
      poll child is under a second and a row that jumped back to "when
      the update finishes" would be the panel losing its place), why a
      held gate at five does restart it (the bytes may still be moving,
      and a 1 that waited would be a promise), why `.opening` is a state
      (the open is asked for in a Task and the next tick lands before it
      answers), and that it is the settle the relauncher used to sleep,
      which is why there is no two-second wait anywhere else. `public
      enum ReopenGate: Sendable, Equatable { case clear, held, refused
      }`, `public enum ReopenStanding: Sendable, Equatable { case held;
      case counting(Int); case refused }`, and `public enum ReopenCount:
      Sendable, Equatable { case held; case counting(began: Date); case
      opening; case refused }` with `public static let seconds = 5`,
      `public var isRefused: Bool`, `public struct Step: Equatable {
      count, standing, opensNow }` and `public func asked(_ gate:
      ReopenGate, now: Date) -> Step`. The seconds on the row are
      `min(seconds, max(1, seconds - Int(elapsed)))`. New values are
      returned, nothing is mutated. Green.
- [ ] `RelaunchPolicyTests.testRelaunchHappensOncePerOnDiskVersionInTenMinutes`:
      the three `.relaunch(version: onDisk, after: 2)` at lines 24, 36
      and 42 become `.relaunch(version: onDisk)`, and line 46's
      `RelaunchPolicy.settle` assertion goes, with a comment that the
      wait is `ReopenCount.seconds` now and `ReopenCountTests` pins it.
      Red.
- [ ] `RelaunchPolicy.swift`: `case relaunch(version: String)` with its
      comment at 103 to 104 no longer saying "after this many seconds";
      `settle` and its comment at 131 to 133 go; `decide` answers
      `.relaunch(version: version)` at 169; the file comment's "waits the
      two seconds" and the `keepTheReopenRow` comment at 113 to 115 now
      say the notice row says quit and reopen, and that the wait before
      the open is `ReopenCount` in the session. Green in PresenceCore;
      `Relauncher.swift:128` no longer compiles.
- [ ] `RelauncherTests.testTheOpenHappensOnTheAskWithNoSleepBeforeIt`:
      `made(pad:)` gains a `sleep` closure that appends every interval
      to a list; a `reopen(bundle: .reopenNeeded("2.16.0"),
      updateRunning: false, childAlive: false, remembering:)` answers
      `.quit`, the pad opened once with `[RelaunchPolicy.argument,
      "4242"]`, and the list is empty: the count in the session is the
      settle, and the open is asked for at zero and not two seconds
      later. Red once the file compiles again.
- [ ] `Relauncher.swift:128` to 133: bind `version` alone, no sleep
      before the open; the file comment at 6 to 7 says the session
      counts five seconds down on the notice row and asks for the open
      at zero. `sleep` stays, for `settle()`. `RelauncherTests.swift:5`
      to 6 no longer says "waits two seconds": the fake sleep is there
      for the new instance's wait. Green, and the two existing tests
      stay green.
- [ ] `BundleVersion.swift:10` to 11: the two sentences about never
      relaunching go, replaced by one that says the session relaunches
      the app on `.reopenNeeded` once `ReopenCount` reaches zero, and
      that the row's words are `PanelModel.notice`'s, by
      `Facts.reopen`. Lines 19 to 20 name the three sentences. The
      message at `BundleVersionTests.swift:19` says "the app reopens
      itself once the count is down; the row says so" instead of "this
      milestone never relaunches itself". No assertion changes.
- [ ] `make app-lint` and `make app-test` green.
- [ ] Commit: `feat: the reopen count, pure, in PresenceCore`.

### Task 2: the facts carry the standing, and the row says it

**Files**

- `presence/Sources/PresenceUI/Facts.swift`
- `presence/Sources/PresenceUI/PanelWords.swift`
- `presence/Sources/PresenceUI/PanelModel.swift`
- `presence/Sources/PresenceApp/HealthPoller.swift`
- `presence/Sources/PresenceApp/UpdateAction.swift`
- `presence/Tests/PresenceUITests/StateTests.swift`
- `presence/Tests/PresenceAppTests/HealthPollerTests.swift`
- `presence/Tests/PresenceAppTests/UpdateActionTests.swift`

**Steps**

- [ ] In `StateTests.swift`, `testTheNoticeRowSaysTheThreeReopenSentences`,
      with `drawn([], at:, facts:)` over `Facts(versions:
      Versions(app: "2.15.0"), gdocPath: Scenes.gdocPath, bundle:
      .reopenNeeded("2.16.0"), reopen: ...)`: `.counting(5)` draws
      `notice == "Updated to v2.16.0. Reopens in 5"`; `.counting(1)`
      draws "Updated to v2.16.0. Reopens in 1"; `.held` draws "Updated
      to v2.16.0. Reopens when the update finishes"; `.refused` draws
      "Updated to v2.16.0. Quit and reopen"; the `v` is the row writing
      the version as the header does, from the plist's bare string, and
      a plist that already says `v2.16.0` is not doubled; `bundle: .same`
      with `reopen: .counting(3)` draws no notice, because the standing
      is read only under `.reopenNeeded`; `.gone` still draws "gdoc
      presence was removed"; and the init with no `reopen` is `.held`.
      Red.
- [ ] `testTheReopenSentencesCarryNoForecastWord`: the four sentences
      above, lowercased, carry none of the needles
      `WordsTests.testNothingPredictsATime` bans, spelled in pieces the
      same way, so "Reopens in 5" is a count the app runs and not a
      forecast in words. Red until the sentences exist; it stays as the
      pin on their wording.
- [ ] `Facts.swift`: `public let reopen: ReopenStanding`, placed after
      `bundle`, default `.held` in the init, with the comment: what the
      app is doing about a `.reopenNeeded` bundle, held, counting or
      refused; it means nothing under `.same` or `.gone`, and the
      default is held because that is what the first ask answers.
- [ ] `PanelWords.swift`, in `Sentences`: `reopensIn(_ version: String,
      _ seconds: Int)`, `reopensAfterTheUpdate(_ version: String)` and
      `quitAndReopen(_ version: String)`, each one line, the three
      literals above through one private `versionWord` that puts the
      header's `v` in front of a bare plist string, with the comment
      that the first two are the decision of 2026-10-30 and the third is
      the row since M23, kept for the one case the reopen did not
      happen.
- [ ] `PanelModel.notice` at 713: under `.reopenNeeded(let version)`,
      switch on `facts.reopen` and return the three sentences through
      `Sentences`. The comment above the newer-record line stays. Green.
- [ ] `HealthPollerTests.testTheReopenStandingSurvivesThePollsRebuild`:
      `applied(to:)` over facts with `bundle: .reopenNeeded("2.16.0"),
      reopen: .counting(3)` keeps both. Red, then `HealthPoller.swift:201`
      gains `reopen: facts.reopen` beside `bundle`. Green.
      `UpdateActionTests.testTheReopenStandingSurvivesTheUpdateActionsRebuild`:
      the same over `UpdateAction.applied(to:)`. Red, then
      `UpdateAction.swift:184` gains it. Green.
- [ ] `make app-lint` and `make app-test` green. The scenes still build
      because `reopen` defaults.
- [ ] Commit: `feat: the notice row says what the app is doing about the reopen`.

### Task 3: the session counts on the tick and the relauncher opens at zero

**Files**

- `presence/Sources/PresenceApp/Relauncher.swift`
- `presence/Sources/PresenceApp/Session.swift`
- `presence/Tests/PresenceAppTests/RelauncherTests.swift`
- `presence/Tests/PresenceAppTests/SessionTests.swift`

**Steps**

- [ ] `RelauncherTests.testDecidingWritesNoStampAndOpensNothing`: a new
      `relauncher.decides(bundle:updateRunning:childAlive:remembering:)`
      answers `.relaunch(version: "2.16.0")` for a clear ask,
      `.hold(.updateRunning)` and `.hold(.childAlive)` for the two
      clearing guards, `.hold(.invalidSignature)` for a relauncher made
      with `signatureValid: { _ in false }`, and after ten of them the
      pad was asked nothing and `preferences.lastRelaunch()` is nil.
      `testAFailedOpenKeepsTheRowAndWritesTheStamp`: `FakePad` gains a
      settable `opens: Bool`, default true; with it false, `reopen(...)`
      answers `.keepTheReopenRow`, the pad was asked once, and
      `preferences.lastRelaunch()?.version` is "2.16.0", so the ten
      minutes count from the attempt. Red on the first; keep the second
      as the pin the session's refused test leans on.
- [ ] `Relauncher.swift`: `reopen()`'s ask-building at 118 to 127 moves
      into a private `ask(...)` both methods call; `decides(...)` returns
      `RelaunchPolicy.decide(ask)` and touches nothing; `reopen()` keeps
      its stamp and open. Green.
- [ ] `SessionTests.made(...)` gains `bundle: URL? = nil, identifier:
      String = "", pad: Launchpad = SystemLaunchpad(), signatureValid:
      @escaping @Sendable (URL) -> Bool = { _ in true }`, and a helper
      `writeTheBundle(version:)` that writes `<root>/GdocPresence.app/
      Contents/Info.plist` naming `CFBundleIdentifier`
      `com.altery.gdoc.presence` and `CFBundleShortVersionString` as
      given, called before `answered(session)` starts the watcher, so
      `Relauncher.identifierRead` reads a real plist and the signature
      is this test's to say. `Session.init` gains `signatureValid:` with
      the default `Relauncher.signatureChecked`, handed to the
      `Relauncher` at 229 to 233 beside `pad`, the way `alive` and
      `clock` are handed in; `bundleMoved(to:)` and the per-tick
      `askTheReopen()` below become internal, so a test drives the clock
      instead of waiting on a timer. `FakePad` is copied into
      `SessionTests` with the `opens` switch, as `FakeDesk` is this
      file's own.
- [ ] `testTheReopenCountsFiveToOneOnTheTickAndOpensOnce`: a session
      over a bundle written at "2.15.0" and a plist rewritten to
      "2.16.0" before `answered`, a movable `now`, a `FakePad`, a `quit`
      closure counting its calls. The test calls
      `session.bundleMoved(to: .reopenNeeded("2.16.0"))`, then
      `refreshed(session)`, and asserts `session.facts.reopen ==
      .counting(5)`, `session.model.notice == "Updated to v2.16.0.
      Reopens in 5"`, `session.tickingEvery == 1` and the pad not asked.
      It moves `now` by one second at a time, calls
      `session.askTheReopen()` and `refreshed(session)`: 4, 3, 2, 1 on
      the facts, no open yet. At five seconds it asks twice in a row
      without moving the clock and waits on an expectation the quit
      closure fulfils: the pad was asked exactly once, with the argument
      and the session's own pid, `preferences.lastRelaunch()?.version`
      is "2.16.0", quit was called once, and `facts.reopen` is still
      `.counting(1)`, never `.refused`. Red.
- [ ] `testTheReopenIsHeldWhileTheUpdateIsLiveAndCountsFromItsEnd`: the
      record holds a `start` of `update` with no end for a pid the
      session's `alive` closure says is alive, through a `var` the test
      flips. `bundleMoved(to: .reopenNeeded("2.16.0"))` and `refreshed`
      leave `facts.reopen == .held`, the notice "Updated to v2.16.0.
      Reopens when the update finishes", `tickingEvery == 1` and the pad
      not asked; three seconds of `askTheReopen()` change none of that.
      The test flips `alive` to false and asks: `.counting(5)` from that
      ask's `now`, and five seconds later the one open.
      `testAGuardUpAtFiveSendsTheCountBackToHeldAndItRestarts`: the
      same session; the count runs to 4, then the record gains a live
      `update` start, the ask at five seconds leaves `.held` on the
      facts and the pad not asked; the start's end is written, and the
      next ask is `.counting(5)` again, five seconds later the one open.
      Red.
- [ ] `testAReopenRefusedForGoodSaysQuitAndReopenAndTheTickGoesBackToFive`:
      the same session with `signatureValid: { _ in false }`.
      `bundleMoved` goes straight to `facts.reopen == .refused`, the
      notice "Updated to v2.16.0. Quit and reopen", `tickingEvery == 5`,
      no open and no stamp: the policy was asked before the count began
      and held for good. A second session with the pad's `opens` false
      counts 5 to 1, the pad was asked once, the stamp is written, and
      the facts read `.refused` with the same sentence. A third session
      made with `bundle: nil`, as every other test's is, reads
      `.refused` on `bundleMoved` at once, because there is no
      relauncher. Red.
- [ ] `testAnAnswerFromBeforeTheBundleMovedAgainIsDropped`: the pad's
      `openNewInstance` waits on a continuation the test holds; the count
      runs to five and the open is asked for; the test calls
      `bundleMoved(to: .same)` and then `.reopenNeeded("2.17.0")`, lets
      the held open answer false, and asserts the new count is
      `.counting(5)` and not `.refused`. Red.
- [ ] `Session.swift`: `var reopen = ReopenCount.held`, `var reopenWord
      = ReopenStanding.held` and `var reopenAsk = 0` take the place of
      `reopenHeld` (87 to 97), with a comment that says what the count
      is, that the tick is one second while it runs, including while the
      hold is a terminal's `gdoc update` live in the record, that the
      generation drops an answer from before the bundle moved again, and
      names `testTheReopenCountsFiveToOneOnTheTickAndOpensOnce`.
      `machine()` at 459 passes `reopen: reopenWord`. A new
      `askTheReopen()` replaces `reaskTheReopen()` (532 to 537) and the
      body of `reopenIfTheBundleMoved()` (872 to 885): it returns at
      once unless `bundle` is `.reopenNeeded` and the count is neither
      refused nor opening; computes `updateRunning = updateIsRunning()`
      and `childAlive = aChildIsUp()` once; asks
      `relauncher.decides(...)` with those two values and turns the
      answer into the gate (`.relaunch` clear, `.updateRunning` or
      `.childAlive` held, any other hold refused, and no relauncher
      refused); takes `reopen.asked(gate, now: clock())` and stores the
      count; where the standing moved, stores it, retick()s and
      recomposes; where `opensNow`, the count is already `.opening`, and
      a Task captures `reopenAsk` and calls `relauncher.reopen(bundle:,
      updateRunning: updateRunning, childAlive: childAlive,
      remembering:)` with the same two values, never a literal; back on
      the actor it returns if the generation moved, quits on `.quit`,
      and on `.keepTheReopenRow` sets the count `.refused`, retick()s
      and recomposes, because an ask the policy had just admitted that
      did not open is an open that failed. `bundleMoved(to:)` bumps
      `reopenAsk`, resets the count to `.held` and the word to `.held`
      on any change of state, then calls `askTheReopen()`, `retick()`
      and recomposes. `retick()` at 511 asks `signingIn || installing ||
      reopenPending`, where `reopenPending` is `.reopenNeeded` with the
      count not refused; the comment at 36 to 38 names three moving
      things. `tick()` at 524 calls `askTheReopen()`. Green.
- [ ] `testTheCheckIsOnTheScreenBeforeTheTwoPollsComeBack` and the rest
      of `SessionTests` green, untouched.
- [ ] `make app-lint`, `make app-test` and `make app` green.
- [ ] Commit: `feat: the app counts five seconds down and reopens at zero`.

### Task 4: the two scenes, the captions, and the renders once

**Files**

- `presence/Tests/PresenceUITests/Scenes.swift`
- `presence/Tests/PresenceUITests/Scenes/e11.jsonl` (new)
- `presence/Tests/PresenceUITests/Scenes/e12.jsonl` (new)
- `presence/Tests/PresenceUITests/RenderTests.swift`

**Steps**

- [ ] `RenderTests.testTheReopenScenesDrawTheTwoSentences`: scene e12's
      model has `notice == "Updated to v2.15.0. Reopens when the update
      finishes"`, `header.versionWaiting` true and `header.update` nil,
      because the ring is up, and NOW holds the update's own job row,
      because the record holds its live start; scene e11's model has
      `notice == "Updated to v2.15.0. Reopens in 3"`, `header.update`
      there and `versionWaiting` false, because the child is gone, and
      `header.version == "v2.14.0"`, the one still running. Both draw
      DOCUMENTS rows under the notice. e12 comes before e11 in
      `Scenes.all`, as their moments do. `RenderTests.swift:54` moves to
      20 with the message "the screens file holds eighteen scenes and
      the reopen count added two". Red.
- [ ] `Scenes.machine(...)` gains `bundle: BundleState = .same` and
      `reopen: ReopenStanding = .held`, handed to `Facts`. Two scenes
      after e10, in time order. `e12`, "Updated, and the update still
      running", moment 2026-10-17T17:05:58Z, `fixture: "e12"` whose
      record holds a live `start` of `update` with its pid in `alive`
      and a document or two, `machine(updating: true, bundle:
      .reopenNeeded("2.15.0"))`, caption: the press of Update at 17:05
      took the v2.15.0 that r8 found; the bundle on disk already names
      it while the update child is up, which is the order gdoc does it
      in, so the header draws the ring beside v2.14.0, the one still
      running, NOW carries the update as a job, and the row says the app
      reopens when the update finishes, not a count: a count that
      started before the child ended would run while the bytes are
      still being written. `e11`, "Updated, and reopening in three
      seconds", moment 2026-10-17T17:06:03Z, `fixture: "e11"` with the
      same update ended and the same documents, `machine(bundle:
      .reopenNeeded("2.15.0"), reopen: .counting(3))`, caption: the
      update ended two seconds ago, so the row above NOW counts five
      seconds down in whole seconds, 3 here, and at zero the app opens
      its new instance and this one quits; the header still says
      v2.14.0 because that is the app drawing it; nothing is for the
      person to do, the count is a clock the app runs, which is the one
      way the panel's no-forecast rule lets a number about the next
      second on, the way the sign-in's countdown is. Green.
- [ ] `Scenes.stillYourCall` at 154 to 156 is reworded, count still
      seven and "decision 39" still in it: the header counts the
      sign-in's three minutes down, as product decision 39 asks;
      principle 5 allows facts and not promises, and a countdown is
      allowed as a clock the app itself runs and acts on, the form the
      reopen count took on 2026-10-30; whether this one belongs on the
      panel is still Nail's. `Scenes.reopenRule`, one sentence printed
      on the index page beside `markRule` (`RenderTests.swift:762`): the
      notice row after an update counts five seconds down and the app
      reopens itself at zero, decided 2026-10-30; "Quit and reopen" is
      kept for the one case the reopen did not happen and no scene draws
      it, because it has shipped since M23 and no scene drew it then.
- [ ] `make app-lint` and `make app-test` green, then
      `PRESENCE_SHOTS="$HOME/gdoc-m32i-board/shots" make app-test`
      green. `ls ~/gdoc-m32i-board/shots` lists `e11-dark.png`,
      `e11-light.png`, `e12-dark.png`, `e12-light.png` and `index.html`,
      and the index carries `reopenRule`. The renders are not in git.
- [ ] Commit: `test: the reopen count in two scenes, and the renders`.

### Task 5: [Final] Verify acceptance criteria and update the documents

**Files**

- `docs/design/2026-10-17-presence-design-principles.md`
- `docs/guide/presence.md`
- `docs/team/README.md`
- `docs/plans/2026-10-30-gdoc-v2-m32i-the-reopen-count.md`

**Steps**

- [ ] Every test named in the Acceptance table exists under
      `presence/Tests` with that name.
- [ ] `git diff --name-only main...HEAD` lists only `presence/Sources`,
      `presence/Tests`, `docs/design/2026-10-17-presence-design-principles.md`,
      `docs/guide/presence.md`, `docs/team/README.md`, and this plan
      under `docs/plans/completed/` with its old path gone. Nothing
      under `go/`, and not `CLAUDE.md`: no invariant sentence names the
      row, and the argv, process and open rules did not move.
- [ ] `grep -rn "waits two\|two seconds\|RelaunchPolicy.settle\|never relaunches\|after this many seconds" presence/`
      prints nothing: `Relauncher.swift:6` to 7, `RelaunchPolicy.swift:103`
      to 104 and 131 to 133, `RelauncherTests.swift:5` to 6 and
      `BundleVersion.swift:10` to 11 were each rewritten in Tasks 1 and
      3. `Relauncher.settle`, `BundleWatch.settle()` and the English
      word in `Scenes.swift:123` are not in the pattern and stay.
- [ ] `grep -rn "Quit and reopen" presence/Sources` prints
      `PanelWords.swift` and the comments that name it as the kept
      sentence, and no other place that builds it.
- [ ] In the principles file, after M32h's heading, a new
      "## Amendments, 2026-10-30, the reopen count" opening with Nail's
      sentence and the decision's name, and two bullets. The first: the
      notice row after an update counts five seconds down, "Updated to
      v2.29.0. Reopens in 5" to 1, and the app opens its new instance at
      zero; while the update child or its record holds the reopen the
      row says "Reopens when the update finishes", and a guard up at
      zero sends the count back there to start again; "Quit and reopen"
      is kept for the one case the reopen did not happen; principle 5 is
      kept in the one form the sign-in's countdown kept it, a clock the
      app runs and acts on, and the checklist's "nothing predicts a
      time" reads that way for both; naming
      `ReopenCountTests.testTheCountReadsFiveDownToOneAndOpensOnceAtFive`,
      `StateTests.testTheNoticeRowSaysTheThreeReopenSentences` and
      `SessionTests.testTheReopenCountsFiveToOneOnTheTickAndOpensOnce`.
      The second: the count is the settle, the two seconds the
      relauncher slept are gone, the policy is asked before the count
      and again before the open, and every guard stays; naming
      `RelauncherTests.testTheOpenHappensOnTheAskWithNoSleepBeforeIt`,
      `testDecidingWritesNoStampAndOpensNothing` and
      `RelaunchPolicyTests.testRelaunchHappensOncePerOnDiskVersionInTenMinutes`.
- [ ] `docs/guide/presence.md`: in "### Update, and Install extension"
      (about line 414), "A run that installed something relaunches the
      app, so the version on the line is the one you are running"
      becomes three sentences: while the update is still running the row
      above the panel says "Updated to v2.29.0. Reopens when the update
      finishes"; once it has ended the row counts five seconds down,
      "Reopens in 5" to 1; at zero the app comes back by itself with the
      new version on the line. In "## Keeping it updated, and taking it
      back" (about lines 604 to 612), "it waits two seconds" becomes the
      count, and the last sentence says the row reads `Updated to
      v<version>. Quit and reopen` only when the open did not happen,
      and quitting and opening it again is all that is needed then.
      Every string `TestThePresencePageWalksTheThirdLooksPanel` requires
      stays, and
      `cd go && go test ./boundary/ -run 'TestThePresencePageWalksTheThirdLooksPanel|TestCLAUDEmd|TestTheTaskMap'`
      prints `ok`.
- [ ] `docs/team/README.md:59` and 99: the design reviewer reads 20
      dark whole-panel renders, not 18, with "since M32i added the two
      reopen scenes" in the seat's row.
- [ ] `make app-lint`, `make app-test` and `make app` green on a clean
      tree.
- [ ] Move this plan to `docs/plans/completed/`.
- [ ] Commit: `docs: the reopen count`.

## Acceptance

| Claim | Test |
|---|---|
| The count begins on the first clear gate, and not before | `ReopenCountTests.testTheCountBeginsOnTheFirstClearGateAndNotBefore` |
| It reads 5 down to 1 in whole seconds, opens once at five, and `.opening` never opens again | `ReopenCountTests.testTheCountReadsFiveDownToOneAndOpensOnceAtFive` |
| A held gate before five keeps counting; at five it goes back to held and restarts from 5 | `ReopenCountTests.testAHeldGateBeforeFiveSecondsKeepsCountingAndAtFiveGoesBackToHeld` |
| A refused gate refuses for good, from any state | `ReopenCountTests.testARefusedGateRefusesForGoodFromAnyState` |
| The row is clamped to 1 through 5 against a clock that goes back | `ReopenCountTests.testTheRowIsClampedToOneThroughFive` |
| Five seconds, as a literal | `ReopenCountTests.testTheCountIsFiveSeconds` |
| The decision carries no wait, every guard and the ten minutes stay | `RelaunchPolicyTests.testRelaunchHappensOncePerOnDiskVersionInTenMinutes`, `testNoRelaunchWhileAnUpdateRunsOrAChildLives`, `testNoRelaunchIntoAnotherIdentifierOrAnInvalidSignature`, `testGoneNeverRelaunches`, `testAFailedOpenLeavesTheReopenRow` |
| The three sentences, with the header's `v`, and nothing under `.same` | `StateTests.testTheNoticeRowSaysTheThreeReopenSentences` |
| None of them is a forecast in words | `StateTests.testTheReopenSentencesCarryNoForecastWord` |
| The standing survives both field-by-field rebuilds of the facts | `HealthPollerTests.testTheReopenStandingSurvivesThePollsRebuild`, `UpdateActionTests.testTheReopenStandingSurvivesTheUpdateActionsRebuild` |
| The relauncher opens on the ask, with no sleep before it | `RelauncherTests.testTheOpenHappensOnTheAskWithNoSleepBeforeIt` |
| Deciding writes no stamp and opens nothing | `RelauncherTests.testDecidingWritesNoStampAndOpensNothing` |
| A failed open keeps the row and writes the stamp | `RelauncherTests.testAFailedOpenKeepsTheRowAndWritesTheStamp` |
| A held reopen still costs nothing and the next ask opens | `RelauncherTests.testAHeldReopenCostsNothingAndTheNextAskOpensTheNewInstance`, `testAChildOfThisAppsOwnHoldsTheReopenAndThenLetsItGo` |
| The session counts 5 to 1 on a one-second tick and opens once at zero, two ticks at zero included | `SessionTests.testTheReopenCountsFiveToOneOnTheTickAndOpensOnce` |
| Held while the update is live, counting from its end | `SessionTests.testTheReopenIsHeldWhileTheUpdateIsLiveAndCountsFromItsEnd` |
| A guard up at five sends it back to held, and it restarts | `SessionTests.testAGuardUpAtFiveSendsTheCountBackToHeldAndItRestarts` |
| Refused for good, a failed open, and no relauncher each read "Quit and reopen", with the tick back at five | `SessionTests.testAReopenRefusedForGoodSaysQuitAndReopenAndTheTickGoesBackToFive` |
| An answer from before the bundle moved again is dropped | `SessionTests.testAnAnswerFromBeforeTheBundleMovedAgainIsDropped` |
| The check after Update still reaches the screen before the polls | `SessionTests.testTheCheckIsOnTheScreenBeforeTheTwoPollsComeBack` |
| Only the relauncher opens an application, `Process` stays in the Runner, the argv set stays six | `SourceRulesTests.testOnlyTheRelauncherOpensAnApplication`, `testTheSystemIsAskedOnlyThroughTheOpenerAndTheRelauncher`, `testProcessAppearsOnlyInTheRunner`, `testTheArgvEnumIsClosedAtSix`, `RunnerTests.testTheArgvSetIsExactlyTheSix` |
| Two scenes draw the held and the counting sentence in time order, and the scene count is twenty | `RenderTests.testTheReopenScenesDrawTheTwoSentences`, `testEveryScreenRendersInBothAppearances` |
| Every sheet word is still a word a scene draws | `RenderTests.testEverySheetWordIsAWordAPanelDraws` |
| The open choices stay at seven, with the countdown line true again | `RenderTests.testTheIndexOpensWithWhatIsStillYourCall` |
| The guide walks the panel with the count named | `TestThePresencePageWalksTheThirdLooksPanel` |

## Post-Completion

- The design reviewer reads `/Users/nailkhusnullin/gdoc-m32i-board/shots`:
  e12 with the ring in the header, the update's job in NOW and "Updated
  to v2.15.0. Reopens when the update finishes" above it, e11 with
  "Updated to v2.15.0. Reopens in 3" above NOW, both in both
  appearances, the index's `reopenRule` sentence and the reworded
  countdown line, and the other eighteen unchanged. Twenty dark
  whole-panel images, not eighteen, which `docs/team/README.md` now says.
- `make release BRANCH=gdoc-v2-m32i-the-reopen-count` from the checkout
  on `main`, after M32h's release has merged; `GO=` with the next version
  only on Nail's word.
- Nail's hand check, in what he sees: on the next release after this one,
  he presses Update; the header draws the ring; the row above the panel
  says "Updated to v<version>. Reopens when the update finishes" while
  the ring is up, then "Reopens in 5", 4, 3, 2, 1, one a second; the
  panel goes and comes back by itself with the new version on the line,
  and there is one app in the menu bar. He does nothing between the
  press and the new panel.
