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

# How long to wait for something GitHub makes a moment after it is asked: the
# tag's workflow run, and the PR's check runs. A push triggers both within
# seconds, but `gh` can answer before GitHub has created them, and an empty
# answer is not a failure.
RUN_TRIES="${RUN_TRIES:-10}"
RUN_WAIT="${RUN_WAIT:-6}"

say() { printf '%s\n' "$*"; }
fail() { printf 'make release: %s\n' "$*" >&2; exit 1; }

# The one temp root this run writes into: the PR's body, and under GO the
# published zip and the bundle unpacked out of it, which together are tens of
# megabytes. A run that ended well removes it. A run that stopped keeps it and
# says where, because the zip a verify step disagreed about is the evidence.
# The template is explicit so TMPDIR is honoured, which macOS's bare `mktemp`
# ignores.
# tmproot sets TMPROOT once. It is not called in a command substitution,
# because that runs it in a subshell and the root the subshell made is one
# nothing in this process knows to remove.
TMPROOT=""
tmproot() {
    local root=${TMPDIR:-/tmp}
    [[ -n "$TMPROOT" ]] || TMPROOT=$(mktemp -d "${root%/}/gdoc-release.XXXXXX")
}

on_exit() {
    local code=$?
    [[ -n "$TMPROOT" ]] || return 0
    if (( code == 0 )); then
        rm -rf "$TMPROOT"
    else
        printf 'make release: what this run wrote is kept in %s\n' "$TMPROOT" >&2
    fi
    return 0
}
trap on_exit EXIT

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
            # The range ends at the next `## ` heading, which is dropped. A
            # plan whose Overview is its last section has no next heading, and
            # the range ends at the end of the file instead.
            sed -n '/^## Overview/,/^## /p' "$plan" | sed '$ {/^## /d;}'
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

# pr_state is OPEN, MERGED or CLOSED. `gh pr view` answers about a merged PR the
# same way it answers about an open one, so the number alone does not say
# whether the merge already happened, and a resumed run has to know.
pr_state() {
    ask gh pr view "$BRANCH" --json state --jq .state
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

# wait_for_checks waits for GitHub to create the PR's check runs. `gh pr checks`
# exits non-zero and names nothing on a PR that reports none, and --watch waits
# only on checks that already exist, so a watch started seconds after the push
# can answer before the workflow's runs are there. An empty answer is "not known
# yet" here, the way wait_for_release reads an empty run list. The gate below is
# still the one that decides.
wait_for_checks() {
    local try=1
    show gh pr checks "$BRANCH" --json name
    [[ -n "$DRY" ]] && return 0
    while (( try <= RUN_TRIES )); do
        [[ -z "$(checks_named)" ]] || return 0
        sleep "$RUN_WAIT"
        try=$((try + 1))
    done
    return 0
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
    tmproot
    body="$TMPROOT/pr-body.md"
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
    wait_for_checks
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

    # A run that merged and then stopped before the tag is resumed by the same
    # command, so the merge is skipped rather than attempted again: `gh pr merge`
    # refuses a merged PR, and without this the only way on was by hand, which
    # is the one thing this script exists to stop.
    if [[ "$(pr_state)" == "MERGED" ]]; then
        say "+ # PR $PR is already merged, so the merge and its gate are skipped"
    else
        green_or_stop "release, the checks are not green"
        step gh pr merge "$PR" --merge \
            || stop "release, the merge failed" "gh pr merge refused PR $PR. Nothing is tagged."
    fi
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
    tmproot
    work="$TMPROOT/release"
    mkdir -p "$work"
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
    #
    # The second and third are `make tag`'s own refusals, made here instead of
    # after the merge. `make tag` runs last, so a dirty tree or a version that
    # is already a tag used to stop the run with the PR merged and nothing
    # published. A lead with one uncommitted file is the likely case: they have
    # just been writing the decision entry.
    if [[ -n "$GO" ]]; then
        printf '%s' "$GO" | grep -Eq '^v[0-9]+\.[0-9]+\.0$' \
            || fail "GO=$GO is not vX.Y.0. Patch versions are the nightly's."
        git diff --quiet HEAD 2>/dev/null \
            || fail "the tree has uncommitted changes, and make tag refuses them after the merge. Commit them first. Nothing is merged."
        git rev-parse -q --verify "refs/tags/$GO" >/dev/null 2>&1 \
            && fail "GO=$GO is already a tag. Nothing is merged."
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
