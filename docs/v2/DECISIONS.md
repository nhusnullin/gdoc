# gdoc v2 decisions

Dated, replaceable. Each names the principle it serves, or says it is a domain
choice with none above it. Same convention as [PRINCIPLES.md](../../PRINCIPLES.md).

v2 is a rewrite in Go. The three principles carry over unchanged. What changes is
which decisions implement them, and this file is where those start.

## 2026-08-29. gdoc never replaces the body of a document that already exists.

Serves principle 3. Uncertainty never resolves toward the destructive answer, and
here the destruction is silent.

Two ways exist to change a Google Doc gdoc did not create.

**Replace the body.** Export it to `.docx`, edit that, upload it back with
`files.update` and conversion. The document id survives, and this is the only way
to create a suggestion, because a `.docx` carrying Word tracked changes imports as
real accept/reject suggestions.

**Change it in place.** `documents.batchUpdate`. Nothing is replaced.

Measured live on 2026-08-29, not reasoned about:

| | replace the body | in place |
|---|---|---|
| comment anchors | **100% destroyed** | **100% preserved**, 355 of 355 characters across three anchors |
| pending suggestions | destroyed and recreated with new ids | byte-identical ids across 22 batches |
| smart chips | flattened; a date chip loses its date and keeps the string | untouched |
| Google Drawings | survive | untouched |
| footnotes, lists, tables, images | survive | untouched |

The anchor result is the one that decides it. Changing one paragraph destroys the
anchors on paragraphs that were never touched, and **the API reports every one of
them as healthy afterwards**. `comments.list` returns the original `anchor` and
`quotedFileContent` unchanged, because both are stored strings rather than live
pointers. A detached comment does not render in Google Docs, so the threads simply
vanish from the screen while the API insists they are fine. Nail confirmed this by
eye before the mechanism was understood.

So the rule is not a preference. Replacement is unusable on any document a person
has worked in, which is every document worth changing.

**In place is the decision. Replacement is reserved for creating a new document,
where there is nothing to lose.**

### What it costs, stated rather than hidden

Three things cannot be created by any Docs API request, each confirmed with a
verbatim refusal:

- **A first-page header**, which is where the template holds the logo and "For
  internal use only". `HeaderFooterType` is exactly
  `["HEADER_FOOTER_TYPE_UNSPECIFIED", "DEFAULT"]`, and all six header and footer
  id fields answer `Unallowed field` on both `updateDocumentStyle` and
  `updateSectionStyle`. `createPositionedObject` does not exist.
- **A table of contents.** `insertTableOfContents` and every other spelling
  answers `Cannot find field`. The `Request` message has exactly forty members and
  none touches a TOC.
- **A page-number field in a footer.** No request inserts AutoText.

### How the cost is paid

gdoc detects what is missing and **writes a finishing checklist into the document**,
as its own page after the cover, with checkbox bullets and the exact menu path for
each item. The block is marked with a named range so it can be removed in one call.

The general rule this is an instance of:

> **Where gdoc cannot act, it says so in the document, with the exact clicks.
> Never a silent omission.**

That is principle 3 again. A document that quietly lacks its contents list is
"I do not know" resolving to "ship it anyway".

Driving the browser was considered and rejected. It would have worked, and it
would have added a signed-in Chrome to the dependencies of a tool whose first
principle is that it runs where nothing can be installed. The checklist costs the
reader five seconds and cannot break when Google moves a menu.

### Consequences

- `restyle` styles in place and never replaces a body.
- `align` proposes; it does not rebuild the document.
- `publish` may render and upload freely, because the document does not exist yet.
- `gdoc release` refuses to export a PDF while the finishing checklist is still in
  the document. Instructions in the body are only safe if they cannot be sent by
  accident.
- Never call `deleteHeader` on a first-page header. It succeeds, and the header can
  never be recreated: `createHeader` then answers "Default header already exists"
  and `firstPageHeaderId` is gone for good. One-way door, no warning.

### The one thing that changes this decision

`writeControl.writeMode: SUGGEST` on `documents.batchUpdate` would make a
suggestion surgically, with no replacement and none of the damage above. It is
gated on the calling Cloud project being allowlisted for the Google Workspace
Developer Preview Program, and on a project that is not allowlisted it returns
**HTTP 200 and performs a direct edit**, with nothing in the response to say so.

If project `4326046141` is enrolled, suggestions come back and this decision keeps
its shape: still no replacement, and now proposals in the document rather than in
the terminal. Until then, approval happens in the terminal before anything is
written.

Never send `writeMode: SUGGEST` and trust the 200. Probe the capability against a
throwaway document first, once, and cache the answer.

## 2026-08-29. Never trust a success. Verify with a second, independent probe.

Serves principle 3.

Four Google APIs reported success while doing something else, in one day:

1. `writeControl.writeMode: SUGGEST` returns 200 and makes a direct edit.
2. `comments.list` reports detached, invisible comments as healthy and anchored.
3. `files.export?revisionId=N` returns the head and ignores the parameter.
4. `revisions.update keepForever` returns 200 and never persists.

So a status code is not evidence. Where an operation matters, gdoc confirms it by a
different route than the one that performed it. The `.docx` export is the honest
witness where the JSON API is not: comment attachment is only truthfully readable
from `word/comments.xml` and the `commentRangeStart` and `commentRangeEnd` markers.

## 2026-08-29. gdoc marks its own changes by colour, never by highlight.

Domain choice, no principle above it, but it exists only because of the decision
above: gdoc cannot create a real suggestion, so it has to show its work some other
way.

**Insertions are green. Deletions are red and struck through. Nothing is
highlighted.**

| | `foregroundColor` | also |
|---|---|---|
| proposed insertion | `rgb(0.18, 0.60, 0.35)` | nothing. No underline, no background |
| proposed deletion | `rgb(0.80, 0.25, 0.22)` | `strikethrough: true` |

Nail chose this from three shades on
`gdoc PROPOSAL MARKING styles`, judged in real sentences at body size rather than
on a swatch. Two paler variants exist in that document if this pair reads too
loud in a long paragraph.

The reason it is colour and not something else is that **it imitates how Google
Docs renders a real suggestion**. Reviewers already read green-and-underlined as
"proposed" and red-and-struck as "proposed for deletion". gdoc is borrowing a
vocabulary the reader already has, so nothing has to be explained in the document.

