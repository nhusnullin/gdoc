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
| `go/cmd/gdoc/` | `main.go`, `read.go` and `write.go`. Arguments in, one JSON object out, exit |
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
| `go/internal/probe/` | the throwaway document that asks whether SUGGEST is honoured today |
| `go/internal/reply/` | one 🤖 reply into a thread, and the thread read back |
| `go/internal/propose/` | a change as a suggestion, its 🤖 comment, and the three read-backs |
| `go/internal/withdraw/` | gdoc taking back one of its own pending proposals |
| `go/internal/plaintext/` | the one rule about what gdoc may write into a comment thread: the 🤖 prefix, and no markdown |
| `go/internal/frontmatter/` | the `gdoc:` block in a note's YAML front matter, and nothing else in the file |
| `go/internal/atomicfile/` | the temp-file-and-rename write. The one room that replaces a file's contents |
| `go/internal/live/` | the two opt-in end-to-end tests, one read and one write. Tests only, no production code |
| `go/boundary/` | the two allowlist tests that keep the wire in one room |
| `bin/` | what `make build` and `make dist` write. Not in git, so both targets create it |

`docs/v2/SPEC.md` is the agreed design and `docs/v2/PLAN.md` the milestone
order. Milestones 1, 2 and 3 are done: the binary exists, prints the envelope,
owns the network, can log in and report its OAuth state, reads a document three
ways with `read`, `comments` and `suggestions`, and writes four ways with
`probe`, `reply`, `propose` and `withdraw`. The review skill is rewritten over
those, and `gdoc` on PATH is v2 from M3 on.

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
than an oversight. There is no help command: `dispatch` matches the nine
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

`commentWrites` carries `POST` and nothing else: no `PATCH` and no `DELETE`, on
a comment or on a reply. Nothing in a comment id says who wrote it, so the guard
cannot tell gdoc's own comment from somebody else's, and no command needs
either method. The milestone that needs one adds it back beside its caller, the
way `GrantInPlace` returns at M7. `uploadShapes` is `multipart` alone for a
related reason: a resumable create is two legs, the guard carries neither the
`PUT` nor `upload_id`, so the shape could never finish and a half-permitted
route reads as a working one.

A create is refused unless it names exactly the one folder the run was given,
and the transport reads the create's response for the new id and teaches the
policy. Those are still principle 3's two doors, ported. Naming a folder to
create in does not put that folder in the reachable set: it is a create target,
not a third door.

`AllowReject` is the same kind of thing as `AllowCreateIn`: a per-run grant
naming one object, not a level and not a file. The guard refuses every
`batchUpdate` request kind whose name carries "suggestion", and `AllowReject(id)`
opens exactly one shape through that wall, a `rejectSuggestion` spelled exactly
and carrying exactly `{"suggestionId": <that id>}`. `cmdWithdraw` seeds it from
the note's `proposals[]`, which is the only record of what gdoc itself wrote.
Nail's decision, 2026-09-07; read "A withdrawal is a `rejectSuggestion` on
gdoc's own id" below for why a delete could not do the job.

**Read this before trusting the level-1 write bar.** What keeps a handed-in
document read-and-suggest only is `writeControl.writeMode == "SUGGEST"` in the
request body, which is a field the client itself supplies.
`docs/v2/BLOCKED-BY-API.md` records the measurement: `writeMode` is absent from
the public Docs discovery document, and one morning this exact call returned 200
and silently made a direct edit. Measured again on 2026-09-07, on a throwaway
document: an `insertText` at index 30 in SUGGEST mode came back 200 and the
read-back carried `suggest.xyh4cb4emh7y`, so the project is enrolled today.
Enrolled today is not a guarantee for tomorrow, and the earlier measurement is
what says so. The field is still a statement of intent rather than a guarantee,
and what makes a write trustworthy is the capability probe before it and the
read-back after it, neither of which lives in the guard. Widening or narrowing
what `isSuggestMode` permits is a decision for Nail, not a refactor.

