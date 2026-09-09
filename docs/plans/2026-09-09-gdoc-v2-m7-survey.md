# gdoc v2 Milestone 7: The survey, and the elements gdoc drops

## Overview

M7 was one milestone in PLAN.md and is now two. This is the first half, and it writes to no document at all.

`gdoc restyle <url> --dry-run` reports what a document holds: its threads with a witness for each, its pending suggestions, its smart chips, its tabs, its named ranges and its revision id. That survey is what the second half hands back before it writes, so the shape it takes here is a contract rather than a convenience.

Getting there means fixing something older. `run()` in `internal/docs` returns `(Run{}, false)` for any paragraph element that is not a text run, an inline object, a footnote reference or an equation. A person chip, a date chip and a calendar link are dropped at decode: no run, no placeholder, no warning. **That is a live defect in `read`, not a gap in the survey.** Every review session since M2 has been reading documents with holes in them and being told nothing. The survey cannot honestly count chips until the decoder stops throwing them away, and the nothing-to-protect detection is defined as no comments, no suggestions and no chips.

Decisions Nail took on 2026-09-09, before this plan:

- **M7 is split.** This milestone is the survey, the decoder and the reads. The in-place write, the guard's third level, the styling engine and the ten-feature run are M7b. Nothing here writes to a Google Doc.
- **The guard level moves with its caller.** `LevelInPlace` is not added here. PLAN.md M2's rule is that a guard door with no production caller is deleted rather than carried, which is why `GrantInPlace` went; adding it a milestone early would repeat that.
- **`--new` stays deferred**, as decided earlier the same day: it needs a markdown export, media extraction and a markdown writer, none of which exist in Go.

Spec: `docs/v2/SPEC.md` ("`restyle`", the survey-first paragraph). Master plan: `docs/v2/PLAN.md`, section M7. Predecessor: `docs/plans/completed/2026-09-08-gdoc-v2-m6-publish.md`. The review that split this milestone is recorded in `docs/v2/DECISIONS.md` under today's date.

## Context (from discovery)

- **Chips are dropped at decode.** `go/internal/docs/walk.go:180`, `run()`, has four cases and a `default` that returns false. `rawParaElement` at `:104` names no `person` and no `richLink`. Because the element vanishes rather than becoming a run, it reaches neither `view`'s placeholder list nor its warnings, unlike an unclassifiable inline object which at least prints `[object]`. No fixture under `internal/docs/testdata/` holds a chip.
- **Seven members are dropped, not three.** `ParagraphElement` is a union of eleven and `run()` handles four. Beyond the chips, `autoText`, `pageBreak`, `columnBreak` and `horizontalRule` also vanish, so a page break and a rule are missing from `read` today. The reference documents every field of all seven, so the fixture is built rather than measured.
- **Named ranges are not decoded either.** Nothing in v2 reads one. What is recorded is the measurement: a range tracks edits exactly and shrinks rather than sliding onto foreign text, and **the key is `namedRangeId`, not the name**, because duplicate names coexist and deleting by name deletes all of them. With `includeTabsContent=true` the field is at `tabs[].documentTab.namedRanges`, not at the top level.
- **`docsReadParams` already permits a `fields` mask**, so `documents.get(fields=namedRanges)` passes the guard unchanged. That read measured 982 bytes against 12,907 for the whole document.
- **The anchor trap.** `comments.list` still returns the original `anchor` and `quotedFileContent` for a comment whose anchor was destroyed, so Drive reports every broken anchor as healthy. The docx export is the only honest witness, and `internal/docx`'s `Match` already provides it, joining on the comment's words plus its author's name and answering `anchored`, `detached` or `unmatched`.
- **The witness has two limits worth stating rather than discovering.** It detects a destroyed anchor, not a moved one: an anchor that slid onto foreign text still reads `anchored`. And `Match` deliberately answers `unmatched` when two exported comments share words and disagree, so a document with duplicated comment bodies gets no answer for either.
- **`suggestions.List` drops whitespace-only suggestions and `suggestions.All` does not.** The survey must count from `All`: a whitespace-only suggestion is still something a replacement would destroy, so a count from `List` would report nothing-to-protect on a document that has something to protect.
- **Type names to use, since the review found the draft inventing them:** `suggestions.Pending`, not `suggestions.Item`. `docs.NamedRange` does not exist yet and Task 3 adds it.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test` runs `-race`; keep it there.
- **CRITICAL: nothing in this milestone writes to a Google Doc.** No `batchUpdate`, no guard change, no new write level. The one file it may touch is the OAuth token, which any command replaces on refresh. A task that finds itself wanting to write has found M7b's work.
- **CRITICAL: Task 2 changes what `read` prints.** That is the point, and it is a behaviour change in a command the review skill depends on. Golden files move, and the moved bytes are the specification of the fix.
- **CRITICAL: facts only in Go.** The survey reports counts, ids and a witness per thread. Whether a document is worth restyling is the skill's and Nail's, and `nothing_to_protect` is a fact about three counts being zero, never a recommendation.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces, code comments, commit messages and the docs included.

## Testing Strategy

- **Fixture, Task 1**: built from the documented `ParagraphElement` union, covering the seven members `internal/docs` drops. Not a measurement: the reference names every field. The live check that a real document agrees is Post-Completion.
- **Unit, `internal/docs`**: the chip fixture decodes, each kind becomes a run with its own `Kind` and its suggestion id lists; an element the walk cannot name becomes a placeholder rather than vanishing; named ranges decode from the per-tab location, keyed by id.
- **Unit, `internal/view`**: each new kind prints a placeholder and warns. Golden files change.
- **Unit, `internal/propose`**: a quote crossing a chip is refused, and now for the right reason. A chip used to be a hole the walk skipped; it is a non-text run now, and the contiguity check has to hold either way.
- **Unit, `internal/restyle`**: the survey over fixtures, `nothing_to_protect` true only when all three counts are zero, a thread the Docs read placed no range for surveyed with a warning rather than a failure, and the per-thread witness carried through.
- **Unit, `cmd/gdoc`**: strict argument parsing, the envelope, and the policy opened at `LevelSuggest` with no grant.
- **Boundary**: `allowedModules` does not change. M7 adds no dependency.
- Coverage standard: every exported function under `go/internal/` has a test; `internal/restyle` at or above 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build` and `make dist`, with the per-platform binary sizes recorded in Task 7

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

