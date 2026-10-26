---
worth: yes
added: 2026-10-24
revisit: 2026-11-24
---
# When a review needs revmux's comprehensive profile

Since 2026-10-24 every milestone reviews the same way: one revmux round under
the `focused` profile, one bugs agent and the codex peer, and ralphex's own
claude loop after it held at three passes by `--max-iterations=10`. Nail chose
that after M28's meter: the review cost 106M of the run's 130M, and 100M of it
was ralphex's claude loop running five passes at the default cap. revmux's own
three lenses added about 19M the meter does not read.

Nail asked to come back to this in a month, on or after 2026-11-24, and decide
when a review should still run `comprehensive` (three claude lens pairs and the
codex peer, patience two, three rounds).

What to bring to that conversation:

- The meter rows in `docs/team/measure.md` for every run since M28, with the
  review cost split from the task cost.
- Every finding a focused review missed that something later caught: the
  design reviewer, CI, a revert, or Nail on his machine. That list is the case
  for the bigger review, and an empty list is the case against it.
- A candidate rule to argue about: comprehensive only when a plan touches the
  trust boundary (`go/internal/guard`, `go/internal/auth`,
  `go/internal/presence`, `go/cmd/gdoc/desktop.go` or an invariant in
  `CLAUDE.md`), the same list that calls the trust guardian.

The knob is `REVIEW_PROFILE` at the top of `.ralphex/board/milestone.sh`.
