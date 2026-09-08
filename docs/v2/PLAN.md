# gdoc v2 master plan

2026-08-29. The build order for [SPEC.md](SPEC.md). Nine milestones, each one
producing working, testable software. Detailed task-by-task plans live in
`docs/plans/` and are written when a milestone starts, so each one
is written against the code that actually exists by then. Milestones 1 to 4
are written, and all four are done.

## Principles

Serves: 1 (a single static Go binary is the whole point of the ordering: the
tool is usable from milestone 3), 3 (the guard is milestone 1, complete, before any
command can reach Drive), 4 (the review loop ships before the generator,
because reading and answering is the daily need now that v1 is retired).

Strains: none.

## Standing facts

- Go 1.27, already installed. Module `gdoc` at `go/` in this repo.
- Dependencies: `beevik/etree v1.7.1`, `yuin/goldmark v1.8.5`, and
  `goccy/go-yaml v1.19.2` (front matter and `house.yaml`; reason written in
  SPEC.md; zero transitive modules). A fourth needs its reason written into
  SPEC.md first; the open candidate is `sergi/go-diff` at milestone 8.
- `goccy/go-yaml` went into `go.mod` at M2, and `beevik/etree` and
  `yuin/goldmark` at M5, which is all three SPEC.md agreed and all there are.
  `allowedModules` in `TestNoThirdPartyDependencies` names each with its reason,
  and refuses every other require line and every other `go.sum` entry. The
  milestone that first needs another adds its path to that map, and nothing
  else: it does not delete the test and it does not widen it to accept whatever
  `go.mod` says.
- No `golang.org/x/oauth2`: the token refresh is a single POST to Google's
  token endpoint and is hand-rolled on the standard library.
- v1 is retired as of 2026-08-29, Nail's call. The Python code stays in the
  repo untouched as reference until v2 replaces its last use, then its removal
  is a separate decision. Nothing maintains it.
- Platforms: darwin/arm64, darwin/amd64, windows/amd64. No linux.
- Live tests split by direction, corrected at M2. A live **read** may be pointed
  at a document the run names and reads nothing else: `internal/live` skips
  unless `GDOC_LIVE_TEST=1` and then needs `GDOC_LIVE_DOC_ID`, with no default.
  A live **write** follows v1's convention: create documents only in the Drive
  test folder, and never touch a document gdoc did not create. M3 turned out to
  be the first milestone with a production writer, not M6: the capability probe
  creates a document. The write test is behind a second variable,
  `GDOC_LIVE_WRITE=1`, on top of `GDOC_LIVE_TEST=1`, because a live write is a
  different decision from a live read and is made on purpose each time.

## The milestones

### M1. Foundation: the binary exists and can refuse (done 2026-08-29)

Module scaffold at `go/`, the JSON output envelope, config paths decided per
platform (`~/.config/gdoc-agent` on macOS, `%AppData%\gdoc-agent` on Windows),
and the guard as the complete network policy, not just a RoundTripper: an
allowlist of exact Google hosts, redirects capped and re-judged so one off the
host allowlist is refused, the id set
with write levels, a create-child capability for the one folder a command was
given, narrowly defined non-file exemptions for the token endpoint, and
response learning from creates. Enforced by a `go/ast` import-boundary test:
only the guard's internal package may import `net/http`, checked across every
module file including tests and build-tagged files. Adversarial URL and body
tests run over a fake transport. OAuth: token load (v1's file format, so no new
login needed), hand-rolled refresh with crash-safe rewrite, `auth login` as a
loopback-and-PKCE desktop flow (when the browser cannot be opened, the
authorization URL goes to stderr; stdout stays reserved for the one JSON
object), `auth status`. Cross-builds for all three
platforms run from this milestone on, in the Makefile, so portability is never
discovered late. Acceptance: spec item 1.
Detailed plan: `docs/plans/2026-08-29-gdoc-v2-m1-foundation.md`.

Landed 2026-08-29, with three things worth carrying forward. The import
allowlist has three rooms rather than one, because `internal/auth` takes the
guard's client as a parameter, so a second check was added: only
`internal/guard` may build a client or dial, and serving an `http.Server` is not
building. The guard refuses percent-encoded paths and dot segments outright,
because it would otherwise read a URL differently from the way the transport
sends it. And no browser is opened at all, so v2 needs no `os/exec` anywhere.
The result is documented in CLAUDE.md under "v2 lives at `go/`".

