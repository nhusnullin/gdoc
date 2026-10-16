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

### Task 5: the read sees where the page turns, and export writes `---`

Rewritten 2026-10-16 after the Task 4 measurement, from the first of the
three ways out in MEASURED.md: `internal/docs` decodes the flag Drive
writes, `read` prints the break it has been silent about since M13, and
`export` writes `---` for it. Claude took that way out under Nail's standing
"do it for me" of the same day, in his absence, and said so; the other two
are left in MEASURED.md for him to reverse to. The spec's `view` row changed
with it: `read` is no longer unchanged.

**Files:**
- Modify: `go/internal/docs/docs.go`, `go/internal/docs/walk.go`,
  `go/internal/docs/doc.go`, a fixture under `go/internal/docs/testdata/`
  in the measured shape, `go/internal/view/text.go`,
  `go/internal/view/doc.go`, `go/internal/export/project.go` (or the file
  that writes a placeholder line), `go/internal/export/doc.go`,
  `go/internal/live/roundtrip_test.go`, `go/internal/live/pagebreak_test.go`,
  `docs/v2/MEASURED.md`, `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md`,
  `docs/plans/2026-10-16-gdoc-v2-m19-page-break-spec.md`,
  `skills/gdoc-export/markers.md`
- Create: `go/internal/export/pagebreak_test.go`,
  `go/internal/view/pagebreak_test.go`

- [x] `docs.Paragraph` gains `PageBreakBefore bool`
      (`json:"page_break_before,omitempty"`), decoded from
      `paragraphStyle.pageBreakBefore`. `TestAParagraphKeepsWhetherThePageBreaksBeforeIt`
      reads a fixture in the measured shape: an empty `NORMAL_TEXT` paragraph
      carrying the flag, and a `HEADING_1` carrying it. The structure print
      carries the field where it carries the style.
- [x] `view`: a paragraph carrying the flag prints `[page break]` on a line of
      its own before it, through the same placeholder and with the same
      warning the native `pageBreak` element gets, so the two shapes read
      alike downstream. `TestABreakBeforeAParagraphPrintsAsAPageBreakLine`,
      `TestTheHousesFirstHeadingBreakPrintsBeforeTheHeading`, and
      `TestANativePageBreakElementStillPrintsWhereItStands`. `view/doc.go`
      says why the read now prints a line it printed nothing for.
- [x] `export`: a `[page break]` line standing on its own, from the flag or
      from a native element that is a paragraph's only content, is written as
      `---` on a line of its own between blank lines, with no warning for it.
      The one standing directly before the first body heading after the
      prelude strip is left out, because `build` puts that break back. A
      native `pageBreak` element inside a sentence keeps its marker and its
      warning. `TestAPageBreakOfItsOwnExportsAsThreeHyphens`,
      `TestTheFirstHeadingsBreakIsLeftOut`,
      `TestAPageBreakInsideASentenceKeepsItsMarker`,
      `TestAHorizontalRuleStillExportsAsItsMarker`. The line is written through
      the same place a picture line is, so a break in a table cell is handled
      the way a picture there is.
- [x] `roundtrip_test.go`: the `rules` entry becomes `page breaks the note
      wrote`, pairing `---`/`***` with `[page break]`; the `page breaks` entry
      becomes the house's own, the first heading's and column breaks, with
      its reason rewritten to what Task 4 measured. `markers.md:25` moves
      `[page break]` out of the no-text row into a line saying a break of its
      own comes home as `---`, and one inside a sentence stays a marker.
- [x] The documents: MEASURED.md's "three ways out" names the one taken and
      by whom; the DECISIONS.md entry of Task 1 and the spec's `view` and
      `export` rows say `read` prints `[page break]` where a published
      document breaks the page, which it printed nothing for since M13.
- [x] Live: `TestLivePageBreakShape` rerun, now printing `[page break]` for
      both kinds, and its MEASURED.md table gains a column for what `read`
      prints since this task. `TestLiveRoundTrip` on the kitchen sink passes
      with the new pairing. Document ids in the log.
- [x] `git commit -m "feat(docs,view,export): a page break is read where the page turns, and comes home as ---"`

**Task 5 outcome, 2026-10-16.** The read decodes `paragraphStyle.pageBreakBefore`
onto `docs.Paragraph`, `view` prints `[page break]` on a line of its own before
such a paragraph, and `export` passes `view.Options.PageBreak` so the same
break comes home as `---`. The option is Picture's rule for a break: the caller
names the line, the placeholder and its warning go with it, and a break inside
a sentence keeps both. A break carrying a suggestion id is never written as a
line, because the markers that say a suggestion is pending go around the span
and the line would land inside them.

One bullet above was written before the measurement settled and is not what the
code does. The `roundtrip_test.go` bullet asked for the note's `---`/`***`
paired with `[page break]`; since `export` writes `---`, both sides now spell
the break the same way and nothing needs excusing. The entry is
`page breaks the note wrote`, and all it carries is the reading that makes
`***` and `___` the same line as `---`. The old `page breaks` entry became
`the house's own breaks`, which is the first heading's, the prelude's and a
column break.

The fixture for all three packages is `go/internal/docs/testdata/page-breaks.json`,
written in the measured shape: a `HEADING_1` carrying the flag, an empty
`NORMAL_TEXT` paragraph carrying it, a native break inside a sentence, a native
break that is a paragraph's only content, and a horizontal rule.