### What was rejected, and why

- **Background highlight.** Nail's call. A background colour dominates a page,
  reads as marker pen rather than as a proposal, and survives into a PDF looking
  like a mistake.
- **Emoji reactions.** No API at all. `"reaction"` appears zero times in both the
  Docs and the Drive discovery documents. It is a UI-only feature.
- **An anchored comment on the changed text.** This is what we actually wanted, and
  Drive will not do it: the documented anchor JSON is accepted, echoed back by
  `comments.list` forever, and produces no `commentRangeStart` in the export. An
  anchor can be *reused* from an existing comment, never *minted*.
- **A footnote.** Visible and unobtrusive, but there is no `deleteFootnote`
  request. A marker gdoc cannot remove is a one-way door.
- **A paragraph change bar** (`borderLeft`). Kept in reserve for a whole rewritten
  paragraph, where styling every word would be noise.

### Rules that come with it

- **A named range underneath every proposal**, `gdoc:proposal:<id>` or
  `gdoc:proposal-delete:<id>`. The colour is for the human; the range is how gdoc
  finds its own work again. `createNamedRange` returns only an id, and
  `documents.get` reads the current span back.
- **Never act on a stored index.** A named range is not anchored to text, so its
  span drifts if somebody edits across its boundary. Before accepting or rejecting,
  re-read the range and confirm the content inside it still matches what gdoc
  wrote. Deleting on a remembered index would eat somebody's paragraph.
- **Accepting clears the styling**, leaving text indistinguishable from text gdoc
  never touched. Rejecting deletes the range's content. A proposed deletion is the
  inverse: accepting deletes, rejecting removes the strikethrough.
- **`gdoc release` refuses to export a PDF while any proposal is outstanding.**
  Coloured text in a regulator's copy is the one failure this design can produce,
  so it is made impossible rather than unlikely.
- **The document shows what changed. The terminal says why.** gdoc cannot attach a
  comment to its own edit, so the explanation lives in `gdoc proposals` output,
  not in the file.

### The one thing that changes this decision

Developer Preview access to `writeMode: SUGGEST`. Then gdoc makes real suggestions,
with real accept and reject chips, and all of the above is deleted: no colours, no
named ranges, no stored-index rule, no release gate. This decision is scaffolding
for an allowlist that has not arrived.

### Measured 2026-08-29, after the decision above

The mechanism was tested against a live document before being written down. What
was learned changes three of the rules.

**Named ranges track edits exactly.** Inserting 200 characters before a range moved
it 232 to 432 with the covered text byte-identical; deleting 49 moved it back. Zero
drift. Typing inside expands it, typing at the start shifts it without swallowing
the new text, typing at the end leaves it alone, and a paragraph style change does
not disturb it. So the range is a real anchor, not a hopeful one.

**Drift is lossy, never destructive.** An edit *across* a boundary makes the range
**shrink to a subset** of the proposed words. It never slides onto text gdoc did
not write. So the re-read-and-compare rule catches every case, and the worst
outcome is a proposal gdoc declines to act on. It happened unprompted during the
test run, so it is not theoretical.

**The key is `namedRangeId`, not the name.** Duplicate names are allowed and
coexist, and `deleteNamedRange` **by name deletes all of them at once**. Store the
id.

**Clearing is clean.** `{"textStyle": {}, "fields": "foregroundColor,underline"}`
returns a run to `{}`, byte-identical to one that was never styled. No explicit
black is needed, and accepting leaves no residue to accumulate over months.

**`insertText` inherits the preceding run's style.** Two inserts in the test run
silently became green strikethrough, one of them a heading. **Every insert needs a
style reset after it**, naming every field the proposal styling can set.

**A named range does not survive a docx round trip**, though it survives
`files.copy`. So proposals must be resolved before any export-based step, and
`gdoc release` refusing while proposals are outstanding covers that too.

**The release check is cheap.** `documents.get(fields="namedRanges")` returns 982
bytes against 12,907 for the whole document. One call, filter names by the
`gdoc:proposal` prefix.

**Rejected alternatives, measured rather than reasoned.** An invisible range with
an explanatory comment and no styling **prints as ordinary prose**: the proposed
sentence sits in the PDF with nothing marking it. That is a failure, not a
trade-off. `quotedFileContent` is settable but Google silently coerces its mimeType
from `text/plain` to `text/html`.

A paragraph `borderLeft` **does** render as a margin bar, confirmed in a PDF, at
3PT with padding. Kept in reserve for a wholly rewritten paragraph.

## 2026-08-29. A comment is a prompt. `align` is a separate question.

Domain choice, no principle above it. Nail's model, recorded after he corrected
mine.

**Writing `ai!` in a document does exactly what typing the same instruction to
Claude in a terminal does.** Same agent, same judgement, same effect on the hub,
in the same moment. The comment thread is one more place the agent is spoken to,
not a queue feeding a later command. If the instruction needs a note in the hub
changed or a new file written, that happens then, not at some reconciliation step.

This is why `pending.md` is gone and nothing replaces it. v1 captured work in a
file because a session ended before the work was done. Nothing is captured now
because nothing waits.

**`align` is not the thing that processes comments.** It answers a different
question, asked when Nail wants it answered: *how far has this document drifted
from what the hub now says, and where?* It runs because he ran it.

It exists because the two sides move independently and nothing should keep them in
step on its own. He edits notes in the hub. Decisions land in other files in the
same folder. People edit the document. "How aligned is this with what we now
think" is a judgement about meaning, so it lives in the skill. The binary supplies
both sides and the differences between them, and says nothing about what they mean.

## 2026-08-29. The marker is the trigger. Always.

Serves principle 3.

A thread stays open once it has been addressed, so the agent keeps the
conversation in mind. **But gdoc acts only on a comment carrying an `ai` marker.**
An unmarked follow-up is reported and never executed.

Both halves matter. v1's failure was silence: a second comment in an answered
thread was ignored with nothing said, and an instruction was lost. Reporting fixes
that. The other half fixes a problem v1 never had: without it, anyone replying in
a thread Nail opened would inherit the authority to drive the agent, and
"agreed, but check the custody wording" would be executed as an instruction
against a regulatory document nobody asked to change.

So stickiness carries **context**, never **authority**.

## 2026-08-29. gdoc judges whether to ask, rather than always asking.

Domain choice. Nail's instruction, and it corrects an earlier decision here.

