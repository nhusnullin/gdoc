# gdoc v2 Milestone 16: settings, the token in the Keychain, and answers an AI can act on, plan

2026-10-04. The task list for the specification beside it,
`docs/plans/2026-10-04-gdoc-v2-m16-settings-spec.md`, whose decisions are
Nail's and settled. One commit per task. Released as v2.11.0, one release
carrying Parts A, B and C, after the Keychain measurement with gdoc's own
Developer ID passes and after Nail's acceptance.

The spec holds the rules, the decisions and the measurements. This plan repeats
none of them. Where the code, the spec or this plan disagree, the task stops and
says so rather than choosing.

Task 1 and Part A (Tasks 2 to 8) are run by hand in the session that wrote
this plan. Part A changes no contract and needs no decision. Parts B and C
follow from Task 9.

## Principles

Serves: 4, every word costs a reader's attention. A refusal says what to do
next, and a colleague picks a quieter mark. Serves: the token leaves a plain
file.

Strains, each written into DECISIONS.md by Task 1 before the code: the robot
prefix invariant, "the marker is the only record of authorship", where secrets
live, and "nothing under go/ runs an external program" only as far as the
existing `/usr/bin/open` exception is named.

## Context (from discovery)

- `go/internal/propose/propose.go:178` and `go/internal/annotate/annotate.go:67`
  refuse a missing reason with "one Nail has to guess at";
  `go/cmd/gdoc/annotate.go:93` has the same words. No test quotes them.
- `go/internal/propose/doc.go:306` says "A quote inside a table cell is refused
  on both routes". The paragraph is about the two block routes. `paragraphs()`
  (`span.go:183`) walks into tables, and no offline test pins a words proposal
  in a cell; the evidence is MEASURED.md "A proposal into a table cell lands",
  2026-09-18.
- `readProposals` (`go/cmd/gdoc/write.go:557`) decodes the whole list with
  `DisallowUnknownFields`, so an unknown field reads `json: unknown field
  "reason"`. `readAnnotations` (`go/cmd/gdoc/annotate.go:143`) does the same.
  `TestProposeRefusesAnUnknownFieldInABlockEntry` asserts only that the field
  is named.
- `Proposal.checkWords` (`propose.go:118`) refuses a deletion with "a plain
  deletion is not a shape this write has", naming no way forward.
- `runPropose` (`write.go:267`) returns `err.Error()` from `propose.Apply` raw
  when a proposal fails before its write, so the refusal does not say which
  proposal it was.
- The `--from` summary (`commands.go:278`) names `quoted, replacement` and the
  block fields but not `why`, which `checkWhy` requires.
- `runPropose` reads the document (`docs.Fetch`, for the tab count), then
  `applyWords` and `ApplyBlock` read it again through `readOneTab`, which is the
  same `docs.Fetch` plus the same tab refusal.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- One commit per task. Before each commit: `cd go && go test -race ./...`,
  `test -z "$(gofmt -l .)"`, `go vet ./...` and `GOOS=darwin go vet ./...`.
- **CRITICAL: every task MUST include new or updated tests.**
- **CRITICAL: all tests must pass before the next task starts.**
- **CRITICAL: no assertion is loosened.**
- **CRITICAL: a test states its value as a literal**, never reading the constant
  it checks.
- **CRITICAL: no doc.go rule without its test named**, and a doc.go or test name
  made false by a task is fixed in that task.
