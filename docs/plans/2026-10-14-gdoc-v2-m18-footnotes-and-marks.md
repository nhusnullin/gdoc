# gdoc v2 Milestone 18: footnotes, and the marks, plan

2026-10-14. The task list for the specification beside it,
`docs/plans/2026-10-14-gdoc-v2-m18-footnotes-and-marks-spec.md`, whose
decisions are Nail's and settled. One commit per task. The spec holds the
rules, the decisions and the two measurements. This plan repeats none of
them. Where the code, the spec or this plan disagree, the task stops and says
so rather than choosing.

Tasks 2 and 3 are the gates. Each stops only its own part: Part A is Tasks 5
to 7, Part B is Tasks 8 to 11, and Tasks 4, 12 and 13 serve both. A part that
stopped at its gate is skipped in the shared tasks and said so in the task
log.

## Principles

Serves: 1, 2 and 4, as the spec says. Strains: the four the spec names,
written into DECISIONS.md by Task 1 before any code.

## Context (from discovery)

- `go/internal/body/body.go:205` turns `extension.Footnote` on, so the parser
  already produces `*east.FootnoteLink`, `*east.Footnote` and
  `*east.FootnoteList`. `appendInline` in `inline.go` skips a
  `*east.FootnoteLink` with `continue`, and the `*east.FootnoteList` arm in
  `block` warns and drops. `body/doc.go:136` is the section that says why.
- goldmark is silent in both footnote failure shapes: `footnoteParser.Parse`
  returns nil for a reference with no definition, so `[^x]` stays prose, and
  `footnoteASTTransformer.Transform` removes a definition nobody references.
  The refusal has to read the source lines, beside the markers check
  `Render` already runs.
- `go/internal/body/mark.go` is the `==text==` extension: a delimiter pair
  producing a `*Highlight` inline node. `inline.go:102` folds it into
  `Run.Highlight`, a string, and `build.go:273` turns that into
  `w:highlight w:val="yellow"`. An emoji at the start of the highlight is
  today ordinary text inside it. `TestAMarkedSpanIsItsOwnNodeKind` is the pin.
- Obsidian 1.14's changelog: "Add a color emoji (🔴, 🟠, 🟡, 🟢, 🔵, 🟣) to
  the start of a highlight to change its color." Six emoji, one position.
- `merge` in `inline.go` drops a run whose `Text` is empty, which a footnote
  run is, and joins neighbours by their formatting, which must now include
  the mark.
- `runOpts` in `build.go:75` carries `Color` and `Highlight`; `addRuns`
  writes `w:color` and `w:highlight`. No `w:shd` on a run, no `w:rStyle`.
- goldmark ships no bracketed-span extension. Pandoc's grammar: `[` inline
  content `]` immediately followed by `{` attributes `}`; a `[` not followed
  by `]{` is ordinary text, and `[words](url)` must stay a link. This
  milestone reads exactly one attribute, `.pencil`.
- `go/internal/render/render.go` writes twelve parts; `FirstMediaRelID` is 8
  and the shell relationships are fixed ids below it. `styles.go` writes
  `houseStyles`, nine paragraph styles, and no character style;
  `house.go`'s `validateStyles` counts nine.
- `go/internal/house/house.yaml` carries `highlight_yellow: '#FFFF00'` under
  `palette:` for the cover's placeholders; `render/front.go:55` writes them
  with Word's named yellow. Untouched here.
- `go/internal/view/text.go`: `emitter.footnote` and `appendFootnotes` print
  `[^n]` inline and the definitions after `---`.
  `TestASecondReferenceToOneFootnotePrintsTheSameNumber` pins a shape Docs
  cannot produce; under decision 2.2 B build never produces it either.
- `go/internal/docs/walk.go` reads no `textStyle.backgroundColor` and no
  `foregroundColor`. Export cannot see a colour until the read keeps it.
- `go/internal/export/project.go:96` calls `view.Project` with options;
  `project.go:255` `footnoteWarnings` says publish refuses a note that holds
  a footnote. `skills/gdoc-export/markers.md:103-108`,
  `skills/gdoc-export/SKILL.md:169` and `docs/v2/SPEC.md:493` say the same.
- `go/internal/live/roundtrip_test.go:85`: the `footnotes` entry of
  `knownDifferences` excuses the definition line on both sides.
