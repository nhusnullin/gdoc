# gdoc

Review a Google Doc from your terminal. Claude Code reads the comments in the
document, answers the marked ones in the thread they were asked in, and proposes
document changes as native suggestions you accept or reject in Google Docs.

The markdown note is the source. The Google Doc is a rendering of it. Every
change gdoc makes to the words of a document you handed it is a suggestion:
nothing in the binary can edit one directly.

The credential is your own Google account, approved once in a browser. It holds
more than the tool needs, because Google's narrow `drive.file` scope only covers
a file the app created or the user picked through Google's own file picker, and
a terminal cannot show a picker. So the grant is full Drive, and the guard
inside the binary narrows it back down: every request goes through it, and it
carries a request only when the file addressed is one gdoc was handed or one
gdoc created. Anything else is refused inside the process, a read included. The
tool cannot see a document you did not point it at, and cannot search your Drive
at all.

## Installing

Two routes, and which one you are on depends on whether you have this
repository checked out.

**From a checkout**, which is how the tool is developed:

```bash
./install.sh
```

It builds `bin/gdoc` with `make build`, links it to `~/.local/bin/gdoc`, and
links all three skills, `skills/gdoc-review`, `skills/gdoc-publish` and
`skills/gdoc-restyle`, into `~/.claude/skills/`. Linked, not copied, so
`make build` refreshes the command and an edit to a skill is live with no
reinstall. It is safe to re-run: every step checks what is there first, and it
refuses to replace a real directory whose contents differ rather than write
over work that exists nowhere else. It prints the commit it installed from, and
`+ uncommitted changes` when the tree is dirty.

It writes `bin/gdoc.zsh`, the zsh completion, again on every run, or warns
saying what stopped it. The binary writes it, so a script this run wrote is the
table the parser reads. When the write fails the earlier run's script is still
on disk, and the summary says it was not rewritten this run. `.zshrc` is never
edited: the summary says `sourced from ~/.zshrc already`, or prints the one
`source` line to add yourself, or points at the warning when nothing was
written.

The script never touches `~/.config/gdoc-agent/`. Your token and your config
are written by `gdoc auth login` and by nothing else. After installing, run
`gdoc auth status` to see whether you are signed in.

`~/.local/bin` is not on the macOS default PATH. The script says so when it is
missing and tells you the line to add.

**Without a checkout**, which is how a colleague installs it. One line for the
binary, which downloads the newest release for this machine, checks it against
the published checksum, and copies the binary into `~/.local/bin`:

```bash
curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash
```

Then two commands inside Claude Code for the skills, which travel as a plugin
from a marketplace in this same repository:

```
/plugin marketplace add nhusnullin/gdoc
/plugin install altery@gdoc
```

Claude Code asks whether you want the skills everywhere or in this project
only, and it keeps their update switch on the marketplace's screen in
`/plugin`. A managed Claude Code can refuse a marketplace, and
[release/README.md](release/README.md) holds the two fallbacks for that, along
with everything else a colleague needs. The zip on the releases page carries
the same `install.sh` for an install with no `curl | bash` in it.

Nothing updates on its own. `gdoc update` takes the newest stable release when
you type it, `gdoc update --check` says what it would take and writes nothing,
and `gdoc update --rollback` puts back the binary that was there. The skills
update through Claude Code.

## What it does

One static binary at `go/`, thirteen commands plus `help` and `completion`,
nothing to install beside it. Each command takes arguments, prints one JSON
object and exits. It holds the credential, it reads a document, it writes
suggestions, it builds a house-style document and publishes it, it surveys what
a document holds before anything is done to it, and it can give that document
the house style where it stands.

```bash
bin/gdoc help              # every command, one sentence each
bin/gdoc help propose      # the words one command takes, its flags, an example
bin/gdoc completion zsh --out ~/.gdoc-completion.zsh
```

`help` prints the command table: the words each command takes, every flag with
a sentence saying what it is for, and one example. `--help` and `-h` mean the
same thing anywhere on the line. `completion` renders that same table as a
shell script and writes it to the file you name, for `zsh` or for `bash`. An
existing file is refused without `--force`, so the path above is not the
`bin/gdoc.zsh` that `install.sh` owns. Both read the one table the parser
reads, so neither offers a word or a flag the table does not hold. The bash
script has one rough edge: it sets `complete -o filenames` for the whole
command, because scoping that to the one arm offering file names needs
`compopt` and macOS ships bash 3.2, which has none. So a command word that
matches a directory in the folder you are standing in gets a trailing slash,
and the binary then refuses it by name.

