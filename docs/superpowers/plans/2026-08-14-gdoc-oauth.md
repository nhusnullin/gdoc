# OAuth Credential Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let gdoc authorise as Nail over OAuth, so a document no longer has to be shared with a service account address before it can be reviewed.

**Architecture:** `gdoc/auth.py` stays the single chokepoint and grows a mode switch fed by `config.auth_mode`. A new `gdoc/oauth.py` owns the browser flow and the token file. Because OAuth means full Drive scope on everything Nail can reach, a new `gdoc/guard.py` wraps the HTTP transport and confines every call to a set of file ids: the one the command was given, plus the ones gdoc's own creates returned. On those files it permits every method; on anything else it permits nothing. Identity stops deciding anything: replies carry a `[gdoc]` marker written by `gdoc/marker.py`, because Drive's `author.me` means the service account today and Nail under OAuth.

**Amended 2026-08-15.** Section 5 of the spec was rewritten after review, and Tasks 3, 4, 6, 10 and 11 changed with it. The guard was a verb allowlist; it is now a file-id allowlist. `allow_document_edits` is gone. See the spec's section 5 for why. If you are reading a cached copy of this plan, re-read those five tasks.

**Tech Stack:** Python 3.11+, `google-auth`, `google-auth-oauthlib` (new), `google-auth-httplib2`, `google-api-python-client`, pytest.

**Spec:** `docs/superpowers/specs/2026-08-14-gdoc-oauth-design.md`

## Principles

`PRINCIPLES.md` did not exist when this plan was first written. The gate is
answered here because five tasks were rewritten after it landed.

Serves: principle 3. The guard's allowed set starts empty, so a command that
does not name a document reaches nothing. Task 6 has a test for it.

Strains: principle 3. On a file in the set the guard permits every method,
including an edit to a reviewed document, so a limit Google enforced becomes one
gdoc respects. Task 11 rewrites the README, `CLAUDE.md` and the dated decision in
`PRINCIPLES.md` rather than leaving a promise standing that stopped being true.

Also touches principle 1: `google-auth-oauthlib` and `google-auth-httplib2` are
added in Task 5, both pip-installable.

## Global Constraints

- **Run tests with** `~/.config/gdoc-agent/venv/bin/pytest`. Nothing else is installed.
- **TDD, always.** Write the failing test, run it, watch it fail, then implement. A step that says "run it and see it fail" is not optional.
- **Scope is exactly** `["https://www.googleapis.com/auth/drive"]`, unchanged, shared by both modes. There is no narrower scope that reads comments and writes replies. Never narrow it to make a test pass.
- **`auth_mode` accepts only** `"oauth"` and `"service_account"`. Default `"oauth"`. It is the only config key this plan adds. *Superseded 2026-08-18: there is no default. An unstated mode is inferred from which credential files exist, because defaulting it flipped every working service_account install on upgrade. See the amendment at the end of the spec.*
- **The marker is exactly** `[gdoc]`, on its own last line, appended at most once.
- **The guard's allowed set may only grow through the two doors in Task 4:** the ids passed to `drive_service`, and the ids a create response returned. Never add a third. Never widen the set to make a test pass, and never relax a refusal for `files.list`, a permission write, or batch.
- **The tool must work without git and without a config file.** `gdoc read` works today with no `config.json`, and it must still work after this plan. A missing config resolves to the documented defaults, never to an error.
- **Writing style in all docs, comments and commit messages:** plain short English, and never the em dash character.
- **Immutability.** `Config`, `Thread` and `Reply` are frozen dataclasses. Return new values, never mutate.
- **New dependency:** `google-auth-oauthlib` is genuinely new. `google-auth-httplib2` is already present at 0.4.1 as a transitive dependency but becomes a direct one, so it is declared too.

### The collision is resolved

The template-merge workstream landed. `main` was merged into this branch on
2026-08-15 and the suite was green at 252 passed, 1 skipped before any task
below was started. `Config` already carries `template`, `cmd_generate` already
writes pairing frontmatter, and `generate` already runs its two passes.

What survives from that merge and matters here:

- **`gdoc/config.py` (Task 3):** `Config` has `output_folder_id` and `template`.
  Add `auth_mode` beside them. Do not reorder existing fields.
- **`gdoc/generate.py` (Task 4 and Task 6):** `generate` creates a measuring
  copy, exports it to PDF, creates the published copy, then trashes the
  measuring copy with `files.update`. Three of those calls address a file that
  did not exist when the client was built. This is the reason the guard learns
  ids from create responses, and `tests/test_generate.py` is the suite that
  proves it still works.
- Do not edit the main checkout at `/Users/nailkhusnullin/src/personal/gdoc`.
  Work only in this worktree.

`gdoc-apply-drains` is still unmerged and still written against `main`. It
renames `has_agent_reply` to `agent_replied_last`. Task 2 keeps the identity
test in one property so that merge stays a one-line change.

---

## File Structure

| File | Responsibility |
|---|---|
| `gdoc/marker.py` | **new.** The `[gdoc]` string and the two functions that add and detect it. Imported by both `model.py` and `reply.py`, so it lives on its own and neither depends on the other. |
| `gdoc/guard.py` | **new.** `file_id(uri)`, `verdict(method, uri, allowed)` and `GuardedHttp`. Knows nothing about credentials. |
| `gdoc/oauth.py` | **new.** The browser flow, the token file, and reading the signed-in account. Takes scopes as an argument so it never imports `auth.py`. |
| `gdoc/auth.py` | **modify.** The mode switch, and the one place the guard is installed. |
| `gdoc/config.py` | **modify.** `auth_mode`. |
| `gdoc/model.py` | **modify.** `Reply` gains `author_name` and `by_marker`. `has_agent_reply` stops depending on identity alone. |
| `gdoc/filters.py` | **modify.** `needs_action` drops the author test. `partition` gains all-comments mode. |
| `gdoc/reply.py` | **modify.** Post the marker. |
| `gdoc/pending.py` | **modify.** Record the author and the marker. Stop saying "Nail asked". |
| `gdoc/cli.py` | **modify.** `gdoc auth` subcommands, `read --all`, richer payload. |
| `tests/test_marker.py`, `tests/test_guard.py`, `tests/test_guard_is_installed.py`, `tests/test_oauth.py` | **new.** |
| `tests/test_auth.py`, `test_config.py`, `test_model.py`, `test_filters.py`, `test_reply.py`, `test_pending.py`, `test_cli.py`, `test_access_integration.py` | **modify.** |
| `pyproject.toml`, `install.sh`, `README.md`, `CLAUDE.md`, `skills/*/SKILL.md` | **modify.** |

---

## Task 1: The `[gdoc]` marker

Replies post under Nail's own name under OAuth, so identity can no longer tell his replies from the agent's. This is the thing that can.

**Files:**
- Create: `gdoc/marker.py`
- Create: `tests/test_marker.py`
- Modify: `gdoc/reply.py`
- Test: `tests/test_reply.py`

**Interfaces:**
- Consumes: nothing.
- Produces: `gdoc.marker.MARKER: str`, `gdoc.marker.with_marker(body: str) -> str`, `gdoc.marker.has_marker(text: str) -> bool`. Tasks 2 and 9 use `has_marker`.

- [ ] **Step 1: Write the failing test**

Create `tests/test_marker.py`:

```python
from gdoc.marker import MARKER, has_marker, with_marker


def test_the_marker_is_exactly_this():
    assert MARKER == "[gdoc]"


def test_marker_goes_on_its_own_last_line():
    assert with_marker("Use safeguarded funds.") == "Use safeguarded funds.\n\n[gdoc]"


def test_trailing_whitespace_does_not_produce_a_gap():
    assert with_marker("Use safeguarded funds.\n\n\n") == "Use safeguarded funds.\n\n[gdoc]"


def test_a_body_that_already_ends_with_the_marker_is_left_alone():
    once = with_marker("Answer.")
    assert with_marker(once) == once


def test_multiline_bodies_keep_their_shape():
    body = "Answer.\n\nSources: domains/regulatory/cbc-emi.md"
    assert with_marker(body) == body + "\n\n[gdoc]"


def test_has_marker_finds_it_as_the_last_line():
    assert has_marker("Answer.\n\n[gdoc]") is True


def test_has_marker_ignores_trailing_blank_lines():
    assert has_marker("Answer.\n\n[gdoc]\n\n") is True


def test_a_mid_sentence_mention_is_not_the_marker():
    assert has_marker("I ran [gdoc] on this and it worked") is False


def test_a_marker_that_is_not_last_does_not_count():
    assert has_marker("[gdoc]\n\nAnswer.") is False


def test_empty_text_has_no_marker():
    assert has_marker("") is False
    assert has_marker(None) is False
```

- [ ] **Step 2: Run test to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_marker.py -v`
Expected: FAIL, collection error `ModuleNotFoundError: No module named 'gdoc.marker'`

- [ ] **Step 3: Write minimal implementation**

Create `gdoc/marker.py`:

```python
"""The label gdoc writes on its own replies.

Under OAuth a reply posts under Nail's own account, so Drive's author.me field
can no longer tell his replies from the agent's. This marker can.

It is visible on purpose. A reader of the thread now needs some way to see which
replies a person wrote, and an invisible marker would hide exactly the thing that
got more confusing. It also survives copy and paste, which invisible state does
not.

It is a label, not a signature. Anyone can type it and make the tool treat a
thread as answered. The result is a skipped thread, which is the safe failure.
"""

MARKER = "[gdoc]"


def _last_content_line(text: str) -> str | None:
    lines = [line.strip() for line in (text or "").splitlines() if line.strip()]
    return lines[-1] if lines else None


def with_marker(body: str) -> str:
    """Return body with the marker as its last line, added at most once."""
    trimmed = (body or "").rstrip()
    if has_marker(trimmed):
        return trimmed
    return f"{trimmed}\n\n{MARKER}"


def has_marker(text: str) -> bool:
    """True when the last line with content on it is exactly the marker."""
    return _last_content_line(text) == MARKER
```

- [ ] **Step 4: Run test to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_marker.py -v`
Expected: PASS, 10 tests

- [ ] **Step 5: Write the failing test for posting it**

Append to `tests/test_reply.py`:

```python
def test_the_posted_body_carries_the_marker():
    drive = MagicMock()
    drive.replies().create.return_value.execute.return_value = {"id": "r1"}
    post_reply(drive, "doc", "comment", "Plain text answer.")
    posted = drive.replies().create.call_args.kwargs["body"]["content"]
    assert posted == "Plain text answer.\n\n[gdoc]"


def test_the_marker_itself_passes_the_markdown_check():
    """Otherwise the tool would refuse its own replies."""
    from gdoc.marker import with_marker

    assert assert_plain_text(with_marker("Plain text answer."))


def test_the_refusal_text_also_carries_the_marker_once_posted():
    drive = MagicMock()
    drive.replies().create.return_value.execute.return_value = {"id": "r1"}
    post_reply(drive, "doc", "comment", GLOBAL_REFUSAL.format(item=2))
    posted = drive.replies().create.call_args.kwargs["body"]["content"]
    assert posted.endswith("\n\n[gdoc]")
    assert posted.count("[gdoc]") == 1
```

- [ ] **Step 6: Run test to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_reply.py -v`
Expected: FAIL on `test_the_posted_body_carries_the_marker` with the posted content equal to `"Plain text answer."`, no marker

- [ ] **Step 7: Write minimal implementation**

In `gdoc/reply.py`, add the import beside the existing `import re`:

```python
from gdoc.marker import with_marker
```

Then change `post_reply` so the body it sends carries the marker:

```python
def post_reply(drive, doc_id: str, comment_id: str, body: str) -> str:
    """Post one reply, marked so a later run can recognise it.

    The markdown check runs on what the agent wrote, before the marker is added.
    The marker is checked into the test suite as plain text, so appending it can
    never turn an accepted body into a rejected one.
    """
    assert_plain_text(body)
    created = (
        drive.replies()
        .create(
            fileId=doc_id,
            commentId=comment_id,
            body={"content": with_marker(body)},
            fields="id,createdTime",
        )
        .execute()
    )
    return created["id"]
```

- [ ] **Step 8: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS. 153 passed before this task, so expect 166.

- [ ] **Step 9: Commit**

```bash
git add gdoc/marker.py gdoc/reply.py tests/test_marker.py tests/test_reply.py
git commit -m "feat: mark gdoc's own replies with a [gdoc] line

Identity stops working as a signal under OAuth, where a reply posts under
Nail's own account. A marker the tool writes still works, and it is visible
so a reader of the thread can tell who wrote what."
```

---

## Task 2: Identity stops deciding

The change OAuth cannot ship without. `needs_action` requires `not thread.by_agent`, and under OAuth `by_agent` is true for every comment Nail writes, so `gdoc read` would return an empty `addressed` list on every document.

**Files:**
- Modify: `gdoc/model.py`
- Modify: `gdoc/filters.py:needs_action`
- Test: `tests/test_model.py`, `tests/test_filters.py`

**Interfaces:**
- Consumes: `gdoc.marker.has_marker` from Task 1.
- Produces: `Reply(id, content, by_agent, author_name="unknown", by_marker=False)` and `Thread.has_agent_reply` counting either signal. Task 9 reads `Reply.author_name` and `Thread.has_agent_reply`.

- [ ] **Step 1: Write the failing test**

Append to `tests/test_model.py`:

```python
MARKED_REPLY = {
    "id": "AAACFjp837Q",
    "content": "ai! renumber the annex",
    "author": {"displayName": "William Mejia", "me": False},
    "quotedFileContent": {"value": "Annex 2"},
    "resolved": False,
    "replies": [
        {
            "id": "AAACFjd7zKQ",
            "content": "Captured as item 2.\n\n[gdoc]",
            "author": {"displayName": "Nail Khusnullin", "me": True},
        }
    ],
}


def test_a_reply_ending_in_the_marker_is_gdocs_own():
    thread = parse_thread(MARKED_REPLY)
    assert thread.replies[0].by_marker is True


def test_the_marker_alone_makes_a_thread_answered():
    """Under OAuth `me` is Nail, so the marker has to carry this on its own."""
    raw = {
        **MARKED_REPLY,
        "replies": [
            {
                "id": "r1",
                "content": "Captured as item 2.\n\n[gdoc]",
                "author": {"displayName": "Nail Khusnullin", "me": False},
            }
        ],
    }
    thread = parse_thread(raw)
    assert thread.replies[0].by_agent is False
    assert thread.replies[0].by_marker is True
    assert thread.has_agent_reply is True


