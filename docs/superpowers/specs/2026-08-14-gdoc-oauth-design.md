# OAuth as a second credential

Status: design, approved in outline on 2026-08-14. Supersedes nothing. Extends
the credential section of `2026-08-13-gdoc-ai-agent-design.md`.

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

## 3. Decisions

| Question | Decision |
|---|---|
| Which credential by default | `auth_mode` in config, defaulting to `oauth` |
| Silent fallback when the token is missing | No. Fail with an error naming both fixes |
| What replaces the cannot-edit guarantee | A write guard in the HTTP transport |
| Can the guard be turned off | Yes, `allow_document_edits: true` in config |
| Who the agent acts for | Anyone. The marker decides, never the author |
| How the agent recognises its own replies | A `[gdoc]` line it writes, not identity |
| Unmarked comments | Ignored, unless Nail asks for all-comments mode |

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
  `"auth_mode": "service_account"` in config restores the old behaviour.
- oauth, no client file: the path, and that it is a Desktop OAuth client
  downloaded from Google Cloud. Points at section 9 of this spec.
- service_account, no key: unchanged from today.

## 5. The write guard

### Why in the transport

The guarantee the README makes is that the agent *cannot* edit a reviewed
document, not that it chooses not to. A guard at the call sites would only prove
the second. A guard in the HTTP transport sees every request the client makes,
including ones added later and including any other Google API built on the same
transport.

### How

`build()` is given an explicit `http=` instead of `credentials=`. The two cannot
be passed together. The value is `google_auth_httplib2.AuthorizedHttp` wrapped in
`gdoc/guard.py:GuardedHttp`, which mirrors that class's own signature, verified
against the installed 0.4.1:

```
request(uri, method="GET", body=None, headers=None,
        redirections=5, connection_type=None, **kwargs)
```

It checks, then delegates unchanged. Every other attribute is proxied through to
the wrapped object, because `googleapiclient` reaches for more than `request`
during media upload.

The rule is an allowlist on method and path, with the query string ignored:

| Allowed | |
|---|---|
| any `GET` or `HEAD` | reads, `files.export`, `comments.list`, discovery |
| `POST /drive/v3/files` | create a new document |
| `POST /upload/drive/v3/files` | the same create, with the docx body |
| `POST /drive/v3/files/{id}/comments` | leave a comment |
| `POST /drive/v3/files/{id}/comments/{cid}/replies` | reply in a thread |

Everything else is refused: every `PATCH`, `PUT` and `DELETE`, and every other
`POST`. That covers `files.update`, `comments.delete`, `replies.delete`,
`permissions.create`, batch requests, and the Docs API's
`documents/{id}:batchUpdate`, without naming any of them.

The refusal raises `PermissionError`. It is an `OSError` subclass, so
`cli.py:main` already catches it and prints the normal JSON error envelope. No
change to the CLI's error handling.

The message names the method, the path, and `allow_document_edits`.

### When it is installed

In both modes. Under the service account it is redundant, because Google refuses
anyway, but one code path is worth more than a saved wrapper, and it lets the
guard's unit tests run with no credentials at all.

`allow_document_edits: true` in config means the guard is not installed.

That flag only removes the block. Nothing in gdoc edits a document in place; it
regenerates a new version from the paired markdown. Making the agent edit a
document directly is a separate feature and is out of scope here. See section 11.

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

### The order is review, then apply, and nothing enforces it

Because `gdoc-apply` reads only the queue, a comment left after the review that
filled that queue is invisible to it. This sequence loses work silently:

1. `/gdoc-review` captures two global items.
2. Nail reads the document again and leaves three more `ai!` comments.
3. `/gdoc-apply` applies the two, generates v2, and reports success.

The three new comments are stranded on v1, which is now superseded, and nothing
said so. Today the pipeline only works if the order is kept by hand.

The check is cheap, because the data already exists. `gdoc read` returns
`addressed`, which already excludes every thread gdoc has answered, including
the refusal replies `capture` posts. So at apply time a non-empty `addressed`
means precisely "comments nobody has reviewed yet".

