# Sync: keeping the note and the Doc the same under direct edits

Read: the package docs named in the brief, `propose/span.go`, `frontmatter/schema.go`, `house.yaml`, `compile.py` and the v0.20 compiled note. Nothing edited, nothing called.

Core claim: the two sides stay the same only if both edits are the same patch, applied where both sides already agree. Everything below follows from that.

## 1. Mapping

**Recommendation: compare in rendered space, never in markdown source.** Render the note's prose through the same goldmark setup `internal/body` uses, to plain text per paragraph, keeping a map from each text byte back to its source byte. Find the old words there, then map back to the source span to edit.

Why: the Doc holds rendered text, the note holds source. `body` turns on the typographer (`TestSmartQuotesAreTheCharacters`), so a straight `"` or `--` becomes a curly quote or a dash in the Doc. `**`, `[x](url)`, `==`, escapes and entities exist only in the note. A wrapped md line is one Doc paragraph. A plain string search misses, and the miss looks like drift.

Same machine both ways. Note to Doc: `propose.FindSpan` rules on the Doc (exactly once, not inside a pending suggestion via `anySuggested`, not across a hole via `matches`). Doc to note (`ai!`): the same exactly-once search over the note's rendered text. The Doc gets the typographer's output of the new md words, so the next publish makes no diff.

| Construct | Maps one to one? | Do |
|---|---|---|
| Plain prose, list item text, table cell text | yes, in rendered space | edit both. Escape `\|` in a cell |
| Prelude: cover, contents, front-matter tables | no, made from the author's keys | refuse. Change the key, say the Doc changes on the next publish |
| Headings | no: house number `1-` in the Doc, slug anchors in the note, the Doc contents list keeps its own copy (`paragraphs()` in `span.go` never walks it) | suggest instead |
| Footnote text | no, separate segment, never in `FindSpan`'s walk | md only, and say the Doc was not changed |
| Span crossing a footnote mark, chip or picture | no, a hole | refuse, as `FindSpan` already does |
| Span touching a `==mark==` or pencil | style inheritance unmeasured, and green would erase the author's mark | suggest instead |
| Span touching link text, or bold/italic boundary | inherited style unmeasured | refuse in v1, quote around it |
| Line break inside a cell or paragraph | `docs/backlog/a-line-break-hides-from-read-and-propose.md` | refuse |

v1 rule: old and new words are plain prose, one paragraph, one uniform style, on both sides.

## 2. Drift before the edit

**Recommendation: a per-paragraph check, no snapshot.**

The span check alone (old words exactly once on both sides) is not enough. A colleague may have changed the rest of the sentence. Claude wrote the new words to fit the note's sentence, and they can be wrong grammar or wrong meaning in the Doc's sentence. So check the whole paragraph that holds the span. It must be equal on both sides in rendered space. On the Doc side, compare the text as written: suggested deletions in, suggested insertions out.

Cost: one `documents.get`, which the write needs anyway, plus one file read. The revision lock covers the seconds after.

A snapshot would only say which side moved, and the action is the same either way: stop. Per-paragraph hashes as a diff3 base are a good later item for `gdoc-align`, which today judges "both moved" by eye.

When the paragraphs differ: change nothing. Show both paragraphs. Offer three routes: bring the Doc paragraph into the note first (align for one paragraph), then redo the edit; Doc only; or a suggestion as today.

## 3. Order and failure

**Recommendation: check both sides, then write the Doc, then write the note.** It is Nail's order ("show me first chnages in openned 0.10 doc, then we do chnages in md") and the safe one: the Doc write is the one that gets refused, so doing it first makes a refusal cost nothing.

Add a small write-ahead entry. Before the Doc write, record `{at, revision_id, quoted, replacement}` as pending on the document's entry in the `gdoc:` block. Delete it once the note body is written. Nothing stays after success. Cost: a new key, and the strict reader means an older binary refuses the note by name. That needs a DECISIONS entry.

