---
worth: yes
where: go/internal/frontmatter/schema.go:34
added: 2026-09-08
---
# A note v1 published carries `gdoc: <id>`, which v2's strict reader refuses

Until M6 lands, the only way to create a document from a note is v1's `gdoc generate`, and it
writes the pairing as a plain string, `gdoc: 1AbC...`. v2's front matter is a block,
`gdoc: {schema: 1, document_id, folder_id, ...}`, read under `yaml.Strict()`. So a note v1
published cannot be handed to v2's `propose --md` or `withdraw`: the reader refuses the shape,
and the run has no provenance record, which is the permission to withdraw later. Reading,
answering and proposing on the link alone still work; only the note pairing is lost.

Found while preparing the M4 acceptance run, 2026-09-08. The workaround is to rewrite the line
by hand after the publish. The fix belongs in M6, when v2 learns to publish and writes the
block itself: either v2's reader accepts the v1 string as a schema-0 shape and upgrades it on
the first write, or M6's publish is the only writer and v1's `generate` is retired for v2 notes.
The second is simpler and matches the plan; the first keeps notes published this week usable
without a hand edit. Nail decides in M6.
