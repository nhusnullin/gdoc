---
worth: yes
where: go/cmd/gdoc/update.go, release/install.sh:309
added: 2026-09-21
---
# `gdoc update` leaves the completion stale, so a new command never completes

M13 added `export`. The binary on PATH has it: `~/.local/bin/gdoc` reports
`v2.3.5`, and its own unknown-command error lists `export` among the sixteen
names. Typing `gdoc export <url>` in full works. Typing `gdoc ex` and pressing
tab offers nothing, because the shell is not reading that binary.

What the shell reads is `~/.config/gdoc-agent/completion.zsh`, sourced from
`.zshrc`. That file is dated 18 September, two days before M13 merged, and its
header comment lists fourteen commands with no `export` among them. The only
time the word appears in it is the description of `--witness`.

The completion is a snapshot. `install.sh` renders it into `bin/gdoc.zsh` for a
checkout, `release/install.sh` renders it into `$CONFIG_DIR/completion.zsh` for
a colleague, and both do it once, with `gdoc completion zsh --out ... --force`.
`gdoc update` replaces the binary and touches nothing else: neither
`go/cmd/gdoc/update.go` nor anything under `go/internal/update/` mentions the
completion. So the list a person tabs through is whatever the binary was on the
day they installed, and the one command that moves them forward never refreshes
it.

It is not only `export`. Every command and flag added since the snapshot is
missing, and a command removed would still be offered. Nothing says any of this
out loud, so the symptom reads as the new command not existing rather than as a
file being old.

## What to decide

The write already exists, `completion zsh --out <path> --force`. The question
is who calls it and where it writes.

`update` cannot just pick a path. A colleague's file sits beside the token,
which `internal/config` knows, but a checkout's sits at `bin/gdoc.zsh` in a
repository the binary must not assume exists. Refreshing one and not the other
makes the two installs drift in different directions.

Three shapes, none chosen:

- `update` re-renders the file beside the token when that file is already
  there, and says a new shell is needed. Leaves the checkout to `install.sh`,
  which is the route that already rebuilds.
- The completion stops being a snapshot. Install writes a small stub that runs
  `gdoc completion zsh` at shell start, so the list follows the binary with no
  refresh step at all. The cost is one process per shell start. It does not
  touch the invariant that nothing under `go/` runs an external program, since
  here the shell runs gdoc.
- `update` changes no file and prints the command to re-run. Cheapest, and it
  puts the work back on the person every time.

Whichever wins, the file is sourced at shell start, so a refresh only reaches a
shell opened afterwards. The run should say that rather than leave somebody
tabbing in the shell they already have.

Nail picks.
