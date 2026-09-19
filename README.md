# gdoc

gdoc works on Google Docs from your terminal, through Claude Code.
Whatever it puts into Drive comes out in the house style, and it never changes a word you wrote: every text edit is a suggestion, with your name on the accept button.
Install it in one line below, then publish a note, restyle a document, review one live, or bring one into your hub.

## Install in one line

```
curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash
```

That downloads the newest release for your Mac, checks it against the published
checksum, and puts one binary into `~/.local/bin`. If it says `~/.local/bin` is
not on your PATH, run the `echo` line it prints and open a new terminal. Then
sign in once with `gdoc auth login`. It prints a URL: open it with your work
Google account, and the token stays on your machine.

The second route is a checkout: `git clone https://github.com/nhusnullin/gdoc`,
then `./install.sh` inside it builds the binary and links the skills. It needs
Go and make, so take it only when you want to change gdoc. Details in
[From a checkout](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/from-a-checkout.md).

## Add the skills to Claude Code

gdoc does the work. Five Claude Code skills drive it: `gdoc-review`,
`gdoc-publish`, `gdoc-restyle`, `gdoc-export` and `gdoc-align`. Take the first
recipe that works.

**1. Plugins are allowed.** In Claude Code, type these two lines. It asks
whether you want them everywhere or in this project only.

```
/plugin marketplace add nhusnullin/gdoc
/plugin install altery@gdoc
```

**2. Claude Code refused the marketplace.** Copy the skills from the release
zip, with `--skills global` for every project or `--skills local` for this one:

```
curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash -s -- --skills global
```

**3. Your administrator locks plugins.** Send them
https://github.com/nhusnullin/gdoc and two keys for `managed-settings.json`:
`extraKnownMarketplaces` naming `nhusnullin/gdoc`, and `enabledPlugins` turning
`altery@gdoc` on.

## Claude Code runs gdoc for you

You work in Claude Code, in a terminal, with the hub as the working directory.
Each skill is a slash command: type its name, then the request in plain words
with the link. The skill runs gdoc: you never type a gdoc command to publish,
restyle, review, export or align. Skills that came as the plugin carry its
prefix, so `/gdoc-review` is `/altery:gdoc-review` there.

Nothing happens in a document by itself. A comment that starts with `ai?` asks a
question, and the answer lands in that thread. A comment that starts with `ai!`
asks for work in the hub, and the thread gets a reply saying where. Claude Code
acts on either mark only while a review session runs on that document, and that
session is the conversation you asked in. Ask once, and it handles the marks
there now. Say "live", and it watches until you stop it. Close the session and
the marks wait.

## Try one of these first

