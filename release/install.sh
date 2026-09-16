#!/usr/bin/env bash
# Install gdoc on a colleague's machine: one binary, its completion, and the
# three skills when the run asks for them.
#
# Two entrances, one install. Unpack a release zip and run the install.sh
# inside it, and it installs the gdoc sitting beside it. Run the one line below
# and it asks GitHub for the newest stable release, downloads the zip for this
# machine and its checksum file, checks one against the other, unpacks it into a
# temp dir and installs from there.
#
#   curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash
#
# Nothing is asked. Every choice is a flag, and a run with no flags installs the
# binary and touches no skill folder:
#
#   --tag <tag>       install that release rather than the newest stable
#   --skills global   copy the three skills into ~/.claude/skills
#   --skills local    copy them into ./.claude/skills
#
# Skills normally arrive as a Claude Code plugin, which Claude Code updates when
# a person asks it to:
#
#   /plugin marketplace add nhusnullin/gdoc
#   /plugin install gdoc@gdoc
#
# --skills is the second route, for a machine where plugins are turned off. It
# copies rather than links, and it marks what it wrote with a .gdoc-installed
# file: a folder carrying that file is this script's to replace, and a folder
# without one is somebody's own work and is refused by name.
#
# ~/.zshrc is never edited. The summary prints the line to add and a person adds
# it. ~/.config/gdoc-agent gets the completion file this run renders and nothing
# else: the token is not read, written or removed here. Signing in is
# `gdoc auth login`.
#
# This is not the developer install. In a checkout of this repository, ./install.sh
# at the root builds from source and links the skills so an edit is live. The two
# would fight over the same path, so this one refuses to run inside a checkout.

set -euo pipefail

# The repository the releases come from. It is public, so every request below
# is unauthenticated and nothing here reads a token.
REPO_SLUG="nhusnullin/gdoc"
API="https://api.github.com/repos/$REPO_SLUG/releases"
DOWNLOAD="https://github.com/$REPO_SLUG/releases/download"

BIN_DIR="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/gdoc-agent"
COMPLETION="$CONFIG_DIR/completion.zsh"

# Every skill the release zip carries. The plugin ships the same three folders,
# and a boundary test compares this array with skills/ on every commit.
SKILLS=(gdoc-review gdoc-publish gdoc-restyle)

# Every platform a release carries a zip for. The same list as release/platforms
# beside this script in the repository, compared on every commit, because this
# script travels inside the zip and cannot read that file from a stranger's
# machine.
PLATFORMS=(darwin-arm64 darwin-amd64)

# The file that says a skill folder is this script's to replace.
MARKER=".gdoc-installed"

fail() {
    printf 'install: %s\n' "$1" >&2
    exit 1
}

warn() {
    printf 'install: warning: %s\n' "$1" >&2
}

usage() {
    cat >&2 <<'USAGE'
Usage: install.sh [--tag <tag>] [--skills global|local]

  --tag <tag>       install that release rather than the newest stable
  --skills global   copy the three skills into ~/.claude/skills
  --skills local    copy them into ./.claude/skills

With no flags it installs the binary into ~/.local/bin and touches no skill
folder, because the plugin is how a colleague gets the skills.
USAGE
}

# --------------------------------------------------------------------------
# The arguments
# --------------------------------------------------------------------------
#
# Refused by name rather than ignored, the way the binary refuses what it did
# not understand. A typo in --skills is somebody expecting three folders to
# appear, and a run that shrugged at it would leave them looking for them.

tag=""
skills_where=""

while [ $# -gt 0 ]; do
    case "$1" in
        --tag)
            [ $# -ge 2 ] || fail "--tag takes the tag to install, such as --tag v2.1.0"
            tag="$2"
            shift 2
            ;;
        --skills)
            [ $# -ge 2 ] || fail "--skills takes global or local"
            case "$2" in
                global|local) skills_where="$2" ;;
                *) fail "--skills takes global or local, and this run said '$2'" ;;
            esac
            shift 2
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

# --------------------------------------------------------------------------
# Not inside a checkout
# --------------------------------------------------------------------------
#
# The developer install links ~/.local/bin/gdoc and the three skill folders into
# a checkout, so an edit is live. This one copies files in. Running this one in
# a checkout would put a release binary over that link and copies over those
# links, and the person would be left with a tool that no longer follows the
# source they are editing. Checked first, before anything is fetched.

