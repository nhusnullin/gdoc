# gdoc v2 Milestone 29: the archive. Plan

Tasks: 6
Review: go

2026-10-23. The pilot of the team M28 built. One task, one commit, on the
local branch. `make release` runs without `GO`, so the push and the PR wait
for Nail's word, and nothing is merged or tagged without it.

## Overview

This milestone is the two document items Nail approved on 2026-10-23, and
it is the first run driven by the lead playbook in `docs/team/lead.md`
instead of by hand. The work is deliberately small and mechanical: the
point of the pilot is the measurement, and a milestone whose content is
contested would measure the argument rather than the team.

The two items are the documents every session and every worker iteration
pays for:

- **The index stays, the old entries move.** `docs/v2/DECISIONS.md` is
  5,506 lines, about 84k tokens, 86 entries and a 90 row register.
  `docs/v2/MEASURED.md` is 1,386 lines and 27 sections. Entries and
  measurements from before 2026-10-16, which is where the presence
  milestones start, move to `docs/v2/DECISIONS-archive.md` and
  `docs/v2/MEASURED-archive.md`. Neither file loses its index: the
  register stays whole in DECISIONS.md and gains a column naming the file
  that holds each entry, and MEASURED.md gains an index of the same shape,
  which it has never had. A citation then resolves in one hop: the reader
  opens the live file, reads the index, and the index says where.
- **The command's reasons stop being a walkthrough.** `go/cmd/gdoc/doc.go`
  is 1,403 lines in 30 sections, and it is the only comment `go doc` prints
  for the command. It keeps the package comment and every rule CLAUDE.md or
  another file sends a reader to: the arguments, the flags, the help, the
  envelope. The per command walkthroughs move to the file whose code they
  describe, as file comments, which is the convention `main.go`,
  `update.go` and `export.go` already follow.

No entry and no measurement loses a word: those moves are byte for byte.
Every citation still resolves, and the two in `docs/v2/SPEC.md` that name a
moved measurement by title are corrected, which the run's own decision
entry authorises and nothing else in SPEC.md is touched.

## Context (from discovery)

- `docs/v2/DECISIONS.md`: the register is at the top, 90 rows, `| Date |
  Decision | Status |`. 86 rows have their entry in this file as
  `## <date>. <title>.`; four have the status `MEASURED.md` and their text
  in that file under a heading of the same name. The entries are in strict
  ascending date order, so the split is one cut: 73 entries are dated
  before 2026-10-16, and the first that is not is line 4449,
  ``## 2026-10-16. `---` is a page break.``
- A register row's text and its entry heading are not the same string. The
  shorter is a prefix of the longer: the row "Anchors are not banked"
  against the heading "Anchors are not banked. *Rejected the day it was
  proposed.*", and the row about the colour scheme, which carries its
  supersession, against the shorter heading.
- `docs/v2/MEASURED.md` has no index and is not in date order. Twelve
  sections were measured before 2026-10-16; "Drive's import of a heading's
  bookmark" is 2026-10-16 and sits after four 2026-10-17 sections, and the
  live list "Not measured yet" sits mid-file with four sections after it.
  Most sections open `Measured <date>`, two open with the date alone, as in
  "2026-10-17, M24a Task 17."
- **MEASURED.md is cited by ordinal, not by date.** 33 comments under `go/`
  say "measurement 2", "MEASURED 3", "row 7" or "rows 5 and 6". Those
  numbers are rows of tables inside two sections, both of which move:
  "Claude Desktop and a local MCP server" (2026-10-03), cited by
  `go/cmd/gdoc/update.go`, `notice.go`, `mcpfiles.go`, `mcp.go`,
  `mcplogin.go`, four tests, `go/internal/chat/`, `go/internal/auth/doc.go`
  and `go/internal/mcp/`; and "A block of new paragraphs, proposed in one
  SUGGEST batch" (2026-09-27), cited eleven times under
  `go/internal/propose/`. Three more cite a section by title. This is why
  MEASURED.md gains an index: without one, a moved section has nothing that
  resolves it.
