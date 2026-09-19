---
worth: later
added: 2026-09-06
---
# `gdoc read` still prints a placeholder where a picture is

Nail's requirement, recorded during the M2 re-cut on 2026-09-06: the AI should
be able to read a document's pictures, diagrams and Google Drawings, not only
the words around them. Half of it landed on 2026-09-19 in M13, and this item is
what is left.

**What landed.** `gdoc export` writes the picture bytes into the hub as PNG
files beside the note, taken from the docx export and paired with the Docs read
by body order, `contentUri` staying out because its host is one the guard does
not admit. A floating picture is no longer silent either: `internal/docs` reads
`positionedObjectIds` and the `positionedObjects` map, and `read` prints
`<!-- image: floating, kix.p1 -->` after the paragraph the object is anchored
to, naming the kind the walk read rather than guessing at a picture. That was
the worse half of this item, because a document with a diagram in it came back
with no sign the diagram existed.

**What is left.** `gdoc read` itself still prints `[image]`, `[drawing]`,
`[equation]` and `[object]`, so a review session reading a document with a
schema in it reads a placeholder. The bytes exist and a route to them exists.
The question nobody has answered is what `read` should do with them, and it is
a question about the projection rather than about the decode:

- A path per object on the envelope means `read` writes files, and `read` is a
  pure read that writes nothing on disk today. Changing that is a decision.
- Inline bytes on the envelope make a read of a picture-heavy document large
  enough to matter in a session's context, for a picture the session may not
  need.
- Saying nothing and pointing at `gdoc export` is the answer today, and it
  costs one command more in a review session.

Worth `later` rather than `yes` because the export route already gives a
session the pictures when it wants them, so what is left is convenience rather
than a gap.
