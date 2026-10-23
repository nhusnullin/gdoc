#!/usr/bin/env bash
# milestone.sh - one milestone run, as a service the team lead starts and
# forgets.
#
# Before M28 a Claude Code session did this by hand: make the worktree, strip
# the session variables, launch ralphex detached, arm a monitor on the log,
# refresh the board after every iteration, notice a dead sign-in, start the
# review. That session held the whole run in its context and paid for it on
# every turn. The script owns the run instead, and writes one status file of
# three lines. The lead reads the status file, never the log.
#
#   .ralphex/board/milestone.sh --plan docs/plans/<file>.md --review go|ui
#
# or through the Makefile:
#
#   make milestone PLAN=docs/plans/<file>.md REVIEW=go [DRY=1]
#
# DRY=1 prints every command and runs none, which is what the tests in
# tests/test_milestone.sh assert.

set -euo pipefail

# The plan file names the run. 2026-10-23-gdoc-v2-m29-the-archive.md gives the
# branch gdoc-v2-m29-the-archive, the worktree ../gdoc-m29 and the board folder
# ~/gdoc-m29-board, which is the naming every run since M20 has used by hand.
PLAN=""
REVIEW=""
DRY=""
IDLE="${IDLE:-20m}"
TASKS=""

# How often the loop looks at the log, and how often it refreshes the board.
# Cut 7 of the retrospective: the board every 30 minutes or on a phase change,
# never per iteration. Each refresh is free in tokens but the old recipe carried
# a session's context through it.
POLL="${POLL:-30}"
BOARD_EVERY="${BOARD_EVERY:-1800}"

# The lens set a UI milestone reviews with. Cut 2 of the retrospective: the full
# roster of eight lenses only where go/ changed; for presence/ the passes after
# the third moved commas. These three are revmux's own "final" profile.
UI_LENSES="${UI_LENSES:-bugs,impl,adversarial}"

# The variables ralphex must not inherit. The four CLAUDE_* ones make the Claude
# CLI refuse to start inside another session. The two GDOC_LIVE ones were set in
# the shell that ran M18 to M27, so a worker's plain `make test` wrote documents
# into Drive; a run states its own live variables in the plan or does without.
STRIP=(CLAUDECODE CLAUDE_CODE_ENTRYPOINT CLAUDE_CODE_SESSION_ID CLAUDE_CODE_CHILD_SESSION GDOC_LIVE_TEST GDOC_LIVE_WRITE)

say() { printf '%s\n' "$*"; }
fail() { printf 'make milestone: %s\n' "$*" >&2; exit 1; }

# env_strip prints the `env -u A -u B ...` prefix every launch carries.
env_strip() {
    local out="env"
    local v
    for v in "${STRIP[@]}"; do out="$out -u $v"; done
    printf '%s' "$out"
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --plan) PLAN="${2:-}"; shift 2 ;;
            --review) REVIEW="${2:-}"; shift 2 ;;
            --idle) IDLE="${2:-}"; shift 2 ;;
            --tasks) TASKS="${2:-}"; shift 2 ;;
            --dry) DRY=1; shift ;;
            *) fail "$1 is not an argument this script takes." ;;
        esac
    done
}

# plan_field reads a one-line field the plan states, such as `Tasks: 7` or
# `Review: go`. The plan is the budget: no separate estimate file.
plan_field() {
    local name=$1 file=$2
    sed -n -E "s/^[*_ ]*${name}:[*_ ]*[[:space:]]*([A-Za-z0-9|]+).*$/\1/p" "$file" | head -1
}

# count_tasks falls back to the plan's own task headers when it states no
# `Tasks:` line, so an older plan still runs.
count_tasks() {
    grep -cE '^### (Task|Iteration) [0-9]+' "$1" || true
}

