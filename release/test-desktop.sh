#!/bin/sh
# What `install.sh --desktop` writes, checked from outside both installers.
#
# This is a shell test because the thing under test is a shell script.
# TestNothingRunsAnExternalProgram bans os/exec in every Go file under go/,
# tests included, so no Go test can run an installer. CI runs this file
# instead: `sh release/test-desktop.sh`.
#
# Nothing here touches the machine it runs on. Each run gets its own HOME and
# its own copy of the script under test, a fake gdoc stands in for the binary,
# and a fake opener stands in for /usr/bin/open and records what it was handed.
# The real installers are the files in this repository, copied, so a change to
# one is a change this file sees.
#
# Extracted manifests are compared, never zip bytes: a zip stores the time each
# entry was written, so two runs a second apart produce different bytes and the
# same extension.

set -eu

repo="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

failures=0

fail() {
    printf 'test-desktop: FAIL %s\n' "$1" >&2
    failures=$((failures + 1))
}

ok() {
    printf 'test-desktop: ok   %s\n' "$1"
}

# A gdoc that answers the three calls an installer makes of it: the version out
# of help, the completion script, and the sign-in state at the end.
fake_gdoc() {
    cat > "$1" <<'FAKE'
#!/bin/sh
case "$1" in
    help)
        printf '{"ok":true,"data":{"commands":[]},"version":"v9.9.9"}\n'
        ;;
    completion)
        out=""
        while [ $# -gt 0 ]; do
            case "$1" in
                --out) out="$2"; shift 2 ;;
                *) shift ;;
            esac
        done
        [ -n "$out" ] || exit 1
        printf '# fake completion\n' > "$out"
        printf '{"ok":true}\n'
        ;;
    auth)
        printf '{"ok":true,"data":{"signed_in":false}}\n'
        ;;
    *)
        printf '{"ok":false,"error":"the fake gdoc was asked for %s"}\n' "$1"
        exit 1
        ;;
esac
FAKE
    chmod 755 "$1"
}

# The opener this test hands the installer instead of /usr/bin/open. It writes
# what it was called with and nothing else, so the test can say that the command
# was exactly the one file.
fake_opener() {
    cat > "$1" <<FAKE
#!/bin/sh
printf '%s\n' "\$@" > "$2"
FAKE
    chmod 755 "$1"
}

# listing is every path under a directory, for the before and after comparison
# that says nothing outside the fake HOME was written.
listing() {
    ( cd "$1" && find . | sort )
}

# manifest_from_mcpb extracts the one file a .mcpb carries into a directory and
# prints its path.
manifest_from_mcpb() {
    mcpb="$1"
    into="$2"
    mkdir -p "$into"
    unzip -o -q "$mcpb" -d "$into"
    printf '%s\n' "$into/manifest.json"
}

# check_manifest is every promise the filled manifest makes: both placeholders
# gone, the binary's own path in, and the version without its v.
check_manifest() {
    where="$1"
    file="$2"
    want_bin="$3"
    want_version="$4"

    if grep -q '@BIN@\|@VERSION@' "$file"; then
        fail "$where: the manifest still carries a placeholder"
        return
    fi
    count="$(grep -c -F -- "\"$want_bin\"" "$file" || true)"
    if [ "$count" -ne 2 ]; then
        fail "$where: the manifest names \"$want_bin\" on $count lines, want 2 (the entry point and the command)"
        return
    fi
    if ! grep -q -F -- "\"version\": \"$want_version\"" "$file"; then
        fail "$where: the manifest's version is not $want_version: $(grep version "$file")"
        return
    fi
    ok "$where: the manifest names $want_bin twice and version $want_version"
}

mac=0
[ "$(uname -s)" = "Darwin" ] && mac=1

# --------------------------------------------------------------------------
# The release installer
# --------------------------------------------------------------------------
#
# A stage is an unpacked release zip: the binary, the installer beside it and
# the extension template under mcpb/.

stage="$work/stage"
mkdir -p "$stage/mcpb"
cp "$repo/release/install.sh" "$stage/install.sh"
cp "$repo/release/mcpb/manifest.json" "$stage/mcpb/manifest.json"
fake_gdoc "$stage/gdoc"

home="$work/home"
cwd="$work/cwd"
mkdir -p "$home" "$cwd"
fake_opener "$work/opener" "$work/opened.txt"

before_stage="$(listing "$stage")"
before_cwd="$(listing "$cwd")"

run_release() {
    ( cd "$cwd" && HOME="$home" GDOC_DESKTOP_OPEN="$work/opener" \
        bash "$stage/install.sh" --desktop ) > "$work/release-$1.log" 2>&1 ||
        fail "the release installer exited non-zero on run $1: $(cat "$work/release-$1.log")"
}

run_release 1

installed="$home/.local/bin/gdoc"
mcpb="$home/.local/bin/gdoc.mcpb"
if [ ! -f "$mcpb" ]; then
    fail "the release installer wrote no $mcpb"
else
    ok "the release installer wrote gdoc.mcpb beside the binary"
    check_manifest "the release installer" "$(manifest_from_mcpb "$mcpb" "$work/ext1")" "$installed" "9.9.9"
fi

if [ "$(listing "$stage")" != "$before_stage" ]; then
    fail "the release installer wrote inside the unpacked zip"
elif [ "$(listing "$cwd")" != "$before_cwd" ]; then
    fail "the release installer wrote inside the working directory"
else
    ok "nothing was written outside the fake HOME"
fi

if [ "$mac" -eq 1 ]; then
    if [ ! -f "$work/opened.txt" ]; then
        fail "the opener was never called on macOS"
    elif [ "$(cat "$work/opened.txt")" != "$mcpb" ]; then
        fail "the opener was handed [$(cat "$work/opened.txt")], want exactly $mcpb"
    else
        ok "the opener was handed the one file and nothing else"
    fi
