"""The contents list is written by us, not by a field and not by LibreOffice.

Every check here runs offline. No credential, no network, no LibreOffice. That is
the point of the module under test.
"""

import re
import zipfile

import pytest

from gdoc.render import build, contents
from gdoc.render.profiles import DEFAULT_TEMPLATE, TEMPLATES_DIR, resolve

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
    build(EXAMPLE, out)
    return out


@pytest.fixture(scope="module")
def stale(tmp_path_factory):
    """A copy of the bundled master template, with its stale field intact.

    build() now writes the contents list itself, so a document it produces
    already carries real entries, not the master's stale ones. The tests that
    check contents.write's own behaviour on a stale field need the master
    template directly instead: it carries real Heading1-3 paragraphs, the
    stale cached field with the GHOSTS text, and no TOC1 style. Copied into
    tmp_path so a test never writes into the installed package.
    """
    out = tmp_path_factory.mktemp("stale") / "template.docx"
    master = resolve(DEFAULT_TEMPLATE)
    out.write_bytes(master.read_bytes())
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


def test_the_source_build_still_carries_the_stale_field(stale):
    """Guards the premise. If this fails, the master changed and so must the fix."""
    xml = document_xml(stale)
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


def test_the_field_is_kept_so_a_reader_can_still_refresh_it(built, tmp_path):
    """Replacing the cached result, not the field.

    Delete the field and Google imports plain text: no table of contents object,
    no update button, and a heading added later never appears. Keeping the field
    means the published document still holds a real, refreshable contents list.
    """
    out = tmp_path / "written.docx"
    contents.write(built, out)
    xml = document_xml(out)
    assert "<w:instrText" in xml
    assert 'w:fldCharType="begin"' in xml
    assert 'w:fldCharType="separate"' in xml
    assert 'w:fldCharType="end"' in xml


def test_the_field_brackets_the_entries(built, tmp_path):
    out = tmp_path / "written.docx"
    contents.write(built, out)
    xml = document_xml(out)
    begin = xml.index('w:fldCharType="begin"')
    end = xml.index('w:fldCharType="end"')
    first_entry = xml.index('w:pStyle w:val="TOC1"')
    assert begin < end
    assert begin < xml.rindex('w:pStyle w:val="TOC') < end
    assert first_entry < end


def test_every_entry_links_to_its_heading(built, tmp_path):
    out = tmp_path / "written.docx"
    result = contents.write(built, out)
    xml = document_xml(out)
    anchors = re.findall(r'<w:hyperlink w:anchor="([^"]+)"', xml)
    bookmarks = re.findall(r'<w:bookmarkStart[^>]*w:name="([^"]+)"', xml)
    ours = [a for a in anchors if a.startswith(contents.BOOKMARK_PREFIX)]
    assert len(ours) == len(result.entries)
    for anchor in ours:
        assert anchor in bookmarks, f"entry links to {anchor}, which no heading defines"


def test_each_heading_gets_exactly_one_bookmark(stale, tmp_path):
    out = tmp_path / "written.docx"
    result = contents.write(stale, out)
    xml = document_xml(out)
    starts = re.findall(rf'<w:bookmarkStart[^>]*w:name="({contents.BOOKMARK_PREFIX}\d+)"', xml)
    assert len(starts) == len(result.entries)
    assert len(set(starts)) == len(starts), "duplicate bookmark names"
    for name in starts:
        bookmark_id = re.search(
            rf'<w:bookmarkStart w:id="(\d+)" w:name="{name}"/>', xml).group(1)
        assert f'<w:bookmarkEnd w:id="{bookmark_id}"/>' in xml


def test_no_stale_cached_entry_survives_inside_the_field(built, tmp_path):
    """The field stays, but its old cached result must not."""
    out = tmp_path / "written.docx"
    contents.write(built, out)
    xml = document_xml(out)
    begin = xml.index('w:fldCharType="begin"')
    end = xml.index('w:fldCharType="end"')
    inside = xml[begin:end]
    for ghost in GHOSTS:
        assert ghost not in inside


def test_toc_styles_are_added_when_the_template_lacks_them(stale, tmp_path):
    assert 'w:styleId="TOC1"' not in styles_xml(stale)
    out = tmp_path / "written.docx"
    contents.write(stale, out)
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


def test_entry_styles_declare_their_own_spacing(built, tmp_path):
    """Inheriting the template's spacing makes the page jump on the first refresh.

    Google imposes its own paragraph spacing when a reader refreshes the contents
    list. The template default computes to about 21.6pt per line; Google's refresh
    gives 16.4pt. Leave the styles to inherit and the contents list visibly tightens
    the first time anyone clicks update. Declaring the tighter value keeps it still.
    """
    out = tmp_path / "written.docx"
    contents.write(built, out)
    after = styles_xml(out)
    for sid in ("TOC1", "TOC2", "TOC3"):
        style = re.search(rf'<w:style [^>]*w:styleId="{sid}".*?</w:style>', after, re.S).group(0)
        assert "<w:spacing" in style, f"{sid} would inherit the template's looser spacing"


def test_entry_indents_match_what_a_refresh_produces(built, tmp_path):
    """Levels sit 18pt and 36pt in, because that is what Google's refresh uses.

    The house template used 283 and 567 twips. Close, but a refresh moves the
    entries sideways, which is exactly the visible jump this avoids.
    """
    out = tmp_path / "written.docx"
    contents.write(built, out)
    after = styles_xml(out)
    for sid, expected in (("TOC1", 0), ("TOC2", 360), ("TOC3", 720)):
        style = re.search(rf'<w:style [^>]*w:styleId="{sid}".*?</w:style>', after, re.S).group(0)
        assert f'w:left="{expected}"' in style, f"{sid} should indent {expected} twips"


