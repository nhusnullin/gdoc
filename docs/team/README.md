# The team

Two days of the presence app shipped what Nail asked for and ate a week of
tokens. `docs/design/2026-10-19-two-days-retrospective.md` counted where they
went: about 107 Opus task iterations, ten revmux rounds, some twenty reviewer
agents reading up to 120 images each, and one project-manager session that held
product, runs, monitors, boards, releases and reports in one context. M26 alone
cost 220 million tokens over its 27 worker sessions, 214 million of them cache
reads, which is the same context sent again on every turn. `docs/team/measure.md`
read that worktree again with the meter, the subagents included, and counted
237.8 million.

This page is the team that replaced that session, organised by Team Topologies.
Nail decides what. The team lead decides how. Nail talks to the lead and to
nobody else.

## The four teams

### Stream-aligned: "gdoc"

The team that ships. It owns the product end to end, from Nail's sentence to
the released zip.

| Seat | Model | Holds |
|---|---|---|
| **Team lead** | Opus | Nail's one interface. The day's backlog, the five product questions before a plan, the decision entry, the spec, the plan, the run, the release go, the summary. The product owner is a role inside this seat, not a second seat |
| **Go engineer** | Opus | the binary: a command, a guard, a package under `go/internal/` |
| **macOS engineer** | Opus | the Swift app under `presence/`: the panel, the bundle, signing, launch at login |
| **Skills author** | Sonnet | the five folders under `skills/`, the guide pages, the docs tasks and the fixtures |

The three engineers are not sessions anybody talks to. They are ralphex workers
with a short area brief, which the plan names task by task ("area: presence").
Cut 8 of the retrospective is why the brief is short: a worker on a `presence/`
task needs the app's two invariants, not the guard's.

### Platform: "delivery"

The thinnest viable platform, which here means scripts and not a service.

| Seat | Model | Holds |
|---|---|---|
| **The two scripts** | none | `.ralphex/board/milestone.sh` and `.ralphex/board/release.sh`, behind `make milestone` and `make release`. The worktree, the branch, the board folder, ralphex, revmux by review size, the board every 30 minutes, the PR, the checks, the tag, the published zip |
| **Platform operator** (`gdoc-platform-operator`) | Sonnet | wakes only when a recipe failed. Reads the status file and the tail of a log, applies the recipe from `.ralphex/board/README.md`, reports three lines. Never pushes, merges or tags |

The scripts carry the recipes that have one: an expired Claude CLI sign-in stops
the run and writes `needs /login`, and a task phase that exited with tasks left
is relaunched once. A stalled worker is the idle timeout's job. Everything else
is the operator's, and the operator is cheap because it reads three lines before
it reads anything else.

### Enabling: "quality", facilitating

They find. The stream team fixes. Two fixed points in a day and no standing
presence.

| Seat | Model | When |
|---|---|---|
| **Plan reviewer** (`planning:plan-review`) | as defined | once, before the run, over the plan file |
| **Design reviewer** (`gdoc-design-reviewer`) | Sonnet | once, after the run and before the PR, over 18 dark whole-panel renders and the captions as text. Read only: it never fixes and never draws |
| **Codex second opinion** (`codex-second-opinion`, gpt-5.5) | as defined | only on a contested decision, and as the adversarial lens inside revmux |

Cut 3 is why there is one design reviewer and not five, and why it reads 18
images rather than 120. Its checklist is "Review before showing Nail" in
`docs/design/2026-10-17-presence-design-principles.md`.

### Complicated subsystem: "trust boundary"

| Seat | Model | Holds |
|---|---|---|
| **Trust guardian with the skeptic** (`gdoc-trust-guardian`) | Opus | `go/internal/guard`, `go/internal/auth`, `go/internal/presence`, `go/cmd/gdoc/desktop.go` and every invariant in `CLAUDE.md`. Read only: it signs a decision entry or names its veto and what would lift it |

It must be consulted before a plan runs when that plan touches any of those. A
veto is not an argument the lead wins. It goes to Nail as a decision.

## The interaction modes

- **Nail and the lead: collaboration.** Sentences, any time, in any order. They
  collect into the day's backlog. A real bug takes the fast lane and jumps the
  backlog.
- **Stream and platform: X-as-a-Service.** The lead starts a run and goes away.
  The interface is `~/gdoc-<mNN>-board/status`, three lines: the phase, the last
  commit, the next action. The lead reads that file and never the log. A line
  that starts with `wait:` means nobody has work.
- **Stream and quality: facilitating.** Two fixed points, before the run and
  after it. A reviewer is never a monitor.
- **Stream and trust: collaboration while the design is open**, then a
  signature on the decision entry. After the signature it is read only again.

## The day

1. Nail's sentences collect at the lead.
2. The lead sends the five product questions in one message. One message, every
   question answerable with a word.
3. Decision entry in `docs/v2/DECISIONS.md`, then the spec change, then the
   plan under `docs/plans/`. The trust guardian first if the plan touches the
   trust surface, then the plan reviewer.
4. `make milestone PLAN=... REVIEW=go` or `REVIEW=ui`. The lead reads the status
   file.
5. For a UI milestone, the renders once, then one design reviewer over 18
   images. Findings go to the lead, who decides fix now or tomorrow.
6. `make release BRANCH=...` to a green PR, and no further. The merge, the tag
   and the zip check need `GO=vX.Y.0`, which is Nail's word.
7. A summary for Nail, and a short checklist of what to look at on his machine.
8. His reaction becomes tomorrow's backlog.

One milestone and one review a day. Cut 1: a review cycle is a day's unit, not
a sentence's.

## The pages

- `docs/team/lead.md`: the lead's own page, which the lead session loads at
  start.
- `docs/team/measure.md`: what a run cost, how it is counted, and the baseline
  of M24a, M25, M26 and M27 the cuts are measured against.
- `.ralphex/board/README.md`: the platform's recipes, for when a script cannot.
- `docs/design/2026-10-19-two-days-retrospective.md`: what the cuts are numbered
  against.
