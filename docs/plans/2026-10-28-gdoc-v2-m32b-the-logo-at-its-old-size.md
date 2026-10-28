# gdoc v2 Milestone 32b: the logo at its old size. Plan

Tasks: 3
Review: go

2026-10-28. One fix Nail chose with "ok, run with the recommendation": the
house mark in the top right corner of page one is drawn 3.9 cm wide where the
firm's template put it, instead of the 5.2 cm mark cut at the paper's edge that
every document since 26 October carries. Run by a ralphex worker on Claude, one
task, one commit, on the local branch `gdoc-v2-m32b-the-logo-at-its-old-size`.
`make release` runs without `GO`, so the push and the PR wait for Nail's word.

## Principles

Serves: 3, because a document in the firm's name carries the firm's mark whole,
and a mark Docs cuts at the page's edge is the wrong answer from a tool that
promises the house look. 4, because the fix is two numbers and the words that
explain them, and nothing is built that nobody asked to see.

Strains: nothing written down. The drift gate's known list grows by two names,
each with this decision as its reason, which is what the list is for.

## Overview

The decision entry of 2026-10-28 in `docs/v2/DECISIONS.md` ("The logo at its
old size") is the scope. The lead wrote it, the matching `docs/v2/SPEC.md`
sentence and the measurement in `docs/v2/MEASURED.md` ("The logo's frame and
its ink") on main before this run. This plan touches none of the three.

What lands, in the order the tasks take it:

1. **The house states the mark's own width and place, and the drift gate
   knows it.** `width_pt` 111.75, `offset_x_pt` 421.5, nothing else in the
   logo block moves; the gate names the width and the left offset against the
   master beside the height it already names, with their pins. One commit,
   because the gate builds from the embedded house and goes red the moment
   the numbers move.
2. Acceptance.
3. Documents.

## Context (from discovery)

- The numbers. The old picture, 195 by 97 px, carried transparent padding: its
  ink ran from column 23 to column 171, so at 146.25 pt wide the mark itself
  was 111.75 pt (3.9 cm) and began 17.25 pt into the frame. The picture in
  Drive since 2026-10-26, 509 by 185 px, has no padding. So `width_pt` 111.75
  at `offset_x_pt` 404.25 + 17.25 = 421.5 draws the new picture's ink exactly
  where the old picture's ink was. The height follows the ratio:
  111.75 × 185 ÷ 509 = 40.6164 pt. In EMU, as `emu()` in
  `go/internal/render/xml.go` rounds (`math.Round(pt × 12700)`): width
  1419225, left offset 5353050, height 515828. The plan reviewer ran the
  three through `emu()` and confirmed them.
- The page. A4 is 595.28 pt wide with 51.05 pt margins, so the column is
  493.18 pt and the paper's right edge is 544.23 pt from the column's left
  edge. The mark's right side at 421.5 + 111.75 = 533.25 pt is 40 pt past the
  text and 11 pt short of the paper: that is the firm's template look, and
  Nail chose it over a mark ending in line with the text.
- `go/internal/house/house.yaml` lines 1148 to 1158 hold the `logo:` block,
  `width_pt` on 1150 and `offset_x_pt` on 1154. The file has no YAML comment
  today and says its reasons in `note:` keys; goccy reads a comment fine, so
  a comment above `width_pt` is allowed, but a `note:` key is the file's own
  way and the worker uses it if the schema admits one under `logo`, else the
  comment. `go/internal/house/testdata/house-base64.yaml` is the same file
  with the placeholder's base64 in place of the Drive id, and `base64House`
  in `house_test.go` (lines 872 to 898) requires it to match `house.yaml`
  byte for byte up to `  base64: |-`, so every line that changes in the one
  changes in the other, the comment or note included.
  `TestTheLogoStatesItsMimeWidthAndPlacement` (house_test.go line 250) states
  146.25 and 404.25 as literals. `go/internal/house/house.go` names it in the
  package comment (line 9).
- Two render tests pin the old numbers against the embedded house.
  `TestTheFirstPageHeaderCarriesTheLogoWhereTheHouseAnchorsIt`
  (`go/internal/render/headfoot_test.go` line 10) wants 1857375, 675077 and
  5133975 EMU, with the point values repeated in its messages (lines 31, 34,
  41). `TestTheLogoHeightFollowsThePicturesRatio` (line 58) builds with a 400
  by 100 PNG and wants cx 1857375 and cy 464344 on both `//wp:extent` and
  `//a:ext`; at 111.75 pt wide a 4:1 picture is 27.9375 pt high, so cx
  1419225 and cy 354806. `go/internal/render/testdata/header2.xml` carries
  1857375 twice, 675077 twice and 5133975 once; the worker finds which test
  reads it and brings the golden to the new numbers in the same commit.
  `emu(146.25)` in `go/internal/render/xml_test.go` tests the unit and stays.
  `render/doc.go` lines 19 to 24 name the two tests.
- The drift gate. `go/internal/drift/items.go` lines 156 to 159 build the
  rows `logo width (pt)`, `logo height (pt)`, `logo leftOffset (pt)` and
  `logo topOffset (pt)`; `leftOffset` carries `offsetTol`, which does not
  cover a move of 17.25 pt. `Known` in `go/internal/drift/compare.go` starts
  at line 99 and names `logo height (pt)` at lines 100 to 103 with the
  2026-10-26 reason, whose comment says "draws it 146.25 pt wide". `buildNote`
  in `drift_test.go` (line 301) builds from the embedded house, so
  `TestTheOfflineGate` (line 189) goes red the moment `house.yaml` moves,
  and `TestEveryKnownDifferenceStillDiffers` (line 225) goes red if a Known
  name is added before the numbers move: that is why the numbers and the
  gate are one commit. The pins live in `knownOffline` in `pins_test.go`
  (lines 29 to 58), one pair per Known name, printed by `show` with `%v`
  after `emuPt` rounds to three places (`docx.go` line 872), so the pairs
  are `logo width (pt)` 111.75 against 146.25, `logo leftOffset (pt)` 421.5
  against 404.25, and `logo height (pt)` 40.616 against 72.75 (today
  53.156, pins_test.go line 30). `Known` holds 28 names today and 30 after
  this run; the "twenty-five" at `doc.go` lines 81 and 114 and in the
  `compare_test.go` comment at line 174 has been stale since the height and
  the two spaceBelow entries were added.
- `TestUnexplainedIsWhatAGateFailsOn` (`compare_test.go` line 95) uses
  `{Item: "logo width (pt)", Verdict: Missing}` as a row that must come back
  unexplained. Once that name is in Known the test gets one row and fails at
  its `t.Fatalf`. The fixture moves to a name that is not in Known.
- The live gate is not only Known. `acceptance()` in
  `go/internal/live/publish_test.go` lines 505 to 510 calls `held(...)` for
  `logo width (pt)`, `logo height (pt)` and `logo leftOffset (pt)`, and
  `held` wants IDENTICAL or CLOSE against the master whatever Known says. The
  height has been red there since 2026-10-26; after this run the other two
  would be too. The three calls go, and the built side's values are stated
  as literals in their place.
- `docs/guide/publishing.md` lines 37 to 40 describe the `logo` entry and say
  the height follows the picture's ratio; nothing in the guides states the
  mark's width or where it sits.
- `docs/backlog/the-logo-drive-file-cannot-be-changed.md` line 25, "A logo of
  any size now fits by the width stated in `house.yaml`", stays true and stays.
- Files that name 146.25 or 404.25 and stay as they are: the synthetic
  `logoBlock` in `house_test.go` (lines 647, 651, 664, 764) for the refusal
  tests, the Docs answer fixture in `drift/docsapi_test.go` (lines 74 to 77
  and 179 to 180), the `emuPt` unit test in `drift/docx_test.go` (lines 87 to
  88), the assertions about the master in `drift/drift_test.go` (lines 110 to
  114), and the master's values inside the drift pins and reasons.
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
the three `held` lines in the live acceptance replaced by literals. No live
run: the change reaches no Google call. The acceptance task builds the example
note offline through `--house` with the base64 test house, which `logoFor` in
`go/cmd/gdoc/logo.go` serves from the file with no policy and no network, and
reads the numbers out of the docx, which is the whole of what Docs will be
handed.

## Solution Overview

Two numbers in the house file move so that the frame is the mark. The render
already draws the frame `width_pt` wide at the anchor's offsets with the
height following the picture, so no code under `render` changes; its tests
and golden state the new numbers. The drift gate learns, in the same commit,
that the width and the left offset differ from the master on purpose, with the
pins that stop them moving again unnoticed.

## What Goes Where

| Piece | Code | Tests | Words |
|---|---|---|---|
| The two numbers | `go/internal/house/house.yaml`, `testdata/house-base64.yaml` | `house_test.go`, `render/headfoot_test.go`, `render/testdata/header2.xml` | `house.go` package comment |
| The gate | `go/internal/drift/compare.go` | `pins_test.go`, `compare_test.go`, `live/publish_test.go` | `drift/doc.go` |
| Guide | | | `docs/guide/publishing.md`, `render/doc.go` |

## Implementation Steps

### Task 1: the mark's own width and place, and the gate that knows it

**Files:** `go/internal/house/house.yaml`,
`go/internal/house/testdata/house-base64.yaml`,
`go/internal/house/house_test.go`, `go/internal/house/house.go`,
`go/internal/render/headfoot_test.go`, `go/internal/render/testdata/header2.xml`,
`go/internal/drift/compare.go`, `go/internal/drift/pins_test.go`,
`go/internal/drift/compare_test.go`, `go/internal/drift/doc.go`,
`go/internal/live/publish_test.go`

- [x] test first: `TestTheLogoStatesItsMimeWidthAndPlacement` wants
      `width_pt` 111.75 and `offset_x_pt` 421.5, `offset_y_pt` still -5.15,
      the placeholder still 509 by 185
- [x] test: `TestTheMarkSitsInTheMarginAndOnThePaper` over `house.Load()`,
      the house that ships. It reads `width_pt` and `offset_x_pt` from the
      config and states every other number as a literal: left margin 51.05 +
      offset + width = 584.3, under the page's 595.28 (the mark is on the
      paper), and offset + width = 533.25, over the column's 493.18 (the mark
      reaches past the text into the margin, as the firm's template drew it)
- [x] test: `TestTheFirstPageHeaderCarriesTheLogoWhereTheHouseAnchorsIt`
      wants cx 1419225, cy 515828 and posOffset 5353050, the top offset
      -65405 and everything else unchanged, and its messages say 111.75pt,
      40.616pt and 421.5pt
- [x] test: `TestTheLogoHeightFollowsThePicturesRatio` wants cx 1419225 and
      cy 354806 on both elements
- [x] test: the three pins in `knownOffline` as literals, `logo width (pt)`
      111.75 against 146.25, `logo leftOffset (pt)` 421.5 against 404.25,
      `logo height (pt)` 40.616 against 72.75
- [x] test: `TestUnexplainedIsWhatAGateFailsOn`'s `Missing` fixture names a
      row that is not in Known, `logo layout`, and its assertions follow; a
      comment says why the row moved
- [x] test: `acceptance()` in `live/publish_test.go` drops the three `held`
      calls for the logo width, height and left offset and states the built
      side's values as literals instead, 111.75, 40.616 and 421.5, with a
      comment naming this decision. This file is not run in this task (the
      live variables are unset); it compiles under `go vet ./...`
- [x] run the house, render and drift tests red
- [x] `house.yaml`: `width_pt: 111.75`, `offset_x_pt: 421.5`, and one line
      saying the width is the mark's own, measured on the firm's template as
      the ink of the old picture, decided 2026-10-28, as a `note:` key if the
      logo block admits one, else a comment
- [x] `testdata/house-base64.yaml`: the same lines, byte for byte
- [x] `testdata/header2.xml`: 1857375 to 1419225, 675077 to 515828, 5133975
      to 5353050, and the commit says the diff is those numbers
- [x] `compare.go`: `Known` gains `logo width (pt)` and
      `logo leftOffset (pt)`, each with the reason: Nail's decision of
      2026-10-28, the mark is drawn at its own width, 111.75 pt at 421.5 pt,
      where the master's padded picture put its ink; the master's frame is
      146.25 pt at 404.25 pt. The `logo height (pt)` reason says 40.6 pt at
      the 2026-10-26 picture's ratio, and the comment above it stops saying
      146.25 pt
- [x] `doc.go` lines 81 and 114 and the `compare_test.go` comment at 174 say
      thirty; the worker counts the "ten of" again and the commit says the
      count
- [x] `house.go`'s package comment names
      `TestTheMarkSitsInTheMarginAndOnThePaper` beside
      `TestTheLogoStatesItsMimeWidthAndPlacement`
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test` and `make vet`
      green, one commit

### Task 2: Verify acceptance criteria

- [ ] `make test` and `make vet` green, live variables unset
- [ ] `make build`, then offline: `bin/gdoc build --md
      release/example/first-note.md --out <tmp>/a.docx --house
      go/internal/house/testdata/house-base64.yaml`; then
      `unzip -p <tmp>/a.docx word/header2.xml` holds `cx="1419225"`,
      `cy="515828"` and `<wp:posOffset>5353050</wp:posOffset>`, and
      `unzip -p <tmp>/a.docx | grep -c "1857375\|5133975"` is 0 (the parts
      are compressed, so the raw zip is never grepped)
- [ ] the arithmetic in the commit message: 51.05 + 421.5 + 111.75 = 584.3
      under 595.28
- [ ] `grep -rn "146.25\|404.25" go/ docs/guide/` names only the files the
      Context lists as staying: `xml_test.go`, the `logoBlock` fixture in
      `house_test.go`, `docsapi_test.go`, `docx_test.go`, the master
      assertions in `drift_test.go`, and the master's values in the pins and
      the Known reasons. Any other hit is a miss to fix

### Task 3: [Final] Update documentation

- [ ] `docs/guide/publishing.md`, the paragraph at line 37: one sentence that
      `width_pt` is the width of the mark itself, which the house draws 3.9 cm
      wide in the top right corner of page one, reaching 4 mm from the paper's
      edge
- [ ] `go/internal/render/doc.go` lines 19 to 24: where it says the house
      states the picture's width, it says the mark's width
- [ ] do not edit `CLAUDE.md`, `docs/v2/SPEC.md`, `docs/v2/DECISIONS.md` or
      `docs/v2/MEASURED.md`
- [ ] move this plan to `docs/plans/completed/`

## Post-Completion

- The lead runs the suites by hand, then `make release
  BRANCH=gdoc-v2-m32b-the-logo-at-its-old-size` without `GO`, and asks Nail
  for the go.
- After the release and Nail's `gdoc update`, the lead publishes the two
  documents again that carry the big mark: "Embedded card: legal actions and
  Jira tickets" and "Request to PCI DSS auditors, corporate cards for the
  company's own use", each from its note in the hub, and tells Nail where
  the new ones are.
- The week's stopwatch row for this run goes into `docs/team/measure.md`.
