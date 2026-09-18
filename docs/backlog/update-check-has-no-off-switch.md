---
worth: later
where: go/cmd/gdoc/notice.go, go/internal/lastcheck/lastcheck.go
added: 2026-09-18
---
# The daily release check has no off switch

`gdoc help` asks GitHub what is published when its stamp is missing or older
than 24 hours, and nothing turns that off. Nail's decision of 2026-09-18,
DECISIONS.md: the switch was considered and deferred, not forgotten.

## Why it was deferred

The cost is already bounded by the design. A machine that cannot reach GitHub
gets its failure written into the stamp with everything else, so it pays the
two-second ceiling once a day and not once a run. A machine that can reach
GitHub pays 0.6 to 0.8 seconds, once a day, on the one command that opens no
document. Measured on this machine, 2026-09-18. A switch would be a second way
to be in, a second state to test and a second sentence in every document that
describes the check, bought against a cost nobody has felt yet.

## What it would look like

`GDOC_NO_UPDATE_CHECK=1` in the environment, read in `notice` before the stamp
is read, returning no facts and no line, the way a checkout build already does.
An environment variable rather than a flag, because the caller is a skill
rather than a person, and rather than a config file, because gdoc's config
folder holds a token and a stamp and no settings, and adding the first setting
is a bigger decision than this one.

The one thing to get right is the object. A run with the check off carries no
`update` key, exactly like a checkout build, so a skill that reads those facts
needs no new case. Saying `"checked": false` would be a field that judges, and
`internal/` prints facts.

## What would make it worth adding

A colleague on a network that refuses GitHub slowly rather than quickly, so the
ceiling is paid in full every morning and it is visible. An air-gapped machine,
where reaching a host nobody named is a policy question and not a delay. Or a
place where `gdoc help` runs far more often than once a session, which would
mean something calls it in a loop and the loop is the real item.
