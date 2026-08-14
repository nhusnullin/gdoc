# gdoc-apply drains the document: Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `gdoc-apply` collects every change made to a document since the last version, from comments and from direct edits, and carries all of it into the next version.

**Architecture:** Three new read-only CLI commands do the mechanical work the skill must not improvise. `gdoc worklist` merges `pending.md` against a live read and returns five buckets. `gdoc drift` diffs `baseline.md` against a fresh export to find direct edits. `gdoc comment` posts the one top-level note on the superseded version. One filter rule changes so a human reply re-opens a thread. Everything else is skill text.

**Tech Stack:** Python 3, `google-api-python-client`, `difflib` from the standard library, pytest. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-08-14-gdoc-apply-drains-design.md`

## Global Constraints

- **One root.** `--repo-root` defaults to `$PWD` and means the tree the tool works in. Never a default that names a repository, never tool files written into this repo. New commands take `--repo-root` with the same default as `capture`.
- **The tool must work without git.** Nothing added here may refuse to run because git is unavailable.
- **No markdown in comment bodies.** Every posted body goes through `gdoc.reply.assert_plain_text`.
- **Never edit the reviewed document.** These commands read comments, post comments, and export. Nothing writes document content.
- **Writing style in every skill, doc and commit message:** plain short English, no em dashes.
- **Tests:** `~/.config/gdoc-agent/venv/bin/pytest`. Suite must stay green and at or above 80% coverage. TDD: the failing test comes first.
- **Two branch dependencies.** This plan is written against `main`.
  - `gdoc-oauth` supplies the `[gdoc]` reply marker. Task 1 uses `Reply.by_agent` and isolates the test in one property so the merge with that branch changes one line, not a rule scattered across files.
  - `gdoc-template` (PR #5) makes `generate` write the pairing into markdown frontmatter. Nothing in this plan blocks on it. `gdoc worklist` and `gdoc drift` resolve the slug through `find_by_doc_id`, exactly as `capture` does today, so they work before and after that branch lands.

## File Structure

| File | Responsibility |
|---|---|
| `gdoc/model.py` | Modify. `has_agent_reply` becomes `agent_replied_last` |
| `gdoc/filters.py` | Modify. `needs_action` uses the new property |
| `gdoc/pending.py` | Modify. Gains a reader: the file has only ever been written |
| `gdoc/worklist.py` | Create. The merge, pure, no network |
| `gdoc/drift.py` | Create. The baseline diff, pure, no network |
| `gdoc/reply.py` | Modify. Gains `post_comment`, and `GLOBAL_REFUSAL` becomes `CAPTURED_NOTE` |
| `gdoc/cli.py` | Modify. Three new subcommands, each a thin handler |
| `tests/test_worklist.py` | Create |
| `tests/test_drift.py` | Create |
| `tests/test_pending.py` | Modify. Reader cases |
| `tests/test_model.py`, `tests/test_filters.py`, `tests/test_reply.py`, `tests/test_cli.py` | Modify |
| `skills/gdoc-apply/SKILL.md` | Rewrite steps 1 and 2, add the closure step |
| `skills/gdoc-review/SKILL.md` | Replace the local/global table with one test |
| `README.md`, `CLAUDE.md` | Modify |

The merge and the diff are separate modules rather than functions in `cli.py`
because both are pure, both have real edge cases, and `cli.py` is already the
largest file in the package.

---

## Task 1: A reply re-opens a thread

**Files:**
- Modify: `gdoc/model.py:31-34`
- Modify: `gdoc/filters.py:44-53`
- Test: `tests/test_model.py:55-63`, `tests/test_filters.py`

**Interfaces:**
- Consumes: nothing.
- Produces: `Thread.agent_replied_last: bool`. `has_agent_reply` is gone, not deprecated. `needs_action(thread) -> bool` keeps its signature and changes its meaning.

- [ ] **Step 1: Write the failing tests**

Add to `tests/test_filters.py`:

```python
def test_agent_reply_closes_the_thread():
    replies = (Reply(id="r1", content="Yes, five days is right.", by_agent=True),)
    assert needs_action(thread(replies=replies)) is False


def test_a_human_reply_after_the_agent_re_opens_the_thread():
    """Answering the agent has to mean something, or the conversation is one-way."""
    replies = (
        Reply(id="r1", content="Yes, five days is right.", by_agent=True),
        Reply(id="r2", content="No, we agreed three in the PDR.", by_agent=False),
    )
    assert needs_action(thread(replies=replies)) is True


def test_the_agent_answering_again_closes_it_again():
    replies = (
        Reply(id="r1", content="Yes, five days.", by_agent=True),
        Reply(id="r2", content="No, three.", by_agent=False),
        Reply(id="r3", content="Corrected, three.", by_agent=True),
    )
    assert needs_action(thread(replies=replies)) is False


def test_a_human_reply_with_no_agent_reply_still_needs_action():
    replies = (Reply(id="r1", content="agreed", by_agent=False),)
    assert needs_action(thread(replies=replies)) is True


def test_a_reply_does_not_re_open_a_resolved_thread():
    replies = (
        Reply(id="r1", content="answered", by_agent=True),
        Reply(id="r2", content="thanks", by_agent=False),
    )
    assert needs_action(thread(replies=replies, resolved=True)) is False


def test_a_reply_does_not_re_open_an_unmarked_thread():
    replies = (
        Reply(id="r1", content="answered", by_agent=True),
        Reply(id="r2", content="thanks", by_agent=False),
    )
    assert needs_action(thread(content="no marker here", replies=replies)) is False
```

Replace `tests/test_model.py:55-63` with:

```python
def test_agent_replied_last_is_true_when_the_agent_spoke_last():
    replies = (Reply(id="r1", content="answered", by_agent=True),)
    assert thread_with(replies=replies).agent_replied_last is True


def test_agent_replied_last_is_false_when_someone_replied_after():
    replies = (
        Reply(id="r1", content="answered", by_agent=True),
        Reply(id="r2", content="not quite", by_agent=False),
    )
    assert thread_with(replies=replies).agent_replied_last is False


def test_agent_replied_last_is_false_with_no_replies():
    assert parse_thread(ANCHORED).agent_replied_last is False
```

Read the existing `tests/test_model.py:55-63` before replacing it. If it builds
its thread by a different helper name than `thread_with`, use that name. Do not
add a second helper.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_filters.py tests/test_model.py -q`
Expected: FAIL. The `test_model.py` cases fail with `AttributeError: 'Thread' object has no attribute 'agent_replied_last'`. `test_a_human_reply_after_the_agent_re_opens_the_thread` fails on the assertion, because today any agent reply closes the thread.

- [ ] **Step 3: Change the property**

In `gdoc/model.py`, replace the `has_agent_reply` property with:

```python
    @property
    def agent_replied_last(self) -> bool:
        """True when the last word in the thread is the agent's.

        The old test was "has the agent ever replied", which made answering the
        agent do nothing: the thread was skipped on every run after the first
        reply, forever. Last-reply is what lets a correction re-open the work.

        by_agent is the only identity signal available on main. Under the
        gdoc-oauth branch the agent's replies are indistinguishable from Nail's
        to Drive, and this is the one line that changes: the marker decides, not
        the account.
        """
        return bool(self.replies) and self.replies[-1].by_agent
```

Drive returns replies oldest first, so `replies[-1]` is the latest. `fetch.py`
preserves that order and must keep doing so.

- [ ] **Step 4: Change the filter**

In `gdoc/filters.py`, in `needs_action`, replace `and not thread.has_agent_reply`
with `and not thread.agent_replied_last`, and replace the docstring's second
paragraph with:

```python
    """True when the thread is waiting on the agent.

    The last-reply check is what makes a second run idempotent without a local
    record of handled ids: the agent never posts twice in a row. It is also what
    lets a human reply re-open a thread the agent had answered, so a correction
    is not lost.
    """
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_filters.py tests/test_model.py -q`
Expected: PASS

- [ ] **Step 6: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS. Nothing outside `model.py` and `filters.py` referenced `has_agent_reply`, so nothing else should break. If something does, it is a real caller and it needs the same change, not a shim.

- [ ] **Step 7: Commit**

```bash
git add gdoc/model.py gdoc/filters.py tests/test_model.py tests/test_filters.py
git commit -m "fix: a reply re-opens a thread the agent had answered

has_agent_reply asked whether the agent had ever replied, so answering the
agent did nothing: the thread was skipped on every later run, forever.

agent_replied_last asks who spoke last. A correction now re-opens the work,
which is what the merge in gdoc worklist depends on."
```

---

## Task 2: `pending.md` can be read, not only written

**Files:**
- Modify: `gdoc/pending.py`
- Test: `tests/test_pending.py`

**Interfaces:**
- Consumes: nothing.
- Produces: `PendingItem(number: int, comment_id: str, text: str)` frozen dataclass, and `read_items(path: Path) -> tuple[PendingItem, ...]`, which returns an empty tuple when the file does not exist.

`text` is the captured request with its `> ` quoting stripped. Task 3 uses it as
the fallback ask when a comment has been deleted from the document.

- [ ] **Step 1: Write the failing tests**

Add to `tests/test_pending.py`:

```python
from gdoc.pending import PendingItem, read_items

QUEUE = """\
# Pending global items

Document: https://docs.google.com/document/d/abc123/edit
Source: domains/policy/refunds.md

Each item needs changes across the document, so it was not answered in the
comment thread. Work through them with /gdoc-apply.

## Item 1

- Captured: 2026-08-14
- Comment id: AAAA1111
- Anchored to: within five working days

Nail asked:

> renumber the annex to follow the new section order

## Item 2

- Captured: 2026-08-14
- Comment id: BBBB2222
- Anchored to: whole document

Nail asked:

> use client, not customer
>
> everywhere except the quoted regulation
"""


def test_read_items_returns_nothing_when_the_file_is_absent(tmp_path):
    assert read_items(tmp_path / "pending.md") == ()


def test_read_items_returns_one_item_per_heading(tmp_path):
    path = tmp_path / "pending.md"
    path.write_text(QUEUE)

    items = read_items(path)

    assert [item.number for item in items] == [1, 2]
    assert [item.comment_id for item in items] == ["AAAA1111", "BBBB2222"]


def test_read_items_strips_the_quote_markers(tmp_path):
    path = tmp_path / "pending.md"
    path.write_text(QUEUE)

    items = read_items(path)

    assert items[0].text == "renumber the annex to follow the new section order"
    assert items[1].text == (
        "use client, not customer\n\neverywhere except the quoted regulation"
    )


def test_read_items_tolerates_fields_it_does_not_know(tmp_path):
    """gdoc-oauth adds Author and Marked lines. An unknown field is not an error."""
    path = tmp_path / "pending.md"
    path.write_text(
        "# Pending global items\n\n"
        "## Item 1\n\n"
        "- Captured: 2026-08-14\n"
        "- Comment id: CCCC3333\n"
        "- Author: William Mejia\n"
        "- Marked: ai!\n"
        "- Anchored to: whole document\n\n"
        "The comment:\n\n"
        "> drop the pilot section\n"
    )

    items = read_items(path)

    assert items == (PendingItem(number=1, comment_id="CCCC3333", text="drop the pilot section"),)


def test_read_items_skips_an_item_with_no_comment_id(tmp_path):
    """A hand-written item is not a bug, but it cannot be matched to a thread."""
    path = tmp_path / "pending.md"
    path.write_text("## Item 1\n\n- Captured: 2026-08-14\n\n> do the thing\n")

    assert read_items(path) == ()


def test_round_trip_from_append_item(tmp_path):
    """The reader must read what the writer writes, not an idea of the format."""
    from gdoc.model import Thread

    thread = Thread(
        id="DDDD4444",
        content="ai! renumber the annex",
        author_name="Nail Khusnullin",
        author_email=None,
        by_agent=False,
        quoted="Annex A",
        resolved=False,
        replies=(),
    )
    append_item(tmp_path, "refunds", thread, doc_id="abc123", today="2026-08-14")

    items = read_items(pending_path(tmp_path, "refunds"))

    assert len(items) == 1
    assert items[0].comment_id == "DDDD4444"
    assert items[0].text == "ai! renumber the annex"
```

`test_pending.py` already imports `append_item` and `pending_path`. Add
`PendingItem` and `read_items` to that import rather than writing a second one.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_pending.py -q`
Expected: FAIL with `ImportError: cannot import name 'PendingItem' from 'gdoc.pending'`

- [ ] **Step 3: Write the reader**

Add to `gdoc/pending.py`, after the existing module constants:

```python
from dataclasses import dataclass

_COMMENT_ID = re.compile(r"^- Comment id: (\S+)\s*$", re.MULTILINE)
_QUOTED_LINE = re.compile(r"^> ?(.*)$")


@dataclass(frozen=True)
class PendingItem:
    number: int
    comment_id: str
    text: str
```

And the function, at the end of the file:

```python
def read_items(path: Path) -> tuple[PendingItem, ...]:
    """Every item in the queue, in file order.

    The parser reads two things and ignores the rest: the item number in the
    heading, and the Comment id. Every other field is metadata for Nail, and
    gdoc-oauth adds more of them, so an unknown line must never be an error.

    An item with no Comment id is skipped. It cannot be matched to a thread, and
    a hand-written note in the queue is not a fault.
    """
    if not path.exists():
        return ()

    items: list[PendingItem] = []
    blocks = _ITEM_HEADING.split(path.read_text())
    # split with one capture group gives [preamble, number, body, number, body...]
    for number, body in zip(blocks[1::2], blocks[2::2]):
        match = _COMMENT_ID.search(body)
        if not match:
            continue
        items.append(
            PendingItem(
                number=int(number),
                comment_id=match.group(1),
                text=_unquote(body),
            )
        )
    return tuple(items)


def _unquote(body: str) -> str:
    """The quoted request, with its markers removed and its blank lines kept."""
    lines = [
        match.group(1)
        for match in (_QUOTED_LINE.match(line) for line in body.splitlines())
        if match
    ]
    return "\n".join(lines).strip()
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_pending.py -q`
Expected: PASS

If `test_round_trip_from_append_item` fails, the writer changed and the reader
did not. Fix the reader. Never change `_ITEM` to suit the parser: the file is
read by Nail first and by code second.

- [ ] **Step 5: Commit**

```bash
git add gdoc/pending.py tests/test_pending.py
git commit -m "feat: read pending.md, not only write it

The queue has only ever been appended to. The merge needs the comment ids
already captured, so it needs a reader.

The parser takes the item number and the Comment id and ignores every other
field, because gdoc-oauth adds two more and an unknown line must not be an
error."
```

---

## Task 3: `gdoc worklist` merges the queue with a live read

**Files:**
- Create: `gdoc/worklist.py`
- Modify: `gdoc/cli.py`
- Test: `tests/test_worklist.py`, `tests/test_cli.py`

**Interfaces:**
- Consumes: `PendingItem`, `read_items` from Task 2. `Thread` from `gdoc.model`. `partition` from `gdoc.filters`, which now uses Task 1's rule.
- Produces:
  - `WorkItem(comment_id: str, item_number: int | None, text: str, author: str | None, anchor: str | None)`
  - `Worklist(captured, reopened, fresh, missing, resolved)`, every field a `tuple[WorkItem, ...]`
  - `merge(items: tuple[PendingItem, ...], threads: tuple[Thread, ...]) -> Worklist`
  - CLI: `gdoc worklist <url> [--repo-root PATH] [--slug SLUG]`

The five buckets are the spec's three plus its two stops:

| Bucket | Meaning |
|---|---|
| `captured` | In the queue, thread quiet since capture |
| `reopened` | In the queue, and someone replied after the agent |
| `fresh` | Marked in the document, not in the queue |
| `missing` | In the queue, the comment is gone from the document |
| `resolved` | In the queue, the thread is resolved |

- [ ] **Step 1: Write the failing tests**

Create `tests/test_worklist.py`:

```python
from gdoc.model import Reply, Thread
from gdoc.worklist import WorkItem, merge
from gdoc.pending import PendingItem


