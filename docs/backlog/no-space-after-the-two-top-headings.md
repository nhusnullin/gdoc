---
worth: yes
where: go/internal/house/house.yaml:89
added: 2026-10-26
---
# No space after the two top headings

Nail, 2026-10-26: after a heading he wants space before the text starts, what
Google Docs calls "Add space after paragraph" under Format, Line and paragraph
spacing. Without it the text reads as glued to the heading.

The house style today:

| Style | space before | space after |
|---|---|---|
| heading_1 | 12 pt | 0 pt |
| heading_2 | 12 pt | 0 pt |
| heading_3 | 12 pt | 6 pt |
| heading_4 | 12 pt | 6 pt |

So the two headings a reader meets most often carry no space after them, and
the smaller ones do.

The values came from the Altery master template, and the offline drift gate
compares the built document against that master. Changing them is a house
decision rather than a fix: a `DECISIONS.md` entry, the new values in
`house.yaml`, the literal tests that state them, and the rows in the drift
gate's `Known` with the reason, the way the logo height went in v2.20.0.

What to settle first: how much space (Docs' own "add space after" is 10 pt on
the default style; 6 pt would match heading_3 and heading_4), and whether
`restyle` should apply the same value to documents gdoc did not write. Small,
and it can ride in the fix milestone with the render fields.
