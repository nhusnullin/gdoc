#!/usr/bin/env bash
# test_milestone.sh - what `make milestone` does, asserted without running a
# model. Every case is a dry run in a temporary repository, or one function
# driven against a recorded log, so the suite costs a second and no tokens.
#
# Run by `make platform-test`.

set -uo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO=$(cd "$HERE/../../.." && pwd)
FIXTURES="$HERE/fixtures"

fails=0
pass() { printf '  ok   %s\n' "$1"; }
bad() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# holds asserts that the text carries the string.
holds() {
    local what=$1 text=$2 want=$3
    if printf '%s' "$text" | grep -qF -- "$want"; then
        pass "$what: $want"
    else
        bad "$what: expected to find \"$want\""
    fi
}

# lacks asserts that the text does not carry the string.
lacks() {
    local what=$1 text=$2 want=$3
    if printf '%s' "$text" | grep -qF -- "$want"; then
        bad "$what: did not expect \"$want\""
    else
        pass "$what: no $want"
    fi
}

# a_dead_pid is a process id nothing holds, which is what the watch_phase cases
# need: the loop returns 0 when `kill -0` on its pid fails, and keeps sleeping
# for POLL seconds while it succeeds. A made-up number is not that: 999999 is
# below `pid_max` on a Linux runner (4194304 on ubuntu-latest), so a live
# process can carry it and the case would sleep until the CI job's own limit
# killed it instead of failing. A process this shell started and reaped is the
# one id the kernel agrees is gone.
a_dead_pid() {
    local pid
    (exit 0) &
    pid=$!
    wait "$pid" 2>/dev/null
    printf '%s' "$pid"
}

# a_repo makes a temporary repository with the pilot plan committed on main,
# which is what milestone.sh refuses to run without.
a_repo() {
    local tmp
    tmp=$(mktemp -d)
    mkdir -p "$tmp/repo/docs/plans" "$tmp/repo/.ralphex/board" "$tmp/home"
    cp "$REPO/.ralphex/board/milestone.sh" "$tmp/repo/.ralphex/board/"
    cp "$REPO/.ralphex/board/refresh_board.py" "$tmp/repo/.ralphex/board/"
    cat > "$tmp/repo/docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md" <<'PLAN'
# gdoc v2 Milestone 29: the archive. Plan

Tasks: 6
Review: go

### Task 1: the archive files
### Task 2: the register
PLAN
    (
        cd "$tmp/repo" || exit 1
        git init -q -b main
        git config user.email t@example.com
        git config user.name test
        git add -A
        git commit -q -m "docs: the pilot plan"
    ) >/dev/null 2>&1
    printf '%s' "$tmp"
}

# --- the task phase and a go-sized review ---------------------------------

t=$(a_repo)
out=$(cd "$t/repo" && HOME="$t/home" .ralphex/board/milestone.sh \
    --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review go --dry 2>&1)
rc=$?
if [[ $rc -ne 0 ]]; then
    bad "REVIEW=go dry run exited $rc"
    printf '%s\n' "$out"
fi
holds "the worktree" "$out" "git worktree add -b gdoc-v2-m29-the-archive ../gdoc-m29 main"
holds "the board folder" "$out" "mkdir -p $t/home/gdoc-m29-board"
holds "the board page" "$out" "refresh_board.py --init"
holds "the session variables" "$out" \
    "env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT -u CLAUDE_CODE_SESSION_ID -u CLAUDE_CODE_CHILD_SESSION"
holds "the live variables" "$out" "-u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE"
holds "the task phase" "$out" "nohup ralphex --tasks-only --branch gdoc-v2-m29-the-archive"
holds "the iteration cap is Tasks plus two" "$out" "--max-iterations=8"
holds "the idle timeout" "$out" "--idle-timeout=20m"
holds "the task log" "$out" "$t/home/gdoc-m29-board/ralphex-m29-tasks.log"
holds "the review phase" "$out" "nohup ralphex --external-only"
# A stalled review is --idle-timeout's job too. Without the flag the review
# phase has no ceiling of its own: watch_phase waits on a silent worker for ever
# and the status file keeps saying the review is running.
holds "the review carries the idle timeout" "$out" \
    "--external-only --branch gdoc-v2-m29-the-archive --idle-timeout=20m"
