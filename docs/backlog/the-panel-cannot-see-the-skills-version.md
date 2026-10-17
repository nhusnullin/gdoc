---
worth: later
where: presence/Sources/PresenceUI/Facts.swift:95
added: 2026-10-17
---
# The panel cannot see the skills version

The M24 blueprint drew a Skills row in HEALTH, reading something like
"v2.14.0 · altery plugin". Nothing can fill it.

`gdoc help --json` prints the commands, the settings and the update facts, and
no word about the skills. The five folders under `skills/` are symlinked into
`~/.claude/skills/` by `install.sh`, or copied there by the plugin, and the app
reads the record directory and its own plist and nothing else of somebody's
home. So the row would have been a blank row on every screen.

The row, the `versions.skills` field and the sentence in `docs/guide/presence.md`
are out until one of these is decided:

- `help --json` carries a `skills` object, which means the binary learning
  where the skills are and what version is in them. `SKILL.md` carries no
  version today, so that is a file format decision first.
- Or the app reads `~/.claude/skills/gdoc-review/SKILL.md`, which widens what
  the app reads by a third place and needs its own line in the invariants.

Worth `later`: nobody has asked what version of the skills they are running,
and `gdoc help` already says what version of the binary wrote them.
