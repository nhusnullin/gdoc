---
worth: later
added: 2026-10-02
---
# A bridge from Claude Desktop chat to the hub

Nail's idea on 2026-10-02, while designing `gdoc mcp`: a chat or voice review
should reach the hub, not only the document. Version 1 of `gdoc mcp` is
document-only, so an `ai!` in chat gets one answer, that it needs Claude Code,
and an `ai?` is answered from the document alone.

Two things belong here, and they can ship apart.

## Reading the hub

A setting in the extension's form, Hub folders, a folder picker that takes
several folders, empty meaning off. With it, a read-only `notes` tool finds the
note paired with the document and reads notes under those folders, so chat can
answer `ai?` from them. `propose` and `suggestions` could then take the note,
and `withdraw` could come to chat. The `.mcpb` manifest passes the folders as
`--root` arguments. This is the smaller half.

## Carrying out an `ai!`

Three shapes were weighed:

1. **gdoc starts Claude Code.** `gdoc mcp` runs `claude -p` in the hub with the
   task, and a status tool reports when it is done. It breaks "nothing under
   `go/` runs an external program", which is principle 1, needs Claude Code
   installed and signed in, outlasts a tool call so it needs a background job,
   and lets an agent edit hub files unwatched.
2. **The document is the bridge.** A `gdoc-review live` session in Claude Code
   watches the document. Chat leaves the task in the document, the live
   session carries it out in the hub and receipts it in the thread, and chat
   reads the receipt. No program is run and nothing is queued. But a comment
   opening with 🤖 is gdoc's own and never work, which is what stops gdoc
   giving itself instructions. A hand-off from chat needs a narrow exception
   to that rule, and a live session running somewhere.
3. **Claude Desktop does it.** Chat and Claude Code live in one app. If chat
   can one day hand work to a Code session, gdoc needs nothing.

The leaning on 2026-10-02 was shape 2. What settles the value decision, and
why it is `later`: whether reviewing from chat is used enough once `gdoc mcp`
ships to make the hub worth reaching from there, and whether shape 3 arrives
first. It is an L of its own: brainstorm, spec, plan.
