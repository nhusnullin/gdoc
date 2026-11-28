---
id: GD-37
title: The panel cannot see whether Claude Desktop loaded the extension
status: To Do
assignee: []
created_date: '2026-11-28T10:50:00Z'
labels:
  - worth-later
dependencies: []
references:
  - presence/Sources/PresenceApp/Session.swift:354
ordinal: 37000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
HEALTH's Claude Desktop row says one of two things today, and only one of them
is a fact the app can stand behind.

`gdoc update --desktop` writes `gdoc.mcpb` beside the binary and hands it to
Claude Desktop. The app looks for that file in the directory it already
resolved the binary in. No file is proof: gdoc never wrote an extension on this
machine, so Desktop cannot have one, and the row says "not installed" with the
one thing to do about it. A file that is there proves nothing: the person may
never have double-clicked it, and Desktop may have refused it.

One half of that is answered, and only inside one run of the app. `update
--desktop` prints `extension_opened` where it handed the file over, and the
app keeps that answer: a run that wrote the file and could not open it leaves
the row and its button where they were, instead of the file taking the row
away on a machine that has no Claude Desktop at all. A later launch reads the
file again and has nothing to go on.

So there is no "extension connected" row. The blueprint drew one, and it is out
until the app has a way to ask. Two routes, neither measured:

- Read where Claude Desktop keeps what it loaded. Nothing published says where
  that is or what shape it has, and a path read off a guess is a row that lies
  on the next Desktop release. It would need a measurement in
  `docs/v2/MEASURED.md` first, and a fourth place the app reads.
- Have `gdoc mcp` write a line into the record when a chat session starts, and
  read "Desktop ran gdoc as a tool at some point" off that. It is a weaker
  claim, but it is the record's own, and the app already reads it.

Worth `later`: the actionable half is there. A person with no extension is told
so and handed the button; a person with one is told nothing, which is the same
as today and never wrong.

Added 2026-10-17 as docs/backlog/the-panel-cannot-see-the-desktop-extension.md, worth: later.
<!-- SECTION:DESCRIPTION:END -->
