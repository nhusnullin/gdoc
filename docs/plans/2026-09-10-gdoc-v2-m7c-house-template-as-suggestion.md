# gdoc v2 Milestone 7c: the house template, proposed rather than written

## Overview

M7b gave a document the house *look*. This gives it the house *template*: the
cover, the three front-matter tables and the legend. It does it without gdoc
gaining the power to write a character on its own authority, and that is the
whole design rather than a caveat on it.

`gdoc restyle <url> --from survey.json --fields fields.json` runs two phases
against one document, with two different permissions:

| Phase | What it sends | What the document is to the guard |
|---|---|---|
| 1. propose the prelude | the cover, the front-matter tables, the legend, all in `writeMode: SUGGEST` | handed in, no grant, exactly what `propose` gets |
| 2. style the body | M7b's four styling kinds, direct | handed in **and** granted `LevelInPlace` |

Nail accepts the prelude in the browser the way he accepts any suggestion. If he
does not like it he rejects it and the document is as it was.

**Nail's idea, 2026-09-10, and it replaced a worse design.** The plan on the
table before it widened `LevelInPlace` from four request kinds to eight, added
two new per-run grants to bound where an insert could land, and rewrote M7b's
"none of the four kinds can change a character" in five places. All of that is
gone. The prelude phase needs **no new guard permission at all**: a `batchUpdate`
in SUGGEST mode on a handed-in document is already carried, because that is what
`propose` does every day.

**The measurement it rests on** is `TestLiveSuggestedInsertProbe`, run
2026-09-10, one request kind per case, each on a document nobody had suggested
anything in. Nine of ten are accepted and recorded as suggestions: the words, a
page break, a whole table (12 marks for one 2x2), an inline image, both style
kinds, the bullets, a deletion, and cell shading. The tenth is refused, and Docs
says why in its own words:

```
createNamedRange: Request does not support application as suggestion.
```

Read `docs/v2/DECISIONS.md` under 2026-09-10, including the three wrong answers
that probe gave before it gave the right one.

## The one thing nobody has measured, and Task 1 measures it

The marker is the idempotence answer: a named range over gdoc's own prelude, so
a second run replaces it rather than adding a second cover. It cannot be a
suggestion, so it is created directly, which is safe on its own terms because
`createNamedRange` adds and removes no text.

What is unknown is what a named range does when it covers **suggested** text:

- The prelude is proposed, not written, so at the moment the marker is created
  the range it covers is a pending insertion.
- If Nail **accepts**, does the range still cover the cover?
- If Nail **rejects**, the text vanishes. Does the range collapse, vanish, or
  survive covering something else?
- Can a named range even be created over a range that exists only as a pending
  insertion, or does Docs refuse it the way it refuses the suggested form?

**Nothing is built on this until it is measured.** M7b waited on the fidelity
probe for exactly this reason, and the same rule applies: a design resting on an
unmeasured API behaviour is a design resting on somebody's memory. Task 1 is a
probe and Task 2 does not start until Nail has read its table.

If the answer is bad, the fallback is already known and costs one feature rather
than the milestone: no marker, and a second run refuses when the document
carries a pending gdoc proposal or an existing house cover, telling Nail to
accept or reject the first one. `restyle --dry-run` already reports pending
suggestions, so the refusal needs no new read.

## Decisions Nail took, 2026-09-10

All of them are in `docs/v2/DECISIONS.md` under that date. Repeated here because
a plan a reviewer reads should not need a second file open.

- **The comments, the pending suggestions, the chips and the URL must survive**,
  so the template goes into the live document. SPEC's `--new` is the other mode
  and stays deferred.
- **Heading numbering is out.** It is the one item that writes inside the
  author's own paragraphs, which a prelude proposed before the body does not
  cover. Its own milestone, with its own decision about writing into prose.
- **The contents list stays manual**, and that is not a choice. `BLOCKED-BY-API.md`
  records that `insertTableOfContents` and four other spellings answer
  `Cannot find field`.