def test_me_alone_still_makes_a_thread_answered():
    """Threads the service account answered before the marker existed."""
    assert parse_thread(ALREADY_ANSWERED).has_agent_reply is True


def test_a_reply_mentioning_the_marker_mid_sentence_is_not_gdocs():
    raw = {
        **MARKED_REPLY,
        "replies": [
            {
                "id": "r1",
                "content": "I saw [gdoc] answer this elsewhere",
                "author": {"displayName": "William Mejia", "me": False},
            }
        ],
    }
    thread = parse_thread(raw)
    assert thread.replies[0].by_marker is False
    assert thread.has_agent_reply is False


def test_replies_carry_their_author_name():
    assert parse_thread(MARKED_REPLY).replies[0].author_name == "Nail Khusnullin"


def test_a_reply_with_no_author_block_is_unknown():
    raw = {**MARKED_REPLY, "replies": [{"id": "r1", "content": "hi"}]}
    assert parse_thread(raw).replies[0].author_name == "unknown"
```

Now replace the existing `test_threads_the_agent_wrote_are_skipped` in `tests/test_filters.py`. Delete it, and add:

```python
def test_a_marked_comment_the_credential_wrote_is_still_work():
    """The regression this whole change turns on.

    Under OAuth the credential is Nail, so by_agent is true for every comment he
    writes. If it still gated needs_action, gdoc read would return nothing.
    """
    assert needs_action(thread(by_agent=True)) is True


def test_a_marker_reply_counts_as_answered_even_without_me():
    answered = thread(
        replies=(Reply(id="r1", content="Done.\n\n[gdoc]", by_agent=False, by_marker=True),)
    )
    assert needs_action(answered) is False
```

- [ ] **Step 2: Run test to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_model.py tests/test_filters.py -v`
Expected: FAIL. `test_a_reply_ending_in_the_marker_is_gdocs_own` fails with `AttributeError: 'Reply' object has no attribute 'by_marker'`, and `test_a_marked_comment_the_credential_wrote_is_still_work` fails asserting `False is True`.

- [ ] **Step 3: Write minimal implementation**

In `gdoc/model.py`, add the import:

```python
from gdoc.marker import has_marker
```

Replace the `Reply` dataclass. The two new fields carry defaults so existing constructions stay valid:

```python
@dataclass(frozen=True)
class Reply:
    id: str
    content: str
    by_agent: bool
    author_name: str = "unknown"
    by_marker: bool = False
```

Replace `Thread.has_agent_reply`:

```python
    @property
    def has_agent_reply(self) -> bool:
        """True when gdoc has already answered here.

        Two signals, because neither covers both credentials. The marker is the
        one that works under OAuth, where `me` is Nail. `me` is kept for threads
        the service account answered before the marker existed: dropping it would
        repost on every one of them.
        """
        return any(reply.by_agent or reply.by_marker for reply in self.replies)
```

In `parse_thread`, build the replies with the two new fields:

```python
    replies = tuple(
        Reply(
            id=item.get("id", ""),
            content=item.get("content") or "",
            by_agent=bool((item.get("author") or {}).get("me")),
            author_name=(item.get("author") or {}).get("displayName") or "unknown",
            by_marker=has_marker(item.get("content") or ""),
        )
        for item in raw.get("replies") or ()
    )
```

In `gdoc/filters.py`, replace `needs_action`:

```python
def needs_action(thread: Thread) -> bool:
    """True when the thread is waiting on gdoc.

    Who wrote the comment is not part of the test. Under OAuth the credential is
    Nail, so an author check would skip every comment he writes. The marker is
    the instruction, and Nail choosing the document is the trust decision.

    has_agent_reply is what makes a second run idempotent: it never posts twice,
    and no local record of handled ids is needed.
    """
    return (
        is_addressed(thread.content)
        and not thread.resolved
        and not thread.has_agent_reply
    )
```

Update the module docstring's second paragraph in `gdoc/filters.py` to say the author is not checked at all, rather than that it could not be verified.

- [ ] **Step 4: Run test to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_model.py tests/test_filters.py -v`
Expected: PASS

- [ ] **Step 5: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add gdoc/model.py gdoc/filters.py tests/test_model.py tests/test_filters.py
git commit -m "fix: stop letting the credential's identity decide what is work

Drive's author.me means the service account today and Nail under OAuth, so
needs_action rejecting by_agent would skip every ai: comment Nail writes.
The marker carries idempotence instead, with me kept only so threads the
service account already answered are not answered twice."
```

---

## Task 3: The auth_mode config key

**Files:**
- Modify: `gdoc/config.py`
- Test: `tests/test_config.py`

**Interfaces:**
- Consumes: nothing.
- Produces: `Config(output_folder_id, template, auth_mode)` and `gdoc.config.AUTH_MODES`. Task 6 reads `auth_mode`.

**Amended 2026-08-15.** The first draft of this task added a second key, `allow_document_edits`. It is gone. Editing the named document is allowed by the guard now, so the key's only remaining effect would have been to switch the guard off entirely, and a key that exists only to disable a safety net gets set once and never unset. Add one field, not two.

**Collision:** `Config` already carries `template` from the template-merge work. Keep it. Add `auth_mode` beside it and leave the existing fields in their current order.

- [ ] **Step 1: Write the failing test**

Append to `tests/test_config.py`:

```python
def test_auth_mode_defaults_to_oauth(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    assert load_config(path).auth_mode == "oauth"


def test_auth_mode_can_be_the_service_account(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"auth_mode": "service_account"}))
    assert load_config(path).auth_mode == "service_account"


def test_an_unknown_auth_mode_is_refused_and_names_both_options(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"auth_mode": "magic"}))
    with pytest.raises(ValueError) as excinfo:
        load_config(path)
    message = str(excinfo.value)
    assert "magic" in message
    assert "oauth" in message
    assert "service_account" in message


def test_nails_live_config_still_loads_without_the_new_key(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Nail Khusnullin", "output_folder_id": None}))
    config = load_config(path)
    assert config.auth_mode == "oauth"
```

- [ ] **Step 2: Run test to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_config.py -v`
Expected: FAIL with `AttributeError: 'Config' object has no attribute 'auth_mode'`

- [ ] **Step 3: Write minimal implementation**

In `gdoc/config.py`, add the constants, extend the dataclass, and validate on load:

```python
AUTH_MODES = ("oauth", "service_account")
DEFAULT_AUTH_MODE = "oauth"
```

Add one field to `Config`, after the existing ones:

```python
    auth_mode: str = DEFAULT_AUTH_MODE
```

In `load_config`, validate before constructing:

```python
    auth_mode = data.get("auth_mode") or DEFAULT_AUTH_MODE
    if auth_mode not in AUTH_MODES:
        raise ValueError(
            f"auth_mode in {path} is {auth_mode!r}. "
            f"It must be one of: {', '.join(AUTH_MODES)}"
        )
```

and pass `auth_mode=auth_mode` to `Config(...)`, leaving every existing argument untouched.

Extend the `load_config` docstring with one sentence: `auth_mode` defaults to `oauth`, so a config written before the key existed keeps loading.

- [ ] **Step 4: Run test to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_config.py -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gdoc/config.py tests/test_config.py
git commit -m "feat: add auth_mode to config

It defaults to oauth and rejects anything but the two known values. A
config written before the key existed keeps loading. Nothing reads it yet."
```

---

## Task 4: The guard

**Rewritten 2026-08-15.** The first version of this task built a verb allowlist: any read passed, four writes passed, everything else was refused. It guarded the wrong axis and it was already out of date, because `generate` gained a `files.update` call that trashes its measuring copy. Read the spec's section 5 before starting. Do not reuse the old test file from git history.

What the guard does now: it holds a set of file ids, and it carries a request only when the file the request addresses is in that set. Method is consulted only to tell a create from a listing.

No credentials are involved, so this tests entirely offline.

**Files:**
- Create: `gdoc/guard.py`
- Create: `tests/test_guard.py`

**Interfaces:**
- Consumes: nothing.
- Produces: `gdoc.guard.file_id(uri) -> str | None`, `gdoc.guard.verdict(method, uri, allowed) -> str`, `gdoc.guard.GuardedHttp(inner, allowed)`. Task 6 wraps the authorised transport in `GuardedHttp` and passes the ids the command was given.

**Why `verdict` returns a string, not a bool.** Three outcomes matter, not two: refuse, carry, and carry-then-learn-the-new-id. A bool cannot say the third, and the create case is what makes `generate` work.

- [ ] **Step 1: Write the failing test**

Create `tests/test_guard.py`:

```python
"""The reachable set, asserted against the guard.

Under a service account the credential reached only the documents shared with
it. OAuth has no such limit: the token is Nail, the scope is full Drive, and
every file Nail owns is reachable. This file is what narrows it back down.

The rule is about files, never about methods. On a file the command was given,
every method is carried, an edit included. On any other file, nothing is,
a read included. Never widen the set to make a test pass.
"""

import json

import pytest

from gdoc.guard import CARRY, LEARN, REFUSE, GuardedHttp, file_id, verdict

MINE = "1TheDocumentIWasGiven"
YOURS = "1SomeOtherFileEntirely"
ALLOWED = frozenset({MINE})

DRIVE = "https://www.googleapis.com/drive/v3"
UPLOAD = "https://www.googleapis.com/upload/drive/v3"
DOCS = "https://docs.googleapis.com/v1"


class FakeHttp:
    """Records what it was asked to carry, and answers with a canned body."""

    def __init__(self, body=b"{}", status="200"):
        self.calls = []
        self.body = body
        self.status = status
        self.credentials = "the credentials"

    def request(self, uri, method="GET", body=None, headers=None, **kwargs):
        self.calls.append((method, uri))
        return ({"status": self.status}, self.body)


def guarded(allowed=ALLOWED, **kwargs):
    inner = FakeHttp(**kwargs)
    return GuardedHttp(inner, allowed), inner


# --- reading the file id out of a url ------------------------------------


def test_the_id_is_read_from_a_drive_path():
    assert file_id(f"{DRIVE}/files/{MINE}") == MINE


def test_the_id_is_read_from_a_drive_subpath():
    assert file_id(f"{DRIVE}/files/{MINE}/comments/c1/replies") == MINE


def test_the_id_is_read_from_an_upload_path():
    assert file_id(f"{UPLOAD}/files/{MINE}?uploadType=media") == MINE


def test_the_id_is_read_from_a_docs_path():
    assert file_id(f"{DOCS}/documents/{MINE}") == MINE


def test_the_id_stops_at_the_colon_in_a_docs_method():
    """documents/{id}:batchUpdate is the write that matters most."""
    assert file_id(f"{DOCS}/documents/{MINE}:batchUpdate") == MINE


def test_a_collection_path_carries_no_id():
    assert file_id(f"{DRIVE}/files") is None


def test_discovery_carries_no_id():
    assert file_id("https://www.googleapis.com/discovery/v1/apis/drive/v3/rest") is None


# --- every method is carried on the file we were given -------------------


@pytest.mark.parametrize(
    "method,uri",
    [
        ("GET", f"{DRIVE}/files/{MINE}"),
        ("GET", f"{DRIVE}/files/{MINE}/export?mimeType=application%2Fpdf"),
        ("GET", f"{DRIVE}/files/{MINE}/comments?fields=comments(id)"),
        ("POST", f"{DRIVE}/files/{MINE}/comments"),
        ("POST", f"{DRIVE}/files/{MINE}/comments/c1/replies?fields=id"),
        ("PATCH", f"{DRIVE}/files/{MINE}"),
        ("PUT", f"{UPLOAD}/files/{MINE}?uploadType=media"),
        ("DELETE", f"{DRIVE}/files/{MINE}/comments/c1"),
        ("POST", f"{DOCS}/documents/{MINE}:batchUpdate"),
    ],
)
def test_any_method_is_carried_on_the_named_file(method, uri):
    assert verdict(method, uri, ALLOWED) == CARRY


def test_trashing_the_measuring_copy_is_carried():
    """generate creates a copy, measures it, then bins it. This is the bin."""
    assert verdict("PATCH", f"{DRIVE}/files/{MINE}", ALLOWED) == CARRY


# --- nothing at all on any other file ------------------------------------


@pytest.mark.parametrize(
    "method,uri",
    [
        ("GET", f"{DRIVE}/files/{YOURS}"),
        ("GET", f"{DRIVE}/files/{YOURS}/comments"),
        ("GET", f"{DRIVE}/files/{YOURS}/export?mimeType=application%2Fpdf"),
        ("POST", f"{DRIVE}/files/{YOURS}/comments"),
        ("PATCH", f"{DRIVE}/files/{YOURS}"),
        ("DELETE", f"{DRIVE}/files/{YOURS}"),
        ("POST", f"{DOCS}/documents/{YOURS}:batchUpdate"),
    ],
)
def test_nothing_is_carried_on_a_file_we_were_not_given(method, uri):
    assert verdict(method, uri, ALLOWED) == REFUSE


def test_reading_someone_elses_document_is_refused():
    """The failure the verb allowlist could not see. A GET is not harmless."""
    assert verdict("GET", f"{DRIVE}/files/{YOURS}", ALLOWED) == REFUSE


def test_an_empty_allowed_set_refuses_a_read():
    """A command that forgets to name its document reaches nothing."""
    assert verdict("GET", f"{DRIVE}/files/{MINE}", frozenset()) == REFUSE


# --- the collection paths ------------------------------------------------


def test_listing_drive_is_refused():
    """gdoc never searches Drive. This is most of 'it sees only what it is given'."""
    assert verdict("GET", f"{DRIVE}/files?q=name+contains+'policy'", ALLOWED) == REFUSE


def test_creating_a_file_is_carried_and_learned():
    assert verdict("POST", f"{DRIVE}/files", frozenset()) == LEARN


def test_creating_a_file_with_media_is_carried_and_learned():
    assert verdict("POST", f"{UPLOAD}/files?uploadType=multipart", frozenset()) == LEARN


# --- refused whatever the file ------------------------------------------


def test_sharing_the_named_file_is_refused():
    """Granting other people access is a different authority from editing."""
    assert verdict("POST", f"{DRIVE}/files/{MINE}/permissions", ALLOWED) == REFUSE


def test_revoking_access_to_the_named_file_is_refused():
    assert verdict("DELETE", f"{DRIVE}/files/{MINE}/permissions/p1", ALLOWED) == REFUSE


def test_changing_a_permission_on_the_named_file_is_refused():
    assert verdict("PATCH", f"{DRIVE}/files/{MINE}/permissions/p1", ALLOWED) == REFUSE


def test_reading_who_has_access_to_the_named_file_is_carried():
    """A read of the named file like any other.

    Under a service account Google answers this with a 403 regardless, which
    tests/test_access_integration.py records. Refusing it here would buy
    nothing and would hide that.
    """
    assert verdict("GET", f"{DRIVE}/files/{MINE}/permissions", ALLOWED) == CARRY


def test_reading_who_has_access_to_another_file_is_still_refused():
    assert verdict("GET", f"{DRIVE}/files/{YOURS}/permissions", ALLOWED) == REFUSE


def test_a_batch_request_is_refused():
    """The file ids live in the body, where the guard cannot see them."""
    assert verdict("POST", "https://www.googleapis.com/batch/drive/v3", ALLOWED) == REFUSE


# --- carried with no file involved ---------------------------------------


def test_discovery_is_carried():
    uri = "https://www.googleapis.com/discovery/v1/apis/drive/v3/rest"
    assert verdict("GET", uri, frozenset()) == CARRY


def test_about_get_is_carried():
    """auth status asks who is signed in, and that names no file."""
    assert verdict("GET", f"{DRIVE}/about?fields=user", frozenset()) == CARRY


# --- matching details ----------------------------------------------------


def test_the_query_string_changes_no_verdict():
    assert verdict("POST", f"{UPLOAD}/files?uploadType=resumable", frozenset()) == LEARN
    assert verdict("GET", f"{DRIVE}/files?corpora=allDrives", ALLOWED) == REFUSE


def test_the_host_is_matched_not_only_the_path():
    assert verdict("GET", f"https://example.com/drive/v3/files/{MINE}", ALLOWED) == REFUSE


def test_the_method_is_case_insensitive():
    assert verdict("patch", f"{DRIVE}/files/{MINE}", ALLOWED) == CARRY
    assert verdict("get", f"{DRIVE}/files/{YOURS}", ALLOWED) == REFUSE


def test_an_unknown_googleapis_path_is_refused():
    assert verdict("GET", "https://www.googleapis.com/gmail/v1/users/me/messages", ALLOWED) == REFUSE


# --- the transport -------------------------------------------------------


def test_a_carried_request_reaches_the_inner_transport():
    http, inner = guarded()
    http.request(f"{DRIVE}/files/{MINE}/comments", method="GET")
    assert inner.calls == [("GET", f"{DRIVE}/files/{MINE}/comments")]


def test_a_refused_request_never_reaches_the_inner_transport():
    http, inner = guarded()
    with pytest.raises(PermissionError):
        http.request(f"{DRIVE}/files/{YOURS}", method="GET")
    assert inner.calls == []


def test_the_refusal_names_the_method_the_path_and_the_file():
    http, _ = guarded()
    with pytest.raises(PermissionError) as excinfo:
        http.request(f"{DRIVE}/files/{YOURS}", method="PATCH")
    message = str(excinfo.value)
    assert "PATCH" in message
    assert f"/drive/v3/files/{YOURS}" in message
    assert YOURS in message
    assert "was given" in message


def test_the_default_method_is_get():
    http, inner = guarded()
    http.request(f"{DRIVE}/files/{MINE}")
    assert inner.calls == [("GET", f"{DRIVE}/files/{MINE}")]


def test_other_attributes_proxy_to_the_inner_transport():
    """googleapiclient reaches past request() during media upload."""
    http, _ = guarded()
    assert http.credentials == "the credentials"


# --- learning a created id -----------------------------------------------


def test_a_created_file_joins_the_allowed_set():
    body = json.dumps({"id": "1FreshlyCreated"}).encode()
    http, _ = guarded(allowed=frozenset(), body=body)
    http.request(f"{UPLOAD}/files?uploadType=multipart", method="POST")
    assert verdict("GET", f"{DRIVE}/files/1FreshlyCreated", http.allowed) == CARRY


def test_the_whole_generate_shape_works_from_an_empty_set():
    """create, export, create, trash. Three of the four address a new file."""
    body = json.dumps({"id": "1Measured"}).encode()
    http, inner = guarded(allowed=frozenset(), body=body)
    http.request(f"{UPLOAD}/files?uploadType=multipart", method="POST")
    http.request(f"{DRIVE}/files/1Measured/export?mimeType=application%2Fpdf")
    http.request(f"{DRIVE}/files/1Measured", method="PATCH")
    assert len(inner.calls) == 3


def test_a_create_that_answers_with_something_else_teaches_nothing():
    http, _ = guarded(allowed=frozenset(), body=b"<html>a proxy said no</html>")
    http.request(f"{DRIVE}/files", method="POST")
    assert http.allowed == frozenset()


def test_a_create_that_answers_without_an_id_teaches_nothing():
    http, _ = guarded(allowed=frozenset(), body=json.dumps({"kind": "drive#file"}).encode())
    http.request(f"{DRIVE}/files", method="POST")
    assert http.allowed == frozenset()


def test_a_failed_create_teaches_nothing():
    body = json.dumps({"id": "1NeverActuallyMade"}).encode()
    http, _ = guarded(allowed=frozenset(), body=body, status="403")
    http.request(f"{DRIVE}/files", method="POST")
    assert http.allowed == frozenset()


def test_learning_does_not_mutate_the_set_it_was_given():
    """The caller's frozenset must not change under it."""
    given = frozenset({MINE})
    body = json.dumps({"id": "1FreshlyCreated"}).encode()
    http, _ = guarded(allowed=given, body=body)
    http.request(f"{DRIVE}/files", method="POST")
    assert given == frozenset({MINE})
    assert http.allowed == frozenset({MINE, "1FreshlyCreated"})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_guard.py -v`
Expected: FAIL, collection error `ModuleNotFoundError: No module named 'gdoc.guard'`

- [ ] **Step 3: Write minimal implementation**

Create `gdoc/guard.py`:

```python
"""Confine every request to the files the command was given.

Under a service account the credential reached only the documents that had been
shared with it, and on those it was a Commenter. Two limits, both Google's: a
small reachable set, and no writing inside it. OAuth removes both, because the
token is Nail and the scope is full Drive. There is no narrower scope that reads
comments and writes replies, so the reachable set has to be narrowed here.

The rule is about files, never about methods. A verb allowlist would have left
every document Nail owns readable, which is the risk OAuth actually introduces,
and it would need editing every time the tool learns a new call.

The check sits in the HTTP transport rather than at the call sites, so it sees
every request the client makes: calls added later, and any other Google API
built on the same transport, which is how the Docs API is covered without the
call sites knowing it exists.
"""

import json
import re
from urllib.parse import urlsplit

CARRY = "carry"
LEARN = "learn"
REFUSE = "refuse"

_DRIVE_HOST = "www.googleapis.com"
_DOCS_HOST = "docs.googleapis.com"

# The three path shapes that name a file.
_FILE_PATHS = (
    re.compile(r"^/drive/v3/files/([^/]+)"),
    re.compile(r"^/upload/drive/v3/files/([^/]+)"),
    re.compile(r"^/v1/documents/([^/:]+)"),
)

# Creating a file names no id, because it is making one.
_CREATE_PATHS = frozenset({"/drive/v3/files", "/upload/drive/v3/files"})

# Neither does asking what the API looks like, or who is signed in.
_NO_FILE_PATHS = (
    re.compile(r"^/discovery/"),
    re.compile(r"^/drive/v3/about$"),
)

# Changing who else can reach a file is refused on every file, allowed or not.
# Granting other people access is a different authority from changing a
# document, and gdoc has no use for it.
#
# Reading the list is not refused here. It is a read of the named file like any
# other, and under a service account Google answers it with a 403 anyway, which
# tests/test_access_integration.py records.
_FORBIDDEN_PATH = re.compile(r"/permissions(/|$)")
_READ_METHODS = frozenset({"GET", "HEAD"})


def file_id(uri: str) -> str | None:
    """The file a request addresses, or None when it names none.

    The Docs API writes documents/{id}:batchUpdate, so the id stops at the
    colon.
    """
    path = urlsplit(uri).path
    for pattern in _FILE_PATHS:
        found = pattern.match(path)
        if found:
            return found.group(1)
    return None


def verdict(method: str, uri: str, allowed) -> str:
    """CARRY, LEARN or REFUSE.

    LEARN is a create: carry it, then take the id out of the response. Three
    outcomes rather than two, because generate addresses files that did not
    exist when the client was built.

    The query string is ignored, so fields and uploadType cannot change a
    verdict. The host is matched, so a familiar path shape elsewhere does not
    pass.
    """
    method = (method or "GET").upper()
    split = urlsplit(uri)
    if split.hostname not in (_DRIVE_HOST, _DOCS_HOST):
        return REFUSE
    path = split.path
    if _FORBIDDEN_PATH.search(path) and method not in _READ_METHODS:
        return REFUSE
    named = file_id(uri)
    if named is not None:
        return CARRY if named in allowed else REFUSE
    if path in _CREATE_PATHS and method == "POST":
        return LEARN
    if any(pattern.match(path) for pattern in _NO_FILE_PATHS):
        return CARRY
    # files.list, batch, and anything else that names no file.
    return REFUSE


class GuardedHttp:
    """An httplib2-shaped transport that reaches only the files it was given.

    The signature mirrors google_auth_httplib2.AuthorizedHttp.request, verified
    against the installed 0.4.1. Everything other than request is proxied,
    because googleapiclient reaches for more than request during media upload.

    The allowed set grows through exactly two doors: the ids handed in at
    construction, and the ids that creates this object carried came back with.
    Never add a third.
    """

    def __init__(self, inner, allowed=frozenset()):
        self._inner = inner
        self.allowed = frozenset(allowed)

    def request(
        self,
        uri,
        method="GET",
        body=None,
        headers=None,
        redirections=5,
        connection_type=None,
        **kwargs,
    ):
        decision = verdict(method, uri, self.allowed)
        if decision == REFUSE:
            self._refuse(method, uri)
        response, content = self._inner.request(
            uri,
            method=method,
            body=body,
            headers=headers,
            redirections=redirections,
            connection_type=connection_type,
            **kwargs,
        )
        if decision == LEARN:
            self._learn(response, content)
        return response, content

    def _refuse(self, method, uri):
        path = urlsplit(uri).path
        named = file_id(uri)
        if named:
            why = f"{named} is not a file it was given"
        else:
            why = "that request names no single file"
        raise PermissionError(
            f"gdoc refused {(method or 'GET').upper()} {path}. "
            f"It reaches only the files it was given, and {why}."
        )

    def _learn(self, response, content):
        """Add the id a create came back with.

        Never raises, and the except is broad for that reason. A body that is
        not JSON, a response object shaped unlike httplib2's, or a create that
        failed all teach nothing, and the next call on that file is refused.
        That is the safe direction: a stray temporary file beats a widened set.
        """
        try:
            status = getattr(response, "status", None)
            if status is None and hasattr(response, "get"):
                status = response.get("status")
            if not str(status or "").startswith("2"):
                return
            created = json.loads(content)
            new_id = created.get("id") if isinstance(created, dict) else None
        except Exception:  # noqa: BLE001 - see the docstring
            return
        if new_id:
            self.allowed = self.allowed | {new_id}

    def __getattr__(self, name):
        return getattr(self._inner, name)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_guard.py -v`
Expected: PASS, 54 tests

- [ ] **Step 5: Commit**

```bash
git add gdoc/guard.py tests/test_guard.py
git commit -m "feat: confine every request to the files the command was given

OAuth means full Drive scope on everything Nail can reach, so the tool has
to narrow that itself. The guard holds a set of file ids: the ones handed
in, plus the ones its own creates came back with. On those, every method.
On anything else, nothing, a read included.

files.list is refused, so gdoc cannot search Drive. Writing a permission is
refused on every file, because granting other people access is a different
authority; reading the list is not, because that is a read like any other.
Batch is refused, because the ids live in the body.

Nothing installs the guard yet."
```

---
## Task 5: The OAuth credential

**Files:**
- Create: `gdoc/oauth.py`
- Create: `tests/test_oauth.py`
- Modify: `pyproject.toml`

**Interfaces:**
- Consumes: nothing. It takes `scopes` as an argument rather than importing `auth.py`, so there is no import cycle.
- Produces: `gdoc.oauth.login(scopes, client_path=None, token_path=None) -> Credentials`, `gdoc.oauth.load(scopes, token_path=None) -> Credentials`, `gdoc.oauth.logout(token_path=None) -> bool`, `gdoc.oauth.account(drive) -> dict`, `gdoc.oauth.DEFAULT_CLIENT_PATH`, `gdoc.oauth.DEFAULT_TOKEN_PATH`. Tasks 6 and 7 call all of them.

**Deviation from the spec:** the spec writes `account(credentials)`. This plan uses `account(drive)`, taking the built Drive client, so nothing has to build a second client just to ask who is signed in.

- [ ] **Step 1: Add the dependency**

In `pyproject.toml`, replace the `dependencies` list:

```toml
dependencies = [
    "google-auth",
    "google-auth-oauthlib",
    "google-auth-httplib2",
    "google-api-python-client",
    "PyYAML",
]
```

`google-auth-oauthlib` is genuinely new. `google-auth-httplib2` is already installed at 0.4.1 as a transitive dependency of `google-api-python-client`, but `gdoc/guard.py` and `gdoc/auth.py` import it directly now, so it is declared. A transitive dependency that a later release drops is a build that breaks for no visible reason.

Install it:

```bash
~/.config/gdoc-agent/venv/bin/pip install -q -e '.[dev]'
```

Confirm:

```bash
~/.config/gdoc-agent/venv/bin/python -c "import google_auth_oauthlib.flow; print('ok')"
```
Expected: `ok`

- [ ] **Step 2: Write the failing test**

Create `tests/test_oauth.py`:

```python
"""The user credential, tested without a browser and without the network.

Real google.oauth2.credentials.Credentials objects are used throughout, because
they parse and serialise the token file the code actually writes. Only refresh is
patched, and it is patched for every test in the file, because of the trap below.

The trap, verified against google-auth 2.56.3 on 2026-08-14: when a token file
has no `expiry` key, from_authorized_user_info stamps expiry with the CURRENT
time, so the token loads as already expired. A real token from the browser flow
always has an expiry, so this only bites hand written files and test fixtures.
Get it wrong in a test and the test reaches Google.

Expiry values must be naive UTC. google-auth compares expiry against its own
naive _helpers.utcnow(), and mixing an aware datetime in raises TypeError.
"""

import json
import stat
from datetime import datetime, timedelta, timezone

import pytest
from google.oauth2.credentials import Credentials

from gdoc import oauth

SCOPES = ["https://www.googleapis.com/auth/drive"]
READONLY = "https://www.googleapis.com/auth/drive.readonly"


def naive_utc(delta=timedelta()):
    """Naive UTC, because that is what google-auth compares expiry against."""
    return datetime.now(timezone.utc).replace(tzinfo=None) + delta


@pytest.fixture(autouse=True)
def never_reach_google(monkeypatch):
    """No test in this file may leave the machine.

    load() refreshes whenever the token is not valid, and a fixture without a
    future expiry is not valid. Patching this for every test is the only way to
    be sure a new test cannot quietly start calling Google.
    """

    def fake_refresh(self, request):
        self.token = "a-fresh-token"
        self.expiry = naive_utc(timedelta(hours=1))

    monkeypatch.setattr(Credentials, "refresh", fake_refresh)


def credentials(expiry=None, scopes=SCOPES):
    return Credentials(
        token="an-access-token",
        refresh_token="a-refresh-token",
        token_uri="https://oauth2.googleapis.com/token",
        client_id="client-id",
        client_secret="client-secret",
        scopes=scopes,
        expiry=expiry if expiry is not None else naive_utc(timedelta(hours=1)),
    )


def write_token(path, creds):
    path.write_text(creds.to_json())
    return path


# --- load ---------------------------------------------------------------


def test_a_valid_token_loads_without_refreshing(tmp_path):
    token = write_token(tmp_path / "token.json", credentials())
    loaded = oauth.load(SCOPES, token_path=token)
    assert loaded.valid is True
    assert loaded.token == "an-access-token"


def test_a_token_file_with_no_expiry_is_refreshed_on_load(tmp_path):
    """Documents the trap in the module docstring.

    google-auth stamps a missing expiry with the current time, so such a file is
    expired the moment it is read. Refreshing is the right answer, and the point
    of the test is that the behaviour is known rather than discovered.
    """
    token = tmp_path / "token.json"
    token.write_text(
        json.dumps(
            {
                "token": "an-access-token",
                "refresh_token": "a-refresh-token",
                "client_id": "c",
                "client_secret": "s",
                "token_uri": "https://oauth2.googleapis.com/token",
                "scopes": SCOPES,
            }
        )
    )
    assert oauth.load(SCOPES, token_path=token).token == "a-fresh-token"


def test_a_missing_token_names_the_path_and_both_ways_out(tmp_path):
    token = tmp_path / "token.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        oauth.load(SCOPES, token_path=token)
    message = str(excinfo.value)
    assert str(token) in message
    assert "gdoc auth login" in message
    assert "service_account" in message


def test_an_expired_token_refreshes_and_the_file_is_rewritten(tmp_path):
    token = write_token(
        tmp_path / "token.json", credentials(expiry=naive_utc(timedelta(hours=-2)))
    )
    loaded = oauth.load(SCOPES, token_path=token)
    assert loaded.token == "a-fresh-token"
    assert json.loads(token.read_text())["token"] == "a-fresh-token"


def test_the_rewritten_token_file_is_still_private(tmp_path):
    token = write_token(
        tmp_path / "token.json", credentials(expiry=naive_utc(timedelta(hours=-2)))
    )
    token.chmod(0o644)
    oauth.load(SCOPES, token_path=token)
    assert stat.S_IMODE(token.stat().st_mode) == 0o600


def test_a_token_missing_scopes_is_refused_at_load(tmp_path):
    token = write_token(tmp_path / "token.json", credentials(scopes=[READONLY]))
    with pytest.raises(PermissionError) as excinfo:
        oauth.load(SCOPES, token_path=token)
    message = str(excinfo.value)
    assert "auth/drive" in message
    assert "gdoc auth login" in message


def test_the_scope_check_reads_the_file_not_the_object(tmp_path):
    """from_authorized_user_file reports the scopes it was ASKED for.

    So the object always looks correctly scoped, and only the file tells the
    truth. If this test ever passes while the previous one fails, the check moved
    to the object and stopped working.
    """
    token = write_token(tmp_path / "token.json", credentials(scopes=[READONLY]))
    loaded = Credentials.from_authorized_user_file(str(token), SCOPES)
    assert loaded.scopes == SCOPES


def test_a_token_file_that_cannot_be_used_says_so_rather_than_crashing(tmp_path):
    token = tmp_path / "token.json"
    token.write_text(json.dumps({"token": "t"}))
    with pytest.raises(PermissionError) as excinfo:
        oauth.load(SCOPES, token_path=token)
    assert "gdoc auth login" in str(excinfo.value)


def test_a_token_file_with_no_scopes_recorded_is_not_refused(tmp_path):
    """Only a hand written file lacks the key. Drive will say if it is wrong.

    Refusing on unknown would force a needless re-login. to_json always writes
    the key, so this only covers a file someone typed.
    """
    token = tmp_path / "token.json"
    token.write_text(
        json.dumps(
            {
                "token": "t",
                "refresh_token": "r",
                "client_id": "c",
                "client_secret": "s",
                "token_uri": "https://oauth2.googleapis.com/token",
                "expiry": naive_utc(timedelta(hours=1)).isoformat(),
            }
        )
    )
    assert oauth.load(SCOPES, token_path=token).token == "t"


# --- login --------------------------------------------------------------


def test_login_without_a_client_file_names_the_path_and_the_spec(tmp_path):
    client = tmp_path / "oauth-client.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        oauth.login(SCOPES, client_path=client, token_path=tmp_path / "token.json")
    message = str(excinfo.value)
    assert str(client) in message
    assert "Desktop" in message
    assert "2026-08-14-gdoc-oauth-design" in message


def test_login_writes_the_token_private(tmp_path, monkeypatch):
    client = tmp_path / "oauth-client.json"
    client.write_text(json.dumps({"installed": {"client_id": "c", "client_secret": "s"}}))
    token = tmp_path / "token.json"

    class FakeFlow:
        @classmethod
        def from_client_secrets_file(cls, path, scopes):
            assert scopes == SCOPES
            return cls()

        def run_local_server(self, **kwargs):
            assert kwargs["access_type"] == "offline"
            assert kwargs["prompt"] == "consent"
            return credentials()

    monkeypatch.setattr(oauth, "InstalledAppFlow", FakeFlow)

    oauth.login(SCOPES, client_path=client, token_path=token)
    assert token.exists()
    assert stat.S_IMODE(token.stat().st_mode) == 0o600


# --- logout -------------------------------------------------------------


def test_logout_removes_the_token_and_says_it_did(tmp_path):
    token = write_token(tmp_path / "token.json", credentials())
    assert oauth.logout(token_path=token) is True
    assert not token.exists()


def test_logout_on_a_machine_that_never_logged_in_says_so(tmp_path):
    assert oauth.logout(token_path=tmp_path / "token.json") is False


# --- account ------------------------------------------------------------


def test_account_asks_drive_who_is_signed_in():
    from unittest.mock import MagicMock

    drive = MagicMock()
    drive.about().get.return_value.execute.return_value = {
        "user": {"displayName": "Nail Khusnullin", "emailAddress": "nail@altery.com"}
    }
    assert oauth.account(drive)["emailAddress"] == "nail@altery.com"


def test_account_survives_a_reply_with_no_user_block():
    from unittest.mock import MagicMock

    drive = MagicMock()
    drive.about().get.return_value.execute.return_value = {}
    assert oauth.account(drive) == {}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_oauth.py -v`
Expected: FAIL, collection error `ModuleNotFoundError: No module named 'gdoc.oauth'`

- [ ] **Step 4: Write minimal implementation**

Create `gdoc/oauth.py`:

```python
"""The user credential.

A service account needs every document shared with its address first. OAuth does
not: the agent authorises once as Nail and reaches whatever Nail can reach. The
trade is that Drive has no scope which reads comments and writes replies without
full Drive access, so gdoc/guard.py carries the cannot-edit promise instead.

This module takes scopes as an argument and never imports gdoc.auth, so the
chokepoint can depend on this without a cycle.
"""

import json
import os
from pathlib import Path

from google.auth.transport.requests import Request
from google.oauth2.credentials import Credentials
from google_auth_oauthlib.flow import InstalledAppFlow

CONFIG_DIR = Path.home() / ".config" / "gdoc-agent"
DEFAULT_CLIENT_PATH = CONFIG_DIR / "oauth-client.json"
DEFAULT_TOKEN_PATH = CONFIG_DIR / "oauth-token.json"
SPEC = "docs/superpowers/specs/2026-08-14-gdoc-oauth-design.md"

LOGIN = "Run: gdoc auth login"
USE_THE_KEY_INSTEAD = (
    'To use the service account instead, set "auth_mode": "service_account" '
    "in ~/.config/gdoc-agent/config.json."
)


def _write_token(path: Path, credentials: Credentials) -> None:
    """Create the file private, then fill it. Never the other way round.

    The chmod is not redundant. O_CREAT applies its mode only when the file does
    not already exist, so refreshing into a file someone had loosened would leave
    it loosened. This is the one file in the tool that holds a live credential.
    """
    path.parent.mkdir(parents=True, exist_ok=True)
    handle = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(handle, "w") as stream:
        stream.write(credentials.to_json())
    os.chmod(path, 0o600)


def _stored_scopes(path: Path) -> set[str] | None:
    """The scopes the token records, or None when the file does not say.

    Read from the file rather than the Credentials object, because
    from_authorized_user_file reports back the scopes it was asked for, not the
    ones the token actually carries.
    """
    try:
        data = json.loads(path.read_text())
    except ValueError:
        return None
    scopes = data.get("scopes")
    return set(scopes) if isinstance(scopes, list) else None


def login(scopes, client_path=None, token_path=None) -> Credentials:
    """Run the browser flow and save the result."""
    client_path = Path(client_path or DEFAULT_CLIENT_PATH)
    token_path = Path(token_path or DEFAULT_TOKEN_PATH)
    if not client_path.exists():
        raise FileNotFoundError(
            f"OAuth client not found at {client_path}. Create a Desktop app "
            f"OAuth client in Google Cloud and save its JSON there. "
            f"Steps are in {SPEC}, section 9."
        )
    flow = InstalledAppFlow.from_client_secrets_file(str(client_path), list(scopes))
    # Both keyword arguments are passed rather than relied on. Without them
    # Google may return no refresh token, and the next run would silently need a
    # browser again.
    credentials = flow.run_local_server(
        port=0, access_type="offline", prompt="consent"
    )
    _write_token(token_path, credentials)
    return credentials


def load(scopes, token_path=None) -> Credentials:
    """Read the saved token, refreshing it if it has expired.

    A token file with no expiry recorded counts as expired, because google-auth
    stamps a missing expiry with the current time. That is fine: it refreshes and
    the file is rewritten with a real expiry. Worth knowing before you conclude
    the refresh path is running when it should not be.
    """
    token_path = Path(token_path or DEFAULT_TOKEN_PATH)
    if not token_path.exists():
        raise FileNotFoundError(
            f"no OAuth token at {token_path}. {LOGIN}\n{USE_THE_KEY_INSTEAD}"
        )

    stored = _stored_scopes(token_path)
    if stored is not None:
        missing = set(scopes) - stored
        if missing:
            raise PermissionError(
                f"the token at {token_path} is missing "
                f"{', '.join(sorted(missing))}. {LOGIN}"
            )

    try:
        credentials = Credentials.from_authorized_user_file(
            str(token_path), list(scopes)
        )
    except ValueError as error:
        raise PermissionError(
            f"the token at {token_path} cannot be used ({error}). {LOGIN}"
        ) from error

    if credentials.valid:
        return credentials

    # A token file always carries a refresh token, because
    # from_authorized_user_file refuses to parse one without it.
    credentials.refresh(Request())
    _write_token(token_path, credentials)
    return credentials


def logout(token_path=None) -> bool:
    """Delete the local token. Returns whether there was one.

    The grant with Google is untouched. Revoking it is a browser action, and
    gdoc auth status prints where.
    """
    token_path = Path(token_path or DEFAULT_TOKEN_PATH)
    if not token_path.exists():
        return False
    token_path.unlink()
    return True


def account(drive) -> dict:
    """Who the credential is.

    Asked of Drive, so knowing the signed-in account costs no extra scope.
    """
    about = drive.about().get(fields="user(displayName,emailAddress)").execute()
    return about.get("user") or {}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_oauth.py -v`
Expected: PASS, 15 tests

**If a test in this file hangs, it is reaching Google.** The `never_reach_google` fixture is autouse for exactly that reason. Check the new test is not constructing `Credentials` directly and bypassing it.

These facts were checked against the installed `google-auth` 2.56.3 on 2026-08-14, so trust them over intuition:

- `to_json()` omits `expiry` when it is `None`, and `from_authorized_user_info` then stamps expiry with the current time. A token file with no expiry is expired on arrival.
- `Credentials.expired` subtracts a 3 minute 45 second threshold, so a one hour future expiry is comfortably valid.
- `from_authorized_user_file(path, scopes)` sets `credentials.scopes` to the scopes you asked for, not the ones in the file. This is why `_stored_scopes` reads the JSON.
- `from_authorized_user_info` raises `ValueError` when `refresh_token`, `client_id` or `client_secret` is missing, which is why `load` has no unreachable no-refresh-token branch.
- `Credentials` is not frozen, so plain attribute assignment works inside a patched `refresh`.

- [ ] **Step 6: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add gdoc/oauth.py tests/test_oauth.py pyproject.toml
git commit -m "feat: add the OAuth credential

login runs the desktop flow and writes the token 0600, creating the file
private before filling it. load refreshes an expired token and rewrites it,
and refuses a token whose recorded scopes are too narrow rather than letting
Drive fail with a 403 halfway through a review.

