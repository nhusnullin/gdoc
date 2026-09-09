# gdoc v2 Milestone 7b: The in-place restyle

## Overview

This is the milestone that opens the direct-edit door. Every write gdoc has made until now has been a suggestion, held there by the guard: a `batchUpdate` on a handed-in document is refused inside the process unless the body says `writeMode: SUGGEST`. M7b lets one document, for one run, be styled directly, and then amends the four places in the repo that currently promise it never will be.

`gdoc restyle <url> --from survey.json` takes the survey M7 emits, rechecks that the document has not moved, and applies the house style in place. The comments keep their authors and their anchors, the pending suggestions stay pending, the smart chips keep their data, and the URL does not change. The three things no Docs request can create are reported, not written into the document.

**Two measurements taken on 2026-09-09 settle what this milestone can promise**, and both are in the repo rather than in somebody's memory.

**What in-place styling reaches** (`go/internal/live/fidelity_test.go`, 13 request kinds against a real document):

| Lands | Refused |
|---|---|
| `updateDocumentStyle` (margins) | `updateNamedStyle` (no such request kind) |
| `updateParagraphStyle` (`namedStyleType`) | `updateParagraphStyle` (`tabStops`, read-only) |
| `updateParagraphStyle` (spacing, indent) | `createHeader` (`FIRST_PAGE_HEADER`) |
| `updateParagraphStyle` (`borderBottom`) | |
| `updateTextStyle` (font, size) | |
| `updateTextStyle` (`foregroundColor`) | |
| `updateTextStyle` (`backgroundColor`) | |
| `updateTableCellStyle` (shading, padding) | |
| `createParagraphBullets` | |
| `createNamedRange` | |

**The limit is durability, not fidelity, and that is the sentence the skill tells Nail before a restyle.** The nine named styles cannot be redefined. But a paragraph can be assigned to `HEADING_1` and its look overridden per paragraph, which is exactly what the master template does and why eight rows sit in `drift.Known`. So the document ends up looking right. The next heading the author types will not.

**The acceptance copy can be made through the API** (`tools/copyprobe`, same day). `files.copy?copyComments=true` carries the comment, still anchored, and the pending suggestion with it; without the parameter it carries neither, which is exactly what was measured in August. So the ten-feature run is unattended, and no copy is made by hand.

Decisions Nail took on 2026-09-09:

- **`LevelInPlace` gets a request-kind allowlist.** Only the styling kinds carry. `deleteHeader`, `deleteContentRange`, `replaceAllText` and `deletePositionedObject` are refused by name.
- **The scope came from the measurement**, not from reading `house.yaml`.
- **The finishing checklist carries the 🤖 mark**, like everything else gdoc writes.
- **The acceptance copy is made through the API**, not in a browser.
- **The restyle keeps the structure and changes the look.** A paragraph already marked `HEADING_1` stays `HEADING_1` and gets the house look; body stays body. gdoc never infers structure from text and never restructures somebody's document. `build` maps markdown headings to styles and a restyle has nothing to map from, so this is the answer to what would otherwise be an unasked question.
- **Typography only. gdoc changes no text at all.** Page geometry, paragraph and text styling, tables and lists. No cover, no front-matter tables, no legend, no contents list, and **no heading numbering**: numbering means writing into the author's prose, and Nail's decision is that a restyle does not do that.
- **No finishing checklist is written into the document.** SPEC has gdoc write a page listing the three things the API cannot create. It reports them instead, and the skill tells Nail. This is Nail's decision of 2026-09-09 and it is also the M2 line held: the binary prints facts, the skill judges.

**Those two decisions together give this milestone its defining property, and it is worth stating as the headline rather than as a detail.** The allowlist at `LevelInPlace` is five request kinds:

```
updateDocumentStyle  updateParagraphStyle  updateTextStyle
updateTableCellStyle  createParagraphBullets
```

**None of them can change a single character.** Every one was measured landing on 2026-09-09. `insertText` and `createNamedRange` were on this list in the previous draft, for the checklist and the heading numbers, and both are gone with them. So the rule the guard enforces is not "these are the styling kinds we thought of": it is that a granted document's **text is unreachable**, and a test states exactly that.

