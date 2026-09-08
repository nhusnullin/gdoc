# gdoc v2 Milestone 6: Publish

## Overview

M5 taught the binary to write a house-style docx and stop. M6 puts that file into Drive as a Google Doc, records the pairing in the note, and takes the document back out again if the note could not be written.

`gdoc publish --md note.md --folder-id FOLDER` renders the note the way `build` renders it, uploads the bytes with conversion into the one folder the run was given, verifies the result by reading it back, and writes `document_id`, `folder_id` and a publish record into the note's `gdoc:` block. If that write fails after a successful upload, the run trashes the document it just made and confirms the trash. Only if the rollback also fails does the error carry the live id and the recovery steps. That failure policy is PLAN.md M6's, not SPEC's: SPEC's `publish` section states no rollback rule at all.

The upload is the thing three other pieces have been waiting for. The guard refuses a multipart create today, on purpose, because the body opens with a MIME boundary and `checkParent` cannot read the parents out of it. `gapi` has no multipart write at all. And `TestLiveDrift`, the gate that means something because Google's import is part of the result, has never run. M6 opens the first, adds the second, and runs the third.

Decisions Nail took on 2026-09-08, before this plan:

- **Publish is the only writer of the `gdoc:` block.** v2's reader keeps refusing v1's plain `gdoc: <id>` string. A note v1 published gets its line rewritten by hand once, or is republished by v2. No schema-0 shape enters the reader.
- **The publish record drops `revision_id`.** It records `at`, the `title` that went on the cover, and the `house` source. A Drive revision id would need a route the guard does not carry, for a field nothing reads.
- **M6 carries the live drift gate, not just the upload.** The `drift.Doc` style-resolution hole is settled here, with the fixture pins, because the backlog item says it has to be settled with the upload rather than after it.
- **One command.** `publish` renders and uploads in one run. `build` stays as the offline command, and it already writes the exact bytes `publish` uploads, because the render is deterministic.

Spec: `docs/v2/SPEC.md` ("`publish`", acceptance items 2 and 3). Master plan: `docs/v2/PLAN.md`, section M6, which holds the rollback policy. Predecessor: `docs/plans/completed/2026-09-08-gdoc-v2-m5-generator-parity.md`.

## Context (from discovery)

