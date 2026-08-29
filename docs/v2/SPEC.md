# gdoc v2 spec

2026-08-29. Status: agreed, awaiting plan.

This restates [DECISIONS.md](DECISIONS.md) as requirements. The decisions hold
the why and the measurements; this file holds what gets built. Where the two
disagree, this file is wrong.

## Principles

Serves: all four.
1. One static Go binary, zero external programs, zero packages to install on the
   target machine. pandoc is gone entirely.
2. The root is where the user stands. No hidden state anywhere: what gdoc knows
   lives in the front matter of the markdown it describes, under a single
   `gdoc:` key that gdoc owns and never beyond it. `.gdoc/`, `baseline.md` and
   `pending.md` do not exist in v2.
3. The guard owns the transport and ids carry write levels. Verification by a
   second route wherever a write matters.
4. Every word costs a reader's attention: outcomes in the reader's language,
   machinery out of documents, diagnostics in the terminal on request.

Strains: none in the machine sense. One accepted external risk: proposing in the
document depends on Google's Developer Preview, which is Pre-GA and can be
withdrawn. Accepted 2026-08-29, on the condition that the loss is loud, never a
silent edit.

## What v2 is

Three parts, one product.

- **The binary** deals in facts. It reads, writes what it is told to write,
  reports, and exits. It never prompts, never judges, never decides what a
  difference means.
- **The skill** is the judgement: what a comment asks for, whether a drift
  matters, what to propose, whether an action needs Nail's confirmation.
- **A comment in the document** is a second door into the same agent, with the
  same hub context as a terminal session. Not a queue, not a reduced mode.

Prompts are part of the product. They live in this repo and a change to one is
reviewed like code.

## Scope

In: `publish`, `restyle` (two modes), reading comments and suggestions, replying,
proposing as native suggestions, withdrawing gdoc's own pending proposal, the
two-sided diff that `align` composes, the finishing checklist, OAuth.

Out, permanently or until a decision says otherwise: PDF export and any `release`
command, tab features of any kind, a daemon or watcher, accepting or rejecting
suggestions, resolving or reopening threads, anchor banking, MCP as transport,
the colour marking scheme, service accounts, running git.

## The binary

Go. One static file per platform. Third-party dependencies, each with its stated
reason as principle 1 requires:

- `beevik/etree`: Go's `encoding/xml` corrupts OOXML; etree round-trips the real
  194 KB template with one apostrophe of difference (spike `go/xmltest`).
- `yuin/goldmark`: parses everything the hub markdown needs, including an image
  nested in a heading; `==mark==` was added in 55 lines (spike `go/gmtest`).

Everything else is the standard library. A third dependency needs its reason
written here first.

### Output contract

- Every command writes exactly one JSON object to stdout and exits. Human prose
  never mixes into stdout.
- The object always carries `ok` (bool). On failure it carries `error`, a message
  that names the problem. Warnings ride in `warnings` and are never dropped: a
  thing gdoc could not do is reported, not omitted.
- Exit code 0 when `ok` is true, non-zero otherwise.
- The binary never prompts and never reads stdin interactively. It must run
  headless.

## Auth

OAuth only. One scope: `https://www.googleapis.com/auth/drive`, which the Docs
API accepts for every call gdoc makes. The bundled Internal client carries over
from v1, with both of its rules: the secret stays in version control, and the
client stays User type Internal. The public-repo caveat in CLAUDE.md applies
unchanged. Token and config live in `~/.config/gdoc-agent/`, never in a repo.

## The guard

Serves principle 3. This is the safety property everything else stands on.

- **One package owns the network.** Nothing else in the binary can construct an
  HTTP request. A test fails the build if HTTP construction appears in any other
  package. This is the v1 allowlist test, made stronger by owning the transport
  instead of wrapping a client.
- **Two doors into the id set**, exactly as v1: ids handed in on the command
  line, and ids learned from a create the guard itself carried. `files.list` is
  refused outright. An empty set refuses everything.
- **Every id carries a write level:**
  - learned from a create: fully writable, gdoc made it
  - handed in: read and suggest only, never direct-editable
  - `restyle` in place is the one exception, granted explicitly per run, never
    inherited or remembered
- The preview surface is hand-rolled JSON (it is absent from the discovery
  document), and it goes through the same package because there is nothing else
  to go through.

## Verification: never trust a success

These are requirements, not advice. Each has a measured 200-that-lied behind it.

