#!/usr/bin/env bash
# release.sh - a finished branch to a green pull request, and, only on Nail's
# word, the merge, the tag and the zip read back.
#
# Before M28 a Claude Code session did this by hand at the end of every
# milestone: push, open the PR, watch the checks, merge, pull, tag, wait for the
# release workflow, download the arm64 zip and check its sum, its bundle, its
# signature, its plist and its architectures. Eight of those ten steps are the
# same every time, and the session paid for them in context on every turn.
#
#   .ralphex/board/release.sh --branch <branch> [--go vX.Y.0]
#
# or through the Makefile:
#
#   make release BRANCH=<branch> [GO=vX.Y.0] [DRY=1]
#
# The script stops at the green PR. Without --go it never merges, tags or
# publishes anything: it writes `ready: PR <n>, checks green` and ends there.
# The lead passes --go only after Nail's message, which is the one step in a
# milestone that is his and not the team's.
#
# DRY=1 prints every command and runs none. Reads still run under DRY=1, the
# way milestone.sh reads main before it refuses: a read costs nothing and the
# refusals are what a dry run is for.

set -euo pipefail

BRANCH=""
GO=""
DRY=""

# The one zip a Mac can check completely: its own. The linux and windows zips
# carry no bundle and no machine here can run their binary, so CI's listing
# step is the only thing that reads them, and it already refuses a darwin zip
# with no app in it.
PLATFORM="darwin-arm64"
BUNDLE="presence/GdocPresence.app"
EXECUTABLE="$BUNDLE/Contents/MacOS/GdocPresence"
ARCHS="x86_64 arm64"

# How long to wait for the tag's workflow run to appear. A tag pushed by a
# person triggers the run within seconds, but `gh run list` can answer before
# GitHub has created it, and an empty answer is not a failure.
RUN_TRIES="${RUN_TRIES:-10}"
RUN_WAIT="${RUN_WAIT:-6}"

say() { printf '%s\n' "$*"; }
fail() { printf 'make release: %s\n' "$*" >&2; exit 1; }

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --branch) BRANCH="${2:-}"; shift 2 ;;
            --go) GO="${2:-}"; shift 2 ;;
            --dry) DRY=1; shift ;;
            *) fail "$1 is not an argument this script takes." ;;
        esac
    done
}

# show prints a command the way a person could paste it. An argument with a
# space in it is printed quoted, because the dry run's output is the contract
# the tests read.
show() {
    local out="" a
    for a in "$@"; do
        case "$a" in
            *[[:space:]]*) out="$out \"$a\"" ;;
            *) out="$out $a" ;;
        esac
    done
    say "+${out}"
}

step() {
    show "$@"
    [[ -n "$DRY" ]] && return 0
    "$@"
}

# ask runs a read and prints its answer, under DRY=1 as well. gh missing or a
# branch with no PR gives the empty answer, which every caller treats as "not
# known yet" rather than as a failure.
ask() {
    [[ -n "$(command -v "$1" || true)" ]] || return 0
    "$@" 2>/dev/null || true
}

# shell_step prints a composite command and runs it. A checksum and a JSON
# field are each a pipe, and a pipe is not an argv.
shell_step() {
    say "+ $1"
    [[ -n "$DRY" ]] && return 0
    bash -c "$1"
}

# verify_step is one of the six answers the release must give. The command is
# printed, run unless this is a dry run, and its output held against what the
# release says. The step that disagrees names itself in the status file, which
# is the whole report the lead reads.
verify_step() {
    local what=$1 want=$2 cmd=$3 got
    say "+ $cmd"
    [[ -n "$DRY" ]] && return 0
    got=$(bash -c "$cmd" 2>&1 || true)
    case "$got" in
        *"$want"*) say "  ok: $what" ;;
        *) stop "released $GO, NOT verified" "$what: expected \"$want\", got \"$got\"" ;;
    esac
}

# The board folder is the one milestone.sh wrote the run's status into, so the
# lead reads one file from the worktree to the release. A branch that carries no
# milestone number writes no status, and the lines on stdout are all there is.
derive_names() {
    # The letter belongs to the milestone, the way milestone.sh and meter.py
    # read it: m24a and m24b were two runs with two board folders.
    MILESTONE=$(printf '%s' "$BRANCH" | grep -oE 'm[0-9]+[a-z]?' | head -1 || true)
    BOARD=""
    [[ -z "$MILESTONE" ]] || BOARD="$HOME/gdoc-$MILESTONE-board"
}

# write_status is the whole interface to the lead: the phase, the last commit on
# the branch, and the one next action. The shape is milestone.sh's, because the
# lead reads the same path through both halves of a run.
write_status() {
    local phase=$1 next=$2 commit
    if [[ -n "$DRY" ]]; then
        say "+ status: phase: $phase | next: $next"
        return 0
    fi
    [[ -n "$BOARD" ]] || return 0
    mkdir -p "$BOARD"
    commit=$(git log -1 --oneline "$BRANCH" 2>/dev/null || true)
    [[ -n "$commit" ]] || commit="(no commit yet)"
    printf 'phase: %s\ncommit: %s\nnext: %s\n' "$phase" "$commit" "$next" > "$BOARD/status"
}

