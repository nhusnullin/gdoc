#!/usr/bin/env bash
# test_release.sh - what `make release` does, asserted without pushing
# anything. Every case is a dry run in a temporary repository with a stub `gh`
# on PATH, so the suite costs a second, no tokens and no network.
#
# Run by `make platform-test`.

set -uo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO=$(cd "$HERE/../../.." && pwd)

fails=0
pass() { printf '  ok   %s\n' "$1"; }
bad() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

holds() {
    local what=$1 text=$2 want=$3
    if printf '%s' "$text" | grep -qF -- "$want"; then
        pass "$what: $want"
    else
        bad "$what: expected to find \"$want\""
    fi
}

lacks() {
    local what=$1 text=$2 want=$3
    if printf '%s' "$text" | grep -qF -- "$want"; then
        bad "$what: did not expect \"$want\""
    else
        pass "$what: no $want"
    fi
}

# in_order asserts that the lines appear in the order given, which is what a
# release is: a merge before a tag, and a checksum before the signature of what
# it checksummed.
in_order() {
    local what=$1 text=$2
    shift 2
    local last=0 n pat
    for pat in "$@"; do
        n=$(printf '%s\n' "$text" | grep -nF -- "$pat" | head -1 | cut -d: -f1)
        if [[ -z "$n" ]]; then
            bad "$what: \"$pat\" is missing"
            return
        fi
        if (( n <= last )); then
            bad "$what: \"$pat\" comes before the step it must follow"
            return
        fi
        last=$n
    done
    pass "$what: $# steps, in order"
}

# a_repo makes a temporary repository with the pilot plan on main and the
# milestone's branch, plus a stub `gh` that records its arguments. The stub
# answers no pull request until `gh pr create` runs, which is the state a first
# release meets. `origin` is a bare repository beside it, so the one case that
# runs for real pushes to a folder and reaches no network.
a_repo() {
    local tmp
    tmp=$(mktemp -d)
    mkdir -p "$tmp/repo/docs/plans" "$tmp/repo/.ralphex/board" "$tmp/bin" "$tmp/home"
    cp "$REPO/.ralphex/board/release.sh" "$tmp/repo/.ralphex/board/"
    cat > "$tmp/repo/docs/plans/2026-10-23-gdoc-v2-m29-the-archive.md" <<'PLAN'
# gdoc v2 Milestone 29: the archive. Plan

Tasks: 4
Review: go

## Overview

The two approved document items: the decisions archive and the command's
topic files.

## Context

Nothing here reaches the PR body.
PLAN
    cat > "$tmp/bin/gh" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$tmp/gh-args"
case "\$1\$2" in
    prview)
        [[ -f "$tmp/pr" ]] && printf '84\n'
        ;;
    prcreate)
        : > "$tmp/pr"
        printf 'https://github.com/altery/gdoc/pull/84\n'
        ;;
    prchecks)
        if [[ "\$*" == *--json* && -f "$tmp/red" ]]; then
            printf 'test / go is fail\n'
        fi
        ;;
    runlist)
        printf '9001\n'
        ;;
esac
exit 0
STUB
    chmod +x "$tmp/bin/gh"
    (
        cd "$tmp/repo" || exit 1
        git init -q -b main
        git config user.email t@example.com
        git config user.name test
        git add -A
        git commit -q -m "docs: the pilot plan"
        git branch gdoc-v2-m29-the-archive
        git init -q --bare "$tmp/remote.git"
        git remote add origin "$tmp/remote.git"
    ) >/dev/null 2>&1
    printf '%s' "$tmp"
}

a_run() {
    local tmp=$1
    shift
    (
        cd "$tmp/repo" || exit 1
        PATH="$tmp/bin:$PATH" HOME="$tmp/home" \
            .ralphex/board/release.sh --branch gdoc-v2-m29-the-archive --dry "$@"
    ) 2>&1
}

# --- without GO: the PR, the checks, and nothing after ----------------------

t=$(a_repo)
out=$(a_run "$t")
holds "the push" "$out" "git push -u origin gdoc-v2-m29-the-archive"
holds "the PR" "$out" "gh pr create --base main --head gdoc-v2-m29-the-archive"
holds "the PR's title is the plan's heading" "$out" "gdoc v2 Milestone 29: the archive"
holds "the PR's body is a file" "$out" "--body-file"
holds "the wait for the checks" "$out" "gh pr checks gdoc-v2-m29-the-archive --watch --fail-fast"
# A dry run opened no pull request, so there is no number to name: the live
# case below is where the number reaches the status file.
holds "the one line the lead reads" "$out" "ready: PR ?, checks green"
holds "the next action is Nail's word" "$out" "Nail's word, then: make release"
lacks "nothing is merged" "$out" "gh pr merge"
lacks "nothing is tagged" "$out" "make tag"
lacks "nothing is downloaded" "$out" "gh release download"
rm -rf "$t"

