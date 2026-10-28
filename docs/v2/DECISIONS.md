# gdoc v2 decisions

Dated, replaceable. Each names the principle it serves, or says it is a domain
choice with none above it. Same convention as [PRINCIPLES.md](../../PRINCIPLES.md).

v2 is a rewrite in Go. The three principles carry over unchanged. What changes is
which decisions implement them. This file holds their whole register and the
entries dated 2026-10-16 and later; earlier entries are frozen in
[DECISIONS-archive.md](DECISIONS-archive.md).

## The register

Every decision in this file, in date order, with its status stated rather than
inferred. Status is one of `holds`, `rejected`, `superseded YYYY-MM-DD (what
replaced it)`, or `measured`. Nothing else.

| Date | Decision | Status | Entry |
|---|---|---|---|
| 2026-08-13 | The markdown is the source, the Google Doc is a rendering | superseded 2026-09-19 (no side is the source by rule; the person decides per run) | DECISIONS-archive.md |
| 2026-08-14 | Skills are symlinked into `~/.claude/skills/`, never copied | holds | DECISIONS-archive.md |
| 2026-08-15 | The client reaches only the files it was given | holds | DECISIONS-archive.md |
| 2026-08-18 | The OAuth client is shipped in git, and the client stays Internal | superseded 2026-09-16 (the secret is injected at build time; the client stays Internal) | DECISIONS-archive.md |
| 2026-08-18 | Nothing runs git, and the agent never commits | holds | DECISIONS-archive.md |
| 2026-08-29 | The next version of gdoc is written in Go | holds | DECISIONS-archive.md |
| 2026-08-29 | gdoc never replaces the body of a document that already exists | holds | DECISIONS-archive.md |
| 2026-08-29 | Never trust a success. Verify with a second, independent probe | holds | DECISIONS-archive.md |
| 2026-08-29 | gdoc marks its own changes by colour, never by highlight | superseded 2026-08-29 (the colour scheme is retired) | DECISIONS-archive.md |
| 2026-08-29 | A comment is a prompt. `align` is a separate question | holds | DECISIONS-archive.md |
| 2026-08-29 | The marker is the trigger. Always | holds | DECISIONS-archive.md |
| 2026-08-29 | gdoc judges whether to ask, rather than always asking | holds | DECISIONS-archive.md |
| 2026-08-29 | There is no `release` command, and gdoc does not export PDFs | holds | DECISIONS-archive.md |
| 2026-08-29 | Anchors are not banked | rejected | DECISIONS-archive.md |
| 2026-08-29 | The Developer Preview arrived, and a third of this file retires | holds | DECISIONS-archive.md |
| 2026-08-29 | gdoc proposes. It never accepts and never rejects | holds | DECISIONS-archive.md |
| 2026-08-29 | MCP does not replace the REST client | holds | DECISIONS-archive.md |
| 2026-08-29 | Tabs are out of scope, and gdoc defends against them | superseded 2026-09-19 (the reader half only: `export` writes one file per tab; every writer still refuses a document with more than one) | DECISIONS-archive.md |
| 2026-08-29 | Amendments from reading the documentation | holds | DECISIONS-archive.md |
| 2026-08-29 | gdoc can retract its own proposal, and must check that it landed | holds | DECISIONS-archive.md |
| 2026-08-29 | The house style is a config file. The .docx becomes provenance | holds | DECISIONS-archive.md |
| 2026-08-29 | The finishing checklist is for `restyle` only | superseded 2026-09-09 (what an in-place restyle changes: the list is reported, not written) | DECISIONS-archive.md |
| 2026-08-29 | `publish` builds a file. `restyle` has two modes | superseded 2026-09-11 (`restyle --new` is not built) | DECISIONS-archive.md |
| 2026-08-29 | A comment is answered with the hub open | holds | DECISIONS-archive.md |
| 2026-08-29 | Emoji reactions: still no API, preview or not | holds | DECISIONS-archive.md |
| 2026-08-29 | The colour scheme is retired, not forgotten. Its "no named ranges" bullet is superseded 2026-09-10 (the house template reaches a document as a suggestion), which gives gdoc one named range as its own marker | holds | DECISIONS-archive.md |
| 2026-08-29 | Alignment runs both ways, and it is not a command | holds | DECISIONS-archive.md |
| 2026-08-29 | The product is three things, and it matters which does what | holds | DECISIONS-archive.md |
| 2026-08-29 | Everything gdoc writes opens with 🤖 | superseded 2026-10-04 (the mark is a setting, ⁘ by default and possibly empty, and 🤖 stays a known mark; why a marker is needed holds) | DECISIONS-archive.md |
| 2026-08-29 | gdoc reads back what it proposed, before saying it worked | holds | DECISIONS-archive.md |
| 2026-08-29 | The preview may vanish, and the risk is accepted. Loudly | superseded 2026-10-02 (suggestions are generally available; a silent edit is still loud, through the read-back and the stop) | DECISIONS-archive.md |
| 2026-08-29 | The guard owns the transport, and every id carries a write level. Widened 2026-09-09 (M7 splits) by a third level, `LevelInPlace`, and 2026-10-17 by a third door, one asset that is not a document | holds | DECISIONS-archive.md |
| 2026-08-29 | Publish runs once. Everything after travels as suggestions | superseded 2026-09-19 (a note may be published more than once, and each publish is a new document) | DECISIONS-archive.md |
| 2026-08-29 | Live review is a session, not a service | holds | DECISIONS-archive.md |
| 2026-09-07 | A replace proposal cannot be fully withdrawn by a delete | holds | DECISIONS-archive.md |
| 2026-09-08 | v1's `gdoc: <id>` stays refused. Publish is the only writer of the block | superseded 2026-09-19 (the one-writer clause only: `export` writes the block too, and the block is a list of documents; the v1 refusal holds) | DECISIONS-archive.md |
| 2026-09-09 | Which styling requests land in place | measured | MEASURED-archive.md |
| 2026-09-09 | The guard carries `files.copy`, for one source | holds | DECISIONS-archive.md |
| 2026-09-09 | Seven paragraph elements were dropped at decode | holds | DECISIONS-archive.md |
| 2026-09-09 | M7 splits. The survey ships without the write | holds | DECISIONS-archive.md |
| 2026-09-09 | What an in-place restyle changes, and what it deliberately does not write | holds | DECISIONS-archive.md |
| 2026-09-10 | In-place styling preserves anchors and pending suggestions | measured | MEASURED-archive.md |
| 2026-09-10 | The house template reaches a document as a suggestion | holds | DECISIONS-archive.md |
| 2026-09-10 | A named range over a suggested insertion | measured | MEASURED-archive.md |
| 2026-09-10 | A table takes one index of its own at the end | measured | MEASURED-archive.md |
| 2026-09-11 | `restyle --new` is not built | holds | DECISIONS-archive.md |
| 2026-09-16 | Help is an answer, completion is a written file, and two skills learn the tool from the tool | superseded 2026-10-03 (its "what was rejected: an MCP server inside the binary" paragraph only; everything else in the entry holds) | DECISIONS-archive.md |
| 2026-09-16 | M8 is deferred to the backlog, and the release goes next | holds | DECISIONS-archive.md |
| 2026-09-16 | The release: `x.y.z` with a nightly, an updater on demand with one read-only guard door, skills as a Claude Code plugin | superseded 2026-09-18 (the "no check, no stamp file" clause only: `help` checks once a day) and 2026-10-17 (the assessment that the Altery logo is public: the tree carries none from then on); everything else in the entry holds | DECISIONS-archive.md |
| 2026-09-16 | The source repository is public, and the client secret is injected at build time | holds | DECISIONS-archive.md |
| 2026-09-16 | What the update door actually reaches: two asset hosts, one bigger page, and a version that is only ever a tag | holds | DECISIONS-archive.md |
| 2026-09-17 | There is no rc. The nightly is the pre-release channel | holds | DECISIONS-archive.md |
| 2026-09-17 | The plugin is named `altery`, and the marketplace stays `gdoc` | holds | DECISIONS-archive.md |
| 2026-09-18 | The wait polls every two seconds | holds | DECISIONS-archive.md |
| 2026-09-18 | The binary notices a release by itself, once a day from `help`, and the plugin carries the stable number | superseded 2026-10-03 (the one-command clause only: `gdoc mcp` asks unasked too, under the same grant, stamp and ceiling; everything else in the entry holds) | DECISIONS-archive.md |
| 2026-09-18 | annotate: a comment on quoted words, and nothing else | holds | DECISIONS-archive.md |
| 2026-09-18 | Five backlog items closed, and what each decided | superseded 2026-09-27 (the bookmark on every heading only: a heading carries one when a link names it; everything else in the entry holds) | DECISIONS-archive.md |
| 2026-09-19 | Export, and the `gdoc:` block as a list of documents | holds | DECISIONS-archive.md |
| 2026-09-19 | A heading link is goldmark's id. A file name is gdoc's own slug | holds | DECISIONS-archive.md |
| 2026-09-19 | The projection is measured against goldmark: a sub-list's indent and a heading's anchor | holds | DECISIONS-archive.md |
| 2026-09-27 | The skills keep to the Agent Skills specification: `needs` under `metadata`, a compatibility line, and a body budget | holds | DECISIONS-archive.md |
| 2026-09-27 | A heading carries a bookmark only when a link names it, and a link to another note is its words | superseded 2026-10-16 (the bookmark half only; a relative link is its words holds) | DECISIONS-archive.md |
| 2026-09-27 | `publish` marks its own cover, and a restyle never styles a marked span | holds | DECISIONS-archive.md |
| 2026-09-28 | `propose` gains a second kind, a block of whole paragraphs | holds | DECISIONS-archive.md |
| 2026-09-28 | A block refuses a renumbered list and a paragraph an object floats beside | holds | DECISIONS-archive.md |
| 2026-10-02 | propose without the probe, and no company domain in the repository | holds | DECISIONS-archive.md |
| 2026-10-03 | `gdoc mcp`: gdoc in Claude Desktop chat, as measured | superseded 2026-10-03 in two paragraphs and one clause only (the link hold and the trusted email domains, both dropped after the first run in Claude Desktop; and "No CLI command gains it" of `AllowAccountRead`, since `auth status` is the grant's second caller); and on 2026-10-04 the `robot_not_ours` fact, removed | DECISIONS-archive.md |
| 2026-10-03 | What the first run in Claude Desktop changed: no link hold, no trusted domains, a hold is not an error, an absent author domain, and no voice mode | holds | DECISIONS-archive.md |
| 2026-10-03 | A change the person approved is not a payload that landed | holds | DECISIONS-archive.md |
| 2026-10-03 | On a terminal, a help screen is for the person: Panels on stderr and nothing on stdout, and `--json` always prints the object | holds | DECISIONS-archive.md |
| 2026-10-04 | Settings and the mark in effect | superseded 2026-10-13 (the list of what a mark may not be, and what counts as a text opening with one, only; everything else in the entry holds) | DECISIONS-archive.md |
| 2026-10-04 | The token in the macOS Keychain, and releases signed on Nail's Mac (built in M17) | holds | DECISIONS-archive.md |
| 2026-10-13 | A blank mark is refused, and a word-like mark is read as a word | holds | DECISIONS-archive.md |
| 2026-10-14 | The body breathes: 12pt under a paragraph and under a list, 5pt around a cell | holds | DECISIONS-archive.md |
| 2026-10-14 | A footnote is carried on every route | holds | DECISIONS-archive.md |
| 2026-10-14 | The marks: Obsidian's six colours and one grey | superseded 2026-10-16 (the pencil's spelling, the span refusals and the "an HTML span for pencil" clause only; everything else in the entry holds) | DECISIONS-archive.md |
| 2026-10-15 | What the M18 review closed: a label defined twice, a heading no line scan sees, a mark inside a footnote, and the two questions the parse answers rather than the line | superseded 2026-10-16 (the span paragraphs only: the attribute counted against the spans the parse formed, and the words a span cannot hold; everything else in the entry holds) | DECISIONS-archive.md |
| 2026-10-16 | `---` is a page break | holds | DECISIONS.md |
| 2026-10-16 | Pencil is a comment: `<!-- words -->` or `%% words %%`, and the span goes | holds | DECISIONS.md |
| 2026-10-16 | What the M20 review closed: a comment with words after it on one line, two comments on one line, a comment the editor wrapped, the line a pencil's refusal names, an empty comment in a quote, and the four comments that are gdoc's own | holds | DECISIONS.md |
| 2026-10-16 | Every heading carries one hidden bookmark, because Drive's import makes it the heading id | holds | DECISIONS.md |
| 2026-10-17 | What the second M20 review closed: a comment the editor wrapped in Obsidian's spelling, a comment the note never closes, and the old span on the way home | holds | DECISIONS.md |
| 2026-10-17 | What the third M20 review closed: an opener with no closer inside a sentence, and the refusal that blamed an export the author never made | holds | DECISIONS.md |
| 2026-10-17 | What the fourth M20 review closed: an escaped opener refused as a comment the line never closes | holds | DECISIONS.md |
| 2026-10-17 | The logo lives in Drive, not in the tree | superseded 2026-10-26 (the pin on exact pixels only: a logo of any size fits by width; the Drive id, the cache and the refusals hold) | DECISIONS.md |
| 2026-10-17 | The presence: gdoc appends one fact line per run to a file it never reads, and a separate menu bar app reads it | holds | DECISIONS.md |
| 2026-10-17 | The panel: the presence as the screens draw it, the app runs six argv lists, a hold dies with the sign-in it was held under | holds | DECISIONS.md |
| 2026-10-18 | The app opens itself: gdoc opens the installed app when it is not running, and launch at login is the person's switch | holds | DECISIONS.md |
| 2026-10-18 | The second morning: the account in the header, the middle scrolls, DOCUMENTS by what was just made and by use, pins, a bright mark, the session a run came from | holds | DECISIONS.md |
| 2026-10-18 | The third look: nothing scrolls, DOCUMENTS is the pinned and the last five, the note is found and not configured, the update glyph has three states | holds | DECISIONS.md |
| 2026-10-26 | The fixes: a cover line and the footer keep their own look, a logo of any size fits by width, six points after the two top headings, Russian heading ids keep their letters, the guard refuses an empty leaf in a style mask, export keeps code and repeated headings apart | holds | DECISIONS.md |
| 2026-10-26 | The fourth look: every document on the panel opens on a click, a mark only for a failed run, names that survive, buttons that say what happened, a panel that stays open, a bounded hub | holds | DECISIONS.md |
| 2026-10-27 | Week one of whole jobs: a watch that wakes a session only for marked work, a short help, live review in its own session, and a stopwatch on every comment | holds | DECISIONS.md |
| 2026-10-28 | Four small fixes: the panel drops the published number, a cancelled update leaves the terminal wrapping, a backslash escape publishes as the character, and Tab completion follows the installed gdoc | holds | DECISIONS.md |
| 2026-10-28 | The skills say when they are older than gdoc, and the plugin route turns on its updates | holds | DECISIONS.md |

