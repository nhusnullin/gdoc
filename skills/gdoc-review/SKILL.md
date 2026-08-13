---
name: gdoc-review
description: Use when Nail gives a Google Doc link and wants his ai: comments handled. Reads the comments, answers local ones in their threads, captures global ones for a later session.
---

# Google Docs review

Answer the `ai:` comments Nail left in a document. Local changes get an answer in
the thread. Global changes get captured, never attempted.

The marker is `ai` plus a sign, at the start of a comment: `ai:` leaves the
choice to you, `ai?` means answer it here, `ai!` means capture it as global. The
old `@ai` form still counts. The CLI does this matching; you never re-derive it.

Spec: `~/src/altery/intelligence-hub/docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md`

## Setup

```bash
HUB="$HOME/src/altery/intelligence-hub"
GDOC="$HOME/.config/gdoc-agent/venv/bin/python -m tools.gdoc.cli"
```

Every CLI call runs inside `(cd "$HUB" && ...)`. The subshell leaves Nail's own
working directory unchanged. That directory is the corpus you search: the cwd he
chose decides which notes ground the answers.

## If Nail passes `--terminal-only`

Do every step, but post nothing to the document. Print each reply in the
terminal instead of running `$GDOC reply`, and print each refusal instead of
posting it. Capture still runs: `pending.md` is a local file, not a change to
the document. Say at the end that nothing was posted.

## Step 1: Read the comments

```bash
(cd "$HUB" && $GDOC read <url>)
```

Returns `mine`, `others` and `skipped`. Threads already answered by the service
account appear under `skipped`, which is what makes a second run safe.

## Step 2: Show Nail what you found, and stop

Print the lists and ask before posting anything. Two things Nail must be told
every run, because the agent cannot check either one:

- The Drive API returns no email address for comment authors, so `mine` is a
  display-name guess, not proof.
- `permissions.list` is refused under Commenter, so the agent cannot see who
  else can read this document. Replies are visible to all of them.

```
Yours (will act):      para 3, para 7, para 11
Others (context only): para 5 (William Mejia)
Already answered:      para 2

I cannot see who else has access to this document. Replies will be
visible to everyone on it, under the service account address.

Proceed?
```

Wait for an answer. Never post before this.

## Step 3: Classify each of Nail's comments

Per comment, not per batch. One run may answer two and capture two.

| | Local | Global |
|---|---|---|
| Test | The change fits inside the quoted span | It touches text the comment does not quote |
| Examples | A question. Rephrase this. Is this term right? Add a missing clause | Renumber sections. Restructure. Apply a term change everywhere |
| Action | Answer in the thread | Capture, and reply with the refusal |

`forced_kind` in the JSON carries Nail's override: `ai?` means answer it in the
thread, `ai!` means capture it. `ai:` means he left the choice to you. Honour an
override without re-deciding.

An unanchored comment has `anchored: false` and no quote. It refers to the
document as a whole, so treat it as global unless it is plainly a question.

If a comment is genuinely ambiguous, ask Nail in the terminal. Do not guess.

## Step 4: Ground the answer

Search Nail's current repo. Cite plain file paths. The corpus mixes Russian and
English, so search in both languages.

If the repo has no source for the answer, say so in the reply. Never write a
plausible sentence to fill the gap.

## Step 5: Write the reply

Plain text only. Docs comment threads show markdown source as typed, so `**`,
backticks and `#` headings arrive broken. The CLI refuses them, which is a
safeguard, not permission to try.

Shape:
1. The replacement text or the answer, first, so the first thing Nail sees is
   the thing he copies.
2. One short reason line, only if the reason is not obvious.
3. Sources as plain paths, last.

Write it to a file and post it:

```bash
cat > /tmp/reply.txt <<'EOF'
<the reply>
EOF
(cd "$HUB" && $GDOC reply <doc_id> <comment_id> --body-file /tmp/reply.txt)
```

## Step 6: Capture global items

```bash
(cd "$HUB" && $GDOC capture <doc_id> <comment_id> --slug <slug> --repo-root "$HUB")
```

Then post the refusal, using the item number the capture returned:

```
Understood. This needs changes across the document, so I am not proposing text
here. Captured as item 2. I will handle it in a terminal session and produce a
new version of the document.
```

```bash
cat > /tmp/refusal.txt <<'EOF'
<the refusal>
EOF
(cd "$HUB" && $GDOC reply <doc_id> <comment_id> --body-file /tmp/refusal.txt)
```

## Step 7: Mirror, only if paired

Mirror only when the document has a paired markdown file, or when Nail owns it
and asks for one. Never mirror a document he does not own: counsel drafts and
partner documents stay in Drive.

```bash
(cd "$HUB" && $GDOC export <url> --repo-root "$HUB")
```

If it reports a mirror conflict, the markdown has uncommitted edits. Tell Nail
and let him decide. Do not pass `--force` on your own.

## Step 8: Report

```
posted   para 3   answered, cited domains/regulatory/cbc-emi.md
posted   para 7   rephrased, ready to paste
captured para 2   global: renumber sections

1 global item in docs/gdoc/<slug>/pending.md
Next session: /gdoc-apply docs/gdoc/<slug>/pending.md
```

## Never

- Never edit the reviewed document. The credential cannot, and neither may you.
- Never resolve a thread. Resolving means Nail accepted the text.
- Never reply twice to the same comment.
- Never act on a comment that is not Nail's.
- Never attempt a global change in a comment thread.
