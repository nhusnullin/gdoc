# gdoc v2: the docs restructure, and the v1 retirement

2026-09-11.

## Principles

Serves: 4, every word costs a reader's attention. The auto-loaded file drops
from ~50k tokens to under 300 lines, and every essay moves to the one place a
reader is already standing when they need it. Serves 1 by retiring the second
runtime: after this, installing gdoc is one binary.

Strains: none in the machine sense. The retirement removes `gdoc-apply` with
no replacement until M8, and that is a decision Nail took with the gap in
view, not a strain on a principle.

## Overview

CLAUDE.md is 3,311 lines, about 50k tokens, and it is loaded into every
session unasked. About 2,700 of those lines are per-package design essays,
each accurate on the day it was written and each a second copy of reasoning
the package's own doc comments do not carry. SPEC.md and PLAN.md are written
as "what was agreed on 2026-08-29, plus what changed since", so a reader
reconstructs the present from the original and its patches. DECISIONS.md has
40 entries with no status on any of them.

After this milestone:

| File | Today | After | Loaded |
|---|---|---|---|
| CLAUDE.md | 3,311 lines | under 300, held by a test | every session |
| `go/**/doc.go` | none | one per package, the essays trimmed | on demand, and in the diff of every change to that package |
| SPEC.md | 569 lines, 10 amendment blocks | ~350, present tense, no amendments | on demand |
| PLAN.md | 992 lines | ~150, open work only | on demand |
| DECISIONS.md | 1,802 lines, no status | same entries, a register on top, four measurements moved out | register by default |
| MEASURED.md | does not exist | what Google does when it accepts | on demand |
| v1 (`gdoc/`, `tests/`, `skills/gdoc-apply`, …) | 67 tracked files | gone | never |

**Nail's decisions, 2026-09-11, taken in the brainstorm that produced this
plan.** Each is a decision and not a refactor; a task that finds one wrong
stops and says so rather than adjusting it.

1. **CLAUDE.md is invariants only.** Read-first pointers, the v2 "what lives
   where" table, invariants each one sentence with the pinning test named, a
   task map ("if you touch X, read Y"), the build and test table, ten lines on
   running a milestone, the Never list, the writing style. Under 300 lines.
   Everything else leaves.
2. **The essays move into `doc.go` package comments, trimmed while moving.**
   The rule: each rule stated once, its reason once, and the test that pins it
   named in the same paragraph, or a `// TODO(test)` marker where none exists.
   No history: "used to", "before M7", "the review found" go, because
   DECISIONS.md holds them. Measurements of Google's behaviour stay with their
   date and point at MEASURED.md. Plain English, under 20 words a sentence, no
   em dashes.
3. **SPEC.md is the present tense.** Every command described as it is on
   `main`. No "Amended" or "Corrected" blocks. Dates appear only in
   one-sentence pointers to DECISIONS.md. A change to SPEC is a DECISIONS entry
   first, the same day.
4. **PLAN.md is open work.** Done milestones become one table row each. Every
   "what Mx leaves for My" item still open moves to M8, M9 or `docs/backlog/`,
   listed so nothing drops silently. The ordering rationale goes.
5. **DECISIONS.md keeps every entry's text**, gains a register with a status
   per entry, is put in date order, and loses four measurement entries to
   MEASURED.md.
6. **v1 goes, docs and code, in one commit at the end.** No relocation of v1
   prose: git holds it.
7. **Three mechanical guards in `go/boundary/`** hold the shape: the CLAUDE.md
   line ceiling, exactly one package comment per package, and every path in
   the task map existing on disk.

## Context (from discovery)

### Where every line of CLAUDE.md goes

This is the coverage table. Every task that takes a range ticks its rules
against this table, and Task 27 checks that every row is claimed. A range
split across tasks names each owner, and a rule inside a split range names
its destination in the tick list of the task that carries it.

| Lines | Holds | Goes to |
|---|---|---|
| 1–11 | the head | Task 24 |
| 12–31 | What lives where, v1 rows | goes (v1); v2 rows to Task 24 |
| 32–85 | the `go/` path table, "milestones 1 to 7c are done" | table to Task 24; the milestone sentence to Task 23 |
| 86–141 | auth commands, the envelope, `--help`, panic | Task 4 |
| 142–182 | the token file, scopes, `MissingScopes` | Task 5 |
| 183–527 | the guard | Task 3 |
| 528–612 | boundary allowlists, no `os/exec`, `allowedModules` | Task 6 |
| 613–642 | facts, never verdicts | Task 9; the one-line invariant to Task 24 |
| 643–768 | the four read commands | split: argument parsing to Task 4; the Docs read, `Places`, ceilings, footnotes to Task 7; the poll order to Task 9; the witness to Task 19 (`docx`); lists and tables to Task 8 |
| 769–866 | `read`'s text | Task 8 |
| 867–977 | paragraph elements, named ranges | Task 7 |
| 978–1291 | the survey, the apply loop, `--from` | Task 10 |
| 1292–1683 | the prelude | Task 11 |
| 1684–1854 | the cursor, `--wait` | Task 9; the `os/signal` rule to Task 4 |
| 1855–1940 | the front-matter block | Task 12; the `List`/`All`/`IDs` rule to Task 19 (`suggestions`) |
| 1941–2306 | the four write commands | split: `propose` to Task 13; `probe`, `reply`, `withdraw`, `plaintext` to Task 14; the "sent" rule to Task 19 (`gapi`); the re-read rule to Task 4 |
| 2307–2394 | publish | Task 15 |
| 2395–2426 | `GrantInPlace` history | Task 3, dropped as history with a pointer |
| 2427–2785 | the generator | split: `house` and `cover` to Task 15; `render` to Task 16; `body` to Task 17; `drift` to Task 18 |
| 2786–2823 | running a milestone | Task 24, ten lines |
| 2824–2851 | building, the make targets, `gdoc` on PATH | Task 24 |
| 2852–2933 | the live tests | Task 19 |
| 2934–2941 | one root | goes (v1) |
| 2942–2957 | nothing runs git | Task 24, an invariant |
| 2958–3016 | no external programs, pandoc | goes (v1); the v2 rule is already in 580–589 |
| 3017–3036 | the v1 guard | goes (v1) |
| 3037–3078 | which credential | goes (v1) |
| 3079–3120 | the OAuth client is shipped and stays Internal; making the repo public | Task 5, and a dated DECISIONS entry in Task 2 |
| 3121–3145 | pictures | goes (v1) |
| 3146–3198 | v1 restyle | goes (v1) |
| 3199–3234 | v1 edits | goes (v1) |
| 3235–3245 | identity is never a gate | Task 14 (`plaintext`) |
| 3246–3263 | skills are linked, not copied | Task 24, an invariant |
| 3264–3283 | testing, the literal-value rule | Task 24, an invariant; Task 15 points at it |
| 3284–3308 | Never | Task 24, minus the v1 lines |
| 3309–3311 | writing style | Task 24 |

