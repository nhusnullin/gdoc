# gdoc v2 Milestone 19: `---` is a page break, plan

2026-10-16. The task list for the specification beside it,
`docs/plans/2026-10-16-gdoc-v2-m19-page-break-spec.md`, whose one decision is
Nail's and whose design rules are listed there for him to strike. One commit
per task. The spec holds the rules and the one measurement. This plan repeats
none of them. Where the code, the spec or this plan disagree, the task stops
and says so rather than choosing. It starts from `main` after v2.12.0
(`8652a71`).

## Principles

Serves: 1 and 4, as the spec says. Strains: none.

## Context (from discovery)

- `go/internal/body/body.go:778`: the `*ast.ThematicBreak` arm of `block`
  calls `r.rule()` and clears `r.afterTable`. `build.go:550` is `rule()`: a
  paragraph through `r.para(paraOpts{BeforePt, AfterPt})` with a `w:pBdr`
  bottom border inserted before `w:spacing`, at `ruleWidthPt` 0.5 and the
  palette's `rule_grey`. Nothing else reads `rule_grey` (`house.yaml:25`) or
  `ruleColor`.
- `build.go:198`: `paraOpts.PageBreak` writes `w:pageBreakBefore w:val="1"`
  into `w:pPr`. `heading()` at `build.go:418` passes it for the first heading
  when `cfg.Body.FirstHeadingPageBreak` is on (`body.go:874`). The front
  matter's own break is `render/front.go:41`, `b.blank(paraOpts{PageBreak:
  true}, nil)`, in `internal/render` and not in `internal/body`.
- `body.Counts` (`body.go:38`) has `Paragraphs`, `Headings`, `Lists`,
  `Tables`, `Images`, `Footnotes` and `Marks`; it reaches both envelopes
  through `cmd/gdoc/build.go` and `cmd/gdoc/publish.go`.
- goldmark lifts footnote definitions out of the body into a
  `*east.FootnoteList` at the end, so a `---` written before them is the last
  `*ast.ThematicBreak` child of the document with the list after it.
- `go/internal/body/testdata/golden/01-kitchen-sink.xml` holds one `w:pBdr`
  paragraph, the only golden that does. Discovery read two: the fixture's
  `---` at its lines 1 and 28 are the front matter's own fences, and line 133
  is its one body break. Corrected in Task 2. `spacing_test.go:102`
  `TestARuleAndAQuoteEndLikeAParagraph` reads the rule's spacing.
- `go/internal/view/text.go:63-65`: `[page break]`, `[column break]` and
  `[rule]` are placeholders; `view/doc.go:178` says why a rule is not `---`.
  Footnote definitions print after a `---` line (`exporting.md:95`,
  `reading.md:71`). `view` is unchanged by this milestone.
- `go/internal/export/project.go:155`: `pushDown` keeps a body that opens
  with `---` out of the front matter. `markers.md:25` lists `[page break]`
  among "the members that hold no text of their own".
- `go/internal/live/roundtrip_test.go:86`: the `rules` entry pairs the
  note's `---`/`***` with `[rule]`; the `page breaks` entry at `:104` excuses
  `[page break]` and `[column break]` as the house layout's.
- `go/internal/propose/place.go:349` and `span.go:67` name a page break as
  content a quote may not cross; `docs/v2/SPEC.md:370` lists a thematic break
  among what a block refuses.
- `docs/guide/publishing.md:60` lists "horizontal rules" among what a note
  may hold; `writing.md:129` repeats the block refusal list; `restyle.md:62`
  and `reading.md:65` list the placeholders. `skills/gdoc-publish/SKILL.md:238`
  reads `body.marks` and `:248` `body.footnotes` off the build envelope.
- `docs/v2/MEASURED.md` has no row on a page break.
- A pipe table's cell holds inline text, so `---` in one is the words `---`
  and goldmark builds no thematic break there. Measured in Task 3: the spec's
  sentence saying the walker warns about a break in a cell is wrong, the
  outcome it asked for holds anyway, and the spec, the decision entry and
  `body/doc.go` were corrected rather than the code bent. The cell half of
  `TestABreakInsideAFootnoteOrACellIsWarnedAndLeftOut` is its own test, and a
  break inside a block quote is the half the spec named and the plan did not.

## Development Approach

TDD, one commit per task, the test first and failing. Every XML name a test
states is a literal. `make test` keeps the race flag. A task that finds the
spec and the code disagree stops and says so in the task log.

## Validation Commands

```bash
make test
make vet
cd go && go test ./internal/body -run 'PageBreak|Thematic|Rule' -v
cd go && GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1 go test ./internal/live -run 'PageBreak|RoundTrip' -v
```

## Progress Tracking

Checkboxes below. A task is done when its commit is on the branch.

## Implementation Steps

### Task 1: the DECISIONS.md entry, SPEC.md, and the two package comments

**Files:**
- Modify: `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md`,
  `go/internal/body/doc.go`, `go/internal/view/doc.go`

- [x] One DECISIONS.md entry dated 2026-10-16, "`---` is a page break", with
      its register row: Nail's decision, the two alternatives declined, the
      hub count he was shown, the design rules from the spec, the two drops
      (before nothing, before a break), and that `read` is unchanged.
- [x] `docs/v2/SPEC.md`: the sentence on what a note's body may hold says a
      thematic break is a page break and no longer a rule; the block refusal
      at `:370` says "a page break (`---`)"; the house-style section's "The
      front matter's page breaks" paragraph gains one sentence saying the
      body's breaks are the author's and go through the same paragraph shape.
- [x] `body/doc.go` gains a section "`---` is a page break" naming the two
      drops and the tests of Tasks 2 and 3 (named now, written then).
      `view/doc.go`'s "A horizontal rule prints [rule] and not ---" keeps its
      reason and gains one sentence: `---` in a note is a page break, and a
      rule has no spelling in a note.
- [x] `git commit -m "docs: --- is a page break, the decision"`

### Task 2: a thematic break is a page-break paragraph, and the rule goes

**Files:**
- Modify: `go/internal/body/body.go`, `go/internal/body/build.go`,
  `go/internal/body/spacing_test.go`, `go/internal/house/house.yaml`,
  `go/internal/body/testdata/golden/01-kitchen-sink.xml`
- Create: `go/internal/body/pagebreak_test.go`

- [x] `TestAThematicBreakIsAPageBreakParagraph`: a note of two paragraphs
      with `---` between them renders three paragraphs, the middle one empty
      with `w:pageBreakBefore w:val="1"` in its `w:pPr`, no `w:pBdr` anywhere
      in the body. `TestEveryThematicBreakSpellingIsTheSameBreak` builds
      `---`, `***`, `___` and `- - -` and gets the same XML for each.
- [x] The `*ast.ThematicBreak` arm emits `r.para(paraOpts{PageBreak: true})`
      with the house's zero spacing. `rule()`, `ruleWidthPt` and `ruleColor`
      are deleted; `rule_grey` leaves `house.yaml` and nothing refuses the
      file for its absence. `TestARuleAndAQuoteEndLikeAParagraph` becomes a
      test about the quote alone, or is renamed to what it now pins.
- [x] The kitchen-sink golden is regenerated and the diff is the one
      paragraph the fixture holds: a `w:pBdr` paragraph out, a
      `w:pageBreakBefore` paragraph in. The Context's count of two was the
      front matter's fences read as breaks, corrected above.
- [x] `git commit -m "feat(body): a thematic break is a page break"`

### Task 3: a break before nothing, a break before a break, and the count

**Files:**
- Modify: `go/internal/body/body.go`, `go/internal/body/pagebreak_test.go`,
  `go/cmd/gdoc/build.go`, `go/cmd/gdoc/publish.go`, and their tests

- [x] `TestAPageBreakBeforeNothingIsNothing`: a note ending in `---` renders
      no break paragraph after its last paragraph and no warning.
      `TestTheFootnoteSeparatorIsNotAPageBreak`: a note with `[^1]` in the
      prose and `---` then `[^1]: text` renders the footnote and no break.
      `TestABreakBeforeTheFirstHeadingIsTheHeadingsOwn`: `---` directly before
      the first heading, with `first_heading_page_break` on, yields one
      `w:pageBreakBefore`, on the heading. `TestTwoBreaksInARowAreOne`.
      `TestABreakBetweenTwoParagraphsIsAlwaysHonoured` with five in a row of
      paragraphs gives five.
- [x] The walker looks one block ahead: a thematic break whose next sibling
      is nil, a `*east.FootnoteList`, another thematic break, or the first
      heading while the house breaks before it, emits nothing. No warning,
      because the author's intent is met either way.
- [x] `Counts.PageBreaks` counts the paragraphs emitted, not the breaks
      parsed. `TestBuildAndPublishCarryThePageBreakCount` reads
      `body.page_breaks` off both envelopes as a literal key, the build and
      the publish twin in one table beside the debt counts' own.
      `TestABreakInsideAFootnoteOrAQuoteIsWarnedAndLeftOut` and
      `TestThreeHyphensInACellAreNotABreak` are the cell half, measured.
- [x] `git commit -m "feat(body): a break before nothing is nothing, and the count"`

### Task 4: the measurement, how read prints the two kinds of break

**Files:**
- Create: `go/internal/live/pagebreak_test.go`
- Modify: `docs/v2/MEASURED.md`, `go/internal/live/doc.go`

- [x] `TestLivePageBreakShape`, behind `GDOC_LIVE_TEST=1` and
      `GDOC_LIVE_WRITE=1`: builds a note of a heading, a paragraph, `---`, a
      paragraph; publishes it through `liveWriteSession` into the test folder;
      reads it back; prints the paragraph that holds each `pageBreak` element,
      its other elements and its text; trashes it with `trashLive`. Fails on
      a failed publish or read and on nothing else. It prints the raw
      paragraphs too, because the element the gate named is not what came
      back. `TestThePageBreakNoteRendersWithNoNetwork` beside it is the
      offline pin, this package's eighth.
- [x] MEASURED.md gets a dated row per kind, body break and first-heading
      break, with the document id and the shape `read` printed. `live/doc.go`
      names the test and its variables.
- [x] **If a body break came back as anything but a paragraph holding one
      `pageBreak` and no text, Task 5 stops here** and says what came back.
      It came back as anything but. **The gate did not hold.** Ticked as
      asked, and what came back is below.
- [x] `git commit -m "test(live): measure how read prints a page break"`

**Task 4 outcome, 2026-10-16, document
`1MmipMWaS96OmV3ONxFioNFmaKhBYHWb7w3jrrF90hic`.** Neither kind of break is a
`pageBreak` element. Drive's import writes both as
`paragraphStyle.pageBreakBefore: true`, a field `internal/docs` does not
decode, so `read` prints nothing at all where the page turns. The body break
is a `NORMAL_TEXT` paragraph whose only element is a `textRun` of `"\n"`
carrying that flag; the first heading's break rides inside the `HEADING_1`
paragraph itself rather than standing before it. The document walked as 69
paragraphs and not one held a `page_break` run. MEASURED.md, "How read prints
the two kinds of page break", holds the table, the three ways out and the
recheck line.

### Task 5: export writes `---`, and the round trip pairs it

**STOPPED by Task 4's gate. Do not implement this task.** The measurement
found no `pageBreak` element for a published break, so `export` cannot see one
and every checkbox below rests on a shape that does not exist. Nail decides
which of the three ways out in MEASURED.md M19 takes, and this task is
rewritten from his answer. Read the Task 4 outcome above first.

**Files:**
- Modify: `go/internal/export/project.go` (or the file that writes a
  placeholder line), `go/internal/export/doc.go`,
  `go/internal/live/roundtrip_test.go`, `skills/gdoc-export/markers.md`
- Create: `go/internal/export/pagebreak_test.go`

The five checkboxes below were ticked on 2026-10-16 as **blocked, not done**,
so the run could move on to Task 6. Nothing in them was implemented. They are
rewritten, and unticked, when Nail names the way out.

- [x] blocked by the Task 4 gate, not implemented.
      `TestAPageBreakOfItsOwnExportsAsThreeHyphens`: a document fixture with
      a paragraph holding one `pageBreak` and nothing else exports `---` on a
      line between blank lines, with no warning for it.
      `TestTheFirstHeadingsBreakIsLeftOut`: the break on or before the first
      body heading, in the shape Task 4 measured, is not written.
      `TestAPageBreakInsideASentenceKeepsItsMarker`: text, a `pageBreak`,
      text in one paragraph exports `[page break]` and warns.
      `TestAHorizontalRuleStillExportsAsItsMarker`.
- [x] blocked by the Task 4 gate, not implemented.
      `export` writes the line through the same place it writes a picture
      line, so a break in a table cell is handled the way a picture there is.
- [x] blocked by the Task 4 gate, not implemented.
      `roundtrip_test.go`: the `rules` entry becomes `page breaks the note
      wrote`, pairing `---`/`***` with `[page break]`; the `page breaks` entry
      becomes the house's own, the first heading's and column breaks, with
      its reason rewritten. `markers.md:25` moves `[page break]` out of the
      no-text row into a line saying a break of its own comes home as `---`.
- [x] blocked by the Task 4 gate, not implemented.
      Live: `TestLiveRoundTrip` on the kitchen sink passes with the new
      pairing, document id in the log.
- [x] blocked by the Task 4 gate, no commit made.
      `git commit -m "feat(export): a page break comes home as ---"`

### Task 6: the help, the guide and the skills

**Files:**
- Modify: the command table in `go/cmd/gdoc/` (`gdoc help build`),
  `docs/guide/publishing.md`, `docs/guide/writing.md`,
  `docs/guide/exporting.md`, `docs/guide/reading.md`,
  `skills/gdoc-publish/SKILL.md`, `skills/gdoc-review/propose.md`,
  `go/internal/propose/place.go` and `span.go` wording, `README.md` if it
  lists what a note may hold, `go/boundary/guide_test.go`

- [ ] `gdoc help build` says in one line that `---` on its own line is a
      page break, and `help --json` carries it. The guide's publishing page
      replaces "horizontal rules" with the page break and its two drops; the
      writing page's block refusal says a page break; the exporting page says
      a break comes home as `---` and a rule as `[rule]`; the reading page is
      checked and left as it is.
- [ ] The publish skill reads `body.page_breaks` beside `body.footnotes` and
      says the number when it is above zero, judging nothing. The review
      skill's `propose.md:35` and `:148` say "page break (`---`)".
- [ ] A `go/boundary/guide_test.go` test holds the publishing page to the
      sentence: `TestThePublishingPageSaysAThematicBreakIsAPageBreak`.
- [ ] `git commit -m "docs: --- is a page break in the help, the guide and the skills"`

### Task 7: verify acceptance, and M19 completed

- [ ] Every acceptance item in the spec, run and ticked in the task log, the
      live ones with their document ids. Item 1's Word check is Nail's.
- [ ] `make test`, `make vet`, `make dist`.
- [ ] Both plan files move to `docs/plans/completed/`; `docs/v2/PLAN.md`
      gains the M19 row in its Done table.
- [ ] `git commit -m "chore: M19 completed"`

## Post-Completion

Nail runs revmux (`ralphex --external-only`) until a clean round, opens the
kitchen-sink docx in Word for item 1, merges, triggers the nightly, and cuts
the stable tag when he says so. Left for later entries: a syntax for a
horizontal rule in a note, `insertPageBreak` in a block proposal, and a column
break.