# --- the PR's body is the plan's Overview -----------------------------------

t=$(a_repo)
body=$(
    cd "$t/repo" || exit 1
    RELEASE_LIB=1
    export RELEASE_LIB
    # shellcheck source=/dev/null
    source .ralphex/board/release.sh
    REPO=$(git rev-parse --show-toplevel)
    BRANCH=gdoc-v2-m29-the-archive
    f=$(mktemp)
    pr_body "$(plan_file)" "$f"
    cat "$f"
)
holds "the body carries the Overview" "$body" "the decisions archive"
lacks "the body stops at the next section" "$body" "Nothing here reaches the PR body"
holds "the body carries a test plan" "$body" "make platform-test"
rm -rf "$t"

# --- with GO: merge, pull, tag, wait, and the six answers -------------------

t=$(a_repo)
: > "$t/pr"
out=$(a_run "$t" --go v2.20.0)
in_order "the release" "$out" \
    "gh pr merge 84 --merge" \
    "git pull --ff-only origin main" \
    "make tag VERSION=v2.20.0" \
    "gh run watch" \
    "shasum -a 256 -c -" \
    "unzip -Z1" \
    "codesign --verify --deep --strict -vv" \
    "plutil -extract CFBundleShortVersionString raw" \
    "lipo -archs" \
    "gdoc help --json | jq -r .version"
holds "the asset is the arm64 zip" "$out" "gdoc-v2.20.0-darwin-arm64.zip"
holds "the checksum file is the tag's" "$out" "SHA256SUMS-v2.20.0"
holds "the bundle is looked for by its executable" "$out" \
    "presence/GdocPresence.app/Contents/MacOS/GdocPresence"
holds "the workflow run is found by the tag" "$out" "--branch v2.20.0"
holds "the last line is the release" "$out" "released v2.20.0, verified"
rm -rf "$t"

# --- the refusals -----------------------------------------------------------

t=$(a_repo)
: > "$t/pr"
out=$(a_run "$t" --go v2.19.1)
holds "a patch version is refused by name" "$out" \
    "GO=v2.19.1 is not vX.Y.0. Patch versions are the nightly's."
lacks "a refused version merges nothing" "$out" "gh pr merge"

: > "$t/red"
out=$(a_run "$t" --go v2.20.0)
holds "a red check is refused by name" "$out" "test / go is fail"
holds "a red check merges nothing" "$out" "Nothing is merged."
lacks "a red check tags nothing" "$out" "make tag"

out=$(a_run "$t")
holds "a red check stops the wait too" "$out" "the checks are not green"
rm -rf "$t"

t=$(a_repo)
out=$(
    cd "$t/repo" || exit 1
    PATH="$t/bin:$PATH" HOME="$t/home" .ralphex/board/release.sh --dry 2>&1
)
holds "a missing branch is refused by name" "$out" "BRANCH=<branch> is required"
out=$(
    cd "$t/repo" || exit 1
    PATH="$t/bin:$PATH" HOME="$t/home" \
        .ralphex/board/release.sh --branch gdoc-v2-m99-nothing --dry 2>&1
)
holds "an unknown branch is refused by name" "$out" "is not a branch in this repository"
rm -rf "$t"

# --- the status file --------------------------------------------------------
# The lead reads ~/gdoc-mNN-board/status from the worktree to the release, so
# the release half writes the same three lines.

t=$(a_repo)
: > "$t/pr"
status=$(
    cd "$t/repo" || exit 1
    PATH="$t/bin:$PATH" HOME="$t/home" \
        .ralphex/board/release.sh --branch gdoc-v2-m29-the-archive >/dev/null 2>&1
    cat "$t/home/gdoc-m29-board/status"
)
holds "the phase" "$status" "phase: ready"
holds "the commit" "$status" "docs: the pilot plan"
holds "the next action" "$status" "PR 84, checks green"
rm -rf "$t"

printf '\n'
if [[ $fails -eq 0 ]]; then
    printf 'test_release: all checks pass\n'
else
    printf 'test_release: %s check(s) failed\n' "$fails"
    exit 1
fi
