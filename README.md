# gdoc

Review a Google Doc by leaving comments in it, and let Claude Code answer them
from your terminal.

You mark a comment with `ai:`. The agent reads it, answers small things right
there in the comment thread, and queues the ones that need a rewrite of the whole
document. Later you work through that queue: the agent edits the source markdown
file and publishes a new version of the Google Doc from it.

The markdown file is the source. The Google Doc is a rendering of it. The tool
never edits a document you point it at, because the credential holds **Commenter**
access only. That is not a rule the agent follows, it is a permission Google
enforces.

## The loop

```
1. You write the document as markdown        notes.md, in your notes folder
2. gdoc publishes it as a Google Doc         house template, cover, contents, page numbers
3. People read it and comment                mark yours with ai: for the agent
4. /gdoc-review answers the comments          replies posted in the threads
5. /gdoc-apply does the bigger rewrites       edits notes.md, publishes v2
```

Steps 4 and 5 are two sessions on purpose. A reply in a thread is cheap. A
rewrite of the whole document is not, so it is never done while you are reading
comments.

## What works today

**Reading comments.** `gdoc read <url>` returns two lists: the comments waiting
for the agent, and the ones it skipped with the reason visible. Comments already
answered by the agent land in the skipped list, so running it twice is safe and
posts nothing twice.

**The marker.** A comment counts when it starts with `ai` plus a sign:

| Marker | Meaning |
|---|---|
| `ai:` | you decide, agent or queue |
| `ai?` | answer it in the thread |
| `ai!` | this is a document-wide change, queue it |
| `@ai` | the old form, still accepted |

Anything else is left alone. A comment starting "AI tools are changing" is not a
prompt.

**Answering in threads.** Replies are plain text, posted into the comment thread.
The agent never resolves a comment. You resolve it, because resolving means you
accepted the answer.

**Queueing the big ones.** Document-wide items go into `.gdoc/<name>/pending.md`
beside your markdown file. Nothing is dropped silently. An item sits in that file
until it is applied or you remove it.

**Publishing.** `gdoc generate` renders your markdown through the house .docx
template and uploads it to Drive as a new Google Doc. You get the cover page, the
running head, the revision table, numbered headings, and a contents list with
real page numbers.

**Page numbers.** Nothing in Python knows where a page break lands, so Google does
the counting. Generate uploads once with blank page numbers, exports that copy as
PDF, reads which page each heading landed on, then uploads the version that gets
published and trashes the measuring copy.

**Comparing.** `gdoc export <url>` fetches a document back as markdown. Every
publish also saves `baseline.md`, the document exactly as it was uploaded, so a
later run can diff it against a fresh export and see what somebody edited by hand
inside the Google Doc.

**Version history.** The markdown file keeps its own lineage in the front matter:
which document it is paired to, when it was last synced, and every version
published from it.

## What you need before you start