- **CRITICAL: no real company domain in examples.** `example.com` only. The
  signing identifier `com.altery.gdoc` is Nail's decision and is not an
  example.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces. Plain English.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && test -z "$(gofmt -l .)" && go vet ./... && GOOS=darwin go vet ./...`
- `make dist` builds all three binaries.
- `wc -l CLAUDE.md` under 300.

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Implementation Steps

### Task 1: the DECISIONS.md entries, SPEC.md and CLAUDE.md

Written before any settings or Keychain code. Part A needs no entry.

**Files:**
- Modify: `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md`, `CLAUDE.md`

- [x] One entry, dated the day this task runs, "Settings, the mark in effect,
      and the token in the macOS Keychain", from the spec's decisions, in the
      file's own shape, with its register row `holds`. Its test list is this
      plan's test names.
- [x] The M14 entry's row names the one clause superseded: chat's
      `robot_not_ours` fact is removed. The 2026-08-29 "Everything gdoc writes
      opens with 🤖" row is superseded in the same words.
- [x] `SPEC.md`: the robot prefix becomes the mark in effect; the token on
      macOS lives in the Keychain; the settings file and its two arguments.
- [x] `CLAUDE.md`, under 300 lines: the robot prefix invariant names the mark
      in effect and that it may be empty; "Secrets live in" names the Keychain
      on macOS. The external-program invariant already names the `open`
      exception on main, so it is unchanged. CLAUDE.md is at 299 lines: a later
      task that adds a line shortens a row first.
- [x] `cd go && go test -race ./boundary/` passes.
- [x] `git commit -m "docs(decisions): settings, the mark in effect, and the token in the Keychain"`

### Task 2: no refusal names a person

**Files:**
- Modify: `go/internal/propose/propose.go`, `go/internal/annotate/annotate.go`,
  `go/cmd/gdoc/annotate.go`
- Create: `go/boundary/refusal_test.go`

- [x] `TestNoRefusalNamesAPerson` reads every non-test `.go` file under `go/`
      and fails on a line that builds an error (`errors.New(`, `fmt.Errorf(`)
      or an `emit.Result` `Error:` and carries `Nail`.
- ➕ `TestTheRefusalScannerFindsAllThreeShapes`, so the scanner cannot pass a
      tree by finding nothing; it reads strings joined with `+`, which is how
      the `cmd/gdoc/annotate.go` sentence is built. `boundary/doc.go` names both.
- [x] The three sentences say "is one the reader has to guess at".
- [x] `git commit -m "fix: no refusal names a person"`

### Task 3: a words proposal lands inside a table cell, and the note says so

**Files:**
- Modify: `go/internal/propose/doc.go`, `go/internal/propose/span_test.go`

- [ ] `TestFindSpanPlacesAQuoteInsideATableCell`: a document whose one table
      holds `Business` and `£20` in two cells; `FindSpan(d, "£20")` returns the
      cell's range, by literal indexes.
- [ ] `doc.go:306` says "on both block routes", and one sentence says a words
      proposal is placed in a cell, naming the test.
- [ ] `git commit -m "docs(propose): a words proposal lands in a table cell, and a test pins it"`

### Task 4: an unknown field is refused with the fields the entry takes

**Files:**
- Modify: `go/cmd/gdoc/write.go`, `go/cmd/gdoc/annotate.go`,
  `go/cmd/gdoc/propose_block_test.go`
- Create: `go/cmd/gdoc/unknownfield_test.go`

- [ ] The list is decoded into raw entries first, then each entry strictly, so
      a refusal names its entry: `proposals[0]`.
- [ ] `TestAnUnknownFieldIsRefusedWithTheFieldsTheEntryTakes`: a words entry
      carrying `reason` is refused with `proposals[0]`, `"reason"`, and the
      literal lists `quoted, replacement, why, assignee` and
      `kind, after, replace_from, replace_to, content, why, assignee`. Nothing
      reaches Google.
- [ ] `TestTheFieldListsNameEveryTag`: the words of both lists, together, are
      exactly the JSON tags of `propose.Proposal`, so the list cannot drift
      from the struct.
- [ ] The same for `annotate --from`:
      `TestAnUnknownAnnotationFieldIsRefusedWithTheFieldsItTakes`.
- [ ] `TestProposeRefusesAnUnknownFieldInABlockEntry` still passes unchanged.
- [ ] `git commit -m "fix(cmd): an unknown field is refused with the fields the entry takes"`

### Task 5: a deletion is refused with the way to propose it

**Files:**
- Modify: `go/internal/propose/propose.go`, `go/internal/propose/propose_test.go`

- [ ] `TestADeletionRefusalSaysToQuoteANeighbour`: the refusal for an empty
      replacement contains "quote a neighbouring word with it, and keep that
      word in the replacement".
- [ ] `git commit -m "fix(propose): a deletion is refused with the way to propose it"`

### Task 6: a refusal during the run names its proposal

**Files:**
- Modify: `go/cmd/gdoc/write.go`
- Create: `go/cmd/gdoc/proposalnumber_test.go`

- [ ] `TestARunTimeRefusalNamesTheProposalByNumber`: the second of two words
      proposals quotes text that is not in the document; the error opens with
      `proposal 2 (` and the quote, and the first entry reports `sent: true`.
- [ ] `git commit -m "fix(cmd): a refusal during the run names its proposal"`

### Task 7: the `--from` summary names `why`

**Files:**
- Modify: `go/cmd/gdoc/commands.go`, help goldens if any carry the line

- [ ] `TestTheFromSummaryNamesEveryRequiredField`: the summary of `propose
      --from` contains `why` and `quoted, replacement`.
- [ ] `git commit -m "fix(help): the propose --from summary names why"`

### Task 8: propose reads the document once before its first write

**Files:**
- Modify: `go/internal/propose/propose.go`, `go/internal/propose/blockapply.go`,
  `go/cmd/gdoc/write.go`, `go/internal/propose/doc.go`
- Create: `go/cmd/gdoc/proposeread_test.go`

- [ ] `propose.ApplyFrom(ctx, s, docID, p, d)` takes the read the caller
      already holds for the first proposal; `nil` reads, as `Apply` does now.
      Every later proposal still reads fresh.
- [ ] `TestProposeReadsTheDocumentOnceBeforeTheFirstWrite`: a run of one
      words proposal makes exactly one Docs GET before its batchUpdate.
- [ ] `TestEveryLaterProposalStillReadsFresh`: a run of two makes a GET
      between the first proposal's read-backs and the second's write.
- [ ] `git commit -m "refactor(propose): one read before the first write, not two"`

<!-- Parts B and C: Tasks 9 onward, merged from the discovery draft. -->

## Post-Completion

- Nail's acceptance, then `make tag VERSION=v2.11.0` after the signed release
  flow exists.