Nothing selects this credential yet."
```

---

## Task 6: The mode switch, installing the guard, and threading the doc id

**Rewritten 2026-08-15,** with the guard. `drive_service` no longer reads a config key to decide whether to guard: the guard is always installed, and what the caller supplies instead is the set of file ids the command is allowed to touch. Five call sites in `cli.py` change with it.

**Files:**
- Modify: `gdoc/auth.py`
- Modify: `gdoc/cli.py`
- Test: `tests/test_auth.py`
- Create: `tests/test_guard_is_installed.py`

**Interfaces:**
- Consumes: `gdoc.oauth.load`, `gdoc.guard.GuardedHttp`, `gdoc.config.load_config`.
- Produces: `gdoc.auth.load_credentials(key_path=None, mode=None)`, `gdoc.auth.load_service_account_credentials(key_path=None)`, `gdoc.auth.drive_service(credentials=None, doc_ids=())`, `gdoc.auth.SCOPES`. Task 7 calls `drive_service()` with no ids, which is correct: `about.get` names no file.

**Three traps in this task.**

First, `tests/test_auth.py::test_missing_key_names_the_path_and_the_spec` calls `load_credentials(missing)` and expects the service account error. With `oauth` as the default it would get the OAuth error instead. That test must pass `mode="service_account"`.

Second, `gdoc read` works today with no `config.json`. If `load_credentials` calls `load_config()` and lets `FileNotFoundError` escape, that stops being true. A missing config must resolve to the documented defaults.

Third, `cmd_capture` builds the client on the line **before** it extracts the doc id. Reorder it, or the id is not available to pass.

- [ ] **Step 1: Write the failing test**

Replace the whole of `tests/test_auth.py`:

```python
from unittest.mock import MagicMock, patch

import pytest

from gdoc.auth import (
    SCOPES,
    drive_service,
    load_credentials,
    load_service_account_credentials,
)
from gdoc.config import Config
from gdoc.guard import GuardedHttp


def test_scope_is_drive_only():
    """There is no narrower scope that reads comments and writes replies."""
    assert SCOPES == ["https://www.googleapis.com/auth/drive"]


def test_missing_key_names_the_path_and_the_spec(tmp_path):
    missing = tmp_path / "sa-key.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        load_service_account_credentials(missing)
    message = str(excinfo.value)
    assert str(missing) in message
    assert "2026-08-13-gdoc-ai-agent-design" in message


def test_service_account_mode_reads_the_key(tmp_path):
    missing = tmp_path / "sa-key.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        load_credentials(key_path=missing, mode="service_account")
    assert str(missing) in str(excinfo.value)


def test_oauth_mode_reads_the_token():
    with patch("gdoc.auth.oauth.load", return_value="oauth-credentials") as load:
        assert load_credentials(mode="oauth") == "oauth-credentials"
    load.assert_called_once_with(SCOPES)


def test_the_mode_comes_from_config_when_not_given():
    with patch("gdoc.auth.load_config", return_value=Config(auth_mode="oauth")), patch(
        "gdoc.auth.oauth.load", return_value="oauth-credentials"
    ):
        assert load_credentials() == "oauth-credentials"


def test_a_missing_config_still_resolves_to_oauth():
    """gdoc read works with no config.json today, and must keep working."""
    with patch("gdoc.auth.load_config", side_effect=FileNotFoundError("no config")), patch(
        "gdoc.auth.oauth.load", return_value="oauth-credentials"
    ):
        assert load_credentials() == "oauth-credentials"


def test_the_client_is_always_guarded():
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock())
    assert isinstance(build.call_args.kwargs["http"], GuardedHttp)


def test_the_guard_starts_with_the_ids_it_was_given():
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock(), doc_ids=["1AbC", "1DeF"])
    assert build.call_args.kwargs["http"].allowed == frozenset({"1AbC", "1DeF"})


def test_no_ids_means_an_empty_set():
    """generate names no input document. It creates, and learns from that."""
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock())
    assert build.call_args.kwargs["http"].allowed == frozenset()


def test_a_single_id_may_be_passed_as_a_string():
    """Every call site but one has exactly one document, so do not make it wrap."""
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock(), doc_ids="1AbC")
    assert build.call_args.kwargs["http"].allowed == frozenset({"1AbC"})


def test_the_guard_is_installed_without_a_config():
    """Not knowing must never resolve to the unguarded client."""
    with patch("gdoc.auth.load_config", side_effect=FileNotFoundError), patch(
        "gdoc.auth.build"
    ) as build:
        drive_service(credentials=MagicMock())
    assert isinstance(build.call_args.kwargs["http"], GuardedHttp)


def test_discovery_caching_stays_off():
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock())
    assert build.call_args.kwargs["cache_discovery"] is False


def test_credentials_are_not_passed_beside_http():
    """googleapiclient refuses both at once."""
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock())
    assert "credentials" not in build.call_args.kwargs
```

Create `tests/test_guard_is_installed.py`:

```python
"""The guard is worth nothing if a module can build its own client.

An allowlist, in the style of tests/test_no_external_programs.py. It fails if
the call to build() spreads to a second module, and it fails just as loudly if
auth.py stops making it, because that would mean the client is being built
somewhere this test is not looking.
"""

from pathlib import Path

PACKAGE = Path(__file__).resolve().parent.parent / "gdoc"


def _modules_calling_build():
    found = set()
    for path in PACKAGE.rglob("*.py"):
        if "build(" in path.read_text() and "discovery import build" in path.read_text():
            found.add(path.relative_to(PACKAGE).as_posix())
    return found


def test_only_auth_builds_the_drive_client():
    assert _modules_calling_build() == {"auth.py"}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_auth.py tests/test_guard_is_installed.py -v`
Expected: FAIL, `ImportError: cannot import name 'load_service_account_credentials' from 'gdoc.auth'`

- [ ] **Step 3: Write minimal implementation**

Replace `gdoc/auth.py` entirely:

```python
"""Credentials and the Drive client.

One chokepoint, so nothing else in the package has to know which credential is
in use. Two modes:

- service_account, shared onto a document as Commenter. Google itself refuses
  every edit, and the credential reaches only what has been shared with it.
- oauth, authorised as Nail. No sharing step, but the token is Nail and the
  scope is full Drive, so the reachable set is everything Nail owns. There is
  no narrower scope that reads comments and writes replies, so gdoc/guard.py
  narrows it here instead.

The guard is installed in both modes and cannot be switched off. Under the
service account it is nearly redundant, and one code path is worth more than a
saved wrapper.

doc_ids is the set of files a command may touch. It is empty for commands that
have no input document, such as generate, which creates its own and lets the
guard learn the ids from the create responses.
"""

from pathlib import Path

import google_auth_httplib2
import httplib2
from google.oauth2 import service_account
from googleapiclient.discovery import build

from gdoc import oauth
from gdoc.config import DEFAULT_AUTH_MODE, Config, load_config
from gdoc.guard import GuardedHttp

DEFAULT_KEY_PATH = Path.home() / ".config" / "gdoc-agent" / "sa-key.json"
SCOPES = ["https://www.googleapis.com/auth/drive"]
SPEC = "docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md"


def _config():
    """The config, or its defaults.

    A missing config.json is not a failure. gdoc read works without one, and
    every key this module reads has a documented default.
    """
    try:
        return load_config()
    except (FileNotFoundError, ValueError):
        return Config()


def load_service_account_credentials(key_path: Path | None = None):
    key_path = key_path or DEFAULT_KEY_PATH
    if not key_path.exists():
        raise FileNotFoundError(
            f"service account key not found at {key_path}. "
            f"Setup steps are in {SPEC}, section 10."
        )
    return service_account.Credentials.from_service_account_file(
        str(key_path), scopes=SCOPES
    )


def load_credentials(key_path: Path | None = None, mode: str | None = None):
    """Pick a credential. The one place that decides."""
    mode = mode or _config().auth_mode or DEFAULT_AUTH_MODE
    if mode == "oauth":
        return oauth.load(SCOPES)
    return load_service_account_credentials(key_path)


def drive_service(credentials=None, doc_ids=()):
    """Build a Drive v3 client that reaches only doc_ids.

    http is passed instead of credentials, because the guard has to wrap the
    transport and googleapiclient refuses both arguments at once.

    A bare string is accepted, because every caller but generate has exactly
    one document and should not have to wrap it.

    cache_discovery is off because the on-disk discovery cache warns noisily
    under a venv and buys nothing for a tool that runs for a few seconds.
    """
    if isinstance(doc_ids, str):
        doc_ids = (doc_ids,)
    credentials = credentials or load_credentials()
    http = google_auth_httplib2.AuthorizedHttp(credentials, http=httplib2.Http())
    return build(
        "drive", "v3", http=GuardedHttp(http, doc_ids), cache_discovery=False
    )
```

- [ ] **Step 4: Thread the doc id through cli.py**

Four commands know their document before they build a client. Pass it.

| Call site | Change |
|---|---|
| `cmd_read` | `drive_service(doc_ids=doc_id)`, the id is already on the line above |
| `cmd_reply` | `post_reply(drive_service(doc_ids=doc_id), doc_id, ...)` |
| `cmd_export` | `export_markdown(drive_service(doc_ids=doc_id), doc_id)` |
| `cmd_capture` | **move `doc_id = extract_doc_id(args.doc)` above the `drive_service()` line**, then pass it |
| `cmd_generate` | unchanged. It has no input document, and the guard learns from its creates |

- [ ] **Step 5: Run test to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_auth.py tests/test_guard_is_installed.py -v`
Expected: PASS, 14 tests

- [ ] **Step 6: Check nothing else imported the old name**

Run: `grep -rn "load_credentials\|drive_service(" gdoc/ tests/ skills/`
Expected: `gdoc/auth.py`, `gdoc/cli.py`, `tests/test_auth.py`, and `tests/test_access_integration.py`. Every `drive_service()` in `gdoc/cli.py` either passes `doc_ids` or is `cmd_generate`.

- [ ] **Step 7: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS. If `tests/test_access_integration.py` fails, that is Task 10's work, but read the failure first: a guard refusal there means the guard is doing its job and the test still expects Google to do it.

- [ ] **Step 8: Commit**

```bash
git add gdoc/auth.py gdoc/cli.py tests/test_auth.py tests/test_guard_is_installed.py
git commit -m "feat: choose the credential from config, and install the guard

load_credentials dispatches on auth_mode, defaulting to oauth. A missing
config.json resolves to the defaults rather than failing, because gdoc read
works without one today.

drive_service passes http instead of credentials so the guard can wrap the
transport, and takes the file ids the command may touch. read, reply,
export and capture each pass the document they were pointed at. generate
passes none: it has no input document, and the guard learns the ids from
its own create responses.

The guard is always installed and there is no way to turn it off. A new
test keeps auth.py the only module that builds a client, so no future
module can quietly make an unguarded one."
```

---
## Task 7: `gdoc auth` subcommands

**Files:**
- Modify: `gdoc/cli.py`
- Test: `tests/test_cli.py`

**Interfaces:**
- Consumes: `gdoc.oauth.login/logout/account`, `gdoc.auth.drive_service/SCOPES/DEFAULT_KEY_PATH`.
- Produces: `gdoc auth login|status|logout`. Nothing later depends on them.

**Collision:** `gdoc/cli.py`. Add the `auth` subparser at the end of `build_parser`, after the `pair` block. Do not touch `cmd_generate` or the `generate` subparser: that is the other branch's ground.

- [ ] **Step 1: Write the failing test**

Append to `tests/test_cli.py`:

```python
# ---------------------------------------------------------------------------
# auth subcommand
# ---------------------------------------------------------------------------


def test_auth_login_reports_the_account(capsys, tmp_path):
    drive = MagicMock()
    with patch("gdoc.cli.oauth.login", return_value="creds") as login, patch(
        "gdoc.cli.drive_service", return_value=drive
    ), patch(
        "gdoc.cli.oauth.account",
        return_value={"displayName": "Nail Khusnullin", "emailAddress": "nail@altery.com"},
    ):
        exit_code = main(["auth", "login"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["logged_in"] is True
    assert payload["account"] == "nail@altery.com"
    assert login.call_args.args[0] == ["https://www.googleapis.com/auth/drive"]


def test_auth_login_passes_explicit_paths_through(capsys, tmp_path):
    client = tmp_path / "client.json"
    token = tmp_path / "token.json"
    with patch("gdoc.cli.oauth.login", return_value="creds") as login, patch(
        "gdoc.cli.drive_service"
    ), patch("gdoc.cli.oauth.account", return_value={}):
        main(["auth", "login", "--client", str(client), "--token", str(token)])
    assert login.call_args.kwargs["client_path"] == str(client)
    assert login.call_args.kwargs["token_path"] == str(token)


def test_auth_login_reports_a_missing_client_file_as_an_error(capsys, tmp_path):
    client = tmp_path / "client.json"
    with patch("gdoc.cli.oauth.login", side_effect=FileNotFoundError("no client")):
        exit_code = main(["auth", "login", "--client", str(client)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "no client" in payload["error"]


def test_auth_status_says_it_is_ready(capsys):
    with patch("gdoc.cli.load_config", return_value=__import__("gdoc.config", fromlist=["Config"]).Config()), patch(
        "gdoc.cli.drive_service"
    ), patch("gdoc.cli.oauth.account", return_value={"emailAddress": "nail@altery.com"}):
        exit_code = main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["ready"] is True
    assert payload["auth_mode"] == "oauth"
    assert payload["account"] == "nail@altery.com"


def test_auth_status_never_fails_and_says_what_is_wrong(capsys):
    """Its whole job is to report a broken credential, so it must not raise."""
    with patch("gdoc.cli.load_config", side_effect=FileNotFoundError("no config")), patch(
        "gdoc.cli.drive_service", side_effect=FileNotFoundError("no OAuth token at /x")
    ):
        exit_code = main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["ready"] is False
    assert "no OAuth token" in payload["problem"]
    assert payload["auth_mode"] == "oauth"


def test_auth_status_prints_where_to_revoke(capsys):
    with patch("gdoc.cli.load_config", side_effect=FileNotFoundError), patch(
        "gdoc.cli.drive_service", side_effect=RuntimeError("nope")
    ):
        main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert "myaccount.google.com" in payload["revoke_url"]


def test_auth_logout_reports_that_it_removed_a_token(capsys, tmp_path):
    token = tmp_path / "token.json"
    token.write_text("{}")
    exit_code = main(["auth", "logout", "--token", str(token)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["logged_out"] is True
    assert not token.exists()


def test_auth_logout_on_a_machine_with_no_token_says_so(capsys, tmp_path):
    exit_code = main(["auth", "logout", "--token", str(tmp_path / "token.json")])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["logged_out"] is False
```

