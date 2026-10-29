# gdoc v2 Milestone 32e: the logo inside the column. Plan

Tasks: 3
Review: go

2026-10-29. One fix Nail chose with "ok" after the design reviewer read the
cover he sent: the house mark in the top right corner of page one ends in
line with the text instead of 1.4 cm past it in the margin, and the label "For
internal use only" sits level with the mark's middle instead of 6 pt above it.
Run by a ralphex worker on Claude, one task, one commit, on the local branch
`gdoc-v2-m32e-the-logo-inside-the-column`. `make release` runs without `GO`,
so the push and the PR wait for Nail's word.

## Principles

Serves: 3, because the house look is the one a reader trusts, and the first
page of every document opened with a mark that read as a slip. 4, because the
fix is three numbers and the words that explain them, and nothing is built
that nobody asked to see.

Strains: nothing written down. The drift gate's known list grows by one name,
with this decision as its reason, which is what the list is for.

## Overview

The decision entry of 2026-10-29 in `docs/v2/DECISIONS.md` ("The logo inside
the column") is the scope. The lead wrote it, the matching `docs/v2/SPEC.md`
sentence and the measurement in `docs/v2/MEASURED.md` ("The 29 October cover
in pixels") on main before this run. This plan touches none of the three.

What lands, in the order the tasks take it:

1. **The mark ends with the text, the label drops six points, and the drift
   gate knows it.** `offset_x_pt` 381.43, `offset_y_pt` -11.15, and
   `space_before_pt` 6 on the first paragraph of the first-page header;
   nothing else in the logo block or the header moves. The gate names the
   top offset against the master beside the width and the left offset it
   already names, with their pins. One commit, because the gate builds from
   the embedded house and goes red the moment the numbers move.
2. Acceptance.
3. Documents.

## Context (from discovery)

- The horizontal number. A4 is 595.28 pt wide with 51.05 pt margins, so the
  column is 493.18 pt (493.1755905511811 to the float) and the paper's right
  edge is 544.23 pt from the column's left edge. The mark stays 111.75 pt
  wide, so `offset_x_pt` 381.43 puts its right side at 493.18 pt, the
  column's right edge, and at 544.23 pt from the paper's left edge: the same
  51.05 pt margin the body keeps. The mark's left side moves 40.07 pt to the
  left. In EMU, as `emu()` in `go/internal/render/xml.go` rounds
  (`math.Round(pt × 12700)`): left offset 4844161. The width and the height
  do not move: cx 1419225 and cy 515828 stay.
- The vertical numbers. The anchor is `relative_v: paragraph`, so the mark is
  placed from the top of the first header paragraph, and that paragraph has
  `space_before_pt: 0` today. The label moves down and the mark does not by
  giving the paragraph six points before it and taking the same six off the
  mark's offset: `space_before_pt` 6 and `offset_y_pt` -11.15, because
  6 + (-11.15) = -5.15, the offset the mark has from the header margin
  today. In EMU the top offset is -141605; in twentieths of a point the
  space before is 120. The label's six points belong to the header's own
  paragraph in `house.yaml`, under `header: first: paragraphs:`, the first
  item, the one with the run "For internal use only " and `logo: true`.
  The second and third paragraphs of that header stay as they are.
- `go/internal/house/house.yaml`: the `header:` block starts at line 179 and
  its first paragraph's `space_before_pt: 0` is line 184; the `logo:` block
  is lines 1148 to 1160 with `offset_x_pt` on 1155 and `offset_y_pt` on
  1157, and a comment above `width_pt` on 1150 from 2026-10-28. The two new
  lines each take a comment of their own in the same voice, one line, naming
  the reason and "Nail, 2026-10-29". `go/internal/house/testdata/house-base64.yaml`
  is the same file with the placeholder's base64 in place of the Drive id,
  and `base64House` in `house_test.go` requires it to match `house.yaml`
  byte for byte up to `  base64: |-`, so every line that changes in the one
  changes in the other, comments included.
- `go/internal/house/house_test.go`: `TestTheLogoStatesItsMimeWidthAndPlacement`
  (line 250) states 421.5 and -5.15 as literals on lines 269 and 270.
  `TestTheMarkSitsInTheMarginAndOnThePaper` (line 282) asserts the opposite
  of this decision: that the right side is 533.25 pt and over the column's
  493.18. It is replaced, name and body, by a test that states the new truth
  as literals: offset + width is 493.18 within 0.005, and 51.05 + offset +
  width is 544.23 within 0.005, under A4's 595.28. A second new test over
  `Load()` states that the first header paragraph's `space_before_pt` is 6
  and the mark's `offset_y_pt` is -11.15, and that the two sum to -5.15,
  which is the sentence "the label moves and the mark does not" as a number.
  `go/internal/house/house.go` names `TestTheMarkSitsInTheMarginAndOnThePaper`
  in its package comment (line 10) and takes the new names instead.
- `go/internal/render/headfoot_test.go`:
  `TestTheFirstPageHeaderCarriesTheLogoWhereTheHouseAnchorsIt` (line 10)
  wants posOffset 5353050 on positionH (line 40) and -65405 on positionV
  (line 47), with the point values in its messages. It wants 4844161 and
  -141605 after this run, and it grows one assertion: the first `w:p` of
  the header carries `w:spacing` with `w:before="120"`, and the third still
  carries `w:before="0"`. `headfoot.go` line 48 already writes
  `SpaceBeforePt` into the paragraph's spacing, so no code under `render`
  changes. `go/internal/render/testdata/header2.xml` carries 5353050 once,
  -65405 once and `w:before="0"` twice (the first and the third paragraph);
  `TestTheGoldenPartsAreWhatTheHouseStyleRendersTo` (`render_test.go` line
  470) reads it through `house.Load()`, so the golden moves to the new
  numbers in the same commit, the first `w:before` to 120 and the second
  left at 0. `TestTheLogoHeightFollowsThePicturesRatio` (line 58) reads
  only cx and cy and stays.
- The drift gate. `go/internal/drift/items.go` lines 158 and 159 build
  `logo leftOffset (pt)` and `logo topOffset (pt)` with `offsetTol` of two
  points, which covers neither a move of 40 pt nor one of 6 pt. `Known` in
  `go/internal/drift/compare.go` starts at line 99 and names the width, the
  left offset and the height at lines 107 to 109 under a comment block that
  says "111.75 pt at 421.5 pt". After this run the left offset's reason is
  Nail's decision of 2026-10-29, the mark ends in line with the text, 381.43
  pt, where the master's frame starts at 404.25 pt; `logo topOffset (pt)`
  joins with the reason that the mark's offset from its paragraph is -11.15
  pt so that the label can sit six points lower and the mark stay, where
  the master's docx reads -5.149 pt (its posOffset is -65398 EMU, which
  `emuPt` rounds to three places; that is why the row read CLOSE until
  now); and the comment block says so. `Known` holds
  thirty names today and thirty-one after; `doc.go` lines 81 and 114 and
  the `compare_test.go` comment at line 177 say "thirty", and "fifteen of"
  is counted again.
- The pins live in `knownOffline` in `pins_test.go` (lines 29 to 32 for the
  logo), one pair per Known name, printed by `show` with `%v` after `emuPt`
  rounds to three places: `logo leftOffset (pt)` becomes
  {`381.43`, `404.25`}, and `logo topOffset (pt)` {`-11.15`, `-5.149`} is
  added, the master's side as `show` prints it and not as the house states
  it. `buildNote` in `drift_test.go` builds from the embedded house, so
  `TestTheOfflineGate` goes red the moment `house.yaml` moves, and
  `TestEveryKnownDifferenceStillDiffers` goes red if a Known name is added
  before the numbers move: that is why the numbers and the gate are one
  commit.
- `TestUnexplainedIsWhatAGateFailsOn` (`compare_test.go` line 95) carries
  `{Item: "logo topOffset (pt)", Verdict: Close}` at line 98 among the rows
  that must not come back unexplained. A Close row is explained with or
  without Known, so the assertion holds, but the row's comment, if any,
  should not say the name is outside Known once it is in.
- The live gate is not only Known. `acceptance()` in
  `go/internal/live/publish_test.go` calls `held("logo topOffset (pt)")` at
  line 508, and `held` wants IDENTICAL or CLOSE against the master whatever
  Known says; -11.15 against -5.15 is 6 pt apart and would be red. That
  call goes, and the top offset joins the literal loop at lines 519 to 523
  as {"logo topOffset (pt)", -11.15}, where `logo leftOffset (pt)` becomes
  381.43. The comment above the loop (lines 511 to 518) says why. This file
  is not run in this task (the live variables are unset); it compiles under
  `go vet ./...`.
- `docs/guide/publishing.md` lines 37 to 40 say the house draws the mark
  "3.9 cm wide in the top right corner of page one, reaching 4 mm from the
  paper's edge". The last clause is false after this run.
- Files that name 421.5, -5.15 or 404.25 and stay as they are: the Docs
  answer fixture in `drift/docsapi_test.go`, the master assertions in
  `drift/drift_test.go` (lines 110 to 114), the synthetic `logoBlock` in
  `house_test.go` (line 688 carries -5.15) for the refusal tests, the unit
  test `emu(-5.15)` is -65405 in `render/xml_test.go` lines 24 to 25, and
  the master's values inside the pins and the Known reasons. `grep -rn
  "421.5\|-5.15\|5353050\|65405" go/ docs/guide/` after the run names only
  those, plus the Known reasons and the comments that quote the old numbers
  as history.
- Nothing here touches `go/internal/guard`, `go/internal/auth`,
  `go/internal/presence`, `go/cmd/gdoc/desktop.go` or an invariant in
  `CLAUDE.md`, so no guardian pass before the run.

## Development Approach

- TDD. Each task writes its failing tests first and runs them red.
- One task, one commit, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`
  green before the commit, with `GDOC_LIVE_TEST` and `GDOC_LIVE_WRITE` unset.
- A house-style test states its value as a literal and never reads the
  constant it checks.
- Every `doc.go` or package comment rule this run changes names its test.
- Plain, short English in comments and documents. No em dashes.

## Testing Strategy

Unit tests in `internal/house`, `internal/render` and `internal/drift`, and
one `held` line in the live acceptance replaced by a literal. No live run:
the change reaches no Google call. The acceptance task builds the example
note offline through `--house` with the base64 test house, which `logoFor` in
`go/cmd/gdoc/logo.go` serves from the file with no policy and no network, and
reads the numbers out of the docx, which is the whole of what Docs will be
handed. How Docs draws the header after the import is the lead's check by
eye after the release, written in the decision entry, and not this run's.
The nightly live run reads the top offset back from Docs against the
literal -11.15 at 02:00 UTC; -5.15 survived the import to three places, so
-11.15 is expected to, and a red nightly is the first word if it does not.

## Solution Overview

Three numbers in the house file move. The render already draws the frame at
the anchor's offsets and writes a header paragraph's space before, so no code
under `render` changes; its tests and golden state the new numbers. The drift
gate learns, in the same commit, that the top offset differs from the master
on purpose and that the left offset differs for a new reason, with the pins
that stop them moving again unnoticed.

## What Goes Where

| Piece | Code | Tests | Words |
|---|---|---|---|
| The three numbers | `go/internal/house/house.yaml`, `testdata/house-base64.yaml` | `house_test.go`, `render/headfoot_test.go`, `render/testdata/header2.xml` | `house.go` package comment |
| The gate | `go/internal/drift/compare.go` | `pins_test.go`, `compare_test.go`, `live/publish_test.go` | `drift/doc.go` |
| Guide | | | `docs/guide/publishing.md` |

## Implementation Steps

### Task 1: the mark ends with the text, the label drops six points, and the gate that knows it

**Files:** `go/internal/house/house.yaml`,
`go/internal/house/testdata/house-base64.yaml`,
`go/internal/house/house_test.go`, `go/internal/house/house.go`,
`go/internal/render/headfoot_test.go`, `go/internal/render/testdata/header2.xml`,
`go/internal/drift/compare.go`, `go/internal/drift/pins_test.go`,
`go/internal/drift/compare_test.go`, `go/internal/drift/doc.go`,
`go/internal/live/publish_test.go`

- [x] test first: `TestTheLogoStatesItsMimeWidthAndPlacement` wants
      `width_pt` still 111.75, `offset_x_pt` 381.43 and `offset_y_pt` -11.15,
      the placeholder still 509 by 185
- [x] test: `TestTheMarkEndsInLineWithTheText` replaces
      `TestTheMarkSitsInTheMarginAndOnThePaper`, over `house.Load()`. It reads
      `width_pt` and `offset_x_pt` from the config and states every other
      number as a literal: offset + width is 493.18 within 0.005, the
      column's right edge, and 51.05 + offset + width is 544.23 within 0.005,
      under A4's 595.2755905511811 (the mark is on the paper, with the body's
      margin to its right)
