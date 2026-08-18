"""Post replies, and refuse anything Docs would render literally."""

import re

from gdoc.marker import with_marker

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
