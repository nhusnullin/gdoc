"""Global items, captured for a later terminal session.

One markdown file per document. The comment id is written into each item so a
re-run can tell that an item is already captured, which matters because the
in-thread reply is the only other record and Nail may resolve it.

Each item records the author and the marker the comment carried. The marker is
what decides that a comment is work, and the author is a label beside it, so
neither can be assumed from the other.
"""

import re
from pathlib import Path

from gdoc.baseline import GDOC_DIR
from gdoc.filters import forced_kind, is_addressed
from gdoc.model import Thread

_ITEM_HEADING = re.compile(r"^## Item (\d+)", re.MULTILINE)
_SOURCE_LINE = re.compile(r"^Source: (.+)$", re.MULTILINE)

_FILE_HEADER = """\
# Pending global items

Document: https://docs.google.com/document/d/{doc_id}/edit
{source_line}
Each item needs changes across the document, so it was not answered in the
comment thread. Work through them with /gdoc-apply.
"""

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


def pending_path(repo_root: Path, slug: str) -> Path:
    return repo_root / GDOC_DIR / slug / "pending.md"


def recorded_source(path: Path) -> str | None:
    """The source file this queue belongs to, or None.

    None also covers queues written before the Source line existed, so an older
    file stays readable.
    """
    if not path.exists():
        return None
    match = _SOURCE_LINE.search(path.read_text())
    return match.group(1).strip() if match else None


def next_item_number(path: Path) -> int:
    if not path.exists():
        return 1
    numbers = [int(match) for match in _ITEM_HEADING.findall(path.read_text())]
    return max(numbers) + 1 if numbers else 1


def append_item(
    repo_root: Path,
    slug: str,
    thread: Thread,
    doc_id: str,
    today: str,
    source: str | None = None,
) -> int:
    path = pending_path(repo_root, slug)
    path.parent.mkdir(parents=True, exist_ok=True)

    existing = path.read_text() if path.exists() else ""

    if thread.id in existing:
        raise ValueError(f"comment {thread.id} is already captured in {path}")

    number = next_item_number(path)
    anchor = thread.quoted.strip() if thread.quoted else "whole document"
    raw_lines = thread.content.strip().splitlines()
    quoted_request = "\n".join(f"> {line}" if line else ">" for line in raw_lines)
    item = _ITEM.format(
        number=number,
        today=today,
        comment_id=thread.id,
        author=thread.author_name,
        anchor=anchor,
        marked=_marked_label(thread.content),
        request=quoted_request,
    )

    # Open in append mode so an interrupted write cannot destroy existing items.
    # _FILE_HEADER ends with \n; _ITEM starts with \n, giving a blank-line
    # separator between sections without any truncate-and-rewrite.
    with open(path, "a") as f:
        if not existing:
            source_line = f"Source: {source}\n" if source else ""
            f.write(_FILE_HEADER.format(doc_id=doc_id, source_line=source_line))
        f.write(item)

    return number
