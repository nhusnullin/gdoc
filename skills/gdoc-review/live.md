# Live mode

Read this when the request says "live", after the first pass through
SKILL.md has run and printed its Step 8 report. The steps it names are
SKILL.md's steps.

You say "live" in the request, and the session stays open on that one
document. Everything in SKILL.md runs first, once, and its Step 3 list and Step 8
report are printed as they always are. Then the session starts watching.

One session watches one document, the link you gave. There is no hub-wide
watch.

## The watch, under the Monitor tool

`CURSOR` starts as the `cursor` from Step 1's `comments` run. That run always
prints one, a document with no comments in it included.

Where this session has the Monitor tool, start the watch under it, with a limit
of 30 minutes:

```bash
$GDOC comments <url> --since "$CURSOR" --follow 29m
```

That call does not come back with a window. It holds the poll open for half an
hour and writes one JSON line the moment it sees a comment or a reply marked
for gdoc, so each window arrives as its own event and a quiet half hour costs
no thinking at all. Where the Monitor tool here allows a shorter limit than
thirty minutes, ask for the longest it allows and set `--follow` a minute under
it.

Read the `line` field to tell the two kinds of line apart. A window carries
`"line": "window"`. The envelope the follow ends on carries no such field.

A window line names ids, markers and times, and no words at all:

```json
{"line":"window","since":"<cursor>","cursor":"<cursor>","noticed":"2026-10-27T09:14:02Z","items":[{"id":"AAABc","thread":"AAABc","marker":"ai?","created":"2026-10-27T09:13:58Z"}],"more":0}
```

To read the window, run `$GDOC comments <url> --since <the line's since>`.
Act on the items the line names, and leave anything newer in that read to the
next line: the follow is still polling, and whatever landed after the line was
written rides in the line after it. Run Steps 2 to 8 over those items, print
the Step 8 report, and keep the line's `cursor` as `CURSOR`.

Items whose `marker` is `none` are listed in that window's report and never
acted on, which is Step 2's rule and not a new one. `more` is how many items
did not fit on the line. The read names them all.

The last line is the ordinary `comments` envelope, with `threads` empty and
`waited` filled in. Take one of three paths:

- **`ok: true`, `waited.interrupted: false`.** The half hour ran out and
  nothing is wrong. Start the next follow from the last cursor you kept.
- **`ok: true`, `waited.interrupted: true`.** You stopped it. Print the totals
  below and do not restart.
- **`ok: false`.** A poll failed. Today's retry rule holds: print the error,
  wait thirty seconds, start again from the cursor you already had, and stop
  after three failures in a row.

When the person says stop in the chat, stop the Monitor task, then
print the totals once and do not restart on the envelope that follows. The
task is stopped from outside, so its last line lands after your answer and
says the same stop twice.

`--follow` needs `--since`, and is refused beside `--wait` and beside
`--witness`.

If the binary refuses `--follow`, or there is no Monitor tool in this session,
use the wait loop below. It watches the same document and costs a turn more
often.

## The fallback loop, on `--wait`

The loop. `CURSOR` starts as the `cursor` from Step 1's `comments` run. That
run always prints one, a document with no comments in it included: there is
nothing to construct by hand and nothing to work around.

```bash
$GDOC comments <url> --since "$CURSOR" --wait 9m
```

Run it with the Bash tool's `timeout` set to `600000`, ten minutes in
milliseconds. The default is two minutes, so a nine minute wait left on the
default is cut short at two: killed outright it is a timeout matching none of
the four paths below, and killed with a signal it comes back
`waited.interrupted: true`, which reads as you having stopped the session. Set
the timeout on every call in this loop.

That one call blocks. The binary polls Drive every two seconds inside it and
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
  poll: that window is unread, not empty. Three failures in a row and the
  session stops and says so.
- **`waited.interrupted: true`.** You stopped it. Print the totals below and
  stop.

Nine minutes, and never more. The binary would look for an hour, but the tool
that runs the command gives up at ten even when it is asked for its longest, and
a wait killed at ten minutes takes its answer with it. Ask for nine, and set the
tool's timeout to ten.

`--wait` needs `--since`. The first read is the baseline and takes no wait: a
wait with no cursor answers with the whole document, which is Step 1 under
another name and reads to a session as news.

## Which cursor to keep

Every `comments` answer carries a `cursor`, and it always means the newest
activity that call saw. It does not mean the newest activity you read. A
cursor you keep moves the watch past everything before it, and nothing brings
that span back: a thread whose only news was inside it stays silent until
somebody touches it again.

- A window's cursor is safe to keep. You just read that window.
- Mid-session, when you want to look at the threads again, run
  `$GDOC comments <url> --since "$CURSOR"`, not a bare listing. The answer is a
  window, so its cursor is one you have read up to.
- Keep a bare listing's cursor only when you read the whole answer, replies
  included.
- Never read a listing through a pipe that narrows it, and then keep its
  cursor. The pipe drops what you did not read, and the cursor still moves past
  it. This is the failed-poll rule again: that span is unread, not empty.

When you are not sure, keep the cursor you already had. The worst case is a
window you read twice, and a thread you already answered carries gdoc's receipt.

## What a window contains

Everything with activity after the cursor. That includes gdoc's own replies:
the binary reports what arrived and judges none of it, exactly as in Step 2.

- A window that is only gdoc's own replies coming back is not work. Say
  nothing, take the new cursor, call again. It does not loop, because the new
  cursor is past those replies.
- A thread whose `range` is `null` can be answered and cannot be proposed into.
  Say which when it matters, and answer it in the thread.
- A thread that was answered before and now carries a new marked comment or
  a reply whose `marker` is not `none` is new work. Step 2 already says so,
  and "Two messages at most" is per piece of work.
- An unmarked reply is still reported and never acted on.
- `--witness` works in a window as it works in a one-shot listing. An empty
  window is not witnessed, so `waited.polls` with no threads carries no witness
  and that is not a gap. After a wait the export runs on what is left of the
  nine minutes, so a window that arrives near the deadline can come back with
  every thread `unmatched` and a warning saying the export was cut short. That
  is the witness missing, not the threads.

## Before acting on a marked comment that may be old

A cursor can come from a session that ended, or be pasted in by hand, so a
window can carry a comment gdoc already acted on. Run the check from Step 2
before acting: `$GDOC suggestions <url>` and the paired note's `proposals`. When
the work is already there, write the missing receipt rather than doing it twice.

## Stop, and the totals

You stop it with Ctrl-C or by saying stop. An answer carrying
`waited.interrupted: true` is you stopping, not a failure: the object says
`ok: true` and the cursor is the one handed in.

Then print the session's totals, in the shape of the Step 8 report:

```
live session ended after 4 windows, 47 minutes
answered   3 threads
carried    1 thread
proposed   2 changes, 1 of them NOT verified: docx_anchored failed
files changed in the hub: domains/regulatory/decisions.md, policy.md
1 marked reply acted on
2 unmarked replies reported, acted on none
1 poll failed and was retried
```

Report the failed polls. A session that could not read part of its own watch
must not read as a clean one, which is Step 8's rule over the whole session.

Dry run applies here as it does everywhere else: print each reply and each
proposal instead of posting it, keep waiting, and say at the end that nothing
was posted.