```bash
make build   # bin/gdoc, for this machine
make dist    # bin/gdoc-darwin-arm64, -darwin-amd64, -windows-amd64.exe
```

```bash
bin/gdoc auth status   # which token, where it is, whether it has expired
bin/gdoc auth login    # prints a sign-in link and waits for you to open it
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

`GDOC_CONFIG_DIR` moves both files somewhere else, which is what the Go test
suite uses so tests never touch your real config.

### Reading a document

Four commands read, and none of them writes to Drive. Each one takes the URL
you paste from the browser, or a bare document id.

```bash
bin/gdoc read <url> [--structure]
bin/gdoc comments <url> [--since CURSOR] [--wait DURATION] [--witness]
bin/gdoc suggestions <url> [--md PATH]
bin/gdoc restyle <url> --dry-run
```

They print facts and nothing else. Whether a comment is answered, whether a
suggestion that disappeared was accepted or thrown away, whether the document
and your markdown have drifted apart in a way that matters: none of that is
decided in Go. The skill reads the JSON and judges.

**`read`** gives you the document as one string of text, with everything that is
pending marked in it:

| In the text | Means |
|---|---|
| `# Heading` | a heading, one `#` per level |
| `{+text+}[s:ID]` | a pending suggested insertion, and its id |
| `{-text-}[s:ID]` | a pending suggested deletion, and its id |
| `[[c:ID]]text[[/c]]` | the text a comment is attached to, and the comment id |
| `<!-- tab t.0: Title -->` | the tab that follows, on a document with more than one |
| `[image]`, `[drawing]`, `[equation]`, `[object]` | content that is not text yet |
| `[person: Ada Lovelace]`, `[date: Sep 9, 2026]`, `[link: Q3 planning]` | a smart chip, with the label it shows |
| `[auto text: PAGE_NUMBER]`, `[page break]`, `[column break]`, `[rule]` | the rest of what a paragraph can hold |
| `[unknown: member]` | something in the document this version of gdoc has never seen |

Every placeholder row in that table comes back with a warning naming what the
read did not take from it, the chips and the breaks included. A policy with
eight person chips and three page breaks in it answers with eleven warnings, so
a non-empty `warnings` list does not on its own mean the read went wrong: read
the messages rather than counting them.

If the document's own text contains one of those markers, it comes back with a
backslash in front of it, and a backslash the author typed comes back doubled.
So the rule for reading the text back is a parity: an even run of backslashes is
the author's own text and the marker behind it is gdoc's, an odd run ends in
gdoc's escape and the marker behind it is the author's. The escaping is done one
text run at a time, so a marker whose two halves fall in two runs, or a document
character sitting against one of gdoc's own markers, can still reach the text
unescaped. That gap is written up in
`docs/backlog/escaping-across-run-boundaries.md`. `[object]`
is an embedded object the read could not classify: calling it an image would be
a guess.

Smart chips are read since M7, and before M7 they were not read at all: a person
chip, a date chip and a calendar link were dropped without a placeholder and
without a warning, so a policy naming its owner through a chip came back naming
nobody and nothing said so. Six other things went the same way, including page
breaks and horizontal rules. They all print a placeholder now. A chip's
placeholder carries the label the document shows, so `[person: Ada Lovelace]`
tells you who is there; the email address and a link's target are in
`--structure`. `[unknown: member]` is the wider half of the same fix: something
in the document this version of gdoc has never seen, named by what Google calls
it, rather than silently absent. Tables become pipe tables, with a literal `|` in a cell escaped as
`\|` so the row keeps its shape, and footnotes are appended after a `---` line.
Lists come back as `- ` items, two spaces of indent per level, so a numbered
list reads back as a bulleted one: telling the two apart needs the document's
`lists` map, which gdoc does not read. `--structure`
adds the document tree with character indexes on it, which is what placing a
suggestion at an exact position needs. The text is not a
summary of the structure, and the structure is not a summary of the text.

**`comments`** lists the threads. Each one carries its author, its content, its
replies, whether it is resolved, the sentence it quotes, and the character range
the Docs read placed it at. A thread the Docs read could not place comes back
with `range: null` and a warning, rather than failing the whole listing. The
marker is a fact too: `ai:`, `ai?`, `ai!` or `none`, taken from the first word.
`@ai` is not one of them: the three are matched exactly.