1. **Publish a note.** Save [first-note.md](https://github.com/nhusnullin/gdoc/blob/main/release/example/first-note.md)
   into the hub, change the words, then call the skill:

   ```
   /gdoc-publish first-note.md into <url of a folder in Google Drive>
   ```

   gdoc builds it in the house style, uploads it and reads it back, and writes
   the document's id into the note's front matter. Then check what gdoc cannot
   see: the logo in the header, the contents list and the page numbers.
2. **Restyle a document somebody else wrote.** Call the skill with its link:

   ```
   /gdoc-restyle <url of the Google Doc> with the cover
   ```

   gdoc surveys the document first. Claude Code shows you the survey and asks
   once more before it sends the styling. gdoc then styles the page, the
   paragraphs, the text and the table cells in place. Not one character moves.
   This styling is the one direct edit gdoc ever makes, so the undo is the
   document's version history. The cover arrives as a suggestion, and leaving
   out "with the cover" styles the body alone.
3. **Review a document live.** Start the session first, with the link:

   ```
   /gdoc-review live <url of the Google Doc>
   ```

   Now open the document and write a comment that starts with `ai?`, then your
   question. Claude Code answers in that thread while you watch. Each reply opens
   with 🤖, so the document shows which words are gdoc's, and a change to the
   words arrives as a Google suggestion with a comment saying why. It never acts
   on an unmarked comment. Stop the session with Ctrl-C or by saying stop.
4. **Bring a document into the hub.** Call the skill with its link:

   ```
   /gdoc-export <url of the Google Doc>
   ```

   gdoc writes the document into your hub as Markdown, one file per tab, with its
   pictures as PNG files in `assets/` beside it. Nothing changes in Drive, and
   nothing on disk is replaced: a taken name takes the next free number. Pending
   suggestions and comments arrive as marks in the text, and Claude Code resolves
   them with you. For a note already paired with that document, say "align":
   `/gdoc-align` merges the two with your agreement.

## Why gdoc can do what the ordinary Docs API cannot

Google Docs has a public API and a Developer Preview: a set of extra requests
Google switches on for one project at a time, on application, and it granted
this project the preview on 2026-08-29. Three things gdoc does need it.

| What you see in the document | Through the public API | With the preview |
|---|---|---|
| An edit as a suggestion, with an accept button | lands as a direct edit, no button | a real suggestion |
| A comment on the exact words it is about | lands on the document as a whole | anchored to those words |
| Which words each comment is about | not returned | returned with every comment |

The preview is not a general release. Google may change it or withdraw it, and
its terms say so. If it goes, gdoc loses proposing in the document, though
replies in existing threads survive on the public API. So gdoc trusts nothing:
before it proposes, it creates a throwaway document and checks that a suggestion
lands as one, and after every write it reads the document back.

## Build your own skill on the binary

The five skills are prompts over one binary. Every command takes arguments and
prints one JSON object, so a skill of your own can build on any of them. `gdoc
help <command>` prints the words, the flags and an example.

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
| `annotate <url>` | leave a comment on the exact words you quote, changing nothing |
| `export <url>` | write the document into the hub as Markdown, one file per tab, with its pictures |
| `build` | build a note as a house-style docx on this machine, with no network |
| `publish` | build a note as a house-style document and upload it into one folder |
| `update` | replace this gdoc with the newest release, or say what one would take |
| `help`, `completion` | print every command, or write the shell completion script |

Every request goes through a guard inside the binary, which carries a request
only for a document gdoc was handed or created. A skill of yours cannot reach
further than the ids it names.

## Nothing updates on its own

Once a day `gdoc help` asks GitHub what is published and prints one line when
yours is behind. It downloads nothing: installing is yours to type.

| Command | Does |
|---|---|
| `gdoc update` | takes the newest stable release |
| `gdoc update --check` | says what a run would take, and writes nothing |
| `gdoc update --rollback` | puts the binary that was here before back |

`--major` and `--nightly` are in `gdoc help update`. Plugin skills update through
Claude Code, copied skills when you run the `--skills` line again.

## A problem report needs four things

Open an issue at https://github.com/nhusnullin/gdoc/issues, or message Nail. Send
your version from `gdoc help`, the command that ran, the JSON it printed, and
what you expected instead.

## What gdoc never does

- It never changes a word of a document you handed it. Every text edit is a suggestion.
- It never replaces a file in your hub. A taken name takes the next free number.
- It sees only the documents you name, and cannot search your Drive.
- gdoc itself never asks you a question or reads your keyboard. Claude Code asks you before it signs in and before it restyles.
- It never runs git.

## The mechanism is on its own pages

| Page | Holds |
|---|---|
| [How it works](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/how-it-works.md) | the credential, the guard, suggestions only, the JSON envelope |
| [From a checkout](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/from-a-checkout.md) | `install.sh`, `make build`, `make dist`, completion, `GDOC_CONFIG_DIR` |
| [Reading](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/reading.md) | `read`, `comments`, `suggestions`, the cursor, `--wait` |
| [Writing](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/writing.md) | `probe`, `reply`, `propose`, `withdraw`, `annotate` |
| [Restyle](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/restyle.md) | the survey, restyle in place, the house template as a suggestion |
| [Publishing](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/publishing.md) | `build`, `publish`, the `gdoc:` block |
| [Exporting](https://github.com/nhusnullin/gdoc/blob/main/docs/guide/exporting.md) | `export`, the numbering rule, `assets/`, the marks |
| [What is planned](https://github.com/nhusnullin/gdoc/blob/main/docs/v2/PLAN.md) | the open work |
