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
  Claude Desktop chat is a third door, through `gdoc mcp` below, and the one
  with no hub behind it, which is why it never carries out a marker.

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
otherwise: PDF export and any `release` command, a daemon or watcher, accepting
or rejecting suggestions, resolving or reopening threads, anchor banking, MCP as
the transport to Google, which stays the REST client behind the guard, the
colour marking scheme, service accounts, running git. Tab
features are out for every writer, which refuses a document with more than one
tab, and `export` is the one reader that reads them, one file per tab. Narrowed
2026-09-19, DECISIONS.md.

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

One command table in `cmd/gdoc/commands.go`, sixteen rows since 2026-10-03,
describes every command once: the
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

- Every command writes exactly one JSON object to stdout and exits, except that
  `gdoc mcp` writes JSON-RPC lines there, and a help screen on a terminal
  without `--json` writes nothing there: `help` when it answers and bare `gdoc`
  draw Panels on stderr instead, and `--json` prints the object wherever stdout
  goes. Every other result keeps its object, refusals and the panic envelope
  included. Narrowed 2026-10-03, DECISIONS.md. Human prose never mixes into
  stdout.
- The object always carries `ok`, and on failure an `error` that names the
  problem. Warnings ride in `warnings` and are never dropped: a thing gdoc could
  not do is reported, not omitted.
- Exit code 0 when `ok` is true, non-zero otherwise, and when a help screen
  replaced the object, 0 for `help` and 1 for bare `gdoc`.
- The binary never prompts, and every command but one never reads stdin. It
  must run headless. `gdoc mcp` is the exception: a JSON-RPC session arrives
  there, and that command prints no envelope at all. Narrowed 2026-10-03,
  DECISIONS.md.

## Auth

OAuth only. The bundled Internal client carries two rules: the secret stays in
version control, and the client stays User type Internal. Token and config live
in `~/.config/gdoc-agent/` on macOS and `%AppData%\gdoc-agent` on Windows, never
in a repo, and `GDOC_CONFIG_DIR` overrides both, which is what the tests use.
The settings file, `settings.yaml`, lives in the config folder; `--config
<file>` names another, and an argument such as `--mark` beats the file. The
token is never a setting (2026-10-04, DECISIONS.md; the token moves to the
macOS Keychain in M17).

**The presence is a file gdoc appends to and never reads.** Under the
`presence` setting, every command that names a document, `update` and `auth
login` write a start and an end line, and a chat hold writes held, released
or expired, into `presence/<yyyy>-<mm>.jsonl` in the config directory. The
line is a fixed projection, never the envelope: no document word but a title,
no quote, no hold text, no error sentence, no path, no account. The binary
exports no reader and nothing it decides depends on the file. A separate
application, `GdocPresence.app`, travels in the darwin zips and is installed
by `gdoc update` under the same setting. Off is today's bytes. Decided
2026-10-17, DECISIONS.md; the launch rule is the 2026-10-18 entry below.

**The panel, the same day.** The app draws the screens Nail chose and runs
four fixed argv lists for the binary, `help --json`, `auth status`, `auth
login`, `auth logout`, `update` and `update --desktop`, and nothing else; it
signals one SIGINT to a `comments --wait` it has proved is gdoc's. The record
gains `made`, `step`, `stop` with `line`, `via: code`, `interrupted` and the
`dropped` hold ending, still no path, sentence, count or account. A hold
remembers a hash of the refresh token it was held under and is dropped when
the sign-in changes, so no account ever posts another's held write. `gdoc
auth logout` removes the token and nothing else. The app relaunches itself
after an update. Decided 2026-10-17, DECISIONS.md, the panel entry.

**The app opens itself.** Under the setting, on macOS, a gdoc command whose
run is recorded opens `~/Applications/GdocPresence.app` with `/usr/bin/open
-g` when the bundle is there and the app does not hold its lock, and does not
wait. Every command but `gdoc update`, which is the one that moves the bundle:
it opens the app itself once, after it placed or rolled it back, so one run
opens the app once and never the build it is replacing. The app
holds `flock` on `presence/app.lock` for its life; gdoc asks the lock without
blocking and reads no byte. Launch at login is off until the person turns the
SETTINGS switch on; no run registers it by itself. Decided 2026-10-18,
DECISIONS.md.

**The second morning.** The header carries the account, its state and Sign
out, the gdoc version and its update; the bottom group is SETTINGS, with the
Hub row that chooses the folder Open md reads, and no Thread mark row. The header, NOW and SETTINGS never scroll and the
middle does. DOCUMENTS lists pinned documents first, then today's newest
first, then the fourteen days by use, and never a hidden one until gdoc
touches it again; pins and hides are the app's preferences. The
mark at rest is full ink. Under `via: code` the start line carries
`session`, Claude Code's own session id, and Open chat opens
`claude://code/continue?session=<id>` where that is measured to work.
Decided 2026-10-18, DECISIONS.md.

**The third look, the same evening.** Nothing in the panel scrolls: it is as
tall as its content. DOCUMENTS is the pinned documents and then the last
five, one row each, hidden ones skipped; the pin is the row's fifth icon and
Hide is in its menu; TODAY and Yesterday are gone. Open md finds the note
through Spotlight and the `gdoc:` block, with `hubPath` as an override and
no Hub row. The update glyph is always in the header, quiet, available or
running, and the app asks `help --json` on open and every six hours.
Decided 2026-10-18, DECISIONS.md, the third look.

`auth login` is a real desktop flow: a loopback redirect with PKCE, hand-rolled.

**Two scopes:** Drive, and `https://www.googleapis.com/auth/documents`, the Docs
read/write scope, because v2 writes suggestions through the Docs API. Read-only
would be enough to read a document and not enough to suggest a change to one,
so the wider of the two is the one asked for. `auth status` never guesses either: being signed out is an answer
and comes back `ok: true`, a token file that exists and cannot be read is a
failure naming the file, and a token carrying less than v2 asks for is reported
and never refused, because the login worked.

**One sign-in at a time, across processes.** Claude Desktop runs two server
processes at once, so a second `login` would open a second loopback listener on
a second port and only one of them could win. A waiting sign-in is written down
beside the token, in `login-pending.json`, with the pid, the port and the link,
and a second process hands on that same link and reports its state rather than
starting a trip of its own. A record whose pid is dead, or older than the three
minutes the listener itself lives, is removed and the next call starts fresh. A
record that is there and will not parse is a failure naming the file, never
read as nothing waiting. The states are `waiting`, `signed in` and `expired`,
and `signed in` names the Google account, read through `AllowAccountRead`
above, so a swapped account is seen. Added 2026-10-03, DECISIONS.md.

