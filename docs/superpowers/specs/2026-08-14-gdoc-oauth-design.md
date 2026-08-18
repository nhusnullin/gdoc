# OAuth as a second credential

Status: design, approved in outline on 2026-08-14, section 5 rewritten and
approved on 2026-08-15. Supersedes nothing. Extends the credential section of
`2026-08-13-gdoc-ai-agent-design.md`.

## Principles

`PRINCIPLES.md` did not exist when this spec was first written. The gate is
answered here because section 5 was rewritten after it landed.

Serves: principle 3. Under OAuth the credential can reach a whole Drive, and the
guard's allowed set starts empty, so a command that does not name a document
refuses every call rather than reaching for one. Not knowing resolves to
refusing, never to acting.

Strains: principle 3, in the other direction, and this is the cost of the
design. The guard permits every method on the file the command names, including
an edit to a reviewed document, which the service account could not make. What
was a limit Google enforced becomes a limit gdoc merely respects. Section 5,
"What this gives up", states it, and the dated decision in `PRINCIPLES.md` is
rewritten rather than quietly left standing.

Also touches principle 1: two packages are added, `google-auth-oauthlib` and
`google-auth-httplib2`, both pip-installable, with reasons in section 10.

## 1. Why

Today the only credential is a service account. Every document has to be shared
with its address before the agent can see it. That is one manual step per
document, on documents Nail already owns or can already read.

OAuth removes the step. The agent authorises once as Nail and reaches every
document Nail can reach.

## 2. What OAuth actually changes

Three things change, and only the first is the point of the exercise.

### Access

No sharing step. Any document Nail can open, the agent can read.

### Scope

The scope stays `https://www.googleapis.com/auth/drive`, the same one the
service account uses. There is no narrower option. Drive's `replies.create`
accepts only `drive` and `drive.file`, and `drive.file` covers only files the
application itself created, which a document written by someone else never is.
So reading comments and posting replies on a document the tool did not create
requires the full scope.

Under a service account, full scope on a document shared as Commenter is still
Commenter. Under OAuth, full scope means everything Nail can do, on everything
Nail can reach. That is the trade.

Comments and replies work normally under OAuth. They post under Nail's own name
instead of the service account address.

### Identity

This is the part that breaks the existing code.

`gdoc/model.py` sets `Thread.by_agent` and `Reply.by_agent` from Drive's
`author.me` field, which means "written by the credential making this request".
`gdoc/filters.py:needs_action` then requires `not thread.by_agent`.

Under the service account, `me` is the service account. Under OAuth, `me` is
Nail. So every `ai:` comment Nail writes would be read as written by the agent
and skipped, and `gdoc read` would return an empty `addressed` list on every
document. OAuth cannot ship without fixing this.

**Amended 2026-08-15, after review.** The first draft dropped `by_agent` from
`needs_action` but kept it inside `has_agent_reply`, on the grounds that it
covered threads the service account answered before the marker existed. That
reasoning was wrong under OAuth, and the error was the same one the section
above describes.

A thread the service account answered has `me` **false** when Nail's token
reads it, because `me` means "the requester wrote this". So under OAuth
`by_agent` provides no legacy coverage at all. What it does provide is a false
positive on every reply Nail types himself, which makes `has_agent_reply` true
and drops that thread into `skipped` on every future run, permanently.

The fix is that `me` only identifies gdoc when the credential is gdoc.
`parse_thread(raw, me_is_agent)` takes the answer, `fetch_threads` passes it,
and the CLI derives it from `auth_mode`. Under `service_account` nothing
changes and the legacy coverage still works. Under `oauth`, `by_agent` is
always false and the marker carries idempotence alone, which is what the marker
was added for.

## 3. Decisions

| Question | Decision |
|---|---|
| Which credential by default | `auth_mode` in config. Unstated is inferred, see the 2026-08-18 amendment |
| Silent fallback when the token is missing | No. Fail with an error naming both fixes |
| What narrows the credential | A guard in the HTTP transport, on file ids |
| What the guard governs | Which files, never which methods |
| Can the guard be turned off | No. There is no config key for it |
| Who the agent acts for | Anyone. The marker decides, never the author |
| How the agent recognises its own replies | A `[gdoc]` line it writes, not identity |
| Unmarked comments | Ignored, unless Nail asks for all-comments mode |