`--since` takes the `cursor` a previous run printed and asks for what changed
after it. The cursor is opaque: it is the newest activity that run saw, encoded,
and nothing reads inside it. Every listing prints one, so a live session can
start on any document: where the run saw no activity at all, which is a document
nobody has commented on, the cursor is dated from the run's own clock a few
minutes back rather than from anything anybody did. Nothing writes it down either, so it lives as long
as whatever is polling.

`--wait` turns that one call into a poll. The binary asks Drive every ten
seconds inside the call, and comes back the moment something happened after the
cursor, or at the deadline with an empty window. It needs `--since`, because a
wait with no cursor answers with the whole document, which is the plain listing
under another name. The value is a Go duration, `9m` or `90s`, and an hour is
the most one call will look for. A wait keeps nothing of its own: the cursor it
prints is all that carries to the next call, and the only file it can touch is
the saved OAuth token, which any command replaces when the access token has to
be refreshed.

With `--wait` the object carries one more field:

```jsonc
"waited": { "polls": 12, "seconds": 118, "interrupted": false }
```

`polls` is how many times it asked, `seconds` how long it looked. A run without
`--wait` has no `waited` field at all. Ctrl-C during a wait is an answer rather
than a crash: the object comes back `ok: true` with no threads, the cursor you
handed in, and `interrupted: true`, and the exit code is still 0. A poll that
failed ends the wait with `ok: false` and says how many polls it made, so
whatever is looping can say the window is unread rather than empty.

`--witness` reads the document a second time, as a docx export, and says of each
thread whether that export carries it `anchored` to text, `detached` from it, or
`unmatched`. The export is the only truthful answer to "is this comment still
attached to anything", which is why it is a second read rather than a field.

**`suggestions`** lists what is pending, each with a stable id, the heading it
sits under and its text. With `--md` it also reports what stopped being pending
since the last look, by comparing against the snapshot in that file's front
matter, then writes the new snapshot. That write happens only after a read that
fully succeeded, and only into a file already paired with the document you read.

### Surveying a document before you restyle it

```bash
bin/gdoc restyle <url> --dry-run
```

This says what a document holds before anything is done to it, and it writes to
no document and to no file:

```jsonc
{
  "schema": 1,
  "document_id": "1AbC...", "revision_id": "ALm37BX...",
  "title": "Supplier Register Policy",
  "tabs": 1,
  "threads": { "open": 3, "resolved": 7,
               "witness": { "anchored": 9, "detached": 1, "unmatched": 0 },
               "witnessed": [ { "id": "AAABc...", "witness": "anchored", "resolved": false } ] },
  "suggestions": { "pending": 2, "on_elements": 0, "ids": [ "suggest.abc", "suggest.def" ] },
  "chips": { "person": 4, "date": 1, "rich_link": 2 },
  "named_ranges": [ { "id": "kix.abc123", "name": "gdoc-checklist" } ],
  "nothing_to_protect": false
}
```

A run gives `--dry-run` or `--from`, never both. The survey is what makes the
restyle safe, so it has to be something you read before you asked for the write,
rather than something the same run produced a moment earlier and never showed
you.

The witness is per thread as well as counted, because it is a before picture:
restyling a document can detach a comment, and a thread that was already
detached beforehand would otherwise look like damage the restyle did. It has the
same two limits it has under `comments --witness`: it finds a destroyed anchor
and not a moved one, and two comments with the same words that disagree give no
answer for either.

`nothing_to_protect` is true only when there are no threads, nothing pending, no
chips, no named ranges, and nothing in the document gdoc could not name. It is a
fact about those five, not advice: whether a document is worth restyling is
yours to decide from the counts. A named range is in that list because it is a
label Docs keeps in step with its own edits, so a replacement of the words it
covers takes it with them. The pending count includes a suggestion whose text is only
whitespace, which the `suggestions` listing leaves out, because it is still
something a rewrite would destroy.

Pending is two numbers, and they are counted in different units, so do not add
them together. `pending` counts the entries the pending walk finds, one per
suggested insertion and one per suggested deletion, so a replacement counts as
2 there. It is not the length of the `suggestions` listing: that listing leaves
out the whitespace-only ones this count keeps, as the paragraph above says.
`on_elements` is counted as ids, so the same replacement counts as 1.

`on_elements` is the suggestions that listing cannot report. It reads text runs
alone, so a suggestion carried only by a run it skips reaches it in no form: a
chip, a page break, a horizontal rule, an auto text, a picture, a footnote
reference. Counting nothing for them would answer "nothing to protect" over a
change somebody is waiting on. A run with one is still printed by `read`, inside
its markers, with its id.

