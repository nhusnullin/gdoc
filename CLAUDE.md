# Notes for AI assistants

Read [README.md](README.md) first. It explains what the tool does and why the
Commenter-only credential shapes the design.

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

`--repo-root` means one thing for every command: the directory the tool works in,
`$PWD` by default. It is the tree scanned for markdown with `gdoc:` frontmatter,
and the tree `.gdoc/` is written into. In practice it is the folder holding the
source document, in a vault such as `Altery-Platform-Hub`.

Never reintroduce a default that names a repository, and never write tool files
into this repo. A bug here writes Altery notes into the wrong tree.

The queue directory is keyed by the **source markdown file**, not the document
title and not the document id. Both change on every iteration; the source file
does not.

## The tool must work without git

`Altery-Platform-Hub`, where the source documents live, is not a git repository
and will not become one. This is the constraint an agent is most likely to break.

- Nothing may refuse to run because git is unavailable.
- `is_dirty` raising `BaselineConflict` is correct only when git exists and the
  command fails. Without git, `write_baseline` refuses to overwrite an existing
  file unless `force` says so, because not knowing must never resolve to
  "overwrite".
- The commit steps in `gdoc-apply` are conditional, and a skipped commit is
  always said out loud.

## No external programs

`gdoc` runs where the source documents live, and that environment has no package
manager. Anything that cannot be pip installed cannot be installed at all. The
rule is scoped to `gdoc/render/`, the publish path.

LibreOffice and poppler were removed for this reason. That was 833MB of programs
that cannot be copied into an isolated environment. Three external programs
became one.

pandoc remains, in three places, and each is tracked separately:

- `gdoc/render/body.py`, as the markdown parser `build` depends on. Load-bearing.
  Removing it means a pure-Python parser plus a rewritten AST walker, in the
  module where the document body's pixel fidelity lives, so it needs its own
  spec.
- `gdoc/export.py`, as a markdown fallback. Drive exports `text/markdown`
  natively, verified against the live account, so this one looks removable on
  its own.
- `gdoc/generate.py`, for the plain non-template path, which PR #5 Task 6
  retires.

`tests/test_no_external_programs.py` enforces this. The five removed program
names must not appear in code under `gdoc/render/`, and the set of modules
there importing `subprocess` must be exactly `{body.py}`. That is an allowlist,
not a ban: it fails if `subprocess` spreads to another module, and it fails just
as loudly if body.py's own dependency vanishes without the test being updated.

Page numbers are the one real cost. They do not exist until something lays the
document out. The intended flow makes Google the layout engine: upload once with
blank numbers, read which page each heading landed on out of the PDF export, write
those numbers in, then upload the version that gets published.

**That flow is not wired into a command yet.** `gdoc generate` still runs
`pandoc md -o docx` and never imports `gdoc.render`. The design spec parks the
wiring for PR #5 Task 6. The only place the two passes are composed today is
`tests/test_contents_integration.py`, which needs the live Drive API and is opt-in
behind `GDOC_LIVE_PUBLISH_TEST=1`.

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