`isSuggestMode` reads both keys **exactly**, and refuses a body where two keys
fold to either name. Google's proto-JSON is case-sensitive, so `WRITEMODE` is
not the field the server reads, while `encoding/json` matched it: the guard must
never be broader than the server on the one field that permits a write.
`hasDuplicateKeys` is the same rule one layer out. It walks the body as a token
stream and refuses any object that names a key twice, folding case, and
`judgeRequests`, `checkCommentWrite` and `checkParent` each call it first.
`encoding/json` keeps the last copy of a repeated key and drops the rest, so a
body carrying two `requests` lists, two `parents` lists or two `action` fields
is judged on one copy and may be served on the other.

**The walk reads numbers as `json.Number`, and a body it cannot walk is
refused.** Both halves are one bug. `json.Decoder.Token` decodes a number into a
float64, so a literal out of that range such as `1e999` ended the walk with an
error, while the callers, which unmarshal into `json.RawMessage` and a
`[]string`, never parse the number and accept the same body. Swallowing that
error carried a `batchUpdate` naming `requests` twice, with a `deleteSuggestion`
in the copy the guard never read. With `UseNumber` the walk ends early only on a
body that is not valid JSON, which every caller refuses on the line after, so
failing closed there costs a message rather than a request.

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
  the parent check can read, `multipart` and an absent parameter, and refuses
  the rest, `resumable` included. It reads `upload_protocol` too, which is the
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
  `comments` key was measured on 2026-09-06, on a real document, because the
  reference does not describe it: each entry carries `commentId` and an
  `anchorId`, and the range sits in the tab under
  `documentTab.commentAnchors[anchorId].ranges`. `docs.anchoredRange` reads
  that first; the three shapes guessed at before the measurement stay as
  fallbacks. `go/internal/docs/testdata/anchors.json` is the measured shape
  with placeholder text, and the recording it was modelled on holds a real
  document and is gitignored. Unusable therefore covers four cases: no range at all, a range
  that does not end after it starts, a range whose ends fall outside the tab's
  text, and a range naming a tab the document does not have. `comments` is the
  command whose output names the range as a position, and M3 places a proposal
  from it, so reporting a range `read` would refuse to mark would be a false
  fact in that field. **One rule decides, and it lives in `docs`.**
  `docs.Document.Places` is what `comments` reports from and what
  `internal/view` warns from, so the two commands cannot drift apart. It is that
  rule over every tab, and `docs.Tab.Places` is the same rule for one tab.
  `view` arms its markers from the tab's form, because the markers go into the
  tab being walked: two tabs sharing an id, which happens when a tab carries no
  `tabId` and takes the default `t.0`, would otherwise arm one tab on the other
  one's indexes and leave an opening marker with no close. On a document whose
  tab ids are unique the two forms answer the same. `view` keeps its own two
  warning messages, which name which half of the rule the range failed, and the
  gating is `Places`'s. An inverted pair is the worst of
  the four, because it crosses the markers it passes on the way.
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
  listing. **Two exported comments that match one thread and disagree about
  being anchored give no answer**: the thread comes back `unmatched` rather than
  taking the first. Neither side is ordered against the other, Drive's
  `comments.list` defines no ordering and `word/comments.xml` is numbered by the
  export, so first-fit would hand one thread id the other's witness. `--since`
  reaches it with one thread in view, because the listing is narrowed to the
  cursor window and the export is not. The export URL carries `mimeType` and nothing else: `files.export`
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
- A table is a pipe table, and a `|` the author typed inside a cell is escaped
  as `\|`. Unescaped it is a column separator, so a two-cell row holding
  `A | B` reads back as three columns under a two-column separator: an ordinary
  cell value would change the table's shape. The escaping happens on the row,
  after the marker escaping has doubled the author's backslashes, so the parity
  rule a reader uses on a marker holds on a pipe too.

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
the indexes come out of the Docs answer unchecked, and the decoder trusts the
measured `commentAnchors` shape without checking the numbers, so it can hand
over an inverted pair. Inverted is the worse half, because it crosses the markers it
passes on the way. A range whose ends are outside the tab's text, or naming a
tab the document does not have, is the same answer for the same reason, and all
of it is `docs.Document.Places`, the one rule `comments` reports from too.
`docs.Tab.Places` is that rule asked of one tab, which is what the walk arms
from: the document form answers off the first tab carrying the id, so on two
tabs sharing one it would arm the tab being walked on the other tab's text.

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