The earlier version said the binary never prompts and Claude puts every choice to
Nail. The first half stands. The second was wrong: it made asking the default,
and a tool that asks permission for everything teaches its user to say yes without
reading.

Instead, each action is weighed for **safety and sensitivity**. Routine and
reversible, gdoc does it. Outward-facing, hard to undo, or touching something
sensitive, it stops and asks. The weighing belongs in the skill and the prompt,
because this is not only a command-line tool: the binary, the skill and the prompt
are one product, and judgement is the part only the agent can supply.

The binary still never prompts. It has no terminal to prompt from and must run
headless.

## 2026-08-29. There is no `release` command, and gdoc does not export PDFs.

Domain choice. Nail's call.

He downloads a PDF from the browser, which he already does. So `gdoc release` is
gone, and with it the rule that refused to export while a finishing checklist or an
unresolved proposal was still in the document. That rule protected the release
command and had nothing else to protect.

The command surface is `publish` and `restyle`, plus the primitives the skills
compose: read the comments, reply, propose, apply, show the differences. The two
named workflows, a review session and an alignment check, are skills rather than
commands, because both are judgement.

## 2026-08-29. Anchors are not banked. *Rejected the day it was proposed.*

It worked. It is not being built.

The idea: a comment anchor cannot be minted, but a document born from a file gdoc
wrote can carry as many as gdoc likes. So publish would seed one anchor per
sentence, `files.copy` would shed the placeholder comments while keeping every
anchor, and gdoc could then attach genuine anchored comments in place for the life
of that document. Measured and confirmed: 1,000 anchors imported intact for 1.8
seconds of publish time, surviving edits, copies and a full round trip.

**Nail rejected it.** Seeding an invisible marker on every sentence of every
document is a large, permanent, load-bearing trick bought for a capability that
only ever works on text gdoc itself wrote. A sentence he types in the browser
could never have one. The complexity is paid on every document forever; the
benefit is partial by construction.

So: **gdoc cannot attach a comment to a specific sentence, and will not pretend
to.** The colour marking stands as written, and the explanation stays in the
terminal rather than in the margin. If the Developer Preview arrives,
`insertComment` does this properly and none of the above is needed.

The measurements are kept in `BLOCKED-BY-API.md` so nobody rediscovers the idea
and proposes it again.


## 2026-08-29. The Developer Preview arrived, and a third of this file retires.

The application went in on the morning of 2026-08-29 and Google granted it the
same day. Cloud project `4326046141`. Measured within the hour, with the ordinary
OAuth token:

| | |
|---|---|
| `writeControl.writeMode: SUGGEST` | **200, and genuine suggestions.** Text absent from `PREVIEW_WITHOUT_SUGGESTIONS`, present with `suggestedInsertionIds` under `SUGGESTIONS_INLINE`, `w:ins` in the export |
| `insertComment` | **200, and genuinely anchored.** The export enclosed exactly the word targeted |
| `commentsViewMode` | 200, comments returned with real character ranges |
| `acceptSuggestion` | 200 |

The `insertComment` payload is undocumented. It is:

```json
{"insertComment": {"range": {"startIndex": N, "endIndex": M}, "content": "..."}}
```

Not `anchor`, not `quotedRange`, not `comment`. Every other shape is
`Cannot find field`.

The answer is undocumented too, and it was measured on 2026-09-07, on a
throwaway document in the test folder, because the first live write test
guessed it wrong and lost the comment id. The `insertComment` reply carries a
`commentThread`, and the id is one level down:

```json
{"replies": [{}, {}, {"insertComment": {"commentThread": {
   "commentId": "AAAC…", "anchorId": "kix.…",
   "headPost": {"postId": "AAAC…", "content": "🤖 …", "author": {"me": true}, "…": "…"},
   "status": "OPEN", "plainTextQuote": "the inserted words"}}}],
 "suggestionResponses": [{"createdSuggestionIds": ["suggest.…"]},
                         {"updatedSummarySuggestionIds": ["suggest.…"]},
                         {"updatedSummarySuggestionIds": ["suggest.…"]}],
 "commentUpdateState": "ALL_SAVED", "writeControl": {"requiredRevisionId": "…"}}
```

`go/internal/propose/testdata/batch-saved-measured.json` is that answer with
the ids replaced. `propose` reads `commentThread.commentId` first and the flat
`insertComment.commentId` it had guessed as a fallback.

The day's behaviour is worth recording, because it explains three contradictory
readings: that morning `SUGGEST` returned **200 and silently made a direct edit**;
by midday it returned **400 Unsupported WriteControl mode**; by evening it worked.
An enrolment landing in stages. Nothing in the response ever said which state it
was in, which is "never trust a success" earning its place twice over.

### What this deletes

Colour marking of proposals. The named ranges that tracked them. The
never-act-on-a-stored-index rule. Explaining changes in the terminal instead of
the document. Anchor banking, already rejected. The Commenter-role idea. The
Chrome extension question. All of it existed to work around a gate that is open.

### What it does not touch

**Layout.** Re-probed with the preview active: `insertTableOfContents`,
`refreshTableOfContents`, `createPositionedObject`, `insertAutoText` and
`insertPageNumber` are all `Cannot find field`; `createHeader` still takes only
`DEFAULT`; `firstPageHeaderId` is still `Unallowed field`. So the finishing
checklist survives, cut to three items: the first-page header with the logo, the
contents list, the footer page numbers. All free when a document is born from the
template, all impossible on one that already exists.

### The cost nobody expected: attribution

**Authorship cannot be set.** Every field refused: `writeControl.author`,
`writeControl.suggestionAuthor`, a top-level `author`, `insertComment.author`,
`insertComment.authorDisplayName`. Everything gdoc writes is authored **Nail
Khusnullin**, in the UI and in the export.

The retired docx route *did* carry a custom `w:author`. So on attribution the
preview is **worse** than the mechanism it replaces. It is still the right choice,
because that route replaced the body and destroyed the review, but the cost is
real and it is paid every time.

The only remedy is the text itself: **a comment gdoc writes opens with `gdoc:`**,
and a suggestion is explained by a comment that does. Nail's call, and the only
option available.

*Superseded the same day: the marker is 🤖, not a `gdoc:` prefix. See
"Everything gdoc writes opens with 🤖."*

