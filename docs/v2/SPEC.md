# gdoc v2 spec

What gdoc v2 is. [DECISIONS.md](DECISIONS.md) holds the why,
[MEASURED.md](MEASURED.md) and [BLOCKED-BY-API.md](BLOCKED-BY-API.md) hold what
Google does and refuses, each package's `doc.go` holds its own rules, and this
file holds what is built. Where this file and a decision disagree, this file is
wrong. It is written in the present tense and carries no amendments: a change
here is a DECISIONS.md entry first, written the same day, with its row in that
file's register, and a date appears here only inside a pointer to one.

## What v2 is

Three parts, one product.

- **The binary** deals in facts. It reads, writes what it is told to write,
  reports, and exits. It never prompts, never judges, never decides what a
  difference means.
- **The skill** is the judgement: what a comment asks for, whether a drift
  matters, what to propose, whether an action needs Nail's confirmation. Prompts
  are part of the product: they live in this repo and a change to one is reviewed
  like code.
- **A comment in the document** is a second door into the same agent, with the
  same hub context as a terminal session. Not a queue, not a reduced mode.

It serves all four principles of [PRINCIPLES.md](../../PRINCIPLES.md), and this
is what each one is here. One static Go binary with zero external programs and
nothing to install beside it. No hidden state, because what gdoc knows lives in
the front matter of the markdown it describes, under one `gdoc:` key it owns and
never beyond it, so `.gdoc/`, `baseline.md` and `pending.md` do not exist. The
guard owning the transport, with ids carrying write levels and verification by a
second route wherever a write matters. And outcomes in the reader's language,
machinery out of documents, diagnostics in the terminal on request. Nothing
strains in the machine sense, and there is one accepted external risk, the only
one: proposing in the document depends on Google's Developer Preview, which is
Pre-GA and can be withdrawn. It is accepted on one condition, that the loss is
loud and never a silent edit.

## Out of scope

The commands below are what is in. Out, permanently or until a decision says
otherwise: PDF export and any `release` command, tab features of any kind, a
daemon or watcher, accepting or rejecting suggestions, resolving or reopening
threads, anchor banking, MCP as transport, the colour marking scheme, service
accounts, running git.

## The binary

Go. One static file per platform. Three third-party modules, each with its stated
reason as principle 1 requires:

- `beevik/etree`, because Go's `encoding/xml` corrupts OOXML and etree
  round-trips the real 194 KB template with one apostrophe of difference.
- `yuin/goldmark`, because it parses everything the hub markdown needs, an image
  nested in a heading included.
- `goccy/go-yaml`, for the `gdoc:` front matter and `house.yaml`, both
  human-authored configuration where a quiet parsing bug silently alters a
  document. It brings no transitive module and rejects duplicate and unknown
  keys.

Everything else is the standard library. A fourth needs its reason written here
first, and the one open candidate is `sergi/go-diff`.

One command table in `cmd/gdoc/commands.go` describes every command once: the
words it takes, its flags with how each one stands in the call, one sentence,
one example, and the function that runs it. The dispatcher, the usage line in
every refusal, `help` and `completion` all read that table, so a command cannot
exist without help and completion for it.

`gdoc help` prints the table, and `gdoc help <command>` prints one entry. The
usage line is a call the binary would accept: what may be left out stands in
brackets, and alternatives are joined by a bar, so `publish` reads `--md
--folder-id [--house <file>]` and `restyle` reads `--dry-run | --from <file>`.
Each flag in the object carries the same under `need`, as `required`, `optional`
or `either`. `--help` and `-h` are aliases anywhere on the line. Help keeps the output
contract: the object on stdout, the words a person reads on stderr, exit 0,
because a question was asked and answered. Bare `gdoc` did no work, so it still
fails with the help on stderr. `gdoc completion zsh --out <path>` and the same
for bash write a shell script and report the path and the line to add: the
script is a written file and never stdout, because stdout is the object.
PowerShell lands with Windows, under the tag that ships it. Decided
2026-09-16, DECISIONS.md.

