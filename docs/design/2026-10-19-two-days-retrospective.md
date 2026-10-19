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
