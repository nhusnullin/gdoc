import shutil

import pytest

from gdoc.render import build
from gdoc.render.profiles import TEMPLATES_DIR

pytestmark = pytest.mark.slow

EXAMPLE = TEMPLATES_DIR / "altery-group-policy-v1.0" / "example.md"


@pytest.mark.skipif(not shutil.which("soffice"), reason="LibreOffice not installed")
def test_the_bundled_example_builds_into_a_real_docx(tmp_path):
    result = build(EXAMPLE, tmp_path / "out.docx")
    assert result.docx_path.read_bytes()[:2] == b"PK"
    assert result.title
    assert result.blocks > 0


def test_skip_toc_builds_without_libreoffice(tmp_path):
    result = build(EXAMPLE, tmp_path / "out.docx", skip_toc=True)
    assert result.docx_path.exists()


def test_a_missing_source_is_reported_before_anything_else(tmp_path):
    from gdoc.render import BuildError

    with pytest.raises(BuildError):
        build(tmp_path / "nope.md", tmp_path / "out.docx", skip_toc=True)
