---
worth: soon
where: presence/Sources/PresenceUI/Components/MiddleTitle.swift:70
added: 2026-10-18
---
# A verb prefix eats the document name in a TODAY row

Scene r9's first TODAY row is drawn as `Live stopped by you · Ch 5.… v0.1`,
where the same document is `Ch 5. Commerc… v0.1` in DOCUMENTS and
`Ch 5. Commercial M… v0.1` three rows lower. The reader is left with four
characters and a full stop to tell the document by.

The cause is the line, not the title. `MiddleTitle.fitted` measures the
whole line against one width, the document row's column, and a TODAY line
carries its verb in front of the name: `Live stopped by you · ` is 22
characters before the title starts. The head floor the second reviewer pass
wrote down, eight characters including the ellipsis, is measured on the
head, and the head here is the verb and the name together, so it passes at
27 characters while the name inside it is cut to five.

The row is not wrong by any rule the principles file states. It is wrong to
read, which is the thing the rule was written for.

Three shapes, all of them Nail's call:

- The floor counts the name rather than the head. That means the row hands
  `MiddleTitle` the two pieces it is made of instead of one string, so the
  component can measure the name on its own.
- The verb is drawn in its own ink and the name keeps the room it has in
  DOCUMENTS, so one document reads the same everywhere. This is the biggest
  change: TODAY's rows become two columns.
- Nothing changes, and the index page says a long verb takes room from the
  name.

Found by the reviewer pass of M25 Task 7, which did not fix it because the
row is M24's and the pass was over what M25 changed.
