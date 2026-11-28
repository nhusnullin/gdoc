# API scout report: edit in place, marked green, note kept in step

Desk research only. No file edited, no API called. Tags: DOC = documented by Google, REPO = already measured here, MEASURE = needs a live test.

## Already in the repo (do not repeat)
- Docs revisionId is read and sent back as `requiredRevisionId` in restyle tests (go/cmd/gdoc/restyle_test.go). The 2026-09-10 restyle run asserted the source revision did not move. REPO.
- In-place `updateTextStyle` works and keeps comment anchors and suggestion ids. Replacing a whole body destroys all anchors (355 of 355 chars). REPO (MEASURED-archive).
- `createNamedRange` works over a pending insertion; rejecting the suggestion removes the range; accepting keeps it (same id, same span). REPO. Partial accepts: not measured.
- `backgroundColor` round-trips. The six house marks are #FFD500 yellow, #3DD67A green, and four more. `export` writes them as `==words==`. REPO. Docx `w:shd` export for a direct-edit run was never measured (the 2026-11-03 notes say so).
- Drive revisions: merge within minutes, cannot be pinned (`keepForever` is a no-op on a Doc), the list omits older ones. Closed as a diff base. REPO (BLOCKED-BY-API).
- Nothing in the tree ever sent `requiredRevisionId` on a content write, `targetRevisionId`, `changes.*`, or Drive Activity.

## 1. requiredRevisionId vs targetRevisionId
- `requiredRevisionId`: if the Doc is not at that revision, nothing is applied and you get 400. DOC.
- `targetRevisionId`: your edit is applied against your read, then the Docs server merges it with later collaborator edits into a new revision. It works only for "recent" revisions, usually a few minutes, less on busy Docs. Too old gives 400. DOC.
- Merge is operational transform on positions. It keeps our insert next to the text it was made against, but it cannot know a person rewrote that sentence. Our `deleteContentRange` could remove changed text. Not safe for a "replace these words" edit. My answer: use `requiredRevisionId` for deletes and replaces. A 400 means re-read and re-find the quote. `targetRevisionId` is only defensible for a pure insert. MEASURE: how often a 400 happens with a person typing elsewhere.
- Sources: https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/batchUpdate , https://developers.google.com/workspace/docs/api/how-tos/best-practices , https://developers.google.com/workspace/docs/api/reference/rest/v1/documents