### M2. Reading, and the honest witness (done 2026-09-06)

Three things M1 left for this milestone to pick up. `Policy.AllowFile`,
`AllowCreateIn` and `GrantInPlace` have no production caller yet: if M2 lands
without one, delete it rather than carry it. `Token.Refresh` is the same, and M2
is the milestone that needs it. And `Policy.Warnings()` collects what the guard
could not do quietly, so whatever command M2 adds should put those on the
envelope rather than dropping them.

`documents.get` with `includeTabsContent=true`, tab detection, comment threads
with real ranges and `ai` markers, the opaque `--since` cursor (held by the
caller, dies with the session), pending suggestions with stable ids,
resolved-since-last-look detection. Plus two things the later milestones
depend on: the **docx export reader** (unzip, read `word/comments.xml` and the
`commentRangeStart`/`End` markers), because the export is the honest witness
the write milestones verify against; and the **versioned `gdoc:` front-matter
schema**, designed here and reviewed by Nail, because publish must write it
crash-safely from its first version: ids, publish record, schema version, plus
the observation state M3 and M4 need across sessions: the last-seen
pending-suggestion snapshot and the provenance of gdoc's own proposals,
committed only after a successful suggestions read. Everything outside the
`gdoc:` key is preserved byte-identically. The YAML dependency
(`goccy/go-yaml`, see SPEC.md) enters here, with strict decoding: duplicate
and unknown keys rejected, exactly one document, required values validated
rather than defaulting to useful-looking zeros.

**Corrected 2026-09-06, during the M2 re-cut.** Two things this paragraph
used to promise are withdrawn, and one is added. Withdrawn: the per-section
fingerprint pairs (SPEC.md "The diff" says why), and any rule in the binary
that decides whether a comment is answered or a gone suggestion was accepted.
The binary reports facts; the skills judge. Added: `read`, the document as
text the skill reads, with pending suggestions inline and comment anchors
marked, plus the raw structure with character indexes on a flag. It is the
piece review (M3) and alignment (M8) both stand on. Reading pictures and
drawings is in `docs/backlog/`.

**Landed 2026-09-06.** Three commands, `read`, `comments` and `suggestions`,
each opening the guard with exactly the one document it was given.
`Policy.AllowFile`, `Token.Refresh` and `Policy.Warnings()` have production
callers now. `GrantInPlace` had none and was deleted with its tests, as this
milestone asked; M7 adds it back beside its caller. `AllowCreateIn` stayed,
because the transport's whole create path reads it and M6 needs that path.
`internal/gapi` is the fourth room in the import allowlist and builds no client
of its own. `goccy/go-yaml` is the first third-party module, named in
`allowedModules` with its reason. The live test reads a document the operator
names and creates nothing, which is why the standing fact above now splits live
reads from live writes: a live write needs a production writer, and M6 is the
first milestone with one. The result is documented in CLAUDE.md under
"v2 lives at `go/`", and in the README under "The Go rewrite".

What M2 leaves for M3:

- The response shape of the Docs read's `comments` key is measured, not
  documented, so the decoder is loose on purpose. The first live run records a
  redacted fixture, and `docs.CommentRanges` is tightened to that shape then.
- `AllowCreateIn` still has no production caller. Its first one is M6's
  publish, and M3 must not invent one to make a test pass. **Answered by M3:**
  it invented nothing for a test, and it found a production caller anyway. The
  capability probe has to create a document to ask its question on, and it is
  the one door a create may go through.
- The review skill reads `comments` and `read` and does all the judging. The
  binary reports the marker, `resolved`, `by_gdoc`, the witness and
  `gone_since_last_look`, and nothing in Go says what any of them means. M3 is
  the milestone most likely to want that rule bent, and it stays.
- The revmux review of the M1 guard (2026-09-06) left one major and four minor
  findings in `go/internal/guard`, all on the write path: `isSuggestMode` reads
  `writeControl` more loosely than the server does; a repeated JSON key
  collapses before `judgeRequests` and `checkParent` see it; `DELETE` and
  `PATCH` on any comment are carried at `LevelSuggest`; `resumable` uploads can
  never complete; nothing tests that a refused request closes the body. M3 is
  the first milestone that sends a write, so they belong to it.

