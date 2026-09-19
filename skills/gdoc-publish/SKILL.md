---
name: gdoc-publish
description: Use when the request names a note in the hub and a Drive folder and asks for that note in Drive as a Google Doc in the Altery house style. Reads the note's front matter, builds it, uploads it into the one folder the request named, and reads back what came out.
needs: v2.4.0
---

# Publish a note as a Google Doc

This is `gdoc-publish`. Say that in the first line of your reply, because three
skills answer a request about a note and a document, and this is the one that
makes a new document.

You name a note and a folder. The note is markdown in the hub. The folder is
where the document goes. gdoc renders the note in the house style, uploads it,
reads it back and records the pairing in the note's own front matter.

Every publish makes a new document. The `gdoc:` block in the note's front matter
is a list of every document that note has met, and a publish appends one entry to
it and touches no other. The block is gdoc's memory of those documents, not a
setting somebody tidies away.

## Setup

```bash
GDOC=gdoc
ROOT="$PWD"
$GDOC help
```

`gdoc` is the v2 binary, on PATH. It prints exactly one JSON object and exits.
Exit 0 means the object says `ok`. Read the object, never the exit code alone.

`help` is the first call of every session, because its object carries
`version`, the release this binary was built from. This skill needs the version
its own front matter names on the `needs` line, or a later one. An older binary
is one the skill is ahead of: say so, say that `gdoc update` is the command
that fixes it, and stop there. No `version` at all is a build made from source
rather than a release, which is not an error: say it once and carry on.

The same object may also carry `update`, which says what is installed and what
is published: `installed`, `latest_stable`, `latest_nightly` and `checked_at`.
When `latest_stable` is there and its three numbers are ahead of `version`,
say once that a newer gdoc is published, name it, and name what installs it:
`gdoc update`, or `gdoc update --major` when the first of the three numbers is
the one that is ahead, because a major release is one a person asks for by
name. Then carry on with the work. This is a remark and never a gate:
`needs` is the only version that stops a session. No `update` key at all is a
build from a checkout, and the skill says nothing about it.

One root, `$ROOT`, and it is `$PWD`. It is the hub: the tree the note lives in,
and the only place a note is read from or written to. Scratch output, the dry
run's docx included, goes to the session's scratchpad and never beside somebody's
note.

Nothing here runs git. Not to commit, not to check whether the note is dirty.

## Learn the command before calling it

Before the first call of a command in this session, run:

```bash
$GDOC help <command>
```

It prints the words that command takes, every flag with one sentence saying
what it is for, and one example. Read the words and the flags out of that
object and build the call from them. This skill names no flag of its own, on
purpose: the binary is the only thing that knows what it takes today, and a
flag list written here would go stale without anyone noticing.

Two commands matter here, `build` and `publish`. `help` with no words prints
every command gdoc answers.

## If the credential is not working

Any command that reaches the network can fail with no token or a permission
error. `build` is not one of them: it touches no network at all. Run `$GDOC auth
status` and read it before guessing. It reports whether a token is present,
whether it expired, and which scopes are missing.

The fix is `$GDOC auth login`. It prints a URL, waits for you to approve in the
browser, and saves the token. Ask before running it: it changes which account
owns the document that is about to be created, and that account is visible to
everyone the document is shared with.

Never edit `~/.config/gdoc-agent/` by hand, and never tell anyone to.

## If the request is a dry run

Do every step, but put nothing in Drive and write nothing into the note.

The dry run is `build`, to a scratch path under the session's scratchpad. It
renders the same bytes `publish` would upload, on this machine, with no
network. Read its report back: the file it wrote, its size, the title and the
running head it took from the note's front matter, the house style it used, and
the counts of what it found in the body.

Say at the end that nothing was uploaded, no document exists, and the note's
front matter is untouched.

## If a picture is refused

`build` and `publish` read PNG and JPEG only. An SVG is refused, and the refusal
names the line and the format. Google Docs does not import SVG out of a docx, so
the PNG has to come from this session.

