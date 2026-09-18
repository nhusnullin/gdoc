# Reading a document: text, comments and suggestions

This page holds the three commands that read a document and write nothing to
Drive. It is for somebody who wants to know what `read`, `comments` and
`suggestions` print.

## Reading a document

Four commands read, and none of them writes to Drive. Each one takes the URL
you paste from the browser, or a bare document id.

```bash
gdoc read <url> [--structure]
gdoc comments <url> [--since CURSOR] [--wait DURATION] [--witness]
gdoc suggestions <url> [--md PATH]
gdoc restyle <url> --dry-run
```

They print facts and nothing else. Whether a comment is answered, whether a
suggestion that disappeared was accepted or thrown away, whether the document
and your markdown have drifted apart in a way that matters: none of that is
decided in Go. The skill reads the JSON and judges.

**`read`** gives you the document as one string of text, with everything that is
pending marked in it:

| In the text | Means |
|---|---|
| `# Heading` | a heading, one `#` per level |
| `{+text+}[s:ID]` | a pending suggested insertion, and its id |
| `{-text-}[s:ID]` | a pending suggested deletion, and its id |
| `[[c:ID]]text[[/c]]` | the text a comment is attached to, and the comment id |
| `<!-- tab t.0: Title -->` | the tab that follows, on a document with more than one |
| `[image]`, `[drawing]`, `[equation]`, `[object]` | content that is not text yet |
| `[person: Ada Lovelace]`, `[date: Sep 9, 2026]`, `[link: Q3 planning]` | a smart chip, with the label it shows |
| `[auto text: PAGE_NUMBER]`, `[page break]`, `[column break]`, `[rule]` | the rest of what a paragraph can hold |
| `[unknown: member]` | something in the document this version of gdoc has never seen |

Every placeholder row in that table comes back with a warning naming what the
read did not take from it, the chips and the breaks included. A policy with
eight person chips and three page breaks in it answers with eleven warnings, so
a non-empty `warnings` list does not on its own mean the read went wrong: read
the messages rather than counting them.

If the document's own text contains one of those markers, it comes back with a
backslash in front of it, and a backslash the author typed comes back doubled.
So the rule for reading the text back is a parity: an even run of backslashes is
the author's own text and the marker behind it is gdoc's, an odd run ends in
gdoc's escape and the marker behind it is the author's. The escaping is done one
text run at a time, so a marker whose two halves fall in two runs, or a document
character sitting against one of gdoc's own markers, can still reach the text
unescaped. That gap is written up in
`docs/backlog/escaping-across-run-boundaries.md`. `[object]`
is an embedded object the read could not classify: calling it an image would be
a guess.

Smart chips are read since M7, and before M7 they were not read at all: a person
chip, a date chip and a calendar link were dropped without a placeholder and
without a warning, so a policy naming its owner through a chip came back naming
nobody and nothing said so. Six other things went the same way, including page
breaks and horizontal rules. They all print a placeholder now. A chip's
placeholder carries the label the document shows, so `[person: Ada Lovelace]`
tells you who is there; the email address and a link's target are in
`--structure`. `[unknown: member]` is the wider half of the same fix: something
in the document this version of gdoc has never seen, named by what Google calls
it, rather than silently absent. Tables become pipe tables, with a literal `|` in a cell escaped as
`\|` so the row keeps its shape, and footnotes are appended after a `---` line.
Lists come back as `- ` items, two spaces of indent per level, so a numbered
list reads back as a bulleted one: telling the two apart needs the document's
`lists` map, which gdoc does not read. `--structure`
adds the document tree with character indexes on it, which is what placing a
suggestion at an exact position needs. The text is not a
summary of the structure, and the structure is not a summary of the text.

**`comments`** lists the threads. Each one carries its author, its content, its
replies, whether it is resolved, the sentence it quotes, and the character range
the Docs read placed it at. A thread the Docs read could not place comes back
with `range: null` and a warning, rather than failing the whole listing. The
marker is a fact too: `ai:`, `ai?`, `ai!` or `none`, taken from the first word.
`@ai` is not one of them: the three are matched exactly.

`--since` takes the `cursor` a previous run printed and asks for what changed
after it. The cursor is opaque: it is the newest activity that run saw, encoded,
and nothing reads inside it. Every listing prints one, so a live session can
start on any document: where the run saw no activity at all, which is a document
nobody has commented on, the cursor is dated from the run's own clock a few
minutes back rather than from anything anybody did. Nothing writes it down either, so it lives as long
as whatever is polling.

`--wait` turns that one call into a poll. The binary asks Drive every ten
seconds inside the call, and comes back the moment something happened after the
cursor, or at the deadline with an empty window. It needs `--since`, because a
wait with no cursor answers with the whole document, which is the plain listing
under another name. The value is a Go duration, `9m` or `90s`, and an hour is
the most one call will look for. A wait keeps nothing of its own: the cursor it
prints is all that carries to the next call, and the only file it can touch is
the saved OAuth token, which any command replaces when the access token has to
be refreshed.

With `--wait` the object carries one more field:

```jsonc
"waited": { "polls": 12, "seconds": 118, "interrupted": false }
```

`polls` is how many times it asked, `seconds` how long it looked. A run without
`--wait` has no `waited` field at all. Ctrl-C during a wait is an answer rather
than a crash: the object comes back `ok: true` with no threads, the cursor you
handed in, and `interrupted: true`, and the exit code is still 0. A poll that
failed ends the wait with `ok: false` and says how many polls it made, so
whatever is looping can say the window is unread rather than empty.

`--witness` reads the document a second time, as a docx export, and says of each
thread whether that export carries it `anchored` to text, `detached` from it, or
`unmatched`. The export is the only truthful answer to "is this comment still
attached to anything", which is why it is a second read rather than a field.

**`suggestions`** lists what is pending, each with a stable id, the heading it
sits under and its text. With `--md` it also reports what stopped being pending
since the last look, by comparing against the snapshot in that file's front
matter, then writes the new snapshot. That write happens only after a read that
fully succeeded, and only into a file already paired with the document you read.