def test_only_the_first_entry_carries_space_above(built, tmp_path):
    """Google leaves 3pt above the first entry and nothing above the rest.

    Word and Google add space-before to space-after rather than collapsing them, so
    putting this on the style would widen every gap instead of just the first.
    """
    out = tmp_path / "written.docx"
    contents.write(built, out)
    paras = entry_paragraphs(out)
    assert 'w:before="60"' in paras[0]
    for para in paras[1:]:
        assert 'w:before="60"' not in para


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
    """Strict <w:t> matching, or the first entry is skipped.

    The first entry paragraph also carries the field instruction. A loose pattern
    matches <w:instrText> too, and then no entry text matches, so the check passes
    while verifying nothing.
    """
    out = tmp_path / "written.docx"
    result = contents.write(built, out)
    paras = entry_paragraphs(out)
    assert len(paras) == len(result.entries)
    checked = 0
    for (level, text), para in zip(result.entries, paras):
        found = "".join(re.findall(r"<w:t(?:\s[^>]*)?>([^<]*)</w:t>", para))
        assert found.startswith(text), f"expected {text!r} at the start of {found!r}"
        assert f'w:pStyle w:val="TOC{min(level, 3)}"' in para
        checked += 1
    assert checked == len(result.entries)


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


def test_rewriting_in_place_leaves_one_file(built, tmp_path):
    """A zip cannot be read and rewritten at the same path at once."""
    target = tmp_path / "inplace.docx"
    target.write_bytes(built.read_bytes())
    result = contents.rewrite_in_place(target, pages={"1-Purpose": 4})
    assert list(tmp_path.iterdir()) == [target], "a temporary file was left behind"
    assert len(result.entries) == 12
    assert "<w:t>4</w:t>" in document_xml(target)


def test_rewriting_in_place_cleans_up_after_a_failure(tmp_path):
    """The temporary file must not survive a failed rewrite."""
    junk = tmp_path / "junk.docx"
    junk.write_bytes(b"not a docx")
    with pytest.raises(zipfile.BadZipFile):
        contents.rewrite_in_place(junk)
    assert list(tmp_path.iterdir()) == [junk], "a temporary file was left behind"


def test_writing_twice_leaves_one_bookmark_per_heading_with_unique_names(built, tmp_path):
    """A rewrite must replace its own bookmarks, not add to them.

    The two-pass publish flow this feature exists for necessarily rewrites a
    document that already has a contents list: build writes blanks, the real
    page numbers come later. Bookmark names are unique document-wide, so
    writing twice must not leave two bookmarks carrying the same name.
    """
    once = tmp_path / "once.docx"
    twice = tmp_path / "twice.docx"
    result = contents.write(built, once)
    contents.write(once, twice)
    xml = document_xml(twice)
    starts = re.findall(rf'<w:bookmarkStart[^>]*w:name="({contents.BOOKMARK_PREFIX}\d+)"', xml)
    assert len(starts) == len(result.entries)
    assert len(set(starts)) == len(starts), "duplicate bookmark names"


def test_writing_twice_leaves_one_contents_entry_per_heading(built, tmp_path):
    """The second write replaces the first write's entries, not adds to them."""
    once = tmp_path / "once.docx"
    twice = tmp_path / "twice.docx"
    contents.write(built, once)
    result = contents.write(once, twice)
    assert len(result.entries) == len(entry_paragraphs(twice))


def test_a_bookmark_not_ours_survives_a_write_untouched(built, tmp_path):
    """Only our own bookmarks are ours to replace on a rewrite.

    The template's own bookmarks, or anyone else's, must be left alone. The
    attribute order here is Google's own, which is what the bundled master
    writes. Our order puts w:id first, so a regex anchored on that finds none of
    the template's bookmarks and every id we pick collides with one of them.
    """
    source = tmp_path / "with_foreign_bookmark.docx"
    xml = document_xml(built)
    foreign_start = (
        '<w:bookmarkStart w:colFirst="0" w:colLast="0" '
        'w:name="_heading=h.gjdgxs" w:id="8"/>'
    )
    foreign_end = '<w:bookmarkEnd w:id="8"/>'
    marker = xml.index("<w:body>") + len("<w:body>")
    xml = xml[:marker] + foreign_start + foreign_end + xml[marker:]
    with zipfile.ZipFile(built) as zin, zipfile.ZipFile(source, "w") as zout:
        for item in zin.infolist():
            data = xml.encode("utf8") if item.filename == "word/document.xml" else zin.read(item.filename)
            zout.writestr(item, data)

    out = tmp_path / "written.docx"
    contents.write(source, out)
    result_xml = document_xml(out)
    assert foreign_start in result_xml
    assert foreign_end in result_xml


def bookmark_ids(xml, tag):
    return re.findall(rf'<w:bookmark{tag}\b[^>]*?\bw:id="(\d+)"[^>]*/>', xml)


def test_writing_twice_keeps_every_bookmark_paired_and_every_id_unique(built, tmp_path):
    """Two invalid shapes that both still parse, so only counting finds them.

    Our ids must not collide with the template's Google-written ones. If they do,
    stripping our bookmarks on the second write also takes the foreign bookmark's
    end tag, and the template is left with a bookmarkStart that nothing closes.
    """
    once = tmp_path / "once.docx"
    twice = tmp_path / "twice.docx"
    contents.write(built, once)
    contents.write(once, twice)
    xml = document_xml(twice)
    starts = bookmark_ids(xml, "Start")
    ends = bookmark_ids(xml, "End")
    assert len(starts) == len(ends), "a bookmark was left unterminated"
    assert len(set(starts)) == len(starts), "duplicate bookmark ids"
    assert sorted(starts) == sorted(ends), "a start and an end disagree on their id"
