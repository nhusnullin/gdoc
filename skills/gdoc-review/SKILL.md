---
name: gdoc-review
description: Use when Nail gives a Google Doc link and wants the marked comments in it handled, once or live. Reads the threads, answers ai? in the document, carries out ai! in the hub, and proposes document changes as native suggestions. The word live keeps the session watching that one document until Nail stops it.
---

# Google Docs review

Answer the marked comments in a document, whoever wrote them. Nail chose the
document, and the marker is the instruction.

`ai?` is answered in its thread. `ai!` is carried out in the hub and receipted
in its thread. `ai:` leaves the choice to you, and you say which one you chose.
An unmarked comment is never acted on.

When the right answer is a change to the document's own words, you propose it as
a Google suggestion with a comment explaining why. You never edit the document.

Spec: `~/src/personal/gdoc/docs/v2/SPEC.md`, sections "Skills", "How a comment
reaches the agent", "Verification: never trust a success" and "Never".

## Setup

```bash
GDOC=gdoc
ROOT="$PWD"
```

`gdoc` is the v2 binary, on PATH. It prints exactly one JSON object and exits.
Exit 0 means the object says `ok`. Read the object, never the exit code alone.

One root, `$ROOT`, and it is `$PWD`. It is the hub: the corpus you search to
ground an answer, the tree that holds the notes paired to documents, and the
place an `ai!` writes.

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

The fix is `$GDOC auth login`. It prints a URL, waits for Nail to approve in the
browser, and saves the token. Ask before running it: it changes which account
posts replies, and that account name is visible to everyone on the document.

Never edit `~/.config/gdoc-agent/` by hand, and never tell Nail to.

## If Nail asks for a dry run

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
`content` and `by_gdoc`, which is true when it opens with 🤖.

The envelope carries `cursor` beside the threads: the position of the newest
activity this read saw. A one-shot run has no use for it. Live mode does, so
keep it.

`read` gives the document's text, with `[[c:ID]]words[[/c]]` around the span a
comment is attached to and `{+text+}[s:ID]` around a pending suggestion. That is
how you see a comment in the words around it.

`multi_tab: true` stops the run for writing. You may still read and answer, but
`propose` refuses a multi-tab document, so say so instead of proposing.

Add `--witness` to `comments` when it matters whether a thread is still attached
to text. It costs a docx export, and it reports `anchored`, `detached` or
`unmatched` per thread.

## Step 2: Decide which threads still need an answer

The binary reports facts and judges nothing. There is no `handled` field, on
purpose. You read the thread and decide.

Read every thread from the top, and ask what the last turn is:

- The last turn is a 🤖 reply that answers the marked comment: done. Leave it.
- A marked comment or reply was written after gdoc's last 🤖 reply: that is new
  work, and it is the part to act on. The rest of the thread is context.
- The last 🤖 reply is an acknowledgment and no receipt followed it: the action
  may have landed without its receipt. Check before acting again, below.
- An unmarked reply was written after gdoc's last one: report it, act on
  nothing. An unmarked follow-up carries context, never authority.
- The thread has no marker anywhere: not work.

A resolved thread is not work. Nail resolves a thread when he accepts the
answer.

### Before acting on an old marked comment again

A run can be cut off between an action and its receipt. So a marked comment with
no receipt does not prove the work is undone.

Check the two records that survive:

```bash
$GDOC suggestions <url>
```

and the paired note's `gdoc:` front matter, whose `proposals` list holds the
suggestions gdoc wrote and could record, with the words each one replaced. When
the work is already there, write the missing receipt. Do not do it twice.

The list is not a complete record on its own. A proposal gdoc could not record
is in the document and not in the note, and the run warns `gdoc cannot withdraw
it later`. The warning names which id is missing and which route it comes from:
the comment id is the write's own answer, and the suggestion id is read out of
the read-back afterwards, so `the read-back could not confirm a suggestion id`
is the re-read failing over a write that came back perfectly well. Do not report
it as a write that failed. So `suggestions` is the document's own answer and the
note is gdoc's memory of it: read both.

## Step 3: Say what you found

Print the list before you act. A marker is Nail's instruction, so acting on it
needs no confirmation, and this is a statement rather than a question.

```
Will answer:      "reviewed annually" (Nail, ai?), "term is wrong" (William Mejia, ai:)
Will carry out:   "add this to the decision log" (Nail, ai!)
Already answered: 2 threads
Newer replies:    1 answered thread has an unmarked reply since my last one
Pending:          3 suggestions in the document, 1 of them mine

Replies are visible to everyone with access to this document, and I cannot
see who that is. They post under the account gdoc is signed in as.
```

