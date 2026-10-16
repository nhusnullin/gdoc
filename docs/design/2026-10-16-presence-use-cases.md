# What a menu bar presence could hold, from the sessions that ran

Brainstorm, second pass, 2026-10-16. The first pass
([2026-10-16-presence-brainstorm.md](2026-10-16-presence-brainstorm.md)) asked
eight seats what the printer icon would be for gdoc. This pass reads the five
skills and the sessions Nail actually ran, finds the use cases in them, and
dreams the functions a presence could carry for each. Technology is out of
scope on purpose: Nail asked for the dream first and the mechanism later.

Nail works in Claude Code's desktop app, not in a terminal, so the menu bar is
his surface. That settles the first pass's open question for the first version.

## What the sessions show

Twenty-four sessions in the hub folders called gdoc between 2026-09-08 and
2026-10-16. Counts are gdoc calls in those transcripts, read as commands only,
never as document text.

| Command | Calls | Command | Calls |
|---|---|---|---|
| `comments` | 105 | `auth status` | 10 |
| `reply` | 32 | `restyle` | 10 |
| `read` | 31 | `export` | 8 |
| `help` | 28 | `suggestions` | 6 |
| `propose` | 26 | `withdraw` | 4 |
| `build` | 14 | `annotate` | 3 |
| `publish` | 13 | `update --check` | 1 |

By skill, in turns: review 470, export 140, publish 101, restyle 67.

The scenarios, in Nail's own requests:

1. **One-shot review.** `/gdoc-review <url>`: answer the marked comments once.
2. **Live review.** `/gdoc-review live <url>`, for hours, polling a document
   that is quiet most of the time. The 105 `comments` calls are mostly this.
3. **Review on a flight.** "I was reviewing the doc while offline, left
   comments with marks ai, let's process them and discuss." Google Docs
   queues the comments; the session meets them later, all at once.
4. **The chapter cycle.** Write a chapter in the hub, run the gate reviewer,
   "give me draft in google doc for review", review it in the Doc, "I finished
   review, left as comments in document, incorporate into chapter md,
   recompile, ask to gate review". Then publish again. The Eagle project ran
   this loop from 2026-10-13 to 2026-10-16, chapter by chapter.
5. **Publish with pictures.** "Compile cannot print picture. I can do it in
   Miro myself, insert into MD, and gdoc publish." The SVG to PNG step is the
   one that stops a publish.
6. **Restyle a colleague's document.** Job descriptions and policies brought
   into the house style where they stand.
7. **Keep a document in the hub.** "I wanna keep Group Channel Management
   Policy in 05-rules-and-policies <url>": export, resolve the markers, done.
8. **Annotate from the terminal.** "Use gdoc cli annotations and leave
   comments in <url>, quote ...": a comment on exact words, on somebody's
   document, without opening it.
9. **Triage somebody else's comments.** "Chingwen left comments but most of
   them are for operations": read the unmarked threads, decide which are mine.
10. **Ask a question about a document.** "Explain me what will be the
    exception <url>": a read and an answer, no write.
11. **Align after a review.** "Reflect in md file what I removed, adjusted,
    feedback in <url>": document into note.
12. **Propose from chat.** "Propose changes for condition 3 in google doc":
    a suggestion on one section, from a conversation.
13. **Tooling state.** "Which version gdoc skills do I use", the plugin
    install, ten `auth status` calls. The tool asking about itself.

## Two kinds of work, two shapes of icon

The first pass said a review has no 87 percent. That is true of a live review
and false of everything else gdoc does.

- **Finite jobs** have steps and an end: publish is build, upload, three
  read-backs; restyle is survey, cover, style; export is read, write, pictures;
  align is export, compare, propose. These are print jobs. The ring with the
  step name fits, and the green check at the end means what it meant on the
  printer.
- **Open-ended work** is the live review. States, not progress: watching,
  working, waiting on you, stuck, last seen.

