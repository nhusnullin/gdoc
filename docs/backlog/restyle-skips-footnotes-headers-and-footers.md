---
worth: yes
added: 2026-09-10
---
# A restyle styles the body and says nothing about the footnotes, headers and footers

`restyle.TabRequests` walks `t.Body` and nothing else, so `restyle --from` gives the house style to
every paragraph in the body and leaves a paragraph inside a footnote, a header or a footer with the
look it had. `ManualSteps` names five things and none of them is that one, so the run comes back
`verified: true`, with a `manual` list a caller reads as the whole of what is left to do.

This is not a promise the milestone broke: the plan's scope is the page geometry, the paragraph and
run styling and the table cells, and it never said footnotes. What is wrong is the silence. A
document imported from Word that already carries a running head reads as house style in the body and
as whatever it was above it, and nothing in the answer says so.

Why it cannot simply be added as a sixth entry:

- **Unconditional is the cry-wolf shape.** Most documents hold no footnote and no header, and an
  entry that fires on all of them is the warning people learn to ignore. Every other conditional
  entry in `ManualSteps` is gated on a count.
- **The count is only half available.** `docs.Document.Footnotes` says whether the document holds
  footnotes, so that gate exists, but `ManualSteps` is given a `Plan` built from one tab and never
  sees it. Headers and footers are worse: `internal/docs` decodes neither, so nothing in the binary
  can tell whether the document has one.

Two ways out, and the choice is Nail's:

- Cheaper, and it closes the silence rather than the gap: plumb the footnote count into `Plan` and
  add a conditional `ManualStep` for it, and state the header and footer exclusion in the command's
  own words, since the run cannot detect one. This is one field and one entry, and it is the M2 line
  held: the binary reports the fact and the skill reads it out.
- Fuller: teach `internal/docs` to carry the footnote, header and footer segments with their indexes,
  write `Range.Segment` into the styling requests, and restyle them properly. `Range` already carries
  `Segment` and `docs.Tab.Places` already refuses a span that names one, so the shape is there and
  the decoder is not. That is a milestone rather than a fix, and it widens what the in-place grant
  writes to, which is its own decision.

Found in the M7b external review, 2026-09-10, by both review agents independently.