## 2026-08-29. gdoc proposes. It never accepts and never rejects.

Nail's rule. `acceptSuggestion` and `rejectSuggestion` work and will not be used.
Resolving a proposal is his hand, in the document, and is not delegated.

*Narrowed 2026-09-07, Nail's call: `rejectSuggestion` is used on gdoc's own
proposals, and on nothing else, because it is the only request that withdraws a
replace proposal whole. The guard carries it for one granted id per run. See
"A replace proposal cannot be fully withdrawn" below.*

This raised a question and answered it. An earlier decision says the hub markdown
changes when a proposal is **accepted**. If gdoc never accepts, how does it know?

**By reading.** An accepted suggestion leaves the pending list and its text becomes
ordinary text; a rejected one leaves and the text does not. So `align` reads the
document, sees which proposals were resolved and how, and offers the accepted ones
to the markdown. gdoc learns the decision rather than making it.

Measured 2026-08-29 that this is safe to build on: twelve pending suggestions,
twelve distinct ids, and resolving three left the other nine byte-identical and
still pending. Ids are stable, so gdoc can list what is outstanding, quote what
each proposes, and notice what changed since it last looked.

It also survives the awkward case: if a suggestion is **edited** before being
accepted, its id tells gdoc nothing, but the document's text does, and that is what
`align` reconciles from anyway.

## 2026-08-29. MCP does not replace the REST client. The binary stays a REST client.

Assessed against Google's own documentation rather than guessed at.

The Docs MCP server is Google-hosted at `https://docsmcp.googleapis.com/mcp/v1`,
itself in Developer Preview, and exposes **exactly two tools**: `read_doc` and
`update_doc`. `update_doc` is a pass-through to `documents.batchUpdate`.

Four reasons it cannot be gdoc's transport:

- **No Drive.** gdoc creates documents, copies the template, exports, reads
  folders and checks write access. None of that exists in two tools. There is a
  separate Drive MCP server, also in preview, which is one more dependency rather
  than a simplification.
- **It is built for AI hosts, not for a CLI.** The documented redirect URIs are
  `claude.ai` and `antigravity.google`. A Go binary speaking MCP over HTTP with
  OAuth is *more* work than calling REST, not less.
- **The guard could not exist.** gdoc's safety property is that every request
  passes through a transport that refuses any file it was not given. Talking to
  Google's server directly leaves nothing to wrap and no way to bound what is
  reached. That is principle 3, not a preference.
- **It is preview too.** No stability is gained.

Where MCP is genuinely useful is outside gdoc: Nail can connect Claude Code to it
for ad-hoc work on documents gdoc has nothing to do with. His enrolment also lit up
the Drive, Gmail, Calendar, Sheets, Slides, Chat, People and Universal Search MCP
servers. None changes this design.

## 2026-08-29. Tabs are out of scope, and gdoc defends against them.

Amended after review: the earlier title read as an intention to support tabs.
There is none. **gdoc has no tab features.** It creates none, styles none, and
addresses none, and this does not change if Google pushes tabs harder.

The entry stays because tabs are **GA**, not preview, so a colleague can put one
in a document gdoc is pointed at, and one of Nail's own test documents already
had one. Both failure modes are silent, on somebody else's document.

Google's wording, verbatim:

> When `includeTabsContent` is not set, "the text fields in the Document Resource
> (e.g. `document.body`) will be populated with content **from the first tab
> only**. The `document.tabs` field will be empty and content from other tabs won't
> be returned."

And for writes: a request without a `tabId` "will in most cases be applied to the
first tab" — **except** `ReplaceAllTextRequest`, `DeleteNamedRangeRequest` and
`ReplaceNamedRangeContentRequest`, which **apply to every tab**.

Two defensive rules, and they are defences rather than features:

- **Always read with `includeTabsContent=true`.** Reading `document.body` without
  it sees one tab and reports the rest as absent, silently. gdoc must never be
  blind to content that exists. `commentsViewMode` requires it anyway.
- **If a document gdoc is asked to write to has more than one tab, gdoc stops and
  reports that tabs are unsupported.** It never guesses which tab was meant and
  never writes anyway. Not knowing must not resolve to "the first one, probably".

## 2026-08-29. Amendments from reading the documentation.

Recorded because these were found by **reading**, after several were missed by
probing. Nail's instruction, and he was right.

**`insertComment` takes an assignee.** The documented shape is:

```json
"insertComment": {
  "content": "...",
  "assigneeEmailAddress": "user@example.com",
  "range": {"startIndex": N, "endIndex": M}
}
```

Probing found `range` and `content` and would never have found
`assigneeEmailAddress`. It means gdoc can raise a question **and assign it to the
person who should answer it**, which is a better shape than a comment nobody owns.

**`addCommentReply` is nested, which is why the probe failed.**

```json
"addCommentReply": {
  "commentId": "...",
  "post": {
    "content": "...",
    "commentAction": "RESOLVE" | "REOPEN",
    "assigneeEmail": "..."
  }
}
```

Note `commentAction`. **gdoc can now resolve and reopen threads.** v1's rule that
it must never resolve a comment was enforced by an API that had no way to. It is
now a choice, and it stays: **resolving means the answer was accepted, and only
Nail accepts.** Same reasoning as never accepting a suggestion.

**Also documented, and confirming the probes:** authorship is not settable
anywhere; `commentsViewMode: COMMENTS_VIEW_MODE_INCLUDED` requires
`includeTabsContent=true`; and comment or suggestion writes can **partially fail**,
committing the text while losing the comment, reported in `commentUpdateState`
as `NO_UPDATES_REQUESTED`, `ALL_SAVED` or `ALL_FAILED_UNKNOWN_REASON`. **Check that
field, not the status code.**

### The rule this whole entry exists to state

**Read the documentation before probing.** Probing finds what you thought to ask
for. It found `range` and `content` and missed the assignee, missed the nesting on
replies, and would have missed `commentUpdateState` entirely, which is a silent
data-loss path. Probing is for confirming what the documentation claims, and for
the cases where the documentation is wrong, which today it was three times.

## 2026-08-29. gdoc can retract its own proposal, and must check that it landed.

Two findings from the first end-to-end run of the whole design, on a copy of the
ideal test document.

