# Front matter reference

Every document's Markdown file starts with a YAML block between two `---` lines.
The block fills the cover page, the Version Control table, the revision
history and the Document Classification table. Everything after the closing
`---` is the body.

## Fields

`title` is the only required field. A file whose front matter is just
`title: Something` builds.

| Field | Required | Fills | Notes |
|---|---|---|---|
| `title` | yes | Cover title, running header | Do not repeat the document type; it is added for you |
| `doc_type` | no | Cover title, running header | Free text, appended to the title. Leave it out and the title stands alone |
| `version` | no | Cover "Version:" line | Quote it, so `1.0` does not become `1`. Defaults to `1.0` |
| `date` | no | Cover date line | Free text, e.g. `August 2026`. Defaults to the current month |
| `alt_title` | no | Second cover title line | Only set it when the cover genuinely needs "X or Y". Left out, the "or" line is removed |
| `owner` | no | Document Owner | A role, not a person, e.g. `Head of Procurement` |
| `last_approval` | no | Date of Last Approval | |
| `review_frequency` | no | Review Frequency | Defaults to `Annually` |
| `board_ratification` | no | Board Ratification Date | |
| `distribution` | no | Policy Distribution | |
| `classification` | no | Shades the matching class row | `Confidential`, `Restricted`, `Internal` or `Public`. Defaults to `Internal` |
| `heading_numbering` | no | Heading prefixes | `auto` (default) or `none` |
| `revisions` | no | Revision history rows | A list; see below |

## Revision entries

Each entry accepts `version`, `date`, `author`, `approved_by`,
`approval_date`, `section` and `change`. Any field you leave out renders as an
empty cell. One table row is produced per entry, in the order given.

```yaml
revisions:
  - version: "1.0"
    date: 2 April 2025
    author: S. White
    approved_by: Group ARCC
    approval_date: 20 April 2025
    section: "-"
    change: New document
```

## Document types

`doc_type` is free text. Anything goes in the template: `Report`, `Brief`,
`Board Submission`, `PRD`, or nothing at all.

It matters only when the document is genuinely a controlled one. Then use a type
the group register actually uses: Policy, Framework, Procedure, Plan, Programme
or Template. Group governance is explicit that you should not invent a document
type, and that there is deliberately no "Standard". Default to **Procedure** for
a step-by-step flow with named artifacts and owners; reserve **Framework** for a
whole risk domain.

## Approving committees

The committees that exist are BoD, Group ARCC, Group EXCO, Group BUSCO, Group
NOMCO and AFCC. There is no "Senior Manager Committee"; where an older
document says "SMC or ARCC" it means ARCC.

The three approval chains are separate and must not be merged:

- product → BUSCO, via a local BoD recommendation
- vendor and outsourcing → Head of Function and ARCC, with MLRO advice via AFCC
- financial crime → ARCC or AFCC

## Body conventions

- Headings are numbered automatically as `1-`, `1.1-`, `1.1.1-`, matching the
  master. Headings starting Appendix, Appendices, Addendum, Addenda, Annex,
  Contents, Glossary or Schedule are skipped, so "Appendix 2" stays itself.
- Heading levels are relative. The shallowest heading in your file becomes
  Heading1 at 16pt, whatever its Markdown level, and numbering still starts at
  1. You do not have to reshuffle heading levels to suit the template.
- A heading that numbers itself keeps its own number. `## 1. Key terms` and
  `### 3.1 Route one` publish as written, and the numbering carries on from
  them, so a hand-written `4.` is followed by a computed `5-`. Set
  `heading_numbering: none` when the whole body is hand numbered and you want
  no computed prefixes at all.
- `==text==` marks text yellow, the master's convention for "a human still has
  to fill this in". It is the only formatting that survives as a placeholder;
  a filled cover value has its highlight cleared automatically.
- Pipe tables, bullet and numbered lists (including nesting), bold, italic and
  inline code all render. Column widths are set from content, so a prose
  column gets the space it needs.
- The first `#` heading starts a new page, matching the master. Later headings
  flow normally.