### Facts about the tree

- Package comments today: ⚠️ two of the four claims below were measured wrong
  and Task 1 corrected them. `cmd/gdoc` carries one, on `main.go`, opening
  "Command gdoc". `internal/render` carries one, on `xml.go`, not two.
  `boundary` and `internal/live`
  carry theirs on a **test file** (`boundary_test.go`, `live_test.go`), and
  neither has a non-test file. Every other package has exactly one, on its main file
  (`policy.go`, `survey.go`, `schema.go`, `text.go`, `compare.go`,
  `session.go`, and so on). `internal/auth/loopback` is a package under a
  package. The convention this plan sets: the package comment lives in
  `doc.go` when it is longer than a paragraph, and the file that carried it
  before keeps a plain comment or none.
- The word "v1" appears 195 times in 61 Go files, 24 production and 37 test.
  Seven test functions carry it in their names (`TestLoadReadsV1Format`,
  `TestReadRefusesV1sPairingAndSaysWhatToDoAboutIt`,
  `TestAuthStatusIsQuietForAV1Token` among them). Seven Go comments point at
  Python files by path (`gdoc/docid.py`, `gdoc/oauth.py`, `gdoc/reply.py`,
  `gdoc/auth.py`, `gdoc/comments.py`).
- SPEC.md has 10 `*Amended*` and `*Corrected*` blocks by
  `grep -cE '\*(Amended|Corrected)' docs/v2/SPEC.md`.
- DECISIONS.md entries 29 and 30, dated 2026-09-09, sit before entries 31
  and 32, dated 2026-09-07 and 2026-09-08.
- PRINCIPLES.md carries nine dated decisions from the v1 era and a "gate"
  pointing at `docs/superpowers/`. Principle 1's text names pandoc and
  principle 3's names `write_baseline` and `gdoc/guard.py`.
- `.venv` is a **tracked symlink** to `~/.config/gdoc-agent/venv`.
  `gdoc.egg-info/` is untracked. `docs/v2/spikes/` is tracked and holds the
  Python probes, `compare.py`, `gen.py` and six Go spike modules. `tools/tlsdiag/`
  is Go and stays; `tools/tlsdiag.zip` is a tracked artifact and goes.
  `spike/render/` is tracked and CLAUDE.md calls it throwaway; it goes.
- Legitimate mentions of pandoc that survive the retirement: `internal/body`
  names it to explain goldmark's extension set, SPEC.md says "pandoc is gone
  entirely", PRINCIPLES.md names it in principle 1.
- `.ralphex/board/refresh_board.py` is Python and stays. It is the run board,
  not v1.
- `install.sh` builds the venv, pip-installs v1, then builds and links v2 and
  links both skills. After the retirement it does the second half only.
- The GitHub workflow runs gofmt, vet, the raced tests and `make dist`. It
  runs no Python.

## Development Approach

- **Testing approach:** TDD for the three guards (Task 1), which are the only
  code in this milestone. Every other task is prose, and its check is the
  tick list plus `make test` and `make vet` green.
- One task, one commit, on branch `gdoc-v2-docs-restructure` from `main`.
  ralphex executes, revmux reviews, as every milestone since M2. The plan and
  `.ralphex/` are committed on `main` before the run starts. Commit type is
  `docs(v2)` for prose, `test(v2)` for the guards, `chore` for the retirement.
- **Every doc.go task ends with a tick list.** The executor lists the rules
  the old CLAUDE.md range states, one line each, and marks each as carried,
  dropped as history, or sent to another task by the coverage table. The list
  goes in the commit message body. A rule that is none of the three is a
  defect in the task.
- **A renamed test updates every doc.go that cites it, in the same commit.**
  Task 21 renames tests; Tasks 3 to 19 cite them. The rule keeps the
  citations true, and Task 27 checks every cited name exists.
- Tasks 3 to 21 do **not** delete from CLAUDE.md. CLAUDE.md is rewritten
  whole in Task 24, once every file its task map points at exists. Until then
  the old file and the new doc.go files coexist, which is deliberate: a
  reviewer diffs the two.
- **Nothing in this milestone changes behaviour.** No production Go moves
  except comments, test names, and `render`'s duplicate package comment. If a
  doc.go task finds a rule the code does not hold, it writes `// TODO(test)`
  or a backlog item and does not fix the code.
- **Plain English in every file this plan writes.** Short sentences, no em
  dashes, no banned words. The house writing rules apply to a doc.go exactly
  as they apply to CLAUDE.md.

