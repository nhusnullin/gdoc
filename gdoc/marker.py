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
