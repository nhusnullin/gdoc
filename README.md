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
where you asked, and each one opens with 🤖 so a later run can tell them from
everybody else's. The agent never resolves a comment. You resolve it, because
resolving means you accepted the answer.

**Proposes changes as suggestions.** When the answer is a change to the
document's own words, it writes that as a native Google suggestion with a comment
saying why, and proves it landed as a suggestion rather than an edit before it
tells you so. It can take its own suggestion back. It never edits the document.

**Queues the big ones.** Document-wide items are written to a queue file beside
your markdown. Nothing is dropped silently. An item stays queued until it is
applied or you remove it. The review skill fills that queue no longer: it was
rewritten for the Go binary, and it now carries an `ai!` out in your notes there
and then, and receipts it in the thread. `/gdoc-apply` still reads the queue, so
anything already in one is still applied, and publishing is unchanged.

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

`install.sh` creates a venv at `~/.config/gdoc-agent/venv`, installs the Python
tool into it, builds the Go binary, links `gdoc` in `~/.local/bin` to the Go
binary so it is on your PATH, and links the skills into `~/.claude/skills/` so
Claude Code can find them. It prints the commit you are running. Safe to re-run:
every step checks the current state first.

`gdoc` on your PATH is the Go binary. The Python tool this README mostly
describes is `~/.config/gdoc-agent/venv/bin/gdoc`, which is the full path its own
skills call, so both keep working. "The Go rewrite" below says what the Go
binary does.

If it says `~/.local/bin` is not on your PATH, it prints the one line to add.

To update later, `git pull` is the whole update. Re-run `./install.sh` only after
dependencies change or a new skill is added.

### 2. Sign in

```bash
gdoc auth login
```

`gdoc` is the Go binary, so this prints a sign-in link rather than opening a
browser for you. Open it, approve, and it saves the token. There is no OAuth
client to create and no config file to edit: gdoc ships its client.

You can skip this step entirely. Run `/gdoc-review <url>` and the skill notices
there is no token, asks whether to sign you in, and does it.

To check:

```bash
gdoc auth status   # whether a token is there, whether it expired, what it is missing
```

Both tools read the same token file, so one login covers both in most cases. The
one exception: the Go login asks for the Docs read/write scope, because writing
suggestions needs it, and the Python tool checks that what *it* asked for is in
the file. So after a Go login, `gdoc edits` on the Python side asks you to sign
in through it once more. Its own commands are the ones that switch credential:

```bash
~/.config/gdoc-agent/venv/bin/gdoc auth status               # which credential, and whether it works
~/.config/gdoc-agent/venv/bin/gdoc auth logout               # delete the local token
~/.config/gdoc-agent/venv/bin/gdoc auth use service_account  # switch credential, if a key is installed
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

It reads the comments, tells you what it found, and answers the marked ones. If
it cannot see the document at all, run `gdoc auth status` and check which account
you signed in as.

### Optional: bring your own OAuth client

Every user of the bundled client draws on the same Google rate limit. If that ever
bites, create a Desktop app OAuth client of your own and save its JSON to
`~/.config/gdoc-agent/oauth-client.json`.

The Python tool reads that file, and it wins over the bundled client there:

```bash
~/.config/gdoc-agent/venv/bin/gdoc auth login
~/.config/gdoc-agent/venv/bin/gdoc auth status   # says which client is in use
```

The Go binary does not read it yet. `gdoc auth login` always signs in with the
bundled client, and `gdoc auth status` reports `client_source: bundled` plus
`client_file_ignored: true` when the file is there, rather than claiming an
override that is not wired up. What it does carry over is the refresh: a token
minted through the Python tool with your own client keeps refreshing under that
client, whichever binary makes the call. So a login through the Python tool moves
your quota; a login through the Go binary puts it back on the shared client.

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

This skill runs on the Go binary now. It lists what it found, then acts: a
marker is your instruction, so it does not ask again. `ai?` is answered in its
thread, `ai!` is carried out in your notes and receipted in the thread, and `ai:`
leaves the choice to it and it says which one it took. An unmarked comment is
never acted on.

When the answer is a change to the document's own words, it proposes that as a
Google suggestion with a comment saying why. It never edits the document, and it
can withdraw its own suggestion, which is why proposing needs no confirmation.
Say "dry run" and it does every step and writes nothing.

Say "live" and the session stays open on that document:

```
/gdoc-review <google doc url> live
```

It does the pass above first, then keeps watching that one document. Write a
marked comment in the browser and the answer appears in its thread a few seconds
later, while you are still on the paragraph that prompted it. A quiet document
costs nothing: the waiting happens inside the binary, not in the model. Stop it
with Ctrl-C or by saying stop, and it prints what the session did: windows seen,
threads answered, work carried out, changes proposed, files changed in your
notes. Nothing keeps running after that. There is no watcher and no daemon, and
liveness ends with the session.

One live session watches one document, the link you gave. Watching every paired
note in the folder is still planned.

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
(`docs/v2/PLAN.md`). It holds the credential, it reads, it writes suggestions,
since M6 it builds a house-style document and publishes it, since M7 it surveys
what a document holds before anything is done to it, and since M7b it can give
that document the house style where it stands.

```bash
make build   # bin/gdoc, for this machine
make dist    # bin/gdoc-darwin-arm64, -darwin-amd64, -windows-amd64.exe
```

```bash
bin/gdoc auth status   # which token, where it is, whether it has expired
bin/gdoc auth login    # prints a sign-in link and waits for you to open it
```

`auth status` answers when you are signed out too: no token is a fact it
reports, not an error. A token file that is there and cannot be read is a
different answer: it fails and names the file, because "signed out" would send
you to `auth login`, which overwrites the file and loses the evidence.

If the token was granted less than the Go binary asks for, `auth status` still
says ok and lists the difference in `missing_scopes`, with a warning naming the
scopes and telling you to sign in again. A token from the Python tool is not one
of these: it asks for the read-only Docs scope, but it also asks for the full
Drive scope, which the Docs API accepts, so nothing is reported missing.

`auth login` prints the link rather than opening a browser for you, because the
Go binary runs no other program at all. While it waits it listens on 127.0.0.1
on a port the kernel picks, which is what your browser comes back to, so a
firewall may ask once. It gives up after three minutes.

Both print exactly one JSON object on stdout and nothing else. Prose and the
sign-in link go to stderr, so anything reading the output has one object to
parse and no filtering to do.

It reads the same `~/.config/gdoc-agent/oauth-token.json` the Python tool
writes, in the same format, so a Python login already signs you in here.

The other direction is not symmetrical yet. `bin/gdoc auth login` asks for the
Docs read/write scope, because writing suggestions needs it, while the Python
tool asks for the read-only one and checks that what it asked for is in the
file. So after a Go login, `gdoc edits` asks you to log in through the Python
tool once more. Nothing else in either tool is affected.

`GDOC_CONFIG_DIR` moves both files somewhere else, which is what the Go test
suite uses so tests never touch your real config.

### Reading a document

Four commands read, and none of them writes to Drive. Each one takes the URL
you paste from the browser, or a bare document id.

```bash
bin/gdoc read <url> [--structure]
bin/gdoc comments <url> [--since CURSOR] [--wait DURATION] [--witness]
bin/gdoc suggestions <url> [--md PATH]
bin/gdoc restyle <url> --dry-run
```

They print facts and nothing else. Whether a comment is answered, whether a
suggestion that disappeared was accepted or thrown away, whether the document
and your markdown have drifted apart in a way that matters: none of that is
decided in Go. The skill reads the JSON and judges.

**`read`** gives you the document as one string of text, with everything that is
pending marked in it:

| In the text | Means |
|---|---|
| `# Heading` | a heading, one `#` per level |
| `{+text+}[s:ID]` | a pending suggested insertion, and its id |
| `{-text-}[s:ID]` | a pending suggested deletion, and its id |
| `[[c:ID]]text[[/c]]` | the text a comment is attached to, and the comment id |
| `<!-- tab t.0: Title -->` | the tab that follows, on a document with more than one |
| `[image]`, `[drawing]`, `[equation]`, `[object]` | content that is not text yet |
| `[person: Ada Lovelace]`, `[date: Sep 9, 2026]`, `[link: Q3 planning]` | a smart chip, with the label it shows |
| `[auto text: PAGE_NUMBER]`, `[page break]`, `[column break]`, `[rule]` | the rest of what a paragraph can hold |
| `[unknown: member]` | something in the document this version of gdoc has never seen |

Every placeholder row in that table comes back with a warning naming what the
read did not take from it, the chips and the breaks included. A policy with
eight person chips and three page breaks in it answers with eleven warnings, so
a non-empty `warnings` list does not on its own mean the read went wrong: read
the messages rather than counting them.

