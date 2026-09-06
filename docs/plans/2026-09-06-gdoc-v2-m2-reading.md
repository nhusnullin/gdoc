# gdoc v2 Milestone 2: Reading, and the honest witness

## Overview

The Go binary learns to read, and it reads facts only. After this milestone three commands exist: `gdoc read <url>` gives the AI a Google Doc as text it can read, with pending suggestions inline and comment anchors marked; `gdoc comments <url>` lists the threads with every fact about them; `gdoc suggestions <url>` lists what is pending and what stopped being pending since the last look. Every read goes through the guard with exactly the one document it was given, and the token is refreshed when it has expired. Two libraries the write milestones depend on land here too: the docx export reader, which is the only truthful way to know whether a comment is attached to text, and the versioned `gdoc:` front-matter block, which publish (M6) must write correctly from its first version.

The line this milestone draws, and holds: the binary prints facts, the skills judge. Whether a comment is answered, whether a gone suggestion was accepted, whether a difference between the hub note and the document matters, which side is the source of truth: all of that is the AI's, reading what `read` and `comments` print. Nothing in Go decides any of it. Nail's call, 2026-09-06, recorded in SPEC.md ("Reading comments", "The diff") and PLAN.md M2.

The problem it solves: v1 is retired and every read of a reviewed document today goes through Python and pandoc. v2 has a guard and a credential and nothing that uses them. This milestone gives `Policy.AllowFile` and `Token.Refresh` their first production callers and puts `Policy.Warnings()` on the envelope, as PLAN.md M2 asks.

How it integrates: the module at `go/` grows read-only packages under `internal/`. Nothing under `gdoc/` (v1) changes. The one new room that may import `net/http` is `internal/gapi`, admitted to the boundary import allowlist the way `internal/auth` was. The first third-party module enters: `goccy/go-yaml`, with its reason already written in SPEC.md.

Spec: `docs/v2/SPEC.md` (agreed 2026-08-29, corrected 2026-09-06). Master plan: `docs/v2/PLAN.md`, section M2. Predecessor: `docs/plans/completed/2026-08-29-gdoc-v2-m1-foundation.md`. Backlog: `docs/backlog/read-pictures-and-drawings.md` (out of scope here, on purpose).

## Context (from discovery)

- Files and components involved: new packages `go/internal/frontmatter`, `go/internal/gapi`, `go/internal/docs`, `go/internal/view`, `go/internal/suggestions`, `go/internal/comments`, `go/internal/docx`; modified `go/cmd/gdoc/main.go`, `go/boundary/boundary_test.go`, `go/go.mod`, `go/internal/guard/params.go` (one comment), `go/internal/guard/policy.go` (one deletion).
- Related patterns found: `gdoc/suggestions.py` (the run-joining walk that turns `suggestedInsertionIds` and `suggestedDeletionIds` into suggestions with a heading context; port it), `gdoc/docid.py` (URL parsing; port the document half), `gdoc/comments.py` (Drive comment fields v1 reads), `docs/v2/spikes/probes/gate.py` (the live-measured Docs read: `commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED&includeTabsContent=true` answers 200 with a top-level `comments` key), `docs/v2/spikes/probes/read_suggestions.py` (`suggestionsViewMode=SUGGESTIONS_INLINE` is the only view carrying suggestion ids).
- What M1 already permits: `judgeDocs` carries `GET /v1/documents/{id}` with `docsReadParams` (`alt`, `fields`, `suggestionsViewMode`, `includeTabsContent`, `commentsViewMode`); `judgeDrive` carries `GET {id}/export` (any mime but PDF), `GET {id}/comments` with `pageSize`, `pageToken`, `includeDeleted`, `startModifiedTime`, `fields`. The `Authorization` header must be exactly one `Bearer <token>` value. No M2 request needs a guard grammar change.
- Dependencies identified: `github.com/goccy/go-yaml v1.19.2` (zero transitive modules, per PLAN.md). Nothing else. `goldmark` waits for M5: nothing here parses markdown.
- Measured facts this plan leans on (DECISIONS.md, BLOCKED-BY-API.md): reading without `includeTabsContent=true` sees the first tab only and reports the rest as absent; `commentsViewMode` requires it; the docx export is the honest witness for comment attachment (`word/comments.xml` plus `commentRangeStart`/`commentRangeEnd` in `word/document.xml`), because Drive's `anchor` and `quotedFileContent` survive detachment.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own commit.
- Make small, focused changes.
- **CRITICAL: every task MUST include new or updated tests** for the code it changes. Tests are a required deliverable of the task, not an optional extra. Success and failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** No exceptions. `make test` runs `-race`; keep it green.
- **CRITICAL: update this plan file when scope changes during implementation.**
- **CRITICAL: facts only.** If a function's answer is one two reasonable people could disagree about, it does not belong in Go. Report the inputs to that judgement instead and leave a note in the plan. This is the rule the whole milestone is cut along.
- Run the validation commands after each change.
- Immutability: functions return new values rather than mutating arguments. The one exception is `Policy`, which is already mutex-guarded state.
- No em dashes in any text this plan produces, code comments and commit messages included.

## Testing Strategy