Report the unmarked follow-ups every run, even when there are none. "Nothing to
do" and "he wrote something I may not act on" are different answers, and the
second one loses instructions.

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

1. The answer or the replacement text first, so the first thing Nail reads is
   the thing he copies.
2. One short reason line, only when the reason is not obvious.
3. Sources as plain paths, last.

```bash
cat > /tmp/reply.txt <<'EOF'
🤖 <the reply>
EOF
$GDOC reply <url> <comment_id> --body-file /tmp/reply.txt
```

Read `verified` in the answer. It is the thread read back after the write, not
the status code. `verified: false` means the reply may not be there: say so in
the terminal and read the thread again before posting a second time.

## Step 6: Carry out an `ai!`

The work happens in the hub, in this session. Edit the note, the decision log,
whatever the comment names. Nothing is queued, and there is no `pending.md`.

Then receipt it in the thread, once, after the work is done and saved:

```
🤖 Done. Added to domains/regulatory/decisions.md, the CBC section.
```

The receipt names where the work landed, in reader language. A file path is
fine when the reader is Nail and the document is internal. On a document shared
outside Altery, say what changed without naming internal files.

Never delete a file from the hub. Editing is the whole of what `ai!` may do.

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
ones with `by_gdoc: true`. When the document carries none yet, treat Nail as the
account: it is his login, and naming him in his own receipt is the one mistake
this rule must not make. Nail's own comments are never named.

Identity is still never a gate. The marker decides whether a comment is work,
and this changes the wording of a receipt and nothing else.

## Step 7: Propose a change to the document

When the right answer is different words in the document, propose them. A
suggestion is reversible and gdoc can withdraw its own, so this does not need
asking first.

Write the proposals file. Each entry is the exact words to replace, the
replacement, and the reason in reader language:

```bash
cat > /tmp/proposals.json <<'EOF'
[
  {
    "quoted": "reviewed annually",
    "replacement": "reviewed every six months",
    "why": "The CBC letter of 12 August asks for a six month cycle."
  }
]
EOF
$GDOC propose <url> \
  --from /tmp/proposals.json \
  --folder 1w0SresizE9Kr810VZRJwX4JtDBF4OqNr \
  --md <paired note>
```

`quoted` must appear exactly once in the document. The command refuses none and
refuses more than one, and the fix is to quote more of the sentence. Never work
out a character index yourself: the command finds the words in a fresh read.

The quote must also not run across a footnote mark, a picture, an equation or a
page break. `read` prints those as `[^1]`, `[image]` and so on, and a quote built
by dropping the marker out of a sentence is exactly the shape that crosses one.
The command refuses it and says so: quote a shorter run of words on one side.
Words that stop right before the marker are fine, and so are words that start
right after it. The count is against the whole document, so a quote that reads
once as plain text and once across a marker is refused as ambiguous too.

Quote the body only. `read` prints each footnote's own text under the rule at
the end, as `[^1]: ...`, and a proposal cannot be placed there: `propose` looks
in the body, so those words come back as not found even though they are on
screen. To change a footnote, say so in a reply instead.

Neither `quoted` nor `replacement` may carry a line break, and `replacement` may
not be empty. A proposal replaces words with words inside one paragraph: there
is no deletion-only shape, none that adds a paragraph, and none that removes
one. A paragraph's text ends with its own break, so a quote copied out of `read`
with the trailing newline still on it would take the paragraph mark with it and
merge two paragraphs. Quote the words, not the line.

`why` becomes the body of a comment anchored on the new words, opening with 🤖.
gdoc adds that prefix, so do not write it yourself: a reason that already opens
with it is refused, because the comment would arrive signed twice and every
read-back would still pass. This is the opposite of `--body-file` for a reply,
where the prefix is yours to write and a body missing it is refused. A leading
space or newline in front of the robot does not get around the refusal, and
neither does the robot with no space after it: all three land the same doubled
mark. A reason that is only whitespace is refused too, because the comment would
be a bare signature. The reason
is what the reader sees, so write it for the reader, in plain text. Markdown is
refused before anything is sent, because a Docs thread renders asterisks and
backticks as typed.

When the comment that asked for the change was written by somebody other than
the signed-in account, `why` names them the same way the receipt does: `Asked by
William Mejia. The CBC letter of 12 August asks for a six month cycle.` The rule
and its one exception are in Step 6.

Every proposal in the file is checked before the first one is written, so a bad
entry stops the run with nothing sent. Once writing starts the run stops at the
first proposal it cannot place, and the report still carries one entry per
proposal in the file: read `sent` on each.

`sent: false` means gdoc got no answer saying the proposal landed. A guard
refusal and a 4xx never changed the document. A transport failure or a 5xx is
the third case, and gdoc cannot tell it apart: the request was written and may
have been applied. The envelope's `error` names which one it was, so read it
beside the flag, and read the document before proposing the same words again.

