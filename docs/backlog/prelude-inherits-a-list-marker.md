---
worth: yes
added: 2026-09-10
---
# A prelude proposed into a list item arrives bulleted

`internal/prelude` states every look one of gdoc's own paragraphs can state, because text inserted into a
document takes the look of the text it lands beside: the named style, the alignment, the spacing, the
indents, and on every run the face, the size, the weight and both colours. One look is left, and it cannot
be stated away.

A list marker is not a paragraph property. A paragraph is in a list because it carries a `bullet`, and the
only request that takes one off is `deleteParagraphBullets`. Nothing in M7c sends one. So a prelude
proposed at index 1 of a document whose first paragraph is a list item arrives as list items: a bulleted
cover, with the house indents fighting the list's own.

It is not the common case. A document worth restyling opens with a title or a heading, and both are
ordinary paragraphs. It is also visible rather than silent, because the prelude is a suggestion and Nail
reads it in the browser before accepting it.

Two ways out, and the choice is a design one:

- Send `deleteParagraphBullets` over the prelude's own range, in the same SUGGEST batch. It is one request
  and it names only the range gdoc inserted. What is unmeasured is whether Docs takes it as a suggestion:
  the 2026-09-10 probe measured `createParagraphBullets` and not its opposite, and the reference's own
  note about `createParagraphBullets` removing leading tabs is a reminder that this family changes text.
- Refuse instead: read the paragraph at the insertion point, and when it carries a bullet, say so and stop.
  Cheaper, and it turns a strange-looking cover into a sentence Nail can act on.

Found while writing the cover builder, M7c Task 4.
