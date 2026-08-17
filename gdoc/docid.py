"""Turn whatever Nail pastes into a Drive document id, or a Drive folder id."""

import re

_URL_PATTERNS = (
    re.compile(r"/document/d/([a-zA-Z0-9_-]+)"),
    re.compile(r"[?&]id=([a-zA-Z0-9_-]+)"),
)

_FOLDER_URL = re.compile(r"/folders/([a-zA-Z0-9_-]+)")
_DOCUMENT_URL = re.compile(r"/document/d/[a-zA-Z0-9_-]+")

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


def extract_folder_id(url_or_id: str) -> str:
    """Accept a Drive folder URL, or a bare folder id.

    Opening the folder and copying the address bar is what a person actually has
    at hand, so it is what the tool takes. A document URL is rejected here by
    name: it carries a valid-looking id, so passing it through would upload into
    nothing and come back as a 404 that names neither mistake.
    """
    text = (url_or_id or "").strip()
    match = _FOLDER_URL.search(text)
    if match:
        return match.group(1)
    if _DOCUMENT_URL.search(text):
        raise ValueError(f"that URL is a document, not a folder: {url_or_id!r}")
    if _BARE_ID.fullmatch(text):
        return text
    raise ValueError(f"not a Drive folder URL or folder id: {url_or_id!r}")
