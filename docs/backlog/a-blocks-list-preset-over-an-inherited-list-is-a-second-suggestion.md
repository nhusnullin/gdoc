---
worth: next
where: go/internal/propose/blockbatch.go, listStretches
added: 2026-11-07
---
# A block's list preset over the list it inherited leaves a second suggestion behind

Measured on 2026-11-07 by M33's `TestLiveProposeSuggestsATaskList`, before the
task stretch learned to take the bullets off first. New text takes the list
membership of the paragraph it lands in front of, so a block placed in front of
a list item arrives in that list. `createParagraphBullets` over those
paragraphs then changes that list's own nesting-level properties, and Docs
records the change as a suggestion of its own:

    after the task block: BulletChanges=1[suggest.i8gbxc1a0o3i]
    InsertionIds=1[suggest.q248yglntsv1]
    ListPropertiesChanges=1[suggest.i8gbxc1a0o3i]
    TextStyleChanges=1[suggest.q248yglntsv1]

The task stretch is fixed: it sends `deleteParagraphBullets` before the
checkbox preset, so the boxes make a list of their own inside the one
insertion, and the same run then came back with one id on everything.

The bulleted and the numbered stretches still do not. The same run of
`TestLiveProposeBlock` on 2026-11-07 shows it, on a block whose content holds a
bulleted list and a numbered one, placed in front of a disc bullet:

    ListPropertiesChanges=1[suggest.u9xbhqb05gr0]

Nothing in the package sees it. `propose`'s own walk reads the insertion and
deletion ids on the runs, which is where every other request of the batch
lands, and a list-properties change is recorded under a key of its own, so
`VerifyBlock` reports one id and `withdraw` takes back one id. The stray
suggestion stays in the document until somebody rejects it there. In the
read it is a list whose numbering is pending, which is why nobody has noticed:
the words are all in the one suggestion.

`worth: next` rather than now because the fix looks like the task one and is
not measured: a `deleteParagraphBullets` in front of every
`createParagraphBullets` would take the paragraphs out of the inherited list
and make a new one, which is what the checkbox stretch now does, but a numbered
list made that way restarts at 1 whatever the inherited list was counting, and
that is the failure `listKind` already refuses a list starting at 3 for. One
live run of a bulleted and a numbered block in front of a disc item, with and
without the removal, would settle it.

Related: `docs/v2/MEASURED.md`, "A block of new paragraphs, proposed in one
SUGGEST batch", which measured the run-level ids and not these two keys.
