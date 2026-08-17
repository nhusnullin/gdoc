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

### 3. Make a folder to publish into

New documents have to be created somewhere. Make a folder for them, **in a Shared
Drive, not in My Drive**, and share it with the service account address as
**Content manager**.

The Shared Drive is worth the extra click. Files created there belong to the
Shared Drive, so your colleagues can open them and they do not sit inside a robot
account nobody logs into. In My Drive they would be owned by the service account
instead.

Then copy the folder URL out of the address bar. That is the whole thing you hand
over when you publish:

```
https://drive.google.com/drive/folders/1AbCdEfGhIjKlMnOpQrStUvWxYz
```

Nothing to configure. Paste that URL when the agent asks which folder, and it
takes the id out of it.

### 4. Share the documents you review

The agent sees only what is shared with it. **Share** as **Commenter**, and
nothing more.

Keep those documents in a Shared Drive too. You can then share the folder once,
as Commenter, and every document in it is covered. Sharing one document at a time
works the same way, it is just more clicks.

Keep this folder separate from the publish folder in step 3. The publish folder
has to allow writes. Anywhere else, Commenter is a wall Google enforces, and that
is the property the whole tool leans on.

To stop the agent working on something, unshare it. There is no list of documents
anywhere, and no run can widen its own reach.

### 5. Check it works

In Claude Code, from the folder that holds your markdown:

```
/gdoc-review <google doc url>
```

It reads the comments and shows you what it found, then asks before posting
anything. If it cannot see the document at all, the share in step 4 did not land
on the right address.

### Optional: stop it asking for the folder

If you publish into the same folder every time, name it once in
`~/.config/gdoc-agent/config.json` and the agent stops asking:

```json
{
  "output_folder_id": "1AbCdEfGhIjKlMnOpQrStUvWxYz",
  "template": "altery-group-policy-v1.0"
}
```

Both keys are optional. `template` defaults to the bundled house style. The id is
the part of the folder URL after `/folders/`.

## Using it

Run the skills from the folder that holds your markdown. That folder is also what
the agent searches to ground its answers, so where you stand decides what it
knows.

### Publish a document

```
/gdoc-apply notes.md <folder url>
```

The folder URL is the one from step 3, copied out of the address bar. Give it in
the same message, or wait to be asked, or set it once in the config and never type
it again. All three work.

A note that has never been published has nothing queued, so publishing it is the
whole job. It asks once, then gives you the link.

Your note needs a `title` in its front matter, because the cover page is built
from it. If there is none, the agent proposes one and waits for you to pick.
`gdoc/templates/altery-group-policy-v1.0/example.md` shows every other field the
house template can use, including the revision table. All of them are optional.

### Review the comments

```
/gdoc-review <google doc url>
```

It shows you what it found and stops. Nothing is posted until you say so.

For each comment it either answers in the thread or queues the item and says so in
the thread, so anyone reading the document can see the change was noticed and is
coming.

### Apply what was queued, and publish again

```
/gdoc-apply notes.md          your file
/gdoc-apply <google doc url>  the document you were just reading
/gdoc-apply                   whatever is queued here, and it asks if there are several
```

All three find the same queue, so use whichever you have at hand.

It takes one item at a time, shows you the change it made to your markdown, and
waits. When the items are done it publishes a new version and gives you the link.

Its own bookkeeping lives in a `.gdoc/` folder beside your markdown. You never
need to open it, or name it.

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

## Limitations

Worth reading before you rely on it.

**It cannot edit the document.** By design. All it can do inside a Google Doc is
post replies in comment threads. Every real change goes to the markdown and comes
back as a new version. The one exception is the folder it publishes into, where it
has to be able to create files. Documents you share for review are behind the
wall. Documents it created itself are not, so keep the two folders apart.

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

**The template decides the look, so formatting done in the document is lost.**
Every publish renders your markdown through the house template again. Fonts,
colours, spacing and manual page breaks that somebody set inside the Google Doc do
not survive into the next version. What survives is text and structure: headings,
lists, tables, and the words. If a change matters, put it in the markdown.

**Page numbers cost an upload.** Every publish creates two documents and trashes
the measuring one, because only Google can say which page a heading landed on. If
a run dies halfway, check the folder for a leftover. To be fixed, tracked as
[issue 23](https://github.com/nhusnullin/gdoc/issues/23).

**One house template.** `altery-group-policy-v1.0` is bundled. A second one needs
code, because the cover and the tables are found by their placeholder text.

## What is planned

**gdoc live: answers while you read.** Today a review is one pass. You run it, it
answers the comments that were already there, and it stops. Live keeps it open
instead. You start it once on a document, then carry on reading. Write an `ai:`
comment and the answer appears in that thread a few seconds later, while you are
still on the paragraph that prompted it. You never leave the document, and you
never run anything again. Designed and planned, not built yet.

**Sign in as yourself.** Today the agent has its own account, and setup is mostly
about creating it and sharing things with it. With normal Google sign-in you would
approve it once in a browser and skip steps 2 to 4 of the setup. Replies would
carry your name instead of a robot address, and you would not have to share
anything, because the agent would see what you already see. The trade is real: the
Commenter wall in the limitations comes from that separate account, so signing in
as yourself replaces a permission Google enforces with a rule the tool follows.
Tracked as [issue 10](https://github.com/nhusnullin/gdoc/issues/10).

**Ordinary comments.** Reading and answering comments that carry no marker.

**Fewer things to install.** Google can hand back markdown by itself, so the
pandoc dependency should shrink.

**A friendly reply name.** A real Workspace user for the agent, so threads stop
showing a service account address.

---

Working on the tool itself? Start with [PRINCIPLES.md](PRINCIPLES.md), then
[CLAUDE.md](CLAUDE.md).
