# gdoc v2 Milestone 18: footnotes, and the marks. Specification

2026-10-14. Every decision below is Nail's and settled, taken on the decision
page of the same day. Two decisions in one milestone, merged the same evening
because both change the inline walk in `internal/body`, the style writer in
`internal/render` and the Markdown writer in `internal/export`, and both add a
character style to a package that has none. The task list is
`docs/plans/2026-10-14-gdoc-v2-m18-footnotes-and-marks.md`. It starts after
the body spacing change of 2026-10-14 merged on main (`0221240`).

Part A is footnotes: written in the note, set in the docx, imported by Drive
as a Docs footnote, printed by `read`, written back by `export`. Part B is the
marks: an author highlights words in Obsidian's own syntax and colours, or
pencils a span grey, and the Doc carries it. Each part has one live
measurement at the front of the plan, and each measurement stops only its own
part.

## Principles

Serves: 1, two renderings of one text, because a footnote and a mark the
author wrote are in the document and come home on export; 2, the binary
prints facts and the skills judge, because the binary counts marks and never
says whether a document with marks may be published; 4, every word costs a
reader's attention, and a citation under the page costs less than one in the
sentence.

Strains, each needing a DECISIONS.md entry before code, one per part:

- "A file gdoc half understands never reaches a document". A note with a
  footnote reached the document with the footnote dropped and a warning; now
  a note whose footnotes do not line up is refused by line.
- The render package's "twelve parts and the logo" becomes thirteen.
- The `==text==` syntax, which `body/mark.go` says is "the only way to say a
  human still has to fill this in", becomes one of six colours.
- `body` warns about inline HTML and drops it, and has no span syntax. The
  one span this milestone adds is Pandoc's, for pencil alone.

## Part A: footnotes. Decisions taken

Nail, 2026-10-14, on the decision page:

| Rule | Where |
|---|---|
| The syntax is the one goldmark's footnote extension already parses: `[^label]` in the prose and `[^label]: text` on a line of its own. Labels may be words | body |
| A reference with no definition, or a definition nothing references, refuses the build and names the line (2.1 A) | body |
| One label referenced twice is allowed: each reference becomes its own footnote carrying the same text, because Word and Docs hold one note per mark (2.2 B). `export` then writes two notes where the author wrote one, and the round-trip rule names that | body, export |
| The first task of the part is the live measurement of Drive's import (2.3 A) | live |

Taken in the design pass and not contested:

| Rule | Where |
|---|---|
| Footnotes are numbered in first-reference order, Word's own, whatever order the definitions were written in | body |
| Inside a footnote, paragraphs with inline marks only. A list, a table, a picture, a code block or a heading inside a footnote is warned about by line and left out | body |
| Footnote text is a tenth house style, `footnote_text`, 10pt, single-spaced, no space above or below, in the body face. The reference mark is a character style set superscript | house, render |
| `read` keeps printing what it prints today: `[^n]` in the prose, then a `---` line and `[^n]: text` for each note. Nothing in `view` changes for footnotes | view |
| `export` keeps writing that shape, and its warning that publish would refuse the file goes | export |
| A footnote mark inside a heading is refused by line: Word copies the mark into the contents field | body |
| No fourth module. goldmark's extension is already on | boundary |

## Part B: the marks. Decisions taken

Nail, 2026-10-14, on the decision page:

| Rule | Where |
|---|---|
| Highlights are written the way Obsidian 1.14 writes them and no other way: `==words==` for the default yellow, and `==🔴words==`, `==🟠words==`, `==🟡words==`, `==🟢words==`, `==🔵words==`, `==🟣words==` for the six colours. No Pandoc span and no HTML for a highlight | body |
| The six are keyed and named as Obsidian names them: `yellow`, `red`, `orange`, `green`, `blue`, `purple`. No name of gdoc's own. What a colour means is Nail's, said in his notes and his skills, never in a key. His own use, recorded on the decision page: yellow a word or figure to replace, green a claim to confirm, purple a question to one reviewer, blue text changed since the last version. Red and orange are carried without a meaning, so a note that uses them is not refused | house |
| The one grey, `pencil`, is prose that stays in the document but is not the argument. Obsidian has no text colour, so pencil is written as Pandoc's bracketed span, `[words]{.pencil}`, the only span gdoc reads. In the Doc it is text colour #9A9A9A, light grey, chosen knowingly under the 4.5:1 floor after both designers asked for #767676. No italic | body, house |
| No per-person logic anywhere, in the binary or in a skill | all |
| `yellow`, `green` and `purple` are authoring debts: counted on the envelope of `build` and `publish`, and the publish skill asks before a document carrying any goes to Drive (3.3 A). The other four are not counted | cmd, skills |
| house.yaml is the one source of the palette, keyed by the colour. `gdoc help marks` prints the seven with their syntax, Doc colour and fate, and the skills read that rather than carrying a copy | cmd, skills |
| The Doc hexes are the merge of a UX designer's and a typographer's sets, each colour as saturated as its lightness allows and the lightnesses a staircase: red #FF6B6B, purple #BA92FF, blue #5DB3FF, green #3DD67A, orange #FFB85C, yellow #FFD500, pencil #9A9A9A. Off the template's #FFFF00, which means placeholder. A proposal until the measurement says what Drive keeps | house, live |
| Dropped on the way: five named text colours, a per-person legend, the writer's desk names sticky, proof, flag and ink, `coin`, Hilo and every other plugin, an HTML span for pencil, a rule that colours a paragraph by its first words | none |