If the document's own text contains one of those markers, it comes back with a
backslash in front of it, and a backslash the author typed comes back doubled.
So the rule for reading the text back is a parity: an even run of backslashes is
the author's own text and the marker behind it is gdoc's, an odd run ends in
gdoc's escape and the marker behind it is the author's. The escaping is done one
text run at a time, so a marker whose two halves fall in two runs, or a document
character sitting against one of gdoc's own markers, can still reach the text
unescaped. That gap is written up in
`docs/backlog/escaping-across-run-boundaries.md`. `[object]`
is an embedded object the read could not classify: calling it an image would be
a guess.

Smart chips are read since M7, and before M7 they were not read at all: a person
chip, a date chip and a calendar link were dropped without a placeholder and
without a warning, so a policy naming its owner through a chip came back naming
nobody and nothing said so. Six other things went the same way, including page
breaks and horizontal rules. They all print a placeholder now. A chip's
placeholder carries the label the document shows, so `[person: Ada Lovelace]`
tells you who is there; the email address and a link's target are in
`--structure`. `[unknown: member]` is the wider half of the same fix: something
in the document this version of gdoc has never seen, named by what Google calls
it, rather than silently absent. Tables become pipe tables, with a literal `|` in a cell escaped as
`\|` so the row keeps its shape, and footnotes are appended after a `---` line.
Lists come back as `- ` items, two spaces of indent per level, so a numbered
list reads back as a bulleted one: telling the two apart needs the document's
`lists` map, which this milestone does not read. `--structure`
adds the document tree with character indexes on it, which is what a later
milestone needs to place a suggestion at an exact position. The text is not a
summary of the structure, and the structure is not a summary of the text.

**`comments`** lists the threads. Each one carries its author, its content, its
replies, whether it is resolved, the sentence it quotes, and the character range
the Docs read placed it at. A thread the Docs read could not place comes back
with `range: null` and a warning, rather than failing the whole listing. The
marker is a fact too: `ai:`, `ai?`, `ai!` or `none`, taken from the first word.
`@ai` is not one of them. The Python tool still accepts that old form; the Go
binary matches the three exactly.

`--since` takes the `cursor` a previous run printed and asks for what changed
after it. The cursor is opaque: it is the newest activity that run saw, encoded,
and nothing reads inside it. Every listing prints one, so a live session can
start on any document: where the run saw no activity at all, which is a document
nobody has commented on, the cursor is dated from the run's own clock a few
minutes back rather than from anything anybody did. Nothing writes it down either, so it lives as long
as whatever is polling.

`--wait` turns that one call into a poll. The binary asks Drive every ten
seconds inside the call, and comes back the moment something happened after the
cursor, or at the deadline with an empty window. It needs `--since`, because a
wait with no cursor answers with the whole document, which is the plain listing
under another name. The value is a Go duration, `9m` or `90s`, and an hour is
the most one call will look for. A wait keeps nothing of its own: the cursor it
prints is all that carries to the next call, and the only file it can touch is
the saved OAuth token, which any command replaces when the access token has to
be refreshed.

With `--wait` the object carries one more field:

```jsonc
"waited": { "polls": 12, "seconds": 118, "interrupted": false }
```

`polls` is how many times it asked, `seconds` how long it looked. A run without
`--wait` has no `waited` field at all. Ctrl-C during a wait is an answer rather
than a crash: the object comes back `ok: true` with no threads, the cursor you
handed in, and `interrupted: true`, and the exit code is still 0. A poll that
failed ends the wait with `ok: false` and says how many polls it made, so
whatever is looping can say the window is unread rather than empty.

`--witness` reads the document a second time, as a docx export, and says of each
thread whether that export carries it `anchored` to text, `detached` from it, or
`unmatched`. The export is the only truthful answer to "is this comment still
attached to anything", which is why it is a second read rather than a field.

**`suggestions`** lists what is pending, each with a stable id, the heading it
sits under and its text. With `--md` it also reports what stopped being pending
since the last look, by comparing against the snapshot in that file's front
matter, then writes the new snapshot. That write happens only after a read that
fully succeeded, and only into a file already paired with the document you read.

### Surveying a document before you restyle it

```bash
bin/gdoc restyle <url> --dry-run
```

This says what a document holds before anything is done to it, and it writes to
no document and to no file:

```jsonc
{
  "document_id": "1AbC...", "revision_id": "ALm37BX...",
  "title": "Supplier Register Policy",
  "tabs": 1,
  "threads": { "open": 3, "resolved": 7,
               "witness": { "anchored": 9, "detached": 1, "unmatched": 0 },
               "witnessed": [ { "id": "AAABc...", "witness": "anchored", "resolved": false } ] },
  "suggestions": { "pending": 2, "on_elements": 0 },
  "chips": { "person": 4, "date": 1, "rich_link": 2 },
  "named_ranges": [ { "id": "kix.abc123", "name": "gdoc-checklist" } ],
  "nothing_to_protect": false
}
```

A run gives `--dry-run` or `--from`, never both. The survey is what makes the
restyle safe, so it has to be something you read before you asked for the write,
rather than something the same run produced a moment earlier and never showed
you.

The witness is per thread as well as counted, because it is a before picture:
restyling a document can detach a comment, and a thread that was already
detached beforehand would otherwise look like damage the restyle did. It has the
same two limits it has under `comments --witness`: it finds a destroyed anchor
and not a moved one, and two comments with the same words that disagree give no
answer for either.

`nothing_to_protect` is true only when there are no threads, nothing pending, no
chips, no named ranges, and nothing in the document gdoc could not name. It is a
fact about those five, not advice: whether a document is worth restyling is
yours to decide from the counts. A named range is in that list because it is a
label Docs keeps in step with its own edits, so a replacement of the words it
covers takes it with them. The pending count includes a suggestion whose text is only
whitespace, which the `suggestions` listing leaves out, because it is still
something a rewrite would destroy.

Pending is two numbers, and they are counted in different units, so do not add
them together. `pending` counts the entries the pending walk finds, one per
suggested insertion and one per suggested deletion, so a replacement counts as
2 there. It is not the length of the `suggestions` listing: that listing leaves
out the whitespace-only ones this count keeps, as the paragraph above says.
`on_elements` is counted as ids, so the same replacement counts as 1.

`on_elements` is the suggestions that listing cannot report. It reads text runs
alone, so a suggestion carried only by a run it skips reaches it in no form: a
chip, a page break, a horizontal rule, an auto text, a picture, a footnote
reference. Counting nothing for them would answer "nothing to protect" over a
change somebody is waiting on. A run with one is still printed by `read`, inside
its markers, with its id.

`revision_id` is what the restyle rechecks. Hand this file back with `--from`
and the run refuses the document if it has moved since, and every batch it sends
carries that revision so Docs refuses it too.

An export that could not be read is a warning, not a failure: you still get the
threads, with every witness reported `unmatched`. A Docs read that failed is a
failure, because the chips, the suggestions, the ranges, the tab count and the
revision id are all in that one read.

### Restyling a document in place

```bash
gdoc restyle <url> --dry-run > survey.json
gdoc restyle <url> --from survey.json
```

Two runs. The first surveys, the second gives that document the Altery house
style where it stands: the page size and margins, each paragraph's spacing and
indent, each run's face, size and colour, and each table cell's padding and
borders.

**This is the one thing gdoc does to a document you handed it.** Everywhere
else, every change is a suggestion you accept or throw away. Here the change is
made. What holds it in is narrow: the run styles the one document you named and
the survey agrees with, only if the document has not moved since the survey, and
the four request kinds it may send cannot change a character of what you wrote.
The grant lasts one run and is not written anywhere.

**A restyle is a moment, not a setting.** The Docs API cannot redefine a
document's named styles, so gdoc applies the look paragraph by paragraph. The
document looks right afterwards, and the next heading you type is Google's
Heading 1 again, not the house one.

**What it overwrites is formatting inside the paragraphs it styles.** The face,
the size and the colour of every run go to the house value. Bold and italic are
left alone, and so is a table cell's own fill: the house style does not say
whether they should be off, and clearing them would be gdoc guessing. Your
words, your comments, your pending suggestions, your smart chips and your named
ranges are not touched.

**A failed run leaves a half-styled document.** There is no rollback, and the
recovery is the document's own version history, by hand. No text was touched, so
nothing you wrote is lost, but your own run formatting inside the paragraphs
that were restyled is. The run says so in its warnings.

Afterwards it reads the document back three ways and reports two things. What
survived: your threads with a witness for each, your pending suggestion ids and
your chips, before against after, with anything that moved named and never
explained. And whether the style is really there: it reads a margin, a
paragraph's spacing, a run's font and a cell's appearance back out of the
document and says which ones landed. `verified: false` is not a failure. What
was applied is in the document either way, and the field says the read-back
could not confirm all of it.

