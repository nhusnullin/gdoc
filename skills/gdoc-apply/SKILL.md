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

Nothing here runs git. Committing is Nail's job, and this skill does not do it
or check whether it could. What it does instead is name every file it changed,
so he can commit them himself if this root is a repository.

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

Do Steps 1 to 4: pull the direct edits, work through the items, edit the
markdown, show the diffs, say what was saved. Then stop. Do not run Step 5: `$GDOC generate`
always tries to upload, so there is no local-only way to produce the document.
Tell Nail the markdown is saved and that re-running without the flag will
generate the document.

A restyle is refused under this flag, not adapted. It exists to publish, and
`$GDOC restyle` uploads before it has anything to show. Say that, and stop
before the confirmation. `--dry-run` is the one part that is safe to run, and it
answers what the restyle would do without creating anything.

## Step 1: Find the queue

Nail passes whatever he already has: his markdown file, the document link, or
nothing. Never ask him for a path inside `.gdoc/`. He should not have to know a
slug to work on his own document.

| He passes | You do |
|---|---|
| nothing | list the queues under `$ROOT` |
| his markdown, `notes.md` | find the queue whose `Source:` line names it |
| the document URL | `$GDOC pair find --doc-id <doc_id>` for the source, then as above. Nothing paired means a restyle, below |
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

Then skip Steps 2, 3 and 4, because there is no baseline to compare against,
no items to work through and no edits to save, and go to Step 5. `generate` writes the first `gdoc:` and the first
`gdoc_versions` entry itself, so nothing has to be paired first.

Never run `pair set` to create that first pairing. Step 5 does it.

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
markdown for it here. That is the restyle below, not a refusal.

If he gave you a markdown file that exists, that same empty answer means the first
publish above, not a refusal. The difference is whether a source file is in his
hands, so check which he passed before you do either.

`baseline.md` in the queue folder is the document as it was generated. It is read
only for comparison, and it is never the file you edit. It normally sits beside
`pending.md`, but Step 5 reports `baseline_path`, and that is the truth if the
two ever disagree.

### The restyle

A document link, no queue, and nothing under this root paired to it. gdoc knows
nothing about it. What Nail wants is the same document in the house template, so
pull it, publish a new one, and carry the open comment threads across.

One command does all of it, and it is the whole job:

```bash
$GDOC restyle --doc "<the document URL>"
```

Ask first, and ask with the answer already on screen. `--dry-run` creates
nothing and gives you the document's name, the folder id and the thread counts:

```bash
$GDOC restyle --doc "<the document URL>" --dry-run
```

Then one confirmation, in this shape:

```
"MC incentive routes" is not paired to anything under this root.
Restyle it: a new document in the house template, in folder 0AFolderId
  (from your config).
  7 comment threads, 4 open. The 4 open ones come across, authored by you,
  with the original names in the text. The 3 resolved ones do not.
  This is a conversion, not a copy: tables and layout may come across differently.
  Nothing is written under this root. The original is untouched.
Go ahead? y
```

That is one turn, not ceremony. It creates a document other people can read, in
a folder, and there is no undo. The line that earns its place is the folder: a
restyled copy of somebody else's document in the wrong folder is the failure
worth being slow about. `--folder-id "<folder URL>"` overrides the config, the
same as everywhere else.

Then skip Steps 2, 3, 4 and 5. There are no items, no markdown to edit, no queue
to clear, and `restyle` has already published. Read its JSON:

| Key | What you say |
|---|---|
| `doc_id`, `link` | the document exists. Give him the link first |
| `comments_copied`, `comments_skipped_resolved` | how many threads came across, and how many resolved ones did not |
| `comment_errors` | threads that were not copied, each naming why. The document is still fine |
| `image_warnings` | pictures that could not be carried. Name them, he may want to add them by hand |
| `drift` | the contents list disagrees with the document. Tell him to check it before sharing |
| `reason` | why nothing was created. Read it whenever `doc_id` is null: a refused publish still exits 0, so the exit code will not tell you |
| `workdir` | only set when nothing was created. The pulled markdown is in there, and it is all the run produced |

Say four things afterwards, every time:

- No comment kept its anchor. The copies sit unattached at the top.
- Every copy is authored by whoever gdoc is signed in as. That is why the
  original names are in the text.
- The new document carries the same name as the original, so Drive now holds two
  documents with that name. Renaming it is his to do.
- Nothing under this root changed, so there is nothing to commit.

Use `--no-comments` when he wants the clean document without the discussion, and
`--template none` when he wants a plain one.

A restyle is a throwaway. It records no pairing, no version and no baseline, and
running it twice makes two unrelated documents. Never run `pair set` afterwards
to tie one to the original: they are not the same document, and the original is
not gdoc's to publish over.


## Step 2: Pull what Nail changed in the document

He reviews in the browser, and not only by leaving comments. He renames a
heading, cuts a sentence, moves a section, or marks the same changes in
suggesting mode. Those edits are his intention exactly as a comment is, and the
version generated in Step 5 is built from the markdown, so anything not carried
across here is gone.

```bash
$GDOC edits <doc url> --repo-root "$ROOT"
```

It writes nothing. It reports two lists and you decide what they mean.

- `hunks` are the differences between the document as it is now and
  `baseline.md`, the snapshot taken when it was generated. Both sides come out
  of the same export, so the export's own distortion cancels.
- `suggestions` are pending suggestions, read through the Docs API. The markdown
  export renders a suggested document as though nothing had been suggested, so
  without this a document reviewed entirely in suggesting mode looks untouched.

### Read `baseline` first

