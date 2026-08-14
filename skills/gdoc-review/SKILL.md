---
name: gdoc-review
description: Use when Nail gives a Google Doc link and wants the ai: comments in it handled. Reads the comments, answers local ones in their threads, captures global ones for a later session.
---

# Google Docs review

Answer the `ai:` comments in a document, whoever wrote them. Nail chose the
document, and the marker is the instruction. Local changes get an answer in the
thread. Global changes get captured, never attempted.

The marker is `ai` plus a sign, at the start of a comment: `ai:` leaves the
choice to you, `ai?` means answer it here, `ai!` means capture it as global. The
old `@ai` form still counts. The CLI does this matching; you never re-derive it.

Spec: `~/src/personal/gdoc/docs/superpowers/specs/2026-08-14-gdoc-storage-and-iteration-design.md`,
which supersedes the storage parts of the 2026-08-13 spec beside it.

## Setup

```bash
GDOC="$HOME/.config/gdoc-agent/venv/bin/gdoc"
ROOT="$PWD"
```

`$GDOC` is an installed command, so it runs from any directory. Nail's own
working directory stays where he put it.

One root, `$ROOT`, and it is `$PWD`:

- It is the corpus you search to ground answers.
- It is the tree scanned for markdown paired to a document.
- Tool-managed files go in `$ROOT/.gdoc/<slug>/`, beside the source markdown.

`$PWD` is the CLI default for every `--repo-root`, so you never pass it.

`$ROOT` may not be a git repository. Nothing here requires one.

## If Nail passes `--terminal-only`

Do every step, but post nothing to the document. Print each reply in the
terminal instead of running `$GDOC reply`, and print each refusal instead of
posting it. Capture still runs: `pending.md` is a local file, not a change to
the document. Say at the end that nothing was posted.

## Step 1: Read the comments

```bash
$GDOC read <url>
```

Returns `addressed` and `skipped`. A marked comment from anyone in the document
is addressed: the marker is the instruction, and Nail chose the document. Each
item carries its `author`, so an unexpected name is visible. Threads already
answered by the service account appear under `skipped`, which is what makes a
second run safe.

## Step 2: Show Nail what you found, and stop

Print the list and ask before posting anything. One thing Nail must be told every
run, because the agent cannot check it: `permissions.list` is refused under
Commenter, so the agent cannot see who else can read this document. Replies are
visible to all of them, and a posted reply cannot be taken back.

```
Will act:         para 3, para 7, para 11 (Nail), para 5 (William Mejia)
Already answered: para 2

I cannot see who else has access to this document. Replies will be
visible to everyone on it, under the service account address.

Proceed?
```

Wait for an answer. Never post before this.

## Step 3: Classify each addressed comment

Per comment, not per batch. One run may answer two and capture two.

| | Local | Global |
|---|---|---|
| Test | The change fits inside the quoted span | It touches text the comment does not quote |
| Examples | A question. Rephrase this. Is this term right? Add a missing clause | Renumber sections. Restructure. Apply a term change everywhere |
| Action | Answer in the thread | Capture, and reply with the refusal |

`forced_kind` in the JSON carries the override written in the comment: `ai?`
means answer it in the thread, `ai!` means capture it. `ai:` leaves the choice to
you. Honour an override without re-deciding.

An unanchored comment has `anchored: false` and no quote. It refers to the
document as a whole, so treat it as global unless it is plainly a question.

If a comment is genuinely ambiguous, ask Nail in the terminal. Do not guess.

## Step 4: Ground the answer

Search `$ROOT`. Cite plain file paths. The corpus mixes Russian and English, so
search in both languages.

If the root has no source for the answer, say so in the reply. Never write a
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
$GDOC reply <doc_id> <comment_id> --body-file /tmp/reply.txt
```

## Step 6: Capture global items

```bash
$GDOC capture <doc_id> <comment_id>
```

The queue directory is named after the paired source markdown file, which the
CLI finds from the document id. Every version of a document resolves to the same
source, so an old version and a new one share one queue.

Two answers that need Nail, not a retry:

- An error naming `--slug` means no markdown under `$ROOT` is paired to this
  document. Either Nail does not own it, or the pairing is missing. Say which
  you think it is and stop.
- `source_collision_warning` means the queue already holds items for a different
  source file with the same name. Tell Nail before you go on.

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
$GDOC reply <doc_id> <comment_id> --body-file /tmp/refusal.txt
```

## Step 7: Report

```
posted   para 3   answered, cited domains/regulatory/cbc-emi.md
posted   para 7   rephrased, ready to paste
captured para 2   global: renumber sections

1 global item in .gdoc/<slug>/pending.md
Next session: /gdoc-apply .gdoc/<slug>/pending.md
```

Write no snapshot of the document here. `gdoc generate` writes the baseline, at
the one moment the document and the markdown provably match. By the end of a
review the document may already carry Nail's direct edits, and a snapshot taken
now would bake them in and hide them from the next apply.

## Never

- Never edit the reviewed document. The credential cannot, and neither may you.
- Never resolve a thread. Resolving means Nail accepted the text.
- Never reply twice to the same comment.
- Never act on a comment without the marker.
- Never attempt a global change in a comment thread.
- Never write the baseline. That is `generate`'s job.
