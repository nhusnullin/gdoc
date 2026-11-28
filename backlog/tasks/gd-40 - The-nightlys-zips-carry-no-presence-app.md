---
id: GD-40
title: The nightly's zips carry no presence app
status: To Do
assignee: []
created_date: '2026-11-28T10:50:00Z'
labels:
  - worth-later
dependencies: []
ordinal: 40000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
2026-10-18. `release.yml` has two jobs: a macOS job builds and signs
`GdocPresence.app`, an ubuntu job packs it into both darwin zips. `nightly.yml`
has the ubuntu half only, so a nightly zip has the binary and no app. Under the
`presence` setting, `gdoc update --nightly` would then replace the binary and
leave the app behind, or warn, depending on `appHere`; nobody has measured
which. The same day a one-line fix to the panel (the sign-out confirm, PR 91)
had to go out as v2.17.0, because `make tag` keeps patch versions for the
nightly and a nightly could not carry the fix.

Two ways out, Nail's call: give `nightly.yml` the macOS job too, so every zip
is one shape; or let `make tag` cut a patch version by hand when the change is
a fix, and keep the nightly for what it is. The first is the cheaper rule to
remember.

Added 2026-10-18 as docs/backlog/nightly-zips-carry-no-presence-app.md, worth: later.
<!-- SECTION:DESCRIPTION:END -->
