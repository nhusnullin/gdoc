---
name: gdoc-restyle
description: Use when Nail gives a link to a Google Doc gdoc did not write and wants it in the Altery house style where it stands. Surveys what the document holds first, proposes the house cover when Nail asks for one, and styles the body in place.
---

# Restyle a document where it stands

Nail names a document somebody else wrote. gdoc gives it the house look without
moving a character of the text: the page geometry, each paragraph's spacing and
indent, each run's face, size and colour, and each table cell's padding and
borders. When Nail asks for the house cover as well, the cover and the three
front-matter tables arrive as a suggestion he accepts or rejects in the browser.

This is the one direct edit gdoc ever makes. Every other command in this binary
proposes. So the run is two runs: a survey that writes nothing and is read by a
person, and an apply that quotes the survey back. Nothing skips the survey.

Spec: `~/src/personal/gdoc/docs/v2/SPEC.md`, sections "The binary", "The
skills", "Verification: never trust a success" and "Never".

## Setup

```bash
GDOC=gdoc
ROOT="$PWD"
```

`gdoc` is the v2 binary, on PATH. It prints exactly one JSON object and exits.
Exit 0 means the object says `ok`. Read the object, never the exit code alone.

One root, `$ROOT`, and it is `$PWD`. It is the hub: where a cover value may be
found, and the only place a note is read from or written to. Scratch output, the
survey and the fields file included, goes under the session's scratchpad, never
beside somebody's note.

Nothing here runs git. Not to commit, not to check whether a file is dirty.

## Learn the command before you call it

Before the first call of a command in this session, run:

```bash
$GDOC help <command>
```

It prints the words that command takes, every flag with one sentence saying
what it is for, and one example. Read the words and the flags out of that
object and build the call from them. This skill names no flag of its own, on
purpose: the binary is the only thing that knows what it takes today, and a
flag list written here would go stale without anyone noticing.

One command matters here, `restyle`. `help restyle` names the survey flag, the
flag that hands the survey back, and the flag that carries the cover values.
`help` with no words prints every command gdoc answers.

## If the credential is not working

Every step here reaches the network, the survey included, so a missing token
stops the run before anything is written. Run `$GDOC auth status` and read it
before guessing. It reports whether a token is present, whether it expired, and
which scopes are missing.

The fix is `$GDOC auth login`. It prints a URL, waits for Nail to approve in
the browser, and saves the token. Ask before running it: it changes which
account edits the document, and that account is visible to everyone the
document is shared with, in its version history.

Never edit `~/.config/gdoc-agent/` by hand, and never tell Nail to.

## If Nail asks for a dry run

Do Step 1 in full, then stop. The survey is the dry run: it reads the comment
listing, the document and the docx export, and it writes to no document and to
no file. Report what it found, as Step 1 says, and offer the cover values as
Step 2 does without writing the fields file.

Say at the end that nothing was styled, nothing was proposed, and the document
is as Nail left it.

## Step 1: The survey, always

Run the survey and send its object to a file under the session's scratchpad.
The file is the whole envelope as the binary printed it, and the apply run
reads it back strictly, so do not edit it, reformat it or append to it.

Then read it out loud, in plain words. It says what the document holds and what
a restyle has to keep:

- **Threads.** Open and resolved, and the witness line for each. The witness is
  the text a comment is attached to, read out of the docx export. A thread the
  export could not answer for reads unmatched, and unmatched is no answer
  rather than a missing anchor.
- **Pending suggestions.** Somebody else's unsettled edits, counted twice and
  in two units: the entries the listing reports, and the ids on elements the
  pending walk skips, such as a suggested page break. Do not add the two.
- **Chips.** People, dates and rich links, which are the fragile things in a
  document.
- **Tabs.** More than one tab is refused by the apply, twice, so say it here
  and stop: a style request names a range, and a range means nothing without
  saying which tab it is in.
- **Named ranges.** Labels Docs keeps in step with its own edits. A marker
  gdoc wrote is one of these, and it means gdoc has proposed a prelude into
  this document before.
- **Every warning.** An element the read could not name is one, and a document
  holding one holds something no count here speaks for.

`nothing_to_protect` is true only when five of those counts are zero: threads,
pending suggestions in both units, chips, elements the read could not name, and
named ranges. Tabs is not one of them and could never be, because every document
has at least one. When it is true, say in one sentence that there is a second
route for a document with nothing in it to lose: `read` the document, put the
text in a note with front matter, and `publish` that note, which gives a
document built in the house style rather than one styled after the fact. Never
take that route on your own. It makes a second document, and which document Nail
keeps is his.

The survey also carries the document's revision id. The apply hands it back and
Docs refuses the batch if the document moved in between, so a survey read a day
ago is not a survey this run can use.

## Step 2: The cover, if Nail wants one

Ask whether the house cover is wanted. Without it the run styles the body
alone, which is the smaller and safer thing. With it, gdoc also proposes the
cover page, the three front-matter tables and the legend, every character of it
as a suggestion Nail accepts or rejects in the browser.

