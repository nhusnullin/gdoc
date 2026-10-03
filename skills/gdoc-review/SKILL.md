---
name: gdoc-review
description: Use when the request gives a Google Doc link and asks for the marked comments in it to be handled, once or live. Reads the threads, answers ai? in the document, carries out ai! in the hub, and proposes document changes as native suggestions. The word live keeps the session watching that one document until it is stopped.
compatibility: Requires the gdoc binary on PATH, signed in with gdoc auth login, and network access to Google Docs and Drive.
metadata:
  needs: v2.8.0
---

# Google Docs review

This is `gdoc-review`. Say that in the first line of your reply, because several
skills answer a request about a document, and this is the one that works through
its comments.

Answer the marked comments in a document, whoever wrote them. You chose the
document, and the marker is the instruction.

`ai?` is answered in its thread. `ai!` is carried out in the hub and receipted
in its thread. `ai:` leaves the choice to the session, which says which one it
chose. An unmarked comment is never acted on.

When the right answer is a change to the document's own words, propose it as a
Google suggestion with a comment explaining why. Never edit the document.

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
its own front matter names under `metadata` on its `needs` line, or a later
one. An older binary is one the skill is ahead of. Say so in one short
sentence that names the version installed and the version this skill needs,
give the command as the paragraph after next says, and stop there. No
`version` at all is a build made from source rather than a release, which is
not an error: say it once and carry on.

The same object may also carry `update`, which says what is installed and what
is published: `installed`, `latest_stable`, `latest_nightly` and `checked_at`.
When `latest_stable` is there and its three numbers are ahead of `version`,
say once, in one short sentence, which gdoc is published and which is
installed. Give the command as the next paragraph says, then carry on with the
work. This is a remark and never a gate: `needs` is the only version that
stops a session. No `update` key at all is a build from a checkout, and the
skill says nothing about it.

The command goes in its own fenced code block tagged `bash`, after the
sentence, so the person can run it with one click. The block holds one line
and nothing else: no `$` prompt, no comment, no second command. That line is
`gdoc update`, or `gdoc update --major` when the first of the three numbers is
the one that is ahead, because a major release is one a person asks for by
name. The block is for the person to run. The session never runs it.

One root, `$ROOT`, and it is `$PWD`. It is the hub: the corpus this session
searches to ground an answer, the tree that holds the notes paired to
documents, and the place an `ai!` writes.

Nothing here runs git. Not to commit, not to check whether a file is dirty.

There is no local record of which comments were handled. The 🤖 reply in the
thread is the receipt, and the document is the ledger. Every run recomputes the
open work from the threads alone.

A live session holds one cursor, in this conversation and nowhere else. It dies
with the session. It says where the last window ended, never what was handled,
so the rule above is unchanged: the thread is still the only receipt.

## If the credential is not working

Any command can fail with no token or a permission error. Run `$GDOC auth
status` and read it before guessing. It reports whether a token is present,
whether it expired, and which scopes are missing.

The fix is `$GDOC auth login`. It prints a URL, waits for you to approve in the
browser, and saves the token. Ask before running it: it changes which account
posts replies, and that account name is visible to everyone on the document.

Never edit `~/.config/gdoc-agent/` by hand, and never tell anyone to.

## If the request is a dry run

Do every step, but write nothing. Print each reply and each proposal in the
terminal instead of running `$GDOC reply` and `$GDOC propose`. Say at the end
that nothing was posted and nothing was proposed.

## Step 1: Read the document

```bash
$GDOC comments <url>
$GDOC read <url>
```

`comments` gives the threads. Each one carries `id`, `author`, `created`,
`modified`, `content`, `marker` (`ai:`, `ai?`, `ai!` or `none`), `resolved`,
`quoted`, `range`, and `replies`. Each reply carries `id`, `author`, `created`,
`content`, `marker`, read by the same rule as a thread's, and `by_gdoc`, which
is true when it opens with 🤖. Never narrow a listing on thread markers alone: a
marked reply is an instruction too.