The third and fourth rows were decided differently on 2026-08-14 and amended on
2026-08-15. Section 5 records both the rule and the reason it changed.

## 4. Credentials

### Files

Both live in `~/.config/gdoc-agent/`, neither in this repo.

| File | Written by | Holds |
|---|---|---|
| `oauth-client.json` | Nail, once, from Google Cloud | the Desktop OAuth client id and secret |
| `oauth-token.json` | `gdoc auth login` | the access and refresh tokens, mode `0600` |

`sa-key.json` stays exactly where it is. Nothing about the service account path
changes.

### `gdoc/oauth.py`

A new module, so `auth.py` stays a thin chooser and neither credential source
grows into the other.

```
login(client_path, token_path) -> Credentials
load(token_path) -> Credentials
logout(token_path) -> bool
account(credentials) -> dict
```

`login` runs `InstalledAppFlow.from_client_secrets_file` and
`run_local_server(port=0, access_type="offline", prompt="consent")`. Both
keyword arguments are passed explicitly rather than relied on, because without
them Google may return no refresh token and the next run would silently need a
browser again. The result is written to `token_path` with mode `0600`, created
before the content is written, never after.

`load` reads the token with `Credentials.from_authorized_user_file`. Three
outcomes:

- Valid: return it.
- Expired with a refresh token: refresh, rewrite the file, return it.
- Expired with no refresh token, or scopes that do not cover `SCOPES`: raise,
  naming `gdoc auth login` as the fix.

A stored token whose scopes are narrower than `SCOPES` is refused at load time,
not left to fail later inside an API call. A 403 from Drive halfway through a
review is a much worse error message than one at startup.

`account` calls Drive `about.get(fields="user(displayName,emailAddress)")`. The
`drive` scope covers it, so no extra scope is requested just to know who is
signed in.

`logout` deletes the token file and returns whether a file was there. It does
not revoke the grant with Google. Revoking is a browser action in the Google
account settings, and `auth status` prints where to do it.

### `gdoc/auth.py`

`load_credentials` stays the single chokepoint. It gains a mode:

```
load_credentials(mode=None, ...)  # mode defaults to config.auth_mode
```

`drive_service()` keeps its signature. `SCOPES` is unchanged and shared by both
modes.

The missing-credential errors name the file and the fix:

- oauth, no token: the path, then `run: gdoc auth login`, then a note that
  `gdoc auth use service_account` restores the old behaviour.
- oauth, no client file: the path, and that it is a Desktop OAuth client
  downloaded from Google Cloud. Points at section 9 of this spec.
- service_account, no key: unchanged from today.

## 5. The guard

Amended 2026-08-15. The first draft of this section guarded verbs: any read was
allowed, and only four writes. That guarded the wrong axis, and this section now
says so. The reasoning is in "Why files and not verbs" below.

### What it is for

Under a service account the credential reached exactly the documents that had
been shared with it, and on those it was a Commenter. Two limits, both Google's:
a small reachable set, and no writing inside it.

OAuth removes both at once. The token is Nail, the scope is full `drive`, and
the reachable set is every file Nail owns or has been given. Nothing external
narrows that, so the tool has to narrow it itself.

The guard narrows the **set**, not the verbs. It answers one question: may this
request touch this file? On a file it may touch, it has no opinion about the
method.

### Why files and not verbs

The danger OAuth introduces is not that gdoc might learn to `PATCH`. It is that
the credential can reach a whole Drive. A verb allowlist does nothing about
that: under the first draft, gdoc could have read every document Nail owns, and
every `GET` would have passed.