If he wants it, the values come from a JSON file you write, and there are
thirteen of them. The keys are a note's own front-matter keys, because a
restyle has no note to read:

`title`, `alt_title`, `doc_type`, `version`, `date`, `owner`,
`last_approval`, `review_frequency`, `board_ratification`, `distribution`,
`classification`, `heading_numbering`, and `revisions`, which is a list of
rows, each with a version, a date, an author, an approver, an approval date, a
section and what changed.

Propose each value from the document you just surveyed and from the hub, show
Nail the whole list, and write the file only after he confirms it. Say where
each proposal came from, so he can see which are read and which are guesses.

`title` is never invented. It is proposed and confirmed, like every other
value. The binary refuses a fields file with no title, and it carries no
candidate to fall back on, because that file has no body and no name to draw
one from. A value left out leaves its line blank rather than taking somebody
else's.

The file is read strictly. A key the binary does not know is refused by name,
and a misspelled key is a cover line that would otherwise never print.

**A document gdoc has proposed a prelude into before carries a marker**, which
is one of the named ranges Step 1 listed. A second run replaces that prelude
rather than adding a second one, and it proposes the deletion of the old one
first. Two things stop it, and both are read out rather than worked around:

- The old prelude still being pending. The refusal names the suggestion ids and
  says to accept or reject in the browser first, because replacing it would
  propose deleting text nobody has written yet. That is Nail's click, not
  yours.
- Two markers over settled text. gdoc cannot guess which prelude is the one,
  and the refusal says so.

## Step 3: The styling run

Say what is about to be sent, and ask once more before sending it. This is the
only direct edit gdoc makes, on a document Nail did not write, and the sentence
worth saying is what it overwrites: the face, the size and the colour of every
run go to the house value, so an author's own emphasis by size or colour is
gone. Bold and italic are left alone. Table cell fills are left alone. Not a
character of the text moves.

Then run the apply, with the URL, the survey file, and the fields file when
Step 2 wrote one. Run it once. A run that stopped has already left what it
left, and Step 4 says how to read that.

## Step 4: The report

The binary prints facts and judges none of them. Read the object and say, in
this order:

- **`manual`, every step, each with its menu path.** This is the part Nail
  acts on. It is what the house style states and no request could send: the
  first-page header with the logo, the contents list, the footer page numbers,
  the lists, the table column widths, and, when there is one, a named style the
  house has no look for. A run that proposed a prelude adds the column widths
  and row heights of the front-matter tables. The list says what is left to do
  and never whether the document is finished.
- **`verified`.** It is every landing check holding, the preservation half
  intact, and no thread whose witness stopped answering. Name what went false.
  `verified: false` is not a failed run. What was applied is in the document
  either way, and a run reported as failed is a run somebody runs again.
- **What the read-back compared.** Thread counts and the per-thread witness,
  pending suggestion ids by id rather than by count, and the chips. A witness
  that reads unmatched now is unwitnessed, which is the export giving no
  answer, not an anchor gdoc destroyed. A thread that reads detached now with
  no witness before is its own case and is neither of those two.
- **The prelude, when there was one.** How many pieces read back as proposed,
  whether any piece read back as written rather than suggested, whether the
  marker landed, and whether the author's own text is unchanged. A piece that
  is written rather than proposed is characters in somebody's document on
  gdoc's own authority: say it plainly, and say the recovery is the document's
  version history.
- **Whether the run stopped half way, and what that leaves.** The warnings
  carry the sentence, so read it out. A styling run that stopped leaves a
  half-styled document, no text lost, and the recovery is the version history
  by hand. A suggested run that stopped is rejected in the browser instead. A
  prelude that landed unmarked has to be accepted or rejected before this
  command runs again, or the next run proposes a second cover in front of the
  first.
- **`maybe_applied`, when it is there.** A batch Docs accepted whose answer
  could not be read. It may be in the document, so it was never sent again.
  Never say the document is as it was on that path.

Finish with the sentence nobody discovers on their own: a restyle is a moment,
not a setting. Docs has no request that changes a document's named styles, so
the look was applied paragraph by paragraph. The document reads right now, and
the next heading Nail types is Google's Heading 1 again.

## Never

- Never run the styling without a survey read in this session. The survey is
  what makes the write safe, and one produced a moment earlier and shown to
  nobody is not a survey.
- Never run on a document with more than one tab. The binary refuses it twice,
  and the answer is not a flag.
- Never retry a batch Docs refused. A stale revision means somebody edited the
  document after the survey, and a retry is gdoc styling a document while
  somebody is working in it.
- Never invent a cover value. A missing value is a question for Nail, and a
  blank line is a better answer than a plausible one.
- Never edit the survey file or the fields file after showing them, and never
  hand back a survey of a different document.
- Never remove a named range by hand, and never tell Nail to. Nothing deletes
  one, and nothing needs to: the marker goes with the text under it.
- Never run git.
- Never trust a status code. Read `verified` and the read-back.
- Never say a run failed because `verified` is false. Say which check did not
  hold.
- Never take the read-note-publish route on your own. Offer it, once.
