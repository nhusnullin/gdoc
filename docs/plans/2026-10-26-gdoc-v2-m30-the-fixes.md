# gdoc v2 Milestone 30: the fixes. Plan

Tasks: 9
Review: go

2026-10-26. Seven fixes Nail chose on 2026-10-26, run overnight by a ralphex
worker on codex. One task, one commit, on the local branch
`gdoc-v2-m30-the-fixes`. `make release` runs without `GO`, so the push and the
PR wait for Nail's word, and nothing is merged or tagged without it.

## Principles

Serves: 3, because the guard refuses one more spelling of a reset, a footer
logo is refused by name, and export stops deleting characters inside code. 4,
because a Russian reader's links work and a heading is no longer glued to the
text under it.

Strains: 3, in one place. The logo stops being held to exact pixels, so a PNG
of the wrong picture but the right type now prints where a PNG of another size
used to be refused. That is accepted: the mime check, the byte ceiling and a
side ceiling still refuse a file gdoc half understands, and the file is the
firm's own, in the firm's Drive, named by id in the house file.

## Overview

The decision entry of 2026-10-26 in `docs/v2/DECISIONS.md` ("The fixes: a cover
line and the footer keep their own look, ...") is the scope. The lead wrote it,
the matching `docs/v2/SPEC.md` changes and the MEASURED.md section on main
before this run. This plan touches none of those three files.

The seven fixes, in the order the tasks take them:

1. **The guard refuses a `null` leaf in an in-place style mask.** A styling
   request whose `fields` mask names a field that the style object holds as
   `null` is refused by name. Today it is carried, and it resets that property.
2. **Render honours a block's own look.** A cover line's `align`, `italic`,
   `color` and `line_spacing`, a footer paragraph's `align` and `rule`, and a
   label's `keep_with_next` reach the docx. Today render drops them. A logo in
   a footer is refused by name.
3. **A logo of any size fits.** `px` and `height_pt` leave `house.yaml`. The
   picture is drawn `width_pt` wide and its height follows its own ratio. The
   mime and the byte ceiling still refuse a wrong file.
4. **Six points after the two top headings.** `heading_1` and `heading_2` get
   `space_after_pt: 6`, as `heading_3` and `heading_4` have.
5. **Export fix 13.** The heading number strip never touches a line inside a
   fenced code block.
6. **Russian heading ids.** `## Цели проекта` gets the id `цели-проекта`, the
   GitHub way, and a repeated heading takes `-1`, `-2`. An id that worked
   before today still resolves. The docx bookmark stays ASCII: `_h_` and a
   short hash.
7. **Export fix 14.** A link to the second of two headings with one id points
   at the second, with GitHub's `-1` suffix.

Fixes 5, 6 and 7 touch the same heading id code in `go/internal/view` and
`go/internal/export`. They are ordered so each builds on the last: Task 5 moves
the strip onto goldmark's own heading lines, Task 6 changes the id rule those
lines carry, and Task 7 makes the links follow the ids by position.

## Context (from discovery)

- **The lead's work on main before the run.** The decision entry and its
  register row, the MEASURED.md section "Drive's import of a Russian heading
  and a link to it", and these SPEC.md sections: "`export`" (the paragraph
  "One file per tab": the heading id rule, the fence, repeated headings), "The
  generator and `house.yaml`" (the logo paragraph, and the two new paragraphs
  "Every value a block states reaches the page" and "A heading id keeps the
  heading's letters"), and "Acceptance for v2.0" item 1 (the `null` mask). The
  worker edits none of `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md` and
  `docs/v2/MEASURED.md`.
- **The trust guardian.** Task 1 touches `go/internal/guard`, so the lead
  calls `gdoc-trust-guardian` over this plan before the run. Every guard
  invariant in `CLAUDE.md` keeps holding, and no assertion about what the guard
  refuses is loosened. `AllowFile` accepting any level is not changed: that is
  Nail's explicit call.
- **The null leaf.** `checkInPlaceMask` and `checkMaskIsSet` in
  `go/internal/guard/policy.go` (around line 1500) judge a styling request's
  mask. `walkMaskPath` follows each mask path into the style object and
  refuses a missing segment. A segment that is present with the JSON value
  `null` passes, because `obj[seg]` exists. Google reads a `null` field as
  unset, and a field named in the mask and left unset is reset to its default.
  So `{"textStyle":{"bold":null},"fields":"bold"}` is carried today and clears
  bold. A `null` parent (`{"foregroundColor":null}` with the path
  `foregroundColor.color`) is already refused, because `json.Unmarshal` of
  `null` into a map gives a nil map and the next segment is missing.
  `go/internal/guard/doc.go`, "A third write level, and the four kinds it
  carries", holds the mask rules and names `TestAMaskMayNameOnlyWhatTheRequestSets`.
  No builder in `go/internal/restyle` writes a `null`.
- **What render drops.** `go/internal/render/front.go`, `cover()`: every cover
  line paragraph takes `c.Align` and never `line.Align`; a line with words
  never passes `line.LineSpacing` (only the blank branch does); `line.Italic`
  is never read; `line.Color` is never read, and only a run's own `r.Color`
  reaches a run. `label()` never passes `l.KeepWithNext`, while
  `paraOpts.KeepNext` exists. `go/internal/render/headfoot.go`,
  `footerPart()`: no alignment at all, neither `spec.Align` nor
  `region.Align`, and `spec.Rule` and `spec.Logo` are ignored. The header path,
  `regionParagraph()`, does all three and refuses a logo outside the first
  page's header with `b.fail`. That refusal has no test today.
  `go/internal/cover` holds the note's fields and no style, so it drops
  nothing and is not touched.
- The embedded `house.yaml` states none of these fields today on a cover line,
  a footer paragraph or a label, so the fix changes no built document and no
  golden part. The tests set the field on a loaded config.