`revision_id` is what the restyle rechecks. Hand this file back with `--from`
and the run refuses the document if it has moved since, and every batch it sends
carries that revision so Docs refuses it too.

`schema` says which shape this survey is, and `--from` refuses one that states
another version or none at all: take the survey again with the gdoc you are
applying with. That is not tidiness. The apply reads this file as the record of
what the document held, so a field a newer gdoc compares against and an older
survey never wrote would read as nothing having been lost. `ids` is that field:
without it a suggestion the run destroyed reads as one that was never pending.

An export that could not be read is a warning, not a failure: you still get the
threads, with every witness reported `unmatched`. A Docs read that failed is a
failure, because the chips, the suggestions, the ranges, the tab count and the
revision id are all in that one read.

### Restyling a document in place

```bash
gdoc restyle <url> --dry-run > survey.json
gdoc restyle <url> --from survey.json
```

Two runs. The first surveys, the second gives that document the Altery house
style where it stands: the page size and margins, each paragraph's spacing and
indent, each run's face, size and colour, and each table cell's padding and
borders.

**This is the one thing gdoc does to a document you handed it.** Everywhere
else, every change is a suggestion you accept or throw away. Here the change is
made. What holds it in is narrow: the run styles the one document you named and
the survey agrees with, only if the document has not moved since the survey, and
the four request kinds it may send cannot change a character of what you wrote.
The grant lasts one run and is not written anywhere.

**A restyle is a moment, not a setting.** The Docs API cannot redefine a
document's named styles, so gdoc applies the look paragraph by paragraph. The
document looks right afterwards, and the next heading you type is Google's
Heading 1 again, not the house one.

**What it overwrites is formatting inside the paragraphs it styles.** The face,
the size and the colour of every run go to the house value. Bold and italic are
left alone, and so is a table cell's own fill: the house style does not say
whether they should be off, and clearing them would be gdoc guessing. Your
words, your comments, your pending suggestions, your smart chips and your named
ranges are not touched.

**A failed run leaves a half-styled document.** There is no rollback, and the
recovery is the document's own version history, by hand. No text was touched, so
nothing you wrote is lost, but your own run formatting inside the paragraphs
that were restyled is. The run says so in its warnings.

Afterwards it reads the document back three ways and reports two things. What
survived: your threads with a witness for each, your pending suggestion ids and
your chips, before against after, with anything that moved named and never
explained. And whether the style is really there: it reads a margin, a
paragraph's spacing, a run's font and a cell's appearance back out of the
document and says which ones landed. `verified: false` is not a failure. What
was applied is in the document either way, and the field says the read-back
could not confirm all of it.

The page is the one check that reads more than the value it sent. If your
document has section breaks that set their own margins, or turn their own pages
on their side, those are what a reader sees, and gdoc cannot change them:
`updateSectionStyle` is not one of the four kinds it may send. So the check says
the page geometry is unconfirmed and names the fields your sections set
differently, rather than reporting a page as restyled because the value came
back. A section restating the margin gdoc asked for is not one of them, and
neither is one turned the same way as the rest of the document: they override
nothing you would see.

A run that stopped because Docs took a batch and the answer could not be read
still reads the document back, and says the batch may be in it. It is the run
those facts are needed for most: what may have landed there is a direct edit.
Every request that reached Docs is read back, that batch's included, and the
checks read the first request of each kind, so on a run that stopped after
several batches they usually answer for the earlier ones. `verified` stays
false either way, because the last batch was never confirmed.

`manual` is what gdoc could not do, each with the menu path: the first-page
header with the logo, the contents list, the footer page numbers, plus any list
it left alone, because the request that sets a bullet also deletes the tabs that
set its nesting level, and any table's column widths, and any paragraph whose
named style the house has no look for.

A document with more than one tab is refused before anything is sent. A style
request names a range, and a range means nothing without saying which tab it is
in.

### Adding the house template as a suggestion

```bash
gdoc restyle <url> --dry-run > survey.json
gdoc restyle <url> --from survey.json --fields fields.json
```

The same two runs, with one flag more. `--fields` adds the house cover, the
three front-matter tables and the legend to the styling above, and **it adds
them as a suggestion**. You accept them in the browser the way you accept any
suggestion, or you reject them and the document is exactly as it was.

That is the whole design rather than a caveat on it. The run has two
permissions, not one. The template goes out in suggesting mode on a connection
that was granted nothing at all, which is the same thing `propose` does every
day. The styling goes out on a second one holding the direct-edit grant, for the
four request kinds that cannot change a character. Neither half can do the
other's job.

