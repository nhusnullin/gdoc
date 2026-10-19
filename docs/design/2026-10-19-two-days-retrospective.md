# Two days of the presence app: who worked, how, and what it cost

2026-10-19. Written at the end of the shift that ran from the night of
2026-10-16 to the night of 2026-10-18: M20 to M27, v2.13.0 to v2.19.0. Two
questions from Nail: fix the team so the same people work the next time, and
find how to get the same result for fewer tokens, because the two days ate a
week's quota. Part 1 is the roster and the process as they were. Part 2 is
the cost and the cuts. Written in chunks so a quota that runs out mid-way
loses little.

## Part 1. The team, by seat, and how it worked

### The seats

Every seat was a Claude subagent or a ralphex worker with a one-page brief.
Names are the brief's, so a future run can call the same seat by name.

| Seat | What it does | When it ran |
|---|---|---|
| **Project manager** (this session) | holds the plan, writes decisions, specs and plans, runs ralphex and revmux, merges, tags, verifies the zip, keeps the board and the report, talks to Nail | the whole shift |
| **Go engineer** | the channel and the flag in the binary: what the record carries, where it is written, what never goes in | M23 design night |
| **macOS engineer** | the Swift app: toolchain, signing, bundle, SMAppService, what the app may read and run | M23 design night |
| **Release engineer** | delivery: the zip, the two CI jobs, `gdoc update` installing the app, rollback | M23 design night |
| **Trust guardian with the skeptic** | the boundary: one-way channel, no path into the binary, the minimum that ships, what is refused by name | M23 design night |
| **Codex second opinion** (gpt-5.5 through `codex-second-opinion`) | reads the design document against the tree and names the holes | after the M23 design; also the adversarial lens in every revmux round |
| **Plan reviewer** (`planning:plan-review`) | reads a plan before it runs: missing tests, wrong order, over-engineering | before M23, M24a, M24b |
| **Design seats** (in `docs/design/2026-10-17-presence-design-principles.md`, "The team, by seat") | designer, iconographer, Mac icon expert, product owner, architect; each owns a question in the screens | the screens, the votes, the product decisions |
| **Reviewer who drew nothing** | a fresh subagent with the checklist, the mockup and the PNGs; rates each finding blocking / should-fix / note; never fixes | inside the plan (one task) and again over the finished branch before the PR, for M24a, M26, M27 |
| **ralphex task worker** (Opus) | one task per iteration: failing test, code, commit | every milestone |
| **revmux** (Codex adversarial, bugs+impl, docs+tests, arch+quality; then Claude critical/major passes) | the code review after the tasks, with fixes committed by the worker | every milestone |

### The process, in the order it ran

1. Nail's sentences, verbatim, into the decision entry (`docs/v2/DECISIONS.md`),
   then the spec paragraph, then the plan under `docs/plans/`. One commit on
   `main` before any code.
2. A worktree with its branch (`git worktree add -b <branch> ../gdoc-mNN main`),
   a board folder under `~/gdoc-mNN-board/` with `launch-tasks.sh` and
   `launch-review.sh` (both `env -u CLAUDECODE ... nohup ralphex ... >> log`),
   the board page published as an artifact.
3. `ralphex --tasks-only`, a Monitor on the log, the board refreshed on each
   task boundary. Main merged into the branch before the task that needs it.
4. `ralphex --external-only` (revmux). Then the reviewer who drew nothing
   over the renders of the finished branch. Blocking findings go back as a
   second small plan (M24a) or into the next milestone.
5. `make vet`, `make test` with the live variables unset, `make app-lint`,
   `make app-test`, `make app`; push; PR; CI; `gh pr merge --merge`; pull;
   `make tag VERSION=vX.Y.0`; download the zip and check sum, bundle,
   codesign, plist, architectures; the report artifact; a push notification.
6. What Nail says about the release becomes the next decision entry, the
   same day.

### What broke, and the recipe that fixed it

- Claude CLI OAuth expired and killed both runs: `/login` inside an
  interactive `claude`, then the launch line again.
- A worker that printed its summary and did not exit: its own background
  wait loops (`pgrep -P <worker> -f never-exists-sentinel`, kill them), or,
  with the background-wait ceiling lifted, a background shell it never
  closed (kill the worker; ralphex exits 143; relaunch; the ticked boxes
  carry it on). Both in `.ralphex/board/README.md`.
- A worktree with no branch: the board counted 0/N; `git worktree add -b`.
- A reviewer pass inside the plan is not enough: the fresh reviewer after
  the run found 33 findings on M24a. Keep both.
- Patch versions are the nightly's and the nightly zip carries no app, so a
  one-line panel fix shipped as a minor (backlog item).

