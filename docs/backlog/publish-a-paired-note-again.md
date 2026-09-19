---
worth: yes
where: go/cmd/gdoc/publish.go:166
added: 2026-09-19
---
# Publish the same note more than once

Nail, 2026-09-19: a note should be able to go through `publish` more than
once. Today the second run is refused. `unpaired` in `cmd/gdoc/publish.go`
reads the `gdoc:` block and refuses a note that already names a document,
and its sentence says to open that document or take the block out by hand.
`TestPublishRefusesANoteThatIsAlreadyPaired` pins it, and the `gdoc-publish`
skill says the same in its first paragraph: a note is published once.

The restyle side is not the block. A document gdoc has proposed a prelude
into carries a marker, and a second `restyle` run replaces that prelude
rather than adding one. What is refused there is a pending old prelude or two
markers, both read out. So the item is the publish route alone.

This reverses a standing decision rather than fixing a defect, so a `yes`
here is a DECISIONS.md entry first, written the same day as the SPEC.md
change. The entry it answers is 2026-08-29, "Publish runs once. Everything
after travels as suggestions", restated in SPEC.md under `publish` on
2026-09-11 and 2026-09-16: colleagues keep one URL for the life of a
document, gdoc never replaces an existing body, and later hub changes travel
as suggestions. The reasons behind that entry still hold, which is why the
shape of a second publish is the open design question and not a flag:

- A new document from the same note, a new URL, and the block rewritten to
  name it. Cheapest, and it is what `restyle --new` would have been before
  2026-09-11 said no. The old document keeps its threads and goes stale.
- The same document with its body replaced. Keeps the URL, but it is the one
  thing the guard exists to make impossible on a handed-in id, and every
  thread anchored to the old text detaches.
- The same document with the note's changes proposed as suggestions. Keeps
  the URL and the threads, and it is alignment, which is
  docs/backlog/m8-alignment-and-the-align-skill.md and deferred with M8.

That M8 item already lists a second version of a note as one of the three
things folded into it, with the hand-edit as the answer until alignment says
more. This item is Nail saying the hand-edit is not enough.