`fields.json` is the cover's own values, and it is the same thirteen a note's
front matter carries:

```json
{
  "title": "Payment Services Policy",
  "doc_type": "Policy",
  "version": "1.2",
  "date": "September 2026",
  "owner": "Head of Compliance",
  "classification": "Internal",
  "revisions": [
    {"version": "1.2", "date": "2026-09-01", "author": "N. Khusnullin", "change": "annual review"}
  ]
}
```

A restyle has no note behind it, so somebody has to write this file. `title` is
required and gdoc will not invent one: a file without it is refused, and the
skill asks you. A key it does not know is refused by name rather than ignored,
because a misspelled key is a cover line that would silently never print.

**Run it twice and you get one cover, not two.** gdoc puts a named range called
`gdoc:house-prelude` over what it proposed, and that marker is its whole memory
of having been here. A second run finds the marker and proposes replacing what
it covers. If you rejected the first prelude the marker went with it, so the
second run proposes cleanly. If the first prelude is still sitting there
unaccepted, the run refuses and tells you to accept or reject it first: it will
not propose deleting text that has never been written.

The marker is the one thing written directly rather than suggested, and only
because Docs refuses to apply it as a suggestion. It adds and removes no
character.

Afterwards gdoc reads the prelude back and asks three things: whether every
piece of it carries a suggestion id, whether the marker covers what was
proposed, and whether your own text is character for character what it was.
`verified` is all three together. `verified: false` is not a failure, the same
as everywhere else.

The template's own `manual` list is reported beside the styling one, under
`prelude`, and it names two things: the contents list, which no Docs request can
create, and the column widths and row heights of the front-matter tables, which
need two request kinds nobody has measured as suggestible, where a request Docs
refuses takes the whole batch with it. The contents list is in both lists,
because the styling half names it on every run. Heading numbering is not
in this yet, and it is not blocked either: it needs its own answer to what a
second run should do about a number gdoc already wrote.

If the template phase fails, the styling phase does not run, and the report says
which phase stopped. Whatever of the template reached the document is a
suggestion, so rejecting it puts the document back.

### Writing into a document

Four commands write into a document you handed in, and every change they make
there is a suggestion. None of them edits it, and the guard refuses the attempt
inside the process: a `batchUpdate` on a document you handed in is carried only
when the body says `writeMode: SUGGEST`. The one exception is the restyle above,
which is granted a direct edit on one document for one run, for four request
kinds that cannot change a character, plus the one named range it marks its own
proposed template with.

```bash
gdoc probe --folder <folder url or id>
gdoc reply <url> <comment id> --body-file reply.txt
gdoc propose <url> --from proposals.json --folder <probe folder> [--md note.md]
gdoc withdraw <url> <suggestion id> --md note.md
```

None of them decides what to write. The body of a reply, the words of a proposal
and the reason for it arrive already written, in a file the skill wrote.

**`probe`** asks Google whether suggestions are honoured for this project today.
It creates a throwaway document in the folder you name, writes one sentence into
it directly, suggests one word inside that sentence, reads the document back,
and trashes it. `enrolled: true` means the suggested word came back carrying a
suggestion id. `enrolled: false` means it came back as plain text, which is a
silent direct edit: the exact failure this asks about. The report names
`probe_document_id` and says whether the trash succeeded, so a document left
behind is named rather than lost.

The probe exists because the field that makes a write a suggestion is not in the
public Docs discovery document, and one morning that call returned 200 and
edited the document for real. `docs/v2/BLOCKED-BY-API.md` records both
measurements. So `propose` runs the probe every time, on a document of its own,
and sends nothing when the answer is no.

**`reply`** posts one reply into a comment thread and reads the thread back to
see it there. The body must open with `🤖 ` and nothing before it, which is how
a later run tells gdoc's own replies from everybody else's, and it must be plain
text: a Docs thread renders markdown literally, so asterisks and backticks
arrive as typed. Both are refused naming what was found.

**`propose`** writes a change as a native suggestion with a comment beside it
saying why. It reads the file `--from` names, a list of
`{quoted, replacement, why, assignee?}`:

```json
[{
  "quoted": "reviewed annually",
  "replacement": "reviewed every six months",
  "why": "the policy above says six months"
}]
```

