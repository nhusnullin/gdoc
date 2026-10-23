#!/usr/bin/env python3
"""meter.py - what a milestone cost, read out of the transcripts it left.

The retrospective of M20 to M27 could only guess at the cost until somebody
summed the transcripts by hand. This script is that sum, so the pilot and the
baseline are measured the same way by the same code.

    .ralphex/board/meter.py ../gdoc-m26
    .ralphex/board/meter.py ../gdoc-m26 --log ~/gdoc-m26-board/ralphex-m26-tasks.log

Claude Code writes one JSONL transcript per session under
~/.claude/projects/<the worktree path with every other character a dash>/, and
every assistant message in it carries a `usage` block. That block is the meter:
input, output, cache read and cache creation tokens, as the API charged them.

Three things make a naive sum wrong, and all three are handled here:

- One API response is written as one line per content block, so a response with
  thinking, text and two tool calls appears four times, each line repeating the
  whole usage of the response. Summing lines overcounts by whatever the shape of
  the turn was: M26 reads as 403 million tokens by line and 238 million by
  response. The `requestId` is the response, so the sum is per requestId and the
  repeats are counted and reported, never added.
- A session's subagents write their own transcripts one level down, in
  `<session>/subagents/agent-*.jsonl`, and their responses appear in no parent
  file. Reading only the top level dropped 101 of M26's 238 million tokens.
  Every `*.jsonl` under the folder is read, which is why the file count is
  higher than the session count.
- A transcript folder holds every session that ever ran in that worktree, the
  tasks, the review and anything run by hand afterwards. The ralphex logs say
  when each iteration ran, so each response lands in the iteration whose window
  holds it, and what falls outside every window is reported as its own row
  rather than dropped or mixed in.

The log timestamps are local time, which is the machine that ran them, and the
transcript timestamps are UTC. Both are made aware before they are compared.
"""

import argparse
import glob
import json
import os
import re
import sys
from datetime import datetime, timedelta, timezone

# A marker line ralphex writes before it hands an iteration to a model, such as
# `--- task iteration 7 ---`, `--- task iteration 12/14 ---` on a newer build,
# `--- custom review iteration 2 ---` or `--- claude evaluating codex findings
# ---`. A newer ralphex stamps the time in front of it.
#
# The label must name an iteration, a review or the findings, because a model's
# own output reaches the same log and a turn that prints `--- a note ---` is not
# a phase.
MARKER = re.compile(
    r"^\s*(?:\[[\d :-]+\]\s*)?---\s*((?=.*(?:iteration|review|findings))(.+?))\s*---\s*$"
)

# The time ralphex puts in front of a line, either with the date or without it
# on an older build.
STAMP = re.compile(r"^\[(?:(\d{2}-\d{2}-\d{2}) )?(\d{2}:\d{2}:\d{2})\]")

# The four fields of a usage block this script adds up, in the order they are
# printed. Nothing else in the block is money.
FIELDS = ("input", "output", "cache read", "cache create")
KEYS = (
    "input_tokens",
    "output_tokens",
    "cache_read_input_tokens",
    "cache_creation_input_tokens",
)


def transcript_dir(worktree, projects):
    """The folder Claude Code keeps a worktree's transcripts in.

    The name is the absolute path with every character that is not a letter or a
    digit replaced by a dash, which is why `/Users/x/src/gdoc-m26` becomes
    `-Users-x-src-gdoc-m26`. The worktree itself need not still exist: a
    milestone's transcripts outlive the folder it ran in, and the baseline is
    read from runs whose worktrees are gone.
    """
    path = os.path.abspath(os.path.expanduser(worktree))
    return os.path.join(projects, re.sub(r"[^A-Za-z0-9]", "-", path))


def phase_of(label, log_name):
    """Which half of a run a marker belongs to: the tasks or the review."""
    if label.startswith("task iteration"):
        return "tasks"
    if "review" in label or "findings" in label:
        return "review"
    return "tasks" if "tasks" in os.path.basename(log_name) else "review"


