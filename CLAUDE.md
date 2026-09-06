# Notes for AI assistants

Read [PRINCIPLES.md](PRINCIPLES.md) before proposing any design. It is four
constraints and the decisions that currently implement them, and it is short.

Read [README.md](README.md) for what the tool does and how to run it.

This file holds the specifics: what lives where, what each rule means in code,
and what never to do. The reasons live in PRINCIPLES.md, and they live there
only.

## What lives where

| Path | Holds |
|---|---|
| `gdoc/` | the package. Imports are `from gdoc.x import y` |
| `gdoc/guard.py` | the reachable set. Which files a client may touch, under either credential |
| `gdoc/marker.py` | the `[gdoc]` label on gdoc's own replies |
| `gdoc/restyle.py` | the house-styled copy of a document nothing here is paired to |
| `gdoc/comments.py` | carrying comment threads from one document onto another |
| `gdoc/edits.py` | the diff between the baseline and a fresh export |
| `gdoc/suggestions.py` | pending suggestions, read through the Docs API |
| `tests/` | pytest suite. Every module has a matching test file |
| `skills/` | `gdoc-review` and `gdoc-apply`. Symlinked into `~/.claude/skills/`, so edits are live |
| `docs/superpowers/` | the implementation plans and the design specs |
| `go/` | v2, the Go rewrite. Its own module, standard library only. See the section below |
| `spike/render/` | the render spike that settled the language decision. Throwaway, and it says so. Renders and publishes; reads nothing back. Kept as the measured reference M5 and M6 build against, not as code to extend |
| `.gdoc/<slug>/` | `pending.md`, `baseline.md`, generated `out/*.docx`. **Not in this repo:** it sits beside the source markdown being reviewed |

Secrets and the venv live in `~/.config/gdoc-agent/`, never in this repo.

## v2 lives at `go/`, and v1 is untouched

`gdoc/` is v1, the Python package every other section here describes. `go/` is
v2, one static binary on the Go standard library. The two trees do not import
each other, and nothing in the Go work has changed a line under `gdoc/`.

| Path | Holds |
|---|---|
| `go/cmd/gdoc/` | `main.go` and `read.go`. Arguments in, one JSON object out, exit |
| `go/internal/emit/` | the output envelope every command prints through |
| `go/internal/guard/` | the network policy, and the only place a client is built |
| `go/internal/auth/` | the token file, its refresh, and the login flow |
| `go/internal/auth/loopback/` | the one-shot localhost listener the browser redirect lands on |
| `go/internal/config/` | where the per-user files live, per platform |
| `go/internal/gapi/` | the authenticated session. The one room that builds a request |
| `go/internal/docs/` | the Docs read: tabs, the document tree, suggestion ids, comment ranges |
| `go/internal/view/` | the document as the text `read` prints, and as the tree `--structure` prints |
| `go/internal/comments/` | Drive's threads joined to the Docs ranges, and the `--since` cursor |
| `go/internal/suggestions/` | what is pending, and what stopped being pending since the snapshot |
| `go/internal/docx/` | the docx export reader, and the witness match against threads |
| `go/internal/frontmatter/` | the `gdoc:` block in a note's YAML front matter, and nothing else in the file |
| `go/internal/atomicfile/` | the temp-file-and-rename write. The one room that replaces a file's contents |
| `go/internal/live/` | the one opt-in end-to-end test. Tests only, no production code |
| `go/boundary/` | the two allowlist tests that keep the wire in one room |
| `bin/` | what `make build` and `make dist` write. Not in git, so both targets create it |

`docs/v2/SPEC.md` is the agreed design and `docs/v2/PLAN.md` the milestone
order. Milestones 1 and 2 are done: the binary exists, prints the envelope, owns
the network, can log in and report its OAuth state, and reads a document three
ways with `read`, `comments` and `suggestions`.

### The auth commands, and what reaches stdout

`gdoc auth status` reports `auth_mode`, `token_path`, `client_source` and
`token_present`, plus `expired`, `scopes` and `missing_scopes` when a token is
there. Being signed out is an answer, so it comes back as `ok: true` with
`token_present: false` rather than as a failure.

The shape is `auth.StatusReport`, a struct with json tags, not a map. The
command reads its fields in Go, so a field renamed in `auth` cannot silently
drop a warning in `cmd`.

Four things about those fields:

- `auth_mode` is the constant `"oauth"`. v2 has no service account and never
  reads v1's `config.json`, so on a machine set to `auth_mode: service_account`
  the two tools disagree on purpose. The resolver documented further down is
  v1's alone.
- `client_source` is always `"bundled"`. v1 lets `oauth-client.json` in the
  config dir override the client; v2's login does not read that file yet, so
  when it exists status adds `client_file_ignored: true` and a warning rather
  than claiming an override that is not wired up.
- `missing_scopes` names what v2 asks for that the token does not carry. A
  partial grant is reported, never refused: the login worked, and this is the
  one place that can say why the Docs calls will 403 before they do. A v1 token
  is not an example of one: it carries the full Drive scope, which covers the
  Docs scope v2 asks for, so it reports nothing. See `MissingScopes` below.
- A token file that exists and cannot be read is a **failure**, not
  `token_present: false`. It comes back `ok: false` with the path still in
  `data`, and with the warnings it would have carried on the way out. Reporting
  it as signed out is how somebody re-runs `auth login`, overwrites the file,
  and never learns what was wrong with it. Only an absent file means signed out.

A panic anywhere is still one JSON object: `cmd/gdoc` recovers, prints
`ok: false` with what happened, and puts the stack trace on stderr. A Go trace
on stdout with exit 2 would break the contract every caller has.

`GDOC_CONFIG_DIR` moves the config dir. It is how every Go test avoids the real
config, the v2 counterpart of v1's `tests/conftest.py`.

`gdoc auth login` prints the authorization URL to **stderr**, waits for the
browser to come back to the loopback listener, saves the token, then reports
what `auth status` would. Exactly one JSON object reaches stdout, always through
`internal/emit`, and the exit code is 0 if and only if that object says `ok`.
Human words and the URL have one place to go, and it is not stdout.

The binary never prompts and never reads stdin. A command missing something
fails and says what is missing. It does not ask.

`gdoc --help` is therefore `ok: false` and exit 1, and that is deliberate rather
than an oversight. There is no help command: `dispatch` matches the five
commands and nothing else, so `--help` comes back as an unknown command with the
one-line `usage` string in the error. A caller reads the same JSON
object it reads for every other run, and the exit code still means what it means
everywhere else. Human-readable help would have to reach stdout beside the
object, or exit 0 on a run that did no work, and both break the contract.

### The token file is v1's, so a v1 login is already a v2 login