`quoted` is text, never a position. An index worked out from an earlier read is
the hazard the whole API has, so `propose` reads the document fresh, finds the
words, and refuses when they occur more than once: quote more of the sentence.
Text already inside a pending suggestion is not matched, so a proposal on top of
a proposal is refused too. A quote running across a footnote mark, a picture, an
equation or a page break is refused as well, naming what it crossed: the span
would take that content with it. An occurrence that crosses one still counts, so
a quote reading once as plain text and once across a mark is refused as
ambiguous rather than placed on the plain one. Each proposal is one
`batchUpdate` after its own fresh read, because the first change moves the
ground under the second, and a document with more than one tab stops the run
before anything is sent.

`replacement` may not be empty, and it may not carry a line break: a proposal
replaces words with words inside one paragraph, and there is neither a
deletion-only shape nor one that adds a paragraph. `why` becomes the comment, so it must be plain text
and must not carry the `🤖 ` prefix itself, which gdoc adds. Every proposal in
the file is checked before the first one is sent, so a bad entry stops the run
with nothing written.

**`verified` is the read-back, never the status code.** After the batch,
`propose` reads the document three more ways, and `checks` says which held:

| Check | Asks |
|---|---|
| `suggestions_inline` | the replacement is there, carrying a suggestion id |
| `preview_without_suggestions` | the original words are still there with suggestions hidden, so it is a suggestion and not an edit |
| `docx_anchored` | the docx export carries the 🤖 comment, attached to text |

All three, plus a write that answered `commentUpdateState: ALL_SAVED`, is
`verified: true`. Anything less is still `ok: true` with `verified: false` and a
warning naming what did not hold, because the write happened and hiding that
would be worse. A batch Docs accepted whose answer could not be read is the case
where every check holds and `verified` is false, and the warning there names the
lost answer. The skill reads `verified` and decides what to say.

**`withdraw`** retracts one of gdoc's own pending proposals. It needs `--md`,
and the reason is the whole rule: the note's `proposals` list is the only record
of which suggestions gdoc wrote, and gdoc withdraws only those. An id that is
not in there is refused. The write is a `rejectSuggestion` naming that id, and
the guard carries it only because the command granted exactly that id for this
run: gdoc still never accepts, rejects or deletes anyone else's suggestion. The
suggestion is gone only when the answer names it in `rejectedSuggestionIds` and
a fresh read shows no run carrying it on either side; only then does the entry
leave the note. The 🤖 comment stays where it is, because deleting a comment is
a write the guard does not carry.

With `--md`, `propose` records what it wrote into the note's `gdoc:` block:
the suggestion id, the comment id, the time and the words that were replaced.
That record is the permission to withdraw later, so a run without `--md` still
lands the suggestion and simply forgets it.

The list is not a complete record on its own. A proposal gdoc does not have both
ids for cannot be written down at all, because an entry missing either fails the
block's own validation, so the run warns that gdoc cannot withdraw it later. The
warning says which id and where it would have come from: the comment id is the
write's own answer, the suggestion id is read back afterwards. The change is in
the document either way. `files_changed` is per run, not per proposal: the note
is left out of it only when nothing at all could be added to it, so a run that
recorded two proposals and lost the third still names the note and carries the
warning about the one it lost. Read the warnings, not the file list. `gdoc suggestions` is the document's own
answer, and the note is gdoc's memory of it: read both.

The review skill runs over these commands. It reads the threads, decides which
ones still need an answer, and writes the reply and proposal files the binary
sends. The binary reports facts either way: there is no `handled` field, and no
rule in Go that says a comment is answered.

### Building the document

`gdoc build` turns a note into an Altery house-style `.docx` on your own
machine. It reaches nothing: no Drive, no Docs, no network at all, and it runs
no other program. `gdoc publish`, below, is the same render with the upload
behind it.

```bash
gdoc build --md note.md --out note.docx [--house house.yaml] [--force]
```

The house style is a file, `house.yaml`, and it is embedded in the binary, so
there is still nothing to install beside it. It states the page geometry, the
nine named styles, the cover, the header and footer with the logo, the three
front-matter tables cell by cell, the legend, the contents field and the heading
numbering. `--house` points at another copy of that file for one run, which is
how a change to the style is looked at before it is committed; the output names
which one was used, `embedded` or the path, so a document built from a draft
says so.

`--out` will not overwrite a file that is already there. Add `--force` when you
mean to replace it. `--force` is consent to replace a document, not a folder and
not something the run reads: an `--out` naming a directory, the note, the
`--house` file or one of the note's own pictures is refused whatever the flag
says. The write goes through a temporary file and a rename, so a failed build
cannot truncate a document you already had.