## The guard

Serves principle 3. This is the safety property everything else stands on.

- **One package owns the network.** `internal/guard` is the only package that may
  build an HTTP client or reach a package-level dialer such as `http.Get`. Naming
  `net/http` is not dialing with it, so two allowlists draw the line: one over
  who may import the type, a narrower one over who may build a client. Both fail
  in both directions.
- **Two doors into the id set:** ids handed in on the command line, and ids
  learned from a create the guard itself carried. `files.list` is refused
  outright. An empty set refuses everything. A third door admits one object
  that is not a document, the house logo, under `AllowAsset` below, and the
  set cannot grow through it. Added 2026-10-17, DECISIONS.md.
- **Every id carries a write level:** an id learned from a create is fully
  writable, because gdoc made it; a handed-in id is read and suggest only, never
  direct-editable; and the styling grant is the one exception, given for one id
  and one run, never inherited and never remembered.
- **Seven grants stand beside the set**, each naming one object for one run,
  never inherited and nowhere written down: `AllowCreateIn`, `AllowReject`,
  `AllowCopy`, `AllowMarker`, `AllowUpdateFrom`, `AllowAccountRead` and
  `AllowAsset`.
  `AllowUpdateFrom` is the fifth, and the only one that reaches a host that is
  not Google's. It names one GitHub repository and admits GET on that
  repository's releases listing, its download path, and the two asset hosts a
  download redirects to. The listing carries one query parameter, `per_page`,
  held to GitHub's own maximum of 100, and nothing else; the download path
  carries none. No request to any of those hosts carries a credential, because
  the only bearer gdoc holds is Google's, and the wire refuses one on the host
  rather than on the grant. A policy nobody granted an update refuses every one
  of those hosts by name, and nothing the grant admits is a document, so the
  reachable set is untouched. Three callers open it: `gdoc update`, the
  once-a-day check in `gdoc help`, which reads the listing and replaces
  nothing, and the same check in `gdoc mcp`. Added 2026-09-16, the second
  caller 2026-09-18, the third 2026-10-03, DECISIONS.md.
- **`AllowAccountRead` is the sixth, and it reads which account is signed in.**
  It admits `GET https://www.googleapis.com/drive/v3/about` with
  `fields=user(emailAddress,displayName)` and nothing else, under the scopes the
  token already has, addressing no document. Two readers open it, both through
  `accountOf`: `login` inside `gdoc mcp`, so the `login` answer can name the
  account and a swapped account is seen, and `gdoc auth status`, on every run
  that finds a token with every scope. No other command gains it, and every
  other `about` request keeps its refusal. Added 2026-10-03, the second reader
  in M15, DECISIONS.md.
- **`AllowAsset` is the seventh, and it reads the house logo.** It names the
  one Drive file id `house.yaml` carries as `logo.drive` and admits
  `GET https://www.googleapis.com/drive/v3/files/{id}` with `alt=media` and
  `supportsAllDrives=true`, under the `drive` scope the token already has,
  and nothing else on that id: no metadata read, no Docs read, no patch, no
  copy. Two commands open it, `build` and `publish`, the two that write a
  docx, and each asks Drive at most once a day through the cache beside the
  token. `export`, `restyle`, `help marks` and `read` never reach it. A PNG
  carries no id, so the reachable set is untouched. Added 2026-10-17,
  DECISIONS.md.
- **The level-1 write bar is what gdoc asks for, not what the server is known to
  enforce.** What holds a handed-in document to suggestions is
  `writeControl.writeMode == "SUGGEST"`, a field the client supplies, absent from
  the public Docs discovery document and once measured returning 200 while making
  a direct edit. What closes that gap is the read-back after a proposal and the
  stop at the first one it cannot confirm, and neither is in the guard.
- **The guard judges the request it sends, never a tidier version of it.** A
  method override, a credential in the URL, a path walking through `.` or `..`,
  and a body whose parents it cannot read are refused. The query and the headers
  are allowlists, one per call shape, because Google gives one capability several
  spellings and a denylist is a patch behind each new one.

## Verification: never trust a success

These are requirements, not advice. Each has a measured 200-that-lied behind it.

- **Capability probe, by hand only.** `gdoc probe --folder` asks whether
  `writeMode: SUGGEST` is honoured today, against a throwaway document, and
  nothing caches the answer. It ran before every proposal until 2026-10-02, when
  suggestions reached general availability; the read-back below and the stop are
  what close the gap now, and the entry of that date in DECISIONS.md holds why.
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

**It runs more than once, and each run makes a new document.** A paired note
publishes again: the run appends an entry to the `gdoc:` block, and the old
document keeps its URL, its threads and its own entry untouched. It goes stale
the moment the new one exists, which is the one thing the publish skill says
before it runs. Publishing into the document that is already there is refused,
because gdoc never replaces a body: that request is a merge, and a merge travels
as suggestions through `gdoc-align`. Changed 2026-09-19, DECISIONS.md.

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
`publish` writes a second marker, `gdoc:house-published`, over its own cover
from index 1 to the end of the contents list, and every restyle walks past the
span either marker covers. `restyle --fields` refuses a document carrying the
published one, whose cover is already the house cover. Changed 2026-09-27,
DECISIONS.md.

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
poll: it repeats the same listing every two seconds and returns the first window
with activity in it, or an empty one at the deadline. It needs `--since` and
keeps no state of its own. `waited` says how many times it asked, how long it
looked and whether a signal ended it: an interrupt is an answer, and a failed
poll is `ok: false` with the polls so far, so the skill can tell an unread window
from an empty one. The cursor is opaque, the binary emits it and the caller hands
it back, and it dies with the session. The interval was ten seconds until
2026-09-18, DECISIONS.md. Anything that must survive across sessions
lives in the front matter, nowhere else. Each comment is reported with its
marker: `ai:`, `ai?`, `ai!`, or none. An unmarked comment, including an unmarked
follow-up in a thread gdoc has answered, is reported and never acted on. The
marker is the trigger, always: stickiness carries context, never authority.

