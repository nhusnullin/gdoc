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
- **Heading numbering is out of M7c, and it is out for two reasons that are not
  the obvious one.** It can be a suggestion, and when it lands it lands as one:
  `mechanism: literal` in `house.yaml` means the number is `insertText` in front
  of the heading's own words, and `insertText` is row one of the probe. Nail's
  recommendation, 2026-09-10: a number per heading shown as a pending insertion
  is the clearest a change of that kind can be, so its own milestone starts from
  suggested mode rather than reopening the question. What keeps it out of this
  one is idempotence and placement. There is no marker for a heading number, and
  `numberedHeadingRE` in `internal/body/numbering.go` does not recognise the
  house's own format, because the separator is `-` and the pattern wants `.`,
  `)` or a space: "1-Introduction" is not read as an authored number, so a
  second run makes it "1-1-Introduction". A named range per heading is the other
  answer, and it multiplies Task 1's unknown by the heading count. Placement is
  the second: the prelude is one insertion at index 1, while a number names a
  position inside the author's prose, which is what `propose`'s "names text,
  never an index" rule exists to refuse.
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

- [x] `TestLiveNamedRangeOverSuggestionProbe` in `go/internal/live`, behind the
      two live variables, one fresh document per case, each binned after.
- [x] Four questions, each its own case, each logged rather than asserted:
      can a named range be created directly over a range that is a pending
      insertion; does it survive the suggestion being **accepted**; does it
      survive the suggestion being **rejected**; and what does it cover
      afterwards in each case.
- [x] Accepting and rejecting are themselves `batchUpdate` requests
      (`acceptSuggestion`, `rejectSuggestion`) on a document the probe created,
      so they run at `LevelFull` and need no grant. **The guard's suggestion
      rules are not touched**: this is the probe's own document.
      ⚠️ **Half of that was wrong, and the guard is right.** `judgeRequests`
      refuses every request kind whose name carries "suggestion" **at every
      level**, because SPEC's Never list names no level: a document the probe
      created a second ago is refused like anybody else's. `rejectSuggestion`
      has one door, `AllowReject`, which withdraw already uses, so the probe
      seeds it with its own suggestion id and the reject case runs.
      `acceptSuggestion` has no door, and opening one to measure a probe would
      be widening the guard for the tail rather than the dog. So the accept case
      leaves its document in the folder, prints the URL, and Nail accepts it in
      the browser the way he will accept a real prelude.
      `TestLiveNamedRangeAfterAcceptedByHand`, read-only and behind
      `GDOC_LIVE_ACCEPTED_DOC_ID`, then finishes the table. The guard was not
      touched.
- [x] It asserts almost nothing and fails only when it cannot create or cannot
      trash, for `TestLiveStyleFidelity`'s reason.
- [x] **Stop here and report the table to Nail.** Task 6's design depends on the
      answer, and the fallback in the overview is what happens if it is bad.
- [x] `git commit -m "test(v2): measure a named range over suggested text"`

#### Measured, 2026-09-10, run against the Drive test folder

| Question | Answer |
|---|---|
| created directly over a pending insertion? | **yes**, id `kix.wi79lhqfq91l` |
| what it covers while the insertion is pending | `gdoc:house-prelude [1,23)`, exactly the proposed line |
| after the suggestion is **rejected** | **the marker is GONE**: no named range at all |
| after the suggestion is **accepted** | **it survives**: same id, same `[1,23)`, now over the accepted text |
| can gdoc accept its own suggestion? | **no**, and by design: the guard refuses `acceptSuggestion` at every level |

**All four are good answers, and the accept row was read on 2026-09-10** after
Nail accepted the probe's suggestion in the browser, through the read-only
`TestLiveNamedRangeAfterAcceptedByHand`. It printed
`gdoc:house-prelude [1,23) "SUGGESTED COVER TITLE "`. So:

- The marker can be created over text that exists only as a proposal, so the
  first run can mark what it proposed.
- It covers exactly the proposed line while that line is pending, so a second
  run reading the marker reads gdoc's own prelude and nothing of the author's.
- **It survives the accept**, keeping its id and its range and now covering the
  text the accept made real. So a second run finds gdoc's own prelude and knows
  where it ends.
- A rejected prelude takes its marker with it. The document goes back to having
  no prelude and no mark of one, so the run after a rejection proposes cleanly
  with nothing to detect and nothing to clean up.