- **The cover's values come from a file the skill proposes and Nail confirms.**
  A restyle has no note to read front matter from.
- **The marker is the one thing written directly**, subject to Task 1.
- **The guard is not widened for the prelude.** Two phases with two permissions
  rather than one permission that does both. Written up under "Two phases" below.

Spec: `docs/v2/SPEC.md` (`restyle`). Master plan: `docs/v2/PLAN.md`, which needs
an M7c section this plan's Task 9 writes. Predecessor:
`docs/plans/completed/2026-09-09-gdoc-v2-m7b-restyle-in-place.md`.

## Context (from discovery)

- **The house template is already built, for docx.** `internal/render`'s
  `cover()`, `frontMatter()`, `legend()` and `contents()` produce it, and the
  drift gate measures it against the master. What does not exist is the same
  content as Docs API requests. This milestone writes that, and it writes it
  from the same `house.Config`, never from a second copy of the layout.
- **`cover.Fields` already names the thirteen values**: `Title`, `AltTitle`,
  `DocType`, `Version`, `Date`, `Owner`, `LastApproval`, `ReviewFrequency`,
  `BoardRatification`, `Distribution`, `Classification`, `HeadingNumbering`,
  `Revisions`. The fields file is that struct, read from JSON instead of from a
  note's YAML. Two readers of one shape, and the shape is `cover`'s.
- **`cover` already refuses to invent a title** and hands back a candidate drawn
  from the first heading or the file name, for the skill to propose. That is the
  pattern the whole fields file follows, one step further out.
- **The guard needs one addition and no widening.** `LevelSuggest` already
  carries a SUGGEST `batchUpdate`, and `judgeRequests` carries the kinds the
  prelude needs. The only new door is `createNamedRange` as a direct write on
  one granted id.
- **`LevelInPlace`'s allowlist gates every `batchUpdate` on a granted id,
  whatever `writeMode` says**, and CLAUDE.md gives the reason. That rule is why
  the two phases need two policies rather than one, and it is **not** changed
  here.
- **`restyle --from` already reads a survey strictly, rechecks the revision and
  refuses a second tab**, all before the grant is opened. The prelude phase
  inherits those checks rather than repeating them.
- **`internal/restyle`'s `Apply` already batches, carries
  `requiredRevisionId`, and reports `MaybeApplied`.** The prelude phase sends
  through it, with SUGGEST added to the body.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own
  commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and
  failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test`
  runs `-race`; keep it there.
- **CRITICAL: Task 1 is a measurement and it gates the rest.** It asserts almost
  nothing, it logs a table, and it fails only when it cannot create or cannot
  trash. Nail reads the table. Do not build Task 6 against a guess about what it
  will say.
- **CRITICAL: the guard is not widened for the prelude.** If a task finds itself
  wanting `insertText` at `LevelInPlace`, that is the two-phase split having
  been collapsed by mistake, not a permission that needs adding. Stop and say so.
- **CRITICAL: one layout, two writers.** The cover, the tables and the legend
  are `house.Config`'s, and the Docs request builder reads the same fields
  `internal/render` reads. A second copy of the layout is two documents that
  drift, which is the failure v1 had with its template.
- **CRITICAL: facts only in Go.** The report says what was proposed and what
  came back. Whether the cover is right is Nail's, in the document.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces, code comments, commit messages
  and the docs included.

## Testing Strategy

- **Live, opt-in, Task 1**: the named-range-over-suggestion probe. Its own
  document per case, binned after.
- **Unit, `internal/guard`**: the one new grant, its refusals, and a test that
  the prelude's SUGGEST batch needs no grant at all. The existing
  `TestNothingAtLevelInPlaceCanChangeACharacter` is untouched and must stay
  green: this milestone adds no request kind to that level.
- **Unit, the new builder package**: the prelude as a pure function of
  `house.Config` and `cover.Fields`, against literals. Every house value written
  out as a number, never read from `cfg`.
- **Unit, `cmd/gdoc`**: strict argument parsing, the fields file read strictly,
  the two phases in order, and the envelope on each failure path.
- **Live, opt-in behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`**: propose a
  prelude onto a copy of a real document, assert every piece came back as a
  suggestion and not a direct edit, assert the author's body text is unchanged
  byte for byte, and leave the copy for a person to look at.
