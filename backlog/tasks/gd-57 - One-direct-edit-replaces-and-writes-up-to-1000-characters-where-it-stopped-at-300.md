---
id: GD-57
title: >-
  One direct edit replaces and writes up to 1000 characters, where it stopped at
  300
status: To check
assignee: []
created_date: '2026-11-28T13:10:00Z'
updated_date: '2026-11-28T13:56:00Z'
labels: []
dependencies: []
ordinal: 57000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Nail, 2026-11-28, in chat: 'подними потолок до 1000'. The edit door's ceiling on one delete and one insert goes from 300 to 1000 UTF-16 units, in internal/edit and in the guard. The chat hold on a propose that removes more than 300 characters is another rule and stays.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A replacement of 1000 units is written and one of 1001 is refused by the command
- [x] #2 The guard carries a delete and an insert of 1000 units and refuses 1001
- [x] #3 The decision entry and its register row are in docs/v2/DECISIONS.md
- [ ] #4 Nail has seen an edit longer than 300 characters land yellow in a real document
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Released as v2.39.0 on 2026-11-28. Not yet seen in a document: an edit longer than 300 characters landing yellow.
<!-- SECTION:NOTES:END -->