**An entry is never edited after this, except its status line.** A decision that
changes is a new entry, dated today, with a new row here, and the old entry's
status says what replaced it. That is what makes a status worth reading: it says
the entry below was looked at, rather than that nobody has been back to it.

The `Entry` column names the file that holds the entry. A row whose status is
`measured` has its text in the measured file that column names, under a heading
of the same name, because a measurement is what Google does, not a choice gdoc
made. [MEASURED.md](MEASURED.md) holds the index over both measured files.

## 2026-10-16. `---` is a page break.

Nail's decision, taken in chat on the date above after the alternatives were
put to him. A line of three hyphens on its own in a note's body is a page
break, on every route. Serves principle 1, two renderings of one text, because
a break the author wrote is in the document and comes home on export as the
line it left as; and principle 4, every word costs a reader's attention,
because a page holding nothing is a page the reader turns for nothing.

Until now the body walker turned a thematic break into a horizontal rule, a
paragraph with a grey bottom border, and nothing an author could write in a
note broke a page. The house file breaks the page in three places of its own,
after the cover, before the contents list and before the first body heading,
and the author could neither move one nor add one.

**The two alternatives, declined.** The HTML div,
`<div style="page-break-after: always;"></div>`, which Obsidian's own PDF
export honours: it reads as machinery in a note somebody else will edit.
Pandoc's `\newpage`, which Obsidian prints as text on the page, so the note
stops being readable where the break is. `---` is the line Obsidian, Pandoc
and CommonMark all already read as a thematic break, and it is the one line
`read` never printed as `---` in the first place, which is why giving it a
second meaning contradicts no standard.

**The count he was shown.** The hub holds 1,348 such lines across 206 notes
today, nearly all of them dividers written before a heading, and a note
carrying 216 dividers becomes a document of 216 pages. He chose `---` anyway.
The hub is his, and gdoc reports the number rather than arguing with it.

**Every spelling is the same break.** `---`, `***` and `___`, three or more,
with or without spaces between them. goldmark parses one node and the source
spelling is not read, so one meaning reaches the document however the author
typed it.

**The docx shape is the one the front matter already uses.** An empty
paragraph carrying `w:pageBreakBefore`, emitted through the same
`paraOpts{PageBreak: true}` path the house's own breaks go through. No run
break, no section break, no new XML, and Drive's import of that attribute has
been the way the front matter's three breaks reach a Doc since M13.

**The horizontal rule goes.** `rule()`, `ruleWidthPt`, `ruleColor` and the
`rule_grey` palette key leave with it, because nothing else reads them. No
other route into a document draws a line. A document that already holds a rule
is unaffected: `read` still prints it as `[rule]` and `export` keeps that
marker for somebody to resolve.

**A break before nothing is nothing.** A thematic break that is the last block
of the body, or that only footnote definitions follow, is left out, with no
warning. That second case is the line `read` and `export` both write before a
document's notes, so an exported note publishes again without a blank last
page. The author's intent is met either way, which is why there is nothing to
warn about.

**A break before a break is one break.** A thematic break standing directly
before the first body heading, while `first_heading_page_break` is on, is left
out, because that heading already opens a page. Two thematic breaks in a row
are one. Nothing else is collapsed: a break between two paragraphs is always
honoured, however many of them stand in the note.

**`Counts` gains `PageBreaks`**, the breaks the body carries after the two
drops above, and both envelopes print it as `page_breaks`. It is the breaks
the document got, not the lines the note held. The binary counts and judges
nothing; the publish skill says the number when it is above zero and goes on.

**`read` prints `[page break]` where the page turns, which it printed nothing
for since M13.** This sentence read "`read` is unchanged" until the Task 4
measurement, below, found that neither kind of break is a `pageBreak` element:
Drive's docx import states both as `pageBreakBefore` on the paragraph's own
style, which `internal/docs` did not decode. So the read decodes it, and a
paragraph the page breaks before prints `[page break]` on a line of its own in
front of it, through the same placeholder and with the same warning a native
break gets. A horizontal rule still prints `[rule]`, and the footnote
separator stays `---`.

`export` writes `---` for a page break that stands on its own, the flag's line
or a native element that is a body paragraph's only content, as a line of its
own between blank lines, and leaves out the break on the first body heading
because `build` puts that one back. A break inside a sentence keeps its
`[page break]` marker and its warning, because no line of markdown says a page
turns mid-sentence.

**Only a body paragraph's break is written as that line**, which this entry
said as "a paragraph's only content" until the review of the same day. A
native break that is all a heading, a list item or a table cell holds keeps the
placeholder and the warning instead, because the line would come back as
something else on the next build: `# ---` as a heading saying `---`, `1. ---`
as an item holding a break, and `---` in a cell as three hyphens of text, since
a pipe table's cell holds inline text and goldmark reads no thematic break in
one. The flag's line is written in front of the paragraph and carries no
prefix, so a heading and a list item take it; a cell does not, because nothing
inside a cell is a line of its own, and a cell paragraph carrying the flag
keeps the placeholder and the warning too. `internal/view`'s
`TestOnlyABodyParagraphsBreakIsWrittenAsTheCallersLine` and
`TestABreakFlaggedOnACellParagraphKeepsThePlaceholder` are the pins.

**No line is written beside a pending suggestion**, which the same review left
half done. The markers that say a span is pending are written around it, so the
line would go inside them or sit against them, and `---{++}[s:ID]` is read back
as prose rather than as a break. The refusal was on the break run alone, so a
paragraph holding a break and a newline somebody suggested still got the line
and lost the break with no warning. It is the whole paragraph now: a standalone
break is one where no run carries a suggestion id.
`TestABreakBesideASuggestionKeepsThePlaceholder` is the pin.

**What is not built here.** No syntax for a horizontal rule in a note. No
`insertPageBreak` in `propose`, in `annotate` or in `restyle`'s body pass, so
`propose --block` keeps refusing a thematic break in its content and its
refusal now says a page break. No page break inside a footnote definition or a
block quote: the walker warns and leaves it out the way it does a picture in a
footnote. A table cell needs no rule, because a pipe table's cell holds inline
text and goldmark builds no thematic break in one, which Task 3 measured after
the specification said the walker warns there. No house value, because the
break is the author's rather than the template's. No column break.

**What is measured.** The docx route needs no gate. What was not known is the
shape `read` gives the two kinds of break, so the milestone published a note
holding both and recorded in MEASURED.md what `read` printed, with the document
id. The export rule assumed a body break comes back as a paragraph holding one
`pageBreak` element and no text, and the gate said that if it came back as
anything else the export task stops and Nail decides the rule.

**It came back as anything else, and the gate did not hold.** Document
`1MmipMWaS96OmV3ONxFioNFmaKhBYHWb7w3jrrF90hic`, 2026-10-16: no `pageBreak`
element anywhere, and four paragraphs carrying `pageBreakBefore`. MEASURED.md
wrote down three ways out and Claude took the first of them, under Nail's
standing "do it for me" of the same day and in his absence, and said so: the
read decodes the flag. The other two, a docx run break and shipping without
the export half, are in MEASURED.md for Nail to reverse to.

