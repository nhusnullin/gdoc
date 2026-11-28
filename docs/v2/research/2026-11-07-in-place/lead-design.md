# Change in place: the lead's merged design, v1

2026-11-07. Merges design.md (co-editing designer) and sync.md (sync engineer), both in this folder, with the evidence in sessions.md and api.md. Where the two disagreed, the lead's call is stated. Nothing built, nothing measured, nothing written to the repo.

## Nail's request (condensed, his words)

"Instead of proposed changes, change text in place and mark them green. Same when I instruct AI through comments to remove or rephrase. Two modes, propose and change in place. When I work on a published document, the change goes to the linked md file, if one exists, and to the Google Doc. Gap: a new session on an old published note (the embedded retail card) must not change my Google Doc unless I mean it."

## The design

1. **No switch to set.** A Doc changes in place when (a) a note's `gdoc:` block has a `published` record for it, and (b) the intent gate passes. Every other Doc gets suggestions, as today. "propose" or "in place" in a request flips that Doc for the rest of the session. The session states the mode once when it starts to hold. Rejected: `mode:` in the note (a written-down permission is the retail-card gap), a setting (one line opens every Doc), the presence app (it writes nothing gdoc reads).

2. **The intent gate (skill, judgement).** In place, or any write, only when this session's own conversation holds one of: Nail typed or pasted the Doc's link; this session published the Doc; a marked comment arrives in a live review Nail started on that link in this session. Not enough: the id in the note, a file, or an earlier session. When the gate fails, Claude changes the note only and ends with one line naming the Doc it left alone and "paste its link to change it there too". No blocking question.

3. **The binary half (fact, enforced).** A new flag on `propose` (name in the plan), never offered by `gdoc mcp`. It takes `--md <note>` and refuses a Doc with no `published` record in that note. It raises the id with a new guard door `Policy.GrantEdit(id)` to a fourth level for one run. Every batch carries `writeControl.requiredRevisionId`. The 2026-11-03 batch grammar is the starting point, widened to the kinds in point 4, each checked as a grammar by the guard.

4. **Kinds, v1.** Words: replace or delete a quote inside one paragraph. Add plain paragraphs after a quoted paragraph. Delete up to three whole body paragraphs or list items, quoted in full. Mass replace: a list of Words edits, each unique by context, one batch, highest index first, a find-all with filters, never `replaceAllText`. v1 rule from sync: old and new words are plain prose, one paragraph, one uniform style. Refused or left as suggestions: headings, tables, section moves, more than three paragraphs, spans touching a link, a bold or italic edge, a mark, pencil, a chip, a footnote mark, a picture, a line break, the prelude. A span crossing a colleague's pending suggestion is refused, with whose change is in the way. A span crossing another thread's anchor is refused, except the `ai!` thread that asked.

