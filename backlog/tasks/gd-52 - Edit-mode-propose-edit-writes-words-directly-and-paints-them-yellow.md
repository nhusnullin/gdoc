---
id: GD-52
title: 'Edit mode: propose --edit writes words directly and paints them yellow'
status: To check
assignee: []
created_date: '2026-11-28T10:56:00Z'
updated_date: '2026-11-28T11:37:00Z'
labels:
  - edit-mode
dependencies: []
ordinal: 52000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Nail, 2026-11-28: edits in two modes. Suggestions stay the default. When the request says edit mode in words (a one-off change or a live review), the words change directly, with no suggestion, and the new words are highlighted yellow. He expects many edits and does not want to accept a suggestion on each.

Needs a fourth guard level opened for one id and one run, a decision entry, and the trust guardian's signature before the code. Earlier drafts: docs/v2/research/2026-11-07-in-place/lead-design.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 propose --edit replaces a quote with new words directly, and the new words carry the edit yellow
- [x] #2 Without --edit every change is still a suggestion
- [x] #3 gdoc mcp never offers --edit
- [x] #4 The edit is read back: new words in place, yellow, no suggestion id
- [x] #5 The review skill uses --edit only when the request says edit mode
- [ ] #6 Nail saw a direct yellow edit in a real document
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Not yet seen in a document: Nail runs one edit in edit mode and one gdoc seen on a real document. Merged in PR 114, not released.
<!-- SECTION:NOTES:END -->
