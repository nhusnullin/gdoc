---
worth: yes
where: go/internal/propose/span.go:183
added: 2026-09-08
---
# A proposal into text inside a Google Docs table does not land

Nail's finding from the M4 acceptance run, 2026-09-08: a suggestion into words that sit inside a
table did not work, while the same flow on ordinary paragraphs did. Everything else in the run
held, so this is a table-specific defect, not a live-mode one.

What the code believes today: `propose.FindSpan` walks tables into paragraphs on purpose (the
comment at `span.go:183` says a cell's paragraphs carry their own indexes in the same tab, so
text in a table is text a proposal can be placed in), and `verify.go:215` walks them the same
way for the inline read-back. So the span is found and a `batchUpdate` goes out. What is not
known is which step lied: whether Docs refused the batch, accepted it and placed the suggestion
somewhere else, or placed it and the three read-backs disagreed. The envelope Nail saw would say
which, and the first thing to do is reproduce on a throwaway document in the test folder with a
two-cell table and read the `warnings` and `checks`.

Suspects, in order: the `insertComment` range crossing a cell boundary (a comment range in a
table may need to stay inside one cell, and the replacement's UTF-16 length is what sets the
end); `deleteContentRange` refusing a range that includes the cell's trailing newline; and the
docx witness, which walks `w:p` under `w:body` and may not descend into `w:tbl`, so
`docx_anchored` comes back false on a proposal that actually landed.

Out of M4's scope. Belongs with the next milestone that touches `propose`, or its own small
plan if it blocks a real review first.

Measured for the comment-only shape by `TestLiveAnnotateParagraphAndTable` in `go/internal/live/`,
added 2026-09-18 with `gdoc annotate`: it places one comment in an ordinary paragraph and one in a
table cell in the same document and prints what each read-back said. An `insertComment` alone tells
the first suspect above from the other two, because no `deleteContentRange` goes with it.

**Measured 2026-09-18**: the table cell answered exactly as the paragraph did. `sent true,
drive_listing true, docx_anchored true, verified true` for both. So two of the three suspects are
cleared for the comment-only shape:

- The `insertComment` range does not have to stay inside one cell to be accepted, and a range
  `propose.FindSpan` found inside a cell is a range Docs anchors a comment on.
- The docx witness does descend into `w:tbl`, so `docx_anchored` coming back false is not something
  the export walk does to a table.

What is left is `deleteContentRange` over a span inside a cell, which is the half of a proposal
annotate does not send. The item stays open for `propose` on that one suspect.
