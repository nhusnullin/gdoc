---
worth: yes
where: tools/
added: 2026-09-29
---
# Measure how well the binary and the skills work together, from the transcripts we already have

## Why

Nail, 2026-09-28: "if we log actions and how cli tools works with session
number, will it give us more insight how optimal system binary + skill works".
A brainstorm that night (a skeptic, a principles guardian and a systems
engineer) came to one answer: the log mostly exists already, so read it
before building one.

Claude Code writes every session to
`~/.claude/projects/<dir>/<session-id>.jsonl`, and a subagent's session to
`<session-id>/subagents/*.jsonl`. Each record carries `sessionId`,
`timestamp`, the message, `message.usage` on assistant turns, and
`attributionSkill` when a skill is loaded. Every `gdoc` call is a Bash
`tool_use`, and its result holds the full JSON envelope, including `version`.
A Bash command also gets `CLAUDE_CODE_SESSION_ID`, which equals the file name.
So this is a pilot, with no change to gdoc. Only if it finds a question the
transcripts cannot answer does step 2 follow (see "Not in scope").

## The target: six questions

"How well binary plus skill work" means these six, and nothing else:

1. **Wasted polls in a live review.** Of the `gdoc comments --since` calls in
   a live `gdoc-review` session, how many returned nothing new? Report polls
   per session, the empty share, the median interval between polls, empty
   polls per comment that needed action, and the tokens spent on the turns
   that handled an empty poll.
2. **Wrong flags the skill passed.** `ok: false` envelopes that are usage
   refusals: unknown command, unknown flag, repeated flag, missing or empty
   value, trailing arguments. Read the exact error wording in
   `go/cmd/gdoc/commands.go` and `go/cmd/gdoc/doc.go` ("An unknown flag, a
   repeated flag...") and match on it, not on guesses. Group by skill,
   subcommand and flag name. Count `gdoc help <command>` calls in the middle
   of a task as a weaker sign of the same thing.
3. **Retries.** After an `ok: false`, the same subcommand called again within
   the next three gdoc calls: split into same arguments and changed
   arguments. Separately, an identical successful call repeated in one
   session (same subcommand, same argument hash), which is a redundant read.
4. **Calls per finished answer.** An outcome is an `ok: true` envelope from
   `reply`, `propose` (with `verified: true`), `annotate`, `withdraw`,
   `publish`, `export` or `restyle`. For each outcome count the gdoc calls,
   the other tool calls, the assistant tokens and the wall time since the
   previous outcome or the session start. Report the median and the worst
   three, per skill.
5. **Time per `ai!` comment.** From the first `comments` envelope that shows
   a thread whose `marker` is `ai!` and has no gdoc reply after it, to the
   first envelope of a `reply` or `propose` that names that thread. Report
   the median, the worst three, and how many `ai!` threads never got one in
   the session. Check the exact field names in the `comments`, `reply` and
   `propose` envelopes (`go/internal/comments/comments.go`, SPEC.md) before
   writing the matcher.
6. **Where Nail corrected the session.** A message Nail typed (a `user`
   record that is not a tool result, not `isMeta`, not a hook or system
   text) within two of his turns after a gdoc step, that carries a
   correction signal: "no", "wrong", "why", "stop", "not what", "again", and
   the same in Russian ("нет", "не так", "почему", "стоп"). Report the count,
   the gdoc command it followed, and the session id with the record `uuid`.
   Never the text.

Every number is split by the `version` the envelopes carry. The sessions
before the release that closes the 2026-09-28 night's three branches are the
baseline. The sessions after it are the comparison. A build from a checkout
has no `version` and gets its own row.

## Before running it

- **Nail's yes in chat, for that run.** On 2026-09-28 the permission check
  refused a read of the hub transcripts, because they hold Altery documents.
  A session never works around that refusal. It asks, and it runs only on a
  yes given in chat for that run.
- **Enough sessions after the release.** At least five live `gdoc-review`
  sessions on the new version, or the comparison is noise. Say so in the
  report when there are fewer.

## What to build

`tools/session-insight/`, one Go program with its own `go.mod` and the
standard library only, beside `tools/tlsdiag`. It is not part of the binary,
not in `make dist`, and not in a release. Add its row to CLAUDE.md "What lives
where". It reads JSONL and prints; it runs no program and opens no network.

- Input: the project folders to read, named on the command line. Default to
  none, so an empty run reads nothing. The ones that ran gdoc on 2026-09-28
  were `-Users-nailkhusnullin-src-altery-Altery-Platform-hub`,
  `-Users-nailkhusnullin-src-altery-intelligence-hub` and
  `-Users-nailkhusnullin-src-altery-Altery-Platform-Hub-11-m2-crypto-project-eagle`.
  This repo's own development sessions are out of scope.
- Output: one JSON object with the six answers and a short Markdown table of
  the same, written to a path the caller names. It never lands in the repo
  and is never committed.
- TDD. The tests use small JSONL fixtures written by hand with made-up
  content, one per question, plus one with a malformed line (skipped and
  counted, never a crash). Read the record shape from this repo's own
  transcripts, which are safe to read, before writing the parser.

## What never leaves the transcript

The output carries counts, durations, tokens, subcommand names, flag names,
error kinds, skill names, versions, session ids and record uuids. It never
carries a document id, URL, title, tab name, thread or suggestion id,
quote, comment or reply text, note path, person's name, token, or raw Google
error text. Arguments are compared by hash only. A test feeds a fixture full
of such values and fails if any of them appears in the output.

## Not in scope

- Any change under `go/`. If the pilot finds slow polls and cannot tell
  whether Google or the model is slow, the next step is an opt-in timing fact
  in the envelope (requests made, time on Google, total time), counted in
  `internal/gapi`. That is its own item, written only then.
- A log file written by gdoc, and any upload from a colleague's machine. The
  first strains principle 2 and needs a DECISIONS.md entry. The second is
  never.

## Done when

The tool is on a branch with its tests green and nothing committed from a
transcript. It has run once, on Nail's yes, over the baseline and the
post-release sessions. The report answers the six questions with numbers, and
names up to five findings that would change a skill or the binary. Each
finding points at the session id and record uuid that shows it, so Nail can
open that one spot himself.