The verb rule also modelled the wrong thing. It was written as if it were
Commenter in Python, but Commenter cannot create files and the tool must, so
the allowlist already had `files.create` in it. What it actually encoded was
gdoc's own list of calls as of 2026-08-14, which means it has to be edited every
time the tool learns a new one. It was already out of date when it was written:
`generate` gained a second pass on 2026-08-15 that trashes its measuring copy
with `files.update`, and the verb allowlist refuses it.

A file-id rule needs no edit when a call is added, and it is aimed at the risk
that OAuth actually creates.

### Why in the transport

The guarantee is meant to hold whatever the call sites do. A check at the call
sites would only prove the tool remembered. A check in the HTTP transport sees
every request the client makes, including ones added later, and including any
other Google API built on the same transport, which is how the Docs API is
covered without being named.

### The allowed set

```
allowed = { the file id the command was given }
        ∪ { file ids returned by creates the guard itself carried }
```

The first comes from the CLI: `cmd_read` and `cmd_reply` already call
`extract_doc_id(args.url)` before building the client, so the id is passed to
`drive_service`. Commands with no input document, `generate` among them, start
with an empty set.

The second is how `generate` works at all. It creates a measuring copy, exports
it to PDF, creates the published copy, then trashes the measuring one. Three of
those four calls address a file that did not exist when the client was built.

An empty set refuses everything. A command that forgets to name its document
fails closed, which is principle 3.

### The rule

Method is never consulted, except to tell a create from a listing.

| Request | Verdict |
|---|---|
| path carries a file id, and the id is allowed | carried |
| path carries a file id, and the id is not allowed | refused |
| `POST /drive/v3/files`, `POST /upload/drive/v3/files` | carried, and the new id is learned |
| `GET /drive/v3/files` | refused. This is `files.list` |
| any path with a `.` or `..` segment | refused. See "Matching" |
| `POST`, `PATCH`, `DELETE` on `.../permissions` | refused, on every file, allowed or not |
| `POST /batch/...` | refused. The ids live in the body, not the path |
| `GET /discovery/...`, `GET /drive/v3/about` | carried. No file is involved |
| anything else | refused |

Two of those rows carry weight beyond their size.

**`files.list` is refused.** This is most of "gdoc cannot see files it was not
given". gdoc never searches Drive today, so the refusal costs nothing, and a
future command that wants to search has to come back to this section.

**Changing `permissions` is refused everywhere.** Granting other people access
is a different kind of authority from changing a document, and the tool has no
reason to hold it. Reading the list is not refused: it is a read of the named
file like any other, and under a service account Google answers it with a 403
regardless, which `tests/test_access_integration.py` records.

### Matching

The host is matched, not only the path, so a path shape on an unexpected host
does not pass. The query string is ignored, so `fields` and `uploadType` cannot
change a verdict. The method is upper-cased before use.

Two things found in review on 2026-08-15, both of which let the guard read a
different request than the one the server acts on:

**Dot segments are refused, not resolved.** Google normalises `..` and answers
302 to the normalised path, and httplib2 follows that redirect inside the
transport the guard wraps. Verified live: `/drive/v3/files/{a}/../{b}` and its
`%2e%2e` form both redirect to `/drive/v3/files/{b}`. So the file the guard
reads out of the path is not the file the server acts on. Any `.` or `..`
segment, encoded or not, is refused. gdoc never builds one, so refusing costs
nothing, and resolving would mean trusting that our normalisation matches
Google's exactly.

The practical exploit was blunted: the redirect loses the Authorization header,
so the follow-up returned 403 rather than the document. That is httplib2's
behaviour rather than a guarantee, and the promise "refuses every request
addressing anything else" was false either way.

**The method override is honoured.** `googleapiclient` rewrites a GET whose URI
exceeds `MAX_URI_LENGTH` as a POST carrying `x-http-method-override: GET`. The
server acts on the override, so the guard must too. Without this, an over-long
`files.list` arrives as `POST /drive/v3/files` and reads as a create.

File ids are read from three path shapes:

```
/drive/v3/files/{id}...                          Drive
/upload/drive/v3/files/{id}...                   Drive, media
/v1/documents/{id}[:method]                      Docs, id stops at the colon
```

