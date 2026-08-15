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
ROOT="$PWD"
```

`$GDOC` is an installed command, so it runs from any directory.

One root, `$ROOT`, and it is `$PWD`:

- `$ROOT/.gdoc/<slug>/` holds `pending.md`, `baseline.md` and the generated
  `.docx` files.
- The paired markdown lives in `$ROOT` too, in the folder's normal structure.

`$PWD` is the CLI default for every `--repo-root`, so you never pass it.

`$ROOT` may not be a git repository. Every git step below is conditional.

## If Nail passes `--terminal-only`

Do Steps 1, 2 and 3: work through the items, edit the markdown, show the diffs,
commit if there is a repository. Then stop. Do not run Step 4: `$GDOC generate`
always tries to upload, so there is no local-only way to produce the document.
Tell Nail the markdown is saved and that re-running without the flag will
generate the document.

## Step 1: Load the context

The argument is a path to `pending.md`, usually `.gdoc/<slug>/pending.md`. Its
header holds two lines:

```
Document: https://docs.google.com/document/d/<doc_id>/edit
Source: <path to the source markdown, relative to the root>
```

Use `Source:` to find the paired markdown. Older queues have no `Source:` line,
so fall back to the document id:

```bash
$GDOC pair find --doc-id <doc_id>
```

If neither answers, Nail does not own this document. The output is a note for
him, not a new document. Say so and stop.

`baseline.md` in the same folder is the document as it was generated. It is read
only for comparison, and it is never the file you edit.

## Step 2: Work through the items in order

One item at a time. For each:

1. Say what you are about to change and where. Name the item's `Author` and
   what it was `Marked` with, both recorded in `pending.md`. An item from
   someone other than Nail, or one carrying `no marker`, is worth him seeing
   before you change anything.
2. Make the edit in the **paired markdown file**, never in `baseline.md`.
3. Show Nail the diff for that item alone.
4. Wait for approval before the next item.

Global changes are handled here, not in comment threads, because a diff is
reviewable and a comment thread is not.

## Step 3: Save the markdown

Commit only inside a git repository. Check first:

```bash
git -C "$ROOT" rev-parse --git-dir >/dev/null 2>&1
```

If that succeeds:

```bash
(cd "$ROOT" && git add <paired md file> && git commit -m "docs: apply global items from <doc name> review")
```

If it fails, skip the commit and say so plainly, in these words or close to
them: "Not a git repository, so nothing was committed. The markdown is saved on
disk." A silent skip would read as a successful commit.

Where there is a repository, commit before generating, so every generated
document corresponds to a commit.

## Step 4: Generate the new version

```bash
$GDOC generate --md <paired md file> --name "<doc name> v<n>" \
  --out "$ROOT/.gdoc/<slug>/out/v<n>.docx" \
  --baseline-root "$ROOT"
```

`--out` is chosen by intent:

| Case | `--out` |
|---|---|
| a new Drive version | `.gdoc/<slug>/out/v<n>.docx` |
| a document Nail asked for | `$PWD` |
| an explicit destination | that path |

This step generates a new version, so it uses the first row. The `.docx` is an
upload intermediate and stays out of sight.

`--baseline-root` is what makes the next apply able to see direct edits. On a
successful upload the new document is exported and written to
`.gdoc/<slug>/baseline.md`, replacing the previous version's snapshot. The JSON
reports the path as `baseline_path`.

Two outcomes, both fine:

- `doc_id` and `link` present: the new Google Doc exists. Give Nail the link.
- `doc_id` null and a `reason`: the `.docx` is on disk at `docx_path`. Give him
  that path and the reason. No baseline is written, because there is no document
  to compare against. Do not retry silently.

## Step 5: Record the version

If `doc_id` was null in Step 4, skip this step and go to Step 6. There is no
version to record.

```bash
$GDOC pair add-version --md <paired md file> \
  --version-id <new doc_id> --created <YYYY-MM-DD>
```

Then commit the updated frontmatter, again only inside a repository:

```bash
(cd "$ROOT" && git add <paired md file> && git commit -m "docs: record <doc name> v<n> in gdoc_versions")
```

Never record a version for a document that failed to upload.

## Step 6: Clear the items

Delete the applied items from `pending.md`. If all items are done, delete the
file. Say what you removed either way.

Inside a repository, commit it so the history shows what was asked:

```bash
(cd "$ROOT" && git add .gdoc/<slug>/pending.md && git commit -m "docs: clear applied items from <doc name> pending.md")
```

Outside one, say that the items were cleared and nothing was committed.

## Never

- Never edit the original Google Doc.
- Never edit `baseline.md`.
- Never write a `gdoc_versions` entry for a document that failed to upload.
- Never report a commit that did not happen.
