# What a milestone cost, and the baseline to beat

The retrospective of M20 to M27 said two days of the presence app ate a week's
quota. Nobody could say which part ate it until the transcripts were summed by
hand. `meter.py` is that sum as code, so the pilot and the four milestones
before it are measured the same way by the same script, and the verdict is a
number rather than a feeling.

## The method

Claude Code writes one JSONL transcript per session under
`~/.claude/projects/<the worktree path with every other character a dash>/`,
and every assistant message in it carries a `usage` block: input, output, cache
read and cache creation tokens, as the API charged them. `meter.py` reads every
transcript of one worktree, sums those four fields, and puts each response in
the iteration that was running when it was written. The iteration windows come
from the ralphex logs `make milestone` writes into `~/gdoc-<mNN>-board/`, whose
marker lines name every task iteration and every review round.

    .ralphex/board/meter.py ../gdoc-m26

That prints the header, a row per iteration, a row per phase, a row for what
fell outside the run and the whole worktree. With no `--log` it reads every
`ralphex-*.log` in the board folder of the milestone its worktree is named
after, because a resumed run leaves more than two. The worktree itself may be
gone: the transcripts outlive it, which is how the baseline below was read.

Three rules make the sum right, and all three are in `meter.py` with a test:

- **One response, not one line.** Claude Code writes one line per content block
  of a response, and each line repeats the whole usage of that response. A turn
  with thinking, text and two tool calls appears four times. The meter sums per
  `requestId` and counts the repeats it dropped. M26's own transcripts read as
  403.5 million tokens by line and 237.8 million by response. The 237.8 million
  is what was charged.
- **The subagents too.** A session's own turns are in
  `<folder>/<session>.jsonl`, and the turns of every agent it spawned with the
  Task tool are one level down, in `<session>/subagents/agent-*.jsonl`. Those
  responses are charged like any other and appear in no parent file: reading
  only the top level put M26 at 136.8 million and dropped 101.0 million, across
  twelve agents spawned by four sessions, the reviewers over the renders among
  them. The meter reads every `*.jsonl` under the folder, which is why it
  reports more files than sessions.
- **A window per iteration.** A transcript folder holds every session that ever
  ran in that worktree, including ones run by hand after the milestone. Windows
  run from one marker to the next, the last one ends an hour after its last
  logged line, and whatever falls outside every window is its own row, never
  dropped and never mixed into a phase.

Cache reads dominate every row. That is the same context sent again on each
turn, which is the cost the team is built to cut: a shorter context, fewer
turns per task, and one review round where the old recipe ran five.

One thing the meter does not read, so every review number below is a floor.
revmux is a separate program, not an agent a session spawns: `revmux-review.sh`
runs the binary, and each lens writes its own stream into
`<worktree>/.revmux/tasks/<task>/round-NN/agents/<lens>.jsonl`, which is nowhere
under `~/.claude/projects/`. Those files carry `usage` the same way, and summed
the same way, once per response, they are 56.4M on M24a, 5.6M on M25, 13.1M on
M26 and 12.5M on M27: 87.6M, all of it review. Added in, the four runs' review
total is 814.9M over 37 rounds, 22.0M a round rather than 19.7M.

Two reasons the bar below is still the one to use. The baseline and the pilot
are read by the same script, so what it cannot see is missing from both sides of
the comparison. And the pilot is a Go milestone with the full lens roster, so
cut 2, the smaller roster for a UI milestone, is not what it measures. Reading
the lens logs is a change to the meter, not to the method, and it is worth
making before the first UI milestone runs under the new team: that is the one
where the cut is the thing being tested and the saving would be invisible.

## The baseline: M24a, M25, M26, M27

Read on 2026-10-23 from the transcripts of the four worktrees, with every
`ralphex-*.log` in `~/gdoc-<mNN>-board/` giving the windows. Every number is
tokens as charged, counted once per response.

