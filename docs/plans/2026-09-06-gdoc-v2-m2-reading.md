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

- [x] write the failing test change first: add `"github.com/goccy/go-yaml": "the gdoc: front matter and house.yaml; reason in SPEC.md"` to `allowedModules`, and make sure a canary asserts an unlisted module path in a scratch `go.mod` string is still refused
- [x] run `cd go && go get github.com/goccy/go-yaml@v1.19.2 && go mod tidy` and confirm `go.sum` names no module other than go-yaml itself (PLAN.md: zero transitive modules; if `go mod tidy` pulls anything else, stop and record it with ⚠️ rather than allowlisting it)
- [x] run the boundary test: green, and the refusal canary still fails on the unlisted path
- [x] commit: `feat(v2): goccy/go-yaml enters, admitted by name in the boundary test`

⚠️ `go mod tidy` is deferred to Task 2. Nothing imports go-yaml yet, so tidy deletes the
require line it was just given, and Task 1 would commit an empty `go.mod` again. `go get`
alone was run, and `go.sum` names go-yaml and nothing else, which is the zero-transitive-
modules claim PLAN.md makes, confirmed. The require line carries `// indirect` until
`internal/frontmatter` imports it in Task 2, which is the honest marker for "required, not
yet used". Task 2 runs `go mod tidy` and the marker goes.

➕ `allowedModules` changed shape from `map[string]bool` to `map[string]string`, path against
reason, so the reason lives beside the entry rather than in the comment above the map. Two
tests were added with it: `TestAllowedModulesAreReallyRequired` is the disappearance half,
which fails when the map names a module `go.mod` no longer requires, and
`TestAllowedModulesStillRefusesAnUnlistedPath` is the canary the checkbox above asks for. The
judgement moved into `unlistedModules` and `summedModules` so scratch text can be put through
it: once a module is both listed and required, a test that only reads the real files passes
whether the refusal still works or not.

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

- [x] write the failing tests for `Read`: each fixture, asserting the nil-nil cases, the decoded values of the full block, and one refusal per invalid fixture with the key named in the error
- [x] write the failing tests for `Write`: `Read` then `Write` of an unchanged block is byte-identical; changing one field changes only lines inside the `gdoc:` span (line diff: no other line moved); CRLF stays CRLF; a file without front matter gains delimiters and the block and nothing else; a file with a `title:` and no `gdoc:` keeps its `title:` line byte-identical; an invalid block returns the error and the input unchanged
- [x] write the failing tests for `Validate`: missing `document_id`, `schema: 2`, an unknown `kind`, a proposal without an id
- [x] run `cd go && go test ./internal/frontmatter/` and watch it fail
- [x] implement `schema.go` and `frontmatter.go`
- [x] run the tests, gofmt, vet: green
- [x] commit: `feat(v2): the gdoc: front-matter block, strict in, byte-preserving out`

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

- [x] add `internal/gapi` to `allowed` in the boundary test and run it: FAIL, the room does not exist yet
- [x] write the failing session tests over a fake `RoundTripper` under a temp `GDOC_CONFIG_DIR` with a fixture token: exactly one `Bearer` header; an expired fixture token triggers exactly one refresh POST before the GET, the saved file carries the new access token, and `Warnings()` says so; a 401 triggers one refresh and one retry; two 401s produce the named error and no further request; a 404 with a Google error body surfaces `message` and the status; `GetBytes` stops at `limit`; a policy with no `AllowFile` refuses the read with a guard refusal
- [x] run the tests and watch them fail
- [x] implement `session.go`
- [x] run the full suite including the boundary test: green in both directions (on a scratch copy, remove the `net/http` import from `session.go` and confirm the disappearance check fires)
- [x] commit: `feat(v2): gapi session: bearer, refresh on expiry and on one 401, through the guard`

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

- [x] write the failing tests on the fixtures: two tabs and `MultiTab()`; the pre-tabs fixture reads as one tab `t.0`; headings carry their style and table text lands in cells in reading order; runs carry their suggestion ids; the image and footnote runs carry their kinds and the footnote text is in `Footnotes`; `CommentRanges` places the ranged comment and lists the unplaced one; `URL` carries the three parameters and nothing else; `Fetch` over a fake wire sends `includeTabsContent=true` (assert on the request the fake saw)
- [x] run the tests and watch them fail
- [x] implement `docs.go` and `walk.go`; fix the `docsReadParams` comment
- [x] run the tests, gofmt, vet: green
- [x] commit: `feat(v2): the Docs read with tabs, the document tree, suggestion ids and comment ranges`

