---
worth: yes
where: .github/workflows/nightly.yml:12
added: 2026-09-21
---
# a skill can need a release nobody cut, and `gdoc update` cannot fix it

M13 merged on 2026-09-20. Four skills came with it carrying `needs: v2.4.0`.
The newest release is `v2.3.5`, which the nightly cut minutes after that merge.
No `v2.4.0` exists, and none will until somebody runs `make tag` by hand.

The two halves travel apart and move at different speeds. A skill reaches a
colleague from `main` through the plugin marketplace, whose auto-update the
hub's `.claude/settings.json` leaves on, so the new `needs` line arrives the
moment the merge lands. The binary reaches the same colleague from a GitHub
release. The nightly cuts `x.y.(z+1)` and only `make tag` cuts `x.y.0`, so a
merge can raise what a skill demands without raising what any release offers.

What the colleague sees is worse than a stale binary. Each skill says the
`needs` version is the one version that stops a session, and that the answer is
`gdoc update`. Typing it installs `v2.3.5`, which is still behind `v2.4.0`, so
the session stops again on the same line. The advice is the only advice the
skill has, and it cannot work. The window opens at the merge and closes when a
person remembers a command the plan wrote down. It is open right now.

Nothing catches it. `TestEverySkillNamesTheGdocItNeeds` checks that the line
exists and parses as three numbers, not that the version it names was ever
released, and there is no test that compares it with anything. `plugin.json`
still says `v2.3.0`, so the plugin's own version is two minors behind the
skills it ships.

## What to decide

The narrow fix is a gate: fail when a skill's `needs` is ahead of the version
`plugin.json` carries, so the mismatch cannot reach `main` at all. That turns a
silent break into a red build and changes nothing about how releases are cut.

The wider question is whether the version belongs in the skill. Four shapes,
none chosen:

- The merge cuts its own release. A milestone that raises `needs` makes CI cut
  the matching `x.y.0`, so the two land together by construction.
- The nightly notices. It reads the skills, and cuts the minor rather than the
  patch when one asks for a version no tag carries.
- The skill stops naming a version. It names the commands it uses, and the
  binary answers whether it has them, so a capability check replaces a version
  check and a skill can never be ahead of a number.
- It stays by hand, with the gate above, and `make tag` moves into the
  milestone's own last task rather than its post-completion notes.

Until one is picked, the rule is that `make tag` happens in the same sitting as
the merge that raised a `needs` line.

Nail picks.
