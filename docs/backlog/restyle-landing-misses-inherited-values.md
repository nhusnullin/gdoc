---
worth: yes
where: go/internal/restyle/landing.go:558
added: 2026-09-25
---
# restyle's landing check reports a value as missing when Docs stored it by inheritance

## What happened

Same run as `restyle-flattens-a-published-cover`, 2026-09-25. The apply came back `verified: false`
with this warning:

> the updateTextStyle the run sent is not in the text runs of the paragraph at 1 as it was sent:
> [foregroundColor weightedFontFamily]

The session told Nail that the "Altery Group" line's font and colour "did not change to the house
values". That was wrong. The request set Calibri and `#000000`. The document's NORMAL_TEXT named style
is already Calibri and black, so Docs stored nothing on the run: the run's `textStyle` holds only
`fontSize: 12`, the one value that differs from the named style. The text renders Calibri and black.
The write landed. The check could not see it.

The HEADING_3 paragraphs show the same thing from the other side: their runs carry `{}`, because 12pt
and `#549F99` are exactly what the document's HEADING_3 named style says.

## Root cause

`compareRuns` passes each run's own `textStyle` to `missingFields`. Docs does not keep a run value
equal to the value the run would inherit from its paragraph's named style. So any request that sets
the value a document already inherits is reported as missing. On a document built from the house
template, that is most requests, and `verified: false` becomes the normal answer on a correct
document: the cry-wolf shape `landing.go` itself warns against.

## Fix

Compare against the effective style, not the stored one: the run's own `textStyle`, then the named
style of its paragraph's `namedStyleType` from the document's `namedStyles`, then NORMAL_TEXT. A field
is missing only when none of the three carries the value sent. `internal/docs` would need to decode
`namedStyles`, which it does not today. A unit test with a run carrying `{}` under a named style that
states the value pins it.

## Also seen, cause not known

The same run reported the `updateTableCellStyle` missing every border and padding field on the table
at 6523. Read again later the same day, both tables in the document have an empty cell style, so this one may be
a real miss rather than inheritance: the grey 0.5pt border is not the Docs default. Two things to
measure before calling it: whether the `tableStartLocation` form lands at all (the fidelity probe
measured cell styling, and which form it sent should be checked), and whether a later paste into the
document rebuilt those tables.
