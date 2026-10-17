# The presence: one glance at what gdoc is doing

A live review is hours of quiet. The chat window says nothing, the document
says nothing, and a run that has finished looks exactly like a run that died.
The presence answers two questions from the menu bar, without you asking
anything: is it alive, and is it waiting on me.

It is two halves. gdoc writes one line for each thing it starts and finishes.
A small application called gdoc presence reads those lines and shows one glyph.
Nothing goes over the network, nothing leaves your machine, and gdoc behaves
the same whether the app is open, closed or never installed.

This is experimental. It is off until you turn it on, and turning it on is one
line in a file you own, so your colleagues keep the gdoc they had.

## What the glyph means

The glyph sits in the menu bar, in the bar's own colour, and never in colour of
its own. It has four states, and the one it shows is the first of these that is
true:

| Glyph | Means |
|---|---|
| A pause sign | A chat write is held, waiting for you to say yes or no |
| A dotted ring | A command is running: a read, a publish, a proposal |
| ⁘ | A live review is watching a document for new comments |
| Nothing | Nothing is happening that the record knows about |

A pause sign is the one state that is about you. The other three are about
gdoc. Rest the pointer on the glyph and it says the state in a word.

Click it and a panel opens under it. NOW is what is running this second, with
the held writes first. TODAY is what ran since midnight, newest first, each row
marked by what its end line said. Under them sit two version rows, the app's
own and the gdoc that wrote the last line, and Quit. Press Escape or click
elsewhere and the panel closes.

A run that stopped without an end line, because the terminal was closed or the
machine slept, is shown as its own row with an unknown outcome. A held write
whose thirty minutes passed with nobody answering it is shown as a row saying
it ran out. Neither is ever a glyph: an icon that stays lit after the work
stopped is an icon nobody believes.

## Turning it on

The order matters, because the flag has to exist in the gdoc you are running
before you write it into a file that gdoc reads.

1. `gdoc update`, so you are on a release that knows the key. A release older
   than the flag refuses a settings file that carries it, by name, and refuses
   it on every command.
2. Write one line into `~/.config/gdoc-agent/settings.yaml`:

   ```yaml
   presence: true
   ```

3. `gdoc update` again. With the flag on, this is the run that installs the app
   at `~/Applications/GdocPresence.app`, out of the same release zip the binary
   came from. It tells you the path and asks you to open it once.
4. `open ~/Applications/GdocPresence.app`. The first launch is yours. gdoc
   never starts the app, which is why it says the line instead.

The app has no window, no dock icon and no settings. It is the glyph and the
panel.

For one run only, without writing anything down, pass the argument instead:

```
gdoc read <link> --presence true
```

The argument takes `true` or `false` and nothing else, and it beats the file
either way. The file is more forgiving, because a line you keep is a line you
edit once: `presence: True` is read as true there. `presence:` with nothing
after it is refused, with the line to write instead.

## Keeping it updated, and taking it back

Every later `gdoc update` refreshes the app along with the binary, out of the
one zip it already verified. If the app is open while that happens, it notices
and shows a row saying `Updated to <version>. Quit and reopen`. Quit it and
open it again; nothing else is needed, and nothing is lost.

If the app cannot be written, the update says so as a warning and names both
paths. The binary is still updated. The two can sit at different versions until
a later run, and the app's version row is what tells you.

An app left far enough behind stops reading the record at all: a later gdoc may
write a line in a shape this app was never taught, and it skips every one of
them rather than guessing. When that happens the panel says `gdoc is newer than
this app` and the sections under it go quiet. Run `gdoc update` again, or
reinstall the app, and both come back.

`gdoc update --rollback` puts the earlier binary back, and the earlier app with
it. There is one trap in that, and it is worth reading twice. A rollback to a
release older than this flag leaves `presence: true` in your settings file, and
that older gdoc does not know the key, so it refuses every command until you
take the line out. If you roll back that far, delete the line first.

Turning the flag off, by removing the line or by `--presence false`, stops the
writing at once. It uninstalls nothing and deletes nothing: the app stays where
it is, with nothing new to show, and what earlier runs wrote waits for you.

## The file, and what is never in it

gdoc appends to `presence/<yyyy>-<mm>.jsonl` in the config directory, beside
the token. One file per month, one line per event, readable by you alone. gdoc
keeps this month and last month and removes the older ones the next time it
writes. A note or a backup you left in that folder is left alone.

Each line is a short, fixed list of facts: when, which process, which command,
whether it came from the terminal or from a chat, the document id, and at the
end whether it worked. A hold adds its own id, which rule held it and when it
runs out.

What is never in a line: a word of the document except its title, a comment or
a reply, a quote, the text of a proposal, the text or the value of a held
write, an error sentence, a file path, your account, anything about the token,
and nothing that judges anything. The title is the one document word, cut at
200 characters, and it is there so the panel can name a row without running a
second gdoc to read the document again.

gdoc never reads the file back. Nothing it decides depends on a line in it: not
a hold, not a reply, not where a review had got to. That is the whole safety of
the design, and it is why a file you edited, a half-written line, or no file at
all changes nothing about what the next command does.

If writing the line fails, gdoc says so once on the error stream and carries on.
The record is never worth a failed command.

Local disks only. Keep your config directory on this machine's own disk. A
folder synced by Dropbox, iCloud or a network share can tear a line in half,
because the promise that two gdoc runs can append to one file at the same time
is the local filesystem's and nobody else's.

## What macOS says about the app

The app is signed ad hoc, which means it is signed with no developer identity
behind it. Two things follow, and both are expected.

Asking Gatekeeper about it directly says no:

```
spctl -a -t exec -vv ~/Applications/GdocPresence.app
# rejected
```

That is what `spctl` says about every ad-hoc bundle, and it is not the verdict
on this one. macOS only consults Gatekeeper for software that arrives carrying
a quarantine flag, from a browser or a mail client. gdoc writes the app out of
a zip it checked itself, in Go, and sets no such flag, so the app opens when you
open it. The signature is still real and still checkable:

```
codesign --verify --deep --strict -vv ~/Applications/GdocPresence.app
```

The second thing is that the app is yours and this machine's. It is not
notarized, it is not in the App Store, and it is not something to hand to a
colleague by AirDrop. They get it the same way you did: turn the flag on, run
`gdoc update`.

## What it is not

The app reads the file and draws the glyph. That is all it does. It runs no
program, not gdoc and not anything else, it opens no connection, and it writes
nothing down. It cannot read a document, answer a comment, accept a held write
or stop a review, and there is no button in it that pretends otherwise. Those
stay where they were: in Claude Code, in the chat, or in the terminal. The
panel tells you a chat is waiting for you, and going to the chat is your step.
