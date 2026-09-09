---
worth: maybe
added: 2026-09-09
---
# A comment anchored outside the body is measured against the body

`rawCommentAnchor.Ranges` in `go/internal/docs/walk.go` reads `startIndex` and `endIndex` and nothing
else, so every `Range` built from a comment anchor has an empty `Segment`. `Places` measures such a range
against the tab's body, because `covers` walks `t.Body` alone.

The measurement behind `go/internal/docs/testdata/anchors.json` was made on 2026-09-06 on a real document
whose comments are all anchored in the body. A body-anchored comment carries no segment id under any
shape, so the fixture cannot say what Docs sends for a comment anchored in a header, a footer or a
footnote. Two things are unmeasured, and they are separate questions:

- Does Google Docs let a person anchor a comment outside the body at all?
- If it does, does `commentAnchors[...].ranges[]` carry a `segmentId`, spelled that way?

What follows if the answer to both is yes: the indexes are segment-relative, `Segment` is empty because
nothing reads it, and a pair that happens to fall inside the tab's body run comes back placed. `read`
would then open `[[c:ID]]` around the wrong words and `comments` would report a range that is not where
the comment is. Both are false facts in the one field the skill places a proposal from.

The way out is one line each in two places once the shape is known: add `segmentId` to
`rawCommentAnchor.Ranges` and set `Segment` on the two comment-anchor `Range` constructions
(`docs.go:appendTab` and `walk.go:anchoredRange`). The refusal already in `Places` then covers the path,
and the comment comes back with `range: null` and a warning, which is the direction this tool is wrong in
everywhere else.

Doing it before the measurement would be a guess about a shape nobody has seen, which is what the three
fallback anchor shapes in `anchoredRange` already are. Measure first, on a document with a comment in a
header, then close it.

Found in the M7 external review, 2026-09-09.