- Two lines of `docs/v2/SPEC.md` name a moved measurement by title, 969 and
  1094, and `docs/v2/PLAN.md:75` does the same. "Not measured yet", which
  stays, points at the moved table by number and says "the section above".
- Citations are never by anchor: `grep -rn "DECISIONS.md#"` and
  `"MEASURED.md#"` find nothing.
- `go/cmd/gdoc/doc.go` holds all 30 `// # ` headings in the command today;
  no other file there has one. `main.go` opens "The entry point, the
  dispatch table and the auth commands", and `authStatus` and `authLogin`
  are in it. `signal.NotifyContext` is in `read.go`.
- `go/boundary/docs_test.go` holds four guards over the docs, and
  `go/boundary/doc.go` describes them under "Four guards over the docs'
  shape". `TestEveryPackageHasExactlyOnePackageComment` and
  `TestOnlyThePackageCommentReachesGoDoc` are the two a move of prose into
  a command file can break: a file paragraph needs a blank line before the
  `package` clause or `go doc` joins it into the package comment.
- `go/boundary/probe_test.go` scans `probeClaimScanned` for any sentence
  saying a proposal runs the capability probe. `go/cmd/gdoc/doc.go` is on
  that list, sections that mention the probe move out of it, and
  `probeClaimFiles` already expands a named directory into every `.md` and
  `.go` under it.
- Before the run: the decision entry of the run's date is in DECISIONS.md
  with its register row, written by the lead from Nail's words, and this
  plan is committed on `main`, which is what `make milestone` checks.

## Development Approach

- **TDD.** Task 1 writes the guard before anything moves, and it passes
  over today's files, where every row's entry is in DECISIONS.md or
  MEASURED.md. Tasks 2 and 3 then move entries and keep it green. A guard
  written after the move proves nothing about the move.
- One commit per task. No push, PR, merge or tag without Nail's word.
- A moved entry or measurement keeps its words. Each move is checked
  mechanically, not by eye: the two halves put back together are the file
  that went in.
- `CLAUDE.md` is not edited. `docs/v2/SPEC.md` is edited in exactly two
  places, the lines that name a moved measurement by title, which the run's
  decision entry authorises; nothing else in it is touched.
- `make milestone` already strips `GDOC_LIVE_TEST` and `GDOC_LIVE_WRITE`
  from the worker, so a task says `make test`. The `env -u` form in Task 5
  is for the acceptance run by hand.
- `make test` keeps the race flag. Nothing here reaches Drive.

## Testing Strategy

- The new guards in `go/boundary/register_test.go`, run by `make test` like
  every other boundary test: the register and the measured index, both
  directions.
- The two `go doc` guards and the probe guard, already in `go/boundary/`,
  are the ones Task 4 must keep green.
- A completeness check in each moving task, in the task's own commands: the
  two files concatenated are the file that went in, and the 30 section
  headings of `doc.go` are still 30 across the command.
- `make vet` for the gofmt half of a file comment moved into a source file.

## Solution Overview

An index is what a citation resolves through, so each file gets one before
anything moves, and one guard reads both. The archives are frozen files
with a header that says so and points back at the index.

The command's package comment keeps the rules a reader is sent to and loses
the walkthroughs, which go to the file whose code they describe. The
destination is always an existing command file: every topic here has one.

## Technical Details

**The register's new column.** `| Date | Decision | Status | Entry |`.
`Entry` is one of `DECISIONS.md`, `DECISIONS-archive.md`, `MEASURED.md`,
`MEASURED-archive.md`, and nothing else. The four rows whose status reads
`MEASURED.md` today take the status `measured` instead, because that column
says what became of a decision and the new one says where it lives. The
paragraph under the register that lists the statuses says so, and the
paragraph that explains the `MEASURED.md` status becomes the paragraph that
explains the `Entry` column. The sentence "An entry is never edited after
this, except its status line" stays as it is.