➕ `Fetch` takes `docs.Reader`, a one-method interface (`GetJSON`), not `*gapi.Session`. A `*gapi.Session` satisfies it, so no caller changes. The reason is the boundary test: its import allowlist reads test files too, so a fake `http.RoundTripper` in `docs_test.go` would have made `internal/docs` a fifth room that names `net/http`. The interface keeps the reader pure and its tests wire-free, and the fake records the URL the reader built, which is the assertion the task asked for. Every reader package after this one takes the same seam.

➕ The guard-refusal case moved out of `internal/docs`: with no wire in the room, the test that a refused id never reaches the transport belongs to the command layer, where Task 10 already asks for it. What `docs_test.go` proves instead is that `Fetch` hands the session's refusal back unwrapped.

➕ The `params.go` change is not comment-only. `driveExportParams` lost `supportsAllDrives` as well, on the measurement recorded under Task 8: an allowlist naming a parameter its method does not define permits a request nothing should send. `docsReadParams` itself is unchanged and still holds five.

➕ `Run.Kind` has a sixth value, `object`: an embedded object carrying neither `imageProperties` nor `embeddedDrawingProperties`. Calling it an image would be a guess, and this package reports rather than guesses. Task 5's projection prints `[object]` for it.

---

### Task 5: `internal/view`: the document as text the AI reads

**Files:**
- Create: `go/internal/view/text.go`
- Create: `go/internal/view/text_test.go`
- Create: `go/internal/view/testdata/*.golden` (the expected text for each `docs` fixture)

**Interfaces:**
- Consumes: `docs.Document`.
- Produces: `view.Text(d *docs.Document) (string, []string)`: the projection under "The `read` text" in Technical Details, and the warnings it raised (placeholders printed, ranges not placed). Deterministic: the same document gives the same bytes. `view.Structure(d *docs.Document) any`: the tree as the `structure` field, with `start_index`/`end_index` on every paragraph and run, JSON tags in snake_case.

- [x] write the failing golden tests, one per `docs` fixture: headings become `#` lines; bullets become `- ` with nesting; the table becomes a pipe table; the split insertion runs print as one `{+...+}[s:ID]` span and the deletion as `{-...-}[s:ID]`; the comment range wraps its text with `[[c:ID]]`/`[[/c]]`; a literal `{+` in document text comes out escaped; the two-tab fixture has two `<!-- tab -->` lines and the single-tab one has none; the image prints `[image]` and raises a warning; the footnote prints `[^1]` and its text after `---`; the output ends with exactly one `\n`
- [x] write the failing test for `Structure`: round-trips through `encoding/json` and carries the indexes of the fixture's first run
- [x] run the tests and watch them fail
- [x] implement `text.go`
- [x] run the tests, gofmt, vet: green
- [x] commit: `feat(v2): read's text projection, suggestions inline and comment anchors marked`

---

### Task 6: `internal/suggestions`: what is pending, and what stopped being pending

**Files:**
- Create: `go/internal/suggestions/suggestions.go`
- Create: `go/internal/suggestions/suggestions_test.go`

**Interfaces:**
- Consumes: `docs.Document`, `frontmatter.SuggestionsSeen`.
- Produces:
  - `type Pending struct { ID, Kind, Section, Text string }` and `suggestions.List(d *docs.Document) []Pending`: port of `gdoc/suggestions.py`. Runs sharing one id and kind are joined in order; `Section` is the text of the heading above, `""` before the first; tables are walked; empty text after trimming is dropped; every id a run carries is reported (v1 took `ids[0]`; a run with two ids is two suggestions).
  - `type Gone struct { Pending; SeenAt time.Time }` and `suggestions.GoneSince(seen *frontmatter.SuggestionsSeen, now []Pending) []Gone`: every item in `seen` whose id is not in `now`, with what it said last time. A fact. Whether it was accepted or rejected is the skill's, reading `read`'s text.
  - `suggestions.Snapshot(now []Pending, at time.Time) *frontmatter.SuggestionsSeen`.

- ⚠️ Naming deviation from the plan, forced by Go: a package cannot hold a type and a function under one name, so `Pending` and `Gone` stayed the types and the functions became `List` and `GoneSince`. `Gone` embeds `Pending`, so `encoding/json` still flattens it to the `id`/`kind`/`section`/`text`/`seen_at` object the Output shapes section prints.
- ➕ Two decisions the plan did not state, both written into the code's comments: heading context resets at each tab boundary (a heading in one tab is not above anything in the next), and only a text run is read as a suggestion (a footnote reference carries its number as text, so a suggested footnote would otherwise be reported as the insertion of "1").