`gdoc-apply` step 1 gains it. Before touching any item, run `gdoc read` on the
document the queue names. If `addressed` is empty, carry on without comment. If
it is not, stop and say so:

```
3 comments on this document have not been reviewed yet:
  para 4 (Nail)           ai! renumber the annex
  para 9 (William Mejia)  ai: is this the right term
  para 12 (Nail)          ai! drop the pilot section

Applying now would generate v2 without them, and they would stay on v1.
Run /gdoc-review first, or say "apply anyway" to work through the queue alone.
```

One refinement. `capture` writes the queue, and the refusal reply that marks the
thread answered is a separate `gdoc reply` call in the skill. If that reply ever
failed, an already-captured comment would still be in `addressed`. So the check
drops any id that already appears as a `Comment id:` in `pending.md`, which step
1 has open anyway. No new command, and no false alarm.

Nail can override, because sometimes the queue is what he wants and the new
comments are for later. What he cannot do is miss it.

One API call, no new state, and it needs credentials `gdoc-apply` already needs
for `generate`. Under `--terminal-only` it still runs: reading is not posting.

This is not caused by OAuth. It is a gap the OAuth work uncovered, and it is
cheap enough to close here rather than file.

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
  "auth_mode": "oauth",
  "allow_document_edits": false
}
```

`auth_mode` defaults to `"oauth"` and accepts only `"oauth"` or
`"service_account"`. Any other value is an error at load, naming both accepted
values. `allow_document_edits` defaults to `false`.

A config file with neither key loads unchanged, the same way an older file with
`display_name` still loads today.

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
  cannot edit a document, and that limit is the design" is corrected to say who
  enforces the limit in each mode.
- `CLAUDE.md`: the `Never` list keeps "never edit a reviewed Google Doc" and
  gains the reason it is now enforced by `gdoc/guard.py` under OAuth.
- `skills/gdoc-review/SKILL.md`: step 2 split, the all-comments variant, the
  first-OAuth-run note, the `Never` line, and the step 2 warning text which
  currently says "under the service account address".
- `skills/gdoc-apply/SKILL.md`: the unreviewed-comments check in step 1, the
  author line when showing each item in step 2, and any wording that assumes the
  service account.

## 11. Out of scope

- **Editing a reviewed document in place.** `allow_document_edits` removes the
  block. It adds no capability. A command that edits a document directly, and
  the question of what that does to `baseline.md` and the regenerate loop, is a
  separate design.
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

**`tests/test_guard.py`** (new): `GET` passes through; `POST` to
`/drive/v3/files` passes; `POST` to `/upload/drive/v3/files` passes; `POST` to a
comments path passes; `POST` to a replies path passes; `PATCH` to a file is
refused; `DELETE` of a comment is refused; `POST` to
`/v1/documents/{id}:batchUpdate` is refused; a query string does not change any
verdict; the refusal is a `PermissionError` naming the method, the path and
`allow_document_edits`; the inner transport is never called on a refusal.

**`tests/test_config.py`**: `auth_mode` defaults to `oauth`; `allow_document_edits`
defaults to `false`; a config with neither key still loads; an unknown
`auth_mode` value raises and names both accepted values.

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
still the one refusing. Under `oauth` the same operations are attempted through
a guarded client and must raise `PermissionError` before any request leaves the
process. A third case asserts that with `allow_document_edits: true` under OAuth
the guard is absent, which is the only honest way to prove the flag does what it
says.

Coverage stays at or above 80%.

## 13. Order of work

1. The marker and the identity fix. It is a correctness change that stands on
   its own and is safe under the service account today.
2. Config keys, with the defaults but nothing reading them yet.
3. `gdoc/guard.py` and its tests, against a fake transport, no credentials.
4. `gdoc/oauth.py`, `auth.py` dispatch, the `auth` subcommands.
5. The captured-item format: author, marked, and the end of `Nail asked:`.
6. `--all` and the payload changes.
7. Wire the guard into `drive_service`.
8. Integration suite split.
9. install.sh, README, CLAUDE.md, both skills.

Steps 1 to 3 and step 5 need no Google Cloud setup, so section 9 can happen in
parallel with them.
