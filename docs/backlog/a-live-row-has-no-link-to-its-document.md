---
worth: yes
where: presence/Sources/
added: 2026-10-26
---
# A live row has no link to its document

Nail, 2026-10-26: every document the menu bar app shows must open on a click.
The NOW row for a live review ("Live · Embedded card: legal… tickets, since
11:58 · quiet 0 min") has only the stop button, so the document being watched
cannot be opened from the panel. Rows under DOCUMENTS do open theirs.

The record already carries the document id for a `comments --wait` call, so
the link is the same `https://docs.google.com/document/d/<id>/edit` the
DOCUMENTS rows build. What the fix has to settle:

- Where the click lands: the whole row, as in DOCUMENTS, or an open icon beside
  the stop button. The stop must stay hard to hit by accident.
- Every other NOW row, not only Live: a publish or a reply in flight names a
  document too, and the rule is "each document has a link".

A presence-only change, so a `ui` review and the design reviewer over the
renders.
