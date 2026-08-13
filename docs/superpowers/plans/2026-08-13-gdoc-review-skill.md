# Google Docs Review Skill Implementation Plan

> **Status: complete.** This plan was executed inside `intelligence-hub`, where the
> package lived at `tools/gdoc/` and imports read `tools.gdoc.*`. The tool now has
> its own repo and the package sits at `gdoc/` with imports `gdoc.*`. Read the paths
> below as history, not as instructions.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a global Claude Code skill that reads the `ai:` comments Nail leaves in a Google Doc, answers the local ones in their threads, and captures the global ones for a later terminal session.

**Architecture:** A Python package in this repo does every Google API call and every file conversion, exposing one CLI with JSON output. A global skill file holds the procedure and the judgement: which comments are local, which are global, and what the replies say. The split matters because the Python is testable and the judgement is not, so all the untestable work lives in prose and all the testable work lives behind `pytest`.

**Tech Stack:** Python 3.12, `google-api-python-client`, `google-auth`, `PyYAML`, `pytest`, pandoc 3.3, Drive API v3.

**Spec:** `docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md` (commit `8e0a866`)

## Global Constraints

Every task's requirements implicitly include this section. Values are copied verbatim from the spec.

- **Service account:** `nail-ai@doc-agent-505414.iam.gserviceaccount.com`
- **Key path:** `~/.config/gdoc-agent/sa-key.json`, mode 600, never in a repo
- **Scope:** `https://www.googleapis.com/auth/drive`
- **Interpreter:** `~/.config/gdoc-agent/venv/bin/python`. Never the system python
- **pandoc:** `/opt/homebrew/bin/pandoc`, version 3.3
- **Role on reviewed documents:** Commenter, never Editor. Write rights exist only on the output folder
- **Output folder:** `1w0SresizE9Kr810VZRJwX4JtDBF4OqNr`, named "test folder", on shared drive `0AA2s2yLSmVTFUk9PVA`. The service account has `canAddChildren`
- **Marker:** the comment must start with `ai:`, `ai?` or `ai!`. `ai?` forces question, `ai!` forces instruction, `ai:` lets the agent decide. `@ai` is still accepted, with or without the sign, because Nail used it first and a silently ignored comment is a bad failure. The punctuation is required on the bare form, so a comment that merely starts with "AI is..." is not a request
- **Replies are plain text.** Docs comment threads do not render markdown
- **Paste-ready text comes first** in any reply proposing wording
- **Never resolve a thread.** Nail resolves, because resolving means the text was accepted
- **Never edit the reviewed document.** Global changes produce a new document
- **Mirror only documents that have a paired markdown file.** A document Nail does not own is never mirrored into a repo
- **No local database.** Two committed files per document, nothing else
- **Immutability:** frozen dataclasses and new objects. Never mutate in place
- **Code layout:** many small files, 200 to 400 lines typical
- **Test coverage:** 80% minimum on the Python package
- **Writing style in all user-facing strings:** plain English, short sentences, no em dashes

**Verified already, do not re-derive** (spec section 10, "Verified on 2026-08-13"): `files.get`, `comments.list` and `replies.create` all work. `quotedFileContent` arrives on anchored comments. `author.emailAddress` is `None` on every comment, so an email allowlist is impossible. `author.me` is reliable.

**Verified on 2026-08-13 after the share was downgraded to Commenter:**

- `capabilities.canEdit` on the reviewed document is `false`, `canComment` is `true`
- `replies.create` still works, and the agent can delete its own reply
- `permissions.list` is **refused with 403**. The agent cannot see who else has access, so the external-audience check belongs to Nail, not to the code
- Both candidate output folders report `canAddChildren: true`. Folder `1w0Sresiz...` is on a shared drive, folder `1KmUrVKL...` is in My Drive. The shared drive one is chosen, because files created there belong to the drive rather than to the service account
- Still not proven: that a write is refused. See Task 3

**Test document:** `1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw`

## File Structure

| Path | Responsibility |
|---|---|
| `tools/gdoc/config.py` | Load `~/.config/gdoc-agent/config.json` |
| `tools/gdoc/docid.py` | Turn a URL or bare id into a document id |
| `tools/gdoc/auth.py` | Credentials and the Drive service factory |
| `tools/gdoc/model.py` | `Thread` and `Reply` frozen dataclasses, and parsing |
| `tools/gdoc/fetch.py` | Paged `comments.list` |
| `tools/gdoc/filters.py` | Marker match, mine versus others, already-answered |
| `tools/gdoc/reply.py` | Plain-text guard and `replies.create` |
| `tools/gdoc/export.py` | Document to markdown, with a pandoc fallback |
| `tools/gdoc/pairing.py` | Frontmatter read and write |
| `tools/gdoc/mirror.py` | Write the mirror, refusing over uncommitted changes |
| `tools/gdoc/pending.py` | Append captured global items |
| `tools/gdoc/generate.py` | markdown to docx to Drive, with a local fallback |
| `tools/gdoc/cli.py` | argparse dispatch, JSON on stdout |
| `tools/gdoc/tests/` | pytest, one test module per source module |
| `~/.claude/skills/gdoc-review/SKILL.md` | The review procedure and the judgement rules |
| `~/.claude/skills/gdoc-apply/SKILL.md` | The next-session procedure for global items |

Why the package is in the repo and the skill is not: the package needs tests and git history, and `~/.claude` gives neither. The skill file is prose, changes rarely, and must be global to work from any cwd.

---

### Task 1: Package skeleton, dependencies, and document id parsing

**Files:**
- Create: `tools/gdoc/__init__.py`
- Create: `tools/gdoc/requirements.txt`
- Create: `tools/gdoc/docid.py`
- Create: `tools/gdoc/tests/__init__.py`
- Create: `tools/gdoc/tests/test_docid.py`

**Interfaces:**
- Consumes: nothing
- Produces: `extract_doc_id(url_or_id: str) -> str`, raising `ValueError` on anything unrecognised

- [ ] **Step 1: Install the two remaining dependencies**

The venv already holds `google-auth` and `google-api-python-client`. The connection is slow, roughly 35 KB/s, so expect this to take a few minutes and do not pipe it through `tail`.

```bash
~/.config/gdoc-agent/venv/bin/pip install --timeout 180 --retries 10 PyYAML pytest
```

- [ ] **Step 2: Write `tools/gdoc/requirements.txt`**

```
google-auth
google-api-python-client
PyYAML
pytest
```

- [ ] **Step 3: Write the failing test**

`tools/gdoc/tests/test_docid.py`:

```python
import pytest

from tools.gdoc.docid import extract_doc_id

DOC_ID = "1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw"


def test_extracts_id_from_edit_url_with_tab_fragment():
    url = f"https://docs.google.com/document/d/{DOC_ID}/edit?tab=t.0"
    assert extract_doc_id(url) == DOC_ID


def test_extracts_id_from_url_with_heading_anchor():
    url = f"https://docs.google.com/document/d/{DOC_ID}/edit#heading=h.abc123"
    assert extract_doc_id(url) == DOC_ID


def test_extracts_id_from_bare_url_without_suffix():
    assert extract_doc_id(f"https://docs.google.com/document/d/{DOC_ID}") == DOC_ID


def test_accepts_a_bare_document_id():
    assert extract_doc_id(DOC_ID) == DOC_ID


def test_strips_surrounding_whitespace():
    assert extract_doc_id(f"  {DOC_ID}  ") == DOC_ID


def test_raises_on_a_non_docs_url():
    with pytest.raises(ValueError, match="not a Google Docs URL"):
        extract_doc_id("https://example.com/page")


def test_raises_on_empty_input():
    with pytest.raises(ValueError):
        extract_doc_id("")
```

- [ ] **Step 4: Run it and confirm it fails**

Run: `cd ~/src/altery/intelligence-hub && ~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_docid.py -v`
Expected: FAIL, `ModuleNotFoundError: No module named 'tools.gdoc.docid'`

- [ ] **Step 5: Write the implementation**

`tools/gdoc/__init__.py` and `tools/gdoc/tests/__init__.py` are both empty files.

`tools/gdoc/docid.py`:

```python
"""Turn whatever Nail pastes into a Drive document id."""

import re

_URL_PATTERNS = (
    re.compile(r"/document/d/([a-zA-Z0-9_-]+)"),
    re.compile(r"[?&]id=([a-zA-Z0-9_-]+)"),
)

_BARE_ID = re.compile(r"[a-zA-Z0-9_-]{20,}")


def extract_doc_id(url_or_id: str) -> str:
    """Accept a Docs URL in any shape, or a bare document id.

    Raises ValueError rather than returning None, because every caller needs
    an id and a silent None would surface later as a confusing 404.
    """
    text = (url_or_id or "").strip()
    for pattern in _URL_PATTERNS:
        match = pattern.search(text)
        if match:
            return match.group(1)
    if _BARE_ID.fullmatch(text):
        return text
    raise ValueError(f"not a Google Docs URL or document id: {url_or_id!r}")
```

- [ ] **Step 6: Run it and confirm it passes**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_docid.py -v`
Expected: 7 passed

- [ ] **Step 7: Commit**

```bash
git add tools/gdoc/
git commit -m "feat: gdoc package skeleton and document id parsing"
```

---

### Task 2: Config and the Drive service factory

**Files:**
- Create: `tools/gdoc/config.py`
- Create: `tools/gdoc/auth.py`
- Create: `tools/gdoc/tests/test_config.py`
- Create: `tools/gdoc/tests/test_auth.py`

**Interfaces:**
- Consumes: nothing
- Produces: `load_config(path: Path | None = None) -> Config` where `Config` is a frozen dataclass with `display_name: str` and `output_folder_id: str | None`; `load_credentials(key_path: Path | None = None)`; `drive_service(credentials=None)`

- [ ] **Step 1: Write the failing tests**

`tools/gdoc/tests/test_config.py`:

```python
import json

import pytest

from tools.gdoc.config import Config, load_config