Spec: `docs/v2/SPEC.md` ("`restyle`", acceptance item 5). Master plan: `docs/v2/PLAN.md`, section M7b. Predecessor: `docs/plans/completed/2026-09-09-gdoc-v2-m7-survey.md`, whose closing section is this milestone's handover list.

## Context (from discovery)

- **The measurement everything rests on.** `DECISIONS.md` head decision: replacing a document's body destroyed 100% of comment anchors, 355 of 355 characters across three anchors, recreated every suggestion id and flattened every chip. In-place `batchUpdate` preserved anchors 100% across 22 batches. That is why this milestone exists at all.
- **`comments.list` reports a destroyed anchor as healthy.** It keeps returning the original `anchor` and `quotedFileContent`. Nothing here may verify an anchor through Drive. The docx export is the witness, and `internal/docx`'s `Match` already provides it, answering `anchored`, `detached` or `unmatched`.
- **The witness has two limits**, already written into `internal/restyle`'s package comment by M7: it detects a destroyed anchor and never a moved one, and two exported comments that share words and disagree get no answer.
- **The before-witness is already in the survey.** M7 put it there precisely so this milestone can tell a thread that was already detached from one the restyle broke.
- **The guard today.** Two levels, `LevelSuggest` from `AllowFile` at two call sites, `LevelFull` only from `Learn` after a create the guard carried. `AllowReject(id)` is the shape this milestone's grant should imitate: per-run, one object, dies with the process.
- **`judgeRequests` carries unknown request kinds by design**, and its own comment justifies that on the grounds that an allowlist "would also be empty today, because M1 has no batchUpdate call site". That argument expires here. What bounded unknown kinds all along was `isSuggestMode`, and `LevelInPlace` removes that bound.
- **`deleteHeader` on a first-page header is a one-way door.** DECISIONS.md: it succeeds, and `createHeader` then answers "Default header already exists" with `firstPageHeaderId` gone for good. An in-place restyler installing a house header is exactly the code that reaches for it.
- **`files.copy` is refused today.** `driveShape` returns "" for `{id}/copy`, so it falls through to `judgeDrive`'s final refusal. It needs a shape, a params allowlist including `copyComments`, and the response learning that puts the new id in the set.
- **`docs` does not carry what the styling needs**: no table start index (`docs.Table` is `[][]Cell` with no index), no section breaks, no document style, no existing run styles.
- **`writeControl.requiredRevisionId` is in the public discovery document**, unlike `writeMode`. It closes the window between the survey's recheck and the first write, which a read-then-compare cannot.
- **Named ranges are keyed by `namedRangeId`**, decoded by M7. Duplicate names coexist and deleting by name deletes all of them.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test` runs `-race`; keep it there.
- **CRITICAL: Task 1 is the most dangerous change in the project.** It opens the direct-edit door shut since M1. Every test there is written as an attack first, and the allowlist is the point: a kind not on it is refused whatever it is called.
- **CRITICAL: the grant is one id and one run.** It dies with the process, like `AllowReject`. Nothing persists it, nothing reads it from a file, no flag turns it on for every document.
- **CRITICAL: the read-back is the only bar left.** Under `LevelSuggest` there were two, the guard's refusal and `writeMode`. Once direct edit is open for one id, that id has no second bar. What replaces it is reading the document back.
- **CRITICAL: never verify an anchor through Drive.** `comments.list` reports a destroyed anchor as healthy. The docx export is the witness, always.
- **CRITICAL: the write loop is read-recompute-send.** Every batch that inserts or deletes shifts the indexes later batches were computed from. The measured run was 22 batches. A pure function computing every batch once is a correctness bug, not a simplification.
- **CRITICAL: facts only in Go.** The apply reports what it sent and what the read-back found. Whether the result is good is Nail's, in the document.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces, code comments, commit messages and the docs included.

## Testing Strategy

- **Unit, `internal/guard`**: the third level and the request-kind allowlist, attacks first. The two deleted `GrantInPlace` tests return adapted. `TestAHandedInDocumentIsNeverDirectlyEdited` is narrowed rather than deleted: it still holds for every id the grant did not name.
- **Unit, `internal/guard`** again for the copy shape: the parameters allowed, the folder check, and the id learned from the answer.
- **Unit, `internal/docs`**: the enrichment, against fixtures.
- **Unit, `internal/restyle`**: the request builder as a pure function of one document and the house style, and the loop's recompute behaviour against a scripted session.
- **Unit, `internal/checklist`**: the block's requests, the 🤖 mark, and the named range captured by id.
- **Unit, `cmd/gdoc`**: strict argument parsing, the survey recheck refusals, and the envelope on each failure path.
- **Live, opt-in behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`**: the ten-feature preservation run, which copies with `copyComments=true`, verifies the copy holds all ten before restyling, restyles, asserts all ten again, asserts the original's `revisionId` is unchanged, and trashes the copy.
- **Boundary**: `allowedModules` does not change. M7b adds no dependency.
- Coverage standard: every exported function under `go/internal/` has a test; the new packages at or above 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build` and `make dist`, with the per-platform binary sizes recorded in Task 11

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

`gdoc restyle <url> --from survey.json` does seven things in order:

1. Read the survey strictly. An unknown key is refused by name, and a survey naming another document is refused.
2. Read the document fresh. Refuse if the `revisionId` has moved, if there is more than one tab, or if the counts have changed in a way the survey did not describe.
3. Open a policy granting `LevelInPlace` on that one id, and nothing else.
4. Send the styling batches, each after its own fresh read, each carrying `writeControl.requiredRevisionId` from the last answer.
5. Write the finishing checklist under a named range, if anything is on it.
6. Read the document back, and export it, and compare against the survey's before-witness.
7. Report what was sent, what held, and what did not.

Key design decisions and why:

- **A third level, narrower than the one that was deleted.** `LevelInPlace` permits a `batchUpdate` without SUGGEST on the granted id, and only for the request kinds on the allowlist. It does not permit a file `PATCH`, so restyle cannot trash or rename the document it is styling, which `LevelFull` would have allowed.
- **The allowlist replaces `isSuggestMode` as the bound.** `judgeRequests` carries unknown kinds because SUGGEST bounded them. At `LevelInPlace` nothing bounds them, so the allowlist is what stands in its place, and `deleteHeader` is refused by name because it is a one-way door.
- **`requiredRevisionId`, not read-then-compare.** A recheck followed by a write leaves a window, and across 22 batches that window is the whole run. Docs will refuse the batch itself if the document moved.
- **The checklist carries 🤖.** It is the first text gdoc writes into a document body, and every other thing gdoc writes is marked. A reader finding that page in six months should be able to tell what put it there.
- **The copy is made with `copyComments=true`**, measured the same day. The acceptance verifies the copy holds all ten features **before** restyling, so a setup failure cannot be read as a restyle failure.
- **The original is never written to.** The 2026-08-29 rule. The acceptance re-reads it at the end, on every path including failure, and asserts the `revisionId` is unchanged.

## Technical Details

**Principles.** Serves 3, and tests it hardest. **Principle 3's own wording changes here**, and Task 9 changes it rather than letting the repo assert something its code contradicts. Four places say a handed-in document is never direct-edited: PRINCIPLES.md's principle 3, CLAUDE.md's Never list, SPEC.md's Never list, and SPEC.md's acceptance item 1. All four become false when this ships, and all four are amended with the date and whose decision it was.

**Global constraints:**

- Exactly one JSON object on stdout, exit 0 if and only if `ok`. No prompting, no stdin.
- `--from` is required for the apply. An apply with no survey is refused naming the flag: the recheck is what makes the write safe.
- A document with more than one tab stops the run before the grant is opened. A style request names a range, and a range means nothing without saying which tab it is in.
- Strict argument parsing, as everywhere else.

## Implementation Steps

---

### Task 1: the third level, and an allowlist that cannot touch text

- [ ] **Write the allowlist as a literal.** Five kinds, every one measured landing on 2026-09-09, and nothing else carries at `LevelInPlace`:
  `updateDocumentStyle`, `updateParagraphStyle`, `updateTextStyle`, `updateTableCellStyle`, `createParagraphBullets`.
- [ ] **Pin the property, not just the list.** A test named for it asserts that every kind able to insert, delete or replace content is refused at this level: `insertText`, `deleteContentRange`, `replaceAllText`, `replaceNamedRangeContent`, `insertTable`, `insertPageBreak`, `insertInlineImage`, `replaceImage`, `deleteTableRow`, `mergeTableCells`. A future reader adding a kind has to answer that test, not just the list.
- [ ] **The allowlist is scoped to `LevelInPlace` alone.** Applied globally it breaks `probe`'s direct `insertText` at `LevelFull` and `propose`'s SUGGEST batches. Pin it at all three levels.
- [ ] Also refused at this level, each for its own reason:
  - `deleteHeader` and `deleteFooter`, one-way doors DECISIONS.md records: `createHeader` then answers "Default header already exists" and `firstPageHeaderId` is gone for good
  - `replaceNamedRangeContent`, which deletes and replaces the content of *every* range wearing a name
  - `mergeTableCells`, which destroys cell structure `unmergeTableCells` cannot reliably restore
  - `replaceImage`, the one kind that would silently falsify the acceptance's byte-identical image
  - `deletePositionedObject`, `deleteNamedRange`, `deleteParagraphBullets`
  - every kind carrying "suggestion", `rejectSuggestion` included unless separately granted
  - a request kind nobody has heard of, which is the inversion of `judgeRequests`' unknown-kinds rule and the whole point of an allowlist here
- [ ] **Note what an allowlist by kind cannot see: the `fields` mask.** `updateDocumentStyle` reaches `defaultHeaderId`, `firstPageHeaderId` and `useFirstPageHeaderFooter`, not only margins. Whether that needs bounding is a decision to write down rather than discover.
- [ ] `GrantInPlace(id)` upgrades only an id already in `files`, and the deleted `TestGrantInPlaceNeverAdmitsAnUnknownID` returns.
- [ ] **`AllowFile(id, LevelInPlace)` must be impossible**, not merely untested: it takes any level today and is the side door around the grant's own invariant.
- [ ] **Five call sites change.** `policy.go`'s level check on `batchUpdate`; its refusal text saying "only SUGGEST is allowed"; `Level.String()`, which would otherwise print `unknown(N)` into a user-facing refusal; `AllowFile`; and **`judgeDrive`'s final refusal**, which says "only a file gdoc created may be changed in place" and becomes false.
- [ ] Comment that the levels are **names, not a ladder**: every comparison is `==`, which is why the file `PATCH` stays refused, and a later `>=` would widen everything silently.
- [ ] Narrow `TestAHandedInDocumentIsNeverDirectlyEdited` rather than deleting it.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): a third write level whose allowlist cannot touch text"`

