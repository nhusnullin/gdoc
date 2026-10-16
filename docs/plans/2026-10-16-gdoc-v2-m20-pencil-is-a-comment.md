# gdoc v2 Milestone 20: pencil is a comment, plan

2026-10-16. The task list for the specification beside it,
`docs/plans/2026-10-16-gdoc-v2-m20-pencil-is-a-comment-spec.md`, whose one
decision is Nail's and whose design rules are listed there for him to strike.
One commit per task. The spec holds the rules. This plan repeats none of
them. Where the code, the spec or this plan disagree, the task stops and says
so rather than choosing. It starts from `main` after M19 merged (`dde7495`).

## Principles

Serves 4 and strains 1 in one place, as the spec says.

## Context (from discovery)

- `go/internal/body/mark.go`: the span is `spanOpenParser` and
  `spanCloseParser` at priority 150, the `Span` node with its `Syntax`, the
  `spanOpen` placeholder, `spanAhead`, `checkSpans` with its six refusals,
  `spanWritten`, `spanClosed`, `spanAttribute`, `pencilAttribute` and
  `spanCloser`. `markTable.span` looks the mark up by `Syntax`. `isMark` and
  `countMarks` name `*Span` beside `*Highlight`. `Mark` is the extender.
- `go/internal/body/inline.go:201`: `case *ast.RawHTML, *ast.HTMLBlock:
  continue`, so a comment inside a sentence is dropped today.
  `body.go:829` is the `*ast.HTMLBlock` arm of `block`; `body.go:1110`
  `warnRawHTML` and `:1147` `hasRawHTML` warn by line on inline HTML in a
  paragraph, a heading (`:955`), a list item (`:1019`), a cell (`:1169`) and
  a footnote (`footnote.go:152`; `footnote.go:230` is a block of HTML inside
  one). `scan.go:335` skips an HTML block in both line scans.
- goldmark parses `<!-- -->` inside a sentence as `*ast.RawHTML` holding
  `Segments`, and on lines of its own as `*ast.HTMLBlock` of
  `HTMLBlockType2`, which may span lines and holds `Lines()`. It has no node
  for `%%`, so the Obsidian spelling needs an inline parser and a block
  parser of this package's own, built so both spellings reach one node.
- `go/internal/house/house.yaml:65`: `pencil.syntax` is `['.pencil']`,
  `kind: text`, `color: '#9A9A9A'`, `debt: false`. `house.go:128` lists the
  seven names, `:147` names `.pencil` in a comment.
  `TestThePencilSyntaxIsTheHouseStyles` pins the body constant to the file.
- `go/internal/export/marks.go:30` `Palette`: a text-colour mark is written
  as `Open: "["`, `Shut: "]{" + m.Syntax[0] + "}"`. `view.Mark`
  (`view/text.go:152`) carries `Open`, `Shut`, `Colour`, `Background`.
  `markedNotes` and `claimedColour` read the pencil hex for the footnote
  loss. Where a run's words would read as a mark's syntax is named in
  `view` (`TestWordsThatWouldReadAsAMarksSpanAreNamedAsALoss`,
  `TestAMarkWhoseWordsHoldAnotherMarksSyntaxComesHomeAsWords`).
- `go/internal/markers/markers.go`: `Pairs` holds four, `looked` seven
  literals. `TestMarkersNamesEachOpener` pins the list against `view`.
  `internal/body`, `internal/plaintext` and `cmd/gdoc`'s `propose --from`
  each ask `markers.Find`.
- `go/internal/view/doc.go:29` and `:49`: `<!-- tab t.0: Title -->` and
  `<!-- image: floating, kix.p1 -->`. `export/doc.go:25`: no file carries
  the tab line.
- `go/cmd/gdoc/commands.go:388`: the help line naming `[words]{.pencil}`.
  `export.go:58` comment. `help marks` prints the table off house.yaml.