**Tests.** `internal/body`'s `TestAThematicBreakIsAPageBreakParagraph`,
`TestEveryThematicBreakSpellingIsTheSameBreak`,
`TestAPageBreakBeforeNothingIsNothing`,
`TestTheFootnoteSeparatorIsNotAPageBreak`,
`TestABreakBeforeTheFirstHeadingIsTheHeadingsOwn`,
`TestTwoBreaksInARowAreOne`,
`TestABreakBetweenTwoParagraphsIsAlwaysHonoured`,
`TestABreakInsideAFootnoteOrAQuoteIsWarnedAndLeftOut` and
`TestThreeHyphensInACellAreNotABreak`;
`cmd/gdoc`'s `TestBuildAndPublishCarryThePageBreakCount`, the build and the
publish twin in one table;
`internal/docs`'s `TestAParagraphKeepsWhetherThePageBreaksBeforeIt` and
`TestTheStructurePrintCarriesThePageBreakFlag`; `internal/view`'s
`TestABreakBeforeAParagraphPrintsAsAPageBreakLine`,
`TestTheHousesFirstHeadingBreakPrintsBeforeTheHeading`,
`TestANativePageBreakElementStillPrintsWhereItStands` and
`TestACallerThatNamesALineForABreakGetsItAndNoWarning`;
`internal/export`'s `TestAPageBreakOfItsOwnExportsAsThreeHyphens`,
`TestTheFirstHeadingsBreakIsLeftOut`,
`TestAPageBreakInsideASentenceKeepsItsMarker` and
`TestAHorizontalRuleStillExportsAsItsMarker`; `internal/live`'s
`TestLivePageBreakShape` and `TestLiveRoundTrip`; and `go/boundary`'s
`TestThePublishingPageSaysAThematicBreakIsAPageBreak`.

## 2026-10-16. Pencil is a comment: `<!-- words -->` or `%% words %%`, and the span goes.

Nail's decision, taken in chat on the date above, two days after the marks
shipped in v2.12.0. The grey mark keeps everything it does and changes its
spelling. A pencil is written as a comment, HTML's `<!-- words -->` or
Obsidian's `%% words %%`, and Pandoc's bracketed span `[words]{.pencil}` is no
longer read. Supersedes the pencil half of the 2026-10-14 marks entry: the
spelling, the span refusals, and the clause declining "an HTML span for
pencil", which declined a span and not a comment. The colour, the character
style, the fate on the envelope and `read` are unchanged.

**Why.** In Obsidian the span renders as its own brackets and braces, and
most of what the M18 review closed was the bracket: a link inside a pencil,
`]{.pencil}` closing nothing, a highlight inside a span, a span the parse
never formed because a picture, a footnote mark or a link claimed the bracket
first. A comment has none of that. The only words a comment cannot hold are
its own closer. Obsidian dims a comment while the author edits and hides it
when anybody reads, Pandoc and CommonMark both parse the HTML spelling, and
gdoc's own two markers in a projection are already HTML comments.

**What this strains.** Principle 1, two renderings of one text, in one place:
the hub's reading view hides what the Doc shows grey. Nail accepted it, and it
changes what a pencil is. The 2026-10-14 entry called it prose that stays in
the document but is not the argument, visible in both renderings. It is now an
aside for the Doc's reader, which the note's reader skips entirely. The hub
was counted before he chose: two HTML comments in two notes, both asides of
exactly that shape, a `TODO(Nail)` and an `Add ideas below`, no `%%` and no
span. Both reach their documents grey from now on instead of being left out.

**Two spellings, one mark.** `<!-- words -->` and `%% words %%` are the same
pencil. house.yaml states both under the pencil's `syntax:`, HTML first, and
`gdoc help marks` prints both. `export` writes the first, because CommonMark
and Pandoc read it and Obsidian's is Obsidian's alone. The Obsidian spelling
is read and never written.

**Two shapes.** A comment inside a sentence is a grey run in that paragraph.
A comment on lines of its own is grey paragraphs of its own, one per
blank-line-separated stretch inside it, the lines within a stretch joined by
one space, which is how markdown reads prose.

**The words are literal.** Nothing inside a comment is markdown: `**x**`
reaches the Doc as asterisks, grey, and `==x==` as its equals signs. goldmark
hands a comment back as raw text, reading markdown inside it would be a second
parse to keep in step, and a pencil is an aside rather than typeset prose. So
a highlight inside a pencil is words and is not refused, and a pencil inside a
highlight is a mark inside a mark and is. `export` writes a grey run's words
and nothing else about it: a bold or a link somebody put on grey words in the
Doc is dropped, with the loss named the way a mark inside a mark is named
today, because the comment cannot carry it home.

**Empty is nothing.** `<!-- -->`, `<!---->` and `%%%%` carry no words and are
left out with no warning. The span's empty case was refused because its syntax
would have reached the document as words; a comment's syntax never does.

**A pencil goes where a mark goes.** Refused by line in a heading, in a
picture's alt text and inside another mark. Carried in a table cell, a list
item and a block quote. Inside a footnote it is carried out and comes home
plain with the loss named, as every mark does. Inside a code block or a code
span it is code. The one rule a highlight has that a pencil does not is the
colour emoji, which a comment never opens with.

**The one word a pencil cannot hold is its closer.** A grey run in the Doc
whose words hold `-->` comes home as the words alone with the loss named,
because the file has to build again. `%%` inside the words is fine, since the
HTML spelling is the one written.

**gdoc's own two comments are markers, not pencil.** `<!-- tab ` and
`<!-- image: ` join `markers.Pairs`, so `build` and `publish` refuse a note
holding one by line, the way they refuse `{+ +}` and `[[c: ]]`, and
`propose`, `reply` and `annotate` refuse one in their text. `export` keeps
writing the image comment for somebody to place the picture by hand. Today
that comment is a block of HTML left out with a warning; refusing it is what
"a marker travels out of a document and never back in" already said.

**The old span is refused by name.** `]{.pencil}` on a line is refused, naming
the comment, because the alternative is brackets and braces printed into a
document with the build saying ok. It is one regular expression. Nothing in
the hub holds one today, and the rule can go in a later entry.

**Other raw HTML is unchanged.** `<b>` inside a sentence is warned about by
line and left out, and a block of HTML that is not a comment is left out
whole, as since M13.

**The docx is unchanged.** The run carries the `Pencil` character style and
the exact hex #9A9A9A, no italic, under the 4.5:1 floor on purpose as the
2026-10-14 entry says. A grey paragraph of its own is a body paragraph whose
one run carries it. Drive's import of that hex was measured in M18 and nothing
new goes over the wire, so there is no gate. The in-place restyle still turns
a pencil black, `docs/backlog/restyle-keeps-pencil.md` holds the fix, and
only its wording changes.

**What goes.** The span parser, both halves, the `Span` node, `checkSpans`
and its six refusals, `spanWritten` and `spanClosed`, and the `.pencil`
constant. `internal/body` reads one inline extension of its own after this,
the highlight, and one of goldmark's, the comment.

**What is not built.** Markdown inside a pencil. A pencil on a heading. The
Obsidian spelling on the way home. A comment in `propose --block` content,
which stays raw HTML and stays refused. A comment in a thread, which
`plaintext` refuses as it refuses any markup. A meaning for pencil in the
binary.

**Tests.** `internal/body`'s `TestAnHTMLCommentIsAPencilRun`,
`TestAnObsidianCommentIsAPencilRun`,
`TestACommentOnLinesOfItsOwnIsGreyParagraphs`,
`TestTheWordsInsideAPencilAreLiteral`, `TestAnEmptyCommentIsLeftOut`,
`TestAPencilGoesWhereAMarkGoes`, `TestAHighlightInsideAPencilIsWords`,
`TestGdocsOwnCommentsAreMarkersAndNotPencil`,
`TestTheOldSpanIsRefusedByLineNamingTheComment`,
`TestOtherRawHTMLIsStillWarnedAndLeftOut` and the kept
`TestThePencilSyntaxIsTheHouseStyles`; `internal/markers`'s
`TestMarkersNamesEachOpener`, grown by two; `internal/export`'s
`TestAPencilRunComesHomeAsAnHTMLComment`,
`TestAGreyParagraphComesHomeAsACommentOfItsOwn`,
`TestAPencilWhoseWordsHoldTheCloserIsNamedAsALoss`,
`TestFormattingOnAPencilIsDroppedAndNamed` and
`TestTheObsidianSpellingIsNeverWritten`; `cmd/gdoc`'s
`TestHelpMarksPrintsBothPencilSpellings`; `internal/live`'s
`TestLiveRoundTrip`, whose pencil pair changes; and `go/boundary`'s
`TestThePublishingPageSaysAPencilIsAComment`.

*Status note, 2026-10-17.* The review widened `markers.Pairs` from two to
four: `read` prints four of gdoc's own comments (tab, `image:`, `drawing:`,
`object:`), where this entry and the spec say two. The rule is unchanged;
the count is.

## 2026-10-16. What the M20 review closed: a comment with words after it on one line, two comments on one line, a comment the editor wrapped, the line a pencil's refusal names, an empty comment in a quote, and the four comments that are gdoc's own.

Six corrections to the entry above, found by the external review of the
milestone the same day. Each is the rule that entry already states, applied
where it was not. Nothing here changes what a pencil is.

**A comment with the author's own words after it on one line is a pencil and
that prose.** CommonMark opens a block of HTML on any line beginning `<!--`
and gives the block the whole of the line holding the closer, so
`<!-- todo --> We ship in May.` was one block of HTML: named by line and left
out whole, with the sentence never reaching the document. Obsidian shows that
line as the sentence with the aside hidden, and `export` writes exactly this
shape whenever a paragraph opens with a grey run, so a document exported and
published again lost the paragraph. `internal/body` claims the line in front
of goldmark's own block parser and reads it as the paragraph it is: the
comment is the pencil, the rest is markdown. A comment alone on its line is
untouched and stays the second of the two shapes. The same line wrapped over
two, where the closer is inside a block of HTML, is still left out, and the
warning names it as a comment with words after its closer rather than as a
block of HTML.

**A pencil's line wraps the way any other paragraph's line wraps.** The claim
above opens a paragraph, and the first reading of it shut that paragraph on
every following line. goldmark treats an open paragraph as continuable, so it
skips the parser's Continue and offers the next line only to the parsers that
may interrupt a paragraph, and its own paragraph parser is not one of them:
nothing opened, the line was advanced past, and it reached no node at all.
`<!-- todo --> Revenue grew 10%.` with `Costs fell 5%.` under it published the
first line and lost the second without a word, which is worse than the loss it
was fixing, and a lazy line in a list item or a quote went the same way. The
line now continues the paragraph, which is also the reading Obsidian shows and
what lets a second comment the editor wrapped reach its closer.

**Two comments on one line are two pencils.** The words of an HTML comment end
at the first closer a reader meets, so a piece of text that opens with `<!--`,
ends with `-->` and holds another closer between them is never one comment.
Read as one, `<!-- first --> <!-- second -->` reached the document as the one
grey paragraph `first --> <!-- second`, with both delimiters printed as words,
and `<!-- a --> prose <!-- b -->` turned the author's black prose grey. The
reading refuses that shape, and the line itself is now two pencils and the
words between them, by the rule above. Obsidian's fence is not affected: it is
delimited by whole lines, so `%%` inside it is words.

**A comment the editor wrapped is one line in the document.** goldmark hands
an inline comment that spans two lines back with the newline inside it, and a
newline in a run is a hard line break in the published paragraph. The lines
are joined by one space, which is what the block form already does and what a
soft break in prose already means.

**A pencil's refusal names the pencil's own line.** A comment is the one
inline node that holds its characters as segments rather than as text
children, so a heading whose whole content is one carries no text at all and
the line lookup climbed past it to the first text in the note: `# <!-- tbd -->`
on line 5 sent the author to line 1. The span this spelling replaced had text
children and never showed it.

**An empty comment inside a block quote is nothing.** "Empty is nothing" is
checked where a paragraph is emitted, and a quote is the one path that emits
one without going through that check, so `> %%%%` left an indented blank line
the author never typed.

