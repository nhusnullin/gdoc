---
id: GD-17
title: '`[s:` is refused on the way in and not escaped on the way out'
status: In Progress
assignee: []
created_date: '2026-11-28T10:49:00Z'
updated_date: '2026-11-28T14:03:00Z'
labels:
  - worth-later
dependencies: []
ordinal: 17000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Found in the M13 review on 2026-09-19.

`internal/markers` lists four markers, and `looked` holds seven literals,
`[s:` among them. So `build`, `publish`, `reply`, `annotate` and
`propose --from` all refuse a file whose text carries a bare `[s:`.

`internal/view` escapes only the six two-character literals in `escapePairs`.
`[s:` is three characters and is not one of them. So a Google Doc whose own
words hold `[s:` exports to a hub file that gdoc then refuses, and the refusal
names a marker the document's author wrote and gdoc never did.

**Why it is not urgent.** The refusal is the safe direction: nothing is written
into a document, and nothing is misread as gdoc's. A person can fix the file by
hand, writing `\[s:`, which `markers.Find` reads as the author's own text by
the same parity rule as every other marker, and which markdown renders back as
`[s:`. `docs/guide/exporting.md` and `skills/gdoc-export/markers.md` both state
that parity rule.

**Why it was not fixed.** The escaping is a two-rune window throughout:
`escapeAt` looks at one rune and the one after it, and the streaming half,
`writeDoc` and `release`, holds one rune back and one ahead so that a pair
split across two Docs runs is still escaped. `view/doc.go` states that
principle. A three-rune literal needs a three-rune window on both halves, and
half a fix, the window but not the stream, would leave a gap on exactly the
case the streaming half was written for.

**What a fix looks like.** Either widen both halves to the longest literal in
`markers.Pairs`, which makes the escaping general rather than pair-shaped, or
decide that `[s:` standing alone is not a marker at all, take it out of
`looked`, and refuse only the shut-marker-then-`[s:` shape that gdoc actually
writes. The second is smaller and is a decision about what the refusal is for:
a lone `[s:ID]` carries no suggestion, because the words it belongs to are the
ones inside the `{+` or `{-` in front of it.

Pins to write with the fix: a document whose text holds `[s:` exports to a file
`publish` accepts, and one that holds `\[s:` still refuses nothing.

Added 2026-09-19 as docs/backlog/the-suggestion-marker-is-refused-but-not-escaped.md, worth: later.
<!-- SECTION:DESCRIPTION:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Took neither of the two shapes the task names: the colon takes the backslash where it completes the marker, so the window stays one rune and both halves are covered, a link's own bracket included.
<!-- SECTION:NOTES:END -->
