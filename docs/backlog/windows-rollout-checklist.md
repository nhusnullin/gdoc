---
worth: later
where: release/platforms, .github/workflows/release.yml
added: 2026-09-16
---
# Windows is built, packed on demand, and offered to nobody until this is run

`make dist` builds `gdoc-windows-amd64.exe` and `release.yml` knows how to pack
it: it names the file inside the zip `gdoc.exe` rather than `gdoc`. What has
never happened is a gdoc on Windows replacing a running `gdoc.exe` with a newer
one. So `release/platforms` lists the two darwin pairs and nothing else, and a
Windows machine asking for an update is refused by `Choose` before it reaches
any of the code below.

This file is the checklist that line waits on. `release/platforms` and the M9
plan both point here.

## Why it is not just a line in `release/platforms`

Windows will not let a running program's file be replaced the way POSIX does.
`os.Rename` over a file that is open for execution fails with a sharing
violation, so `internal/update`'s move is the part that is unmeasured, not the
download and not the checksum. The usual answer is to rename the running binary
out of the way first and then move the new one in, which is close to what
`Apply` already does for the rollback copy, but "close to" is not measured.

## The checklist

Steps 1 to 5 run against a zip packed by hand. Steps 6 to 8 cannot: they need a
published release that carries a windows zip, because `release.yml` packs only
what `release/platforms` lists and `Choose` refuses a platform no asset names.
So the line goes in first, on a branch, and step 7 is what decides whether it
stays.

0. On a branch, add `windows-amd64` to `release/platforms` and cut a release
   from it, a pre-release is fine. That is the only way steps 6 to 8 have a
   `gdoc-<tag>-windows-amd64.zip` to find. Without it step 6 stops with
   "release <tag> carries no gdoc-<tag>-windows-amd64.zip".
1. `make dist` and pack a `gdoc-<tag>-windows-amd64.zip` by hand, the way
   `release.yml` would: the binary in at the top as `gdoc.exe`, plus
   `install.sh`, `README.md`, `example/` and `skills/`. This is the zip steps 2
   to 5 run against, so they need no release at all.
2. Unpack it and put `gdoc.exe` somewhere on PATH. `install.sh` is bash and is
   not part of this checklist: say in the result whether a Windows colleague is
   expected to use Git Bash or whether the installer needs a second entrance.
3. `gdoc help` reports the tag it was built from.
4. `gdoc auth login` prints a URL, the browser lands on the loopback listener,
   and `gdoc auth status` reports the token.
5. `gdoc read` against a document, to prove the wire and the config dir.
6. `gdoc update --check` against a release newer than the one installed, and
   read the object: it has to name the windows zip, not a darwin one.
7. `gdoc update` from a terminal where that same `gdoc.exe` is the running
   process. This is the step the whole file exists for. Record what happens,
   the exact error if it fails, and whether the binary that was there is still
   runnable afterwards.
8. `gdoc update --rollback`, and run the restored binary.
9. Config dir and completion: `internal/config` picks a Windows path, and the
   completion file is written for a shell Windows may not have. Say what a
   colleague actually gets.

## What comes out of it

Either step 7 passes, the `windows-amd64` line step 0 put in `release/platforms`
stays and merges with the result written into `docs/v2/MEASURED.md`, or step 7
fails, that line comes back out, and the fix to `internal/update`'s move comes
first with its own decision in `docs/v2/DECISIONS.md`.
