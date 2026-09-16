---
worth: later
where: go/cmd/gdoc/update.go, go/internal/guard/params.go
added: 2026-09-16
---
# `gdoc update` reads one page of releases, so the ceiling moved rather than went

`gdoc update` asks GitHub for the releases listing with `per_page=100`, which is
GitHub's maximum for one page, and reads that page and no other. The guard
admits `per_page` on that call alone and admits no `page`.

The number is not arbitrary. The default is thirty, and the nightly cuts a
release most nights main moved, so a page of thirty stops carrying the last
hand-cut `x.y.0` about a month after it was cut. `Choose` then finds no stable
candidate and a bare `gdoc update` tells the colleague there is no stable
release at all, while `--nightly` keeps working, which reads as arbitrary. A
hundred buys about three months instead of about one. It does not remove the
problem, and a quiet quarter followed by a busy one still reaches it.

## Why paging was not just added

Walking pages is a read count the server decides. The guard can state the shape
of one bounded read; it cannot state the shape of "keep asking until you find
something", and `gapi.MaxListingBody` bounds a body rather than a sequence. So a
page walk needs a bound of its own written down first: how many pages, what a
run says when it hits that bound, and whether `Unreachable` or a plain failure
is the honest answer when the stable release is simply further back than gdoc
will look.

## Two shapes

A bounded walk: `page=1..N` with N a constant in the command, stopping as soon
as both channels have a candidate. Honest, and it makes the guard's allowlist
carry `page`, whose value it would hold to `1..N` the way `per_page` is held to
`1..100`.

A pruned releases page: the nightly deletes its own releases older than some
age, so the listing stays short by itself. Cheaper in code and it throws away
history a colleague might want to roll back to, which is a decision rather than
a patch.

Nail picks. Until then the ceiling is written here and in the comment on
`releasesURL`.