- `body.Counts` reaches both envelopes: `cmd/gdoc/build.go:50` and
  `cmd/gdoc/publish.go:62`.
- `docs/v2/MEASURED.md` has no row on Drive's import of footnotes, of run
  shading or of a run's text colour.

## Development Approach

TDD, one commit per task, the test first and failing. Every number, colour
and emoji a test states is a literal. A task that finds the spec and the code
disagree stops and says so in the task log. `make test` keeps the race flag.

## Validation Commands

```bash
make test
make vet
cd go && go test ./internal/body -run 'Footnote|Mark' -v
cd go && GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1 go test ./internal/live -run 'Footnote|Mark' -v
```

## Progress Tracking

Checkboxes below. A task is done when its commit is on the branch.

## Implementation Steps

### Task 1: the two DECISIONS.md entries, SPEC.md, CLAUDE.md, the body package comment

**Files:**
- Modify: `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md`, `CLAUDE.md`,
  `go/internal/body/doc.go`

- [x] Two DECISIONS.md entries dated 2026-10-14, each with its register row:
      "A footnote is carried on every route", naming its two strains and the
      three decisions of the page; and "The marks: Obsidian's six colours
      and one grey", carrying the palette table, its two strains, what was
      dropped, and why `read` is unchanged.
- [x] `docs/v2/SPEC.md:493` stops saying publish refuses a footnote; SPEC
      gains a section on marks beside the one on the house style.
- [x] `CLAUDE.md`'s `go/internal/render/` row reads "the thirteen parts and
      the logo"; `TestCLAUDEmdIsUnderTheCeiling` still passes.
- [x] `body/doc.go`'s footnote section and its `==mark==` sentence are
      rewritten to say what is carried, refused and warned about.
- [x] `git commit -m "docs: footnotes and marks, the two decisions"`

### Task 2: the measurement, Drive's import of a Word footnote

**Files:**
- Create: `go/internal/live/footnoteprobe_test.go`,
  `go/internal/live/testdata/footnote-probe.docx`
- Modify: `docs/v2/MEASURED.md`, `go/internal/live/doc.go`

- [x] The fixture is the built `04-minimal.md` with a `word/footnotes.xml`
      added by hand, one note, the reference run in the first body paragraph,
      and the relationship and content type rows.
- [x] `TestLiveFootnoteImportProbe`, behind `GDOC_LIVE_TEST=1` and
      `GDOC_LIVE_WRITE=1`, uploads it with conversion through
      `liveWriteSession`, reads it back with `driftReadURL`, prints whether
      the body holds a `footnoteReference` and whether the document holds a
      `footnotes` map with the note's text, exports it and prints whether
      `word/footnotes.xml` came back, then trashes it with `trashLive`. It
      fails on a read or an export that failed and on nothing else.
- [x] `TestTheFootnoteProbeFixtureIsADocx` runs in `make test`.
- [x] MEASURED.md gets the three answers by hand, dated, with the document
      id. `live/doc.go` names the test.
- [x] **If Drive dropped the footnote, Part A stops here.** Drive did not
      drop it. All three answers came back yes on 2026-10-14: one
      `footnoteReference` in the body, a `footnotes` map keyed `kix.fn1`
      carrying the note's words, and `word/footnotes.xml` in the export with
      those words in it. Part A goes ahead, and nothing of Task 1's is taken
      back out.
- [x] `git commit -m "test(live): measure Drive's import of a Word footnote"`

### Task 3: the measurement, Drive's import of run shading and text colour

**Files:**
- Create: `go/internal/live/markprobe_test.go`,
  `go/internal/live/testdata/mark-probe.docx`
- Modify: `docs/v2/MEASURED.md`, `go/internal/live/doc.go`

- [x] The fixture is the built `04-minimal.md` with eight runs added by hand
      in the first body paragraph: `w:shd w:fill` for each of the six
      highlight hexes, `w:color` #9A9A9A, and `w:highlight yellow`.
- [x] `TestLiveMarkImportProbe`, behind the two live variables, uploads it,
      reads it back bare, prints each run's text beside the
      `backgroundColor` and `foregroundColor` Docs reports as hex, runs one
      in-place restyle over the document the way `live/restyle_test.go` does,
      `restyle.TabRequests` then `restyle.Apply`, and reads the eight again,
      then trashes it. It fails on a read, an upload or a
      restyle that failed and on nothing else.
