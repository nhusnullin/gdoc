#!/usr/bin/env bash
# test_meter.sh - what the meter reads out of a transcript folder, asserted
# against two small transcripts whose numbers are chosen so every sum is
# readable at a glance.
#
# The fixture is built here rather than committed, because a transcript's
# timestamps are UTC and the ralphex log's are the machine's local time: the
# two are written from one list of instants, so the test says the same thing
# wherever it runs. The numbers are the fixture: a response of the first task
# iteration carries 10 input, 100 output, 1,000 cache read and 10,000 cache
# creation tokens, the next 20 of each, and so on, so a wrong total names
# itself.
#
# Run by `make platform-test`.

set -uo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
METER="$HERE/../meter.py"

fails=0
pass() { printf '  ok   %s\n' "$1"; }
bad() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

holds() {
    local what=$1 text=$2 want=$3
    if printf '%s' "$text" | grep -qE -- "$want"; then
        pass "$what: $want"
    else
        bad "$what: expected a line matching \"$want\""
        printf '%s\n' "$text" | sed 's/^/       | /'
    fi
}

# a_run writes the fixture into a temporary folder and prints its path: a
# projects folder holding the two transcripts of the worktree /Users/nobody/
# gdoc-m99, and the two logs of its run.
#
# The run is two task iterations and one review iteration. Session A is the
# ralphex worker, session B the review and its subagent. One response of
# session A is written as two lines, one per content block, which is the repeat
# the meter must drop. One response of session B is an hour before the run
# started, which is the row the meter must keep outside it. The subagent's
# response is where Claude Code puts it, in
# <session>/subagents/agent-1.jsonl, carrying the session that spawned it.
a_run() {
    local tmp
    tmp=$(mktemp -d)
    python3 -I - "$tmp" <<'PY'
import json, os, sys
from datetime import datetime, timedelta, timezone

tmp = sys.argv[1]
folder = os.path.join(tmp, "projects", "-Users-nobody-gdoc-m99")
os.makedirs(folder)

# One list of instants, in the machine's own local time. The logs print them as
# local time and the transcripts as UTC, which is exactly how a real run leaves
# them.
base = datetime(2026, 10, 23, 9, 0, 0).astimezone()
at = lambda minutes: base + timedelta(minutes=minutes)

def log(path, lines):
    with open(path, "w") as f:
        for line in lines:
            f.write(line + "\n")

stamp = lambda when: when.strftime("[%y-%m-%d %H:%M:%S]")

log(os.path.join(tmp, "ralphex-m99-tasks.log"), [
    "ralphex v1.6.1-fixture",
    "starting ralphex loop (max 4 iterations) (tasks-only mode)",
    "",
    "starting task execution phase",
    "--- task iteration 1 ---",
    f"{stamp(at(0))} the first task",
    f"{stamp(at(9))} the first task is committed",
    "--- task iteration 2 ---",
    f"{stamp(at(10))} the second task",
    f"{stamp(at(19))} the second task is committed",
])
log(os.path.join(tmp, "ralphex-m99-review.log"), [
    "ralphex v1.6.1-fixture",
    "--- custom external review ---",
    "--- custom review iteration 1 ---",
    f"{stamp(at(20))} the review",
    f"{stamp(at(29))} the review found nothing",
])

def response(session, when, scale, blocks, sidechain=False):
    """One API response as Claude Code writes it: one line per content block,
    each repeating the whole usage."""
    out = []
    for i, kind in enumerate(blocks):
        out.append(json.dumps({
            "type": "assistant",
            "uuid": f"{session}-{scale}-{i}",
            "requestId": f"req_{session}_{scale}",
            "sessionId": session,
            "isSidechain": sidechain,
            "timestamp": when.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.000Z"),
            "message": {
                "model": "claude-opus-5",
                "content": [{"type": kind}],
                "usage": {
                    "input_tokens": 10 * scale,
                    "output_tokens": 100 * scale,
                    "cache_read_input_tokens": 1000 * scale,
                    "cache_creation_input_tokens": 10000 * scale,
                },
            },
        }))
    return out

lines = []
lines += response("sessionA", at(1), 1, ["text", "tool_use"])   # the repeat
lines += response("sessionA", at(5), 2, ["text"])
lines += response("sessionA", at(12), 3, ["text"])
with open(os.path.join(folder, "sessionA.jsonl"), "w") as f:
    f.write("\n".join(lines) + "\n")

lines = []
lines += response("sessionB", at(-60), 6, ["text"])             # before the run
lines += response("sessionB", at(21), 4, ["text"])
with open(os.path.join(folder, "sessionB.jsonl"), "w") as f:
    f.write("\n".join(lines) + "\n")

# The subagent of session B, one level down, as Claude Code writes it.
nested = os.path.join(folder, "sessionB", "subagents")
os.makedirs(nested)
with open(os.path.join(nested, "agent-1.jsonl"), "w") as f:
    f.write("\n".join(response("sessionB", at(25), 5, ["text"], sidechain=True)) + "\n")
PY
    printf '%s' "$tmp"
}

echo "the meter over two small transcripts"
tmp=$(a_run)
out=$(python3 -I "$METER" /Users/nobody/gdoc-m99 \
    --projects "$tmp/projects" \
    --log "$tmp/ralphex-m99-tasks.log" \
    --log "$tmp/ralphex-m99-review.log" 2>&1)

# Six responses of the seven lines, because one response was two content
# blocks. Three files, because the subagent's transcript is one of them.
holds "the header" "$out" "3 files, 2 sessions, 6 responses, 1 repeats dropped"

# Task iteration 1 holds the scale 1 and scale 2 responses: 30 input, 300
# output, 3,000 cache read, 30,000 cache creation.
holds "task iteration 1" "$out" "^task iteration 1 +1 +2 +30 +300 +3,000 +30,000 +33,330"
holds "task iteration 2" "$out" "^task iteration 2 +1 +1 +30 +300 +3,000 +30,000 +33,330"
holds "the task phase" "$out" "^tasks, all iterations +1 +3 +60 +600 +6,000 +60,000 +66,660"

# The review iteration holds the session and its subagent: scale 4 and 5. The
# subagent's 5 is the half the meter used to drop.
holds "custom review iteration 1" "$out" "^custom review iteration 1 +1 +2 +90 +900 +9,000 +90,000 +99,990"
holds "the review phase" "$out" "^review, all iterations +1 +2 +90 +900 +9,000 +90,000 +99,990"

# The response an hour before the first marker is reported, not dropped and not
# charged to the run.
holds "outside the run" "$out" "^outside the run +1 +1 +60 +600 +6,000 +60,000 +66,660"
holds "the whole worktree" "$out" "^the whole worktree +2 +6 +210 +2,100 +21,000 +210,000 +233,310"
holds "the millions line" "$out" "the whole worktree: 0.23M, 0.02M of it cache reads"

echo "a worktree with no transcripts"
out=$(python3 -I "$METER" /Users/nobody/gdoc-m98 --projects "$tmp/projects" 2>&1)
status=$?
holds "names the folder it looked in" "$out" "no transcripts at .*-Users-nobody-gdoc-m98"
if (( status != 0 )); then pass "refuses with a non-zero status"; else bad "a missing folder exited 0"; fi

rm -rf "$tmp"

if (( fails )); then
    printf '\n%d check(s) failed\n' "$fails"
    exit 1
fi
printf '\nall checks passed\n'
