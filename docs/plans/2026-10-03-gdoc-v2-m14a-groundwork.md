# gdoc v2 Milestone 14, run 1: the groundwork, plan

2026-10-03. The first of the task lists for the specification at
`docs/plans/2026-10-02-gdoc-v2-m14-chat.md`. The spec holds the nineteen
decisions Nail took, the server's shapes, and the seventeen scenarios that are
the acceptance list. This file holds the work that no measurement can change,
one commit per task, for ralphex.

M14 runs in more than one ralphex run, on Nail's call of 2026-10-03. The spec
puts twelve measurements in Claude Desktop before any server code, and a
measurement that contradicts the spec stops the plan. ralphex cannot wait
halfway through a run for a person to measure, so the work is cut where the
measurements fall:

| Step | Holds | When |
|---|---|---|
| run 1, this file | the firm's domain out, `propose` without the probe, the `review.md` split, the two-step login, the token race, contexts threaded through the six chat commands, `notice` returning its line | now |
| the tag sitting | the two skills stop passing `--folder` and move to `needs: v2.8.0`, in the same sitting as `make tag VERSION=v2.8.0` | by hand, after run 1 merges |
| the spike | the twelve measurements, with a throwaway stub server outside the tree, recorded in MEASURED.md | beside run 1, by Nail with a Claude Code session |
| run 2 and later | `internal/mcp`, `gdoc mcp`, the tools, the labelled text, the holds, the extension, `update --desktop`, released as v2.9.0 | after the spike, from a plan written with the measured values |

Run 1 is released alone as v2.8.0, Nail's call of 2026-10-03, so colleagues
stop making a throwaway probe document on every proposal now. That moves
decision 19's number: `gdoc mcp` ships in v2.9.0, and everything else decision
19 says holds for that release. Task 2 records it.

A task names the decision and the scenarios it serves. A task that finds the
spec wrong stops and says so rather than choosing.

## Principles

Serves: 1, it runs on someone else's machine. Everything here makes the one
binary ready to serve a second front door without a second copy: commands that
take the context they are handed (Task 9), a login that can be started and
finished in two calls (Task 7), a token file two processes can share (Task 8).

Strains: 3, uncertainty never resolves toward the destructive answer. Task 3
takes the probe out of `propose`, so a day when Google ignores
`writeMode: SUGGEST` lands one unconfirmed proposal before the read-back sees
it. Nail accepted that on 2026-10-02 (decision 11). The bound is per call: the
run stops at the first proposal either read-back cannot confirm (Task 4).

## Overview

| Piece | Today | After run 1 |
|---|---|---|
| the firm's domain | twenty lines in tests, two plans and DECISIONS.md | none; a boundary test fails on any file git does not ignore that holds it |
| `propose` | probe on a throwaway document in `--folder`, then the proposals | no probe; `--folder` optional and ignored with a warning; the run stops at the first proposal either read-back cannot confirm; a lost batch answer is `sent: false, outcome: "unknown"` and stops the run |
| `gdoc probe` | run inside every `propose` | a manual command only, unchanged |
| `gdoc-review`, `gdoc-align` | pass `--folder`, read `enrolled`, `needs: v2.7.0` | unchanged in run 1; they keep working, because the flag is accepted and warned about. The tag sitting changes them |
| `skills/gdoc-review/SKILL.md` | 19,951 characters, rules and calls together | setup, calls, live mode and its step headings; the rules in `review.md`, with no call |
| `auth.Login` | one call that prints, waits, exchanges and saves | `StartLogin` and `Pending.Wait`, with `Login` the two in a row; the browser page says "Signed in" only after the save |
| a token refresh | saves over whatever is in the file | looks at the file first, and never saves an old login over a newer one |
| `read`, `comments`, `suggestions`, `reply`, `annotate`, `propose` | five drop their context for `context.Background()` | all six take the context they are handed; a write is never cut between its send and its read-back |
| `notice` | writes its line to stderr | returns the line; `help` writes it |

## Decisions

In the spec, numbered 1 to 19 under "Decisions Nail took, 2026-10-02". This
plan repeats none of them. Run 1 serves decisions 4 (the two-step login and the
saved-then-signed-in page only), 9 (the split only), 11, 13 (the split of
`notice` only), 15, 17 (the domain test only) and 19 (the `--folder` half, and
its version number moved to v2.9.0).

Two calls Nail took on 2026-10-03, after the plan review:

- **Run 1 is tagged v2.8.0, and the skill edits land in the tag sitting**,
  not inside ralphex. `TestNoSkillNeedsAReleaseNobodyCut` refuses a `needs`
  line above the version in `.claude-plugin/plugin.json`, and only `make tag`
  moves that file, so a `needs: v2.8.0` committed inside the run would turn
  `make test` red. `gdoc mcp` ships as v2.9.0.
- **A lost answer carries `sent: false` and `outcome: "unknown"`.** `sent`
  stays a plain bool on every entry, because its documented meaning is "gdoc got
  no answer saying it landed", which is exactly this case. `outcome` appears on
  that entry only. The spec's words "rather than `sent: false`" mean the entry
  does not stop at `sent: false`; they do not remove the field.

The spec's DECISIONS.md section asks for one entry with eighteen rows. Run 1
writes the rows its own code changes, in an entry dated 2026-10-02 (Task 2).
Run 2 writes the server's rows in a second entry, after the spike, so no row
records a decision a measurement may still overturn.

## Context (from discovery)

