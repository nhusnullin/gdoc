---
worth: yes
added: 2026-09-06
---
# A marker split across two runs reaches the text unescaped

`internal/view.writeText` escapes one run at a time. It looks at `rs[i]` and `rs[i+1]`, so a two-character
marker whose halves fall in two different `docs.Run`s is never seen. Docs splits a run at every formatting
change, so a paragraph where the first `[` is bold and the second is not gives two runs, `"...["` and
`"[..."`, and the projection emits `[[`. An AI reading it sees gdoc's comment-open marker in somebody's
sentence, which is exactly what the escaping exists to prevent. The same holds for `{` followed by `+`.

Found in the M2 review, 2026-09-06. The overlapping case inside one run (`{-}` leaving a bare `-}`) was
fixed then; this one was not, because it is a design question rather than a one-line change.

Two things make it more than a matter of carrying the previous rune across the call:

- The escape convention puts the backslash before the pair, and the first half has already been written by
  the time the second run arrives. Escaping the second half instead (`[` then `\[`) breaks the pair and
  reads consistently, but it is a second convention in the same output and it should be Nail's call.
- The document's own text sitting next to one of gdoc's own markers has the same ambiguity, and it is not
  addressed at all today. A run ending in `[` followed by a comment-open marker emits `[[[c:ID]]`, and a run
  ending in `{` before an insertion marker emits `{{+`. Whatever rule is chosen has to cover both, or the
  invariant CLAUDE.md states ("a marker in the output is always gdoc's") is still only mostly true.

`TestEscapingLeavesNoMarkerBehindWhenTwoOverlap` in `internal/view/text_test.go` holds the within-run half
of the invariant and is the place to extend.
