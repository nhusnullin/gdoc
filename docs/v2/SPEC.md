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
two-sided diff that `align` composes, the finishing checklist, live review
sessions (session-scoped polling), OAuth.

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
- `goccy/go-yaml`: parses the `gdoc:` front matter and `house.yaml`. The reason:
  both are human-authored configuration, a product interface where a quiet
  parsing bug silently alters a document, and a hand-rolled parser is where
  such bugs live. This library has zero transitive modules, rejects duplicate
  keys, and supports strict unknown-field rejection. Chosen over `yaml.v3`,
  whose upstream is frozen. Second-opinion reviewed 2026-08-29.

Everything else is the standard library. A fourth dependency needs its reason
written here first; the one open candidate is `sergi/go-diff`, decided at the
alignment milestone.

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

OAuth only. The bundled Internal client carries over from v1, with both of its
rules: the secret stays in version control, and the client stays User type
Internal. The public-repo caveat in CLAUDE.md applies unchanged. Token and
config live in `~/.config/gdoc-agent/` on macOS, and in `%AppData%\gdoc-agent`
on Windows, never in a repo. `GDOC_CONFIG_DIR` overrides both, which is what the
test suite uses. `auth login` is a real desktop flow: loopback redirect with
PKCE, hand-rolled on the standard library. v2 reads a token file v1 wrote, so an
existing install needs no new login.

**Scopes, corrected 2026-08-30.** This section used to say one scope,
`https://www.googleapis.com/auth/drive`. M1 shipped two: Drive, and
`https://www.googleapis.com/auth/documents`, the Docs read/write scope, because
v2 writes suggestions through the Docs API. That is wider than v1, which asks
for `documents.readonly`, and it runs one way: a v1 token still satisfies v2,
but after a v2 login v1's own scope check fails and `gdoc edits` asks for a
fresh v1 login. Written down in CLAUDE.md and README.md as well.

`auth status` never guesses. An absent token file is `token_present: false` with
`ok: true`; a token file that exists and cannot be read is `ok: false` naming
the file, and it still carries the warnings the successful run would have.
`auth_mode` is the constant `oauth`, and `client_source` is the constant
`bundled`: v1's `oauth-client.json` override is not implemented in v2, and
status says so instead of claiming it. When the token carries less than v2 asks
for, `missing_scopes` names the difference and a warning says the Docs calls
will be refused. That is a report, not a failure: the login worked.

`Save` writes a temp file in the same directory, syncs it, and renames.
Same-directory rename is atomic on POSIX. On Windows it is not guaranteed, which
is why the token write is part of the M9 Windows smoke test.

## The guard

Serves principle 3. This is the safety property everything else stands on.

- **One package owns the network.** `internal/guard` is the only package that
  may build an HTTP client or reach a package-level dialer such as `http.Get`.
  Naming `net/http` is not the same thing as dialing with it, and the boundary
  test draws that line: a second, wider allowlist says who may import the type
  at all, and `internal/auth`, `internal/auth/loopback` and `internal/gapi` are
  on it. `internal/auth` takes the guard's client as a parameter and sends
  requests through it; `loopback` runs an `http.Server`, and serving is not
  building; `internal/gapi` builds the `*http.Request` every read goes out as
  and sets the bearer on it, and the client it sends them on is a parameter, so
  the builder allowlist is what proves it makes none.
  Both checks fail in both directions, so a room that stops owning what it was
  listed for fails too. This is the v1 allowlist test, made stronger by owning
  the transport instead of wrapping a client.
- **Two doors into the id set**, exactly as v1: ids handed in on the command
  line, and ids learned from a create the guard itself carried. `files.list` is
  refused outright. An empty set refuses everything.
- **Every id carries a write level:**
  - learned from a create: fully writable, gdoc made it
  - handed in: read and suggest only, never direct-editable
  - `restyle` in place is the one exception, granted explicitly per run, never
    inherited or remembered
- **The level-1 write bar is not yet what this spec assumes.** What holds a
  handed-in document to suggestions today is `writeControl.writeMode ==
  "SUGGEST"` in the request body, a field the client supplies.
  `BLOCKED-BY-API.md` records that `writeMode` is absent from the public Docs
  discovery document and that this call was measured returning 200 while making
  a direct edit. The per-invocation capability probe this spec relies on to
  close that gap **does not exist yet**. Until it does, "never direct-editable"
  is what gdoc asks for, not what the server is known to enforce. Changing it is
  a decision, not a refactor.
- **The guard judges the request it sends.** Method-override headers and a
  `_method` query parameter are refused, a URL carrying credentials is refused,
  and a path whose escaping differs from its plain form, or that walks through
  `.` or `..`, is refused before the host is looked at.