else
    if [ -f "$work/opened.txt" ]; then
        fail "the opener was called off macOS"
    elif ! grep -q "$mcpb" "$work/release-1.log"; then
        fail "off macOS the run does not say where the extension was written: $(cat "$work/release-1.log")"
    else
        ok "off macOS the file is written, the opener is not called, and the run says so"
    fi
fi

# The second run, which is what an upgrade is. The same extension, so a person
# who re-runs the installer does not get a different connector.
run_release 2
if [ -f "$mcpb" ]; then
    second="$(manifest_from_mcpb "$mcpb" "$work/ext2")"
    if ! diff -q "$work/ext1/manifest.json" "$second" >/dev/null 2>&1; then
        fail "a second run gives a different manifest: $(diff "$work/ext1/manifest.json" "$second" || true)"
    else
        ok "a second run gives the same manifest"
    fi
fi

# A release that carries no template refuses --desktop, and refuses it before
# anything on the machine is replaced.
norun="$work/notemplate"
mkdir -p "$norun"
cp "$stage/install.sh" "$norun/install.sh"
fake_gdoc "$norun/gdoc"
home2="$work/home2"
mkdir -p "$home2"
if ( cd "$cwd" && HOME="$home2" GDOC_DESKTOP_OPEN="$work/opener" \
        bash "$norun/install.sh" --desktop ) > "$work/notemplate.log" 2>&1; then
    fail "a release with no mcpb/manifest.json installed anyway"
elif [ -e "$home2/.local/bin/gdoc" ]; then
    fail "a release with no mcpb/manifest.json replaced the binary before it refused"
else
    ok "a release with no template refuses --desktop and installs nothing"
fi

# --------------------------------------------------------------------------
# The root installer, from a checkout
# --------------------------------------------------------------------------
#
# A copy of what the root install.sh reads: itself, the template, the five skill
# folders and a built binary. A fake make is put ahead of the real one on PATH,
# because this test is about the extension and not about compiling Go.

copy="$work/checkout"
mkdir -p "$copy/bin" "$copy/release/mcpb" "$copy/skills" "$work/path"
cp "$repo/install.sh" "$copy/install.sh"
cp "$repo/release/mcpb/manifest.json" "$copy/release/mcpb/manifest.json"
for skill in gdoc-align gdoc-export gdoc-publish gdoc-restyle gdoc-review; do
    mkdir -p "$copy/skills/$skill"
done
fake_gdoc "$copy/bin/gdoc"
cat > "$work/path/make" <<'MAKE'
#!/bin/sh
exit 0
MAKE
chmod 755 "$work/path/make"

home3="$work/home3"
mkdir -p "$home3"
rm -f "$work/opened3.txt"
fake_opener "$work/opener3" "$work/opened3.txt"

run_root() {
    ( cd "$copy" && HOME="$home3" PATH="$work/path:$PATH" \
        GDOC_DESKTOP_OPEN="$work/opener3" \
        bash "$copy/install.sh" --desktop ) > "$work/root-$1.log" 2>&1 ||
        fail "the root installer exited non-zero on run $1: $(cat "$work/root-$1.log")"
}

run_root 1

root_mcpb="$copy/bin/gdoc.mcpb"
if [ ! -f "$root_mcpb" ]; then
    fail "the root installer wrote no $root_mcpb"
else
    ok "the root installer wrote gdoc.mcpb beside bin/gdoc"
    # The checkout's own binary, so make build is the whole upgrade, and the
    # version a checkout has no release number for.
    check_manifest "the root installer" "$(manifest_from_mcpb "$root_mcpb" "$work/root-ext1")" "$copy/bin/gdoc" "0.0.0-dev"
fi
if [ -e "$home3/.local/bin/gdoc.mcpb" ]; then
    fail "the root installer wrote an extension into ~/.local/bin, where the release installer's one lives"
fi

if [ "$mac" -eq 1 ]; then
    if [ "$(cat "$work/opened3.txt" 2>/dev/null)" != "$root_mcpb" ]; then
        fail "the root installer handed the opener [$(cat "$work/opened3.txt" 2>/dev/null)], want $root_mcpb"
    else
        ok "the root installer opened the file it wrote"
    fi
elif [ -f "$work/opened3.txt" ]; then
    fail "the root installer called the opener off macOS"
else
    ok "off macOS the root installer writes the file and calls nothing"
fi

run_root 2
if [ -f "$root_mcpb" ]; then
    root_second="$(manifest_from_mcpb "$root_mcpb" "$work/root-ext2")"
    if ! diff -q "$work/root-ext1/manifest.json" "$root_second" >/dev/null 2>&1; then
        fail "a second root run gives a different manifest"
    else
        ok "a second root run gives the same manifest"
    fi
fi

# A run with no --desktop writes no extension at all.
home4="$work/home4"
mkdir -p "$home4"
rm -f "$copy/bin/gdoc.mcpb"
( cd "$copy" && HOME="$home4" PATH="$work/path:$PATH" \
    bash "$copy/install.sh" ) > "$work/root-plain.log" 2>&1 ||
    fail "the root installer exited non-zero with no flags: $(cat "$work/root-plain.log")"
if [ -e "$copy/bin/gdoc.mcpb" ]; then
    fail "a run with no --desktop wrote an extension"
else
    ok "a run with no --desktop writes no extension"
fi

printf '\n'
if [ "$failures" -ne 0 ]; then
    printf 'test-desktop: %d check(s) failed\n' "$failures" >&2
    exit 1
fi
printf 'test-desktop: every check passed\n'