**The binary size delta.** All of M2 against M1's last binary, darwin/arm64:
10,162,818 to 12,132,722 bytes, +1,969,904 (+19.4%). Roughly half of that is
`goccy/go-yaml` (+1,039,728 on darwin/arm64, +1,121,728 on darwin/amd64,
+1,101,824 on windows/amd64, each about +9.4%) and half is the eight new
packages. All three targets still build with `CGO_ENABLED=0`.

### M3. Writing, and the first usable review (done 2026-09-07)

The capability probe, run on every `propose` invocation against a throwaway
document, with cleanup on every failure path. `reply` with the 🤖 prefix and
`commentUpdateState`. `propose` in SUGGEST mode: anchored 🤖 comment, read-back
(absent under `PREVIEW_WITHOUT_SUGGESTIONS`, span reads as intended), and
comment attachment proven through the docx export. Withdraw: only a proposal
proven to be gdoc's own, confirmed by the answer naming the id **and** a
read-back. Then the **non-live review skill**: marked comment in, hub-context
answer or proposal out, verified on a real document end to end. **v2 becomes
daily-usable here.** Acceptance: spec items 4 and 6.

**Landed 2026-09-07.** Four write commands, and every change inside a Google Doc
is a suggestion the guard holds to `writeMode: SUGGEST`. `probe` creates a
throwaway document in the folder it was given, suggests a word in it, reads it
back and trashes it, and `propose` runs it every time rather than caching an
answer the binary has nowhere to keep. `reply` posts one 🤖 reply and reads the
thread back. `propose` writes one `batchUpdate` per proposal after its own fresh
read, finds the span by the words rather than an index, and reports `verified`
as three read-backs: `suggestions_inline`, `preview_without_suggestions` and
`docx_anchored`. Fewer than three is `ok: true` with `verified: false` and the
route named, because the write happened. `withdraw` retracts only what the
note's `proposals[]` records as gdoc's own, through a `rejectSuggestion` the
guard carries only for that granted id (Nail's decision, 2026-09-07, after the
live write test showed a delete retracts half of a replace proposal), and only
when `rejectedSuggestionIds` and a fresh read agree it is gone. The review skill is rewritten over those
commands and does all the judging; the binary grew no field that decides
anything. The five revmux findings on the M1 guard were fixed before any of it
was written, and `PATCH` and `DELETE` on a comment left `commentWrites` because
nothing here calls them. `gdoc` on PATH is the Go binary now and `gdoc2` is
gone. The result is documented in CLAUDE.md under "The four write commands" and
in the README under "Writing into a document".

What M3 leaves for M4:

- The skill written here is the one-pass one. M4's live session polls
  `comments --since` on the cursor this binary already prints, and the same
  skill acts on what arrives, so the work is the loop and the partial-read
  behaviour rather than a second set of write commands.
- A poll that could not read half the review must say so and not act as if the
  review were clean. The one-pass skill has no equivalent, because a failed read
  simply stops the run.

What M3 leaves for M6:

- `AllowCreateIn` has a production caller now, the probe, rather than waiting
  for M6 as M2 expected. M6's publish is a second caller of the same door, and
  the create path it needs, the parent check, the upload-shape check and the
  response learning, has been exercised against Drive by every `propose` run.
- The probe's create is a bare `files.create` with a JSON body. M6's is a
  multipart upload, which `checkUploadShape` permits and nothing had sent yet,
  so the first real test of that grammar was M6's. **Done**: the guard reads the
  first MIME part now, on the three-signal rule under M6 below.

What M3 leaves open elsewhere:

- Deleting or editing a comment gdoc wrote is not carried by the guard. A
  withdrawn proposal's 🤖 comment stays, with a reply saying so. M7 or M8 adds
  the method back beside a caller.
- The Docs read's `comments` key is still decoded loosely, as M2 left it. The
  live write run records the fixture to tighten it against.

### M4. The live session (done 2026-09-07)

