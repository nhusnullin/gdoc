---
worth: yes
where: go/internal/live/roundtrip_test.go
added: 2026-10-15
---
# The kitchen sink round trip fails on two things that are not footnotes or marks

Run on 2026-10-15, `TestLiveExportRoundTrip/01-kitchen-sink.md` against real
Drive. Every footnote line and every mark line is named: the `footnotes` entry
covered four lines, `inline markup` four, and nothing of either is in the
unnamed list. So M18's part A and part B come home.

The run still fails. Eight lines no name covers, and both causes are older than
this milestone: the note has carried the quote and the list since the commit
that first rendered it, `744534f`.

**A blockquote wrapped over three lines.** The note writes one quote paragraph
across three `> ` lines, and the document holds it as one paragraph of prose.
`noteLines` treats every line matching `blockStarter` as the start of a block,
and `> ` is one of them, so the three lines never join and the `blockquotes`
rule has three partial lines to strip rather than the one the file holds. That
is a defect in the comparison and not a difference in the document: once the
markers and the wrap are off, both sides are the same words. The fix is in
`noteLines`, a `> ` line that follows another one continuing the quote it is
part of rather than starting a block. Nothing is loosened by it.

**A numbered list starting at seven.** The note writes `7.` and `8.`, the
document numbers them `1.` and `2.`, and `read` prints the document's numbers.
`internal/body` already warns at render time, "this numbered list starts at 7
in the note and at 1 in the document", so the difference is known and
unrecorded here. Two ways out, and they are not the same decision:

- **Name it.** One `knownDifferences` entry saying a numbered list's own start
  is not carried, with the warning as its reason. By this file's own rule a
  name joining that list is Nail's decision with its reason written down, never
  a test somebody loosens, so it waits for him.
- **Carry the start.** `render` writes a `w:start` override on the numbering
  definition so the document numbers from seven, the warning goes, and the two
  sides agree with no name at all. That is the better document and the larger
  change, and it belongs with whatever else touches numbering.

Until one of them lands, the kitchen sink round trip is red on the eight lines
above and green on everything M18 added, and that is what the live log says.