def thread(**kwargs):
    defaults = dict(
        id="t1",
        content="ai! renumber the annex",
        author_name="Nail Khusnullin",
        author_email=None,
        by_agent=False,
        quoted="Annex A",
        resolved=False,
        replies=(),
    )
    return Thread(**{**defaults, **kwargs})


AGENT = (Reply(id="r1", content="Captured as item 1.", by_agent=True),)
HUMAN_AFTER = AGENT + (Reply(id="r2", content="also the fees table", by_agent=False),)


def item(number=1, comment_id="t1", text="renumber the annex"):
    return PendingItem(number=number, comment_id=comment_id, text=text)


def test_a_captured_item_whose_thread_is_quiet_is_captured():
    result = merge((item(),), (thread(replies=AGENT),))

    assert [w.comment_id for w in result.captured] == ["t1"]
    assert result.reopened == ()
    assert result.fresh == ()


def test_a_captured_item_with_a_reply_since_is_reopened():
    result = merge((item(),), (thread(replies=HUMAN_AFTER),))

    assert [w.comment_id for w in result.reopened] == ["t1"]
    assert result.captured == ()


def test_a_marked_comment_not_in_the_queue_is_fresh():
    result = merge((), (thread(id="t9"),))

    assert [w.comment_id for w in result.fresh] == ["t9"]
    assert result.fresh[0].item_number is None


def test_an_unmarked_comment_is_in_no_bucket():
    result = merge((), (thread(id="t9", content="just a note"),))

    assert result.fresh == ()
    assert result.captured == ()


def test_a_captured_item_whose_comment_is_gone_is_missing():
    result = merge((item(comment_id="t404"),), (thread(id="t1", replies=AGENT),))

    assert [w.comment_id for w in result.missing] == ["t404"]


def test_a_missing_item_carries_the_captured_text():
    """It is the only record left of what was asked, so it has to reach Nail."""
    result = merge((item(comment_id="t404", text="renumber the annex"),), ())

    assert result.missing[0].text == "renumber the annex"


def test_a_captured_item_whose_thread_was_resolved_is_resolved():
    result = merge((item(),), (thread(resolved=True, replies=AGENT),))

    assert [w.comment_id for w in result.resolved] == ["t1"]
    assert result.captured == ()


def test_a_reopened_item_carries_the_live_thread_text_not_the_captured_text():
    """A reply may narrow the ask. Working from the file alone acts on stale text."""
    result = merge(
        (item(text="renumber the annex"),),
        (thread(content="ai! renumber the annex", replies=HUMAN_AFTER),),
    )

    assert result.reopened[0].text == "ai! renumber the annex"
    assert result.reopened[0].item_number == 1


def test_a_fresh_item_carries_its_author_and_anchor():
    result = merge((), (thread(id="t9", author_name="William Mejia", quoted="five days"),))

    assert result.fresh[0].author == "William Mejia"
    assert result.fresh[0].anchor == "five days"


def test_file_order_is_kept():
    items = (item(number=1, comment_id="a"), item(number=2, comment_id="b"))
    threads = (thread(id="b", replies=AGENT), thread(id="a", replies=AGENT))

    result = merge(items, threads)

    assert [w.item_number for w in result.captured] == [1, 2]


def test_an_empty_queue_and_an_empty_document_give_an_empty_worklist():
    result = merge((), ())

    assert result == Worklist(captured=(), reopened=(), fresh=(), missing=(), resolved=())
```

Add `Worklist` to the import at the top of the file, alongside `WorkItem` and
`merge`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_worklist.py -q`
Expected: FAIL with `ModuleNotFoundError: No module named 'gdoc.worklist'`

- [ ] **Step 3: Write the merge**

Create `gdoc/worklist.py`:

```python
"""Merge the captured queue with what the document says now.

gdoc-apply used to read pending.md and nothing else, so a comment left after the
review that filled it never reached the new version. The merge is what closes
that: the queue holds accepted work, a live read holds everything nobody has
looked at, and the union is what the apply session owes Nail.

The dedup key is the comment id. Nothing else is stable. The text is edited, the
anchor moves when the paragraph moves, and the author is a display name.
"""

from dataclasses import dataclass

from gdoc.filters import partition
from gdoc.model import Thread
from gdoc.pending import PendingItem


@dataclass(frozen=True)
class WorkItem:
    comment_id: str
    item_number: int | None
    text: str
    author: str | None
    anchor: str | None


@dataclass(frozen=True)
class Worklist:
    captured: tuple[WorkItem, ...]
    reopened: tuple[WorkItem, ...]
    fresh: tuple[WorkItem, ...]
    missing: tuple[WorkItem, ...]
    resolved: tuple[WorkItem, ...]


def _from_thread(thread: Thread, number: int | None) -> WorkItem:
    return WorkItem(
        comment_id=thread.id,
        item_number=number,
        text=thread.content,
        author=thread.author_name,
        anchor=thread.quoted,
    )


def _from_item(item: PendingItem) -> WorkItem:
    """Everything known about an item whose comment is no longer readable."""
    return WorkItem(
        comment_id=item.comment_id,
        item_number=item.number,
        text=item.text,
        author=None,
        anchor=None,
    )


def merge(items: tuple[PendingItem, ...], threads: tuple[Thread, ...]) -> Worklist:
    """Sort every known piece of work into one of five buckets.

    A queued item is placed by what its thread says now, which is why the live
    read wins on text: a reply since capture may narrow the ask or withdraw it,
    and working from the file alone would act on a stale instruction.
    """
    addressed, _ = partition(threads)
    by_id = {thread.id: thread for thread in threads}
    addressed_ids = {thread.id for thread in addressed}
    queued_ids = {item.comment_id for item in items}

    captured: list[WorkItem] = []
    reopened: list[WorkItem] = []
    missing: list[WorkItem] = []
    resolved: list[WorkItem] = []

    for item in items:
        thread = by_id.get(item.comment_id)
        if thread is None:
            missing.append(_from_item(item))
        elif thread.resolved:
            resolved.append(_from_thread(thread, item.number))
        elif thread.id in addressed_ids:
            reopened.append(_from_thread(thread, item.number))
        else:
            captured.append(_from_thread(thread, item.number))

    fresh = tuple(
        _from_thread(thread, None) for thread in addressed if thread.id not in queued_ids
    )

    return Worklist(
        captured=tuple(captured),
        reopened=tuple(reopened),
        fresh=fresh,
        missing=tuple(missing),
        resolved=tuple(resolved),
    )
```

`partition` already excludes resolved threads, so a resolved thread can never be
in `addressed` and the two branches cannot both fire.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_worklist.py -q`
Expected: PASS

- [ ] **Step 5: Write the failing CLI test**

Add to `tests/test_cli.py`. The file's style is `patch("gdoc.cli.X")` around a
call to `main([...])`, not a handler called directly. Match it.

```python
def _fake_thread(id, content, resolved=False, replies=()):
    from gdoc.model import Thread

    return Thread(
        id=id,
        content=content,
        author_name="Nail Khusnullin",
        author_email=None,
        by_agent=False,
        quoted=None,
        resolved=resolved,
        replies=replies,
    )


