# Proposing a change

Read this before the first `propose` or `withdraw` of a session. It is
SKILL.md's Step 7 in full, and the steps it names are SKILL.md's steps.

Write the proposals file. Each entry is the exact words to replace, the
replacement, and the reason in reader language:

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

Read `verified` and `checks` per proposal:

- `suggestions_inline`: the new words are in the document as a suggestion.
- `preview_without_suggestions`: the old words are still what a reader sees, so
  it is a suggestion and not an edit.
- `docx_anchored`: the 🤖 comment is attached to the new words.

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