lacks "a go review keeps the config" "$out" "--max-external-iterations"
lacks "a go review keeps the roster" "$out" "REVMUX_LENSES"
holds "the status file" "$out" "status: phase: done | next: make release BRANCH=gdoc-v2-m29-the-archive"
rm -rf "$t"

# --- a ui-sized review ------------------------------------------------------

t=$(a_repo)
out=$(cd "$t/repo" && HOME="$t/home" .ralphex/board/milestone.sh \
    --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review ui --dry 2>&1)
holds "a ui review caps the loop" "$out" "--max-external-iterations=1"
holds "a ui review caps the patience" "$out" "--review-patience=1"
holds "a ui review names the smaller lens set" "$out" "REVMUX_LENSES=bugs,impl,adversarial"
holds "a ui review still runs the task phase" "$out" "ralphex --tasks-only"
holds "a ui review carries the idle timeout" "$out" \
    "--external-only --branch gdoc-v2-m29-the-archive --idle-timeout=20m"
rm -rf "$t"

# --- the refusals -----------------------------------------------------------

t=$(a_repo)
out=$(cd "$t/repo" && HOME="$t/home" .ralphex/board/milestone.sh \
    --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review both --dry 2>&1)
holds "REVIEW=both is refused by name" "$out" "REVIEW=both is neither go nor ui"
out=$(cd "$t/repo" && HOME="$t/home" .ralphex/board/milestone.sh --review go --dry 2>&1)
holds "a missing plan is refused by name" "$out" "PLAN=docs/plans/<file>.md is required"
rm -rf "$t"

# --- a worktree holding another branch --------------------------------------
# A worktree is kept so a stopped run resumes. One on another branch is another
# run's: M24 ran twice, as m24a and m24b.

t=$(a_repo)
(
    cd "$t/repo" || exit 1
    git worktree add -q -b somebody-elses ../gdoc-m29 main
) >/dev/null 2>&1
out=$(cd "$t/repo" && HOME="$t/home" .ralphex/board/milestone.sh \
    --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review go --dry 2>&1)
holds "a worktree on another branch is refused by name" "$out" \
    "is on somebody-elses, not gdoc-v2-m29-the-archive"
lacks "nothing is launched into it" "$out" "nohup ralphex"
rm -rf "$t"

# --- a run already in that worktree -----------------------------------------
# The worktree on this branch is kept, which is how a stopped run resumes and
# was also how a second run started into a live one. ralphex takes no lock, so
# two workers would commit onto one branch and read each other's answers.

t=$(a_repo)
mkdir -p "$t/home/gdoc-m29-board"
# A pid something holds: this shell's own. The claim refuses while `kill -0` on
# the pid in the file succeeds.
printf '%s\n' "$$" > "$t/home/gdoc-m29-board/run.pid"
out=$(cd "$t/repo" && HOME="$t/home" .ralphex/board/milestone.sh \
    --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review go --dry 2>&1)
holds "a live run is refused by name" "$out" "a milestone is already running"
holds "and the refusal names the process" "$out" "as process $$"
lacks "nothing is launched beside it" "$out" "nohup ralphex"
status=$(cat "$t/home/gdoc-m29-board/status" 2>/dev/null || true)
lacks "the live run's status file is left alone" "$status" "phase: refused"

# A pid whose process is gone is a run that died, and the folder is taken over:
# that is what resuming after a crash needs.
printf '%s\n' "$(a_dead_pid)" > "$t/home/gdoc-m29-board/run.pid"
out=$(cd "$t/repo" && HOME="$t/home" .ralphex/board/milestone.sh \
    --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review go --dry 2>&1)
lacks "a dead run's claim does not refuse" "$out" "already running"
holds "the run is claimed instead" "$out" "claiming the run"
rm -rf "$t"

# --- a lettered milestone ---------------------------------------------------
# M24 was run twice, as m24a and m24b, each with its own worktree and board by
# hand. Dropping the letter would give the second run the first one's.

t=$(a_repo)
(
    cd "$t/repo" || exit 1
    git mv docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md \
        docs/plans/2026-10-23-gdoc-v2-m29b-the-archive-again.md
    git commit -q -m "docs: the second half of the pilot"
) >/dev/null 2>&1
out=$(cd "$t/repo" && HOME="$t/home" .ralphex/board/milestone.sh \
    --plan docs/plans/2026-10-23-gdoc-v2-m29b-the-archive-again.md --review go --dry 2>&1)
