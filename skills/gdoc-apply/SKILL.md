---
name: gdoc-apply
description: Use when Nail wants the captured global items from a Google Doc review applied, or wants a markdown note published as a Google Doc for the first time. Edits the paired markdown, then generates a new Google Doc version from it.
---

# Apply global doc items

Work through `pending.md` with Nail, edit the paired markdown, and generate a
new document version from it. The original document is never edited.

A note that has never been published has no queue, and publishing it is then the
whole job. See "The first publish" in Step 1.

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

## If the credential is not working

`$GDOC auth status` says which credential is in use and what is broken. It never
fails. `$GDOC auth login` signs Nail in and switches to oauth, with no OAuth
client to create first, and it waits for him to approve in the browser;
`$GDOC auth use service_account` switches back. Ask before running either, and
never edit `~/.config/gdoc-agent/config.json` by hand.

Under `oauth` the publish folder does not have to be shared with anything. Under
`service_account` it does, as Content manager, and a refused upload usually means
that share is missing rather than that the folder id is wrong.

## If Nail passes `--terminal-only`

Do Steps 1, 2 and 3: work through the items, edit the markdown, show the diffs,
commit if there is a repository. Then stop. Do not run Step 4: `$GDOC generate`
always tries to upload, so there is no local-only way to produce the document.
Tell Nail the markdown is saved and that re-running without the flag will
generate the document.

## Step 1: Find the queue

Nail passes whatever he already has: his markdown file, the document link, or
nothing. Never ask him for a path inside `.gdoc/`. He should not have to know a
slug to work on his own document.

| He passes | You do |
|---|---|
| nothing | list the queues under `$ROOT` |
| his markdown, `notes.md` | find the queue whose `Source:` line names it |
| the document URL | `$GDOC pair find --doc-id <doc_id>` for the source, then as above |
| `.gdoc/<slug>/pending.md` | use it directly |

The last row still works, because a path that names the file needs no resolving.
It is not the form to suggest.

Listing the queues:

```bash
ls "$ROOT"/.gdoc/*/pending.md 2>/dev/null
```

- One file: use it, and name the document it belongs to before you start.
- Several: show each with its document name and item count, and ask which. Never
  choose for him.
- None: say nothing is queued under this root, and that `$ROOT` is `$PWD`, so the
  usual cause is being in the wrong folder.

Given his markdown instead, read the `Source:` line of each queue and match the
one that names that file. If no queue names it, say so rather than guessing at a
directory name: an empty queue and a queue you failed to find look identical to
him, and only one of them is safe to proceed from.

### The first publish

A markdown file that exists, with no queue and no `gdoc:` in its front matter, is
not a stranger's document. It is step one of the loop this skill serves, and
publishing it is the whole job.

Say so and get one confirmation:

```
2026-08-13-topic.md has no queued items and has never been published.
Publish it as a new Google Doc? y
```

Then skip Steps 2 and 3, because there are no items to work through and no edits
to commit, and go to Step 4. `generate` writes the first `gdoc:` and the first
`gdoc_versions` entry itself, so nothing has to be paired first.

Never run `pair set` to create that first pairing. Step 4 does it.

Every `pending.md` header holds two lines:

```
Document: https://docs.google.com/document/d/<doc_id>/edit
Source: <path to the source markdown, relative to the root>
```

`Source:` is how you get from the queue to the paired markdown. Older queues have
no `Source:` line, so fall back to the document id:

```bash
$GDOC pair find --doc-id <doc_id>
```

If neither answers **and he gave you a document link**, there is no source
markdown for it here. That is a document he does not own the source of, so the
output is a note for him, not a new document. Say so and stop.

If he gave you a markdown file that exists, that same empty answer means the first
publish above, not a refusal. The difference is whether a source file is in his
hands, so check which he passed before you refuse anything.

`baseline.md` in the queue folder is the document as it was generated. It is read
only for comparison, and it is never the file you edit. It normally sits beside
`pending.md`, but Step 4 reports `baseline_path`, and that is the truth if the
two ever disagree.

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
$GDOC generate --md <paired md file> \
  --out "$ROOT/.gdoc/<slug>/out/v<n>.docx" \
  --baseline-root "$ROOT"
```

No `--name`. The tool names the version `<cover title> v<n>` itself, counting the
versions already recorded in the note. A name you write can only disagree with
that count.

The `<n>` in `--out` is a local file name and nothing else. Read the current
count with `$GDOC pair show --md <paired md file>` and add one. On a first
publish that command answers `paired: false`, so `<n>` is 1.

`--out` is chosen by intent:

| Case | `--out` |
|---|---|
| a new Drive version | `.gdoc/<slug>/out/v<n>.docx` |
| a document Nail asked for | `$PWD` |
| an explicit destination | that path |

This step generates a new version, so it uses the first row. The `<slug>` there
is the queue directory `pending.md` came from, or, on a first publish where no
queue exists, the source file's stem. Either way the run reports the `slug` it
used, and that report wins. The `.docx` is an upload intermediate and stays out of
sight.

`--baseline-root` does two things, and the next apply needs both. On a successful
upload the new document is exported into `.gdoc/<slug>/baseline.md`, replacing
the previous version's snapshot, reported as `baseline_path`. It also records the
new version in the note's `gdoc_versions`, reported as `version`.

The tool works out that `<slug>` itself, from the source file name, and reports
it as `slug`. Use the reported `slug` and `baseline_path` for every `.gdoc/`
path after the run, rather than the one you inferred for `--out`. They differ
when the queue was made with `gdoc capture --slug`, and then the baseline lands
outside the queue folder. If they differ, say so to Nail.

**Do not run `$GDOC pair add-version` after this.** The generate above already
recorded the version. `add-version` does not check for duplicates, so a second
call records the same document twice and the next version is numbered one too
high. The one time it is right is repair, when `pairing_error` says the write
failed. That case is below.

### Which folder

The document has to be created somewhere, and Nail is the one who decides where.

| What you have | What you do |
|---|---|
| a Drive folder URL in his message | pass it as `--folder-id "<the URL>"`. The CLI takes the id out of it |
| no URL, and `output_folder_id` in the config | leave `--folder-id` off. The config answers |
| neither | ask him for the folder URL, and wait |

```bash
$GDOC generate --md <paired md file> \
  --folder-id "https://drive.google.com/drive/folders/<id>" \
  --out "$ROOT/.gdoc/<slug>/out/v<n>.docx" \
  --baseline-root "$ROOT"
