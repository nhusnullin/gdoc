---
worth: yes
where: go/internal/guard/policy.go:663
added: 2026-09-19
---
# A live review shows what it is doing in the thread, and the trace goes when the answer lands

Nail, 2026-09-19: in a live review, while the AI is working on a comment, the
thread should show that something is going on and what. The progress line is
updated in place, not appended, because a thread is not a log and nobody
wants the AI's working history in it. Once the answer is posted, the progress
is gone. v1 had something like this, and it was not bad; v2 can do it smarter.

On the v1 claim: the tree retired on 2026-09-15 holds no such feature. Its
Python module never called `replies.update` or `replies.delete`, and its
`gdoc-apply` skill says nothing about progress. What v1 did have, the memory
may be of, was a slower loop with more terminal output. This item is the ask
on its own merits.

## What stands in the way, and each is a decision rather than a patch

- **The rule.** SPEC.md under `reply`, 2026-08-29: a long action writes at
  most two messages into its thread, an acknowledgment and a receipt, as two
  replies and never an edit of one, and never a progress feed, because the
  margin is the readers' room and the terminal is the operator's. The skill
  restates it under "Never a progress feed". A `yes` here is a DECISIONS.md
  entry first, and it is a reversal of that entry, not a widening.
- **The guard.** `commentItemWrites` in `internal/guard/policy.go` refuses
  PATCH and DELETE on any comment or reply, for everyone, at every level.
  Finding 3 of the M1 review: a path names a reply id, and nothing in the id
  says who wrote it, so an edit on R1 is as likely to rewrite somebody else's
  words as gdoc's own. `TestNoCommentPatchOrDelete` and
  `TestADrivePathIsJudgedBySegmentCount` pin it. A progress line updated in
  place is `replies.update` on gdoc's own reply, and its removal on landing
  is `replies.delete`. Both come back the way GrantInPlace came back at M7b:
  a per-run grant naming one reply id, opened by the command that posted it,
  never inherited, and the guard checks the id it was told rather than the
  author. Whatever the author field says decides nothing, which is the
  identity invariant.
- **The plaintext rule.** A progress line is written into the thread, so it
  opens with the robot prefix and carries no markdown like every reply. The
  same package checks it. Nothing new there.

## The shape the ask points at

One reply per thread per action, posted when the work starts, rewritten as
the work moves, and deleted when the answer lands. The answer is a fresh
reply, so a thread that ends holds the question and the answer and nothing
between. A failure keeps the last progress line in place rather than deleting
it, because a thread that ends in nothing after a "working" line is the
dangling "on it" the current rule exists to prevent.

Two things the binary needs, and one thing the skill needs:

- `reply` grows a way to rewrite one reply it wrote in this run, and a way to
  delete it, each verified by reading the thread back through comments.get,
  the way `Post` already verifies. The grant is opened on the id `Post`
  returned and dies with the process. A run that crashed leaves its progress
  line behind, and the next tick sees a robot reply with no answer after it,
  which is a fact the skill can act on rather than a state nobody records.
- The daily-tick cost matters, because the live loop is one call on a quiet
  document. Each progress rewrite is a write plus a read-back, so the skill
  rewrites on a change of step and not on a timer.
- The skill decides what a step is. Reading the thread, reading the hub,
  drafting, verifying. Four lines at most, in reader language, and the one
  reply holds the current one alone.

Deleting a reply on a document that arrived by import is refused by Drive
for everyone, BLOCKED-BY-API.md. gdoc's own replies are not imported, so the
delete lands, but the read-back has to say so rather than assume it.
