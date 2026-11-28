---
id: GD-51
title: A proposal explains itself in a comment nobody asked for
status: To Do
assignee: []
created_date: '2026-11-28T10:51:00Z'
labels:
  - worth-yes
dependencies: []
references:
  - go/internal/propose/doc.go
ordinal: 51000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Nail, 2026-11-18, over a screenshot of an align run: "i dont need additional
comment which explain me what you do". The suggestion already shows the change
in green and struck through. The thread beside it, "Aligned with Renata's
wording, which Nail set as the source of truth on 18 November 2026: the person
who holds the card is called the user", repeats what he told the session
himself. Ten proposals in one run are ten such threads in the margin.

What he wants to see: the suggestion alone, with no thread beside it.

Why it is not a one-line change. Today the comment is not optional, and three
things lean on it:

- `Proposal.Check` refuses an empty `why`, for both kinds.
- The batch is three requests, and `insertComment` is the third. The comment id
  it answers with is what the note's `proposals[]` records.
- `docx_anchored`, one of the three read-backs, looks for that comment in the
  docx export. Without a comment there are two read-backs left, and the
  package comment has to say why two are enough or what the third becomes.

The mark is also the only record of authorship in a thread. A suggestion with
no thread carries the account's name and nothing that says gdoc wrote it. The
note still records the suggestion id, so `withdraw` keeps working from the
note, but a reader of the document alone can no longer tell.

Decisions to take before a plan:

- Is the comment gone for every proposal, or is it a setting, or gone only
  where the person in the session asked for the change themself and kept where
  gdoc proposes something on its own judgement (a review answering `ai!`)?
- What `verified` means with no comment to find in the export.
- Whether `why` stays in the proposals file and goes into the note only.

This is a `docs/v2/DECISIONS.md` entry first, then SPEC, then a plan. It
touches the read-back rule in `CLAUDE.md`, so the trust guardian reads the plan
before it runs. Related: `a-withdrawn-proposals-comment-stays.md`, which
disappears as a problem for every proposal that writes no comment.

Added 2026-11-18 as docs/backlog/a-proposal-explains-itself-in-a-comment-nobody-asked-for.md, worth: yes.
<!-- SECTION:DESCRIPTION:END -->