The roster above is the one to call again. The two seats that paid for
themselves most were the reviewer who drew nothing and the trust guardian;
the plan reviewer prevented at least one rerun per milestone.

## Part 2. Where the tokens went, and the cuts

No meter was read during the shift, so these are counts of what ran, from
the logs and the boards, ranked by what each kind of run costs. The ranking
is the honest part; the shares are estimates.

### What ran

| Run | Count over the two days | Why it is expensive |
|---|---|---|
| ralphex task iterations (Opus, a fresh worker per task) | M20 10, M21 8, M22 9, M23 12, M24b 9, M24a 26 + 6, M25 7, M26 11, M27 9: about 107 iterations | each worker reads CLAUDE.md (300 lines), the plan, the spec, the package comments and the files it touches, then writes tests and code; a UI task also renders every scene (`PRESENCE_SHOTS`) and reads PNGs |
| revmux rounds | M24a twice (3h43m and 3h30m), M26 1h27m, M27 50m, M25 16m, the night's four: 10 rounds; each is Codex ×3 plus up to five Claude passes, each pass reading the whole diff | the five Claude passes are five full reads of a 20k-line diff; passes 3 to 5 found little |
| reviewer-who-drew-nothing passes | inside the plan: M24a ×2, M26, M27; after the run: M24a, M26 (×5 reviewers), M27 (×5): about 20 reviewer agents | each reads 60 to 120 PNGs; an image is far dearer than its caption |
| this PM session | one context of about 15M tokens, compacted twice | 30-minute monitors re-armed about 60 times, a board refresh and an artifact publish after nearly every iteration, renders read by me after the reviewers had read them, the night report rebuilt and published five times |
| design night subagents | four seats, Codex, plan reviewer | cheap next to the rest |

The order of cost: revmux (five passes over big diffs, run twice on M24a),
then the task iterations, then the image-reading reviewers, then this
session. The three releases on 2026-10-18 each paid a whole review cycle for
a few hundred changed lines.

### The cuts, in the order they save

1. **One review cycle per day, not per sentence.** M25, the sign-out fix,
   M26 and M27 each ran their own revmux. Collect the morning's sentences
   into one milestone and run one review in the evening. Same result, a
   third of the review cost. The sign-out fix alone was right to ship fast;
   everything else could have waited six hours.
2. **Cap revmux for UI milestones.** One Codex round and two Claude passes
   for `presence/`; the full five passes only where `go/` changed. On M24a
   the fourth and fifth passes moved commas. The reviewer who drew nothing
   is the review that matters for a panel.
3. **One reviewer over fewer, smaller images.** Five reviewers each read the
   same 120 PNGs. One reviewer over the whole-panel dark renders at 1x (18
   images), with the light appearance only where dark had a finding, and
   the captions read as text from `index.html`. Keep the second,
   independent pass after the run; it is the one that found the 33.
4. **Render once per milestone.** `PRESENCE_SHOTS` only in the reviewer's
   task, never in a task's own validation. Every render is 120 PNGs the
   worker then reads.
5. **Fewer, fuller tasks.** "The words", "the acceptance rows" and "the
   close" were three iterations on every plan; one is enough. A measurement
   task that spawns experiments (M25 Task 5, M26 Task 8) should be the PM's
   ten minutes, not a worker's forty.
6. **A smaller model where the task is mechanical.** Docs tasks, fixture
   edits, board scripts and the close do not need Opus. Where ralphex
   allows a per-task model, say so in the plan; where it does not, run the
   mechanical tasks as one Sonnet subagent from the PM session.
7. **The PM session does less.** Refresh and publish the board every 30
   minutes or on a phase change, not per iteration; never read a render the
   reviewer already read; rebuild the report once at the end of the day;
   compact as soon as a milestone ships. The board and report publishes
   themselves cost nothing in tokens, but each round trip carries the
   context with it.
8. **Shorter briefs for the worker.** CLAUDE.md at 300 lines is read 107
   times; the invariant list is the expensive half. A worker for a `presence/`
   task needs the two app invariants and the app's doc comments, not the
   guard's. A per-area `CLAUDE.md` section the plan names ("read the presence
   half") would halve the fixed cost per iteration.

Rules 1, 2 and 3 are worth more than the other five together. The next
morning's sentences go into one plan, one run, one capped review, one
reviewer with eighteen images.

### What to keep as it is

The decision entry before the spec, the plan review before the run, the
worktree with its branch, the merge of main before a dependent task, the
fresh reviewer before the PR, the zip verified after the release, and the
report Nail reads in the morning. None of these is where the tokens went.
