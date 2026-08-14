import pytest

from gdoc.render import BuildError, build
from gdoc.render.profiles import TEMPLATES_DIR

pytestmark = pytest.mark.slow

EXAMPLE = TEMPLATES_DIR / "altery-group-policy-v1.0" / "example.md"


def test_the_bundled_example_builds_into_a_real_docx(tmp_path):
    """No skipif. A build needs no external program now, which is the point."""
    result = build(EXAMPLE, tmp_path / "out.docx")
    assert result.docx_path.read_bytes()[:2] == b"PK"
    assert result.title
    assert result.blocks > 0
    assert result.entries > 0


def test_the_build_writes_its_own_contents_list(tmp_path):
    result = build(EXAMPLE, tmp_path / "out.docx")
    assert result.entries == 12


def test_page_numbers_are_written_when_the_caller_knows_them(tmp_path):
    import zipfile

    out = tmp_path / "out.docx"
    build(EXAMPLE, out, pages={"1-Purpose": 4})
    xml = zipfile.ZipFile(out).read("word/document.xml").decode("utf8")
    assert "<w:t>4</w:t>" in xml


def test_a_missing_source_is_reported_before_anything_else(tmp_path):
    with pytest.raises(BuildError):
        build(tmp_path / "nope.md", tmp_path / "out.docx")