- [x] write the failing tests: two runs one id join to one pending; a run with two ids is two pendings; heading context follows the walk into a table; `Gone` lists exactly the ids that left and carries their old text and `seen_at`; `Gone` with a nil snapshot is empty; `Snapshot` round-trips through `frontmatter.Write` and `Read`
- [x] run the tests and watch them fail
- [x] implement
- [x] run the tests, gofmt, vet: green
- [x] commit: `feat(v2): pending suggestions with stable ids, and what stopped being pending`

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

- ⚠️ **`supportsAllDrives` is not on this call, and the plan's URL above is wrong about it.** `comments.list` does not define the parameter: it belongs to the files collection, and the guard's `driveCommentListParams`, written from the Drive v3 reference, names `fields`, `pageSize`, `pageToken`, `includeDeleted` and `startModifiedTime` and not that one. Sending it would be refused inside the process by the guard's own query allowlist, so every comments read would die before the wire. The shared-drive 404 the global constraint describes is a files-collection problem; a comment collection hangs off a file already addressed by id. The milestone's other Drive read, the docx export in Task 8, does not carry it either, for the same reason: see Task 8. `files.create` in the live test does define it, and keeps it.
- ➕ `ListURL` builds its query through `url.Values.Encode` rather than concatenation. A page token is Drive's opaque string, and one carrying an `&` would end the parameter and start another: the request on the wire would then not be the request the guard judged. The test proves the URL by handing it to a real `guard.Policy.Judge`, not by comparing it to a second copy of the allowlist, which would pass while the command failed.
- ⚠️ `Fetch` takes a local `Reader` interface (one `GetJSON`) rather than `*gapi.Session`, the same deviation `internal/docs` made and for the same reason: this room may not name `net/http`, and a test of it must not have to.
- ➕ **The cursor carries a third field, `i`, and the plan's shape above is short by it.** The encoding is `base64url(JSON{"v":1,"t":"<RFC3339 UTC>","i":["<comment id>"]})` and `Cursor` is `struct { At time.Time; Ids []string }`. `t` keeps its milliseconds, because Drive sends them and a floor rounded down to the second re-reports the same thread on every poll for ever. `i` names the threads whose own newest instant was `t`: `startModifiedTime` is an inclusive bound, so `Fetch` narrows the answer to what is strictly newer, and two comments sharing one millisecond with only the first reported cannot be told apart by any comparison on instants. The version stays 1, because a cursor written before `i` existed still reads and costs one repeated thread rather than a lost one. Written up in full in CLAUDE.md.
- ➕ Three decisions the plan did not state, all written into the code's comments: `Threads` tolerates a nil document (no ranges, every thread unplaced) because the witness path reads threads with no Docs read behind them; a thread's `Replies` is never nil, so an empty thread prints `[]` rather than `null`; and `prev` is the cursor's floor rather than only its fallback, because a cursor that goes backwards makes the next poll re-report what this one just reported. A `modifiedTime` that does not parse is stepped over rather than failing the listing: one repeated thread is cheaper than a failed review.

- [x] write the failing tests: `ListURL` carries exactly the allowed parameters and adds `startModifiedTime` only with a cursor; `Fetch` over a fake wire follows two pages and stops; markers for `ai:`, `ai?`, `ai!`, `AI:` (not a marker: exact match) and `none`; `ByGdoc` true only for a reply opening with 🤖; `Resolved` passes through; a thread without a range has `Range == nil` and its id in the unplaced list; `NextCursor` picks the newest time across comments and replies and falls back to `prev`; `ParseCursor` refuses `v:2` and non-base64, and round-trips `String()`
- [x] run the tests and watch them fail
- [x] implement
- [x] run the tests, gofmt, vet: green
- [x] commit: `feat(v2): comment threads as facts, with ranges, markers and the --since cursor`

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