derive_names() {
    local base slug
    base=$(basename "$PLAN" .md)
    # Strip the leading date, which is how every branch since M2 is named.
    slug=$(printf '%s' "$base" | sed -E 's/^[0-9]{4}-[0-9]{2}-[0-9]{2}-//')
    BRANCH="$slug"
    MILESTONE=$(printf '%s' "$slug" | grep -oE 'm[0-9]+' | head -1 || true)
    [[ -n "$MILESTONE" ]] || fail "the plan file name carries no milestone number: $PLAN"
    WORKTREE="../gdoc-$MILESTONE"
    BOARD="$HOME/gdoc-$MILESTONE-board"
    TASK_LOG="$BOARD/ralphex-$MILESTONE-tasks.log"
    REVIEW_LOG="$BOARD/ralphex-$MILESTONE-review.log"
    HTML="$BOARD/$MILESTONE-run-board.html"
    HEADING="Milestone ${MILESTONE#m}: $(printf '%s' "$slug" | sed -E "s/^.*${MILESTONE}-//" | tr '-' ' ')"
}

# write_status is the whole interface to the lead: the phase, the last commit on
# the branch, and the one next action. Three lines, replaced in place.
write_status() {
    local phase=$1 next=$2 commit
    if [[ -n "$DRY" ]]; then
        say "+ status: phase: $phase | next: $next"
        return 0
    fi
    commit=$(git -C "$WORKTREE" log -1 --oneline 2>/dev/null || true)
    [[ -n "$commit" ]] || commit="(no commit yet)"
    printf 'phase: %s\ncommit: %s\nnext: %s\n' "$phase" "$commit" "$next" > "$BOARD/status"
}

refresh_board() {
    local phase=$1
    [[ -f "$HTML" ]] || return 0
    python3 "$REPO/.ralphex/board/refresh_board.py" --html "$HTML" \
        --repo "$WORKTREE" >/dev/null 2>&1 || true
    write_status "$phase" "wait: the board and this file refresh every $((BOARD_EVERY / 60)) minutes"
}

# step prints a command and runs it, or only prints it under DRY=1. An argument
# with a space in it is printed quoted, because the dry run's output is the
# contract the tests read and a person pastes.
step() {
    local line="" a
    for a in "$@"; do
        case "$a" in
            *[[:space:]]*) line="$line \"$a\"" ;;
            *) line="$line $a" ;;
        esac
    done
    say "+${line}"
    [[ -n "$DRY" ]] && return 0
    "$@"
}

# launch prints the detached ralphex line and returns its pid in LAUNCHED_PID.
# The command is built as one string so that what the dry run prints is exactly
# what a real run executes.
launch() {
    local log=$1; shift
    local prefix=$1; shift
    local cmd
    [[ -z "$prefix" ]] || prefix="$prefix "
    cmd="( cd \"$WORKTREE\" && ${prefix}$(env_strip) nohup ralphex $* >> \"$log\" 2>&1 )"
    say "+ $cmd &"
    LAUNCHED_PID=0
    if [[ -z "$DRY" ]]; then
        eval "$cmd &"
        LAUNCHED_PID=$!
    fi
}

# watch_phase waits for one ralphex to end, refreshing the board on the way.
# Returns 0 when the process ended, 2 when the Claude CLI's sign-in expired.
#
# Only the two recipes ralphex does not already cover live here. A stalled
# worker is --idle-timeout's job, and a rate limit is ralphex's own wait.
watch_phase() {
    local log=$1 pid=$2 phase=$3
    local since=0
    while :; do
        # "Failed to authenticate: OAuth session expired" kills the run a line
        # later, and no amount of waiting fixes it: only /login inside an
        # interactive claude does.
        if [[ -f "$log" ]] && grep -q 'Failed to authenticate' "$log"; then
            write_status "stopped in $phase" \
                "needs /login: run /login inside an interactive claude, then start make milestone again"
            return 2
        fi
        kill -0 "$pid" 2>/dev/null || return 0
        sleep "$POLL"
        since=$((since + POLL))
        if (( since >= BOARD_EVERY )); then
            since=0
            refresh_board "$phase"
        fi
    done
}

