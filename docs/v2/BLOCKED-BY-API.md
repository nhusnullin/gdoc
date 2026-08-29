# Blocked by the API

Things gdoc cannot do because Google offers no way to do them. Kept separately
from [DECISIONS.md](DECISIONS.md) on purpose: a decision is a choice, and none of
these are chosen. They are the shape of the wall.

Every entry carries the verbatim refusal, so a future reader can retest rather
than re-argue. All measured 2026-08-29 against the live APIs.

Recheck this file when Google's release notes move. Anything that opens here may
delete a decision in the other file.

## Refused outright

| What | The refusal, verbatim | Workaround today |
|---|---|---|
| **A first-page header on a document that already exists.** This is where the Altery logo and "For internal use only" live. | `Invalid value at 'requests[0].create_header.type' (…HeaderFooterType), "FIRST_PAGE_HEADER"`. The enum is exactly `["HEADER_FOOTER_TYPE_UNSPECIFIED", "DEFAULT"]`. All six header and footer id fields answer `Unallowed field` on both `updateDocumentStyle` and `updateSectionStyle`. | Inherited free when a document is **born** from a docx carrying `<w:titlePg/>` and a first-page `headerReference`, or from a copy of one that has it. On an existing document, the finishing checklist. |
| **A table of contents.** | `Unknown name "insertTableOfContents" at 'requests[0]': Cannot find field.` Same for `createTableOfContents`, `updateTableOfContents`, `refreshTableOfContents`, `deleteTableOfContents`. The `Request` message has exactly 40 public members and none touches one. | A docx carrying a Word TOC field imports as a **live, refreshable** one. On an existing document, the checklist. Nail refreshes it himself. |
| **A page-number field in a footer.** | No request type inserts AutoText. | Same: born with it, or the checklist. |
| **A positioned (floating) image.** | `Unknown name "createPositionedObject" at 'requests[0]': Cannot find field.` Only `deletePositionedObject` exists. | Inline images. Apps Script's `Paragraph.addPositionedImage()` can do it, which is the single thing Apps Script offers that REST does not, and not worth a second execution surface. |
| **Minting a comment anchor.** | No refusal: Drive returns **HTTP 200 to every anchor string** and attaches it to nothing. 37 forms tried, including named range ids, heading ids, footnote ids, tab ids, both documented JSON shapes, and fabricated `kix.cmt` numbers. Google documents the behaviour: *"the anchor is saved and returned when retrieving the comment, however Google Workspace editor apps treat these comments as un-anchored comments."* | **None.** Anchor banking was proven to work and **rejected 2026-08-29**: seeding a marker on every sentence of every document is too much permanent machinery for a capability that only ever covers text gdoc wrote. gdoc does not attach comments to specific text, and says so. |
| **Emoji reactions.** | `"reaction"` appears **zero** times in both the Docs and Drive discovery documents. UI-only feature, no API surface. | None. Not needed. |
| **Deleting a comment that arrived by import.** | `403 insufficientFilePermissions`, to everyone including the file owner, permanently. An imported comment's author is a bare name string owned by nobody. Matching the display name does not help; nor does `w15:userId` with the right address. | `files.copy` drops all comments while keeping all anchors. That is how placeholders are shed. Never try to delete them. |
| **Updating a comment that arrived by import.** | `comments.update` returns **HTTP 200 and silently does nothing**. Only `replies.create(action="resolve")` works on them. | Do not update imported comments. |
| **Pinning a revision of a native Doc.** | `keepForever` returns 200 and never persists. Documented as *"only applicable to files with binary content in Drive"*, and a Doc has none. Five authored states merged to two within minutes. | The section fingerprints in the front matter. This is not a workaround for a gap: it is the only correct design. |

## Gated behind the Developer Preview

Application submitted 2026-08-29, awaiting Google. Cloud project `4326046141`.

The public discovery document exposes **40** request types. The HTML reference
lists **49**. The nine missing ones are the gate:

```
insertComment      addCommentReply     updateCommentPost
deleteComment      deleteCommentReply
acceptSuggestion   rejectSuggestion    deleteSuggestion
writeControl.writeMode
```

| What | Status |
|---|---|
| **Writing a suggestion** (`writeMode: SUGGEST`) | `400 "Unsupported WriteControl mode."` Note it changed behaviour during the day: that morning the same call returned **200 and silently made a direct edit**. Never send it and trust the status code. |
| **Accepting, rejecting or deleting a suggestion** | Gated. So even with suggestions, gdoc could not resolve them for Nail. |
| **Creating an anchored comment properly** (`insertComment`) | Gated. Would retire anchor banking entirely. |
| **Reading who wrote a suggestion** (`PostAuthor`) | Gated. Note authorship is **output only** and can never be set: a suggestion made through the API is authored by the credential. A tracked-changes import preserves a custom `w:author`, so the older route is better on attribution. |
| **Reading comments with real character ranges** (`commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED`) | Gated, needs `includeTabsContent=true`, accepts `documents.readonly`. **The most valuable item here**, and read-only, so no new authority. |

Two consequences worth holding. None of the preview surface is in the discovery
document, so a generated client cannot build these calls **even once allowlisted**;
they must be hand-rolled JSON. And Pre-GA is "AS IS" with no promise of GA, and
the terms bar access for users outside the domain.

## Closed, so nobody reopens them

| Route | Why it is dead |
|---|---|
| **Editing by export, modify, re-import** | Comments that arrive by import can never be deleted, so every round leaves permanent duplicates. Counts went 4, 7, 10, 13, 16. From round two the threading corrupts and a comment's own body reappears as a reply to itself. |
| **ODT instead of DOCX** | Identical in every measured respect: same anchors, same spans, same authors, same suggestions. An ODT `files.update` replaces wholesale too, proven with a body-identical upload that still destroyed a paragraph and both smart chips. |
| **RTF** | Anchors import mangled, three ranges where there should be two, author lost entirely, `\deleted` text silently dropped. |
| **HTML** | Re-importing Google's own HTML export of a commented document yields zero comments and zero anchors. |
| **Apps Script** | A subset of the REST API, not a superset. No comment API, no table of contents, no page numbers, no suggestions. Solves one of these entries, positioned images. |
| **Drive revision history as a diff base** | Revisions merge within minutes, cannot be pinned, and the list silently omits older ones. |
| **A partial-body update by any protocol route** | None exists. Multipart means metadata plus one media representation. Resumable `Content-Range` chunks the same whole replacement. Conversion on `files.update` explicitly replaces the full contents. |