The envelope carries `cursor` beside the threads: the position of the newest
activity this read saw, or, on a document nobody has commented on, a position
dated from the run's own clock so that live mode has somewhere to start. Every
listing carries one. A one-shot run has no use for it. Live mode does, so keep
it.

`read` gives the document's text, with `[[c:ID]]words[[/c]]` around the span a
comment is attached to and `{+text+}[s:ID]` around a pending suggestion. That is
how a comment is seen in the words around it. Every read takes in every tab, so
a document with more than one is read whole and reported as what it is.

`multi_tab: true` stops the run for writing. The session may still read and
answer, but `propose` refuses a multi-tab document, so say so instead of
proposing.

Add `--witness` to `comments` when it matters whether a thread is still attached
to text. It costs a docx export, and it reports `anchored`, `detached` or
`unmatched` per thread. The export is the honest witness here: Drive's own
anchor and its copy of the quoted text survive a detachment and prove nothing on
their own.

## Step 2: Decide which threads still need an answer

Before the first thread, read `review.md`, beside this file. It holds the rules
this run judges by: which threads are still work, the check before acting on an
old marked comment again, the two messages a thread gets, when to stop and show
a draft instead of posting it, what all-comments mode changes, and the list of
what this skill never does. It names no command, so read it once and keep it.

That check reads the two records that survive a run cut off between an action
and its receipt. One of them is the document's own pending suggestions:

```bash
$GDOC suggestions <url>
```

The other is the paired note's `gdoc:` front matter. `review.md` says how to
read the two together, and what a missing id means.

## Step 3: Say what the run found

Print the list before acting. A marker is your instruction, so acting on it
needs no confirmation, and this is a statement rather than a question.

```
Will answer:      "reviewed annually" (William Mejia, ai?), "term is wrong" (Priya Nair, ai:)
Will carry out:   "add this to the decision log" (William Mejia, ai!)
Already answered: 2 threads
Marked replies:   1 answered thread has an ai! reply since my last one, carried above
Newer replies:    1 answered thread has an unmarked reply since my last one
Pending:          3 suggestions in the document, 1 of them mine

Replies are visible to everyone with access to this document, and I cannot
see who that is. They post under the account gdoc is signed in as.
```

Report the marked and unmarked follow-ups every run, even when there are
none. "Nothing to do" and "somebody wrote something I may not act on" are
different answers, and the second one loses instructions.

## Step 4: Ground the answer

Search `$ROOT`. Cite plain file paths. The corpus mixes Russian and English, so
search in both.

If the hub has no source for the answer, say so in the reply. Never write a
plausible sentence to fill the gap.

## Step 5: Answer an `ai?`

Plain text only. Docs threads render markdown as typed, so `**`, backticks and
`#` headings arrive broken. The binary refuses them, and that is a safeguard
rather than permission to try.

Every reply opens with `🤖 ` and nothing before it. The binary refuses a body
that does not.

Shape:

1. The answer or the replacement text first, so the first thing the reader sees
   is the thing they copy.
2. One short reason line, only when the reason is not obvious.
3. Sources as plain paths, last.

```bash
cat > /tmp/reply.txt <<'EOF'
🤖 <the reply>
EOF
$GDOC reply <url> <comment_id> --body-file /tmp/reply.txt
```

Read `verified` in the answer. It is the thread read back after the write, and
`commentUpdateState: ALL_SAVED` in what the write itself answered, not the
status code. `verified: false` means the reply may not be there: say so in
the terminal and read the thread again before posting a second time.

## Step 6: Carry out an `ai!`

The work happens in the hub, in this session. Edit the note, the decision log,
whatever the comment names. Nothing is queued, and there is no `pending.md`.

Then receipt it in the thread, once, after the work is done and saved:

```
🤖 Done. Added to domains/regulatory/decisions.md, the CBC section.
```

The receipt names where the work landed, in reader language. A file path is
fine when the readers are your own team and the document is internal. On a
document shared outside Altery, say what changed without naming internal
files.

Never delete a file from the hub. Editing is the whole of what `ai!` may do.

### An `ai!` that asks for an export

`ai! export this to the hub`, or any comment asking for the document to be
brought into the hub, is not work this skill carries out. An export writes a new
file in the hub and its markers are then resolved with a person, question by
question, and a comment thread is not where that conversation happens.

