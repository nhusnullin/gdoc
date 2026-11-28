---
id: GD-53
title: 'gdoc seen: remove the yellow from every edit gdoc made'
status: To check
assignee: []
created_date: '2026-11-28T10:56:00Z'
updated_date: '2026-11-28T11:37:00Z'
labels:
  - edit-mode
dependencies:
  - GD-52
ordinal: 53000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Nail, 2026-11-28: a second command. 'Everything marked yellow in edit mode, remove the highlight. I am ok with it, I read them, I saw them.' Clears the edit yellow and nothing else: no character changes, and a mark the author made stays.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The command clears every run painted the edit yellow and reports how many
- [x] #2 A run of another colour, the house yellow mark included, is untouched
- [x] #3 No character changes
- [x] #4 The review skill runs it only when the person asks
- [ ] #5 Nail saw the yellow go in a real document
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Not yet seen in a document: Nail runs one edit in edit mode and one gdoc seen on a real document. Merged in PR 114, not released.
<!-- SECTION:NOTES:END -->