**gdoc's own comments are four, not two.** `<!-- tab ` and `<!-- image: ` were
named; the floating placeholder names the kind, so `<!-- drawing: ` and
`<!-- object: ` are written by the same line of code and are the same sort of
statement about a document. Left out of `markers.Pairs`, a drawing that floats
beside the text came home in the file and the next build published the grey
paragraph `drawing: floating, kix.p1` without a word. All four join the pairs,
and `internal/view`'s own test holds the two lists in step, because a label
this package does not know is a line the next build reads as the author's
aside.

**Two more on the way home, from the same reading.** A grey run whose words
would be written home as one of those markers comes home as its words with the
loss named, the way a mark that cannot carry its syntax already does. The
question is asked of the comment as it will be written, not of its opening:
`<!-- ` and the words `tab` give `<!-- tab`, which carries no marker, while
the line gdoc writes is `<!-- tab -->`, which carries one, and a marker in the
middle of the words is the same refused line. And the document's own `%%` is
named where the next build reads it as syntax: `export` writes HTML's spelling
and never Obsidian's, but the next build reads both, so `LIKE '%%abc%%'` came
home with no warning and published again with the words between the pairs grey
and the delimiters gone. Inside a pencil's own words it is not named, because
nothing inside a comment is markdown and the characters come home as
themselves. A footnote mark inside those words is still named: `build` refuses
a mark naming no note off the source line, and an aside in the middle of a
paragraph's line is on that line.

**Where they are pinned.** `internal/body`'s
`TestACommentWithProseAfterItIsAPencilAndThatProse`,
`TestTwoCommentsOnOneLineAreTwoPencils`,
`TestAWrappedCommentInsideASentenceIsOneLine`,
`TestAPencilsRefusalNamesItsOwnLine`,
`TestAnEmptyCommentInsideAQuoteIsLeftOut`,
`TestAPencilsLineWrapsLikeAnyOtherParagraph` and
`TestACommentsWordsCarryNoSpaceOnEitherEdge`; `internal/markers`'s
`TestMarkersNamesEachOpener`, `TestFindNamesTheFirstMarkerOnTheLine` and
`TestCommentReadsTheLineGdocIsAboutToWrite`;
`internal/view`'s `TestEveryFloatingPlaceholderIsAMarker`; and
`internal/export`'s `TestAPencilWhoseWordsWouldReadAsAMarkerComesHomeAsItsWords`,
`TestWordsHoldingTheObsidianSpellingAreNamed` and
`TestAGreyRunHoldingAFootnoteMarkIsStillNamed`.


## 2026-10-16. Every heading carries one hidden bookmark, because Drive's import makes it the heading id.

Found by Nail on 2026-10-16 in "Embedded retail card concept paper v0.9": after
a refresh of the contents list in Google Docs, every entry's hover card said
"Tab 1" and a click landed on the title page, not on the heading it names.
Every document gdoc has published has it, and the measurement is MEASURED.md,
2026-10-16, "Drive's import of a heading's bookmark". Serves principle 3,
because the publish read-back gains a check that catches this class of silent
failure, and principle 4, because a reader who clicks a contents entry should
land on the section and not on the cover.

**Every heading `render` writes carries one empty bookmark before its first
run, named `_h_<slug>`.** A contents entry is a link to a heading by its id,
and Drive's import mints that id for a heading paragraph only when the
paragraph carries a bookmark whose name opens with `_`. That is Google's own
docx export convention, and Word treats such a bookmark as hidden, so it shows
in no bookmark list there and no flag sits beside the heading in Docs. The
leading underscore is the whole trigger: the bookmark's range does not matter,
and the name is not kept, because Google mints its own `h.` id from it.
`bookmarkName` keeps its rule, letters, digits and underscores, at most 40
characters with a seven-digit hash past the ceiling, and opens with `_h_`. A
figure-only heading and a heading inside a footnote definition still carry
none, because neither emits a heading paragraph.

**A `#` link in the note anchors at that name**, so it comes back as a heading
link naming the heading's id rather than as a bookmark link, which
`internal/docs` already decodes and `internal/view` already prints as `#` and
the heading's words. The `h_` bookmark that wrapped a linked heading's runs
goes. The pre-walk still collects every heading id, so a dead `#` still warns
by line; the set of linked headings stops deciding anything and goes with it.

**`publish` counts what came back.** The read-back `publish` already makes
gains one fact, `headings_without_id`, and a warning when the count is not
zero. It is a fact and a warning rather than a failure, because the document is
there. That is the check that would have caught this a month ago.

**What this retires.** The first half of 2026-09-27, "a heading carries a
bookmark only when a link names it", including its reasoning that a note with
no internal link showed a flag on every heading for nothing: that held for a
visible bookmark, and a hidden one shows no flag. Its second half, a relative
link is its words, holds untouched. The 2026-09-18 entry, which the 2026-09-27
one superseded on this point, had a bookmark on every heading for a different
reason, that a note is edited after it is published; the reason here is the
heading id and nothing else.

**What was rejected.** Repairing a document already published: re-applying
`namedStyleType: HEADING_1` through `updateParagraphStyle` mints no id, the id
is minted at import and nowhere else, and the route for an existing document is
a fresh publish from its note. Nail's call of 2026-10-16 is that the documents
already out stay as they are. Refreshing the contents list from gdoc is not
rejected but blocked: the Docs API has no request that builds or refreshes one,
so the refresh stays a click in the UI, as the guide says.

Nail's gate, passed on 2026-10-16 on the variant f probe document
`118is97pMYpL14kgtolRM3JuBlFup5B8rTZD5Pr8Ah5s`: an entry names its heading and
opens it, no bookmark flag sits beside any heading, and the `#` jump lands.

Tests: `TestEveryHeadingCarriesAHiddenBookmark`,
`TestAnAnchorLinkIsAJumpAndNotARelationship`, `TestBookmarkNameIsWordSafe`,
`TestAFigureOnlyHeadingCarriesNoBookmark`,
`TestAHeadingInsideAFootnoteIsNoAnchor`,
`TestAPublishReportsHeadingsThatCameBackWithNoId`, and `internal/live`'s
`TestAPublishedHeadingCarriesAnIdAndAJumpIsAHeadingLink` under
`GDOC_LIVE_WRITE`. The plan is
`docs/plans/2026-10-16-gdoc-v2-m21-heading-ids.md`.

## 2026-10-17. What the second M20 review closed: a comment the editor wrapped in Obsidian's spelling, a comment the note never closes, and the old span on the way home.

Three corrections to the two entries above, found by the second review of the
milestone. Each is a rule those entries already state, applied where it was
not. Nothing here changes what a pencil is.

**A comment the author's editor wrapped is one pencil in both spellings.** The
entry above says a wrapped comment is one line in the document, and the HTML
spelling has always reached across the wrap because goldmark's own comment
parser walks forward to its closer. Obsidian's parser shut on the opening
line, so `%% an aside that` with `wraps over two lines %%` under it was
claimed by neither parser and published as the words the author typed,
delimiters and all, with the build saying ok. That is the one failure every
line check in `internal/body` exists to prevent, in the spelling Obsidian
hides while somebody reads the note, and "two parsers, one meaning" is the
premise it broke. The inline parser now walks forward the way goldmark's does,
over the lines of the paragraph in hand and no further, and puts the position
back where it found it when there is no closer to reach. A single delimiter
that closes nothing is still the character the author typed.

**A comment the note never closes is named as a comment in both spellings.**
The milestone named the unclosed fence and left the unclosed `<!--` as "a
block of HTML", which is the spelling the guide teaches and the one `export`
writes. An author told they wrote a block of HTML goes looking for a tag that
is not in the note. Both spellings run to the end of the note, so the warning
now names the comment and says the lines under it went with it. The same
reading found the other half: Obsidian's opener is also its closer, and the
block was read as closed whenever its last line happened to end in `%%`, so
`%%` with `note 50%%` under it swallowed every line after it as one grey aside
with nothing named. A fence shuts on a fence line and on nothing else, which is
what `unclosedComment` now asks.

**The old span is named on the way home.** `internal/body` refuses a line
carrying `[words]{.pencil}` by name, and the projection stopped naming that
shape when it took on the comment's opener. So a document whose own prose
shows the retired spelling, a page about gdoc among them, came home with no
warning and refused the next build by line, blaming the file rather than the
export. It is the third shape `gdocShapes` asks for, with its own sentence,
because the next build does not read those characters in place of the words:
it refuses the file for them. The shape goes the day that refusal goes.

**What was left.** A grey run's words are still written home through the
projection's own escaping, so a pencil holding `[[`, `{+` or a backslash
gains a character nobody typed. That is
`docs/backlog/build-never-unescapes-a-backslash.md` and it is Nail's to settle,
because unescaping inside a comment contradicts "nothing inside a comment is
markdown". The old span inside a pencil's own words is refused by `build` and
named by `export`, which is consistent but stricter than "nothing inside a
comment is markdown" would have it; loosening a refusal is a decision rather
than a review fix.

**Where they are pinned.** `internal/body`'s
`TestAWrappedObsidianCommentIsOnePencil`,
`TestAnObsidianDelimiterWithNoCloserIsWords`,
`TestAnUnclosedHTMLCommentIsNamedAsAComment`,
`TestAnUnclosedFenceEndingInTheDelimiterIsStillUnclosed` and
`TestABlockOfHTMLThatIsNoCommentIsStillNamedAsOne`; and `internal/export`'s
`TestWordsHoldingTheOldPencilSpanAreNamed`.

## 2026-10-17. What the third M20 review closed: an opener with no closer inside a sentence, and the refusal that blamed an export the author never made.

Two corrections to the entries above, found by the third review of the
milestone. Both are rules those entries already state, applied where they were
not. Nothing here changes what a pencil is.

**An opener with no closer inside a sentence is refused by line.** The entry
above says a comment the note never closes runs to the end of the note and is
named as the comment it is. That is true of an opener on a line of its own,
which opens a block of HTML. An opener in the middle of a sentence opens
nothing: goldmark reads raw HTML only once it has the closer, so
`Some prose <!-- an aside with no closer` stayed one paragraph of prose, the
aside reached the document black, and the typographer rewrote the two dashes
of the opener into an en dash on the way. The build said ok. That is the one
failure every line check in `internal/body` exists to prevent, and the words
are an aside the author wrote for nobody, so it is a refusal rather than the
warning the block gets. It is read off the lines every other line check is
read off, and it counts the openers goldmark did read before it refuses: a
comment the author's editor wrapped, an opener inside a comment's own words,
and an opener inside backticks or a code block all refuse nothing. Only the
HTML spelling is asked about, because a lone `%%` is two characters an author
may well have typed for themselves.

**A note refused for a comment of gdoc's own is told what to do about it.**
The entry above made four of the eight markers a comment opening with an
ordinary English word. The refusal they share was written when every marker
was syntax nobody types by hand, so it says the note was exported out of a
document and its suggestions have not been resolved. For
`Text with <!-- tab order matters --> inside` that sentence is false: the
author exported nothing, and it sends them looking for an export that never
happened and names no way out. The refusal now has two halves.
`internal/markers`' `IsComment` is which half a marker is in, there because
that package owns the list. The markup half keeps the sentence it had. The
comment half names the opener as gdoc's own and asks for the aside to be
reworded. The refusal itself does not move: a comment opening with those words
is read as gdoc's on every route back into a document, which is what makes it
a marker.

**What was left.** The two shapes the entry above left for Nail are still
Nail's: the backslash a grey run's words gain on the way home, and the old
span refused inside a pencil's own words. `internal/view` names a shape twice
when a pencil's mark cannot be carried and a non-commented shape stands in the
same run, and it says the next build "refuses the file for" the old span where
a whole grey paragraph comes home as a block of HTML and the next build does
not. Both are wording in a warning that is already raised, so neither puts a
character into a document and neither is worth a change this late in the
milestone.

