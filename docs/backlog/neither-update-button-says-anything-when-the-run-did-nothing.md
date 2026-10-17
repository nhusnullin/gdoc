---
worth: soon
where: presence/Sources/PresenceApp/UpdateAction.swift:73, :133
added: 2026-10-17
---
# Neither update button says anything when the run did nothing

`UpdateAction.installExtension` hands back one Bool, `extension_opened`, and
drops everything else the child said: the thrown error, `envelope.error` and
the warnings. The row the button sits on is then drawn from
`extensionState()`, so a run that did nothing leaves the row and the button
exactly where they were, which is correct, and says nothing about why.

Three runs write nothing and say why only in a warning or in the error:

- A newer major version is listed. `refreshHere` in `go/cmd/gdoc/update.go`
  answers `ok: true` with the warning "--desktop writes the extension of the
  gdoc that is installed, and this run installed nothing". The button can
  never work on that machine until gdoc is updated with `--major`, and the
  panel gives no reason.
- `binaryPath` refuses a symlinked checkout binary with `ok: false`.
- A release that cannot be read answers `ok: false` with its stop.

What the panel does today is by the spec: the Claude Desktop row is drawn
"where Claude Desktop can have no extension", and a run that handed nothing
over leaves it there. What is missing is the word about it, and that is a
decision about the panel's words, not a refactor:

- The panel shows no child's error text anywhere. Every sentence on a row is
  the app's own, as `NoteFault.tip` is. So the fix is probably a fixed
  sentence per case and not `envelope.error` on the screen.
- It needs a field to live in. `Facts.extensionState` carries no reason, and
  a quiet line on the row is the shape the account row already uses.

## Update is the same silence

`UpdateAction.ran` already reads the whole answer and keeps the sentence:
`.failed` carries the envelope's error or one of `UpdateWords`' two lines,
and `.unchanged` carries the word the command printed. Nothing reads either
case. `applied(to:)` takes a version out of `.installed` alone, so every
other standing leaves the gdoc row exactly as `help --json` drew it.

Press Update with no network, or on a machine where the only newer release
is a major one, and the ring runs, the row goes back to
"v2.14.0 · v2.15.0 available", and a second press does the same. The major
case is the one the first half of this item names: a button that can never
succeed and never explains.

The two halves want the same field, which is why they are one item: a quiet
line on the row the button sits on, from a fixed sentence per case. Until
there is one, `UpdateStanding.failed`, `.unchanged` and `UpdateWords` are
kept rather than deleted, because the reading that fills them is the part
that would be expensive to write twice.

Worth `soon` because the first case is a button that can never work, and the
person pressing it is told nothing at all.