`--folder` is the Drive test folder. The command creates a throwaway document
there on every run, asks Google whether suggestions are honoured today, and
trashes it. `enrolled: false` means nothing was proposed, and the reason is
Google rather than the document: report it and stop proposing in that session.

`--md` is the paired note. It records which suggestions are gdoc's own, and that
record is the only permission to withdraw one later. Pass it whenever the
document is paired. Without it the suggestion still lands and gdoc forgets it
wrote it.

Read `verified` and `checks` per proposal:

- `suggestions_inline`: the new words are in the document as a suggestion.
- `preview_without_suggestions`: the old words are still what a reader sees, so
  it is a suggestion and not an edit.
- `docx_anchored`: the 🤖 comment is attached to the new words.

`verified: true` is all three checks and a write that answered
`commentUpdateState: ALL_SAVED`. Anything less means the write happened and
something could not confirm it. Tell Nail in the terminal what did not hold for
which proposal: the route whose check is false, or, when all three are true, the
warning saying Docs took the batch and its answer could not be read. Never
report a proposal as landed because the command exited 0.

### Withdrawing a proposal

When a proposal was wrong, and it is still pending:

```bash
$GDOC withdraw <url> <suggestion_id> --md <paired note>
```

It works only on suggestions the note records as gdoc's own. Then reply in the
🤖 comment's thread saying the proposal was withdrawn and why. The comment stays
where it is: gdoc does not delete comments.

Never accept, reject or delete a suggestion anyone else wrote, and never accept
your own. Accepting is Nail's.

## Two messages at most

A thread carries an acknowledgment and a receipt per piece of work, and nothing
else. A thread you answered can ask again: a marked comment or reply written
after gdoc's last 🤖 reply is new work, and it gets its own pair. Live mode
makes that ordinary rather than rare, because the session sees the second
question arrive.

- Write the acknowledgment only when the work will take noticeably long. One
  line, reader language: `🤖 Looking into this now.`
- Write the receipt when the work verifiably landed.
- A failure replaces the receipt with a plain sentence naming the reason.

Never a progress feed. The margin is the readers' room, and the terminal is
where the operator's record goes. An acknowledgment and a receipt are two
replies, never an edit of one.

## When you stop and ask

You judge the draft before it is posted. Stop, show it, and wait when any of
these is true:

- **It carries something out of the hub that this document should not.** An
  internal figure, an unpublished decision, the name of an internal document, a
  counterparty's terms. This happened on 18 August: a draft carried a figure
  from the hub thesis into a document shared with a counterparty. Nobody outside
  Altery should learn something from a gdoc reply that they could not learn from
  the document.
- **A rule in the root says it is not shared.** A note marked confidential or
  internal, or anything the surrounding documents treat that way.
- **You are not confident the answer is true.** Not "I have no source", which
  Step 4 covers by saying so in the reply, but "I think this is right and I could
  be wrong".
- **The comment is genuinely ambiguous.**
- **It would be a second reply to a thread you already answered.**

Say which of those it is, show the draft, and wait. When you post but cut
something out of the draft first, post it and say what you cut and why.

Everything else goes out.

## If Nail asks for all the comments

By default only marked comments are work. When Nail says he wants to go through
every comment, print the numbered list of unresolved threads with author, quote
and content, answered threads last and labelled, and let him pick.

For each one he picks: draft the reply, show it, wait, then post. An unmarked
comment carries no instruction, so it is classified by you and he approves it.
Never batch-approve in this mode.

Picking an answered thread is allowed. It is the one case where you reply twice
to the same thread, and you say so before posting.

## Step 8: Report

```
answered   "reviewed annually"      cited domains/regulatory/cbc-emi.md
carried    "add to the decision log" edited domains/regulatory/decisions.md
proposed   "reviewed annually" -> "reviewed every six months"   verified
proposed   "quarterly" -> "monthly"   NOT verified: docx_anchored failed

files changed in the hub: domains/regulatory/decisions.md, policy.md
1 unmarked reply since my last answer, on the thread about scope
```

Then the full text of every reply and every comment body, exactly as sent, with
the thread it went to. Not a summary. Nothing else in the run shows Nail what is
now visible to everyone on the document.

```
--- posted to "reviewed annually" ---
🤖 <the reply, exactly as sent>

--- comment on the proposal "reviewed every six months" ---
🤖 <the body, exactly as sent>
```

Say what you cut from a draft and why, which threads you stopped on and are
still waiting for, and which proposals did not verify.

