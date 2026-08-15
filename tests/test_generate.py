from unittest.mock import MagicMock, patch

import pytest
from googleapiclient.errors import HttpError

from gdoc.generate import generate, md_to_docx

TEMPLATE = "altery-group-policy-v1.0"
# A template document carries the master's cover, control tables and contents
# list, so it cannot be confused with a bare pandoc file of the same two lines.
# The master alone is 58KB; the pandoc file is a few KB.
TEMPLATE_SIZE = 40_000
TEMPLATED_MARKDOWN = "---\ntitle: Kickoff Notes\n---\n\n# Section one\n\nBody.\n"


class FakeResponse:
    def __init__(self, status):
        self.status = status
        self.reason = "test"


def _drive(ids=("1Throwaway", "1New")):
    """A Drive that accepts uploads and exports something PDF-shaped.

    The PDF bytes are never parsed: every test that uses them patches
    pagination.from_pdf, because building a real multi-page PDF here would test
    pypdf rather than this module.
    """
    drive = MagicMock()
    drive.files().create.return_value.execute.side_effect = [
        {"id": file_id, "webViewLink": f"https://docs.google.com/document/d/{file_id}/edit"}
        for file_id in ids
    ]
    drive.files().export.return_value.execute.return_value = b"%PDF-1.4 not really"
    return drive