## 2. Marking: highlight vs named range
- Highlight (`backgroundColor`) is visible to every human and survives in docx and Markdown export. It is also what a person can clear by hand, and a person can paint the same colour themselves. So it is a mark for people, not proof of authorship. REPO for the colour path.
- Named range: indexes update automatically as text is added or removed. DOC. It can be split into several pieces by user edits (the docs' own sample warns of it). DOC. It can vanish: samples throw "no longer present". DOC.
- Not documented: what happens when someone types inside, at either edge, or deletes the whole span. Not documented: whether names must be unique (samples return a list per name, so several ranges per name look allowed). MEASURE both.
- Recommended pairing: highlight for people, one named range per edit (name like `gdoc-edit-<id>`) for us. Read both back. If the highlight is gone but the range remains, a person cleared it. If the range is gone, treat the edit as "adopted by a person" and stop tracking. Do not make the range the only truth.

## 3. Undo
- No API restores an older revision of a Doc. `revisions` has get, list, update (patch), delete only. `keepForever` is for binary files only. DOC. Repo already measured the merge and omission. REPO.
- Options, all ours:
  1. Reverse batch: `deleteContentRange` over the new span, `insertText` of the old text, guarded by `requiredRevisionId` or by re-finding the new text exactly once. Works only while the new text is still there unchanged.
  2. Keep the old text in the Drive comment thread gdoc already posts (durable, visible to all, no new scope). Plain text only, no markdown, as the rule says.
  3. Keep it in the Markdown note's `gdoc:` block (private, git-friendly, but no one in the Doc sees it).
  4. Named range name: not recommended. Name length limit is not documented, and the old text is shown to nobody. MEASURE if you want it.
- My answer: comment (for people) plus the note block (for us). Undo is a new edit, never a restore.

## 4. Knowing the Doc changed
- `files.get` `version`: "monotonically increasing", rises on every server-side change, including ones users cannot see. `modifiedTime`: last change by anyone. `lastModifyingUser`: only when a signed-in user did it. DOC. `headRevisionId` and `md5Checksum`: binary files only, so empty for a Doc. DOC.
- Check: `files.get?fields=version,modifiedTime,lastModifyingUser`. MEASURE: whether the guard's `checkFields` allows these today, and whether `version` moves on our own writes only, or also when someone just opens the Doc.
- `changes.list` with `changes.getStartPageToken`: pull model, oldest first, covers the whole Drive, no content. DOC. Needs a new guard shape: the guard refuses everything outside `/drive/v3/files`. Same host (www.googleapis.com), so no new host. Not better than `files.get` for one Doc.
- Push (`files.watch`, `changes.watch`): needs a public HTTPS endpoint with a valid certificate. Files channel lives up to 1 day, changes up to 1 week, no auto-renew. Notification has headers only and an empty body. DOC. For a local CLI no direct route works. A tunnel or relay would be a new host and a new component. Not worth it; polling every second already works and costs 1 read per tick.
- Drive Activity API (`activity.query`, itemName=items/ID): new host `driveactivity.googleapis.com`, scope `drive.activity.readonly`, not in the token today. DOC. It gives who and when and an action type, with a time filter. The page I could read did not show the field list; from memory the Edit action carries no text and no revision id. MEASURE if you ever want it. It cannot say what changed.

## 5. Diff between two revisions
- No diff route. DOC says nothing on one. `Revision.exportLinks` exists ("links for exporting Docs Editors files"). Whether `revisions.list` gives usable export links for old Doc revisions, and how many survive, is not documented. Repo says the list omits older ones and merges. REPO + MEASURE (cheap: list, then fetch one export link).
- Guard refuses `/revisions` today, so this is a deliberate new door.

## 6. Limits
- Per project: 3,000 reads and 600 writes per minute. Per user per project: 300 reads and 60 writes per minute. Over quota gives 429; retry with exponential backoff. Charges for overage planned later in 2026. DOC.
- A `batchUpdate` counts as one write however many requests it holds, and all requests in a batch apply atomically. DOC (the limits page lists `update_doc` as 1 write; the atomic claim is on the batchUpdate page). So: many small edits in separate calls is a bad idea (60 a minute, also a long history, also a race window between each). One batch per paragraph or per run, ordered from the highest index to the lowest, is the documented way.

## 7. How a direct edit looks to others
- Not documented on the pages I read. Expected, from product behaviour I did not verify here: the edit appears live to anyone who has the Doc open, as a change by the signed-in account; version history groups edits into sessions by author and time and names the author as that account. An API edit therefore shows as Nail (or whoever the token is), not as "gdoc". The mark (green, plus the comment) is the only sign it was a robot. This matters for the "authorship is the mark" invariant.
- MEASURE by eye on a throwaway Doc: open it in a second browser, send one batch, watch the live view and File > Version history.

## 8. Other things gdoc does not use
- Apps Script and Drive Activity: new host and scope, still not worth it. `replaceAllText` replaces every match, which breaks the exactly-once rule. `documents.get` with a `fields` mask of revisionId is a cheap pre-write check (check the guard's read grammar).
- Own snapshot beats a Google diff: after each write, store a hash of each edited span in the note and compare next time.
- `deleteNamedRange` by id or by name: the cheap way to "clear our green" together with a style reset. Guard has `AllowMarker` for create only, so delete needs a grant.
- Drive Activity: new host and scope, gives who/when only. Not recommended.
- New host or scope: Drive Activity, push (public endpoint), Apps Script. Same host, new guard shape: `changes.*`, `revisions.*`.

## Measurements we need before building
1. `POST docs.googleapis.com/v1/documents/{id}:batchUpdate` with `writeControl.requiredRevisionId` set to the id from the last read, after a second browser typed one character elsewhere. Expect 400 and no change. If it returns 200, the lock does not protect us and the whole mode needs a different guard (re-read and compare).
2. Same call with `targetRevisionId` and the same stale id, one pure insert and one delete of a quote someone just edited. Compare the result. Decides whether `targetRevisionId` is ever allowed (inserts only) or banned.
3. One batch of `deleteContentRange` + `insertText` + `updateTextStyle(backgroundColor #3DD67A, fields exactly backgroundColor)` on a throwaway Doc. Read back with `PREVIEW_WITHOUT_SUGGESTIONS`: text, colour on exactly the inserted span, and the new revision id in the response. Decides the batch grammar.
4. `files.export` as docx of that Doc, then look for `w:shd` fill 3DD67A on exactly those runs; then `gdoc export` and look for `==words==`. Decides whether the note stays in step through the existing export path.
5. `createNamedRange` over the inserted span, then by hand: type inside it, type at its start, type at its end, delete the whole span, select-all and retype. After each, `documents.get` and list the ranges. Decides whether a named range can find "our green" later, or only the highlight can.
6. Two `createNamedRange` calls with the same name on different spans, then `deleteNamedRange` by name. Decides one-range-per-edit naming versus a shared name.
7. `GET drive/v3/files/{id}?fields=version,modifiedTime,lastModifyingUser` before and after our write, after a person types, and after someone only opens the Doc. Decides if `version` is a usable cheap "changed" signal and whether it can tell our write from theirs.
8. One batch while a second browser has the Doc open, then File > Version history. Decides the answer to item 7: whose name shows, and how edits are grouped.
9. `GET .../revisions` then one old `exportLinks` entry, minutes and a day apart. Decides if any revision diff or undo base exists. Needs a guard door; run from a probe.
10. 61 one-request batches in a minute. Confirms the write quota and the 429 shape.

## Sources by section
1: developers.google.com/workspace/docs/api/reference/rest/v1/documents/batchUpdate , .../docs/api/how-tos/best-practices , .../docs/api/reference/rest/v1/documents
2: .../docs/api/how-tos/named-ranges
3, 5: .../drive/api/reference/rest/v3/revisions
4: .../drive/api/reference/rest/v3/files , .../drive/api/guides/manage-changes , .../drive/api/guides/push , .../drive/activity/v2/reference/rest/v2/activity/query
6: .../docs/api/limits
