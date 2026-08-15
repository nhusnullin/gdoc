"""Turn approved markdown into a new Google Doc.

Two converters, one destination. With a template, gdoc.render puts the markdown
into the house master. Without one, pandoc makes a plain .docx. Either way Drive
converts the .docx into a native Doc on create. If the upload cannot happen, the
.docx is already on disk and stays there, so the fallback is this pipeline
stopping one step early rather than a second code path.

The template path uploads twice, because page numbers do not exist until
something lays the document out and Google is that something. The first upload
carries blank page numbers and exists only to be measured: it is exported as
PDF, read for the page each heading landed on, and then trashed. The second
upload is the document that gets published, and it is measured too, so a
contents list that disagrees with its own document is reported rather than
hidden.
"""

import subprocess
import tempfile
import zipfile
from dataclasses import dataclass
from pathlib import Path

from googleapiclient.errors import HttpError
from googleapiclient.http import MediaFileUpload

from gdoc import render
from gdoc.export import DOCX_MIME, find_pandoc
from gdoc.render import contents, pagination

GOOGLE_DOC_MIME = "application/vnd.google-apps.document"
PDF_MIME = "application/pdf"
# The throwaway says what it is in its own title, so a run that dies before the
# cleanup leaves something a human can recognise and delete.
MEASURING_SUFFIX = " (pagination pass, delete me)"


@dataclass(frozen=True)
class Result:
    docx_path: Path
    doc_id: str | None = None
    link: str | None = None
    reason: str | None = None
    # Headings whose published page differs from the number written next to them,
    # as {heading: (written, published)}. None means nothing moved.
    drift: dict | None = None


def md_to_docx(md_path: Path, out_path: Path) -> Path:
    out_path.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        [find_pandoc(), str(md_path), "-o", str(out_path)],
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


def generate(
    drive,
    md_path: Path,
    name: str,
    out_path: Path,
    folder_id: str | None,
    *,
    template: str | None = None,
) -> Result:
    """Convert, then try to upload. Never claim a document that was not created.

    template names the house style to render through. None keeps the plain
    pandoc path, which is what a caller asking for no template gets.
    """
    if not md_path.exists():
        raise FileNotFoundError(f"markdown file not found: {md_path}")
    if template is not None:
        return _through_template(drive, md_path, name, out_path, folder_id, template)
    docx_path = md_to_docx(md_path, out_path)
    if not folder_id:
        return Result(docx_path=docx_path, reason=_no_folder(docx_path))
    try:
        created = upload_as_gdoc(drive, docx_path, name, folder_id)
    except HttpError as error:
        return Result(docx_path=docx_path, reason=_refused(error, folder_id, docx_path))
    return Result(
        docx_path=docx_path,
        doc_id=created["id"],
        link=created.get("webViewLink"),
    )


def _through_template(drive, md_path, name, out_path, folder_id, template) -> Result:
    """Render through the house template, with the page numbers Google measured."""
    out_path.parent.mkdir(parents=True, exist_ok=True)
    if not folder_id:
        render.build(md_path, out_path, template=template)
        return Result(docx_path=out_path, reason=_no_folder(out_path, blank_pages=True))

    with tempfile.TemporaryDirectory() as workspace:
        scratch = Path(workspace)
        blank = render.build(md_path, scratch / "blank.docx", template=template).docx_path
        headings = _headings_of(blank)
        measured = None
        try:
            # The upload is its own statement so the id is held before anything
            # else can fail. Bound inside the call, a failure would lose it and
            # leave the measuring copy in the folder for good.
            measured = upload_as_gdoc(drive, blank, name + MEASURING_SUFFIX, folder_id)
            pages = pagination.from_pdf(
                _export_pdf(drive, measured["id"], scratch / "blank.pdf"), headings
            )
            render.build(md_path, out_path, template=template, pages=pages)
            created = upload_as_gdoc(drive, out_path, name, folder_id)
        except HttpError as error:
            return _upload_refused(drive, error, folder_id, md_path, out_path, template, measured)
        except pagination.PaginationError:
            # Nothing was published, so bin the measuring copy before the error escapes.
            _trash_measured(drive, measured)
            raise
        # The document exists from here on, so nothing below may withhold its id.
        moved, unchecked = _check_drift(
            drive, created["id"], scratch / "published.pdf", headings, pages
        )
        notes = [
            note
            for note in (
                _trash_measured(drive, measured),
                unchecked,
                _drift_note(moved) if moved else None,
            )
            if note
        ]

    return Result(
        docx_path=out_path,
        doc_id=created["id"],
        link=created.get("webViewLink"),
        reason=". ".join(notes) or None,
        drift=moved or None,
    )


def _export_pdf(drive, doc_id: str, pdf_path: Path) -> Path:
    pdf_path.write_bytes(drive.files().export(fileId=doc_id, mimeType=PDF_MIME).execute())
    return pdf_path


def _check_drift(drive, doc_id: str, pdf_path: Path, headings, pages: dict) -> tuple:
    """Compare the published document against the numbers written into it.

    The document already exists, so a failure here is a note about a real
    document rather than a reason to hide it. Returns (drift, note).
    """
    try:
        published = pagination.from_pdf(_export_pdf(drive, doc_id, pdf_path), headings)
    except (HttpError, pagination.PaginationError) as error:
        return {}, (
            "the published document could not be read back, so its page numbers "
            f"were never checked against its contents list: {error}"
        )
    return pagination.drift(pages, published), None


def _headings_of(docx_path: Path) -> tuple:
    with zipfile.ZipFile(docx_path) as archive:
        return contents.headings(archive.read("word/document.xml").decode("utf8"))


def _upload_refused(drive, error, folder_id, md_path, out_path, template, measured) -> Result:
    """Keep the promise that the .docx is on disk, and clean up what was uploaded.

    The build that was published may never have happened, so it happens here,
    with blank page numbers because there is no measurement to write in.
    """
    if not out_path.exists():
        render.build(md_path, out_path, template=template)
    notes = [_refused(error, folder_id, out_path), _trash_measured(drive, measured)]
    return Result(docx_path=out_path, reason=". ".join(note for note in notes if note))


def _trash_measured(drive, measured: dict | None) -> str | None:
    """Bin the measuring copy, if there is one. Returns a note when it survived."""
    if not measured:
        return None
    file_id = measured["id"]
    try:
        drive.files().update(
            fileId=file_id, body={"trashed": True}, supportsAllDrives=True
        ).execute()
    except HttpError as error:
        return (
            f"the measuring copy {file_id} could not be trashed "
            f"({error.resp.status}), so delete it by hand"
        )
    return None


def _drift_note(moved: dict) -> str:
    listed = ", ".join(
        f"{text!r} says page {written} but sits on page {published}"
        for text, (written, published) in sorted(moved.items())
    )
    return (
        "the contents list disagrees with the document it sits in: "
        f"{listed}. The document was created, so check its contents page before sharing it"
    )


def _no_folder(docx_path: Path, blank_pages: bool = False) -> str:
    reason = (
        "no output_folder_id in config, so nothing was uploaded. "
        f"The document is ready at {docx_path}"
    )
    if blank_pages:
        reason += (
            ". Its contents list has blank page numbers, because the page numbers "
            "come from uploading to Google"
        )
    return reason


def _refused(error: HttpError, folder_id: str, docx_path: Path) -> str:
    return (
        f"Drive refused the publish with {error.resp.status}. Check that the service "
        f"account is a Content manager on folder {folder_id}. "
        f"The document is ready at {docx_path}"
    )
