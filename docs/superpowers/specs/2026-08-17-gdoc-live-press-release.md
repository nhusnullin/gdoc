# gdoc live: press release

2026-08-17

## Principles

Serves: principle 3. The agent deletes only what it wrote itself, the `ai:`
marker is the whole gate, and a comment it cannot read confidently is reported,
not acted on. Also principle 1: it needs nothing a machine cannot already
install.

Strains: none today. The OAuth idea, if it lands, would turn the "cannot edit
is a permission, not a policy" claim into a policy, which means arguing with
the 2026-08-13 Commenter-only decision before that page stays honest.

This is the working-backwards page: the result, written before the how. The
design spec comes later and must aim at this.

---

# gdoc live

**Make the document live, and the margin answers.**

Open a session on a Google Doc with `/gdoc-live <url>`. From then on, every
comment you mark `ai:` is a prompt, and the answer arrives in that thread while
you are still reading the paragraph it belongs to. Behind it is everything your
terminal has: the hub, the skills, the ability to test four readings at once
and come back with the one that survived.

## The problem it removes

Reviewing a document today is a round trip you make by hand.

You read in the doc. You think in the terminal. Between them you carry the
context yourself: paste the paragraph, explain what you are pointing at, read
the answer somewhere the document will never see, then type the conclusion
back in.

Every carry costs. And most of the thinking does not survive the trip. The
scrollback dies. The paragraph stays, and nobody can tell why it says what it
says.

The document and the thinking were always in two places, and only one of them
was kept.

## One session

You are reading a draft post on the four-day work week. Paragraph four cites a
trial's productivity number. You are not sure it is right.

You select the sentence and write: `ai: is this number right, and where does
it come from`

Three seconds later there is a line in the thread. *Finding the trial,
checking your notes.*

You keep reading. When you glance back, that same line has become *found the
trial report, your notes cite the older interim figure, confirming.* It did
not add a second line. It rewrote the first one. There is never more than one
of these, and it always says where the work is now, not where it has been.

Then the answer lands, and the progress line is gone. The thread reads: your
question, then the answer, with the final report it rests on and the stale
note named by file and line. Ninety seconds of working noise deleted itself.

You reply in the same thread: `ai: then fix the note too`. It goes.

Further down there is a claim that could be read three ways. `ai: run all
three readings of this and tell me which survives.` Three agents go out at
once. What comes back is one recommendation and the two it killed, with why.

The last question is a big one, and the answer to it is four thousand words.
The margin does not get four thousand words. The thread gets what you can read
where you are standing: what was found, what it means, what to do. The full
version is written to a file beside your markdown, and the thread tells you
its name. The margin holds the thinking. Your machine holds the artifact,
waiting for when you sit down.

Through all of it, you never opened a terminal.

When you are done, `gdoc-apply` takes what you captured into the paired
markdown and generates the next version of the document. That part already
works, and none of it changes.

## What it cannot do

It cannot touch the document. The credential is Commenter, and that is not a
policy, it is a permission. Every word it produces is comment surface, beside
the text and never inside it. The body changes in exactly one place: the
markdown you own, through apply, on purpose.

It can delete only what it wrote itself. Your comments are yours. Its own
progress notes are its to clean up, and that is the whole reason the margin
stays clean.

It answers `ai:` and nothing else. A comment it cannot read confidently is
reported, not acted on.

It does not need git, and it needs nothing your machine cannot already
install.

## Why the margin

The question is anchored. You ask about this sentence, and it knows which one,
because you selected it.

You can be away from the machine. Ask from a phone in a meeting, read the
answer there. The long research waits at your desk, which is where you were
going to read it anyway.

The thinking stays. The thread sits next to the text it produced, so the next
person to read that paragraph can read why it says what it says.

And you never leave reading mode. You do not switch out of the headspace of
prose to go drive a tool. The tool comes to you.

## Who it is for

One person and one document, thinking hard.

The document does not have to be yours. Anything shared with you can be made
live: a colleague's draft, a doc gdoc generated last week, a page someone
pasted together by hand. Whoever wrote it, the margin works the same.

Not a shared room, not a team chat. The agent answers you and nobody else.

Every tool before this one made you come to it. You left the page, opened the
terminal, explained where you had been, and carried the answer back by hand.

Make the document live, and that trip is gone. You ask about a sentence by
standing on it. You never describe what you are pointing at, because you are
pointing at it.

---

## Decided during the press release, binding on the spec

- The command is `gdoc live`, the skill `/gdoc-live <url>`. "The margin" is
  the page's word, not the command's.
- Solo: one voice talks to the agent. The document can be anyone's.
- Progress comments: at most one live line per thread, rewritten in place,
  deleted when the answer lands. A progress note that turns out to be a
  finding is promoted into the answer, not deleted.
- Long answers: executive summary in the thread, full version in a local
  markdown file beside the source, named in the thread.
- Verified 2026-08-16 against the live account: the Commenter service account
  can create, update, and delete its own replies and its own top-level
  comments. A deleted reply leaves no visible tombstone in a normal read.
  It still cannot delete anyone else's comment.
  `tests/test_access_integration.py::test_deleting_a_comment_is_refused`
  should be renamed to say the narrower truth.

## Open, deliberately not answered here

- How fast is real time. Drive has no push for comments, so something polls.
  The number sets the feel and the cost.
- Whether live-session comment threads should also be captured somewhere
  outside the document (for apply, or for context in later sessions).
- OAuth as an alternative to the service account. Touches the Commenter-only
  decision in PRINCIPLES.md, argue there first.
- Whether the Docs UI shows an "(edited)" marker on a rewritten reply, and
  whether a rewrite bumps the thread in the margin. Cosmetic, UI-only,
  check with a browser during implementation.
