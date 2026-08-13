---
name: gdoc-apply
description: Use when Nail wants the captured global items from a Google Doc review applied. Edits the paired markdown, then generates a new Google Doc version from it.
---

# Apply global doc items

Work through `pending.md` with Nail, edit the paired markdown, and generate a
new document version from it. The original document is never edited.

## Setup

```bash
GDOC="$HOME/.config/gdoc-agent/venv/bin/gdoc"
GDOC_REPO="$HOME/src/personal/gdoc"
VAULT="$PWD"
```

`$GDOC` is an installed command, so it runs from any directory.

Two roots, and they are not the same thing:

- `$GDOC_REPO` holds `pending.md`, the mirror, and the generated `.docx` files.
- `$VAULT` is the repo Nail is working in. The paired markdown lives there, and
  so do the commits that record edits to it.

## If Nail passes `--terminal-only`

Do Steps 1, 2 and 3: work through the items, edit the markdown, show the diffs,
commit. Then stop. Do not run Step 4: `$GDOC generate` always tries to upload,
so there is no local-only way to produce the document. Tell Nail the markdown is
committed and that re-running without the flag will generate the document.

## Step 1: Load the context

The argument is a path to `pending.md`, relative to `$GDOC_REPO`. Its header line is:

```
Document: https://docs.google.com/document/d/<doc_id>/edit
```

Extract the `doc_id`. Read `mirror.md` in the same folder. Then find the
paired markdown file in the vault:

```bash
$GDOC pair find --repo-root "$VAULT" --doc-id <doc_id>
```

If `pair find` returns nothing, Nail does not own this document. The output is
a note for him, not a new document. Say so and stop.

## Step 2: Work through the items in order

One item at a time. For each:

1. Say what you are about to change and where.
2. Make the edit in the **paired markdown file**, never in the mirror.
3. Show Nail the diff for that item alone.
4. Wait for approval before the next item.

Global changes are handled here, not in comment threads, because a diff is
reviewable and a comment thread is not.

## Step 3: Commit the markdown

```bash
(cd "$VAULT" && git add <paired md file> && git commit -m "docs: apply global items from <doc name> review")
```

Commit before generating. Every generated document must correspond to a commit.

## Step 4: Generate the new version

```bash
$GDOC generate --md <paired md file> --name "<doc name> v<n>" \
  --out "$GDOC_REPO/docs/gdoc/<slug>/out/v<n>.docx"
```

Two outcomes, both fine:

- `doc_id` and `link` present: the new Google Doc exists. Give Nail the link.
- `doc_id` null and a `reason`: the `.docx` is on disk at `docx_path`. Give him
  that path and the reason. Do not retry silently.

## Step 5: Record the version

If `doc_id` was null in Step 4, skip this step and go to Step 6. There is no version to record.

```bash
$GDOC pair add-version --md <paired md file> \
  --version-id <new doc_id> --created <YYYY-MM-DD>
```

Then commit the updated frontmatter:

```bash
(cd "$VAULT" && git add <paired md file> && git commit -m "docs: record <doc name> v<n> in gdoc_versions")
```

Never record a version for a document that failed to upload.

## Step 6: Clear the items

Delete the applied items from `pending.md`. If all items are done, delete the
file. Commit either way, so the history shows what was asked.

```bash
(cd "$GDOC_REPO" && git add docs/gdoc/<slug>/pending.md && git commit -m "docs: clear applied items from <doc name> pending.md")
```

## Never

- Never edit the original Google Doc.
- Never generate before the markdown is committed.
- Never write a `gdoc_versions` entry for a document that failed to upload.