- **What exists.** `internal/render`, `internal/body`, `internal/cover` and `internal/house` turn a note into docx bytes with no network at all (M5). `internal/probe` already creates a document in a granted folder, reads it back and trashes it with a confirmation, over a plain JSON create (M3). `internal/docx` reads a docx export, through `ExportURL`, `Export` and `Parse` (M2). `internal/frontmatter` writes the `gdoc:` block byte-preservingly, verifies its own render, and refuses a block it would break (M2). `internal/atomicfile` does the temp-file-and-rename write.
- **The guard's create path.** `filesCollection` already strips a `/upload` prefix, so the policy's path grammar carries the upload route. `driveCreateParams` already names `uploadType` and `upload_protocol`, and `uploadShapes` already permits `multipart` and refuses `media`, `raw` and `resumable`. What is missing is in `transport.checkParent`, which `json.Unmarshal`s the peeked body and therefore refuses every multipart create. The refusal is deliberate and its own comment names M6.
- **⚠️ The guard and Drive pick the body parser from different signals, and Task 1 owns it.** Drive decides from `uploadType` / `upload_protocol` and the `/upload` path; the guard decides from nothing at all and just tries JSON. `TestUploadCreateAlsoTeachesThePolicy` (`internal/guard/transport_test.go:137`) shows the guard **carrying** `POST /upload/drive/v3/files?uploadType=multipart` with `Content-Type: application/json` and a plain JSON body, and learning the id at `LevelFull`. `checkUploadShape` also returns nil when no upload parameter is present, so a multipart body on the `/upload` path with no `uploadType` is the mirror case. **Neither mismatch is exploitable today**, and the reason is luck rather than design: Drive refuses the first, so no 2xx comes back and nothing is learned, and the guard's own JSON parse refuses the second. Task 1 is what makes it matter: the moment the guard learns to parse multipart, a parser chosen from the wrong signal is a body judged one way and sent another, which is the class CLAUDE.md already documents for `Authorization` beside `authorization`. So the three-signal rule lands in the same change as the parse, never after it.
- **The body peek.** `transport` reads at most `maxPeek` bytes of the request body, restores them in front of the rest, and drops the replay when the peek filled exactly. A docx is megabytes and the metadata part is first, so the metadata falls inside the peek. `TestACreateBiggerThanThePeekIsRefused` already pins the refusal for a body that does not.
- **`gapi`'s write surface** is `PostJSON` and `PatchJSON`, both through `writeJSON` into `bodyRequest`, which **hardcodes** `Content-Type: application/json`. `send` rebuilds the request through the `requestFor` closure on the 401 retry, so anything generated inside that closure differs between the judged attempt and the retry.
- **`frontmatter.Published`** is `{At, RevisionID}` today. No production code writes or reads `RevisionID`. The fixture `internal/frontmatter/testdata/full.md:9` carries one, and `frontmatter_test.go:68` asserts it, so Task 3 changes both.
- **`Block.Validate`** requires a valid `document_id`. Any block that reads at all therefore carries one, which is why publish has no "read the folder from the note" path: a note whose block reads is a note publish refuses.
- **`freshNote`** (`cmd/gdoc/write.go:412`) refuses a block that has **gone** and a block naming another document. Publish needs the inverse rule, so it does not reuse it.
- **Backlog items this milestone settles and deletes**: `docs/backlog/v1-frontmatter-migration.md` (by Nail's decision, not by code) and `docs/backlog/drift-fromdoc-style-fallback.md` (by Task 7).

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own commit.
- **CRITICAL: every task MUST include new or updated tests**, success and failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** Each task carries that gate as its last checkbox. `make test` runs `-race`; keep it there.
- **CRITICAL: the guard task is adversarial or it is nothing.** Task 1 widens the one room that decides what reaches Drive. Every test there is written as an attack first. Failing closed is the right direction to be wrong in.
- **CRITICAL: the guard must judge the body the way Drive parses it.** Three signals decide a multipart create: the `/upload` path, the `uploadType` / `upload_protocol` value, and the media type. They must agree, and every disagreement is refused by name. A guard that reads JSON where Drive reads multipart, or the reverse, is judging a request it is not sending.
- **CRITICAL: facts only in Go.** `publish` reports what it uploaded, what it verified and what it wrote. No field says the document is good. Nail opens it.
- **CRITICAL: not knowing never resolves to keeping the document.** A note that could not be written means the upload is rolled back. A rollback that could not be confirmed is reported with the live id, never as a clean run.
- **CRITICAL: `cmd/gdoc` owns the note's bytes.** `internal/publish` reads and writes no file, the way `propose.Record` hands its results back and `cmd/gdoc` does the I/O. That is what keeps the package testable on a fake session.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces, code comments, commit messages and the docs included.

## Testing Strategy

- **Unit, `internal/guard`** (`transport_test.go`, `params_test.go`): the three-signal agreement and the multipart part parse, attacks first, then the carry. Two existing tests change: `TestAMultipartCreateIsRefusedForNow` inverts, and `TestUploadCreateAlsoTeachesThePolicy` becomes a refusal.
- **Unit, `internal/gapi`** (`session_test.go`): the multipart request's content type, part order and bearer; the same bytes on the 401 retry; a 2xx answer that fails to decode marked `Sent()`.
- **Unit, `internal/publish`** (`publish_test.go`): over a fake `Session`, the way `probe`, `propose` and `withdraw` are tested. Eleven cases, listed in Task 4.
- **Unit, `internal/frontmatter`** (`frontmatter_test.go`, `schema_test.go`): the new `published` shape round-trips, and `revision_id` is now refused by name.
- **Unit, `internal/drift`** (`doc_test.go`): fixture cases where a run inherits rather than states, and a pin naming the items the fixture is expected to answer.
- **Unit, `cmd/gdoc`** (`publish_test.go`): strict argument parsing, the envelope on each failure path, and the shared render function used by both commands.
- **Live, opt-in behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`** (`internal/live`): `TestLivePublish` and `TestLiveDrift`.
- **Boundary and modules**: `allowedModules` does not change; M6 adds no dependency. The import and builder allowlists do not change: `internal/publish` takes a session as a parameter and builds no client.
- Coverage standard: every exported function under `go/internal/` has a test; `internal/publish` at or above 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build` and `make dist`, with the per-platform binary sizes recorded before and after in Task 9

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

`gdoc publish --md note.md --folder-id FOLDER` does eight things in order:

1. Read the note. `cover` reads the author's keys, `frontmatter` reads the `gdoc:` block. A note that already carries a block is refused before anything else happens: publish runs once per document, and there is no republish.
2. Render, through the same function `build` calls, so the two commands cannot drift.
3. Open a `guard.Policy` whose only door is `AllowCreateIn(folderID)`. No file is in the reachable set at the start of the run. The new document's id is learned from the create the guard itself carried.
4. Upload multipart: a JSON metadata part naming the folder, the title and `mimeType: application/vnd.google-apps.document`, then the docx bytes.
5. Verify. Read the new document through `documents.get` for its title and its tab count, and export it as docx through `internal/docx` to confirm the export reads as one. Both are reported as facts.
6. Re-read the note off disk, because seconds to tens of seconds of network have passed and these notes live in a synced vault.
7. Write the block: `document_id`, `folder_id`, `published: {at, title, house}`.
8. If step 7 failed, trash the document and confirm the trash.

Key design decisions and why:

- **The guard parses the body the way Drive picks its parser.** `uploadType=multipart` or `upload_protocol=multipart` under the `/upload` path, with a `multipart/related` media type: all three, or the request is refused naming the one that disagrees. Today the two mismatches fail closed by accident; teaching the guard to parse multipart is what would turn them into a body judged one way and sent another, so the rule lands with the parse.
- **The boundary comes from the header, parsed with `mime.ParseMediaType`, never from the body.** Scanning the body for something that looks like a boundary would let the file's own bytes move where the guard thinks the metadata ends. The header is the value Google's own parser uses. `mime.ParseMediaType` also refuses a `boundary` given twice, which a hand-rolled split would not, and it reads `application/json; charset=UTF-8`, which is the media type Google's own documentation puts on the metadata part.
- **The guard reads the first part and nothing else.** Its media type must be JSON, and its body goes to the duplicate-key and parents check the plain create already runs. Everything after that part is opaque bytes.
- **A metadata part outside the peek is a refusal.** The peek is a fixed cap and the metadata part is small and first, so this cannot happen to a request gdoc builds. It can happen to a request somebody adds later, and the refusal is how they find out.
- **The multipart body is built once per call, outside the request closure.** `gapi.send` rebuilds the request on the 401 retry, so a boundary generated inside the closure would send different bytes than the attempt the guard judged.
- **Publish's re-read rule is the inverse of `freshNote`'s.** `freshNote` guards a paired note and refuses a block that vanished. Publish starts from an unpaired note, so what it refuses is a block that has **appeared**: another run published this note while this one was uploading, and this run must roll back rather than overwrite the pairing. It also refuses a note whose body changed under the render, because the document in Drive is then a render of bytes the note no longer holds. That is a rollback, not a warning.
- **Rollback first, and the report says which.** A publish that uploaded and could not record the pairing leaves a document nobody knows about, and the next run would publish a second one. So the document goes, and `rolled_back: true` says so. A trash that could not be confirmed is `rolled_back: false` with the live id and the steps, exactly as `probe` names a document it left behind.
- **The publish record is three facts.** `at`, `title` and `house`. When the document was made, what went on its cover, and whether the style was the embedded one or a file under review.
- **`internal/publish` takes a `Session` interface** and does no file I/O, so it is testable on a fake and `net/http` stays in its four rooms.

## Technical Details

**Principles.** Serves: 1 (one binary, no new dependency), 2 (the note is the root, and the run writes into it and nowhere else), 3 (the run's only door is one folder, and the new id is learned from the create the guard carried), 4 (the report carries facts and the warnings; whether the document is right is Nail's).

**Global constraints:**

- Exactly one JSON object on stdout, exit 0 if and only if `ok`. No prompting, no stdin.
- `--md` is required and must exist. `--folder-id` is required and must be a Drive id. There is no fallback to the note's `folder_id`: a note whose block reads at all carries a `document_id`, and publish refuses that note in step 1.
- `--house PATH` replaces the embedded style for one run, as in `build`, and `data.house` names `"embedded"` or the path.
- Strict argument parsing: an unknown flag, a repeated flag, a missing value, an empty value and an extra positional each fail naming the offender.

**SPEC acceptance item 3** ("a publish produces a document with the positioned logo, a live contents list and footer page numbers, verified by export") is answered by **Task 8's `TestLiveDrift`**, whose item list already reads the logo, the TOC field and the four header and footer segments through the Docs API. Publish's own three checks are read-back facts, not the acceptance, and the plan says so rather than duplicating `drift`'s readers inside `publish`.

**`gdoc publish` output:**

```jsonc
{
  "document_id": "1AbC...", "folder_id": "0BxY...",
  "url": "https://docs.google.com/document/d/1AbC.../edit",
  "title": "Supplier Register Policy",
  "house": "embedded", "bytes": 48213,
  "verified": true,
  "checks": { "read_back": true, "one_tab": true, "docx_export": true },
  "files_changed": ["/abs/path/note.md"],
  "body": { "paragraphs": 41, "headings": 9, "lists": 3, "tables": 2, "images": 1 },
  "warnings": []
}
```

Three failure shapes, and each says something different:

- A failed front-matter write with a good rollback: `ok: false`, `document_id` empty, `rolled_back: true`, and the reason.
- A failed front-matter write with a failed rollback: `ok: false`, the live id, `rolled_back: false`, and a warning naming the document to trash by hand.
- A create Drive accepted whose answer could not be read: there is no id to verify, to record or to trash, so the error names the **folder** and says a document may be in it, the way `probe.create` does. The guard learned nothing either, so it would refuse the trash in any case.

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): the guard, `gapi`, the record shape, the publish package, the command, the v1 decision, the drift resolution chain, the two live tests, the docs.
- **Post-Completion** (no checkboxes): Nail's live publish run, the live drift table read by a person, the binary-size delta, and the two acceptance runs M4 and M5 left open.

