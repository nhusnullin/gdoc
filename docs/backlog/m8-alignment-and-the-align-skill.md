---
worth: later
added: 2026-09-16
---
# M8, alignment and the align skill, deferred

Nail deferred the whole of M8 on 2026-09-16 to put the release milestone
first: the tool goes out to the team as it stands, and their feedback decides
what is built next. This item holds M8 so PLAN.md can hold only what is next.

What M8 was: the align skill over `read`'s output and the hub markdown, which
composes the comparison, judges what matters with Nail's word on which side is
the source of truth, proposes both ways, never deletes from the hub, and works
on a document gdoc never published. SPEC.md "The diff (what `align` composes)"
and "The skills, and how a comment reaches one" still describe it.

The unknown that settles the value: whether anyone on the team needs a
document compared back to its note. Today a note is published once and later
changes travel as suggestions through the review skill, so the loop works
without alignment. If the team asks for a way to bring a document's edits back
into the hub, this is the milestone; if they do not, it stays here.

Three things were folded into M8 and are deferred with it:

- Whether a binary diff command earns its place at all, and with it the fourth
  dependency question, `sergi/go-diff`, proven in a spike. PLAN.md's standing
  facts still name it as the one open candidate.
- The hub-wide live session, one watch over every paired note in a folder,
  Nail's decision of 2026-09-07. It makes the cost of a poll matter, which is
  `wait-polls-both-apis-every-tick.md` in this directory.
- A second version of a note. `publish` refuses a paired note, and the
  refusal's own sentence, take the `gdoc:` block out by hand, is the answer
  until alignment says more. `restyle --new` is not built, decided 2026-09-11.
