# gdoc v2 Milestone 18: footnotes on every route, plan

2026-10-14. The task list for the specification beside it,
`docs/plans/2026-10-14-gdoc-v2-m18-footnotes-spec.md`, whose decisions are
Nail's and settled. One commit per task. The spec holds the rules, the
decisions and the measurement. This plan repeats none of them. Where the code,
the spec or this plan disagree, the task stops and says so rather than
choosing.

Task 2 is the gate. It is a measurement, and the plan does not go past it
until the measurement is in MEASURED.md and says Drive carries the footnote.

## Principles

Serves: 1 and 4, as the spec says. Strains: the strict-decoding invariant,
written into DECISIONS.md by Task 1 before any code.

## Context (from discovery)

- `go/internal/body/body.go:205` turns `extension.Footnote` on, so the parser
  already produces `*east.FootnoteLink`, `*east.Footnote` and
  `*east.FootnoteList`. `appendInline` in `inline.go` skips a
  `*east.FootnoteLink` with `continue`, and the `*east.FootnoteList` arm in
  `block` warns and drops. `body/doc.go:136` is the section that says why.
- goldmark is silent in both failure shapes: `footnoteParser.Parse` returns
  nil for a reference with no definition, so `[^x]` stays prose, and
  `footnoteASTTransformer.Transform` removes a definition nobody references.
  So the refusal has to read the source lines, beside the markers check
  `Render` already runs.
- `merge` in `inline.go` drops a run whose `Text` is empty, which a footnote
  run is.
- `go/internal/render/render.go` writes twelve parts; `FirstMediaRelID` is 8
  and the shell relationships are fixed ids below it. `styles.go` writes
  `houseStyles`, nine of them, from `house.yaml`'s `styles:`;
  `house.go`'s `validateStyles` counts nine.
- `go/internal/view/text.go`: `emitter.footnote` and `appendFootnotes` print
  `[^n]` inline and the definitions after `---`, first-reference order.
  `TestASecondReferenceToOneFootnotePrintsTheSameNumber` pins a shape Docs
  cannot produce; under decision 2.2 B build never produces it either.
- `go/internal/export/project.go:255` `footnoteWarnings` says publish refuses
  a note that holds a footnote. `skills/gdoc-export/markers.md:103-108`,
  `skills/gdoc-export/SKILL.md:169` and `docs/v2/SPEC.md:493` say the same.
- `go/internal/live/roundtrip_test.go:85`: the `footnotes` entry of
  `knownDifferences` excuses the definition line on both sides, which would
  hide a footnote whose text never came back.
- `go/internal/propose/span.go` already refuses a quote crossing a footnote
  mark; `span_test.go` pins it. Nothing there changes.
- `go/internal/drift` reads nine named styles by id on both halves and never
  the footnote style. Unchanged.
- `docs/v2/MEASURED.md` has no row on Drive's import of footnotes.
  DECISIONS.md 2026-09-09 is a restyle surviving footnotes, not an import.

## Development Approach

TDD, one commit per task, the test first and failing. Every number a test
states is a literal. A task that finds the spec and the code disagree stops
and says so in the task log. `make test` keeps the race flag.

## Validation Commands

```bash
make test
make vet
cd go && go test ./internal/body -run Footnote -v
cd go && GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1 go test ./internal/live -run 'Footnote' -v
```

## Progress Tracking

Checkboxes below. A task is done when its commit is on the branch.

## Implementation Steps

### Task 1: the DECISIONS.md entry, SPEC.md, CLAUDE.md

**Files:**
- Modify: `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md`, `CLAUDE.md`

- [ ] A DECISIONS.md entry dated 2026-10-14, "A footnote is carried on every
      route", with its register row, naming the two strains the spec names
      and the four decisions of the decision page.
- [ ] `docs/v2/SPEC.md:493` stops saying publish refuses a footnote and says
      what build does with one.
- [ ] `CLAUDE.md`'s `go/internal/render/` row reads "the thirteen parts and
      the logo", and `TestCLAUDEmdIsUnderTheCeiling` still passes.
- [ ] `git commit -m "docs: the footnote decision, and the spec says what build does with one"`

### Task 2: the measurement, Drive's import of a Word footnote

**Files:**
- Create: `go/internal/live/footnoteprobe_test.go`,
  `go/internal/live/testdata/footnote-probe.docx`
- Modify: `docs/v2/MEASURED.md`, `go/internal/live/doc.go`

- [ ] The fixture is the built `04-minimal.md` with a `word/footnotes.xml`
      added by hand, one note, the reference run in the first body paragraph,
      and the relationship and content type rows. It is a fixture, not
      generated, so the probe asks Drive about a footnote before the
      generator knows how to write one.