`--structure` is the same document as a tree with character indexes on it.
M3 was expected to place a proposal from it and does not: a proposal names text,
`propose` reads the document itself and finds the words, and the index never
leaves the run that computed it. Read "A proposal names text, never an index"
below. The view is kept because neither view is the other's summary, and because
an index view is what a later milestone would need if one ever places a change
without quoting it.

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
record (M6 writes it), the `suggestions_seen` snapshot and `proposals`, which
`propose` writes and `withdraw` reads and shortens. A proposal is
`{id, comment_id, at, quoted}`: the suggestion id, the comment `insertComment`
returned, the time, and the words that were replaced. `quoted` arrived in M3 and
is optional in the decoder, so a note written under M2 still reads.

Six rules, and each one has a reason:

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

### The four write commands, and none of them trusts a success

`probe`, `reply`, `propose` and `withdraw`. They are the read commands with one
thing added, something leaves the machine, so the shape is the same: turn the
argument into an id, open a `guard.Policy` holding exactly that id, open a
`gapi.Session`, hand it to a writer package. The writers make one request, read
the answer, and then read again through a route the write did not go out on.

**Every change inside a Google Doc is a suggestion, and the guard holds it.**
Not the call site. A `batchUpdate` on a handed-in document is refused inside the
process unless the body says `writeMode: SUGGEST`, which is `LevelSuggest` and
is the only level a handed-in id ever gets. Nothing in these packages can widen
that by phrasing a request differently. Read "the guard owns the wire" above for
what `isSuggestMode` reads and why it reads it exactly.

**The probe runs on every `propose`, and it creates its own document.** What
makes a write a suggestion is `writeMode`, a field the client supplies, absent
from the public discovery document, and one morning the same call returned 200
and made a direct edit. `docs/v2/BLOCKED-BY-API.md` holds both measurements. So
the guard's refusal is about gdoc's own words, and the probe is the question of
what Google does with them: create a document in the folder the run named, write
a sentence directly, suggest a word inside it, read back with
`SUGGESTIONS_INLINE`, then trash it and confirm the trash. `enrolled` is true
only when the word came back carrying a suggestion id. A word that came back as
plain text is `enrolled: false`, and `propose` sends nothing.

Three rules about the probe that are decisions rather than details:

- **It runs every time, and nothing caches the answer.** The binary is one-shot
  and holds no hidden state, so there is nowhere to put it, and an asserted
  "already probed" from outside reopens the exact failure the probe prevents.
- **It never touches the document being reviewed.** `propose` opens one policy
  with two doors: the document at `LevelSuggest`, and the probe's folder as the
  one place a create may land. The probe document's id is learned from the
  create the guard carried, never handed in.
  `TestTheProbeDocumentIsNeverHandedIn` states it.
- **Every failure path still trashes, and the report names the document.** A
  probe document left behind is named in `probe_document_id` with
  `trashed: false`, never silent. The one case it cannot name is the create
  Drive accepted whose answer could not be read: the id was in that answer, so
  there is nothing to put in the field and nothing to trash. The failure says
  the document **may** be in the folder rather than that it was not created,
  because Drive made it. The guard learns the new id from the same answer, so it
  would refuse the trash in any case, and it warns.

The probe is `AllowCreateIn`'s first production caller. PLAN.md expected that to
be M6's publish, and M3 arrived first.

**A proposal names text, never an index.** The skill hands over the exact words
to replace, `propose` reads the document fresh and finds them, and refuses when
they occur more than once: quote more of the sentence. Text already inside a
pending suggestion does not match, so a proposal on top of a proposal is
refused. An index computed a minute ago is the hazard the whole API has, and
DECISIONS.md says never to act on a stored one. Index lengths on the wire are
UTF-16 code units, which is what the Docs API counts, so a replacement carrying
a non-BMP character has its own test.