`internal/auth` reads `oauth-token.json` from the config dir in v1's google-auth
"authorized user" shape, field names included. Somebody logged in through v1
needs no migration and no second browser trip. The bundled client id and secret
are v1's two constants copied verbatim, and the rule about them has not changed:
read "The OAuth client is shipped, and stays Internal" below.

**The other direction is not symmetrical, and this is a scope widening.** v1's
`LOGIN_SCOPES` is Drive plus `documents.readonly`. v2's login asks for Drive
plus `documents`, the read/write scope, because v2 writes suggestions through
the Docs API. v1 refuses a token whose stored scopes lack one it requested
(`gdoc/oauth.py`), so after a `bin/gdoc auth login` v1's `gdoc edits` asks for a
fresh v1 login. Nothing else in either tool changes. Do not "fix" the comment in
`login.go` by calling the two sets equal: they are not, and the widening has to
stay written down.

**`MissingScopes` knows that the full Drive scope covers the Docs calls.** The
Docs API accepts `auth/drive` on `documents.get` and `documents.batchUpdate`, so
a v1 token, which is Drive plus `documents.readonly`, is missing nothing v2
needs. Comparing the requested list literally warned on Nail's own working
token, and a warning on the working case is one people learn to ignore.
`coveredBy` in `login.go` is where that lives. It is a report, never a refusal,
and `drive.file` is deliberately not in it: that scope reaches only files the
app itself created.

A v2 `Save` carries `universe_domain`, `account` and `rapt_token` through
untouched. They are google-auth's fields, `Credentials.to_json` writes all three
when they are set, v2 uses none of them, and dropping one would quietly rewrite
a file both tools share. `rapt_token` is the reauth proof token, so losing it
makes v1 ask for reauthentication again.

`Load` refuses a file that parses but cannot be refreshed. The required set is
google-auth's, not one v2 invented: `from_authorized_user_info` raises without
`refresh_token`, `client_id` and `client_secret`, and v1 reads this same file
through it. A `{}` that read as a token would have `auth status` report a token
present and the first Docs call fail with something else. `token_uri` is not in
that set, because google-auth overrides it with its own constant whatever the
file says; `Load` fills the same value in rather than posting a refresh to an
empty URL.

### The guard owns the wire, and it exists before any client

`guard.NewClient` is the only place an `*http.Client` is made, and it is made
from a `*Policy`. So the first request in the program's history has already been
judged. v1 fitted a guard around a client that already existed, which is why v1
needs a test proving `build()` is called in one module only.

Write levels live in the policy, never at the call site. `LevelSuggest` is what
a handed-in id gets: read, comment, suggest, and never a direct edit.
`LevelFull` is what a create returned, and `Learn` is the only door to it. M7
adds a per-run in-place grant back beside its caller; read "`GrantInPlace` is
gone until M7" below before looking for one now. A call site cannot widen its own reach by phrasing a
request differently, because the policy reads the method, the URL and the body:
a `batchUpdate` on a handed-in document is refused inside the process unless the
body says `SUGGEST`.

A create is refused unless it names exactly the one folder the run was given,
and the transport reads the create's response for the new id and teaches the
policy. Those are still principle 3's two doors, ported. Naming a folder to
create in does not put that folder in the reachable set: it is a create target,
not a third door.

**Read this before trusting the level-1 write bar.** What keeps a handed-in
document read-and-suggest only is `writeControl.writeMode == "SUGGEST"` in the
request body, which is a field the client itself supplies.
`docs/v2/BLOCKED-BY-API.md` records the measurement: `writeMode` is absent from
the public Docs discovery document, and one morning this exact call returned 200
and silently made a direct edit. So the server is not known to honour it, and
what the guard holds today is a statement of intent rather than a guarantee. The
capability probe the spec relies on, which would ask the server what it will do
before the write goes out, **does not exist yet**. Widening or narrowing what
`isSuggestMode` permits is a decision for Nail, not a refactor.

The guard judges the request it actually sends. It refuses
`X-HTTP-Method-Override` and its two cousins, and a `_method` query parameter,
because Google's REST stack performs the overridden method: a GET the guard
allowed would arrive as a DELETE it never saw. A create it cannot read the
parents of is refused, which today includes a multipart upload: that body opens
with the MIME boundary, so the `/upload` grammar is unreachable until M6 teaches
the transport to read the first MIME part. Failing closed is the right direction
to be wrong in.

Three more shapes belong to that same rule, and each closes a way the request on
the wire differed from the one that was judged. The two allowlists below them,
one over the query and one over the headers, are the same rule again and are
broader than all three.

- **The upload parameter decides what the body is.** With `uploadType=media`
  the body IS the file's content, so `{"parents":["FOLDER1"]}` reads as bytes to
  Drive and as metadata to `checkParent`: the file lands unparented and the
  guard then learns its id at `LevelFull`. `checkUploadShape` permits the shapes
  the parent check can read, `multipart` and `resumable` and an absent
  parameter, and refuses the rest. It reads `upload_protocol` too, which is the
  same choice under Google's newer name, where `raw` is what `media` was. A
  request naming both parameters is refused rather than guessed at: the guard
  would be reading the one Drive may not obey. Nothing may learn an id from a
  create it could not verify.
- **`fields` reaches the permission surface.** A GET is judged on its path, and
  `fields=*` or `fields=permissions(...)` returns exactly what refusing
  `/permissions` was for. `checkFields` refuses `*`, refuses `permissions` by
  any spelling, and refuses an empty mask, because Google reads a mask with
  nothing in it as every field.
- **The base transport carries no proxy.** `http.DefaultTransport` reads
  `HTTPS_PROXY`, so a nil base would send an unjudged `CONNECT` to whatever host
  the environment named, with the credential following it there. `baseTransport`
  sets `Proxy: nil` for that reason. If a proxy is ever wanted it has to be
  judged, not inherited from the environment.

A create whose response carries no readable id is recorded on the policy and
readable through `Policy.Warnings()`. Silence there turns into "file was not
given to this command" on the next request, which names the wrong problem.

`Policy` is mutex-guarded. `NewClient` hands out an `*http.Client`, which Go
documents as safe for concurrent use, and `Learn` writes the set from inside
`RoundTrip` while `Judge` reads it. `make test` runs `-race`; keep it there.

### Two allowlists over one request, the query and the headers

This is the guard's broadest rule, and it is easy to read past because neither
half names a single attack.

Google gives one capability several spellings. `fields` is also `$fields` and
also the header `X-Goog-FieldMask`. `uploadType` has the sibling
`upload_protocol`. `key` is also `$key` and the header `X-Goog-Api-Key`. A list
of blocked spellings needs a patch each time somebody finds another one, and
every miss is a live hole until then. So both halves are allowlists. They are
wrong in the direction of a refusal the next milestone widens on purpose.

