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
  `<folder>/<session>.jsonl`, and the turns of every subagent it spawned are one
  level down, in `<session>/subagents/agent-*.jsonl`. Those responses are
  charged like any other and appear in no parent file. They are most of a review
  round, because revmux's lenses are subagents: reading only the top level put
  M26 at 136.8 million and dropped 101.0 million. The meter reads every
  `*.jsonl` under the folder, which is why it reports more files than sessions.
- **A window per iteration.** A transcript folder holds every session that ever
  ran in that worktree, including ones run by hand after the milestone. Windows
  run from one marker to the next, the last one ends an hour after its last
  logged line, and whatever falls outside every window is its own row, never
  dropped and never mixed into a phase.

Cache reads dominate every row. That is the same context sent again on each
turn, which is the cost the team is built to cut: a shorter context, fewer
turns per task, and one review round where the old recipe ran five.

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
  14.6, and 30.5 against 14.8 on M24a. The round is one session fanning out to
  revmux's lenses, each a subagent with its own copy of the context. Cut 2, one
  round over three lenses for a UI milestone, is aimed at the most expensive row
  in this table.
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

## Nail's confirmation

The method and the baseline above are the lead's reading. They are not settled
until Nail says so, because every comparison after this one is made against
them.

- [ ] Nail confirms the method, and the baseline of 14.6M per task iteration
      and 19.7M per review round, before the pilot runs.

One thing to confirm with it. The rule Nail approved is "a third of the
baseline", and the two numbers under it are arithmetic on the baseline. The
baseline rose when the subagents were added to the sum, so the bar rose with it,
from 3.8M to 4.9M per task iteration and from 2.0M to 6.6M per review round. A
review round that fans out to subagents was always costing what it costs; only
the measurement changed. If the intent was the absolute numbers rather than the
third, the review bar is the one to say so about.
