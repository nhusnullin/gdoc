# The presence vote: what Nail kept

2026-10-16, evening. Nail voted on 61 ideas from
[the use-case pass](2026-10-16-presence-use-cases.md) on a page with yes,
no and maybe per row. This file is the result, so the design that follows
builds on what was kept and not on the whole catalogue.

One correction from Nail came first and changes the frame: the `ai?` and
`ai!` markers are his. Colleagues leave plain comments and do not mark them.
The binary has no author check and by the identity invariant never will, but
the presence should be designed for one operator marking his own documents,
with colleagues' comments as a separate, unmarked signal.

## Kept: yes, 16

The icon:

- the mark ⁘ in the menu bar while a live review runs, gone when it ends
- a ring with the step for a finite job (publish, restyle, export, align)
- a green check after a verified write, for five seconds, Nail's number
- red "stuck" that time never clears: a publish stopped, or a read-back
  failed
- amber "waiting for you" with its age, for a hold in Desktop chat or an
  unverified proposal

The menu:

- today's jobs with their result, one line each, with the time
- a sent reply shows the check too, and the history keeps "which comment
  went into which document" (Nail's note on the silent-success row)
- the hold card: rule name and age, "Open chat", "Release without writing",
  the account the reply would post from
- an unanswered hold expires as "not approved, nothing written"
- the health section: account, token, network, gdoc version, skills
  version, Desktop extension, the mark
- an "Update" button in the menu

Getting a document in:

- paste a link with a shortcut
- drag a link onto the icon

## Maybe, 1

The "Live" button beside "Start review", with a note asking for a list of
every file Nail touched through gdoc, each with an action panel. The two
rows that drew exactly that list (`d-list`, `d-actions`) were later voted
no, so the list is an open question for the screens pass.

## No, 39

Everything else: the inbox of marked comments and the counter in the icon,
the return-from-offline card, the documents-in-motion list and drift, the
proposals tracker in all its forms, the thread-side ideas (progress line,
departure line, "no text was changed", the mark explained), the network
red and its wording, the team ideas, quiet mode, privacy mode, mute, the
clipboard watcher, the terminal status line.

Two of the no votes carry notes that read as "did not understand" rather
than "do not want": the proposals ring ("what is accepted where?") and the
grey last-seen state ("at which moment?"). Both were reworded with context
in the second pass and voted no again, so they stand as no.

## Not voted, 5

The job's step list, the facts per step, the stop reason in reader words,
the per-document proposal bar, and the "colleagues wrote" section. The
first three are the click-through of the ring that was kept, so the
screens pass shows them anyway.

## What this means for the design

The presence that survived is small and personal: one icon with five
states, one menu with three sections (today, a hold when there is one,
health), and two ways to hand it a link. Nothing in it is for the colleague
in the document, nothing is for the team, and nothing watches comments on
Nail's behalf. The next pass draws exactly that as screens and asks for
feedback per screen.

## Round two: the screens

The same evening Nail read nine screens of one day drawn from the sixteen
kept ideas and left feedback per screen. What changed:

- **The menu is a log.** "Today" lists everything gdoc did, newest first:
  comment, reply, proposal, publish, export, restyle, withdraw, hold raised,
  hold closed, live started and stopped. Each line says what, in which
  document, on whose request (an `ai?`, a request from Claude Code, a
  Desktop chat) and whether it was verified. Earlier days are one click
  down.
- **Documents gdoc touched.** A section derived from the log, grouped by
  document, with the last action and two buttons: open the document, open
  the note. Nail does not yet know which other actions he wants there.
- **Stop live.** A button on the live row. Stopping is the one destructive
  action in the menu.
- **No hold timer.** Nail asked why six hours. There is no timer: a hold
  lives as long as the Desktop chat that raised it, which is how the binary
  already works, and the log records "hold closed, chat ended, nothing
  written". A timer is an optional setting, off by default, if ever.
- **The link scene is gone.** Not needed for now.
- **Intersections.** The icon shows one thing by priority: red (stuck),
  amber (a hold), the ring (a finite job), the mark (a live watch), nothing.
  The menu's "Now" section shows everything running at once. A green check
  shows for five seconds over any state after a verified write.

Open questions left on the screens: which app opens a note, whether a
hold timer is wanted at all, and what the menu bar shows when nothing runs
(a grey mark, or no icon and the log reachable elsewhere).
