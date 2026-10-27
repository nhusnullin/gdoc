---
worth: later
added: 2026-10-26
---
# Select text in a Google Doc and ask Claude Desktop, without an ai! comment

Nail's idea, 2026-10-26, in the evening. He works a lot in Google Docs and a
lot in Claude Desktop, and wants the two to talk more natively.

Today the route back from a published document is a comment that opens with
`ai?` or `ai!`, handled by `gdoc-review`. What he wants instead:

- Select text in the Google Doc and ask Claude to reformulate it, rename
  something, or give feedback on it, from his own context menu or something
  close to it. No `ai?` or `ai!` comment in the thread.
- The change lands as a plain edit of the text, the way Claude edits a local
  Markdown file. He said it does not have to be a suggestion. That runs
  against the invariant "every change inside a handed-in document is a
  suggestion", so a direct edit is a `docs/v2/DECISIONS.md` entry before any
  code, and the trust guardian's call.
- It works on his laptop only. He never edits on his phone and does not expect
  a phone edit to reach Claude Desktop, not now.
- The way in can be a localhost server on his machine, reached from a Google
  Docs add-on or Apps Script, or from a browser extension that passes the
  selection along.

## What settles the value

This is `later` because the shape is open, not the wish. A brainstorm session
started 2026-10-26 looks at what Google allows: Docs add-ons and their menus,
Apps Script reaching localhost, a browser extension, Claude Desktop's MCP side
through `gdoc mcp`. It comes back with options on a page. The value is
settled when Nail picks one of those options, or none of them.

The brainstorm's page is https://claude.ai/artifact/HWU52BJNXsvtyajHN6vpUF,
and it is private. It recommends Claude Desktop with the Claude in Chrome
connector, calling the existing `gdoc mcp` read and propose. That needs no
change to the binary. Its one open check is whether the connector can read a
selection, because a signed-out spike found neither the text nor the selection
in the Docs page. The open decision is whether one Accept click on a native
suggestion is close enough to an edit.

## Invariants it touches

- Every change in a handed-in document is a suggestion (`GrantInPlace` is the
  one door, and it styles without changing a character).
- The guard owns the wire, and the reachable set has three doors. A selection
  arriving from a browser names a document gdoc was never handed.
- The binary never prompts, and nothing under `go/` runs an external program
  except one `open`.
- A local listener exists today only as the one-shot login redirect in
  `go/internal/auth/loopback/`. A long-lived one is new.