- **Boundary**: `allowedModules` does not change. M7c adds no dependency.
- Coverage standard: every exported function under `go/internal/` has a test;
  the new package at or above 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build` and `make dist`, with the per-platform binary sizes recorded in
  Task 10

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

`gdoc restyle <url> --from survey.json --fields fields.json` does this:

1. Read the survey strictly and recheck the document, exactly as M7b does. One
   tab, revision unmoved, same document.
2. Read the fields file strictly. A missing required value is refused by name
   before anything leaves the machine.
3. Decide what the prelude phase should do: propose, replace an accepted one, or
   refuse because one is already pending.
4. **Phase 1**, on a policy with no grant: the prelude as SUGGEST batches.
5. **Phase 2**, on a second policy with `GrantInPlace`: M7b's styling, unchanged.
6. Read back, and report both phases as one result.

Key design decisions and why:

- **Two policies, because the guard's rule is right and stays.** A granted
  document takes four request kinds and nothing else, `writeMode` included. The
  prelude is therefore sent on a policy that granted nothing, which is the same
  policy `propose` uses. Neither phase can do the other's job, and that is the
  point rather than an inconvenience.
- **Phase 1 goes first.** A prelude proposed before the body is styled means the
  styling phase reads a document whose indexes already include the pending
  insertion, and M7b's styling requests name ranges. Reading fresh between the
  phases is what makes that safe, and it is one read.
- **The fields file is `cover.Fields` in JSON.** Not a new shape, not eleven
  flags. Read with `DisallowUnknownFields` and a refusal for a second object
  behind the first, the way `readProposals` and `readSurvey` already read theirs.
- **Nothing infers a title.** `cover.MissingTitle` already carries a candidate,
  and the skill proposes it to Nail. gdoc writes no title into anybody's
  document without a person having confirmed it.

## Technical Details

**Principles.** Serves 3, and unlike M7b it does not amend it. PRINCIPLES.md,
CLAUDE.md's Never list and SPEC's Never list all keep the sentences M7b left
them with, because the prelude is a proposal and the one direct write adds no
character. Task 9 checks that claim rather than assuming it.

**Global constraints:**

- Exactly one JSON object on stdout, exit 0 if and only if `ok`. No prompting,
  no stdin.
- `--from` stays required. `--fields` is required for the prelude, and a run
  with `--from` alone is M7b's styling-only restyle, which must keep working
  unchanged.
- A document with more than one tab stops the run before either phase.
- Strict argument parsing, as everywhere else.

## Implementation Steps

---

### Task 1: measure what a named range does over suggested text

- [ ] `TestLiveNamedRangeOverSuggestionProbe` in `go/internal/live`, behind the
      two live variables, one fresh document per case, each binned after.
- [ ] Four questions, each its own case, each logged rather than asserted:
      can a named range be created directly over a range that is a pending
      insertion; does it survive the suggestion being **accepted**; does it
      survive the suggestion being **rejected**; and what does it cover
      afterwards in each case.
- [ ] Accepting and rejecting are themselves `batchUpdate` requests
      (`acceptSuggestion`, `rejectSuggestion`) on a document the probe created,
      so they run at `LevelFull` and need no grant. **The guard's suggestion
      rules are not touched**: this is the probe's own document.
- [ ] It asserts almost nothing and fails only when it cannot create or cannot
      trash, for `TestLiveStyleFidelity`'s reason.
- [ ] **Stop here and report the table to Nail.** Task 6's design depends on the
      answer, and the fallback in the overview is what happens if it is bad.
- [ ] `git commit -m "test(v2): measure a named range over suggested text"`

### Task 2: the one new guard door, and proof the prelude needs none

- [ ] Test first, as an attack: a direct `createNamedRange` on a handed-in
      document is refused; the same on a granted document with no marker grant
      is refused; with the grant it carries; and a second one naming a different
      range is refused.
- [ ] `Policy.AllowMarker(...)` in `AllowReject`'s shape: per-run, one object,
      dying with the process. Nothing persists it and no flag turns it on.
- [ ] **The other direction, and it is the more important test.** A SUGGEST
      `batchUpdate` carrying `insertText`, `insertTable`, `insertPageBreak`,
      `createParagraphBullets` and `deleteContentRange` on a handed-in document
      with **no grant of any kind** must carry today, unchanged. That is the
      whole prelude phase, and the test states that this milestone added no
      permission for it.
- [ ] `TestNothingAtLevelInPlaceCanChangeACharacter` stays green and untouched.
      A task that needs to edit it has collapsed the two phases.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): one grant for the marker, and none for the prelude"`