- ⚠️ `Export` takes a local `Reader` interface (one `GetBytes`) rather than `*gapi.Session`, the same deviation `internal/docs` and `internal/comments` made and for the same reason: this room may not name `net/http`, and a test of it must not have to.
- ➕ Four decisions the plan did not state, all written into the code's comments: a zip with no `word/document.xml` is refused by name, because a failed export is usually an HTML sign-in page and reading that as a document with no comments would report every thread as detached; a part that does not parse is refused naming the part, for the same reason; the join is on the comment's normalised words because `w:id` in the docx is the export's own numbering and carries no Drive comment id; and one exported comment answers for one thread, so two threads with the same words take two of them rather than both taking the first.
- ➕ Elements are matched by the WordprocessingML namespace URI, never by the `w:` prefix, and every part read is bounded by `MaxExportBytes`. A prefix is the document's choice, and an unbounded decompression is a memory limit somebody else sets.
- ⚠️ **`supportsAllDrives` is not on the export either, and the plan's URL above is wrong about it.** Measured against the live Drive v3 discovery document on 2026-09-06, `files.export` defines `fileId` and `mimeType` and nothing else; `supportsAllDrives` lives on `files.get`. So `ExportURL` carries `mimeType` alone, and `driveExportParams` in `go/internal/guard/params.go` had `supportsAllDrives` removed with it: an allowlist naming a parameter the method does not have permits a request nothing should send. Sending one the method does not define is one the server may reject, and it would take every `--witness` run with it. The global constraint at the top of this plan is about the files collection, which is where the live test's `files.create` still carries it.

- [x] write the failing tests: `Parse` on the helper-built docx finds two comments, one anchored with the right span and one detached; a docx without `word/comments.xml` parses to zero comments; a zip that is not a docx is refused by name; `Match` gives `anchored`, `detached` and `unmatched` across three threads and leaves the input slice untouched; `Export` over a fake wire asks for the docx mime and nothing else, and the URL it builds is one a real `guard.Policy` carries
- [x] run the tests and watch them fail
- [x] implement
- [x] run the tests, gofmt, vet: green
- [x] commit: `feat(v2): the docx export reader, and the witness match against comment threads`

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
  - `suggestions --md`: after a successful read, `frontmatter.Read` the file; refuse when the block's `document_id` is not the document read (writing another document's observation into this file is the wrong file); compute `Gone`; write the new snapshot with `frontmatter.Write` through a temp file in the same directory plus `os.Rename`, as `auth.Save` does; report `files_changed`. ⚠️ That write became its own package, `go/internal/atomicfile`, and `auth.Save` was moved onto it: two rooms doing the temp-file-and-rename dance slightly differently is how one of them loses a file's mode. The plan named neither the package nor the move. A read that failed writes nothing and the envelope says so.
  - A session factory behind a package variable, as `login` is, so the command tests stand in for the wire: `var openSession = func(p *guard.Policy) (*gapi.Session, error)`.
  - `usage` becomes `Commands: auth status, auth login, read, comments, suggestions`.
- PLAN.md M2 says `GrantInPlace`, `AllowCreateIn` and `Token.Refresh` are deleted if M2 lands without a production caller. `Token.Refresh` gains one in Task 3. `GrantInPlace` has none and nothing depends on it: delete it with its tests; M7 adds it back beside its caller. ⚠️ `AllowCreateIn` also has no production caller in M2, but the transport's create path (`checkParent`, `learnFromCreate`, the parent-check refusals, the upload-shape checks) is built on `createIn` and tested through it, so deleting it deletes the second door and a page of guard tests. This plan keeps it and says so here; Nail can flip that decision in review.
- The live test: gated on `GDOC_LIVE_TEST=1`, skipping with a message otherwise. It opens a policy with `AllowCreateIn` on the test folder, creates a Docs file there (`files.create` with `supportsAllDrives=true`; the guard learns the id at `LevelFull`), inserts two paragraphs and a heading, creates one Drive comment on a word, runs `read` and `comments --witness` through the real code path, asserts the heading is a `#` line and the comment's text is wrapped with `[[c:ID]]`, asserts one thread with `marker: none` and `witness` not `unmatched`, saves the export and the Docs read as redacted fixtures only when `GDOC_LIVE_RECORD=1`, and trashes the file, asserting `trashed: true` on the read-back. This is the one place M2 writes to Drive, on a document the run itself created.