| Value | What it means | What you do |
|---|---|---|
| `ok` | the snapshot is this document's | use the hunks |
| `unverified` | published before the tool recorded which document a snapshot came from | show them, say they are unverified, and read the warning below before applying any |
| `missing` | never generated by gdoc, or the snapshot was lost | say so, use the suggestions only |
| `stale` | the snapshot belongs to an earlier version | say so, use the suggestions only |

**An `unverified` baseline may belong to a different version.** That is what
`unverified` means: nothing recorded which document the snapshot came from. Seen
live on 2026-08-19, on a real note: the baseline held v0.3 and the document was
v0.2, so the diff read as Nail deleting a whole section he had in fact added.
Applying those hunks would have deleted it from the source as well.

Read the hunks for that signature before applying any of them:

- a version number, date or heading in `before` that is *later* than the one in
  `after`;
- whole sections deleted, with nothing suggesting he cut them;
- deletes that restore an earlier state rather than change the current one.

Any of those means the baseline is newer than the document, and the hunks are
backwards. Say so, apply none of them, and use the suggestions only. Suggestions
are unaffected either way: they come out of the document itself, so a wrong
baseline cannot corrupt them.

`missing` and `stale` are not failures and they do not stop the items. They do
stop the publish. No hunk was computed under either, so an edit Nail made in
editing mode cannot be seen at all, and Step 5 would build the new version
without it. Say which case it is, work through the items, then ask before Step 5:

```
The baseline is stale, so I could not see editing-mode edits on this run.
If you changed the document outside suggesting mode, those changes are not
in what I applied. Publish anyway? y
```

`suggestions_error` naming `documents.readonly` means the token predates this
feature. Say that suggestions were not read, and **stop before Step 5**. Do not
publish a version built from markdown that half the review is missing from: a
suggestion the export cannot see is invisible everywhere else in the run too, so
nothing later would catch it.

```
Suggestions were not read: this token predates the Docs scope.
Run gdoc auth login once, then I will re-run gdoc edits before publishing.
```

Then run `$GDOC edits` again and read `counts.suggestions` before you go on. The
one thing that overrules either stop is Nail saying publish anyway, and then the
report in Step 6 says what was not read.

### Apply them

Into the **paired source markdown**, never into `baseline.md`. The source is
hand-written and not export-shaped, so a hunk is located by its heading and the
text around it, never by line number.

Judge, do not interrogate. Apply what is clear and report it afterwards in one
summary: twenty wording fixes are one line, not twenty questions. Stop and ask
only where the right answer is genuinely unclear:

- a hunk you cannot place in the source with confidence;
- a delete and an insert that may be one section moved, rather than two edits;
- a change that touches the same paragraph as a queued item in `pending.md`;
- anything that reads as a decision rather than a correction.

Then say what you did:

```
From the document: 3 edits applied, 1 to ask about.
  applied   heading "2. Routes" renamed to "2. Routes to membership"
  applied   sentence cut under "1. Key terms"
  applied   suggested insertion under "3. Timing"
  ask       a paragraph moved from "1. Key terms" to "3. Timing", or rewritten?
```

A suggestion is applied to the markdown and stays pending in the document. gdoc
does not accept it, and neither may you: the credential is read only there, and
accepting is Nail's to do. Say that out loud when you applied one.

## Step 3: Work through the items in order

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

## Step 4: Say what was saved

The edits are on disk as soon as they are made. Name the file:

```
Saved <paired md file>. Not committed: committing is yours.
```

That is the whole step. Do not run git, and do not check whether this root is a
repository. Nail commits when he wants to, and a run that leaves the working
tree dirty is the expected outcome, not a fault.

## Step 5: Generate the new version

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
Then run Step 5 again, and finish it.

### What the publish wrote back

`generate` writes `gdoc:` and `gdoc_versions` into the paired markdown itself,
and a `title:` line may have gone in just before the run. Say so, with the
version number the JSON reported:

```
<paired md file> now records <title> v<version>.
```

Nothing was written when `doc_id` is null, so say nothing then. After a
`pairing_error`, repair it first and then say what the repair wrote.

## Step 6: Clear the items

Delete the applied items from `pending.md`. If all items are done, delete the
file. Say what you removed either way.

Then close the run by listing every file that changed on disk, so Nail can
commit them if he wants to:

```
Changed, not committed:
  <paired md file>       edits from items 1 and 2, and the v3 record
  .gdoc/<slug>/pending.md  items 1 and 2 removed
```

## Never

- Never edit the original Google Doc.
- Never apply a hunk from an `unverified` baseline that reads backwards. A
  snapshot of a different version turns his additions into deletions.
- Never publish while `suggestions_error` is set, unless Nail says publish
  anyway after being told. Suggestions are the half of his review that
  nothing else in the run can see.
- Never accept or reject a suggestion in the document. Read it, apply it to the
  markdown, and leave it pending. Accepting is Nail's.
- Never edit `baseline.md`.
- Never run `pair add-version` after a generate that recorded the version itself.
  Repair is the exception: when `pairing_error` says the write failed, that
  command is the fix, run once.
- Never write a `gdoc_versions` entry by hand for a document that failed to upload.
- Never run `pair set` to fix an unpaired file. It clears `gdoc_versions` when
  the id differs, which drops the history. `generate` pairs the file itself.
- Never pair, version or baseline a restyled document. It is a throwaway copy,
  and the original is not gdoc's to publish over.
- Never invent a title, and never approve the suggested one on Nail's behalf.
- Never commit, and never check whether this root is a repository. That is
  Nail's job, and the skill's job is to name what it changed.
