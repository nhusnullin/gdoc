---
worth: yes
where: skills/gdoc-review/SKILL.md:520
added: 2026-09-21
---
# A live session can take a cursor from a listing it did not read, and the skipped activity never comes back

Every `comments` answer carries a `cursor`, and it means the same thing in all
of them: the newest activity this call saw. A window's cursor is safe to keep,
because the session just read that window. A plain full listing's cursor is not,
because it covers the whole document whether or not the session read what came
back. The live loop in the skill says "set `CURSOR` to the answer's `cursor`"
and never says which answers that holds for.

Nothing brings the skipped span back. Windows carry activity after the cursor,
so a thread whose only news was inside the skipped span is silent from then on,
until somebody touches it again.

## What it cost, 2026-09-21

Live review of "B2B2C crypto card concept: the frozen vocabulary"
(`1jgncELOexSdcGvhSWnLlVkvtAI-35pZY9PNOCd2tPWo`). From the session transcript,
`8ad5fa85-08e9-41f1-9051-528b1985b206`:

- 08:44:53 a window comes back, its cursor at 08:44:50.973.
- 08:45:54 Nail replies inside the already-answered `ai?` thread `AAACAqfstkQ`
  with `ai! b2b2c is M2, mark in hub, and change here`.
- 08:46:18 the session runs a plain `gdoc comments <url>`, mid-session, and
  pipes it through a `python3 -c` one-liner printing thread id, marker,
  resolved, content and quote. Replies are dropped by the pipe. The reply text
  never enters the session's context: it appears in no tool result before
  09:04:09.
- 08:47:44 the loop resumes with `--since` at 08:46:09.401, that listing's
  cursor. The span 08:44:50.973 to 08:46:09.401 is now behind the cursor,
  unread.
- Every later window is clean, because nothing touches that thread again.
- 09:04:05 Nail posts an empty reply in the thread. That moves its `modified`,
  the next window carries it, and the work lands at 09:05:20. Eighteen minutes
  between the instruction and the nudge that recovered it.

## The fix, and the decision inside it

The skill half is small and certain: name which cursors may be kept. A window's
cursor, yes. Any other listing's cursor, only when the session read the whole
answer. In a live session, re-read with `--since "$CURSOR"` rather than bare, so
the answer is a window and its cursor is one the session has earned. Never
narrow a listing through a pipe and then keep its cursor: the rule already
exists for a failed poll, "that window is unread, not empty", and this is the
same rule with the session rather than the network doing the losing.

The binary half is a decision. A cursor is opaque base64, so a session cannot
compare the one it holds with the one it was handed, and cannot see that a
cursor jumped over a span. Options, and picking one is Nail's:

- Print the cursor's own timestamp beside it as a plain field, so a session can
  see a jump and say so.
- Echo `since` in the envelope of any `--since` call, so the window's span is
  visible at both ends.
- Leave it to the skill. The facts are already in the answer; what failed was a
  rule, and a rule is cheaper than a field.

Nothing here is the wait's fault. The wait delivered every window it was asked
for, and the thread's `modified` did move when the reply landed.