Session-scoped polling on the cursor: continuity across polls, partial-read
behaviour (a poll that could not read half the review says so and does not act
as if clean), colleague `ai!`, and clean cancellation. The review skill gains
its live mode. Acceptance: a live run against a real document with a second
account commenting.

**Landed 2026-09-07.** The polling moved inside the call, on Nail's decision the
same day: `comments <url> --since CURSOR --wait 9m` asks Drive every ten seconds
and comes back on the first activity after the cursor, or at the deadline with
an empty window. One call per poll would have made the latency a minute, because
a Claude Code session cannot wake more often than that, and would have spent a
model turn on every quiet tick. `comments.Wait` holds the loop and takes the
poll as a closure, so the package still holds no session and no URL. The
envelope gained one object, `waited`, carrying `polls`, `seconds` and
`interrupted`, and nothing else: it is a count, a duration and a flag, and no
field in it decides anything. An interrupt is an answer, `ok: true` with no
threads and the cursor handed in, so the output contract holds under the one
signal a live session sends every time it ends; a failed poll is `ok: false`
with the polls so far, so the skill can tell an unread window from an empty one.
The binary still keeps no state of its own: the wait writes no file it learned
anything from, and the cursor it prints is all that carries to the next call.
The one file it can touch is the saved OAuth token, which a poll replaces
through `gapi.Session.send` when the access token has to be refreshed, exactly
as every other command does. The review skill gained a
"Live mode" section, the loop over those calls, and the rule that a receipt on a
colleague's marked comment names who asked. `internal/live` gained
`TestLiveWaitSeesANewComment` behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`. No
new import and no new module. Documented in CLAUDE.md under "`--wait` is one
call that polls" and in the README under "Reading a document".

What M4 leaves for M8:

- One live session watches one document, the link Nail gave. The hub-wide
  session, one watch over every paired note in the folder, is M8's with the
  align skill. Nail's decision, 2026-09-07.
- The wait polls Drive and Docs both on every tick, because the ranges come from
  the Docs read. A cheaper tick that lists comments alone and reads the document
  only when something arrived is an optimisation for later, if the quota ever
  matters.
- Suggestions accepted or rejected during a live session are not watched. The
  next `suggestions --md` run reports them as `gone_since_last_look`.

What M4 leaves open elsewhere:

- The acceptance run with a second account commenting is still outstanding. The
  colleague `ai!` path is exercised by the skill's rule and its wording; the
  first real proof is a session with a colleague in the margin.

### M5. Generator parity (done 2026-09-08)

`house.yaml` to a complete docx on etree plus `archive/zip`, ported from
`spikes/config/gen.py`; `compare.py` ported as the Go drift test with the
master docx as fixture. Gate: the 160 items with no new differences.
`house.yaml` is parsed with the M2 YAML dependency under the same strict
rules, and the milestone records the cross-platform binary-size delta and
confirms the module graph gained nothing transitive.

**Landed 2026-09-08.** `gdoc build --md note.md --out file.docx` writes an
Altery house-style docx and reaches nothing: no Drive, no Docs, no network at
all, and it opens no policy and no session because there is no wire to judge.
Five packages: `house` parses the style, embedded with `//go:embed` and replaced
for one run by `--house`; `cover` reads the author's front matter under v1's own
key names, so a note written for the Python tool builds here with no edits;
`body` walks the markdown with goldmark, tables included; `render` writes the
twelve parts and the logo with etree, so every `w:t` is escaped by the library rather than by
whoever formatted the string; `drift` holds the measured list. `--out` refuses a
file that is already there without `--force`, and the write goes through
`internal/atomicfile`.

The gate runs both ways off one item list, with one reader per item and a
`Source` interface giving that reader either a docx or a Docs answer. Offline it
is in `make test`: 169 items, 142 IDENTICAL, 2 CLOSE, 22 DIFFERENT, 3 MISSING,
and every DIFFERENT or MISSING row is in `drift.Known` with its reason. A CLOSE
row is never in it: `Unexplained` asks for an entry on the two verdicts that
fail the gate and on nothing else.
`TestTheGateReadsTheWholeList` states that the gate reads the whole list rather
than a subset. The three PDF items compare.py carried are dropped, because the
Never list says gdoc never exports a PDF; 160 was its count including them, 157
without, and `bold` and `italic` for all nine named styles make 169. The 0.001pt
logo rounding DECISIONS.md counted is inside that item's two-point tolerance and
reads as CLOSE.

