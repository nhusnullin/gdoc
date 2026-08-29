# gdoc

Review a Google Doc by leaving comments in it, and let Claude Code answer them
from your terminal.

You mark a comment with `ai:`. The agent reads it, answers small things right
there in the comment thread, and queues the ones that need a rewrite of the whole
document. Later you work through that queue: the agent edits the source markdown
file and publishes a new version of the Google Doc from it.

The markdown file is the source. The Google Doc is a rendering of it. Nothing in
the tool edits a document you point it at.

There are two credentials, and `auth_mode` in the config picks one.

**`oauth`** is the default. You approve it once in a browser and the agent acts as
you, so nothing has to be shared with anything first. It holds more than the tool
needs: the narrow `drive.file` scope does cover comments, but a file only enters
that scope when the app created it or the user picked it through Google's file
picker, and a terminal cannot show one. Pasting a URL means full Drive. `gdoc/guard.py` narrows it back down: every
request goes through it, and it carries a request only when the file addressed is
one gdoc was given or one gdoc created. Anything else is refused inside the
process, a read included. So the tool cannot see a document you did not point it
at, and cannot search your Drive at all.

**`service_account`** is the original. The agent has its own account, you share a
document with it as **Commenter**, and Google itself refuses every edit. No
browser and no token to refresh, at the cost of one sharing step per document.

You never edit the config to choose. `gdoc auth login` switches to `oauth`, and
`gdoc auth use service_account` switches back. An install that already works on a
service account keeps using it after an upgrade, because a config that never
mentioned `auth_mode` is read as unstated rather than as oauth. `gdoc auth status`
says which credential is in use and whether that came from the config or was
worked out from the files present.

The difference worth knowing: under `service_account` the tool *cannot* edit a
reviewed document, because Google will not let it. Under `oauth` it *does not*,
because no code in it does. The guard bounds which files are reachable. It does
not bound what happens inside one.

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

**Restyles a document it knows nothing about.** Point it at a Google Doc with no
markdown behind it and it pulls the document, puts it in the house template, and
publishes a new one. The open comment threads come across, each message carrying
the name of whoever wrote it. The original is untouched, and nothing is saved on
your side: a restyle is a throwaway copy, not a new note to look after.

## What you need before you start