### Task 2: `files.copy` as a per-run grant, not a capability

The first draft made copying a capability of any policy holding a create folder. `cmdPropose` is exactly that shape: `AllowFile(docID, LevelSuggest)` beside `AllowCreateIn(probeFolder)`. So every `propose` run on a document Nail handed in could have copied it and learned the copy at `LevelFull`, which is direct edit with no allowlist and `PATCH`. Wider reach than this milestone's own door, from a document somebody only wanted read.

- [ ] `Policy.AllowCopy(sourceID)` in the shape of `AllowReject` and `AllowCreateIn`: per-run, one source, dying with the process. A copy of any other id is refused, and a policy without the grant carries no copy at all.
- [ ] **The transport half, which `driveShape` alone does not cover.** `isCreate` keys the parent check and `learnFromCreate` on `filesCollection(u.Path)`, and `{id}/copy` is not the files collection, so a copy would otherwise be carried with no parent check and no id learned.
- [ ] A copy omitting `parents` lands in the **source's** parent, a folder gdoc was never given, so the existing `len(parents) != 1` rule must be reached rather than skipped.
- [ ] Parameters: `copyComments`, `supportsAllDrives`, `fields`, nothing else.
- [ ] Record that this door exists for the acceptance run, and that M2's rule about doors without production callers is met because Task 10 is that caller.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): a per-run grant to copy one document, comments included"`

