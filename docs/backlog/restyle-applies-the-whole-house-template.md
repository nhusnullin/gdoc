---
worth: yes
where: go/internal/guard/policy.go:923
added: 2026-09-10
---
# A restyle should apply the whole house template, not typography only

Nail's decision of 2026-09-10, reversing his own scope decision of 2026-09-09. He looked at the first
restyled document and said the typography-only call was wrong. What he wants from `restyle --from` is
a document that looks like the house template: the cover, the front-matter tables, the legend, the
contents list, the heading numbering, and the footer's own text and colour.

**This is the first thing after M7b merges**, and it is his own words: important, and next.

The decision it reverses is in `docs/v2/DECISIONS.md` under 2026-09-09 ("What an in-place restyle
changes, what bounds it, and what it deliberately does not write") and in the M7b plan's overview. It
is not one task's: it shaped Task 4's page request, Task 5's paragraph and table work, and Task 1's
allowlist, which is where the wall actually is.

## Read this before planning it: it breaks M7b's defining property

M7b's whole security argument is one sentence. The guard's allowlist at `LevelInPlace` is four kinds,
`updateDocumentStyle`, `updateParagraphStyle`, `updateTextStyle` and `updateTableCellStyle`, and
**none of them can change a single character**. `TestNothingAtLevelInPlaceCanChangeACharacter` states
it as a property rather than a list, and `createParagraphBullets` is off the list for exactly that
reason: the reference says it removes leading tabs the author typed.

Every item Nail is asking for writes text:

| Wanted | Needs | Today |
|---|---|---|
| cover / title page | `insertText`, `insertPageBreak` | refused, both change text |
| front-matter tables | `insertTable`, `insertText` | refused, both change text |
| legend | `insertTable`, `insertText` | refused |
| heading numbering | `insertText` into the author's prose | refused, and it was refused on its own grounds too |
| contents list | no Docs request creates one at all | not an allowlist question |
| footer text and colour | `updateTextStyle` with a `segmentId` | allowlisted already, just unbuilt |

So this is not a matter of adding kinds to a slice. It reopens the question the allowlist answers: what
may a run that was granted one document do to it, once "it cannot touch the words" stops being true.
Whoever plans this owes an answer to that before writing any of it, and the answer is Nail's.

## The four distinct problems under one request

1. **The allowlist.** Opening `insertText` at `LevelInPlace` means the read-back becomes the only bar
   left, on a level that already has no `writeMode` for the server to disagree with. The `fields` mask
   rule stays either way.
2. **There is nothing to write the cover from.** `build` reads the title, doc type, version, owner,
   dates, distribution and classification out of a note's YAML front matter. A restyle has no note. It
   either takes a `--md` pairing, or the fields on the command line, or it infers them from the
   document, and inferring is the one thing gdoc does not do. This is a design question, not a coding
   one.
3. **The contents list may be impossible.** `build` gets one because a docx carries a Word field that
   Google refreshes on import. Nothing in the Docs API inserts a table of contents into a document
   that already exists. This needs measuring before it is planned, the way the fidelity probe measured
   the styling kinds. If it cannot be done, it stays a `ManualStep` and the plan says so.
4. **Idempotence, which none of the above has.** Styling twice is harmless: the second run sets the
   same values. Inserting twice gives a document two covers and two front-matter tables. A restyle
   that writes content has to be able to tell "already has a house cover" from "has none", and there
   is no marker for that today. A named range would be one, and `createNamedRange` is off the
   allowlist and was measured landing.

## What is cheap and separable

The footer's own text and colour is the odd one out: it needs no new request kind. A Docs `Range`
carries a `segmentId`, `updateTextStyle` is already allowlisted, and `Range.Segment` and
`docs.Tab.Places` already exist. What is missing is that `internal/docs` decodes no `headers` and no
`footers`, so nothing can see the footer, and `pageMask` carries no `marginHeader` or `marginFooter`,
so its position is untouched too. That half overlaps
[restyle-skips-footnotes-headers-and-footers.md](restyle-skips-footnotes-headers-and-footers.md),
which was found independently in the M7b review and offers the same two ways out. Doing the footer
first would be a real improvement that costs no scope decision at all.

## How the gap was found

Nail opened the first live restyle, a copy of a real policy document, and asked why it had no title
page, no contents table, no version-control table, and an untouched footer. The guard had refused
nothing: all 181 requests were carried, and the missing pieces were the scope decision plus one
unbuilt path. The report was silent about most of them, which is its own defect and is the other
half of the item linked above.