### Output contract

- Every command writes exactly one JSON object to stdout and exits. Human prose
  never mixes into stdout.
- The object always carries `ok`, and on failure an `error` that names the
  problem. Warnings ride in `warnings` and are never dropped: a thing gdoc could
  not do is reported, not omitted.
- Exit code 0 when `ok` is true, non-zero otherwise.
- The binary never prompts and never reads stdin. It must run headless.

## Auth

OAuth only. The bundled Internal client carries two rules: the secret stays in
version control, and the client stays User type Internal. Token and config live
in `~/.config/gdoc-agent/` on macOS and `%AppData%\gdoc-agent` on Windows, never
in a repo, and `GDOC_CONFIG_DIR` overrides both, which is what the tests use.
`auth login` is a real desktop flow: a loopback redirect with PKCE, hand-rolled.

**Two scopes:** Drive, and `https://www.googleapis.com/auth/documents`, the Docs
read/write scope, because v2 writes suggestions through the Docs API. Read-only
would be enough to read a document and not enough to suggest a change to one,
so the wider of the two is the one asked for. `auth status` never guesses either: being signed out is an answer
and comes back `ok: true`, a token file that exists and cannot be read is a
failure naming the file, and a token carrying less than v2 asks for is reported
and never refused, because the login worked.

## The guard

Serves principle 3. This is the safety property everything else stands on.

- **One package owns the network.** `internal/guard` is the only package that may
  build an HTTP client or reach a package-level dialer such as `http.Get`. Naming
  `net/http` is not dialing with it, so two allowlists draw the line: one over
  who may import the type, a narrower one over who may build a client. Both fail
  in both directions.
- **Two doors into the id set:** ids handed in on the command line, and ids
  learned from a create the guard itself carried. `files.list` is refused
  outright. An empty set refuses everything.
- **Every id carries a write level:** an id learned from a create is fully
  writable, because gdoc made it; a handed-in id is read and suggest only, never
  direct-editable; and the styling grant is the one exception, given for one id
  and one run, never inherited and never remembered.
- **Five grants stand beside the set**, each naming one object for one run,
  never inherited and nowhere written down: `AllowCreateIn`, `AllowReject`,
  `AllowCopy`, `AllowMarker`, and `AllowUpdateFrom`, which is the fifth and the
  only one that is not about a Google file. It names one GitHub repository and
  admits GET on that repository's releases listing, its download path, and the
  two asset hosts a download redirects to. The listing carries one query
  parameter, `per_page`, held to GitHub's own maximum of 100, and nothing else;
  the download path carries none. No request to any of those hosts carries a
  credential, because the only bearer gdoc holds is Google's, and the wire
  refuses one on the host rather than on the grant. A policy nobody granted an
  update refuses every one of those hosts by name, and nothing the grant admits
  is a document, so the reachable set is untouched. `gdoc update` is what opens
  it, and nothing else does. Added 2026-09-16, DECISIONS.md.
- **The level-1 write bar is what gdoc asks for, not what the server is known to
  enforce.** What holds a handed-in document to suggestions is
  `writeControl.writeMode == "SUGGEST"`, a field the client supplies, absent from
  the public Docs discovery document and once measured returning 200 while making
  a direct edit. What closes that gap is the probe before a proposal and the
  read-back after it, and neither is in the guard.
- **The guard judges the request it sends, never a tidier version of it.** A
  method override, a credential in the URL, a path walking through `.` or `..`,
  and a body whose parents it cannot read are refused. The query and the headers
  are allowlists, one per call shape, because Google gives one capability several
  spellings and a denylist is a patch behind each new one.

## Verification: never trust a success

These are requirements, not advice. Each has a measured 200-that-lied behind it.

- **Capability probe.** On every `propose` invocation, before its first
  suggest-mode write, probe `writeMode: SUGGEST` against a throwaway document.
  One invocation may batch many proposals behind it, and nothing caches it.
