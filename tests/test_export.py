from unittest.mock import MagicMock, patch

import pytest
from googleapiclient.errors import HttpError

from gdoc.export import DOCX_MIME, MARKDOWN_MIME, export_markdown


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
    with patch("gdoc.export._pandoc_docx_to_markdown", return_value="# From docx") as pandoc:
        assert export_markdown(drive, "doc") == "# From docx"
    pandoc.assert_called_once()


def test_unexpected_api_errors_are_not_swallowed():
    drive = MagicMock()
    drive.files().export.return_value.execute.side_effect = http_error(500)
    with pytest.raises(HttpError):
        export_markdown(drive, "doc")


def test_docx_mime_is_the_openxml_one():
    assert DOCX_MIME.endswith("wordprocessingml.document")


# ------------------------------------------------------------------- pictures --
# Issue #28. Drive's markdown export drops embedded pictures and Google Drawings
# entirely: verified against a real document on 2026-08-18, four drawings, no
# image markup of any kind in the markdown. The docx export carries all four as
# PNG bytes, so that is the only path that can bring them across.

from pathlib import Path  # noqa: E402

from gdoc.export import export_with_media, relocate_media  # noqa: E402


def test_media_is_moved_beside_the_markdown_and_linked_relatively(tmp_path):
    extracted = tmp_path / "tmp" / "media"
    extracted.mkdir(parents=True)
    (extracted / "image1.png").write_bytes(b"PNG")
    markdown = f"Text.\n\n![]({extracted / 'image1.png'})\n"

    out = tmp_path / "notes.md"
    result = relocate_media(markdown, tmp_path / "tmp", tmp_path / "notes-media", out)

    assert "![](notes-media/image1.png)" in result.markdown
    assert (tmp_path / "notes-media" / "image1.png").read_bytes() == b"PNG"
    assert result.images == (tmp_path / "notes-media" / "image1.png",)


def test_a_nested_extraction_directory_is_flattened(tmp_path):
    """pandoc writes <dir>/media/imageN.png. The nesting helps nobody."""
    extracted = tmp_path / "tmp" / "media" / "deeper"
    extracted.mkdir(parents=True)
    (extracted / "image2.png").write_bytes(b"PNG")
    markdown = f"![]({extracted / 'image2.png'})"

    result = relocate_media(markdown, tmp_path / "tmp", tmp_path / "media", tmp_path / "notes.md")
    assert (tmp_path / "media" / "image2.png").exists()
    assert "![](media/image2.png)" in result.markdown


def test_pandoc_attributes_survive_the_rewrite(tmp_path):
    extracted = tmp_path / "tmp" / "media"
    extracted.mkdir(parents=True)
    (extracted / "image1.png").write_bytes(b"PNG")
    markdown = f'![]({extracted / "image1.png"}){{width="7.9in" height="3.5in"}}'
    result = relocate_media(markdown, tmp_path / "tmp", tmp_path / "m", tmp_path / "notes.md")
    assert 'width="7.9in"' in result.markdown


def test_a_document_with_no_pictures_writes_no_directory(tmp_path):
    result = relocate_media("Just text.\n", tmp_path / "tmp", tmp_path / "media", tmp_path / "n.md")
    assert result.images == ()
    assert not (tmp_path / "media").exists()


def test_an_extracted_file_nothing_links_to_is_reported(tmp_path):
    """Google puts header images in the docx that the body never references."""
    extracted = tmp_path / "tmp" / "media"
    extracted.mkdir(parents=True)
    (extracted / "used.png").write_bytes(b"PNG")
    (extracted / "unused.png").write_bytes(b"PNG")
    markdown = f"![]({extracted / 'used.png'})"
    result = relocate_media(markdown, tmp_path / "tmp", tmp_path / "m", tmp_path / "n.md")
    assert any("unused.png" in warning for warning in result.warnings)


def test_export_with_media_always_takes_the_docx_route(tmp_path):
    """The markdown export has no pictures in it, so asking for it is pointless."""
    drive = MagicMock()
    drive.files().export.return_value.execute.return_value = b"PK fake docx"

    def fake_pandoc(docx_path, extract_to):
        Path(extract_to).mkdir(parents=True, exist_ok=True)
        (Path(extract_to) / "image1.png").write_bytes(b"PNG")
        return f"![]({Path(extract_to) / 'image1.png'})"

    with patch("gdoc.export._pandoc_docx_with_media", side_effect=fake_pandoc):
        result = export_with_media(drive, "doc", tmp_path / "media", tmp_path / "notes.md")

    drive.files().export.assert_called_with(fileId="doc", mimeType=DOCX_MIME)
    assert "![](media/image1.png)" in result.markdown
    assert len(result.images) == 1


def test_a_picture_pandoc_never_extracts_is_still_reported(tmp_path):
    """The real document holds four images in the docx; pandoc extracted three.

    The fourth is not in the markdown and not in the media directory, so without
    this it disappears with nobody told. That silence is the bug.
    """
    import io
    import zipfile

    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w") as archive:
        archive.writestr("word/document.xml", "<w:document/>")
        for name in ("image1.png", "image3.png"):
            archive.writestr(f"word/media/{name}", b"PNG")

    drive = MagicMock()
    drive.files().export.return_value.execute.return_value = buffer.getvalue()

    def fake_pandoc(docx_path, extract_to):
        Path(extract_to).mkdir(parents=True, exist_ok=True)
        (Path(extract_to) / "image1.png").write_bytes(b"PNG")
        return f"![]({Path(extract_to) / 'image1.png'})"

    with patch("gdoc.export._pandoc_docx_with_media", side_effect=fake_pandoc):
        result = export_with_media(drive, "doc", tmp_path / "media", tmp_path / "n.md")

    assert len(result.images) == 1
    assert any("image3.png" in warning for warning in result.warnings)