- macOS or Linux, and a terminal.
- Python 3.11 or newer.
- [Claude Code](https://claude.com/claude-code), because the two skills run inside it.
- pandoc. `brew install pandoc`, or your package manager. It is used to read
  markdown, so publishing does not work without it.
- A Google altery.com account. Nothing to create in Google Cloud: gdoc ships the
  OAuth client it signs in with.

## Setup

### 1. Install the tool

```bash
git clone https://github.com/nhusnullin/gdoc.git
cd gdoc
./install.sh
```

If GitHub says the repository does not exist, it is private. Ask Nail for access.

`install.sh` creates a venv at `~/.config/gdoc-agent/venv`, installs the tool into
it, links `gdoc` into `~/.local/bin` so it is on your PATH, and links the two
skills into `~/.claude/skills/` so Claude Code can find them. It prints the commit
you are running. Safe to re-run: every step checks the current state first.

If it says `~/.local/bin` is not on your PATH, it prints the one line to add.

To update later, `git pull` is the whole update. Re-run `./install.sh` only after
dependencies change or a new skill is added.

### 2. Sign in

```bash
gdoc auth login
```

A browser opens, you approve, and that is the whole step. There is no OAuth client
to create and no config file to edit: gdoc ships its client, and the login writes
`auth_mode` for you.

You can skip this step entirely. Run `/gdoc-review <url>` and the skill notices
there is no token, asks whether to sign you in, and does it.

To check, or to change your mind later:

```bash
gdoc auth status               # which credential, and whether it works
gdoc auth logout               # delete the local token
gdoc auth use service_account  # switch credential, if a key is installed
```

The token lands in `~/.config/gdoc-agent/oauth-token.json`, mode `0600`, and never
leaves your machine. Revoke the grant at
[myaccount.google.com/permissions](https://myaccount.google.com/permissions).

### 3. Make a folder to publish into

New documents have to be created somewhere. Make a folder for them, **in a Shared
Drive, not in My Drive**.

The Shared Drive is worth the extra click. Files created there belong to the
Shared Drive, so your colleagues can open them without you sharing each one.

Then copy the folder URL out of the address bar. That is the whole thing you hand
over when you publish:

```
https://drive.google.com/drive/folders/1AbCdEfGhIjKlMnOpQrStUvWxYz
```

Nothing to configure. Paste that URL when the agent asks which folder, and it
takes the id out of it.

### 4. Nothing to share

Under `oauth` there is no sharing step. The agent reaches whatever you can reach,
and the guard keeps it to the document you named plus the folder you publish into.

To keep the agent off something, do not point it at it. There is no list of
documents anywhere, and no run can widen its own reach.

### 5. Check it works

In Claude Code, from the folder that holds your markdown:

```
/gdoc-review <google doc url>
```

It reads the comments and shows you what it found, then asks before posting
anything. If it cannot see the document at all, run `gdoc auth status` and check
which account you signed in as.

### Optional: bring your own OAuth client

Every user of the bundled client draws on the same Google rate limit. If that ever
bites, create a Desktop app OAuth client of your own and save its JSON to
`~/.config/gdoc-agent/oauth-client.json`. A file there wins over the bundled
client, and `gdoc auth status` reports which one is in use.

Security is not the reason to do this. A client shipped to many users is a public
client by definition, per RFC 8252 section 8.5, and `gh` and `gcloud` both ship
theirs the same way. Quota is the reason.


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

### Optional: use a service account instead

The agent can have its own identity instead, with Commenter as a wall Google
enforces. Create the account in **IAM and Admin → Service Accounts**, no project
roles needed, add a JSON key, and save it as `~/.config/gdoc-agent/sa-key.json`
with `chmod 600`. Share the publish folder with its address as **Content
manager**, and every document you want reviewed as **Commenter**, keeping the two
folders apart. Then:

```bash
gdoc auth use service_account
```

It writes the setting for you and warns if the key is not there yet. Going back is
`gdoc auth login`.

### For whoever maintains gdoc: the OAuth client

Done once, for everybody. `gdoc/oauth.py` holds `BUNDLED_CLIENT_ID` and
`BUNDLED_CLIENT_SECRET`. To create or replace them, in
[console.cloud.google.com](https://console.cloud.google.com):

1. Pick the project, and keep the **Google Drive API** enabled. It is the only
   API gdoc calls.
2. **OAuth consent screen**, User type **Internal**. Not optional. Internal is
   what exempts gdoc from OAuth verification, from the unverified-app screen and
   from the 100-user cap. gdoc needs the full Drive scope, which Google classes as
   restricted, so going External would mean a
   [CASA security assessment](https://developers.google.com/identity/protocols/oauth2/production-readiness/restricted-scope-verification)
   every 12 months. Internal also keeps refresh tokens from expiring after seven
   days.
3. **Credentials → Create credentials → OAuth client ID**, Application type
   **Desktop app**.
4. Paste the id and secret into the two constants in `gdoc/oauth.py`.

Internal means only altery.com accounts can sign in. That is the audience.

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

**It does not edit the document, but under `oauth` nothing stops it.** All it does
inside a Google Doc is post replies in comment threads. Every real change goes to
the markdown and comes back as a new version. Under `service_account` that is a
permission Google enforces. Under `oauth` the guard bounds which files the tool can
reach, not what it could do inside the one you named, so it is design discipline
instead. Pointing the skill at a document is the trust decision.

**Replies carry your name.** Under `oauth`, Drive reports you as the author of
every reply, so a thread does not show that an agent wrote it. gdoc signs each
reply with a `[gdoc]` line on its last line, which is also how it recognises its
own replies on a later run. Under `service_account` the thread shows the raw
`doc-agent@your-project.iam.gserviceaccount.com` address instead.

**It cannot always see who else has access.** Under `service_account`, Commenter
means Google refuses to say, so it cannot warn you that an outside collaborator is
on the document. It says so every run, and the decision to post is yours.

**A document you cannot share, it cannot read, under `service_account` only.** If
you hold Commenter on somebody else's document you cannot add the service account,
so you have to ask the owner. Under `oauth` any document you can open is readable.

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
lists, tables, and the words.

**Words you change in the document are carried back, once you ask.** `/gdoc-apply`
starts by reading what changed in the document since it was generated, in editing
mode and in suggesting mode both, and puts those changes into your markdown before
it publishes the next version. It reports what it applied and asks about anything
it could not place. Suggestions stay pending in the document: gdoc reads them, and
accepting them is yours. If you never run `/gdoc-apply`, the next publish still
overwrites the document with what the markdown says.

**Page numbers cost an upload.** Every publish creates two documents and trashes
the measuring one, because only Google can say which page a heading landed on. If
a run dies halfway, check the folder for a leftover. To be fixed, tracked as
[issue 23](https://github.com/nhusnullin/gdoc/issues/23).

**A shared folder is safe while filenames stay unique.** Your notes folder may sync
through Dropbox or Nextcloud. The tool's bookkeeping lives in `.gdoc/` beside your
markdown, so it syncs too, and it is keyed by the filename rather than by who you
are. You and a colleague working on differently named notes never collide. Two
notes with the same filename share one queue and one snapshot, and so do two people
reviewing the same document. Watch for sync conflict copies as well:
`notes (conflicted copy).md` carries the same document id as `notes.md`, and the
agent would pick up the copy. Tracked as
[issue 25](https://github.com/nhusnullin/gdoc/issues/25).

**Pictures come across, and drawings need one flag.** An ordinary embedded
picture survives a plain `gdoc export`: Drive hands it back inside the markdown
and the publish puts it in the new document. A **Google Drawing** does not: Drive
leaves those out of its markdown entirely, so pull the document with
`gdoc export --out note.md --media-dir note-media`, which takes the docx route
and writes the pictures beside the note. Either way, a picture that could not be
carried is named in the output rather than dropped quietly.

**One house template.** `altery-group-policy-v1.0` is bundled. A second one needs
code, because the cover and the tables are found by their placeholder text.

## The Go rewrite

A second implementation lives at `go/`: one static binary, no Python, no pandoc,
nothing to install beside it. It is being built a milestone at a time
(`docs/v2/PLAN.md`), and so far it does one job, the credential.

```bash
make build   # bin/gdoc, for this machine
make dist    # bin/gdoc-darwin-arm64, -darwin-amd64, -windows-amd64.exe
```

```bash
bin/gdoc auth status   # which token, where it is, whether it has expired
bin/gdoc auth login    # prints a sign-in link and waits for you to open it
```

`auth status` answers when you are signed out too: no token is a fact it
reports, not an error. `auth login` prints the link rather than opening a
browser for you, because the Go binary runs no other program at all.

Both print exactly one JSON object on stdout and nothing else. Prose and the
sign-in link go to stderr, so anything reading the output has one object to
parse and no filtering to do.

It reads the same `~/.config/gdoc-agent/oauth-token.json` the Python tool
writes, in the same format. Sign in once and you are signed in to both.

The `gdoc` on your PATH is still the Python tool this README describes. `bin/gdoc`
is the new one, and nothing installs it yet.

## What is planned

**gdoc live: answers while you read.** Today a review is one pass. You run it, it
answers the comments that were already there, and it stops. Live keeps it open
instead. You start it once on a document, then carry on reading. Write an `ai:`
comment and the answer appears in that thread a few seconds later, while you are
still on the paragraph that prompted it. You never leave the document, and you
never run anything again. Designed and planned, not built yet.

**Ordinary comments.** Reading and answering comments that carry no marker.

**Fewer things to install.** Google can hand back markdown by itself, so the
pandoc dependency should shrink.

**A friendly reply name.** A real Workspace user for the agent, so `service_account`
threads stop showing a raw address.

---

Working on the tool itself? Start with [PRINCIPLES.md](PRINCIPLES.md), then
[CLAUDE.md](CLAUDE.md).
