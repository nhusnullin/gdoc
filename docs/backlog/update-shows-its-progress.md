---
worth: yes
where: go/cmd/gdoc/update.go:209
added: 2026-09-19
---
# `gdoc update` shows its progress in the terminal, for a person

Nail, 2026-09-19: `gdoc update` is the one command a person types for
themselves rather than a skill for them, and it should show what it is doing
while it does it, in a modern, informative and good-looking way. Today it is
silent from the keypress until the one JSON object lands, and on a slow
network that is a checksum read, a download of megabytes and a file swap with
nothing on screen.

The steps are already there to narrate, in `runUpdate` and `install`: read
the listing, choose the release for this platform and channel, decide, fetch
the checksums, fetch the zip, verify, replace, read the file back. Each is a
line a person would want to see, with the tag, the size and the path.

Three rules shape the answer, and none of them is negotiable without a
DECISIONS.md entry:

- Human words go to stderr, and stdout stays one JSON object. That is the
  envelope invariant, `TestOnlyJSONObjectRefusesAnythingAfterTheObject`, and
  `auth login` and `help` already print for a person on stderr. So the
  progress is stderr, and the object at the end is unchanged.
- The module list is three. A terminal UI library is a fourth, so the
  progress is written by hand: carriage-return lines, a spinner or a bar,
  and colour only when stderr is a terminal. Detecting a terminal is
  `os.Stderr.Stat()` and the char device bit, no library needed, and a run
  whose stderr is a pipe prints plain lines or nothing.
- `GetBytes` in `internal/gapi/plain.go` reads the whole body before it
  returns, so there is no byte count to show while the zip comes down. A
  byte-level bar needs a reader that reports as it goes, which is a change
  in the one room that builds a request, and the Content-Length GitHub sends
  on the redirect target. A step-level narration needs none of that and is
  most of the value.

The question the design has to answer is whether the progress is a stderr
concern of `cmd/gdoc/update.go` alone, or a small writer in `internal/emit`
beside the envelope, so that `auth login` and the daily notice print through
the same thing. The second is the tidier shape and the bigger change.