## Implementation Steps

---

### Task 1: the guard judges a multipart create the way Drive parses one

- [x] Write the attacks first, in `go/internal/guard/transport_test.go` and `params_test.go`, each a refusal naming what disagreed or what could not be read:
  - **the three signals disagreeing**: `uploadType=multipart` with a JSON media type and a JSON body (this is `TestUploadCreateAlsoTeachesThePolicy` today, carried; it must become a refusal); a `multipart/related` body with no upload parameter; a `multipart/related` body on the plain `/drive/v3/files` path
  - a body whose boundary never appears inside the peek
  - a `Content-Type` with no `boundary` parameter, one whose boundary the body does not use, and one giving `boundary` twice
  - a first part whose media type is not JSON, and one that is not valid JSON
  - a first part carrying **two `Content-Type` headers**, and one carrying **`Content-Transfer-Encoding`**: the guard reads raw bytes and Drive would decode, so neither may be guessed at
  - a first part naming `parents` twice, one naming a folder other than the granted one, and one naming two folders
  - a boundary string that also appears inside the file bytes, proving the guard reads the first part only and does not scan
  - a multipart create with no `AllowCreateIn` grant at all
- [x] Then the carry, in both directions: a well-formed multipart create naming exactly the granted folder is carried and `learnFromCreate` puts the new id at `LevelFull`; a plain JSON create with no upload parameter still works exactly as it does today.
- [x] Invert `TestAMultipartCreateIsRefusedForNow` and rewrite its comment: the pin it left for M6 is now spent.
- [x] Implement: pick the parser from the three agreeing signals, take the boundary through `mime.ParseMediaType`, read the first part with `mime/multipart`, require its media type to be JSON, and hand its body to the existing duplicate-key and parents check.
- [x] Update `checkParent`'s and `uploadShapes`' doc comments, which both name M6 as the milestone that would open this.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the guard judges a multipart create the way Drive parses one"`