**A match has to be contiguous, and that is not a detail.** `internal/propose`'s
index walk reads text runs and skips the rest, but the document numbers what it
skipped, so words either side of a footnote mark, a picture, an equation or a
page break read as one string in the walk and are two spans in the document.
`matches` refuses a match whose span is longer than the words in it, because the
`deleteContentRange` built from one would mark the skipped content for deletion
along with them, and all three read-backs would still pass: the inline check
reads text runs, so the footnote it just proposed deleting is invisible to it.
`withdraw.Span` refuses two spans with somebody else's words between them for the
same reason, and this is that rule on the other side. Such a quote comes back as
a refusal naming what it crossed, not as "not found".

Two things about that check are decisions rather than details. **The span's end
comes from the last rune of the match, never from the byte behind it.** The
position of that byte is the start of the next indexed run, so a quote ending
exactly where a footnote mark or a picture begins would measure a unit too long
and be refused for crossing a hole it only touches. That is the one refusal a
reader walks straight into: `read` prints a footnote reference as `[^1]`, so the
obvious sub-quote is the words right before the mark. **A crossing occurrence
still counts towards the exactly-once rule.** `FindSpan` refuses a quote that
occurs once as written text and once across a hole, rather than placing it on
the contiguous one: picking would choose for the caller, and `Carries` rests on
that same guarantee, so a dropped crossing copy is one the preview check would
still find after a direct edit took the other.

**A proposal replaces words with words.** An empty `replacement` is refused in
`Proposal.Check`, before the probe: the batch would carry an `insertText` with no
text and a comment anchored on a range of length zero, which Docs rejects, and
`inlineHolds` looks for an insertion a plain deletion never makes, so it could
never verify either. A milestone that wants a deletion-only proposal gives it its
own request shape.

**A line break is refused on both sides, in the same place, for two different
reasons.**

A `quoted` carrying one is refused because a paragraph's last text run carries
the paragraph mark itself: the Docs read hands back `...operations team.\n`, so
a quote ending in a newline matches inside that one paragraph and the span
`FindSpan` returns ends past the mark. The `deleteContentRange` built from it
marks the mark for deletion, and accepting the suggestion merges the paragraph
with the one behind it while the `insertText` puts back a replacement that
cannot carry a break. Nothing downstream would name it: `inlineHolds` compares
the deleted runs against that same `quoted`, and `Carries` finds that same
string in the preview, so all three read-backs hold over a proposal that removes
a paragraph. A quote with a break in the middle is already unreachable, because
it spans two paragraphs and the walk indexes one at a time, so the rule costs a
caller nothing: the words without the trailing mark are always writable instead.

A `replacement` carrying one is refused because of the preview check. `Carries`
asks one paragraph at a time, and a paragraph ends at its own break, so a string
whose newline is anywhere but the very end is in no single paragraph and comes
back false whatever the document holds. A trailing one is the exception: a
paragraph's last run carries the mark, which is the quote rule above, so a want
ending in a newline can be found. Either way the answer is worthless. The quote
rule gives the first question a
`quoted` that cannot carry one. Nothing gives the second question that, so a
replacement containing both the quote and a newline would fall past the
ambiguity arm and report `preview_without_suggestions` as holding on exactly the
silent direct edit that route exists to name. Both preconditions are enforced at
the door rather than left implied.

**Every proposal in the file is checked before the first one is sent.**
`readProposals` runs `Proposal.Check` over the whole list, for the reason it
refuses an empty list: all of them are in hand, and a third entry turned down
after the first two have landed is a run that half happened in somebody's
document, with a probe document created and trashed on the way.

One `batchUpdate` per proposal, three requests inside it, in this order:
`deleteContentRange` over the quoted span, `insertText` at its start, and
`insertComment` over the inserted span. One batch because Docs applies the
requests in order with consistent indexes, so the comment lands on the span the
insert made. Two proposals are two batches, each after its own fresh read,
because the first moves the ground under the second. A document with more than
one tab stops the run before anything is sent, the probe included: a range means
nothing without saying which tab it is in.

**`verified` is three read-backs, and `verified: false` is not a failure.**
`Checks` carries them as three fields, and each answers something the other two
cannot:

| Check | Asks |
|---|---|
| `suggestions_inline` | the replacement is in the document, carrying a suggestion id |
| `preview_without_suggestions` | the quoted words are still somewhere in the tab with suggestions hidden, so it is a suggestion and not an edit |
| `docx_anchored` | the docx export carries the 🤖 comment, attached to text |