- **Capability probe.** Before the first suggest-mode write of a session, probe
  `writeMode: SUGGEST` against a throwaway document and cache the answer for the
  session only. An unenrolled project answers 200 and direct-edits.
- **Read-back after every proposal.** Confirm the change is absent from the
  document read with `PREVIEW_WITHOUT_SUGGESTIONS` (so it is a suggestion, not an
  edit) and that the affected span reads as intended (index arithmetic is the
  hazard: a 200 once placed text one character before a full stop).
- **`commentUpdateState`, not the status code**, on every write that carries a
  comment. `ALL_SAVED` is success; anything else is reported as the partial
  failure it is.
- **The docx export is the honest witness** for comment attachment. Drive's
  `anchor` and `quotedFileContent` survive detachment and prove nothing.
- **Always read with `includeTabsContent=true`.** Reading without it silently
  sees one tab.
- **A write target with more than one tab stops the command**: tabs are
  unsupported, reported as such, never guessed at.

## Commands

### `publish`

Builds a document that does not exist yet. Renders the hub markdown plus
`house.yaml` to a docx and uploads it with conversion into the folder it was
given. One upload; no measuring pass, because the docx carries live fields.

Born free, because they come from the file: the positioned logo in a first-page
header (`wp:anchor` behind `<w:titlePg/>`), the live refreshable contents list
(Word TOC field), footer page numbers (PAGE field). No finishing checklist.

After upload, `publish` verifies the document exists and records what it needs
in the source file's front matter: at minimum the document id, the folder, and
the publish record. It then reports every file it changed on disk.

`publish` runs once per document. There is no republish. Later hub-to-doc
changes travel as suggestions; a document whose review has run its course gets
`restyle --new` and a new URL, chosen by a person.

### `restyle`

Puts a document gdoc did not create into the house style. Two modes, and the
careful one is the default.

**In place (default).** `batchUpdate` on the original. Keeps every comment
thread with real authors and anchors, every pending suggestion, every smart chip
with its data, every Drawing, and the URL. The house style is approximate. The
three things the API cannot create (first-page header with logo, contents list,
footer page numbers) go on the **finishing checklist**: a page after the cover,
checkbox bullets, the exact menu path per item, marked with a named range so it
is removable in one call. Requires the explicit per-run write grant from the
guard.

**As a new document (`--new`).** Renders the content to a docx and uploads it as
new. Exact house style, new URL, and it loses the comments, the suggestions and
the chips. The report says so before anything is created.

**The detected exception.** A document with no comments, no suggestions and no
chips has nothing to protect. gdoc says so and offers `--new` as the better
route. It never takes it silently: a new URL is a person's choice.

**Survey first.** A dry-run mode reports what the document holds (threads,
suggestions, chips, tabs) so the skill can confirm with the facts on screen
before a write happens. Nothing about the original is ever modified in survey.

### Reading comments

Lists threads with real character ranges (`commentsViewMode`, which requires
`includeTabsContent=true`). Reports the marker on each comment: `ai:`, `ai?`,
`ai!`, or none. An unmarked comment, including an unmarked follow-up in a thread
gdoc has answered, is reported and never acted on. The marker is the trigger,
always: stickiness carries context, never authority.

### `reply`

Posts into an existing thread. Plain text only. Every reply gdoc writes opens
with 🤖 and nothing else: no name, no prefix text. `commentUpdateState` is
checked. gdoc never resolves and never reopens a thread, even though the API now
can: resolving means the answer was accepted, and only Nail accepts.

### `propose`

Writes a change as a native Google suggestion (`writeMode: SUGGEST`) with an
anchored comment on the exact words explaining why, opening with 🤖. Subject to
the capability probe and the read-back above. On a handed-in id this is the only
write level the guard allows, which is the point.

An `assigneeEmailAddress` may be set when the skill knows who should answer.

### Withdrawing a proposal

gdoc may retract its own pending, unaccepted suggestion: `deleteContentRange` in
suggest mode, confirmed by `deletedSuggestionIds` in the response and a
read-back. This does not touch Nail's rule: there was no decision yet to
resolve. gdoc never accepts, rejects or deletes anyone else's suggestion.

### Reading suggestions

Lists pending suggestions with their stable ids and quotes what each proposes.
On a later read it reports which were resolved and how: an accepted suggestion's
text became ordinary text, a rejected one's did not. gdoc learns decisions by
reading; it never makes them.

### The diff (what `align` composes)