- [x] `TestTheMarkProbeFixtureIsADocx` runs in `make test`.
- [x] MEASURED.md gets the sixteen answers by hand, dated, with the document
      id. `live/doc.go` names the test.
- [x] **If any hex came back changed, or the restyle cleared one, Part B
      stops here.** The gate came back split on 2026-10-14, document
      `139xi2N5QoSwTmSkkkzLM1M1Fgb7IK7u3JDITv7YCAy0`. All six highlight hexes
      came back hex for hex and survived the restyle, and Word's named yellow
      arrived as #FFFF00, a different colour from the mark yellow #FFD500, so
      the cover placeholders stay tellable from a mark. The pencil's #9A9A9A
      survived the import and did not survive the restyle: `textLook` writes
      `foregroundColor` from `styles.normal.color`, #000000, over every body
      run, so a pencil mark in a document somebody later restyles in place
      becomes black prose.
      Nothing is taken back out of Task 1's four files and no task is
      skipped, because six of seven marks pass the gate outright and the one
      answer that failed is about `restyle` rather than about the import.
      Whether that stops Part B, drops the pencil from the seven, or is a
      strain written into DECISIONS.md and accepted is Nail's decision and
      not this task's, and Tasks 8 to 11 wait for it. Written out in
      MEASURED.md, "Drive's import of run shading and text colour".
      Nail chose on 2026-10-15: the strain, written down and accepted. Part
      B runs with all seven marks; Task 8 writes the strain and its backlog
      item.
- [x] `git commit -m "test(live): measure Drive's import of run shading and text colour"`

### Task 4: character styles, a mechanism both parts use

**Files:**
- Modify: `go/internal/render/styles.go`, `go/internal/render/doc.go`,
  `go/internal/body/build.go`
- Test: `go/internal/render/charstyles_test.go`, `go/internal/body/runs_test.go`

- [x] `TestACharacterStyleIsWrittenWithItsRunProperties`: `styles.go` gains
      a writer for `w:type="character"` styles, taking an id, a name and the
      run properties it carries, with the rPr in schema order.
- [x] `TestARunMayNameItsCharacterStyle`: body's `runOpts` in `build.go`
      gains `Style`, `runOpts.empty()` counts it, and `addRuns` writes
      `w:rStyle` first in the rPr. Render's own run writer in `render/xml.go`
      is a different copy and is not touched.
- [x] `TestARunsPropertiesAreInSchemaOrder` is written here, beside the
      Quote and Table ones in `body_test.go`: `w:rStyle`, then the order
      `runProps` writes today through `w:u`, then `w:shd`, then
      `w:vertAlign`. The comment in `build.go` that says underline is last
      is corrected: it is last of today's properties, and shading and
      vertical alignment follow it in the schema.
- [x] `render/doc.go` names the mechanism and says the two parts use it.
- [x] `git commit -m "feat(render): character styles, and a run that names one"`

### Task 5: a footnote reference is a run, and the list renders

**Files:**
- Modify: `go/internal/body/inline.go`, `go/internal/body/build.go`,
  `go/internal/body/body.go`
- Test: `go/internal/body/footnote_test.go`

- [x] `TestAFootnoteReferenceCarriesItsNumber`: `Run` gains `Footnote int`,
      1-based, numbered in walk order across the note, one number per
      reference (decision 2.2 B). `east.FootnoteLink.Index` only selects
      which definition's text the note carries; it is never the number,
      because goldmark gives both references to one label the same Index.
      `appendInline` emits it; `FootnoteBacklink` stays skipped.
- [x] `TestALinkInsideAFootnoteIsWarnedAndPrintedAsItsWords`: `linkID`
      writes into `document.xml.rels`, which the footnotes part cannot
      reach, so a link in a footnote is its words, warned about by line,
      like a list in a footnote. A relationships part for footnotes is a
      later entry if citations need live links.
- [x] `TestAFootnoteReferenceInATableCellIsRefusedByLine`: the spec allows
      none, and `cellParagraph` has no footnotes part to write into.
- [x] `body/doc.go:117`, which lists footnotes among what is not rendered,
      is updated in the same commit.
- [x] `TestAFootnoteRunSurvivesMerge`.
- [x] `addRuns` writes the `FootnoteReference` style run with
      `w:footnoteReference w:id="N"`, before its own empty-text guard.
