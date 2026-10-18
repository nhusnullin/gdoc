# gdoc v2 Milestone 25: the app opens itself. Plan

2026-10-18. Nail ran a publish from Claude Code with v2.15.0 installed and
the panel was not there: the app had never been opened, so nothing had
registered it to start at login, and gdoc by rule launched nothing. His
expectation, in his words: if the app is not open, the gdoc command opens it;
he can open it by hand; and it starts at login when he has said so in a
setting. This plan is those three, and it reverses two rules of M23 and M24:
gdoc now runs `open` for the app it installed, and launch at login is off
until the person turns it on. The decision is `docs/v2/DECISIONS.md`,
2026-10-18, "The app opens itself"; the spec paragraphs are already changed.
One commit per task. The spec holds the rules; where the code, the spec and
this plan disagree, the task stops and says so.

## Principles

Serves 4 (the person sees what gdoc does without asking) and strains 1 (one
more thing the binary does on this machine), which the decision names.

## Context (from discovery)

- `go/cmd/gdoc/main.go` dispatch: `presence.Begin(ctx, set.Presence &&
  c.report, ...)` after the parse and before the work; `rec` is nil with the
  flag off and every method is a no-op. `viaOfEnv` marks Claude Code's runs.
- `go/cmd/gdoc/desktop.go`: `macOSOpen = "/usr/bin/open"`, `runProgram =
  startProgram` is the seam, `startProgram(name, arg)` checks the name and
  runs `exec.Command(macOSOpen, arg)` with one argument; `withExtension` is
  its one caller.
- `go/boundary/boundary_test.go`: `TestNothingRunsAnExternalProgram` (os/exec
  only in desktop.go), `TestOnlyDesktopRunsAProgram` (one exec.Command call,
  the opener constant, exactly two arguments to exec.Command),
  `TestTheOneProgramOpensOnlyTheExtension` (the seam's one caller is
  `withExtension`, test files skipped). All three change in Task 2 and say
  what the new narrowness is.
- `go/internal/update/bundle.go`: `ApplyBundle` puts the app at its `dest`
  under `~/Applications`; `go/cmd/gdoc/update.go` `appHere(presenceOn)`
  decides whether the bundle travels with the update.
- `presence/Sources/Presence/main.swift`: `Shell.applicationDidFinishLaunching`
  builds the `Session`; `applicationWillTerminate` stops it.
- `presence/Sources/PresenceApp/LoginItem.swift`: the first run ever registers
  the login item and records "on"; the HEALTH switch records the person's
  choice; a source scan holds `SMAppService` to that file. Tests
  `testTheFirstEverRunRegistersOnceAndRecordsOn`,
  `testAPersonsOffIsNeverOverridden` in `PresenceAppTests`.
- `presence/Sources/PresenceApp/Places.swift` knows the config directory the
  record lives in; the record is `presence/<yyyy>-<mm>.jsonl` under it.
- The invariant list in `CLAUDE.md` says the menu bar app is never started by
  gdoc; `docs/guide/presence.md` says the first launch is yours; the M24 spec's
  decision table row says launch at login is on by default.
- Measured 2026-10-17 (MEASURED.md): the bundle `gdoc update` installs
  launches under `open`, and the app relaunches itself after a swap.

## The mechanism, so the tasks agree

Whether the app is running is one `flock`. The app, as its first act, opens
`presence/app.lock` in the config directory (creating it) and holds
`LOCK_EX` on it for its whole life; the kernel drops the lock when the process
dies, however it dies. gdoc, under the flag, after `presence.Begin` and only
on macOS, asks three things and runs `open` when all three say yes: the
bundle is at `~/Applications/GdocPresence.app`; the lock file exists, or does
not yet, because an app that never ran made none; and a `LOCK_EX | LOCK_NB`
on it succeeds, which gdoc releases at once. No byte is read from any file;
the record stays the one file gdoc appends to and never reads. The program is
`/usr/bin/open -g <bundle>`: `-g` so no window or focus moves, because the app
has no window and the person is in a terminal or a chat. The open is started
and not waited for, so a command costs nothing it did not cost before; its
exit status is never read, and a machine with no window session (ssh) runs
the command exactly as before. `gdoc update`, after it installs or replaces
the bundle under the flag, runs the same opener once, so an install is a
running app.

Launch at login: nothing registers on the first run. The HEALTH switch
registers and unregisters, and a choice once made is kept as today. The
`Launch at login` row's switch reads what macOS says.

## Development Approach

