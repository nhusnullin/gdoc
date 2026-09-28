# propose adds and replaces whole paragraphs

## Overview

`propose` changes words inside one paragraph and nothing else. A review answer
that is "rewrite this section" or "add a section here" has no route into the
document, and on 2026-09-25 Nail pasted one by hand
(`docs/backlog/propose-cannot-add-paragraphs.md`).

This milestone adds a second proposal kind, a block: new paragraphs, headings
and lists, placed after a quoted paragraph or in place of a run of whole
paragraphs. Same file, same probe, one batch per proposal, the same comment
with the reason, and three read-backs of its own.

Google is not the blocker. The prelude already proposes 70 paragraphs in one
SUGGEST batch, and `TestLiveBlockProposalProbe` measured the block shape on
2026-09-27 (`docs/v2/MEASURED.md`, "A block of new paragraphs, proposed in one
SUGGEST batch").

## Decisions already taken

- **Refuse a replace over somebody's work, and name it** (Nail, 2026-09-27).
  When the paragraphs a replace would delete hold a pending suggestion or a
  comment's anchor, the binary refuses and names each suggestion id and
  comment id. That is a fact, not a judgement. The skills check the same thing
  before they ask, and tell the person what is in the way.
- **One id per block.** Measured: insert, styles, bullets, bullet removal and
  the deletion of a replace all carry one suggestion id. So
  `frontmatter.Proposal` keeps its one `id`, and `withdraw` is unchanged.
- **Insert at the start of the paragraph after the anchor, never at the end
  of the anchor.** Measured: an end insert hands the anchor's old mark to the
  last new paragraph, and restyling that mark is a second suggestion.
- **Clear inherited bullets.** New text takes the list membership of the
  paragraph it lands in front of. Every new paragraph that is not a list item
  gets `deleteParagraphBullets`, which folds into the same id.
- **Clear inherited text style.** Text inserted at a paragraph's start also
  takes that paragraph's first-run style, so a block in front of a bold or
  linked paragraph would arrive bold or linked. The first `updateTextStyle`
  covers the whole insert and clears bold, italic, underline, strikethrough
  and link through its fields mask. The block's own marks come after it.
- **After the last paragraph there is no paragraph to insert in front of.** The
  text goes in before the body's final newline as `"\n"` plus the content
  without its trailing newline, so the last new paragraph owns the final mark
  (MEASURED row 7). Two cases are refused by name: a block whose last
  paragraph is not plain body text, because restyling the final mark is the
  unmeasured second-id case; and a last paragraph that is a list item, because
  clearing the inherited bullet there is MEASURED row 4, which gave two ids.
  Both messages say what would work. Nail can widen either after a measurement.
- **More than one id is `verified: false`.** `withdraw` takes back one id, so a
  proposal it cannot fully take back is not a verified one.
- **Out of scope, refused by name:** tables (`propose-inside-tables` is not
  settled), nested lists (nesting through leading tabs is unmeasured in
  SUGGEST mode), pictures, and anything goldmark parses that the subset below
  does not name.

## Context (from discovery)

- `go/internal/propose/propose.go`: `Proposal{Quoted, Replacement, Why,
  Assignee}`, `Check()` refuses a line break on both sides, `Apply` reads,
  finds, writes one batch, verifies. `Batch` builds deleteContentRange,
  insertText, insertComment in SUGGEST. `Result.SuggestionIDs` is already a
  list.
- `go/internal/propose/span.go`: `FindSpan` finds a quote exactly once,
  refuses a quote crossing a chip. `paragraphs` indexes the body.
- `go/internal/propose/verify.go`: the three routes for the words kind:
  `inlineHolds`, the preview read, `docxHolds`.
- `go/cmd/gdoc/write.go`: the `propose` command and its proposals file.
- `go/internal/frontmatter/schema.go:111`: `Proposal{ID, CommentID, At,
  Quoted}` in the note.
- `go/internal/propose/propose.go:200` `Apply`: the read, find, post, answer
  handling and `Verified` sequence, with `notSent` and `sent`. The block kind
  needs its own copy of that sequence (Task 7).
- `go/cmd/gdoc/write.go`: `readProposals` decodes `[]propose.Proposal`
  strictly, and the marker check runs per field around line 507.
- `go/internal/docs`: `Document.CommentRanges` is where a comment's anchor is
  read from.
- `go/internal/guard/policy.go`: at `LevelSuggest` any kind carries inside a
  SUGGEST batch, except kinds naming "suggestion". `maxPeek` is 1 MiB, in
  `guard/transport.go:28`.
  `TestThePreludeNeedsNoGrantAtAll` pins the prelude's kinds;
  `deleteParagraphBullets` is not in it yet.
- `go/internal/body/`: already walks goldmark, one of the three modules.
- `go/internal/markers/`: every route into a document passes it.
- `go/internal/prelude/`: the only other code that builds paragraphs in a
  SUGGEST batch. Read it before Task 5; reuse its style requests where they
  fit.
- `skills/gdoc-review/propose.md` and Step 7, `skills/gdoc-align/SKILL.md`:
  both can only propose words today.

## Development Approach

- **testing approach**: TDD. The failing test first, then the smallest change
  that passes it.
- One task at a time, all tests green before the next. `make test` (raced) and
  `make vet` after each task.
- A `doc.go` rule names its test (CLAUDE.md). A SPEC change is a DECISIONS
  entry first, the same day.
- Plain English in every comment and message. No em dashes.

## Testing Strategy

- Unit tests in each package that changes, over recorded Docs answers in
  `testdata/`, never a live call.
- One live test, opt-in under `GDOC_LIVE_WRITE`, in documents it creates in the
  test folder (Task 10).

## Progress Tracking

- `[x]` when done, ➕ for a task found on the way, ⚠️ for a blocker.

## Solution Overview

A proposals file entry is either the words kind, as today, or a block:

```json
{"kind": "block", "after": "quoted words in the anchor paragraph",
 "content": "## 3.6 Limits\n\nBody text with **bold**.\n\n- one\n- two\n",
 "why": "...", "assignee": "..."}
{"kind": "block", "replace_from": "words in the first paragraph",
 "replace_to": "words in the last paragraph", "content": "...", "why": "..."}
```

Exactly one of `after` or the pair `replace_from` and `replace_to`. Each quote
is found exactly once, with the same rules as `FindSpan`. A replace covers
whole paragraphs, from the start of the first to the end of the last, so an
accept can never merge a neighbour.

`content` is a markdown subset: paragraphs, `#` to `######` headings, `-` and
`1.` lists one level deep, bold, italic, links. It becomes a list of new
paragraphs, each with a named style, runs with marks, and a list kind.

The batch, in this order, all in SUGGEST: one `insertText` at the start of the
paragraph after the anchor (for a replace, the start of the first replaced
paragraph; after the last paragraph, the shape in the decisions above); per new
paragraph `updateParagraphStyle` naming `namedStyleType`; one `updateTextStyle`
over the whole insert that clears the inherited marks; `updateTextStyle` for
each marked run; `createParagraphBullets` per list run
with `BULLET_DISC_CIRCLE_SQUARE` or `NUMBERED_DECIMAL_ALPHA_ROMAN`;
`deleteParagraphBullets` over every new paragraph that is not a list item;
for a replace, `deleteContentRange` over the old paragraphs at their shifted
indexes; `insertComment` over the first new paragraph. All lengths in UTF-16.

## Implementation Steps

### Task 1: The decision, and the spec

**Files:**
- Modify: `docs/v2/DECISIONS.md` (entry and register row)
- Modify: `docs/v2/SPEC.md` (the propose section)

- [x] DECISIONS entry dated the day it is written: the block kind, the
  decisions above with their measurement, and what is refused by name
- [x] SPEC: the proposals file's block entry, the refusals, the read-backs.
  "Changed <date>, DECISIONS.md"
- [x] `make test` (the docs boundary tests read these files)

### Task 2: The guard carries a block batch at the suggest level

**Files:**
- Modify: `go/internal/guard/marker_test.go` or a new `block_test.go`

- [x] write `TestABlockProposalNeedsNoGrant`: a SUGGEST batch of the block
  shape above, `deleteParagraphBullets` included, carries on a document at
  `LevelSuggest`, and the same batch without SUGGEST is refused
- [x] no production change is expected. If one is, stop: widening the guard
  is Nail's decision (CLAUDE.md, "Never")

### Task 3: The content, read strictly

**Files:**
- Create: `go/internal/propose/block.go`, `go/internal/propose/block_test.go`
- Modify: `go/internal/propose/doc.go`

- [x] write the failing tests first: a heading, a paragraph with bold, italic
  and a link, a bullet list and a numbered list come out as the right
  paragraphs, styles, runs and list kinds
- [x] refused by name, each with a test: a table, a nested list, a picture,
  a code block, a block quote, raw HTML, a thematic break, and empty content.
  gdoc's markers are not checked here: Task 8 extends the existing check
- [x] parse with goldmark, the way `internal/body` does. No new module
- [x] state the subset and each refusal in `propose/doc.go`, naming the tests

### Task 4: Where a block goes

**Files:**
- Modify: `go/internal/propose/span.go` or create `go/internal/propose/place.go`
- Create: tests beside it, with recorded documents in `testdata/`

- [x] write the failing tests first: `after` gives the start index of the
  paragraph after the anchor; a replace gives the start of the first and the
  end of the last whole paragraph
- [x] refused by name, each with a test: a quote not found exactly once;
  `replace_to` before `replace_from`; an anchor inside a table cell; a
  multi-tab document; a replace whose paragraphs hold a pending suggestion or
  a comment range (from `Document.CommentRanges`), naming every suggestion id
  and comment id; a replace whose run ends at the body's last paragraph,
  because Docs cannot delete the final newline; an `after` whose next element
  is a table, a contents list or a section break, because there is no
  paragraph start to insert at; an `after` on the last paragraph when the
  block does not end with a plain paragraph, or when that last paragraph is a
  list item
- [x] ➕ a replace whose run covers a table, the contents list or a section
  break is refused too: `deleteContentRange` over one would take it with the
  paragraphs
- [x] no index is stored anywhere: placement is computed from the read that
  the write is built from

### Task 5: The batch

**Files:**
- Modify: `go/internal/propose/propose.go` (or `block.go`)
- Create: tests with golden request bodies

- [x] write the failing tests first: the batch for an `after` block and for a
  replace, request by request, lengths in UTF-16 with a non-BMP character in
  one test
- [x] `deleteParagraphBullets` covers every new non-list paragraph, and no
  existing paragraph
- [x] the clearing `updateTextStyle` covers the whole insert and comes before
  the marks; test it with a next paragraph that opens bold and linked
- [x] the after-the-last-paragraph text shape, with its own golden body
- [x] write `TestALargeBlockStaysUnderThePeek`: a 60-paragraph block batch from
  this builder is under the guard's `maxPeek`, and name the content size
  limit as a constant that Task 7's `Check` enforces
- [x] the comment is anchored on the first new paragraph, with the robot
  prefix and no markdown (`internal/plaintext`)
- [x] ➕ the guard's `blockKinds` fixture now carries the indexes this builder
  computes, so its own claim that the two files describe one write is true

### Task 6: Three read-backs for a block

**Files:**
- Modify: `go/internal/propose/verify.go`, `verify_test.go`

- [x] `VerifyBlock` takes the id, the placement and the parsed paragraphs,
  not the `Proposal` type, which Task 7 grows
- [x] write the failing tests first, over recorded answers:
  - `suggestions_inline`: every new paragraph carries the insertion id, and
    for a replace every old paragraph carries a deletion id
  - `preview_without_suggestions`: the anchor paragraph, or the replaced run,
    reads as before, and the block's first line is absent. This is the
    direct-edit catch
  - `docx_anchored`: unchanged, the robot comment attached to text
- [x] a block whose read-back shows more than one suggestion id reports every
  one in `SuggestionIDs`, is `verified: false`, and warns that `withdraw`
  takes back only the first, which is the one the note records
- [x] the preview check answers "no answer" rather than a pass when the
  block's first line already appears elsewhere in the document

### Task 7: The type and the apply sequence

**Files:**
- Modify: `go/internal/propose/propose.go`, `doc.go`
- Create: `go/internal/propose/blockapply.go`, `blockapply_test.go`

- [x] `Proposal` grows `Kind` (empty or `"block"`), `After`, `ReplaceFrom`,
  `ReplaceTo` and `Content`. An empty `Kind` is the words kind, so every
  existing proposals file still reads
- [x] `Result` grows the fields a block reports: `after` or `replace_from`
  and `replace_to`, beside `suggestion_ids`, `comment_id` and the checks.
  `quoted` and `replacement` stay empty for a block
- [x] write the failing tests first, over a fake session: `ApplyBlock` reads,
  places, builds, posts once and calls `VerifyBlock`, the way `Apply` does. A refusal
  before the post sends nothing; everything after the post is reported, not
  raised; a lost answer is `verified: false` with the warning `Apply` gives
- [x] `Apply` dispatches on `Kind`, so the command has one call
- [x] `Check` for a block: exactly one placement form, content present, why
  present, content under Task 5's size limit, so the guard's "cannot be read"
  refusal is never what a person sees. This is the only place the placement
  form is checked. The words kind's `Check` is unchanged
- [x] ➕ a block carrying `quoted` or `replacement` is refused by name too:
  those belong to the words kind, and a block that carried them would have
  them silently ignored while its real placement came from somewhere else

### Task 8: The command and the note

**Files:**
- Modify: `go/cmd/gdoc/write.go`, its tests, and the command table's help
- Modify: `go/cmd/gdoc/doc.go` if a rule is stated there

- [x] write the failing tests first: the proposals file decodes a block entry
  strictly, and an unknown field or a block field on an entry with no
  `kind` is refused by name before anything is sent
- [x] extend the marker check around `write.go:507` to `after`,
  `replace_from`, `replace_to` and `content`, and add the block route to the
  route list in `internal/markers/markers.go`
- [x] a block and a words proposal in one file run in file order, each with
  its own read
- [x] the result and the note's `proposals` entry for a block record the id,
  the comment id and the `after` or `replace_from` quote
- [x] `gdoc help propose` describes the block entry

### Task 9: The skills

**Files:**
- Modify: `skills/gdoc-review/propose.md`, `skills/gdoc-review/SKILL.md`
- Modify: `skills/gdoc-align/SKILL.md`
- Modify: every skill's `needs` line that now depends on the block kind

- [x] review: when the right answer is a new or rewritten section, propose a
  block. Before a replace, run `suggestions` and `comments` over the range and
  tell the person what is in it; propose only with their answer
- [x] align: a section only the note has is proposed as a block after the
  paragraph before it, not left to the person
- [x] raise `needs` to the release this milestone will be: v2.7.0 on
  gdoc-review and gdoc-align, and `skillWants` says so too. The gate from
  `TestNoSkillNeedsAReleaseNobodyCut` is red until `make tag VERSION=v2.7.0`
  runs, as planned, so the tag is part of the merge (Post-Completion)
- [x] `TestEverySkillNamesOnlyCommandsAndFlagsTheBinaryHas` passes

### Task 10: Live acceptance

**Files:**
- Create: `go/internal/live/proposeblock_test.go`
- Modify: `go/internal/live/doc.go`

- [ ] `TestLiveProposeBlock`, under `GDOC_LIVE_WRITE`, in a document it
  creates in the test folder: an `after` block with a heading, body and both
  list kinds, and a replace over two paragraphs, after a list item so the
  bullet clearing is exercised. Both come back `verified: true` with one id.
  Numbered lists in SUGGEST are unmeasured until this test runs
- [ ] then `withdraw` each, and the read-back matches the document before
- [ ] trash what it made, and name every variable in `live/doc.go`

### Task 11: Verify acceptance criteria

- [ ] every decision above holds and is tested
- [ ] `make test` (raced), `make vet`, `make build`
- [ ] `docs/backlog/propose-cannot-add-paragraphs.md` is `git rm`ed in the
  last feature commit

### Task 12: [Final] Documentation

- [ ] `docs/guide/writing.md`: the block kind and its refusals. No new page
- [ ] README only if it names what propose can do
- [ ] move this plan to `docs/plans/completed/`

## Post-Completion

- `make tag VERSION=vX.Y.0` in the same sitting as the merge, because Task 9
  raises the skills' `needs`.
- Run the nightly after the merge, per the standing rule.
- `docs/backlog/prelude-inherits-a-list-marker.md` now has its measurement:
  `deleteParagraphBullets` over the prelude's range folds into the insertion
  id. Its fix is one request and a test, and can reuse Task 5's helper.
- Restyling the last paragraph of a block placed after the document's last
  paragraph is unmeasured. If Nail wants it, one more probe case settles it.
