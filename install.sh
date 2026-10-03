#!/usr/bin/env bash
# Build the gdoc binary and link it, its completion and its skills, into place.
#
# Safe to re-run. Every step checks the current state first and does nothing
# when it is already correct. A skill is linked, not copied, so editing a
# skills/<name>/SKILL.md takes effect immediately and a skill can never
# disagree with the binary it calls.
#
# One file is written again on every run: bin/gdoc.zsh, the shell completion.
# When it cannot be written the run warns, says what the binary reported, and
# carries on. It is a rendering of the binary that was just built, so it has to
# be written again after an upgrade or a Tab would offer a flag the parser no
# longer takes. Nothing else here replaces a file. .zshrc is never edited: the
# summary prints the one line to add and a person adds it.
#
# One flag, and a run without it is the run this script has always been:
#
#   --desktop   write bin/gdoc.mcpb, the Claude Desktop extension, and open it
#
# The extension names bin/gdoc's own path, the same path ~/.local/bin/gdoc links
# to, so `make build` is the whole upgrade: quit Claude Desktop, open it again,
# and the chat is running the build you just made.
#
# This script never touches ~/.config/gdoc-agent/. Your token and your config
# are yours: nothing here creates, rewrites or removes a file in that folder.
# Signing in is `gdoc auth login`, and it is the only thing that writes there.

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILLS_DIR="$HOME/.claude/skills"
BIN_DIR="$HOME/.local/bin"
GO_BIN="$REPO/bin/gdoc"
COMPLETION="$REPO/bin/gdoc.zsh"
# The Claude Desktop extension: the template this repository owns, and the file
# --desktop writes from it, beside the binary it names. bin/ is not in git, so
# the written extension is nobody's to commit.
TEMPLATE="$REPO/release/mcpb/manifest.json"
MCPB="$REPO/bin/gdoc.mcpb"
# A checkout names no release, and a manifest version field cannot be empty.
DEV_VERSION="0.0.0-dev"
# Every skill this repo owns. The link block below runs once per name, so a
# sixth skill is one word here and nothing else. A boundary test compares this
# array with skills/ on every commit, in both directions.
SKILLS=(gdoc-align gdoc-export gdoc-publish gdoc-restyle gdoc-review)

fail() {
    printf 'install: %s\n' "$1" >&2
    exit 1
}

warn() {
    printf 'install: warning: %s\n' "$1" >&2
}

usage() {
    cat >&2 <<'USAGE'
Usage: ./install.sh [--desktop]

  --desktop   write bin/gdoc.mcpb, the Claude Desktop extension, and open it

With no flags it builds the binary, links it and links the skills, and says
nothing to Claude Desktop.
USAGE
}

# --------------------------------------------------------------------------
# The arguments
# --------------------------------------------------------------------------
#
# Refused by name rather than ignored, the way the binary refuses what it did
# not understand.

desktop=0
while [ $# -gt 0 ]; do
    case "$1" in
        --desktop)
            desktop=1
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            usage
            fail "'$1' is not an argument this script takes"
            ;;
    esac
done

# What --desktop needs, judged before anything is built, so a run that cannot
# write the extension has changed nothing. Every one of the three is asked here
# and nowhere else: a check further down runs after the binary is built and the
# links are moved, which is the outcome this block exists to prevent.
if [ "$desktop" -eq 1 ]; then
    [ -f "$TEMPLATE" ] || fail "$TEMPLATE is missing, and --desktop fills it. Check out this repository again."
    command -v zip >/dev/null 2>&1 || fail "zip is not on PATH, and the Claude Desktop extension is a zip."
    # This checkout's path goes into JSON as a string and into sed as a
    # replacement, and a checkout sits wherever somebody cloned it. The four
    # characters that would break either are refused by name, the ampersand
    # among them: in a sed replacement it stands for the text that matched, so a
    # path holding one would be written with @BIN@ pasted back into it.
    case "$GO_BIN" in
        *'"'*|*'\'*|*'|'*|*'&'*)
            fail "$GO_BIN carries a quote, a backslash, a pipe or an ampersand, and a manifest naming it could not be written.
  Move the checkout to a path without those, or drop --desktop."
            ;;
    esac
