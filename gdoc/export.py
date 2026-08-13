"""Get a Google Doc as markdown.

Drive can export a Doc straight to markdown. That path is preferred because it
avoids a temp file and a subprocess. It is new enough that it may not be
available for every document, so a docx export through pandoc is kept as a
fallback: pandoc is already required for generating new versions, so the
fallback adds a code path but no new dependency.
"""

import subprocess
import tempfile
from pathlib import Path

from googleapiclient.errors import HttpError

MARKDOWN_MIME = "text/markdown"
DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
PANDOC = "/opt/homebrew/bin/pandoc"

# Statuses that mean "this conversion is not offered", as opposed to a real fault.
_UNSUPPORTED = (400, 415)


def _pandoc_docx_to_markdown(docx_bytes: bytes) -> str:
    with tempfile.TemporaryDirectory() as tmp:
        docx_path = Path(tmp) / "doc.docx"
        docx_path.write_bytes(docx_bytes)
        result = subprocess.run(
            [PANDOC, str(docx_path), "-f", "docx", "-t", "markdown", "--wrap=none"],
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