The note needs a `title` in its front matter and nothing else. These are the
keys it reads:

| Key | Does |
|---|---|
| `title` | required. The cover, and the running head in the page header |
| `alt_title` | the second title line on the cover, under the word `or`, and the running head. A note that states none prints neither line |
| `doc_type` | joined to the title on the cover, so `Third Party Risk` plus `Policy`. Free text, and a title that already ends in its own type is left alone |
| `version` | the cover, behind the word `Version:`. `1.0` when the note states none |
| `date` | rendered on the cover in UK long form. The month the build runs in when the note states none |
| `owner` | the Document Owner row of the version-control table |
| `last_approval` | the Date of Last Approval row |
| `review_frequency` | the Review Frequency row |
| `board_ratification` | the Board Ratification Date row |
| `distribution` | the Policy Distribution row |
| `classification` | one of `confidential`, `restricted`, `internal`, `public`. It shades that class's row in the classification table, and the rest are left clear. `internal` when the note states none |
| `heading_numbering` | `false` turns off the `1-Scope` numbering on level-one headings |
| `revisions` | rows of the revision-history table: `version`, `date`, `author`, `approved_by`, `approval_date`, `section`, `change`. A note that declares none keeps the template's own row |

A note with no `title` is refused, and the refusal proposes one: the first
heading in the body, or the file name. The binary never invents a title and
never writes one into your note. Any other key in the front matter is carried
untouched, the `gdoc:` block included.

The body is your markdown: headings to six levels, bulleted and numbered lists
three deep, bold, italic, strikeout, `==marked==` text as a highlight, links,
pictures, tables, horizontal rules and block quotes. A picture is read from the
note's own directory, or decoded when the markdown carries it as a `data:` URI;
PNG and JPEG. A picture at an `http` address is refused naming the line, because
a document built from a link is a document that breaks when the link expires.

Code blocks are not rendered. The house style has nothing to render them in, so
a note carrying one gets a warning naming the line and the block is left out
rather than dropped in silence. Blocks of HTML, inline HTML and footnotes are
the same answer. Footnotes matter more than they look: `gdoc read` writes a
document's footnotes as `[^1]` in the prose and the definitions after a `---`
line, so a note pulled out of a Doc carries them, and each one is named by the
line the author wrote it on. So is a picture inside a list item, a block quote or a table cell: the
house style puts a figure on a centred line of its own, which it cannot be
there, and the warning names the line.

Two things about a numbered list carry a warning rather than the numbers you
wrote. The document defines one numbered list, so a second one carries on from
the first: your 1. and 2. print as 3. and 4. And every level of it starts at 1,
so a list you opened at "5." opens at 1. Both name the line. A heading that
skips a level is the third: a `###` under a `#` is numbered `1.0.1-`, and the
warning carries the number it wrote.

A list item that holds one of those and nothing else, or nothing at all, has no
words to put a marker on. It takes none, and the warning names the line and
says what that costs: in a numbered list the items after it print one lower, and
in a bulleted list the item loses only its own bullet and its indent.

What you get back is one JSON object: the file it wrote and its size, the title
and the running head it used, which house file it read, and what the walker
counted.

```json
{
  "ok": true,
  "data": {
    "out": "/Users/you/notes/supplier-register-policy.docx",
    "bytes": 48213,
    "title": "Supplier Register Policy",
    "running_head": "Altery - Supplier Register Policy",
    "house": "embedded",
    "body": {"paragraphs": 41, "headings": 9, "lists": 3, "tables": 2, "images": 1}
  },
  "warnings": ["line 88: a code block is not rendered in the house style and was left out"]
}
```

Those counts are facts and nothing more. Whether the document is right is
answered by opening it, in Word or in Drive, and not by the binary.

The contents list is a Word field, which is what the master template carries
too. Word fills it in when the document is opened and refreshed, and Google Docs
turns it into a live contents list on import. Until then it shows the one line
Word writes into an unrefreshed field.

There is one test that keeps this honest on every commit. It builds a document
from the embedded style, opens the Word master the style was extracted from, and
compares 169 measured values across the two: page geometry, all nine styles, the
header, the footer, the logo's position, the contents field, the three tables
and the body. Every row that differs, and every row the master has nothing to
compare against, is named in a list with the reason it is there, most of them
because the master states a value twice and because the two documents hold
different words. A row inside its own tolerance reads as close and needs no
entry. Any other difference fails the test suite, so the style cannot drift away
from the master quietly.

