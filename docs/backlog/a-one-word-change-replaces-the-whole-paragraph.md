---
worth: yes
where: go/internal/propose/doc.go
added: 2026-11-18
---
# A one-word change is proposed as a replace of the whole paragraph

Nail, 2026-11-18, over a screenshot of an align run. The change was one word,
"cardholder" to "user", and the word stood twice in the paragraph
("cardholders in one Commercial Partner's app" and "charged to the
cardholder"). The document shows two sentences struck through and the same two
sentences inserted in green, so a reader has to compare them word by word to
find what changed.

What he wants to see: only "cardholders" struck with "users" beside it, and
only "cardholder" struck with "user" beside it. Two small suggestions are fine,
"it might be more than one call because you see it more than once". The other
side matters as much: when the whole sentence really changes, it stays one
suggestion, and gdoc does not make a suggestion per word.

Why it happens. The words kind takes a `quoted` that must stand exactly once in
the document. One word that stands twice cannot be quoted alone, so the skill
widens the quote until it is unique, and here that was most of the paragraph.
The binary then deletes the whole quote and inserts the whole replacement,
although most of both are the same words.

A shape that would work, for whoever plans it:

- Keep the file as it is. The wide `quoted` still places the change, because it
  is unique.
- In the binary, compare `quoted` and `replacement` word by word and send only
  the spans that differ, each as its own delete and insert, placed back to
  front so the indexes hold. Unchanged words are never touched.
- The cutoff: when the changed words are more than some share of the quote, or
  the changed spans are separated by only a word or two, send one replace as
  today. The number is measured on real align runs, not guessed.
- The read-backs then confirm several suggestion ids for one proposal, and the
  note records all of them so `withdraw` takes back the whole proposal.

Open before a plan:

- One batch with several pairs, or one batch per span after a fresh read. Docs
  may give one suggestion id to the whole batch or one per pair. Measure it and
  write the answer into `docs/v2/MEASURED.md`.
- Where the comment anchors when a proposal is several spans, if the comment
  survives `a-proposal-explains-itself-in-a-comment-nobody-asked-for.md`.
- Whether the skills should also be told to quote as little as possible. The
  binary fix makes that unnecessary for a repeated word, but a skill that pastes
  a rewritten paragraph as a block is a different route and is not covered.

`a-replace-blocks-list-leaves-a-second-suggestion-id.md` is the nearest prior
case of one proposal owning more than one suggestion id.