**Where they are pinned.** `internal/body`'s
`TestAnUnclosedCommentInsideASentenceIsRefusedByLine`,
`TestAnOpenerGoldmarkReadIsNotRefused`,
`TestAnAsideOpeningLikeGdocsOwnCommentSaysSo` and
`TestAMarkupMarkerStillNamesTheExport`.

## 2026-10-17. What the fourth M20 review closed: an escaped opener refused as a comment the line never closes.

One correction to the entry above, found by the fourth review of the
milestone. It is a rule the entries above already state, applied where it was
not.

**An escaped opener refuses nothing.** The refusal the entry above added
counts the comment openers on a line and compares that count with the openers
goldmark read. It counted every `<!--` on the line, the escaped ones with the
rest. A backslash makes the one character after it the author's own, which is
the parity `internal/markers` holds for every route out of the hub and the
parity `read` writes an author's own opener home with, so `\<!--` is four
characters of prose: goldmark reads no raw HTML there, nothing is left
unclosed, and the characters reach the page as the ones that were typed. The
count said otherwise, so `An escaped opener \<!-- aside --> here.` was refused
with a sentence saying the line never closes, which it does, and the way out
it named was a closer the line already had.

What that cost is the door the marker refusal leaves open. The entry above
says an escaped marker is the author's own text and is left alone, so a note
written about the markers themselves still publishes. Four of the eight
markers are comments now, so for those four the marker refusal let the note
through and this refusal stopped it: a note explaining
`\<!-- image: picture.png -->`, which is the shape of `skills/gdoc-export/markers.md`,
could not be published at all.

Both counts are read through the parity now, the line's and the raw HTML's,
because an opener escaped inside a comment's own words is that comment's words
on both sides of the comparison. `internal/markers`' `escaped` is exported as
`Escaped` for it, rather than copied, because that package is where the parity
lives.

**Where it is pinned.** `internal/body`'s `TestAnEscapedOpenerIsNotRefused`,
`TestAnOpenerAfterAnEscapedBackslashIsStillRefused` and
`TestBuildLeavesAnEscapedMarkerAlone`, which now asks its question of a
comment marker as well as of the markup.
## 2026-10-17. The logo lives in Drive, not in the tree.

Nail's decision, taken on 2026-10-16 in a brainstorm that put the question to
three readers: a Google Workspace API researcher, a UK brand counsel, and
Codex for a fresh read of the code against the principles. The repository is
public since 2026-09-16, and the 2026-10-02 entry took the firm's domain out
of the tree. The firm's logo stayed: a base64 PNG in `house.yaml`, embedded in
every release, written into every document gdoc builds, including one built
by a stranger who installs the tool and never signs in. From this entry the
tree carries no logo. `house.yaml` names the firm's logo by the id of one file
in the firm's Shared Drive, and a run gets the bytes only when the account it
signed in as can read that file. Supersedes one assessment in the 2026-09-16
release entry, "what is public is the Altery logo", and nothing else in it.

**What was found first.** No Google API returns a Workspace organisation's
logo. The Admin console logo is upload-only, Directory `customers.get` returns
nine fields and none is a picture, Cloud Identity has no branding resource,
and the template gallery cannot be listed. The only two readable places are a
file in Drive by id, under the `drive` scope the token already carries, and
an inline image inside a Doc, whose `contentUri` lives thirty minutes on a
host the guard refuses. The OAuth client is User type Internal, so only an
account in the firm's Workspace can sign in at all: `publish` and `restyle`
were gated before this entry, and what was exposed is `build`, which writes a
branded docx with no sign-in, and the bytes themselves in `house.yaml`, three
test docx and every release. The legal risk of the public PNG is low, a logo
on the firm's website is already public; the concern is a stranger producing
a document that looks like the firm's.

**What was rejected.** An identity gate, reading the signed-in account through
`AllowAccountRead` and comparing its domain to a configured value: the logo
would stay public in the repository, and identity would become a gate, which
no invariant lets it be. The whole house file in Drive: the sizes and the
styles are the generator's own business, a firm that forks gdoc edits one
file in the tree, and it is the picture that is the firm's. The house file in
the hub: it ties the brand to one vault, and Nail wants the same file to serve
a colleague without one. A Drive folder found by listing or by name:
`files.list` stays refused, and anybody who shares a file of the right name
with a colleague would brand their documents. An id in `settings.yaml`
written by the installer: every install would depend on one line on the
wiki being right. A fetch from `login`, from the daily release check, from
the installer or from a command of its own: `login` is the token and nothing
else, the daily check carries no credential on purpose, the installer has no
token, and a command of its own is a line on the wiki somebody forgets.

**`house.yaml` stays, and names the logo by id.** `logo:` keeps `mime`, `px`,
the sizes and the anchor and carries exactly one of `drive`, a Drive file id,
and `base64`, the bytes. Both or neither is refused naming the key. The
embedded file carries `drive`. `base64` stays for a test house and for a firm
that prefers embedding, so `--house <file>` is the override and there is no
setting. `internal/house` still reaches nothing: it checks the id's shape, and
the PNG check moves to the room that has the bytes.

**One room, `internal/asset`, and one grant, `AllowAsset`.** The room reads
one Drive file by id as bytes, checks them against what the house says of
them, and caches them. It is handed a client and never builds one, and it
knows no logo as such, so test data kept in the same Drive later is the same
call with another id. The grant is the seventh of the shape the six before it
have: one id for one run, dying with the process, nowhere written down. It
admits `GET /drive/v3/files/{id}` with `alt=media` and `supportsAllDrives`,
both already in the guard's parameter list for `files.get`, and nothing else
on that id, no metadata read, no Docs read, no patch, no copy. It is a third
door into the reachable set, for one object that is not a document, and the
set cannot grow through it because a PNG carries no id.

**Fetched at render, through a cache, once a day.** `build` and `publish` are
the two commands that write a docx, and they are the two that read the logo.
Each looks in the config dir beside the token, at `assets/<id>` with a stamp
in `internal/lastcheck`'s shape; a stamp older than a day, missing or
unreadable makes one request, a fresh one makes none. Six moments were
weighed, and this one costs one five-kilobyte read a day across both
commands, keeps `build` as fast as it was on every other day, and is the same
mechanism the release check already uses, so nothing new is kept on disk
that gdoc does not already keep. `export`, `restyle` and `help marks` read the
house file for other reasons and never touch the room, so `read` still makes
no request. The invariant that two commands ask unasked is untouched: the
request here is asked, by the house file the run named.

**Not knowing never resolves to an unbranded document.** Offline, refused or
timed out, the cached bytes are used and the envelope says so, with their age
and the reason. No token and a cache is the same. No token and no cache
refuses, naming `gdoc auth login`: a stranger's `build` on the embedded house
refuses, which is the point, and a colleague has a token before the first
build because every skill asks for one. Bytes that come back and are not a
PNG of the stated size refuse the run even when a good cache exists, because
the firm's file is wrong and the refusal is the only message that reaches
whoever can fix it. A cache that cannot be written still builds the document
and says the cache was not saved. The envelope carries a `logo` block of
facts, `source` as `drive`, `cache` or `file`, with the id and the time or the
age and the reason, and judges none of them; in chat the person hears about
the logo only when it came from the cache.

**What this strains.** Principle 1. The logo no longer travels with the
binary. It travels with the firm's sign-in, is cached beside the token, and a
machine that has never signed in builds nothing in the firm's style. Nail
accepted that on 2026-10-16. History and the 28 tags before this entry are
not rewritten: every old binary carries the old logo, and the removal stops
gdoc shipping it from now on rather than recalling what is out. The README
says in one line that the firm's logo and marks are not licensed with the
code; the repository has no licence file, and adding one is a decision of its
own, not taken here.

**The tree carries no logo.** The three test docx, the drift master and the
two live probes, carry a neutral placeholder PNG of the same 195 by 97 pixels
in place of the firm's, and drift's seven logo items still measure because
they read geometry and never a pixel. A boundary test holds the real PNG's
SHA-256 the way the domain test holds the domain's, walks every tracked file
and inside every docx, and fails on a match.

**What is not built.** A folder of assets found by listing. A settings
override for the id. A metadata read for the file's revision. A rewrite of
history. Any second caller of the grant.

**Tests.** `internal/house`'s `TestALogoNamesExactlyOneOfDriveAndBase64`,
`TestADriveIdThatIsNotOneIsRefusedByName`, `TestADriveLogoCarriesNoBytes` and
`TestTheEmbeddedLogoIsADriveIdAndCarriesNoBytes`; `internal/guard`'s
`TestTheAssetGrantAdmitsOneMediaReadAndNothingElse`,
`TestWithoutTheAssetGrantTheReadIsRefused`, `TestTheAssetGrantAdmitsNoDocument`
and `TestTheAssetGrantDiesWithThePolicy`; `internal/asset`'s
`TestAStaleStampFetchesOnce`, `TestAFreshStampMakesNoRequest`,
`TestNoTokenAndACacheUsesTheCache`, `TestNoTokenAndNoCacheRefusesNamingTheLogin`,
`TestARefusedReadUsesTheCacheAndNamesTheReason`,
`TestABadFileRefusesEvenWithAGoodCache`, `TestACacheWriteFailureStillBuilds`
and `TestTheCacheIsWrittenAtomically`; `cmd/gdoc`'s
`TestBuildReportsTheLogoSourceFile`, `TestBuildReportsTheLogoSourceDrive`,
`TestBuildReportsTheLogoSourceCache`, `TestPublishReportsTheLogoSource` and
`TestBuildWithNoTokenAndNoCacheRefusesNamingTheLogin`; `go/boundary`'s
`TestTheLogoIsNotInTheTree`, `TestTheLogoScanFindsAPlantedPNG`,
`TestOnlyBuildAndPublishCallTheAssetRoom` and
`TestReadExportRestyleAndHelpNeverReachTheAssetRoom`; and `internal/live`'s
`TestPublishCarriesTheHouseLogo`, under `GDOC_LIVE_WRITE`. The specification
is
`docs/plans/completed/2026-10-17-gdoc-v2-m22-the-logo-lives-in-drive-spec.md`
and the task list is beside it.

## 2026-10-17. The presence: gdoc appends one fact line per run to a file it never reads, and a separate menu bar app reads it.

Nail's brief, given the night of 2026-10-17 and designed the same night by
four seats in parallel, with the project manager resolving their
disagreements and Nail not yet having reviewed the result
(`docs/design/2026-10-17-presence-mechanism.md` lists what he is most likely
to reverse). The brief, in meaning: gdoc must not change in kind; it may
report what it is doing in real time, and the engineers choose how. The
presence is a separate application whose changes never touch gdoc. `gdoc
update` updates it too. It is behind a settings flag an argument can set, so
only whoever sets it has it. Main gdoc stays backward compatible. Serves
principle 4: a live review is hours of silence in a chat, and one glance at
the menu bar now answers "is it alive" and "is it waiting on me". Strains
principle 2 and principle 1, and the specification's gate names both.

