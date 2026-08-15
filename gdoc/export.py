"""Get a Google Doc as markdown.

Drive can export a Doc straight to markdown. That path is preferred because it
avoids a temp file and a subprocess. It is new enough that it may not be
available for every document, so a docx export through pandoc is kept as a
fallback: pandoc is already required for generating new versions, so the
fallback adds a code path but no new dependency.
"""

import shutil
import subprocess
import tempfile
from pathlib import Path

from googleapiclient.errors import HttpError

MARKDOWN_MIME = "text/markdown"
DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

# Where Homebrew puts it on an Apple silicon Mac. A fallback, not the answer: this
# path was hardcoded once and meant the tool only ran on one machine.
HOMEBREW_PANDOC = "/opt/homebrew/bin/pandoc"


class PandocNotFound(RuntimeError):
    """pandoc is needed here and could not be found."""


def find_pandoc() -> str:
    """Locate pandoc, at call time rather than at import time.

    Resolved on every call on purpose. Resolving once at import would freeze the
    answer for the life of the process, so installing pandoc while the tool is
    running, or shipping a wheelhouse that adds it later, would not take effect.
    """
    found = shutil.which("pandoc")
    if found:
        return found
    if Path(HOMEBREW_PANDOC).is_file():
        return HOMEBREW_PANDOC
    raise PandocNotFound(
        "pandoc is not installed, or not on PATH. It is needed to read markdown. "
        "Install it and make sure `pandoc` runs from your shell, or put it at "
        f"{HOMEBREW_PANDOC}."
    )

# Statuses that mean "this conversion is not offered", as opposed to a real fault.
_UNSUPPORTED = (400, 415)


def _pandoc_docx_to_markdown(docx_bytes: bytes) -> str:
    with tempfile.TemporaryDirectory() as tmp:
        docx_path = Path(tmp) / "doc.docx"
        docx_path.write_bytes(docx_bytes)
        result = subprocess.run(
            [find_pandoc(), str(docx_path), "-f", "docx", "-t", "markdown", "--wrap=none"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout


def export_markdown(drive, doc_id: str) -> str:
    try:
        data = drive.files().export(fileId=doc_id, mimeType=MARKDOWN_MIME).execute()
        return data.decode("utf-8") if isinstance(data, bytes) else data
    except HttpError as error:
        if error.resp.status not in _UNSUPPORTED:
            raise
    docx = drive.files().export(fileId=doc_id, mimeType=DOCX_MIME).execute()
    return _pandoc_docx_to_markdown(docx)
