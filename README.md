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
2. /gdoc-apply publishes it as a Google Doc   house template, cover, contents, page numbers
3. People read it and comment                 mark yours with ai: for the agent
4. /gdoc-review answers the comments           replies posted in the threads
5. /gdoc-apply publishes the next version      edits notes.md, publishes v2
```

Two skills, and `/gdoc-apply` does both publishes. The first time, there is
nothing queued and publishing is the whole job. Later it works through what the
review captured, then publishes again.

Steps 4 and 5 are separate sessions on purpose. A reply in a thread is cheap. A
rewrite of the whole document is not, so it is never done while you are reading
comments.

## What it does today

**Reads your comments.** It shows you two lists: the comments waiting for the
agent, and the ones it skipped with the reason visible. Comments it has already
answered land in the skipped list, so running it twice posts nothing twice.

**The marker decides.** A comment counts when it starts with `ai` plus a sign:

| Marker | Meaning |
|---|---|
| `ai:` | you decide, agent or queue |
| `ai?` | answer it in the thread |
| `ai!` | this is a document-wide change, queue it |
| `@ai` | the old form, still accepted |

Anything else is left alone. A comment starting "AI tools are changing" is not a
prompt.

**Answers in the thread.** Replies are plain text, posted into the comment thread
where you asked. The agent never resolves a comment. You resolve it, because
resolving means you accepted the answer.

**Queues the big ones.** Document-wide items are written to a queue file beside
your markdown. Nothing is dropped silently. An item stays queued until it is
applied or you remove it.

**Publishes.** It renders your markdown through the house .docx template and
uploads it to Drive as a new Google Doc. You get the cover page, the running head,
the revision table, numbered headings, and a contents list with real page numbers.

**Counts the pages properly.** Nothing in Python knows where a page break lands,
so Google does the counting. It uploads once with blank page numbers, exports that
copy as a PDF, reads which page each heading landed on, then uploads the version
you publish and trashes the measuring copy.

**Spots what you edited in the doc.** Every publish saves a snapshot of the
document exactly as it was uploaded. A later run compares that snapshot with a
fresh copy, so it can tell you what somebody changed by hand inside the Google Doc.

**Keeps the version history.** Your markdown file records which document it is
paired to, when it was last synced, and every version published from it.

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
git clone https://github.com/nhusnullin/gdoc.git
cd gdoc
./install.sh
```

If GitHub says the repository does not exist, it is private. Ask Nail for access.

`install.sh` creates a venv at `~/.config/gdoc-agent/venv`, installs the tool into
it, and links the two skills into `~/.claude/skills/` so Claude Code can find
them. It prints the commit you are running. Safe to re-run: every step checks the
current state first.

To update later, `git pull` is the whole update. Re-run `./install.sh` only after
dependencies change or a new skill is added.

### 2. Create the service account

This is the account the agent acts as. It is not you, and it holds no access to
anything until you share something with it.

