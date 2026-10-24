---
worth: later
added: 2026-10-21
---
# The panel does not show the house logo

Nail asked for the house logo in the menu bar app. The app draws no picture of
the firm today.

The bytes are already on the machine: `internal/asset` caches the logo beside
the token under `assets/<id>`, after the first build or publish that reached
Drive. The app could read that file, but the invariants hold the app to three
things it reads: the record, its plist and the notes' `gdoc:` front matter. A
fourth is a decision and a new line in `CLAUDE.md`, the same as
`the-panel-cannot-see-the-skills-version.md` ran into.

Two ways in:

- The app reads the cached file directly. That widens what it reads, and the
  cache may not exist yet on a machine that never built a document.
- `help --json` names the cache path, or the binary hands the bytes over. The
  app already runs `help --json`, so no new read, but the binary learns to
  speak about a picture.

Worth `later`: the unknown is where the logo goes on the panel and what it is
for. The panel is a status surface and no row asks for a brand mark. Settle
that before choosing a route. It also depends on
`the-logo-drive-file-cannot-be-changed.md` if the logo can move.
