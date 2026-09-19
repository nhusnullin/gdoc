---
name: gdoc-align
description: Use when a note in the hub and a Google Doc are paired and the request asks for the two brought back together, in either direction. Exports the document, shows the differences, merges what you agree to, and proposes the note's own changes into the document as suggestions.
needs: v2.4.0
---

# Align a note and its document

This is `gdoc-align`. Say that in the first line of your reply, because two
skills answer a request about a note and a document: `gdoc-export` brings a
document in when there is no note for it, and this one aligns a note with the
document it is already paired with.

The note is markdown in the hub. The document is in Drive. Colleagues edit the
document in the browser, sessions edit the note in the hub, and either side can
move. This skill reads both, shows you the differences, and carries what you
agree to across, in whichever direction you ask.

Neither side is the source of truth by rule. You decide, per run. The note's
`gdoc:` block records dated facts only: when a document was published from this
note, and when it was exported into it. No field says which side was right.

Every change this skill makes in Drive is a Google suggestion. It never edits a
document.

## Is this the right skill

- **A note whose `gdoc:` block names the document in the link.** This skill.
- **A link and no note in the hub.** Not this skill. Say so in one line, name
  `/gdoc-export`, and stop.
- **A note and a folder, and a fresh document wanted.** `/gdoc-publish`.
- **A document that has no house style and should get one where it stands.**
  `/gdoc-restyle`.

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
and the only place a note is read from or written to.

Nothing here runs git. Not to commit, not to check whether the note is dirty.

## Learn the command before calling it

Before the first call of a command in this session, run:

```bash
$GDOC help <command>
```

It prints the words that command takes, every flag with one sentence saying
what it is for, and one example. Read the words and the flags out of that
object and build the call from them. This skill names no flag of its own, on
purpose: the binary is the only thing that knows what it takes today.

Five commands, and no others: `export`, `read`, `comments` for a thread's author
and first line, `suggestions` for what is still pending, and `propose`. This
skill never publishes, never replies and never withdraws.

## If the credential is not working

Every command here but the file reads reaches the network. Run `$GDOC auth
status` and read it before guessing. It reports whether a token is present,
whether it expired, and which scopes are missing.

The fix is `$GDOC auth login`. It prints a URL, waits for you to approve in the
browser, and saves the token. Ask before running it: it changes which account
writes the suggestions, and that account is visible to everyone on the document.

Never edit `~/.config/gdoc-agent/` by hand, and never tell anyone to.

## If the request is a dry run

Do every step. Show the merge and the proposals, write nothing into the note and
send nothing to Drive. Print each proposal in the terminal instead of running
`$GDOC propose`. Export still writes a copy, because that is the only way to read
the document as Markdown: say which copy it wrote and delete it at the end, as
any run does.

## Step 0: Clear the stale copies, and say which direction this run takes

Two things before anything else, and both go in the reply.

**The copies.** Look in the note's own folder for a file whose `gdoc:` block
carries `exported.note` naming this note. That is a copy a run left behind. For
each one:

1. Say when it was made, from `exported.at` in its block.
2. Export again, Step 1.
3. Compare the fresh copy's body with the stale one. Identical means the stale
   copy holds nothing the document does not: delete it and say so.
4. Different means somebody edited the copy, or the document moved since. Show
   the difference and ask before deleting anything. There is no record of what
   the dead run decided, so reconstructing it is not on offer: the question is
   whether that copy's own edits are wanted.

**The direction.** Read the note and the document and say which of the four runs
below this is, before you do anything else:

- **Document into note**, when the document moved and the note did not.
- **Note into document**, when the note moved and the document did not.
- **Both moved**, when both did.
- **Resolve only**, when a marked file is already in the hub and the request is
  to finish it.

Say which one, and why you read it that way. A request that reads as "publish my
changes into the document" is note into document: the binary never replaces a
body, so a merge is what that sentence means, and the changes arrive as
suggestions the document's owner accepts.

## Step 1: Export the document

Run `$GDOC help export`, then:

```bash
$GDOC export <url> --out <note>.md
```

`--out` is the note, not a new name. The path is taken, so the copy lands beside
it at the next free number, `<note>.2.md`, and the reply says which file it
wrote. Nothing replaces the note: the one byte export writes into it is
`exported: {at}` on this document's entry, through the block write that keeps
every other byte.