**Whether a comment is answered is the skill's judgement, and no rule in the
binary.** The binary reports the facts of a thread: every reply, its author, its
time, and whether it opens with a known mark (`by_gdoc`). A marked reply newer
than the comment is not an answer, because it may be an acknowledgment and the
follow-up may have changed what was asked. **The marked reply is the receipt,
and the document is the ledger**:
no local record of handled comments exists, so any session recomputes the open
work from the threads alone. The receipt is written last, after the action
landed, and before re-acting on an old marked comment the skill checks the
pending proposals and their provenance, so a crash in between resolves to writing
the missing receipt rather than to acting twice.

### `reply`

Posts into an existing thread. Plain text only. Every reply gdoc writes opens
with the mark in effect and nothing else: no name, no prefix text. gdoc adds the
mark; the default is ⁘, a person may set any other or none, and 🤖, ⁘ and ∴ stay
known marks (changed 2026-10-04, DECISIONS.md). `commentUpdateState` is
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
anchored comment on the exact words explaining why, opening with the mark. Subject to
the read-back above, and the run stops at the first proposal either the inline
read or the preview read cannot confirm. On a handed-in id this is the only
write level the guard allows, which is the point, and an `assigneeEmailAddress`
may be set when the skill knows who should answer.

**One writer per document.** Work fans out across documents freely, and replies
within one document may run in parallel, because they target thread ids rather
than indexes. Proposals within one document are serialized: a `batchUpdate` lands
at indexes computed from a read, so a concurrent proposer shifts the ground under
the other and the API reports 200 either way. Each proposal re-reads before
computing its indexes and reads back after landing.

**Two kinds in the proposals file.** An entry with no `kind` changes words
inside one paragraph: `{quoted, replacement, why, assignee?}`, and
a line break on either side is refused. An entry with `kind: block` adds or
replaces whole paragraphs. It names exactly one placement, either `after` with a
quote in the paragraph the block follows, or the pair `replace_from` and
`replace_to` which covers whole paragraphs from the start of the first to the
end of the last. It carries `content` as markdown: paragraphs, `#` to `######`
headings, `-` and `1.` lists one level deep, bold, italic and links. Every quote
is found exactly once, by the same span walk the words kind uses. A block is one
SUGGEST batch and comes back with one suggestion id, so `withdraw` takes it back
whole. Changed 2026-09-28, DECISIONS.md.

**What a block refuses, by name.** Content the subset does not hold: a table, a
nested list, a numbered list that does not begin at 1, a picture, a code block, a
block quote, raw HTML, a page break (`---`), and empty content. A character the Docs API strips out of an inserted text, and
a line break inside a paragraph, each in whichever spelling it arrives in: the
character itself, or an entity reference such as `&#12;` that decodes to one.
gdoc counts what it sends to place everything after it, so a unit the server
drops moves every later index, and a decoded newline is a paragraph the block
never proposed. A placement that is not exactly one form, a quote that is not
there exactly once, a `replace_to` before its `replace_from`, an anchor inside a
table cell, and a replace whose run reaches the body's last paragraph, because
Docs cannot delete the final newline. A replace over a paragraph carrying
anything but words, named: a picture, a drawing, an equation, a footnote mark, a
break or a chip in its text, or a picture, a drawing or an object floating
beside it, because the deletion takes any of them and no read-back reads
anything but text. A replace whose paragraphs hold a pending
suggestion or a comment's anchor, with every suggestion id and comment id named,
so the deletion never takes a colleague's work with it. An `after` whose next
element is a table, a contents list or a section break, because there is no
paragraph start to insert at. And an `after` on the document's last paragraph
when the block does not end in a plain paragraph, or when that last paragraph is
a list item or carries a named style other than body text, because there the
last new paragraph owns the existing final mark and the batch restates nothing
on it.

**Three read-backs for a block**, on routes the write did not go out on: the
inline suggestions read carries the insertion id on every new paragraph, and a
deletion id on every replaced one; the preview read without suggestions still
shows the anchor or the replaced run as it was and not the block's first new
line, which is the direct-edit catch; and the docx export carries the robot
comment anchored to text. A line that already appears elsewhere in the document
makes the preview check answer "no answer" rather than a pass. The first new
line is the block's own first line for an `after` block, and for a replace the
first line the replaced paragraphs did not already carry: a direct edit whose
new paragraphs carry the old ones inside them leaves every replaced paragraph
findable, so the run going missing does not catch that one on its own, and a
rewritten section is never failed for keeping its own heading. A block with no
new line left answers "no answer" as well, unless it drops a paragraph the
placement stood on, which is the replace that only shortens. More than one
suggestion id is reported in full and is `verified: false`, with a warning that
`withdraw` takes back the one the note records.

### `annotate`

Leaves one comment on the exact words a caller quotes, anchored to them, and
changes nothing else. Two input forms, exclusive: `--quote` with `--body-file`
for one comment by hand, `--from` for a file of
`{quoted, why, assignee?}`. The words are found in a document that just came
back, through the same span walk `propose` uses, so a quote that is not there
exactly once is refused. The reason arrives bare and gdoc writes the mark in
front of it; a reason already carrying a known mark is refused, as is markdown a thread
would render literally. Every entry is checked before the first one is sent.

**The probe does not run here**, and since 2026-10-02 it runs in no writer at
all. The batch holds one `insertComment` and nothing beside it, so no character
can move even if `writeMode: SUGGEST` were ignored, and the question the probe
asks never had a bearing on what this write can do wrong. Decided 2026-09-18,
and `propose` followed on 2026-10-02, both in DECISIONS.md. The guard is
unchanged: the batch is carried because it says SUGGEST, and refused without
it.

**Two read-backs, on routes the write did not go out on**: Drive's comment
listing must carry the returned id with the body that was sent, and the docx
export must wrap the quoted words in a comment range. Both holding, with a
write that answered `commentUpdateState: ALL_SAVED`, is `verified: true`.
Anything less is a warning naming the route, never an error, because the
comment exists.

A run stops at the first entry that cannot be sent: the envelope is `ok: false`
and every annotation is still reported with `sent` answered for itself. The
limits are `propose`'s: one tab, body text only, and no folder and no note,
because nothing here can be withdrawn. A wrong comment is removed by a person
in the document.

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

### `export`

Writes a Google Doc into the hub as Markdown: `gdoc export <url> --out <path>`.
The reverse direction of `publish`, and the fifteenth command rather than a flag
on `read`, so `help` and completion carry it and `read` stays a pure read.
Added 2026-09-19, DECISIONS.md.

