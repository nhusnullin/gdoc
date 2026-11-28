---
id: GD-56
title: >-
  A replace block over a paragraph that holds somebody's comment is refused,
  where it could be a suggestion that names the comment
status: To Do
assignee: []
created_date: '2026-11-28T12:14:00Z'
labels: []
dependencies: []
ordinal: 56000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Nail, 2026-11-28, in his words: 'да, предлагай удаление как suggestion и называй комментарий'. Today a replace block whose paragraphs hold a comment that is not gdoc's is refused ('the paragraphs this replace would delete hold work that is not gdoc's: a comment'). A suggestion deletes nothing until a person accepts it, so the block is proposed as a suggestion and the answer names each comment the paragraphs hold, so the session tells the person before they accept. Other work that is not gdoc's (somebody's pending suggestion) stays refused.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A replace block over paragraphs holding somebody's comment lands as a suggestion
- [ ] #2 The envelope names each such comment by id, and the skill tells the session to say whose comment it is before the person accepts
- [ ] #3 A replace over somebody's pending suggestion is still refused
- [ ] #4 Nail sees it on a real document
<!-- AC:END -->