**A suggestion can be withdrawn.** A `deleteContentRange` issued in `SUGGEST` mode
over gdoc's own pending insertion removes it cleanly and comes back with
`deletedSuggestionIds`. So a proposal gdoc got wrong can be taken back without
touching the document and without asking Nail to reject it. This matters more than
it sounds: it is the difference between a tool that can correct itself and one whose
mistakes are the reader's problem.

It does not weaken the rule that gdoc never accepts or rejects. Withdrawing its own
unaccepted proposal is not resolving Nail's decision; there was no decision yet.

**An insertion point is off by one until proved otherwise.** The first attempt of
that run put text one character before a full stop, producing
`…this line Keep one idea in each item..`, and `batchUpdate` returned **200**. The
error was visible only in the rendered PDF. Index arithmetic is the hazard of this
whole API, not the requests.

So: after inserting a suggestion, **read back the span and confirm it reads as
intended** before telling anyone it worked. The same "never trust a success" rule,
now with a third instance behind it.

### What the run proved, which is the whole design in one document

Style applied in place, and every one of ten features survived: person chip, date
chip with its timestamp **and** locale **and** format, calendar rich link, the
Google Drawing, three footnotes, both lists, the table with its pinned shaded
header, the inline image byte-identical, a hand-made anchored comment still
enclosing exactly its original words, and a hand-made pending suggestion still
pending.

The date chip is the sharpest evidence. Every docx route flattened it to the string
"13 Aug 2026" with no date behind it. In place, untouched.

The source document was verified byte-identical, same revisionId, before and after.
Only the copy was written to, per Nail's rule.

## 2026-08-29. The house style is a config file. The .docx becomes provenance.

Nail's decision, and the reason he gave for it is the whole argument.

`house.yaml` is the house style: 1,110 lines, 30,866 bytes, including the logo as
base64. It holds page geometry, all nine named styles, the cover, the header and
footer, three tables cell by cell, the contents field, heading numbering and the
logo. A generator reads it plus markdown and emits a .docx. **Nothing reads the
master at runtime.**

Measured 2026-08-29 against a document built the old way, by copying the template:
160 items compared, **137 identical**. Of the 23 differences, 21 are the same value
stated two ways, because the template declares a heading colour and then overrides
it on every paragraph while the config states the effective one. Two are real: a
0.001pt rounding on the logo offset, and a page count caused by a stale contents
list. On the rendered PDFs the **worst position offset anywhere is 2.5pt**.

The logo comes out **positioned, not inline**: a `wp:anchor` in the header part
behind `<w:titlePg/>` survives Google's import as a real floating object at the
template's exact offset. The Docs API cannot create one; the docx can, so nothing
needs the API to.

### Why not the hybrid, which was the earlier recommendation

Because v1 already ran that experiment and we know how it ends.

v1's template was a .docx. The style improvements Nail made afterwards, table
changes and paragraph spacing among them, **could not go into a Word file**, so
they went into code. That is what `render/shell.py` is: 574 lines, roughly forty
accumulated decisions, three to five weeks to reproduce. **The code became the real
template and the .docx stopped describing the output.**

The hybrid rebuilds that trap the first time an improvement touches a table, which
is exactly where Nail's changes went.

### The reason that actually decided it

Nail does not make these changes by hand. In his words: *"I didn't do the changes
in the template by myself because I just didn't know what exactly I need to
improve. I asked AI to analyse and understand better and give me advice."*

If the house style improves by **an agent reading it, reasoning about it and
proposing a change**, then it has to be something an agent can read and change, and
a change has to be something Nail can review. `space_after_pt: 6 → 8` is a line in
a diff he can judge. The same change buried in a surgery function is not. An agent
cannot meaningfully edit a .docx; it writes code that pokes at the XML, which is
v1 again.

### The risk, and the thing that answers it

The house look lives in details nobody would think to write down: which blank
paragraph is 18pt and which 29pt, that two of three tables have zero cell padding
while the third has 5pt, that Word insets a table by its first cell's left margin.
Each was invisible until measured.

So **the 160-item comparison stays as a test.** It renders from the config, renders
from the master, and fails when any value drifts. That test is what makes this
decision safe rather than brave.

The master `.docx` stays in the repo as provenance and as that test's fixture.
Nothing else reads it.

## 2026-08-29. The finishing checklist is for `restyle` only.

Clarified because it was easy to misread, and Nail did.

The Developer Preview solved comments and suggestions. **It did not touch layout.**
Re-probed with the preview active: no request creates a table of contents, a
first-page header or a page-number field, and `createPositionedObject` does not
exist.

But those three are **free on a document gdoc creates**, because they come from the
.docx it writes: a `wp:anchor` behind `<w:titlePg/>` gives a positioned logo in a
first-page header, a Word TOC field imports as a live refreshable contents list,
and a PAGE field gives footer numbers. All three verified.

| | born from gdoc's docx | already exists |
|---|---|---|
| first-page header with positioned logo | **free** | no |
| live contents list | **free** | no |
| footer page numbers | **free** | no |

So `publish` needs no checklist at all. **Only `restyle` does**, when a colleague's
document arrives and gdoc styles it in place rather than replacing it, which is the
one case where the document was not born from gdoc's own file.

One caveat that misled once already: a document that happens to already carry an
empty first-page header segment **can** have a logo written into it with
`insertInlineImage` at that `segmentId`. That is not a general capability. It works
only where the segment already exists, and gdoc cannot create one.

## 2026-08-29. `publish` builds a file. `restyle` has two modes, and defaults to the careful one.

Nail's reading, confirmed against everything measured today.

### `publish` renders a .docx and uploads it

Not a hundred API calls. The document does not exist yet, so there is nothing to
protect, and the file gives what the API refuses:

- a **positioned logo** in a real first-page header, from a `wp:anchor` behind
  `<w:titlePg/>`
- a **live, refreshable contents list**, from a Word TOC field
- **footer page numbers**, from a PAGE field

All three verified. There is no reason to build a new document any other way.

### `restyle` has two modes and they are a genuine trade

| | **in place** (default) | **as a new document** (`--new`) |
|---|---|---|
| how | `batchUpdate` on the original | render its content to a .docx, upload as new |
| house style | approximate | **exact** |
| comment threads, with real authors and anchors | **kept** | lost |
| pending suggestions | **kept** | lost |
| **smart chips** | **kept, with their data** | **flattened** |
| Google Drawings | kept | kept |
| the URL | **unchanged** | new link, old one still live |

