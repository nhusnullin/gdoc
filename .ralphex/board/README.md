# The delivery platform

Two scripts behind two make targets, a page that shows where a run is, and a
meter that says what it cost. `make milestone` owns a run from the worktree to
the end of the review. `make release` takes the finished branch to a green pull
request and stops there. Both are what a Claude Code session used to do by
hand, and the hand recipes are still here, at the bottom, for the day a script
cannot.

Who runs all this, and why it is split the way it is, is `docs/team/README.md`,
with the lead's own page beside it in `docs/team/lead.md`.

## Files

| File | Holds |
|---|---|
| `milestone.sh` | one run: the worktree, the task phase, the review phase, the status file |
| `release.sh` | the branch to a green pull request, and, on Nail's go, the merge, the tag and the zip read back |
| `meter.py` | what a run cost, summed out of its transcripts. `docs/team/measure.md` is the method and the baseline |
| `board.html` | the page template. Header placeholders and an empty snapshot the script fills |
| `refresh_board.py` | rebuilds the snapshot from the logs, the plan and git. Prints one status line for the artifact label |
| `tests/` | the platform's own suite. Every case is a dry run or a stub, so `make platform-test` costs a second and no tokens |

## `make milestone`

```bash
nohup make milestone PLAN=docs/plans/<file>.md REVIEW=go > ~/milestone-launch.log 2>&1 &
```

The plan file names everything else. `2026-10-23-gdoc-v2-m29-the-archive.md`
gives the branch `gdoc-v2-m29-the-archive`, the worktree `../gdoc-m29` and the
board folder `~/gdoc-m29-board`, which is the naming every run since M20 has
used by hand.

`REVIEW` is `go` or `ui`, and the plan's own `Review:` line answers it when the
command line does not. `go` keeps the configured review: three external
iterations, patience two, revmux's full roster. `ui` is one iteration, patience
one and three lenses, because on M26 and M27 the passes after the third moved
commas. The plan's `Tasks:` line becomes the iteration cap.

`DRY=1` prints every command and runs none. `IDLE` overrides the worker's idle
timeout, twenty minutes by default.

The script strips the four `CLAUDE_*` variables, which make the Claude CLI
refuse to start inside another session, and the two `GDOC_LIVE_*` ones, which
were set in the shell that ran M18 to M27 and made a worker's plain `make test`
write documents into Drive.

### What the lead reads while it runs

`~/gdoc-<mNN>-board/status`, three lines, replaced in place:

```
phase: review (go)
commit: 1180aaf feat: the lead playbook and the team page
next: wait: the board and this file refresh every 30 minutes
```

The phase, the last commit on the branch, and the one next action. That is the
whole interface. The lead does not read the log, and the board refreshes every
thirty minutes or on a phase change, never per iteration.

Two recipes are in the script, because ralphex does not cover them. A ralphex
line `Failed to authenticate` stops the run and writes `needs /login`; the words
inside a worker's own sentence do not, and neither does the line a dead run left
in the log before this one started. A ralphex that exited with tasks left is
relaunched once; after that the run stops and names the log. A stalled worker is
`--idle-timeout`'s job.

Each phase has to say it finished: the tasks with `ALL_TASKS_DONE`, the review
with ralphex's own `completed in`. Both are read the way the sign-in line is,
where the message starts the line: a worker writing about `ALL_TASKS_DONE` is
not a finished task list, and the M28 log carries those words on ten lines that
are not the marker. The patterns and the lines they were measured against are
in `milestone.sh` beside `TASKS_DONE`. A
phase that ended without its own line writes `stopped in <phase>` and the log
to read, never `done`.

A run that was refused before it started says so in the same file, as
`phase: refused` with the refusal as the next line, because the launch above is
detached and nobody is watching stderr.

## `make release`

```bash
make release BRANCH=gdoc-v2-m29-the-archive
make release BRANCH=gdoc-v2-m29-the-archive GO=v2.20.0
```

Without `GO` the script pushes the branch, opens the pull request with the
plan's summary, waits for the checks and writes `ready: PR <n>, checks green`.
It merges nothing and tags nothing.

With `GO=vX.Y.0`, and only on a pull request whose checks are green, it merges,
pulls, runs `make tag`, waits for the release workflow and reads the arm64 zip
back: the checksum, the bundle, the signature, the plist version, the
architectures, and `gdoc help --json`. It writes `released vX.Y.0, verified` or
the step that failed.

The lead passes `GO` only after Nail's message. That is the one step in a
milestone that is his and not the team's.

Run it from the checkout that is on `main`, never from the milestone's worktree.
`make tag` commits the version bump on HEAD, tags HEAD and pushes HEAD, so from
a feature branch the bump would land there and `main` would never get it. The
script refuses `GO` anywhere but `main` for that reason.

## When the script cannot

The hand recipe, as a Claude Code session ran a milestone before M28. It is
here for a run the scripts cannot carry: a plan shaped unlike the others, a
board that needs an option `milestone.sh` does not pass, a platform the team is
still fixing. It costs a session's context on every turn, which is why it is no
longer the first thing on this page.

`SCRATCH` below is the session's scratchpad directory.

1. **Plan on main.** Write `docs/plans/<date>-<slug>.md`, commit and push it on
   `main`. ralphex creates the branch from `main`, and the reviewer must not see
   its own scripts as new code.

2. **Make the worktree with its branch**, so the board's `git log main..<branch>`
   and the push both find it: `git worktree add -b <branch> ../gdoc-mNN main`.
   A worktree made with `--detach` works too since 2026-10-17 (the board falls
   back to HEAD), but the push then needs `git checkout -b <branch>` first.
   Launch scripts append to their log (`>>`): a relaunch after a Claude CLI
   sign-in expiry keeps the earlier iterations, which the board pairs tasks with.

