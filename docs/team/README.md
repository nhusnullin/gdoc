# The team

Nail decides what. The team lead decides how. Nail talks to the lead and to
nobody else.

Until 2026-11-28 a milestone was a plan run by ralphex and reviewed by revmux,
behind `make milestone`. Nail's decision of that date replaced it with the way
Etude works: Backlog.md, one task in one branch in one PR, and the Compound
Engineering loop per task. `docs/v2/DECISIONS.md` holds the entry, and
`docs/design/2026-10-19-two-days-retrospective.md` holds what the old way cost.

## The seats

| Seat | Model | Holds |
|---|---|---|
| **Team lead** | the session Nail opens | Nail's one interface. The backlog, the five product questions, the decision entry, the spec, the task, the PR, the release go, the summary. The product owner is a role inside this seat |
| **Trust guardian with the skeptic** (`gdoc-trust-guardian`) | Opus | `go/internal/guard`, `go/internal/auth`, `go/internal/presence`, `go/cmd/gdoc/desktop.go` and every invariant in `CLAUDE.md`. Read only: it signs a decision entry or names its veto and what would lift it |
| **Design reviewer** (`gdoc-design-reviewer`) | Sonnet | once, after a UI task's renders are made and before the PR, over 20 dark whole-panel renders and the captions as text. Read only: it never fixes and never draws |
| **Codex second opinion** (`codex-second-opinion`) | as defined | only on a contested decision |

The guardian must be consulted before work starts on a task that touches its
area. A veto is not an argument the lead wins. It goes to Nail as a decision.

## The day

1. Nail's sentences collect at the lead.
2. The lead sends the five product questions in one message, every question
   answerable with a word, or acts when the sentence already answers them.
3. Decision entry in `docs/v2/DECISIONS.md`, then the spec change. The trust
   guardian first if the task touches the trust surface.
4. A task in Backlog.md, a branch, the work with its tests, a PR.
5. For a UI task, the renders once, then one design reviewer over 20 images.
6. `make release BRANCH=...` to a green PR, and no further. The merge, the tag
   and the zip check need `GO=vX.Y.0`, which is Nail's word.
7. A summary for Nail, and a short checklist of what to look at on his machine.

## The pages

- `docs/team/lead.md`: the lead's own page, which the lead session loads at
  start.
- `backlog/`: the tasks. `backlog browser` shows the board.
- `scripts/release.sh`: the release recipe behind `make release`.