- **The probe in `propose`.** `go/cmd/gdoc/write.go:200` `cmdPropose` requires
  `--folder` at `:205`, grants `p.AllowCreateIn(probeFolder)` at `:234`, and
  `runPropose` at `:244` runs `probe.Run` at `:265` and refuses on
  `!report.Enrolled` at `:273`. `proposeData.Probe` at `:195`. The table entry
  is at `go/cmd/gdoc/commands.go:253`, with `--folder` `needRequired`.
- **Tests that answer the probe first.** `proposeAnswers` (probe, then the
  proposal) is used in `go/cmd/gdoc/paired_test.go` (8 places),
  `propose_block_test.go` (10), `propose_markers_test.go` (4) and
  `write_test.go`. Every one changes when the probe goes.
- **The proposal loop.** `runPropose` at `write.go:286-296` stops only when
  `propose.Apply` returns an error. `propose.Checks` at
  `go/internal/propose/propose.go:224` holds `SuggestionsInline`,
  `PreviewWithoutSuggestions` and `DocxAnchored`. `Verify` at
  `go/internal/propose/verify.go:31` leaves a check false when its read fails,
  with a warning, which is what "a read-back that could not be made counts as
  false" needs.
- **A lost answer.** `propose.send` at `propose.go:365` returns an error when
  nothing was written, and a warning when the server took the batch and its
  2xx answer could not be read (`sentAnyway` at `:423`). The session marks only
  that second case: `mark` at `go/internal/gapi/session.go:412`. A guard
  refusal (`go/internal/guard/policy.go:460`) is a plain error that comes back
  through `client.Do` exactly like a dropped connection, a dial failure or a
  TLS handshake timeout, so nothing today says whether the request was
  written. The package comment at `session.go:40-55`, "What is not marked is
  three cases", says widening the mark is Nail's decision and carries a
  `TODO(test)`. Both kinds of proposal call `send`: `propose.go:327` and
  `blockapply.go:114`.
- **`proposalReport`** at `write.go:176`. Its comment already says `sent: false`
  means "gdoc got no answer saying it landed", and names the 5xx case as the
  third case.
- **Contexts.** `context.Background()` at `read.go:350` (`cmdRead`), `:731`
  (`cmdSuggestions`), `write.go:133` (`cmdReply`), `:245` (`runPropose`) and
  `annotate.go:173` (`runAnnotate`). `cmdComments` at `read.go:483` already
  takes `ctx`. The table entries hand `_ context.Context`.
- **The login.** `go/internal/auth/login.go:170` `Login` listens, prints,
  `WaitCode`, exchanges on `context.Background()` and saves.
  `go/internal/auth/loopback/loopback.go:50` `callback` writes "Signed in. You
  can close this tab." at `:71`, before the code is exchanged, and
  `flushThenAnswer` at `:79` exists so the page is out before the listener
  closes.
- **The refresh.** `go/internal/gapi/session.go:427` `refresh` calls
  `Token.Refresh` then `auth.Save` with no look at the file in between.
  `auth.Refresh` at `go/internal/auth/auth.go:192` keeps the old refresh token
  when Google omits one. `gapi` has no `doc.go`; its package comment is at the
  top of `session.go`.
- **The notice.** `go/cmd/gdoc/notice.go:56` `notice(ctx, errOut)` writes the
  line itself at `:96`. `noticeLine` at `:165` builds it from parsed versions.
- **The domain.** `git grep -i 'altery\.com'` lists twenty lines in the eight
  places of the spec's table; `body_test.go` and `inline_test.go` hold seven of
  them, not the table's six. The walker shape to copy is
  `TestNoGoogleClientSecretInTheTree` at `go/boundary/secrets_test.go:18`.
  Three `.gitignore` files decide what git ignores: the root one,
  `.ralphex/.gitignore` (`progress/`, `worktrees/`) and `.revmux/.gitignore`
  (`tasks/`). revmux writes its findings, with the lines it read, under
  `.revmux/tasks/`.
- **The golden.** `checkGolden` at `go/internal/propose/blockbatch_test.go:22`
  has no update flag; `testdata/block-replace-batch.json:108` is edited by
  hand.
- **The review skill.** Its sections are "Step 2: Decide which threads still
  need an answer", "Before acting on an old marked comment again", "Two
  messages at most", "When to stop and ask", "If the request is for all the
  comments" and "Never". `live.md` and `propose.md` point at "SKILL.md's Step
  3", "Step 7" and "Step 8", and `TestStep7SaysABlockIsTheOtherKindOfProposal`
  pins "block" and "new or rewritten section" in `SKILL.md`. The call parser
  `TestProseAboutGdocIsNotReadAsACall` exercises lives in
  `go/cmd/gdoc/skills_test.go`. The body budget is `specBodyCharsMax = 20000`
  at `go/cmd/gdoc/skillspec_test.go:43`.
- **Test files over 800 lines**: `write_test.go`, `read_test.go`,
  `body_test.go`, `go/cmd/gdoc/skills_test.go` (804),
  `go/internal/gapi/session_test.go` (1158).