## Testing Strategy

- **Unit tests:** `go/boundary/docs_test.go`, three tests, written in Task 1
  and live from Task 1. Two of them use the repo's own pattern for a check
  that must fail on a named set until later tasks clear it: a `known` list
  with a second assertion that a listed entry still fails, the way
  `drift.Known` and `TestEveryKnownDifferenceStillDiffers` work. The third
  carries a `t.Skip` naming the task that removes it, because the heading it
  parses does not exist yet.
- **The tick list** is the test for prose. A rule is a sentence in the old
  range that says "never", "always", "refuses", "is the only", "must", or
  names a test. Each is accounted for.
- **Sizes are asserted at the end**, in Task 27, as numbers: CLAUDE.md under
  300, SPEC.md under 400, PLAN.md under 200. The doc.go files have no ceiling.

## Validation Commands

```bash
make test
```

```bash
make vet
```

```bash
make dist
```

## Progress Tracking

- mark completed items with `[x]` immediately when done
- add newly discovered tasks with ➕ prefix
- document issues or blockers with ⚠️ prefix
- update this plan if implementation deviates from the original scope

## Solution Overview

| Kind | Holds | Never holds |
|---|---|---|
| CLAUDE.md | the invariants, where things live, what to read for a task, how to build | an argument, a history, an example |
| `doc.go` | why this package refuses what it refuses, with the test that pins each rule | when it was decided, what it used to do |
| SPEC.md | what v2 is today | amendments; a date except in a pointer |
| PLAN.md | what is next | what is done |
| DECISIONS.md | why, dated, never edited except a status line | measurements |
| MEASURED.md, BLOCKED-BY-API.md | what Google does and refuses, dated, verbatim | a choice |

The guards in `go/boundary/` keep the first row true after this milestone.

## Technical Details

### The doc.go convention

```go
// Package guard is the network policy, and the only place a client is built.
//
// # The levels are names, not a ladder
//
// <one rule, its reason, its test>
package guard
```

- The `// Package x` line is the one-sentence summary, the same sentence
  CLAUDE.md's "what lives where" table carries.
- Headings inside the comment are `// # Heading` lines, which `go doc`
  renders as headings.
- A rule's paragraph ends by naming its test:
  `TestNothingAtLevelInPlaceCanChangeACharacter is the pin.` Where none
  exists: `TODO(test): no test pins this rule yet.`
- A measurement cites its date and MEASURED.md by heading:
  `Measured 2026-09-09, MEASURED.md "Which styling requests land".`
- The file that carried the old package comment loses the `// Package x`
  prefix on that comment, so the package has exactly one.

### The three guards

`go/boundary/docs_test.go`:

- `TestCLAUDEmdIsUnderTheCeiling`: reads `../../CLAUDE.md`, fails over
  `claudeCeiling`. Task 1 sets it to 3311, today's count, with a comment that
  Task 24 lowers it to 300 and that raising it afterwards is a decision
  somebody explains in the commit.
- `TestEveryPackageHasExactlyOnePackageComment`: walks `cmd/gdoc`,
  `internal/` recursively and `boundary` with `filepath.WalkDir`, parses every
  `.go` file including tests with `go/parser` in `ParseComments` mode, and
  counts files whose `Doc` is set. Exactly one per package passes. A package
  on `known` (`cmd/gdoc` with none; `boundary`, `internal/live` with theirs on
  a test file; `internal/render` with two) is expected to fail, and the test
  fails again when a listed package already passes, so each task that fixes a
  package deletes its row.
- `TestTheTaskMapNamesFilesThatExist`: reads CLAUDE.md, finds the table under
  the heading `## If you touch`, extracts every backticked path in it, and
  stats each relative to the repo root. A missing heading is a failure.
  `t.Skip("fails until Task 24 writes the heading; Task 24 removes this skip")`.

### The DECISIONS register

```markdown
| Date | Decision | Status |
|---|---|---|
| 2026-08-29 | gdoc never replaces the body of a document that already exists | holds |
| 2026-08-29 | gdoc marks its own changes by colour, never by highlight | superseded 2026-08-29 (the colour scheme is retired) |
| 2026-08-29 | Anchors are not banked | rejected |
| 2026-09-09 | What an in-place styling request actually reaches | MEASURED.md |
```

Status is one of `holds`, `rejected`, `superseded YYYY-MM-DD (entry)`, or
`MEASURED.md`. Nothing else. Every status is stated in Task 2, not defaulted.

### MEASURED.md

Same shape as BLOCKED-BY-API.md: a first paragraph saying what the file is
and is not, then one section per measurement carrying the date, the test that
made it, the verbatim answer where there is one, and a "recheck when" line.
Four entries move with their text intact and their DECISIONS heading turned
into a section heading. The section headings are fixed in Task 2 so every
later doc.go cites a heading that exists:

| From entry | Heading |
|---|---|
| 29 | Which styling requests land in place |
| 36 | In-place styling preserves anchors and pending suggestions |
| 38 | A named range over a suggested insertion |
| 39 | A table takes one index of its own at the end |

Entry 33 (the seven dropped elements) stays in DECISIONS.md: its subject is
gdoc's own decoder and a fixture built from the reference, which is a
decision. Its outstanding live check is one line under MEASURED.md's "recheck
when".

## Implementation Steps

### Task 1: the three guards, live from the first commit

**Files:**
- Create: `go/boundary/docs_test.go`

