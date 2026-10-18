---
worth: later
where: presence/Tests/PresenceUITests/Scenes/r1.jsonl, r2.jsonl
added: 2026-10-18
---
# The r Saturday's publish is two runs that cannot both be true

The index page says the nine r screens are one Saturday told nine times and
agree with each other. Two of them do not.

- r1, at 09:04, draws `Publishing · Chapter 4`, step 2 of 4, with
  `Built the docx` done and `Uploading` running.
- r2, at 09:05:30, draws the same publish stopped with
  `SVG picture at line 41. Nothing was created`.

A picture the render cannot read stops the build. It cannot stop a publish
whose docx is already built and already going up, so the two records are
two different runs wearing one clock.

Either fixture can move. r2's stop could become one the upload can raise,
or r1's moment could fall before its build step. The second is smaller and
keeps the stop code the card's reason line was written for.

Found by the reviewer pass of M25 Task 7, which did not fix it because the
fixtures are M24's and the pass was over what M25 changed. Nothing in the
app is wrong: both screens draw their own record correctly.
