---
worth: yes
added: 2026-10-15
---
# The kitchen-sink round trip is red on a wrapped blockquote and a list that starts at seven

`TestLiveExportRoundTrip/01-kitchen-sink.md` stays red on eight lines that have
nothing to do with M18: a blockquote the note wraps over three lines, which comes
home as one, and a numbered list that starts at seven, which comes home numbered
from one. Every footnote line and every mark line of the same run is named by
`knownDifferences` and clean, so the milestone's own round trip holds, which is
why M18's acceptance item 7 is recorded as partial rather than failed.

The fix is one of two: `export` learns to keep a list's start number and leave a
wrapped quote wrapped, or `knownDifferences` gains two entries that name the two
shapes with the reason. The first is the better one if Docs keeps a list's start
number in its answer, which nobody has measured.