fi

# --------------------------------------------------------------------------
# The binary
# --------------------------------------------------------------------------
#
# One static binary, built from go/. It is the whole tool: there is nothing
# else to install.

# Whether this run rebuilt the binary is remembered, because the completion
# block below is the one other place that has to know. A binary this run built
# and a binary an earlier run left behind fail that step for different reasons,
# and only the second one is fixed by installing Go.
if command -v go >/dev/null 2>&1; then
    make -C "$REPO" build >/dev/null || fail "go build failed. Run: make build"
    [ -x "$GO_BIN" ] || fail "make build wrote no $GO_BIN"
    go_rebuilt=1
elif [ -x "$GO_BIN" ]; then
    warn "go is not on PATH, so $GO_BIN was not rebuilt. The one already there is used."
    go_rebuilt=0
else
    fail "go is not on PATH and there is no $GO_BIN. Install Go, then re-run."
fi

# --------------------------------------------------------------------------
# One command on PATH
# --------------------------------------------------------------------------
#
# Linked rather than copied, so `make build` refreshes it with no reinstall.
#
# ~/.local/bin because that is where pipx and uv put tools. It is not on the
# macOS default PATH, which is /etc/paths plus /etc/paths.d, so the check below
# is not decoration.

mkdir -p "$BIN_DIR"
link="$BIN_DIR/gdoc"

if [ -L "$link" ]; then
    # An older install pointed this somewhere else. Repointing it is the upgrade.
    [ "$(readlink "$link")" = "$GO_BIN" ] || ln -sf "$GO_BIN" "$link"
elif [ -e "$link" ]; then
    # Somebody else's gdoc. Leave it: shadowing a real program is worse than
    # asking the person to look.
    warn "$link exists and is not a link to this install. Left alone."
else
    ln -s "$GO_BIN" "$link"
fi

# gdoc2 was the temporary name the binary was tested under, beside the old
# Python tool. `gdoc` is the binary now, so the second name is removed rather
# than left to mean the same thing twice.
# Only a link into a checkout of this repo is removed, the way the retired
# skill below is decided on: somebody else's gdoc2 is theirs.
if [ -L "$BIN_DIR/gdoc2" ]; then
    case "$(readlink "$BIN_DIR/gdoc2")" in
        */bin/gdoc)
            rm "$BIN_DIR/gdoc2"
            printf 'install: removed %s/gdoc2. gdoc is the binary now.\n' "$BIN_DIR"
            ;;
        *)
            warn "$BIN_DIR/gdoc2 is not a link to this install. Left alone."
            ;;
    esac
elif [ -e "$BIN_DIR/gdoc2" ]; then
    warn "$BIN_DIR/gdoc2 exists and is not a link to this install. Left alone."
fi

case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *)
        printf 'install: %s is not on your PATH. Add it with:\n' "$BIN_DIR" >&2
        printf '  echo '"'"'export PATH="$HOME/.local/bin:$PATH"'"'"' >> ~/.zshrc\n' >&2
        printf 'install: until then, call it as %s\n' "$GO_BIN" >&2
        ;;
esac

# --------------------------------------------------------------------------
# The skills
# --------------------------------------------------------------------------
#
# Every source is checked before any link is made, so a name in SKILLS that
# the repo does not have stops the run with nothing half done.

mkdir -p "$SKILLS_DIR"

for skill in "${SKILLS[@]}"; do
    [ -d "$REPO/skills/$skill" ] || fail "missing $REPO/skills/$skill"
done

for skill in "${SKILLS[@]}"; do
    src="$REPO/skills/$skill"
    dst="$SKILLS_DIR/$skill"

    if [ -L "$dst" ]; then
        # Already a link. Leave it alone when it points here, repoint otherwise.
        [ "$(readlink "$dst")" = "$src" ] || { rm "$dst"; ln -s "$src" "$dst"; }
    elif [ -e "$dst" ]; then
        # A real directory from an older copy-based install. Replacing it is only
        # safe when it holds no edits that exist nowhere else.
        if ! diff -rq -x .DS_Store "$src" "$dst" >/dev/null 2>&1; then
            fail "$dst differs from the repo.
  Copy the edits you want into $src, then re-run.
  Compare with: diff -r -x .DS_Store '$src' '$dst'"
        fi
        rm -rf "$dst"
        ln -s "$src" "$dst"
    else
        ln -s "$src" "$dst"
    fi
