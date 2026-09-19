# gdoc v2 plan

What is left to build. [SPEC.md](SPEC.md) says what v2 is today, each package's
`doc.go` says why that package refuses what it refuses, and
[DECISIONS.md](DECISIONS.md) says why, dated. This file holds the open work and
nothing else: a milestone that is done is one row in the table below and its own
plan file under `docs/plans/completed/`.

A milestone gets its task-by-task plan in `docs/plans/` when it starts, so each
one is written against the code that exists by then.

## Standing facts

- Go 1.27. The module is `gdoc`, at `go/`.
- Three dependencies, `beevik/etree`, `yuin/goldmark` and `goccy/go-yaml`, each
  with its reason in SPEC.md and its line in `allowedModules`. A fourth needs
  its reason written into SPEC.md first. The one candidate, `sergi/go-diff`,
  is deferred with M8 to the backlog, 2026-09-16.
- Platforms: darwin/arm64, darwin/amd64, windows/amd64. No linux. Every target
  cross-builds on every commit, so portability is never discovered late. A
  release carries a zip per line of `release/platforms`, which is the two darwin
  ones until a colleague has run the Windows checklist.
- Versions are `x.y.z` and they come from the tag. `x.y.0` is stable and Nail
  cuts it with `make tag`; `x.y.(z+1)` is nightly and CI cuts it when main has
  moved. `.claude-plugin/plugin.json` carries the last stable number, written by
  `make tag` alone: Claude Code delivers a plugin when that string changes, so
  the nightly leaves it where it is and there is no nightly channel for skills.
  2026-09-18, DECISIONS.md.
- One template. Everything is measured against `altery-group-policy-v1.0`, and a
  second one is a decision nobody has taken.

## Done

| Milestone | What it delivered | Done | Plan |
|---|---|---|---|
| M1 | the binary, the output envelope, the per-platform config paths, the guard as the whole network policy before any command could reach Drive, and `auth login` and `auth status` over a token file written crash-safely | 2026-08-29 | `2026-08-29-gdoc-v2-m1-foundation.md` |
| M2 | the three reads, `read`, `comments` and `suggestions`, the opaque `--since` cursor, the docx export as the honest witness, and the versioned `gdoc:` front-matter schema | 2026-09-06 | `2026-09-06-gdoc-v2-m2-reading.md` |
| M3 | the four writes, `probe`, `reply`, `propose` and `withdraw`, each one verified by a route it did not go out on, and the review skill written over them | 2026-09-07 | `2026-09-07-gdoc-v2-m3-writing.md` |
| M4 | `comments --wait`, one call that polls, so a session can stay live on one document for as long as the review lasts | 2026-09-07 | `2026-09-07-gdoc-v2-m4-live-session.md` |
| M5 | `build`, the house-style docx written part by part out of `house.yaml`, and the offline drift gate over 169 measured items | 2026-09-08 | `2026-09-08-gdoc-v2-m5-generator-parity.md` |
| M6 | `publish`, the upload with conversion into one folder, three read-backs, the pairing written into the note or the document trashed, and the live drift gate | 2026-09-08 | `2026-09-08-gdoc-v2-m6-publish.md` |
| M7 | `restyle --dry-run`, the survey of what a document holds before anything is done to it, and the seven paragraph elements the decoder used to drop in silence | 2026-09-09 | `2026-09-09-gdoc-v2-m7-survey.md` |
| M7b | `restyle --from`, the house style given to a handed-in document where it stands, under a grant that lasts one run and carries four request kinds, none of which can change a character | 2026-09-09 | `2026-09-09-gdoc-v2-m7b-restyle-in-place.md` |
| M7c | `restyle --fields`, the cover, the three front-matter tables and the legend proposed as suggestions on a policy that granted nothing, marked by one named range | 2026-09-10 | `2026-09-10-gdoc-v2-m7c-house-template-as-suggestion.md` |
| M7d | one command table as the only description of a command, read by the dispatcher, the usage line, `help` and `completion`, `--help` and `-h` anywhere on the line, the zsh and bash completion scripts the binary writes, and the `gdoc-publish` and `gdoc-restyle` skills, each learning its flags from `gdoc help` | 2026-09-16 | `2026-09-16-gdoc-v2-m7d-help-completion-skills.md` |
| M9 | the version `x.y.z` from the tag in every envelope, a tag that builds and publishes a zip per platform as a GitHub Release, the one-line install, the skills as a Claude Code plugin from a marketplace in this repository, the nightly that tags `x.y.(z+1)` when main moved, and `gdoc update`, which runs only when a person types it and verifies what it downloads through the fifth guard grant | 2026-09-16 | `2026-09-16-gdoc-v2-m9-release.md` |
| M10 | the release notice: a stamp file beside the token, `help` refreshing it once a day under a two-second ceiling and printing the facts, the skills mentioning a newer gdoc once and carrying on, and the nightly leaving `plugin.json` on the stable number | 2026-09-18 | `2026-09-18-gdoc-v2-m10-update-notice.md` |
| M11 | `annotate`, a fifth writer: a comment on the exact words a colleague quotes, anchored, under the robot prefix, changing nothing, taking `--quote` with `--body-file` or `--from` a file of many, verified by two routes the write did not go out on, and no probe, because a batch holding one `insertComment` cannot move a character | 2026-09-18 | `2026-09-18-gdoc-v2-m11-annotate.md` |
| M12 | five backlog items: `GrantInPlace` leaves a created document at full and says the grant changed nothing, a styling batch whose answer names no revision stops the run instead of reading the document for one, the offline drift gate pins the measured pair of every known difference, every numbered list gets its own definition and starts at 1, and an internal anchor link jumps to a bookmark every heading with words carries | 2026-09-18 | `2026-09-18-gdoc-v2-m12-five-backlog-items.md` |
| the docs restructure | CLAUDE.md cut to the invariants under a test that holds its size, every package's essay moved into its own `doc.go`, SPEC.md and PLAN.md in the present tense, a status register over DECISIONS.md, MEASURED.md split out of it, and v1 retired | 2026-09-15 | `2026-09-11-gdoc-v2-docs-restructure.md` |

## M8. Deferred

Alignment, the align skill, the diff question and the two things folded into
M8 are deferred whole to
[the backlog](../backlog/m8-alignment-and-the-align-skill.md), Nail's decision
of 2026-09-16, DECISIONS.md. The team's feedback after the release decides
whether it comes back.

## Outstanding by hand

Three checks nobody can automate, each one Nail's.

- **A built document read by a person.** A house-style docx opened in Word and
  in Drive, against the master. The offline and live drift gates measure 169
  values between them and neither of them looks at the page.
- **The live drift table.** `TestLiveDrift` prints its rows and no person has
  read them yet, so no row has joined `drift.Known` for a reason the live gate
  found. A row joining `Known` is a decision written down with its reason, never
  a test somebody loosens.
- **A second account in the margin.** The colleague `ai!` path is exercised by
  the skill's own rules and its wording. The first real proof is a session with
  somebody else commenting.

`docs/v2/MEASURED.md` holds a fourth under "Not measured yet": the seven
paragraph elements are a fixture built from the API reference, and a real
document holding a person chip, a date chip and a calendar link has not been
read against it.
