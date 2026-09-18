# Restyling a document where it stands

This page holds the survey, the in-place restyle and the house template added as
a suggestion. It is for somebody who wants to know what a restyle changes, what
it protects, and what it cannot do.

## Surveying a document before you restyle it

```bash
gdoc restyle <url> --dry-run
```

This says what a document holds before anything is done to it, and it writes to
no document and to no file:

```jsonc
{
  "schema": 1,
  "document_id": "1AbC...", "revision_id": "ALm37BX...",
  "title": "Supplier Register Policy",
  "tabs": 1,
  "threads": { "open": 3, "resolved": 7,
               "witness": { "anchored": 9, "detached": 1, "unmatched": 0 },
               "witnessed": [ { "id": "AAABc...", "witness": "anchored", "resolved": false } ] },
  "suggestions": { "pending": 2, "on_elements": 0, "ids": [ "suggest.abc", "suggest.def" ] },
  "chips": { "person": 4, "date": 1, "rich_link": 2 },
  "named_ranges": [ { "id": "kix.abc123", "name": "gdoc-checklist" } ],
  "nothing_to_protect": false
}
```

A run gives `--dry-run` or `--from`, never both. The survey is what makes the
restyle safe, so it has to be something you read before you asked for the write,
rather than something the same run produced a moment earlier and never showed
you.

The witness is per thread as well as counted, because it is a before picture:
restyling a document can detach a comment, and a thread that was already
detached beforehand would otherwise look like damage the restyle did. It has the
same two limits it has under `comments --witness`: it finds a destroyed anchor
and not a moved one, and two comments with the same words that disagree give no
answer for either.

`nothing_to_protect` is true only when there are no threads, nothing pending, no
chips, no named ranges, and nothing in the document gdoc could not name. It is a
fact about those five, not advice: whether a document is worth restyling is
yours to decide from the counts. A named range is in that list because it is a
label Docs keeps in step with its own edits, so a replacement of the words it
covers takes it with them. The pending count includes a suggestion whose text is only
whitespace, which the `suggestions` listing leaves out, because it is still
something a rewrite would destroy.

Pending is two numbers, and they are counted in different units, so do not add
them together. `pending` counts the entries the pending walk finds, one per
suggested insertion and one per suggested deletion, so a replacement counts as
2 there. It is not the length of the `suggestions` listing: that listing leaves
out the whitespace-only ones this count keeps, as the paragraph above says.
`on_elements` is counted as ids, so the same replacement counts as 1.

`on_elements` is the suggestions that listing cannot report. It reads text runs
alone, so a suggestion carried only by a run it skips reaches it in no form: a
chip, a page break, a horizontal rule, an auto text, a picture, a footnote
reference. Counting nothing for them would answer "nothing to protect" over a
change somebody is waiting on. A run with one is still printed by `read`, inside
its markers, with its id.

`revision_id` is what the restyle rechecks. Hand this file back with `--from`
and the run refuses the document if it has moved since, and every batch it sends
carries that revision so Docs refuses it too.

`schema` says which shape this survey is, and `--from` refuses one that states
another version or none at all: take the survey again with the gdoc you are
applying with. That is not tidiness. The apply reads this file as the record of
what the document held, so a field a newer gdoc compares against and an older
survey never wrote would read as nothing having been lost. `ids` is that field:
without it a suggestion the run destroyed reads as one that was never pending.

An export that could not be read is a warning, not a failure: you still get the
threads, with every witness reported `unmatched`. A Docs read that failed is a
failure, because the chips, the suggestions, the ranges, the tab count and the
revision id are all in that one read.

## Restyling a document in place

```bash
gdoc restyle <url> --dry-run > survey.json
gdoc restyle <url> --from survey.json
```

Two runs. The first surveys, the second gives that document the Altery house
style where it stands: the page size and margins, each paragraph's spacing and
indent, each run's face, size and colour, and each table cell's padding and
borders.

**This is the one thing gdoc does to a document you handed it.** Everywhere
else, every change is a suggestion you accept or throw away. Here the change is
made. What holds it in is narrow: the run styles the one document you named and
the survey agrees with, only if the document has not moved since the survey, and
the four request kinds it may send cannot change a character of what you wrote.
The grant lasts one run and is not written anywhere.

**A restyle is a moment, not a setting.** The Docs API cannot redefine a
document's named styles, so gdoc applies the look paragraph by paragraph. The
document looks right afterwards, and the next heading you type is Google's
Heading 1 again, not the house one.

**What it overwrites is formatting inside the paragraphs it styles.** The face,
the size and the colour of every run go to the house value. Bold and italic are
left alone, and so is a table cell's own fill: the house style does not say
whether they should be off, and clearing them would be gdoc guessing. Your
words, your comments, your pending suggestions, your smart chips and your named
ranges are not touched.