**Task 6 is therefore the replace-in-place design, not the fallback**, and
Task 2's `AllowMarker` has a production caller. The two shapes a second run
meets are a marker that is there, which it replaces, and no marker at all, which
is either a document gdoc has never touched or one whose prelude was rejected,
and both of those are proposed into cleanly. The third shape is a marker over a
prelude still **pending**, which is neither: the plan's refusal stands there, and
Task 6 tells Nail to accept or reject the one already in front of him.

The probe document `1D0ErMFgR3Gz3W_1pZ4IRWZDzfTGmVBFBP_wnNvfkSW0` is left in the
test folder with the accepted line in it, for Nail to bin. gdoc does not trash
it: nothing here holds a `--trash` command, and the answer is written down.

### Task 2: the one new guard door, and proof the prelude needs none

- [x] Test first, as an attack: a direct `createNamedRange` on a handed-in
      document is refused; the same on a granted document with no marker grant
      is refused; with the grant it carries; and a second one naming a different
      range is refused.
- [x] `Policy.AllowMarker(...)` in `AllowReject`'s shape: per-run, one object,
      dying with the process. Nothing persists it and no flag turns it on.
- [x] **The other direction, and it is the more important test.** A SUGGEST
      `batchUpdate` carrying `insertText`, `insertTable`, `insertPageBreak`,
      `createParagraphBullets` and `deleteContentRange` on a handed-in document
      with **no grant of any kind** must carry today, unchanged. That is the
      whole prelude phase, and the test states that this milestone added no
      permission for it.
- [x] `TestNothingAtLevelInPlaceCanChangeACharacter` stays green and untouched.
      A task that needs to edit it has collapsed the two phases.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): one grant for the marker, and none for the prelude"`

### Task 3: the fields file

- [x] Test first: read strictly, refuse an unknown key by name, refuse a second
      object behind the first, refuse a missing title carrying `cover`'s own
      candidate, and accept a file naming every one of the thirteen fields.
      ⚠️ **A fields file carries no candidate, and that is the honest answer.**
      `MissingTitle` draws its candidate from a note's first heading or its file
      name, and this file has neither. So the refusal is the same error type,
      which is what the skill reads, with an empty `Candidate`.
      `MissingTitle` gained a `Where` field for it: empty is "front matter",
      which is what `cmd/gdoc`'s `titleError` still builds, and the fields
      reader sets "the fields file". One sentence, two files, and neither names
      the other's.
- [x] It is `cover.Fields` decoded from JSON. **No second shape**, and no
      translation layer: a field this file names is a field `internal/render`
      already reads. `TestTheFieldsFileNamesTheSameThirteenValuesTheNoteDoes`
      states that as a test: one note and one fields file stating the same
      values read to the same `Fields`, revisions included.
      The keys are the note's own keys, and the two normalisations are shared
      rather than copied: `classificationLabel` and `numbering` are now one
      rule each, asked by both readers. The defaults are shared too, so a file
      stating no version publishes as 1.0 and one stating no date as this month,
      exactly as a note does.
- [x] `Revisions` is a list, which is why this is a file rather than flags.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the cover fields file, read strictly"`

### Task 4: the cover as Docs requests

- [x] Test first, literals throughout: the cover's lines, their order, their
      alignment, their sizes and colours, and the page break under it. Every
      expected value written as a number, never read from `cfg`.
- [x] Built from `house.Config`'s own cover spec, the same fields
      `render.builder.cover()` reads. **State in the package doc that these are
      two writers of one layout**, and that a value added to `house.yaml` has to
      reach both.
      ➕ Three rules the two writers share now live once rather than twice, each
      where its value lives: `cover.Fields.Placeholder` is what a placeholder
      name means, `house.Cell.FillFor` is the classification shading, and the
      new `internal/docsreq` holds a measurement, a colour, an alignment, a
      length in the units the API counts, and a style object built with the mask
      that names it. `internal/restyle` was moved onto `docsreq` in the same
      commit, so the line-spacing rounding and the colour conversion have one
      copy between the two request builders.
- [x] The placeholder rules `render` already holds carry over and are tested
      here too: a line naming `with:` is left out when the field is empty, the
      `Version: ` label is a run and only the number is the note's, and a
      classification cell is shaded only when the fields declare that class.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the house cover, as Docs requests"`

**Two decisions this task took, and neither is in the plan above.**