`manual` is what gdoc could not do, each with the menu path: the first-page
header with the logo, the contents list, the footer page numbers, plus any list
it left alone, because the request that sets a bullet also deletes the tabs that
set its nesting level, and any table's column widths, and any paragraph whose
named style the house has no look for.

A document with more than one tab is refused before anything is sent. A style
request names a range, and a range means nothing without saying which tab it is
in.

### Writing into a document

Four commands write into a document you handed in, and every change they make
there is a suggestion. None of them edits it, and the guard refuses the attempt
inside the process: a `batchUpdate` on a document you handed in is carried only
when the body says `writeMode: SUGGEST`. The one exception is the restyle above,
which is granted a direct edit on one document for one run, for four request
kinds that cannot change a character.

```bash
gdoc probe --folder <folder url or id>
gdoc reply <url> <comment id> --body-file reply.txt
gdoc propose <url> --from proposals.json --folder <probe folder> [--md note.md]
gdoc withdraw <url> <suggestion id> --md note.md
```

None of them decides what to write. The body of a reply, the words of a proposal
and the reason for it arrive already written, in a file the skill wrote.

**`probe`** asks Google whether suggestions are honoured for this project today.
It creates a throwaway document in the folder you name, writes one sentence into
it directly, suggests one word inside that sentence, reads the document back,
and trashes it. `enrolled: true` means the suggested word came back carrying a
suggestion id. `enrolled: false` means it came back as plain text, which is a
silent direct edit: the exact failure this asks about. The report names
`probe_document_id` and says whether the trash succeeded, so a document left
behind is named rather than lost.

The probe exists because the field that makes a write a suggestion is not in the
public Docs discovery document, and one morning that call returned 200 and
edited the document for real. `docs/v2/BLOCKED-BY-API.md` records both
measurements. So `propose` runs the probe every time, on a document of its own,
and sends nothing when the answer is no.

**`reply`** posts one reply into a comment thread and reads the thread back to
see it there. The body must open with `🤖 ` and nothing before it, which is how
a later run tells gdoc's own replies from everybody else's, and it must be plain
text: a Docs thread renders markdown literally, so asterisks and backticks
arrive as typed. Both are refused naming what was found.

**`propose`** writes a change as a native suggestion with a comment beside it
saying why. It reads the file `--from` names, a list of
`{quoted, replacement, why, assignee?}`:

```json
[{
  "quoted": "reviewed annually",
  "replacement": "reviewed every six months",
  "why": "the policy above says six months"
}]
```

`quoted` is text, never a position. An index worked out from an earlier read is
the hazard the whole API has, so `propose` reads the document fresh, finds the
words, and refuses when they occur more than once: quote more of the sentence.
Text already inside a pending suggestion is not matched, so a proposal on top of
a proposal is refused too. A quote running across a footnote mark, a picture, an
equation or a page break is refused as well, naming what it crossed: the span
would take that content with it. An occurrence that crosses one still counts, so
a quote reading once as plain text and once across a mark is refused as
ambiguous rather than placed on the plain one. Each proposal is one
`batchUpdate` after its own fresh read, because the first change moves the
ground under the second, and a document with more than one tab stops the run
before anything is sent.

`replacement` may not be empty, and it may not carry a line break: a proposal
replaces words with words inside one paragraph, and there is neither a
deletion-only shape nor one that adds a paragraph. `why` becomes the comment, so it must be plain text
and must not carry the `🤖 ` prefix itself, which gdoc adds. Every proposal in
the file is checked before the first one is sent, so a bad entry stops the run
with nothing written.

**`verified` is the read-back, never the status code.** After the batch,
`propose` reads the document three more ways, and `checks` says which held:

| Check | Asks |
|---|---|
| `suggestions_inline` | the replacement is there, carrying a suggestion id |
| `preview_without_suggestions` | the original words are still there with suggestions hidden, so it is a suggestion and not an edit |
| `docx_anchored` | the docx export carries the 🤖 comment, attached to text |

All three, plus a write that answered `commentUpdateState: ALL_SAVED`, is
`verified: true`. Anything less is still `ok: true` with `verified: false` and a
warning naming what did not hold, because the write happened and hiding that
would be worse. A batch Docs accepted whose answer could not be read is the case
where every check holds and `verified` is false, and the warning there names the
lost answer. The skill reads `verified` and decides what to say.

