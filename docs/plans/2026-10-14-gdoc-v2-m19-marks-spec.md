# gdoc v2 Milestone 19: the marks, four highlights and one grey. Specification

2026-10-14. Every decision below is Nail's and settled, taken on the decision
page of the same day after a design pass by a UX designer, a typographer and a
compliance reader, and after reading how Nail's own v0.5 concept paper uses
colour. The task list is `docs/plans/2026-10-14-gdoc-v2-m19-marks.md`. It runs
after M18, footnotes, because both touch `internal/body`'s inline walk.

An author marks a span of a note with a name, and the name says what a reader
does with those words. gdoc carries the palette: the name, the colour, and
whether the mark is an authoring debt that must be gone before approval. gdoc
never knows who a mark is for. Nail's v0.5 used three colours for three
reviewers; under this milestone the person stays in his words, as it does
today, and the colour says what kind of thing the passage is.

## Principles

Serves: 1, two renderings of one text, because a mark the author wrote in the
note is in the document and comes home on export; 2, the binary prints facts
and the skills judge, because the binary counts marks and never says whether a
document with marks may be published.

Strains, each needing a DECISIONS.md entry before code:

- "Every change inside a handed-in document is a suggestion" is not touched.
  Marks are written by `build`, into a document that does not exist yet.
- The `==text==` syntax, which `body/mark.go` says is "the only way to say a
  human still has to fill this in", gains a second spelling and four siblings.

## Decisions taken

Nail, 2026-10-14, on the decision page:

| Rule | Where |
|---|---|
| The syntax is Pandoc's bracketed span, `[words]{.name}`, decision 3.1 A. `==words==` stays and means `{.sticky}` | body |
| Four highlights: `sticky` #FFD500, a word or figure to replace before sign-off; `proof` #4DD94D, a claim somebody confirms against a rule or a source; `flag` #FF4FA3, a question to one reviewer in the author's voice; `ink` #5CE1E6, text changed since the last version a reviewer read | house |
| One text colour: `pencil` #9A9A9A, prose that stays in the document but is not the argument. Chosen knowingly under the 4.5:1 accessibility floor, because it must read as secondary. No italic | house |
| No per-person logic anywhere, in the binary or in a skill. Who a mark is for is the author's words | all |
| `sticky`, `proof` and `flag` are authoring debts: counted on the envelope of `build` and `publish`, and the publish skill asks before a document carrying any goes to Drive, decision 3.3 A. `ink` and `pencil` are not counted: they are meant to reach the reader | cmd, skills |
| The binary is the one source of the palette. `gdoc help marks` prints the six names with their colours and their fate, and the skills read that rather than carrying a copy | cmd, skills |
| `coin`, orange, two options still open, was proposed and dropped. `added` and `removed` as text colours were dropped earlier the same day: a redline is a Docs suggestion | none |

Taken in the design pass and not contested:

| Rule | Where |
|---|---|
| A name the palette does not carry is refused by line, naming the six. A span inside a span, an empty span, and a span crossing a paragraph are refused by line | body |
| A mark inside a heading is refused by line, because Word copies the run into the contents field | body |
| In the docx each mark is a named character style, `Sticky`, `Proof`, `Flag`, `Ink`, `Pencil`, so Word lists them and a reader finds every sticky with one search. Google Docs has no character styles, so after import a mark is direct formatting, and that is what export reads | render, export |
| The four highlight colours are run shading with the exact hex, not Word's sixteen named highlights, which have no #FFD500 and no #FF4FA3. Whether Drive imports run shading as `backgroundColor` with the hex intact is measured first; if it does not, Task 2 stops the plan and the fallback is Nail's decision | render, live |
| `export` recognises a mark by its exact colour and writes `[words]{.name}` back. A background or text colour that is not in the palette is left as plain words and counted on the envelope by hex, because a colour gdoc did not write is a fact about the document, not a mark | export |
| `read` is unchanged in this milestone: the words of a flag reach a reviewing session, the mark around them does not. Printing marks in the projection changes what a session quotes, and is a decision for a later entry | view |
| `restyle` is unchanged. Whether an in-place restyle leaves run shading alone is measured in Task 2 and recorded | restyle |
| The cover's own yellow placeholders are the prelude's and keep writing Word's named yellow. They are not marks | render |

## What is measured before anything is trusted

Task 2 uploads a docx carrying one run with shading #FFD500, one with #4DD94D,
one with #FF4FA3, one with #5CE1E6, one with text colour #9A9A9A, and one with
Word's named yellow highlight for comparison, reads the document back bare,
and prints the `backgroundColor` and `foregroundColor` of each run as Docs
reports them. The answer goes into MEASURED.md either way. If a hex came back
changed, the plan stops after Task 2 and says what came back: export cannot
recognise a colour Drive rewrote, and which colours to use instead is Nail's
decision.

## What this milestone does not do

- No mark on a whole paragraph or a table cell. A mark is a span of words.
- No per-person legend, no owner field, no argument on a mark.
- No change to `propose` and `annotate`: a quote matches words, and the words
  are the same with or without the mark around them.
- No mark in `read`'s projection.

## Acceptance

1. `gdoc build` on a note carrying all six marks writes a docx whose
   `word/styles.xml` carries the five character styles and whose body carries
   each span in its style with the exact shading or text colour; `==T+1==` and
   `[T+1]{.sticky}` produce the same run.
2. An unknown name, a nested span, an empty span and a span in a heading are
   each refused by line, and nothing is written.
3. The `build` and `publish` envelopes carry `marks` with a count per name;
   `sticky`, `proof` and `flag` only, and zero when there are none.
4. `gdoc help marks` prints the six names, their colours and their fate, and
   `skills/gdoc-publish/SKILL.md` tells the session to run it and to ask
   before publishing a note whose debts are not zero.
5. `TestLivePublishCarriesTheMarksIntoDocs`: the published document's bare
   read reports each span's colour as the palette states it.
6. `TestLiveExportRoundTrip` over a note with all six marks passes: every mark
   comes home as the span it left as.
7. The six goldens render, the drift gate reads 171 rows with no new
   difference, and `make test` is green with the race flag.
