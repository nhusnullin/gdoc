# Versioning

One version for the whole repo: the CLI and both skills. `pyproject.toml`
holds it. `gdoc/version.py` reads it at runtime with
`importlib.metadata.version("gdoc")`. Nothing else may define a version
string.

## One version, not three

The skills are symlinks into `skills/`, and the CLI ships from the same
checkout (see `CLAUDE.md`). All three move together on every `git pull`, so a
skill can never run against a CLI from a different commit. Versioning them
separately would track a distinction that cannot happen in practice.

## Semver

`MAJOR.MINOR.PATCH`. The only consumers of this version are
`skills/gdoc-review` and `skills/gdoc-apply`, both in this repo. "Breaking"
means: a skill's documented CLI usage stops working.

- **MAJOR** — a command, flag, or JSON field a skill relies on is removed or
  changes meaning. Examples: `--repo-root` starts meaning something
  different, a JSON key a skill reads is renamed, a subcommand is removed.
- **MINOR** — a new command or flag is added, and every existing usage still
  works unchanged.
- **PATCH** — a bug fix. No documented behavior changes.

Bump the version in `pyproject.toml` in the same commit as the change that
needs it, not in a separate release commit. There is no tagging or publishing
step; the number in `pyproject.toml` is the release.

## Skills checking the CLI

A `SKILL.md` step that depends on a flag or JSON shape added after some
version should check for it first:

```bash
$GDOC version --min 0.2.0
```

This prints `{"version": "..."}` and exits 0 when the installed CLI is new
enough. Otherwise it prints `{"error": "needs gdoc >= 0.2.0, found 0.1.0"}`
and exits 1, so a skill fails with a sentence instead of an argparse error
partway through a review.

`gdoc --version` prints the bare version, for a human running the CLI
directly.