holds "the letter stays in the worktree" "$out" "../gdoc-m29b main"
holds "the letter stays in the board folder" "$out" "mkdir -p $t/home/gdoc-m29b-board"
holds "the letter stays in the log" "$out" "ralphex-m29b-tasks.log"
rm -rf "$t"

# --- the OAuth recipe -------------------------------------------------------
# watch_phase reads the log, so the recorded log of a dead sign-in is the test.

t=$(mktemp -d)
mkdir -p "$t/board"
status=$(
    MILESTONE_LIB=1
    export MILESTONE_LIB
    # shellcheck source=/dev/null
    source "$REPO/.ralphex/board/milestone.sh"
    BOARD="$t/board"
    WORKTREE="$t/nowhere"
    rc=0
    watch_phase "$FIXTURES/oauth-expired.log" "$(a_dead_pid)" tasks || rc=$?
    printf 'rc=%s\n' "$rc"
    cat "$BOARD/status"
) 2>&1
holds "the expired sign-in stops the phase" "$status" "rc=2"
holds "the expired sign-in names the fix" "$status" "needs /login"
holds "the phase is named" "$status" "phase: stopped in tasks"

# The log is appended to, never truncated, so the restart after /login meets the
# dead run's own line. Everything before the launch is another run's.
status=$(
    MILESTONE_LIB=1
    export MILESTONE_LIB
    # shellcheck source=/dev/null
    source "$REPO/.ralphex/board/milestone.sh"
    BOARD="$t/board-restart"
    mkdir -p "$BOARD"
    WORKTREE="$t/nowhere"
    log="$t/restart.log"
    cp "$FIXTURES/oauth-expired.log" "$log"
    rc=0
    watch_phase "$log" "$(a_dead_pid)" tasks "$(wc -c < "$log" | tr -d '[:space:]')" || rc=$?
    printf 'rc=%s\n' "$rc"
    cat "$BOARD/status" 2>/dev/null || true
) 2>&1
holds "a restart reads nothing the dead run wrote" "$status" "rc=0"
lacks "a restart does not ask for /login again" "$status" "needs /login"

# A reviewer quoting the words is not a dead sign-in: the M28 review log carries
# them five times. Only ralphex's own line, where the message starts the line,
# stops a run.
status=$(
    MILESTONE_LIB=1
    export MILESTONE_LIB
    # shellcheck source=/dev/null
    source "$REPO/.ralphex/board/milestone.sh"
    BOARD="$t/board"
    WORKTREE="$t/nowhere"
    log="$t/prose.log"
    printf '%s\n' "[21:04:11] --- custom review iteration 1 ---" \
        "[21:04:12] the recipe greps for a \`Failed to authenticate\` line" > "$log"
    rc=0
    watch_phase "$log" "$(a_dead_pid)" review || rc=$?
    printf 'rc=%s\n' "$rc"
) 2>&1
holds "the words inside a sentence are not the line" "$status" "rc=0"
rm -rf "$t"

# --- a worker's prose is not a phase's answer -------------------------------
# The same distinction as the sign-in, for the two markers that decide whether
# a phase finished. ralphex writes `[26-10-23 22:18:41] ALL_TASKS_DONE` and
# `completed in 49m38s (73 files, +3637/-4163 lines)`, each the only thing on
# its line. The M28 run's own task log carries the words ALL_TASKS_DONE six
# times from the task that wrote this script, the first of them 48 minutes
# before the real line. Read as this run's answer, that prose would skip the one
# relaunch and say `done` over tasks nobody ran.
#
# Two of those six are the shapes that matter, and both are copied below from
# `~/gdoc-m28-board/ralphex-m28.log` lines 1470 and 1580: ralphex stamps wrapped
# prose too, so a sentence can begin a stamped line with the bare token on it.
# Matching the line's start is not enough; the marker's whole line is.

