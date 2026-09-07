# gdoc v2 Milestone 3: Writing, and the first usable review

## Overview

The Go binary learns to write, and every write it makes inside a Google Doc is a suggestion. After this milestone four commands exist beside the M2 readers: `gdoc probe` asks Google, on a throwaway document, whether suggestions are honoured today; `gdoc reply` posts a 🤖 reply into a thread; `gdoc propose` writes a change as a native suggestion with an anchored 🤖 comment explaining it; `gdoc withdraw` retracts one of gdoc's own pending proposals. Each write is verified by a second route before the command reports it: a read-back, and for a proposal the docx export as well.

Then the first piece of judgement: the **review skill**, rewritten for v2. It reads the threads, decides which ones still need an answer, answers `ai?` from the hub, carries out `ai!` in the hub, and proposes document changes as suggestions. **v2 becomes daily-usable here**, which is why this milestone sits before the generator.

The line M2 drew holds. The binary reports facts and writes exactly what it is told; the skill decides what to write. The binary refuses what SPEC.md's Never list refuses, at the guard where it can, and the skill carries the rules the guard cannot see.

Decisions Nail took on 2026-09-07, before this plan: every document change is a suggestion (hub markdown files are plain edits, a file has no suggesting mode); the skill answers `ai?` and carries out `ai!` without asking, and proposes without asking, because a suggestion is reversible and gdoc can withdraw its own; `skills/gdoc-review` is replaced rather than duplicated; the probe document is created fresh in the test folder on every `propose` run and trashed after; the project **is enrolled** (measured that morning, `BLOCKED-BY-API.md`), and the code still treats a silent direct edit as possible; the five revmux findings on the M1 guard are fixed first; a long action writes at most an acknowledgment and a receipt into a thread; and `gdoc2` was a temporary name for testing, so this milestone makes `gdoc` on PATH the Go binary everywhere.

Spec: `docs/v2/SPEC.md` ("Verification: never trust a success", `reply`, `propose`, "Withdrawing a proposal", "Skills", "Never"). Master plan: `docs/v2/PLAN.md`, section M3. Predecessor: `docs/plans/completed/2026-09-06-gdoc-v2-m2-reading.md`.

## Context (from discovery)