`allowedModules` gained `beevik/etree` and `yuin/goldmark`, which is all three
SPEC.md agreed. The module graph gained nothing transitive: `go list -m all` is
those two plus `goccy/go-yaml`. The binary-size delta against `c876576`, the
last M4 commit, is about 1.4 MB per platform:

| Platform | M4 (c876576) | M5 | Delta |
|---|---|---|---|
| darwin/arm64 | 12,393,250 | 13,786,466 | +1,393,216 (+11.2%) |
| darwin/amd64 | 13,271,536 | 14,728,192 | +1,456,656 (+11.0%) |
| windows/amd64 | 13,133,312 | 14,564,864 | +1,431,552 (+10.9%) |

Documented in CLAUDE.md under "The generator reads `house.yaml` and nothing
else" and in the README under "Building the document".

What M5 left for M6, all four **settled** on 2026-09-08 and written up under M6
below:

- **The upload.** `build` writes a file and stops. Uploading it with conversion,
  verifying by export and trashing on a failed front-matter write is `publish`.
- **The live drift gate.** `TestLiveDrift` is written and runs, on the guard's
  multipart create and `gapi.PostMultipart`.
- **The `gdoc:` block written by the publish.** `publish` writes the ids and the
  publish record, and it is the only command that creates the block.
- **The v1 front matter migration.** A note v1 published carries `gdoc: <id>` as
  a plain string and v2's strict reader refuses it. `cover` is unaffected,
  because it skips the `gdoc:` key whatever it holds, so a v1 note builds today;
  it is `propose --md` and `withdraw` that cannot read the pairing. The reader
  keeps refusing it and publish is the only command that creates the block. The backlog
  item is deleted and the entry is in `docs/v2/DECISIONS.md` under that date.

What M5 leaves open elsewhere:

- Code blocks are not rendered, by decision. A note carrying one gets a warning
  naming the line, and blocks of HTML, inline HTML and footnotes are the same
  answer. A list item holding only one of those, or nothing at all, takes no
  marker, and that is a warning of its own naming the line and what it cost.
- An internal anchor link, `[see](#scope)`, publishes as a link that opens
  nothing, because no heading carries a bookmark. It is written up in
  `docs/backlog/internal-anchor-links.md` and it is Nail's call.
- The offline gate exempts a row by name, so a `Known` row is exempt whatever
  the two sides now measure. Fourteen of the twenty-five are measurements
  rather than words, and `docs/backlog/drift-known-pins-no-value.md` says what
  pinning the pair would buy.
- `drift`'s `FromDoc` half read what a Docs answer states on the run or the
  paragraph and resolved no style behind it, so a live row whose value was
  inherited read nil on both sides and passed. **Settled at M6**: `Doc.resolve`
  walks the chain and the backlog item is deleted.
- One template. Everything is measured against `altery-group-policy-v1.0`.
- The acceptance run by hand, a built docx opened in Word and in Drive beside
  the same note published by v1, is Nail's.

### M6. Publish

Render, upload with conversion into the given folder, verify by export, write
the front matter crash-safely to the M2 schema: ids and the publish record.
Failure policy is rollback-first: if the front-matter write fails after a
successful upload, trash the created document and verify `trashed=true`; only
if the rollback also fails does the error carry the live id and the exact
recovery steps. Acceptance: spec items 2 and 3.

**Landed 2026-09-08.** `gdoc publish --md note.md --folder-id FOLDER` renders
the note the way `build` renders it, uploads the bytes with conversion into the
one folder the run was given, reads the new document back three ways, and
records the pairing in the note. It is the only command that creates the
`gdoc:` block.
The policy opens with `AllowCreateIn` and no file at all, so the run's only door
is the folder and the new document's id is learned from the create the guard
itself carried.

