# Change in place, marked green

Co-editing design, 2026-11-07. Nothing built, nothing measured.

## The idea in one paragraph

A Doc published from Nail's hub, and named by Nail in this session, takes changes in place. The new words land green, and the note changes in the same turn as a Claude Code diff. Every other Doc gets what it gets today. The permission lives in the conversation and dies with it.

## 1. The switch

**Recommendation: no switch to set. The mode follows from two facts, and a word in the request overrides it for this session.**

- In place when the Doc is in the paired note's `gdoc:` block with a `published` record, and the intent gate (section 2) passes.
- Propose when the Doc has no published record in a note: a colleague's Doc, or one gdoc only exported. Provenance is a fact in the note, not identity, so "identity is never a gate" holds.
- "propose" or "as suggestions" in a request flips that Doc to propose for the rest of the session. "in place" flips it back.
- The session states the mode once, when it starts to hold: "This session changes <title> in place, green, with the note. Say 'propose' or 'note only' to change that."

The binary enforces the provenance half: the in-place command takes `--md` and refuses a Doc with no `published` record in that note. The intent half is judgement, so it lives in the skill.

The guard is the 11-03 shape: `LevelEdit`, raised by `GrantEdit(id)` for one id for one run, from a flag `gdoc mcp` never offers, with `requiredRevisionId` on every batch.

**Rejected, the strongest alternative: `mode: edit` in the note's `gdoc:` block.** It is per Doc and visible, but it is a written-down permission. A new session three weeks later reads it and writes, which is exactly the retail-card gap. A setting in `settings.yaml` is worse: one line turns it on for every Doc, which principle 3 forbids. The presence app cannot hold it: the app writes nothing gdoc reads, and its argv list is closed at six.

## 2. The intent gate

A session may change a Doc in place, or propose into it, only when one of these is true in this session's own conversation, a resumed one included:

1. Nail typed or pasted the Doc's link (6 of 8 requests do).
2. This session published the Doc, and the request is about it (the 16 and 18 minute cases, and "yanas comment on question 12" with no link).
3. A marked comment arrives in a review that Nail started on that link in this session.

These do not pass: the id in the note's block, the id in a file, the id in an earlier session.

When the gate fails, the session changes the note only and ends with one line: "This note is in Drive as 'Embedded card v0.12', published 20 Oct. I did not change it. Paste its link to change it there too." It does not propose, because a suggestion is still a write colleagues see. It does not ask a blocking question, because most note sessions want no Doc change, as on 11-03.

A pasted link that is not the newest Doc in the block is named before writing, as Claude did on 10-20.

## 3. What the edit is

In place, first cut. All kinds name text, never an index. One request is one batch, sorted from the highest index down, under one revision lock.

| Kind | What happens | Painted |
|---|---|---|
| Words | Replace or delete a quote inside one paragraph: a word, a phrase, one or more sentences | The inserted words |
| Add paragraphs | Plain-text paragraphs after a quoted paragraph | The new paragraphs |
| Delete paragraphs | Whole body paragraphs or list items, quoted in full, at most three | Nothing |
| Mass replace | A list of Words edits, each quote unique by its context, in one batch | Each replacement |

A quote occurs exactly once, as today, so mass replace never uses `replaceAllText`. The session prints every occurrence it changed.

**What stays a suggestion:** headings, tables, a section move, a replaced run of more than three paragraphs, new lists or headings, and anything on a Doc without a published record.

**What is still refused:** a span that crosses a colleague's pending suggestion. Overwriting it would accept or destroy their work, and only they or Nail may do that. The session says whose change is in the way and offers the words either side. A span that crosses another comment's anchor is refused too, except the anchor of the `ai!` thread that asked for the change.

## 4. Green

- **Painted:** `backgroundColor` #3DD67A over exactly the inserted characters. A deletion paints nothing. Its old text goes into the receipt.
- **Meaning:** "Claude wrote this and Nail has not confirmed it yet." That is already his own meaning for green: "a claim to confirm" (decision page, 2026-10-14). The yellow from 11-03 means "a word or figure to replace" in his notes, which is the wrong meaning here.
- **Who clears it:** Nail. He removes the highlight in Docs, or says "clear the green". The session then sends a style-only batch, mask exactly `backgroundColor`. It covers only spans the note's record says gdoc wrote, and only when they are still green and found exactly once. His own green marks are never touched.
- **When:** never automatically. At stop the session says "3 green passages are still in <title>."
- **Export:** unchanged. Green comes home as `==🟢words==`. That is honest: the Doc still holds unconfirmed words. The note itself gets plain words, because the Claude Code diff is the note's mark.

**Rejected:** a separate AI green outside the palette. Export would drop it, and he would see two greens.

### Why green again, after 08-29