- **Read-back after every proposal.** The change is absent from the read made
  with `PREVIEW_WITHOUT_SUGGESTIONS`, so it is a suggestion rather than an edit,
  and the span reads as intended. Index arithmetic is the hazard: a 200 once
  placed text one character before a full stop.
- **`commentUpdateState`, not the status code**, on every write that carries a
  comment. `ALL_SAVED` is success; anything else is the partial failure it is.
- **The docx export is the honest witness** for comment attachment, because
  Drive's `anchor` and `quotedFileContent` survive detachment and prove nothing.
- **Always read with `includeTabsContent=true`**, because reading without it
  silently sees one tab. A write target with more than one tab stops the command:
  tabs are unsupported, reported as such, never guessed at.

## Commands

### `publish`

Builds a document that does not exist yet. Renders the hub markdown plus
`house.yaml` to a docx and uploads it with conversion into the folder it was
given. One upload, and no measuring pass, because the docx carries live fields.
Three things come free for that reason, where a restyle reports them as manual:
the positioned logo in a first-page header, the refreshable contents list, and
footer page numbers. After upload it verifies the document exists, records the
document id, the folder and the publish record in the source file's front
matter, and reports every file it changed on disk.

**It runs once per document, and there is no republish.** Later hub-to-doc
changes travel as suggestions, and a document whose review has run its course is
published again from its note, because the note is the source of truth:
`publish` refuses a paired note, and its refusal says to take the `gdoc:` block
out by hand when it names a document that has gone. Anything more is alignment's, which
is deferred to the backlog. Changed 2026-09-11 and 2026-09-16, DECISIONS.md.

### `restyle`

Puts a document gdoc did not create into the house style. One command and three
modes: the survey, the styling, and the house prelude. There is no mode that
renders the document to a docx and uploads it as a new file. Changed 2026-09-11,
DECISIONS.md.

**The survey (`--dry-run`).** Reports what the document holds: threads with a
witness for each, pending suggestions, chips, tabs, named ranges and the revision
the reads were made against. It writes to no document and to no file. It and the
styling are two runs on purpose, and a run naming both flags is refused: the
survey is what makes the write safe, so a person has to have read it.

**The styling (`--from survey.json`).** `batchUpdate` on the original, at the
per-run write grant the guard hands out for one id. It keeps every thread with
its authors and anchors, every pending suggestion, every chip with its data,
every Drawing, and the URL. What it writes is typography: page geometry,
paragraph and text styling, table cell appearance. What it never writes is a
character of the author's text. **The limit is durability rather than fidelity**,
because `updateNamedStyle` does not exist: the document looks right and the next
heading the author types will not.

**The house prelude (`--fields fields.json`).** The cover, the three front-matter
tables and the legend, and every character of it is a **suggestion**. Two phases
against one document with two permissions: phase 1 proposes the prelude in
`writeMode: SUGGEST` on a policy that granted nothing, which is what `propose`
sends every day, and phase 2 is the styling above on a second policy holding the
grant. Nail accepts it in the browser or rejects it, and rejecting it leaves the
document as it was. Its thirteen cover values come from a fields file the skill
proposes and Nail confirms, because a restyle has no note. Changed 2026-09-10,
DECISIONS.md. The one thing written directly is the marker, a named range called
`gdoc:house-prelude` over what phase 1 proposed, which is how a second run finds
gdoc's own prelude and replaces it rather than adding a second cover. It is
written rather than suggested because `createNamedRange` is the one request Docs
refuses to apply as a suggestion, and it is safe because a named range adds and
removes no character. `Policy.AllowMarker` is its door: per-run, one range.

**What gdoc cannot do it reports, with the menu path for each, and it writes no
checklist into the document.** Writing that page needs `insertText` and
`createParagraphBullets` back on the styling level's allowlist, which is the
whole of what keeps a restyle unable to change a character. The list is the
first-page header with the logo, the contents list, the footer page numbers, the
front-matter tables' column widths and row heights, list styling, and a named
style the house has no look for, and BLOCKED-BY-API.md holds why the first four
cannot be asked for at all. Heading numbering is out for a reason of its own: it
is `insertText`, which can be proposed, and what keeps it out is idempotence
and placement.