Read the object as `gdoc-export`'s Step 3 says: `files`, `pictures`, `pending`,
`own`, `threads`, `stripped`, `stamped` and every warning. Two lines belong in
the reply here and nowhere else:

- **`stamped` with `schema_rewritten`.** The note's block was written before this
  milestone and has been rewritten from schema 1 to schema 2. Say it once, and
  say that a colleague on an older binary now gets a refusal naming `documents`
  until they run `gdoc update`.
- **`stripped`.** Compare each piece with the note's own front-matter keys. A
  cover that says something the note's keys do not is an edit somebody made in
  the document, and it is yours to decide. Show anything a piece held that the
  note does not.

A document with more than one tab writes one file per tab. Every writer still
refuses a multi-tab document, so a second tab's file can be merged into a note
and cannot be proposed back.

## Step 2: Document into note

Read the copy and the note, and show three lists:

```
only in the document   the paragraphs the copy has and the note does not
only in the note       the paragraphs the note has and the copy does not
changed on both        the paragraphs that differ
```

Then work through them with the person, section by section, one question each.
Nothing in the note is deleted because the document does not have it: a paragraph
only the note holds is reported and kept. Say that out loud, because it is what
makes the run safe to accept.

Resolve the markers as you merge. `markers.md`, beside `gdoc-export`'s SKILL.md,
holds every marker, what it means, how it is escaped, and how to undo it. Read it
before you touch one. Three of its rules decide questions in this step:

- A pending suggestion is asked about by id and words, never by author, and
  resolving it in the file changes nothing in Drive.
- A suggestion whose id the note's block lists under `proposals` is gdoc's own:
  the note already holds those words, so keep them and ask nothing. Say in one
  line how many were gdoc's own and that they are still waiting in the document.
  That count is `own` in the export envelope.
- A comment anchor loses its marks and keeps its words, always.

When `threads` is more than zero, run `$GDOC comments <url>` and list each thread
by id, author and first line. The comments stay in Drive. A thread carrying `ai?`
or `ai!` is `/gdoc-review`'s work, not this skill's.

Write the note in small edits, each one with a tool that refuses to write a file
that changed since it was read. A live review session may be writing `proposals`
into the same block: on a refusal, read the note again and redo the edit. Nothing
here stops a live session and nothing asks it to stop.

## Step 3: Note into document

```bash
$GDOC read <url>
```

Compose the difference between the note's body and what `read` printed: one entry
per paragraph the document should take. Each entry is the exact words to replace,
the replacement, and the reason in reader language.

**Then stop and ask, once.** Print the list of paragraphs you would change, with
the first words of each, and wait. A merge can send twenty suggestions where a
review sends one, and twenty suggestions in somebody's document is a thing they
did not agree to. One question for the whole list, not one per paragraph.

Before the first `propose`, look in the note's folder for another file whose
block names this same document, and name it in the reply. Two files can hold
proposals for one document, the binary cannot see a folder, and a proposal
recorded in the wrong file cannot be withdrawn from the right one.

A note carrying a marker is refused by `propose` before anything is sent, by
line. Resolve the markers first. That is Step 2 or Step 5, never a deletion of
the marker's words to get past the refusal.

Then, once the person has agreed:

```bash
$GDOC propose <url> \
  --from /tmp/proposals.json \
  --folder 1w0SresizE9Kr810VZRJwX4JtDBF4OqNr \
  --md <note>.md
```

`--md` is the note, and it is what records the proposals so they can be withdrawn
later. Pass it every time. `--folder` is the Drive test folder, where the command
creates a throwaway document to ask whether suggestions are honoured today and
trashes it; the request may name a different one, and then that is the folder.

The rules for the proposals file are `/gdoc-review`'s Step 7 and they hold here
unchanged: `quoted` must appear exactly once, no line breaks in either field, no
empty replacement, the quote must not run across a footnote mark or a picture, no
markdown in `why`, and no 🤖 in `why` because gdoc puts it there. A paragraph the
note lost arrives as a suggested deletion, which is a replace whose words say the
paragraph is proposed for removal: there is no deletion-only shape.

Read `verified` and the three checks per proposal, and say which route did not
hold for which proposal. `sent: false` is a proposal gdoc got no answer for: read
the `error` beside it, and read the document before proposing the same words
again.

`enrolled: false` means Google is not honouring suggestions today. Nothing was
proposed. Report it and stop proposing in this session.

