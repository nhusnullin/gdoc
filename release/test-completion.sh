#!/bin/sh
# What the completion loader does in a real shell, checked from outside Go.
#
# This is a shell test because the thing under test is a few lines of zsh.
# TestNothingRunsAnExternalProgram bans os/exec in every Go file under go/,
# tests included, and bans the calls underneath it, so no Go test can start a
# shell. The Go tests read the loader as bytes; this file sources it. CI runs
# it beside the desktop extension's: `sh release/test-completion.sh`.
#
# Nothing here touches the machine it runs on. The run gets its own HOME, the
# loader is written by a binary built from this checkout, and the gdoc the
# loader finds on PATH is a fake that writes a script defining _gdoc and
# nothing else, so what is measured is the loader and not the completion
# script, which the Go tests already read line by line.
#
# Skipped, not failed, where zsh or go is missing: a colleague on a machine
# with neither still gets the rest of the suite.

set -eu

repo="$(cd "$(dirname "$0")/.." && pwd)"

if ! command -v zsh >/dev/null 2>&1; then
    printf 'test-completion: skipped, no zsh on this machine\n'
    exit 0
fi
if ! command -v go >/dev/null 2>&1; then
    printf 'test-completion: skipped, no go on this machine\n'
    exit 0
fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

failures=0

fail() {
    printf 'test-completion: FAIL %s\n' "$1" >&2
    failures=$((failures + 1))
}

ok() {
    printf 'test-completion: ok   %s\n' "$1"
}

# The binary this checkout builds, which is what writes the loader.
( cd "$repo/go" && go build -o "$work/gdoc-real" ./cmd/gdoc )

home="$work/home"
config="$home/.config/gdoc-agent"
loader="$config/completion.zsh"
written="$config/completion-script.zsh"
mkdir -p "$config"

"$work/gdoc-real" completion zsh --loader --out "$loader" >/dev/null

if [ ! -f "$loader" ]; then
    fail "no loader was written to $loader"
    exit 1
fi
if [ "$(head -n 1 "$loader")" != "# gdoc completion loader" ]; then
    fail "the loader opens with [$(head -n 1 "$loader")]"
fi

# The gdoc the loader finds. It answers the one call the loader makes and
# writes a script that defines _gdoc, so `whence -w _gdoc` is the whole
# question this file asks.
fake_bin="$work/bin"
mkdir -p "$fake_bin"
cat > "$fake_bin/gdoc" <<'FAKE'
#!/bin/sh
out=""
while [ $# -gt 0 ]; do
    case "$1" in
        --out) out="$2"; shift 2 ;;
        *) shift ;;
    esac
done
[ -n "$out" ] || exit 1
cat > "$out" <<'SCRIPT'
# the fake completion script
_gdoc() { return 0 }
compdef _gdoc gdoc
SCRIPT
printf '{"ok":true}\n'
FAKE
chmod 755 "$fake_bin/gdoc"

# The driver, which is what a .zshrc does: compinit, then source the loader.
# The markers are there so the test can say the loader printed nothing, with
# compinit's own words left out of the comparison.
cat > "$work/driver.zsh" <<'DRIVER'
autoload -U compinit
compinit -u -d "$ZCOMPDUMP" >/dev/null 2>&1
print -r -- "BEGIN"
source "$GDOC_LOADER"
print -r -- "END"
print -r -- "whence:$(whence -w _gdoc)"
print -r -- "left:${+_gdoc_completion}"
DRIVER

# between is what the sourced loader printed, and nothing else.
between() {
    sed -n '/^BEGIN$/,/^END$/p' "$1" | sed '1d;$d'
}

run_zsh() {
    out="$1"
    path="$2"
    HOME="$home" PATH="$path" GDOC_LOADER="$loader" ZCOMPDUMP="$home/.zcompdump" \
        zsh -f "$work/driver.zsh" > "$out" 2>&1 ||
        fail "zsh exited non-zero: $(cat "$out")"
}

# --------------------------------------------------------------------------
# A shell with gdoc on PATH
# --------------------------------------------------------------------------

run_zsh "$work/with.txt" "$fake_bin:/usr/bin:/bin"

if [ ! -f "$written" ]; then
    fail "the loader had gdoc write no script to $written"
elif ! grep -q '_gdoc()' "$written"; then
    fail "the script the loader asked for defines no _gdoc: $(cat "$written")"
else
    ok "the loader asks the gdoc on PATH for the script, beside itself"
fi

if ! grep -q '^whence:_gdoc: function$' "$work/with.txt"; then
    fail "_gdoc is not a function after sourcing the loader: $(cat "$work/with.txt")"
else
    ok "sourcing the loader defines _gdoc"
fi

if [ -n "$(between "$work/with.txt")" ]; then
    fail "the loader printed something: [$(between "$work/with.txt")]"
else
    ok "the loader printed nothing"
fi

if ! grep -q '^left:0$' "$work/with.txt"; then
    fail "the loader left a variable behind in the shell: $(cat "$work/with.txt")"
else
    ok "the loader leaves no variable behind"
fi

# --------------------------------------------------------------------------
# A shell with no gdoc on PATH
# --------------------------------------------------------------------------
#
# The completion a person already had keeps working, because the source is
# guarded by the file being readable and not by gdoc being on PATH. That is
# what makes a .zshrc that adds ~/.local/bin after the source line fine.

before="$(cat "$written")"
run_zsh "$work/without.txt" "/usr/bin:/bin"

if [ "$(cat "$written")" != "$before" ]; then
    fail "the script was rewritten by a shell with no gdoc on PATH"
else
    ok "a shell with no gdoc writes nothing"
fi

if [ -n "$(between "$work/without.txt")" ]; then
    fail "the loader printed something with no gdoc on PATH: [$(between "$work/without.txt")]"
else
    ok "with no gdoc on PATH the loader prints nothing"
fi

if ! grep -q '^whence:_gdoc: function$' "$work/without.txt"; then
    fail "_gdoc is not a function with no gdoc on PATH: $(cat "$work/without.txt")"
else
    ok "with no gdoc on PATH the completion already there is still sourced"
fi

# --------------------------------------------------------------------------

if [ "$failures" -ne 0 ]; then
    printf 'test-completion: %d failed\n' "$failures" >&2
    exit 1
fi
printf 'test-completion: all passed\n'