The upload is what three other pieces had been waiting for, and all three are
settled. The guard judges a multipart create now: `multipartCreate` reads the
same three signals Drive picks its parser from, the `/upload` path, the
`uploadType` or `upload_protocol` value and the media type, and refuses the
request unless all three agree or none does. `metadataPart` takes the boundary
off the header through `mime.ParseMediaType`, reads the first part with
`NextRawPart`, requires its media type to be JSON, refuses a
`Content-Transfer-Encoding` and any header outside `content-type`, and hands the
bytes to the same duplicate-key and parents check a plain JSON create goes
through. `gapi` gained `Session.PostMultipart`, which builds the whole body and
its boundary once from `crypto/rand`, so the 401 retry sends the same bytes and
the same boundary. And `TestLiveDrift` exists and runs.

`verified` is three read-backs: the Docs read, the tab count, and a docx export
that reads as a docx. Fewer than three is `ok: true` with `verified: false` and
the route named, the way `propose` reports, because a document that exists is a
document that exists. The failure policy is rollback-first: a front-matter write
that failed after a successful upload trashes the document and confirms the
trash, and only a rollback that also failed puts the live id on the envelope
with the recovery steps. `rolled_back` is a `*bool` so that a clean run says
nothing about a rollback nobody tried. The trash itself moved into
`internal/drive`, because `probe` and `publish` both believe the same rule: an
unconfirmed trash is a failure, and two copies of that are two chances for one
of them to report a document gone that is still there.

`build` and `publish` render through one function, `renderNote`, returning one
`noteDocx`, and `TestBuildAndPublishRenderTheSameBytes` compares the file
`build` wrote against the part `publish` uploaded so the two cannot drift. The
re-read before the note is written is publish's own inverse of `freshNote`: it
refuses a block that has **appeared**, and it compares the whole file rather
than the body alone, because the author's front matter feeds the cover.

Two decisions Nail took on 2026-09-08, both in `docs/v2/DECISIONS.md`. The
publish record drops `revision_id` and is `{at, title, house}`: a Drive revision
id needs a route the guard does not carry, for a field nothing reads. And v2's
reader keeps refusing v1's plain `gdoc: <id>` string, with publish the only
writer of the block, so no schema-0 shape enters the reader; a note v1 published
is rewritten by hand once or republished by v2.

`drift.Doc` resolves a style behind a value now, which the live gate needed:
`textRun.textStyle`, then the paragraph's `namedStyleType`, then
`namedStyles[type]`, then `namedStyles["NORMAL_TEXT"]`, with paragraph
properties folding down the same chain. A property is taken only when the
message carries it, because an absent `bold` on a run is that run inheriting. A
style the answer does not carry is still not found, which is what the docx half
answers too. `docsSilent` pins the 71 names the fixture is expected to answer
nil for, so a row that starts reading nil fails rather than passing as IDENTICAL
against another nil. Nothing was re-baselined: the offline gate reads two docx
files, so its 169 rows did not move.

