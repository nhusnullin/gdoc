---
name: gdoc-export
description: Use when the request gives a Google Doc link and asks for that document in the hub as a Markdown note, and no note is paired with it yet. Reads the document, writes one file per tab with its pictures beside it, and resolves gdoc's markers with you.
needs: v2.4.0
---

# Bring a document into the hub

This is `gdoc-export`. Say that in the first line of your reply, because two
skills answer a request about a document and a note: this one brings a document
in when there is no note for it, and `gdoc-align` merges a document with the
note it is already paired with.

You give a link. gdoc reads the document and writes it into the hub as Markdown:
one file per tab, the pictures as PNG files in `assets/` beside it, the house
cover stripped, and everything the document says still in the file. A pending
suggestion arrives as a marker. A comment's words arrive with a marker around
them. Those markers are gdoc's, and this skill resolves them with you.

Export changes nothing in Drive and replaces nothing on disk.

## Is this the right skill

Read the request and the hub before running anything:

- **A link, and no note in the hub for that document.** This skill. Carry on.
- **A link, and a note whose `gdoc:` block names that document.** Not this
  skill. Say so in one line, name `/gdoc-align`, and stop. That skill exports
  too, compares the copy with the note, and merges with your agreement.
- **A note and a folder, and no document yet.** Not this skill either:
  `/gdoc-publish` makes the document.

When you are not sure whether a note exists, search `$ROOT` for the document id
out of the link before you decide, and say what you found.

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

One root, `$ROOT`, and it is `$PWD`. It is the hub: the tree the new note is
written into, and the only place this skill writes at all.

Nothing here runs git. Not to commit, not to check whether a file is dirty.

## Learn the command before calling it

Before the first call of a command in this session, run:

```bash
$GDOC help <command>
```

It prints the words that command takes, every flag with one sentence saying
what it is for, and one example. Read the words and the flags out of that
object and build the call from them. This skill names no flag of its own, on
purpose: the binary is the only thing that knows what it takes today.

Two commands, and no others: `export`, and `comments` for a thread's author and
first line. This skill never publishes, never proposes and never replies.

## If the credential is not working

Both commands reach the network and can fail with no token or a permission
error. Run `$GDOC auth status` and read it before guessing. It reports whether a
token is present, whether it expired, and which scopes are missing.

The fix is `$GDOC auth login`. It prints a URL, waits for you to approve in the
browser, and saves the token. Ask before running it: it changes which account
reads the document, and a document you can open in the browser is not always one
that account can read.

Never edit `~/.config/gdoc-agent/` by hand, and never tell anyone to.

## If the request is a dry run

Do Step 1 and Step 2, write nothing, and say at the end that no file was
created. There is no dry run in the binary for this: `export` writes files, so
a dry run stops before it.

## Step 1: Ask for the file name

The document's title is the candidate. Slug it the way a file in the hub is
named: lower case, every run of characters outside letters and digits turned
into one hyphen, `.md` on the end. Say the path you would write, under the
folder the request named or the hub's own place for that kind of note, and wait
for your word on it.

Ask once. A name you approve is the name; a name you change is the name you
gave. This is the one question before anything is written, and it is asked
because a file in the hub is somebody's to find later.

Nothing is overwritten whatever you answer. A path that is taken gets the next
free number, `<stem>.2.md` and then `<stem>.3.md`, and the reply says the path
was taken. There is no flag that replaces a file.

## Step 2: Export

Run `$GDOC help export`, then build the call from what it printed: the link, and
the path you agreed.

```bash
$GDOC export <url> --out <path>
```

One call. It makes two reads of the document and no writes, so running it again
costs a read and another file, never a change in Drive.

## Step 3: Read the object and say what it says

The binary prints facts and judges none of them. Read the object and say, in
this order:

- **`files`.** Every file it wrote, with `tab_id` and `tab_title` where the
  document had more than one tab. Name each path. On a document with tabs, say
  how many files came out and which tab each one is.
- **`pictures`.** Each one carries `file`, the path it wrote, or `matched`, the
  note's own picture when the bytes were the same. In this skill there is no
  note yet, so expect a `file` on every picture.
- **`pending`.** How many suggestions are still pending in the document. Those
  are the `{+ +}` and `{- -}` markers in the file.
- **`threads`.** How many comment threads the read found. Those are the
  `[[c:ID]]` marks.
- **`stripped`.** The house prelude pieces that came out, each with `kind` and
  the `text` it held. Show them. A piece whose text is not what a cover usually
  holds is worth a second look, because it is the author's own words leaving the
  file.
- **`stamped`, when it is there.** The notes this run wrote a date into. In this
  skill it is normally absent: there is no note to stamp.