**`withdraw`** retracts one of gdoc's own pending proposals. It needs `--md`,
and the reason is the whole rule: the note's `proposals` list is the only record
of which suggestions gdoc wrote, and gdoc withdraws only those. An id that is
not in there is refused. The write is a `rejectSuggestion` naming that id, and
the guard carries it only because the command granted exactly that id for this
run: gdoc still never accepts, rejects or deletes anyone else's suggestion. The
suggestion is gone only when the answer names it in `rejectedSuggestionIds` and
a fresh read shows no run carrying it on either side; only then does the entry
leave the note. The 🤖 comment stays where it is, because deleting a comment is
a write the guard does not carry.

With `--md`, `propose` records what it wrote into the note's `gdoc:` block:
the suggestion id, the comment id, the time and the words that were replaced.
That record is the permission to withdraw later, so a run without `--md` still
lands the suggestion and simply forgets it.

The list is not a complete record on its own. A proposal gdoc does not have both
ids for cannot be written down at all, because an entry missing either fails the
block's own validation, so the run warns that gdoc cannot withdraw it later. The
warning says which id and where it would have come from: the comment id is the
write's own answer, the suggestion id is read back afterwards. The change is in
the document either way. `files_changed` is per run, not per proposal: the note
is left out of it only when nothing at all could be added to it, so a run that
recorded two proposals and lost the third still names the note and carries the
warning about the one it lost. Read the warnings, not the file list. `gdoc suggestions` is the document's own
answer, and the note is gdoc's memory of it: read both.

The review skill runs over these commands. It reads the threads, decides which
ones still need an answer, and writes the reply and proposal files the binary
sends. The binary reports facts either way: there is no `handled` field, and no
rule in Go that says a comment is answered.

### Building the document

`gdoc build` turns a note into an Altery house-style `.docx` on your own
machine. It reaches nothing: no Drive, no Docs, no network at all, and it runs
no other program. `gdoc publish`, below, is the same render with the upload
behind it.

```bash
gdoc build --md note.md --out note.docx [--house house.yaml] [--force]
```

The house style is a file, `house.yaml`, and it is embedded in the binary, so
there is still nothing to install beside it. It states the page geometry, the
nine named styles, the cover, the header and footer with the logo, the three
front-matter tables cell by cell, the legend, the contents field and the heading
numbering. `--house` points at another copy of that file for one run, which is
how a change to the style is looked at before it is committed; the output names
which one was used, `embedded` or the path, so a document built from a draft
says so.

`--out` will not overwrite a file that is already there. Add `--force` when you
mean to replace it. `--force` is consent to replace a document, not a folder and
not something the run reads: an `--out` naming a directory, the note, the
`--house` file or one of the note's own pictures is refused whatever the flag
says. The write goes through a temporary file and a rename, so a failed build
cannot truncate a document you already had.

The note needs a `title` in its front matter and nothing else. These are the
keys it reads, and they are the same names the Python tool uses, so a note
written for that publishes here with no edits:

| Key | Does |
|---|---|
| `title` | required. The cover, and the running head in the page header |
| `alt_title` | the second title line on the cover, under the word `or`, and the running head. A note that states none prints neither line |
| `doc_type` | joined to the title on the cover, so `Third Party Risk` plus `Policy`. Free text, and a title that already ends in its own type is left alone |
| `version` | the cover, behind the word `Version:`. `1.0` when the note states none |
| `date` | rendered on the cover in UK long form. The month the build runs in when the note states none |
| `owner` | the Document Owner row of the version-control table |
| `last_approval` | the Date of Last Approval row |
| `review_frequency` | the Review Frequency row |
| `board_ratification` | the Board Ratification Date row |
| `distribution` | the Policy Distribution row |
| `classification` | one of `confidential`, `restricted`, `internal`, `public`. It shades that class's row in the classification table, and the rest are left clear. `internal` when the note states none |
| `heading_numbering` | `false` turns off the `1-Scope` numbering on level-one headings |
| `revisions` | rows of the revision-history table: `version`, `date`, `author`, `approved_by`, `approval_date`, `section`, `change`. A note that declares none keeps the template's own row |

A note with no `title` is refused, and the refusal proposes one: the first
heading in the body, or the file name. The binary never invents a title and
never writes one into your note. Any other key in the front matter is carried
untouched, the `gdoc:` block included.