- **The query.** `params.go` holds one allowlist per call shape, not one across
  all of them: `driveGetParams`, `driveExportParams`, `driveCommentListParams`,
  `driveReplyListParams`, `driveCommentGetParams`, `driveWriteParams`,
  `driveCreateParams`, `docsReadParams`, and `noParams` for a call that carries
  no query at all. One list for everything was wrong in both directions: it put
  paging on a metadata read and an export format on a comment listing, neither
  of which is a call Drive has, and it put `alt` on the bare `files.get`, where
  `alt=media` stops being a metadata read and hands back the file's bytes.
  `checkQuery` parses the raw query itself rather than through `u.Query()`,
  which drops a pair it cannot read and returns the rest, and it refuses a
  parameter given twice.
- **The headers.** `allowedHeaders` in `transport.go` names seven, in lower
  case, and every other header is refused. They are `authorization`,
  `content-type`, `content-length`, `accept`, `accept-encoding`, `user-agent`
  and `referer`. A Drive read whose query carries no `fields` can still ask for
  `permissions(...)` in `X-Goog-FieldMask`, so a guard that judges only the
  query judges half the request. `referer` is on the list because
  `http.Client.Do` builds a redirect hop itself and sets it, and leaving it out
  refused every redirect the policy allows. `cookie` is deliberately absent:
  `NewClient` sets no jar, so a Cookie is a second credential nobody decided
  about. A milestone needing another header, a resumable upload's
  `X-Upload-Content-Type` for instance, adds it here on purpose.

Both loops walk the raw header map and fold case themselves. `Header.Get`
canonicalises the key it looks up, so it finds nothing stored under
`X-HTTP-METHOD-OVERRIDE`, and two keys that fold to the same name are two lines
on the wire. Counting one key at a time read each of them as the only one, which
is how `Authorization` beside `authorization` put two credentials on a judged
request and left the server to pick.

`checkAuthorization` is the most the guard can honestly say about the
credential, and the limit is worth holding. It checks one value, the `Bearer `
scheme, and a token after it. That refuses a second credential, another scheme
carrying another principal, and an empty grant. It cannot check whose token it
is: the guard is built from a policy and a base transport and never sees the
token, and pinning a literal would refuse the request after every refresh. So a
caller that swaps in another person's bearer token runs the judged operation as
that person. What the guard still bounds is which files are reachable and what
may be done to them, which is principle 3's actual claim. Nothing here is a
claim about identity. The refusal never prints the header value, because a
refusal goes into the JSON the caller reports.

### The guard judges plain paths only

A path whose escaping differs from its plain form (`%2F`, `%2E` and the like),
or a `.` or `..` segment, is refused before the host is looked at. A harmless
`%20` passes: Go only sets `RawPath` when the escaped form differs from the
default encoding of `Path`. A URL carrying credentials is refused too, because
those become an Authorization header gdoc did not build. The reason is not tidiness. `u.Path` is decoded, `u.EscapedPath()` is
what goes out, and Go cleans no dot segments out of a URL. Without the rule the
guard reads one URL and the transport sends another:
`/drive/v3/files/DOC1/../../../about` passes as a read of DOC1 and arrives as
`drive.about.get`, an endpoint the guard refuses when it is asked plainly.
`DOC1%2F..%2Fabout` is the same walk through a second door. Real Docs and Drive
ids are `[A-Za-z0-9_-]`, so refusing both shapes costs nothing.

### The boundary test holds two allowlists, and the difference is the point

Naming `net/http` and dialing with it are not the same thing, so
`go/boundary/boundary_test.go` checks both.

- The **import allowlist** says who may name the type. Four rooms:
  `internal/guard`, `internal/auth`, `internal/auth/loopback` and
  `internal/gapi`. `internal/auth` is on it because `Refresh` and `Login` take
  the guard's client as a parameter. `internal/gapi` was added at M2 for the
  same kind of reason: it builds the `*http.Request` every read goes out as and
  sets the bearer on it, and the `*http.Client` it sends them on is a parameter.
  Keeping request building in one room is what stops the bearer, the Accept
  header and the refresh rule from being written three slightly different ways
  in three reader packages. It is in the import allowlist and not the builder
  one, and that difference is the whole point of having two lists.
- The **builder allowlist** says who may construct an outbound client or reach a
  package-level dialer such as `http.Get`. That is `internal/guard` alone.
  Serving is not building: `internal/auth/loopback` runs an `http.Server`, which
  answers a request somebody else made, so it stays out of this set. A canary
  test states that rather than leaving it to luck.

The builder scanner resolves the import name per file and knows eight ways to
make a wire: a composite literal, `new(http.Client)`, a zero-value `var c
http.Client`, the dialers, a type declaration that renames the wire (`type C =
http.Client`, or the same without the equals sign), a struct holding one by
value, a container holding one by value (`make([]http.Client, 1)` and every
shape `holdsWire` walks), and a function handing one back by value (`func
Build() (c http.Client) { return }`). `holdsWire` recurses, so it also sees a
wire that is only a generic type argument. A struct field counts whether it is
embedded or named: `struct{ C http.Client }` is the same zero-value client as
`struct{ http.Client }`, reached through one extra word.

That is not thoroughness for its own sake. A scanner that assumes the name is
always `http` and looks only for composite literals is walked around by
`import nh "net/http"`, by `import . "net/http"`, by `new(...)`, by a zero-value
declaration, by `type C = http.Client; var _ = &C{}`, by `type T struct{
http.Client }`, by a slice, and by a named return, and every one of those builds
a wire outside the guard while passing both checks. The canary carries a case
for each, and each case was watched failing against the scanner that missed it.

The two type shapes are flagged on the declaration, not on the values built from
it. Following an alias would mean resolving names across a package, and a
package outside the guard has no reason to give the wire a second name. Embedding
a **pointer** is not flagged: that is holding a client somebody else made, the
same as taking one as a parameter.

Both fail in both directions, like v1's `test_guard_is_installed`. They fail
when an import or a builder spreads, and they fail when an allowlisted room
stops holding what it was listed for. The disappearance half reads production
files only: a `_test.go` that fakes the wire must never stand in for the room
that owns the wire.

### No external programs at all

v1 scopes that ban to `gdoc/render/` and keeps pandoc. v2 runs nothing: no
`os/exec` anywhere under `go/`, and nothing may add one. What makes the stronger
rule possible is that `auth login` prints the URL instead of opening a browser.
Opening a browser is the one thing a CLI usually shells out for.

`go/boundary/boundary_test.go` enforces it, the way
`tests/test_no_external_programs.py` enforces v1's.

### One module, named in `allowedModules`, and the rest still refused

The same file holds the dependency list to the tree: `allowedModules` in
`TestNoThirdPartyDependencies` names what `go.mod` may require, and every other
`require` line and every other module in `go.sum` is refused.