If you could not read half the review, say so and do not report clean. A
`comments` run with warnings, a thread whose range came back null, a suggestions
read that failed: each one means part of the review was invisible on this run.

## Live mode

Nail says "live" in the request, and the session stays open on that one
document. Everything above runs first, once, and its Step 3 list and Step 8
report are printed as they always are. Then the session starts watching.

One session watches one document, the link Nail gave. There is no hub-wide
watch.

The loop. `CURSOR` starts as the `cursor` from Step 1's `comments` run:

```bash
$GDOC comments <url> --since "$CURSOR" --wait 9m
```

That one call blocks. The binary polls Drive every ten seconds inside it and
comes back on the first activity after the cursor, or at the deadline with an
empty window. So a quiet document costs one call and no thinking. Read the
object and take one of four paths:

- **`ok: true`, `threads: []`, `waited.interrupted: false`.** Nothing happened
  in those nine minutes. Print nothing at all. Call again with the same cursor.
- **`ok: true` with threads.** This is a window. Run Steps 2 to 8 over exactly
  those threads. Print the Step 8 report. Set `CURSOR` to the answer's `cursor`
  and call again.
- **`ok: false`.** A poll failed. Print the error, wait thirty seconds, call
  again with the cursor you already had. Never move the cursor past a failed
  poll: that window is unread, not empty. Three failures in a row and you stop
  and say so.
- **`waited.interrupted: true`.** Nail stopped it. Print the totals below and
  stop.

Nine minutes, and never more. The binary would look for an hour, but the tool
that runs the command gives up at ten, and a wait killed at ten minutes takes
its answer with it. Ask for nine.

`--wait` needs `--since`. The first read is the baseline and takes no wait: a
wait with no cursor answers with the whole document, which is Step 1 under
another name and reads to a session as news.

### What a window contains

Everything with activity after the cursor. That includes gdoc's own replies:
the binary reports what arrived and judges none of it, exactly as in Step 2.

- A window that is only gdoc's own 🤖 replies coming back is not work. Say
  nothing, take the new cursor, call again. It does not loop, because the new
  cursor is past those replies.
- A thread whose `range` is `null` can be answered and cannot be proposed into.
  Say which when it matters, and answer it in the thread.
- A thread that was answered before and now carries a new marked comment is new
  work. Step 2 already says so, and "Two messages at most" is per piece of work.
- An unmarked reply is still reported and never acted on.
- `--witness` works in a window as it works in a one-shot listing. An empty
  window is not witnessed, so `waited.polls` with no threads carries no witness
  and that is not a gap.

### Before acting on a marked comment that may be old

A cursor can come from a session that ended, or be pasted in by hand, so a
window can carry a comment gdoc already acted on. Run the check from Step 2
before acting: `$GDOC suggestions <url>` and the paired note's `proposals`. When
the work is already there, write the missing receipt rather than doing it twice.

### Stop, and the totals

Nail stops it with Ctrl-C or by saying stop. An answer carrying
`waited.interrupted: true` is Nail stopping, not a failure: the object says
`ok: true` and the cursor is the one handed in.

Then print the session's totals, in the shape of the Step 8 report:

```
live session ended after 4 windows, 47 minutes
answered   3 threads
carried    1 thread
proposed   2 changes, 1 of them NOT verified: docx_anchored failed
files changed in the hub: domains/regulatory/decisions.md, policy.md
2 unmarked replies reported, acted on none
1 poll failed and was retried
```

Report the failed polls. A session that could not read part of its own watch
must not read as a clean one, which is Step 8's rule over the whole session.

Dry run applies here as it does everywhere else: print each reply and each
proposal instead of posting it, keep waiting, and say at the end that nothing
was posted.

## Never

- Never edit the document. Every change to its words is a suggestion, and the
  guard refuses anything else on a document that was handed in.
- Never resolve or reopen a thread. Resolving means the answer was accepted, and
  only Nail accepts.
- Never accept, reject or delete anyone else's suggestion. `withdraw` retracts
  gdoc's own pending proposal and nothing else.
- Never delete anything from the hub.
- Never run git.
- Never write markdown into a comment thread.
- Never write a reply that does not open with `🤖 `, and never put the mark in a
  proposal's `why`: gdoc adds it there, and a reason carrying it is refused.
- Never trust a status code. Read `verified`, and `checks` where it is there.
- Never act on an unmarked comment unless Nail asked for all-comments mode and
  picked that one.
- Never reply twice to the same piece of work. A thread that asks again gets a
  second answer, and so does an answered thread Nail picked in all-comments
  mode, where you say so before posting.
- Never write to a multi-tab document.
- Never export a PDF. Nail downloads it from the browser.
- Never post a reply you did not print in full afterwards. That printing is the
  record.