**The measured index**, new, at the top of `docs/v2/MEASURED.md` under the
preamble: `| Measured | Section | Entry |`, one row per section, all 27, in
the order the sections stand in. The date is the one the section's body
opens with, in either of the two forms it takes. "Not measured yet" has a
row like the rest, with no date. Three sentences follow it:

- a measurement is never edited, and a different answer is a new section in
  this file rather than a change in the archive;
- a citation of the form "measurement N" or "MEASURED N" is a row of the
  numbered table in "Claude Desktop and a local MCP server", and one of the
  form "row N" is a row of the table in "A block of new paragraphs,
  proposed in one SUGGEST batch". The numbers inside those tables are
  unchanged by the move, and this index says which file each section is in;
- "Not measured yet" names the two sections it leans on, rather than saying
  "the section above", because the section above it may be in the archive.

**The guards**, `go/boundary/register_test.go`:

- `TestEveryRegisterRowNamesTheFileThatHoldsIt` reads the register, and for
  each row opens the file the `Entry` column names and finds the entry
  there: for the two decisions files a heading `## <date>. <title>`, for
  the two measured files a heading `## <title>`. It also checks the entry
  is in that file only, so a half-finished move that copied rather than cut
  fails.
- `TestEveryDecisionEntryHasItsRegisterRow` is the other direction over the
  two decisions files: every `## <date>.` heading has a row. It is green
  from the first commit, which is the point: it is what stops Task 2
  dropping an entry.
- `TestEveryMeasurementIsIndexed` is both directions over the measured
  index: every `## ` heading in MEASURED.md and in its archive has a row
  naming the file it is in, and every row has its section.
- Matching is by date, plus the shorter of the row's text and the heading's
  title being a prefix of the longer, both normalised: lowercased, trailing
  punctuation and emphasis dropped. No exception list: the rule holds over
  all 90 rows today, and a row that needs an exception is a row somebody
  reworded, which is what the guard is for.
- `go/boundary/doc.go`'s "Four guards over the docs' shape" becomes five,
  with the paragraph saying why an index is the thing guarded: it is what
  every citation in the tree resolves through, and it is the one part of
  either file that is read rather than searched.

**The two archives.** Each opens with the same four sentences: what it
holds, that its entries are frozen and keep the words they were written
with, that the index in the live file is the way in, and that a new entry
is never written here. Each archive keeps the order the sections had in the
file they came from.

**What stays in `go/cmd/gdoc/doc.go`:** the opening paragraphs, "One JSON
object, and the exit code says which", "The commands, and the one table
that describes them", "The binary never prompts, and help is an answer",
"On a terminal a help screen replaces the object", "A note on an entry is
what belongs to no flag", "Three setting flags, taken off the line first,
one of them experimental", "Argument parsing is strict", "A panic is still
one envelope" and "GDOC_CONFIG_DIR moves the config dir". That is the
arguments, the flags, the help and the envelope, which is what CLAUDE.md's
task map promises a reader who opens this file, and it is about 430 lines.

**What moves, and where:**

| Section | Moves to |
|---|---|
| the four `auth` sections: status is a report, status names the account, logout removes one file, login prints the URL | `main.go`, which says it holds the auth commands and does |
| Five facts of the record are filled in here | `main.go` |
| help marks is a topic, and the only one | `helpmarks.go` |
| Completion is a file, and the reason is the output contract | `completion.go` |
| The update runs when it is typed; help and mcp ask once a day | `update.go` |
| The update draws its steps on stderr | `update.go` |
| --desktop writes the Claude Desktop extension | `desktop.go` |
| The presence app is installed beside the binary, and opened | `desktop.go` |
| Only the wait traps a signal | `read.go`, which is where `signal.NotifyContext` is |
| The chat commands take the context they are handed | `mcptools.go` |
| A hold dies with the sign-in it was held under | `mcphold.go` |
| the four propose sections: the proposals file, the first unconfirmed read-back, the lost answer, the folder it ignores | `write.go` |
| the three export sections: export reads twice, the URL picks the entry, the note is read again | `export.go` |

