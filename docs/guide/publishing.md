# Publishing a note: build, publish and the gdoc: block

This page holds the render of a note into the house style, and the upload into
Drive. It also holds the `gdoc:` block the note carries afterwards. It is for
somebody who wants to know what the front matter does and what the run checks.

## Building the document

`gdoc build` turns a note into an Altery house-style `.docx` on your own
machine. It reaches nothing: no Drive, no Docs, no network at all, and it runs
no other program. `gdoc publish`, below, is the same render with the upload
behind it.

```bash
gdoc build --md note.md --out note.docx [--house house.yaml] [--force]
```

The house style is a file, `house.yaml`, and it is embedded in the binary, so
there is still nothing to install beside it. It states the page geometry, the
nine named styles, the cover, the header and footer with the logo, the three
front-matter tables cell by cell, the legend, the contents field and the heading
numbering. `--house` points at another copy of that file for one run, which is
how a change to the style is looked at before it is committed; the output names
which one was used, `embedded` or the path, so a document built from a draft
says so.

`--out` will not overwrite a file that is already there. Add `--force` when you
mean to replace it. `--force` is consent to replace a document, not a folder and
not something the run reads: an `--out` naming a directory, the note, the
`--house` file or one of the note's own pictures is refused whatever the flag
says. The write goes through a temporary file and a rename, so a failed build
cannot truncate a document you already had.

The note needs a `title` in its front matter and nothing else. These are the
keys it reads:

| Key | Does |
|---|---|
| `title` | required. The cover, and the running head in the page header |
| `alt_title` | the second title line on the cover, under the word `or`, and the running head. A note that states none prints neither line |
| `doc_type` | joined to the title on the cover, so `Third Party Risk` plus `Policy`. Free text, and a title that already ends in its own type is left alone |
| `version` | the cover, behind the word `Version:`. `1.0` when the note states none |
| `date` | rendered on the cover in UK long form. The month the build runs in when the note states none |
| `owner` | the Document Owner row of the version-control table |
| `last_approval` | the Date of Last Approval row |
| `review_frequency` | the Review Frequency row |
| `board_ratification` | the Board Ratification Date row |
| `distribution` | the Policy Distribution row |
| `classification` | one of `confidential`, `restricted`, `internal`, `public`. It shades that class's row in the classification table, and the rest are left clear. `internal` when the note states none |
| `heading_numbering` | `false` turns off the `1-Scope` numbering on level-one headings |
| `revisions` | rows of the revision-history table: `version`, `date`, `author`, `approved_by`, `approval_date`, `section`, `change`. A note that declares none keeps the template's own row |

A note with no `title` is refused, and the refusal proposes one: the first
heading in the body, or the file name. The binary never invents a title and
never writes one into your note. Any other key in the front matter is carried
untouched, the `gdoc:` block included.

The body is your markdown: headings to six levels, bulleted and numbered lists
three deep, bold, italic, strikeout, `==marked==` text as a highlight, links,
pictures, tables, horizontal rules and block quotes. A picture is read from the
note's own directory, or decoded when the markdown carries it as a `data:` URI;
PNG and JPEG. A picture at an `http` address is refused naming the line, because
a document built from a link is a document that breaks when the link expires.

Code blocks are not rendered. The house style has nothing to render them in, so
a note carrying one gets a warning naming the line and the block is left out
rather than dropped in silence. Blocks of HTML, inline HTML and footnotes are
the same answer. Footnotes matter more than they look: `gdoc read` writes a
document's footnotes as `[^1]` in the prose and the definitions after a `---`
line, so a note pulled out of a Doc carries them, and each one is named by the
line the author wrote it on. So is a picture inside a list item, a block quote or a table cell: the
house style puts a figure on a centred line of its own, which it cannot be
there, and the warning names the line.

Every numbered list gets its own definition in the document, so a second
numbered list starts again at 1 rather than carrying on from the first. A list
nested inside a numbered item shares its parent's, which is what makes it
restart under each item the way you wrote it. One thing about a numbered list
still carries a warning rather than the numbers you wrote: every level starts at
1, so a list you opened at "5." opens at 1, and the warning names the line. A
heading that skips a level is the other: a `###` under a `#` is numbered
`1.0.1-`, and the warning carries the number it wrote.

A list item that holds one of those and nothing else, or nothing at all, has no
words to put a marker on. It takes none, and the warning names the line and
says what that costs: in a numbered list the items after it print one lower, and
in a bulleted list the item loses only its own bullet and its indent.

What you get back is one JSON object: the file it wrote and its size, the title
and the running head it used, which house file it read, and what the walker
counted.

```json
{
  "ok": true,
  "data": {
    "out": "/Users/you/notes/supplier-register-policy.docx",
    "bytes": 48213,
    "title": "Supplier Register Policy",
    "running_head": "Altery - Supplier Register Policy",
    "house": "embedded",
    "body": {"paragraphs": 41, "headings": 9, "lists": 3, "tables": 2, "images": 1}
  },
  "warnings": ["line 88: a code block is not rendered in the house style and was left out"]
}
```

