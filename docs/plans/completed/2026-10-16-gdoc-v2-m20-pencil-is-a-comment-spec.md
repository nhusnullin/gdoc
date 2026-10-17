# gdoc v2 Milestone 20: pencil is a comment. Specification

2026-10-16. One decision, Nail's, taken in chat the same day: the grey mark
keeps everything it does and changes its spelling. A pencil is written as a
comment, HTML's `<!-- words -->` or Obsidian's `%% words %%`, and Pandoc's
span `[words]{.pencil}` is no longer read. The decision entry is in
`docs/v2/DECISIONS.md` under the same date, and the task list is
`docs/plans/2026-10-16-gdoc-v2-m20-pencil-is-a-comment.md`. It starts from
`main` after M19 merged (`dde7495`).

Today a pencil is the one Pandoc span gdoc reads, built in M18 two days ago.
In Obsidian it renders as its own brackets and braces, and six of M18's
refusals exist because a bracket is also a link's, a picture's and a footnote
mark's. After this milestone the author writes a comment, which Obsidian dims
while editing and hides while reading, and the words reach the Doc grey at
#9A9A9A as they do today.

## Principles

Serves: 4, every word costs a reader's attention, because an aside the hub's
reader never pays for still reaches the Doc's reader, grey, where it is
cheapest; and the round trip, because the only words a comment cannot hold
are its own closer, where the span could not hold a bracket, a link or a
highlight.

Strains: 1, two renderings of one text, in one place. The hub's reading view
hides what the Doc shows grey. Nail accepted that on 2026-10-16, knowing it
turns the pencil from prose both readers see into an aside for the Doc's
reader. The hub holds two HTML comments today, both asides of that shape.

## Decisions taken

Nail, 2026-10-16, in chat:

| Rule | Where |
|---|---|
| A pencil is a comment: `<!-- words -->` or `%% words %%`. The span `[words]{.pencil}` is no longer read | body, export, house |

Taken in the design pass by Claude, each one a sentence Nail can strike:

| Rule | Where |
|---|---|
| Two spellings, one mark. house.yaml states both under `pencil.syntax`, HTML first; `help marks` prints both; `export` writes the first and never Obsidian's | house, export, cmd |
| Two shapes. A comment inside a sentence is a grey run in that paragraph. A comment on lines of its own is grey paragraphs of its own, one per blank-line-separated stretch, the lines within a stretch joined by one space | body |
| The words are literal. Nothing inside a comment is markdown: `**x**` and `==x==` reach the Doc as typed, grey. A highlight inside a pencil is words and is not refused; a pencil inside a highlight is a mark inside a mark and is | body |
| Empty is nothing. `<!-- -->`, `<!---->` and `%%%%` are left out with no warning | body |
| A pencil goes where a mark goes. Refused by line in a heading, in a picture's alt text and inside another mark. Carried in a table cell, a list item and a block quote. In a footnote, carried out and home plain with the loss named. In code, code | body, export |
| The one word a pencil cannot hold is its closer. A grey run whose words hold `-->` comes home as the words alone, loss named. Any formatting other than the grey on a pencilled run is dropped, loss named | export |
| gdoc's own two comments are markers. `<!-- tab ` and `<!-- image: ` join `markers.Pairs`, so every route in refuses them by line and `export` keeps writing the image one | markers, body, export |
| The old span is refused by name. `]{.pencil}` on a line is refused naming the comment. One regular expression, removable in a later entry | body |
| Other raw HTML is unchanged: inline, warned and left out; a block that is not a comment, left out whole | body |
| The docx is unchanged: the `Pencil` character style, #9A9A9A, no italic. A grey paragraph of its own is a body paragraph whose one run carries it. No gate: Drive's import of the hex was measured in M18 and nothing new crosses the wire | body, render |
| `read` is unchanged. The restyle backlog item is unchanged in substance and reworded | view, backlog |
| What goes: both halves of the span parser, the `Span` node, `checkSpans` and its six refusals, `spanWritten`, `spanClosed` and the `.pencil` constant | body |
| The guide, the skills, the help line and the README say a pencil is a comment, in one line each, and show one | docs, skills, cmd |

## What is not built

Markdown inside a pencil. A pencil on a heading. The Obsidian spelling on the
way home. A comment in `propose --block` content, which stays raw HTML and
stays refused. A comment in a thread, which `plaintext` refuses as any markup.
A meaning for pencil in the binary.

## Measurement