It held nothing at M1. M2 added one line, `github.com/goccy/go-yaml`, for the
`gdoc:` front matter, with the reason already written in SPEC.md and repeated in
the map. It brings no transitive modules of its own, and it costs about 1 MB on
each platform binary, measured in the M2 plan.

SPEC.md agreed three modules in total. The other two are `beevik/etree`, because
`encoding/xml` corrupts OOXML on the way back out, and `yuin/goldmark`, for the
markdown M5 parses. The milestone that first needs one adds its path to
`allowedModules` and nothing else. It does not delete the test, and it does not
widen it to "whatever `go.mod` says". A fourth module needs its reason in
SPEC.md before its line in the map, and the open candidate is `sergi/go-diff` at
M8.

`TestAllowedModulesAreReallyRequired` is the other direction: a path in the map
that `go.mod` no longer requires fails too. An allowlist naming something that
is not there stops describing the tree.

### The binary prints facts, and the skills judge

This is M2's line, and it is the one an agent is most likely to cross by being
helpful. Nail's call, 2026-09-06, recorded in `docs/v2/SPEC.md` and PLAN.md M2.

Nothing under `go/internal/` decides whether a comment is answered, whether a
suggestion that stopped being pending was accepted or thrown away, whether the
note and the document differ in a way that matters, or which side is the source
of truth. Every one of those is the skill's, reading what the commands print.

So the packages report a marker (`ai:`, `ai?`, `ai!`, `none`), a `resolved`
flag, a reply's `by_gdoc`, a witness of `anchored`, `detached` or `unmatched`,
and a list called `gone_since_last_look`. Each is a fact with a neutral name.
None of them is a verdict, and none may grow into one.

Two tests state the rule rather than leaving it to review:
`TestThreadsCarriesEveryFactAndJudgesNone` in `internal/comments`, and
`TestMatchGivesAnchoredDetachedAndUnmatched` in `internal/docx`. A field named
`handled`, `accepted`, `rejected`, `matters` or `drift` appearing under
`go/internal/` is either a fact wearing the wrong name or a defect.

### The three read commands

`read`, `comments` and `suggestions`. Each one takes the document URL Nail
pastes, or a bare id, and each does the same four things in the same order:
turn the argument into an id, open a `guard.Policy` holding exactly that id at
`LevelSuggest`, open a `gapi.Session` on the guard's client, and hand what came
back to a pure reader package. The readers take no client and touch no wire, so
every one of them is testable on a fixture, and the fixtures under `testdata/`
are the specification of what Google actually returns.

- `read <url> [--structure]`: the text projection, and the tree with the flag.
- `comments <url> [--since CURSOR] [--witness]`: the threads with their ranges,
  markers, replies and the next cursor.
- `suggestions <url> [--md PATH]`: what is pending, and with a paired file what
  stopped being pending.

Rules that hold across all three:

- **One Docs read, three views.** `documents.get` with
  `includeTabsContent=true`, `suggestionsViewMode=SUGGESTIONS_INLINE` and
  `commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED` carries the structure, the
  suggestion ids and the comment ranges in one call.
- **Drive is the source of threads, Docs is the source of ranges.**
  `comments.list` carries the replies, the authors, `resolved` and
  `modifiedTime`; the Docs read carries the character range, keyed by the same
  comment id. A thread the Docs read gave no usable range comes back with
  `range: null` and a warning, never a failed listing. The shape of the Docs
  `comments` key is measured rather than documented, so the decoder is loose on
  purpose, and unusable therefore covers two cases: no range at all, and a range
  that does not end after it starts. `comments` is the command whose output
  names the range as a position, and M3 places a proposal from it, so reporting
  a range `read` would refuse to mark would be a false fact in that field. The
  test is the same `Start >= End` as `internal/view`'s.
- **Argument parsing is strict.** An unknown flag, a repeated flag, a missing
  value, an empty value written either way, an extra positional argument, and a
  flag standing where another flag's value belongs each fail naming the
  offender. A command that accepts and ignores what it did not understand tells
  the caller it did something it did not. The last of those is why `parseArgs`
  looks the next argument up in the command's own flag set rather than refusing
  anything starting with a dash: a cursor is base64url, and `-` is in that
  alphabet.
- **Reads only.** Nothing in these commands POSTs to Docs or Drive. The single
  write anywhere is the snapshot in a local markdown file, and only when the
  caller named that file with `--md`.
- **`--witness` is a second read, not a field.** The docx export is the only
  truthful answer to whether a comment is still attached to text, so the witness
  costs an export. A thread is joined to an exported comment on its words and
  its author's name, because the docx carries no Drive comment id. An export
  that could not be read is a warning and every thread `unmatched`, not a failed
  listing. The export URL carries `mimeType` and nothing else: `files.export`
  defines two parameters, measured against the live Drive v3 discovery document
  on 2026-09-06, and `supportsAllDrives` is on `files.get` instead. Sending a
  parameter the method does not define is one the server may reject, and it
  would take every `--witness` run with it.
- Pictures, drawings, equations and objects print as `[image]`, `[drawing]`,
  `[equation]` and `[object]` placeholders, each with a warning. `[object]` is
  the embedded object the read could not classify: calling it an image would be
  a guess. Reading any of them is
  `docs/backlog/read-pictures-and-drawings.md`.
- A footnote's text is flattened, so a suggestion inside one is in neither
  `read`'s markers nor `pending`, and nothing warns. That is
  `docs/backlog/suggestions-inside-footnotes.md`.
- A 2xx body larger than the read's ceiling is an **error naming the ceiling**,
  never a short read. Truncating made the docx reader say "the export is not a
  docx" and the JSON reader say the answer is not JSON, both naming something
  the server did not do. A failed request's body is still cut, because
  `statusError` only reads Google's message out of it. The same rule holds one
  layer in, on a zip member: `docx.part` takes the ceiling as a parameter and
  refuses a `word/document.xml` over it, rather than handing the XML parser a
  document cut mid-element and blaming the export for a limit gdoc chose.
- Lists come back as `- ` items, two spaces of indent per level. A numbered list
  reads back as a bulleted one: telling the two apart needs the document's
  `lists` map, which this milestone does not read.

### `read`'s text, and why every marker is escaped

The text is a deterministic projection of one `docs.Document`: the same
document gives the same bytes, so a golden file is a specification rather than a
snapshot. Six markers, and the meaning of each:

| In the text | Means |
|---|---|
| `{+text+}[s:ID]` | a pending suggested insertion, with its suggestion id |
| `{-text-}[s:ID]` | a pending suggested deletion |
| `[[c:ID]]text[[/c]]` | the range a comment is attached to, `ID` being the Drive comment id |
| `<!-- tab t.0: Title -->` | the tab that follows, printed only when there is more than one |
| `# ` to `###### ` | a `HEADING_n` paragraph. `TITLE` and `SUBTITLE` are plain paragraphs |
| `[image]`, `[drawing]`, `[equation]`, `[object]` | content this milestone does not read |