The body is your markdown: headings to six levels, bulleted and numbered lists
three deep, bold, italic, strikeout, `==marked==` text as a highlight, links,
pictures, tables, horizontal rules and block quotes. A picture is read from the
note's own directory, or decoded when the markdown carries it as a `data:` URI;
PNG and JPEG. A picture at an `http` address is refused naming the line, because
a document built from a link is a document that breaks when the link expires.

Code blocks are not rendered. The house style has nothing to render them in, so
a note carrying one gets a warning naming the line and the block is left out
rather than dropped in silence. Blocks of HTML, inline HTML and footnotes are
the same answer. Footnotes matter more than they look: `gdoc read` writes a
document's footnotes as `[^1]` in the prose and the definitions after a `---`
line, so a note pulled out of a Doc carries them, and each one is named by the
line the author wrote it on. So is a picture inside a list item, a block quote or a table cell: the
house style puts a figure on a centred line of its own, which it cannot be
there, and the warning names the line.

Two things about a numbered list carry a warning rather than the numbers you
wrote. The document defines one numbered list, so a second one carries on from
the first: your 1. and 2. print as 3. and 4. And every level of it starts at 1,
so a list you opened at "5." opens at 1. Both name the line. A heading that
skips a level is the third: a `###` under a `#` is numbered `1.0.1-`, and the
warning carries the number it wrote.

A list item that holds one of those and nothing else, or nothing at all, has no
words to put a marker on. It takes none, and the warning names the line and
says what that costs: in a numbered list the items after it print one lower, and
in a bulleted list the item loses only its own bullet and its indent.

What you get back is one JSON object: the file it wrote and its size, the title
and the running head it used, which house file it read, and what the walker
counted.

```json
{
  "ok": true,
  "data": {
    "out": "/Users/you/notes/supplier-register-policy.docx",
    "bytes": 48213,
    "title": "Supplier Register Policy",
    "running_head": "Altery - Supplier Register Policy",
    "house": "embedded",
    "body": {"paragraphs": 41, "headings": 9, "lists": 3, "tables": 2, "images": 1}
  },
  "warnings": ["line 88: a code block is not rendered in the house style and was left out"]
}
```

Those counts are facts and nothing more. Whether the document is right is
answered by opening it, in Word or in Drive, and not by the binary.

The contents list is a Word field, which is what the master template carries
too. Word fills it in when the document is opened and refreshed, and Google Docs
turns it into a live contents list on import. Until then it shows the one line
Word writes into an unrefreshed field.

There is one test that keeps this honest on every commit. It builds a document
from the embedded style, opens the Word master the style was extracted from, and
compares 169 measured values across the two: page geometry, all nine styles, the
header, the footer, the logo's position, the contents field, the three tables
and the body. Every row that differs, and every row the master has nothing to
compare against, is named in a list with the reason it is there, most of them
because the master states a value twice and because the two documents hold
different words. A row inside its own tolerance reads as close and needs no
entry. Any other difference fails the test suite, so the style cannot drift away
from the master quietly.

### Publishing the document

`gdoc publish` does what `build` does and then puts the result into Drive as a
Google Doc, in one folder you name, and writes the pairing back into your note.

```bash
gdoc publish --md note.md --folder-id FOLDER [--house house.yaml]
```

The note is rendered exactly the way `build` renders it, so everything in the
section above applies: the same front-matter keys, the same warnings, the same
refusals. The docx bytes are uploaded with conversion, so Drive turns them into
a document rather than leaving a .docx sitting in a folder.

The folder is the only thing the run can reach. No document is in reach when it
starts, and the new document's id comes back from the create the tool itself
made. There is no `--folder-id` default and no fallback to the folder in your
note: a note that already names a document is refused before anything leaves
your machine. Publishing a second version of a note is `restyle --new`, which is
still deferred: it needs a markdown export and a markdown writer that the Go
tool does not have.

Three things are checked after the upload, and each answers something the other
two cannot: the document reads back through the Docs API, it has exactly one
tab, and Drive can export it as a .docx again. `verified` is the three together.
Fewer than three is still `ok: true` with the check that did not hold named,
because a document that exists is a document that exists, and being told the run
failed is what makes somebody upload a second one.

If the upload worked but your note could not be written, the document goes back:
it is trashed and the trash is confirmed, and the output says `rolled_back:
true` with no document id. Only if that also fails do you get the live id and
what to do with it. The reason is not tidiness. A document nobody's note points
at is one the next publish makes a second of.