All three, plus a write that answered `commentUpdateState: ALL_SAVED`, is
`verified: true`. Anything less is `ok: true` with `verified: false` and a
warning naming what did not hold, because the write happened: a caller told the
run failed is a caller that writes it again. The fourth condition is why a
`verified: false` run can carry three true checks: a batch Docs accepted whose
answer could not be read leaves no state to report, and the warning there names
the lost answer rather than blaming a route. The skill reads `verified` and
decides what to tell Nail. `preview_without_suggestions` is the route that would
catch the silent direct edit, which is why it is one of the three rather than a
nicety.

**The preview check asks by words, never at the index.** `r.Start` was counted
in the view that shows pending suggestions, and the preview hides them, so every
position after one sits lower there. Looking at that index reported the second
proposal of every run, and every document already carrying somebody's pending
insertion, as a direct edit: the one warning that must never cry wolf. `Carries`
asks whether the tab still holds the quoted words anywhere, which holds up
because `FindSpan` required them to occur exactly once, so a direct edit usually
takes the only copy with it.

Usually, because a replacement that contains the quote carries it through the
edit: the write puts the replacement where the quoted words were, so "reviewed
annually" is still in the preview inside "reviewed annually by the operations
team", and the quote alone would report the route as holding on the exact
failure it exists for. So `Verify` asks a second question in that shape only,
and it is whether the preview carries the **replacement**. After an honest
suggestion it does not, because the preview hides the insertion, unless the
document already read that way before the write, which is a proposal that
duplicates the words behind it. Those two cannot be told apart from here, so the
check is false with a warning naming the ambiguity rather than one naming a
direct edit. Asking for the replacement outside that shape would be the cry-wolf
mistake again: a replacement that does not contain the quote can occur anywhere
in the document.

What no question catches is a second copy of the quote inside somebody's pending
suggested deletion, which the preview still shows. That is the price of asking
by words, and it is the cheaper of the two mistakes.

**`docx_anchored` gives no answer when two comments read the same words and
disagree about being attached.** The export carries no Drive comment id, so two
proposals in one run with the same reason are two comments with one body. Taking
the first would report one proposal on the strength of the other's comment, so
when the matches disagree the check is false with a warning naming the
ambiguity. Duplicates that agree answer correctly for both proposals, and the
check is whatever they agree on. It is the rule
`internal/docx`'s own witness follows, on the same join.

**Provenance in `proposals[]` is the permission to withdraw.** `withdraw`
requires `--md`, and refuses an id the note does not record as gdoc's own. The
front matter is the only place that memory lives, so a `propose` run without
`--md` still lands the suggestion and simply forgets it, and the entry is
recorded whether or not the read-backs *held*: a proposal reported
`verified: false` is in the document either way, and a proposal gdoc has
forgotten is one it will refuse to withdraw.

**What it cannot record, it names.** An entry needs both ids, and
`frontmatter.Block.Validate` refuses one missing either, so a change that landed
without one of them in hand cannot be written down at all. `propose.Record`
hands those results back to the caller instead of dropping them, and `record`
turns each into a warning naming the quoted words: the change is in the document
and `withdraw` will refuse it for ever. **The warning names the route the id
would have come from**, and the two routes are not the same one: the comment id
is the batch's own answer, while the suggestion id is read out of the inline
read-back afterwards. `missingID` in `cmd/gdoc/write.go` is that split. Blaming
the write for a read-back that failed sends somebody to look at Docs while the
envelope's other warning is already saying the re-read is what broke. The note is not listed
in `files_changed` when nothing was added to it. Silence there read as a
verification gap rather than as a permission thrown away.

**The note is read again just before it is written, by both writers.** The
pairing is checked before the session opens, and the run then spends seconds to
tens of seconds on the network: the probe plus a read, a write and three
read-backs per proposal for `propose`, and two whole-document reads plus a
`batchUpdate` for `withdraw`. These notes live in a synced vault, so writing the
bytes the run started with would throw away whatever landed in that window.
`freshNote` reads the file again, and refuses four things rather than writing
them: a file it cannot read again, one whose front matter no longer parses, one
whose `gdoc:` block has gone, and one that now names another document. Each is a
warning carrying the reason, and nothing is written into the note. The block is
re-parsed from the fresh bytes too, so a proposal another run recorded in that
window survives: writing the block this run read into bytes it did not would
keep the author's prose and still drop that entry. `notePath` therefore keeps
the path and the block the pairing check read, and not the bytes it read them
from: a copy held there would only be the stale bytes somebody later wrote
back.

