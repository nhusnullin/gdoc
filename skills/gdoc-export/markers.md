# The markers in an exported file, and how to undo them

`gdoc export` writes everything the document says into one Markdown file. Some of
what a document says is not text: a suggestion is pending, a range carries a
comment, a picture sits beside the paragraph rather than in it. Those arrive as
markers.

The markers are gdoc's own. That is the whole reason a file can sit in the hub
with its markers for a week and still be resolvable: nothing about them decays,
and any later session reads this file and undoes them. `gdoc-export` and
`gdoc-align` both read this page, and it is the only copy of these rules.

## The markers

| Marker | What the document says |
|---|---|
| `{+words+}[s:ID]` | a pending suggested insertion, with its suggestion id |
| `{-words-}[s:ID]` | a pending suggested deletion, with its id |
| `[[c:ID]]words[[/c]]` | the words a comment thread is anchored to, by Drive comment id |
| `[^1]` and `[^1]: text` | a footnote reference, with its text under a `---` line at the end. That one `---` is the separator above the notes and not a page break: it is the last one in the file and the definitions follow it |
| `![](assets/<stem>-<n>.png)` | a picture, as the file export wrote beside the note |
| `<!-- image: floating, <id> -->` | a picture laid out beside the text, after the paragraph it is anchored to. A drawing reads `drawing:`, and an object the read cannot name reads `object:` |
| `[drawing]` `[equation]` `[object]` | content the read does not take, with a warning naming it |
| `[person: Name]` `[date: Sep 9, 2026]` `[link: Title]` | a smart chip, with the label the document shows |
| `[auto text: PAGE_NUMBER]` `[column break]` `[rule]` | the members that hold no text of their own |
| `---` on a line of its own | a page break. The document breaks the page there, `---` is what a note writes for it, and the next build puts the same break back. Nothing to undo: keep the line |
| `[page break]` | a page break the file has no line for: one inside a sentence, or one that is all a heading, a list item or a table cell holds. `---` on those lines would read back as text rather than as a break, so this one stays a marker with a warning: decide where the break belongs, or take it out |
| `[unknown: member]` | a paragraph element gdoc has never seen, named by its member |
| `==words==` | a highlight, in the colour the house gives the bare delimiter |
| `==<emoji> words==` | a highlight, in the colour that emoji names |
| `<!-- words -->` | the grey pencil, which is a comment rather than a highlight. One on a line of its own is a grey paragraph of the document |

The seven marks are not markers to undo. They are the author's own syntax:
written in the note, carried into the document as a colour, and written back
here as the words left. Keep them.

gdoc's own comments are markers and not pencils: `<!-- tab `, and the three a
floating object is written with, `<!-- image: `, `<!-- drawing: ` and
`<!-- object: `, each shut by `-->`. The build refuses a note carrying one by
line, the way it refuses the other markers, so an author's own comment stays a
pencil and gdoc's never comes back as one. A pencil an author wrote is also
written `%% words %%` in a note, which Obsidian hides while somebody reads it;
the export never writes that spelling, only the HTML one.

Say nothing about what a colour means. What a mark means is the author's, said
in their own notes, and the binary gives no colour a meaning. Run
`gdoc help marks --json` once if you need the list: it prints each mark with
its syntax, its colour and whether the envelope counts it, out of the house
style, which is the one place the palette lives. Do not keep a copy of it here
or in a reply.

A highlight the export found in a colour no house mark claims is plain words in
this file, and the envelope counts it under `marks.other` by hex: those words
were marked in the document and nothing in the note will say so.

A mark the export could not write around its own words is plain words too, with
a warning naming the index. The words hold the `==` a highlight is written
with, or a highlight's words hold the `<!--` or `%%` a pencil is written with,
or they hold the `-->` that would shut a pencil in the middle of them, or the
comment a grey run would be written inside would carry one of gdoc's own
comment markers, so the syntax written round them would not be read back: the
file has to build again. Read the warning out and say which words lost their
colour, because nothing in the note will.

A link is `[words](target)`, an external URL as it stands and `#slug` for a
heading in this document. A numbered list is numbered, `1.` and `2.`, and a
sub-list is indented to the column its parent's content starts at. Neither is a
marker: both are ordinary Markdown and both stay.

## Escaping, which is the part to get right

A document may itself contain the characters a marker is made of. An author who
wrote `{+` in a sentence about this very tool would otherwise have their sentence
read as a suggestion. So gdoc escapes a literal marker in the document's own
words with a backslash, and escapes the backslash too:

```
the document says          the file says
{+                         \{+
[[c:                       \[[c:
a backslash \              \\
\{+ the author typed       \\\{+
```

Read the parity, always. The backslashes immediately in front of a marker,
counted: an even run is the author's own text and the marker behind it is gdoc's;
an odd run ends in gdoc's escape and the marker behind it is the document's own
words. gdoc's own markers never carry a backslash, because one in front of a
marker hands that marker to the document.