tasks_done() {
    [[ -f "$1" ]] && grep -q 'ALL_TASKS_DONE' "$1"
}

run_task_phase() {
    local attempt=1
    while :; do
        launch "$TASK_LOG" "" --tasks-only --branch "$BRANCH" \
            "--max-iterations=$((TASKS + 2))" "--idle-timeout=$IDLE" "$PLAN"
        write_status "tasks, attempt $attempt of 2" "wait: the tasks are running"
        [[ -n "$DRY" ]] && return 0
        watch_phase "$TASK_LOG" "$LAUNCHED_PID" "tasks" || exit 1
        tasks_done "$TASK_LOG" && return 0
        if (( attempt == 2 )); then
            write_status "stopped in tasks" \
                "tasks are left after one relaunch: read $TASK_LOG"
            exit 1
        fi
        attempt=2
        say "+ # ralphex exited with tasks left, relaunching once"
    done
}

run_review_phase() {
    local prefix="" args=(--external-only --branch "$BRANCH")
    if [[ "$REVIEW" == "ui" ]]; then
        # Cut 2: one round over a smaller lens set. A panel's review is the
        # reviewer who draws nothing, not the fifth pass over the diff.
        prefix="REVMUX_LENSES=$UI_LENSES"
        args+=(--max-external-iterations=1 --review-patience=1)
    fi
    launch "$REVIEW_LOG" "$prefix" "${args[@]}"
    write_status "review ($REVIEW)" "wait: the review is running"
    [[ -n "$DRY" ]] && return 0
    watch_phase "$REVIEW_LOG" "$LAUNCHED_PID" "review ($REVIEW)" || exit 1
}

main() {
    parse_args "$@"
    REPO=$(git rev-parse --show-toplevel)

    [[ -n "$PLAN" ]] || fail "PLAN=docs/plans/<file>.md is required."
    [[ -f "$REPO/$PLAN" || -f "$PLAN" ]] || fail "$PLAN is not a file."

    derive_names

    [[ -n "$REVIEW" ]] || REVIEW=$(plan_field Review "$PLAN")
    case "$REVIEW" in
        go|ui) ;;
        "") fail "REVIEW=go or REVIEW=ui is required, and the plan states no Review: line." ;;
        *) fail "REVIEW=$REVIEW is neither go nor ui." ;;
    esac

    [[ -n "$TASKS" ]] || TASKS=$(plan_field Tasks "$PLAN")
    [[ -n "$TASKS" ]] || TASKS=$(count_tasks "$PLAN")
    [[ "$TASKS" =~ ^[0-9]+$ && "$TASKS" -gt 0 ]] || fail "the plan states no task count and holds no task headers."

    # The plan must be on main before the run: ralphex branches from main, and
    # the reviewer must not see the plan as new code.
    git cat-file -e "main:$PLAN" 2>/dev/null || fail "$PLAN is not committed on main."

    if [[ -d "$WORKTREE" ]]; then
        say "+ # $WORKTREE is already a worktree, keeping it"
    else
        step git worktree add -b "$BRANCH" "$WORKTREE" main
    fi
    step mkdir -p "$BOARD"
    step python3 "$REPO/.ralphex/board/refresh_board.py" --init \
        --plan "$PLAN" --branch "$BRANCH" --heading "$HEADING" \
        --html "$HTML" --logs "$TASK_LOG" "$REVIEW_LOG" --repo "$WORKTREE"

    run_task_phase
    run_review_phase

    write_status "done" "make release BRANCH=$BRANCH"
}

# Sourced with MILESTONE_LIB=1, the file defines its functions and runs nothing,
# so a test can drive watch_phase against a recorded log.
if [[ -n "${MILESTONE_LIB:-}" ]]; then
    return 0
fi

main "$@"