**A withdrawal is a `rejectSuggestion` on gdoc's own id, and that is Nail's
decision of 2026-09-07.** It was a `deleteContentRange` in SUGGEST mode over the
insertion until the first live write test ran, after the revmux review had
passed. A `propose` is a replace, one suggestion id over a suggested deletion
and a suggested insertion, and that delete retracts only the insertion half:
the new words go, Docs answers `updatedSummarySuggestionIds` rather than
`deletedSuggestionIds`, and the quoted words stay suggested-deleted under the
same id. A second delete over them is a no-op, and one delete over both halves
marks the new words inserted and deleted at once. The 2026-08-29 measurement
that a delete "comes back with `deletedSuggestionIds`" was on a pure insertion.
`rejectSuggestion` takes the whole thing back in one request, in SUGGEST mode:
the document reads as it did before the proposal, the answer names the id in
`suggestionResponses[].rejectedSuggestionIds`, and the 🤖 comment survives,
still anchored by id, so the skill can reply into it. The five measurements are
in `docs/v2/DECISIONS.md` under that date.

**The permission is provenance, and the guard holds it as a per-run grant.**
The guard refuses every request kind whose name carries "suggestion", and that
rule stands: gdoc never accepts, rejects or deletes anyone else's. The one door
is `Policy.AllowReject(id)`, which `cmdWithdraw` seeds with the suggestion the
note's `proposals[]` records as gdoc's own, after `Mine` has said so and before
the session is built. The guard then carries a `rejectSuggestion` only when it
is spelled exactly, carries exactly `{"suggestionId": <that id>}` and nothing
beside it, and the id is the granted one. `acceptSuggestion` and
`deleteSuggestion` stay refused whatever id they name, a `rejectSuggestion`
naming another id is refused, and a second field beside the id is refused
because nobody here has read what it does. `TestAGrantedRejectSuggestionCarriesAndNothingElseInTheFamilyDoes`
in `internal/guard` states all of it. The grant is one id and dies with the
process. Nothing else gdoc does grants it, and widening it is Nail's decision.

**A withdrawn suggestion leaves the note only once it has provably left the
document.** It is gone only when `rejectedSuggestionIds` names it **and** a
fresh read shows no run carrying it on either side, and only then does the
entry leave the note. Either side, because a run still suggested-deleted under
the id is half a proposal still pending, and it is the half the old delete used
to leave behind: `withdraw` takes those back too, so a document the old
withdraw half-retracted is repaired by running it again. A reject names no
range, so a document with more than one tab is withdrawn from like any other.
Forgetting the entry while the suggestion is still pending would leave gdoc
refusing to withdraw its own work. The price is a reject Docs accepted whose
answer could not be read: usually no ids came back, so `verified` stays false,
the entry stays in the note, and a retry is refused by the pending check rather
than by the note. The run says so, and says to take the entry out by hand once
the suggestion is gone. Relaxing the two facts to one on that path is a
decision for Nail, not a refactor.

**"Usually" is the whole word there, and neither writer gates on it.** Valid
JSON of the wrong shape is the one failure that reaches the caller with fields
in hand, because `encoding/json` saves the first type error and keeps decoding.
What the server really did send is kept: `withdraw` keeps the
`rejectedSuggestionIds` it was given, `propose` keeps the comment id, which is
the provenance `withdraw` later needs, and `reply` keeps the reply id and reads
the thread back on it rather than saying the reply could not be looked for. So both warnings are built from what
was decoded rather than from the path being taken. A withdrawal whose two facts
both held is reported `verified: true` and its entry does leave the note, and
the warning then says the answer was lost without telling anybody to take out an
entry the same run removed.

