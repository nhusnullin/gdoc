---
title: Rendering Conformance
doc_type: Note
version: 2.3
date: 2026-08-29
owner: Platform Engineering
last_approval: 2026-08-01
review_frequency: Annually
board_ratification: 2026-08-14
distribution: Internal staff and contractors
classification: internal
heading_numbering: auto
revisions:
  - version: 1.0
    date: 2026-06-02
    author: N Khusnullin
    approved_by: Board
    approval_date: 2026-06-10
    section: All
    change: First issue
  - version: 2.3
    date: 2026-08-29
    author: N Khusnullin
    approved_by: Board
    approval_date: 2026-08-29
    section: 3, 4
    change: Added the table and list cases
---

# Rendering Conformance Note

## Purpose

This note exists to be rendered. Every construct below is here because some
renderer somewhere got it wrong. The prose is justified, sits at 1.15 line
spacing, and runs long enough to wrap at least twice so that the line breaking
itself can be compared between two builds of the same document.

A second paragraph, so the spacing between paragraphs is measurable. It carries
**bold text**, *italic text*, ***both at once***, `inline code`, ==a highlighted
span that a human still has to fill in==, and a [link to the handbook](https://example.com/handbook)
sitting inside the sentence rather than on a line of its own.

Punctuation matters too: "double quotes", 'single quotes', an em dash --- like
this, an en dash -- like that, and an ellipsis... all of which a smart-quotes
pass rewrites.

## Lists

### Bulleted, three deep

- A first bullet, long enough to wrap so that the hanging indent under it can be
  compared. The second line must line up under the first character of the text,
  not under the glyph.
- A second bullet.
  - A nested bullet, which should carry a hollow circle rather than a disc.
  - Another nested bullet with `code` in it.
    - A third level, which should carry a filled square.
- A fourth top-level bullet.

### Numbered, restarting

1. The first numbered item.
2. The second numbered item, long enough to wrap and check the hanging indent
   the same way the bulleted list does.
   1. A nested number, which should be a lower-case letter.
   2. Another nested number.
3. The third numbered item.

A paragraph between the two lists, so the second list has to restart at one
rather than carry on from three.

1. This must be numbered one.
2. And this two.

### A list starting at seven

7. Seven.
8. Eight.

## Tables

### A narrow table with a date column

| Version | Date       | Owner              | Status   |
|---------|------------|--------------------|----------|
| 1.0     | 2026-06-02 | Platform           | Retired  |
| 2.0     | 2026-07-19 | Platform           | Retired  |
| 2.3     | 2026-08-29 | Platform           | Current  |

The date column must not break inside a date. A column narrower than its longest
unbreakable token makes Word hyphenate the token itself.

### A table with one wide column

| Control | Description |
|---------|-------------|
| AC-1 | Access to the production environment is granted only through the identity provider, and every grant carries an expiry. A quarterly review removes anything that outlived its purpose. |
| AC-2 | Shared accounts are prohibited. Where a system cannot support per-person credentials, the exception is recorded and reviewed at each renewal. |
| AC-3 | Break-glass credentials are sealed, and opening one raises an alert to the on-call engineer within one minute. |

### A table carrying marks and a link

| Field | Example | Reference |
|-------|---------|-----------|
| **Bold** | *italic* | [the handbook](https://example.com/handbook) |
| `code` | ==to fill in== | [the register](https://example.com/register) |

## Quoted and preformatted

> A block quote, which this renderer flattens into ordinary prose rather than
> indenting. That is a deliberate choice in the reference implementation, and it
> has to be reproduced rather than improved on.

```
a fenced code block
  with an indented second line
and a third
```

---

## Deep headings

#### A fourth-level heading

Text under the fourth level, which must be the same size as this prose and
lighter in weight than the heading above it.

##### A fifth-level heading

Text under the fifth level.

###### A sixth-level heading

Text under the sixth level.

## 7. A heading that numbers itself

The author's number wins, and the next computed heading follows on from it.

## A heading after the authored one

Which should be numbered eight.

## Appendix A: things that are never numbered

An appendix names itself, so it takes no computed number.
