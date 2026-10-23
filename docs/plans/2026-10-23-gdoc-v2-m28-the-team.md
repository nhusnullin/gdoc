# gdoc v2 Milestone 28: the team. Plan

2026-10-23. Nail allowed commits for this milestone the same day: one task,
one commit. Pushes, PRs, merges and tags still wait for his word. Whether the
retrospective commits af4a751 and 7be00df stay is still his call.

## Overview

Two days of the presence app (M20 to M27, 2026-10-16 to 18) shipped what
Nail asked for and ate a week's token quota. The retrospective found the
cost in full review runs on every milestone, about 107 Opus task
iterations, some twenty reviewer agents reading up to 120 PNGs each, and one
PM session that held product, runs, monitors, boards, releases and reports
in one context. Measured after the fact: M26 alone cost 220 million tokens
across its 27 worker sessions, 214 million of them cache reads, that is the
same context sent again on every turn. Nail approved ten cuts on 2026-10-23.

This milestone turns those cuts into a team organised by Team Topologies,
where Nail talks to one team lead only. Nail decides what; the lead decides
how:

- **Stream-aligned team "gdoc".** The team lead (Opus) is Nail's one
  interface: backlog, five product questions before a plan, decision entry,
  plan, release, summary. The product owner is a role inside the lead.
  Go engineer, macOS engineer and skills author are ralphex workers with a
  short area brief.
- **Platform "delivery"**, thinnest viable platform: scripts first.
  `make milestone` and `make release` do what the PM did by hand. A Sonnet
  platform operator wakes only when a recipe fails.
- **Enabling "quality"**, facilitating: the plan reviewer before a run, one
  design reviewer over 18 images after a run, Codex on contested decisions.
- **Complicated-subsystem "trust boundary"**: the trust guardian with the
  skeptic (Opus, read-only), consulted and signing when a plan touches the
  guard, auth, presence, `desktop.go` or an invariant.

Then a pilot milestone runs through the new team and is measured against
M26 by the same method.

## Context (from discovery)

- Milestones run today from `~/gdoc-mNN-board/launch-tasks.sh` and
  `launch-review.sh`, the board from `.ralphex/board/refresh_board.py`, the
  recipes in `.ralphex/board/README.md`.
- `.ralphex/config`: `external_review_tool = custom`,
  `custom_review_script = .ralphex/scripts/revmux-review.sh`,
  `max_external_iterations = 3`, `review_patience = 2`. ralphex also takes
  `--max-iterations`, `--max-external-iterations`, `--review-patience` and
  `--idle-timeout`. revmux takes `--profile`, `--lenses` and
  `--max-parallel`; `revmux-review.sh` calls it with no profile today.
- `Makefile` has `test`, `vet`, `build`, `dist`, `app`, `app-test`,
  `app-lint`, `tag`; `make tag` takes `vX.Y.0` only and commits, tags and
  pushes.
- `.claude/` holds settings only; no `.claude/agents/` yet. The plan
  reviewer (`planning:plan-review`) and `codex-second-opinion` exist as
  user-level agents.
- `docs/v2/DECISIONS.md`: 5,506 lines, about 84k tokens, the register table
  at the top (90 rows), 86 entries; M20 starts at the entries of 2026-10-16.
  `docs/v2/MEASURED.md`: 1,386 lines. `go/cmd/gdoc/doc.go`: 1,403 lines, 30
  sections, the only file `go doc` shows for the command.
- Worker transcripts land in `~/.claude/projects/<worktree path>/*.jsonl`
  with a `usage` block per message: that is the meter.
- Design notes: the brainstorm file in the session scratchpad
  (`team-topology-design.md`); the retrospective in
  `docs/design/2026-10-19-two-days-retrospective.md`.

## Development Approach

- **Testing approach: TDD** where code exists. Every script gets a `DRY=1`
  mode that prints the commands it would run, and a test that runs it in dry
  mode against a temporary repository and asserts the commands.
- One commit per task, as Nail allowed. No push, PR, merge or tag without
  his word.
- CLAUDE.md is not edited (approved item 9). The team pages link from
  `.ralphex/board/README.md`, which CLAUDE.md already names.
- `make test` keeps the race flag; live variables unset.

## Testing Strategy

- Script tests in `.ralphex/board/tests/`, run by `make platform-test`: dry
  runs of `milestone` and `release`, the OAuth recipe against a recorded
  log, the refusals of `release`.
- Agent definitions and team pages checked by `make team-check`:
  frontmatter, model, tools, every path named exists.
- The pilot's own tests are the Go suite plus its new boundary test.

## Solution Overview

