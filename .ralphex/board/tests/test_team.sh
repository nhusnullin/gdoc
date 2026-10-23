#!/usr/bin/env bash
# test_team.sh - the team's definitions, checked as files. No model runs here:
# every case reads `.claude/agents/gdoc-*.md` and the repository around it, so
# the suite costs a second and no tokens.
#
# Three things are checked. The frontmatter, because a definition with a typo
# in it is a seat Claude Code does not offer. The model and the tools, because
# the whole point of the seat is that it is pinned: a read-only reviewer that
# can write is not the reviewer the retrospective asked for. And every path a
# definition names, because a brief that sends a seat to a file that moved
# sends it nowhere and the seat invents the answer.
#
# Run by `make team-check`.

set -uo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO=$(cd "$HERE/../../.." && pwd)
AGENTS=$REPO/.claude/agents

fails=0
pass() { printf '  ok   %s\n' "$1"; }
bad() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# front prints the frontmatter of a definition: the lines between the first
# `---` and the next one. A file that does not open with `---` prints nothing,
# which every check below then fails by name.
front() {
    local file=$1
    [[ $(head -1 "$file") == "---" ]] || return 0
    sed -n '2,$p' "$file" | sed -n '/^---$/q;p'
}

# body prints what is after the frontmatter: the brief itself, which is every
# line after the second `---`.
body() {
    awk 'seen >= 2 { print } /^---$/ { seen++ }' "$1"
}

field() {
    front "$1" | sed -n "s/^$2: *//p" | head -1
}

# tools prints the frontmatter's tool list as one space-separated line, in the
# order it is written.
tools() {
    front "$1" | sed -n '/^tools:/,/^[a-zA-Z]/p' | sed -n 's/^  *- *//p' | tr '\n' ' ' | sed 's/ $//'
}

holds() {
    local what=$1 text=$2 want=$3
    if printf '%s' "$text" | grep -qF -- "$want"; then
        pass "$what: $want"
    else
        bad "$what: expected to find \"$want\""
    fi
}

is() {
    local what=$1 got=$2 want=$3
    if [[ "$got" == "$want" ]]; then
        pass "$what: $want"
    else
        bad "$what: expected \"$want\", got \"$got\""
    fi
}

# --- the frontmatter, the model and the tools -------------------------------
# The seat, its model and its tools, as the plan pinned them. A seat whose
# tools widen here is a seat that can do what its brief says it never does.