- Files and components involved: new packages `go/internal/probe`, `go/internal/reply`, `go/internal/propose`, `go/internal/withdraw`; modified `go/internal/gapi` (a POST), `go/internal/guard` (the five findings, and `commentWrites` narrowed), `go/internal/frontmatter` (the `proposals[]` shape gains `quoted`), `go/cmd/gdoc` (four commands), `go/internal/live` (an opt-in write test), `skills/gdoc-review/SKILL.md` (rewritten), `go/boundary` (no change expected; `gapi` is already the room that builds requests).
- Related patterns found: `gdoc/reply.py` (`assert_plain_text`: a reply carrying markdown is refused, because Docs threads render it literally; port it), `gdoc/marker.py` (v1's `[gdoc]` label; v2's is the 🤖 prefix, SPEC), `docs/v2/spikes/probes/gate.py` (the measured SUGGEST insert and `SUGGESTIONS_INLINE` read the probe reproduces), `docs/v2/DECISIONS.md` 2026-08-29 (`insertComment` takes `range`, `content`, `assigneeEmailAddress`; `addCommentReply` is nested under `post`; `commentUpdateState` is `NO_UPDATES_REQUESTED`, `ALL_SAVED` or `ALL_FAILED_UNKNOWN_REASON`; a `deleteContentRange` in SUGGEST mode over gdoc's own insertion answers `deletedSuggestionIds`; an insertion point is off by one until proved otherwise).
- What M2 already gives this milestone: `docs.Fetch` with suggestion ids and comment ranges, `comments.Threads`, `docx.Export` and `docx.Match`, `frontmatter.Read`/`Write` with `proposals[]`, `gapi.Session` for GETs, `guard` carrying `POST {id}/comments/{cid}/replies` at `LevelSuggest`, `POST {id}:batchUpdate` on a handed-in id only when the body says `writeMode: SUGGEST`, and refusing any request kind whose name carries "suggestion".
- Measured this morning (2026-09-07): on a throwaway document, `insertText` at index 30 in SUGGEST mode returned 200 and the read-back carried `suggest.xyh4cb4emh7y`, so the project is enrolled. The same insert landed inside a word, which is the index hazard `propose` is built around.
- Dependencies identified: none new. Standard library plus `goccy/go-yaml`.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own commit.
- Make small, focused changes.
- **CRITICAL: every task MUST include new or updated tests** for the code it changes. Success and failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test` runs `-race`; keep it green.
- **CRITICAL: update this plan file when scope changes during implementation.**
- **CRITICAL: facts only in Go.** A write does what it was told and reports what it saw. Whether to write, and what, is the skill's. The grep from M2 (`handled`, `accepted`, `rejected`, `matters`, `drift`) runs again in Task 10, joined by `should` and `decide`.
- **CRITICAL: never trust a success.** Every write in this milestone is followed by a read through a different route before the envelope says `ok`. A write that cannot be verified is reported as unverified, never as done.
- No em dashes in any text this plan produces, code comments, commit messages and the skill included.

## Testing Strategy

- **Unit tests**: required for every task. Writers are tested over a fake `http.RoundTripper` handed to `guard.NewClient`, with fixtures for the answers Google gives, so the unit suite never reaches the network.
- **Guard tests**: the five findings each get the test that would have caught them, and the tests fail before the fix and pass after.
- **Boundary test**: unchanged. `internal/gapi` already builds requests; the new packages take a session interface, as `docs` and `comments` do.
- **Live test**: `go/internal/live` gains an opt-in write test behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`. It creates a document in the Drive test folder `1w0SresizE9Kr810VZRJwX4JtDBF4OqNr`, proposes a change into it, replies to the comment the proposal made, withdraws the proposal, and trashes the document, asserting every read-back on the way. It writes only to documents it created. The unattended run does not set the variables.
- Coverage standard: every exported function under `go/internal/` has a test. `atomicfile` and `config`, below the bar since M2, are topped up in Task 10.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build`

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

A write flows the same way a read did: the command opens a `guard.Policy` with exactly the ids it was given, builds a `gapi.Session`, and hands it to a writer package that makes one request, reads the answer, and then reads again through another route. `reply` goes to Drive's `replies.create` and reads the thread back. `propose` runs the probe, then for each proposal reads the document fresh, finds the quoted text, sends one `batchUpdate` in SUGGEST mode (a `deleteContentRange`, an `insertText`, an `insertComment` anchored on the new span), checks `commentUpdateState`, reads back with `SUGGESTIONS_INLINE` and with `PREVIEW_WITHOUT_SUGGESTIONS`, and finally exports the docx to prove the comment is attached. `withdraw` reads the document fresh to find the suggested span, sends one `deleteContentRange` in SUGGEST mode, and requires `deletedSuggestionIds` to name the id and the read-back to show it gone.

Key design decisions and why:

- **The probe creates and trashes a document on every `propose` run.** SPEC: the binary is one-shot and holds no hidden state, so there is nothing to cache the answer in, and an asserted "already probed" from outside reopens the exact failure the probe prevents. The probe is `AllowCreateIn`'s first production caller, one milestone earlier than PLAN.md guessed.
- **A proposal names text, never an index.** DECISIONS.md: "never act on a stored index". The skill hands over the exact words to replace; `propose` finds them in a fresh read and refuses when they occur more than once. An index computed a minute ago is the hazard the whole API has.
- **One `batchUpdate` per proposal, three requests inside it.** Docs applies the requests in order and the indexes inside one batch are consistent, so the comment lands on the span the insert made. Two proposals are two batches, each after its own fresh read, because a proposal shifts the ground under the next.
- **`reply` uses Drive, `propose`'s comment uses Docs.** `replies.create` is documented, carried by the guard at `LevelSuggest`, and its answer is the reply itself, which the read-back confirms. `insertComment` is the only way to anchor a comment to a span, and its answer carries `commentUpdateState`, which SPEC says to check instead of the status code.
- **Withdraw is guarded by provenance.** The binary retracts a suggestion only when the id is in the note's `proposals[]`, written by `propose`. SPEC: gdoc never deletes anyone else's suggestion, and the front matter is the only place that memory may live.
- **The guard gets narrower before it gets a caller.** The five revmux findings are fixed first, and `PATCH`/`DELETE` on comments and replies leave `commentWrites`: no M3 call needs them, and the guard cannot tell whose comment it is. M7 or M8 adds them back beside a caller, the way `GrantInPlace` will return.
- **`gdoc` on PATH becomes v2 in this milestone, and `gdoc2` goes.** `gdoc2` was Nail's temporary name for testing v2 beside v1 (2026-09-07). Both v1 skills call the venv binary by its full path (`$HOME/.config/gdoc-agent/venv/bin/gdoc`), not `gdoc` on PATH, so repointing `~/.local/bin/gdoc` at the Go binary breaks neither of them. `install.sh` does the repointing (Task 7), the skill calls `gdoc` (Task 8), and CLAUDE.md's M2 note about `gdoc2` is corrected (Task 11). The install story proper, one file copied to a machine with nothing else on it, is still M9.

## Technical Details

**Principles.** Serves: 3 (every write goes through the guard with the ids it was given, and is verified by a second route), 4 (the thread carries at most an acknowledgment and a receipt; the terminal carries the record), 2 (provenance lives in the note's front matter and nowhere else). Strains: the accepted external risk in SPEC.md: proposing depends on Google's Developer Preview, which can be withdrawn, and the loss must be loud. The probe is that loudness.

**Tech stack.** Go 1.27, standard library plus `goccy/go-yaml`. The skill is markdown under `skills/`, symlinked as v1's are.

**Global constraints:**

- Every command: exactly one JSON object on stdout, exit 0 iff `ok`. The binary never prompts.
- The guard is opened with exactly the ids a command was given: the document, and for `propose` the probe folder as the create target. Nothing else.
- Every Drive call carries `supportsAllDrives=true` where the method defines it (`files.create`, `files.update`, `files.get`); `comments.list`, `replies.create` and the export do not define it, and M2 measured that.
- Every write to a document is `writeMode: SUGGEST`. The guard refuses a `batchUpdate` on a handed-in id without it, and that stays.
- Every `batchUpdate` answer has its `commentUpdateState` read when it carried a comment; `ALL_SAVED` is success and anything else is reported as the partial failure it is.
- A write target with more than one tab stops the command: `multi_tab` is reported and nothing is sent.
- Every reply and every comment gdoc writes opens with `🤖 ` and nothing before it. The binary refuses a body that does not, and refuses a body carrying markdown (v1's rule, ported).
- Commit messages: `feat(v2): ...`, `fix(v2): ...`, `test(v2): ...`, `docs: ...`. All commits from the repo root.

**The probe.** In the folder it was given: `files.create` a Docs file named `gdoc probe <RFC3339>`; `batchUpdate` (direct, the file is gdoc's) inserting one sentence; `batchUpdate` in SUGGEST mode inserting one word inside it; `documents.get?suggestionsViewMode=SUGGESTIONS_INLINE`; `enrolled` is true iff a text run carries a `suggestedInsertionIds` entry; then `files.update {trashed: true}` and `files.get?fields=trashed` to confirm. Every failure path still runs the trash, and the report carries `probe_document_id` and `trashed` so a document left behind is named, never silent. A probe that finds the word as plain text reports `enrolled: false` and `propose` sends nothing.

**Finding the span.** `propose` reads the document (`docs.Fetch`), concatenates each paragraph's runs, and searches every paragraph of the one tab for the quoted text. Exactly one match is required: none is `quoted text not found`, more than one is `quoted text occurs N times; quote more of the sentence`. The match gives `start` and `end` in the tab's indexes. Text inside a pending suggestion is not matched: a proposal on top of a proposal is refused.

**The proposal batch.** One `batchUpdate` with `writeControl: {writeMode: SUGGEST}` and three requests in this order: `deleteContentRange {range: {startIndex: start, endIndex: end}}`, `insertText {location: {index: start}, text: replacement}`, `insertComment {range: {startIndex: start, endIndex: start + len(replacement)}, comment: {content: "🤖 " + why, assigneeEmailAddress?}}`. Index lengths are UTF-16 code units, which is what the Docs API counts; a test covers a replacement with a non-BMP character. The answer's `writeControl.requiredRevisionId` and `commentUpdateState` are read.

**Read-back, three routes.** After the batch: `documents.get` with `SUGGESTIONS_INLINE` must show a run at `start` whose `suggestedInsertionIds` is non-empty and whose text is the replacement, and a run carrying `suggestedDeletionIds` with the quoted text; `documents.get` with `PREVIEW_WITHOUT_SUGGESTIONS` must still show the quoted text at `start`, so the change is a suggestion and not an edit; the docx export must carry a comment whose text is the 🤖 body with `Anchored: true`. The report says which of the three held. All three is `verified: true`. Fewer is `ok: true` with `verified: false` and a warning naming the route that failed, because the write happened and hiding it would be the worse outcome; the skill reads `verified` and decides what to tell Nail.

**Withdraw.** Given a suggestion id: the id must be in the note's `proposals[]` (or the command is refused: "not one of gdoc's own proposals in this note"); `documents.get` with `SUGGESTIONS_INLINE` finds the runs carrying that id in `suggestedInsertionIds`; one `batchUpdate` in SUGGEST mode with `deleteContentRange` over that span; the answer's `deletedSuggestionIds` must contain the id; the read-back must show no run carrying it. Then the entry leaves `proposals[]` and the note is rewritten through `frontmatter.Write`. The 🤖 comment the proposal made is left in place: deleting a comment is a write the guard no longer carries, and the skill can reply to it saying the proposal was withdrawn.

**The front matter, schema 1, `proposals[]` shape.** `{id, comment_id, at, quoted}`: the suggestion id (the first of the insertion ids), the comment id `insertComment` returned, the time, and the quoted text that was replaced. `quoted` is new in this milestone and optional in the decoder, so an M2 note still reads.

**Output shapes.** Each command's `data`:

```jsonc
// gdoc probe --folder <folder-url-or-id>
{ "enrolled": true, "probe_document_id": "1Ab...", "trashed": true, "suggestion_ids": ["suggest.abc"] }

// gdoc reply <url> <comment_id> --body-file reply.txt
{ "document_id": "...", "comment_id": "AAAA", "reply_id": "AAAB", "created": "2026-09-07T10:00:00Z", "verified": true }

// gdoc propose <url> --from proposals.json --folder <probe-folder> [--md note.md]
{
  "document_id": "...", "tabs": 1, "multi_tab": false,
  "probe": { "enrolled": true, "probe_document_id": "...", "trashed": true },
  "proposals": [{
    "quoted": "reviewed annually", "replacement": "reviewed every six months",
    "suggestion_ids": ["suggest.abc"], "comment_id": "AAAC", "comment_update_state": "ALL_SAVED",
    "verified": true, "checks": { "suggestions_inline": true, "preview_without_suggestions": true, "docx_anchored": true }
  }],
  "files_changed": ["policy.md"]   // only with --md
}

// gdoc withdraw <url> <suggestion_id> --md note.md
{ "document_id": "...", "suggestion_id": "suggest.abc", "deleted_suggestion_ids": ["suggest.abc"], "verified": true, "files_changed": ["policy.md"] }
```

`proposals.json` is a list of `{quoted, replacement, why, assignee?}`. `why` becomes the comment body after `🤖 `. The skill writes this file; a person can too.

Warnings ride in `warnings`: the policy's, the session's, a read-back route that did not hold, a probe document that could not be trashed.

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): everything achievable inside this repo. The packages, their tests, the commands, the skill, the documentation.
- **Post-Completion** (no checkboxes): the live write run, and the first real review session on one of Nail's documents.

## Implementation Steps

---

### Task 1: The guard, narrower before it gets a caller

**Files:**
- Modify: `go/internal/guard/policy.go`, `go/internal/guard/policy_test.go`
- Modify: `go/internal/guard/params.go`, `go/internal/guard/params_test.go`
- Modify: `go/internal/guard/transport.go`, `go/internal/guard/transport_test.go`
- Modify: `CLAUDE.md` (the `writeMode` caveat gains the 2026-09-07 measurement)

**Interfaces:**
- Consumes: the five findings from the revmux review of 2026-09-06, recorded in the completed M2 plan.
- Produces, one per finding:
  1. `isSuggestMode` reads the body into `map[string]json.RawMessage`, finds `writeControl` by exact key (refusing when two keys fold to it), then `writeMode` the same way one level down, and permits SUGGEST only when both are spelled exactly and once. Google's proto-JSON is case-sensitive; the guard must not be broader than the server on the one field that permits a write.
  2. `hasDuplicateKeys(body []byte) error` walks the body with a `json.Decoder` token stream and refuses any object that repeats a key, folding case. Called at the top of `judgeRequests`, `checkCommentWrite` and `checkParent`. A repeated key collapses to last-wins inside `encoding/json`, so a body the guard judged on one copy could be acted on by the server on the other.
  3. `commentWrites` drops `PATCH` and `DELETE` from `comment` and `reply`: the guard cannot tell whose comment C1 is, and no M3 call needs either. The refusal names the rule and the milestone that may add them back beside a caller.
  4. `uploadShapes` drops `resumable`: the `PUT` leg and `upload_id` are refused elsewhere, so the shape could never complete, and a half-permitted shape reads as a working route. `multipart` stays refused in `checkParent` as before.
  5. A request body type in the tests that records `Close`, and one test per refusal path (unreadable body, wire mismatch, policy refusal, parent-check refusal) asserting the body was closed, so the `RoundTripper` contract the comment claims is held by a test.
- ⚠️ Finding 6 from the same review, the `docsReadParams` comment, was fixed in M2.

- [x] write the failing tests, one per finding: `{"WriteControl":{"WRITEMODE":"SUGGEST"}}` and a doubled `writeControl` are refused as direct edits; a doubled `requests` key, a doubled `action` key and a doubled `parents` key are each refused by name; `PATCH` and `DELETE` on `{id}/comments/C1` and on a reply are refused with the rule in the message; `uploadType=resumable` is refused; each refusal path closes the body
- [x] run `cd go && go test ./internal/guard/` and watch the new tests fail
- [x] implement the five changes; update the `isSuggestMode` doc comment and the CLAUDE.md caveat with the 2026-09-07 measurement, keeping the rule that the probe and the read-back are what make a write trustworthy
- [x] run the guard tests, the boundary test, gofmt, vet: green
- [x] commit: `fix(v2): the guard reads writeMode exactly, refuses repeated keys, and carries no comment PATCH or DELETE`

---

### Task 2: `gapi.PostJSON`

**Files:**
- Modify: `go/internal/gapi/session.go`, `go/internal/gapi/session_test.go`

**Interfaces:**
- Produces: `(*Session).PostJSON(ctx, rawURL string, body any, into any) error`: marshals `body`, sends `POST` with `Content-Type: application/json` and the bearer, through the same expired-token and one-401 refresh policy `get` has, decodes a 2xx answer into `into`. The request is built with `GetBody` set, so the guard's peek reads the same bytes the wire sends and a retry after a 401 resends them. `into == nil` discards the answer. The package doc comment loses "Everything here is a GET".

- [ ] write the failing tests over a fake wire: the body arrives as sent with the JSON content type; a 401 refreshes once and the retried request carries the same body; a 400 with a Google error body surfaces `message` and the status; a guard refusal (a `batchUpdate` on a handed-in id without SUGGEST) comes back as the refusal and the fake saw nothing
- [ ] run the tests and watch them fail
- [ ] implement `PostJSON` by extracting the refresh loop from `get` into one function both call
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): gapi posts JSON through the guard with the same refresh policy`

---

### Task 3: `internal/probe`: does Google honour a suggestion today?

**Files:**
- Create: `go/internal/probe/probe.go`, `go/internal/probe/probe_test.go`
- Create: `go/internal/probe/testdata/*.json` (a `SUGGESTIONS_INLINE` answer with the word carrying an insertion id; the same answer with the word as plain text; a `files.get` answer with `trashed: true`)

**Interfaces:**
- Consumes: a session interface with `GetJSON` and `PostJSON`; `docs.Parse` for the read-back.
- Produces: `type Report struct { Enrolled bool; ProbeDocumentID string; Trashed bool; SuggestionIDs []string }` and `probe.Run(ctx, s Session, folderID string) (Report, error)`: the sequence under "The probe" in Technical Details. The create names exactly `folderID` as its parent, which is how the guard's `AllowCreateIn` lets it through and learns the id at `LevelFull`. Trash runs on every path after the create succeeded, including an error mid-way, and its outcome is in the report. An error after the create carries the report too, so the document id is never lost.
- The policy for a probe is `AllowCreateIn(folderID)` and nothing else: the probe document is learned, not handed in.

- [ ] write the failing tests: the enrolled fixture gives `Enrolled: true` with the id; the plain-text fixture gives `Enrolled: false`; a failing SUGGEST write still trashes and reports `Trashed`; a create whose answer has no id is an error naming the folder; the requests the fake saw are, in order, create, direct insert, SUGGEST insert, read, trash, get; the direct insert has no `writeControl` and the SUGGEST insert has exactly `{"writeMode":"SUGGEST"}`
- [ ] run the tests and watch them fail
- [ ] implement
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): the capability probe, a throwaway document that says whether SUGGEST is honoured`

---

### Task 4: `internal/reply`: one 🤖 reply, checked

**Files:**
- Create: `go/internal/reply/reply.go`, `go/internal/reply/reply_test.go`

**Interfaces:**
- Consumes: the session interface; `comments.Fetch` for the read-back.
- Produces:
  - `reply.Check(body string) error`: refuses an empty body, a body not opening with `🤖 ` (exactly the robot and one space, nothing before), and a body carrying markdown, with v1's pattern ported from `gdoc/reply.py` and its tests (`**`, backticks, a `#` heading at a line start, a fenced block, a link in brackets).
  - `reply.Post(ctx, s Session, docID, commentID, body string) (Result, error)` with `type Result struct { ReplyID, Created string; Verified bool }`: `POST /drive/v3/files/{id}/comments/{cid}/replies?fields=id,createdTime,content` with `{content: body}`, then `comments.Fetch` of the one thread and `Verified` true iff a reply with that id and that content is in it. The body sent is exactly the body checked; nothing is appended (v1 appended `[gdoc]`; v2's mark is the prefix the skill wrote).
- The two-message rule (an acknowledgment and a receipt at most) is the skill's: the binary posts one reply per call and knows nothing about the previous one.

- [ ] write the failing tests: `Check` accepts `🤖 The 2026 register.` and refuses each bad shape by name; `Post` sends the body verbatim to the replies URL with only `fields` in the query; the read-back marks `Verified` true when the reply is in the thread and false when it is not, with the reply id still reported; a guard refusal on a document not in the policy reaches the caller
- [ ] run the tests and watch them fail
- [ ] implement
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): reply posts one 🤖 reply through Drive and reads the thread back`

---

### Task 5: `internal/propose`: a change as a suggestion, verified three ways

**Files:**
- Create: `go/internal/propose/propose.go` (the batch and its answer)
- Create: `go/internal/propose/span.go` (finding the quoted text)
- Create: `go/internal/propose/verify.go` (the three read-backs)
- Create: `go/internal/propose/propose_test.go`, `go/internal/propose/span_test.go`, `go/internal/propose/verify_test.go`
- Create: `go/internal/propose/testdata/*.json` (a document before; the same document after with the suggestion runs; the `PREVIEW_WITHOUT_SUGGESTIONS` view; a `batchUpdate` answer with `ALL_SAVED` and a comment id; one with `ALL_FAILED_UNKNOWN_REASON`)
- Modify: `go/internal/frontmatter/schema.go`, `go/internal/frontmatter/schema_test.go` (`Proposal` gains `Quoted string`, optional)

**Interfaces:**
- Consumes: the session interface; `docs.Fetch`/`docs.Parse`; `docx.Export`/`docx.Parse`; `frontmatter`.
- Produces:
  - `type Proposal struct { Quoted, Replacement, Why, Assignee string }`, read from `proposals.json`.
  - `propose.FindSpan(d *docs.Document, quoted string) (docs.Range, error)`: exactly one match across the one tab's paragraphs, none inside a pending suggestion; the two refusals named under "Finding the span". Indexes in UTF-16 code units.
  - `propose.Batch(r docs.Range, p Proposal) []byte`: the three-request body, `writeControl` exactly `{"writeMode":"SUGGEST"}`.
  - `propose.Apply(ctx, s Session, docID string, p Proposal) (Result, error)`: fresh read, `FindSpan`, `PostJSON` the batch, read `commentUpdateState` and the comment id, then `Verify`.
  - `propose.Verify(ctx, s Session, docID string, r docs.Range, p Proposal, commentBody string) Checks` with `type Checks struct { SuggestionsInline, PreviewWithoutSuggestions, DocxAnchored bool }` and the suggestion ids it found.
  - `type Result struct { Quoted, Replacement string; SuggestionIDs []string; CommentID, CommentUpdateState string; Verified bool; Checks Checks }`.
  - `propose.Record(note []byte, results []Result, at time.Time) ([]byte, error)`: appends to `proposals[]` through `frontmatter.Write`.
- A document with `MultiTab()` is refused before any write, naming the tab count.

- [ ] write the failing tests for `FindSpan`: one match gives the range in UTF-16 units (a fixture paragraph with an emoji before the match proves it); no match and two matches refuse by name; a match inside a suggested run refuses
- [ ] write the failing tests for `Batch`: three requests in order, the comment content opens with `🤖 `, the assignee is present only when given, `writeControl` is exact
- [ ] write the failing tests for `Apply` and `Verify` over a fake wire that answers the read-backs from fixtures: the happy path is `Verified: true` with all three checks; an `ALL_FAILED_UNKNOWN_REASON` answer is `Verified: false` with the state reported; a preview that shows the replacement (a direct edit happened) fails the second check and warns; a docx without the comment fails the third; a two-tab document is refused before the fake sees a POST
- [ ] write the failing schema test: `proposals[]` with and without `quoted` both read
- [ ] run the tests and watch them fail
- [ ] implement
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): propose writes a suggestion with an anchored 🤖 comment and verifies it three ways`

---

### Task 6: `internal/withdraw`: gdoc takes back its own proposal

**Files:**
- Create: `go/internal/withdraw/withdraw.go`, `go/internal/withdraw/withdraw_test.go`
- Create: `go/internal/withdraw/testdata/*.json` (a document with the suggestion; the `batchUpdate` answer carrying `deletedSuggestionIds`; the document after)

**Interfaces:**
- Consumes: the session interface; `docs`; `frontmatter`.
- Produces: `withdraw.Run(ctx, s Session, docID, suggestionID string, note *frontmatter.Block) (Result, error)` with `type Result struct { SuggestionID string; DeletedSuggestionIDs []string; Verified bool }`: refuses an id not in `note.Proposals`; finds the runs carrying the id; one SUGGEST `deleteContentRange`; `Verified` iff the answer's `deletedSuggestionIds` contains the id and the read-back carries no run with it. `withdraw.Forget(note *frontmatter.Block, suggestionID string) *frontmatter.Block` returns a new block without the entry.

- [ ] write the failing tests: an id not in the note is refused before any request; the happy path deletes the right span and is `Verified`; an answer without `deletedSuggestionIds` is `Verified: false` with a warning; `Forget` leaves the other proposals in place and does not mutate its input
- [ ] run the tests and watch them fail
- [ ] implement
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): withdraw retracts one of gdoc's own suggestions and proves it is gone`

---

### Task 7: The four commands on the envelope

**Files:**
- Modify: `go/cmd/gdoc/main.go` (dispatch, `usage`)
- Create: `go/cmd/gdoc/write.go`, `go/cmd/gdoc/write_test.go`
- Modify: `go/cmd/gdoc/main_test.go`
- Modify: `install.sh` (`~/.local/bin/gdoc` links to the built Go binary; the `gdoc2` link goes)

**Interfaces:**
- Produces: `gdoc probe --folder <id-or-url>`, `gdoc reply <url> <comment_id> --body-file <path>`, `gdoc propose <url> --from <proposals.json> --folder <probe-folder> [--md <note>]`, `gdoc withdraw <url> <suggestion_id> --md <note>`, each printing the `data` shape under Technical Details with `warnings` from the session and the command.
- `propose`: probe first; `enrolled: false` stops the run with every proposal reported `not sent` and `ok: false`; otherwise proposals are applied one at a time, each after its own fresh read, and the run stops at the first one that cannot be sent (span not found, multi-tab) with the earlier results reported. `--md` records provenance only for proposals whose batch was accepted (`commentUpdateState` read), verified or not, so a proposal that landed is never forgotten; the note's `document_id` must match the document.
- `withdraw` requires `--md`: provenance is the permission.
- Argument parsing is strict, as before. The folder argument accepts a Drive folder URL or a bare id (the folder half of v1's `docid.py`, added to `documentID`'s file).
- `usage` becomes `Commands: auth status, auth login, read, comments, suggestions, probe, reply, propose, withdraw`.

- [ ] write the failing command tests over the stubbed session and fixtures: each command prints exactly one JSON object; `probe` reports the fixture's verdict; `reply` with a markdown body fails before any request and names the offending text; `propose` with a not-enrolled probe sends nothing and fails naming it; `propose --md` records provenance for an accepted-but-unverified proposal and leaves the rest of the note byte-identical; `propose --md` on a note whose `document_id` differs fails before any write; `withdraw` without `--md` fails naming it; the unknown-command error lists the nine commands
- [ ] run the tests and watch them fail
- [ ] implement `write.go`, wire `dispatch`
- [ ] run the full suite with `-race`, gofmt, vet, `make build`: green
- [ ] update `install.sh`: after the v1 venv step it keeps, run `make build` and link `~/.local/bin/gdoc` to `bin/gdoc` (replacing the link to the venv binary), remove a `~/.local/bin/gdoc2` link when one exists, and print which binary `gdoc` now is; it stays safe to re-run. v1's two skills are untouched: they name the venv binary by its full path
- [ ] run `./install.sh` on this machine and confirm `gdoc auth status` prints v2's envelope and `gdoc2` is gone from PATH
- [ ] commit: `feat(v2): probe, reply, propose and withdraw on the envelope; gdoc on PATH is v2`

---

### Task 8: The review skill, rewritten for v2

**Files:**
- Rewrite: `skills/gdoc-review/SKILL.md`
- Modify: `install.sh` only if the symlink target changes (it should not: the skill directory keeps its name)

**Interfaces:**
- Consumes: `gdoc comments`, `gdoc read`, `gdoc reply`, `gdoc propose`, `gdoc withdraw`, and the hub folder the session runs in.
- Produces: the skill Nail invokes with a document URL. It carries the judgement M2 kept out of Go, and it keeps the rules of v1's skill that are about the world rather than about v1's storage:
  - **Setup**: `GDOC=gdoc`, `ROOT=$PWD` (the hub). Nothing runs git. `gdoc auth status` before guessing at a credential failure.
  - **Read**: `gdoc comments <url>` for the threads, `gdoc read <url>` for the text with `[[c:ID]]` anchors. The skill decides which threads still need an answer by reading the replies: a thread whose last turn is a 🤖 reply answering the marked comment is done; a marked reply written after gdoc's last one is new work; an unmarked follow-up is reported and not acted on. No `handled` field exists, on purpose.
  - **Markers**: `ai?` is answered in the thread; `ai!` is carried out in the hub (edit the note, the decision log, whatever it names) and receipted in the thread; `ai:` the skill classifies as one of the two and says which. Anyone's marker counts; the marker is the trigger, identity is not a gate. An unmarked comment is never acted on unless Nail asks for all-comments mode, which keeps v1's per-comment picking.
  - **Proposing**: when the right answer is a change to the document's text, the skill writes `proposals.json` (exact quoted words, the replacement, the reason in reader language) and runs `gdoc propose ... --folder 1w0SresizE9Kr810VZRJwX4JtDBF4OqNr --md <paired note>` when the document is paired, without asking first. It reads `verified` and `checks` and tells Nail in the terminal which proposals did not verify.
  - **Two messages at most** for a long action: one acknowledgment when the work will take noticeably long, one receipt when it verifiably landed, never a progress feed; a failure replaces the receipt with a plain sentence naming the reason.
  - **Stop and ask** rules carried from v1 verbatim: a draft that carries something out of the hub the document should not (the 18 August counterparty figure), a note the root marks confidential, an answer the skill is not confident is true, a genuinely ambiguous comment, a second reply to an answered thread.
  - **Report**: what was posted, proposed and edited, then the full text of every reply and every comment body, exactly as sent, because nothing else shows Nail what is now visible to everyone on the document.
  - **Never**: the SPEC list, in the skill's words: never a direct edit, never resolve or reopen, never accept, reject or delete anyone's suggestion, never delete from the hub, never git, never markdown in a thread, never a reply without the 🤖 prefix, never trust a status code.
  - Removed from v1's skill: `capture`, `pending.md`, `edits`, `baseline`, `--terminal-only` (replaced by "say what you would post" when Nail asks for a dry run), the service-account branch.
- The skill is a product artifact: a change to it is reviewed like code, and the revmux round reads it.

- [ ] write the skill, section by section as above, plain English, short sentences, no em dashes
- [ ] read it against SPEC.md "Skills", "How a comment reaches the agent" and "Never", and list in this plan any rule the skill states that SPEC does not, or the reverse
- [ ] check `install.sh` still links the directory and that `~/.claude/skills/gdoc-review` resolves to it
- [ ] commit: `feat(v2): the review skill, rewritten over gdoc comments, read, reply and propose`

---

### Task 9: The live write test

**Files:**
- Modify: `go/internal/live/live_test.go` (a second test), `go/internal/live/testdata/` stays gitignored

**Interfaces:**
- Produces: `TestLiveProposeReplyWithdraw`, gated on `GDOC_LIVE_TEST=1` and `GDOC_LIVE_WRITE=1`, skipping with a message otherwise. It opens a policy with `AllowCreateIn` on the test folder, runs `probe.Run` (which creates and trashes its own document), creates a second document there with one sentence, proposes a replacement into it and asserts `Verified` with all three checks, replies `🤖 ...` to the comment the proposal made and asserts `Verified`, withdraws the proposal and asserts `Verified`, then trashes the document and asserts `trashed: true`. Every document it touches is one it created. With `GDOC_LIVE_RECORD=1` it saves the `batchUpdate` answer and the `SUGGESTIONS_INLINE` read as redacted fixtures for `propose`'s tests to be tightened against.

- [ ] write the test, run it with the variables unset and confirm it skips with a message
- [ ] run the full suite: green; do not run it live in the unattended run
- [ ] commit: `test(v2): the opt-in live write: probe, propose, reply, withdraw, trash`

---

### Task 10: Verify acceptance criteria

**Files:**
- Modify: `go/internal/atomicfile/atomicfile_test.go`, `go/internal/config/config_test.go` (the two coverage stragglers reach 80%)
- Modify: none other expected. Fix whatever the checks below break.

- [ ] verify SPEC.md acceptance item 4 is testable end to end: a propose lands as a genuine suggestion (absent under `PREVIEW_WITHOUT_SUGGESTIONS`), with its 🤖 comment anchored to the exact words, verified through the docx export; the unit test over fixtures proves the code path and the live test proves Google
- [ ] verify SPEC.md acceptance item 6: the probe distinguishes an enrolled project from an unenrolled one without writing to any real document; both fixtures exist and the probe never touches a handed-in id
- [ ] verify no judgement leaked into Go: grep the new packages for `handled`, `accepted`, `rejected`, `matters`, `drift`, `should`, `decide`; each hit is a fact with a different name or a defect
- [ ] verify every write is verified: grep the writers for `PostJSON` and confirm each call is followed by a read-back the tests cover
- [ ] verify the guard refuses what SPEC's Never list names, with the guard's own tests: a direct edit on a handed-in id, `action` on a comment, any `*Suggestion` request kind, `PATCH`/`DELETE` on a comment, a create outside the named folder
- [ ] bring `atomicfile` and `config` to 80% or better with tests of their error paths
- [ ] run the full suite with `-race`, gofmt, vet, `make build`, `make dist`
- [ ] verify coverage: every exported function under `go/internal/` has a test
- [ ] commit any fixes this task made

---

### Task 11: [Final] Update documentation

**Files:**
- Modify: `README.md`, `CLAUDE.md`, `docs/v2/PLAN.md`

- [ ] update `README.md`: the four write commands, `proposals.json`, the probe, what `verified` and `checks` mean, and that the review skill runs over `gdoc2`
- [ ] update `CLAUDE.md` under "v2 lives at `go/`": every document write is a suggestion and the guard holds it; the probe runs on every `propose` and creates in the test folder; the three read-backs and what `verified: false` means; provenance in `proposals[]` is the permission to withdraw; `commentWrites` carries no `PATCH`/`DELETE` until a caller arrives; the skill decides which threads need an answer and the binary does not; `gdoc` on PATH is v2 from this milestone and `gdoc2` is gone, while v1's skills keep calling the venv binary by its full path (correcting the M2 note about `gdoc2`)
- [ ] update `docs/v2/PLAN.md`: mark M3 done with the date, record what it left for M4 (the live session polls `comments --since` and the same skill acts on what arrives) and for M6 (`AllowCreateIn` now has a production caller, the probe)
- [ ] run the full test suite one more time
- [ ] commit: `docs: M3 lands, the writes, the probe and the review skill written down`
- The harness moves this plan to `docs/plans/completed/` when the run finishes.

## Post-Completion

*Items needing a real account or a person. No checkboxes.*

**Manual verification:**

- Run the live write test once, on Nail's machine: `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1 GDOC_LIVE_RECORD=1`. It creates two documents in the test folder and trashes both. Then tighten `propose`'s fixtures to the recorded answers and commit them redacted.
- Run the review skill on a **copy** of one of Nail's real documents, placed in the test folder (`files.copy` carries the body but not the comments; add two marked comments by hand, one `ai?` and one `ai!`). Watch it answer, carry out, and propose. Read the terminal record against the document.
- Then run it on a real document Nail names, with Nail watching the first time.

**Known and out of scope for this plan:**

- The live session (polling on `--since`) is M4. The skill written here is the non-live one.
- Deleting or editing a comment gdoc wrote is not carried by the guard after Task 1. A withdrawn proposal's comment stays, with a reply saying so.
- Numbered lists still print like bulleted ones in `read`; pictures and drawings are in `docs/backlog/`.

**External system updates:**

- `gdoc` on Nail's PATH becomes the Go binary and `gdoc2` disappears, through `install.sh` in Task 7. Typing `gdoc read <url>` in a terminal now prints the document text instead of v1's comment list; `gdoc comments` is the comment list. v1 is still reachable at `~/.config/gdoc-agent/venv/bin/gdoc`, which is what its two skills call. The token and the config are untouched.