def read_log(path):
    """The segments one ralphex log describes, in order.

    A segment is a marker and the window from its first timestamped line to the
    next marker's. The end is left open here and filled in once every log is
    read, because the review log of a run starts after the task log ends and the
    two are read separately.
    """
    segments = []
    day = None
    pending = None
    for line in open(path, encoding="utf-8", errors="replace"):
        marker = MARKER.match(line)
        if marker:
            pending = {
                "label": marker.group(1),
                "phase": phase_of(marker.group(1), path),
                "start": None,
                "end": None,
            }
            segments.append(pending)
            continue
        stamp = STAMP.match(line)
        if not stamp:
            continue
        if stamp.group(1):
            day = stamp.group(1)
        if day is None:
            # A log that stamps the time and not the date, before any date was
            # seen. The file's own day is the only date there is.
            day = datetime.fromtimestamp(os.path.getmtime(path)).strftime("%y-%m-%d")
        when = datetime.strptime(f"{day} {stamp.group(2)}", "%y-%m-%d %H:%M:%S").astimezone()
        if pending is not None and pending["start"] is None:
            pending["start"] = when
        if segments:
            segments[-1]["end"] = when
    return [s for s in segments if s["start"] is not None]


def close_windows(segments, grace=3600):
    """Give every segment the end of the one that follows it in time.

    The windows are contiguous: a model's last turn of one iteration and the
    first of the next are a second apart, so a gap between them would only put
    responses outside the run that plainly belong in it. The final iteration has
    no successor, so it ends at its own last stamped line plus one hour, which
    is long enough for a last turn to finish writing and short enough that a
    session somebody ran in the same worktree the next day is not charged to the
    run.
    """
    segments.sort(key=lambda s: s["start"])
    for i, seg in enumerate(segments[:-1]):
        seg["end"] = segments[i + 1]["start"]
    if segments:
        last = segments[-1]
        last["end"] = (last["end"] or last["start"]) + timedelta(seconds=grace)
    return segments


def default_logs(worktree):
    """Every ralphex log in the board folder of the run that worktree held.

    One run can leave more than the two logs `make milestone` writes: M24a was
    resumed and relaunched, and its board folder holds four. Any
    `ralphex-*.log` beside them describes a phase of the same run, and the
    windows are sorted by time afterwards, so reading all of them is what keeps
    an iteration out of the "outside the run" row.
    """
    name = os.path.basename(os.path.abspath(os.path.expanduser(worktree)))
    found = re.search(r"m\d+[a-z]?", name)
    if not found:
        return []
    board = os.path.join(os.path.expanduser("~"), f"gdoc-{found.group(0)}-board")
    return sorted(glob.glob(os.path.join(board, "ralphex-*.log")))


def blank_row():
    return {"sessions": set(), "responses": 0, "tokens": [0, 0, 0, 0]}


def add(row, session, usage):
    row["sessions"].add(session)
    row["responses"] += 1
    for i, key in enumerate(KEYS):
        row["tokens"][i] += usage.get(key, 0) or 0


def transcripts(folder):
    """Every transcript file of a worktree, the subagents' included.

    Claude Code writes a session's own turns to `<folder>/<session>.jsonl` and
    the turns of each subagent it spawned to
    `<folder>/<session>/subagents/agent-*.jsonl`. A subagent's responses are
    charged like any other and appear in no parent file: reading only the top
    level dropped 101 million tokens of M26 alone, and the team's seats are
    subagents, so the pilot would be undercounted the most.

    What this does not read, named so no reader mistakes the review numbers for
    the whole: revmux is a separate program, not an agent a session spawns, and
    each lens writes its own stream into
    `<worktree>/.revmux/tasks/<task>/round-NN/agents/<lens>.jsonl`, which is
    nowhere under this folder. Those lines carry `usage` and no `requestId`, so
    `message.id` would be the key. Measured once, they are 87.6 million tokens
    across M24a, M25, M26 and M27, all of it review. `docs/team/measure.md`
    states the floor that follows. Adding them is worth doing before a UI
    milestone runs under the new team, because cut 2 shrinks the lens roster and
    that saving is invisible here.
    """
    return sorted(glob.glob(os.path.join(folder, "**", "*.jsonl"), recursive=True))


