# gdoc v2 Milestone 32: the quiet watch. Plan

Tasks: 9
Review: go

2026-10-27. Week one of the whole-jobs road in `docs/v2/PLAN.md`. A live review
stops paying a model turn for a quiet document and for gdoc's own replies, the
skills open on a short help, and every marked comment gets a written, noticed
and answered time. Run by a ralphex worker on Claude, one task, one commit, on
the local branch `gdoc-v2-m32-the-quiet-watch`. `make release` runs without
`GO`, so the push and the PR wait for Nail's word, and nothing is merged or
tagged without it.

## Principles

Serves: 1, because a live review costs about half the tokens and a quiet hour
costs none, so Nail's work is not slowed by a watch that spends. 4, because the
panel keeps telling the truth about a watch that now runs for half an hour.

Strains: the output contract, in one place. `comments --follow` writes one JSON
object per line, the third exception beside `mcp` and the help screen. That is
accepted in the decision entry: the last line is the envelope, the exit code
follows it, and every other command keeps one object. Source rules hold the
line writer to this one mode, and the chat route hands none.

## Overview

The decision entry of 2026-10-27 in `docs/v2/DECISIONS.md` ("Week one of whole
jobs: ...") is the scope, signed by the trust guardian. The lead wrote it and
the matching `docs/v2/SPEC.md` changes on main before this run. This plan
touches neither file.

What lands, in the order the tasks take it:

1. **`help --brief`.** The mark in effect and the update notice, and no table.
2. **`comments.Follow`.** The same poll as `comments.Wait`, held open, asking
   from the last cursor it handed over, reading the document once.
3. **The line writer.** `emit.Lines`, handed from `run` to `comments` alone,
   and never to the chat route.
4. **`comments --follow` on the command line.** One short JSON line per window
   with work, the envelope last, the start recorded as a wait.
5. **The panel's proof.** The app already shows a wait live while its process
   runs and lets Stop reach it after the process check. Tests prove it for a
   follow of half an hour, and a stale line number is fixed.
6. **The live loop under the Monitor tool**, with today's loop as the fallback.
7. **The five skills open on `help --brief --json`**, falling back to `help
   --json` on an older binary.
8. Acceptance.
9. Documentation, including two invariant lines in `CLAUDE.md`, net zero lines.

## Context (from discovery)

Read on main after the M30 and M31 merges.

**The wait today.**
- The table entry is `go/cmd/gdoc/commands.go:312-327`: `--since` kindCursor,
  `--witness` kindNone, `--wait` kindDuration, all needOptional, example
  `--since <cursor> --wait 9m`, `report: true`.
- `cmdComments` is `go/cmd/gdoc/read.go:525-660`, 135 lines, in a file of 851.
  `--wait` without `--since` is refused (:541-544). `parseWait` (:511-523)
  refuses a non-duration, a value at or under zero, and one over
  `maxWait = time.Hour` (:435), and its three messages name `--wait` literally.
  The `list` closure for the wait captures a fixed `since` (:574-576).
- `waitInterval = 2 * time.Second` (read.go:430), pinned by
  `TestTheWaitIntervalIsTwoSeconds` (read_test.go:1199).
- SIGINT is trapped with `signal.NotifyContext(ctx, os.Interrupt,
  syscall.SIGTERM)` at read.go:618. `TestOnlyTheWaitTrapsTheSignal`
  (main_test.go:337) holds read.go as the one non-test file importing
  `os/signal`.
- The envelope is `commentsData` (read.go:455-463) with `waited,omitempty`, and
  `waitedData` (:441-445) is `{polls, seconds, interrupted}`. A failed poll
  builds its envelope with `commentsBase(r, d, since)` (read.go:647), the
  cursor handed in.
- The loop is `comments.Wait`, `go/internal/comments/wait.go:94-215`. Its `List`
  type takes no cursor (wait.go:40), and the package holds no session and no
  URL (wait.go:34-39). It reads the document when the listing is non-empty or
  on the first poll (:174), and returns on the first poll where `Threads(...)`
  is non-empty (:182-189). gdoc's own replies count as activity, on purpose:
  `TestAWindowThatIsOnlyAGdocReplyIsStillReturned` (wait_test.go:294). A quiet
  poll reads no document: `TestAQuietPollDoesNotReadTheDocument`. The deadline
  context and the `failed` classification, with its `DeadlineExceeded` gate,
  are wait.go:124-161. `now` and `sleep` are package variables (:19-32).
- `Fetch` is comments.go:184, `Threads` comments.go:221 and accepts a nil
  document, `NextCursor` cursor.go:241. The cursor's own boundary rule is
  `newer`, `sameInstant` and `reported` (cursor.go:155-190): an item is news
  when created after `At`, or at `At` on a thread not in `Ids`.

**What a window carries.**
- A `Thread` (comments.go:99-115) has id, author, author_domain, created,
  modified, content, marker, resolved, quoted, range, replies, witness. A
  `Reply` (:120-128) has id, author, author_domain, created, content, marker and
  `by_gdoc`.
- `markerOf` (:285) is an exact first token of `ai:`, `ai?` or `ai!`, else
  `"none"`. `byGdoc` (:316) calls `plaintext.OpensWithMark` (plaintext.go:278),
  which knows the three known marks and the mark in effect. A top-level comment
  gdoc writes opens with the mark too (propose.go:355, :512, blockapply.go:121).

**Dispatch and stdout.**
- `emit.Print` (emit.go:55-59) is one `json.Encoder.Encode`.
- A command's `run` is `func(ctx, a *args, errOut io.Writer) emit.Result`
  (commands.go:192). `run` (main.go:362) calls `safeDispatch(ctx, args, errOut,
  set)` and hands `out` only to `printResult`; the chain is `run`,
  `safeDispatch`, the test seam `dispatching`, `dispatch`, `parseArgsN`. The
  chat route calls `safeDispatch` from `mcpSend` (go/cmd/gdoc/mcptools.go:502),
  and a fake `dispatching` lives in settings_test.go:68. The chat comments tool
  maps only `--since` and `--witness`, with `additionalProperties: false`
  (mcptools.go:297-310).
- `onlyJSONObject` and `decodeOne` (main_test.go:20-51) require one object and
  then EOF; `TestOnlyJSONObjectRefusesAnythingAfterTheObject` (main_test.go:58)
  tests that helper. `TestStdoutCarriesOnlyJSONRPC` (mcp_test.go:89).

**Help.**
- `cmdHelp` is help.go:108-141 and returns `helpReport{commands, settings{mark,
  mark_from, presence?, presence_from?}, update?}` (help.go:22-80), and calls
  `notice(ctx)`, the once-a-day GitHub check. `version` is the envelope's own
  top-level field, set by `printResult` (main.go:375), absent under `dev`.
  `help` is the entry at commands.go:515-529 with `--json` as its one flag.
  `helpAsked` and `helpWords` are help.go:340-385. The screen is `writeHelp`
  (helpscreen.go:127), and the notice head is `writePlainHead`.
- Help goldens: `go/cmd/gdoc/testdata/pipe/help.stdout.golden`,
  `help-comments.stdout.golden`, `help-comments.stderr.golden`. Literal usage
  line in `TestTheUsageLineMarksWhatIsOptionalAndWhatIsAnAlternative`
  (help_test.go:299). `TestHelpTakesWordsAndOnlyTheJSONFlag` (help_test.go:258).

**The presence record.**
- `presence.Begin` (event.go:129) writes the start with `Wait: wait`. Its one
  call site is main.go:493-494, passing `a.has("--wait")`, held by
  `TestBeginHasOneCaller` (go/boundary/presence_test.go:185). `End` lifts
  `interrupted` from `data.waited.interrupted` (event.go:498-544) and the title
  from the envelope.
- `TestTheEventCarriesOnlyItsListedFields` (presence_test.go:130) and
  `TestTheSwiftFixtureIsTheGoFixture` (go/boundary/presence_test.go:310).
  `go/internal/presence/doc.go:111` and event.go:490 say "`comments --wait`";
  that package is not touched here.
- In cmd/gdoc, presence_test.go:124 is a `comments --wait` scene in
  `presenceScenes` for `TestTheFlagChangesNoByteOfTheObject`.

**The panel.**
- `LiveSession.swift`: `Live.gap = 3*60` (:118). A wait with no end whose pid is
  alive is in flight (:245-266), whatever its age. A stopped wait ends the
  session at once; any other end waits out the gap.
- `ProcessCheck.verdict` (PresenceCore/ProcessCheck.swift:109-137) checks the
  record's start (`command == "comments"` and `wait`), not `via: chat`, no end
  yet, pid alive, `proc_pidpath` resolving to the app's gdoc, the uid, and a
  start inside the 10 s window. It reads no argv.
- `Stopper.swift`'s comment cites `go/cmd/gdoc/read.go:589`; the trap is now at
  read.go:618.
- Tests build events inline with `wait(from:to:run:)` (LiveSessionTests.swift:67):
  `testWaitsWithinThreeMinutesAreOneSession` (:52),
  `testAnInterruptedWaitEndsTheWatchAtOnce` (:155),
  `testTheSessionKeepsPidRunAndCallForStop` (:298),
  `testAWaitWhoseProcessWentAwayEndsTheWatch` (:323),
  `testStopSendsOneSIGINTOnlyAfterTheProcessCheck` (StopperTests.swift:31).

**The skills.**
- `skills/gdoc-review/live.md` runs `$GDOC comments <url> --since "$CURSOR"
  --wait 9m` with a Bash timeout of 600000, four paths (empty, threads, `ok:
  false`, interrupted), the cursors it keeps, what a window holds ("An unmarked
  reply is still reported and never acted on"), and the stop and its totals.
- The preamble `$GDOC help --json` is at gdoc-review SKILL.md:30, gdoc-export
  :43, gdoc-restyle :30, gdoc-publish :29, gdoc-align :42. gdoc-publish is at
  19,256 of 20,000 characters. Every skill's `needs:` is v2.11.0, and
  `TestNoSkillNeedsAReleaseNobodyCut` ties it to plugin.json, v2.21.0.
- Tests: `TestEverySkillPassesJSONToHelp` (skills_test.go:207),
  `TestEverySkillNamesOnlyCommandsAndFlagsTheBinaryHas` (:170), which reads
  `*/SKILL.md` only through `skillFiles` (:39); `skillProblems` and
  `skillCalls`; `TestTheLiveLoopSaysWhichCursorsItKeeps` (:775);
  `TestEverySkillBodyFitsTheSpecBudget` (skillspec_test.go:177).

**Test helpers.** `runJSON`, `runJSONCtx`, `runJSONSet` (main_test.go:82-103);
`fakeWire` and `stubWire` (write_test.go:62, :210); `waitWire`,
`emptyListing`, `quietListing`, `newsListing`, `waitSince`, `waitNext`,
`waitedOf` (read_test.go:821-860); `stubNow` (:629) and `shortPolls` (:810);
`newScript` in wait_test.go:40-104; `atVersion` and the notice stub the help
goldens use.

**The ceiling.** `CLAUDE.md` is exactly 300 lines, and `claudeCeiling = 300`
(go/boundary/docs_test.go:40).

## Development Approach

- **TDD.** Each task writes its tests first and runs them red before the code.
- One commit per task. No push, PR, merge or tag.
- `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md` and `docs/v2/PLAN.md` are not
  edited. `CLAUDE.md` is edited in Task 9 only, at two invariant lines, with no
  net change in its line count.
- `--wait` keeps every behaviour it has, and `wait_test.go` is not edited.
  Wait's deadline context and its `failed` classification may move into one
  unexported helper in wait.go that both `Wait` and `Follow` call, as a pure
  extraction, with every Wait test green before and after.
- A help golden is regenerated only where a task names it, by deleting it and
  running the test twice, and the commit message says the diff is the new flag
  lines only.
- Every `doc.go` a task touches names the test for each rule it adds.
- `make test` keeps the race flag, run as `env -u GDOC_LIVE_TEST -u
  GDOC_LIVE_WRITE make test`, so nothing reaches Drive.

## Testing Strategy

- `internal/comments`: the follow loop, with a cursor-aware list in
  follow_test.go and `now` and `sleep`.
- `cmd/gdoc`: the flag, the refusals, the lines, the envelope, the exit code,
  the interrupt, the presence start, the help, with a new `runLines` helper
  that reads every stdout line as one object.
- Source rules in `cmd/gdoc`: who builds the line writer and who writes with it,
  and that the chat route hands none.
- Swift: the panel's reading of a long wait, and Stop.
- Skills: the loop text, the preamble and its fallback, and every call in
  `live.md` checked against the table.
- `make vet`, `make app-test`, `make app-lint`.

## Solution Overview

- `comments.Follow(ctx, since *Cursor, FollowOptions{Interval, Deadline, List
  func(context.Context, *Cursor) ([]RawComment, error), Read, Mark, Hand
  func(Window) error}) (Followed, error)`. No session, no id: `read.go` hands a
  `List` closure that calls `comments.Fetch(ctx, session, id, c)` for the
  cursor it is given.
- Follow reads the document once, on its first poll, for the envelope's title,
  tabs and `multi_tab`, and judges every window on `Threads(raw, nil, mark)`.
- The wake rule is one unexported predicate named for the fact it tests,
  `hasNewMarked(threads, since, mark)`: a thread or reply that is news by the
  cursor's own rule, whose own marker is `ai?`, `ai!` or `ai:`, and which does
  not open with gdoc's mark.
- A window with no such item prints nothing and keeps the cursor, so the next
  poll asks from the same place and the held items ride in the next line.
- The line writer is `emit.Lines`. `run` builds it from `out` and passes it
  down `safeDispatch` and `dispatch` into `args`; `mcpSend` passes nil;
  `comments --follow` is refused when it has none.
- The start is recorded as a wait: `presence.Begin` gets `a.has("--wait") ||
  a.has("--follow")` at its one call site. The record gains no field.

## Technical Details

**The flag.** `--follow <duration>` in the comments entry, kindDuration,
needOptional, example `--since <cursor> --follow 29m`. It needs `--since`, and
is refused beside `--wait` or `--witness`, each refusal naming the pair.
`parseWait` becomes `parseWait(flag, text string)`; the `--wait` call passes
`"--wait"`, so its messages stay byte for byte.

**A line.** One JSON object, newline-terminated, written as soon as the window
is found:

```
{"line":"window","since":"<cursor>","cursor":"<cursor>","noticed":"<RFC3339 UTC>","items":[{"id":"...","thread":"...","marker":"ai?","created":"<RFC3339>"}],"more":0}
```

`items` lists the new threads and replies in the window, held ones included,
each with its own marker, at most twenty, and `more` counts the rest. No author,
no content, no quote. The line and item shapes and their conversion live in a
new `go/cmd/gdoc/follow.go`, which imports no `os/signal`.

**The last line.** The ordinary comments envelope, `ok` and all, with `cursor`
as the last cursor handed over, `threads` empty, and `waited` as today:
`{polls, seconds, interrupted}`. A deadline ends with `interrupted: false`, a
signal with `interrupted: true`, a failed poll with `ok: false`, the polls so
far, and the last cursor handed over, not the one handed in. The exit code
follows `ok`, through `printResult` as today.

**The interrupt.** The same `signal.NotifyContext` in read.go, around a short
`commentsFollow` there. A line is written whole before the context is checked
again, because writes and polls share one goroutine.

**`help --brief`.** A second flag of the `help` entry, kindNone. Under it
`cmdHelp` drops `commands`, keeps `settings` and `update`, writes only the
notice head (`writePlainHead`) to stderr, and returns `Screen: false`, so the
object prints on a terminal too. `version` stays the envelope's own field. It
takes no command word: `help publish --brief` is refused by name. `gdoc --help
--brief` is refused as today, because `--brief` is a flag of `help`, not of
every command.

**What is not built.** No state file, no lock, no listener, no new presence
field, no change to `Wait`'s behaviour, no change to the chat tools or Claude
Desktop, no `needs:` bump.

## What Goes Where

- **Implementation Steps**: help, the follow loop, the line writer, the
  command, the panel's proof, the live loop, the preamble, acceptance,
  documentation.
- **Post-Completion**: Nail's hand check, the stopwatch at the end of the
  week, the meter, and Nail's go.
- **Out of scope, on purpose**: the job packet, a session per comment, answers
  with no session open, the echo filter in `--wait`, the chat tools and Claude
  Desktop, `help <command>` calls inside the four other skills, any visible
  change to the panel, and `go/internal/presence`'s own wording.

## Implementation Steps

### Task 1: `help --brief`

**Files:**
- Modify: `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/help.go`,
  `go/cmd/gdoc/helpjson_test.go`, `go/cmd/gdoc/help_test.go`,
  `go/cmd/gdoc/testdata/pipe/help.stdout.golden`, `go/cmd/gdoc/doc.go`

- [x] test first in `helpjson_test.go`, with `atVersion` and the notice stub:
  - `TestHelpBriefCarriesTheMarkAndTheNotice`: `help --brief --json` decodes
    `version`, `settings.mark`, `settings.mark_from`, and `update` when a notice
    is due, and has no `commands` key
  - `TestHelpBriefIsAFewHundredBytes`: under 600 bytes with the default
    settings and no notice
  - `TestHelpBriefTakesNoCommandWord`: `help publish --brief` is refused by name
  - `TestHelpBriefPrintsItsObjectOnATerminal`: with both streams a terminal,
    stdout carries the object and stderr only the notice head
- [x] run them red
- [x] `--brief` in the `help` entry; `cmdHelp` as in Technical Details
- [x] `TestHelpTakesWordsAndOnlyTheJSONFlag` becomes
      `TestHelpTakesWordsAndOnlyItsTwoFlags`, with its comment
- [x] regenerate `help.stdout.golden` only; the commit says the diff is the
      `--brief` lines
- [x] `doc.go`'s help paragraph names `--brief` and its tests
- [x] `make test` green

### Task 2: `comments.Follow`, the poll held open

**Files:**
- Create: `go/internal/comments/follow.go`, `go/internal/comments/follow_test.go`
- Modify: `go/internal/comments/wait.go` (a pure extraction only, if taken),
  `go/internal/comments/doc.go`

- [ ] test first in `follow_test.go`, with a cursor-aware list that records the
      cursor each poll asked with, and `now` and `sleep`:
  - `TestAWindowWithAMarkedCommentIsHandedOver`: a new top-level `ai?` comment
    is handed over once, with `Since`, `Cursor` and `Noticed` set
  - `TestAWindowOfOnlyGdocsRepliesIsHeld`: one new reply opening with the mark
    in effect hands nothing over; so does one opening with each known mark
  - `TestGdocsReplyInAnOldMarkedThreadIsHeld`: a thread whose top comment is an
    old `ai?`, already handed over, gains a reply from gdoc: held
  - `TestAnEditedOldCommentIsHeld`: a changed `modified` on an old comment with
    no new item is held
  - `TestAnUnmarkedReplyIsHeldAndRidesInTheNextWindow`: an unmarked reply, then
    a later `ai!` reply; one window is handed over, holding both
  - `TestEveryMarkerCounts`: `ai?`, `ai!` and `ai:` each hand a window over;
    `none` does not
  - `TestAMarkedReplyAtTheCursorsInstantIsNews`: a marked reply created at
    `since.At` on a thread not in `since.Ids` is handed over
  - `TestAHandedOverWindowIsNotHandedOverAgain`: two polls see the same marked
    comment, one of them at the cursor's instant; handed over once
  - `TestAHeldPollAsksFromTheLastPrintedCursor` and
    `TestThePollAfterAWindowAsksFromItsCursor`, from the recorded cursors
  - `TestAHeldWindowDoesNotReadTheDocument`: three held polls, one read
  - `TestFollowRunsToItsDeadline`: a quiet script ends at the deadline with
    `Interrupted` false
  - `TestHeldItemsAtTheEndLeaveTheLastPrintedCursor`: a deadline and a signal,
    each with items held, end on the last cursor handed over
  - `TestASignalEndsTheFollowAndSaysSo`: a cancelled context ends it with
    `Interrupted` true, after any window already handed over
  - `TestAFailedPollEndsTheFollow`: a listing error ends it with the error, the
    polls so far and the last cursor handed over
- [ ] run them red
- [ ] `follow.go` as in Solution Overview: `Follow`, `FollowOptions`, `Window`,
      `Followed`, and the unexported `hasNewMarked`, which reuses the cursor's
      `newer`, `sameInstant` and `reported`
- [ ] if Wait's deadline context and `failed` classification move into a
      shared unexported helper, `wait_test.go` is untouched and every Wait test
      is green before and after; otherwise copy them and say so in the commit
- [ ] `doc.go`: a paragraph "The follow" stating the wake rule, the hold, the
      one document read, and that a window is handed over once, naming the tests
- [ ] `make test` green

### Task 3: the line writer, and only `comments` gets one

**Files:**
- Modify: `go/internal/emit/emit.go`, `go/internal/emit/emit_test.go`,
  `go/cmd/gdoc/main.go`, `go/cmd/gdoc/mcptools.go`,
  `go/cmd/gdoc/settings_test.go`, `go/cmd/gdoc/main_test.go`,
  `go/cmd/gdoc/read.go`, `go/cmd/gdoc/doc.go`

- [ ] test first:
  - `TestLinesWritesOneObjectPerLine` in emit_test.go
  - `TestOnlyFollowWritesLines`, a source rule: the `emit.Lines` constructor is
    named in main.go alone, and its write method only in read.go and
    follow.go
  - `TestTheMcpRouteHandsNoLineWriter`, a source rule: `mcpSend`'s call to
    `safeDispatch` passes nil
  - `TestAFollowWithNoLineWriterIsRefused`: `cmdComments` with `--follow` and a
    nil writer refuses by name, before any network
- [ ] run them red
- [ ] `emit.Lines`: one object per line on the writer it holds, and nothing else
- [ ] `safeDispatch`, `dispatching` and `dispatch` gain one `*emit.Lines`
      parameter; `run` passes one built from `out`; `mcpSend` passes nil; the
      fake in settings_test.go:68 takes the new parameter; `args` carries it
- [ ] `cmdComments` refuses `--follow` with no writer; nothing else reads it yet
- [ ] every existing test green, `TestStdoutCarriesOnlyJSONRPC` included
- [ ] `make test` green

### Task 4: `comments --follow` on the command line

**Files:**
- Create: `go/cmd/gdoc/follow.go`, `go/cmd/gdoc/follow_test.go`
- Modify: `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/read.go`,
  `go/cmd/gdoc/main.go` (the `Begin` argument and its comment),
  `go/cmd/gdoc/main_test.go` (the `runLines` helper),
  `go/cmd/gdoc/presence_test.go`, `go/cmd/gdoc/help_test.go`,
  `go/cmd/gdoc/testdata/pipe/help.stdout.golden`,
  `go/cmd/gdoc/testdata/pipe/help-comments.stdout.golden`,
  `go/cmd/gdoc/testdata/pipe/help-comments.stderr.golden`

- [ ] test first in `follow_test.go`, with `waitWire`, `newsListing`,
      `shortPolls`, `runJSONCtx` and a new `runLines`:
  - `TestFollowNeedsSince`, `TestFollowRefusesWait`, `TestFollowRefusesWitness`,
    `TestFollowRefusesMoreThanAnHour`, each naming `--follow` and what it
    refuses
  - `TestFollowWritesOneLinePerWindowAndTheEnvelopeLast`: every stdout line
    decodes as one object; window lines carry `line`, `since`, `cursor`,
    `noticed`, `items` and `more`, and no author or content; the last line is
    the envelope with `ok` and `waited`
  - `TestALineListsAtMostTwentyItems`: twenty-five new items give twenty and
    `more: 5`
  - `TestFollowsExitCodeFollowsTheLastLine`: a failed poll exits non-zero with
    `ok: false` last, carrying the last cursor handed over
  - `TestAnInterruptedFollowIsAnAnswer`: a context cancelled after one window;
    exit 0, `waited.interrupted` true
  - `TestTheWaitKeepsItsOwnWords`: `--wait 2h` is refused with today's bytes
- [ ] in `presence_test.go`: a `comments --follow` scene in `presenceScenes`
      with `done(t)`, so the flag-on and flag-off bytes match; and
      `TestAFollowsStartIsAWaitAndItsStopIsSaid`: the start has `wait: true`,
      and the end after a signal has `interrupted: true`
- [ ] run them red
- [ ] `parseWait(flag, text)`; the flag, its example and the refusals; a short
      `commentsFollow` in read.go around the existing `signal.NotifyContext`;
      shapes and conversion in follow.go; the `List` closure asks with the
      cursor it is given
- [ ] `presence.Begin`'s wait argument is `a.has("--wait") || a.has("--follow")`,
      and its comment in main.go says a follow is recorded as a wait;
      `TestBeginHasOneCaller` green
- [ ] the literal usage line in help_test.go:299 gains `[--follow <duration>]`;
      regenerate the three help goldens, the commit saying the diff is the
      `--follow` lines and the new example only
- [ ] `TestOnlyTheWaitTrapsTheSignal` green: follow.go imports no `os/signal`
- [ ] read.go is no longer than it was before this task
- [ ] `make test` green

### Task 5: the panel's proof for a long wait

**Files:**
- Modify: `presence/Tests/PresenceCoreTests/LiveSessionTests.swift`,
  `presence/Tests/PresenceAppTests/StopperTests.swift`,
  `presence/Sources/PresenceApp/Stopper.swift` (a comment only)

- [ ] test first, building events inline with `wait(from:to:run:)`:
  - `testAWaitLongerThanTheGapStaysLiveWhileItsProcessRuns`: a `comments` start
    with `wait: true`, no end, pid alive, 29 minutes old: in flight, and live
  - `testAFollowRestartedInsideTheGapIsOneSession`: an end at minute 29 with
    `interrupted` false and a new start 20 s later are one session
  - `testAStoppedFollowEndsTheSessionAtOnce`: the same end with `interrupted`
    true ends it at once
  - `testAReplyWhileTheWaitIsInFlightIsCounted`: a `reply` end inside a wait
    still in flight counts toward the session
  - in `StopperTests`: `testStopReachesALongWaitAfterTheProcessCheck`, the
    29-minute start, one SIGINT after every check holds
- [ ] run them: they are expected green on today's code; if any is red, the
      fix is in `LiveSession.swift` or `Stopper.swift`, named in the commit
- [ ] `Stopper.swift`'s comment points at the trap by name (`cmdComments` in
      read.go), not by line number
- [ ] no fixture file changes; `TestTheSwiftFixtureIsTheGoFixture` green
- [ ] `make app-test` and `make app-lint` green

### Task 6: the live loop under the Monitor tool

**Files:**
- Modify: `skills/gdoc-review/live.md`, `skills/gdoc-review/SKILL.md`,
  `go/cmd/gdoc/skills_test.go`

- [ ] test first in `skills_test.go`:
  - `TestTheLiveLoopFollowsUnderTheMonitorTool` reads `live.md` and finds the
    `--follow 29m` call, "Monitor tool", the restart when the follow ends, that
    a follow ended by a stop is never restarted, "Act on the items the line
    names", that a stop in the chat stops the Monitor task and prints the
    totals once, and the fallback `--wait 9m` loop, used when there is no
    Monitor tool or the binary refuses `--follow`; and it runs
    `skillProblems(skillCalls("live.md", src))`, so every call in `live.md` is
    checked against the table
  - `TestALiveReviewAsksForItsOwnSession` finds the own-session sentence in
    `SKILL.md`'s live pointer
- [ ] run them red
- [ ] `live.md`: a section before today's loop. Where the session has the
      Monitor tool, start `$GDOC comments <url> --since "$CURSOR" --follow 29m`
      under it with a 30-minute limit. A window line: read the window with
      `$GDOC comments <url> --since <its since>`, act on the items the line
      names and leave anything newer in that read to the next line, run Steps
      2 to 8, and keep its `cursor`. Items whose marker is `none` are listed in
      that window's report and never acted on. The last line:
      `waited.interrupted` true means print totals and stop; false means start
      the next follow from the last cursor; `ok: false` keeps today's retry
      rule. When the person says stop in the chat, stop the Monitor task, print
      the totals once, and do not restart on the envelope that follows. If the
      binary refuses `--follow`, use the loop below. Today's `--wait` loop
      stays, word for word, as that fallback
- [ ] `SKILL.md`'s live pointer: when this session already holds other work,
      ask the person to open a second Claude Code session in the same hub
      folder and start the live review there
- [ ] `TestTheLiveLoopSaysWhichCursorsItKeeps`,
      `TestEverySkillNamesOnlyCommandsAndFlagsTheBinaryHas` and
      `TestEverySkillBodyFitsTheSpecBudget` green
- [ ] `make test` green

### Task 7: the five skills open on `help --brief --json`

**Files:**
- Modify: `skills/gdoc-review/SKILL.md`, `skills/gdoc-export/SKILL.md`,
  `skills/gdoc-restyle/SKILL.md`, `skills/gdoc-publish/SKILL.md`,
  `skills/gdoc-align/SKILL.md`, `go/cmd/gdoc/skills_test.go`

- [ ] test first: `TestEverySkillOpensOnTheBriefHelp` finds `$GDOC help --brief
      --json` as each skill's setup call, followed by the fallback sentence
- [ ] run it red
- [ ] each setup: `$GDOC help --brief --json`, and one short sentence: "If it
      is refused, run `$GDOC help --json` instead." What each skill reads from
      the object, the mark and the update notice, is unchanged; the four `help
      <command> --json` calls stay; no `needs:` line changes
- [ ] `TestEverySkillPassesJSONToHelp`, `TestEverySkillBodyFitsTheSpecBudget`
      (gdoc-publish is near its ceiling) and `TestNoSkillNeedsAReleaseNobodyCut`
      green
- [ ] `make test` green

### Task 8: Verify acceptance criteria

- [ ] every test named in Tasks 1 to 7 exists and is green
- [ ] `git diff --name-only main...HEAD` lists none of `docs/v2/DECISIONS.md`,
      `docs/v2/SPEC.md`, `docs/v2/PLAN.md`, `go/internal/comments/wait_test.go`,
      and nothing under `go/internal/mcp`, `go/internal/chat`,
      `go/internal/presence`, `go/internal/guard` or `go/internal/auth`
- [ ] if wait.go changed, the diff is the pure extraction Task 2 allows, and
      every Wait test is green
- [ ] no `--wait` test lost a line: `git diff main...HEAD --
      go/cmd/gdoc/read_test.go` removes nothing
- [ ] `presence/Sources` changed only in `Stopper.swift`'s comment, unless
      Task 5 found a red test, which its commit names
- [ ] the only goldens that moved are the three help goldens
- [ ] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
      `make app-test`, `make app-lint`, `make platform-test`, `make team-check`

### Task 9: [Final] Update documentation

- [ ] every `doc.go` touched names a test for every rule it states, or
      `TODO(test)`: `comments`, `cmd/gdoc`, `emit`
- [ ] `CLAUDE.md`, two invariant bullets and nothing else, with the file's line
      count unchanged: "One JSON object reaches stdout" says `comments
      --follow` writes one object per line with its envelope last, naming
      `TestOnlyFollowWritesLines`; "The menu bar app runs six argv lists,
      signals one wait" says a `comments --wait` or `--follow`. Tighten the
      wording inside each bullet so neither grows. `TestCLAUDEmdIsUnderTheCeiling`
      green in this step
- [ ] `docs/guide/reading.md`: the usage line at :14 gains `[--follow
      DURATION]`, and a paragraph after the `--wait` section (:109-126) shows a
      window line, the envelope last, and the hold; `docs/guide/presence.md`
      :126 says a live watch is the stretch of `--wait` or `--follow` runs, and
      nothing else there changes
- [ ] `go doc` over each touched package prints its package comment and
      nothing a file comment meant for itself
      (`TestOnlyThePackageCommentReachesGoDoc` green)
- [ ] move this plan to `docs/plans/completed/`

## Post-Completion

These are the lead's, with Nail.

**Nail's hand check, once, after the release and `gdoc update`.** Open a
second Claude Code session in the hub folder and start a live review of one
document.
- Write an `ai?` comment and see it answered.
- Watch the session stay quiet after gdoc's own reply.
- Press Stop on the panel's live row and see the review end without a restart.
- Start it again, then close the session in the middle of a follow: the gdoc
  process exits and the panel's row ends.

**The stopwatch, at the end of the week.** The lead reads `comments` on the
paired documents, read only, for written and answered, and the live sessions'
transcripts for noticed, and writes one table into `docs/team/measure.md`: each
marked comment's three times, split into comments written while a follow ran
and while none did, and the tokens per live hour of the live sessions, by the
meter, against 8.6M.

**The measurement of the run.** `.ralphex/board/meter.py ../gdoc-m32` into
`docs/team/measure.md`.

**The release.** `make release BRANCH=gdoc-v2-m32-the-quiet-watch` stops at a
green PR. The merge and the tag wait for Nail's word. This adds a flag and a
mode, so it is a minor version. The skills fall back on an older binary, so no
`needs:` bump is required; whether to raise it later is a separate choice.