gdoc's own paragraphs state their look in full: the named style, the alignment,
the spacing, the indents, and on every run the face, the size, the weight, the
slope, the underline and both colours. Text inserted into a document takes the
look of the text it lands beside, so a cover line proposed in front of somebody's
indented, justified Heading 1 would arrive wearing all of it, in the contents
list with it. That is the opposite of `internal/restyle`'s rule, which never
writes a flag `house.yaml` did not state, and the difference is whose words are
being styled: a restyle writes onto the author's text, and these are gdoc's own
lines that nobody else's emphasis can be in.

The one look an inserted paragraph inherits and this package cannot state away
is a list marker: taking one off needs `deleteParagraphBullets`, which nothing
here sends. `docs/backlog/prelude-inherits-a-list-marker.md` holds it, with the
two ways out.

### Task 5: the front-matter tables and the legend

- [x] Test first: the three tables cell by cell, the repeat and without rules
      for `revisions`, and the legend's bullets. Literals throughout.
- [x] `insertTable` then fill: a table is inserted and its cells are written,
      which is more requests than the docx writer needs and is the shape Docs
      has. Say so in the package doc.
- [x] **The bullets are `createParagraphBullets`**, which M7b refuses at
      `LevelInPlace` because it removes leading tabs. Here it is a suggestion on
      text gdoc itself just proposed, so there are no author tabs to remove.
      That difference is the reason it is allowed here and refused there, and it
      goes in the package doc rather than being left for somebody to rediscover.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the front-matter tables and the legend"`

⚠️ **The house legend carries no bullets, so nothing sends
`createParagraphBullets`.** The plan expected some. `house.yaml`'s `legend`
block is four lines of a bold word and a sentence, and the master's own markup
under "New information has been added" has no `w:numPr` on it, so bullets there
would be a layout this writer invented, which is exactly what "one layout, two
writers" forbids. The reasoning the checkbox asked for is written into the
package doc anyway, beside the statement that nothing sends the request, so the
next person to need a bullet here has the answer rather than the question.

**What this task settled, and it is the number the whole table rests on.**
insertTable's index accounting is computed here, never read back, so it had to
be pinned: an empty table is one unit for the newline insertTable writes in
front of it, one for the table, one per row, and one per cell plus one for that
cell's own paragraph mark. A 2x2 is twelve, which is exactly the twelve marks
Docs recorded for one 2x2 in the 2026-09-10 suggested-insert probe.
`TestATablesIndexesFollowTheDocsAccounting` states it as literals, and Task 10's
live run is what confirms it against a real document.

**Two things the house file states and no request here sends:** every column
width and every row height. `updateTableColumnProperties` and
`updateTableRowStyle` were not among the nine kinds the probe measured as
suggestible, and a request Docs refuses takes the whole batch with it. They are
reported in `Manual` with the menu path, beside the contents list the API cannot
make at all.

**One request per cell rather than one per table**, which is the opposite of
`internal/restyle`'s rule and for the opposite reason. A restyle gives every
cell of somebody's table one look and cannot read that table's real width, so it
names the table. Here the fills differ cell by cell, because a classification
row is shaded only when the fields declare that class, and this writer built the
grid itself so it knows exactly how wide it is.

### Task 6: the marker, and what a second run does

- [x] **Built to Task 1's measured answer**, not to this plan's guess. If the
      answer was bad, this task is the fallback instead: refuse a second run and
      name what to do.
- [x] Test first: a first run marks; a second run on a document carrying the
      marker replaces rather than adds; a second run on a document carrying a
      **pending** gdoc proposal refuses and says to accept or reject it first.
- [x] The marker's name is a constant, and it is read by id rather than by name:
      `docs.NamedRange` is keyed by id because two ranges may share a name, and
      deleting by name deletes both.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the prelude marker, and the second run"`

**The replace design, because all four measured rows were good.** `Decide` in
`internal/prelude/marker.go` reads the document and answers one of three shapes.
No marker is a first run, which proposes at index 1. One marker over settled
text is a second run, which proposes a `deleteContentRange` over the marked span
and then the fresh prelude at that span's own start, which is the order and the
shape `internal/propose` sends a replacement in: a delete in SUGGEST mode marks
text rather than removing it, so nothing behind it moves and every index the
front matter computed is still the index it named. One marker over text that is
still pending is neither, and the refusal names the suggestion ids and says to
accept or reject in the browser first.