### Task 3: the fields file

- [ ] Test first: read strictly, refuse an unknown key by name, refuse a second
      object behind the first, refuse a missing title carrying `cover`'s own
      candidate, and accept a file naming every one of the thirteen fields.
- [ ] It is `cover.Fields` decoded from JSON. **No second shape**, and no
      translation layer: a field this file names is a field `internal/render`
      already reads.
- [ ] `Revisions` is a list, which is why this is a file rather than flags.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the cover fields file, read strictly"`

### Task 4: the cover as Docs requests

- [ ] Test first, literals throughout: the cover's lines, their order, their
      alignment, their sizes and colours, and the page break under it. Every
      expected value written as a number, never read from `cfg`.
- [ ] Built from `house.Config`'s own cover spec, the same fields
      `render.builder.cover()` reads. **State in the package doc that these are
      two writers of one layout**, and that a value added to `house.yaml` has to
      reach both.
- [ ] The placeholder rules `render` already holds carry over and are tested
      here too: a line naming `with:` is left out when the field is empty, the
      `Version: ` label is a run and only the number is the note's, and a
      classification cell is shaded only when the fields declare that class.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the house cover, as Docs requests"`

### Task 5: the front-matter tables and the legend

- [ ] Test first: the three tables cell by cell, the repeat and without rules
      for `revisions`, and the legend's bullets. Literals throughout.
- [ ] `insertTable` then fill: a table is inserted and its cells are written,
      which is more requests than the docx writer needs and is the shape Docs
      has. Say so in the package doc.
- [ ] **The bullets are `createParagraphBullets`**, which M7b refuses at
      `LevelInPlace` because it removes leading tabs. Here it is a suggestion on
      text gdoc itself just proposed, so there are no author tabs to remove.
      That difference is the reason it is allowed here and refused there, and it
      goes in the package doc rather than being left for somebody to rediscover.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the front-matter tables and the legend"`

### Task 6: the marker, and what a second run does

- [ ] **Built to Task 1's measured answer**, not to this plan's guess. If the
      answer was bad, this task is the fallback instead: refuse a second run and
      name what to do.
- [ ] Test first: a first run marks; a second run on a document carrying the
      marker replaces rather than adds; a second run on a document carrying a
      **pending** gdoc proposal refuses and says to accept or reject it first.
- [ ] The marker's name is a constant, and it is read by id rather than by name:
      `docs.NamedRange` is keyed by id because two ranges may share a name, and
      deleting by name deletes both.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the prelude marker, and the second run"`

### Task 7: the command, and the two phases

- [ ] Test first in `go/cmd/gdoc/restyle_test.go`: strict argument parsing;
      `--fields` refused without `--from`; a run with `--from` alone still doing
      M7b's styling and nothing else; the survey and fields checks **before**
      either phase; and the envelope on each failure path.
