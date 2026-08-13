from unittest.mock import MagicMock

import pytest
from googleapiclient.errors import HttpError

from tools.gdoc.generate import generate, md_to_docx


class FakeResponse:
    def __init__(self, status):
        self.status = status
        self.reason = "test"


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