**The live rerun, 2026-10-16.** `TestLivePageBreakShape` on
`1-GaZ0aqihDhoGn7cFqjZIPkNqH4pADp59QlqbOCOqc4`, trashed: still no `page_break`
run anywhere and still four `pageBreakBefore` paragraphs, and `read` now prints
`[page break]` for both kinds, once before `# 1-Where the page turns` and once
between the two body paragraphs. The MEASURED.md table carries that as its
fourth column.

`TestLiveExportRoundTrip` on the six notes:
`10sOVi7QytsRtMVNqL0bgUuSFQRqGSSrjoSFwEO4aLAY` (kitchen sink),
`1UK6myj7eQ1O2ZY9S7dmK890Ot29ZA_x78ouvKNVuSn0` (pictures),
`1LmYcykDdet-OsKIPXvv2PSDKWXJ9u2Jp3PAvPrYY4Sw` (policy),
`1rbTQn-wf7S4jy4kXui2uI_ZrgiPphHeFQWnh0h5KXT8` (minimal),
`1iC29VegQ781X85glrxESjUFINY5CGszYEOipqvKDTSw` (edge cases),
`15Frtb5oIa1XHEduA9GochJAW0KC2suBEVET2yWulpzo` (long). Policy, minimal and long
pass. The kitchen sink's unnamed lines are the same eight as on 2026-10-15, the
wrapped block quote and the list starting at seven of
`docs/backlog/round-trip-fails-on-a-wrapped-quote-and-a-list-starting-at-seven.md`.
Pictures and edge cases are red on constructs of their own, a reference-style
image with a base64 definition, task boxes against the house's ☐, and an
ampersand heading: all three notes have carried them since `744534f`, and the
first run of this milestone hid them behind a `head -40`.

**Not one unnamed line on any of the six is a `---` or a `[page break]`.** The
note's break and the export's `---` match each other directly, and the
footnote separator the file adds is still the `footnotes` entry's. So the
pairing holds and nothing was loosened to make it.

`make test` and `make vet` are green with the live variables unset. With them
set, `make test` is red on the three round trips above and on
`TestLiveRestyleKeepsAnchorsAndSuggestions`, which asks for
`GDOC_LIVE_ANCHOR_DOC_ID` and is unset in this session. Neither is this task's.

### Task 6: the help, the guide and the skills

**Files:**
- Modify: the command table in `go/cmd/gdoc/` (`gdoc help build`),
  `docs/guide/publishing.md`, `docs/guide/writing.md`,
  `docs/guide/exporting.md`, `docs/guide/reading.md`,
  `skills/gdoc-publish/SKILL.md`, `skills/gdoc-review/propose.md`,
  `go/internal/propose/place.go` and `span.go` wording, `README.md` if it
  lists what a note may hold, `go/boundary/guide_test.go`

- [x] `gdoc help build` says in one line that `---` on its own line is a
      page break, and `help --json` carries it. The guide's publishing page
      replaces "horizontal rules" with the page break and its two drops; the
      writing page's block refusal says a page break; the exporting page says
      a break comes home as `---` and a rule as `[rule]`; the reading page
      says `[page break]` prints where a published document breaks the page.
- [x] The publish skill reads `body.page_breaks` beside `body.footnotes` and
      says the number when it is above zero, judging nothing. The review
      skill's `propose.md:35` and `:148` say "page break (`---`)".
- [x] A `go/boundary/guide_test.go` test holds the publishing page to the
      sentence: `TestThePublishingPageSaysAThematicBreakIsAPageBreak`.
- [x] `git commit -m "docs: --- is a page break in the help, the guide and the skills"`

**Task 6 outcome, 2026-10-16.** `gdoc help build` carries a third note beside
the footnote's and the marks', and `TestHelpBuildSaysAThematicBreakIsAPageBreak`
in `cmd/gdoc/pagebreak_test.go` reads it off `help build --json` and out of the
words a person reads. `TestHelpBuildNamesFootnotesAndMarks` counted two notes
and now counts three. The frozen pipe golden `help.stdout.golden` moved by that
one note, which is the same one-line move `565d708` made when the marks note
was added: a note deliberately added to the table is not the bug that file's
comment warns about, and the diff is the note and nothing else.

Two of this task's file references were stale, the way Task 3's and Task 5's
were, and neither is a disagreement between the code and the spec. First,
`propose/place.go` and `span.go` already say "a page break": they name a
document's own `KindPageBreak` run, which a quote may not cross, and that
wording was right before this milestone. The construct that still called a
note's `---` a horizontal rule is `propose/block.go:219`, the block content
refusal, so that is the message this task renamed, with `doc.go` and
`block_test.go` beside it. `place.go`'s `KindHorizontalRule` arm stays "a
horizontal rule", because a document's rule is still a rule and has no spelling
in a note. Second, `propose.md:35` and `:148` are the quote-crossing sentences,
which name the break `read` prints as `[page break]`; spelling `---` there
would be wrong, so the "page break (`---`)" went into the content refusal list
at `:157`, which is what the bullet meant. `README.md` lists nothing about what
a note may hold, so it is unchanged.

The exporting page gained three table rows, `---` of its own, `[page break]`
inside a sentence and `[rule]`, and a paragraph in "What comes home as the
syntax it left as"; its footnote row and prose now say the last `---` in a file
is the separator and not a page break. The reading page gained the row and the
paragraph saying why the read printed nothing for a break until this milestone.
`make test` and `make vet` are green with the live variables unset.

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