```

A folder URL on the command line is enough on its own: with one, no config file is
needed at all. Read the config with `cat ~/.config/gdoc-agent/config.json` if you
need to know whether a folder is already set.

Never guess a folder, and never reuse one from an older run in this session
without saying which you used. Publishing into the wrong folder puts a document in
front of the wrong people, and unpublishing it is not something you can do.

If he pastes a document URL by mistake, the CLI refuses it by name. Show him what
it said and ask for the folder.

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

### The five outcomes

Read the JSON. If the output is not JSON, something failed below the CLI: show
Nail exactly what came out, and stop. Do not guess what it meant.

Given JSON, `doc_id` decides whether a document exists. Nothing else does.

| Outcome | How you tell | What you do |
|---|---|---|
| Published | `doc_id` and `link` are set, `drift` and `reason` are null | Give Nail the link. Carry on. |
| Published, with a warning | `doc_id` is set, and one of `drift`, `reason`, `baseline_error`, `pairing_error` says something | Give him the link first, then say what did not get recorded. |
| Nothing created | `doc_id` is null and `reason` says why | Give him `docx_path` and the reason. Do not retry silently. |
| Missing title | exit 1, and `missing`, `suggested_title` and `suggested_from` are there | Stop and ask Nail. See below. |
| Refused for anything else | exit 1, and `error` is the only key | Show Nail the `error` as written, and stop. Do not touch the note. |

`drift` and `reason` are always in the JSON, and are null when there is nothing
to say. `baseline_error` and `pairing_error` appear only when that write failed.

The last two rows both exit 1 with an `error`, so read the keys, not the exit
code. Only the row with `missing` and `suggested_title` is about a title. The
most common of the others is a note with no YAML front matter block at all,
which is not a missing title and is not fixed by adding one. That `error`
message says what is wrong. Pass it on as it is written.

**Published, with a warning.** The document is real. Do not read a warning as a
failure and skip ahead, and do not bury the link under it. Give the link, then
say plainly what each key means:

| Key | What it means |
|---|---|
| `drift` | the page numbers in the contents list disagree with the document, as `{heading: [written, published]}`. Tell him to check the contents page before sharing it |
| `reason` | a note about a document that was still created, such as a measuring copy left in the Drive folder for him to delete |
| `baseline_error` | `.gdoc/<slug>/baseline.md` still describes the previous version, so the next apply cannot see direct edits |
| `pairing_error` | the version is missing from `gdoc_versions`, so the next version reuses this one's number. Repairable, see below |

`pairing_error` is the one case where `pair add-version` is the right command.
The write failed, so there is nothing to duplicate. Run it once, with the
`doc_id` the JSON just reported:

```bash
$GDOC pair add-version --md <paired md file> \
  --version-id <doc_id> --created <YYYY-MM-DD>
```

If that fails too, the note's front matter is the problem, not the version.
Tell Nail what it said and stop.

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
Then run Step 4 again, and finish it, including the commit below.

### Commit the note

`generate` writes `gdoc:` and `gdoc_versions` into the paired markdown itself,
and a `title:` line may have gone in just before the run. Both are unsaved work
until they are committed. Step 3 commits the text, this commits what the publish
wrote back, and together they keep every generated document matched to a commit.

Skip this when `doc_id` is null: nothing was written, so there is nothing to
commit. After a `pairing_error`, repair it first, then commit.

Inside a repository, and only there:

```bash
git -C "$ROOT" rev-parse --git-dir >/dev/null 2>&1
```

If that succeeds:

```bash
(cd "$ROOT" && git add <paired md file> && git commit -m "docs: record <title> v<version> in gdoc_versions")
```

`<version>` is the number the JSON reported. If it fails, say the front matter
was updated and nothing was committed. The note is saved on disk either way.

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
- Never run `pair add-version` after a generate that recorded the version itself.
  Repair is the exception: when `pairing_error` says the write failed, that
  command is the fix, run once.
- Never write a `gdoc_versions` entry by hand for a document that failed to upload.
- Never run `pair set` to fix an unpaired file. It clears `gdoc_versions` when
  the id differs, which drops the history. `generate` pairs the file itself.
- Never invent a title, and never approve the suggested one on Nail's behalf.
- Never report a commit that did not happen.
