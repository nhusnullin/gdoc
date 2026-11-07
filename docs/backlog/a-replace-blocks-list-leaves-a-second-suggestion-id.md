---
worth: later
where: go/internal/propose/blockbatch.go:285
added: 2026-11-07
---
# A replace block's list leaves a second suggestion id under ListPropertiesChanges

`TestLiveProposeBlock` used to count a block's suggestion ids across every
suggested key only for the block at the end of the document. On 2026-11-07 it
counted the other two as well, and the replace block came back with two ids:
its own, under insertions, deletions and text style changes, and a second one
under `ListPropertiesChanges`. The block's content holds a bulleted list, so the
second id most likely comes from `createParagraphBullets` in SUGGEST mode making
a new list.

It is older than that day. The same run with no look, which is the batch gdoc
sent before 2026-11-07, left the same second id under the same key. The house
look adds no id.

propose's own walk reads insertion and deletion ids only, so the note records
one id and `verified` is true. withdraw rejects that one id. In the live run the
document read as before once all three blocks were withdrawn, so the words come
back. Whether the list properties suggestion is gone too, or stays pending with
nothing a person can see, was not read.

`worth: later` because the cost is unknown. Settling it means one read after the
withdrawal: does `ListPropertiesChanges` still carry the id? If it does, a
person sees a suggestion nobody made in the margin, or the Docs UI shows nothing
and it is only noise. That answer decides whether it matters. If it matters,
propose would read the id and record it beside the first, so withdraw takes
both.

The live test counts the replace block at two ids (`blockAddedIDs`), so a third
fails it.