None on the wire. The hex, the character style and Drive's import of both
were measured in M18 (`MEASURED.md`, "Drive's import of run shading and text
colour"). The one thing to confirm live is that the round trip still pairs: a
note holding `<!-- aside -->` inline and a comment on lines of its own
publishes, and `export` brings both home as `<!-- aside -->` in the same two
shapes. `TestLiveRoundTrip`'s pencil pair is that check. If a grey paragraph
of its own comes home as anything but a comment on a line of its own, the
export task stops and Nail decides the rule.

## Acceptance

Filled in on 2026-10-16, when the run finished. One row per rule above, in the
order the rules are stated, with the tests that hold it. `make test`, `make
vet` and gofmt are green over the whole tree, and the live round trip ran
against Drive.

| Rule | Verdict | Where it is pinned |
|---|---|---|
| A pencil is a comment, `<!-- words -->` or `%% words %%`; the span is no longer read | Pass | body `TestAnHTMLCommentIsAPencilRun`, `TestAnObsidianCommentIsAPencilRun`, `TestTheOldSpanIsRefusedByLineNamingTheComment`; export `TestAPencilRunComesHomeAsAnHTMLComment`; house `TestTheSevenMarksAreStatedAsLiterals` |
| Two spellings, one mark: HTML first in house.yaml, `help marks` prints both, export writes the first and never Obsidian's | Pass | house `TestTheSevenMarksAreStatedAsLiterals`; body `TestThePencilSyntaxIsTheHouseStyles`; cmd `TestHelpMarksPrintsBothPencilSpellings`; export `TestTheObsidianSpellingIsNeverWritten` |
| Two shapes: a run inside a sentence, grey paragraphs on lines of its own, one per stretch, the lines joined by one space | Pass | body `TestAnHTMLCommentIsAPencilRun`, `TestACommentOnLinesOfItsOwnIsGreyParagraphs`, `TestACommentOnOneLineOfItsOwnIsOneGreyParagraph`, `TestAnObsidianFenceIsGreyParagraphs`, `TestAnObsidianFenceInterruptsAParagraph`, `TestAnObsidianCommentOnOneLineOfItsOwnIsOneGreyParagraph`, `TestAnUnterminatedObsidianFenceIsNamedAndLeftOut` |
| The words are literal; a highlight inside a pencil is words, a pencil inside a highlight is refused | Pass | body `TestTheWordsInsideAPencilAreLiteral`, `TestTheWordsInsideAnObsidianCommentAreLiteral`, `TestAHighlightInsideAPencilIsWords`, the two `mark inside` rows of `TestAPencilGoesWhereAMarkGoes`; export `TestAPencilWhoseWordsHoldAHighlightComesHomeInTheComment`; live `TestTheComparisonLeavesAPencilsWordsAsTyped`, which runs offline |
| Empty is nothing: `<!-- -->`, `<!---->` and `%%%%` are left out with no warning | Pass | body `TestAnEmptyCommentIsLeftOut`, `TestAnEmptyObsidianCommentIsLeftOut` |
| A pencil goes where a mark goes: refused in a heading, in alt text and inside another mark; carried in a cell, an item and a quote; in a footnote out and home plain with the loss named; in code, code | Pass, the footnote read as two shapes, note 1 | body `TestAPencilGoesWhereAMarkGoes`, `TestAnHTMLCommentInCodeIsCode`, `TestAnObsidianCommentInCodeIsCode`; export `TestAPencilMarkInsideAFootnoteIsNamedAsALoss` |
| The one word a pencil cannot hold is its closer; any other formatting on a pencilled run is dropped, loss named | Pass, read two ways, notes 2 and 3 | export `TestAPencilWhoseWordsHoldTheCloserIsNamedAsALoss`, `TestFormattingOnAPencilIsDroppedAndNamed`, `TestWordsThatWouldReadAsAPencilsCommentAreNamedAsALoss`, `TestProseWithNoCommentSyntaxIsNotWarnedAboutOne` |
| gdoc's own two comments are markers; every route in refuses them and export keeps writing the image one | Pass | markers `TestMarkersNamesEachOpener`, `TestAPencilIsNotAMarker`; body `TestGdocsOwnCommentsAreMarkersAndNotPencil`, `TestAnAuthorsOwnCommentIsStillAPencil`, `TestBuildRefusesAMarkerByLine`; plaintext `TestPlaintextRefusesAMarker`; cmd `TestProposeRefusesAMarkerInTheFile`, `TestProposeRefusesAMarkerInABlockField`; view `TestAFloatingObjectIsAPlaceholderAndAWarning` |
| The old span is refused by name, one regular expression, removable later | Pass | body `TestTheOldSpanIsRefusedByLineNamingTheComment`, `TestTheOldSpanWhereNothingIsMarkdownIsNotRefused`. The expression is `checkOldPencilSpan` in `internal/body/comment.go`, built off `oldPencilSpelling`, which is the one place the old string is still written down |
| Other raw HTML is unchanged: inline warned and left out, a block that is not a comment left out whole | Pass | body `TestOtherRawHTMLIsStillWarnedAndLeftOut`, `TestInlineHTMLWarnsAndIsLeftOut`, `TestInlineHTMLInATableCellOrAQuoteWarnsToo` |
| The docx is unchanged: the `Pencil` style at #9A9A9A, a grey paragraph of its own is a body paragraph whose one run carries it, no gate | Pass | render `TestTheSevenCharacterStylesAreWritten`, `TestTheMarkStyleIDIsTheHouseKeyRaised`; body's own `pencil` helper inside `TestAPencilGoesWhereAMarkGoes` and `TestACommentOnLinesOfItsOwnIsGreyParagraphs`. Nothing new crosses the wire: `TestLiveMarkImportProbe` is unchanged since M18 |
| `read` is unchanged, and the restyle backlog item is reworded rather than changed | Pass | export `TestReadPrintsNoMark`; `internal/view` holds no change of this milestone's. `docs/backlog/restyle-keeps-pencil.md` carries no test, because a backlog file has none |
| What goes: both span parsers, the `Span` node, `checkSpans` and its six refusals, `spanWritten`, `spanClosed` and the `.pencil` constant | Pass | No test names an absence. `grep -rn 'spanAhead\|checkSpans\|spanWritten\|spanClosed\|pencilAttribute\|spanCloser\|spanOpenParser' go/` is empty, and house.yaml states the two comment spellings alone: house `TestTheSevenMarksAreStatedAsLiterals` |
| The guide, the skills, the help line and the README say a pencil is a comment, one line each, and show one | Pass | boundary `TestThePublishingPageSaysAPencilIsAComment`, `TestTheExportPageSaysWhatComesHome`, `TestTheBuildPageShowsAFootnoteAndTheSevenMarks`; cmd `TestHelpBuildNamesFootnotesAndMarks` and `testdata/pipe/help.stdout.golden`. The skills and the backlog carry no test and the files are the record. The README names no mark syntax at all, so nothing moved there |

The measurement is done. `TestLiveExportRoundTrip` ran on 2026-10-16 against
Drive with `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1` and the test folder, over a
`01-kitchen-sink.md` carrying a pencil inside a sentence and a comment on lines
of its own. Both came home in the shape they went out in, so no rule went back
to Nail. `TestTheRoundTripNoteCarriesAFootnoteAndEveryMark` pins both shapes in
that note offline, so a note holding one shape cannot quietly leave the other
unmeasured.

Three words of the plan read differently once the code was in hand. Each is
written out in the task it belongs to and named here.

1. A footnote holds two shapes, not one. A pencil inside a sentence inside a
   footnote is carried, which is this file's row and what a highlight already
   does there. A comment on lines of its own inside a footnote is left out and
   warned, because `footnoteBlocks` sets paragraphs and the house style has no
   grey paragraph under the page. Both halves are in
   `TestAPencilGoesWhereAMarkGoes`.
2. The formatting a pencil can lose is the link, not the bold. `docs.Run`
   carries the background, the text colour and the link and nothing else, so a
   bold pencil is not a fact gdoc has and cannot be written as a test.
   `TestFormattingOnAPencilIsDroppedAndNamed` states the link.
3. The loss rule learns the opener rather than the closer. `-->` alone in the
   document's prose is the characters the next build reads, so naming it would
   call an arrow in a caption a loss. `<!--` alone is a loss every way it
   falls. The closer is still refused inside a pencil's own words, which is the
   row above.

What is still red is not this milestone's. The live round trip reports eight
unnamed differences in `01-kitchen-sink.md`, five in `02-pictures.md` and
fourteen in `05-edge-cases.md`, and no comment is in any of them: an ordered
list starting at 7, a block quote wrapped over three lines, a task box, an
ampersand and angle brackets in a heading, the data-URI definition and a bare
URL. The same eight came back with the kitchen sink note reverted to HEAD, so
they predate M20. Naming one is a decision Nail writes down with its reason.