The trade is sharper than "layout against review". Going to a new document costs
the **chips** as well, because the content has to pass through a file to get there,
and that is where a date chip becomes the string "13 Aug 2026" with no date behind
it.

### In place is the default

`restyle` exists for a colleague's document that already has comments on it. That
is exactly when in-place is worth most and a new document costs most. The three
things in-place cannot do are three clicks on the finishing checklist, once, on a
document Nail will have open anyway.

### The exception gdoc can detect

A fresh draft with **no comments, no suggestions and no smart chips** has nothing
to protect. Rebuilding it costs nothing and gives the exact house style.

So gdoc checks before it styles, and when there is nothing to lose it **says so and
offers the better route**. It does not take it silently: a new document means a new
URL, and that is the kind of thing a person should choose. Uncertainty resolves to
asking, and certainty about a better option resolves to offering it.

## 2026-08-29. A comment is answered with the hub open, not against the document alone.

Nail's emphasis, added because the earlier entry understated it.

"A comment is a prompt" says how the instruction arrives. This says what the agent
has in its hands when it acts on one.

**The same context a terminal session would have.** The markdown the document was
generated from, and the folder around it: the notes, the decision log, the meeting
records, the raw material. The hub exists to be the context an agent reasons over,
and a comment in a Google Doc is simply another place Nail speaks to that agent.

So:

- An `ai?` question is answered **from what he actually knows**, not from the words
  on the page. "Why did we take this position" is answered out of the decision log,
  not paraphrased back out of the paragraph it was asked about.
- An `ai!` instruction is carried out **against his material**, not against the
  document in isolation.
- If carrying it out needs a note in the hub changed or a new file written, that
  happens in the same moment, exactly as it would in a terminal session.

The rule underneath: **there is no reduced mode.** Working through comments is not
a lesser kind of session with only the document loaded. It is the ordinary agent,
with the ordinary context, reached through a different door.

## 2026-08-29. Emoji reactions: still no API, preview or not.

Re-probed after the Developer Preview was granted, because the earlier finding
predated it and should not have been assumed to hold.

`insertReaction`, `addReaction`, `createReaction`, `insertEmoji`,
`addEmojiReaction`, `createEmojiReaction`, `insertEmojiReaction`, `reactToRange`
and `addTextReaction` all return `Cannot find field`. The word **`reaction`
appears zero times** in the entire Docs discovery document. The only `emoji` in the
API is `iconEmoji`, which sets the icon on a document **tab** and has nothing to do
with reacting to text.

Emoji as ordinary characters work everywhere, in the body and inside a comment,
because they are just Unicode. So gdoc can write `gdoc: ✅ applied`. It cannot
attach a reaction to a span.

Nothing in the design wanted this. Recorded so it is not probed a third time.

## 2026-08-29. The colour scheme is retired, not forgotten.

Marking proposals in green and explaining them in the terminal was decided earlier
on **the same day**, and it is now withdrawn. It existed for one reason: at the
time gdoc could not create a suggestion or attach a comment to text, so it had to
signal a change some other way.

The Developer Preview removed that constraint. So:

- **no green, no red, no strikethrough** as a marking convention
- **no named ranges** tracking gdoc's own proposals, and no stored-index rule
- **no explanation stranded in a terminal** where the people reading the document
  cannot see it

Changes arrive as **native Google suggestions**, with the accept and reject chips
colleagues already understand, and the reason arrives as a **real comment anchored
to the exact words**.

Recorded as a withdrawal rather than deleted, because the reasoning was sound for
the constraint that existed, and someone reading this in six months should be able
to tell a retired workaround from an oversight. If the preview is ever withdrawn,
this is the fallback and it is written down.

## 2026-08-29. Alignment runs both ways, and it is not a command.

Nail's correction. The earlier entry said `align` asks how far the document has
drifted from the hub. That is half of it.

**Both sides move, and the second direction is the dangerous one.**

- The **document** drifts when somebody edits it in the browser. Visible, and the
  kind of drift anyone would think to look for.
- The **hub** moves ahead when a decision lands in the folder, a meeting note
  changes a position, or the source markdown is edited in another session. The
  document is then quietly **wrong** while looking untouched, and nobody is
  prompted to check.

So alignment reports in both directions, and it exists for the next iteration
rather than as a tidying step.

### It is not a command-line operation

The binary can say what each side contains and where the two differ. It cannot say
whether a difference **matters**, and it must not pretend to: "this document no
longer reflects the position we took in the decision log" is a judgement about
meaning.

So alignment lives in the **skill**. The binary supplies both sides and the
differences. This is the same split as everywhere else in the design: facts below,
judgement above.

### And it never deletes

Something present only in the hub is **work not yet published**, not drift to be
removed. Alignment offers what the document has that the notes lack, reports what
the notes have that the document lacks, and reports a conflict where both moved.
It writes to the markdown only with agreement, and never removes from it.

## 2026-08-29. The product is three things, and it matters which does what.

Recorded because it is the shape everything else hangs on, and it was implicit
rather than written.

- **The binary** deals in facts. What is in the document, what is in the hub, where
  they differ, what was changed. It never decides what any of it means. It never
  prompts. It emits structured output and exits.
- **The skill** is the judgement. What a comment is asking for. Whether a
  difference matters. What to propose. Whether an action is safe enough to take
  without asking.
- **A comment in the document** is another door into that same agent, alongside the
  terminal. Not a lesser mode, not a queue: the same agent, the same context, the
  same behaviour.

This is why the binary is opaque and why it never prompts, and it is why "a comment
is a prompt" is true rather than a slogan. It is also the answer whenever something
new is proposed: **facts belong below, judgement belongs above**, and anything that
needs to weigh meaning does not go in the binary.

## 2026-08-29. Everything gdoc writes opens with 🤖.

Nail chose it from six candidates seen side by side in a real comment sidebar:
🤖 robot, ✨ sparkle, 🪄 wand, ✍️ pencil, a plain `gdoc:` prefix, and the robot with
the name spelled out.

**The rule: every comment gdoc leaves begins with the robot, and nothing else.** No
name, no colon, no prefix text.

### Why a marker is needed at all

Authorship cannot be set. Measured 2026-08-29: `writeControl.author`,
`writeControl.suggestionAuthor`, a top-level `author`, `insertComment.author` and
`insertComment.authorDisplayName` are all `Cannot find field`. Every comment and
every suggestion gdoc makes is signed **Nail Khusnullin**, with his photo, in the
UI and in the export.