The refusal happens on this machine before anything leaves it: no document was
created and the note is untouched. So rendering the picture and running again is
not a second publish.

What to do, in order:

1. Read the refusal. It names the file and the line. Read the SVG's root element
   for `width` and `height`; when only `viewBox` is there, take its third and
   fourth numbers.
2. Render it at scale 2, which lands at about 400 ppi once gdoc scales the picture
   to the text column. Not higher: it only makes the docx bigger.
3. Render to the session's scratchpad first and look at the PNG before anything
   goes near the hub. Arrowheads present, text at the right weight. Both dead
   ends below were found by looking, not by an exit code.
4. The PNG goes beside the SVG with the same stem. A PNG already there under that
   name is a stop, not an overwrite: this session cannot tell stale output from
   somebody's own picture.
5. Repoint the one link on the named line from `.svg` to `.png`. The SVG stays.
   It is the master.
6. Run `build` to the scratchpad. A clean object means publish, once.

### What renders it

On macOS nothing has to be installed. `svg2png.js` sits beside this file, in the
base directory Claude Code printed when it loaded this skill, and it drives
WebKit, which ships with the OS:

```bash
osascript -l JavaScript svg2png.js in.svg out.png <width> <height> 2
```

It prints `ok <pixels>` or the reason it could not. Measured on 2026-09-19 over
four diagrams: the same pixels a headless Chrome run gave, a fifth of a second
each, and `build` embedded the result.

When it says the legacy WebView is unavailable, which a later macOS may do, fall
back to a Chromium-family browser by its app path:

```bash
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new \
  --disable-gpu --hide-scrollbars --force-device-scale-factor=2 \
  --window-size=<width>,<height> --screenshot=out.png "file://$PWD/in.svg"
```

`rsvg-convert --zoom 2 in.svg -o out.png` is cheaper still where librsvg is
installed, and `brew install librsvg` is what installs it.

When none of them is present, stop and say so: name the picture, name what to
install, and offer the other door, a PNG you export by hand and link from the
note. Never download a browser or a package to get one. That would be this
session fetching code onto your machine on its own.

Four dead ends, one line each, so nobody spends the calls again: QuickLook
(`qlmanage -t`) crops to a square, `cairosvg` needs libcairo, `NSImage` drops
`marker-end` so every arrowhead vanishes and semibold text comes out regular, and
`safaridriver` needs a one-time `sudo` and opens a visible window.

## Step 1: Read the note's front matter

Open the note and read its YAML front matter before running anything.

**A `gdoc:` block naming a document means this note has been published before,
and it is a stop for a question rather than a refusal.** The binary will publish
it again, into a new document. So ask, once, before running anything:

- **A new document.** The old one keeps its URL, its threads and its entry in the
  block, and it goes stale the moment the new one exists. Say that in one line,
  then carry on with Step 2.
- **The changes in the document that is already there.** Not this skill. gdoc
  never replaces a document's body: the route is `/gdoc-align`, which reads the
  note and the document and proposes each change as a suggestion for the owner to
  accept. Name it and stop.

Name the documents the block already lists, with the date each was published, so
the answer is given on facts. A block naming a document that has gone is the same
question: a new document is the answer, and nothing has to be taken out by hand.

**A note with no `gdoc:` block is a note being published for the first time.**
Read the author's own keys at the same time. Only `title` is required. The
optional keys feed the cover page and the version-control table, and a key that is
not there leaves its line blank rather than taking somebody else's value.

**A body carrying a gdoc marker is refused, by line.** `{+words+}[s:ID]`,
`{-words-}[s:ID]` and `[[c:ID]]words[[/c]]` are what `gdoc export` writes into a
file, and a note that still holds one has markers nobody resolved. The refusal
names the line. The fix is `/gdoc-align`, resolve only, which runs
`$GDOC suggestions <url> --md <note>.md` and works through the markers with the
person. Never delete a marker's words to get past the refusal.