**Nothing to protect is a fact, never an offer.** A document with no threads, no
pending suggestions, no chips, no unnamed element and no named range has nothing
a restyle could destroy, and the survey reports `nothing_to_protect`. What the
skill offers over that fact is the two-step route: `read` the document, put the
text in a note, `publish --md`. It never takes it silently, because a new URL is
a person's choice. Changed 2026-09-11, DECISIONS.md.

### Reading comments

Lists threads with real character ranges (`commentsViewMode`, which requires
`includeTabsContent=true`). Takes a `--since` cursor and reports only activity
after it, so a live session's poll is one cheap call. `--wait` makes that call
poll: it repeats the same listing every ten seconds and returns the first window
with activity in it, or an empty one at the deadline. It needs `--since` and
keeps no state of its own. `waited` says how many times it asked, how long it
looked and whether a signal ended it: an interrupt is an answer, and a failed
poll is `ok: false` with the polls so far, so the skill can tell an unread window
from an empty one. The cursor is opaque, the binary emits it and the caller hands
it back, and it dies with the session. Anything that must survive across sessions
lives in the front matter, nowhere else. Each comment is reported with its
marker: `ai:`, `ai?`, `ai!`, or none. An unmarked comment, including an unmarked
follow-up in a thread gdoc has answered, is reported and never acted on. The
marker is the trigger, always: stickiness carries context, never authority.

**Whether a comment is answered is the skill's judgement, and no rule in the
binary.** The binary reports the facts of a thread: every reply, its author, its
time, and whether it opens with 🤖. A 🤖 reply newer than the comment is not an
answer, because it may be an acknowledgment and the follow-up may have changed
what was asked. **The 🤖 reply is the receipt, and the document is the ledger**:
no local record of handled comments exists, so any session recomputes the open
work from the threads alone. The receipt is written last, after the action
landed, and before re-acting on an old marked comment the skill checks the
pending proposals and their provenance, so a crash in between resolves to writing
the missing receipt rather than to acting twice.

### `reply`

Posts into an existing thread. Plain text only. Every reply gdoc writes opens
with 🤖 and nothing else: no name, no prefix text. `commentUpdateState` is
checked. gdoc never resolves and never reopens a thread, even though the API can:
resolving means the answer was accepted, and only Nail accepts. **A long
action writes at most two messages into its thread**, as two replies rather than
edits of one: an acknowledgment when the work will take noticeably long, and the
receipt when the action verifiably landed. Never a progress feed,
because the margin is the readers' room and the terminal is the operator's. A
failed action replaces the receipt with a plain failure naming the reason, so a
thread ends in a true statement rather than a dangling "on it".

### `propose`

Writes a change as a native Google suggestion (`writeMode: SUGGEST`) with an
anchored comment on the exact words explaining why, opening with 🤖. Subject to
the capability probe and the read-back above. On a handed-in id this is the only
write level the guard allows, which is the point, and an `assigneeEmailAddress`
may be set when the skill knows who should answer.

**One writer per document.** Work fans out across documents freely, and replies
within one document may run in parallel, because they target thread ids rather
than indexes. Proposals within one document are serialized: a `batchUpdate` lands
at indexes computed from a read, so a concurrent proposer shifts the ground under
the other and the API reports 200 either way. Each proposal re-reads before
computing its indexes and reads back after landing.

### Withdrawing a proposal

gdoc may retract its own pending, unaccepted suggestion: `rejectSuggestion`
naming the id, in suggest mode, confirmed by `rejectedSuggestionIds` in the
response and a read-back showing no run under the id on either side. A reject
takes the whole proposal back, both the suggested deletion of the quoted words
and the suggested insertion of the new ones, which a `deleteContentRange` cannot
do. Changed 2026-09-07, DECISIONS.md. This does not touch Nail's rule, because
there was no decision yet to resolve. gdoc never accepts, rejects or deletes
anyone else's suggestion, and the guard holds that: it refuses every request kind
whose name carries "suggestion", and the one door is a per-run grant `withdraw`
seeds with the id the note's `proposals[]` records as gdoc's own.

