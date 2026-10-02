---
worth: yes
where: go/cmd/gdoc/commands.go:147
added: 2026-10-02
---
# gdoc mcp: the binary as a local MCP server for Claude Desktop chat

Nail's ask on 2026-10-02: review a Google Doc from Claude Desktop chat, and
later from voice mode, rather than only from a Claude Code terminal. Chat has
no shell, so it cannot run `gdoc`. It can run a local MCP server that the
desktop app starts from `claude_desktop_config.json`. Voice mode uses
connectors and tools, not skills that run code, so a tool is the only way in.

## The shape

One more command in the same binary, `gdoc mcp`. Claude Desktop starts it once
and keeps it running. It speaks JSON-RPC over stdin and stdout. Each
`tools/call` turns its arguments into the same words a person types and calls
the same dispatch the CLI calls, in the same process. No subprocess, so the
rule that nothing under `go/` runs an external program holds.

- `tools/list` is generated from the command table in `commands.go`. The
  table already carries each flag, its kind, its sentence and whether it is
  required, so the CLI, `help`, completion and MCP keep reading one
  description.
- The JSON-RPC loop is written by hand, about 300 lines, so there is no fourth
  module.
- The token stays in `~/.config/gdoc-agent/`. Login stays a terminal command.
- A flag that takes a file today (`--body-file`, `--from`) needs a text form
  for a tool: the adapter writes a temp file, or the command learns an inline
  value.
- First version: `read`, `comments`, `suggestions`, `reply`, `annotate` and
  `propose`. `propose` brings its probe folder and its note into a chat flow,
  and the spec has to say where a chat session gets both. `withdraw` follows.

Install for Nail is one entry in
`~/Library/Application Support/Claude/claude_desktop_config.json`, with the
absolute path to `~/.local/bin/gdoc` and `args: ["mcp"]`, then a restart of
the app. A colleague would get a `.mcpb` desktop extension per platform from
the release workflow, later.

## The rejection this reopens

`docs/v2/DECISIONS.md`, 2026-09-16, the M7d entry, rejected an MCP server
inside the binary for three reasons. Building this is a new DECISIONS entry
that supersedes that paragraph, written before any code. What has changed:

- **Grants die with the process.** Every command builds its own
  `guard.NewPolicy()` inside its run (`read.go`, `write.go`, `publish.go`,
  `restyle.go`, `update.go`), so a grant already ends with the call, not the
  process. That needs a test pinning that a grant from one tool call does not
  reach the next, and a check for package-level state in `cmd/gdoc`.
- **Tool schemas cost tokens in every session.** True in Claude Code, which is
  why it is not installed there: Claude Code keeps the CLI and the skills. In
  desktop chat and voice there is no cheaper route, so the cost buys the only
  way in.
- **It does nothing at the terminal.** Still true, and not the point. The
  terminal keeps the CLI unchanged.

The 2026-08-29 entry, MCP does not replace the REST client, is untouched. This
is MCP on the near side of the binary, and the guard still owns the wire.

## Rules it bends, each a DECISIONS line

- The binary never reads stdin: becomes no CLI command reads stdin.
- One JSON object reaches stdout: holds for the CLI path, not for `gdoc mcp`.
- Skills are linked, not copied: desktop chat reads uploaded skill zips, not
  `~/.claude/skills/`. Either the judgement moves into the tool descriptions
  and MCP prompts served from the binary, or the skills are uploaded copies.
  This is likely the largest cost, as it is in `skills-for-openai-codex.md`.
- `comments --wait` does not fit a tool call with a timeout. Left out, and
  live mode in chat is a later item.

## Nail's answers, 2026-10-02

- **Audience:** Nail only at first, installed from the config entry. The
  `.mcpb` bundle and any change to `release.yml` wait.
- **Where the judgement lives:** served by the binary. `gdoc mcp` embeds the
  skill text from `skills/` and serves it as MCP prompts and tool
  descriptions, so there is one source and nothing to upload. The spec has to
  say how the embedded copy stays in step with `skills/` when a skill is
  edited, since an edit is no longer live until the next build.
- **Scope:** the review set plus `propose`, as above.

Size: the adapter is about a day. With the decisions and the skill question it
is an L: brainstorm, spec, plan, then a milestone.
