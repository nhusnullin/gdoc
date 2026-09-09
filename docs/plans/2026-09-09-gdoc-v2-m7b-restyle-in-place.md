# gdoc v2 Milestone 7b: The in-place restyle

## Overview

This is the milestone that opens the direct-edit door. Every write gdoc has made until now has been a suggestion, held there by the guard: a `batchUpdate` on a handed-in document is refused inside the process unless the body says `writeMode: SUGGEST`. M7b lets one document, for one run, be styled directly, and then amends the four places in the repo that currently promise it never will be.

`gdoc restyle <url> --from survey.json` takes the survey M7 emits, rechecks that the document has not moved, and applies the house style in place. The comments keep their authors and their anchors, the pending suggestions stay pending, the smart chips keep their data, and the URL does not change. The three things no Docs request can create are reported, not written into the document.

**Corrected 2026-09-09, after the second review.** An earlier draft of this plan claimed the allowed request kinds could not change a single character, and included `createParagraphBullets`. That was false: the reference says leading tabs determine a bullet's nesting level and "these leading tabs are removed by this request", so it deletes text the author typed, and it merges a bulleted range into an adjacent list with a matching preset, renumbering their items. The fidelity probe missed it because its content had no leading tabs. Bullets are dropped, and the property below is now true rather than nearly true.

**Two measurements taken on 2026-09-09 settle what this milestone can promise**, and both are in the repo rather than in somebody's memory.

**What in-place styling reaches** (`go/internal/live/fidelity_test.go`, 13 request kinds against a real document). This is the measurement, not the allowlist: `createParagraphBullets` and `createNamedRange` land and are still refused, for the reasons above.

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
- **The acceptance copy is made through the API**, not in a browser.
- **The restyle keeps the structure and changes the look.** A paragraph already marked `HEADING_1` stays `HEADING_1` and gets the house look; body stays body. gdoc never infers structure from text and never restructures somebody's document. `build` maps markdown headings to styles and a restyle has nothing to map from, so this is the answer to what would otherwise be an unasked question.
- **Typography only. gdoc changes no text at all.** Page geometry, paragraph and text styling, and table cell appearance. **No list styling**, no cover, no front-matter tables, no legend, no contents list, and **no heading numbering**: numbering means writing into the author's prose, and Nail's decision is that a restyle does not do that.
- **No finishing checklist is written into the document.** SPEC has gdoc write a page listing the three things the API cannot create. It reports them instead, and the skill tells Nail. This is Nail's decision of 2026-09-09 and it is also the M2 line held: the binary prints facts, the skill judges.

**Those decisions together give this milestone its defining property.** The allowlist at `LevelInPlace` is four request kinds:

```
updateDocumentStyle  updateParagraphStyle  updateTextStyle  updateTableCellStyle
```

**None of them can change a single character**, and unlike the previous draft that is now literally true. `insertText` and `createNamedRange` went with the checklist and the heading numbers. `createParagraphBullets` went because it removes leading tabs.

**The allowlist bounds the kind. It does not bound the `fields` mask, and that is where this milestone's real danger lives.** The reference: "To reset a property to its default value, include its field name in the field mask but leave the field itself unset." So an `updateTextStyle` carrying `fields: "*"` over a range resets every property it does not set: bold, italic, links, colours, highlights, permanently, **with every character intact**.

An earlier draft filed that as a note to write down. It is a rule the guard enforces, and Task 1 owns it. Saying "the text is unreachable" while leaving the mask unbounded would be reassuring about the wrong thing: what this milestone can destroy is everything except the text.

**What a restyle overwrites, stated rather than discovered.** Applying the house look to a paragraph replaces the run styling the author chose there. That is what a restyle is for, and it is still worth a person knowing before they run one: a hand-bolded phrase inside a body paragraph does not survive a body-text pass. The report says which paragraphs were restyled, and the skill says this sentence.

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
5. Read the document back, and export it, and compare against the survey's before-witness.
6. Report what was sent, what held, what did not, and the three things it could not do at all.

Key design decisions and why:

- **A third level, narrower than the one that was deleted.** `LevelInPlace` permits a `batchUpdate` without SUGGEST on the granted id, and only for the request kinds on the allowlist. It does not permit a file `PATCH`, so restyle cannot trash or rename the document it is styling, which `LevelFull` would have allowed.
- **The allowlist replaces `isSuggestMode` as the bound.** `judgeRequests` carries unknown kinds because SUGGEST bounded them. At `LevelInPlace` nothing bounds them, so the allowlist is what stands in its place, and `deleteHeader` is refused by name because it is a one-way door.
- **`requiredRevisionId`, not read-then-compare.** A recheck followed by a write leaves a window, and across 22 batches that window is the whole run. Docs will refuse the batch itself if the document moved.
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

### Task 1: the third level, the allowlist, and the mask rule

- [x] **Write the allowlist as a literal.** Four kinds, each measured landing on 2026-09-09, and nothing else carries at `LevelInPlace`:
  `updateDocumentStyle`, `updateParagraphStyle`, `updateTextStyle`, `updateTableCellStyle`.
- [x] **Pin the property, not the list.** A test named for it asserts every kind able to insert, delete or replace content is refused: `insertText`, `deleteContentRange`, `replaceAllText`, `replaceNamedRangeContent`, `insertTable`, `insertTableRow`, `insertTableColumn`, `insertPageBreak`, `insertInlineImage`, `replaceImage`, `createFootnote` (it inserts a reference character), `deleteTableRow`, `mergeTableCells`, and **`createParagraphBullets`**, which removes leading tabs. A future reader adding a kind answers that test, not just the list.
- [x] **The allowlist gates every `batchUpdate` on a granted id, whatever `writeMode` says.** `judgeDocs` reads `lvl == LevelFull || isSuggestMode(body)`, so an allowlist hung off the direct-edit branch alone would still let a granted document take an `insertText` under SUGGEST, and the property above would be false on exactly the id it is meant to protect. Pin it with an attack test that sends a SUGGEST `insertText` to a granted id and expects a refusal.
- [x] **The allowlist is scoped to `LevelInPlace` alone.** Applied globally it breaks `probe`'s direct `insertText` at `LevelFull` and `propose`'s SUGGEST batches. Pin it at all three levels.
- [x] **The `fields` mask is a rule here, not a note.** The guard reads it the way `isSuggestMode` reads `writeMode`, exactly and case-sensitively: refuse `*`, refuse an empty mask, which Google reads as every field, and refuse a mask naming `useFirstPageHeaderFooter` or `useEvenPageHeaderFooter`, because switching one off hides the first-page header carrying the logo, the one thing this milestone reports as unreachable. `defaultHeaderId` and `firstPageHeaderId` are read-only in the reference, so they need no rule.
- [x] Also refused: `deleteHeader` and `deleteFooter`, one-way doors DECISIONS.md records; `deletePositionedObject`; `deleteNamedRange`; every kind carrying "suggestion", `rejectSuggestion` included unless separately granted; and a request kind nobody has heard of, which is the inversion of `judgeRequests`' unknown-kinds rule and the whole point of an allowlist here.
- [x] `GrantInPlace(id)` upgrades only an id already in `files`, and the deleted `TestGrantInPlaceNeverAdmitsAnUnknownID` returns.
- [x] **`AllowFile(id, LevelInPlace)` must be impossible**, not merely untested: it takes any level today and is the side door around the grant's own invariant.
- [x] **Five call sites change.** `policy.go`'s level check on `batchUpdate`; its refusal text saying "only SUGGEST is allowed"; `Level.String()`, which would otherwise print `unknown(N)` into a user-facing refusal; `AllowFile`; and `judgeDrive`'s final refusal, which says "only a file gdoc created may be changed in place" and becomes false.
- [x] Comment that the levels are **names, not a ladder**: every comparison is `==`, which is why the file `PATCH` stays refused.
- [x] Narrow `TestAHandedInDocumentIsNeverDirectlyEdited` rather than deleting it.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): a third write level, its allowlist, and its field-mask rule"`

### Task 2: `files.copy` as a per-run grant

The first draft made copying a capability of any policy holding a create folder. `cmdPropose` is exactly that shape, so every `propose` run on a document Nail handed in could have copied it and learned the copy at `LevelFull`: direct edit, no allowlist, `PATCH`.

