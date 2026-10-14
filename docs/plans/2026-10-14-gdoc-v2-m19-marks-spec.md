# gdoc v2 Milestone 19: the marks, Obsidian's colours and one grey. Specification

2026-10-14. Every decision below is Nail's and settled, taken on the decision
page of the same day after a design pass by a UX designer, a typographer and a
compliance reader, after reading how Nail's own v0.5 concept paper uses
colour, and after finding that Obsidian 1.14 (2 September 2026) carries
colour highlights of its own. The task list is
`docs/plans/2026-10-14-gdoc-v2-m19-marks.md`. It runs after M18, footnotes,
because both touch `internal/body`'s inline walk.

An author marks words in a note, and the mark says what a reader does with
them. The note carries only syntax Obsidian renders by itself: a highlight,
with one of Obsidian's six colour emoji at its start. gdoc carries the palette
in house.yaml under Obsidian's own colour names: what each becomes in the
Doc, and whether it is an authoring debt that must be gone before approval.
No name of gdoc's own: what a colour means is Nail's, said in his notes and
his skills, never in a key. gdoc never
knows who a mark is for. Nail's v0.5 used three colours for three reviewers;
the person stays in his words.

## Principles

Serves: 1, two renderings of one text, because a mark the author wrote in the
note is in the document and comes home on export; 2, the binary prints facts
and the skills judge, because the binary counts marks and never says whether a
document with marks may be published.

Strains, each needing a DECISIONS.md entry before code:

- The `==text==` syntax, which `body/mark.go` says is "the only way to say a
  human still has to fill this in", becomes one of six colours with six
  meanings.
- `body` warns about inline HTML and drops it, and it has no span syntax. The
  one span this milestone adds is Pandoc's, for pencil alone.

## Decisions taken

Nail, 2026-10-14, on the decision page:

| Rule | Where |
|---|---|
| Highlights are written the way Obsidian 1.14 writes them and no other way: `==words==` for the default yellow, and `==🔴words==`, `==🟠words==`, `==🟡words==`, `==🟢words==`, `==🔵words==`, `==🟣words==` for the six colours. No Pandoc span and no HTML for a highlight | body |
| The six are keyed and named as Obsidian names them: `yellow`, `red`, `orange`, `green`, `blue`, `purple`. No name of gdoc's own. Nail's own use, written on the decision page and not into the binary: yellow a word or figure to replace, green a claim to confirm, purple a question to one reviewer, blue text changed since the last version. Red and orange are carried without a meaning, so a note that uses them is not refused | house |
| The one grey, `pencil`, is prose that stays in the document but is not the argument. Obsidian has no text colour, so pencil is written as Pandoc's bracketed span, `[words]{.pencil}`, the only span gdoc reads. In the Doc it is text colour #9A9A9A, light grey, chosen knowingly under the 4.5:1 accessibility floor. No italic | body, house |
| No per-person logic anywhere, in the binary or in a skill. Who a mark is for is the author's words | all |
| `yellow`, `green` and `purple` are authoring debts: counted on the envelope of `build` and `publish`, and the publish skill asks before a document carrying any goes to Drive. `blue`, `red`, `orange` and `pencil` are not counted: they are meant to reach the reader | cmd, skills |
| house.yaml is the one source of the palette, keyed by the emoji. `gdoc help marks` prints the seven with their syntax, their Doc colour and their fate, and the skills read that rather than carrying a copy | cmd, skills |
| The Doc hexes are a proposal until Task 2 measures what Drive keeps: yellow #FFD500, green #3DD67A, purple #BA92FF, blue #5DB3FF, red #FF6B6B, orange #FFB85C, pencil #9A9A9A | house, live |
| Dropped on the way: five named text colours, a per-person legend, the writer's desk names sticky, proof, flag and ink, `coin`, Hilo and every other plugin, an HTML span for pencil, a rule that colours a paragraph by its first words | none |

Taken in the design pass and not contested:

| Rule | Where |
|---|---|
| A highlight carrying anything other than one of the six emoji at its start, or an emoji anywhere else in it, is the words as written. A pencil span inside a highlight, a highlight inside a pencil span, an empty span and a span crossing a paragraph are refused by line | body |
| A mark inside a heading is refused by line, because Word copies the run into the contents field | body |
| In the docx each mark is a named character style, `Yellow`, `Green`, `Purple`, `Blue`, `Red`, `Orange`, `Pencil`, so Word lists them and a reader finds every yellow with one search. Google Docs has no character styles, so after import a mark is direct formatting, and that is what export reads | render, export |
| The six highlight colours are run shading with the exact hex, not Word's sixteen named highlights, which carry none of these hexes. Whether Drive imports run shading as `backgroundColor` with the hex intact is measured first; if it does not, Task 2 stops the plan | render, live |
| `export` recognises a mark by its exact colour and writes Obsidian's syntax back, or the pencil span. A colour that is not in the palette is left as plain words and counted on the envelope by hex, because a colour gdoc did not write is a fact about the document, not a mark | export |
| `read` is unchanged in this milestone: the words of a purple question reach a reviewing session, the mark around them does not. Printing marks in the projection changes what a session quotes, and is a decision for a later entry | view |
| `restyle` is unchanged. Whether an in-place restyle leaves run shading and run colour alone is measured in Task 2 | restyle |
| The cover's own yellow placeholders are the prelude's and keep writing Word's named yellow. They are not marks | render |

## What is measured before anything is trusted

Task 2 uploads a docx carrying one run with shading for each of the six
highlight hexes, one with text colour #9A9A9A, and one with Word's named
yellow highlight for comparison, reads the document back bare, and prints the
`backgroundColor` and `foregroundColor` of each run as Docs reports them. Then
one in-place restyle over the document, and the eight read again. The answers
go into MEASURED.md either way. If a hex came back changed, or the restyle
cleared one, the plan stops after Task 2 and says what came back.

## House values that change in this milestone

Nail will change style values during this milestone, size and alignment among
them. The mechanism is fixed and the values are not: each change is a
DECISIONS.md line, a house.yaml line, a literal test re-stated, and a `Known`
entry where the master states the old value. The master docx stays as
provenance and the gate is never loosened, only re-pinned with the reason
written down. Task 10 of the plan holds the steps.

## What this milestone does not do

- No mark on a whole paragraph or a table cell. A mark is a run of words.
- No per-person legend, no owner field, no argument on a mark.
- No change to `propose` and `annotate`: a quote matches words, and the words
  are the same with or without the mark around them.
- No mark in `read`'s projection.
- No meaning for any colour in the binary. All six render, round-trip and are counted or not, and that is all.

## Acceptance

1. `gdoc build` on a note carrying all seven marks writes a docx whose
   `word/styles.xml` carries the seven character styles and whose body
   carries each run in its style with the exact shading or text colour;
   `==T+1==` and `==🟡T+1==` produce the same run.
2. A nested mark, an empty span, a mark in a heading and a span crossing a
   paragraph are each refused by line, and nothing is written.
3. The `build` and `publish` envelopes carry `marks` with a count per name
   for `yellow`, `green` and `purple` only, zero included.
4. `gdoc help marks` prints the seven, their syntax, their Doc colour and
   their fate, and `skills/gdoc-publish/SKILL.md` tells the session to run it
   and to ask before publishing a note whose debts are not zero.
5. `TestLivePublishCarriesTheMarksIntoDocs`: the published document's bare
   read reports each run's colour as the palette states it.
6. `TestLiveExportRoundTrip` over a note with all seven marks passes: every
   mark comes home as the syntax it left as.
7. The six goldens render, the drift gate reads 171 rows with no new
   difference, and `make test` is green with the race flag.