- [x] write `TestCLAUDEmdIsUnderTheCeiling` with `claudeCeiling = 3311` and the comment that Task 24 lowers it to 300
- [x] write `TestEveryPackageHasExactlyOnePackageComment` with `WalkDir`, `go/parser` over every `.go` file including tests, `known` naming the packages that fail today, failing on a package off the list with other than one comment and on a listed package that already holds the rule
- [x] write `TestTheTaskMapNamesFilesThatExist` parsing the table under `## If you touch`; a missing heading fails; `t.Skip` naming Task 24
- [x] watch guard 2 fail with `known` emptied, then restore the list; note the failure messages in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "test(v2): three guards over the docs' shape"`

⚠️ `known` holds two names, not four. The plan's discovery facts are wrong on
two packages, measured 2026-09-15: `cmd/gdoc` carries one package comment, on
`main.go`, opening "Command gdoc", and `internal/render` carries one, on
`xml.go`, not two. Both already hold the rule, so listing either would fail the
test on its own second assertion. `known` is `boundary` and `internal/live`,
the two packages whose only files are tests, so their package comment sits
where `go doc` does not read it. Tasks 6 and 19 clear them.

⚠️ The rule the guard asks is "exactly one package comment, in a file that is
not a test". A package comment is a Doc comment on the package clause that
opens `Package <name>` or, for `main`, `Command `. Counting every file whose
`Doc` is set, which is what the plan's Technical Details says, would refuse the
repo's own convention: most files here carry a paragraph above the package
clause saying what that file is for, so nine packages off `known` would fail
from the first commit.

### Task 2: MEASURED.md, and the DECISIONS register

**Files:**
- Create: `docs/v2/MEASURED.md`
- Modify: `docs/v2/DECISIONS.md`, `docs/v2/BLOCKED-BY-API.md`

- [x] create MEASURED.md with the head paragraph and the four sections named in Technical Details, moving entries 29, 36, 38, 39 with their text intact; entry 33's live check becomes one "recheck when" line
- [x] reorder DECISIONS.md so every entry is in date order
- [x] add the OAuth client decision from CLAUDE.md 3079–3120 as an entry dated 2026-08-18 (the secret stays in git, the client stays Internal, making the repo public means a fresh client first)
- [x] write the register table at the top with every status stated: 3 superseded 2026-08-29 (entry 20); 8 rejected; 9 holds; 16 superseded 2026-09-09 (entry 35, the checklist is not built); 17 superseded 2026-09-11 (entry 40); 20 holds, with its "no named ranges" bullet noted as superseded 2026-09-10 (entry 37); 26 holds, widened 2026-09-09 (entry 34); 29, 36, 38, 39 MEASURED.md; every remaining entry `holds`, each listed by number in the commit body
- [x] add the rule under the register: an entry is never edited after this except its status line; a new decision is a new entry and a new row
- [x] BLOCKED-BY-API.md's head names MEASURED.md as its sibling
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): MEASURED.md, and a register over DECISIONS.md"`

### Task 3: `internal/guard/doc.go`

**Files:**
- Create: `go/internal/guard/doc.go`
- Modify: `go/internal/guard/policy.go` (demote its package comment)

- [x] move CLAUDE.md 183–527 under the convention: the levels as names, the six doors (`Learn`, `GrantInPlace`, `AllowReject`, `AllowCopy`, `AllowCreateIn`, `AllowMarker`), `isSuggestMode` and `hasDuplicateKeys`, the query and header allowlists, `checkAuthorization`'s limit, the multipart agreement rule, plain paths only, `inPlaceKinds`, `checkInPlaceMask`, `checkMaskIsSet`, the nil proxy, the mutex
- [x] drop 2395–2426 as history with one sentence pointing at DECISIONS.md 2026-09-09
- [x] drop "before M6 the guard picked from nothing" and the inverted-test story; cite MEASURED.md and BLOCKED-BY-API.md for the `writeMode` measurements
- [x] name the pinning test in every rule's paragraph; `TODO(test)` where none exists
- [x] rewrite every "v1" in this package's production comments to say the thing itself
- [x] tick list in the commit body against 183–527 and 2395–2426
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): the guard's package comment"`

### Task 4: `cmd/gdoc/doc.go`

**Files:**
- Create: `go/cmd/gdoc/doc.go`
- Modify: `go/cmd/gdoc/main.go` (demote its package comment)

- [x] move CLAUDE.md 86–141: the envelope, `auth status` fields, `auth login` to stderr, no prompts and no stdin, `--help` is `ok: false`, panic recovery, `GDOC_CONFIG_DIR`
- [x] move the strict argument parsing rule from 643–768, the `os/signal` rule from 1758–1854, and the "note is read again just before it is written" rule (`freshNote`, `notePath`) from 1941–2306
- [x] a table of the twelve commands, one line each, pointing at the package that does the work
- [x] name the pinning tests: `TestOnlyTheWaitTrapsTheSignal`, the `--help`, panic and argument tests
- [x] `cmd/gdoc` is not on guard 2's `known` (it already holds the rule, see Task 1); confirm it still passes after the demotion
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): cmd/gdoc's package comment"`

⚠️ `go doc ./cmd/gdoc` leads with `build.go`'s file paragraph, because go/doc
joins every comment above a package clause in file order and `build.go` sorts
before `doc.go`. That is the repo's own convention meeting alphabetical order,
not a rule this task broke: `internal/guard` reads well only because `doc.go`
sorts first there. Guard 2 passes either way, since it counts only a comment
opening "Command " or "Package <name>". Left as it is.

### Task 5: `internal/auth/doc.go`

**Files:**
- Create: `go/internal/auth/doc.go`
- Modify: `go/internal/auth/auth.go` (demote), `go/internal/auth/login.go` (the scope comment)