def read_usage(folder):
    """Every charged response in a transcript folder, once each.

    Returns the responses as (time, session, usage) and the number of lines that
    repeated a response already counted.
    """
    responses = []
    repeats = 0
    seen = set()
    for path in transcripts(folder):
        for line in open(path, encoding="utf-8", errors="replace"):
            line = line.strip()
            if not line:
                continue
            try:
                row = json.loads(line)
            except json.JSONDecodeError:
                continue
            message = row.get("message") or {}
            usage = message.get("usage")
            if not isinstance(usage, dict):
                continue
            key = (path, row.get("requestId") or row.get("uuid"))
            if key in seen:
                repeats += 1
                continue
            seen.add(key)
            stamp = row.get("timestamp")
            when = None
            if stamp:
                try:
                    when = datetime.fromisoformat(stamp.replace("Z", "+00:00"))
                except ValueError:
                    when = None
                if when is not None and when.tzinfo is None:
                    when = when.replace(tzinfo=timezone.utc)
            session = row.get("sessionId") or os.path.basename(path)
            responses.append((when, session, usage))
    return responses, repeats


def millions(n):
    return f"{n / 1_000_000:.2f}M"


def print_table(title, rows):
    """One table: a row per segment, with its sessions, responses and tokens."""
    width = max([len(name) for name, _ in rows] + [len(title)])
    head = f"{title.ljust(width)}  {'sessions':>8}  {'responses':>9}"
    for field in FIELDS:
        head += f"  {field:>12}"
    head += f"  {'total':>12}"
    print(head)
    print("-" * len(head))
    for name, row in rows:
        total = sum(row["tokens"])
        line = f"{name.ljust(width)}  {len(row['sessions']):>8}  {row['responses']:>9}"
        for value in row["tokens"]:
            line += f"  {value:>12,}"
        line += f"  {total:>12,}"
        print(line)


def main(argv):
    parser = argparse.ArgumentParser(
        description="Sum what a milestone's sessions cost, per task and per review round."
    )
    parser.add_argument("worktree", help="the worktree the run happened in, existing or gone")
    parser.add_argument(
        "--projects",
        default=os.path.join(os.path.expanduser("~"), ".claude", "projects"),
        help="where Claude Code keeps its transcripts",
    )
    parser.add_argument(
        "--log",
        action="append",
        default=[],
        help="a ralphex log to read the iteration windows from; may repeat",
    )
    args = parser.parse_args(argv)

    folder = transcript_dir(args.worktree, args.projects)
    if not os.path.isdir(folder):
        print(f"meter: no transcripts at {folder}", file=sys.stderr)
        return 1

    logs = args.log or default_logs(args.worktree)
    segments = close_windows([s for log in logs for s in read_log(log)])
    for seg in segments:
        seg["row"] = blank_row()

    responses, repeats = read_usage(folder)
    outside = blank_row()
    for when, session, usage in responses:
        placed = False
        if when is not None:
            for seg in segments:
                if seg["start"] <= when < seg["end"]:
                    add(seg["row"], session, usage)
                    placed = True
                    break
        if not placed:
            add(outside, session, usage)

    everything = blank_row()
    for _, session, usage in responses:
        add(everything, session, usage)

    print(f"worktree:    {os.path.abspath(os.path.expanduser(args.worktree))}")
    print(f"transcripts: {folder}")
    print(f"             {len(transcripts(folder))} files, "
          f"{len(everything['sessions'])} sessions, {len(responses)} responses, "
          f"{repeats} repeats dropped")
    for log in logs:
        print(f"log:         {log}")
    print()

    rows = [(seg["label"], seg["row"]) for seg in segments]
    phases = {}
    for seg in segments:
        phases.setdefault(seg["phase"], blank_row())
        row = phases[seg["phase"]]
        row["sessions"] |= seg["row"]["sessions"]
        row["responses"] += seg["row"]["responses"]
        for i in range(4):
            row["tokens"][i] += seg["row"]["tokens"][i]
    for phase in ("tasks", "review"):
        if phase in phases:
            rows.append((f"{phase}, all iterations", phases[phase]))
    rows.append(("outside the run", outside))
    rows.append(("the whole worktree", everything))
    print_table("segment", rows)
    print()
    print(f"the whole worktree: {millions(sum(everything['tokens']))}, "
          f"{millions(everything['tokens'][2])} of it cache reads")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
