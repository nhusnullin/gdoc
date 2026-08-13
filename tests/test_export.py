from unittest.mock import MagicMock, patch

import pytest
from googleapiclient.errors import HttpError

from tools.gdoc.export import DOCX_MIME, MARKDOWN_MIME, export_markdown


class FakeResponse:
    def __init__(self, status):
        self.status = status
        self.reason = "test"


def http_error(status):
    return HttpError(FakeResponse(status), b"{}")


def test_uses_native_markdown_export_when_available():
    drive = MagicMock()
    drive.files().export.return_value.execute.return_value = b"# Title\n\nBody."
    assert export_markdown(drive, "doc") == "# Title\n\nBody."
    drive.files().export.assert_called_with(fileId="doc", mimeType=MARKDOWN_MIME)


def test_falls_back_to_docx_and_pandoc_when_markdown_is_unsupported():
    drive = MagicMock()
    drive.files().export.return_value.execute.side_effect = [
        http_error(400),
        b"PK\x03\x04 fake docx bytes",
    ]
    with patch("tools.gdoc.export._pandoc_docx_to_markdown", return_value="# From docx") as pandoc:
        assert export_markdown(drive, "doc") == "# From docx"
    pandoc.assert_called_once()


def test_unexpected_api_errors_are_not_swallowed():
    drive = MagicMock()
    drive.files().export.return_value.execute.side_effect = http_error(500)
    with pytest.raises(HttpError):
        export_markdown(drive, "doc")


def test_docx_mime_is_the_openxml_one():
    assert DOCX_MIME.endswith("wordprocessingml.document")