The binary command emits both sides and the raw differences: the document's
current content and the hub markdown, side by side. Whether a difference matters
is the skill's judgement. Alignment runs both ways, writes to the hub only with
agreement, and never deletes from it: content that exists only in the hub is
unpublished work, not drift.

How gdoc classifies which side moved since the last publish is **deferred to the
plan** (the front-matter fingerprint scheme is the standing recommendation). The
front matter is the only place that state may live.

## The generator and `house.yaml`

`house.yaml` is the house style: page geometry, all nine named styles, the
cover, header and footer, the three tables cell by cell, the contents field,
heading numbering, and the logo as base64. The generator reads it plus markdown
and emits a docx. Nothing reads the master `.docx` at runtime; it stays in the
repo as provenance and as the drift test's fixture.

The generator and the drift test exist today as Python spikes (`config/gen.py`,
`config/compare.py`) and are ported to Go. **The 160-item comparison is the
acceptance gate:** it renders from the config and from the master and fails on
any drift beyond the two known real differences (a 0.001pt logo rounding, a
stale-TOC page count).

House-style tests state house values as literals, never by reading the constant
they test. The v1 rule carries over unchanged.

## Skills

Two named workflows, both judgement, both skills rather than commands:

- **The review session.** Reads the threads, answers `ai?` from the hub, carries
  out `ai!` against the hub, proposes document changes as suggestions. If it
  could not read half the review (suggestions unreadable, comments partial), it
  says so and does not report clean.
- **The alignment check.** Composes the diff, judges what matters, proposes both
  ways: suggestions into the document, edits into the hub with agreement.

Skills stay symlinked from this repo, one copy, as in v1.

## How a comment reaches the agent

The part that connects the document's margin to the hub folder, stated here
because it is design, not wiring.

- **There is no watcher.** Nothing polls Drive and nothing runs on its own. A
  comment written today waits, visible in the document, until a session looks.
- **The loop starts when Nail starts it.** He opens a session in the hub folder
  (or runs the review skill there). The hub being the working directory is what
  gives the agent its context: the notes, the decision log, the raw material.
- **The front matter is the join.** Each source markdown names its document id.
  The skill walks the folder, finds the documents that belong to it, and calls
  the binary to read their threads.
- **Then the model applies:** `ai?` is answered from the hub, `ai!` is carried
  out against the hub, and if that needs a note changed or a file written, it
  happens in that moment, in that session. Nothing is queued.

What stays out of the spec: how a session is launched, scheduled or named. That
is ordinary Claude usage, not gdoc's concern, and it goes in the plan only where
the skill needs a convention.

## Never

The hard list. Each is transport-enforced where the guard can reach it, and a
skill rule where it cannot.

- Never direct-edit a document gdoc did not create. Guard-enforced.
- Never replace the body of a document that exists. Guard-enforced.
- Never accept, reject or delete anyone else's suggestion.
- Never resolve or reopen a comment thread.
- Never delete from the hub.
- Never run git, in the binary or in the skills.
- Never prompt, in the binary.
- Never export a PDF. Nail downloads it from the browser.
- Never write to a multi-tab document.
- Never write markdown into a comment thread.
- Never trust a status code where the write matters.
- Never write a comment or reply that does not open with 🤖.

## Acceptance for v2.0

Each is a test or a checkable run, not a claim.

1. The guard test: HTTP construction outside the guard package fails the build;
   a request for an id outside the set is refused; a direct edit on a handed-in
   id is refused.
2. The drift test: config-rendered and master-rendered documents compare across
   the 160 items with no new differences.
3. A publish produces a document with the positioned logo, a live contents list
   and footer page numbers, verified by export.
4. A propose lands as a genuine suggestion (absent under
   `PREVIEW_WITHOUT_SUGGESTIONS`), with its 🤖 comment anchored to the exact
   words, verified through the docx export.
5. An in-place restyle on a copy of the ideal test document preserves all ten
   features of the 2026-08-29 run: person chip, date chip with timestamp, locale
   and format, calendar link, Drawing, three footnotes, both lists, the pinned
   shaded table header, the byte-identical inline image, the anchored comment,
   the pending suggestion. The original is byte-identical afterwards, same
   `revisionId`.
6. The capability probe distinguishes an enrolled project from an unenrolled one
   without writing to any real document.

## Open questions, deferred to the plan

- The front-matter schema and the drift-direction classification for `align`.
- Go module layout in this repo, beside the Python v1.
- Which platforms get built binaries.
