---
worth: later
where: go/internal/propose/span.go:183
added: 2026-09-08
---
# A proposal into text inside a Google Docs table does not land

Nail's finding from the M4 acceptance run, 2026-09-08: a suggestion into words that sit inside a
table did not work, while the same flow on ordinary paragraphs did. Everything else in the run
held, so this is a table-specific defect, not a live-mode one.

Measured 2026-09-18, `docs/v2/MEASURED.md` "A proposal into a table cell
lands": the same command into a cell of a two-column table on a document
`publish` created works end to end, `verified: true` with all three checks
true, and `read` prints the suggestion inside the cell. The control into an
ordinary paragraph behaved the same. So the simple shape is not the defect.

`worth: later` because what is unknown is the shape that failed. The
2026-09-08 document, its table and the envelope were not kept. What would
settle it: the document id of that run, or a description of its table (merged
cells, a nested table, a quote crossing a cell boundary, a quote in a header
row). With one of those the reproduction is one `propose` on a copy in the test
folder and the suspects below can be checked against a real envelope.

Suspects, in order, still unmeasured on the failing shape: the `insertComment`
range crossing a cell boundary (a comment range in a table may need to stay
inside one cell, and the replacement's UTF-16 length is what sets the end);
`deleteContentRange` refusing a range that includes the cell's trailing
newline; and the docx witness on a nested table.
