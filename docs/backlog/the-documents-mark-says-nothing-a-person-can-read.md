---
worth: yes
where: presence/Sources/PresenceUI/
added: 2026-10-26
---
# The mark before a document row says nothing a person can read

Nail, 2026-10-26, on the DOCUMENTS list: some rows open with a grey dot and
others with a green check, and nothing on the panel says why. Asked to make the
difference clear to a person, or to drop it if there is none worth showing.

What the panel showed that day:

| Mark | Row's second line |
|---|---|
| grey dot | live stopped 12:26 |
| green check | commented Thu |
| grey dot | live stopped Wed, no note |
| green check | published Tue |
| green check | published Tue |

So the mark follows the last call on the document: a publish or a reply that
ended `ok` gets the check, and a live review that was stopped gets the dot.
That is a fact about the last run, not about the document, and the second line
already says what the last run was.

What the fix has to decide, before a `ui` milestone:

- Whether the mark earns its column at all. If the second line says "published"
  and "live stopped", the check repeats it and the dot adds nothing.
- If it stays, one meaning a person can name in two words, the same on every
  row: for instance "needs you" (an open thread, a failed call) against
  nothing, rather than "ok" against "not ok" for a run that was never an error.
- A failed last call must not look like a stopped one. Check what the panel
  draws for `ok: false` today, because neither row above was a failure.

Five product questions before the plan, and the design reviewer over the
renders after it.