t=$(mktemp -d)
answers=$(
    MILESTONE_LIB=1
    export MILESTONE_LIB
    # shellcheck source=/dev/null
    source "$REPO/.ralphex/board/milestone.sh"
    printf '%s\n' "[26-10-23 21:30:13] exits with no \`ALL_TASKS_DONE\` is relaunched once, then" \
        "[26-10-23 21:31:02] the review phase has no ALL_TASKS_DONE of its own" > "$t/prose-tasks.log"
    printf '%s\n' "[26-10-23 23:00:16] ALL_TASKS_DONE and \"completed in ...\" and wrote \"phase:" \
        "[26-10-23 23:22:43] ALL_TASKS_DONE\`. That prose skipped the one relaunch and" \
        > "$t/wrapped-tasks.log"
    printf '%s\n' "[26-10-23 22:18:41] ALL_TASKS_DONE" > "$t/real-tasks.log"
    printf '%s\n' "[21:04:12] the external review completed successfully, it says" > "$t/prose-review.log"
    printf '%s\n' "[26-10-23 23:00:16] completed in 3h7m (51 files, +4537/-1190 lines) is the" \
        > "$t/wrapped-review.log"
    printf '%s\n' "completed in 49m38s (73 files, +1/-2 lines)" > "$t/real-review.log"
    # ralphex drops the parenthetical when the branch diff is empty, and prints
    # `completed in 0s` alone. Both format strings are in the binary, and a
    # finished review read as unfinished never sends the lead to the release.
    printf '%s\n' "completed in 0s" > "$t/bare-review.log"
    printf '%s\n' "[26-10-23 23:00:16] completed in 0s was all the log said, which" \\
        > "$t/wrapped-bare-review.log"
    tasks_done "$t/prose-tasks.log" && printf 'prose-tasks=done\n' || printf 'prose-tasks=left\n'
    tasks_done "$t/wrapped-tasks.log" && printf 'wrapped-tasks=done\n' || printf 'wrapped-tasks=left\n'
    tasks_done "$t/real-tasks.log" && printf 'real-tasks=done\n' || printf 'real-tasks=left\n'
    review_done "$t/prose-review.log" && printf 'prose-review=done\n' || printf 'prose-review=unfinished\n'
    review_done "$t/wrapped-review.log" && printf 'wrapped-review=done\n' || printf 'wrapped-review=unfinished\n'
    review_done "$t/real-review.log" && printf 'real-review=done\n' || printf 'real-review=unfinished\n'
    review_done "$t/bare-review.log" && printf 'bare-review=done\n' || printf 'bare-review=unfinished\n'
    review_done "$t/wrapped-bare-review.log" && printf 'wrapped-bare=done\n' || printf 'wrapped-bare=unfinished\n'
) 2>&1
holds "prose about the marker is not the marker" "$answers" "prose-tasks=left"
holds "prose wrapped onto the marker is not the marker" "$answers" "wrapped-tasks=left"
holds "ralphex's own line is the marker" "$answers" "real-tasks=done"
holds "prose about a finished review is not one" "$answers" "prose-review=unfinished"
holds "prose wrapped onto the last line is not one" "$answers" "wrapped-review=unfinished"
holds "ralphex's own last line is one" "$answers" "real-review=done"
holds "the last line without the diff count is one" "$answers" "bare-review=done"
holds "prose wrapped onto that shorter line is not one" "$answers" "wrapped-bare=unfinished"
rm -rf "$t"

# The quoted words reach the log at the start of a stamped line the same way,
# and the colon is what tells ralphex's own sign-in line from a sentence about
# it. Both shapes below are from the M28 log.
t=$(mktemp -d)
mkdir -p "$t/board"
status=$(
    MILESTONE_LIB=1
    export MILESTONE_LIB
    # shellcheck source=/dev/null
    source "$REPO/.ralphex/board/milestone.sh"
    BOARD="$t/board"
    WORKTREE="$t/nowhere"
    log="$t/wrapped-auth.log"
    printf '%s\n' "[26-10-23 22:25:54] \`Failed to authenticate\` in the log stops the run, and" \
        "[26-10-23 22:42:35] Failed to authenticate\` five times from the task that wrote it" > "$log"
    rc=0
    watch_phase "$log" "$(a_dead_pid)" tasks || rc=$?
    printf 'rc=%s\n' "$rc"
) 2>&1
holds "prose wrapped onto the sign-in words is not the line" "$status" "rc=0"
rm -rf "$t"