- **Unit tests**: required for every task. Readers are pure functions over decoded JSON, bytes or text, so they are tested on fixtures under `testdata/` with no wire at all. Table-driven wherever the input is a set of cases.
- **Fake wire**: `internal/gapi` and the commands are tested over a fake `http.RoundTripper` handed to `guard.NewClient`, the way M1 tests `auth`. Nothing in the unit suite reaches the network.
- **Boundary test**: gains one import room (`internal/gapi`) and one allowed module (`github.com/goccy/go-yaml`). Both are deliberate one-line widenings with the reason beside them, and both still fail in both directions.
- **Live test**: one opt-in end-to-end test behind `GDOC_LIVE_TEST=1`, skipping otherwise. It creates a document in the Drive test folder `1w0SresizE9Kr810VZRJwX4JtDBF4OqNr` (a shared drive: every Drive call carries `supportsAllDrives=true`), writes a comment into it through the API, reads it back through `read` and `comments --witness`, and trashes the document. It never touches a document the run did not create. The unattended run does not set the variable, so this is a Post-Completion item.
- **E2E tests**: no UI, none to add.
- Coverage standard: every exported function under `go/internal/` has a test. `cd go && go test ./... -cover`; add tests rather than lowering the bar.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build` (from the repo root; the binary must still link)

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.
- Update the plan if the implementation deviates from the original scope.

## Solution Overview

Reads flow one way: a command parses its arguments, opens a `guard.Policy` with exactly the id it was given, builds the guard's client, wraps it in a `gapi.Session` that carries the bearer token and refreshes it when expired, fetches the raw JSON or bytes, and hands them to a pure reader package. The reader returns a struct; the command puts it on the envelope with the policy's warnings. Every reader is testable on a fixture file with no wire, so the fixtures are the specification of what Google actually returns.

Key design decisions and why:

- **One package builds requests.** `internal/gapi` is the only new package that names `net/http`. It sets `Authorization: Bearer` and takes the client as a parameter, so the builder allowlist does not move. Putting request building in one room mirrors v1's `fetch.py` and keeps every reader a pure function.
- **One Docs read, three views.** `documents.get` with `includeTabsContent=true`, `suggestionsViewMode=SUGGESTIONS_INLINE` and `commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED` returns the structure, the suggestion ids and the comment ranges in one call. `read`, `suggestions` and the range half of `comments` all stand on that one document tree.
- **`read` is a projection of the tree, and the tree stays available.** The text is what the AI reads; the structure with character indexes (`--structure`) is what M3 needs to place a proposal at an exact position. Both are facts; neither is the other's summary.
- **Drive is the source of threads; Docs is the source of ranges.** `comments.list` is documented and stable and carries replies, authors, `resolved` and `modifiedTime`. The Docs read carries the character ranges, keyed by the same comment id. The reader joins them and reports `range: null` with a warning for a thread the Docs read did not place, rather than failing the whole read. The response shape of the Docs `comments` key is measured, not documented, so the decoder is loose on purpose and the first live run records a redacted fixture.
- **The cursor is opaque and dies with the session.** It encodes the newest `modifiedTime` seen. The binary emits it, the skill hands it back, and nothing writes it anywhere.
- **The front matter holds facts only.** The pairing (`document_id`), and the observation state the later milestones need across sessions: the last-seen suggestion snapshot (written here) and gdoc's own proposal ids (M3). No content hashes; SPEC.md "The diff" says why. The snapshot is written only after a fully successful read and only when the caller named the markdown file.
- **The docx reader uses `encoding/xml`, read-only.** SPEC's reason for `etree` is that `encoding/xml` corrupts OOXML on the way back out. Nothing here writes OOXML.

## Technical Details

**Principles.** Serves: 1 (one more static binary, one dependency with a written reason), 3 (every read goes through the guard with exactly the one id it was given; the witness is a second route where a write will matter), 4 (outcomes on the envelope, machinery in `warnings`, nothing printed but the one object; the text `read` prints is for a reader, the AI, and carries no markup it does not need). Strains: none.

**Tech stack.** Go 1.27. Standard library plus `github.com/goccy/go-yaml v1.19.2`.

**Global constraints:**

- Module path `gdoc`, directory `go/`. Every command: exactly one JSON object on stdout, exit 0 iff `ok`. Human prose to stderr. Never prompts, never reads stdin.
- The guard is opened with exactly the one document a command was given. No command in this milestone creates anything, so `AllowCreateIn` is never called by production code.
- Every Drive call carries `supportsAllDrives=true`: the test folder is a shared drive and a call without it returns a flat 404.
- Every Docs read carries `includeTabsContent=true`. A document with more than one tab is reported (`tabs`, `multi_tab: true`) and still read; only writes stop on it, and there are none here.
- Reads only. Nothing in this milestone sends a `POST` to Docs or Drive. The one write anywhere is to a local markdown file's front matter, and only when the caller named that file.
- Front matter outside the `gdoc:` key is preserved byte-identically. Line endings and the trailing newline are preserved too.
- `goccy/go-yaml` decodes strictly: unknown keys refused, duplicate keys refused, exactly one YAML document, required values validated rather than defaulting to useful-looking zeros.
- The document argument is the URL Nail pastes. A bare id is accepted too because it costs one line. Ids are `[A-Za-z0-9_-]{20,}`; anything else is refused before the guard is opened.
- Commit messages: `feat(v2): ...`, `test(v2): ...`, `docs: ...`, `refactor(v2): ...`. All commits from the repo root.

**The front-matter block, schema version 1.** What gdoc owns in a markdown note, and nothing beyond it:

```yaml
---
title: Supplier register policy        # everything outside gdoc: is the author's, untouched
gdoc:
  schema: 1
  document_id: 1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd
  folder_id: 1w0SresizE9Kr810VZRJwX4JtDBF4OqNr   # optional in M2, required by M6
  published:                            # written by publish (M6); absent until then
    at: 2026-09-06T10:12:00Z
    revision_id: "ALm37BX..."
  suggestions_seen:                     # written by `suggestions --md`, after a successful read
    at: 2026-09-06T11:00:00Z
    items:
      - id: suggest.abc123
        kind: insertion                 # insertion | deletion
        section: Scope                  # the heading text above it, "" before the first heading
        text: "critical "
  proposals: []                         # gdoc's own proposals; M3 writes, M2 defines the shape