**A comment range that does not end after it starts is a warning, not a
marker.** Armed, it puts its own close before its own open, and at the end of
the last run the closes-only drain emits the close and leaves the open behind.
Either way the text carries half a pair, which is exactly what the escaping
below exists to make impossible. The test is `r.Start >= r.End`, not equality:
the indexes come out of the Docs answer unchecked, the shape of the `comments`
key is measured rather than documented, so the loose decoder can hand over an
inverted pair. Inverted is the worse half, because it crosses the markers it
passes on the way.

**A literal `{+`, `{-`, `+}`, `-}`, `[[` or `]]` in the document's own text is
escaped with a backslash.** That is not tidiness. Without it a document that
quotes one of gdoc's own markers makes the AI read somebody's sentence as a
pending suggestion, and there is no way for it to tell. `escapePairs` in
`internal/view/text.go` has to stay in step with the constants above it, and the
escape advances by **one** rune rather than two: two literals can share a
character, so consuming both halves of `{-` in `{-}` walks past the `-}` behind
it and leaves half a marker in the text.

**The backslash is escaped too, as `\\`.** Without it the encoding cannot be
read back: a backslash the author typed in front of a marker looks like the one
gdoc writes, so a real comment anchor after a word ending in `\` reads as a
literal and is dropped, and a sentence the author wrote as `\{+text+}` reads as
a pending suggestion gdoc never marked. Both directions hand a marker to the
wrong side, which is the thing the escaping exists to prevent. The rule for a
reader is a parity: an even run of backslashes is the author's own text and the
marker behind it is gdoc's, an odd run ends in gdoc's escape and the marker
behind it is the author's.

The escaping is per run, so a marker whose two halves fall in two runs, or a
document character sitting against one of gdoc's own markers, still reaches the
text unescaped. That is `docs/backlog/escaping-across-run-boundaries.md`.

A run carrying both an insertion id and a deletion id prints twice, as the
deletion then the insertion, because that is what Docs shows. The comment
marker opens once, on the first copy.

`--structure` is the same document as a tree with character indexes on it, which
is what M3 needs to place a proposal at an exact position. Neither view is the
other's summary, so both are kept.

### The cursor is opaque, and it dies with the session

`base64url(JSON{"v":1,"t":"<RFC3339 UTC>","i":["<comment id>"]})`, holding the
newest `modifiedTime` seen across the comments and their replies, and the ids of
the threads whose own newest instant was that one. The instant keeps its **milliseconds**,
because that is what Drive sends: rounded down to the second, the floor sits up
to 999 ms below the activity the run just reported, Drive returns that thread
again, and the next cursor rounds down to the same second. The thread is then
news on every poll for ever. `ParseCursor` reads both shapes. The binary emits it, the caller
hands it back on the next poll, and **nothing writes it anywhere**: a live
session holds it in memory and it dies with the session. What must survive a
session lives in the front matter, and this does not.

**The precision is only half of that fix, and `Cursor.narrow` is the other
half.** `startModifiedTime` is documented as the *minimum* value of
`modifiedTime`, so Drive's bound is inclusive: handed the instant of the newest
thread the last run saw, it sends that thread back, and `NextCursor` cannot
advance past an instant it already holds. So the thread would be news on every
poll again, for the boundary reason rather than the rounding one. The request
stays inclusive on purpose, because asking for a window a millisecond later
would tell Drive to withhold a comment modified inside the cursor's own
millisecond, and losing a comment is the wrong direction to be wrong in. `Fetch`
narrows the answer instead: a comment is kept when its own `modifiedTime` or any
reply's `createdTime` is strictly newer than the cursor. The replies are in that
test for the reason `NextCursor` reads them, and an instant gdoc cannot parse
keeps its comment.

**The ids are the third part, and without them "strictly newer" loses a
comment.** Two comments can share a millisecond with only one of them reported,
when the poll landed between the two writes. The second one then sits exactly on
the cursor's instant, has never been seen, and no comparison on instants can say
so: Drive's precision cannot tell the two apart. So the cursor carries the ids it
reported at its own instant, and a comment on that instant is news unless the
cursor names it. `NextCursor` **adds** to those ids while the instant does not
move, and replaces them when it does. Adding is what makes the poll go quiet: a
thread `narrow` dropped is a thread reported on an earlier poll, and forgetting
its id makes the two threads sharing that millisecond take turns being news for
ever. What is left is a comment edited twice inside one millisecond, which is
Drive's precision rather than a choice made here.

A cursor written before `i` existed still reads, and the version stays 1 for that
reason: the ids only ever narrow further, so their absence costs one repeated
thread and never a lost one.

It is opaque on purpose. A caller that decodes the instant and does arithmetic
on it has made the encoding a contract, and it is not one. The version field is
there so a later shape is refused by name. A cursor that cannot be decoded is an
error naming the problem, never silently read as "from the beginning": that
would report a window nobody asked for and look like a clean poll.

### The `gdoc:` front-matter block, schema 1

`internal/frontmatter` owns one key in a note's YAML front matter and nothing
else in the file. It carries `schema`, `document_id`, `folder_id`, a `published`
record (M6 writes it), the `suggestions_seen` snapshot and `proposals` (M3
writes those; M2 defines the shape and carries them through untouched).

Four rules, and each one has a reason:

- **The read is strict.** `goccy/go-yaml` with `yaml.Strict()`: an unknown key,
  a key given twice, a missing `document_id`, a `document_id` that is not a
  Drive id, a `kind` that is neither `insertion` nor `deletion`. Each is refused
  naming the key, and the file is left untouched. A block gdoc half understands
  is a pairing it may act on wrongly. The front matter is one YAML document by
  construction, so there is no check for a second one: it closes at the first
  `---` or `...` line, which is where a second document would have begun. A
  delimiter is recognised with trailing spaces or tabs after it, and behind a
  leading byte order mark, because Jekyll, python-frontmatter and goldmark-meta
  all read those as front matter: a note gdoc reads as unpaired is a note
  `Write` puts a second block in front of, demoting the author's keys to prose.
  The
  author's own keys are checked too, and on every path: a file being paired for
  the first time has no `gdoc:` key at all, so checking only when one is already
  there would check every case but the first.
- **`schema` must be exactly 1.** A block stating another version is refused
  rather than read on a guess. `Schema` is the constant; bumping it is a
  decision, not a refactor.
- **The write is byte-preserving.** Only the `gdoc:` span changes. The author's
  keys, their order, the line endings and the trailing newline come through
  unchanged, and a file with no front matter at all gets the block added with
  new delimiters. `frontmatter.Read` and `frontmatter.Write` share one parse, so
  they cannot disagree about where the span is. A file whose opening `---` never
  closes is neither of those cases: `Read` reports no block, because there is no
  front matter to read, and `Write` **refuses** it. Writing there would put a
  second block in front of the author's keys and demote their own `gdoc:` key to
  prose, which is gdoc pairing a note it had just broken.
- **The write is checked against its own parse before it leaves the package.**
  A string carrying a control character is written double quoted, because the
  emitter writes it as a plain scalar the parser reads differently: a tab inside
  one is dropped on the way back in, and a bare carriage return produces a block
  that fails to parse at all. Google Docs puts a tab in a text run wherever the
  author typed one, so this is the snapshot's own words. `verify` then renders
  the block, reads it back and renders it again, and refuses a block whose two
  renderings differ. The reason it has to be a refusal rather than a warning is
  that `Write` reads the block it finds before replacing it: a block gdoc broke
  is a note gdoc would then never touch again.
- **The snapshot is written after a successful read, never before.** A read that
  failed knows nothing about what is pending, and a snapshot taken then would
  report everything this run could not see as gone on the next one. The write
  goes through `internal/atomicfile`, a temp file in the same directory and a
  rename, keeping the file's mode, because the markdown is the source and gdoc
  is not its only reader. A file whose block names another document is refused
  rather than repaired.
- **What is pending is a question about ids.** `suggestions.List` drops the
  suggestions whose text is only whitespace, which say nothing a reader can act
  on, so `GoneSince` is given `suggestions.IDs` instead: a suggestion the author
  has since edited down to a space is still in the document, and putting it in
  `gone_since_last_look` would be a false fact in the one field the skill judges
  accepted-or-rejected from. The snapshot is written from `suggestions.All` for
  the same reason, and not from the listing the run prints: a snapshot built
  from the filtered list forgets that suggestion, so the run that later sees it
  accepted or rejected has no record it was ever there and reports nothing.

### `GrantInPlace` is gone until M7, and `AllowCreateIn` stayed

PLAN.md M2 asked that a guard door with no production caller be deleted rather
than carried. `GrantInPlace` had none, so it went, with its tests. M7's in-place
restyle adds it back beside its caller, and the level it raises to is a decision
for Nail then, not something to restore from git because a test wants it.

`AllowCreateIn` stayed even though M2 calls it nowhere. The transport's whole
create path is built on it: the parent check, the upload-shape check and the
response learning all read it, and deleting it would mean deleting the create
half of the guard that M6 needs. `AllowFile` and `Token.Refresh` got their first
production callers here, which is the other half of what M2 was asked to settle.

### Building

| Command | Does |
|---|---|
| `make test` | `cd go && go test -race ./...` |
| `make vet` | `go vet ./...` and the `gofmt -l` check |
| `make build` | `bin/gdoc`, for this machine |
| `make dist` | the three platform binaries |

`GDOC_LIVE_TEST=1` runs the one opt-in end-to-end test, in `go/internal/live`.
It then needs `GDOC_LIVE_DOC_ID=<document id>`, and there is no default: the
guard is opened with exactly the document the run names. It reads, and creates
nothing on Drive. `GDOC_LIVE_RECORD=1` additionally saves the Docs read and the
docx export into `testdata/`, which is a real document's content, so a person
redacts those before they are committed.

`make dist` builds darwin/arm64, darwin/amd64 and windows/amd64 with
`CGO_ENABLED=0`, so each one is static and the binary is the whole dependency.
There is no linux target. Cross-building proves the binaries link, not that they
run, so the real Windows smoke test belongs to M9.

`.github/workflows/go.yml` runs all four on every push: gofmt, vet, the raced
test suite, and `make dist`. dist is in CI because a cross-compile break is
invisible to whoever is working on macOS, and "portability is never discovered
late" is only true if something checks it on every commit rather than when
somebody remembers. The workflow runs on ubuntu and builds no linux binary,
which is fine: cross-compiling is what is being proved.

## One root, and it is never this repo

Principle 2. Read it there.

In code: `--repo-root` is the tree scanned for markdown with `gdoc:` frontmatter,
and the tree `.gdoc/` is written into. In practice it is the folder holding the
source document, in a vault such as `Altery-Platform-Hub`.

## Nothing runs git

Principles 1 and 3, and the decision dated 2026-08-18. This is the constraint an
agent is most likely to break, usually by adding a helpful commit.

- No module and no skill runs git. Not to commit, not to check whether the root
  is a repository, not to ask whether a file is dirty. `gdoc-apply` names every
  file it changed and stops there.
- `write_baseline` refuses to overwrite an existing `baseline.md` unless `force`
  says so. It asks nothing and nobody: not knowing must never resolve to
  "overwrite", and in a synced vault git could not have answered anyway.
- `generate` is the only caller that writes a baseline, and it passes `force` at
  the one moment the document and the markdown provably match.
- `install.sh` still reads git, and that is about this repository rather than
  about the user's documents.

## No external programs

Principle 1. The rule is scoped to `gdoc/render/`, the publish path.

**Decided 2026-08-29: the next version is written in Go**, and the reason is this
section. Read the decision in PRINCIPLES.md before proposing work that assumes
otherwise. In Go the first two uses below stop existing, because a Markdown parser
is an import rather than a spec. The third, the docx reader, has to be written and
is unproven. Nothing here is deleted until the Go side covers it: this repository
is still the tool.

pandoc remains, in three places, and each is tracked separately:

- `gdoc/render/body.py`, as the markdown parser `build` depends on. Load-bearing.
  Removing it means a pure-Python parser plus a rewritten AST walker, in the
  module where the document body's pixel fidelity lives, so it needs its own
  spec.
- `gdoc/export.py`, twice. As a markdown fallback, and as the way to carry
  pictures a Google Drawing holds. What Drive's `text/markdown` export does with
  pictures depends on the kind, verified against real documents on 2026-08-18
  and 2026-08-19: an **embedded image** comes back as a reference-style link to
  a base64 `data:` URI, and a **Google Drawing** does not come back at all. The
  docx export carries both as PNG bytes, so `export_with_media` always takes the
  docx route and runs pandoc with `--extract-media`. That makes this use
  load-bearing, not a fallback, and removing it now needs a docx reader.
- `gdoc/generate.py`, for the plain non-template path, which is what
  `--template none` selects. Kept on purpose: it is the fallback when the house
  template is not wanted, and keeping it narrowed the blast radius of wiring the
  template in.

`tests/test_no_external_programs.py` enforces this. The five removed program
names must not appear in code under `gdoc/render/`, and the set of modules
there importing `subprocess` must be exactly `{body.py}`. That is an allowlist,
not a ban: it fails if `subprocess` spreads to another module, and it fails just
as loudly if body.py's own dependency vanishes without the test being updated.

Page numbers are the one real cost. They do not exist until something lays the
document out. The intended flow makes Google the layout engine: upload once with
blank numbers, read which page each heading landed on out of the PDF export, write
those numbers in, then upload the version that gets published.

`gdoc generate` runs that flow. It builds with blank page numbers, uploads that
copy, exports it as PDF, reads the pages back, builds again with the numbers in,
uploads the version that gets published, and trashes the measuring copy. The
published copy is measured too, so a contents list that disagrees with its own
document comes back in the result as `drift` rather than passing quietly.

The template comes from `--template`, or from `template` in the config, which
defaults to the bundled profile. `--template none` keeps the plain
`pandoc md -o docx` path.

`tests/test_contents_integration.py` still composes the two passes against live
Drive, and is opt-in behind `GDOC_LIVE_PUBLISH_TEST=1`. It is the only test that
creates real documents.

`gdoc.render.build` on its own has no pagination to offer, so it leaves the page
numbers blank. Any desktop refresh fills them in, and a wrong number would be
worse than a blank one.

## The client reaches only the files it was given

Principle 3, and the decision dated 2026-08-15. Read it there.

In code: `drive_service(doc_ids=...)` wraps the transport in
`gdoc.guard.GuardedHttp`, which carries a request only when the file it
addresses is in its set. The set is seeded from the CLI, where `read`, `reply`,
`export` and `capture` each pass the document they were pointed at and
`generate` passes the output folder, and it grows only when a create the guard
itself carried comes back with an id.

Two rules an agent is likely to break:

- The set has exactly two doors, the ids passed in and the ids learned from a
  create. Never add a third, and never widen it to make a test pass. A refused
  call usually means the command did not say which document it was for.
- `gdoc/auth.py` is the only module that may call `build()`.
  `tests/test_guard_is_installed.py` enforces it, as an allowlist: it fails if
  the call spreads, and it fails if it moves.

## Which credential, and who decides

`gdoc.auth.resolve_auth_mode` is the one place. It is a pure function of the mode
the config states plus which credential files exist, so it never reads the config
itself and a test can hand it either case.

- A stated mode wins outright. Nothing on disk may overrule the author's word,
  and a mode it does not recognise is refused rather than inferred past. A typo
  must never resolve to oauth, which is the credential with the wider reach.
- An unstated mode is inferred: a token means oauth, a key with no token means
  service_account, neither means oauth.

`auth._config()` returns the defaults for a **missing** config only. A config that
exists and cannot be understood reaches the caller, so every command fails naming
the problem. `gdoc auth status` is the one place that catches it, reports
`auth_mode: null` with source `unknown`, and never guesses.

`Config.auth_mode` is `None` when the file does not say, and that is the point.
Defaulting it to oauth in `load_config` would flip every working service_account
install the moment it upgraded, because a config written before the key existed
cannot mention it. Never give that field a default.

Two consequences an agent is likely to break:

- `gdoc auth login` writes `auth_mode: oauth` through `config.write_auth_mode`,
  after the browser flow returns and **before** the account lookup. Later would
  mean a network failure leaves the person signed in with the old credential
  configured. `--token` skips the write entirely and says so, because nothing
  else reads a custom token path. `gdoc auth use <mode>` is the same write, in
  either direction, so no setup step is ever a hand edit of JSON.
- `write_auth_mode` carries every other key over, `display_name` included, and
  refuses a file it could not parse rather than replacing it. It writes through a
  temporary file and `os.replace`, so a failed write cannot leave an empty config.
  Truncating first lost settings that were readable a moment earlier, and the
  empty file then blocked the next write too.

`tests/conftest.py` redirects the config and both credential paths into a tmp
directory for every non-integration test, autouse. Without it a test that runs
`auth login` edits the developer's real config, and resolving an unstated mode
would answer differently per machine. Integration tests are exempt, because the
real credential is their point.

## The OAuth client is shipped, and stays Internal

`gdoc/oauth.py` holds `BUNDLED_CLIENT_ID` and `BUNDLED_CLIENT_SECRET`, so nobody
using gdoc visits a cloud console. Both rules matter:

- **The secret belongs in version control.** RFC 8252 section 8.5: a secret
  shipped to many users "should not be treated as confidential" and serves no
  purpose "beyond client identification". `gh` ships its own with the comment
  "This value is safe to be embedded in version control", and `gcloud` ships a
  Google secret in a constant named `CLOUDSDK_CLIENT_NOTSOSECRET`. Do not
  "fix" this by moving it to `~/.config/gdoc-agent/`. What protects an account
  is the per-user token, which never leaves the machine.
- **The client must stay User type Internal.** gdoc needs the full Drive scope,
  which Google classes as restricted. Internal exempts gdoc from verification,
  the unverified-app screen and the 100-user cap. External would mean a CASA
  assessment every 12 months, and refresh tokens expiring weekly.

A file at `~/.config/gdoc-agent/oauth-client.json` wins over the bundled client.
That is an override for quota, not a setup step, the way gcloud treats
`--client-id-file`. One shared client shares one Google rate limit, which is why
rclone is retiring its shared Drive client during 2026.

`oauth.client_config` is the one place that decides, and it returns where the
client came from so `gdoc auth status` can report it.

### The one thing that changes this decision: making the repo public

Nail decided on 2026-08-18 to keep the client in git, on the RFC and on the `gh`
and `gcloud` precedent. That decision assumed a private repo, and one condition
would break it.

GitHub secret scanning carries a **partner** pattern for
`google_oauth_client_id, google_oauth_client_secret`. On a public repository it is
reported to Google, who may revoke the client. So publishing this repo would
break every colleague's login at once, without warning and without a commit to
blame. rclone obfuscates its Google secret for exactly this reason, which is
evasion of automated revocation rather than security.

So before this repo is ever made public: create a fresh client, distribute it as
a file out of band, and clear these two constants. Do not obfuscate them to get
past the scanner.

## Pictures

Two halves, and both were broken.

- **Pulling a document in.** Plain `gdoc export` is enough for an embedded
  image: Drive hands it back as a base64 `data:` URI and the publish decodes it.
  `--media-dir` is what a Google Drawing needs, because Drive leaves those out
  of the markdown altogether; it takes the docx route instead. A picture in the
  docx that pandoc never extracts, usually a header image, is reported in
  `warnings` rather than dropped: silence is what made #28 show up only after
  somebody read the new document.
- **Publishing it back.** `gdoc/render/body.py` embeds a picture wherever it
  finds one: alone on its line (`lone_image`), inside a paragraph, and inside a
  heading. `split_images` searches **recursively**, and that is not a refinement.
  Drive writes a picture on its own line as a bold heading,
  `# **![][image1]**`, so the image sits inside a `Strong` node, and a top-level
  search finds nothing and drops it in silence. That was the 2026-08-19 bug.
  A heading holding nothing but a picture emits no heading at all: it takes no
  number and no contents entry, because it is a figure.

