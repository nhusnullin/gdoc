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
