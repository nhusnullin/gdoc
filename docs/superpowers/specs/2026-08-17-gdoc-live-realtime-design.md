# gdoc live: how real time is possible

2026-08-17

## Principles

Serves: principle 1. The recommended mechanism needs no new program, no new
scope, no new Google API, and no infrastructure. It is the existing credential
making the same list call it already makes, in a loop. It travels to any
machine the tool travels to.

Strains: none. The two mechanisms that would strain principle 1 (a webhook
endpoint, a new OAuth scope) are specced below and rejected for exactly that
reason.

Aims at: [the press release](2026-08-17-gdoc-live-press-release.md). Target:
progress line visible within 10 seconds of the comment, 30 acceptable.

## The question

`/gdoc-live <url>` keeps a session attached to a document. Something must
notice a new `ai:` comment and hand it to the agent. How, how fast, and at
what cost?

Two halves: **detection** (how the machine learns a comment exists) and
**dispatch** (how the detected comment wakes the agent). Detection is where
the mechanisms differ. Dispatch is nearly the same for all three.

## Measured facts, live account, 2026-08-16

Probes run against the integration test document with the existing
Commenter service account. Probe scripts were throwaway.

- A fresh comment is visible to `comments.list` in **under one second** of
  its creation. Detection latency is therefore just the poll interval.
- A single `comments.list` poll round-trip is **~0.3s**.
- A new comment **does** appear in the `changes.list` feed (changeType
  `file`), so a change-feed trigger is possible.
- The Drive Activity API returns 403 `ACCESS_TOKEN_SCOPE_INSUFFICIENT` for
  our credential: it needs a scope we do not carry.
- The credential can create, update, and delete **its own** replies and
  top-level comments. A deleted reply leaves no visible tombstone in a
  normal read. It cannot delete anyone else's.

Quota (Drive API docs, checked 2026-08-17): 325,000 quota units per minute
per user, a list call costs 100 units. Polling every 5 seconds is 12 list
calls, 1,200 units, per minute: **0.4% of quota**. Exceeding quota returns
403/429 and asks for exponential backoff.

## Solution 1, recommended: a blocking `gdoc wait` command

Plain polling, packaged so the agent pays nothing while idle.

### Shape

A new CLI command:

```
gdoc wait <url> [--interval 5] [--timeout 570]
```

It polls `comments.list` every `interval` seconds. The moment the actionable
set is non-empty it prints the same JSON report `gdoc read` prints and exits
0. On `timeout` with nothing found it exits 3 with an empty report. The
session loop just runs it again.

The skill runs `wait` as a background Bash command. While `wait` blocks, the
agent is not being invoked at all: zero tokens idle. When `wait` exits, the
harness wakes the agent with the result. Worst-case end-to-end latency is
interval + poll round-trip + one agent wake: **about 6 seconds at the
default interval**, inside the 10-second target.

### Components

| Piece | Where | What changes |
|---|---|---|
| `gdoc wait` | `gdoc/cli.py` + new `gdoc/wait.py` | new command, a loop over the existing `fetch_threads` + filters |
| Turn-based actionability | `gdoc/filters.py` | see below |
| Reply lifecycle | `gdoc/reply.py` | add `update_reply`, `delete_reply` (both verified against the live API) |
| Progress marker | `gdoc/filters.py` + skill | see below |
| `/gdoc-live` skill | `skills/gdoc-live/SKILL.md` | the session loop |

### Turn-based actionability

Today `needs_action` (`gdoc/filters.py:41`) treats a thread as done once it
has any agent reply. That is right for a batch pass and wrong for a
conversation: in a live thread, Nail replies `ai: then fix the note too`
after the agent's answer, and the thread is actionable again.

The live rule: **a thread needs action when its latest `ai:`-marked human
message has no agent answer after it.** Position in the reply list is the
order. The top-level comment sits before all replies. `is_addressed` applies
to replies the same way it applies to the top-level comment.

The batch rule stays untouched for `gdoc read`. The live rule is a second
function beside it, not a change to the first, so `gdoc-review` behaves
exactly as before.

### The progress marker

The progress line is an agent reply, posted early and rewritten in place.
Without care it would count as the agent's answer and close the turn before
the answer exists.

So progress replies carry a fixed prefix, `⏳` followed by a space, and the
turn rule ignores marked replies when deciding whether a turn is answered.
Consequences, all wanted:

- Posting progress does not close the turn. Only the real answer does.
- A crash that leaves a stale progress line behind cannot permanently mark
  a thread as answered.
- On attach, the session deletes any of its own leftover progress replies:
  they are identifiable by author (`me`) plus prefix.