`gdoc restyle <url> --dry-run` opens a policy holding the document at `LevelSuggest`, and does four reads: the document with `includeTabsContent=true`, the comment listing, the named ranges, and the docx export for the witness. It prints the counts, the revision id and a witness per thread. Nothing is written.

Key design decisions and why:

- **The witness belongs in the survey, not in the apply.** M7b compares before with after, and a thread that was already detached before the restyle would otherwise read as damage the restyle did. The before-witness has no other home, so the survey pays for the export.
- **The revision id is why the survey is machine-readable.** M7b hands it back and refuses a document that moved. That is principle 3 at the one moment gdoc will have the power to overwrite, so the field exists here even though nothing reads it yet.
- **A chip is a run, not a hole.** Decoding it fixes `read` for every command, makes the survey's count honest, and keeps the index arithmetic `propose` depends on describing what is actually in the document.
- **The `default` arm reports rather than vanishes.** This is wider than chips and it is the real fix: the next element kind Google adds shows up as a placeholder with a warning instead of silently absent. A decoder that drops what it does not recognise makes every reader downstream confidently wrong.
- **Named ranges are keyed by id.** Duplicate names coexist and deleting by name deletes all of them, so a name is not an identifier here.

**`gdoc restyle --dry-run` output:**

```jsonc
{
  "document_id": "1AbC...", "revision_id": "ALm37BX...",
  "title": "Supplier Register Policy",
  "tabs": 1,
  "threads": { "open": 3, "resolved": 7,
               "witness": { "anchored": 9, "detached": 1, "unmatched": 0 } },
  "suggestions": { "pending": 2 },
  "chips": { "person": 4, "date": 1, "rich_link": 2 },
  "named_ranges": [ { "id": "kix.abc123", "name": "gdoc-checklist" } ],
  "nothing_to_protect": false,
  "warnings": []
}
```

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): the measurement, the decoder, named ranges, the survey, the command, the amendments, the documentation.
- **Post-Completion** (no checkboxes): Nail surveying a real document, and the chip fixture read by a person against a document they can see.

## Implementation Steps

---

### Task 1: the fixture, from the documented shape

The Docs API reference **does** document these elements, so this is not a measurement task and it blocks on nothing. `ParagraphElement` is a union of eleven members and `internal/docs` decodes four of them. The other seven are dropped silently:

| Member | Carries | Dropped today |
|---|---|---|
| `person` | `personId`, `personProperties.name`, `.email` | yes |
| `richLink` | `richLinkId`, `richLinkProperties.title`, `.uri`, `.mimeType` | yes |
| `dateElement` | the date chip | yes |
| `autoText` | a page number or a date field | yes |
| `pageBreak` | a page break | yes |
| `columnBreak` | a column break | yes |
| `horizontalRule` | a rule | yes |