- ⚠️ **The session seam is an interface, not `*gapi.Session`.** The plan wrote `var openSession = func(p *guard.Policy) (*gapi.Session, error)`. A stub for the concrete type has to build an `http.RoundTripper`, which means `cmd/gdoc` naming `net/http`, and the boundary test's import allowlist would then have to admit a room that only fakes the wire. So `read.go` declares a three-method `session` interface (`GetJSON`, `GetBytes`, `Warnings`) and `openSession` returns that. The factory is still one package variable, `login`'s pattern, and production still gets `gapi.Open(p, nil)`.
- ⚠️ **The live test reads and writes nothing to Drive.** The plan's version created a document in the test folder, wrote a comment into it and trashed it. A create and a comment are POSTs, `internal/live` would have to build them, and building a request means naming `net/http`: the import allowlist cannot admit a package whose only files are tests, because the disappearance half reads production files only and would then fail for the opposite reason. So `internal/live/live_test.go` is a read of a document the run names in `GDOC_LIVE_DOC_ID`, through `gapi.Open`, `docs.Parse`, `view.Text`, `comments.Fetch`/`Threads` and `docx.Export`/`Parse`/`Match`, with `GDOC_LIVE_RECORD=1` writing the two answers into `testdata/` for a person to redact. A live write belongs to M6, which has a production writer to run it through. Nail can flip that in review.

- [x] write the failing command tests over a stubbed `openSession` and fixtures: each command prints exactly one JSON object (`decodeOne`); `read` carries the golden text and no `structure` without the flag, and the tree with it; `comments` puts the policy's and the session's warnings on the envelope and reports an unplaced thread; `--since` with a bad cursor fails naming it; `--witness` sets `witness` per thread; `suggestions --md` on a fixture file with a matching `document_id` writes the snapshot, leaves every other line byte-identical, reports `gone_since_last_look` and `files_changed`; `suggestions --md` on a file whose `document_id` differs fails and leaves the file untouched; `suggestions --md` when the read fails leaves the file untouched; `documentID` accepts the two URL shapes and a bare id and refuses a short id quoting it; an unknown flag, a repeated flag and an extra argument each fail naming the offender; `usage` in the unknown-command error lists the five commands
- [x] write the failing guard test change: remove `TestGrantInPlaceUpgrades` and every `GrantInPlace` reference; the package must compile without the method
- [x] run the tests and watch them fail
- [x] implement `read.go`, wire `dispatch`, delete `GrantInPlace`
- [x] write the live test, run it with the variable unset and confirm it skips with a message; do not run it live in the unattended run
- [x] run the full suite with `-race`, gofmt, vet, `make build`: green
- [x] commit: `feat(v2): read, comments and suggestions on the envelope; GrantInPlace leaves until M7`

---

### Task 10: Verify acceptance criteria

**Files:**
- Modify: none expected. Fix whatever the checks below break.

**Interfaces:**
- Consumes: everything Tasks 1 to 9 produced. Produces: proof that PLAN.md's M2 description, as corrected 2026-09-06, holds: `documents.get` with `includeTabsContent=true`, tab detection, `read` as text with suggestions inline and anchors marked plus the structure on a flag, threads with real ranges and markers as facts, the opaque cursor, pending suggestions with stable ids and what stopped being pending, the docx reader, the versioned front-matter block, strict YAML.