- macOS or Linux, and a terminal.
- Python 3.11 or newer.
- [Claude Code](https://claude.com/claude-code), because the two skills run inside it.
- pandoc. `brew install pandoc`, or your package manager. It is used to read
  markdown, so publishing does not work without it.
- A Google account, and permission to create a service account in Google Cloud.

## Setup

### 1. Install the tool

```bash
git clone <this repo>
cd gdoc
./install.sh
```

That creates a venv at `~/.config/gdoc-agent/venv`, installs the `gdoc` command
into it, and links the two skills into `~/.claude/skills/`. It prints the commit
you are running. Safe to re-run: every step checks the current state first.

To update later, `git pull` is the whole update. The package is an editable
install and the skills are symlinks, so both follow the working tree. Re-run
`./install.sh` only after dependencies change or a new skill is added.

### 2. Create the service account

In [console.cloud.google.com](https://console.cloud.google.com):

1. Create a project, or pick one.
2. **APIs and Services → Library →** enable **Google Drive API**.
3. **IAM and Admin → Service Accounts → Create service account.** Give it a name
   you will recognise in a comment thread, for example `doc-agent`. No project
   roles are needed: everything it can do comes from Drive sharing, not from IAM.
4. Open the account, **Keys → Add key → Create new key → JSON.** A file
   downloads.

Then put the key where the tool looks for it, and lock it down:

```bash
mkdir -p ~/.config/gdoc-agent
mv ~/Downloads/<the-downloaded-key>.json ~/.config/gdoc-agent/sa-key.json
chmod 600 ~/.config/gdoc-agent/sa-key.json
```

Copy the account's address from the console. It looks like
`doc-agent@your-project.iam.gserviceaccount.com`. You need it twice below.

Never commit this key. It belongs in `~/.config/gdoc-agent/`, never in a repo.

### 3. Give it an output folder

New versions have to be created somewhere.

Make a folder in a **Shared Drive**, then share that folder with the service
account address as **Content manager**.

A Shared Drive matters. Files created there are owned by the Shared Drive, so you
find them in your normal Drive and they do not disappear into a robot account's
storage. In My Drive they would be owned by the service account instead.

Open the folder and copy the id from the URL, the part after `/folders/`.

### 4. Write the config

`~/.config/gdoc-agent/config.json`:

```json
{
  "output_folder_id": "1AbCdEfGhIjKlMnOpQrStUvWxYz",
  "template": "altery-group-policy-v1.0"
}
```

`template` is optional. It defaults to the bundled house style. Use
`"none"` to publish a plain document with no cover and no template.

### 5. Share a document with it

On any document you want reviewed, **Share** it with the service account address
as **Commenter**. Nothing else. That share is the whole access model: to let the
agent work on a document you share it, to stop it you unshare it. There is no
list of document ids anywhere.

### 6. Check it works

```bash
~/.config/gdoc-agent/venv/bin/gdoc read <google doc url>
```

You should get JSON back with the document name and its comments. If you get a
404, the document is not shared with the service account.

## Using it

Run these from the folder that holds your markdown, because that folder is also
the corpus the agent searches to ground its answers:

```
/gdoc-review <google doc url>
/gdoc-apply notes.md
```

`/gdoc-review` shows you what it found and stops. Nothing is posted until you say
so. Add `--terminal-only` to print the replies here and post nothing at all,
which is the safe way to try it the first time.

`/gdoc-apply` works through the queued items with you, edits the markdown, shows
you the diff, and publishes a new version. Pass your own markdown file, or the
document link, or nothing at all:

```
/gdoc-apply notes.md          the file you write in
/gdoc-apply <google doc url>  the document you were just reading
/gdoc-apply                   whatever is queued here, and it asks if there are several
```

All three find the same queue. You never type a path into `.gdoc/`.

## The commands underneath

The skills call these. You can run them directly. Every one prints a single JSON
object, except `export`, which prints markdown so you can pipe it into `diff`.

```bash
gdoc read <url>                                  # comments, split into to-do and skipped
gdoc reply <doc_id> <comment_id> --body-file reply.txt
gdoc capture <doc_id> <comment_id>               # queue a document-wide item
gdoc export <url>                                # the document as markdown, to stdout
gdoc export <url> --out fetched.md
gdoc generate --md notes.md --out out.docx --baseline-root .
gdoc generate --md notes.md --template none --out out.docx
gdoc pair show --md notes.md                     # which document is this paired to
gdoc pair find --doc-id <doc_id>                 # which markdown is this document paired to
```

`--name` on generate is optional. Without it the new version is called
`<cover title> v<n>`, where n is the version count plus one.

## What your markdown needs

`title` in the front matter, and nothing else is required:

```yaml
---
title: Third Party and Outsourcing
doc_type: Policy
version: "2.0"
classification: Internal
---
```

A document with no `title` is refused, because the cover page and the running
head would be blank. The refusal suggests one, taken from the first heading or the
file name, and waits for you to approve it. You then add it to the note, or pass
`--title "..."` to publish once without editing the note.

`gdoc/templates/altery-group-policy-v1.0/example.md` shows every field the house
template can use, including the revision table.

## Where files go

The tool works in one directory, `$PWD` by default. Its own files go in `.gdoc/`,
beside the markdown they belong to. Nothing is ever written into this repo.

```
your notes folder/
  2026-08-13-topic.md              your source, hand written
  2026-08-13-topic.docx            generated
  .gdoc/
    2026-08-13-topic/
      pending.md                   queued items, waiting to be applied
      baseline.md                  the document as it was last published
      out/v2.docx                  upload intermediate
```

The directory is named after the markdown file, not the document. Every iteration
raises a new Google Doc with a new id and usually a new title, so either of those
would split the queue in two. The file survives.

## Limitations

Worth reading before you rely on it.

**It cannot edit the document.** By design. All it can do inside a Google Doc is
post replies in comment threads. Every real change goes to the markdown and comes
back as a new version.

**Replies are signed with the raw address.** A thread shows
`doc-agent@your-project.iam.gserviceaccount.com`, not a friendly name. Anyone
reading the document sees that. A nicer label needs a real Google Workspace user
for the agent, which is a separate piece of work.

**It cannot see who else has access.** Under Commenter, Google refuses the
permissions call. So the agent cannot warn you that an outside collaborator is on
the document. It says so every run, and the decision to post is yours.

**A document you cannot share, it cannot read.** If you only hold Commenter on
somebody else's document, you cannot add the service account. You have to ask the
owner.

**Unmarked comments are ignored.** Only `ai:`, `ai?`, `ai!` and `@ai` count.
Handling ordinary comments is planned, not built.

**The author name is a label, not a gate.** A marked comment from anyone on the
document is acted on. Drive returns no email address for comment authors and
display names are editable, so an identity check would be a guess. Pointing the
skill at a document is the trust decision. Every name still shows up in the
report, so an unexpected one is visible.

**Nothing runs by itself.** No watcher, no polling, no schedule. You start every
run.

**pandoc is required.** Publishing and the export fallback both need it. Without
it you get a clear message, not a crash, but you get no document.

**Round trips lose formatting.** The markdown you get back from `export` is not
your original file. Comparisons work because `baseline.md` went through the same
conversion, so the distortion cancels. Do not treat an export as a replacement
for your source.

**Page numbers cost an upload.** Generate creates two documents and trashes the
measuring one. If a run dies halfway, check the output folder.

**No git needed, and one consequence.** The tool never refuses to run because git
is missing, and `/gdoc-apply` says out loud when it skipped a commit. Outside a
repository there is no way to tell an edit from a stale copy, so an existing
`baseline.md` is never overwritten unless `--force` says so.

**One house template.** `altery-group-policy-v1.0` is bundled. A second template
needs code, because the cover and the tables are found by their placeholder text.

## What is planned

**gdoc live.** The next piece, specced and planned, not built. Today a review is a
batch: you run the skill, it sweeps the comments, it stops. Live turns it into a
session. You open `/gdoc-live <url>` and keep reading. Every `ai:` comment you
write gets an answer in its thread within a few seconds, with a single progress
line that rewrites itself while the agent works. The design is in
`docs/superpowers/specs/2026-08-17-gdoc-live-realtime-design.md`, aimed at the
press release beside it.

**Ordinary comments.** Reading and answering comments that carry no marker.

**Dropping pandoc.** Drive can export markdown natively, so the export fallback
looks removable. The markdown parser behind publishing is the harder half and
needs its own design.

**A friendly reply name.** A real Workspace user for the agent, so threads stop
showing a service account address.

## For maintainers

- [PRINCIPLES.md](PRINCIPLES.md) is three durable constraints and the dated
  decisions under them. Read it before proposing a design.
- [CLAUDE.md](CLAUDE.md) is what lives where, and what never to do.
- `docs/superpowers/` holds the specs and the implementation plans.

```bash
~/.config/gdoc-agent/venv/bin/pytest
```

Tests that call Drive skip themselves without credentials. One test creates real
documents and is opt-in behind `GDOC_LIVE_PUBLISH_TEST=1`.

Editing a `SKILL.md` in `skills/` is live in every Claude Code session
immediately, before you commit it, because the skills are symlinked.
`./install.sh` prints `+ uncommitted changes` so you can tell what you are
actually running.