---
```

Rules: `schema` is required and must be `1`; `document_id` is required and must match `^[A-Za-z0-9_-]{20,}$`; `suggestions_seen.items[].kind` is one of the two words; `proposals[]` items carry `id`, `comment_id`, `at`. A block that fails any rule is refused with an error naming the key, and the file is left untouched.

**The `read` text.** A deterministic projection of the document tree, one string:

- A paragraph with `namedStyleType` `HEADING_n` becomes a line of `n` `#` characters, a space, and its text. `TITLE` and `SUBTITLE` paragraphs are plain paragraphs (the title is also on the envelope). Other paragraphs are their text on one line; a paragraph with a `bullet` becomes `- ` plus its text, indented two spaces per `nestingLevel`. Paragraphs are separated by one blank line.
- A table becomes a pipe table: one row per `tableRow`, cells separated by ` | `, a `---` separator row after the first row, newlines inside a cell replaced by a space.
- A text run carrying `suggestedInsertionIds` is printed as `{+text+}[s:ID]`; one carrying `suggestedDeletionIds` as `{-text-}[s:ID]`. Adjacent runs with the same id and kind print as one span. A run carrying both is printed as a deletion then an insertion, because that is what Docs shows.
- A comment range from the Docs read wraps its text as `[[c:ID]]text[[/c]]`, where `ID` is the Drive comment id. Ranges that nest or overlap are opened and closed in index order; a range the reader cannot place is not printed and its id goes to `warnings`.
- Literal `{+`, `{-`, `+}`, `-}`, `[[` and `]]` in the document's own text are escaped with a backslash, so a marker in the output is always gdoc's.
- With more than one tab, each tab is preceded by a line `<!-- tab T: Title -->`. A single-tab document, which is every document gdoc makes, carries no tab lines.
- Line endings are `\n`. The string ends with one `\n`.

Pictures, drawings and equations print as `[image]`, `[drawing]` and `[equation]` placeholders in this milestone; reading them is `docs/backlog/read-pictures-and-drawings.md`. A footnote reference prints as `[^n]` and the footnote text is appended after the body under a `---` line, in order.

**The cursor.** `base64url(JSON{"v":1,"t":"<RFC3339 UTC>"})`. `t` is the newest `modifiedTime` seen across the comments returned, including their replies' `createdTime`. A run that saw nothing returns the cursor it was given, or omits the field when it was given none. A cursor the binary cannot decode is refused as an error naming the problem, never silently treated as "from the beginning".

**Markers.** A comment's `marker` is the first token of its content after leading whitespace, exactly `ai:`, `ai?` or `ai!`, else `none`. A reply's `by_gdoc` is true when its content opens with `🤖`. Both are string facts. Nothing here says whether the comment is answered; the skill reads the replies and decides.

**Output shapes.** Each command's `data`:

```jsonc
// gdoc read <url> [--structure]
{
  "document_id": "1AbC...", "title": "...", "revision_id": "...", "tabs": 1, "multi_tab": false,
  "text": "# Scope\n\nThe supplier register is reviewed {-annually-}[s:suggest.a1] {+every quarter+}[s:suggest.a1] by [[c:AAAA]]the operations team[[/c]].\n",
  "structure": { "tabs": [ { "id": "t.0", "title": "", "blocks": [ /* paragraphs, runs, tables, with start_index/end_index, style, suggestion ids */ ] } ] }   // only with --structure
}

// gdoc comments <url> [--since CURSOR] [--witness]
{
  "document_id": "1AbC...", "title": "...", "tabs": 1, "multi_tab": false,
  "cursor": "eyJ2IjoxLCJ0IjoiMjAyNi0wOS0wNlQxMTowMDowMFoifQ",
  "threads": [{
    "id": "AAAA...", "author": "Nail Khusnullin", "created": "...", "modified": "...",
    "content": "ai? which register does this refer to", "marker": "ai?", "resolved": false,
    "quoted": "the operations team", "range": {"tab": "t.0", "start": 1204, "end": 1223},   // or null
    "replies": [{"id": "...", "author": "...", "created": "...", "content": "🤖 The 2026 register.", "by_gdoc": true}],
    "witness": "anchored"   // only with --witness: anchored | detached | unmatched
  }]
}

// gdoc suggestions <url> [--md PATH]
{
  "document_id": "...", "tabs": 1, "multi_tab": false,
  "pending": [{"id": "suggest.abc", "kind": "insertion", "section": "Scope", "text": "critical "}],
  "gone_since_last_look": [{"id": "suggest.old", "kind": "deletion", "section": "Scope", "text": "annually", "seen_at": "..."}],   // only with --md
  "files_changed": ["policy.md"]   // only with --md, only after a successful read
}
```

Warnings ride in the envelope's `warnings`: the policy's `Warnings()`, a token that was refreshed and saved, a comment the Docs read did not place, a docx the witness could not match a thread in, a content element `read` printed as a placeholder.

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): everything achievable inside this repo. The packages, their tests, the command wiring, the documentation.
- **Post-Completion** (no checkboxes): the live run against a real document, and the redacted fixture it produces.

## Implementation Steps

---

### Task 1: The YAML dependency enters, and the boundary test admits it by name

**Files:**
- Modify: `go/go.mod`, `go/go.sum`
- Modify: `go/boundary/boundary_test.go` (`allowedModules` gains one entry)