- The preview surface is hand-rolled JSON (it is absent from the discovery
  document), and it goes through the same package because there is nothing else
  to go through.

## Verification: never trust a success

These are requirements, not advice. Each has a measured 200-that-lied behind it.

- **Capability probe.** On every `propose` invocation, before its first
  suggest-mode write, probe `writeMode: SUGGEST` against a throwaway document.
  One invocation may batch many proposals behind one probe. The binary is
  one-shot and holds no hidden state, so there is nothing to cache the answer
  in, and an asserted "--already-probed" from outside would reopen the exact
  silent-direct-edit failure the probe exists to prevent. An unenrolled project
  answers 200 and direct-edits.
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
`includeTabsContent=true`). Takes a `--since` cursor and reports only activity
after it, so a live session's poll is one cheap call. `--wait` makes that call
poll: it repeats the same listing every ten seconds and returns the first window
with activity in it, or an empty one at the deadline. It needs `--since`, and it keeps no state of
its own: the one file it can touch is the saved OAuth token, which any command
replaces when the access token has to be refreshed. The cursor is an opaque
value the binary emits and the caller hands back; the live skill holds it for
the session and it dies with the session. Anything that must survive across
sessions lives in the front matter, nowhere else. Reports the marker on
each comment: `ai:`, `ai?`, `ai!`, or none. An unmarked comment, including an unmarked follow-up in a thread
gdoc has answered, is reported and never acted on. The marker is the trigger,
always: stickiness carries context, never authority.

**The 🤖 reply is the receipt, and the document is the ledger.** No local
record of handled comments exists anywhere, so any session at any time
recomputes the open work from the threads alone, and gdoc's own replies
surfacing in the next poll read as receipts, not as news. Two ordering rules
follow: the receipt is written last, only after the action verifiably landed;
and before re-acting on an old marked comment, the skill checks the pending
proposals and their front-matter provenance, so a crash between the action
and the receipt resolves to writing the missing receipt, never to acting twice.

**Corrected 2026-09-06: whether a comment is answered is the skill's
judgement, not a rule in the binary.** This section used to say a marked
comment "counts as handled exactly when a 🤖 reply newer than it sits in its
thread". That is a heuristic, and a wrong one when the reply was an
acknowledgment or a follow-up question changed what was asked. The binary
reports the facts of a thread: every reply, its author, its time, and whether
it opens with 🤖. The skill reads them and decides what still needs an answer.

### `reply`

Posts into an existing thread. Plain text only. Every reply gdoc writes opens
with 🤖 and nothing else: no name, no prefix text. `commentUpdateState` is
checked. gdoc never resolves and never reopens a thread, even though the API now
can: resolving means the answer was accepted, and only Nail accepts.

**A long action writes at most two messages into its thread.** An
acknowledgment when the work will take noticeably long (one line, reader
language), and the receipt when the action verifiably landed. Never a progress
feed: principle 4, the margin is the readers' room and the terminal is the
operator's. A failed action replaces the receipt with a plain failure naming
the reason, so a thread always ends in a true statement and never in a
dangling "on it". Acknowledgment and receipt are separate replies, never
edits of one message.

### `propose`

Writes a change as a native Google suggestion (`writeMode: SUGGEST`) with an
anchored comment on the exact words explaining why, opening with 🤖. Subject to
the capability probe and the read-back above. On a handed-in id this is the only
write level the guard allows, which is the point.

An `assigneeEmailAddress` may be set when the skill knows who should answer.

**One writer per document.** Work may fan out across documents freely, one
worker per document, and replies within one document may run in parallel
(they target thread ids, not indexes). Proposals within one document are
serialized: a `batchUpdate` lands at character indexes computed from a read,
so a concurrent proposer shifts the ground under the other and the API
reports 200 either way. Each proposal re-reads before computing its indexes
and reads back after landing.

### Withdrawing a proposal

gdoc may retract its own pending, unaccepted suggestion: `rejectSuggestion`
naming the id, in suggest mode, confirmed by `rejectedSuggestionIds` in the
response and a read-back showing no run under the id on either side. This does
not touch Nail's rule: there was no decision yet to resolve. gdoc never
accepts, rejects or deletes anyone else's suggestion, and the guard holds that:
it refuses every request kind whose name carries "suggestion", and the one door
is a per-run grant the `withdraw` command seeds with the id the note's
`proposals[]` records as gdoc's own. A `rejectSuggestion` naming any other id is
refused before it leaves the machine.

**Corrected 2026-09-07, Nail's decision.** This section used to say
`deleteContentRange` in suggest mode, confirmed by `deletedSuggestionIds`. That
was measured on a pure insertion. A `propose` is a replace, and the delete
retracts only the insertion half: the quoted words stay suggested-deleted under
the same id and the answer carries no `deletedSuggestionIds`. The measurements
are in DECISIONS.md under that date.