### Publishing the document

`gdoc publish` does what `build` does and then puts the result into Drive as a
Google Doc, in one folder you name, and writes the pairing back into your note.

```bash
gdoc publish --md note.md --folder-id FOLDER [--house house.yaml]
```

The note is rendered exactly the way `build` renders it, so everything in the
section above applies: the same front-matter keys, the same warnings, the same
refusals. The docx bytes are uploaded with conversion, so Drive turns them into
a document rather than leaving a .docx sitting in a folder.

The folder is the only thing the run can reach. No document is in reach when it
starts, and the new document's id comes back from the create the tool itself
made. There is no `--folder-id` default and no fallback to the folder in your
note: a note that already names a document is refused before anything leaves
your machine. There is no second-version command, and `restyle --new` is not
being built. To publish a note again, take the `gdoc:` block out of it by hand
and run `publish` once more; the refusal message says the same.

Three things are checked after the upload, and each answers something the other
two cannot: the document reads back through the Docs API, it has exactly one
tab, and Drive can export it as a .docx again. `verified` is the three together.
Fewer than three is still `ok: true` with the check that did not hold named,
because a document that exists is a document that exists, and being told the run
failed is what makes somebody upload a second one.

If the upload worked but your note could not be written, the document goes back:
it is trashed and the trash is confirmed, and the output says `rolled_back:
true` with no document id. Only if that also fails do you get the live id and
what to do with it. The reason is not tidiness. A document nobody's note points
at is one the next publish makes a second of.

```json
{
  "ok": true,
  "data": {
    "document_id": "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
    "folder_id": "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr",
    "url": "https://docs.google.com/document/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd/edit",
    "title": "Supplier Register Policy",
    "house": "embedded",
    "bytes": 48213,
    "tabs": 1,
    "verified": true,
    "checks": {"read_back": true, "one_tab": true, "docx_export": true},
    "files_changed": ["/Users/you/notes/supplier-register-policy.md"],
    "body": {"paragraphs": 41, "headings": 9, "lists": 3, "tables": 2, "images": 1}
  },
  "warnings": []
}
```

Your note is read again just before it is written, because the upload takes
seconds and these notes live in a synced vault. If it changed in that window, in
its body or in its front matter, nothing is written and the document is rolled
back: the file on disk is no longer the one that was rendered, and publishing it
as though it were would pair your note with a document it does not match.

There is a second test that only runs when you ask for it, and it is the one
that means something: it builds the document, uploads it and the Word master
side by side, reads both back through the Docs API, and compares the same 169
values. Google's import is part of that reading, so a difference that shows up
there and not offline is Drive's doing rather than the generator's. It creates
two real documents and trashes them, so it is behind two environment variables
and never runs by itself.

### The `gdoc:` block

A markdown note paired with a Go-published document carries one key in its front
matter, and everything else in there stays the author's:

```yaml
---
title: Supplier register policy
gdoc:
  schema: 1
  document_id: 1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd
  folder_id: 1w0SresizE9Kr810VZRJwX4JtDBF4OqNr
  published:
    at: 2026-09-08T10:14:00Z
    title: Supplier register policy
    house: embedded
  suggestions_seen:
    at: 2026-09-06T11:00:00Z
    items:
      - id: suggest.abc123
        kind: insertion
        section: Scope
        text: "critical "
---
```

The read is strict. An unknown key, a key given twice, a missing `document_id`
or a `schema` this version does not know is refused naming the key, and the file
is left alone. The write touches the `gdoc:` lines and nothing else: your keys,
your line endings and the trailing newline come back byte for byte, and the file
is replaced through a temporary file and a rename, so a failed write cannot
truncate your note.

`gdoc publish` is the only thing that creates this block. `gdoc suggestions
--md`, `gdoc propose --md` and `gdoc withdraw` update it, and each of them
refuses a note that does not carry one already. A note whose `gdoc:` key is a
bare string rather than a block is refused by name: gdoc tells you to rewrite
the line by hand once, or to publish the note again with `gdoc publish`.
Building is unaffected, because `gdoc build` skips the `gdoc:` key whatever is
in it.

## What is planned

`docs/v2/PLAN.md` holds the open work. The two you would notice from outside:
a restyle that stays restyled after the run, and comments that carry no marker
read and answered like the marked ones.

---

Working on the tool itself? Start with [PRINCIPLES.md](PRINCIPLES.md), then
[CLAUDE.md](CLAUDE.md).
