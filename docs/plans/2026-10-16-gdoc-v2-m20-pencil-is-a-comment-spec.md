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

Filled in when the run finishes, one row per rule above, pass or the test
that says otherwise.
