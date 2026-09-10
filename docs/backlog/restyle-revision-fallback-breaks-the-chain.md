---
worth: yes
added: 2026-09-10
---
# The apply loop's revision fallback adopts an edit the chain exists to refuse

`restyle.Apply` sends each batch carrying `writeControl.requiredRevisionId`, and takes the next
revision out of the batch's own answer. That chain is the milestone's one server-side guarantee: a
batch is sent against exactly the revision the batch before it produced, so an edit somebody else
made in the gap is refused by Docs rather than styled over.

`revisionOf` is what stands in when an answer names no revision, and it reads the document's current
revision instead. That is the document as it is now, so a foreign edit landing between the batch and
that read is adopted as this run's own, and every batch behind it is accepted against it. The loop's
own doc comment says a refusal on a stale revision is never retried against a fresh one, "which is
the exact case requiredRevisionId exists to refuse", and this path does the same thing by another
route. No refusal happened here, so it is not the retry that sentence forbids, but the exposure is
the same one.

What it costs, and what bounds it:

- The requests were all computed from the pre-run read, so a concurrent insertion shifts the indexes
  behind it and the remaining `updateParagraphStyle` and `updateTextStyle` ranges land on the wrong
  paragraphs.
- None of the four kinds `LevelInPlace` carries can change a character, so nothing anybody wrote is
  lost. What a mis-landed request overwrites is that paragraph's own run formatting, which is what
  every restyle overwrites anyway, on paragraphs this run did not mean to touch.
- The answer was measured carrying the field (`docs/v2/DECISIONS.md`, 2026-09-09), so this is the
  rare path rather than the usual one. That is also why it has never been seen.

Two ways out, and the choice is Nail's:

- Stop the run when an answer names no revision, which is the treatment the last batch already gets
  one branch above and the treatment a refused batch gets. It fails closed, which is this project's
  direction everywhere else, and it pays for that with a half-styled document on every run whose
  answer goes quiet, recoverable only through the document's version history by hand.
- Keep the fallback and say so in the report: a warning naming the batch whose answer carried no
  revision, so the caller knows the chain was rejoined rather than unbroken. The exposure stays and
  stops being silent, which is the cheaper half of the same argument the survey's own warnings make.

Found in the M7b external review, 2026-09-10. The doc comment on `revisionOf` now states the gap.