def test_worklist_merges_the_queue_with_the_document(capsys, tmp_path):
    queue = tmp_path / ".gdoc" / "policy"
    queue.mkdir(parents=True)
    (queue / "pending.md").write_text(
        "## Item 1\n\n- Comment id: c1\n\n> renumber the annex\n"
    )

    drive = MagicMock()
    drive.files().get.return_value.execute.return_value = {"name": "Test doc"}
    threads = (
        _fake_thread("c1", "ai! renumber the annex"),
        _fake_thread("c9", "ai: is this the right term"),
    )
    with patch("gdoc.cli.drive_service", return_value=drive), patch(
        "gdoc.cli.fetch_threads", return_value=threads
    ):
        exit_code = main(
            [
                "worklist",
                "https://docs.google.com/document/d/1AbCdefghijklmnopqrstuvwx/edit",
                "--slug",
                "policy",
                "--repo-root",
                str(tmp_path),
            ]
        )

    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert [w["comment_id"] for w in payload["reopened"]] == ["c1"]
    assert [w["comment_id"] for w in payload["fresh"]] == ["c9"]
    assert payload["missing"] == []
    assert payload["slug"] == "policy"
    assert payload["total"] == 2


def test_worklist_reports_an_item_whose_comment_is_gone(capsys, tmp_path):
    queue = tmp_path / ".gdoc" / "policy"
    queue.mkdir(parents=True)
    (queue / "pending.md").write_text(
        "## Item 1\n\n- Comment id: c404\n\n> renumber the annex\n"
    )

    drive = MagicMock()
    drive.files().get.return_value.execute.return_value = {"name": "Test doc"}
    with patch("gdoc.cli.drive_service", return_value=drive), patch(
        "gdoc.cli.fetch_threads", return_value=()
    ):
        exit_code = main(
            [
                "worklist",
                "https://docs.google.com/document/d/1AbCdefghijklmnopqrstuvwx/edit",
                "--slug",
                "policy",
                "--repo-root",
                str(tmp_path),
            ]
        )

    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert [w["comment_id"] for w in payload["missing"]] == ["c404"]
    assert payload["missing"][0]["text"] == "renumber the annex"
    assert payload["total"] == 0
```

`c1` lands in `reopened` rather than `captured` because it has no agent reply, so
`needs_action` is true. That is correct: a queued item nobody answered in the
thread is still waiting on the agent.

- [ ] **Step 6: Run it to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -k worklist -q`
Expected: FAIL. argparse rejects `worklist` as an unknown subcommand and exits 2.

- [ ] **Step 7: Add the subcommand**

In `gdoc/cli.py`, add to the imports:

```python
from gdoc.pending import append_item, pending_path, read_items, recorded_source
from gdoc.worklist import merge
```

Add the handler after `cmd_capture`:

```python
def _work_json(item) -> dict:
    return {
        "comment_id": item.comment_id,
        "item": item.item_number,
        "text": item.text,
        "author": item.author,
        "anchor": item.anchor,
    }


def cmd_worklist(args) -> int:
    """Everything this document is still owed, from the queue and from itself.

    Read only. It posts nothing and writes nothing, so it runs under
    --terminal-only like any other read.
    """
    drive = drive_service()
    doc_id = extract_doc_id(args.url)
    repo_root = Path(args.repo_root)

    source = find_by_doc_id(repo_root, doc_id)
    if source is None and not args.slug:
        return _fail(
            f"no markdown under {repo_root} is paired to {doc_id}. "
            "Pair it with gdoc pair set, or pass --slug to choose a queue directory."
        )
    slug = args.slug or slug_for_source(source)

    items = read_items(pending_path(repo_root, slug))
    result = merge(items, fetch_threads(drive, doc_id))
    buckets = {
        name: [_work_json(w) for w in getattr(result, name)]
        for name in ("captured", "reopened", "fresh", "missing", "resolved")
    }
    return _emit(
        {
            "doc_id": doc_id,
            "name": _file_meta(drive, doc_id).get("name"),
            "slug": slug,
            "source": _source_label(repo_root, source) if source else None,
            "total": len(buckets["captured"]) + len(buckets["reopened"]) + len(buckets["fresh"]),
            **buckets,
        }
    )
```

`total` counts the three buckets that are work. `missing` and `resolved` are
questions for Nail, not items to apply, so they are not in it.

Register it in the parser, beside `capture`:

```python
    worklist = sub.add_parser(
        "worklist", help="merge pending.md with a live read of the comments"
    )
    worklist.add_argument("url")
    worklist.add_argument("--repo-root", default=".")
    worklist.add_argument("--slug", default=None)
    worklist.set_defaults(func=cmd_worklist)
```

Match the `--repo-root` default already used by `capture`. If `capture` uses
something other than `"."`, use the same thing.

- [ ] **Step 8: Run the tests**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py tests/test_worklist.py -q`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add gdoc/worklist.py gdoc/cli.py tests/test_worklist.py tests/test_cli.py
git commit -m "feat: gdoc worklist merges the queue with a live read

gdoc-apply reads pending.md and nothing else, so a comment left after the
review that filled it never reaches the new version.

worklist returns five buckets keyed by comment id: captured, reopened, fresh,
missing and resolved. A queued item is placed by what its thread says now, so
a reply that narrows the ask is not lost to the captured text."
```

---

## Task 4: `gdoc drift` finds direct edits to the document

**Files:**
- Create: `gdoc/drift.py`
- Modify: `gdoc/cli.py`
- Test: `tests/test_drift.py`, `tests/test_cli.py`

**Interfaces:**
- Consumes: `export_markdown` from `gdoc.export`, `baseline_path` from `gdoc.baseline`.
- Produces: `diff_markdown(baseline: str, current: str) -> str`, empty string when they match. CLI: `gdoc drift <url> [--repo-root PATH] [--slug SLUG]`.

`baseline.md` is the export of the version that was generated, taken when the
document and the source provably matched. Comparing it to a fresh export shows
what was typed into the document since. Both sides carry the same pandoc round
trip, so the formatting distortion cancels.

Nothing here diffs the baseline against the **source** markdown. Those are
different text and always will be, and under PR #5 the baseline also carries a
cover page, control tables and a contents list that the source has never had.

- [ ] **Step 1: Write the failing tests**

Create `tests/test_drift.py`:

```python
from gdoc.drift import diff_markdown

BASELINE = """\
# Refund policy

We refund within five working days.

## Annex A

Fees are listed below.
"""


def test_identical_text_is_no_drift():
    assert diff_markdown(BASELINE, BASELINE) == ""


def test_a_changed_line_shows_both_sides():
    current = BASELINE.replace("five working days", "three working days")

    diff = diff_markdown(BASELINE, current)

    assert "-We refund within five working days." in diff
    assert "+We refund within three working days." in diff


def test_an_added_line_shows_as_an_addition():
    current = BASELINE.replace("## Annex A", "This was added.\n\n## Annex A")

    diff = diff_markdown(BASELINE, current)

    assert "+This was added." in diff


def test_a_removed_line_shows_as_a_removal():
    current = BASELINE.replace("Fees are listed below.\n", "")

    diff = diff_markdown(BASELINE, current)

    assert "-Fees are listed below." in diff


def test_a_trailing_newline_alone_is_not_drift():
    """Exports differ in trailing whitespace between calls. That is not an edit."""
    assert diff_markdown(BASELINE, BASELINE.rstrip() + "\n\n\n") == ""


def test_the_labels_name_which_side_is_which():
    current = BASELINE.replace("five", "three")

    diff = diff_markdown(BASELINE, current)

    assert diff.splitlines()[0] == "--- baseline.md"
    assert diff.splitlines()[1] == "+++ the document now"
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_drift.py -q`
Expected: FAIL with `ModuleNotFoundError: No module named 'gdoc.drift'`