The 🤖 comment a withdrawn proposal made stays where it is. `commentWrites`
carries `POST` and nothing else, so deleting or editing a comment is a write the
guard does not carry, and no command here needs one. The skill replies to the
comment saying the proposal was withdrawn. A milestone that needs `PATCH` or
`DELETE` adds it back beside its caller, the way `GrantInPlace` returns at M7.

**The 🤖 prefix is the only record of authorship there is.** The Docs API cannot
set an author, so everything gdoc writes is signed by whoever is logged in.
Every reply and every comment gdoc writes opens with `🤖 ` and nothing before
it. A body carrying markdown is refused
too, ported from v1's `assert_plain_text`: a Docs thread renders it literally,
so asterisks and backticks arrive as typed. That rule lives in
`internal/plaintext` and both writers ask it: a reply through `reply.Check`, and
a proposal's `why` through `Proposal.Check`, because the reason is written into a
thread as a comment. Two copies of one regular expression are two rules that
drift.

**Both writers ask that rule behind the mark, never in front of it.** The
heading arm is anchored to a line start, so a `🤖 ` sitting in front of a `# `
moves the hash off offset zero and the arm cannot fire: asked of the whole
string, a `# ` on the first line passes while the same words on the second line
are refused, which is one rule firing or not depending on where the author put
them. So `Proposal.Check` reads `why` alone, which is what it is given, and
`reply.Check` trims `Prefix` first, which is exact because the line above it has
already required the body to open with exactly that.

**The two writers own the mark differently, and a caller has to know which.**
A reply body arrives with the mark already on it, so `reply.Check` **requires**
it and refuses a body that does not open with exactly `🤖 `. A proposal's
`why` arrives without it, because `Batch` writes the comment as `Prefix + Why`,
so `Proposal.Check` **refuses** a reason that already carries it. An agent
following the reply sentence when it writes `proposals.json` has its run refused
at `readProposals`, before anything leaves the machine. The refusal is asked of
the robot alone rather than the robot and its space, because `🤖the policy`, a
robot behind a space and a robot behind a newline each land the same doubled
mark; a whitespace-only reason is refused for the same kind of reason, since the
comment would be a bare signature and every read-back would still hold over it. This is v1's `[gdoc]` rule in v2's
shape, and "Identity is never a gate" below is why the marker decides rather
than the account.

**A write whose answer could not be read is not a write that never happened.**
`internal/gapi` marks the failures raised after the server answered 2xx, a body
that is not JSON and a body over the ceiling among them, and a writer package
asks by behaviour (a `Sent() bool` method) rather than by importing that package:
naming a `Session` interface is what keeps `net/http` out of those rooms, and an
imported sentinel would bring it back through the side door. All three writers
ask it, and each keeps what the answer still carried. In the usual case nothing
decoded: `reply.Post` then warns and says to check the thread before posting
again, which is what its own doc comment already promised; `propose.Apply` runs
the read-backs and reports the proposal with the comment id unknown; and
`withdraw.Run` runs its read-back and reports the withdrawal it cannot confirm.
When fields did decode all three report them instead, and `reply.Post` looks the
reply up in the thread on an id that survived. Read the "Usually" paragraph
above: the gate is the decoded field, never the path being taken.

**What is not marked is three cases, not two.** A guard refusal never left the
machine, and a 4xx is Docs rejecting the batch whole: a caller is right to treat
both as a change that did not happen. A 5xx or a dropped connection is the
third, and gdoc cannot tell it apart: the request was written and may have been
applied. Nothing claims otherwise, in either direction, so a caller that sees a
transport failure or a 5xx reads the document before sending the same write
again. Widening the mark to cover it would be a decision, and it would make
every one of those a reported-not-raised failure.

**Nothing here decides what to write.** The body of a reply, the words of a
proposal and the reason for it arrive already written, in a file the skill
wrote: `--body-file` for a reply, `--from proposals.json` for a proposal, a list
of `{quoted, replacement, why, assignee?}`. That file is read strictly, the way
every other input here is: an unknown key is refused by name, and so is a second
list behind the first. A misspelled `quoted`, `replacement` or `why` is caught by
`Proposal.Check` because their empty values are refused, but `assignee` is
optional, so a dropped one landed a comment with nobody assigned and warned about
nothing. The skill reads the threads and
decides which ones still need an answer; the binary reports the marker,
`resolved`, `by_gdoc` and the witness, and says nothing about what any of them
means. That is M2's line, held. Read "The binary prints facts, and the skills
judge" above.

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
Nail confirmed both in the M2 review, 2026-09-07: `AllowCreateIn` stays, and a
comment whose range the Docs read did not place is a warning on the envelope,
never an error. Do not reopen either without him.