The prose sits above the `package` clause with a blank line between, it
joins the paragraph the file already carries rather than starting a second
one, and no new package comment appears. Each moved section keeps its
`// # <title>` heading, so the 30 headings stay 30 and the count is the
completeness check.

Where the destination already says a sentence the moved section says, the
file's own sentence stays and the moved one goes, and the commit message
names what it dropped. That is the one place prose is edited rather than
moved, and it is why `doc.go`'s rules, which are cited elsewhere, are the
ones that stay put.

**The probe guard follows the prose.** `probeClaimScanned` replaces
`go/cmd/gdoc/doc.go` with the directory `go/cmd/gdoc`, which
`probeClaimFiles` expands into every `.md` and `.go` under it, so the scan
survives this move and the next one.
`TestNoDocumentSaysAProposalRunsTheProbe` and
`TestTheProbeClaimScanFindsAPlantedSentence` are the pins.

## What Goes Where

- **Implementation Steps**: the two indexes and their guard, the two moves,
  the command's prose, the acceptance run, the close.
- **Post-Completion**: the meter over this worktree and the row in
  `docs/team/measure.md`, then Nail's go for the push and the PR. Both are
  the lead's, not a worker's.

## Implementation Steps

### Task 1: the indexes name the file, and a guard reads them

**Files:**
- Create: `go/boundary/register_test.go`
- Modify: `docs/v2/DECISIONS.md` (the register and the two paragraphs under
  it), `docs/v2/MEASURED.md` (the new index), `go/boundary/doc.go`

- [x] test first: `TestEveryRegisterRowNamesTheFileThatHoldsIt`,
      `TestEveryDecisionEntryHasItsRegisterRow` and
      `TestEveryMeasurementIsIndexed` as in Technical Details. The first and
      the third are red before the columns exist; the second is green from
      the start, and it is what guards Task 2
- [x] the register gains the `Entry` column, every row saying
      `DECISIONS.md` or `MEASURED.md`, which is where every entry is today;
      the four measured rows take the status `measured`
- [x] the paragraphs under the register say what the statuses are now and
      what the `Entry` column is for; the never-edited sentence is untouched
- [x] MEASURED.md gains its index, 28 rows, every `Entry` saying
      `MEASURED.md` today, and the three sentences under it
- [x] `go/boundary/doc.go`: four guards over the docs become five, with the
      reason an index is guarded
- [x] `make test` green with not one entry moved yet

### Task 2: the decisions before 2026-10-16 move

**Files:**
- Create: `docs/v2/DECISIONS-archive.md`
- Modify: `docs/v2/DECISIONS.md`

- [x] the 73 entries before line 4449, ``## 2026-10-16. `---` is a page
      break.``, move to the archive in the order they stand, byte for byte,
      no word of any entry edited
- [x] the archive's header: what it holds, frozen, the index in
      DECISIONS.md is the way in, no new entry is written here
- [x] DECISIONS.md keeps its own header, the whole register and the 13
      entries from 2026-10-16 on
- [x] every moved row's `Entry` column says `DECISIONS-archive.md`
- [x] completeness: the archive's entry half and DECISIONS.md's entry half,
      concatenated in that order, are byte for byte the entry half of the
      file before the task, checked with `git show HEAD:docs/v2/DECISIONS.md`
      and `diff`
- [x] `make test` green

### Task 3: the measurements before 2026-10-16 move

**Files:**
- Create: `docs/v2/MEASURED-archive.md`
- Modify: `docs/v2/MEASURED.md`, `docs/v2/DECISIONS.md` (the measured rows),
  `docs/v2/SPEC.md` (two lines), `docs/v2/PLAN.md` (one line)

- [x] these twelve sections move, in the order they stand, byte for byte:
      Which styling requests land in place; In-place styling preserves
      anchors and pending suggestions; A named range over a suggested
      insertion; A table takes one index of its own at the end; A proposal
      into a table cell lands; A block of new paragraphs, proposed in one
      SUGGEST batch; Claude Desktop and a local MCP server; gdoc mcp in
      Claude Desktop, the first run; gdoc mcp in Claude Desktop, the red
      team and a clean install; Drive's import of a Word footnote; Drive's
      import of run shading and text colour; Footnotes and the marks,
      published by gdoc
