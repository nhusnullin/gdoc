---
worth: later
where: go/internal/guard/policy.go, commentWrites
added: 2026-09-15
---
# A withdrawn proposal leaves its comment behind, and nothing can take it back

`withdraw` rejects gdoc's own suggestion, so the words go back to what they
were. The comment gdoc wrote beside that suggestion stays where it is, anchored
to a range that no longer holds a proposal. The skill replies into the thread
saying the proposal was withdrawn, which is the whole of the cleanup.

That is the guard working rather than a hole in it. `commentWrites` carries
`POST` and nothing else: no `PATCH` and no `DELETE`, on a comment or on a reply.
Nothing in a comment id says who wrote it, so the guard cannot tell gdoc's own
comment from somebody else's, and a delete door opened for gdoc's cleanup is a
delete door over every comment in the document.

What it costs is a document that collects dead threads. A reviewer who proposes
and withdraws three times leaves three threads whose only content is gdoc
explaining itself, and a person reading the margin cannot tell at a glance which
of them still asks for something.

The shape that would work is the one `AllowReject` already has: a per-run grant
naming exactly the comment id the note's `proposals[]` records as gdoc's own,
opening exactly `comments.delete` on that id, seeded before the session is built
and dying with the process. Provenance is the permission, the way it is for the
withdrawal itself. Read `go/internal/guard/doc.go` under the per-run grants and
`go/internal/withdraw/doc.go` before building it.

`worth: later` because the reply already says what happened and nobody has
complained about the clutter. Widening what the guard carries on a comment is
Nail's decision, not a refactor, and it should be taken when a caller needs it
rather than in advance.

PLAN.md said M7b or M8 would add the method back beside a caller. Neither needed
one. Carried out of `docs/v2/PLAN.md`'s M3 and M6 leftovers, written 2026-09-07,
lifted here when PLAN.md was trimmed to open work on 2026-09-15.
