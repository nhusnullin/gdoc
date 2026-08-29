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
| `.gdoc/<slug>/` | `pending.md`, `baseline.md`, generated `out/*.docx`. **Not in this repo:** it sits beside the source markdown being reviewed |

Secrets and the venv live in `~/.config/gdoc-agent/`, never in this repo.

## v2 lives at `go/`, and v1 is untouched

`gdoc/` is v1, the Python package every other section here describes. `go/` is
v2, one static binary on the Go standard library. The two trees do not import
each other, and nothing in the Go work has changed a line under `gdoc/`.

| Path | Holds |
|---|---|
| `go/cmd/gdoc/` | `main.go`. Arguments in, one JSON object out, exit |
| `go/internal/emit/` | the output envelope every command prints through |
| `go/internal/guard/` | the network policy, and the only place a client is built |
| `go/internal/auth/` | the token file, its refresh, and the login flow |
| `go/internal/auth/loopback/` | the one-shot localhost listener the browser redirect lands on |
| `go/internal/config/` | where the per-user files live, per platform |
| `go/boundary/` | the two allowlist tests that keep the wire in one room |
| `bin/` | what `make build` and `make dist` write. Not in git |

`docs/v2/SPEC.md` is the agreed design and `docs/v2/PLAN.md` the milestone
order. Milestone 1 is done: the binary exists, prints the envelope, owns the
network, and can log in and report its OAuth state.

### The two commands, and what reaches stdout

`gdoc auth status` reports `auth_mode`, `token_path`, `client_source` and
`token_present`, plus `expired` and `scopes` when a token is there. Being signed
out is an answer, so it comes back as `ok: true` with `token_present: false`
rather than as a failure.

Three things about those fields:

- `auth_mode` is the constant `"oauth"`. v2 has no service account and never
  reads v1's `config.json`, so on a machine set to `auth_mode: service_account`
  the two tools disagree on purpose. The resolver documented further down is
  v1's alone.
- `client_source` is always `"bundled"`. v1 lets `oauth-client.json` in the
  config dir override the client; v2's login does not read that file yet, so
  when it exists status adds `client_file_ignored: true` and a warning rather
  than claiming an override that is not wired up.
- A token file that exists and cannot be read is a **failure**, not
  `token_present: false`. It comes back `ok: false` with the path still in
  `data`. Reporting it as signed out is how somebody re-runs `auth login`,
  overwrites the file, and never learns what was wrong with it. Only an absent
  file means signed out.

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

A v2 `Save` carries `universe_domain` and `account` through untouched. They are
google-auth's fields, v2 uses neither, and dropping them would quietly rewrite a
file both tools share.

### The guard owns the wire, and it exists before any client

`guard.NewClient` is the only place an `*http.Client` is made, and it is made
from a `*Policy`. So the first request in the program's history has already been
judged. v1 fitted a guard around a client that already existed, which is why v1
needs a test proving `build()` is called in one module only.

Write levels live in the policy, never at the call site. `LevelSuggest` is what
a handed-in id gets: read, comment, suggest, and never a direct edit.
`LevelFull` is what a create returned, or what `GrantInPlace` raises a handed-in
id to for one process. A call site cannot widen its own reach by phrasing a
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

A create whose response carries no readable id is recorded on the policy and
readable through `Policy.Warnings()`. Silence there turns into "file was not
given to this command" on the next request, which names the wrong problem.

`Policy` is mutex-guarded. `NewClient` hands out an `*http.Client`, which Go
documents as safe for concurrent use, and `Learn` writes the set from inside
`RoundTrip` while `Judge` reads it. `make test` runs `-race`; keep it there.

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

- The **import allowlist** says who may name the type: `internal/guard`,
  `internal/auth` and `internal/auth/loopback`. `internal/auth` is on it because
  `Refresh` and `Login` take the guard's client as a parameter.
- The **builder allowlist** says who may construct an outbound client or reach a
  package-level dialer such as `http.Get`. That is `internal/guard` alone.
  Serving is not building: `internal/auth/loopback` runs an `http.Server`, which
  answers a request somebody else made, so it stays out of this set. A canary
  test states that rather than leaving it to luck.

The builder scanner resolves the import name per file and knows four ways to
make a wire: a composite literal, `new(http.Client)`, a zero-value `var c
http.Client`, and the dialers. That is not thoroughness for its own sake. A
scanner that assumes the name is always `http` and looks only for composite
literals is walked around by `import nh "net/http"`, by `import . "net/http"`,
by `new(...)` and by a zero-value declaration, and every one of those builds a
wire outside the guard while passing both checks. The canary carries a case for
each, and each case was watched failing against the older scanner.

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
`tests/test_no_external_programs.py` enforces v1's. The same file also holds
"standard library only" to the tree: it fails on a `require` block in `go.mod`
and on a `go.sum` existing at all.

### Building

| Command | Does |
|---|---|
| `make test` | `cd go && go test -race ./...` |
| `make vet` | `go vet ./...` and the `gofmt -l` check |
| `make build` | `bin/gdoc`, for this machine |
| `make dist` | the three platform binaries |

`make dist` builds darwin/arm64, darwin/amd64 and windows/amd64 with
`CGO_ENABLED=0`, so each one is static and the binary is the whole dependency.
There is no linux target. Cross-building proves the binaries link, not that they
run, so the real Windows smoke test belongs to M9.

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