**Nothing deletes a named range, and nothing needs to.** `deleteNamedRange` is
not on any allowlist and no request here sends one. The marker tracks its text:
measured, a rejected insertion took its marker with it and an accepted one kept
its id and its range. So the old marker goes when the deletion it now sits under
is accepted, and comes back when that deletion is rejected. Either way one
marker is left, which is the shape `Decide` requires. **That last step is the
inference the measurement makes rather than a fifth measured row**, and Task 10's
live run is what confirms it. It is written down in `Decide`'s own doc comment as
an inference and not as a measurement.

**Pending is read over the marked span, in both lists and inside the tables.** A
run overlapping the span carrying an insertion id says the prelude has not been
accepted; a deletion id says a run before this one already proposed replacing
it. Neither is text this run may propose deleting. The walk goes into table
cells, because the front matter is three tables and a walk reading paragraphs
alone would call a wholly proposed prelude settled.

**Two refusals the plan did not ask for, and both are the same rule.** What a
caller does with a marker is delete the text under it, so a marker this package
cannot read as one contiguous span of one tab's body is refused rather than
reported: a span in a header, a footer or a footnote, and two spans with the
author's own words between them. Two spans that *touch* are one span, because
Docs may cut a range at a boundary of its own and the text is still gdoc's.

➕ **The pin that the builder and the guard spell one shape.** The guard reads
`createNamedRange` exactly, and `internal/guard/marker_test.go` writes those
bytes out by hand, so two files spelled one shape.
`TestTheMarkerRequestIsWhatTheGuardGrants` in `internal/prelude` hands the real
policy the real bytes `MarkerRequest` builds, and its twin asserts a marker over
a range the run did not grant is refused.

### Task 7: the command, and the two phases

- [x] Test first in `go/cmd/gdoc/restyle_test.go`: strict argument parsing;
      `--fields` refused without `--from`; a run with `--from` alone still doing
      M7b's styling and nothing else; the survey and fields checks **before**
      either phase; and the envelope on each failure path.
- [x] **Two policies, two sessions, in one command.** Phase 1's policy grants
      nothing. Phase 2's grants `GrantInPlace`. A test asserts that the phase 1
      session has no in-place grant, because collapsing them is the one mistake
      that would undo this milestone's whole shape.
      ➕ The test judges **the bytes the run really sent**, not a shape written
      by hand beside it. `TestThePreludePhaseIsSentOnAPolicyThatGrantsNothing`
      hands each recorded batch body to the policy it went out on, and asks the
      two refusals as well as the two carries: the granted policy refuses the
      prelude batch, and the ungranted one refuses the styling batch.
- [x] Phase 1 first, then a fresh read, then phase 2. The styling requests name
      ranges and the prelude moved them.
- [x] **A failed phase 1 does not run phase 2**, and the report says which phase
      stopped. A failed phase 2 after a successful phase 1 leaves a proposed
      prelude and an unstyled body, which is a document Nail can still act on.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): gdoc restyle --fields, in two phases"`

**Three decisions this task took, and none of them is in the plan above.**

⚠️ **Phase 2 walks past the span phase 1 wrote, and without that the milestone
defeats itself.** Every paragraph `internal/prelude` proposes is a `NORMAL_TEXT`
stating the cover's own sizes and colours in full, because inserted text takes
the look of the text it lands beside. A styling phase reading the document after
phase 1 finds those paragraphs and gives each of them the house body look, so
the 26pt cover title Nail is being asked to accept is 11pt prose by the time he
reads it. `restyle.TabRequestsExcept` takes the span and reports what it left
alone as `planned.skipped`; `TabRequests` is that function with no span, so the
M7b run is unchanged. The overlap is read rather than containment, and the
reason is in `Span.covers`: a block half gdoc's words and half the author's is
one no request can name without writing over one of them.

⚠️ **The marker is phase 2's, it goes out in a batch of its own, and a marker
that does not land stops the run.** `createNamedRange` is refused at every level
but `LevelInPlace`, so the request cannot be sent on phase 1's policy at all.
Its own batch ahead of the styling, because a styling batch that does not land
still leaves a prelude Nail can accept, and a marked one is a prelude the next
run can find: folded into the styling it would be lost with it. A failed marker
is a proposed prelude gdoc has no record of writing, and the next run over one
proposes a second cover in front of the first, so the run stops and the warning
says to accept or reject before running again.

