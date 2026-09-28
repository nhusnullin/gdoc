# Proposing a change

Read this before the first `propose` or `withdraw` of a session. It is
SKILL.md's Step 7 in full, and the steps it names are SKILL.md's steps.

There are two kinds of proposal, and one file holds both. The words kind
changes words inside one paragraph. The block kind writes whole new paragraphs,
after a paragraph you quote or in place of a run of paragraphs. Both carry the
reason in reader language, both go out as Google suggestions, and both are read
back three ways.

The words kind first. Each entry is the exact words to replace, the replacement,
and the reason:

```bash
cat > /tmp/proposals.json <<'EOF'
[
  {
    "quoted": "reviewed annually",
    "replacement": "reviewed every six months",
    "why": "The CBC letter of 12 August asks for a six month cycle."
  }
]
EOF
$GDOC propose <url> \
  --from /tmp/proposals.json \
  --folder 1w0SresizE9Kr810VZRJwX4JtDBF4OqNr \
  --md <paired note>
```

`quoted` must appear exactly once in the document. The command refuses none and
refuses more than one, and the fix is to quote more of the sentence. Never work
out a character index by hand: the command finds the words in a fresh read.

The quote must also not run across a footnote mark, a picture, an equation or a
page break. `read` prints those as `[^1]`, `[image]` and so on, and a quote built
by dropping the marker out of a sentence is exactly the shape that crosses one.
The command refuses it and says so: quote a shorter run of words on one side.
Words that stop right before the marker are fine, and so are words that start
right after it. The count is against the whole document, so a quote that reads
once as plain text and once across a marker is refused as ambiguous too.

A link is the other way round: its words are the document's own, but the markup
around them is not. `read` prints text that points somewhere as
`[words](target)`, and `propose` searches the document's text runs, which hold
`words` and nothing else. So a quote copied with the brackets and the address
still on it comes back as not found. Take the markup off and quote the words:
`[the policy](https://example.com/p) is reviewed` is quoted as
`the policy is reviewed`. Words that run from before a link into it, or out of
it, are fine, because the document's text does not break where the brackets do.

Quote the body only. `read` prints each footnote's own text under the rule at
the end, as `[^1]: ...`, and a proposal cannot be placed there: `propose` looks
in the body, so those words come back as not found even though they are on
screen. To change a footnote, say so in a reply instead.

Neither `quoted` nor `replacement` may carry a line break, and `replacement` may
not be empty. A proposal replaces words with words inside one paragraph: there
is no deletion-only shape, none that adds a paragraph, and none that removes
one. A paragraph's text ends with its own break, so a quote copied out of `read`
with the trailing newline still on it would take the paragraph mark with it and
merge two paragraphs. Quote the words, not the line.

`why` becomes the body of a comment anchored on the new words, opening with 🤖.
gdoc adds that prefix, so never write it into the reason: one that already opens
with it is refused, because the comment would arrive signed twice and every
read-back would still pass. This is the opposite of `--body-file` for a reply,
where the prefix belongs in the body and a body missing it is refused. A leading
space or newline in front of the robot does not get around the refusal, and
neither does the robot with no space after it: all three land the same doubled
mark. A reason that is only whitespace is refused too, because the comment would
be a bare signature. The reason
is what the reader sees, so write it for the reader, in plain text. Markdown is
refused before anything is sent, because a Docs thread renders asterisks and
backticks as typed.

When the comment that asked for the change was written by somebody other than
the signed-in account, `why` names them the same way the receipt does: `Asked by
William Mejia. The CBC letter of 12 August asks for a six month cycle.` The rule
and its one exception are in Step 6.

Every proposal in the file is checked before the first one is written, so a bad
entry stops the run with nothing sent. Once writing starts the run stops at the
first proposal it cannot place, and the report still carries one entry per
proposal in the file: read `sent` on each.

`sent: false` means gdoc got no answer saying the proposal landed. A guard
refusal and a 4xx never changed the document. A transport failure or a 5xx is
the third case, and gdoc cannot tell it apart: the request was written and may
have been applied. The envelope's `error` names which one it was, so read it
beside the flag, and read the document before proposing the same words again.

`--folder` is the Drive test folder. The command creates a throwaway document
there on every run, asks Google whether suggestions are honoured today, and
trashes it. `enrolled: false` means nothing was proposed, and the reason is
Google rather than the document: report it and stop proposing in that session.

`--md` is the paired note. It records which suggestions are gdoc's own, and that
record is the only permission to withdraw one later. Pass it whenever the
document is paired. Without it the suggestion still lands and gdoc forgets it
wrote it.

## A block of whole paragraphs

When the answer is a new section, or a section rewritten rather than a phrase
changed, propose a block. Its `content` is markdown, and it becomes whole new
paragraphs in the document: headings, body text and lists. A block names where
it goes in one of two ways, and never both:

```bash
cat > /tmp/proposals.json <<'EOF'
[
  {
    "kind": "block",
    "after": "the last words of the paragraph it goes behind",
    "content": "## 3.6 Limits\n\nThe limit is 50,000 EUR a month.\n\n- paid in\n- paid out\n",
    "why": "The CBC letter of 12 August asks for the limits in a section of their own."
  },
  {
    "kind": "block",
    "replace_from": "words in the first paragraph of the run",
    "replace_to": "words in the last paragraph of the run",
    "content": "The section, rewritten as whole paragraphs.\n",
    "why": "The old wording says annual, and the letter asks for six months."
  }
]
EOF
```

`kind` is what makes an entry a block. An entry with no `kind` is the words
kind, so a file written before blocks existed still runs, and one file can hold
both kinds: they go out in file order, each from its own fresh read. A block
carries no `quoted` and no `replacement`. Those belong to the words kind, and an
entry carrying fields from both is refused before anything is sent.

`after` is a quote in the paragraph the block goes behind. The new paragraphs
land at the start of the paragraph that follows it, so an accept cannot merge
them into a neighbour. `replace_from` and `replace_to` are quotes in the first
and the last paragraph of a run, and the block stands in for that whole run,
paragraph marks included. Every quote is found the way `quoted` is: exactly
once, in the body, without the markup around a link, and never across a footnote
mark, a picture, an equation or a page break.

### What `content` may hold

Paragraphs, headings `#` to `######`, bulleted and numbered lists one level
deep, and bold, italic and links inside a line. A wrapped line is one paragraph,
because what makes two paragraphs is the empty line between them.

Everything else is refused by name, before anything is sent: a table, a nested
list, a picture, a code block, a block quote, raw HTML, a horizontal rule, and
content that is empty. There is also a size limit, so a whole document is not a
block, and the refusal names it. When the answer needs one of those, say so in a
reply and leave the shape to the person.

### Before a replace, read what is in the run

A replace deletes the paragraphs it stands in for, and that is the one proposal
that takes somebody's words out. Before writing the entry, read what is in the
run:

```bash
$GDOC suggestions <url>
$GDOC comments <url>
```

Say what falls inside the paragraphs you would replace, by id and by first
words, and propose only once the person has answered. gdoc refuses a replace
over a pending suggestion or a comment's anchor anyway, and names every
suggestion id and comment id it found, but a refusal at the terminal is a
message to the operator and the person whose work it is never hears it.

### Where a block cannot go

Each of these is refused by name, before anything is sent:

- a quote that is not in the document exactly once;
- `replace_to` in front of `replace_from`;
- an anchor, or a run, inside a table;
- a document with more than one tab;
- a run that ends in the document's last paragraph, because Docs refuses to
  delete a document's final newline: leave that paragraph out of the run, or
  change its words with a words proposal;
- an `after` whose next element is a table, the contents list or a section
  break, because there is no paragraph start to put the text at;
- an `after` on the document's own last paragraph when the block does not end
  with a plain paragraph, or when that last paragraph is a list item or a
  heading.

Every one of those says what would work instead. Read it and do that, rather
than reshaping the quote until something lands.

### What a block comes back with

One suggestion id. The insert, the styles, the bullets and a replace's deletion
all carry the same one, so `withdraw` takes the whole block back in one call.
A block that comes back with more than one id is `verified: false`: every id is
in `suggestion_ids`, and the warning says `withdraw` takes back only the first,
which is the one the note records. Say that in the terminal, because the rest
are then the person's to reject in the document.

The note records where the block went, the `after` quote or the first quote of a
replace, beside the suggestion id and the comment id.

Read `verified` and `checks` per proposal:

- `suggestions_inline`: the new words are in the document as a suggestion.
- `preview_without_suggestions`: the old words are still what a reader sees, so
  it is a suggestion and not an edit.
- `docx_anchored`: the 🤖 comment is attached to the new words.

For a block those three read the same document three ways: every new paragraph
carries the insertion id, and a replace's old paragraphs carry a deletion id;
the anchor paragraph, or the replaced run, still reads to a reader as it did,
and the block's first new line is nowhere in that reading; and the 🤖 comment is
attached to the first new paragraph. For a replace the first new line is the
first one the replaced paragraphs did not already carry, so a section rewritten
under its own heading verifies. A block whose every line is already in the words
it stands on, a list conversion or a reorder, leaves that question nothing to
ask, and the preview check answers "no answer": say so rather than calling it a
suggestion.

`verified: true` is all three checks and a write that answered
`commentUpdateState: ALL_SAVED`. Anything less means the write happened and
something could not confirm it. Say in the terminal what did not hold for which
proposal: the route whose check is false, or, when all three are true, the
warning saying Docs took the batch and its answer could not be read. Never
report a proposal as landed because the command exited 0.

## Withdrawing a proposal

When a proposal was wrong, and it is still pending:

```bash
$GDOC withdraw <url> <suggestion_id> --md <paired note>
```

It works only on suggestions the note records as gdoc's own. Then reply in the
🤖 comment's thread saying the proposal was withdrawn and why. The comment stays
where it is: gdoc does not delete comments.

Never accept, reject or delete a suggestion anyone else wrote, and never accept
one of gdoc's own. Accepting is yours.
