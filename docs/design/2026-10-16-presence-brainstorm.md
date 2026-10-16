# A presence for gdoc: what the printer icon would be here

Brainstorm, 2026-10-16. Nail's spark: an HP printer puts a small icon in the
Mac menu bar while a job runs, a ring with the number 87, then a green check,
then nothing. One glance says running, how far, ended well. What is that for
gdoc?

Eight seats answered one brief, in parallel and without seeing each other.
This file is their answers joined, the places they agree, the places they
fight, and three shapes the work could take. Nothing here is decided. It is
the brainstorm stage of an L; the spec comes after Nail answers the one
question at the bottom.

## The team

Nail asked for product, business, designer and visioner, and asked who else.
Four more seats were added, and the team itself named three it still lacks.

| Seat | What it watched for |
|---|---|
| Product manager | moments: who is waiting on what, and what one glance changes |
| Business owner, Altery | adoption by non-engineers, trust, the record a regulator reads, the number a manager sees |
| Designer | the grammar of a glance: states versus events, colour, when a thing leaves the screen |
| Visionary | the agent's body language two years out, presence shared by a team |
| Trust guardian (added) | principles 3 and 4, holds, what a presence must never leak or do |
| The colleague (added) | compliance at Altery, no terminal, leaves `ai?` in a Doc, talks to Desktop by voice |
| Platform engineer (added) | the mechanism that carries a presence without breaking "one binary, no daemon" |
| Skeptic (added) | principle 4 as a weapon: which signals survive the attack |

Seats the team said are still missing:

- **The document author** who never typed `ai?`, installed nothing, and is the
  one person who clicks accept or reject. Five of eight seats named her.
- **The Head of Compliance**, who a month later asks "who approved this write,
  on what evidence", and should say what the record must hold before it is
  built.
- **A Windows colleague**, because every route but the tab ring and the status
  file is Mac-shaped.

## Where the eight agree

Seven of eight ended on the same question, in nearly the same words, so it is
the decision and not a detail:

> Who is the glance for: the operator who started the session at a terminal,
> or the colleague in the browser whose document it is? They sit in different
> rooms, and one surface cannot serve both.

The rest of the agreement, in reader terms:

1. **A review has no 87.** A print job is one thing with a known length and a
   yes-or-no end. A review is open-ended, multi-document, and mostly waiting
   on a person. The ring is the wrong shape. The two questions that survive
   are *is it still alive* and *is it waiting on me*.
2. **The printer does not draw its own icon. macOS does.** gdoc should state
   its condition where the desk can find it and let something else draw. The
   binary stays one static file with no daemon. That is the whole mechanism.
3. **Green means one thing.** A write read back `verified: true`. Never "a 200
   came back", never "the review went well", never "the AI did well". The
   binary cannot know the last two.
4. **Stale is unknown, not fine.** A dead process leaves a stale stamp. A
   reader shows "last seen 14:31", never a green dot. Grants die with the
   process; so must the green.
5. **Two moments earn an interruption, and only two.** A hold waiting for a
   person, and a proposal whose read-back failed. An answer posted is the
   normal case, and the normal case is silent.
6. **Nothing in the Doc body, ever.** The page is the readers' room.
7. **No words from documents in the presence.** Shapes, counts, states, rule
   names and ages. Never a title, a quote, a comment, a hub path or an account
   address. A menu bar is on screen in every screen-share. Not a mode, the
   only mode.
8. **No approve button outside the card.** Not in the menu bar, not on a
   phone, not by a spoken yes. A tap on a lock screen is approving without
   reading, which is the habit the hold cards exist to break. The trust
   guardian, the colleague and the designer each refused this one unprompted.
9. **gdoc builds no new channel.** No Slack, no email, no push of its own.
   Google already emails a thread's participants on a reply, and every new
   channel is a new host in the guard. A hold that must reach a phone belongs
   to Claude Desktop's own notifications, with a hold text short enough to
   decide from.
10. **The office wifi failure is a wording bug first.** "TLS handshake
    timeout" reads as a login problem. Five seats asked for one sentence at
    the moment of failure: the network blocked the connection, not your
    sign-in, another network usually fixes it. The platform engineer notes
    the binary carries the fact and the skill speaks the hint.

## The state machine the designer drew

Six states. Four may stay on screen; one lasts two seconds; one has no pixels.

| State | Looks like | May stay |
|---|---|---|
| Absent | nothing | yes, and it is the message when a session ends |
| Watching | the plain mark ⁘, a faint tick per poll | yes |
| Working | the mark with a ring | yes |
| Waiting on you | amber, a pause bar, the age ("since 14:02") | yes, the one state that waits for a person |
| Stuck | red: `verified: false`, or stopped and does not know | yes, until a person clicks "I looked"; time never clears it |
| Done | green check | two seconds, then Watching or Absent |

Two reds, says the trust guardian: "the network refused the handshake" and
"the token was refused" are different pictures, so the wifi stops looking
like a login.