**`restyle.Suggest` is `Apply` with one field, and one sentence.** The loop is
the same because it wants the same three things: batches the guard can read
whole, a revision id on every one of them, and a batch Docs accepted whose
answer could not be read reported as itself. What changes is
`writeControl.writeMode`, and what that changes is the recovery a run that
stopped early prints: a suggested batch is rejected in the browser, not undone
through the version history. `requiredRevisionId` and `writeMode` together are
**unmeasured**, and Task 10's live run is what confirms them: Docs refusing the
pair fails the prelude phase whole, which is the direction to be wrong in.

### Task 8: the read-back, and one report for two phases

- [x] Test first: the prelude read back as suggestions, counted by kind, and
      **a check that the author's own body text is unchanged**. That check is
      the milestone's own claim, so it is a test rather than a hope.
- [x] The report carries both phases: what was proposed, what was styled, what
      is pending for Nail to accept, and M7b's `manual` list unchanged.
- [x] **`proposed` is a count of suggestions, never a verdict.** A field named
      `looks_right`, `complete` or `ready` here is the defect CLAUDE.md names.
- [x] `verified` is the checks together, and fewer than all is `ok: true` with
      `verified: false` and the route named.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the read-back over both phases"`

**Three questions, and none of them can see the other two's failure.**
`prelude.Verify` in `internal/prelude/readback.go` asks whether every piece of
the prelude carries a suggestion id, whether the marker is over the span that
was proposed, and whether the author's own text is character for character what
it was. A prelude wholly written rather than proposed would pass the marker
check; a prelude wholly proposed and unmarked would pass the first; and a run
that took a paragraph of somebody's prose with it would pass both. So the three
are separate fields and `verified` is all of them together, with the read having
found the prelude at all.

**The classification unit is the run, and the counts are paragraphs, tables and
cells.** `Pieces` counts what `Result` counts, so what was sent and what came
back sit beside each other in one report with nothing to convert. A paragraph, a
table or a cell is `proposed` when every run of it inside the span carries an
insertion id and `written` when one of them does not, and `written` is the
failure rather than a difference: those are characters in somebody's document on
gdoc's own authority. The walk goes into table cells, because the front matter
is three tables and a walk reading paragraphs alone would call a wholly proposed
table settled.

**The two sides of the body check are read by one rule, which is what makes the
answer mean something.** `AuthorText` drops every text run carrying a suggested
insertion id and keeps every run carrying a suggested deletion id: an insertion
is nobody's text yet, whoever proposed it, and a deletion in SUGGEST mode marks
characters rather than removing them, so they are still the author's until
somebody accepts it. The before side is taken in `proposeThenStyle` off the
document phase 1 was computed from, which is the last moment it can be taken:
every read after that one carries the prelude.

➕ **A run that proposed a prelude reads back whatever phase 2 did.** M7b's gate
was the styling batches alone, and on a two-phase run that answered "nothing was
written, so the document is as it was" about a document phase 1 had just written
a cover into. The gate is `applied.Batches > 0 || applied.MaybeApplied ||
one != nil`, and a read the run could not make leaves both halves absent with the
warning naming which read failed.

### Task 9: the documentation, and the promise that does not change

- [x] `docs/v2/PLAN.md` gains an M7c section; SPEC's `restyle` section gains the
      prelude and says the contents list is still manual and why.
- [x] CLAUDE.md gains a section for the two phases, the one new grant, and the
      `createParagraphBullets` difference between the two levels.
- [x] **Check, rather than assume, that principle 3's wording still holds.**
      M7b amended four places. This milestone should amend none: the prelude is
      a proposal and the marker adds no character. If any of the four is now
      inaccurate, say so and fix it rather than leaving it.
      ⚠️ **Three of the four were inaccurate by one word, and each gained a
      sentence rather than losing one.** PRINCIPLES.md's principle 3, CLAUDE.md's
      Never list and SPEC's Never list and acceptance item 1 all said the
      granted level carries four request kinds and refuses every other kind
      whatever it is called. `createNamedRange` now carries there under
      `AllowMarker`. The claim each of them was really making, that nothing the
      grant carries can change a character, is still true, so every amendment
      names the fifth kind and says it adds and removes no character. The
      prelude itself needed no amendment anywhere: it is a suggestion on a
      policy that granted nothing.
      ➕ `inPlaceKinds`' own doc comment said `createNamedRange` "is left out
      because M7b writes no checklist and needs no range. Both are refused
      here", which stopped being true in this milestone's own guard commit. It
      now says where the kind is judged instead.
- [x] Record Task 1's measurement in DECISIONS.md whatever it said.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "docs: the house template, proposed rather than written"`

