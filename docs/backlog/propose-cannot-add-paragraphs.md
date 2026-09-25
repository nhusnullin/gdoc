---
worth: yes
where: go/internal/propose/propose.go:94
added: 2026-09-25
---
# propose cannot add a paragraph, a heading or a list

## What happened

Hub session `fdf1cd3b-d6d7-4528-a0cf-882ea655b601`, 2026-09-25, the public API v2 document. Nail
agreed a rewrite in two parts. Part 1 was eight edits inside existing paragraphs, and all eight went in
as suggestions and verified. Part 2 was a new section 3.4 (about 30 paragraphs: bold sub-heads,
bullets, one numbered list, replacing the 9 paragraphs there now) and a new section 3.6 after 3.5.
gdoc could not propose either. The session tried the browser pane in Suggesting mode, hit the Google
sign-in, and Nail stopped it. Part 2 ended as a file Nail pasted by hand.

So the one real shape of a review answer that is more than a wording fix, "rewrite this section" or
"add a section here", has no route into the document through gdoc.

## Why propose refuses it

`Proposal.Check` refuses a line break in `quoted` and in `replacement`, and `internal/propose/doc.go`
says why ("A line break is refused on both sides"). Both reasons are about the read-backs, not about
Google:

- A quote ending in a newline takes the paragraph mark, the accept merges two paragraphs, and all three
  read-backs still pass.
- `Carries` asks one paragraph at a time, so a replacement spanning paragraphs is found nowhere and the
  preview check means nothing.

The refusal is right for a words-for-words proposal. The gap is that no second shape exists.

## Google is not the blocker

The prelude already does this. `restyle --fields` proposes the cover, three tables and the legend as
one SUGGEST batch: 250 requests, 70 paragraphs, `insertText` with newlines, `updateParagraphStyle`
with `namedStyleType`, `updateTextStyle`, all carrying suggestion ids (`docs/v2/MEASURED.md`,
2026-09-10). `TestThePreludeNeedsNoGrantAtAll` shows the guard carries all of it on a handed-in
document with no grant. `internal/prelude/doc.go` already argues why `createParagraphBullets` is safe
on text gdoc just inserted. So this is new code over proven requests, not a new door in the guard.

## Proposed shape

A second proposal kind, a block, beside the words kind. Same file, same probe, same one batch per
proposal, same comment with the reason.

- **Where.** Anchored by quoted words, never an index, each found exactly once:
  - `after`: a quote inside the paragraph the block goes after (3.6 after 3.5).
  - `replace_from` and `replace_to`: quotes in the first and last paragraph of a run of whole
    paragraphs to replace (the new 3.4). Whole paragraphs only, start of the first to the mark of the
    last, so an accept can never merge a neighbour.
- **What.** The content as a small markdown subset: paragraphs, `#` headings, `-` and `1.` lists with
  nesting, bold, italic, links. Anything outside it is refused by name, as every other input is.
  Tables are out until `propose-inside-tables` is settled. goldmark is already one of the three
  modules, and `internal/body` already walks it, so no new dependency.
- **The batch.** `deleteContentRange` over the whole paragraphs (replace only), one `insertText` with
  the block's text, then per new paragraph `updateParagraphStyle` naming its style explicitly,
  `updateTextStyle` for inline marks, `createParagraphBullets` for list runs, and `insertComment` over
  the first new paragraph.
- **Inheritance.** Text inserted at a paragraph end takes that paragraph's style and bullet. State
  `namedStyleType` on every new paragraph, and clear a list marker the block did not ask for. That is
  the same defect as `prelude-inherits-a-list-marker`, so one fix serves both.
- **Markers.** The block passes `internal/markers` like every other route into a document.

## Verification needs its own three routes

The words-kind checks do not transfer. A block version, each a read gdoc already makes:

- `suggestions_inline`: every new paragraph carries a suggestion id, and every replaced paragraph
  carries a suggested deletion.
- `preview_without_suggestions`: the anchor paragraph, or the replaced run, still reads as it did
  before, and the block's first line is absent. This is the direct-edit catch.
- `docx_anchored`: unchanged, the robot comment attached to text.

## What is unknown, to measure first

- Does one batch give one suggestion id or one per paragraph? The note's `proposals[]` records one id,
  and `withdraw` rejects one. A block may need a list of ids and a withdraw that rejects them all.
- Does `deleteParagraphBullets` land as a suggestion? Never measured, same open point as the prelude
  item.
- A `replace` whose run holds somebody's pending suggestion or an anchored comment: refuse, or let the
  comment ride the deletion? Refusing is the safe first answer.

## Where it touches

`internal/propose` (a new kind, its batch, its verify), `cmd/gdoc` (the proposals file schema and
`gdoc help propose`), `internal/frontmatter` provenance and `internal/withdraw` if ids become a list, and
`skills/gdoc-review` Step 7 and `skills/gdoc-align`, which today can only propose words and leave new
paragraphs to the person. Probably its own milestone plan rather than a backlog fix.
