# Exporting a document: the file, the pictures and the marks

This page holds `gdoc export`: what it writes, where it puts it, what the marks
in the file mean, and what it refuses. It is for somebody who opened an exported
file and wants to know what they are reading. Publishing is the other direction,
and it is on [its own page](publishing.md).

## The command

```bash
gdoc export <url> --out <file>
```

It reads the document twice, through the Docs API for the words and through
Drive's docx export for the picture bytes, and writes Markdown into your hub. It
changes nothing in Drive. It sends no request that could: the two reads are the
same two `gdoc read` and `gdoc comments --witness` already make.

You do not normally type it. `/gdoc-export` runs it for a document you have no
note for, and `/gdoc-align` runs it for a note that is already paired with one.

## Nothing is replaced, ever

A path that is taken takes the next free number. `note.md` is there, so the file
lands at `note.2.md`; that is there too, so the next one lands at `note.3.md`.
There is no `--force` and no other flag that could replace a file. The reply
names every file it wrote, so you always know which one is new.

That rule holds for pictures too. Each picture lands in an `assets` folder
beside the file, as `assets/<stem>-1.png`, then `-2`, counting up from the first
free number. The folder is created when it is missing. A picture already sitting
under one of those names is never touched.

If you want the old note gone, export to the new file, read the old one, and
rename by hand. The one thing gdoc will not do is guess that you meant to lose
something.

## One file per tab

A document with one tab gives one file. A document with three gives three. The
first tab lands at `--out` and each other tab lands beside it, named from its
own title: a tab called `Appendix` beside `note.md` is `note-appendix.md`. The
title is lowercased and every run of characters that are not letters or digits
becomes one hyphen, which is the same rule a link to a heading follows. A tab
with no title that can be spelled that way is `note-tab-2.md`, by its position.
Two tabs with the same title take the numbering rule like any other taken name.

Pictures follow their own tab. Each file's `gdoc:` block names the document and
the tab it came from.

## What the marks in the file mean

The file is the document as it stands, including what is still being argued
about in it. Pending suggestions and comments cannot be left out, because the
words they cover are the words you are reading.

| In the file | Means |
|---|---|
| `{+words+}[s:ID]` | somebody has suggested inserting these words, and the suggestion is still pending |
| `{-words-}[s:ID]` | somebody has suggested deleting these words |
| `[[c:ID]]words[[/c]]` | these are the words a comment thread is attached to |
| `<!-- image: floating, kix.p1 -->` | a picture laid out beside the text rather than in it, after the paragraph it is anchored to |
| `![](assets/note-1.png)` | a picture, written as a file |
| `[^1]` | a footnote, with the notes under a rule at the end of the file |

A backslash makes the one character after it the document's own. So `\{+` is
text somebody typed into the document and `{+` is gdoc's mark. That rule has no
exceptions and runs across the whole file, so you can undo the marks by reading
the backslashes and nothing else.

**A marked file cannot become a document.** `build`, `publish`, `propose`,
`reply` and `annotate` all refuse a file carrying one of those marks, naming the
line. That is the whole cost of keeping them: nothing else in the hub minds
them. There is no deadline to resolve them and no command that expires them. The
marks are gdoc's own, so any later session can take them out for you, on the day
you have time. Ask for that in words, and the session reads the document once
more first, so a suggestion that was decided in the meantime is not put to you
again.

Comment threads live in Drive and never come into the file as text. The anchors
say where they are; `gdoc comments <url>` says what they say.

## The house cover comes out

A document `gdoc publish` made opens with the house cover, the three
front-matter tables, the legend and the contents list, and its headings carry
their `1-` numbers as plain text. None of that is your writing, and a note built
from a file that kept it would be numbered twice over. So export takes it out
and lists what it removed, with the text each piece held, which is how the skill
can tell you that somebody edited the Document Owner cell.