- [x] move CLAUDE.md 142–182: the token file is google-auth's "authorized user" shape, the scope widening, `MissingScopes` and `coveredBy`, `Save` carrying the three fields it does not use, `Load` refusing an unrefreshable file, `token_uri` filled in
- [x] move 3079–3120: the client is shipped, the secret belongs in git (RFC 8252 8.5, the `gh` and `gcloud` precedent), the client stays Internal, and the one thing that changes it, making the repo public; point at the DECISIONS entry Task 2 added
- [x] rewrite `login.go`'s scope comment to state the difference between the two scope sets without naming v1
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): auth's package comment, and the OAuth client rule"`

### Task 6: `boundary/doc.go`

**Files:**
- Create: `go/boundary/doc.go`
- Modify: `go/boundary/boundary_test.go` (demote), `go/boundary/docs_test.go` (delete `boundary` from `known`)

- [x] move CLAUDE.md 528–612: the import allowlist and the builder allowlist and why they differ, the eight ways to make a wire and the canary, no `os/exec`, `allowedModules` with the three modules and their reasons, `TestAllowedModulesAreReallyRequired`
- [x] describe the three docs guards from Task 1
- [x] demote `boundary_test.go`'s package comment; delete `boundary` from `known`
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): boundary's package comment"`

### Task 7: `internal/docs/doc.go`

**Files:**
- Create: `go/internal/docs/doc.go`
- Modify: `go/internal/docs/docs.go` (demote)

- [x] move from 643–768 the decoder's rules: one Docs read with three views, Drive is the source of threads and Docs of ranges, the measured `commentAnchors` shape, `Places` as the one rule with its two forms, the read ceiling, footnotes flattened
- [x] move 867–977: the `default` arm reports, every member carries its suggestion ids, the fixture built from the reference with the live check outstanding (MEASURED.md "recheck when"), named ranges keyed by id, `Range.Segment`, `NamedRangesURL`
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): docs' package comment"`

### Task 8: `internal/view/doc.go`

**Files:**
- Create: `go/internal/view/doc.go`
- Modify: `go/internal/view/text.go` (demote)

- [x] move CLAUDE.md 769–866: the markers as a table, every marker escaped and why, the backslash and the parity rule, the escape advances by one rune, a chip's label is escaped, `[rule]` not `---`, inverted ranges are warnings, `--structure`; and from 643–768 the list and pipe-table rules
- [x] the per-run escaping limit points at `docs/backlog/escaping-across-run-boundaries.md`
- [x] name the pinning tests, the golden file among them
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): view's package comment"`

### Task 9: `internal/comments/doc.go`

**Files:**
- Create: `go/internal/comments/doc.go`
- Modify: `go/internal/comments/comments.go` (demote)

- [x] move 613–642 (facts, never verdicts, with the named fields and `TestThreadsCarriesEveryFactAndJudgesNone`), 1684–1757 (the cursor's shape, milliseconds, `narrow`, the ids, the empty cursor and the five-minute floor, opaque), and 1758–1854 minus the `os/signal` rule (`Wait`: `--since` required, the interval, first non-empty window, the deadline bounds the call, the refresh carries the context, nothing kept)
- [x] the poll-order rule (listing first, document second) from 643–768, stated once here; `restyle` points at it
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): comments' package comment"`

➕ `go doc` joins every package-clause comment in file order, so a file
comment alphabetically ahead of `doc.go` leads the rendered package doc.
`internal/comments` and `internal/auth` both read that way today (`comments.go`,
`cursor.go`, `auth.go`). Each of those file comments points at `doc.go`, which
is the repo's convention from Task 5 on, so this task matched it rather than
diverging. Whether every such header loses its blank-line adjacency to the
package clause is one decision for Task 27, not a choice per package.

### Task 10: `internal/restyle/doc.go`

**Files:**
- Create: `go/internal/restyle/doc.go`
- Modify: `go/internal/restyle/survey.go` (demote)

- [x] the survey, from 978–1064: three reads, listing first, a failed Docs read vs a failed export, pending is `All`'s, the witness twice, the revision id, `schema` 1, `nothing_to_protect` is five zeros and never a recommendation; the route the skill offers over it is `read`, a note, `publish`, pointer to DECISIONS 2026-09-11
- [x] the apply loop, from 1065–1140: `requiredRevisionId`, read between batches for the revision only, never retried, `maybe_applied`, batches in bytes, no rollback and `leftBehind`
- [x] `--from`, from 1141–1291: survey and apply are two runs, four refusals before the grant, a restyle is a moment, one request per table, what it overwrites, the read-back's two halves and its seven rules, `manual`, `verified: false` is not a failure
- [x] cite MEASURED.md "Which styling requests land in place" and "In-place styling preserves anchors and pending suggestions"
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): restyle's package comment"`

### Task 11: `internal/prelude/doc.go`

**Files:**
- Create: `go/internal/prelude/doc.go`
- Modify: `go/internal/prelude/prelude.go` (demote)

- [x] the two phases and two policies, unprobed and why, from 1292–1360
- [x] `AllowMarker`, why the marker is written, `createParagraphBullets` at the two levels, the marker's name and id, from 1360–1420
- [x] `Decide`'s three shapes, pending asked before the count, nothing deletes a named range, the marker batch on its own, `marker_maybe_created`, the revision chained across the phase boundary, from 1420–1530
- [x] phase 2 walks past the span, `Occupies` on a replace run, full-look paragraphs, the page break's two units, the table's index accounting as literals, one request per cell, from 1530–1610
- [x] the fields file, the control-character refusal, no inferred title, `Verify`'s three questions, a failed phase 1, `Manual`, heading numbering out, from 1610–1683
- [x] every measurement cites MEASURED.md by heading rather than restating numbers
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): prelude's package comment"`

### Task 12: `internal/frontmatter/doc.go`

**Files:**
- Create: `go/internal/frontmatter/doc.go`
- Modify: `go/internal/frontmatter/schema.go` (demote)

