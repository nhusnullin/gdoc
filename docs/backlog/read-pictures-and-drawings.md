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