TDD, one commit per task, the test first and failing. `make test` keeps the
race flag; `make app-test` and `make app-lint` for the Swift half; `make
vet`; `make app`. Every path, flag and word a test states is a literal. No
test starts the real `open` or the real app: the seam records the argv, and
the Swift lock is tested against a temporary directory. A task that finds the
spec, the decision and the code disagree stops and says so.

## Validation Commands

```bash
make vet
env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test
make app-lint
make app-test
make app
```

## Progress Tracking

Seven tasks. A task is done when every box is ticked and its commit is on the
branch.

## Tasks

### Task 1: the app holds its lock

**Files:**
- Create: `presence/Sources/PresenceApp/Lock.swift`,
  `presence/Tests/PresenceAppTests/LockTests.swift`
- Modify: `presence/Sources/PresenceApp/Places.swift` (the lock's path),
  `presence/Sources/Presence/main.swift`

- [x] `Places` names `presence/app.lock` under the config directory, beside
      the month files; a test pins the literal path
- [x] `AppLock.hold(at:)` opens the file (creating it, mode 0o644), takes
      `flock(LOCK_EX | LOCK_NB)` and keeps the descriptor; it answers whether
      it got the lock, and a second hold on the same path from another
      process does not; the test proves the second half with a child
      `/bin/sh -c 'flock'`-free method: a second `open` + `flock(LOCK_NB)` in
      the test process on a different descriptor, which macOS refuses while
      the first is held
- [x] the app takes the lock as the first line of
      `applicationDidFinishLaunching`; a second instance that does not get it
      terminates at once without drawing a status item, and a test over the
      decision function pins that
      (`AppLock.decide`). One instance waits instead of leaving: the one an
      update opened, which carries `--relaunched-by <pid>` and whose launcher
      quits as soon as it is up. It asks again for `RelaunchPolicy.patience`
      and then yields, so M24's relaunch still works; `AppLock.patience` is
      that rule and `testOnlyAnInstanceAnotherOneLaunchedWaitsForTheLock`
      pins it
- [x] the descriptor is never closed before `applicationWillTerminate`
      (`testTheShellTakesTheLockFirstAndLetsGoLast`, a source scan over the
      shell, which no test can import)
- [x] `make app-lint`, `make app-test`

### Task 2: gdoc opens the app it installed

**Files:**
- Modify: `go/cmd/gdoc/desktop.go`, `go/cmd/gdoc/main.go`,
  `go/cmd/gdoc/update.go`, `go/cmd/gdoc/doc.go`, `go/cmd/gdoc/*_test.go`,
  `go/boundary/boundary_test.go`, `go/boundary/doc.go`

- [x] the seam takes the program's arguments as a list: `runProgram(name,
      args ...string)`; `startProgram` still refuses any name but
      `macOSOpen`; `withExtension` passes one argument as before
- [x] a second function in `desktop.go`, `startProgramDetached`, or a flag on
      the one, starts the program and does not wait; it is used by the
      opener only
- [x] `openPresenceApp(set, dirs)` in `desktop.go`: on macOS, under the flag,
      when the bundle is at `~/Applications/GdocPresence.app`, when the lock
      file is absent or a `LOCK_EX | LOCK_NB` on it succeeds (released at
      once, nothing read), runs `/usr/bin/open -g <bundle>` detached; every
      other case does nothing and says nothing; it never creates the lock
      file; a table test over the eight combinations through a recorder seam
      and a temporary directory, with a real `flock` held by the test for the
      "running" rows
- [x] dispatch calls it right after `presence.Begin` and only when `rec` is
      not nil, so the flag off is today's bytes:
      `TestTheFlagChangesNoByteOfTheObject` still passes, and a new test
      shows a command under the flag asks the seam for exactly
      `/usr/bin/open -g <bundle>` once, and a command with the lock held asks
      for nothing
- [x] `gdoc update`, after `ApplyBundle` placed or replaced the bundle under
      the flag, calls the same opener once; a test over the update screen
      shows the argv once and the `open` line gone from the answer's words
      (the answer no longer tells the person to open it)
- [x] the three boundary tests state the new narrowness: os/exec in
      `desktop.go` alone; one `exec.Command` call whose first argument is the
      opener constant and whose arguments are the constant and a list, with
      the list's two shapes pinned (`[mcpb]` and `[-g, bundle]`); the seam has
      two callers, `withExtension` and `openPresenceApp`, and no third; the
      names of the tests change where their sentences change, and `doc.go`
      and `CLAUDE.md` name the new names in Task 6
- [x] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`

