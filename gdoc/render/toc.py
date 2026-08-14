"""Give the contents list real page numbers.

A .docx stores its contents list as a field, and page numbers do not exist
until something lays the document out. Headless LibreOffice is that something.
The same pass repairs the malformed TOC and PAGE fields Google Docs left in the
master, where the field start, instruction and separator all sit in one run.
"""

import shutil
import subprocess
from pathlib import Path

UPDATE_TOC = Path(__file__).resolve().parent / "scripts" / "update-toc.sh"


class TocError(RuntimeError):
    """The contents-list refresh could not run, or failed."""


def refresh_toc(docx_path: Path, pdf_path: Path | None = None) -> None:
    if not shutil.which("soffice"):
        raise TocError(
            "LibreOffice (soffice) is not installed, so the contents list "
            "cannot be given page numbers. Install it with "
            "`brew install --cask libreoffice`, or pass --skip-toc."
        )
    # Two passes on purpose. update-toc.sh exports a PDF *instead of* re-saving
    # the .docx when given an output path, so asking only for the PDF would
    # leave the .docx carrying the master's stale page numbers.
    commands = [[str(UPDATE_TOC), str(docx_path)]]
    if pdf_path:
        commands.append([str(UPDATE_TOC), str(docx_path), str(pdf_path)])
    for command in commands:
        result = subprocess.run(command, capture_output=True, text=True)
        if result.returncode != 0:
            raise TocError(f"the contents-list refresh failed:\n{result.stderr}")