Every one also carries `suggestedInsertionIds` and `suggestedDeletionIds`, like every other element, so each needs the same id handling.

- [x] Build `go/internal/docs/testdata/elements.json` from the documented shape, covering all seven, each with its own suggestion id lists. Placeholder text throughout, so nothing in it is anybody's document.
- [x] Write into `docs/v2/DECISIONS.md` under today's date: the seven were dropped, the reference documents all of them, and the fixture is built from the reference rather than measured. Say plainly that the live verification is outstanding, because a fixture built from a reference is a hypothesis until a real document agrees with it.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "test(v2): a fixture for the seven paragraph elements gdoc drops"`

### Task 2: the decoder stops dropping what it cannot name

- [x] Test first in `go/internal/docs/walk_test.go` against the fixture: each of the seven decodes into a run with its own `Kind`, carrying the fields the reference names and its two suggestion id lists.
- [x] Test that an element the walk still cannot name becomes a placeholder run with a warning naming the field, rather than returning false. **This is the wider fix and it is the point of the task.** Seven kinds went missing because the `default` arm vanished rather than reported; the eighth must not.
- [x] Test in `go/internal/view/text_test.go`: each new kind prints a placeholder and raises a warning, the way `[image]` and `[drawing]` do. A `horizontalRule` may deserve `---` rather than a placeholder, and a `pageBreak` a blank line; whichever is chosen, the golden file states it and the changed bytes are the specification.
- [x] Test in `go/internal/propose`: a quote crossing any of the seven is refused. They were holes the walk skipped and are now runs, so the contiguity check has to hold across the change rather than by luck. This is a behaviour change in a shipped write command and it gets its own test.
- [x] Implement: the fields on `rawParaElement`, the cases in `run()`, the reporting `default`, the kinds and placeholders in `view`.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "fix(v2): seven paragraph elements stop vanishing"`

### Task 3: named ranges, keyed by id

- [x] Test first: named ranges decode from `tabs[].documentTab.namedRanges`, not from the top level, because every read carries `includeTabsContent=true`. Duplicate names both decode and keep their own ids.
- [x] Implement `docs.NamedRange` and its decoding. The id is the identifier; the name is a label.
- [x] Test that the read asking only for named ranges passes the guard, which `docsReadParams` already permits.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): named ranges, read and keyed by id"`

➕ The pre-tabs location is read too. `Parse` already has a branch for a document with a top-level `body` and no tabs, and its named ranges sit beside that body. Reading the body from one place and the ranges from another would report none on exactly the documents whose ranges are at the top level.

➕ `Range` gained `Segment`. A named range span names the header, footer or footnote it sits in, and a header span read as a body span names a position the body does not have. It is empty on every comment anchor, so no output shape moved.

### Task 4: `internal/restyle`, the survey

- [x] Test first in `go/internal/restyle/survey_test.go` over fixtures: the counts, the revision id, the tab count, the per-thread witness, and `nothing_to_protect` true only when threads, suggestions and chips are all zero.
- [x] Test that the pending count comes from `suggestions.All` and not `List`: a whitespace-only suggestion is still something a replacement would destroy, so counting from `List` reports nothing-to-protect on a document that has something to protect.
- [x] Test that a thread the Docs read placed no range for is surveyed with a warning rather than failing the listing, and that an export that could not be read leaves every thread `unmatched` with a warning, exactly as `comments --witness` behaves.
- [x] Implement `restyle.Survey`, taking the decoded reads and returning the report. It takes no session and touches no wire, like every other reader package. Use `suggestions.Pending`, which is the type's real name.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the restyle survey"`

➕ `nothing_to_protect` reads a fourth thing: an element the decoder could not name. Task 2 made an unknown paragraph element report itself instead of vanishing, and a document holding one holds something no count here speaks for. The survey warns naming the member and refuses to say there is nothing to protect.

➕ The witness is carried twice, as counts and as one line per thread. M7b compares before with after per thread, which totals cannot answer, and the survey is the only place the before-witness is read.

### Task 5: `restyle --dry-run`

- [ ] Test first in `go/cmd/gdoc/restyle_test.go`: strict argument parsing in every shape, `--dry-run` required for now with a refusal naming M7b for the apply, the envelope on each failure path, and the policy opened at `LevelSuggest` with no grant.
- [ ] Implement `cmdRestyle` in `go/cmd/gdoc/restyle.go`, composing the four reads. Reuse the export path `comments --witness` already builds rather than a second one.
- [ ] Bound the export the way `--wait --witness` does: an export that failed is a warning on an envelope that still carries the survey.
- [ ] `dispatch` gains `restyle`, and the usage string with it.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): gdoc restyle --dry-run"`

