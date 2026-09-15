---
worth: yes
where: go/internal/prelude/prelude.go
added: 2026-09-10
updated: 2026-09-10
---
# A restyle should apply the whole house template, not typography only

Nail's decision of 2026-09-10, reversing his own scope decision of 2026-09-09. He looked at the first
restyled document and said the typography-only call was wrong. What he wants from `restyle --from` is
a document that looks like the house template: the cover, the front-matter tables, the legend, the
contents list, the heading numbering, and the footer's own text and colour.

## Most of it landed at M7c, and this file is what is left

`gdoc restyle <url> --from survey.json --fields fields.json` proposes the cover, the three
front-matter tables and the legend. Read `go/internal/prelude/doc.go`, and
`docs/plans/completed/2026-09-10-gdoc-v2-m7c-house-template-as-suggestion.md` for how it was built.

The scope question this file said had to be answered first was answered, and the answer was better
than the question. Nothing was added to `LevelInPlace` except `createNamedRange` under its own
per-run grant, and that adds no character. The template is a **suggestion**, sent on a policy that
granted nothing at all, which is what `propose` has done since M3. So M7b's defining property, that
nothing the grant carries can change a character, is untouched.

Idempotence was the other blocker and it has an answer too. A named range called
`gdoc:house-prelude` marks what gdoc proposed. It was measured surviving an accept and vanishing with
a reject, so a second run replaces the prelude it finds, proposes cleanly when there is none, and
refuses when the one already there is still pending.

## What is still open

| Wanted | State |
|---|---|
| cover, front-matter tables, legend | done at M7c, as suggestions |
| contents list | **blocked**, no Docs request makes one. `BLOCKED-BY-API.md` |
| front-matter column widths and row heights | reported in `manual`: their two request kinds were not among the nine measured suggestible |
| heading numbering | not blocked. `insertText` is suggestible, so it can be proposed |
| footer text and colour | not blocked and unbuilt |

**Heading numbering** needs two answers before it is planned, and neither is about permission.
Nothing marks a number gdoc wrote, and `numberedHeadingRE` in `internal/body/numbering.go` does not
recognise the house's own `-` separator, so a second run makes "1-1-Introduction". And a number names
a position inside the author's prose, which is what `propose`'s "names text, never an index" rule
exists to refuse. Nail's recommendation of 2026-09-10 is that it gets its own milestone and starts
from suggested mode.

**The footer's own text and colour** is the cheap one and still the odd one out: it needs no new
request kind. A Docs `Range` carries a `segmentId`, `updateTextStyle` is already allowlisted, and
`Range.Segment` and `docs.Tab.Places` already exist. What is missing is that `internal/docs` decodes
no `headers` and no `footers`, so nothing can see the footer, and `pageMask` carries no
`marginHeader` or `marginFooter`, so its position is untouched too. That half overlaps
[restyle-skips-footnotes-headers-and-footers.md](restyle-skips-footnotes-headers-and-footers.md),
which was found independently in the M7b review and offers the same two ways out.

## How the gap was found

Nail opened the first live restyle, a copy of a real policy document, and asked why it had no title
page, no contents table, no version-control table, and an untouched footer. The guard had refused
nothing: all 181 requests were carried, and the missing pieces were the scope decision plus one
unbuilt path.
