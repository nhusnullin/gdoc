---
worth: yes
where: go/internal/propose/blockbatch.go:182
added: 2026-11-07
---
# A block after a document's last paragraph is not one suggestion, and is not 12pt

A block placed after the document's last paragraph goes in as `"\n" + text`
in front of the body's final mark, so its own last paragraph owns that mark
(`layOutBlock`, MEASURED.md row 7). Row 7 measured a hand-written batch and got
one suggestion id. The production batch was never measured there: the live
test's setup ended its five paragraphs with a newline, so the document kept its
empty last paragraph after them and the "end" block was an ordinary placement.

Measured on 2026-11-07, with the setup fixed so the block really lands at the
end:

- the batch gdoc has sent since M14 leaves two ids, the block's own and a
  second under `TextStyleChanges`;
- with the house look on every new paragraph it leaves five, every style
  request near that mark coming back as an id of its own.

The note records one id, and `withdraw` rejects that one. In the live run the
document read as before after the withdrawal, so the words come back, but the
other ids were not read afterwards.

So since 2026-11-07 a block at the end takes no look (Nail's choice: ship the
12pt fix everywhere else, keep the end as it was). Two things are wrong there
and stay wrong until this is done:

- the block is not one suggestion, which breaks the promise withdraw rests on;
- its paragraphs show the Normal style's 11pt, against the rule that body text
  is 12pt whatever gdoc is doing (DECISIONS.md, 2026-11-07).

`worth: yes` because both are agreed defects. What a fix needs is a placement
at the end that leaves the body's final mark alone, or a batch whose style
requests fold into the insertion's id there, and a live measurement of it.
`TestLiveProposeBlock` holds the end block at two ids (`blockAddedIDs`) and
`TestABlockAtTheEndTakesNoLook` pins the batch, so a fix changes both.