### Task 6: the documents that go stale, corrected now rather than at M7b

- [ ] `docs/v2/PLAN.md`: split M7 into M7 and M7b, with what each carries and why the split happened. Record that the guard's third level travels with its caller, per M2's rule.
- [ ] `docs/v2/DECISIONS.md`: the split, the request-kind allowlist Nail chose for M7b's level, and the decision to scope the in-place styling from a measurement rather than from `house.yaml`'s contents.
- [ ] **Do not amend the Never lists yet.** PRINCIPLES.md, CLAUDE.md and SPEC.md all say a handed-in document is never direct-edited, and that stays true for the whole of this milestone. M7b amends all three, SPEC's acceptance item 1 included, which the earlier draft missed.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "docs: M7 splits, and why the write half moves"`

### Task 7: documentation and the size delta

- [ ] CLAUDE.md: the chip decoding and the reporting `default` under the read commands, named ranges, and the survey.
- [ ] README: surveying a document before restyling it.
- [ ] `docs/v2/PLAN.md`: the M7 landed paragraph, the "What M7 leaves for M7b" list, and the binary-size table against the last M6 commit.
- [ ] Move this plan to `docs/plans/completed/`.
- [ ] `git commit -m "docs: M7 landed"`

## Post-Completion

- Nail surveys a real document and reads the counts against what he can see on screen.
- **The live check on the fixture.** A real document holding a person chip, a date chip and a calendar link, read through `documents.get`, compared against `elements.json`. The fixture is built from the reference, and a reference is a hypothesis until a document agrees with it. If it disagrees, that is a decision recorded with the difference, not a test loosened.
- The binary-size delta per platform.
- Still outstanding, and Nail's: M4's live session with a second account, M5's docx opened in Word, and M6's live publish and live drift table.

## What M7 leaves for M7b

- **The guard's third level**, `LevelInPlace`, with the request-kind allowlist Nail chose: only the styling kinds carry, and `deleteHeader`, `deleteContentRange`, `replaceAllText` and `deletePositionedObject` are refused. `deleteHeader` on a first-page header is a one-way door DECISIONS.md already records, and the "an allowlist would be empty" argument in `judgeRequests`' own comment expires the moment SUGGEST stops bounding unknown kinds. The call sites to change are `policy.go`'s level check and its refusal text, `Level.String()`, and `AllowFile`, which takes any level and is the side door around the grant's own invariant.
- **The fidelity measurement.** Nail's decision: what the in-place styling attempts is scoped from a probe that applies each candidate request kind to a throwaway document and records what lands, not from reading `house.yaml`. The 2026-08-29 run measured survival, not fidelity.
- **What in-place cannot reach**, written down as a list rather than the word "approximate": the nine named styles cannot be redefined at all, so styling means applying paragraph and text style over every paragraph one at a time, and the next heading the author types is not house style. `highlight` is an OOXML name with no Docs equivalent, bullet glyphs and number formats are a fixed enum, tab stops are read-only, and the Docs API accepts no image bytes, so even an inline logo is closed.
- **The `docs` enrichment the styling needs**: table start indexes, section breaks, document style and existing run styles, none of which the package carries today.
- **The write loop is read-recompute-send**, not a pure function computed once. Every batch that inserts or deletes shifts the indexes later batches were computed from, and the measured run was 22 batches.
- **`writeControl.requiredRevisionId`** rather than read-then-compare. It is in the public discovery document, unlike `writeMode`, and it closes the window between the survey's recheck and the first write.
- **The checklist**, its named range captured by id, and the decision about whether text gdoc writes into a document body carries the 🤖 mark the way everything else it writes does.
- **The live ten-feature run, and how the copy is made.** `files.copy` is refused by the guard today and drops every comment anyway, so an API copy cannot carry the anchored comment the acceptance asserts. Either the guard gains a copy door and the test builds its comments afterwards, or the copy is made in the browser and named in an environment variable.
- **The amendments**: PRINCIPLES.md's principle 3, CLAUDE.md's Never list, and SPEC.md's Never list **and acceptance item 1**, which says a direct edit on a handed-in id is refused.
- **`--new`**, still deferred, and with it PLAN.md's rule that a document left unpaired is refused by alignment until it is paired on purpose, which is M8's constraint.