- Docs and skills naming the span: `skills/gdoc-publish/SKILL.md:221`,
  `skills/gdoc-export/markers.md:31`, `docs/guide/exporting.md:74` and
  `:128`, `docs/guide/publishing.md:217-245` (the example block and the six
  refusals paragraph), `docs/backlog/restyle-keeps-pencil.md:24`,
  `docs/backlog/export-never-names-a-highlight-in-the-documents-own-words.md:13`
  and `:33`, `go/internal/live/doc.go:200` and `:216`, README (check).
- `go/internal/live/roundtrip_test.go`: the pairs table; the pencil pair
  reads `[words]{.pencil}` today. `TestLiveMarkImportProbe` measured the hex
  and is unchanged.
- Hub count on 2026-10-16: two HTML comments in two notes, no `%%`, no span.

- The first launch, 2026-10-16 15:03, stopped on its own Task 1 and reverted
  to a clean tree: dropping `.pencil` from house.yaml alone turns twelve tests
  red across nine task boundaries (eight in `internal/body`, because
  `markTable.span` looks the mark up by the syntax the author wrote; three in
  `internal/export`, because `Palette` writes `Shut` as `]{` + `Syntax[0]` +
  `}`; two in `go/boundary`, because both guide gates require every opener
  house.yaml states to appear in `publishing.md` and `exporting.md`). The
  reorder below keeps every commit green: the old spelling stays first in the
  file until export writes the comment, and the span goes after that.
  `.ralphex/progress/progress-gdoc-v2-m20-pencil-is-a-comment.txt` holds the
  failure list.

## Development Approach

TDD, one commit per task, the test first and failing. Every XML name and
every syntax a test states is a literal. `make test` keeps the race flag. A
task that finds the spec and the code disagree stops and says so in the task
log.

## Validation Commands

```bash
make test
make vet
cd go && go test ./internal/body -run 'Pencil|Comment|Span|RawHTML' -v
cd go && go test ./internal/export ./internal/markers ./cmd/gdoc -run 'Pencil|Comment|Marker|Marks' -v
cd go && GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1 go test ./internal/live -run 'RoundTrip' -v
```

## Progress Tracking

- [x] Task 1: an HTML comment inside a sentence is a pencil run, and the file states three spellings
- [x] Task 2: a comment on lines of its own is grey paragraphs
- [x] Task 3: the Obsidian spelling reaches the same node
- [x] Task 4: a pencil goes where a mark goes
- [x] Task 5: gdoc's own two comments are markers
- [x] Task 6: export writes a pencil home as an HTML comment, and the comment spelling comes first
- [x] Task 7: the span goes, `]{.pencil}` is refused by name, and the file states two spellings
- [ ] Task 8: the live round trip
- [ ] Task 9: the words in the guide, the skills, the backlog and the README
- [ ] Task 10: the acceptance table

## Tasks

### Task 1: an HTML comment inside a sentence is a pencil run, and the file states three spellings

`TestAnHTMLCommentIsAPencilRun`: `a <!-- quiet --> b` builds a paragraph of
three runs, the middle one carrying the pencil mark and the words `quiet`,
no italic, and the paragraph's text around it unchanged.
`TestTheWordsInsideAPencilAreLiteral`: `<!-- **x** and ==y== -->` carries
the asterisks and the equals signs. `TestAnEmptyCommentIsLeftOut`: `<!-- -->`
and `<!---->` build the sentence with no run for them and no warning. The
`*ast.RawHTML` arm of `inline.go` becomes a pencil run when the raw HTML is
a comment and keeps its `warnRawHTML` path otherwise
(`TestOtherRawHTMLIsStillWarnedAndLeftOut`, which also covers the block).
One space inside each delimiter is taken off the words, as `withoutOpener`
takes the highlight's.

In the same commit `house.yaml`'s `pencil.syntax` becomes
`['.pencil', '<!--', '%%']`, the old spelling still first so `Palette` keeps
writing the span and every span test stays green, and the `house.go` comment
says so. `TestThePencilSyntaxIsTheHouseStyles` reads the list rather than its
first element. `docs/guide/publishing.md` and `docs/guide/exporting.md` each
gain one line naming the two comment spellings, so the two boundary gates that
require every stated opener to appear in the guide pass. `gdoc help marks`
prints all three without a change of its own.

