---
worth: yes
where: go/cmd/gdoc/commands.go:147
added: 2026-10-02
---
# gdoc mcp: the binary as a local MCP server for Claude Desktop chat

Nail's ask on 2026-10-02: review a Google Doc from Claude Desktop chat and from
voice mode, not only from a Claude Code terminal. Chat has no shell, so it
cannot run `gdoc`. It can run a local MCP server that Claude Desktop starts
from an installed extension. Voice mode uses connectors and tools, not skills
that run code, so a tool is the only way in.

The brainstorm ran the same day. This item holds what it decided, until the
spec takes it over.

## What was decided, 2026-10-02

- **One binary.** `gdoc mcp` is a command in the same binary. Claude Desktop
  starts it once and keeps it running. JSON-RPC over stdin and stdout, written
  by hand in a new `internal/mcp`, so no fourth module.
- **Tools come from the command table.** An entry marked as exposed becomes a
  tool, its description and arguments built from the entry's summary and
  flags. A call runs the same dispatch the CLI runs, in the same process, and
  returns the same object. Where the CLI takes a file, the tool takes text,
  written to a private temp file for that call. Every command already builds
  its own `guard.NewPolicy()`, so a grant ends with the call; a test pins that
  one call's grant never reaches the next.
- **Exposed:** `read`, `comments`, `suggestions`, `reply`, `annotate`,
  `propose`, `login`, and a new `guide`. **Not exposed:** `update`, `restyle`,
  `publish`, `export`, `build`, `probe`, `help`, `completion`. The rule: chat
  gets what is safe to trigger by a spoken sentence, and what creates files,
  rewrites a document or changes the installation stays where it is typed.
- **Login in chat.** With no token, every tool answers that the user is not
  signed in and should call `login`. `login` opens the loopback listener in the
  running process and returns the sign-in link as the tool's answer. The token
  goes from Google to the listener and never through the chat. The link must
  be opened on the computer running Claude Desktop; the answer says so.
- **Document only.** No hub in version 1, so the extension has no settings.
  Reaching the hub is `a-bridge-from-chat-to-the-hub.md`.
- **An `ai!` in chat has two answers.** The user answers it there, and Claude
  drafts the 🤖 reply, reads it back and posts after a yes, as in the
  all-comments mode; or it is left for a Claude Code session to carry out in
  the hub. Claude says before posting that a 🤖 reply makes a later session
  treat the thread as answered.
- **Chat and a live Claude Code session run side by side** on one document,
  with one token. Chat's replies open with 🤖, so the live session never takes
  them as work. The day-one hand-off is an `ai!` the user types in the browser;
  gdoc cannot write one, because a marker is read only from a comment's first
  word and gdoc's comments open with 🤖, which is the loop protection. Two
  guards go into the chat core against both answering one comment: while a
  live session runs, chat leaves marked comments to it, and chat asks when it
  does not know; and chat reads the thread again just before posting and stops
  when a 🤖 reply has appeared. The spec checks that two processes refreshing
  the token at once is safe.
- **No live mode in chat.** An MCP server cannot wake Claude. The cursor from
  `comments` lets Claude check again whenever the user asks.
- **The review rules are one core.** `skills/gdoc-review/review.md` holds them;
  `SKILL.md` keeps only what is Claude Code's. `make build` copies `review.md`
  into the package that embeds it, the copy is committed, and a test fails when
  the two differ. Chat gets about six always-on lines in `initialize`, and the
  `guide` tool returns a short chat header plus `review.md`. The core gains a
  voice adaptation, drafts read aloud and a yes before posting, and a flow for a
  new comment on words the user names, through `annotate`.
- **The probe leaves `propose`.** Suggestions are generally available from
  2026-09-30, rolling out over 15 days. Nail accepts the risk of one silent
  direct edit. `propose` drops `--folder` and the probe, keeps the read-back,
  and stops the run at the first proposal whose read-back finds no suggestion,
  so a silent failure costs at most one edit. `gdoc probe` stays as a manual
  command. Same milestone as `gdoc mcp`.
- **Colleagues from day one, Macs only.** `release.yml` builds `gdoc.mcpb`
  with a universal binary inside, made with `lipo` on the macOS runner.
  Windows is untouched. `make mcpb-dev` builds a thin bundle whose command
  points at the checkout's `bin/gdoc`, so `make build` still updates Nail's.
- **Updates.** A private extension does not auto-update. `gdoc mcp` runs the
  daily release check `help` runs, under the same grant and ceiling, and a
  tool answer carries a one-line notice with the `.mcpb` link that the
  instructions tell Claude to mention once.
- **No Apple Developer ID for now.** `install.sh` strips the quarantine mark,
  but nothing does that for a bundle. If Claude Desktop marks what it unpacks,
  a colleague allows the binary once in System Settings, Privacy & Security.

## The spike, first task of the plan

Install a `.mcpb` downloaded in a browser and check: a command outside the
bundle with `${HOME}` or a checkout path, the empty settings form, whether the
unpacked binary carries the quarantine mark, and the universal binary on both
chips if an Intel Mac is at hand.

## DECISIONS entries, written before code

- Supersede the 2026-09-16 rejection of an MCP server inside the binary. Its
  three reasons: grants end per call already; the token cost is paid only in
  chat, where there is no cheaper route, and the always-on text is short with
  the rest behind `guide`; it does nothing at the terminal, which keeps the CLI.
- Supersede the probe half of 2026-08-29, "Never send `writeMode: SUGGEST` and
  trust the 200": the read-back and the stop now carry it. Update the
  "loss must be loud" paragraph and the PRINCIPLES.md amendment of 2026-09-18.
- The binary never reads stdin: becomes no CLI command reads stdin.
- One JSON object reaches stdout: holds for the CLI, not for `gdoc mcp`.
- `help` is the one command that asks unasked: `gdoc mcp` is the second.
- Skills are linked, not copied: the review core is embedded in the binary.

## Before the plan

A review of the spec by four readers in parallel: an architect, a security
reviewer, a Codex second opinion and a Go and protocol reviewer. Their
disagreements go to Nail before the plan is written.