- [x] `TestAFootnoteBecomesAFootnoteTextParagraph`: the `*east.FootnoteList`
      arm renders each note's paragraphs into
      `Result.Footnotes []Footnote{ID int; Blocks []*etree.Element}`, each
      paragraph in the `FootnoteText` style opening with a `w:footnoteRef`
      run. `Counts` gains `Footnotes`.
- [x] `TestFootnotesAreNumberedInReferenceOrderNotDefinitionOrder`.
- [x] `TestAReferenceWithNoDefinitionIsRefusedByLine`,
      `TestADefinitionNobodyReferencesIsRefusedByLine`,
      `TestAFootnoteMarkInsideAHeadingIsRefusedByLine`: source-line checks
      in `Render` beside the markers check.
- [x] `TestAFootnoteReferencedTwiceIsTwoNotesWithOneText` (decision 2.2 B).
- [x] `TestAListInsideAFootnoteWarnsAndIsLeftOut`, and the same shape for a
      table, a picture, a code block and a heading.
- [x] `TestAHeadingInsideAFootnoteIsNoAnchor` and
      `TestAHeadingInsideAFootnoteSetsNoHeadingDepth` still pass;
      `TestAFootnoteWarnsAndIsLeftOut` and
      `TestEveryFootnoteIsNamedByItsOwnLine` are retired.
- [x] `git commit -m "feat(body): footnotes render, and ones that do not line up are refused"`

### Task 6: the footnotes part and the tenth house style

**Files:**
- Modify: `go/internal/render/render.go`, `go/internal/render/styles.go`,
  `go/internal/house/house.yaml`, `go/internal/house/house.go`
- Test: `go/internal/render/footnotes_test.go`, `go/internal/house/house_test.go`

- [x] `TestTheTenStylesAreThere`, renamed from `TestTheNineStylesAreThere`:
      `styles.footnote_text` at 10pt, single-spaced, 0 above and below, in
      the body face; `validateStyles` counts ten.
- [x] `TestTheFootnotesPartCarriesTheTwoSeparators`: `word/footnotes.xml` is
      always written, with the separator and continuation separator, then
      one `w:footnote` per note.
- [x] `TestTheFootnotesPartIsDeclaredAndRelated`: the content type and the
      relationship row; the footnotes relationship takes the next fixed id
      and `FirstMediaRelID` moves by one; the image goldens are regenerated
      and the diff read. The footnotes relationship took rId8 and
      `FirstMediaRelID` moved from 8 to 9. The four body goldens and the
      three render goldens were regenerated: every line of the body diff is
      one relationship id a picture or a link holds, raised by one, and the
      render diff is the content type row, the relationship row and the two
      new styles. The house file's first draft of `footnote_text` landed
      between `subtitle`'s values and its two `keep_` keys, which gave the
      notes Subtitle's keepNext and took it off Subtitle; the style is now
      after them and `TestFootnoteTextIsTenPointSingleSpacedAndTight` asks
      both keys are false.
- [x] `TestAFootnoteReferenceIsASuperscriptCharacterStyle`, through Task 4's
      writer, `w:vertAlign superscript`, a constant with its reason.
- [x] `TestBuildWritesTheThirteenPartsAndTheLogo`, renamed from twelve.
- [x] `git commit -m "feat(render): the footnotes part, and the two styles it needs"`

### Task 7: the footnote goldens, and the round trip carries one

**Files:**
- Modify: `go/internal/body/testdata/docs/01-kitchen-sink.md`,
  `go/internal/body/testdata/docs/05-edge-cases.md`, their goldens,
  `go/internal/export/project.go`, `go/internal/export/doc.go`,
  `skills/gdoc-export/markers.md`, `skills/gdoc-export/SKILL.md`,
  `go/internal/live/roundtrip_test.go`
- Create: `go/internal/body/testdata/golden/01-kitchen-sink.footnotes.xml`

- [x] Two footnotes in the kitchen sink, the second defined first; an edge
      case with a word label and one with a reference used twice.
      `TestTheSixDocumentsRenderToTheirGoldens` also compares
      `Result.Footnotes` against `<name>.footnotes.xml` where one exists.
      Both notes got one, so `05-edge-cases.footnotes.xml` was written
      beside the kitchen sink's: three notes there, 2 and 3 carrying one
      text. A note with no footnote must have no such file, so a footnote
      added to a fixture with no golden written for it fails rather than
      passing in silence.