### Task 2: a comment on lines of its own is grey paragraphs

`TestACommentOnLinesOfItsOwnIsGreyParagraphs`: an `HTMLBlockType2` of three
lines, a blank line and two more lines builds two body paragraphs, each with
one pencil run, the first holding the three lines joined by single spaces.
The `*ast.HTMLBlock` arm of `block` does this for a comment and leaves every
other block type as it is. The empty case joins Task 1's test.

### Task 3: the Obsidian spelling reaches the same node

`TestAnObsidianCommentIsAPencilRun`: `a %% quiet %% b` builds what Task 1
built. The block form, `%%` alone on a line to `%%` alone on a line, builds
what Task 2 built. Design: the inline parser claims `%%` when a closing `%%`
sits on the same line and leaves a node the walk treats exactly as a comment
`RawHTML`; the block parser claims a line that is `%%` alone and runs to the
next. The cheapest way to keep one meaning is for both parsers to produce
goldmark's own node kinds, `RawHTML` and `HTMLBlock`, with the segments set
so the walk reads the words; if that cannot be done without reaching into
goldmark's unexported state, a `Pencil` node of this package's own is the
fallback and the comment arm wraps the same way. The task log says which.

### Task 4: a pencil goes where a mark goes

`TestAPencilGoesWhereAMarkGoes`, table-driven: refused by line in a heading,
in a picture's alt text and inside a highlight; carried in a table cell, a
list item and a block quote; left out and warned in a footnote the way a
footnote's own loss is warned today. `TestAHighlightInsideAPencilIsWords`:
`<!-- a ==b== c -->` is not refused and carries the words. `isMark` and
`checkMarks` learn the comment shape; `countMarks` is unchanged because the
pencil is no debt.

Ran 2026-10-16. The footnote clause above and the spec's own row read two ways,
and the run did not choose between them: both are true of a different shape.
A pencil inside a sentence inside a footnote is carried, which is the spec's
"carried out and home plain with the loss named", what a highlight and the old
span already do there, and what the milestone's one decision requires, since
leaving it out would change what the grey mark does rather than how it is
spelled; `internal/export` names that loss already. A comment on lines of its
own inside a footnote is left out and warned, which is this clause, because a
note under the page is a paragraph `footnoteBlocks` sets and the house style
has no grey paragraph of its own in one. It is named "a comment" rather than
"a block of HTML", for the reason the unclosed fence is. Both halves are in
`TestAPencilGoesWhereAMarkGoes`.

### Task 5: gdoc's own two comments are markers

`TestMarkersNamesEachOpener` grows by `{"<!-- tab ", "-->"}` and
`{"<!-- image: ", "-->"}`, and `TestGdocsOwnCommentsAreMarkersAndNotPencil`
in `internal/body` shows a note holding `<!-- image: floating, kix.p1 -->`
is refused by line as a marker and builds no grey run.
`TestBuildRefusesAMarkerByLine`, `TestPlaintextRefusesAMarker` and
`TestProposeRefusesAMarkerInTheFile` gain the two cases. `view`'s own
writing of the two is untouched; `skills/gdoc-export/markers.md` lists them
as markers in Task 9.

Ran 2026-10-16. The two openers join `Pairs` and `looked`; the `-->` closer
does not join `looked`, for the reason the bare `]` of `[s:ID]` does not: it
shuts every pencil an author writes, so looking for it would refuse a note
nothing is wrong with. Each opener always comes first, so no door is left
open. `TestAPencilIsNotAMarker` in `internal/markers` and
`TestAnAuthorsOwnCommentIsStillAPencil` in `internal/body` are that line from
both sides, beside the two the plan names. `make vet` and the raced suite are
green; `internal/live` timed out at 600s on the network with no assertion
failing, and this session has no `GDOC_LIVE_DOC_ID`, so the live round trip
is Task 8's.