- [x] `Policy.AllowCopy(sourceID)` in the shape of `AllowReject` and `AllowCreateIn`: per-run, one source, dying with the process.
- [x] **The transport half.** `isCreate` keys the parent check and `learnFromCreate` on `filesCollection(u.Path)`, and `{id}/copy` is not the files collection, so a copy would otherwise be carried with no parent check and no id learned.
- [x] A copy omitting `parents` lands in the **source's** parent, a folder gdoc was never given, so the `len(parents) != 1` rule must be reached rather than skipped.
- [x] Parameters: `copyComments`, `supportsAllDrives`, `fields`, nothing else.
- [x] **Write this down as Nail's decision, not as a rule satisfied.** M2's rule is that a guard door needs a *production* caller, and this one's only caller is a live test. A copy also takes a full duplicate of a handed-in document into gdoc's folder at `LevelFull`, which is the widest reach any handed-in id has produced. `tools/copyprobe` shows a document with an anchored comment and a pending suggestion can be built from scratch; what it cannot build is the image and the Drawing, which is why the door is probably right. It is still a decision.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): a per-run grant to copy one document, comments included"`

### Task 3: `docs` learns the one thing a caller reads

- [x] Test first, against fixtures: a table's start index. Task 5 reads it. `docs.Table` is a struct now, `{StartIndex, Rows}`, rather than the bare `[][]Cell` it was: a table and where it starts are one fact, and a sibling field on `Block` would let a paragraph block carry a table's index. `TestATableCarriesTheIndexAWriteNeeds` reads the 184 in `single-tab.json`, and `TestANestedTableCarriesItsOwnIndex` pins the recursion, because a table inside a cell is an element of that cell's content and must carry its own index rather than the outer table's.
- [x] **Nothing else is decoded.** Section breaks, document style and existing run styles each had no reader once the scope narrowed: Task 4 applies the look with a narrow mask and does not need to know what is there. M2's rule, the one Task 2 is written under. Column widths and row heights are the same answer for a second reason: both need a request kind `LevelInPlace` does not carry.
- [x] **State the effect on `read`'s golden files** if the enrichment moves them. **They did not move.** The text projection prints no index, so every `.golden` under `internal/view/testdata/` is byte-identical. What did move is `read --structure`, where a table block was the rows alone and is now `{"start_index": N, "rows": [[...]]}`. Nothing pinned that shape before, so `TestStructureCarriesATablesStartIndex` states it now.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): docs carries the table index a write needs"`

### Task 4: the document style, and the page

- [x] Test first: page size and margins from the house style, with every expected value a **literal**, never read from `house.yaml`. `internal/restyle/page_test.go` writes out 595.28, 841.89, 62.35, 51, 51.05 and 51.05, and prints the house value beside the want on a failure.
- [x] **Narrow `fields` masks throughout**, naming exactly what is set. Task 1's guard rule refuses `*`, and the builder must never rely on being refused. The mask is the constant `pageMask`, and `TestThePageMaskNamesExactlyWhatItSets` checks both directions: no path in the mask the request leaves unset, and no field set outside the mask. `TestTheGuardCarriesThePageRequest` judges the built request through a policy at `LevelInPlace`, so the builder and the guard cannot drift apart in silence.
- [x] **A document carrying section breaks has its margins governed by `sectionStyle`**, and `updateSectionStyle` is not on the allowlist. So `updateDocumentStyle` can be accepted and invisible. Report it rather than claiming the page was restyled: Task 8 is where that check lives. Written into `PageRequest`'s own doc comment, with the header and footer margins deliberately left unset beside it: gdoc writes no header here, so it does not move the margin one reserves.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the page geometry, in place"`

### Task 5: paragraphs, runs, and table cells