- [x] move 1855–1940 minus the `List`/`All`/`IDs` rule: the block's fields, the publish record, the strict read, `schema` 1, byte-preserving write, the write checked against its own parse, snapshot after a successful read
- [x] the refused string form is described as "a bare string where a block is expected", with the refusal's own message, not as another tool's shape
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): frontmatter's package comment"`

### Task 13: `internal/propose/doc.go`

**Files:**
- Create: `go/internal/propose/doc.go`
- Modify: `go/internal/propose/propose.go` (demote)

- [x] from 1941–2306, `propose`'s own rules: names text never an index, contiguous, the span's end from the last rune, a crossing occurrence still counts, replaces words with words, line breaks refused on both sides, every proposal checked first, one batch of three, the three read-backs and the fourth condition, the preview asks by words and the replacement-contains-quote case, `docx_anchored`'s no-answer, provenance recorded and `missingID`
- [x] the "sent" rule and the re-read rule are pointed at, not restated (`gapi`, `cmd/gdoc`)
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): propose's package comment"`

### Task 14: `probe`, `reply`, `withdraw`, `plaintext`

**Files:**
- Create: `go/internal/probe/doc.go`, `go/internal/withdraw/doc.go`
- Modify: `go/internal/probe/probe.go`, `go/internal/withdraw/withdraw.go` (demote), `go/internal/reply/reply.go`, `go/internal/plaintext/plaintext.go` (extend in place; both are short)

- [x] `probe`: runs every time and nothing caches, its own document and never the reviewed one, every failure path trashes and names the document, the first production caller of `AllowCreateIn`
- [x] `withdraw`: a `rejectSuggestion` on gdoc's own id (DECISIONS 2026-09-07), provenance is the permission and `AllowReject` holds it, gone only when both facts hold, "usually" and the decoded-field gate, the 🤖 comment stays
- [x] `reply`: the mark required, asked behind the mark, the thread read back on a surviving id
- [x] `plaintext`: the 🤖 prefix as the only record of authorship, no markdown, the two writers own the mark differently; and from 3235–3245, identity is never a gate: the marker decides, never the account
- [x] name the pinning tests
- [x] tick list in the commit body, per package
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): the package comments of probe, reply, withdraw and plaintext"`

### Task 15: `internal/publish/doc.go`, `drive`, `house`, `cover`

**Files:**
- Create: `go/internal/publish/doc.go`
- Modify: `go/internal/publish/publish.go` (demote), `go/internal/drive/drive.go`, `go/internal/house/house.go`, `go/internal/cover/cover.go` (extend in place)

- [x] `publish`, from 2307–2394: the create-only policy, not knowing never resolves to keeping the document, the three failure shapes, `rolled_back` as `*bool`, three read-backs, `title` twice, the re-read rule is publish's own, one render function for `build` and `publish`
- [x] `drive`: the trash is believed only on its confirming read, and its two callers
- [x] `house`, from 2427–2785: the style is a file and embedded, `--house`, points stay points, a highlight is a name, the master is provenance; "a house-style test states its value as a literal" points at CLAUDE.md's invariant
- [x] `cover`: the keys, no inferred title and `MissingTitle`'s candidate, the two defaults, classification validated, the `gdoc:` key skipped
- [x] name the pinning tests
- [x] tick list in the commit body, per package
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): the package comments of publish, drive, house and cover"`

### Task 16: `internal/render/doc.go`

**Files:**
- Create: `go/internal/render/doc.go`
- Modify: `go/internal/render/xml.go` (demote; it is the one file carrying the package comment, not `render.go`)

- [x] from 2427–2785: nothing concatenated into XML, properties in schema order with the three order tests, every list level states `w:start`, paragraph marks carry their size, `placeholder` and the three cover rules, the contents list is a Word field, no network
- [x] the package has exactly one package comment; `internal/render` is not on guard 2's `known` (it already holds the rule, see Task 1)
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): render's package comment"`

### Task 17: `internal/body/doc.go`

**Files:**
- Create: `go/internal/body/doc.go`
- Modify: `go/internal/body/body.go` (demote)

- [x] from 2427–2785: one marker per item and the pending marker, what the walker refuses with a line, a picture at `http` refused, footnotes on so they can be refused, the email autolink, the numbering warnings and the heading zero, `Sources`
- [x] the pandoc mentions in this package's comments stay where they explain goldmark's extension set; each is reworded to say "the previous parser" only if it names a tool the reader cannot see
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): body's package comment"`

### Task 18: `internal/drift/doc.go`

**Files:**
- Create: `go/internal/drift/doc.go`
- Modify: `go/internal/drift/compare.go` (demote)
- ⚠️ Rename: `go/internal/drift/doc.go` → `docsapi.go`, and `doc_test.go` → `docsapi_test.go`. The plan's discovery missed that this package already had a `doc.go`, and it is not a package comment: it is the Docs half of the reader, 607 lines of code beside `docx.go`. The name had to be freed before the package comment could take it, which every other package in the tree already holds. A pure rename, no behaviour and no code changed.

- [x] from 2427–2785: the item list written once, both gates and what each measures, `Known` explains a difference and never a fault, `Bug`, the PDF items dropped, the two closed reader holes, the live read is not `docs.URL`
- [x] name the pinning tests
- [x] tick list in the commit body
- [x] `make test`, `make vet` green
- [x] `git commit -m "docs(v2): drift's package comment"`

### Task 19: `internal/live/doc.go`, and the small packages

**Files:**
- Create: `go/internal/live/doc.go`
- Modify: `go/internal/live/live_test.go` (demote), `go/boundary/docs_test.go` (delete `internal/live` from `known`), `go/internal/suggestions/suggestions.go`, `go/internal/gapi/session.go`, `go/internal/docx/docx.go` (extend in place)