- [ ] **Step 3: Write the diff**

Create `gdoc/drift.py`:

```python
"""Find what was typed into the document after it was generated.

generate builds every version from the source markdown, so anything Nail edits
in the document by hand is absent from the next version. Nothing used to notice.
gdoc/baseline.py has always said this comparison is what the file is for.

baseline.md and a fresh export both went through the same pandoc round trip, so
the formatting distortion cancels and only real edits stand out. That holds only
for baseline against export. Diffing the baseline against the source markdown
would surface the entire round trip, and under the template work the whole cover
page and contents list too, as spurious edits. Nothing may do that.
"""

import difflib


def diff_markdown(baseline: str, current: str) -> str:
    """A unified diff, or an empty string when the two match.

    Trailing whitespace is stripped from both sides before comparing. Two
    exports of an unchanged document differ there often enough that reporting it
    as an edit would train Nail to ignore the report.
    """
    diff = difflib.unified_diff(
        baseline.rstrip().splitlines(),
        current.rstrip().splitlines(),
        fromfile="baseline.md",
        tofile="the document now",
        lineterm="",
    )
    return "\n".join(diff)
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_drift.py -q`
Expected: PASS

- [ ] **Step 5: Write the failing CLI tests**

Add to `tests/test_cli.py`:

```python
DOC_URL = "https://docs.google.com/document/d/1AbCdefghijklmnopqrstuvwx/edit"


def _run_drift(tmp_path, current):
    with patch("gdoc.cli.drive_service", return_value=MagicMock()), patch(
        "gdoc.cli.export_markdown", return_value=current
    ):
        return main(
            ["drift", DOC_URL, "--slug", "policy", "--repo-root", str(tmp_path)]
        )


def _write_baseline(tmp_path, text):
    queue = tmp_path / ".gdoc" / "policy"
    queue.mkdir(parents=True)
    (queue / "baseline.md").write_text(text)


def test_drift_reports_a_direct_edit(capsys, tmp_path):
    _write_baseline(tmp_path, "We refund within five working days.\n")

    exit_code = _run_drift(tmp_path, "We refund within three working days.\n")

    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["has_drift"] is True
    assert "+We refund within three working days." in payload["diff"]


def test_drift_reports_nothing_when_the_document_is_untouched(capsys, tmp_path):
    _write_baseline(tmp_path, "We refund within five working days.\n")

    exit_code = _run_drift(tmp_path, "We refund within five working days.\n")

    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["has_drift"] is False
    assert payload["diff"] == ""


def test_drift_says_so_when_there_is_no_baseline(capsys, tmp_path):
    """A first-ever apply has no baseline. That is not an error and not drift."""
    exit_code = _run_drift(tmp_path, "anything")

    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["has_drift"] is None
    assert "no baseline" in payload["reason"]
```

`cmd_drift` never calls `_file_meta`, so the `MagicMock` drive needs no stubbing
here. `export_markdown` is patched at `gdoc.cli`, where it is imported, not at
`gdoc.export`.

- [ ] **Step 6: Run them to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -k drift -q`
Expected: FAIL. argparse rejects `drift` as an unknown subcommand and exits 2.

- [ ] **Step 7: Add the subcommand**

In `gdoc/cli.py`, add to the imports:

```python
from gdoc.baseline import baseline_path, slugify, write_baseline
from gdoc.drift import diff_markdown
```

Add the handler after `cmd_worklist`:

```python
def cmd_drift(args) -> int:
    """What was edited in the document since it was generated.

    Read only. No baseline is not a fault: the first apply on a document that
    predates baselines has nothing to compare against, and saying so is more
    use than an error.
    """
    drive = drive_service()
    doc_id = extract_doc_id(args.url)
    repo_root = Path(args.repo_root)

    source = find_by_doc_id(repo_root, doc_id)
    if source is None and not args.slug:
        return _fail(
            f"no markdown under {repo_root} is paired to {doc_id}. "
            "Pair it with gdoc pair set, or pass --slug to choose a queue directory."
        )
    slug = args.slug or slug_for_source(source)
    path = baseline_path(repo_root, slug)

    if not path.exists():
        return _emit(
            {
                "doc_id": doc_id,
                "slug": slug,
                "has_drift": None,
                "diff": "",
                "baseline_path": str(path),
                "reason": (
                    f"no baseline at {path}. This document was generated before "
                    "baselines existed, or by something other than gdoc generate. "
                    "Direct edits cannot be detected until the next generate writes one."
                ),
            }
        )

    diff = diff_markdown(path.read_text(), export_markdown(drive, doc_id))
    return _emit(
        {
            "doc_id": doc_id,
            "slug": slug,
            "has_drift": bool(diff),
            "diff": diff,
            "baseline_path": str(path),
        }
    )
```

Register it beside `worklist`:

```python
    drift = sub.add_parser(
        "drift", help="diff baseline.md against the document as it is now"
    )
    drift.add_argument("url")
    drift.add_argument("--repo-root", default=".")
    drift.add_argument("--slug", default=None)
    drift.set_defaults(func=cmd_drift)
```

- [ ] **Step 8: Run the tests**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py tests/test_drift.py -q`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add gdoc/drift.py gdoc/cli.py tests/test_drift.py tests/test_cli.py
git commit -m "feat: gdoc drift finds edits made in the document

generate builds every version from the source markdown, so anything typed
into the document by hand is dropped from the next version without a word.

baseline.py has always said this diff is what the file is for. Nothing did
it. Both sides carry the same pandoc round trip, so it cancels and only real
edits show. A missing baseline reports itself rather than failing."
```

---

## Task 5: `gdoc comment` posts the note on the superseded version

**Files:**
- Modify: `gdoc/reply.py`
- Modify: `gdoc/cli.py`
- Test: `tests/test_reply.py`, `tests/test_cli.py`

**Interfaces:**
- Consumes: `assert_plain_text` from `gdoc.reply`.
- Produces: `post_comment(drive, doc_id: str, body: str) -> str`. CLI: `gdoc comment <doc> --body-file PATH`.

This is a top-level comment, not a reply, so it needs `comments().create` rather
than `replies().create`. It is used once per apply, for the "superseded by v2"
note on the old version.

- [ ] **Step 1: Write the failing tests**

Add to `tests/test_reply.py`:

```python
from gdoc.reply import CAPTURED_NOTE, assert_plain_text, post_comment, post_reply


class FakeComments:
    def __init__(self):
        self.calls = []

    def create(self, **kwargs):
        self.calls.append(kwargs)
        return self

    def execute(self):
        return {"id": "c1"}


class FakeDrive:
    def __init__(self):
        self._comments = FakeComments()

    def comments(self):
        return self._comments


def test_post_comment_creates_a_top_level_comment():
    drive = FakeDrive()

    assert post_comment(drive, "abc123", "Superseded by v2: https://example.com") == "c1"
    assert drive.comments().calls[0]["fileId"] == "abc123"
    assert drive.comments().calls[0]["body"] == {
        "content": "Superseded by v2: https://example.com"
    }


def test_post_comment_refuses_markdown():
    drive = FakeDrive()

    with pytest.raises(ValueError, match="markdown"):
        post_comment(drive, "abc123", "**Superseded** by v2")


def test_post_comment_refuses_an_empty_body():
    drive = FakeDrive()

    with pytest.raises(ValueError, match="empty"):
        post_comment(drive, "abc123", "   ")


def test_captured_note_says_it_will_be_in_the_next_version():
    assert "item 2" in CAPTURED_NOTE.format(item=2)
    assert assert_plain_text(CAPTURED_NOTE.format(item=1))
