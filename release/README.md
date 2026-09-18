# gdoc

gdoc is a command line tool that works on Google Docs for you, through Claude
Code. It reads a document, answers the comments you marked for it, and proposes
every edit as a native suggestion you accept or reject yourself. It also
publishes a markdown note into Drive in the Altery house style, and restyles a
document that is already there.

## Install

One line. It downloads the newest release for your machine, checks it against
the published checksum, and copies one binary into `~/.local/bin`:

```
curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash
```

Nothing else lands on your machine. If the install says `~/.local/bin` is not on
your PATH, add the line it prints to `~/.zshrc` and open a new terminal. You can
also download the zip from the releases page, unpack it, and run the
`install.sh` inside it.

## The skills

The binary does the work and three Claude Code skills drive it: `gdoc-review`,
`gdoc-publish` and `gdoc-restyle`.

**In the hub, they are already there.** The hub asks you once to trust its
settings. Say yes, and you have all three, kept up to date for you.

**Outside the hub**, two commands install them:

```
/plugin marketplace add nhusnullin/gdoc
/plugin install altery@gdoc
```

Claude Code asks whether you want them everywhere or in this project only, and
it keeps their update switch in `/plugin`, on the marketplace's own screen.

**Marketplaces were refused.** Your Claude Code does not take them. Install the
skills from the release zip instead, with `--skills global` for every project or
`--skills local` for this one:

```
curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash -s -- --skills global
```

These are copies, so run the same line again after `gdoc update`.

**They were refused and `--skills` changed nothing.** Your Claude Code is locked
to plugins your administrator turns on. Send them this repository and these two
keys for `managed-settings.json`: `extraKnownMarketplaces` naming
`nhusnullin/gdoc`, and `enabledPlugins` turning `altery@gdoc` on.

## Sign in

```
gdoc auth login
```

It prints a URL. Open it, sign in with your `altery.com` account, and gdoc has
the token it needs. The token stays on your machine. Everyone signs in once more
after 2026-09-16: the OAuth secret was rotated that day, so a token issued
before it stops refreshing.

## Three things to try

1. Open a Google Doc, write `ai?` in a comment, and ask Claude Code to handle
   the marked comments in that document.
2. Copy `example/first-note.md` from the zip, change the words, and ask Claude
   Code to publish it into a Drive folder.
3. Give Claude Code a link to a document somebody else wrote and ask for it in
   the Altery house style.

## Updates

Once a day `gdoc help` asks GitHub what is published and prints one line when
yours is behind. It downloads nothing: installing is yours to type.

```
gdoc update              take the newest stable release
gdoc update --check      say what a run would take, and write nothing
gdoc update --rollback   put the binary that was here before back
```

`--major` and `--nightly` are in `gdoc help update`. The skills update through
Claude Code, or through the `--skills` line above.

## Reporting a problem

Open an issue at https://github.com/nhusnullin/gdoc/issues, or message Nail. The
form asks four things: the version from `gdoc help`, the command as you typed
it, the JSON object gdoc printed, and what you expected instead.

## What gdoc never does

It never edits a document you handed it: every change is a suggestion, with your
name on the accept button. It never asks you a question, never reads your
keyboard, never runs git, and never updates itself.