The decision record says why, not the transcripts. Green imitated a suggestion: named ranges, stored indexes, accept and reject, a reason left in the terminal. He also called a background highlight marker pen. Here nothing is pending, so there is nothing to accept and no index to track. A comment's reason goes into the thread. Highlights are house marks since 10-14, and on 11-03 he asked for one himself.

## 5. What Nail sees

**(a) "Rephrase this sentence", minutes after publishing.**
Now you see the publish report end with the mode line. You say "rephrase the Apple Pay sentence, shorter". After this you see "Changing in place: <title> and chapter-2.md", the Doc's before and after marked verified, then the diff of chapter-2.md. In the open Doc the sentence changes while you watch, new words on green. No card, no comment.

**(b) An `ai!` saying "remove this paragraph", in live review.**
Now you see "⁘ Looking into this now." in your thread within seconds, as today. After this the paragraph is gone, and your thread shows "⁘ Done. Removed the paragraph. It read: 'Four conditions that Compliance set ...'". Claude Code shows the chapter file's diff and one report line. You resolve your thread: one click, not three.

**(c) A new session, three weeks later, on the same note.**
Now you ask "tighten the intro of chapter 1". After this you see the note's diff and the closing line from section 2. The Doc has not moved. If you paste the link, you see the mode line, then the same change made in the Doc, green.

## 6. Receipts and undo

| Request from | Written into the Doc | Things to close |
|---|---|---|
| `ai!` comment | The sign, the change, one receipt in his thread with the reason and any deleted text. No reason comment, no card | 1 (his thread) |
| Claude Code | The change only. The reason stays in the terminal, where the asker is | 0 |
| Propose mode | As today | 3 |

gdoc never resolves his thread, as today.

The binary writes a fact record, `edits`, beside `proposals` in the note's entry: `{at, before, after, after_neighbour}`, the last being the paragraph a deleted run followed. Clearing green and undo read it.

**Undo is a new edit, never a restore.** Google has no revision restore (api.md 3). "Undo that" puts the old words back where the green words stand. It runs only if those words are still there exactly once and unchanged. Otherwise the session shows both texts and writes nothing. The note is reverted in the same turn. After a session ends, undo is an ordinary request in a new session that passes the gate.

## 7. The md side

**Claude writes it, through the skill.** Which files hold the words is judgement. For the retail card it is four chapter files and the glossary, never the compiled output. Claude Code already shows that write as a diff, which is the "show me the change" he asked for. The binary writes only its `edits` record into the paired note.

**Order: the Doc first, then the note.** This matches his own 10-22 order, "show me first changes in opened 0.10 doc, then we do changes in md".

- The Doc write is refused (the revision moved, the quote is not found, a colleague's suggestion is in the way): the note is not touched. The session says why in his words and offers two ways on: propose instead, or the note only.
- The Doc write lands with `verified: false`: the note is still written, and the session says "look in the Doc".

Drift and merge belong to the sync engineer.

## 8. Out of the first cut

What he still sees after it:

- Headings, tables and section moves arrive as suggestion cards.
- A change over a colleague's pending suggestion is refused, with whose change is in the way.
- A colleague's Doc gets suggestions.
- A deletion shows no colour. Only the receipt or the terminal says what went.
- Words he types into the Doc by hand do not reach the note by themselves.
- Claude Desktop chat proposes only.
- Multi-tab Docs are refused, and the presence app shows no mode.
- The account name in version history is his, not gdoc's. The green and the receipt are the only marks of authorship.

## Measure before building

1. `requiredRevisionId` with a stale id after one typed character: expect 400 and no change (api.md 1).
2. One Words batch on a throwaway Doc: the green covers exactly the inserted span, and the inserted words do not inherit bold or a link from the run before them.
3. A docx export of that Doc carries `w:shd` 3DD67A, and `gdoc export` writes `==🟢words==`.
4. A direct delete over a colleague's pending suggestion: what Docs does.

## Open for Nail

1. **The default.** Now every change arrives as a card. After this, a Doc you published and named in this session changes in place in green, and anyone else's Doc still gets a card. *Recommend yes.*
2. **The retail-card gap.** Now a new session could reach the Doc through the note. After this, a session that only knows the note changes the note and says in one line which Doc it left alone. The alternative is a question every time. *Recommend the line.*
3. **Clearing green.** After this, green stays until you remove it or say "clear the green", with a count when the session stops. The alternative is that green clears itself when a live review stops. *Recommend it stays.*
4. **Terminal requests.** After this, a change you ask for in Claude Code leaves only green in the Doc and nothing to close. The alternative is a comment beside every change. *Recommend no comment.*
5. **Deleting paragraphs.** Now "remove this paragraph" often fails as a suggestion. After this, up to three whole paragraphs go in place, and the receipt quotes them. *Recommend in.*