The Docs API's `documents/{id}:batchUpdate` is the write that matters most and
it is not a Drive URL, so its shape is matched explicitly.

### Learning a created id

After a create the guard carried returns a 2xx, the guard parses the response
body and adds `id` to the allowed set. It is wrapped: a body that is not JSON,
or has no `id`, teaches it nothing and never raises. `MediaFileUpload` is built
with `resumable=False`, so a create is one request and one response, with no
multi-step upload to follow.

The alternative was for the call site to register the id after each create. That
was rejected. It puts the guarantee back into discipline, and a create added
later that forgets the line fails at runtime with no test noticing.

### Refusal

`PermissionError`, which is an `OSError` subclass, so `cli.py:main` already
catches it and prints the normal JSON error envelope. No change to the CLI's
error handling.

The message names the method, the path, and the file id, and says gdoc only
touches the document it was given.

### When it is installed

Always, in both credential modes. Under the service account it is nearly
redundant, but one code path is worth more than a saved wrapper, and it lets the
guard's unit tests run with no credentials at all.

There is no way to turn it off. The first draft had `allow_document_edits: true`
in config, and that key is dropped: it was named for a lock this design no
longer has, and its only real effect would have been to disable the safety net
so that a temporary file could be binned. A config key whose one purpose is to
switch off the guard gets set once and never unset.

`tests/test_guard_is_installed.py` asserts that `gdoc/auth.py` is the only
module calling `build()`, so no future module can construct an unguarded client
quietly. It is written in the style of `tests/test_no_external_programs.py`, as
an allowlist rather than a ban.

### What this gives up

Under the service account, Google refused an edit to a reviewed document. After
this, nothing refuses it. The guard confines the damage to the one file the
command was given, and gdoc has no code that edits a document, but the promise
moves from "it cannot" to "it does not".

Two documents change to say so, in section 10: the README's cannot-edit
paragraph, and the dated decision in `PRINCIPLES.md` that reads "the credential
is Commenter-only".

The threat model moves with it. A comment inside a reviewed document that talked
the agent into editing that document would now succeed. Pointing the skill at a
document was already the trust decision, recorded in `PRINCIPLES.md` on
2026-08-13; this design makes that decision carry more.

### One thing the guard cannot offer

There is no middle setting where the agent writes only proposals a human accepts
or rejects. The Docs API can read suggestions, through `suggestionsViewMode` and
the `suggested*` fields, but it cannot create, accept or reject them. A
suggestion is a byproduct of saving in the editor's suggesting mode, and there
is no request type for it and no flag that puts a `batchUpdate` into it. Google
tracks the gap as issue 287903901, unresolved.

The one indirect route, uploading a `.docx` carrying Word tracked changes and
letting the conversion turn them into suggestions, needs a full-content write
over the existing file, which would orphan every comment anchor the tool depends
on.

So the guard's rule is binary because Google's API is.

## 6. Identity, and the `[gdoc]` marker

### Two markers, opposite directions

The document will now carry two kinds of marker, and they cannot be confused.

| | `ai:` `ai?` `ai!` | `[gdoc]` |
|---|---|---|
| Sits on | a comment | a reply |
| At the | start | end |
| Written by | a person | the tool |
| Means | this is for the agent | the agent wrote this |

One is input, one is output. A comment is never a reply, so no rule ever has to
choose between them.

`[gdoc]` is a label, not a signature. A person can type it and make the tool
treat a thread as answered. The result is a skipped thread, which is the safe
failure, so it is not worth defending against.

### The marker

Every reply `gdoc reply` posts ends with a blank line and `[gdoc]` on its own
last line.

```
Suggested replacement: ...

Sources: domains/regulatory/cbc-emi.md

[gdoc]
```

Visible on purpose. Replies now post under Nail's own name, so a reader of the
thread needs some way to tell which replies a person wrote and which the agent
wrote. An invisible marker would hide exactly the thing that got more confusing.
It also survives copy and paste, which invisible state does not.

`assert_plain_text` still runs on the body Nail's agent wrote, before the marker
is appended. The marker itself contains no character the markdown check rejects.

