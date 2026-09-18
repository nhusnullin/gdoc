# gdoc

Whatever gdoc puts into Google Drive looks like Altery.
It never changes a word you wrote: every text edit is a suggestion, with your name on the accept button.
Install it below, then ask Claude Code to publish a note, restyle a document, or review one live.

## Install in one line, then sign in once

The line below downloads the newest release for your Mac and checks it against
the published checksum. It puts one binary into `~/.local/bin` and one zsh
completion file into `~/.config/gdoc-agent`, and nothing else.

```
curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash
```

If it says `~/.local/bin` is not on your PATH, run the `echo` line it prints.
Then open a new terminal. The zip at
https://github.com/nhusnullin/gdoc/releases carries the same `install.sh`.

Then run `gdoc auth login` once. It prints a URL. Open it and sign in with your
`altery.com` account. The token stays on your machine. If you signed in before
2026-09-16, sign in again: the secret changed that day.

## The skills reach you by the first route that fits

gdoc does the work. Three Claude Code skills drive it: `gdoc-review`,
`gdoc-publish` and `gdoc-restyle`. Take the first row that fits you.

| Where you are | What to do |
|---|---|
| Claude Code accepts plugins | In Claude Code, type `/plugin marketplace add nhusnullin/gdoc`, then `/plugin install altery@gdoc`. It asks: everywhere, or this project only. |
| Claude Code refused the marketplace | Run the line below, with `--skills global` for every project or `--skills local` for this one. |
| `--skills` changed nothing | Your Claude Code is locked to the plugins your administrator turns on. Send them https://github.com/nhusnullin/gdoc and two keys for `managed-settings.json`: `extraKnownMarketplaces` naming `nhusnullin/gdoc`, and `enabledPlugins` turning `altery@gdoc` on. |

```
curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash -s -- --skills global
```

## Claude Code runs gdoc for you

You work in Claude Code, in a terminal, with the hub as the working directory.
The hub is the folder that holds Altery's notes. You ask in plain words and
paste the document link. The skill runs gdoc: you never type a gdoc command to
publish, restyle or review.

Nothing happens in a document by itself. A comment that starts with `ai?` asks
a question. The answer lands in that thread. A comment that starts with `ai!`
asks for work in the hub. The thread gets a reply saying where. Claude Code acts
on either mark only while a review session runs on that document. The session
is the Claude Code conversation you asked in. Ask once, and it handles the marks
that are there now. Say "live", and it watches until you stop it with Ctrl-C or
by saying stop. Close the session and the marks wait.

## Try one of these first

1. **Publish a note.** Save
   https://github.com/nhusnullin/gdoc/blob/main/release/example/first-note.md
   into the hub and change the words. Paste a Drive folder link and ask Claude
   Code to publish `first-note.md` into that folder. gdoc builds it in the house style,
   uploads it, reads it back and writes the document's id into the note. You see
   a new Google Doc in that folder, in the house style. Then check the three
   things gdoc cannot see: the logo in the header, the contents list and the
   page numbers.
2. **Restyle a document somebody else wrote.** Paste its link and ask for it in
   the Altery house style. gdoc surveys the document first. Claude Code shows
   you the survey. It asks whether you want the Altery cover page, and asks once
   more before it sends the styling. gdoc then styles the page, the paragraphs,
   the text and the table cells in place. Not one character of the text moves.
   This styling is the one direct edit gdoc ever makes, so the undo is the
   document's version history. The cover arrives as a suggestion. You see the
   same document in the house style, with your comments and suggestions still
   there. Claude Code lists what you finish in Docs yourself.
3. **Review a document.** Open a Google Doc you can comment on and write a
   comment that starts with `ai?`, then your question. In Claude Code, paste the
   link and ask it to handle the marked comments. Add the word "live" to keep it
   watching. Claude Code answers each `ai?` in its own thread. Each reply opens
   with 🤖, so the document shows which words are gdoc's. A change to the words
   arrives as a Google suggestion with a comment saying why. Claude Code never
   acts on an unmarked comment. You see the reply in the thread and the
   suggestion waiting for you to accept or reject.

## Nothing updates on its own

Once a day `gdoc help` asks GitHub what is published and prints one line when
yours is behind. It downloads nothing: installing is yours to type.

| Command | Does |
|---|---|
| `gdoc update` | takes the newest stable release |
| `gdoc update --check` | says what a run would take, and writes nothing |
| `gdoc update --rollback` | puts the binary that was here before back |

`--major` and `--nightly` are in `gdoc help update`. Plugin skills update
through Claude Code. Skills from the `--skills` line update when you run it again.

## A problem report needs four things

Open an issue at https://github.com/nhusnullin/gdoc/issues, or message Nail.
Send your version from `gdoc help`, the gdoc command that ran, the JSON gdoc
printed, and what you expected instead.

## What gdoc never does

- It never changes a word of a document you handed it. Every text edit is a suggestion.
- It sees only the documents you name, and cannot search your Drive.
- gdoc itself never asks you a question or reads your keyboard. Claude Code asks you before it signs in and before it restyles.
- It never runs git.

## The mechanism is on its own pages

| Page | Holds |
|---|---|
| [How it works](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/how-it-works.md) | the credential, the guard, suggestions only, the JSON envelope |
| [From a checkout](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/from-a-checkout.md) | `install.sh`, `make build`, `make dist`, completion, `GDOC_CONFIG_DIR` |
| [Reading](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/reading.md) | `read`, `comments`, `suggestions`, the cursor, `--wait` |
| [Writing](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/writing.md) | `probe`, `reply`, `propose`, `withdraw` |
| [Restyle](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/restyle.md) | the survey, restyle in place, the house template as a suggestion |
| [Publishing](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/publishing.md) | `build`, `publish`, the `gdoc:` block |
| [What is planned](https://github.com/nhusnullin/gdoc/blob/main/docs/v2/PLAN.md) | the open work |