done

# The apply skill was removed with the Python tool it called. An old install
# left a link to it here, and a link to a folder that no longer exists is a
# skill Claude Code fails to load. The symlink is decided on first, as the two
# links above are, because -e follows the link and a dangling one is the whole
# point: after the repo moved, the target reads as the old path. Any link whose
# target is a checkout's skills/gdoc-apply is an old install of this repo and is
# removed. A folder somebody wrote themselves is theirs.
stale="$SKILLS_DIR/gdoc-apply"
if [ -L "$stale" ]; then
    case "$(readlink "$stale")" in
        */skills/gdoc-apply)
            rm "$stale"
            printf 'install: removed %s. That skill was retired with the Python tool.\n' "$stale"
            ;;
        *)
            warn "$stale is not a link into this repo. Left alone."
            ;;
    esac
elif [ -e "$stale" ]; then
    warn "$stale is not a link into this repo. Left alone."
fi

# --------------------------------------------------------------------------
# The completion
# --------------------------------------------------------------------------
#
# Written by the binary that was just built, into bin/ beside it, so the tool
# and what Tab offers are always the same table. --force because this file is
# gdoc's own: rewriting it is what this line is for.
#
# Last, after the skills, and a warning rather than a failure. The binary here
# is not always the one this checkout describes: with no Go on PATH the block
# at the top keeps whatever was already built, and a binary older than the
# completion command refuses the call. That is a reason to say so, not a reason
# to leave the skills unlinked. The summary reads this run's outcome and not the
# file: bin/ is not in git and survives between runs, so a script an earlier run
# wrote is still sitting there after this one failed, and it renders a binary
# that is no longer the one on PATH.
#
# .zshrc is not edited here. The summary prints the line to add when that file
# does not already name this one.

# The reason is captured, never discarded. Every command prints one JSON object
# on stdout and that object carries the error, so >/dev/null would throw away
# the only diagnostic there is and leave this block guessing a cause: a full
# disk and a binary too old to have this command read the same from out here.
# The second line is printed only when this run did not rebuild, because then
# a binary too old to answer is the likely cause and installing Go is the fix.
# A run that did build the binary it just called has some other cause, and the
# captured reason names it: telling that person to install Go would be wrong.
completion_written=0
if reason="$("$GO_BIN" completion zsh --out "$COMPLETION" --force 2>&1)"; then
    completion_written=1
else
    warn "$GO_BIN wrote no completion to $COMPLETION: $reason"
    if [ "$go_rebuilt" -eq 0 ]; then
        warn "that binary is the one an earlier run built: install Go, then re-run."
    fi
fi

# --------------------------------------------------------------------------
# The Claude Desktop extension, when the run asked for it
# --------------------------------------------------------------------------
#
# A .mcpb is a zip holding a manifest, and the manifest names the command Claude
# Desktop starts. Here that command is this checkout's own bin/gdoc, which is
# the point of the flag: the extension follows the build rather than a copy of
# it. Written into a temp directory and moved over, so a second run leaves one
# entry in the zip rather than two.