- [ ] **Two policies, two sessions, in one command.** Phase 1's policy grants
      nothing. Phase 2's grants `GrantInPlace`. A test asserts that the phase 1
      session has no in-place grant, because collapsing them is the one mistake
      that would undo this milestone's whole shape.
- [ ] Phase 1 first, then a fresh read, then phase 2. The styling requests name
      ranges and the prelude moved them.
- [ ] **A failed phase 1 does not run phase 2**, and the report says which phase
      stopped. A failed phase 2 after a successful phase 1 leaves a proposed
      prelude and an unstyled body, which is a document Nail can still act on.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): gdoc restyle --fields, in two phases"`

### Task 8: the read-back, and one report for two phases

- [ ] Test first: the prelude read back as suggestions, counted by kind, and
      **a check that the author's own body text is unchanged**. That check is
      the milestone's own claim, so it is a test rather than a hope.
- [ ] The report carries both phases: what was proposed, what was styled, what
      is pending for Nail to accept, and M7b's `manual` list unchanged.
- [ ] **`proposed` is a count of suggestions, never a verdict.** A field named
      `looks_right`, `complete` or `ready` here is the defect CLAUDE.md names.
- [ ] `verified` is the checks together, and fewer than all is `ok: true` with
      `verified: false` and the route named.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the read-back over both phases"`

### Task 9: the documentation, and the promise that does not change

- [ ] `docs/v2/PLAN.md` gains an M7c section; SPEC's `restyle` section gains the
      prelude and says the contents list is still manual and why.
- [ ] CLAUDE.md gains a section for the two phases, the one new grant, and the
      `createParagraphBullets` difference between the two levels.
- [ ] **Check, rather than assume, that principle 3's wording still holds.**
      M7b amended four places. This milestone should amend none: the prelude is
      a proposal and the marker adds no character. If any of the four is now
      inaccurate, say so and fix it rather than leaving it.
- [ ] Record Task 1's measurement in DECISIONS.md whatever it said.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "docs: the house template, proposed rather than written"`

### Task 10: the live acceptance, and the size delta

- [ ] `TestLivePreludeIsProposedNotWritten` in `go/internal/live`, behind the two
      live variables: copy a document with `copyComments=true`, propose the
      prelude onto the copy, assert every piece came back carrying a suggestion
      id, assert the author's body text is byte-identical, and leave the copy
      for a person to look at.
- [ ] Re-read the **original** on every path and assert its `revisionId` never
      moved, as the M7b acceptance does.
- [ ] Record the result and the binary-size delta per platform under
      Post-Completion.
- [ ] Move this plan to `docs/plans/completed/`.
- [ ] `git commit -m "test(v2): the prelude acceptance, and M7c landed"`

## Post-Completion

- Nail runs it on a real document, looks at the proposed cover, and accepts or
  rejects it.
- Task 1's table read by a person, and its answer recorded as a decision.
- The binary-size delta per platform.
- Still outstanding from earlier milestones, and Nail's: M4's live session with a
  second account, M5's docx opened in Word, M6's live publish and live drift
  table, M7's live check that a real document agrees with `elements.json`, and
  M7b's ten-feature acceptance, which has never run and needs a one-tab document
  holding all ten.

## What M7c leaves for later

- **Heading numbering.** The one item that writes inside the author's own
  paragraphs. Its own milestone and its own decision.
- **The contents list.** Not deferred, blocked: the Docs API has no request that
  makes one. `BLOCKED-BY-API.md` records the measurement.
- **The first-page header with the logo, and the footer page numbers.** Blocked
  the same way, and reported with their menu paths.
- **The footer's own text and colour.** Reachable today with no new permission,
  and unbuilt: `internal/docs` decodes no headers and no footers. It is
  `docs/backlog/restyle-skips-footnotes-headers-and-footers.md`.
- **A restyle skill.** `skills/` holds v1's restyle instructions and there is no
  v2 one. The prelude makes that gap wider, because somebody has to write the
  fields file and read the report. M9, unless Nail wants it sooner.