### The rules that use it

`Reply` gains `by_marker`, true when the content's last non-empty line is
`[gdoc]`.

`Thread.has_agent_reply` becomes `any(r.by_marker or r.by_agent for r in
replies)`.

`by_agent` is kept in that test on purpose. Threads the service account answered
before this change carry no marker, and dropping `me` would repost on every one
of them. Keeping it costs one thing: under OAuth, a thread Nail replied to by
hand counts as answered and is skipped. That is the safe direction to fail, and
all-comments mode shows the thread anyway.

`needs_action` drops `not thread.by_agent` entirely. A marked comment is work no
matter who wrote it, including Nail, which is what OAuth requires and what the
project already decided in commit 989e744. The author name still travels in the
payload, so an unexpected one is visible in the report.

### One migration risk, stated rather than solved

Switching a document from the service account to OAuth changes who `me` is. The
service account's old replies stop being `me` and carry no marker, so the first
OAuth run on a document the service account already reviewed can list threads
that were in fact answered.

Not worth code. `gdoc read` will now return each thread's existing replies, and
step 2 of `gdoc-review` already stops and waits for Nail before posting
anything. He sees the existing reply in the list and says no. The skill gets one
line telling it to expect this on the first OAuth run of an older document.

## 7. All-comments mode

### The problem

Today the marker is the only way a comment becomes work. Sometimes Nail wants to
go through a document's comments with the agent, including ones nobody marked.

### The shape

`gdoc read <url> --all` widens the candidate set. It does not change what
happens next, because what happens next is a conversation.

| | default | `--all` |
|---|---|---|
| `addressed` | unresolved, marked, not yet answered | every unresolved thread |
| `skipped` | everything else | resolved threads only |

`--all` filters on nothing but `resolved`. It does not drop threads that already
have an answer, which is deliberate: `has_agent_reply` counts a `me` reply, and
under OAuth `me` is Nail, so dropping them would hide every thread Nail replied
to by hand. All-comments mode exists to show everything and let Nail choose. It
must not quietly apply the one rule whose meaning changes with the credential.

The payload gains `"mode": "marked"` or `"all"` at the top level, and per
thread:

- `"marked": true|false`, so the skill can show which carry `ai:`
- `"answered": true|false`, the `has_agent_reply` value, so the skill can list
  answered threads apart from the rest instead of hiding them
- `"replies": [{"author": ..., "content": ..., "by_gdoc": ...}]`, so the skill
  can show what the existing answer actually says

### What the skill does with it

`gdoc-review/SKILL.md` step 2 splits into two variants.

Default stays as it is: show the list, ask once, proceed.

All-comments mode never batch-approves. The agent prints the numbered list with
author, quote and content, threads already carrying an answer listed last and
labelled, and Nail picks the ones to work on. For each picked comment the agent
drafts the reply, shows it in the terminal, waits for Nail, and only then posts.
Unmarked comments carry no `forced_kind`, so the agent classifies local against
global itself and says which it chose.

Picking an already-answered thread is allowed, and is the one case where the
agent replies twice to the same comment. It says so before posting.

Two lines in the `Never` list change, and both change the same way, by naming
the one exception rather than dropping the rule:

- "Never act on a comment without the marker" becomes "Never act on an unmarked
  comment unless Nail asked for all-comments mode and picked that comment".
- "Never reply twice to the same comment" becomes "Never reply twice to the same
  comment unless Nail picked an answered thread in all-comments mode, and say so
  before posting".

## 8. What reaches gdoc-apply

`gdoc-apply` never reads comments. It reads `pending.md`. A comment reaches it
only through `gdoc capture`, run by `gdoc-review`, which writes the comment text
into the queue quoted. Neither marker survives that trip, so nothing in
`gdoc-apply` has to know either one exists.

The rule stays what commit 989e744 made it: feedback is acted on whoever left
it. The author is a label in the report, never a gate.

### The order is review, then apply, and nothing here enforces it