**A note with no title is refused, and gdoc never invents one.** The refusal
carries a candidate, drawn from the note's first heading or from its file name.
Show the candidate and wait for your word on it. Then write it into the note's
front matter yourself, as `title`, and run again. Nothing else in the note
changes.

## Step 2: Publish

Run `$GDOC help publish`, then build the call from what it printed: the note,
and the folder you named in this request.

The folder is the one door this run has. gdoc is handed the folder and learns
the new document's id from the create it carried itself. So the folder must be
the one you named, in this conversation, and nothing else: not a folder another
note was published into, not one found in a different note's block.

Publish is one call per run. A second call is a second document, so never run it
twice to get one. A run that failed has already dealt with what it left behind,
and Step 3 says how to read that. A refusal that happened before anything left
this machine, a missing title or a picture gdoc cannot read, is not a publish at
all: fix it and run once.

## Step 3: Read the object back and say what it says

The binary prints facts and judges none of them. Read the object and say, in
this order:

- **The URL**, so you can open it.
- **`verified`, and each of the three checks under `checks`.** They are three
  read-backs on separate routes: the Docs read says the document is there and
  readable, the tab count says it is one document rather than a shape a later
  command would refuse, and the docx export says Drive can hand it back. Name
  the ones that came back false. `verified: false` is not a failed publish. The
  document exists. A route that did not hold is a route that did not hold, and
  saying the run failed is how a second document gets made.
- **`files_changed`.** The note whose front matter now records the pairing. If
  it is not there, nothing in the hub changed. When the block was written before
  this milestone, the reply also says it was rewritten to schema 2: say that once,
  and say a colleague on an older binary gets a refusal naming `documents` until
  they run `gdoc update`.
- **`rolled_back`, when it is there at all.** It is absent on a run that
  recorded the pairing. `true` means the pairing could not be written and gdoc
  trashed the document it had just made, confirmed by reading it back: there is
  nothing in Drive and nothing in the note, and the run can be tried again.
  `false` means the pairing could not be written and the rollback did not hold:
  the document is live, its id is on the envelope, and somebody has to trash it
  by hand. Say that plainly, with the id.
- **Every warning**, each one in full. A title that came back from Drive
  different from the one on the cover is a warning, not a check.
- **Any picture this session rendered.** A PNG made from an SVG is a change to
  the hub the binary did not make and cannot list, so say it yourself: which SVG,
  which PNG, and which line was rewritten. The hub may not be under git, and then
  this reply is the only record.

If the run failed before anything left the machine, say so: no document was
created and the note is untouched. If it failed after the upload and the answer
could not be read, the error names the folder and says a document may be in it.
Repeat that. Do not guess which.

## Step 4: The three things only you can check

The binary cannot see a rendered page. After a publish that produced a document,
say these three, as things to look at:

- the Altery logo in the first-page header,
- the contents list, which Docs fills in when the document is opened,
- the page numbers in the footer.

Those are facts about the document gdoc just made, and you are the one who
judges them. The session never claims them itself.

## Never

- Never remove or edit a `gdoc:` block in a note. gdoc writes it, and `publish`
  and `export` are what append to it. A block in the way is a question, not an
  obstacle, and never something to take out.
- Never publish twice in one run. Each publish is another document, and the
  colleagues reading the old one are not told.
- Never publish a body that still carries a gdoc marker, and never strip a marker
  to get past the refusal.
- Never pass a folder the binary was not handed in this request.
- Never invent a title, a date, an owner or a classification to make a build
  pass. A missing value is a question for you.
- Never run git.
- Never trust a status code. Read `verified` and `checks`.
- Never say a publish failed because `verified` is false. Say which route did
  not hold.
- Never delete anything from Drive, except through the rollback gdoc did
  itself, which it reports.
- Never export a PDF. You download it from the browser.