- [x] the archive's header, as in Technical Details
- [x] the index's `Entry` column says `MEASURED-archive.md` for those
      twelve, and the four register rows whose entry moved say the same
- [x] "Not measured yet" names the two sections it leans on instead of
      "the section above", and keeps the table numbers it cites
- [x] the two lines of `docs/v2/SPEC.md` and the one of `docs/v2/PLAN.md`
      that name a moved measurement by title say which file holds it now;
      nothing else in either file changes
- [x] completeness: the two measured files concatenated are the file that
      went in, checked the same way as Task 2
- [x] `make test` green

### Task 4: the command's reasons keep the rules and lose the walkthrough

**Files:**
- Modify: `go/cmd/gdoc/doc.go`, the destination files of the table in
  Technical Details, `go/boundary/probe_test.go`,
  `go/cmd/gdoc/update_test.go`

- [x] the sections listed as staying stay in `doc.go`, unedited
- [x] every other section moves to the file the table names, as a file
      comment with a blank line before the `package` clause, joining the
      paragraph that file already carries, keeping its `// # ` heading
- [x] a comment that sends a reader to `doc.go` for prose that moved points
      at its new home: `update_test.go:558` and anything `grep -rn "doc.go"
      go/cmd/gdoc` finds like it
- [x] `probeClaimScanned` names the directory `go/cmd/gdoc` in place of
      `go/cmd/gdoc/doc.go`
- [x] completeness: `grep -c '^// # ' go/cmd/gdoc/*.go` sums to 30, and
      `doc.go` is under 500 lines
- [x] `go doc ./cmd/gdoc` prints the package comment and nothing from a
      moved section
- [x] `TestOnlyThePackageCommentReachesGoDoc`,
      `TestEveryPackageHasExactlyOnePackageComment`,
      `TestNoDocumentSaysAProposalRunsTheProbe` and
      `TestTheProbeClaimScanFindsAPlantedSentence` green
- [x] `make vet` and `make test` green

### Task 5: Verify acceptance criteria

- [x] every citation still resolves. For each `DECISIONS.md, <date>` and
      `<date> entry in docs/v2/DECISIONS.md` under `go/`, `docs/` and in
      `CLAUDE.md`, the register has that date and its `Entry` column names
      a file holding the entry. For each `MEASURED.md` citation, by title
      or by number, the measured index has the section and names the file
- [x] `CLAUDE.md` is unchanged and `docs/v2/SPEC.md` changed in two lines
      only: `git diff --name-only main...HEAD` and `git diff main...HEAD --
      docs/v2/SPEC.md`
- [x] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
      `make platform-test`, `make team-check`

### Task 6: [Final] Update documentation

- [x] `docs/v2/DECISIONS.md`'s opening paragraph says the file holds the
      register and the entries from 2026-10-16, and names the archive;
      `docs/v2/MEASURED.md`'s preamble does the same for its index
- [x] `go/boundary/doc.go` reads as the five guards it now holds
- [x] CLAUDE.md untouched, and its task map still names files that exist
- [x] move this plan to `docs/plans/completed/`

## Post-Completion

These are the lead's, with Nail.

**The measurement.** `.ralphex/board/meter.py ../gdoc-m29`, per task
iteration and per review round, into the table in `docs/team/measure.md`
next to the M24a, M25, M26 and M27 baseline, with the two numbers the meter
cannot read: the lead session's own context at the end, and how many
messages Nail had to send. The verdict the pilot is measured against is in
`docs/plans/completed/2026-10-23-gdoc-v2-m28-the-team.md`: Nail spoke only to the
lead, zero blocking findings at the end, per task and per review round at
most a third of the baseline, and nothing reverted the next day.

**The release.** `make release BRANCH=gdoc-v2-m29-the-archive` stops at a
green PR. The push, the PR, the merge and any tag wait for Nail's word, and
this milestone changes no binary behaviour, so there may be no version in
it at all.
