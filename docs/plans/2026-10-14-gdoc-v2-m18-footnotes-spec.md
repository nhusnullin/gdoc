# gdoc v2 Milestone 18: footnotes on every route. Specification

2026-10-14. Every decision below is Nail's and settled, taken on the decision
page of the same day after a design pass over the four routes. The task list
is `docs/plans/2026-10-14-gdoc-v2-m18-footnotes.md`. It starts after the body
spacing change of 2026-10-14 merged on main (`0221240`), because both touch
`internal/body`.

Today a footnote in a note is warned about and left out of the document, and
`export` tells a reader that publish would refuse the file. This milestone
makes a footnote a thing gdoc carries: written in the note, set in the docx,
imported by Drive as a Docs footnote, printed by `read`, and written back by
`export`.

## Principles

Serves: 1, the markdown and the document are two renderings of one text, and
a footnote the author wrote is part of that text; 4, every word costs a
reader's attention, and a citation that sits under the page costs less than
one in the sentence.

Strains, each needing a DECISIONS.md entry before code:

- "A file gdoc half understands never reaches a document". A note with a
  footnote reached the document today with the footnote dropped and a
  warning. Under this milestone a note whose footnotes do not line up is
  refused instead, by line.
- The render package's "twelve parts and the logo" becomes thirteen.

## Decisions taken

Nail, 2026-10-14, on the decision page:

| Rule | Where |
|---|---|
| The syntax is the one goldmark's footnote extension already parses: `[^label]` in the prose and `[^label]: text` on a line of its own. Labels may be words | body |
| A reference with no definition, or a definition nothing references, refuses the build and names the line (2.1 A) | body |
| One label referenced twice is allowed: each reference becomes its own footnote carrying the same text, because Word and Docs hold one note per mark (2.2 B). `export` then writes two notes where the author wrote one, and the round-trip rule names that | body, export |
| The work starts after the spacing merge, and its first task is the live measurement of Drive's import (2.3 A) | live |

Taken in the design pass and not contested:

| Rule | Where |
|---|---|
| Footnotes are numbered in first-reference order, which is Word's own order, whatever order the definitions were written in | body |
| Inside a footnote, paragraphs with inline marks only. A list, a table, a picture, a code block or a heading inside a footnote is warned about by line and left out, the rule every other unrendered block follows | body |
| Footnote text is a tenth house style, `footnote_text`, 10pt, single-spaced, no space above or below, in the body face. The reference mark is a character style set superscript | house, render |
| `read` keeps printing what it prints today: `[^n]` in the prose, then a `---` line and `[^n]: text` for each note. Nothing in `view` changes | view |
| `export` keeps writing that shape, and its warning that publish would refuse the file goes, because it no longer would | export |
| A footnote mark inside a heading is refused by line: Word copies the mark into the contents field | body |
| No fourth module. goldmark's extension is already on | boundary |

## What is measured before anything is trusted

Drive's conversion of a Word footnote into a Docs footnote is believed to
work and has never been measured in this repository. Task 2 uploads a docx
carrying one footnote, reads the document back bare, and looks for a
`footnoteReference` element in the body and a `footnotes` map on the
document. The answer goes into MEASURED.md either way. If Drive drops the
footnote, the plan stops after Task 2 and says so: footnotes would then exist
in the docx `build` writes and nowhere else, and whether that is worth
building is Nail's decision, not the plan's.

## What this milestone does not do

- A suggestion or a comment inside a footnote stays invisible to `read` and
  `suggestions`, which is `docs/backlog/suggestions-inside-footnotes.md`.
- `propose` and `annotate` do not change. Both already refuse a quote that
  crosses a footnote mark, and footnote text stays unquotable.
- `restyle` does not change. Whether a footnote survives an in-place restyle
  was measured on 2026-09-09 and holds.
- No footnote in a table cell. A mark inside a cell is warned about and left
  out, with the cell's words kept.

## Acceptance

Each is a test or a checkable run.

1. `gdoc build` on a note with two footnotes, the second defined before the
   first, writes a docx whose `word/footnotes.xml` carries the two
   separators and two notes numbered in reference order, and whose body
   carries two `w:footnoteReference` runs in the superscript style.
2. A note with a dangling reference, an unused definition, or a mark in a
   heading is refused by line, and nothing is written.
3. `TestLivePublishCarriesAFootnoteIntoDocs`: the published document's bare
   read carries a `footnotes` map with the note's text and a
   `footnoteReference` in the sentence, and its docx export carries a
   `word/footnotes.xml`.
4. `TestLiveExportRoundTrip` over a note with a footnote passes with the
   `footnotes` entry of `knownDifferences` narrowed to the renumbering rule.
5. The six goldens render, the drift gate reads 171 rows with no new
   difference, and `make test` is green with the race flag.
