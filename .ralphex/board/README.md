# The run board

One HTML page that shows where a ralphex milestone run is, published as a Claude
Code artifact and republished on every task iteration and every review round.
Nail follows the run on his phone; the chat carries one short line per refresh
and nothing else. Used for M2 to M5 (2026-09-06 to 2026-09-08). Nail asked for
it to be kept.

The page reads what ralphex already writes: its logs, the plan's checkboxes,
`git log main..<branch>`. Nothing here talks to ralphex or changes the run.

## Files

| File | Holds |
|---|---|
| `board.html` | the page template. Header placeholders and an empty snapshot the script fills |
| `refresh_board.py` | rebuilds the snapshot from the logs, the plan and git. Prints one status line for the artifact label |

## The recipe, as a Claude Code session runs it

The whole run lives in one Claude Code session with a scratchpad directory for
the logs and the page. `SCRATCH` below is that directory.

1. **Plan on main.** Write `docs/plans/<date>-<slug>.md`, commit and push it on
   `main`. ralphex creates the branch from `main`, and the reviewer must not see
   its own scripts as new code.

2. **Start the task phase, detached.** ralphex refuses to start inside a Claude
   Code session unless the session variables are stripped:

   ```bash
   env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT -u CLAUDE_CODE_SESSION_ID -u CLAUDE_CODE_CHILD_SESSION \
     nohup ralphex --tasks-only docs/plans/<plan>.md > "$SCRATCH/ralphex-<m>-tasks.log" 2>&1 &
   echo $!
   ```

3. **Create the page once.** A `warns.json` is a list of
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

   Publish `mN-run-board.html` with the Artifact tool, with a favicon and a
   one-sentence description. Every later publish uses the same file path so the
   URL stays.

4. **Arm a Monitor on the log** (persistent), with a filter that emits one line
   per event and covers failure as well as progress:

   ```
   ^--- task iteration|ALL_TASKS_DONE|task execution completed|TASK_FAILED|max iterations|^error:|session limit|usage limit|^completed in
   ```

   plus a loop that waits on the pid and prints `PROCESS EXITED` when it goes.

5. **Loop.** `/loop` in dynamic mode with a 1800 s fallback heartbeat. On each
   Monitor event: run `refresh_board.py --html ... --pid <pid>`, republish the
   page with the label the script printed, one short line in chat, reschedule.

6. **When the tasks finish** (`ALL_TASKS_DONE`, process exited), start the
   review the same way and re-arm a Monitor on its log:

   ```bash
   env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT -u CLAUDE_CODE_SESSION_ID -u CLAUDE_CODE_CHILD_SESSION \
     nohup ralphex --external-only > "$SCRATCH/ralphex-<m>-review.log" 2>&1 &
   ```

   Review filter:

   ```
   ^--- (custom|codex|claude) review|^--- claude evaluating|── complete ──|findings: |NO ISSUES FOUND|TASK_FAILED|^error:|session limit|^completed in
   ```

7. **When the review exits** (`completed in`): `make test`, `make vet`,
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
- The Monitor's log filter matches prose too (`OAuth` in a finding's text, for
  instance). A noise event is a `noop` reschedule, nothing more.
- Commit prefixes the board counts as task commits: `feat`, `fix`, `test`,
  `docs`, `refactor`, `chore`. A ralphex fix commit says "review findings" and
  is counted separately.
