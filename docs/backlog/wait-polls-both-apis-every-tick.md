---
worth: later
where: go/cmd/gdoc/read.go, cmdComments
added: 2026-09-15
---
# `comments --wait` reads the whole document on every tick, whether or not anything arrived

A poll is two requests: `comments.list` on Drive, then `documents.get` on Docs.
Both go out every ten seconds for as long as the wait runs, so a nine-minute
wait on a quiet document costs about 54 document reads that answer the same
bytes each time.

The document read is there because the ranges come from Docs, not from Drive:
`comments` reports a thread's character range, and the listing alone cannot say
where a thread sits. On a tick where the listing carried nothing new there is no
range to report, so the read bought nothing.

The cheaper shape is to list first and read the document only when the narrowed
window is non-empty. The order is already right for it: the listing goes out
before the document read on purpose, so a comment written in the gap is placed
by the next poll rather than reported unplaced. Read
`go/internal/comments/doc.go` under the poll order before changing anything
here.

`worth: later` because nothing has hit a quota. One reviewer on one document is
the whole load today, and the Drive and Docs read quotas are per-minute rather
than per-day. If a hub-wide session ever watches every paired note in a folder,
which is M8's, the tick cost multiplies by the number of documents and this is
the first thing to fix.

Carried out of `docs/v2/PLAN.md`'s M4 leftovers, written 2026-09-07, lifted here
when PLAN.md was trimmed to open work on 2026-09-15.