- [ ] **Step 2: Run test to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -k auth -v`
Expected: FAIL with `SystemExit: 2`, argparse rejecting `invalid choice: 'auth'`

- [ ] **Step 3: Write minimal implementation**

In `gdoc/cli.py`, extend the imports:

```python
from gdoc import oauth
from gdoc.auth import DEFAULT_KEY_PATH, SCOPES, drive_service
```

(the existing line is `from gdoc.auth import drive_service`)

Add the handlers after `cmd_pair_find`, before the parser section:

```python
# ---------------------------------------------------------------------------
# auth subcommand handlers
# ---------------------------------------------------------------------------


def cmd_auth(args) -> int:
    return args.auth_func(args)


def cmd_auth_login(args) -> int:
    credentials = oauth.login(
        SCOPES, client_path=args.client, token_path=args.token
    )
    user = oauth.account(drive_service(credentials=credentials))
    return _emit(
        {
            "logged_in": True,
            "account": user.get("emailAddress"),
            "name": user.get("displayName"),
            "token_path": str(args.token or oauth.DEFAULT_TOKEN_PATH),
        }
    )


def _configured_mode() -> str:
    try:
        return load_config().auth_mode
    except (FileNotFoundError, ValueError):
        return "oauth"


def cmd_auth_status(args) -> int:
    """Say what is set up and what is broken. Never fail.

    The except is deliberately broad. This command exists to report a broken
    credential, so any exception is its output rather than its failure.
    """
    payload = {
        "auth_mode": _configured_mode(),
        "token_path": str(oauth.DEFAULT_TOKEN_PATH),
        "client_path": str(oauth.DEFAULT_CLIENT_PATH),
        "key_path": str(DEFAULT_KEY_PATH),
        "revoke_url": "https://myaccount.google.com/permissions",
    }
    try:
        user = oauth.account(drive_service())
        payload["account"] = user.get("emailAddress")
        payload["name"] = user.get("displayName")
        payload["ready"] = True
    except Exception as error:  # noqa: BLE001
        payload["ready"] = False
        payload["problem"] = str(error)
    return _emit(payload)


def cmd_auth_logout(args) -> int:
    removed = oauth.logout(token_path=args.token)
    return _emit(
        {
            "logged_out": removed,
            "token_path": str(args.token or oauth.DEFAULT_TOKEN_PATH),
        }
    )
```

Add the subparser at the end of `build_parser`, immediately before `return parser`:

```python
    auth = sub.add_parser("auth", help="manage the Google credential")
    auth.set_defaults(func=cmd_auth)
    auth_sub = auth.add_subparsers(dest="auth_command", required=True)

    login_cmd = auth_sub.add_parser("login", help="authorise in a browser as yourself")
    login_cmd.add_argument("--client", help="path to the Desktop OAuth client JSON")
    login_cmd.add_argument("--token", help="where to write the token")
    login_cmd.set_defaults(auth_func=cmd_auth_login)

    status_cmd = auth_sub.add_parser("status", help="show the credential in use")
    status_cmd.set_defaults(auth_func=cmd_auth_status)

    logout_cmd = auth_sub.add_parser("logout", help="delete the local OAuth token")
    logout_cmd.add_argument("--token", help="path to the token to delete")
    logout_cmd.set_defaults(auth_func=cmd_auth_logout)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -k auth -v`
Expected: PASS, 8 tests

- [ ] **Step 5: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add gdoc/cli.py tests/test_cli.py
git commit -m "feat: add gdoc auth login, status and logout

status never fails. Reporting a broken credential is the whole point of it,
so any exception becomes its output instead of its exit code."
```

---

## Task 8: The captured item records who asked

`pending.py` writes the literal line `Nail asked:` and stores no author. Wrong now that any author's marked comment is work, and about to be more wrong when all-comments mode captures unmarked ones.

**Files:**
- Modify: `gdoc/pending.py`
- Test: `tests/test_pending.py`

**Interfaces:**
- Consumes: `gdoc.filters.is_addressed`, `gdoc.filters.forced_kind`.
- Produces: no signature change. `append_item(repo_root, slug, thread, doc_id, today, source=None)` is unchanged; it reads the author and the marker off the `Thread` it already receives. So `cmd_capture` needs no edit, which keeps `cli.py` out of this task.

- [ ] **Step 1: Write the failing test**

Append to `tests/test_pending.py`:

```python
def marked(content):
    return thread(content=content)


def test_the_item_records_who_asked(tmp_path):
    append_item(tmp_path, "policy", thread(), doc_id="1AbC", today=TODAY)
    assert "Author: Nail Khusnullin" in pending_path(tmp_path, "policy").read_text()


def test_an_item_from_someone_else_records_their_name(tmp_path):
    other = Thread(
        id="t9",
        content="ai! renumber the annex",
        author_name="William Mejia",
        author_email=None,
        by_agent=False,
        quoted="Annex 2",
        resolved=False,
        replies=(),
    )
    append_item(tmp_path, "policy", other, doc_id="1AbC", today=TODAY)
    assert "Author: William Mejia" in pending_path(tmp_path, "policy").read_text()


def test_the_item_never_claims_nail_asked(tmp_path):
    """The marker decides, not the author, so the queue must not assume one."""
    append_item(tmp_path, "policy", thread(), doc_id="1AbC", today=TODAY)
    assert "Nail asked" not in pending_path(tmp_path, "policy").read_text()


def test_the_item_records_the_marker_it_carried(tmp_path):
    append_item(tmp_path, "policy", marked("ai! renumber"), doc_id="1AbC", today=TODAY)
    assert "Marked: ai!" in pending_path(tmp_path, "policy").read_text()


def test_a_question_marker_is_recorded_as_such(tmp_path):
    append_item(tmp_path, "policy", marked("ai? is this right"), doc_id="1AbC", today=TODAY)
    assert "Marked: ai?" in pending_path(tmp_path, "policy").read_text()


def test_a_plain_colon_marker_is_recorded_as_such(tmp_path):
    append_item(tmp_path, "policy", marked("ai: rephrase"), doc_id="1AbC", today=TODAY)
    assert "Marked: ai:" in pending_path(tmp_path, "policy").read_text()


def test_an_unmarked_comment_says_so(tmp_path):
    """All-comments mode can capture a comment nobody marked."""
    append_item(tmp_path, "policy", marked("This annex is out of order"), doc_id="1AbC", today=TODAY)
    assert "Marked: no marker" in pending_path(tmp_path, "policy").read_text()


def test_the_comment_text_is_still_quoted(tmp_path):
    append_item(tmp_path, "policy", marked("ai! renumber"), doc_id="1AbC", today=TODAY)
    assert "> ai! renumber" in pending_path(tmp_path, "policy").read_text()


def test_capturing_the_same_comment_twice_is_still_refused(tmp_path):
    append_item(tmp_path, "policy", thread(id="t1"), doc_id="1AbC", today=TODAY)
    with pytest.raises(ValueError, match="already captured"):
        append_item(tmp_path, "policy", thread(id="t1"), doc_id="1AbC", today=TODAY)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_pending.py -v`
Expected: FAIL on `test_the_item_records_who_asked`, because the file has no `Author:` line

- [ ] **Step 3: Write minimal implementation**

In `gdoc/pending.py`, add the import:

```python
from gdoc.filters import forced_kind, is_addressed
```

Add the label helper above `append_item`:

```python
_MARKER_LABELS = {"question": "ai?", "instruction": "ai!"}


def _marked_label(content: str) -> str:
    """What the comment's marker said, for Nail's eyes rather than a rule.

    An all-comments pass can capture a comment nobody marked, and a later
    session should be able to tell that apart from an explicit instruction. Both
    are applied the same way.

    The older @ai form reports as ai:, because it carries no sign.
    """
    if not is_addressed(content):
        return "no marker"
    return _MARKER_LABELS.get(forced_kind(content), "ai:")
```

Replace the `_ITEM` template:

```python
_ITEM = """
## Item {number}

- Captured: {today}
- Comment id: {comment_id}
- Author: {author}
- Anchored to: {anchor}
- Marked: {marked}

The comment:

{request}
"""
```

In `append_item`, pass the two new values into the format call:

```python
    item = _ITEM.format(
        number=number,
        today=today,
        comment_id=thread.id,
        author=thread.author_name,
        anchor=anchor,
        marked=_marked_label(thread.content),
        request=quoted_request,
    )
```

Update the module docstring: one sentence saying the item records the author and the marker, because the marker decides and the author is a label.

- [ ] **Step 4: Run test to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_pending.py -v`
Expected: PASS

- [ ] **Step 5: Check for an import cycle**

Run: `~/.config/gdoc-agent/venv/bin/python -c "import gdoc.cli; print('ok')"`
Expected: `ok`. `pending` imports `filters`, `filters` imports `model`, and neither imports `pending`, so there is no cycle.

- [ ] **Step 6: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add gdoc/pending.py tests/test_pending.py
git commit -m "feat: record the author and the marker on a captured item

The queue said 'Nail asked' and stored no author. Any author's marked comment
is work now, and all-comments mode will capture unmarked ones, so the item
has to say who asked and what the comment carried."
```

---

## Task 9: All-comments mode

**Files:**
- Modify: `gdoc/filters.py:partition`
- Modify: `gdoc/cli.py:_thread_json`, `gdoc/cli.py:cmd_read`, the `read` subparser
- Test: `tests/test_filters.py`, `tests/test_cli.py`

**Interfaces:**
- Consumes: `Thread.has_agent_reply` and `Reply.author_name` from Task 2, `is_addressed` from `filters`.
- Produces: `partition(threads, include_unmarked=False)`, and a `read` payload with `mode`, plus `marked`, `answered` and `replies` on each thread.

**Collision:** `gdoc/cli.py` again. The `read` subparser and `_thread_json` are not touched by the template-merge branch.

- [ ] **Step 1: Write the failing test**

Append to `tests/test_filters.py`:

```python
def test_all_mode_takes_unmarked_comments_too():
    threads = (thread(content="ai: rephrase"), thread(content="This reads oddly"))
    addressed, skipped = partition(threads, include_unmarked=True)
    assert len(addressed) == 2
    assert skipped == ()


def test_all_mode_still_leaves_resolved_threads_alone():
    addressed, skipped = partition((thread(resolved=True),), include_unmarked=True)
    assert addressed == ()
    assert len(skipped) == 1


def test_all_mode_shows_a_thread_gdoc_already_answered():
    """has_agent_reply counts a `me` reply, and under OAuth `me` is Nail.

    Hiding answered threads here would hide every thread he replied to by hand,
    which is the opposite of what all-comments mode is for.
    """
    answered = thread(
        replies=(Reply(id="r1", content="Done.\n\n[gdoc]", by_agent=False, by_marker=True),)
    )
    addressed, _ = partition((answered,), include_unmarked=True)
    assert len(addressed) == 1


def test_default_mode_is_unchanged_by_the_new_argument():
    threads = (thread(content="ai: rephrase"), thread(content="This reads oddly"))
    assert partition(threads) == partition(threads, include_unmarked=False)
```

Append to `tests/test_cli.py`:

```python
def _read_payload(capsys, argv, threads):
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [{"comments": threads}]
    drive.files().get.return_value.execute.return_value = {"name": "Test doc"}
    with patch("gdoc.cli.drive_service", return_value=drive):
        exit_code = main(argv)
    return exit_code, json.loads(capsys.readouterr().out)


_MARKED = {
    "id": "t1",
    "content": "ai: rephrase",
    "author": {"displayName": "Nail Khusnullin", "me": False},
}
_UNMARKED = {
    "id": "t2",
    "content": "This annex reads oddly",
    "author": {"displayName": "William Mejia", "me": False},
}
_ANSWERED = {
    "id": "t3",
    "content": "ai? who owns this",
    "author": {"displayName": "Nail Khusnullin", "me": False},
    "replies": [
        {
            "id": "r1",
            "content": "Compliance owns it.\n\n[gdoc]",
            "author": {"displayName": "Nail Khusnullin", "me": True},
        }
    ],
}


def test_read_reports_the_default_mode(capsys):
    _, payload = _read_payload(capsys, ["read", "https://docs.google.com/document/d/1AbC/edit"], [_MARKED])
    assert payload["mode"] == "marked"


def test_read_all_reports_all_mode(capsys):
    _, payload = _read_payload(
        capsys, ["read", "https://docs.google.com/document/d/1AbC/edit", "--all"], [_MARKED]
    )
    assert payload["mode"] == "all"


def test_read_skips_unmarked_comments_by_default(capsys):
    _, payload = _read_payload(
        capsys, ["read", "https://docs.google.com/document/d/1AbC/edit"], [_MARKED, _UNMARKED]
    )
    assert [t["id"] for t in payload["addressed"]] == ["t1"]
    assert [t["id"] for t in payload["skipped"]] == ["t2"]


def test_read_all_offers_unmarked_comments(capsys):
    _, payload = _read_payload(
        capsys,
        ["read", "https://docs.google.com/document/d/1AbC/edit", "--all"],
        [_MARKED, _UNMARKED],
    )
    assert [t["id"] for t in payload["addressed"]] == ["t1", "t2"]


def test_each_thread_says_whether_it_was_marked(capsys):
    _, payload = _read_payload(
        capsys,
        ["read", "https://docs.google.com/document/d/1AbC/edit", "--all"],
        [_MARKED, _UNMARKED],
    )
    marked = {t["id"]: t["marked"] for t in payload["addressed"]}
    assert marked == {"t1": True, "t2": False}


def test_each_thread_says_whether_it_was_answered(capsys):
    _, payload = _read_payload(
        capsys,
        ["read", "https://docs.google.com/document/d/1AbC/edit", "--all"],
        [_MARKED, _ANSWERED],
    )
    answered = {t["id"]: t["answered"] for t in payload["addressed"]}
    assert answered == {"t1": False, "t3": True}


def test_the_payload_carries_the_existing_replies(capsys):
    _, payload = _read_payload(
        capsys, ["read", "https://docs.google.com/document/d/1AbC/edit", "--all"], [_ANSWERED]
    )
    reply = payload["addressed"][0]["replies"][0]
    assert reply["author"] == "Nail Khusnullin"
    assert reply["by_gdoc"] is True
    assert "Compliance owns it." in reply["content"]


def test_a_thread_with_no_replies_carries_an_empty_list(capsys):
    _, payload = _read_payload(
        capsys, ["read", "https://docs.google.com/document/d/1AbC/edit"], [_MARKED]
    )
    assert payload["addressed"][0]["replies"] == []
```