# --- a refusal reaches the lead ---------------------------------------------
# The documented launch line is detached, with stdout and stderr discarded, so
# a refusal printed on stderr is a run the lead never hears about. Where the
# board folder is known the refusal is written where the lead looks.

t=$(a_repo)
(
    cd "$t/repo" || exit 1
    HOME="$t/home" .ralphex/board/milestone.sh \
        --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review both
) >/dev/null 2>&1
status=$(cat "$t/home/gdoc-m29-board/status" 2>/dev/null)
holds "a refusal says so in the status file" "$status" "phase: refused"
holds "and names itself there" "$status" "REVIEW=both is neither go nor ui"
lacks "and sends the lead nowhere else" "$status" "wait:"
rm -rf "$t"

# --- the plan read from another directory -----------------------------------
# Everything after the first refusal reads the plan, the worktree and main as
# paths from the repository root. From any other directory the old script died
# with a bare `sed:` error instead of one of its own refusals.

t=$(a_repo)
out=$(cd "$t/repo/docs" && HOME="$t/home" ../.ralphex/board/milestone.sh \
    --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review go --dry 2>&1)
holds "the worktree is still beside the repository" "$out" "../gdoc-m29 main"
holds "the plan's own task count is still read" "$out" "--max-iterations=8"
lacks "and nothing dies in sed" "$out" "No such file or directory"
rm -rf "$t"

# --- a resumed run keeps the board's clock ----------------------------------
# A kept worktree is a resumed run, and `--init` resets the page's start time to
# now. The time the first launch wrote is handed back, so the page says when the
# run began rather than when it was picked up.

t=$(a_repo)
mkdir -p "$t/home/gdoc-m29-board"
printf '{\n  "started": "09:07"\n}\n' > "$t/home/gdoc-m29-board/m29-run-board.json"
out=$(cd "$t/repo" && HOME="$t/home" .ralphex/board/milestone.sh \
    --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review go --dry 2>&1)
holds "the started time is handed back" "$out" "--started 09:07"
rm -rf "$t"

# --- the lens set reaches revmux --------------------------------------------
# A stub revmux records its arguments, so the test asserts what the review
# script passes without running a review.

stub_round() {
    local lenses=${1:-}
    local tmp
    tmp=$(mktemp -d)
    mkdir -p "$tmp/bin" "$tmp/repo"
    cat > "$tmp/bin/revmux" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$tmp/args"
if [[ "\$1" == "new" ]]; then
    : > "$tmp/scope.md"
    printf '{"scope":"%s"}\n' "$tmp/scope.md"
    exit 0
fi
printf '{"findings":[]}\n'
exit 0
STUB
    chmod +x "$tmp/bin/revmux"
    printf 'a prompt\n' > "$tmp/prompt.txt"
    (
        cd "$tmp/repo" || exit 1
        git init -q -b main >/dev/null 2>&1
        git config user.email t@example.com
        git config user.name test
        git commit -q --allow-empty -m init >/dev/null 2>&1
        git checkout -q -b gdoc-v2-m29-the-archive
        if [[ -n "$lenses" ]]; then export REVMUX_LENSES="$lenses"; fi
        PATH="$tmp/bin:$PATH" "$REPO/.ralphex/scripts/revmux-review.sh" "$tmp/prompt.txt"
    ) >/dev/null 2>&1
    cat "$tmp/args" 2>/dev/null
    rm -rf "$tmp"
}

args=$(stub_round "bugs,impl,adversarial")
holds "REVMUX_LENSES becomes --lenses" "$args" "--lenses bugs,impl,adversarial"
args=$(stub_round)
lacks "no variable, no --lenses" "$args" "--lenses"
holds "the round still runs" "$args" "--no-tui"

# --- the one relaunch -------------------------------------------------------
# A stub ralphex stands in for the real one: the first task phase ends with
# tasks left, which is the shape of a run the Claude CLI dropped, and the
# second either finishes or does not.

