# gdoc

Review Google Doc comments from the terminal. The agent answers the `ai:`
comments in their threads, captures the ones that need document-wide changes,
and fetches a document as markdown so you can see what drifted.

The credential is a service account with **Commenter** access. It cannot edit a
document, and that limit is the design: replies go in comment threads, and
document-wide changes are applied to a paired markdown file and published as a
new version.

## The marker decides

`gdoc read` returns one actionable list, `addressed`. A comment is in it when it
carries the `ai` marker, is not resolved, and has no reply from the agent yet.

The author name is a label, not a gate. A marked comment from anyone on the
document is acted on, and Nail chooses the document: pointing the skill at it is
the trust decision. The name still travels in the payload, so an unexpected one
is visible in the report.

There was a `display_name` config field that split the list into Nail's comments
and everyone else's. It is gone. Drive returns no email address for comment
authors and display names are editable, so the split was a guess the skill had to
disclaim every run.

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

The skills drive the tool. Run them from the folder holding the source markdown:

```
/gdoc-review <google doc url>
/gdoc-apply .gdoc/<slug>/pending.md
```

`/gdoc-review` reads the comments, answers the local ones, and captures the
global ones to `pending.md`. `/gdoc-apply` works through those captured items,
edits the paired markdown, and generates a new document version.

Direct CLI use:

```bash
gdoc read <url>
gdoc reply <doc_id> <comment_id> --body-file reply.txt
gdoc capture <doc_id> <comment_id>
gdoc export <url>                       # markdown to stdout
gdoc export <url> --out fetched.md
gdoc pair find --doc-id <doc_id>
gdoc generate --md <paired.md> --out <out.docx> --baseline-root .
gdoc generate --md <paired.md> --name "<name>" --template none --out <out.docx>
```

## One root, and `.gdoc/` inside it

`--repo-root` is the same thing for every command: the directory the tool works
in, `$PWD` by default. No path in the package or the skills names a specific
repository.

Tool-managed files go in `.gdoc/`, beside the source markdown they belong to:

```
<folder holding the source md>/
  2026-08-13-topic.md              the source, hand written
  2026-08-13-topic.docx            generated, distributed
  .gdoc/
    2026-08-13-topic/
      pending.md                   captured items awaiting apply
      baseline.md                  the document as it was generated
      out/v2.docx                  upload intermediate
```

Not `docs/gdoc/`. Once the root can be any directory, `docs/` cannot be assumed
free or appropriate, and a dot-namespace signals tool-owned. It also keeps these
files out of Obsidian's search and graph.

The directory is named after the **source markdown file**, not the document
title and not the document id. Every iteration raises a new Google Doc with a new
id and usually a new version in its title, so both would fork the queue. The
source file survives, and its frontmatter records the whole lineage, so
reviewing an old version still lands in the one right directory.

`baseline.md` is the document as `generate` uploaded it, written at the one
moment the document and the markdown provably match. A later apply diffs it
against a fresh export to see what was edited directly in the document. Both
sides carry the same pandoc round-trip distortion, so it cancels.

`gdoc.render.build` writes the .docx directly from the house template, rather
than letting pandoc write it. It is a function, not a command: `gdoc --help`
lists read, reply, export, capture, generate and pair, and none of them exposes
it directly. `gdoc generate` calls it, which is how the house style reaches a
real document. Page numbers come from Google, so generate uploads twice: once
with blank numbers to measure the layout, once to publish. `--template none`
keeps the old `pandoc md -o docx` path.

`--name` is optional. Without it the new version is called
`<cover title> v<n>`, where n is the recorded version count plus one.

A note with no `title:` in its front matter is refused, because the cover and
the running head would be blank and the tool does not invent one. The refusal
names the file and suggests a title, taken from the first heading or from the
file name:

```json
{
  "missing": "title",
  "suggested_title": "Miguel kickoff call — 2026-08-12",
  "suggested_from": "h1"
}
```

The candidate is copied from the note exactly as written, punctuation and all,
because a suggestion the tool has quietly reworded is no longer the author's
own words.

The suggestion is a proposal, never a decision. Get it approved, then add the
title to the note, or pass `--title "..."` to publish once without editing it.

`build` uses pandoc only as the markdown parser that feeds the template. The
export side still uses pandoc as a markdown fallback, which is tracked
separately.

## No git requirement

The tool works without git. Nothing refuses to run because git is unavailable,
and `/gdoc-apply` skips its commit step and says so.

One consequence: outside a repository there is no way to tell an edit from a
stale copy, so an existing `baseline.md` is never overwritten unless `--force`
says so. `generate` passes it, because replacing the previous version's snapshot
is exactly its job.

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
docs/superpowers/        the plans and the design specs
```

`.gdoc/` directories live beside the documents being reviewed, not in this repo.

## Test

```bash
~/.config/gdoc-agent/venv/bin/pytest
```

Integration tests that call Drive are skipped unless credentials are present.
