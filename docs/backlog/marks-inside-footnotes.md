---
worth: yes
added: 2026-10-15
---
# A mark inside a footnote comes home as plain words

`internal/body` shades a mark inside a footnote and counts it: `footnoteRuns` walks a definition's
inlines through the same `inlineRuns` every paragraph goes through, and `countMarks` walks the whole
tree, definitions included. So `[^1]: ==to fill in==` reaches the docx as a yellow run and adds to
`body.marks.yellow`.

Export cannot read it back. `docs.Document` holds `Footnotes map[string]string`: `addFootnotes`
flattens each note through `plainText`, which keeps the run texts and drops `Background` and
`Colour`. `view.appendFootnotes` writes that string, so the projection's `Marks` option never reaches
a footnote's runs, and `export.otherColours` walks `t.Body` alone. The note comes home as
`[^1]: to fill in`, the mark is gone, and the next build reports one debt fewer than the document
holds.

Since 2026-10-15 the read records the colours each note's own runs carried, on
`docs.Document.FootnoteColours`, and `export.markedNotes` asks whether they say somebody marked
words, which is any background at all or a text colour the palette claims, before
`export.footnoteWarnings` names the loss. So the hole is no longer silent, and a note
coloured only by its own link is not called one. What is left is carrying the mark itself, and it is
the same choice `docs/backlog/suggestions-inside-footnotes.md` names:

- Keep each footnote's blocks on `docs.Document` rather than its text, and project them through the
  same mark grouping the body uses. Correct, and it changes a shape `view` and `suggestions` both
  read. Doing it once would close both backlog items.
- Leave the words plain and keep the warning. A document whose debts live inside its footnotes
  cannot be round-tripped, and the session is told so rather than finding out from a count.
