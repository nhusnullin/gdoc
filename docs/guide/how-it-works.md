# How gdoc stays inside the document you gave it

This page holds the credential, the guard, why every change is a suggestion,
and the JSON envelope. It is for an engineer who wants to know what the binary
can reach and what it refuses.

Review a Google Doc from your terminal. Claude Code reads the comments in the
document, answers the marked ones in the thread they were asked in, and proposes
document changes as native suggestions you accept or reject in Google Docs.

Neither side is the source by rule. A note in the hub and the document it was
published to can both move on, and you decide per run which one is right: the
skill asks, and the `gdoc:` block in the note records only dated facts, when a
document was published from it and when one was exported into it. What does not
change is the direction of a write. Every change gdoc makes to the words of a
document you handed it is a suggestion: nothing in the binary can edit one
directly.

The credential is your own Google account, approved once in a browser. It holds
more than the tool needs, because Google's narrow `drive.file` scope only covers
a file the app created or the user picked through Google's own file picker, and
a terminal cannot show a picker. So the grant is full Drive, and the guard
inside the binary narrows it back down: every request goes through it, and it
carries a request only when the file addressed is one gdoc was handed or one
gdoc created. Anything else is refused inside the process, a read included. The
tool cannot see a document you did not point it at, and cannot search your Drive
at all.

## What it does

One static binary at `go/`, fifteen commands plus `help` and `completion`,
nothing to install beside it. Each command takes arguments, prints one JSON
object and exits. It holds the credential, it reads a document, it writes
suggestions, it leaves a comment on words a caller quotes, it builds a
house-style document and publishes it, it surveys what a document holds before
anything is done to it, it can give that document the house style where it
stands, and it can write a document back into the hub as Markdown.

That last one is the only command that writes more than one file at a time.
`gdoc export` writes one Markdown file per tab and puts the document's pictures beside it as
PNG files in an `assets` folder, which it creates when it is missing. It
replaces nothing: a name that is taken takes the next free number, and there is
no flag that could overwrite a file. [Exporting](exporting.md) holds the rest.

```bash
gdoc auth status   # which token, where it is, whether it has expired
gdoc auth login    # prints a sign-in link and waits for you to open it
```

`auth status` answers when you are signed out too: no token is a fact it
reports, not an error. A token file that is there and cannot be read is a
different answer: it fails and names the file, because "signed out" would send
you to `auth login`, which overwrites the file and loses the evidence.

If the token was granted less than the binary asks for, `auth status` still
says ok and lists the difference in `missing_scopes`, with a warning naming the
scopes and telling you to sign in again.

`auth login` prints the link rather than opening a browser for you, because the
binary runs no other program at all. While it waits it listens on 127.0.0.1
on a port the kernel picks, which is what your browser comes back to, so a
firewall may ask once. It gives up after three minutes.

Both print exactly one JSON object on stdout and nothing else. Prose and the
sign-in link go to stderr, so anything reading the output has one object to
parse and no filtering to do.

The token lives in `~/.config/gdoc-agent/oauth-token.json`, and the login asks
for the Docs read and write scope, because writing a suggestion needs it.