def test_pandoc_produces_a_real_docx(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nSome body text.\n")
    out = md_to_docx(md, tmp_path / "out.docx")
    assert out.exists()
    assert out.read_bytes()[:2] == b"PK"  # docx is a zip


def test_generate_uploads_and_reports_the_new_document(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nBody.\n")
    drive = MagicMock()
    drive.files().create.return_value.execute.return_value = {
        "id": "1NewDoc",
        "webViewLink": "https://docs.google.com/document/d/1NewDoc/edit",
    }
    result = generate(drive, md, "Policy v4", tmp_path / "v4.docx", folder_id="0AFolder")
    assert result.doc_id == "1NewDoc"
    assert result.link.endswith("/edit")
    assert result.reason is None


def test_generate_falls_back_to_local_docx_when_there_is_no_folder(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nBody.\n")
    drive = MagicMock()
    result = generate(drive, md, "Policy v4", tmp_path / "v4.docx", folder_id=None)
    assert result.doc_id is None
    assert result.docx_path.exists()
    assert "output_folder_id" in result.reason
    drive.files().create.assert_not_called()


def test_generate_falls_back_and_keeps_the_docx_when_upload_is_refused(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nBody.\n")
    drive = MagicMock()
    drive.files().create.return_value.execute.side_effect = HttpError(FakeResponse(403), b"{}")
    result = generate(drive, md, "Policy v4", tmp_path / "v4.docx", folder_id="0AFolder")
    assert result.doc_id is None
    assert result.docx_path.exists()
    assert "403" in result.reason


def test_generate_never_reports_a_document_it_did_not_create(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nBody.\n")
    drive = MagicMock()
    drive.files().create.return_value.execute.side_effect = HttpError(FakeResponse(500), b"{}")
    result = generate(drive, md, "Policy v4", tmp_path / "v4.docx", folder_id="0AFolder")
    assert result.doc_id is None and result.link is None


def test_missing_markdown_file_is_reported_before_anything_else(tmp_path):
    drive = MagicMock()
    with pytest.raises(FileNotFoundError):
        generate(drive, tmp_path / "missing.md", "X", tmp_path / "x.docx", folder_id=None)


# ---------------------------------------------------------------------------
# the house template
# ---------------------------------------------------------------------------


@pytest.mark.slow
def test_generate_renders_through_the_template_when_one_is_given(tmp_path):
    """Two uploads: one to learn Google's pagination, one to publish."""
    md = tmp_path / "in.md"
    md.write_text(TEMPLATED_MARKDOWN)
    drive = _drive()
    with patch("gdoc.generate.pagination.from_pdf", return_value={"Section one": 3}):
        result = generate(
            drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
            folder_id="0AFolder", template=TEMPLATE,
        )
    assert result.doc_id == "1New"
    assert result.reason is None
    assert not result.drift
    assert result.docx_path.stat().st_size > TEMPLATE_SIZE
    assert drive.files().create.call_count == 2


@pytest.mark.slow
def test_a_title_override_publishes_a_note_that_has_none(tmp_path):
    """Without the override the build would refuse, because the note has no title."""
    md = tmp_path / "2026-08-12-miguel-kickoff-call.md"
    original = "---\ngdoc: 1abc\n---\n\n# Section one\n\nBody.\n"
    md.write_text(original)
    drive = _drive()
    with patch("gdoc.generate.pagination.from_pdf", return_value={"Section one": 3}):
        result = generate(
            drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
            folder_id="0AFolder", template=TEMPLATE, title="Kickoff Notes",
        )
    assert result.doc_id == "1New"
    assert md.read_text() == original


@pytest.mark.slow
def test_the_first_pass_document_is_trashed(tmp_path):
    """The blank-page-number upload exists only to be measured. It must not linger."""
    md = tmp_path / "in.md"
    md.write_text(TEMPLATED_MARKDOWN)
    drive = _drive()
    with patch("gdoc.generate.pagination.from_pdf", return_value={"Section one": 3}):
        generate(
            drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
            folder_id="0AFolder", template=TEMPLATE,
        )
    trashed = drive.files().update.call_args.kwargs
    assert trashed["fileId"] == "1Throwaway"
    assert trashed["body"] == {"trashed": True}


@pytest.mark.slow
def test_generate_reports_drift_instead_of_hiding_it(tmp_path):
    """A page number that moved means the contents list describes another document."""
    md = tmp_path / "in.md"
    md.write_text(TEMPLATED_MARKDOWN)
    drive = _drive()
    with patch(
        "gdoc.generate.pagination.from_pdf",
        side_effect=[{"Section one": 3}, {"Section one": 4}],
    ):
        result = generate(
            drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
            folder_id="0AFolder", template=TEMPLATE,
        )
    assert result.doc_id == "1New"
    assert result.drift == {"Section one": (3, 4)}
    assert "Section one" in result.reason


@pytest.mark.slow
def test_a_template_without_a_folder_builds_locally_and_says_the_pages_are_blank(tmp_path):
    """No Drive means no layout engine, so there are no page numbers to write."""
    md = tmp_path / "in.md"
    md.write_text(TEMPLATED_MARKDOWN)
    drive = _drive()
    result = generate(
        drive, md, "Kickoff Notes v1", tmp_path / "v1.docx", folder_id=None, template=TEMPLATE,
    )
    assert result.doc_id is None
    assert result.docx_path.stat().st_size > TEMPLATE_SIZE
    assert "output_folder_id" in result.reason
    assert "page numbers" in result.reason
    drive.files().create.assert_not_called()


@pytest.mark.slow
def test_a_refused_second_upload_keeps_the_docx_and_bins_the_measuring_copy(tmp_path):
    md = tmp_path / "in.md"
    md.write_text(TEMPLATED_MARKDOWN)
    drive = _drive()
    drive.files().create.return_value.execute.side_effect = [
        {"id": "1Throwaway", "webViewLink": "x"},
        HttpError(FakeResponse(403), b"{}"),
    ]
    with patch("gdoc.generate.pagination.from_pdf", return_value={"Section one": 3}):
        result = generate(
            drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
            folder_id="0AFolder", template=TEMPLATE,
        )
    assert result.doc_id is None
    assert "403" in result.reason
    assert result.docx_path.stat().st_size > TEMPLATE_SIZE
    assert drive.files().update.call_args.kwargs["fileId"] == "1Throwaway"


@pytest.mark.slow
def test_a_refused_first_upload_still_leaves_a_document_on_disk(tmp_path):
    """Nothing was measured, so the document is built again with blank page numbers."""
    md = tmp_path / "in.md"
    md.write_text(TEMPLATED_MARKDOWN)
    drive = _drive()
    drive.files().create.return_value.execute.side_effect = [HttpError(FakeResponse(403), b"{}")]
    result = generate(
        drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
        folder_id="0AFolder", template=TEMPLATE,
    )
    assert result.doc_id is None
    assert "403" in result.reason
    assert result.docx_path.stat().st_size > TEMPLATE_SIZE
    drive.files().update.assert_not_called()


@pytest.mark.slow
def test_a_measuring_copy_that_survives_is_named_in_the_reason(tmp_path):
    """A document nobody can find is worse than one the reason names."""
    md = tmp_path / "in.md"
    md.write_text(TEMPLATED_MARKDOWN)
    drive = _drive()
    drive.files().update.return_value.execute.side_effect = HttpError(FakeResponse(500), b"{}")
    with patch("gdoc.generate.pagination.from_pdf", return_value={"Section one": 3}):
        result = generate(
            drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
            folder_id="0AFolder", template=TEMPLATE,
        )
    assert result.doc_id == "1New"
    assert "1Throwaway" in result.reason
    assert "by hand" in result.reason


@pytest.mark.slow
def test_a_document_that_cannot_be_read_back_is_still_reported(tmp_path):
    """The drift check is the last step. By then the document exists, so it is reported.

    The failure is a plain network error, not an HttpError: reading the document
    back can fail in ways the Drive client never wraps.
    """
    md = tmp_path / "in.md"
    md.write_text(TEMPLATED_MARKDOWN)
    drive = _drive()
    with patch(
        "gdoc.generate.pagination.from_pdf",
        side_effect=[{"Section one": 3}, ConnectionResetError("connection reset")],
    ):
        result = generate(
            drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
            folder_id="0AFolder", template=TEMPLATE,
        )
    assert result.doc_id == "1New"
    assert result.link.endswith("/edit")
    assert "never checked" in result.reason
    assert result.drift is None
    assert drive.files().update.call_args.kwargs["fileId"] == "1Throwaway"


@pytest.mark.slow
def test_a_failed_second_build_bins_the_measuring_copy(tmp_path):
    """Nothing was published, so the error stands, but the stray copy must not."""
    from gdoc import render

    md = tmp_path / "in.md"
    md.write_text(TEMPLATED_MARKDOWN)
    drive = _drive()
    builds = []
    real_build = render.build

    def flaky_build(*args, **kwargs):
        builds.append(kwargs.get("pages"))
        if len(builds) == 2:
            raise render.BuildError("the second build failed")
        return real_build(*args, **kwargs)

    with patch("gdoc.generate.render.build", side_effect=flaky_build), patch(
        "gdoc.generate.pagination.from_pdf", return_value={"Section one": 3}
    ), pytest.raises(render.BuildError):
        generate(
            drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
            folder_id="0AFolder", template=TEMPLATE,
        )
    assert drive.files().update.call_args.kwargs["fileId"] == "1Throwaway"


@pytest.mark.slow
def test_an_unreadable_export_bins_the_measuring_copy_before_it_raises(tmp_path):
    """Nothing was published, so the failure must not leave a stray document behind."""
    from gdoc.render.pagination import PaginationError

    md = tmp_path / "in.md"
    md.write_text(TEMPLATED_MARKDOWN)
    drive = _drive()
    with patch(
        "gdoc.generate.pagination.from_pdf", side_effect=PaginationError("not a PDF")
    ), pytest.raises(PaginationError):
        generate(
            drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
            folder_id="0AFolder", template=TEMPLATE,
        )
    assert drive.files().update.call_args.kwargs["fileId"] == "1Throwaway"


def test_template_none_keeps_the_plain_pandoc_path(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nBody.\n")
    drive = _drive(ids=("1New",))
    result = generate(
        drive, md, "X v1", tmp_path / "v1.docx", folder_id="0AFolder", template=None,
    )
    assert result.doc_id == "1New"
    assert result.docx_path.stat().st_size < TEMPLATE_SIZE
    assert drive.files().create.call_count == 1