One icon carries both: a ring while a job runs, a state while a watch runs,
and a count for what waits on Nail.

## The functions, dreamed per use case

### 1. The glance: the icon alone

The mark ⁘. A ring while a finite job runs. Amber with an age when something
waits on Nail. A count of marked comments nobody has answered, across every
document, even with no session running. Grey with "last seen" when a watch
died. Absent when nothing is in motion.

### 2. Documents in motion

The menu's first section: every document gdoc has met through a note's
`gdoc:` block or a recent session, one row each. A row says the document's
state in one chip and what waits in it:

- watching since 14:02, quiet 2 h
- 3 marked comments, nobody is answering (no session running)
- 2 of my proposals pending, 1 accepted today
- note moved since publish, 4 days (drift)
- published 2 days ago, v3

Each row opens on click: open the Doc, open the note in the hub, start a
review, go live, stop, align, publish again, copy the link. The chapter cycle
(scenario 4) is one row cycling through these states for a week.

### 3. The inbox: marked comments across documents

Scenario 3 and 9. Comments arrive while no session runs: after a flight,
overnight, from a colleague. The icon counts them, and the menu lists them by
document with the marker: 4 `ai!`, 2 `ai?`, 5 unmarked by others. One button
starts the review in the hub. Unmarked comments are listed as a fact and
never acted on.

### 4. Jobs: the ring with a name

Scenario 4, 5, 6, 7, 11. While a publish runs the ring shows the step: build,
upload, read-back 1 of 3. A stop shows where it stopped and why, in reader
words: "an SVG picture on line 41, render it to PNG first". Today's finished
jobs sit below with their check and their receipt: published, 7 pages, into
Eagle/Drafts.

### 5. Proposals: what happened to what gdoc suggested

The number a review is really measured by: of the suggestions gdoc made, how
many the author accepted, rejected, or left. Per document and across all.
An event when somebody decides: "Vaida accepted 3 of 4 in Chapter 5". Today
nobody knows this without opening the document.

### 6. Holds: waiting for you

A hold in Desktop chat shows amber with its age and the rule name. Click
opens the chat where the card is. No approve anywhere else, which the first
pass settled.

### 7. Drift: the note and the document moved apart

Scenario 11. gdoc knows when a note was published and exported and can see
whether either side changed since. A row shows "note moved since publish" or
"document moved since export", and offers align. The chapter cycle lives on
this signal.

### 8. Health: the tool about itself

Scenario 13. Signed in as which account, token fresh or expiring, network
reachable or the office wifi case named plainly, gdoc version, skills version,
update available with one button. Ten `auth status` calls in a month say this
section is wanted.

### 9. Quick actions: a link in, an action out

Scenario 8, 10, 12. Paste or drop a Docs link on the icon and choose: review,
go live, export, restyle, annotate, ask. A global shortcut opens the same
palette. A copied Docs link lights the icon with "review this?" (the trust
seat would want this off by default).

### 10. The return from offline

Scenario 3. When the machine comes back online and Google syncs the queued
comments, the icon says "7 comments landed from your flight, 4 marked" and
offers to start. The one moment where gdoc knows something before Nail does.

### 11. The colleague feed

Facts about other people's activity, never judged: a new unmarked comment, a
suggestion accepted or rejected, a document shared with the signed-in
account, a reply in a thread gdoc answered. Each with document and age.

### 12. Quiet

Mute one document, mute all until tomorrow, stop every watch, privacy mode
that hides titles on a shared screen. A weekly digest instead of the feed.

## What the first pass's trust seat would still cut

Titles in the menu bar on a shared screen, so privacy mode is a real toggle.
Any approve outside the chat card. Any feed item that judges, "handled",
"important", rather than states a fact. The clipboard watcher, on by default.

## What is next

Nail looks at the canvas and says which of the twelve he would open every
day. Then the mechanism question from the first pass is asked again, with the
answer narrowed to those.
