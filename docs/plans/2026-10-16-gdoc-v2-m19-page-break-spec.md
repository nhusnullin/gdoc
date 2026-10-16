# gdoc v2 Milestone 19: `---` is a page break. Specification

2026-10-16. One decision, Nail's, taken in chat the same day after the
alternatives were put to him: a line of three hyphens in a note is a page
break, on every route. The task list is
`docs/plans/2026-10-16-gdoc-v2-m19-page-break.md`. It starts from `main`
after v2.12.0 (`8652a71`).

Today the body walker turns a thematic break into a horizontal rule, a
paragraph with a grey bottom border, and nothing an author can write in a
note breaks a page. The house file breaks the page in three places of its
own, after the cover, before the contents list and before the first body
heading, and the author cannot move or add one. After this milestone the
author breaks a page by writing `---` on a line of its own, which is the
line Obsidian, Pandoc and CommonMark all read as a thematic break, and the
document carries it the way the house's own breaks are carried.

## Principles

Serves: 1, two renderings of one text, because a break the author wrote is in
the document and comes home on export as the line it left as; 4, every word
costs a reader's attention, because a page that holds nothing is a page the
reader turns for nothing, so a break before nothing is dropped.

Strains: none. Markdown has no page break of its own, so no standard is
contradicted, and the one line gdoc gives a second meaning is one `read`
never printed as `---` in the first place (`view/doc.go`, "A horizontal rule
prints [rule] and not ---").

## Decisions taken

Nail, 2026-10-16, in chat:

| Rule | Where |
|---|---|
| `---` on a line of its own in a note's body is a page break. Not an HTML div, not `\newpage`, not a plugin's `///` | body |

The alternatives put to him, and declined: the HTML div
`<div style="page-break-after: always;"></div>`, which Obsidian's own PDF
export honours but which reads as machinery in a note; and Pandoc's
`\newpage`, which Obsidian prints as text. He was told that the hub holds
1,348 such lines across 206 notes today, nearly all of them dividers before
a heading, and that a note of 216 dividers becomes a document of 216 pages.
He chose `---` anyway. The hub is his, and gdoc reports the count.

Taken in the design pass by Claude, each one a sentence Nail can strike:

| Rule | Where |
|---|---|
| Every thematic break goldmark parses is the same break: `---`, `***` and `___`, three or more, with or without spaces. One node, one meaning. The source spelling is not read | body |
| The docx shape is the one the front matter already uses: an empty paragraph carrying `w:pageBreakBefore`, emitted through the same `paraOpts{PageBreak: true}` path. No run break, no section break, no new XML | body |
| The horizontal rule goes. `rule()`, `ruleWidthPt`, `ruleColor` and the `rule_grey` palette key leave with it, because nothing else reads them. No other route into a document draws a line; `read` keeps printing a document's own rule as `[rule]` | body, house |
| A break before nothing is nothing. A thematic break that is the last block of the body, or that only footnote definitions follow, is left out with no warning. That is the line `read` and `export` write before the notes, so an exported note publishes again without a blank last page | body |
| A break before a break is one break. A thematic break standing directly before the first body heading, when `first_heading_page_break` is on, is left out with no warning, because that heading already opens a page. Two thematic breaks in a row are one. Nothing else is collapsed: a break between two paragraphs is always honoured, however many stand in the note | body |
| `Counts` gains `PageBreaks`, the breaks the body carries after the two rules above, and both envelopes print it as `page_breaks`. The binary counts and judges nothing; the publish skill says the number when it is above zero and goes on | body, cmd, skill |
| `read` is unchanged. A page break still prints `[page break]` where it stands, a horizontal rule `[rule]`, and the footnote separator stays `---` | view |
| `export` writes `---` for a page break that is a paragraph of its own, as a line of its own between blank lines. It leaves out the break on the first body heading, because `build` puts that one back. A break inside a sentence keeps its `[page break]` marker and its warning, because no line of markdown says a page turns mid-sentence | export |
| The round-trip rule pairs the note's `---` with the document's `[page break]`, and the house's own breaks stay excused: the first heading's, and the column breaks | live |
| `propose --block` keeps refusing a thematic break in its content, and its refusal now says a page break. A suggestion that inserts a page break is a different request kind and is not built here | propose |
| The guide, the help and the skills say `---` is a page break in one line each, and say that a document holding a horizontal rule exports it as `[rule]`, which is a marker to resolve | docs, skills |

## What is measured before export is trusted

The docx route needs no gate: Drive's import of `w:pageBreakBefore` is already
the way the front matter's three breaks reach a Doc, and the live round trip
has excused their `[page break]` lines since M13. What is not measured is the
shape `read` gives them: whether a break paragraph of its own comes back as a
paragraph holding one `pageBreak` element and nothing else, and whether the
first heading's break comes back inside the heading paragraph or as a
paragraph before it. The export rule above assumes the first for a body
break and reads either for the heading's. Task 4 publishes a note holding
both and records what `read` printed in MEASURED.md. If a body break comes
back in any other shape, the export task stops and says so, and Nail
decides the rule.

## What this milestone does not do

- No syntax for a horizontal rule in a note. A document's own rule still
  exports as `[rule]`, to be resolved by hand.
- No `insertPageBreak` in `propose`, in `annotate` or in `restyle`'s body
  pass. The house's own breaks in the prelude are untouched.
- No page break inside a footnote or a block quote: goldmark parses a
  thematic break there as a block of its own and the walker warns and leaves
  it out the way it does a picture in a footnote. A test pins each. The table
  cell this sentence also named needs no rule: Task 3 measured a pipe table's
  cell as inline text, in which goldmark builds no thematic break at all, so
  there is nothing for the walker to warn about. A test pins that too.
- No house value. The break is the author's, not the template's, so
  `house.yaml` loses a key and gains none.
- No column break.

## Acceptance

Each item is run in the task log and ticked; the live ones carry the
document id.

1. A note with `---` between two paragraphs builds a docx in which the
   second paragraph opens a new page. The kitchen-sink golden shows two
   `w:pageBreakBefore` paragraphs where its two rules were, and no `w:pBdr`.
2. That note published to Drive shows the break where `---` stood, and
   `read` prints `[page break]` there. Live.
3. A `---` at the end of a note, and one before its footnote definitions,
   adds no page: the golden holds no break after the last paragraph.
4. `---` directly before the first heading adds no second break.
5. The build and publish envelopes carry `page_breaks`, and the number is the
   breaks the document got, not the lines the note held.
6. `export` of the document from item 2 writes `---` where the break stands
   and no `---` before the first heading; built again, the docx holds the same
   number of `w:pageBreakBefore` paragraphs. Live.
7. `read` of a document holding a horizontal rule still prints `[rule]`, and
   `export` keeps the marker and warns.
8. `make test`, `make vet` and `make dist` green, and the live round trip
   green with the new pairing.
