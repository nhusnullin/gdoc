---
worth: yes
added: 2026-09-06
---
# `gdoc read` must also read pictures, diagrams and Google Drawings

Nail's requirement, recorded during the M2 re-cut on 2026-09-06: `gdoc read` gives the AI the document as
text today. A document also carries inline pictures, schemas and diagrams, and Google Drawings, and the AI
must be able to read those too, not only the words around them. Out of M2's scope on purpose: M2 lands
the text side first.

What is known already (v1, CLAUDE.md "Pictures"): Drive's markdown export returns an embedded image as a
base64 `data:` URI and leaves a Google Drawing out entirely; the docx export carries both as PNG bytes.
So the docx export reader M2 builds is the likely route for the bytes, and the open question is the form
the AI reads them in (a file path per image on the envelope, or inline bytes).

## A positioned picture is silent, and an inline one is not

Found in the M2 review, 2026-09-06. `internal/docs/walk.go` decodes
`inlineObjectElement`, so an inline picture, drawing, equation or object prints
`[image]`, `[drawing]`, `[equation]` or `[object]` and warns. It decodes neither
`positionedObjectIds` on a paragraph nor the document's `positionedObjects` map,
so a **wrapped or floating** picture prints nothing and warns nothing. The AI
reading that document has no sign the picture exists.

That is worse than not reading the bytes, because M2's own rule is that content
`read` cannot read prints a placeholder and warns. Reading the bytes can wait;
the silence should not.

Why it is not a one-line fix: a positioned object is anchored to a paragraph
rather than sitting in a run, so where its placeholder goes in the text is a
decision about the projection, and the golden files are the projection's
specification. Options are a placeholder at the end of the anchoring paragraph,
one at the start, or a warning with no mark in the text at all.
