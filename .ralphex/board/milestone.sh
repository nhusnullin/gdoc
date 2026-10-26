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
#   .ralphex/board/milestone.sh --plan docs/plans/<file>.md --review go|ui [--codex]
#
# or through the Makefile:
#
#   make milestone PLAN=docs/plans/<file>.md REVIEW=go [CODEX=1] [DRY=1]
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
# CODEX runs the whole pipeline on the codex CLI and the OpenAI plan, the way
# the M29 pilot is measured. Empty runs it on claude.
CODEX=""

# How often the loop looks at the log, and how often it refreshes the board.
# Cut 7 of the retrospective: the board every 30 minutes or on a phase change,
# never per iteration. Each refresh is free in tokens but the old recipe carried
# a session's context through it.
POLL="${POLL:-30}"
BOARD_EVERY="${BOARD_EVERY:-1800}"

# The revmux profile every review runs, go and ui alike: one bugs agent and
# the codex peer. Nail, 2026-10-24, after M28's meter: its review cost 106M of
# the run's 130M. He revisits when a review needs comprehensive in a month.
REVIEW_PROFILE="${REVIEW_PROFILE:-focused}"

# ralphex follows a round with findings by its own claude review loop, capped
# at max(3, max_iterations / 10). At the default of 50 that is five passes, and
# on M28 those five were 100M tokens. Ten holds the loop at its floor of three.
REVIEW_MAX_ITERATIONS=10

# The variables ralphex must not inherit. The four CLAUDE_* ones make the Claude
# CLI refuse to start inside another session. The two GDOC_LIVE ones were set in
# the shell that ran M18 to M27, so a worker's plain `make test` wrote documents
# into Drive; a run states its own live variables in the plan or does without.
STRIP=(CLAUDECODE CLAUDE_CODE_ENTRYPOINT CLAUDE_CODE_SESSION_ID CLAUDE_CODE_CHILD_SESSION GDOC_LIVE_TEST GDOC_LIVE_WRITE)

say() { printf '%s\n' "$*"; }

# fail refuses by name, on stderr and in the status file. The documented launch
# line is detached with stdout and stderr discarded, so stderr alone is a run
# the lead never hears about: they come back to no board, no status and no
# message. Everything after the first two refusals knows BOARD, so that is
# where the refusal is written.
fail() {
    printf 'make milestone: %s\n' "$*" >&2
    if [[ -n "${BOARD:-}" && -z "$DRY" ]]; then
        { mkdir -p "$BOARD" \
            && printf 'phase: refused\ncommit: (nothing ran)\nnext: %s\n' "$*" > "$BOARD/status"; } || true
    fi
    exit 1
}

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
            --codex) CODEX=1; shift ;;
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
    # The letter belongs to the milestone: M24 was run twice, as m24a and m24b,
    # and dropping the letter would give the second run the first one's worktree
    # and the first one's logs. meter.py reads the same shape.
    MILESTONE=$(printf '%s' "$slug" | grep -oE 'm[0-9]+[a-z]?' | head -1 || true)
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

# refresh_html redraws the board page. The pid is what tells a running task from
# a stopped one, so a refresh without it reads as "Stopped" all run long. The
# board is a page a person opens, so a refresh that fails is never a failed run.
refresh_html() {
    local pid=${1:-0}
    [[ -f "$HTML" ]] || return 0
    python3 "$REPO/.ralphex/board/refresh_board.py" --html "$HTML" \
        --repo "$WORKTREE" --pid "$pid" >/dev/null 2>&1 || true
}

refresh_board() {
    local phase=$1 pid=${2:-0}
    refresh_html "$pid"
    write_status "$phase" "wait: the board and this file refresh every $((BOARD_EVERY / 60)) minutes"
}

