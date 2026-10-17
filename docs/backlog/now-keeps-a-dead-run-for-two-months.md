---
worth: later
where: presence/Sources/PresenceCore/Fold.swift:135
added: 2026-10-17
---
# NOW keeps a dead run and a card that ran out for as long as the record holds them

`Fold.glance` puts every start with no end into `active` and every held with no
ending into `holds`, with no age limit at all. `AppState` folds both month
files the tail delivers, so a start whose process died without writing an end
stays in NOW as "stopped, outcome unknown" for up to about two months, and an
app restart rebuilds it. A card nobody answered stays as "ran out" the same
way.

Only `comments --wait` traps SIGINT, so a run killed with Ctrl-C leaves a start
with no end. That is the ordinary way a terminal run ends, which makes this the
ordinary case rather than the rare one.

Neither lights the glyph: a dead pid is not live, and after
`testAHoldThatRanOutLeavesTheMenuBar` a hold past its `until` is not a pause
sign either. So this is the panel reading long after the fact, not an icon
telling a lie.

Two things sit under it:

- The spec accepts pid reuse as an unmeasured risk "within one day". NOW is not
  bounded by a day, so the fold asks `kill(pid, 0)` about pids that are weeks
  old, and macOS hands pids out again. A dead start from weeks ago can come
  back as `.running` for as long as the unrelated process lives.
- The app has no dismissals in this milestone. The spec's sentence about
  dismissals in `UserDefaults` describes M24, and "the stuck card's dismiss" is
  on M24's list, so there is no way for a person to clear a row either.

`worth: later` because the bound is a product decision and the obvious one is
wrong. Dropping active runs that are not in the local day of the moment, which
is what TODAY does, would take the mark off a live review that was started
before midnight and is still watching, which is the one run that matters most
and the longest-lived thing gdoc does. A horizon that is not a calendar day,
an age read off `started`, or a dismissal are three different answers, and M24
already carries the 14-day horizon and the dismiss.

What would settle it: Nail using M23 for a week and saying what NOW should
still be showing him the next morning.

Related: `docs/plans/completed/2026-10-17-gdoc-v2-m23-the-presence-spec.md`,
the row about the tail and `kill(pid, 0)`, and the M24 list in
`docs/plans/completed/2026-10-17-gdoc-v2-m23-the-presence.md`.
