---
worth: later
where: go/internal/comments/wait.go
added: 2026-09-15
---
# A live session sees a new comment and never sees a suggestion accepted or rejected

`comments --wait` ends on the first window of comment activity after the cursor.
A suggestion accepted or rejected in the browser is not comment activity, so a
live session sits through it and says nothing. The next `suggestions --md` run
reports it in `gone_since_last_look`, which may be hours later or on another day.

What this costs a session is the receipt. gdoc proposes a change, Nail accepts
it in the document a minute later, and the session watching that document has no
way to know: it can still be waiting on a comment while the thing it asked about
has already been answered. The skill either asks for a `suggestions` run by hand
or reports the proposal as still pending.

Two shapes, and picking one is a decision rather than a patch.

A second poll inside the wait would read the document's pending suggestion ids
each tick and end the wait when the set shrinks. It is honest, and since
2026-09-18 it is the whole cost: a quiet tick is one listing and no document
read, so watching suggestions would put a document read back on every tick.
That is what `wait-polls-both-apis-every-tick.md` was filed to remove, and it
was removed the same day the tick went to two seconds.

The other is to read the ids only when the document is read anyway, which is
now the first tick and any tick with a new comment. That answers a session
that asked about a comment, and it still misses an accept with no comment
beside it, which is the case this item is about. Cheaper and only half an
answer; the two costs have to be weighed together.

Either way the field is a fact and not a verdict: what left the pending set, not
whether it was accepted or thrown away. Nothing in the binary can tell those
apart, which is the rule `gone_since_last_look` already holds.

Carried out of `docs/v2/PLAN.md`'s M4 leftovers, written 2026-09-07, lifted here
when PLAN.md was trimmed to open work on 2026-09-15.
