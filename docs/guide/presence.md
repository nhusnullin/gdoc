# The presence: one glance at what gdoc is doing

A live review is hours of quiet. The chat window says nothing, the document
says nothing, and a run that has finished looks exactly like a run that died.
The presence answers two questions from the menu bar, without you asking
anything: is it alive, and is it waiting on me.

It is two halves. gdoc writes one line for each thing it starts and finishes.
A small application called gdoc presence reads those lines, shows one glyph,
and opens one panel under it. The app opens no connection of its own: the only
thing in it that reaches the network is gdoc, which the panel runs the way you
would run it at a terminal. What gdoc answers is the same whether the app is
open, closed or never installed. The one thing gdoc does about the app is open
it: with the flag on, a command that is recorded opens the app it installed
where that app is not already running.

This is experimental. It is off until you turn it on, and turning it on is one
line in a file you own, so your colleagues keep the gdoc they had.

## What the glyph means

The glyph sits in the menu bar, in the bar's own colour, and never in colour of
its own. It is the mark, with a small badge beside it. It has six states, and
the one it shows is the first of these that is true:

| Glyph | Means |
|---|---|
| The mark with an exclamation | A write stopped, the sign-in does not work, or gdoc was not found |
| The mark with a pause sign | A chat write is held, waiting for you to say yes or no |
| The mark with a ring | A command is running: a read, a publish, a proposal |
| The mark alone | A live review is watching a document for new comments |
| The mark with a check | A write read itself back in the last five seconds |
| The dim mark | Nothing is happening that the record knows about |

The check is about a moment rather than a state, and it is shown only for a
write that finished while the app was running: a line already in the file when
the app opened is a replay, and a check for last Tuesday's reply is a check
nobody asked for. The dim mark is there so the panel stays reachable in a quiet
hour.

The exclamation and the pause sign are the two states that are about you. The
other four are about gdoc. Rest the pointer on the glyph and it says the state
in a word.

## The panel

Click the glyph and the panel opens under it. It is 320 points wide and never
taller than 600, with the long section scrolling inside. Press Escape or click
elsewhere and it closes. Nothing in it is a window, there is no dock icon, and
the only settings in it are the switch for launching at login and Quit.

The sections are always in this order, except that Settings comes first on the
two screens where Settings is the whole point: no token at all, and a sign-in
in flight. The sign-out question is asked in the header, which is above every
section already.

### Now

What is happening this second, and nothing else. Holds first, then writes that
stopped, then running jobs, then live watches, oldest first inside each group.
A section with nothing in it is left out rather than labelled.

- **A hold** is a chat write waiting for you. The card says how long it has
  been waiting, which rule held it, where the words are, and who they would be
  posted as. Its two buttons are Open chat and Dismiss. The panel never
  approves a held write and has no button that pretends to: the yes or the no
  is yours, in the chat, and the card says so.
- **A write that stopped** is a reply, proposal, withdrawal, comment, publish,
  restyle or export whose end said it failed, or whose process is gone. The
  card names the document and why it stopped, in gdoc's own word for it. Where
  gdoc wrote no reason, which is a run whose process went away, the card says
  when it started and which terminal or session to look in. It goes when the
  same command starts again on that document, or when you dismiss it. A failed read is not one of
  these, and `verified: false` is not either: a read-back that did not confirm
  is a hollow check, not a failed write, and the row says "not read back".
- **A running job** is a command with an end that has not arrived yet.
- **A live watch** is the stretch of `gdoc comments --wait` runs on one
  document, with gaps under three minutes counted as one watch. A wait you
  stopped ends the watch where that wait ended, because no poll follows one.
  The row says
  since when, how long it has been quiet, and how many replies and proposals
  went out. It never says anybody is waiting for the author. Its one icon is
  Stop. A watch whose poll Google refused the token on is over, and it stays
  here reading "Live stopped" with no Stop icon until the sign-in works
  again: nothing is polling that document, and starting it again is yours.
- **A sign-in that stopped working** raises its own card here, under the
  holds. It is the one card in Now that comes from the machine and not from
  the record, and it says how many watches that sign-in stopped.

### Documents

Every document gdoc touched in the last fourteen days, newest action first,
five rows with Show more under them. A read alone makes no row: reading is not
something that happened to the document. Each row says what the last action
was, when, and where the note paired with it is, or "not in the hub". The
label is "3 today" where every row in the list is from today, and
"14 days · 12" where one of them is older, counting the rows behind Show more
as well.

Four icons sit on the right of every document row, always visible:

| Icon | Does |
|---|---|
| Google Docs | Opens `https://docs.google.com/document/d/<id>/edit` in your browser |
| md | Opens the note paired with the document in Obsidian |
| Branch | Brings Claude Desktop to the front |
| Link | Copies the document's link, and turns into a green check for a second |

An icon with nothing to do is greyed, and resting the pointer on it says why:
"No Google Doc yet", "No link yet", "No note is paired with this document",
"Note moved or deleted", "Install the Desktop extension in Settings". The first
two are one case: a row whose id is not one Drive could have written has no
document to open and no link to copy. "Note moved or deleted"
is the one a click produces: Obsidian refused the note the index still names.
A greyed icon cannot be clicked, so that one lasts as long as the panel is up,
and the next time you open the panel it is live again. The first time Documents
has a row, four labels appear under it once, naming the icons, and they go the
first time you click one.

Continue with Claude activates Claude Desktop and nothing more: no argument, no
file and no URL. Claude.app does claim a `claude:` scheme, which was measured,
but nothing in the bundle and nothing published says what a URL under it means,
so a URL built on a guess is one that may do something else on the next
release. Until that is written down, the panel brings the app to the front and
you type.

### Today and Yesterday

Today is the local day the calendar names, newest first, eight rows with Show
more. Reads are left out. Besides the writes it carries four rows the record
does not write as single lines: a hold raised, a hold closed, a live watch
started and a live watch stopped, the last of those with what the watch came
to. Yesterday is one row with the counts, and it opens in place. Nothing older
than yesterday is in this list; Documents is where the fortnight is.

### Settings

The last section, where you look for it. It was called Health until the second
morning, and three of its rows left with the name. The account and the gdoc
version are in the header now, each with its one action beside it, and the
Thread mark row is gone: the mark is drawn in that header, at the panel's top
left, so the row said the same thing twice.

What is left is Network, Claude Desktop, Launch at login and Quit. The first
two are drawn only where a fact carries them, so on a good machine the group
is the switch and Quit. Quit is always there, and it is ⌘Q.

Network is the record's own word and never a connection the app opened: gdoc
writes `network` on a run that could not reach Google, and the row says so
until a later run works. Two ends are read past, because neither is about
Google: `auth logout` opens no connection, and `update` reads GitHub. There is
no Skills row: nothing gdoc prints names a skills version, and the app does not
read your `~/.claude`.

Claude Desktop appears only while there is no extension to load. The app looks
for `gdoc.mcpb` beside the binary, which is where `gdoc update --desktop`
leaves it: no file means gdoc never wrote one, so the row offers Install
extension. A file that is there says nothing about what Desktop loaded, so the
row goes and the panel claims nothing. The one exception is a run of Install
extension that wrote the file and could not hand it over, which is a machine
with no Claude Desktop on it: the row and its button stay for as long as the
app is running.

The header carries the installed version after the name, and the update icon
beside it when gdoc's own daily check heard of a newer release. The icon says
Update and not Check for updates, because this app never asks GitHub anything:
see what the app runs, below.

## The actions, one by one

Every action in the panel is something you could have done yourself: a link you
could have opened, or a command you could have typed. The one exception is
Dismiss, which hides a card the app itself drew and changes nothing else.

### Dismiss

Dismiss is on a hold card and on a card for a write that stopped. It hides the
card and rewrites nothing. The row in Today stays. A dismissed hold goes on
living in the chat until its thirty minutes run out, and the card says that
before you press it. The app remembers the dismissal under the run and the call,
or under the hold's own id, for sixty-two days, and never the document.

### Stop live

Stop sends one SIGINT to one `gdoc comments --wait`, which traps it and still
writes its end. That is the whole reason Stop can exist. It is sent once, never
twice, and never as SIGKILL: a second signal is a stop nobody trusts, and a
stronger one takes the end away. Between two polls there is no process, so there
is nothing to signal and the icon is off.