**Three documents were written beyond the checkboxes, and each is this
milestone's own documentation rather than scope.** The README gained "Adding the
house template as a suggestion", because `--fields` is a flag a person types and
a fields file is a file a person writes. Its "What is planned" list gained
heading numbering and lost the sentence saying the two table-layout kinds were
unmeasured, which the 2026-09-10 probe measured. And
`docs/backlog/restyle-applies-the-whole-house-template.md`, the item Nail wrote
on 2026-09-10 asking for exactly this, is narrowed to what is left: the contents
list, the column widths, heading numbering and the footer. It said "this is the
first thing after M7b merges", which stopped being true when Task 4 landed.

CLAUDE.md's live-test paragraph was corrected while M7c's two probes were being
added to it: it said six write tests and named six, and there were eight before
this milestone, because `TestLiveRestyleKeepsAnchorsAndSuggestions` was never
written into it. The counts are gone rather than raised, and the two M7c probes
and the read-only accepted-document reader are named.

### Task 10: the live acceptance, and the size delta

- [x] `TestLivePreludeIsProposedNotWritten` in `go/internal/live`, behind the two
      live variables: copy a document with `copyComments=true`, propose the
      prelude onto the copy, assert every piece came back carrying a suggestion
      id, assert the author's body text is byte-identical, and leave the copy
      for a person to look at.
- [x] Re-read the **original** on every path and assert its `revisionId` never
      moved, as the M7b acceptance does.
- [x] Record the result and the binary-size delta per platform under
      Post-Completion.
- [x] Move this plan to `docs/plans/completed/`.
- [x] `git commit -m "test(v2): the prelude acceptance, and M7c landed"`

The test names its source document with `GDOC_LIVE_PRELUDE_DOC_ID`, which has no
default for the reason `GDOC_LIVE_IDEAL_DOC_ID` has none: the run copies the
whole of somebody's document, comments included, into gdoc's folder.

Three things in it are the milestone stated as a live fact rather than plumbing.
The prelude phase sends on a policy that granted nothing at all, which is what
says the prelude needed no permission M7c added. The `createNamedRange` is sent
once **before** `GrantInPlace` and `AllowMarker` and the refusal is asserted, so
a policy that had quietly kept the copy at `LevelFull` fails the test rather
than passing the whole acceptance through the door this milestone keeps shut.
The read-back is `prelude.Verify`, the production one, because a second reader
written for a test is a second rule that drifts from the one the skill reads.

`copyDocument` was split into `copyDocumentNamed`, so the copy carries the
milestone's own name in Drive and the parent rule stays in one room. Nothing
else in `internal/live` changed.

## Post-Completion

- Nail runs it on a real document, looks at the proposed cover, and accepts or
  rejects it. `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1 GDOC_LIVE_PRELUDE_DOC_ID=<id>
  go test -run TestLivePreludeIsProposedNotWritten ./internal/live` is the run,
  and it leaves the copy behind with its URL in the log. It has not been run
  from here: the unattended run sets neither live variable, so what the test
  asserts is written down and unmeasured until Nail runs it.
- Task 1's table read by a person, and its answer recorded as a decision.
- The binary-size delta per platform, measured 2026-09-10 against the merge base
  8633881, both sides built with `CGO_ENABLED=0`:

  | Platform | M7b | M7c | Delta |
  |---|---|---|---|
  | darwin/arm64 | 14.16 MB | 14.31 MB | +151 KB, +1.09% |
  | darwin/amd64 | 15.14 MB | 15.30 MB | +158 KB, +1.07% |
  | windows/amd64 | 14.98 MB | 15.14 MB | +156 KB, +1.06% |

  About 150 KB per platform for the whole milestone, and no new module: the
  three in `allowedModules` are what M5 left, and M7c added none. The cost is
  `internal/prelude` and `internal/docsreq`'s own code.
- Still outstanding from earlier milestones, and Nail's: M4's live session with a
  second account, M5's docx opened in Word, M6's live publish and live drift
  table, M7's live check that a real document agrees with `elements.json`, and
  M7b's ten-feature acceptance, which has never run and needs a one-tab document
  holding all ten.

## What M7c leaves for later

- **Heading numbering.** Not blocked, and not blocked by the guard either: it is
  `insertText`, so it can be proposed. Its own milestone, and it starts from
  suggested mode. What that milestone has to settle is a second run, since
  nothing marks a number gdoc wrote, and where each insert goes, since a number
  names a position in the author's prose rather than text to match.
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
