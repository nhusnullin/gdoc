"""Turn approved markdown into a new Google Doc.

One converter, two destinations. pandoc makes a .docx; Drive converts that .docx
into a native Doc on create. If the upload cannot happen, the .docx is already on
disk and stays there, so the fallback is this pipeline stopping one step early
rather than a second code path.
"""

import subprocess
from dataclasses import dataclass
from pathlib import Path

from googleapiclient.errors import HttpError
from googleapiclient.http import MediaFileUpload

from gdoc.export import DOCX_MIME, PANDOC

GOOGLE_DOC_MIME = "application/vnd.google-apps.document"


@dataclass(frozen=True)
class Result:
    docx_path: Path
    doc_id: str | None = None
    link: str | None = None
    reason: str | None = None


def md_to_docx(md_path: Path, out_path: Path) -> Path:
    out_path.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        [PANDOC, str(md_path), "-o", str(out_path)],
        check=True,
        capture_output=True,
        text=True,
    )
    return out_path


def upload_as_gdoc(drive, docx_path: Path, name: str, folder_id: str) -> dict:
    media = MediaFileUpload(str(docx_path), mimetype=DOCX_MIME, resumable=False)
    return (
        drive.files()
        .create(
            body={"name": name, "mimeType": GOOGLE_DOC_MIME, "parents": [folder_id]},
            media_body=media,
            fields="id,webViewLink",
            supportsAllDrives=True,
        )
        .execute()
    )


def generate(drive, md_path: Path, name: str, out_path: Path, folder_id: str | None) -> Result:
    """Convert, then try to upload. Never claim a document that was not created."""
    if not md_path.exists():
        raise FileNotFoundError(f"markdown file not found: {md_path}")
    docx_path = md_to_docx(md_path, out_path)
    if not folder_id:
        return Result(
            docx_path=docx_path,
            reason=(
                "no output_folder_id in config, so nothing was uploaded. "
                f"The document is ready at {docx_path}"
            ),
        )
    try:
        created = upload_as_gdoc(drive, docx_path, name, folder_id)
    except HttpError as error:
        return Result(
            docx_path=docx_path,
            reason=(
                f"upload refused with {error.resp.status}. Check that the service "
                f"account is a Content manager on folder {folder_id}. "
                f"The document is ready at {docx_path}"
            ),
        )
    return Result(
        docx_path=docx_path,
        doc_id=created["id"],
        link=created.get("webViewLink"),
    )