It recognises the prelude by its position and never by its words, so an edited
cover title still comes out. A heading somebody added inside the cover does not
match, and neither does a tab with no level-one heading at all, and then nothing
is stripped: the whole document stays in the file and a warning names what stood
there. Stripping on a guess would take somebody's own front page out of their
note, and nothing puts it back.

The `1-` numbers come off only where the layout of a published document was
recognised. That is the one route that wrote them: `gdoc restyle` never numbers
a heading, so a document it styled keeps every heading as it stands, and so does
a document with no house layout. `2024-2025 Budget` and `1-on-1 meetings` stay
your words.

The layout is a position and not a proof. A document of your own with a title
page, three tables and a contents list in front of its first Heading 1 looks
like a published one, so its front page is stripped and a heading prefix comes
off with it. Nothing goes quietly: `stripped` lists every number the run took,
with the whole heading as it stood, so you can put it back.

When the prelude does not match, the numbers stay too, and a second warning
names each heading that opens with digits and a `-`. It does not say whose they
are, because it cannot: the same shape reaches a published document whose cover
somebody edited and a document gdoc never touched. If `gdoc publish` wrote the
document, take those numbers out with the cover. If it did not, they are the
author's words and they stay.

## What it writes into your note

One date. When `--out` is a note whose block already names this document, that
note keeps every other byte and gains `exported: {at}` on that document's entry.
The export itself lands beside it under the next free number, carrying
`exported: {note: <path>}` in its own block, which makes every writer refuse
that copy in one sentence pointing at the note. That field is there so a copy
left behind by a session cannot quietly collect proposals. To keep the copy as a
note of its own, delete the `note:` line by hand.

A file export creates from scratch carries the `gdoc:` block and nothing else in
its front matter. It has no `title` yet, so it cannot be built or published
until somebody adds one, which is what `/gdoc-export` does before it hands the
file over.

## What it refuses

Two things, both before a single byte is written, and both of them about the
path rather than the document:

- A note at that path whose block names other documents. The refusal names the
  ones it does hold. Every `--md` command refuses the same way.
- Front matter at that path that does not read. The refusal names what is
  broken.

Every tab's path takes the same two checks, so one bad path refuses the whole
run rather than leaving half a document in the hub. Anything else at a path is
simply a taken path, and the run goes to the next free number.

## What comes back

```json
{
  "ok": true,
  "data": {
    "document_id": "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
    "title": "Supplier Register Policy",
    "revision_id": "ALm37BW...",
    "tabs": 1,
    "multi_tab": false,
    "files": [{"path": "/Users/you/notes/supplier-register-policy.2.md",
               "note": "supplier-register-policy.md"}],
    "pictures": [{"file": "assets/supplier-register-policy-1.png"}],
    "pending": 3,
    "own": 1,
    "threads": 2,
    "stripped": [{"kind": "cover", "text": "Supplier Register Policy"},
                 {"kind": "table", "text": "Document Owner ..."}],
    "stamped": [{"path": "/Users/you/notes/supplier-register-policy.md"}]
  },
  "warnings": ["the match against the note's own pictures is off until an exported picture is measured byte for byte against the one publish uploaded, so every picture here was written as a new file even where the note already holds those bytes"]
}
```

`pending` is how many suggestions are still open in the document, and `own` is
how many of those gdoc proposed itself, which it knows only when the note lists
them. `threads` is how many comment threads there are. `stripped` is the prelude,
piece by piece. Every one of those is a count and nothing more: whether a
difference matters is the skill's judgement, with you, and no field here answers
it.

A warning rides with anything that could not be carried: a picture with no
bytes, a footnote, a prelude that was not recognised, a document whose two
routes disagreed about how many pictures it holds. When they disagree no picture
file is written at all, because a pairing one place out would put the wrong
bytes under the right name and nobody reading the note afterwards could see it.

The warning above is the standing one, and it is not about your document. A
picture the note already holds should stay the note's own file, so the SVG it
was rendered from stays the master. That rests on a published PNG coming back
from Drive byte for byte, which nobody has measured yet, so the match is off and
every picture is written as a new file. The warning says so rather than saying
nothing.