### Task 3: launch at login is the person's switch

**Files:**
- Modify: `presence/Sources/PresenceApp/LoginItem.swift`,
  `presence/Tests/PresenceAppTests/LoginItemTests.swift` (or where the two
  tests live), `presence/Sources/PresenceUI/Words.swift` if a word changes

- [x] no run registers the login item by itself; the first run ever leaves
      macOS as it found it; `testTheFirstEverRunRegistersOnceAndRecordsOn`
      becomes `testTheFirstEverRunRegistersNothing`
- [x] the HEALTH switch registers on "on" and unregisters on "off", records
      the choice, and `requiresApproval` keeps its row and button;
      `testAPersonsOffIsNeverOverridden` stays and a sibling pins that an
      "on" is never made for them
- [x] the switch's state is what `SMAppService.mainApp.status` says, read on
      open, so a registration removed in System Settings shows off
- [x] the scene fixtures and renders that showed the switch on by default now
      show it off where the record has no choice, and on in the one scene
      whose fixture says the person turned it on; the index caption's "Still
      your call" line about launch at login is replaced by the decision's
      sentence
- [x] `make app-lint`, `make app-test`

### Task 4: the words

**Files:**
- Modify: `docs/guide/presence.md`, `go/cmd/gdoc/doc.go`,
  `go/internal/presence/doc.go`, `CLAUDE.md`

- [x] `docs/guide/presence.md`: the install walk-through loses its "open it
      once" step and says what happens instead: the next gdoc command under
      the flag opens the app, `open ~/Applications/GdocPresence.app` still
      works by hand, and Launch at login is the switch in HEALTH, off until
      you turn it on; the "What it is not" section keeps every sentence that
      is still true and loses "gdoc never launches it"
- [x] `CLAUDE.md`: the external-program invariant names two argument lists
      for the one program and the two callers of the seam, and the presence
      invariant adds the lock in one sentence (gdoc holds no lock and reads
      no byte; it asks once whether the app holds its own); the test names
      are the ones Task 2 chose; the file stays under 300 lines
- [x] `go/cmd/gdoc/doc.go` and `go/internal/presence/doc.go` name the rule
      and its tests
- [x] `make test` (the CLAUDE.md tests), `make app-lint`

### Task 5: measured on the machine

**Files:**
- Modify: `docs/v2/MEASURED.md`

- [x] measured and written down, with the method: `open -g` on the installed
      bundle while it is not running starts it and moves no focus from the
      terminal; `open -g` while it is running (lock held) is never issued, and
      issued by hand it brings nothing to the front; a gdoc command from a
      shell with no window session (`ssh localhost` or `launchctl asuser`
      with no GUI) answers exactly as before and leaves no error on stderr;
      the lock is free within a second of `kill -9` of the app. MEASURED.md,
      2026-10-18, "The lock, `open -g`, and a command whose opener cannot
      start anything". Two readings came out weaker than the sentence asked
      for and say so there: the display was locked, so no focus could move in
      either direction, and the no-window-session run could not be made
      because Remote Login is off and the account has no administrator
      rights, so a failing `open` stands in for it and the real run is a
      recheck line
- [x] where a measurement contradicts the mechanism paragraph, the task stops
      and says so (nothing contradicted it)

### Task 6: the acceptance rows and the close

**Files:**
- Modify: this plan, `docs/plans/2026-10-17-gdoc-v2-m24-the-panel-spec.md`
  (its decision table row about launch at login points at the 2026-10-18
  decision), `docs/v2/DECISIONS.md` only if a task found the decision wrong

- [ ] an acceptance table at the foot of this plan: each sentence of the
      mechanism paragraph with the test that holds it
- [ ] the M24 spec's launch-at-login row and the render index's "Still your
      call" list say the 2026-10-18 decision reversed them
- [ ] move this plan to `docs/plans/completed/`
- [ ] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
      `make app-lint`, `make app-test`, `make app`

### Task 7: the reviewer's pass over the changed scenes

**Files:**
- Modify: whatever the pass finds under `presence/`

- [ ] render every scene (`PRESENCE_SHOTS=... make app-test`) and run the
      checklist of `docs/design/2026-10-17-presence-design-principles.md`
      over the scenes Task 3 changed, as a reviewer who drew nothing (a
      subagent that reads images); every blocking or should-fix finding is
      fixed here and listed in the commit message
- [ ] `make app-lint`, `make app-test`
