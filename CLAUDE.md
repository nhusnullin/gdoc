# Notes for AI assistants

Read [PRINCIPLES.md](PRINCIPLES.md) before proposing any design. It is three
constraints and the decisions that currently implement them, and it is short.

Read [README.md](README.md) for what the tool does and how to run it.

This file holds the specifics: what lives where, what each rule means in code,
and what never to do. The reasons live in PRINCIPLES.md, and they live there
only.

## What lives where

| Path | Holds |
|---|---|
| `gdoc/` | the package. Imports are `from gdoc.x import y` |
| `tests/` | pytest suite. Every module has a matching test file |
| `skills/` | `gdoc-review` and `gdoc-apply`. Symlinked into `~/.claude/skills/`, so edits are live |
| `docs/superpowers/` | the implementation plans and the design specs |
| `.gdoc/<slug>/` | `pending.md`, `baseline.md`, generated `out/*.docx`. **Not in this repo:** it sits beside the source markdown being reviewed |

Secrets and the venv live in `~/.config/gdoc-agent/`, never in this repo.

## One root, and it is never this repo

Principle 2. Read it there.

In code: `--repo-root` is the tree scanned for markdown with `gdoc:` frontmatter,
and the tree `.gdoc/` is written into. In practice it is the folder holding the
source document, in a vault such as `Altery-Platform-Hub`.

## The tool must work without git

Principles 1 and 3. This is the constraint an agent is most likely to break.

- Nothing may refuse to run because git is unavailable.
- `is_dirty` raising `BaselineConflict` is correct only when git exists and the
  command fails. Without git, `write_baseline` refuses to overwrite an existing
  file unless `force` says so, because not knowing must never resolve to
  "overwrite".
- The commit steps in `gdoc-apply` are conditional, and a skipped commit is
  always said out loud.

## No external programs

Principle 1. The rule is scoped to `gdoc/render/`, the publish path.

pandoc remains, in three places, and each is tracked separately:

- `gdoc/render/body.py`, as the markdown parser `build` depends on. Load-bearing.
  Removing it means a pure-Python parser plus a rewritten AST walker, in the
  module where the document body's pixel fidelity lives, so it needs its own
  spec.
- `gdoc/export.py`, as a markdown fallback. Drive exports `text/markdown`
  natively, verified against the live account, so this one looks removable on
  its own.
- `gdoc/generate.py`, for the plain non-template path, which is what
  `--template none` selects. Kept on purpose: it is the fallback when the house
  template is not wanted, and keeping it narrowed the blast radius of wiring the
  template in.

`tests/test_no_external_programs.py` enforces this. The five removed program
names must not appear in code under `gdoc/render/`, and the set of modules
there importing `subprocess` must be exactly `{body.py}`. That is an allowlist,
not a ban: it fails if `subprocess` spreads to another module, and it fails just
as loudly if body.py's own dependency vanishes without the test being updated.

Page numbers are the one real cost. They do not exist until something lays the
document out. The intended flow makes Google the layout engine: upload once with
blank numbers, read which page each heading landed on out of the PDF export, write
those numbers in, then upload the version that gets published.

`gdoc generate` runs that flow. It builds with blank page numbers, uploads that
copy, exports it as PDF, reads the pages back, builds again with the numbers in,
uploads the version that gets published, and trashes the measuring copy. The
published copy is measured too, so a contents list that disagrees with its own
document comes back in the result as `drift` rather than passing quietly.

The template comes from `--template`, or from `template` in the config, which
defaults to the bundled profile. `--template none` keeps the plain
`pandoc md -o docx` path.

`tests/test_contents_integration.py` still composes the two passes against live
Drive, and is opt-in behind `GDOC_LIVE_PUBLISH_TEST=1`. It is the only test that
creates real documents.

`gdoc.render.build` on its own has no pagination to offer, so it leaves the page
numbers blank. Any desktop refresh fills them in, and a wrong number would be
worse than a blank one.

## Skills are linked, not copied

`~/.claude/skills/gdoc-review` and `gdoc-apply` are symlinks into `skills/` in
this repo. There is one copy of each SKILL.md, so editing it here changes what
Claude Code loads. No copy step, and no way for the skill to drift from the CLI
it calls.

Two consequences worth holding:

- An edit is live the moment it is saved, before it is committed. Nothing warns
  you. `./install.sh` prints `+ uncommitted changes` when the tree is dirty.
- Deleting or moving `skills/` breaks the installed skills. Re-run
  `./install.sh` after any move.

`install.sh` is safe to re-run. It refuses to replace a real
`~/.claude/skills/<name>` directory whose contents differ from this repo, so an
older copy-based install cannot be destroyed silently.

## Testing

TDD. Write the failing test first.

```bash
~/.config/gdoc-agent/venv/bin/pytest
```

`tests/test_access_integration.py` calls Drive and skips without credentials.
Never make a test pass by loosening an assertion about what the credential can do.

## Never

- Never edit a reviewed Google Doc. The credential cannot, and neither may the agent.
- Never commit anything from `~/.config/gdoc-agent/`.
- Never post markdown into a comment thread. The CLI refuses it for a reason.

## Writing style

Plain, short English. No em dashes.