So the text is the only place the truth can go.

### Why the robot rather than the others

The marker has **one** job, and it is narrower than "this is AI". The comment
already carries Nail's name and face. The job is to say **this was not written by a
person**.

- 🤖 says exactly that and nothing more.
- ✨ says "AI-assisted", which is a softer and different claim, and it is Google's
  own Gemini mark, so it would read as a Google feature rather than as Nail's tool.
- A `gdoc:` prefix is more text in a column already full of text. It does not carry
  at a glance.
- The robot plus the name was the safe option, and was rejected as noise: the name
  repeats in every thread and adds nothing once a reader has learned the symbol.

Emoji survive the API as ordinary Unicode, in the document body and inside a
comment, both verified.

## 2026-08-29. gdoc reads back what it proposed, before saying it worked.

Kept, narrowed, and settled by Nail: "I don't care at all how many calls you need."
Google's write quota is 60 a minute per user and nothing in this design comes near
it, so the extra round trip is free in practice.

**The rule: after gdoc makes a proposal, it reads the document back and confirms
the change arrived as a suggestion and not as an edit, before reporting success.**

This is narrower than the "never trust a success" version it replaces. Checking
`commentUpdateState` rather than the HTTP status, and similar care elsewhere, is
ordinary correct code and does not need a decision record. This one does, because
it is a **choice that costs something** and a reasonable implementer would skip it:
the API returns 200 and gives no hint that anything is wrong.

The reason it is not optional, measured on the morning of 2026-08-29: that exact
call returned **200 and silently rewrote a reviewed document** instead of proposing
anything. Without the read-back gdoc would have reported a proposal while having
edited somebody's document, and nobody would have known until they read it.

The detector is cheap and unambiguous: a genuine suggestion is **absent** from the
document read with `PREVIEW_WITHOUT_SUGGESTIONS`; a direct edit is present.

The four incidents that produced this rule are recorded as facts about Google's
behaviour in `BLOCKED-BY-API.md`, not here.

## 2026-08-29. The preview may vanish, and the risk is accepted. Loudly.

Nail's call, on review. If Google withdraws the Developer Preview, gdoc loses
proposing in the document: suggestions and anchored comments. Replies in existing
threads survive, because they are plain Drive API. That loss is accepted.

What is not built: fallback code. The colour scheme stays on paper, in the
withdrawal entry above, and is rebuilt only if the day comes. Maintaining a live
second path for a retired design is the cost that was refused.

What is not negotiable: the loss must be loud. The capability probe against a
throwaway document and the read-back after every proposal stay, because an
unenrolled project answers 200 while silently direct-editing. The feature may
vanish. It must never vanish as a silent edit to a reviewed document.

## 2026-08-29. The guard owns the transport, and every id carries a write level.

Serves principle 3, and carries the v1 guard into Go stronger than it was.

In v1 the guard wrapped a client library's transport. v2 has no client library,
and the whole preview surface is hand-rolled JSON, which is exactly the kind of
request that walks around a bolted-on wrapper. So the guard is not a wrapper any
more: **one package owns the network**, nothing else in the binary can make an
HTTP request, and a test fails if HTTP construction appears anywhere else.

The set keeps v1's two doors exactly: ids handed in on the command line, ids
learned from a create the guard itself carried. New in v2, each id carries a
write level:

- an id learned from a create is fully writable, because gdoc made it
- an id handed in is **read-and-suggest only**, never direct-editable
- `restyle` in place is the one exception, granted explicitly per run, never
  inherited

A rule the skill follows is advisory. A rule in the transport is physics. The
morning incident, a 200 that direct-edited a reviewed document, becomes locally
impossible to cause and the read-back catches the remote half.

## 2026-08-29. Publish runs once. Everything after travels as suggestions.

Colleagues keep one URL for the life of a document, and gdoc never replaces an
existing body. Together those close the question of how a big hub update lands:
there is **no republish mechanism**, in any form.

Scale is handled by judgement, in the skill, like everything else:

- small drift arrives as inline suggestions
- a structural rewrite arrives as **one suggestion per affected section**, each
  with a 🤖 comment saying what changed and why, so accepting is one chip per
  section rather than a wall of green
- when a document's review has run its course, the escape hatch already exists:
  `restyle --new` builds a fresh document from the hub, exact style, new URL,
  said out loud and chosen by a person

One more line from the same review: the prompts behind the skills, alignment
above all, are part of the product. They live in the repo and a change to one is
reviewed like code, because a wrong alignment judgement writes a wrong suggestion
into a document colleagues read.

## 2026-08-29. Live review is a session, not a service.

Nail's requirement, added after the spec was first written: once a review is
launched, the conversation moves into the document and stays there. He writes
`ai!` in a comment and the suggestion and the 🤖 reply arrive within seconds,
for as long as the session runs. Review is live until he stops it.

What does not change: nothing runs on its own. The live loop exists only while
a session Nail launched is running. Closing the terminal ends it. The 2026-08-17
standing watcher stays dead.

**The mechanism is session-scoped polling.** The skill loops; every 5 to 15
seconds the binary makes one cheap call, "any comment activity since this
cursor", and exits. The binary stays one-shot and gains a `--since` cursor.
Liveness is the skill's, which is where judgement already lives.

Two alternatives were reviewed on Nail's ask, and both lose mechanically:

- **Apps Script** cannot push to a machine with no public endpoint, so the poll
  never goes away; it would only add a component in front of it. And its own
  timers bottom out at one minute, six times slower than the poll it would
  replace. Plus the costs already priced in BLOCKED-BY-API.md: a second
  execution surface that sees less of the API than the binary does.
- **A Chrome extension** can push for real, via native messaging, but only
  while the document is open in a browser, only until Google moves the DOM, and
  only by putting a signed-in Chrome inside the trust boundary of a tool whose
  first principle is one file you copy. Driving the browser was already
  rejected once, for the checklist, on the same grounds.

**A colleague's `ai!` acts.** Nail's call, made knowing what it widens: during
a live session, anyone who can comment on the document can drive the agent,
without Nail between them. The marker stays the trigger and identity stays not
a gate, consistent with everything above. The guard caps the blast radius: on a
handed-in document the agent can only suggest and reply, so the worst a
colleague's instruction can produce is a suggestion Nail rejects. A colleague
who can already edit the whole document was always trusted with more than this.