## Step 4: Both moved

Both directions in one run, document into note first. Then:

- Merge every paragraph only one side changed, without asking. One side moved and
  the other did not: there is nothing to decide.
- Show the paragraphs both sides changed. One question each.

Then read the merged whole once more, from beginning to end, and name every place
where the logic broke. A document is coherent or it is not, and a merge that took
one paragraph from each side can leave one that contradicts itself. These are
conflicts too, and the person decides each. What to look for:

- A term defined one way in a paragraph from one side and used the other way in a
  paragraph from the other.
- A number, a date or a count that two paragraphs now disagree about.
- A list that says three things and a sentence that still says two.
- A cross-reference to a heading that the merge renamed or removed.
- An order of steps where one side inserted a step the other side's summary does
  not count.

Show them beside the paragraph conflicts, each one naming the two paragraphs and
what disagrees. Then the note into document direction, Step 3, with its one
question.

## Step 5: Resolve only

A marked file is already in the hub and the request is to finish it. No export.

```bash
$GDOC suggestions <url> --md <note>.md
```

Run it first. It says which suggestions are still pending and which of gdoc's own
have gone, so a suggestion the document accepted or rejected since the export is
resolved from that answer with no question: an accepted insertion's words stay, a
rejected one's go, and say which ones the document decided for you.

Then ask the person about each suggestion still pending, by id and words. Then
take the comment anchors off and keep the words. Then `markers.md` for everything
else in the file.

Nothing in this run writes to Drive.

## Step 6: Delete the copies

Every copy this run made is deleted when the run ends. The copy holds nothing the
document does not, and export is cheap enough to run again.

When the run stops early, the copy stays and the reply names it: a later run finds
it through `exported.note` and Step 0 handles it.

The copy is the only thing this skill ever deletes. Not the note, not another
file in the folder, not anything in Drive.

## Step 7: Two sentences, when they apply

Say each one when it is true, once:

- **A document gdoc did not publish has no house style.** Nothing in this run
  gives it one, and `/gdoc-restyle` is the route: it styles a document where it
  stands and keeps its threads.
- **A new document made from this note carries none of this document's threads.**
  When the person asks for a fresh house-style document instead of suggestions,
  that is `/gdoc-publish`, and the comments stay behind in the old one.

## Step 8: Report

```
direction  both moved
stale      <note>.2.md from 2026-09-18, body matched the fresh export, deleted
exported   <note>.2.md, stamped exported: 2026-09-19T14:02:00Z on <note>.md
merged     6 paragraphs from the document, 2 only in the note kept untouched
markers    4 suggestions resolved with you, 3 were gdoc's own and kept
           2 comment anchors removed, threads listed below
conflicts  1 paragraph both sides changed, resolved with you
logic      1 break: "reporting period" is annual in section 2 and six-monthly in 4
proposed   3 suggestions, 2 verified, 1 NOT verified: docx_anchored failed
copies     <note>.2.md deleted
```

Then the threads, one line each. Then the full text of every proposal's `why`,
exactly as sent, because that is the only place the person sees what is now
visible to everyone on the document.

Say what did not hold: a proposal whose checks came back false, a conflict left as
the note had it, a copy still on disk, a warning from any run. A run that could
not read half of one side must not report clean.

## Never

- Never write the note before the merge has been shown and agreed.
- Never delete what only the note holds. A paragraph the document does not have
  is reported and kept.
- Never decide anyone's pending suggestion. Resolving a marker in a file leaves
  the suggestion pending in Drive, and accepting is the owner's.
- Never edit the document. Every change to its words is a suggestion.
- Never propose before the one question in Step 3 has been answered.
- Never propose from a file that still carries a marker. `propose` refuses it by
  line, and that refusal is the safeguard.
- Never touch the author's own front-matter keys. Adding a missing `title` is the
  one exception, and the `gdoc:` block is gdoc's.
- Never delete anything but the copies this run made.
- Never publish. A fresh document is `/gdoc-publish`, and it carries no threads.
- Never reply in a thread and never withdraw a proposal: those are
  `/gdoc-review`'s.
- Never stop a live review session, and never ask anyone to.
- Never run from an `ai!` comment. This skill starts from a person's sentence.
- Never run git.
- Never trust a status code. Read `verified`, and `checks` where it is there.
- Never export a PDF. You download it from the browser.
