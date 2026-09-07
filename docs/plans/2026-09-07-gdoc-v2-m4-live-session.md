# gdoc v2 Milestone 4: The live session

## Overview

A review session can stay live. Nail opens it on one document and it keeps watching that document's comments until he stops it: a marked comment is seen within seconds, answered or carried out or proposed the same way the one-shot review does, and the conversation stays in the document. Nothing runs on its own; liveness ends with the session.

The binary grows one thing: `gdoc comments <url> --since <cursor> --wait <duration>`. Inside that one call it polls Drive every ten seconds and exits the moment there is activity after the cursor, or at the deadline with an empty window. Still one JSON object, still exit 0 if and only if `ok`, still no state kept anywhere. The skill loops on it. A quiet document costs no model turns, because the waiting happens in the binary.

The line M2 drew holds. The binary reports the threads that arrived and the next cursor; the skill reads each thread and decides whether it is work, exactly as it does today. The live mode is a section of the review skill, not a second skill and not a second binary.

Decisions Nail took on 2026-09-07, before this plan: the wait is the binary's (`--wait`), because a Claude Code session cannot wake more often than once a minute and every idle tick would spend a turn; one live session watches one document, the link he gave, and the hub-wide session waits for M8; on a partial poll the skill acts on the threads it could fully read and reports the rest; launch and stop are ordinary Claude usage (the word "live" in the request, Ctrl-C or "stop" to end), with no duration cap; the acceptance run uses Nail's own account commenting from the browser; the 🤖 receipt on a colleague's marked comment names who asked.

Spec: `docs/v2/SPEC.md` ("Reading comments", "Skills", "How a comment reaches the agent"). Master plan: `docs/v2/PLAN.md`, section M4. Predecessor: `docs/plans/completed/2026-09-07-gdoc-v2-m3-writing.md`. Decision record: `docs/v2/DECISIONS.md`, "2026-08-29. Live review is a session, not a service."

## Context (from discovery)