3. **Start the task phase, detached.** ralphex refuses to start inside a Claude
   Code session unless the session variables are stripped:

   ```bash
   env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT -u CLAUDE_CODE_SESSION_ID -u CLAUDE_CODE_CHILD_SESSION \
     nohup ralphex --tasks-only docs/plans/<plan>.md > "$SCRATCH/ralphex-<m>-tasks.log" 2>&1 &
   echo $!
   ```

4. **Create the page once.** A `warns.json` is a list of
   `{"m": "⚠", "b": "bold lead", "d": "one sentence"}`; put the milestone's
   decisions and out-of-scope notes there. A `tasks.json` of
   `{"1": {"what": "...", "pkg": "..."}}` replaces the plan-derived one-liners
   with hand-written ones, which Nail asked for after M2 ("self explainable").

   ```bash
   python3 .ralphex/board/refresh_board.py --init \
     --plan docs/plans/<plan>.md --branch <branch> --heading "Milestone N: <name>" \
     --html "$SCRATCH/mN-run-board.html" \
     --logs "$SCRATCH/ralphex-<m>-tasks.log" "$SCRATCH/ralphex-<m>-review.log" "$SCRATCH/ralphex-<m>-review2.log" \
     --warns warns.json --tasks tasks.json --pid <pid>
   ```

   A plan written on the branch itself, not on `main`, adds `--base <commit>`
   naming the plan's commit, so the docs commits before it are not paired
   with Task 1. A plan whose first tasks landed on `main` before the run adds
   `--offset <n>`, so Tasks 1 to n count as done by their checkboxes and Task
   n+1 pairs with the first commit. A task phase resumed into a second log
   lists both task logs first; the board reads a log with no task iteration
   in it as a review log.

   Publish `mN-run-board.html` with the Artifact tool, with a favicon and a
   one-sentence description. Every later publish uses the same file path so the
   URL stays.

5. **Arm a Monitor on the log** (persistent), with a filter that emits one line
   per event and covers failure as well as progress:

   ```
   ^--- task iteration|ALL_TASKS_DONE|task execution completed|TASK_FAILED|max iterations|^error:|session limit|usage limit|^completed in
   ```

   plus a loop that waits on the pid and prints `PROCESS EXITED` when it goes.

6. **Loop.** `/loop` in dynamic mode with a 1800 s fallback heartbeat. On each
   Monitor event: run `refresh_board.py --html ... --pid <pid>`, republish the
   page with the label the script printed, one short line in chat, reschedule.

7. **When the tasks finish** (`ALL_TASKS_DONE`, process exited), start the
   review the same way and re-arm a Monitor on its log:

   ```bash
   env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT -u CLAUDE_CODE_SESSION_ID -u CLAUDE_CODE_CHILD_SESSION \
     nohup ralphex --external-only > "$SCRATCH/ralphex-<m>-review.log" 2>&1 &
   ```

   Review filter:

   ```
   ^--- (custom|codex|claude) review|^--- claude evaluating|── complete ──|findings: |NO ISSUES FOUND|TASK_FAILED|^error:|session limit|^completed in
   ```

8. **When the review exits** (`completed in`): `make test`, `make vet`,
   `make build`, the milestone's live tests, commit any uncommitted review fixes
   (the external loop leaves each round's fixes in the working tree until it
   ends), push, open the PR, wait for CI, refresh the board once more, stop the
   loop. Never merge without Nail.

## What the runs taught

- `review_patience` counts rounds that change nothing. Opus fixes something
  every round, so the loop runs until revmux comes back clean or hits the cap
  (`max(3, max_iterations/5)` = 10). M3 hit the cap; M4 came back clean at
  round 8; M5 ran to the cap.
- A Claude session limit kills the run mid-round ("You've hit your session
  limit · resets 3am"). Recovery: `make test` on the working tree, commit the
  green fixes as `fix: address code review findings (revmux rounds so far)`,
  hourly `noop` wake-ups until the reset, then start `--external-only` again on
  the same branch. It resumes with a fresh round and re-finds anything the
  killed round left unevaluated.
- The Claude CLI's OAuth session expires mid-run too ("Failed to
  authenticate: OAuth session expired" in the log, the run dead a line
  later). `claude login` from a shell only sends the word "login" as a prompt;
  the fix is `/login` inside an interactive `claude`, then the launch line
  again. `claude -p "say ok"` is the probe. `milestone.sh` stops the run and
  writes `needs /login` when it reads that line, so the lead does not have to
  find it.
- A task worker can finish its task, commit, print its summary and still not
  exit: `claude -p` waits for the worker's own background shells, and a worker
  that idled on reviewers with `until [ -f /tmp/never-exists-... ]; do sleep;
  done` loops leaves them running. The log goes quiet after a "Validation:"
  paragraph and the next iteration never starts. Kill the loops
  (`pgrep -P <worker pid> -f never-exists-sentinel`), the worker exits and
  ralphex moves on. M24a Task 23 sat like that for 35 minutes.
- The Monitor's log filter matches prose too (`OAuth` in a finding's text, for
  instance). A noise event is a `noop` reschedule, nothing more.
- Commit prefixes the board counts as task commits: `feat`, `fix`, `test`,
  `docs`, `refactor`, `chore`. A ralphex fix commit says "review findings" and
  is counted separately. `docs(plans)` and `chore(board)` are never task
  commits: a plan edited mid-run, or the board itself.
- Two days of presence milestones (M20 to M27) ate a week's token quota, and
  the retrospective in `docs/design/2026-10-19-two-days-retrospective.md` found
  most of it in the recipe above rather than in the work. The cuts it approved
  are what the two scripts hold.