### Task 2: `gapi` gains a multipart write

- [x] Test first in `go/internal/gapi/session_test.go`: the request carries `multipart/related` with its boundary, the JSON metadata part first and the file part second, the bearer, and the caller's context; the 401 retry sends **the same bytes and the same boundary**; a 2xx answer that does not decode is marked `Sent()`, and a guard refusal is not.
- [x] Implement `Session.PostMultipart(ctx, rawURL string, meta any, part []byte, partType string, into any) error`. The whole body and its boundary are built once, before the closure, from `crypto/rand`; the closure only wraps those bytes in a request.
- [x] Give `bodyRequest` a content-type parameter rather than reusing it as it is, since it hardcodes `application/json` today. `PostJSON` and `PatchJSON` pass the same value they set now.
- [x] The metadata part is encoded with `encoding/json`, so nothing a note holds reaches the wire as text somebody formatted.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): a multipart write in gapi"`

### Task 3: the publish record's shape

- [x] Test first in `go/internal/frontmatter/`: a block whose `published` carries `at`, `title` and `house` round-trips byte-preservingly; a block carrying `revision_id` is refused by name under `yaml.Strict()`; `Validate` refuses a `published` missing `at` or `title`.
- [x] Change `Published` to `{At time.Time, Title string, House string}`.
- [x] Update the fixture `internal/frontmatter/testdata/full.md:9` and the assertion at `frontmatter_test.go:68`, which carry `revision_id` today. Without this the fixture stops parsing and takes the round-trip tests with it.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the publish record is when, what title and which house"`