- [ ] `live`, from 2852–2933: the two variables and why two, what each test does, which copy and which create, `GDOC_LIVE_RECORD`, `TestTheLiveFixturesRenderWithNoNetwork`
- [ ] demote `live_test.go`'s package comment; delete `internal/live` from `known`; `known` is now empty and the test's second assertion has nothing to check
- [ ] `suggestions`: the `List`/`All`/`IDs` rule from 1855–1940
- [ ] `gapi`: the "sent" rule from 1941–2306, the one place, with the three unmarked cases
- [ ] `docx`: the witness's two limits and the no-answer on disagreement, from 643–768
- [ ] name the pinning tests
- [ ] tick list in the commit body, per package
- [ ] `make test`, `make vet` green
- [ ] `git commit -m "docs(v2): live's package comment, and the small packages"`

### Task 20: "v1" and the Python paths leave the production comments

**Files:**
- Modify: every non-test file under `go/` that `grep -rnE '\bv1\b|gdoc/[a-z_]+\.py|tests/test_|docs/superpowers|spike/render' go --include='*.go' --exclude='*_test.go'` names

- [ ] list every hit; each is rewritten to say the thing it means (the token shape, the marker, the ported rule, the measured value) or deleted where it was only history
- [ ] the grep above returns nothing except an API path such as `drive/v3`; the exceptions are listed in the commit body
- [ ] `make test`, `make vet` green
- [ ] `git commit -m "docs(v2): production comments name things, not v1"`

### Task 21: "v1" leaves the test files, and the doc.go citations follow

**Files:**
- Modify: every `_test.go` under `go/` the same grep names, and every `doc.go` that cites a renamed test

- [ ] rename the seven test functions carrying "V1" to say what they test (`TestLoadReadsTheAuthorizedUserShape`, and so on), and rewrite every test comment the grep names
- [ ] `grep -rn` each old test name across `go/`; every doc.go citation is updated in this commit
- [ ] the grep from Task 20 over test files returns nothing; exceptions listed
- [ ] `make test`, `make vet` green
- [ ] `git commit -m "test(v2): test names say what they test"`

### Task 22: SPEC.md in the present tense

**Files:**
- Modify: `docs/v2/SPEC.md`

- [ ] new header: what the file is (what v2 is today), the rule (a change here is a DECISIONS entry first, the same day), no status line, no date
- [ ] fold every block `grep -nE '\*(Amended|Corrected)' docs/v2/SPEC.md` finds into the sentence it amended and delete the old sentence; each of the ten is listed in the commit body with what it became
- [ ] `restyle` has one mode: survey, `--from`, `--fields`; the "detected exception" reports a fact and the skill offers the two-step route
- [ ] `publish` runs once; a second version is the note published again after the block is taken out by hand; pointer to DECISIONS 2026-09-11
- [ ] "The diff" is two paragraphs: the binary reads the document, the skill composes
- [ ] the Developer Preview risk paragraph stays as the one accepted external risk
- [ ] dates appear only in pointers of the form "changed YYYY-MM-DD, DECISIONS.md"; `grep -c "2026-" docs/v2/SPEC.md` is reported in the commit body
- [ ] under 400 lines
- [ ] `make test`, `make vet` green
- [ ] `git commit -m "docs(v2): SPEC.md says what v2 is today"`

### Task 23: PLAN.md as open work

**Files:**
- Modify: `docs/v2/PLAN.md`, `docs/backlog/*.md` (new items where an open leftover belongs there)

- [ ] "Standing facts" trimmed of anything CLAUDE.md's invariants will hold
- [ ] M1 to M7c become one table: milestone, one line of what it delivered, date, plan file under `docs/plans/completed/`; the "milestones 1 to 7c are done" sentence from CLAUDE.md 32–85 is the source for the one-line summaries
- [ ] walk every "what Mx leaves for My" list in the deleted bodies; each item is done (say where), moved to M8 or M9, or written as a `docs/backlog/` item; the full list with its disposition goes in the commit body
- [ ] M8 and M9 stay in full; M9 loses "the v1 retirement note in the README"
- [ ] the ordering rationale goes
- [ ] under 200 lines
- [ ] `make test`, `make vet` green
- [ ] `git commit -m "docs(v2): PLAN.md holds the open work"`

### Task 24: CLAUDE.md, rewritten whole

**Files:**
- Modify: `CLAUDE.md`, `go/boundary/docs_test.go` (lower the ceiling, remove the skip)

- [ ] write the file in seven parts: read first (PRINCIPLES.md, then the doc.go of the package the task touches); "What lives where", v2 rows only from CLAUDE.md 12–85, one line each; the invariants, each one sentence naming its test, including nothing runs git (2942–2957), skills are linked (3246–3263), a house-style test states its value as a literal (3264–3283), the binary prints facts, one JSON object on stdout, no prompts; `## If you touch`, the task map table with a backticked path per row; building and testing as a table from 2824–2851; running a milestone in ten lines from 2786–2823 with a pointer to `.ralphex/board/README.md`; Never from 3284–3308 minus the v1 lines; writing style
- [ ] the two written rules in the head: a SPEC change is a DECISIONS entry the same day with its register row; a doc.go names the test for every rule or carries `TODO(test)`
- [ ] tick list in the commit body: every rule in the old Never list and every invariant is in the new file or in a doc.go the task map points at
- [ ] lower `claudeCeiling` to 300; remove the `t.Skip` from `TestTheTaskMapNamesFilesThatExist`
- [ ] `wc -l CLAUDE.md` under 300; `make test`, `make vet` green
- [ ] `git commit -m "docs(v2): CLAUDE.md is the invariants"`

### Task 25: README.md and PRINCIPLES.md without v1

**Files:**
- Modify: `README.md`, `PRINCIPLES.md`