| Run | Task iterations | Tasks, total | Per task iteration | Biggest iteration | Review rounds | Review, total | Per review round | Outside the run | Whole worktree | Sessions |
|---|---|---|---|---|---|---|---|---|---|---|
| M24a, the panel | 32 | 474.0M | 14.8M | 144.1M | 20 | 609.4M | 30.5M | 2.0M | 1085.3M | 49 |
| M25, gdoc opens the app | 10 | 86.5M | 8.6M | 28.2M | 2 | 5.4M | 2.7M | 0 | 91.8M | 11 |
| M26, the second morning | 11 | 160.8M | 14.6M | 70.8M | 9 | 77.0M | 8.6M | 0 | 237.8M | 17 |
| M27, the third look | 9 | 183.6M | 20.4M | 69.8M | 6 | 35.6M | 5.9M | 0 | 219.2M | 13 |
| **All four** | **62** | **904.8M** | **14.6M** | | **37** | **727.3M** | **19.7M** | | **1634.2M** | **90** |

Five readings of the table:

- **M24a cost more than the other three together.** 1,085.3 million tokens and
  49 sessions for one panel, against 548.8 million and 41 sessions for M25, M26
  and M27. It is the run that carried the full review on every milestone and the
  reviewers over the renders, which is what cut 2 and cut 3 take out.
- **A review round costs more than a task iteration.** 19.7 million against
  14.6, and 30.5 against 14.8 on M24a, and more than that once the lens logs
  above are added: 22.0 against 14.6. A round is a session, whatever it spawned
  with the Task tool, and revmux's own lenses beside it, each with its own copy
  of the context. Cut 2, one round over three lenses for a UI milestone, is
  aimed at the most expensive row in this table.
- **One task iteration costing 144.1 million is a context, not a task.** M24a's
  iteration 23 was one session sending its whole growing context back on every
  turn for 633 responses. The area briefs and the smaller plans aim at that row.
- **M24a's run was resumed twice, so its board holds four logs.** The meter
  reads every `ralphex-*.log` in the folder, which is why only 2.0 million of
  its 1,085.3 falls outside the windows. The other three runs have nothing
  outside theirs.
- **A repeated label is a relaunch.** M25's log holds four segments called
  `task iteration 7`, because the run was relaunched and ralphex appended to
  the same file. `make milestone` relaunches once at most, so the pilot's log
  holds at most one repeat.

The retrospective's own figures, for the record: M26 at 220 million across 27
worker sessions, 214 million of them cache reads. Its 220 million is M26's
top-level transcripts summed by line, which the meter reads as 219.7 million.
The meter's own answer for the same worktree is 237.8 million: every response
counted once, the subagents included. The sessions differ because the
retrospective counted each subagent as a session of its own; the meter counts
17, because a subagent's turns carry the session that spawned it.

## What the pilot has to beat

The verdict of the pilot, M29 the archive, is the one Nail approved on
2026-10-23:

- Nail spoke only to the lead.
- Zero blocking findings at the end of the review.
- **At most a third of the baseline per task iteration and per review round**,
  which against the all-four row means at most **4.9M per task iteration** and
  at most **6.6M per review round**.
- Nothing reverted the next day.

The pilot is a Go milestone, so its review keeps the full roster: the review
cap is the one to watch, not the one the UI cuts protect. The lead writes the
pilot's own table under this section when the run ends, beside the baseline
row, and adds two numbers the meter cannot read: the lead session's own context
at the end of the day, and the number of messages Nail had to send.

### The pilot: M29, the archive, on codex

Measured 2026-10-26 by the lead. The run went on codex (`CODEX=1`), so
`meter.py` finds no Claude transcript for M29's worktree. The codex numbers are
summed from `~/.codex/sessions`, every session whose `cwd` is the worktree,
each at its last `total_token_usage`. The run's window was 07:33 to 08:32 UTC.

| Run | Task iterations | Tasks, total | Per task iteration | Biggest iteration | Review rounds | Review, total | Per review round | Whole worktree | Sessions |
|---|---|---|---|---|---|---|---|---|---|
| M29, the archive (codex) | 6 | 9.8M | 1.6M | 2.5M | 2 | 7.3M | 3.7M | 17.1M | 15 |
| M29, Claude tokens | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |

Against the bars: 1.6M per task iteration is under 4.9M, and 3.7M per review
round is under 6.6M. Two cautions. The codex tokens are OpenAI's, billed on a
separate plan, so they do not compare one to one with Claude's. And 16.0M of
the 17.1M were cached input, which costs less than fresh input on either
side. M29 was docs and tests only, so M30 on codex is the first fair test on
code.

Nail sent one message about M29, the go to merge it without a tag. The
review ended with its fixes committed in `1a465e0`. Nothing was reverted on
the day it merged. The lead's own context at the end of the day was not
read.

### The night of 2026-10-26: M30 on codex, M31 on Claude

Measured 2026-10-27 by the lead. M30, the fixes, ran on codex, so its numbers
are summed from `~/.codex/sessions` the same way as M29's. The task phase ran
from 23:47 to 00:38 local time and the review from 00:38 to 01:11, and each
session is counted in the phase its start falls in. M31, the fourth look, ran
on Claude, so `meter.py` reads it. Its review called codex once as the outside
reviewer, which found nothing.

| Run | Task iterations | Tasks, total | Per task iteration | Biggest iteration | Review rounds | Review, total | Per review round | Whole worktree | Sessions |
|---|---|---|---|---|---|---|---|---|---|
| M30, the fixes (codex) | 9 | 15.3M | 1.7M | 3.1M | 2 | 23.8M | 11.9M | 39.1M | 21 |
| M31, the fourth look (Claude) | 10 | 68.0M | 6.8M | 12.1M | 1 | 0.1M | 0.1M | 68.2M | 11 |
| M31, its codex reviewer | 0 | 0 | 0 | 0 | 1 | 0.8M | 0.8M | 0.8M | 1 |

Against the bars of 4.9M per task iteration and 6.6M per review round:

- M30's tasks pass at 1.7M. Its review fails at 11.9M per round, because a
  codex review round fans out to several sessions. The first fair test on code
  says codex is cheap at writing and not at reviewing. 37.0M of its 39.1M were
  cached input.
- M31's tasks fail at 6.8M, under half the 14.6M baseline but over the third.
  66.4M of its 68.2M were cache reads. Its review passes because it found
  nothing to fix.

After the runs the lead fixed three things by hand: a path in this page that CI
does not have, a flaky Swift test re-run once, and a placeholder the renderer
drew over every DOCUMENTS row, which blocked the design review. Nail sent two
messages for the night, the go on the plan and the merge of both. The lead's
own context at the end of the night was not read.

## 2026-10-27: M32 on Claude

| Run | Task iterations | Tasks, total | Per task iteration | Biggest iteration | Review rounds | Review, total | Per review round | Whole worktree | Sessions |
|---|---|---|---|---|---|---|---|---|---|
| M32, the quiet watch (Claude) | 9 | 35.3M | 3.9M | 8.7M | 1 | 17.2M | 17.2M | 52.8M | 11 |

- M32's tasks pass at 3.9M, under the bar of 4.9M. Nine tasks ran in nine
  iterations, with no relaunch, in 55 minutes.
- Its review fails at 17.2M against the bar of 6.6M. 13.7M of it is one
  Claude pass over critical and major findings, and 3.5M is Claude weighing
  what the codex peer found. 51.0M of the run's 52.8M were cache reads.

The lead ran the suites by hand after the run and changed nothing. Nail sent
three messages for the run: the go, "feel free to start", and the merge and
release. The lead's own context at the end of the day was not read.

## 2026-10-28: M32a on Claude

| Run | Task iterations | Tasks, total | Per task iteration | Biggest iteration | Review rounds | Review, total | Per review round | Whole worktree | Sessions |
|---|---|---|---|---|---|---|---|---|---|
| M32a, four small fixes (Claude) | 8 | 28.9M | 3.6M | 7.0M | 1 | 0.2M | 0.2M | 29.1M | 9 |

- M32a's tasks pass at 3.6M, under the bar of 4.9M. Eight tasks ran in eight
  iterations, with no relaunch, in 44 minutes.
