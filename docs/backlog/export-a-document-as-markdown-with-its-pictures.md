---
worth: later
added: 2026-09-18
---
# Export a Google Doc as a readable Markdown file, with its pictures beside it

A colleague's idea, passed on by Nail on 2026-09-18: turn a Google Doc into a
Markdown file a person can read, and carry across what the document holds
besides words. Pictures, diagrams and schemas as image files the Markdown
links to, hyperlinks as Markdown links, and whatever other artifacts the
document carries, each in the nearest Markdown form.

This is not what `gdoc read` prints. `read` is a projection for an AI in a
review session, with suggestion and comment markers and no file on disk. It
prints `[image]` for an inline picture, nothing for a positioned one, and the
label of a link chip without its target. A plain hyperlink on a text run
loses its target too: `internal/docs/walk.go` decodes no link off a text
style. So the export is a new command, or a new output mode, not a flag on
`read`.

What is known already. Drive's markdown export returns an embedded image as
a base64 `data:` URI and leaves a Google Drawing out entirely, so it is not
the route on its own. The docx export carries both as PNG bytes, and
`internal/docx` already reads that export for the witness match. The bytes
side is docs/backlog/read-pictures-and-drawings.md, and this item is the
other half: a Markdown document on disk, in the reverse direction of
`gdoc publish`.

Why `later` rather than `yes`: the value decision is Nail's, and two things
settle it. Who the file is for, a person reading in the hub or an AI, because
that decides whether the export keeps suggestion and comment markers or
strips them. And where it lands, because a note in the hub with a `gdoc:`
block is a document `publish` can push back, and a loose Markdown file is not.