### Reading suggestions

Lists pending suggestions with their stable ids and quotes what each proposes. On
a later read it reports which ids stopped being pending, as
`gone_since_last_look`: the id, what it said at the last snapshot, and when that
snapshot was taken. **Whether one was accepted or thrown away is not reported,
because it is not in the API**: the id is gone either way, so the skill reads
`read`'s text and decides.

### The diff (what `align` composes)

The binary emits the document's current content in a form the skill can read
(`read`: headings, paragraphs, tables, pending suggestions inline with their ids,
comment anchors marked), and it reads the document only. Making it read markdown
too would add coupling and do nothing for an external document. It records no
content baseline either: the front matter holds facts, which are the pairing, the
publish record and gdoc's own proposals. The skill reads the hub markdown from
disk and composes the comparison. Whether a difference matters is its judgement,
and Nail can tell it which side is the source of truth. Alignment runs both ways, writes to the hub only with agreement,
and never deletes from it: content that exists only in the hub is unpublished
work, not drift. It also works on a document gdoc never published, where the
person supplies the context instead of a paired note.

## The generator and `house.yaml`

`house.yaml` is the house style: page geometry, all nine named styles, the cover,
header and footer, the three tables cell by cell, the contents field, heading
numbering, and the logo as base64. The generator reads it plus markdown and emits
a docx.