- [ ] **Step 2: Run test to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_filters.py tests/test_cli.py -k "all_mode or mode or marked or answered or replies" -v`
Expected: FAIL. `partition() got an unexpected keyword argument 'include_unmarked'`, and `KeyError: 'mode'`.

- [ ] **Step 3: Write minimal implementation**

In `gdoc/filters.py`, replace `partition`:

```python
def partition(
    threads: tuple[Thread, ...],
    include_unmarked: bool = False,
) -> tuple[tuple[Thread, ...], tuple[Thread, ...]]:
    """Split into (addressed, skipped).

    By default needs_action holds the whole test. skipped is returned rather
    than discarded so the skill can tell Nail why a comment he can see was
    ignored.

    include_unmarked is all-comments mode. It filters on nothing but resolved:
    unmarked comments are candidates, and so are threads gdoc already answered.
    Dropping answered ones would use has_agent_reply, which counts a `me` reply,
    and under OAuth `me` is Nail. That would hide every thread he replied to by
    hand, which is the opposite of what this mode is for. Nothing is posted from
    this list without Nail picking it.
    """
    addressed: list[Thread] = []
    skipped: list[Thread] = []
    for thread in threads:
        wanted = not thread.resolved if include_unmarked else needs_action(thread)
        (addressed if wanted else skipped).append(thread)
    return tuple(addressed), tuple(skipped)
```

In `gdoc/cli.py`, extend the filters import:

```python
from gdoc.filters import forced_kind, is_addressed, partition
```

Add the preview constant near the top, under the imports:

```python
# Enough of an existing reply to recognise it, not enough to bloat the payload.
_REPLY_PREVIEW = 400
```

Replace `_thread_json`:

```python
def _reply_json(reply) -> dict:
    return {
        "id": reply.id,
        "author": reply.author_name,
        "content": reply.content[:_REPLY_PREVIEW],
        "by_gdoc": reply.by_agent or reply.by_marker,
    }


def _thread_json(thread: Thread) -> dict:
    return {
        "id": thread.id,
        "content": thread.content,
        "author": thread.author_name,
        "quoted": thread.quoted,
        "anchored": thread.is_anchored,
        "forced_kind": forced_kind(thread.content),
        "marked": is_addressed(thread.content),
        "answered": thread.has_agent_reply,
        "replies": [_reply_json(reply) for reply in thread.replies],
    }
```

In `cmd_read`, pass the flag through and report the mode:

```python
def cmd_read(args) -> int:
    doc_id = extract_doc_id(args.url)
    drive = drive_service()
    threads = fetch_threads(drive, doc_id)
    addressed, skipped = partition(threads, include_unmarked=args.all)
    meta = _file_meta(drive, doc_id)
    return _emit(
        {
            "doc_id": doc_id,
            "name": meta.get("name"),
            "slug": slugify(meta.get("name", "")),
            "mode": "all" if args.all else "marked",
            "addressed": [_thread_json(t) for t in addressed],
            "skipped": [_thread_json(t) for t in skipped],
        }
    )
```

In `build_parser`, add the flag to the `read` subparser:

```python
    read.add_argument(
        "--all",
        action="store_true",
        help="offer every unresolved comment, not only the marked ones",
    )
```

- [ ] **Step 4: Run test to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_filters.py tests/test_cli.py -v`
Expected: PASS

- [ ] **Step 5: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add gdoc/filters.py gdoc/cli.py tests/test_filters.py tests/test_cli.py
git commit -m "feat: add gdoc read --all for working through every comment

All-comments mode filters on nothing but resolved. It keeps answered threads
in the list on purpose: has_agent_reply counts a me reply, and under OAuth me
is Nail, so dropping them would hide his own threads.

The payload now says which mode ran, whether each thread was marked and
answered, and what the existing replies say."
```

---

## Task 10: Split the integration suite by mode

**Rewritten 2026-08-15,** with the guard.

`tests/test_access_integration.py` asserts Google refuses every write. Under a service account that is still exactly right. Under OAuth it is no longer what the design promises: an edit to the named document now reaches Google and Google permits it. What the guard promises instead is a boundary, so that is what the OAuth cases assert.

**Read this before writing a line of it.** Under OAuth, an attempted edit on the test document is a real edit on a real document. No test in this file may attempt one. The old plan had `test_lifting_the_guard_really_lifts_it` rename the document and rename it back; that test is deleted along with the flag it tested, and nothing replaces it. What the OAuth cases probe is what is *refused*, which costs nothing when the refusal works and, when it fails, fails inside the process before a request is sent.

**Files:**
- Modify: `tests/conftest.py`
- Modify: `tests/test_access_integration.py`

**Interfaces:**
- Consumes: `gdoc.auth.drive_service`, `gdoc.config.load_config`.
- Produces: an `auth_mode` fixture, a `drive` fixture scoped to the test document, and two skip markers. Nothing else depends on them.

- [ ] **Step 1: Add the fixtures**

Replace `tests/conftest.py`:

```python
import pytest

TEST_DOC_ID = "1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw"

# A real Drive id shape that is not the test document. Every request to it must
# be refused inside the process, so it is never fetched and never has to exist.
OTHER_DOC_ID = "1NotTheDocumentTheseTestsWereGiven00000000"


@pytest.fixture(scope="session")
def auth_mode():
    """The configured mode, or the default when there is no config."""
    from gdoc.config import Config, load_config

    try:
        return load_config().auth_mode
    except (FileNotFoundError, ValueError):
        return Config().auth_mode


@pytest.fixture(scope="session")
def drive():
    """A client pointed at the test document, the way the CLI points one."""
    from gdoc.auth import drive_service

    return drive_service(doc_ids=TEST_DOC_ID)


@pytest.fixture(scope="session")
def test_doc_id():
    return TEST_DOC_ID


@pytest.fixture(scope="session")
def other_doc_id():
    return OTHER_DOC_ID


@pytest.fixture
def service_account_only(auth_mode):
    if auth_mode != "service_account":
        pytest.skip("asserts what Google refuses under Commenter")


@pytest.fixture
def oauth_only(auth_mode):
    if auth_mode != "oauth":
        pytest.skip("asserts what the guard refuses under OAuth")
```

- [ ] **Step 2: Write the failing test**

Replace the module docstring of `tests/test_access_integration.py`:

```python
"""What the credential is allowed to do, asserted against the live API.

These assert on what is permitted, not on what the agent chooses to do. A test
that only proved the agent did not try would prove nothing.

What is guaranteed depends on the credential, so the file splits by mode.

- service_account: Google refuses every write. Commenter access cannot edit,
  rename, or delete, and the credential reaches only documents shared with it.
- oauth: Google refuses nothing, because the token is Nail and the scope is full
  Drive. gdoc/guard.py provides a different guarantee: the client reaches only
  the file it was built for. So the OAuth cases assert the boundary, not the
  verb.

No test here attempts an edit on the test document. Under service_account Google
would refuse it; under oauth it would succeed, and succeeding means editing a
real document. Never add one.

Never loosen an assertion to suit the configured mode. Add the other mode's case.
"""
```

Add the `service_account_only` fixture to every test that asserts a Google refusal, so their signatures become:

```python
def test_google_says_the_agent_cannot_edit(drive, test_doc_id, service_account_only):
def test_renaming_the_file_is_refused(drive, test_doc_id, service_account_only):
def test_permissions_list_is_refused(drive, test_doc_id, service_account_only):
def test_editing_the_text_is_refused(drive, test_doc_id, service_account_only):
def test_deleting_a_comment_is_refused(drive, test_doc_id, service_account_only):
```

Every one of those attempts a write on the test document, and under OAuth that write would land. They belong to the service account mode and nowhere else.

Then append the OAuth cases:

```python
# ---------------------------------------------------------------------------
# Under OAuth the guarantee is the boundary, not the verb
# ---------------------------------------------------------------------------


def test_reading_a_document_we_were_not_given_is_refused(
    drive, other_doc_id, oauth_only
):
    """The failure the first draft's verb allowlist could not see.

    OAuth reaches every file Nail owns, so a read is not harmless. This is the
    guarantee that replaces Commenter.
    """
    with pytest.raises(PermissionError) as excinfo:
        drive.files().get(fileId=other_doc_id, fields="name").execute()
    assert other_doc_id in str(excinfo.value)


def test_listing_drive_is_refused(drive, oauth_only):
    """gdoc never searches Drive, and under OAuth it must not be able to."""
    with pytest.raises(PermissionError):
        drive.files().list(pageSize=1, fields="files(id)").execute()


def test_reading_someone_elses_comments_is_refused(drive, other_doc_id, oauth_only):
    with pytest.raises(PermissionError):
        drive.comments().list(fileId=other_doc_id, fields="comments(id)").execute()


def test_sharing_the_named_document_is_refused(drive, test_doc_id, oauth_only):
    """Allowed file, refused anyway. Granting access is a different authority.

    Refused inside the process, so nothing is ever shared with anyone. If this
    test starts failing, the next run of it publishes a real document to the
    whole internet.
    """
    with pytest.raises(PermissionError):
        drive.permissions().create(
            fileId=test_doc_id, body={"role": "reader", "type": "anyone"}
        ).execute()


def test_the_docs_api_is_bounded_by_the_same_set(drive, other_doc_id, oauth_only):
    """The guard was never told the Docs API exists. It shares the transport."""
    from googleapiclient.discovery import build

    docs = build("docs", "v1", http=drive._http, cache_discovery=False)
    with pytest.raises(PermissionError):
        docs.documents().batchUpdate(
            documentId=other_doc_id,
            body={"requests": [{"insertText": {"location": {"index": 1}, "text": "PROBE"}}]},
        ).execute()


def test_reading_and_commenting_still_work_under_the_guard(
    drive, test_doc_id, oauth_only
):
    """The guard must not break what the tool is for."""
    meta = drive.files().get(fileId=test_doc_id, fields="id").execute()
    assert meta["id"] == test_doc_id
    res = drive.comments().list(fileId=test_doc_id, fields="comments(id)").execute()
    assert len(res.get("comments", [])) >= 3
```

- [ ] **Step 3: Run the integration suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_access_integration.py -v -m integration`

Expected: with `auth_mode` still `service_account` in Nail's live config, the service account tests run and pass, and the six `oauth_only` tests skip with a reason. Nothing fails.

If the live config has no `auth_mode` key, the fixture reports `oauth`, the OAuth cases run, and they need a token. If there is no token yet, the whole file errors on the `drive` fixture. That is expected before Task 11's setup, and it is why this task is late in the order. Record the result either way and move on.

**Note on `test_google_says_the_agent_cannot_edit`:** it may already be building its Docs client from raw credentials rather than from `drive._http`. Leave it. Under `service_account_only` the enforcer is Google, and how the client was built does not change what Google answers.

- [ ] **Step 4: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS, with skips reported

- [ ] **Step 5: Commit**

```bash
git add tests/conftest.py tests/test_access_integration.py
git commit -m "test: split the access suite by what each mode guarantees

Under a service account Google refuses every write, and those assertions
are unchanged. Under OAuth nothing external refuses, so the guarantee is a
different one and the tests say so: a document the client was not built for
is unreachable, files.list is refused, and sharing is refused even on the
allowed file.

No test attempts an edit on the test document. Under OAuth that would
succeed, and succeeding means editing a real document."
```

---
## Task 11: The documents, install.sh, and the skills

Nothing here changes behaviour, and all of it is load-bearing: the README's central claim is now mode-dependent, and both skills tell Claude things that stopped being true.

**Files:**
- Modify: `install.sh`
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `skills/gdoc-review/SKILL.md`
- Modify: `skills/gdoc-apply/SKILL.md`

**Interfaces:**
- Consumes: everything above.
- Produces: nothing code depends on. The skills are symlinked into `~/.claude/skills/`, so an edit here is live in every Claude Code session the moment it is saved, before it is committed.

- [ ] **Step 1: install.sh, the credential check**

Replace the credentials block in `install.sh`:

```bash
# --------------------------------------------------------------------------
# Credentials, checked but never written
# --------------------------------------------------------------------------

[ -f "$CONFIG_DIR/config.json" ] || warn "no $CONFIG_DIR/config.json yet. See README.md"

# Which credential to check for depends on auth_mode, which defaults to oauth.
auth_mode="$("$VENV/bin/python" - "$CONFIG_DIR/config.json" <<'PY' 2>/dev/null || echo oauth
import json
import sys

try:
    print(json.load(open(sys.argv[1])).get("auth_mode") or "oauth")
except Exception:
    print("oauth")
PY
)"

case "$auth_mode" in
    service_account)
        [ -f "$CONFIG_DIR/sa-key.json" ] || warn "no $CONFIG_DIR/sa-key.json yet. See README.md"
        ;;
    *)
        if [ ! -f "$CONFIG_DIR/oauth-client.json" ]; then
            warn "no $CONFIG_DIR/oauth-client.json yet. See README.md, Configure"
        elif [ ! -f "$CONFIG_DIR/oauth-token.json" ]; then
            # Not a fault. It is the next step.
            printf 'install: next step: gdoc auth login\n'
        fi
        ;;
esac
```

- [ ] **Step 2: Run install.sh and read what it says**

Run: `./install.sh`
Expected: it completes, prints the commit, and prints the warning or the next-step line matching the live config's `auth_mode`. It must not fail. Run it twice to confirm it is still safe to re-run.

- [ ] **Step 3: README.md**

Replace the second paragraph, the one beginning "The credential is a service account with **Commenter** access":

```markdown
There are two credentials, and `auth_mode` in the config picks one.

**`oauth`** (the default) authorises you in a browser. No document has to be
shared with anything first: the agent reaches whatever you can reach. Drive has
no scope that reads comments and writes replies without full Drive access, so
this credential holds more than the tool needs. `gdoc/guard.py` narrows it back
down. Every request goes through it, and it carries a request only when the file
the request addresses is one gdoc was given or one gdoc created. Anything else
is refused inside the process, a read included, so the tool cannot see a
document you did not point it at and cannot search your Drive at all.

**`service_account`** is the original. Give the service account address
**Commenter** access on a document, and Google itself refuses every edit. No
browser, no token to refresh, one sharing step per document.

The difference worth knowing: under `service_account` the tool *cannot* edit a
reviewed document, because Google will not let it. Under `oauth` it *does not*,
because no code in it does. The guard bounds which files are reachable; it does
not bound what happens inside one.

Either way the shape of the tool is the same. Replies go in comment threads, and
document-wide changes are applied to a paired markdown file and published as a
new version.
```

