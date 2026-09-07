#!/usr/bin/env bash
# Install the gdoc CLI and link its skills into Claude Code.
#
# Safe to re-run. Every step checks the current state first and does nothing
# when it is already correct. The skills are linked, not copied, so editing
# skills/*/SKILL.md takes effect immediately and the skill can never disagree
# with the CLI it calls.

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV="$HOME/.config/gdoc-agent/venv"
CONFIG_DIR="$HOME/.config/gdoc-agent"
SKILLS_DIR="$HOME/.claude/skills"
BIN_DIR="$HOME/.local/bin"
GO_BIN="$REPO/bin/gdoc"
SKILLS=(gdoc-review gdoc-apply)

fail() {
    printf 'install: %s\n' "$1" >&2
    exit 1
}

warn() {
    printf 'install: warning: %s\n' "$1" >&2
}

# --------------------------------------------------------------------------
# The CLI
# --------------------------------------------------------------------------

if [ ! -x "$VENV/bin/python" ]; then
    command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"
    printf 'creating venv at %s\n' "$VENV"
    python3 -m venv "$VENV"
fi

"$VENV/bin/pip" install -q --disable-pip-version-check -e "${REPO}[dev]" || fail "pip install failed"

[ -x "$VENV/bin/gdoc" ] || fail "the gdoc command was not created. Check [project.scripts] in pyproject.toml"

# --------------------------------------------------------------------------
# The v2 binary
# --------------------------------------------------------------------------
#
# One static binary, built from go/. It is what `gdoc` on PATH means from M3
# onwards. v1 stays installed and reachable at $VENV/bin/gdoc, which is the
# path its two skills call, so nothing about them changes.

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
    # An older install pointed this at the venv script. Repointing it is the
    # upgrade, and the v1 skills are unaffected: they name the venv path in full.
    [ "$(readlink "$link")" = "$GO_BIN" ] || ln -sf "$GO_BIN" "$link"
elif [ -e "$link" ]; then
    # Somebody else's gdoc. Leave it: shadowing a real program is worse than
    # asking the person to look.
    warn "$link exists and is not a link to this install. Left alone."
else
    ln -s "$GO_BIN" "$link"
fi

# gdoc2 was the temporary name v2 was tested under, beside v1. `gdoc` is v2 now,
# so the second name is removed rather than left to mean the same thing twice.
if [ -L "$BIN_DIR/gdoc2" ]; then
    rm "$BIN_DIR/gdoc2"
    printf 'install: removed %s/gdoc2. gdoc is the Go binary now.\n' "$BIN_DIR"
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

mkdir -p "$SKILLS_DIR"

for skill in "${SKILLS[@]}"; do
    src="$REPO/skills/$skill"
    dst="$SKILLS_DIR/$skill"

    [ -d "$src" ] || fail "missing $src"

    if [ -L "$dst" ]; then
        # Already a link. Leave it alone when it points here, repoint otherwise.
        [ "$(readlink "$dst")" = "$src" ] && continue
        rm "$dst"
    elif [ -e "$dst" ]; then
        # A real directory from an older copy-based install. Replacing it is
        # only safe when it holds no edits that exist nowhere else.
        if ! diff -rq -x .DS_Store "$src" "$dst" >/dev/null 2>&1; then
            fail "$dst differs from the repo.
  Copy the edits you want into $src, then re-run.
  Compare with: diff -r -x .DS_Store '$src' '$dst'"
        fi
        rm -rf "$dst"
    fi

    ln -s "$src" "$dst"
done

# --------------------------------------------------------------------------
# Credentials, checked but never written
# --------------------------------------------------------------------------

[ -f "$CONFIG_DIR/config.json" ] || warn "no $CONFIG_DIR/config.json yet. See README.md"

# Which credential to check for. Asked of the package rather than worked out
# here, because a second copy of the rule would drift from the one in
# gdoc/auth.py and would warn about the wrong missing file.
auth_mode="$("$VENV/bin/python" -c '
from gdoc.auth import configured_mode, resolve_auth_mode

print(resolve_auth_mode(configured_mode()))
' 2>/dev/null || echo oauth)"

case "$auth_mode" in
    service_account)
        [ -f "$CONFIG_DIR/sa-key.json" ] || warn "no $CONFIG_DIR/sa-key.json yet. See README.md"
        ;;
    *)
        # Which client a login would use, asked of the package for the same
        # reason as the mode. Normally "bundled", and then there is nothing to
        # set up: warning about a missing oauth-client.json would be wrong.
        client="$("$VENV/bin/python" -c '
from gdoc.oauth import client_config

try:
    print(client_config()[1])
except Exception:
    print("none")
' 2>/dev/null || echo none)"

        if [ "$client" = "none" ]; then
            warn "this build ships no OAuth client, and there is none at
  $CONFIG_DIR/oauth-client.json. See README.md, the OAuth client section."
        elif [ ! -f "$CONFIG_DIR/oauth-token.json" ]; then
            # Not a fault. It is the next step, and the only one.
            printf 'install: next step: gdoc auth login\n'
        fi
        ;;
esac

# --------------------------------------------------------------------------
# What is installed
# --------------------------------------------------------------------------

commit="$(git -C "$REPO" rev-parse --short HEAD 2>/dev/null || echo unknown)"
branch="$(git -C "$REPO" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
# --porcelain, not "diff --quiet", so a brand new untracked SKILL.md still
# counts as unsaved work. The linked skills make untracked files live.
state=""
[ -z "$(git -C "$REPO" status --porcelain 2>/dev/null)" ] || state=" + uncommitted changes"

printf '\ngdoc installed\n\n'
printf '  source   %s\n' "$REPO"
printf '  version  %s on %s%s\n' "$commit" "$branch" "$state"
printf '  gdoc     %s (v2, the Go binary)\n' "$GO_BIN"
printf '  v1 cli   %s (what the v1 skills call)\n' "$VENV/bin/gdoc"
printf '\n  skills (linked, so edits are live with no reinstall)\n'
for skill in "${SKILLS[@]}"; do
    printf '    %-12s -> %s\n' "$skill" "$(readlink "$SKILLS_DIR/$skill")"
done
printf '\n'