a_run() {
    local done_on_second=$1 review_finishes=${2:-yes} tmp
    tmp=$(a_repo)
    cp "$REPO/.ralphex/board/board.html" "$tmp/repo/.ralphex/board/"
    mkdir -p "$tmp/bin"
    cat > "$tmp/bin/ralphex" <<STUB
#!/usr/bin/env bash
phase=tasks
for a in "\$@"; do [[ "\$a" == "--external-only" ]] && phase=review; done
n=1
if [[ -f "$tmp/calls-\$phase" ]]; then n=\$(( \$(cat "$tmp/calls-\$phase") + 1 )); fi
printf '%s' "\$n" > "$tmp/calls-\$phase"
if [[ "\$phase" == review ]]; then
    printf -- '--- custom review iteration 1 ---\n'
    # ralphex's own last line. A review that was killed never writes it.
    [[ "$review_finishes" == yes ]] && printf 'completed in 1m2s (3 files, +4/-5 lines)\n'
    exit 0
fi
printf -- '--- task iteration %s/8 ---\n' "\$n"
if [[ "$done_on_second" == yes ]] && (( n >= 2 )); then
    printf '<<<RALPHEX:ALL_TASKS_DONE>>>\n'
fi
STUB
    chmod +x "$tmp/bin/ralphex"
    (
        cd "$tmp/repo" || exit 1
        PATH="$tmp/bin:$PATH" HOME="$tmp/home" POLL=0 BOARD_EVERY=99999 \
            .ralphex/board/milestone.sh \
            --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review go
    ) >/dev/null 2>&1
    printf 'calls=%s\n' "$(cat "$tmp/calls-tasks" 2>/dev/null)"
    cat "$tmp/home/gdoc-m29-board/status" 2>/dev/null
    rm -rf "$tmp"
}

run=$(a_run yes)
holds "a dropped task phase is relaunched once" "$run" "calls=2"
holds "the relaunch that finishes reaches the review" "$run" "phase: done"
holds "the lead's next action is the release" "$run" "next: make release BRANCH=gdoc-v2-m29-the-archive"

run=$(a_run no)
holds "the second attempt is the last" "$run" "calls=2"
holds "tasks left after the relaunch stop the run" "$run" "tasks are left after one relaunch"

# --- a review that ended before it finished ---------------------------------
# The task phase has ALL_TASKS_DONE. A review that was killed, timed out or
# crashed writes no completion line, and the run must not say `done` over it.

run=$(a_run yes no)
holds "an unfinished review stops the run" "$run" "phase: stopped in review (go)"
holds "it names the log to read" "$run" "the review ended before it finished"
lacks "it does not send the lead to the release" "$run" "make release"

# --- a second run over the first one's log -----------------------------------
# The logs are appended to, never truncated, so a run on a branch that ran
# before meets that run's ALL_TASKS_DONE and its `completed in ...` line.
# Reading either as this run's would say `done` over tasks nobody ran.

a_resume() {
    local tmp
    tmp=$(a_repo)
    cp "$REPO/.ralphex/board/board.html" "$tmp/repo/.ralphex/board/"
    mkdir -p "$tmp/bin" "$tmp/home/gdoc-m29-board"
    printf -- '--- task iteration 1/8 ---\n<<<RALPHEX:ALL_TASKS_DONE>>>\n' \
        > "$tmp/home/gdoc-m29-board/ralphex-m29-tasks.log"
    printf 'completed in 1m2s (3 files, +4/-5 lines)\n' \
        > "$tmp/home/gdoc-m29-board/ralphex-m29-review.log"
    # A ralphex that ends with tasks left and writes no marker of its own.
    cat > "$tmp/bin/ralphex" <<'STUB'
#!/usr/bin/env bash
printf -- '--- task iteration 1/8 ---\n'
STUB
    chmod +x "$tmp/bin/ralphex"
    (
        cd "$tmp/repo" || exit 1
        PATH="$tmp/bin:$PATH" HOME="$tmp/home" POLL=0 BOARD_EVERY=99999 \
            .ralphex/board/milestone.sh \
            --plan docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md --review go
    ) >/dev/null 2>&1
    cat "$tmp/home/gdoc-m29-board/status" 2>/dev/null
    rm -rf "$tmp"
}

run=$(a_resume)
holds "an earlier run's marker is not this run's" "$run" "tasks are left after one relaunch"
lacks "and the lead is not sent to the release" "$run" "make release"

printf '\n'
if [[ $fails -eq 0 ]]; then
    printf 'test_milestone: all checks pass\n'
else
    printf 'test_milestone: %s check(s) failed\n' "$fails"
    exit 1
fi