- [x] Test first: per paragraph its spacing, indent and border; per run the house font, size and colour; per table cell the shading, padding and borders house.yaml states. Literals throughout. **Three of those turned out to be values house.yaml does not state, and each is written down rather than invented.** The house style draws **no paragraph border**, so no paragraph request names one: naming `borderBottom` in a mask without setting it is how a restyle would erase a rule the author drew. It states **no cell shading** for a table an author wrote, and which row of somebody's table is a header is not something gdoc can read, so a cell's own fill is left alone and `backgroundColor` is never in a mask. The cell padding and the grey grid are `internal/body/table.go`'s own measured values, which house.yaml does not state either, carried here so a restyled table looks like a table gdoc builds from a note.
- [x] **A paragraph's `namedStyleType` is read, never decided.** Keep the structure, change the look. Nothing infers structure from text. `TestNoRequestWritesANamedStyleType` pins it, and a named style the house has no look for is reported in `Plan.Unstyled` and left untouched rather than mapped to the nearest one.
- [x] Apply the look **per paragraph**, because `updateNamedStyle` does not exist. One `updateParagraphStyle` and one `updateTextStyle` over each paragraph's own range.
- [x] **The house style is written where it states a value, and nowhere else.** Stated means a value the file can tell apart from silence: an optional number that is there, or a non-empty string. **A plain flag is never written**, so `bold`, `italic`, `keep_with_next` and `keep_lines_together` are absent: absent and false are one word in house.yaml, and writing them would clear the emphasis an author put inside a paragraph on the strength of a value the file may never have stated. Alignment is written where a style states one, which is the same rule. **Body prose reads `body:` rather than `styles.normal`**, because `build` writes an ordinary paragraph at the body size and alignment over a Normal style stating 11pt, and a restyle reading the style alone would not match a document gdoc built.
- [x] **A paragraph inside a table cell takes `table_text` and nothing else.** The body's justified alignment is not the look inside a narrow cell, and house.yaml states no spacing for a cell of a table the author wrote, so writing one would be gdoc inventing a value rather than applying the style.
- [x] **Tables are cell appearance only.** Column widths need `updateTableColumnProperties` and row heights need `updateTableRowStyle`; neither is on the allowlist, neither was measured, and both change a table's layout rather than its look, which the scope decision puts out of bounds. Report them as not applied: `Plan.Tables` counts them and Task 8 says so.
- [x] **No list styling.** `createParagraphBullets` removes leading tabs, so lists keep whatever bullets they have. `Plan.Bulleted` counts them and Task 8 reports them alongside the other things gdoc could not do.
- [x] **The mask rule is tested over every request, not per builder.** `TestEveryMaskNamesExactlyWhatItSets` walks the whole plan, so a builder added later answers it too, and `TestTheGuardCarriesEveryRestyleRequest` judges the page request and the whole tab through a policy at `LevelInPlace`.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): paragraphs, runs and table cells"`

### Task 6: the apply loop

- [x] Test first, over a scripted session: each batch carrying `writeControl.requiredRevisionId` from the last answer, and a batch Docs refuses on a stale revision reported as what it is and never retried.
- [x] **Refuse an empty `requiredRevisionId` at the call site.** If a read did not carry one the body ships `""` and the only protection this milestone has disappears silently.
- [x] **Indexes no longer move**, now that nothing on the allowlist changes text, so the loop re-reads for the revision id rather than to recompute positions. Say so: an earlier draft justified the loop by shifting indexes and that reason has gone with the bullets.
- [x] **Say how batches are sized.** `maxPeek` is 1 MB and `judgeRequests` refuses a `batchUpdate` it cannot read whole.
- [x] **Write down what a failed run leaves behind.** No rollback, so a run failing at batch twelve leaves a half-styled document and the recovery is version history by hand. **No text was touched, so nothing the author wrote is lost, but their own run formatting inside restyled paragraphs is.** That sentence goes in the report and in CLAUDE.md.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the apply loop, revision-checked between batches"`

### Task 7: the command, and the line that opens the grant

- [x] Test first in `go/cmd/gdoc/restyle_test.go`: strict argument parsing; the survey read strictly and refused for an unknown key, another document, a moved `revisionId` or a second tab, **each before the grant is opened**; and the envelope on each failure path.
- [x] Implement the apply half of `cmdRestyle`, including the `GrantInPlace` call site, the most security-relevant line in the milestone.
- [x] Replace the hard refusal `cmd/gdoc/restyle.go` gives without `--dry-run`, which names M7b as future work.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): gdoc restyle --from, and the grant it opens"`

