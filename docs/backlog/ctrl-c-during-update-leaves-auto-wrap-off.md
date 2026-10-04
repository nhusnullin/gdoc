---
worth: yes
where: go/cmd/gdoc/progress.go:324, go/cmd/gdoc/progress.go:331
added: 2026-10-04
---
# Ctrl-C during `gdoc update` leaves the terminal's auto-wrap off

On a terminal, `gdoc update` turns the terminal's auto-wrap off on its first
draw (`tty.WrapOff`, progress.go:324) and back on only in the last one
(`tty.WrapOn`, progress.go:331). The redraw counts lines, and a line that
wrapped would be two rows on screen and one in the count, so wrap is off while
the list moves.

`runUpdate` traps no signal. So Ctrl-C during the download kills the process
between those two writes, and the shell it ran in stops wrapping long lines
until the person opens a new tab or runs `tput smam`. Nothing says why.

This is older than M15: the wrap-off came with the update's live progress. The
M15 review found it on 2026-10-04 and Nail put it here at acceptance, rather
than in that release.

## What to decide

Both fixes break something written down, which is why this is not a refactor:

- **Trap the signal in `update`**, turn wrap back on, then exit. That breaks
  `TestOnlyTheWaitTrapsTheSignal` (main_test.go:323) and the decision that only
  the wait in `comments` traps a signal, so every other command dies where it
  stands on Ctrl-C.
- **Turn wrap back on at the end of every draw**, not once at the end. A killed
  process then leaves wrap on, because the terminal only reads the state while
  lines are being written. That breaks `TestALiveListTurnsWrapOffAndBackOn`
  (progress_test.go:196), which pins one off and one on. It is the cheaper of
  the two and looks the same on screen.

The second keeps the signal rule intact. It needs the test rewritten to pin
"wrap is on whenever the process is between draws", and a line in
`cmd/gdoc/doc.go` saying why.