**A failed run leaves a half-styled document.** There is no rollback, and the
recovery is the document's own version history, by hand. No text was touched, so
nothing you wrote is lost, but your own run formatting inside the paragraphs
that were restyled is. The run says so in its warnings.

Afterwards it reads the document back three ways and reports two things. What
survived: your threads with a witness for each, your pending suggestion ids and
your chips, before against after, with anything that moved named and never
explained. And whether the style is really there: it reads a margin, a
paragraph's spacing, a run's font and a cell's appearance back out of the
document and says which ones landed. `verified: false` is not a failure. What
was applied is in the document either way, and the field says the read-back
could not confirm all of it.

The page is the one check that reads more than the value it sent. If your
document has section breaks that set their own margins, or turn their own pages
on their side, those are what a reader sees, and gdoc cannot change them:
`updateSectionStyle` is not one of the four kinds it may send. So the check says
the page geometry is unconfirmed and names the fields your sections set
differently, rather than reporting a page as restyled because the value came
back. A section restating the margin gdoc asked for is not one of them, and
neither is one turned the same way as the rest of the document: they override
nothing you would see.

A run that stopped because Docs took a batch and the answer could not be read
still reads the document back, and says the batch may be in it. It is the run
those facts are needed for most: what may have landed there is a direct edit.
Every request that reached Docs is read back, that batch's included, and the
checks read the first request of each kind, so on a run that stopped after
several batches they usually answer for the earlier ones. `verified` stays
false either way, because the last batch was never confirmed.

`manual` is what gdoc could not do, each with the menu path: the first-page
header with the logo, the contents list, the footer page numbers, plus any list
it left alone, because the request that sets a bullet also deletes the tabs that
set its nesting level, and any table's column widths, and any paragraph whose
named style the house has no look for.

A document with more than one tab is refused before anything is sent. A style
request names a range, and a range means nothing without saying which tab it is
in.

## Adding the house template as a suggestion

```bash
gdoc restyle <url> --dry-run > survey.json
gdoc restyle <url> --from survey.json --fields fields.json
```

The same two runs, with one flag more. `--fields` adds the house cover, the
three front-matter tables and the legend to the styling above, and **it adds
them as a suggestion**. You accept them in the browser the way you accept any
suggestion, or you reject them and the document is exactly as it was.

That is the whole design rather than a caveat on it. The run has two
permissions, not one. The template goes out in suggesting mode on a connection
that was granted nothing at all, which is the same thing `propose` does every
day. The styling goes out on a second one holding the direct-edit grant, for the
four request kinds that cannot change a character. Neither half can do the
other's job.

`fields.json` is the cover's own values, and it is the same thirteen a note's
front matter carries:

```json
{
  "title": "Payment Services Policy",
  "doc_type": "Policy",
  "version": "1.2",
  "date": "September 2026",
  "owner": "Head of Compliance",
  "classification": "Internal",
  "revisions": [
    {"version": "1.2", "date": "2026-09-01", "author": "N. Khusnullin", "change": "annual review"}
  ]
}
```

A restyle has no note behind it, so somebody has to write this file. `title` is
required and gdoc will not invent one: a file without it is refused, and the
skill asks you. A key it does not know is refused by name rather than ignored,
because a misspelled key is a cover line that would silently never print.

**Run it twice and you get one cover, not two.** gdoc puts a named range called
`gdoc:house-prelude` over what it proposed, and that marker is its whole memory
of having been here. A second run finds the marker and proposes replacing what
it covers. If you rejected the first prelude the marker went with it, so the
second run proposes cleanly. If the first prelude is still sitting there
unaccepted, the run refuses and tells you to accept or reject it first: it will
not propose deleting text that has never been written.

The marker is the one thing written directly rather than suggested, and only
because Docs refuses to apply it as a suggestion. It adds and removes no
character.

Afterwards gdoc reads the prelude back and asks three things: whether every
piece of it carries a suggestion id, whether the marker covers what was
proposed, and whether your own text is character for character what it was.
`verified` is all three together. `verified: false` is not a failure, the same
as everywhere else.

The template's own `manual` list is reported beside the styling one, under
`prelude`, and it names two things: the contents list, which no Docs request can
create, and the column widths and row heights of the front-matter tables, which
need two request kinds nobody has measured as suggestible, where a request Docs
refuses takes the whole batch with it. The contents list is in both lists,
because the styling half names it on every run. Heading numbering is not
in this yet, and it is not blocked either: it needs its own answer to what a
second run should do about a number gdoc already wrote.

If the template phase fails, the styling phase does not run, and the report says
which phase stopped. Whatever of the template reached the document is a
suggestion, so rejecting it puts the document back.