It makes two requests and neither can change a document: the Docs read, which
carries the text with its suggestion and comment ids, its link targets and its
list numbering, and the docx export, which carries the picture bytes. It opens
no host the guard does not already open and asks for no grant.

**One file per tab.** The first tab lands at `--out`, each further tab beside it
as `<stem>-<tab title>.md`, the title lower-cased with every run of characters
that are neither letters nor digits as one hyphen. It is not the heading id
rule, which is goldmark's and drops an accented letter. Pictures follow
their own tab.

**Nothing is overwritten, and no flag could.** A taken path takes the next free
number, `<stem>.2.md` and then `<stem>.3.md`. Pictures land in `assets/` beside
the file as `assets/<stem>-<n>.png`, from the next free `n`, and the folder is
created when it is missing. There is no `--force`. The reply names every file it
wrote.

**The file holds what the document says, and decides nothing.** The text as
`read` prints it, with the same escaping and the same markers for a pending
suggestion and a comment anchor. Beyond `read`: link targets, list numbering,
pictures as files, and the house prelude taken out and listed with the text each
piece held, so an edit inside a cover table reaches the envelope rather than the
note. A prelude it cannot recognise stays in the file under a warning naming
what stood there. A footnote comes across as `[^n]` in the prose with its
definition after a `---` line, and that file publishes again: `build` sets
footnotes rather than refusing them (2026-10-14, DECISIONS.md). A label
referenced twice in the hub comes back as two notes, because a document holds
one note per mark. A mark the author wrote comes back as the syntax it left
as. A page break that stands on its own comes back as `---` on a line of its
own, which is the line the next build reads as that same break; the one on the
first body heading is left out, because `build` puts it back, and a break the
file has no line for, inside a sentence or alone in a heading, a list item or a
table cell, keeps its `[page break]` marker and its warning.
The front matter of a file `export` creates holds
the `gdoc:` block and nothing else: a skill adds `title` when it makes the file
a note.

**The one thing it writes into a file somebody else owns is a date.** When
`--out` holds a note whose block names this document, that note keeps every
other byte and gains `exported: {at}` on that document's entry, and the export
lands beside it carrying `exported: {note: <path>}`, which makes every writer
refuse the copy in one sentence pointing at the note. Refused at the door,
before a byte is written: a note at any path the run would land on that names
other documents, and front matter that does not read.

**The markers travel one way only.** A file holding one may sit in the hub as
long as a person likes. It cannot become a document: `build` and `publish`
refuse a marker by line, and so does every writer that takes text from a file,
`propose` through its `--from` read and `reply` and `annotate` through
`internal/plaintext`.

### The `gdoc:` block is a list of documents

`schema: 2`, with `documents` holding one entry per document the note has met,
each carrying its own `published`, `exported`, `folder_id`, `suggestions_seen`
and `proposals`. A command works on the entry whose id the URL names and refuses
a URL the list does not hold, naming the ids it does. A proposal and a snapshot
live under their own document, because a `withdraw` that sent an id from another
document would reject against the wrong file. `publish` appends an entry and
`export` stamps one, so `publish` is no longer the block's only writer.

A schema 1 block reads as a one-entry list and is written back byte for byte
while nothing changes it. The first write that changes anything renders schema
2, and the reply says so once. A gdoc from before this milestone meets a schema
2 block with its ordinary strict-read refusal,
`gdoc front matter: [3:3] unknown field "documents"`, which names the key and
says nothing about versions: no shipped binary can be taught a new sentence, and
the daily notice in `help` is where a colleague learns a newer release exists.
Added 2026-09-19, DECISIONS.md.

### The diff (what the align skill composes)

There is no diff command and no diff dependency. The binary emits both sides and
the skill composes the comparison. One side is `export`, the document as a
Markdown file on disk with its markers; the other is the note, which the skill
reads itself. Nothing in the binary reads markdown for this, because that would
add coupling and do nothing for an external document, and nothing records a
content baseline: the block holds dated facts, which are the pairing, the
publish record, the export date and gdoc's own proposals.

`gdoc-align` is that judgement, built 2026-09-19, and it runs both ways. Into
the hub it merges with agreement, paragraph by paragraph, and never deletes:
content that exists only in the note is unpublished work rather than drift. Into
the document it proposes each change as a native suggestion, and it asks once
before the first one, because a merge can send twenty where a review sends one.
It reads the merged whole once more and names every place where the logic broke,
which is a conflict as much as a paragraph that differs. A stale copy from a run
that died is never merged from: it says when the copy was made, exports again,
and deletes the copy only when the fresh body matches it.

`gdoc-export` is the other half, for a document with no note at all, where the
person supplies the context a paired note would have given.

## The generator and `house.yaml`

`house.yaml` is the house style: page geometry, all ten named styles, the cover,
header and footer, the three tables cell by cell, the contents field, heading
numbering, and the logo. The generator reads it plus markdown and emits a docx.

**The logo is named, not carried.** `logo:` holds the sizes and the anchor and
exactly one of `drive`, the id of one PNG in the firm's Shared Drive, and
`base64`, the bytes. The embedded file names the firm's file by id, so the
tree, the releases and the binary carry no logo, and a boundary test holding
the PNG's hash fails on any tracked file or docx member that does. `build` and
`publish` fetch the bytes under `AllowAsset`, check them as a PNG of the stated
size, and keep them in `assets/<id>` beside the token with a stamp: one
request a day, the cache on any other day and whenever Drive cannot be
reached, with the envelope's `logo` block saying `drive`, `cache` or `file`
and why. No cache and no access refuses the run, and so does a file in Drive
that is not the PNG the house states, because an unbranded document in the
firm's name is the wrong answer and a wrong file in Drive is somebody's to
fix now. `base64` stays for a test house and for a firm that embeds, and
`--house <file>` is the only override. Decided 2026-10-17, DECISIONS.md.

**`---` on a line of its own in a note's body is a page break**, not a
horizontal rule. Every thematic break goldmark parses is that one break,
`---`, `***` or `___`, and the source spelling is not read. A break before
nothing is nothing: one that ends the body, or that only footnote definitions
follow, is left out with no warning. A break before a break is one break: one
standing directly before the first body heading while
`first_heading_page_break` is on is left out, and two in a row are one.
Nothing else is collapsed. `Counts` carries `PageBreaks`, the breaks the
document got rather than the lines the note held, and both envelopes print it
as `page_breaks`. The horizontal rule goes with its three constants and the
`rule_grey` palette key, because no other route into a document draws a line;
a document that holds a rule of its own still reads as `[rule]` and exports as
that marker. `read` prints `[page break]` where a published document breaks
the page, which it printed nothing for from M13 to v2.12.0, and `export`
writes `---` back for a break standing on its own: the two halves of one line.
Decided 2026-10-16, DECISIONS.md, with the measurement that changed the read
half in MEASURED.md.