### Task 4: `internal/publish`

- [x] Test first in `go/internal/publish/publish_test.go`, over a fake `Session`: the happy path; an upload the guard refused; an upload Drive rejected; an upload whose answer could not be read (no id to name, so the folder is named); the read-back that failed; a document that came back with two tabs; an export that did not read as a docx; a rollback that succeeded; a rollback that failed; and the two re-read refusals from Task 4's caller side, exercised through the returned report.
- [x] Implement `publish.Run(ctx, s Session, o Options) (Report, error)`: upload, verify, and hand the report back. It reads and writes no file. The caller does the note I/O and calls `publish.Rollback` when the write failed.
- [x] Reuse `internal/docx`'s `ExportURL`, `Export` and `Parse` for the export check rather than a second export path, and `probe`'s trash-and-confirm shape for the rollback. Extract the trash helper if that is cleaner, and record the move in this plan.
- [x] `verified` is the three checks together. Fewer than three is `ok: true` with `verified: false` and the route named, exactly as `propose` reports: a document that exists is a document that exists.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): publish uploads with conversion and verifies the result"`

**The trash helper was extracted, into `go/internal/drive`.** It holds
`FileURL`, `TrashedURL` and `Trash`, which is the PATCH, the confirming read and
the rule that the read is what is believed. `probe.trash` now calls it and keeps
only what the document is and what a failure costs, and `publish.Rollback` does
the same on the other side. The reason it is worth a package rather than a
second copy: an unconfirmed trash counts as a failure in both callers, and two
copies of that rule are two chances for one of them to start reporting a
document as gone that is still there. `probe`'s three warning sentences are one
sentence now, carrying `drive.Trash`'s own reason, so the three failures still
read differently.

`publish.Run` reports `title` as **what the read-back carried**, not what the
upload asked for, and a disagreement between the two is a warning naming both.
It is not a fourth check: Drive takes the name from the metadata part, so the
usual answer is that they match and the warning says nothing.

Coverage: `internal/publish` and `internal/drive` are both at 100% of
statements, against the 80% the plan asks for.

### Task 5: the `publish` command, and the re-read rule that is publish's own

- [x] Test first in `go/cmd/gdoc/publish_test.go`: strict argument parsing in every shape; a note that already carries a `gdoc:` block, refused before anything leaves the machine; a note whose block **appeared** during the upload, rolled back; a note whose **body changed** under the render, rolled back; a note that could not be read again, rolled back; and the envelope on each.
- [x] Implement `cmdPublish` in `go/cmd/gdoc/publish.go`: render, open the policy with `AllowCreateIn` and nothing else, open the session, call `publish.Run`, re-read the note, write the block, and roll back on any failure of the last two.
- [x] Extract the render step `build` and `publish` share into one function returning one struct, carrying `title`, `house`, `bytes` and the body counts, so the two commands cannot print overlapping shapes that drift. Carry over `cmdBuild`'s `readHouse`, `pictures`, `notAnInput` and `absolute` helpers whole rather than re-deriving them.
- [x] `dispatch` gains `publish`, and the usage string with it.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): gdoc publish"`

**The shared render is `noteSource` and `renderNote`, both in `build.go`.**
`renderNote` returns one `noteDocx`: the bytes, the cover title, the running
head, the style's name, the walker's counts, the pictures the note names and the
warnings. `cmdBuild` writes the bytes and `cmdPublish` uploads them, and
`TestBuildAndPublishRenderTheSameBytes` compares the file `build` wrote against
the part `publish` uploaded, so the two cannot drift. The four helpers were
carried over whole: `readHouse` is called inside `renderNote`, and `pictures`,
`notAnInput` and `absolute` stayed in `cmdBuild`, because publish has no `--out`
and so has no file to alias.

