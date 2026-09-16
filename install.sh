#!/usr/bin/env bash
# Build the gdoc binary and link it, its completion and its skills, into place.
#
# Safe to re-run. Every step checks the current state first and does nothing
# when it is already correct. A skill is linked, not copied, so editing a
# skills/<name>/SKILL.md takes effect immediately and a skill can never
# disagree with the binary it calls.
#
# One file is rewritten on every run: bin/gdoc.zsh, the shell completion. It
# is a rendering of the binary that was just built, so it has to be written
# again after an upgrade or a Tab would offer a flag the parser no longer
# takes. Nothing else here replaces a file. .zshrc is never edited: the
# summary prints the one line to add and a person adds it.
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
# Every skill this repo owns. The link block below runs once per name, so a
# fourth skill is one word here and nothing else.
SKILLS=(gdoc-review gdoc-publish gdoc-restyle)

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
# The completion
# --------------------------------------------------------------------------
#
# Written by the binary that was just built, into bin/ beside it, so the tool
# and what Tab offers are always the same table. --force because this file is
# gdoc's own: rewriting it is what this line is for.
#
# .zshrc is not edited here. The summary prints the line to add when that file
# does not already name this one.

"$GO_BIN" completion zsh --out "$COMPLETION" --force >/dev/null \
    || fail "the completion could not be written to $COMPLETION"

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
printf '  complete %s\n' "$COMPLETION"
# grep -F, so the path is a string and not a pattern. A missing .zshrc reads
# the same as one that does not name the file: the line is printed either way.
if grep -qF "$COMPLETION" "$HOME/.zshrc" 2>/dev/null; then
    printf '           sourced from ~/.zshrc already\n'
else
    printf '           add this line to ~/.zshrc:  source %s\n' "$COMPLETION"
fi
printf '\n  skills (linked, so edits are live with no reinstall)\n'
for skill in "${SKILLS[@]}"; do
    printf '    %-12s -> %s\n' "$skill" "$(readlink "$SKILLS_DIR/$skill")"
done
printf '\n  next     gdoc auth status, and gdoc auth login if it says signed out\n\n'