desktop_opened=0
if [ "$desktop" -eq 1 ]; then
    # The path this names was judged before the build, where a refusal still
    # costs nothing.
    mcpb_work="$(mktemp -d)"
    sed -e "s|@BIN@|$GO_BIN|g" -e "s|@VERSION@|$DEV_VERSION|g" \
        "$TEMPLATE" > "$mcpb_work/manifest.json"
    ( cd "$mcpb_work" && zip -q "$mcpb_work/gdoc.mcpb" manifest.json ) ||
        fail "zip wrote no gdoc.mcpb, so nothing was written to $MCPB"
    mv "$mcpb_work/gdoc.mcpb" "$MCPB"
    rm -rf "$mcpb_work"

    # Opening a .mcpb is how Claude Desktop installs one. GDOC_DESKTOP_OPEN is
    # the seam release/test-desktop.sh runs this under.
    if [ "$(uname -s)" = "Darwin" ]; then
        opener="${GDOC_DESKTOP_OPEN:-/usr/bin/open}"
        if reason="$("$opener" "$MCPB" 2>&1)"; then
            desktop_opened=1
        else
            warn "$opener did not open $MCPB: $reason"
        fi
    else
        printf 'install: %s is written. Claude Desktop runs on macOS, so open it there.\n' "$MCPB" >&2
    fi
fi

# --------------------------------------------------------------------------
# What is installed
# --------------------------------------------------------------------------

commit="$(git -C "$REPO" rev-parse --short HEAD 2>/dev/null || echo unknown)"
branch="$(git -C "$REPO" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
# --porcelain, not "diff --quiet", so a brand new untracked SKILL.md still
# counts as unsaved work. The linked skill makes untracked files live.
state=""
[ -z "$(git -C "$REPO" status --porcelain 2>/dev/null)" ] || state=" + uncommitted changes"

printf '\ngdoc installed\n\n'
printf '  source   %s\n' "$REPO"
printf '  version  %s on %s%s\n' "$commit" "$branch" "$state"
# The measured target, not the intended one: when the branch above left
# somebody else's gdoc alone, the summary has to say so rather than claim a link
# it did not make.
printf '  gdoc     %s -> %s\n' "$link" "$(readlink "$link" 2>/dev/null || echo 'left alone, not this install')"
# This run, not the filesystem: a file at that path may be the previous run's,
# and reporting it as fresh would tell somebody a stale script matches the
# binary they just built. Three outcomes, so all three are said out loud.
if [ "$completion_written" -eq 1 ]; then
    printf '  complete %s\n' "$COMPLETION"
    # grep -F, so the path is a string and not a pattern. A missing .zshrc reads
    # the same as one that does not name the file: the line is printed either way.
    if grep -qF "$COMPLETION" "$HOME/.zshrc" 2>/dev/null; then
        printf '           sourced from ~/.zshrc already\n'
    else
        printf '           add this line to ~/.zshrc:  source %s\n' "$COMPLETION"
    fi
elif [ -f "$COMPLETION" ]; then
    printf '  complete %s\n' "$COMPLETION"
    printf '           not rewritten this run, see the warning above\n'
else
    printf '  complete not written, see the warning above\n'
fi
printf '\n  claude desktop\n'
if [ "$desktop" -eq 1 ]; then
    printf '    extension %s -> %s\n' "$MCPB" "$GO_BIN"
    if [ "$desktop_opened" -eq 1 ]; then
        printf '    opened. Install it, then quit Claude Desktop and open it again.\n'
    else
        printf '    open that file to install it, then quit Claude Desktop and open it again.\n'
    fi
    printf '    it names bin/gdoc, so after make build quit Claude Desktop and open it again.\n'
else
    printf '    not touched. For gdoc in Claude Desktop, re-run this with --desktop.\n'
fi

printf '\n  skills (linked, so edits are live with no reinstall)\n'
for skill in "${SKILLS[@]}"; do
    printf '    %-12s -> %s\n' "$skill" "$(readlink "$SKILLS_DIR/$skill")"
done
# The links above are for whoever works on this repository, and they exist so
# an edit to a SKILL.md is live before it is committed. That is not how anyone
# else should get these skills: a colleague adds this repository as a Claude
# Code marketplace and installs the plugin, which carries the same five
# folders and updates when they ask it to. Said here because the person reading
# this summary is the one a colleague asks how to get them.
printf '\n  colleagues install the skills as a plugin, not as links:\n'
printf '    /plugin marketplace add nhusnullin/gdoc\n'
printf '    /plugin install altery@gdoc\n'
printf '\n  next     gdoc auth status, and gdoc auth login if it says signed out\n\n'
