---
worth: yes
added: 2026-09-06
---
# A suggestion inside a footnote is invisible, and nothing says so

`docs.Document` holds `Footnotes map[string]string`: `addFootnotes` flattens each footnote through
`plainText`, which concatenates the run texts and discards `InsertionIDs` and `DeletionIDs`. The shape is
the plan's own (`docs/plans/2026-09-06-gdoc-v2-m2-reading.md`, Task 4), so this is a scope limit rather
than a defect against M2.

What follows from it:

- `suggestions.walk` iterates `d.Tabs` only, so a suggestion in a footnote is absent from `pending` and
  from `IDs`. It never enters the `suggestions_seen` snapshot, so it never appears in
  `gone_since_last_look` either.
- `view.appendFootnotes` prints the flattened string, so `read` shows a suggested deletion in a footnote as
  ordinary text, with no `{-...-}` around it.
- Nothing warns. A review written in suggesting mode inside a footnote reads as a document with nothing
  suggested, which is the silent-failure shape `internal/suggestions` exists to close for the body.

Found in the M2 review, 2026-09-06. Two ways out, and the choice is a design one:

- Keep the footnote's blocks on `docs.Document` rather than its text, and walk them in `suggestions.walk`
  and `view.appendFootnotes` like any other body. Correct, and it changes a shape M3 is about to build on.
- Cheaper for now: have `docs.Parse` record which footnotes carry suggestion ids, and have `view.Text` and
  the `suggestions` command warn. The blind spot stays, but it stops being silent.