- [x] `TestAFootnoteIsWrittenAndWarned` shrinks to what is still true, the
      suggestion-and-comment loss inside a footnote; `footnoteWarnings`
      loses its first sentence; the two skill pages follow. The test now
      also asserts no warning says "publish refuses" or counts the
      references, so the sentence cannot come back unnoticed.
- [x] `knownDifferences`'s `footnotes` entry keeps the `---` line on the
      `Added` side only and drops `Gone`. Its `Note` rule cannot be a
      per-line function, because a line does not know the order of the
      references in the whole note: `roundTrip` builds one closure over the
      note's text that numbers every `[^label]` reference in order of
      appearance, a label referenced twice taking two numbers, and `Added`
      excuses the second definition line that produces. The rule is tested
      offline on the edge-cases note before the live run.
      `numberFootnotes` is the closure and `footnoteNumbering` its two
      answers. `TestTheFootnoteRuleNumbersTheNotesTheWayTheDocumentWill`
      states every number over both notes as a literal, and
      `TestARoundTripOverAFootnoteLeavesNothingUnnamed` runs `roundTrip`
      itself over one label marked twice: nothing unnamed, four lines under
      the footnotes name. `TestLiveExportRoundTrip` is Task 13's and was
      not run here.
- [x] `git commit -m "feat(export): a footnote goes home and comes back"`

### Task 8: the palette in house.yaml

**Files:**
- Modify: `go/internal/house/house.yaml`, `go/internal/house/house.go`
- Test: `go/internal/house/marks_test.go`

- [ ] `TestTheSevenMarksAreStatedAsLiterals`: a `marks:` section keyed by
      Obsidian's colour names plus `pencil`, each with `syntax` (the emoji,
      `==` for the plain highlight, or `.pencil`), `kind` (highlight or
      text), `color` and `debt` (true for yellow, green and purple). Every
      emoji and hex a literal.
- [ ] `TestAMarkWithAnUnknownKindIsRefused`,
      `TestAMarkWhoseColourIsNotHexIsRefused`,
      `TestTwoMarksCannotShareASyntax`: `validateMarks` in `house.go`.
- [ ] `TestTheMarksAreTheSevenAndNoOther`.
- [ ] The DECISIONS.md entry "The marks: Obsidian's six colours and one
      grey" gains a paragraph under "What this strains", dated 2026-10-15:
      gate B measured that an in-place restyle writes `styles.normal.color`
      over every body run and so turns pencil black, Nail accepted it, and
      publish and export are unaffected. Its register row stays "holds".
- [ ] `docs/backlog/restyle-keeps-pencil.md`, in the shape of the other
      backlog files: `worth: yes`, `added: 2026-10-15`, what happens today
      with the MEASURED.md section named, and the fix, restyle splitting its
      `foregroundColor` request around runs whose colour is the pencil hex,
      which needs the run colours Task 11's read keeps.
- [ ] `git commit -m "feat(house): the seven marks, stated once"`

### Task 9: the colour emoji on a highlight, the pencil span, and what is refused

**Files:**
- Modify: `go/internal/body/mark.go`, `go/internal/body/inline.go`,
  `go/internal/body/body.go`
- Test: `go/internal/body/marks_test.go`

- [ ] `TestAHighlightOpeningWithAColourEmojiCarriesItsMark`: a `*Highlight`
      whose text opens with one of the six emoji sets `Run.Mark` to that
      colour's name and drops the emoji from the words; a plain highlight is
      `yellow`; `==🟡words==` is `yellow` too. The emoji-to-name map is read
      off `cfg.Marks`, Task 8's palette, because `Render` already takes
      `cfg` and `body/doc.go` says no emoji and no hex is a constant in body.
- [ ] `TestAMarkInATableCellIsCarried`: `cellParagraph` walks inlines
      through the same path, so `01-kitchen-sink.md:107`'s `==to fill in==`
      keeps working and becomes a yellow mark. The spec's "no mark in a
      table cell" sentence under "What this milestone does not do" is
      corrected to footnotes alone, in the same commit.
- [ ] `TestAnEmojiElsewhereInAHighlightIsText`,
      `TestAnEmojiOutsideAHighlightIsText`.