```

Delete the two existing `GLOBAL_REFUSAL` tests at `tests/test_reply.py:52` and
`:56`. The last test above replaces both.

- [ ] **Step 2: Run them to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_reply.py -q`
Expected: FAIL with `ImportError: cannot import name 'CAPTURED_NOTE' from 'gdoc.reply'`

- [ ] **Step 3: Rename the note and add the function**

In `gdoc/reply.py`, replace `GLOBAL_REFUSAL` with:

```python
CAPTURED_NOTE = (
    "Captured as item {item}. This one changes the document's text, so it is "
    "not answered here. It will be in the next version, and I will post the "
    "link in this thread when it exists."
)
```

The old text said "this needs changes across the document", which was the global
half of a distinction that no longer exists. Every item changes the text now,
whether it changes one sentence or forty.

Add at the end of the file:

```python
def post_comment(drive, doc_id: str, body: str) -> str:
    """Post a top-level comment, not a reply.

    Used once per apply, for the note on the version being superseded. Anyone
    still reading the old document needs a way to find the new one, and a reply
    inside a thread would only reach whoever was in that thread.
    """
    assert_plain_text(body)
    created = (
        drive.comments()
        .create(fileId=doc_id, body={"content": body}, fields="id")
        .execute()
    )
    return created["id"]
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_reply.py -q`
Expected: PASS

- [ ] **Step 5: Write the failing CLI test**

Add to `tests/test_cli.py`:

```python
def test_comment_posts_a_top_level_comment(capsys, tmp_path):
    body = tmp_path / "note.txt"
    body.write_text("Superseded by v2: https://example.com/doc")

    with patch("gdoc.cli.drive_service", return_value=MagicMock()), patch(
        "gdoc.cli.post_comment", return_value="c1"
    ) as post:
        exit_code = main(
            [
                "comment",
                "https://docs.google.com/document/d/1AbCdefghijklmnopqrstuvwx/edit",
                "--body-file",
                str(body),
            ]
        )

    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["comment_id"] == "c1"
    assert post.call_args.args[1] == "1AbCdefghijklmnopqrstuvwx"
    assert post.call_args.args[2] == "Superseded by v2: https://example.com/doc"
```

- [ ] **Step 6: Run it to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -k comment -q`
Expected: FAIL. `patch("gdoc.cli.post_comment")` raises `AttributeError` because
`cli.py` does not import it yet.

- [ ] **Step 7: Add the subcommand**

In `gdoc/cli.py`, change the reply import to:

```python
from gdoc.reply import post_comment, post_reply
```

Add the handler after `cmd_reply`:

```python
def cmd_comment(args) -> int:
    """Post one top-level comment. Not a reply, and not a document edit."""
    doc_id = extract_doc_id(args.doc)
    body = Path(args.body_file).read_text()
    comment_id = post_comment(drive_service(), doc_id, body)
    return _emit({"comment_id": comment_id, "doc_id": doc_id})
```

Register it beside `reply`:

```python
    comment = sub.add_parser("comment", help="post one plain-text top-level comment")
    comment.add_argument("doc")
    comment.add_argument("--body-file", required=True)
    comment.set_defaults(func=cmd_comment)
```

- [ ] **Step 8: Run the suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add gdoc/reply.py gdoc/cli.py tests/test_reply.py tests/test_cli.py
git commit -m "feat: gdoc comment, for the note on the superseded version

Every version is a new Google Doc, so anyone reading the old one has no way
to reach the new one. One top-level comment on the old version fixes that.

GLOBAL_REFUSAL becomes CAPTURED_NOTE. Its old text named the global half of
a distinction that no longer exists, and it promised nothing back."
```

---

## Task 6: `gdoc-apply` drains the document

**Files:**
- Modify: `skills/gdoc-apply/SKILL.md`

**Interfaces:**
- Consumes: `gdoc worklist`, `gdoc drift`, `gdoc comment`, `gdoc capture`, all from Tasks 3, 4 and 5.
- Produces: nothing code depends on. This is the skill Claude Code loads, and it is a symlink into this repo, so the edit is live the moment it is saved.

There is no test for a SKILL.md. Read it end to end after editing and check that
every command it names exists and every flag it passes is real.

- [ ] **Step 1: Replace Step 1 of the skill**

Replace the whole of `## Step 1: Load the context` with:

```markdown
## Step 1: Load everything the document is owed

The argument is a path to `pending.md`, usually `.gdoc/<slug>/pending.md`, or a
document URL. Either way you need both the queue and the document.

`pending.md`'s header holds two lines:

```
Document: https://docs.google.com/document/d/<doc_id>/edit
Source: <path to the source markdown, relative to the root>
```

Use `Source:` to find the paired markdown. Older queues have no `Source:` line,
so fall back to the document id:

```bash
$GDOC pair find --doc-id <doc_id>
```

If neither answers, Nail does not own this document. The output is a note for
him, not a new document. Say so and stop.

Then get the full picture. This is one command and it does the merge for you:

```bash
$GDOC worklist <url>
```

Five buckets, all keyed by comment id:

| Bucket | Meaning | What you do |
|---|---|---|
| `captured` | In the queue, quiet since | Work it |
| `reopened` | In the queue, someone replied after the agent | Work it, reading the whole thread. The reply may narrow or withdraw the ask |
| `fresh` | Marked in the document, never captured | Apply the test in Step 2 first |
| `missing` | In the queue, the comment is gone from the document | Stop and ask |
| `resolved` | In the queue, the thread is resolved | Stop and ask |

Never re-derive these buckets by hand from `gdoc read`. The command exists so
the id matching is done once and the same way every time.

`baseline.md` in the same folder is the document as it was generated. It is read
only for comparison in Step 3, and it is never the file you edit.
```

- [ ] **Step 2: Add the new Step 2**

Insert a new section directly after Step 1:

```markdown
## Step 2: Show Nail the worklist, and stop

Print all five buckets and wait. Mark which items are new, because a `fresh`
item is one he has not seen in this session and may not have seen at all.

```
Already captured (2)
  item 1  renumber the annex to follow the new section order
  item 2  use client, not customer

Reopened (1)
  item 3  ai! drop the pilot section
          William Mejia replied: "keep the last paragraph though"

New since the last review (2)
  para 9  (William Mejia)  ai: is this the right term
  para 12 (Nail)           ai! the fees table needs the new rates

Needs your answer before I start
  item 4  in the queue, but its comment is gone from the document.
          It asked: "renumber the annex"
          Apply it, or drop it?

Proceed?
```

For every `fresh` item, apply one test before capturing it:

**Does this comment change the document's text?**

Yes, it is an item. Capture it:

```bash
$GDOC capture <doc_id> <comment_id>
```

No, it is a question. Answer it in the thread with `$GDOC reply` and capture
nothing. Do not put a question in the queue: there would be nothing to apply and
it would sit there until someone deleted it by hand.

This is the same test `gdoc-review` runs. It has to give the same answer here, or
running the two skills in a different order would give a different result.

A `missing` or `resolved` item is never guessed at. Ask, then either work it from
the captured text or delete it from `pending.md`.
```

- [ ] **Step 3: Add the fold-back step**

Insert after the new Step 2:

```markdown
## Step 3: Fold direct edits back into the markdown

Before touching any item. Every item's edit is written against the source
markdown, so an edit folded in afterwards lands on text the items have moved.

```bash
$GDOC drift <url>
```

Three answers:

- `has_drift: false`. Say nothing, go to Step 4.
- `has_drift: null`. There is no baseline, so direct edits cannot be detected on
  this document. Say that plainly and go to Step 4. Do not treat it as clean.
- `has_drift: true`. The `diff` field is a unified diff of `baseline.md` against
  the document now.

For a real diff, read it and tell Nail what he changed, in his terms, not in
diff terms:

```
You edited v1 in 4 places since it was generated.

  section 2.1   "within five working days" -> "within three working days"
  section 4     one sentence added after the first paragraph
  section 6.2   the second bullet removed
  annex A       a row added to the fees table