checkout_above() {
    dir="$1"
    while [ "$dir" != "/" ] && [ -n "$dir" ]; do
        if [ -e "$dir/.git" ] && [ -f "$dir/go/cmd/gdoc/main.go" ] && [ -f "$dir/install.sh" ]; then
            printf '%s\n' "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

script_dir=""
if [ -n "${BASH_SOURCE[0]:-}" ]; then
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

for where in "$PWD" "$script_dir"; do
    [ -n "$where" ] || continue
    if root="$(checkout_above "$where")"; then
        fail "$root is a checkout of this repository, and this script is the one that installs a release.
  For the checkout, run: $root/install.sh
  It builds from source and links the skills, so an edit is live."
    fi
done

# --------------------------------------------------------------------------
# Which entrance
# --------------------------------------------------------------------------
#
# A gdoc binary beside this script means an unpacked release zip: install what
# is there. Anything else, including this script read from a pipe, means fetch.
# The pipe is decided on BASH_SOURCE rather than on what happens to be in the
# working directory, so `curl | bash` in a folder holding somebody else's gdoc
# still installs the release it was asked for.

src=""
origin=""
if [ -n "$script_dir" ] && [ -x "$script_dir/gdoc" ]; then
    src="$script_dir"
    origin="the zip unpacked at $script_dir"
fi

work=""
cleanup() {
    [ -z "$work" ] || rm -rf "$work"
}
trap cleanup EXIT

if [ -z "$src" ]; then
    for tool in curl unzip shasum uname; do
        command -v "$tool" >/dev/null 2>&1 || fail "$tool is not on PATH, and fetching a release needs it"
    done

    os="$(uname -s)"
    arch="$(uname -m)"
    case "$os" in
        Darwin) os="darwin" ;;
        *) os="$(printf '%s' "$os" | tr '[:upper:]' '[:lower:]')" ;;
    esac
    case "$arch" in
        arm64|aarch64) arch="arm64" ;;
        x86_64|amd64) arch="amd64" ;;
    esac
    platform="$os-$arch"

    offered=0
    for known in "${PLATFORMS[@]}"; do
        [ "$known" = "$platform" ] && offered=1
    done
    if [ "$offered" -eq 0 ]; then
        fail "a release carries no zip for $platform. It carries ${PLATFORMS[*]}.
  Windows joins that list after a colleague has run its checklist."
    fi

    if [ -z "$tag" ]; then
        # The newest stable release. A stable tag is x.y.0: patch numbers are
        # the nightly's, and a nightly is installed by naming it with --tag.
        # An anonymous listing does not carry drafts, so nothing here has to
        # skip one.
        listing="$(curl -fsSL -H 'Accept: application/vnd.github+json' "$API?per_page=100")" ||
            fail "$API did not answer. Check the network, then re-run."
        # The || true is load bearing under pipefail: a listing with no stable
        # release in it makes the middle grep exit 1, and the empty tag below
        # is the sentence this run should end on rather than a silent exit.
        tag="$(printf '%s' "$listing" |
            grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' |
            sed -e 's/.*"\([^"]*\)"$/\1/' |
            grep -E '^v[0-9]+\.[0-9]+\.0$' |
            sort -t. -k1.2,1n -k2,2n -k3,3n |
            tail -1)" || true
        [ -n "$tag" ] || fail "$REPO_SLUG has published no stable release yet. Ask Nail, or name a tag with --tag."
    fi

    asset="gdoc-$tag-$platform.zip"
    sums="SHA256SUMS-$tag"
    work="$(mktemp -d)"

    curl -fsSL -o "$work/$asset" "$DOWNLOAD/$tag/$asset" ||
        fail "$DOWNLOAD/$tag/$asset could not be downloaded. Check the tag, then re-run."
    curl -fsSL -o "$work/$sums" "$DOWNLOAD/$tag/$sums" ||
        fail "$DOWNLOAD/$tag/$sums could not be downloaded, so nothing could be verified."

    # Verified before anything is unpacked, and unpacked before anything on this
    # machine is replaced. A zip that does not match its published checksum
    # leaves the gdoc already installed exactly where it was.
    line="$(grep -E "[ *]$asset\$" "$work/$sums" || true)"
    [ -n "$line" ] || fail "$sums names no $asset, so what was downloaded cannot be checked against it."
    want="${line%% *}"
    got="$(shasum -a 256 "$work/$asset" | awk '{print $1}')"
    [ "$want" = "$got" ] || fail "$asset hashes to $got and $sums promised $want. Nothing was installed."

    unzip -q "$work/$asset" -d "$work/unpacked" || fail "$asset is not a zip that unpacks."
    src="$work/unpacked"
    origin="$tag, downloaded and verified against $sums"
fi

[ -x "$src/gdoc" ] || fail "$src holds no gdoc binary, so there is nothing here to install"

# --------------------------------------------------------------------------
# One command on PATH
# --------------------------------------------------------------------------
#
# Copied, not linked: the temp dir this may have come from is gone by the time
# anyone types gdoc. ~/.local/bin because that is where pipx and uv put tools,
# and it is not on the macOS default PATH, so the check below is not decoration.
#
# Written beside the old one and renamed over it. A binary that is running
# cannot be written through, and rename is what replaces it on both platforms.

mkdir -p "$BIN_DIR"
installed="$BIN_DIR/gdoc"

if [ -L "$installed" ]; then
    fail "$installed is a symlink to $(readlink "$installed"), which is how a checkout installs gdoc.
  Installing a release over it would leave that checkout's own build unreachable.
  Run ./install.sh in the checkout instead, or remove the link and re-run this."
fi

