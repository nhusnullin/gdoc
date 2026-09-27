---
worth: later
where: go/internal/restyle/landing.go
added: 2026-09-27
---
# restyle's table cell styling may not land, and nobody has measured why

Split out of `restyle-landing-misses-inherited-values` on 2026-09-27, when the
inheritance half was fixed.

The restyle run of 2026-09-25 on the public API v2 document reported the
`updateTableCellStyle` missing every border and padding field on the table at
6523. Read again later the same day, both tables in the document had an empty
cell style. Inheritance does not explain it: the house border is grey 0.5pt,
which is not the Docs default, so an empty cell style means the border is not
there.

What would settle whether this is worth fixing, and how:

- Whether the `tableStartLocation` form of the request lands at all. The
  fidelity probe measured cell styling, but which form it sent has not been
  checked against what `restyle` sends.
- Whether a later paste into the document rebuilt those tables, which would
  make the empty style the paste's work and not the restyle's.

Both are one live run in the test folder. Until then it is not known whether
this is a defect or a document that changed after the run.