Because `gdoc-apply` reads only the queue, a comment left after the review that
filled that queue is invisible to it, and it is stranded on a version the next
generate supersedes. This is a real gap and OAuth widens it, because a colleague
can now leave a marked comment on a document nobody shared with the agent.

It is not fixed here. Closing it properly means `gdoc-apply` stops being a queue
consumer, which is a change to what the skill is for, not a check bolted onto its
first step. That is designed separately in
`2026-08-14-gdoc-apply-drains-design.md` and lands after this branch.

Nothing in this branch depends on that design, and nothing in it should
anticipate it.

### One thing that must change

`gdoc/pending.py:_ITEM` writes the literal line `Nail asked:` and records no
author at all. That was true when only Nail's own comments could be captured. It
is wrong twice over now. The marker already decides irrespective of author, and
all-comments mode will start capturing comments nobody marked.

The item gains an author field, and the hardcoded line goes:

```
## Item {number}

- Captured: {today}
- Comment id: {comment_id}
- Author: {author}
- Anchored to: {anchor}
- Marked: ai! | no marker

The comment:

> ...
```

`Marked:` records whether the comment carried a marker, so a later session can
tell an explicit instruction from something picked out of the document during an
all-comments pass. Both are applied the same way. The distinction is for Nail's
eyes, not for a rule.

Items already in a `pending.md` have no `Author:` line. `gdoc-apply` treats a
missing author as unknown and says so, the same way it already falls back when
an older queue has no `Source:` line.

`skills/gdoc-apply/SKILL.md` step 2 gains one line: name the author when showing
each item, so Nail knows whose feedback he is approving.

## 9. Setup, once

In the same Google Cloud project as the service account.

1. Drive API stays enabled. It is still the only API the package calls. The Docs
   API is not required: the one `docs v1` call in the repo is a probe inside
   `tests/test_access_integration.py` that asserts text insertion is refused.
2. OAuth consent screen, User type **Internal**. altery.com is a Workspace org,
   so Internal is available. In External plus Testing, Google expires refresh
   tokens after seven days and `gdoc auth login` would be a weekly chore.
   Internal has no such expiry.
3. Credentials, Create credentials, OAuth client ID, Application type **Desktop
   app**. Download the JSON to `~/.config/gdoc-agent/oauth-client.json`.
4. `gdoc auth login`. A browser opens, approve, the token is written.
5. `gdoc auth status` to confirm the account.

## 10. Surfaces changed

### CLI

```
gdoc auth login     run the browser flow, write the token
gdoc auth status    mode, account, token expiry, file paths, how to revoke
gdoc auth logout    delete the local token
gdoc read <url> --all
```

`auth status` prints, and never fails, when the credential is missing. Its job
is to say what is wrong.

### Config

```json
{
  "output_folder_id": "0AFolderId",
  "auth_mode": "oauth"
}
```

`auth_mode` accepts only `"oauth"` or `"service_account"`. Any other value is an
error at load, naming both accepted values. It is the only key this design adds.

A config file without it loads unchanged, the same way an older file with
`display_name` still loads today. Which credential that means is the subject of
the amendment below.

### Dependencies

`google-auth-oauthlib` is added to `pyproject.toml`. It is the only genuinely
new package: checked in the venv on 2026-08-14, it is the one import in this
design that is not already there.

`google-auth-httplib2` is added too, even though version 0.4.1 is already
installed as a transitive dependency of `google-api-python-client`. The guard
imports it directly now, so it is a direct dependency and should be declared as
one. A transitive dependency that a later release drops is a build that breaks
for no visible reason.

### install.sh

The credential check reads `auth_mode` from `config.json` using the venv python,
defaulting to `oauth`, and warns for the file that mode needs. Under `oauth`
with a client file but no token it prints the `gdoc auth login` line rather than
a warning, because that is the next step and not a fault.

### Documentation

- `README.md`: the credential section describes both modes. The sentence "It
  cannot edit a document, and that limit is the design" is replaced. Under OAuth
  the limit that holds is the reachable set, not the verb, and the paragraph has
  to say so rather than keep a promise the guard no longer makes.