### Reading suggestions

Lists pending suggestions with their stable ids and quotes what each proposes.
On a later read it reports which ids stopped being pending. gdoc never accepts,
rejects or deletes anyone else's suggestion.

**Corrected 2026-09-06.** This section used to say the binary reports which
suggestions were resolved *and how*. Withdrawn, for the same reason the comment
heuristic was. Whether a suggestion that left was accepted or thrown away is not
in the API: the id is gone either way. So the binary reports
`gone_since_last_look`, which is the id, what it said at the last snapshot, and
when that snapshot was taken. The skill reads `read`'s text and decides.

### The diff (what `align` composes)

The binary emits the document's current content in a form the skill can read
(`read`: headings, paragraphs, tables, pending suggestions inline with their
ids, comment anchors marked). The skill reads the hub markdown itself, from
disk, and composes the comparison. Whether a difference matters is the
skill's judgement, and Nail can tell it which side is the source of truth.
Alignment runs both ways, writes to the hub only with agreement, and never
deletes from it: content that exists only in the hub is unpublished work, not
drift. Alignment also works on a document gdoc never published: there is no
paired note then, and the person or the skill supplies the hub context.

**Corrected 2026-09-06.** This section used to say the binary "emits both
sides and the raw differences", and that classifying which side moved since
the last publish was deferred to the plan with a per-section fingerprint
scheme as the standing recommendation. Both are withdrawn, Nail's call. The
binary reads the document only, because making it read markdown adds coupling
and does nothing for an external document. Fingerprints are dropped: they
answer "did this section change since publish", which says nothing about
meaning, nothing about authority, and nothing at all about a document without
a publish record. The front matter records facts (the pairing, the publish
record, gdoc's own proposals) and no content baseline.

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
  says so and does not report clean. It can run **live**: polling on the
  `--since` cursor until Nail stops it, acting on new marked comments as they
  arrive. A colleague's `ai!` acts too, by decision: the marker is the trigger,
  identity is not a gate, and the guard caps a handed-in document at suggest
  and reply.
- **The alignment check.** Composes the diff, judges what matters, proposes both
  ways: suggestions into the document, edits into the hub with agreement.

Skills stay symlinked from this repo, one copy, as in v1.

## How a comment reaches the agent

The part that connects the document's margin to the hub folder, stated here
because it is design, not wiring.

- **There is no watcher.** Nothing runs on its own. A comment written today
  waits, visible in the document, until a session looks.
- **A session can stay live.** Once launched, a review session may keep the
  document's comments in view: the skill calls
  `comments <url> --since CURSOR --wait 9m`, and that one call asks Drive
  "activity since this cursor" every ten seconds until something arrives or the
  deadline passes, then exits. New marked comments are acted on within seconds,
  and the conversation lives in the document. The loop over those calls is the
  skill's; the binary stays one-shot and keeps nothing but the cursor it prints.
  Liveness ends with the session.
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

**Corrected 2026-09-07, Nail's decision.** This section used to say the skill
loops and the binary makes one cheap call every 5 to 15 seconds, one call per
poll. The polling moved inside the call: `--wait` looks for up to a deadline and
comes back on the first activity after the cursor. The reason is the tool that
runs the command. A Claude Code session cannot wake more often than once a
minute, so a call per poll makes the latency a minute rather than seconds, and
every idle tick spends a model turn on a document where nothing happened. With
the wait inside the binary a quiet document costs one call and no thinking. What
did not change: the binary is still one-shot, still prints one JSON object, and
still keeps no state between calls. What is new on the envelope is `waited`, a
count of polls, the seconds looked and whether a signal ended it. An interrupt
is an answer (`ok: true`, no threads, the cursor handed in), and a failed poll
is `ok: false` with the polls so far, so the skill can tell an unread window
from an empty one. The skill asks for nine minutes because its Bash tool gives
up at ten.

## Never

The hard list. Each is transport-enforced where the guard can reach it, and a
skill rule where it cannot.

- Never direct-edit a document gdoc did not create. Guard-enforced.
- Never replace the body of a document that exists. Guard-enforced.
- Never accept, reject or delete anyone else's suggestion. Guard-enforced: the
  only `rejectSuggestion` that carries names the id `withdraw` granted from the
  note's `proposals[]`, which is gdoc's own.
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

- The front-matter schema. (The drift-direction classification for `align`
  was on this line until 2026-09-06; it is withdrawn, see "The diff".)
- Go module layout in this repo, beside the Python v1.
- Which platforms get built binaries.