- **The documents.** PRINCIPLES.md `:75` ("The capability probe runs every
  time") and its 2026-09-18 amendment at `:135`. SPEC.md names the probe at
  `:147`, `:159-160`, `:301`, `:376` and `:750`. `go/internal/probe/doc.go:4-5`
  says `writeMode` is absent from the public discovery document.
  `go/internal/live/proposeblock_test.go:139-141` says the probe runs first
  because the command runs it before it writes.
- **The register.** "Never send `writeMode: SUGGEST` and trust the 200" is not
  a register row: it is the closing paragraph of the 2026-08-29 entry "gdoc
  never replaces the body of a document that already exists"
  (`DECISIONS.md:328`). The spec's "The Developer Preview loss must be loud" is
  the row "The preview may vanish, and the risk is accepted. Loudly"
  (`:1148`). "Never trust a success. Verify with a second, independent probe"
  is the row at `:24`.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own
  commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and
  failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test`
  runs `-race`; keep it there.
- **CRITICAL: no server code in this run.** No `internal/mcp`, no `gdoc mcp`
  entry, no tool schema, no extension file. If a task seems to need one, it
  stops and says so.
- **CRITICAL: no skill changes what it sends in this run.** No file under
  `skills/` changes except in Task 6, and Task 6 moves rules without changing
  one. No `needs` line moves. The tag sitting does the rest.
- **CRITICAL: the guard does not move.** No task touches
  `go/internal/guard/`. Task 3 removes a caller of `AllowCreateIn`; it adds
  nothing.
- **CRITICAL: no `make build` inside the run.** `~/.local/bin/gdoc` links to
  `bin/gdoc`, and a build without `GDOC_OAUTH_CLIENT_SECRET` puts a binary on
  the path that cannot sign anyone in. Task 12 builds into a temp directory.
- **CRITICAL: facts only in Go.** `outcome: "unknown"` is a fact about the
  answer. No field says whether a proposal should be sent again.
- **CRITICAL: a test states its value as a literal**, never reading the
  constant it checks. The domain test is the one exception the spec makes: it
  holds the domain as a hash, so the test does not name it, not even in parts.
- **CRITICAL: new tests go in new files** where the existing test file is over
  800 lines (the list in Context).
- **CRITICAL: no doc.go rule without its test named.**
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces. Plain English.

## Testing Strategy

- **Unit, `boundary`**: the domain scan finds a planted host; it passes on the
  tree; it skips what the three `.gitignore` files name and nothing else.
- **Unit, `cmd/gdoc` and `propose`**: the probe is never run; `--folder` is
  accepted, warned about and opens no create; each read-back failure stops the
  run; nothing after the stop is sent; a lost answer is `outcome: "unknown"`
  and stops the run; no stop error claims a direct edit.
- **Unit, `gapi`**: a 5xx or a drop after the write is marked unknown; a guard
  refusal, a dial failure, a TLS failure and a 4xx are not.
- **Unit, skills**: `review.md` holds no call, no fence and no flag; `SKILL.md`
  names it and keeps its step numbers.
- **Unit, `auth` and `loopback`**: start then wait is the CLI flow; the page
  says "Signed in" only after the save, and says the save failed when it did;
  the browser always gets an answer.
- **Unit, `gapi` refresh**: a newer login in the file is adopted and never
  overwritten; a refresh never writes an empty refresh token.
- **Unit, contexts**: a cancelled context cancels a read; `propose`,
  `annotate` and `reply` never cut a write from its read-back.
- **Live, opt-in**: no new live test. The live propose tests that pass a folder
  keep passing with it ignored.
- Coverage standard: every changed function has a test; no package below 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && test -z "$(gofmt -l .)" && go vet ./...`
- `git grep -i -c 'altery\.com'` prints nothing.
- `cd go && go build -o "$TMPDIR/gdoc-m14a" ./cmd/gdoc && "$TMPDIR/gdoc-m14a" help propose`
  prints the usage line
  `propose <url> --from <file> [--md <file>] [--folder <folder id>]` and says
  `--folder` is ignored and will be removed.
- `git diff main...HEAD --stat -- skills/` lists only `skills/gdoc-review/`.
- `grep -c 'needs: v2.7.0' skills/gdoc-review/SKILL.md skills/gdoc-align/SKILL.md`
  prints 1 and 1: no `needs` line moved.
- `wc -c skills/gdoc-review/SKILL.md` prints a number well under 20000.
- `grep -n 'context.Background' go/cmd/gdoc/read.go go/cmd/gdoc/annotate.go`
  prints nothing, and in `go/cmd/gdoc/write.go` only `cmdProbe` and
  `cmdWithdraw` keep it.
- `wc -l CLAUDE.md` under 300.

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

**Task 1** takes the firm's domain out of the tree and adds the test that keeps
it out, first, because decision 15 orders it first and every later fixture must
use the RFC 2606 domains.

**Task 2** writes run 1's DECISIONS.md entry, because a SPEC.md change is a
DECISIONS entry first and Task 3 changes SPEC.md.

**Tasks 3, 4 and 5** are the probe leaving `propose`, one behaviour each: the
probe and the folder in 3, the stop on an unconfirmed read-back in 4, the lost
answer in 5. Each is green on its own. After Task 5, main is a binary a
colleague can take without `gdoc mcp`, which is what decision 15 means by
"tagged alone"; the skills follow in the tag sitting.

**Task 6** is the `review.md` split, as one commit, with no rule changed.

**Tasks 7 and 8** are the sign-in pieces the server's `login` tool stands on:
the two-step login and the token race.

**Task 9** threads the context through the six commands chat will reach, and
keeps every write whole: the clock is read between items, never between a
send and its read-back.

**Task 10** splits `notice`. **Task 11** is the documents. **Tasks 12 and 13**
close.

## Technical Details

**The domain scan.** `go/boundary/domain_test.go`. A helper
`scanForHost(root string, want [32]byte) ([]string, error)` walks `root` with
`filepath.WalkDir`, skips `.git` and whatever the `.gitignore` files under
`root` name, and returns `path:line` for every line holding a host whose
SHA-256 is `want`. It reads the `.gitignore` files itself, which is reading a
file and not running git: a pattern ending in `/` names a directory, a pattern
with a `*` is matched against the base name with `filepath.Match`, a pattern
holding a `/` elsewhere is a path relative to that `.gitignore`'s directory,
and anything else names a file or directory at any depth. A pattern the helper
cannot read (a `!`, a `**`, a `[` it does not handle) fails the test by name
rather than being skipped, so a new `.gitignore` line cannot quietly widen
what the scan misses. Hosts are found by lowercasing the line and taking
tokens of `[a-z0-9-]+(\.[a-z0-9-]+)+`; every dot-boundary suffix with at least
one dot is hashed, so `mail.<domain>` is caught too. The firm's hash is one
constant in the test with a comment saying what kind of value it is and that
it must never be written out. The canary,
`TestTheDomainScanFindsAPlantedHost`, plants `person@example.net` in a temp
tree and passes `sha256("example.net")`, so no test names the firm's domain,
whole or in parts.

**propose without the probe.** `cmdPropose` stops requiring `--folder`. When it
is present, it is still parsed as a folder id, so a malformed value is refused
as before, and the run carries one warning:
`--folder is ignored: propose no longer creates a working copy, and the flag will be removed in a later release`.
No `AllowCreateIn`. `runPropose(r, proposals, note)` drops the probe and
`proposeData.Probe`. The `probeData` type stays for `cmdProbe`.

**The stop.** After each `propose.Apply`, the loop reads
`res.Checks.SuggestionsInline && res.Checks.PreviewWithoutSuggestions`. When
either is false, the proposal is reported `sent: true` with its checks, the
note records what was sent, every later proposal stays `sent: false`, and the
envelope is `ok: false` with:
`gdoc could not confirm that proposal <n> ("<quote>") landed as a suggestion, so nothing after it was sent. Look at it in the browser before proposing again: <document url>`.
The sentence never says "direct edit". `DocxAnchored` false alone does not
stop the run, as today.

**The lost answer.** `gapi` learns whether a request was written from
`net/http/httptrace`: `attempt` attaches a `ClientTrace` whose `WroteRequest`
hook records that the request went out (with a nil `Err`). A failure is marked
unknown only when the request was written and then came a transport error or
a status of 500 or above. A guard refusal, a dial failure and a TLS handshake
failure never reach `WroteRequest`, so they stay unmarked, and a 4xx is
Docs refusing the batch whole. The mark is a new wrapper beside `sent`, with an
`Unknown() bool` method, asked by behaviour in `propose` the way `sentAnyway`
asks for `Sent()`, so `propose` imports nothing new. The fake transports the
`gapi` and `cmd/gdoc` tests use fire the hook themselves, through
`httptrace.ContextClientTrace(req.Context())`, to stand in for a drop after
the write. `send` returns a distinct error for it; `Apply` returns a `Result`
with `Outcome: "unknown"` and the error; `runPropose` reports that proposal as
`sent: false, outcome: "unknown"`, stops, and says:
`proposal <n> ("<quote>") was written and its answer was lost, so it may or may not be in the document. Read the suggestions before proposing it again`.
Nothing retries. `outcome` is omitted on every other entry. The package
comment's section on what is not marked is rewritten to say which of the three
cases is now marked and how, and its `TODO(test)` closes with the test's name.

**The two-step login.**

```go
type Pending struct{ URL string /* unexported: srv, state, verifier, redirect, client */ }
func StartLogin(c *http.Client) (*Pending, error) // listens, builds the URL, returns at once
func (p *Pending) Wait(ctx context.Context) error // waits loginTimeout, exchanges, saves, finishes the browser
func (p *Pending) Close()                         // idempotent
func Login(c *http.Client, w io.Writer) error     // StartLogin, print p.URL to w, Wait, Close
```

`loopback`'s callback stops writing "Signed in" itself. It hands the code to
the waiter and holds the browser's request open. `Server.Finish(err error)`
writes the page, "Signed in. You can close this tab." on nil and "Sign-in
failed: <reason>" otherwise, flushes it, and returns only once the handler has
returned, so a `Close` straight after it cannot drop the page; that is what
`flushThenAnswer` protects today. `Wait` calls `Finish` on every path after a
code arrives, the exchange failing, the save failing and the context ending
included, through a `defer`. A held request that gets no `Finish` within
`finishWait` (30 seconds, a named constant) answers "gdoc could not confirm the
sign-in; look at the terminal or the chat that started it". A second callback
carrying the right state while one is held is answered at once with "a sign-in
is already being finished here" and changes nothing.

**The token race.** `Session` remembers the refresh token it loaded. `refresh`
first runs `auth.Load`. When the file's refresh token differs from the one the
session loaded, a newer login happened: the session takes the file's token,
and refreshes it only if it too has expired. Otherwise it refreshes its own.
Just before `auth.Save`, it loads the file once more, and when the file's
refresh token has changed again since, it takes that token and saves nothing.
`refresh` refuses to save a token whose refresh token is empty. Nothing locks:
two refreshes of one login both save a working token.

**The deadline between items.** `propose` and `annotate` check `ctx.Err()`
before each item, and hand `Apply` `context.WithoutCancel(ctx)`, so a write
that started finishes with all its read-backs. An item not reached is
`sent: false`, and the envelope says:
`the time for this call ran out after <k> of <n>; nothing after that was sent. Call again with the rest`.
The first read of the document runs on `ctx` and is cancelled with it.
`reply` checks `ctx.Err()` before it posts, then posts and reads back on
`context.WithoutCancel(ctx)`.

**The notice.** `notice(ctx) (*updateFacts, string, []string)`: facts, the line,
the warnings. `cmdHelp` writes the line to `errOut` with the blank line after
it, exactly as today.

## Implementation Steps

### Task 1: the firm's domain leaves the repository

Serves decision 17 (the domain half) and decision 15 (first). Scenarios: none
directly; every later fixture depends on it.

**Files:**
- Create: `go/boundary/domain_test.go`
- Modify: `go/internal/annotate/annotate_test.go`, `go/internal/propose/propose_test.go`,
  `go/internal/propose/blockbatch_test.go`, `go/internal/propose/testdata/block-replace-batch.json`,
  `go/internal/body/body_test.go`, `go/internal/body/inline_test.go`,
  `go/cmd/gdoc/annotate_test.go`, `go/cmd/gdoc/write_test.go`
- Modify: `docs/v2/DECISIONS.md` (the 2026-09-16 entry, one line),
  `docs/plans/completed/2026-09-16-gdoc-v2-m9-release.md`,
  `docs/plans/completed/2026-09-18-gdoc-v2-m11-annotate.md`
- Modify: `go/boundary/doc.go`

- [x] Test first, `TestTheDomainScanFindsAPlantedHost`: the canary of
      Technical Details. Watch it fail: no scanner yet.
- [x] Test, `TestTheDomainScanSkipsWhatGitIgnores`: a temp tree with three
      `.gitignore` files shaped like the real ones, `example.net` planted only
      under ignored paths (`bin/`, `.ralphex/progress/`, `.revmux/tasks/`, a
      `*.pyc`), passes; the same host in a file no pattern names fails.
- [x] Test, `TestTheDomainScanRefusesAPatternItCannotRead`: a `.gitignore`
      holding `!keep` or `a/**/b` fails naming the line.
- [x] Test, `TestNoCompanyDomainInTheTree`: the scan over the real tree with
      the firm's hash. Watch it fail on the twenty lines.
- [x] Replace each line as the spec's table says: `x@example.com` and
      `person@example.com` in the tests, the golden at
      `block-replace-batch.json:108` edited by hand (no update flag is added),
      "a sign-in from the firm's Google Workspace" in DECISIONS.md, "a firm
      Workspace account" in the M9 plan, `x@example.com` in the M11 plan. Each
      test checks what it checked.
- [x] `go/boundary/doc.go`: one paragraph on the domain rule, naming the four
      tests.
- [x] `cd go && go test -race ./...` passes; `make vet` passes.
- [x] `git commit -m "test(boundary): no company domain in the tree, and the examples use example.com"`

### Task 2: run 1's DECISIONS.md entry

Serves decisions 11, 17 and 19. Written before the code it records, because
Task 3 changes SPEC.md.

**Files:**
- Modify: `docs/v2/DECISIONS.md`

- [x] An entry dated 2026-10-02, "propose without the probe, and no company
      domain in the repository", in the file's own shape: what was decided,
      the principle served, what was rejected (keeping the probe until the
      rollout ends, about 2026-10-15; a lock after a stop), and Nail's
      acceptance of one unconfirmed proposal per call.
- [x] In the entry: the stop rule; a lost answer as `sent: false` with
      `outcome: "unknown"`, and why `sent` stays; run 1 released alone as
      v2.8.0 and `gdoc mcp` as v2.9.0, Nail's call of 2026-10-03; and the
      one-line edit Task 1 made inside the 2026-09-16 entry, with the reason,
      since an entry is otherwise never edited.
- [x] Register rows: the row "gdoc never replaces the body of a document that
      already exists" (2026-08-29) keeps `holds`, and the entry says its
      closing paragraph, "Never send `writeMode: SUGGEST` and trust the 200",
      is superseded 2026-10-02 in its method only. The row "The preview may
      vanish, and the risk is accepted. Loudly" becomes `superseded 2026-10-02
      (suggestions are generally available; a silent edit is still loud,
      through the read-back and the stop)`. The row "Never trust a success.
      Verify with a second, independent probe" is unchanged. A new row for
      this entry, `holds`.
- [x] A sentence saying the server's rows follow in a second entry after the
      spike.
- [x] `cd go && go test -race ./boundary/` passes (the docs tests read this
      file).
- [x] `git commit -m "docs(decisions): propose without the probe, and no company domain in the repository"`

### Task 3: propose runs no probe, and the folder is ignored

Serves decisions 11 and 19. Scenarios 10, 15.

**Files:**
- Modify: `go/cmd/gdoc/write.go`, `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/doc.go`
- Create: `go/cmd/gdoc/propose_noprobe_test.go`
- Modify: `go/cmd/gdoc/write_test.go`, `go/cmd/gdoc/paired_test.go`,
  `go/cmd/gdoc/propose_block_test.go`, `go/cmd/gdoc/propose_markers_test.go`,
  `go/cmd/gdoc/help_test.go` (the usage line)
- Modify: `go/internal/propose/doc.go`, `go/internal/probe/doc.go`
- Modify: `docs/v2/SPEC.md`, `PRINCIPLES.md`

- [x] Test first, `TestProposeRunsNoProbe`: a propose against the fakes sends
      no create, no trash and no request to any document but the target.
      Watch it fail on today's probe.
- [x] Test, `TestTheFolderFlagIsAcceptedAndIgnored`: with `--folder` the run
      carries the one warning, opens no create door on the policy, and sends
      the same requests as without it; a malformed folder id is still refused.
- [x] Implement as Technical Details. `proposeAnswers` loses its probe
      answers; every test that used it keeps checking what it checked about
      the proposal. Tests of the probe's own behaviour move to `gdoc probe`;
      tests of the coupling go.
- [x] `--folder` in the table: `needOptional`, after `--md`, its sentence
      "ignored: propose no longer creates a working copy; removed in a later
      release". The example drops it. The usage-line test spells the new line.
- [x] `propose/doc.go` and `cmd/gdoc/doc.go`: the probe paragraph goes, and the
      folder rule names its test. `probe/doc.go` loses the sentence that
      `writeMode` is absent from the discovery document, says it is listed
      there now, labelled Developer Preview while the rollout lands, and that
      `gdoc probe` is a manual command only.
- [x] SPEC.md: the probe paragraphs at `:147`, `:159-160`, `:301`, `:376` and
      `:750` say the probe no longer runs before a proposal and the read-back
      stop does instead. PRINCIPLES.md: an amendment dated 2026-10-02 under the
      2026-09-18 one, saying the same in two sentences.
- [x] `cd go && go test -race ./...` passes; `make vet` passes.
- [x] `git commit -m "feat(propose): no capability probe, and --folder accepted and ignored"`

### Task 4: propose stops at the first proposal a read-back cannot confirm

Serves decision 11. Scenario 11.

**Files:**
- Modify: `go/cmd/gdoc/write.go`, `go/cmd/gdoc/doc.go`, `go/internal/propose/doc.go`
- Create: `go/cmd/gdoc/propose_stop_test.go`

- [ ] Test first, `TestAFalseInlineCheckStopsTheRun`,
      `TestAFalsePreviewCheckStopsTheRun` and
      `TestAReadBackThatFailedStopsTheRun`: three proposals, the first fails
      the named check; the first is `sent: true` with its checks, the other
      two `sent: false`, nothing after the first reaches the wire, `ok: false`,
      and the error is the literal of Technical Details. Watch them fail.
- [ ] Test, `TestTheStopNeverClaimsADirectEdit`: the error holds no "direct
      edit" for any of the three.
- [ ] Test, `TestAFalseDocxCheckAloneDoesNotStop`: `docx_anchored: false`
      with the other two true sends all three.
- [ ] Test, `TestAStoppedRunRecordsWhatWasSent`: with `--md`, the note records
      the first proposal and nothing else.
- [ ] Implement the stop in the loop.
- [ ] The stop rule in both doc.go files, naming the tests.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(propose): stop at the first proposal either read-back cannot confirm"`

### Task 5: a lost batch answer is outcome unknown, and stops the run

Serves decision 11. Scenario 11.

**Files:**
- Modify: `go/internal/gapi/session.go` (the code and the package comment)
- Create: `go/internal/gapi/unknown_test.go`
- Modify: `go/internal/propose/propose.go`, `go/internal/propose/blockapply.go`,
  `go/internal/propose/doc.go`
- Create: `go/internal/propose/unknown_test.go`
- Modify: `go/cmd/gdoc/write.go`, `go/cmd/gdoc/doc.go`, the fake transport the
  `cmd/gdoc` tests use
- Create: `go/cmd/gdoc/propose_unknown_test.go`

- [ ] Test first, in `gapi`, `TestAFiveHundredAfterTheWriteIsMarkedUnknown`
      and `TestADropAfterTheWriteIsMarkedUnknown`. Watch them fail.
- [ ] Test, `TestNothingBeforeTheWriteIsMarkedUnknown`: a guard refusal, a
      dial failure, a TLS handshake failure and a 4xx are not `Unknown()`.
- [ ] Test, in `propose`, `TestSendNamesALostAnswer`, on a words proposal and
      on a block.
- [ ] Test, in `cmd/gdoc`, `TestALostAnswerIsOutcomeUnknownAndStops`: the
      entry is `sent: false, outcome: "unknown"`, every later entry has no
      `outcome`, the run stops, the error is the literal, and the fake sees
      exactly one write.
- [ ] Implement as Technical Details, the trace hook in the fakes included.
- [ ] The package comment of `gapi`: the section on what is not marked says
      which case is now marked and how; its `TODO(test)` closes with the
      tests' names. `propose/doc.go` and `cmd/gdoc/doc.go`: the lost answer,
      naming the tests.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(propose): a batch whose answer was lost is outcome unknown, and stops the run"`

### Task 6: the review rules move into review.md

Serves decision 9 (the split; the embedded copy is run 2's). One commit,
because the skill is live the moment it is saved.

**Files:**
- Create: `skills/gdoc-review/review.md`
- Modify: `skills/gdoc-review/SKILL.md`
- Create: `go/cmd/gdoc/skills_review_test.go`

- [ ] Test first, `TestTheReviewCoreCarriesNoCall`: `review.md` gives zero
      calls through the parser `TestProseAboutGdocIsNotReadAsACall`
      exercises, holds no fenced block, and no `--flag` token. Plain English
      that names a command word, like "reply" or "read", passes. Watch it
      fail: the file does not exist.
- [ ] Test, `TestTheReviewSkillNamesItsCore`: `SKILL.md` names `review.md` and
      tells the session to read it before the first thread.
- [ ] Test, `TestEveryReviewRuleLandedOnce`: the rule text of "Before acting
      on an old marked comment again", "Two messages at most", "When to stop
      and ask", "If the request is for all the comments" and "Never", and the
      rules of "Step 2: Decide which threads still need an answer", is in
      `review.md` and no longer in `SKILL.md`.
- [ ] Test, `TestTheReviewStepsKeepTheirNumbers`: `SKILL.md` still has the
      headings Step 1 to Step 8 in order, because `live.md` and `propose.md`
      point at Step 3, Step 7 and Step 8. A step whose rules moved keeps its
      heading and one line pointing at its section of `review.md`.
- [ ] Move the rules, reworded only where a sentence named a shell call. No
      rule is added, dropped or changed in meaning: the two guards, the
      annotate flow and the chat header are run 2's, written with the
      measured values. `TestStep7SaysABlockIsTheOtherKindOfProposal` stays
      green.
- [ ] `SKILL.md` keeps setup, the calls and live mode, and is well under the
      20,000-character budget.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "refactor(skills): the review rules in review.md, the calls in SKILL.md"`

### Task 7: the login in two steps, and "Signed in" only after the save

Serves decision 4 (the parts that are not the tool). Scenario 2.

**Files:**
- Modify: `go/internal/auth/login.go`, `go/internal/auth/doc.go`,
  `go/internal/auth/loopback/loopback.go`
- Create: `go/internal/auth/twostep_test.go`, `go/internal/auth/loopback/finish_test.go`
- Modify: `go/cmd/gdoc/main.go` (only if `login` needs the new names)

- [ ] Test first, `TestStartLoginReturnsTheLinkAtOnce`: `StartLogin` returns a
      URL naming the listener's address and the state, without waiting.
- [ ] Test, `TestWaitExchangesAndSaves`: a fake callback with a code, a fake
      token endpoint; `Wait` saves the token.
- [ ] Test, `TestThePageSaysSignedInOnlyAfterTheSave`: the browser's response
      is not written until `Finish`; `Finish(nil)` writes "Signed in"; a save
      that failed writes "Sign-in failed" and the reason.
- [ ] Test, `TestTheBrowserIsAnsweredOnEveryPath`: an exchange that fails, a
      save that fails and a context that ends each give the browser a page,
      and a `Close` straight after `Finish` never drops it.
- [ ] Test, `TestAFinishThatNeverComesStillAnswersTheBrowser`: past
      `finishWait`, the page says gdoc could not confirm the sign-in.
- [ ] Test, `TestASecondCallbackWhileOneIsHeldChangesNothing`.
- [ ] Test, `TestCloseIsSafeTwice`.
- [ ] Implement as Technical Details; `Login` is the two in a row and prints
      exactly what it printed. The CLI's `auth login` tests stay as they are
      and green.
- [ ] `auth/doc.go`: the two steps and the saved-then-signed-in page, naming
      the tests.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(auth): start and wait as two calls, and the browser hears signed in only after the save"`

### Task 8: a refresh never writes an old login over a newer one

Serves the spec's "The token" paragraph. Scenario 2.

**Files:**
- Modify: `go/internal/gapi/session.go` (the code and the package comment)
- Create: `go/internal/gapi/refresh_test.go`

- [ ] Test first, `TestARefreshAfterANewerLoginSavesNothing`: the session
      loads token A; the file is replaced by a valid token B with another
      refresh token; the session's next request carries B's access token, B
      stays in the file byte for byte, and no refresh request is sent.
- [ ] Test, `TestARefreshAfterANewerExpiredLoginRefreshesThatOne`: B is
      expired; the refresh is of B's refresh token, never A's.
- [ ] Test, `TestALoginBetweenTheRefreshAndTheSaveWins`: the file changes
      between the refresh answer and the save; nothing is saved.
- [ ] Test, `TestARefreshNeverWritesAnEmptyRefreshToken`.
- [ ] Test, `TestTwoRefreshesOfOneLoginBothSave`: the ordinary path is
      unchanged.
- [ ] Implement as Technical Details.
- [ ] The rule in the package comment at the top of `session.go`, naming the
      tests.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "fix(gapi): a refresh looks at the token file first and never saves over a newer login"`

### Task 9: the six chat commands take the context they are handed

Serves the spec's "Deadlines" paragraph, the part inside the commands.
Scenarios 7, 10.

**Files:**
- Modify: `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/read.go`, `go/cmd/gdoc/write.go`,
  `go/cmd/gdoc/annotate.go`, `go/cmd/gdoc/doc.go`
- Create: `go/cmd/gdoc/context_test.go`

- [ ] Test first, `TestACancelledContextCancelsTheRead`: `read`,
      `suggestions` and `comments` handed a done context send no request and
      fail naming the cancellation.
- [ ] Test, `TestProposeStopsBetweenProposalsWhenTimeRunsOut`: the context
      ends during proposal 1's read-backs; proposal 1 finishes all three
      read-backs, proposals 2 and 3 are `sent: false`, the error is the
      literal of Technical Details.
- [ ] Test, `TestAReadBackIsNeverCutByTheDeadline`: a context that ends
      between the write and the first read-back still yields all three
      read-backs.
- [ ] Test, `TestAnnotateStopsBetweenItemsWhenTimeRunsOut`: the same for
      `annotate`.
- [ ] Test, `TestReplyNeverCutsItsReadBack`: a done context sends no reply; a
      context that ends after the post still yields the read-back.
- [ ] Implement as Technical Details: the table entries pass `ctx`;
      `cmdRead`, `cmdSuggestions`, `cmdReply`, `cmdPropose` and
      `cmdAnnotate` take it. `cmdProbe` and `cmdWithdraw` keep
      `context.Background()`: neither is a chat tool.
- [ ] `cmd/gdoc/doc.go`: the deadline rule, naming the tests.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(cmd): the chat commands take their context, and no write is cut from its read-back"`

### Task 10: notice returns its line

Serves decision 13 (the split). Scenario 13.

**Files:**
- Modify: `go/cmd/gdoc/notice.go`, `go/cmd/gdoc/help.go`
- Modify: `go/cmd/gdoc/notice_test.go`

- [ ] Test first, `TestNoticeReturnsTheLine`: with a stale stamp and a newer
      release, `notice` returns the line
      ``gdoc v2.9.0 is published and this is v2.8.0. `gdoc update` installs it.``
      as one literal in the test.
- [ ] Test, `TestHelpPrintsTheNoticeLineAndABlankLine`: `help` writes that
      line and one blank line to stderr, and nothing else of the notice.
- [ ] `TestAStaleStampMakesHelpAskOnce`, `TestAFreshStampMakesNoRequest`,
      `TestACheckoutBuildNeverChecks`, `TestTheCheckIsBoundedByTwoSeconds` and
      `TestReadNeverReachesTheCheck` stay green.
- [ ] Implement as Technical Details.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "refactor(cmd): notice returns its line and help writes it"`

### Task 11: the documents

**Files:**
- Modify: `go/cmd/gdoc/doc.go`, `CLAUDE.md`, `docs/guide/`, `README.md`,
  `go/internal/live/doc.go`, `go/internal/live/proposeblock_test.go`, each
  only where a sentence is stale
- Modify: `docs/v2/PLAN.md`
- Modify: `docs/plans/2026-10-02-gdoc-v2-m14-chat.md` (the status line and
  decision 19's number)

- [ ] `grep -rn -i 'probe' CLAUDE.md README.md docs/guide/ go/cmd/gdoc/doc.go go/internal/live/`:
      every sentence still saying a proposal runs the probe is fixed,
      `proposeblock_test.go:139-141` included.
- [ ] `docs/v2/PLAN.md`: M14 split into run 1 (v2.8.0), the tag sitting, the
      spike, and run 2 (v2.9.0).
- [ ] The spec: a status line under its title naming this plan and the
      version split, and decision 19's number changed to v2.9.0 with the date
      of Nail's call.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "docs: M14 run 1, propose without the probe"`

### Task 12: Verify acceptance criteria

- [ ] `make test`, `make vet` and `make dist` pass. No `make build`: see
      Development Approach.
- [ ] Every Validation Command above gives the answer it states; record the
      counts here as ➕ notes.
- [ ] Scenarios 10, 11 and 15 (the binary's half) are served by Tasks 3 to 5,
      read against the spec; scenario 2's sign-in half by Tasks 7 and 8.
- [ ] `git diff main...HEAD -- go/internal/guard/` is empty.
- [ ] `ls go/internal/mcp 2>/dev/null` prints nothing: no server code landed.
- [ ] Nothing to commit unless a ➕ note was added.

### Task 13: Update documentation

- [ ] Move this plan to `docs/plans/completed/`. The spec stays in
      `docs/plans/` until the last run.
- [ ] `cd go && go test -race ./boundary/` passes.
- [ ] `git commit -m "docs(v2): M14 run 1, completed"`

## Post-Completion

**The tag sitting, by hand, right after run 1 merges.** In one sitting, with a
Claude Code session:

- `skills/gdoc-review/propose.md` (the call at `:27`, the paragraph at
  `:93-95`) and `skills/gdoc-align/SKILL.md` (`:263-286`) pass no folder, say
  what a stopped run means in the stop error's own words, name
  `outcome: "unknown"`, and say that after it, or after a timeout,
  `suggestions` is read before anything is proposed again.
- `needs: v2.8.0` on both, and `skillWants` at
  `go/cmd/gdoc/skills_test.go:543-553` moved with them.
- A test in a new file, `TestNoSkillPassesAFolderToPropose`, matching
  `--folder ` and `--folder=` so `--folder-id` cannot trip it.
- `make tag VERSION=v2.8.0`, which moves `plugin.json` and so satisfies
  `TestNoSkillNeedsAReleaseNobodyCut`, then
  `gh workflow run nightly.yml --ref main`.
- `make build` with `GDOC_OAUTH_CLIENT_SECRET` set, so the binary on the path
  is the one the skills now expect.

**Live, by Nail, before the tag**: one `propose` from `gdoc-review` on a
document in the test folder, and a look at the suggestion in the browser.

**The spike, by Nail with a Claude Code session, beside or after this run**:
a stub MCP server outside the tree (scratchpad, not committed) that logs every
line Claude Desktop sends, answers `initialize`, lists one read tool and one
write tool, and can add a `confirm_<hex>` tool through `tools/list_changed`.
Install it as a thin `.mcpb` whose command is an absolute path, and answer the
twelve measurements of the spec in MEASURED.md. Any answer that contradicts the
spec goes back to Nail before run 2's plan is written.

**Run 2's plan** is written after the spike, from the measured values: the
quiet gap, whether the `guide` code stays, card or draft, `.mcpb` or a config
entry, voice now or later. It also takes the server's DECISIONS.md entry, the
embedded `review.md`, the two guards, the annotate flow and the chat header,
and releases as v2.9.0.