A picture at an http URL is refused with a message naming `--media-dir`. Nothing
here downloads: a publish that reached the network would depend on a link that
expires. A `data:` URI is decoded instead, because those bytes arrived with the
document.

## Restyling a document gdoc knows nothing about

`gdoc restyle` publishes a house-styled copy of a document with no queue, no
paired markdown and no baseline. `gdoc/restyle.py` composes the pull, the
publish and the comment copy; `gdoc/comments.py` is the copy on its own.

Four rules, and each one is the reason something is shaped the way it is.

- **The folder is never derived from the original.** It comes from
  `--folder-id`, or from `output_folder_id`. Reading the original's parent and
  creating there would work today, because a create names no file, but it would
  mean reaching a folder Nail never named. That is the third door principle 3
  forbids. So `cli.cmd_restyle` hands `drive_service` exactly two ids, the
  document and the folder, and the new document's id is learned from the create
  that made it. `gdoc/restyle.py` never builds a client; it is given one.
- **It is one command, not three the skill composes.** Only because of the
  guard: the comment copy writes to a document that did not exist when the run
  started. In one client its id is already learned. Split across CLI runs the
  last one would have to be handed an id from outside.
- **The front matter is built with `yaml.safe_dump`, never an f-string.** The
  only field in it is the title, and that title is a Drive document's name.
  "Q3: Roadmap" written by hand is `title: Q3: Roadmap`, which is not a mapping,
  and the run dies parsing its own front matter before it publishes anything.