### Task 3: `docs` learns only what a named caller reads

- [ ] Test first, against fixtures: a table's start index, and the text style already on a run. **Both have named callers**, Task 5 and Task 4.
- [ ] **Section breaks and document style are not decoded unless a task reads them.** M2's rule, the one this plan's Task 2 is written under.
- [ ] **State the effect on `read`'s golden files.** `internal/view` walks these blocks to produce a projection the project treats as a specification, so moved bytes are the change and the goldens say so.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): docs carries the index and the style a write needs"`

### Task 4: the request builder, document and paragraphs

- [ ] Test first: given one `docs.Document` and the house style, the page geometry, and per paragraph its spacing, indent, border and run styling. Every expected value is a **literal**, never read from `house.yaml`: the project's standing rule for a house-style test.
- [ ] **A paragraph's `namedStyleType` is read, never decided.** Keep the structure, change the look. Nothing infers structure from text.
- [ ] Apply the look **per paragraph**, because `updateNamedStyle` does not exist. Measured, not assumed, and the reason the master states a heading colour on the style and overrides it on every paragraph.
- [ ] **The `highlight` name-to-hex mapping.** `house.Validate` accepts the seventeen OOXML names and Docs needs an `RgbColor`, so the mapping is real work with a named owner here.
- [ ] Record in the package comment what cannot be reached: tab stops, the first-page header, redefining a named style.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the house style as document and paragraph requests"`