# claim_the_run is the one run a worktree may hold at a time.
#
# A worktree on another branch is refused above, but one on this branch is kept
# so a stopped run resumes, and until this refusal that was also how a second
# run started into a live one: the lead launches the same line twice, or the
# operator relaunches a phase the board already says is running. ralphex takes
# no lock of its own, so two workers would then commit onto one branch, append
# to one log, and read each other's ALL_TASKS_DONE as their own answer.
#
# The claim is a file of pids beside the board. A pid whose process is gone is
# a run that died, and this run takes the folder over, which is what resuming
# needs. `flock` is not used, because it is not on a Mac and the lead's machine
# is one.
#
# Honest about its limit: the kernel reuses process ids, so a file naming a pid
# that something unrelated now carries refuses a run that could have started.
# That errs towards the lead reading one line and deciding, which is the safe
# way round.
# The file holds this script's pid on the first line and the ralphex it launched
# on the second, because the worker is the half that outlives the script: it is
# launched with nohup, so killing `make milestone` leaves it running. A claim
# that named only the script would then read as a dead run, and the next launch
# would start a second worker beside the first.
claim_the_run() {
    local pid
    if [[ -f "$BOARD/run.pid" ]]; then
        while read -r pid; do
            pid=$(printf '%s' "$pid" | tr -dc '0-9')
            [[ -n "$pid" && "$pid" != "$$" ]] || continue
            kill -0 "$pid" 2>/dev/null || continue
            # Not `fail`: the status file is the live run's, and three lines
            # saying `refused` over a run that is running is worse than no
            # lines at all.
            printf 'make milestone: %s\n' \
                "a milestone is already running in $WORKTREE as process $pid. Read $BOARD/status, or stop that run first." >&2
            exit 1
        done < "$BOARD/run.pid"
    fi
    say "+ # claiming the run: $$ into $BOARD/run.pid"
    [[ -n "$DRY" ]] && return 0
    printf '%s\n' "$$" > "$BOARD/run.pid"
    # The claim dies with the process that made it. A run killed before the trap
    # leaves the file, and the `kill -0` above reads it as the dead run it is.
    trap 'release_the_run' EXIT
}

# claim_the_worker rewrites the claim with the worker this phase launched, so
# the pid that survives the script is in the file the next launch reads. The
# phase before it left a dead pid; only the live one is worth keeping.
claim_the_worker() {
    [[ -z "$DRY" ]] || return 0
    [[ "${LAUNCHED_PID:-0}" -gt 0 ]] || return 0
    printf '%s\n%s\n' "$$" "$LAUNCHED_PID" > "$BOARD/run.pid"
}

