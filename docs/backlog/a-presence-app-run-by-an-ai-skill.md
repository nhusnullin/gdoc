---
worth: later
where: presence/
added: 2026-10-26
---
# A presence app run by an AI skill, prototyped beside the one we have

Nail's idea, 2026-10-26, in the evening while the v2.21 night plan waited for
his answers: the menu bar app could be managed by an AI agent or an AI skill
instead of by its own Swift code.

What he said, in short:

- Build a prototype as a second presence app, beside the current one, not in
  place of it.
- That second app is run purely by an AI agent.
- Everything the current app does could be managed by an AI skill. That
  includes how the information reaches the person, not only what the app
  shows.
- "It leads that AI skill need to be ..." The message ends there. What the
  skill needs to be is still his to say.

## What settles the value

This is `later` because the value decision is open, not the work. Three things
settle it:

1. **The end of his sentence**: what the skill must be or do for this to work.
2. **What the AI version adds that the app cannot.** Examples are judging which
   document needs him, writing the row in plain words, or answering a question
   about a run. If the AI only redraws the same rows, it costs tokens on every
   refresh and gains nothing.
3. **Whether it stays inside the invariants or needs a decision against them.**

## Invariants it touches

- "The menu bar app runs six argv lists, signals one wait and reads three
  things" (`CLAUDE.md`). An app driven by an agent runs whatever the agent
  chooses. A prototype either keeps that closed list, with the agent choosing
  only among the six, or it is a `docs/v2/DECISIONS.md` entry before code.
- "The binary prints facts and the skills judge." This idea fits that rule: the
  presence record stays facts, and the skill judges and phrases them. It is the
  natural split if the prototype goes ahead.
- "The presence file is the one file gdoc appends to, and gdoc never reads
  it." A skill reading the record is a new reader of it. That is allowed, since
  the rule binds gdoc and not a skill, but the record's field list becomes a
  contract the skill depends on.

## Shape, if it goes ahead

It would be a `ui` milestone, with the trust guardian before the plan because
it touches presence. The five questions come first.