**Interfaces:**
- Produces: `github.com/goccy/go-yaml v1.19.2` in `go.mod`; `TestNoThirdPartyDependencies` passes with exactly that module allowed and still refuses any other. This is the one-line widening PLAN.md and CLAUDE.md describe: the milestone that first needs a module adds its path, and nothing else.

- [ ] write the failing test change first: add `"github.com/goccy/go-yaml": "the gdoc: front matter and house.yaml; reason in SPEC.md"` to `allowedModules`, and make sure a canary asserts an unlisted module path in a scratch `go.mod` string is still refused
- [ ] run `cd go && go get github.com/goccy/go-yaml@v1.19.2 && go mod tidy` and confirm `go.sum` names no module other than go-yaml itself (PLAN.md: zero transitive modules; if `go mod tidy` pulls anything else, stop and record it with ⚠️ rather than allowlisting it)
- [ ] run the boundary test: green, and the refusal canary still fails on the unlisted path
- [ ] commit: `feat(v2): goccy/go-yaml enters, admitted by name in the boundary test`

---

### Task 2: The front matter: read the `gdoc:` block strictly, write it back without touching a byte around it

**Files:**
- Create: `go/internal/frontmatter/frontmatter.go` (split and join)
- Create: `go/internal/frontmatter/schema.go` (the typed block and its validation)
- Create: `go/internal/frontmatter/frontmatter_test.go`, `go/internal/frontmatter/schema_test.go`
- Create: `go/internal/frontmatter/testdata/*.md` (no front matter; front matter without `gdoc:`; a full block; CRLF line endings; an unknown key; a duplicate key; `schema: 2`)