- **The logo.** `house.yaml` states `px: [509, 185]`, `width_pt: 146.25` and
  `height_pt: 53.16` since v2.20.0 (`git show 6c3f64c`). `px` is read in three
  places: `house.validateLogo` and `validateLogoPNG` (a base64 house),
  `cmd/gdoc/logo.go` (it builds `asset.Want{Mime, Width, Height}`), and
  `asset.hold`, which compares the PNG header to the want. `height_pt` is read
  by `render.logoDrawing` for `wp:extent` and `a:ext`, and by
  `validateLogoPlacement`. `go/internal/prelude` writes no logo at all, and
  `restyle` only reports the first-page header as something it cannot write
  (`restyle/readback.go`). So the Docs side needs no rule.
- **Why a side ceiling stays in `internal/asset`.** `hold` reads the PNG
  header first and judges the size on it, because `png.Decode` allocates the
  whole picture from what the header states.
  `TestAHugeStatedSizeIsRefusedWithoutDecoding` feeds a 68-byte PNG that states
  30000x30000 and wants it refused naming `30000x30000` while allocating under
  64 MiB. The byte ceiling, `MaxBytes`, does not bound that, because the
  compressed size says nothing about the decoded size. With `px` gone from the
  want, this room still needs a bound before the decode. It becomes a constant
  of the room, `MaxSide = 4096` pixels on either side, and not a statement in
  the house file. 4096 is eight times the firm's current logo across, and the
  worst decode it admits is 4096 x 4096 x 4 bytes, 64 MiB, once a day.
- **The drift pins for the logo.** `go/internal/drift/compare.go` `Known`
  carries "logo height (pt)", and `pins_test.go` `knownOffline` pins it at
  {`53.16`, `72.75`}. Drawn by ratio from the 509x185 placeholder,
  146.25 x 185 / 509 = 53.155697 pt, which `emu` rounds to 675077 EMU and
  drift's `emuPt` prints as `53.156`. So the pin becomes {`53.156`, `72.75`},
  and `render/headfoot_test.go`'s `cy` literal goes from `675132` to `675077`.
- **The top headings.** `house.yaml` lines 89-104: `heading_1` and `heading_2`
  carry `space_after_pt: 0`. The literal tests that state it:
  `render/styles_test.go` `TestHeading1IsSixteenPointBoldNavyWithTwelvePointsAbove`
  (`w:after` want 0), `restyle/style_test.go` `TestAHeadingTakesTheHouseHeadingLook`
  (`spaceBelow` want 0), and `house/house_test.go` `TestHeading1IsTheHouseHeading`,
  which states no `space_after_pt` yet. `restyle/style.go` reads
  `cfg.Styles[key].SpaceAfterPt`, so restyle applies the new value with no
  code change. The drift items `HEADING_1 spaceBelow` and `HEADING_2
  spaceBelow` will differ from the master and need `Known` rows, the way the
  logo height went in v2.20.0.
- **Fix 13.** `go/internal/export/strip.go`, `stripHeadingNumbers`, runs the
  regex `numberedHeading` (`^(#{1,6} )([0-9]+(?:\.[0-9]+)*)-(\S.*)$`) over every
  line of the projected markdown. A `# 1-Install` line inside a fenced code
  block loses its number. `keptNumbers` calls the same function, so it names
  that line in its warning too. `repoint` rewrites `](#...)` anywhere,
  code included.
- **The id rule today.** `body` parses with `parser.WithAutoHeadingID()` and
  goldmark's default `parser.IDs`. That generator keeps ASCII letters and
  digits, lower-cased, turns a space, `-` or `_` into `-`, skips every
  multi-byte character, falls back to `heading` when nothing is left, and adds
  `-1`, `-2` to a repeat. So `## Цели проекта` is `heading`, every one-word
  Russian heading is `heading`, `heading-1`, ..., and a link written
  `#цели-проекта` names no heading and prints as plain text with a warning.
  `view.Anchor` (`view/text.go:1095`) asks goldmark's default generator, fresh
  per call, and `export` writes every heading link through it.
- **GitHub's rule against goldmark's, on ASCII.** They agree on letters,
  digits, spaces, hyphens and dropped punctuation, and on the `-1`, `-2`
  suffix. They differ in two places: GitHub keeps `_` where goldmark writes
  `-` ("under_score and space" is `under_score-and-space` against
  `under-score-and-space`), and GitHub gives nothing for a heading with no
  letters where goldmark gives `heading`. On accented Latin they differ too:
  `## Résumé of Änderungen: naïve café` in `body/testdata/docs/05-edge-cases.md`
  is `rsum-of-nderungen-nave-caf` today and `résumé-of-änderungen-naïve-café`
  by GitHub's rule. No other heading in `body/testdata/docs/` holds a `_` or a
  non-ASCII character. The live probe note `headingIDNoteSource` in
  `go/internal/live/publish_test.go` holds three ASCII headings, "Where a
  heading id comes from", "Who decides" and "What the reader gets", and one
  link, `#who-decides`.
- **How an id that worked before keeps working.** The new rule is GitHub's
  whole, with one exception: a heading with nothing left keeps goldmark's
  `heading`, because GitHub's answer there is no id at all. Every heading also
  keeps its old goldmark id as an alias that a `#` link resolves through, so a
  note written before today builds exactly as it did. A new id always wins
  over another heading's alias. `export` writes the new id. ASCII ids with no `_`
  keep their bookmark names byte for byte, and no heading in
  `body/testdata/docs/` holds a `_`, so the body goldens change only at the
  Résumé heading.
