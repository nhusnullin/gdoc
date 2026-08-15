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
item carries its `author`, so an unexpected name is visible. Threads gdoc has
already answered appear under `skipped`, which is what makes a second run safe.

Every reply gdoc posts ends with `[gdoc]` on its own line. That marker, not the
account name, is how a later run knows the thread was answered. Under `oauth`
replies post under Nail's own name, so the marker is the only signal that works.

One case to expect once: the first `oauth` run on a document the service account
reviewed earlier can list threads that were in fact answered. Those old replies
carry no marker and no longer read as gdoc's. Each item's `replies` field shows
what is already there, so say so and let Nail decide.

## If Nail asks for all the comments

By default only marked comments are work. When Nail says he wants to go through
every comment, add `--all`:

```bash
$GDOC read <url> --all
```

`addressed` then holds every unresolved thread, marked or not, including ones
that already have an answer. Each item says `marked`, `answered`, and carries
its existing `replies`.

This mode never batch-approves. Print the numbered list with author, quote and
content, answered threads last and labelled, and let Nail pick. For each comment
he picks: draft the reply, show it in the terminal, wait, then post. Unmarked
comments carry no `forced_kind`, so classify local against global yourself and
say which you chose.

Picking an answered thread is allowed. It is the one case where you reply twice
to the same comment, and you say so before posting.

## Step 2: Show Nail what you found, and stop

Print the list and ask before posting anything. One thing Nail must be told every
run, because the agent should not rely on checking it: under `service_account`
`permissions.list` is refused, so the agent cannot see who else can read this
document. Replies are visible to all of them, and a posted reply cannot be
taken back.

```
Will act:         para 3, para 7, para 11 (Nail), para 5 (William Mejia)
Already answered: para 2

I cannot see who else has access to this document. Replies will be
visible to everyone on it. Under auth_mode oauth they post under your
own name; under service_account they post under the service account.

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

- Never edit the reviewed document. Under `service_account` the credential
  cannot. Under `oauth` it could, and you still may not: replies go in threads,
  and document-wide changes go through the paired markdown.
- Never resolve a thread. Resolving means Nail accepted the text.
- Never reply twice to the same comment, unless Nail picked an answered thread
  in all-comments mode, and say so before posting.
- Never act on an unmarked comment unless Nail asked for all-comments mode and
  picked that comment.
- Never attempt a global change in a comment thread.
- Never write the baseline. That is `generate`'s job.
