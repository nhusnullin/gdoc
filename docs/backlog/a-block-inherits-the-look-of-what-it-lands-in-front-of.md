---
worth: later
where: go/internal/propose/blockbatch.go:45
added: 2026-09-28
---
# A block inherits the face, the size and the colour of the paragraph it lands in front of

Text inserted at a paragraph's start takes that paragraph's first run style.
`BlockBatch` clears five of those marks, `clearedMarks`: bold, italic,
underline, strikethrough and link. Those are the marks the content itself can
carry, so inheriting one is always wrong. It clears nothing else, and
`updateParagraphStyle` states only `namedStyleType`, which does not touch
direct run formatting.

So a block inherits whatever direct `weightedFontFamily`, `fontSize` and
`foregroundColor` the paragraph behind the anchor carries. `restyle` writes all
three, on every named style: `go/internal/restyle/style.go`, `textLook`. In a
restyled document an `after` on a section's last paragraph inserts in front of
the next heading, so the block's body paragraphs can arrive in the heading's
face, size and colour.

The obvious fix is wrong. Adding those three to `clearedMarks` puts the block
back to the document's untouched defaults, and in a restyled document the
neighbours are not at the defaults either: a block landing in front of ordinary
body text, which is the common case, would then be the one paragraph that does
not match. Neither answer is right everywhere:

- inherit, and the block is right beside body text and wrong beside a heading;
- clear, and the block is right in a document nobody restyled and wrong in one
  that gdoc restyled itself.

`worth: later` because the decision underneath it is not made. A block that
matched its neighbours would have to read the look of the paragraph it is
placed after, or of the document's own body style, and restate it per new
paragraph. That is a third kind of request in the batch, on paragraphs the
suggestion created, and whether it stays one suggestion id is unmeasured.

What would settle it: a measurement of one block proposed into a document
`restyle` has been over, read back as a docx, saying what the new paragraphs
actually look like. Then Nail decides whether a block matches the document it
lands in or the house style, and this becomes a request with its own rule in
`propose/doc.go`.

Related: `docs/v2/MEASURED.md`, "A block of new paragraphs, proposed in one
SUGGEST batch", which measured the ids rather than the look.
