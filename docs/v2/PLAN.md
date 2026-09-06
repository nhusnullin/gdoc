# gdoc v2 master plan

2026-08-29. The build order for [SPEC.md](SPEC.md). Nine milestones, each one
producing working, testable software. Detailed task-by-task plans live in
`docs/plans/` and are written when a milestone starts, so each one
is written against the code that actually exists by then. Milestone 1's is
written, and that milestone is done.

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
- None of those three is in `go.mod` yet, and the M1 boundary test refuses
  every one of them. `allowedModules` in `TestNoThirdPartyDependencies` is an
  empty allowlist, so "standard library only" is M1's state rather than v2's
  rule. The milestone that first needs a module adds its path to that map, and
  nothing else: it does not delete the test and it does not widen it to accept
  whatever `go.mod` says. M5 is the likely first, for goldmark.
- No `golang.org/x/oauth2`: the token refresh is a single POST to Google's
  token endpoint and is hand-rolled on the standard library.
- v1 is retired as of 2026-08-29, Nail's call. The Python code stays in the
  repo untouched as reference until v2 replaces its last use, then its removal
  is a separate decision. Nothing maintains it.
- Platforms: darwin/arm64, darwin/amd64, windows/amd64. No linux.
- Live tests follow v1's convention: skip without credentials, create documents
  only in the Drive test folder, never touch a document gdoc did not create.

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

### M2. Reading, and the honest witness

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

### M3. Writing, and the first usable review

The capability probe, run on every `propose` invocation against a throwaway
document, with cleanup on every failure path. `reply` with the 🤖 prefix and
`commentUpdateState`. `propose` in SUGGEST mode: anchored 🤖 comment, read-back
(absent under `PREVIEW_WITHOUT_SUGGESTIONS`, span reads as intended), and
comment attachment proven through the docx export. Withdraw: only a proposal
proven to be gdoc's own, confirmed by `deletedSuggestionIds` **and** a
read-back. Then the **non-live review skill**: marked comment in, hub-context
answer or proposal out, verified on a real document end to end. **v2 becomes
daily-usable here.** Acceptance: spec items 4 and 6.

### M4. The live session

Session-scoped polling on the cursor: continuity across polls, partial-read
behaviour (a poll that could not read half the review says so and does not act
as if clean), colleague `ai!`, and clean cancellation. The review skill gains
its live mode. Acceptance: a live run against a real document with a second
account commenting.

### M5. Generator parity

`house.yaml` to a complete docx on etree plus `archive/zip`, ported from
`spikes/config/gen.py`; `compare.py` ported as the Go drift test with the
master docx as fixture. Gate: the 160 items with no new differences.
`house.yaml` is parsed with the M2 YAML dependency under the same strict
rules, and the milestone records the cross-platform binary-size delta and
confirms the module graph gained nothing transitive.

### M6. Publish

Render, upload with conversion into the given folder, verify by export, write
the front matter crash-safely to the M2 schema: ids and the publish record.
Failure policy is rollback-first: if the front-matter write fails after a
successful upload, trash the created document and verify `trashed=true`; only
if the rollback also fails does the error carry the live id and the exact
recovery steps. Acceptance: spec items 2 and 3.

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