A click shows one card: document, last check, threads waiting, last receipt
and its verdict, the daily update line. Two actions: open the Doc, stop
watching. Nothing else. A hover is one sentence.

## The idea catalogue, by surface

Numbers in brackets say how many seats proposed the idea independently.

### Beside the token: the status file [6]

`comments --wait` rewrites one small JSON beside `update-check.json` on every
poll, through `internal/atomicfile`, and every command writes it at start and
end. Facts only: started, document id, last poll, polls since activity, last
error kind, unverified count, hold pending since, pid. A reader checks the
pid; a dead pid with no end mark is a crash; a stale file resolves to unknown.

Every other glance is a reader of this file, which is why six seats chose it
and three put it first. It costs one rename per poll.

What it touches: principle 2, mildly, with `lastcheck` as the precedent. It
also reverses one pinned sentence, `go/internal/comments/doc.go` line 281,
"the wait writes no file of its own", so it is a DECISIONS.md entry. The
chat ledger's rule that nothing reaches disk stays true if the file carries
counts and instants and never an id, a title or a word of text.

### `gdoc status` [4]

Reads the status files, the stamp and the token expiry, prints one JSON
object, opens no wire. On a terminal, one line and one box through the
Panels. Every drawer below calls this and nothing else, so the formatting
stays out of the binary and the facts stay out of the drawers.

### The terminal, where the operator is

- **Claude Code's status line** [1, the engineer's pick]: a one-line script in
  the hub's `.claude/settings.json` calling `gdoc status`. The pulse sits under
  the prompt with nothing new installed. This is the one route that reaches
  the operator inside a Claude Code session, because of the next point.
- **Tab title and tab ring** [4]: `comments --wait` writes the title escape
  (`gdoc ⁘ live · quiet 2h14 · 1 pending`) and the `OSC 9;4` progress
  sequence that Ghostty, iTerm2 and Windows Terminal draw as a ring in the
  tab, to stderr, only when stderr is a tty. No file, no program. Honest
  limit, from the engineer: under Claude Code stderr is captured, so this
  serves a person running the wait by hand and is Windows's only free route.
- **A terminal notification or bell, twice only** [3]: `OSC 9` / `OSC 777`,
  or the bell that macOS turns into a Dock bounce, on a hold and on a failed
  read-back. Same capture limit.
- **A line rewritten in place while the wait runs** [3]: `listening · last
  event 11:02 ai? answered · tick ok`. Today hours pass with nothing moving,
  so alive and dead look the same.
- **A closing card when live stops** [3]: `3h12 · 4 answered · 2 proposed ·
  1 accepted · 1 pending`, plus the stop reason in reader words: Ctrl-C, token
  expired, handshake timed out, Desktop ceiling.
- **`gdoc help` carries the heartbeat line** beside the update line [2].
  People already type `help`.
- **`--quiet-after 4h`** [1]: a live session that hears nothing for four hours
  ends with the closing card instead of polling into tomorrow.

### The menu bar [5 proposed, 3 would refuse]

A SwiftBar or xbar script of about twenty lines in `release/menubar/`, reading
`gdoc status`. The binary never becomes a menu bar app. Cost: a second
install, macOS only, and it serves the one person who already has the
terminal. The business owner and the skeptic refuse it on exactly that
ground; the designer, the engineer and the visionary keep it as an optional
drawer over the file, which the binary is whole without.

### The Doc thread, where the colleague is

- **One progress line, replaced by the answer** [5]: the 2026-09-19 backlog
  item. "⁘ Seen 14:02, working on it", rewritten in place, deleted when the
  receipt lands, kept when the run dies so the thread never ends on a
  dangling "on it". The PM narrows it to `ai!` tasks past thirty seconds; the
  skeptic keeps only the failure half. It reverses the two-messages decision
  of 2026-08-29 and needs a per-run grant on gdoc's own reply id, shaped like
  withdraw's.
- **Departure line** [4]: "⁘ Stopped watching at 17:02. Nobody is watching
  this document now." Once, in each still-open thread or in the first marked
  one, so the author stops waiting. A crash cannot write it, and the next
  session finds the orphan and says so.
- **Arrival line** [1]: "⁘ watching this document until stopped", so the
  author who never opens a terminal learns someone is listening.
