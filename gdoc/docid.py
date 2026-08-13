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