Answer it in the thread, once, and act on nothing:

```
🤖 An export starts from a session by name, never from a comment. Ask a session:
"Bring this into the hub", with this document's link.
```

The same holds for a comment asking to publish the note, to align it with this
document, or to restyle this document. Each of those is a skill a person starts:
name the request in plain words in the reply, and do nothing else.

### Name a colleague who asked

When the marked comment was written by somebody other than the account gdoc is
signed in as, the receipt names them:

```
🤖 Done, asked by William Mejia. Added to domains/regulatory/decisions.md.
```

The document already shows who wrote the comment. Saying it in the receipt puts
it where the delegation happened, so a reader scrolling the margin sees that the
work came from William and not from whoever gdoc posts as.

The signed-in name is the `author` on gdoc's own replies in this document, the
ones with `by_gdoc: true`. Read it there first: it is a fact in the object, not
a guess.

When the document carries no such reply yet, the account is the operator's, the
person who asked for this run in this session. So name a commenter only when
the comment was written by somebody other than them, and when this session
cannot tell the two apart, name nobody. A receipt that says nothing about who
asked is thin; a receipt that credits the operator with asking themselves is
the one mistake this rule must not make, and silence is the safe side of it.

Identity is still never a gate. The marker decides whether a comment is work,
and this changes the wording of a receipt and nothing else.

## Step 7: Propose a change to the document

When the right answer is different words in the document, propose them. A
suggestion is reversible and gdoc can withdraw its own, so this does not need
asking first.

When the right answer is a new or rewritten section rather than a phrase,
propose a block: whole new paragraphs, placed after a paragraph you quote or in
place of a run of paragraphs. Same file, same comment, same read-backs. A block
that replaces paragraphs is the one proposal that takes somebody's words out, so
read `$GDOC suggestions <url>` and `$GDOC comments <url>` first, say what falls
inside the run you would replace, and propose only once the person has answered.
gdoc refuses a replace over a pending suggestion or a comment's anchor anyway,
and names every id, but that refusal reaches you and never reaches them.

Before the first proposal of the session, read `propose.md`, beside this file.
It holds the proposals file, the shapes of quote the command refuses and why,
the reason line, and how to read `sent`, `verified` and `checks`. It also holds
how to withdraw a proposal that was wrong.

The short version, for when you are deciding whether to propose at all: quote
the document's own words, exactly once, and inside one paragraph for a words
proposal; give the replacement, or the block's markdown content, and a
plain-text reason for the reader; pass the paired note so gdoc remembers the
suggestion is its own. Never report a proposal as landed because the command
exited 0.

## Step 8: Report

```
answered   "reviewed annually"      cited domains/regulatory/cbc-emi.md
carried    "add to the decision log" edited domains/regulatory/decisions.md
proposed   "reviewed annually" -> "reviewed every six months"   verified
proposed   "quarterly" -> "monthly"   NOT verified: docx_anchored failed
proposed   block of 4 paragraphs after "the six month cycle"   verified

files changed in the hub: domains/regulatory/decisions.md, policy.md
1 marked reply since my last answer, on the thread about vocabulary, carried above
1 unmarked reply since my last answer, on the thread about scope
```

Then the full text of every reply and every comment body, exactly as sent, with
the thread it went to. Not a summary. Nothing else in the run shows you what is
now visible to everyone on the document.

```
--- posted to "reviewed annually" ---
🤖 <the reply, exactly as sent>

--- comment on the proposal "reviewed every six months" ---
🤖 <the body, exactly as sent>
```

Say what was cut from a draft and why, which threads the session stopped on and
is still waiting on, and which proposals did not verify.

If half the review could not be read, say so and do not report clean. A
`comments` run with warnings, a thread whose range came back null, a suggestions
read that failed: each one means part of the review was invisible on this run.

## Live mode

When the request says "live", run everything above once, print its Step 3
list and Step 8 report as always, then read `live.md`, beside this file. It
holds the watch loop on that one document, the Bash timeout it needs, the four
answers a wait can come back with, and the totals printed when you stop it.