In the `## Configure` section, replace the file list:

```markdown
Files outside this repo, none of them in git:

- `~/.config/gdoc-agent/config.json`: settings, including `auth_mode` and the
  Drive folder new versions are written to.
- `~/.config/gdoc-agent/oauth-client.json`: a Desktop OAuth client from Google
  Cloud. Needed by `auth_mode: oauth`.
- `~/.config/gdoc-agent/oauth-token.json`: written by `gdoc auth login`, mode
  `0600`. Never edit it by hand.
- `~/.config/gdoc-agent/sa-key.json`: the service account key. Needed by
  `auth_mode: service_account`.

While you are in that section, replace the em dashes in the surrounding README
prose with a colon or a full stop. Nail's writing rules forbid the character and
that file predates the rule.

### Signing in with OAuth

Once, in Google Cloud, in the same project as the service account:

1. Keep the Drive API enabled. It is the only API the package calls.
2. OAuth consent screen, User type **Internal**. On External plus Testing,
   Google expires refresh tokens after seven days and you would sign in weekly.
3. Credentials, OAuth client ID, Application type **Desktop app**. Save the JSON
   to `~/.config/gdoc-agent/oauth-client.json`.

Then:

```bash
gdoc auth login     # a browser opens, approve
gdoc auth status    # confirm the account
gdoc auth logout    # delete the local token
```

`auth status` never fails. Reporting a broken credential is its job.
```

In the `## Use` section, add `--all` under the direct CLI list:

```
gdoc read <url> --all                   # every unresolved comment, not only marked ones
```

- [ ] **Step 4: PRINCIPLES.md**

The three principles are untouched. One dated decision is retired and one is
added. Retired entries stay, marked retired, with the reason, which is the
file's own rule.

Replace the `**2026-08-13. The credential is Commenter-only.**` entry with:

```markdown
**2026-08-13. The credential is Commenter-only.** *Retired 2026-08-15.* It
enforced the decision above by permission rather than by discipline, and it was
Google's to enforce. OAuth has no scope that reads comments and writes replies
without full Drive access, so keeping this would have meant keeping the sharing
step forever. Replaced by the decision below.

**2026-08-15. The client reaches only the files it was given.** Serves principle
3. Under OAuth the credential can reach every file Nail owns, so `gdoc/guard.py`
holds a set of file ids, the ones the command was handed plus the ones its own
creates returned, and refuses every request addressing anything else. An empty
set refuses everything, so a command that does not name a document reaches
nothing.

The cost is stated rather than hidden: on a file in the set, every method is
permitted, including an edit to a reviewed document. "The agent cannot edit a
reviewed document" becomes "it does not". Nothing in gdoc edits one, and the
markdown-is-the-source decision above is now held up by design rather than by
permission. `tests/test_guard.py` and `tests/test_guard_is_installed.py` are
what keep it honest.
```

Then, in the `### Open violations` section, delete the entry about
`gdoc/export.py` hardcoding pandoc's path if `main` has already fixed it. Check
first with `grep -n "opt/homebrew" gdoc/export.py`; the parallel branch it names
merged on 2026-08-15.

- [ ] **Step 5: CLAUDE.md**

In the `## Never` section, replace the first bullet:

```markdown
- Never edit a reviewed Google Doc. Under `service_account` the credential
  cannot. Under `oauth` it could: `gdoc/guard.py` bounds which files are
  reachable, not what may be done inside one. Nothing in gdoc edits a document,
  and nothing may start.
```

Add to the `## What lives where` table, after the `gdoc/` row:

```markdown
| `gdoc/guard.py` | the reachable set. Which files a client may touch, under either credential |
| `gdoc/marker.py` | the `[gdoc]` label on gdoc's own replies |
```

Add a new section after "No external programs":

```markdown
## The client reaches only the files it was given

Principle 3, and the decision dated 2026-08-15. Read it there.

In code: `drive_service(doc_ids=...)` wraps the transport in
`gdoc.guard.GuardedHttp`, which carries a request only when the file it
addresses is in its set. The set is seeded from the CLI, where `read`, `reply`,
`export` and `capture` each pass the document they were pointed at, and grows
only when a create the guard itself carried comes back with an id. `generate`
starts with an empty set and works entirely off that second door.

Two rules an agent is likely to break:

- The set has exactly two doors. Never add a third, and never widen it to make
  a test pass. A refused call means the command did not say which document it
  was for.
- `gdoc/auth.py` is the only module that may call `build()`.
  `tests/test_guard_is_installed.py` enforces it, as an allowlist: it fails if
  the call spreads, and it fails if it moves.
```

Add a new section after "The tool must work without git":

```markdown
## Identity is never a gate

Drive's `author.me` means the service account under `auth_mode: service_account`
and Nail under `oauth`. Nothing may branch on it to decide whether a comment is
work: that would skip every comment Nail writes. The marker decides.

`[gdoc]` on the last line of a reply is how gdoc recognises its own replies.
`Reply.by_agent`, which is `me`, is kept in `has_agent_reply` for one reason
only: threads the service account answered before the marker existed carry no
marker, and dropping it would answer them twice.
```

- [ ] **Step 6: skills/gdoc-review/SKILL.md**

Four edits.

Replace the Step 1 paragraph that says "Threads already answered by the service account appear under `skipped`":

```markdown
Returns `addressed` and `skipped`. A marked comment from anyone in the document
is addressed: the marker is the instruction, and Nail chose the document. Each
item carries its `author`, so an unexpected name is visible. Threads gdoc has
already answered appear under `skipped`, which is what makes a second run safe.

Every reply gdoc posts ends with `[gdoc]` on its own line. That marker, not the
account name, is how a later run knows the thread was answered. Under `oauth`
replies post under Nail's own name, so the marker is the only signal that works.

One case to expect once: the first `oauth` run on a document the service account
reviewed earlier can list threads that were in fact answered. Those old replies
carry no marker and no longer read as gdoc's. Each item's `replies` field shows
what is already there, so say so and let Nail decide.
```

Replace the warning inside the Step 2 example, which currently says "under the service account address":

```
I cannot see who else has access to this document. Replies will be
visible to everyone on it. Under auth_mode oauth they post under your
own name; under service_account they post under the service account.
```

Add a new section between Step 1 and Step 2:

```markdown
## If Nail asks for all the comments

By default only marked comments are work. When Nail says he wants to go through
every comment, add `--all`:

```bash
$GDOC read <url> --all
```

`addressed` then holds every unresolved thread, marked or not, including ones
that already have an answer. Each item says `marked`, `answered`, and carries its
existing `replies`.

This mode never batch-approves. Print the numbered list with author, quote and
content, answered threads last and labelled, and let Nail pick. For each comment
he picks: draft the reply, show it in the terminal, wait, then post. Unmarked
comments carry no `forced_kind`, so classify local against global yourself and
say which you chose.

Picking an answered thread is allowed. It is the one case where you reply twice
to the same comment, and you say so before posting.
```

Replace two lines in the `## Never` list:

```markdown
- Never act on an unmarked comment unless Nail asked for all-comments mode and
  picked that comment.
- Never reply twice to the same comment, unless Nail picked an answered thread in
  all-comments mode, and say so before posting.
```

Replace the first `Never` line, which says the credential cannot edit:

```markdown
- Never edit the reviewed document. Under service_account the credential cannot.
  Under oauth gdoc's own guard refuses, and neither may you.
```

- [ ] **Step 7: skills/gdoc-apply/SKILL.md**

One edit. This skill does not learn to read comments here. That is
`2026-08-14-gdoc-apply-drains-design.md` and it lands after this branch.

In Step 2, replace the first numbered item:

```markdown
1. Say what you are about to change and where, and name the item's `Author:`, so
   Nail knows whose feedback he is approving. An older queue has no `Author:`
   line; say the author is not recorded rather than guessing.
```

- [ ] **Step 8: Verify the docs against the code**

Run: `grep -rn "service account address\|the credential cannot\|Nail asked" README.md CLAUDE.md PRINCIPLES.md skills/`
Expected: no hits except the deliberate contrast phrasing inside the new README, PRINCIPLES.md and CLAUDE.md text. Any other hit is a stale claim, so fix it.

Run: `grep -rn "Commenter-only\|cannot edit" README.md CLAUDE.md PRINCIPLES.md`
Expected: every hit either sits under the retired 2026-08-13 decision or is scoped to `service_account`. An unqualified claim that the tool cannot edit a document is now false and must be fixed.

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add install.sh README.md CLAUDE.md PRINCIPLES.md skills/gdoc-review/SKILL.md skills/gdoc-apply/SKILL.md
git commit -m "docs: describe both credentials, and what each one guarantees

The README's central claim was that the credential cannot edit a document.
Under oauth that stops being true, and saying it anyway would be the worst
outcome. What holds instead is the reachable set: the client sees only the
file it was given. The README, CLAUDE.md and PRINCIPLES.md all say so, and
the Commenter-only decision is retired rather than quietly dropped.

Both skills gain what changed: the [gdoc] marker, all-comments mode, and the
author line gdoc-apply now shows when it presents an item.

install.sh now checks for the credential the configured mode needs."
```

---

## Task 12: Final verification

**Files:** none. This task only runs things.

- [ ] **Step 1: Full suite with coverage**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q --cov=gdoc --cov-report=term-missing`
Expected: PASS, total coverage at or above 80 percent. If `pytest-cov` is not installed, run `~/.config/gdoc-agent/venv/bin/pip install -q pytest-cov` first.

- [ ] **Step 2: Confirm the guard cannot be bypassed by an import**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_guard_is_installed.py -v`
Expected: PASS. `gdoc/auth.py` is the only module calling `build()`.

Run: `grep -rn "credentials=" gdoc/`
Expected: only `gdoc/auth.py`, inside `load_service_account_credentials` and `drive_service`'s own parameter. If any other module builds a client with `credentials=`, it bypasses the guard. Route it through `drive_service`.

Run: `grep -rn "drive_service(" gdoc/cli.py`
Expected: every call passes `doc_ids`, except `cmd_generate`, which has no input document and learns its ids from its own creates.

- [ ] **Step 3: Confirm identity is not a gate anywhere**

Run: `grep -rn "by_agent" gdoc/`
Expected: `gdoc/model.py` only, in the `Reply` field, `parse_thread`, and `has_agent_reply`. Any use in `filters.py` or `cli.py` deciding what is work is the bug this plan exists to remove.

- [ ] **Step 4: Smoke test the CLI without credentials**

Run: `~/.config/gdoc-agent/venv/bin/gdoc auth status`
Expected: a JSON object, exit code 0, with `ready` either true or false and a `problem` string if false. It must never traceback.

Run: `~/.config/gdoc-agent/venv/bin/gdoc read --help`
Expected: the help text lists `--all`.

- [ ] **Step 5: Report to Nail**

Say which mode the live config is in, whether the integration suite ran or skipped, and the coverage number. Do not claim the OAuth path works end to end until `gdoc auth login` has actually run against a real client file.

Say plainly that the cannot-edit promise changed shape, and that `PRINCIPLES.md` records it. That is the part of this branch Nail has to agree with, and it must not arrive buried in a list of passing tests.

---

## Self-Review

**Spec coverage.** Every section maps to a task:

| Spec section | Task |
|---|---|
| 4, credentials, `oauth.py`, `auth.py` dispatch | 5, 6 |
| 5, the guard | 4, wired into auth and the CLI in 6, proven in 10 |
| 6, the marker and identity | 1, 2 |
| 7, all-comments mode | 9, skill in 11 |
| 8, what reaches gdoc-apply, the review-then-apply check, the item format | 8, skill in 11 |
| 9, setup | documented in 11, done by Nail |
| 10, CLI, config, dependencies, install.sh, docs | 3, 5, 7, 9, 11 |
| 11, out of scope | nothing built |
| 12, testing | every task, plus 12 |
| 13, order of work | this plan's task order |

**Two deviations from the spec, both deliberate:**

1. `account()` takes the built Drive client, not credentials, so nothing builds a second client to ask who is signed in.
2. The `[gdoc]` marker lives in a new `gdoc/marker.py` rather than in `reply.py`. Both `model.py` and `reply.py` need it, and a value module importing a write module would be the wrong direction.

**One thing the spec asked for that this plan makes cheaper.** The spec wanted the review-then-apply check to compare against `pending.md`. That is entirely inside `skills/gdoc-apply/SKILL.md`, with no new CLI command, because `gdoc read` already returns what is needed.

**Type consistency, checked:**

- `with_marker` and `has_marker` are defined in Task 1 and used in Tasks 2 and 9. Same names throughout.
- `Reply(id, content, by_agent, author_name="unknown", by_marker=False)` is defined in Task 2 and constructed with those keywords in Tasks 2 and 9's tests, and read by `_reply_json` in Task 9.
- `partition(threads, include_unmarked=False)` is defined in Task 9 and called with that keyword in Task 9's tests only.
- `oauth.load(scopes, token_path=None)` is defined in Task 5 and called as `oauth.load(SCOPES)` in Task 6.
- `oauth.login(scopes, client_path=None, token_path=None)` is defined in Task 5 and called with those keywords in Task 7.
- `drive_service(credentials=None, doc_ids=())` is defined in Task 6, called with `doc_ids=` from four places in `cli.py`, and used by the `drive` fixture in Task 10.
- `guard.verdict(method, uri, allowed)` returns `CARRY`, `LEARN` or `REFUSE`, all three defined in Task 4 and imported by that task's tests only. `GuardedHttp(inner, allowed)` exposes `.allowed` as a frozenset, read by Task 6's tests.
- `DEFAULT_AUTH_MODE` is defined in Task 3 and imported by Task 6.
- `load_service_account_credentials` is introduced in Task 6, and Task 6's step 5 checks no other caller expects the old behaviour of `load_credentials`.

**Order dependency worth respecting.** Task 2 imports `gdoc.marker`, which Task 1 creates. Task 6 imports `gdoc.oauth` and `gdoc.guard`, which Tasks 5 and 4 create, and `DEFAULT_AUTH_MODE` from Task 3. Task 10 needs Task 6's `drive_service` signature. Do the tasks in order.
