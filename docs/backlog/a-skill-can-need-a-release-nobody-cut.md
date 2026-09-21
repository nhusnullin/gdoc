---
worth: yes
where: .github/workflows/nightly.yml:12
added: 2026-09-21
---
# a skill can need a release nobody cut

gdoc ships as two artefacts from one repository, and they travel on separate
roads at different speeds.

A skill goes out from `main` through the plugin marketplace, which the hub's
committed `.claude/settings.json` declares with auto-update on. It arrives on a
colleague's machine when the merge lands, with nothing for them to type.

The binary goes out as a GitHub release. It arrives when somebody types
`gdoc update`, and only as far as the newest tag. The nightly cuts `x.y.(z+1)`
when main moved. Only `make tag`, by hand, cuts `x.y.0`.

A skill states the binary it needs on its `needs` line, and that line is a
floor: each skill says `needs` is the one version that stops a session, and
that the answer is `gdoc update`. Nothing anywhere ties that floor to a version
that was ever released. A merge can therefore raise what every colleague's
skill demands while raising nothing that any colleague can install.

When the floor is above the newest tag, the failure is not a stale binary. It
is a dead end. The session stops on the `needs` line and gives the only advice
it has. The colleague types `gdoc update`, gets the newest release, which is
still below the floor, and the session stops again on the same line. There is
no command that fixes it, and nothing tells them that. The window opens at the
merge and closes when a person remembers a command written down in a plan.

Nothing catches it either. `TestEverySkillNamesTheGdocItNeeds` checks that the
line exists and parses as three numbers. No test compares it with a tag, with a
release, or with `plugin.json`, whose own version is bumped by the same hand
command and so drifts the same way.

## The case that surfaced it

M13 merged on 2026-09-20 with four skills carrying `needs: v2.4.0`. The nightly
cut `v2.3.5` minutes later, because a nightly only bumps the patch.
`plugin.json` still said `v2.3.0`. Every colleague's export and align skills
asked for a version that did not exist, and would not until `make tag`.

## What to decide

The narrow fix is a gate: fail when a skill's `needs` is ahead of the version
`plugin.json` carries, so the mismatch cannot reach `main`. It turns a silent
break on somebody else's machine into a red build here, and changes nothing
about how releases are cut.

The wider question is whether a version belongs in a skill at all. Four shapes,
none chosen:

- The merge cuts its own release. A change that raises a floor makes CI cut the
  matching `x.y.0`, so the two roads meet by construction.
- The nightly notices. It reads the skills and cuts the minor rather than the
  patch when one asks for a version no tag carries.
- The skill stops naming a version. It names the commands and flags it uses,
  and the binary answers whether it has them, so a capability check replaces a
  version check and a skill can never be ahead of a number.
- It stays by hand with the gate above, and `make tag` becomes the milestone's
  own last task rather than a line in its post-completion notes.

Until one is picked, the rule is that `make tag` happens in the same sitting as
the merge that raised a floor.

Nail picks.
