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
# One command on PATH
# --------------------------------------------------------------------------
#
# Linked rather than copied, so it follows the venv with no reinstall. The venv
# script carries an absolute shebang, so a symlink to it resolves correctly.
#
# ~/.local/bin because that is where pipx and uv put tools. It is not on the
# macOS default PATH, which is /etc/paths plus /etc/paths.d, so the check below
# is not decoration.

mkdir -p "$BIN_DIR"
link="$BIN_DIR/gdoc"

if [ -L "$link" ]; then
    [ "$(readlink "$link")" = "$VENV/bin/gdoc" ] || ln -sf "$VENV/bin/gdoc" "$link"
elif [ -e "$link" ]; then
    # Somebody else's gdoc. Leave it: shadowing a real program is worse than
    # asking the person to look.
    warn "$link exists and is not a link to this install. Left alone."
else
    ln -s "$VENV/bin/gdoc" "$link"
fi

case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *)
        printf 'install: %s is not on your PATH. Add it with:\n' "$BIN_DIR" >&2
        printf '  echo '"'"'export PATH="$HOME/.local/bin:$PATH"'"'"' >> ~/.zshrc\n' >&2
        printf 'install: until then, call it as %s/bin/gdoc\n' "$VENV" >&2
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
        if [ ! -f "$CONFIG_DIR/oauth-client.json" ]; then
            warn "no $CONFIG_DIR/oauth-client.json yet. See README.md, Configure"
        elif [ ! -f "$CONFIG_DIR/oauth-token.json" ]; then
            # Not a fault. It is the next step.
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
printf '  cli      %s\n' "$VENV/bin/gdoc"
printf '\n  skills (linked, so edits are live with no reinstall)\n'
for skill in "${SKILLS[@]}"; do
    printf '    %-12s -> %s\n' "$skill" "$(readlink "$SKILLS_DIR/$skill")"
done
printf '\n'