- Its review passes at 0.2M against the bar of 6.6M. The Claude pass found
  nothing to fix, and weighing the codex peer's findings took 0.1M. 28.0M of
  the run's 29.1M were cache reads.

After the run the lead fixed two captions in the panel's test scenes, which
the design review had named, and changed nothing else. Nail sent two messages
for the run: "do fixes" and the release.

## Nail's confirmation

The method and the baseline above are the lead's reading. They are not settled
until Nail says so, because every comparison after this one is made against
them.

- [ ] Nail confirms the method, and the baseline of 14.6M per task iteration
      and 19.7M per review round as the meter reads it, before the pilot runs.
      The review figure is a floor: with revmux's own lens logs added it is
      22.0M, and the bar a third of it at 7.3M rather than 6.6M.

One thing to confirm with it. The rule Nail approved is "a third of the
baseline", and the two numbers under it are arithmetic on the baseline. The
baseline rose when the subagents were added to the sum, so the bar rose with it,
from 3.8M to 4.9M per task iteration and from 2.0M to 6.6M per review round. A
review round that fans out to subagents was always costing what it costs; only
the measurement changed. If the intent was the absolute numbers rather than the
third, the review bar is the one to say so about.

## 2026-10-28: M32b on Claude

| Run | Task iterations | Tasks, total | Per task iteration | Biggest iteration | Review rounds | Review, total | Per review round | Whole worktree | Sessions |
|---|---|---|---|---|---|---|---|---|---|
| M32b, the logo at its old size (Claude) | 3 | 7.5M | 2.5M | 5.1M | 1 | 0.1M | 0.1M | 7.6M | 4 |

- M32b's tasks pass at 2.5M, under the bar of 4.9M. Three tasks ran in three
  iterations, with no relaunch, in 12 minutes.
- Its review passes at 0.1M against the bar of 6.6M. The codex peer found
  nothing, and Claude weighing it took under 0.1M. 7.3M of the run's 7.6M were
  cache reads.

The plan reviewer found two blocking gaps in the lead's first draft before the
run, a task that could not end green on its own and a test fixture that named
the row the run makes known, and the lead folded both in; the run then needed
no relaunch. The lead ran the suites and the offline build by hand after the
run and changed nothing. Nail sent four messages for the run: the go, a
question on how the logo moves, "merged", and the tag step, which he ran
himself because the guard refused it to the lead. The lead's own context at
the end of the day was not read.

## 2026-10-28: M32c on Claude

| Run | Task iterations | Tasks, total | Per task iteration | Biggest iteration | Review rounds | Review, total | Per review round | Whole worktree | Sessions |
|---|---|---|---|---|---|---|---|---|---|
| M32c, the fifth look (Claude) | 9 | 47.9M | 5.3M | 12.5M | 1 | 3.1M | 3.1M | 51.0M | 10 |

- M32c's tasks fail at 5.3M against the bar of 4.9M. Eight tasks ran in nine
  iterations in 67 minutes: the ninth was a closing pass that found every box
  ticked and cost 0.4M. The biggest iteration, 12.5M, was Task 5, the card
  itself, which wired the rest machine, the carrier and the forced scene in
  one task. Without the closing pass the eight iterations average 5.9M, so
  the miss is the size of the tasks and not the extra iteration.
- Its review passes at 3.1M against the bar of 6.6M. The codex peer found
  two things and fixed one, a title's tooltip lost under a row with no card,
  and Claude weighing it took 1.2M. 49.6M of the run's 51.0M were cache reads.

The plan reviewer ran twice before the run and found, among other gaps, that
a click and a held press sat on two different views, so a click on a name
could open nothing; the lead folded it in and the run needed no relaunch. The
design reviewer found nothing blocking and two should-fix items, a render
that hides the rows under the forced card and an open icon on the stuck-run
card, and the lead changed nothing after the run. Nail sent three messages
for the run: the go on the recommendation, "did you finish?", and the GO for
v2.25.0, which the lead ran this time. The lead's own context at the end of
the day was not read.