### Task 8: the read-back, both halves

- [x] **The preservation half.** Thread counts and per-thread witness before and after, pending suggestion ids before and after, chips before and after. A count that moved is reported, never explained. **Nothing asks Drive whether an anchor survived**: the witness is the docx export, and the before-witness comes from M7's survey. `Preserve` in `internal/restyle/readback.go`. **Two facts the writing settled.** A witness that reads `unmatched` now is reported as `unwitnessed` and never as a lost anchor: unmatched is the export giving no answer, and calling absence of evidence damage is the cry-wolf warning this tool avoids everywhere else. It keeps `verified` false all the same, because nothing then says the anchor survived. And the survey had to learn `suggestions.ids`: two counts that do not move cannot tell one suggestion destroyed and another created from nothing having happened, so `SuggestionCounts` carries the ids the read could see, from both the listing's walk and the elements'.
- [x] **The landed half, which the previous draft had no check for at all.** Read back a margin, a paragraph's spacing, a run's font and a cell's appearance, and report whether the style is actually there. The fidelity probe's whole point was the accepted-but-not-landed row, and a `verified: true` printed over an invisible change would be exactly that failure. `Landed` in `internal/restyle/landing.go`. ➕ **Scope correction: a cell's padding, not its shading.** Task 5 decided a restyle writes no `backgroundColor`, because which row of somebody's table is a header is not something gdoc can read, so reading shading back would be checking a value gdoc never sent. The check reads the cell appearance it does write. **Three more decisions.** The check is made against the requests that were sent rather than against `house.yaml`, because a check written from the house style asks the question the builder already answers and the two drift the first time a builder stops setting a field. It reads the first request of each kind and names where it looked, because a restyle sends one request per paragraph and hundreds of lookups answer one question. And a field the read does not carry at all is the document's own default, so a zero holds where the read is silent and anything else does not: Docs leaves a property equal to its default out of the answer, and reading that silence as a failure would report every zero the house style states as not landed.
- [x] **Report the things gdoc could not do**, with the exact menu path for each: the first-page header with its logo, the contents list, the footer page numbers, plus any list left unstyled and any table whose widths were not touched. `ManualSteps`, with the named styles the house has no look for as a fourth conditional entry.
- [x] `verified` is the checks together, and fewer than all is `ok: true` with `verified: false` and the route named. It is every landing check holding, the preservation half intact, and no thread whose witness stopped answering. A run that wrote nothing reads nothing back, because the document is as it was; a run that stopped half way does read back, because that is the run the preservation facts are most needed for; and a read-back the run could not make is a warning and no read-back at all, because a preservation half built from a listing that never arrived names every thread in the survey as gone.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the read-back, preservation and landing both"`

### Task 9: the promises that stop being true, and two records that are wrong

- [x] `PRINCIPLES.md` principle 3, `CLAUDE.md`'s Never list, `docs/v2/SPEC.md`'s Never list, **and SPEC's acceptance item 1**.
- [x] **Three more places**: CLAUDE.md's guard section, its "`GrantInPlace` is gone until M7b" section, and `policy.go`'s comment about the grant returning at M7b.
- [x] **Correct DECISIONS.md's 2026-09-09 entry**, which still says three request kinds went untested when commit `a5fc6e0` measured all three.
- [x] **Record the `createParagraphBullets` correction.** The fidelity entry lists it as landing, which is true and incomplete: it lands and it removes leading tabs. The probe missed it because its content had none.
- [x] Record the scope decisions, the field-mask rule and why it exists, that SPEC's finishing checklist is deliberately not built, and that restyle has **no capability probe** because `LevelInPlace` has no client-supplied bar, so the read-back stands alone.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "docs: the direct-edit door, and the promises it amends"`

### Task 10: the ten-feature preservation run