**The front matter's page breaks are blocks in that file**, `- block:
page_break`, one after the cover and one after the classification table, beside
`cover`, `label`, `table`, `blank`, `legend` and `toc`. One layout, two writers:
the docx renderer emits the block as a paragraph carrying `pageBreakBefore`, the
prelude emits it as `insertPageBreak`, and neither holds a page break of its own
any more. The drift gate reports nothing new for it, because no item in the
measured list reads a page break or counts a front-matter paragraph. Added
2026-09-16, DECISIONS.md. Nothing reads the master `.docx` at runtime: it stays in the repo as
provenance and as the drift test's fixture. **The comparison is the acceptance
gate, and it runs both ways.** The list of
169 measured values is written once, with one reader per item: offline that
reader is given the docx XML, live it is given a Docs API answer, so a row that
fails in one gate is findable in the other. Offline runs in `make test` on every
commit; live is opt-in, uploads both documents with conversion, and is the
measurement that means something, because Google's import is part of the result.
Either way a difference not named in the list of known ones, with its reason,
fails the gate. No item reads a PDF. House-style tests state house values as
literals, never by reading the constant they test.

## Install and update

**The version is `x.y.z`, and the number is the channel.** Nail cuts `x.y.0` by
hand with `make tag`; the nightly cuts `x.y.(z+1)` when main has moved since the
last tag. `z == 0` is stable, `z > 0` is nightly, and nothing else records which
is which. The linker sets it from the tag, so a binary built from a checkout is
`dev` and prints no version at all: `version` rides in the envelope only when
there is a release behind it, and `gdoc help` and `gdoc auth status` read the
same value. The plugin carries the same number, and a release whose tag and
`.claude-plugin/plugin.json` disagree is refused by the workflow. Builds are
trimmed and stripped, so no home path ships and one tag is the same bytes on
every machine.

**A release is one GitHub Release of this repository**, which is public since
2026-09-16. A tag builds it: the four checks every push runs, then `make dist`
with the client secret from a repository secret, then one zip per line of
`release/platforms`, named `gdoc-<tag>-<platform>.zip`, and one
`SHA256SUMS-<tag>` over all of them. A zip carries five things: the binary, the
user `README.md`, `install.sh`, `skills/` and `example/`. `release/platforms` is
`darwin-arm64` and `darwin-amd64`, and Windows joins it under its own tag once a
colleague has run its checklist.

**The installer is one script with two entrances.** From an unpacked zip it
installs the binary beside it; from `curl -fsSL .../release/install.sh | bash`
it asks the releases API for the newest stable tag, downloads the zip for this
machine and its checksum file, verifies it, and installs out of a temp dir.
Either way it copies the binary to `~/.local/bin/gdoc`, strips quarantine,
writes the completion into the config dir, prints the `source` line when
`.zshrc` lacks it, prints the two `/plugin` commands, and ends with `gdoc auth
status`. It asks nothing and it edits no shell file.

**The skills travel as a Claude Code plugin.** This repository carries
`.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json`, so it is
both the plugin and the marketplace, and `skills/` stays where it is. A
colleague runs `/plugin marketplace add nhusnullin/gdoc` and `/plugin install
gdoc@gdoc`, chooses global or per-project in Claude Code's own terms, and
updates on that marketplace's toggle. Two fallbacks exist by policy, because a
managed Claude Code can refuse a marketplace and can refuse personal and project
skills with it: `install.sh --skills global|local` copies `skills/` out of the
zip, marked so a re-run replaces only what it wrote, and an administrator names
this marketplace in `extraKnownMarketplaces` and turns the plugin on in
`enabledPlugins`. A checkout still links its skills and never copies them, and
nothing in the binary copies a skill folder.

**`gdoc update` runs when a person types it and never otherwise.** No check when
a session starts, no scheduler, no stamp file. It reads one page of releases,
asking for a hundred, because GitHub answers thirty by default and the nightly
would push the last hand-cut stable release off a page that size in about a
month. What it takes: the same `x` with
a higher `y` by default, a higher `x` only with `--major`, a higher `z` only
with `--nightly`, and never a version below the one installed. `--check` reports
what a run would take and writes nothing; `--rollback` puts the previous binary
back. The zip is verified against the release's own checksum before a byte is
replaced, the binary that was there is kept beside the new one, and the object
names one of `up_to_date`, `updated`, `major_available`, `checked`,
`unreachable` and `rolled_back`, with `run` naming the command that would go
further where there is one. Unreachable is `ok: true` with a warning, because a
network that is not there is not a failed update. The command holds no state
between runs.

## The skills, and how a comment reaches one

Four named workflows, all judgement rather than commands. Three of them are
skills today, one copy in this repo: symlinked into a checkout, and installed on
a colleague's machine as the plugin above. They are the review session, the
publish run and the restyle run. The alignment check is described here and
is deferred to the backlog, 2026-09-16, DECISIONS.md. Two of the four run on a marked comment, and two Nail invokes by
name:

- **The review session.** Reads the threads, answers `ai?` from the hub, carries
  out `ai!` against the hub, proposes document changes as suggestions. If it
  could not read half the review it says so and does not report clean. It can run
  **live**, polling on the `--since` cursor until Nail stops it. A colleague's
  `ai!` acts too, by decision: the marker is the trigger, identity is not a gate,
  and the guard caps a handed-in document at suggest and reply.
- **The alignment check**, deferred to the backlog on 2026-09-16, not built. Composes the diff, judges what
  matters, proposes both ways: suggestions into the document, edits into the hub
  with agreement.
- **The publish run**, `gdoc-publish`. Nail names a note, and the skill builds
  it, publishes it into the folder, and reads back what came out.
- **The restyle run**, `gdoc-restyle`. Nail gives a link, and the skill surveys
  the document before it proposes anything to it.

**A skill names the binary version it needs**, as `needs:` in its own front
matter. Its setup runs `gdoc help` first and reads `version` from the object,
and an older binary stops the skill with one sentence naming `gdoc update`. No
`version` at all is a build made from source rather than a release, which is not
an error: the skill says so once and carries on. Skill and binary may drift by a minor
version without harm, because a skill holds no flag list, which is the next
rule. Added 2026-09-16, DECISIONS.md.

**No skill holds a flag list.** Before the first call of a command in a session
a skill runs `gdoc help <command>` and reads the words and flags from the binary
it is about to run. `gdoc-publish` and `gdoc-restyle` do this, and `gdoc-review`
is the one that still names its flags, until it is converted. One direction
holds for all three meanwhile:
`TestEverySkillNamesOnlyCommandsAndFlagsTheBinaryHas` reads the call lines, the
ones opening with `gdoc` or `$GDOC`, and fails when one names a command the
table does not have or a flag that command does not take. A flag named in prose,
or standing on a continuation line under the call, is not covered, so a rename
in the table can still leave a skill telling a session to pass it. Added
2026-09-16, DECISIONS.md.

**There is no watcher, and the loop starts when Nail starts it**, in the hub
folder. A comment written today waits, visible in the document, until a session
looks. The hub being the working directory is what gives the agent its context:
the notes, the decision log, the raw material. The front matter is the join: each
source markdown names its document id, so the skill walks the folder, finds the
documents that belong to it, and reads their threads. Then `ai?` is answered from
the hub and `ai!` is carried out against it, and if that needs a note changed or
a file written it happens in that session. Nothing is queued.

**A live session keeps the polling inside the call**, which is `comments --wait`
above. A call per poll would make the latency a minute rather than seconds,
because the tool that runs the command cannot wake more often than that, and
every idle tick would spend a model turn on a quiet document. The loop over those
calls is the skill's, which asks for nine minutes because its Bash tool gives up
at ten. Liveness ends with the session. Changed 2026-09-07, DECISIONS.md.

## Never

The hard list, transport-enforced where the guard can reach it and a skill rule
where it cannot.

- Never direct-edit a document gdoc did not create, **except the one document a
  restyle was granted, for the one run it was granted in**. Guard-enforced both
  ways: a handed-in id refuses a direct edit, and the grant raises one id to the
  styling level, where four styling request kinds carry, anything else is refused
  whatever it is called, and none of the four can change a character.
  `createNamedRange` is the fifth, over the one range `AllowMarker` granted, and
  it adds and removes no character too. Changed 2026-09-09 and 2026-09-10,
  DECISIONS.md.
- Never replace the body of a document that exists. Guard-enforced.
- Never accept, reject or delete anyone else's suggestion. Guard-enforced: the
  only `rejectSuggestion` that carries names the id `withdraw` granted from the
  note's `proposals[]`, which is gdoc's own.
- Never resolve or reopen a comment thread.
- Never delete from the hub.
- Never run git, in the binary or in the skills.
- Never prompt, in the binary, and never in the installer either.
- Never update unasked. `gdoc update` runs when a person types it and at no
  other moment.
- Never export a PDF. Nail downloads it from the browser.
- Never write to a multi-tab document.
- Never write markdown into a comment thread.
- Never trust a status code where the write matters.
- Never write a comment or reply that does not open with 🤖.

## Acceptance for v2.0

Each is a test or a checkable run, not a claim.
1. The guard test: HTTP construction outside the guard package fails the build;
   a request for an id outside the set is refused; a direct edit on a handed-in
   id is refused unless the run granted that one id the styling level, and there
   every kind but the four styling ones is refused, as is a `fields` mask of `*`,
   an empty mask or one naming either header toggle. `createNamedRange` carries
   only over the range `AllowMarker` granted.
2. The drift test: config-rendered and master-rendered documents compare across
   the 169 items with no new differences.
3. A publish produces a document with the positioned logo, a live contents list
   and footer page numbers, verified by export.
4. A propose lands as a genuine suggestion, absent under
   `PREVIEW_WITHOUT_SUGGESTIONS`, with its 🤖 comment anchored to the exact words,
   verified through the docx export.
5. An in-place restyle on a copy of the ideal test document preserves all ten of
   its features, the chips, the Drawing, the footnotes, the lists, the pinned
   shaded table header, the byte-identical inline image, the anchored comment and
   the pending suggestion, and the original is untouched, same `revisionId`.
6. The capability probe distinguishes an enrolled project from an unenrolled one
   without writing to any real document.
