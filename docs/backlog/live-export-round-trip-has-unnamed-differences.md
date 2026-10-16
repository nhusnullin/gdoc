---
worth: soon
where: go/internal/live/export_test.go:125
added: 2026-10-17
---
# The live export round trip fails on main with 27 differences no name covers

Found 2026-10-17 at night while merging M20. `TestLiveExportRoundTrip` fails
on `main` before M20 and after it alike, with the same lines, so it is not a
regression of any one milestone: nobody had run it with the live variables
set since the fixture notes under `go/internal/body/testdata/docs/` last
grew. The comparison in `go/internal/live/roundtrip_test.go` names every
difference the round trip is allowed to have, and these have no name:

- `01-kitchen-sink.md`: a numbered list that starts at 7 in the note comes
  home starting at 1 (the build already warns about it), and a block quote
  written over three lines comes home as one paragraph with no `>`.
- `02-pictures.md`: a heading that is only a bold picture, a bare URL written
  `<https://…>` coming home as `[url](url)`, a data-URI picture definition,
  and the sentence about where a pulled Doc puts its diagrams.
- `05-edge-cases.md`: `<ampersands>` in a heading, task-list boxes `[ ]` and
  `[x]` coming home as `☐` and `☒`, and a quote containing a list.

Each is either a fault in the build or the export, or a row for the named
list with its reason. The decision per line is Nail's: a name says "both
sides are right and spell it differently", and a fix says one side is wrong.
Until then the test is red and the live suite cannot be used as a gate. The
other live tests pass, and M20's own live checks were re-run by hand on
2026-10-17 (`TestLivePublishCarriesTheMarksIntoDocs`, `TestLiveMarkImportProbe`).