- Files and components involved: `go/internal/comments` (gains `Wait`), `go/cmd/gdoc/read.go` (`comments` gains `--wait`, and `main.go` gains an interrupt context), `go/internal/live` (an opt-in wait test), `skills/gdoc-review/SKILL.md` (a live section), `README.md`, `CLAUDE.md`, `docs/v2/PLAN.md`, `docs/v2/SPEC.md`.
- What M2 and M3 already give this milestone: `comments.Fetch(ctx, s, id, since)` narrows the listing to activity strictly after the cursor and keeps the ids reported at the cursor's own instant; `comments.NextCursor(prev, threads)` advances only on news; `comments.Threads(raw, d)` joins the Docs ranges; `docs.Fetch`; the guard carries `GET {id}` and `GET {id}/comments` at `LevelSuggest`, and `POST {id}/comments` (which the live test uses to write the comment it then waits for); the four writers and the review skill's Steps 1 to 8, which the live loop reuses unchanged.
- Related patterns: `cmd/gdoc/write_test.go`'s `fakeWire` with `once: true` answers, which is how a sequence of polls is scripted; `internal/live/live_test.go`'s `createSubject` and `trashSubject`; the `session` interface in `read.go`; `parseArgs` and its strict flag rules.
- The Claude Code fact that shaped decision 1: its scheduled wake-ups clamp to 60 seconds at minimum, and its Bash tool waits at most 600 seconds on one command. So the skill's wait is under ten minutes per call, and the binary polls inside it.
- Dependencies identified: none new. Standard library plus `goccy/go-yaml`.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own commit.
- Make small, focused changes.
- **CRITICAL: every task MUST include new or updated tests** for the code it changes. Success and failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test` runs `-race`; keep it green.
- **CRITICAL: update this plan file when scope changes during implementation.**
- **CRITICAL: facts only in Go.** `Wait` returns what arrived and how long it looked. Whether a thread is work, whether a poll that half failed is safe to act on, and when to stop are the skill's. The grep from M3 (`handled`, `accepted`, `rejected`, `matters`, `drift`, `should`, `decide`) runs again in Task 5.
- **CRITICAL: the binary stays one-shot.** `--wait` is one call that ends. It writes nothing, keeps nothing, and the cursor it prints is the only thing that carries to the next call.
- No em dashes in any text this plan produces, code comments, commit messages and the skill included.

## Testing Strategy

- **Unit tests**: `comments.Wait` over a fake reader that answers a scripted sequence of polls, with an injected clock and sleeper so a test of a ten-second interval takes no time. `cmd/gdoc` tests over `fakeWire` with `once: true` answers: an empty window then news, a deadline with nothing, a failed poll, an interrupt.
- **Guard tests**: none needed. The wait makes the two reads `comments` already makes, on the same URLs, and the boundary test's allowlists do not change.
- **Live test**: `go/internal/live` gains `TestLiveWaitSeesANewComment` behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`. It creates a document in the test folder, takes a baseline cursor, starts a wait, posts a marked comment into the document through Drive while the wait is running, and asserts the wait returns that one thread with an advanced cursor before its deadline. It writes only to the document it created and trashes it.
- Coverage standard: every exported function under `go/internal/` has a test; `comments` and `cmd/gdoc` stay at or above 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build`

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

The one-shot review is unchanged and the live mode is a loop around its tail. Step 1 reads the document and the threads once and keeps the cursor `comments` printed. Steps 2 to 8 run on those threads as today. Then, in live mode, the skill calls `comments <url> --since <cursor> --wait 9m` and blocks. The binary polls inside that call and returns the first non-empty window, or an empty one at the deadline. On an empty window the skill calls again with the same cursor and says nothing. On threads it runs Steps 2 to 8 over exactly those threads, prints the same report the one-shot review prints, replaces its cursor with the one the answer carried, and calls again. Nail ends it with Ctrl-C or by saying stop, and the skill prints the session's totals.

Key design decisions and why:

- **The wait is the binary's, the loop is the skill's.** SPEC.md says the binary makes one cheap call and exits; the call is now allowed to take up to a deadline, and it still exits with one object. The alternative, a model turn per poll, costs tokens on every quiet tick and cannot poll faster than once a minute. Nail's call, 2026-09-07.
- **The first non-empty window ends the wait.** Latency is the point of a live session. A window is the threads `Fetch` narrows to, so gdoc's own 🤖 reply arriving after a poll does end the wait, and the skill reads it as a receipt and goes back to waiting: that is the ledger rule from SPEC.md, and it costs one short turn.
- **A failed poll ends the wait with `ok: false`.** A Docs read or a Drive listing that fails is not something the binary retries silently: the skill sees it, says so, and calls again. Decision 3-a is about the threads inside a successful answer: a thread that came back with `range: null` is fully read for answering and not for proposing, and the skill says which.
- **An interrupt is an answer, not a crash.** Ctrl-C during a wait makes the binary print the envelope it has, `ok: true` with no threads, the same cursor, and `interrupted: true`, and exit 0. The output contract holds under the one signal a live session sends every time it ends.
- **One document.** The policy holds the one id the link named, as every command's does. Walking the hub is M8's.
- **The receipt names who asked.** On a marked comment whose author is not the account gdoc is signed in as, the 🤖 receipt opens with the action and names the author: "🤖 Done, asked by William Mejia: ...". A fact the thread already shows, made visible where the delegation happened. Identity is still never a gate.

## Technical Details

**Principles.** Serves: 1 (one binary, no watcher, nothing installed to run on its own), 3 (the wait makes the reads the guard already carries and nothing else), 4 (the thread carries the receipt, the terminal carries the record). Declines: a daemon, a webhook, Apps Script, a browser extension, all rejected in DECISIONS.md 2026-08-29.

**Tech stack.** Go 1.27, standard library plus `goccy/go-yaml`. The skill is markdown under `skills/`, symlinked.

**Global constraints:**

- Every command: exactly one JSON object on stdout, exit 0 iff `ok`. The binary never prompts and never reads stdin. A wait that is interrupted still prints one object.
- The guard is opened with exactly the one document id. The wait adds no URL, no parameter and no header the `comments` command does not already send.
- `--wait` requires `--since`. The first read has no cursor and is the baseline; a wait with no cursor would return everything at once, which is the one-shot read under another name, so it is refused naming the missing flag.
- `--wait` takes a Go duration (`9m`, `90s`). Zero, negative, unreadable, or over one hour is refused. The interval is a constant, ten seconds, inside the spec's five to fifteen; a test overrides it through an unexported variable, never a flag.
- Nothing is written anywhere during a wait, on disk or in the config dir. `--witness` with `--wait` applies to the window that ends the wait, as it applies to a one-shot listing.
- Commit messages: `feat(v2): ...`, `fix(v2): ...`, `test(v2): ...`, `docs: ...`. All commits from the repo root.

**`comments.Wait`.** In `go/internal/comments/wait.go`:

```go
type WaitOptions struct {
    Interval time.Duration            // between polls; the command passes waitInterval
    Deadline time.Duration            // how long to look before answering empty
    Fetch    func(ctx) (*docs.Document, []RawComment, error) // one poll: the Docs read and the listing, as the command makes them
}
type Waited struct {
    Threads   []Thread
    Unplaced  []string
    Cursor    *Cursor
    Polls     int
    Waited    time.Duration
    Interrupted bool
}
func Wait(ctx context.Context, since *Cursor, o WaitOptions) (Waited, error)
```

The loop: poll; if the poll errored, return the error with `Polls` so far; join the raw comments to the document with `Threads`; if any thread came back, return it with `NextCursor(since, threads)`; if the deadline has passed, return empty with `since` unchanged; if `ctx` is done, return empty with `Interrupted: true` and `since` unchanged; otherwise sleep the interval (through a sleeper the test replaces) and poll again. The first poll happens at once, not after an interval. `Fetch` is a parameter so the package still holds no session and no URL: the command builds the two reads it already builds, and the test hands in a script.

**`comments --wait` on the envelope.** `cmdComments` gains the flag, checks it beside `--since`, builds the `Fetch` closure from `docs.Fetch` and `comments.Fetch` on the reach's session, and calls `Wait` with a context that `main` cancels on `SIGINT` and `SIGTERM` (`signal.NotifyContext`, in `main.go`, handed to `dispatch`). The commands that do not wait ignore the context as they do today. Data gains three fields:

```jsonc
// gdoc comments <url> --since <cursor> --wait 9m
{
  "document_id": "...", "title": "...", "tabs": 1, "multi_tab": false,
  "cursor": "<next or same>",
  "threads": [ ... ],            // the window that ended the wait, or empty
  "waited": { "polls": 12, "seconds": 118, "interrupted": false }
}
```

`waited` is present only when `--wait` was given. `interrupted: true` comes with an empty `threads` and the cursor handed in.

**The live section of the skill.** After Step 8, a section "Live mode":

- It is on when Nail's request says "live". Everything above it runs first, once, and its report is printed.
- The loop, as one bash block: `$GDOC comments <url> --since "$CURSOR" --wait 9m`. Read `ok`. On `ok: false`: print the error, wait thirty seconds, call again with the same cursor; three failures in a row, stop and say so. On `threads: []`: call again, print nothing. On threads: run Steps 2 to 8 over those threads, print the Step 8 report, set `CURSOR` to the answer's cursor, call again.
- Nine minutes because the tool that runs the command waits ten at most. The binary would take an hour; the skill never asks for one.
- What a returned window contains: everything with activity after the cursor, gdoc's own replies included. A window that is only gdoc's receipts is not work; say nothing and call again. A thread with `range: null` can be answered and cannot be proposed into; say which when it matters.
- A colleague's marked comment is work, and the receipt names them. Nail's own is not named.
- Before acting on a marked comment that could be old (a session restarted, a cursor from a previous run pasted in), the check from Step 2 runs: suggestions and the note's provenance, then the missing receipt rather than the action twice.
- Stop: Ctrl-C, or Nail says stop. `interrupted: true` in an answer is Nail stopping, not a failure. Then print the session totals: windows seen, threads answered, carried out, proposed, files changed in the hub, and unmarked follow-ups reported, in the shape of the Step 8 report.
- Dry run applies to live mode as it does to the rest: print instead of post, and say so.

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): `Wait`, the flag, the interrupt, the skill's live section, the live wait test, the documentation.
- **Post-Completion** (no checkboxes): the live acceptance run with Nail commenting from the browser, and the first live session on a real document.

## Implementation Steps

---

### Task 1: `comments.Wait`: poll until news, the deadline, or an interrupt

**Files:**
- Create: `go/internal/comments/wait.go`, `go/internal/comments/wait_test.go`

**Interfaces:**
- Consumes: `Threads`, `NextCursor`, a `Fetch` closure handed in by the caller.
- Produces: `WaitOptions`, `Waited`, `Wait(ctx, since, o)` as in Technical Details. The sleeper is an unexported package variable, `sleep = time.Sleep`-shaped but taking a context so an interrupt cuts a sleep short, replaced in tests.

- [x] write the failing tests: the first poll runs at once with no sleep before it; an empty poll then a poll with one thread returns that thread, `Polls: 2`, and a cursor advanced to the thread's instant; a poll that errors returns the error and the polls so far; nothing before the deadline returns empty with the cursor handed in unchanged; a cancelled context returns empty with `Interrupted: true` and no further poll; a window that is only a 🤖 reply is still returned (facts only: the skill reads it as a receipt); `Unplaced` carries the ids `Threads` could not place
- [x] run the tests and watch them fail
- [x] implement
- [x] run the tests, gofmt, vet: green
- [x] commit: `feat(v2): comments.Wait polls until activity, a deadline or an interrupt`

➕ Three tests beyond the list, each for a case the implementation had to decide: the sleep is clamped to what is left of the deadline; a context already cancelled polls nothing; and a poll that failed because the interrupt cut the request short is reported as the interrupt rather than as a failed read. `Wait` also refuses options it cannot run (no poll, a non-positive interval, a negative deadline), naming the field.

---

### Task 2: `comments --wait` on the envelope, and an interrupt is an answer

**Files:**
- Modify: `go/cmd/gdoc/read.go`, `go/cmd/gdoc/read_test.go`, `go/cmd/gdoc/main.go`, `go/cmd/gdoc/main_test.go` (or wherever `dispatch` is tested)

**Interfaces:**
- Consumes: `comments.Wait`, `parseArgs`, `signal.NotifyContext`.
- Produces: the `--wait` flag; `commentsData.Waited *waitedData` with `polls`, `seconds`, `interrupted`; `dispatch` takes a `context.Context` that `main` cancels on `SIGINT` and `SIGTERM`. Only `comments --wait` reads it today.

- [x] write the failing tests: `--wait` without `--since` is refused naming `--since`; `--wait 0`, `--wait -1m`, `--wait soon` and `--wait 2h` are each refused naming the value; over `fakeWire` with `once` answers, an empty listing then a listing with news returns the news and `waited.polls: 2`; a listing that fails on the second poll returns `ok: false` with the error and the polls so far in `data`; a cancelled context returns `ok: true`, empty threads, the same cursor and `waited.interrupted: true`; `--witness` with `--wait` witnesses the window that ended the wait; the usage line is unchanged (no new command)
- [x] run the tests and watch them fail
- [x] implement, with the poll interval as an unexported variable the tests shorten
- [x] run the tests, gofmt, vet, `make build`: green; the by-hand run against a real document is manual (skipped, no account or document id in an unattended run; the built binary was smoke tested on both refusal paths, and Post-Completion carries the live run)
- [x] commit: `feat(v2): comments --wait polls inside one call and exits on the first news`

➕ Four tests and two decisions beyond the list. A run without `--wait` carries no `waited` object at all, because reporting `polls: 1` on a call that never waited says the binary polls when it does not. A wait that reaches its deadline is asserted beside the interrupt, so the two quiet endings are told apart by the flag rather than by the empty window they share. An empty window is not witnessed: the export would be one more request for no question, and on the way out of an interrupted session it would fail on the cancelled context and warn about a read nobody made. `--wait 0s` is refused beside `0`, since `time.ParseDuration` reads both.

➕ `commentsBase` and `commentsResult` are shared by the one-shot listing and the wait, so a window cannot come back described one way and a listing another. `commentsBase` answers with the id the run was given when no poll read the document, which is a wait interrupted before its first read.

---

### Task 3: The review skill's live mode

**Files:**
- Modify: `skills/gdoc-review/SKILL.md` (a "Live mode" section after Step 8, the description line mentions live, and the receipt rule in Step 6 names a colleague who asked)

- [x] write the section as specified in Technical Details: trigger, the loop block, the nine-minute reason, what a window contains, receipts on gdoc's own replies, `range: null`, the colleague receipt, the old-comment check, stop and totals, dry run
- [x] update Step 6 and Step 7 (the receipt and the proposal's `why`): when the marked comment's author is not the account gdoc is signed in as, the 🤖 text names them ("asked by <name>")
- [x] update the frontmatter description so the skill is chosen for "review this document live" as well as the one-shot request
- [x] read the whole skill once top to bottom for a sentence the live section makes false, and fix it
- [x] commit: `feat(v2): the review skill can stay live on one document`

---

### Task 4: The live wait test

**Files:**
- Modify: `go/internal/live/live_test.go`

**Interfaces:**
- Consumes: `createSubject`, `trashSubject`, `comments.Fetch`, `comments.NextCursor`, `comments.Wait`, the guard's `POST {id}/comments`.

- [x] write `TestLiveWaitSeesANewComment` behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`: create a document in the test folder with one sentence; read its threads once and take the baseline cursor; start `Wait` with a 90-second deadline and a 5-second interval in a goroutine; after one interval post `ai? live wait test` as a comment through Drive's `comments.create`; assert the wait returns one thread carrying that content with marker `ai?`, an advanced cursor, and `Polls` at least 2; then call `Wait` once more with the new cursor and a 15-second deadline and assert it returns empty with the cursor unchanged; trash the document
- [x] run it once on this machine (skipped, not automatable: an unattended run has no account to sign in as, and the run creates a real document in Nail's Drive. Post-Completion carries it, and the log lines it will print are written)
- [x] commit: `test(v2): the opt-in live wait sees a comment posted while it waits`

➕ Two things beyond the list. The writer runs on the goroutine and the wait on the test's own, rather than the other way round, so the wait's answer and its error stay off a channel; the observable order is the same, the comment is written one interval into a running wait. And the poster gets its own `gapi.Session` on the same policy: a Session refreshes its own token in place, so sharing one across two goroutines is a race `-race` would report, and the policy is the part that is mutex guarded. The test also asserts the wait answered before its deadline and was not interrupted, which is what tells the news apart from the two quiet endings that share an empty window, and it logs the thread as unplaced: a comment created through Drive carries no anchor, so this is what the skill's `range: null` case looks like against Google.

---

### Task 5: Verify acceptance criteria

**Files:**
- Modify: whatever the checks below break.

- [x] verify PLAN.md M4's four items are covered: continuity across polls (the cursor from each answer feeds the next, Task 1 and 2 tests), partial-read behaviour (a failed poll is `ok: false` and the skill says so and does not report clean, Task 2 and 3), colleague `ai!` (Task 3's receipt rule), clean cancellation (Task 2's interrupt)
- [x] verify the binary kept no state: grep `internal/comments/wait.go` and `cmd/gdoc/read.go` for any write to disk or to the config dir; there must be none
- [x] verify no judgement leaked into Go: grep `wait.go` and the `comments` command for `handled`, `accepted`, `rejected`, `matters`, `drift`, `should`, `decide`; each hit is a fact with a different name or a defect
- [x] verify the boundary test still passes and its allowlists did not change
- [x] run the full suite with `-race`, gofmt, vet, `make build`, `make dist`
- [x] verify coverage: every exported function under `go/internal/` has a test, `comments` and `cmd/gdoc` at or above 80%
- [x] commit any fixes this task made

**What the six checks found.**

- **The four M4 items.** Continuity: `TestWaitReturnsTheFirstWindowWithActivityAndAdvancesTheCursor` and `TestWaitAtItsDeadlineIsEmptyWithTheCursorItWasGiven` in `internal/comments`, with `TestAWaitEndsOnTheFirstWindowWithNewsInIt` on the envelope, so the cursor an answer carries is the one the next call is given whether or not the window had news. Partial read: `TestWaitCarriesAFailedPollOutWithThePollsSoFar` and `TestAFailedPollEndsTheWaitAndSaysHowManyItMade`, plus the skill's rule to print the error and not report clean. Colleague `ai!`: `skills/gdoc-review/SKILL.md`, "Name a colleague who asked". Cancellation: five tests, `TestAnInterruptEndsTheWaitAsAnAnswer` through `TestAnInterruptedWaitIsAnAnswerAndNotAFailure`.
- **No state.** The only disk write anywhere in `read.go` is `recordSnapshot`, which has exactly one caller, `suggestions --md`. `cmdComments` cannot reach it, and `wait.go` opens no file at all.
- **No judgement.** Every field on `WaitOptions` and `Waited` is a duration, a count, a list or a flag. The one grep hit outside a comment is `read.go:218`, the error naming a flag given twice, which is a refusal rather than a verdict.
- **The boundary.** All ten tests pass and `go/boundary/boundary_test.go` has not been touched since M2, so neither allowlist moved. M4 added no import and no module.
- **The suite.** `-race` green across 21 packages, gofmt and vet clean, `make build` and `make dist` build all three targets.
- **Coverage.** `comments` 96.9%, `cmd/gdoc` 85.6%, both above the 80% bar.

➕ One fix, in code M4 did not write. The exported-function audit over `go/internal/` found a single gap: `sentError.Unwrap` in `internal/gapi`, with no test. The writer packages ask `errors.As(err, &sent)`, which matches the type itself and never walks the chain, so an `Unwrap` returning nil would pass every existing test while silently breaking the first caller that asks `errors.Is` what a sent failure actually was. `TestASentErrorStillCarriesItsCause` closes it, and it was watched failing against a broken `Unwrap` before it was kept. `internal/gapi` is at 91.9%.

---

### Task 6: [Final] Update documentation

**Files:**
- Modify: `README.md`, `CLAUDE.md`, `docs/v2/PLAN.md`, `docs/v2/SPEC.md`

- [x] update `README.md`: `comments --wait`, the `waited` fields, and how a live review session is started and stopped
- [x] update `CLAUDE.md` under "The three read commands": `--wait` requires `--since`, the interval is ten seconds and a constant, the first non-empty window ends the wait, an interrupt is `ok: true` with `interrupted: true`, a failed poll is `ok: false`, and the loop is the skill's; under "The binary prints facts": `Waited` carries counts and a flag and no verdict
- [x] update `docs/v2/SPEC.md` "How a comment reaches the agent" with a dated correction: the binary polls inside one call up to a deadline and exits on the first activity, rather than one call per poll, and why (Nail's decision, 2026-09-07)
- [x] update `docs/v2/PLAN.md`: mark M4 done with the date, record what it leaves for M8 (the hub-wide live session over every paired document)
- [x] run the full test suite one more time
- [x] commit: `docs: M4 lands, the live session written down`
- The harness moves this plan to `docs/plans/completed/` when the run finishes.

## Post-Completion

*Items needing a real account or a person. No checkboxes.*

**Manual verification:**

- Run the live wait test once on Nail's machine: `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`. It creates one document in the test folder and trashes it.
- The acceptance run, Nail's decision 5: copy one of his documents into the test folder, start `/gdoc-review <copy url> live` in the hub folder, and from the browser add an `ai?` comment, then an `ai!`, then an unmarked reply to an answered thread, a minute or so apart. Watch each one land as a reply, a hub edit with its receipt, and a reported-not-acted line, within seconds of being written. Then Ctrl-C and read the totals. The colleague-`ai!` path is exercised by the skill's rule and Task 3's wording; a second account is not available for this run, and a later session with a colleague commenting is the first real proof of that line.
- Then the first live session on a real document, with Nail watching.

**Known and out of scope for this plan:**

- One live session watches one document. Watching every paired document in the hub is M8, with the align skill.
- The wait polls Drive and Docs both on every tick, because the ranges come from the Docs read. A cheaper first tick that lists comments alone and reads the document only when something arrived is an optimisation for later, if the quota ever matters.
- Suggestions made in the document during a live session (somebody accepting or rejecting one of gdoc's proposals) are not watched; `suggestions --md` on the next one-shot run reports them as `gone_since_last_look`.