- **Nothing is recorded.** No pairing, no version, no baseline, and the pulled
  markdown is deleted. A restyle is a throwaway by design, so running it twice
  makes two unrelated documents. The temp directory survives only when nothing
  was created, and then `workdir` names it, because the pull is all the run
  produced.
- **A copied comment is somebody else's text, so it is carried verbatim.**
  `gdoc/comments.py` does not run `reply.assert_plain_text` and does not add the
  `[gdoc]` marker. The check guards what gdoc writes, which gdoc can always
  rephrase; refusing to carry a comment over an asterisk would lose the comment
  to protect its formatting. The marker means "gdoc wrote this", and stamping it
  on other people's words would claim authorship of text gdoc only moved.

`restyle.survey` is the same read without any of the writes, behind
`--dry-run`. The confirmation the skill asks for has to name the folder and the
thread counts, and both have to be on screen before a document exists.

Two things cannot come across, and no code should try. Drive creates every
comment as the authenticated user, so every copy is authored by whoever gdoc is
signed in as. The author's name in the text is the only honest record of who
said it. An anchor is the second: it is computed against a document's
structure, and the house template changes that by construction. So the copies
are unanchored, and the quoted sentence is the pointer that is left.

## Reading what was changed in the document

`gdoc edits` answers "what did Nail change in the browser", and it writes
nothing. Porting a hunk into hand-written markdown is judgment, so the CLI
reports and `gdoc-apply` decides.

