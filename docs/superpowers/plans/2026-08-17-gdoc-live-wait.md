# gdoc live: wait, progress, and the session loop — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `/gdoc-live <url>` session that answers new `ai:` comments within ~6 seconds, shows a single self-rewriting progress line per thread while it works, and costs nothing while idle.

**Architecture:** A new blocking CLI command `gdoc wait` polls `comments.list` every 5s and exits the moment a turn needs action, so the agent sleeps in a background Bash call between comments. Actionability becomes turn-based (a new `ai:` message with no agent answer after it), and progress replies carry a `⏳ ` prefix so they never count as answers. The `/gdoc-live` skill is a loop: sweep, wait, handle, repeat.

**Tech Stack:** Python 3, existing `google-api-python-client` Drive v3 wiring, pytest with `MagicMock` fakes, argparse CLI in `gdoc/cli.py`.

**Spec:** `docs/superpowers/specs/2026-08-17-gdoc-live-realtime-design.md` (aims at `docs/superpowers/specs/2026-08-17-gdoc-live-press-release.md`)

## Global Constraints

- Poll interval default **5 seconds**, `wait` timeout default **570 seconds** (stays under the Bash tool's 600s foreground cap).
- Backoff on Drive 403 rate-limit / 429 / 5xx / network errors: start 5s, double to max 60s, reset on success. After **300 seconds** of unbroken failure, give up with a message.
- The progress prefix is exactly `"⏳ "` (U+23F3 + space). It is data, defined once in `gdoc/model.py`.
- Comment replies are plain text. `assert_plain_text` guards every posted body, progress included.
- No new pip dependency, no new OAuth scope, no new Google API, no external program.
- Batch behaviour is untouched: `gdoc read` and `needs_action` keep their exact current semantics.
- Every command prints one JSON object (except documented raw outputs). Exit codes for `wait`: 0 found, 3 timeout, 4 document unavailable, 1 persistent failure.
- Immutability: frozen dataclasses, tuples over lists, no mutation of inputs.
- Never edit the reviewed document. Never delete anyone else's comment or reply.
- Plain short English in all docs and messages. No em dashes.
- Run tests with `~/.config/gdoc-agent/venv/bin/pytest`. Commit after each task. Never push.

---

### Task 1: The progress marker lives in the model

**Files:**
- Modify: `gdoc/model.py` (add `PROGRESS_PREFIX`, `Reply.is_progress`)
- Test: `tests/test_model.py`

**Interfaces:**
- Produces: `gdoc.model.PROGRESS_PREFIX: str` (== `"⏳ "`), `Reply.is_progress -> bool` property. Tasks 2, 3, and 6 rely on both names exactly.

- [ ] **Step 1: Write the failing tests**

Append to `tests/test_model.py`:

```python
from gdoc.model import PROGRESS_PREFIX, Reply


def test_progress_prefix_is_the_hourglass():
    assert PROGRESS_PREFIX == "⏳ "


def test_reply_with_prefix_is_progress():
    reply = Reply(id="r1", content=PROGRESS_PREFIX + "reading the notes", by_agent=True)
    assert reply.is_progress


def test_reply_without_prefix_is_not_progress():
    reply = Reply(id="r1", content="the answer", by_agent=True)
    assert not reply.is_progress


def test_prefix_mid_text_is_not_progress():
    reply = Reply(id="r1", content="done ⏳ waiting", by_agent=True)
    assert not reply.is_progress
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_model.py -v`
Expected: FAIL with `ImportError: cannot import name 'PROGRESS_PREFIX'`

- [ ] **Step 3: Write minimal implementation**

In `gdoc/model.py`, add after the module docstring:

```python
# The one visible mark that a reply is a live progress line, not an answer.
# Filters use it to keep a working thread open; the skill deletes marked
# replies when the answer lands. Defined here because it is part of what a
# Reply means, not part of how one is posted.
PROGRESS_PREFIX = "⏳ "
```

Add to the `Reply` dataclass:

```python
    @property
    def is_progress(self) -> bool:
        return self.by_agent and self.content.startswith(PROGRESS_PREFIX)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_model.py -v`
Expected: PASS, all of them

- [ ] **Step 5: Commit**

```bash
git add gdoc/model.py tests/test_model.py
git commit -m "feat: a reply knows whether it is a progress line"
```

---

### Task 2: Turn-based actionability for live sessions

**Files:**
- Modify: `gdoc/filters.py` (add `needs_action_live`, `partition_live`; change nothing existing)
- Test: `tests/test_filters.py`

**Interfaces:**
- Consumes: `Thread`, `Reply`, `PROGRESS_PREFIX` from `gdoc.model`; `is_addressed` already in `gdoc/filters.py`.
- Produces: `needs_action_live(thread: Thread) -> bool` and `partition_live(threads: tuple[Thread, ...]) -> tuple[tuple[Thread, ...], tuple[Thread, ...]]`. Task 4's wait loop and Task 6's sweep rely on `partition_live` exactly.

- [ ] **Step 1: Write the failing tests**

Append to `tests/test_filters.py` (it already imports `Thread`; add `Reply` and the new functions):

```python
from gdoc.filters import needs_action_live, partition_live
from gdoc.model import PROGRESS_PREFIX, Reply


def _thread(content, replies=(), resolved=False, by_agent=False):
    return Thread(
        id="t1",
        content=content,
        author_name="Nail",
        author_email=None,
        by_agent=by_agent,
        quoted=None,
        resolved=resolved,
        replies=tuple(replies),
    )


def _human(content):
    return Reply(id="rh", content=content, by_agent=False)


def _agent(content):
    return Reply(id="ra", content=content, by_agent=True)


def test_fresh_marked_thread_needs_action():
    assert needs_action_live(_thread("ai: is this number right"))


def test_answered_thread_is_done():
    thread = _thread("ai: is this number right", replies=[_agent("it is, source attached")])
    assert not needs_action_live(thread)


def test_new_marked_reply_reopens_an_answered_thread():
    thread = _thread(
        "ai: is this number right",
        replies=[_agent("it is"), _human("ai: then fix the note too")],
    )
    assert needs_action_live(thread)


def test_progress_reply_does_not_close_the_turn():
    thread = _thread(
        "ai: is this number right",
        replies=[_agent(PROGRESS_PREFIX + "checking the notes")],
    )
    assert needs_action_live(thread)


def test_unmarked_human_reply_does_not_reopen():
    thread = _thread(
        "ai: is this number right",
        replies=[_agent("it is"), _human("thanks, looks good")],
    )
    assert not needs_action_live(thread)


def test_resolved_thread_stays_closed():
    thread = _thread("ai: is this number right", resolved=True)
    assert not needs_action_live(thread)


def test_agents_own_thread_is_not_actionable():
    assert not needs_action_live(_thread("captured as item 2", by_agent=True))


def test_unmarked_thread_with_marked_reply_is_actionable():
    thread = _thread("just a note to self", replies=[_human("ai: check this claim")])
    assert needs_action_live(thread)


def test_partition_live_splits_by_the_live_rule():
    open_thread = _thread("ai: question")
    closed_thread = _thread("ai: question", replies=[_agent("answer")])
    actionable, skipped = partition_live((open_thread, closed_thread))
    assert actionable == (open_thread,)
    assert skipped == (closed_thread,)
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_filters.py -v`
Expected: FAIL with `ImportError: cannot import name 'needs_action_live'`

- [ ] **Step 3: Write minimal implementation**

Append to `gdoc/filters.py`:

```python
def needs_action_live(thread: Thread) -> bool:
    """True when the thread's latest ai:-marked human message has no answer yet.

    The batch rule (needs_action) closes a thread on any agent reply, which is
    right for one pass and wrong for a conversation. Here the unit is the turn:
    a marked human message reopens the thread, an agent answer closes it, and a
    progress line closes nothing because it is not an answer.
    """
    if thread.resolved:
        return False
    last_marked = 0 if (is_addressed(thread.content) and not thread.by_agent) else -1
    last_answer = -1
    for position, reply in enumerate(thread.replies, start=1):
        if reply.by_agent:
            if not reply.is_progress:
                last_answer = position
        elif is_addressed(reply.content):
            last_marked = position
    return last_marked > last_answer


def partition_live(
    threads: tuple[Thread, ...],
) -> tuple[tuple[Thread, ...], tuple[Thread, ...]]:
    """Split into (actionable, skipped) under the live turn rule."""
    actionable: list[Thread] = []
    skipped: list[Thread] = []
    for thread in threads:
        if needs_action_live(thread):
            actionable.append(thread)
        else:
            skipped.append(thread)
    return tuple(actionable), tuple(skipped)
```

Note the edge the tests pin down: `last_marked` starts at 0 only when the top-level comment itself is marked and human, so a thread with no marked message returns `-1 > -1`, false.

- [ ] **Step 4: Run tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_filters.py -v`
Expected: PASS, including every pre-existing test, unchanged

- [ ] **Step 5: Commit**

```bash
git add gdoc/filters.py tests/test_filters.py
git commit -m "feat: turn-based actionability for live sessions"
```

---

### Task 3: The reply lifecycle: progress, update, delete

**Files:**
- Modify: `gdoc/reply.py` (add `post_progress`, `update_reply`, `delete_reply`)
- Test: `tests/test_reply.py`

**Interfaces:**
- Consumes: `PROGRESS_PREFIX` from `gdoc.model`, `assert_plain_text` already in `gdoc/reply.py`.
- Produces: `post_progress(drive, doc_id, comment_id, text) -> str` (returns reply id, prefixes the marker itself), `update_reply(drive, doc_id, comment_id, reply_id, body) -> str` (returns reply id), `delete_reply(drive, doc_id, comment_id, reply_id) -> None`. Task 5's CLI and Task 6's sweep rely on these names exactly. Callers of `post_progress` pass unprefixed text.

- [ ] **Step 1: Write the failing tests**

Append to `tests/test_reply.py`:

```python
from gdoc.model import PROGRESS_PREFIX
from gdoc.reply import delete_reply, post_progress, update_reply


def test_post_progress_prefixes_the_marker():
    drive = MagicMock()
    drive.replies().create.return_value.execute.return_value = {"id": "r9"}
    assert post_progress(drive, "doc", "comment", "reading the notes") == "r9"
    body = drive.replies().create.call_args.kwargs["body"]
    assert body == {"content": PROGRESS_PREFIX + "reading the notes"}


def test_post_progress_refuses_markdown():
    drive = MagicMock()
    with pytest.raises(ValueError):
        post_progress(drive, "doc", "comment", "checking **everything**")
    drive.replies().create.assert_not_called()


def test_post_progress_refuses_a_second_prefix():
    drive = MagicMock()
    drive.replies().create.return_value.execute.return_value = {"id": "r9"}
    post_progress(drive, "doc", "comment", PROGRESS_PREFIX + "already marked")
    body = drive.replies().create.call_args.kwargs["body"]
    assert body["content"].count(PROGRESS_PREFIX) == 1


def test_update_reply_sends_the_new_body():
    drive = MagicMock()
    drive.replies().update.return_value.execute.return_value = {"id": "r9"}
    assert update_reply(drive, "doc", "comment", "r9", "new text") == "r9"
    kwargs = drive.replies().update.call_args.kwargs
    assert kwargs["fileId"] == "doc"
    assert kwargs["commentId"] == "comment"
    assert kwargs["replyId"] == "r9"
    assert kwargs["body"] == {"content": "new text"}


def test_update_reply_refuses_markdown():
    drive = MagicMock()
    with pytest.raises(ValueError):
        update_reply(drive, "doc", "comment", "r9", "`code`")
    drive.replies().update.assert_not_called()


def test_delete_reply_names_the_reply():
    drive = MagicMock()
    delete_reply(drive, "doc", "comment", "r9")
    kwargs = drive.replies().delete.call_args.kwargs
    assert kwargs == {"fileId": "doc", "commentId": "comment", "replyId": "r9"}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_reply.py -v`
Expected: FAIL with `ImportError: cannot import name 'delete_reply'`

- [ ] **Step 3: Write minimal implementation**

In `gdoc/reply.py`, add `from gdoc.model import PROGRESS_PREFIX` to the imports, then append:

```python
def post_progress(drive, doc_id: str, comment_id: str, text: str) -> str:
    """Post the live progress line. The caller passes unprefixed text.

    The prefix is added here, once, so a caller that already prefixed by
    mistake does not produce a double marker.
    """
    bare = text.removeprefix(PROGRESS_PREFIX)
    return post_reply(drive, doc_id, comment_id, PROGRESS_PREFIX + bare)


def update_reply(drive, doc_id: str, comment_id: str, reply_id: str, body: str) -> str:
    """Rewrite one of the agent's own replies in place."""
    assert_plain_text(body)
    updated = (
        drive.replies()
        .update(
            fileId=doc_id,
            commentId=comment_id,
            replyId=reply_id,
            body={"content": body},
            fields="id",
        )
        .execute()
    )
    return updated["id"]


def delete_reply(drive, doc_id: str, comment_id: str, reply_id: str) -> None:
    """Delete one of the agent's own replies. Drive refuses anyone else's."""
    drive.replies().delete(fileId=doc_id, commentId=comment_id, replyId=reply_id).execute()
```

Note: `post_reply` already calls `assert_plain_text`, so `post_progress` is guarded through it. The `fields="id"` on update matters: without a fields mask this endpoint can return 400.

- [ ] **Step 4: Run tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_reply.py -v`
Expected: PASS, all old and new

- [ ] **Step 5: Commit**

```bash
git add gdoc/reply.py tests/test_reply.py
git commit -m "feat: the agent can post, rewrite, and delete its own progress line"
```

---

### Task 4: The wait loop

**Files:**
- Create: `gdoc/wait.py`
- Test: `tests/test_wait.py` (new file)

**Interfaces:**
- Consumes: `fetch_threads` from `gdoc.fetch`, `partition_live` from `gdoc.filters`, `Thread` from `gdoc.model`.
- Produces:

```python
@dataclass(frozen=True)
class WaitResult:
    outcome: str                     # "found" or "timeout"
    actionable: tuple[Thread, ...]
    skipped: tuple[Thread, ...]

class DocumentUnavailable(RuntimeError): ...
class WaitGaveUp(RuntimeError): ...

def wait_for_action(drive, doc_id, *, interval=5.0, timeout=570.0,
                    sleep=time.sleep, clock=time.monotonic) -> WaitResult
```

Task 5's CLI relies on all five names exactly. `sleep` and `clock` are injected for tests only.

- [ ] **Step 1: Write the failing tests**

Create `tests/test_wait.py`:

```python
"""The wait loop, against a fake drive and a fake clock.

No test here sleeps for real: sleep advances the fake clock, so a 570-second
timeout runs in microseconds.
"""

from unittest.mock import MagicMock

import pytest
from googleapiclient.errors import HttpError

from gdoc.wait import DocumentUnavailable, WaitGaveUp, WaitResult, wait_for_action


class FakeClock:
    def __init__(self):
        self.now = 0.0

    def clock(self):
        return self.now

    def sleep(self, seconds):
        self.now += seconds


def _http_error(status, reason=b""):
    resp = MagicMock()
    resp.status = status
    resp.reason = "boom"
    return HttpError(resp, reason or b"boom")


def _comment(comment_id, content):
    return {
        "id": comment_id,
        "content": content,
        "author": {"displayName": "Nail", "me": False},
        "resolved": False,
        "replies": [],
    }


def _drive_returning(pages):
    """A fake drive whose comments.list returns each page dict in turn,
    repeating the last one forever."""
    drive = MagicMock()
    calls = {"n": 0}

    def execute():
        page = pages[min(calls["n"], len(pages) - 1)]
        calls["n"] += 1
        if isinstance(page, Exception):
            raise page
        return page

    drive.comments().list.return_value.execute.side_effect = execute
    return drive


def test_returns_immediately_when_work_is_already_waiting():
    fake = FakeClock()
    drive = _drive_returning([{"comments": [_comment("c1", "ai: check this")]}])
    result = wait_for_action(drive, "doc", sleep=fake.sleep, clock=fake.clock)
    assert result.outcome == "found"
    assert result.actionable[0].id == "c1"
    assert fake.now == 0.0


def test_polls_until_a_comment_appears():
    fake = FakeClock()
    drive = _drive_returning(
        [{"comments": []}, {"comments": []}, {"comments": [_comment("c1", "ai: now")]}]
    )
    result = wait_for_action(drive, "doc", interval=5.0, sleep=fake.sleep, clock=fake.clock)
    assert result.outcome == "found"
    assert fake.now == 10.0  # two empty polls, two sleeps


def test_times_out_with_nothing():
    fake = FakeClock()
    drive = _drive_returning([{"comments": []}])
    result = wait_for_action(
        drive, "doc", interval=5.0, timeout=12.0, sleep=fake.sleep, clock=fake.clock
    )
    assert result.outcome == "timeout"
    assert result.actionable == ()
    assert fake.now >= 12.0


def test_an_unmarked_comment_is_skipped_not_found():
    fake = FakeClock()
    drive = _drive_returning([{"comments": [_comment("c1", "note to self")]}])
    result = wait_for_action(
        drive, "doc", interval=5.0, timeout=7.0, sleep=fake.sleep, clock=fake.clock
    )
    assert result.outcome == "timeout"
    assert result.skipped[0].id == "c1"


def test_rate_limit_backs_off_and_recovers():
    fake = FakeClock()
    drive = _drive_returning(
        [
            _http_error(429),
            _http_error(429),
            {"comments": [_comment("c1", "ai: here")]},
        ]
    )
    result = wait_for_action(drive, "doc", interval=5.0, sleep=fake.sleep, clock=fake.clock)
    assert result.outcome == "found"
    # backoff slept 5 then 10, doubling from the interval
    assert fake.now == 15.0


def test_missing_document_raises_immediately():
    fake = FakeClock()
    drive = _drive_returning([_http_error(404)])
    with pytest.raises(DocumentUnavailable):
        wait_for_action(drive, "doc", sleep=fake.sleep, clock=fake.clock)


def test_persistent_failure_gives_up_after_five_minutes():
    fake = FakeClock()
    drive = _drive_returning([_http_error(429)])
    with pytest.raises(WaitGaveUp):
        wait_for_action(
            drive, "doc", interval=5.0, timeout=100000.0, sleep=fake.sleep, clock=fake.clock
        )
    assert fake.now >= 300.0


def test_network_error_backs_off_like_a_rate_limit():
    fake = FakeClock()
    drive = _drive_returning([OSError("connection reset"), {"comments": [_comment("c1", "ai: x")]}])
    result = wait_for_action(drive, "doc", interval=5.0, sleep=fake.sleep, clock=fake.clock)
    assert result.outcome == "found"
    assert fake.now == 5.0
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_wait.py -v`
Expected: FAIL with `ModuleNotFoundError: No module named 'gdoc.wait'`

- [ ] **Step 3: Write minimal implementation**

Create `gdoc/wait.py`:

```python
"""Block until a document has a turn that needs the agent.

This is the whole real-time mechanism: comments.list is polled on an
interval, and the command exits the moment the live filter finds work. The
caller (the gdoc-live skill) runs it in the background and pays nothing
while it blocks. See the 2026-08-17 realtime design spec.

Backoff doubles from the interval to BACKOFF_MAX on rate limits, server
errors, and network failures, and resets on any success. Failure that never
breaks for GIVE_UP_AFTER seconds raises WaitGaveUp: a wait that cannot see
the document should say so rather than look like a quiet margin.
"""

import time
from dataclasses import dataclass

from googleapiclient.errors import HttpError

from gdoc.fetch import fetch_threads
from gdoc.filters import partition_live
from gdoc.model import Thread

BACKOFF_MAX = 60.0
GIVE_UP_AFTER = 300.0

# Statuses that mean "try again later", not "the document is gone".
_TRANSIENT = {429, 500, 502, 503, 504}


class DocumentUnavailable(RuntimeError):
    """The document cannot be read at all: deleted, or access revoked."""


class WaitGaveUp(RuntimeError):
    """Every poll failed for GIVE_UP_AFTER seconds straight."""


@dataclass(frozen=True)
class WaitResult:
    outcome: str  # "found" or "timeout"
    actionable: tuple[Thread, ...]
    skipped: tuple[Thread, ...]


def _is_transient(error: HttpError) -> bool:
    if error.resp.status in _TRANSIENT:
        return True
    # 403 is both "slow down" and "you cannot see this". The reason string
    # is the only thing that tells them apart.
    if error.resp.status == 403:
        text = str(error)
        return "ratelimit" in text.lower().replace(" ", "").replace("_", "")
    return False


def wait_for_action(
    drive,
    doc_id: str,
    *,
    interval: float = 5.0,
    timeout: float = 570.0,
    sleep=time.sleep,
    clock=time.monotonic,
) -> WaitResult:
    started = clock()
    backoff = interval
    failing_since: float | None = None
    skipped: tuple[Thread, ...] = ()

    while True:
        try:
            threads = fetch_threads(drive, doc_id)
        except HttpError as error:
            if not _is_transient(error):
                raise DocumentUnavailable(
                    f"Drive API {error.resp.status} reading {doc_id}: {error.reason}"
                ) from error
            failing_since = clock() if failing_since is None else failing_since
            if clock() - failing_since + backoff >= GIVE_UP_AFTER:
                sleep(backoff)
                raise WaitGaveUp(
                    f"every poll failed for {GIVE_UP_AFTER:.0f}s straight "
                    f"(last: Drive API {error.resp.status})"
                ) from error
            sleep(backoff)
            backoff = min(backoff * 2, BACKOFF_MAX)
            continue
        except OSError as error:
            failing_since = clock() if failing_since is None else failing_since
            if clock() - failing_since + backoff >= GIVE_UP_AFTER:
                sleep(backoff)
                raise WaitGaveUp(
                    f"every poll failed for {GIVE_UP_AFTER:.0f}s straight (last: {error})"
                ) from error
            sleep(backoff)
            backoff = min(backoff * 2, BACKOFF_MAX)
            continue

        failing_since = None
        backoff = interval
        actionable, skipped = partition_live(threads)
        if actionable:
            return WaitResult(outcome="found", actionable=actionable, skipped=skipped)
        if clock() - started >= timeout:
            return WaitResult(outcome="timeout", actionable=(), skipped=skipped)
        sleep(interval)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_wait.py -v`
Expected: PASS. If the backoff arithmetic tests disagree with the implementation, fix the implementation to match the tests: the tests state the contract (5 then 10 on consecutive failures, reset on success).

- [ ] **Step 5: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest`
Expected: PASS, nothing else broke

- [ ] **Step 6: Commit**

```bash
git add gdoc/wait.py tests/test_wait.py
git commit -m "feat: wait_for_action blocks until a turn needs the agent"
```

---

### Task 5: `gdoc wait` and `gdoc progress` in the CLI

**Files:**
- Modify: `gdoc/cli.py` (two new subcommands)
- Test: `tests/test_cli.py`

**Interfaces:**
- Consumes: `wait_for_action`, `WaitResult`, `DocumentUnavailable`, `WaitGaveUp` from `gdoc.wait`; `post_progress`, `update_reply`, `delete_reply` from `gdoc.reply`; `_thread_json`, `_emit`, `_fail` already in `gdoc/cli.py`.
- Produces: `gdoc wait <url> [--interval] [--timeout]` exiting 0 found / 3 timeout / 4 unavailable / 1 gave up, printing `{"outcome", "doc_id", "actionable", "skipped"}` with thread JSON matching `gdoc read`'s shape. `gdoc progress <doc> <comment_id> --text T` creates (prints `{"reply_id"}`), `--reply-id R --text T` updates, `--reply-id R --clear` deletes. Task 7's skill text quotes these invocations verbatim.

- [ ] **Step 1: Write the failing tests**

Append to `tests/test_cli.py`. Follow the file's existing fake/monkeypatch style; the tests below assume `main` is imported from `gdoc.cli` and a `capsys` fixture, which the file already uses:

```python
import json

from gdoc import cli
from gdoc.model import Thread
from gdoc.wait import DocumentUnavailable, WaitGaveUp, WaitResult


def _wait_thread():
    return Thread(
        id="c1",
        content="ai: check this",
        author_name="Nail",
        author_email=None,
        by_agent=False,
        quoted=None,
        resolved=False,
        replies=(),
    )


def test_wait_found_prints_the_report_and_exits_zero(monkeypatch, capsys):
    monkeypatch.setattr(cli, "drive_service", lambda: object())
    monkeypatch.setattr(
        cli,
        "wait_for_action",
        lambda drive, doc_id, interval, timeout: WaitResult(
            outcome="found", actionable=(_wait_thread(),), skipped=()
        ),
    )
    code = cli.main(["wait", "https://docs.google.com/document/d/DOC123/edit"])
    assert code == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["outcome"] == "found"
    assert payload["actionable"][0]["id"] == "c1"


def test_wait_timeout_exits_three(monkeypatch, capsys):
    monkeypatch.setattr(cli, "drive_service", lambda: object())
    monkeypatch.setattr(
        cli,
        "wait_for_action",
        lambda drive, doc_id, interval, timeout: WaitResult(
            outcome="timeout", actionable=(), skipped=()
        ),
    )
    code = cli.main(["wait", "https://docs.google.com/document/d/DOC123/edit"])
    assert code == 3
    assert json.loads(capsys.readouterr().out)["outcome"] == "timeout"


def test_wait_unavailable_exits_four(monkeypatch, capsys):
    monkeypatch.setattr(cli, "drive_service", lambda: object())

    def boom(drive, doc_id, interval, timeout):
        raise DocumentUnavailable("Drive API 404 reading DOC123: notFound")

    monkeypatch.setattr(cli, "wait_for_action", boom)
    code = cli.main(["wait", "https://docs.google.com/document/d/DOC123/edit"])
    assert code == 4
    assert "404" in json.loads(capsys.readouterr().out)["error"]


def test_wait_gave_up_exits_one(monkeypatch, capsys):
    monkeypatch.setattr(cli, "drive_service", lambda: object())

    def boom(drive, doc_id, interval, timeout):
        raise WaitGaveUp("every poll failed for 300s straight")

    monkeypatch.setattr(cli, "wait_for_action", boom)
    code = cli.main(["wait", "https://docs.google.com/document/d/DOC123/edit"])
    assert code == 1


def test_progress_create(monkeypatch, capsys):
    monkeypatch.setattr(cli, "drive_service", lambda: object())
    monkeypatch.setattr(cli, "post_progress", lambda drive, d, c, t: "r7")
    code = cli.main(["progress", "DOC123", "c1", "--text", "reading the notes"])
    assert code == 0
    assert json.loads(capsys.readouterr().out) == {
        "reply_id": "r7", "comment_id": "c1", "action": "created",
    }


def test_progress_update(monkeypatch, capsys):
    monkeypatch.setattr(cli, "drive_service", lambda: object())
    seen = {}

    def fake_update(drive, doc_id, comment_id, reply_id, body):
        seen["body"] = body
        return reply_id

    monkeypatch.setattr(cli, "update_reply", fake_update)
    code = cli.main(["progress", "DOC123", "c1", "--reply-id", "r7", "--text", "still checking"])
    assert code == 0
    assert seen["body"].startswith("⏳ ")
    assert json.loads(capsys.readouterr().out)["action"] == "updated"


def test_progress_clear(monkeypatch, capsys):
    monkeypatch.setattr(cli, "drive_service", lambda: object())
    monkeypatch.setattr(cli, "delete_reply", lambda drive, d, c, r: None)
    code = cli.main(["progress", "DOC123", "c1", "--reply-id", "r7", "--clear"])
    assert code == 0
    assert json.loads(capsys.readouterr().out)["action"] == "cleared"


def test_progress_clear_without_reply_id_fails(capsys):
    code = cli.main(["progress", "DOC123", "c1", "--clear"])
    assert code == 1
    assert "reply-id" in json.loads(capsys.readouterr().out)["error"]
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -v`
Expected: FAIL. The wait tests fail with `AttributeError: module 'gdoc.cli' has no attribute 'wait_for_action'`; the progress tests fail on the unknown subcommand.

- [ ] **Step 3: Write minimal implementation**

In `gdoc/cli.py`:

Add to imports:

```python
from gdoc.model import PROGRESS_PREFIX, Thread
from gdoc.reply import delete_reply, post_progress, post_reply, update_reply
from gdoc.wait import DocumentUnavailable, WaitGaveUp, wait_for_action
```

(`post_reply` is already imported; fold the names into the existing lines.)

Add the handlers, next to `cmd_read`:

```python
def cmd_wait(args) -> int:
    """Block until the document has a turn for the agent. Exit 3 on timeout,
    4 when the document cannot be read, 1 when polling failed for too long."""
    doc_id = extract_doc_id(args.url)
    drive = drive_service()
    try:
        result = wait_for_action(
            drive, doc_id, interval=args.interval, timeout=args.timeout
        )
    except DocumentUnavailable as error:
        print(json.dumps({"error": str(error), "outcome": "unavailable"}, indent=2))
        return 4
    payload = {
        "outcome": result.outcome,
        "doc_id": doc_id,
        "actionable": [_thread_json(t) for t in result.actionable],
        "skipped": [_thread_json(t) for t in result.skipped],
    }
    _emit(payload)
    return 0 if result.outcome == "found" else 3


def cmd_progress(args) -> int:
    """Create, rewrite, or clear the one live progress line on a thread."""
    if args.clear and not args.reply_id:
        return _fail("--clear needs --reply-id: only an existing line can be cleared")
    if not args.clear and not args.text:
        return _fail("--text is required unless --clear is passed")
    doc_id = extract_doc_id(args.doc)
    drive = drive_service()
    if args.clear:
        delete_reply(drive, doc_id, args.comment_id, args.reply_id)
        return _emit(
            {"reply_id": args.reply_id, "comment_id": args.comment_id, "action": "cleared"}
        )
    if args.reply_id:
        bare = args.text.removeprefix(PROGRESS_PREFIX)
        update_reply(drive, doc_id, args.comment_id, args.reply_id, PROGRESS_PREFIX + bare)
        return _emit(
            {"reply_id": args.reply_id, "comment_id": args.comment_id, "action": "updated"}
        )
    reply_id = post_progress(drive, doc_id, args.comment_id, args.text)
    return _emit({"reply_id": reply_id, "comment_id": args.comment_id, "action": "created"})
```

Add to `build_parser()`, after the `read` block:

```python
    wait = sub.add_parser("wait", help="block until a comment needs the agent")
    wait.add_argument("url")
    wait.add_argument("--interval", type=float, default=5.0)
    wait.add_argument("--timeout", type=float, default=570.0)
    wait.set_defaults(func=cmd_wait)

    progress = sub.add_parser("progress", help="create, rewrite, or clear the live progress line")
    progress.add_argument("doc")
    progress.add_argument("comment_id")
    progress.add_argument("--text", help="the progress line, unprefixed")
    progress.add_argument("--reply-id", help="rewrite or clear this existing line")
    progress.add_argument("--clear", action="store_true", help="delete the line")
    progress.set_defaults(func=cmd_progress)
```

In `main()`, `WaitGaveUp` must exit 1 with the message. It is a `RuntimeError`, and `main` already maps `RuntimeError` to `_fail`, so no change is needed; verify the test passes rather than adding a handler.

- [ ] **Step 4: Run tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -v`
Expected: PASS, old and new

- [ ] **Step 5: Commit**

```bash
git add gdoc/cli.py tests/test_cli.py
git commit -m "feat: gdoc wait and gdoc progress commands"
```

---

### Task 6: `gdoc sweep` clears the agent's stale progress lines

**Files:**
- Modify: `gdoc/cli.py` (one new subcommand), `gdoc/wait.py` (one pure helper)
- Test: `tests/test_wait.py`, `tests/test_cli.py`

**Interfaces:**
- Consumes: `fetch_threads`, `delete_reply`, `Thread.replies`, `Reply.is_progress`.
- Produces: `stale_progress(threads: tuple[Thread, ...]) -> tuple[tuple[str, str], ...]` in `gdoc/wait.py` returning `(comment_id, reply_id)` pairs; `gdoc sweep <url>` deleting each pair and printing `{"swept": <count>, "doc_id": ...}`. Task 7's skill runs `sweep` on attach.

- [ ] **Step 1: Write the failing tests**

Append to `tests/test_wait.py`:

```python
from gdoc.model import PROGRESS_PREFIX as _PP, Reply, Thread
from gdoc.wait import stale_progress


def _thread_with(replies):
    return Thread(
        id="t1",
        content="ai: question",
        author_name="Nail",
        author_email=None,
        by_agent=False,
        quoted=None,
        resolved=False,
        replies=tuple(replies),
    )


def test_stale_progress_finds_only_marked_agent_replies():
    threads = (
        _thread_with(
            [
                Reply(id="r1", content=_PP + "leftover", by_agent=True),
                Reply(id="r2", content="a real answer", by_agent=True),
                Reply(id="r3", content=_PP + "not ours", by_agent=False),
            ]
        ),
    )
    assert stale_progress(threads) == (("t1", "r1"),)


def test_stale_progress_empty_when_clean():
    threads = (_thread_with([Reply(id="r2", content="answer", by_agent=True)]),)
    assert stale_progress(threads) == ()
```

Append to `tests/test_cli.py`:

```python
def test_sweep_deletes_each_stale_line(monkeypatch, capsys):
    monkeypatch.setattr(cli, "drive_service", lambda: object())
    monkeypatch.setattr(cli, "fetch_threads", lambda drive, doc_id: ())
    monkeypatch.setattr(cli, "stale_progress", lambda threads: (("c1", "r1"), ("c2", "r9")))
    deleted = []
    monkeypatch.setattr(
        cli, "delete_reply", lambda drive, d, c, r: deleted.append((c, r))
    )
    code = cli.main(["sweep", "https://docs.google.com/document/d/DOC123/edit"])
    assert code == 0
    assert deleted == [("c1", "r1"), ("c2", "r9")]
    assert json.loads(capsys.readouterr().out)["swept"] == 2
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_wait.py tests/test_cli.py -v`
Expected: FAIL with `ImportError: cannot import name 'stale_progress'`

- [ ] **Step 3: Write minimal implementation**

Append to `gdoc/wait.py`:

```python
def stale_progress(threads: tuple[Thread, ...]) -> tuple[tuple[str, str], ...]:
    """(comment_id, reply_id) for every progress line the agent left behind.

    Only the agent's own marked replies qualify: is_progress already requires
    by_agent, and Drive would refuse deleting anyone else's reply anyway.
    """
    return tuple(
        (thread.id, reply.id)
        for thread in threads
        for reply in thread.replies
        if reply.is_progress
    )
```

In `gdoc/cli.py`, add `stale_progress` to the `gdoc.wait` import line, then add the handler and parser entry:

```python
def cmd_sweep(args) -> int:
    """Delete the agent's own leftover progress lines. Run on attach."""
    doc_id = extract_doc_id(args.url)
    drive = drive_service()
    pairs = stale_progress(fetch_threads(drive, doc_id))
    for comment_id, reply_id in pairs:
        delete_reply(drive, doc_id, comment_id, reply_id)
    return _emit({"doc_id": doc_id, "swept": len(pairs)})
```

```python
    sweep = sub.add_parser("sweep", help="delete the agent's own stale progress lines")
    sweep.add_argument("url")
    sweep.set_defaults(func=cmd_sweep)
```

- [ ] **Step 4: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gdoc/wait.py gdoc/cli.py tests/test_wait.py tests/test_cli.py
git commit -m "feat: gdoc sweep clears stale progress lines on attach"
```

---

### Task 7: The `/gdoc-live` skill

**Files:**
- Create: `skills/gdoc-live/SKILL.md`
- Modify: `install.sh:15` (`SKILLS=(gdoc-review gdoc-apply)` becomes `SKILLS=(gdoc-review gdoc-apply gdoc-live)`)

**Interfaces:**
- Consumes: `gdoc wait`, `gdoc progress`, `gdoc sweep`, `gdoc read`, `gdoc reply`, `gdoc capture`, exactly as produced by Tasks 5 and 6.
- Produces: the live session loop Claude Code runs. No code relies on it; Nail does.

- [ ] **Step 1: Write the skill**

Create `skills/gdoc-live/SKILL.md`:

````markdown
---
name: gdoc-live
description: Use when Nail wants a live session on a Google Doc. Waits for new ai: comments, shows one rewriting progress line while working, answers in the thread, and loops until Nail detaches.
---

# Google Docs live session

Attach to one document and stay attached. Every new `ai:` comment is a prompt.
Answer it with everything the terminal has, show progress in the margin while
working, and go back to waiting. Detach is Nail ending the session.

Spec: `~/src/personal/gdoc/docs/superpowers/specs/2026-08-17-gdoc-live-realtime-design.md`.
The result it aims at: the press release beside it.

## Setup

```bash
GDOC="$HOME/.config/gdoc-agent/venv/bin/gdoc"
ROOT="$PWD"
```

Same meaning as in gdoc-review: `$ROOT` is the corpus, the pairing tree, and
where `.gdoc/` lives. It may not be a git repository, and nothing here needs one.

## Attach

1. Sweep stale progress lines from a previous session:

```bash
$GDOC sweep <url>
```

2. Catch up on everything already waiting:

```bash
$GDOC read <url>
```

3. Show Nail what is pending and say the standing warning once, at attach,
   not per reply:

```
Attached to <name>. N comments already waiting.
I cannot see who else has access. Replies are visible to everyone on the
document, under the service account address.
Handling the backlog, then watching for new ai: comments.
```

Handle the backlog with the same per-comment flow as below, then start the loop.
Do not ask permission again for comments that arrive during the session: Nail
attached to this document, and that was the decision.

## The loop

Run wait in the background and handle whatever it returns:

```bash
$GDOC wait <url> --timeout 570
```

- Exit 0: `actionable` has turns. Handle each one, then run wait again.
- Exit 3: timeout, nothing new. Run wait again. Say nothing.
- Exit 4: the document is gone or access was revoked. Tell Nail and stop.
- Exit 1: polling failed for five minutes straight. Tell Nail what the error
  was and ask whether to keep trying.

A turn can be a brand-new comment or a new `ai:` reply in a thread you already
answered. The JSON does not distinguish them; read the thread before answering.

## Handling one turn

1. Post the progress line the moment you start:

```bash
$GDOC progress <doc_id> <comment_id> --text "reading the notes"
```

Keep the reply_id it prints.

2. Work. Search `$ROOT`, run agents, use the skills, whatever the question
   needs. When the work changes phase, rewrite the line, never add a second:

```bash
$GDOC progress <doc_id> <comment_id> --reply-id <reply_id> --text "notes cite the older figure, confirming"
```

Progress is for a person glancing at a margin: say where the work is now,
not a log of where it has been. Never more than one line per thread.

3. Answer in the thread, plain text, same shape as gdoc-review: the answer
   first, one reason line if needed, sources as plain paths last.

```bash
$GDOC reply <doc_id> <comment_id> --body-file /tmp/reply.txt
```

Long answer: put the summary in the thread and write the full text to
`$ROOT/.gdoc/<slug>/answers/<date>-<comment_id>.md`, naming that path in
the summary. The margin gets what fits a margin.

4. Clear the progress line. The answer replaces it:

```bash
$GDOC progress <doc_id> <comment_id> --reply-id <reply_id> --clear
```

If a progress note turned out to be a finding, put it in the answer before
clearing it. Findings are promoted, never deleted.

5. Global changes are captured, never attempted, exactly as in gdoc-review:

```bash
$GDOC capture <doc_id> <comment_id>
```

Then post the standard refusal naming the item number, and clear the
progress line.

## Classification

Same table as gdoc-review: local changes get an answer in the thread, global
changes get captured. `ai?` forces an answer, `ai!` forces a capture, `ai:`
leaves it to you. An unanchored comment is global unless it is plainly a
question. Ambiguity is a question for Nail in the terminal, not a guess.

## Detach

Nail ends the session. Before stopping, if any turn is mid-flight, finish it
or clear its progress line. Report:

```
Detached from <name>.
Answered 4, captured 1 (item 3 in .gdoc/<slug>/pending.md).
Next: /gdoc-apply .gdoc/<slug>/pending.md
```

## Never

- Never edit the document. The credential cannot, and neither may you.
- Never resolve a thread. Resolving means Nail accepted the text.
- Never leave two agent progress lines on one thread.
- Never delete anything that is not your own progress line.
- Never act on a comment without the marker.
- Never attempt a global change in a comment thread.
- Never post markdown into a thread. The CLI refuses it for a reason.
````

- [ ] **Step 2: Update install.sh**

Change line 15 from:

```bash
SKILLS=(gdoc-review gdoc-apply)
```

to:

```bash
SKILLS=(gdoc-review gdoc-apply gdoc-live)
```

- [ ] **Step 3: Verify the install**

Run: `./install.sh`
Expected: prints the link for gdoc-live; `ls -la ~/.claude/skills/gdoc-live` shows a symlink into this repo's `skills/gdoc-live`.

- [ ] **Step 4: Commit**

```bash
git add skills/gdoc-live/SKILL.md install.sh
git commit -m "feat: the gdoc-live skill, a session that answers the margin"
```

---

### Task 8: Integration tests tell the narrower truth

**Files:**
- Modify: `tests/test_access_integration.py`

**Interfaces:**
- Consumes: the live Drive API, credentialed runs only; `post_progress`, `update_reply`, `delete_reply` from `gdoc.reply`.
- Produces: nothing for other tasks; the Commenter guarantee stays asserted.

- [ ] **Step 1: Rename the delete test to what it actually proves**

In `tests/test_access_integration.py`, the test at line 75 deletes a comment the agent did not author. Rename it and sharpen its docstring:

```python
def test_deleting_someone_elses_comment_is_refused(drive, test_doc_id):
    """The agent can delete only what it wrote itself.

    Verified 2026-08-16: its own replies and comments delete fine, which the
    live progress line depends on. Anyone else's comment is refused, which
    this asserts. comments[0] on the test document is human-authored.
    """
    res = drive.comments().list(
        fileId=test_doc_id, fields="comments(id,author(me))"
    ).execute()
    others = [c for c in res["comments"] if not c.get("author", {}).get("me")]
    assert others, "test document must hold at least one human-authored comment"
    with pytest.raises(HttpError) as excinfo:
        drive.comments().delete(fileId=test_doc_id, commentId=others[0]["id"]).execute()
    assert excinfo.value.resp.status in (403, 404)
```

- [ ] **Step 2: Add the progress lifecycle test**

Append to the same file:

```python
def test_the_agent_can_rewrite_and_delete_its_own_progress_line(drive, test_doc_id):
    """The live progress line, end to end: create, rewrite, delete, gone.

    Cleans up its own comment even on failure.
    """
    from gdoc.model import PROGRESS_PREFIX
    from gdoc.reply import delete_reply, post_progress, update_reply

    comment = drive.comments().create(
        fileId=test_doc_id, body={"content": "PROBE live progress"}, fields="id"
    ).execute()
    try:
        reply_id = post_progress(drive, test_doc_id, comment["id"], "reading the notes")
        update_reply(
            drive, test_doc_id, comment["id"], reply_id,
            PROGRESS_PREFIX + "confirming",
        )
        delete_reply(drive, test_doc_id, comment["id"], reply_id)
        thread = drive.comments().get(
            fileId=test_doc_id, commentId=comment["id"], fields="replies(id)"
        ).execute()
        assert thread.get("replies", []) == []
    finally:
        drive.comments().delete(fileId=test_doc_id, commentId=comment["id"]).execute()
```

- [ ] **Step 3: Run the integration tests**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_access_integration.py -v`
Expected: PASS with credentials present, SKIP without. Both are acceptable here; the credentialed run happened on 2026-08-16 and the plan executor may not have credentials.

- [ ] **Step 4: Commit**

```bash
git add tests/test_access_integration.py
git commit -m "test: the delete guarantee is about other people's comments"
```

---

### Task 9: Documentation

**Files:**
- Modify: `README.md` (Use section: add the live session; CLI list: add wait, progress, sweep)
- Modify: `CLAUDE.md` (What lives where: skills row mentions gdoc-live; add the live-session rules to Never if any are new)

**Interfaces:**
- Consumes: everything above, as shipped.
- Produces: docs that match the tool.

- [ ] **Step 1: Update README.md**

In the Use section, after the `/gdoc-apply` line, add:

```markdown
/gdoc-live <google doc url>
```

with one sentence:

```markdown
`/gdoc-live` keeps a session attached: new `ai:` comments are answered as they
arrive, with a single progress line in the thread while the agent works. See
the press release in `docs/superpowers/specs/2026-08-17-gdoc-live-press-release.md`
for what a session feels like.
```

In the direct CLI list, add:

```markdown
gdoc wait <url> --timeout 570             # block until a comment needs the agent
gdoc progress <doc_id> <comment_id> --text "..."   # the live progress line
gdoc sweep <url>                          # delete stale progress lines
```

- [ ] **Step 2: Update CLAUDE.md**

In the "What lives where" table, the `skills/` row becomes:

```markdown
| `skills/` | `gdoc-review`, `gdoc-apply` and `gdoc-live`. Symlinked into `~/.claude/skills/`, so edits are live |
```

Add one line under Never:

```markdown
- Never delete a comment or reply the agent did not write. Its own progress lines are the one exception, and the only deletable thing.
```

- [ ] **Step 3: Verify docs match the CLI**

Run: `~/.config/gdoc-agent/venv/bin/gdoc --help`
Expected: read, reply, export, capture, generate, pair, wait, progress, sweep all listed, matching what the README now says.

- [ ] **Step 4: Run the whole suite one last time**

Run: `~/.config/gdoc-agent/venv/bin/pytest`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: the live session, wait, progress, and sweep"
```

---

## First real session, manual, after the plan

Not a task for the executor; a note for Nail's first run.

- Measure comment-to-progress-line latency from the Docs UI against the ~6s
  expectation.
- Watch whether a rewritten reply shows an "(edited)" mark or bumps the
  thread. Both are UI cosmetics the API cannot show. If rewriting is ugly,
  the knob is fewer rewrites, not a design change.