**The channel.** One append-only JSONL file per month,
`presence/<yyyy>-<mm>.jsonl` in the config directory, 0700 and 0600 like the
token. Each event is one open with `O_APPEND`, one write of one line, one
close. The binary writes it at `dispatch`, start and end of every command
that names a document plus `update` and `auth login`, and at the holds in
`gdoc mcp`: held, released, expired. It never reads it: the package exports
no reader, `O_APPEND` appears in no other room, and no hold rule, receipt or
cursor depends on a line in it, because the day a hold rule read the log, any
local process could append "gdoc wrote reply X" and a stranger's marked remark
would pass as gdoc's own. Rejected, each independently by more than one seat:
a unix socket (two-way at the OS level, loses every event while the app is
closed, a dial outside the guard), a named pipe (blocks or fails with no
reader, no history), stderr or terminal sequences (land in Claude Code
transcripts, never reach the app), distributed notifications (cgo, and the
build is CGO off), and a `gdoc status` command (turns the binary into a
reader of its own record).

**The event is a projection, never the envelope.** Product decision 7 of the
design pass said "append the same envelope"; that would write every document
body `read` prints and every comment word onto disk beside the token, and
all four seats refused it. The event is a fixed allowlisted struct, schema
version 1: the instant, the pid and the process start, a call counter, the
event kind, the command, terminal or chat, the document id, at the end the
title, `ok` and `verified`, and for a hold its id, its rule and when it runs
out. Never a document word but the title, never a quote, a hold's value or
text, an error sentence, a path, the account, the token, argv, the cursor,
the mark, or a judging word. A canary test holds the line. The title is the
one document word, capped at 200 runes, because without it the app would run
`gdoc read` to fill a row, which pulls a whole body into a second process.
The binary keeps this and last month's file and removes older ones when it
writes, which reverses product decision 8 of the design pass ("the app never
prunes"); a forever history of titles would become the most sensitive
readable thing in the config directory once the token moves to the Keychain.

**The flag.** `presence: true` in `settings.yaml` and `--presence true` on
the line, a third setting flag beside `--mark` and `--config`, the argument
over the file, a setting and never an identity check. It is in the table
marked experimental: no help screen line, no completion line, and `help
--json` carries it only when it is on, so the goldens stay byte for byte for a
colleague. Off is today's bytes: no directory is created, no envelope
changes, `gdoc update` is unchanged, and the app is never uninstalled.
Settings are decoded strictly, so a binary older than this key refuses a file
that carries it: the flag ships one release before anyone sets it, and the
guide says so.

**The delivery.** The app travels inside both darwin zips at
`presence/GdocPresence.app/`, one universal bundle under the one
`SHA256SUMS`, built by a second job in `release.yml` on a macOS image, which
the ubuntu job needs; the platforms file stays the one list. A broken Swift
build fails the release rather than shipping a split version. With the flag
on, `gdoc update` installs the bundle at `~/Applications/GdocPresence.app`
out of the verified zip, by a stage, a read-back and a rename swap after the
binary's own three renames, keeping the previous bundle; every app failure
is a warning and never undoes the binary, and rollback swaps both. gdoc
never launches the app: the first launch is the person, and after an update
the app sees its on-disk version differ and relaunches itself. The one
external program stays the one, pinned by a new test on its one caller.
Gatekeeper is not asked about a bundle Go wrote, because Go sets no
quarantine attribute; measured in part, and the plan carries the rest into
MEASURED.md with Nail confirming the method.

**The app.** Native Swift in its own package at `presence/`, an AppKit
status item and panel with SwiftUI content, macOS 14 minimum. It tails the
file, decodes strictly, treats every line as forgeable data, and runs
exactly one gdoc command, `auth status`, as a fixed argv at an absolute path.
No flag, variable or subcommand exists for the app alone, and gdoc never
behaves differently because the app is the caller. No channel runs from the
app into `gdoc mcp`: the hold card says "Open chat" and nothing else, and the
panel never says "Release", because in the binary that word means send. The
first milestone shows a ring, the mark, a pause glyph or nothing, with NOW
and TODAY; the panel the design files draw is the next milestone.

**What this reverses or defers.** "The wait writes no file of its own" in
`comments/doc.go` stays literally true, because the hook is in `dispatch`,
and gains a pointer. Product decisions 8, 31, 47 and 48 of the design pass
are reversed or deferred as the mechanism document says. `gdoc auth logout`
waits.

## 2026-10-17. The panel: the presence as the screens draw it, the app runs six argv lists, a hold dies with the sign-in it was held under.

Nail's instruction the morning after M23 shipped: the panel looks nothing
like the design discussed the evening before, run the next milestone now,
and make the app start by itself. The M23 scoping that deferred the drawn
panel was the project manager's, and this entry undoes it in one milestone
run as two halves, the Swift panel and what the binary adds for it. The
rules are in `docs/plans/completed/2026-10-17-gdoc-v2-m24-the-panel-spec.md`;
the visual source is `docs/design/2026-10-17-presence-screens-v3.html`, the
third version of the screens Nail reviewed. Serves principle 4, the screens
are the reader's own choice, and principle 3 in three places named below.
Strains 1 and 2 no further than the M23 entry; one M23 sentence is reversed
and said so here.

**The app launches six argv lists and nothing else.** The M23 rule "the app
launches no process" is reversed: the panel needs the account, the mark, the
version, a sign-in, a sign-out and an update, and each is one fixed argv for
the gdoc binary at an absolute path, `help --json`, `auth status`, `auth
login`, `auth logout`, `update` and `update --desktop`, under one runner
file with stdin the null device and four environment variables. No setting
flag, no flag only the app uses, no PATH lookup, one child per kind at a
time, and a source scan that enumerates the set. The app also signals one
kind of process: one SIGINT to a `comments --wait` it has proved is gdoc's,
alive, the user's and the one the record names, which is what the review
skill already reads as "you stopped it". gdoc itself still runs the one
program it ran, and never the app.

**The record gains fields, never words.** At schema version 1, because the
M23 app ignores unknown fields and skips unknown kinds: `made`, the document
a publish created; a `step` event for publish with a fixed name and index;
`stop`, a fixed code for why a run failed, with `line` where the cause has
one; `via: code` under Claude Code; `interrupted` on a wait; and `dropped`
as a hold's fourth ending. Refused, each by name: a path to the note, a Drive
folder name, an error sentence, a count from the document, a judgement on
thread state, the mark, the account. Open md needs no path: the app reads
the front matter of the hub the person chose, one read the M23 entry did not
grant and this one does.

**A hold dies with the sign-in it was held under.** Found this morning: a
release checked only that a token exists, so a held write would post under
whatever account signed in next, which is the fear product decision 48
named. `mcpHolds` keeps a hash of the refresh token beside the hold, in
memory, never in `internal/chat`; a release or a sweep that finds another
token or none drops the hold and says so; a token that cannot be read keeps
it and sends nothing. No email is compared, so identity is still no gate.
With that, a sign-out needs no channel from the app, and the panel's hold
card carries "Open chat" and an app-local "Dismiss", never "Release".

**`gdoc auth logout`** removes the token file and nothing else, reads the
absence back, refuses a symlink, a file that is not the token and a waiting
sign-in, is no chat tool, and is a visible seventeenth command.

**The app starts by itself.** Launch at login is on by default, registered
by the app on its first run ever and never re-registered over a person's
"off", which reverses product decision 32 at Nail's word. After `gdoc
update` replaces the bundle the app relaunches itself into its own path,
once per on-disk version, never while an update runs, quitting only after
the new instance is up; the swap under a running instance was measured on
2026-10-17 at 09:25 (MEASURED.md). gdoc registers nothing and launches
nothing.

**What yields to a principle.** The vote's "a failed read-back is red"
yields to principle 5 of the design: `verified: false` is a hollow check and
never stuck. The screens' "Release without writing" yields to the no-channel
rule. The screens' per-step counts and folder names yield to the allowlist.
Idle keeps the dim mark so the panel stays reachable, until Nail says
otherwise.

## 2026-10-18. The app opens itself: gdoc opens the installed app when it is not running, and launch at login is the person's switch.

Nail ran a publish from Claude Code the morning after v2.15.0 and the panel
was not there. The app had never been opened: the M24 rule registered it to
start at login on its first run, and the M23 rule said gdoc never launches
it, so between the two the first run was his and nobody had told him in the
moment. His expectation, in his words: if it is not open, the gdoc command
opens it; he can open it by hand; it starts at login when he has decided so
in a setting. This entry is those three. It reverses one sentence of the M23
entry ("gdoc never launches it") and one of the M24 entry ("launch at login
is on by default"). Serves principle 4: a person who turned the presence on
sees what gdoc is doing without remembering to open anything. Strains
principle 1 by one more thing the binary does on the machine, named below in
full.

**gdoc runs `open` for the app it installed.** Under the `presence` setting,
on macOS, after the record's start line and before the work, when
`~/Applications/GdocPresence.app` is there and the app does not hold its
lock, gdoc starts `/usr/bin/open -g <bundle>` and does not wait for it. `-g`
because the app has no window to bring forward and the person is in a
terminal or a chat. The exit status is never read, so a shell with no window
session runs the command exactly as before. Every recorded command but `gdoc
update`, which is the one command that moves the bundle: it runs the same
opener itself, once, after it places, replaces or rolls back the bundle, so an
install is a running app, the answer no longer tells the person to open it,
and no run starts the build it is about to rename away. The one
program gdoc runs is still `/usr/bin/open` at its full path, from the one
file, through the one seam; the seam now has two callers and two argument
lists, both pinned by the boundary tests, and no third. Off is today's bytes.

**Whether the app is running is one lock, and no byte is read.** The app, as
its first act, holds `flock(LOCK_EX)` on `presence/app.lock` in the config
directory for its whole life; the kernel releases it when the process dies,
however it dies. gdoc asks `LOCK_EX | LOCK_NB` on that file and lets go at
once; a file that is not there means an app that never ran. gdoc never
creates the lock file and reads nothing from it; the record stays the one
file gdoc appends to and never reads, and nothing gdoc decides depends on a
line in it. A second instance of the app that does not get the lock quits
without drawing a status item.

**Launch at login is off until the person turns it on.** Product decision 32
stands again: it is a setting. No run registers the login item by itself;
the HEALTH switch registers and unregisters, records the choice, and shows
what macOS says. The M24 sentence that registered on the first run is
withdrawn, because the open above does what that registration was for
without touching the person's login items.

**What was offered and refused.** Opening the app from the skills: the
skills are five files and the chat in Claude Desktop runs none of them, so
the binary is the one place every route goes through. Opening it on every
command without asking whether it runs: `open` on a running agent app is
harmless but is a process per command. A pid file the app writes: a pid is
reused, a lock is not. Reading the record to see whether the app is alive:
the record is written by gdoc and says nothing about the app.

The plan is `docs/plans/completed/2026-10-18-gdoc-v2-m25-the-app-opens-itself.md`.

## 2026-10-18. The second morning: the account in the header, the middle scrolls, DOCUMENTS by what was just made and by use, pins, a bright mark, and the session a run came from.

Nail used the panel for a morning and said six things. The scrolling was
wrong: everything under the header moved as one piece, so NOW and the bottom
group left the screen. The account and its sign-out belong at the top, where
the address already is. The group they leave is not HEALTH any more. Open
chat opened Claude and not the session the document came from. The menu bar
mark was too dim at rest. And DOCUMENTS should put what gdoc just made first,
then the most used, with a way to pin a document so it never drops off. This
entry is the six answers. Serves principle 4 and 3; strains 2 by one field,
the last paragraph.

**The edges stay and the middle scrolls.** The header, NOW and the bottom
group never move; DOCUMENTS, TODAY and Yesterday are the one scroll box,
with the thumb drawn only while the content moves. NOW alone, when it is
taller than half the body, scrolls inside itself so the bottom group is
never pushed off. Product decision 2's "each section scrolls inside itself"
is narrowed to this. Nail chose it over per-section scrolling and over no
cap.

**The account and the version are the header's.** The header carries the
account's glyph, the address and Sign out, and asks "Sign out?" in its own
line; after the name it carries the gdoc version and, when a newer release
is listed, the update action. The bottom group is SETTINGS: Hub, Launch at
login, Quit, and the Network and Claude Desktop rows when a fact carries
them. HEALTH as a word goes, and so does the Thread mark row: the mark is
already the first thing in the header, and a row that repeats it was, in
Nail's words, a pointless thing. The Hub row is the picker the guide's
`defaults write` stood in for: on 2026-10-18 the md icon was grey on every
row of Nail's panel because nothing had ever set the path.

**DOCUMENTS orders by what was just made, then by use.** Documents with an
action today come first, newest first; the rest of the fourteen days follow
by the count of finished actions, most first. The count is never printed:
it orders and says nothing. A pinned document comes before all of these, in
the order pinned, and never leaves the list; the pin keeps the id and the
last title so a document whose lines aged out of the record still has its
row and its open action. A document can also be hidden: it leaves the list
and its actions stop counting, TODAY still shows its runs because the log is
the log, and the next action gdoc does on it brings it back. Pins and hides
are the app's own preferences, reached from the row's own menu so the four
icons stay four; never a line in the record.

**The mark at rest is as bright as the mark at work.** The 45 percent idle
mark was the one choice principle 1 argued against and Nail has now
answered: he wants to see it.

**The record gains `session`, and Open chat opens that session.** Claude
Code hands its children `CLAUDE_CODE_SESSION_ID`; under `via: code` the
start line carries it, and nothing else does. The Claude Desktop bundle
carries `claude://code/continue?session=…`, so a document whose latest
Claude Code run has a session opens that link; whether the Code tab takes a
session id and not only `last` is measured by Nail watching the window, and
the link ships only if it does. A session id is an identifier of the
person's own session and no document word, which is why it is allowed into
a record that refuses paths, sentences and accounts; it is still one more
thing in the file beside the token, and principle 2 notes it.

The plan is `docs/plans/completed/2026-10-18-gdoc-v2-m26-the-second-morning.md`.

## 2026-10-18. The third look: nothing scrolls, DOCUMENTS is the pinned and the last five, the note is found and not configured, the update glyph has three states.

Nail opened v2.18.0 and sent a screenshot with three scroll thumbs on one
panel: the middle box, DOCUMENTS inside it and TODAY inside it. His words:
double scrolling is awful; no more than five documents and no activity log
for today and yesterday at all; why a hub, it differs by document and the
notes live in different folders; he did not see how to pin; the use
statistics are not needed; the last five plus the pinned, no duplicates, no
scroll, let the panel be as long as it comes out; check for a new version
periodically and give the update glyph states, so he can press it to check
and install before the timer has asked. This entry is those answers, and it
withdraws three sentences of the same day's "second morning" entry: the
middle scroll, the by-use order and the Hub row. Serves principles 4 and 1.

**Nothing scrolls.** The panel is as tall as its content. The 600 pt cap,
the middle box, the inner scrolls and Show more go. A panel taller than the
screen is cut at the screen's edge, which Nail chose over any scroll; the
height it happens at is measured and written down, and a later entry may
cap NOW if it ever bites.

**DOCUMENTS is the pinned and the last five.** Pinned documents first, in
the order pinned, then the last five documents by their latest action, one
row per document, hidden ones skipped, no count in the header. TODAY and
Yesterday are gone: the log is the record, and the panel shows what gdoc is
doing now and the documents it touched. The pin is the fifth icon of a row,
outline and filled, because a menu nobody finds is no affordance; Hide
stays in the menu.

**The note is found, not configured.** The hub folder was one setting for a
thing that differs by document. The app asks Spotlight for Markdown files
whose text holds the document id, reads the `gdoc:` front-matter block of
each candidate as it already did for a whole hub, and opens the one whose
block names the id; a file that mentions the id in its body is a candidate
the check rejects. Measured on Nail's machine: both notes found in 0.25 s,
727 Markdown files indexed under the vault. The Hub row goes; `hubPath`
stays as a `defaults write` override for a person whose notes are not
indexed. The app's reads are now the record, its own plist, and the front
matter of the files Spotlight names: narrower than a whole hub.

**The update glyph has three states.** Quiet in the caption grey, when
nothing newer is known: pressing it runs `gdoc update`, which asks GitHub
every run and installs or says this is current. Available, in the accent ink
with a dot and the newer version after the current one, when `help --json`
listed a newer stable: pressing it runs the same update. Running, the ring.
The app asks `help --json` on open and every six hours while it runs;
gdoc's own stamp keeps GitHub to once in 24 hours, and the person's press is
the way past it.

The plan is `docs/plans/completed/2026-10-18-gdoc-v2-m27-the-third-look.md`.

## 2026-10-26. The fixes: a cover line and the footer keep their own look, a logo of any size fits by width, six points after the two top headings, Russian heading ids keep their letters, the guard refuses an empty leaf in a style mask, export keeps code and repeated headings apart.

An outside review of commit `7be00df` named sixteen bugs, and the team checked
each one with a test against the code. Nail chose which to fix before this
day, and the lead's handoff of 2026-10-26 holds his choices: the render
fields, the Russian heading ids, the null leaf in the guard, the two export
heading bugs, and a logo that fits any size. He added one of his own that
day: after a heading he wants space before the text starts. The rest of the
report stays in `docs/backlog/small-defects-from-the-outside-bug-report.md`.
On the evening of 2026-10-26 the lead put five questions with a
recommendation on the night plan page, and Nail answered, in his words: "I'm
ok with night plan and proposed recommendations". Serves principles 1 and 2.

**A cover line and the footer keep their own look.** A cover line's `align`,
`italic`, `color` and `line_spacing` in `house.yaml` reach the docx, where the
render dropped them. The footer paragraph's alignment and its rule reach it
too, and so does a label's `keep_with_next`. A footer logo is refused by name,
the way a header logo in the wrong place is refused, because Nail does not
want one.

**A logo of any size fits by width.** This replaces one clause of
2026-10-17's "The logo lives in Drive". The exact-pixel pin goes, because a
bigger picture in Drive stopped every build until v2.20.0 pinned the new
size. The picture is drawn `width_pt` wide and its height follows its own
ratio. The mime type and the byte ceiling still refuse a wrong file, a side
over 4096 pixels is refused before the picture is decoded, so a small file
cannot claim a huge one, and the
Drive id, the cache and the refusal to build without a logo all hold.

**Six points after the two top headings.** `heading_1` and `heading_2` take
`space_after_pt: 6`, the value `heading_3` and `heading_4` already carry. The
master template said 0. That difference is named in the drift gate's known
list with this entry as its reason. A restyle applies the same value. Nail
chose 6 over Google's default of 10.

**Russian heading ids keep their letters.** A Markdown id keeps Cyrillic the
way GitHub does, so `## Цели проекта` is `#цели-проекта`, and a repeated
heading takes `-1`, `-2` after the first. An id that works today keeps
working. For most headings it is the same id. A heading with an accent or an
underscore gets GitHub's id, and its old id stays as an alias a `#` link still
resolves through. A heading with no letter at all keeps the id `heading`. The docx bookmark stays ASCII, `_h_` and a hash, because Word repairs
any other name. Measured the same evening in MEASURED.md, "Drive's import of a
Russian heading and a link to it": a link through such a bookmark lands on the
Russian heading, so the fallback Nail approved, plain ids, is not needed.

**The guard refuses an empty leaf in a style mask.** A styling request whose
`fields` names a field the style object holds as `null` is refused by name,
because an in-place restyle must not clear a value nobody stated.
`AllowFile` accepting any level stays as it is, by Nail's call.

**Export keeps code and repeated headings apart.** The strip of a heading's
number never runs on a line inside a fenced code block. A link to the second
of two headings with one name points at the second.

**Two runs in one night.** The team's rule is one milestone a day. Nail
approved two for this night: this one on codex, as the M29 pilot ran, and the
panel on Claude. Out of both: restyle applying the whole template,
suggestions in footnotes, the release pipeline rethink, a live review's
progress in the thread, the installer signing in, and the rest of the outside
report.

Trust guardian, 2026-10-26: signed. The guard only narrows, refusing a mask
path held as null, and `TestAMaskMayNameOnlyWhatTheRequestSets` and
`TestTheAssetGrantAdmitsNoDocument` hold unchanged. The logo keeps its mime
check, its 1 MiB byte ceiling and, in place of the pixel pin, a 4096-pixel side
ceiling judged on the header before any decode.

The plan is `docs/plans/completed/2026-10-26-gdoc-v2-m30-the-fixes.md`.

## 2026-10-26. The fourth look: every document on the panel opens on a click, a mark only for a failed run, names that survive, buttons that say what happened, a panel that stays open, a bounded hub.

Nail, 2026-10-26, on the panel of v2.20.0: every document the menu bar app
shows must open on a click, and the live row could not be opened. On the
DOCUMENTS list, a grey dot and a green check sat before rows and nothing said
why; he asked for the difference made clear to a person, or dropped. He also
said of the panel "feel free, tune it, adjust". The lead put five questions on
the night plan page, and Nail answered: "I'm ok with night plan and proposed
recommendations". Serves principles 4 and 1.

**Every document on the panel opens on a click.** A NOW row that names a
document opens it on a click of the row, the live row included, the way a
DOCUMENTS row's first icon opens its own. Stop stays its own control, as it
is today with no confirm, so a click on the row never stops a review. The
whole row is the target, not an extra icon: the lead's call. A DOCUMENTS row
keeps its icon and gains no row click.

**A mark only for a failed run.** The grey dot and the green check go,
because the row's second line already says what the last run was. A row
carries a mark only when the last run on its document failed, and a failed
run never looks like a stopped one: it says "reply failed Tue", not "stopped".
Every other glyph in the column goes too, the hold, live and running ones
included, because NOW already shows what is happening. The column stays, so
every title starts at one edge. Nail chose this over dropping the column.

**Names survive, and the buttons say what happened.** A verb in front of a
row keeps enough of the document's name to tell documents apart, and a
publish that stopped names its document. An update button that did nothing
says why on its row, in one fixed sentence with the time. A publish that
stopped in the render names its note through the cover title, which is the
one change to gdoc itself: `title` on the end line may now be that cover
title, and the record gains no field.

**The panel stays open, and the hub has a bound.** A held press opens the
system menu the right-click opens, so no second window takes the panel's
place. The folder picker that closed the panel went with the third look, so
its backlog item closes with no code. The walk of a `hubPath` override stops
at 20,000 entries and lets Spotlight answer, and a new walk stops the old
one. The flaky test of the override before Spotlight was a real race in the
app: Spotlight now waits for the override's walk.

The six argv lists, the one signal and the three things the app reads hold
unchanged. Out: an app run by an AI skill, the logo on the panel, the skills
and extension versions on the panel, and the app in the nightly zips.

Trust guardian, 2026-10-26: signed. `title` on a publish's end may be the
note's cover title, the same word its step lines already carry, from one
caller held by go/boundary; no field, reader, URL, argv or system call is
added, and the hub walk only narrows what the app reads.

The plan is `docs/plans/2026-10-26-gdoc-v2-m31-the-fourth-look.md`.

## 2026-10-27. Week one of whole jobs: a watch that wakes a session only for marked work, a short help, live review in its own session, and a stopwatch on every comment.

The research of 2026-10-27, `docs/v2/research/how-ai-skills-talk-to-gdoc.md`,
measured where a live review's time and tokens go. gdoc is about a quarter of
a live window and starts in 20 to 60 ms. The rest is model turns, and each turn
re-reads the whole session: 445k to 725k tokens in a hub session, 257k to 317k
in a session of its own. Of 120 waits, 43 came back on the first poll, often
with the session's own reply, and 26 ran the full nine minutes, and each of
those cost a full turn. Nail asked for a plan for the first step of the road
the research recommends: "Do Plan for week 1. Also keep somewhere out plans for
next week's." The lead put five questions with a recommendation: the five parts
only and no echo filter in `--wait`; a reply with no marker stays quiet and is
listed at the next real wake-up; the panel's live row exactly as today; the job
packet, a session per comment, unattended answers and Claude Desktop all out;
and the run tonight. Nail answered "ok". The later weeks are the table under
"What is next" in `docs/v2/PLAN.md`. Serves principles 1 and 4.

**A watch that keeps running, `comments --follow`.** It takes the same `--since`
cursor as `--wait`, polls the same listing every two seconds, and runs until
its duration, at most an hour, or a signal. It prints one line for each window
that holds work, and keeps running. A window holds work when it has a new
comment or reply whose own marker is `ai?`, `ai!` or `ai:` and which does not
open with gdoc's mark. Everything else in the window is held, not dropped:
gdoc's own replies, a reply with no marker, an edit. Held items ride in the next
line that is printed, so nothing is lost and nothing wakes the session alone.
That rule reads two facts gdoc already reports, the marker and `by_gdoc`, and
judges nothing. The line is short, so the tool that reads it never cuts it: the
window's own `since` and `cursor`, when gdoc noticed it, and for each new item
its id, thread, marker and created time, at most twenty with a count of the
rest. The session reads the window whole with the one-shot `comments --since`,
as it does today. The follow reads the document once, on its first poll, and
judges a window from the listing alone, so a held reply never costs a read. The last line is the
command's envelope, with `waited` as `--wait` writes it, so an interrupt is
still an answer.

**One object per line, for this one mode.** The output contract held one JSON
object per command, with `mcp`'s protocol lines and the help screen as the two
exceptions. `comments --follow` is the third: every line on stdout is one JSON
object, and the last is the envelope that says `ok`. The exit code follows that
last object. Every other command keeps exactly one object.

**The watch is still not a watcher.** It runs inside a call a session started,
it is bounded to an hour, and it dies with that session. "There is no watcher,
and the loop starts when Nail starts it" holds unchanged. A comment written
while no session is open still waits until one looks, and this week measures
how long.

**The panel sees it as a wait, and Stop reaches it.** The record writes a
follow's start with `wait: true`, the field it already has, so the panel shows
it live for as long as its process runs, joins a restart within three minutes
into one session, and ends it at once when it was stopped. Stop sends its one
SIGINT after the same process check as today: the record names `comments` with
a wait, the process is this user's gdoc, and it started when the record says.
The record gains no field, and the app gains no argv list and reads nothing new.

**A short help, and live review in its own session.** `help --brief` prints
the version, the mark in effect and the update notice, a few hundred bytes in
place of the whole table, and the five skills open with it. The review skill
runs a live review under Claude Code's Monitor tool when it has one: one
`--follow` of 29 minutes, restarted when it ends, and never restarted after a
stop. Without the Monitor tool, as in Claude Desktop or an older Claude Code, or
with a binary that refuses `--follow` or `--brief`, it keeps today's `--wait`
loop and today's `help --json`, so no session breaks before its binary updates. When the session already holds other work, the
skill asks the person to open a second session in the same hub folder for the
live review. `--wait` keeps returning every window, gdoc's own replies included.

**The stopwatch.** Three times for every marked comment: written, which is
Drive's time on the comment; noticed, which is the time on the `--follow` line;
and answered, which is Drive's time on gdoc's reply. For comments written while
no session watched, written and answered alone. The lead reads written and
answered at the end of the week from `comments` on the paired documents, read
only, and noticed from the live sessions' transcripts, where each line landed,
and puts the table in `docs/team/measure.md` beside the tokens per live hour.

Out: the job packet, a session per comment, answers with no session open, the
echo filter in `--wait`, and any change to Claude Desktop.

Trust guardian, 2026-10-27: signed. No URL, grant, scope, host or presence
field is added: the follow reads the one handed-in document through the same
`comments.Fetch` and `docs.Fetch` the wait uses, Stop reaches only a `comments`
start with `wait: true` that the unchanged `ProcessCheck.verdict` proves is this
user's gdoc, and the extra stdout lines come from `cmdComments` alone, held by
`TestOnlyFollowWritesLines` beside `TestOnlyJSONObjectRefusesAnythingAfterTheObject`
and `TestStdoutCarriesOnlyJSONRPC`. The hold reads the marker and the mark and
drops nothing, because a follow's cursor moves only when a line is printed.

The plan is `docs/plans/2026-10-27-gdoc-v2-m32-the-quiet-watch.md`.

## 2026-10-28. Four small fixes: the panel drops the published number, a cancelled update leaves the terminal wrapping, a backslash escape publishes as the character, and Tab completion follows the installed gdoc.

Nail looked at the panel after v2.22.0 and wrote: "feedback: no need to show
exact published version number, blue icon is enough". The lead put that beside
three backlog items marked worth fixing, as what he sees before and after, and
asked whether to do all of them. Nail answered "all 4". The lead first sent
the same questions in terms of flags, tests and rules, and Nail answered "даже
на русском ничего не понял" (even in Russian I understood nothing), which is
why `docs/team/lead.md` now has "How to write to Nail". Serves principles 1 and
4.

**The panel drops the published number.** When a newer release is published,
the header shows the update glyph in the accent ink with its dot, and nothing
under the header. The number stays in the glyph's tooltip. The line under the
header is kept for one thing: the answer to a press of the glyph that installed
nothing, such as a release that needs `gdoc update --major`, because that is
the answer to what the person just did. This replaces the "v2.19.0 is
published" line of the third look.

**A cancelled update leaves the terminal wrapping.** `gdoc update` turns the
terminal's line wrap off while its live list moves, and Ctrl-C used to kill it
with wrap still off. Each draw now turns wrap back on when it ends, so a
process killed between two draws leaves wrap on. No signal is trapped, so the
rule that only the comment wait traps a signal holds. Closes
`docs/backlog/ctrl-c-during-update-leaves-auto-wrap-off.md`.

**A backslash escape publishes as the character.** A note that says `\[`,
`\*` or `\{+` publishes as `[`, `*` or `{+`, the way CommonMark and pandoc read
it. One pass over a text run: a backslash before ASCII punctuation writes the
punctuation alone, an `&` starts an entity as before, and anything else is
itself. `\&amp;` is the literal `&amp;`. Code keeps its backslashes. This makes
true what `docs/guide/exporting.md` and `skills/gdoc-export/markers.md`
already say, and lets a link text `export` escaped round-trip. A note that held
a backslash on purpose before ASCII punctuation now needs two. Closes
`docs/backlog/build-never-unescapes-a-backslash.md`.

**Tab completion follows the installed gdoc.** The completion file the
installer writes beside the token becomes a loader of a few lines: at each new
shell it asks the `gdoc` on PATH to write its own completion into a second file
beside it, and sources that. The gdoc doing an update is the old one and cannot
know the new one's commands, so the refresh has to come from the new binary,
and a new shell is the first moment it can. It costs one start of gdoc per new
shell, about 20 to 60 ms, with no network. After this update, a shell that
loads the file starts the `gdoc` on PATH once per new terminal, which it did
not do before; `update` makes that change on a machine where nobody asked for
it, and the `completion` field in its result is where it says so.
`gdoc completion zsh --loader --out <path>` writes the loader. `gdoc update`
replaces the completion file beside the token with the loader when it is a
plain file that opens with the header gdoc writes, and leaves any other file
there alone, so
not knowing still never resolves to overwrite; the write goes through
`internal/atomicfile`. The checkout's `install.sh` and bash keep the static
script they write today. Closes
`docs/backlog/update-leaves-the-completion-stale.md`.

**Out.** Removing `propose`'s ignored `--folder` flag, because a caller may
still send it. A line break that `read` hides, because its fix needs a design
first. Both stay in `docs/backlog/`.

Trust guardian, 2026-10-28: signed. No URL, grant, scope, host, argv list or
presence field is added, and the app's three reads are unchanged: the panel
loses a line of text. The binary still runs no program: the loader is shell
text, the person's shell starts the `gdoc` on PATH, and no file under `go/`
starts zsh, held by `TestNothingRunsAnExternalProgram`, now also refusing
`os.StartProcess` and syscall's exec calls, and `TestOnlyDesktopRunsAProgram`.
`completion` stays offline and unrecorded, so a new shell is not a third command
that asks unasked: `TestCompletionNeverReachesTheCheck` and the reports table in
`TestOnlyTheReadBackWritersCarryVerified`. `update` replaces one file,
`completion.zsh` beside the token, only after a run that installed, only when it
is a regular file that opens with the header gdoc writes or the loader's own
line, through `internal/atomicfile`; anything else is left alone with a warning,
so not knowing still resolves to refusing: `TestAnUpdateLeavesAFileItDidNotWriteAlone`
and `TestAnUpdateLeavesASymlinkedCompletionAlone`. Wrap goes back on with no
signal trapped, so `TestOnlyTheWaitTrapsTheSignal` holds as it is.

## 2026-10-28. The skills say when they are older than gdoc, and the plugin route turns on its updates.

Nail ran a live review and the chat sat on "Running a command 14m", the
foreground wait that M32 replaced. He asked: "i thought we did changes in how
live review works". The session had run `/altery:gdoc-review`, the plugin copy,
and the plugin on his Mac was v2.4.0 from 2026-09-21 beside a v2.23.0 binary.
Claude Code turns plugin auto-update on only for Anthropic's own marketplaces,
and nothing had turned it on for gdoc's. He asked "why its not update
automatically?", then: "Yes: make the install steps turn on plugin auto-update
for colleagues", and "Yes: make each skill notice when it is older than gdoc and
tell the person what to type to update it."

**The plugin route turns on its updates.** The README's first route says to
choose Enable auto-update for the gdoc marketplace in `/plugin`, the
administrator's route puts `"autoUpdate": true` on the marketplace entry, and
the release installer's closing note says the same:
`TestThePluginRouteTurnsOnItsUpdates`.

**Each skill carries the release it shipped in.** A `release` line under
`metadata`, beside `needs`, which `make tag` writes with plugin.json's version
and commits with it: `TestEverySkillCarriesTheReleaseItShippedIn` and
`TestMakeTagRefusesAVersionThatIsNotAMinor`. When the `version` that `help`
prints is a later major or minor, the skill says so once and gives one line to
type, by the folder it was loaded from: `claude plugin update altery@gdoc` for
the plugin, the `--skills` installer line for a copy. It is a remark and never a
gate, the session never runs the line, and no `release` line or no `version`
says nothing: `TestEverySkillSaysWhatToTypeWhenItIsOlderThanGdoc`.

What it cannot reach. A copy installed before this release has neither the line
nor the paragraph, so it says nothing. The first warning lands for a colleague
on the release after the one that ships this, and the README is what reaches
anyone older.