Fold them in, or continue without them?
```

If he says fold them in, translate each one into the source markdown, one at a
time, showing the diff and waiting, exactly as you do for an item.

**Translate, never patch.** The diff's hunks will not apply to the source. The
source went markdown to docx to Google Doc to markdown to become the baseline,
and that trip is lossy. Find the same sentence in the source and change it there.

If he says continue without them, say so again in the final report. A discarded
edit is never silent.
```

- [ ] **Step 4: Renumber the rest and fix its references**

The old Step 2 becomes Step 4, and so on to Step 8. In the new Step 4, "Work
through the items in order", replace the framing sentence:

```markdown
One item at a time, in the order the worklist printed them. For each:

1. Say what you are about to change and where, and name who asked.
2. Make the edit in the **paired markdown file**, never in `baseline.md`.
3. Show Nail the diff for that item alone.
4. Wait for approval before the next item.

Every item changes the document's text. That is what made it an item. Whether it
touches one sentence or forty is a matter of size, not of kind, and it does not
change how you work it.
```

Delete the old sentence "Global changes are handled here, not in comment threads,
because a diff is reviewable and a comment thread is not." The distinction it
rests on is gone.

- [ ] **Step 5: Add the closure step**

Add a new step after the old Step 5, "Record the version", and before "Clear the
items":

```markdown
## Step 7: Post the closure

Only after a successful generate. If `doc_id` was null, skip this step: a reply
saying "applied" against a version that does not exist is a lie the thread keeps.

One reply in each thread whose item you applied:

```bash
cat > /tmp/closure.txt <<'EOF'
Applied. This is in v2: https://docs.google.com/document/d/<new_doc_id>/edit
EOF
$GDOC reply <old_doc_id> <comment_id> --body-file /tmp/closure.txt
```

An item Nail dropped rather than applied gets no reply. You did not do it, and
saying why is his to write.

Then one top-level comment on the version you superseded, so anyone still reading
it can find the new one:

```bash
cat > /tmp/superseded.txt <<'EOF'
Superseded by v2: https://docs.google.com/document/d/<new_doc_id>/edit
EOF
$GDOC comment <old_doc_id> --body-file /tmp/superseded.txt
```

One comment, not one per thread.
```

- [ ] **Step 6: Update the terminal-only section**

Replace the `--terminal-only` section with:

```markdown
## If Nail passes `--terminal-only`

Do Steps 1 to 5: read the worklist, fold back direct edits, work the items, edit
the markdown, show the diffs, commit if there is a repository. Then stop.

`$GDOC worklist` and `$GDOC drift` both run: they read, and reading is not
posting. `$GDOC capture` runs too, because `pending.md` is a local file.

Do not run `$GDOC generate`, which always tries to upload, and do not run Step 7,
which posts. Print each reply you would have posted instead. Tell Nail the
markdown is saved and that re-running without the flag will generate the document.
```

- [ ] **Step 7: Update the Never list**

Add to the `## Never` list:

```markdown
- Never patch a drift hunk into the source markdown. Translate it.
- Never capture a question. If there is nothing to edit, answer it in the thread.
- Never post a closure reply before generate has succeeded.
- Never re-derive the worklist buckets by hand.
```

- [ ] **Step 8: Read it end to end and check every command**

Run: `grep -n '\$GDOC' skills/gdoc-apply/SKILL.md`
Expected: only `worklist`, `drift`, `capture`, `reply`, `comment`, `generate`,
`pair find`, `pair add-version`. Any other subcommand is a typo or a command that
does not exist.

Run: `~/.config/gdoc-agent/venv/bin/gdoc --help`
Expected: every subcommand named above is listed.

Read the whole file top to bottom. Check the step numbers run 1 to 8 with no gap
and no repeat, and that every cross-reference names the step it means.

- [ ] **Step 9: Commit**

```bash
git add skills/gdoc-apply/SKILL.md
git commit -m "feat: gdoc-apply drains the document

The skill read pending.md and nothing else, so a comment left after the
review, a colleague's comment, a reply to the agent and a direct edit to the
document all missed the next version without a word.

It now loads the worklist, folds direct edits back into the source first,
works every item, and posts a closure reply plus one note on the version it
supersedes."
```

---

## Task 7: `gdoc-review` runs the same one test

**Files:**
- Modify: `skills/gdoc-review/SKILL.md`

**Interfaces:**
- Consumes: `CAPTURED_NOTE` from Task 5, which is the text the skill posts after a capture.
- Produces: nothing code depends on.

This task is last among the behaviour changes on purpose. Until apply can drain
the document, review capturing more would be capturing into a queue that still
loses things.

- [ ] **Step 1: Replace the classification step**

Replace the whole of `## Step 3: Classify each addressed comment` with:

```markdown
## Step 3: Item or question

One test per comment, not per batch. One run may capture two and answer two.

**Does this comment change the document's text?**

| | Yes | No |
|---|---|---|
| Examples | Rephrase this. Add a missing clause. Renumber the annex. Apply a term change everywhere | Is this the right term? Why does this section say five days? Which PDR decided this? |
| Action | It is an item. Capture it, and answer in the thread too if there is useful text to show | It is a question. Answer it in the thread |

`gdoc-apply` runs this same test on anything it finds that you never captured. It
has to give the same answer, or running the two skills in a different order would
give a different result.

`forced_kind` in the JSON carries the override written in the comment: `ai?` means
answer it in the thread, `ai!` means it needs a new version. Neither decides
capture, because a comment can be both an answer and an item. Honour the override
for the reply, and run the test for the capture.

An unanchored comment has `anchored: false` and no quote. It refers to the
document as a whole, so it is an item unless it is plainly a question.

If a comment is genuinely ambiguous, ask Nail in the terminal. Do not guess.
```

- [ ] **Step 2: Fix the header sentence**

In the paragraph under `# Google Docs review`, replace:

```markdown
Local changes get an answer in the
thread. Global changes get captured, never attempted.
```

with:

```markdown
A comment that changes the
document's text becomes an item for a later session. A comment that only asks
something gets its answer in the thread.
```

- [ ] **Step 3: Update the capture step**

In `## Step 6`, retitle it `## Step 6: Capture the items` and replace the refusal
text with:

```
Captured as item 2. This one changes the document's text, so it is not answered
here. It will be in the next version, and I will post the link in this thread
when it exists.
```

Keep the two error cases below it, `--slug` and `source_collision_warning`,
unchanged.

- [ ] **Step 4: Update the report**

In `## Step 7: Report`, replace the example with:

```
posted    para 3   answered, cited domains/regulatory/cbc-emi.md
posted    para 7   answered, cited domains/policy/refunds.md
captured  para 2   item 1: renumber sections
captured  para 5   item 2: rephrase the opening, text posted in the thread

2 items in .gdoc/<slug>/pending.md
Next session: /gdoc-apply .gdoc/<slug>/pending.md
```

- [ ] **Step 5: Update the Never list**

Replace `- Never attempt a global change in a comment thread.` with:

```markdown
- Never capture a question. If there is nothing to edit, answer it in the thread.
- Never post the text of a change without also capturing it. Text posted and not
  captured is text Nail pastes into the document by hand, which is a direct edit
  the next apply has to find and translate back.
```

- [ ] **Step 6: Check for stale terms**

Run: `grep -n -i "global\|local" skills/gdoc-review/SKILL.md skills/gdoc-apply/SKILL.md`
Expected: no hits. If any remain, they are the retired distinction and they go.

Run: `grep -rn "GLOBAL_REFUSAL" gdoc/ tests/ skills/`
Expected: no hits.

- [ ] **Step 7: Commit**