### Task 6: export writes a pencil home as an HTML comment, and the comment spelling comes first

`TestAPencilRunComesHomeAsAnHTMLComment`: a run at #9A9A9A inside a sentence
writes `<!-- words -->` in place. `TestAGreyParagraphComesHomeAsACommentOfItsOwn`:
a paragraph whose only run is grey writes `<!-- words -->` on a line of its
own. `TestAPencilWhoseWordsHoldTheCloserIsNamedAsALoss`:
words holding `-->` come home plain with the loss named.
`TestFormattingOnAPencilIsDroppedAndNamed`: a bold grey run comes home as
`<!-- words -->` with the bold named as lost.
`TestTheObsidianSpellingIsNeverWritten` reads the palette. In the same
commit `house.yaml` reorders to `['<!--', '%%', '.pencil']`, `Palette` builds
the text-colour mark as `Open: m.Syntax[0] + " "`, `Shut: " -->"`, and the
`view` loss rule for words that would read as a mark's syntax learns `-->`
and forgets `]{`. The span tests in `internal/export` that read the old
`Shut` are rewritten here, because this is the commit that changes what
export writes; the span still parses on the way in until Task 7.

Ran 2026-10-16. Two words of the task read differently once the code was in
hand, and both are in the tests.

The bold on a grey run is not a fact gdoc has. `docs.Run` carries the
background, the text colour and the link, because `walk.go` decodes
`textStyle.backgroundColor` and `textStyle.foregroundColor` and nothing else,
so a bold pencil cannot be written as a test. The spec's own row is wider than
the plan's sentence, "any formatting other than the grey on a pencilled run is
dropped, loss named", and the one piece of formatting the read does carry on a
run that also carries the grey is the link.
`TestFormattingOnAPencilIsDroppedAndNamed` states that one: `<!-- the grey
link -->` comes home with the address dropped and named, because inside a
comment `[words](url)` is characters rather than a link. `view.Mark.Literal`
is the field that says so, and `emitter.literal` is the one route that writes
a mark's words with nothing of gdoc's around them.

The loss rule learns the opener rather than the closer. `-->` on its own in
the document's prose is the characters the next build reads, so naming it
would call an arrow in a caption a loss; `<!--` on its own is a loss every
way it falls, because it reaches the closer of the next pencil in the file or,
on a line of its own, swallows every line after it. So `spanAttributes`
became `commentOpeners`, `TestWordsThatWouldReadAsAMarksSpanAreNamedAsALoss`
became `TestWordsThatWouldReadAsAPencilsCommentAreNamedAsALoss`, and
`TestProseWithNoCommentSyntaxIsNotWarnedAboutOne` holds the arrow. The closer
is still refused inside a pencil's own words, which is `cannotCarry`'s one
rule for a literal mark and the spec's row.

Three of the span's refusals went with the span's syntax: a bracket, a caret
and a link no longer stop the mark being written, because a comment holds all
three as the characters they are. `holdsAnotherMark` now refuses a highlight
whose words hold `<!--`, which is a mark inside a mark, and asks nothing of a
pencil's own words, which is the spec's "a highlight inside a pencil is
words". `make vet`, gofmt and the raced suite are green over every package but
`internal/live`, which is Task 8's.

### Task 7: the span goes, `]{.pencil}` is refused by name, and the file states two spellings