- **The bookmark.** `body.bookmarkName` turns an id into `_h_` plus the id
  with every character outside `[A-Za-z0-9_]` as `_`, capped at 40 characters
  with seven hex digits of FNV-1a. A Cyrillic id becomes `_h_` and a row of
  underscores, so two Russian headings of one length share a name. The
  measurement of 2026-10-26 (MEASURED.md, "Drive's import of a Russian heading
  and a link to it") showed that a link through an ASCII `_h_` plus hash
  bookmark lands on a Russian heading after Drive's import. The form is
  decided.
- **Fix 14.** `view/text.go` `headingWords` maps each Docs heading id to the
  heading's words, and `target()` writes `"#" + Anchor(words)`. `Anchor` is a
  fresh generator per call, so two headings named "Scope" both get `#scope`,
  and a link to the second lands on the first. The comment above `Anchor`
  says so as if it were markdown's own behaviour. It is not: goldmark, and
  so `body`, gives the second one `scope-1`. In `export`, the strip makes the
  same collapse out of distinct headings: "1-Scope" and "2-Scope" are
  `1-scope` and `2-scope`, both become "Scope", and `repoint` maps both links
  to `#scope`.
- `go/internal/body` does not import `go/internal/view` today. Its tests do.
  `view` imports goldmark's parser and `internal/docs`, and nothing that
  imports `body`, so `body` importing `view` makes no cycle.
- The run strips `GDOC_LIVE_*`. No live test runs in the worker.

## Development Approach

- **TDD.** Each task writes its tests first and runs them red before the fix.
  The bug report's own verification tests are lost, so every test below is
  described from the code as it stands.
- One commit per task. No push, PR, merge or tag.
- A house-style test states its value as a literal, never by reading the
  constant it checks (`TestHeadingNumberingIsTheLiteralFormat`).
- A golden part is regenerated with `-update` only where a task says it
  changes, and the commit message names which parts moved and why.
- Every `doc.go` a task touches names the test for each rule it adds, in the
  same task.
- `CLAUDE.md`, `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md` and
  `docs/v2/MEASURED.md` are not edited.
- `make test` keeps the race flag. Nothing here reaches Drive.

## Testing Strategy

- Unit tests in the package each fix lands in: `guard`, `render`, `house`,
  `asset`, `drift`, `restyle`, `export`, `view`, `body`, and `cmd/gdoc` for the
  logo want.
- The render goldens: Task 2 changes none, Task 3 changes `header2.xml` (the
  logo extent), Task 4 changes `styles.xml` (two `w:after` values).
- The body goldens: Task 6 changes `05-edge-cases.xml` at the Résumé heading's
  bookmark, and no other golden.
- The offline drift gate in `make test`, with the `Known` rows and pins Tasks
  3 and 4 name.
- A round trip test in `body` per id task: the text `view` writes is parsed by
  `body`, and every link resolves to the heading it named.
- `make vet` for gofmt.

## Solution Overview

Each fix lands in the room that owns the rule, and nowhere else:

- The guard reads a `null` the way it reads a missing segment, at every
  segment of the walk.
- Render reads the fields it was dropping, with the same fallback the header
  already uses: the paragraph's own value, else the region's or the cover's.
- The house file states the logo's mime, width and placement, and nothing
  about its pixels. Render reads the ratio off the PNG header it is handed.
  Asset holds the bytes to the mime, the byte ceiling and a side ceiling of its
  own.
- The heading id rule lives once, in `view`, as a goldmark `parser.IDs`. `body`
  hands it to its parse, `view.Anchor` asks it, and a new `view.HeadingLines`
  parses a markdown file with it, which `export` uses to find its headings.
  So the strip, the links and the build all read headings the same way.

## Technical Details

**The null leaf.** In `walkMaskPath`, a segment whose raw value is `null`
(after `bytes.TrimSpace`) is refused, naming the kind, the mask path and the
word `null`, for example: `the fields mask of "updateTextStyle" names "bold",
and "textStyle" holds it as null, which the server reads as unset: the
property is reset to its default`. The check sits beside the missing-segment
refusal, so it holds at the leaf and at every step on the way.

**The cover line.** For a line with words, `paraOpts` takes `Align:
line.Align` else `c.Align`, and `Line: line.LineSpacing`. The run takes
`Italic: line.Italic` and `Color: line.Color`. For a line made of runs, each
run's colour is `r.Color` else `line.Color`, and italic is the line's. A
filled placeholder loses the template's own run colour and highlight, as
today, and keeps the line's colour, because the line's colour is its look and
not a mark for a person to fill in. A blank cover line takes `line.Align` else
`c.Align`.

**The footer.** `footerPart` gains what `regionParagraph` does: `Align:
spec.Align` else `region.Align`; a paragraph with a `rule` writes `ruleRun` and
nothing else; a paragraph with `logo: true` fails the build with `the logo is
anchored in a footer, and only the first page's header carries it`. The
indent stays the region's.

**The label.** `label()` passes `KeepNext: l.KeepWithNext`.

**The logo.**

- `house.Logo` loses `Px` and `HeightPt`. The strict YAML read then refuses a
  house file that still states `px` or `height_pt`, naming the key, which is
  the message a draft house file pinning pixels should get.
- `validateLogo` keeps the mime check, the one-of-two-places check and the
  placement check, which now asks `width_pt` and the anchor. `validateLogoPNG`
  checks that the base64 bytes decode as a PNG header with two sizes above
  zero.
- `asset.Want` is `{Mime}`. `hold` reads the header, refuses a side over
  `MaxSide` naming the stated size (`30000x30000`) before any decode, then
  decodes whole as today. `TestAHugeStatedSizeIsRefusedWithoutDecoding`
  stays as it is. `MaxSide` is 4096, with the reason in its comment and a
  literal test.
- `render.Build` reads the logo's PNG header with `png.DecodeConfig` and refuses
  bytes that are not a PNG, by name, beside its refusal of empty bytes. The
  height is `width_pt x height_px / width_px`, and `logoDrawing` writes it into
  both `wp:extent` and `a:ext`. The anchor offsets do not change.
- `cmd/gdoc/logo.go` builds `asset.Want{Mime: cfg.Logo.Mime}`.

**The top headings.** `house.yaml` `heading_1` and `heading_2`:
`space_after_pt: 6`. Two `Known` rows, `HEADING_1 spaceBelow` and `HEADING_2
spaceBelow`, with the reason: `Nail's decision of 2026-10-26: six points after
the two top headings, as heading_3 and heading_4 carry; the master says 0`.
Two `knownOffline` pins, expected {`6`, `0`}. The master side is whatever the
gate's failure prints for it, written as that literal.

**The heading id rule** (`view.NewHeadingIDs`, a goldmark `parser.IDs`):

- trim the value, lower-case it with `unicode.ToLower`;
- keep a rune that is a letter (`unicode.IsLetter`), a number
  (`unicode.IsNumber`) or a mark (`unicode.IsMark`), and keep `-` and `_`;
- turn each space into `-`, with no collapsing, as GitHub does;
- drop everything else, emoji and punctuation included;
- when nothing is left, use `heading`, goldmark's answer, because GitHub's is
  no id at all;
- a repeat takes `-1`, then `-2`, skipping any value already taken, which is
  goldmark's loop and GitHub's result.

`view.Anchor(s)` is a fresh `NewHeadingIDs()` asked once. Examples, all
literals in the tests: `Цели проекта` is `цели-проекта`; `1.1-Цели проекта` is
`11-цели-проекта`; `Ёлка` is `ёлка`; `under_score and space` is
`under_score-and-space`; `Risk & Control` is `risk--control`; `+++` is
`heading`; `Résumé of Änderungen: naïve café` is
`résumé-of-änderungen-naïve-café`.

**The build side** (`body`):

- `Render` already builds one parser context, `pc` (`body.go`, about line
  190), which carries the footnote snapshot. That same `pc` gains the ids:
  `parser.NewContext(parser.WithIDs(view.NewHeadingIDs()))`. No second,
  fresh context is made.
- `headingAnchors` returns `map[string]string`, a link target to the bookmark
  name of the heading it lands on. It holds every heading's new id. Then, in a
  second pass over the same headings in the same order, it adds each
  heading's legacy id, from goldmark's default generator
  (`parser.NewContext().IDs()`), where no new id already holds that key. A
  figure-only heading and a heading inside a footnote definition are left out,
  as today, but they still advance both generators, because goldmark counted
  them.
- A `#` target is looked up as written. A percent-encoded target is not
  decoded: the trust guardian's skeptic found nothing that asks for it.
- `bookmarkName(id)`: an id that is all ASCII and holds no `_` keeps today's
  name, byte for byte. An id with any other character, or with a `_`, is
  `_h_` plus the first seven hex digits of the FNV-1a 32 sum of the id's UTF-8
  bytes. The lead's call: under GitHub's rule `a_b` and `a-b` are two ids, and
  today's rewrite would give both the name `_h_a_b`, so a note with both
  headings would trip the collision refusal below and stop building. Hashed,
  they get two names, and a note that built yesterday still builds. A
  bookmark name lives only inside the docx, and every `#` link resolves to it
  at build time, so changing the name for a `_` id breaks no link.
- Two different headings that would carry one bookmark name fail the build,
  naming both headings' words. **This refusal is an addition beyond the
  decision, accepted by the lead**: two ids that hash to one bookmark would
  send a link to the wrong heading, which is the failure the bookmark exists
  to prevent, and the check is cheap.

**The read side** (`view`):

- `HeadingLines(md string) []HeadingLine`, where `HeadingLine` is
  `{Line int, Text string, ID string}`, with `Line` zero-based. It parses `md`
  with goldmark, with the GFM and Footnote extensions `body` uses and the
  shared ids, and lists every ATX or setext heading goldmark found, in order,
  except a heading inside a footnote definition. A `#` line inside a fenced
  code block is not a heading to goldmark, so it is not one here.
- The emitter keeps an ordered slice of the `#` heading lines it writes, in
  output order: the Docs heading id and the projected line, appended as each
  line is written. It does not read the `e.lines` map for this, because a map
  has no order. Once the fixpoint walk has settled, one `NewHeadingIDs()` runs
  over that slice in order, so a repeat gets its suffix, and the result maps
  each Docs heading id to its anchor. `target()` reads that map.
- Only the headings the tab file itself writes are counted. A prelude heading
  that export strips, a heading in another tab, and a heading inside a table
  cell are never appended to the slice, so they take no number in the
  sequence. A link to a heading outside the slice keeps today's answer: a
  fresh `Anchor` of its words, or its id.

**The export side** (`export/strip.go`):

- `stripHeadingNumbers` takes its candidate lines from `view.HeadingLines`,
  not from every line, so a fenced code block is left alone (Task 5).
- `repoint` stops comparing `Anchor(numbered)` with `Anchor(words)`. It asks
  `HeadingLines` for the ids before the strip and after it. The i-th heading's
  old id maps to its new id, so "1-Scope" and "2-Scope" map to `scope` and
  `scope-1`. Old ids are unique, so one pass over the targets still cannot
  chain (`TestRepointDoesNotChainRewrites`). A `](#...)` inside a fenced code
  block is not rewritten (Task 7).

## What Goes Where

- **Implementation Steps**: the guard, the render fields, the logo, the
  spacing, the fence, the ids, the repeated headings, the acceptance run, the
  close.
- **Post-Completion**: the live checks a person or the lead runs, the meter,
  and Nail's go for the push and the PR.
- **Out of scope, on purpose**: the rest of
  `docs/backlog/small-defects-from-the-outside-bug-report.md`, `AllowFile`
  accepting any level, restyle applying the whole template
  (`docs/backlog/restyle-applies-the-whole-house-template.md`), the Docs-side
  cover in `internal/prelude` (it also drops a line's own `align`, `italic` and
  `color`, and the house file states none), and anything in `presence/`.

## Implementation Steps

### Task 1: the guard refuses a null leaf in an in-place mask

**Files:**
- Modify: `go/internal/guard/policy.go`, `go/internal/guard/inplace_test.go`,
  `go/internal/guard/doc.go`

- [x] test first: `TestAMaskNamingANullFieldIsRefused` in `inplace_test.go`,
      table-driven like `TestAMaskMayNameOnlyWhatTheRequestSets`, using
      `granted(t)`, `batch(kind, inner)` and `inPlaceURL`. Cases, each refused
      with an error naming `null` and the path:
      `updateTextStyle` `{"textStyle":{"bold":null},"fields":"bold"}`;
      `updateParagraphStyle` `{"paragraphStyle":{"spaceBelow":null},"fields":"spaceBelow"}`;
      `updateDocumentStyle`
      `{"documentStyle":{"pageSize":{"width":{"magnitude":595,"unit":"PT"},"height":null}},"fields":"pageSize.height"}`;
      `updateTableCellStyle` `{"tableCellStyle":{"backgroundColor":null},"fields":"backgroundColor"}`;
      and a `null` beside a set field,
      `{"textStyle":{"bold":true,"italic":null},"fields":"bold,italic"}`,
      which names `italic`. Two cases that are refused today stay refused: a
      `null` parent (`{"textStyle":{"foregroundColor":null},"fields":"foregroundColor.color"}`)
      and a `null` style object (`{"textStyle":null,"fields":"bold"}`). These
      two assert the refusal and that it names the path, and not the word
      `null`, because today's refusal for them is the missing-segment one
- [x] run it: the five `null` leaf cases are red, the two parent cases green
- [x] `walkMaskPath` refuses a segment whose raw value is `null`, as in
      Technical Details
- [x] `TestAMaskNamingExactlyWhatItSetsCarries` and `TestAnOrdinaryMaskCarries`
      still green, untouched; no other guard test edited
- [x] `guard/doc.go`, the paragraph "A mask may name only what the request
      sets": one sentence that a field held as `null` is a field left unset,
      naming `TestAMaskNamingANullFieldIsRefused`
- [x] `make test` green

### Task 2: render writes a cover line's, a footer's and a label's own look

**Files:**
- Modify: `go/internal/render/front.go`, `go/internal/render/headfoot.go`,
  `go/internal/render/front_test.go`, `go/internal/render/headfoot_test.go`,
  `go/internal/render/doc.go`

- [x] test first, each on `config(t)` with one field set, built with
      `houseLogo(t)` and `fields()`:
  - `TestACoverLineCarriesItsOwnLook`: the first cover line ("Altery Group ")
    gets `Align: "right"`, `Italic: true`, `Color: "#123456"`,
    `LineSpacing: 1.5`. Its paragraph has `w:jc w:val="right"` and
    `w:spacing w:line="360" w:lineRule="auto"`; its run has `w:i w:val="1"`
    and `w:color w:val="123456"`. The second line, untouched, still has
    `w:jc w:val="center"`
  - `TestARunLineTakesTheLinesColourWhereTheRunStatesNone`: the version line
    (a `Version: ` run and a run with `placeholder: version`) gets
    `Color: "#123456"`, and its `Version: ` run carries `123456`. The title
    line, built with fields that state no title, gets the same colour: its
    `(Name of)` run carries `123456` and its `Framework/Policy` run keeps
    `ff0000`, because that run states its own
  - `TestAFilledLineKeepsTheLinesColour`: the title line gets
    `Color: "#123456"`; with a title in the fields, the one run carries the
    title's words and `123456`, and no `w:highlight`
  - `TestAFooterParagraphTakesItsAlignment`: `Footer.Default.Paragraphs[1]`
    gets `Align: "center"`, and `Footer.First.Align` is `"right"`. The default
    footer's second paragraph has `w:jc w:val="center"`, and every paragraph
    of the first-page footer has `w:jc w:val="right"`
  - `TestAFooterParagraphDrawsItsRule`: `Footer.Default.Paragraphs[0]` gets
    `Rule: &house.Rule{HeightPt: 1.5, Color: "#A0A0A0"}`. That paragraph holds
    one `v:rect` with `style="width:0.0pt;height:1.5pt"` and
    `fillcolor="#A0A0A0"`
  - `TestALogoInAFooterIsRefusedByName`: `Footer.Default.Paragraphs[1].Logo =
    true`. `Build` fails, and the error contains `footer`
  - `TestALogoInTheDefaultHeaderIsRefusedByName`: `Header.Default.Paragraphs[0].Logo
    = true`. `Build` fails naming "not the first page's". This pins the
    refusal that has no test today
  - `TestALabelKeepsWithTheBlockBehindIt`: `Labels["version_control"]` gets
    `KeepWithNext: true`. Its paragraph carries `w:keepNext w:val="1"`. The
    contents label, untouched, carries none
- [x] run them: all red but the default-header logo test
- [x] `cover()`, `footerPart()` and `label()` as in Technical Details
- [x] `TestTheGoldenPartsAreWhatTheHouseStyleRendersTo` green with no
      `-update`: the embedded house states none of these fields, so no part
      moves
- [x] `render/doc.go`: a short section "A block's own look reaches the page",
      naming the eight tests above, and the footer logo beside the header one
- [x] `make test` green

### Task 3: a logo of any size fits by width

**Files:**
- Modify: `go/internal/house/house.go`, `go/internal/house/house.yaml`,
  `go/internal/house/house_test.go`, `go/internal/house/testdata/house-base64.yaml`,
  `go/internal/asset/asset.go`, `go/internal/asset/asset_test.go`,
  `go/internal/asset/doc.go`, `go/internal/render/render.go`,
  `go/internal/render/headfoot.go`, `go/internal/render/headfoot_test.go`,
  `go/internal/render/render_test.go`, `go/internal/render/testdata/header2.xml`,
  `go/internal/render/doc.go`, `go/cmd/gdoc/logo.go`,
  `go/cmd/gdoc/logo_test.go`, `go/internal/drift/compare.go`,
  `go/internal/drift/pins_test.go`, `go/internal/live/houselogo_test.go`
  (it reads `cfg.Logo.Px` and builds a `Want` from it, and `go test ./...`
  compiles it even with the live variables stripped: the trust guardian's
  condition),
  `docs/backlog/the-logo-drive-file-cannot-be-changed.md`
- Delete: `docs/backlog/a-logo-of-another-size-stops-every-build.md`

- [x] test first:
  - `render`: `TestTheLogoHeightFollowsThePicturesRatio`: a 400x100 PNG made
    in the test with `image/png`. `wp:extent` and `a:ext` read `cx="1857375"`
    (146.25 pt) and `cy="464344"` (36.5625 pt)
  - `render`: `TestBuildRefusesALogoThatIsNotAPNG`: the bytes `GIF89a` fail
    `Build`, naming `PNG`
  - `render`: `TestTheFirstPageHeaderCarriesTheLogoWhereTheHouseAnchorsIt`
    states `cy="675077"` (53.156 pt, the placeholder's own ratio) in place of
    `675132`
  - `house`: `TestTheLogoDecodesAsAPNGOfTheStatedSize` becomes
    `TestTheLogoStatesItsMimeWidthAndPlacement`: mime `image/png`,
    `width_pt` 146.25, anchor offsets 404.25 and -5.15, all literals; no
    pixel or height assertion
  - `house`: `TestALogoThatStatesPixelsIsRefusedNamingTheKey`: a copy of the
    embedded file with `px:` put back, and another with `height_pt:` put back,
    each refused naming the key
  - `asset`: `TestAPictureOfAnySizeIsCarried`: Drive answers a 16x8 PNG and
    then a 509x185 PNG with `Want{Mime: "image/png"}`, each carried and
    cached
  - `asset`: `TestASideOverTheCeilingIsRefusedBeforeDecoding`:
    `pngHeaderOf(t, 4097, 1)` is refused naming `4097x1`;
    `TestTheSideCeilingIs4096` states the literal
  - `asset`: in `TestABadFileRefusesEvenWithAGoodCache`, the case "the wrong
    pixels" goes, because that is now a carried file. "not a PNG" and "a mime
    this room does not carry" stay. `TestAHugeStatedSizeIsRefusedWithoutDecoding`
    and `TestAHalfPNGIsRefusedByBothRoutes` stay unchanged
  - `drift`: `knownOffline["logo height (pt)"]` becomes {`53.156`, `72.75`}
- [x] run them: red
- [x] `house`, `asset`, `render` and `cmd/gdoc/logo.go` as in Technical
      Details. `house.yaml` and `testdata/house-base64.yaml` lose `px` and
      `height_pt`. Every other `Want{...Width, Height}` in the asset and
      `cmd/gdoc` tests drops the two sizes
- [x] `go/internal/live/houselogo_test.go`: the two places that read
      `cfg.Logo.Px` (about line 96, and `liveHouseLogo` about line 240) build
      `asset.Want{Mime: cfg.Logo.Mime}`. Where the first one logged the stated
      size, the test logs the size `png.DecodeConfig` reads off the bytes it
      got instead. `go vet ./internal/live` compiles it with no live variable
      set
- [x] `drift/compare.go`: the `Known` comment says the house draws the mark
      `width_pt` wide at the picture's own ratio, and the master carries the
      old mark at 72.75 pt
- [x] `go test ./internal/render -run TestTheGoldenParts -update`: only
      `testdata/header2.xml` moves, at the two `cy` values; the commit message
      says so
- [x] `house/house.go` package comment, `asset/doc.go` ("A wrong file refuses
      even when the cache is good", the size sentences, `Want`),
      `render/doc.go` (the logo paragraph) and the comment on `logoFor` in
      `cmd/gdoc/logo.go` say mime, byte ceiling and side ceiling, and name the
      new tests. Nothing in any of them still says "the stated size" or
      "pixels the file states"
- [x] `docs/backlog/the-logo-drive-file-cannot-be-changed.md`: the last bullet,
      which says a logo of another size is refused, is replaced by one line
      saying any size now fits by width
- [x] `git rm docs/backlog/a-logo-of-another-size-stops-every-build.md`
- [x] `make test` green

### Task 4: six points after the two top headings

**Files:**
- Modify: `go/internal/house/house.yaml`, `go/internal/house/house_test.go`,
  `go/internal/render/styles_test.go`, `go/internal/render/testdata/styles.xml`,
  `go/internal/restyle/style_test.go`, `go/internal/drift/compare.go`,
  `go/internal/drift/pins_test.go`
- Delete: `docs/backlog/no-space-after-the-two-top-headings.md`

- [x] test first, literals only:
  - `house`: `TestHeading1IsTheHouseHeading` asserts
    `heading_1.space_after_pt` is 6; a new `TestHeading2HasSixPointsAfter`
    asserts the same of `heading_2`
  - `render`: `TestHeading1IsSixteenPointBoldNavyWithTwelvePointsAbove` wants
    `w:after="120"`; a new `TestHeading2HasSixPointsAfter` wants
    `w:after="120"` on `Heading2`
  - `restyle`: `TestAHeadingTakesTheHouseHeadingLook` wants `spaceBelow` 6,
    and its comment says "6pt below"
- [x] run them: red
- [x] `house.yaml`: `space_after_pt: 6` on `heading_1` and `heading_2`
- [x] run `make test` and read the drift gate's failure: it names
      `HEADING_1 spaceBelow` and `HEADING_2 spaceBelow` and prints both sides
- [x] `drift/compare.go` `Known`: the two rows with the reason in Technical
      Details and a comment above them. `pins_test.go` `knownOffline`: the
      two pairs, written as the literals the failure printed, expected
      {`6`, `0`}
- [x] `go test ./internal/render -run TestTheGoldenParts -update`: only
      `testdata/styles.xml` moves, at two `w:after` values
- [x] any other test that stated the old 0 for these two headings, found by
      the run, states 6; none is loosened to stop stating a value
- [x] `git rm docs/backlog/no-space-after-the-two-top-headings.md`
- [x] `make test` green

### Task 5: export leaves a fenced code block alone

**Files:**
- Modify: `go/internal/view/text.go`, `go/internal/view/text_test.go`,
  `go/internal/export/strip.go`, `go/internal/export/project_test.go`,
  `go/internal/export/doc.go`, `go/internal/view/doc.go`

- [x] test first:
  - `export`: `TestAHeadingNumberInsideAFenceStays`. Input to
    `stripHeadingNumbers`: `# 1-Scope`, a blank line, a ```` ```sh ```` fence
    holding `# 1-Install the tool`, and a `~~~` fence holding
    `## 2.1-Not a heading`. Output: `# Scope`, both fences byte for byte, one
    piece (`1-Scope`)
  - `export`: `TestKeptNumbersNamesNoLineInsideAFence`: `keptNumbers` over the
    same input names only `1-Scope`
  - `view`: `TestHeadingLinesFindsWhatGoldmarkFinds`: over a file holding an
    ATX heading, a setext heading, a `#` line in a fence, a `#` line indented
    four spaces, and a heading inside a footnote definition,
    `HeadingLines` lists the first two only, with their zero-based line
    numbers and words
- [x] run them: red
- [x] `view.HeadingLines` as in Technical Details, with goldmark's default ids
      for now; `stripHeadingNumbers` applies `numberedHeading` only to the
      lines `HeadingLines` names
- [x] `export/doc.go`, "A heading gives its house number back": one sentence
      that a heading is what goldmark parses as one, so a line inside a code
      block keeps its digits, naming the two export tests. `view/doc.go`:
      `HeadingLines` and its test, in the link section
- [x] `make test` green

### Task 6: a heading id keeps its letters, and an old id still resolves

**Files:**
- Modify: `go/internal/view/text.go`, `go/internal/view/text_test.go`,
  `go/internal/view/doc.go`, `go/internal/body/body.go`,
  `go/internal/body/build.go`, `go/internal/body/anchors_test.go`,
  `go/internal/body/doc.go`, `go/internal/body/testdata/golden/05-edge-cases.xml`,
  `go/internal/export/project_test.go`
- Create: `go/internal/body/testdata/golden/heading-ids.txt`

- [x] test first, the pin on today's ids, before any rule changes:
  - `body`: `TestTodaysHeadingIDsStillResolve`. For each
    `testdata/docs/*.md`, and for the three headings of the live probe note
    stated as literals in the test (`where-a-heading-id-comes-from`,
    `who-decides`, `what-the-reader-gets`), it holds the golden
    `testdata/golden/heading-ids.txt`, one line per heading:
    `<file>\t<id>`. Written now, it asserts two things. Each golden id equals
    what goldmark's default generator (`parser.NewContext().IDs()`) gives that
    heading, so the golden cannot drift onto the new rule. Each golden id, as
    a `#` link in that file, resolves to that same heading's bookmark
  - the pin has its own flag, `-update-heading-ids`, declared beside the test,
    and not the shared `-update` of `body_test.go:22`, so regenerating the
    body goldens can never rewrite it
  - write the golden with `go test ./internal/body -run
    TestTodaysHeadingIDsStillResolve -update-heading-ids` now, on today's
    code, and run it without the flag: green
- [x] test first, the new rule:
  - `view`: `TestAnAnchorKeepsEveryLetter`: the seven literal examples in
    Technical Details
  - `view`: `TestARepeatedIDTakesTheNextSuffix`: one `NewHeadingIDs()` asked
    `Итоги`, `Итоги`, `Итоги-1`, `Итоги` gives `итоги`, `итоги-1`,
    `итоги-1-1`, `итоги-2`
  - `body`: `TestALinkToACyrillicHeadingIsAJump`: `## Цели проекта` and
    `[goals](#цели-проекта)`. No warning; the link is a `w:hyperlink` whose
    `w:anchor` is the heading's bookmark, and that name matches
    `^_h_[0-9a-f]{7}$`
  - `body`: `TestTwoOneWordRussianHeadingsAreTwoIDs`: `# Цели` and `# Риски`,
    linked as `#цели` and `#риски`, land on two different bookmarks
  - `body`: `TestARepeatedCyrillicHeadingTakesASuffix`: `## Итоги` twice;
    `#итоги` and `#итоги-1` land on two different bookmarks
  - `body`: `TestAnAccentedHeadingAnswersToBothIDs`: the Résumé heading;
    `#résumé-of-änderungen-naïve-café` and `#rsum-of-nderungen-nave-caf` land
    on the one bookmark, with no warning
  - `body`: `TestTwoHeadingsSharingABookmarkNameAreRefused`: the test searches
    `я1`, `я2`, ... for two ids whose seven-digit hashes match (a few tens of
    thousands of tries, deterministic), builds a note with those two
    headings, and wants the build refused naming both
  - `body`: `TestBookmarkNameIsWordSafe` keeps its ASCII cases that hold no
    `_` unchanged and adds two: `bookmarkName("цели-проекта")` matches
    `^_h_[0-9a-f]{7}$`, and so does `bookmarkName("a_b")`
  - `body`: `TestAnUnderscoreAndAHyphenAreTwoBookmarks`: a note holding
    `# a_b` and `# a-b`, each linked as `#a_b` and `#a-b`, builds with no
    refusal, and the two links land on two different bookmarks
  - `export`: `TestAHeadingLinkRoundTripsToItsSlug` in `project_test.go`
    (about line 215) wants `[the changes](#résumé-of-änderungen-naïve-café)`
    in place of `#rsum-of-nderungen-nave-caf`. The ampersand case stays
    `#risk--control-ampersands`, which both rules give. Its comment says the
    rule is GitHub's through `view.NewHeadingIDs`, which keeps an accented
    letter, rather than goldmark's default, which dropped it
  - `body`: `TestTheAnchorViewWritesIsTheIDBodyCollects` parses through the
    same context `Render` builds rather than through `parse()` alone, and adds
    `Цели проекта` and `1.1-Цели проекта` to its headings
- [x] run them: red
- [x] `view.NewHeadingIDs` and `view.Anchor` as in Technical Details;
      `view.HeadingLines` uses the new ids
- [x] `body`: the parse context, `headingAnchors` as a map to bookmark names
      with legacy aliases, `bookmarkName`, the
      collision refusal, and `build.go` reading the map
- [x] once `view.NewHeadingIDs` exists, `TestTodaysHeadingIDsStillResolve`
      gains its third assertion: for every heading that is ASCII and has no
      `_`, the new id equals the golden id, so export writes the same link it
      wrote yesterday
- [x] `TestTodaysHeadingIDsStillResolve` green, and
      `git diff -- go/internal/body/testdata/golden/heading-ids.txt` is empty
      after its first write: the file is never regenerated after that
- [x] `go test ./internal/body -run TestTheSixDocumentsRenderToTheirGoldens
      -update` for the body goldens: only
      `05-edge-cases.xml` moves, at the Résumé heading's bookmark name and
      any jump into it; the commit message names it
- [x] `body/doc.go`, "Every heading carries a hidden bookmark": the id rule is
      `view`'s and GitHub's, the one exception (`heading`), the legacy alias
      and why, the hashed name for a non-ASCII id with the measurement of
      2026-10-26 named, the collision refusal, each with its test.
      `view/doc.go`, the link section: the rule is no longer goldmark's
      default but `NewHeadingIDs`, which `body` hands to goldmark, with its
      tests
- [x] `make test` green

### Task 7: a link to a repeated heading names that heading

**Files:**
- Modify: `go/internal/view/text.go`, `go/internal/view/text_test.go`,
  `go/internal/view/doc.go`, `go/internal/export/strip.go`,
  `go/internal/export/project_test.go`, `go/internal/export/doc.go`,
  `go/internal/body/anchors_test.go`

- [x] test first:
  - `view`: `TestALinkToTheSecondOfTwoHeadingsNamesTheSecond`: a Docs
    fixture with two `HEADING_2` paragraphs reading "Scope", heading ids
    `h.a` and `h.b`, and a paragraph linking to each. The projection holds
    `](#scope)` and `](#scope-1)`
  - `body`: `TestARepeatedHeadingsLinkIsTheIDBodyCollects`: the text the
    fixture above projects, walked by `body`: both links resolve with no
    warning and land on two different bookmarks
  - `export`: `TestNumbersThatCollapseIntoOneHeadingKeepTheirLinksApart`:
    `# 1-Scope`, `[a](#1-scope)`, `# 2-Scope`, `[b](#2-scope)` strip to two
    `# Scope` headings with the links `#scope` and `#scope-1`
  - `export`: `TestALinkInsideAFenceIsNotRepointed`: a fenced block holding
    `[a](#1-scope)` keeps it byte for byte while the same link outside is
    repointed
  - `TestRepointDoesNotChainRewrites` and
    `TestAHeadingLosesItsHouseNumberAndKeepsItsLinks` stay green, unchanged
- [x] run them: red
- [x] `view`: the ordered slice of the `#` heading lines the tab file writes,
      the generator run over it in output order once the walk settles, and
      `target()` reading the result, as in Technical Details
- [x] `export`: `repoint` by position through `HeadingLines`, skipping lines
      inside a fence
- [x] `view/text.go`: the comment above `Anchor` no longer says two headings
      share an anchor
- [x] `view/doc.go` and `export/doc.go` say a repeated heading is linked by
      its own suffixed id, naming the tests
- [x] `make test` green

### Task 8: Verify acceptance criteria

- [ ] each of the seven fixes has its red-first test green: Tasks 1 to 7
- [ ] `git diff --name-only main...HEAD` lists none of `CLAUDE.md`,
      `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md`, `docs/v2/MEASURED.md`, and
      nothing under `presence/`
- [ ] the two backlog items are gone and
      `docs/backlog/small-defects-from-the-outside-bug-report.md` is unchanged
- [ ] `git log --format=%h main..HEAD -- go/internal/body/testdata/golden/heading-ids.txt`
      names exactly one commit, Task 6's, and `grep -n "update-heading-ids"
      go/internal/body/*_test.go` shows the pin's own flag, so no later
      `-update` of the body goldens could have rewritten it
- [ ] the goldens that moved are the ones the tasks named and no other:
      `header2.xml`, `styles.xml`, `05-edge-cases.xml`
- [ ] no guard test lost an assertion: `git diff main...HEAD --
      go/internal/guard/*_test.go` adds lines and removes none
- [ ] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
      `make platform-test`, `make team-check`

### Task 9: [Final] Update documentation

- [ ] every `doc.go` touched names a test for every rule it states, or
      `TODO(test)`: `guard`, `render`, `house`, `asset`, `body`, `view`,
      `export`
- [ ] `go doc` over each touched package prints its package comment and
      nothing a file comment meant for itself
      (`TestOnlyThePackageCommentReachesGoDoc` green)
- [ ] `CLAUDE.md` untouched, and its task map still names files that exist
- [ ] move this plan to `docs/plans/completed/`

## Post-Completion

These are the lead's, with Nail.

**Live checks, by hand, outside the worker.**

- Publish a note holding `## Цели проекта`, a second `## Цели проекта`, and a
  link to each, with `GDOC_LIVE_WRITE=1` into the test folder. Both links open
  their own heading in Docs.
- `gdoc export` that document back: the links read `#цели-проекта` and
  `#цели-проекта-1`.
- `gdoc build` with the firm's logo from Drive: the mark is 146.25 pt wide and
  53.16 pt high, at its own ratio, in Word and after publish.
- The live drift gate (`GDOC_LIVE_TEST=1`) reports the two heading spacing
  rows as known and nothing new.

**The measurement.** `.ralphex/board/meter.py ../gdoc-m30` into
`docs/team/measure.md`, with the lead's context at the end and the number of
messages Nail sent.

**The release.** `make release BRANCH=gdoc-v2-m30-the-fixes` stops at a green
PR. The push, the PR, the merge and the tag wait for Nail's word. This changes
what a build writes, so it is a minor version.
