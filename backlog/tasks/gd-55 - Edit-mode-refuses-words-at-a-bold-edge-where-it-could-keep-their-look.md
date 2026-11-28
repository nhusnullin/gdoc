---
id: GD-55
title: Edit mode refuses words at a bold edge, where it could keep their look
status: To Do
assignee: []
created_date: '2026-11-28T12:09:00Z'
updated_date: '2026-11-28T12:14:00Z'
labels: []
dependencies: []
references:
  - go/internal/edit/edit.go
  - go/internal/guard/doc.go
ordinal: 55000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Nail, 2026-11-28: a session sent glossary headings as suggestions because edit mode refuses a quote whose neighbour has another style, and a quote that changes style part way. The cause: Docs gives inserted text the style of the character before it, so new words written after the old ones are deleted take the neighbour's look. A bold term at the start of a paragraph is always refused today, since its only neighbour is the plain text behind it.

Fix for the edge case: write the new words so they take the old words' own look, either by inserting inside the old run before deleting it, or by stating the old look on the inserted range. Both change the exact edit grammar the guard admits, which is Nail's decision (asked 2026-11-28) and the trust guardian's review. Measure the insert behaviour live before choosing. A quote that changes style part way stays refused: nobody can say which new word is bold. With the surrounding-words task done, a caller no longer has to quote across an edge to be unique.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 An edit of a bold term followed by plain text lands bold and yellow, and the read-back says the look held
- [ ] #2 An edit of plain words beside a bold or linked neighbour lands plain
- [ ] #3 The guard admits the new grammar and nothing else, with its test named in guard/doc.go
- [ ] #4 Nail sees it on a real document
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Nail, 2026-11-28: '2: да, пусть правка сохраняет вид слов'. The grammar may widen to keep the old words' look.
<!-- SECTION:NOTES:END -->
