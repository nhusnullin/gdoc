---
worth: yes
added: 2026-09-08
---
# The offline gate exempts a drift row by name, not by value

`drift.Known` is a set of item names, and `Unexplained` skips a row whose name
is in it whatever the two sides now say. So an exempt row passes on any
difference, in either direction and of any size.

Fourteen of the twenty-five entries are measurements or counts rather than
words: `HEADING_1/2 colour`, `HEADING_1..6 indentStart`, the two body heading run
colours, `table count`, `bulleted paragraphs`, and the four
`table Revision History` rows. Measured by the M5 review on 2026-09-08: setting
`heading_1.color` to `#FF00FF` and `heading_3.indent_start_pt` to `200` in
`house.yaml` leaves `TestTheOfflineGate` green with an identical summary, and
`TestEveryKnownDifferenceStillDiffers` and `TestTheGateReadsTheWholeList` pass
too. A non-exempt change, `heading_1.size_pt` 16 to 18, fails correctly.

The incentive is inverted on those rows: disagreement passes and agreement
fails. If the body walker dropped every list marker, `bulleted paragraphs` would
go from 17 to 0, stay DIFFERENT, stay exempt, and nothing would report it.

The exemptions themselves are right and written down with their reasons in
CLAUDE.md: the master states a heading colour and indent on the style and
overrides both on the paragraph, and the two documents hold different words. What
is missing is the pair. Two shapes:

- `Known` records the measured pair as well as the reason, and a row whose
  values move fails until somebody re-records it. Exact, and it makes the list a
  second golden file to refresh.
- `Known` records a difference class, "differs in words" or "differs by an
  override", and the gate fails when the class changes. Coarser, and it stays
  readable.

Either way `Known` stops being a list of names. That is Nail's call, because
CLAUDE.md says adding a name to it is a decision somebody writes down.