**Interfaces:**
- Produces:
  - `type Block struct { Schema int; DocumentID string; FolderID string; Published *Published; SuggestionsSeen *SuggestionsSeen; Proposals []Proposal }` with yaml tags matching the schema above; `type SuggestionSeen struct { ID, Kind, Section, Text string }`; `type SuggestionsSeen struct { At time.Time; Items []SuggestionSeen }`; `type Proposal struct { ID, CommentID string; At time.Time }`.
  - `frontmatter.Read(src []byte) (*Block, error)`: `nil, nil` when the file has no front matter or the front matter has no `gdoc:` key; an error naming the key when the block is present and invalid. Strict go-yaml decode: unknown fields refused, duplicate keys refused (go-yaml's default), more than one YAML document refused.
  - `frontmatter.Write(src []byte, b *Block) ([]byte, error)`: a new file where only the `gdoc:` top-level key's span is replaced by the marshalled block. A file without front matter gets delimiters and a `gdoc:` block. A file with front matter but no `gdoc:` gets the block appended inside the existing delimiters. Everything else is byte-identical, line endings and trailing newline included. `Write` validates `b` before touching anything.
  - `(*Block).Validate() error`: the rules under "The front-matter block" above.
- The span is found by lines, never by re-marshalling the author's YAML: a top-level key is a line matching `^gdoc:` inside the `---` delimiters, and its span runs to the next line that starts at column 0 with a non-space character, or to the closing delimiter.

- [ ] write the failing tests for `Read`: each fixture, asserting the nil-nil cases, the decoded values of the full block, and one refusal per invalid fixture with the key named in the error
- [ ] write the failing tests for `Write`: `Read` then `Write` of an unchanged block is byte-identical; changing one field changes only lines inside the `gdoc:` span (line diff: no other line moved); CRLF stays CRLF; a file without front matter gains delimiters and the block and nothing else; a file with a `title:` and no `gdoc:` keeps its `title:` line byte-identical; an invalid block returns the error and the input unchanged
- [ ] write the failing tests for `Validate`: missing `document_id`, `schema: 2`, an unknown `kind`, a proposal without an id
- [ ] run `cd go && go test ./internal/frontmatter/` and watch it fail
- [ ] implement `schema.go` and `frontmatter.go`
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): the gdoc: front-matter block, strict in, byte-preserving out`

---

### Task 3: `internal/gapi`: the authenticated session, and one more room for `net/http`

**Files:**
- Create: `go/internal/gapi/session.go`
- Create: `go/internal/gapi/session_test.go`
- Modify: `go/boundary/boundary_test.go` (`allowed` gains `internal/gapi`, with the reason beside it; `builders` does not change)

**Interfaces:**
- Consumes: `guard.NewClient`, `auth.Load`, `(Token).Refresh`, `auth.Save`.
- Produces:
  - `gapi.Open(p *guard.Policy, base http.RoundTripper) (*Session, error)`: loads the token (an absent token is the error `auth.Load` already returns, which names `gdoc auth login`), builds the guard's client from `p`, returns a session. `base == nil` means the real wire.
  - `(*Session).GetJSON(ctx, rawURL string, into any) error` and `(*Session).GetBytes(ctx, rawURL string, limit int64) ([]byte, error)`: set `Authorization: Bearer <token>` and `Accept`, send through the guard's client, decode or read up to `limit`. A non-2xx answer is an error carrying the status and Google's `error.message` when there is one, never the raw body.
  - Refresh policy: before the first request, if `Token.Expired()`, refresh, save, and record `"the access token was refreshed and saved"` in `(*Session).Warnings()`. On a 401, refresh once, save, retry that request once; a second 401 is `"the token was refused twice; run: gdoc auth login"`. A refresh that fails names the failure and does not save.
  - `(*Session).Warnings() []string`: the policy's `Warnings()` plus the session's own, in that order.
- Boundary: `internal/gapi` names `*http.Request` and `*http.Client` and builds neither, so it joins `allowed` with the comment "builds requests and sets the bearer; takes the guard's client as a parameter, and builders below proves it never makes one", and stays out of `builders`. Write the allowlist change first, watch the boundary test fail on the missing room, then create the package.

- [ ] add `internal/gapi` to `allowed` in the boundary test and run it: FAIL, the room does not exist yet
- [ ] write the failing session tests over a fake `RoundTripper` under a temp `GDOC_CONFIG_DIR` with a fixture token: exactly one `Bearer` header; an expired fixture token triggers exactly one refresh POST before the GET, the saved file carries the new access token, and `Warnings()` says so; a 401 triggers one refresh and one retry; two 401s produce the named error and no further request; a 404 with a Google error body surfaces `message` and the status; `GetBytes` stops at `limit`; a policy with no `AllowFile` refuses the read with a guard refusal
- [ ] run the tests and watch them fail
- [ ] implement `session.go`
- [ ] run the full suite including the boundary test: green in both directions (on a scratch copy, remove the `net/http` import from `session.go` and confirm the disappearance check fires)
- [ ] commit: `feat(v2): gapi session: bearer, refresh on expiry and on one 401, through the guard`

---

### Task 4: `internal/docs`: the Docs read, tabs, the document tree, suggestion ids, comment ranges

**Files:**
- Create: `go/internal/docs/docs.go` (fetch and the top-level model)
- Create: `go/internal/docs/walk.go` (paragraphs, tables, headings, runs, footnotes)
- Create: `go/internal/docs/docs_test.go`, `go/internal/docs/walk_test.go`
- Create: `go/internal/docs/testdata/*.json` (single tab with headings, a bulleted list and a table; two tabs; pending insertions and deletions split across formatting runs; `comments` ranges; the pre-tabs shape with no `tabs` key; a paragraph with an inline image and a footnote reference)
- Modify: `go/internal/guard/params.go` (the `docsReadParams` comment says "These four" and the map holds five: change it to "these three documents.get parameters plus the two system parameters gdoc sets")

**Interfaces:**
- Consumes: `gapi.Session`.
- Produces:
  - `docs.URL(id string) string`: `https://docs.googleapis.com/v1/documents/{id}?includeTabsContent=true&suggestionsViewMode=SUGGESTIONS_INLINE&commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED`. One URL, one read, three views.
  - `docs.Fetch(ctx, s *gapi.Session, id string) (*Document, error)` and `docs.Parse(raw []byte) (*Document, error)`, so the fixtures test the same code the wire feeds.
  - `type Document struct { ID, Title, RevisionID string; Tabs []Tab; CommentRanges map[string]Range; Unplaced []string; Footnotes map[string]string }`; `type Tab struct { ID, Title string; Body []Block }`; `type Block` holding either a `Paragraph { Style string; Bullet *Bullet; Runs []Run; StartIndex, EndIndex int }` or a `Table [][]Cell` (cells hold blocks); `type Run struct { Kind string /* text | image | drawing | equation | footnote_ref */; Text string; StartIndex, EndIndex int; InsertionIDs, DeletionIDs []string; FootnoteID string }`; `type Range struct { Tab string; Start, End int }`; `type Bullet struct { NestingLevel int }`.
  - `(*Document).MultiTab() bool`.
  - The pre-tabs shape: when the JSON has no `tabs`, the top-level `body` is the one tab with id `t.0`.
  - `CommentRanges` decodes the top-level `comments` array loosely (`id`, and a range under whatever key the measured shape uses: try `range`, then `anchor.range`, then top-level `startIndex`/`endIndex`). An entry without a readable range goes to `Unplaced`. ⚠️ The shape is measured, not documented: the first live run records a redacted fixture into `testdata/` and the decoder is tightened to it in a follow-up commit.

- [ ] write the failing tests on the fixtures: two tabs and `MultiTab()`; the pre-tabs fixture reads as one tab `t.0`; headings carry their style and table text lands in cells in reading order; runs carry their suggestion ids; the image and footnote runs carry their kinds and the footnote text is in `Footnotes`; `CommentRanges` places the ranged comment and lists the unplaced one; `URL` carries the three parameters and nothing else; `Fetch` over a fake wire sends `includeTabsContent=true` (assert on the request the fake saw)
- [ ] run the tests and watch them fail
- [ ] implement `docs.go` and `walk.go`; fix the `docsReadParams` comment
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): the Docs read with tabs, the document tree, suggestion ids and comment ranges`

---

### Task 5: `internal/view`: the document as text the AI reads

**Files:**
- Create: `go/internal/view/text.go`
- Create: `go/internal/view/text_test.go`
- Create: `go/internal/view/testdata/*.golden` (the expected text for each `docs` fixture)

**Interfaces:**
- Consumes: `docs.Document`.
- Produces: `view.Text(d *docs.Document) (string, []string)`: the projection under "The `read` text" in Technical Details, and the warnings it raised (placeholders printed, ranges not placed). Deterministic: the same document gives the same bytes. `view.Structure(d *docs.Document) any`: the tree as the `structure` field, with `start_index`/`end_index` on every paragraph and run, JSON tags in snake_case.

- [ ] write the failing golden tests, one per `docs` fixture: headings become `#` lines; bullets become `- ` with nesting; the table becomes a pipe table; the split insertion runs print as one `{+...+}[s:ID]` span and the deletion as `{-...-}[s:ID]`; the comment range wraps its text with `[[c:ID]]`/`[[/c]]`; a literal `{+` in document text comes out escaped; the two-tab fixture has two `<!-- tab -->` lines and the single-tab one has none; the image prints `[image]` and raises a warning; the footnote prints `[^1]` and its text after `---`; the output ends with exactly one `\n`
- [ ] write the failing test for `Structure`: round-trips through `encoding/json` and carries the indexes of the fixture's first run
- [ ] run the tests and watch them fail
- [ ] implement `text.go`
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): read's text projection, suggestions inline and comment anchors marked`

