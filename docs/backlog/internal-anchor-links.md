---
worth: yes
added: 2026-09-08
---
# An internal anchor link publishes as a link that opens nothing

`[see below](#scope)` reaches `word/_rels/document.xml.rels` as
`<Relationship Target="#scope" TargetMode="External"/>`, which is what
`internal/body/build.go` builds for every link and what `internal/render`
writes. Word reads it as an external address relative to the document's own
location, so clicking it opens nothing. Nothing warns.

The OOXML form for an in-document jump is `<w:hyperlink w:anchor="scope">` with
no relationship at all, and it needs a `w:bookmarkStart`/`w:bookmarkEnd` pair on
the heading it names. No heading carries a bookmark today, so switching to
`w:anchor` on its own would swap one broken link for another.

Found by the M5 review, 2026-09-08, on the same read as the `mailto:` fix.

Two shapes, and picking one is a decision rather than a patch:

- Write a bookmark on every heading, from goldmark's own auto heading id, which
  `parser.WithAutoHeadingID()` already computes, and emit `w:anchor` for a link
  whose destination opens with `#`. A real cross-reference, and it puts a
  bookmark on every heading in the house style.
- Refuse it the way a code block is refused: render the link's words as plain
  text and warn naming the line. Cheap, honest, and it loses the jump.

`TestALinkIsAHyperlinkWithARelationship` in `internal/body/body_test.go` states
the current behaviour, and `TestAnEmailAutolinkCarriesTheMailtoScheme` beside it
is the shape a test for this would take.
