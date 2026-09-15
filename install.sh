#!/usr/bin/env bash
# Build the gdoc binary and link it, and its skill, into place.
#
# Safe to re-run. Every step checks the current state first and does nothing
# when it is already correct. The skill is linked, not copied, so editing
# skills/gdoc-review/SKILL.md takes effect immediately and the skill can never
# disagree with the binary it calls.
#
# This script never touches ~/.config/gdoc-agent/. Your token and your config
# are yours: nothing here creates, rewrites or removes a file in that folder.
# Signing in is `gdoc auth login`, and it is the only thing that writes there.

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILLS_DIR="$HOME/.claude/skills"
BIN_DIR="$HOME/.local/bin"
GO_BIN="$REPO/bin/gdoc"
SKILL=gdoc-review

fail() {
    printf 'install: %s\n' "$1" >&2
    exit 1
}

warn() {
    printf 'install: warning: %s\n' "$1" >&2
}

# --------------------------------------------------------------------------
# The binary
# --------------------------------------------------------------------------
#
# One static binary, built from go/. It is the whole tool: there is nothing
# else to install.

if command -v go >/dev/null 2>&1; then
    make -C "$REPO" build >/dev/null || fail "go build failed. Run: make build"
    [ -x "$GO_BIN" ] || fail "make build wrote no $GO_BIN"
elif [ -x "$GO_BIN" ]; then
    warn "go is not on PATH, so $GO_BIN was not rebuilt. The one already there is used."
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
if [ -L "$BIN_DIR/gdoc2" ]; then
    rm "$BIN_DIR/gdoc2"
    printf 'install: removed %s/gdoc2. gdoc is the binary now.\n' "$BIN_DIR"
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
# The skill
# --------------------------------------------------------------------------

mkdir -p "$SKILLS_DIR"

src="$REPO/skills/$SKILL"
dst="$SKILLS_DIR/$SKILL"

[ -d "$src" ] || fail "missing $src"

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

# The apply skill was removed with the Python tool it called. An old install
# left a link to it here, and a link to a folder that no longer exists is a
# skill Claude Code fails to load. Only a link into this repo is removed: a
# folder somebody wrote themselves is theirs.
stale="$SKILLS_DIR/gdoc-apply"
if [ -L "$stale" ] && [ "$(readlink "$stale")" = "$REPO/skills/gdoc-apply" ]; then
    rm "$stale"
    printf 'install: removed %s. That skill was retired with the Python tool.\n' "$stale"
elif [ -e "$stale" ]; then
    warn "$stale is not a link into this repo. Left alone."
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
printf '  gdoc     %s -> %s\n' "$link" "$GO_BIN"
printf '\n  skill (linked, so edits are live with no reinstall)\n'
printf '    %-12s -> %s\n' "$SKILL" "$(readlink "$dst")"
printf '\n  next     gdoc auth status, and gdoc auth login if it says signed out\n\n'
