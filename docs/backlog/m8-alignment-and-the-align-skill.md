---
worth: later
added: 2026-09-16
---
# The hub-wide live session, and the fourth dependency

M8 was alignment whole, deferred on 2026-09-16 so the release could go first.
Most of it landed on 2026-09-19 in M13: `gdoc-align` is built, it composes the
comparison over `gdoc export` and the note, it merges into the hub with
agreement, it proposes into the document as suggestions, and it never deletes
what only the note holds. Two things were folded into M8 and neither landed.
This item holds those, so PLAN.md can hold only what is next.

**A diff command in the binary, and `sergi/go-diff`.** M8's open question was
whether the binary should compose the comparison itself, and with it whether
gdoc takes a fourth dependency. M13 answered it by not needing one: the skill
reads two files, the exported copy and the note, and judges paragraph by
paragraph, which is judgement and belongs to the skill by the rule the whole
product is built on. So the candidate stays a candidate and nothing asks for
it today. What would reopen it: a merge that is too big for a session to hold,
or a person asking for a machine-readable diff in the envelope. Either one is
a decision written into SPEC.md first, because that is where a dependency's
reason lives.

**The hub-wide live session.** One watch over every paired note in a folder,
rather than one document per session, Nail's idea of 2026-09-07. A session
today names one document. The unknown that settles it is whether anybody wants
to leave one running over a folder for a day, and the thing that makes it
costly is the poll: a quiet tick has been one listing and no document read
since 2026-09-18, so the cost is linear in the number of documents watched.
The block being a list of documents since M13 makes the walk easier, because
one note may now name several.