- [ ] README.md: the v1 sections (The loop, What it does today, What you need, Setup, Using it, What you get told, Limitations) go; "The Go rewrite" becomes the body, retitled; the install section is left as one line saying Task 26 rewrites it, so this commit does not document a script that does not exist yet; "What is planned" points at PLAN.md
- [ ] PRINCIPLES.md: the four principles stay; a principle's supporting text is rewritten where it names deleted code (pandoc in 1, `write_baseline` and `gdoc/guard.py` in 3), and the principle itself does not change
- [ ] PRINCIPLES.md "Decisions": the entries that bind v2 move to DECISIONS.md with their dates and register rows: 2026-08-13 the markdown is the source; 2026-08-14 skills are symlinked; 2026-08-15 the client reaches only the files it was given; 2026-08-18 nothing runs git; 2026-08-29 the next version is written in Go. The rest go: Commenter-only (retired), the unstated `auth_mode` (v1), `ai:`-marked only (superseded by the 🤖 marker entries). "Open violations" goes. The classification goes in the commit body
- [ ] "The gate" points at `docs/plans/` instead of `docs/superpowers/` and drops `tests/test_pandoc_path.py`
- [ ] `make test`, `make vet` green
- [ ] `git commit -m "docs: README and PRINCIPLES describe one tool"`

### Task 26: the v1 retirement

**Files:**
- Delete (tracked): `gdoc/`, `pyproject.toml`, `tests/`, `skills/gdoc-apply/`, `docs/superpowers/`, `docs/v2/spikes/`, `spike/render/`, `tools/tlsdiag.zip`, `.venv`
- Delete (untracked): `gdoc.egg-info/`
- Modify: `install.sh`, `.gitignore`, `README.md` (the install section), `Makefile` (only if it names anything removed)

- [ ] `git rm -r` the tracked paths; `rm -rf gdoc.egg-info`; confirm `tools/tlsdiag/` and `.ralphex/board/refresh_board.py` are untouched
- [ ] `install.sh` becomes: `make build`, link `bin/gdoc` to `~/.local/bin/gdoc`, remove a stale `gdoc2` link, link `skills/gdoc-review`, remove the `~/.claude/skills/gdoc-apply` symlink when it points into this repo and say so, refuse to replace a real directory whose contents differ, print `+ uncommitted changes` on a dirty tree; it never touches `~/.config/gdoc-agent/` and its head comment says so
- [ ] README.md's install section describes that script
- [ ] `.gitignore` loses `.pytest_cache/`, `.venv/`, `venv/`, `docs/gdoc/*/out/`
- [ ] the gate: `grep -rnE 'venv|pytest|pandoc|gdoc-apply|superpowers|spike/render|docs/v2/spikes' --exclude-dir=.git --exclude-dir=plans .` returns only the named exceptions: SPEC.md's "pandoc is gone entirely", PRINCIPLES.md principle 1, `internal/body`'s parser comments, DECISIONS.md and MEASURED.md history; anything else is fixed in this task
- [ ] run `./install.sh` on this machine; it exits 0 and `gdoc auth status` still answers `ok: true`
- [ ] `make test`, `make vet`, `make dist` green
- [ ] `git commit -m "chore: retire v1"`

### Task 27: verify acceptance criteria

- [ ] `wc -l CLAUDE.md docs/v2/SPEC.md docs/v2/PLAN.md` under 300, 400, 200
- [ ] `cd go && go test ./boundary/ -run 'TestCLAUDEmd|TestEveryPackage|TestTheTaskMap' -v` shows all three run and pass, none skipped, `known` empty
- [ ] `cd go && go doc ./internal/guard | head -40` renders the essay with headings
- [ ] every `MEASURED.md "..."` and `DECISIONS.md YYYY-MM-DD` pointer in `go/` resolves: extract each and grep for the heading or the date
- [ ] every `Test[A-Za-z]+` name cited in a `doc.go` exists: extract, `grep -rn "func <name>("`, zero misses
- [ ] every row of the coverage table is claimed: for each range, the task named holds a tick list in its commit body that accounts for it
- [ ] the grep from Task 20 over all of `go/` and the gate from Task 26 both hold
- [ ] the register has one row per `## ` entry in DECISIONS.md: counts match
- [ ] `make test`, `make vet`, `make dist` green; CI green on the branch

### Task 28: close the milestone

- [ ] `.ralphex/board/README.md` names no v1 Python: the venv, pytest, the `gdoc` package. `refresh_board.py` stays and is named as the board tool
- [ ] move this plan to `docs/plans/completed/`
- [ ] PLAN.md's done table gains this milestone's row
- [ ] `git commit -m "docs(v2): close the docs restructure"`

## Post-Completion

**Manual verification:**

- Open a fresh Claude Code session in this repo and confirm the auto-loaded
  context is the new CLAUDE.md alone. Ask it to change something in the guard
  and watch which file it reads first.
- Delete `~/.config/gdoc-agent/venv` by hand. Nothing in the repo does it.
- Run one live review session on a real document to confirm `gdoc-review`
  still resolves `gdoc` on PATH to `bin/gdoc` after the new `install.sh`.
- Update the session memory index for this repo's docs layout, if one is
  kept.

**External:**

- The `gdoc-apply` skill is gone with no replacement until M8's align skill.
  Between now and then, a hub note is published with `gdoc publish --md` and
  changed in the document with `gdoc propose`. That is the v2 loop, and it is
  the state of the product, not a gap this milestone introduced.

## What this milestone leaves for later

- M8 inherits every open leftover Task 23 moves to it, listed in that task's
  commit body.
- Whether the doc.go files want a ceiling of their own is a question for the
  first time one grows past what a reader can hold. Nothing here sets one.
