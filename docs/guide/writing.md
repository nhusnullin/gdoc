# Writing into a document: probe, reply, propose, withdraw, annotate

This page holds the five commands that write into a document you handed in.
Every change they make there is a suggestion, and the page says why. It is for
somebody who wants to know what the review skill sends and how the read-back is
checked.

## Writing into a document

Five commands write into a document you handed in. Every change they make there
is a suggestion, and the fifth changes nothing at all: it leaves a comment. None
of them edits the document, and the guard refuses the attempt inside the
process: a `batchUpdate` on a document you handed in is carried only when the
body says `writeMode: SUGGEST`. The one exception is the restyle in [restyle.md](restyle.md),
which is granted a direct edit on one document for one run, for four request
kinds that cannot change a character, plus the one named range it marks its own
proposed template with.

```bash
gdoc probe --folder <folder url or id>
gdoc reply <url> <comment id> --body-file reply.txt
gdoc propose <url> --from proposals.json --folder <probe folder> [--md note.md]
gdoc withdraw <url> <suggestion id> --md note.md
gdoc annotate <url> --from annotations.json
gdoc annotate <url> --quote "reviewed annually" --body-file why.txt
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

**`annotate`** leaves a comment on the exact words you quote and changes
nothing else. It takes either `--from`, a file of `{quoted, why, assignee?}`,
or `--quote` with `--body-file` for one comment by hand:

```json
[{
  "quoted": "reviewed annually",
  "why": "the register above says quarterly"
}]
```

`quoted` is found the way `propose` finds it: a fresh read, the exact words,
refused when they are not there exactly once. `why` is the bare reason, with no
`🤖 ` on it, because gdoc writes the prefix itself and refuses a reason that
already carries one. It must be plain text, for the reason a reply must: a Docs
thread renders markdown literally. Every entry is checked before the first one
is sent, and a run stops at the first entry that cannot be sent, with
`ok: false` and every annotation still reported with `sent` answered for itself.

`annotate` needs no folder and no note. There is no probe, because the batch
holds one `insertComment` and nothing beside it, so no character can move even
if Google ignored the write mode. There is no note either, because a comment
cannot be withdrawn: a wrong one is removed by a person in the document, since
deleting a comment is a write the guard does not carry.

After the write it reads the comment back two ways, and `checks` says which
held:

| Check | Asks |
|---|---|
| `drive_listing` | Drive's comments carry the returned id, with the body that was sent |
| `docx_anchored` | the docx export wraps the quoted words in that comment |

Both, plus `commentUpdateState: ALL_SAVED`, is `verified: true`. Anything less
is `verified: false` with a warning naming the route, and never an error: the
comment is in the document, and a caller told the run failed would write it
again. The limits are `propose`'s. A document with more than one tab stops the
run before anything is sent, and only body text can be quoted: not a header, a
footer or a footnote.

The review skill runs over these commands. It reads the threads, decides which
ones still need an answer, and writes the reply and proposal files the binary
sends. `annotate` is there for a review skill to call when a comment is all that
is called for, and the one in this repository does not call it yet: a colleague's
own skill is what will, and this page is what it reads. The binary reports facts
either way: there is no `handled` field, and no rule in Go that says a comment is
answered.