`TestLiveDrift` is the live gate, behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`.
It builds `03-policy.md`, uploads it and the master with conversion, reads both
through the Docs API, runs the same 169-item list, prints the table, fails on
any verdict that is not IDENTICAL, CLOSE or a name in `drift.Known`, and trashes
both documents. Its read is a bare `documents.get` with no query, not `docs.URL`:
`includeTabsContent=true` moves the content into `tabs[]` and empties the legacy
`body`, `headers` and `footers`, which are exactly what `drift.Doc` reads, so
through `docs.URL` the gate would answer nil for every one of those rows and
pass on a document it never looked inside. `TestLivePublish` is beside it, and
`TestTheLiveFixturesRenderWithNoNetwork` renders both notes in `make test` so a
fixture that moved is found there rather than in the middle of a live run.

`allowedModules` did not change and the module graph gained nothing: `go list -m
all` is still `beevik/etree`, `goccy/go-yaml` and `yuin/goldmark`. The
binary-size delta against `e77d9f3`, the last M5 commit, is about 90 KB per
platform:

| Platform | M5 (e77d9f3) | M6 | Delta |
|---|---|---|---|
| darwin/arm64 | 13,881,474 | 13,952,082 | +70,608 (+0.5%) |
| darwin/amd64 | 14,838,832 | 14,933,472 | +94,640 (+0.6%) |
| windows/amd64 | 14,676,480 | 14,768,640 | +92,160 (+0.6%) |

Acceptance: **spec item 2** is answered by `TestLivePublish`, which publishes a
note and asserts the read-back, the title, the one tab and the block written
into the note. **Spec item 3** ("the positioned logo, a live contents list and
footer page numbers, verified by export") is answered by `TestLiveDrift`'s
`acceptance`, which asks each of the three twice: the value, so a generator that
stopped emitting it fails rather than matching a master that lost it too, and
the verdict, so a value that survived Google's import as something else fails as
well.

Documented in CLAUDE.md under "The multipart create is three signals that have
to agree" and "`publish` is the fifth write, and it makes the document rather
than changing one", and in the README under "Publishing the document".

What M6 leaves for M7:

- **A second version of a note.** `publish` refuses a note that already names a
  document. Replacing the paired document, and the state transition that gives
  the note the new id and a fresh publish record in the same change, is
  `restyle --new`'s.
- **`GrantInPlace`.** Still deleted, still M7's, and the level it raises to is
  still Nail's decision then.
- **Editing or deleting a comment.** `commentWrites` is still `POST` alone, so
  the 🤖 comment a withdrawn proposal made stays where it is.

What M6 leaves open elsewhere:

- `publish` has no skill caller. It is Nail-invoked, and wiring it into a skill
  is M9's with the install story.
- A resumable upload is still refused. The guard carries neither the `PUT` nor
  `upload_id`, so the shape could never finish, and nothing here needs one: a
  house-style docx is tens of kilobytes.
- The live drift table has not been read by a person yet, so no row has joined
  `drift.Known` for a reason the live gate found. That is Nail's, and a row
  joining `Known` is a decision written down with its reason, never a test
  somebody loosens.
- The offline gate's `Known` still exempts a row by name rather than by value,
  which `docs/backlog/drift-known-pins-no-value.md` holds.
- One template. Everything is still measured against `altery-group-policy-v1.0`.

### M7. Restyle

Two-phase by design: `restyle --dry-run` emits a machine-readable survey
(threads, suggestions, chips, tabs, revision id) and the apply invocation
hands the survey back, rechecking revision and tabs before writing. In-place
styling under the per-run write grant, the finishing checklist page under a
named range, `--new` through the generator including the document-content
extraction it needs, and the nothing-to-protect detection that offers rather
than takes. `--new` also defines its state transition: when the new document
replaces the paired one, the front matter gets the new id and a fresh publish
record in the same change; a document left unpaired is refused by alignment
until it is paired on purpose. Acceptance: spec item 5, the ten-feature
preservation run.

### M8. The diff, alignment, and the align skill

The align skill over `read`'s output and the hub markdown it reads itself:
composes the comparison, judges what matters with Nail's word on which side is
the source of truth, proposes both ways, never deletes from the hub, and works
on a document gdoc never published. Whether a binary diff command earns its
place at all is decided here, and with it the second dependency decision:
`sergi/go-diff` (proven in the spike) with a written reason, a hand-rolled word
diff, or nothing, if the skill reads both sides well enough without one.
(Corrected 2026-09-06: this used to name a two-sided diff command and
drift-direction classification from M2 fingerprints. Both are withdrawn, see
SPEC.md "The diff".)

### M9. Release

Packaging only, because portability was continuous since M1: the dist matrix
(darwin/arm64, darwin/amd64, windows/amd64, `CGO_ENABLED=0`), a real Windows
smoke test (config path, token write-and-replace, console output), the
copy-one-file install story, a stated install path for the skills on the one
machine that runs them, and the v1 retirement note in the README.

## Ordering rationale, in one paragraph

The guard exists before the first network call, because retrofitting a safety
property is how it gets holes. Reading before writing, because reads are safe
to test against real documents and the readers are what every later milestone
debugs with. The review skill lands with the writes in M3, not at the end,
because v1 is retired now and answering comments is the daily need; the plan
is only honest if "usable early" names the milestone where it becomes true.
The front-matter schema is designed in M2 even though publish arrives in M6,
because publish is one-shot and the record it writes (ids, publish record,
schema version) has to be right from its first version. Generator parity is its own milestone because it is the
largest single risk and deserves its own gate. Restyle after publish, because
`--new` and the checklist depend on the generator. Alignment last among the
features, because it consumes everything: the readers, `read`'s output, the
proposals. Platform work is continuous from M1; only packaging waits.