M3 settled the `AllowCreateIn` half. It has a production caller now, the
capability probe, which creates the throwaway document it asks its question on.
So the door M2 kept on the strength of M6 needing it was wanted well before M6,
and keeping it was right for a reason nobody had yet.

### Running a milestone

A milestone is a plan under `docs/plans/`, executed by ralphex and reviewed by
revmux. The plan and the `.ralphex/` configuration are committed on `main`
before the run starts, because ralphex creates the feature branch from `main`
and the reviewer would otherwise see its own scripts as new code.

Review is revmux only, Nail's decision after M2 (2026-09-07). ralphex's full
mode runs its own Claude review rounds before the external tool and has no
switch that drops only those, so a milestone runs as two commands from a
normal terminal, not from inside a Claude Code session:

```bash
ralphex --tasks-only docs/plans/<plan>.md
```

```bash
ralphex --external-only
```

The first executes the tasks, one commit each, on the feature branch. The
second runs revmux through `.ralphex/scripts/revmux-review.sh` until a clean
round or `review_patience` unchanged rounds, then one post-external
critical/major check. The plan moves to `docs/plans/completed/` when the run
finishes; if the run is cut short (the M2 run stopped on the Claude session
limit in round 6), the fixes revmux produced sit uncommitted in the working
tree and the plan move is done by hand.

### Building

| Command | Does |
|---|---|
| `make test` | `cd go && go test -race ./...` |
| `make vet` | `go vet ./...` and the `gofmt -l` check |
| `make build` | `bin/gdoc`, for this machine |
| `make dist` | the three platform binaries |

**`gdoc` on PATH is v2, and `gdoc2` is gone.** M3 repointed it:
`~/.local/bin/gdoc` links to this repo's `bin/gdoc`, so `make build` refreshes
it with no reinstall, and `install.sh` removes the `gdoc2` link it made earlier
rather than leaving one word with two meanings. `gdoc2` was Nail's temporary
name for testing v2 beside v1 (2026-09-07); the M2 note here that described it
as the permanent arrangement is superseded by this paragraph.

v1 is untouched by that. Both v1 skills call the venv binary by its full path,
`$HOME/.config/gdoc-agent/venv/bin/gdoc`, never `gdoc` on PATH, so repointing
the link breaks neither of them. What it does change is what a person typing
`gdoc` gets, and the two tools share one word with two meanings: v1 `read`
lists comments, v2 `read` prints the document text and v2 `comments` lists
comments. The install story proper, one file copied to a machine with nothing
else on it, is still M9.

`GDOC_LIVE_TEST=1` runs the opt-in end-to-end read test, in `go/internal/live`.
It then needs `GDOC_LIVE_DOC_ID=<document id>`, and there is no default: the
guard is opened with exactly the document the run names. It reads, and creates
nothing on Drive. `GDOC_LIVE_RECORD=1` additionally saves the Docs read and the
docx export into `testdata/`, which is a real document's content, so a person
redacts those before they are committed.

`GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1` adds the write test beside it. It creates
its own document in the Drive test folder, proposes into it, replies, withdraws
and trashes it, asserting every read-back on the way, and it writes only to
documents it made. Two variables rather than one, because a live read is
somebody's document and a live write is a document that did not exist a second
ago: the second is a different decision, and it is made on purpose each time.
The unattended run sets neither.

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
  and nothing may start. v2 writes into a document and this rule is unchanged:
  every write is a suggestion, `go/internal/guard` refuses a `batchUpdate`
  without `writeMode: SUGGEST`, and the read-back through
  `PREVIEW_WITHOUT_SUGGESTIONS` is there because Google has broken that promise
  once.
- Never commit anything from `~/.config/gdoc-agent/`.
- Never post markdown into a comment thread. The CLI refuses it for a reason.

## Writing style

Plain, short English. No em dashes.