Two sources, because one of them lies. The markdown export diffed against
`baseline.md` sees edits made in editing mode. It does not see suggestions at
all: Drive renders a suggested document as though nothing had been suggested, so
a review done entirely in suggesting mode diffs to nothing. `gdoc/suggestions.py`
reads those through `documents.get` with `suggestionsViewMode=SUGGESTIONS_INLINE`.

Three rules an agent is likely to break:

- **The Docs scope is asked for separately.** `SCOPES` is Drive and stays Drive.
  `DOCS_SCOPES` is `documents.readonly`, requested only by `docs_service`, so a
  token issued before this existed keeps every other command working and only
  `gdoc edits` says to log in again. `LOGIN_SCOPES` is both, so one browser trip
  covers it.
- **Read only, and it stays read only.** gdoc reads a suggestion as an intention
  and applies it to the markdown. It never accepts one in the document.
- **A baseline is only diffed when it belongs to this document.**
  `baseline.json` beside it records the document id it was taken from, and every
  version is a new document, so a mismatch is the stale case. `baseline_state`
  returns `ok`, `unverified`, `missing` or `stale`, and `edits` reports which
  rather than refusing to run.

Both skills read it, and they read it for different reasons. `gdoc-review` reports
the counts and touches nothing: a review that lists only comments reads as
"nothing else changed", which is the sentence that loses an edit. `gdoc-apply`
applies them, and does not publish when it could not read them. A
`suggestions_error`, or a `missing` or `stale` baseline, means half the review was
invisible on that run, and a version built from the markdown anyway would drop
it silently. The stop is in the skill rather than in `generate`, because
`generate` is handed markdown and cannot know a document was reviewed at all.

## Identity is never a gate

Drive's `author.me` means the service account under `auth_mode: service_account`
and Nail under `oauth`. Nothing may branch on it to decide whether a comment is
work: that would skip every comment Nail writes. The marker decides.

`[gdoc]` on the last line of a reply is how gdoc recognises its own replies.
`Reply.by_agent`, which is `me`, is kept in `has_agent_reply` for one reason
only: threads the service account answered before the marker existed carry no
marker, and dropping it would answer them twice.

## Skills are linked, not copied

`~/.claude/skills/gdoc-review` and `gdoc-apply` are symlinks into `skills/` in
this repo. There is one copy of each SKILL.md, so editing it here changes what
Claude Code loads. No copy step, and no way for the skill to drift from the CLI
it calls.

Two consequences worth holding:

- An edit is live the moment it is saved, before it is committed. Nothing warns
  you. `./install.sh` prints `+ uncommitted changes` when the tree is dirty.
- Deleting or moving `skills/` breaks the installed skills. Re-run
  `./install.sh` after any move.

`install.sh` is safe to re-run. It refuses to replace a real
`~/.claude/skills/<name>` directory whose contents differ from this repo, so an
older copy-based install cannot be destroyed silently.

## Testing

TDD. Write the failing test first.

```bash
~/.config/gdoc-agent/venv/bin/pytest
```

`tests/test_access_integration.py` calls Drive and skips without credentials.
Never make a test pass by loosening an assertion about what the credential can do.

### A house-style test must never read the constant it tests

`assert style.spacing == HEADING_LINE_SPACING` is worth nothing. It is a mirror:
set the constant back to the master's single spacing and the assertion follows it
and still passes. Write the house value as a literal instead, so the test says
what the house style is. `tests/test_render_shell.py` does this on purpose, and
where a value has a safe range rather than one right answer it asserts the floor
instead, as with the 40-twip cell margin.

## Never

- Never edit a reviewed Google Doc. Under `service_account` the credential
  cannot. Under `oauth` it could: `gdoc/guard.py` bounds which files are
  reachable, not what may be done inside one. Nothing in gdoc edits a document,
  and nothing may start.
- Never commit anything from `~/.config/gdoc-agent/`.
- Never post markdown into a comment thread. The CLI refuses it for a reason.

## Writing style

Plain, short English. No em dashes.