def test_loads_display_name_and_folder(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Test User", "output_folder_id": "0AFolderId"}))
    config = load_config(path)
    assert config == Config(display_name="Test User", output_folder_id="0AFolderId")


def test_output_folder_may_be_absent(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Test User"}))
    assert load_config(path).output_folder_id is None


def test_missing_file_names_the_path_and_the_required_key(tmp_path):
    path = tmp_path / "config.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        load_config(path)
    assert str(path) in str(excinfo.value)
    assert "display_name" in str(excinfo.value)


def test_missing_display_name_is_rejected(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    with pytest.raises(ValueError, match="display_name"):
        load_config(path)


def test_config_is_immutable(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Test User"}))
    config = load_config(path)
    with pytest.raises(Exception):
        config.display_name = "Someone Else"
```

`tools/gdoc/tests/test_auth.py`:

```python
import pytest

from tools.gdoc.auth import SCOPES, load_credentials


def test_scope_is_drive_only():
    assert SCOPES == ["https://www.googleapis.com/auth/drive"]


def test_missing_key_names_the_path_and_the_spec(tmp_path):
    missing = tmp_path / "sa-key.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        load_credentials(missing)
    message = str(excinfo.value)
    assert str(missing) in message
    assert "2026-08-13-gdoc-ai-agent-design" in message
```

- [ ] **Step 2: Run and confirm failure**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_config.py tools/gdoc/tests/test_auth.py -v`
Expected: FAIL with `ModuleNotFoundError`

- [ ] **Step 3: Write `tools/gdoc/config.py`**

```python
"""Persistent settings that are not secrets."""

import json
from dataclasses import dataclass
from pathlib import Path

DEFAULT_PATH = Path.home() / ".config" / "gdoc-agent" / "config.json"


@dataclass(frozen=True)
class Config:
    display_name: str
    output_folder_id: str | None = None


def load_config(path: Path | None = None) -> Config:
    """Read the config file.

    display_name is required because it is the only way to tell Nail's comments
    from a colleague's: the Drive API returns no email address for comment
    authors. output_folder_id is optional, because generation falls back to a
    local .docx when there is no folder.
    """
    path = path or DEFAULT_PATH
    if not path.exists():
        raise FileNotFoundError(
            f"config not found at {path}. Create it with a display_name, "
            'for example: {"display_name": "Nail Khusnullin", "output_folder_id": null}'
        )
    data = json.loads(path.read_text())
    display_name = data.get("display_name")
    if not display_name:
        raise ValueError(f"display_name is required in {path}")
    return Config(
        display_name=display_name,
        output_folder_id=data.get("output_folder_id"),
    )
```

- [ ] **Step 4: Write `tools/gdoc/auth.py`**

```python
"""Credentials and the Drive client."""

from pathlib import Path

from google.oauth2 import service_account
from googleapiclient.discovery import build

DEFAULT_KEY_PATH = Path.home() / ".config" / "gdoc-agent" / "sa-key.json"
SCOPES = ["https://www.googleapis.com/auth/drive"]
SPEC = "docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md"


def load_credentials(key_path: Path | None = None):
    key_path = key_path or DEFAULT_KEY_PATH
    if not key_path.exists():
        raise FileNotFoundError(
            f"service account key not found at {key_path}. "
            f"Setup steps are in {SPEC}, section 10."
        )
    return service_account.Credentials.from_service_account_file(
        str(key_path), scopes=SCOPES
    )


def drive_service(credentials=None):
    """Build a Drive v3 client.

    cache_discovery is off because the on-disk discovery cache warns noisily
    under a venv and buys nothing for a tool that runs for a few seconds.
    """
    return build(
        "drive", "v3", credentials=credentials or load_credentials(), cache_discovery=False
    )
```

- [ ] **Step 5: Run and confirm passing**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/ -v`
Expected: 14 passed

- [ ] **Step 6: Create the real config file**

```bash
cat > ~/.config/gdoc-agent/config.json <<'EOF'
{
  "display_name": "Nail Khusnullin",
  "output_folder_id": "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"
}
EOF
chmod 600 ~/.config/gdoc-agent/config.json
```

`display_name` must match exactly what the Drive API returns as `author.displayName`, which the earlier probe showed is `Nail Khusnullin`.

The folder id is the shared drive folder named "test folder", on drive `0AA2s2yLSmVTFUk9PVA`. The probe confirmed `canAddChildren: true` there. It is a test folder, so expect to change this value once Nail picks a permanent one.

- [ ] **Step 7: Commit**

```bash
git add tools/gdoc/
git commit -m "feat: gdoc config and Drive service factory"
```

---

### Task 3: Prove the Commenter guarantee

This task exists because the spec's one hard control has never been exercised. Until an edit is shown to fail, "the agent cannot change your document" is a claim rather than a fact.

The share was downgraded to Commenter on 2026-08-13 and probed. What is settled: `canEdit` is `false`, `canComment` is `true`, replies still post, and `permissions.list` is refused with 403. What is not settled: whether an actual write is refused. The write probe was blocked by the sandbox before it reached Google, so this task still has to run it.

**Files:**
- Create: `tools/gdoc/tests/conftest.py`
- Create: `tools/gdoc/tests/test_access_integration.py`
- Create: `pytest.ini`
- Modify: `docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md` (section 10)

**Interfaces:**
- Consumes: `drive_service` from Task 2
- Produces: a `pytest.mark.integration` convention, and `drive` / `test_doc_id` fixtures for later tasks

- [ ] **Step 1: Get Nail's approval to attempt a write against his document**

The share is already Commenter, so no change is needed there. The obstacle is local: Claude Code's auto mode blocks a command that renames the document or inserts text into it, and it is right to. Two of the tests below attempt exactly that.

Tell Nail what the tests will try, and ask him to approve the run when the prompt appears. Both writes are expected to fail; if one succeeds, the test reverts it.

Never use `--dangerously-skip-permissions` to get past this. If he will not approve, stop and say the guarantee is unproven, which is a real answer.

- [ ] **Step 2: Write `tools/gdoc/tests/conftest.py`**

```python
import pytest

TEST_DOC_ID = "1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw"


@pytest.fixture(scope="session")
def drive():
    from tools.gdoc.auth import drive_service

    return drive_service()


@pytest.fixture(scope="session")
def test_doc_id():
    return TEST_DOC_ID
```

- [ ] **Step 3: Write the failing test**

`tools/gdoc/tests/test_access_integration.py`:

```python
"""The Commenter guarantee, asserted against the live API.

These assert on what Google permits, not on what the agent chooses to do.
A test that only proved the agent did not try to edit would prove nothing:
the guarantee is that it cannot.
"""

import pytest
from googleapiclient.errors import HttpError

pytestmark = pytest.mark.integration


def test_can_still_read_the_file(drive, test_doc_id):
    meta = drive.files().get(fileId=test_doc_id, fields="id,name").execute()
    assert meta["id"] == test_doc_id


def test_can_still_read_comments(drive, test_doc_id):
    res = drive.comments().list(fileId=test_doc_id, fields="comments(id)").execute()
    assert len(res.get("comments", [])) >= 3


def test_google_says_the_agent_cannot_edit(drive, test_doc_id):
    caps = drive.files().get(fileId=test_doc_id, fields="capabilities").execute()["capabilities"]
    assert caps["canEdit"] is False
    assert caps["canComment"] is True


def test_renaming_the_file_is_refused(drive, test_doc_id):
    """The rename must be a real change.

    An earlier probe patched the name to its existing value and got 200 back.
    That proved nothing: Drive had nothing to change. The new name has to
    differ, and the test reverts it if the call unexpectedly succeeds.
    """
    original = drive.files().get(fileId=test_doc_id, fields="name").execute()["name"]
    try:
        drive.files().update(fileId=test_doc_id, body={"name": original + " PROBE"}).execute()
    except HttpError as error:
        assert error.resp.status in (403, 404)
        return
    drive.files().update(fileId=test_doc_id, body={"name": original}).execute()
    pytest.fail("rename succeeded under Commenter, so the guarantee does not hold")


def test_editing_the_text_is_refused(drive, test_doc_id):
    """The write that actually matters.

    A rename is metadata. This inserts a character into the body, which is the
    thing the design promises can never happen. The Docs API is reached with
    the same drive scope.
    """
    from googleapiclient.discovery import build

    docs = build("docs", "v1", credentials=drive._http.credentials, cache_discovery=False)
    try:
        docs.documents().batchUpdate(
            documentId=test_doc_id,
            body={"requests": [{"insertText": {"location": {"index": 1}, "text": "PROBE"}}]},
        ).execute()
    except HttpError as error:
        assert error.resp.status in (403, 404), f"unexpected status: {error}"
        return
    pytest.fail("text insert succeeded under Commenter, so the guarantee does not hold")


def test_deleting_a_comment_is_refused(drive, test_doc_id):
    res = drive.comments().list(fileId=test_doc_id, fields="comments(id)").execute()
    comment_id = res["comments"][0]["id"]
    with pytest.raises(HttpError) as excinfo:
        drive.comments().delete(fileId=test_doc_id, commentId=comment_id).execute()
    assert excinfo.value.resp.status in (403, 404)


def test_permissions_list_is_refused(drive, test_doc_id):
    """Already observed on 2026-08-13: 403.

    This is asserted, not printed, because a change here would change the
    design. Section 10's external-audience check cannot be done by the agent,
    so it belongs to Nail. If this ever starts passing, revisit that decision.
    """
    with pytest.raises(HttpError) as excinfo:
        drive.permissions().list(
            fileId=test_doc_id, fields="permissions(role,type,emailAddress)"
        ).execute()
    assert excinfo.value.resp.status in (403, 404)


def test_the_output_folder_accepts_new_files(drive):
    """The one place the agent is allowed to write."""
    from tools.gdoc.config import load_config

    folder_id = load_config().output_folder_id
    folder = drive.files().get(
        fileId=folder_id, supportsAllDrives=True, fields="name,driveId,capabilities(canAddChildren)"
    ).execute()
    assert folder["capabilities"]["canAddChildren"] is True
    assert folder.get("driveId"), "the output folder should be on a shared drive"
```

If `drive._http.credentials` turns out to be unavailable on the built client, build
the Docs client from `load_credentials()` directly instead. Reaching into a
private attribute is the shortcut; the fallback is one extra import.

- [ ] **Step 4: Add `pytest.ini` at the repo root**

```ini
[pytest]
addopts = -m "not integration"
markers =
    integration: touches the real Drive API and the test document
```

- [ ] **Step 5: Run the integration tests**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_access_integration.py -v -m integration -s`
Expected: 8 passed. The reads pass by working, the three write attempts pass by getting 403 or 404, and `permissions.list` passes by being refused.

If a write test fails because the call succeeded, **stop and tell Nail**. Every later task assumes this guarantee holds, and the design's main safety claim would be false. Do not carry on and fix it later.

- [ ] **Step 6: Confirm the default run skips them**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/ -v`
Expected: unit tests pass, integration tests deselected.

- [ ] **Step 7: Record the outcome in the spec**

Edit `docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md`, section 10. Record in the "Verified" table that a rename and a text insert are both refused, and drop item 1 from the "still untested" list. The `permissions.list` 403 and the external-audience change are already written into the spec, so leave those alone.

- [ ] **Step 8: Commit**

```bash
git add tools/gdoc/ pytest.ini docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md
git commit -m "test: prove the Commenter guarantee holds

Downgraded the test doc share to Commenter, then asserted the API refuses a
rename and a comment delete while reads still work. Records what
permissions.list does under Commenter, which section 10's external-audience
check depends on."
```

---

### Task 4: Comment model and parsing

**Files:**
- Create: `tools/gdoc/model.py`
- Create: `tools/gdoc/tests/test_model.py`

**Interfaces:**
- Consumes: nothing
- Produces: frozen dataclasses `Reply(id: str, content: str, by_agent: bool)` and `Thread(id, content, author_name, author_email, by_agent, quoted, resolved, replies)` with properties `is_anchored: bool` and `has_agent_reply: bool`; `parse_thread(raw: dict) -> Thread`

- [ ] **Step 1: Write the failing test**

`tools/gdoc/tests/test_model.py`:

```python
from tools.gdoc.model import parse_thread

ANCHORED = {
    "id": "AAACFjp837M",
    "content": "ai: rephrase",
    "author": {"displayName": "Nail Khusnullin", "emailAddress": None, "me": False},
    "quotedFileContent": {"value": "asdasd"},
    "resolved": False,
    "replies": [],
}

UNANCHORED = {
    "id": "AAACFjp837Z",
    "content": "ai? does this read as a policy or a procedure",
    "author": {"displayName": "Nail Khusnullin", "me": False},
    "resolved": False,
    "replies": [],
}

ALREADY_ANSWERED = {
    "id": "AAACFjp837I",
    "content": "ai? who you are",
    "author": {"displayName": "Nail Khusnullin", "me": False},
    "quotedFileContent": {"value": "Asdasdasd"},
    "resolved": False,
    "replies": [
        {
            "id": "AAACFjd7zKM",
            "content": "I am Nail AI, a service account.",
            "author": {"displayName": "nail-ai@doc-agent-505414.iam.gserviceaccount.com", "me": True},
        }
    ],
}


def test_parses_an_anchored_comment():
    thread = parse_thread(ANCHORED)
    assert thread.id == "AAACFjp837M"
    assert thread.content == "ai: rephrase"
    assert thread.author_name == "Nail Khusnullin"
    assert thread.quoted == "asdasd"
    assert thread.is_anchored is True


def test_author_email_is_none_and_that_is_expected():
    assert parse_thread(ANCHORED).author_email is None


def test_unanchored_comment_has_no_quote():
    thread = parse_thread(UNANCHORED)
    assert thread.quoted is None
    assert thread.is_anchored is False


def test_detects_an_existing_agent_reply():
    thread = parse_thread(ALREADY_ANSWERED)
    assert thread.has_agent_reply is True
    assert thread.replies[0].by_agent is True


def test_thread_without_replies_has_no_agent_reply():
    assert parse_thread(ANCHORED).has_agent_reply is False


def test_missing_author_block_does_not_crash():
    thread = parse_thread({"id": "x", "content": "ai: hi"})
    assert thread.author_name == "unknown"
    assert thread.by_agent is False


def test_thread_is_immutable():
    thread = parse_thread(ANCHORED)
    try:
        thread.content = "changed"
    except Exception:
        return
    raise AssertionError("Thread should be frozen")


def test_thread_is_hashable_so_it_can_go_in_a_set():
    assert isinstance(hash(parse_thread(ANCHORED)), int)
```

- [ ] **Step 2: Run and confirm failure**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_model.py -v`
Expected: FAIL with `ModuleNotFoundError`

- [ ] **Step 3: Write `tools/gdoc/model.py`**

```python
"""Comment threads as immutable values.

replies is a tuple rather than a list so Thread stays hashable and can go in a
set, which the filters rely on.
"""

from dataclasses import dataclass


@dataclass(frozen=True)
class Reply:
    id: str
    content: str
    by_agent: bool


@dataclass(frozen=True)
class Thread:
    id: str
    content: str
    author_name: str
    author_email: str | None
    by_agent: bool
    quoted: str | None
    resolved: bool
    replies: tuple[Reply, ...]

    @property
    def is_anchored(self) -> bool:
        return self.quoted is not None

    @property
    def has_agent_reply(self) -> bool:
        return any(reply.by_agent for reply in self.replies)


def parse_thread(raw: dict) -> Thread:
    """Build a Thread from one Drive comments.list entry.

    Every field is defensive because the API omits rather than nulls: an
    unanchored comment has no quotedFileContent key at all, and author
    emailAddress is absent for every comment we have seen.
    """
    author = raw.get("author") or {}
    quoted_block = raw.get("quotedFileContent") or {}
    replies = tuple(
        Reply(
            id=item.get("id", ""),
            content=item.get("content") or "",
            by_agent=bool((item.get("author") or {}).get("me")),
        )
        for item in raw.get("replies") or ()
    )
    return Thread(
        id=raw["id"],
        content=raw.get("content") or "",
        author_name=author.get("displayName") or "unknown",
        author_email=author.get("emailAddress"),
        by_agent=bool(author.get("me")),
        quoted=quoted_block.get("value"),
        resolved=bool(raw.get("resolved")),
        replies=replies,
    )
```

- [ ] **Step 4: Run and confirm passing**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_model.py -v`
Expected: 8 passed

- [ ] **Step 5: Commit**

```bash
git add tools/gdoc/
git commit -m "feat: immutable comment thread model"
```

---

### Task 5: Paged fetch and filtering

**Files:**
- Create: `tools/gdoc/fetch.py`
- Create: `tools/gdoc/filters.py`
- Create: `tools/gdoc/tests/test_fetch.py`
- Create: `tools/gdoc/tests/test_filters.py`

**Interfaces:**
- Consumes: `Thread`, `parse_thread` from Task 4
- Produces: `fetch_threads(drive, doc_id: str) -> tuple[Thread, ...]`; `is_addressed(content: str) -> bool`; `forced_kind(content: str) -> str | None` returning `"question"`, `"instruction"` or `None`; `needs_action(thread: Thread) -> bool`; `partition(threads, display_name: str) -> tuple[tuple[Thread, ...], tuple[Thread, ...], tuple[Thread, ...]]` returning `(mine, others, skipped)`

- [ ] **Step 1: Write the failing tests**

`tools/gdoc/tests/test_fetch.py`:

```python
from unittest.mock import MagicMock

from tools.gdoc.fetch import fetch_threads


def _page(comments, next_token=None):
    page = {"comments": comments}
    if next_token:
        page["nextPageToken"] = next_token
    return page


def test_follows_pagination_until_exhausted():
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [
        _page([{"id": "a", "content": "ai: one"}], next_token="tok"),
        _page([{"id": "b", "content": "ai: two"}]),
    ]
    threads = fetch_threads(drive, "docid")
    assert [t.id for t in threads] == ["a", "b"]


def test_returns_an_immutable_tuple():
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [_page([{"id": "a", "content": "x"}])]
    assert isinstance(fetch_threads(drive, "docid"), tuple)


def test_empty_document_returns_empty_tuple():
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [_page([])]
    assert fetch_threads(drive, "docid") == ()
```

`tools/gdoc/tests/test_filters.py`:

```python
from tools.gdoc.filters import forced_kind, needs_action, partition
from tools.gdoc.model import Reply, Thread

ME = "Nail Khusnullin"


def thread(**kwargs):
    defaults = dict(
        id="t1",
        content="ai: rephrase",
        author_name=ME,
        author_email=None,
        by_agent=False,
        quoted="some text",
        resolved=False,
        replies=(),
    )
    return Thread(**{**defaults, **kwargs})


def test_marker_is_required():
    assert needs_action(thread(content="just a note to myself")) is False


def test_marker_is_case_insensitive():
    assert needs_action(thread(content="AI: rephrase")) is True


def test_marker_must_start_the_comment():
    assert needs_action(thread(content="I think ai: should handle this")) is False


def test_bare_ai_without_punctuation_is_not_a_request():
    """Otherwise every comment starting with a sentence about AI becomes a job."""
    assert needs_action(thread(content="AI is going to change how we write these")) is False


def test_all_three_signs_are_markers():
    assert needs_action(thread(content="ai: rephrase")) is True
    assert needs_action(thread(content="ai? is this right")) is True
    assert needs_action(thread(content="ai! renumber the sections")) is True


def test_the_old_at_form_still_works():
    """Nail used @ai first. Ignoring it silently would be the worst outcome."""
    assert needs_action(thread(content="@ai rephrase")) is True
    assert needs_action(thread(content="@ai: rephrase")) is True


def test_leading_whitespace_is_tolerated():
    assert needs_action(thread(content="  ai: rephrase")) is True


def test_resolved_threads_are_skipped():
    assert needs_action(thread(resolved=True)) is False


def test_threads_the_agent_wrote_are_skipped():
    assert needs_action(thread(by_agent=True)) is False


def test_threads_the_agent_already_answered_are_skipped():
    answered = thread(replies=(Reply(id="r1", content="done", by_agent=True),))
    assert needs_action(answered) is False


def test_a_human_reply_does_not_count_as_answered():
    discussed = thread(replies=(Reply(id="r1", content="good point", by_agent=False),))
    assert needs_action(discussed) is True


def test_question_override():
    assert forced_kind("ai? is this right") == "question"


def test_instruction_override():
    assert forced_kind("ai! renumber everything") == "instruction"


def test_colon_leaves_the_decision_to_the_agent():
    assert forced_kind("ai: rephrase this") is None


def test_old_at_form_overrides_too():
    assert forced_kind("@ai? is this right") == "question"
    assert forced_kind("@ai rephrase this") is None


def test_partition_splits_mine_others_and_skipped():
    mine_thread = thread(id="mine", author_name=ME)
    other_thread = thread(id="other", author_name="William Mejia")
    skipped_thread = thread(id="skipped", resolved=True)
    mine, others, skipped = partition(
        (mine_thread, other_thread, skipped_thread), display_name=ME
    )
    assert [t.id for t in mine] == ["mine"]
    assert [t.id for t in others] == ["other"]
    assert [t.id for t in skipped] == ["skipped"]


def test_partition_returns_tuples():
    mine, others, skipped = partition((thread(),), display_name=ME)
    assert isinstance(mine, tuple) and isinstance(others, tuple) and isinstance(skipped, tuple)
```

- [ ] **Step 2: Run and confirm failure**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_fetch.py tools/gdoc/tests/test_filters.py -v`
Expected: FAIL with `ModuleNotFoundError`

- [ ] **Step 3: Write `tools/gdoc/fetch.py`**

```python
"""Read every comment thread on a document."""

from tools.gdoc.model import Thread, parse_thread

FIELDS = (
    "nextPageToken,"
    "comments(id,createdTime,modifiedTime,resolved,quotedFileContent(value),"
    "content,author(displayName,emailAddress,me),"
    "replies(id,content,author(displayName,emailAddress,me)))"
)


def fetch_threads(drive, doc_id: str) -> tuple[Thread, ...]:
    """Page through comments.list.

    fields is mandatory on this endpoint, and nextPageToken has to be inside it
    or pagination silently stops after the first page.
    """
    collected: list[Thread] = []
    page_token = None
    while True:
        response = (
            drive.comments()
            .list(
                fileId=doc_id,
                fields=FIELDS,
                pageSize=100,
                includeDeleted=False,
                pageToken=page_token,
            )
            .execute()
        )
        collected.extend(parse_thread(raw) for raw in response.get("comments") or ())
        page_token = response.get("nextPageToken")
        if not page_token:
            return tuple(collected)
```

- [ ] **Step 4: Write `tools/gdoc/filters.py`**

```python
"""Decide which threads the agent may act on.

Authorship cannot be verified: the Drive API returns no email address for
comment authors, and display names are editable. So display_name matching is a
sorting aid, not an access control, and the skill always shows both lists for
confirmation before anything is posted.

The marker is `ai` plus a sign, at the start of the comment. It starts with a
letter because `@` opens the people picker in Google Docs and Nail does not want
to pick himself out of an address book every time. The sign is required on the
bare form, so a comment that merely begins with "AI is..." is left alone. The
older `@ai` form is still accepted, with or without a sign, because a comment
that is silently ignored is worse than a rare false positive.
"""

import re

from tools.gdoc.model import Thread

_ADDRESSED = re.compile(r"^\s*(?:@ai\b|ai(?=[:?!]))", re.IGNORECASE)
_SIGN = re.compile(r"^\s*@?ai([:?!])", re.IGNORECASE)

_KINDS = {"?": "question", "!": "instruction"}


def is_addressed(content: str) -> bool:
    """True when the comment is written to the agent."""
    return bool(_ADDRESSED.match(content or ""))


def forced_kind(content: str) -> str | None:
    """Return the kind Nail forced with ai? or ai!, or None to let the agent decide.

    A colon is a deliberate "you choose", so it maps to None, same as no sign.
    """
    match = _SIGN.match(content or "")
    return _KINDS.get(match.group(1)) if match else None


def needs_action(thread: Thread) -> bool:
    """True when the thread is waiting on the agent.

    The agent-reply check is what makes a second run idempotent: it never posts
    twice, and no local record of handled ids is needed.
    """
    return (
        is_addressed(thread.content)
        and not thread.resolved
        and not thread.by_agent
        and not thread.has_agent_reply
    )


def partition(
    threads: tuple[Thread, ...], display_name: str
) -> tuple[tuple[Thread, ...], tuple[Thread, ...], tuple[Thread, ...]]:
    """Split into (mine, others, skipped).

    skipped is returned rather than discarded so the skill can tell Nail why a
    comment he can see was ignored.
    """
    mine: list[Thread] = []
    others: list[Thread] = []
    skipped: list[Thread] = []
    for thread in threads:
        if not needs_action(thread):
            skipped.append(thread)
        elif thread.author_name == display_name:
            mine.append(thread)
        else:
            others.append(thread)
    return tuple(mine), tuple(others), tuple(skipped)
```

- [ ] **Step 5: Run and confirm passing**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/ -v`
Expected: all unit tests pass, integration deselected

- [ ] **Step 6: Commit**

```bash
git add tools/gdoc/
git commit -m "feat: paged comment fetch and action filtering

Idempotency comes from the thread itself: a thread that already has a reply
from the service account is never actioned again, so no local record of
handled comment ids is needed."
```

---

### Task 6: Plain-text reply posting

**Files:**
- Create: `tools/gdoc/reply.py`
- Create: `tools/gdoc/tests/test_reply.py`

**Interfaces:**
- Consumes: nothing
- Produces: `assert_plain_text(body: str) -> str`; `post_reply(drive, doc_id: str, comment_id: str, body: str) -> str` returning the new reply id; `GLOBAL_REFUSAL` template string with one `{item}` placeholder

- [ ] **Step 1: Write the failing test**

`tools/gdoc/tests/test_reply.py`:

```python
from unittest.mock import MagicMock

import pytest

from tools.gdoc.reply import GLOBAL_REFUSAL, assert_plain_text, post_reply


def test_plain_prose_is_accepted():
    body = "Use safeguarded funds, not client money.\n\nReason: it matches the policy."
    assert assert_plain_text(body) == body


def test_bold_markers_are_rejected():
    with pytest.raises(ValueError, match="markdown"):
        assert_plain_text("Use **safeguarded funds** here.")


def test_backticks_are_rejected():
    with pytest.raises(ValueError, match="markdown"):
        assert_plain_text("Rename it to `client_money`.")


def test_heading_markers_are_rejected():
    with pytest.raises(ValueError, match="markdown"):
        assert_plain_text("## Proposed wording\n\nSomething.")


def test_hyphen_lists_are_allowed_because_they_read_fine_as_text():
    body = "Two options:\n- keep the clause\n- move it to the SOP"
    assert assert_plain_text(body) == body


def test_empty_body_is_rejected():
    with pytest.raises(ValueError, match="empty"):
        assert_plain_text("   ")


def test_post_reply_returns_the_new_reply_id():
    drive = MagicMock()
    drive.replies().create.return_value.execute.return_value = {"id": "AAACFjd7zKM"}
    assert post_reply(drive, "doc", "comment", "plain text") == "AAACFjd7zKM"


def test_post_reply_refuses_markdown_before_calling_the_api():
    drive = MagicMock()
    with pytest.raises(ValueError):
        post_reply(drive, "doc", "comment", "has **markdown**")
    drive.replies().create.assert_not_called()


def test_global_refusal_names_the_item_number():
    assert "item 2" in GLOBAL_REFUSAL.format(item=2)


def test_global_refusal_is_itself_plain_text():
    assert assert_plain_text(GLOBAL_REFUSAL.format(item=1))
```

- [ ] **Step 2: Run and confirm failure**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_reply.py -v`
Expected: FAIL with `ModuleNotFoundError`

- [ ] **Step 3: Write `tools/gdoc/reply.py`**

```python
"""Post replies, and refuse anything Docs would render literally."""

import re

# Only the markers that arrive visibly broken. Hyphen bullets read fine as
# plain text, so they are allowed.
_MARKDOWN = re.compile(r"(\*\*|`|^\s{0,3}#{1,6}\s)", re.MULTILINE)

GLOBAL_REFUSAL = (
    "Understood. This needs changes across the document, so I am not proposing "
    "text here. Captured as item {item}. I will handle it in a terminal session "
    "and produce a new version of the document."
)


def assert_plain_text(body: str) -> str:
    """Return body unchanged, or raise if it would not paste cleanly.

    Google Docs comment threads show markdown source as typed, so a fenced
    block or a bold marker becomes something Nail has to strip by hand before
    using the text. Failing here is better than posting it.
    """
    if not body or not body.strip():
        raise ValueError("reply body is empty")
    match = _MARKDOWN.search(body)
    if match:
        raise ValueError(
            f"reply contains markdown ({match.group(0)!r}); Docs comments render it literally"
        )
    return body


def post_reply(drive, doc_id: str, comment_id: str, body: str) -> str:
    assert_plain_text(body)
    created = (
        drive.replies()
        .create(
            fileId=doc_id,
            commentId=comment_id,
            body={"content": body},
            fields="id,createdTime",
        )
        .execute()
    )
    return created["id"]
```

- [ ] **Step 4: Run and confirm passing**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_reply.py -v`
Expected: 10 passed

- [ ] **Step 5: Commit**

```bash
git add tools/gdoc/
git commit -m "feat: plain-text reply posting with a markdown guard"
```

---

### Task 7: Export a document to markdown

**Files:**
- Create: `tools/gdoc/export.py`
- Create: `tools/gdoc/tests/test_export.py`
- Modify: `tools/gdoc/tests/test_access_integration.py` (append one test)

**Interfaces:**
- Consumes: nothing
- Produces: `export_markdown(drive, doc_id: str) -> str`; `MARKDOWN_MIME`; `DOCX_MIME`; `PANDOC`

- [ ] **Step 1: Write the failing test**

`tools/gdoc/tests/test_export.py`:

```python
from unittest.mock import MagicMock, patch

import pytest
from googleapiclient.errors import HttpError

from tools.gdoc.export import DOCX_MIME, MARKDOWN_MIME, export_markdown


class FakeResponse:
    def __init__(self, status):
        self.status = status
        self.reason = "test"


def http_error(status):
    return HttpError(FakeResponse(status), b"{}")


def test_uses_native_markdown_export_when_available():
    drive = MagicMock()
    drive.files().export.return_value.execute.return_value = b"# Title\n\nBody."
    assert export_markdown(drive, "doc") == "# Title\n\nBody."
    drive.files().export.assert_called_with(fileId="doc", mimeType=MARKDOWN_MIME)


def test_falls_back_to_docx_and_pandoc_when_markdown_is_unsupported():
    drive = MagicMock()
    drive.files().export.return_value.execute.side_effect = [
        http_error(400),
        b"PK\x03\x04 fake docx bytes",
    ]
    with patch("tools.gdoc.export._pandoc_docx_to_markdown", return_value="# From docx") as pandoc:
        assert export_markdown(drive, "doc") == "# From docx"
    pandoc.assert_called_once()


def test_unexpected_api_errors_are_not_swallowed():
    drive = MagicMock()
    drive.files().export.return_value.execute.side_effect = http_error(500)
    with pytest.raises(HttpError):
        export_markdown(drive, "doc")


def test_docx_mime_is_the_openxml_one():
    assert DOCX_MIME.endswith("wordprocessingml.document")
```

- [ ] **Step 2: Run and confirm failure**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_export.py -v`
Expected: FAIL with `ModuleNotFoundError`

- [ ] **Step 3: Write `tools/gdoc/export.py`**

```python
"""Get a Google Doc as markdown.

Drive can export a Doc straight to markdown. That path is preferred because it
avoids a temp file and a subprocess. It is new enough that it may not be
available for every document, so a docx export through pandoc is kept as a
fallback: pandoc is already required for generating new versions, so the
fallback adds a code path but no new dependency.
"""

import subprocess
import tempfile
from pathlib import Path

from googleapiclient.errors import HttpError

MARKDOWN_MIME = "text/markdown"
DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
PANDOC = "/opt/homebrew/bin/pandoc"

# Statuses that mean "this conversion is not offered", as opposed to a real fault.
_UNSUPPORTED = (400, 415)


def _pandoc_docx_to_markdown(docx_bytes: bytes) -> str:
    with tempfile.TemporaryDirectory() as tmp:
        docx_path = Path(tmp) / "doc.docx"
        docx_path.write_bytes(docx_bytes)
        result = subprocess.run(
            [PANDOC, str(docx_path), "-f", "docx", "-t", "markdown", "--wrap=none"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout


def export_markdown(drive, doc_id: str) -> str:
    try:
        data = drive.files().export(fileId=doc_id, mimeType=MARKDOWN_MIME).execute()
        return data.decode("utf-8") if isinstance(data, bytes) else data
    except HttpError as error:
        if error.resp.status not in _UNSUPPORTED:
            raise
    docx = drive.files().export(fileId=doc_id, mimeType=DOCX_MIME).execute()
    return _pandoc_docx_to_markdown(docx)
```

- [ ] **Step 4: Run and confirm passing**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_export.py -v`
Expected: 4 passed

- [ ] **Step 5: Add the integration test that says which path is really used**

Append to `tools/gdoc/tests/test_access_integration.py`:

```python
def test_native_markdown_export_works_on_the_real_document(drive, test_doc_id):
    """Records whether the pandoc fallback is dead code or load-bearing."""
    from tools.gdoc.export import export_markdown

    text = export_markdown(drive, test_doc_id)
    print(f"exported {len(text)} characters of markdown")
    assert text.strip()
```

- [ ] **Step 6: Run it**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_access_integration.py -v -m integration -s -k markdown`
Expected: PASS, printing a non-zero character count.

- [ ] **Step 7: Commit**

```bash
git add tools/gdoc/
git commit -m "feat: export a doc to markdown, with a pandoc fallback"
```

---

### Task 8: Pairing frontmatter

**Files:**
- Create: `tools/gdoc/pairing.py`
- Create: `tools/gdoc/tests/test_pairing.py`

**Interfaces:**
- Consumes: nothing
- Produces: `Pairing` frozen dataclass with `doc_id: str`, `synced: str | None`, `versions: tuple[dict, ...]`; `read_pairing(md_path: Path) -> Pairing | None`; `write_pairing(md_path: Path, pairing: Pairing) -> None`; `add_version(pairing: Pairing, doc_id: str, created: str) -> Pairing`; `find_by_doc_id(root: Path, doc_id: str) -> Path | None`

- [ ] **Step 1: Write the failing test**

`tools/gdoc/tests/test_pairing.py`:

```python
from tools.gdoc.pairing import Pairing, add_version, find_by_doc_id, read_pairing, write_pairing

PAIRED = """\
---
title: Safeguarding policy
gdoc: 1AbC
gdoc_synced: 2026-08-13
gdoc_versions:
  - id: 1XyZ
    created: 2026-08-14
---

# Safeguarding policy

Body text.
"""

UNPAIRED = """\
---
title: Some note
---

Body.
"""

NO_FRONTMATTER = "# Just a heading\n\nBody.\n"


def test_reads_an_existing_pairing(tmp_path):
    path = tmp_path / "policy.md"
    path.write_text(PAIRED)
    pairing = read_pairing(path)
    assert pairing.doc_id == "1AbC"
    assert pairing.synced == "2026-08-13"
    assert pairing.versions == ({"id": "1XyZ", "created": "2026-08-14"},)


def test_returns_none_when_frontmatter_has_no_gdoc_key(tmp_path):
    path = tmp_path / "note.md"
    path.write_text(UNPAIRED)
    assert read_pairing(path) is None


def test_returns_none_when_there_is_no_frontmatter(tmp_path):
    path = tmp_path / "plain.md"
    path.write_text(NO_FRONTMATTER)
    assert read_pairing(path) is None


def test_write_preserves_other_frontmatter_keys_and_the_body(tmp_path):
    path = tmp_path / "policy.md"
    path.write_text(PAIRED)
    write_pairing(path, Pairing(doc_id="1AbC", synced="2026-08-20", versions=()))
    text = path.read_text()
    assert "title: Safeguarding policy" in text
    assert "2026-08-20" in text
    assert "Body text." in text


def test_write_adds_frontmatter_to_a_file_that_has_none(tmp_path):
    path = tmp_path / "plain.md"
    path.write_text(NO_FRONTMATTER)
    write_pairing(path, Pairing(doc_id="1New", synced="2026-08-20", versions=()))
    text = path.read_text()
    assert text.startswith("---\n")
    assert "gdoc: 1New" in text
    assert "Just a heading" in text


def test_add_version_returns_a_new_pairing_and_leaves_the_original_alone():
    original = Pairing(doc_id="1AbC", synced="2026-08-13", versions=())
    updated = add_version(original, doc_id="1XyZ", created="2026-08-14")
    assert original.versions == ()
    assert updated.versions == ({"id": "1XyZ", "created": "2026-08-14"},)
    assert updated is not original


def test_find_by_doc_id_locates_the_paired_file(tmp_path):
    (tmp_path / "sub").mkdir()
    target = tmp_path / "sub" / "policy.md"
    target.write_text(PAIRED)
    (tmp_path / "other.md").write_text(UNPAIRED)
    assert find_by_doc_id(tmp_path, "1AbC") == target


def test_find_by_doc_id_returns_none_when_nothing_matches(tmp_path):
    (tmp_path / "other.md").write_text(UNPAIRED)
    assert find_by_doc_id(tmp_path, "1AbC") is None
```

- [ ] **Step 2: Run and confirm failure**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_pairing.py -v`
Expected: FAIL with `ModuleNotFoundError`

- [ ] **Step 3: Write `tools/gdoc/pairing.py`**

```python
"""The document to markdown pairing, stored in the markdown file's frontmatter.

Frontmatter rather than a central index because the skill is global and runs in
several repos. An index would have to live outside them all and would go stale
silently when a file moves.
"""

import datetime
import os
import tempfile
from dataclasses import dataclass, replace
from pathlib import Path

import yaml

_FENCE = "---"


@dataclass(frozen=True)
class Pairing:
    doc_id: str
    synced: str | None = None
    versions: tuple[dict, ...] = ()


def _split(text: str) -> tuple[dict, str]:
    """Return (frontmatter dict, body). Missing frontmatter gives an empty dict."""
    if not text.startswith(_FENCE):
        return {}, text
    parts = text.split(_FENCE, 2)
    if len(parts) < 3:
        return {}, text
    return yaml.safe_load(parts[1]) or {}, parts[2].lstrip("\n")


def _str(value: object) -> str | None:
    """Convert YAML-parsed date objects to ISO strings; return strings unchanged."""
    if value is None:
        return None
    if isinstance(value, (datetime.date, datetime.datetime)):
        return value.isoformat()
    return str(value)


def read_pairing(md_path: Path) -> Pairing | None:
    front, _ = _split(md_path.read_text())
    doc_id = front.get("gdoc")
    if not doc_id:
        return None
    raw_versions = front.get("gdoc_versions") or ()
    versions = tuple(
        {k: _str(v) if isinstance(v, (datetime.date, datetime.datetime)) else v for k, v in ver.items()}
        for ver in raw_versions
    )
    return Pairing(doc_id=str(doc_id), synced=_str(front.get("gdoc_synced")), versions=versions)


def write_pairing(md_path: Path, pairing: Pairing) -> None:
    """Rewrite only the three gdoc keys, leaving every other key and the body alone.

    The write is atomic: content goes to a sibling temp file first, then
    os.replace renames it onto md_path in one syscall.  A crash or kill
    between the two leaves the original file intact.
    """
    front, body = _split(md_path.read_text())
    updated = dict(front)
    updated["gdoc"] = pairing.doc_id
    if pairing.synced:
        updated["gdoc_synced"] = pairing.synced
    else:
        updated.pop("gdoc_synced", None)
    if pairing.versions:
        updated["gdoc_versions"] = [dict(version) for version in pairing.versions]
    else:
        updated.pop("gdoc_versions", None)
    rendered = yaml.safe_dump(updated, sort_keys=False, allow_unicode=True).rstrip("\n")
    content = f"{_FENCE}\n{rendered}\n{_FENCE}\n\n{body}"
    fd, tmp_str = tempfile.mkstemp(dir=md_path.parent, suffix=".tmp")
    tmp = Path(tmp_str)
    try:
        os.write(fd, content.encode())
        os.close(fd)
        os.replace(tmp, md_path)
    except Exception:
        try:
            os.close(fd)
        except OSError:
            pass
        tmp.unlink(missing_ok=True)
        raise


def add_version(pairing: Pairing, doc_id: str, created: str) -> Pairing:
    """Return a new Pairing with one more version. Never mutates the input."""
    return replace(pairing, versions=pairing.versions + ({"id": doc_id, "created": created},))


def find_by_doc_id(root: Path, doc_id: str) -> Path | None:
    """Search a tree for the markdown file already paired to this document."""
    for candidate in sorted(root.rglob("*.md")):
        try:
            pairing = read_pairing(candidate)
        except Exception:
            continue  # a malformed file must not stop the search
        if pairing and pairing.doc_id == doc_id:
            return candidate
    return None
```

- [ ] **Step 4: Run and confirm passing**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_pairing.py -v`
Expected: 8 passed

- [ ] **Step 5: Commit**

```bash
git add tools/gdoc/
git commit -m "feat: doc-to-markdown pairing in frontmatter"
```

---

### Task 9: Mirror with an uncommitted-changes guard

This resolves open question 1 in the spec: the markdown is what Nail approves, so a mirror must never silently overwrite local edits. Git already knows whether that risk exists.

**Files:**
- Create: `tools/gdoc/mirror.py`
- Create: `tools/gdoc/tests/test_mirror.py`
- Modify: `docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md` (sections 15 and 16)

**Interfaces:**
- Consumes: nothing
- Produces: `MIRROR_DIR`; `MirrorConflict`; `slugify(name: str) -> str`; `mirror_path(repo_root: Path, slug: str) -> Path`; `is_dirty(path: Path) -> bool`; `write_mirror(repo_root: Path, slug: str, markdown: str, force: bool = False) -> Path`

- [ ] **Step 1: Write the failing test**

`tools/gdoc/tests/test_mirror.py`:

```python
import subprocess

import pytest

from tools.gdoc.mirror import MirrorConflict, mirror_path, slugify, write_mirror


def git(repo, *args):
    subprocess.run(["git", *args], cwd=repo, check=True, capture_output=True)


@pytest.fixture
def repo(tmp_path):
    git(tmp_path, "init")
    git(tmp_path, "config", "user.email", "test@example.com")
    git(tmp_path, "config", "user.name", "Test")
    (tmp_path / "seed.txt").write_text("seed")
    git(tmp_path, "add", ".")
    git(tmp_path, "commit", "-m", "seed")
    return tmp_path


def test_slugify_lowercases_and_hyphenates():
    assert slugify("Safeguarding Policy v3") == "safeguarding-policy-v3"


def test_slugify_drops_punctuation_and_collapses_gaps():
    assert slugify("  Q4 Budget (draft!) ") == "q4-budget-draft"


def test_slugify_falls_back_when_nothing_survives():
    assert slugify("!!!") == "untitled"


def test_mirror_path_is_under_docs_gdoc(repo):
    assert mirror_path(repo, "policy") == repo / "docs" / "gdoc" / "policy" / "mirror.md"


def test_write_mirror_creates_the_file_and_parents(repo):
    path = write_mirror(repo, "policy", "# Title\n")
    assert path.read_text() == "# Title\n"


def test_write_mirror_overwrites_a_committed_mirror(repo):
    write_mirror(repo, "policy", "# One\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "mirror")
    write_mirror(repo, "policy", "# Two\n")
    assert mirror_path(repo, "policy").read_text() == "# Two\n"


def test_write_mirror_refuses_to_clobber_uncommitted_edits(repo):
    write_mirror(repo, "policy", "# One\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "mirror")
    mirror_path(repo, "policy").write_text("# Edited by hand\n")
    with pytest.raises(MirrorConflict, match="uncommitted"):
        write_mirror(repo, "policy", "# From Drive\n")


def test_force_overrides_the_guard(repo):
    write_mirror(repo, "policy", "# One\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "mirror")
    mirror_path(repo, "policy").write_text("# Edited by hand\n")
    write_mirror(repo, "policy", "# From Drive\n", force=True)
    assert mirror_path(repo, "policy").read_text() == "# From Drive\n"
```

- [ ] **Step 2: Run and confirm failure**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_mirror.py -v`
Expected: FAIL with `ModuleNotFoundError`

- [ ] **Step 3: Write `tools/gdoc/mirror.py`**

```python
"""Write the markdown mirror of a document, without losing local edits."""

import re
import subprocess
from pathlib import Path

MIRROR_DIR = Path("docs") / "gdoc"

_NON_WORD = re.compile(r"[^a-z0-9]+")


class MirrorConflict(RuntimeError):
    """The mirror on disk has uncommitted changes, so overwriting could lose work."""


def slugify(name: str) -> str:
    slug = _NON_WORD.sub("-", (name or "").lower()).strip("-")
    return slug or "untitled"


def mirror_path(repo_root: Path, slug: str) -> Path:
    return repo_root / MIRROR_DIR / slug / "mirror.md"


def is_dirty(path: Path) -> bool:
    """True when git reports uncommitted changes for this path.

    An untracked file is not dirty in the sense that matters here: there is
    nothing committed to lose. Only tracked-and-modified counts.
    """
    result = subprocess.run(
        ["git", "status", "--porcelain", "--", str(path)],
        cwd=path.parent,
        capture_output=True,
        text=True,
    )
    return any(line and not line.startswith("??") for line in result.stdout.splitlines())


def write_mirror(repo_root: Path, slug: str, markdown: str, force: bool = False) -> Path:
    """Write the mirror, refusing when that would discard uncommitted edits.

    The markdown is what Nail approves and edits, so a refresh from Drive must
    not overwrite work that is not yet in git. git already knows the answer, so
    no extra state is needed.
    """
    path = mirror_path(repo_root, slug)
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and not force and is_dirty(path):
        raise MirrorConflict(
            f"{path} has uncommitted changes. Commit or discard them, "
            "or pass force to overwrite."
        )
    path.write_text(markdown)
    return path
```

- [ ] **Step 4: Run and confirm passing**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_mirror.py -v`
Expected: 8 passed

- [ ] **Step 5: Close open question 1 in the spec**

In `docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md`, delete open question 1 from section 16 and add this row to the section 15 decisions table:

```
| Mirror refuses to overwrite uncommitted edits | The markdown is what Nail approves. git already knows whether local work would be lost, so no extra state is needed |
```

- [ ] **Step 6: Commit**

```bash
git add tools/gdoc/ docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md
git commit -m "feat: mirror writing with an uncommitted-changes guard

Closes open question 1: a refresh from Drive never overwrites markdown edits
that are not yet in git."
```

---

### Task 10: Capture global items

**Files:**
- Create: `tools/gdoc/pending.py`
- Create: `tools/gdoc/tests/test_pending.py`

**Interfaces:**
- Consumes: `Thread` from Task 4, `MIRROR_DIR` from Task 9
- Produces: `pending_path(repo_root: Path, slug: str) -> Path`; `next_item_number(path: Path) -> int`; `append_item(repo_root: Path, slug: str, thread: Thread, doc_id: str, today: str) -> int`

- [ ] **Step 1: Write the failing test**

`tools/gdoc/tests/test_pending.py`:

```python
import pytest

from tools.gdoc.model import Thread
from tools.gdoc.pending import append_item, next_item_number, pending_path

TODAY = "2026-08-13"


def thread(id="t1", content="ai! renumber the sections", quoted="Section 4"):
    return Thread(
        id=id,
        content=content,
        author_name="Nail Khusnullin",
        author_email=None,
        by_agent=False,
        quoted=quoted,
        resolved=False,
        replies=(),
    )


def test_pending_path_sits_beside_the_mirror(tmp_path):
    assert pending_path(tmp_path, "policy") == tmp_path / "docs" / "gdoc" / "policy" / "pending.md"


def test_first_item_is_number_one(tmp_path):
    assert next_item_number(pending_path(tmp_path, "policy")) == 1


def test_append_creates_the_file_with_a_heading(tmp_path):
    number = append_item(tmp_path, "policy", thread(), doc_id="1AbC", today=TODAY)
    text = pending_path(tmp_path, "policy").read_text()
    assert number == 1
    assert text.startswith("# Pending global items")
    assert "1AbC" in text


def test_item_records_the_comment_id_text_quote_and_date(tmp_path):
    append_item(tmp_path, "policy", thread(), doc_id="1AbC", today=TODAY)
    text = pending_path(tmp_path, "policy").read_text()
    assert "## Item 1" in text
    assert "ai! renumber the sections" in text
    assert "Section 4" in text
    assert TODAY in text
    assert "t1" in text


def test_second_append_numbers_two_and_keeps_the_first(tmp_path):
    append_item(tmp_path, "policy", thread(id="t1"), doc_id="1AbC", today=TODAY)
    number = append_item(
        tmp_path, "policy", thread(id="t2", content="ai! split section 3"), doc_id="1AbC", today=TODAY
    )
    text = pending_path(tmp_path, "policy").read_text()
    assert number == 2
    assert "## Item 1" in text and "## Item 2" in text


def test_unanchored_thread_says_whole_document(tmp_path):
    append_item(tmp_path, "policy", thread(quoted=None), doc_id="1AbC", today=TODAY)
    assert "whole document" in pending_path(tmp_path, "policy").read_text()


def test_appending_the_same_comment_twice_is_refused(tmp_path):
    append_item(tmp_path, "policy", thread(id="t1"), doc_id="1AbC", today=TODAY)
    with pytest.raises(ValueError, match="already captured"):
        append_item(tmp_path, "policy", thread(id="t1"), doc_id="1AbC", today=TODAY)
```

- [ ] **Step 2: Run and confirm failure**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_pending.py -v`
Expected: FAIL with `ModuleNotFoundError`

- [ ] **Step 3: Write `tools/gdoc/pending.py`**

```python
"""Global items, captured for a later terminal session.

One markdown file per document. The comment id is written into each item so a
re-run can tell that an item is already captured, which matters because the
in-thread reply is the only other record and Nail may resolve it.
"""

import re
from pathlib import Path

from tools.gdoc.mirror import MIRROR_DIR
from tools.gdoc.model import Thread

_ITEM_HEADING = re.compile(r"^## Item (\d+)", re.MULTILINE)

_FILE_HEADER = """\
# Pending global items

Document: https://docs.google.com/document/d/{doc_id}/edit

Each item needs changes across the document, so it was not answered in the
comment thread. Work through them with /gdoc-apply.
"""

_ITEM = """
## Item {number}

- Captured: {today}
- Comment id: {comment_id}
- Anchored to: {anchor}

Nail asked:

> {request}
"""


def pending_path(repo_root: Path, slug: str) -> Path:
    return repo_root / MIRROR_DIR / slug / "pending.md"


def next_item_number(path: Path) -> int:
    if not path.exists():
        return 1
    numbers = [int(match) for match in _ITEM_HEADING.findall(path.read_text())]
    return max(numbers) + 1 if numbers else 1


def append_item(repo_root: Path, slug: str, thread: Thread, doc_id: str, today: str) -> int:
    path = pending_path(repo_root, slug)
    path.parent.mkdir(parents=True, exist_ok=True)
    existing = path.read_text() if path.exists() else ""
    if thread.id in existing:
        raise ValueError(f"comment {thread.id} is already captured in {path}")
    if not existing:
        existing = _FILE_HEADER.format(doc_id=doc_id)
    number = next_item_number(path)
    anchor = thread.quoted.strip() if thread.quoted else "whole document"
    item = _ITEM.format(
        number=number,
        today=today,
        comment_id=thread.id,
        anchor=anchor,
        request=thread.content.strip(),
    )
    path.write_text(existing.rstrip("\n") + "\n" + item)
    return number
```

- [ ] **Step 4: Run and confirm passing**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_pending.py -v`
Expected: 7 passed

- [ ] **Step 5: Commit**

```bash
git add tools/gdoc/
git commit -m "feat: capture global items to pending.md"
```

---

### Task 11: Generate a new version

**Files:**
- Create: `tools/gdoc/generate.py`
- Create: `tools/gdoc/tests/test_generate.py`
- Modify: `.gitignore`

**Interfaces:**
- Consumes: `DOCX_MIME` and `PANDOC` from Task 7
- Produces: `Result` frozen dataclass with `docx_path: Path`, `doc_id: str | None`, `link: str | None`, `reason: str | None`; `md_to_docx(md_path: Path, out_path: Path) -> Path`; `upload_as_gdoc(drive, docx_path: Path, name: str, folder_id: str) -> dict`; `generate(drive, md_path: Path, name: str, out_path: Path, folder_id: str | None) -> Result`

- [ ] **Step 1: Write the failing test**

`tools/gdoc/tests/test_generate.py`:

```python
from unittest.mock import MagicMock

import pytest
from googleapiclient.errors import HttpError

from tools.gdoc.generate import generate, md_to_docx


class FakeResponse:
    def __init__(self, status):
        self.status = status
        self.reason = "test"


def test_pandoc_produces_a_real_docx(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nSome body text.\n")
    out = md_to_docx(md, tmp_path / "out.docx")
    assert out.exists()
    assert out.read_bytes()[:2] == b"PK"  # docx is a zip


def test_generate_uploads_and_reports_the_new_document(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nBody.\n")
    drive = MagicMock()
    drive.files().create.return_value.execute.return_value = {
        "id": "1NewDoc",
        "webViewLink": "https://docs.google.com/document/d/1NewDoc/edit",
    }
    result = generate(drive, md, "Policy v4", tmp_path / "v4.docx", folder_id="0AFolder")
    assert result.doc_id == "1NewDoc"
    assert result.link.endswith("/edit")
    assert result.reason is None


def test_generate_falls_back_to_local_docx_when_there_is_no_folder(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nBody.\n")
    drive = MagicMock()
    result = generate(drive, md, "Policy v4", tmp_path / "v4.docx", folder_id=None)
    assert result.doc_id is None
    assert result.docx_path.exists()
    assert "output_folder_id" in result.reason
    drive.files().create.assert_not_called()


def test_generate_falls_back_and_keeps_the_docx_when_upload_is_refused(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nBody.\n")
    drive = MagicMock()
    drive.files().create.return_value.execute.side_effect = HttpError(FakeResponse(403), b"{}")
    result = generate(drive, md, "Policy v4", tmp_path / "v4.docx", folder_id="0AFolder")
    assert result.doc_id is None
    assert result.docx_path.exists()
    assert "403" in result.reason


def test_generate_never_reports_a_document_it_did_not_create(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nBody.\n")
    drive = MagicMock()
    drive.files().create.return_value.execute.side_effect = HttpError(FakeResponse(500), b"{}")
    result = generate(drive, md, "Policy v4", tmp_path / "v4.docx", folder_id="0AFolder")
    assert result.doc_id is None and result.link is None


def test_missing_markdown_file_is_reported_before_anything_else(tmp_path):
    drive = MagicMock()
    with pytest.raises(FileNotFoundError):
        generate(drive, tmp_path / "missing.md", "X", tmp_path / "x.docx", folder_id=None)
```

- [ ] **Step 2: Run and confirm failure**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_generate.py -v`
Expected: FAIL with `ModuleNotFoundError`

- [ ] **Step 3: Write `tools/gdoc/generate.py`**

```python
"""Turn approved markdown into a new Google Doc.

One converter, two destinations. pandoc makes a .docx; Drive converts that .docx
into a native Doc on create. If the upload cannot happen, the .docx is already on
disk and stays there, so the fallback is this pipeline stopping one step early
rather than a second code path.
"""

import subprocess
from dataclasses import dataclass
from pathlib import Path

from googleapiclient.errors import HttpError
from googleapiclient.http import MediaFileUpload

from tools.gdoc.export import DOCX_MIME, PANDOC

GOOGLE_DOC_MIME = "application/vnd.google-apps.document"


@dataclass(frozen=True)
class Result:
    docx_path: Path
    doc_id: str | None = None
    link: str | None = None
    reason: str | None = None


def md_to_docx(md_path: Path, out_path: Path) -> Path:
    out_path.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        [PANDOC, str(md_path), "-o", str(out_path)],
        check=True,
        capture_output=True,
        text=True,
    )
    return out_path


def upload_as_gdoc(drive, docx_path: Path, name: str, folder_id: str) -> dict:
    media = MediaFileUpload(str(docx_path), mimetype=DOCX_MIME, resumable=False)
    return (
        drive.files()
        .create(
            body={"name": name, "mimeType": GOOGLE_DOC_MIME, "parents": [folder_id]},
            media_body=media,
            fields="id,webViewLink",
            supportsAllDrives=True,
        )
        .execute()
    )


def generate(drive, md_path: Path, name: str, out_path: Path, folder_id: str | None) -> Result:
    """Convert, then try to upload. Never claim a document that was not created."""
    if not md_path.exists():
        raise FileNotFoundError(f"markdown file not found: {md_path}")
    docx_path = md_to_docx(md_path, out_path)
    if not folder_id:
        return Result(
            docx_path=docx_path,
            reason=(
                "no output_folder_id in config, so nothing was uploaded. "
                f"The document is ready at {docx_path}"
            ),
        )
    try:
        created = upload_as_gdoc(drive, docx_path, name, folder_id)
    except HttpError as error:
        return Result(
            docx_path=docx_path,
            reason=(
                f"upload refused with {error.resp.status}. Check that the service "
                f"account is a Content manager on folder {folder_id}. "
                f"The document is ready at {docx_path}"
            ),
        )
    return Result(
        docx_path=docx_path,
        doc_id=created["id"],
        link=created.get("webViewLink"),
    )
```

- [ ] **Step 4: Run and confirm passing**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_generate.py -v`
Expected: 6 passed

- [ ] **Step 5: Ignore the docx output directory**

Append to `.gitignore`:

```
docs/gdoc/*/out/
```

- [ ] **Step 6: Commit**

```bash
git add tools/gdoc/ .gitignore
git commit -m "feat: generate a new doc version via pandoc, with a local fallback"
```

---

### Task 12: The CLI

**Files:**
- Create: `tools/gdoc/cli.py`
- Create: `tools/gdoc/tests/test_cli.py`

**Interfaces:**
- Consumes: everything from Tasks 1 to 11
- Produces: `python -m tools.gdoc.cli <subcommand>` with subcommands `read`, `reply`, `export`, `capture`, `generate`; every subcommand prints one JSON object to stdout and returns 0 or 1

- [ ] **Step 1: Write the failing test**

`tools/gdoc/tests/test_cli.py`:

```python
import json
from unittest.mock import MagicMock, patch

import pytest

from tools.gdoc.cli import main


def test_read_prints_partitioned_threads_as_json(capsys):
    threads = [
        {
            "id": "t1",
            "content": "ai: rephrase",
            "author": {"displayName": "Nail Khusnullin", "me": False},
            "quotedFileContent": {"value": "asdasd"},
        },
        {
            "id": "t2",
            "content": "ai: check this",
            "author": {"displayName": "William Mejia", "me": False},
        },
    ]
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [{"comments": threads}]
    drive.files().get.return_value.execute.return_value = {"name": "Test doc"}
    with patch("tools.gdoc.cli.drive_service", return_value=drive), patch(
        "tools.gdoc.cli.load_config"
    ) as config:
        config.return_value.display_name = "Nail Khusnullin"
        exit_code = main(["read", "https://docs.google.com/document/d/1AbC/edit"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert [t["id"] for t in payload["mine"]] == ["t1"]
    assert [t["id"] for t in payload["others"]] == ["t2"]
    assert payload["doc_id"] == "1AbC"
    assert payload["slug"] == "test-doc"


def test_reply_reads_the_body_from_a_file(capsys, tmp_path):
    body = tmp_path / "body.txt"
    body.write_text("Plain text answer.")
    drive = MagicMock()
    drive.replies().create.return_value.execute.return_value = {"id": "r1"}
    with patch("tools.gdoc.cli.drive_service", return_value=drive):
        exit_code = main(["reply", "1AbC", "t1", "--body-file", str(body)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["reply_id"] == "r1"


def test_reply_rejects_markdown_and_exits_nonzero(capsys, tmp_path):
    body = tmp_path / "body.txt"
    body.write_text("Has **markdown**.")
    drive = MagicMock()
    with patch("tools.gdoc.cli.drive_service", return_value=drive):
        exit_code = main(["reply", "1AbC", "t1", "--body-file", str(body)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "markdown" in payload["error"]
    drive.replies().create.assert_not_called()


def test_bad_url_exits_nonzero_with_json_error(capsys):
    exit_code = main(["read", "https://example.com/nope"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "error" in payload


def test_unknown_subcommand_exits_nonzero():
    with pytest.raises(SystemExit):
        main(["nonsense"])
```

- [ ] **Step 2: Run and confirm failure**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/test_cli.py -v`
Expected: FAIL with `ModuleNotFoundError`

- [ ] **Step 3: Write `tools/gdoc/cli.py`**

```python
"""One CLI for every Google side effect.

Everything prints a single JSON object so the skill can read the result without
parsing prose. Reply bodies come from a file rather than an argument, because
they are multi-line plain text and shell quoting would mangle them.
"""

import argparse
import json
import sys
from dataclasses import asdict
from datetime import date
from pathlib import Path

from googleapiclient.errors import HttpError

from tools.gdoc.auth import drive_service
from tools.gdoc.config import load_config
from tools.gdoc.docid import extract_doc_id
from tools.gdoc.export import export_markdown
from tools.gdoc.fetch import fetch_threads
from tools.gdoc.filters import forced_kind, partition
from tools.gdoc.generate import generate
from tools.gdoc.mirror import slugify, write_mirror
from tools.gdoc.model import Thread
from tools.gdoc.pending import append_item
from tools.gdoc.reply import post_reply


def _thread_json(thread: Thread) -> dict:
    return {
        "id": thread.id,
        "content": thread.content,
        "author": thread.author_name,
        "quoted": thread.quoted,
        "anchored": thread.is_anchored,
        "forced_kind": forced_kind(thread.content),
    }


def _emit(payload: dict) -> int:
    print(json.dumps(payload, indent=2))
    return 0


def _fail(message: str) -> int:
    print(json.dumps({"error": message}, indent=2))
    return 1


def cmd_read(args) -> int:
    doc_id = extract_doc_id(args.url)
    drive = drive_service()
    config = load_config()
    threads = fetch_threads(drive, doc_id)
    mine, others, skipped = partition(threads, config.display_name)
    meta = drive.files().get(fileId=doc_id, fields="name").execute()
    return _emit(
        {
            "doc_id": doc_id,
            "name": meta.get("name"),
            "slug": slugify(meta.get("name", "")),
            "mine": [_thread_json(t) for t in mine],
            "others": [_thread_json(t) for t in others],
            "skipped": [_thread_json(t) for t in skipped],
        }
    )


def cmd_reply(args) -> int:
    body = Path(args.body_file).read_text()
    reply_id = post_reply(drive_service(), extract_doc_id(args.doc), args.comment_id, body)
    return _emit({"reply_id": reply_id, "comment_id": args.comment_id})


def cmd_export(args) -> int:
    doc_id = extract_doc_id(args.url)
    drive = drive_service()
    markdown = export_markdown(drive, doc_id)
    meta = drive.files().get(fileId=doc_id, fields="name").execute()
    slug = args.slug or slugify(meta.get("name", ""))
    path = write_mirror(Path(args.repo_root), slug, markdown, force=args.force)
    return _emit({"doc_id": doc_id, "slug": slug, "path": str(path), "characters": len(markdown)})


def cmd_capture(args) -> int:
    drive = drive_service()
    doc_id = extract_doc_id(args.doc)
    threads = {t.id: t for t in fetch_threads(drive, doc_id)}
    thread = threads.get(args.comment_id)
    if thread is None:
        return _fail(f"comment {args.comment_id} not found on {doc_id}")
    number = append_item(
        Path(args.repo_root), args.slug, thread, doc_id=doc_id, today=date.today().isoformat()
    )
    return _emit({"item": number, "comment_id": args.comment_id})


def cmd_generate(args) -> int:
    config = load_config()
    result = generate(
        drive_service(),
        Path(args.md),
        args.name,
        Path(args.out),
        folder_id=args.folder_id or config.output_folder_id,
    )
    payload = {k: (str(v) if isinstance(v, Path) else v) for k, v in asdict(result).items()}
    return _emit(payload)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="gdoc")
    sub = parser.add_subparsers(dest="command", required=True)

    read = sub.add_parser("read", help="list comment threads, partitioned")
    read.add_argument("url")
    read.set_defaults(func=cmd_read)

    reply = sub.add_parser("reply", help="post one plain-text reply")
    reply.add_argument("doc")
    reply.add_argument("comment_id")
    reply.add_argument("--body-file", required=True)
    reply.set_defaults(func=cmd_reply)

    export = sub.add_parser("export", help="write the markdown mirror")
    export.add_argument("url")
    export.add_argument("--repo-root", default=".")
    export.add_argument("--slug")
    export.add_argument("--force", action="store_true")
    export.set_defaults(func=cmd_export)

    capture = sub.add_parser("capture", help="append a global item to pending.md")
    capture.add_argument("doc")
    capture.add_argument("comment_id")
    capture.add_argument("--slug", required=True)
    capture.add_argument("--repo-root", default=".")
    capture.set_defaults(func=cmd_capture)

    gen = sub.add_parser("generate", help="markdown to a new Google Doc")
    gen.add_argument("--md", required=True)
    gen.add_argument("--name", required=True)
    gen.add_argument("--out", required=True)
    gen.add_argument("--folder-id")
    gen.set_defaults(func=cmd_generate)

    return parser


def main(argv=None) -> int:
    args = build_parser().parse_args(argv if argv is not None else sys.argv[1:])
    try:
        return args.func(args)
    except (ValueError, FileNotFoundError, RuntimeError) as error:
        return _fail(str(error))
    except HttpError as error:
        return _fail(f"Drive API {error.resp.status}: {error.reason}")


if __name__ == "__main__":
    sys.exit(main())
```

- [ ] **Step 4: Run and confirm passing**

Run: `~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/ -v`
Expected: all unit tests pass

- [ ] **Step 5: Check coverage meets 80%**

```bash
~/.config/gdoc-agent/venv/bin/pip install --timeout 180 --retries 10 pytest-cov
~/.config/gdoc-agent/venv/bin/python -m pytest tools/gdoc/tests/ --cov=tools/gdoc --cov-report=term-missing
```

If any module is below 80%, add tests for the uncovered branches before committing.

- [ ] **Step 6: Smoke test the CLI against the real document**

```bash
~/.config/gdoc-agent/venv/bin/python -m tools.gdoc.cli read 1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw
```

Expected: JSON with three entries under `skipped`, because all three test comments already have replies from the service account, and empty `mine` and `others`. That result is itself the idempotency proof.

- [ ] **Step 7: Commit**

```bash
git add tools/gdoc/
git commit -m "feat: gdoc CLI with JSON output"
```

---

### Task 13: The review skill

**Files:**
- Create: `~/.claude/skills/gdoc-review/SKILL.md`
- Create: `tools/gdoc/skills/gdoc-review/SKILL.md` (tracked copy)

**Interfaces:**
- Consumes: the CLI from Task 12
- Produces: the `/gdoc-review` entry point

- [ ] **Step 1: Write the skill file**

`~/.claude/skills/gdoc-review/SKILL.md`:

````markdown
---
name: gdoc-review
description: Use when Nail gives a Google Doc link and wants his ai: comments handled. Reads the comments, answers local ones in their threads, captures global ones for a later session.
---

# Google Docs review

Answer the `ai:` comments Nail left in a document. Local changes get an answer in
the thread. Global changes get captured, never attempted.

The marker is `ai` plus a sign, at the start of a comment: `ai:` leaves the
choice to you, `ai?` means answer it here, `ai!` means capture it as global. The
old `@ai` form still counts. The CLI does this matching; you never re-derive it.

Spec: `~/src/altery/intelligence-hub/docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md`

## Setup

```bash
GDOC="$HOME/.config/gdoc-agent/venv/bin/python -m tools.gdoc.cli"
```

Run the CLI from `~/src/altery/intelligence-hub`, but keep Nail's own working
directory as the corpus you search. The cwd he chose is deliberate: it decides
which notes ground the answers.

## Step 1: Read the comments

```bash
$GDOC read <url>
```

Returns `mine`, `others` and `skipped`. Threads already answered by the service
account appear under `skipped`, which is what makes a second run safe.

## Step 2: Show Nail what you found, and stop

Print the lists and ask before posting anything. Two things Nail must be told
every run, because the agent cannot check either one:

- The Drive API returns no email address for comment authors, so `mine` is a
  display-name guess, not proof.
- `permissions.list` is refused under Commenter, so the agent cannot see who
  else can read this document. Replies are visible to all of them.

```
Yours (will act):      para 3, para 7, para 11
Others (context only): para 5 (William Mejia)
Already answered:      para 2

I cannot see who else has access to this document. Replies will be
visible to everyone on it, under the service account address.

Proceed?
```

Wait for an answer. Never post before this.

## Step 3: Classify each of Nail's comments

Per comment, not per batch. One run may answer two and capture two.

| | Local | Global |
|---|---|---|
| Test | The change fits inside the quoted span | It touches text the comment does not quote |
| Examples | A question. Rephrase this. Is this term right? Add a missing clause | Renumber sections. Restructure. Apply a term change everywhere |
| Action | Answer in the thread | Capture, and reply with the refusal |

`forced_kind` in the JSON carries Nail's override: `ai?` means answer it in the
thread, `ai!` means capture it. `ai:` means he left the choice to you. Honour an
override without re-deciding.

An unanchored comment has `anchored: false` and no quote. It refers to the
document as a whole, so treat it as global unless it is plainly a question.

If a comment is genuinely ambiguous, ask Nail in the terminal. Do not guess.

## Step 4: Ground the answer

Search Nail's current repo. Cite plain file paths. The corpus mixes Russian and
English, so search in both languages.

If the repo has no source for the answer, say so in the reply. Never write a
plausible sentence to fill the gap.

## Step 5: Write the reply

Plain text only. Docs comment threads show markdown source as typed, so `**`,
backticks and `#` headings arrive broken. The CLI refuses them, which is a
safeguard, not permission to try.

Shape:
1. The replacement text or the answer, first, so the first thing Nail sees is
   the thing he copies.
2. One short reason line, only if the reason is not obvious.
3. Sources as plain paths, last.

Write it to a file and post it:

```bash
cat > /tmp/reply.txt <<'EOF'
<the reply>
EOF
$GDOC reply <doc_id> <comment_id> --body-file /tmp/reply.txt
```

## Step 6: Capture global items

```bash
$GDOC capture <doc_id> <comment_id> --slug <slug> --repo-root .
```

Then post the refusal, using the item number the capture returned:

```
Understood. This needs changes across the document, so I am not proposing text
here. Captured as item 2. I will handle it in a terminal session and produce a
new version of the document.
```

## Step 7: Mirror, only if paired

Mirror only when the document has a paired markdown file, or when Nail owns it
and asks for one. Never mirror a document he does not own: counsel drafts and
partner documents stay in Drive.

```bash
$GDOC export <url> --repo-root .
```

If it reports a mirror conflict, the markdown has uncommitted edits. Tell Nail
and let him decide. Do not pass `--force` on your own.

## Step 8: Report

```
posted   para 3   answered, cited domains/regulatory/cbc-emi.md
posted   para 7   rephrased, ready to paste
captured para 2   global: renumber sections

1 global item in docs/gdoc/<slug>/pending.md
Next session: /gdoc-apply docs/gdoc/<slug>/pending.md
```

## Never

- Never edit the reviewed document. The credential cannot, and neither may you.
- Never resolve a thread. Resolving means Nail accepted the text.
- Never reply twice to the same comment.
- Never act on a comment that is not Nail's.
- Never attempt a global change in a comment thread.
````

- [ ] **Step 2: Verify the skill is discovered**

Ask Nail to start a new session and type `/gdoc-review`. Confirm Claude Code
loads the skill rather than reporting an unknown command.

- [ ] **Step 3: Commit a tracked copy**

The skill lives outside git, so keep a copy in the repo for history:

```bash
mkdir -p tools/gdoc/skills/gdoc-review
cp ~/.claude/skills/gdoc-review/SKILL.md tools/gdoc/skills/gdoc-review/SKILL.md
git add tools/gdoc/skills/
git commit -m "feat: gdoc-review skill procedure"
```

---

### Task 14: The apply skill

**Files:**
- Create: `~/.claude/skills/gdoc-apply/SKILL.md`
- Create: `tools/gdoc/skills/gdoc-apply/SKILL.md` (tracked copy)

**Interfaces:**
- Consumes: the CLI from Task 12, `pending.md` from Task 10, `pairing.py` from Task 8
- Produces: the `/gdoc-apply` entry point

- [ ] **Step 1: Write the skill file**

`~/.claude/skills/gdoc-apply/SKILL.md`:

````markdown
---
name: gdoc-apply
description: Use when Nail wants the captured global items from a Google Doc review applied. Edits the paired markdown, then generates a new Google Doc version from it.
---

# Apply global doc items

Work through `pending.md` with Nail, edit the paired markdown, then generate a
new document from it. The original document is never edited.

## Step 1: Load the context

Read three things: the `pending.md` given as the argument, the `mirror.md` beside
it, and the paired markdown file whose frontmatter `gdoc` key holds the document
id named at the top of `pending.md`.

If there is no paired markdown file, the document is one Nail does not own. Then
the output is a note for him, not a new document. Say so and stop.

## Step 2: Work through the items in order

One item at a time. For each:

1. Say what you are about to change and where.
2. Make the edit in the **markdown file**, never in the mirror.
3. Show Nail the diff for that item alone.
4. Wait for approval before the next item.

The markdown is what he approves. That is the whole reason global changes are
not attempted in comment threads: a diff is reviewable and a comment thread is
not.

## Step 3: Commit the markdown

```bash
git add <paired md file>
git commit -m "docs: apply global items from <doc name> review"
```

Commit before generating, so every generated document corresponds to a commit.

## Step 4: Generate the new version

```bash
GDOC="$HOME/.config/gdoc-agent/venv/bin/python -m tools.gdoc.cli"
$GDOC generate --md <paired md file> --name "<doc name> v<n>" \
  --out docs/gdoc/<slug>/out/v<n>.docx
```

Two outcomes, both fine:

- `doc_id` and `link` present: the new Google Doc exists. Give Nail the link.
- `doc_id` null and a `reason`: the `.docx` is on disk at `docx_path`. Give him
  that path and the reason. Do not retry silently.

## Step 5: Record the version

Add the new id and today's date to `gdoc_versions` in the paired markdown file's
frontmatter, then commit. Never record a version that was not created.

## Step 6: Clear the items

Delete the applied items from `pending.md`, or delete the file if all are done.
Commit that too, so the record of what was asked stays in history.

## Never

- Never edit the original Google Doc.
- Never generate before the markdown is committed.
- Never write a `gdoc_versions` entry for a document that failed to upload.
````

- [ ] **Step 2: Commit the tracked copy**

```bash
mkdir -p tools/gdoc/skills/gdoc-apply
cp ~/.claude/skills/gdoc-apply/SKILL.md tools/gdoc/skills/gdoc-apply/SKILL.md
git add tools/gdoc/skills/
git commit -m "feat: gdoc-apply skill procedure"
```

---

### Task 15: End-to-end verification

**Files:**
- Modify: `docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md` (sections 10 and 16)

**Interfaces:**
- Consumes: everything
- Produces: a spec that records what is proven, and a list of anything that failed

- [ ] **Step 1: Prepare a clean test**

Ask Nail to resolve the three existing test comments and leave five fresh ones on
the test document, using the new marker:

| Comment | Purpose |
|---|---|
| `ai? <a question about the quoted text>` | forced question |
| `ai: <asks for better wording>` | agent decides, should answer locally |
| `ai! renumber the sections` | forced instruction, should be captured |
| an unanchored `ai: ...` on the whole document | closes the last untested path in the spec |
| `AI tools are changing this` | a decoy, must be ignored |

The decoy matters. The marker starts with a plain letter now, so the test that
it does not fire on ordinary prose is worth running for real, not just in unit
tests.

- [ ] **Step 2: Run the skill**

Invoke `/gdoc-review https://docs.google.com/document/d/1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw/edit`
from `~/src/altery/intelligence-hub`.

- [ ] **Step 3: Check each behaviour**

- [ ] The confirmation prompt appeared before anything was posted
- [ ] The question was answered with a file path cited
- [ ] The rephrase reply put the replacement text first
- [ ] The structural request was captured, not answered with text
- [ ] The unanchored comment was handled without crashing
- [ ] The decoy comment was left alone
- [ ] `pending.md` exists with the right item number
- [ ] No thread was resolved
- [ ] The document itself is unchanged

- [ ] **Step 4: Prove idempotency**

Run the same command again. Expected: every comment reports as already answered,
and nothing is posted.

- [ ] **Step 5: Generate, for real and then in fallback**

Run `/gdoc-apply` on the pending file. With `output_folder_id` set to the shared
drive folder, expect a link to a new document in "test folder". Open it and check
the headings and lists survived the pandoc conversion.

Then prove the fallback still works: run `generate` once more with
`--folder-id 1invalidFolderIdForTesting`. Expect no `doc_id`, a `.docx` on disk,
a reason naming the folder, and no `gdoc_versions` entry.

- [ ] **Step 6: Update the spec**

Rewrite section 10's "Verified" table with what is now proven, including the
unanchored comment result. Remove from section 16 anything that is now settled.

- [ ] **Step 7: Commit**

```bash
git add docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md
git commit -m "docs: record end-to-end verification of the gdoc review skill"
```

---

## Still needed from Nail

Both blockers from the first draft are cleared. The share is Commenter and the
output folder is chosen. Two smaller things remain, neither blocking the start.

1. **Approve the write attempts in Task 3.** Claude Code's auto mode blocks a command that would rename the document or insert text into it. Those two attempts are the test. Blocks Task 3 only.
2. **Pick a permanent output folder** when the test one has served its purpose. The current value is a folder literally named "test folder". Changing it later is one line in `~/.config/gdoc-agent/config.json`.

Folder `1KmUrVKL0-4QjeifT-SrENmAwgUcz5SPA` ("test doc folder") is not used. It is
in My Drive, so files created there would be owned by the service account rather
than by the shared drive. If Nail wants output next to the source document, that
trade-off needs a decision first.

## Deferred to v2

Not in this plan, and specified in the spec at commit `1615eb2` if they are ever
wanted: automatic triggering by polling or Workspace Events, the per-tier OS user
sandbox, and the `policy.yaml` tier map. All of it exists to defend an automatic
trigger, which v1 does not have.