# stop writes the failing step into the status file and ends the run. Nothing is
# undone: a release that failed its fifth check is published and wrong, and
# hiding that is worse than saying it.
stop() {
    local phase=$1 next=$2
    write_status "$phase" "$next"
    printf 'make release: %s\n' "$next" >&2
    exit 1
}

# The plan is the PR's summary. It sits under docs/plans/ during the run and
# under docs/plans/completed/ once the run's last task moved it, so both are
# looked at.
plan_file() {
    local f
    for f in "$REPO/docs/plans/"*-"$BRANCH".md "$REPO/docs/plans/completed/"*-"$BRANCH".md; do
        [[ -f "$f" ]] || continue
        printf '%s' "$f"
        return 0
    done
    return 0
}

# The PR title is the plan's heading without the word Plan, and the body is the
# plan's Overview. What the milestone did is written down once, in the plan, and
# a hand-written PR summary is the second copy that goes stale.
pr_title() {
    local plan=$1
    [[ -n "$plan" ]] || { printf '%s' "$BRANCH"; return 0; }
    sed -n -E '1,20s/^# (.*)$/\1/p' "$plan" | head -1 | sed -E 's/\.? ?Plan$//'
}

pr_body() {
    local plan=$1 out=$2
    {
        if [[ -n "$plan" ]]; then
            sed -n '/^## Overview/,/^## /p' "$plan" | sed '$d'
        else
            printf 'The branch %s.\n' "$BRANCH"
        fi
        printf '\n## Test plan\n\n'
        printf -- '- [ ] `make vet`\n'
        printf -- '- [ ] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`\n'
        printf -- '- [ ] `make platform-test`\n'
        printf '\n🤖 Generated with [Claude Code](https://claude.com/claude-code)\n'
    } > "$out"
}

pr_number() {
    ask gh pr view "$BRANCH" --json number --jq .number
}

# checks_red names every check that is not a pass. gh's buckets are pass, fail,
# pending, skipping and cancel; a pending check is not green either, and the
# merge waits for it rather than racing it.
checks_red() {
    ask gh pr checks "$BRANCH" --json name,bucket \
        --jq '.[] | select(.bucket != "pass" and .bucket != "skipping") | .name + " is " + .bucket'
}

# checks_named is every check the PR has, pass or not. An empty answer is the
# one green claim this script will not make: a read that failed and a PR with no
# checks at all both look like "nothing is red", and neither is green.
checks_named() {
    ask gh pr checks "$BRANCH" --json name --jq '.[].name'
}

# green_or_stop is the one gate. The caller says what to do when it is not
# green, because before the merge nothing is published and after it everything
# is.
green_or_stop() {
    local phase=$1 named red
    named=$(checks_named)
    [[ -n "$named" ]] || stop "$phase" \
        "the PR's checks could not be read, or the PR has none. Nothing is merged. Run: gh pr checks $BRANCH"
    red=$(checks_red)
    [[ -z "$red" ]] || stop "$phase" "the PR's checks are not green: $red. Nothing is merged."
}

open_pr() {
    local plan title body n
    plan=$(plan_file)
    title=$(pr_title "$plan")
    body=$(mktemp)
    pr_body "$plan" "$body"

    step git push -u origin "$BRANCH" \
        || stop "release, the push failed" "git push refused $BRANCH. Nothing is open."
    n=$(pr_number)
    if [[ -n "$n" ]]; then
        say "+ # PR $n is already open for $BRANCH"
    else
        step gh pr create --base main --head "$BRANCH" \
            --title "$title" --body-file "$body" \
            || stop "release, the PR was not opened" "gh pr create failed for $BRANCH."
        n=$(pr_number)
    fi

    write_status "release, waiting for the checks" "wait: the PR's checks are running"
    # The watch exits non-zero on the first red check, and on a PR that reports
    # none at all. Either way the answer is the read below, which names the
    # check: a failed watch must not kill the script before the status file
    # stops saying the checks are still running.
    step gh pr checks "$BRANCH" --watch --fail-fast || true

    green_or_stop "release, the checks are not green"

    [[ -n "$n" ]] || n="?"
    PR="$n"
    say "ready: PR $PR, checks green"
    write_status "ready" \
        "PR $PR, checks green. Nail's word, then: make release BRANCH=$BRANCH GO=vX.Y.0"
}