# release_the_run drops the claim this process made, and takes the worker with
# it. A phase that ended leaves no worker to kill. The exits that do leave one
# are the dead sign-in, where no amount of running fixes the worker, and the
# lead killing the script, where stopping the run is what they meant; a worker
# that outlived either would hold the worktree against the relaunch the status
# file asks for. Honest about its limit: the kill reaches ralphex and not a
# claude it has already started, so a worker killed mid-iteration may still
# write that iteration's commit.
release_the_run() {
    local first="" pid=""
    [[ -f "$BOARD/run.pid" ]] || return 0
    first=$(head -1 "$BOARD/run.pid" | tr -dc '0-9')
    [[ "$first" == "$$" ]] || return 0
    pid=$(sed -n 2p "$BOARD/run.pid" | tr -dc '0-9')
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null || true
    fi
    rm -f "$BOARD/run.pid"
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

# launch prints the detached ralphex line and returns its pid in LAUNCHED_PID
# and the log's size before the launch in LAUNCHED_AT. The command is built as
# one string so that what the dry run prints is exactly what a real run
# executes.
#
# A launch appends to the phase log, so the log of a run that died is still
# there when the lead starts the next one. LAUNCHED_AT is where this run's
# output begins, and watch_phase reads nothing before it.
launch() {
    local log=$1; shift
    local prefix=$1; shift
    local cmd
    [[ -z "$prefix" ]] || prefix="$prefix "
    cmd="( cd \"$WORKTREE\" && ${prefix}$(env_strip) nohup ralphex $* >> \"$log\" 2>&1 )"
    say "+ $cmd &"
    LAUNCHED_PID=0
    LAUNCHED_AT=0
    [[ ! -f "$log" ]] || LAUNCHED_AT=$(wc -c < "$log" | tr -d '[:space:]')
    if [[ -z "$DRY" ]]; then
        eval "$cmd &"
        LAUNCHED_PID=$!
        # The claim carries the worker from here on, so a script that dies
        # leaves a file that names what is still running.
        claim_the_worker
    fi
}

# ralphex's own line, "[21:18:06] Failed to authenticate: OAuth session
# expired", with the message where the line starts and the colon after it. A
# worker's prose reaches the same log, and the words quoted inside a sentence
# are not a dead sign-in: `~/gdoc-m28-board/ralphex-m28.log` carries them on six
# lines and none of them is ralphex's. The count is not the evidence, because
# every sentence a worker writes about the words lands in the same log and moves
# it; the shape is. The line start alone does not tell the two apart,
# because ralphex stamps every line it passes through, wrapped prose included,
# so a sentence that wraps onto the words begins a stamped line too. The colon
# is ralphex's own form and no quoting of the words carries it.
AUTH_FAILED='^(\[[0-9 :-]+\] )?Failed to authenticate:'

# watch_phase waits for one ralphex to end, refreshing the board on the way.
# Returns 0 when the process ended, 2 when the Claude CLI's sign-in expired.
# `from` is the log's size before this launch, so nothing an earlier run wrote
# is read as this one's.
#
# Only the two recipes ralphex does not already cover live here. A stalled
# worker is --idle-timeout's job, and a rate limit is ralphex's own wait.
watch_phase() {
    local log=$1 pid=$2 phase=$3 from=${4:-0}
    local since=0
    while :; do
        # "Failed to authenticate: OAuth session expired" kills the run a line
        # later, and no amount of waiting fixes it: only /login inside an
        # interactive claude does.
        if [[ -f "$log" ]] && grep -Eq "$AUTH_FAILED" <(tail -c "+$((from + 1))" "$log"); then
            write_status "stopped in $phase" \
                "needs /login: run /login inside an interactive claude, then start make milestone again"
            return 2
        fi
        kill -0 "$pid" 2>/dev/null || return 0
        sleep "$POLL"
        since=$((since + POLL))
        if (( since >= BOARD_EVERY )); then
            since=0
            refresh_board "$phase" "$pid"
        fi
    done
}

# ralphex's own marker, "[26-10-23 22:18:41] ALL_TASKS_DONE", and its own last
# line, "completed in 49m38s (73 files, +3637/-4163 lines)". The same
# distinction AUTH_FAILED draws, for the two answers that decide whether a phase
# finished: `~/gdoc-m28-board/ralphex-m28.log` carries the words ALL_TASKS_DONE
# on ten lines beside the one line that is the marker, all from the task that
# wrote this script, the first of them 48 minutes before the marker. That count
# grew while the task wrote, so the shape and not the number is the evidence.
#
# The whole line is matched, not its start, because ralphex stamps wrapped prose
# too: two of those sentences, on lines 1470 and 1580, wrap onto a line that
# begins with the bare token, and a start-anchored pattern reads both as the
# marker. The marker is the only thing on its line in all eleven runs from M20
# to M28. The
# `<<<RALPHEX:...>>>` form is the signal a worker prints, which reaches the log
# on a line of its own.
#
# The diff count is optional, because ralphex carries both format strings and
# drops the parenthetical when the branch diff against main is empty: the line
# is then `completed in 0s` alone. Requiring "lines)" read that finished review
# as a branch nobody reviewed and never sent the lead to the release. The
# duration must start with a digit and the line must end there or after the
# count, which is what keeps wrapped prose out.
TASKS_DONE='^(\[[0-9 :-]+\] )?(<<<RALPHEX:)?ALL_TASKS_DONE(>>>)?[[:space:]]*$'
REVIEW_DONE='^(\[[0-9 :-]+\] )?completed in [0-9][0-9a-z.]*( \([0-9]+ files, [+][0-9]+/-[0-9]+ lines\))?[[:space:]]*$'

# tasks_done and review_done read from `from`, the log's size before this
# launch, for the same reason watch_phase does: the log is appended to, so the
# marker an earlier run wrote is still in it, and reading it as this run's would
# say `done` over tasks nobody ran.
tasks_done() {
    local log=$1 from=${2:-0}
    [[ -f "$log" ]] && grep -Eq "$TASKS_DONE" <(tail -c "+$((from + 1))" "$log")
}

# review_done is ralphex's own last line, which refresh_board.py reads as
# finished too. A review that was killed, timed out or crashed never writes it,
# and a branch whose log lacks it is not a reviewed branch.
review_done() {
    local log=$1 from=${2:-0}
    [[ -f "$log" ]] && grep -Eq "$REVIEW_DONE" <(tail -c "+$((from + 1))" "$log")
}

run_task_phase() {
    local attempt=1
    while :; do
        launch "$TASK_LOG" "" --tasks-only ${CODEX:+--codex} --branch "$BRANCH" \
            "--max-iterations=$((TASKS + 2))" "--idle-timeout=$IDLE" "$PLAN"
        write_status "tasks, attempt $attempt of 2" "wait: the tasks are running"
        [[ -n "$DRY" ]] && return 0
        refresh_html "$LAUNCHED_PID"
        local from=$LAUNCHED_AT
        watch_phase "$TASK_LOG" "$LAUNCHED_PID" "tasks" "$from" || exit 1
        tasks_done "$TASK_LOG" "$from" && return 0
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
    # --idle-timeout for the same reason the task phase carries it: the fix
    # rounds between the external rounds are claude sessions, and watch_phase
    # has no ceiling of its own. Without it a silent worker leaves the status
    # file saying the review is running until somebody looks at the log.
    # One round, one profile, for both sizes. REVIEW still names the size,
    # because a ui run is followed by the design reviewer and a go run is not.
    local prefix="REVMUX_PROFILE=$REVIEW_PROFILE"
    local args=(--external-only --branch "$BRANCH" "--idle-timeout=$IDLE"
        "--max-iterations=$REVIEW_MAX_ITERATIONS"
        --max-external-iterations=1 --review-patience=1)
    if [[ -n "$CODEX" ]]; then
        # --codex skips the external review, so --external-only would review
        # nothing. --review runs the first pass and the critical/major loop,
        # both on codex, and revmux is not called.
        prefix=""
        args=(--review --codex --branch "$BRANCH" "--idle-timeout=$IDLE"
            "--max-iterations=$REVIEW_MAX_ITERATIONS")
    fi
    launch "$REVIEW_LOG" "$prefix" "${args[@]}"
    write_status "review ($REVIEW)" "wait: the review is running"
    [[ -n "$DRY" ]] && return 0
    refresh_html "$LAUNCHED_PID"
    local from=$LAUNCHED_AT
    watch_phase "$REVIEW_LOG" "$LAUNCHED_PID" "review ($REVIEW)" "$from" || exit 1
    # The task phase has ALL_TASKS_DONE; this is the review's own answer. A
    # review that ended without it is a branch nobody reviewed, and saying
    # `done` over it would send the lead to the release.
    review_done "$REVIEW_LOG" "$from" && return 0
    write_status "stopped in review ($REVIEW)" \
        "the review ended before it finished: read $REVIEW_LOG"
    exit 1
}

main() {
    parse_args "$@"
    REPO=$(git rev-parse --show-toplevel)

    [[ -n "$PLAN" ]] || fail "PLAN=docs/plans/<file>.md is required."
    # Everything after this reads the plan, the worktree and main as paths from
    # the repository root: `docs/plans/<file>.md` and `../gdoc-<mNN>`. One
    # directory down, the plan's fields were read from a path that was not
    # there and the run died in sed rather than in one of its own refusals.
    PLAN=${PLAN#"$REPO/"}
    cd "$REPO"
    [[ -f "$PLAN" ]] || fail "$PLAN is not a file."

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
        # A worktree is kept so a stopped run resumes where it was. One holding
        # another branch is another run's, and launching into it would append to
        # that run's logs and commit onto its branch.
        local here
        here=$(git -C "$WORKTREE" rev-parse --abbrev-ref HEAD 2>/dev/null || true)
        [[ "$here" == "$BRANCH" ]] || fail \
            "$WORKTREE is on ${here:-no branch}, not $BRANCH. Remove it, or run the milestone it belongs to."
        say "+ # $WORKTREE is already a worktree on $BRANCH, keeping it"
    else
        # A branch can outlive its worktree: the lead removed the folder, or a
        # release merged the branch and left it behind. `add -b` refuses a
        # branch that exists, and a folder removed with `rm -rf` leaves its
        # registration, which `add` refuses too. Prune, then pick the form by
        # whether the ref is there, so the second run of one milestone adds the
        # worktree back instead of dying in git.
        #
        # Each of the three carries `|| fail`, because a setup command that
        # exits through `set -e` writes no status file, and the status file is
        # the only thing the lead reads.
        step git worktree prune || fail "git worktree prune failed in $REPO."
        if git show-ref -q --verify "refs/heads/$BRANCH"; then
            step git worktree add "$WORKTREE" "$BRANCH" \
                || fail "$WORKTREE could not be added for the existing branch $BRANCH."
        else
            step git worktree add -b "$BRANCH" "$WORKTREE" main \
                || fail "$WORKTREE could not be added on a new $BRANCH."
        fi
    fi
    step mkdir -p "$BOARD" || fail "$BOARD could not be made."
    claim_the_run
    # --init forgets the page and starts its clock at now, which on a resumed
    # run would say the run began when the lead picked it up. The time the
    # first launch wrote is in the page's own sidecar, and is handed back.
    local init=(python3 "$REPO/.ralphex/board/refresh_board.py" --init
        --plan "$PLAN" --branch "$BRANCH" --heading "$HEADING"
        --html "$HTML" --logs "$TASK_LOG" "$REVIEW_LOG" --repo "$WORKTREE")
    local started=""
    [[ ! -f "${HTML%.html}.json" ]] \
        || started=$(sed -n -E 's/.*"started"[^"]*"([^"]+)".*/\1/p' "${HTML%.html}.json" | head -1)
    [[ -z "$started" ]] || init+=(--started "$started")
    step "${init[@]}" || fail "the board page could not be initialised: $HTML."

    run_task_phase
    run_review_phase

    # The last refresh, with no pid: the run is over and the page says so.
    [[ -n "$DRY" ]] || refresh_html 0
    write_status "done" "make release BRANCH=$BRANCH"
}

# Sourced with MILESTONE_LIB=1, the file defines its functions and runs nothing,
# so a test can drive watch_phase against a recorded log.
if [[ -n "${MILESTONE_LIB:-}" ]]; then
    return 0
fi

main "$@"