check_seat() {
    local name=$1 model=$2 want_tools=$3
    local file=$AGENTS/$name.md

    if [[ ! -f $file ]]; then
        bad "$name: .claude/agents/$name.md is missing"
        return
    fi
    is "$name opens with frontmatter" "$(head -1 "$file")" "---"
    is "$name names itself" "$(field "$file" name)" "$name"
    is "$name pins its model" "$(field "$file" model)" "$model"
    is "$name pins its tools" "$(tools "$file")" "$want_tools"

    local description
    description=$(field "$file" description)
    if [[ ${#description} -ge 40 ]]; then
        pass "$name has a description that says when to call it"
    else
        bad "$name: the description is \"$description\""
    fi
    if [[ -n "$(body "$file" | tr -d '[:space:]')" ]]; then
        pass "$name has a brief under its frontmatter"
    else
        bad "$name: the frontmatter is the whole file"
    fi
}

check_seat gdoc-platform-operator sonnet "Bash Read"
check_seat gdoc-design-reviewer sonnet "Read"
check_seat gdoc-trust-guardian opus "Read Grep Glob"

# --- what each seat never does ---------------------------------------------
# The three sentences the retrospective paid for. Each is in the brief, by
# name, because a seat reads its brief and not this test.

operator=$(cat "$AGENTS/gdoc-platform-operator.md" 2>/dev/null)
holds "the operator reads the status file" "$operator" "status"
holds "the operator reads the recipes" "$operator" ".ralphex/board/README.md"
holds "the operator wakes only on a failed recipe" "$operator" "a recipe failed"
holds "the operator reports three lines" "$operator" "three lines"
holds "the operator never pushes, merges or tags" "$operator" \
    "never pushes, merges or tags"

reviewer=$(cat "$AGENTS/gdoc-design-reviewer.md" 2>/dev/null)
holds "the reviewer reads the principles" "$reviewer" \
    "docs/design/2026-10-17-presence-design-principles.md"
holds "the reviewer reads eighteen images" "$reviewer" "18"
holds "the reviewer reads the dark renders at 1x" "$reviewer" "dark"
holds "the light renders follow a finding" "$reviewer" \
    "light appearance only where dark has a finding"
holds "the captions are text, not pixels" "$reviewer" "index.html"
holds "the reviewer rates" "$reviewer" "blocking"
holds "the reviewer never fixes" "$reviewer" "never fixes"

guardian=$(cat "$AGENTS/gdoc-trust-guardian.md" 2>/dev/null)
for area in CLAUDE.md go/internal/guard go/internal/auth go/internal/presence \
    go/cmd/gdoc/desktop.go; do
    holds "the guardian's area" "$guardian" "$area"
done
holds "the guardian signs or vetoes" "$guardian" "veto"
holds "the guardian writes into the register" "$guardian" "docs/v2/DECISIONS.md"
holds "the guardian carries the skeptic" "$guardian" "skeptic"

# --- the two seats that already exist --------------------------------------
# The plan reviewer and Codex are user-level agents. The team names them by
# the name Claude Code offers them under, and defines neither again: a second
# definition of a seat is two briefs for one job.

named=$(cat "$AGENTS"/gdoc-*.md 2>/dev/null)
holds "the plan reviewer is named" "$named" "planning:plan-review"
holds "Codex is named" "$named" "codex-second-opinion"
for copy in "$AGENTS/planning-plan-review.md" "$AGENTS/plan-review.md" \
    "$AGENTS/codex-second-opinion.md"; do
    if [[ -e $copy ]]; then
        bad "an existing seat is defined again: ${copy#"$REPO"/}"
    else
        pass "no second definition of ${copy#"$REPO"/}"
    fi
done

# --- every path a definition names exists ----------------------------------
# A backticked token is a path when it holds a slash and no placeholder. A
# `~`, a `$` or a `<` opens something this repository does not hold: a home
# directory, an environment variable, a name the caller fills in.

paths_in() {
    local file=$1
    grep -o '`[^`]*`' "$file" |
        tr -d '`' |
        grep '/' |
        grep -v '[ <>*$~|]' |
        grep -v '^/' |
        sed 's|/$||' |
        sort -u
}

for file in "$AGENTS"/gdoc-*.md; do
    [[ -f $file ]] || continue
    short=${file#"$REPO"/}
    missing=()
    while read -r path; do
        [[ -n $path ]] || continue
        [[ -e $REPO/$path ]] || missing+=("$path")
    done <<< "$(paths_in "$file")"
    if [[ ${#missing[@]} -eq 0 ]]; then
        pass "$short names only paths that exist"
    else
        bad "$short names paths that do not exist: ${missing[*]}"
    fi
done

# --- the team pages --------------------------------------------------------
# The two pages a session reads instead of asking: `docs/team/README.md` for
# who holds what, `docs/team/lead.md` for what the lead does. They are checked
# the same way a definition is, because they are briefs too: a page that sends
# the lead to a file that moved sends it nowhere.

TEAM=$REPO/docs/team

for page in README.md lead.md; do
    if [[ -f $TEAM/$page ]]; then
        pass "docs/team/$page exists"
    else
        bad "docs/team/$page is missing"
    fi
done

teampage=$(cat "$TEAM/README.md" 2>/dev/null)
for team in "Stream-aligned" "Platform" "Enabling" "Complicated subsystem"; do
    holds "the team page names the team" "$teampage" "$team"
done
for seat in gdoc-platform-operator gdoc-design-reviewer gdoc-trust-guardian \
    planning:plan-review codex-second-opinion; do
    holds "the team page names the seat" "$teampage" "$seat"
done
holds "the team page names the models" "$teampage" "Opus"
holds "the team page names the models" "$teampage" "Sonnet"
for mode in "X-as-a-Service" "facilitating" "collaboration" "The day"; do
    holds "the team page names the interaction mode" "$teampage" "$mode"
done

leadpage=$(cat "$TEAM/lead.md" 2>/dev/null)
holds "the lead has the five questions as a template" "$leadpage" \
    "the five questions"
if [[ $(grep -c '^[1-5]\. <' "$TEAM/lead.md" 2>/dev/null) -eq 5 ]]; then
    pass "the lead's template has five numbered questions"
else
    bad "the lead's template does not hold five numbered questions"
fi
holds "the lead knows who to call" "$leadpage" "gdoc-trust-guardian"
holds "the lead knows who to call" "$leadpage" "planning:plan-review"
holds "the lead knows who to call" "$leadpage" "gdoc-design-reviewer"
holds "the lead knows who to call" "$leadpage" "gdoc-platform-operator"
holds "the lead messages Nail on a decision" "$leadpage" "A decision"
holds "the lead messages Nail on a blocked run" "$leadpage" \
    "A blocked run he must act on"
holds "the lead messages Nail on a release" "$leadpage" "A release"
holds "one milestone and one review a day" "$leadpage" \
    "One milestone and one review a day"
holds "the release stops without the go" "$leadpage" "Without \`GO\`"
holds "the lead compacts after a release" "$leadpage" "compact"
holds "the lead never re-reads the renders" "$leadpage" \
    "Never re-read the renders"
holds "the lead reads the status file, not the log" "$leadpage" "status"

for file in "$TEAM"/README.md "$TEAM"/lead.md; do
    [[ -f $file ]] || continue
    short=${file#"$REPO"/}
    missing=()
    while read -r path; do
        [[ -n $path ]] || continue
        [[ -e $REPO/$path ]] || missing+=("$path")
    done <<< "$(paths_in "$file")"
    if [[ ${#missing[@]} -eq 0 ]]; then
        pass "$short names only paths that exist"
    else
        bad "$short names paths that do not exist: ${missing[*]}"
    fi
done

# The board README is the platform's page, and it points at the team's.
board=$(cat "$REPO/.ralphex/board/README.md" 2>/dev/null)
holds "the board README points at the team" "$board" "docs/team/README.md"

printf '\n'
if [[ $fails -eq 0 ]]; then
    printf 'test_team: all checks pass\n'
else
    printf 'test_team: %s check(s) failed\n' "$fails"
    exit 1
fi