| Event | Doc | Note | Truth after |
|---|---|---|---|
| 400 from `requiredRevisionId` | unchanged (documented) | unchanged | both as before. Re-read, re-check, rebuild, retry at most twice, then stop |
| Answer lost (`OutcomeUnknown`) | maybe | unchanged | read the Doc. A retry is safe: if the first landed, the revision moved and the retry gets 400 |
| Lands, `verified: false` | changed | write only if the read-back holds the new words. Colour missing alone: write, report | the Doc as read back |
| Note write fails (file changed: vault sync or a live session's block write) | changed | re-read, re-find old words exactly once, retry | the Doc. If the retry fails, print the exact edit and keep the pending entry |
| Session dies between writes | changed | old | the Doc. The next session reads the pending entry: new words in the Doc, write the note; old words, drop it; neither, ask |

## 4. The compiled Doc

What exists today: the paired note is the compiled `compiled/..._v0.20.md`. The `gdoc:` block lives there, with 15 proposals recorded. `compile.py` writes an author key, `compiled_from:`, with one file name per chapter, plus the addendum and `00_shared/glossary.md`. The `gdoc:` schema (`frontmatter.Entry`) has no sources field. Export knows nothing about multi-file sources. `compile.py` refuses to rewrite a version ("a version is never rewritten").

**Recommendation:** the session finds the chapter file by quote, inside the files the note's `compiled_from` names, read-only, in the skill. No binary change and no new key in gdoc's block. That key is the author's, and the binary judges nothing.

- Search below `# Prose` in each source, in rendered space. One file holds the words exactly once: edit it. Otherwise do the Doc edit, say no md file was written, and name the candidates.
- Never edit the compiled note's body. It is the record of what v0.20 was. Its block carries the pending entry.
- Headings and the glossary order are rewritten by the compile (`name_the_heading`, `glossary_appendix`). Heading edits are refused anyway. Glossary entries are copied word for word, so they match.

A note with no `compiled_from` is its own source.

## 5. Green and export

The house green `#3DD67A` is already an author's mark (`==🟢 words==`, `debt: true` in `house.yaml`). Painted by the robot, export brings it home as `==🟢 words==`, align shows every edited paragraph as changed, and a merged copy makes the next publish ask about a green debt. The yellow-door plan's `#FFD500` has the same problem.

**Recommendation:** the note never carries a mark for a robot edit. The robot colour should be a hex outside the palette. `export/doc.go` already writes an unclaimed colour as plain words and counts it under `marks.other` by hex (`TestAColourOutsideThePaletteIsPlainWordsAndCounted`). A clean round trip with no code. The colour is the designer's call, so this is a flag to them. Keeping a palette colour means export must decide "this green is gdoc's", which is identity by colour.

Green never matters for sync: its truth is equal text, and the pending entry is its only record.

## 6. Mass replace

One batch, highest index first, under one `requiredRevisionId`, is right: descending keeps every lower index valid, and the batch is atomic and one write. The traps:

- `FindSpan` wants exactly once. This needs a find-all with its filters. Never `replaceAllText`.
- Word boundary and case: "card scheme licences" and "Card scheme licence" at a sentence start. Count each variant on both sides.
- A match inside a colleague's pending suggestion, or across a chip, link or mark edge: skip it and list it. Never delete inside their suggestion.
- Overlapping matches (`matches` steps one byte): refuse.
- Matches in headings: the Doc contents list keeps the old text. Matches in footnotes, headers and footers are outside the body walk. The md count will be higher there. Report it, never hide it.
- The compiled case: the md count includes working notes above `# Prose`, and the Doc count does not. Show both counts and ask once.
- The nine md writes are not one transaction. List any that failed. Read-back: the old count drops by exactly the number sent.

## 7. Measure before building (beyond the scout's ten)

1. Replace the words an `ai!` thread is anchored on. Try both orders: delete then insert, and insert then delete. Does the thread stay anchored, move to the new words, or detach?
2. `insertText` at a span start after bold, after a link and after a yellow mark. What style does the new text carry?
3. A direct edit in a paragraph holding a colleague's pending suggestion somewhere else. Do their ids and range survive?
4. Export an off-palette robot hex. Expect plain words and a `marks.other` count.
5. Get `TestLiveExportRoundTrip` green or its 27 differences named (`docs/backlog/live-export-round-trip-has-unnamed-differences.md`). The rendered-space comparison rests on that round trip.