- `PRINCIPLES.md`: the dated decision "The credential is Commenter-only" is
  retired, with the reason, and replaced by one about the reachable set. This is
  the only principles change; the three principles themselves are untouched, and
  principle 3 is what makes the empty allowed set refuse rather than permit.
- `CLAUDE.md`: the `Never` list entry "never edit a reviewed Google Doc" loses
  its "the credential cannot" clause, which stops being true under OAuth, and
  gains what does hold: the guard confines every call to the file the command
  was given.
- `skills/gdoc-review/SKILL.md`: step 2 split, the all-comments variant, the
  first-OAuth-run note, the `Never` line, and the step 2 warning text which
  currently says "under the service account address".
- `skills/gdoc-apply/SKILL.md`: the author line when showing each item in step 2,
  and any wording that assumes the service account.

## 11. Out of scope

- **Editing a reviewed document in place.** The guard now permits it, but no
  code does it. A command that edits a document directly, and the question of
  what that does to `baseline.md` and the regenerate loop, is a separate design.
  The same is true of republishing into the same file id instead of creating a
  new document each version, which the guard would also now allow.
- **More than one Google account.** One token file, one account.
- **Encrypting the token at rest.** File mode `0600`, the same protection
  `sa-key.json` gets today.
- **Revoking the grant from the CLI.** `auth logout` is local. `auth status`
  prints the Google account page URL.

## 12. Testing

TDD. Every test below is written before the code that satisfies it. No test
reaches the network except the integration file, which already skips without
credentials.

**`tests/test_oauth.py`** (new): a valid token loads; an expired token with a
refresh token refreshes and the file is rewritten; an expired token with no
refresh token raises and names `gdoc auth login`; a token whose scopes do not
cover `SCOPES` raises at load; a missing client file names the path; the token
file is written `0600`; `logout` removes the file and reports whether one was
there.