Taken in the design pass and not contested:

| Rule | Where |
|---|---|
| A highlight carrying anything other than one of the six emoji at its start, or an emoji anywhere else in it, is the words as written. A pencil span inside a highlight, a highlight inside a pencil span, an empty span and a span crossing a paragraph are refused by line | body |
| A mark inside a heading is refused by line, for the reason a footnote mark is | body |
| In the docx each mark is a named character style, `Yellow`, `Green`, `Purple`, `Blue`, `Red`, `Orange`, `Pencil`, so Word lists them and a reader finds every yellow with one search. Google Docs has no character styles, so after import a mark is direct formatting, and that is what export reads | render, export |
| The six highlight colours are run shading with the exact hex, not Word's sixteen named highlights, which carry none of these hexes | render |
| `export` recognises a mark by its exact colour and writes Obsidian's syntax back, or the pencil span. A colour not in the palette is left as plain words and counted on the envelope by hex | export |
| `read` is unchanged for marks: the words reach a reviewing session, the mark around them does not. Printing marks in the projection changes what a session quotes, and is a decision for a later entry | view |
| `restyle` is unchanged. Whether an in-place restyle leaves run shading and run colour alone is measured | restyle |
| The cover's own yellow placeholders are the prelude's and keep writing Word's named yellow. They are not marks | render |

## What is measured before anything is trusted

Two probes, Tasks 2 and 3, each a fixture docx uploaded with conversion into
the test folder, read back bare, exported, trashed, with the answers written
into MEASURED.md either way.

- Footnotes: a docx carrying one Word footnote. Expected: a
  `footnoteReference` element in the body and a `footnotes` map on the
  document, and `word/footnotes.xml` in the export. If Drive drops it, Part A
  stops: footnotes would exist in the docx `build` writes and nowhere else,
  and whether that is worth building is Nail's decision.
- Marks: a docx carrying one run with shading for each of the six hexes, one
  with text colour #9A9A9A, one with Word's named yellow for comparison, then
  one in-place restyle and the eight read again. Expected: every hex back as
  sent, before and after the restyle. If a hex came back changed, or the
  restyle cleared one, Part B stops and says what came back.

## House values that change in this milestone

Nail will change style values during this milestone, size and alignment among
them. The mechanism is fixed and the values are not: each change is a
DECISIONS.md line, a house.yaml line, a literal test re-stated, and a `Known`
entry where the master states the old value. The master docx stays as
provenance and the gate is never loosened, only re-pinned with the reason
written down. Task 14 of the plan holds the steps.

## What this milestone does not do

- A suggestion or a comment inside a footnote stays invisible to `read` and
  `suggestions`, which is `docs/backlog/suggestions-inside-footnotes.md`.
- `propose` and `annotate` do not change. Both already refuse a quote that
  crosses a footnote mark, and a quote matches words with or without a mark
  around them.
- No footnote in a table cell. A mark in one is carried: a cell's paragraph
  walks its inlines the way every other paragraph does, and the kitchen sink's
  own table holds a `==to fill in==` cell (corrected 2026-10-15, Task 9).
- No mark on a whole paragraph, no mark in `read`'s projection, no meaning
  for any colour in the binary.

## Acceptance

Each is a test or a checkable run.

1. `gdoc build` on a note with two footnotes, the second defined before the
   first, writes a docx whose `word/footnotes.xml` carries the two
   separators and two notes numbered in reference order, and whose body
   carries two `w:footnoteReference` runs in the superscript style.
2. A dangling reference, an unused definition and a footnote mark in a
   heading are each refused by line, and nothing is written.
3. `gdoc build` on a note carrying all seven marks writes a docx whose
   `word/styles.xml` carries the seven character styles and whose body
   carries each run in its style with the exact shading or text colour;
   `==T+1==` and `==🟡T+1==` produce the same run.
4. A nested mark, an empty span, a mark in a heading and a span crossing a
   paragraph are each refused by line, and nothing is written.
5. The `build` and `publish` envelopes carry `marks` with a count for
   `yellow`, `green` and `purple`, zero included, and `gdoc help marks`
   prints the seven from the binary.
6. `TestLivePublishCarriesAFootnoteIntoDocs` and
   `TestLivePublishCarriesTheMarksIntoDocs` pass.
7. `TestLiveExportRoundTrip` over a note with footnotes and all seven marks
   passes with the `footnotes` entry of `knownDifferences` narrowed to the
   renumbering rule and no entry for marks.
8. The six goldens render, the drift gate reads 171 rows with no new
   difference, and `make test` is green with the race flag.