`TestTheOldSpanIsRefusedByLineNamingTheComment`: `[words]{.pencil}` is
refused by line, and the refusal names `<!-- words -->`. Then delete both
span parsers, the `Span` and `spanOpen` nodes, `spanAhead`, `checkSpans`,
`spanWritten`, `spanClosed`, `spanAttribute`, `spanCloser` and
`pencilAttribute`, and every test under `TestASpan*`, `TestAnEmptySpan*`,
`TestAPencilAttribute*` and `TestAPencilSpanIsARunCarryingItsMark`. The new
refusal is one regular expression on the lines `checkLines` reads, beside
`checkFootnotes`. `house.yaml` drops `.pencil`, leaving `['<!--', '%%']`;
`TestThePencilSyntaxIsTheHouseStyles` states those two literals, and
`TestHelpMarksPrintsBothPencilSpellings` in `cmd/gdoc` reads them off
`help marks --json`. `doc.go`'s span section is rewritten as the comment
section, naming the tests above.

Ran 2026-10-16. The span is gone: both parsers, the `Span` and `spanOpen`
nodes, `spanAhead`, `checkSpans`, `spanLines`, `spanWritten`, `spanClosed`,
`spanAttribute`, `spanCloser`, `pencilAttribute` and the eleven tests the plan
names, with `markTable.span` and the `*Span` arms of `countMarks`, `isMark` and
`inline.go`. `checkOldPencilSpan` in `comment.go` is the one regular
expression left, built off `oldPencilSpelling`, the one place the old string is
still written down.

Three things the task found. The refusal names the words the author bracketed,
because `(?:\[([^\[\]]*))?\]\{\.pencil\}` captures them when the bracket
is on the line, so `[a note]{.pencil}` is answered with `<!-- a note -->`
rather than with a shape; a closer with no bracket in front of it is answered
with `<!-- words -->`. Both spellings are offered, because an author who wrote
the span once chose where the aside shows.

`TestAPencilAttributeInsideABlockOfHTMLRefusesNothing` went with the span, and
the rule it held did not: `checkLines` still leaves out a block of HTML, so
that case is a fourth row of
`TestTheOldSpanWhereNothingIsMarkdownIsNotRefused` beside the code span, the
fence and the indented block, and `scan.go` names it. What is not exempt is an
author writing `]{.pencil}` inside a comment of their own on a line of prose:
goldmark reads that comment as inline raw HTML, the line reaches `checkLines`,
and the note is refused. One regular expression on the prose lines is what the
plan asks for, the shape is a sentence about gdoc inside an aside about gdoc,
and the refusal names the fix, so it is left as it is.

Two fixtures carried the span and now carry a comment, which moved both
goldens. `01-kitchen-sink.md` took three tries: a comment that opens a line is
an HTML block even in the middle of a paragraph, and a line whose comment shuts
before the end of it is no comment at all, so the aside was warned about and
left out with the sentence's full stop. The pencil sits inside the line now.
`TestTwoMarksSideBySideStayTwoRuns` moved for the same reason and says so.
`internal/live/roundtrip_test.go`'s guard over the kitchen sink reads `<!--`
rather than `{.pencil}`, because this commit changed the note it reads;
`TestLiveRoundTrip` itself is Task 8's. `make vet`, gofmt and the raced suite
are green over the whole tree with the live variables unset, which is how this
session ran them: `internal/live` publishes the pencil pair and that pair is
Task 8's.

### Task 8: the live round trip

`TestLiveRoundTrip`'s pencil pair becomes `<!-- words -->` inline and a
comment on a line of its own, and the test publishes a note holding both and
reads both home. `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`. If the paragraph form
comes home as anything but a comment on a line of its own, stop and write
what came back into the task log for Nail.

### Task 9: the words in the guide, the skills, the backlog and the README

Every file the Context lists: one line saying a pencil is a comment in both
spellings, the example in `publishing.md` rewritten with one, the six
refusals paragraph cut to the ones that remain, `markers.md` listing the two
gdoc comments as markers and `<!-- words -->` as the pencil,
`restyle-keeps-pencil.md` and the other backlog item reworded, `live/doc.go`
reworded, the help line in `commands.go`. `go/boundary`'s
`TestThePublishingPageSaysAPencilIsAComment` pins the guide's line.

### Task 10: the acceptance table

One row per rule in the spec, pass or the test that says otherwise, written
into the spec file. Both plan files move to `docs/plans/completed/`.