### Task 5: tables and lists

- [ ] Test first: table cells shaded and padded as house.yaml states, addressed by the table's own start index; bullets applied with the preset whose glyphs match the house style's.
- [ ] `createParagraphBullets` takes a preset, not glyphs, so name the preset chosen and the glyphs it produces.
- [ ] **A list gdoc cannot reach with a preset is reported, never rebuilt.** Rebuilding would need `deleteParagraphBullets` and `insertText`, and neither is on the allowlist.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): tables and lists, in place"`

### Task 6: the apply loop, read-recompute-send

- [ ] Test first, over a scripted session: each batch computed from a fresh read and carrying `writeControl.requiredRevisionId` from the last answer; a batch Docs refuses on a stale revision reported as what it is and never retried.
- [ ] **Refuse an empty `requiredRevisionId` at the call site.** If a read did not carry one the body ships `""`, and the only protection this milestone has disappears silently.
- [ ] **Say how batches are sized.** `maxPeek` is 1 MB and `judgeRequests` refuses a `batchUpdate` it cannot read whole, so a per-paragraph pass on a long document can approach it and the refusal would name the ceiling rather than the document.
- [ ] **Write down what a failed run leaves behind.** There is no rollback, by design, because nothing on the allowlist can undo a style. A run failing at batch twelve of twenty-two leaves a half-styled document, and the recovery is Docs version history by hand. **The one comfort, and it is the reason the scope narrowed: no text was touched, so nothing the author wrote can be lost.** That sentence goes in the report and in CLAUDE.md.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the apply loop, recomputed between batches"`

### Task 7: the command, and the line that opens the grant

- [ ] Test first in `go/cmd/gdoc/restyle_test.go`: strict argument parsing; the survey read strictly and refused for an unknown key, another document, a moved `revisionId` or a second tab, **each before the grant is opened**; and the envelope on each failure path.
- [ ] Implement the apply half of `cmdRestyle`, including the `GrantInPlace` call site, which is the most security-relevant line in the milestone and belonged to no task in the first draft.
- [ ] Replace the hard refusal `cmd/gdoc/restyle.go` currently gives without `--dry-run`, which names M7b as future work.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): gdoc restyle --from, and the grant it opens"`

### Task 8: the read-back, and the three things gdoc could not do

- [ ] Test first: thread counts and per-thread witness before and after, pending suggestion ids before and after, chips before and after. A count that moved is reported, never explained.
- [ ] **Nothing asks Drive whether an anchor survived.** The witness is the docx export, and the before-witness comes from M7's survey rather than being recomputed.
- [ ] **Report the three things no Docs request can create**, with the exact menu path for each: the first-page header with its logo, the contents list, and the footer page numbers. SPEC has these written into the document; Nail's decision of 2026-09-09 is that they are reported and the skill tells him. Nothing is written into the document body.
- [ ] `verified` is the checks together, and fewer than all of them is `ok: true` with `verified: false` and the route named, exactly as `propose` and `publish` report.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the read-back, and the manual steps reported not written"`