cp "$src/gdoc" "$installed.new"
chmod 755 "$installed.new"
mv "$installed.new" "$installed"

# macOS marks a downloaded file and then refuses to run it. curl and unzip do
# not set the attribute themselves, but a zip a person downloaded in a browser
# and unpacked in Finder carries it, and that is the route somebody takes when
# the one-line install is blocked. Removing an attribute that is not there is
# not an error worth stopping for.
if command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "$installed" >/dev/null 2>&1 || true
fi

version="$("$installed" help 2>/dev/null | sed -n 's/.*"version":"\([^"]*\)".*/\1/p' || true)"
[ -n "$version" ] || version="unknown"

case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *)
        printf 'install: %s is not on your PATH. Add it with:\n' "$BIN_DIR" >&2
        printf '  echo '"'"'export PATH="$HOME/.local/bin:$PATH"'"'"' >> ~/.zshrc\n' >&2
        printf 'install: until then, call it as %s\n' "$installed" >&2
        ;;
esac

# --------------------------------------------------------------------------
# The completion
# --------------------------------------------------------------------------
#
# Rendered by the binary that was just installed, so Tab offers that binary's
# table and never a flag the parser no longer takes. --force because this file
# is gdoc's own. A warning rather than a failure: a missing completion is a
# smaller loss than a person left without the tool.

mkdir -p "$CONFIG_DIR"
completion_written=0
if reason="$("$installed" completion zsh --out "$COMPLETION" --force 2>&1)"; then
    completion_written=1
else
    warn "$installed wrote no completion to $COMPLETION: $reason"
fi

# --------------------------------------------------------------------------
# The skills, when the run asked for them
# --------------------------------------------------------------------------
#
# Every folder is judged before any is written, so a run that refuses one
# refuses before it has replaced another. Three answers per folder: a symlink is
# the developer install and is refused, a folder without the marker is
# somebody's own and is refused, and a folder with the marker is this script's
# and is replaced.

if [ -n "$skills_where" ]; then
    if [ "$skills_where" = "global" ]; then
        skills_root="$HOME/.claude/skills"
    else
        skills_root="$PWD/.claude/skills"
    fi

    problems=""
    for skill in "${SKILLS[@]}"; do
        [ -d "$src/skills/$skill" ] ||
            problems="$problems  $src holds no skills/$skill, so this release cannot install it
"
    done
    for skill in "${SKILLS[@]}"; do
        dst="$skills_root/$skill"
        if [ -L "$dst" ]; then
            problems="$problems  $dst is a symlink, which is how a checkout installs its skills. Remove it, or drop --skills.
"
        elif [ -e "$dst" ] && [ ! -f "$dst/$MARKER" ]; then
            problems="$problems  $dst is a folder this script did not write: it holds no $MARKER. Move it aside, then re-run.
"
        fi
    done
    if [ -n "$problems" ]; then
        printf 'install: the skills were not installed:\n%s' "$problems" >&2
        fail "no skill folder was touched."
    fi

    mkdir -p "$skills_root"
    for skill in "${SKILLS[@]}"; do
        dst="$skills_root/$skill"
        rm -rf "$dst"
        cp -R "$src/skills/$skill" "$dst"
        cat > "$dst/$MARKER" <<MARKERFILE
gdoc install.sh wrote this folder from gdoc $version on $(date -u +%Y-%m-%d).
A folder carrying this file is replaced by the next --skills run.
Delete this file to keep the folder, and the next run will refuse it by name.
MARKERFILE
    done
fi

# --------------------------------------------------------------------------
# What is installed
# --------------------------------------------------------------------------

printf '\ngdoc installed\n\n'
printf '  version  %s\n' "$version"
printf '  from     %s\n' "$origin"
printf '  gdoc     %s\n' "$installed"
if [ "$completion_written" -eq 1 ]; then
    printf '  complete %s\n' "$COMPLETION"
    # grep -F, so the path is a string and not a pattern, and on the tail of the
    # path, so ~/.config, $HOME/.config and the written out home all count as
    # the line already being there.
    if grep -qF "gdoc-agent/completion.zsh" "$HOME/.zshrc" 2>/dev/null; then
        printf '           sourced from ~/.zshrc already\n'
    else
        printf '           add this line to ~/.zshrc:  source %s\n' "$COMPLETION"
    fi
else
    printf '  complete not written, see the warning above\n'
fi

printf '\n  skills\n'
if [ -n "$skills_where" ]; then
    for skill in "${SKILLS[@]}"; do
        printf '    %-12s %s\n' "$skill" "$skills_root/$skill"
    done
    printf '    these are copies. Re-run this with --skills %s after a gdoc update.\n' "$skills_where"
else
    printf '    not touched. The plugin is the first route, and Claude Code updates it:\n'
    printf '      /plugin marketplace add nhusnullin/gdoc\n'
    printf '      /plugin install gdoc@gdoc\n'
    printf '    Where plugins are turned off, re-run this with --skills global.\n'
fi

printf '\n  next     gdoc auth login, if this says signed out:\n\n'
"$installed" auth status || true
printf '\n'