---

### Task 6: `internal/suggestions`: what is pending, and what stopped being pending

**Files:**
- Create: `go/internal/suggestions/suggestions.go`
- Create: `go/internal/suggestions/suggestions_test.go`

**Interfaces:**
- Consumes: `docs.Document`, `frontmatter.SuggestionsSeen`.
- Produces:
  - `type Pending struct { ID, Kind, Section, Text string }` and `suggestions.Pending(d *docs.Document) []Pending`: port of `gdoc/suggestions.py`. Runs sharing one id and kind are joined in order; `Section` is the text of the heading above, `""` before the first; tables are walked; empty text after trimming is dropped; every id a run carries is reported (v1 took `ids[0]`; a run with two ids is two suggestions).
  - `type Gone struct { ID, Kind, Section, Text string; SeenAt time.Time }` and `suggestions.Gone(seen *frontmatter.SuggestionsSeen, now []Pending) []Gone`: every item in `seen` whose id is not in `now`, with what it said last time. A fact. Whether it was accepted or rejected is the skill's, reading `read`'s text.
  - `suggestions.Snapshot(now []Pending, at time.Time) *frontmatter.SuggestionsSeen`.

- [ ] write the failing tests: two runs one id join to one pending; a run with two ids is two pendings; heading context follows the walk into a table; `Gone` lists exactly the ids that left and carries their old text and `seen_at`; `Gone` with a nil snapshot is empty; `Snapshot` round-trips through `frontmatter.Write` and `Read`
- [ ] run the tests and watch them fail
- [ ] implement
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): pending suggestions with stable ids, and what stopped being pending`

---

### Task 7: `internal/comments`: threads as facts, and the cursor

**Files:**
- Create: `go/internal/comments/comments.go` (fetch and join)
- Create: `go/internal/comments/cursor.go`
- Create: `go/internal/comments/comments_test.go`, `go/internal/comments/cursor_test.go`
- Create: `go/internal/comments/testdata/*.json` (a Drive `comments.list` page with replies; a resolved thread; a thread with a 🤖 reply; a two-page listing with `nextPageToken`)

**Interfaces:**
- Consumes: `gapi.Session`, `docs.Document` (for `CommentRanges`).
- Produces:
  - `comments.ListURL(id, pageToken, since string) string`: `https://www.googleapis.com/drive/v3/files/{id}/comments?supportsAllDrives=true&pageSize=100&includeDeleted=false&fields=nextPageToken,comments(id,author(displayName,me),createdTime,modifiedTime,content,resolved,quotedFileContent(value),replies(id,author(displayName,me),createdTime,content))` plus `startModifiedTime` when `since` is set. Every parameter is one `driveCommentListParams` names; the `fields` mask names no permission surface.
  - `comments.Fetch(ctx, s *gapi.Session, id string, since *Cursor) ([]RawComment, error)`: follows `nextPageToken` to the end.
  - `type Thread struct { ID, Author, Created, Modified, Content, Marker string; Resolved bool; Quoted string; Range *docs.Range; Replies []Reply; Witness string }`, `type Reply struct { ID, Author, Created, Content string; ByGdoc bool }`. No `Handled`: SPEC.md, corrected 2026-09-06.
  - `comments.Threads(raw []RawComment, d *docs.Document) ([]Thread, []string)`: joins ranges by id, sets `Marker` and `ByGdoc` per Technical Details, returns the ids the Docs read did not place.
  - `type Cursor struct { At time.Time }`, `comments.ParseCursor(s string) (*Cursor, error)` (refuses anything not `v:1`), `comments.NextCursor(prev *Cursor, threads []Thread) *Cursor` (newest `Modified` and reply `Created` seen, else `prev`), `(*Cursor).String() string`.

- [ ] write the failing tests: `ListURL` carries exactly the allowed parameters and adds `startModifiedTime` only with a cursor; `Fetch` over a fake wire follows two pages and stops; markers for `ai:`, `ai?`, `ai!`, `AI:` (not a marker: exact match) and `none`; `ByGdoc` true only for a reply opening with 🤖; `Resolved` passes through; a thread without a range has `Range == nil` and its id in the unplaced list; `NextCursor` picks the newest time across comments and replies and falls back to `prev`; `ParseCursor` refuses `v:2` and non-base64, and round-trips `String()`
- [ ] run the tests and watch them fail
- [ ] implement
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): comment threads as facts, with ranges, markers and the --since cursor`

---

### Task 8: `internal/docx`: the honest witness

**Files:**
- Create: `go/internal/docx/docx.go` (zip, comments.xml, document.xml)
- Create: `go/internal/docx/match.go` (threads against the docx)
- Create: `go/internal/docx/docx_test.go`, `go/internal/docx/match_test.go`
- Create: `go/internal/docx/testdata/` (a minimal docx built by a test helper from two XML strings so the fixture is readable, plus room for the redacted real export from Post-Completion)

**Interfaces:**
- Consumes: `gapi.Session` (`GetBytes` on `https://www.googleapis.com/drive/v3/files/{id}/export?supportsAllDrives=true&mimeType=application/vnd.openxmlformats-officedocument.wordprocessingml.document`, limit 32 MiB).
- Produces:
  - `docx.Export(ctx, s *gapi.Session, id string) ([]byte, error)` and `docx.Parse(b []byte) (*File, error)`.
  - `type File struct { Comments []Comment }`, `type Comment struct { ID, Author, Date, Text string; Anchored bool; Span string }`: from `word/comments.xml` (`w:comment` with `w:id`, `w:author`, `w:date`, concatenated `w:t`); `Anchored` true when `word/document.xml` carries a `w:commentRangeStart` with that id, `Span` the concatenated `w:t` text between it and its `w:commentRangeEnd`. `encoding/xml`, namespace by URI, read-only.
  - `docx.Match(threads []comments.Thread, f *File) []comments.Thread`: sets `Witness` on a copy of each thread: `anchored` when a docx comment with the same normalised content (and the same author display name when the docx has one) has `Anchored`; `detached` when matched and not anchored; `unmatched` when nothing matches. Replies are not matched: the witness question is whether the thread is attached, and the first comment is the thread.

- [ ] write the failing tests: `Parse` on the helper-built docx finds two comments, one anchored with the right span and one detached; a docx without `word/comments.xml` parses to zero comments; a zip that is not a docx is refused by name; `Match` gives `anchored`, `detached` and `unmatched` across three threads and leaves the input slice untouched; `Export` over a fake wire asks for the docx mime and `supportsAllDrives=true`
- [ ] run the tests and watch them fail
- [ ] implement
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): the docx export reader, and the witness match against comment threads`

---

### Task 9: The three commands on the envelope, URL parsing, and `GrantInPlace` leaves

**Files:**
- Modify: `go/cmd/gdoc/main.go` (dispatch for `read`, `comments`, `suggestions`; `usage`)
- Create: `go/cmd/gdoc/read.go` (the three command functions, argument parsing, `documentID(arg string) (string, error)` ported from `gdoc/docid.py`'s document half)
- Create: `go/cmd/gdoc/read_test.go`
- Modify: `go/cmd/gdoc/main_test.go` (usage string, unknown-flag refusals)
- Modify: `go/internal/guard/policy.go`, `go/internal/guard/policy_test.go` (delete `GrantInPlace` and its tests)
- Create: `go/internal/live/live_test.go` (the opt-in end-to-end test)

**Interfaces:**
- Consumes: everything Tasks 1 to 8 produced.
- Produces:
  - `gdoc read <url> [--structure]`, `gdoc comments <url> [--since CURSOR] [--witness]`, `gdoc suggestions <url> [--md PATH]`, each printing the `data` shape under Technical Details, with `warnings` from `Session.Warnings()` plus the command's own.
  - `documentID` accepts `/document/d/{id}` and `?id={id}` URLs and a bare id of 20 or more `[A-Za-z0-9_-]`; anything else is refused quoting the input. A folder parser is not built: no M2 command takes a folder.
  - Argument parsing is strict, as `dispatch` already is: an unknown flag, a repeated flag, a missing value or an extra positional argument fails naming it. Nothing is ignored.
  - The policy is opened with exactly one id: `p.AllowFile(id, guard.LevelSuggest)`.
  - `suggestions --md`: after a successful read, `frontmatter.Read` the file; refuse when the block's `document_id` is not the document read (writing another document's observation into this file is the wrong file); compute `Gone`; write the new snapshot with `frontmatter.Write` through a temp file in the same directory plus `os.Rename`, as `auth.Save` does; report `files_changed`. A read that failed writes nothing and the envelope says so.
  - A session factory behind a package variable, as `login` is, so the command tests stand in for the wire: `var openSession = func(p *guard.Policy) (*gapi.Session, error)`.
  - `usage` becomes `Commands: auth status, auth login, read, comments, suggestions`.
- PLAN.md M2 says `GrantInPlace`, `AllowCreateIn` and `Token.Refresh` are deleted if M2 lands without a production caller. `Token.Refresh` gains one in Task 3. `GrantInPlace` has none and nothing depends on it: delete it with its tests; M7 adds it back beside its caller. ⚠️ `AllowCreateIn` also has no production caller in M2, but the transport's create path (`checkParent`, `learnFromCreate`, the parent-check refusals, the upload-shape checks) is built on `createIn` and tested through it, so deleting it deletes the second door and a page of guard tests. This plan keeps it and says so here; Nail can flip that decision in review.
- The live test: gated on `GDOC_LIVE_TEST=1`, skipping with a message otherwise. It opens a policy with `AllowCreateIn` on the test folder, creates a Docs file there (`files.create` with `supportsAllDrives=true`; the guard learns the id at `LevelFull`), inserts two paragraphs and a heading, creates one Drive comment on a word, runs `read` and `comments --witness` through the real code path, asserts the heading is a `#` line and the comment's text is wrapped with `[[c:ID]]`, asserts one thread with `marker: none` and `witness` not `unmatched`, saves the export and the Docs read as redacted fixtures only when `GDOC_LIVE_RECORD=1`, and trashes the file, asserting `trashed: true` on the read-back. This is the one place M2 writes to Drive, on a document the run itself created.

- [ ] write the failing command tests over a stubbed `openSession` and fixtures: each command prints exactly one JSON object (`decodeOne`); `read` carries the golden text and no `structure` without the flag, and the tree with it; `comments` puts the policy's and the session's warnings on the envelope and reports an unplaced thread; `--since` with a bad cursor fails naming it; `--witness` sets `witness` per thread; `suggestions --md` on a fixture file with a matching `document_id` writes the snapshot, leaves every other line byte-identical, reports `gone_since_last_look` and `files_changed`; `suggestions --md` on a file whose `document_id` differs fails and leaves the file untouched; `suggestions --md` when the read fails leaves the file untouched; `documentID` accepts the two URL shapes and a bare id and refuses a short id quoting it; an unknown flag, a repeated flag and an extra argument each fail naming the offender; `usage` in the unknown-command error lists the five commands
- [ ] write the failing guard test change: remove `TestGrantInPlaceUpgrades` and every `GrantInPlace` reference; the package must compile without the method
- [ ] run the tests and watch them fail
- [ ] implement `read.go`, wire `dispatch`, delete `GrantInPlace`
- [ ] write the live test, run it with the variable unset and confirm it skips with a message; do not run it live in the unattended run
- [ ] run the full suite with `-race`, gofmt, vet, `make build`: green
- [ ] commit: `feat(v2): read, comments and suggestions on the envelope; GrantInPlace leaves until M7`

---

### Task 10: Verify acceptance criteria

**Files:**
- Modify: none expected. Fix whatever the checks below break.

**Interfaces:**
- Consumes: everything Tasks 1 to 9 produced. Produces: proof that PLAN.md's M2 description, as corrected 2026-09-06, holds: `documents.get` with `includeTabsContent=true`, tab detection, `read` as text with suggestions inline and anchors marked plus the structure on a flag, threads with real ranges and markers as facts, the opaque cursor, pending suggestions with stable ids and what stopped being pending, the docx reader, the versioned front-matter block, strict YAML.

- [ ] verify every item in the Overview and in PLAN.md M2 is implemented, by reading each command's test and the fixture it runs on
- [ ] verify no judgement leaked into Go: grep the new packages for `handled`, `accepted`, `rejected`, `matters`, `drift`; each hit is either a fact with a different name or a defect to remove
- [ ] verify the reads are guard-bounded: a command test where the fake wire fails the test if anything reaches it proves a refused id never reaches the transport (mirror `acceptance_test.go`)
- [ ] verify the boundary test in both directions on a scratch copy: a stray `net/http` in `internal/comments` fails it; removing the import from `internal/gapi/session.go` fails it; a second module in `go.mod` fails it
- [ ] verify front-matter preservation on real files: `frontmatter.Write` of an unchanged block on a fixture with front matter, then `cmp` the bytes
- [ ] run the full test suite: `cd go && go test -race ./...`
- [ ] run the formatter check: `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- [ ] run the vet check: `cd go && go vet ./...`
- [ ] verify coverage: every exported function under `go/internal/` has a test (`cd go && go test ./... -cover`); add tests rather than lowering the bar
- [ ] verify `make build` and `make dist` still produce the binaries with `CGO_ENABLED=0`, and record the binary size delta from the YAML module in this plan with a ➕ line
- [ ] commit any fixes this task made

---

### Task 11: [Final] Update documentation

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `docs/v2/PLAN.md`

**Interfaces:**
- Produces: the repo's account of what now exists, so M3 starts from documentation that matches the tree.

- [ ] update `README.md`: the three read commands, their arguments, the `read` text conventions (`{+ +}`, `{- -}`, `[s:ID]`, `[[c:ID]]`), the cursor, `--witness`, `--md`, and the `gdoc:` block a paired note carries
- [ ] update `CLAUDE.md` under "v2 lives at `go/`": the facts-only rule and the test for it; the four rooms of the import allowlist and why `internal/gapi` is one; `allowedModules` holds go-yaml; the front-matter block version 1 and its rules (strict decode, byte-preserving write, snapshot written only after a successful read); `read`'s text conventions and the escaping rule; the cursor is opaque and dies with the session; `GrantInPlace` is gone until M7 and `AllowCreateIn` stayed, with the reason; pictures and drawings are in the backlog
- [ ] update `docs/v2/PLAN.md`: mark M2 done with the date, record what it left for M3 (the Docs `comments` range shape is measured from the live fixture; `AllowCreateIn` has its first production caller in M6; the review skill reads `comments` and judges), and the binary size delta
- [ ] run the full test suite one more time: `cd go && go test -race ./...`
- [ ] commit the documentation updates: `docs: M2 lands, the read commands and the front-matter block written down`
- The harness moves this plan to `docs/plans/completed/` when the run finishes. Nobody here moves it, so this is not a checkbox.

## Post-Completion

*Items needing a real account or a person. No checkboxes: these are informational.*

**Manual verification:**

- Run the live test once, on Nail's machine, with `GDOC_LIVE_TEST=1 GDOC_LIVE_RECORD=1`: it creates a document in the test folder, comments on it through the API, reads it through `read` and `comments --witness`, records a redacted docx export and a redacted Docs read into `testdata/`, and trashes the document. Then tighten `docs.CommentRanges` to the recorded shape and commit the fixtures.
- Run `gdoc read <one of Nail's real documents>` and read the text against the document open in the browser: headings, a table, a pending suggestion, a comment anchor. Reading a real document is allowed; nothing here writes to one.
- Run `gdoc comments <that document>` and compare thread count, markers and replies against the browser.

**Known and out of scope for this plan:**

- Pictures, diagrams and Google Drawings print as placeholders. `docs/backlog/read-pictures-and-drawings.md`.
- Numbered lists print as `- ` like bulleted ones: telling them apart needs the `lists` map and its glyph types. A ➕ candidate for M3 if the review skill needs it.
- The revmux review of the M1 guard (2026-09-06) reported one major and four minor findings in `go/internal/guard`: `isSuggestMode` reads `writeControl` more loosely than the server does; a repeated JSON key collapses before `judgeRequests` and `checkParent` see it; `DELETE` and `PATCH` on any comment are carried at `LevelSuggest`; `resumable` uploads can never complete; nothing tests that a refused request closes the body. None is a read path, so none is in this plan. They belong to M3, the first milestone that sends a write, and are recorded here so they are not lost.

**External system updates:**

- None. Nothing in this milestone changes the token file, the config, or anything v1 reads.