Before it signals anything, the app asks the kernel about that pid and refuses
unless every claim agrees: the start has no end, the process is alive, its
executable is the gdoc the app itself resolved (or that binary's `.previous`),
it belongs to you, and it started within ten seconds before the line that named
it. Anything else greys the icon with one sentence, "This watch is not one gdoc
presence can stop". A pid is a number out of a file anybody can write, so
nothing here believes the file.

What Stop cannot do: it cannot end a review session in Claude Code, only the
wait that session is sitting in. The skill sees the interrupted wait, prints its
totals and stops. It cannot stop a chat write, a publish, or anything that is
not a wait. And it cannot stop a watch started by another user, by another gdoc,
or before this app could see it.

### Sign in

Sign in runs `gdoc auth login` as the app's own child. That command prints one
link to its standard error and waits three minutes for the browser to come back
on its own loopback listener. The app takes the link from that child and from
nowhere else, after the sentence the command prints before it, and then holds it
to one scheme, one host and one path: `https://accounts.google.com/o/oauth2/auth`.
A child that printed anything else opens nothing at all.

The link is opened for you and also shown as a line you can copy, with the
child's own countdown beside it and Cancel next to that. The countdown is the
three minutes gdoc waits, not a guess at how long Google will take. Cancel
terminates the child.

### Sign out

Sign out asks first, inline, and the question names what will stop: the live
watches and the holds, counted, or "Nothing is running. Only the token goes."
The question waits for you: Keep puts it away, and so does closing the panel.
Say yes and the app stops those watches, runs `gdoc auth logout`, and asks
`gdoc auth status` again. `auth logout` removes the token file and nothing
else.

Whose account it is decides nothing anywhere in gdoc, so the address in the
header is a fact and not a gate. The one place the sign-in does decide something is a held
chat write: a hold remembers the sign-in it was raised under, as a hash of the
refresh token and never an email, and a hold whose sign-in changed is dropped
rather than sent. Sign out, then sign in as somebody else, and the words you
held yesterday are not posted under the new account.

"Account changed" is the app's own memory of the addresses it has read, never a
line in the record: the record carries no account, and a line in it would be
forgeable.

### Launch at login

It is off until you turn it on. No run of the app registers anything by
itself: not the first run ever, not a relaunch after an update, and not a
launch that notices macOS dropped the item. The switch in Settings is the one
route, and it is yours both ways: on registers, off unregisters, and an off is
never turned back on. A registration nobody asked for is a notice in somebody's
menu bar nobody asked for.

The switch shows what macOS says, read each time the panel opens. Remove the
item in System Settings and the switch reads off here, with nothing putting it
back.

Where macOS wants you to allow it, the row says "macOS is waiting for you to
allow it in Login Items" and offers to open that settings pane. It is not
retried.

### Update, and Install extension

Update runs `gdoc update` with no flag at all: what you would have typed, and
nothing the app chose on your behalf. While the child is up the gdoc row draws a
ring and still names the version that is installed, because a row that said the
new number before the bytes were in place would be a row that lied for a few
seconds. Install extension is `gdoc update --desktop`, which writes the Claude
Desktop extension and hands it to Desktop. Whether Desktop then loads it is
not something the app is told; whether the handing over happened is, and that
is what keeps the row.

## What the app runs, and what it never does

The app runs six command lines and no seventh:

| Run | When |
|---|---|
| `gdoc help --json` | On panel open, at most once a minute |
| `gdoc auth status` | On panel open, at most once a minute, and every ten minutes while a live watch is open |
| `gdoc auth login` | Sign in |
| `gdoc auth logout` | Sign out |
| `gdoc update` | Update |
| `gdoc update --desktop` | Install extension |

The set is a closed list in one file, and a test over the sources holds it at
six. There is no flag in it the app invented, and no setting of its own.

`help --json` is asked on panel open and never on a timer, because that command
carries gdoc's own once-a-day check of what is published. A timer here would be
this app asking GitHub on gdoc's behalf. The two questions are also asked again
straight after the app signs in, signs out, or updates the binary, because the
last answer is then known to be wrong.

The binary is found at `~/.local/bin/gdoc` first and beside the app's own bundle
second, resolved to an absolute path before anything runs, and never from PATH:
a PATH is written by a shell, a launch agent or an installer, so a gdoc found
through one is a program somebody else chose. A child gets the null device for
standard input and four variables: HOME, PATH, LANG and NO_COLOR. Everything
else is dropped, `GDOC_CONFIG_DIR` included, so the app asks about the token your
own terminal uses. One child per kind at a time.

What the app never does, and no button in it offers:

- It opens no connection. Every byte that leaves this machine leaves it through
  gdoc, under a command you can read above.
- It writes nothing into the record, ever. The record is the binary's file.
- It approves no held write, and it releases none.
- It reads no document, answers no comment and proposes nothing.
- It runs no program but those six, and signals nothing but one wait.
- It never starts gdoc on a schedule of its own.

What it remembers between launches is six things, in its own preferences: the
cards you hid, the hub folder, whether the four icon labels have been read, your
launch-at-login answer, its last self-relaunch, and the last address it saw
signed in. The two that are about a moment, the hidden cards and the relaunch,
are dropped once they are older than sixty-two days, because the app reads two
month files and a stamp older than that is about a line it can no longer see.
The other four are answers and are kept until you change them, because your
answer is yours. The launch-at-login one drives nothing: what starts the app at
login is the login item macOS holds, and the switch shows what macOS says. A
seventh key is a decision somebody writes down, not a line somebody adds.

## The note beside the document

Open md needs no field in the record. What pairs a note with a document is the
`gdoc:` block gdoc writes into that note's front matter, so the app reads that
one block of every note in your hub, builds a map from document id to note path,
and rebuilds it when files there change. Nothing else in a note ever reaches the
app: the reading stops at the closing fence, a `gdoc:` block inside a fenced
example pairs nothing, and every id is held to the Drive shape.

The hub is a folder you name, and the app has no picker for it yet. Set it once
from a terminal, then quit the app and open it again:

```
defaults write com.altery.gdoc.presence hubPath -string ~/path/to/hub
```

With no hub set, the md icon is greyed on every row. With one set, a note is
opened through `obsidian://` with the vault and the relative path, after the app
has proved the file is a regular `.md` file inside the folder you named. A
symlink is not the note either: what is at the end of it is somewhere you did
not choose. No line number is promised, because gdoc does not record one.

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
   came from, and opens it. The glyph is in the menu bar when that run ends.

There is no fourth step. Every later gdoc command under the flag opens the app
again where it is installed and not running, so a quit app comes back with the
next thing you ask gdoc to do. `open ~/Applications/GdocPresence.app` still
works by hand, whenever you want it open and have nothing to ask gdoc. Launch
at login is the switch in Settings, off until you turn it on.

For one run only, without writing anything down, pass the argument instead:

```
gdoc read <link> --presence true
```

The argument takes `true` or `false` and nothing else, and it beats the file
either way. The file is more forgiving, because a line you keep is a line you
edit once: `presence: True` is read as true there. `presence:` with nothing
after it is refused, with the line to write instead.

## Keeping it updated, and taking it back

Every later `gdoc update` refreshes the app along with the binary, out of the one
zip it already verified. If the app is open while that happens, it notices the
new bundle under itself and comes back on its own: it waits two seconds, checks
the bundle on disk is the same app and still signed, opens that path as a new
instance without bringing it to the front, and quits once the open succeeded. It
does this at most once per version in ten minutes, and the second instance of a
pair always quits, so you are left with exactly one app at the new version. If
the open does not succeed, the panel keeps a row saying `Updated to <version>.
Quit and reopen`, and quitting it and opening it again is all that is needed.

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

One more file sits in that folder and holds no record: `presence/app.lock`.
The app takes a lock on it as its first act and holds it until it exits,
however it exits, and that lock is the whole of how gdoc knows whether the app
is running. gdoc asks the kernel about it once, without waiting, and lets go
at once. It reads no byte of that file and never creates it: no file there is
an app that never ran. A second copy of the app that cannot get the lock quits
without drawing a glyph, so you are never looking at two.

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

The app reads the record, draws the glyph, and opens the panel. What it can do
beyond that is the six command lines above, one SIGINT to one wait, and a link
handed to your browser, to Obsidian or to Claude Desktop.

It cannot read a document, answer a comment, propose a change or approve a held
write, and there is no button in it that pretends otherwise. Those stay where
they were: in Claude Code, in the chat, or in the terminal. The panel tells you
a chat is waiting for you, and going to the chat is your step.

## Signing out, and what a held write does then

`gdoc auth logout` deletes one file: the token in `~/.config/gdoc-agent/`. It
reads the path back to be sure the file is gone, and it answers whether there
was one to remove, so running it twice is not an error. It makes no network
call, and that is the part worth knowing: the permission you gave gdoc still
stands at Google. Nothing on this machine can use it, because the token is
gone, but to take the grant away as well, open your Google account's
third-party access settings and remove gdoc there. Logout touches nothing else
in that folder either. Your settings file stays, and so does the presence
record of what already ran.

It refuses rather than guessing in two cases, and names what it found. A
sign-in waiting in a browser is one: finish it or cancel it first, so a login
landing a second later does not write the file back. The other is a file at
that path that is not a token gdoc wrote, a link among them. Nothing is
removed either time.

A held chat write dies with the sign-in it was held under. A write can wait
half an hour for your yes or no, and gdoc remembers which sign-in raised it. If
you sign out in that time, or sign in as somebody else, approving the card
sends nothing: gdoc tells the chat the sign-in changed, drops the hold and
takes the card's approval away. Nothing is posted under the new account,
because the account signed in now is not the one that asked. Ask in the chat
again and the write is built again under the sign-in you have. Nobody's name is
compared to do this, and your account is in no line of the record: gdoc only
knows that the sign-in is a different one.