`noteSource` is separate from `renderNote` for publish's sake. Publish needs the
note's bytes twice over: the `gdoc:` block it refuses to republish is read out
of them before the render, and the same bytes are what `pair` compares the
re-read against.

**The re-read refuses four things, and each one is a rollback.** A note that
could not be read again, one whose front matter no longer parses, one whose
block has **appeared**, and one whose bytes changed at all. The third is the
inverse of `freshNote`'s rule, which is what the plan called for. The fourth is
written as the whole file rather than the body alone: the author's own front
matter feeds the cover, so a title edited during the upload is as stale a render
as an edited paragraph, and there is no honest way to call one of them a change
and the other not.

**`rolled_back` is a `*bool`, absent on a run that recorded the pairing.** With
a plain bool and `omitempty` the failed-rollback case, which is the run where
the live id matters most, would have printed nothing at all; without
`omitempty` every clean publish would say `rolled_back: false` about a rollback
nobody tried. A rollback that held clears `document_id` and `url`, because
naming a document that has gone sends somebody to look for it.

**`title` on the envelope is the read-back's and `published.title` in the note
is the cover's.** They are two different facts: what Drive named the file, and
what went on the cover. `publish.Run` already warns when they disagree, and a
read-back that did not happen leaves the envelope's field out rather than
filling it in with the title that was asked for.

`session` in `cmd/gdoc` gained `PostMultipart`, so it satisfies
`publish.Session`. Both test fakes gained it too: `fakeWire` records the
metadata part as the call's body and keeps the file part beside it, and
`fakeSession` refuses it by name, the way it already refuses a POST from a read
command.

Coverage: `cmd/gdoc` is at 87.7% of statements.

### Task 6: the v1 pairing decision, written down

- [x] Record Nail's decision of 2026-09-08 in `docs/v2/DECISIONS.md`: v2's reader keeps refusing v1's plain `gdoc: <id>` string, publish is the only writer of the block, and a note v1 published is rewritten by hand once or republished by v2.
- [x] Check the refusal `frontmatter.Read` gives on a v1 string names the shape and says what to do. Fix it if it does not, with a test.
- [x] Delete `docs/backlog/v1-frontmatter-migration.md`, which the decision settles.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "docs: the v1 pairing decision, and publish is the only writer"`

### Task 7: `drift.Doc` resolves a style, and the fixture says what it answers

- [x] Test first in `go/internal/drift/doc_test.go`: fixture cases where a run inherits rather than states its size, font and colour, and the resolved value comes back.
- [x] Give `Doc` the resolution chain in Google's own vocabulary: `textRun.textStyle`, then `paragraph.paragraphStyle.namedStyleType`, then `namedStyles[type].textStyle`, then `namedStyles["NORMAL_TEXT"]`. The Docs API has no `docDefaults`. Normalise an absent colour the way `Docx.Style` does.
- [x] Pin the set of items the fixture is expected to answer, the way `unstated` pins the docx half, so a row that starts reading nil fails rather than passing as IDENTICAL against another nil.
- [x] State in the test file that this touches the Docs half only: the offline gate reads two docx files, so its 169 rows do not move and nothing is re-baselined.
- [x] Delete `docs/backlog/drift-fromdoc-style-fallback.md`.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "fix(v2): the drift Docs reader resolves the style behind a value"`

**The chain is one function, `Doc.resolve`, and `Style` is its last two links.**
A `layer` is one `{textStyle, paragraphStyle}` pair, and `fold` applies them
outermost first, so the nearer one wins. A property is taken only when the
message carries it: an absent `bold` on a run is that run inheriting, so writing
false there would clear a weight the named style states. That presence rule is
why `truth` is gone, and it is what the old reader could not express.

**Paragraph properties inherit too, not just text.** Docs documents both, and
the docx half already folds `pPr` down the `basedOn` chain, so a Doc half that
resolved only the run would have `HEADING_1 lineSpacing` reading nil against a
docx half reading 115.

**A style the answer does not carry is still not found.** Inheriting NORMAL_TEXT
under an absent style's name would invent a style Docs never sent, and the docx
half answers `Style{}` for a style that is in no file.

