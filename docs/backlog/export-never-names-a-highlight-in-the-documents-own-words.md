---
worth: yes
added: 2026-10-15
---
# Export never names a pair of == in the document's own words

A highlight is written `==words==`, so a document whose own prose holds a pair of `==` comes home as
a file whose next build reads a mark the document never carried. `a ==b== c` in a Google Doc exports
as `a ==b== c` and publishes `b` as a yellow run, with the four `=` dropped out of the text and a
debt counted on the envelope. Nothing says so on either side.

`view.gdocShapes` names the two shapes it can: a footnote mark, `[^n]`, and the opener a mark's
comment is written with, `<!--`. Both are shapes. A pair of `==` is not: whether it opens a highlight
is goldmark's flanking rule, and `internal/body` refuses to read that off a source line for exactly
this reason, which `checkMarks` states as "a source scan would have to decide for itself whether
`a == b == c` is a highlight". A scan in `view` that guessed would warn about every sentence holding
two `==`, and one that did not guess would miss the case that matters.

The pre-M18 shape of the same hole: `==` was the one mark before the palette, so this is older than
the marks milestone.

What it would take, in order of cost:

- One reading of a markdown line, shared by `internal/body` and `internal/view`, with goldmark's
  flanking rule in it once. Correct, and it is a new package that both depend on, which is Nail's
  decision rather than a refactor.
- Ask goldmark. `view` would parse the projected line and look for a `Highlight` node, which is the
  honest answer and makes the projection depend on `internal/body`'s parser.
- Warn on the shape and accept the false ones. Cheapest, and it tells a session its own prose is
  suspect in documents where nothing is wrong.

Until then the loss is `export`'s unnamed one, and `cannotCarry` catches the half that sits inside
another mark: a highlight whose words hold the `<!--` a pencil opens with comes home as its words
with the loss named. A pencil's own words are not that half, because nothing inside a comment is
markdown, so a pair of `==` in a grey aside comes home inside the comment as the characters it is.