**`tests/test_guard.py`** (new): every method on the allowed id is carried, `GET`
through `DELETE`, including `documents/{id}:batchUpdate`; every one of those is
refused on a second, unnamed id, `GET` included; `files.list` is refused;
`writing a permission is refused on the allowed id while reading the list is
not; a batch POST is refused; discovery
and `about.get` are carried; a create is carried with an empty allowed set and
its returned id is learned, so the next call on it passes; a create whose
response is not JSON teaches nothing and does not raise; an empty allowed set
refuses a read; the query string changes no verdict; the host is matched; the id
is read correctly from all three path shapes, colon suffix included; the refusal
is a `PermissionError` naming the method, the path and the file id; the inner
transport is never called on a refusal.

**`tests/test_guard_is_installed.py`** (new): `gdoc/auth.py` is the only module
under `gdoc/` that calls `build()`. An allowlist in the style of
`tests/test_no_external_programs.py`, so it fails both if the call spreads and
if it moves.

**`tests/test_config.py`**: a config without `auth_mode` still loads, and reads
as unstated rather than as oauth; an unknown `auth_mode` value raises and names
both accepted values; `write_auth_mode` carries every other key over, refuses a
file it could not parse, and writes mode 0600.

**`tests/test_reply.py`**: the marker is appended on its own last line; the
markdown check runs on the body before the marker is added; a body that already
ends with the marker does not get a second one.

**`tests/test_model.py`**: `by_marker` is true for a reply ending in `[gdoc]`,
false for one merely mentioning it mid-sentence; `has_agent_reply` is true from
the marker alone and true from `me` alone.

**`tests/test_filters.py`**: the regression this whole change turns on, a marked
comment whose author is `me` is addressed; `--all` includes unmarked unresolved
threads; `--all` excludes resolved threads; `--all` includes a thread that
already carries a `[gdoc]` reply and flags it `answered`.

**`tests/test_pending.py`**: a captured item records the comment author; it
records whether the comment carried a marker; an unmarked comment captured in
all-comments mode reads back correctly; the string `Nail asked:` appears nowhere
in a written queue; capturing the same comment twice still raises.

**`tests/test_cli.py`**: `read --all` sets `mode` to `all` and marks each thread;
the default payload sets `mode` to `marked`; the payload carries each thread's
replies; `capture` passes the thread's author and marked state through to
`append_item`; the three `auth` subcommands parse and dispatch.

**`tests/test_access_integration.py`**: splits by `auth_mode`. Under
`service_account` every existing assertion stands unchanged, because Google is
still the one refusing. Under `oauth` the assertion changes shape: an edit to the
named document now reaches Google, so what is asserted is the boundary instead.
A read of a second, unnamed document raises `PermissionError` before any request
leaves the process, and `files.list` does the same.

Coverage stays at or above 80%.

## 13. Order of work

1. The marker and the identity fix. It is a correctness change that stands on
   its own and is safe under the service account today.
2. The `auth_mode` config key, with its default but nothing reading it yet.
3. `gdoc/guard.py` and its tests, against a fake transport, no credentials.
4. `gdoc/oauth.py`, `auth.py` dispatch, the `auth` subcommands.
5. The captured-item format: author, marked, and the end of `Nail asked:`.
6. `--all` and the payload changes.
7. Wire the guard into `drive_service`, and thread the doc id from the CLI.
8. Integration suite split.
9. install.sh, README, PRINCIPLES.md, CLAUDE.md, both skills.

Steps 1 to 3 and step 5 need no Google Cloud setup, so section 9 can happen in
parallel with them.

---

## Amendment, 2026-08-18: an unstated mode is inferred, and login writes it down

Shipped with the branch. Three defects in the design above, all in the same place.

**Defaulting `auth_mode` to oauth breaks every existing install.** A config
written before this design cannot mention the key, so "defaults to oauth" meant
that upgrading flipped a working service_account setup to a credential with no
token, and every command failed. `Config.auth_mode` is now `None` when the file
does not say, and `gdoc.auth.resolve_auth_mode` decides: a stated mode wins
outright, a token means oauth, a key with no token means service_account, and
neither means oauth, which is the new install this design was written for.

The resolver never reads the config, so "stated as oauth" and "stated nothing"
stay distinguishable. It takes both paths as arguments, which is also what lets
the suite stop depending on which credentials the developer has installed.

**A login did not take effect.** `gdoc auth login` ran the browser flow and wrote
the token, but left `auth_mode` alone, so a config saying service_account kept
using the service account and the login reported success while changing nothing.
It now writes `auth_mode: oauth` through `config.write_auth_mode`, after the flow
returns and never before, so a failed login cannot break a working setup. A config
that cannot be written is a warning beside the account, not a failure, because the
token is already on disk.

**There was no supported way back.** The only route to service_account was
editing JSON in a directory the tool otherwise owns. `gdoc auth use <mode>` is
that write in either direction, and warns when the credential it switched to is
not installed yet. `gdoc auth logout` leaves the mode alone, because logging out
to sign in as another account is the common case, but it now says which commands
will fail and what to run.

`gdoc auth status` gained `auth_mode_source`, either `config` or `inferred`, and
reports a config it could not read rather than falling back quietly.

`install.sh` no longer carries its own copy of the rule. It asks the package.

**What a review pass caught afterwards**, all fixed with tests:

- The mode was written after the account lookup, a network call, so a lookup
  failure left the person signed in with the old credential still configured.
  It is written first now.
- `auth login --token <elsewhere>` claimed the mode anyway, leaving a config
  saying oauth and no token where every other command looks. It now skips the
  write and says why.
- `write_auth_mode` truncated the target before writing, so a failed write
  emptied the config and the empty file then blocked recovery. It writes to a
  temporary file and renames.
- An unrecognised stated mode, a hyphen typo for instance, fell through to
  inference and resolved to oauth on a machine holding a token. Both the resolver
  and `auth._config` now refuse rather than infer past it.
- `auth status` could raise on a config holding a JSON list, while SKILL.md said
  it never fails. `load_config` refuses a non-object by name, and status reports
  `auth_mode: null` with source `unknown`.
- `auth logout` silently repointed the credential when the config was unstated
  and a key was present. It now reports the change and how to settle it.
