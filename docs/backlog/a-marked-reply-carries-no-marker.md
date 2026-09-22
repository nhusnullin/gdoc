---
worth: yes
where: go/internal/comments/comments.go:101
added: 2026-09-21
---
# A marked reply carries no marker, so nothing in a listing shows it is an instruction

`markerOf` runs on a thread's own content and nowhere else. `Thread` carries
`marker`; `Reply` carries `id`, `author`, `created`, `content` and `by_gdoc`,
and no marker. A session reading a listing sees every marked top-level comment
as a fact, and has to find a marked reply by reading the text itself.

The skill has the rule and the binary has no fact behind it. Step 2 of
`skills/gdoc-review/SKILL.md` says a marked comment **or reply** written after
gdoc's last robot reply is new work. The Step 3 and Step 8 report shapes name
only "unmarked replies": there is no line for a reply that is an instruction,
so a session that missed one prints a report that reads clean.

This is what makes a marked reply the easiest thing in a listing to drop. A
session narrowing a long listing keys on the fields the binary gives it, and
for replies there is nothing to key on. It happened on 2026-09-21, in the live
review of `1jgncELOexSdcGvhSWnLlVkvtAI-35pZY9PNOCd2tPWo`: a filter written over
thread markers dropped the replies wholesale, and an `ai!` reply went unseen.
The cursor defect that made it unrecoverable is its own item,
`a-live-cursor-can-jump-past-unread-activity.md`; this one is why the filter
looked complete.

The fix, both halves:

- `Reply` grows `marker`, from the same `markerOf` a thread uses, so the match
  stays exactly one of `ai:`, `ai?`, `ai!` as the first token. A marker is a
  fact about the text and judges nothing, so this sits inside the
  facts-not-verdicts invariant rather than against it.
- The skill's Step 3 and Step 8 report shapes grow the line that is missing: a
  thread already answered that now carries a marked reply is work, and the list
  says so beside the unmarked-reply line it already prints.

Printing the field and leaving the report shape alone gives the next session the
same wall of threads to scan.