Those counts are facts and nothing more. Whether the document is right is
answered by opening it, in Word or in Drive, and not by the binary.

The contents list is a Word field, which is what the master template carries
too. Word fills it in when the document is opened and refreshed, and Google Docs
turns it into a live contents list on import. Until then it shows the one line
Word writes into an unrefreshed field.

There is one test that keeps this honest on every commit. It builds a document
from the embedded style, opens the Word master the style was extracted from, and
compares 169 measured values across the two: page geometry, all nine styles, the
header, the footer, the logo's position, the contents field, the three tables
and the body. Every row that differs, and every row the master has nothing to
compare against, is named in a list with the reason it is there, most of them
because the master states a value twice and because the two documents hold
different words. A row inside its own tolerance reads as close and needs no
entry. Any other difference fails the test suite, so the style cannot drift away
from the master quietly.

## Publishing the document

`gdoc publish` does what `build` does and then puts the result into Drive as a
Google Doc, in one folder you name, and writes the pairing back into your note.

```bash
gdoc publish --md note.md --folder-id FOLDER [--house house.yaml]
```

The note is rendered exactly the way `build` renders it, so everything in the
section above applies: the same front-matter keys, the same warnings, the same
refusals. The docx bytes are uploaded with conversion, so Drive turns them into
a document rather than leaving a .docx sitting in a folder.

The folder is the only thing the run can reach. No document is in reach when it
starts, and the new document's id comes back from the create the tool itself
made. There is no `--folder-id` default and no fallback to the folder in your
note: a note that already names a document is refused before anything leaves
your machine. There is no second-version command, and `restyle --new` is not
being built. To publish a note again, take the `gdoc:` block out of it by hand
and run `publish` once more; the refusal message says the same.

Three things are checked after the upload, and each answers something the other
two cannot: the document reads back through the Docs API, it has exactly one
tab, and Drive can export it as a .docx again. `verified` is the three together.
Fewer than three is still `ok: true` with the check that did not hold named,
because a document that exists is a document that exists, and being told the run
failed is what makes somebody upload a second one.

If the upload worked but your note could not be written, the document goes back:
it is trashed and the trash is confirmed, and the output says `rolled_back:
true` with no document id. Only if that also fails do you get the live id and
what to do with it. The reason is not tidiness. A document nobody's note points
at is one the next publish makes a second of.

```json
{
  "ok": true,
  "data": {
    "document_id": "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
    "folder_id": "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr",
    "url": "https://docs.google.com/document/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd/edit",
    "title": "Supplier Register Policy",
    "house": "embedded",
    "bytes": 48213,
    "tabs": 1,
    "verified": true,
    "checks": {"read_back": true, "one_tab": true, "docx_export": true},
    "files_changed": ["/Users/you/notes/supplier-register-policy.md"],
    "body": {"paragraphs": 41, "headings": 9, "lists": 3, "tables": 2, "images": 1}
  },
  "warnings": []
}
```

Your note is read again just before it is written, because the upload takes
seconds and these notes live in a synced vault. If it changed in that window, in
its body or in its front matter, nothing is written and the document is rolled
back: the file on disk is no longer the one that was rendered, and publishing it
as though it were would pair your note with a document it does not match.

There is a second test that only runs when you ask for it, and it is the one
that means something: it builds the document, uploads it and the Word master
side by side, reads both back through the Docs API, and compares the same 169
values. Google's import is part of that reading, so a difference that shows up
there and not offline is Drive's doing rather than the generator's. It creates
two real documents and trashes them, so it is behind two environment variables
and never runs by itself.

## The `gdoc:` block

A markdown note paired with a Go-published document carries one key in its front
matter, and everything else in there stays the author's:

```yaml
---
title: Supplier register policy
gdoc:
  schema: 1
  document_id: 1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd
  folder_id: 1w0SresizE9Kr810VZRJwX4JtDBF4OqNr
  published:
    at: 2026-09-08T10:14:00Z
    title: Supplier register policy
    house: embedded
  suggestions_seen:
    at: 2026-09-06T11:00:00Z
    items:
      - id: suggest.abc123
        kind: insertion
        section: Scope
        text: "critical "
---
```

The read is strict. An unknown key, a key given twice, a missing `document_id`
or a `schema` this version does not know is refused naming the key, and the file
is left alone. The write touches the `gdoc:` lines and nothing else: your keys,
your line endings and the trailing newline come back byte for byte, and the file
is replaced through a temporary file and a rename, so a failed write cannot
truncate your note.

`gdoc publish` is the only thing that creates this block. `gdoc suggestions
--md`, `gdoc propose --md` and `gdoc withdraw` update it, and each of them
refuses a note that does not carry one already. A note whose `gdoc:` key is a
bare string rather than a block is refused by name: gdoc tells you to rewrite
the line by hand once, or to publish the note again with `gdoc publish`.
Building is unaffected, because `gdoc build` skips the `gdoc:` key whatever is
in it.