## 2026-09-07. A replace proposal cannot be fully withdrawn by a delete. Decided: reject gdoc's own.

Found by the first live write test, on throwaway documents in the test folder,
after M3's revmux review had passed. The 2026-08-29 finding that a
`deleteContentRange` in `SUGGEST` mode "removes it cleanly and comes back with
`deletedSuggestionIds`" was measured on a **pure insertion**. A `propose` is a
**replace**: one suggestion id covering a suggested deletion of the quoted words
and a suggested insertion of the replacement. Five things were measured:

| Sent, on gdoc's own replace proposal | What Docs did |
|---|---|
| `deleteContentRange` in `SUGGEST` over the insertion half (what `withdraw` sends) | 200. The inserted words are gone. The answer carries `suggestionResponses[].updatedSummarySuggestionIds: [id]` and **no `deletedSuggestionIds`**. The read-back still carries the quoted words as a suggested deletion under the same id. **Half the proposal stays pending.** |
| a second `deleteContentRange` in `SUGGEST` over the still-suggested-deleted words | 200, `updatedSummarySuggestionIds` again, and nothing changed. |
| one `deleteContentRange` in `SUGGEST` over both halves at once | 200. The inserted words become suggested-inserted **and** suggested-deleted under the same id. Worse than before. |
| `rejectSuggestion {suggestionId}`, in `SUGGEST` mode and plain | **Refused by the guard before it left the machine**: any request kind whose name carries "suggestion" is refused, which is the 2026-08-29 rule "gdoc never accepts and never rejects" in code. Not measured against Google. |
| the 🤖 comment, after the insertion half is deleted | Survives, `deleted: false`, still anchored by id, quoting words that are no longer in the document. The skill can still reply into it. |

So today `withdraw` reports `verified: false` with two warnings on every
replace proposal, leaves the entry in the note, and the document keeps a
suggested deletion gdoc cannot take back. The live write test fails on exactly
this, and it is left failing on purpose.

The one route that would retract the whole suggestion is `rejectSuggestion` on
gdoc's own id, and the guard refuses it by Nail's rule. Widening that rule to
"gdoc may reject a suggestion the note records as its own" is his decision, not
a refactor: the guard cannot tell whose a suggestion is, so the door would have
to be a per-run grant the `withdraw` command seeds from `proposals[]`, the way
`AllowCreateIn` names one folder. The alternatives are to leave `withdraw` as a
half-retraction that says so, or to make withdrawing a thing Nail does by hand.

**Decided the same day. Nail: "allow reject on own ids; merge after the fix."**
Measured before it was built, through the new door and not around the guard:
`rejectSuggestion {suggestionId}` inside a `writeMode: SUGGEST` batch returns
200, the document reads exactly as it did before the proposal, the answer
carries `suggestionResponses[].rejectedSuggestionIds: [id]` with
`commentUpdateState: ALL_SAVED`, and the 🤖 comment survives, `deleted: false`,
still anchored by id. The same request without `writeMode` behaves the same, so
the guard's SUGGEST requirement on a handed-in document stays as it was.

What changed: `Policy.AllowReject(id)` is the per-run grant, seeded by
`cmdWithdraw` from the note's `proposals[]` after `Mine` has answered; the guard
carries a `rejectSuggestion` only when it is spelled exactly, carries exactly
`{"suggestionId": <that id>}`, and the id is the granted one, while
`acceptSuggestion`, `deleteSuggestion`, any other id and any second field stay
refused; `withdraw` sends that one request, checks that the document still
carries the id on either side before it writes, and is verified when
`rejectedSuggestionIds` names the id and a fresh read carries no run under it.
A document the old delete half-retracted is repaired by running `withdraw`
again, because a run still suggested-deleted under the id counts as pending.
The multi-tab refusal went with the range. The live write test passes end to
end. The rule "gdoc never accepts, rejects or deletes anyone else's suggestion"
is unchanged and is now guard-enforced rather than a family ban.

## 2026-09-08. v1's `gdoc: <id>` stays refused. Publish is the only writer of the block.

v1's `gdoc generate` writes the pairing as a plain string, `gdoc: 1AbC...`. v2's
front matter is a block, `gdoc: {schema: 1, document_id, folder_id, ...}`, read
under `yaml.Strict()`. So until M6 a note v1 published could not be handed to
v2's `propose --md` or `withdraw`: the reader refused the shape, and the run had
no provenance record, which is the permission to withdraw later. Reading,
answering and proposing on the link alone always worked; only the note pairing
was lost.

Two ways out were on the table. The reader could accept the v1 string as a
schema-0 shape and upgrade it on the first write, which keeps notes published
that week usable with no hand edit. Or M6's publish becomes the only writer of
the block and v1's `generate` is retired for v2 notes.

**Decided: the second.** No schema-0 shape enters the reader. A note v1
published gets its `gdoc:` line rewritten by hand once, or the line is taken out
and the note is published again by v2.

The reason is what the first option costs for what it buys. It buys one hand
edit on the handful of notes published in one week. It costs a second shape in
the one reader that decides whether gdoc is allowed to act on a note, for ever,
plus an upgrade-on-write path that rewrites somebody's front matter from a shape
gdoc only half understands. `frontmatter`'s whole rule is that a block gdoc half
understands is a pairing it may act on wrongly, and a schema-0 branch is that
rule with an exception in it. There is also no folder in a v1 string, so an
upgraded block would carry a `document_id` and no `folder_id`, which is a
pairing that reads and is not the one publish writes.

What that puts on the refusal is the whole migration story, because there is no
other one. So it is not the parser's own sentence. A `gdoc:` key holding a plain
string comes back naming the shape it found, the id inside it, and the two ways
out: write it as a block carrying `schema: 1` and that `document_id`, or take
the line out and publish again. `v1Pairing` and `v1Refusal` in
`internal/frontmatter/frontmatter.go` are where that lives, and they are asked
only after the strict read has already refused, so a block that reads pays
nothing. A key holding some other scalar, `gdoc: 3`, is not v1's pairing and
does not get v1's sentence: sending somebody to rewrite a number as a document
id is the wrong way.

`docs/backlog/v1-frontmatter-migration.md` is deleted by this entry. It asked
the question and said Nail decides in M6.