So when you resolve a marker, you also unescape the text around it: take one
backslash off each escaped pair, and halve every run of backslashes. A file left
with its escapes is a file whose text is not the document's text, which is what
anybody reading the note reads. The backslashes do not reach a document:
`build` and `publish` read an escape in front of a punctuation mark as that
mark alone, so an escaped marker left in the file publishes as the marker's
characters. The note is still wrong to leave as it is.

Inside a link's words a `[` is escaped as well, because an unescaped one there
breaks the link. Inside a table row a `|` the author typed is escaped, because an
unescaped one is a column boundary.

A marker can sit inside a link's words: a comment anchored over hyperlinked
words reads `[[[c:ID]]words[[/c]]](target)`. The brackets a marker is made of
are gdoc's, so step over the whole marker when you are deciding where the link's
words start and end. The link's own brackets are the unescaped ones left.

## How a session undoes each one

**A pending insertion, `{+words+}[s:ID]`.** Somebody proposed adding these words
and nobody has accepted them. Two answers, and the person decides:

- The note takes the words: delete the `{+`, the `+}` and the `[s:ID]`, keep the
  words.
- The note does not: delete the whole thing, words included.

**A pending deletion, `{-words-}[s:ID]`.** Somebody proposed removing these
words. The words are still in the document today.

- The note follows the proposal: delete the whole thing.
- The note keeps the words: delete the `{-`, the `-}` and the `[s:ID]`.

**Either way, Drive is untouched.** Resolving a marker in a file is not accepting
or rejecting anything: the suggestion stays pending where it is, and the
document's owner decides it in the browser. Say that once when you first ask.

**Ask by id and words, never by author.** The Docs API never says who wrote a
suggestion. A session that names an author has guessed.

**gdoc's own pending proposals are not asked about.** When the note's `gdoc:`
block lists an id under `proposals` for this document, those words came out of
this note in the first place: the note already says them. Keep the note's
version, resolve the marker to the note's words, and say in one line how many
were gdoc's own and that they are still waiting in the document. No question.
The export envelope carries `own` beside `pending` when it had a note to check
against, which is the count to report.

**A comment anchor, `[[c:ID]]words[[/c]]`.** Take the two marks off and keep the
words. Always: a comment is not a change to the text. Then list the thread by id,
author and first line, from `gdoc comments <url>`, and say the comments stay in
Drive. A comment's text never goes into a note body.

**A footnote, `[^1]`.** Keep it where it is, marker and definition both. A note
holding a footnote in its prose publishes: `gdoc build` writes it under the
page and Drive imports it, so nothing has to be moved into the body. Where the
mark came home inside a heading or a table cell, `# Title[^1]` or `| x[^2] |`,
it has to move: Docs allows a note there and the house style carries none, so
`build` refuses that line. Decide with the person where the mark goes. The
warnings that ride with a footnote all matter: a suggestion inside one exported
as plain text and carries no id, a comment anchored inside a footnote, a header
or a footer got no mark at all, and where the export says so, a mark the author
wrote inside a note came home as plain words, counted nowhere, so ask whether
to mark them again. One thing is not the author's own: the numbers are the
document's, so a label somebody wrote as `[^method]` comes home as `[^3]`, and
a note marked twice in the document comes home as two definitions carrying one
text. Renaming them is the person's call, not yours.

**A picture line.** Keep it. The file is already beside the note under
`assets/`. When the picture carried `matched`, export wrote no file and the line
is the note's own link: leave that alone, because the note's own picture may be
rendered from an SVG that is the master.

**A floating picture's comment.** `<!-- image: floating, <id> -->`,
`<!-- drawing: floating, <id> -->` or `<!-- object: floating, <id> -->` says
something sits beside the text there.
The file for it is written. Decide with the person whether the note wants the
picture as a line at that place, then delete the comment.

**A placeholder in brackets.** `[drawing]`, `[equation]`, `[object]`,
`[page break]` and the rest are content the read could not take. Each one has a
warning naming it. Show them and ask what the note should say there. Never leave
a bracket in a note as if it were text.

**A chip.** `[person: Name]`, `[date: Sep 9, 2026]` and `[link: Title]` are smart
chips, and the export puts the address behind the label where there is one. A
note cannot hold a chip, so write what the chip meant in words, or a plain link.

## Before the file goes anywhere

A marker is refused on every route out of the hub, by line:

- `build` and `publish` refuse a body carrying one, and name the line.
- `reply` and `annotate` refuse one in a comment body.
- `propose` refuses one in its `--from` file.

That refusal is the safeguard, not an obstacle. It is what makes keeping a marked
file safe: nothing can carry a forgotten marker into a document as prose. When
one of those commands names a line, the answer is to resolve that marker, not to
delete the refusal's reason.

An escaped marker is the author's own text and is refused by none of them, so a
note written about the markers themselves publishes fine.
