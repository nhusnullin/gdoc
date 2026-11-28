---
id: GD-54
title: >-
  A quote that stands in the document more than once cannot be changed, a table
  cell most of all
status: To Do
assignee: []
created_date: '2026-11-28T12:09:00Z'
labels: []
dependencies:
  - GD-4
references:
  - go/internal/propose/span.go
  - skills/gdoc-review/propose.md
ordinal: 54000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Nail, 2026-11-28: a session told him to change three table cells by hand. gdoc never refused them. The session did not send them, because the cell text ('USDT Exchange Service Provider', 'Crypto terms') stands in the document several times and FindSpan requires exactly one match. In running text the fix is to quote more of the sentence. A cell is its whole paragraph, so there is nothing more to quote, and the same holds for a short heading or a bold term. The same limit made the session quote across a bold edge ('Crypto terms. The contract') only to be unique, which edit mode then refused.

Design: an entry may carry a second field, the surrounding words as gdoc read prints them, a table row included ('USDT Exchange Service Provider | Holds the crypto relationship'). The surrounding words must stand in the document exactly once and hold the quote exactly once. Only the quote changes. The rule stays 'a change names text, never a stored index': no occurrence number. Works for a suggestion and for an edit. Needs a DECISIONS.md entry first, and the trust guardian, because it restates an invariant in CLAUDE.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A words proposal into one of two table cells with the same text lands in the cell the surrounding words name, as a suggestion and in edit mode
- [ ] #2 Surrounding words that stand twice, or hold the quote twice, are refused by name
- [ ] #3 The refusal for a repeated quote says how to name the place, and the gdoc-review skill says the same
- [ ] #4 Nail sees it on a real document: the three cells of the M2 note changed without his hands
<!-- AC:END -->