- [ ] `TestLiveFootnoteImportProbe`, behind `GDOC_LIVE_TEST=1` and
      `GDOC_LIVE_WRITE=1`, uploads it with conversion into the test folder
      through `liveWriteSession`, reads it back with `driftReadURL`, prints
      whether the body holds a `footnoteReference` element and whether the
      document holds a `footnotes` map with the note's text, exports it as
      docx and prints whether `word/footnotes.xml` came back, then trashes
      it with `trashLive`. It fails on a read or an export that failed and on
      nothing else: the answer is Google's.
- [ ] `TestTheFootnoteProbeFixtureIsADocx` runs in `make test` and opens the
      fixture with `docx.Parse`, so a fixture that stopped being a zip is
      found by the suite.
- [ ] The three answers go into MEASURED.md by hand, dated, with the document
      id the run printed. `live/doc.go` names the test.
- [ ] **Stop here if Drive dropped the footnote.** Say so in the task log and
      end the run.
- [ ] `git commit -m "test(live): measure Drive's import of a Word footnote"`

### Task 3: a footnote reference is a run

**Files:**
- Modify: `go/internal/body/inline.go`, `go/internal/body/build.go`
- Test: `go/internal/body/footnote_test.go`

- [ ] `TestAFootnoteReferenceCarriesItsNumber`: `Run` gains `Footnote int`,
      1-based, from `east.FootnoteLink.Index`; `appendInline` emits it in the
      `*east.FootnoteLink` arm; `FootnoteBacklink` stays skipped.
- [ ] `TestAFootnoteRunSurvivesMerge`: `merge` keeps a run whose `Footnote`
      is set although its `Text` is empty.
- [ ] `addRuns` writes
      `<w:r><w:rPr><w:rStyle w:val="FootnoteReference"/></w:rPr><w:footnoteReference w:id="N"/></w:r>`
      for such a run, before its own empty-text guard.
- [ ] `git commit -m "feat(body): a footnote reference is a run with its number"`

### Task 4: the footnote list renders, and what does not line up is refused

**Files:**
- Modify: `go/internal/body/body.go`, `go/internal/body/doc.go`
- Test: `go/internal/body/footnote_test.go`

- [ ] `TestAFootnoteBecomesAFootnoteTextParagraph`: the `*east.FootnoteList`
      arm renders each `*east.Footnote`'s paragraphs into
      `Result.Footnotes []Footnote{ID int; Blocks []*etree.Element}`, each
      paragraph in the `FootnoteText` style opening with a `w:footnoteRef`
      run. `Counts` gains `Footnotes`.
- [ ] `TestFootnotesAreNumberedInReferenceOrderNotDefinitionOrder`.
- [ ] `TestAReferenceWithNoDefinitionIsRefusedByLine` and
      `TestADefinitionNobodyReferencesIsRefusedByLine`: a source-line check
      in `Render` beside the markers check, skipping an escaped `\[`, with
      the refusals "line 7: the footnote [^x] has no definition" and
      "line 12: the footnote [^y] is defined and never referenced".
- [ ] `TestAFootnoteReferencedTwiceIsTwoNotesWithOneText` (decision 2.2 B).
- [ ] `TestAFootnoteMarkInsideAHeadingIsRefusedByLine`.
- [ ] `TestAListInsideAFootnoteWarnsAndIsLeftOut`, and the same shape for a
      table, a picture, a code block and a heading, one test each or one
      table test.
- [ ] `headingAnchors`, `linkedAnchors` and `shallowestHeadingLevel` keep
      skipping the list: `TestAHeadingInsideAFootnoteIsNoAnchor` and
      `TestAHeadingInsideAFootnoteSetsNoHeadingDepth` still pass, and
      `TestAFootnoteWarnsAndIsLeftOut` and `TestEveryFootnoteIsNamedByItsOwnLine`
      are retired.
- [ ] `body/doc.go:136`'s section is rewritten to say what is carried, what is
      refused and what is warned about, naming every test above.
- [ ] `git commit -m "feat(body): footnotes render, and ones that do not line up are refused"`

### Task 5: the footnotes part, the two styles, the tenth house style

**Files:**
- Modify: `go/internal/render/render.go`, `go/internal/render/styles.go`,
  `go/internal/render/doc.go`, `go/internal/house/house.yaml`,
  `go/internal/house/house.go`
- Test: `go/internal/render/footnotes_test.go`, `go/internal/house/house_test.go`

