---
worth: later
where: go/internal/body/build.go:295
added: 2026-09-19
---
# A relative link to another note publishes as an address that opens nothing

`[bybit](2026-08-20-bybit-card-regional-setups.md)` in a note's Related
section reaches `word/_rels/document.xml.rels` as an external relationship
whose target is the bare path, the same shape the `#` anchor links had before
M12. Word resolves it against the document's own location; in Drive that file
does not exist, so clicking the link opens nothing and nothing warns. Every
hub note ends with a Related section written this way, so every published
note carries a few dead links at the bottom.

Seen 2026-09-19 in the Bridge sales-deck note, whose four Related links all
point at other hub notes by relative path.

`later` because the value decision is open, not the work. Three shapes, and
none is obviously right:

- Print the words as plain text and warn naming the line, which is how a code
  block and a dead `#` anchor are already refused. Cheap and honest; the
  reader loses the pointer to the other note, which was the point of the
  section.
- Rewrite the link to the other note's own Drive document, when the other
  note has been published. That needs the `gdoc:` block of the other file,
  read from the note's directory the way a relative picture path already is,
  and a rule for a note that has no block yet. It makes publish order matter.
- Keep the file name as plain text and append nothing. The Related section
  reads as a list of note names, which is what a colleague in Drive can search
  the hub for.

The unknown that settles it: whether a colleague reading in Drive should be
able to follow a Related link at all, or whether Related is hub navigation
that has no meaning outside the hub. That is a question about what Related is
for, not about the docx.

`TestALinkIsAHyperlinkWithARelationship` in `body/body_test.go:323` states the
current behaviour.