- [ ] `TestAPencilSpanIsARunCarryingItsMark`: a new inline parser in
      `mark.go` for `[words]{.pencil}`, producing a `*Span` node;
      `inline.go` folds it into `Run.Mark`. `Run.Highlight` is retired, with
      `build.go:273` following.
- [ ] `TestALinkIsStillALink`, `TestABracketNotFollowedByAttributesIsText`,
      `TestASpanWithAnotherAttributeIsRefusedByLineNamingPencil`.
- [ ] `TestAMarkMayCarryEmphasisAndCode`.
- [ ] `TestANestedMarkIsRefusedByLine`, `TestAnEmptySpanIsRefusedByLine`,
      `TestAMarkInAHeadingIsRefusedByLine`,
      `TestASpanCrossingAParagraphIsRefusedByLine`.
- [ ] `merge` joins neighbours only when their `Mark` is the same:
      `TestTwoMarksSideBySideStayTwoRuns`.
- [ ] `Counts` gains `Marks map[string]int` over the three debts, zero
      included: `TestTheDebtsAreCountedAndTheOthersAreNot`.
- [ ] `git commit -m "feat(body): Obsidian's colour emoji on a highlight, and the pencil span"`

### Task 10: the seven character styles and the shaded runs

**Files:**
- Modify: `go/internal/body/build.go`, `go/internal/render/styles.go`,
  the two test notes and their goldens
- Test: `go/internal/body/marks_test.go`, `go/internal/render/marks_test.go`