The platform is two scripts behind two make targets, built on the caps
ralphex and revmux already have rather than on new machinery.
`milestone.sh` owns a run and writes one status file; the lead reads the
status file, not the log. `release.sh` owns the PR up to green checks and
stops; merge and tag happen only when the lead passes Nail's go. The agents
are short definitions with the model and tools pinned. The lead playbook is
one page the lead session loads at start.

## Technical Details

- `make milestone PLAN=docs/plans/<file> REVIEW=ui|go [DRY=1]`:
  - worktree `../gdoc-<mNN>` with its branch named from the plan file; board
    folder `~/gdoc-<mNN>-board/`
  - task phase: `ralphex --tasks-only --branch <b> --max-iterations=<tasks+2>
    --idle-timeout=<min>` with the session variables stripped as today
  - review phase by size: `go` keeps the config (3 external iterations,
    patience 2, revmux default roster); `ui` passes
    `--max-external-iterations=1 --review-patience=1` and an environment
    variable `REVMUX_LENSES` that `revmux-review.sh` turns into `--lenses`
    with the smaller set
  - writes `~/gdoc-<mNN>-board/status` in three lines: phase, last commit,
    next action for the lead; refreshes the board every 30 minutes
- Recipes in the script, only the ones ralphex does not already cover: a log
  line `Failed to authenticate` stops the run and writes `needs /login`; a
  ralphex that exited with tasks left is relaunched once, then the run
  stops. Stalled workers are `--idle-timeout`'s job.
- Budget: the plan states `Tasks: N` and `Review: ui|go`; those become the
  caps above. No separate estimate file.
- `make release BRANCH=<b> [GO=vX.Y.0] [DRY=1]`: without `GO`, push the
  branch, `gh pr create` with the plan's summary, wait for checks, write
  `ready: PR <n>, checks green` and stop. With `GO=vX.Y.0`, and only on a PR
  whose checks are green: `gh pr merge --merge`, pull, `make tag
  VERSION=vX.Y.0`, wait for the release workflow, verify the arm64 zip
  (checksum, bundle present, codesign, plist version, architectures, `gdoc
  help --json` version), write `released vX.Y.0, verified` or the failing
  step. The lead passes `GO` only after Nail's message.

## What Goes Where

- **Implementation Steps**: scripts, make targets, tests, agent definitions,
  the playbook, the meter and its baseline, the pilot plan.
- **Post-Completion**: the pilot run and its measurement by the lead with
  Nail; Nail's go for the pilot's push and PR; the fixture
  from his real record as its own milestone; his first days with the lead
  as the only interface.

## Implementation Steps

### Task 1: `make milestone`, the run as a service

**Files:**
- Create: `.ralphex/board/milestone.sh`, `.ralphex/board/tests/test_milestone.sh`,
  `.ralphex/board/tests/fixtures/oauth-expired.log`
- Modify: `Makefile`, `.ralphex/scripts/revmux-review.sh`

- [x] `milestone.sh` with the worktree, branch, board folder, task phase,
      review phase by size and status file as in Technical Details; `DRY=1`
      prints every command and runs none
- [x] `revmux-review.sh` passes `--lenses "$REVMUX_LENSES"` when the variable
      is set and behaves as today when it is not
- [x] the OAuth recipe and the one relaunch
- [x] test: dry runs for `REVIEW=ui` and `REVIEW=go` print the expected
      ralphex flags and lens set
- [x] test: the OAuth fixture log makes the script write `needs /login`
- [x] `make platform-test` green

### Task 2: `make release`, the PR up to green and the go

**Files:**
- Create: `.ralphex/board/release.sh`, `.ralphex/board/tests/test_release.sh`
- Modify: `Makefile`

- [x] `release.sh` as in Technical Details: without `GO` it stops at a green
      PR; with `GO` it merges, tags and verifies
- [x] test: a dry run without `GO` prints push, PR and wait, and nothing
      after
- [x] test: a dry run with `GO=v2.20.0` prints merge, pull, tag, wait and the
      six verification commands in that order
- [x] test: `GO=v2.19.1` and a red check are each refused by name
- [x] `make platform-test` green

### Task 3: the agent definitions

**Files:**
- Create: `.claude/agents/gdoc-platform-operator.md`,
  `.claude/agents/gdoc-design-reviewer.md`,
  `.claude/agents/gdoc-trust-guardian.md`,
  `.ralphex/board/tests/test_team.sh`
- Modify: `Makefile`

- [x] platform operator: Sonnet, Bash and Read; reads the status file and
      the README recipes; acts only when a recipe failed; reports three lines;
      never pushes, merges or tags
- [x] design reviewer: Sonnet, Read only; the principles checklist, the
      mockup, the dark whole-panel renders at 1x (18 images), the light ones
      only where dark has a finding, captions as text from `index.html`;
      rates blocking / should-fix / note and never fixes