```json
{
  "ok": true,
  "data": {
    "document_id": "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
    "folder_id": "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr",
    "url": "https://docs.google.com/document/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd/edit",
    "title": "Supplier Register Policy",
    "house": "embedded",
    "bytes": 48213,
    "tabs": 1,
    "verified": true,
    "checks": {"read_back": true, "one_tab": true, "docx_export": true},
    "files_changed": ["/Users/you/notes/supplier-register-policy.md"],
    "body": {"paragraphs": 41, "headings": 9, "lists": 3, "tables": 2, "images": 1}
  },
  "warnings": []
}
```

Your note is read again just before it is written, because the upload takes
seconds and these notes live in a synced vault. If it changed in that window, in
its body or in its front matter, nothing is written and the document is rolled
back: the file on disk is no longer the one that was rendered, and publishing it
as though it were would pair your note with a document it does not match.

There is a second test that only runs when you ask for it, and it is the one
that means something: it builds the document, uploads it and the Word master
side by side, reads both back through the Docs API, and compares the same 169
values. Google's import is part of that reading, so a difference that shows up
there and not offline is Drive's doing rather than the generator's. It creates
two real documents and trashes them, so it is behind two environment variables
and never runs by itself.

### The `gdoc:` block

A markdown note paired with a Go-published document carries one key in its front
matter, and everything else in there stays the author's:

```yaml
---
title: Supplier register policy
gdoc:
  schema: 1
  document_id: 1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd
  folder_id: 1w0SresizE9Kr810VZRJwX4JtDBF4OqNr
  published:
    at: 2026-09-08T10:14:00Z
    title: Supplier register policy
    house: embedded
  suggestions_seen:
    at: 2026-09-06T11:00:00Z
    items:
      - id: suggest.abc123
        kind: insertion
        section: Scope
        text: "critical "
---
```

The read is strict. An unknown key, a key given twice, a missing `document_id`
or a `schema` this version does not know is refused naming the key, and the file
is left alone. The write touches the `gdoc:` lines and nothing else: your keys,
your line endings and the trailing newline come back byte for byte, and the file
is replaced through a temporary file and a rename, so a failed write cannot
truncate your note.

`gdoc publish` is the only thing that creates this block. `gdoc suggestions
--md`, `gdoc propose --md` and `gdoc withdraw` update it, and each of them
refuses a note that does not carry one already. A note the Python tool
published carries `gdoc: <id>` as a plain string instead, and the Go binary
refuses to read that: it names the shape and tells you to rewrite the line by
hand once, or to publish the note again with `gdoc publish`. Building is
unaffected, because `gdoc build` skips the `gdoc:` key whatever is in it.

The `gdoc` on your PATH is the Go binary from here on. `./install.sh` links
`~/.local/bin/gdoc` to `bin/gdoc`, and the temporary second name `gdoc2` is
removed. The Python tool is still there and still does the publishing: it is
`~/.config/gdoc-agent/venv/bin/gdoc`, which is the full path its two skills call,
so nothing about them changed. Every `gdoc ...` example outside this section is
the Python tool, and needs that path now.

Two words mean two things across the two tools, and both are worth knowing
before you type them. v1's `read` lists the comments; v2's `read` prints the
document text, and v2's `comments` lists the comments. v1's `restyle` publishes
a house-styled copy of a document into another folder; v2's `restyle --dry-run`
reads one document and writes nothing.

## What is planned

**A restyled document that stays restyled.** `gdoc restyle --from` gives a
document the house style today, paragraph by paragraph, because the Docs API
cannot redefine a document's named styles. Making the style stick, so the next
heading you type is the house one, needs something Google does not offer yet.

**Lists and table layout in a restyle.** Bullets are left alone, because the
request that sets one also deletes the tabs that set its nesting level, and
column widths and row heights need two request kinds nobody has measured.

**A live session over the whole folder.** Live works today on one document, the
link you give it. Starting it once and having it watch every note you have
published, so a comment on any of them is answered without naming which, is
designed and not built yet.

**Ordinary comments.** Reading and answering comments that carry no marker.

**Fewer things to install.** Google can hand back markdown by itself, so the
pandoc dependency should shrink.

**A friendly reply name.** A real Workspace user for the agent, so `service_account`
threads stop showing a raw address.

---

Working on the tool itself? Start with [PRINCIPLES.md](PRINCIPLES.md), then
[CLAUDE.md](CLAUDE.md).
