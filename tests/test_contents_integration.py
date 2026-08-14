"""The whole publish path, against the live Drive API, with no LibreOffice.

This one **creates** documents, unlike the read-only integration checks, so it is
opt-in rather than merely credential-gated. A plain `pytest` must not write to
somebody's Drive. Run it deliberately:

    GDOC_LIVE_PUBLISH_TEST=1 pytest -m integration tests/test_contents_integration.py -s

It uploads two throwaway documents and trashes both before it finishes, including
when an assertion fails.
"""

import os
import shutil

import pytest

from gdoc.auth import drive_service
from gdoc.config import load_config
from gdoc.generate import upload_as_gdoc
from gdoc.render import build, contents, pagination
from gdoc.render.profiles import TEMPLATES_DIR

pytestmark = [
    pytest.mark.integration,
    pytest.mark.skipif(
        os.environ.get("GDOC_LIVE_PUBLISH_TEST") != "1",
        reason="creates Drive documents, set GDOC_LIVE_PUBLISH_TEST=1 to run",
    ),
]

EXAMPLE = TEMPLATES_DIR / "altery-group-policy-v1.0" / "example.md"
GHOSTS = ("Policy Statements 1 - Details", "Variation of procedure for legal entity X")


@pytest.fixture
def drive():
    try:
        return drive_service()
    except FileNotFoundError as error:
        pytest.skip(str(error))


@pytest.fixture
def folder():
    try:
        folder_id = load_config().output_folder_id
    except FileNotFoundError as error:
        pytest.skip(str(error))
    if not folder_id:
        pytest.skip("no output_folder_id configured")
    return folder_id


def _publish(drive, docx, name, folder):
    created = upload_as_gdoc(drive, docx, name, folder)
    pdf = drive.files().export(fileId=created["id"], mimeType="application/pdf").execute()
    return created["id"], pdf


@pytest.mark.skipif(not shutil.which("pdftotext"), reason="poppler not installed")
def test_the_published_contents_list_describes_its_own_document(drive, folder, tmp_path):
    source = tmp_path / "source.docx"
    build(EXAMPLE, source, skip_toc=True)          # no LibreOffice: skip_toc
    headings = contents.headings(
        __import__("zipfile").ZipFile(source).read("word/document.xml").decode("utf8")
    )
    assert headings, "the build produced no headings"

    trash = []
    try:
        # pass one: blank page numbers, purely to learn Google's pagination
        first = tmp_path / "pass1.docx"
        contents.write(source, first)
        first_id, first_pdf = _publish(drive, first, "gdoc test pass 1 (delete me)", folder)
        trash.append(first_id)
        (tmp_path / "pass1.pdf").write_bytes(first_pdf)
        pages = pagination.from_pdf(tmp_path / "pass1.pdf", headings)

        unresolved = [text for _l, text in headings if text not in pages]
        assert not unresolved, f"headings never found in the rendered document: {unresolved}"

        # pass two: the real page numbers, and the version that would be published
        second = tmp_path / "pass2.docx"
        contents.write(source, second, pages=pages)
        second_id, second_pdf = _publish(drive, second, "gdoc test pass 2 (delete me)", folder)
        trash.append(second_id)
        (tmp_path / "pass2.pdf").write_bytes(second_pdf)

        published = pagination.from_pdf(tmp_path / "pass2.pdf", headings)
        moved = pagination.drift(pages, published)
        assert not moved, f"the contents list disagrees with the document it sits in: {moved}"

        text = (tmp_path / "pass2.pdf").read_bytes()
        for ghost in GHOSTS:
            assert ghost.encode() not in text, f"template placeholder leaked: {ghost}"
    finally:
        for file_id in trash:
            try:
                drive.files().update(fileId=file_id, body={"trashed": True},
                                     supportsAllDrives=True).execute()
            except Exception as error:  # noqa: BLE001 - cleanup must not mask the failure
                print(f"could not trash {file_id}: {error}")