# merge_and_tag is everything after Nail's go, and it runs only on a PR whose
# checks are green, and only from a checkout that is on main.
merge_and_tag() {
    # `make tag` commits the version bump on HEAD, tags HEAD and pushes HEAD. In
    # a milestone's worktree that is the feature branch: the bump would land
    # there, main would never get it, and the tag would name a commit no branch
    # carries.
    local here
    here=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)
    [[ "$here" == "main" ]] || fail \
        "GO=$GO must be run from a checkout on main; this one is on ${here:-no branch}. Nothing is merged."

    green_or_stop "release, the checks are not green"

    step gh pr merge "$PR" --merge \
        || stop "release, the merge failed" "gh pr merge refused PR $PR. Nothing is tagged."
    step git pull --ff-only origin main \
        || stop "release, merged but not tagged" "PR $PR is merged, and git pull --ff-only did not fast-forward main. Nothing is tagged."
    step make tag "VERSION=$GO" \
        || stop "release, merged but not tagged" "PR $PR is merged, and make tag VERSION=$GO failed. Nothing is published."
}

# wait_for_release waits for the run the tag started. The tag is the run's
# branch, which is how the one run among the nightly's and the pushes is named.
wait_for_release() {
    local run="" try=1
    write_status "release, waiting for the workflow" "wait: the release workflow is building $GO"
    say "+ gh run list --workflow release.yml --branch $GO --limit 1 --json databaseId --jq '.[0].databaseId'"
    if [[ -n "$DRY" ]]; then
        say "+ gh run watch <run> --exit-status"
        return 0
    fi
    while (( try <= RUN_TRIES )); do
        run=$(ask gh run list --workflow release.yml --branch "$GO" --limit 1 \
            --json databaseId --jq '.[0].databaseId')
        [[ -n "$run" ]] && break
        sleep "$RUN_WAIT"
        try=$((try + 1))
    done
    [[ -n "$run" ]] || stop "released $GO, NOT verified" \
        "no release workflow run appeared for $GO after $((RUN_TRIES * RUN_WAIT))s"
    step gh run watch "$run" --exit-status \
        || stop "released $GO, NOT verified" "the release workflow run $run failed. The tag is pushed."
}

# verify reads the release back the way a colleague's Mac meets it: the asset
# GitHub published, its own checksum file, and the bundle unzipped out of it.
# Nothing here trusts the workflow's own green tick, which is the invariant
# every write in this repository is held to.
verify() {
    local ver=${GO#v}
    local asset="gdoc-$GO-$PLATFORM.zip" sums="SHA256SUMS-$GO"
    local work app
    work=$(mktemp -d)
    app="$work/unpacked/$BUNDLE"

    step gh release download "$GO" --dir "$work" --pattern "$asset" --pattern "$sums" \
        || stop "released $GO, NOT verified" "the release's assets could not be downloaded: $asset, $sums."
    verify_step "the checksum" "$asset: OK" \
        "cd $work && grep ' $asset\$' $sums | shasum -a 256 -c -"
    verify_step "the bundle is in the zip" "$EXECUTABLE" \
        "unzip -Z1 $work/$asset | grep '^$EXECUTABLE\$'"
    step unzip -q "$work/$asset" -d "$work/unpacked" \
        || stop "released $GO, NOT verified" "$asset could not be unzipped."
    verify_step "the signature" "valid on disk" \
        "codesign --verify --deep --strict -vv $app"
    verify_step "the plist version" "$ver" \
        "plutil -extract CFBundleShortVersionString raw -o - $app/Contents/Info.plist"
    verify_step "the architectures" "$ARCHS" \
        "lipo -archs $work/unpacked/$EXECUTABLE"
    verify_step "the binary's version" "$GO" \
        "$work/unpacked/gdoc help --json | jq -r .version"

    say "released $GO, verified"
    write_status "released" "$GO, verified. The decision entry is next."
}

main() {
    parse_args "$@"
    REPO=$(git rev-parse --show-toplevel)

    [[ -n "$BRANCH" ]] || fail "BRANCH=<branch> is required."
    git show-ref -q --verify "refs/heads/$BRANCH" 2>/dev/null \
        || fail "$BRANCH is not a branch in this repository."

    # The refusals come before anything is pushed, so a typo costs the message
    # and nothing else. Only a minor: the nightly owns the patch numbers, and
    # two tags on one number are two releases a colleague cannot tell apart.
    if [[ -n "$GO" ]]; then
        printf '%s' "$GO" | grep -Eq '^v[0-9]+\.[0-9]+\.0$' \
            || fail "GO=$GO is not vX.Y.0. Patch versions are the nightly's."
    fi

    derive_names

    PR=$(pr_number)
    if [[ -z "$GO" ]]; then
        open_pr
        return 0
    fi

    [[ -n "$PR" ]] || fail "no pull request is open for $BRANCH. Run make release BRANCH=$BRANCH first."
    merge_and_tag
    wait_for_release
    verify
}

# Sourced with RELEASE_LIB=1, the file defines its functions and runs nothing,
# so a test can drive one of them on its own.
if [[ -n "${RELEASE_LIB:-}" ]]; then
    return 0
fi

main "$@"