**`firstRunStyle` became a method,** because it now needs the document to reach
`namedStyles`. `Segment` is its only caller.

**The fixture's two body paragraphs now state nothing on their runs,** which is
what a converted document looks like, so `TestFromDocReadsEveryItem` walks the
chain rather than around it. `body text size` reads 11 rather than the 12 the
run used to state, which is the house body size.

**The pin is `docsSilent`, 71 names.** Four reasons, each a fact about the
fixture: a value the style leaves to the reader's default, the six named styles
the fixture does not carry, the two tables and two heading levels it does not
carry, and the contents field instruction Docs never sends. It fails in both
directions. Against the reader before this change it names exactly the three
rows the backlog item named, `body text size`, `body font` and `body H1 run
colour`, plus the two `HEADING_1` and two `TITLE` rows the named-style link
fixes.

Nothing was re-baselined: the offline gate reads two docx files, so its 169 rows
did not move.

### Task 8: the two live tests

- [x] `TestLivePublish` in `go/internal/live`, behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`: publish a temp note into the test folder, assert the read-back, the title, the one tab and the note's written block, then trash the document.
- [x] `TestLiveDrift` beside it: build `03-policy.md`, upload it and the master with conversion, read both through the Docs API, run the 169-item list, print the table, fail on any verdict that is not IDENTICAL, CLOSE or a named known difference, and trash both documents.
- [x] Assert the three rows that answer SPEC acceptance item 3 by name: the positioned logo, the TOC field, and the footer page numbers.
- [x] Record what the live table actually says in this plan under Post-Completion, including any row that has to join `drift.Known` with its reason. A row joining `Known` is a decision Nail takes, not a test somebody loosens. **(not automatable here: the table only exists after a live run, which creates real documents in Nail's Drive and is Nail's decision each time. Post-Completion carries the empty row waiting for it.)**
- [x] `cd go && go test -race ./...` passes, and the live pair runs clean on Nail's machine. **(the suite passes, and both new tests skip without the two variables. The live half is Nail's run: the unattended run sets neither variable.)**
- [x] `git commit -m "test(v2): the live publish and the live drift gate"`

**The live drift read is not `docs.URL`, and that is the one thing this task
discovered.** `docs.URL` asks for `includeTabsContent=true`, which moves the
content into `tabs[]` and leaves the legacy `body`, `headers` and `footers`
empty. Those legacy fields are exactly what `drift.Doc` reads, and they carry
the first tab, which is the whole of a document converted from one docx. Read
through `docs.URL` the gate would answer nil for every body, header and footer
row and pass on a document it never looked inside. So `readForDrift` spells a
bare `documents.get` with no query at all, which the guard's `docsReadParams`
allowlist carries because an empty query names no parameter.

➕ `TestTheLiveFixturesRenderWithNoNetwork`, in the same file and in `make test`.
Both live tests render a note before they reach Drive. A note that stopped
rendering, or a fixture path that moved, would otherwise be found by Nail in
the middle of a live run rather than by the suite, so this renders both notes,
checks each makes a publishable upload, and opens the master. It asks for
neither live variable, because there is nothing in it to protect.

### Task 9: documentation and the size delta

- [ ] CLAUDE.md: `publish` under the write commands; the multipart create and the three-signal rule in the guard section; the changed publish record; the live drift gate moving from "does not exist yet" to what it measures.
- [ ] README: publishing a note.
- [ ] `docs/v2/PLAN.md`: the M6 "Landed" paragraph, the "What M6 leaves for M7" list, the binary-size table against `e77d9f3`, and acceptance items 2 and 3 marked with the test that answers each.
- [ ] Note that `publish` has no skill caller: it is Nail-invoked, and wiring it into a skill is M9's with the install story.
- [ ] Move this plan to `docs/plans/completed/`.
- [ ] `git commit -m "docs: M6 landed"`

## Post-Completion

- Nail publishes a real note and opens the document in Drive.
- Nail reads the live drift table and decides on any row that has to join `drift.Known`. The table is printed by `TestLiveDrift` under `-v`, with the summary line in front of it, and it goes here when the run has happened:

  ```
  (the live table, built vs master: recorded after the first live run)
  ```

- The binary-size delta per platform, recorded in PLAN.md.
- Still outstanding from earlier milestones, and Nail's: M4's live session with a second account commenting, and M5's built docx opened in Word beside a v1-published copy.
