# Notes for AI assistants

Read [README.md](README.md) first. It explains what the tool does and why the
Commenter-only credential shapes the design.

## What lives where

| Path | Holds |
|---|---|
| `gdoc/` | the package. Imports are `from gdoc.x import y` |
| `tests/` | pytest suite. Every module has a matching test file |
| `skills/` | `gdoc-review` and `gdoc-apply`. Edit here, then copy to `~/.claude/skills/` |
| `docs/gdoc/<slug>/` | mirrors, `pending.md`, generated `out/*.docx` |
| `docs/superpowers/` | the implementation plan and the design spec |

Secrets and the venv live in `~/.config/gdoc-agent/`, never in this repo.

## Two roots, do not conflate them

`--repo-root` is not one concept:

- `export` and `capture` write mirrors and `pending.md` **into this repo**.
- `pair find` scans **the vault Nail is working in** (usually
  `~/src/altery/intelligence-hub`) for markdown with `gdoc:` frontmatter.

A bug here writes Altery notes into the wrong tree, so check which one a change
means before touching `--repo-root` handling.

## Skills are duplicated on purpose

`skills/*/SKILL.md` is the source. `~/.claude/skills/*/SKILL.md` is the loaded
copy. Change the source, then copy across, or the change does nothing.

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