5. **Drift check (sync).** Before writing, the whole paragraph holding the span must be equal on both sides in rendered text (goldmark with `internal/body`'s typographer on the note side; on the Doc side, text as written, suggested deletions in, suggested insertions out). If they differ: write nothing, show both, offer: bring the Doc paragraph into the note first, Doc only, or a suggestion.

6. **Order (both seats agree).** Check both sides, write the Doc, read it back, then Claude writes the note through the skill. A refused Doc write touches nothing. After any failure the Doc as read back is the truth.

7. **The record. Lead's call: sync's write-ahead entry only.** Before the Doc write, the binary writes a `pending` entry `{at, revision_id, quoted, replacement}` on the document's entry in the `gdoc:` block. The skill deletes it once the note is written (or the binary does, on a second call; plan decides). Nothing stays after success. The designer's lasting `edits` record is dropped, because the colour in point 8 makes it unneeded for clearing, and undo uses the receipt, the terminal, or the hub's git history. A new key, so older binaries refuse the note by name: a DECISIONS entry.

8. **Green. Lead's call: a green outside the house palette** (a light green, hex fixed in the plan, not #3DD67A and not #FFD500). Reason (sync): #3DD67A is Nail's own "claim to confirm" debt mark and would come home on export as `==🟢 words==`, making align noisy and the next publish ask about debt. An off-palette colour already exports as plain words, counted under `marks.other` (`TestAColourOutsideThePaletteIsPlainWordsAndCounted`). Painted over inserted characters only; a deletion paints nothing. Stays until Nail removes it or says "clear the green": a style-only batch, mask exactly `backgroundColor`, over runs of exactly that hex. The session counts remaining green passages at stop.

9. **Receipts and undo.** From an `ai!`: the sign, the change, one receipt in his thread with the reason and any deleted text; no reason comment, no card; one thing to close. From Claude Code: the change only, nothing to close. gdoc never resolves his thread. Undo is a new edit, only while the green words are still there exactly once and unchanged.

10. **Compiled Docs (retail card).** The skill finds the chapter file by quote among the files the compiled note's `compiled_from:` key names, below `# Prose`, rendered text, exactly once. It never edits the compiled body. No match or many: Doc edit only, "no md file was written", candidates named. The pending entry lives on the compiled note's block.

11. **Out of v1.** Claude Desktop chat proposes only. Multi-tab Docs refused. Presence app shows no mode. Hand edits in the Doc do not reach the note by themselves. Version history shows Nail's account name; the green and the receipt are the only authorship marks.

## Measure first (task 1 of the plan, merged and deduplicated)

1. `requiredRevisionId` with a stale id after one typed character elsewhere: expect 400 and no change.
2. One Words batch: the colour covers exactly the inserted span; the new words do not inherit bold, link or mark from the run before.
3. `insertText` at a span start after bold, a link, a yellow mark: inherited style.
4. Docx export carries the off-palette shading, and `gdoc export` writes plain words plus a `marks.other` count.
5. Replace the words an `ai!` thread is anchored on, both orders (delete then insert, insert then delete): anchored, moved, or detached.
6. A direct edit in a paragraph holding a colleague's pending suggestion elsewhere: their ids and range survive.
7. A direct delete over a colleague's pending suggestion: what Docs does (to confirm the refusal is needed).
8. A second browser open during a batch, then version history: whose name, how grouped.
9. `TestLiveExportRoundTrip` green or its 27 differences named, since the rendered comparison rests on it.

## v2 after the guardian and Codex (2026-11-07)

Guardian: vetoed six items, signed words replace and mass replace (each item its own exactly-once quote, strictly descending, count cap, no replaceAllText), signed the provenance fact gate (id from argv via AllowFile, the note's entry names the same id), no conflict on off-palette green (nothing in Go keys on the hex). Its signed sentence for the entry, once the vetoes are lifted: "A document handed in may be edited directly only at LevelEdit, opened by GrantEdit for the one id given on the command line beside a note that records publishing it, for one run, by a batch whose every range the guard derives from the body or from a span the command granted out of its read at the required revision, and nothing at that level can restyle outside its own insertion, touch another segment, retry a moved revision, or be reached from `gdoc mcp`."
Codex: smallest cut is one non-empty replacement in one plain paragraph; a pasted link picks the target but is not permission; stop both writes when the md source is ambiguous; the vault is not git (PRINCIPLES.md line 36), so no undo from git.

Lead's v1 cut:
- Kind: words replace only, non-empty insert, one plain body paragraph, the 11-03 grammar. Mass replace signed but deferred (Codex). Pure deletes, add/delete paragraphs stay suggestions (guardian vetoes 1, 2).
- Gate: link or this-session publish picks the Doc; the request must ask for a change; live review ai! counts. Go holds: positional URL required, flag requires --md, flag in flagsNeverOffered["propose"], no settings or note key turns it on, the binary never retries a 400.
- Ambiguous md source: stop both writes (Codex).
- No pending key (guardian veto 5 and skeptic; the drift check stops the next edit on that paragraph). No clear-the-green command in v1 (veto 3, skeptic); Nail removes by hand.
- Supersede rows and missing pins: the guardian's list above, carried into the entry and the plan.
- To Nail: veto 1 (deletes) and veto 6 (a colleague's ai! writes in his name) as questions 2 and 3.