In [console.cloud.google.com](https://console.cloud.google.com):

1. Create a project, or pick one.
2. **APIs and Services → Library →** enable **Google Drive API**.
3. **IAM and Admin → Service Accounts → Create service account.** Give it a name
   you will recognise in a comment thread, for example `doc-agent`. No project
   roles are needed: everything it can do comes from Drive sharing.
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

`template` is optional. It defaults to the bundled house style.

### 5. Share a document with it

On any document you want reviewed, **Share** it with the service account address
as **Commenter**. Nothing else.

That share is the whole access model. To let the agent work on a document you
share it, to stop it you unshare it. There is no list of documents anywhere, and
no run can widen its own reach.

### 6. Check it works

In Claude Code, from the folder that holds your markdown:

```
/gdoc-review <google doc url> --terminal-only
```

It reads the comments and prints what it would say, and posts nothing at all. If
it cannot see the document, the share in step 5 did not land on the right address.

## Using it

Run the skills from the folder that holds your markdown. That folder is also what
the agent searches to ground its answers, so where you stand decides what it
knows.

### Publish a document

```
/gdoc-apply notes.md
```

A note that has never been published has nothing queued, so it asks once and
publishes it. You get a link back.

### Review the comments

```
/gdoc-review <google doc url>
```

It shows you what it found and stops. Nothing is posted until you say so. Add
`--terminal-only` to print the replies in the terminal and post nothing, which is
the safe way to try anything new.

For each comment it either answers in the thread or queues the item and says so in
the thread, so anyone reading the document can see the change was noticed and is
coming.

### Apply what was queued, and publish again

```
/gdoc-apply notes.md          your file
/gdoc-apply <google doc url>  the document you were just reading
/gdoc-apply                   whatever is queued here, and it asks if there are several
```

All three find the same queue, so use whichever you have at hand. You never type a
path into `.gdoc/`.

It takes one item at a time, shows you the change it made to your markdown, and
waits. When the items are done it publishes a new version and gives you the link.

## What you get told after a publish

The skill reads the result and reports it in plain words. This is what those words
mean, so a warning is not mistaken for a failure.

One thing decides everything: **the link**. If you were given a link, the document
exists.

| What the skill says | What it means |
|---|---|
| a link, nothing else | Published. Open it. |
| a link, plus one of the warnings below | Published, and something did not get recorded. Open the document, then read the warning. |
| no link, and a reason | Nothing was created in Drive. The .docx is on disk, and the reason usually names the output folder. |
| it needs a title, with a suggestion | Nothing was published. See below. |
| a refusal | The message says what is wrong. Most often the note has no front matter block at all. |

The four warnings, and what to do about each:

- **Drift.** The page numbers in the contents list disagree with where the
  headings actually landed. It names each heading, the number written and the real
  one. Check the contents page before you share the document.
- **A note about the document.** It was still created. Usually a measuring copy
  was left in the output folder for you to delete.
- **The snapshot was not saved.** The document is published, but the local
  snapshot still describes the previous version. The next review cannot tell what
  people edited by hand until you publish again.
- **The version was not recorded.** It did not reach your markdown's front matter,
  so the next version would reuse this number. Ask the agent to repair it. It knows
  the one command that does it.

None of these means "run it again and hope". A document either exists or it does
not, and the link is the answer.

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

A document with no `title` is refused, because the cover page and the running head
would be blank. The tool suggests one from the first heading or the file name, and
the agent will usually propose a better one from reading the note. It must show
you both, and nothing is published until you pick one.

Once you pick, the title goes into the front matter and every later version reuses
it. You can also approve a title for one publish only, without changing the note.

`gdoc/templates/altery-group-policy-v1.0/example.md` shows every field the house
template can use, including the revision table.

## Where the files go

The tool works in one folder, the one you are standing in. Its own files go in a
`.gdoc/` folder beside the markdown they belong to. Nothing is written into this
repo.

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

You never need to open any of it. The folder is named after your markdown file, so
it survives every new version of the document.

## Limitations

Worth reading before you rely on it.

**It cannot edit the document.** By design. All it can do inside a Google Doc is
post replies in comment threads. Every real change goes to the markdown and comes
back as a new version.

**Replies are signed with the raw address.** A thread shows
`doc-agent@your-project.iam.gserviceaccount.com`, not a friendly name. Everyone
reading the document sees that. A nicer label needs a real Google Workspace user
for the agent, which is separate work.

**It cannot see who else has access.** Under Commenter, Google refuses to say. So
it cannot warn you that an outside collaborator is on the document. It says so
every run, and the decision to post is yours.

**A document you cannot share, it cannot read.** If you only hold Commenter on
somebody else's document, you cannot add the service account. You have to ask the
owner.

**Unmarked comments are ignored.** Only `ai:`, `ai?`, `ai!` and `@ai` count.
Handling ordinary comments is planned, not built.

**The author name is a label, not a gate.** A marked comment from anyone on the
document is acted on. Google returns no email address for comment authors and
display names are editable, so an identity check would be a guess. Pointing the
skill at a document is the trust decision. Every name shows up in the report, so
an unexpected one is visible.

**Nothing runs by itself.** No watcher, no polling, no schedule. You start every
run.

**pandoc is required.** Publishing needs it. Without it you get a clear message,
not a crash, but you get no document.

**Round trips lose formatting.** The copy fetched back from Google is not your
original file. Comparisons still work, because the snapshot went through the same
conversion and the distortion cancels out. Never treat a fetched copy as a
replacement for your source.

**Page numbers cost an upload.** Every publish creates two documents and trashes
the measuring one. If a run dies halfway, check the output folder.

**Git is optional, with one consequence.** Nothing refuses to run because git is
missing, and the skill says out loud when it skipped a commit. Outside a
repository it cannot tell an edit from a stale copy, so it refuses to overwrite an
existing snapshot rather than guess.

**One house template.** `altery-group-policy-v1.0` is bundled. A second one needs
code, because the cover and the tables are found by their placeholder text.

## What is planned

**gdoc live.** The next piece, designed and planned, not built. Today a review is
a batch: you run the skill, it sweeps the comments, it stops. Live turns it into a
session. You open it once on a document and keep reading. Every `ai:` comment you
write gets an answer in its thread within a few seconds, with a single progress
line that rewrites itself while the agent works.

**Ordinary comments.** Reading and answering comments that carry no marker.

**Fewer things to install.** Google can hand back markdown by itself, so the
pandoc dependency should shrink.

**A friendly reply name.** A real Workspace user for the agent, so threads stop
showing a service account address.

---

Working on the tool itself? Start with [PRINCIPLES.md](PRINCIPLES.md), then
[CLAUDE.md](CLAUDE.md).