- [ ] `TestAMarkedRunCarriesItsStyleAndItsShading`: `addRuns` names the
      mark's style through Task 4, then writes `w:shd w:val="clear"
      w:color="auto" w:fill="<hex>"` for a highlight mark or `w:color` for
      pencil.
- [ ] `TestTheSevenCharacterStylesAreWritten`, each carrying the same
      shading or colour as its runs.
- [ ] `TestTheCoverPlaceholdersAreNotMarks`: the cover's yellow keeps
      `w:highlight yellow` and no `w:rStyle`.
- [ ] Goldens: `01-kitchen-sink.md` and `05-edge-cases.md` gain the seven
      marks; `go test ./internal/body -update`, and the diff read.
- [ ] `git commit -m "feat(render): a character style per mark, and the run shaded to match"`

### Task 11: the read keeps a run's colours, and export writes the mark back

**Files:**
- Modify: `go/internal/docs/walk.go`, `go/internal/docs/docs.go`,
  `go/internal/view/text.go`, `go/internal/export/project.go`,
  `skills/gdoc-export/markers.md`
- Test: `go/internal/docs/walk_test.go`, `go/internal/export/marks_test.go`

- [ ] `TestARunKeepsItsBackgroundAndTextColour`: `docs.Run` gains
      `Background` and `Colour`, hex strings, empty when the API states none.
- [ ] `TestExportWritesObsidiansSyntaxForEachPaletteColour`: under a
      `view.Options` flag export passes and read does not, a palette
      background becomes `==🟢words==` and so on, yellow as plain
      `==words==`, pencil's text colour as `[words]{.pencil}`.
- [ ] `TestAColourOutsideThePaletteIsPlainWordsAndCounted`: the count is
      `marks.other` on the export envelope, a map from hex to count.
- [ ] `TestAMarkSplitOverSeveralRunsComesHomeWhole`: Docs splits a mark that
      carries bold or code into runs sharing one background, and export
      joins adjacent runs of one palette colour into one mark.
- [ ] `stripDelimiters` in `live/roundtrip_test.go` stops removing `==`, and
      the `inline markup` entry's reason drops the highlight, because from
      here export writes the mark syntax back and both sides must agree.
- [ ] `TestReadPrintsNoMark`: `view.Text` without the flag is byte for byte
      what it was.
- [ ] `markers.md` gains the rows for the six highlights and the pencil span.
- [ ] `git commit -m "feat(export): a mark comes home as the syntax it left as"`

### Task 12: the envelopes, gdoc help marks, and the publish skill

**Files:**
- Modify: `go/cmd/gdoc/build.go`, `go/cmd/gdoc/publish.go`,
  `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/help.go`, `go/cmd/gdoc/doc.go`,
  `skills/gdoc-publish/SKILL.md`
- Test: `go/cmd/gdoc/marks_test.go`

- [ ] `TestBuildAndPublishCarryTheDebtCounts`: `body.marks` on both
      envelopes, the three debts, zero included; `body.footnotes` beside it.
- [ ] `TestHelpMarksPrintsTheSevenFromTheBinary`: `marks` is a help topic,
      not a command, so `cmdHelp` gains a `marks` branch before
      `helpMatches`, which would otherwise answer `unknownCommand`, and
      `completion` lists the word. It prints syntax, kind, colour and fate
      for each, from `house.Load`, in JSON under `--json` and as a table on
      a terminal.
- [ ] `TestARefusedNoteWritesNothing`: `gdoc build` on a note with a
      dangling footnote, and on one with a nested mark, refuses by line and
      leaves no file at `--out` (acceptance 2 and 4 at the command level).
- [ ] The publish skill's Step 1 runs `gdoc help marks` once and Step 2 reads
      `body.marks` from the dry run: a debt above zero is shown to the
      person with the count per colour, and the skill asks before
      publishing. The binary says nothing about whether to publish.
- [ ] `git commit -m "feat(cmd): the debts on the envelope, and help marks"`

### Task 13: the live acceptance

**Files:**
- Create: `go/internal/live/footnote_test.go`, `go/internal/live/marks_test.go`
- Modify: `go/internal/live/roundtrip_test.go`, `go/internal/live/doc.go`,
  `docs/v2/MEASURED.md`

- [ ] `TestLivePublishCarriesAFootnoteIntoDocs`: publishes the kitchen sink,
      reads it back bare, asserts a `footnoteReference` in the body and the
      note's text in the `footnotes` map, exports it and asserts
      `word/footnotes.xml` is in the zip, then trashes it.
- [ ] `TestLivePublishCarriesTheMarksIntoDocs`: the same publish, each
      marked run's background or colour as the palette states it.
- [ ] `TestLiveExportRoundTrip` over the kitchen sink passes; one of
      `roundTripNotes` carries both footnotes and marks.
- [ ] MEASURED.md gets the acceptance rows beside the probes'.
- [ ] `git commit -m "test(live): footnotes and marks survive publish, read-back and export"`

### Task 14: the house values Nail changes in this milestone

Nail said on 14 October that he will change style values in this milestone,
size and alignment among them. Each is one line in house.yaml and follows one
mechanism, so the master stays provenance and the gate stays honest.

**Files:**
- Modify: `go/internal/house/house.yaml`, `docs/v2/DECISIONS.md`,
  `go/internal/drift/compare.go` (Known), `go/internal/drift/pins_test.go`
  (knownOffline), the test that states the old value as a literal

- [ ] For each value Nail names: one DECISIONS.md line saying the value, the
      old one and why; the house.yaml line; the test that pinned the old
      literal re-stated with the new literal, never loosened; and, where the
      master states the old value, a `Known` entry with its reason and its
      pair re-recorded in `knownOffline`.
- [ ] A value the master does not state needs no Known entry and gets its
      own literal test instead.
- [ ] `TestTheOfflineGate` green with no row unexplained, and the live gate
      run once at the end of the milestone.
- [ ] `git commit -m "feat(house): <the value>, <old> to <new>"`, one commit
      per value.

### Task 15: the documents

**Files:**
- Modify: `docs/guide/` (the build and export pages), `README.md`,
  `CLAUDE.md`'s task map if a new file needs a row

- [ ] The guide's build page shows a footnote and the seven marks in the
      example note, says the marks are Obsidian's own syntax, and names the
      refusals. The export page says what comes home.
- [ ] `gdoc help build` names footnotes and marks in one line each and
      points at `help marks`.
- [ ] `git commit -m "docs: footnotes and marks in the guide and the help"`

### Task 16: verify acceptance, and M18 completed

- [ ] Every acceptance item in the spec, run and ticked in the task log,
      with the items of a part that stopped at its gate marked as such.
- [ ] `make test`, `make vet`, `make dist`.
- [ ] Both plan files move to `docs/plans/completed/`.
- [ ] `git commit -m "chore: M18 completed"`

## Post-Completion

Nail runs revmux (`ralphex --external-only`) until a clean round, merges, and
triggers the nightly. Left for later entries: suggestions inside footnotes
(its backlog item), marks in `read`'s projection, a mark on a whole paragraph,
and a meaning for any colour.
