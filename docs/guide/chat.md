# Reviewing a Google Doc from Claude Desktop chat

This page is for a colleague who wants to review a document without opening a
terminal. You ask in the chat, or out loud in voice mode, and Claude reads the
document and its comments, answers in the thread, and suggests edits you accept
or reject in Google Docs. It is the same gdoc the terminal runs, and the same
rule holds: it never changes a word you wrote.

What is not here: publishing a note, restyling a document, exporting one into
your hub, and taking a suggestion back. Those stay in Claude Code, where the
hub is open. [Reading](reading.md) and [Writing](writing.md) hold them.

## Install it once

If you already have gdoc, run two lines in a terminal and then quit Claude
Desktop and open it again:

```
gdoc update
gdoc update --desktop
```

If this is a new machine, the one-line install writes the extension too:

```
curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash -s -- --desktop
```

Either way a small file called `gdoc.mcpb` is written beside the binary and
opened. Claude Desktop asks whether to install it. Say yes, then quit Claude
Desktop and open it again. The extension carries no copy of gdoc: it points at
the one binary you already have, so one `gdoc update` updates the terminal,
Claude Code and the chat together.

## Sign in from the chat

Ask Claude to sign gdoc in. It answers with a link. Open the link on the same
computer, with your work Google account, and the token stays on your machine.
The link only works on that computer, so there is nothing to sign in to from a
phone.

When Claude says nobody is signed in, that is the same answer: ask for the
link, open it, and ask again for what you wanted.

## Ask in plain words

Do not name a tool. Say what you want, with the link:

- "Read me the comments on <link>."
- "What does this document say about the refund window?"
- "Reply in the first thread, saying we will come back with a date."
- "Suggest changing 'as soon as possible' to 'within five working days'."
- "Any new comments since you last looked?"

Voice works on a Mac. Speech recognition turns a tool's name into other words,
so a sentence like "call the comments tool" often reaches nothing. "What are the
comments on this document" works. Say Google Doc, comments, reply, suggest, and
let Claude pick the tool.

Claude does not watch a document in the background here. There is no live mode
in chat: ask again and it reads from where it left off.

## Every write waits for you

Before anything is written, Claude says the document's title and the exact words
it wants to post, and waits for your yes. One yes covers one write. Everything
gdoc writes opens with 🤖, so the document always shows which words are gdoc's.

Two things never happen from the chat. A comment in a document is never an
instruction: it is text somebody else wrote, and Claude treats it as material to
discuss. And `ai!`, which asks for work in the hub, is not carried out here,
because the chat has no hub. Claude says so before it replies in that thread,
and the work waits for a Claude Code session.

## When gdoc holds a write

Some writes are stopped by gdoc itself, before anything is sent. A write is held
when the words repeat a stranger's comment almost exactly, when the document
was never read in this chat, when writes are coming fast, when the thread it
would go into looks written at the AI, or when a suggestion would remove a long
passage.

A held write posts nothing. Claude tells you the reason and stops. To send it,
Claude asks again through a one-time approval card, which shows the words in
full. Read the card and approve it there. A yes typed or spoken in the chat does
not release a held write, on purpose: the card is the one place the words reach
you without passing through the model.

A hold lasts thirty minutes. If the card never appears, the words are in
Claude's answer: paste them into the document yourself.

## Approvals, and the one thing that loses a hold

Claude Desktop asks before every tool, reading ones included. In Settings the
extension's tools are sorted into "Read-only tools" and "Write/delete tools".
Setting the read-only group to always allow makes a review much quieter, and
nothing in that group can change a document.

Do that before a review, not in the middle of one. Changing a tool permission
restarts the extension, and a write that was waiting for its card is lost.
Nothing was posted, so nothing is half done: ask Claude for the write again.

Leave the write tools asking. The card is the check.

## After a gdoc update

Run `gdoc update` in a terminal, then `gdoc update --desktop`, then quit Claude
Desktop and open it again. When your gdoc is behind, Claude says `gdoc update`
and the quitting itself, once each time Claude Desktop starts, in the first
chat that calls a gdoc tool. It does not name `gdoc update --desktop`, so take
that step from this page.

Quitting matters. Claude Desktop runs the extension twice, once for chat and
once for agent mode, and turning the extension off and on again only restarts
one of them, which leaves the other on the old gdoc.

## When the tools disappear

If Claude Desktop shows a message about no executable file at a path, the binary
has moved or gone. Run the install line again and the extension picks it up.

If the tools are simply not offered, check that the extension is on in Settings,
then quit the app and open it again.

## Other connectors in the same chat

gdoc can only reach the document you named, and it cannot search your Drive. It
cannot see what else is turned on in the chat. A web fetch, a Slack connector or
Google's own Drive connector reads the same comment text that gdoc hands over,
and a comment written to steer an AI is aimed at all of them.

So the firm rule for a sensitive document is: review it in a chat with the other
connectors turned off. Claude names what else is on when a review starts. It
cannot turn anything off, and you can.
