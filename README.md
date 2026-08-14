# gdoc

Review Google Doc comments from the terminal. The agent answers the `ai:`
comments in their threads, captures the ones that need document-wide changes,
and mirrors a document to markdown.

The credential is a service account with **Commenter** access. It cannot edit a
document, and that limit is the design: replies go in comment threads, and
document-wide changes are applied to a paired markdown file and published as a
new version.

## Install

```bash
./install.sh
```

It creates the venv at `~/.config/gdoc-agent/venv`, installs the package in
editable mode, and links `skills/gdoc-review` and `skills/gdoc-apply` into
`~/.claude/skills/`. It prints the commit you are running.

Safe to re-run. Every step checks the current state first.

### Updating

`git pull` is the whole update. The package is an editable install and the
skills are symlinks, so both halves follow the working tree with no reinstall.

Re-run `./install.sh` only after changing dependencies in `pyproject.toml`, or
after adding a new skill directory.

One consequence of linking: an uncommitted edit to a `SKILL.md` is already live
in every Claude Code session. `install.sh` prints `+ uncommitted changes` when
the working tree is dirty, so you can tell what you are actually running.

## Configure

Two files, both outside this repo, neither in git:

- `~/.config/gdoc-agent/sa-key.json` — the service account key.
- `~/.config/gdoc-agent/config.json` — settings, including the key path and the
  Drive folder new versions are written to.

Give the service account address Commenter access on any document you want
reviewed.

## Use

The skills drive the tool. From the repo you want to work in:

```
/gdoc-review <google doc url>
/gdoc-apply docs/gdoc/<slug>/pending.md
```

`/gdoc-review` reads the comments, answers the local ones, and captures the
global ones to `pending.md`. `/gdoc-apply` works through those captured items,
edits the paired markdown, and generates a new document version.

Direct CLI use:

```bash
gdoc read <url>
gdoc reply <doc_id> <comment_id> --body-file reply.txt
gdoc capture <doc_id> <comment_id> --slug <slug> --repo-root ~/src/personal/gdoc
gdoc export <url> --repo-root ~/src/personal/gdoc
gdoc pair find --repo-root <vault> --doc-id <doc_id>
gdoc generate --md <paired.md> --name "<name>" --out <out.docx>
```

## Two roots

`--repo-root` means different things per command, and the difference matters:

| Command | `--repo-root` is |
|---|---|
| `export`, `capture` | where mirrors and `pending.md` are written: this repo |
| `pair find` | the vault scanned for markdown paired via `gdoc:` frontmatter |

A paired note carries this frontmatter:

```yaml
gdoc: <doc_id>
gdoc_synced: 2026-01-31
gdoc_versions:
  - id: <doc_id>
    created: 2026-01-31
```

## Layout

```
gdoc/                    the package
tests/                   pytest suite
skills/                  gdoc-review and gdoc-apply, symlinked into ~/.claude/skills/
docs/gdoc/<slug>/        mirror.md, pending.md, out/
docs/superpowers/        the plan and the design spec
```

## Test

```bash
~/.config/gdoc-agent/venv/bin/pytest
```

Integration tests that call Drive are skipped unless credentials are present.
