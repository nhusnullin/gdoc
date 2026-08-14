"""The contents list is written by us, not by a field and not by LibreOffice.

Every check here runs offline. No credential, no network, no LibreOffice. That is
the point of the module under test.
"""

import re
import zipfile

import pytest

from gdoc.render import build, contents
from gdoc.render.profiles import TEMPLATES_DIR

EXAMPLE = TEMPLATES_DIR / "altery-group-policy-v1.0" / "example.md"

# Sections that exist only in the master template's stale cached field result.
# If any of these reach the output, the reader gets a contents page describing a
# different document.
GHOSTS = (
    "Policy Statements 1 - Details",
    "Variation of procedure for legal entity X",
    "Documentation and Record Keeping",
)


@pytest.fixture(scope="module")
def built(tmp_path_factory):
    out = tmp_path_factory.mktemp("built") / "source.docx"
    build(EXAMPLE, out, skip_toc=True)
    return out


def document_xml(path):
    return zipfile.ZipFile(path).read("word/document.xml").decode("utf8")


def styles_xml(path):
    return zipfile.ZipFile(path).read("word/styles.xml").decode("utf8")


def entry_paragraphs(path):
    xml = document_xml(path)
    return [p for p in re.findall(r"<w:p\b.*?</w:p>", xml, re.S)
            if re.search(r'w:pStyle w:val="TOC\d"', p)]


def test_headings_are_found_in_document_order(built):
    found = contents.headings(document_xml(built))
    assert [text for _level, text in found][:4] == [
        "1-Purpose", "1.1-Policy Owner", "1.1.1-Delegation", "2-Scope"
    ]
    assert [level for level, _text in found][:4] == [1, 2, 3, 1]


def test_the_source_build_still_carries_the_stale_field(built):
    """Guards the premise. If this fails, the master changed and so must the fix."""
    xml = document_xml(built)
    assert "<w:instrText" in xml
    assert any(ghost in xml for ghost in GHOSTS)


def test_writing_replaces_the_field_with_one_entry_per_heading(built, tmp_path):
    out = tmp_path / "written.docx"
    result = contents.write(built, out)
    assert len(result.entries) == len(entry_paragraphs(out))
    assert len(result.entries) == len(contents.headings(document_xml(built)))


def test_no_stale_template_section_survives(built, tmp_path):
    out = tmp_path / "written.docx"
    contents.write(built, out)
    xml = document_xml(out)
    for ghost in GHOSTS:
        assert ghost not in xml, f"the template's own contents list leaked: {ghost}"


def test_the_field_itself_is_gone(built, tmp_path):
    out = tmp_path / "written.docx"
    contents.write(built, out)
    xml = document_xml(out)
    assert "<w:instrText" not in xml
    assert 'w:fldCharType="begin"' not in xml


def test_toc_styles_are_added_when_the_template_lacks_them(built, tmp_path):
    assert 'w:styleId="TOC1"' not in styles_xml(built)
    out = tmp_path / "written.docx"
    contents.write(built, out)
    after = styles_xml(out)
    for sid in ("Index", "TOC1", "TOC2", "TOC3"):
        assert f'w:styleId="{sid}"' in after


def test_entries_declare_no_font_and_no_bold(built, tmp_path):
    """Google applies Arial and bold level one when we leave this to its defaults.

    The entries must carry the TOC styles and nothing direct, so they inherit the
    document default, which is Calibri.
    """
    out = tmp_path / "written.docx"
    contents.write(built, out)
    for para in entry_paragraphs(out):
        assert "w:ascii=" not in para
        assert not re.search(r"<w:b(\s+w:val=\"(1|true|on)\")?\s*/>", para)


def test_page_numbers_are_written_when_supplied(built, tmp_path):
    out = tmp_path / "written.docx"
    contents.write(built, out, pages={"1-Purpose": 4, "2-Scope": 7})
    paras = entry_paragraphs(out)
    purpose = next(p for p in paras if "1-Purpose" in p)
    scope = next(p for p in paras if "2-Scope" in p)
    assert "<w:t>4</w:t>" in purpose
    assert "<w:t>7</w:t>" in scope


def test_a_heading_with_no_page_number_gets_an_empty_cell(built, tmp_path):
    """Better a blank than a wrong number. gdoc build has no pagination to offer."""
    out = tmp_path / "written.docx"
    contents.write(built, out, pages={})
    paras = entry_paragraphs(out)
    assert paras
    assert all(re.search(r"<w:tab/><w:t[^>]*></w:t>", p) for p in paras)


def test_levels_map_to_their_styles(built, tmp_path):
    out = tmp_path / "written.docx"
    result = contents.write(built, out)
    by_text = {text: level for level, text in result.entries}
    paras = entry_paragraphs(out)
    for para in paras:
        text = "".join(re.findall(r"<w:t[^>]*>([^<]*)</w:t>", para))
        for known, level in by_text.items():
            if text.startswith(known):
                assert f'w:pStyle w:val="TOC{min(level, 3)}"' in para
                break


def test_a_document_with_no_headings_is_refused(built, tmp_path):
    stripped = tmp_path / "no_headings.docx"
    zin = zipfile.ZipFile(built)
    xml = document_xml(built).replace('w:pStyle w:val="Heading', 'w:pStyle w:val="Plain')
    with zipfile.ZipFile(stripped, "w") as zout:
        for item in zin.infolist():
            data = xml.encode("utf8") if item.filename == "word/document.xml" else zin.read(item.filename)
            zout.writestr(item, data)
    with pytest.raises(contents.ContentsError):
        contents.write(stripped, tmp_path / "out.docx")


def test_xml_special_characters_in_a_heading_survive(built, tmp_path):
    out = tmp_path / "written.docx"
    contents.write(built, out, pages={})
    # the bundled example has an ampersand-free set, so assert the escaper directly
    assert contents.escape('Risk & "Control" <x>') == "Risk &amp; &quot;Control&quot; &lt;x&gt;"