### Task 9: the promises that stop being true, and one record that is already wrong

- [ ] `PRINCIPLES.md` principle 3, `CLAUDE.md`'s Never list, `docs/v2/SPEC.md`'s Never list, **and SPEC's acceptance item 1**, which says a direct edit on a handed-in id is refused.
- [ ] **Three more places**: CLAUDE.md's guard section saying `LevelSuggest` is what a handed-in id gets, its "`GrantInPlace` is gone until M7b" section, and `policy.go`'s comment about the grant returning at M7b.
- [ ] **Correct DECISIONS.md's 2026-09-09 entry.** It still says "Three things this run did not test": `updateTableCellStyle`, tab stops and the first-page header. Commit `a5fc6e0` measured all three and only its message says so, so the repo's own record contradicts this plan's Overview table.
- [ ] **Record the scope decisions and what they bought**: typography only, no text touched, no checklist written, and the resulting property that the allowlist cannot change a character.
- [ ] **Record that SPEC's finishing checklist is not built**, with the reason, so a later reader does not treat it as an oversight.
- [ ] Record that restyle has **no capability probe**: SPEC requires one before a write whose bar is a client-supplied field, and `LevelInPlace` has no such field, so the read-back stands alone. A decision, not an omission.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "docs: the direct-edit door, and the promises it amends"`

### Task 10: the ten-feature preservation run

- [ ] `TestLiveRestylePreservesTenFeatures` in `go/internal/live`, behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1` and needing `GDOC_LIVE_IDEAL_DOC_ID`.
- [ ] Copy with `copyComments=true` under `AllowCopy`. **Verify the copy holds all ten before restyling**, and fail the setup rather than the restyle if it does not.
- [ ] **Restyle the copy at `LevelInPlace`, not at the `LevelFull` the copy is learned at.** Open a fresh policy, hand the copy id in at `LevelSuggest`, then `GrantInPlace` it. Otherwise the acceptance exercises a path nobody runs and would pass with `LevelInPlace` completely broken.
- [ ] Assert all ten. Re-read the **original** and assert its `revisionId` is unchanged, on every path including failure. Trash the copy.
- [ ] Record the result under Post-Completion. A feature that does not survive is a decision written down with the difference, not a test loosened.
- [ ] `git commit -m "test(v2): the ten-feature preservation run"`

### Task 11: documentation and the size delta

- [ ] CLAUDE.md and the README: restyling a document gdoc did not create, the durability sentence the fidelity measurement earned, and what a failed run leaves behind.
- [ ] `docs/v2/PLAN.md`: the M7b landed paragraph, the "What M7b leaves for M8" list, and the binary-size table.
- [ ] Delete `tools/copyprobe`, whose question is answered and whose own doc comment says so. `tools/tlsdiag` stays: its question recurs.
- [ ] Move this plan to `docs/plans/completed/`.
- [ ] `git commit -m "docs: M7b landed"`

## Post-Completion

- Nail restyles a real document he cares about and reads the result.
- The ten-feature run read by a person, with any feature that did not survive recorded as a decision.
- The binary-size delta per platform.
- Still outstanding, and Nail's: M4's live session with a second account, M5's docx opened in Word, M6's live publish and live drift table, and M7's live check that a real document agrees with `elements.json`.

## What M7b leaves for M8 and later

- **The finishing checklist**, SPEC's design, not built. The three items are reported instead. If a terminal report turns out to be too easy to lose, writing them into the document is a decision with a plan of its own, and it would need `insertText` back on the allowlist.
- **Heading numbering**, dropped for the same reason: it writes into the author's prose.
- **`--new`**, still deferred, and with it the front-matter state transition.
- **The nothing-to-protect offer.** `NothingToProtect` is a fact M7 emits, and SPEC has gdoc offer `--new` as the better route. With `--new` deferred there is nothing to offer, so the offer is deferred rather than half-built.
- **The suggestions walker**, still Nail's: whether element-only suggestions reach `pending`, `gone_since_last_look` and the snapshot.
