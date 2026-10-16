---
worth: maybe
where: go/internal/body/body.go:871
added: 2026-10-16
---
# A break before a block the house leaves out opens a page holding nothing

Found on 2026-10-16 in the external review of M19. Measured offline with a
throwaway test in `internal/body`: `Prose.\n\n---\n\n` and then a fenced code
block renders one break paragraph, `Counts.PageBreaks` is 1, and the only
warning is the code block's. No live document was used.

`breakReachesTheDocument` drops a break only when the next block the document
holds is nothing, a `FootnoteList`, another break, or the first body heading.
A code block, an indented code block and a block of HTML are each warned about
and emit nothing, so a note that ends with

    Prose.

    ---

    ```dataview
    ...
    ```

publishes with a last page holding not one word, and `page_breaks` counts it.
Two breaks with only a code block between them give two pages, one of them
blank.

This is what the spec's "Nothing else is collapsed" says today, so changing it
is a `docs/v2/SPEC.md` change and a `DECISIONS.md` entry first, not a fix. The
question for Nail is which of the two readings the author gets:

- The break is the author's own page turn and something did follow it in the
  note, so the document gets the page and the warning says the block was left
  out. That is today's behaviour.
- A page holding nothing is never what the author meant, so the look-ahead
  reads the next block that renders anything rather than the next block, and
  the empty page goes.

The second is about six lines: the look-ahead skips a `FencedCodeBlock`, a
`CodeBlock` and an `HTMLBlock` before it decides, and the decision is then the
one it already makes. A list whose items all render nothing is the same case
one level down and would stay as it is, because answering it means walking the
list.
