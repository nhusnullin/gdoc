# gdoc

gdoc works on Google Docs from your terminal, through Claude Code.
Whatever it puts into Drive looks like Altery.
It never changes a word you wrote: every text edit is a suggestion, with your name on the accept button.
Install it in one line below, then publish a note, restyle a document, or review one live.

## Install in one line

```
curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash
```

That downloads the newest release for your Mac, checks it against the published
checksum, and puts one binary into `~/.local/bin`. If it says `~/.local/bin` is
not on your PATH, run the `echo` line it prints and open a new terminal.

Then sign in once with `gdoc auth login`. It prints a URL. Open it with your
`altery.com` Google account. The token stays on your machine.

The second route is a checkout: `git clone https://github.com/nhusnullin/gdoc`,
then `./install.sh` inside it builds the binary and links the skills. It needs
Go and make, so take it only when you want to change gdoc. Details in
[From a checkout](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/from-a-checkout.md).

## Add the skills to Claude Code

gdoc does the work. Three Claude Code skills drive it: `gdoc-review`,
`gdoc-publish` and `gdoc-restyle`. Take the first recipe that works.

**1. Plugins are allowed.** In Claude Code, type these two lines. It asks
whether you want them everywhere or in this project only.

```
/plugin marketplace add nhusnullin/gdoc
/plugin install altery@gdoc
```

**2. Claude Code refused the marketplace.** Install the skills as copies from
the release zip, with `--skills global` for every project or `--skills local`
for this one:

```
curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash -s -- --skills global
```

**3. Your administrator locks plugins.** Send them
https://github.com/nhusnullin/gdoc and two keys for `managed-settings.json`:
`extraKnownMarketplaces` naming `nhusnullin/gdoc`, and `enabledPlugins` turning
`altery@gdoc` on.

## Claude Code runs gdoc for you

You work in Claude Code, in a terminal, with the hub as the working directory.
You ask in plain words and paste the document link. The skill runs gdoc: you
never type a gdoc command to publish, restyle or review.

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
   into the hub and change the words. Then ask, with a Drive folder link:

   > Publish first-note.md into https://drive.google.com/drive/folders/1AbC…

   gdoc builds it in the house style, uploads it and reads it back. It writes
   the document's id into the note's front matter. You see a new Google Doc in
   that folder, in the house style. Then check the three things gdoc cannot
   see: the logo in the header, the contents list and the page numbers.
2. **Restyle a document somebody else wrote.** Ask, with its link:

   > Give https://docs.google.com/document/d/1AbC…/edit the Altery house style

   gdoc surveys the document first. Claude Code shows you the survey and asks
   whether you want the Altery cover page. It asks once more before it sends
   the styling. gdoc then styles the page, the paragraphs, the text and the
   table cells in place. Not one character of the text moves. This styling is
   the one direct edit gdoc ever makes, so the undo is the document's version
   history. The cover arrives as a suggestion. You see the same document in
   the house style, with your comments and suggestions still there. Claude
   Code lists what you finish in Docs yourself.
3. **Review a document live.** Start the session first, with the link:

   > Review https://docs.google.com/document/d/1AbC…/edit live

   Now open the document and write a comment that starts with `ai?`, then your
   question. Claude Code answers in that thread while you watch. Each reply
   opens with 🤖, so the document shows which words are gdoc's. A change to
   the words arrives as a Google suggestion with a comment saying why. Claude
   Code never acts on an unmarked comment. Stop the session with Ctrl-C or by
   saying stop. Without the word "live", the session answers the marks that
   are there now and ends.

## Why gdoc can do what the ordinary Docs API cannot

Google granted this project the Docs API Developer Preview on 2026-08-29. The
preview lets gdoc write a real suggestion and anchor a comment to the exact
words it is about. It also returns every comment with its character range.
Through the ordinary API a "suggestion" lands as a direct edit. A comment
cannot be pinned to a range at all. The preview is pre-GA and may be withdrawn.
gdoc probes it on every run that proposes and reads back every write. A change
of behaviour is reported, never trusted.

## Build your own skill on the binary

The three skills are prompts over one binary. Every command takes arguments,
prints one JSON object and exits. A skill of your own can build on any of them.
`gdoc help <command>` prints the words, the flags and an example.

| Command | Does |
|---|---|
| `auth status`, `auth login` | say whether gdoc has a token and what it may reach, or sign in |
| `read <url>` | print the document as text, with the ids a comment or suggestion is named by |
| `comments <url>` | list the threads with their ranges, replies and the next cursor, or wait for new ones |
| `suggestions <url>` | list the suggestions pending in the document |
| `restyle <url>` | survey a document against the house style, or apply it where the document stands |
| `probe` | create a throwaway document and say whether Docs honours a suggestion today |
| `reply <url> <comment id>` | write one reply into a thread, under the robot prefix |
| `propose <url>` | propose changes as native suggestions, each with the comment that says why |
| `withdraw <url> <suggestion id>` | take back one pending suggestion gdoc proposed itself |
| `build` | build a note as a house-style docx on this machine, with no network |
| `publish` | build a note as a house-style document and upload it into one folder |
| `update` | replace this gdoc with the newest release, or say what one would take |
| `help`, `completion` | print every command, or write the shell completion script |

Every request goes through a guard inside the binary. It carries a request only
for a document gdoc was handed or created, so a skill of yours cannot reach
further than the ids it names.

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