- [x] test: `TestTheLabelDropsSixPointsAndTheMarkStays`, over `house.Load()`:
      the first paragraph of `header.first` carries `space_before_pt` 6, the
      logo anchor's `offset_y_pt` is -11.15, and 6 + (-11.15) is -5.15 within
      0.005, the mark's offset from the header margin before this run
- [x] test: `TestTheFirstPageHeaderCarriesTheLogoWhereTheHouseAnchorsIt`
      wants posOffset 4844161 on positionH and -141605 on positionV, with
      381.43pt and -11.15pt in the messages, cx 1419225 and cy 515828 as
      they are, and one new assertion: the header's first `w:p` carries
      `w:spacing` with `w:before="120"`, and its third carries `w:before="0"`
- [x] test: the pins in `knownOffline` as literals, `logo leftOffset (pt)`
      {`381.43`, `404.25`} and a new `logo topOffset (pt)` {`-11.15`, `-5.149`};
      the width and the height pairs stay
- [x] test: `acceptance()` in `live/publish_test.go` drops
      `held("logo topOffset (pt)")`, and the literal loop reads
      `logo leftOffset (pt)` 381.43 and gains `logo topOffset (pt)` -11.15,
      with the comment above it naming this decision beside the one of
      2026-10-28
- [x] run the house, render and drift tests red
- [x] `house.yaml`: `offset_x_pt: 381.43` with a one-line comment above it
      (the mark ends in line with the text, 493.18 less 111.75; Nail,
      2026-10-29), `offset_y_pt: -11.15` with a one-line comment (six points
      up to meet the six the label's paragraph gained; Nail, 2026-10-29), and
      `space_before_pt: 6` on the first paragraph of `header.first` with a
      one-line comment (the label sits level with the mark's middle; Nail,
      2026-10-29)
- [x] `testdata/house-base64.yaml`: the same lines, byte for byte
- [x] `testdata/header2.xml`: 5353050 to 4844161, -65405 to -141605, the
      first `w:before="0"` to `w:before="120"` and the second left alone, and
      the commit says the diff is those numbers
- [x] `compare.go`: the comment block above the logo names and the reasons
      for `logo leftOffset (pt)` and the new `logo topOffset (pt)` say what
      the Context states; the width and the height reasons stay
- [x] `doc.go` lines 81 and 114 and the `compare_test.go` comment at 177 say
      thirty-one; the worker counts the "fifteen of" again, there and at
      `compare.go` line 334, and the commit says the count
- [x] `house.go`'s package comment names `TestTheMarkEndsInLineWithTheText`
      and `TestTheLabelDropsSixPointsAndTheMarkStays` where it named
      `TestTheMarkSitsInTheMarginAndOnThePaper`
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test` and `make vet`
      green, one commit

### Task 2: Verify acceptance criteria

- [ ] `make test` and `make vet` green, live variables unset
- [ ] `make build`, then offline: `bin/gdoc build --md
      release/example/first-note.md --out <tmp>/a.docx --house
      go/internal/house/testdata/house-base64.yaml`; then
      `unzip -p <tmp>/a.docx word/header2.xml` holds `cx="1419225"`,
      `<wp:posOffset>4844161</wp:posOffset>`,
      `<wp:posOffset>-141605</wp:posOffset>` and `w:before="120"`, and
      `unzip -p <tmp>/a.docx word/header2.xml | grep -c "5353050\|65405"`
      is 0. The logo's PNG bytes travel in the full stream, so a grep over
      the whole package needs `grep -a` or a C locale; the one part is text
- [ ] the arithmetic in the commit message: 381.43 + 111.75 = 493.18, the
      column; 51.05 + 493.18 = 544.23 under 595.28; 6 + (-11.15) = -5.15
- [ ] `grep -rn "421.5\|5353050\|65405" go/ docs/guide/` names only the
      files the Context lists as staying: the Known reasons and comments
      that quote the old numbers as history, the synthetic fixtures, and
      the `emu(-5.15)` unit test in `render/xml_test.go`. Any other hit is a
      miss to fix

### Task 3: [Final] Update documentation

- [ ] `docs/guide/publishing.md`, the paragraph at line 37: the mark is
      drawn 3.9 cm wide in the top right corner of page one, its right side
      in line with the text, and the label before it sits level with it
- [ ] do not edit `CLAUDE.md`, `docs/v2/SPEC.md`, `docs/v2/DECISIONS.md` or
      `docs/v2/MEASURED.md`
- [ ] move this plan to `docs/plans/completed/`

## Post-Completion

- The lead runs the suites by hand, then `make release
  BRANCH=gdoc-v2-m32e-the-logo-inside-the-column` without `GO`. M32d stands
  at the same step, so the lead takes the two to Nail as one go.
- After the release and Nail's `gdoc update`, the lead publishes today's
  chapter again from its note, measures the published page the way
  MEASURED.md "The 29 October cover in pixels" did, and tells Nail where the
  new document is and whether the label sits level. A label that does not is
  a second pass with two numbers and its own decision line, not a blocker.
- The week's stopwatch row for this run goes into `docs/team/measure.md`.