At most one progress reply exists per thread at a time. New progress
rewrites it (`replies.update`); the landed answer deletes it.

### The session loop, in the skill

1. **Attach.** `gdoc read` once: sweep stale progress replies, then handle
   everything already waiting. Catch-up is free.
2. **Wait.** Run `gdoc wait <url> --timeout 570` in the background. Idle
   costs nothing.
3. **Handle.** For each actionable turn: post the progress line, do the
   work with everything the terminal has, rewrite progress as the work
   moves, post the answer, delete the progress line. Long answers: summary
   in the thread, full text to a file beside the source markdown, named in
   the summary. Global items: captured to `pending.md`, as today.
4. **Loop** to 2. Detach is just ending the session; there is nothing to
   clean because progress lines die with their answers.

### Error handling

- 403/429 from Drive: exponential backoff inside `wait`, 5s doubling to
  60s, reset on success. Backoff never raises past the loop.
- Network failure: same backoff. After 5 minutes of continuous failure,
  `wait` exits 1 with a message saying how long it tried, so the agent can
  tell Nail instead of dying silently.
- Document gone or permission revoked: exit 4 with the API's reason. The
  skill reports it and stops the loop.
- A comment `wait` cannot classify is in the report's `skipped` list with
  the reason, per principle 3: reported, not acted on.

### Testing

- Unit: the turn rule (new `ai:` reply reopens; progress reply does not
  close; answer closes; resolved stays closed), the progress-marker filter,
  `wait`'s loop against a fake drive with an injected clock (finds on nth
  poll, times out, backs off on 403, gives up cleanly on persistent
  failure).
- Unit: `update_reply` / `delete_reply` request shapes.
- Integration, credentialed and skipped otherwise: reply update and delete
  against the live test doc, and the narrowed rename of
  `test_deleting_a_comment_is_refused` to say the truth: others' comments
  are refused, its own are allowed.
- Manual, first real session: measure comment-to-progress-line latency
  from the Docs UI, and observe whether rewritten replies show an
  "(edited)" marker or bump the thread. Cosmetic, UI-only, unknowable from
  the API.

## Solution 2, not recommended: two-stage polling via `changes.list`

Poll the `changes.list` feed as a cheap trigger, and call `comments.list`
only when the feed says the file changed. Verified: a new comment does
surface in the feed.

Why not now:

- **It saves nothing for one document.** A `changes.list` call is a list
  call, the same 100 units as polling the comments directly. Same cost,
  same latency, plus page-token state to keep and a second stage to get
  wrong.
- **The feed is noisy.** Any file change bumps it, not just comments, so
  the second-stage call happens on every edit anyway.

When it becomes right: a session watching **many** documents at once. One
feed covers all of them, versus one comment poll per document per tick.
That is a real future, and this spec's `wait` loop is the piece that would
swap out. Nothing else changes, which is why the two-stage design is
recorded rather than discarded.

## Solution 3, not recommended: push via `files.watch`

Google's real push channel. Register a webhook, and Drive POSTs a ping when
the file changes.

Why not:

- **The ping is empty by design.** The payload carries no comment, so the
  poll call happens anyway; push only replaces the trigger.
- **It demands infrastructure per install.** The endpoint must be public
  HTTPS with a CA-signed certificate. That means a server or a tunnel on
  every machine the tool travels to, which is exactly what principle 1
  exists to refuse. It is the LibreOffice mistake in network form.
- **Channels expire and do not renew themselves.** One day maximum on a
  `files` channel, one hour default. Renewal machinery would run forever to
  save five seconds once.
- **The gain is beneath the target.** Push turns ~6 seconds into ~1. The
  target is 10.

Cloud is allowed by Nail's constraint but wanted only as "turn something on
in the Google project". Push is not that: it is standing infrastructure with
a certificate. If a sub-second margin ever becomes the requirement, this is
the door, and it opens without changing the session loop.

## Rejected early

- **Drive Activity API.** Needs a new OAuth scope and an API enablement,
  and after both it is still polling, with no better latency than
  `comments.list` and one more moving part. All cost, no gain.
- **Email notifications.** Comment emails go to the account's inbox and
  parsing them is scraping, fragile by design and slower than the poll.

## Recommendation

Solution 1. It hits the ideal latency target, not just the acceptable one,
using only what is already installed and already permitted. Its one clever
part, the progress marker, exists to protect a correctness rule rather than
to add machinery. Solutions 2 and 3 both reduce to "replace the trigger in
front of the same `comments.list` call", so choosing 1 now forecloses
nothing: either can be swapped in later behind the same `gdoc wait`
interface without touching the skill or the filters.