```bash
git add skills/gdoc-review/SKILL.md
git commit -m "feat: one test in gdoc-review, does this change the text

Local and global decided whether review captured a comment. A local answer
carrying replacement text is a change the markdown never received: Nail
pasted it into the document and it became a direct edit.

Both skills now ask one question, so the same comment gets the same answer
whichever one sees it first."
```

---

## Task 8: The documents, and the final check

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Interfaces:**
- Consumes: everything above.
- Produces: nothing.

- [ ] **Step 1: Update CLAUDE.md**

In the `## What lives where` table, no change is needed: `.gdoc/<slug>/` still
holds the same files.

Add a new section after `## The tool must work without git`:

```markdown
## pending.md is the working set, not a review's output

`gdoc-review` writes it and `gdoc-apply` writes it too. Apply appends anything
marked in the document that no review captured, so a half-finished session is
resumable: items already applied are cleared, the rest are still there.

Apply never trusts the file alone. `gdoc worklist` merges it against a live read
by comment id, and a queued item is placed by what its thread says now. A reply
since capture may narrow the ask or withdraw it.

Both skills run one test on a comment: does it change the document's text. Yes is
an item, no is a question answered in the thread. The old local and global
distinction is retired. If you find those words in a skill or in the code, they
are stale.
```

Add to the `## Never` list:

```markdown
- Never diff `baseline.md` against the source markdown. It is compared to a fresh
  export and to nothing else. The two carry the same pandoc round trip so it
  cancels; the source does not, and under the template work it would surface the
  whole cover page as spurious edits.
```

- [ ] **Step 2: Update README.md**

Find the section describing the two skills and the review-then-apply flow. Replace
the description of what each skill does with:

```markdown
`/gdoc-review` reads the marked comments, answers the questions in their threads,
and captures anything that changes the document's text as an item.

`/gdoc-apply` collects everything the document is still owed: the captured items,
any marked comment left since the last review, any thread where someone replied
to an answer, and any edit typed into the document by hand. It works through them
with you, edits the source markdown, and generates the next version.

The order is the usual one, but it is no longer load bearing. Apply reads the
document itself, so nothing is lost by running it first or by leaving comments
between the two.
```

Add the three new commands to the CLI table, if the README has one:

| Command | What |
|---|---|
| `gdoc worklist <url>` | Merge `pending.md` with a live read. Five buckets |
| `gdoc drift <url>` | Diff `baseline.md` against the document now |
| `gdoc comment <doc> --body-file` | One top-level comment, for the superseded note |

- [ ] **Step 3: Check the docs against the code**

Run: `grep -rn -i "global item\|local change\|global change" README.md CLAUDE.md skills/`
Expected: no hits.

Run: `grep -n "pending global items" gdoc/pending.py`
Expected: one hit, in `_FILE_HEADER`. That heading is written into files that
already exist on disk, so changing it would make old and new queues differ for no
gain. Leave it. Note it here so the next reader knows it was seen and kept.

- [ ] **Step 4: Full suite with coverage**

Run: `~/.config/gdoc-agent/venv/bin/pytest -q --cov=gdoc --cov-report=term-missing`
Expected: PASS, and total coverage at or above 80%. `gdoc/worklist.py` and
`gdoc/drift.py` should both be at or near 100%, since both are pure.

- [ ] **Step 5: Check the installed skills still resolve**

Run: `./install.sh`
Expected: it re-runs cleanly and reports the symlinks. It prints
`+ uncommitted changes` if the tree is dirty, which it should not be at this
point.

Run: `ls -l ~/.claude/skills/gdoc-apply ~/.claude/skills/gdoc-review`
Expected: both are symlinks into `skills/` in this repo.

- [ ] **Step 6: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: apply drains the document, and one test decides an item

CLAUDE.md gains what pending.md now is, that apply writes it too, and the
rule that baseline.md is only ever compared to a fresh export.

README describes what each skill does now, and says the review-then-apply
order is no longer load bearing."
```

- [ ] **Step 7: Verify by hand, once**

No test can prove this. Run the sequence from section 1 of the spec against a
real document:

1. Generate v1 from a markdown file.
2. Leave two marked comments.
3. Run `/gdoc-review`. It answers one and captures one.
4. Leave two more marked comments, one of them from a second account if you can.
5. Reply under the agent's answer on the first thread.
6. Edit one sentence in the document by hand.
7. Run `/gdoc-apply`.

Expected: the worklist shows 1 captured, 1 reopened, 2 fresh. The drift step finds
the hand-edited sentence. v2 carries all of it. The old version has one closure
reply per applied thread and one "superseded by" comment.

Write down anything that surprised you. Do not fix it in this branch unless it is
a fault in what this plan built.

---

## Self-Review

**Spec coverage:**

| Spec section | Task |
|---|---|
| 3. What gdoc-apply does, in order | 6 |
| 4. The merge, five buckets, the gate, the two stops | 3, 6 |
| 4. Apply reads the thread, not the captured text | 3 (`merge` takes text from the live thread) |
| 5. Direct edits are folded back, translate not patch | 4, 6 |
| 5. Terminal-only still runs the diff | 6 |
| 6. has_agent_reply becomes last-reply | 1 |
| 7. One test, local and global retired | 6, 7 |
| 8. pending.md written by apply too | 6 (apply runs `capture`) |
| 9. Closure replies and the superseded note | 5, 6 |
| 10. What leaves the OAuth branch | Already done, commit `302f5b5` on `gdoc-oauth` |
| 12. Out of scope | Nothing in this plan copies comments forward or re-anchors |
| 13. Testing | 1, 2, 3, 4, 5, and Task 8 step 7 for the by-hand run |

**Gap found and closed:** the spec's section 13 asks for a credential-gated
integration test. There is no task for it. It is deliberate: `tests/test_access_integration.py`
tests what the credential can and cannot do, and a merge test against a live
document would assert on comments a person can change at any time. Task 8 step 7
covers the same ground by hand, once, which is what the spec's own "by hand,
because no test can prove it" line asks for. If a maintainer wants it automated
later, it needs a fixture document nobody comments on.

**Placeholder scan:** none. Every code step carries the code. Task 6 and Task 7
carry the replacement markdown verbatim rather than describing it.

**Test style checked against the file, not assumed.** `tests/test_cli.py` imports
`json`, `pytest`, `MagicMock`, `patch` and `main`, and every test drives the CLI
through `main([...])` with `patch("gdoc.cli.X")` rather than calling a handler
with a hand-built namespace. The three CLI test blocks match that. File metadata
is stubbed the way line 30 already does it,
`drive.files().get.return_value.execute.return_value = {"name": "Test doc"}`, and
only `cmd_worklist` needs it. `tests/test_reply.py` already imports `pytest` and
`MagicMock`, so the new `post_comment` tests need no new imports. `DOC_URL`,
`_fake_thread`, `_run_drift` and `_write_baseline` are new module-level helpers
in `tests/test_cli.py` and none of those names is taken.

**Failure modes checked.** A missing subcommand makes argparse exit 2, not raise
`AttributeError`, so the two "verify it fails" steps for `worklist` and `drift`
say that. `patch("gdoc.cli.post_comment")` does raise `AttributeError` before the
import exists, so Task 5's says that instead.

**Type consistency:** `PendingItem(number, comment_id, text)` from Task 2 is what
Task 3's `merge` consumes and what `_from_item` reads. `WorkItem.item_number` is
`int | None` in Task 3's dataclass, its tests, and `_work_json`. `diff_markdown`
returns `str` in Task 4 and `cmd_drift` tests it with `bool(diff)`. `post_comment`
returns `str` in Task 5 and `cmd_comment` emits it as `comment_id`.
`agent_replied_last` is spelled the same in Task 1's model, filter, and tests.
