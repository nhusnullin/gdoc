"""Global items, captured for a later terminal session.

One markdown file per document. The comment id is written into each item so a
re-run can tell that an item is already captured, which matters because the
in-thread reply is the only other record and Nail may resolve it.
"""

import re
from pathlib import Path

from gdoc.mirror import MIRROR_DIR
from gdoc.model import Thread

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

{request}
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

    number = next_item_number(path)
    anchor = thread.quoted.strip() if thread.quoted else "whole document"
    raw_lines = thread.content.strip().splitlines()
    quoted_request = "\n".join(f"> {line}" if line else ">" for line in raw_lines)
    item = _ITEM.format(
        number=number,
        today=today,
        comment_id=thread.id,
        anchor=anchor,
        request=quoted_request,
    )

    # Open in append mode so an interrupted write cannot destroy existing items.
    # _FILE_HEADER ends with \n; _ITEM starts with \n, giving a blank-line
    # separator between sections without any truncate-and-rewrite.
    with open(path, "a") as f:
        if not existing:
            f.write(_FILE_HEADER.format(doc_id=doc_id))
        f.write(item)

    return number