- **`multi_tab`.** True means the document has more than one tab. Every writer
  still refuses such a document, so say that the new notes can be published as
  new documents and cannot yet propose back into their tab.
- **Every warning, each one in full.** A warning is something the file could not
  carry: a floating picture as a placeholder comment, a footnote flattened, a
  Drawing with no PNG, a picture count that did not match, a prelude the run did
  not recognise. Read them out. A prelude warning means nothing was stripped and
  the cover is still in the file, which is yours to take out. A second warning
  then names the headings that open with digits and a `-`. It says nothing about
  whose they are, and neither do you: the same warning reaches a published
  document whose cover somebody edited and a document gdoc never touched. Read
  the headings it names, say that `2024-2025 Budget` looks like the author's own
  words, and ask. Take a number out only when you say `gdoc publish` wrote this
  document. Leaving one costs a second edit; taking one out deletes words
  nothing puts back.

## Step 4: The threads, when there are any

```bash
$GDOC comments <url>
```

Run it only when `threads` is more than zero. List each thread by id, author and
first line. The comments stay in Drive: this skill never copies a comment's text
into the note, and never replies.

The words a comment was attached to are in the file, wrapped in `[[c:ID]]` and
`[[/c]]`. Step 5 takes the marks off and keeps the words.

When a thread carries a marker, `ai?` or `ai!`, say that `/gdoc-review` is the
skill that answers it. This one does not.

## Step 5: Resolve the markers with the person

Read `markers.md`, beside this file, before you touch a marker. It holds every
marker, what it means, how it is escaped and how a session undoes it. Follow it.

The short of it: a `{+words+}[s:ID]` is somebody's pending insertion and a
`{-words-}[s:ID]` is their pending deletion. The document is untouched either
way: taking a suggestion into the note is not accepting it in Drive, and the
person who left it still has it pending. So ask per suggestion, by id and words,
and never by author: the Docs API never says who wrote one.

Then take the comment marks off and keep the words, and say which threads they
belonged to.

When the file is done, add `title` to its front matter, and nothing else. The
`gdoc:` block is gdoc's and you never write into it by hand. A note with no
`title` cannot be built or published, so this one key is the skill's to add; the
other cover keys are the author's.

## Step 6: When there is no time today

A person with ten minutes says so. Then: export, add `title`, stop.

Say it in one line: the file is in the hub with its markers, the markers are
gdoc's own, and any later session can resolve them. Nothing decays. The one
thing a marked file cannot do is go out again: `build`, `publish` and every
writer that takes text from a note refuse a marker by line and name the line.

The later session's request is "resolve the markers in `<name>.md`", and the
skill that answers it is `/gdoc-align`, which runs `$GDOC suggestions <url> --md
<note>` first so a suggestion the document decided in the meantime is resolved
from that answer without a question.

## Step 7: Report

```
wrote      notes/cbc-emi-guidance.md              from tab t.0
wrote      notes/cbc-emi-guidance-appendix.md     from tab t.1 "Appendix"
pictures   assets/cbc-emi-guidance-1.png, assets/cbc-emi-guidance-2.png
stripped   cover, 3 tables, legend, contents
pending    4 suggestions, resolved 3 with you, 1 left as a marker
threads    2 open, listed below, nothing written into the note
title      added: CBC EMI guidance
warnings   1 floating picture is a placeholder comment after its paragraph
```

Then the threads, one line each with id, author and first line. Then say what is
next in plain words: the note is a note like any other, `/gdoc-publish` makes a
house-style document from it, and that new document carries none of this one's
comment threads.

If part of the export could not be read, say so and do not report clean. A
warning naming a picture, a prelude or a footnote means the file is not the whole
document, and the person merging it later has to know which part.

## Never

- Never run from an `ai!` comment. Export starts from a session by name. The
  review skill answers such a comment with one reply saying so.
- Never publish and never propose. This skill reads a document and writes in the
  hub. The two doors the other way are `/gdoc-publish` and `/gdoc-align`.
- Never accept, reject or resolve anything in Drive. Taking a suggestion's words
  into the note leaves the suggestion pending where it is.
- Never reply to a thread, and never copy a comment's text into the note.
- Never write into the `gdoc:` block. gdoc writes it. `title` is the one key this
  skill adds.
- Never replace a file. A taken path takes the next free number, and there is no
  flag that says otherwise.
- Never delete anything from the hub or from Drive.
- Never run git.
- Never trust a status code. Read the object, and read every warning.
- Never report a marked file as finished. Say which markers are left and that a
  later session resolves them.
- Never export a PDF. You download it from the browser.