- [x] verify every item in the Overview and in PLAN.md M2 is implemented, by reading each command's test and the fixture it runs on
- [x] verify no judgement leaked into Go: grep the new packages for `handled`, `accepted`, `rejected`, `matters`, `drift`; each hit is either a fact with a different name or a defect to remove
- [x] verify the reads are guard-bounded: a command test where the fake wire fails the test if anything reaches it proves a refused id never reaches the transport (mirror `acceptance_test.go`)
- [x] verify the boundary test in both directions on a scratch copy: a stray `net/http` in `internal/comments` fails it; removing the import from `internal/gapi/session.go` fails it; a second module in `go.mod` fails it
- [x] verify front-matter preservation on real files: `frontmatter.Write` of an unchanged block on a fixture with front matter, then `cmp` the bytes
- [x] run the full test suite: `cd go && go test -race ./...`
- [x] run the formatter check: `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- [x] run the vet check: `cd go && go vet ./...`
- [x] verify coverage: every exported function under `go/internal/` has a test (`cd go && go test ./... -cover`); add tests rather than lowering the bar
- [x] verify `make build` and `make dist` still produce the binaries with `CGO_ENABLED=0`, and record the binary size delta from the YAML module in this plan with a ➕ line
- [x] commit any fixes this task made

**What the checks found, and what they cost.**

➕ The guard-bounded check needed a second half and got one:
`go/internal/gapi/acceptance_test.go`, `internal/guard/acceptance_test.go`'s
`neverCalled` transport brought to the room where a session is what carries a
read. `cmd/gdoc`'s `TestAReadIsBoundedToTheOneDocumentItWasGiven` already put the
policy the command built through `Policy.Judge`, which proves the setup; it does
not prove a refused read never reaches the wire, because that test stubs the
session out entirely. The new test builds the policy the way `open()` does, one
`AllowFile` at `LevelSuggest`, and sends the URLs the three commands actually
build (`docs.URL`, `comments.ListURL`, `docx.ExportURL`, taken from the packages
rather than retyped) for another document, over a transport that fails the test
on contact. Both `GetJSON` and `GetBytes` are covered. It was watched failing:
adding the other id to the policy fires the transport on all four cases.

➕ One coverage hole, and a test rather than a lower bar. `view`'s `plain` had
zero coverage: it prints the second copy of a run carrying both an insertion and
a deletion id, text suggested and then suggested away, and no fixture had one.
`TestTextSuggestedAndThenSuggestedAwayPrintsTwiceAndIsMarkedOnce` covers it and
states the rule the code comment claims: the text prints twice, the comment
marker opens once, and it belongs to the first copy. After it, no function
anywhere under `go/internal/` has zero coverage, and the total is 93.3%.

⚠️ The boundary test does not count `import _ "net/http"`, and that is correct
rather than a hole: `httpRefs` skips a blank import with the comment "a blank
import cannot name anything", and a package that cannot name the type cannot
build a client out of it either. The first attempt at the scratch check used a
blank import and passed, which looked like a miss and was not. A real import
fails it, under the canonical name and under an alias, and both were watched.

➕ All three boundary directions were watched failing on a scratch copy of the
repo at `HEAD`: a used `net/http` in `internal/comments` fails
`TestNetHTTPStaysInItsRooms` naming the room and the four that may; deleting the
import from `internal/gapi/session.go` fails the same test's disappearance half;
a second `require` line fails `TestNoThirdPartyDependencies` twice, once for
`go.mod` and once for `go.sum`.

➕ Front-matter preservation was checked outside the suite, on files nobody wrote
for it. Every fixture carrying a readable `gdoc:` block round-trips
byte-identical under `cmp`, CRLF included. On five real files with no block at
all, two hub notes and this repo's `README.md`, `CLAUDE.md` and
`docs/v2/SPEC.md`, `Write` adds the block for exactly 83 bytes and every original
line survives in order.

➕ **The binary size delta.** `make build` and `make dist` both still produce
their binaries, `CGO_ENABLED=0`, and `file` reports Mach-O arm64 and PE32+ for
Windows. Two measurements, because the milestone's growth and the module's are
not the same number. go-yaml alone, measured against a build of `HEAD` with the
module dropped and its two calls stubbed:

| Target | Without go-yaml | With | Delta |
|---|---|---|---|
| darwin/arm64 | 11,092,994 | 12,132,722 | +1,039,728 (+9.4%) |
| darwin/amd64 | 11,858,512 | 12,980,240 | +1,121,728 (+9.5%) |
| windows/amd64 | 11,758,080 | 12,859,904 | +1,101,824 (+9.4%) |

All of M2 against M1's last binary (`47c638b^`), which is the number a person
downloading the tool sees: darwin/arm64 10,162,818 to 12,132,722, +1,969,904
(+19.4%). So roughly half the milestone's growth is the YAML module and half is
the eight new packages.

---

### Task 11: [Final] Update documentation

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `docs/v2/PLAN.md`

**Interfaces:**
- Produces: the repo's account of what now exists, so M3 starts from documentation that matches the tree.

- [x] update `README.md`: the three read commands, their arguments, the `read` text conventions (`{+ +}`, `{- -}`, `[s:ID]`, `[[c:ID]]`), the cursor, `--witness`, `--md`, and the `gdoc:` block a paired note carries
- [x] update `CLAUDE.md` under "v2 lives at `go/`": the facts-only rule and the test for it; the four rooms of the import allowlist and why `internal/gapi` is one; `allowedModules` holds go-yaml; the front-matter block version 1 and its rules (strict decode, byte-preserving write, snapshot written only after a successful read); `read`'s text conventions and the escaping rule; the cursor is opaque and dies with the session; `GrantInPlace` is gone until M7 and `AllowCreateIn` stayed, with the reason; pictures and drawings are in the backlog
- [x] update `docs/v2/PLAN.md`: mark M2 done with the date, record what it left for M3 (the Docs `comments` range shape is measured from the live fixture; `AllowCreateIn` has its first production caller in M6; the review skill reads `comments` and judges), and the binary size delta
- [x] run the full test suite one more time: `cd go && go test -race ./...`
- [x] commit the documentation updates: `docs: M2 lands, the read commands and the front-matter block written down`
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
