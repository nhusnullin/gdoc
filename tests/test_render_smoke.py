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


def test_a_heading_matching_the_title_is_dropped_from_the_body():
    from gdoc.render import drop_title_heading

    body = "# Kickoff Notes\n\nFirst paragraph.\n"
    assert drop_title_heading(body, "Kickoff Notes").strip() == "First paragraph."


def test_a_heading_that_is_not_the_title_stays():
    from gdoc.render import drop_title_heading

    body = "# Background\n\nFirst paragraph.\n"
    assert drop_title_heading(body, "Kickoff Notes") == body


def test_only_the_first_heading_is_considered():
    from gdoc.render import drop_title_heading

    body = "Intro line.\n\n# Kickoff Notes\n\nMore.\n"
    assert drop_title_heading(body, "Kickoff Notes") == body


def test_a_heading_matching_the_raw_title_is_dropped_even_with_a_different_cover_title():
    """The form that already worked must keep working once cover_title is added."""
    from gdoc.render import drop_title_heading

    body = "# Kickoff Notes\n\nFirst paragraph.\n"
    result = drop_title_heading(body, "Kickoff Notes", "Kickoff Notes Policy")
    assert result.strip() == "First paragraph."


def test_a_heading_matching_the_cover_title_is_dropped():
    """A user typing what the cover shows, title plus doc_type, is a duplicate too."""
    from gdoc.render import drop_title_heading

    body = "# Kickoff Notes Policy\n\nFirst paragraph.\n"
    result = drop_title_heading(body, "Kickoff Notes", "Kickoff Notes Policy")
    assert result.strip() == "First paragraph."


def test_a_heading_matching_neither_form_stays():
    from gdoc.render import drop_title_heading

    body = "# Background\n\nFirst paragraph.\n"
    assert drop_title_heading(body, "Kickoff Notes", "Kickoff Notes Policy") == body