- [x] trust guardian with the skeptic: Opus, Read, Grep and Glob; the
      CLAUDE.md invariants, `guard`, `auth`, `presence`, `desktop.go`; signs
      a decision entry or names its veto
- [x] the plan reviewer and Codex are named by their existing agent names
- [x] test: `make team-check` checks frontmatter, model, tools and that every
      path a definition names exists

### Task 4: the lead playbook and the team page

**Files:**
- Create: `docs/team/README.md`, `docs/team/lead.md`
- Modify: `.ralphex/board/README.md` (one line pointing at `docs/team/`)

- [x] `docs/team/README.md`: the four teams, their seats and models, the
      interaction modes, the day's flow, from the brainstorm file
- [x] `docs/team/lead.md`, one page: the five product questions as a
      template; who to call and when; message Nail only on a decision, a
      blocked run he must act on, or a release; one milestone and one review
      a day; `make release` without `GO` until Nail says; compact after a
      release; never re-read renders the reviewer read
- [x] test: `make team-check` also checks every path the two pages name

### Task 5: the meter and the baseline, before the pilot

**Files:**
- Create: `.ralphex/board/meter.py`, `.ralphex/board/tests/test_meter.sh`,
  `docs/team/measure.md`

- [x] `meter.py <worktree path>` sums the `usage` blocks of every transcript
      under `~/.claude/projects/<that path>/`: input, output, cache read,
      cache creation, sessions; and splits task sessions from review sessions
      by the ralphex log's phase markers
- [x] test: a fixture folder of two small transcripts gives the expected
      sums
- [x] `docs/team/measure.md`: the method in a paragraph, and the baseline
      run on M24a, M25, M26 and M27 as a table per task and per review round
- [x] the file ends with an open line for Nail's confirmation of the method
      and the baseline, which the lead records before the pilot runs

### Task 6: the pilot plan

**Files:**
- Create: `docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md`

- [x] the pilot is the two approved document items. DECISIONS.md and
      MEASURED.md entries before 2026-10-16 move to
      `docs/v2/DECISIONS-archive.md` and `docs/v2/MEASURED-archive.md`; the
      register stays whole in DECISIONS.md with each row saying which file
      holds its entry, so a citation such as "DECISIONS.md, 2026-10-04" in
      CLAUDE.md or a Go comment still resolves through the register; a
      boundary test checks every register row has its entry in one of the
      two files
- [x] `go/cmd/gdoc/doc.go` keeps the package comment and every rule CLAUDE.md
      or another file sends a reader to; the long walkthrough sections move
      to topic files as file comments; `TestOnlyThePackageCommentReachesGoDoc`
      and `TestEveryPackageHasExactlyOnePackageComment` pass
- [x] the pilot plan states `Tasks:` and `Review: go`, follows the lead
      playbook, and the plan reviewer reads it before the run

### Task 7: Verify acceptance criteria

- [x] every item of the Overview this run builds is in place: two make
      targets, three agents, the team page and the lead playbook, the meter
      with its baseline, the pilot plan
- [x] `make platform-test`, `make team-check`, `make vet`,
      `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`

### Task 8: [Final] Update documentation

- [ ] `.ralphex/board/README.md` leads with `make milestone` and
      `make release`; the hand recipes move under "when the script cannot"
- [ ] CLAUDE.md untouched (approved item 9)
- [ ] move this plan to `docs/plans/completed/`

## Post-Completion

These are the lead's, with Nail, after this run merges. A ralphex worker
cannot run a milestone or ask Nail anything.

**The pilot runs through the team.** Nail confirms the meter's method and
baseline in `docs/team/measure.md`. The lead session loads
`docs/team/lead.md`, runs `make milestone` on the pilot plan and reads only
the status file while it runs. The pilot commits one commit per task on its
local branch; `make release` runs without `GO`, and the push and PR wait for
Nail.

**The pilot is measured.** `meter.py` on the pilot's worktree, per task and
per review round, next to the baseline; the lead session's own context at
the end; the number of Nail's messages. Verdict: Nail spoke only to the
lead; zero blocking at the end; per task and per review round at most a
third of the baseline; nothing reverted the next day. Written into
`docs/team/measure.md`.

- Nail's go for the pilot's push and PR, and for any release.
- The fixture from Nail's real record (approved item 6) is its own
  milestone, because the repo is public: keep only allowlisted fields, give
  `doc` and `made` fake ids from one map, shift `at` to a fake date,
  regenerate `run`, `pid` and `hold`, use fixed-length titles, and test that
  no string of the input reaches the output.
- Nail's first days with the lead as his only interface; after a week, the
  week's quota against the two days of M20 to M27.
