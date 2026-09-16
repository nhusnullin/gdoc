---
worth: yes
where: go/cmd/gdoc/read.go:388
added: 2026-09-16
---
# The live wait polls every ten seconds; Nail wants two to three

`waitInterval` is ten seconds, so a comment written into a document under a
live session is seen up to ten seconds after it lands. Nail asked on
2026-09-16 for two to three seconds, so a reply in the margin feels like a
conversation rather than a queue.

The number is one constant, and the change itself is one line. What makes it
an item rather than an edit is the cost per tick: today every tick is two
requests, the Drive listing and a whole-document read, whatever arrived. At
three seconds that is twenty document reads a minute on a quiet document, and
at two it is thirty. Fix `wait-polls-both-apis-every-tick.md` first, so a
quiet tick is one listing and nothing else, then shorten the interval. In that
order the faster tick costs listings alone, which are the cheap half.

Two things to check when it lands: the per-user, per-minute read quotas on
both APIs against the new rate, written into MEASURED.md with the date, and
the interrupted-wait test in `read_test.go`, which hands the wait a done
context and must stay green at the shorter interval. The skill's nine-minute
wait and its ten-minute Bash ceiling do not change.
