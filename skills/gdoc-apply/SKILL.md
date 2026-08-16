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

1. Say what you are about to change and where.
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
$GDOC generate --md <paired md file> \
  --out "$ROOT/.gdoc/<slug>/out/v<n>.docx" \
  --baseline-root "$ROOT"
```

No `--name`. The tool names the version `<cover title> v<n>` itself, counting the
versions already recorded in the note. A name you write can only disagree with
that count.

The `<n>` in `--out` is a local file name and nothing else. Read the current
count with `$GDOC pair show --md <paired md file>` and add one.

`--out` is chosen by intent:

| Case | `--out` |
|---|---|
| a new Drive version | `.gdoc/<slug>/out/v<n>.docx` |
| a document Nail asked for | `$PWD` |
| an explicit destination | that path |

This step generates a new version, so it uses the first row. The `.docx` is an
upload intermediate and stays out of sight.

`--baseline-root` does two things, and the next apply needs both. On a successful
upload the new document is exported into `.gdoc/<slug>/baseline.md`, replacing
the previous version's snapshot, reported as `baseline_path`. It also records the
new version in the note's `gdoc_versions`, reported as `version`.

**Do not run `$GDOC pair add-version` after this.** The generate above already
recorded the version. `add-version` does not check for duplicates, so a second
call records the same document twice and the next version is numbered one too
high. That command is for repairing a pairing by hand, nothing else.

### Which template

`--template` defaults to the house style named in the config, so leave it off for
anything Nail will share. It gives the document a cover, the control tables and a
contents list.

`--template none` makes a plain document with none of those: no cover, no control
tables, no contents list. Use it only when Nail asks for a plain document.

### The title

The house style needs a `title` in the note's front matter. It fills the cover
and the running head, and the version name is built from it. `--template none`
needs no title, because there is no cover to put one on.

`--title "<text>"` sets the cover title for this run alone. It writes nothing
into the note, so the next run without it asks again.

### The four outcomes

Read the JSON. `doc_id` decides whether a document exists. Nothing else does.

| Outcome | How you tell | What you do |
|---|---|---|
| Published | `doc_id` and `link` are set, `drift` and `reason` are null | Give Nail the link. Carry on. |
| Published, with a warning | `doc_id` is set, and one of `drift`, `reason`, `baseline_error`, `pairing_error` says something | Give him the link first, then say what did not get recorded. |
| Nothing created | `doc_id` is null and `reason` says why | Give him `docx_path` and the reason. Do not retry silently. |
| Missing title | exit 1, a top-level `error`, with `missing`, `md`, `suggested_title`, `suggested_from` and `hint` | Stop and ask Nail. See below. |

`drift` and `reason` are always in the JSON, and are null when there is nothing
to say. `baseline_error` and `pairing_error` appear only when that write failed.

**Published, with a warning.** The document is real. Do not read a warning as a
failure and skip ahead, and do not bury the link under it. Give the link, then
say plainly what each key means:

| Key | What it means |
|---|---|
| `drift` | the page numbers in the contents list disagree with the document, as `{heading: [written, published]}`. Tell him to check the contents page before sharing it |
| `reason` | a note about a document that was still created, such as a measuring copy left in the Drive folder for him to delete |
| `baseline_error` | `.gdoc/<slug>/baseline.md` still describes the previous version, so the next apply cannot see direct edits |
| `pairing_error` | the version is missing from `gdoc_versions`, so the next version reuses this one's number |

**Missing title.** Stop there. Nothing is written and nothing is published until
Nail picks a title.

`suggested_title` is mechanical. It is the first `# heading` in the note, or,
failing that, the file name with a leading date stripped and the hyphens and
underscores turned into spaces. `suggested_from` says which: `h1` or `filename`.
The tool has never read the note, so a filename suggestion is a file name wearing
a title's clothes.

You have read the note. Reading it and proposing a better title is your job, so
do it: a kickoff note about ASV assessment scope should be called that, not
called after its file.

Give Nail three options, numbered, and wait:

1. **What the tool extracted:** `<suggested_title>`, from `<suggested_from>`.
2. **What you propose from the content:** your title, with one line saying what
   in the note it came from.
3. **One he types himself.**

Label options 1 and 2 as what they are. They are different kinds of answer, and
he should see both before choosing. Never choose for him, whichever you prefer.
The title goes on the cover of a document other people read, and every version is
named after it.

Once he picks, decide where it goes, and say which you are doing:

| Where | Command | When |
|---|---|---|
| the note's front matter | add one line, `title: <approved>` | the normal case. He will publish this note again, and the title is then decided once and reused |
| this run only | `--title "<approved>"` | a one-off document, or a title he is still unsure about. The note is untouched, so the next run asks again |

Prefer the front matter. Reach for `--title` only when the run really is a
one-off. Adding the line to the front matter changes nothing else in the note.
Then run Step 4 again.

## Step 5: Clear the items

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
- Never run `pair add-version` after `generate`. The version is already recorded.
- Never write a `gdoc_versions` entry by hand for a document that failed to upload.
- Never invent a title, and never approve the suggested one on Nail's behalf.
- Never report a commit that did not happen.