- **The one sentence a compliance reader needs on every receipt** [1, the
  colleague's]: "No text in the document was changed. One suggestion is
  waiting for the author." The binary already knows it.
- **The mark explained once per document** [1]: the first ⁘ reply in a
  document carries one extra line saying what the mark means, for a reader who
  never learned it.
- **Waiting goes to the person through Google's own mail** [2]: a pause
  receipt with an @mention when a hold or a failed read-back waits on someone.
  Google carries it to the phone. No new channel.

### Claude Desktop chat and voice

- **`notifications/progress` during a long call** [1]: Desktop shows the call
  is alive instead of a 200-second hang. Still JSON-RPC on stdout.
- **The hold card names the account** [1]: "This will be posted from your
  Google account, with the ⁘ mark." The colleague wants to know whose name is
  on it before saying yes.
- **A voice hold is a chime, not a spoken comment** [1]: spoken comment text
  is a stranger's words read aloud in the firm's own voice.
- **A hold that nobody answers expires** [2]: after a fixed time it closes as
  "not approved, nothing written" and says so. The person returns to a
  closed question. Principle 3 in time rather than space.

### The hub note and the record

- **`last_review` under the note's `gdoc:` block** [3]: the closing card,
  written once at session end and never per poll, so a glance at the note
  says when gdoc last listened to this document.
- **A review ledger and a regulator page** [1, the business owner's]: per run,
  start, end, answered, proposed, read-backs failed, holds raised and
  confirmed. From it a skill builds the one-pager: every AI write was a
  suggestion, N accepted by a named human, 0 direct edits. "The story I need
  to tell the FCA about AI in a policy document."
- **A weekly manager line** [1]: reviews run, median minutes from `ai?` to
  answer, share of proposals accepted.

### Shared presence

- **The lease** [1, the visionary's fight]: before a live session starts, it
  asks whether another gdoc is already watching this document and fresh, and
  refuses if so. Two robots answering one `ai?` is a real risk today. It is a
  write on a handed-in document's metadata, which the guard refuses at every
  level, and it must expire. A DECISIONS.md entry and Nail's yes.
- **A register of watched documents** [1]: the hub variant of the same idea.
  The visionary refuses this one: machinery in the readers' room that rots
  when a process dies.
- **gdoc in the avatar row** [1]: a named collaborator with a presence cursor,
  "here since 14:02". Probably blocked; measure it and write the row in
  BLOCKED-BY-API.md.
- **A convention, not a feature** [1]: in two years every agent on the team
  publishes the same three-state presence, and the stamp format is the seed.

### Wording fixes that need no design [5]

- The handshake timeout names the network, not the sign-in.
- The stop reason of a wait, in reader words, in the one JSON object.
- The guide's rule for the colleague: no acknowledgment within a minute means
  no session is running, here is how to start one.

## Where they fight

- **Menu bar.** Three refuse it, three keep it as an optional drawer. Nobody
  wants the binary to be it.
- **The thread progress line.** Five want it; the trust guardian says the mark
  in front of a reply is all the presence a Doc needs, and the skeptic keeps
  only the failure half. All agree it is a reversal of a decision, not a
  widening.
- **Arrival and departure lines.** The business owner fights for both; others
  take departure only. Both add replies to the readers' room.
- **What the quiet means.** The skeptic's question: after three hours of
  quiet, do you want to know it is alive, or that nothing happened? The
  second needs no tool at all.

## Three shapes, and a recommendation

**A. The operator's pulse** (recommended first)
The status file, `gdoc status`, the Claude Code status line script, the
closing card with the stop reason, and the two wording fixes. Optional:
a SwiftBar script in `release/`. It answers "is it alive" and "is it waiting
on me" for the person at the terminal, keeps the binary one file with no
daemon, and reverses one pinned sentence in `comments/doc.go`. It is the
cheapest thing on the list and every other glance is built on it.

**B. The colleague's thread**
The in-place progress line, the departure line, the "no text was changed"
sentence, the mark explained once. It reaches the one person gdoc reaches
nowhere today, and it is the only shape the author of the document can see.
It reverses the two-messages decision, adds `replies.update` and
`replies.delete` on gdoc's own id under a per-run grant, and spends the
readers' room. An L of its own; the backlog item is its first page.

**C. The record**
`last_review` in the `gdoc:` block, the ledger, the regulator page, the
manager line. Mostly skill work over facts the binary already prints. It
serves the business owner's audience and nobody at a glance.

Recommendation: A first, because it is small, it unblocks every drawer, and it
settles the mechanism question before anyone argues about icons. Then the
question below decides whether B or C is next. The two wording fixes are S
tasks and could land this week on their own.

## The one question for Nail

Who is the glance for: you at the terminal, or the colleague in the Doc?

If it is you, shape A is the feature and B is a later courtesy. If it is the
colleague, B is the feature and A is plumbing. The team could not decide this
for you, and said so seven times.

Two smaller questions came out of the night and want an answer eventually:

- Should an unanswered hold expire, so a person returning in the morning
  cannot release a write into a thread that has moved on?
- Is a per-process file in `~/.config/gdoc-agent/` the agreed place for
  session facts, or does it count as something hidden beside the hub?

## What the printer gets wrong, and the one thing it gets right

From the visionary: a print job ends, has a percent, waits on nobody but
paper, and shows its light only on the machine that sent it. A review never
ends, has no percent, mostly waits on a person, and its audience is on other
machines. The one thing the printer gets right: it does not draw its own
icon. It states its condition and lets the desk show it.