- [ ] `TestTheTenStylesAreThere` in house: `styles.footnote_text` at 10pt,
      single-spaced, 0 above and below, in the body face; `validateStyles`
      counts ten.
- [ ] `TestTheFootnotesPartCarriesTheTwoSeparators`: `word/footnotes.xml` is
      always written, with `w:type="separator" w:id="-1"` and
      `w:type="continuationSeparator" w:id="0"`, then one `w:footnote` per
      note from `Result.Footnotes`.
- [ ] `TestTheFootnotesPartIsDeclaredAndRelated`: the content type
      `wml + "footnotes+xml"` and the relationship row. The footnotes
      relationship takes the next fixed id and `FirstMediaRelID` moves by one;
      the image goldens are regenerated and the diff is read, not assumed.
- [ ] `TestAFootnoteReferenceIsASuperscriptCharacterStyle`: `FootnoteReference`
      as a `w:type="character"` style with `w:vertAlign superscript`, a
      constant with its reason, since it is not a measurement.
- [ ] `TestBuildWritesTheThirteenPartsAndTheLogo`, renamed from twelve.
- [ ] `render/doc.go` names the part and the two styles.
- [ ] `git commit -m "feat(render): the footnotes part, and the two styles it needs"`

### Task 6: the goldens carry footnotes

**Files:**
- Modify: `go/internal/body/testdata/docs/01-kitchen-sink.md`,
  `go/internal/body/testdata/docs/05-edge-cases.md`, their goldens
- Create: `go/internal/body/testdata/golden/01-kitchen-sink.footnotes.xml`

- [ ] Two footnotes in the kitchen sink, the second defined first; an edge
      case with a word label and one with a reference used twice.
- [ ] `TestTheSixDocumentsRenderToTheirGoldens` also compares
      `Result.Footnotes` against `<name>.footnotes.xml` where one exists.
- [ ] `go test ./internal/body -update`, and the diff read.
- [ ] `git commit -m "test(body): the goldens carry footnotes"`

### Task 7: export stops saying publish refuses, and the round trip carries one

**Files:**
- Modify: `go/internal/export/project.go`, `go/internal/export/doc.go`,
  `skills/gdoc-export/markers.md`, `skills/gdoc-export/SKILL.md`,
  `go/internal/live/roundtrip_test.go`

- [ ] `TestAFootnoteIsWrittenAndWarned` shrinks to what is still true: the
      suggestion-and-comment loss inside a footnote, which is the backlog
      item. `footnoteWarnings` loses its first sentence.
- [ ] The two skill pages stop saying publish refuses a footnote and say that
      a footnote comes home as `[^n]` and goes back as one.
- [ ] `knownDifferences`'s `footnotes` entry keeps the `---` line on the
      `Added` side only, drops `Gone`, and gains a `Note` rule that renumbers
      `[^label]` to its ordinal in first-reference order, so `[^why]: text`
      matches `[^1]: text`. One of `roundTripNotes` carries a footnote.
- [ ] `git commit -m "feat(export): a footnote goes home and comes back"`

### Task 8: the live acceptance

**Files:**
- Create: `go/internal/live/footnote_test.go`
- Modify: `go/internal/live/doc.go`, `docs/v2/MEASURED.md`

- [ ] `TestLivePublishCarriesAFootnoteIntoDocs`: publishes the kitchen sink
      through the publish path, reads it back bare, asserts a
      `footnoteReference` in the body and the note's text in the `footnotes`
      map, exports it and asserts `word/footnotes.xml` is in the zip, then
      trashes it.
- [ ] `TestLiveExportRoundTrip` over the note with a footnote passes.
- [ ] MEASURED.md gets the acceptance row beside the probe's.
- [ ] `git commit -m "test(live): a footnote survives publish, read-back and export"`

### Task 9: the documents

**Files:**
- Modify: `docs/guide/` (the build and export pages), `README.md` if it
  lists what a note may hold, `skills/gdoc-publish/SKILL.md`

- [ ] The guide's build page shows a footnote in the example note and names
      the three refusals. The export page says what comes home.
- [ ] `gdoc help build` names footnotes in one line.
- [ ] `git commit -m "docs: footnotes in the guide and the skills"`

### Task 10: verify acceptance, and M18 completed

- [ ] Every acceptance item in the spec, run and ticked in the task log.
- [ ] `make test`, `make vet`, `make dist`.
- [ ] Both plan files move to `docs/plans/completed/`.
- [ ] `git commit -m "chore: M18 completed"`

## Post-Completion

Nail runs revmux (`ralphex --external-only`) until a clean round, merges, and
triggers the nightly. The backlog item on suggestions inside footnotes stays
open and now has a shape to build on.