- [x] `TestLiveRestylePreservesTenFeatures` in `go/internal/live`, behind the two live variables and needing `GDOC_LIVE_IDEAL_DOC_ID`. It is `go/internal/live/restyle_test.go`, and the id has no default, for the reason `GDOC_LIVE_DOC_ID` has none: the copy grant names exactly one source, and it is a document somebody already owns.
- [x] Copy with `copyComments=true` under `AllowCopy`. **Verify the copy holds all ten before restyling**, failing the setup rather than the restyle if it does not. `tenFeatures.missing()` names the ones that are not there, in SPEC item 5's own words, and a copy short of any of them is a `t.Fatalf` before a single request is built.
- [x] **Restyle the copy at `LevelInPlace`**: a fresh policy, the copy id handed in at `LevelSuggest`, then `GrantInPlace`. Otherwise the acceptance runs at the `LevelFull` the copy was learned at, exercises no allowlist, and passes with the whole milestone broken. ➕ **The refusal before the grant is asserted, not assumed.** The first request is sent through `restyle.Apply` before `GrantInPlace` and has to come back a guard refusal: a policy that had quietly kept the copy at `LevelFull` would otherwise run the whole acceptance through the door this milestone is about closing, with every assertion below it still passing.
- [x] **The trash runs on the first policy, not the second.** `judgeDrive` carries `PATCH` only at `LevelFull`, which the restyle policy deliberately does not have. The obvious fix for that refusal is to add `PATCH` to `LevelInPlace` or restyle at `LevelFull`, and both undo the point of this task.
- [x] Assert all ten. Re-read the **original** and assert its `revisionId` is unchanged on every path including failure. The re-read is a `t.Cleanup` registered before anything is created, so a failed setup reports it too. ➕ **Three of the ten are asked in a shape worth writing down.** A chip is compared as the JSON Docs sends for it with `textStyle` and `suggestedTextStyleChanges` taken off, because the restyle changes the look on purpose and what has to survive is the date chip's timestamp, locale and format; the inline image is compared as the bytes of every `word/media/` part of the docx export, because a Docs read's `contentUri` is regenerated on every read and says nothing; and the comment is compared both by its docx witness and by the words its anchor encloses, sliced per run in UTF-16, because a witness alone cannot see an anchor that moved and a paragraph-wide read would cry wolf over runs the restyle merged.
- [x] Record the result under Post-Completion.
- [x] `git commit -m "test(v2): the ten-feature preservation run"`

### Task 11: documentation and the size delta

- [ ] CLAUDE.md and the README: restyling a document gdoc did not create, the durability sentence, what a restyle overwrites, and what a failed run leaves behind.
- [ ] `docs/v2/PLAN.md`: the M7b landed paragraph, and **rewrite the M7b section itself**, which still lists the finishing checklist and the nothing-to-protect offer as this milestone's content.
- [ ] Delete `tools/copyprobe`, whose question is answered. `tools/tlsdiag` stays.
- [ ] Move this plan to `docs/plans/completed/`.
- [ ] `git commit -m "docs: M7b landed"`

## Post-Completion

- Nail restyles a real document he cares about and reads the result.
- The ten-feature run read by a person, with any feature that did not survive recorded as a decision. **Written and not yet run**, 2026-09-09: it needs the real token, the network and `GDOC_LIVE_IDEAL_DOC_ID` naming the ideal document, so it is Nail's to run with `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`. It logs the ten before and after, the manual steps, and every read-back warning, which is what a person reads.
- The binary-size delta per platform.
- Still outstanding, and Nail's: M4's live session with a second account, M5's docx opened in Word, M6's live publish and live drift table, and M7's live check that a real document agrees with `elements.json`.

## What M7b leaves for M8 and later

- **The restyle skill.** `skills/` holds v1's restyle instructions and there is no v2 one. The sentence about durability, and the report's list of what gdoc could not do, need a skill to say them. M9 with the install story, unless Nail wants it sooner.
- **List styling**, dropped because `createParagraphBullets` removes leading tabs.
- **Table column widths and row heights**, which need two request kinds nobody has measured and which change layout rather than look.
- **The finishing checklist**, SPEC's design, reported instead of written. If a terminal report proves too easy to lose, writing it into the document is its own plan and needs `insertText` back.
- **Heading numbering**, dropped because it writes into the author's prose.
- **`--new`**, still deferred, and the nothing-to-protect offer with it: `NothingToProtect` is a fact M7 emits, and with `--new` deferred there is nothing to offer.
- **The suggestions walker**, still Nail's.
