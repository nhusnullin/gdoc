---
worth: yes
added: 2026-09-09
---
# A suggestion the pending listing skips is reported by no command that lists

`suggestions.walker.paragraph` skips every run whose `Kind` is not `docs.KindText`. That filter is
deliberate for the two cases it was written for, and both reasons are still good: a footnote reference
carries its number as text, so reporting "1" as a suggested insertion would be a lie, and a suggested
picture has no text to report at all (`docs/backlog/read-pictures-and-drawings.md`).

M7 widened what falls into it. The decoder's default arm now names eleven members rather than four, so a
person chip, a date chip, a rich link, an auto text, a page break, a column break and a horizontal rule
each carry their own `InsertionIDs` and `DeletionIDs`. `read` prints all of them, inside `{+...+}` and
`{-...-}` markers with the suggestion id, and `suggestions` reports none of them.

What follows from it:

- `read` and `suggestions` disagree about the same document. `go/internal/docs/testdata/elements.json`
  carries nine suggestion ids, every one on a non-text element; `go/internal/view/testdata/elements.golden`
  prints all nine as markers, and `suggestions.All` on that document returns nothing.
- The snapshot is written from `All`, so an element-only suggestion never enters `suggestions_seen` and can
  never be reported in `gone_since_last_look`.

What M7 did about it, which is the survey half only: `restyle --dry-run` counts those ids itself as
`suggestions.on_elements`, subtracting the ids the pending walk already saw, and `nothing_to_protect` reads
that count. So the survey cannot answer "nothing to protect" over a pending suggestion, which is the field
M7b reads before it overwrites. The listing half is untouched.

Two ways out, and the choice is Nail's:

- Record the ids from every run, and give a non-text run its placeholder (`[person: A Name]`, `[page
  break]`) as the reported text rather than an empty string, so the words in `pending` are not a lie. The
  whitespace filter in `List` stays where it is. This closes the read/list disagreement and the snapshot
  hole in one move, and it changes the shape of a command two skills already read.
- Cheaper: leave the listing as it is and warn from the `suggestions` command when a document holds ids the
  listing cannot report, the way the survey now does. The blind spot stays, but it stops being silent.

Found in the M7 external review, 2026-09-09.