**The front matter's page breaks are blocks in that file**, `- block:
page_break`, one after the cover and one after the classification table, beside
`cover`, `label`, `table`, `blank`, `legend` and `toc`. One layout, two writers:
the docx renderer emits the block as a paragraph carrying `pageBreakBefore`, the
prelude emits it as `insertPageBreak`, and neither holds a page break of its own
any more. The drift gate reports nothing new for it, because no item in the
measured list reads a page break or counts a front-matter paragraph. Added
2026-09-16, DECISIONS.md. The body's page breaks are the author's own,
written as `---` in the note, and they go through the same paragraph shape
the docx renderer writes for this block. Nothing reads the master `.docx` at runtime: it stays in the repo as
provenance and as the drift test's fixture. **The comparison is the acceptance
gate, and it runs both ways.** The list of
171 measured values is written once, with one reader per item: offline that
reader is given the docx XML, live it is given a Docs API answer, so a row that
fails in one gate is findable in the other. Offline runs in `make test` on every
commit; live is opt-in, uploads both documents with conversion, and is the
measurement that means something, because Google's import is part of the result.
Either way a difference not named in the list of known ones, with its reason,
fails the gate. No item reads a PDF. House-style tests state house values as
literals, never by reading the constant they test.

## The marks, and footnotes

Both are the author's own markdown, carried into the document and written back
by `export`. Decided 2026-10-14, DECISIONS.md, one entry each.

**A highlight is Obsidian's syntax and no other.** `==words==` is yellow, and a
colour emoji at the start of the highlight picks the rest: 🔴 red, 🟠 orange,
🟡 yellow, 🟢 green, 🔵 blue, 🟣 purple. One grey, `pencil`, is a comment,
HTML's `<!-- words -->` or Obsidian's `%% words %%`, and it is text colour
rather than a highlight. Inside a sentence it is a grey run; on lines of its
own it is grey paragraphs of its own, one per blank-line-separated stretch.
The words inside it are literal and not markdown, so a highlight inside a
pencil is words, and an empty comment is nothing and is left out with no
warning. A comment with the author's own words after it on one line is that
pencil and that prose, and that line wraps the way any other paragraph's line
wraps; two comments on one line are two pencils.
`export` writes the HTML spelling and never Obsidian's, and names the
document's own `%%` as a loss where the next build reads it as syntax, which is
everywhere but inside a pencil's own words. gdoc's own four comments,
`<!-- tab ` and the three a floating object is written with, `<!-- image: `,
`<!-- drawing: ` and `<!-- object: `, are markers and not pencil, and are
refused by line on every route in; a grey run whose comment would carry one of
them comes home as its words with the loss named. The old span
`[words]{.pencil}` is no longer read and is refused by line naming the
comment. Changed 2026-10-16,
DECISIONS.md, two entries. The seven are stated once, in `house.yaml`
under `marks:`, keyed by Obsidian's own colour names, each with its syntax, its
kind and its Doc colour, and one of the highlights has to claim the bare `==`. `gdoc help marks` prints that table, so a skill reads
the binary rather than carrying a copy.

In the docx each mark is a named character style and the run carries the exact
hex, as shading for a highlight and as text colour for pencil. Docs has no
character styles, so after import a mark is direct formatting and that is what
`export` reads back. A colour outside the palette stays plain words and is
counted by hex. A mark whose words cannot carry the syntax it is written with,
the highlight's own delimiter inside a highlight or the comment's closer `-->`
inside a pencil, comes home as the words alone with the loss named, because
the file has to build again, and so does any formatting other than the grey
on a pencilled run. A mark inside a footnote is the one that does not come home:
the read keeps a footnote's text alone, so the words arrive plain and `export`
names the loss. `read` prints the words and not the mark. `restyle` is
unchanged, and the cover's yellow placeholders are the prelude's rather than
marks.

Yellow, green and purple are authoring debts: `build` and `publish` carry a
count of each on the envelope, zero included, and the publish skill asks before
a document carrying any goes to Drive. The binary counts and never judges.

**A footnote is `[^label]` with `[^label]: the text` on its own line**, which is
what goldmark already parses and what `read` already prints. Notes are numbered
in first-reference order. One label referenced twice becomes two notes carrying
the same text, because Word and Docs hold one note per mark. The note text is
the tenth house style, `footnote_text`, and the reference mark is a superscript
character style. `word/footnotes.xml` is always written, with the separator and
the continuation separator, which makes the generator's parts thirteen.

**What is refused by line, in both halves.** A reference with no definition, a
definition nothing references, a label defined twice, a footnote mark in a
heading, a mark in a heading, a nested mark, the old pencil span `]{.pencil}`,
and a footnote mark or a mark inside a picture's alt text. The heading refusals are read off the tree as
well as off the line, because goldmark builds a heading from a setext line, from
a block quote and from a list item, and no line scan sees one; the alt-text
refusals are read off the tree alone, because an alt text sits on a line of
prose. Which labels are defined is the parse's answer too: goldmark reads a
definition wherever a block may start. Inside a
footnote, a list, a table, a picture, a code block or a heading is warned about
by line and left out. No footnote in a table cell; a mark in one is carried. A
code block is measured from the content indent of the block it sits in, so a
code sample in a nested list item or inside a footnote defines nothing and
refuses nothing, and a block of HTML that is not a comment is left out of
both scans whole.

## Install and update

**The version is `x.y.z`, and the number is the channel.** Nail cuts `x.y.0` by
hand with `make tag`; the nightly cuts `x.y.(z+1)` when main has moved since the
last tag. `z == 0` is stable, `z > 0` is nightly, and nothing else records which
is which. The linker sets it from the tag, so a binary built from a checkout is
`dev` and prints no version at all: `version` rides in the envelope only when
there is a release behind it, and `gdoc help` and `gdoc auth status` read the
same value. The plugin carries the last stable number: `make tag` writes it
into `.claude-plugin/plugin.json`, the nightly leaves that file alone, and an
`x.y.0` release whose tag and plugin disagree is refused by the workflow. A
nightly is a binary release and never a plugin release. Builds are trimmed and
stripped, so no home path ships and one tag is the same bytes on every
machine.

**A release is one GitHub Release of this repository**, which is public since
2026-09-16. A tag builds it: the four checks every push runs, then `make dist`
with the client secret from a repository secret, then one zip per line of
`release/platforms`, named `gdoc-<tag>-<platform>.zip`, and one
`SHA256SUMS-<tag>` over all of them. A zip carries seven things on darwin: the
binary, the user `README.md`, `install.sh`, `skills/`, `example/`, the Claude
Desktop extension template under `mcpb/`, and the presence app at
`presence/GdocPresence.app`, which a second job on a macOS image builds and the
publishing job waits for. Every other platform carries the first six.
`release/platforms` is
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

**One copy of the binary serves all three front doors.** Claude Desktop starts a
command named in an extension rather than typing `gdoc`, so `install.sh
--desktop` writes a thin `gdoc.mcpb` beside the installed binary, whose command
is that binary's absolute path, and opens it so Claude Desktop shows its Install
button. The bundle carries no binary of its own: the terminal, Claude Code's
skills and Claude Desktop all run the same file, and `gdoc update` updates all
three. The extension travels as a template in the zip, `mcpb/manifest.json`,
with two placeholders for the path and the version, because the settings live in
Claude Desktop's own form rather than in a file a person edits. macOS only, and
the one program gdoc runs is `/usr/bin/open`: on the extension file it has just
written, and, under the `presence` setting, on the app bundle as the entry above
says. On nothing else. Added 2026-10-03, DECISIONS.md, the app 2026-10-17, the
opener 2026-10-18.

**The skills travel as a Claude Code plugin.** This repository carries
`.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json`, so it is
both the plugin and the marketplace, and `skills/` stays where it is. A
colleague working in the hub gets all three from the hub's own committed
`.claude/settings.json`, which names this marketplace in
`extraKnownMarketplaces` with `autoUpdate: true` and turns `altery@gdoc` on in
`enabledPlugins`: one trust prompt and the plugin is there. Outside the hub the
two `/plugin` lines are the route, `/plugin marketplace add nhusnullin/gdoc`
and `/plugin install altery@gdoc`, global or per-project in Claude Code's own
terms. Claude Code delivers a plugin when its `version` string changes, which
is why that string moves on `make tag` and on nothing else. Two fallbacks exist
by policy, because a managed Claude Code can refuse a marketplace and can
refuse personal and project skills with it: `install.sh --skills global|local`
copies `skills/` out of the zip, marked so a re-run replaces only what it
wrote, and an administrator names the same two keys in
`managed-settings.json`. A checkout still links its skills and never copies
them, and nothing in the binary copies a skill folder.

**`gdoc update` installs when a person types it and never otherwise.** It reads
one page of releases, asking for a hundred, because GitHub answers thirty by
default and the nightly would push the last hand-cut stable release off a page
that size in about a month. What it takes: the same `x` with a higher `y` by
default, a higher `x` only with `--major`, a higher `z` only with `--nightly`,
and never a version below the one installed. `--check` reports
what a run would take and writes nothing; `--rollback` puts the previous binary
back. The zip is verified against the release's own checksum before a byte is
replaced, the binary that was there is kept beside the new one, and the object
names one of `up_to_date`, `updated`, `major_available`, `checked`,
`unreachable` and `rolled_back`, with `run` naming the command that would go
further where there is one. Unreachable is `ok: true` with a warning, because a
network that is not there is not a failed update. The command holds no state
between runs.

**`gdoc update --desktop` refreshes the extension, and it is the one place gdoc
runs a program.** It takes the template out of the release zip it has just
verified, so the template is always the new version's, writes `gdoc.mcpb` with
the path and the version filled, and opens that file. Past the point where the
file is written everything about it is a warning rather than a refusal, because
the binary has already been replaced; the one refusal is a missing template,
which is read first. A run that installed nothing, a declined major included,
writes no extension and says so, because the template would name a version this
machine does not run. `--desktop` is refused beside `--rollback`. Plain `gdoc
update` writes no extension and runs nothing: where a `gdoc.mcpb` sits beside
the binary and the new release's template differs from it, it ends with one line
saying to run `gdoc update --desktop`. On a checkout build, where `update`
already refuses, the route is `./install.sh --desktop`. Added 2026-10-03,
DECISIONS.md.

**`gdoc help` asks what is published, once a day.** It is the one thing gdoc
does that nobody typed, and the whole of it is one read. A stamp,
`update-check.json` beside the token file, holds when gdoc last asked and what
it heard. `help` refreshes it when it is missing, unreadable or older than 24
hours, on the grant above, under a two-second ceiling rather than the listing's
five, with no credential and no document id. A failed check is stamped too, so
a machine that cannot reach GitHub pays the ceiling once a day rather than once
a run. A binary built from a checkout carries no version, so it has nothing to
compare and never asks. Every other command is as fast and as offline from
GitHub as it ever was. The object carries `update` with `installed`,
`latest_stable`, `latest_nightly`, `checked_at` and `error`, and judges none of
them; one line on stderr, beside the help prose, names a newer stable of the
same major and `gdoc update`, or `gdoc update --major` across a major, and is
absent when there is nothing to say. Nothing is downloaded and nothing is
replaced: the install still waits for a person to type it. Added 2026-09-18,
DECISIONS.md.

**`gdoc mcp` asks the same question once per process**, and it is the second and
last command that asks unasked. The stamp, the 24 hours, the two-second ceiling
and the grant are `help`'s, so a chat and a terminal cost one request a day
between them, and the first tool answer of a server process carries one line
naming the release. The words differ: a chat is told to run `gdoc update` in a
terminal, and then to quit Claude Desktop and open it again, because toggling
the extension restarts the chat process alone and leaves agent mode on the old
binary. A checkout build never asks. Added 2026-10-03, DECISIONS.md.

## The skills, and how a comment reaches one

Five named workflows, all judgement rather than commands. Each is a skill, one
copy in this repo: symlinked into a checkout, and installed on a colleague's
machine as the plugin above. They are the review session, the publish run, the
restyle run, the export run and the alignment run, the last two added
2026-09-19, DECISIONS.md. One of the five runs on a marked comment, and every
one of them starts from a colleague's own sentence:

- **The review session.** Reads the threads, answers `ai?` from the hub, carries
  out `ai!` against the hub, proposes document changes as suggestions. If it
  could not read half the review it says so and does not report clean. It can run
  **live**, polling on the `--since` cursor until Nail stops it. A colleague's
  `ai!` acts too, by decision: the marker is the trigger, identity is not a gate,
  and the guard caps a handed-in document at suggest and reply.
- **The alignment run**, `gdoc-align`, for a note already paired with a
  document. Composes the comparison over `export` and the note, judges what
  matters with the person, and goes both ways: edits into the hub with
  agreement, suggestions into the document after one question. It checks the
  merged whole for logic that broke, and it deletes a copy only after a fresh
  export says the copy holds nothing new.
- **The export run**, `gdoc-export`, for a document with no note. Runs `gdoc
  export`, names the file from the document's title, adds `title`, and resolves
  the markers with the person or leaves them for a later day.
- **The publish run**, `gdoc-publish`. Nail names a note, and the skill builds
  it, publishes it into the folder, and reads back what came out. On a note that
  already names a document it stops and asks, a new document or a merge through
  `gdoc-align`.
- **The restyle run**, `gdoc-restyle`. Nail gives a link, and the skill surveys
  the document before it proposes anything to it.

**A skill names the binary version it needs**, as `needs:` under `metadata` in
its own front matter. Its setup runs `gdoc help` first and reads `version` from the object,
and an older binary stops the skill with one sentence naming `gdoc update`. No
`version` at all is a build made from source rather than a release, which is not
an error: the skill says so once and carries on. Skill and binary may drift by a minor
version without harm, because a skill holds no flag list, which is the next
rule. Added 2026-09-16, DECISIONS.md.

**A skill keeps to the Agent Skills specification.** Its front matter sets only
the six top-level keys agentskills.io allows, carries a `compatibility` line,
and parses as YAML. Its SKILL.md stays under 500 lines and 20,000 characters.
What only some runs need sits in a file beside it, which SKILL.md names at the
step that needs it, and whose calls are read against the table like SKILL.md's:
`TestEverySkillFrontMatterKeepsToTheAgentSkillsSpec`,
`TestEverySkillBodyFitsTheSpecBudget`, `TestEveryFileBesideASkillIsNamedByIt`
and `TestEveryFileBesideASkillNamesOnlyWhatTheBinaryHas`. Added 2026-09-27,
DECISIONS.md.

**No skill holds a flag list.** Before the first call of a command in a session
a skill runs `gdoc help <command>` and reads the words and flags from the binary
it is about to run. `gdoc-publish`, `gdoc-restyle`, `gdoc-export` and `gdoc-align` do this, and
`gdoc-review` is the one that still names its flags, until it is converted. One
direction holds for all five meanwhile:
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

## Chat: `gdoc mcp`, the sixteenth command

Serves principle 1. A colleague who has the binary and Claude Desktop reviews a
document by asking for it in their own words in a Claude Desktop chat, typed or
dictated, with no terminal open. Live voice mode does not reach a local
extension (MEASURED-archive.md, "gdoc mcp in Claude Desktop, the first run"). `gdoc mcp` is this binary as a local MCP server: JSON-RPC 2.0,
one message per line, over stdin and stdout, written by hand in `internal/mcp`
so there is no fourth module. `main()` routes `mcp` before `run()`, because
`run()` prints one envelope after every command and a table entry's `run` sees
neither stdin nor stdout. The table keeps the row, so `help` and `completion`
name it. The command takes no words and no flags. Added 2026-10-03, and its one
flag removed the same day, DECISIONS.md.

**Eight tools, and nothing else is one.** Six are commands of the table, in the
order a review runs: `read`, `comments`, `suggestions`, `reply`, `annotate` and
`propose`. Two are the server's own, `login` and `guide`. A call runs the same
`safeDispatch` the terminal runs, in the same process, and answers with the same
envelope, so there is no second route into a document and nothing is started as
a subprocess. Each schema is written by hand beside its mapping to words and
flags, and a test pins every property to a word or a flag, and every word or
flag to a property or a named exclusion. `update`, `restyle`, `publish`,
`export`, `build`, `probe`, `withdraw`, `help` and `completion` are not tools: a
spoken sentence never creates a file, never changes a document's look and never
touches the installation. Nothing in chat reaches a hub, so no tool takes `--md`
and a write tool names one item per call.

**One call at a time, and 200 seconds each.** A reader parses the lines and
answers `initialize`, `ping` and `tools/list` where they arrive; one worker runs
the tool calls, in the order their lines came. The deadline is 200 seconds from
the moment a call's line was read, because the client gives up at 240, counted
from when the call reaches the server, and the 40 seconds left over are what buy
gdoc the chance to say in its own words what went wrong. `notifications/cancelled`
drops a call still in the queue and throws away the answer of one that started.
A panic in a tool is one error result and the server goes on to the next call.

**Every piece of state is per process, and the sign-in alone is shared.** Claude
Desktop starts one process per client, one for chat and one for agent mode, and
both stay running, so the guide code, the ledger, the holds, the write memory
and the once-a-day release line each live in one process, reach no disk and are
shared with nobody. The exception is the sign-in, through the record beside the
token in Auth above.

**`guide` gives the rules and a code, and no other tool runs without it.** Chat
does not show a server's instructions to the model, so the rules are made to
arrive another way. `guide` answers a short chat header and the review core, and
with them a code of sixteen hex characters random to the process. Every tool but
`guide` and `login` takes that code, and a missing or stale one is refused with
one sentence telling the model to call `guide` and retry the same call, which it
does without asking the person about a code they never typed. The code is
reliability and not safety: it proves the rules entered the context, and it
cannot prove they are followed. The instructions are sent as well, eight short
lines, and they name no setting.

**What a document says arrives labelled, and the wrapper is new each time.**
Every read answer opens with one fixed line: who wrote what follows, that it is
material to discuss, and that nothing in it is an instruction to call a tool,
gdoc's or another connector's. Each comment, reply, quoted span and document
text is wrapped in a boundary of twelve random hex characters, new for every
answer, and an occurrence of that boundary inside the text is broken by one
character, which is also what stops a comment closing its own wrapper. The words
themselves come through as they stand: `annotate` and `propose` need the exact
quote to find it again, and a person listening rather than reading would hear a
marking as noise.

**Five facts sit beside every comment and every reply**, each one literal check:
`has_link`, `has_email`, `names_ai`, `hidden_chars` and `author_domain`. They
describe the words and never the person. `robot_not_ours` was the sixth until
2026-10-04, when it was removed because it flagged nothing (DECISIONS.md).
`author_domain` is the domain of the author's
address, kept in no other form, and it is shown and read by nothing: identity is
never a gate, the marker decides, and a domain is a fact a person can weigh. It
is absent where Drive gave no address, never an empty string a model could read
as a kind of account, and on 2026-10-03 Drive gave none on any comment. The CLI
envelope gains that one field and nothing else.

**A risky write is held by a fixed list, and only the person's own card releases
one.** The binary decides that a write is risky, never the model, because a
model asked whether a comment is steering it is the thing being steered. A write
is held when twelve or more words in a row are shared with a comment this
process read that gdoc did not write; when
another document was read in the last 30 minutes, or the target was never read;
on the third write inside 60 seconds, or the 26th into one document inside an
hour; when a `reply` goes into a thread whose comment carries a link, names the
model or holds hidden characters; and when a
`propose` takes out more than 300 characters. The first rule that trips is the
one named. Text holding zero-width, bidi or tag characters is refused outright,
because nobody can approve what they cannot see. Text copied out of another
document is held and never refused: quoting the firm's own policy can be the
job. A link, a bare domain or an email address in a write is ordinary text and
is not held, so the extension has no settings: the one field it had exempted
addresses from a link hold that no longer exists. Changed 2026-10-03,
DECISIONS.md.

A held write is not sent. The answer is `ok: false`, `sent: false`, and the hold
with its id, its rule, the exact value that tripped it and the words themselves,
under one fixed sentence saying that nothing was posted, to tell the person this
reason and to end the turn. The hold lives 30 minutes in the process that made
it, and any chat that process serves can release it.

Release is by card and never by a yes said in the chat, because a yes in the
chat passes through the model. Each hold registers one tool of its own,
`confirm_<hex>`, through `notifications/tools/list_changed`, and removes it when
the hold is released or runs out: always-allow is stored per tool name and
cannot be granted in advance to a name that never existed. The confirm call
carries the hold id, the document's title, the reason and the text, and a byte
of difference in the title, the reason or the text is refused with the hold
kept, so the person cannot be shown a paraphrase of what they approved. A
release with less than five seconds of quiet behind it is refused the same way.
What then goes out is the call the hold kept, never anything the release sends.
A write this session already made comes back from memory for ten minutes under
the same tool and the same arguments, so a retry is not a second reply.

**There is no live mode in chat**, because a server cannot wake the client.
`comments` answers a cursor and the model reads again from it when the person
asks, and `--wait` is no part of the tool. Chat and a live Claude Code session
run side by side on one document and one token: chat's replies open with the
mark, so the live session never takes them as work, chat leaves marked comments
to that session, and chat reads a thread again just before it posts and stops
when a marked reply has appeared.

**The review rules are one core, read through either front door.**
`skills/gdoc-review/review.md` holds the markers, when to answer, when to wait
for approval, what must never leak, the all-comments mode, the two guards above
and the flow for a comment on words the person names. It carries no call line,
so it reads the same in a terminal and in a chat, and `SKILL.md` keeps what
belongs to Claude Code. `go:embed` cannot reach outside its own package
directory, so the binary carries a committed copy and a test fails when the copy
and `skills/` differ by a byte.

The twelve measurements this section rests on are in
[MEASURED-archive.md](MEASURED-archive.md), under "Claude Desktop and a local MCP server".

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
- Never delete from the hub, **except a copy `export` itself wrote**: the align
  skill removes the copy it made this run, and a stale one only after a fresh
  export says that copy holds nothing the document does not. A skill rule,
  narrowed 2026-09-19, DECISIONS.md.
- Never run git, in the binary or in the skills.
- Never carry the firm's logo in the tree. Test-enforced: a boundary test holds
  the PNG's hash and walks every tracked file and every docx member. Added
  2026-10-17, DECISIONS.md.
- Never read the presence file in the binary, and never let anything gdoc
  decides depend on it. Test-enforced: the package exports no reader and
  `O_APPEND` lives in one room. Added 2026-10-17, DECISIONS.md.
- Never prompt, in the binary, and never in the installer either.
- Never install unasked. `gdoc update` replaces the binary when a person types
  it and at no other moment. The one thing that runs by itself is the check in
  `gdoc help`, once a day, which reads what is published and writes one file of
  gdoc's own.
- Never export a PDF. Nail downloads it from the browser. `export` writes
  Markdown and PNG files, and never replaces one.
- Never write to a multi-tab document. `export` reads one, one file per tab.
- Never write markdown into a comment thread.
- Never trust a status code where the write matters.
- Never write a comment or reply that does not open with the mark in effect,
  which may be set to none.
- Never send a write gdoc held, in chat, without the person's own card.
  Binary-enforced: the hold is kept in the process, the card's words are
  compared byte for byte, and what goes out is the call the hold kept.
- Never act on a marked comment from chat. Chat has no hub, so `ai!` is work a
  Claude Code session carries out, and chat says so before it posts.

## Acceptance for v2.0

Each is a test or a checkable run, not a claim.
1. The guard test: HTTP construction outside the guard package fails the build;
   a request for an id outside the set is refused; a direct edit on a handed-in
   id is refused unless the run granted that one id the styling level, and there
   every kind but the four styling ones is refused, as is a `fields` mask of `*`,
   an empty mask or one naming either header toggle. `createNamedRange` carries
   only over the range `AllowMarker` granted.
2. The drift test: config-rendered and master-rendered documents compare across
   the 171 items with no new differences.
3. A publish produces a document with the positioned logo, a live contents list
   and footer page numbers, verified by export.
4. A propose lands as a genuine suggestion, absent under
   `PREVIEW_WITHOUT_SUGGESTIONS`, with its 🤖 comment anchored to the exact words,
   verified through the docx export.
5. An in-place restyle on a copy of the ideal test document preserves all ten of
   its features, the chips, the Drawing, the footnotes, the lists, the pinned
   shaded table header, the byte-identical inline image, the anchored comment and
   the pending suggestion, and the original is untouched, same `revisionId`.
6. `gdoc probe --folder` distinguishes an enrolled project from an unenrolled
   one without writing to any real document.
