---
name: gdoc-publish
description: Use when the request names a note in the hub and a Drive folder and asks for that note in Drive as a Google Doc in the Altery house style. Reads the note's front matter, builds it, uploads it into the one folder the request named, and reads back what came out.
needs: v2.0.0
---

# Publish a note as a Google Doc

You name a note and a folder. The note is markdown in the hub. The folder is
where the document goes. gdoc renders the note in the house style, uploads it,
reads it back and records the pairing in the note's own front matter.

A note is published once. The `gdoc:` block in its front matter is the record of
that, and it is the reason the binary refuses a second publish. The block is
gdoc's memory of a document, not a setting somebody tidies away.

## Setup

```bash
GDOC=gdoc
ROOT="$PWD"
$GDOC help
```

`gdoc` is the v2 binary, on PATH. It prints exactly one JSON object and exits.
Exit 0 means the object says `ok`. Read the object, never the exit code alone.

`help` is the first call of every session, because its object carries
`version`, the release this binary was built from. This skill needs the version
its own front matter names on the `needs` line, or a later one. An older binary
is one the skill is ahead of: say so, say that `gdoc update` is the command
that fixes it, and stop there. No `version` at all is a build made from source
rather than a release, which is not an error: say it once and carry on.

The same object may also carry `update`, which says what is installed and what
is published: `installed`, `latest_stable`, `latest_nightly` and `checked_at`.
When `latest_stable` is there and its three numbers are ahead of `version`,
say once that a newer gdoc is published, name it, and name what installs it:
`gdoc update`, or `gdoc update --major` when the first of the three numbers is
the one that is ahead, because a major release is one a person asks for by
name. Then carry on with the work. This is a remark and never a gate:
`needs` is the only version that stops a session. No `update` key at all is a
build from a checkout, and the skill says nothing about it.

One root, `$ROOT`, and it is `$PWD`. It is the hub: the tree the note lives in,
and the only place a note is read from or written to. Scratch output, the dry
run's docx included, goes to the session's scratchpad and never beside somebody's
note.

Nothing here runs git. Not to commit, not to check whether the note is dirty.

## Learn the command before calling it

Before the first call of a command in this session, run:

```bash
$GDOC help <command>
```

It prints the words that command takes, every flag with one sentence saying
what it is for, and one example. Read the words and the flags out of that
object and build the call from them. This skill names no flag of its own, on
purpose: the binary is the only thing that knows what it takes today, and a
flag list written here would go stale without anyone noticing.

Two commands matter here, `build` and `publish`. `help` with no words prints
every command gdoc answers.

## If the credential is not working

Any command that reaches the network can fail with no token or a permission
error. `build` is not one of them: it touches no network at all. Run `$GDOC auth
status` and read it before guessing. It reports whether a token is present,
whether it expired, and which scopes are missing.

The fix is `$GDOC auth login`. It prints a URL, waits for you to approve in the
browser, and saves the token. Ask before running it: it changes which account
owns the document that is about to be created, and that account is visible to
everyone the document is shared with.

Never edit `~/.config/gdoc-agent/` by hand, and never tell anyone to.

## If the request is a dry run

Do every step, but put nothing in Drive and write nothing into the note.

The dry run is `build`, to a scratch path under the session's scratchpad. It
renders the same bytes `publish` would upload, on this machine, with no
network. Read its report back: the file it wrote, its size, the title and the
running head it took from the note's front matter, the house style it used, and
the counts of what it found in the body.

Say at the end that nothing was uploaded, no document exists, and the note's
front matter is untouched.

## Step 1: Read the note's front matter

Open the note and read its YAML front matter before running anything.

**A `gdoc:` block naming a document means this note is already published.**
Stop. Say it in plain words: this note is already paired with that document,
gdoc publishes a note once, so open that document, or, if it names a document
that has gone, take the block out by hand. Taking it out is yours, never this
session's. The binary refuses the run for the same reason, and the refusal names
the document id.

**A note with no `gdoc:` block is a note that can be published.** Read the
author's own keys at the same time. Only `title` is required. The optional
keys feed the cover page and the version-control table, and a key that is not
there leaves its line blank rather than taking somebody else's value.

**A note with no title is refused, and gdoc never invents one.** The refusal
carries a candidate, drawn from the note's first heading or from its file name.
Show the candidate and wait for your word on it. Then write it into the note's
front matter yourself, as `title`, and run again. Nothing else in the note
changes.

## Step 2: Publish

Run `$GDOC help publish`, then build the call from what it printed: the note,
and the folder you named in this request.

The folder is the one door this run has. gdoc is handed the folder and learns
the new document's id from the create it carried itself. So the folder must be
the one you named, in this conversation, and nothing else: not a folder another
note was published into, not one found in a different note's block.

Publish is one call. Never run it twice on one note. A run that failed has
already dealt with what it left behind, and Step 3 says how to read that.

## Step 3: Read the object back and say what it says

The binary prints facts and judges none of them. Read the object and say, in
this order:

- **The URL**, so you can open it.
- **`verified`, and each of the three checks under `checks`.** They are three
  read-backs on separate routes: the Docs read says the document is there and
  readable, the tab count says it is one document rather than a shape a later
  command would refuse, and the docx export says Drive can hand it back. Name
  the ones that came back false. `verified: false` is not a failed publish. The
  document exists. A route that did not hold is a route that did not hold, and
  saying the run failed is how a second document gets made.
- **`files_changed`.** The note whose front matter now records the pairing. If
  it is not there, nothing in the hub changed.
- **`rolled_back`, when it is there at all.** It is absent on a run that
  recorded the pairing. `true` means the pairing could not be written and gdoc
  trashed the document it had just made, confirmed by reading it back: there is
  nothing in Drive and nothing in the note, and the run can be tried again.
  `false` means the pairing could not be written and the rollback did not hold:
  the document is live, its id is on the envelope, and somebody has to trash it
  by hand. Say that plainly, with the id.
- **Every warning**, each one in full. A title that came back from Drive
  different from the one on the cover is a warning, not a check.

If the run failed before anything left the machine, say so: no document was
created and the note is untouched. If it failed after the upload and the answer
could not be read, the error names the folder and says a document may be in it.
Repeat that. Do not guess which.

## Step 4: The three things only you can check

The binary cannot see a rendered page. After a publish that produced a document,
say these three, as things to look at:

- the Altery logo in the first-page header,
- the contents list, which Docs fills in when the document is opened,
- the page numbers in the footer.

Those are facts about the document gdoc just made, and you are the one who
judges them. The session never claims them itself.

## Never

- Never remove or edit a `gdoc:` block in a note. gdoc writes it, and only
  `publish` creates it. A block in the way is a stop, not an obstacle.
- Never publish a note twice. One note, one document.
- Never pass a folder the binary was not handed in this request.
- Never invent a title, a date, an owner or a classification to make a build
  pass. A missing value is a question for you.
- Never run git.
- Never trust a status code. Read `verified` and `checks`.
- Never say a publish failed because `verified` is false. Say which route did
  not hold.
- Never delete anything from Drive, except through the rollback gdoc did
  itself, which it reports.
- Never export a PDF. You download it from the browser.
