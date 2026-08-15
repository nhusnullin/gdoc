"""Decide which threads the agent may act on.

The author is not checked at all. Launching the skill on a document is the trust
decision: Nail chose the document, and a marked comment inside it is an
instruction to act, whoever wrote it. Authorship could not be verified anyway,
because the Drive API returns no email address for comment authors and display
names are editable, so matching them was a guess dressed as a check.

Under `auth_mode: oauth` this stops being a preference. Drive's `author.me` is
Nail, so a check on it would skip every comment he writes and gdoc read would
return nothing at all.

The marker is `ai` plus a sign, at the start of the comment. It starts with a
letter because `@` opens the people picker in Google Docs and Nail does not want
to pick himself out of an address book every time. The sign is required on the
bare form, so a comment that merely begins with "AI is..." is left alone. The
older `@ai` form is still accepted, with or without a sign, because a comment
that is silently ignored is worse than a rare false positive.
"""

import re

from gdoc.model import Thread

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
    """True when the thread is waiting on gdoc.

    Who wrote the comment is not part of the test. Under OAuth the credential is
    Nail, so an author check would skip every comment he writes.

    has_agent_reply is what makes a second run idempotent: it never posts twice,
    and no local record of handled ids is needed.
    """
    return (
        is_addressed(thread.content)
        and not thread.resolved
        and not thread.has_agent_reply
    )


def partition(
    threads: tuple[Thread, ...],
) -> tuple[tuple[Thread, ...], tuple[Thread, ...]]:
    """Split into (addressed, skipped).

    needs_action holds the whole test. skipped is returned rather than discarded
    so the skill can tell Nail why a comment he can see was ignored.
    """
    addressed: list[Thread] = []
    skipped: list[Thread] = []
    for thread in threads:
        if needs_action(thread):
            addressed.append(thread)
        else:
            skipped.append(thread)
    return tuple(addressed), tuple(skipped)
